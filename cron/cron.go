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
// Package cron provides a Cron Job Manager plugin for lied.
// It lists the current user's crontab entries and allows adding, editing,
// enabling/disabling and deleting them, writing changes back with `crontab -`.
// ****************************************************************************
package cron

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"fmt"
	"lied/dialog"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	PluginID        = "cronmanager"
	ContentPageName = "cronManager"
	UniqueID        = PluginID + "://" + PluginID
)

// cronLineRe matches the schedule and command portions of an (uncommented)
// crontab entry, supporting both the classic 5-field syntax and the "@"
// shorthand macros (e.g. @reboot, @daily).
var cronLineRe = regexp.MustCompile(`^(@\S+|(?:\S+\s+){4}\S+)\s+(.*)$`)

// ****************************************************************************
// TYPES
// ****************************************************************************

// CronJob describes a single parsed crontab entry.
type CronJob struct {
	// LineIndex is the position of this entry within CronManagerPlugin.rawLines.
	LineIndex int
	Enabled   bool
	Schedule  string
	Command   string
}

// CronManagerPlugin implements ui.ViewPlugin and shows the current user's
// crontab entries in an editable table.
type CronManagerPlugin struct {
	TblJobs *tview.Table
	layout  *tview.Flex
	// rawLines holds every line of the crontab verbatim (including blank
	// lines, comments and env-var assignments) so that anything the plugin
	// does not understand is preserved when writing changes back.
	rawLines []string
	jobs     []CronJob
	confirm  *dialog.Dialog
}

// NewCronManagerPlugin creates and wires up the Cron Job Manager plugin.
func NewCronManagerPlugin() *CronManagerPlugin {
	p := &CronManagerPlugin{}

	p.TblJobs = tview.NewTable()
	p.TblJobs.SetBorder(true)
	p.TblJobs.SetTitle("Cron Jobs")
	p.TblJobs.SetSelectable(true, false)

	p.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblJobs, 0, 1, true)

	ui.PgsEditorContent.AddPage(ContentPageName, p.layout, true, false)

	p.TblJobs.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *CronManagerPlugin) ID() string    { return PluginID }
func (p *CronManagerPlugin) Title() string { return "Cron Manager" }
func (p *CronManagerPlugin) Icon() string  { return "⏱" }

// Activate switches the application to the cron manager content page and
// sets focus to the jobs table.
func (p *CronManagerPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	ui.SetFindPanelVisible(false)
	p.RefreshOutline()
	ui.App.SetFocus(p.TblJobs)
	ui.LblKeys.SetText(p.KeyHints())
}

// FocusWidget returns the jobs table as the primary focus target.
func (p *CronManagerPlugin) FocusWidget() tview.Primitive { return p.TblJobs }

// Open refreshes the crontab listing. param is unused.
func (p *CronManagerPlugin) Open(_ any) error {
	p.Refresh()
	return nil
}

// Close is a no-op; the cron manager holds no resources that need cleanup.
func (p *CronManagerPlugin) Close() error { return nil }

// IsDirty always returns false; changes are written to crontab immediately.
func (p *CronManagerPlugin) IsDirty() bool { return false }

// StatusFields returns values for the bottom status bar widgets.
func (p *CronManagerPlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d job(s)", len(p.jobs)),
		Encoding:  "cron",
	}
}

// KeyHints returns the two-line key-hint string for the LblKeys bar.
func (p *CronManagerPlugin) KeyHints() string {
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[a] Add  [e] Edit  [Space] Toggle  [Del] Delete  [F5] Refresh  [Ctrl+F] Find  [Ctrl+T] Close"
}

func (p *CronManagerPlugin) InternalCommand() string      { return "!cron" }
func (p *CronManagerPlugin) CommandOpensPluginView() bool { return true }

func (p *CronManagerPlugin) ExecuteInternalCommand() error {
	// Command is handled by opening plugin view in dispatcher.
	return nil
}

