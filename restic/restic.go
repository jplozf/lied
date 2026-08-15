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
// Package restic provides a Restic Backup Manager plugin for lied.
// It lists the snapshots of the repository configured through the standard
// RESTIC_REPOSITORY / RESTIC_PASSWORD(_COMMAND|_FILE) environment variables,
// shows the files of a selected snapshot, and lets the user trigger a backup,
// a restore or a repository check — all from within the editor.
// ****************************************************************************
package restic

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"lied/conf"
	"lied/dialog"
	"lied/menu"
	"lied/ui"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	PluginID        = "resticmanager"
	ContentPageName = "resticManager"
	// UniqueID is used as the synthetic FName when this plugin is open in the
	// views list so that duplicate opens can be detected.
	UniqueID = PluginID + "://" + PluginID
)

// ****************************************************************************
// Snapshot
// ****************************************************************************

// Snapshot holds the fields of interest from `restic snapshots --json`.
type Snapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Tags     []string  `json:"tags"`
	Paths    []string  `json:"paths"`
}

// ****************************************************************************
// ResticPlugin
// ****************************************************************************

// ResticPlugin implements ui.ViewPlugin and shows a live list of restic
// backup snapshots together with a file-listing panel for the selected one.
type ResticPlugin struct {
	// TblSnapshots is the selectable table that lists all snapshots.
	TblSnapshots *tview.Table
	// TxtDetail shows the file listing (or error) for the selected snapshot.
	TxtDetail     *tview.TextView
	layout        *tview.Flex
	snapshots     []Snapshot
	backupRunning bool
	backupOutput  *tview.TextView
	pruneDialog   *dialog.Dialog
}

// NewResticPlugin creates and wires up the Restic Backup Manager plugin.
// It registers its content page directly with ui.PgsEditorContent so that it
// is rendered within the standard editor frame (header / key-hints / status bar)
// without needing its own full-screen layout.
func NewResticPlugin() *ResticPlugin {
	p := &ResticPlugin{}

	p.TblSnapshots = tview.NewTable()
	p.TblSnapshots.SetBorder(true)
	p.TblSnapshots.SetTitle("Snapshots")
	p.TblSnapshots.SetSelectable(true, false)

	p.TxtDetail = tview.NewTextView()
	p.TxtDetail.SetBorder(true)
	p.TxtDetail.SetTitle("Files")
	p.TxtDetail.SetDynamicColors(true)
	p.TxtDetail.SetScrollable(true)
	p.TxtDetail.SetWrap(true)

	p.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblSnapshots, 0, 2, true).
		AddItem(p.TxtDetail, 0, 2, false)

	ui.PgsEditorContent.AddPage(ContentPageName, p.layout, true, false)

	// Keep the shared Outline panel in sync with the highlighted snapshot.
	p.TblSnapshots.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *ResticPlugin) ID() string    { return PluginID }
func (p *ResticPlugin) Title() string { return "Restic Backup" }
func (p *ResticPlugin) Icon() string  { return "🗄" }

// Activate switches the application to the restic manager content page and
// sets focus to the snapshots table.  It also repurposes the shared Search
// and Outline panels: Search filters the snapshot list by host/path/tag, and
// Outline shows the properties of the currently selected snapshot.
func (p *ResticPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	p.configureSearchPanel()
	p.RefreshOutline()
	ui.App.SetFocus(p.TblSnapshots)
	ui.LblKeys.SetText(p.KeyHints())
}

// configureSearchPanel adapts the shared Find form to search snapshots by
// host, path or tag instead of searching text/hex buffer content.
func (p *ResticPlugin) configureSearchPanel() {
	ui.SetFindPanelVisible(true)
	ui.FrmFind.SetTitle("Find Snapshot")
	ui.TxtReplace.SetDisabled(true)
	ui.ChkToggleReplace.SetDisabled(true)
	ui.FrmFind.GetButton(2).SetDisabled(true) // Replace One
	ui.FrmFind.GetButton(3).SetDisabled(true) // Replace All
	ui.DpdSearchType.SetDisabled(true)
	ui.DpdSearchType.SetCurrentOption(0)
	ui.FrmFind.GetButton(0).SetSelectedFunc(func() { p.FindNext() })
	ui.FrmFind.GetButton(1).SetSelectedFunc(func() { p.FindPrevious() })
}

