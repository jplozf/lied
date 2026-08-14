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
// Package services provides a Service Manager plugin for lied.
// It lists systemd services and allows the user to start, stop, restart them
// and view their journal output — all from within the editor.
// ****************************************************************************
package services

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"fmt"
	"lied/menu"
	"lied/ui"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	PluginID        = "servicemanager"
	ContentPageName = "serviceManager"
	// UniqueID is used as the synthetic FName when this plugin is open in the
	// views list so that duplicate opens can be detected.
	UniqueID = PluginID + "://" + PluginID
)

// ****************************************************************************
// ServiceManagerPlugin
// ****************************************************************************

// ServiceManagerPlugin implements ui.ViewPlugin and shows a live list of
// systemd services together with a journal tail panel.
type ServiceManagerPlugin struct {
	// TblServices is the selectable table that lists all service units.
	TblServices *tview.Table
	// TxtJournal shows the last journal entries for the selected service.
	TxtJournal *tview.TextView
	layout     *tview.Flex
}

// NewServiceManagerPlugin creates and wires up the Service Manager plugin.
// It registers its content page directly with ui.PgsEditorContent so that it
// is rendered within the standard editor frame (header / key-hints / status bar)
// without needing its own full-screen layout.
func NewServiceManagerPlugin() *ServiceManagerPlugin {
	p := &ServiceManagerPlugin{}

	p.TblServices = tview.NewTable()
	p.TblServices.SetBorder(true)
	p.TblServices.SetTitle("Services")
	p.TblServices.SetSelectable(true, false)

	p.TxtJournal = tview.NewTextView()
	p.TxtJournal.SetBorder(true)
	p.TxtJournal.SetTitle("Journal")
	p.TxtJournal.SetDynamicColors(true)
	p.TxtJournal.SetScrollable(true)
	p.TxtJournal.SetWrap(true)

	p.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblServices, 0, 2, true).
		AddItem(p.TxtJournal, 0, 1, false)

	// Register the content with the shared PgsEditorContent pages so that the
	// standard editor frame (header, key hints, status bar) is retained.
	ui.PgsEditorContent.AddPage(ContentPageName, p.layout, true, false)

	// Keep the shared Outline panel in sync with the highlighted service.
	p.TblServices.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *ServiceManagerPlugin) ID() string    { return PluginID }
func (p *ServiceManagerPlugin) Title() string { return "Service Manager" }
func (p *ServiceManagerPlugin) Icon() string  { return "⚙" }

// Activate switches the application to the service manager content page and
// sets focus to the services table.  It also repurposes the shared Search and
// Outline panels: Search filters the services table by unit name, and Outline
// shows `systemctl show` properties for the currently selected service.
func (p *ServiceManagerPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	p.configureSearchPanel()
	p.RefreshOutline()
	ui.App.SetFocus(p.TblServices)
	ui.LblKeys.SetText(p.KeyHints())
}

// configureSearchPanel adapts the shared Find form to search services by unit
// name instead of searching text/hex buffer content.
func (p *ServiceManagerPlugin) configureSearchPanel() {
	ui.SetFindPanelVisible(true)
	ui.FrmFind.SetTitle("Find Service")
	ui.TxtReplace.SetDisabled(true)
	ui.ChkToggleReplace.SetDisabled(true)
	ui.FrmFind.GetButton(2).SetDisabled(true) // Replace One
	ui.FrmFind.GetButton(3).SetDisabled(true) // Replace All
	ui.DpdSearchType.SetDisabled(true)
	ui.DpdSearchType.SetCurrentOption(0)
	ui.FrmFind.GetButton(0).SetSelectedFunc(func() { p.FindNext() })
	ui.FrmFind.GetButton(1).SetSelectedFunc(func() { p.FindPrevious() })
}

// FocusWidget returns the services table as the primary focus target.
func (p *ServiceManagerPlugin) FocusWidget() tview.Primitive { return p.TblServices }

// Open refreshes the service list.  param is unused.
func (p *ServiceManagerPlugin) Open(_ any) error {
	p.Refresh()
	return nil
}

// Close is a no-op; the service manager holds no resources that need cleanup.
func (p *ServiceManagerPlugin) Close() error { return nil }

// IsDirty always returns false because the service manager is read-only.
func (p *ServiceManagerPlugin) IsDirty() bool { return false }

