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
package ui

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"bufio"
	"bytes"
	"fmt"
	"lied/conf"
	"lied/utils"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/pgavlin/femto"
	"github.com/rivo/tview"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type Fn func()
type FnAny func(any)

type Mode int

type MyScreen struct {
	ID    string
	Title string
	Page  *tview.Pages
	Keys  string
	Mode  Mode
	Init  FnAny
	Param any
	Flex  *tview.Flex
}

type Config struct {
	StartupScreen Mode   `json:"startup_screen"`
	FormatDate    string `json:"format_date"`
	FormatTime    string `json:"format_time"`
}

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	ModeShell Mode = iota
	ModeHelp
	ModeFiles
	ModeTextEdit
	ModeHexEdit
	ModeProcess
	ModeNetwork
	ModeSQLite3
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	SessionID      string
	IdxScreens     int
	ArrScreens     []MyScreen
	CurrentMode    Mode
	lblTime        *tview.TextView
	lblDate        *tview.TextView
	LblKeys        *tview.TextView
	App            *tview.Application
	FlxShell       *tview.Flex
	FlxFiles       *tview.Flex
	FlxProcess     *tview.Flex
	FlxHelp        *tview.Flex
	FlxEditor      *tview.Flex
	FlxSQL         *tview.Flex
	FlxHexEdit     *tview.Flex
	TxtPrompt      *tview.TextArea
	TxtConsole     *tview.TextView
	TxtFileInfo    *tview.TextView
	TxtProcInfo    *tview.TextView
	TxtHelp        *tview.TextView
	lblTitle       *tview.TextView
	lblStatus      *tview.TextView
	LblHostname    *tview.TextView
	LblScreen      *tview.TextView
	LblPID         *tview.TextView
	LblRC          *tview.TextView
	LblHourglass   *tview.TextView
	PgsApp         *tview.Pages
	DlgQuit        *tview.Modal
	TblFiles       *tview.Table
	TblProcess     *tview.Table
	TxtPath        *tview.TextView
	TxtProcess     *tview.TextView
	FrmFileInfo    *tview.TextView
	TblProcUsers   *tview.Table
	TxtSelection   *tview.TextView
	StdoutBuf      bytes.Buffer
	EdtMain        *femto.View
	TxtEditName    *tview.TextView
	TblOpenFiles   *tview.Table
	TrvExplorer    *tview.TreeView
	TxtSQLName     *tview.TextView
	TblSQLOutput   *tview.Table
	TblSQLTables   *tview.Table
	TrvSQLDatabase *tview.TreeView
	TxtHexName     *tview.TextView
	TblHexEdit     *tview.Table
	CmdOutput      string
	CmdOutputOld   string
	ScanCmd        *bufio.Scanner
	MyConfig       Config
)

// ****************************************************************************
// UnmarshalText() *Mode
// ****************************************************************************
func (m *Mode) UnmarshalText(b []byte) error {
	str := strings.Trim(string(b), `"`)

	switch {
	case str == "ModeShell":
		*m = ModeShell
	case str == "ModeHelp":
		*m = ModeHelp
	case str == "ModeFiles":
		*m = ModeFiles
	case str == "ModeTextEdit":
		*m = ModeTextEdit
	case str == "ModeHexEdit":
		*m = ModeHexEdit
	case str == "ModeProcess":
		*m = ModeProcess
	case str == "ModeNetwork":
		*m = ModeNetwork
	case str == "ModeSQLite3":
		*m = ModeSQLite3
	}

	return nil
}

// ****************************************************************************
// String() Mode
// ****************************************************************************
func (m Mode) String() string {

	switch m {
	case ModeShell:
		return "ModeShell"
	case ModeHelp:
		return "ModeHelp"
	case ModeFiles:
		return "ModeFiles"
	case ModeTextEdit:
		return "ModeTextEdit"
	case ModeHexEdit:
		return "ModeHexEdit"
	case ModeProcess:
		return "ModeProcess"
	case ModeNetwork:
		return "ModeNetwork"
	case ModeSQLite3:
		return "ModeSQLite3"
	}
	return "?"
}

