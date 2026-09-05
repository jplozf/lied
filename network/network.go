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
// Package network provides a Network Tools plugin for lied.
// It keeps a list of targets (hosts/IPs) and lets the user run ping,
// traceroute, nslookup and dig against one or several of them, all from
// within the editor.
// ****************************************************************************
package network

import (
	"bufio"
	"fmt"
	"lied/conf"
	"lied/dialog"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	PluginID          = "networktools"
	ContentPageName   = "networkTools"
	UniqueID          = PluginID + "://" + PluginID
	pingCount         = "4"
	mainViewName      = "main"
	listeningViewName = "listening"
	listeningInterval = 2 * time.Second
)

// defaultTargets seeds the target list the first time it is created.
var defaultTargets = []string{"8.8.8.8", "1.1.1.1"}

// NetworkPlugin implements ui.ViewPlugin and shows a list of network targets
// together with an output panel for ping/traceroute/nslookup/dig results.
type NetworkPlugin struct {
	TblTargets  *tview.Table
	TxtOutput   *tview.TextView
	views       *tview.Pages
	layout      *tview.Flex
	targets     []string
	statuses    []string
	busy        bool
	addDialog   *dialog.Dialog
	confirm     *dialog.Dialog
	pingRunning bool
	stopPing    chan struct{}

	// Listening ports ("watch") view.
	TblListening    *tview.Table
	listeningLayout *tview.Flex
	listeningActive bool
	watchRunning    bool
	stopWatch       chan struct{}
	listening       []ListeningPort
}

// NewNetworkPlugin creates and wires up the Network Tools plugin.
func NewNetworkPlugin() *NetworkPlugin {
	p := &NetworkPlugin{}

	p.TblTargets = tview.NewTable()
	p.TblTargets.SetBorder(true)
	p.TblTargets.SetTitle("Targets")
	p.TblTargets.SetSelectable(true, false)

	p.TxtOutput = tview.NewTextView()
	p.TxtOutput.SetBorder(true)
	p.TxtOutput.SetTitle("Output")
	p.TxtOutput.SetDynamicColors(true)
	p.TxtOutput.SetScrollable(true)
	p.TxtOutput.SetWrap(true)

	p.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblTargets, 0, 1, true).
		AddItem(p.TxtOutput, 0, 2, false)

	p.TblListening = tview.NewTable()
	p.TblListening.SetBorder(true)
	p.TblListening.SetTitle("Listening Ports")
	p.TblListening.SetSelectable(true, false)

	p.listeningLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblListening, 0, 1, true)

	p.views = tview.NewPages().
		AddPage(mainViewName, p.layout, true, true).
		AddPage(listeningViewName, p.listeningLayout, true, false)

	ui.PgsEditorContent.AddPage(ContentPageName, p.views, true, false)

	p.TblTargets.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})
	p.TblListening.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *NetworkPlugin) ID() string    { return PluginID }
func (p *NetworkPlugin) Title() string { return "Network Tools" }
func (p *NetworkPlugin) Icon() string  { return "🖧" }

// Activate switches the application to the network tools content page.
func (p *NetworkPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	ui.SetFindPanelVisible(false)
	p.RefreshOutline()
	ui.App.SetFocus(p.FocusWidget())
	ui.LblKeys.SetText(p.KeyHints())
}

// FocusWidget returns the primary focus target for the currently active
// sub-view (targets table, or listening ports table when watching).
func (p *NetworkPlugin) FocusWidget() tview.Primitive {
	if p.listeningActive {
		return p.TblListening
	}
	return p.TblTargets
}

// Open loads the target list if it hasn't been loaded yet.
func (p *NetworkPlugin) Open(_ any) error {
	if len(p.targets) == 0 {
		p.loadTargets()
		p.populateTable()
	}
	return nil
}

// Close stops any running continuous ping or listening-ports watch so they
// don't keep running in the background after the view is closed.
func (p *NetworkPlugin) Close() error {
	p.StopContinuousPing()
	p.StopWatchListening()
	return nil
}
func (p *NetworkPlugin) IsDirty() bool { return false }

