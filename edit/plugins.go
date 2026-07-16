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
	"fmt"
	"lied/conf"
	"lied/ui"
	"lied/utils"
	"os"
	"path/filepath"

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

// ****************************************************************************
// hexModePlugin  (singleton – binary files are always read-only)
// ****************************************************************************

type hexModePlugin struct{}

// hexPlugin is the package-level singleton for all binary-file views.
var hexPlugin = &hexModePlugin{}

func (p *hexModePlugin) ID() string    { return "binary" }
func (p *hexModePlugin) Title() string { return filepath.Base(CurrentView.FName) }
func (p *hexModePlugin) Icon() string  { return conf.ICON_BINARY }
func (p *hexModePlugin) FocusWidget() tview.Primitive { return ui.HexView }

// Activate switches to the hex-viewer content page and refreshes the display.
func (p *hexModePlugin) Activate() {
	CurrentWidget = ui.HexView
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

// ****************************************************************************
// sqlModePlugin  (singleton)
// ****************************************************************************

type sqlModePlugin struct{}

// sqlPlugin is the package-level singleton for all SQLite3 views.
var sqlPlugin = &sqlModePlugin{}

func (p *sqlModePlugin) ID() string    { return "sqlite3" }
func (p *sqlModePlugin) Title() string { return filepath.Base(CurrentView.FName) }
func (p *sqlModePlugin) Icon() string  { return conf.ICON_DATABASE }
func (p *sqlModePlugin) FocusWidget() tview.Primitive { return ui.TxtPromptSQL }

// Activate switches to the SQL-viewer page and refreshes the schema tree.
func (p *sqlModePlugin) Activate() {
	CurrentWidget = ui.TxtPromptSQL
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

// ****************************************************************************
// explorerModePlugin  (singleton)
// ****************************************************************************

type explorerModePlugin struct{}

// explorerPlugin is the package-level singleton for all Explorer (directory) views.
var explorerPlugin = &explorerModePlugin{}

func (p *explorerModePlugin) ID() string    { return "explorer" }
func (p *explorerModePlugin) Title() string { return filepath.Base(CurrentView.FName) }
func (p *explorerModePlugin) Icon() string  { return conf.ICON_EXPLORER }
func (p *explorerModePlugin) FocusWidget() tview.Primitive { return ui.TblFiles }

// Activate switches to the file-manager page.
func (p *explorerModePlugin) Activate() {
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
