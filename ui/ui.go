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
	"bytes"
	"fmt"
	"lied/conf"
	"lied/utils"
	"sort"
	"strconv"
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
	Idx   int
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
	FormatDate string
	FormatTime string
}

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	ModeHelp Mode = iota
	ModeTextEdit
)

// ****************************************************************************
// GLOBALS
// ****************************************************************************
var (
	SessionID    string
	IdxScreens   int
	ArrScreens   []MyScreen
	CurrentMode  Mode
	lblTime      *tview.TextView
	lblDate      *tview.TextView
	LblKeys      *tview.TextView
	App          *tview.Application
	FlxHelp      *tview.Flex
	FlxEditor    *tview.Flex
	TxtHelp      *tview.TextView
	lblTitle     *tview.TextView
	lblStatus    *tview.TextView
	LblHostname  *tview.TextView
	LblScreen    *tview.TextView
	LblPID       *tview.TextView
	LblRC        *tview.TextView
	LblHourglass *tview.TextView
	PgsApp       *tview.Pages
	DlgQuit      *tview.Modal
	StdoutBuf    bytes.Buffer
	EdtMain      *femto.View
	TxtEditName  *tview.TextView
	TblOpenFiles *tview.Table
	TrvExplorer  *tview.TreeView
	MyConfig     Config
	LblEncoding  *tview.TextView
	LblCursor    *tview.TextView
	LblDirty     *tview.TextView
	LblPercent   *tview.TextView
	LblCommit    *tview.TextView
	LblGITStatus *tview.TextView
	LblGITBranch *tview.TextView
)

// ****************************************************************************
// UnmarshalText() *Mode
// ****************************************************************************
func (m *Mode) UnmarshalText(b []byte) error {
	str := strings.Trim(string(b), `"`)

	switch {
	case str == "ModeHelp":
		*m = ModeHelp
	case str == "ModeTextEdit":
		*m = ModeTextEdit
	}

	return nil
}

// ****************************************************************************
// String() Mode
// ****************************************************************************
func (m Mode) String() string {

	switch m {
	case ModeHelp:
		return "ModeHelp"
	case ModeTextEdit:
		return "ModeTextEdit"
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
	lblTime.SetTextAlign(tview.AlignRight)

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

	LblEncoding = tview.NewTextView()
	LblEncoding.SetBorder(false)
	LblEncoding.SetBackgroundColor(tcell.ColorDarkGreen)
	LblEncoding.SetTextColor(tcell.ColorWheat)

	LblCursor = tview.NewTextView()
	LblCursor.SetBorder(false)
	LblCursor.SetBackgroundColor(tcell.ColorDarkGreen)
	LblCursor.SetTextColor(tcell.ColorWheat)

	LblDirty = tview.NewTextView()
	LblDirty.SetBorder(false)
	LblDirty.SetBackgroundColor(tcell.ColorDarkGreen)
	LblDirty.SetTextColor(tcell.ColorWheat)

	LblPercent = tview.NewTextView()
	LblPercent.SetBorder(false)
	LblPercent.SetBackgroundColor(tcell.ColorDarkGreen)
	LblPercent.SetTextColor(tcell.ColorWheat)

	LblCommit = tview.NewTextView()
	LblCommit.SetBorder(false)
	LblCommit.SetBackgroundColor(tcell.ColorDarkGreen)
	LblCommit.SetTextColor(tcell.ColorWheat)

	LblGITStatus = tview.NewTextView()
	LblGITStatus.SetBorder(false)
	LblGITStatus.SetBackgroundColor(tcell.ColorDarkGreen)
	LblGITStatus.SetTextColor(tcell.ColorWheat)

	LblGITBranch = tview.NewTextView()
	LblGITBranch.SetBorder(false)
	LblGITBranch.SetBackgroundColor(tcell.ColorDarkGreen)
	LblGITBranch.SetTextColor(tcell.ColorWheat)

	TxtHelp = tview.NewTextView().Clear()
	TxtHelp.SetBorder(true)
	TxtHelp.SetDynamicColors(true)

	buffer := femto.NewBufferFromString(string("content"), "./dummy")
	EdtMain = femto.NewView(buffer)
	EdtMain.SetBorder(true)
	TxtEditName = tview.NewTextView()
	TxtEditName.Clear()
	TxtEditName.SetBorder(true)
	TxtEditName.SetDynamicColors(true)
	TblOpenFiles = tview.NewTable()
	TblOpenFiles.SetBorder(true)
	TblOpenFiles.SetSelectable(true, false)
	TblOpenFiles.SetTitle("Open Files")
	TrvExplorer = tview.NewTreeView()
	TrvExplorer.SetBorder(true)
	TrvExplorer.SetTitle("Explorer")

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
			AddItem(lblTime, 10, 0, false), 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TxtEditName, 3, 0, false).
				AddItem(EdtMain, 0, 1, true), 0, 2, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TblOpenFiles, 12, 0, false).
				AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(TrvExplorer, 0, 1, false).
					AddItem(tview.NewFlex().
						AddItem(LblGITBranch, 0, 1, false).
						AddItem(LblCommit, 0, 1, false).
						AddItem(LblGITStatus, 0, 1, false), 1, 0, false), 0, 1, false), 0, 1, false), 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblPercent, 6, 0, false).
			AddItem(LblCursor, 15, 0, false).
			AddItem(LblEncoding, 10, 0, false).
			AddItem(LblDirty, 10, 0, false).
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
			// return (ArrScreens[i].Title + "_" + ArrScreens[i].ID)
			return strconv.Itoa(ArrScreens[i].Idx)
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
		AddNewScreen(ModeTextEdit, nil, nil)
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
	case ModeTextEdit:
		screen.Title = "Editor"
		screen.Keys = conf.CKEY_LABELS
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxEditor, true, true)
	case ModeHelp:
		screen.Title = "Help"
		screen.Keys = ""
		PgsApp.AddPage(screen.Title+"_"+screen.ID, FlxHelp, true, true)
	}
	IdxScreens++
	screen.Idx = IdxScreens
	ArrScreens = append(ArrScreens, screen)
	ShowScreen(IdxScreens)
	if selfInit != nil {
		selfInit(param)
	}
	SetStatus(fmt.Sprintf("New screen [%s-%s]", screen.Title, strings.ToUpper(screen.ID)))
}

// ****************************************************************************
// PleaseWait()
// ****************************************************************************
func PleaseWait() {
	SetStatus("Running...")
	LblHourglass.SetText("⌛")
	App.ForceDraw()
}

// ****************************************************************************
// JobsDone()
// ****************************************************************************
func JobsDone() {
	LblHourglass.SetText("")
}
