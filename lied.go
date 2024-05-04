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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"

	"lied/conf"
	"lied/edit"
	"lied/help"
	"lied/menu"
	"lied/ui"
	"lied/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	appDir   string
	hostname string
	greeting string
	err      error
	MnuMain  *menu.Menu
)

// ****************************************************************************
// init()
// ****************************************************************************
func init() {
	ui.SessionID, _ = utils.RandomHex(3)
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	user, err := user.Current()
	if err != nil {
		log.Fatalf(err.Error())
	}
	cmd.CurrentUser = user.Username
	greeting = cmd.CurrentUser + "@" + hostname + "⯈"

	cmd.ICmd = 0
	ui.App = tview.NewApplication()
	ui.SetUI(appQuit, greeting)

	ui.PgsApp.AddPage("shell", ui.FlxShell, true, true)
	ui.PgsApp.AddPage("dlgQuit", ui.DlgQuit, false, false)

	userDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	// Set the Current Working Directory
	conf.Cwd, _ = os.Getwd()
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

	jsonFile, err := os.Open(filepath.Join(appDir, conf.FILE_CONFIG))
	if err == nil {
		// Read config from json file
		defer jsonFile.Close()
		bValues, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(bValues, &ui.MyConfig)
		ui.SetStatus("Reading config from json")
	} else {
		// Set default config (Sorry, default time and date formats are the French way ;)
		ui.MyConfig.StartupScreen = ui.ModeShell
		ui.MyConfig.FormatDate = "02/01/2006"
		ui.MyConfig.FormatTime = "15:04:05"
		ui.SetStatus("Set default config")
		// Write config to json file
		jsonFile, _ := json.MarshalIndent(ui.MyConfig, "", " ")
		_ = ioutil.WriteFile(filepath.Join(appDir, conf.FILE_CONFIG), jsonFile, 0644)
	}

	ui.SetStatus(fmt.Sprintf("Starting session #%s", ui.SessionID))
	readSettings()
}

