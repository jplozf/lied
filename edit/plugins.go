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
// Package edit - built-in ViewPlugin implementations for the four standard
// editing modes (Text, Binary, SQLite3, Explorer).  Each satisfies the
// ui.ViewPlugin interface so that mode-switching logic can be expressed
// uniformly via plugin.Activate() / plugin.StatusFields() / etc. instead of
// large if/else chains scattered across edit.go and lied.go.
// ****************************************************************************
package edit

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"errors"
	"fmt"
	"lied/conf"
	"lied/dialog"
	"lied/menu"
	"lied/preview"
	"lied/ui"
	"lied/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/pgavlin/femto"
	"github.com/rivo/tview"
)

// ****************************************************************************
// TextModePlugin
// ****************************************************************************

// TextModePlugin is the ViewPlugin implementation for text-file views
// (Mode == Text).  One instance is created per open file so that the
// TblOpenFiles loop can call IsDirty() / Icon() on individual entries without
// referencing the global CurrentView.
type TextModePlugin struct {
	// buf is a stable pointer to the buffer that belongs to this particular
	// ViewScreen entry.  It must be updated whenever ViewScreen.FemtoBuffer is
	// replaced (Follow mode, RestoreBuffer, ReplaceAll).
	buf *femto.Buffer
}

// NewTextModePlugin constructs a TextModePlugin for the given buffer.
// Exported so that callers outside the edit package (e.g. lied.go) can create
// an appropriate plugin when building a ViewScreen directly.
func NewTextModePlugin(buf *femto.Buffer) *TextModePlugin {
	return &TextModePlugin{buf: buf}
}

func (p *TextModePlugin) ID() string    { return "text" }
func (p *TextModePlugin) Title() string { return filepath.Base(CurrentView.FName) }

// Icon returns a blank space for a clean text file.  The TblOpenFiles render
// loop overrides it with ICON_MODIFIED when IsDirty() is true, which keeps the
// pattern consistent with all other plugin types.
func (p *TextModePlugin) Icon() string { return " " }

// FocusWidget returns the main text editor primitive.
func (p *TextModePlugin) FocusWidget() tview.Primitive { return ui.EdtMain }

// Activate switches the application to the text-editor content page, loads the
// current buffer and configures the Find/Replace form for text mode.
func (p *TextModePlugin) Activate() {
	CurrentWidget = ui.EdtMain
	ui.SetFindPanelVisible(true)
	if CurrentView.FemtoBuffer != nil {
		ui.EdtMain.OpenBuffer(CurrentView.FemtoBuffer)
	}
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage("textEditor")
	ui.TxtReplace.SetDisabled(!ui.ChkToggleReplace.IsChecked())
	ui.ChkToggleReplace.SetDisabled(false)
	ui.FrmFind.GetButton(2).SetDisabled(!ui.ChkToggleReplace.IsChecked())
	ui.FrmFind.GetButton(3).SetDisabled(!ui.ChkToggleReplace.IsChecked())
	ui.DpdSearchType.SetDisabled(true)
	ui.DpdSearchType.SetCurrentOption(0)
	ui.ChkCase.SetDisabled(false)
	RefreshTextOutline()
	ui.LblKeys.SetText(conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS)
}

// Open is a no-op; the file loading is handled by edit.OpenView.
func (p *TextModePlugin) Open(_ any) error { return nil }

// Close is a no-op; save-on-close is handled by the main quit flow.
func (p *TextModePlugin) Close() error { return nil }

// IsDirty returns true when the buffer has unsaved changes.
func (p *TextModePlugin) IsDirty() bool { return p.buf != nil && p.buf.Modified() }

