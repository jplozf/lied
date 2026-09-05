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
// Package security provides a Security Manager plugin for lied.
// It manages the host firewall (ufw or firewalld, whichever is installed) —
// showing rules and allowing them to be added, removed, enabled or disabled —
// and ClamAV — showing installation status and letting the user scan a path
// or update virus definitions.
// ****************************************************************************
package security

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"fmt"
	"lied/conf"
	"lied/dialog"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	PluginID        = "securitymanager"
	ContentPageName = "securityManager"
	// UniqueID is used as the synthetic FName when this plugin is open in the
	// views list so that duplicate opens can be detected.
	UniqueID = PluginID + "://" + PluginID

	firewallViewName = "firewall"
	clamavViewName   = "clamav"
)

// ****************************************************************************
// TYPES
// ****************************************************************************

// FirewallRule is a single firewall rule, either a numbered ufw rule or a
// firewalld allowed port/service.
type FirewallRule struct {
	Num    string
	To     string
	Action string
	From   string
	// ID is a backend-specific identifier used to delete this rule (e.g. the
	// ufw rule number, or "port:8080/tcp" / "service:http" for firewalld).
	// ufw rules use Num directly instead.
	ID string
}

// RuleSpec describes a firewall rule to add, gathered from the Add Rule
// form.
type RuleSpec struct {
	// Action is one of the backend's Actions(), e.g. "allow", "deny", "reject".
	Action string
	// Target is a port/protocol (e.g. "22/tcp") or a service name (e.g. "http").
	Target string
	// From is the source address/CIDR; empty or "any" means anywhere.
	From string
	// To is the destination address/CIDR (ufw only); empty or "any" means
	// anywhere.
	To string
}

// clamavStatusRE extracts the ClamAV engine version and database date from
// `clamscan --version` output, e.g. "ClamAV 1.0.0/27000/Mon Jan 1 00:00:00 2024".
var clamavStatusRE = regexp.MustCompile(`ClamAV\s+([\d.]+)/(\d+)/(.+)`)

// ufwRuleRE matches a numbered ufw rule line, e.g. "[ 1] 22/tcp ...".
var ufwRuleRE = regexp.MustCompile(`^\[\s*(\d+)\]\s+(.*)$`)

// ufwColumnsRE splits the remainder of a rule line into To/Action/From
// columns, which ufw separates with two or more spaces.
var ufwColumnsRE = regexp.MustCompile(`\s{2,}`)

// clamavDateLayout is the ctime-like format ClamAV reports its database
// timestamp in, e.g. "Wed Sep  3 08:12:00 2025".
const clamavDateLayout = "Mon Jan _2 15:04:05 2006"

// clamavStatusRow is a single labelled status line in the ClamAV table, with
// an explicit display color.
type clamavStatusRow struct {
	Label string
	Value string
	Color tcell.Color
}

// SecurityPlugin implements ui.ViewPlugin and shows firewall rules together
// with a ClamAV status/scan sub-view.
type SecurityPlugin struct {
	// TblFirewall lists the current firewall rules.
	TblFirewall *tview.Table
	// TblClamav shows ClamAV component status.
	TblClamav *tview.Table
	// TxtOutput shows the result of the last command run in either view.
	TxtOutput *tview.TextView

	views          *tview.Pages
	firewallLayout *tview.Flex
	clamavLayout   *tview.Flex

	// fw is the detected firewall backend (ufw or firewalld), or nil when
	// neither is installed.
	fw             firewallBackend
	firewallActive bool
	firewallOK     bool
	rules          []FirewallRule

	clamavActive bool
	clamavStatus []clamavStatusRow

	busy bool

	scanDialog *dialog.Dialog
	confirm    *dialog.Dialog
}

// NewSecurityPlugin creates and wires up the Security Manager plugin.
func NewSecurityPlugin() *SecurityPlugin {
	p := &SecurityPlugin{}

	p.TblFirewall = tview.NewTable()
	p.TblFirewall.SetBorder(true)
	p.TblFirewall.SetTitle("Firewall Rules")
	p.TblFirewall.SetSelectable(true, false)

	p.TblClamav = tview.NewTable()
	p.TblClamav.SetBorder(true)
	p.TblClamav.SetTitle("ClamAV")
	p.TblClamav.SetSelectable(true, false)

	p.TxtOutput = tview.NewTextView()
	p.TxtOutput.SetBorder(true)
	p.TxtOutput.SetTitle("Output")
	p.TxtOutput.SetDynamicColors(true)
	p.TxtOutput.SetScrollable(true)
	p.TxtOutput.SetWrap(true)

	p.firewallLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblFirewall, 0, 1, true).
		AddItem(p.TxtOutput, 0, 1, false)

	p.clamavLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblClamav, 0, 1, true).
		AddItem(p.TxtOutput, 0, 1, false)

	p.views = tview.NewPages().
		AddPage(firewallViewName, p.firewallLayout, true, true).
		AddPage(clamavViewName, p.clamavLayout, true, false)

	ui.PgsEditorContent.AddPage(ContentPageName, p.views, true, false)

	p.TblFirewall.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})
	p.TblClamav.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *SecurityPlugin) ID() string    { return PluginID }