// ****************************************************************************
// setUI()
// setUI defines the user interface's fields
// ****************************************************************************
func SetUI(fQuit Fn, hostname string) {
	PgsApp = tview.NewPages()

	lblDate = tview.NewTextView().SetText(currentDateString())
	lblDate.SetBorder(false)

	lblTime = tview.NewTextView().SetText(currentTimeString())
	lblTime.SetBorder(false)

	LblKeys = tview.NewTextView()
	LblKeys.SetBorder(false)
	LblKeys.SetBackgroundColor(tcell.ColorBlack)
	LblKeys.SetTextColor(tcell.ColorLightBlue)

	lblTitle = tview.NewTextView()
	lblTitle.SetBorder(false)
	lblTitle.SetBackgroundColor(tcell.ColorBlack)
	lblTitle.SetTextColor(tcell.ColorGreen)
	lblTitle.SetBorderColor(tcell.ColorDarkGreen)
	lblTitle.SetTextAlign(tview.AlignCenter)

	lblStatus = tview.NewTextView()
	lblStatus.SetBorder(false)
	lblStatus.SetBackgroundColor(tcell.ColorDarkGreen)
	lblStatus.SetTextColor(tcell.ColorWheat)

	LblScreen = tview.NewTextView()
	LblScreen.SetBorder(false)
	LblScreen.SetBackgroundColor(tcell.ColorDarkGreen)
	LblScreen.SetTextColor(tcell.ColorWheat)

	LblPID = tview.NewTextView()
	LblPID.SetBorder(false)
	LblPID.SetBackgroundColor(tcell.ColorDarkGreen)
	LblPID.SetTextColor(tcell.ColorWheat)

	LblRC = tview.NewTextView()
	LblRC.SetDynamicColors(true)
	LblRC.SetBorder(false)
	LblRC.SetBackgroundColor(tcell.ColorDarkGreen)
	LblRC.SetTextColor(tcell.ColorWheat)

	LblHourglass = tview.NewTextView()
	LblHourglass.SetBorder(false)
	LblHourglass.SetBackgroundColor(tcell.ColorDarkGreen)
	LblHourglass.SetTextColor(tcell.ColorWheat)

	LblHostname = tview.NewTextView()
	LblHostname.SetBorder(false)
	LblHostname.SetBackgroundColor(tcell.ColorDarkGreen)
	LblHostname.SetTextColor(tcell.ColorBlack)

	TxtPrompt = tview.NewTextArea().SetPlaceholder("Command to run")
	TxtPrompt.SetBorder(false)

	TxtHelp = tview.NewTextView().Clear()
	TxtHelp.SetBorder(true)
	TxtHelp.SetDynamicColors(true)

	TxtConsole = tview.NewTextView().Clear()
	TxtConsole.SetBorder(true)
	TxtConsole.SetDynamicColors(true)

	FrmFileInfo = tview.NewTextView()
	FrmFileInfo.SetBorder(true)
	FrmFileInfo.SetDynamicColors(true)
	FrmFileInfo.SetTitle("Infos")

	TxtFileInfo = tview.NewTextView().Clear()
	TxtFileInfo.SetBorder(true)
	TxtFileInfo.SetDynamicColors(true)
	TxtFileInfo.SetTitle("Preview")
	TxtFileInfo.SetWrap(false)
	TxtFileInfo.SetScrollable(true)

	TxtSelection = tview.NewTextView()
	TxtSelection.SetBorder(true)
	TxtSelection.SetDynamicColors(true)
	TxtSelection.SetTitle("Selection")

	TblFiles = tview.NewTable()
	TblFiles.SetBorder(true)
	TblFiles.SetSelectable(true, false)

	TxtPath = tview.NewTextView()
	TxtPath.Clear()
	TxtPath.SetBorder(true)

	TblProcUsers = tview.NewTable()
	TblProcUsers.SetBorder(true)
	TblProcUsers.SetTitle("Users")
	TblProcUsers.SetSelectable(true, false)

	TxtProcInfo = tview.NewTextView().Clear()
	TxtProcInfo.SetBorder(true)
	TxtProcInfo.SetDynamicColors(true)
	TxtProcInfo.SetTitle("Details")
	TxtProcInfo.SetWrap(false)
	TxtProcInfo.SetScrollable(true)

	TblProcess = tview.NewTable()
	TblProcess.SetBorder(true)
	TblProcess.SetSelectable(true, false)

	TxtProcess = tview.NewTextView()
	TxtProcess.Clear()
	TxtProcess.SetBorder(true)
	TxtProcess.SetDynamicColors(true)

	buffer := femto.NewBufferFromString(string("content"), "./dummy")
	EdtMain = femto.NewView(buffer)
	EdtMain.SetBorder(true)
	TxtEditName = tview.NewTextView()
	TxtEditName.Clear()
	TxtEditName.SetBorder(true)
	TblOpenFiles = tview.NewTable()
	TblOpenFiles.SetBorder(true)
	TblOpenFiles.SetSelectable(true, false)
	TblOpenFiles.SetTitle("Open Files")
	TrvExplorer = tview.NewTreeView()
	TrvExplorer.SetBorder(true)
	TrvExplorer.SetTitle("Explorer")

	TxtSQLName = tview.NewTextView()
	TxtSQLName.Clear()
	TxtSQLName.SetBorder(true)
	TxtSQLName.SetDynamicColors(true)
	TblSQLOutput = tview.NewTable()
	TblSQLOutput.SetBorder(true)
	TblSQLOutput.SetSelectable(true, true)
	TblSQLOutput.SetTitle("Output")
	TblSQLTables = tview.NewTable()
	TblSQLTables.SetBorder(true)
	TblSQLTables.SetSelectable(true, false)
	TblSQLTables.SetTitle("Tables")
	TrvSQLDatabase = tview.NewTreeView()
	TrvSQLDatabase.SetBorder(true)
	TrvSQLDatabase.SetTitle("Database")

	TxtHexName = tview.NewTextView()
	TxtHexName.Clear()
	TxtHexName.SetBorder(true)
	TxtHexName.SetDynamicColors(true)
	TblHexEdit = tview.NewTable()
	TblHexEdit.SetBorder(true)
	TblHexEdit.SetSelectable(true, true)
	TblHexEdit.SetTitle("Hexa View")

	//*************************************************************************
	// Main Layout (Shell)
	//*************************************************************************
	FlxShell = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 8, 0, false), 1, 0, false).
		AddItem(TxtConsole, 0, 1, false).
		AddItem(TxtPath, 3, 0, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(TxtPrompt, 2, 1, true).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblPID, 12, 0, false).
			AddItem(LblRC, 8, 0, false).
			AddItem(LblScreen, 5, 0, false).
			AddItem(LblHourglass, 2, 0, false), 1, 0, false)

	//*************************************************************************
	// Help Layout
	//*************************************************************************
	FlxHelp = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 8, 0, false), 1, 0, false).
		AddItem(TxtHelp, 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(TxtPrompt, 2, 1, true).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblScreen, 5, 0, false).
			AddItem(LblHourglass, 2, 0, false), 1, 0, false)

	//*************************************************************************
	// Files Layout
	//*************************************************************************
	FlxFiles = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 8, 0, false), 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TxtPath, 3, 0, false).
				AddItem(TblFiles, 0, 1, true), 0, 2, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(FrmFileInfo, 9, 0, false).
				AddItem(TxtFileInfo, 0, 1, false).
				AddItem(TxtSelection, 5, 0, false), 0, 1, false), 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(TxtPrompt, 2, 1, true).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblScreen, 5, 0, false).
			AddItem(LblHourglass, 2, 0, false), 1, 0, false)

	TblFiles.Select(0, 0).SetFixed(1, 1).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			TblFiles.SetSelectable(true, true)
		}
	}).SetSelectedFunc(func(row int, column int) {
		TblFiles.GetCell(row, column).SetTextColor(tcell.ColorRed)
		TblFiles.SetSelectable(false, false)
	})

	//*************************************************************************
	// Process Layout
	//*************************************************************************
	FlxProcess = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 8, 0, false), 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TxtProcess, 3, 0, false).
				AddItem(TblProcess, 0, 1, true), 0, 2, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TblProcUsers, 12, 0, false).
				AddItem(TxtProcInfo, 0, 1, false), 0, 1, false), 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(TxtPrompt, 2, 1, true).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblScreen, 5, 0, false).
			AddItem(LblHourglass, 2, 0, false), 1, 0, false)

	//*************************************************************************
	// Editor Layout
	//*************************************************************************
	FlxEditor = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 8, 0, false), 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TxtEditName, 3, 0, false).
				AddItem(EdtMain, 0, 1, true), 0, 2, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TblOpenFiles, 12, 0, false).
				AddItem(TrvExplorer, 0, 1, false), 0, 1, false), 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(TxtPrompt, 2, 1, true).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblScreen, 5, 0, false).
			AddItem(LblHourglass, 2, 0, false), 1, 0, false)

	//*************************************************************************
	// SQLite3 Layout
	//*************************************************************************
	FlxSQL = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 8, 0, false), 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TxtSQLName, 3, 0, false).
				AddItem(TblSQLOutput, 0, 1, true), 0, 2, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TblSQLTables, 12, 0, false).
				AddItem(TrvSQLDatabase, 0, 1, false), 0, 1, false), 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(TxtPrompt, 2, 1, true).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblScreen, 5, 0, false).
			AddItem(LblHourglass, 2, 0, false), 1, 0, false)

	//*************************************************************************
	// HexaEditor Layout
	//*************************************************************************
	FlxHexEdit = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 8, 0, false), 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TxtHexName, 3, 0, false).
				AddItem(TblHexEdit, 0, 1, true), 0, 2, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TblSQLTables, 12, 0, false).
				AddItem(TxtFileInfo, 0, 1, false), 0, 1, false), 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(TxtPrompt, 2, 1, true).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblScreen, 5, 0, false).
			AddItem(LblHourglass, 2, 0, false), 1, 0, false)

	//*************************************************************************
	// Misc
	//*************************************************************************
	DlgQuit = tview.NewModal().
		SetText("Do you want to quit the application ?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				fQuit()
			} else {
				PgsApp.SwitchToPage(GetCurrentScreen())
			}
		})
	IdxScreens = -1
}