// StatusFields computes the current text-editor label values from CurrentView.
// It is safe to call on every status-update tick because the computation is
// cheap (no I/O).
func (p *TextModePlugin) StatusFields() ui.ViewStatus {
	if CurrentView.FemtoBuffer == nil {
		return ui.ViewStatus{}
	}
	rw := "RO"
	if CurrentView.Follow {
		rw = "FL"
	} else if CurrentView.ReadWrite {
		rw = "RW"
	}
	dirty := ""
	if CurrentView.FemtoBuffer.Modified() {
		dirty = "MODIFIED"
	}
	cursor := ""
	percent := ""
	if CurrentView.FemtoBuffer.NumLines > 0 {
		x := CurrentView.FemtoBuffer.Cursor.X + 1
		y := CurrentView.FemtoBuffer.Cursor.Y + 1
		cursor = fmt.Sprintf("Ln %d, Col %d", y, x)
		percent = fmt.Sprintf("%d%%", int((float32(CurrentView.FemtoBuffer.Cursor.Y)/float32(CurrentView.FemtoBuffer.NumLines))*100.0))
	}
	return ui.ViewStatus{
		ReadWrite: rw,
		Cursor:    cursor,
		Dirty:     dirty,
		Percent:   percent,
		Size:      utils.HumanFileSize(float64(CurrentView.FemtoBuffer.Len())),
		Encoding:  CurrentView.Encoding,
	}
}

// KeyHints returns the standard editor key-hint text.
func (p *TextModePlugin) KeyHints() string {
	return conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS
}

func (p *TextModePlugin) InternalCommand() string       { return "" }
func (p *TextModePlugin) CommandOpensPluginView() bool  { return false }
func (p *TextModePlugin) ExecuteInternalCommand() error { return nil }
func (p *TextModePlugin) ShowContextMenu(defaultMenu func()) bool {
	if defaultMenu != nil {
		defaultMenu()
		return true
	}
	return false
}

// ****************************************************************************
// hexModePlugin  (singleton – binary files are always read-only)
// ****************************************************************************

type hexModePlugin struct{}

// hexPlugin is the package-level singleton for all binary-file views.
var hexPlugin = &hexModePlugin{}
var hexMenusInitialized bool
var MnuHex *menu.Menu
var DlgHexOffset *dialog.Dialog

func (p *hexModePlugin) initMenus() {
	if hexMenusInitialized {
		return
	}

	MnuHex = MnuHex.New("Hex Viewer", "edit", ui.HexView)
	MnuHex.AddItem("mnuHexRefresh", "Refresh hex view", p.menuRefresh, nil, true, false)
	MnuHex.AddItem("mnuHexFind", "Find...", p.menuFind, nil, true, false)
	MnuHex.AddItem("mnuHexCopyOffset", "Copy current offset", p.menuCopyOffset, nil, true, false)
	MnuHex.AddItem("mnuHexGotoOffset", "Go to offset...", p.menuGotoOffset, nil, true, false)
	MnuHex.AddItem("mnuHexFileInfo", "Show file info", p.menuFileInfo, nil, true, false)
	MnuHex.AddItem("mnuHexOpenViews", "Focus Open Views", p.menuFocusOpenViews, nil, true, false)
	MnuHex.AddSeparator()
	MnuHex.AddItem("mnuHexClose", "Close current view", p.menuCloseCurrent, nil, true, false)

	ui.PgsApp.AddPage("dlgHexAction", MnuHex.Popup(), true, false)
	hexMenusInitialized = true
}

func (p *hexModePlugin) updateMenuState() {
	hasContent := len(CurrentView.ContentBytes) > 0
	MnuHex.SetEnabled("mnuHexRefresh", hasContent)
	MnuHex.SetEnabled("mnuHexFind", hasContent)
	MnuHex.SetEnabled("mnuHexCopyOffset", hasContent)
	MnuHex.SetEnabled("mnuHexGotoOffset", hasContent)
	MnuHex.SetEnabled("mnuHexFileInfo", CurrentView.FName != "")
	MnuHex.SetEnabled("mnuHexOpenViews", true)
	MnuHex.SetEnabled("mnuHexClose", true)
}

func (p *hexModePlugin) menuRefresh(_ any) {
	CurrentView.HexContentDirty = true
	displayBinaryContent()
	ui.SetStatus("Hex view refreshed")
}

