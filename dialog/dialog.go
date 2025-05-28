// ****************************************************************************
//
//	 _____ _____ _____ _____
//	|   __|     |   __|  |  |
//	|  |  |  |  |__   |     |
//	|_____|_____|_____|__|__|
//
// ****************************************************************************
// G O S H   -   Copyright © JPL 2023
// ****************************************************************************
package dialog

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"lied/ui"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type DlgButton int

const (
	BUTTON_OK DlgButton = iota
	BUTTON_YES
	BUTTON_NO
	BUTTON_CANCEL
)

type DlgInput int

const (
	INPUT_NONE DlgInput = iota
	INPUT_TEXT
	INPUT_LIST
	INPUT_FOLDER
	INPUT_FILE
	INPUT_CLI
)

type DlgRC struct {
	Button DlgButton
	Value  string
}

type Dialog struct {
	*tview.Form
	title   string
	message string
	done    func(rc DlgButton, idx int)
	buttons []*tview.Button
	Value   string
	Values  []string
	Path    string
	parent  string
	focus   tview.Primitive
	width   int
	height  int
	idx     int
	dtype   DlgInput
	UIMsg   tview.TextView
	UIList  tview.DropDown
	uiInput tview.InputField
}

// ****************************************************************************
// YesNoCancel()
// ****************************************************************************
func (m *Dialog) YesNoCancel(title string, message string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive) *Dialog {
	m = &Dialog{
		Form:    tview.NewForm(),
		title:   title,
		message: message,
		done:    done,
		parent:  parent,
		focus:   focus,
		idx:     idx,
		dtype:   INPUT_NONE,
	}

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	m.buttons = append(m.buttons, tview.NewButton("Yes").SetSelectedFunc(m.doYes))
	m.buttons = append(m.buttons, tview.NewButton("No").SetSelectedFunc(m.doNo))
	m.buttons = append(m.buttons, tview.NewButton("Cancel").SetSelectedFunc(m.doCancel))
	return m
}

// ****************************************************************************
// YesNo()
// ****************************************************************************
func (m *Dialog) YesNo(title string, message string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive) *Dialog {
	m = &Dialog{
		Form:    tview.NewForm(),
		title:   title,
		message: message,
		done:    done,
		parent:  parent,
		focus:   focus,
		idx:     idx,
		dtype:   INPUT_NONE,
	}

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	m.buttons = append(m.buttons, tview.NewButton("Yes").SetSelectedFunc(m.doYes))
	m.buttons = append(m.buttons, tview.NewButton("No").SetSelectedFunc(m.doNo))
	return m
}

// ****************************************************************************
// OK()
// ****************************************************************************
func (m *Dialog) OK(title string, message string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive) *Dialog {
	m = &Dialog{
		Form:    tview.NewForm(),
		title:   title,
		message: message,
		done:    done,
		parent:  parent,
		focus:   focus,
		idx:     idx,
		dtype:   INPUT_NONE,
	}

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	m.buttons = append(m.buttons, tview.NewButton("OK").SetSelectedFunc(m.doOK))
	return m
}

// ****************************************************************************
// Input()
// ****************************************************************************
func (m *Dialog) Input(title string, message string, value string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive) *Dialog {
	m = &Dialog{
		Form:    tview.NewForm(),
		title:   title,
		message: message,
		Value:   value,
		done:    done,
		parent:  parent,
		focus:   focus,
		idx:     idx,
		dtype:   INPUT_TEXT,
	}

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	m.buttons = append(m.buttons, tview.NewButton("OK").SetSelectedFunc(m.doOK))
	m.buttons = append(m.buttons, tview.NewButton("Cancel").SetSelectedFunc(m.doCancel))
	return m
}