// ****************************************************************************
// currentDateString()
// currentDateString returns the current date formatted as a string
// ****************************************************************************
func currentDateString() string {
	d := time.Now()
	return fmt.Sprint(d.Format(MyConfig.FormatDate))
}

// ****************************************************************************
// currentTimeString()
// currentTimeString returns the current time formatted as a string
// ****************************************************************************
func currentTimeString() string {
	t := time.Now()
	return fmt.Sprint(t.Format(MyConfig.FormatTime))
}

// ****************************************************************************
// updateTime()
// updateTime is the go routine which refresh the time and date
// ****************************************************************************
func UpdateTime() {
	for {
		time.Sleep(5 * time.Millisecond)
		App.QueueUpdateDraw(func() {
			lblDate.SetText(currentDateString())
			lblTime.SetText(currentTimeString())
			// TxtConsole.SetText(TxtConsole.GetText(false) + string(StdoutBuf.Bytes()))
			// StdoutBuf.Reset()
			// TxtConsole.SetText(string(StderrBuf.Bytes()))
			/*
				if CmdOutput != CmdOutputOld {
					TxtConsole.SetText(TxtConsole.GetText(false) + CmdOutput + "\n")
				}
				CmdOutputOld = CmdOutput
			*/
		})
	}
}

// ****************************************************************************
// setTitle()
// setTitle displays the title centered
// ****************************************************************************
func SetTitle(t string) {
	lblTitle.SetText(t)
}