func (p *hexModePlugin) menuFind(_ any) {
	ui.FrmFind.GetButton(0).SetSelectedFunc(FindNext)
	ui.FrmFind.GetButton(1).SetSelectedFunc(FindPrevious)
	ui.FrmFind.GetButton(2).SetDisabled(true)
	ui.FrmFind.GetButton(3).SetDisabled(true)
	ui.App.SetFocus(ui.FrmFind)
}

func (p *hexModePlugin) menuCopyOffset(_ any) {
	if len(CurrentView.ContentBytes) == 0 {
		ui.SetStatus("No hex content available")
		return
	}
	offsetLine, _ := ui.HexView.GetScrollOffset()
	offset := offsetLine * 16
	offsetText := fmt.Sprintf("%08X", offset)
	if err := clipboard.WriteAll(offsetText); err != nil {
		ui.SetStatus(err.Error())
		return
	}
	ui.SetStatus("Copied offset " + offsetText)
}

func parseHexOffset(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("empty offset")
	}

	if parsed, err := strconv.ParseInt(value, 0, 64); err == nil {
		if parsed < 0 {
			return 0, errors.New("offset must be >= 0")
		}
		return int(parsed), nil
	}

	parsed, err := strconv.ParseInt(value, 16, 64)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, errors.New("offset must be >= 0")
	}
	return int(parsed), nil
}

func (p *hexModePlugin) menuGotoOffset(_ any) {
	if len(CurrentView.ContentBytes) == 0 {
		ui.SetStatus("No hex content available")
		return
	}

	offsetLine, _ := ui.HexView.GetScrollOffset()
	defaultValue := fmt.Sprintf("%X", offsetLine*16)

	DlgHexOffset = DlgHexOffset.Input(
		"Go to offset",
		"Enter byte offset (hex like 1A3 or 0x1A3, decimal accepted)",
		defaultValue,
		func(rc dialog.DlgButton, idx int) {
			if rc != dialog.BUTTON_OK {
				return
			}

			offset, err := parseHexOffset(DlgHexOffset.Value)
			if err != nil {
				ui.SetStatus("Invalid offset: " + err.Error())
				return
			}

			maxOffset := len(CurrentView.ContentBytes) - 1
			if offset > maxOffset {
				offset = maxOffset
			}

			line := offset / 16
			ui.HexView.ScrollTo(line, 0)
			ui.LblCursor.SetText(fmt.Sprintf("Offset: %08X", offset))

			percent := 100
			if len(CurrentView.ContentBytes) > 1 {
				percent = int((float64(offset) / float64(len(CurrentView.ContentBytes)-1)) * 100.0)
			}
			ui.LblPercent.SetText(fmt.Sprintf("%d%%", percent))
			ui.SetStatus(fmt.Sprintf("Moved to offset %08X", offset))
		},
		0,
		"edit",
		ui.HexView,
	)
	ui.PgsApp.AddPage("dlgHexGotoOffset", DlgHexOffset.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgHexGotoOffset")
}

func (p *hexModePlugin) menuFileInfo(_ any) {
	if CurrentView.FName == "" {
		ui.SetStatus("No file selected")
		return
	}
	ui.DisplayExifInfo(CurrentView.FName)
	ui.SetStatus("File info updated")
}

func (p *hexModePlugin) menuFocusOpenViews(_ any) {
	ui.App.SetFocus(ui.TblOpenFiles)
}

func (p *hexModePlugin) menuCloseCurrent(_ any) {
	CloseCurrentFile()
}

func (p *hexModePlugin) ID() string                   { return "binary" }
func (p *hexModePlugin) Title() string                { return filepath.Base(CurrentView.FName) }
func (p *hexModePlugin) Icon() string                 { return conf.ICON_BINARY }
func (p *hexModePlugin) FocusWidget() tview.Primitive { return ui.HexView }

// Activate switches to the hex-viewer content page and refreshes the display.
func (p *hexModePlugin) Activate() {
	CurrentWidget = ui.HexView
	ui.SetFindPanelVisible(true)
	CurrentView.HexContentDirty = true
	displayBinaryContent()
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage("hexViewer")
	ui.DisplayExifInfo(CurrentView.FName)
	ui.ConfigureFindFormForBinary(true)
	ui.LblKeys.SetText(conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS)
}

func (p *hexModePlugin) Open(_ any) error { return nil }
func (p *hexModePlugin) Close() error     { return nil }

// IsDirty always returns false; binary files are opened read-only.
func (p *hexModePlugin) IsDirty() bool { return false }

func (p *hexModePlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "RO",
		Cursor:    "",
		Dirty:     "",
		Percent:   "",
		Size:      utils.HumanFileSize(float64(len(CurrentView.ContentBytes))),
		Encoding:  CurrentView.Encoding,
	}
}