// ****************************************************************************
// Command()
// ****************************************************************************
func (m *Dialog) Command(title string, message string, value string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive) *Dialog {
	m = &Dialog{
		Form:    tview.NewForm(),
		title:   title,
		message: message,
		Value:   value,
		done:    done,
		parent:  parent,
		focus:   focus,
		idx:     idx,
		dtype:   INPUT_CLI,
	}

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	return m
}

// ****************************************************************************
// List()
// ****************************************************************************
func (m *Dialog) List(title string, message string, values []string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive) *Dialog {
	m = &Dialog{
		Form:    tview.NewForm(),
		title:   title,
		message: message,
		Values:  values,
		done:    done,
		parent:  parent,
		focus:   focus,
		idx:     idx,
		dtype:   INPUT_LIST,
	}

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	m.buttons = append(m.buttons, tview.NewButton("OK").SetSelectedFunc(m.doOK))
	m.buttons = append(m.buttons, tview.NewButton("Cancel").SetSelectedFunc(m.doCancel))
	return m
}

// ****************************************************************************
// FileBrowser()
// ****************************************************************************
func (m *Dialog) FileBrowser(title string, path string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive) *Dialog {
	m = &Dialog{
		Form:   tview.NewForm(),
		title:  title,
		Path:   path,
		done:   done,
		parent: parent,
		focus:  focus,
		idx:    idx,
		dtype:  INPUT_FILE,
	}
	m.UIMsg = *tview.NewTextView()
	m.UIMsg.SetSize(1, 60)
	m.UIMsg.SetLabel("Current :")
	m.UIMsg.SetText(m.Path)
	m.AddFormItem(&m.UIMsg)
	m.UIList = *tview.NewDropDown()
	m.UIList.SetLabel("Browse  :")
	// m.UIList.SetSelectedFunc(m.selectPath)
	m.AddFormItem(&m.UIList)

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	m.buttons = append(m.buttons, tview.NewButton("OK").SetSelectedFunc(m.doOK))
	m.buttons = append(m.buttons, tview.NewButton("Cancel").SetSelectedFunc(m.doCancel))
	m.setPath(path, 0)
	return m
}

// ****************************************************************************
// setPath()
// ****************************************************************************
func (m *Dialog) setPath(option string, optionIndex int) {
	m.Values = nil
	// m.Path = option // CRASH !!!
	if option[len(option)-1:] != "/" {
		option = option + "/"
	}
	entries, err := os.ReadDir(option)
	if err != nil {
		ui.SetStatus(option + " : " + err.Error())
	}

	m.Values = append(m.Values, "..")
	ui.SetStatus("ENTRIES=" + strconv.Itoa(len(entries)))
	for _, v := range entries {
		m.Values = append(m.Values, filepath.Join(option, v.Name()))
		ui.SetStatus("append " + filepath.Join(option, v.Name()))
	}
	m.UIList.SetOptions(m.Values, m.selectPath)
}

// ****************************************************************************
// selectPath()
// ****************************************************************************
func (m *Dialog) selectPath(text string, index int) {
	fi, _ := os.Stat(text)
	switch mode := fi.Mode(); {
	case mode.IsDir():
		m.UIMsg.SetText(text)
		ui.SetStatus("DIR " + text)
		m.setPath(text, index)
		m.setUI()

	case mode.IsRegular():
		ui.SetStatus("FILE " + text)
		m.Value = text
	}
}

