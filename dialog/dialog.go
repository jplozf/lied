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
	parent  string
	focus   tview.Primitive
	width   int
	height  int
	idx     int
	dtype   DlgInput
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
// refresh() the dialog
// ****************************************************************************
func (m *Dialog) refresh() {
	m.SetTitle(m.title)

	switch m.dtype {
	case INPUT_TEXT:
		m.AddTextView("", m.message, 0, 1, true, false)
		m.AddInputField(">", m.Value, 0, nil, nil)
	case INPUT_LIST:
		m.AddTextView("", m.message, 0, 1, true, false)
		m.AddDropDown("", m.Values, 0, nil)
	default:
		m.AddTextView("", m.message, 0, 1, true, false)
	}

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
	m.width += 10
	m.height = 9
	if m.dtype == INPUT_TEXT || m.dtype == INPUT_LIST {
		m.height += 2
	}
}

// ****************************************************************************
// Popup()
// ****************************************************************************
func (m *Dialog) Popup() tview.Primitive {
	m.refresh()
	m.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			ui.PgsApp.SwitchToPage(m.parent)
			ui.App.SetFocus(m.focus)
			return nil
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
	m.done(BUTTON_YES, m.idx)
}

// ****************************************************************************
// doNo()
// ****************************************************************************
func (m *Dialog) doNo() {
	ui.PgsApp.SwitchToPage(m.parent)
	ui.App.SetFocus(m.focus)
	m.done(BUTTON_NO, m.idx)
}

// ****************************************************************************
// doCancel()
// ****************************************************************************
func (m *Dialog) doCancel() {
	ui.PgsApp.SwitchToPage(m.parent)
	ui.App.SetFocus(m.focus)
	m.done(BUTTON_CANCEL, m.idx)
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
	default:
		m.Value = ""
	}
	m.done(BUTTON_OK, m.idx)
}
