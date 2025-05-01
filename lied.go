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
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"lied/conf"
	"lied/dialog"
	"lied/edit"
	"lied/help"
	"lied/menu"
	"lied/ui"
	"lied/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/ini.v1"
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	appDir              string
	hostname            string
	greeting            string
	err                 error
	MnuMain             *menu.Menu
	MnuConfig           *menu.Menu
	args                []string
	config              conf.Config
	MnuInputTheme       *menu.Menu
	DlgInputGitUser     *dialog.Dialog
	DlgInputGitPassword *dialog.Dialog
	DlgInputFormatTime  *dialog.Dialog
	DlgInputFormatDate  *dialog.Dialog
	DlgInputFileOpen    *dialog.Dialog
	DlgInputShell       *dialog.Dialog
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
	ui.CurrentMode = ui.ModeTextEdit
	// ui.AddNewScreen(ui.ModeTextEdit, edit.SelfInit, nil)
	ui.PgsApp.AddPage("dlgQuit", ui.DlgQuit, false, false)

	userDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	// Set the Current Working Directory
	config.Workspace, _ = os.Getwd()
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

	/*
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
	*/

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
			// ui.AddNewScreen(ui.ModeHelp, help.SelfInit, nil)
			SwitchHelp()
		case tcell.KeyF8:
			ShowConfigMenu()
		case tcell.KeyF6:
			// ui.ShowPreviousScreen()
			edit.SwitchPreviousFile()
		case tcell.KeyF7:
			// ui.ShowNextScreen()
			edit.SwitchNextFile()
		case tcell.KeyF10:
			ShowMainMenu()
		case tcell.KeyF4:
			InputShell(nil)
		case tcell.KeyF12:
			ShowQuitDialog(nil)
		case tcell.KeyCtrlC:
			edit.CurrentFile.View.Copy()
			return nil
		case tcell.KeyCtrlX:
			edit.CurrentFile.View.Cut()
			return nil
		case tcell.KeyCtrlZ:
			edit.CurrentFile.View.Undo()
			return nil
		case tcell.KeyCtrlY:
			edit.CurrentFile.View.Redo()
			return nil
		case tcell.KeyCtrlA:
			edit.CurrentFile.View.SelectAll()
			return nil
		case tcell.KeyCtrlV:
			edit.CurrentFile.View.Paste()
			return nil
		case tcell.KeyCtrlL:
			edit.CurrentFile.View.DeleteLine()
			return nil
		case tcell.KeyCtrlS:
			edit.SaveFile()
			return nil
		case tcell.KeyCtrlN:
			edit.NewFile(config.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(config.Workspace)
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
			/*
				case tcell.KeyEsc:
					ui.App.SetFocus(ui.TblOpenFiles)
					return nil
			*/
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
			edit.NewFile(config.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(config.Workspace)
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
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TrvExplorer)
			return nil
		case tcell.KeyEnter:
			idx, _ := ui.TblOpenFiles.GetSelection()
			fName := ui.TblOpenFiles.GetCell(idx, 3).Text
			edit.SwitchOpenFile(fName)
			edit.SetFocusOnPath(fName)
			ui.App.SetFocus(ui.EdtMain)
			return nil
		}
		return event
	})
	ui.TrvExplorer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			ui.App.SetFocus(ui.EdtMain)
			return nil
		}
		return event
	})
	ui.EdtMain.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblOpenFiles)
			return nil
		}
		return event
	})

	edit.ShowTreeDir(config.Workspace, config.ShowHidden)

	// * Launching lied without args : Open last workspace and last open files if any, else open a temporary file into the current directory as workspace
	// * Launching lied with directory as argument : Open a temporary file into this directory as workspace
	// * Launching lied with file name as argument : Open this file into its directory as workspace
	if len(args) > 1 {
		edit.NewFileOrLastFile(config.Workspace)
		fName, _ := filepath.Abs(args[1])
		if utils.IsFileExist(fName) {
			edit.OpenFile(fName)
		} else {
			f, e := os.Create(fName)
			if e != nil {
				ui.SetStatus(fmt.Sprintf("Can't create '%s' file", fName))
			} else {
				f.Close()
				edit.OpenFile(fName)
			}
		}
	} else {
		edit.NewFileOrLastFile(config.Workspace)
	}

	ui.SetTitle(conf.APP_STRING)
	ui.SetStatus("Welcome")
	ui.LblHostname.SetText("♯" + greeting)

	go ui.UpdateTime()
	if err := ui.App.SetRoot(ui.PgsApp, true).SetFocus(ui.EdtMain).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
	// ui.App.SetFocus(ui.EdtMain)
}

