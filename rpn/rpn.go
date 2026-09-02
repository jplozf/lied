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
// Package rpn provides an RPN (Reverse Polish Notation) calculator plugin for
// lied.  Tokens/expressions are typed into a command input at the bottom of
// the view and the resulting stack is displayed in the main panel above it.
// ****************************************************************************
package rpn

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"fmt"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	PluginID        = "rpncalculator"
	ContentPageName = "rpnCalculator"
	UniqueID        = PluginID + "://" + PluginID
)

// ****************************************************************************
// RPNPlugin
// ****************************************************************************

// RPNPlugin implements ui.ViewPlugin and shows an RPN calculator: the stack
// is rendered in the main panel while commands/operands are typed into the
// input field at the bottom.
type RPNPlugin struct {
	// TxtStack shows the current contents of the calculator stack.
	TxtStack *tview.TextView
	// TxtInput is the command/operand entry field.
	TxtInput *tview.InputField
	layout   *tview.Flex
	stack    []float64
	// history holds previously entered input lines, oldest first.
	history []string
	// historyIdx is the current position while navigating history with
	// Up/Down; it equals len(history) when not navigating.
	historyIdx int
	// angMode is either "DEG" or "RAD" and controls how sin/cos/tan interpret
	// their argument and how asin/acos/atan report their result.
	angMode string
}

// NewRPNPlugin creates and wires up the RPN Calculator plugin.  It registers
// its content page directly with ui.PgsEditorContent so that it is rendered
// within the standard editor frame (header / key-hints / status bar) without
// needing its own full-screen layout.
func NewRPNPlugin() *RPNPlugin {
	p := &RPNPlugin{angMode: "DEG"}

	p.TxtStack = tview.NewTextView()
	p.TxtStack.SetBorder(true)
	p.TxtStack.SetTitle("Stack")
	p.TxtStack.SetDynamicColors(true)
	p.TxtStack.SetScrollable(true)
	p.TxtStack.SetWrap(true)

	p.TxtInput = tview.NewInputField()
	p.TxtInput.SetBorder(true)
	p.TxtInput.SetTitle("Input (Enter to evaluate)")
	p.TxtInput.SetLabel("> ")
	p.TxtInput.SetFieldWidth(0)
	p.TxtInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			line := p.TxtInput.GetText()
			p.pushHistory(line)
			p.processInput(line)
			p.TxtInput.SetText("")
		}
	})

	p.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TxtStack, 0, 1, false).
		AddItem(p.TxtInput, 3, 0, true)

	ui.PgsEditorContent.AddPage(ContentPageName, p.layout, true, false)

	p.updateStackView()

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *RPNPlugin) ID() string    { return PluginID }
func (p *RPNPlugin) Title() string { return "RPN Calculator" }
func (p *RPNPlugin) Icon() string  { return "🖩" }

// Activate switches the application to the RPN calculator content page and
// sets focus to the input field.
func (p *RPNPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	ui.SetFindPanelVisible(false)
	p.RefreshOutline()
	ui.App.SetFocus(p.TxtInput)
	ui.LblKeys.SetText(p.KeyHints())
}

// FocusWidget returns the input field as the primary focus target.
func (p *RPNPlugin) FocusWidget() tview.Primitive { return p.TxtInput }

// Open is a no-op; the stack is kept across activations. param is unused.
func (p *RPNPlugin) Open(_ any) error { return nil }

// Close is a no-op; the RPN calculator holds no resources that need cleanup.
func (p *RPNPlugin) Close() error { return nil }

// IsDirty always returns false because the calculator has no file to save.
func (p *RPNPlugin) IsDirty() bool { return false }

// StatusFields returns values for the bottom status bar widgets.
func (p *RPNPlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d value(s)", len(p.stack)),
		Encoding:  "rpn",
	}
}

// KeyHints returns the two-line key-hint string for the LblKeys bar.
func (p *RPNPlugin) KeyHints() string {
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[Enter] Evaluate  [↑/↓] History  [F5] Clear stack  [Ctrl+T] Close"
}

func (p *RPNPlugin) InternalCommand() string      { return "!rpn" }
func (p *RPNPlugin) CommandOpensPluginView() bool { return true }

