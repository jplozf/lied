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
// Package process provides a Process Manager plugin for lied.
// Its primary view lists running processes (auto-refreshing, with the
// ability to kill a selected process); a secondary view lists systemd
// services and allows the user to start, stop, restart them and view their
// journal output — all from within the editor.
// ****************************************************************************
package process

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"fmt"
	"lied/dialog"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	PluginID        = "processmanager"
	ContentPageName = "processManager"
	// UniqueID is used as the synthetic FName when this plugin is open in the
	// views list so that duplicate opens can be detected.
	UniqueID = PluginID + "://" + PluginID

	servicesViewName  = "services"
	processesViewName = "processes"
	processesInterval = 2 * time.Second
)

// ****************************************************************************
// ProcessManagerPlugin
// ****************************************************************************

// ProcessManagerPlugin implements ui.ViewPlugin. Its primary view is a live,
// auto-refreshing list of running processes; a secondary view lists systemd
// services together with a journal tail panel.
type ProcessManagerPlugin struct {
	// TblServices is the selectable table that lists all service units.
	TblServices *tview.Table
	// TxtJournal shows the last journal entries for the selected service.
	TxtJournal *tview.TextView
	views      *tview.Pages
	layout     *tview.Flex
	confirm    *dialog.Dialog

	// Running processes ("watch") view — the plugin's default/primary view.
	TblProcesses    *tview.Table
	processesLayout *tview.Flex
	// servicesActive is true when the secondary systemd services view is
	// showing instead of the default processes view.
	servicesActive bool
	watchRunning   bool
	stopWatch      chan struct{}
	processes      []ProcessInfo
}

// NewProcessManagerPlugin creates and wires up the Process Manager plugin.
// It registers its content page directly with ui.PgsEditorContent so that it
// is rendered within the standard editor frame (header / key-hints / status bar)
// without needing its own full-screen layout.
func NewProcessManagerPlugin() *ProcessManagerPlugin {
	p := &ProcessManagerPlugin{}

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

	p.TblProcesses = tview.NewTable()
	p.TblProcesses.SetBorder(true)
	p.TblProcesses.SetTitle("Processes")
	p.TblProcesses.SetSelectable(true, false)

	p.processesLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblProcesses, 0, 1, true)

	p.views = tview.NewPages().
		AddPage(processesViewName, p.processesLayout, true, true).
		AddPage(servicesViewName, p.layout, true, false)

	// Register the content with the shared PgsEditorContent pages so that the
	// standard editor frame (header, key hints, status bar) is retained.
	ui.PgsEditorContent.AddPage(ContentPageName, p.views, true, false)

	// Keep the shared Outline panel in sync with the highlighted service.
	p.TblServices.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})
	p.TblProcesses.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *ProcessManagerPlugin) ID() string    { return PluginID }
func (p *ProcessManagerPlugin) Title() string { return "Process Manager" }
func (p *ProcessManagerPlugin) Icon() string  { return "⚙" }

// Activate switches the application to the process manager content page and
// sets focus to whichever sub-view (processes or services) is currently
// active.  It also repurposes the shared Search and Outline panels
// accordingly.
func (p *ProcessManagerPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	if p.servicesActive {
		p.configureSearchPanel()
	} else {
		p.configureProcessSearchPanel()
		p.populateProcessesTable()
		p.startWatchProcesses()
	}
	p.RefreshOutline()
	ui.App.SetFocus(p.FocusWidget())
	ui.LblKeys.SetText(p.KeyHints())
}

// configureSearchPanel adapts the shared Find form to search services by unit
// name instead of searching text/hex buffer content.
func (p *ProcessManagerPlugin) configureSearchPanel() {
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

// FocusWidget returns the primary focus target for the currently active
// sub-view (processes table by default, or services table when watching
// systemd units).
func (p *ProcessManagerPlugin) FocusWidget() tview.Primitive {
	if p.servicesActive {
		return p.TblServices
	}
	return p.TblProcesses
}

// Open populates the processes table and starts auto-refreshing it, the
// plugin's default view.  param is unused.
func (p *ProcessManagerPlugin) Open(_ any) error {
	p.populateProcessesTable()
	p.startWatchProcesses()
	return nil
}

// Close stops the processes watch loop, if running.
func (p *ProcessManagerPlugin) Close() error {
	p.StopWatchProcesses()
	return nil
}

// IsDirty always returns false because the process manager is read-only.
func (p *ProcessManagerPlugin) IsDirty() bool { return false }

// StatusFields returns values for the bottom status bar widgets.
func (p *ProcessManagerPlugin) StatusFields() ui.ViewStatus {
	if p.servicesActive {
		return ui.ViewStatus{
			ReadWrite: "--",
			Cursor:    "Services",
			Dirty:     "",
			Percent:   "",
			Size:      "",
			Encoding:  "systemd",
		}
	}
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d process(es)", len(p.processes)),
		Encoding:  "processes",
	}
}

