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
	"fmt"
	"lied/conf"
	"lied/ui"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
	BUTTON_DELETE
)

type DlgInput int

const (
	INPUT_NONE DlgInput = iota
	INPUT_TEXT
	INPUT_LIST
	INPUT_FOLDER
	INPUT_FILE
	INPUT_CLI
	INPUT_DELETE
)

type DlgRC struct {
	Button DlgButton
	Value  string
}

type Dialog struct {
	*tview.Form
	title       string
	message     string
	done        func(rc DlgButton, idx int)
	buttons     []*tview.Button
	Value       string
	Values      []string
	parent      string
	focus       tview.Primitive
	width       int
	height      int
	idx         int
	dtype       DlgInput
	UIMsg       *tview.TextView
	UIList      *tview.DropDown
	uiInput     tview.InputField
	IValues     int
	Path        string
	onlyFolders bool
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
func (m *Dialog) Command(title string, message string, value string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive, aCommands []string) *Dialog {
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
		Values:  aCommands,
	}

	m.IValues = 0
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
func (m *Dialog) FileBrowser(title string, path string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive, onlyFolders bool) *Dialog {
	m = &Dialog{
		Form:        tview.NewForm(),
		title:       title,
		Path:        path,
		done:        done,
		parent:      parent,
		focus:       focus,
		idx:         idx,
		dtype:       INPUT_FILE,
		onlyFolders: onlyFolders,
	}
	m.UIMsg = tview.NewTextView()
	m.UIMsg.SetSize(1, 60)
	m.UIMsg.SetLabel("Current :")
	m.UIMsg.SetText(m.Path)
	m.UIList = tview.NewDropDown()
	m.UIList.SetLabel("Browse  :")
	m.UIList.SetOptions([]string{}, nil)
	// m.UIList.SetSelectedFunc(m.selectPath)

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
// DeleteFileBrowser()
// ****************************************************************************
func (m *Dialog) DeleteFileBrowser(title string, path string, done func(rc DlgButton, idx int), idx int, parent string, focus tview.Primitive, onlyFolders bool) *Dialog {
	m = &Dialog{
		Form:        tview.NewForm(),
		title:       title,
		Path:        path,
		done:        done,
		parent:      parent,
		focus:       focus,
		idx:         idx,
		dtype:       INPUT_DELETE,
		onlyFolders: onlyFolders,
	}
	m.UIMsg = tview.NewTextView()
	m.UIMsg.SetSize(1, 60)
	m.UIMsg.SetLabel("Current :")
	m.UIMsg.SetText(m.Path)
	m.UIList = tview.NewDropDown()
	m.UIList.SetLabel("Browse  :")
	m.UIList.SetOptions([]string{}, nil)
	// m.UIList.SetSelectedFunc(m.selectPath)

	m.SetButtonsAlign(tview.AlignCenter)
	m.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	m.buttons = append(m.buttons, tview.NewButton("Delete").SetSelectedFunc(m.doDelete))
	m.buttons = append(m.buttons, tview.NewButton("Cancel").SetSelectedFunc(m.doCancel))
	m.setPath(path, 0)
	return m
}

// ****************************************************************************
// setPath()
// ****************************************************************************
func (m *Dialog) setPath(option string, optionIndex int) {
	ui.SetStatus("setPath: Received option: " + option)
	m.Values = nil
	if len(option) == 0 {
		option = "/"
	}
	if option[len(option)-1:] != "/" {
		option = option + "/"
	}

	go func(currentPath string) {
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			ui.SetStatus(currentPath + " : " + err.Error())
			return
		}

		var newValues []string
		if currentPath != "/" {
			newValues = append(newValues, "..")
		}
		for _, v := range entries {
			// Filter hidden files/folders if ShowHidden is false
			if !conf.ConfigGeneral.ShowHidden && strings.HasPrefix(v.Name(), ".") {
				continue
			}
			// Filter files if onlyFolders is true, but allow symlinks to directories
			if m.onlyFolders && !v.IsDir() {
				// If it's not a directory, check if it's a symlink to a directory
				fullPath := filepath.Join(currentPath, v.Name())
				fileInfo, err := os.Stat(fullPath)
				if err != nil || !fileInfo.IsDir() {
					continue // It's not a directory and not a symlink to a directory
				}
			}
			var entryName string
			if v.Type()&os.ModeSymlink != 0 {
				entryName = fmt.Sprintf("[yellow]%s[white]", v.Name())
			} else if v.IsDir() {
				entryName = fmt.Sprintf("[%s]%s[white]", conf.ConfigGeneral.ColorAccent, v.Name())
			} else {
				entryName = v.Name()
			}
			newValues = append(newValues, filepath.Join(currentPath, entryName))
		}

		ui.App.QueueUpdate(func() {
			m.Values = newValues
			m.UIList.SetOptions(m.Values, m.selectPath)
			m.Path = currentPath
			m.UIMsg.SetText(currentPath)
			ui.SetStatus("OPTION=" + currentPath)
			ui.App.SetFocus(m.UIList)
		})
	}(option)
}

// ****************************************************************************
// selectPath()
// ****************************************************************************
func (m *Dialog) selectPath(text string, index int) {
	go func(selectedText string) {
		// Strip tview color tags from the selectedText before using it for file system operations
		re := regexp.MustCompile(`\[[a-zA-Z0-9#]+\]|\[[a-zA-Z0-9#]+:[a-zA-Z0-9#]+\]|\[::\]`)
		cleanedSelectedText := re.ReplaceAllString(selectedText, "")

		if cleanedSelectedText == ".." {
			ui.App.QueueUpdate(func() {
				ui.SetStatus("selectPath: Handling '..' from m.Path: " + m.Path)
			})
			// Remove trailing slash before calculating parent to ensure correct behavior of filepath.Dir
			cleanedPath := m.Path
			if len(cleanedPath) > 1 && cleanedPath[len(cleanedPath)-1] == '/' {
				cleanedPath = cleanedPath[:len(cleanedPath)-1]
			}
			// Strip color tags from cleanedPath before passing to filepath.Dir
			cleanedPath = re.ReplaceAllString(cleanedPath, "")
			parentPath := filepath.Dir(cleanedPath)
			// Ensure parentPath also has a trailing slash if it's not the root
			if parentPath != "/" && parentPath[len(parentPath)-1] != '/' {
				parentPath += "/"
			}
			ui.App.QueueUpdate(func() {
				ui.SetStatus("selectPath: Cleaned path: " + cleanedPath)
				ui.SetStatus("selectPath: Calculated parentPath: " + parentPath)
				ui.SetStatus(fmt.Sprintf("selectPath: parentPath != m.Path: %t", parentPath != m.Path))
			})
			if parentPath != m.Path {
				ui.App.QueueUpdate(func() {
					ui.SetStatus("selectPath: Navigating to parent: " + parentPath)
				})
				m.setPath(parentPath, 0)
			} else {
				ui.App.QueueUpdate(func() {
					ui.SetStatus("selectPath: Already at root or parent is same as current: " + m.Path)
				})
			}
			return
		}

		fi, err := os.Stat(cleanedSelectedText)
		if err != nil {
			ui.App.QueueUpdate(func() {
				ui.SetStatus("selectPath: Error stating file: " + err.Error())
			})
			return
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			ui.App.QueueUpdate(func() {
				ui.SetStatus("selectPath: Selected directory: " + cleanedSelectedText)
				m.Value = cleanedSelectedText // Set m.Value to the selected directory
			})
			m.setPath(cleanedSelectedText, 0)

		case mode.IsRegular():
			ui.App.QueueUpdate(func() {
				ui.SetStatus("FILE " + cleanedSelectedText)
				m.Value = cleanedSelectedText
			})
		}
	}(text)
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
	case INPUT_FILE, INPUT_DELETE:
		m.Clear(true)
		m.AddFormItem(m.UIMsg)
		m.AddFormItem(m.UIList)
	case INPUT_CLI:
		if m.message != "" {
			m.AddTextView("", m.message, 0, 1, true, false)
		}
		m.AddInputField(">", m.Value, 0, nil, nil)
	default:
		m.AddTextView("", m.message, 60, 21, false, true)
		m.GetFormItem(0).(*tview.TextView).SetWrap(false)
		m.GetFormItem(0).(*tview.TextView).SetDynamicColors(true)
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
			} else if l == "Delete" {
				f = m.doDelete
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
		m.height = 27
	}
}

