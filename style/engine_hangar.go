//go:build theme_engine_all || theme_engine_hangar

package style

import (
	"strings"

	"github.com/codemodify/paintengine2d"
)

// hangarEngine paints a telemetry console: the flight-deck interface of a
// mission display, drawn the way those are actually drawn rather than the
// way a desktop is.
//
// It is not an era's desktop. Every other engine in here is a rebuild of
// something that shipped — Win95's bevels, Aqua's gel, Breeze's outlines —
// and this one is a *genre*: the instrument panel of a vehicle that does
// not exist, which games, film graphics and real avionics all converged on
// because the constraints are the same. Dark ground so the room can be
// dark; one hot colour and nothing else, so the eye goes where the colour
// is; hairlines instead of bevels, because a bevel says "press me" and an
// instrument is mostly read; capitals and a tick scale, because a label on
// an instrument is a *name* and not a sentence.
//
// What that means in rules:
//
//   - everything is square and one pixel thick. Nothing is raised, nothing
//     is sunken; a face is a fill and an outline, and a disabled face is
//     the outline alone;
//   - one accent, the ember, and it is spent on exactly two things: what
//     is *on* (a selected tab, a checked box, the default button) and what
//     is *wrong*. A hover is a wash of white, never of the accent, so the
//     accent always means something;
//   - chrome labels are capitals ([Classic.ControlLabel]), because that is
//     what the genre reads as — and the words themselves are untouched, so
//     a screen reader and a test see what the application wrote;
//   - a meter is segmented: a bar of blocks with a gap between them, which
//     is how a value reads at a glance from across a room;
//   - the window is a gradient from gunmetal to a warm shadow, with a
//     diagonal hatch over it, so a full-screen panel of flat boxes has
//     somewhere to sit.
//
// Pack data: params "caps" (1 = chrome labels in capitals, the default),
// "hatch" (1 = the diagonal weave on the window ground), "segments" (the
// width of a meter's block, 0 for a solid bar); extra "grid" is the
// hairline the ticks and rules are drawn in, "glow" the wash under a
// selected row.
type hangarEngine struct{ BaseEngine }

func init() {
	RegisterEngine(hangarEngine{})
	for _, p := range hangarPacks() {
		RegisterPack(p)
	}
}

func (hangarEngine) ID() string { return "hangar" }

// DefaultMetrics are instrument proportions: compact, square, hairline.
// A console shows more rows than a desktop does, so rows and bars are
// shorter than the stock ones and the scroll bar is a sliver.
func (hangarEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Square: true, ViewFrame: 1, Border: 1, FocusWidth: 1,
		ControlH: 28, FieldH: 26, ComboH: 26,
		Checkbox: 14, Radio: 14,
		MenuItemH: 24, MenuBarH: 26, TabH: 26, RowH: 22,
		TitleBar: 30, HeaderH: 24, ProgressH: 8, SliderH: 20, Thumb: 6,
		Scroll: 10, Pad: 10, FieldPad: 8,
		ToolBarH: 30, StatusBarH: 22, SpinnerW: 14, SwitchW: 36, SwitchH: 16,
	}
}

// ---- the palette this engine reads ------------------------------------------

// hangarColors is the look's own reading of its pack.
type hangarColors struct {
	ground, panel, band paintengine2d.Color
	line, grid          paintengine2d.Color
	text, dim, faint    paintengine2d.Color
	accent, hot, press  paintengine2d.Color
	onAccent, glow      paintengine2d.Color
	field               paintengine2d.Color
}

func hangarColorsOf(l *Classic) hangarColors {
	p := l.palette
	return hangarColors{
		ground:   p.Background,
		panel:    p.Surface,
		band:     p.SurfaceAlt,
		line:     p.Border,
		grid:     l.X("grid", p.Divider),
		text:     p.Text,
		dim:      p.TextMuted,
		faint:    l.X("faint", Mix(p.TextMuted, p.Background, 0.45)),
		accent:   p.Accent,
		hot:      p.AccentHover,
		press:    p.AccentPress,
		onAccent: p.TextOnAccent,
		glow:     l.X("glow", p.Accent.WithAlpha(0.18)),
		field:    p.Field,
	}
}