// ****************************************************************************
// main()
// ****************************************************************************
func main() {
	// Main keyboard's events manager
	ui.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			ui.AddNewScreen(ui.ModeHelp, help.SelfInit, nil)
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TxtPrompt)
		case tcell.KeyF3:
			ui.CloseCurrentScreen()
		case tcell.KeyF6:
			ui.ShowPreviousScreen()
		case tcell.KeyF7:
			ui.ShowNextScreen()
		case tcell.KeyF10:
			ShowMainMenu()
		case tcell.KeyF12:
			ShowQuitDialog(nil)
		case tcell.KeyCtrlC:
			return nil
		case tcell.KeyCtrlO:
			if ui.CurrentMode == ui.ModeSQLite3 {
				sq3.DoOpenDB(conf.Cwd)
			}
			if ui.CurrentMode == ui.ModeHexEdit {
				hexedit.DoOpen(conf.Cwd)
			}
		case tcell.KeyCtrlS:
			if ui.CurrentMode == ui.ModeSQLite3 {
				sq3.DoCloseDB()
			}
			if ui.CurrentMode == ui.ModeHexEdit {
				hexedit.Close()
			}
		case tcell.KeyEsc:
			if ui.CurrentMode == ui.ModeShell {
				ui.SetStatus("YO!")
				ui.App.ForceDraw()
			}
		}
		return event
	})

	// Files panel keyboard's events manager
	ui.TblFiles.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			fm.ProceedFileAction()
			return nil
		case tcell.KeyF5:
			fm.RefreshMe()
		case tcell.KeyF8:
			fm.ShowMenu()
			return nil
		case tcell.KeyCtrlS:
			fm.ShowMenuSort()
			return nil
		case tcell.KeyInsert:
			fm.ProceedFileSelect()
			return nil
		case tcell.KeyCtrlA:
			fm.SelectAll(nil)
			return nil
		case tcell.KeyCtrlC:
			fm.DoCopy(nil)
			return nil
		case tcell.KeyCtrlX:
			fm.DoCut(nil)
			return nil
		case tcell.KeyCtrlV:
			fm.DoPaste(nil)
			return nil
		case tcell.KeyDelete:
			fm.DoDelete(nil)
			return nil
		case tcell.KeyTab:
			if ui.TxtPrompt.HasFocus() {
				ui.App.SetFocus(ui.TblFiles)
			} else {
				ui.App.SetFocus(ui.TxtFileInfo)
			}
			return nil
		}
		return event
	})

	// Process panel keyboard's events manager
	ui.TblProcess.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			pm.ProceedProcessAction()
			return nil
		case tcell.KeyF5:
			pm.RefreshMe()
		case tcell.KeyF8:
			pm.ShowMenu()
			return nil
		case tcell.KeyCtrlF:
			pm.DoFindProcess(nil)
			return nil
		case tcell.KeyCtrlS:
			pm.ShowMenuSort()
			return nil
		case tcell.KeyCtrlV:
			pm.SwitchView()
			return nil
		case tcell.KeyTab:
			if ui.TxtPrompt.HasFocus() {
				ui.App.SetFocus(ui.TblProcess)
				return nil
			}
			if ui.TblProcess.HasFocus() {
				ui.App.SetFocus(ui.TblProcUsers)
				return nil
			}
			if ui.TblProcUsers.HasFocus() {
				ui.App.SetFocus(ui.TxtPrompt)
				return nil
			}
		}
		return event
	})

	// TblProcUsers panel keyboard's events manager
	ui.TblProcUsers.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			idx, _ := ui.TblProcUsers.GetSelection()
			pm.ShowProcesses(ui.TblProcUsers.GetCell(idx, 1).Text)
			ui.App.Sync()
			ui.App.SetFocus(ui.TblProcess)
			return nil
		/*
			case tcell.KeyF8:
				fm.ShowMenu()
				return nil
		*/
		case tcell.KeyTab:
			if ui.TblProcUsers.HasFocus() {
				ui.App.SetFocus(ui.TxtProcInfo)
				return nil
			}
		}
		return event
	})

	// ProcInfo keyboard's events manager
	ui.TxtProcInfo.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			ui.App.SetFocus(ui.TxtPrompt)
			return nil
		}
		return event
	})

	// FileInfo keyboard's events manager
	ui.TxtFileInfo.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			ui.App.SetFocus(ui.TxtPrompt)
			return nil
		}
		return event
	})

	// Prompt keyboard's events manager
	ui.TxtPrompt.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			if ui.CurrentMode == ui.ModeSQLite3 {
				if ui.TxtPrompt.GetText() != "" {
					sq3.Xeq(ui.TxtPrompt.GetText())
				}
			} else {
				if ui.TxtPrompt.GetText() != "" {
					cmd.Xeq(ui.TxtPrompt.GetText())
				}
			}
			return nil
		case tcell.KeyUp:
			if ui.CurrentMode == ui.ModeSQLite3 {
				if len(sq3.ACmd) > 0 {
					if sq3.ICmd < len(sq3.ACmd)-1 {
						sq3.ICmd++
					} else {
						sq3.ICmd = 0
					}
					ui.TxtPrompt.SetText(sq3.ACmd[sq3.ICmd], true)
					ui.TxtPrompt.Select(0, ui.TxtPrompt.GetTextLength())
				}
			} else {
				if len(cmd.ACmd) > 0 {
					if cmd.ICmd < len(cmd.ACmd)-1 {
						cmd.ICmd++
					} else {
						cmd.ICmd = 0
					}
					ui.TxtPrompt.SetText(cmd.ACmd[cmd.ICmd], true)
					ui.TxtPrompt.Select(0, ui.TxtPrompt.GetTextLength())
				}
			}
			return nil
		case tcell.KeyDown:
			if ui.CurrentMode == ui.ModeSQLite3 {
				if len(sq3.ACmd) > 0 {
					if sq3.ICmd > 0 {
						sq3.ICmd--
					} else {
						sq3.ICmd = len(sq3.ACmd) - 1
					}
					ui.TxtPrompt.SetText(sq3.ACmd[sq3.ICmd], true)
					ui.TxtPrompt.Select(0, ui.TxtPrompt.GetTextLength())
				}
			} else {
				if len(cmd.ACmd) > 0 {
					if cmd.ICmd > 0 {
						cmd.ICmd--
					} else {
						cmd.ICmd = len(cmd.ACmd) - 1
					}
					ui.TxtPrompt.SetText(cmd.ACmd[cmd.ICmd], true)
					ui.TxtPrompt.Select(0, ui.TxtPrompt.GetTextLength())
				}
			}
			return nil
		case tcell.KeyTab:
			if ui.CurrentMode == ui.ModeFiles {
				ui.App.SetFocus(ui.TblFiles)
			}
			if ui.CurrentMode == ui.ModeShell {
				ui.App.SetFocus(ui.TxtConsole)
			}
			if ui.CurrentMode == ui.ModeProcess {
				ui.App.SetFocus(ui.TblProcess)
			}
			if ui.CurrentMode == ui.ModeTextEdit {
				ui.App.SetFocus(ui.EdtMain)
			}
			if ui.CurrentMode == ui.ModeSQLite3 {
				ui.App.SetFocus(ui.TblSQLOutput)
			}
			if ui.CurrentMode == ui.ModeHexEdit {
				ui.App.SetFocus(ui.TblHexEdit)
			}
			return nil
		}
		return event
	})

	// HexEdit keyboard's events manager
	ui.TblHexEdit.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			ui.App.SetFocus(ui.TxtFileInfo)
			return nil
		}
		return event
	})

	// Console keyboard's events manager
	ui.TxtConsole.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			ui.App.SetFocus(ui.TxtPrompt)
			return nil
		}
		return event
	})

	// Editor keyboard's events manager
	ui.EdtMain.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		evkSaveAs := tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt)
		if event.Key() == evkSaveAs.Key() && event.Rune() == evkSaveAs.Rune() && event.Modifiers() == evkSaveAs.Modifiers() {
			edit.SaveFileAs()
			return nil
		}
		switch event.Key() {
		case tcell.KeyCtrlS:
			edit.SaveFile()
			return nil
		case tcell.KeyCtrlN:
			edit.NewFile(conf.Cwd)
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
		case tcell.KeyEsc:
			ui.App.SetFocus(ui.TblOpenFiles)
			return nil
		}
		return event
	})
	ui.TblOpenFiles.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			ui.App.SetFocus(ui.TrvExplorer)
			return nil
		case tcell.KeyEnter:
			idx, _ := ui.TblOpenFiles.GetSelection()
			fName := ui.TblOpenFiles.GetCell(idx, 3).Text
			edit.SwitchOpenFile(fName)
			ui.App.SetFocus(ui.EdtMain)
			return nil
		}
		return event
	})
	ui.TrvExplorer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			ui.App.SetFocus(ui.TxtPrompt)
			return nil
		}
		return event
	})

	// SQLite3 keyboard's events manager
	ui.TblSQLOutput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF8:
			sq3.ShowMenu()
			return nil
		case tcell.KeyTab:
			ui.App.SetFocus(ui.TblSQLTables)
			return nil
		}
		return event
	})
	ui.TblSQLTables.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			ui.App.SetFocus(ui.TrvSQLDatabase)
			return nil
		}
		return event
	})
	ui.TrvSQLDatabase.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			ui.App.SetFocus(ui.TxtPrompt)
			return nil
		}
		return event
	})

	ui.SetTitle(conf.APP_STRING)
	ui.SetStatus("Welcome.")
	switch ui.MyConfig.StartupScreen {
	case ui.ModeShell:
		SwitchToShell(nil)
	case ui.ModeFiles:
		SwitchToFiles(nil)
	case ui.ModeProcess:
		SwitchToProcess(nil)
	case ui.ModeTextEdit:
		SwitchToTextEdit(nil)
	case ui.ModeSQLite3:
		SwitchToSQLite3(nil)
	case ui.ModeHelp:
		SwitchToHelp(nil)
	case ui.ModeHexEdit:
		SwitchToHexEdit(nil)
	}
	welcome()

	go ui.UpdateTime()
	go utils.GetCpuUsage()
	// FIXME : SetFocus on the correct ui.FlxXXX depending on the ui.MyConfig.StartupScreen
	if err := ui.App.SetRoot(ui.PgsApp, true).SetFocus(ui.FlxShell).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// ****************************************************************************
