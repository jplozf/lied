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
package main

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"lied/conf"
	"lied/dialog"
	"lied/edit"
	"lied/help"
	"lied/menu"
	"lied/sysinfo"
	"lied/ui"
	"lied/utils"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
	"github.com/pgavlin/femto"
	"github.com/rivo/tview"
	"gopkg.in/ini.v1"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type createFunc func(string, string) bool

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	appDir              string
	hostname            string
	greeting            string
	err                 error
	MnuConfig           *menu.Menu
	MnuWorkspace        *menu.Menu
	MnuLicenses         *menu.Menu
	MnuTemplates        *menu.Menu
	args                []string
	MnuInputTheme       *menu.Menu
	DlgInputGitUser     *dialog.Dialog
	DlgInputGitPassword *dialog.Dialog
	DlgInputGitEmail    *dialog.Dialog
	DlgInputFormatTime  *dialog.Dialog
	DlgInputFormatDate  *dialog.Dialog
	DlgInputFileOpen    *dialog.Dialog
	DlgInputFileDelete  *dialog.Dialog
	DlgInputShell       *dialog.Dialog
	DlgInputColorAccent *dialog.Dialog
	DlgInput            *dialog.Dialog
	DlgYesNo            *dialog.Dialog
	DlgYesNo1           *dialog.Dialog
	DlgYesNo2           *dialog.Dialog
	DlgRename           *dialog.Dialog
	DlgNewFile          *dialog.Dialog
	DlgNewFolder        *dialog.Dialog
	DlgNewDatabase      *dialog.Dialog
	ACmd                []string
	ICmd                int
	MsgBox              *dialog.Dialog
	activeCmd           *cmd.Cmd
	promptVisible       bool
	LocalClipboard      string
)