func (p *hexModePlugin) KeyHints() string {
	return conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS
}

func (p *hexModePlugin) InternalCommand() string       { return "" }
func (p *hexModePlugin) CommandOpensPluginView() bool  { return false }
func (p *hexModePlugin) ExecuteInternalCommand() error { return nil }
func (p *hexModePlugin) ShowContextMenu(defaultMenu func()) bool {
	p.initMenus()
	p.updateMenuState()
	ui.PgsApp.ShowPage("dlgHexAction")
	return true
}

// ****************************************************************************
// sqlModePlugin  (singleton)
// ****************************************************************************

type sqlModePlugin struct{}

// sqlPlugin is the package-level singleton for all SQLite3 views.
var sqlPlugin = &sqlModePlugin{}
var sqlMenusInitialized bool

func (p *sqlModePlugin) initMenus() {
	if sqlMenusInitialized {
		return
	}

	MnuSQL = MnuSQL.New("SQLite3", "edit", ui.TxtPromptSQL)
	MnuSQL.AddItem("mnuSQLRun", "Run query", p.menuRunQuery, nil, true, false)
	MnuSQL.AddItem("mnuSQLRefresh", "Refresh schema tree", p.menuRefreshSchema, nil, true, false)
	MnuSQL.AddItem("mnuSQLTables", "List tables", p.menuListTables, nil, true, false)
	MnuSQL.AddItem("mnuSQLDatabase", "Show databases", p.menuShowDatabases, nil, true, false)
	MnuSQL.AddSeparator()
	MnuSQL.AddItem("mnuSQLCopyCell", "Copy current cell", DoCopyCell, nil, false, false)
	MnuSQL.AddItem("mnuSQLExportCell", "Export cell", DoExportCell, nil, false, false)
	MnuSQL.AddItem("mnuSQLExportRow", "Export row to CSV", DoExportRow, nil, false, false)
	MnuSQL.AddItem("mnuSQLExportAll", "Export all to CSV", DoExportAll, nil, false, false)
	MnuSQL.AddItem("mnuSQLExportAllJSON", "Export all to JSON", DoExportAllJSON, nil, false, false)
	MnuSQL.AddSeparator()
	MnuSQL.AddItem("mnuSQLOpen", "Open database...", p.menuOpenDatabase, nil, true, false)
	MnuSQL.AddItem("mnuSQLNew", "New SQLite3 database...", p.menuNewDatabase, nil, true, false)
	MnuSQL.AddItem("mnuSQLClose", "Close current database", p.menuCloseDatabase, nil, true, false)

	ui.PgsApp.AddPage("dlgSQLiteAction", MnuSQL.Popup(), true, false)
	sqlMenusInitialized = true
}

func (p *sqlModePlugin) updateMenuState() {
	hasDB := CurrentView.Database != nil
	r, _ := ui.TblSQLOutput.GetSelection()
	hasSelectedRow := r > 0
	hasResultRows := ui.TblSQLOutput.GetRowCount() > 1

	MnuSQL.SetEnabled("mnuSQLRun", hasDB)
	MnuSQL.SetEnabled("mnuSQLRefresh", hasDB)
	MnuSQL.SetEnabled("mnuSQLTables", hasDB)
	MnuSQL.SetEnabled("mnuSQLDatabase", hasDB)
	MnuSQL.SetEnabled("mnuSQLClose", hasDB)

	MnuSQL.SetEnabled("mnuSQLCopyCell", hasSelectedRow)
	MnuSQL.SetEnabled("mnuSQLExportCell", hasSelectedRow)
	MnuSQL.SetEnabled("mnuSQLExportRow", hasSelectedRow)
	MnuSQL.SetEnabled("mnuSQLExportAll", hasResultRows)
	MnuSQL.SetEnabled("mnuSQLExportAllJSON", hasResultRows)
}