// ****************************************************************************
// GetTitle()
// setTitle displays the title centered
// ****************************************************************************
func GetTitle() string {
	return lblTitle.GetText(true)
}

// ****************************************************************************
// SetStatus()
// SetStatus displays the status message during a specific time
// ****************************************************************************
func SetStatus(txt string) {
	lblStatus.SetText(txt)
	DurationOfTime := time.Duration(conf.STATUS_MESSAGE_DURATION) * time.Second
	f := func() {
		lblStatus.SetText("")
	}
	time.AfterFunc(DurationOfTime, f)
	current := time.Now()
	conf.LogFile.WriteString(fmt.Sprintf("%s [%s] : %s\n", current.Format("20060102-150405"), SessionID, txt))
}

// ****************************************************************************
// HeaderConsole()
// ****************************************************************************
func HeaderConsole(cmd string) {
	TxtConsole.SetText(TxtConsole.GetText(false) + "\n[red]⯈ " + cmd + ":\n[white]")
	TxtConsole.ScrollToEnd()
}

// ****************************************************************************
// outConsole()
// ****************************************************************************
func OutConsole(out string) {
	TxtConsole.SetText(TxtConsole.GetText(false) + "[white]" + out + "\n")
	TxtConsole.ScrollToEnd()
	App.Sync()
}

// ****************************************************************************
// DisplayMap()
// ****************************************************************************
func DisplayMap(tv *tview.TextView, m map[string]string) {
	// out := tv.GetText(true)
	out := ""
	maxi := 0
	for key := range m {
		if len(key) > maxi {
			maxi = len(key)
		}
	}
	// create slice and store keys
	fields := make([]string, 0, len(m))
	for k := range m {
		fields = append(fields, k)
	}

	// sort the slice by keys
	sort.Strings(fields)

	// iterate by sorted keys
	for _, field := range fields {
		out = out + "[red]" + field[2:] + strings.Repeat(" ", maxi-len(field)) + "[white]  " + m[field] + "\n"
	}
	tv.SetText(out)
}

// ****************************************************************************
// PromptInput()
// ****************************************************************************
func PromptInput(msg string, choice string) {
	TxtPrompt.SetText(msg, true)
}

