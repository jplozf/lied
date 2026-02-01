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
	SessionID           string
	IdxScreens          int
	ArrScreens          []MyScreen
	lblTime             *tview.TextView
	lblDate             *tview.TextView
	LblKeys             *tview.TextView
	App                 *tview.Application
	FlxEditor           *tview.Flex
	MidColumn           *tview.Flex
	MidColumnSQL        *tview.Flex
	FlxSQLite           *tview.Flex
	TxtHelp             *tview.TextView
	lblTitle            *tview.TextView
	lblStatus           *tview.TextView
	LblHostname         *tview.TextView
	LblScreen           *tview.TextView
	LblPID              *tview.TextView
	LblRC               *tview.TextView
	LblHourglass        *tview.TextView
	PgsApp              *tview.Pages
	DlgQuit             *tview.Modal
	DlgSave             *tview.Modal
	StdoutBuf           bytes.Buffer
	EdtMain             *femto.View
	TxtCurrentWorkspace *tview.TextView
	TxtCurrentEditName  *tview.TextView
	TblOpenFiles        *tview.Table
	TblSQLOutput        *tview.Table
	TrvExplorer         *tview.TreeView
	MyConfig            Config
	LblReadWrite        *tview.TextView
	LblCursor           *tview.TextView
	LblDirty            *tview.TextView
	LblPercent          *tview.TextView
	LblSize             *tview.TextView
	LblCommit           *tview.TextView
	LblGITStatus        *tview.TextView
	LblGITBranch        *tview.TextView
	FrmFind             *tview.Form
	TxtFind             *tview.InputField
	ChkCase             *tview.Checkbox
	ChkToggleReplace    *tview.Checkbox
	TxtReplace          *tview.InputField
	DpdSearchType       *tview.DropDown
	TblOutline          *tview.Table
	HexView             *tview.TextView
	FlxHexViewer        *tview.Flex
	PgsEditorContent    *tview.Pages
	TxtPrompt           *tview.InputField
	TxtPromptSQL        *tview.TextArea
)