func (p *NetworkPlugin) StatusFields() ui.ViewStatus {
	if p.listeningActive {
		return ui.ViewStatus{
			ReadWrite: "--",
			Cursor:    fmt.Sprintf("%d listening port(s)", len(p.listening)),
			Encoding:  "network",
		}
	}
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d target(s)", len(p.targets)),
		Encoding:  "network",
	}
}

func (p *NetworkPlugin) KeyHints() string {
	if p.listeningActive {
		return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
			"[Esc] Back [k] Kill process [Ctrl+T] Close"
	}
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[a] Add [Del] Remove [Enter] Ping [p] Ping all (Esc to stop) [t] Traceroute [n] nslookup [g] dig [w] whois [l] Listening ports [Tab] Output [Ctrl+T] Close"
}

func (p *NetworkPlugin) InternalCommand() string       { return "!net" }
func (p *NetworkPlugin) CommandOpensPluginView() bool  { return true }
func (p *NetworkPlugin) ExecuteInternalCommand() error { return nil }

func (p *NetworkPlugin) ShowContextMenu(defaultMenu func()) bool {
	if p.listeningActive {
		return p.showListeningContextMenu()
	}
	m := (&menu.Menu{}).New(" Network Tools ", ui.PopupParentPage(), p.FocusWidget())
	edit.AddOpenViewsMenuItems(m)
	hasTarget := p.SelectedTarget() != ""

	m.AddItem("mnuNetAdd", "Add target...", func(any) {
		p.ShowAddTargetDialog()
	}, nil, true, false)
	m.AddItem("mnuNetRemove", "Remove selected target", func(any) {
		p.RemoveSelectedTarget()
	}, nil, hasTarget, false)
	m.AddSeparator()
	m.AddItem("mnuNetPing", "Ping selected", func(any) {
		p.PingSelected()
	}, nil, hasTarget, false)
	m.AddItem("mnuNetPingAll", "Continuous ping all (Esc to stop)", func(any) {
		p.PingAll()
	}, nil, len(p.targets) > 0, false)
	m.AddItem("mnuNetTraceroute", "Traceroute selected", func(any) {
		p.TracerouteSelected()
	}, nil, hasTarget, false)
	m.AddItem("mnuNetNslookup", "nslookup selected", func(any) {
		p.NslookupSelected()
	}, nil, hasTarget, false)
	m.AddItem("mnuNetDig", "dig selected", func(any) {
		p.DigSelected()
	}, nil, hasTarget, false)
	m.AddItem("mnuNetWhois", "whois selected", func(any) {
		p.WhoisSelected()
	}, nil, hasTarget, false)
	m.AddSeparator()
	m.AddItem("mnuNetListening", "Watch listening ports...", func(any) {
		p.ShowListeningPorts()
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgNetworkMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNetworkMenu")
	return true
}

// ****************************************************************************
// Target list persistence
// ****************************************************************************

// targetsPath returns the path of the user-editable target list.
func (p *NetworkPlugin) targetsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return conf.FILE_NET_TARGETS
	}
	return filepath.Join(home, conf.APP_FOLDER, conf.FILE_NET_TARGETS)
}

// loadTargets reads the target list, seeding it with defaults on first run.
func (p *NetworkPlugin) loadTargets() {
	path := p.targetsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		p.targets = append([]string(nil), defaultTargets...)
		p.saveTargets()
		p.statuses = make([]string, len(p.targets))
		return
	}

	var targets []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		targets = append(targets, line)
	}
	p.targets = targets
	p.statuses = make([]string, len(p.targets))
}

// saveTargets persists the current target list.
func (p *NetworkPlugin) saveTargets() {
	path := p.targetsPath()
	_ = os.WriteFile(path, []byte(strings.Join(p.targets, "\n")+"\n"), 0600)
}

// ****************************************************************************
// Table management
// ****************************************************************************

