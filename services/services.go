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

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *ServiceManagerPlugin) ID() string    { return PluginID }
func (p *ServiceManagerPlugin) Title() string { return "Service Manager" }
func (p *ServiceManagerPlugin) Icon() string  { return "⚙" }

// Activate switches the application to the service manager content page and
// sets focus to the services table.
func (p *ServiceManagerPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	ui.App.SetFocus(p.TblServices)
	ui.LblKeys.SetText(p.KeyHints())
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
		"[s] Start  [S] Stop  [r] Restart  [Enter] Journal  [F5] Refresh  [Ctrl+T] Close"
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
		unit   := fields[0]
		load   := fields[1]
		active := fields[2]
		sub    := fields[3]
		desc   := ""
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