// hangarPx is the hairline: one device pixel at every scale, never less.
func hangarPx(l *Classic) float32 { return max(snap(l.S(1)), 1) }

// hangarSnap puts a box on the pixel grid, which every hairline in here
// needs: a 1px line on a half pixel is a 2px smear of half the colour.
func hangarSnap(b paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(b.Max.X)-x0, snap(b.Max.Y)-y0)
}

// hangarOver composites c over base, which is how a wash of white is
// turned back into an opaque fill.
func hangarOver(base, c paintengine2d.Color) paintengine2d.Color {
	return Mix(base, c, c.A)
}

// hangarCaps is a label in capitals.
func hangarCaps(s string) string { return strings.ToUpper(s) }

// hangarWash is the white a hover lifts a face by. The accent is never
// spent on a hover: a console where the hot colour means "the pointer is
// here" is one where it no longer means "this is on".
func hangarWash(st ControlState) float32 {
	switch {
	case st.Pressed():
		return 0.14
	case st.Hovered():
		return 0.07
	}
	return 0
}

// hangarLift is c with the state's wash of white over it.
func hangarLift(c paintengine2d.Color, st ControlState) paintengine2d.Color {
	if a := hangarWash(st); a > 0 {
		return hangarOver(c, paintengine2d.RGBA(1, 1, 1, a))
	}
	return c
}

// hangarFrame strokes a hairline rectangle just inside b, so the line is
// in the box rather than straddling its edge.
func hangarFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	lw := hangarPx(l)
	ctx.DrawRect(hangarSnap(b).Inset(lw*0.5), paintengine2d.StrokePaint(col, lw))
}

// hangarTick draws the short accent mark that starts a rule, a panel's top
// edge or a status strip: the one piece of punctuation this look repeats.
func hangarTick(l *Classic, ctx *paintengine2d.Context, at paintengine2d.Point, long float32, col paintengine2d.Color) {
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(at.X, at.Y, long, hangarPx(l)*2)), paintengine2d.Fill(col))
}

// ---- the window ground -------------------------------------------------------

// DrawWindowBackground is the gunmetal-to-shadow gradient and its weave.
//
// It depends on the window's box alone, which is the rule this hook is
// under: every sub-rect of a partial redraw paints the same pixels it
// would have in a whole one.
func (hangarEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	if b.Empty() {
		return
	}
	c := hangarColorsOf(l)
	warm := l.X("warm", Hex("#241a14"))
	ctx.DrawRect(b, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: b.Min, End: b.Max,
		Stops: []paintengine2d.GradientStop{
			{Offset: 0, Color: c.ground},
			{Offset: 0.62, Color: Mix(c.ground, warm, 0.35)},
			{Offset: 1, Color: warm},
		},
	}))
	if l.P("hatch", 1) <= 0 {
		return
	}
	// The weave: 45° hairlines, far enough apart to read as texture and
	// faint enough to disappear under anything placed on it.
	step := l.S(14)
	if step < 6 {
		step = 6
	}
	ink := paintengine2d.RGBA(1, 1, 1, 0.014)
	ctx.Save()
	ctx.ClipRect(b)
	p := paintengine2d.NewPath()
	for x := b.Min.X - b.Dy(); x < b.Max.X; x += step {
		p.MoveTo(x, b.Max.Y)
		p.LineTo(x+b.Dy(), b.Min.Y)
	}
	ctx.DrawPath(p, paintengine2d.StrokePaint(ink, hangarPx(l)))
	ctx.Restore()
}

// ---- faces -------------------------------------------------------------------