func (p *SecurityPlugin) Title() string { return "Security Manager" }
func (p *SecurityPlugin) Icon() string  { return "🛡" }

// Activate switches the application to the security manager content page.
func (p *SecurityPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	ui.SetFindPanelVisible(false)
	p.RefreshOutline()
	ui.App.SetFocus(p.FocusWidget())
	ui.LblKeys.SetText(p.KeyHints())
}

// FocusWidget returns the primary focus target for the currently active
// sub-view (firewall rules table, or ClamAV table).
func (p *SecurityPlugin) FocusWidget() tview.Primitive {
	if p.clamavActive {
		return p.TblClamav
	}
	return p.TblFirewall
}

// Open refreshes the firewall rules the first time the view is opened.
func (p *SecurityPlugin) Open(_ any) error {
	if len(p.rules) == 0 && p.TblFirewall.GetRowCount() == 0 {
		p.RefreshFirewall()
	}
	return nil
}

// Close is a no-op; the security manager holds no resources that need cleanup.
func (p *SecurityPlugin) Close() error { return nil }

// IsDirty always returns false because the security manager is read-only.
func (p *SecurityPlugin) IsDirty() bool { return false }

func (p *SecurityPlugin) StatusFields() ui.ViewStatus {
	if p.clamavActive {
		return ui.ViewStatus{
			ReadWrite: "--",
			Cursor:    "ClamAV",
			Encoding:  "security",
		}
	}
	status := "inactive"
	if p.firewallOK {
		status = "active"
	}
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("Firewall %s, %d rule(s)", status, len(p.rules)),
		Encoding:  "security",
	}
}

func (p *SecurityPlugin) KeyHints() string {
	if p.clamavActive {
		return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
			"[s] Scan path  [u] Update definitions  [F5] Refresh  [Esc] Back  [Ctrl+T] Close"
	}
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[a] Add rule  [Del] Delete rule  [e] Enable  [d] Disable  [c] ClamAV  [F5] Refresh  [Ctrl+T] Close"
}

func (p *SecurityPlugin) InternalCommand() string       { return "!sec" }
func (p *SecurityPlugin) CommandOpensPluginView() bool  { return true }
func (p *SecurityPlugin) ExecuteInternalCommand() error { return nil }