func (p *RPNPlugin) ExecuteInternalCommand() error {
	// Command is handled by opening plugin view in dispatcher.
	return nil
}

func (p *RPNPlugin) ShowContextMenu(defaultMenu func()) bool {
	m := (&menu.Menu{}).New(" RPN Calculator ", ui.PopupParentPage(), p.FocusWidget())
	edit.AddOpenViewsMenuItems(m)
	m.AddItem("mnuRPNClear", "Clear stack", func(any) {
		p.Clear()
	}, nil, len(p.stack) > 0, false)

	ui.PgsApp.AddPage("dlgRPNCalculatorMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgRPNCalculatorMenu")
	return true
}

// ****************************************************************************
// Calculator logic
// ****************************************************************************

// Clear empties the stack and refreshes the display.
func (p *RPNPlugin) Clear() {
	p.stack = p.stack[:0]
	p.updateStackView()
	ui.SetStatus("Stack cleared")
}

// pushHistory appends a non-empty input line to the history and resets the
// navigation cursor to point past the end (i.e. an empty new entry).
func (p *RPNPlugin) pushHistory(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	p.history = append(p.history, line)
	p.historyIdx = len(p.history)
}

// History returns the input history, oldest first, for persistence.
func (p *RPNPlugin) History() []string { return p.history }

// LoadHistory replaces the input history with lines restored from disk and
// resets the navigation cursor to point past the end.
func (p *RPNPlugin) LoadHistory(lines []string) {
	p.history = append([]string{}, lines...)
	p.historyIdx = len(p.history)
	p.RefreshOutline()
}

// Stack returns the calculator stack, bottom first, for persistence.
func (p *RPNPlugin) Stack() []float64 { return p.stack }

// LoadStack replaces the calculator stack with values restored from disk.
func (p *RPNPlugin) LoadStack(values []float64) {
	p.stack = append([]float64{}, values...)
	p.updateStackView()
}

// AngMode returns the current angular mode ("DEG" or "RAD") for persistence.
func (p *RPNPlugin) AngMode() string { return p.angMode }

// SetAngMode restores the angular mode from disk; invalid values are ignored.
func (p *RPNPlugin) SetAngMode(mode string) {
	if mode != "DEG" && mode != "RAD" {
		return
	}
	p.angMode = mode
	p.RefreshOutline()
}

// HistoryPrev moves back one entry in the input history and fills the input
// field with it. It is a no-op when there is no older entry.
func (p *RPNPlugin) HistoryPrev() {
	if len(p.history) == 0 || p.historyIdx <= 0 {
		return
	}
	p.historyIdx--
	p.TxtInput.SetText(p.history[p.historyIdx])
}

// HistoryNext moves forward one entry in the input history, clearing the
// input field once the end of the history is reached.
func (p *RPNPlugin) HistoryNext() {
	if len(p.history) == 0 || p.historyIdx >= len(p.history) {
		return
	}
	p.historyIdx++
	if p.historyIdx == len(p.history) {
		p.TxtInput.SetText("")
	} else {
		p.TxtInput.SetText(p.history[p.historyIdx])
	}
}

func (p *RPNPlugin) push(v float64) {
	p.stack = append(p.stack, v)
	p.updateStackView()
}

func (p *RPNPlugin) pop() (float64, error) {
	if len(p.stack) == 0 {
		return 0, fmt.Errorf("stack underflow")
	}
	v := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	return v, nil
}

// dup duplicates the top of the stack.
func (p *RPNPlugin) dup() error {
	if len(p.stack) < 1 {
		return fmt.Errorf("not enough operands for 'dup'")
	}
	p.push(p.stack[len(p.stack)-1])
	return nil
}

// drop discards the top of the stack.
func (p *RPNPlugin) drop() error {
	_, err := p.pop()
	if err != nil {
		return fmt.Errorf("not enough operands for 'drop'")
	}
	p.updateStackView()
	return nil
}

// swap exchanges the top two stack values.
func (p *RPNPlugin) swap() error {
	if len(p.stack) < 2 {
		return fmt.Errorf("not enough operands for 'swap'")
	}
	n := len(p.stack)
	p.stack[n-1], p.stack[n-2] = p.stack[n-2], p.stack[n-1]
	p.updateStackView()
	return nil
}

