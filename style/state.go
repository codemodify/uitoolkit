package style

import "github.com/codemodify/paintengine2d"

// ControlState is a bitset of interactive chrome states.
type ControlState uint64

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
	// StateFrameless marks a part that sits inside a frame its parent drew
	// (Qt's frameless QLineEdit in a QSpinBox): a field paints only its
	// text, a stepper only its buttons.
	StateFrameless
	// StateTreeChain marks a tree row that carries its chain bits (see
	// TreeChain); without it a painter cannot tell where branch lines end.
	StateTreeChain
)

// Named states above the tree chain bits (16 to 31).
const (
	// StateSidebar marks a view used as a sidebar (macOS source lists,
	// libadwaita's navigation sidebar, WinUI's NavigationView pane) and
	// its rows: looks with a sidebar style paint it; others ignore it.
	StateSidebar ControlState = 1 << (32 + iota)
	// StateAutoRaise marks a tool button that sits in a tool bar (Qt's
	// State_AutoRaise): flat until the pointer is over it, where a free
	// tool button keeps its bezel.
	StateAutoRaise
	// StateEditable marks a combo box whose text is typed into (QComboBox's
	// editable, GTK's combo box with an entry): Windows and GTK looks draw
	// it as a field with an arrow button. DrawComboBox gets no text for
	// it; the field inside draws its own.
	StateEditable
	// StateSelectedAbove and StateSelectedBelow mark a selected row of an
	// item view whose neighbour above (below) is selected too, so a look
	// that boxes its selection can join consecutive rows into one block with
	// square shared corners (SourceGit's sidebar lists, macOS's inset
	// lists).
	StateSelectedAbove
	StateSelectedBelow
)

// treeChainShift is where a tree row's chain bits start: above every
// state named before them.
const treeChainShift = 16

// TreeChain is the state of a tree row whose chain of nodes (its ancestors
// at depths 0…, then the row itself at its own depth) has, for each depth d
// with bit d set, a sibling after the node at that depth: which branch
// lines run on past the row (Qt's State_Sibling per branch). Depths from 16
// on count as having one.
func TreeChain(bits uint16) ControlState {
	return StateTreeChain | ControlState(bits)<<treeChainShift
}

// HasNextSibling reports whether the node at depth d on a tree row's chain
// has a sibling after it. Without chain bits (see StateTreeChain) every
// depth reports true, so painters keep their lines running.
func (s ControlState) HasNextSibling(d int) bool {
	if d < 0 {
		return false
	}
	if s&StateTreeChain == 0 || d >= 16 {
		return true
	}
	return s&(1<<(treeChainShift+d)) != 0
}

func (s ControlState) Sidebar() bool     { return s&StateSidebar != 0 }
func (s ControlState) AutoRaise() bool   { return s&StateAutoRaise != 0 }
func (s ControlState) Editable() bool    { return s&StateEditable != 0 }
func (s ControlState) Hovered() bool     { return s&StateHovered != 0 }
func (s ControlState) Pressed() bool     { return s&StatePressed != 0 }
func (s ControlState) Disabled() bool    { return s&StateDisabled != 0 }
func (s ControlState) Focused() bool     { return s&StateFocused != 0 }
func (s ControlState) Checked() bool     { return s&StateChecked != 0 }
func (s ControlState) Primary() bool     { return s&StatePrimary != 0 }
func (s ControlState) Toggle() bool      { return s&StateToggle != 0 }
func (s ControlState) First() bool       { return s&StateFirst != 0 }
func (s ControlState) Last() bool        { return s&StateLast != 0 }
func (s ControlState) Inactive() bool    { return s&StateInactive != 0 }
func (s ControlState) Backdrop() bool    { return s&StateBackdrop != 0 }
func (s ControlState) Alternate() bool   { return s&StateAlternate != 0 }
func (s ControlState) ExpanderHot() bool { return s&StateExpanderHot != 0 }
func (s ControlState) Frameless() bool   { return s&StateFrameless != 0 }

// SelectedAbove reports that the row above this selected row is selected.
func (s ControlState) SelectedAbove() bool { return s&StateSelectedAbove != 0 }

// SelectedBelow reports that the row below this selected row is selected.
func (s ControlState) SelectedBelow() bool { return s&StateSelectedBelow != 0 }

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