func (p *NetworkPlugin) populateTable() {
	p.TblTargets.Clear()
	for col, header := range []string{"Target", "Last result"} {
		p.TblTargets.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold).SetSelectable(false))
	}
	p.TblTargets.SetFixed(1, 0)
	for row, target := range p.targets {
		status := ""
		if row < len(p.statuses) {
			status = p.statuses[row]
		}
		color := tcell.ColorWhite
		switch {
		case strings.HasPrefix(status, "OK"):
			color = tcell.ColorGreen
		case strings.HasPrefix(status, "FAIL"):
			color = tcell.ColorRed
		}
		p.TblTargets.SetCell(row+1, 0, tview.NewTableCell(target).SetTextColor(tcell.ColorWhite))
		p.TblTargets.SetCell(row+1, 1, tview.NewTableCell(status).SetTextColor(color))
	}
	p.TblTargets.SetTitle(fmt.Sprintf("Targets (%d)", len(p.targets)))
	if len(p.targets) > 0 {
		row, _ := p.TblTargets.GetSelection()
		if row < 1 {
			p.TblTargets.Select(1, 0)
		}
	}
	p.RefreshOutline()
}

// SelectedTarget returns the currently selected target, or "" when no valid
// row is selected.
func (p *NetworkPlugin) SelectedTarget() string {
	row, _ := p.TblTargets.GetSelection()
	if row <= 0 || row > len(p.targets) {
		return ""
	}
	return p.targets[row-1]
}

func (p *NetworkPlugin) setStatus(target string, status string) {
	for i, t := range p.targets {
		if t == target {
			if i < len(p.statuses) {
				p.statuses[i] = status
			}
			return
		}
	}
}

// ShowAddTargetDialog prompts for a new target hostname/IP.
func (p *NetworkPlugin) ShowAddTargetDialog() {
	p.addDialog = p.addDialog.Input(
		"Add Target",
		"Hostname or IP address:",
		"",
		func(rc dialog.DlgButton, _ int) {
			if rc != dialog.BUTTON_OK {
				return
			}
			target := strings.TrimSpace(p.addDialog.Value)
			if target == "" {
				ui.SetStatus("No target entered")
				return
			}
			for _, existing := range p.targets {
				if existing == target {
					ui.SetStatus("Target already in list")
					return
				}
			}
			p.targets = append(p.targets, target)
			p.statuses = append(p.statuses, "")
			p.saveTargets()
			p.populateTable()
			ui.SetStatus("Added target " + target)
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgNetAddTarget", p.addDialog.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNetAddTarget")
}

// RemoveSelectedTarget deletes the currently selected target from the list.
func (p *NetworkPlugin) RemoveSelectedTarget() {
	row, _ := p.TblTargets.GetSelection()
	if row <= 0 || row > len(p.targets) {
		ui.SetStatus("No target selected")
		return
	}
	removed := p.targets[row-1]
	p.targets = append(p.targets[:row-1], p.targets[row:]...)
	if row-1 < len(p.statuses) {
		p.statuses = append(p.statuses[:row-1], p.statuses[row:]...)
	}
	p.saveTargets()
	p.populateTable()
	ui.SetStatus("Removed target " + removed)
}

// ****************************************************************************
// Outline panel integration
// ****************************************************************************

func (p *NetworkPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	if p.listeningActive {
		p.refreshListeningOutline()
		return
	}
	target := p.SelectedTarget()
	if target == "" {
		ui.TblOutline.SetTitle("Outline")
		return
	}
	row, _ := p.TblTargets.GetSelection()
	status := ""
	if row-1 < len(p.statuses) && row >= 1 {
		status = p.statuses[row-1]
	}
	fields := [][2]string{
		{"Target", target},
		{"Last result", status},
	}
	for r, field := range fields {
		ui.TblOutline.SetCell(r, 0, tview.NewTableCell(field[0]).SetTextColor(tcell.ColorLightCyan))
		ui.TblOutline.SetCell(r, 1, tview.NewTableCell(field[1]).SetTextColor(tcell.ColorWhite))
	}
	ui.TblOutline.SetTitle("Outline — " + target)
}

// refreshListeningOutline shows details for the currently selected listening
// port in the shared Outline panel.
func (p *NetworkPlugin) refreshListeningOutline() {
	port := p.SelectedListeningPort()
	if port == nil {
		ui.TblOutline.SetTitle("Outline")
		return
	}
	fields := [][2]string{
		{"Protocol", port.Proto},
		{"Local address", port.Local},
		{"State", port.State},
		{"PID", strconv.Itoa(port.PID)},
		{"Process", port.Process},
	}
	for r, field := range fields {
		ui.TblOutline.SetCell(r, 0, tview.NewTableCell(field[0]).SetTextColor(tcell.ColorLightCyan))
		ui.TblOutline.SetCell(r, 1, tview.NewTableCell(field[1]).SetTextColor(tcell.ColorWhite))
	}
	ui.TblOutline.SetTitle("Outline — " + port.Local)
}

// ****************************************************************************
// Tool execution
// ****************************************************************************

// runTool executes a command asynchronously and streams its combined output
// into TxtOutput without blocking the UI goroutine.
func (p *NetworkPlugin) runTool(label string, args []string) {
	if p.busy {
		ui.SetStatus("A network operation is already running")
		return
	}
	p.busy = true
	p.TxtOutput.SetTitle(label)
	p.TxtOutput.SetText("Running " + label + "...\n")
	ui.SetStatus("Running " + label)
	go func() {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			p.TxtOutput.SetText(string(out))
			p.TxtOutput.ScrollToBeginning()
			if err != nil {
				ui.SetStatus(label + " failed: " + err.Error())
				return
			}
			ui.SetStatus(label + " completed")
		})
	}()
}