// over copies the second-from-top value onto the top of the stack.
func (p *RPNPlugin) over() error {
	if len(p.stack) < 2 {
		return fmt.Errorf("not enough operands for 'over'")
	}
	p.push(p.stack[len(p.stack)-2])
	return nil
}

// rot rotates the top three values: (a b c -> b c a).
func (p *RPNPlugin) rot() error {
	if len(p.stack) < 3 {
		return fmt.Errorf("not enough operands for 'rot'")
	}
	n := len(p.stack)
	a := p.stack[n-3]
	p.stack[n-3] = p.stack[n-2]
	p.stack[n-2] = p.stack[n-1]
	p.stack[n-1] = a
	p.updateStackView()
	return nil
}

func (p *RPNPlugin) updateStackView() {
	var sb strings.Builder
	sb.WriteString("[::b]Stack (top at bottom):[::-]\n")
	for i := len(p.stack) - 1; i >= 0; i-- {
		sb.WriteString(fmt.Sprintf("%d:  %g\n", len(p.stack)-i, p.stack[i]))
	}
	p.TxtStack.SetText(sb.String())
	p.RefreshOutline()
}

// ****************************************************************************
// Outline panel integration
// ****************************************************************************

// RefreshOutline populates the shared Outline panel with pertinent indicators
// about the current calculator state (stack depth, top of stack, angular
// mode, history size).
func (p *RPNPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	top := "-"
	if len(p.stack) > 0 {
		top = fmt.Sprintf("%g", p.stack[len(p.stack)-1])
	}
	fields := [][2]string{
		{"Stack depth", fmt.Sprintf("%d", len(p.stack))},
		{"Top of stack", top},
		{"Angular mode", p.angMode},
		{"History entries", fmt.Sprintf("%d", len(p.history))},
	}
	for row, field := range fields {
		ui.TblOutline.SetCell(row, 0, tview.NewTableCell(field[0]).SetTextColor(tcell.ColorLightCyan))
		ui.TblOutline.SetCell(row, 1, tview.NewTableCell(field[1]).SetTextColor(tcell.ColorWhite))
	}
	ui.TblOutline.SetTitle("Outline")
}

func (p *RPNPlugin) showError(msg string) {
	ui.SetStatus("RPN error: " + msg)
}

// toRadians converts v to radians according to the current angular mode.
func (p *RPNPlugin) toRadians(v float64) float64 {
	if p.angMode == "DEG" {
		return v * math.Pi / 180
	}
	return v
}

// fromRadians converts v from radians back to the current angular mode.
func (p *RPNPlugin) fromRadians(v float64) float64 {
	if p.angMode == "DEG" {
		return v * 180 / math.Pi
	}
	return v
}

