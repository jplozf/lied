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
	"fmt"
	"io/ioutil"
	"lied/conf"
	"lied/dialog"
	"lied/ui"
	"lied/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	OpenFiles         []editfile
	CurrentFile       editfile
	DlgSaveFile       *dialog.Dialog
	DlgSaveFileAs     *dialog.Dialog
	currentFlow       int
	showHidden        bool
	CurrentWorkspace  string
	Founds            []found
	iFounds           int
	currentFoundIndex int
	findSession       bool
	previousWhat      string
	whatToFind        string
	whereToFind       *femto.Buffer
	previousWhere     *femto.Buffer
	caseToFind        bool
	previousCase      bool
	AFind             []string
	IFind             int
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
	ui.App.SetFocus(ui.EdtMain)
}

// ****************************************************************************
// OpenWorkspace()
// ****************************************************************************
func OpenWorkspace() {
	// DO IT
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
	CurrentWorkspace = filepath.Dir(fName)
	if isFileAlreadyOpen(fName) {
		SwitchOpenFile(fName)
	} else {
		ui.EdtMain.SetRuntimeFiles(runtime.Files)
		content, err := ioutil.ReadFile(fName)
		if err != nil {
			ui.SetStatus(fmt.Sprintf("Could not read %v", fName))
			ui.SetStatus(fmt.Sprintf("%v", err))
		} else {
			// dat, _ := os.ReadFile(fName)
			detector := chardet.NewTextDetector()
			result, err := detector.DetectBest(content)
			if err == nil {
				// fmt.Printf("Detected charset is %s", result.Charset)
				// ui.LblScreen.SetText(result.Charset)
				CurrentFile.Encoding = result.Charset
			} else {
				CurrentFile.Encoding = "Unknown"
			}

			CurrentFile.FName = fName
			CurrentFile.Buffer = femto.NewBufferFromString(string(content), CurrentFile.FName)
			CurrentFile.View = femto.NewView(CurrentFile.Buffer)
			CurrentFile.ReadWrite = rw
			CurrentFile.Follow = false
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
		}
	}
	ShowTreeDir(CurrentWorkspace, showHidden)
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
			ui.TxtCurrentWorkspace.SetText(conf.ConfigGeneral.Workspace)
			// ui.TxtCurrentEditName.SetText(filepath.Dir(CurrentFile.FName) + string(os.PathSeparator) + "[yellow]" + filepath.Base(CurrentFile.FName))
			ui.TxtCurrentEditName.SetText("[yellow]" + filepath.Base(CurrentFile.FName))
			if CurrentFile.Buffer.Modified() {
				// status = conf.ICON_MODIFIED
				ui.LblDirty.SetText("*modified*")
			} else {
				// status = " "
				ui.LblDirty.SetText("")
			}
			x := CurrentFile.Buffer.Cursor.X + 1
			y := CurrentFile.Buffer.Cursor.Y + 1
			CurrentFile = UpdateGITInfos(CurrentFile)
			ui.LblGITBranch.SetText("⎇  " + CurrentFile.GitBranch)
			ui.LblCommit.SetText("⟟ " + CurrentFile.GitCommit)
			ui.LblGITStatus.SetText("🗨  " + CurrentFile.GitStatus)
			ui.EdtMain.SetTitle(fmt.Sprintf("[ %s %s %s ]", CurrentFile.Encoding, CurrentFile.Buffer.Settings["filetype"].(string), CurrentFile.Buffer.Settings["fileformat"].(string)))
			ui.LblCursor.SetText(fmt.Sprintf("Ln %d, Col %d", y, x))
			if CurrentFile.ReadWrite {
				ui.LblReadWrite.SetText("RW")
			} else {
				ui.LblReadWrite.SetText("RO")
			}
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
			ui.LblPercent.SetText(fmt.Sprintf("%d%%", int((float32(CurrentFile.Buffer.Cursor.Y)/float32(CurrentFile.Buffer.NumLines))*100.0)))
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
		commit = "No GIT"
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
	CurrentWorkspace = filepath.Dir(fName)
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
			ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
			CurrentWorkspace = filepath.Dir(CurrentFile.FName)
			// FocusOnPath(fName)
			ui.SetStatus(fmt.Sprintf("Switching to %s", CurrentFile.FName))
			go focusOpenFile(fName)
			break
		}
	}
	ShowTreeDir(CurrentWorkspace, showHidden)
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
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
	if rc == dialog.BUTTON_NO {
		OpenFiles[idx].Buffer.IsModified = false
		if currentFlow == FLOW_CLOSE {
			CloseCurrentFile()
		}
	}
	currentFlow = FLOW_NONE
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
	for i, f := range OpenFiles {
		if f.Buffer.Modified() {
			ui.SetStatus(fmt.Sprintf("File %s is modified", f.FName))
			proposeToSaveFile(i, FLOW_QUIT)
			break
		}
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
		ui.SetStatus("Closing file " + CurrentFile.FName)
		if CurrentFile.Buffer.IsModified {
			proposeToSaveFile(n, FLOW_CLOSE)
		} else {
			copy(OpenFiles[n:], OpenFiles[n+1:])
			OpenFiles = OpenFiles[:len(OpenFiles)-1]
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
		CurrentFile.ReadWrite = CurrentFile.RWBackup
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
						OpenFile(path)
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

	for i := fromLine; i < CurrentFile.Buffer.NumLines; i++ {
		if caseInsensitive {
			if strings.Contains(strings.ToLower(CurrentFile.Buffer.Line(i)), strings.ToLower(s)) {
				idx := strings.Index(strings.ToLower(CurrentFile.Buffer.Line(i)), strings.ToLower(s))
				if fromLine == i {
					if fromColumn > idx {
						foundColumn = idx
						foundLine = i
						break
					} else {
						continue
					}
				} else {
					foundColumn = idx
					foundLine = i
					break
				}
			}
		} else {
			if strings.Contains(CurrentFile.Buffer.Line(i), s) {
				idx := strings.Index(CurrentFile.Buffer.Line(i), s)
				if fromLine == i {
					if fromColumn > idx {
						foundColumn = idx
						foundLine = i
						break
					} else {
						continue
					}
				} else {
					foundColumn = idx
					foundLine = i
					break
				}
			}
		}
	}

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
// FindNext()
// ****************************************************************************
func FindNext() {
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
	} else {
		ui.FrmFind.SetTitle("Find & Replace")
		ui.SetStatus("Nothing found")
	}
}

// ****************************************************************************
// FindPrevious()
// ****************************************************************************
func FindPrevious() {
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
	} else {
		ui.FrmFind.SetTitle("Find & Replace")
		ui.SetStatus("Nothing found")
	}
}

