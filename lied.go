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
	"slices"
	"sort"
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

	"github.com/gdamore/tcell/v2"
	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
	"github.com/rivo/tview"
	"gopkg.in/ini.v1"
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	appDir       string
	hostname     string
	greeting     string
	err          error
	MnuMacros    *menu.Menu
	MnuConfig    *menu.Menu
	MnuGit       *menu.Menu
	MnuWorkspace *menu.Menu
	MnuLicenses  *menu.Menu
	MnuTemplates *menu.Menu
	args         []string
	// conf.ConfigGeneral       conf.SConfigGeneral
	// configPrivate       conf.SConfigPrivate
	MnuInputTheme       *menu.Menu
	DlgInputGitUser     *dialog.Dialog
	DlgInputGitPassword *dialog.Dialog
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
	ACmd                []string
	ICmd                int
	MsgBox              *dialog.Dialog
	Macros              map[string]string
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
		switch event.Key() {
		case tcell.KeyF1:
			ShowHelp()
		case tcell.KeyF8:
			ShowConfigMenu()
		case tcell.KeyF6:
			edit.SwitchPreviousFile()
		case tcell.KeyF7:
			edit.SwitchNextFile()
		case tcell.KeyF9:
			ShowWorkspaceMenu()
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
			edit.NewFile(conf.ConfigGeneral.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(conf.ConfigGeneral.Workspace)
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
		// ALT+S
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
			ui.App.SetFocus(ui.FrmFind)
			return nil
		case tcell.KeyEnter:
			idx, _ := ui.TblOpenFiles.GetSelection()
			fName := ui.TblOpenFiles.GetCell(idx, 3).Text
			edit.SwitchOpenFile(fName)
			edit.SetFocusOnPath(fName)
			if edit.CurrentFile.IsBinary {
				ui.App.SetFocus(ui.HexView)
			} else {
				ui.App.SetFocus(ui.EdtMain)
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
		}
		return event
	})

	// Outline Panel keyboard's events manager
	ui.TblOutline.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF2:
			if edit.CurrentFile.IsBinary {
				ui.App.SetFocus(ui.HexView)
			} else {
				ui.App.SetFocus(ui.EdtMain)
			}
			return nil
		case tcell.KeyEnter:
			if !edit.CurrentFile.IsBinary {
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
		switch event.Key() {
		case tcell.KeyCtrlN:
			edit.NewFile(conf.ConfigGeneral.Workspace)
			return nil
		case tcell.KeyCtrlO:
			InputFileOpen(conf.ConfigGeneral.Workspace)
			return nil
		case tcell.KeyCtrlT:
			edit.CloseCurrentFile()
			return nil
		case tcell.KeyF2:
			ui.App.SetFocus(ui.TblOpenFiles)
			return nil
		case tcell.KeyCtrlF:
			ui.FrmFind.GetButton(0).SetSelectedFunc(edit.FindNext)
			ui.FrmFind.GetButton(1).SetSelectedFunc(edit.FindPrevious)
			ui.FrmFind.GetButton(2).SetDisabled(true) // Disable Replace for binary
			ui.FrmFind.GetButton(3).SetDisabled(true) // Disable Replace All for binary
			ui.App.SetFocus(ui.FrmFind)
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
			edit.OpenFile(fName, true)
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
				edit.OpenFile(fName, true)
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

	if err := ui.App.SetRoot(ui.PgsApp, true).SetFocus(edit.CurrentView).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// ****************************************************************************
// ShowMainMenu()
// ****************************************************************************
func ShowMainMenu() {
	MnuMacros = MnuMacros.New(" "+conf.APP_NAME+" ", ui.GetCurrentScreen(), edit.CurrentView)
	// Dynamic options (files currently open)
	for i, e := range edit.OpenFiles {
		chk := false
		if e.FName == edit.CurrentFile.FName {
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
	MnuMacros.AddItem("mnuSave", "Save", edit.SaveAnyFile, nil, edit.CurrentFile.ReadWrite, false)
	MnuMacros.AddItem("mnuSaveAs", "Save as…", edit.SaveAnyFileAs, nil, true, false)
	MnuMacros.AddItem("mnuNew", "New", edit.NewAnyFile, conf.ConfigGeneral.Workspace, true, false)
	MnuMacros.AddItem("mnuOpen", "Open File…", InputFileOpen, conf.ConfigGeneral.Workspace, true, false)
	MnuMacros.AddItem("mnuClose", "Close", edit.CloseAnyFile, nil, true, false)
	MnuMacros.AddItem("mnuReadOnly", "Read Only", edit.SwitchReadWrite, nil, true, !edit.CurrentFile.ReadWrite)
	MnuMacros.AddItem("mnuFollow", "Follow", edit.SwitchFollow, nil, true, edit.CurrentFile.Follow)
	MnuMacros.AddSeparator()
	MnuMacros.AddItem("mnuGitAdd", "Git add…", DoGitAdd, edit.CurrentFile.FName, !IsFileGitTracked(edit.CurrentFile.FName), false)
	MnuMacros.AddItem("mnuArchive", "Archive", DoArchive, conf.ConfigGeneral.Workspace, true, false)
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
	MnuConfig = MnuConfig.New(" Settings ", ui.GetCurrentScreen(), edit.CurrentView)
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
	// Popup menu
	ui.PgsApp.AddPage("dlgConfigMenu", MnuConfig.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgConfigMenu")
}

// ****************************************************************************
// ShowGitMenu()
// ****************************************************************************
func ShowGitMenu() {
	MnuGit = MnuGit.New(" Git Tracking ", ui.GetCurrentScreen(), edit.CurrentView)
	// Menu Options
	MnuGit.AddItem("mnuGitStatus", "Status", DoGitStatus, nil, true, false)
	MnuGit.AddItem("mnuGitLog", "Log", DoGitLog, nil, true, false)
	MnuGit.AddItem("mnuGitAddAll", "Add All (.)", DoGitAddAll, nil, true, false)
	MnuGit.AddItem("mnuGitCommit", "Commit", DoGitCommit, nil, true, false)
	MnuGit.AddItem("mnuGitPush", "Push", DoGitPush, nil, true, false)
	MnuGit.AddItem("mnuGitCommitPush", "Commit & Push", DoGitCommitPush, nil, true, false)
	MnuGit.AddItem("mnuGitFetch", "Fetch", DoGitFetch, nil, true, false)
	MnuGit.AddItem("mnuGitPull", "Pull (Fetch & Merge)", DoGitPull, nil, true, false)
	MnuGit.AddItem("mnuGitInit", "Initialize", DoGitInit, nil, true, false)
	MnuGit.AddItem("mnuGitBang", "Initialize & Push", DoGitBang, nil, true, false)
	MnuGit.AddItem("mnuGitClone", "Clone", DoGitClone, nil, true, false)
	MnuGit.AddItem("mnuGitConfigure", "Configure", DoGitConfigure, nil, true, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgGitMenu", MnuGit.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgGitMenu")
}

// ****************************************************************************
// ShowWorkspaceMenu()
// ****************************************************************************
func ShowWorkspaceMenu() {
	ui.SetStatus(fmt.Sprintf("Current Workspace is %s", conf.ConfigGeneral.Workspace))
	MnuWorkspace = MnuWorkspace.New(" Workspace ", ui.GetCurrentScreen(), edit.CurrentView)
	// Menu Options
	MnuWorkspace.AddItem("mnuOpen", "Open Workspace", InputWorkspaceOpen, conf.ConfigGeneral.Workspace, true, false) // OK
	MnuWorkspace.AddItem("mnuSaveAll", "Save all", doSaveAll, conf.ConfigGeneral.Workspace, true, false)             // OK
	// MnuWorkspace.AddItem("mnuClose", "Close", InputWorkspaceOpen, conf.ConfigGeneral.Workspace, true, false)          // Not yet
	MnuWorkspace.AddItem("mnuRename", "Rename file or folder", InputRename, conf.ConfigGeneral.Workspace, true, false) // OK
	// MnuWorkspace.AddItem("mnuNewFile", "New file", doNewFile, conf.ConfigGeneral.Workspace, true, false)
	MnuWorkspace.AddItem("mnuAddFileTemplate", "New file", ShowTemplatesMenu, conf.ConfigGeneral.Workspace, true, false)        // OK
	MnuWorkspace.AddItem("mnuNewFolder", "New folder", doNewFolder, conf.ConfigGeneral.Workspace, true, false)                  // OK
	MnuWorkspace.AddItem("mnuAddLicense", "Add license", ShowLicensesMenu, conf.ConfigGeneral.Workspace, true, false)           // OK
	MnuWorkspace.AddItem("mnuDelete", "Delete file or folder", InputWorkspaceDelete, conf.ConfigGeneral.Workspace, true, false) // OK
	// Popup menu
	ui.PgsApp.AddPage("dlgWorkspaceMenu", MnuWorkspace.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgWorkspaceMenu")
}

// ****************************************************************************
// ShowLicensesMenu()
// ****************************************************************************
func ShowLicensesMenu(f any) {
	// Read the directory entry for the "licenses" embedded folder
	entries, err := conf.LicensesFS.ReadDir("licenses")
	ui.SetStatus("Reading licences")
	if err == nil {
		MnuLicenses = MnuLicenses.New(" Licenses ", ui.GetCurrentScreen(), edit.CurrentView)
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
		MnuTemplates = MnuTemplates.New(" Templates ", ui.GetCurrentScreen(), edit.CurrentView)
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
				templateFileName := d
				ui.SetStatus(fmt.Sprintf("Adding template %s to the current workspace", templateFileName))
				sourceFileName := filepath.Join("templates", templateFileName)
				destFileName := filepath.Join(conf.ConfigGeneral.Workspace, DlgNewFile.Value)
				fileContent, err := conf.TemplatesFS.ReadFile(sourceFileName)
				if err != nil {
					ui.SetStatus(fmt.Sprintf("Error reading file: %v", err))
				}
				if err := os.WriteFile(destFileName, fileContent, 0644); err != nil { // nolint: gosec
					ui.SetStatus(fmt.Sprintf("Error writing file: %w", err))
				}
				edit.OpenFile(destFileName, true)
				edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
			} else {
				ui.SetStatus("Canceling creating new file")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
	ui.PgsApp.AddPage("dlgNewFile", DlgNewFile.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNewFile")
}

// ****************************************************************************
// ShowMacrosMenu()
// ****************************************************************************
func ShowMacrosMenu() {
	ReadMacros()
	MnuMacros = MnuMacros.New(" Macros ", ui.GetCurrentScreen(), edit.CurrentView)
	// Sort macros
	keys := make([]string, 0, len(Macros))
	for k := range Macros {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Read macros
	for _, k := range keys {
		MnuMacros.AddItem(k,
			k,
			XeqMacro,
			k,
			true,
			false)
	}
	// Fixed options
	if len(Macros) == 0 {
		MnuMacros.AddItem("mnuEmpty", "Empty", nil, nil, false, false)
	}
	MnuMacros.AddSeparator()
	MnuMacros.AddItem("mnuEditMacros", "Edit", editMacrosFile, nil, edit.CurrentFile.ReadWrite, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgMacrosMenu", MnuMacros.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgMacrosMenu")
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
		conf.ConfigGeneral.Workspace = section.Key("Workspace").String()
		conf.ConfigGeneral.ShowHidden, _ = section.Key("ShowHidden").Bool()
		conf.ConfigGeneral.ConfirmExit, _ = section.Key("ConfirmExit").Bool()
		conf.ConfigGeneral.CleanUpOnExit, _ = section.Key("CleanUpOnExit").Bool()
		conf.ConfigGeneral.FormatTime = section.Key("FormatTime").String()
		conf.ConfigGeneral.FormatDate = section.Key("FormatDate").String()
		conf.ConfigGeneral.ColorAccent = section.Key("ColorAccent").String()
		// Set them
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
		edit.SwitchOpenFile(section.Key("CurrentFile").String())
		if edit.CurrentFile.Buffer != nil {
			edit.CurrentFile.Buffer.Cursor.X, _ = section.Key("CurrentX").Int()
			edit.CurrentFile.Buffer.Cursor.Y, _ = section.Key("CurrentY").Int()
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
		for _, oFile := range edit.OpenFiles {
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
	sec.NewKey("Workspace", conf.ConfigGeneral.Workspace)
	sec.NewKey("ShowHidden", utils.If(conf.ConfigGeneral.ShowHidden, "True", "False"))
	sec.NewKey("ConfirmExit", utils.If(conf.ConfigGeneral.ConfirmExit, "True", "False"))
	sec.NewKey("CleanUpOnExit", utils.If(conf.ConfigGeneral.CleanUpOnExit, "True", "False"))
	sec.NewKey("FormatTime", conf.ConfigGeneral.FormatTime)
	sec.NewKey("FormatDate", conf.ConfigGeneral.FormatDate)
	sec.NewKey("CurrentFile", edit.CurrentFile.FName)
	sec.NewKey("CurrentX", strconv.Itoa(edit.CurrentFile.Buffer.Cursor.X))
	sec.NewKey("CurrentY", strconv.Itoa(edit.CurrentFile.Buffer.Cursor.Y))
	sec.NewKey("ColorAccent", conf.ConfigGeneral.ColorAccent)

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
	MnuInputTheme = MnuInputTheme.New(" Themes ", ui.GetCurrentScreen(), edit.CurrentView)
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
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
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
// InputConfigGitUser()
// ****************************************************************************
func InputConfigGitUser(f any) {
	DlgInputGitUser = DlgInputGitUser.Input("Git User", // Title
		"Please, enter the Git user :", // Message
		conf.ConfigGeneral.GitUser,
		setGitUser,
		0,
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
	ui.PgsApp.AddPage("dlgInputGitUser", DlgInputGitUser.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputGitUser")
}

// ****************************************************************************
// setGitUser()
// ****************************************************************************
func setGitUser(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		conf.ConfigGeneral.GitUser = DlgInputGitUser.Value
		ui.SetStatus(fmt.Sprintf("Git User is set to %s", conf.ConfigGeneral.GitUser))
	}
}

// ****************************************************************************
// InputConfigGitPassword()
// ****************************************************************************
func InputConfigGitPassword(f any) {
	DlgInputGitPassword = DlgInputGitPassword.Input("Git Password", // Title
		"Please, enter the Git password :", // Message
		conf.ConfigGeneral.GitKey,
		setGitPassword,
		0,
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
	ui.PgsApp.AddPage("dlgInputGitPassword", DlgInputGitPassword.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgInputGitPassword")
}

// ****************************************************************************
// setGitPassword()
// ****************************************************************************
func setGitPassword(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		conf.ConfigGeneral.GitKey = DlgInputGitPassword.Value
		ui.SetStatus(fmt.Sprintf("Git Password is set to %s", conf.ConfigGeneral.GitKey))
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
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
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
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
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
		ui.GetCurrentScreen(), edit.CurrentView, false) // Focus return
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
		ui.GetCurrentScreen(), edit.CurrentView, false) // Focus return
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
		ui.GetCurrentScreen(), edit.CurrentView, true) // Focus return
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
		ui.GetCurrentScreen(), edit.CurrentView, // Focus return
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
		edit.OpenFile(fn, utils.IsTextFile(fn))
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
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
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
				edit.CreateThisFile(filepath.Join(d, DlgNewFile.Value))
			} else {
				ui.SetStatus("Canceling creating new file")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
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
				err := os.Mkdir(filepath.Join(d, DlgNewFolder.Value), os.ModePerm)
				if err != nil {
					ui.SetStatus(err.Error())
				} else {
					ui.SetStatus(fmt.Sprintf("Folder %s created", filepath.Join(d, DlgNewFolder.Value)))
				}
			} else {
				ui.SetStatus("Canceling creating new folder")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentView) // Focus return
	ui.PgsApp.AddPage("dlgNewFolder", DlgNewFolder.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNewFolder")
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
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
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
// InputShell()
// ****************************************************************************
func InputShell(f any) {
	sh := ""
	DlgInputShell = DlgInputShell.Command("Shell", // Title
		"CWD:"+conf.ConfigGeneral.Workspace,
		sh,
		runShell,
		0,
		ui.GetCurrentScreen(), edit.CurrentView, ACmd) // Focus return
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
			case "!conf":
				edit.OpenFile(filepath.Join(appDir, conf.FILE_INI), false)
			case "!macr":
				edit.OpenFile(filepath.Join(appDir, conf.FILE_MACROS), false)
			case "!help":
				ShowHelp()
			case "!info":
				ShowSysInfo()
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
			cmdOptions := cmd.Options{
				Buffered:  false,
				Streaming: true,
			}

			xCmd := cmd.NewCmdOptions(cmdOptions, sCmd[0], sCmd[1:]...)
			xCmd.Dir = conf.ConfigGeneral.Workspace
			fOut, _ := os.OpenFile(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer fOut.Close()
			wOut := bufio.NewWriter(fOut)
			fmt.Fprintln(wOut, time.Now().Format("20060102-150405")+" ⯈ "+c+"\n")
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
// DoGitStatus()
// ****************************************************************************
func DoGitStatus(f any) {
	if IsInsideGitWorkTree() {
		out := fmt.Sprintf("Current local commit : %s\n\n", XeqOut("git rev-parse --short HEAD"))
		out += fmt.Sprintf("%s\n", XeqOut("git status"))
		out += fmt.Sprintf("%s\n", XeqOut("git diff"))
		MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
		ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
		ui.PgsApp.ShowPage("msgBox")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitLog()
// ****************************************************************************
func DoGitLog(f any) {
	if IsInsideGitWorkTree() {
		out := fmt.Sprintf("%s", XeqOut("git log"))
		MsgBox = MsgBox.OK("Git Log", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
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
	if !IsInsideGitWorkTree() {
		pjname := filepath.Base(filepath.Dir(edit.CurrentFile.FName))
		DlgYesNo1 = DlgYesNo1.YesNo("Git Bang", // Title
			fmt.Sprintf("This will initialize and push a Git environment\nfor your project \"%s\".\n\nAre you sure you want to proceed ?", pjname), // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					if conf.ConfigGeneral.GitUser == "" {
						ui.SetStatus("Git User is not yet configured")
					} else {
						DlgYesNo2 = DlgYesNo2.YesNo("Git Bang", // Title
							"The following repository should already exist :\n\nhttps://github.com/"+conf.ConfigGeneral.GitUser+"/"+pjname+"\n\nHas the repository already been created ?", // Message
							func(rc dialog.DlgButton, idx int) {
								if rc == dialog.BUTTON_YES {
									url := "https://" + conf.ConfigGeneral.GitKey + "@github.com/" + conf.ConfigGeneral.GitUser + "/" + pjname
									Xeq("git init")
									Xeq("git add .")
									Xeq("git commit -m \"First Commit\"")
									Xeq("git remote add origin " + url)
									Xeq("git branch -M main")
									Xeq("git pull --rebase origin main")
									Xeq("git push -u origin main")
									currentDir := filepath.Dir(edit.CurrentFile.FName)
									if !utils.IsFileExist(filepath.Join(currentDir, "README.md")) {
										createFile(filepath.Join(currentDir, "README.md"), "#"+pjname)
										Xeq("git add README.md")
									}
									if !utils.IsFileExist(filepath.Join(currentDir, ".gitignore")) {
										createFile(filepath.Join(currentDir, ".gitignore"),
											`# This file is used to ignore files which are generated
# ----------------------------------------------------------------------------

github.key
*~
*.autosave
*.a
*.core
*.moc
*.o
*.obj
*.orig
*.rej
*.so
*.so.*
*_pch.h.cpp
*_resource.rc
*.qm
.#*
*.*#
core
!core/
tags
.DS_Store
.directory
*.debug
Makefile*
*.prl
*.app
moc_*.cpp
ui_*.h
qrc_*.cpp
Thumbs.db
*.res
*.rc
/.qmake.cache
/.qmake.stash

# qtcreator generated files
*.pro.user*
*.qbs.user*
CMakeLists.txt.user*

# xemacs temporary files
*.flc

# Vim temporary files
.*.swp

# Visual Studio generated files
*.ib_pdb_index
*.idb
*.ilk
*.pdb
*.sln
*.suo
*.vcproj
*vcproj.*.*.user
*.ncb
*.sdf
*.opensdf
*.vcxproj
*vcxproj.*

# MinGW generated files
*.Debug
*.Release

# Python byte code
*.pyc

# Binaries
# --------
*.dll
*.exe

# Directories with generated files
.moc/
.obj/
.pch/
.rcc/
.uic/
/build*/
`)
										Xeq("git add .gitignore")
									}
									DoGitStatus("dummy")
								} else {
									ui.SetStatus("Aborting Git Bang")
								}
							},
							0,
							ui.GetCurrentScreen(), edit.CurrentView) // Focus return
						ui.PgsApp.AddPage("dlgYesNo2", DlgYesNo2.Popup(), true, false)
						ui.PgsApp.ShowPage("dlgYesNo2")
					}
				} else {
					ui.SetStatus("Aborting Git Bang")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgYesNo1", DlgYesNo1.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo1")
	} else {
		ui.SetStatus("Git repository already created")
	}
}

// ****************************************************************************
// DoGitInit()
// ****************************************************************************
func DoGitInit(f any) {
	if !IsInsideGitWorkTree() {
		pjname := filepath.Base(filepath.Dir(edit.CurrentFile.FName))
		DlgYesNo1 = DlgYesNo1.YesNo("Git Init", // Title
			fmt.Sprintf("This will initialize a Git environment\nfor your project \"%s\".\n\nAre you sure you want to proceed ?", pjname), // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					if conf.ConfigGeneral.GitUser == "" {
						ui.SetStatus("Git User is not yet configured")
					} else {
						currentDir := filepath.Dir(edit.CurrentFile.FName)
						if !utils.IsFileExist(filepath.Join(currentDir, "README.md")) {
							createFile(filepath.Join(currentDir, "README.md"), "#"+pjname)
						}
						if !utils.IsFileExist(filepath.Join(currentDir, ".gitignore")) {
							createFile(filepath.Join(currentDir, ".gitignore"),
								`# This file is used to ignore files which are generated
# ----------------------------------------------------------------------------

github.key
*~
*.autosave
*.a
*.core
*.moc
*.o
*.obj
*.orig
*.rej
*.so
*.so.*
*_pch.h.cpp
*_resource.rc
*.qm
.#*
*.*#
core
!core/
tags
.DS_Store
.directory
*.debug
Makefile*
*.prl
*.app
moc_*.cpp
ui_*.h
qrc_*.cpp
Thumbs.db
*.res
*.rc
/.qmake.cache
/.qmake.stash

# qtcreator generated files
*.pro.user*
*.qbs.user*
CMakeLists.txt.user*

# xemacs temporary files
*.flc

# Vim temporary files
.*.swp

# Visual Studio generated files
*.ib_pdb_index
*.idb
*.ilk
*.pdb
*.sln
*.suo
*.vcproj
*vcproj.*.*.user
*.ncb
*.sdf
*.opensdf
*.vcxproj
*vcxproj.*

# MinGW generated files
*.Debug
*.Release

# Python byte code
*.pyc

# Binaries
# --------
*.dll
*.exe

# Directories with generated files
.moc/
.obj/
.pch/
.rcc/
.uic/
/build*/
`)
						}
						Xeq("git init")
						// Xeq("git add .")
						DoGitStatus("dummy")
						Xeq("git branch -M main")
						go edit.UpdateStatus()
						edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
					}
				} else {
					ui.SetStatus("Aborting Git Init")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgYesNo1", DlgYesNo1.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo1")
	} else {
		ui.SetStatus("Git repository already created")
	}
}

// ****************************************************************************
// DoGitCommitPush()
// ****************************************************************************
func DoGitCommitPush(f any) {
	if IsInsideGitWorkTree() {
		DlgInput = DlgInput.Input("Git Commit & Push", // Title
			"Please, enter the message :", // Message
			"",                            // Default value
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_OK {
					out := fmt.Sprintf("Committing...\n%s", XeqOut("git commit -a -m \""+DlgInput.Value+"\""))
					branch := XeqRaw("git rev-parse --abbrev-ref HEAD")
					out += fmt.Sprintf("\n\nPushing...\n%s", XeqOut("git push origin "+branch))

					MsgBox = MsgBox.OK("Git Commit & Push", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Commit & Push")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgInput", DlgInput.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgInput")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitConfigure()
// ****************************************************************************
func DoGitConfigure(f any) {
	form := tview.NewForm()
	form.SetTitle("Git Configure")
	form.AddInputField("User", conf.ConfigGeneral.GitUser, 40, nil, nil)
	form.AddInputField("Key", conf.ConfigGeneral.GitKey, 40, nil, nil)
	form.AddButton("OK", func() {
		conf.ConfigGeneral.GitUser = form.GetFormItem(0).(*tview.InputField).GetText()
		conf.ConfigGeneral.GitKey = form.GetFormItem(1).(*tview.InputField).GetText()
		ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
		ui.App.SetFocus(edit.CurrentView)
	})
	form.AddButton("Cancel", func() {
		ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
		ui.App.SetFocus(ui.EdtMain)
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
			ui.App.SetFocus(edit.CurrentView)
			return nil
		}
		return event
	})

	form.SetButtonsAlign(tview.AlignCenter)
	form.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	form.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	form.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	form.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)

	ui.PgsApp.AddPage("myForm",
		tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(form, 9, 1, true).
				AddItem(nil, 0, 1, false), 49, 1, true).
			AddItem(nil, 0, 1, false),
		true, false)
	ui.PgsApp.ShowPage("myForm")
}

// ****************************************************************************
// DoGitFetch()
// ****************************************************************************
func DoGitFetch(f any) {
	if IsInsideGitWorkTree() {
		DlgYesNo = DlgYesNo.YesNo("Git Fetch", // Title
			"The Git Fetch will fetch the remote version but no merging is processed locally.\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					out := fmt.Sprintf("Fetching...\n%s", XeqOut("git fetch origin"))
					MsgBox = MsgBox.OK("Git Fetch", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Fetch")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitPull()
// ****************************************************************************
func DoGitPull(f any) {
	if IsInsideGitWorkTree() {
		branch := XeqOut("git rev-parse --abbrev-ref HEAD")
		DlgYesNo = DlgYesNo.YesNo("Git Pull", // Title
			"The Git Pull will fetch the remote version and merge it with you local branch which is currently ["+branch+"].\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					out := fmt.Sprintf("Pulling...\n%s", XeqOut("git pull origin "+branch))
					MsgBox = MsgBox.OK("Git Pull", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Pull")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitPush()
// ****************************************************************************
func DoGitPush(f any) {
	if IsInsideGitWorkTree() {
		branch := XeqRaw("git rev-parse --abbrev-ref HEAD")
		out := fmt.Sprintf("Pushing...\n%s", XeqOut("git push origin "+branch))
		MsgBox = MsgBox.OK("Git Push", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
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
					MsgBox = MsgBox.OK("Git Commit", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Commit")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgInput", DlgInput.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgInput")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitClone()
// ****************************************************************************
func DoGitClone(f any) {
	if !IsInsideGitWorkTree() {
		DlgInput = DlgInput.Input("Git Clone", // Title
			"Please, enter the distant repository to clone locally :", // Message
			"https://github.com/repo/project.git",                     // Default value
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_OK {
					out := fmt.Sprintf("Cloning...\n%s", XeqOut("git clone \""+DlgInput.Value+"\""))
					MsgBox = MsgBox.OK("Git Clone", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Clone")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgInput", DlgInput.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgInput")
	} else {
		ui.SetStatus("Already in a Git repository")
	}
}

// ****************************************************************************
// DoGitAdd()
// ****************************************************************************
func DoGitAdd(f any) {
	if IsInsideGitWorkTree() {
		DlgYesNo = DlgYesNo.YesNo("Git Add", // Title
			"This will add the file :\n\n"+f.(string)+"\n\nto the Git tracking.\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					b := filepath.Base(f.(string))
					out := fmt.Sprintf("Adding...\n%s", XeqOut("git add ./"+utils.EscapeSpaces(b)))
					MsgBox = MsgBox.OK("Git Add", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Add")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitAddAll()
// ****************************************************************************
func DoGitAddAll(f any) {
	if IsInsideGitWorkTree() {
		DlgYesNo = DlgYesNo.YesNo("Git Add All", // Title
			"This will add all the files to the Git tracking.\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					out := fmt.Sprintf("Adding...\n%s", XeqOut("git add ."))
					MsgBox = MsgBox.OK("Git Add All", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Add All")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentView) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
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
	MsgBox = MsgBox.OK("New Version Available", msg, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
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
// IsInsideGitWorkTree()
// ****************************************************************************
func IsInsideGitWorkTree() bool {
	rc := false
	out := XeqOut("git rev-parse --is-inside-work-tree")
	if out[:4] == "true" {
		rc = true
	}
	return rc
}

// ****************************************************************************
// IsFileGitTracked()
// ****************************************************************************
func IsFileGitTracked(f string) bool {
	rc := false
	out := XeqOut("git ls-files --error-unmatch " + utils.EscapeSpaces(f))
	b := filepath.Base(f)
	if out[:len(b)] == b {
		ui.SetStatus(fmt.Sprintf("File [%s] is git-tracked", f))
		rc = true
	} else {
		ui.SetStatus(fmt.Sprintf("File [%s] is NOT git-tracked", f))
	}
	return rc
}

// ****************************************************************************
// SaveMacros()
// ****************************************************************************
func SaveMacros() {
	fMac, err := os.Create(filepath.Join(appDir, conf.FILE_MACROS))
	if err == nil {
		defer fMac.Close()
		wMac := bufio.NewWriter(fMac)
		fmt.Fprintln(wMac, "################################################################################")
		fmt.Fprintln(wMac, "# Macros file generated on "+time.Now().Format("20060201-150405"))
		fmt.Fprintln(wMac, "################################################################################")
		fmt.Fprintln(wMac, "# These placeholders can be used into macros :")
		fmt.Fprintln(wMac, "# %D : Full directory of current file")
		fmt.Fprintln(wMac, "# %P : Parent directory of current file")
		fmt.Fprintln(wMac, "# %W : Full directory of current workspace")
		fmt.Fprintln(wMac, "# %F : Full file name with directory and extension of current file")
		fmt.Fprintln(wMac, "# %f : File name without path and with extension of current file")
		fmt.Fprintln(wMac, "# %e : File name without path nor extension of current file")
		fmt.Fprintln(wMac, "# %L : Line number of current file in editor")
		fmt.Fprintln(wMac, "# %T : Current timestamp")
		fmt.Fprintln(wMac, "# %H : Home directory of current user")
		fmt.Fprintln(wMac, "# %s : OS path separator")
		fmt.Fprintln(wMac, "################################################################################")
		fmt.Fprintln(wMac, "")
		for k, v := range Macros {
			fmt.Fprintln(wMac, k+" : "+v)
		}
		wMac.Flush()
	}
}

// ****************************************************************************
// ReadMacros()
// ****************************************************************************
func ReadMacros() {
	fMac, err := os.Open(filepath.Join(appDir, conf.FILE_MACROS))
	if err == nil {
		for k := range Macros {
			delete(Macros, k)
		}
		defer fMac.Close()
		sMac := bufio.NewScanner(fMac)
		for sMac.Scan() {
			if strings.TrimSpace(string(sMac.Text())) != "" {
				if strings.TrimSpace(string(sMac.Text()[0])) != "#" {
					m := strings.Split(sMac.Text(), ":")
					Macros[strings.TrimSpace(m[0])] = strings.TrimSpace(strings.Join(m[1:], ":"))
				}
			}
		}
	}
}

// ****************************************************************************
// XeqMacro()
// ****************************************************************************
func XeqMacro(k any) {
	ui.SetStatus("Executing macro : [" + k.(string) + "]")
	out := fmt.Sprintf("%s\n", XeqOutErr(replaceVariablesInMacro(k.(string))))
	edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
	MsgBox = MsgBox.OK("Macro : "+k.(string), out, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// replaceVariablesInMacro()
// ****************************************************************************
func replaceVariablesInMacro(k string) string {
	// %D : Full directory of current file
	// %P : Parent directory of current file
	// %W : Full directory of current workspace
	// %F : Full file name with directory and extension of current file
	// %f : File name without path and with extension of current file
	// %e : File name without path nor extension of current file
	// %L : Line number of current file in editor
	// %T : Current timestamp
	// %H : Home directory of current user
	// %s : OS path separator
	out := Macros[k]
	userDir, _ := os.UserHomeDir()
	r := strings.NewReplacer(
		"%D", utils.EscapeSpaces(filepath.Dir(edit.CurrentFile.FName)),
		"%P", utils.EscapeSpaces(filepath.Base(filepath.Dir(edit.CurrentFile.FName))),
		"%W", utils.EscapeSpaces(conf.ConfigGeneral.Workspace),
		"%F", utils.EscapeSpaces(edit.CurrentFile.FName),
		"%f", utils.EscapeSpaces(filepath.Base(edit.CurrentFile.FName)),
		"%e", utils.EscapeSpaces(filepath.Base(strings.TrimSuffix(filepath.Base(edit.CurrentFile.FName), filepath.Ext(edit.CurrentFile.FName)))),
		"%T", time.Now().Format("20060102-150405"),
		"%L", strconv.Itoa(edit.CurrentFile.Buffer.Cursor.Y+1),
		"%s", string(os.PathSeparator),
		"%H", userDir,
	)
	out = r.Replace(out)
	return out
}

// ****************************************************************************
// editMacrosFile()
// ****************************************************************************
func editMacrosFile(f any) {
	edit.OpenFile(filepath.Join(appDir, conf.FILE_MACROS), true)
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
// ShowHelp()
// ****************************************************************************
func ShowHelp() {
	MsgBox = MsgBox.OK(" Help ", help.Help, nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
	ui.SetStatus("Displaying Help screen")
}

// ****************************************************************************
// ShowSysInfo()
// ****************************************************************************
func ShowSysInfo() {
	MsgBox = MsgBox.OK(" System Info ", sysinfo.GetFullReport(), nil, 0, ui.GetCurrentScreen(), edit.CurrentView)
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
