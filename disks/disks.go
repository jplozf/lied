// ****************************************************************************
//
//	 _ _          _
//	| (_) ___  __| |
//	| | |  __/ (_| |
//	|_|_|\___|\__,_|
//
// ****************************************************************************
// L I E D   -   Copyright © JPL 2024
// ****************************************************************************
// Package disks provides a mounted filesystem and disk-usage plugin for lied.
// ****************************************************************************
package disks

import (
	"bufio"
	"fmt"
	"lied/dialog"
	"lied/menu"
	"lied/ui"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	PluginID        = "diskmanager"
	ContentPageName = "diskManager"
	UniqueID        = PluginID + "://" + PluginID
)

// Mount describes a mounted filesystem reported by df.
type Mount struct {
	Filesystem string
	Type       string
	Size       string
	Used       string
	Available  string
	UsePercent string
	MountPoint string
}

// DiskManagerPlugin implements ui.ViewPlugin for mounted filesystems and
// disk-usage operations.
type DiskManagerPlugin struct {
	TblMounts     *tview.Table
	TxtDetail     *tview.TextView
	layout        *tview.Flex
	mounts        []Mount
	busy          bool
	confirm       *dialog.Dialog
	humanReadable bool
}

// NewDiskManagerPlugin creates the disk manager view.
func NewDiskManagerPlugin() *DiskManagerPlugin {
	p := &DiskManagerPlugin{}
	p.TblMounts = tview.NewTable()
	p.TblMounts.SetBorder(true)
	p.TblMounts.SetTitle("Mounted filesystems")
	p.TblMounts.SetSelectable(true, false)

	p.TxtDetail = tview.NewTextView()
	p.TxtDetail.SetBorder(true)
	p.TxtDetail.SetTitle("Disk details")
	p.TxtDetail.SetScrollable(true)
	p.TxtDetail.SetWrap(false)

	p.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblMounts, 0, 2, true).
		AddItem(p.TxtDetail, 0, 2, false)
	ui.PgsEditorContent.AddPage(ContentPageName, p.layout, true, false)

	p.TblMounts.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})
	return p
}

func (p *DiskManagerPlugin) ID() string    { return PluginID }
func (p *DiskManagerPlugin) Title() string { return "Disk Manager" }
func (p *DiskManagerPlugin) Icon() string  { return "◫" }

// Activate switches to the disk manager page and focuses its mount table.
func (p *DiskManagerPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	ui.SetFindPanelVisible(false)
	p.RefreshOutline()
	ui.App.SetFocus(p.TblMounts)
	ui.LblKeys.SetText(p.KeyHints())
}

func (p *DiskManagerPlugin) FocusWidget() tview.Primitive { return p.TblMounts }

// Open refreshes the mounted filesystem list.
func (p *DiskManagerPlugin) Open(_ any) error {
	p.Refresh()
	return nil
}

func (p *DiskManagerPlugin) Close() error  { return nil }
func (p *DiskManagerPlugin) IsDirty() bool { return false }

func (p *DiskManagerPlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d mount(s)", len(p.mounts)),
		Encoding:  "df",
	}
}

func (p *DiskManagerPlugin) KeyHints() string {
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[Enter] df  [d] du  [t] Top  [h] Human sizes  [F5] Refresh  [Ctrl+T] Close"
}

func (p *DiskManagerPlugin) InternalCommand() string       { return "!disk" }
func (p *DiskManagerPlugin) CommandOpensPluginView() bool  { return true }
func (p *DiskManagerPlugin) ExecuteInternalCommand() error { return nil }

func (p *DiskManagerPlugin) ShowContextMenu(defaultMenu func()) bool {
	m := (&menu.Menu{}).New(" Disk Manager ", ui.PopupParentPage(), p.FocusWidget())
	hasMount := p.SelectedMount() != nil
	m.AddItem("mnuDiskRefresh", "Refresh mounted filesystems", func(any) {
		p.Refresh()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuDiskDF", "Show df for selected mount", func(any) {
		p.ShowDF()
	}, nil, hasMount, false)
	m.AddItem("mnuDiskDU", "Show du for selected mount", func(any) {
		p.ShowDU()
	}, nil, hasMount, false)
	m.AddItem("mnuDiskTop", "Top ten largest files", func(any) {
		p.ShowTopFiles()
	}, nil, hasMount, false)
	m.AddSeparator()
	m.AddItem("mnuDiskUnmount", "Unmount selected filesystem", func(any) {
		p.ConfirmUnmount()
	}, nil, hasMount, false)
	m.AddItem("mnuDiskMount", "Mount filesystem...", func(any) {
		p.ShowMountForm()
	}, nil, true, false)
	ui.PgsApp.AddPage("dlgDiskManagerMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgDiskManagerMenu")
	return true
}