// StatusFields returns values for the bottom status bar widgets.
func (p *ServiceManagerPlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    "Services",
		Dirty:     "",
		Percent:   "",
		Size:      "",
		Encoding:  "systemd",
	}
}

// KeyHints returns the two-line key-hint string for the LblKeys bar.
func (p *ServiceManagerPlugin) KeyHints() string {
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[s] Start  [S] Stop  [r] Restart  [Enter] Journal  [F5] Refresh  [Ctrl+F] Find  [Ctrl+T] Close"
}

func (p *ServiceManagerPlugin) InternalCommand() string { return "!serv" }

func (p *ServiceManagerPlugin) CommandOpensPluginView() bool { return true }

func (p *ServiceManagerPlugin) ExecuteInternalCommand() error {
	// Command is handled by opening plugin view in dispatcher.
	return nil
}

func (p *ServiceManagerPlugin) ShowContextMenu(defaultMenu func()) bool {
	m := (&menu.Menu{}).New(" Service Manager ", ui.PopupParentPage(), p.FocusWidget())
	unit := p.SelectedUnit()
	hasUnit := unit != ""

	m.AddItem("mnuSvcRefresh", "Refresh services", func(any) {
		p.Refresh()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuSvcJournal", "Show journal", func(any) {
		u := p.SelectedUnit()
		if u != "" {
			p.ShowJournal(u)
		}
	}, nil, hasUnit, false)
	m.AddItem("mnuSvcStart", "Start service", func(any) {
		p.runSystemctl("start")
	}, nil, hasUnit, false)
	m.AddItem("mnuSvcStop", "Stop service", func(any) {
		p.runSystemctl("stop")
	}, nil, hasUnit, false)
	m.AddItem("mnuSvcRestart", "Restart service", func(any) {
		p.runSystemctl("restart")
	}, nil, hasUnit, false)

	ui.PgsApp.AddPage("dlgServiceManagerMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgServiceManagerMenu")
	return true
}

func (p *ServiceManagerPlugin) runSystemctl(action string) {
	unit := p.SelectedUnit()
	if unit == "" {
		ui.SetStatus("No service selected")
		return
	}
	if err := exec.Command("systemctl", action, unit).Run(); err != nil {
		ui.SetStatus(fmt.Sprintf("systemctl %s %s failed: %v", action, unit, err))
		return
	}
	p.Refresh()
	ui.SetStatus(fmt.Sprintf("Service %s: %s", unit, action))
}

// ****************************************************************************
// Public helpers used by lied.go keyboard handlers
// ****************************************************************************

// Refresh rebuilds the services table by running `systemctl list-units`.
func (p *ServiceManagerPlugin) Refresh() {
	p.TblServices.Clear()

	// Fixed header row
	headers := []string{"Unit", "Load", "Active", "Sub", "Description"}
	for col, h := range headers {
		p.TblServices.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAttributes(tcell.AttrBold).
				SetSelectable(false))
	}
	p.TblServices.SetFixed(1, 0)

	out, err := exec.Command(
		"systemctl", "list-units",
		"--type=service",
		"--no-pager", "--plain", "--no-legend",
	).Output()
	if err != nil {
		p.TblServices.SetCell(1, 0,
			tview.NewTableCell("Error: "+err.Error()).SetTextColor(tcell.ColorRed))
		return
	}

	row := 1
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		unit := fields[0]
		load := fields[1]
		active := fields[2]
		sub := fields[3]
		desc := ""
		if len(fields) > 4 {
			desc = strings.Join(fields[4:], " ")
		}

		color := tcell.ColorWhite
		switch active {
		case "active":
			color = tcell.ColorGreen
		case "failed":
			color = tcell.ColorRed
		case "activating", "deactivating":
			color = tcell.ColorYellow
		}

		p.TblServices.SetCell(row, 0, tview.NewTableCell(unit).SetTextColor(color))
		p.TblServices.SetCell(row, 1, tview.NewTableCell(load).SetTextColor(color))
		p.TblServices.SetCell(row, 2, tview.NewTableCell(active).SetTextColor(color))
		p.TblServices.SetCell(row, 3, tview.NewTableCell(sub).SetTextColor(color))
		p.TblServices.SetCell(row, 4, tview.NewTableCell(desc).SetTextColor(color))
		row++
	}

	p.TblServices.SetTitle(fmt.Sprintf("Services (%d)", row-1))
	p.RefreshOutline()
}