// ShowMainMenu()
// ****************************************************************************
func ShowMainMenu() {
	MnuMain = MnuMain.New(" "+conf.APP_NAME+" ", ui.GetCurrentScreen(), ui.TxtPrompt)
	// Dynamic options (screens currently open)
	for i := 0; i < len(ui.ArrScreens); i++ {
		chk := false
		if i == ui.IdxScreens {
			chk = true
		}
		MnuMain.AddItem(ui.ArrScreens[i].ID, fmt.Sprintf("%2d) %s-%s", i+1, ui.ArrScreens[i].Title, ui.ArrScreens[i].ID), ui.ShowScreen, i, true, chk)
	}
	MnuMain.AddSeparator()
	// Fixed options
	MnuMain.AddItem("mnuHelp", "Help", SwitchToHelp, nil, true, false)
	MnuMain.AddItem("mnuShell", "Shell", SwitchToShell, nil, true, false)
	MnuMain.AddItem("mnuFiles", "File Manager", SwitchToFiles, nil, true, false)
	MnuMain.AddItem("mnuProcess", "Process and Services", SwitchToProcess, nil, true, false)
	MnuMain.AddItem("mnuTextEdit", "Text Editor", SwitchToTextEdit, nil, true, false)
	MnuMain.AddItem("mnuSQLite3", "SQLite3 Manager", SwitchToSQLite3, nil, true, false)
	MnuMain.AddItem("mnuHexEdit", "Hexadecimal Editor", SwitchToHexEdit, nil, true, false)
	MnuMain.AddSeparator()
	MnuMain.AddItem("mnuQuit", "Quit", ShowQuitDialog, nil, true, false)

	ui.PgsApp.AddPage("dlgMainMenu", MnuMain.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgMainMenu")
}

// ****************************************************************************
// SwitchToHelp(p any)
// ****************************************************************************
func SwitchToHelp(p any) {
	ui.AddNewScreen(ui.ModeHelp, help.SelfInit, nil)
}

// ****************************************************************************
// SwitchToShell(p any)
// ****************************************************************************
func SwitchToShell(p any) {
	ui.AddNewScreen(ui.ModeShell, nil, nil)
}

// ****************************************************************************
// SwitchToFiles(p any)
// ****************************************************************************
func SwitchToFiles(p any) {
	ui.AddNewScreen(ui.ModeFiles, fm.SelfInit, nil)
}

// ****************************************************************************
// SwitchToProcess(p any)
// ****************************************************************************
func SwitchToProcess(p any) {
	ui.AddNewScreen(ui.ModeProcess, pm.SelfInit, cmd.CurrentUser)
}

// ****************************************************************************
// SwitchToTextEdit(p any)
// ****************************************************************************
func SwitchToTextEdit(p any) {
	ui.AddNewScreen(ui.ModeTextEdit, edit.SelfInit, nil)
}

// ****************************************************************************
// SwitchToSQLite3(p any)
// ****************************************************************************
func SwitchToSQLite3(p any) {
	ui.AddNewScreen(ui.ModeSQLite3, sq3.SelfInit, nil)
}

// ****************************************************************************
// SwitchToHexEdit(p any)
// ****************************************************************************
func SwitchToHexEdit(p any) {
	ui.AddNewScreen(ui.ModeHexEdit, hexedit.SelfInit, nil)
}

// ****************************************************************************
// appQuit()
// appQuit performs some cleanup and saves persistent data before quitting application
// ****************************************************************************
func appQuit() {
	// TODO : Clean up gosh_edit_ null files
	edit.CheckOpenFilesForSaving()
	saveSettings()
	ui.SetStatus(fmt.Sprintf("Quitting session #%s", ui.SessionID))
	ui.App.Stop()
	fmt.Printf("\n👻%s\n\n", conf.APP_STRING)
}

// ****************************************************************************
// readSettings()
// ****************************************************************************
func readSettings() {
	// Read commands history file
	ui.SetStatus("Reading commands history")
	fCmd, err := os.Open(filepath.Join(appDir, conf.FILE_HISTORY_CMD))
	if err != nil {
		return
	}
	defer fCmd.Close()
	sCmd := bufio.NewScanner(fCmd)
	for sCmd.Scan() {
		cmd.ACmd = append(cmd.ACmd, sCmd.Text())
	}
	// Read SQL history file
	ui.SetStatus("Reading SQL history")
	fSQL, err := os.Open(filepath.Join(appDir, conf.FILE_HISTORY_SQL))
	if err != nil {
		return
	}
	defer fSQL.Close()
	sSQL := bufio.NewScanner(fSQL)
	for sSQL.Scan() {
		sq3.ACmd = append(sq3.ACmd, sSQL.Text())
	}
}

// ****************************************************************************
// saveSettings()
// ****************************************************************************
func saveSettings() {
	// Save commands history file
	ui.SetStatus("Saving commands history")
	fCmd, err := os.Create(filepath.Join(appDir, conf.FILE_HISTORY_CMD))
	if err != nil {
		return
	}
	defer fCmd.Close()
	wCmd := bufio.NewWriter(fCmd)
	for _, line := range cmd.ACmd {
		fmt.Fprintln(wCmd, line)
	}
	wCmd.Flush()
	// Save SQL history file
	ui.SetStatus("Saving SQL history")
	fSQL, err := os.Create(filepath.Join(appDir, conf.FILE_HISTORY_SQL))
	if err != nil {
		return
	}
	defer fSQL.Close()
	wSQL := bufio.NewWriter(fSQL)
	for _, line := range sq3.ACmd {
		fmt.Fprintln(wSQL, line)
	}
	wSQL.Flush()
}

// ****************************************************************************
// welcome()
// ****************************************************************************
func welcome() {
	w1 := ":: Welcome to " + conf.APP_STRING + " :"
	w2 := conf.APP_NAME + " version " + conf.APP_VERSION + " - " + conf.APP_URL + "\n"
	os := runtime.GOOS
	if os == "windows" {
		out, err := exec.Command("ver").Output()
		if err == nil {
			w2 = w2 + string(out)
		}

	} else {
		out, err := exec.Command("uname", "-a").Output()
		if err == nil {
			w2 = w2 + string(out)
		}
	}
	ui.LblHostname.SetText("👻" + greeting)
	ui.HeaderConsole(w1)
	ui.OutConsole(w2)
}

// ****************************************************************************
// ShowQuitDialog()
// ****************************************************************************
func ShowQuitDialog(p any) {
	ui.PgsApp.SwitchToPage("dlgQuit")
}
