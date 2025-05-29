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
	"path/filepath"
	"strconv"
	"strings"

	"lied/conf"
	"lied/dialog"
	"lied/edit"
	"lied/help"
	"lied/menu"
	"lied/ui"
	"lied/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/go-cmd/cmd"
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
	MnuGit              *menu.Menu
	args                []string
	configGeneral       conf.ConfigGeneral
	configPrivate       conf.ConfigPrivate
	MnuInputTheme       *menu.Menu
	DlgInputGitUser     *dialog.Dialog
	DlgInputGitPassword *dialog.Dialog
	DlgInputFormatTime  *dialog.Dialog
	DlgInputFormatDate  *dialog.Dialog
	DlgInputFileOpen    *dialog.Dialog
	DlgInputShell       *dialog.Dialog
	DlgInputColorAccent *dialog.Dialog
	DlgInput            *dialog.Dialog
	ACmd                []string
	ICmd                int
	MsgBox              *dialog.Dialog
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
	configGeneral.Workspace, _ = os.Getwd()
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
	ui.SetColorAccent(configGeneral.ColorAccent)
}

// ****************************************************************************
// main()
// ****************************************************************************
func main() {
	// Main keyboard's events manager
	ui.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		evkSaveAs := tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt)
		if event.Key() == evkSaveAs.Key() && event.Rune() == evkSaveAs.Rune() && event.Modifiers() == evkSaveAs.Modifiers() {
			edit.SaveFileAs()
			return nil
		}
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
		case tcell.KeyF3:
			ShowGitMenu()
		case tcell.KeyF4:
			InputShell(nil)
		case tcell.KeyF12:
			ShowQuitDialog(nil)
		case tcell.KeyCtrlC:
			edit.CurrentFile.View.Copy()
			return nil
		case tcell.KeyCtrlX:
			if edit.CurrentFile.ReadWrite {
				edit.CurrentFile.View.Cut()
			}
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
			if edit.CurrentFile.ReadWrite {
				edit.CurrentFile.View.Paste()
			}
			return nil
		case tcell.KeyCtrlL:
			if edit.CurrentFile.ReadWrite {
				edit.CurrentFile.View.DeleteLine()
			}
			return nil
		case tcell.KeyCtrlS:
			if edit.CurrentFile.ReadWrite {
				edit.SaveFile()
			}
			return nil
		case tcell.KeyCtrlN:
			edit.NewFile(configGeneral.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(configGeneral.Workspace)
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
			edit.NewFile(configGeneral.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(configGeneral.Workspace)
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
		case tcell.KeyEsc:
			ui.App.SetFocus(ui.TblOpenFiles)
			return nil
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblOpenFiles)
			return nil
		}
		if edit.CurrentFile.ReadWrite == true {
			return event
		} else {
			switch event.Key() {
			case tcell.KeyEnter, tcell.KeyCtrlS, tcell.KeyCtrlV:
				return nil
			}
			switch event.Rune() {
			// there must be an easier way to do this...
			case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
				'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
				'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '&', 'é', '"', '\'', '(', '-', 'è', '_', 'ç', 'à', ')', '=', '+', '°', 'ê', 'ë',
				'~', '#', '{', '[', '|', '`', '\\', '^', '@', ']', '}', '/', '*', '<', '>', ',', ';', ':', '!', '?', '.', '§', 'µ', 'ù', '%':
				return nil
			}
			return event
		}
	})

	// Open Files Panel keyboard's events manager
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

	// Explorer Panel keyboard's events manager
	ui.TrvExplorer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			ui.App.SetFocus(ui.EdtMain)
			return nil
		}
		return event
	})

	edit.ShowTreeDir(configGeneral.Workspace, configGeneral.ShowHidden)

	// * Launching lied without args : Open last workspace and last open files if any, else open a temporary file into the current directory as workspace
	// * Launching lied with directory as argument : Open a temporary file into this directory as workspace
	// * Launching lied with file name as argument : Open this file into its directory as workspace
	if len(args) > 1 {
		edit.NewFileOrLastFile(configGeneral.Workspace)
		fName, _ := filepath.Abs(args[1])
		if utils.IsFileExist(fName) {
			edit.OpenFile(fName, true)
		} else {
			f, e := os.Create(fName)
			if e != nil {
				ui.SetStatus(fmt.Sprintf("Can't create '%s' file", fName))
			} else {
				f.Close()
				edit.OpenFile(fName, true)
			}
		}
	} else {
		edit.NewFileOrLastFile(configGeneral.Workspace)
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
	MnuMain.AddItem("mnuSave", "Save", edit.SaveAnyFile, nil, edit.CurrentFile.ReadWrite, false)
	MnuMain.AddItem("mnuSaveAs", "Save as…", edit.SaveAnyFileAs, nil, true, false)
	MnuMain.AddItem("mnuNew", "New", edit.NewAnyFile, configGeneral.Workspace, true, false)
	MnuMain.AddItem("mnuOpen", "Open…", InputFileOpen, configGeneral.Workspace, true, false)
	MnuMain.AddItem("mnuClose", "Close", edit.CloseAnyFile, nil, true, false)
	MnuMain.AddItem("mnuReadOnly", "Read Only", edit.SwitchReadWrite, nil, true, !edit.CurrentFile.ReadWrite)
	MnuMain.AddItem("mnuFollow", "Follow", edit.SwitchFollow, nil, true, edit.CurrentFile.Follow)
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
	MnuConfig.AddItem("mnuCfgColorAccent", "Color Accent", InputColorAccent, nil, true, false)
	MnuConfig.AddItem("mnuCfgGitUser", "Git User", InputConfigGitUser, nil, true, false)
	MnuConfig.AddItem("mnuCfgGitPassword", "Git Password", InputConfigGitPassword, nil, true, false)
	MnuConfig.AddItem("mnuCfgConfirmExit", "Confirm Exit", SwitchConfirmExit, nil, true, configGeneral.ConfirmExit)
	MnuConfig.AddItem("mnuCfgShowHidden", "Show Hidden", SwitchShowHidden, nil, true, configGeneral.ShowHidden)
	MnuConfig.AddItem("mnuCfgFormatTime", "Time Format", InputConfigFormatTime, nil, true, false)
	MnuConfig.AddItem("mnuCfgFormatDate", "Date Format", InputConfigFormatDate, nil, true, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgConfigMenu", MnuConfig.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgConfigMenu")
}