func (p *RPNPlugin) processToken(token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}

	if v, err := strconv.ParseFloat(token, 64); err == nil {
		p.push(v)
		return
	}

	switch token {
	case "+":
		b, e1 := p.pop()
		a, e2 := p.pop()
		if e1 == nil && e2 == nil {
			p.push(a + b)
		} else {
			p.showError("not enough operands for '+'")
		}
	case "-":
		b, e1 := p.pop()
		a, e2 := p.pop()
		if e1 == nil && e2 == nil {
			p.push(a - b)
		} else {
			p.showError("not enough operands for '-'")
		}
	case "*":
		b, e1 := p.pop()
		a, e2 := p.pop()
		if e1 == nil && e2 == nil {
			p.push(a * b)
		} else {
			p.showError("not enough operands for '*'")
		}
	case "/":
		b, e1 := p.pop()
		a, e2 := p.pop()
		if e1 == nil && e2 == nil {
			if b == 0 {
				p.showError("division by zero")
			} else {
				p.push(a / b)
			}
		} else {
			p.showError("not enough operands for '/'")
		}
	case "sqrt":
		a, e1 := p.pop()
		if e1 == nil {
			if a < 0 {
				p.showError("sqrt of negative number")
			} else {
				p.push(math.Sqrt(a))
			}
		} else {
			p.showError("not enough operands for 'sqrt'")
		}
	case "sin":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(math.Sin(p.toRadians(a)))
		} else {
			p.showError("not enough operands for 'sin'")
		}
	case "cos":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(math.Cos(p.toRadians(a)))
		} else {
			p.showError("not enough operands for 'cos'")
		}
	case "tan":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(math.Tan(p.toRadians(a)))
		} else {
			p.showError("not enough operands for 'tan'")
		}
	case "asin":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(p.fromRadians(math.Asin(p.toRadians(a))))
		} else {
			p.showError("not enough operands for 'asin'")
		}
	case "acos":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(p.fromRadians(math.Acos(p.toRadians(a))))
		} else {
			p.showError("not enough operands for 'acos'")
		}
	case "atan":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(p.fromRadians(math.Atan(p.toRadians(a))))
		} else {
			p.showError("not enough operands for 'atan'")
		}
	case "atan2":
		b, e1 := p.pop()
		a, e2 := p.pop()
		if e1 == nil && e2 == nil {
			p.push(p.fromRadians(math.Atan2(p.toRadians(a), p.toRadians(b))))
		} else {
			p.showError("not enough operands for 'atan2'")
		}
	case "inv":
		a, e1 := p.pop()
		if e1 == nil {
			if a == 0 {
				p.showError("division by zero")
			} else {
				p.push(1 / a)
			}
		} else {
			p.showError("not enough operands for 'inv'")
		}
	case "pow":
		b, e1 := p.pop()
		a, e2 := p.pop()
		if e1 == nil && e2 == nil {
			p.push(math.Pow(a, b))
		} else {
			p.showError("not enough operands for 'pow'")
		}
	case "log":
		b, e1 := p.pop()
		a, e2 := p.pop()
		if e1 == nil && e2 == nil {
			if a <= 0 || b <= 0 {
				p.showError("logarithm of non-positive number")
			} else {
				p.push(math.Log(b) / math.Log(a))
			}
		} else {
			p.showError("not enough operands for 'log'")
		}
	case "ln":
		a, e1 := p.pop()
		if e1 == nil {
			if a <= 0 {
				p.showError("logarithm of non-positive number")
			} else {
				p.push(math.Log(a))
			}
		} else {
			p.showError("not enough operands for 'ln'")
		}
	case "int":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(math.Trunc(a))
		} else {
			p.showError("not enough operands for 'int'")
		}
	case "frac":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(a - math.Trunc(a))
		} else {
			p.showError("not enough operands for 'frac'")
		}
	case "chs":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(-a)
		} else {
			p.showError("not enough operands for 'chs'")
		}
	case "abs":
		a, e1 := p.pop()
		if e1 == nil {
			p.push(math.Abs(a))
		} else {
			p.showError("not enough operands for 'abs'")
		}
	case "rand":
		p.push(rand.Float64())
	case "pi":
		p.push(math.Pi)
	case "e":
		p.push(math.E)
	case "phi":
		p.push((1 + math.Sqrt(5)) / 2)
	case "deg":
		p.angMode = "DEG"
		p.RefreshOutline()
		ui.SetStatus("Angular mode set to DEG")
	case "rad":
		p.angMode = "RAD"
		p.RefreshOutline()
		ui.SetStatus("Angular mode set to RAD")
	case "dup":
		if err := p.dup(); err != nil {
			p.showError(err.Error())
		}
	case "drop":
		if err := p.drop(); err != nil {
			p.showError(err.Error())
		}
	case "swap":
		if err := p.swap(); err != nil {
			p.showError(err.Error())
		}
	case "over":
		if err := p.over(); err != nil {
			p.showError(err.Error())
		}
	case "rot":
		if err := p.rot(); err != nil {
			p.showError(err.Error())
		}
	case "c", "clear", "cls":
		p.Clear()
	default:
		p.showError(fmt.Sprintf("unknown token: %s", token))
	}
}

// processInput splits a line into whitespace-separated tokens and evaluates
// them in order against the stack.
func (p *RPNPlugin) processInput(line string) {
	for _, tok := range strings.Fields(line) {
		p.processToken(tok)
	}
}