// ****************************************************************************
// ShowMainMenu()
// ****************************************************************************
func ShowMainMenu() {
	MnuMain = MnuMain.New(" "+conf.APP_NAME+" ", ui.GetCurrentScreen(), ui.EdtMain)
	// Dynamic options (files currently open)
	for i, e := range edit.OpenFiles {
		chk := false
		if e.FName == edit.CurrentFile.FName {
			chk = true
		}
		sha, _ := utils.GetSha256(e.FName)
		MnuMain.AddItem(sha,
			fmt.Sprintf("%2d) %s", i+1, filepath.Base(e.FName)),
			edit.SwitchAnyFile,
			e.FName,
			true,
			chk)
	}
	// Fixed options
	MnuMain.AddSeparator()
	// MnuMain.AddItem("mnuOpenWorkspace", "Open Workspace", edit.OpenWorkspace, nil, true, false)
	MnuMain.AddItem("mnuSave", "Save", edit.SaveAnyFile, nil, true, false)
	MnuMain.AddItem("mnuSaveAs", "Save as…", edit.SaveAnyFileAs, nil, true, false)
	MnuMain.AddItem("mnuNew", "New", edit.NewAnyFile, config.Workspace, true, false)
	MnuMain.AddItem("mnuOpen", "Open…", InputFileOpen, config.Workspace, true, false)
	MnuMain.AddItem("mnuClose", "Close", edit.CloseAnyFile, nil, true, false)
	MnuMain.AddSeparator()
	MnuMain.AddItem("mnuQuit", "Quit", ShowQuitDialog, nil, true, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgMainMenu", MnuMain.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgMainMenu")
}

// ****************************************************************************
// ShowConfigMenu()
// ****************************************************************************
func ShowConfigMenu() {
	MnuConfig = MnuConfig.New(" Settings ", ui.GetCurrentScreen(), ui.EdtMain)
	// Menu Options
	MnuConfig.AddItem("mnuCfgTheme", "Theme", InputConfigTheme, nil, true, false)
	MnuConfig.AddItem("mnuCfgGitUser", "Git User", InputConfigGitUser, nil, true, false)
	MnuConfig.AddItem("mnuCfgGitPassword", "Git Password", InputConfigGitPassword, nil, true, false)
	MnuConfig.AddItem("mnuCfgConfirmExit", "Confirm Exit", SwitchConfirmExit, nil, true, config.ConfirmExit)
	MnuConfig.AddItem("mnuCfgShowHidden", "Show Hidden", SwitchShowHidden, nil, true, config.ShowHidden)
	MnuConfig.AddItem("mnuCfgFormatTime", "Time Format", InputConfigFormatTime, nil, true, false)
	MnuConfig.AddItem("mnuCfgFormatDate", "Date Format", InputConfigFormatDate, nil, true, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgConfigMenu", MnuConfig.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgConfigMenu")
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
	fmt.Printf("♯%s - %s\n", conf.APP_STRING, conf.APP_URL)
}

