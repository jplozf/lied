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
// fm is the File Manager module
// ****************************************************************************

import (
	"fmt"
	"io/fs"
	"lied/conf"
	"lied/dialog"
	"lied/menu"
	"lied/preview"
	"lied/ui"
	"lied/utils"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type SortColumn int

type selecao struct {
	fName string
	fSize int64
	fType string
}

const (
	SORT_NAME SortColumn = iota
	SORT_TIME
	SORT_SIZE
)

const (
	SORT_ASCENDING  = 0
	SORT_DESCENDING = 1
)

type PasteMode int

const (
	PASTE_DEFAULT PasteMode = iota
	PASTE_COPY
	PASTE_CUT
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var sortColumn = SORT_NAME
var sortOrder = SORT_ASCENDING
var (
	Hidden       bool
	MnuFiles     *menu.Menu
	MnuFilesSort *menu.Menu
	DlgConfirm   *dialog.Dialog
	sel          []selecao
)
var pasteMode = PASTE_DEFAULT
var pasteSource string
var pasteTarget string

// ****************************************************************************
// SetFilesMenu()
// ****************************************************************************
func SetFilesMenu() {
	MnuFiles = MnuFiles.New("Actions", ui.GetCurrentScreen(), ui.TblFiles)
	MnuFiles.AddItem("mnuEdit", "Edit", DoEdit, nil, true, false)
	MnuFiles.AddItem("mnuSelect", "Select / Unselect All", SelectAll, nil, true, false)
	MnuFiles.AddItem("mnuDelete", "Delete", DoDelete, nil, true, false)
	MnuFiles.AddItem("mnuRename", "Rename", DoRename, nil, true, false)
	MnuFiles.AddItem("mnuCopy", "Copy", DoCopy, nil, true, false)
	MnuFiles.AddItem("mnuCut", "Cut", DoCut, nil, true, false)
	MnuFiles.AddItem("mnuPaste", "Paste", DoPaste, nil, false, false)
	MnuFiles.AddItem("mnuCreateFile", "New File", DoNewFile, nil, true, false)
	MnuFiles.AddItem("mnuCreateFolder", "New Folder", DoNewFolder, nil, true, false)
	MnuFiles.AddItem("mnuZip", "Zip", DoZip, nil, true, false)
	MnuFiles.AddItem("mnuSnapshot", "Snapshot", DoSnapshot, nil, true, false)
	MnuFiles.AddItem("mnuShowHiddenFiles", "Show hidden files", DoSwitchHiddenFiles, nil, true, false)
	ui.PgsApp.AddPage("dlgFileAction", MnuFiles.Popup(), true, false)

	MnuFilesSort = MnuFilesSort.New("Sort by", ui.GetCurrentScreen(), ui.TblFiles)
	MnuFilesSort.AddItem("mnuSortNameA", "Name Ascending", doSortNameA, nil, false, true)
	MnuFilesSort.AddItem("mnuSortNameD", "Name Descending", doSortNameD, nil, true, false)
	MnuFilesSort.AddItem("mnuSortSizeA", "Size Ascending", doSortSizeA, nil, true, false)
	MnuFilesSort.AddItem("mnuSortSizeD", "Size Descending", doSortSizeD, nil, true, false)
	MnuFilesSort.AddItem("mnuSortTimeA", "Time Ascending", doSortTimeA, nil, true, false)
	MnuFilesSort.AddItem("mnuSortTimeD", "Time Descending", doSortTimeD, nil, true, false)

	ui.PgsApp.AddPage("dlgFileSort", MnuFilesSort.Popup(), true, false)
}

// ****************************************************************************
// ShowFilesMenu()
// ****************************************************************************
func ShowFilesMenu() {
	idx, _ := ui.TblFiles.GetSelection()
	targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)
	// fName := filepath.Join(Cwd, ui.TblFiles.GetCell(idx, 1).Text)
	if targetType == "FOLDER" {
		MnuFiles.SetEnabled("mnuEdit", false)
		MnuFiles.SetEnabled("mnuOpen", false)
		MnuFiles.SetEnabled("mnuEncrypt", false)
	}
	if targetType == "FILE" {
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		mtype, xtype := preview.DisplayFilePreview(fName)
		if mtype[:4] == "text" || strings.HasSuffix(xtype, "sqlite3") {
			MnuFiles.SetEnabled("mnuEdit", true)
		} else {
			MnuFiles.SetEnabled("mnuEdit", false)
		}
		// MnuFiles.SetEnabled("mnuOpen", true)
		// MnuFiles.SetEnabled("mnuEncrypt", true)
	}
	if Hidden {
		MnuFiles.SetLabel("mnuShowHiddenFiles", "Hide hidden files")
	} else {
		MnuFiles.SetLabel("mnuShowHiddenFiles", "Show hidden files")
	}
	ui.PgsApp.ShowPage("dlgFileAction")
}

// ****************************************************************************
// ShowMenuSort()
// ****************************************************************************
func ShowMenuSort() {
	ui.PgsApp.ShowPage("dlgFileSort")
}

// ****************************************************************************
// DoDelete()
// ****************************************************************************
func DoDelete(p any) {
	if len(sel) == 0 {
		idx, _ := ui.TblFiles.GetSelection()
		if ui.TblFiles.GetCell(idx, 3).Text != conf.LABEL_PARENT_FOLDER {
			targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)
			fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
			if targetType == "FILE" {
				DlgConfirm = DlgConfirm.YesNoCancel(fmt.Sprintf("Delete File %s", fName), // Title
					"Are you sure you want to delete this file ?", // Message
					DeleteFile,
					idx,
					ui.GetCurrentScreen(), ui.TblFiles) // Focus return
				ui.PgsApp.AddPage("dlgConfirmDeleteFile", DlgConfirm.Popup(), true, false)
				ui.PgsApp.ShowPage("dlgConfirmDeleteFile")
			} else {
				DlgConfirm = DlgConfirm.YesNoCancel(fmt.Sprintf("Delete Folder %s", fName), // Title
					"Are you sure you want to delete this folder and all its content ?", // Message
					DeleteFolder,
					idx,
					ui.GetCurrentScreen(), ui.TblFiles) // Focus return
				ui.PgsApp.AddPage("dlgConfirmDeleteFolder", DlgConfirm.Popup(), true, false)
				ui.PgsApp.ShowPage("dlgConfirmDeleteFolder")
			}
		} else {
			ui.SetStatus("Can't delete parent folder")
		}
	} else {
		DlgConfirm = DlgConfirm.YesNoCancel("Delete Selection", // Title
			"Are you sure you want to delete all of these files ?", // Message
			DeleteSelection,
			0,
			ui.GetCurrentScreen(), ui.TblFiles) // Focus return
		ui.PgsApp.AddPage("dlgConfirmDeleteSelection", DlgConfirm.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgConfirmDeleteSelection")
	}
}

// ****************************************************************************
// DeleteFile()
// ****************************************************************************
func DeleteFile(button dialog.DlgButton, idx int) {
	if button == dialog.BUTTON_YES {
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		err := os.Remove(fName)
		if err != nil {
			ui.SetStatus(err.Error())
		} else {
			ui.SetStatus("Deleting file " + fName)
			RefreshMe()
		}
	}
	if button == dialog.BUTTON_NO {
		ui.SetStatus("Aborting deletion of file " + ui.TblFiles.GetCell(idx, 2).Text)
	}
	if button == dialog.BUTTON_CANCEL {
		ui.SetStatus("Cancelling deletion of file " + ui.TblFiles.GetCell(idx, 2).Text)
	}
}

// ****************************************************************************
// DeleteFolder()
// ****************************************************************************
func DeleteFolder(button dialog.DlgButton, idx int) {
	if button == dialog.BUTTON_YES {
		ui.SetStatus("Deleting folder " + ui.TblFiles.GetCell(idx, 2).Text)
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		err := os.RemoveAll(fName)
		if err != nil {
			ui.SetStatus(err.Error())
		} else {
			ui.SetStatus("Deleting folder " + fName)
			RefreshMe()
		}
	}
	if button == dialog.BUTTON_NO {
		ui.SetStatus("Aborting deletion of folder " + ui.TblFiles.GetCell(idx, 2).Text)
	}
	if button == dialog.BUTTON_CANCEL {
		ui.SetStatus("Cancelling deletion of folder " + ui.TblFiles.GetCell(idx, 2).Text)
	}
}

// ****************************************************************************
// DeleteSelection()
// ****************************************************************************
func DeleteSelection(button dialog.DlgButton, idx int) {
	if button == dialog.BUTTON_YES {
		for _, s := range sel {
			if s.fType == "FOLDER" {
				ui.SetStatus("Deleting folder " + s.fName)
				err := os.RemoveAll(s.fName)
				if err != nil {
					ui.SetStatus(err.Error())
				} else {
					ui.SetStatus("Deleting folder " + s.fName)
				}
			} else {
				err := os.Remove(s.fName)
				if err != nil {
					ui.SetStatus(err.Error())
				} else {
					ui.SetStatus("Deleting file " + s.fName)
				}
			}
		}
		sel = nil
		RefreshMe()
	}
	if button == dialog.BUTTON_NO {
		ui.SetStatus("Aborting deletion of selection")
	}
	if button == dialog.BUTTON_CANCEL {
		ui.SetStatus("Cancelling deletion of selection")
	}
}

// ****************************************************************************
// DoRename(p any)
// ****************************************************************************
func DoRename(p any) {
	if len(sel) == 0 {
		idx, _ := ui.TblFiles.GetSelection()
		targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		if targetType == "FILE" {
			DlgConfirm = DlgConfirm.Input(fmt.Sprintf("Rename File %s", fName), // Title
				"Please, enter the new name :", // Message
				filepath.Base(fName),
				RenameFile,
				idx,
				ui.GetCurrentScreen(), ui.TblFiles) // Focus return
			ui.PgsApp.AddPage("dlgConfirmRenameFile", DlgConfirm.Popup(), true, false)
			ui.PgsApp.ShowPage("dlgConfirmRenameFile")
		} else {
			DlgConfirm = DlgConfirm.Input(fmt.Sprintf("Rename Folder %s", fName), // Title
				"Please, enter the new name :", // Message
				filepath.Base(fName),
				RenameFolder,
				idx,
				ui.GetCurrentScreen(), ui.TblFiles) // Focus return
			ui.PgsApp.AddPage("dlgConfirmRenameFolder", DlgConfirm.Popup(), true, false)
			ui.PgsApp.ShowPage("dlgConfirmRenameFolder")
		}
	} else {
		ui.SetStatus("Can't rename selection")
	}
}

// ****************************************************************************
// RenameFile()
// ****************************************************************************
func RenameFile(button dialog.DlgButton, idx int) {
	if button == dialog.BUTTON_OK {
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		fNew := filepath.Join(conf.ConfigGeneral.Workspace, DlgConfirm.Value)
		err := os.Rename(fName, fNew)
		if err != nil {
			ui.SetStatus(err.Error())
			focusOn(fName)
		} else {
			ui.SetStatus(fmt.Sprintf("Renaming file %s to %s", fName, fNew))
			RefreshMe()
			focusOn(fNew)
		}
	}
	if button == dialog.BUTTON_CANCEL {
		ui.SetStatus("Cancelling renaming of file " + ui.TblFiles.GetCell(idx, 2).Text)
	}
}

// ****************************************************************************
// RenameFolder()
// ****************************************************************************
func RenameFolder(button dialog.DlgButton, idx int) {
	if button == dialog.BUTTON_OK {
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		fNew := filepath.Join(conf.ConfigGeneral.Workspace, DlgConfirm.Value)
		err := os.Rename(fName, fNew)
		if err != nil {
			ui.SetStatus(err.Error())
			focusOn(fName)
		} else {
			ui.SetStatus(fmt.Sprintf("Renaming folder %s to %s", fName, fNew))
			RefreshMe()
			focusOn(fNew)
		}
	}
	if button == dialog.BUTTON_CANCEL {
		ui.SetStatus("Cancelling renaming of folder " + ui.TblFiles.GetCell(idx, 2).Text)
	}
}

// ****************************************************************************
// DoTimestamp(p any)
// ****************************************************************************
func DoTimestamp(p any) {
	if len(sel) == 0 {
		idx, _ := ui.TblFiles.GetSelection()
		targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		current := time.Now()
		if targetType == "FILE" {
			fNew := utils.FilenameWithoutExtension(fName) + current.Format("_20060102-150405") + filepath.Ext(fName)
			err := utils.CopyFile(fName, fNew)
			if err == nil {
				ui.SetStatus("File timestamped successfully")
				RefreshMe()
				focusOn(fNew)
			} else {
				ui.SetStatus(err.Error())
				RefreshMe()
				focusOn(fName)
			}
		} else {
			fNew := fName + current.Format("_20060102-150405")
			err := utils.CopyDir(fName, fNew)
			if err == nil {
				ui.SetStatus("Folder timestamped successfully")
				RefreshMe()
				focusOn(fNew)
			} else {
				ui.SetStatus(err.Error())
				RefreshMe()
				focusOn(fName)
			}
		}
	} else {
		ui.SetStatus("Can't timestamp a selection")
	}
}

// ****************************************************************************
// DoSnapshot(p any)
// ****************************************************************************
func DoSnapshot(p any) {
	if len(sel) == 0 {
		idx, _ := ui.TblFiles.GetSelection()
		targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		current := time.Now()
		if targetType == "FILE" {
			fNew := utils.FilenameWithoutExtension(fName) + current.Format("_20060102-150405") + filepath.Ext(fName)
			err := utils.CopyFile(fName, fNew)
			if err == nil {
				fArchive := utils.FilenameWithoutExtension(fNew) + ".zip"
				utils.ZipFile(fArchive, fNew)
				os.Remove(fNew)
				ui.SetStatus("File snapshoted successfully")
				RefreshMe()
				focusOn(fArchive)
			} else {
				ui.SetStatus(err.Error())
				RefreshMe()
				focusOn(fName)
			}
		} else {
			fNew := fName + current.Format("_20060102-150405")
			err := utils.CopyDir(fName, fNew)
			if err == nil {
				fArchive := fNew + ".zip"
				utils.ZipFolder(fArchive, fNew)
				os.RemoveAll(fNew)
				ui.SetStatus("Folder snapshoted successfully")
				RefreshMe()
				focusOn(fArchive)
			} else {
				ui.SetStatus(err.Error())
				RefreshMe()
				focusOn(fName)
			}
		}
	} else {
		ui.SetStatus("Can't snapshot a selection")
	}
}

// ****************************************************************************
// DoZip(p any)
// ****************************************************************************
func DoZip(p any) {
	if len(sel) == 0 {
		idx, _ := ui.TblFiles.GetSelection()
		targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		if targetType == "FILE" {
			fArchive := utils.FilenameWithoutExtension(fName) + ".zip"
			fArchive = utils.GetFilenameWhichDoesntExist(fArchive)
			utils.ZipFile(fArchive, fName)
			ui.SetStatus("File zipped successfully")
			RefreshMe()
			focusOn(fArchive)
		} else {
			fArchive := fName + ".zip"
			fArchive = utils.GetFilenameWhichDoesntExist(fArchive)
			utils.ZipFolder(fArchive, fName)
			ui.SetStatus("Folder zipped successfully")
			RefreshMe()
			focusOn(fArchive)
		}
	} else {
		dirTemp, err := os.MkdirTemp("", "temp")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dirTemp)
		for _, s := range sel {
			if s.fType == "FOLDER" {
				_ = utils.CopyFolderIntoFolder(s.fName, dirTemp)
			}
			if s.fType == "FILE" {
				_ = utils.CopyFileIntoFolder(s.fName, dirTemp)
			}
		}
		fArchive := sel[0].fName + ".zip"
		fArchive = utils.GetFilenameWhichDoesntExist(fArchive)
		utils.ZipFolder(fArchive, dirTemp)
		ui.SetStatus(fmt.Sprintf("Selection zipped successfully to file %s", fArchive))
		sel = nil
		RefreshMe()
		focusOn(fArchive)
	}
}