// ****************************************************************************
// ShowGitMenu()
// ****************************************************************************
func ShowGitMenu() {
	MnuGit = MnuGit.New(" Git Tracking ", ui.GetCurrentScreen(), ui.EdtMain)
	// Menu Options
	MnuGit.AddItem("mnuGitStatus", "Status", DoGitStatus, nil, true, false)
	MnuGit.AddItem("mnuGitCommit", "Commit", DoGitCommit, nil, true, false)
	MnuGit.AddItem("mnuGitPush", "Push", DoGitPush, nil, true, false)
	MnuGit.AddItem("mnuGitCommitPush", "Commit & Push", DoGitCommitPush, nil, true, false)
	MnuGit.AddItem("mnuGitFetch", "Fetch", DoGitFetch, nil, true, false)
	MnuGit.AddItem("mnuGitPull", "Pull (Fetch & Merge)", DoGitPull, nil, true, false)
	MnuGit.AddItem("mnuGitBang", "Initialize (Git Bang)", DoGitBang, nil, true, false)
	MnuGit.AddItem("mnuGitConfigure", "Configure", DoGitConfigure, nil, true, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgGitMenu", MnuGit.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgGitMenu")
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
			rec := sMRU.Text()
			rw := true
			if rec[0] == '0' {
				rw = false
			}
			edit.OpenFile(rec[2:], rw)
		}
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

	// Read INI file
	ui.SetStatus("Reading INI file")
	inidata, err := ini.Load(filepath.Join(appDir, conf.FILE_INI))
	if err != nil {
		ui.SetStatus("No INI file found")
	} else {
		// Read them
		section := inidata.Section("general")
		configGeneral.Theme = section.Key("Theme").String()
		configGeneral.GitUser = section.Key("GitUser").String()
		configGeneral.GitPassword = section.Key("GitPassword").String()
		configGeneral.Workspace = section.Key("Workspace").String()
		configGeneral.ShowHidden, _ = section.Key("ShowHidden").Bool()
		configGeneral.ConfirmExit, _ = section.Key("ConfirmExit").Bool()
		configGeneral.FormatTime = section.Key("FormatTime").String()
		configGeneral.FormatDate = section.Key("FormatDate").String()
		configGeneral.ColorAccent = section.Key("ColorAccent").String()
		// Set them
		setTheme(configGeneral.Theme)
		if configGeneral.FormatTime == "" {
			configGeneral.FormatTime = "15:04:05"
		}
		ui.MyConfig.FormatTime = configGeneral.FormatTime
		if configGeneral.FormatDate == "" {
			configGeneral.FormatDate = "02/01/2006"
		}
		ui.MyConfig.FormatDate = configGeneral.FormatDate
		if configGeneral.Workspace == "" {
			configGeneral.Workspace, _ = os.Getwd()
		}
		edit.SwitchOpenFile(section.Key("CurrentFile").String())
		edit.CurrentFile.Buffer.Cursor.X, _ = section.Key("CurrentX").Int()
		edit.CurrentFile.Buffer.Cursor.Y, _ = section.Key("CurrentY").Int()
	}
	if configGeneral.ColorAccent == "" {
		configGeneral.ColorAccent = conf.DEFAULT_COLOR_ACCENT
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
			rw := "0,"
			if oFile.ReadWrite {
				rw = "1,"
			}
			fmt.Fprintln(wMRU, rw+oFile.FName)
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

	// Save INI file
	inidata := ini.Empty()
	sec, _ := inidata.NewSection("general")
	sec.NewKey("Theme", configGeneral.Theme)
	sec.NewKey("GitUser", configGeneral.GitUser)
	sec.NewKey("GitPassword", configGeneral.GitPassword)
	sec.NewKey("Workspace", edit.CurrentWorkspace)
	sec.NewKey("ShowHidden", utils.If(configGeneral.ShowHidden, "True", "False"))
	sec.NewKey("ConfirmExit", utils.If(configGeneral.ConfirmExit, "True", "False"))
	sec.NewKey("FormatTime", configGeneral.FormatTime)
	sec.NewKey("FormatDate", configGeneral.FormatDate)
	sec.NewKey("CurrentFile", edit.CurrentFile.FName)
	sec.NewKey("CurrentX", strconv.Itoa(edit.CurrentFile.Buffer.Cursor.X))
	sec.NewKey("CurrentY", strconv.Itoa(edit.CurrentFile.Buffer.Cursor.Y))
	sec.NewKey("ColorAccent", configGeneral.ColorAccent)

	err = inidata.SaveTo(filepath.Join(appDir, conf.FILE_INI))
	if err != nil {
		ui.SetStatus(err.Error())
	}
}