func (p *CronManagerPlugin) ShowContextMenu(defaultMenu func()) bool {
	m := (&menu.Menu{}).New(" Cron Manager ", ui.PopupParentPage(), p.FocusWidget())
	edit.AddOpenViewsMenuItems(m)
	hasJob := p.SelectedJob() != nil

	m.AddItem("mnuCronRefresh", "Refresh cron jobs", func(any) {
		p.Refresh()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuCronAdd", "Add job...", func(any) {
		p.ShowJobForm(nil)
	}, nil, true, false)
	m.AddItem("mnuCronEdit", "Edit job...", func(any) {
		if job := p.SelectedJob(); job != nil {
			jobCopy := *job
			p.ShowJobForm(&jobCopy)
		}
	}, nil, hasJob, false)
	m.AddItem("mnuCronToggle", "Enable/disable job", func(any) {
		p.ToggleSelected()
	}, nil, hasJob, false)
	m.AddItem("mnuCronDelete", "Delete job", func(any) {
		p.ConfirmDelete()
	}, nil, hasJob, false)

	ui.PgsApp.AddPage("dlgCronManagerMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgCronManagerMenu")
	return true
}

// ****************************************************************************
// crontab I/O
// ****************************************************************************

// Refresh reloads the crontab from the system and repopulates the table.
func (p *CronManagerPlugin) Refresh() {
	p.loadCrontab()
	p.populateTable()
	ui.SetStatus(fmt.Sprintf("Found %d cron job(s)", len(p.jobs)))
}

// loadCrontab runs `crontab -l` and parses its output into rawLines/jobs.
// A missing crontab (exit status with "no crontab" message) is treated as an
// empty crontab rather than an error.
func (p *CronManagerPlugin) loadCrontab() {
	out, err := exec.Command("crontab", "-l").CombinedOutput()
	text := string(out)
	if err != nil {
		if strings.Contains(strings.ToLower(text), "no crontab") {
			p.rawLines = nil
			p.jobs = nil
			return
		}
		ui.SetStatus("crontab -l failed: " + strings.TrimSpace(text))
		return
	}

	p.rawLines = splitLines(text)
	p.jobs = nil
	for idx, line := range p.rawLines {
		if job, ok := parseCronLine(line); ok {
			job.LineIndex = idx
			p.jobs = append(p.jobs, job)
		}
	}
}