// ****************************************************************************
// readSettings()
// ****************************************************************************
func readSettings() {
	// Read MRU list and open them
	ui.SetStatus("Reading MRU list")
	fMRU, err := os.Open(filepath.Join(appDir, conf.FILE_MRU))
	if err == nil {
		defer fMRU.Close()
		sMRU := bufio.NewScanner(fMRU)
		for sMRU.Scan() {
			edit.OpenFile(sMRU.Text())
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
		config.Theme = section.Key("Theme").String()
		config.GitUser = section.Key("GitUser").String()
		config.GitPassword = section.Key("GitPassword").String()
		config.Workspace = section.Key("Workspace").String()
		config.ShowHidden, _ = section.Key("ShowHidden").Bool()
		config.ConfirmExit, _ = section.Key("ConfirmExit").Bool()
		config.FormatTime = section.Key("FormatTime").String()
		config.FormatDate = section.Key("FormatDate").String()
		// Set them
		setTheme(config.Theme)
		if config.FormatTime == "" {
			config.FormatTime = "15:04:05"
		}
		ui.MyConfig.FormatTime = config.FormatTime
		if config.FormatDate == "" {
			config.FormatDate = "02/01/2006"
		}
		ui.MyConfig.FormatDate = config.FormatDate
		if config.Workspace == "" {
			config.Workspace, _ = os.Getwd()
		}
		edit.SwitchOpenFile(section.Key("CurrentFile").String())
		edit.CurrentFile.Buffer.Cursor.X, _ = section.Key("CurrentX").Int()
		edit.CurrentFile.Buffer.Cursor.Y, _ = section.Key("CurrentY").Int()
	}
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
		for _, oFile := range edit.OpenFiles {
			fmt.Fprintln(wMRU, oFile.FName)
		}
		wMRU.Flush()
	}

	// Save INI file
	inidata := ini.Empty()
	sec, _ := inidata.NewSection("general")
	sec.NewKey("Theme", config.Theme)
	sec.NewKey("GitUser", config.GitUser)
	sec.NewKey("GitPassword", config.GitPassword)
	sec.NewKey("Workspace", edit.CurrentWorkspace)
	sec.NewKey("ShowHidden", utils.If(config.ShowHidden, "True", "False"))
	sec.NewKey("ConfirmExit", utils.If(config.ConfirmExit, "True", "False"))
	sec.NewKey("FormatTime", config.FormatTime)
	sec.NewKey("FormatDate", config.FormatDate)
	sec.NewKey("CurrentFile", edit.CurrentFile.FName)
	sec.NewKey("CurrentX", strconv.Itoa(edit.CurrentFile.Buffer.Cursor.X))
	sec.NewKey("CurrentY", strconv.Itoa(edit.CurrentFile.Buffer.Cursor.Y))

	err = inidata.SaveTo(filepath.Join(appDir, conf.FILE_INI))
	if err != nil {
		ui.SetStatus(err.Error())
	}
}

// ****************************************************************************
// ShowQuitDialog()
// ****************************************************************************
func ShowQuitDialog(p any) {
	if config.ConfirmExit {
		ui.PgsApp.SwitchToPage("dlgQuit")
	} else {
		appQuit()
	}
}

// ****************************************************************************
// SwitchHelp()
// ****************************************************************************
func SwitchHelp() {
	if ui.CurrentMode == ui.ModeTextEdit {
		// We are in TextEdit mode, so we want to switch to Help mode (if any)
		idx := ui.GetScreenFromTitle("Help")
		ui.SetStatus(fmt.Sprintf("Help IDX=%s", idx))
		if idx == "NIL" {
			// There is no Help mode yet
			ui.AddNewScreen(ui.ModeHelp, help.SelfInit, nil)
		} else {
			i, _ := strconv.Atoi(idx)
			ui.ShowScreen(i)
		}
	} else {
		// We are in Help mode, so we want to go back to TextEdit mode (if any)
		idx := ui.GetScreenFromTitle("Editor")
		ui.SetStatus(fmt.Sprintf("Editor IDX=%s", idx))
		if idx == "NIL" {
			// There is no TextEdit mode yet
			ui.AddNewScreen(ui.ModeTextEdit, edit.SelfInit, config.Workspace)
		} else {
			i, _ := strconv.Atoi(idx)
			ui.ShowScreen(i)
		}
	}
}

// ****************************************************************************
// InputConfigTheme()
// ****************************************************************************
func InputConfigTheme(f any) {
	MnuInputTheme = MnuInputTheme.New(" Themes ", ui.GetCurrentScreen(), ui.EdtMain)
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
		if thm == config.Theme {
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
	config.Theme = theme.(string)
	ui.SetStatus(fmt.Sprintf("Theme is set to %s", config.Theme))
}

// ****************************************************************************
// InputConfigGitUser()
// ****************************************************************************
func InputConfigGitUser(f any) {
	DlgInputGitUser = DlgInputGitUser.Input("Git User", // Title
		"Please, enter the Git user :", // Message
		config.GitUser,
		setGitUser,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgInputGitUser", DlgInputGitUser.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputGitUser")
}

// ****************************************************************************
// setGitUser()
// ****************************************************************************
func setGitUser(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		config.GitUser = DlgInputGitUser.Value
		ui.SetStatus(fmt.Sprintf("Git User is set to %s", config.GitUser))
	}
}

