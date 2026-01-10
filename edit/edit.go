// ****************************************************************************
//
//	 _ _          _
//	| (_) ___  __| |
//	| | |/ _ \\/ _` |
//	| | |  __/ (_| |
//	|_|_|\\___|\\__,_|\
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
type editfile struct {
	Buffer              *femto.Buffer
	View                *femto.View
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
	IsBinary            bool
	ContentBytes        []byte
	HexContentDirty     bool
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
	OpenFiles             []editfile
	CurrentFile           editfile
	CurrentView           tview.Primitive
	DlgSaveFile           *dialog.Dialog
	DlgSaveFileAs         *dialog.Dialog
	currentFlow           int
	showHidden            bool
	Founds                []found
	iFounds               int
	currentFoundIndex     int
	findSession           bool
	previousWhat          string
	whatToFind            string
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
)

// ****************************************************************************
// SwitchToEditor()
// ****************************************************************************
func SwitchToEditor(fName string) {
	ui.CurrentMode = ui.ModeTextEdit
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
	ui.PgsApp.SwitchToPage(scr) // ???
	// ShowTreeDir(filepath.Dir(fName))
	// ShowTreeDir("/")
	OpenFile(fName, true)
	if CurrentFile.IsBinary {
		ui.App.SetFocus(ui.HexView)
	} else {
		ui.App.SetFocus(ui.EdtMain)
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

// ****************************************************************************
// OpenFile()
// ****************************************************************************
func OpenFile(fName string, rw bool) {
	onlyOnce = false
	if isFileAlreadyOpen(fName) {
		SwitchOpenFile(fName)
	} else {
		ui.EdtMain.SetRuntimeFiles(runtime.Files)
		// Check if the file exists
		if _, err := os.Stat(fName); errors.Is(err, os.ErrNotExist) {
			ui.SetStatus(fmt.Sprintf("File %v doesn't exist", fName))
			CreateThisFile(fName)
		} else {
			// Check if the file is binary
			if utils.IsBinaryFile(fName) {
				content, err := ioutil.ReadFile(fName)
				if err != nil {
					ui.SetStatus(fmt.Sprintf("Could not read binary file %v: %v", fName, err))
					return
				}
				CurrentFile.FName = fName
				CurrentFile.IsBinary = true
				CurrentFile.ReadWrite = false // Binary files are read-only for now
				CurrentFile.Follow = false
				CurrentFile.Encoding = "Binary"
				CurrentFile.ContentBytes = content
				CurrentFile.HexContentDirty = true // Mark for refresh
				OpenFiles = append(OpenFiles, CurrentFile)
				go UpdateStatus()
				go focusOpenFile(fName)
				ui.SetStatus(fmt.Sprintf("Opening binary file %s", CurrentFile.FName))
				ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Files (%d)", len(OpenFiles)))
				ui.App.SetFocus(ui.HexView) // Set focus to HexView for binary files
				ui.PgsEditorContent.SwitchToPage("hexViewer")
				displayBinaryContent()
				ui.DisplayExifInfo(CurrentFile.FName) // Display EXIF info for binary files

				// Configure FrmFind for binary files
				ui.ConfigureFindFormForBinary(true)
				return
			}

			content, err := ioutil.ReadFile(fName)
			if err != nil {
				ui.SetStatus(fmt.Sprintf("Could not read %v", fName))
				ui.SetStatus(fmt.Sprintf("%v", err))
			} else {
				detector := chardet.NewTextDetector()
				result, err := detector.DetectBest(content)
				if err == nil {
					CurrentFile.Encoding = result.Charset
				} else {
					CurrentFile.Encoding = "Unknown"
				}

				CurrentFile.FName = fName
				CurrentFile.Buffer = femto.NewBufferFromString(string(content), CurrentFile.FName)
				// 				CurrentFile.Buffer.Settings["wordwrap"] = false
				CurrentFile.Buffer.Settings["keepautoindent"] = true
				CurrentFile.Buffer.Settings["softwrap"] = true
				CurrentFile.Buffer.Settings["scrollbar"] = true
				CurrentFile.Buffer.Settings["statusline"] = false

				CurrentFile.View = femto.NewView(CurrentFile.Buffer)
				CurrentFile.ReadWrite = rw
				CurrentFile.Follow = false
				CurrentFile.IsBinary = false
				CurrentFile.ContentBytes = nil // Ensure this is nil for text files
				ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
				SetTheme("monokai")
				ui.EdtMain.SetTitleAlign(tview.AlignRight)
				ui.LblScreen.SetText(CurrentFile.Encoding)
				CurrentFile = UpdateGITInfos(CurrentFile)
				OpenFiles = append(OpenFiles, CurrentFile)
				go UpdateStatus()
				go focusOpenFile(fName)
				ui.SetStatus(fmt.Sprintf("Opening file %s", CurrentFile.FName))
				ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Files (%d)", len(OpenFiles)))
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
	err := ioutil.WriteFile(CurrentFile.FName, []byte(CurrentFile.Buffer.String()), 0600)
	if err == nil {
		ui.SetStatus(fmt.Sprintf("File %s successfully saved", CurrentFile.FName))
		CurrentFile.Buffer.IsModified = false
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
	of := len(OpenFiles)
	for ; fIndex < of; fIndex++ {
		err := ioutil.WriteFile(OpenFiles[fIndex].FName, []byte(OpenFiles[fIndex].Buffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", OpenFiles[fIndex].FName))
			OpenFiles[fIndex].Buffer.IsModified = false
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
		CurrentFile.FName,
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
		// SwitchToEditor(f.Name())
		OpenFile(f.Name(), true)
	} else {
		ui.SetStatus(err.Error())
	}
}

// ****************************************************************************
// CreateThisFile()
// ****************************************************************************
func CreateThisFile(dir string) {
	f, err := os.Create(dir)
	ui.SetStatus(fmt.Sprintf("Creating the file %v", dir))
	if err == nil {
		OpenFile(f.Name(), true)
	} else {
		ui.SetStatus(err.Error())
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
	if len(OpenFiles) > 0 {
		SwitchToEditor(CurrentFile.FName)
	} else {
		NewFile(dir)
	}
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
			relativePath, err := filepath.Rel(conf.ConfigGeneral.Workspace, CurrentFile.FName)
			if err != nil {
				// Handle error: perhaps log it or display a default value
				relativePath = filepath.Base(CurrentFile.FName)
			}
			dirPath := filepath.Dir(relativePath)
			if dirPath == "." {
				dirPath = ""
			} else {
				dirPath += string(os.PathSeparator)
			}
			ui.TxtCurrentEditName.SetText(dirPath + "[yellow]" + filepath.Base(relativePath))
			if CurrentFile.Buffer.Modified() {
				// status = conf.ICON_MODIFIED
				ui.LblDirty.SetText("*modified*")
			} else {
				// status = " "
				ui.LblDirty.SetText("")
			}
			CurrentFile = UpdateGITInfos(CurrentFile)
			ui.LblGITBranch.SetText("⎇  " + CurrentFile.GitBranch)
			ui.LblCommit.SetText("⟟ " + CurrentFile.GitCommit)
			ui.LblGITStatus.SetText("🗨  " + CurrentFile.GitStatus)
			ui.EdtMain.SetTitle(fmt.Sprintf("[ %s %s %s ]", CurrentFile.Encoding, CurrentFile.Buffer.Settings["filetype"].(string), CurrentFile.Buffer.Settings["fileformat"].(string)))
			if CurrentFile.Follow {
				_, _, _, lines := ui.EdtMain.GetInnerRect()
				ui.LblReadWrite.SetText("FL")
				c := exec.Command("tail", "-n", strconv.Itoa(lines-1), CurrentFile.FName)
				output, _ := c.Output()
				CurrentFile.Buffer = femto.NewBufferFromString(string(output), CurrentFile.FName)
				CurrentFile.Buffer.Cursor.Y = CurrentFile.Buffer.End().Y
				ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
			}
			ui.LblSize.SetText(utils.HumanFileSize(float64(CurrentFile.Buffer.Len())))
			ui.TblOpenFiles.Clear()
			count++
			for i, f := range OpenFiles {
				if count%20 == 0 {
					// Update GIT infos only once in 20 to prevent huge CPU use
					f = UpdateGITInfos(f)
					OpenFiles[i] = f
				}
				if f.Buffer.Modified() {
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(conf.ICON_MODIFIED+f.GitFileStatus))
				} else {
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(" "+f.GitFileStatus))
				}
				ui.TblOpenFiles.SetCell(i, 1, tview.NewTableCell(filepath.Base(f.FName)))
				ui.TblOpenFiles.SetCell(i, 2, tview.NewTableCell("⯈"))
				ui.TblOpenFiles.SetCell(i, 3, tview.NewTableCell(f.FName))
			}

			if CurrentFile.IsBinary {
				ui.LblSize.SetText(utils.HumanFileSize(float64(len(CurrentFile.ContentBytes))))
				ui.EdtMain.SetTitle(fmt.Sprintf("[ %s ]", CurrentFile.Encoding))
				ui.LblScreen.SetText(CurrentFile.Encoding)
				ui.LblReadWrite.SetText("RO")
				ui.LblDirty.SetText("")
			} else {
				x := CurrentFile.Buffer.Cursor.X + 1
				y := CurrentFile.Buffer.Cursor.Y + 1
				ui.LblCursor.SetText(fmt.Sprintf("Ln %d, Col %d", y, x))
				ui.LblPercent.SetText(fmt.Sprintf("%d%%", int((float32(CurrentFile.Buffer.Cursor.Y)/float32(CurrentFile.Buffer.NumLines))*100.0)))
				if CurrentFile.ReadWrite {
					ui.LblReadWrite.SetText("RW")
				} else {
					ui.LblReadWrite.SetText("RO")
				}

				// Get funcs for current file and populate the TblOutline
				if count%20 == 0 {
					ui.TblOutline.Clear()
					var funcs = GetFuncs(CurrentFile.Buffer.String(), CurrentFile.Buffer.Settings["filetype"].(string))
					sort.Slice(funcs, func(i, j int) bool {
						a := funcs[i]
						b := funcs[j]
						return strings.ToUpper(a.name) < strings.ToUpper(b.name)
					})

					for i, f := range funcs {
						ui.TblOutline.SetCell(i+1, 0, tview.NewTableCell(strconv.Itoa(f.line)).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignRight))
						ui.TblOutline.SetCell(i+1, 1, tview.NewTableCell(f.name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
					}
					if !onlyOnce {
						ui.TblOutline.ScrollToBeginning()
						onlyOnce = true
					}
				}
				// Original text file status updates
				if CurrentFile.Buffer.Modified() {
					ui.LblDirty.SetText("*modified*")
				} else {
					ui.LblDirty.SetText("")
				}
				ui.EdtMain.SetTitle(fmt.Sprintf("[ %s %s %s ]", CurrentFile.Encoding, CurrentFile.Buffer.Settings["filetype"].(string), CurrentFile.Buffer.Settings["fileformat"].(string)))
				ui.LblScreen.SetText(CurrentFile.Encoding)
				if CurrentFile.ReadWrite {
					ui.LblReadWrite.SetText("RW")
				} else {
					ui.LblReadWrite.SetText("RO")
				}
				ui.LblSize.SetText(utils.HumanFileSize(float64(CurrentFile.Buffer.Len())))
				ui.LblPercent.SetText(fmt.Sprintf("%d%%", int((float32(CurrentFile.Buffer.Cursor.Y)/float32(CurrentFile.Buffer.NumLines))*100.0)))
			}
			CurrentFile = UpdateGITInfos(CurrentFile)
			ui.LblGITBranch.SetText("⎇  " + CurrentFile.GitBranch)
			ui.LblCommit.SetText("⟟ " + CurrentFile.GitCommit)
			ui.LblGITStatus.SetText("🗨  " + CurrentFile.GitStatus)
		})
	}
}

// ****************************************************************************
// UpdateGITInfos()
// ****************************************************************************
func UpdateGITInfos(f editfile) editfile {
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
// SwitchOpenFile()
// ****************************************************************************
func SwitchOpenFile(fName string) {
	for _, e := range OpenFiles {
		if e.FName == fName {
			CurrentFile.FName = e.FName
			CurrentFile.Buffer = e.Buffer
			CurrentFile.Encoding = e.Encoding
			CurrentFile.GitCommit = e.GitCommit
			CurrentFile.GitStatus = e.GitStatus
			CurrentFile.GitBranch = e.GitBranch
			CurrentFile.ReadWrite = e.ReadWrite
			CurrentFile.Follow = e.Follow
			CurrentFile.IsBinary = e.IsBinary
			if !CurrentFile.IsBinary {
				CurrentView = ui.EdtMain
				ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
				ui.PgsEditorContent.SwitchToPage("textEditor")
				ui.App.SetFocus(CurrentView)

				// Configure FrmFind for text files
				ui.TxtReplace.SetDisabled(!ui.ChkToggleReplace.IsChecked())
				ui.ChkToggleReplace.SetDisabled(false)
				ui.FrmFind.GetButton(2).SetDisabled(!ui.ChkToggleReplace.IsChecked()) // Replace button
				ui.FrmFind.GetButton(3).SetDisabled(!ui.ChkToggleReplace.IsChecked()) // All button
				ui.DpdSearchType.SetDisabled(true)
				ui.DpdSearchType.SetCurrentOption(0) // Default to ASCII
				ui.ChkCase.SetDisabled(false)
			} else {
				CurrentView = ui.HexView
				CurrentFile.HexContentDirty = true // Mark for refresh
				displayBinaryContent()
				ui.PgsEditorContent.SwitchToPage("hexViewer")
				ui.App.SetFocus(CurrentView)
				ui.DisplayExifInfo(CurrentFile.FName) // Display EXIF info for binary files

				// Configure FrmFind for binary files
				ui.ConfigureFindFormForBinary(true)
			}

			// FocusOnPath(fName)
			ui.SetStatus(fmt.Sprintf("Switching to %s", CurrentFile.FName))
			for idx, file := range OpenFiles {
				if file.FName == CurrentFile.FName {
					ui.TblOpenFiles.Select(idx, 0)
					break
				}
			}
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
	SwitchOpenFile(fName.(string))
}

// ****************************************************************************
// SwitchPreviousFile()
// ****************************************************************************
func SwitchPreviousFile() {
	for i, e := range OpenFiles {
		if e.FName == CurrentFile.FName {
			prev := i - 1
			if prev < 0 {
				prev = len(OpenFiles) - 1
			}
			SwitchOpenFile(OpenFiles[prev].FName)
			break
		}
	}
}

// ****************************************************************************
// SwitchNextFile()
// ****************************************************************************
func SwitchNextFile() {
	for i, e := range OpenFiles {
		if e.FName == CurrentFile.FName {
			next := i + 1
			if next == len(OpenFiles) {
				next = 0
			}
			SwitchOpenFile(OpenFiles[next].FName)
			break
		}
	}
}

// ****************************************************************************
// isFileAlreadyOpen()
// ****************************************************************************
func isFileAlreadyOpen(fName string) bool {
	rc := false
	for _, e := range OpenFiles {
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
	<-time.After(200 * time.Millisecond) // must be greater than the updateStatus sleep
	for idx := 0; idx < ui.TblOpenFiles.GetRowCount(); idx++ {
		if fName == ui.TblOpenFiles.GetCell(idx, 3).Text {
			ui.TblOpenFiles.Select(idx, 0)
			break
		}
	}
}

// ****************************************************************************
// GetGlobalDirtyFlag()
// ****************************************************************************
func GetGlobalDirtyFlag() bool {
	rc := false
	for _, f := range OpenFiles {
		if f.Buffer.Modified() {
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
	DlgSaveFile = DlgSaveFile.YesNoCancel(fmt.Sprintf("Save File %s", OpenFiles[idx].FName), // Title
		"This file has been modified. Do you want to save it ?", // Message
		confirmSave,
		idx,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgSaveFile", DlgSaveFile.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSaveFile")
}

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

// ****************************************************************************
// confirmSave()
// ****************************************************************************
func confirmSave(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_YES {
		err := ioutil.WriteFile(OpenFiles[idx].FName, []byte(OpenFiles[idx].Buffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", OpenFiles[idx].FName))
			OpenFiles[idx].Buffer.IsModified = false
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
		OpenFiles[idx].Buffer.IsModified = false
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
		err := ioutil.WriteFile(newName, []byte(CurrentFile.Buffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", CurrentFile.FName))
			CurrentFile.Buffer.IsModified = false
			if currentFlow == FLOW_CLOSE {
				CloseCurrentFile()
			} else {
				var n = -1
				for i, f := range OpenFiles {
					if f.FName == CurrentFile.FName {
						n = i
						break
					}
				}
				copy(OpenFiles[n:], OpenFiles[n+1:])
				OpenFiles = OpenFiles[:len(OpenFiles)-1]
				OpenFile(newName, true)
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
	if rc == dialog.BUTTON_CANCEL {
		if currentFlow == FLOW_CLOSE {
			OpenFiles[idx].Buffer.IsModified = false
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
	for ; quitFlowIndex < len(OpenFiles); quitFlowIndex++ {
		f := OpenFiles[quitFlowIndex]
		if f.Buffer.Modified() {
			ui.SetStatus(fmt.Sprintf("File %s is modified", f.FName))
			proposeToSaveFile(quitFlowIndex, FLOW_QUIT)
			return // Wait for user input from dialog
		}
		// Delete empty files
		if conf.ConfigGeneral.CleanUpOnExit {
			if f.Buffer.Len() == 0 {
				ui.SetStatus(fmt.Sprintf("Deleting empty file %s", f.FName))
				err := os.Remove(f.FName)
				if err != nil {
					ui.SetStatus(fmt.Sprintf("Error when deleting empty file %s : %s", f.FName, err.Error()))
				}
			}
		}
	}
	// All files checked, proceed to quit
	ui.App.Stop()
}

// ****************************************************************************
// CloseAll()
// ****************************************************************************
func CloseAll() {
	of := len(OpenFiles)
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
	for i, f := range OpenFiles {
		if f.FName == CurrentFile.FName {
			n = i
			d = filepath.Dir(f.FName)
			break
		}
	}
	if n >= 0 {
		onlyOnce = false
		ui.SetStatus("Closing file " + CurrentFile.FName)
		if CurrentFile.Buffer.IsModified {
			proposeToSaveFile(n, FLOW_CLOSE)
		} else {
			copy(OpenFiles[n:], OpenFiles[n+1:])
			OpenFiles = OpenFiles[:len(OpenFiles)-1]
			ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Files (%d)", len(OpenFiles)))
			if n > 0 {
				CurrentFile = OpenFiles[n-1]
				SwitchOpenFile(CurrentFile.FName)
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
	CurrentFile.ReadWrite = !CurrentFile.ReadWrite
	ui.SetStatus(fmt.Sprintf("Read Only attribute is set to %t", !CurrentFile.ReadWrite))
}

// ****************************************************************************
// SwitchFollow()
// ****************************************************************************
func SwitchFollow(f any) {
	CurrentFile.Follow = !CurrentFile.Follow
	ui.SetStatus(fmt.Sprintf("Follow mode is set to %t", CurrentFile.Follow))
	if CurrentFile.Follow {
		CurrentFile.RWBackup = CurrentFile.ReadWrite
		CurrentFile.ReadWrite = false
	} else {
		ui.SetStatus("Restoring buffer")
		content, err := ioutil.ReadFile(CurrentFile.FName)
		if err != nil {
			ui.SetStatus(fmt.Sprintf("Could not read %v", CurrentFile.FName))
			ui.SetStatus(fmt.Sprintf("%v", err))
		} else {
			CurrentFile.Buffer = femto.NewBufferFromString(string(content), CurrentFile.FName)
			CurrentFile.Buffer.Cursor.Y = CurrentFile.Buffer.End().Y
			ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
		}
	}
}

// ****************************************************************************
// GoBottom()
// ****************************************************************************
func GoBottom() {
	var loc femto.Loc
	loc.X = 0
	loc.Y = CurrentFile.Buffer.End().Y
	CurrentFile.Buffer.Cursor.GotoLoc(loc)
	ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
	ui.SetStatus("Go to bottom")
}

// ****************************************************************************
// GoTop()
// ****************************************************************************
func GoTop() {
	var loc femto.Loc
	loc.X = 0
	loc.Y = 0
	CurrentFile.Buffer.Cursor.GotoLoc(loc)
	ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
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
		if l <= CurrentFile.Buffer.LinesNum() {
			var loc femto.Loc
			loc.X = 0
			loc.Y = l - 1
			CurrentFile.Buffer.Cursor.GotoLoc(loc)
			ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
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
					OpenFile(path, true)
					ui.SetStatus(fmt.Sprintf("Opening %s", path))
				} else {
					ui.SetStatus(fmt.Sprintf("%s is not a text file", path))
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
	for i := fromLine - 1; i < CurrentFile.Buffer.NumLines; i++ {
		lineContent := CurrentFile.Buffer.Line(i)
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
	iFounds = 0
	currentFoundIndex = 0
	Founds = nil
	fromL := 1
	fromC := 1
	previousWhat = s
	previousWhere = CurrentFile.Buffer
	previousCase = caseInsensitive
	findSession = true
	foundSomething := true
	AFind = append(AFind, s)
	for foundSomething == true {
		l, c := findStringInLines(s, fromL, fromC, caseInsensitive)
		if l != 0 && c != 0 {
			Founds = append(Founds, found{s, l, c})
			fromL = l
			fromC = c
			iFounds++
		} else {
			foundSomething = false
		}
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
	CurrentFile.HexContentDirty = false
}

// ****************************************************************************
// findStringInHexContent()
// ****************************************************************************
func findStringInHexContent(s string, searchType string, fromOffset int) int {
	ui.SetStatus(fmt.Sprintf("findStringInHexContent called. Search string: '%s', Search type: '%s', From offset: %d", s, searchType, fromOffset))
	if fromOffset >= len(CurrentFile.ContentBytes) {
		ui.SetStatus("findStringInHexContent: From offset is beyond content length.")
		return -1
	}

	contentToSearch := CurrentFile.ContentBytes[fromOffset:]

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
	if CurrentFile.IsBinary {
		findHexNext()
	} else {
		findTextNext()
	}
}

// ****************************************************************************
// FindPrevious()
// ****************************************************************************
func FindPrevious() {
	if CurrentFile.IsBinary {
		findHexPrevious()
	} else {
		findTextPrevious()
	}
}

// ****************************************************************************
// findTextNext()
// ****************************************************************************
func findTextNext() {
	whatToFind = ui.TxtFind.GetText()
	whereToFind = CurrentFile.Buffer
	caseToFind = !ui.ChkCase.IsChecked()
	if whatToFind != previousWhat || whereToFind != previousWhere || caseToFind != previousCase || whereToFind.Modified() {
		go startFindSession(whatToFind, caseToFind)
		time.Sleep(300 * time.Millisecond)
	}
	if iFounds > 0 {
		if currentFoundIndex <= iFounds-1 {
			l := Founds[currentFoundIndex].l
			c := Founds[currentFoundIndex].c
			ui.SetStatus("Found at Line " + strconv.Itoa(l) + ", Col " + strconv.Itoa(c))
			var loc femto.Loc
			loc.X = c - 1
			loc.Y = l - 1
			CurrentFile.Buffer.Cursor.GotoLoc(loc)
			ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
			currentFoundIndex++
			ui.FrmFind.SetTitle(fmt.Sprintf("Find & Replace (%d/%d)", currentFoundIndex, iFounds))
		} else {
			ui.SetStatus("No more found")
			currentFoundIndex = 0
		}
	}
}

// ****************************************************************************
// findTextPrevious()
// ****************************************************************************
func findTextPrevious() {
	whatToFind = ui.TxtFind.GetText()
	whereToFind = CurrentFile.Buffer
	caseToFind = !ui.ChkCase.IsChecked()
	if whatToFind != previousWhat || whereToFind != previousWhere || caseToFind != previousCase || whereToFind.Modified() {
		go startFindSession(whatToFind, caseToFind)
		time.Sleep(300 * time.Millisecond)
	}
	if iFounds > 0 {
		if currentFoundIndex > 0 {
			l := Founds[currentFoundIndex].l
			c := Founds[currentFoundIndex].c
			ui.SetStatus("Found at Line " + strconv.Itoa(l) + ", Col " + strconv.Itoa(c))
			var loc femto.Loc
			loc.X = c - 1
			loc.Y = l - 1
			CurrentFile.Buffer.Cursor.GotoLoc(loc)
			ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
			currentFoundIndex--
			ui.FrmFind.SetTitle(fmt.Sprintf("Find & Replace (%d/%d)", currentFoundIndex, iFounds))
		} else {
			ui.SetStatus("No more found")
			currentFoundIndex = iFounds - 1
		}
	}
}

// ****************************************************************************
// findHexNext()
// ****************************************************************************
func findHexNext() {
	whatToFind := ui.TxtFind.GetText()
	_, searchType := ui.DpdSearchType.GetCurrentOption()
	ui.SetStatus(fmt.Sprintf("findHexNext called. Search string: '%s', Search type: '%s'", whatToFind, searchType))

	if whatToFind != previousHexWhat || searchType != previousHexSearchType || CurrentFile.HexContentDirty {
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

	if whatToFind != previousHexWhat || searchType != previousHexSearchType || CurrentFile.HexContentDirty {
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
	if CurrentFile.IsBinary {
		ui.SetStatus("Cannot replace in binary files (read-only)")
		return
	}
	// If no search has been performed or nothing was found, try to find the next occurrence first.
	if iFounds == 0 || currentFoundIndex == 0 {
		FindNext()
		// After FindNext, if still nothing found, return
		if iFounds == 0 {
			ui.SetStatus("Nothing found to replace")
			return
		}
	}

	if currentFoundIndex > 0 && currentFoundIndex <= iFounds {
		// Get the location of the current found item
		foundItem := Founds[currentFoundIndex-1]
		replaceText := ui.TxtReplace.GetText()

		// Perform the replacement in the buffer
		// Note: This is a simplified replacement. For more complex scenarios,
		// you might need to adjust cursor position and handle line changes.
		start := femto.Loc{X: foundItem.c - 2, Y: foundItem.l - 1}
		end := femto.Loc{X: start.X + len(foundItem.s), Y: start.Y}
		CurrentFile.Buffer.Replace(start, end, replaceText)

		// After replacing, automatically find the next occurrence
		FindNext()
	} else {
		ui.SetStatus("Nothing selected to replace")
	}
}

// ****************************************************************************
// ReplaceAll()
// ****************************************************************************
func ReplaceAll() {
	if CurrentFile.IsBinary {
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

	bufferContent := CurrentFile.Buffer.String()
	var newBufferContent string
	var replacements int

	if caseSensitive {
		replacements = strings.Count(bufferContent, whatToFind)
		newBufferContent = strings.Replace(bufferContent, whatToFind, replaceText, -1)
	} else {
		// Use regex for case-insensitive replacement
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
		cursor := CurrentFile.Buffer.Cursor
		CurrentFile.Buffer = femto.NewBufferFromString(newBufferContent, CurrentFile.FName)
		CurrentFile.Buffer.IsModified = true
		for i, e := range OpenFiles {
			if e.FName == CurrentFile.FName {
				OpenFiles[i].Buffer = CurrentFile.Buffer
			}
		}
		ui.EdtMain.OpenBuffer(CurrentFile.Buffer)

		if cursor.Y < CurrentFile.Buffer.NumLines {
			CurrentFile.Buffer.Cursor.Y = cursor.Y
			lineLen := len(CurrentFile.Buffer.Line(cursor.Y))
			if cursor.X < lineLen {
				CurrentFile.Buffer.Cursor.X = cursor.X
			} else {
				CurrentFile.Buffer.Cursor.X = lineLen
			}
		}

		SetTheme(conf.ConfigGeneral.Theme)
		ui.SetStatus(fmt.Sprintf("%d replacement(s) made", replacements))
		ui.App.SetFocus(ui.EdtMain)
		previousWhat = ""
	} else {
		ui.SetStatus("Nothing to replace")
	}
}

// ****************************************************************************
// ReplaceOnlyThisOne()
// ****************************************************************************
func ReplaceOnlyThisOne(l int, c int, n int, subst string) {
	array := bytes.Split([]byte(CurrentFile.Buffer.String()), []byte("\n"))
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
	CurrentFile.Buffer.Insert(CurrentFile.Buffer.Cursor.Loc, txt)
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
	hexAndAsciiStr := utils.BytesToHexAndASCII(CurrentFile.ContentBytes)

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
		totalBytes := len(CurrentFile.ContentBytes)
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