// saveCrontab writes rawLines back to the system crontab via `crontab -`.
func (p *CronManagerPlugin) saveCrontab() error {
	content := strings.Join(p.rawLines, "\n")
	if content != "" {
		content += "\n"
	}
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// commitAndReload saves rawLines to the crontab, then reloads it from the
// system so that the table always reflects the actual installed crontab.
func (p *CronManagerPlugin) commitAndReload(successMsg string) {
	if err := p.saveCrontab(); err != nil {
		ui.SetStatus("crontab failed: " + err.Error())
		return
	}
	p.loadCrontab()
	p.populateTable()
	ui.SetStatus(successMsg)
}

// splitLines splits crontab text into lines, returning nil for empty input.
func splitLines(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// parseCronLine attempts to parse line as a (possibly disabled) cron entry.
// Plain comments and blank lines return ok == false.
func parseCronLine(line string) (CronJob, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return CronJob{}, false
	}

	enabled := true
	body := trimmed
	if strings.HasPrefix(body, "#") {
		body = strings.TrimSpace(strings.TrimPrefix(body, "#"))
		enabled = false
	}
	if body == "" || strings.HasPrefix(body, "#") {
		return CronJob{}, false
	}

	m := cronLineRe.FindStringSubmatch(body)
	if m == nil {
		return CronJob{}, false
	}
	return CronJob{Enabled: enabled, Schedule: m[1], Command: strings.TrimSpace(m[2])}, true
}

// ****************************************************************************
// Job operations
// ****************************************************************************

// AddJob appends a new enabled cron entry and commits it to the crontab.
func (p *CronManagerPlugin) AddJob(schedule, command string) {
	p.rawLines = append(p.rawLines, schedule+" "+command)
	p.commitAndReload("Added cron job")
}

// EditJob replaces the raw line for job with a new schedule/command,
// preserving its enabled/disabled state.
func (p *CronManagerPlugin) EditJob(job CronJob, schedule, command string) {
	if job.LineIndex < 0 || job.LineIndex >= len(p.rawLines) {
		ui.SetStatus("Cron job no longer exists")
		return
	}
	line := schedule + " " + command
	if !job.Enabled {
		line = "# " + line
	}
	p.rawLines[job.LineIndex] = line
	p.commitAndReload("Updated cron job")
}

// ToggleSelected enables or disables the currently selected job by adding or
// removing a leading "#" on its raw line.
func (p *CronManagerPlugin) ToggleSelected() {
	job := p.SelectedJob()
	if job == nil {
		ui.SetStatus("No cron job selected")
		return
	}
	trimmed := strings.TrimSpace(p.rawLines[job.LineIndex])
	if strings.HasPrefix(trimmed, "#") {
		p.rawLines[job.LineIndex] = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
	} else {
		p.rawLines[job.LineIndex] = "# " + trimmed
	}
	p.commitAndReload("Toggled cron job")
}

// DeleteSelected removes the currently selected job's raw line entirely.
func (p *CronManagerPlugin) DeleteSelected() {
	job := p.SelectedJob()
	if job == nil {
		ui.SetStatus("No cron job selected")
		return
	}
	p.rawLines = append(p.rawLines[:job.LineIndex], p.rawLines[job.LineIndex+1:]...)
	p.commitAndReload("Deleted cron job")
}

// ConfirmDelete asks for confirmation before deleting the selected job.
func (p *CronManagerPlugin) ConfirmDelete() {
	job := p.SelectedJob()
	if job == nil {
		ui.SetStatus("No cron job selected")
		return
	}
	p.confirm = p.confirm.YesNo(
		"Delete cron job",
		fmt.Sprintf("Delete job '%s'?", job.Command),
		func(button dialog.DlgButton, _ int) {
			if button == dialog.BUTTON_YES {
				p.DeleteSelected()
			}
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgCronConfirmDelete", p.confirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgCronConfirmDelete")
}

// ShowJobForm opens an Add/Edit form. Pass existing == nil to add a new job,
// or a copy of the job to edit its schedule/command.
func (p *CronManagerPlugin) ShowJobForm(existing *CronJob) {
	title := " Add Cron Job "
	defaultSchedule := "* * * * *"
	defaultCommand := ""
	if existing != nil {
		title = " Edit Cron Job "
		defaultSchedule = existing.Schedule
		defaultCommand = existing.Command
	}

	schedule := tview.NewInputField().SetLabel("Schedule (m h dom mon dow): ").SetFieldWidth(37).SetText(defaultSchedule)
	command := tview.NewInputField().SetLabel("Command: ").SetFieldWidth(37).SetText(defaultCommand)

	form := tview.NewForm()
	form.SetBorder(true).SetTitle(title)
	form.AddFormItem(schedule)
	form.AddFormItem(command)
	form.AddButton("Save", func() {
		sched := strings.TrimSpace(schedule.GetText())
		cmd := strings.TrimSpace(command.GetText())
		if sched == "" || cmd == "" {
			ui.SetStatus("Schedule and command are required")
			return
		}
		ui.PgsApp.HidePage("dlgCronJobForm")
		ui.App.SetFocus(p.FocusWidget())
		if existing != nil {
			p.EditJob(*existing, sched, cmd)
		} else {
			p.AddJob(sched, cmd)
		}
	})
	form.AddButton("Cancel", func() {
		ui.PgsApp.HidePage("dlgCronJobForm")
		ui.App.SetFocus(p.FocusWidget())
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.PgsApp.HidePage("dlgCronJobForm")
			ui.App.SetFocus(p.FocusWidget())
			return nil
		}
		return event
	})

	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(form, 70, 0, true).
		AddItem(nil, 0, 1, false)
	ui.PgsApp.AddPage("dlgCronJobForm", centered, true, true)
	ui.App.SetFocus(schedule)
}

// HandleInput handles cron-view actions; callers may handle global close keys.
func (p *CronManagerPlugin) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF5:
		p.Refresh()
		return nil
	case tcell.KeyCtrlT:
		return event
	case tcell.KeyDelete:
		p.ConfirmDelete()
		return nil
	case tcell.KeyEnter:
		p.ToggleSelected()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'a':
			p.ShowJobForm(nil)
			return nil
		case 'e':
			if job := p.SelectedJob(); job != nil {
				jobCopy := *job
				p.ShowJobForm(&jobCopy)
			}
			return nil
		case ' ':
			p.ToggleSelected()
			return nil
		}
	}
	return event
}