// ****************************************************************************
// InputConfigGitPassword()
// ****************************************************************************
func InputConfigGitPassword(f any) {
	DlgInputGitPassword = DlgInputGitPassword.Input("Git Password", // Title
		"Please, enter the Git password :", // Message
		config.GitPassword,
		setGitPassword,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgInputGitPassword", DlgInputGitPassword.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputGitPassword")
}

// ****************************************************************************
// setGitPassword()
// ****************************************************************************
func setGitPassword(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		config.GitPassword = DlgInputGitPassword.Value
		ui.SetStatus(fmt.Sprintf("Git Password is set to %s", config.GitPassword))
	}
}

// ****************************************************************************
// InputConfigFormatTime()
// ****************************************************************************
func InputConfigFormatTime(f any) {
	DlgInputFormatTime = DlgInputFormatTime.Input("Time Format", // Title
		"Please, enter the time format :", // Message
		config.FormatTime,
		setFormatTime,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgInputFormatTime", DlgInputFormatTime.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFormatTime")
}

// ****************************************************************************
// setFormatTime()
// ****************************************************************************
func setFormatTime(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		config.FormatTime = DlgInputFormatTime.Value
		ui.SetStatus(fmt.Sprintf("Time Format is set to %s", config.FormatTime))
		ui.MyConfig.FormatTime = config.FormatTime
	}
}

// ****************************************************************************
// InputConfigFormatDate()
// ****************************************************************************
func InputConfigFormatDate(f any) {
	DlgInputFormatDate = DlgInputFormatDate.Input("Date Format", // Title
		"Please, enter the date format :", // Message
		config.FormatDate,
		setFormatDate,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgInputFormatDate", DlgInputFormatDate.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFormatDate")
}

// ****************************************************************************
// setFormatDate()
// ****************************************************************************
func setFormatDate(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		config.FormatDate = DlgInputFormatDate.Value
		ui.SetStatus(fmt.Sprintf("Date Format is set to %s", config.FormatDate))
		ui.MyConfig.FormatDate = config.FormatDate
	}
}

// ****************************************************************************
// InputFileOpen()
// ****************************************************************************
func InputFileOpen(f any) {
	DlgInputFileOpen = DlgInputFileOpen.FileBrowser("Open File", // Title
		edit.CurrentWorkspace,
		doOpenFile,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgInputFileOpen", DlgInputFileOpen.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputFileOpen")
}

// ****************************************************************************
// doOpenFile()
// ****************************************************************************
func doOpenFile(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		fn := DlgInputFileOpen.Value
		ui.SetStatus("Opening " + fn)
	}
}

// ****************************************************************************
// SwitchShowHidden()
// ****************************************************************************
func SwitchShowHidden(dummy any) {
	config.ShowHidden = !config.ShowHidden
	ui.SetStatus(fmt.Sprintf("Show Hidden is set to %t", config.ShowHidden))
	edit.ShowTreeDir(config.Workspace, config.ShowHidden)
}

// ****************************************************************************
// SwitchConfirmExit()
// ****************************************************************************
func SwitchConfirmExit(dummy any) {
	config.ConfirmExit = !config.ConfirmExit
	ui.SetStatus(fmt.Sprintf("Confirm Exit is set to %t", config.ConfirmExit))
}

// ****************************************************************************
// InputShell()
// ****************************************************************************
func InputShell(f any) {
	sh := ""
	DlgInputShell = DlgInputShell.Input("Shell", // Title
		"$> ", // Message
		sh,
		runShell,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgInputShell", DlgInputShell.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputShell")
}

// ****************************************************************************
// runShell()
// ****************************************************************************
func runShell(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		ui.SetStatus(fmt.Sprintf("Running %s", DlgInputShell.Value))
	}
}