func (p *SecurityPlugin) ShowContextMenu(defaultMenu func()) bool {
	if p.clamavActive {
		return p.showClamavContextMenu()
	}
	m := (&menu.Menu{}).New(" Security Manager ", ui.PopupParentPage(), p.FocusWidget())
	edit.AddOpenViewsMenuItems(m)
	hasRule := p.SelectedRule() != nil

	m.AddItem("mnuSecRefresh", "Refresh firewall rules", func(any) {
		p.RefreshFirewall()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuSecAddRule", "Add rule...", func(any) {
		p.ShowAddRuleDialog()
	}, nil, true, false)
	m.AddItem("mnuSecDeleteRule", "Delete selected rule", func(any) {
		p.ConfirmDeleteSelectedRule()
	}, nil, hasRule, false)
	m.AddSeparator()
	m.AddItem("mnuSecEnable", "Enable firewall", func(any) {
		p.EnableFirewall()
	}, nil, true, false)
	m.AddItem("mnuSecDisable", "Disable firewall", func(any) {
		p.ConfirmDisableFirewall()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuSecClamav", "ClamAV...", func(any) {
		p.ShowClamav()
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgSecurityMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSecurityMenu")
	return true
}

// ****************************************************************************
// Firewall management (ufw or firewalld)
// ****************************************************************************

// firewallBackend abstracts the host firewall manager so the plugin works
// with either ufw (Debian/Ubuntu) or firewalld (Fedora/RHEL and friends).
type firewallBackend interface {
	// Name returns the backend's display name, e.g. "ufw" or "firewalld".
	Name() string
	// Refresh returns whether the firewall is active and its current rules;
	// raw holds the last command's raw output for diagnostics on error.
	Refresh() (active bool, rules []FirewallRule, raw string, err error)
	// Actions lists the rule actions this backend supports, for the Add Rule
	// form's Action dropdown.
	Actions() []string
	// TargetHint is shown next to the port/service field in the Add Rule form.
	TargetHint() string
	// SupportsDestination reports whether this backend can restrict a rule to
	// a destination address (ufw can; firewalld rich rules only filter source).
	SupportsDestination() bool
	// AddRuleSteps returns the command(s) needed to add spec.
	AddRuleSteps(spec RuleSpec) [][]string
	// DeleteSteps returns the command(s) needed to remove rule.
	DeleteSteps(rule FirewallRule) [][]string
	// EnableSteps/DisableSteps turn the firewall on/off.
	EnableSteps() [][]string
	DisableSteps() [][]string
}

// detectFirewallBackend picks whichever supported firewall manager is
// installed, preferring ufw when both happen to be present.
func detectFirewallBackend() firewallBackend {
	if _, err := exec.LookPath("ufw"); err == nil {
		return ufwBackend{}
	}
	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		return firewalldBackend{}
	}
	return nil
}

// ---- ufw ----

type ufwBackend struct{}

func (ufwBackend) Name() string { return "ufw" }

// parseUfwStatus parses the output of `ufw status numbered` into an active
// flag and a list of rules.
func parseUfwStatus(output string) (bool, []FirewallRule) {
	active := strings.Contains(output, "Status: active")
	var rules []FirewallRule
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, " \r")
		m := ufwRuleRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		cols := ufwColumnsRE.Split(strings.TrimSpace(m[2]), -1)
		rule := FirewallRule{Num: m[1]}
		if len(cols) > 0 {
			rule.To = cols[0]
		}
		if len(cols) > 1 {
			rule.Action = cols[1]
		}
		if len(cols) > 2 {
			rule.From = strings.Join(cols[2:], " ")
		}
		rules = append(rules, rule)
	}
	return active, rules
}

func (ufwBackend) Refresh() (bool, []FirewallRule, string, error) {
	out, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if err != nil {
		return false, nil, string(out), err
	}
	active, rules := parseUfwStatus(string(out))
	return active, rules, string(out), nil
}

func (ufwBackend) Actions() []string { return []string{"allow", "deny", "reject", "limit"} }

func (ufwBackend) TargetHint() string { return "e.g. 22/tcp, 80, or http" }

func (ufwBackend) SupportsDestination() bool { return true }

// AddRuleSteps builds a ufw command from spec, using the simple
// "ACTION TARGET" form when no source/destination is given, or the extended
// "ACTION [proto P] from FROM to TO [port PORT]" form otherwise.
func (ufwBackend) AddRuleSteps(spec RuleSpec) [][]string {
	target := strings.TrimSpace(spec.Target)
	from := strings.TrimSpace(spec.From)
	to := strings.TrimSpace(spec.To)
	args := []string{"sudo", "-n", "ufw"}
	if from == "" && (to == "" || strings.EqualFold(to, "any")) {
		args = append(args, spec.Action)
		args = append(args, strings.Fields(target)...)
		return [][]string{args}
	}
	if from == "" {
		from = "any"
	}
	if to == "" {
		to = "any"
	}
	port, proto, _ := strings.Cut(target, "/")
	args = append(args, spec.Action)
	if proto != "" {
		args = append(args, "proto", proto)
	}
	args = append(args, "from", from, "to", to)
	if port != "" {
		args = append(args, "port", port)
	}
	return [][]string{args}
}

func (ufwBackend) DeleteSteps(rule FirewallRule) [][]string {
	return [][]string{{"sudo", "-n", "ufw", "--force", "delete", rule.Num}}
}

func (ufwBackend) EnableSteps() [][]string {
	return [][]string{{"sudo", "-n", "ufw", "--force", "enable"}}
}

func (ufwBackend) DisableSteps() [][]string {
	return [][]string{{"sudo", "-n", "ufw", "disable"}}
}

// ---- firewalld ----

type firewalldBackend struct{}

func (firewalldBackend) Name() string { return "firewalld" }

// firewalldPortRE matches a firewalld port/protocol token, e.g. "8080/tcp".
var firewalldPortRE = regexp.MustCompile(`^\d+(-\d+)?/(tcp|udp)$`)