// ****************************************************************************
// ReplaceOne()
// ****************************************************************************
func ReplaceOne() {
	whatToFind = ui.TxtFind.GetText()
	whereToFind = CurrentFile.Buffer
	caseToFind = !ui.ChkCase.IsChecked()
	if whatToFind != previousWhat || whereToFind != previousWhere || caseToFind != previousCase || whereToFind.Modified() {
		go startFindSession(whatToFind, caseToFind)
		time.Sleep(300 * time.Millisecond)
	}
	if iFounds > 0 {
		// replacing
	} else {
		ui.FrmFind.SetTitle("Find & Replace")
		ui.SetStatus("Nothing to replace")
	}
}

// ****************************************************************************
// ReplaceAll()
// ****************************************************************************
func ReplaceAll() {
	whatToFind = ui.TxtFind.GetText()
	whereToFind = CurrentFile.Buffer
	caseToFind = !ui.ChkCase.IsChecked()
	replaceText := ui.TxtReplace.GetText()
	ui.SetStatus(fmt.Sprintf("Replacing [%s] by [%s]", whatToFind, replaceText))
	if whatToFind != previousWhat || whereToFind != previousWhere || caseToFind != previousCase || whereToFind.Modified() {
		go startFindSession(whatToFind, caseToFind)
		time.Sleep(300 * time.Millisecond)
	}
	if iFounds > 0 {
		b := CurrentFile.Buffer.String()
		b = strings.Replace(b, whatToFind, replaceText, -1)
		CurrentFile.Buffer = femto.NewBufferFromString(b, CurrentFile.FName)
		CurrentFile.Buffer.IsModified = true
		for i, e := range OpenFiles {
			if e.FName == CurrentFile.FName {
				OpenFiles[i].Buffer = CurrentFile.Buffer
			}
		}
		ui.EdtMain.Buf = CurrentFile.Buffer
		SetTheme(conf.ConfigGeneral.Theme)
		ui.SetStatus(fmt.Sprintf("%d replacement(s) made", iFounds))
		ui.App.SetFocus(ui.EdtMain)
	} else {
		ui.FrmFind.SetTitle("Find & Replace")
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