// Face is every control's ground: a fill and a hairline, no bevel at all.
// The stock token face would do most of this; it is here because the three
// rules that make the look — the white wash, the accent kept for what is
// on, and a disabled face that is its outline alone — are one decision and
// belong in one place.
func (e hangarEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := hangarColorsOf(l)
	fill, line, fg := c.panel, c.line, c.text

	switch role {
	case RoleButton:
		fill, line = c.band, c.line
		if st.Primary() && !st.Disabled() {
			fill, line, fg = c.accent, c.accent, c.onAccent
		}
	case RoleTool:
		fill, line = paintengine2d.Color{}, paintengine2d.Color{}
		if st.Checked() {
			fill, line, fg = c.accent, c.accent, c.onAccent
		}
	case RoleField, RoleCombo:
		fill, line = c.field, c.line
		if st.Focused() && !st.Disabled() {
			line = c.accent
		}
	case RoleCheck:
		fill, line = c.field, c.line
		if st.Checked() && !st.Disabled() {
			fill, line, fg = c.accent, c.accent, c.onAccent
		}
	case RoleRow:
		fill, line = paintengine2d.Color{}, paintengine2d.Color{}
		if st.Checked() {
			fill, line = c.glow, paintengine2d.Color{}
		}
	case RoleTab:
		fill, line = paintengine2d.Color{}, paintengine2d.Color{}
		if st.Checked() {
			fill, line, fg = c.accent, c.accent, c.onAccent
		}
	case RoleThumb:
		fill, line = c.faint, paintengine2d.Color{}
		if st.Hovered() || st.Pressed() {
			fill = c.accent
		}
	case RoleTrack:
		fill, line = Mix(c.panel, c.ground, 0.5), c.grid
	case RoleMenu:
		fill, line = paintengine2d.Color{}, paintengine2d.Color{}
		if st.Hovered() || st.Pressed() || st.Checked() {
			fill, line, fg = c.accent, c.accent, c.onAccent
		}
	case RoleBar, RolePanel:
		fill, line = c.panel, c.line
	case RoleSplitter:
		fill, line = paintengine2d.Color{}, paintengine2d.Color{}
	}

	if st.Disabled() {
		// The outline alone: an instrument that cannot be used is still
		// drawn, because its absence would move everything else.
		fill, fg = paintengine2d.Color{}, c.faint
		if !colorUnset(line) {
			line = Mix(c.line, c.ground, 0.5)
		}
	} else {
		fill = hangarLift(fill, st)
	}
	if ctx != nil && !b.Empty() {
		if !colorUnset(fill) && fill.A > 0.01 {
			ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(fill))
		}
		if !colorUnset(line) && line.A > 0.01 {
			hangarFrame(l, ctx, b, line)
		}
	}
	return fg
}

// ---- labels ------------------------------------------------------------------

// ControlLabel puts the chrome's own words in capitals. The application's
// string is untouched — this is the painting, and nothing else reads it —
// which is the same rule a skin's "case": "upper" follows.
func (hangarEngine) ControlLabel(l *Classic, role Role, s string) string {
	if s == "" || l.P("caps", 1) <= 0 {
		return s
	}
	switch role {
	case RoleButton, RoleTool, RoleTab, RoleCombo:
		return hangarCaps(s)
	}
	return s
}

// ---- panels, bars and rules ---------------------------------------------------

// DrawPanel is a hairline box with an ember tick in its top-left corner:
// the mark every panel in this genre wears, and the only ornament here.
func (e hangarEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := hangarColorsOf(l)
	fill := c.panel
	if raised {
		fill = c.band
	}
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(fill))
	hangarFrame(l, ctx, b, c.line)
	hangarTick(l, ctx, paintengine2d.Pt(b.Min.X, b.Min.Y), min(l.S(18), b.Dx()*0.25), c.accent)
}

func (e hangarEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := hangarColorsOf(l)
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(c.panel))
	hangarFrame(l, ctx, b, c.line)
}

// DrawToolBar and DrawMenuBar are a rule, not a slab: the ground shows
// through and a hairline says where the strip ends.
func (e hangarEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(c.panel.WithAlpha(0.6)))
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw)), paintengine2d.Fill(c.line))
	hangarTick(l, ctx, paintengine2d.Pt(b.Min.X, b.Max.Y-lw*2), l.S(22), c.accent)
}

func (e hangarEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	e.DrawToolBar(l, ctx, b)
}