// ****************************************************************************
// DoNewFile(p any)
// ****************************************************************************
func DoNewFile(p any) {
	DlgConfirm = DlgConfirm.Input("Create New File", // Title
		"Please, enter the name for this new file :", // Message
		"new_file",
		CreateNewFile,
		0,
		ui.GetCurrentScreen(), ui.TblFiles) // Focus return
	ui.PgsApp.AddPage("dlgCreateNewFile", DlgConfirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgCreateNewFile")

}

// ****************************************************************************
// DoNewFolder(p any)
// ****************************************************************************
func DoNewFolder(p any) {
	DlgConfirm = DlgConfirm.Input("Create New Folder", // Title
		"Please, enter the name for this new folder :", // Message
		"new_folder",
		CreateNewFolder,
		0,
		ui.GetCurrentScreen(), ui.TblFiles) // Focus return
	ui.PgsApp.AddPage("dlgCreateNewFolder", DlgConfirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgCreateNewFolder")

}

// ****************************************************************************
// CreateNewFile()
// ****************************************************************************
func CreateNewFile(button dialog.DlgButton, idx int) {
	if button == dialog.BUTTON_OK {
		fNew := filepath.Join(conf.ConfigGeneral.Workspace, DlgConfirm.Value)
		if utils.IsFileExist(fNew) {
			ui.SetStatus(fmt.Sprintf("File %s already exists", fNew))
		} else {
			if f, err := os.Create(fNew); err != nil {
				ui.SetStatus(err.Error())
			} else {
				f.Close()
				ui.SetStatus(fmt.Sprintf("File %s successfully created", fNew))
				RefreshMe()
				focusOn(fNew)
			}
		}
	}
	if button == dialog.BUTTON_CANCEL {
		ui.SetStatus("Cancelling creation of file")
	}
}

// ****************************************************************************
// CreateNewFolder()
// ****************************************************************************
func CreateNewFolder(button dialog.DlgButton, idx int) {
	if button == dialog.BUTTON_OK {
		fNew := filepath.Join(conf.ConfigGeneral.Workspace, DlgConfirm.Value)
		if utils.IsFileExist(fNew) {
			ui.SetStatus(fmt.Sprintf("Folder %s already exists", fNew))
		} else {
			if err := os.Mkdir(fNew, os.ModePerm); err != nil {
				ui.SetStatus(err.Error())
			} else {
				ui.SetStatus(fmt.Sprintf("Folder %s successfully created", fNew))
				RefreshMe()
				focusOn(fNew)
			}
		}
	}
	if button == dialog.BUTTON_CANCEL {
		ui.SetStatus("Cancelling creation of folder")
	}
}

// ****************************************************************************
// DoCopy(p any)
// ****************************************************************************
func DoCopy(p any) {
	idx, _ := ui.TblFiles.GetSelection()
	if ui.TblFiles.GetCell(idx, 0).Text == "   " {
		if strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text) == "FILE" {
			// SELECT FILE
			fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
			fSize, _ := strconv.Atoi(ui.TblFiles.GetCell(idx, 6).Text)
			ui.SetStatus(fName)
			ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
			ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
			sel = append(sel, selecao{fName: fName, fSize: int64(fSize), fType: "FILE"})
		} else {
			// SELECT FOLDER
			fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
			fSize, _ := utils.DirSize(fName)
			ui.SetStatus(fName)
			ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
			ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
			sel = append(sel, selecao{fName: fName, fSize: fSize, fType: "FOLDER"})
		}
		pasteMode = PASTE_COPY
		pasteSource = conf.ConfigGeneral.Workspace
		displaySelection()
	}
}

// ****************************************************************************
// DoCut(p any)
// ****************************************************************************
func DoCut(p any) {
	idx, _ := ui.TblFiles.GetSelection()
	if ui.TblFiles.GetCell(idx, 0).Text == "   " {
		if strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text) == "FILE" {
			// SELECT FILE
			fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
			fSize, _ := strconv.Atoi(ui.TblFiles.GetCell(idx, 6).Text)
			ui.SetStatus(fName)
			ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
			ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
			sel = append(sel, selecao{fName: fName, fSize: int64(fSize), fType: "FILE"})
		} else {
			// SELECT FOLDER
			fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
			fSize, _ := utils.DirSize(fName)
			ui.SetStatus(fName)
			ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
			ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
			ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
			sel = append(sel, selecao{fName: fName, fSize: fSize, fType: "FOLDER"})
		}
		pasteMode = PASTE_CUT
		pasteSource = conf.ConfigGeneral.Workspace
		displaySelection()
	}
}