// FocusWidget returns the snapshots table as the primary focus target.
func (p *ResticPlugin) FocusWidget() tview.Primitive { return p.TblSnapshots }

// Open refreshes the snapshot list.  param is unused.
func (p *ResticPlugin) Open(_ any) error {
	p.Refresh()
	return nil
}

// Close is a no-op; the restic manager holds no resources that need cleanup.
func (p *ResticPlugin) Close() error { return nil }

// IsDirty always returns false because the restic manager is read-only.
func (p *ResticPlugin) IsDirty() bool { return false }

// StatusFields returns values for the bottom status bar widgets.
func (p *ResticPlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d snapshot(s)", len(p.snapshots)),
		Dirty:     "",
		Percent:   "",
		Size:      "",
		Encoding:  "restic",
	}
}

// KeyHints returns the two-line key-hint string for the LblKeys bar.
func (p *ResticPlugin) KeyHints() string {
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[Enter] List files  [b] Backup now  [c] Check repo  [F5] Refresh  [Ctrl+F] Find  [Ctrl+T] Close"
}

func (p *ResticPlugin) InternalCommand() string { return "!rest" }

func (p *ResticPlugin) CommandOpensPluginView() bool { return true }

func (p *ResticPlugin) ExecuteInternalCommand() error {
	// Command is handled by opening plugin view in dispatcher.
	return nil
}

func (p *ResticPlugin) ShowContextMenu(defaultMenu func()) bool {
	m := (&menu.Menu{}).New(" Restic Backup ", ui.PopupParentPage(), p.FocusWidget())
	id := p.SelectedSnapshotID()
	hasSnapshot := id != ""

	m.AddItem("mnuResticRefresh", "Refresh snapshots", func(any) {
		p.Refresh()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuResticListFiles", "List files", func(any) {
		if id != "" {
			p.ShowSnapshotFiles(id)
		}
	}, nil, hasSnapshot, false)
	m.AddItem("mnuResticBackup", "Backup now", func(any) {
		p.ShowBackupFolderMenu()
	}, nil, true, false)
	m.AddItem("mnuResticRestore", "Restore to workspace subfolder", func(any) {
		if id != "" {
			p.Restore(id)
		}
	}, nil, hasSnapshot, false)
	m.AddItem("mnuResticCheck", "Check repository", func(any) {
		p.CheckRepository()
	}, nil, true, false)
	m.AddItem("mnuResticPrune", "Prune repository...", func(any) {
		p.ShowPruneDialog()
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgResticMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgResticMenu")
	return true
}

// ShowPruneDialog asks how many monthly snapshots should be retained.
func (p *ResticPlugin) ShowPruneDialog() {
	p.pruneDialog = p.pruneDialog.Input(
		"Prune Restic Repository",
		"Keep how many monthly snapshots?",
		"12",
		func(rc dialog.DlgButton, _ int) {
			if rc != dialog.BUTTON_OK {
				return
			}
			keepMonthly, err := strconv.Atoi(strings.TrimSpace(p.pruneDialog.Value))
			if err != nil || keepMonthly < 1 {
				ui.SetStatus("Keep-monthly must be a positive integer")
				return
			}
			p.Prune(keepMonthly)
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgResticPrune", p.pruneDialog.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgResticPrune")
}

// ShowBackupFolderMenu opens a navigable multi-select filesystem browser.
func (p *ResticPlugin) ShowBackupFolderMenu() {
	workspace := conf.ConfigGeneral.Workspace
	if workspace == "" {
		workspace = "."
	}
	workspace, _ = filepath.Abs(workspace)

	type pickerEntry struct {
		name string
		path string
		dir  bool
	}

	selected := make(map[string]bool)
	selectedOrder := make([]string, 0)
	currentPath := workspace
	currentEntries := make([]pickerEntry, 0)
	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.SetBorder(true)
	close := func() {
		ui.PgsApp.HidePage("dlgResticBackupFolders")
		ui.App.SetFocus(p.FocusWidget())
	}
	startBackup := func() {
		backupPaths := append([]string(nil), selectedOrder...)
		if len(backupPaths) == 0 {
			ui.SetStatus("Select at least one file or folder to backup")
			return
		}
		close()
		p.Backup(backupPaths...)
	}
	toggleCurrent := func() {
		index := list.GetCurrentItem()
		if index < 0 || index >= len(currentEntries) {
			return
		}
		path := currentEntries[index].path
		if selected[path] {
			delete(selected, path)
			for i, selectedPath := range selectedOrder {
				if selectedPath == path {
					selectedOrder = append(selectedOrder[:i], selectedOrder[i+1:]...)
					break
				}
			}
		} else {
			selected[path] = true
			selectedOrder = append(selectedOrder, path)
		}
	}
	populate := func(path string) {
		entries, err := os.ReadDir(path)
		if err != nil {
			ui.SetStatus(fmt.Sprintf("Cannot read %s: %v", path, err))
			return
		}
		currentPath = path
		currentEntries = currentEntries[:0]
		list.Clear()
		list.SetTitle(" " + path + " ")
		if path != string(filepath.Separator) {
			currentEntries = append(currentEntries, pickerEntry{name: "..", path: filepath.Dir(path), dir: true})
			list.AddItem("..", "Parent directory", 0, nil)
		}
		for _, entry := range entries {
			if !conf.ConfigGeneral.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			entryPath := filepath.Join(path, entry.Name())
			isDir := entry.IsDir()
			if !isDir {
				if info, statErr := os.Stat(entryPath); statErr == nil {
					isDir = info.IsDir()
				}
			}
			currentEntries = append(currentEntries, pickerEntry{name: entry.Name(), path: entryPath, dir: isDir})
			prefix := "[ ] "
			if selected[entryPath] {
				prefix = "[x] "
			}
			label := prefix + entry.Name()
			if isDir {
				label += "/"
			}
			list.AddItem(label, entryPath, 0, nil)
		}
		list.SetCurrentItem(0)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			close()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'b':
				startBackup()
				return nil
			case ' ':
				toggleCurrent()
				populate(currentPath)
				return nil
			}
		}
		return event
	})
	list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if index < 0 || index >= len(currentEntries) {
			return
		}
		if currentEntries[index].name == ".." || currentEntries[index].dir {
			populate(currentEntries[index].path)
			return
		}
		toggleCurrent()
		populate(currentPath)
	})

	help := tview.NewTextView().SetText("Enter: open folder   Space: select   b: backup   Esc: cancel")
	help.SetTextAlign(tview.AlignCenter)
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true).
		AddItem(help, 1, 0, false)
	modal.SetBorder(true).SetTitle(" Restic Backup ")

	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(modal, 70, 0, true).
		AddItem(nil, 0, 1, false)
	ui.PgsApp.AddPage("dlgResticBackupFolders", centered, true, true)
	populate(currentPath)
}