// parseFirewalldStatus parses the "services:" and "ports:" lines from
// `firewall-cmd --list-all` into a list of allowed rules.
func parseFirewalldStatus(output string) []FirewallRule {
	var rules []FirewallRule
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "services:"):
			for _, name := range strings.Fields(strings.TrimPrefix(trimmed, "services:")) {
				rules = append(rules, FirewallRule{To: name, Action: "ALLOW", From: "Anywhere", ID: "service:" + name})
			}
		case strings.HasPrefix(trimmed, "ports:"):
			for _, port := range strings.Fields(strings.TrimPrefix(trimmed, "ports:")) {
				rules = append(rules, FirewallRule{To: port, Action: "ALLOW", From: "Anywhere", ID: "port:" + port})
			}
		}
	}
	for i := range rules {
		rules[i].Num = strconv.Itoa(i + 1)
	}
	return rules
}

func (firewalldBackend) Refresh() (bool, []FirewallRule, string, error) {
	stateOut, stateErr := exec.Command("firewall-cmd", "--state").CombinedOutput()
	active := stateErr == nil && strings.TrimSpace(string(stateOut)) == "running"
	if !active {
		return false, nil, string(stateOut), nil
	}
	out, err := exec.Command("firewall-cmd", "--list-all").CombinedOutput()
	if err != nil {
		return active, nil, string(out), err
	}
	return active, parseFirewalldStatus(string(out)), string(out), nil
}

func (firewalldBackend) Actions() []string { return []string{"allow", "deny", "reject"} }

func (firewalldBackend) TargetHint() string { return "e.g. 8080/tcp or http" }

func (firewalldBackend) SupportsDestination() bool { return false }

// AddRuleSteps adds spec via the simple --add-port/--add-service flags when
// it's an unrestricted allow rule, or via a permanent rich rule otherwise
// (needed to express a source restriction or a deny/reject action).
func (firewalldBackend) AddRuleSteps(spec RuleSpec) [][]string {
	target := strings.TrimSpace(spec.Target)
	from := strings.TrimSpace(spec.From)
	isPort := firewalldPortRE.MatchString(target)

	if spec.Action == "allow" && (from == "" || strings.EqualFold(from, "any")) {
		addFlag := "--add-service=" + target
		if isPort {
			addFlag = "--add-port=" + target
		}
		return [][]string{
			{"sudo", "-n", "firewall-cmd", "--permanent", addFlag},
			{"sudo", "-n", "firewall-cmd", "--reload"},
		}
	}

	verb := "accept"
	if spec.Action == "deny" || spec.Action == "reject" {
		verb = "reject"
	}
	var rule strings.Builder
	rule.WriteString(`rule family="ipv4"`)
	if from != "" && !strings.EqualFold(from, "any") {
		fmt.Fprintf(&rule, ` source address="%s"`, from)
	}
	if isPort {
		port, proto, _ := strings.Cut(target, "/")
		fmt.Fprintf(&rule, ` port port="%s" protocol="%s"`, port, proto)
	} else {
		fmt.Fprintf(&rule, ` service name="%s"`, target)
	}
	fmt.Fprintf(&rule, " %s", verb)
	return [][]string{
		{"sudo", "-n", "firewall-cmd", "--permanent", "--add-rich-rule=" + rule.String()},
		{"sudo", "-n", "firewall-cmd", "--reload"},
	}
}

func (firewalldBackend) DeleteSteps(rule FirewallRule) [][]string {
	kind, value, ok := strings.Cut(rule.ID, ":")
	if !ok {
		return nil
	}
	removeFlag := "--remove-service=" + value
	if kind == "port" {
		removeFlag = "--remove-port=" + value
	}
	return [][]string{
		{"sudo", "-n", "firewall-cmd", "--permanent", removeFlag},
		{"sudo", "-n", "firewall-cmd", "--reload"},
	}
}

func (firewalldBackend) EnableSteps() [][]string {
	return [][]string{{"sudo", "-n", "systemctl", "enable", "--now", "firewalld"}}
}

func (firewalldBackend) DisableSteps() [][]string {
	return [][]string{{"sudo", "-n", "systemctl", "disable", "--now", "firewalld"}}
}

// RefreshFirewall reloads the firewall rule list using whichever backend
// (ufw or firewalld) is installed.
func (p *SecurityPlugin) RefreshFirewall() {
	if p.busy {
		ui.SetStatus("A security operation is already running")
		return
	}
	if p.fw == nil {
		p.fw = detectFirewallBackend()
	}
	if p.fw == nil {
		p.rules = nil
		p.firewallOK = false
		p.populateFirewallTable()
		p.TxtOutput.SetTitle("Firewall")
		p.TxtOutput.SetText("No supported firewall manager found (looked for ufw and firewalld).")
		ui.SetStatus("No firewall manager found")
		return
	}
	p.busy = true
	ui.SetStatus("Reading firewall status...")
	go func() {
		active, rules, raw, err := p.fw.Refresh()
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			if err != nil {
				p.rules = nil
				p.firewallOK = false
				p.populateFirewallTable()
				p.TxtOutput.SetTitle(p.fw.Name() + " status")
				p.TxtOutput.SetText(raw + "\nError: " + err.Error())
				ui.SetStatus(p.fw.Name() + " not available: " + err.Error())
				return
			}
			p.firewallOK, p.rules = active, rules
			p.populateFirewallTable()
			ui.SetStatus("Firewall status refreshed (" + p.fw.Name() + ")")
		})
	}()
}