// ****************************************************************************
// RemoveScreen()
// ****************************************************************************
func RemoveScreen(s []MyScreen, i int) []MyScreen {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// ****************************************************************************
// GetCurrentScreen()
// ****************************************************************************
func GetCurrentScreen() string {
	return (ArrScreens[IdxScreens].Title + "_" + ArrScreens[IdxScreens].ID)
}

// ****************************************************************************
// GetScreenFromTitle()
// ****************************************************************************
func GetScreenFromTitle(t string) string {
	for i := 0; i < len(ArrScreens); i++ {
		if ArrScreens[i].Title == t {
			return (ArrScreens[i].Title + "_" + ArrScreens[i].ID)
		}
	}
	return "NIL"
}

// ****************************************************************************
// CloseCurrentScreen()
// ****************************************************************************
func CloseCurrentScreen() {
	ArrScreens = RemoveScreen(ArrScreens, IdxScreens)
	if len(ArrScreens) == 0 {
		IdxScreens = -1
		AddNewScreen(ModeShell, nil, nil)
	} else {
		ShowPreviousScreen()
	}
	SetStatus("Closing current screen")
}

// ****************************************************************************
// ShowPreviousScreen()
// ****************************************************************************
func ShowPreviousScreen() {
	if IdxScreens > 0 {
		IdxScreens--
	} else {
		IdxScreens = len(ArrScreens) - 1
	}
	ShowScreen(IdxScreens)
	SetStatus("Switching to previous screen")
}

// ****************************************************************************
// ShowNextScreen()
// ****************************************************************************
func ShowNextScreen() {
	if IdxScreens < len(ArrScreens)-1 {
		IdxScreens++
	} else {
		IdxScreens = 0
	}
	ShowScreen(IdxScreens)
	SetStatus("Switching to next screen")
}

// ****************************************************************************
// ShowScreen()
// ****************************************************************************
func ShowScreen(idx any) {
	var screen MyScreen = ArrScreens[idx.(int)]
	SetTitle(screen.Title)
	CurrentMode = screen.Mode
	LblKeys.SetText(conf.FKEY_LABELS + "\n" + screen.Keys)
	PgsApp.SwitchToPage(screen.Title + "_" + screen.ID)
	IdxScreens = idx.(int)
	LblScreen.SetText(fmt.Sprintf("%d/%d", IdxScreens+1, len(ArrScreens)))
}

// ****************************************************************************
// AddNewScreen()
// ****************************************************************************
func AddNewScreen(mode Mode, selfInit FnAny, param any) {
	var screen MyScreen
	screen.ID, _ = utils.RandomHex(3)
	screen.Mode = mode
	screen.Init = selfInit
	screen.Param = param

	switch mode {
	case ModeFiles:
		screen.Title = "Files"
		screen.Keys = "Del=Delete Ins=Select Ctrl+A=Select/Unselect All Ctrl+C=Copy Ctrl+X=Cut Ctrl+V=Paste Ctrl+S=Sort"
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxFiles, true, true)
	case ModeHexEdit:
		screen.Title = "Hexedit"
		screen.Keys = "Ctrl+O=Open Ctrl+S=Save Ctrl+F=Find Ctrl+G=Go"
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxHexEdit, true, true)
	case ModeProcess:
		screen.Title = "Process"
		screen.Keys = "Ctrl+F=Find Ctrl+S=Sort Ctrl+V=Switch View"
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxProcess, true, true)
	case ModeSQLite3:
		screen.Title = "SQLite3"
		screen.Keys = "Ctrl+O=Open Ctrl+S=Save"
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxSQL, true, true)
	case ModeShell:
		screen.Title = "Shell"
		screen.Keys = ""
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxShell, true, true)
	case ModeTextEdit:
		screen.Title = "Editor"
		screen.Keys = "Ctrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+T=Close"
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxEditor, true, true)
	case ModeHelp:
		screen.Title = "Help"
		screen.Keys = ""
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxHelp, true, true)
	}
	ArrScreens = append(ArrScreens, screen)
	IdxScreens++
	ShowScreen(IdxScreens)
	if selfInit != nil {
		selfInit(param)
	}
	SetStatus(fmt.Sprintf("New screen [%s-%s]", screen.Title, strings.ToUpper(screen.ID)))
}

func PleaseWait() {
	SetStatus("Running...")
	LblHourglass.SetText("⌛")
	App.ForceDraw()
}

func JobsDone() {
	LblHourglass.SetText("")
}
