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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

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
	args     []string
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
	greeting = user.Username + "@" + hostname + "⯈"

	ui.App = tview.NewApplication()
	ui.SetUI(appQuit, greeting)

	ui.PgsApp.AddPage("edit", ui.FlxEditor, true, true)
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
			// ui.App.SetFocus(ui.TxtPrompt)
			return nil
		}
		return event
	})

	// edit.SwitchToEditor("")
	if len(args) > 1 {
		fName, _ := filepath.Abs(args[1])
		edit.OpenFile(fName)
	} else {
		edit.NewFileOrLastFile(conf.Cwd)
	}

	ui.SetTitle(conf.APP_STRING)
	ui.SetStatus("Welcome.")
	ui.LblHostname.SetText("♯" + greeting)

	go ui.UpdateTime()
	if err := ui.App.SetRoot(ui.PgsApp, true).SetFocus(ui.FlxEditor).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// ****************************************************************************
// ShowMainMenu()
// ****************************************************************************
func ShowMainMenu() {
	MnuMain = MnuMain.New(" "+conf.APP_NAME+" ", ui.GetCurrentScreen(), ui.EdtMain)
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
	MnuMain.AddSeparator()
	MnuMain.AddItem("mnuQuit", "Quit", ShowQuitDialog, nil, true, false)

	ui.PgsApp.AddPage("dlgMainMenu", MnuMain.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgMainMenu")
}

// ****************************************************************************
// appQuit()
// appQuit performs some cleanup and saves persistent data before quitting application
// ****************************************************************************
func appQuit() {
	// TODO : Clean up lied_XXX null files
	edit.CheckOpenFilesForSaving()
	saveSettings()
	ui.SetStatus(fmt.Sprintf("Quitting session #%s", ui.SessionID))
	ui.App.Stop()
	fmt.Printf("\n♯%s\n\n", conf.APP_STRING)
}

// ****************************************************************************
// readSettings()
// ****************************************************************************
func readSettings() {
	// TODO : Restore the MRU
}

// ****************************************************************************
// saveSettings()
// ****************************************************************************
func saveSettings() {
	// TODO : Save the MRU
}

// ****************************************************************************
// ShowQuitDialog()
// ****************************************************************************
func ShowQuitDialog(p any) {
	ui.PgsApp.SwitchToPage("dlgQuit")
}
