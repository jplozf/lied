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
package ui

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"github.com/rivo/tview"
)

// ****************************************************************************
// TYPES
// ****************************************************************************

// ViewStatus holds the values displayed in the status bar for a plugin view.
type ViewStatus struct {
	ReadWrite string
	Cursor    string
	Dirty     string
	Percent   string
	Size      string
	Encoding  string
}

// ViewPlugin is the interface that every pluggable view must implement.
// Adding a new kind of view (e.g. Service Manager, Process Explorer, …) only
// requires writing a new struct that satisfies this interface and registering
// it once at startup with RegisterPlugin.
type ViewPlugin interface {
	// ID returns the unique string key for this plugin (e.g. "servicemanager").
	ID() string

	// Title returns the human-readable name shown in lists and status areas.
	Title() string

	// Icon returns a single-cell glyph used in the Open Views table.
	Icon() string

	// Activate switches the UI to show this plugin's content and sets focus.
	// Implementations call ui.PgsApp.SwitchToPage / ui.PgsEditorContent.SwitchToPage
	// and ui.App.SetFocus as appropriate.
	Activate()

	// FocusWidget returns the tview.Primitive that should receive keyboard focus
	// when this view is active.
	FocusWidget() tview.Primitive

	// Open initialises or refreshes the plugin view.  param is plugin-specific
	// (e.g. a path string); pass nil when not needed.
	Open(param any) error

	// Close performs any teardown needed when the view is closed by the user.
	Close() error

	// IsDirty reports whether this view has unsaved changes.  Always false for
	// read-only tools like Service Manager.
	IsDirty() bool

	// StatusFields returns the values to display in the bottom status bar.
	StatusFields() ViewStatus

	// KeyHints returns the two-line key-hint text shown in the LblKeys bar.
	KeyHints() string
}

// ****************************************************************************
// Plugin Registry
// ****************************************************************************

var pluginRegistry = make(map[string]ViewPlugin)

// RegisterPlugin stores a plugin in the global registry, keyed by plugin.ID().
func RegisterPlugin(p ViewPlugin) {
	pluginRegistry[p.ID()] = p
}

// GetPlugin retrieves a plugin by its ID.  The second return value is false
// when no plugin with that ID has been registered.
func GetPlugin(id string) (ViewPlugin, bool) {
	p, ok := pluginRegistry[id]
	return p, ok
}

// AllPlugins returns every registered plugin in no guaranteed order.
func AllPlugins() []ViewPlugin {
	result := make([]ViewPlugin, 0, len(pluginRegistry))
	for _, p := range pluginRegistry {
		result = append(result, p)
	}
	return result
}