// ****************************************************************************
// init()
// ****************************************************************************
func init() {
	args = os.Args
	ui.SessionID, _ = utils.RandomHex(3)
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	user, err := user.Current()
	if err != nil {
		log.Fatalf(err.Error())
	}
	greeting = fmt.Sprintf("%s@%s⯈", user.Username, hostname)

	ui.App = tview.NewApplication()
	ui.SetUI(appQuit, greeting)

	ui.PgsApp.AddPage("edit", ui.FlxEditor, true, true)
	ui.PgsApp.AddPage("fileManager", ui.FlxFileManager, true, false)
	ui.PgsApp.AddPage("dlgQuit", ui.DlgQuit, false, false)

	userDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	// Set the Current Working Directory
	conf.ConfigGeneral.Workspace, _ = os.Getwd()

	// Define application folder
	appDir = filepath.Join(userDir, conf.APP_FOLDER)
	if _, err := os.Stat(appDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(appDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}

	conf.LogFile, err = os.OpenFile(filepath.Join(appDir, conf.FILE_LOG), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	promptVisible = false
	ui.SetStatus(fmt.Sprintf("Starting session #%s", ui.SessionID))
	Macros = make(map[string]string)
	readSettings()
	ui.SetColorAccent(conf.ConfigGeneral.ColorAccent)
}

// ****************************************************************************
// main()
// ****************************************************************************
func main() {
	// Main keyboard's events manager
	ui.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// ALT+M
		evkMacros := tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModAlt)
		if event.Key() == evkMacros.Key() && event.Rune() == evkMacros.Rune() && event.Modifiers() == evkMacros.Modifiers() {
			ShowMacrosMenu()
			return nil
		}
		// ALT+S
		evkSaveAs := tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt)
		if event.Key() == evkSaveAs.Key() && event.Rune() == evkSaveAs.Rune() && event.Modifiers() == evkSaveAs.Modifiers() {
			edit.SaveFileAs()
			return nil
		}
		// ALT+Q
		evkShell := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModAlt)
		if event.Key() == evkShell.Key() &&
			event.Rune() == evkShell.Rune() &&
			event.Modifiers() == evkShell.Modifiers() {

			shellEscape()
			return nil
		}
		switch event.Key() {
		case tcell.KeyF1:
			ShowManual()
		case tcell.KeyF8:
			ShowConfigMenu()
		case tcell.KeyF6:
			edit.SwitchPreviousFile()
		case tcell.KeyF7:
			edit.SwitchNextFile()
		case tcell.KeyF9:
			ShowContextMenu()
		case tcell.KeyF10:
			ShowMainMenu()
		case tcell.KeyF3:
			ShowGitMenu()
		case tcell.KeyF4:
			if conf.ConfigGeneral.InteractiveShell {
				if !promptVisible {
					promptVisible = true
					doInteractiveShell()
				} else {
					promptVisible = false
					ui.MidColumn.RemoveItem(ui.TxtPrompt)
					edit.CloseThisFile(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT))
					switch edit.CurrentView.Mode {
					case edit.SQLite3:
						ui.App.SetFocus(ui.TxtPromptSQL)
					case edit.Binary:
						ui.App.SetFocus(ui.HexView)
					case edit.Text:
						ui.App.SetFocus(ui.EdtMain)
					case edit.Explorer:
						ui.PgsApp.SwitchToPage("fileManager")
						ui.App.SetFocus(ui.TrvExplorer)
					}
				}
			} else {
				doDialogShell(nil)
			}
		case tcell.KeyF12, tcell.KeyCtrlQ:
			ShowQuitDialog(nil)
		case tcell.KeyCtrlC:
			LocalClipboard = edit.CurrentView.FemtoBuffer.Cursor.GetSelection()
			if LocalClipboard != "" {
				clipboard.WriteAll(LocalClipboard)
				ui.SetStatus("Copied to system clipboard")
			}
			edit.CurrentView.FemtoView.Copy()
			return nil
		case tcell.KeyCtrlX:
			if edit.CurrentView.ReadWrite == false {
				ui.SetStatus("Cannot cut from a read-only file")
				return nil
			}
			LocalClipboard = edit.CurrentView.FemtoBuffer.Cursor.GetSelection()
			if LocalClipboard != "" {
				clipboard.WriteAll(LocalClipboard)
				ui.SetStatus("Cut to system clipboard")
			}
			edit.CurrentView.FemtoView.Cut()
			edit.CurrentView.FemtoBuffer.Cursor.Relocate()
			return nil
		case tcell.KeyCtrlZ:
			edit.CurrentView.FemtoView.Undo()
			return nil
		case tcell.KeyCtrlY:
			edit.CurrentView.FemtoView.Redo()
			return nil
		case tcell.KeyCtrlA:
			edit.CurrentView.FemtoView.SelectAll()
			return nil
		case tcell.KeyCtrlV:
			if edit.CurrentView.ReadWrite == false {
				ui.SetStatus("Cannot paste into a read-only file")
				return nil
			}
			ui.SetStatus("Copying from clipboard")
			systemContent, err := clipboard.ReadAll()

			if err != nil {
				// Diagnostic message for clipboard read error (e.g., no clipboard available, xsel/xclip not installed, etc.)
				ui.SetStatus("[red]Clipboard error: " + err.Error())
				return nil
			}
			if systemContent == "" && LocalClipboard == "" {
				ui.SetStatus("[yellow]Clipboard is empty (system & local)")
				return nil
			}

			if systemContent != "" {
				cleanContent := strings.ReplaceAll(systemContent, "\r\n", "\n")
				edit.CurrentView.FemtoBuffer.Insert(edit.CurrentView.FemtoBuffer.Cursor.Loc, cleanContent)
				ui.SetStatus("Pasted from system clipboard")
			} else if LocalClipboard != "" {
				edit.CurrentView.FemtoBuffer.Insert(edit.CurrentView.FemtoBuffer.Cursor.Loc, LocalClipboard)
				ui.SetStatus("Pasted from local clipboard")
			}

			edit.CurrentView.FemtoBuffer.Cursor.Relocate()
			return nil
		case tcell.KeyCtrlL:
			if edit.CurrentView.ReadWrite {
				edit.CurrentView.FemtoView.DeleteLine()
			}
			return nil
		case tcell.KeyCtrlS:
			if edit.CurrentView.ReadWrite {
				edit.SaveFile()
			}
			return nil
		case tcell.KeyCtrlN:
			edit.NewFile(conf.ConfigGeneral.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(conf.ConfigGeneral.Workspace)
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
		case tcell.KeyCtrlE:
			DoExplorer(edit.CurrentView.FName)
			return nil
		case tcell.KeyEsc:
			if activeCmd != nil {
				// Stop the command immediately
				activeCmd.Stop()
				ui.SetStatus("Command cancelled by user.")
				activeCmd = nil // Clear the reference
				return nil      // Consume the event so it doesn't propagate
			}
		}

		return event
	})

	// Editor keyboard's events manager
	ui.EdtMain.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// ALT+S
		evkSaveAs := tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt)
		if event.Key() == evkSaveAs.Key() && event.Rune() == evkSaveAs.Rune() && event.Modifiers() == evkSaveAs.Modifiers() {
			edit.SaveFileAs()
			return nil
		}
		switch event.Key() {
		case tcell.KeyLeft:
			ui.EdtMain.CursorLeft()
			return nil // Consume event
		case tcell.KeyRight:
			ui.EdtMain.CursorRight()
			return nil // Consume event
		case tcell.KeyCtrlS:
			edit.SaveFile()
			return nil
		case tcell.KeyCtrlN:
			edit.NewFile(conf.ConfigGeneral.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(conf.ConfigGeneral.Workspace)
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
			//		case tcell.KeyEsc:
			//			ui.App.SetFocus(ui.TblOpenFiles)
			//			return nil
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblOpenFiles)
			return nil
		case tcell.KeyCtrlF:
			ui.FrmFind.GetButton(0).SetSelectedFunc(edit.FindNext)
			ui.FrmFind.GetButton(1).SetSelectedFunc(edit.FindPrevious)
			ui.FrmFind.GetButton(2).SetSelectedFunc(edit.ReplaceOne)
			ui.FrmFind.GetButton(3).SetSelectedFunc(edit.ReplaceAll)
			ui.App.SetFocus(ui.FrmFind)
			return nil
		}
		if edit.CurrentView.ReadWrite == true {
			return event
		} else {
			switch event.Key() {
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight,
				tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
				return event
			default:
				// If a key produces a character or a modification action, we cancel it (return nil)
				if event.Rune() != 0 || event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyTab {
					return nil
				}
				return nil
			}
		}
	})

	// Open Files Panel keyboard's events manager
	ui.TblOpenFiles.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			ui.App.SetFocus(ui.FrmFind)
			return nil
		case tcell.KeyEnter:
			idx, _ := ui.TblOpenFiles.GetSelection()
			fName := ui.TblOpenFiles.GetCell(idx, 3).Text
			edit.SwitchOpenView(fName)
			edit.SetFocusOnPath(fName)
			if edit.CurrentView.Mode == edit.SQLite3 {
				ui.App.SetFocus(ui.TxtPromptSQL)
			} else {
				if edit.CurrentView.Mode == edit.Binary {
					ui.App.SetFocus(ui.HexView)
				} else {
					ui.App.SetFocus(ui.EdtMain)
				}
			}
			return nil
		case tcell.KeyCtrlF:
			ui.FrmFind.GetButton(0).SetSelectedFunc(edit.FindNext)
			ui.FrmFind.GetButton(1).SetSelectedFunc(edit.FindPrevious)
			ui.FrmFind.GetButton(2).SetSelectedFunc(edit.ReplaceOne)
			ui.FrmFind.GetButton(3).SetSelectedFunc(edit.ReplaceAll)
			ui.App.SetFocus(ui.FrmFind)
			return nil
		}
		return event
	})

	// Find Panel keyboard's events manager
	ui.FrmFind.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TrvExplorer)
			return nil
		case tcell.KeyCtrlF:
			ui.App.SetFocus(ui.EdtMain)
			return nil
		}
		return event
	})

	// Text Find Field keyboard's events manager
	ui.TxtFind.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			edit.RecallFind(edit.FIND_UP)
			return nil
		case tcell.KeyDown:
			edit.RecallFind(edit.FIND_DOWN)
			return nil
		}
		return event
	})

	// Explorer Panel keyboard's events manager
	ui.TrvExplorer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblOutline)
			return nil
		case tcell.KeyCtrlF:
			ui.FrmFind.GetButton(0).SetSelectedFunc(edit.FindNext)
			ui.FrmFind.GetButton(1).SetSelectedFunc(edit.FindPrevious)
			ui.FrmFind.GetButton(2).SetSelectedFunc(edit.ReplaceOne)
			ui.FrmFind.GetButton(3).SetSelectedFunc(edit.ReplaceAll)
			ui.App.SetFocus(ui.FrmFind)
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
		}
		return event
	})

	// Outline Panel keyboard's events manager
	ui.TblOutline.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			if promptVisible {
				ui.App.SetFocus(ui.TxtPrompt)
			} else {
				if edit.CurrentView.Mode == edit.SQLite3 {
					ui.App.SetFocus(ui.TxtPromptSQL)
				} else {
					if edit.CurrentView.Mode == edit.Binary {
						ui.App.SetFocus(ui.HexView)
					} else {
						ui.App.SetFocus(ui.EdtMain)
					}
				}
			}
			return nil
		case tcell.KeyEnter:
			if edit.CurrentView.Mode != edit.Binary {
				idx, _ := ui.TblOutline.GetSelection()
				funcLine := ui.TblOutline.GetCell(idx, 0).Text
				l, _ := strconv.Atoi(funcLine)
				edit.GoLine(l)
				ui.App.SetFocus(ui.EdtMain)
			}
			return nil
		}
		return event
	})

	// HexView keyboard's events manager
	ui.HexView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentOffset, _ := ui.HexView.GetScrollOffset()
		newOffset := currentOffset
		_, _, _, height := ui.HexView.GetInnerRect()
		switch event.Key() {
		case tcell.KeyUp:
			newOffset = currentOffset - 1
			if newOffset < 0 {
				newOffset = 0
			}
			ui.HexView.ScrollTo(newOffset, 0)
		case tcell.KeyDown:
			newOffset = currentOffset + 1
			ui.HexView.ScrollTo(newOffset, 0)
		case tcell.KeyPgUp:
			newOffset = currentOffset - height
			if newOffset < 0 {
				newOffset = 0
			}
			ui.HexView.ScrollTo(newOffset, 0)
		case tcell.KeyPgDn:
			newOffset = currentOffset + height
			ui.HexView.ScrollTo(newOffset, 0)
		case tcell.KeyHome:
			newOffset = 0
			ui.HexView.ScrollToBeginning()
		case tcell.KeyEnd:
			ui.HexView.ScrollToEnd()
		case tcell.KeyCtrlN:
			edit.NewFile(conf.ConfigGeneral.Workspace)
		case tcell.KeyCtrlO:
			InputFileOpen(conf.ConfigGeneral.Workspace)
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblOpenFiles)
		case tcell.KeyCtrlF:
			ui.FrmFind.GetButton(0).SetSelectedFunc(edit.FindNext)
			ui.FrmFind.GetButton(1).SetSelectedFunc(edit.FindPrevious)
			ui.FrmFind.GetButton(2).SetDisabled(true) // Disable Replace for binary
			ui.FrmFind.GetButton(3).SetDisabled(true) // Disable Replace All for binary
			ui.App.SetFocus(ui.FrmFind)
		default:
			return event
		}
		// Update LblCursor with byte offset
		byteOffset := newOffset * 16 // 16 bytes per line
		ui.LblCursor.SetText(fmt.Sprintf("Offset: %08X", byteOffset))
		// Update LblPercent with scroll percentage
		hexContent := ui.HexView.GetText(false)
		totalLines := strings.Count(hexContent, "\n")
		if totalLines > height {
			percent := int((float64(currentOffset) / float64(totalLines-height)) * 100.0)
			if percent > 100 {
				percent = 100
			}
			ui.LblPercent.SetText(fmt.Sprintf("%d%%", percent))
		} else {
			ui.LblPercent.SetText("100%")
		}
		return nil
	})

	// Shell Prompt Field keyboard's events manager
	ui.TxtPrompt.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			Xeq(ui.TxtPrompt.GetText())
			ui.TxtPrompt.SetText("")
			return nil
		case tcell.KeyUp:
			if len(ACmd) > 0 {
				ICmd--
				if ICmd < 0 {
					ICmd = len(ACmd) - 1
				}
				ui.TxtPrompt.SetText(ACmd[ICmd])
			}
			return nil
		case tcell.KeyDown:
			if len(ACmd) > 0 {
				ICmd++
				if ICmd > len(ACmd)-1 {
					ICmd = 0
				}
				ui.TxtPrompt.SetText(ACmd[ICmd])
			}
			return nil
		case tcell.KeyF2:
			if edit.CurrentView.Mode == edit.Binary {
				ui.App.SetFocus(ui.HexView)
			} else {
				ui.App.SetFocus(ui.EdtMain)
			}
			return nil
		default:
			return event
		}
	})

	// SQL Prompt Field keyboard's events manager
	ui.TxtPromptSQL.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// ALT+ENTER and F5 Key
		evkAltEnter := tcell.NewEventKey(tcell.KeyEnter, 'z', tcell.ModAlt)
		if (event.Key() == evkAltEnter.Key() && event.Modifiers() == evkAltEnter.Modifiers()) || event.Key() == tcell.KeyF5 {
			err := edit.XeqSQL(ui.TxtPromptSQL.GetText())
			if err == nil {
				ui.TxtPromptSQL.SetText("", true)
			} else {
				ui.TxtPromptSQL.SetText(ui.TxtPromptSQL.GetText()+" => "+err.Error(), true)
			}
			return nil
		}
		switch event.Key() {
		// Key UP
		case tcell.KeyUp:
			if len(edit.ASql) > 0 {
				edit.ISql--
				if edit.ISql < 0 {
					edit.ISql = len(edit.ASql) - 1
				}
				ui.TxtPromptSQL.SetText(edit.ASql[edit.ISql], true)
			}
			return nil
		// Key DOWN
		case tcell.KeyDown:
			if len(edit.ASql) > 0 {
				edit.ISql++
				if edit.ISql > len(edit.ASql)-1 {
					edit.ISql = 0
				}
				ui.TxtPromptSQL.SetText(edit.ASql[edit.ISql], true)
			}
			return nil
		// F2 Key
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblSQLOutput)
			return nil
		default:
			return event
		}
	})

	// SQL Output Field keyboard's events manager
	ui.TblSQLOutput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblOpenFiles)
			return nil
		default:
			return event
		}
	})

	// Files panel keyboard's events manager
	ui.TblFiles.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			edit.ProceedFileAction()
			return nil
		case tcell.KeyF5:
			edit.RefreshMe()
		case tcell.KeyF8:
			edit.ShowFilesMenu()
			return nil
		case tcell.KeyCtrlS:
			edit.ShowMenuSort()
			return nil
		case tcell.KeyInsert:
			edit.ProceedFileSelect()
			return nil
		case tcell.KeyCtrlA:
			edit.SelectAll(nil)
			return nil
		case tcell.KeyCtrlC:
			edit.DoCopy(nil)
			return nil
		case tcell.KeyCtrlX:
			edit.DoCut(nil)
			return nil
		case tcell.KeyCtrlV:
			edit.DoPaste(nil)
			return nil
		case tcell.KeyDelete:
			edit.DoDelete(nil)
			return nil
		case tcell.KeyTab:
			if ui.TxtPrompt.HasFocus() {
				ui.App.SetFocus(ui.TblFiles)
			} else {
				ui.App.SetFocus(ui.TxtFileInfo)
			}
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
		}
		return event
	})

	edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)

	// * Launching lied without args : Open last workspace and last open files if any, else open a temporary file into the current directory as workspace
	// * Launching lied with directory as argument : Open a temporary file into this directory as workspace
	// * Launching lied with file name as argument : Open this file into its directory as workspace
	if len(args) > 1 {
		edit.NewFileOrLastFile(conf.ConfigGeneral.Workspace)
		fName, _ := filepath.Abs(args[1])
		if utils.IsFileExist(fName) {
			ui.SetStatus("Opening existing file " + fName)
			// edit.CloseAll()
			conf.ConfigGeneral.Workspace = filepath.Dir(fName)
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
			edit.OpenView(fName, true)
		} else {
			f, e := os.Create(fName)
			if e != nil {
				ui.SetStatus(fmt.Sprintf("Can't create '%s' file", fName))
			} else {
				f.Close()
				ui.SetStatus("Opening new file " + fName)
				// edit.CloseAll()
				conf.ConfigGeneral.Workspace = filepath.Dir(fName)
				edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
				edit.OpenView(fName, true)
			}
		}
	} else {
		edit.NewFileOrLastFile(conf.ConfigGeneral.Workspace)
	}

	conf.Version = Version
	ui.SetTitle(fmt.Sprintf("%s %s", conf.APP_NAME, Version))
	ui.SetStatus(fmt.Sprintf("Welcome on %s", conf.APP_STRING))
	ui.LblHostname.SetText("♯" + greeting)

	go ui.UpdateTime()

	// Check for new version online
	go checkNewVersion()

	if err := ui.App.SetRoot(ui.PgsApp, true).SetFocus(edit.CurrentWidget).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// ****************************************************************************