// ConfirmUnmount asks for confirmation before unmounting the selected mount.
func (p *DiskManagerPlugin) ConfirmUnmount() {
	mount := p.SelectedMount()
	if mount == nil {
		ui.SetStatus("No filesystem selected")
		return
	}
	p.confirm = p.confirm.YesNo(
		"Unmount filesystem",
		fmt.Sprintf("Unmount %s?", mount.MountPoint),
		func(button dialog.DlgButton, _ int) {
			if button == dialog.BUTTON_YES {
				p.runMountCommand("Unmount", []string{"umount", "--", mount.MountPoint})
			}
		},
		0,
		ui.GetCurrentScreen(),
		p.FocusWidget(),
	)
	ui.PgsApp.AddPage("dlgDiskConfirmUnmount", p.confirm.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgDiskConfirmUnmount")
}

// ShowMountForm opens the mount command options in a UI form.
func (p *DiskManagerPlugin) ShowMountForm() {
	device := tview.NewInputField().SetLabel("Device: ").SetFieldWidth(42)
	mountPoint := tview.NewInputField().SetLabel("Mount point: ").SetFieldWidth(42)
	filesystem := tview.NewInputField().SetLabel("Filesystem type: ").SetFieldWidth(42)
	options := tview.NewInputField().SetLabel("Options: ").SetFieldWidth(42)
	useSudo := tview.NewCheckbox().SetLabel("Use sudo: ").SetChecked(false)

	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Mount filesystem ")
	form.AddFormItem(device)
	form.AddFormItem(mountPoint)
	form.AddFormItem(filesystem)
	form.AddFormItem(options)
	form.AddFormItem(useSudo)
	form.AddButton("Mount", func() {
		deviceName := strings.TrimSpace(device.GetText())
		target := strings.TrimSpace(mountPoint.GetText())
		if deviceName == "" || target == "" {
			ui.SetStatus("Device and mount point are required")
			return
		}
		args := []string{"mount"}
		if filesystemName := strings.TrimSpace(filesystem.GetText()); filesystemName != "" {
			args = append(args, "-t", filesystemName)
		}
		if mountOptions := strings.TrimSpace(options.GetText()); mountOptions != "" {
			args = append(args, "-o", mountOptions)
		}
		args = append(args, deviceName, target)
		if useSudo.IsChecked() {
			args = append([]string{"sudo", "-n"}, args...)
		}
		ui.PgsApp.HidePage("dlgDiskMount")
		p.runMountCommand("Mount", args)
	})
	form.AddButton("Cancel", func() {
		ui.PgsApp.HidePage("dlgDiskMount")
		ui.App.SetFocus(p.FocusWidget())
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.PgsApp.HidePage("dlgDiskMount")
			ui.App.SetFocus(p.FocusWidget())
			return nil
		}
		return event
	})
	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(form, 64, 0, true).
		AddItem(nil, 0, 1, false)
	ui.PgsApp.AddPage("dlgDiskMount", centered, true, true)
	ui.App.SetFocus(device)
}

func (p *DiskManagerPlugin) runMountCommand(operation string, args []string) {
	if p.busy {
		ui.SetStatus("A disk operation is already running")
		return
	}
	p.busy = true
	ui.SetStatus(operation + " filesystem...")
	go func() {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			if err != nil {
				p.TxtDetail.SetTitle(operation + " failed")
				p.TxtDetail.SetText(string(out) + "\nError: " + err.Error())
				ui.SetStatus(operation + " failed")
				return
			}
			p.TxtDetail.SetTitle(operation + " completed")
			p.TxtDetail.SetText(string(out))
			ui.SetStatus(operation + " completed")
			p.Refresh()
		})
	}()
}

// Refresh lists mounted filesystems using portable POSIX df output.
func (p *DiskManagerPlugin) Refresh() {
	if p.busy {
		ui.SetStatus("A disk operation is already running")
		return
	}
	p.busy = true
	ui.SetStatus("Reading mounted filesystems...")
	go func() {
		args := []string{"-P", "-T"}
		if p.humanReadable {
			args = append(args, "-h")
		}
		out, err := exec.Command("df", args...).Output()
		ui.App.QueueUpdateDraw(func() {
			defer func() { p.busy = false }()
			if err != nil {
				p.mounts = nil
				p.TblMounts.Clear()
				p.TblMounts.SetCell(0, 0, tview.NewTableCell("df failed: "+err.Error()).SetTextColor(tcell.ColorRed))
				ui.SetStatus("Unable to read mounted filesystems")
				return
			}
			p.populateMounts(string(out))
			ui.SetStatus(fmt.Sprintf("Found %d mounted filesystem(s)", len(p.mounts)))
		})
	}()
}

