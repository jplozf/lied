// ****************************************************************************
//
//	 _____ _____ _____ _____
//	|   __|     |   __|  |  |
//	|  |  |  |  |__   |     |
//	|_____|_____|_____|__|__|
//
// ****************************************************************************
// G O S H   -   Copyright © JPL 2023
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
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type editfile struct {
	buffer *femto.Buffer
	fName  string
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
	openFiles     []editfile
	currentFile   editfile
	DlgSaveFile   *dialog.Dialog
	DlgSaveFileAs *dialog.Dialog
	currentFlow   int
)

// ****************************************************************************
// SwitchToEditor()
// ****************************************************************************
func SwitchToEditor(fName string) {
	ui.CurrentMode = ui.ModeTextEdit
	ui.SetTitle("Editor")
	ui.LblKeys.SetText(conf.FKEY_LABELS + "\nCtrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+T=Close")
	scr := ui.GetScreenFromTitle("Editor")
	if scr == "NIL" {
		var screen ui.MyScreen
		screen.ID, _ = utils.RandomHex(3)
		screen.Mode = ui.ModeTextEdit
		screen.Title = "Editor"
		screen.Keys = "Ctrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+T=Close"
		ui.PgsApp.AddPage(screen.Title+"_"+screen.ID, ui.FlxEditor, true, true)
		scr = screen.Title + "_" + screen.ID
		ui.ArrScreens = append(ui.ArrScreens, screen)
		ui.IdxScreens++
	}
	ui.PgsApp.SwitchToPage(scr) // ???
	OpenFile(fName)
	ShowTreeDir(filepath.Dir(fName))
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
			ui.SetStatus(fmt.Sprintf("Could not read %v: %v", fName, err))
		} else {
			currentFile.fName = fName
			currentFile.buffer = femto.NewBufferFromString(string(content), currentFile.fName)
			ui.EdtMain.OpenBuffer(currentFile.buffer)
			ui.EdtMain.SetColorscheme(colorscheme)
			ui.EdtMain.SetTitleAlign(tview.AlignRight)
			openFiles = append(openFiles, currentFile)
			go UpdateStatus()
			go focusOpenFile(fName)
			ui.SetStatus(fmt.Sprintf("Opening file %s", currentFile.fName))
			ui.TblOpenFiles.SetTitle(fmt.Sprintf("Open Files (%d)", len(openFiles)))
		}
	}
}

// ****************************************************************************
// SaveFile()
// ****************************************************************************
func SaveFile() {
	err := ioutil.WriteFile(currentFile.fName, []byte(currentFile.buffer.String()), 0600)
	if err == nil {
		ui.SetStatus(fmt.Sprintf("File %s successfully saved", currentFile.fName))
		currentFile.buffer.IsModified = false
	} else {
		ui.SetStatus(err.Error())
	}
}

// ****************************************************************************
// SaveFileAs()
// ****************************************************************************
func SaveFileAs() {
	currentFlow = FLOW_SELF
	DlgSaveFileAs = DlgSaveFileAs.Input("Save File as...", // Title
		"Please, enter the new name for this file :", // Message
		currentFile.fName,
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
// NewFileOrLastFile()
// ****************************************************************************
func NewFileOrLastFile(dir string) {
	if len(openFiles) > 0 {
		SwitchToEditor(currentFile.fName)
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
			ui.TxtEditName.SetText(currentFile.fName)
			if currentFile.buffer.Modified() {
				status = conf.ICON_MODIFIED
			} else {
				status = " "
			}
			x := currentFile.buffer.Cursor.X + 1
			y := currentFile.buffer.Cursor.Y + 1
			ui.EdtMain.SetTitle(fmt.Sprintf("[ Ln %d, Col %d %s ]", y, x, status))
			ui.TblOpenFiles.Clear()
			for i, f := range openFiles {
				if f.buffer.Modified() {
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(conf.ICON_MODIFIED))
				} else {
					ui.TblOpenFiles.SetCell(i, 0, tview.NewTableCell(" "))
				}
				ui.TblOpenFiles.SetCell(i, 1, tview.NewTableCell(filepath.Base(f.fName)))
				ui.TblOpenFiles.SetCell(i, 2, tview.NewTableCell("⯈"))
				ui.TblOpenFiles.SetCell(i, 3, tview.NewTableCell(f.fName))
			}
		})
	}
}