// KeyHints returns the two-line key-hint string for the LblKeys bar.
func (p *ProcessManagerPlugin) KeyHints() string {
	if p.servicesActive {
		return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
			"[s] Start  [S] Stop  [r] Restart  [Enter] Journal  [F5] Refresh  [Ctrl+F] Find  [Esc] Back  [Ctrl+T] Close"
	}
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[k] Kill process  [v] Services  [Ctrl+F] Find  [Ctrl+T] Close"
}

func (p *ProcessManagerPlugin) InternalCommand() string { return "!proc" }

func (p *ProcessManagerPlugin) CommandOpensPluginView() bool { return true }

func (p *ProcessManagerPlugin) ExecuteInternalCommand() error {
	// Command is handled by opening plugin view in dispatcher.
	return nil
}

func (p *ProcessManagerPlugin) ShowContextMenu(defaultMenu func()) bool {
	if p.servicesActive {
		return p.showServicesContextMenu()
	}
	return p.showProcessesContextMenu()
}

// showServicesContextMenu shows the context menu available while watching
// systemd services.
func (p *ProcessManagerPlugin) showServicesContextMenu() bool {
	m := (&menu.Menu{}).New(" Service Manager ", ui.PopupParentPage(), p.FocusWidget())
	edit.AddOpenViewsMenuItems(m)
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
	m.AddSeparator()
	m.AddItem("mnuSvcBackToProcesses", "Back to processes", func(any) {
		p.closeServices()
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgServiceManagerMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgServiceManagerMenu")
	return true
}

func (p *ProcessManagerPlugin) runSystemctl(action string) {
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
func (p *ProcessManagerPlugin) Refresh() {
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
func (p *ProcessManagerPlugin) ShowJournal(unit string) {
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
func (p *ProcessManagerPlugin) SelectedUnit() string {
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

// RefreshOutline populates the shared Outline panel with details of the
// current selection in whichever sub-view is active.
func (p *ProcessManagerPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	if !p.servicesActive {
		p.refreshProcessOutline()
		return
	}
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

// refreshProcessOutline shows details for the currently selected process in
// the shared Outline panel.
func (p *ProcessManagerPlugin) refreshProcessOutline() {
	proc := p.SelectedProcess()
	if proc == nil {
		ui.TblOutline.SetTitle("Outline")
		return
	}
	fields := [][2]string{
		{"PID", strconv.Itoa(proc.PID)},
		{"User", proc.User},
		{"CPU %", proc.CPU},
		{"MEM %", proc.MEM},
		{"State", proc.State},
		{"Command", proc.Command},
	}
	for r, field := range fields {
		ui.TblOutline.SetCell(r, 0, tview.NewTableCell(field[0]).SetTextColor(tcell.ColorLightCyan))
		ui.TblOutline.SetCell(r, 1, tview.NewTableCell(field[1]).SetTextColor(tcell.ColorWhite))
	}
	ui.TblOutline.SetTitle(fmt.Sprintf("Outline — pid %d", proc.PID))
}

// ****************************************************************************
// Running processes ("watch") view
// ****************************************************************************

// ProcessInfo describes a single running process as reported by `ps`.
type ProcessInfo struct {
	PID     int
	User    string
	CPU     string
	MEM     string
	State   string
	Command string
}

// processLineRE parses a line of `ps -eo pid,user,%cpu,%mem,stat,args`
// output into its fixed-width fields followed by the free-form command.
var processLineRE = regexp.MustCompile(`^\s*(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.*)$`)

// getProcesses runs `ps -eo pid,user,%cpu,%mem,stat,args` and parses its
// output into a list of ProcessInfo entries, sorted by CPU usage.
func getProcesses() ([]ProcessInfo, error) {
	out, err := exec.Command("ps", "-eo", "pid,user,%cpu,%mem,stat,args", "--no-headers", "--sort=-%cpu").Output()
	if err != nil {
		return nil, err
	}
	var procs []ProcessInfo
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := processLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pid, _ := strconv.Atoi(m[1])
		procs = append(procs, ProcessInfo{
			PID:     pid,
			User:    m[2],
			CPU:     m[3],
			MEM:     m[4],
			State:   m[5],
			Command: m[6],
		})
	}
	return procs, nil
}

// ShowServices switches to the systemd services sub-view, pausing the
// processes watch loop until closeServices returns to the default view.
func (p *ProcessManagerPlugin) ShowServices() {
	p.StopWatchProcesses()
	p.servicesActive = true
	p.views.SwitchToPage(servicesViewName)
	p.configureSearchPanel()
	p.Refresh()
	ui.App.SetFocus(p.TblServices)
	ui.LblKeys.SetText(p.KeyHints())
}

// configureProcessSearchPanel adapts the shared Find form to search running
// processes by PID or command name instead of searching text/hex buffer
// content.
func (p *ProcessManagerPlugin) configureProcessSearchPanel() {
	ui.SetFindPanelVisible(true)
	ui.FrmFind.SetTitle("Find Process")
	ui.TxtReplace.SetDisabled(true)
	ui.ChkToggleReplace.SetDisabled(true)
	ui.FrmFind.GetButton(2).SetDisabled(true) // Replace One
	ui.FrmFind.GetButton(3).SetDisabled(true) // Replace All
	ui.DpdSearchType.SetDisabled(true)
	ui.DpdSearchType.SetCurrentOption(0)
	ui.FrmFind.GetButton(0).SetSelectedFunc(func() { p.FindNextProcess() })
	ui.FrmFind.GetButton(1).SetSelectedFunc(func() { p.FindPreviousProcess() })
}

// closeServices stops the services view and returns to the default
// processes view, resuming the watch loop.
func (p *ProcessManagerPlugin) closeServices() {
	p.servicesActive = false
	p.views.SwitchToPage(processesViewName)
	p.configureProcessSearchPanel()
	ui.App.SetFocus(p.TblProcesses)
	ui.LblKeys.SetText(p.KeyHints())
	p.populateProcessesTable()
	p.startWatchProcesses()
}

// startWatchProcesses launches the background loop that periodically
// refreshes the processes table until StopWatchProcesses is called.
func (p *ProcessManagerPlugin) startWatchProcesses() {
	if p.watchRunning {
		return
	}
	p.watchRunning = true
	p.stopWatch = make(chan struct{})
	ui.SetStatus("Watching processes (Esc to stop)")
	go p.watchProcesses(p.stopWatch)
}

// StopWatchProcesses signals the running watch loop to stop.
func (p *ProcessManagerPlugin) StopWatchProcesses() {
	if !p.watchRunning || p.stopWatch == nil {
		return
	}
	close(p.stopWatch)
	p.stopWatch = nil
}

// watchProcesses periodically refreshes the processes table until stop is
// closed, or until another view becomes active (in which case it stops
// itself so it no longer overwrites the shared Outline panel; Activate
// resumes it when the Process Manager view is shown again).
func (p *ProcessManagerPlugin) watchProcesses(stop chan struct{}) {
	for {
		select {
		case <-stop:
			ui.App.QueueUpdateDraw(func() { p.watchRunning = false })
			return
		case <-time.After(processesInterval):
		}
		procs, err := getProcesses()
		active := true
		ui.App.QueueUpdateDraw(func() {
			if edit.CurrentView.Plugin != p {
				p.watchRunning = false
				p.stopWatch = nil
				active = false
				return
			}
			if err != nil {
				ui.SetStatus("Processes: " + err.Error())
				return
			}
			p.processes = procs
			p.populateProcessesTable()
		})
		if !active {
			return
		}
	}
}

// populateProcessesTable rebuilds the processes table from p.processes,
// preserving the current selection when possible.
func (p *ProcessManagerPlugin) populateProcessesTable() {
	row, _ := p.TblProcesses.GetSelection()
	p.TblProcesses.Clear()
	for col, header := range []string{"PID", "User", "CPU%", "MEM%", "State", "Command"} {
		p.TblProcesses.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold).SetSelectable(false))
	}
	p.TblProcesses.SetFixed(1, 0)
	for r, proc := range p.processes {
		color := tcell.ColorWhite
		if strings.HasPrefix(proc.State, "R") {
			color = tcell.ColorGreen
		} else if strings.HasPrefix(proc.State, "Z") {
			color = tcell.ColorRed
		}
		p.TblProcesses.SetCell(r+1, 0, tview.NewTableCell(strconv.Itoa(proc.PID)).SetTextColor(color))
		p.TblProcesses.SetCell(r+1, 1, tview.NewTableCell(proc.User).SetTextColor(color))
		p.TblProcesses.SetCell(r+1, 2, tview.NewTableCell(proc.CPU).SetTextColor(color))
		p.TblProcesses.SetCell(r+1, 3, tview.NewTableCell(proc.MEM).SetTextColor(color))
		p.TblProcesses.SetCell(r+1, 4, tview.NewTableCell(proc.State).SetTextColor(color))
		p.TblProcesses.SetCell(r+1, 5, tview.NewTableCell(proc.Command).SetTextColor(color))
	}
	p.TblProcesses.SetTitle(fmt.Sprintf("Processes (%d) — watching", len(p.processes)))
	if len(p.processes) > 0 {
		if row < 1 {
			row = 1
		}
		if row > len(p.processes) {
			row = len(p.processes)
		}
		p.TblProcesses.Select(row, 0)
	}
	p.RefreshOutline()
}

// SelectedProcess returns the currently selected process, or nil when no
// valid row is selected.
func (p *ProcessManagerPlugin) SelectedProcess() *ProcessInfo {
	row, _ := p.TblProcesses.GetSelection()
	if row <= 0 || row > len(p.processes) {
		return nil
	}
	return &p.processes[row-1]
}

// ConfirmKillSelectedProcess asks for confirmation before sending SIGTERM to
// the currently selected process.
func (p *ProcessManagerPlugin) ConfirmKillSelectedProcess() {
	proc := p.SelectedProcess()
	if proc == nil {
		ui.SetStatus("No process selected")
		return
	}
	p.confirm = p.confirm.YesNo(
		"Kill process",
		fmt.Sprintf("Kill pid %d (%s)?", proc.PID, proc.Command),
		func(button dialog.DlgButton, _ int) {
			if button != dialog.BUTTON_YES {
				return
			}
			p.killProcess(proc.PID, proc.Command)
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgSvcKillProcess", p.confirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSvcKillProcess")
}

// killProcess sends SIGTERM to pid and refreshes the processes table.
func (p *ProcessManagerPlugin) killProcess(pid int, name string) {
	proc, err := os.FindProcess(pid)
	if err == nil {
		err = proc.Signal(syscall.SIGTERM)
	}
	if err != nil {
		ui.SetStatus(fmt.Sprintf("Failed to kill %s (pid %d): %s", name, pid, err.Error()))
		return
	}
	ui.SetStatus(fmt.Sprintf("Sent SIGTERM to %s (pid %d)", name, pid))
	go func() {
		procs, err := getProcesses()
		if err != nil {
			return
		}
		ui.App.QueueUpdateDraw(func() {
			p.processes = procs
			p.populateProcessesTable()
		})
	}()
}

// showProcessesContextMenu shows the context menu available while watching
// showProcessesContextMenu shows the context menu available in the default
// running processes view.
func (p *ProcessManagerPlugin) showProcessesContextMenu() bool {
	m := (&menu.Menu{}).New(" Process Manager ", ui.PopupParentPage(), p.FocusWidget())
	edit.AddOpenViewsMenuItems(m)
	hasProc := p.SelectedProcess() != nil
	m.AddItem("mnuSvcKillProcess", "Kill selected process", func(any) {
		p.ConfirmKillSelectedProcess()
	}, nil, hasProc, false)
	m.AddSeparator()
	m.AddItem("mnuSvcServices", "Services...", func(any) {
		p.ShowServices()
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgProcessManagerMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgProcessManagerMenu")
	return true
}

// HandleInput handles key presses while the default processes view is
// active; callers may handle global keys such as Ctrl+T (close) and F2
// (focus open views) before delegating here.
func (p *ProcessManagerPlugin) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlF:
		ui.App.SetFocus(ui.FrmFind)
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'k':
			p.ConfirmKillSelectedProcess()
			return nil
		case 'v':
			p.ShowServices()
			return nil
		}
	}
	return event
}

// HandleServicesInput handles key presses while the systemd services view is
// active; callers may handle global keys such as Ctrl+T (close) and F2
// (focus open views) before delegating here.
func (p *ProcessManagerPlugin) HandleServicesInput(event *tcell.EventKey) *tcell.EventKey {
	unit := p.SelectedUnit()
	switch event.Key() {
	case tcell.KeyEscape:
		p.closeServices()
		return nil
	case tcell.KeyF5:
		p.Refresh()
		return nil
	case tcell.KeyCtrlF:
		ui.App.SetFocus(ui.FrmFind)
		return nil
	case tcell.KeyEnter:
		if unit != "" {
			p.ShowJournal(unit)
		}
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 's':
			if unit != "" {
				exec.Command("systemctl", "start", unit).Run() //nolint:errcheck
				p.Refresh()
			}
			return nil
		case 'S':
			if unit != "" {
				exec.Command("systemctl", "stop", unit).Run() //nolint:errcheck
				p.Refresh()
			}
			return nil
		case 'r':
			if unit != "" {
				exec.Command("systemctl", "restart", unit).Run() //nolint:errcheck
				p.Refresh()
			}
			return nil
		case 'R':
			p.Refresh()
			return nil
		}
	}
	return event
}

// ****************************************************************************
// Search panel integration
// ****************************************************************************

// matchesUnit reports whether row's unit name contains the given
// case-insensitive substring.
func (p *ProcessManagerPlugin) matchesUnit(row int, needle string) bool {
	cell := p.TblServices.GetCell(row, 0)
	if cell == nil {
		return false
	}
	return strings.Contains(strings.ToLower(cell.Text), needle)
}

// findService selects the next (or, if backward, previous) service row whose
// unit name contains the Find field's text, wrapping around the table.
func (p *ProcessManagerPlugin) findService(backward bool) {
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
func (p *ProcessManagerPlugin) FindNext() { p.findService(false) }

// FindPrevious selects the previous service matching the shared Find field.
func (p *ProcessManagerPlugin) FindPrevious() { p.findService(true) }

// matchesProcess reports whether row's PID or command contains the given
// case-insensitive substring.
func (p *ProcessManagerPlugin) matchesProcess(row int, needle string) bool {
	pidCell := p.TblProcesses.GetCell(row, 0)
	cmdCell := p.TblProcesses.GetCell(row, 5)
	if pidCell == nil || cmdCell == nil {
		return false
	}
	return strings.Contains(pidCell.Text, needle) || strings.Contains(strings.ToLower(cmdCell.Text), needle)
}

// findProcess selects the next (or, if backward, previous) process row whose
// PID or command contains the Find field's text, wrapping around the table.
func (p *ProcessManagerPlugin) findProcess(backward bool) {
	needle := strings.ToLower(strings.TrimSpace(ui.TxtFind.GetText()))
	rowCount := p.TblProcesses.GetRowCount()
	if needle == "" || rowCount <= 1 {
		ui.FrmFind.SetTitle("Find Process")
		ui.SetStatus("Nothing to search")
		return
	}

	total := 0
	for r := 1; r < rowCount; r++ {
		if p.matchesProcess(r, needle) {
			total++
		}
	}
	if total == 0 {
		ui.FrmFind.SetTitle("Find Process (0/0)")
		ui.SetStatus(fmt.Sprintf("No process matching '%s'", needle))
		return
	}

	current, _ := p.TblProcesses.GetSelection()
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
		if p.matchesProcess(r, needle) {
			p.TblProcesses.Select(r, 0)
			p.RefreshOutline()
			ui.FrmFind.SetTitle(fmt.Sprintf("Find Process (%d/%d)", r, total))
			ui.SetStatus(fmt.Sprintf("Found '%s' at pid %s", needle, p.TblProcesses.GetCell(r, 0).Text))
			return
		}
	}
}

// FindNextProcess selects the next process matching the shared Find field.
func (p *ProcessManagerPlugin) FindNextProcess() { p.findProcess(false) }

// FindPreviousProcess selects the previous process matching the shared Find field.
func (p *ProcessManagerPlugin) FindPreviousProcess() { p.findProcess(true) }