// populateFirewallTable rebuilds the firewall rules table from p.rules,
// preserving the current selection when possible.
func (p *SecurityPlugin) populateFirewallTable() {
	row, _ := p.TblFirewall.GetSelection()
	p.TblFirewall.Clear()
	for col, header := range []string{"#", "To", "Action", "From"} {
		p.TblFirewall.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold).SetSelectable(false))
	}
	p.TblFirewall.SetFixed(1, 0)
	for r, rule := range p.rules {
		color := tcell.ColorWhite
		switch {
		case strings.Contains(rule.Action, "ALLOW"):
			color = tcell.ColorGreen
		case strings.Contains(rule.Action, "DENY"), strings.Contains(rule.Action, "REJECT"):
			color = tcell.ColorRed
		case strings.Contains(rule.Action, "LIMIT"):
			color = tcell.ColorYellow
		}
		p.TblFirewall.SetCell(r+1, 0, tview.NewTableCell(rule.Num).SetTextColor(color))
		p.TblFirewall.SetCell(r+1, 1, tview.NewTableCell(rule.To).SetTextColor(color))
		p.TblFirewall.SetCell(r+1, 2, tview.NewTableCell(rule.Action).SetTextColor(color))
		p.TblFirewall.SetCell(r+1, 3, tview.NewTableCell(rule.From).SetTextColor(color))
	}
	status := "inactive"
	if p.firewallOK {
		status = "active"
	}
	backend := "no backend"
	if p.fw != nil {
		backend = p.fw.Name()
	}
	p.TblFirewall.SetTitle(fmt.Sprintf("Firewall Rules — %s (%s, %d)", backend, status, len(p.rules)))
	if len(p.rules) > 0 {
		if row < 1 {
			row = 1
		}
		if row > len(p.rules) {
			row = len(p.rules)
		}
		p.TblFirewall.Select(row, 0)
	}
	p.RefreshOutline()
}

// SelectedRule returns the currently selected firewall rule, or nil when no
// valid row is selected.
func (p *SecurityPlugin) SelectedRule() *FirewallRule {
	row, _ := p.TblFirewall.GetSelection()
	if row <= 0 || row > len(p.rules) {
		return nil
	}
	return &p.rules[row-1]
}

// ShowAddRuleDialog opens a form to add a rule, with fields adapted to the
// detected firewall backend (action, port/service, source and, for ufw,
// destination).
func (p *SecurityPlugin) ShowAddRuleDialog() {
	fw := p.fw
	if fw == nil {
		ui.SetStatus("No firewall manager found")
		return
	}

	action := tview.NewDropDown().SetLabel("Action: ").SetOptions(fw.Actions(), nil).SetCurrentOption(0)
	target := tview.NewInputField().SetLabel("Port/Service (" + fw.TargetHint() + "): ").SetFieldWidth(24)
	from := tview.NewInputField().SetLabel("Source (blank = any): ").SetFieldWidth(24)

	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Add Firewall Rule ")
	form.AddFormItem(action)
	form.AddFormItem(target)
	form.AddFormItem(from)

	var to *tview.InputField
	if fw.SupportsDestination() {
		to = tview.NewInputField().SetLabel("Destination (blank = any): ").SetFieldWidth(24)
		form.AddFormItem(to)
	}

	form.AddButton("Add", func() {
		targetText := strings.TrimSpace(target.GetText())
		if targetText == "" {
			ui.SetStatus("No port/service entered")
			return
		}
		_, actionText := action.GetCurrentOption()
		spec := RuleSpec{
			Action: actionText,
			Target: targetText,
			From:   strings.TrimSpace(from.GetText()),
		}
		if to != nil {
			spec.To = strings.TrimSpace(to.GetText())
		}
		ui.PgsApp.HidePage("dlgSecAddRule")
		ui.App.SetFocus(p.FocusWidget())
		p.runFirewallSteps("Add rule", fw.AddRuleSteps(spec))
	})
	form.AddButton("Cancel", func() {
		ui.PgsApp.HidePage("dlgSecAddRule")
		ui.App.SetFocus(p.FocusWidget())
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.PgsApp.HidePage("dlgSecAddRule")
			ui.App.SetFocus(p.FocusWidget())
			return nil
		}
		return event
	})

	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(form, 70, 0, true).
		AddItem(nil, 0, 1, false)
	ui.PgsApp.AddPage("dlgSecAddRule", centered, true, true)
	ui.App.SetFocus(target)
}