// PingSelected runs a finite `ping -c 4` against the currently selected
// target and reports the full output.
func (p *NetworkPlugin) PingSelected() {
	target := p.SelectedTarget()
	if target == "" {
		ui.SetStatus("No target selected")
		return
	}
	if p.busy || p.pingRunning {
		ui.SetStatus("A network operation is already running")
		return
	}
	p.busy = true
	p.TxtOutput.SetTitle("Ping " + target)
	p.TxtOutput.SetText("Pinging " + target + "...\n")
	ui.SetStatus("Pinging " + target + "...")
	go func() {
		out, err := exec.Command("ping", "-c", pingCount, "-W", "2", target).CombinedOutput()
		status := "OK"
		if err != nil {
			status = "FAIL"
		}
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			p.setStatus(target, status)
			p.updateStatusCell(target, status)
			p.TxtOutput.SetText(string(out))
			p.TxtOutput.ScrollToBeginning()
			ui.SetStatus("Ping " + target + " completed")
		})
	}()
}

// PingAll starts a continuous ping loop against every configured target.
// Each target is pinged once per round; the "Last result" column updates
// live in green (success) or red (failure) as results arrive. Press Esc to
// stop.
func (p *NetworkPlugin) PingAll() {
	if len(p.targets) == 0 {
		ui.SetStatus("No targets configured")
		return
	}
	if p.pingRunning {
		ui.SetStatus("Continuous ping is already running")
		return
	}
	if p.busy {
		ui.SetStatus("A network operation is already running")
		return
	}
	p.pingRunning = true
	p.stopPing = make(chan struct{})
	targets := append([]string(nil), p.targets...)
	p.TxtOutput.SetTitle(fmt.Sprintf("Continuous ping (%d target(s)) — Esc to stop", len(targets)))
	p.TxtOutput.SetText("Continuous ping started. Press Esc to stop.\n")
	ui.SetStatus("Continuous ping started (Esc to stop)")
	go p.continuousPing(targets, p.stopPing)
}

// StopContinuousPing signals the running continuous ping loop to stop.
func (p *NetworkPlugin) StopContinuousPing() {
	if !p.pingRunning || p.stopPing == nil {
		return
	}
	close(p.stopPing)
	p.stopPing = nil
}

// continuousPing pings every target once per round until stop is closed,
// updating each target's status cell live as results arrive.
func (p *NetworkPlugin) continuousPing(targets []string, stop chan struct{}) {
	for {
		for _, target := range targets {
			select {
			case <-stop:
				p.finishContinuousPing()
				return
			default:
			}
			out, err := exec.Command("ping", "-c", "1", "-W", "2", target).CombinedOutput()
			status := "OK"
			if err != nil {
				status = "FAIL"
			}
			result := summarizePing(string(out))
			ui.App.QueueUpdateDraw(func() {
				p.setStatus(target, status)
				p.updateStatusCell(target, status)
				fmt.Fprintf(p.TxtOutput, "%s: %s %s\n", target, status, result)
				p.TxtOutput.ScrollToEnd()
			})
		}
		select {
		case <-stop:
			p.finishContinuousPing()
			return
		case <-time.After(time.Second):
		}
	}
}