// ****************************************************************************
// Table / outline helpers
// ****************************************************************************

// populateTable rebuilds TblJobs from p.jobs.
func (p *CronManagerPlugin) populateTable() {
	p.TblJobs.Clear()

	headers := []string{"On", "Schedule", "Command"}
	for col, h := range headers {
		p.TblJobs.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAttributes(tcell.AttrBold).
				SetSelectable(false))
	}
	p.TblJobs.SetFixed(1, 0)

	for row, job := range p.jobs {
		onMark := "✓"
		color := tcell.ColorGreen
		if !job.Enabled {
			onMark = "✗"
			color = tcell.ColorGray
		}
		p.TblJobs.SetCell(row+1, 0, tview.NewTableCell(onMark).SetTextColor(color).SetAlign(tview.AlignCenter))
		p.TblJobs.SetCell(row+1, 1, tview.NewTableCell(job.Schedule).SetTextColor(color))
		p.TblJobs.SetCell(row+1, 2, tview.NewTableCell(job.Command).SetTextColor(color))
	}

	p.TblJobs.SetTitle(fmt.Sprintf("Cron Jobs (%d)", len(p.jobs)))
	if len(p.jobs) > 0 {
		row, _ := p.TblJobs.GetSelection()
		if row <= 0 || row > len(p.jobs) {
			p.TblJobs.Select(1, 0)
		}
	}
	p.RefreshOutline()
}

// SelectedJob returns the currently highlighted job, or nil when no valid
// row is selected.
func (p *CronManagerPlugin) SelectedJob() *CronJob {
	row, _ := p.TblJobs.GetSelection()
	if row <= 0 || row > len(p.jobs) {
		return nil
	}
	return &p.jobs[row-1]
}

// RefreshOutline populates the shared Outline panel with a field-by-field
// breakdown of the selected job's schedule plus its full command.
func (p *CronManagerPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	job := p.SelectedJob()
	if job == nil {
		ui.TblOutline.SetTitle("Outline")
		return
	}

	row := 0
	for _, field := range scheduleFields(job.Schedule) {
		ui.TblOutline.SetCell(row, 0, tview.NewTableCell(field[0]).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignLeft))
		ui.TblOutline.SetCell(row, 1, tview.NewTableCell(field[1]).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		row++
	}
	ui.TblOutline.SetCell(row, 0, tview.NewTableCell("Command").SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignLeft))
	ui.TblOutline.SetCell(row, 1, tview.NewTableCell(job.Command).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
	ui.TblOutline.SetTitle("Outline — " + job.Schedule)
}

// scheduleFields breaks a cron schedule into labelled fields for display.
func scheduleFields(schedule string) [][2]string {
	if strings.HasPrefix(schedule, "@") {
		return [][2]string{{"Special", schedule}}
	}
	parts := strings.Fields(schedule)
	labels := []string{"Minute", "Hour", "Day of month", "Month", "Day of week"}
	result := make([][2]string, 0, len(labels))
	for i, label := range labels {
		value := "*"
		if i < len(parts) {
			value = parts[i]
		}
		result = append(result, [2]string{label, value})
	}
	return result
}
