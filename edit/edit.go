// ****************************************************************************
//
//	 _ _          _
//	| (_) ___  __| |
//	| | |/ _ \/ _` |
//	| | |  __/ (_| |
//	|_|_|\___|\__,_|
//
// ****************************************************************************
// L I E D   -   Copyright © JPL 2024
// ****************************************************************************
package edit

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"lied/conf"
	"lied/dialog"
	"lied/ui"
	"lied/utils"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"encoding/hex"

	"github.com/gdamore/tcell/v2"
	"github.com/pgavlin/femto"
	"github.com/pgavlin/femto/runtime"
	"github.com/rivo/tview"
	"github.com/saintfish/chardet"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type Modes int64

const (
	Text Modes = iota
	Binary
	Shell
	SQLite3
	Explorer
	// PluginView is used for views backed by a ui.ViewPlugin (e.g. Service Manager).
	// The associated ViewScreen.Plugin field holds the concrete plugin instance.
	PluginView
)

type ViewScreen struct {
	FemtoBuffer         *femto.Buffer
	FemtoView           *femto.View
	FName               string
	Encoding            string
	GitCommit           string
	GitStatus           string
	GitBranch           string
	GitFileStatus       string
	ReadWrite           bool
	Follow              bool
	RWBackup            bool
	IsMemberOfWorkspace bool
	Mode                Modes
	ContentBytes        []byte
	HexContentDirty     bool
	Database            *sql.DB
	// Plugin holds the ViewPlugin implementation when Mode == PluginView.
	// It is nil for all file-based modes (Text, Binary, SQLite3, Explorer).
	Plugin ui.ViewPlugin
}

type found struct {
	s string
	l int
	c int
}

const (
	FLOW_SELF = iota
	FLOW_CLOSE
	FLOW_QUIT
	FLOW_NONE
	FIND_UP
	FIND_DOWN
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	OpenViews             []ViewScreen
	CurrentView           ViewScreen
	CurrentWidget         tview.Primitive
	DlgSaveFile           *dialog.Dialog
	DlgSaveFileAs         *dialog.Dialog
	currentFlow           int
	showHidden            bool
	Founds                []found
	iFounds               int
	currentFoundIndex     int // Cet index doit être géré avec soin
	findSession           bool
	previousWhat          string
	whatToFind            string // Renommé pour éviter la confusion avec previousWhat
	whereToFind           *femto.Buffer
	previousWhere         *femto.Buffer
	caseToFind            bool
	previousCase          bool
	AFind                 []string
	IFind                 int
	quitFlowIndex         int
	onlyOnce              bool
	HexFounds             []found
	hexCurrentFoundIndex  int
	hexFindSession        bool
	previousHexWhat       string
	previousHexSearchType string
	// AJOUTÉ : La dernière commande de recherche lancée
	lastSearchString string
	lastSearchCase   bool
	trackedNoname    map[string]struct{}
)

func init() {
	trackedNoname = make(map[string]struct{})
}

// ****************************************************************************
// SwitchToEditor()
// ****************************************************************************
func SwitchToEditor(fName string) {
	ui.SetTitle(conf.APP_NAME)
	ui.LblKeys.SetText(conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS)
	scr := ui.GetScreenFromTitle(conf.APP_NAME)
	if scr == "NIL" {
		var screen ui.MyScreen
		screen.ID, _ = utils.RandomHex(3)
		screen.Mode = ui.ModeTextEdit
		screen.Title = conf.APP_NAME
		screen.Keys = conf.CKEY_LABELS
		ui.PgsApp.AddPage(screen.Title+"_"+screen.ID, ui.FlxEditor, true, true)
		scr = screen.Title + "_" + screen.ID
		ui.ArrScreens = append(ui.ArrScreens, screen)
		ui.IdxScreens++
	}
	ui.PgsApp.SwitchToPage(scr)
	// ShowTreeDir(filepath.Dir(fName))
	// ShowTreeDir("/")
	OpenView(fName, true)
	if CurrentView.Plugin != nil {
		ui.App.SetFocus(CurrentView.Plugin.FocusWidget())
	}
}

// ****************************************************************************
// SetTheme()
// ****************************************************************************
func SetTheme(theme string) {
	if monokai := runtime.Files.FindFile(femto.RTColorscheme, theme); monokai != nil {
		if data, err := monokai.Data(); err == nil {
			var colorscheme femto.Colorscheme
			colorscheme = femto.ParseColorscheme(string(data))
			ui.EdtMain.SetColorscheme(colorscheme)
		}
	}
}

func directoryFromPath(path string) string {
	if path == "" {
		return ""
	}
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return path
		}
		d := filepath.Dir(path)
		if d != "." && d != "" {
			return d
		}
		return ""
	}
	d := filepath.Dir(path)
	if d != "." && d != "" {
		return d
	}
	return ""
}

func isPreferredWorkView(v ViewScreen) bool {
	if v.FName == "" {
		return false
	}
	if v.Mode == PluginView {
		return false
	}
	if isInternalShellOutputPath(v.FName) {
		return false
	}
	if v.Follow && isInInternalAppFolder(v.FName) {
		return false
	}
	return true
}

func isInInternalAppFolder(path string) bool {
	if path == "" {
		return false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	appFolder := filepath.Join(home, conf.APP_FOLDER)
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		pathAbs = filepath.Clean(path)
	}
	appFolderAbs, err := filepath.Abs(appFolder)
	if err != nil {
		appFolderAbs = filepath.Clean(appFolder)
	}

	pathAbs = filepath.Clean(pathAbs)
	appFolderAbs = filepath.Clean(appFolderAbs)

	if pathAbs == appFolderAbs {
		return true
	}

	return strings.HasPrefix(pathAbs, appFolderAbs+string(os.PathSeparator))
}

func isInternalShellOutputPath(path string) bool {
	if path == "" {
		return false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	internalOutput := filepath.Join(home, conf.APP_FOLDER, conf.FILE_SHELL_OUTPUT)
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		pathAbs = filepath.Clean(path)
	}
	internalAbs, err := filepath.Abs(internalOutput)
	if err != nil {
		internalAbs = filepath.Clean(internalOutput)
	}

	return filepath.Clean(pathAbs) == filepath.Clean(internalAbs)
}

