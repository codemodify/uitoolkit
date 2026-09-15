package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// BaseEngine is the stock painter: the five legacy bevel languages
// (classic-3d, luna-hottrack, soft-shadow, fluent-accent, none) driven by
// ThemeTokens. Every other engine embeds it and overrides what differs.
//
// Its methods forward to the look's base* implementations, which dispatch
// every part back through l.Engine() — see the dispatch rule on [Engine].
type BaseEngine struct{}

var baseEngine Engine = BaseEngine{}

func init() { RegisterEngine(baseEngine) }

func (BaseEngine) ID() string                    { return "base" }
func (BaseEngine) DefaultMetrics() ChromeMetrics { return ChromeMetrics{} }

// ---- parts ----------------------------------------------------------------

func (BaseEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	return l.basePaintFace(ctx, b, role, st)
}

func (BaseEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	l.baseCheckIndicator(ctx, box, st, checked)
}

func (BaseEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	l.baseRadioIndicator(ctx, box, st, selected)
}

func (BaseEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	FillArrow(ctx, b, dir, col)
}

func (BaseEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	l.eng().Arrow(l, ctx, b, dir, col)
}

func (BaseEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	l.baseMenuHighlight(ctx, b, attachBottom)
}

func (BaseEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	if hot && l.menuInvertText() {
		return l.palette.TextOnAccent
	}
	return l.palette.Text
}

func (BaseEngine) FieldFocusRing(l *Classic) bool {
	b := l.tokensOr().Bevel
	return b != BevelLunaHottrack && b != BevelFluentAccent
}

// ---- scrollbars -------------------------------------------------------------

func (BaseEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Overlay: true, Inset: 2}
}

func (BaseEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	if !p.Dec.Empty() {
		l.baseScrollArrow(ctx, p.Dec, vertical, false, st.Part(ScrollDec))
	}
	if !p.Inc.Empty() {
		l.baseScrollArrow(ctx, p.Inc, vertical, true, st.Part(ScrollInc))
	}
	if p.Thumb.Empty() {
		return
	}
	thumb := StateNone
	if st.Hovered {
		thumb |= StateHovered
	}
	if st.Pressed == ScrollThumbPart {
		thumb |= StatePressed
	}
	l.DrawScrollBar(ctx, p.Track, p.Thumb, thumb)
}

// baseScrollArrow is a raised step button with an arrow glyph.
func (l *Classic) baseScrollArrow(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical, inc bool, st ControlState) {
	fg := l.eng().Face(l, ctx, b, RoleButton, st)
	dir := DirUp
	switch {
	case vertical && inc:
		dir = DirDown
	case !vertical && !inc:
		dir = DirLeft
	case !vertical && inc:
		dir = DirRight
	}
	g := b.Inset(b.Dx() * 0.28)
	if st.Pressed() && l.tokensOr().Bevel == BevelClassic3D {
		g = g.Translate(paintengine2d.Pt(1, 1))
	}
	l.eng().Arrow(l, ctx, g, dir, fg)
}

// ---- frames -----------------------------------------------------------------

func (BaseEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	m := l.metrics
	in := Insets{Top: m.Pad, Right: m.Pad, Bottom: m.Pad, Left: m.Pad}
	if hasTitle {
		in.Top += m.TitleBar
	}
	return in
}

func (BaseEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	l.DrawPanel(ctx, b, raised)
	if title != "" {
		bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), l.metrics.TitleBar)
		l.DrawTitleBar(ctx, bar, title, "")
	}
}

func (BaseEngine) WindowFrameInsets(l *Classic) Insets {
	return Insets{Top: l.metrics.TitleBar, Right: 1, Bottom: 1, Left: 1}
}

func (BaseEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	p := l.palette
	r := l.metrics.RadiusSmall
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(p.Surface))
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), l.metrics.TitleBar)
	l.DrawTitleBar(ctx, bar, title, "")
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(p.Border, 1))
	if st.CanClose {
		cb := CaptionCloseRect(l, bar)
		fg := p.TextMuted
		if st.CloseHot || st.ClosePress {
			ctx.DrawRoundRect(cb, r, r, paintengine2d.Fill(p.Danger.WithAlpha(0.85)))
			fg = p.TextOnAccent
		}
		DrawCross(ctx, cb.Inset(cb.Dx()*0.3), fg, l.S(1.4))
	}
}