// ****************************************************************************
// setUI() for the dialog
// ****************************************************************************
func (m *Dialog) setUI() {
	m.SetTitle(m.title)
	switch m.dtype {
	case INPUT_TEXT:
		m.AddTextView("", m.message, 0, 1, true, false)
		m.AddInputField(">", m.Value, 0, nil, nil)
	case INPUT_LIST:
		m.AddTextView("", m.message, 0, 1, true, false)
		m.AddDropDown("", m.Values, 0, nil)
	case INPUT_FILE:
		m.Clear(true)
		m.setPath(m.Path, 0)
		m.AddFormItem(&m.UIMsg)
		m.AddFormItem(&m.UIList)
	case INPUT_CLI:
		if m.message != "" {
			m.AddTextView("", m.message, 0, 1, true, false)
		}
		m.AddInputField(">", m.Value, 0, nil, nil)
	default:
		m.AddTextView("", m.message, 60, 30, false, true)
	}

	if m.dtype != INPUT_CLI {
		for _, button := range m.buttons {
			l := button.GetLabel()
			var f func()
			if l == "Yes" {
				f = m.doYes
			} else if l == "No" {
				f = m.doNo
			} else if l == "OK" {
				f = m.doOK
			} else {
				f = m.doCancel
			}
			m.AddButton(l, f)
			m.width += len(l) + 2
		}
		if m.width < len(m.message) {
			m.width = len(m.message)
		}
		if m.width < len(m.title) {
			m.width = len(m.title)
		}
		if m.width < (len(m.Path) + 15) {
			m.width = len(m.Path) + 15
		}
		m.width += 10
		m.height = 9
		if m.dtype == INPUT_TEXT || m.dtype == INPUT_LIST {
			m.height += 2
		}
	} else {
		m.width = len(m.message)
		if m.width < len(m.title) {
			m.width = len(m.title)
		}
		if m.width < (len(m.Path) + 15) {
			m.width = len(m.Path) + 15
		}
		m.width += 10
		m.height = 7
	}
	if m.dtype == INPUT_NONE {
		m.width = 64
		m.height = 36
	}
}

// ****************************************************************************
// Popup()
// ****************************************************************************
func (m *Dialog) Popup() tview.Primitive {
	m.setUI()
	m.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			ui.PgsApp.SwitchToPage(m.parent)
			ui.App.SetFocus(m.focus)
			return nil
		case tcell.KeyEnter:
			if m.dtype == INPUT_CLI {
				ui.PgsApp.SwitchToPage(m.parent)
				ui.App.SetFocus(m.focus)
				m.doOK()
				return nil
			} else {
				return event
			}
		}
		return event
	})

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(m, m.height, 1, true).
			AddItem(nil, 0, 1, false), m.width, 1, true).
		AddItem(nil, 0, 1, false)
}

// ****************************************************************************
// doYes()
// ****************************************************************************
func (m *Dialog) doYes() {
	ui.PgsApp.SwitchToPage(m.parent)
	ui.App.SetFocus(m.focus)
	if m.done != nil {
		m.done(BUTTON_YES, m.idx)
	}
}

// ****************************************************************************
// doNo()
// ****************************************************************************
func (m *Dialog) doNo() {
	ui.PgsApp.SwitchToPage(m.parent)
	ui.App.SetFocus(m.focus)
	if m.done != nil {
		m.done(BUTTON_NO, m.idx)
	}
}

// ****************************************************************************
// doCancel()
// ****************************************************************************
func (m *Dialog) doCancel() {
	ui.PgsApp.SwitchToPage(m.parent)
	ui.App.SetFocus(m.focus)
	if m.done != nil {
		m.done(BUTTON_CANCEL, m.idx)
	}
}

// ****************************************************************************
// doOK()
// ****************************************************************************
func (m *Dialog) doOK() {
	ui.PgsApp.SwitchToPage(m.parent)
	ui.App.SetFocus(m.focus)
	switch m.dtype {
	case INPUT_TEXT:
		m.Value = m.GetFormItem(1).(*tview.InputField).GetText()
	case INPUT_LIST:
		_, m.Value = m.GetFormItem(1).(*tview.DropDown).GetCurrentOption()
	case INPUT_FILE:
		_, m.Value = m.UIList.GetCurrentOption()
	case INPUT_CLI:
		m.Value = m.GetFormItem(m.GetFormItemCount() - 1).(*tview.InputField).GetText()
	default:
		m.Value = ""
	}
	if m.done != nil {
		m.done(BUTTON_OK, m.idx)
	}
}