// ConfirmDeleteSelectedRule asks for confirmation before deleting the
// selected firewall rule.
func (p *SecurityPlugin) ConfirmDeleteSelectedRule() {
	if p.fw == nil {
		ui.SetStatus("No firewall manager found")
		return
	}
	rule := p.SelectedRule()
	if rule == nil {
		ui.SetStatus("No rule selected")
		return
	}
	p.confirm = p.confirm.YesNo(
		"Delete Firewall Rule",
		fmt.Sprintf("Delete rule #%s (%s %s %s)?", rule.Num, rule.To, rule.Action, rule.From),
		func(button dialog.DlgButton, _ int) {
			if button != dialog.BUTTON_YES {
				return
			}
			p.runFirewallSteps("Delete rule", p.fw.DeleteSteps(*rule))
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgSecDeleteRule", p.confirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSecDeleteRule")
}

// EnableFirewall turns the detected firewall backend on.
func (p *SecurityPlugin) EnableFirewall() {
	if p.fw == nil {
		ui.SetStatus("No firewall manager found")
		return
	}
	p.runFirewallSteps("Enable firewall", p.fw.EnableSteps())
}

// ConfirmDisableFirewall asks for confirmation before turning the firewall
// off.
func (p *SecurityPlugin) ConfirmDisableFirewall() {
	if p.fw == nil {
		ui.SetStatus("No firewall manager found")
		return
	}
	p.confirm = p.confirm.YesNo(
		"Disable Firewall",
		"Disable the firewall?",
		func(button dialog.DlgButton, _ int) {
			if button != dialog.BUTTON_YES {
				return
			}
			p.runFirewallSteps("Disable firewall", p.fw.DisableSteps())
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgSecDisableFirewall", p.confirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSecDisableFirewall")
}

// runFirewallSteps executes a sequence of commands asynchronously (stopping
// at the first failure), shows their combined output and refreshes the rule
// list.
func (p *SecurityPlugin) runFirewallSteps(label string, steps [][]string) {
	if p.busy {
		ui.SetStatus("A security operation is already running")
		return
	}
	if len(steps) == 0 {
		ui.SetStatus(label + ": nothing to do")
		return
	}
	p.busy = true
	p.TxtOutput.SetTitle(label)
	p.TxtOutput.SetText("Running " + label + "...\n")
	ui.SetStatus("Running " + label)
	go func() {
		var combined strings.Builder
		var stepErr error
		for _, args := range steps {
			out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
			combined.Write(out)
			combined.WriteString("\n")
			if err != nil {
				stepErr = err
				break
			}
		}
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			p.TxtOutput.SetText(combined.String())
			p.TxtOutput.ScrollToBeginning()
			if stepErr != nil {
				ui.SetStatus(label + " failed: " + stepErr.Error())
				return
			}
			ui.SetStatus(label + " completed")
			p.RefreshFirewall()
		})
	}()
}

// ****************************************************************************
// ClamAV management
// ****************************************************************************

// ShowClamav switches to the ClamAV sub-view and refreshes its status.
func (p *SecurityPlugin) ShowClamav() {
	p.clamavActive = true
	p.views.SwitchToPage(clamavViewName)
	p.RefreshClamav()
	ui.App.SetFocus(p.TblClamav)
	ui.LblKeys.SetText(p.KeyHints())
}

// closeClamav returns to the firewall view.
func (p *SecurityPlugin) closeClamav() {
	p.clamavActive = false
	p.views.SwitchToPage(firewallViewName)
	ui.App.SetFocus(p.TblFirewall)
	ui.LblKeys.SetText(p.KeyHints())
	p.RefreshOutline()
}