// ****************************************************************************
// ConfigureFindFormForBinary()
// ****************************************************************************
// ConfigureFindFormForBinary configures the FrmFind for binary or text files.
func ConfigureFindFormForBinary(isBinary bool) {
	TxtReplace.SetDisabled(isBinary)
	ChkToggleReplace.SetDisabled(isBinary)
	FrmFind.GetButton(2).SetDisabled(isBinary) // Replace button
	FrmFind.GetButton(3).SetDisabled(isBinary) // All button

	if isBinary {
		DpdSearchType.SetCurrentOption(1) // Default to Hexadecimal
	} else {
		DpdSearchType.SetCurrentOption(0) // Default to ASCII
	}
}

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
	tview.Styles.ContrastBackgroundColor = tcell.Color100

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

	LblReadWrite = tview.NewTextView()
	LblReadWrite.SetBorder(false)
	LblReadWrite.SetBackgroundColor(tcell.ColorDarkGreen)
	LblReadWrite.SetTextColor(tcell.ColorWheat)

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

	LblSize = tview.NewTextView()
	LblSize.SetBorder(false)
	LblSize.SetBackgroundColor(tcell.ColorDarkGreen)
	LblSize.SetTextColor(tcell.ColorWheat)

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
	buffer.Settings["keepautoindent"] = true
	buffer.Settings["softwrap"] = true
	buffer.Settings["scrollbar"] = true
	buffer.Settings["statusline"] = false

	EdtMain = femto.NewView(buffer)
	EdtMain.SetBorder(true)

	HexView = tview.NewTextView().SetWrap(false).SetWordWrap(false).SetDynamicColors(true)
	HexView.SetBorder(true).SetTitle("Hexadecimal")

	FlxHexViewer = tview.NewFlex().
		AddItem(HexView, 0, 1, false)

	TxtPromptSQL = tview.NewTextArea()
	TxtPromptSQL.SetTitle("SQL (Alt+Enter to Run)")
	TxtPromptSQL.SetBorder(true)

	TblSQLOutput = tview.NewTable()
	TblSQLOutput.SetBorder(true)
	TblSQLOutput.SetSelectable(true, false)
	TblSQLOutput.SetTitle("SQL Output")

	FlxSQLite = tview.NewFlex().SetDirection(tview.FlexRow).AddItem(TblSQLOutput, 0, 1, false).AddItem(TxtPromptSQL, 7, 0, true)

	PgsEditorContent = tview.NewPages().
		AddPage("textEditor", EdtMain, true, true).
		AddPage("hexViewer", FlxHexViewer, true, false). // Initially hidden
		AddPage("sqlViewer", FlxSQLite, true, false)     // Initially hidden

	TxtCurrentWorkspace = tview.NewTextView()
	TxtCurrentWorkspace.SetBorder(true)
	TxtCurrentWorkspace.SetDynamicColors(true)
	TxtCurrentWorkspace.SetTitleAlign(tview.AlignLeft)
	TxtCurrentWorkspace.SetTitle("Workspace")

	TxtCurrentEditName = tview.NewTextView()
	TxtCurrentEditName.SetBorder(true)
	TxtCurrentEditName.SetDynamicColors(true)
	TxtCurrentEditName.SetTitleAlign(tview.AlignLeft)
	TxtCurrentEditName.SetTitle("File")

	TblOpenFiles = tview.NewTable()
	TblOpenFiles.SetBorder(true)
	TblOpenFiles.SetSelectable(true, false)
	TblOpenFiles.SetTitle("Open Files")

	TrvExplorer = tview.NewTreeView()
	TrvExplorer.SetBorder(true)
	TrvExplorer.SetTitle("Explorer")

	TblOutline = tview.NewTable()
	TblOutline.SetBorder(true)
	TblOutline.SetSelectable(true, false)
	TblOutline.SetTitle("Outline")

	TxtPrompt = tview.NewInputField()
	TxtPrompt.SetLabel(":>")
	TxtPrompt.SetBorder(false)

	FrmFind = tview.NewForm()
	FrmFind.SetBorder(true)
	FrmFind.SetTitle("Find & Replace")
	TxtFind = tview.NewInputField()
	TxtFind.SetLabel("Find")
	TxtFind.SetBorder(false)
	TxtReplace = tview.NewInputField()
	TxtReplace.SetLabel("Replace")
	TxtReplace.SetBorder(false)
	TxtReplace.SetDisabled(true)
	ChkToggleReplace = tview.NewCheckbox()
	ChkToggleReplace.SetLabel("Toggle Replace")
	ChkToggleReplace.SetBorder(false)
	ChkToggleReplace.SetChangedFunc(func(_ bool) {
		TxtReplace.SetDisabled(!ChkToggleReplace.IsChecked())
		FrmFind.GetButton(2).SetDisabled(!ChkToggleReplace.IsChecked())
		FrmFind.GetButton(3).SetDisabled(!ChkToggleReplace.IsChecked())
	})
	ChkCase = tview.NewCheckbox()
	ChkCase.SetLabel("Case sensitive")
	ChkCase.SetBorder(false)

	DpdSearchType = tview.NewDropDown().
		SetLabel("Search Type").
		SetOptions([]string{"ASCII", "Hexadecimal"}, nil)
	DpdSearchType.SetCurrentOption(0) // Default to ASCII

	FrmFind.SetItemPadding(0)
	FrmFind.AddFormItem(TxtFind)
	FrmFind.AddFormItem(DpdSearchType)
	FrmFind.AddFormItem(ChkToggleReplace)
	FrmFind.AddFormItem(TxtReplace)
	FrmFind.AddFormItem(ChkCase)
	FrmFind.AddButton("Next", nil)
	FrmFind.AddButton("Previous", nil)
	FrmFind.AddButton("Replace", nil)
	FrmFind.AddButton("All", nil)
	FrmFind.GetButton(2).SetDisabled(!ChkToggleReplace.IsChecked())
	FrmFind.GetButton(3).SetDisabled(!ChkToggleReplace.IsChecked())

	//*************************************************************************
	// Editor Layout
	//*************************************************************************
	MidColumn = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(TxtCurrentWorkspace, 0, 2, false).
			AddItem(TxtCurrentEditName, 0, 1, false), 3, 0, false).
		AddItem(PgsEditorContent, 0, 1, true)
		// AddItem(TxtPrompt, 1, 1, false) // TxtPrompt is currently index 2

	FlxEditor = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(lblDate, 10, 0, false).
			AddItem(lblTitle, 0, 1, false).
			AddItem(lblTime, 10, 0, false), 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(MidColumn, 0, 2, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(TblOpenFiles, 12, 0, false).
				AddItem(FrmFind, 11, 0, false).
				AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(TrvExplorer, 0, 1, false).
					AddItem(TblOutline, 12, 0, false).
					AddItem(tview.NewFlex().
						AddItem(LblGITBranch, 0, 1, false).
						AddItem(LblCommit, 0, 1, false).
						AddItem(LblGITStatus, 0, 1, false), 1, 0, false), 0, 1, false), 0, 1, false), 0, 1, false).
		AddItem(LblKeys, 2, 1, false).
		AddItem(tview.NewFlex().
			AddItem(LblHostname, len(hostname)+3, 0, false).
			AddItem(lblStatus, 0, 1, false).
			AddItem(LblSize, 10, 0, false).
			AddItem(LblPercent, 6, 0, false).
			AddItem(LblCursor, 20, 0, false).
			AddItem(LblReadWrite, 4, 0, false).
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
				App.SetFocus(EdtMain)
			}
		})

	IdxScreens = -1
	ConfigureFindFormForBinary(false) // Default to text file configuration
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
// SetColorAccent()
// ****************************************************************************
func SetColorAccent(c string) {
	color := tcell.GetColor(c)
	lblDate.SetTextColor(color)
	lblTime.SetTextColor(color)
	lblTitle.SetTextColor(color)
	lblStatus.SetBackgroundColor(color)
	LblHostname.SetBackgroundColor(color)
	LblScreen.SetBackgroundColor(color)
	LblPID.SetBackgroundColor(color)
	LblRC.SetBackgroundColor(color)
	LblHourglass.SetBackgroundColor(color)
	LblReadWrite.SetBackgroundColor(color)
	LblCursor.SetBackgroundColor(color)
	LblDirty.SetBackgroundColor(color)
	LblPercent.SetBackgroundColor(color)
	LblSize.SetBackgroundColor(color)
	LblCommit.SetBackgroundColor(color)
	LblGITStatus.SetBackgroundColor(color)
	LblGITBranch.SetBackgroundColor(color)
	TxtFind.SetFieldBackgroundColor(color)
	TxtReplace.SetFieldBackgroundColor(color)
	ChkCase.SetFieldBackgroundColor(color)
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
	splittedText := strings.Split(txt, "\n")
	if len(splittedText) <= 1 {
		current := time.Now()
		conf.LogFile.WriteString(fmt.Sprintf("%s [%s] : %s\n", current.Format("20060102-150405"), SessionID, txt))
	} else {
		for _, s := range splittedText {
			current := time.Now()
			conf.LogFile.WriteString(fmt.Sprintf("%s [%s] : %s\n", current.Format("20060102-150405"), SessionID, s))
		}
	}
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
