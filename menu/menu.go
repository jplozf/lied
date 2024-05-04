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
package menu

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"lied/ui"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type MenuItem struct {
	Name    string
	Label   string
	Done    func(any)
	Param   any
	Enabled bool
	Checked bool
}

type Menu struct {
	*tview.Table
	title  string
	items  []MenuItem
	parent string
	focus  tview.Primitive
	width  int
	height int
}

// ****************************************************************************
// New() MenuItem
// ****************************************************************************
func (mi *MenuItem) New(name string, label string, done func(any), param any, enabled bool, checked bool) *MenuItem {
	mi = &MenuItem{
		Name:    name,
		Label:   label,
		Done:    done,
		Param:   param,
		Enabled: enabled,
		Checked: checked,
	}
	return mi
}

// ****************************************************************************
// New() Menu
// ****************************************************************************
func (m *Menu) New(title string, parent string, focus tview.Primitive) *Menu {
	m = &Menu{
		Table:  tview.NewTable(),
		title:  title,
		parent: parent,
		focus:  focus,
	}
	return m
}

// ****************************************************************************
// AddItem() Menu
// ****************************************************************************
func (m *Menu) AddItem(name string, label string, event func(any), param any, enabled bool, checked bool) {
	var item *MenuItem
	item = item.New(name, label, event, param, enabled, checked)
	m.items = append(m.items, *item)
}

// ****************************************************************************
// AddSeparator() Menu
// ****************************************************************************
func (m *Menu) AddSeparator() {
	var item *MenuItem
	item = item.New("SEPARATOR", "-", nil, nil, false, false)
	m.items = append(m.items, *item)
}

// ****************************************************************************
// SetEnabled() Menu
// ****************************************************************************
func (m *Menu) SetEnabled(miName string, e bool) {
	for index, item := range m.items {
		if item.Name == miName {
			m.items[index].Enabled = e
		}
	}
	m.refresh()
}

// ****************************************************************************
// SetChecked() Menu
// ****************************************************************************
func (m *Menu) SetChecked(miName string, c bool) {
	for index, item := range m.items {
		if item.Name == miName {
			m.items[index].Checked = c
		}
	}
	m.refresh()
}

// ****************************************************************************
// IsChecked() Menu
// ****************************************************************************
func (m *Menu) IsChecked(miName string) bool {
	for _, item := range m.items {
		if item.Name == miName {
			if item.Checked {
				return true
			}
		}
	}
	return false
}

// ****************************************************************************
// IsEnabled() Menu
// ****************************************************************************
func (m *Menu) IsEnabled(miName string) bool {
	for _, item := range m.items {
		if item.Name == miName {
			if item.Enabled {
				return true
			}
		}
	}
	return false
}

// ****************************************************************************
// SetLabel() Menu
// ****************************************************************************
func (m *Menu) SetLabel(miName string, label string) {
	for index, item := range m.items {
		if item.Name == miName {
			m.items[index].Label = label
		}
	}
	m.refresh()
}

// ****************************************************************************
// refresh() Menu
// ****************************************************************************
func (m *Menu) refresh() {
	m.width = 0
	m.height = len(m.items)
	m.Table.SetBorder(true)
	m.Table.SetTitle(m.title)
	m.Table.SetSelectable(true, false)
	m.Table.SetBackgroundColor(tcell.ColorBlue)
	for i, item := range m.items {
		prf := "  "
		if item.Checked {
			prf = "✓ "
		}
		item.Label = prf + item.Label + "  "
		if item.Enabled {
			m.Table.SetCell(i, 0, tview.NewTableCell(item.Label).SetTextColor(tcell.ColorYellow))
		} else {
			m.Table.SetCell(i, 0, tview.NewTableCell(item.Label).SetTextColor(tcell.ColorGray))
		}
		if len(item.Label) > m.width {
			m.width = len(item.Label)
		}
	}
	// Add some space around
	m.width = m.width + 2
	m.height = m.height + 2
	// Adapt length separator (if any) to the menu width
	for i, item := range m.items {
		if item.Name == "SEPARATOR" {
			item.Label = " " + strings.Repeat("─", m.width-4)
			m.Table.SetCell(i, 0, tview.NewTableCell(item.Label).SetTextColor(tcell.ColorGray))
		}
	}
}

// ****************************************************************************
// Popup()
// ****************************************************************************
func (m *Menu) Popup() tview.Primitive {
	m.refresh()
	m.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			idx, _ := m.Table.GetSelection()
			if m.items[idx].Enabled {
				ui.PgsApp.SwitchToPage(m.parent)
				ui.App.SetFocus(m.focus)
				m.items[idx].Done(m.items[idx].Param)
			}
			return nil
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