// ShowJournal fills TxtJournal with the last 100 journal entries for unit.
func (p *ServiceManagerPlugin) ShowJournal(unit string) {
	p.TxtJournal.Clear()
	out, err := exec.Command(
		"journalctl", "-u", unit,
		"-n", "100",
		"--no-pager",
	).Output()
	if err != nil {
		fmt.Fprintf(p.TxtJournal, "[red]Error reading journal: %v[-]", err)
		return
	}
	p.TxtJournal.SetText(string(out))
	p.TxtJournal.ScrollToEnd()
	p.TxtJournal.SetTitle(fmt.Sprintf("Journal — %s", unit))
}

// SelectedUnit returns the name of the currently selected service, or "" when
// no valid row is selected.
func (p *ServiceManagerPlugin) SelectedUnit() string {
	row, _ := p.TblServices.GetSelection()
	if row <= 0 || row >= p.TblServices.GetRowCount() {
		return ""
	}
	cell := p.TblServices.GetCell(row, 0)
	if cell == nil {
		return ""
	}
	return cell.Text
}

// ****************************************************************************
// Outline panel integration
// ****************************************************************************

// RefreshOutline populates the shared Outline panel with `systemctl show`
// properties for the currently selected service.
func (p *ServiceManagerPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	unit := p.SelectedUnit()
	if unit == "" {
		ui.TblOutline.SetTitle("Outline")
		return
	}

	out, err := exec.Command("systemctl", "show", unit, "--no-pager").Output()
	if err != nil {
		ui.TblOutline.SetCell(0, 0,
			tview.NewTableCell("Error: "+err.Error()).SetTextColor(tcell.ColorRed))
		return
	}

	row := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found || value == "" {
			continue
		}
		ui.TblOutline.SetCell(row, 0,
			tview.NewTableCell(key).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignLeft))
		ui.TblOutline.SetCell(row, 1,
			tview.NewTableCell(value).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		row++
	}
	ui.TblOutline.ScrollToBeginning()
	ui.TblOutline.SetTitle(fmt.Sprintf("Outline — %s", unit))
}

// ****************************************************************************
// Search panel integration
// ****************************************************************************

// matchesUnit reports whether row's unit name contains the given
// case-insensitive substring.
func (p *ServiceManagerPlugin) matchesUnit(row int, needle string) bool {
	cell := p.TblServices.GetCell(row, 0)
	if cell == nil {
		return false
	}
	return strings.Contains(strings.ToLower(cell.Text), needle)
}

// findService selects the next (or, if backward, previous) service row whose
// unit name contains the Find field's text, wrapping around the table.
func (p *ServiceManagerPlugin) findService(backward bool) {
	needle := strings.ToLower(strings.TrimSpace(ui.TxtFind.GetText()))
	rowCount := p.TblServices.GetRowCount()
	if needle == "" || rowCount <= 1 {
		ui.FrmFind.SetTitle("Find Service")
		ui.SetStatus("Nothing to search")
		return
	}

	total := 0
	for r := 1; r < rowCount; r++ {
		if p.matchesUnit(r, needle) {
			total++
		}
	}
	if total == 0 {
		ui.FrmFind.SetTitle("Find Service (0/0)")
		ui.SetStatus(fmt.Sprintf("No service matching '%s'", needle))
		return
	}

	current, _ := p.TblServices.GetSelection()
	if current < 1 {
		current = 1
	}
	dataRows := rowCount - 1
	step := 1
	if backward {
		step = -1
	}
	start := current - 1 // 0-based index within data rows
	for i := 1; i <= dataRows; i++ {
		idx := ((start+i*step)%dataRows + dataRows) % dataRows
		r := idx + 1 // back into the 1..rowCount-1 range
		if p.matchesUnit(r, needle) {
			p.TblServices.Select(r, 0)
			p.RefreshOutline()
			ui.FrmFind.SetTitle(fmt.Sprintf("Find Service (%d/%d)", r, total))
			ui.SetStatus(fmt.Sprintf("Found '%s' at %s", needle, p.TblServices.GetCell(r, 0).Text))
			return
		}
	}
}

// FindNext selects the next service matching the shared Find field.
func (p *ServiceManagerPlugin) FindNext() { p.findService(false) }

// FindPrevious selects the previous service matching the shared Find field.
func (p *ServiceManagerPlugin) FindPrevious() { p.findService(true) }