// ****************************************************************************
// Public helpers used by lied.go keyboard handlers
// ****************************************************************************

// resticRepositoryConfigured reports whether the environment is set up to
// reach a restic repository (RESTIC_REPOSITORY plus one of the supported
// password sources). It never reads or stores the password itself.
func resticRepositoryConfigured() bool {
	if strings.TrimSpace(os.Getenv("RESTIC_REPOSITORY")) == "" {
		return false
	}
	return os.Getenv("RESTIC_PASSWORD") != "" ||
		os.Getenv("RESTIC_PASSWORD_COMMAND") != "" ||
		os.Getenv("RESTIC_PASSWORD_FILE") != ""
}

// Refresh rebuilds the snapshots table by running `restic snapshots --json`.
// Credentials are taken exclusively from the process environment
// (RESTIC_REPOSITORY / RESTIC_PASSWORD[_COMMAND|_FILE]); lied never stores or
// prompts for the repository password itself.
func (p *ResticPlugin) Refresh() {
	p.TblSnapshots.Clear()

	headers := []string{"ID", "Time", "Host", "Tags", "Paths"}
	for col, h := range headers {
		p.TblSnapshots.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAttributes(tcell.AttrBold).
				SetSelectable(false))
	}
	p.TblSnapshots.SetFixed(1, 0)

	if !resticRepositoryConfigured() {
		p.TblSnapshots.SetCell(1, 0,
			tview.NewTableCell("RESTIC_REPOSITORY / RESTIC_PASSWORD* not set in environment").
				SetTextColor(tcell.ColorRed))
		p.snapshots = nil
		p.RefreshOutline()
		return
	}

	out, err := exec.Command("restic", "snapshots", "--json").Output()
	if err != nil {
		p.TblSnapshots.SetCell(1, 0,
			tview.NewTableCell("Error: "+resticErrorMessage(err)).SetTextColor(tcell.ColorRed))
		p.snapshots = nil
		p.RefreshOutline()
		return
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(out, &snapshots); err != nil {
		p.TblSnapshots.SetCell(1, 0,
			tview.NewTableCell("Error parsing snapshots: "+err.Error()).SetTextColor(tcell.ColorRed))
		p.snapshots = nil
		p.RefreshOutline()
		return
	}
	p.snapshots = snapshots

	row := 1
	for _, s := range snapshots {
		p.TblSnapshots.SetCell(row, 0, tview.NewTableCell(s.ShortID).SetTextColor(tcell.ColorWhite))
		p.TblSnapshots.SetCell(row, 1, tview.NewTableCell(s.Time.Format("2006-01-02 15:04:05")).SetTextColor(tcell.ColorWhite))
		p.TblSnapshots.SetCell(row, 2, tview.NewTableCell(s.Hostname).SetTextColor(tcell.ColorWhite))
		p.TblSnapshots.SetCell(row, 3, tview.NewTableCell(strings.Join(s.Tags, ",")).SetTextColor(tcell.ColorWhite))
		p.TblSnapshots.SetCell(row, 4, tview.NewTableCell(strings.Join(s.Paths, ", ")).SetTextColor(tcell.ColorWhite))
		row++
	}

	p.TblSnapshots.SetTitle(fmt.Sprintf("Snapshots (%d)", row-1))
	p.RefreshOutline()
}

