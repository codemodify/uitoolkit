package style

import "github.com/codemodify/paintengine2d"

// ControlState is a bitset of interactive chrome states.
type ControlState uint32

const (
	StateNone    ControlState = 0
	StateHovered ControlState = 1 << iota
	StatePressed
	StateDisabled
	StateFocused
	StateChecked
	StatePrimary
	StateToggle
	// StateFirst / StateLast mark the first and last item of a strip
	// (tabs, segmented controls, the cells of a table row), for engines
	// that shape the ends. A table cell with neither is a middle cell.
	StateFirst
	StateLast
	// StateInactive marks an item whose view does not have keyboard focus
	// (always so in an inactive window). Windows and Mac looks grey such a
	// selection; GTK / Qt-style looks keep it until the window is inactive.
	StateInactive
	// StateBackdrop marks chrome in an inactive window (GTK's :backdrop,
	// Qt's inactive palette group): every look may subdue it.
	StateBackdrop
	// StateAlternate marks the odd rows of an item view, for looks with
	// striped lists (Mac OS X, Nimbus, Qt's alternatingRowColors).
	StateAlternate
	// StateExpanderHot marks a tree row whose expander is under the pointer
	// (Vista's bright triangle, GTK's prelit arrow).
	StateExpanderHot
)

func (s ControlState) Hovered() bool   { return s&StateHovered != 0 }
func (s ControlState) Pressed() bool   { return s&StatePressed != 0 }
func (s ControlState) Disabled() bool  { return s&StateDisabled != 0 }
func (s ControlState) Focused() bool   { return s&StateFocused != 0 }
func (s ControlState) Checked() bool   { return s&StateChecked != 0 }
func (s ControlState) Primary() bool   { return s&StatePrimary != 0 }
func (s ControlState) Toggle() bool    { return s&StateToggle != 0 }
func (s ControlState) First() bool     { return s&StateFirst != 0 }
func (s ControlState) Last() bool      { return s&StateLast != 0 }
func (s ControlState) Inactive() bool  { return s&StateInactive != 0 }
func (s ControlState) Backdrop() bool  { return s&StateBackdrop != 0 }
func (s ControlState) Alternate() bool { return s&StateAlternate != 0 }
func (s ControlState) ExpanderHot() bool { return s&StateExpanderHot != 0 }

// CellSpan is the box a table row's selection spans, seen from one cell:
// b grown by reach past each side where the row goes on (a cell that is
// not StateFirst / StateLast). An engine paints its whole rounded row box
// over the span, clipped to the cell, and the cells of a row join into one
// box (Explorer's details view, a Fluent list item).
func CellSpan(b paintengine2d.Rect, st ControlState, reach float32) paintengine2d.Rect {
	if !st.First() {
		b.Min.X -= reach
	}
	if !st.Last() {
		b.Max.X += reach
	}
	return b
}

// RowState is the item state of a plain row: selected and hovered.
func RowState(selected, hovered bool) ControlState {
	st := StateNone
	if selected {
		st |= StateChecked
	}
	if hovered {
		st |= StateHovered
	}
	return st
}

// LookAndFeel paints control chrome. Widgets never hard-code a skin.
type LookAndFeel interface {
	Name() string
	Palette() Palette
	Metrics() Metrics
	Font() *Font
	TitleFont() *Font
	BoldFont() *Font
	MutedFont() *Font
	OnAccentFont() *Font
	MonoFont() *Font

	DrawPanel(ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool)
	DrawButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string)
	DrawLabel(ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align)
	DrawCheckbox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string)
	DrawSlider(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32)
	DrawTextField(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font)
	DrawScrollBar(ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState)
	DrawFocusRing(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawSplitter(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState)
	// Row painters take the item state: Checked (selected), Hovered,
	// Focused (the current row of a focused view), Inactive, Disabled. The
	// current row's focus mark is DrawItemFocus, painted over the row.
	DrawListRow(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string)
	DrawOverlay(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuBar(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuTitle(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool)
	DrawMenuFrame(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuItem(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow)
	DrawTabBar(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawTab(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool)
	DrawTreeRow(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool)
	DrawStatusBar(ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string)
	DrawToolBar(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawToolButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon)
	DrawProgressBar(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32)
	DrawRadio(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string)
	DrawComboBox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool)
	DrawTitleBar(ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string)
	DrawMessageIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon)
	DrawTableHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool)
	DrawTableCell(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font)
	DrawSpinner(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool)
	DrawTooltip(ctx *paintengine2d.Context, b paintengine2d.Rect, text string)
	DrawTextArea(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font)
	DrawSwitch(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string)
	DrawAccordionHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool)
	DrawSeparator(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool)
}

// TextLine is one visual line of a TextArea (hard break or wrap).
type TextLine struct {
	Text  string
	Start int // inclusive rune index into the source
	End   int // exclusive rune index into the source
}

// Theme is a LookAndFeel plus a display scale (DPI).
type Theme struct {
	Look  LookAndFeel
	Scale float32
}

func (t Theme) S(v float32) float32 {
	if t.Scale <= 0 {
		return v
	}
	return v * t.Scale
}