func (p *sqlModePlugin) runAndClearPrompt(query string) {
	err := XeqSQL(query)
	if err == nil {
		ui.TxtPromptSQL.SetText("", true)
	} else {
		ui.TxtPromptSQL.SetText(ui.TxtPromptSQL.GetText()+" => "+err.Error(), true)
	}
}

func (p *sqlModePlugin) menuRunQuery(_ any) {
	query := strings.TrimSpace(ui.TxtPromptSQL.GetText())
	if query == "" {
		ui.SetStatus("No SQL query to run")
		return
	}
	p.runAndClearPrompt(query)
}

func (p *sqlModePlugin) menuRefreshSchema(_ any) {
	if CurrentView.Database == nil {
		ui.SetStatus("No open database")
		return
	}
	showTreeDB()
	ui.SetStatus("Schema tree refreshed")
}

func (p *sqlModePlugin) menuListTables(_ any) {
	p.runAndClearPrompt(".TABLE")
}

func (p *sqlModePlugin) menuShowDatabases(_ any) {
	p.runAndClearPrompt(".DATABASE")
}

func (p *sqlModePlugin) menuOpenDatabase(_ any) {
	path := conf.ConfigGeneral.Workspace
	if CurrentView.FName != "" {
		path = filepath.Dir(CurrentView.FName)
	}
	DoOpenDB(path)
}

func (p *sqlModePlugin) menuNewDatabase(_ any) {
	path := conf.ConfigGeneral.Workspace
	if CurrentView.FName != "" {
		path = filepath.Dir(CurrentView.FName)
	}
	DoNewDB(path)
}

func (p *sqlModePlugin) menuCloseDatabase(_ any) {
	if CurrentView.Database == nil {
		ui.SetStatus("No open database")
		return
	}
	CloseCurrentFile()
}

func (p *sqlModePlugin) ID() string                   { return "sqlite3" }
func (p *sqlModePlugin) Title() string                { return filepath.Base(CurrentView.FName) }
func (p *sqlModePlugin) Icon() string                 { return conf.ICON_DATABASE }
func (p *sqlModePlugin) FocusWidget() tview.Primitive { return ui.TxtPromptSQL }

// Activate switches to the SQL-viewer page and refreshes the schema tree.
func (p *sqlModePlugin) Activate() {
	CurrentWidget = ui.TxtPromptSQL
	ui.SetFindPanelVisible(false)
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage("sqlViewer")
	showTreeDB()
	ui.LblKeys.SetText(conf.FKEY_LABELS + "\nCtrl+O=Open Ctrl+S=Save")
}

func (p *sqlModePlugin) Open(_ any) error { return nil }
func (p *sqlModePlugin) Close() error     { return nil }
func (p *sqlModePlugin) IsDirty() bool    { return false }

func (p *sqlModePlugin) StatusFields() ui.ViewStatus {
	sz := ""
	if fi, err := os.Stat(CurrentView.FName); err == nil {
		sz = utils.HumanFileSize(float64(fi.Size()))
	}
	return ui.ViewStatus{
		ReadWrite: "RW",
		Cursor:    "SQLite3",
		Dirty:     "",
		Percent:   "",
		Size:      sz,
		Encoding:  "SQLite3",
	}
}

func (p *sqlModePlugin) KeyHints() string {
	return conf.FKEY_LABELS + "\nCtrl+O=Open Ctrl+S=Save"
}

func (p *sqlModePlugin) InternalCommand() string       { return "" }
func (p *sqlModePlugin) CommandOpensPluginView() bool  { return false }
func (p *sqlModePlugin) ExecuteInternalCommand() error { return nil }
func (p *sqlModePlugin) ShowContextMenu(defaultMenu func()) bool {
	p.initMenus()
	p.updateMenuState()
	ui.PgsApp.ShowPage("dlgSQLiteAction")
	return true
}