// resticErrorMessage trims the noisy prefix off exec.ExitError output so the
// status bar shows something readable.
func resticErrorMessage(err error) string {
	if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
		return strings.TrimSpace(string(exitErr.Stderr))
	}
	return err.Error()
}

// ShowSnapshotFiles fills TxtDetail with the file listing of the given
// snapshot ID (`restic ls <id> --long`).
func (p *ResticPlugin) ShowSnapshotFiles(id string) {
	p.TxtDetail.Clear()
	out, err := exec.Command("restic", "ls", id, "--long").Output()
	if err != nil {
		fmt.Fprintf(p.TxtDetail, "[red]Error listing files: %v[-]", resticErrorMessage(err))
		return
	}
	p.TxtDetail.SetText(string(out))
	p.TxtDetail.ScrollToBeginning()
	p.TxtDetail.SetTitle(fmt.Sprintf("Files — %s", id))
}

// Backup starts `restic backup <paths...>` asynchronously and displays its
// output in a progress popup.
func (p *ResticPlugin) Backup(paths ...string) {
	validPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) != "" {
			validPaths = append(validPaths, path)
		}
	}
	if len(validPaths) == 0 {
		ui.SetStatus("No path to backup")
		return
	}
	if p.backupRunning {
		ui.SetStatus("A restic backup is already running")
		return
	}

	p.backupRunning = true
	progress := p.showResticProgress(fmt.Sprintf("Restic Backup (%d path(s))", len(validPaths)))
	ui.SetStatus(fmt.Sprintf("Running restic backup for %d path(s)…", len(validPaths)))
	args := append([]string{"backup"}, validPaths...)
	go p.runRestic(args, progress, "Backup")
}

// Prune runs restic forget with the requested monthly retention and prune flag.
func (p *ResticPlugin) Prune(keepMonthly int) {
	if p.backupRunning {
		ui.SetStatus("A restic operation is already running")
		return
	}
	p.backupRunning = true
	progress := p.showResticProgress("Restic Prune")
	ui.SetStatus(fmt.Sprintf("Pruning repository, keeping %d monthly snapshots…", keepMonthly))
	args := []string{"forget", "--keep-monthly", strconv.Itoa(keepMonthly), "--prune"}
	go p.runRestic(args, progress, "Prune")
}

