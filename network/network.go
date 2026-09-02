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
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	PluginID        = "networktools"
	ContentPageName = "networkTools"
	UniqueID        = PluginID + "://" + PluginID
	pingCount       = "4"
)

// defaultTargets seeds the target list the first time it is created.
var defaultTargets = []string{"8.8.8.8", "1.1.1.1"}

// NetworkPlugin implements ui.ViewPlugin and shows a list of network targets
// together with an output panel for ping/traceroute/nslookup/dig results.
type NetworkPlugin struct {
	TblTargets  *tview.Table
	TxtOutput   *tview.TextView
	layout      *tview.Flex
	targets     []string
	statuses    []string
	busy        bool
	addDialog   *dialog.Dialog
	pingRunning bool
	stopPing    chan struct{}
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

	ui.PgsEditorContent.AddPage(ContentPageName, p.layout, true, false)

	p.TblTargets.SetSelectionChangedFunc(func(row, column int) {
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
	ui.App.SetFocus(p.TblTargets)
	ui.LblKeys.SetText(p.KeyHints())
}

// FocusWidget returns the targets table as the primary focus target.
func (p *NetworkPlugin) FocusWidget() tview.Primitive { return p.TblTargets }

// Open loads the target list if it hasn't been loaded yet.
func (p *NetworkPlugin) Open(_ any) error {
	if len(p.targets) == 0 {
		p.loadTargets()
		p.populateTable()
	}
	return nil
}

// Close stops any running continuous ping so it doesn't keep pinging in the
// background after the view is closed.
func (p *NetworkPlugin) Close() error {
	p.StopContinuousPing()
	return nil
}
func (p *NetworkPlugin) IsDirty() bool { return false }

func (p *NetworkPlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d target(s)", len(p.targets)),
		Encoding:  "network",
	}
}

func (p *NetworkPlugin) KeyHints() string {
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[a] Add [Del] Remove [Enter] Ping [p] Ping all (Esc to stop) [t] Traceroute [n] nslookup [g] dig [w] whois [Tab] Output [Ctrl+T] Close"
}

func (p *NetworkPlugin) InternalCommand() string       { return "!net" }
func (p *NetworkPlugin) CommandOpensPluginView() bool  { return true }
func (p *NetworkPlugin) ExecuteInternalCommand() error { return nil }

func (p *NetworkPlugin) ShowContextMenu(defaultMenu func()) bool {
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
		}
	}
	return event
}