// PreferredWorkingDirectory returns the best directory to use when opening
// explorer/shell flows: current file directory first, then previously-opened
// file/view directory, then workspace.
func PreferredWorkingDirectory() string {
	if isPreferredWorkView(CurrentView) {
		if d := directoryFromPath(CurrentView.FName); d != "" {
			return d
		}
	}

	currentIdx := -1
	for i, v := range OpenViews {
		if v.FName == CurrentView.FName {
			currentIdx = i
			break
		}
	}

	if currentIdx >= 0 && len(OpenViews) > 1 {
		for step := 1; step < len(OpenViews); step++ {
			idx := currentIdx - step
			if idx < 0 {
				idx += len(OpenViews)
			}
			candidate := OpenViews[idx]
			if !isPreferredWorkView(candidate) {
				continue
			}
			if d := directoryFromPath(candidate.FName); d != "" {
				return d
			}
		}
	}

	if d := directoryFromPath(CurrentView.FName); d != "" {
		return d
	}
	if conf.ConfigGeneral.Workspace != "" {
		return conf.ConfigGeneral.Workspace
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// ****************************************************************************
// OpenView()
// ****************************************************************************
func OpenView(fName string, rw bool) {
	onlyOnce = false
	if isViewAlreadyOpen(fName) {
		SwitchOpenView(fName)
	} else {
		ui.EdtMain.SetRuntimeFiles(runtime.Files)
		// Check if the file exists
		if _, err := os.Stat(fName); errors.Is(err, os.ErrNotExist) {
			ui.SetStatus(fmt.Sprintf("File %v doesn't exist", fName))
			CreateThisFile(fName)
		} else {
			// Check if it's a path, then explorer view
			if utils.IsDir(fName) {
				CurrentView.FName = fName
				CurrentView.Mode = Explorer
				CurrentView.ReadWrite = false
				CurrentView.Follow = false
				CurrentView.Plugin = explorerPlugin
				OpenViews = append(OpenViews, CurrentView)
				//
				go UpdateStatus()
				go focusOpenFile(fName)
				//
				ui.SetStatus(fmt.Sprintf("Exploring %s", CurrentView.FName))
				if CurrentView.Plugin != nil {
					CurrentView.Plugin.Activate()
				}
				ShowFiles()
				ui.App.SetFocus(ui.TblFiles)
				return
			}
			// Check if the file is a SQLite3 database
			if utils.IsSQLite3(fName) {
				// content, err := ioutil.ReadFile(fName)
				if err != nil {
					ui.SetStatus(fmt.Sprintf("Could not read SQLite3 file %v: %v", fName, err))
					return
				}
				CurrentView.FName = fName
				// CurrentFile.IsBinary = false
				CurrentView.Mode = SQLite3
				CurrentView.ReadWrite = false
				CurrentView.Follow = false
				CurrentView.Encoding = "SQLite3"
				// var errDB error
				CurrentView.Database, _ = sql.Open("sqlite3", CurrentView.FName)
				CurrentView.Plugin = sqlPlugin
				OpenViews = append(OpenViews, CurrentView)
				// defer CloseDB(CurrentFile.Database)
				go UpdateStatus()
				go focusOpenFile(fName)
				ui.SetStatus(fmt.Sprintf("Opening SQLite3 file %s", CurrentView.FName))
				ui.DisplayExifInfo(CurrentView.FName) // Display EXIF info for this data
				showTreeDB()
				ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
				ui.SetFindPanelVisible(false)
				ui.PgsEditorContent.SwitchToPage("sqlViewer")
				ui.App.SetFocus(ui.TxtPromptSQL)
				return
			} else {
				ui.SetStatus(fmt.Sprintf("%s is not a SQLite3 database", CurrentView.FName))
			}

			// Check if the file is binary
			if utils.IsBinaryFile(fName) {
				content, err := ioutil.ReadFile(fName)
				if err != nil {
					ui.SetStatus(fmt.Sprintf("Could not read binary file %v: %v", fName, err))
					return
				}
				CurrentView.FName = fName
				CurrentView.Mode = Binary
				// CurrentFile.IsDatabase = false
				CurrentView.ReadWrite = false // Binary files are read-only for now
				CurrentView.Follow = false
				CurrentView.Encoding = "Binary"
				CurrentView.ContentBytes = content
				CurrentView.HexContentDirty = true // Mark for refresh
				CurrentView.Plugin = hexPlugin
				OpenViews = append(OpenViews, CurrentView)
				go UpdateStatus()
				go focusOpenFile(fName)
				ui.SetStatus(fmt.Sprintf("Opening binary file %s", CurrentView.FName))
				displayBinaryContent()
				ui.DisplayExifInfo(CurrentView.FName) // Display EXIF info for binary files
				ui.ConfigureFindFormForBinary(true)
				ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
				ui.SetFindPanelVisible(true)
				ui.App.SetFocus(ui.HexView) // Set focus to HexView for binary files
				ui.PgsEditorContent.SwitchToPage("hexViewer")
				return
			} else {
				ui.SetStatus(fmt.Sprintf("%s is not a binary file", CurrentView.FName))
			}

			content, err := ioutil.ReadFile(fName)
			if err != nil {
				ui.SetStatus(fmt.Sprintf("Could not read %v", fName))
				ui.SetStatus(fmt.Sprintf("%v", err))
			} else {
				detector := chardet.NewTextDetector()
				result, err := detector.DetectBest(content)
				if err == nil {
					CurrentView.Encoding = result.Charset
				} else {
					CurrentView.Encoding = "Unknown"
				}

				CurrentView.FName = fName
				CurrentView.FemtoBuffer = femto.NewBufferFromString(string(content), CurrentView.FName)
				// 				CurrentFile.Buffer.Settings["wordwrap"] = false
				CurrentView.FemtoBuffer.Settings["keepautoindent"] = true
				CurrentView.FemtoBuffer.Settings["softwrap"] = true
				CurrentView.FemtoBuffer.Settings["scrollbar"] = true
				CurrentView.FemtoBuffer.Settings["statusline"] = false

				CurrentView.FemtoView = femto.NewView(CurrentView.FemtoBuffer)
				CurrentView.ReadWrite = rw
				CurrentView.Follow = false
				CurrentView.Mode = Text
				// CurrentFile.IsDatabase = false
				CurrentView.ContentBytes = nil // Ensure this is nil for text files
				ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
				SetTheme(conf.ConfigGeneral.Theme)
				ui.EdtMain.SetTitleAlign(tview.AlignRight)
				ui.LblScreen.SetText(CurrentView.Encoding)
				CurrentView = UpdateGITInfos(CurrentView)
				CurrentView.Plugin = NewTextModePlugin(CurrentView.FemtoBuffer)
				OpenViews = append(OpenViews, CurrentView)
				go UpdateStatus()
				go focusOpenFile(fName)
				ui.SetStatus(fmt.Sprintf("Opening file %s", CurrentView.FName))
				ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
				ui.SetFindPanelVisible(true)
				ui.App.SetFocus(ui.EdtMain)
				ui.PgsEditorContent.SwitchToPage("textEditor")
			}
		}
	}
	ShowTreeDir(conf.ConfigGeneral.Workspace, showHidden)
}

// ****************************************************************************
// SaveFile()
// ****************************************************************************
func SaveFile() {
	err := ioutil.WriteFile(CurrentView.FName, []byte(CurrentView.FemtoBuffer.String()), 0600)
	if err == nil {
		ui.SetStatus(fmt.Sprintf("File %s successfully saved", CurrentView.FName))
		CurrentView.FemtoBuffer.IsModified = false
	} else {
		ui.SetStatus(err.Error())
	}
}

// ****************************************************************************
// SaveAnyFile()
// ****************************************************************************
func SaveAnyFile(f any) {
	SaveFile()
}

// ****************************************************************************
// SaveAll()
// ****************************************************************************
func SaveAll() {
	fIndex := 0
	of := len(OpenViews)
	for ; fIndex < of; fIndex++ {
		err := ioutil.WriteFile(OpenViews[fIndex].FName, []byte(OpenViews[fIndex].FemtoBuffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", OpenViews[fIndex].FName))
			OpenViews[fIndex].FemtoBuffer.IsModified = false
		} else {
			ui.SetStatus(err.Error())
		}
	}
}

// ****************************************************************************
// SaveFileAs()
// ****************************************************************************
func SaveFileAs() {
	ui.SetStatus("Save as...")
	currentFlow = FLOW_SELF
	DlgSaveFileAs = DlgSaveFileAs.Input("Save File as...", // Title
		"Please, enter the new name for this file :", // Message
		CurrentView.FName,
		confirmSaveAs,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgSaveFileAs", DlgSaveFileAs.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSaveFileAs")

	/*
		err := ioutil.WriteFile(currentFile.fName, []byte(currentFile.buffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", currentFile.fName))
			currentFile.buffer.IsModified = false
		} else {
			ui.SetStatus(err.Error())
		}
	*/
}

// ****************************************************************************
// SaveAnyFileAs()
// ****************************************************************************
func SaveAnyFileAs(f any) {
	SaveFileAs()
}

// ****************************************************************************
// NewFile()
// ****************************************************************************
func NewFile(dir string) {
	f, err := os.CreateTemp(dir, conf.NEW_FILE_TEMPLATE)
	if err == nil {
		trackedNoname[f.Name()] = struct{}{}
		// SwitchToEditor(f.Name())
		OpenView(f.Name(), true)
	} else {
		ui.SetStatus(err.Error())
	}
}

func cleanupTrackedNonameFiles() int {
	deleted := 0
	for fName := range trackedNoname {
		info, err := os.Stat(fName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				delete(trackedNoname, fName)
				continue
			}
			ui.SetStatus(fmt.Sprintf("Error when checking file %s : %s", fName, err.Error()))
			continue
		}
		if info.IsDir() {
			continue
		}
		if info.Size() == 0 {
			ui.SetStatus(fmt.Sprintf("Deleting empty draft file %s", fName))
			if err := os.Remove(fName); err != nil {
				ui.SetStatus(fmt.Sprintf("Error when deleting empty draft file %s : %s", fName, err.Error()))
				continue
			}
			delete(trackedNoname, fName)
			deleted++
		}
	}
	return deleted
}

// ****************************************************************************
// CreateThisFile()
// ****************************************************************************
func CreateThisFile(dir string) bool {
	f, err := os.Create(dir)
	ui.SetStatus(fmt.Sprintf("Creating the file %v", dir))
	if err == nil {
		OpenView(f.Name(), true)
		return true
	} else {
		ui.SetStatus(err.Error())
		return false
	}
}

// ****************************************************************************
// NewAnyFile()
// ****************************************************************************
func NewAnyFile(f any) {
	NewFile(f.(string))
}

// ****************************************************************************
// NewFileOrLastFile()
// ****************************************************************************
func NewFileOrLastFile(dir string) {
	if len(OpenViews) > 0 {
		SwitchToEditor(CurrentView.FName)
	} else {
		NewFile(dir)
	}
}

// ****************************************************************************
// OpenPluginView()
// OpenPluginView opens a plugin-backed view.  If the plugin is already in the
// open-views list the existing entry is switched to; otherwise a new ViewScreen
// with Mode == PluginView is created and appended.
// ****************************************************************************
func OpenPluginView(plugin ui.ViewPlugin) {
	syntheticID := plugin.ID() + "://" + plugin.ID()

	// Already open? Switch to it.
	if isViewAlreadyOpen(syntheticID) {
		SwitchOpenView(syntheticID)
		return
	}

	var vs ViewScreen
	vs.FName = syntheticID
	vs.Mode = PluginView
	vs.Plugin = plugin
	vs.ReadWrite = false
	vs.Follow = false

	if err := plugin.Open(nil); err != nil {
		ui.SetStatus(fmt.Sprintf("Error opening %s: %v", plugin.Title(), err))
		return
	}

	OpenViews = append(OpenViews, vs)
	CurrentView = vs
	CurrentWidget = plugin.FocusWidget()

	plugin.Activate()
	ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
	ui.SetStatus(fmt.Sprintf("Opened %s", plugin.Title()))

	go focusOpenFile(syntheticID)
}

// ****************************************************************************
// UpdateStatus()
// ****************************************************************************
func UpdateStatus() {
	// var status string
	var count int = 0
	for {
		time.Sleep(100 * time.Millisecond)
		ui.App.QueueUpdateDraw(func() {
			// ui.TxtEditName.SetText(currentFile.FName)
			bp := filepath.Base(conf.ConfigGeneral.Workspace)
			dp := filepath.Dir(conf.ConfigGeneral.Workspace)
			dp += string(os.PathSeparator)
			ui.TxtCurrentWorkspace.SetText(dp + "[yellow]" + bp)
			// ui.TxtCurrentEditName.SetText(filepath.Dir(CurrentFile.FName) + string(os.PathSeparator) + "[yellow]" + filepath.Base(CurrentFile.FName))
			relativePath, err := filepath.Rel(conf.ConfigGeneral.Workspace, CurrentView.FName)
			if err != nil {
				// Handle error: perhaps log it or display a default value
				relativePath = filepath.Base(CurrentView.FName)
			}
			dirPath := filepath.Dir(relativePath)
			if dirPath == "." {
				dirPath = ""
			} else {
				dirPath += string(os.PathSeparator)
			}
			ui.TxtCurrentEditName.SetText(dirPath + "[yellow]" + filepath.Base(relativePath))

			if CurrentView.Mode == Text && CurrentView.FemtoBuffer != nil {
				// Refresh title and handle Follow (tail) mode.
				ui.EdtMain.SetTitle(fmt.Sprintf("[ %s %s %s ]", CurrentView.Encoding, CurrentView.FemtoBuffer.Settings["filetype"].(string), CurrentView.FemtoBuffer.Settings["fileformat"].(string)))
				if CurrentView.Follow {
					_, _, _, lines := ui.EdtMain.GetInnerRect()
					c := exec.Command("tail", "-n", strconv.Itoa(lines-1), CurrentView.FName)
					output, _ := c.Output()
					CurrentView.FemtoBuffer = femto.NewBufferFromString(string(output), CurrentView.FName)
					CurrentView.FemtoBuffer.Cursor.Y = CurrentView.FemtoBuffer.End().Y
					// Keep the per-entry plugin's buffer pointer in sync.
					if p, ok := CurrentView.Plugin.(*TextModePlugin); ok {
						p.buf = CurrentView.FemtoBuffer
					}
					ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
				}
			}

			ui.TblOpenFiles.Clear()
			count++
			for i, f := range OpenViews {
				if count%20 == 0 {
					// Update GIT infos only once in 20 to prevent huge CPU use
					f = UpdateGITInfos(f)
					OpenViews[i] = f
				}
				// All modes now have a plugin; use it for icon and dirty flag.
				if f.Plugin != nil {
					icon := f.Plugin.Icon()
					if f.Plugin.IsDirty() {
						icon = conf.ICON_MODIFIED
					}
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(icon+f.GitFileStatus))
				} else {
					// Fallback for views created without a plugin (should not happen).
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(" "+f.GitFileStatus))
				}
				ui.TblOpenFiles.SetCell(i, 1, tview.NewTableCell(filepath.Base(f.FName)))
				ui.TblOpenFiles.SetCell(i, 2, tview.NewTableCell("⯈"))
				ui.TblOpenFiles.SetCell(i, 3, tview.NewTableCell(f.FName))
			}
			syncCurrentOpenViewSelection()

			// Uniform status-bar update: every plugin provides the label values.
			if CurrentView.Plugin != nil {
				fields := CurrentView.Plugin.StatusFields()
				ui.LblReadWrite.SetText(fields.ReadWrite)
				ui.LblCursor.SetText(fields.Cursor)
				ui.LblDirty.SetText(fields.Dirty)
				ui.LblPercent.SetText(fields.Percent)
				ui.LblSize.SetText(fields.Size)
				ui.LblScreen.SetText(fields.Encoding)
			}

			// Heavy per-mode operations executed only every 20 ticks.
			if count%20 == 0 {
				if CurrentView.Mode == Text && CurrentView.FemtoBuffer != nil {
					// Populate TblOutline with the function/symbol list.
					ui.TblOutline.Clear()
					var funcs = GetFuncs(CurrentView.FemtoBuffer.String(), CurrentView.FemtoBuffer.Settings["filetype"].(string))
					sort.Slice(funcs, func(i, j int) bool {
						a := funcs[i]
						b := funcs[j]
						return strings.ToUpper(a.name) < strings.ToUpper(b.name)
					})
					for i, f := range funcs {
						ui.TblOutline.SetCell(i, 0, tview.NewTableCell(strconv.Itoa(f.line)).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignRight))
						ui.TblOutline.SetCell(i, 1, tview.NewTableCell(f.name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
					}
					if !onlyOnce {
						ui.TblOutline.ScrollToBeginning()
						onlyOnce = true
					}
				} else if CurrentView.Mode == SQLite3 {
					// Show file metadata in the outline panel for databases.
					ui.DisplayExifInfo(CurrentView.FName)
				}
			}

			CurrentView = UpdateGITInfos(CurrentView)
			ui.LblGITBranch.SetText("⎇  " + CurrentView.GitBranch)
			ui.LblCommit.SetText("⟟ " + CurrentView.GitCommit)
			ui.LblGITStatus.SetText("🗨  " + CurrentView.GitStatus)
		})
	}
}

// ****************************************************************************
// UpdateGITInfos()
// ****************************************************************************
func UpdateGITInfos(f ViewScreen) ViewScreen {
	ws := filepath.Dir(f.FName)
	// Get GIT Commit
	commit, err2 := utils.Xeq(ws, "git", "rev-parse", "--short", "HEAD")
	if err2 != "" {
		commit = "No commit"
	}
	f.GitCommit = commit
	// Get GIT Status
	status, _ := utils.Xeq(ws, "git", "status", "-s")
	if status != "" {
		status = "Pending Commit"
	} else {
		status = "Up to date"
	}
	f.GitStatus = status
	// Get GIT Branch
	branch, err3 := utils.Xeq(ws, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err3 != "" {
		branch = "Unknown"
	}
	f.GitBranch = branch
	// Get GIT File Status
	fstatus, _ := utils.Xeq(ws, "git", "status", "-s", f.FName)
	if fstatus != "" {
		fstatus = fstatus[0:2]
	} else {
		fstatus = "  "
	}
	f.GitFileStatus = fstatus

	return f
}

// ****************************************************************************
// syncExplorerViewPath()
// ****************************************************************************
func syncExplorerViewPath(oldPath, newPath string) bool {
	for i := range OpenViews {
		if OpenViews[i].FName == oldPath {
			OpenViews[i].FName = newPath
			OpenViews[i].Mode = Explorer
			break
		}
	}
	if CurrentView.FName == oldPath {
		CurrentView.FName = newPath
		CurrentView.Mode = Explorer
		return true
	}
	return false
}

// ****************************************************************************
// SwitchOpenView()
// ****************************************************************************
func SwitchOpenView(fName string) {
	for _, e := range OpenViews {
		if e.FName == fName {
			CurrentView.FName = e.FName
			CurrentView.FemtoBuffer = e.FemtoBuffer
			CurrentView.Encoding = e.Encoding
			CurrentView.GitCommit = e.GitCommit
			CurrentView.GitStatus = e.GitStatus
			CurrentView.GitBranch = e.GitBranch
			CurrentView.ReadWrite = e.ReadWrite
			CurrentView.Follow = e.Follow
			CurrentView.Mode = e.Mode
			CurrentView.Database = e.Database
			CurrentView.Plugin = e.Plugin
			if utils.IsDir(CurrentView.FName) {
				CurrentView.Mode = Explorer
			}
			// Uniform dispatch: every mode now has a plugin that owns its
			// page/focus switching, key hints and focus widget.
			if CurrentView.Plugin != nil {
				CurrentWidget = CurrentView.Plugin.FocusWidget()
				CurrentView.Plugin.Activate()
				ui.LblKeys.SetText(CurrentView.Plugin.KeyHints())
			} else {
				// Fallback for views that were created without a plugin (should
				// not happen for properly initialised ViewScreens).
				CurrentWidget = ui.EdtMain
				ui.PgsApp.SwitchToPage("edit")
				ui.PgsEditorContent.SwitchToPage("textEditor")
			}

			if CurrentWidget != nil {
				ui.App.SetFocus(CurrentWidget)
			}

			// FocusOnPath(fName)
			ui.SetStatus(fmt.Sprintf("Switching to %s", CurrentView.FName))
			syncCurrentOpenViewSelection()
			go focusOpenFile(fName)
			break
		}
	}
	ShowTreeDir(conf.ConfigGeneral.Workspace, showHidden)
}

// ****************************************************************************
// SwitchAnyFile()
// ****************************************************************************
func SwitchAnyFile(fName any) {
	SwitchOpenView(fName.(string))
}

// ****************************************************************************
// SwitchPreviousFile()
// ****************************************************************************
func SwitchPreviousFile() {
	for i, e := range OpenViews {
		if e.FName == CurrentView.FName {
			prev := i - 1
			if prev < 0 {
				prev = len(OpenViews) - 1
			}
			SwitchOpenView(OpenViews[prev].FName)
			break
		}
	}
}

// ****************************************************************************
// SwitchNextFile()
// ****************************************************************************
func SwitchNextFile() {
	for i, e := range OpenViews {
		if e.FName == CurrentView.FName {
			next := i + 1
			if next == len(OpenViews) {
				next = 0
			}
			SwitchOpenView(OpenViews[next].FName)
			break
		}
	}
}

// ****************************************************************************
// syncCurrentOpenViewSelection()
// ****************************************************************************
func syncCurrentOpenViewSelection() {
	for idx, file := range OpenViews {
		if file.FName == CurrentView.FName {
			ui.TblOpenFiles.Select(idx, 0)
			return
		}
	}
}

// ****************************************************************************
// isViewAlreadyOpen()
// ****************************************************************************
func isViewAlreadyOpen(fName string) bool {
	rc := false
	for _, e := range OpenViews {
		if e.FName == fName {
			rc = true
			break
		}
	}
	return rc
}

// ****************************************************************************
// focusOpenFile()
// ****************************************************************************
func focusOpenFile(fName string) {
	_ = fName
	<-time.After(500 * time.Millisecond) // must be greater than the updateStatus sleep
	syncCurrentOpenViewSelection()
}

// ****************************************************************************
// GetGlobalDirtyFlag()
// ****************************************************************************
func GetGlobalDirtyFlag() bool {
	rc := false
	for _, f := range OpenViews {
		if f.Mode == PluginView {
			if f.Plugin != nil && f.Plugin.IsDirty() {
				rc = true
				break
			}
		} else if f.FemtoBuffer != nil && f.FemtoBuffer.Modified() {
			rc = true
			break
		}
	}
	return rc
}

// ****************************************************************************
// proposeToSaveFile()
// ****************************************************************************
func proposeToSaveFile(idx int, flow int) {
	currentFlow = flow
	DlgSaveFile = DlgSaveFile.YesNoCancel(fmt.Sprintf("Save File %s", OpenViews[idx].FName), // Title
		"This file has been modified. Do you want to save it ?", // Message
		confirmSave,
		idx,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgSaveFile", DlgSaveFile.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSaveFile")
	/*
	   DlgSave = tview.NewModal().

	   	SetText("Do you want to quit the application ?").
	   	AddButtons([]string{"Yes", "No", "Cancel"}).
	   	SetDoneFunc(func(buttonIndex int, buttonLabel string) {
	   		if buttonLabel == "Quit" {
	   			fQuit()
	   		} else {
	   			PgsApp.SwitchToPage(GetCurrentScreen())
	   		}
	   	})

	   DlgYesNo = DlgYesNo.YesNo("Git Fetch", // Title

	   	"The Git Fetch will fetch the remote version but no merging is processed locally.\n\nAre you sure you want to proceed ?", // Message
	   	func(rc dialog.DlgButton, idx int) {
	   		if rc == dialog.BUTTON_YES {
	   			out := fmt.Sprintf("Fetching...\n%s", XeqOut("git fetch origin"))
	   			MsgBox = MsgBox.OK("Git Fetch", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
	   			ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	   			ui.PgsApp.ShowPage("msgBox")
	   		} else {
	   			ui.SetStatus("Aborting Git Fetch")
	   		}
	   	},
	   	0,
	   	ui.GetCurrentScreen(), ui.EdtMain) // Focus return

	   ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
	   ui.PgsApp.ShowPage("dlgYesNo")
	*/
}

// ****************************************************************************
// confirmSave()
// ****************************************************************************
func confirmSave(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_YES {
		err := ioutil.WriteFile(OpenViews[idx].FName, []byte(OpenViews[idx].FemtoBuffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", OpenViews[idx].FName))
			OpenViews[idx].FemtoBuffer.IsModified = false
			if currentFlow == FLOW_CLOSE {
				CloseCurrentFile()
			} else if currentFlow == FLOW_QUIT {
				quitFlowIndex++
				startQuitSaveFlow()
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
	if rc == dialog.BUTTON_NO {
		OpenViews[idx].FemtoBuffer.IsModified = false
		if currentFlow == FLOW_CLOSE {
			CloseCurrentFile()
		} else if currentFlow == FLOW_QUIT {
			quitFlowIndex++
			startQuitSaveFlow()
		}
	}
	if rc == dialog.BUTTON_CANCEL {
		currentFlow = FLOW_NONE
		ui.PgsApp.SwitchToPage(ui.GetCurrentScreen()) // Return focus to the editor
		ui.App.SetFocus(ui.EdtMain)
	}
}

// ****************************************************************************
// confirmSaveAs()
// ****************************************************************************
func confirmSaveAs(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		newName := DlgSaveFileAs.Value
		err := ioutil.WriteFile(newName, []byte(CurrentView.FemtoBuffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", CurrentView.FName))
			CurrentView.FemtoBuffer.IsModified = false
			if currentFlow == FLOW_CLOSE {
				CloseCurrentFile()
			} else {
				var n = -1
				for i, f := range OpenViews {
					if f.FName == CurrentView.FName {
						n = i
						break
					}
				}
				copy(OpenViews[n:], OpenViews[n+1:])
				OpenViews = OpenViews[:len(OpenViews)-1]
				OpenView(newName, true)
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
	if rc == dialog.BUTTON_CANCEL {
		if currentFlow == FLOW_CLOSE {
			OpenViews[idx].FemtoBuffer.IsModified = false
			CloseCurrentFile()
		}
	}
	currentFlow = FLOW_NONE
}

// ****************************************************************************
// CheckOpenFilesForSaving()
// ****************************************************************************
func CheckOpenFilesForSaving() {
	quitFlowIndex = 0
	startQuitSaveFlow()
}

// ****************************************************************************
// startQuitSaveFlow()
// ****************************************************************************
func startQuitSaveFlow() {
	for ; quitFlowIndex < len(OpenViews); quitFlowIndex++ {
		f := OpenViews[quitFlowIndex]
		// Plugin views and non-text file modes never need saving.
		if f.Mode != SQLite3 && f.Mode != Explorer && f.Mode != PluginView {
			if f.FemtoBuffer.Modified() {
				ui.SetStatus(fmt.Sprintf("File %s is modified", f.FName))
				proposeToSaveFile(quitFlowIndex, FLOW_QUIT)
				return // Wait for user input from dialog
			}
		}
	}
	if conf.ConfigGeneral.CleanUpOnExit {
		deleted := cleanupTrackedNonameFiles()
		ui.SetStatus(fmt.Sprintf("Cleaned %d empty draft file(s)", deleted))
	}
	// All files checked, proceed to quit
	ui.App.Stop()
}

// ****************************************************************************
// CloseAll()
// ****************************************************************************
func CloseAll() {
	of := len(OpenViews)
	for ; quitFlowIndex < of; quitFlowIndex++ {
		CloseCurrentFile()
	}
}

// ****************************************************************************
// CloseCurrentFile()
// ****************************************************************************
func CloseCurrentFile() {
	var n = -1
	var d = ""
	for i, f := range OpenViews {
		if f.FName == CurrentView.FName {
			n = i
			d = filepath.Dir(f.FName)
			break
		}
	}
	if n >= 0 {
		onlyOnce = false
		ui.SetStatus("Closing file " + CurrentView.FName)
		if CurrentView.Mode == PluginView {
			// Plugin views: delegate teardown to the plugin, then remove from list.
			if CurrentView.Plugin != nil {
				CurrentView.Plugin.Close()
			}
			copy(OpenViews[n:], OpenViews[n+1:])
			OpenViews = OpenViews[:len(OpenViews)-1]
			ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
			if len(OpenViews) > 0 {
				next := n
				if next >= len(OpenViews) {
					next = len(OpenViews) - 1
				}
				CurrentView = OpenViews[next]
				SwitchOpenView(CurrentView.FName)
			} else {
				NewFile(conf.ConfigGeneral.Workspace)
			}
		} else if CurrentView.Mode == Text {
			if CurrentView.FemtoBuffer.IsModified {
				proposeToSaveFile(n, FLOW_CLOSE)
			} else {
				copy(OpenViews[n:], OpenViews[n+1:])
				OpenViews = OpenViews[:len(OpenViews)-1]
				ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
				if len(OpenViews) > 0 {
					next := n
					if next >= len(OpenViews) {
						next = len(OpenViews) - 1
					}
					CurrentView = OpenViews[next]
					SwitchOpenView(CurrentView.FName)
				} else {
					NewFile(d)
				}
			}
		} else {
			if CurrentView.Mode == SQLite3 {
				CloseDB(CurrentView.Database)
			}
			copy(OpenViews[n:], OpenViews[n+1:])
			OpenViews = OpenViews[:len(OpenViews)-1]
			ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
			if len(OpenViews) > 0 {
				next := n
				if next >= len(OpenViews) {
					next = len(OpenViews) - 1
				}
				CurrentView = OpenViews[next]
				SwitchOpenView(CurrentView.FName)
			} else {
				NewFile(d)
			}
		}
	}
}

// ****************************************************************************
// CloseThisFile()
// ****************************************************************************
func CloseThisFile(fName string) {
	var n = -1
	var d = ""
	for i, f := range OpenViews {
		if f.FName == fName {
			n = i
			d = filepath.Dir(f.FName)
			break
		}
	}
	if n >= 0 {
		onlyOnce = false
		ui.SetStatus("Closing file " + fName)
		CurrentView = OpenViews[n]
		if CurrentView.FemtoBuffer.IsModified {
			proposeToSaveFile(n, FLOW_CLOSE)
		} else {
			copy(OpenViews[n:], OpenViews[n+1:])
			OpenViews = OpenViews[:len(OpenViews)-1]
			ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Views (%d)", len(OpenViews)))
			if len(OpenViews) > 0 {
				next := n
				if next >= len(OpenViews) {
					next = len(OpenViews) - 1
				}
				CurrentView = OpenViews[next]
				SwitchOpenView(CurrentView.FName)
			} else {
				NewFile(d)
			}
		}
	}
}

// ****************************************************************************
// CloseAnyFile()
// ****************************************************************************
func CloseAnyFile(f any) {
	CloseCurrentFile()
}

// ****************************************************************************
// SwitchReadWrite()
// ****************************************************************************
func SwitchReadWrite(f any) {
	CurrentView.ReadWrite = !CurrentView.ReadWrite
	ui.SetStatus(fmt.Sprintf("Read Only attribute is set to %t", !CurrentView.ReadWrite))
}

// ****************************************************************************
// SwitchFollow()
// ****************************************************************************
func SwitchFollow(f any) {
	CurrentView.Follow = !CurrentView.Follow
	ui.SetStatus(fmt.Sprintf("Follow mode is set to %t", CurrentView.Follow))
	if CurrentView.Follow {
		CurrentView.RWBackup = CurrentView.ReadWrite
		CurrentView.ReadWrite = false
	} else {
		ui.SetStatus("Restoring buffer")
		content, err := ioutil.ReadFile(CurrentView.FName)
		if err != nil {
			ui.SetStatus(fmt.Sprintf("Could not read %v", CurrentView.FName))
			ui.SetStatus(fmt.Sprintf("%v", err))
		} else {
			CurrentView.FemtoBuffer = femto.NewBufferFromString(string(content), CurrentView.FName)
			CurrentView.FemtoBuffer.Cursor.Y = CurrentView.FemtoBuffer.End().Y
			if p, ok := CurrentView.Plugin.(*TextModePlugin); ok {
				p.buf = CurrentView.FemtoBuffer
			}
			ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
		}
	}
}

// ****************************************************************************
// GoBottom()
// ****************************************************************************
func GoBottom() {
	var loc femto.Loc
	loc.X = 0
	loc.Y = CurrentView.FemtoBuffer.End().Y
	CurrentView.FemtoBuffer.Cursor.GotoLoc(loc)
	ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
	ui.SetStatus("Go to bottom")
}

// ****************************************************************************
// GoTop()
// ****************************************************************************
func GoTop() {
	var loc femto.Loc
	loc.X = 0
	loc.Y = 0
	CurrentView.FemtoBuffer.Cursor.GotoLoc(loc)
	ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
	ui.SetStatus("Go to top")
}

// ****************************************************************************
// GoLine()
// ****************************************************************************
func GoLine(l int) {
	if l < 1 {
		ui.SetStatus(fmt.Sprintf("Jump outside bounds"))
		GoTop()
	} else {
		if l <= CurrentView.FemtoBuffer.LinesNum() {
			var loc femto.Loc
			loc.X = 0
			loc.Y = l - 1
			CurrentView.FemtoBuffer.Cursor.GotoLoc(loc)
			ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
			ui.SetStatus(fmt.Sprintf("Go to line #%d", l))
		} else {
			ui.SetStatus(fmt.Sprintf("Jump too far"))
			GoBottom()
		}
	}
}

// ****************************************************************************
// ShowTreeDir()
// ****************************************************************************
func ShowTreeDir(rootDir string, sh bool) {
	if CurrentView.Mode != SQLite3 {
		root := tview.NewTreeNode(rootDir).
			SetColor(tcell.ColorYellow)
		ui.TrvExplorer.SetRoot(root).SetCurrentNode(root)
		showHidden = sh

		// A helper function which adds the files and directories of the given path
		// to the given target node.
		/*
			add := func(target *tview.TreeNode, path string) {
				fileInfo, err := os.Stat(path)
				if err != nil {
					ui.SetStatus(err.Error())
				} else {
					if fileInfo.IsDir() {
						files, err := os.ReadDir(path)
						if err != nil {
							ui.SetStatus(err.Error())
						}
						for _, file := range files {
							node := tview.NewTreeNode(file.Name()).
								SetReference(filepath.Join(path, file.Name())).
								SetSelectable(file.IsDir() || file.Type().IsRegular())
							if file.IsDir() {
								node.SetColor(tcell.ColorGreen)
							}
						target.AddChild(node)
						}
					} else {
						mtype := utils.GetMimeType(path)
						if mtype[:4] == "text" {
							OpenFile(path, true)
							ui.SetStatus(fmt.Sprintf("Opening %s", path))
						} else {
							ui.SetStatus(fmt.Sprintf("%s is not a text file", path))
						}
					}
				}

			}
		*/

		// Add the current directory to the root node.
		addDirToNode(root, rootDir, showHidden)

		// If a directory was selected, open it.
		ui.TrvExplorer.SetSelectedFunc(selectNode)
	} else {
		showTreeDB()
	}
}

// ****************************************************************************
// selectNode()
// ****************************************************************************
func selectNode(node *tview.TreeNode) {
	reference := node.GetReference()
	if reference == nil {
		return // Selecting the root node does nothing.
	}
	children := node.GetChildren()
	if len(children) == 0 {
		// Load and show files in this directory.
		path := reference.(string)
		addDirToNode(node, path, showHidden)
	} else {
		// Collapse if visible, expand if collapsed.
		node.SetExpanded(!node.IsExpanded())
	}
}

// ****************************************************************************
// addDirToNode()
// ****************************************************************************
func addDirToNode(target *tview.TreeNode, path string, showHidden bool) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		ui.SetStatus(err.Error())
	} else {
		if fileInfo.IsDir() || (fileInfo.Mode()&os.ModeSymlink != 0) {
			files, err := os.ReadDir(path)
			if err != nil {
				ui.SetStatus(err.Error())
			}
			for _, file := range files {
				node := tview.NewTreeNode(file.Name()).
					SetReference(filepath.Join(path, file.Name())).
					SetSelectable(true)
				fi, er := os.Lstat(filepath.Join(path, file.Name()))
				if er == nil {
					if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
						node.SetColor(tcell.ColorBlue)
					}
				}
				if file.IsDir() {
					node.SetColor(tcell.ColorGreen)
				}
				if !showHidden {
					if file.Name()[0:1] != "." {
						target.AddChild(node)
					}
				} else {
					target.AddChild(node)
				}

			}
		} else {
			mtype := utils.GetMimeType(path)
			if len(mtype) >= 4 {
				if mtype[:4] == "text" {
					OpenView(path, true)
					ui.SetStatus(fmt.Sprintf("Opening %s", path))
					ui.App.SetFocus(ui.EdtMain)
				} else {
					OpenView(path, false)
					if CurrentView.Mode == SQLite3 {
						ui.SetStatus(fmt.Sprintf("Opening %s as a SQLite3 database", path))
						ui.App.SetFocus(ui.TxtPromptSQL)
					} else {
						ui.SetStatus(fmt.Sprintf("Opening %s in hexadecimal view", path))
						ui.App.SetFocus(ui.HexView)
					}
				}
			} else {
				ui.SetStatus(fmt.Sprintf("Can't open file %s of type %s", path, mtype))
			}
		}
	}
}

// ****************************************************************************
// SelfInit()
// ****************************************************************************
func SelfInit(a any) {
	NewFileOrLastFile(a.(string))
}

// ****************************************************************************
// SetFocusOnPath()
// ****************************************************************************
func SetFocusOnPath(fName string) {
	ui.SetStatus(fmt.Sprintf("Focusing on %s", fName))
	noeuds := strings.Split(fName, string(os.PathSeparator))
	noeuds = append([]string{"/"}, noeuds...)
	ref := ui.TrvExplorer.GetRoot()
	if ref == nil {
		ui.SetStatus("NIL")
	} else {
		for _, noeud := range noeuds {
			// fmt.Println(noeud)
			ui.SetStatus(fmt.Sprintf("Select Node %s", ref.GetText()))
			selectNode(ref)
			children := ref.GetChildren()
			if len(children) != 0 {
				for _, child := range children {
					ui.SetStatus(fmt.Sprintf("Select Children %s", child.GetText()))
					if child.GetText() == noeud {
						ui.SetStatus(fmt.Sprintf("Set Ref on Child %s", child.GetText()))
						child.SetExpanded(true)
						ui.TrvExplorer.SetCurrentNode(child)
						ref = child
						selectNode(ref)
						break
					}
				}
			}
		}
	}
}

// ****************************************************************************
// findStringInLines()
// ****************************************************************************
func findStringInLines(s string, fromLine int, fromColumn int, caseInsensitive bool) (int, int) {
	foundColumn := -1
	foundLine := -1

	// Iterate from the given line (0-based)
	for i := fromLine - 1; i < CurrentView.FemtoBuffer.NumLines; i++ {
		lineContent := CurrentView.FemtoBuffer.Line(i)
		searchString := s

		if caseInsensitive {
			lineContent = strings.ToLower(lineContent)
			searchString = strings.ToLower(searchString)
		}

		// Determine the starting index for the search on the current line
		startIndex := 0
		if i == fromLine-1 { // If we are on the starting line
			startIndex = fromColumn - 1 // Start searching from the given column (0-based)
			if startIndex < 0 {
				startIndex = 0 // Ensure startIndex is not negative
			}
		}

		// If the starting index is beyond the line length, no match possible on this line from this point
		if startIndex >= len(lineContent) {
			continue
		}

		// Search for the string from the determined startIndex
		idx := strings.Index(lineContent[startIndex:], searchString)
		if idx != -1 {
			// Calculate the absolute column index
			foundColumn = startIndex + idx
			foundLine = i
			break // Found it, so break the loop
		}
	}

	// Return 1-based line and column numbers
	return foundLine + 1, foundColumn + 1
}

// ****************************************************************************
// startFindSession()
// ****************************************************************************
func startFindSession(s string, caseInsensitive bool) {
	Founds = nil // Réinitialise les résultats de la recherche
	iFounds = 0

	tempFromL := 1
	tempFromC := 1
	for {
		l, c := findStringInLines(s, tempFromL, tempFromC, caseInsensitive)
		if l != 0 && c != 0 {
			Founds = append(Founds, found{s: s, l: l, c: c})
			// Pour la prochaine recherche, on commence juste après la trouvaille actuelle
			tempFromL = l
			// Correction : s'assurer que tempFromC ne dépasse pas la longueur de la ligne
			currentLineLen := len(CurrentView.FemtoBuffer.Line(tempFromL - 1))
			tempFromC = c + len(s) // Commence après la fin de la correspondance trouvée
			if tempFromC > currentLineLen {
				tempFromL++
				tempFromC = 1
			}
		} else {
			break
		}
	}
	iFounds = len(Founds)
	findSession = true // Marque qu'une session de recherche est active

	// Après avoir trouvé toutes les occurrences, trouver celle la plus proche du curseur actuel
	// pour initialiser currentFoundIndex.
	if len(Founds) > 0 {
		cursorLoc := CurrentView.FemtoBuffer.Cursor.Loc
		bestIndex := 0
		minDist := (CurrentView.FemtoBuffer.NumLines * 1000) // Une très grande distance

		for i, f := range Founds {
			// Calculer la "distance" du résultat par rapport au curseur actuel
			// Priorité à la ligne, puis à la colonne
			dist := (f.l-1-cursorLoc.Y)*1000 + (f.c - 1 - cursorLoc.X)
			if dist >= 0 && dist < minDist { // Cherche la première occurrence après ou sur le curseur
				minDist = dist
				bestIndex = i
			}
		}
		currentFoundIndex = bestIndex + 1 // Convertir en 1-basé
		ui.SetStatus(fmt.Sprintf("Search found %d occurrences. Initial focus on %d/%d.", len(Founds), currentFoundIndex, len(Founds)))
	} else {
		currentFoundIndex = 0
		ui.SetStatus("No occurrences found.")
	}
}

// ****************************************************************************
// startHexFindSession()
// ****************************************************************************
func startHexFindSession(s string, searchType string) {
	ui.SetStatus(fmt.Sprintf("startHexFindSession called. Search string: '%s', Search type: '%s'", s, searchType))
	HexFounds = nil
	hexCurrentFoundIndex = 0
	previousHexWhat = s
	previousHexSearchType = searchType
	hexFindSession = true

	var currentOffset int
	for {
		foundOffset := findStringInHexContent(s, searchType, currentOffset)
		if foundOffset != -1 {
			HexFounds = append(HexFounds, found{s: s, l: foundOffset})
			currentOffset = foundOffset + 1 // Start searching from the next byte
		} else {
			break
		}
	}
	ui.SetStatus(fmt.Sprintf("startHexFindSession finished. Found %d occurrences.", len(HexFounds)))
	CurrentView.HexContentDirty = false
}

// ****************************************************************************
// findStringInHexContent()
// ****************************************************************************
func findStringInHexContent(s string, searchType string, fromOffset int) int {
	ui.SetStatus(fmt.Sprintf("findStringInHexContent called. Search string: '%s', Search type: '%s', From offset: %d", s, searchType, fromOffset))
	if fromOffset >= len(CurrentView.ContentBytes) {
		ui.SetStatus("findStringInHexContent: From offset is beyond content length.")
		return -1
	}

	contentToSearch := CurrentView.ContentBytes[fromOffset:]

	if searchType == "ASCII" {
		// ASCII search
		if ui.ChkCase.IsChecked() { // Case-sensitive
			idx := bytes.Index(contentToSearch, []byte(s))
			if idx != -1 {
				ui.SetStatus(fmt.Sprintf("findStringInHexContent: ASCII (case-sensitive) found at relative index %d", idx))
				return fromOffset + idx
			}
		} else { // Case-insensitive
			searchBytesLower := bytes.ToLower([]byte(s))
			contentBytesLower := bytes.ToLower(contentToSearch)
			idx := bytes.Index(contentBytesLower, searchBytesLower)
			if idx != -1 {
				ui.SetStatus(fmt.Sprintf("findStringInHexContent: ASCII (case-insensitive) found at relative index %d", idx))
				return fromOffset + idx
			}
		}
	} else if searchType == "Hexadecimal" {
		// Validate hex string
		if len(s)%2 != 0 {
			ui.SetStatus("Invalid hexadecimal string: must have an even number of characters")
			return -1
		}
		hexBytes, err := hex.DecodeString(s)
		if err != nil {
			ui.SetStatus("Invalid hexadecimal string: " + err.Error())
			return -1
		}
		idx := bytes.Index(contentToSearch, hexBytes)
		if idx != -1 {
			ui.SetStatus(fmt.Sprintf("findStringInHexContent: Hexadecimal found at relative index %d", idx))
			return fromOffset + idx
		}
	}

	ui.SetStatus("findStringInHexContent: No match found.")
	return -1
}

// ****************************************************************************
// FindNext()
// ****************************************************************************
func FindNext() {
	if CurrentView.Mode == Binary {
		findHexNext()
	} else {
		findTextNavigate(FIND_DOWN) // <--- Passage de direction
	}
}

// ****************************************************************************
// FindPrevious()
// ****************************************************************************
func FindPrevious() {
	if CurrentView.Mode == Binary {
		findHexPrevious()
	} else {
		findTextNavigate(FIND_UP) // <--- Passage de direction
	}
}

// ****************************************************************************
// findTextNavigate() - FUSIONNÉE
// ****************************************************************************
func findTextNavigate(direction int) {
	searchString := ui.TxtFind.GetText()
	caseInsensitive := !ui.ChkCase.IsChecked()

	// Si la recherche ou le fichier a changé, relancer une nouvelle session
	if searchString != lastSearchString || caseInsensitive != lastSearchCase || CurrentView.FemtoBuffer.Modified() {
		ui.SetStatus(fmt.Sprintf("Starting new search session for '%s'...", searchString))
		startFindSession(searchString, caseInsensitive) // Cette fonction doit remplir Founds
		lastSearchString = searchString
		lastSearchCase = caseInsensitive
		// Après une nouvelle session, l'index doit pointer vers le début
		currentFoundIndex = 0
		if len(Founds) == 0 {
			ui.SetStatus(fmt.Sprintf("No occurrences of '%s' found.", searchString))
			ui.FrmFind.SetTitle("Find & Replace (0/0)")
			return
		}
	} else if len(Founds) == 0 { // Aucune occurrence trouvée lors de la session précédente
		ui.SetStatus(fmt.Sprintf("No occurrences of '%s' found.", searchString))
		ui.FrmFind.SetTitle("Find & Replace (0/0)")
		return
	}

	// Naviguer dans les occurrences trouvées
	if direction == FIND_DOWN {
		if currentFoundIndex < len(Founds) {
			currentFoundIndex++
		} else { // Wrap around
			currentFoundIndex = 1
			ui.SetStatus("Wrapping around to first occurrence.")
		}
	} else if direction == FIND_UP {
		if currentFoundIndex > 1 {
			currentFoundIndex--
		} else { // Wrap around
			currentFoundIndex = len(Founds)
			ui.SetStatus("Wrapping around to last occurrence.")
		}
	}

	if len(Founds) > 0 {
		// Ajuster l'index pour accéder à la slice (0-basé)
		displayIndex := currentFoundIndex - 1
		foundItem := Founds[displayIndex]

		ui.SetStatus(fmt.Sprintf("Found at Line %d, Col %d (%d/%d)", foundItem.l, foundItem.c, currentFoundIndex, len(Founds)))

		var loc femto.Loc
		loc.X = foundItem.c - 1 // Convertir en 0-basé
		loc.Y = foundItem.l - 1 // Convertir en 0-basé

		CurrentView.FemtoBuffer.Cursor.GotoLoc(loc)
		// CurrentView.FemtoBuffer.Make // Cette ligne était peut-être une erreur de frappe ? (pas de méthode .Make sur FemtoBuffer)
		ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer) // S'assure que le buffer est bien affiché

		ui.FrmFind.SetTitle(fmt.Sprintf("Find & Replace (%d/%d)", currentFoundIndex, len(Founds)))
	} else {
		ui.SetStatus("Nothing found.")
		ui.FrmFind.SetTitle("Find & Replace (0/0)")
	}
}

// ****************************************************************************
// findHexNext()
// ****************************************************************************
func findHexNext() {
	whatToFind := ui.TxtFind.GetText()
	_, searchType := ui.DpdSearchType.GetCurrentOption()
	ui.SetStatus(fmt.Sprintf("findHexNext called. Search string: '%s', Search type: '%s'", whatToFind, searchType))

	if whatToFind != previousHexWhat || searchType != previousHexSearchType || CurrentView.HexContentDirty {
		ui.SetStatus("Starting new hex find session...")
		go startHexFindSession(whatToFind, searchType)
		time.Sleep(300 * time.Millisecond)
	}

	if len(HexFounds) > 0 {
		if hexCurrentFoundIndex < len(HexFounds) {
			foundItem := HexFounds[hexCurrentFoundIndex]
			ui.SetStatus(fmt.Sprintf("Found at Offset %08X (index %d/%d)", foundItem.l, hexCurrentFoundIndex+1, len(HexFounds)))
			hexCurrentFoundIndex++
			ui.FrmFind.SetTitle(fmt.Sprintf("Find (%d/%d)", hexCurrentFoundIndex, len(HexFounds)))
			displayBinaryContent() // Refresh to highlight and scroll
		} else {
			ui.SetStatus("No more found. Wrapping around.")
			hexCurrentFoundIndex = 0
			if len(HexFounds) > 0 { // If there are any finds, wrap to the first one
				foundItem := HexFounds[hexCurrentFoundIndex]
				ui.SetStatus(fmt.Sprintf("Found at Offset %08X (index %d/%d)", foundItem.l, hexCurrentFoundIndex+1, len(HexFounds)))
				hexCurrentFoundIndex++
				ui.FrmFind.SetTitle(fmt.Sprintf("Find (%d/%d)", hexCurrentFoundIndex, len(HexFounds)))
				displayBinaryContent()
			}
		}
	} else {
		ui.FrmFind.SetTitle("Find")
		ui.SetStatus("Nothing found for current search.")
	}
}

// ****************************************************************************
// findHexPrevious()
// ****************************************************************************
func findHexPrevious() {
	whatToFind := ui.TxtFind.GetText()
	_, searchType := ui.DpdSearchType.GetCurrentOption()
	ui.SetStatus(fmt.Sprintf("findHexPrevious called. Search string: '%s', Search type: '%s'", whatToFind, searchType))

	if whatToFind != previousHexWhat || searchType != previousHexSearchType || CurrentView.HexContentDirty {
		ui.SetStatus("Starting new hex find session (previous)....")
		go startHexFindSession(whatToFind, searchType)
		time.Sleep(300 * time.Millisecond)
	}

	if len(HexFounds) > 0 {
		if hexCurrentFoundIndex > 0 {
			hexCurrentFoundIndex--
			foundItem := HexFounds[hexCurrentFoundIndex]
			ui.SetStatus(fmt.Sprintf("Found at Offset %08X (index %d/%d)", foundItem.l, hexCurrentFoundIndex+1, len(HexFounds)))
			ui.FrmFind.SetTitle(fmt.Sprintf("Find (%d/%d)", hexCurrentFoundIndex+1, len(HexFounds)))
			displayBinaryContent() // Refresh to highlight and scroll
		} else {
			ui.SetStatus("No more found. Wrapping around to end.")
			hexCurrentFoundIndex = len(HexFounds) - 1
			if len(HexFounds) > 0 { // If there are any finds, wrap to the last one
				foundItem := HexFounds[hexCurrentFoundIndex]
				ui.SetStatus(fmt.Sprintf("Found at Offset %08X (index %d/%d)", foundItem.l, hexCurrentFoundIndex+1, len(HexFounds)))
				ui.FrmFind.SetTitle(fmt.Sprintf("Find (%d/%d)", hexCurrentFoundIndex+1, len(HexFounds)))
				displayBinaryContent()
			}
		}
	} else {
		ui.FrmFind.SetTitle("Find")
		ui.SetStatus("Nothing found for current search.")
	}
}

// ****************************************************************************
// ReplaceOne()
// ****************************************************************************
func ReplaceOne() {
	if CurrentView.Mode == Binary {
		ui.SetStatus("Cannot replace in binary files (read-only)")
		return
	}

	if len(Founds) == 0 {
		searchString := ui.TxtFind.GetText()
		caseInsensitive := !ui.ChkCase.IsChecked()
		if searchString != "" {
			startFindSession(searchString, caseInsensitive)
			time.Sleep(100 * time.Millisecond)
		}
		if len(Founds) == 0 {
			ui.SetStatus("Nothing found to replace.")
			return
		}
	}

	if currentFoundIndex <= 0 || currentFoundIndex > len(Founds) {
		currentFoundIndex = 1
		if len(Founds) == 0 {
			ui.SetStatus("Nothing found to replace.")
			return
		}
	}

	foundItem := Founds[currentFoundIndex-1]
	replaceText := ui.TxtReplace.GetText()
	whatToFind := ui.TxtFind.GetText()

	start := femto.Loc{X: foundItem.c - 1, Y: foundItem.l - 1}
	end := femto.Loc{X: start.X + len(whatToFind), Y: start.Y}

	CurrentView.FemtoBuffer.Replace(start, end, replaceText)
	CurrentView.FemtoBuffer.IsModified = true

	ui.SetStatus(fmt.Sprintf("Replaced '%s' with '%s' at Line %d, Col %d", whatToFind, replaceText, foundItem.l, foundItem.c))

	searchString := ui.TxtFind.GetText()
	caseInsensitive := !ui.ChkCase.IsChecked()

	startFindSession(searchString, caseInsensitive)
	time.Sleep(100 * time.Millisecond)

	// Après un remplacement, le curseur est probablement après le texte remplacé.
	// Nous voulons naviguer vers la *prochaine* occurrence valide.
	findTextNavigate(FIND_DOWN)
	ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
	ui.App.SetFocus(ui.EdtMain)
}

// ****************************************************************************
// ReplaceAll()
// ****************************************************************************
func ReplaceAll() {
	if CurrentView.Mode == Binary {
		ui.SetStatus("Cannot replace in binary files (read-only)")
		return
	}
	whatToFind := ui.TxtFind.GetText()
	if whatToFind == "" {
		ui.SetStatus("Nothing to find")
		return
	}
	replaceText := ui.TxtReplace.GetText()
	caseSensitive := ui.ChkCase.IsChecked()

	bufferContent := CurrentView.FemtoBuffer.String()
	var newBufferContent string
	var replacements int

	if caseSensitive {
		replacements = strings.Count(bufferContent, whatToFind)
		newBufferContent = strings.ReplaceAll(bufferContent, whatToFind, replaceText) // Use ReplaceAll
	} else {
		re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(whatToFind))
		if err != nil {
			ui.SetStatus("Error in find pattern: " + err.Error())
			return
		}
		matches := re.FindAllStringIndex(bufferContent, -1)
		replacements = len(matches)
		newBufferContent = re.ReplaceAllString(bufferContent, replaceText)
	}

	if replacements > 0 {
		cursor := CurrentView.FemtoBuffer.Cursor // Sauvegarder la position du curseur
		CurrentView.FemtoBuffer = femto.NewBufferFromString(newBufferContent, CurrentView.FName)
		CurrentView.FemtoBuffer.IsModified = true
		// Keep the per-entry plugin's buffer pointer in sync.
		if p, ok := CurrentView.Plugin.(*TextModePlugin); ok {
			p.buf = CurrentView.FemtoBuffer
		}

		for i, e := range OpenViews {
			if e.FName == CurrentView.FName {
				OpenViews[i].FemtoBuffer = CurrentView.FemtoBuffer
				if p, ok := OpenViews[i].Plugin.(*TextModePlugin); ok {
					p.buf = OpenViews[i].FemtoBuffer
				}
				break
			}
		}
		ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)

		// CORRECTION : Implémenter GotoEnd manuellement
		if cursor.Y < CurrentView.FemtoBuffer.NumLines {
			CurrentView.FemtoBuffer.Cursor.Y = cursor.Y
			lineLen := len(CurrentView.FemtoBuffer.Line(cursor.Y))
			if cursor.X < lineLen {
				CurrentView.FemtoBuffer.Cursor.X = cursor.X
			} else {
				CurrentView.FemtoBuffer.Cursor.X = lineLen
			}
		} else {
			// Aller à la fin du document si l'ancienne ligne n'existe plus
			lastLine := CurrentView.FemtoBuffer.NumLines - 1
			if lastLine < 0 {
				lastLine = 0
			} // Gérer le cas d'un buffer vide
			CurrentView.FemtoBuffer.Cursor.Y = lastLine
			CurrentView.FemtoBuffer.Cursor.X = len(CurrentView.FemtoBuffer.Line(lastLine))
		}

		SetTheme(conf.ConfigGeneral.Theme)
		ui.SetStatus(fmt.Sprintf("%d replacement(s) made", replacements))
		ui.App.SetFocus(ui.EdtMain)

		lastSearchString = ""
		Founds = nil
		iFounds = 0
		currentFoundIndex = 0
		ui.FrmFind.SetTitle("Find & Replace (0/0)")
	} else {
		ui.SetStatus("Nothing to replace")
	}
}

// ****************************************************************************
// ReplaceOnlyThisOne()
// ****************************************************************************
func ReplaceOnlyThisOne(l int, c int, n int, subst string) {
	array := bytes.Split([]byte(CurrentView.FemtoBuffer.String()), []byte("\n"))
	line := array[l-1]
	sLine := string(line[:])
	ui.SetStatus(sLine)
	// nLine := sLine[:c-1] + subst + sLine[c-1+n:]
	// outBuffer := strings.Join(string(array[:l-2]),"\n")
}

// ****************************************************************************
// RecallFind()
// ****************************************************************************
func RecallFind(way int) {
	if len(AFind) > 0 {
		if way == FIND_UP {
			if IFind > 0 {
				IFind--
			} else {
				IFind = len(AFind) - 1
			}
		} else {
			if IFind < len(AFind)-1 {
				IFind++
			} else {
				IFind = 0
			}
		}
		ui.TxtFind.SetText(AFind[IFind])
	} else {
		ui.SetStatus("Find history is empty")
	}
}

// ****************************************************************************
// InsertString()
// ****************************************************************************
func InsertString(txt string) {
	CurrentView.FemtoBuffer.Insert(CurrentView.FemtoBuffer.Cursor.Loc, txt)
}

// ****************************************************************************
// stripTviewColorTags removes tview color tags from a string.
// ****************************************************************************
func stripTviewColorTags(s string) string {
	re := regexp.MustCompile(`\[[a-zA-Z0-9,:#]+\]`)
	return re.ReplaceAllString(s, "")
}

// ****************************************************************************
// displayBinaryContent()
// ****************************************************************************
func displayBinaryContent() {
	hexAndAsciiStr := utils.BytesToHexAndASCII(CurrentView.ContentBytes)

	// Split the content into lines for individual processing
	lines := strings.Split(hexAndAsciiStr, "\n")
	var highlightedLines []string

	// Constants for line parsing (adjust if BytesToHexAndASCII changes)
	const (
		// Lengths of tview color tags
		yellowTagLen = 8 // "[yellow]"
		whiteTagLen  = 7 // "[white]"

		// Character counts in the *formatted* string, including spaces and color tags
		offsetLenWithTags = 8 + yellowTagLen + whiteTagLen // 8 (offset) + 8 (yellow) + 7 (white) = 23
		offsetSpaces      = 3
		hexByteLen        = 2
		hexByteSpace      = 1
		hexPartRawLen     = (hexByteLen+hexByteSpace)*16 - hexByteSpace // 47 (16 bytes, 15 spaces)
		hexAsciiSeparator = 2
		asciiPartLen      = 16

		// Start indices within a formatted line (including color tags)
		hexPartStartCol   = offsetLenWithTags + offsetSpaces
		asciiPartStartCol = hexPartStartCol + hexPartRawLen + hexAsciiSeparator
	)

	if hexFindSession && len(HexFounds) > 0 && hexCurrentFoundIndex > 0 && hexCurrentFoundIndex <= len(HexFounds) {
		foundItem := HexFounds[hexCurrentFoundIndex-1]
		searchLenBytes := len(foundItem.s)
		// _, searchTypeOption := ui.DpdSearchType.GetCurrentOption()

		// Iterate through each line to apply highlighting
		for lineIdx, line := range lines {
			if line == "" { // Skip empty lines (e.g., last line after split)
				highlightedLines = append(highlightedLines, line)
				continue
			}

			lineStartByteOffset := lineIdx * 16
			lineEndByteOffset := lineStartByteOffset + 16

			// Check if the found item starts within this line's byte range
			if foundItem.l >= lineStartByteOffset && foundItem.l < lineEndByteOffset {
				// Calculate the portion of the match that falls on this line
				matchStartByteOnLine := foundItem.l
				matchEndByteOnLine := int(math.Min(float64(foundItem.l+searchLenBytes), float64(lineEndByteOffset)))
				currentLineMatchLen := matchEndByteOnLine - matchStartByteOnLine

				if currentLineMatchLen > 0 {
					var sbLine strings.Builder

					// Calculate relative column in the 16-byte block
					colInLine := matchStartByteOnLine % 16

					// Calculate start and end indices for highlighting in the current line (with tags)
					hexHighlightStart := hexPartStartCol + colInLine*(hexByteLen+hexByteSpace)
					hexHighlightEnd := hexHighlightStart + (currentLineMatchLen * hexByteLen) + (currentLineMatchLen-1)*hexByteSpace - 1
					asciiHighlightStart := asciiPartStartCol + colInLine
					asciiHighlightEnd := asciiHighlightStart + currentLineMatchLen

					// Apply bounds checking before slicing
					hexHighlightStart = int(math.Min(float64(hexHighlightStart), float64(len(line))))
					hexHighlightEnd = int(math.Min(float64(hexHighlightEnd), float64(len(line)-1))) // -1 because slice end is exclusive
					asciiHighlightStart = int(math.Min(float64(asciiHighlightStart), float64(len(line))))
					asciiHighlightEnd = int(math.Min(float64(asciiHighlightEnd), float64(len(line))))

					// Ensure start indices are not greater than end indices after bounds checking
					if hexHighlightStart > hexHighlightEnd {
						hexHighlightEnd = hexHighlightStart
					}
					if asciiHighlightStart > asciiHighlightEnd {
						asciiHighlightEnd = asciiHighlightStart
					}

					// Reconstruct the line with highlighting tags
					sbLine.WriteString(line[:hexHighlightStart])
					sbLine.WriteString("[::b]")
					sbLine.WriteString(line[hexHighlightStart : hexHighlightEnd+1])
					sbLine.WriteString("[::-]")
					sbLine.WriteString(line[hexHighlightEnd+1 : asciiHighlightStart])
					sbLine.WriteString("[::b]")
					sbLine.WriteString(line[asciiHighlightStart:asciiHighlightEnd])
					sbLine.WriteString("[::-]")
					sbLine.WriteString(line[asciiHighlightEnd:])
					highlightedLines = append(highlightedLines, sbLine.String())

					// If the match spans multiple lines, we need to continue highlighting on subsequent lines
					// For now, we'll just highlight the first part and indicate a partial highlight.
					if matchEndByteOnLine < foundItem.l+searchLenBytes {
						ui.SetStatus(fmt.Sprintf("Pattern '%s' found at Offset %08X (multi-line match, partial highlight)", foundItem.s, foundItem.l))
					} else {
						ui.SetStatus(fmt.Sprintf("Pattern '%s' found at Offset %08X", foundItem.s, foundItem.l))
					}
				} else {
					highlightedLines = append(highlightedLines, line)
				}
			} else {
				highlightedLines = append(highlightedLines, line)
			}
		}
		hexAndAsciiStr = strings.Join(highlightedLines, "\n")
	}

	ui.HexView.SetText(hexAndAsciiStr)

	// Scroll to the currently found item if a search is active
	if hexFindSession && len(HexFounds) > 0 && hexCurrentFoundIndex > 0 && hexCurrentFoundIndex <= len(HexFounds) {
		foundItem := HexFounds[hexCurrentFoundIndex-1]
		lineNum := foundItem.l / 16
		ui.HexView.ScrollTo(lineNum, 0)
	}

	// Update LblCursor and LblPercent based on the current found item, if any
	if hexFindSession && len(HexFounds) > 0 && hexCurrentFoundIndex > 0 && hexCurrentFoundIndex <= len(HexFounds) {
		foundItem := HexFounds[hexCurrentFoundIndex-1]
		ui.LblCursor.SetText(fmt.Sprintf("Offset: %08X", foundItem.l))

		// Calculate percentage based on the found offset
		totalBytes := len(CurrentView.ContentBytes)
		if totalBytes > 0 {
			percent := int((float64(foundItem.l) / float64(totalBytes)) * 100.0)
			ui.LblPercent.SetText(fmt.Sprintf("%d%%", percent))
		} else {
			ui.LblPercent.SetText("0%")
		}
	} else {
		// If no search is active or no item found, revert to initial status
		ui.LblCursor.SetText(fmt.Sprintf("Offset: %08X", 0))
		_, _, _, height := ui.HexView.GetInnerRect()
		hexContent := ui.HexView.GetText(false)
		totalLines := strings.Count(hexContent, "\n")
		if totalLines > height {
			ui.LblPercent.SetText("0%") // At the beginning, 0% scrolled
		} else {
			ui.LblPercent.SetText("100%") // All content visible
		}
	}
}

// ****************************************************************************
// Reload()
// ****************************************************************************
func (f *ViewScreen) Reload() error {
	oldX, oldY := f.FemtoBuffer.Cursor.X, f.FemtoBuffer.Cursor.Y
	data, err := os.ReadFile(f.FName)
	if err != nil {
		return err
	}
	f.FemtoBuffer.Replace(f.FemtoBuffer.Start(), f.FemtoBuffer.End(), string(data))
	f.FemtoBuffer.Cursor.Y = oldY
	f.FemtoBuffer.Cursor.X = oldX
	f.FemtoBuffer.Cursor.Relocate()
	f.FemtoBuffer.IsModified = false

	return nil
}