// ViewFrameInsets: the ViewFrame metric on every side, in whole pixels so
// rows never overlap the frame's lines (the stock looks keep 0: flat).
func (BaseEngine) ViewFrameInsets(l *Classic) Insets {
	v := float32(math.Ceil(float64(l.metrics.ViewFrame)))
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame: a view is framed like a text field (the look's field
// face) unless the look has no frame.
func (BaseEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	if l.eng().ViewFrameInsets(l).Zero() {
		return
	}
	l.eng().Face(l, ctx, b, RoleField, st&^(StateHovered|StatePressed))
}

// PopupShadow: the stock looks float menus, tooltips and dialogs on a soft
// shadow. A pack's "shadow" param scales its strength; 0 turns it off.
func (BaseEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	sp, ok := baseShadow(l, kind)
	if !ok {
		return Insets{}
	}
	return ShadowReach(0, sp.dy, sp.blur, 0)
}

func (BaseEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if sp, ok := baseShadow(l, kind); ok {
		DropShadow(ctx, b, sp.r, sp.col, 0, sp.dy, sp.blur, 0)
	}
}

type baseShadowSpec struct {
	col         paintengine2d.Color
	r, dy, blur float32
}

func baseShadow(l *Classic, kind PopupKind) (baseShadowSpec, bool) {
	k := l.P("shadow", 1)
	if k <= 0 {
		return baseShadowSpec{}, false
	}
	dark := Luma(l.palette.Background) < 0.5
	m := l.metrics
	var sp baseShadowSpec
	var a float32
	switch kind {
	case PopupTooltip:
		sp = baseShadowSpec{r: m.RadiusSmall, dy: l.S(1.5), blur: l.S(6)}
		a = 0.16
		if dark {
			a = 0.4
		}
	case PopupDialog:
		sp = baseShadowSpec{r: m.Radius, dy: l.S(8), blur: l.S(28)}
		a = 0.3
		if dark {
			a = 0.55
		}
	default:
		sp = baseShadowSpec{r: m.RadiusSmall, dy: l.S(4), blur: l.S(14)}
		a = 0.22
		if dark {
			a = 0.5
		}
	}
	sp.col = paintengine2d.RGBA(0, 0, 0, min(a*k, 1))
	return sp, true
}

func (BaseEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return CaptionCloseRect(l, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), l.metrics.TitleBar))
}

func (BaseEngine) StyleHint(l *Classic, h StyleHint) int { return 0 }

func (BaseEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(l.palette.Background))
}

func (BaseEngine) TabOutset(l *Classic) Insets { return Insets{} }

func (BaseEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.DrawPanel(ctx, b, false)
}

// CaptionCloseRect is where an in-app window's close button sits inside
// its caption bar. Engines with a different layout (Mac: left) override
// DrawWindowFrame and hit-testing uses [WindowCloseRect].
func CaptionCloseRect(l *Classic, bar paintengine2d.Rect) paintengine2d.Rect {
	side := bar.Dy() - l.S(8)
	if side < l.S(12) {
		side = bar.Dy() * 0.7
	}
	return paintengine2d.XYWH(bar.Max.X-side-l.S(5), bar.Min.Y+(bar.Dy()-side)*0.5, side, side)
}

// ---- whole controls: forward to the base implementations -------------------

func (BaseEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	l.baseDrawPanel(ctx, b, raised)
}
func (BaseEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	l.baseDrawButton(ctx, b, st, label)
}
func (BaseEngine) DrawLabel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align) {
	l.baseDrawLabel(ctx, b, text, col, align)
}
func (BaseEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	l.baseDrawCheckbox(ctx, b, st, checked, label)
}
func (BaseEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	l.baseDrawSlider(ctx, b, st, t)
}
func (BaseEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
}
func (BaseEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	l.baseDrawScrollBar(ctx, track, thumb, st)
}
func (BaseEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.baseDrawFocusRing(ctx, b)
}
func (BaseEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	l.baseDrawSplitter(ctx, b, vertical, st)
}
func (BaseEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string) {
	l.baseDrawListRow(ctx, b, selected, hovered, label)
}
func (BaseEngine) DrawOverlay(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.baseDrawOverlay(ctx, b)
}
func (BaseEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.baseDrawMenuBar(ctx, b)
}
func (BaseEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	l.baseDrawMenuTitle(ctx, b, st, label, underline, open)
}
func (BaseEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.baseDrawMenuFrame(ctx, b)
}
func (BaseEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	l.baseDrawMenuItem(ctx, b, st, row)
}
func (BaseEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.baseDrawTabBar(ctx, b)
}
func (BaseEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	l.baseDrawTab(ctx, b, st, label, selected)
}
func (BaseEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool) {
	l.baseDrawTreeRow(ctx, b, selected, hovered, expanded, leaf, depth, label, bold)
}
func (BaseEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	l.baseDrawStatusBar(ctx, b, parts)
}
func (BaseEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.baseDrawToolBar(ctx, b)
}
func (BaseEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	l.baseDrawToolButton(ctx, b, st, label, icon)
}
func (BaseEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	l.baseDrawProgressBar(ctx, b, st, t, indeterminate, phase)
}
func (BaseEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	l.baseDrawRadio(ctx, b, st, selected, label)
}
func (BaseEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	l.baseDrawComboBox(ctx, b, st, text, open)
}
func (BaseEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	l.baseDrawTitleBar(ctx, b, title, subtitle)
}
func (BaseEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	l.baseDrawMessageIcon(ctx, b, icon)
}
func (BaseEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	l.baseDrawTableHeader(ctx, b, st, label, sorted, asc)
}
func (BaseEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string, align Align, face *Font) {
	l.baseDrawTableCell(ctx, b, selected, hovered, label, align, face)
}
func (BaseEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	l.baseDrawSpinner(ctx, b, st, upHover, downHover, upPress, downPress)
}
func (BaseEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	l.baseDrawTooltip(ctx, b, text)
}
func (BaseEngine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	l.baseDrawTextArea(ctx, b, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face)
}
func (BaseEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	l.baseDrawSwitch(ctx, b, st, on, label)
}
func (BaseEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	l.baseDrawAccordionHeader(ctx, b, st, title, expanded)
}
func (BaseEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	l.baseDrawSeparator(ctx, b, vertical)
}