// finishContinuousPing updates UI state after the continuous ping loop ends.
func (p *NetworkPlugin) finishContinuousPing() {
	ui.App.QueueUpdateDraw(func() {
		p.pingRunning = false
		p.stopPing = nil
		ui.SetStatus("Continuous ping stopped")
	})
}

// updateStatusCell updates a single target's "Last result" cell in place,
// colored green for success and red for failure, without redrawing the
// whole table (which would reset the current selection).
func (p *NetworkPlugin) updateStatusCell(target string, status string) {
	for row, t := range p.targets {
		if t != target {
			continue
		}
		color := tcell.ColorWhite
		switch status {
		case "OK":
			color = tcell.ColorGreen
		case "FAIL":
			color = tcell.ColorRed
		}
		p.TblTargets.SetCell(row+1, 1, tview.NewTableCell(status).SetTextColor(color))
		if p.TblTargets.GetRowCount() > row+1 {
			row2, _ := p.TblTargets.GetSelection()
			if row2 == row+1 {
				p.RefreshOutline()
			}
		}
		return
	}
}

// summarizePing extracts the packet-loss line from ping output for the
// compact target-table status column.
func summarizePing(output string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "packet loss") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// TracerouteSelected runs traceroute against the selected target.
func (p *NetworkPlugin) TracerouteSelected() {
	target := p.SelectedTarget()
	if target == "" {
		ui.SetStatus("No target selected")
		return
	}
	p.runTool("traceroute "+target, []string{"traceroute", target})
}

// NslookupSelected runs nslookup against the selected target.
func (p *NetworkPlugin) NslookupSelected() {
	target := p.SelectedTarget()
	if target == "" {
		ui.SetStatus("No target selected")
		return
	}
	p.runTool("nslookup "+target, []string{"nslookup", target})
}

// DigSelected runs dig against the selected target.
func (p *NetworkPlugin) DigSelected() {
	target := p.SelectedTarget()
	if target == "" {
		ui.SetStatus("No target selected")
		return
	}
	p.runTool("dig "+target, []string{"dig", target})
}

// WhoisSelected runs whois against the registrable domain of the selected
// target (e.g. "antalis.fr" rather than "www.antalis.fr").
func (p *NetworkPlugin) WhoisSelected() {
	target := p.SelectedTarget()
	if target == "" {
		ui.SetStatus("No target selected")
		return
	}
	domain := extractDomain(target)
	p.runTool("whois "+domain, []string{"whois", domain})
}

// commonSecondLevelSuffixes lists widely used two-part public suffixes so
// extractDomain can keep an extra label for them (e.g. "co.uk").
var commonSecondLevelSuffixes = map[string]bool{
	"co.uk": true, "org.uk": true, "gov.uk": true, "ac.uk": true, "net.uk": true,
	"co.jp": true, "co.nz": true, "co.za": true, "co.in": true, "co.kr": true,
	"com.au": true, "net.au": true, "org.au": true,
	"com.br": true, "com.cn": true, "com.mx": true, "com.tr": true, "com.tw": true,
}

// extractDomain reduces a hostname to its registrable domain (e.g.
// "www.antalis.fr" -> "antalis.fr") using a small list of common two-part
// public suffixes. IP addresses are returned unchanged.
func extractDomain(host string) string {
	host = strings.TrimSpace(host)
	if net.ParseIP(host) != nil {
		return host
	}
	labels := strings.Split(host, ".")
	if len(labels) <= 2 {
		return host
	}
	lastTwo := strings.Join(labels[len(labels)-2:], ".")
	if commonSecondLevelSuffixes[lastTwo] && len(labels) >= 3 {
		return strings.Join(labels[len(labels)-3:], ".")
	}
	return lastTwo
}

// ****************************************************************************
// Public helper used by lied.go keyboard handlers
// ****************************************************************************