// ****************************************************************************
// DoPaste(p any)
// ****************************************************************************
func DoPaste(p any) {
	var fName string
	if conf.ConfigGeneral.Workspace == pasteSource {
		ui.SetStatus("Can't paste into the same folder")
	} else {
		pasteTarget = conf.ConfigGeneral.Workspace
		if pasteMode == PASTE_COPY || pasteMode == PASTE_DEFAULT {
			ui.PleaseWait()
			for _, s := range sel {
				if s.fType == "FOLDER" {
					_ = utils.CopyFolderIntoFolder(s.fName, pasteTarget)
				}
				if s.fType == "FILE" {
					_ = utils.CopyFileIntoFolder(s.fName, pasteTarget)
				}
				fName = s.fName
			}
			sel = nil
			RefreshMe()
			ui.JobsDone()
			focusOn(fName)
		}
		if pasteMode == PASTE_CUT {
			ui.PleaseWait()
			for _, s := range sel {
				if s.fType == "FOLDER" {
					err := utils.CopyFolderIntoFolder(s.fName, pasteTarget)
					if err != nil {
						os.RemoveAll(s.fName)
					} else {
						ui.SetStatus(err.Error())
					}
				}
				if s.fType == "FILE" {
					err := utils.CopyFileIntoFolder(s.fName, pasteTarget)
					if err == nil {
						err := os.Remove(s.fName)
						if err != nil {
							ui.SetStatus(err.Error())
						}
					} else {
						ui.SetStatus(err.Error())
					}
				}
				fName = s.fName
			}
			sel = nil
			RefreshMe()
			ui.JobsDone()
			focusOn(fName)
		}
	}
}