// RefreshClamav re-checks ClamAV component availability and status.
func (p *SecurityPlugin) RefreshClamav() {
	rows := make([]clamavStatusRow, 0, 5)

	if path, err := exec.LookPath("clamscan"); err == nil {
		version := "installed"
		var dbDate string
		if out, verr := exec.Command("clamscan", "--version").Output(); verr == nil {
			if m := clamavStatusRE.FindStringSubmatch(strings.TrimSpace(string(out))); m != nil {
				dbDate = strings.TrimSpace(m[3])
				version = fmt.Sprintf("engine %s, db %s (%s)", m[1], m[2], dbDate)
			}
		}
		rows = append(rows, clamavStatusRow{"clamscan", version + " — " + path, tcell.ColorWhite})
		rows = append(rows, clamavDatabaseRow(dbDate))
	} else {
		rows = append(rows, clamavStatusRow{"clamscan", "not installed", tcell.ColorRed})
	}

	if path, err := exec.LookPath("freshclam"); err == nil {
		rows = append(rows, clamavStatusRow{"freshclam", "installed — " + path, tcell.ColorWhite})
	} else {
		rows = append(rows, clamavStatusRow{"freshclam", "not installed", tcell.ColorRed})
	}

	daemonStatus := "unknown"
	if out, err := exec.Command("systemctl", "is-active", "clamav-daemon").Output(); err == nil || len(out) > 0 {
		daemonStatus = strings.TrimSpace(string(out))
	}
	daemonColor := tcell.ColorWhite
	switch daemonStatus {
	case "active":
		daemonColor = tcell.ColorGreen
	case "not installed", "inactive", "failed":
		daemonColor = tcell.ColorRed
	}
	rows = append(rows, clamavStatusRow{"clamav-daemon", daemonStatus, daemonColor})

	p.clamavStatus = rows
	p.populateClamavTable()
	ui.SetStatus("ClamAV status refreshed")
}

// clamavDatabaseRow builds the "Database" status row from the raw timestamp
// reported by `clamscan --version`, colored green when updated today, yellow
// when up to 2 days old, and red otherwise (or when the date can't be parsed).
func clamavDatabaseRow(dbDate string) clamavStatusRow {
	if dbDate == "" {
		return clamavStatusRow{"Database", "unknown", tcell.ColorRed}
	}
	updated, err := time.ParseInLocation(clamavDateLayout, dbDate, time.Local)
	if err != nil {
		return clamavStatusRow{"Database", dbDate, tcell.ColorRed}
	}
	age := time.Since(updated)
	color := tcell.ColorRed
	switch {
	case age < 24*time.Hour:
		color = tcell.ColorGreen
	case age < 2*24*time.Hour:
		color = tcell.ColorYellow
	}
	return clamavStatusRow{"Database", dbDate, color}
}

// populateClamavTable rebuilds the ClamAV status table from p.clamavStatus.
func (p *SecurityPlugin) populateClamavTable() {
	row, _ := p.TblClamav.GetSelection()
	p.TblClamav.Clear()
	for col, header := range []string{"Component", "Status"} {
		p.TblClamav.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold).SetSelectable(false))
	}
	p.TblClamav.SetFixed(1, 0)
	for r, status := range p.clamavStatus {
		p.TblClamav.SetCell(r+1, 0, tview.NewTableCell(status.Label).SetTextColor(tcell.ColorWhite))
		p.TblClamav.SetCell(r+1, 1, tview.NewTableCell(status.Value).SetTextColor(status.Color))
	}
	p.TblClamav.SetTitle(fmt.Sprintf("ClamAV (%d)", len(p.clamavStatus)))
	if len(p.clamavStatus) > 0 {
		if row < 1 {
			row = 1
		}
		if row > len(p.clamavStatus) {
			row = len(p.clamavStatus)
		}
		p.TblClamav.Select(row, 0)
	}
	p.RefreshOutline()
}

// ShowScanPathDialog prompts for a path to scan with clamscan.
func (p *SecurityPlugin) ShowScanPathDialog() {
	def := conf.ConfigGeneral.Workspace
	p.scanDialog = p.scanDialog.Input(
		"ClamAV Scan",
		"Path to scan:",
		def,
		func(rc dialog.DlgButton, _ int) {
			if rc != dialog.BUTTON_OK {
				return
			}
			path := strings.TrimSpace(p.scanDialog.Value)
			if path == "" {
				ui.SetStatus("No path entered")
				return
			}
			p.RunScan(path)
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgSecScanPath", p.scanDialog.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSecScanPath")
}

// RunScan runs `clamscan -r -i` against path and shows the results.
func (p *SecurityPlugin) RunScan(path string) {
	if p.busy {
		ui.SetStatus("A security operation is already running")
		return
	}
	p.busy = true
	p.TxtOutput.SetTitle("Scan " + path)
	p.TxtOutput.SetText("Scanning " + path + "...\n")
	ui.SetStatus("Scanning " + path + "...")
	go func() {
		out, err := exec.Command("clamscan", "-r", "-i", path).CombinedOutput()
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			p.TxtOutput.SetText(string(out))
			p.TxtOutput.ScrollToBeginning()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
					ui.SetStatus("Scan completed — infected files found")
					return
				}
				ui.SetStatus("Scan failed: " + err.Error())
				return
			}
			ui.SetStatus("Scan completed — no threats found")
		})
	}()
}