// HandleInput handles network-view actions; callers may handle global keys
// such as Ctrl+T (close) and F2 (focus open views) before delegating here.
func (p *NetworkPlugin) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if p.listeningActive {
		return p.handleListeningInput(event)
	}
	switch event.Key() {
	case tcell.KeyEscape:
		if p.pingRunning {
			p.StopContinuousPing()
			return nil
		}
		return event
	case tcell.KeyEnter:
		p.PingSelected()
		return nil
	case tcell.KeyDelete:
		p.RemoveSelectedTarget()
		return nil
	case tcell.KeyTab:
		ui.App.SetFocus(p.TxtOutput)
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'a':
			p.ShowAddTargetDialog()
			return nil
		case 'p':
			p.PingAll()
			return nil
		case 't':
			p.TracerouteSelected()
			return nil
		case 'n':
			p.NslookupSelected()
			return nil
		case 'g':
			p.DigSelected()
			return nil
		case 'w':
			p.WhoisSelected()
			return nil
		case 'l':
			p.ShowListeningPorts()
			return nil
		}
	}
	return event
}

// ****************************************************************************
// Listening ports ("watch") view
// ****************************************************************************

// ListeningPort describes a single listening socket and, when available, the
// process that owns it.
type ListeningPort struct {
	Proto   string
	Local   string
	State   string
	PID     int
	Process string
}

// listeningProcessRE extracts the process name and pid from the "Process"
// column produced by `ss -tulnp`, e.g. `users:(("sshd",pid=1234,fd=3))`.
var listeningProcessRE = regexp.MustCompile(`\(\("([^"]+)",pid=(\d+)`)

// getListeningPorts runs `ss -tulnp` and parses its output into a list of
// ListeningPort entries. Process/PID may be empty when the caller lacks the
// permissions to see the owning process of a given socket.
func getListeningPorts() ([]ListeningPort, error) {
	out, err := exec.Command("ss", "-tulnp").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	var ports []ListeningPort
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue // header line
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		port := ListeningPort{
			Proto: fields[0],
			State: fields[1],
			Local: fields[4],
		}
		if m := listeningProcessRE.FindStringSubmatch(line); m != nil {
			port.Process = m[1]
			port.PID, _ = strconv.Atoi(m[2])
		}
		ports = append(ports, port)
	}
	return ports, nil
}

// ShowListeningPorts switches to the listening-ports sub-view and starts
// periodically refreshing it until StopWatchListening is called (Esc).
func (p *NetworkPlugin) ShowListeningPorts() {
	if p.pingRunning {
		ui.SetStatus("Stop the running ping before watching listening ports")
		return
	}
	p.listeningActive = true
	p.views.SwitchToPage(listeningViewName)
	p.populateListeningTable()
	ui.App.SetFocus(p.TblListening)
	ui.LblKeys.SetText(p.KeyHints())
	p.startWatchListening()
}

// closeListeningPorts stops watching and returns to the main targets view.
func (p *NetworkPlugin) closeListeningPorts() {
	p.StopWatchListening()
	p.listeningActive = false
	p.views.SwitchToPage(mainViewName)
	ui.App.SetFocus(p.TblTargets)
	ui.LblKeys.SetText(p.KeyHints())
	p.RefreshOutline()
}

// startWatchListening launches the background loop that periodically
// refreshes the listening ports table until StopWatchListening is called.
func (p *NetworkPlugin) startWatchListening() {
	if p.watchRunning {
		return
	}
	p.watchRunning = true
	p.stopWatch = make(chan struct{})
	ui.SetStatus("Watching listening ports (Esc to stop)")
	go p.watchListening(p.stopWatch)
}

// StopWatchListening signals the running watch loop to stop.
func (p *NetworkPlugin) StopWatchListening() {
	if !p.watchRunning || p.stopWatch == nil {
		return
	}
	close(p.stopWatch)
	p.stopWatch = nil
}

// watchListening periodically refreshes the listening ports table until stop
// is closed.
func (p *NetworkPlugin) watchListening(stop chan struct{}) {
	for {
		select {
		case <-stop:
			ui.App.QueueUpdateDraw(func() { p.watchRunning = false })
			return
		case <-time.After(listeningInterval):
		}
		ports, err := getListeningPorts()
		ui.App.QueueUpdateDraw(func() {
			if err != nil {
				ui.SetStatus("Listening ports: " + err.Error())
				return
			}
			p.listening = ports
			p.populateListeningTable()
		})
	}
}