// ****************************************************************************
// DoSwitchHiddenFiles(p any)
// ****************************************************************************
func DoSwitchHiddenFiles(p any) {
	Hidden = !Hidden
	RefreshMe()
}

// ****************************************************************************
// ShowFiles()
// ****************************************************************************
func ShowFiles() {
	// ui.TxtSelection.Clear()
	files, err := os.ReadDir(conf.ConfigGeneral.Workspace)
	if err != nil {
		ui.SetStatus(err.Error())
	}

	ui.TblFiles.Clear()
	ui.TxtFileInfo.Clear()
	iStart := 0
	iFile := 0
	if conf.ConfigGeneral.Workspace != "/" {
		ui.TblFiles.SetCell(0, 0, tview.NewTableCell("   "))
		ui.TblFiles.SetCell(0, 1, tview.NewTableCell(" "))
		ui.TblFiles.SetCell(0, 2, tview.NewTableCell("..").SetTextColor(tcell.ColorYellow))
		ui.TblFiles.SetCell(0, 3, tview.NewTableCell(conf.LABEL_PARENT_FOLDER))
		ui.TblFiles.SetCell(0, 4, tview.NewTableCell(" "))
		ui.TblFiles.SetCell(0, 5, tview.NewTableCell(" "))
		ui.TblFiles.SetCell(0, 6, tview.NewTableCell(" "))
		ui.TblFiles.SetCell(0, 7, tview.NewTableCell(" "))
		iStart = 1
	}
	ui.TxtPath.SetText(conf.ConfigGeneral.Workspace)
	switch sortColumn {
	case SORT_NAME:
		if sortOrder == SORT_ASCENDING {
			SortFileNameAscend(files)
		} else {
			SortFileNameDescend(files)
		}
	case SORT_SIZE:
		if sortOrder == SORT_ASCENDING {
			SortFileSizeAscend(files)
		} else {
			SortFileSizeDescend(files)
		}
	case SORT_TIME:
		if sortOrder == SORT_ASCENDING {
			SortFileModAscend(files)
		} else {
			SortFileModDescend(files)
		}
	}
	for _, file := range files {
		if !Hidden && file.Name()[0] == '.' { // Don't want to see hidden files ?
			continue
		}
		ui.TblFiles.SetCell(iFile+iStart, 0, tview.NewTableCell("   "))
		ui.TblFiles.SetCell(iFile+iStart, 1, tview.NewTableCell(" "))
		ui.TblFiles.SetCell(iFile+iStart, 2, tview.NewTableCell(file.Name()).SetTextColor(tcell.ColorYellow))
		fi, err := file.Info()
		if err == nil {
			ui.TblFiles.SetCell(iFile+iStart, 3, tview.NewTableCell(fi.ModTime().String()[0:19]))
			if fi.IsDir() {
				ui.TblFiles.SetCell(iFile+iStart, 4, tview.NewTableCell("  FOLDER"))
				ui.TblFiles.SetCell(0, 7, tview.NewTableCell(" "))
				ui.TblFiles.GetCell(iFile+iStart, 2).SetTextColor(tcell.ColorLightGreen)
			} else {
				if fi.Mode().String()[0] == 'L' {
					ui.TblFiles.SetCell(iFile+iStart, 1, tview.NewTableCell("🔗"))
					ui.TblFiles.SetCell(iFile+iStart, 4, tview.NewTableCell("  LINK"))

					lnk, err := os.Readlink(filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(iFile+iStart, 2).Text))
					if err == nil {
						ui.TblFiles.SetCell(iFile+iStart, 7, tview.NewTableCell(lnk))
					} else {
						ui.TblFiles.SetCell(iFile+iStart, 7, tview.NewTableCell(err.Error()))
					}
				} else {
					ui.TblFiles.SetCell(iFile+iStart, 4, tview.NewTableCell("  FILE"))
					ui.TblFiles.SetCell(0, 7, tview.NewTableCell(" "))
					// Is the file executable ?
					if fi.Mode()&0111 != 0 {
						ui.TblFiles.SetCell(iFile+iStart, 1, tview.NewTableCell("⚙"))
						ui.TblFiles.GetCell(iFile+iStart, 2).SetTextColor(tcell.ColorLightYellow)
					}
				}
			}
			ui.TblFiles.SetCell(iFile+iStart, 5, tview.NewTableCell(fi.Mode().String()))
			ui.TblFiles.SetCell(iFile+iStart, 6, tview.NewTableCell(strconv.FormatInt(fi.Size(), 10)).SetAlign(tview.AlignRight))
		}
		iFile++
	}
	ui.TblFiles.Select(0, 0)
}

