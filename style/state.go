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
)

func (s ControlState) Hovered() bool  { return s&StateHovered != 0 }
func (s ControlState) Pressed() bool  { return s&StatePressed != 0 }
func (s ControlState) Disabled() bool { return s&StateDisabled != 0 }
func (s ControlState) Focused() bool  { return s&StateFocused != 0 }
func (s ControlState) Checked() bool  { return s&StateChecked != 0 }
func (s ControlState) Primary() bool  { return s&StatePrimary != 0 }

// LookAndFeel paints control chrome. Widgets never hard-code a skin.
type LookAndFeel interface {
	Name() string
	Palette() Palette
	Metrics() Metrics
	Font() *Font
	TitleFont() *Font
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
	DrawListRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string)
	DrawOverlay(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuBar(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuTitle(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool)
	DrawMenuFrame(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuItem(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label, shortcut string, underline int, sep, checked bool)
	DrawTabBar(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawTab(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool)
	DrawTreeRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool)
	DrawStatusBar(ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string)
	DrawToolBar(ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawToolButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon)
	DrawProgressBar(ctx *paintengine2d.Context, b paintengine2d.Rect, t float32, indeterminate bool, phase float32)
	DrawRadio(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string)
	DrawComboBox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool)
	DrawTitleBar(ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string)
	DrawMessageIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon)
	DrawTableHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool)
	DrawTableCell(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string, align Align, face *Font)
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