// populateListeningTable rebuilds the listening ports table from p.listening,
// preserving the current selection when possible.
func (p *NetworkPlugin) populateListeningTable() {
	row, _ := p.TblListening.GetSelection()
	p.TblListening.Clear()
	for col, header := range []string{"Proto", "Local Address", "State", "PID", "Process"} {
		p.TblListening.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold).SetSelectable(false))
	}
	p.TblListening.SetFixed(1, 0)
	for r, port := range p.listening {
		pid := ""
		if port.PID > 0 {
			pid = strconv.Itoa(port.PID)
		}
		p.TblListening.SetCell(r+1, 0, tview.NewTableCell(port.Proto).SetTextColor(tcell.ColorWhite))
		p.TblListening.SetCell(r+1, 1, tview.NewTableCell(port.Local).SetTextColor(tcell.ColorWhite))
		p.TblListening.SetCell(r+1, 2, tview.NewTableCell(port.State).SetTextColor(tcell.ColorWhite))
		p.TblListening.SetCell(r+1, 3, tview.NewTableCell(pid).SetTextColor(tcell.ColorWhite))
		p.TblListening.SetCell(r+1, 4, tview.NewTableCell(port.Process).SetTextColor(tcell.ColorWhite))
	}
	p.TblListening.SetTitle(fmt.Sprintf("Listening Ports (%d) — watching", len(p.listening)))
	if len(p.listening) > 0 {
		if row < 1 {
			row = 1
		}
		if row > len(p.listening) {
			row = len(p.listening)
		}
		p.TblListening.Select(row, 0)
	}
	p.RefreshOutline()
}

// SelectedListeningPort returns the currently selected listening port, or nil
// when no valid row is selected.
func (p *NetworkPlugin) SelectedListeningPort() *ListeningPort {
	row, _ := p.TblListening.GetSelection()
	if row <= 0 || row > len(p.listening) {
		return nil
	}
	return &p.listening[row-1]
}

// ConfirmKillSelectedProcess asks for confirmation before sending SIGTERM to
// the process owning the selected listening port.
func (p *NetworkPlugin) ConfirmKillSelectedProcess() {
	port := p.SelectedListeningPort()
	if port == nil {
		ui.SetStatus("No listening port selected")
		return
	}
	if port.PID <= 0 {
		ui.SetStatus("No process information for this port (insufficient permissions?)")
		return
	}
	p.confirm = p.confirm.YesNo(
		"Kill process",
		fmt.Sprintf("Kill %s (pid %d) listening on %s?", port.Process, port.PID, port.Local),
		func(button dialog.DlgButton, _ int) {
			if button != dialog.BUTTON_YES {
				return
			}
			p.killProcess(port.PID, port.Process)
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgNetKillProcess", p.confirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNetKillProcess")
}

// killProcess sends SIGTERM to pid and refreshes the listening ports table.
func (p *NetworkPlugin) killProcess(pid int, name string) {
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
		ports, err := getListeningPorts()
		if err != nil {
			return
		}
		ui.App.QueueUpdateDraw(func() {
			p.listening = ports
			p.populateListeningTable()
		})
	}()
}

// showListeningContextMenu shows the context menu available while watching
// listening ports.
func (p *NetworkPlugin) showListeningContextMenu() bool {
	m := (&menu.Menu{}).New(" Listening Ports ", ui.PopupParentPage(), p.FocusWidget())
	hasPort := p.SelectedListeningPort() != nil
	m.AddItem("mnuNetKill", "Kill selected process", func(any) {
		p.ConfirmKillSelectedProcess()
	}, nil, hasPort, false)
	m.AddItem("mnuNetBack", "Back to targets", func(any) {
		p.closeListeningPorts()
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgNetworkListeningMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNetworkListeningMenu")
	return true
}

// handleListeningInput handles key presses while the listening-ports view is
// active.
func (p *NetworkPlugin) handleListeningInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		p.closeListeningPorts()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'k':
			p.ConfirmKillSelectedProcess()
			return nil
		}
	}
	return event
}