// ShowMainMenu()
// ****************************************************************************
func ShowMainMenu() {
	MnuMacros = MnuMacros.New(" "+conf.APP_NAME+" ", ui.GetCurrentScreen(), edit.CurrentWidget)
	// Dynamic options (files currently open)
	for i, e := range edit.OpenViews {
		chk := false
		if e.FName == edit.CurrentView.FName {
			chk = true
		}
		sha, _ := utils.GetSha256(e.FName)
		MnuMacros.AddItem(sha,
			fmt.Sprintf("%2d) %s", i+1, filepath.Base(e.FName)),
			edit.SwitchAnyFile,
			e.FName,
			true,
			chk)
	}
	// Fixed options
	MnuMacros.AddSeparator()
	// MnuMain.AddItem("mnuOpenWorkspace", "Open Workspace", edit.OpenWorkspace, nil, true, false)
	MnuMacros.AddItem("mnuSave", "Save", edit.SaveAnyFile, nil, edit.CurrentView.ReadWrite, false)
	MnuMacros.AddItem("mnuSaveAs", "Save as…", edit.SaveAnyFileAs, nil, true, false)
	MnuMacros.AddItem("mnuNew", "New", edit.NewAnyFile, conf.ConfigGeneral.Workspace, true, false)
	MnuMacros.AddItem("mnuOpen", "Open File…", InputFileOpen, conf.ConfigGeneral.Workspace, true, false)
	MnuMacros.AddItem("mnuClose", "Close", edit.CloseAnyFile, nil, true, false)
	MnuMacros.AddItem("mnuReadOnly", "Read Only", edit.SwitchReadWrite, nil, true, !edit.CurrentView.ReadWrite)
	MnuMacros.AddItem("mnuFollow", "Follow", edit.SwitchFollow, nil, true, edit.CurrentView.Follow)
	MnuMacros.AddSeparator()
	MnuMacros.AddItem("mnuGitAdd", "Git add…", DoGitAdd, edit.CurrentView.FName, !IsFileGitTracked(edit.CurrentView.FName), false)
	MnuMacros.AddItem("mnuArchive", "Archive", DoArchive, conf.ConfigGeneral.Workspace, true, false)
	MnuMacros.AddItem("mnuExplorer", "Explorer", DoExplorer, conf.ConfigGeneral.Workspace, true, false)
	MnuMacros.AddSeparator()
	MnuMacros.AddItem("mnuQuit", "Quit", ShowQuitDialog, nil, true, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgMainMenu", MnuMacros.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgMainMenu")
}