// UpdateDefinitions runs freshclam to update the virus definitions.
func (p *SecurityPlugin) UpdateDefinitions() {
	if p.busy {
		ui.SetStatus("A security operation is already running")
		return
	}
	p.busy = true
	p.TxtOutput.SetTitle("Update virus definitions")
	p.TxtOutput.SetText("Running freshclam...\n")
	ui.SetStatus("Updating virus definitions...")
	go func() {
		out, err := exec.Command("freshclam").CombinedOutput()
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			p.TxtOutput.SetText(string(out))
			p.TxtOutput.ScrollToBeginning()
			if err != nil {
				ui.SetStatus("Update failed: " + err.Error())
				return
			}
			ui.SetStatus("Virus definitions updated")
			p.RefreshClamav()
		})
	}()
}

// showClamavContextMenu shows the context menu available in the ClamAV view.
func (p *SecurityPlugin) showClamavContextMenu() bool {
	m := (&menu.Menu{}).New(" ClamAV ", ui.PopupParentPage(), p.FocusWidget())
	m.AddItem("mnuSecScan", "Scan path...", func(any) {
		p.ShowScanPathDialog()
	}, nil, true, false)
	m.AddItem("mnuSecUpdate", "Update virus definitions", func(any) {
		p.UpdateDefinitions()
	}, nil, true, false)
	m.AddItem("mnuSecRefreshClamav", "Refresh status", func(any) {
		p.RefreshClamav()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuSecBack", "Back to firewall", func(any) {
		p.closeClamav()
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgSecurityClamavMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgSecurityClamavMenu")
	return true
}

// ****************************************************************************
// Outline panel integration
// ****************************************************************************

// RefreshOutline populates the shared Outline panel with details of the
// current selection in whichever sub-view is active.
func (p *SecurityPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	if p.clamavActive {
		p.refreshClamavOutline()
		return
	}
	rule := p.SelectedRule()
	if rule == nil {
		ui.TblOutline.SetTitle("Outline")
		return
	}
	fields := [][2]string{
		{"Rule #", rule.Num},
		{"To", rule.To},
		{"Action", rule.Action},
		{"From", rule.From},
	}
	for r, field := range fields {
		ui.TblOutline.SetCell(r, 0, tview.NewTableCell(field[0]).SetTextColor(tcell.ColorLightCyan))
		ui.TblOutline.SetCell(r, 1, tview.NewTableCell(field[1]).SetTextColor(tcell.ColorWhite))
	}
	ui.TblOutline.SetTitle("Outline — rule #" + rule.Num)
}

// refreshClamavOutline shows the currently selected ClamAV component in the
// shared Outline panel.
func (p *SecurityPlugin) refreshClamavOutline() {
	row, _ := p.TblClamav.GetSelection()
	if row <= 0 || row > len(p.clamavStatus) {
		ui.TblOutline.SetTitle("Outline")
		return
	}
	status := p.clamavStatus[row-1]
	ui.TblOutline.SetCell(0, 0, tview.NewTableCell("Component").SetTextColor(tcell.ColorLightCyan))
	ui.TblOutline.SetCell(0, 1, tview.NewTableCell(status.Label).SetTextColor(tcell.ColorWhite))
	ui.TblOutline.SetCell(1, 0, tview.NewTableCell("Status").SetTextColor(tcell.ColorLightCyan))
	ui.TblOutline.SetCell(1, 1, tview.NewTableCell(status.Value).SetTextColor(status.Color))
	ui.TblOutline.SetTitle("Outline — " + status.Label)
}

// ****************************************************************************
// Public helper used by lied.go keyboard handlers
// ****************************************************************************

// HandleInput handles firewall-view actions; callers may handle global keys
// such as Ctrl+T (close) and F2 (focus open views) before delegating here.
func (p *SecurityPlugin) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF5:
		p.RefreshFirewall()
		return nil
	case tcell.KeyDelete:
		p.ConfirmDeleteSelectedRule()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'a':
			p.ShowAddRuleDialog()
			return nil
		case 'e':
			p.EnableFirewall()
			return nil
		case 'd':
			p.ConfirmDisableFirewall()
			return nil
		case 'c':
			p.ShowClamav()
			return nil
		}
	}
	return event
}

// HandleClamavInput handles key presses while the ClamAV view is active.
func (p *SecurityPlugin) HandleClamavInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		p.closeClamav()
		return nil
	case tcell.KeyF5:
		p.RefreshClamav()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 's':
			p.ShowScanPathDialog()
			return nil
		case 'u':
			p.UpdateDefinitions()
			return nil
		}
	}
	return event
}