// ****************************************************************************
// RefreshMe()
// ****************************************************************************
func RefreshMe() {
	ShowFiles()
	applySelection()
	displaySelection()
	ui.App.SetFocus(ui.TblFiles)
}

// ****************************************************************************
// SortFileNameAscend()
// ****************************************************************************
func SortFileNameAscend(files []fs.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		return strings.ToUpper(files[i].Name()) < strings.ToUpper(files[j].Name())
	})
}

// ****************************************************************************
// SortFileNameDescend()
// ****************************************************************************
func SortFileNameDescend(files []fs.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		return strings.ToUpper(files[i].Name()) > strings.ToUpper(files[j].Name())
	})
}

// ****************************************************************************
// SortFileSizeAscend()
// ****************************************************************************
func SortFileSizeAscend(files []fs.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		ifs, _ := files[i].Info()
		jfs, _ := files[j].Info()
		return ifs.Size() < jfs.Size()
	})
}

// ****************************************************************************
// SortFileSizeDescend()
// ****************************************************************************
func SortFileSizeDescend(files []fs.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		ifs, _ := files[i].Info()
		jfs, _ := files[j].Info()
		return ifs.Size() > jfs.Size()
	})
}

// ****************************************************************************
// SortFileModAscend()
// ****************************************************************************
func SortFileModAscend(files []fs.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		ifs, _ := files[i].Info()
		jfs, _ := files[j].Info()
		return ifs.ModTime().Unix() < jfs.ModTime().Unix()
	})
}