// ****************************************************************************
// explorerModePlugin  (singleton)
// ****************************************************************************

type explorerModePlugin struct{}

// explorerPlugin is the package-level singleton for all Explorer (directory) views.
var explorerPlugin = &explorerModePlugin{}
var explorerMenusInitialized bool

func (p *explorerModePlugin) initMenus() {
	if explorerMenusInitialized {
		return
	}
	// Explorer context menus are defined by the explorer plugin implementation.
	MnuFiles = MnuFiles.New("Files", "fileManager", ui.TblFiles)
	MnuFiles.AddItem("mnuEdit", "Edit", DoEdit, nil, true, false)
	MnuFiles.AddItem("mnuSelect", "Select / Unselect All", SelectAll, nil, true, false)
	MnuFiles.AddItem("mnuDelete", "Delete", DoDelete, nil, true, false)
	MnuFiles.AddItem("mnuRename", "Rename", DoRename, nil, true, false)
	MnuFiles.AddItem("mnuCopy", "Copy", DoCopy, nil, true, false)
	MnuFiles.AddItem("mnuCut", "Cut", DoCut, nil, true, false)
	MnuFiles.AddItem("mnuPaste", "Paste", DoPaste, nil, false, false)
	MnuFiles.AddItem("mnuCreateFile", "New File", DoNewFile, nil, true, false)
	MnuFiles.AddItem("mnuCreateFolder", "New Folder", DoNewFolder, nil, true, false)
	MnuFiles.AddItem("mnuZip", "Zip", DoZip, nil, true, false)
	MnuFiles.AddItem("mnuSnapshot", "Snapshot", DoSnapshot, nil, true, false)
	MnuFiles.AddItem("mnuShowHiddenFiles", "Show hidden files", DoSwitchHiddenFiles, nil, true, false)
	ui.PgsApp.AddPage("dlgFileAction", MnuFiles.Popup(), true, false)

	MnuFilesSort = MnuFilesSort.New("Sort by", "fileManager", ui.TblFiles)
	MnuFilesSort.AddItem("mnuSortNameA", "Name Ascending", doSortNameA, nil, false, true)
	MnuFilesSort.AddItem("mnuSortNameD", "Name Descending", doSortNameD, nil, true, false)
	MnuFilesSort.AddItem("mnuSortSizeA", "Size Ascending", doSortSizeA, nil, true, false)
	MnuFilesSort.AddItem("mnuSortSizeD", "Size Descending", doSortSizeD, nil, true, false)
	MnuFilesSort.AddItem("mnuSortTimeA", "Time Ascending", doSortTimeA, nil, true, false)
	MnuFilesSort.AddItem("mnuSortTimeD", "Time Descending", doSortTimeD, nil, true, false)
	ui.PgsApp.AddPage("dlgFileSort", MnuFilesSort.Popup(), true, false)

	explorerMenusInitialized = true
}

func (p *explorerModePlugin) ID() string                   { return "explorer" }
func (p *explorerModePlugin) Title() string                { return filepath.Base(CurrentView.FName) }
func (p *explorerModePlugin) Icon() string                 { return conf.ICON_EXPLORER }
func (p *explorerModePlugin) FocusWidget() tview.Primitive { return ui.TblFiles }

// Activate switches to the file-manager page.
func (p *explorerModePlugin) Activate() {
	p.initMenus()
	CurrentWidget = ui.TblFiles
	ui.PgsApp.SwitchToPage("fileManager")
	ui.LblKeys.SetText(conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS)
}

func (p *explorerModePlugin) Open(_ any) error { return nil }
func (p *explorerModePlugin) Close() error     { return nil }
func (p *explorerModePlugin) IsDirty() bool    { return false }

func (p *explorerModePlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    "Explorer",
		Dirty:     "",
		Percent:   "",
		Size:      "",
		Encoding:  "",
	}
}

func (p *explorerModePlugin) KeyHints() string {
	return conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS
}

func (p *explorerModePlugin) InternalCommand() string      { return "!expl" }
func (p *explorerModePlugin) CommandOpensPluginView() bool { return false }