func (e hangarEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw)), paintengine2d.Fill(c.line))
	if len(parts) == 0 {
		return
	}
	// Each part in its own slot, divided by a hairline, in capitals: a
	// status strip is a row of readings and reads as one.
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			ctx.DrawRect(hangarSnap(paintengine2d.XYWH(x, b.Min.Y+l.S(5), lw, b.Dy()-l.S(10))), paintengine2d.Fill(c.grid))
		}
		l.drawFittedText(ctx, l.muted, hangarCaps(s),
			paintengine2d.XYWH(x+l.S(10), b.Min.Y, slot-l.S(16), b.Dy()), c.dim, AlignStart, 0)
	}
}

// DrawSeparator is a hairline that begins with an ember tick.
func (e hangarEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	if vertical {
		x := snap(b.Min.X + (b.Dx()-lw)*0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(c.grid))
		ctx.DrawRect(paintengine2d.XYWH(x-lw*0.5, b.Min.Y, lw*2, min(l.S(12), b.Dy())), paintengine2d.Fill(c.accent))
		return
	}
	y := snap(b.Min.Y + (b.Dy()-lw)*0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), lw), paintengine2d.Fill(c.grid))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y-lw*0.5, min(l.S(12), b.Dx()), lw*2), paintengine2d.Fill(c.accent))
}

// DrawTitleBar is a panel's own caption: capitals at the left, a hairline
// under it, and the subtitle as a reading at the right.
func (e hangarEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(c.band))
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw)), paintengine2d.Fill(c.line))
	hangarTick(l, ctx, paintengine2d.Pt(b.Min.X, b.Min.Y), l.S(16), c.accent)
	pad := l.S(12)
	box := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()-lw)
	if subtitle != "" {
		w := l.muted.Advance(hangarCaps(subtitle)) + l.S(8)
		l.drawFittedText(ctx, l.muted, hangarCaps(subtitle),
			paintengine2d.XYWH(box.Max.X-w, box.Min.Y, w, box.Dy()), c.dim, AlignEnd, 0)
		box.Max.X -= w
	}
	l.drawFittedText(ctx, l.bold, hangarCaps(title), box, c.text, AlignStart, 0)
}

func (e hangarEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := hangarColorsOf(l)
	top := b.Min.Y
	if title != "" {
		top += l.body.Height() * 0.5
	}
	hangarFrame(l, ctx, paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top), c.line)
	if title == "" {
		return
	}
	caps := hangarCaps(title)
	w := l.muted.Advance(caps) + l.S(10)
	x := b.Min.X + l.S(10)
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(x-l.S(4), b.Min.Y, w, l.body.Height())), paintengine2d.Fill(c.panel))
	l.muted.Draw(ctx, caps, paintengine2d.Pt(x, b.Min.Y), c.accent)
}

// ---- values: meters, sliders, scroll bars -------------------------------------

// DrawProgressBar is a segmented meter: blocks with a gap between them,
// which is how a value is read at a glance. "segments" 0 makes it solid.
func (e hangarEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := hangarColorsOf(l)
	track := hangarSnap(b)
	ctx.DrawRect(track, paintengine2d.Fill(Mix(c.panel, c.ground, 0.5)))
	hangarFrame(l, ctx, track, c.grid)
	ink := c.accent
	if st.Disabled() {
		ink = c.faint
	}
	in := track.Inset(hangarPx(l) * 2)
	if in.Dx() <= 0 || in.Dy() <= 0 {
		return
	}
	run := in
	if indeterminate {
		// A block of a fifth, travelling: a console says "working" by
		// moving something, not by filling.
		w := in.Dx() * 0.2
		x := in.Min.X + (in.Dx()+w)*clamp01(phase) - w
		run = paintengine2d.XYWH(max(x, in.Min.X), in.Min.Y, min(w, in.Max.X-max(x, in.Min.X)), in.Dy())
	} else {
		run.Max.X = in.Min.X + in.Dx()*clamp01(t)
	}
	hangarBlocks(l, ctx, run, in, ink)
}