// ****************************************************************************
// SortFileModDescend()
// ****************************************************************************
func SortFileModDescend(files []fs.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		ifs, _ := files[i].Info()
		jfs, _ := files[j].Info()
		return ifs.ModTime().Unix() > jfs.ModTime().Unix()
	})
}

// ****************************************************************************
// doSortNameA(p any)
// ****************************************************************************
func doSortNameA(p any) {
	sortColumn = SORT_NAME
	sortOrder = SORT_ASCENDING
	MnuFilesSort.SetEnabled("mnuSortNameA", false)
	MnuFilesSort.SetEnabled("mnuSortNameD", true)
	MnuFilesSort.SetEnabled("mnuSortSizeA", true)
	MnuFilesSort.SetEnabled("mnuSortSizeD", true)
	MnuFilesSort.SetEnabled("mnuSortTimeA", true)
	MnuFilesSort.SetEnabled("mnuSortTimeD", true)

	MnuFilesSort.SetChecked("mnuSortNameA", true)
	MnuFilesSort.SetChecked("mnuSortNameD", false)
	MnuFilesSort.SetChecked("mnuSortSizeA", false)
	MnuFilesSort.SetChecked("mnuSortSizeD", false)
	MnuFilesSort.SetChecked("mnuSortTimeA", false)
	MnuFilesSort.SetChecked("mnuSortTimeD", false)
	RefreshMe()
}

// ****************************************************************************
// doSortNameD(p any)
// ****************************************************************************
func doSortNameD(p any) {
	sortColumn = SORT_NAME
	sortOrder = SORT_DESCENDING
	MnuFilesSort.SetEnabled("mnuSortNameA", true)
	MnuFilesSort.SetEnabled("mnuSortNameD", false)
	MnuFilesSort.SetEnabled("mnuSortSizeA", true)
	MnuFilesSort.SetEnabled("mnuSortSizeD", true)
	MnuFilesSort.SetEnabled("mnuSortTimeA", true)
	MnuFilesSort.SetEnabled("mnuSortTimeD", true)

	MnuFilesSort.SetChecked("mnuSortNameA", false)
	MnuFilesSort.SetChecked("mnuSortNameD", true)
	MnuFilesSort.SetChecked("mnuSortSizeA", false)
	MnuFilesSort.SetChecked("mnuSortSizeD", false)
	MnuFilesSort.SetChecked("mnuSortTimeA", false)
	MnuFilesSort.SetChecked("mnuSortTimeD", false)
	RefreshMe()
}

// ****************************************************************************
// doSortSizeA(p any)
// ****************************************************************************
func doSortSizeA(p any) {
	sortColumn = SORT_SIZE
	sortOrder = SORT_ASCENDING
	MnuFilesSort.SetEnabled("mnuSortNameA", true)
	MnuFilesSort.SetEnabled("mnuSortNameD", true)
	MnuFilesSort.SetEnabled("mnuSortSizeA", false)
	MnuFilesSort.SetEnabled("mnuSortSizeD", true)
	MnuFilesSort.SetEnabled("mnuSortTimeA", true)
	MnuFilesSort.SetEnabled("mnuSortTimeD", true)

	MnuFilesSort.SetChecked("mnuSortNameA", false)
	MnuFilesSort.SetChecked("mnuSortNameD", false)
	MnuFilesSort.SetChecked("mnuSortSizeA", true)
	MnuFilesSort.SetChecked("mnuSortSizeD", false)
	MnuFilesSort.SetChecked("mnuSortTimeA", false)
	MnuFilesSort.SetChecked("mnuSortTimeD", false)
	RefreshMe()
}

// ****************************************************************************
// doSortSizeD(p any)
// ****************************************************************************
func doSortSizeD(p any) {
	sortColumn = SORT_SIZE
	sortOrder = SORT_DESCENDING
	MnuFilesSort.SetEnabled("mnuSortNameA", true)
	MnuFilesSort.SetEnabled("mnuSortNameD", true)
	MnuFilesSort.SetEnabled("mnuSortSizeA", true)
	MnuFilesSort.SetEnabled("mnuSortSizeD", false)
	MnuFilesSort.SetEnabled("mnuSortTimeA", true)
	MnuFilesSort.SetEnabled("mnuSortTimeD", true)

	MnuFilesSort.SetChecked("mnuSortNameA", false)
	MnuFilesSort.SetChecked("mnuSortNameD", false)
	MnuFilesSort.SetChecked("mnuSortSizeA", false)
	MnuFilesSort.SetChecked("mnuSortSizeD", true)
	MnuFilesSort.SetChecked("mnuSortTimeA", false)
	MnuFilesSort.SetChecked("mnuSortTimeD", false)
	RefreshMe()
}

// ****************************************************************************
// doSortTimeA(p any)
// ****************************************************************************
func doSortTimeA(p any) {
	sortColumn = SORT_TIME
	sortOrder = SORT_ASCENDING
	MnuFilesSort.SetEnabled("mnuSortNameA", true)
	MnuFilesSort.SetEnabled("mnuSortNameD", true)
	MnuFilesSort.SetEnabled("mnuSortSizeA", true)
	MnuFilesSort.SetEnabled("mnuSortSizeD", true)
	MnuFilesSort.SetEnabled("mnuSortTimeA", false)
	MnuFilesSort.SetEnabled("mnuSortTimeD", true)

	MnuFilesSort.SetChecked("mnuSortNameA", false)
	MnuFilesSort.SetChecked("mnuSortNameD", false)
	MnuFilesSort.SetChecked("mnuSortSizeA", false)
	MnuFilesSort.SetChecked("mnuSortSizeD", false)
	MnuFilesSort.SetChecked("mnuSortTimeA", true)
	MnuFilesSort.SetChecked("mnuSortTimeD", false)
	RefreshMe()
}