// showResticProgress creates the popup used while a Restic operation runs.
func (p *ResticPlugin) showResticProgress(title string) *tview.TextView {
	progress := tview.NewTextView()
	progress.SetBorder(true)
	progress.SetTitle(" " + title + " ")
	progress.SetScrollable(true)
	progress.SetWrap(true)
	progress.SetText("Starting restic operation...\n\nPress Esc to hide this popup; the operation will continue.")
	p.backupOutput = progress

	progress.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.PgsApp.HidePage("dlgResticBackupProgress")
			ui.App.SetFocus(p.FocusWidget())
			return nil
		}
		return event
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(progress, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Esc: hide popup"), 1, 0, false)
	modal.SetBorder(true).SetTitle(" Backup progress ")
	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(modal, 80, 0, true).
		AddItem(nil, 0, 1, false)
	ui.PgsApp.AddPage("dlgResticBackupProgress", centered, true, true)
	return progress
}

// runRestic streams Restic output without occupying the UI goroutine.
func (p *ResticPlugin) runRestic(args []string, progress *tview.TextView, operation string) {
	cmd := exec.Command("restic", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.finishRestic(progress, "Could not capture restic output: "+err.Error(), err, operation)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.finishRestic(progress, "Could not capture restic errors: "+err.Error(), err, operation)
		return
	}
	if err := cmd.Start(); err != nil {
		p.finishRestic(progress, "Could not start restic: "+err.Error(), err, operation)
		return
	}

	lines := make(chan string)
	var readers sync.WaitGroup
	readOutput := func(reader io.Reader) {
		defer readers.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}
	readers.Add(2)
	go readOutput(stdout)
	go readOutput(stderr)
	go func() {
		readers.Wait()
		close(lines)
	}()

	waitResult := make(chan error, 1)
	go func() { waitResult <- cmd.Wait() }()
	var output strings.Builder
	for line := range lines {
		output.WriteString(line)
		output.WriteByte('\n')
		text := output.String()
		ui.App.QueueUpdateDraw(func() {
			progress.SetText(text)
			progress.ScrollToEnd()
		})
	}
	err = <-waitResult
	if err != nil {
		p.finishRestic(progress, output.String()+"\n"+operation+" failed: "+resticErrorMessage(err), err, operation)
		return
	}
	p.finishRestic(progress, output.String()+"\n"+operation+" completed successfully.", nil, operation)
}

// finishRestic updates the popup and the editor after the background command
// exits.
func (p *ResticPlugin) finishRestic(progress *tview.TextView, output string, err error, operation string) {
	ui.App.QueueUpdateDraw(func() {
		p.backupRunning = false
		progress.SetText(output)
		progress.ScrollToEnd()
		if err != nil {
			progress.SetTitle(" Restic " + operation + " - failed ")
			ui.SetStatus("restic " + strings.ToLower(operation) + " failed: " + resticErrorMessage(err))
			return
		}
		progress.SetTitle(" Restic " + operation + " - completed ")
		ui.SetStatus(operation + " completed")
		p.TxtDetail.SetText(output)
		p.TxtDetail.ScrollToEnd()
		p.TxtDetail.SetTitle("Backup output")
		if operation == "Backup" {
			p.Refresh()
		}
	})
}

// Restore restores the given snapshot into a dedicated subfolder of the
// current workspace, never overwriting existing files in place.
func (p *ResticPlugin) Restore(id string) {
	target := filepath.Join(conf.ConfigGeneral.Workspace, "restic-restore-"+id)
	ui.SetStatus(fmt.Sprintf("Restoring %s to %s…", id, target))
	out, err := exec.Command("restic", "restore", id, "--target", target).CombinedOutput()
	if err != nil {
		ui.SetStatus("restic restore failed: " + resticErrorMessage(err))
		p.TxtDetail.SetText(string(out))
		return
	}
	ui.SetStatus(fmt.Sprintf("Restored %s to %s", id, target))
	p.TxtDetail.SetText(string(out))
	p.TxtDetail.ScrollToEnd()
	p.TxtDetail.SetTitle("Restore output")
}

// CheckRepository runs `restic check` and shows its output in TxtDetail.
func (p *ResticPlugin) CheckRepository() {
	ui.SetStatus("Checking restic repository…")
	out, err := exec.Command("restic", "check").CombinedOutput()
	p.TxtDetail.SetText(string(out))
	p.TxtDetail.ScrollToBeginning()
	p.TxtDetail.SetTitle("Repository check")
	if err != nil {
		ui.SetStatus("restic check reported errors: " + resticErrorMessage(err))
		return
	}
	ui.SetStatus("Repository check OK")
}