// hangarBlocks fills run with the look's blocks, clipped to in.
func hangarBlocks(l *Classic, ctx *paintengine2d.Context, run, in paintengine2d.Rect, ink paintengine2d.Color) {
	if run.Dx() <= 0 {
		return
	}
	seg := l.S(l.P("segments", 5))
	gap := hangarPx(l) * 2
	if seg <= 0 {
		ctx.DrawRect(hangarSnap(run), paintengine2d.Fill(ink))
		return
	}
	for x := run.Min.X; x < run.Max.X; x += seg + gap {
		w := min(seg, run.Max.X-x)
		if w <= 0 {
			break
		}
		ctx.DrawRect(hangarSnap(paintengine2d.XYWH(x, in.Min.Y, w, in.Dy())), paintengine2d.Fill(ink))
	}
}

// DrawSlider is a hairline groove with a square ember thumb: a setting is
// a thing you move, so it is the one control with a handle.
func (e hangarEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	cy := (b.Min.Y + b.Max.Y) * 0.5
	groove := hangarSnap(paintengine2d.XYWH(b.Min.X, cy-lw, b.Dx(), lw*2))
	ctx.DrawRect(groove, paintengine2d.Fill(c.grid))
	ink := c.accent
	if st.Disabled() {
		ink = c.faint
	}
	at := clamp01(t)
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(groove.Min.X, groove.Min.Y, groove.Dx()*at, groove.Dy())), paintengine2d.Fill(ink))
	// The thumb: a tall, narrow block, the way a cursor on a scale is
	// drawn on an instrument.
	w, h := l.S(6), l.S(16)
	x := b.Min.X + (b.Dx()-w)*at
	thumb := hangarSnap(paintengine2d.XYWH(x, cy-h*0.5, w, h))
	ctx.DrawRect(thumb, paintengine2d.Fill(hangarLift(ink, st)))
	if st.Focused() && !st.Disabled() {
		hangarFrame(l, ctx, thumb.Inset(-lw*2), c.hot)
	}
}

func (e hangarEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Overlay: true, Inset: hangarPx(l) * 2, Thickness: l.S(8)}
}

func (e hangarEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	c := hangarColorsOf(l)
	if !track.Empty() {
		ctx.DrawRect(hangarSnap(track), paintengine2d.Fill(Mix(c.panel, c.ground, 0.4)))
	}
	if thumb.Empty() {
		return
	}
	ink := c.faint
	if st.Hovered() || st.Pressed() {
		ink = c.accent
	}
	ctx.DrawRect(hangarSnap(thumb), paintengine2d.Fill(ink))
}

// ---- rows, tabs and headers ---------------------------------------------------

// DrawListRow washes a selected row in the ember and marks it with a bar
// at the leading edge: the row is still readable, which a solid fill of
// the accent would not be.
func (e hangarEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := hangarColorsOf(l)
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.faint
	case st.Checked():
		ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(c.glow))
		ctx.DrawRect(hangarSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y, hangarPx(l)*2, b.Dy())), paintengine2d.Fill(c.accent))
	case st.Hovered():
		ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.05)))
	}
	if label == "" {
		return
	}
	pad := l.S(10)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), fg, AlignStart, 0)
}

// DrawTab: the on tab is the ember, the rest are words on the ground.
func (e hangarEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := hangarColorsOf(l)
	fg := c.dim
	switch {
	case st.Disabled():
		fg = c.faint
	case selected:
		ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(c.accent))
		fg = c.onAccent
	case st.Hovered():
		ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.06)))
		fg = c.text
	}
	l.drawFittedText(ctx, l.body, hangarCaps(label), b, fg, AlignCenter, 8)
	if st.Focused() && !st.Disabled() {
		hangarFrame(l, ctx, b.Inset(hangarPx(l)*2), c.hot)
	}
}

func (e hangarEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw)), paintengine2d.Fill(c.line))
}