// ****************************************************************************
// doSortTimeD(p any)
// ****************************************************************************
func doSortTimeD(p any) {
	sortColumn = SORT_TIME
	sortOrder = SORT_DESCENDING
	MnuFilesSort.SetEnabled("mnuSortNameA", true)
	MnuFilesSort.SetEnabled("mnuSortNameD", true)
	MnuFilesSort.SetEnabled("mnuSortSizeA", true)
	MnuFilesSort.SetEnabled("mnuSortSizeD", true)
	MnuFilesSort.SetEnabled("mnuSortTimeA", true)
	MnuFilesSort.SetEnabled("mnuSortTimeD", false)

	MnuFilesSort.SetChecked("mnuSortNameA", false)
	MnuFilesSort.SetChecked("mnuSortNameD", false)
	MnuFilesSort.SetChecked("mnuSortSizeA", false)
	MnuFilesSort.SetChecked("mnuSortSizeD", false)
	MnuFilesSort.SetChecked("mnuSortTimeA", false)
	MnuFilesSort.SetChecked("mnuSortTimeD", true)
	RefreshMe()
}

// ****************************************************************************
// ProceedFileAction()
// ****************************************************************************
func ProceedFileAction() {
	ui.PleaseWait()
	idx, _ := ui.TblFiles.GetSelection()
	// TODO : manage LINK
	targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)
	if targetType == "LINK" {
		targetType = "FILE"
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		fFile, err := os.Readlink(fName)
		if err == nil {
			info, err := os.Stat(fFile)
			if err == nil {
				if info.IsDir() {
					targetType = "FOLDER"
				}
			} else {
				ui.SetStatus(err.Error())
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
	if targetType == "FILE" { // or type(readlink)==file
		fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		ui.FrmFileInfo.Clear()

		mtype, xmtype := preview.DisplayFilePreview(fName)
		size, _ := strconv.ParseFloat(ui.TblFiles.GetCell(idx, 6).Text, 64)
		if size <= conf.HASH_THRESHOLD_SIZE || true { // !!! TEST SKIPPED !!!
			infos := map[string]string{
				"00Name":          ui.TblFiles.GetCell(idx, 2).Text,
				"01Change Date":   ui.TblFiles.GetCell(idx, 3).Text,
				"02Access":        ui.TblFiles.GetCell(idx, 5).Text,
				"03Size":          ui.TblFiles.GetCell(idx, 6).Text + " Bytes (" + utils.HumanFileSize(size) + ")",
				"04Mime Type":     mtype,
				"05Extended Mime": xmtype,
			}
			ui.DisplayMap(ui.FrmFileInfo, infos)

		} else {
			infos := map[string]string{
				"00Name":        ui.TblFiles.GetCell(idx, 2).Text,
				"01Change Date": ui.TblFiles.GetCell(idx, 3).Text,
				"02Access":      ui.TblFiles.GetCell(idx, 5).Text,
				"03Size":        ui.TblFiles.GetCell(idx, 6).Text + " Bytes (" + utils.HumanFileSize(size) + ")",
			}
			ui.DisplayMap(ui.FrmFileInfo, infos)
			ui.TxtFileInfo.SetText("VERY BIG FILE, can't display a preview.")
		}
	} else { //  or type(readlink)==folder
		conf.ConfigGeneral.Workspace = filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
		ui.FrmFileInfo.Clear()
		nFiles, nFolders, err := utils.NumberOfFilesAndFolders(conf.ConfigGeneral.Workspace)
		if err != nil {
			ui.SetStatus(err.Error())
		}
		infos := map[string]string{
			"00Name":    conf.ConfigGeneral.Workspace,
			"01Files":   strconv.Itoa(nFiles),
			"02Folders": strconv.Itoa(nFolders),
		}
		ui.DisplayMap(ui.FrmFileInfo, infos)
		ShowFiles()
		ui.App.SetFocus(ui.TblFiles)
	}
	// ui.App.Sync()
	ui.JobsDone()
}

// ****************************************************************************
// ProceedFileSelect()
// ****************************************************************************
func ProceedFileSelect() {
	ui.PleaseWait()
	idx, _ := ui.TblFiles.GetSelection()
	if ui.TblFiles.GetCell(idx, 3).Text != conf.LABEL_PARENT_FOLDER {
		if strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text) == "FILE" {
			if ui.TblFiles.GetCell(idx, 0).Text == "   " {
				// SELECT FILE
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				fSize, _ := strconv.Atoi(ui.TblFiles.GetCell(idx, 6).Text)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
				ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
				sel = append(sel, selecao{fName: fName, fSize: int64(fSize), fType: "FILE"})
				displaySelection()
			} else {
				// UNSELECT FILE
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell("   "))
				if ui.TblFiles.GetCell(idx, 1).Text == "⚙" {
					ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_EXECUTABLE)
					ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_EXECUTABLE)
					ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_EXECUTABLE)
				} else {
					ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_FILE)
					ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_FILE)
					ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_FILE)
				}
				sel = findAndDelete(sel, selecao{fName: fName, fSize: 0, fType: "FILE"})
				displaySelection()
			}
		} else {
			if ui.TblFiles.GetCell(idx, 0).Text == "   " {
				// SELECT FOLDER
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				fSize, _ := utils.DirSize(fName)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
				ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
				sel = append(sel, selecao{fName: fName, fSize: fSize, fType: "FOLDER"})
				displaySelection()
			} else {
				// UNSELECT FOLDER
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell("   "))
				ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_FOLDER)
				ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_FOLDER)
				ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_FOLDER)
				sel = findAndDelete(sel, selecao{fName: fName, fSize: 0, fType: "FOLDER"})
				displaySelection()
			}
		}
		// Move cursor to next line
		if idx < ui.TblFiles.GetRowCount()-1 {
			ui.TblFiles.Select(idx+1, 0)
		}
	}
	ui.JobsDone()
}