func (p *explorerModePlugin) ExecuteInternalCommand() error {
	OpenView(PreferredWorkingDirectory(), false)
	return nil
}

func (p *explorerModePlugin) showFilesMenu() {
	idx, _ := ui.TblFiles.GetSelection()
	targetType := strings.TrimSpace(ui.TblFiles.GetCell(idx, 4).Text)

	if targetType == "FOLDER" {
		MnuFiles.SetEnabled("mnuEdit", false)
		MnuFiles.SetEnabled("mnuOpen", false)
		MnuFiles.SetEnabled("mnuEncrypt", false)
	}

	if targetType == "FILE" {
		fName := filepath.Join(CurrentView.FName, ui.TblFiles.GetCell(idx, 2).Text)
		mtype, xtype := preview.DisplayFilePreview(fName)
		canEdit := strings.HasPrefix(mtype, "text") || strings.HasSuffix(xtype, "sqlite3") || utils.IsBinaryFile(fName)
		MnuFiles.SetEnabled("mnuEdit", canEdit)
	}

	if Hidden {
		MnuFiles.SetLabel("mnuShowHiddenFiles", "Hide hidden files")
	} else {
		MnuFiles.SetLabel("mnuShowHiddenFiles", "Show hidden files")
	}
	ui.PgsApp.ShowPage("dlgFileAction")
}

func (p *explorerModePlugin) showSortMenu() {
	p.initMenus()
	ui.PgsApp.ShowPage("dlgFileSort")
}

func (p *explorerModePlugin) ShowContextMenu(defaultMenu func()) bool {
	p.initMenus()
	p.showFilesMenu()
	return true
}

// ShowExplorerSortMenu displays the explorer sort menu through the explorer
// plugin implementation.
func ShowExplorerSortMenu() {
	if CurrentView.Mode != Explorer || CurrentView.Plugin != explorerPlugin {
		return
	}
	explorerPlugin.showSortMenu()
}

// editorCommandPlugin provides !edit as a plugin-registered internal command.
// It is command-launchable only and does not create its own plugin view.
type editorCommandPlugin struct{}

func (p *editorCommandPlugin) ID() string                   { return "editor" }
func (p *editorCommandPlugin) Title() string                { return "Editor" }
func (p *editorCommandPlugin) Icon() string                 { return " " }
func (p *editorCommandPlugin) FocusWidget() tview.Primitive { return ui.EdtMain }
func (p *editorCommandPlugin) Activate()                    {}
func (p *editorCommandPlugin) Open(_ any) error             { return nil }
func (p *editorCommandPlugin) Close() error                 { return nil }
func (p *editorCommandPlugin) IsDirty() bool                { return false }
func (p *editorCommandPlugin) StatusFields() ui.ViewStatus  { return ui.ViewStatus{} }
func (p *editorCommandPlugin) KeyHints() string             { return conf.FKEY_LABELS + "\n" + conf.CKEY_LABELS }
func (p *editorCommandPlugin) InternalCommand() string      { return "!edit" }
func (p *editorCommandPlugin) CommandOpensPluginView() bool { return false }

func (p *editorCommandPlugin) ExecuteInternalCommand() error {
	if CurrentView.FName == "" {
		return errors.New("no file selected to edit")
	}

	if CurrentView.Mode == Explorer {
		idx, _ := ui.TblFiles.GetSelection()
		if idx <= 0 {
			return errors.New("select a file in explorer first")
		}
		DoEdit(nil)
		return nil
	}

	if utils.IsDir(CurrentView.FName) {
		return errors.New("current view is a directory, not a text file")
	}

	SwitchToEditor(CurrentView.FName)
	return nil
}

func (p *editorCommandPlugin) ShowContextMenu(defaultMenu func()) bool {
	if defaultMenu != nil {
		defaultMenu()
		return true
	}
	return false
}

var editorPlugin = &editorCommandPlugin{}

// RegisterInternalCommandPlugins registers command-driven built-ins.
func RegisterInternalCommandPlugins() {
	ui.RegisterPlugin(explorerPlugin)
	ui.RegisterPlugin(editorPlugin)
}