// ****************************************************************************
// SwitchOpenFile()
// ****************************************************************************
func SwitchOpenFile(fName string) {
	for _, e := range openFiles {
		if e.fName == fName {
			currentFile.fName = e.fName
			currentFile.buffer = e.buffer
			ui.EdtMain.OpenBuffer(currentFile.buffer)
			ui.SetStatus(fmt.Sprintf("Switching to %s", currentFile.fName))
			go focusOpenFile(fName)
			break
		}
	}
}

// ****************************************************************************
// isFileAlreadyOpen()
// ****************************************************************************
func isFileAlreadyOpen(fName string) bool {
	rc := false
	for _, e := range openFiles {
		if e.fName == fName {
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
	for _, f := range openFiles {
		if f.buffer.Modified() {
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
	DlgSaveFile = DlgSaveFile.YesNoCancel(fmt.Sprintf("Save File %s", openFiles[idx].fName), // Title
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
		err := ioutil.WriteFile(openFiles[idx].fName, []byte(openFiles[idx].buffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", openFiles[idx].fName))
			openFiles[idx].buffer.IsModified = false
			if currentFlow == FLOW_CLOSE {
				CloseCurrentFile()
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
	if rc == dialog.BUTTON_NO {
		openFiles[idx].buffer.IsModified = false
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
		err := ioutil.WriteFile(newName, []byte(currentFile.buffer.String()), 0600)
		if err == nil {
			ui.SetStatus(fmt.Sprintf("File %s successfully saved", currentFile.fName))
			currentFile.buffer.IsModified = false
			if currentFlow == FLOW_CLOSE {
				CloseCurrentFile()
			} else {
				var n = -1
				for i, f := range openFiles {
					if f.fName == currentFile.fName {
						n = i
						break
					}
				}
				copy(openFiles[n:], openFiles[n+1:])
				openFiles = openFiles[:len(openFiles)-1]
				OpenFile(newName)
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
	if rc == dialog.BUTTON_CANCEL {
		if currentFlow == FLOW_CLOSE {
			openFiles[idx].buffer.IsModified = false
			CloseCurrentFile()
		}
	}
	currentFlow = FLOW_NONE
}

// ****************************************************************************
// CheckOpenFilesForSaving()
// ****************************************************************************
func CheckOpenFilesForSaving() {
	for i, f := range openFiles {
		if f.buffer.Modified() {
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
	for i, f := range openFiles {
		if f.fName == currentFile.fName {
			n = i
			d = filepath.Dir(f.fName)
			break
		}
	}
	if n >= 0 {
		if currentFile.buffer.IsModified {
			proposeToSaveFile(n, FLOW_CLOSE)
		} else {
			copy(openFiles[n:], openFiles[n+1:])
			openFiles = openFiles[:len(openFiles)-1]
			if n > 0 {
				currentFile = openFiles[n-1]
				SwitchOpenFile(currentFile.fName)
			} else {
				NewFile(d)
			}
		}
	}
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

	// Add the current directory to the root node.
	add(root, rootDir)

	// If a directory was selected, open it.
	ui.TrvExplorer.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			return // Selecting the root node does nothing.
		}
		children := node.GetChildren()
		if len(children) == 0 {
			// Load and show files in this directory.
			path := reference.(string)
			add(node, path)
		} else {
			// Collapse if visible, expand if collapsed.
			node.SetExpanded(!node.IsExpanded())
		}
	})
}

// ****************************************************************************
// SelfInit()
// ****************************************************************************
func SelfInit(a any) {
	if ui.CurrentMode == ui.ModeFiles {
		idx, _ := ui.TblFiles.GetSelection()
		fName := filepath.Join(conf.Cwd, strings.TrimSpace(ui.TblFiles.GetCell(idx, 2).Text))
		mtype := utils.GetMimeType(fName)
		if len(mtype) > 3 {
			if mtype[:4] == "text" {
				SwitchToEditor(fName)
			} else {
				NewFileOrLastFile(conf.Cwd)
			}
		} else {
			NewFileOrLastFile(conf.Cwd)
		}
	} else {
		NewFileOrLastFile(conf.Cwd)
	}
}