// ****************************************************************************
// Popup()
// ****************************************************************************
func (m *Dialog) Popup() tview.Primitive {
	m.setUI()
	// Focus is set in setPath after options are loaded

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
		case tcell.KeyUp:
			if m.dtype == INPUT_CLI {
				if len(m.Values) > 0 {
					m.IValues--
					if m.IValues < 0 {
						m.IValues = len(m.Values) - 1
					}
					m.GetFormItem(m.GetFormItemCount() - 1).(*tview.InputField).SetText(m.Values[m.IValues])
				}
				return nil
			} else {
				return event
			}
		case tcell.KeyDown:
			if m.dtype == INPUT_CLI {
				if len(m.Values) > 0 {
					m.IValues++
					if m.IValues > len(m.Values)-1 {
						m.IValues = 0
					}
					m.GetFormItem(m.GetFormItemCount() - 1).(*tview.InputField).SetText(m.Values[m.IValues])
				}
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
	case INPUT_FILE, INPUT_DELETE:
		if m.onlyFolders {
			m.Value = m.Path
		} else {
			_, selectedOption := m.UIList.GetCurrentOption()
			re := regexp.MustCompile(`\[[a-zA-Z0-9#]+\]|\[[a-zA-Z0-9#]+:[a-zA-Z0-9#]+\]|\[::\]`)
			m.Value = re.ReplaceAllString(selectedOption, "")
		}
	case INPUT_CLI:
		m.Value = m.GetFormItem(m.GetFormItemCount() - 1).(*tview.InputField).GetText()
	default:
		m.Value = ""
	}
	if m.done != nil {
		m.done(BUTTON_OK, m.idx)
	}
}

// ****************************************************************************
// doDelete()
// ****************************************************************************
func (m *Dialog) doDelete() {
	ui.PgsApp.SwitchToPage(m.parent)
	ui.App.SetFocus(m.focus)
	switch m.dtype {
	case INPUT_TEXT:
		m.Value = m.GetFormItem(1).(*tview.InputField).GetText()
	case INPUT_LIST:
		_, m.Value = m.GetFormItem(1).(*tview.DropDown).GetCurrentOption()
	case INPUT_FILE, INPUT_DELETE:
		if m.onlyFolders {
			m.Value = m.Path
		} else {
			_, selectedOption := m.UIList.GetCurrentOption()
			re := regexp.MustCompile(`\[[a-zA-Z0-9#]+\]|\[[a-zA-Z0-9#]+:[a-zA-Z0-9#]+\]|\[::\]`)
			m.Value = re.ReplaceAllString(selectedOption, "")
		}
	case INPUT_CLI:
		m.Value = m.GetFormItem(m.GetFormItemCount() - 1).(*tview.InputField).GetText()
	default:
		m.Value = ""
	}
	if m.done != nil {
		m.done(BUTTON_DELETE, m.idx)
	}
}