// ****************************************************************************
// ShowQuitDialog()
// ****************************************************************************
func ShowQuitDialog(p any) {
	if configGeneral.ConfirmExit {
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
			ui.AddNewScreen(ui.ModeTextEdit, edit.SelfInit, configGeneral.Workspace)
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
		if thm == configGeneral.Theme {
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
	configGeneral.Theme = theme.(string)
	ui.SetStatus(fmt.Sprintf("Theme is set to %s", configGeneral.Theme))
}

// ****************************************************************************
// InputColorAccent()
// ****************************************************************************
func InputColorAccent(f any) {
	DlgInputColorAccent = DlgInputColorAccent.Input("Color Accent", // Title
		"Please, enter the color accent :", // Message
		configGeneral.ColorAccent,
		setColorAccent,
		0,
		ui.GetCurrentScreen(), ui.EdtMain) // Focus return
	ui.PgsApp.AddPage("dlgInputColorAccent", DlgInputColorAccent.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputColorAccent")
}

// ****************************************************************************
// setColorAccent()
// ****************************************************************************
func setColorAccent(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		configGeneral.ColorAccent = DlgInputColorAccent.Value
		ui.SetColorAccent(configGeneral.ColorAccent)
		ui.SetStatus(fmt.Sprintf("Color accent is set to %s", configGeneral.ColorAccent))
	}
}

// ****************************************************************************
// InputConfigGitUser()
// ****************************************************************************
func InputConfigGitUser(f any) {
	DlgInputGitUser = DlgInputGitUser.Input("Git User", // Title
		"Please, enter the Git user :", // Message
		configGeneral.GitUser,
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
		configGeneral.GitUser = DlgInputGitUser.Value
		ui.SetStatus(fmt.Sprintf("Git User is set to %s", configGeneral.GitUser))
	}
}

// ****************************************************************************
// InputConfigGitPassword()
// ****************************************************************************
func InputConfigGitPassword(f any) {
	DlgInputGitPassword = DlgInputGitPassword.Input("Git Password", // Title
		"Please, enter the Git password :", // Message
		configGeneral.GitPassword,
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
		configGeneral.GitPassword = DlgInputGitPassword.Value
		ui.SetStatus(fmt.Sprintf("Git Password is set to %s", configGeneral.GitPassword))
	}
}

// ****************************************************************************
// InputConfigFormatTime()
// ****************************************************************************
func InputConfigFormatTime(f any) {
	DlgInputFormatTime = DlgInputFormatTime.Input("Time Format", // Title
		"Please, enter the time format :", // Message
		configGeneral.FormatTime,
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
		configGeneral.FormatTime = DlgInputFormatTime.Value
		ui.SetStatus(fmt.Sprintf("Time Format is set to %s", configGeneral.FormatTime))
		ui.MyConfig.FormatTime = configGeneral.FormatTime
	}
}

// ****************************************************************************
// InputConfigFormatDate()
// ****************************************************************************
func InputConfigFormatDate(f any) {
	DlgInputFormatDate = DlgInputFormatDate.Input("Date Format", // Title
		"Please, enter the date format :", // Message
		configGeneral.FormatDate,
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
		configGeneral.FormatDate = DlgInputFormatDate.Value
		ui.SetStatus(fmt.Sprintf("Date Format is set to %s", configGeneral.FormatDate))
		ui.MyConfig.FormatDate = configGeneral.FormatDate
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
	ui.App.SetFocus(&DlgInputFileOpen.UIList)
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
	configGeneral.ShowHidden = !configGeneral.ShowHidden
	ui.SetStatus(fmt.Sprintf("Show Hidden is set to %t", configGeneral.ShowHidden))
	edit.ShowTreeDir(configGeneral.Workspace, configGeneral.ShowHidden)
}

// ****************************************************************************
// SwitchConfirmExit()
// ****************************************************************************
func SwitchConfirmExit(dummy any) {
	configGeneral.ConfirmExit = !configGeneral.ConfirmExit
	ui.SetStatus(fmt.Sprintf("Confirm Exit is set to %t", configGeneral.ConfirmExit))
}

// ****************************************************************************
// InputShell()
// ****************************************************************************
func InputShell(f any) {
	sh := ""
	DlgInputShell = DlgInputShell.Command("Shell", // Title
		"CWD:"+edit.CurrentWorkspace,
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
		Xeq(DlgInputShell.Value)
	}
}

// ****************************************************************************
// Xeq()
// ****************************************************************************
func Xeq(c string) {
	sCmd := strings.Fields(c)
	ACmd = append(ACmd, c)
	ICmd++

	ui.SetStatus(fmt.Sprintf("Running [%s]", c))
	if sCmd[0][0] == '!' {
		xCmd := sCmd[0] + "     "
		xCmd = xCmd[:5]
		xCmd = strings.TrimSpace(xCmd)
		switch xCmd {
		case "!quit", "!exit", "!bye":
			ui.PgsApp.SwitchToPage("dlgQuit")
		case "!log":
			edit.OpenFile(filepath.Join(appDir, conf.FILE_LOG), false)
		case "!out":
			edit.OpenFile(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), false)
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
		default:
			ui.SetStatus(fmt.Sprintf("Invalid command %s", sCmd[0]))
		}
	} else {
		cmdOptions := cmd.Options{
			Buffered:  false,
			Streaming: true,
		}

		xCmd := cmd.NewCmdOptions(cmdOptions, sCmd[0], sCmd[1:]...)
		xCmd.Dir = edit.CurrentWorkspace
		fOut, _ := os.OpenFile(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer fOut.Close()
		wOut := bufio.NewWriter(fOut)
		fmt.Fprintln(wOut, "> "+c+"\n")
		wOut.Flush()
		doneChan := make(chan struct{})
		go func() {
			defer close(doneChan)
			// Done when both channels have been closed
			// https://dave.cheney.net/2013/04/30/curious-channels
			for xCmd.Stdout != nil || xCmd.Stderr != nil {
				select {
				case line, open := <-xCmd.Stdout:
					if !open {
						xCmd.Stdout = nil
						continue
					}
					wOut := bufio.NewWriter(fOut)
					fmt.Fprintln(wOut, line)
					wOut.Flush()
					ui.SetStatus(line)
					ui.App.ForceDraw()
				case line, open := <-xCmd.Stderr:
					if !open {
						xCmd.Stderr = nil
						continue
					}
					wOut := bufio.NewWriter(fOut)
					fmt.Fprintln(wOut, line)
					wOut.Flush()
					ui.SetStatus(line)
					ui.App.ForceDraw()
				}
			}
			// conf.Cwd = getWorkingDirectoryOfPID(xCmd.Status().PID)
		}()

		// Run and wait for Cmd to return
		<-xCmd.Start()

		// Wait for goroutine to print everything
		<-doneChan

		// Job's done !
		fmt.Fprintln(wOut, "\n")
		wOut.Flush()
		ui.SetStatus(fmt.Sprintf("Done [%s]", c))
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

	cmd := exec.Command(sCmd[0], sCmd[1:]...)
	cmd.Dir = edit.CurrentWorkspace
	ui.SetStatus(fmt.Sprintf("Executing [%s] in %s", c, cmd.Dir))
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	out := ""
	err := cmd.Run()
	if err != nil {
		out = "Error : " + err.Error()
	} else {
		out = outb.String()
		out = out + errb.String()
	}
	ui.SetStatus(out)
	ui.SetStatus(fmt.Sprintf("Done [%s]", c))
	return strings.TrimSpace(out)
}

// ****************************************************************************
// DoGitStatus()
// ****************************************************************************
func DoGitStatus(f any) {
	if IsInsideGitWorkTree() {
		out := fmt.Sprintf("Current local commit : %s\n", XeqOut("git rev-parse --short HEAD"))
		out += fmt.Sprintf("%s\n", XeqOut("git status"))
		out += fmt.Sprintf("%s\n", XeqOut("git diff"))
		MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
		ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
		ui.PgsApp.ShowPage("msgBox")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitBang()
// ****************************************************************************
func DoGitBang(f any) {
	out := fmt.Sprintf("Current commit : %s\n", XeqOut("git rev-parse --short HEAD"))
	out += fmt.Sprintf("%s\n", XeqOut("git status"))
	out += fmt.Sprintf("%s\n", XeqOut("git diff"))
	// out := XeqOut("git rev-parse --short HEAD")
	MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// DoGitCommitPush()
// ****************************************************************************
func DoGitCommitPush(f any) {
	out := fmt.Sprintf("Current commit : %s\n", XeqOut("git rev-parse --short HEAD"))
	out += fmt.Sprintf("%s\n", XeqOut("git status"))
	out += fmt.Sprintf("%s\n", XeqOut("git diff"))
	// out := XeqOut("git rev-parse --short HEAD")
	MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// DoGitConfigure()
// ****************************************************************************
func DoGitConfigure(f any) {
	out := fmt.Sprintf("Current commit : %s\n", XeqOut("git rev-parse --short HEAD"))
	out += fmt.Sprintf("%s\n", XeqOut("git status"))
	out += fmt.Sprintf("%s\n", XeqOut("git diff"))
	// out := XeqOut("git rev-parse --short HEAD")
	MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// DoGitFetch()
// ****************************************************************************
func DoGitFetch(f any) {
	out := fmt.Sprintf("Current commit : %s\n", XeqOut("git rev-parse --short HEAD"))
	out += fmt.Sprintf("%s\n", XeqOut("git status"))
	out += fmt.Sprintf("%s\n", XeqOut("git diff"))
	// out := XeqOut("git rev-parse --short HEAD")
	MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// DoGitPull()
// ****************************************************************************
func DoGitPull(f any) {
	out := fmt.Sprintf("Current commit : %s\n", XeqOut("git rev-parse --short HEAD"))
	out += fmt.Sprintf("%s\n", XeqOut("git status"))
	out += fmt.Sprintf("%s\n", XeqOut("git diff"))
	// out := XeqOut("git rev-parse --short HEAD")
	MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// DoGitPush()
// ****************************************************************************
func DoGitPush(f any) {
	if IsInsideGitWorkTree() {
		branch := XeqOut("git rev-parse --abbrev-ref HEAD")
		out := fmt.Sprintf("Pushing...\n%s", XeqOut("git push origin "+branch))
		MsgBox = MsgBox.OK("Git Push", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
		ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
		ui.PgsApp.ShowPage("msgBox")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitCommit()
// ****************************************************************************
func DoGitCommit(f any) {
	if IsInsideGitWorkTree() {
		DlgInput = DlgInput.Input("Git Commit", // Title
			"Please, enter the message :", // Message
			"",                            // Default value
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_OK {
					out := fmt.Sprintf("Committing...\n%s", XeqOut("git commit -a -m \""+DlgInput.Value+"\""))
					MsgBox = MsgBox.OK("Git Commit", out, nil, 0, ui.GetCurrentScreen(), ui.EdtMain)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				}
			},
			0,
			ui.GetCurrentScreen(), ui.EdtMain) // Focus return
		ui.PgsApp.AddPage("dlgInput", DlgInput.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgInput")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// IsInsideGitWorkTree()
// ****************************************************************************
func IsInsideGitWorkTree() bool {
	rc := false
	out := XeqOut("git rev-parse --is-inside-work-tree")
	if out == "true" {
		rc = true
	}
	return rc
}