func (e hangarEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(c.band))
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw)), paintengine2d.Fill(c.line))
	ctx.DrawRect(hangarSnap(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+l.S(5), lw, b.Dy()-l.S(10))), paintengine2d.Fill(c.grid))
	fg := c.dim
	if sorted {
		fg = c.accent
	}
	pad := l.S(8)
	box := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2-l.S(10), b.Dy())
	l.drawFittedText(ctx, l.muted, hangarCaps(label), box, fg, AlignStart, 0)
	if !sorted {
		return
	}
	dir := DirDown
	if asc {
		dir = DirUp
	}
	side := l.S(7)
	e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-pad-side, (b.Min.Y+b.Max.Y-side)*0.5, side, side), dir, c.accent)
}

// ---- marks -------------------------------------------------------------------

// CheckIndicator is a square well with an ember fill when it is on; the
// tick is cut out of the fill rather than drawn over it.
func (e hangarEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := hangarColorsOf(l)
	b := hangarSnap(box)
	line := c.line
	switch {
	case st.Disabled():
		line = Mix(c.line, c.ground, 0.5)
	case checked:
		line = c.accent
	case st.Hovered() || st.Pressed():
		line = c.hot
	}
	ctx.DrawRect(b, paintengine2d.Fill(hangarLift(c.field, st)))
	hangarFrame(l, ctx, b, line)
	if !checked {
		return
	}
	in := b.Inset(hangarPx(l) * 3)
	ink := c.accent
	if st.Disabled() {
		ink = c.faint
	}
	ctx.DrawRect(in, paintengine2d.Fill(ink))
}

// RadioIndicator is the same well, round: a radio is round in every era
// and a console is no exception — it is the one curve in here.
func (e hangarEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	r := min(box.Dx(), box.Dy())*0.5 - lw*0.5
	at := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
	line := c.line
	switch {
	case st.Disabled():
		line = Mix(c.line, c.ground, 0.5)
	case selected:
		line = c.accent
	case st.Hovered() || st.Pressed():
		line = c.hot
	}
	ctx.DrawCircle(at, r, paintengine2d.Fill(hangarLift(c.field, st)))
	ctx.DrawCircle(at, r, paintengine2d.StrokePaint(line, lw))
	if !selected {
		return
	}
	ink := c.accent
	if st.Disabled() {
		ink = c.faint
	}
	ctx.DrawCircle(at, r*0.45, paintengine2d.Fill(ink))
}

// Arrow is a filled triangle: no stroked chevrons here, because a
// hairline chevron disappears at the sizes this look uses.
func (e hangarEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	if s < 3 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	r := s * 0.42
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx, cy-r)
		p.LineTo(cx+r, cy+r*0.7)
		p.LineTo(cx-r, cy+r*0.7)
	case DirLeft:
		p.MoveTo(cx-r, cy)
		p.LineTo(cx+r*0.7, cy-r)
		p.LineTo(cx+r*0.7, cy+r)
	case DirRight:
		p.MoveTo(cx+r, cy)
		p.LineTo(cx-r*0.7, cy-r)
		p.LineTo(cx-r*0.7, cy+r)
	default:
		p.MoveTo(cx, cy+r)
		p.LineTo(cx+r, cy-r*0.7)
		p.LineTo(cx-r, cy-r*0.7)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// DrawFocusRing is a hairline in the hot ember, just outside the control:
// no dots, no halo. The keyboard is somewhere, and that is all it says.
func (e hangarEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	hangarFrame(l, ctx, b, hangarColorsOf(l).hot)
}

func (e hangarEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	hangarFrame(l, ctx, b, hangarColorsOf(l).accent)
}

// ---- menus and tips ------------------------------------------------------------

func (e hangarEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := hangarColorsOf(l)
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(c.panel))
	hangarFrame(l, ctx, b, c.line)
}

func (e hangarEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(hangarColorsOf(l).accent))
}

func (e hangarEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := hangarColorsOf(l)
	if hot {
		return c.onAccent
	}
	return c.text
}

func (e hangarEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := hangarColorsOf(l)
	ctx.DrawRect(hangarSnap(b), paintengine2d.Fill(hangarOver(c.ground, c.band.WithAlpha(0.96))))
	hangarFrame(l, ctx, b, c.line)
	hangarTick(l, ctx, paintengine2d.Pt(b.Min.X, b.Min.Y), l.S(12), c.accent)
	ts := e.TooltipStyle(l)
	l.drawFittedText(ctx, ts.Face, text, b.Inset(ts.Pad), c.text, AlignStart, 0)
}