// SelectedSnapshotID returns the short ID of the currently selected
// snapshot, or "" when no valid row is selected.
func (p *ResticPlugin) SelectedSnapshotID() string {
	row, _ := p.TblSnapshots.GetSelection()
	if row <= 0 || row >= p.TblSnapshots.GetRowCount() {
		return ""
	}
	cell := p.TblSnapshots.GetCell(row, 0)
	if cell == nil {
		return ""
	}
	return cell.Text
}

// selectedSnapshot returns the Snapshot struct backing the selected row.
func (p *ResticPlugin) selectedSnapshot() (Snapshot, bool) {
	id := p.SelectedSnapshotID()
	if id == "" {
		return Snapshot{}, false
	}
	for _, s := range p.snapshots {
		if s.ShortID == id {
			return s, true
		}
	}
	return Snapshot{}, false
}

// ****************************************************************************
// Outline panel integration
// ****************************************************************************

// RefreshOutline populates the shared Outline panel with the properties of
// the currently selected snapshot.
func (p *ResticPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	s, ok := p.selectedSnapshot()
	if !ok {
		ui.TblOutline.SetTitle("Outline")
		return
	}

	fields := [][2]string{
		{"ID", s.ID},
		{"Short ID", s.ShortID},
		{"Time", s.Time.Format(time.RFC3339)},
		{"Hostname", s.Hostname},
		{"Tags", strings.Join(s.Tags, ", ")},
		{"Paths", strings.Join(s.Paths, ", ")},
	}
	for row, f := range fields {
		ui.TblOutline.SetCell(row, 0,
			tview.NewTableCell(f[0]).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignLeft))
		ui.TblOutline.SetCell(row, 1,
			tview.NewTableCell(f[1]).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
	}
	ui.TblOutline.ScrollToBeginning()
	ui.TblOutline.SetTitle(fmt.Sprintf("Outline — %s", s.ShortID))
}

// ****************************************************************************
// Search panel integration
// ****************************************************************************

// matchesSnapshot reports whether row's host, tags or paths contain the
// given case-insensitive substring.
func (p *ResticPlugin) matchesSnapshot(row int, needle string) bool {
	for _, col := range []int{0, 2, 3, 4} {
		cell := p.TblSnapshots.GetCell(row, col)
		if cell != nil && strings.Contains(strings.ToLower(cell.Text), needle) {
			return true
		}
	}
	return false
}

// findSnapshot selects the next (or, if backward, previous) snapshot row
// matching the Find field's text, wrapping around the table.
func (p *ResticPlugin) findSnapshot(backward bool) {
	needle := strings.ToLower(strings.TrimSpace(ui.TxtFind.GetText()))
	rowCount := p.TblSnapshots.GetRowCount()
	if needle == "" || rowCount <= 1 {
		ui.FrmFind.SetTitle("Find Snapshot")
		ui.SetStatus("Nothing to search")
		return
	}

	total := 0
	for r := 1; r < rowCount; r++ {
		if p.matchesSnapshot(r, needle) {
			total++
		}
	}
	if total == 0 {
		ui.FrmFind.SetTitle("Find Snapshot (0/0)")
		ui.SetStatus(fmt.Sprintf("No snapshot matching '%s'", needle))
		return
	}

	current, _ := p.TblSnapshots.GetSelection()
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
		if p.matchesSnapshot(r, needle) {
			p.TblSnapshots.Select(r, 0)
			p.RefreshOutline()
			ui.FrmFind.SetTitle(fmt.Sprintf("Find Snapshot (%d/%d)", r, total))
			ui.SetStatus(fmt.Sprintf("Found '%s' at %s", needle, p.TblSnapshots.GetCell(r, 0).Text))
			return
		}
	}
}

// FindNext selects the next snapshot matching the shared Find field.
func (p *ResticPlugin) FindNext() { p.findSnapshot(false) }

// FindPrevious selects the previous snapshot matching the shared Find field.
func (p *ResticPlugin) FindPrevious() { p.findSnapshot(true) }