// ****************************************************************************
// ShowConfigMenu()
// ****************************************************************************
func ShowConfigMenu() {
	MnuConfig = MnuConfig.New(" Settings ", ui.GetCurrentScreen(), edit.CurrentWidget)
	// Menu Options
	MnuConfig.AddItem("mnuCfgTheme", "Theme", InputConfigTheme, nil, true, false)
	MnuConfig.AddItem("mnuCfgColorAccent", "Color Accent", InputColorAccent, nil, true, false)
	// These two options are now into the Git Menu :
	// MnuConfig.AddItem("mnuCfgGitUser", "Git User", InputConfigGitUser, nil, true, false)
	// MnuConfig.AddItem("mnuCfgGitPassword", "Git Password", InputConfigGitPassword, nil, true, false)
	MnuConfig.AddItem("mnuCfgConfirmExit", "Confirm Exit", SwitchConfirmExit, nil, true, conf.ConfigGeneral.ConfirmExit)
	MnuConfig.AddItem("mnuCfgCleanUpOnExit", "Clean Up on Exit", SwitchCleanUpOnExit, nil, true, conf.ConfigGeneral.CleanUpOnExit)
	MnuConfig.AddItem("mnuCfgShowHidden", "Show Hidden", SwitchShowHidden, nil, true, conf.ConfigGeneral.ShowHidden)
	MnuConfig.AddItem("mnuCfgFormatTime", "Time Format", InputConfigFormatTime, nil, true, false)
	MnuConfig.AddItem("mnuCfgFormatDate", "Date Format", InputConfigFormatDate, nil, true, false)
	MnuConfig.AddItem("mnuCfgInteractiveShell", "Interactive Shell by default", SwitchInteractiveShell, nil, true, conf.ConfigGeneral.InteractiveShell)
	MnuConfig.AddItem("mnuCfgOutErrPrefix", "Prefix OUT & ERR in Shell", SwitchPrefixShell, nil, true, conf.ConfigGeneral.OutErrPrefix)
	// Popup menu
	ui.PgsApp.AddPage("dlgConfigMenu", MnuConfig.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgConfigMenu")
}

// ****************************************************************************
// ShowWorkspaceMenu()
// ****************************************************************************
func ShowWorkspaceMenu() {
	ui.SetStatus(fmt.Sprintf("Current Workspace is %s", conf.ConfigGeneral.Workspace))
	MnuWorkspace = MnuWorkspace.New(" Workspace ", ui.GetCurrentScreen(), edit.CurrentWidget)
	// Menu Options
	MnuWorkspace.AddItem("mnuOpen", "Open Workspace", InputWorkspaceOpen, conf.ConfigGeneral.Workspace, true, false) // OK
	MnuWorkspace.AddItem("mnuSaveAll", "Save all", doSaveAll, conf.ConfigGeneral.Workspace, true, false)             // OK
	// MnuWorkspace.AddItem("mnuClose", "Close", InputWorkspaceOpen, conf.ConfigGeneral.Workspace, true, false)          // Not yet
	MnuWorkspace.AddItem("mnuRename", "Rename file or folder", InputRename, conf.ConfigGeneral.Workspace, true, false) // OK
	// MnuWorkspace.AddItem("mnuNewFile", "New file", doNewFile, conf.ConfigGeneral.Workspace, true, false)
	MnuWorkspace.AddItem("mnuAddFileTemplate", "New file", ShowTemplatesMenu, conf.ConfigGeneral.Workspace, true, false)        // OK
	MnuWorkspace.AddItem("mnuNewFolder", "New folder", doNewFolder, conf.ConfigGeneral.Workspace, true, false)                  // OK
	MnuWorkspace.AddItem("mnuNewDatabase", "New SQLite3 database", doNewDatabase, conf.ConfigGeneral.Workspace, true, false)    // OK
	MnuWorkspace.AddItem("mnuAddLicense", "Add license", ShowLicensesMenu, conf.ConfigGeneral.Workspace, true, false)           // OK
	MnuWorkspace.AddItem("mnuDelete", "Delete file or folder", InputWorkspaceDelete, conf.ConfigGeneral.Workspace, true, false) // OK
	// Popup menu
	ui.PgsApp.AddPage("dlgWorkspaceMenu", MnuWorkspace.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgWorkspaceMenu")
}

// ****************************************************************************
// ShowContextMenu()
// ****************************************************************************
func ShowContextMenu() {
	switch edit.CurrentView.Mode {
	case edit.Text:
		ShowWorkspaceMenu()
	case edit.Binary:
		ShowWorkspaceMenu()
	case edit.SQLite3:
		ShowWorkspaceMenu()
	case edit.Shell:
		ShowWorkspaceMenu()
	case edit.Explorer:
		edit.SetFilesMenu()
		edit.ShowFilesMenu()
	}
}

// ****************************************************************************
// ShowLicensesMenu()
// ****************************************************************************
func ShowLicensesMenu(f any) {
	// Read the directory entry for the "licenses" embedded folder
	entries, err := conf.LicensesFS.ReadDir("licenses")
	ui.SetStatus("Reading licences")
	if err == nil {
		MnuLicenses = MnuLicenses.New(" Licenses ", ui.GetCurrentScreen(), edit.CurrentWidget)
		for _, entry := range entries {
			if !entry.IsDir() { // Only list files, not subdirectories
				lic := entry.Name()
				MnuLicenses.AddItem(lic,
					strings.TrimSuffix(lic, filepath.Ext(lic)),
					AddLicense,
					lic,
					true,
					false)
			}
		}
		// Popup menu
		ui.PgsApp.AddPage("dlgLicensesMenu", MnuLicenses.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgLicensesMenu")
	} else {
		ui.SetStatus("No license found")
	}
}

// ****************************************************************************
// AddLicense()
// ****************************************************************************
func AddLicense(l any) {
	licenseFileName := l.(string)
	ui.SetStatus(fmt.Sprintf("Adding license %s to the current workspace", licenseFileName))
	sourceFileName := filepath.Join("licenses", licenseFileName)
	destFileName := filepath.Join(conf.ConfigGeneral.Workspace, licenseFileName)
	fileContent, err := conf.LicensesFS.ReadFile(sourceFileName)
	if err != nil {
		ui.SetStatus(fmt.Sprintf("Error reading file: %v", err))
	}
	if err := os.WriteFile(destFileName, fileContent, 0644); err != nil { // nolint: gosec
		ui.SetStatus(fmt.Sprintf("Error writing file: %w", err))
	}
	// Refresh the TrvExplorer
	edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
}

// ****************************************************************************
// ShowTemplatesMenu()
// ****************************************************************************
func ShowTemplatesMenu(f any) {
	// Read the directory entry for the "templates" embedded folder
	entries, err := conf.TemplatesFS.ReadDir("templates")
	ui.SetStatus("Reading templates")
	if err == nil {
		MnuTemplates = MnuTemplates.New(" Templates ", ui.GetCurrentScreen(), edit.CurrentWidget)
		for _, entry := range entries {
			if !entry.IsDir() { // Only list files, not subdirectories
				template := entry.Name()
				MnuTemplates.AddItem(template,
					strings.TrimSuffix(template, filepath.Ext(template)),
					AddTemplate,
					template,
					true,
					false)
			}
		}
		// Popup menu
		ui.PgsApp.AddPage("dlgTemplatesMenu", MnuTemplates.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgTemplatesMenu")
	} else {
		ui.SetStatus("No template found")
	}
}

// ****************************************************************************
// AddTemplate()
// ****************************************************************************
func AddTemplate(f any) {
	d := f.(string)
	DlgNewFile = DlgNewFile.Input("New File", // Title
		fmt.Sprintf("Creating a new file into %s", conf.ConfigGeneral.Workspace), // Message
		d, // Default file name
		func(rc dialog.DlgButton, idx int) {
			if rc == dialog.BUTTON_OK {
				CreateOrOverwriteIfItAlreadyExists(filepath.Join(conf.ConfigGeneral.Workspace, DlgNewFile.Value), filepath.Join("templates", d), func(s1, s2 string) bool {
					// Close the file if it is open
					edit.CloseThisFile(s1)
					ui.SetStatus(fmt.Sprintf("Adding template %s to the current workspace", s2))
					fileContent, err := conf.TemplatesFS.ReadFile(s2)
					if err != nil {
						ui.SetStatus(fmt.Sprintf("Error reading file: %v", err))
						return false
					}
					if err := os.WriteFile(s1, fileContent, 0644); err != nil { // nolint: gosec
						ui.SetStatus(fmt.Sprintf("Error writing file: %w", err))
						return false
					}
					// and open it
					edit.OpenView(s1, true)
					edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
					return true
				})
			} else {
				ui.SetStatus("Canceling creating new file")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgNewFile", DlgNewFile.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNewFile")
}

// ****************************************************************************
// appQuit()
// appQuit performs some cleanup and saves persistent data before quitting application
// ****************************************************************************
func appQuit() {
	// TODO : Clean up NEW_FILE_TEMPLATE_XXX null files
	edit.CheckOpenFilesForSaving()
	saveSettings()
	ui.SetStatus(fmt.Sprintf("Quitting session #%s", ui.SessionID))
	// ui.App.Stop()
	fmt.Printf("♯%s %s - %s\n", conf.APP_NAME, Version, conf.APP_URL)
	ArchiveLogs()
}

// ****************************************************************************
// readSettings()
// ****************************************************************************
func readSettings() {
	// Read MRU list and open them
	atLeastOneFile := false
	ui.SetStatus("Reading MRU list")
	fMRU, err := os.Open(filepath.Join(appDir, conf.FILE_MRU))
	if err == nil {
		defer fMRU.Close()
		sMRU := bufio.NewScanner(fMRU)
		for sMRU.Scan() {
			rec := sMRU.Text()
			rw := true
			if rec[0] == '0' {
				rw = false
			}
			edit.OpenView(rec[2:], rw)
			atLeastOneFile = true
		}
	}
	if !atLeastOneFile {
		edit.NewFile(conf.ConfigGeneral.Workspace)
	}

	// Read shell history
	ui.SetStatus("Reading shell history")
	fCmd, err := os.Open(filepath.Join(appDir, conf.FILE_SHELL_HISTORY))
	if err == nil {
		defer fCmd.Close()
		sCmd := bufio.NewScanner(fCmd)
		for sCmd.Scan() {
			ACmd = append(ACmd, sCmd.Text())
		}
	}

	// Read SQL history
	ui.SetStatus("Reading SQL history")
	fSql, err := os.Open(filepath.Join(appDir, conf.FILE_SQL_HISTORY))
	if err == nil {
		defer fSql.Close()
		sSql := bufio.NewScanner(fSql)
		for sSql.Scan() {
			edit.ASql = append(edit.ASql, sSql.Text())
		}
	}

	// Read find history
	ui.SetStatus("Reading find history")
	fFind, err := os.Open(filepath.Join(appDir, conf.FILE_FIND_HISTORY))
	if err == nil {
		defer fFind.Close()
		sFind := bufio.NewScanner(fFind)
		for sFind.Scan() {
			edit.AFind = append(edit.AFind, sFind.Text())
		}
	}

	// Read INI file
	ui.SetStatus("Reading INI file")
	inidata, err := ini.Load(filepath.Join(appDir, conf.FILE_INI))
	if err != nil {
		ui.SetStatus("No INI file found")
	} else {
		// Read them
		section := inidata.Section("general")
		conf.ConfigGeneral.Theme = section.Key("Theme").String()
		conf.ConfigGeneral.GitUser = section.Key("GitUser").String()
		conf.ConfigGeneral.GitKey = section.Key("GitKey").String()
		conf.ConfigGeneral.GitEmail = section.Key("GitEmail").String()
		conf.ConfigGeneral.Workspace = section.Key("Workspace").String()
		conf.ConfigGeneral.ShowHidden, _ = section.Key("ShowHidden").Bool()
		conf.ConfigGeneral.ConfirmExit, _ = section.Key("ConfirmExit").Bool()
		conf.ConfigGeneral.CleanUpOnExit, _ = section.Key("CleanUpOnExit").Bool()
		conf.ConfigGeneral.FormatTime = section.Key("FormatTime").String()
		conf.ConfigGeneral.FormatDate = section.Key("FormatDate").String()
		conf.ConfigGeneral.ColorAccent = section.Key("ColorAccent").String()
		conf.ConfigGeneral.InteractiveShell, _ = section.Key("InteractiveShell").Bool()
		conf.ConfigGeneral.OutErrPrefix, _ = section.Key("OutErrPrefix").Bool()
		// Set them
		if conf.ConfigGeneral.Theme == "" {
			conf.ConfigGeneral.Theme = "monokai"
		}
		setTheme(conf.ConfigGeneral.Theme)
		if conf.ConfigGeneral.FormatTime == "" {
			conf.ConfigGeneral.FormatTime = "15:04:05"
		}
		ui.MyConfig.FormatTime = conf.ConfigGeneral.FormatTime
		if conf.ConfigGeneral.FormatDate == "" {
			conf.ConfigGeneral.FormatDate = "02/01/2006"
		}
		ui.MyConfig.FormatDate = conf.ConfigGeneral.FormatDate
		if conf.ConfigGeneral.Workspace == "" {
			conf.ConfigGeneral.Workspace, _ = os.Getwd()
		}
		edit.SwitchOpenView(section.Key("CurrentFile").String())

		if edit.CurrentView.FemtoBuffer != nil {
			tmpX, _ := section.Key("CurrentX").Int()
			tmpY, _ := section.Key("CurrentY").Int()
			if edit.CurrentView.FemtoBuffer.NumLines > tmpY {
				edit.CurrentView.FemtoBuffer.Cursor.Y = tmpY
			} else {
				edit.CurrentView.FemtoBuffer.Cursor.Y = edit.CurrentView.FemtoBuffer.NumLines - 1
			}
			if len(edit.CurrentView.FemtoBuffer.Line(edit.CurrentView.FemtoBuffer.Cursor.Y)) > tmpX {
				edit.CurrentView.FemtoBuffer.Cursor.X = tmpX
			} else {
				edit.CurrentView.FemtoBuffer.Cursor.X = len(edit.CurrentView.FemtoBuffer.Line(edit.CurrentView.FemtoBuffer.Cursor.Y)) - 1
			}
			// edit.CurrentFile.Buffer.Cursor.X, _ = section.Key("CurrentX").Int()
			// edit.CurrentFile.Buffer.Cursor.Y, _ = section.Key("CurrentY").Int()
		}

	}
	if conf.ConfigGeneral.ColorAccent == "" {
		conf.ConfigGeneral.ColorAccent = conf.DEFAULT_COLOR_ACCENT
	}
	ReadMacros()
}

// ****************************************************************************
// saveSettings()
// ****************************************************************************
func saveSettings() {
	// Save MRU list
	ui.SetStatus("Saving MRU list")
	fMRU, err := os.Create(filepath.Join(appDir, conf.FILE_MRU))
	if err == nil {
		defer fMRU.Close()
		wMRU := bufio.NewWriter(fMRU)
		for _, oFile := range edit.OpenViews {
			// We record only existing files
			if utils.IsFileExist(oFile.FName) {
				rw := "0,"
				if oFile.ReadWrite {
					rw = "1,"
				}
				fmt.Fprintln(wMRU, rw+oFile.FName)
			}
		}
		wMRU.Flush()
	}

	// Save shell history
	ui.SetStatus("Saving shell history")
	fCmd, err := os.Create(filepath.Join(appDir, conf.FILE_SHELL_HISTORY))
	if err == nil {
		defer fCmd.Close()
		wCmd := bufio.NewWriter(fCmd)
		for _, line := range ACmd {
			fmt.Fprintln(wCmd, line)
		}
		wCmd.Flush()
	}

	// Save SQL history
	ui.SetStatus("Saving SQL history")
	fSql, err := os.Create(filepath.Join(appDir, conf.FILE_SQL_HISTORY))
	if err == nil {
		defer fSql.Close()
		wSql := bufio.NewWriter(fSql)
		for _, line := range edit.ASql {
			fmt.Fprintln(wSql, line)
		}
		wSql.Flush()
	}

	// Save find history
	ui.SetStatus("Saving find history")
	fFind, err := os.Create(filepath.Join(appDir, conf.FILE_FIND_HISTORY))
	if err == nil {
		defer fFind.Close()
		wFind := bufio.NewWriter(fFind)
		for _, line := range edit.AFind {
			fmt.Fprintln(wFind, line)
		}
		wFind.Flush()
	}

	// Save INI file
	inidata := ini.Empty()
	sec, _ := inidata.NewSection("general")
	sec.NewKey("Theme", conf.ConfigGeneral.Theme)
	sec.NewKey("GitUser", conf.ConfigGeneral.GitUser)
	sec.NewKey("GitKey", conf.ConfigGeneral.GitKey)
	sec.NewKey("GitEmail", conf.ConfigGeneral.GitEmail)
	sec.NewKey("Workspace", conf.ConfigGeneral.Workspace)
	sec.NewKey("ShowHidden", utils.If(conf.ConfigGeneral.ShowHidden, "True", "False"))
	sec.NewKey("ConfirmExit", utils.If(conf.ConfigGeneral.ConfirmExit, "True", "False"))
	sec.NewKey("CleanUpOnExit", utils.If(conf.ConfigGeneral.CleanUpOnExit, "True", "False"))
	sec.NewKey("FormatTime", conf.ConfigGeneral.FormatTime)
	sec.NewKey("FormatDate", conf.ConfigGeneral.FormatDate)
	sec.NewKey("CurrentFile", edit.CurrentView.FName)
	if edit.CurrentView.Mode != edit.SQLite3 && edit.CurrentView.Mode != edit.Explorer && edit.CurrentView.FemtoBuffer != nil {
		sec.NewKey("CurrentX", strconv.Itoa(edit.CurrentView.FemtoBuffer.Cursor.X))
		sec.NewKey("CurrentY", strconv.Itoa(edit.CurrentView.FemtoBuffer.Cursor.Y))
	} else {
		sec.NewKey("CurrentX", "0")
		sec.NewKey("CurrentY", "0")
	}
	sec.NewKey("ColorAccent", conf.ConfigGeneral.ColorAccent)
	sec.NewKey("InteractiveShell", utils.If(conf.ConfigGeneral.InteractiveShell, "True", "False"))
	sec.NewKey("OutErrPrefix", utils.If(conf.ConfigGeneral.OutErrPrefix, "True", "False"))

	err = inidata.SaveTo(filepath.Join(appDir, conf.FILE_INI))
	if err != nil {
		ui.SetStatus(err.Error())
	}
	SaveMacros()
}

// ****************************************************************************
// ShowQuitDialog()
// ****************************************************************************
func ShowQuitDialog(p any) {
	if conf.ConfigGeneral.ConfirmExit {
		ui.PgsApp.SwitchToPage("dlgQuit")
	} else {
		appQuit()
	}
}

// ****************************************************************************
// InputConfigTheme()
// ****************************************************************************
func InputConfigTheme(f any) {
	MnuInputTheme = MnuInputTheme.New(" Themes ", ui.GetCurrentScreen(), edit.CurrentWidget)
	arrThemes := []string{"atom-dark-tc",
		"bubblegum",
		"cmc-16",
		"cmc-paper",
		"cmc-tc",
		"darcula",
		"default",
		"geany",
		"github-tc",
		"gruvbox-tc",
		"gruvbox",
		"material-tc",
		"monokai",
		"railscast",
		"simple",
		"solarized-tc",
		"solarized",
		"twilight",
		"zenburn"}

	for _, thm := range arrThemes {
		chk := false
		if thm == conf.ConfigGeneral.Theme {
			chk = true
		}
		MnuInputTheme.AddItem(thm,
			thm,
			setTheme,
			thm,
			true,
			chk)
	}
	// Popup menu
	ui.PgsApp.AddPage("dlgThemeMenu", MnuInputTheme.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgThemeMenu")
}

// ****************************************************************************
// setTheme()
// ****************************************************************************
func setTheme(theme any) {
	edit.SetTheme(theme.(string))
	conf.ConfigGeneral.Theme = theme.(string)
	ui.SetStatus(fmt.Sprintf("Theme is set to %s", conf.ConfigGeneral.Theme))
}

// ****************************************************************************
// InputColorAccent()
// ****************************************************************************
func InputColorAccent(f any) {
	DlgInputColorAccent = DlgInputColorAccent.Input("Color Accent", // Title
		"Please, enter the color accent :", // Message
		conf.ConfigGeneral.ColorAccent,
		setColorAccent,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgInputColorAccent", DlgInputColorAccent.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputColorAccent")
}

// ****************************************************************************
// setColorAccent()
// ****************************************************************************
func setColorAccent(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		conf.ConfigGeneral.ColorAccent = DlgInputColorAccent.Value
		ui.SetColorAccent(conf.ConfigGeneral.ColorAccent)
		ui.SetStatus(fmt.Sprintf("Color accent is set to %s", conf.ConfigGeneral.ColorAccent))
	}
}

// ****************************************************************************
// InputConfigFormatTime()
// ****************************************************************************
func InputConfigFormatTime(f any) {
	DlgInputFormatTime = DlgInputFormatTime.Input("Time Format", // Title
		"Please, enter the time format :", // Message
		conf.ConfigGeneral.FormatTime,
		setFormatTime,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgInputFormatTime", DlgInputFormatTime.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFormatTime")
}

// ****************************************************************************
// setFormatTime()
// ****************************************************************************
func setFormatTime(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		conf.ConfigGeneral.FormatTime = DlgInputFormatTime.Value
		ui.SetStatus(fmt.Sprintf("Time Format is set to %s", conf.ConfigGeneral.FormatTime))
		ui.MyConfig.FormatTime = conf.ConfigGeneral.FormatTime
	}
}

// ****************************************************************************
// InputConfigFormatDate()
// ****************************************************************************
func InputConfigFormatDate(f any) {
	DlgInputFormatDate = DlgInputFormatDate.Input("Date Format", // Title
		"Please, enter the date format :", // Message
		conf.ConfigGeneral.FormatDate,
		setFormatDate,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgInputFormatDate", DlgInputFormatDate.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFormatDate")
}

// ****************************************************************************
// setFormatDate()
// ****************************************************************************
func setFormatDate(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		conf.ConfigGeneral.FormatDate = DlgInputFormatDate.Value
		ui.SetStatus(fmt.Sprintf("Date Format is set to %s", conf.ConfigGeneral.FormatDate))
		ui.MyConfig.FormatDate = conf.ConfigGeneral.FormatDate
	}
}

// ****************************************************************************
// InputFileOpen()
// ****************************************************************************
func InputFileOpen(f any) {
	startPath := conf.ConfigGeneral.Workspace
	if startPath == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			ui.SetStatus("Error getting home directory: " + err.Error())
			return
		}
		startPath = userHome
	}
	DlgInputFileOpen = DlgInputFileOpen.FileBrowser("Open File", // Title
		startPath,
		doOpenFile,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget, false) // Focus return
	ui.PgsApp.AddPage("dlgInputFileOpen", DlgInputFileOpen.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFileOpen")
	ui.App.SetFocus(DlgInputFileOpen)
}

// ****************************************************************************
// InputRename()
// ****************************************************************************
func InputRename(f any) {
	DlgInputFileOpen = DlgInputFileOpen.FileBrowser("Rename", // Title
		f.(string),
		doRename,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget, false) // Focus return
	ui.PgsApp.AddPage("dlgInputFileOpen", DlgInputFileOpen.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFileOpen")
	ui.App.SetFocus(DlgInputFileOpen)
}

// ****************************************************************************
// InputWorkspaceOpen()
// ****************************************************************************
func InputWorkspaceOpen(f any) {
	startPath := conf.ConfigGeneral.Workspace
	if startPath == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			ui.SetStatus("Error getting home directory: " + err.Error())
			return
		}
		startPath = userHome
	}
	DlgInputFileOpen = DlgInputFileOpen.FileBrowser("Open Workspace", // Title
		startPath,
		doOpenWorkspace,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget, true) // Focus return
	ui.PgsApp.AddPage("dlgInputFileOpen", DlgInputFileOpen.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFileOpen")
	ui.App.SetFocus(DlgInputFileOpen)
}

// ****************************************************************************
// InputWorkspaceDelete()
// ****************************************************************************
func InputWorkspaceDelete(f any) {
	DlgInputFileDelete = DlgInputFileDelete.DeleteFileBrowser("Delete file/folder", // Title
		f.(string),
		doDeleteFile,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget, // Focus return
		false) // Files & Folders
	ui.PgsApp.AddPage("dlgInputFileDelete", DlgInputFileDelete.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFileDelete")
	ui.App.SetFocus(DlgInputFileDelete)
}

// ****************************************************************************
// doOpenFile()
// ****************************************************************************
func doOpenFile(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		fn := DlgInputFileOpen.Value
		ui.SetStatus("Opening " + fn)
		edit.OpenView(fn, utils.IsTextFile(fn))
	}
}

// ****************************************************************************
// doRename()
// ****************************************************************************
func doRename(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		fn := DlgInputFileOpen.Value
		ui.SetStatus("Renaming " + fn)
		// var v string
		d := filepath.Dir(fn)
		DlgRename = DlgRename.Input("Rename", // Title
			"Renaming "+fn,    // Message
			filepath.Base(fn), // We want to rename only the base itself, not the whole path
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_OK {
					os.Rename(fn, filepath.Join(d, DlgRename.Value))
				} else {
					ui.SetStatus("Canceling rename")
				}
				// Refresh the TrvExplorer
				edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgRename", DlgRename.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgRename")
	}
}

// ****************************************************************************
// doNewFile()
// ****************************************************************************
func doNewFile(f any) {
	// d := filepath.Dir(f.(string))
	d := f.(string)
	DlgNewFile = DlgNewFile.Input("New File", // Title
		"Creating a new file into "+d, // Message
		"",                            // Empty
		func(rc dialog.DlgButton, idx int) {
			if rc == dialog.BUTTON_OK {
				CreateOrOverwriteIfItAlreadyExists(filepath.Join(d, DlgNewFile.Value), "", func(s1 string, s2 string) bool {
					edit.CloseThisFile(s1)
					edit.CreateThisFile(s1)
					return true
				})
			} else {
				ui.SetStatus("Canceling creating new file")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgNewFile", DlgNewFile.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNewFile")
}

// ****************************************************************************
// doSaveAll()
// ****************************************************************************
func doSaveAll(f any) {
	edit.SaveAll()
}

// ****************************************************************************
// doNewFolder()
// ****************************************************************************
func doNewFolder(f any) {
	// d := filepath.Dir(f.(string))
	d := f.(string)
	DlgNewFolder = DlgNewFolder.Input("New Folder", // Title
		"Creating a new folder into "+d, // Message
		"",                              // Empty
		func(rc dialog.DlgButton, idx int) {
			if rc == dialog.BUTTON_OK {
				CreateOrOverwriteIfItAlreadyExists(filepath.Join(d, DlgNewFolder.Value), "", func(s1 string, s2 string) bool {
					err := os.Mkdir(s1, os.ModePerm)
					if err != nil {
						ui.SetStatus(err.Error())
						return false
					} else {
						ui.SetStatus(fmt.Sprintf("Folder %s created", s1))
						return true
					}
				})
			} else {
				ui.SetStatus("Canceling creating new folder")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgNewFolder", DlgNewFolder.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNewFolder")
}

// ****************************************************************************
// doNewDatabase()
// ****************************************************************************
func doNewDatabase(f any) {
	// d := filepath.Dir(f.(string))
	d := f.(string)
	DlgNewDatabase = DlgNewDatabase.Input("New SQLite3 Database", // Title
		"Creating a new SQLite3 Database into "+d, // Message
		"", // Empty
		func(rc dialog.DlgButton, idx int) {
			if rc == dialog.BUTTON_OK {
				CreateOrOverwriteIfItAlreadyExists(filepath.Join(d, DlgNewDatabase.Value), "", func(s1 string, s2 string) bool {
					err := edit.OpenDB(s1)
					if err != nil {
						ui.SetStatus(err.Error())
						return false
					} else {
						ui.SetStatus(fmt.Sprintf("Database %s created", s1))
						edit.OpenView(s1, true)
						return true
					}
				})
			} else {
				ui.SetStatus("Canceling creating new database")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgNewDatabase", DlgNewDatabase.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNewDatabase")
}

// ****************************************************************************
// doOpenWorkspace()
// ****************************************************************************
func doOpenWorkspace(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		fn := DlgInputFileOpen.Value
		ui.SetStatus("Opening Workspace " + fn)
		edit.CloseAll()
		conf.ConfigGeneral.Workspace = fn
		edit.ShowTreeDir(fn, conf.ConfigGeneral.ShowHidden)
	}
}

// ****************************************************************************
// doDeleteFile()
// ****************************************************************************
func doDeleteFile(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_DELETE {
		fn := DlgInputFileDelete.Value
		if fn == "" {
			fn = DlgInputFileDelete.Path
		}
		ui.SetStatus("Deleting " + fn)
		DlgYesNo = DlgYesNo.YesNo("Delete", // Title
			"Deleting "+fn+"\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					fi, err := os.Stat(fn)
					if err != nil {
						ui.SetStatus(err.Error())
					}
					if fi.Mode().IsRegular() {
						err := os.Remove(fn)
						if err != nil {
							ui.SetStatus(err.Error())
						}
					} else {
						err := os.RemoveAll(fn)
						if err != nil {
							ui.SetStatus(err.Error())
						}
					}
				} else {
					ui.SetStatus("Canceling delete")
				}
				// Refresh the TrvExplorer
				edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	}
}

// ****************************************************************************
// SwitchShowHidden()
// ****************************************************************************
func SwitchShowHidden(dummy any) {
	conf.ConfigGeneral.ShowHidden = !conf.ConfigGeneral.ShowHidden
	ui.SetStatus(fmt.Sprintf("Show Hidden is set to %t", conf.ConfigGeneral.ShowHidden))
	edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
}

// ****************************************************************************
// SwitchConfirmExit()
// ****************************************************************************
func SwitchConfirmExit(dummy any) {
	conf.ConfigGeneral.ConfirmExit = !conf.ConfigGeneral.ConfirmExit
	ui.SetStatus(fmt.Sprintf("Confirm Exit is set to %t", conf.ConfigGeneral.ConfirmExit))
}

// ****************************************************************************
// SwitchCleanUpOnExit()
// ****************************************************************************
func SwitchCleanUpOnExit(dummy any) {
	conf.ConfigGeneral.CleanUpOnExit = !conf.ConfigGeneral.CleanUpOnExit
	ui.SetStatus(fmt.Sprintf("Clean Up on Exit is set to %t", conf.ConfigGeneral.CleanUpOnExit))
}

// ****************************************************************************
// SwitchInteractiveShell()
// ****************************************************************************
func SwitchInteractiveShell(dummy any) {
	conf.ConfigGeneral.InteractiveShell = !conf.ConfigGeneral.InteractiveShell
	ui.SetStatus(fmt.Sprintf("Interactive Shell by default is set to %t", conf.ConfigGeneral.InteractiveShell))
}

// ****************************************************************************
// SwitchPrefixShell()
// ****************************************************************************
func SwitchPrefixShell(dummy any) {
	conf.ConfigGeneral.OutErrPrefix = !conf.ConfigGeneral.OutErrPrefix
	ui.SetStatus(fmt.Sprintf("OUT & ERR prefix is set to %t", conf.ConfigGeneral.OutErrPrefix))
}

// ****************************************************************************
// doDialogShell()
// ****************************************************************************
func doDialogShell(f any) {
	sh := ""
	DlgInputShell = DlgInputShell.Command("Shell", // Title
		"CWD:"+conf.ConfigGeneral.Workspace,
		sh,
		runShell,
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget, ACmd) // Focus return
	ui.PgsApp.AddPage("dlgInputShell", DlgInputShell.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputShell")
}

// ****************************************************************************
// doInteractiveShell()
// ****************************************************************************
func doInteractiveShell() {
	// ui.FlxEditor.SetItemIndex(TxtPrompt, 1, 1, false)
	ui.PgsApp.SwitchToPage("edit")
	ui.MidColumn.AddItem(ui.TxtPrompt, 1, 1, false)
	edit.OpenView(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), false)
	edit.SwitchFollow("dummy")
	ui.TxtPrompt.SetDisabled(false)
	ui.App.SetFocus(ui.TxtPrompt)
}

// ****************************************************************************
// runShell()
// ****************************************************************************
func runShell(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		Xeq(DlgInputShell.Value)
	}
}

// ****************************************************************************
// Xeq()
// ****************************************************************************
func Xeq(c string) {
	sCmd := strings.Fields(c)
	if len(ACmd) > 0 {
		if ACmd[len(ACmd)-1] != c {
			ACmd = append(ACmd, c)
			ICmd++
		}
	} else {
		ACmd = append(ACmd, c)
		ICmd++
	}

	if len(sCmd) > 0 {
		ui.SetStatus(fmt.Sprintf("Running [%s]", c))
		if sCmd[0][0] == '!' {
			xCmd := sCmd[0] + "     "
			// Is it a line number ?
			if l, err := strconv.Atoi(strings.TrimSpace(xCmd[1:])); err == nil {
				// Yes, go to that line number
				edit.GoLine(l)
				return
			}
			// No, continue...
			xCmd = xCmd[:5]
			xCmd = strings.TrimSpace(xCmd)
			switch xCmd {
			case "!quit", "!exit", "!bye":
				ui.PgsApp.SwitchToPage("dlgQuit")
			case "!log":
				edit.OpenView(filepath.Join(appDir, conf.FILE_LOG), false)
			case "!out":
				edit.OpenView(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), false)
				edit.SwitchFollow("dummy")
			case "!foll", "!tail":
				edit.SwitchFollow("dummy")
			case "!next":
				edit.SwitchNextFile()
			case "!prev":
				edit.SwitchPreviousFile()
			case "!clos":
				edit.CloseCurrentFile()
			case "!save":
				edit.SaveFile()
			case "!conf":
				edit.OpenView(filepath.Join(appDir, conf.FILE_INI), false)
			case "!macr":
				edit.OpenView(filepath.Join(appDir, conf.FILE_MACROS), false)
			case "!help":
				ShowManual()
			case "!info":
				ShowSysInfo()
			case "!shel":
				doInteractiveShell()
			case "!b", "!bott":
				edit.GoBottom()
			case "!t", "!top":
				edit.GoTop()
			case "!h", "!time":
				t := time.Now()
				edit.InsertString(t.Format("20060102-150405"))
			case "!uuid":
				id := uuid.New()
				edit.InsertString(id.String())
			case "!lore":
				edit.InsertString(utils.GenerateLoremIpsum(1, 3, 5, 8, 15))
			default:
				ui.SetStatus(fmt.Sprintf("Invalid command %s", sCmd[0]))
			}
		} else {
			// 1. Setup the command
			cmdOptions := cmd.Options{
				Buffered:  false, // We want streaming
				Streaming: true,
			}
			xCmd := cmd.NewCmdOptions(cmdOptions, sCmd[0], sCmd[1:]...)
			activeCmd = xCmd // Assign to the shared variable
			xCmd.Dir = conf.ConfigGeneral.Workspace

			// 2. Open the log file once (use O_APPEND)
			fOut, err := os.OpenFile(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				// Handle error
			}

			// 3. Start the command in the background
			statusChan := xCmd.Start()
			// Check if the command failed to even start (e.g., command not found)
			initialStatus := xCmd.Status()
			if initialStatus.Error != nil {
				ui.SetStatus("Error: " + initialStatus.Error.Error())
				// Log it to the file as well
				fmt.Fprintln(fOut, "START ERROR: "+initialStatus.Error.Error())
				return
			}

			// 4. Handle Output & Lifecycle in a single Goroutine

			go func() {
				defer fOut.Close()

				// Write header to file
				fmt.Fprintf(fOut, "%s ⯈ %s\n", time.Now().Format("20060102-150405"), c)
				outPrefix := ""
				errPrefix := ""
				if conf.ConfigGeneral.OutErrPrefix {
					outPrefix = "OUT : "
					errPrefix = "ERR : "
				}
				for {
					select {
					case line, open := <-xCmd.Stdout:
						if !open {
							xCmd.Stdout = nil
						} else {
							fmt.Fprintln(fOut, outPrefix+line)
						}
					case line, open := <-xCmd.Stderr:
						if !open {
							xCmd.Stderr = nil
						} else {
							fmt.Fprintln(fOut, errPrefix+line)
						}
					case status := <-statusChan:
						// Command finished!
						fmt.Fprintln(fOut, fmt.Sprintf("%s ⯈ Done [%s] Exit Code: %d\n", time.Now().Format("20060102-150405"), c, status.Exit))
						ui.App.QueueUpdateDraw(func() {
							ui.SetStatus(fmt.Sprintf("Done [%s] Exit Code: %d", c, status.Exit))
						})
						return // Exit the goroutine
					}

					// If both streams are closed but status hasn't arrived,
					// we still need to wait for statusChan to avoid leaking
					if xCmd.Stdout == nil && xCmd.Stderr == nil && statusChan == nil {
						return
					}
				}
			}()
		}
	} else {
		ui.SetStatus("Nothing to run")
	}
}

// ****************************************************************************
// XeqOut()
// ****************************************************************************
func XeqOut(c string) string {
	// sCmd := strings.Fields(c)
	// https://stackoverflow.com/questions/47489745/splitting-a-string-at-space-except-inside-quotation-marks
	quoted := false
	sCmd := strings.FieldsFunc(c, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	out := ""
	if len(sCmd) > 0 {
		cmd := exec.Command(sCmd[0], sCmd[1:]...)
		cmd.Dir = conf.ConfigGeneral.Workspace
		ui.SetStatus(fmt.Sprintf("Executing [%s] in %s", c, cmd.Dir))
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			out = "Error : " + err.Error()
			if exitError, ok := err.(*exec.ExitError); ok {
				out = out + fmt.Sprintf("\nExit code %d", exitError.ExitCode())
			}
		} else {
			out = outb.String()
			out = out + errb.String()
			out = out + "\nExit code 0"
		}
	} else {
		out = "Nothing to run\n\nExit code 0"
	}

	out = strings.TrimSpace(out)
	ui.SetStatus(out)
	ui.SetStatus(fmt.Sprintf("Done [%s]", c))
	return out
}

// ****************************************************************************
// XeqOutErr()
// ****************************************************************************
func XeqOutErr(c string) string {
	// sCmd := strings.Fields(c)
	// https://stackoverflow.com/questions/47489745/splitting-a-string-at-space-except-inside-quotation-marks
	quoted := false
	sCmd := strings.FieldsFunc(c, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	out := ""
	if len(sCmd) > 0 {
		cmd := exec.Command(sCmd[0], sCmd[1:]...)
		cmd.Dir = conf.ConfigGeneral.Workspace
		ui.SetStatus(fmt.Sprintf("Executing [%s] in %s", c, cmd.Dir))
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			out = "Error : " + err.Error()
			if exitError, ok := err.(*exec.ExitError); ok {
				out = out + fmt.Sprintf("\nExit code %d", exitError.ExitCode())
				out = out + outb.String()
				out = out + errb.String()
			}
		} else {
			out = outb.String()
			out = out + errb.String()
			out = out + "\nExit code 0"
		}
	} else {
		out = "Nothing to run\n\nExit code 0"
	}

	out = strings.TrimSpace(out)
	ui.SetStatus(out)
	ui.SetStatus(fmt.Sprintf("Done [%s]", c))
	return out
}

// ****************************************************************************
// XeqRaw()
// ****************************************************************************
func XeqRaw(c string) string {
	// sCmd := strings.Fields(c)
	// https://stackoverflow.com/questions/47489745/splitting-a-string-at-space-except-inside-quotation-marks
	quoted := false
	sCmd := strings.FieldsFunc(c, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	out := ""
	if len(sCmd) > 0 {
		cmd := exec.Command(sCmd[0], sCmd[1:]...)
		cmd.Dir = conf.ConfigGeneral.Workspace
		ui.SetStatus(fmt.Sprintf("Executing [%s] in %s", c, cmd.Dir))
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			out = "Error : " + err.Error()
			if exitError, ok := err.(*exec.ExitError); ok {
				out = out + fmt.Sprintf("\nExit code %d", exitError.ExitCode())
			}
		} else {
			out = outb.String()
			out = out + errb.String()
		}
	} else {
		out = "Nothing to run\n\nExit code 0"
	}

	out = strings.TrimSpace(out)
	ui.SetStatus(out)
	ui.SetStatus(fmt.Sprintf("Done [%s]", c))
	return out
}

// ****************************************************************************
// checkNewVersion()
// ****************************************************************************
func checkNewVersion() {
	ui.SetStatus("Checking for new version...")

	// Extract local commit hash from conf.Version
	versionParts := strings.Split(conf.Version, "-")
	if len(versionParts) < 2 {
		ui.SetStatus("Invalid version string format. Skipping version check.")
		return
	}
	localCommitHash := strings.TrimSpace(versionParts[1])

	// Get remote commit hash
	remoteInfo := XeqRaw("git ls-remote https://github.com/jplozf/lied HEAD")
	if strings.Contains(remoteInfo, "fatal") {
		ui.SetStatus("Could not fetch remote version. Skipping version check.")
		return
	}

	remoteCommitHash := strings.Fields(remoteInfo)[0]

	// Truncate remote hash to match local hash length for comparison
	if len(remoteCommitHash) > len(localCommitHash) {
		remoteCommitHash = remoteCommitHash[:len(localCommitHash)]
	}

	ui.SetStatus(fmt.Sprintf("Local: '%s' (len %d), Remote: '%s' (len %d)", localCommitHash, len(localCommitHash), remoteCommitHash, len(remoteCommitHash)))
	if localCommitHash != remoteCommitHash {
		ShowNewVersionPopup(localCommitHash, remoteCommitHash)
	} else {
		ui.SetStatus("You are running the latest version.")
	}
}

// ****************************************************************************
// ShowNewVersionPopup()
// ****************************************************************************
func ShowNewVersionPopup(localHash, remoteHash string) {
	msg := fmt.Sprintf("A new version of Lied is available online!\n\nYour version  : %s\nLatest online : %s\n\nPlease update your application.", localHash, remoteHash)
	MsgBox = MsgBox.OK("New Version Available", msg, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
	ui.PgsApp.AddPage("msgNewVersion", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgNewVersion")
}

// ****************************************************************************
// DoArchive()
// ****************************************************************************
func DoArchive(f any) {
	ui.SetStatus("Creating archive for " + f.(string))
	userDir, _ := os.UserHomeDir()
	b := utils.FilenameWithoutExtension(filepath.Base(f.(string))) + "_" + time.Now().Format("20060102-150405") + ".zip"
	if err := utils.ZipIt(f.(string), filepath.Join(userDir, b)); err != nil {
		ui.SetStatus(err.Error())
	} else {
		ui.SetStatus(fmt.Sprintf("Archive [%s] created successfully into [%s]", b, userDir))
	}
}

// ****************************************************************************
// DoExplorer()
// ****************************************************************************
func DoExplorer(f any) {
	/*
		ui.SetStatus("Exploring " + f.(string))
		ui.PgsApp.SwitchToPage("fileManager")
		edit.ShowFiles()
		ui.App.SetFocus(ui.TblFiles)
	*/
	edit.OpenView(path.Dir(f.(string)), false)
}

// ****************************************************************************
// createFile()
// ****************************************************************************
func createFile(fName string, text string) {
	f, err := os.Create(fName)
	if err == nil {
		defer f.Close()
		fmt.Fprintln(f, text)
	}
}

// ****************************************************************************
// ShowSysInfo()
// ****************************************************************************
func ShowSysInfo() {
	MsgBox = MsgBox.OK(" System Info ", sysinfo.GetFullReport(), nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
	ui.SetStatus("Displaying System Info")

	// Generate HTML System Information report
	userDir, err := os.UserHomeDir()
	if err == nil {
		f, err := os.Create(filepath.Join(userDir, "sysinfo.html"))
		if err == nil {
			defer f.Close()
			f.WriteString(sysinfo.GetHTMLReport())
			ui.SetStatus("Generating System Info HTML report into [" + userDir + "] folder")
		}
	}
}

// ****************************************************************************
// ArchiveLogs()
// ****************************************************************************
func ArchiveLogs() {
	ui.SetStatus("Archiving logs")
	type tLogs struct {
		yyyymm string
		text   string
	}
	var logs []tLogs
	// Opening the log file
	file, err := os.Open(filepath.Join(appDir, conf.FILE_LOG))
	if err != nil {
	} else {
		defer file.Close()
		// Read the log file
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			logs = append(logs, tLogs{line[:6], line})
		}
		var currentMonth = time.Now().Format("200601")
		var tagFiles []string
		for _, t := range logs {
			// Keep a trace of files we'll have to zip after
			if !slices.Contains(tagFiles, t.yyyymm) {
				tagFiles = append(tagFiles, t.yyyymm)
			}
			// If the file doesn't exist, create it, or append to the file
			f, err := os.OpenFile(filepath.Join(appDir, strings.ToLower(conf.APP_NAME)+"_"+t.yyyymm+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(t.text + "\n")
			}
		}
		// Zipping the different files except for the actual month
		for _, tag := range tagFiles {
			if tag != currentMonth {
				utils.ZipIt(filepath.Join(appDir, strings.ToLower(conf.APP_NAME)+"_"+tag+".log"), filepath.Join(appDir, strings.ToLower(conf.APP_NAME)+"_"+tag+".zip"))
				os.Remove(filepath.Join(appDir, strings.ToLower(conf.APP_NAME)+"_"+tag+".log"))
			} else {
				os.Rename(filepath.Join(appDir, conf.FILE_LOG), filepath.Join(appDir, conf.FILE_LOG+".bak"))
				os.Rename(filepath.Join(appDir, strings.ToLower(conf.APP_NAME)+"_"+tag+".log"), filepath.Join(appDir, conf.FILE_LOG))
			}
		}
	}
}

// ****************************************************************************
// CreateOrOverwriteIfItAlreadyExists()
// ****************************************************************************
func CreateOrOverwriteIfItAlreadyExists(target string, source string, fn createFunc) {
	if utils.IsFileExist(target) {
		DlgYesNo = DlgYesNo.YesNo("Already exists", // Title
			fmt.Sprintf("The file %s already exists.\nIt will be overwritten.\n\nAre you sure you want to proceed ?", target), // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					if fn(target, source) {
						ui.SetStatus(fmt.Sprintf("%s created successfully", target))
					} else {
						ui.SetStatus(fmt.Sprintf("Error when created %s", target))
					}
				} else {
					ui.SetStatus("Aborting creation")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		if fn(target, source) {
			ui.SetStatus(fmt.Sprintf("%s created successfully", target))
		} else {
			ui.SetStatus(fmt.Sprintf("Error when created %s", target))
		}
	}
}

// ****************************************************************************
// shellEscape()
// ****************************************************************************
func shellEscape() {
	ui.App.Suspend(func() {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}

		cmd := exec.Command(shell)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Println()
		fmt.Println("--------- Temporary return to shell ---------")
		fmt.Println("Type 'exit' or press Ctrl+D to return to Lied")
		fmt.Println("---------------------------------------------")
		fmt.Println()

		_ = cmd.Run()

		fmt.Println("Returning to Lied...")
	})
}

// ****************************************************************************
// ShowManual()
// ****************************************************************************
func ShowManual() {
	for _, f := range edit.OpenViews {
		if f.FName == "Lied Manual" {
			edit.SwitchAnyFile(f.FName)
			ui.SetStatus("Switching to help manual")
			return
		}
	}

	helpBuf := femto.NewBufferFromString(help.HelpText, "Help.txt")

	helpView := &edit.ViewScreen{
		FName:       "Lied Manual",
		FemtoBuffer: helpBuf,
		FemtoView:   femto.NewView(helpBuf),
		ReadWrite:   false,
	}

	/*
		efiles = append(efiles, helpFile)
		switchDocument(len(efiles) - 1)
	*/
	edit.OpenViews = append(edit.OpenViews, *helpView)
	edit.SwitchAnyFile(helpView.FName)
	ui.SetStatus("Opening help manual")
}