// StyleHint: nothing fades and nothing pulses. An instrument changes
// state the moment the state changes, or it is lying about the state.
func (e hangarEngine) StyleHint(l *Classic, h StyleHint) int {
	switch h {
	case HintHoverFadeMs, HintDefaultPulseMs:
		return 0
	}
	return e.BaseEngine.StyleHint(l, h)
}

// ---- the pack ------------------------------------------------------------------

func hangarPacks() []ThemePack {
	return []ThemePack{hangarPack()}
}

func hangarPack() ThemePack {
	ember := Hex("#ef4e23")
	ground := Hex("#0f141b")
	panel := Hex("#151c25")
	band := Hex("#1b2530")
	line := Hex("#2b3845")
	grid := Hex("#223040")
	text := Hex("#c7d2db")
	dim := Hex("#7c8b98")
	pal := Palette{
		Background: ground, Surface: panel, SurfaceAlt: band,
		Border: line, Divider: grid,
		Text: text, TextMuted: dim, TextOnAccent: Hex("#0f141b"),
		Accent: ember, AccentHover: Hex("#ff6a3d"), AccentPress: Hex("#c33a17"),
		Field: Hex("#0b1017"), FieldBorder: line,
		Focus: Hex("#ff6a3d"), Selection: ember,
		Track: grid, Thumb: Hex("#455767"),
		Highlight: ember.WithAlpha(0.18),
		Shadow:    paintengine2d.RGBA(0, 0, 0, 0.6),
		MenuHover: ember, MenuHoverBorder: ember, MenuGutter: Hex("#111822"),
		Danger: ember, Success: Hex("#4fb89a"), Warning: Hex("#e8a33d"),
		Overlay:    paintengine2d.RGBA(3.0/255, 6.0/255, 10.0/255, 0.62),
		BevelLight: line, BevelDark: Hex("#070b10"),
	}
	tok := ThemeTokens{
		Engine: "hangar", Bevel: BevelNone, Family: ThemeDark, Era: "Console",
		Palette: pal,
		Params:  map[string]float32{"caps": 1, "hatch": 1, "segments": 5},
		Extra: map[string]paintengine2d.Color{
			"grid":  grid,
			"glow":  ember.WithAlpha(0.18),
			"faint": Hex("#4a5866"),
			"warm":  Hex("#241a14"),
			// A selected row keeps its own ink: the wash is a tint, not a
			// fill, so white-on-ember would be wrong.
			"selectionText": text,
		},
		// The genre's typefaces: a narrow grotesque for the labels and a
		// real monospace for the readings, with what is actually installed
		// on a Linux box behind each.
		Fonts: FontPrefs{
			UI:   []string{"Titillium Web", "Archivo Narrow", "Roboto Condensed", "DejaVu Sans Condensed", "Noto Sans"},
			Mono: []string{"JetBrains Mono", "Hack", "DejaVu Sans Mono", "Liberation Mono"},
		},
	}
	tok.Hot = ChromeState{Fill: paintengine2d.RGBA(1, 1, 1, 0.07), Border: line}
	tok.Pressed = ChromeState{Fill: paintengine2d.RGBA(1, 1, 1, 0.14), Border: ember}
	tok.Selected = ChromeState{Fill: ember.WithAlpha(0.18), Border: ember}
	tok.Focus = ChromeState{Fill: paintengine2d.Color{}, Border: Hex("#ff6a3d")}
	tok.Disabled = ChromeState{Fill: paintengine2d.Color{}, Border: Mix(line, ground, 0.5)}
	return ThemePack{
		Name: "hangar", Label: "Hangar", Year: 2026, Lineage: "Console",
		Summary: "A telemetry console: gunmetal ground, one ember accent, hairline boxes, capitals and segmented meters.",
		Era:     "Console", Palette: ThemeDark, Tokens: tok,
	}
}
