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
	"fmt"
	"io/ioutil"
	"lied/conf"
	"lied/dialog"
	"lied/ui"
	"lied/utils"
	"os"
	"path/filepath"
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
	Buffer   *femto.Buffer
	FName    string
	Encoding string
}

const (
	FLOW_SELF = iota
	FLOW_CLOSE
	FLOW_QUIT
	FLOW_NONE
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	OpenFiles     []editfile
	CurrentFile   editfile
	DlgSaveFile   *dialog.Dialog
	DlgSaveFileAs *dialog.Dialog
	currentFlow   int
)

// ****************************************************************************
// SwitchToEditor()
// ****************************************************************************
func SwitchToEditor(fName string) {
	ui.CurrentMode = ui.ModeTextEdit
	ui.SetTitle(conf.APP_NAME)
	ui.LblKeys.SetText(conf.FKEY_LABELS + "\nCtrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+T=Close")
	scr := ui.GetScreenFromTitle(conf.APP_NAME)
	if scr == "NIL" {
		var screen ui.MyScreen
		screen.ID, _ = utils.RandomHex(3)
		screen.Mode = ui.ModeTextEdit
		screen.Title = conf.APP_NAME
		screen.Keys = "Ctrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+T=Close"
		ui.PgsApp.AddPage(screen.Title+"_"+screen.ID, ui.FlxEditor, true, true)
		scr = screen.Title + "_" + screen.ID
		ui.ArrScreens = append(ui.ArrScreens, screen)
		ui.IdxScreens++
	}
	ui.PgsApp.SwitchToPage(scr) // ???
	// ShowTreeDir(filepath.Dir(fName))
	// ShowTreeDir("/")
	OpenFile(fName)
	ui.App.SetFocus(ui.EdtMain)
}

// ****************************************************************************
// OpenFile()
// ****************************************************************************
func OpenFile(fName string) {
	if isFileAlreadyOpen(fName) {
		SwitchOpenFile(fName)
	} else {
		var colorscheme femto.Colorscheme
		if monokai := runtime.Files.FindFile(femto.RTColorscheme, "monokai"); monokai != nil {
			if data, err := monokai.Data(); err == nil {
				colorscheme = femto.ParseColorscheme(string(data))
			}
		}
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
			ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
			ui.EdtMain.SetColorscheme(colorscheme)
			ui.EdtMain.SetTitleAlign(tview.AlignRight)
			ui.LblScreen.SetText(CurrentFile.Encoding)
			OpenFiles = append(OpenFiles, CurrentFile)
			go UpdateStatus()
			go focusOpenFile(fName)
			ui.SetStatus(fmt.Sprintf("Opening file %s", CurrentFile.FName))
			ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Files (%d)", len(OpenFiles)))
			ui.App.SetFocus(ui.EdtMain)
		}
	}
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
		SwitchToEditor(f.Name())
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
	var status string
	for {
		time.Sleep(100 * time.Millisecond)
		ui.App.QueueUpdateDraw(func() {
			// ui.TxtEditName.SetText(currentFile.FName)
			ui.TxtEditName.SetText(filepath.Dir(CurrentFile.FName) + string(os.PathSeparator) + "[yellow]" + filepath.Base(CurrentFile.FName))
			if CurrentFile.Buffer.Modified() {
				status = conf.ICON_MODIFIED
			} else {
				status = " "
			}
			x := CurrentFile.Buffer.Cursor.X + 1
			y := CurrentFile.Buffer.Cursor.Y + 1
			ui.EdtMain.SetTitle(fmt.Sprintf("[ Ln %d, Col %d %s ]", y, x, status))
			ui.TblOpenFiles.Clear()
			for i, f := range OpenFiles {
				if f.Buffer.Modified() {
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(conf.ICON_MODIFIED))
				} else {
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(" "))
				}
				ui.TblOpenFiles.SetCell(i, 1, tview.NewTableCell(filepath.Base(f.FName)))
				ui.TblOpenFiles.SetCell(i, 2, tview.NewTableCell("⯈"))
				ui.TblOpenFiles.SetCell(i, 3, tview.NewTableCell(f.FName))
			}
		})
	}
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
			ui.EdtMain.OpenBuffer(CurrentFile.Buffer)
			ui.LblScreen.SetText(CurrentFile.Encoding)
			// FocusOnPath(fName)
			ui.SetStatus(fmt.Sprintf("Switching to %s", CurrentFile.FName))
			go focusOpenFile(fName)
			break
		}
	}
}

// ****************************************************************************
// SwitchAnyFile()
// ****************************************************************************
func SwitchAnyFile(fName any) {
	SwitchOpenFile(fName.(string))
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
				OpenFile(newName)
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
// ShowTreeDir()
// ****************************************************************************
func ShowTreeDir(rootDir string) {
	root := tview.NewTreeNode(rootDir).
		SetColor(tcell.ColorYellow)
	ui.TrvExplorer.SetRoot(root).SetCurrentNode(root)

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
	addDirToNode(root, rootDir)

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
		addDirToNode(node, path)
	} else {
		// Collapse if visible, expand if collapsed.
		node.SetExpanded(!node.IsExpanded())
	}
}

// ****************************************************************************
// addDirToNode()
// ****************************************************************************
func addDirToNode(target *tview.TreeNode, path string) {
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
			if len(mtype) >= 4 {
				if mtype[:4] == "text" {
					OpenFile(path)
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
	NewFileOrLastFile(conf.Cwd)
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