func (p *DiskManagerPlugin) populateMounts(output string) {
	p.mounts = parseDF(output)
	p.TblMounts.Clear()
	for col, header := range []string{"Filesystem", "Type", "Size", "Used", "Avail", "Use%", "Mount point"} {
		p.TblMounts.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold).SetSelectable(false))
	}
	p.TblMounts.SetFixed(1, 0)
	for row, mount := range p.mounts {
		values := []string{mount.Filesystem, mount.Type, mount.Size, mount.Used, mount.Available, mount.UsePercent, mount.MountPoint}
		for col, value := range values {
			color := tcell.ColorWhite
			if col == 5 {
				color = usageColor(mount.UsePercent)
			}
			p.TblMounts.SetCell(row+1, col, tview.NewTableCell(value).SetTextColor(color))
		}
	}
	p.TblMounts.SetTitle(fmt.Sprintf("Mounted filesystems (%d)", len(p.mounts)))
	if len(p.mounts) > 0 {
		p.TblMounts.Select(1, 0)
	}
	p.RefreshOutline()
}

func parseDF(output string) []Mount {
	var mounts []Mount
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 7 {
			continue
		}
		mounts = append(mounts, Mount{
			Filesystem: fields[0], Type: fields[1], Size: fields[2], Used: fields[3],
			Available: fields[4], UsePercent: fields[5], MountPoint: strings.Join(fields[6:], " "),
		})
	}
	return mounts
}

func usageColor(value string) tcell.Color {
	percent, _ := strconv.Atoi(strings.TrimSuffix(value, "%"))
	if percent >= 90 {
		return tcell.ColorRed
	}
	if percent >= 75 {
		return tcell.ColorYellow
	}
	return tcell.ColorGreen
}

func (p *DiskManagerPlugin) SelectedMount() *Mount {
	row, _ := p.TblMounts.GetSelection()
	if row <= 0 || row > len(p.mounts) {
		return nil
	}
	return &p.mounts[row-1]
}

func (p *DiskManagerPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	mount := p.SelectedMount()
	if mount == nil {
		ui.TblOutline.SetTitle("Outline")
		return
	}
	fields := [][2]string{
		{"Filesystem", mount.Filesystem}, {"Type", mount.Type}, {"Size", mount.Size},
		{"Used", mount.Used}, {"Available", mount.Available}, {"Use", mount.UsePercent},
		{"Mount point", mount.MountPoint},
	}
	for row, field := range fields {
		ui.TblOutline.SetCell(row, 0, tview.NewTableCell(field[0]).SetTextColor(tcell.ColorLightCyan))
		ui.TblOutline.SetCell(row, 1, tview.NewTableCell(field[1]).SetTextColor(tcell.ColorWhite))
	}
	ui.TblOutline.SetTitle("Outline — " + mount.MountPoint)
}

func (p *DiskManagerPlugin) ShowDF() {
	mount := p.SelectedMount()
	if mount == nil {
		return
	}
	args := []string{"df", "-P", "-T"}
	if p.humanReadable {
		args = append(args, "-h")
	}
	args = append(args, "--", mount.MountPoint)
	p.runDetail("df -P -T "+mount.MountPoint, args, "df — "+mount.MountPoint)
}

func (p *DiskManagerPlugin) ShowDU() {
	mount := p.SelectedMount()
	if mount == nil {
		return
	}
	p.runDetail("du -x -h "+mount.MountPoint, []string{"du", "-x", "-h", "--", mount.MountPoint}, "du — "+mount.MountPoint)
}

func (p *DiskManagerPlugin) ShowTopFiles() {
	mount := p.SelectedMount()
	if mount == nil {
		return
	}
	command := "find \"$1\" -xdev -type f -printf '%s\\t%p\\n' 2>/dev/null | sort -nr | head -10 | numfmt --field=1 --to=iec --suffix=B"
	p.runDetail("top ten files — "+mount.MountPoint, []string{"sh", "-c", command, "lied-top-files", mount.MountPoint}, "Top ten files — "+mount.MountPoint)
}

func (p *DiskManagerPlugin) runDetail(label string, args []string, title string) {
	if p.busy {
		ui.SetStatus("A disk operation is already running")
		return
	}
	p.busy = true
	p.TxtDetail.SetTitle(title)
	p.TxtDetail.SetText(label + "...\n")
	ui.SetStatus("Running " + label)
	go func() {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		ui.App.QueueUpdateDraw(func() {
			p.busy = false
			if err != nil {
				p.TxtDetail.SetText(string(out) + "\nError: " + err.Error())
				ui.SetStatus(label + " failed")
				return
			}
			p.TxtDetail.SetText(string(out))
			p.TxtDetail.ScrollToBeginning()
			ui.SetStatus(label + " completed")
		})
	}()
}

// HandleInput handles disk-view actions; callers may handle global close keys.
func (p *DiskManagerPlugin) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF5:
		p.Refresh()
		return nil
	case tcell.KeyCtrlT:
		return event
	case tcell.KeyEnter:
		p.ShowDF()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'h':
			p.humanReadable = !p.humanReadable
			p.Refresh()
			return nil
		case 'd':
			p.ShowDU()
			return nil
		case 't':
			p.ShowTopFiles()
			return nil
		}
	}
	return event
}