// ****************************************************************************
// displaySelection()
// ****************************************************************************
func displaySelection() {
	if len(sel) > 0 {
		nFiles := 0
		nFolders := 0
		nSize := 0
		for _, s := range sel {
			nSize += int(s.fSize)
			if s.fType == "FILE" {
				nFiles++
			} else {
				nFolders++
			}
		}
		infos := map[string]string{
			"00Files":   fmt.Sprintf("%d", nFiles),
			"01Folders": fmt.Sprintf("%d", nFolders),
			"02Size":    fmt.Sprintf("%d Bytes (%s)", nSize, utils.HumanFileSize(float64(nSize))),
		}
		ui.DisplayMap(ui.TxtSelection, infos)
		MnuFiles.SetEnabled("mnuPaste", true)
	} else {
		ui.TxtSelection.Clear()
		MnuFiles.SetEnabled("mnuPaste", false)
		pasteMode = PASTE_DEFAULT
	}
	switch pasteMode {
	case PASTE_DEFAULT:
		ui.TxtSelection.SetTitle("Selection")
	case PASTE_COPY:
		ui.TxtSelection.SetTitle("Selection (COPY)")
	case PASTE_CUT:
		ui.TxtSelection.SetTitle("Selection (CUT)")
	}
}

// ****************************************************************************
// findAndDelete()
// ****************************************************************************
func findAndDelete(s []selecao, item selecao) []selecao {
	index := 0
	for _, i := range s {
		if i.fName != item.fName {
			s[index] = i
			index++
		}
	}
	return s[:index]
}

// ****************************************************************************
// applySelection()
// ****************************************************************************
func applySelection() {
	if len(sel) > 0 {
		for idx := 0; idx < ui.TblFiles.GetRowCount(); idx++ {
			fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
			for _, s := range sel {
				if s.fName == fName && s.fType == strings.Trim(ui.TblFiles.GetCell(idx, 4).Text, " ") {
					ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
					ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
					ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
					ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
				}
			}
		}
	}
}

// ****************************************************************************
// focusOn()
// ****************************************************************************
func focusOn(fName string) {
	for idx := 0; idx < ui.TblFiles.GetRowCount(); idx++ {
		fBase := filepath.Base(fName)
		if ui.TblFiles.GetCell(idx, 2).Text == fBase {
			ui.TblFiles.Select(idx, 0)
			break
		}
	}
}

// ****************************************************************************
// SelectAll(p any)
// ****************************************************************************
func SelectAll(p any) {
	ui.PleaseWait()
	if len(sel) == 0 {
		for idx := 1; idx < ui.TblFiles.GetRowCount(); idx++ {
			if strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text) == "FILE" {
				// SELECT FILE
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				fSize, _ := strconv.Atoi(ui.TblFiles.GetCell(idx, 6).Text)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
				ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
				sel = append(sel, selecao{fName: fName, fSize: int64(fSize), fType: "FILE"})
			} else {
				// SELECT FOLDER
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				fSize, _ := utils.DirSize(fName)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell(" ✓ "))
				ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_SELECTED)
				ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_SELECTED)
				sel = append(sel, selecao{fName: fName, fSize: fSize, fType: "FOLDER"})
			}
		}
		RefreshMe()
		displaySelection()
	} else {
		for idx := 1; idx < ui.TblFiles.GetRowCount(); idx++ {
			if strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text) == "FILE" {
				// UNSELECT FILE
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell("   "))
				if ui.TblFiles.GetCell(idx, 1).Text == "⚙" {
					ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_EXECUTABLE)
					ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_EXECUTABLE)
					ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_EXECUTABLE)
				} else {
					ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_FILE)
					ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_FILE)
					ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_FILE)
				}
				sel = findAndDelete(sel, selecao{fName: fName, fSize: 0, fType: "FILE"})
			} else {
				// UNSELECT FOLDER
				fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
				ui.SetStatus(fName)
				ui.TblFiles.SetCell(idx, 0, tview.NewTableCell("   "))
				ui.TblFiles.GetCell(idx, 0).SetTextColor(conf.COLOR_FOLDER)
				ui.TblFiles.GetCell(idx, 1).SetTextColor(conf.COLOR_FOLDER)
				ui.TblFiles.GetCell(idx, 2).SetTextColor(conf.COLOR_FOLDER)
				sel = findAndDelete(sel, selecao{fName: fName, fSize: 0, fType: "FOLDER"})
			}
		}
		sel = nil
		RefreshMe()
		displaySelection()
	}
	ui.JobsDone()
}

// ****************************************************************************
// DoEdit(p any)
// ****************************************************************************
func DoEdit(p any) {
	idx, _ := ui.TblFiles.GetSelection()
	fName := filepath.Join(conf.ConfigGeneral.Workspace, ui.TblFiles.GetCell(idx, 2).Text)
	mtype, xtype := preview.DisplayFilePreview(fName)
	if mtype[:4] == "text" {
		SwitchToEditor(fName)
	}
	if strings.HasSuffix(xtype, "sqlite3") {
		OpenDB(fName)
		SwitchToSQLite3()
	}
}

/*
// ****************************************************************************
// SelfInit()
// ****************************************************************************
func SelfInit(a any) {
	SetFilesMenu()
	ShowFiles()
	ui.App.Sync()
	ui.App.SetFocus(ui.TblFiles)
}
*/
