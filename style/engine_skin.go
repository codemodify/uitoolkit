package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// skinEngine paints a control from a skin's sprites, and hands anything the
// skin does not describe to the engine of the skin's base pack.
//
// That second half is the whole safety story and it is worth stating
// plainly: a skin is always a *partial* override. Every method below asks
// the skin for art, and when there is none it calls the base pack's own
// engine — so a skin over "win95" gets Win95's bevels for the controls it
// never drew, and a skin over "breeze" gets Breeze's. A half-finished skin
// is a coherent app in a real look, not a grid of holes.
//
// The engine overrides *parts* rather than whole controls wherever it can.
// Face alone re-skins buttons, tool buttons, fields, combos, tabs, rows,
// thumbs, tracks, menus, splitters, bars and panels — thirteen roles — while
// the base look keeps doing the text, the layout, the mnemonics and the
// focus. That is the dispatch rule on [Engine] used for what it is for, and
// it is why a skin needs a dozen sprites rather than two hundred.
//
// What it will not do:
//
//   - It never removes the focus ring. A skin may re-draw it (the "focus"
//     part) and may change a control's face when it is focused (the "focus"
//     state), but DrawFocusRing always paints something, so no skin can ship
//     a keyboard trap.
//   - It never invents a control. The parts a skin may bind are a fixed
//     table (skinPartNames); anything else fails to load.
//   - It never runs anything. There is no expression in the format.
type skinEngine struct{ BaseEngine }

// skinEngineID is the engine registry id and the value of a skin pack's
// "engine" token. It is deliberately neutral: this is the toolkit's skin
// format, not an imitation of any one player's.
const skinEngineID = "skin"

func init() { RegisterEngine(skinEngine{}) }

func (skinEngine) ID() string { return skinEngineID }

// DefaultMetrics are none of its own: a skin's geometry is its base pack's,
// with the skin's own "metrics" on top. An engine that guessed here would
// fight the pack it is standing on.
func (skinEngine) DefaultMetrics() ChromeMetrics { return ChromeMetrics{} }

// under is the engine that paints what the skin leaves out.
//
// Forwarding to the base *pack's* engine is a departure from the original
// design sketch, which had the skin engine embed BaseEngine and fall back to
// the stock bevel painter. The engines were checked first: no engine method
// dispatches itself through l.Engine(), so a skin method that calls the base
// engine's same method descends through strictly different methods and
// always terminates.
func under(l *Classic) Engine {
	if sk := skinFor(l); sk != nil {
		return sk.baseEngine(l)
	}
	return baseEngine
}

// underFor is the engine that paints a whole control made of parts.
//
// This is the rule that makes a partial skin cohere, and it is worth stating
// carefully, because the obvious version is wrong.
//
// An era engine is free to paint a whole control directly instead of
// assembling it from parts — Breeze 6 draws its own menu row, its own button
// and its own check box, because that is how those controls differ from the
// stock ones. Forwarding such a control to the base engine therefore paints
// it in the base look *even when the skin has art for the part it is made
// of*, and the result is a skinned app with Breeze's blue menu highlight in
// it.
//
// So: when the skin binds any of the parts a control is made of, the control
// is painted by the *stock* engine instead, because BaseEngine re-dispatches
// every part and every nested control through l.Engine() — which is this
// engine — so the skin's art is used and the base's layout, text, mnemonics
// and focus handling are kept. When the skin binds none of them, the base
// pack's own engine paints the whole control, era look and all.
func underFor(l *Classic, parts ...string) Engine {
	if sk := skinFor(l); sk != nil {
		for _, p := range parts {
			if sk.has(p) {
				return baseEngine
			}
		}
	}
	return under(l)
}

// ---- resolving art --------------------------------------------------------

// skinRolePart maps an engine Role onto the part name a skin binds it with.
var skinRolePart = map[Role]string{
	RoleButton:   "button",
	RoleTool:     "tool",
	RoleField:    "field",
	RoleCheck:    "check",
	RoleRow:      "row",
	RoleTab:      "tab",
	RoleThumb:    "thumb",
	RoleTrack:    "track",
	RoleMenu:     "menu",
	RoleCombo:    "combo",
	RoleSplitter: "splitter",
	RoleBar:      "bar",
	RolePanel:    "panel",
}

// skinStateName is the state a control is in, as the manifest names it.
//
// The order is the order a person would read the control: disabled first
// because it outranks everything, then the on/off axis, then the pointer,
// then focus, then the default button, then a backdrop window.
func skinStateName(st ControlState) string {
	switch {
	case st.Disabled():
		return "disabled"
	case st.Checked() && st.Pressed():
		return "checkedPressed"
	case st.Checked() && st.Hovered():
		return "checkedHover"
	case st.Checked():
		return "checked"
	case st.Pressed():
		return "pressed"
	case st.Hovered():
		return "hover"
	case st.Primary():
		return "default"
	case st.Focused():
		return "focus"
	case st.Inactive() || st.Backdrop():
		return "inactive"
	}
	return "normal"
}

// art is the sprite for a state, walking skinStateFallback until something
// is bound. Nil means the part has nothing to say and the base engine paints.
func (p *SkinPart) art(state string) *SkinSprite {
	if p == nil {
		return nil
	}
	if sp := p.States[state]; sp != nil {
		return sp
	}
	for _, alt := range skinStateFallback[state] {
		if sp := p.States[alt]; sp != nil {
			return sp
		}
	}
	return nil
}

// part is the named binding, or nil.
func (sk *Skin) part(name string) *SkinPart {
	if sk == nil {
		return nil
	}
	return sk.Parts[name]
}

// draw paints the named part's art for st into b, returning false when the
// skin has nothing for it so the caller can fall through.
func (sk *Skin) draw(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, name string, st ControlState) bool {
	p := sk.part(name)
	sp := p.art(skinStateName(st))
	if sp == nil {
		return false
	}
	return sk.skinDraw(ctx, sk.box(l, b, p), sp, l.Scale(), sk.textColor(l, p, st))
}

// drawState paints a part's art for a literal state name (for parts whose
// two forms are not a ControlState: an open expander, a checked tick).
func (sk *Skin) drawState(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, name, state string, tint paintengine2d.Color) bool {
	p := sk.part(name)
	sp := p.art(state)
	if sp == nil {
		return false
	}
	return sk.skinDraw(ctx, sk.box(l, b, p), sp, l.Scale(), tint)
}

// has reports whether the skin binds a part at all.
func (sk *Skin) has(name string) bool { return sk.part(name).art("normal") != nil }

// box applies a part's design-pixel padding at the look's scale.
func (sk *Skin) box(l *Classic, b paintengine2d.Rect, p *SkinPart) paintengine2d.Rect {
	if p == nil || p.Pad.Zero() {
		return b
	}
	s := l.Scale() / sk.Design.Scale
	return Insets{
		Top: p.Pad.Top * s, Right: p.Pad.Right * s,
		Bottom: p.Pad.Bottom * s, Left: p.Pad.Left * s,
	}.Apply(b)
}

// textColor is the label colour a part's text role gives for st, falling
// back through the role's own resting colour to the base look's.
func (sk *Skin) textColor(l *Classic, p *SkinPart, st ControlState) paintengine2d.Color {
	var t *SkinText
	if p != nil {
		t = p.Text
	}
	if t == nil {
		t = sk.Text["control"]
	}
	if t == nil {
		return paintengine2d.Color{}
	}
	pick := func(cs ...paintengine2d.Color) paintengine2d.Color {
		for _, c := range cs {
			if !colorUnset(c) {
				return c
			}
		}
		return paintengine2d.Color{}
	}
	switch {
	case st.Disabled():
		return pick(t.Disabled, t.Color)
	case st.Checked():
		return pick(t.Checked, t.Hover, t.Color)
	case st.Pressed():
		return pick(t.Pressed, t.Hover, t.Color)
	case st.Hovered():
		return pick(t.Hover, t.Color)
	}
	return t.Color
}

// ---- parts ----------------------------------------------------------------

func (skinEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	sk := skinFor(l)
	if sk != nil {
		if name, ok := skinRolePart[role]; ok {
			p := sk.part(name)
			if sp := p.art(skinStateName(st)); sp != nil {
				sk.skinDraw(ctx, sk.box(l, b, p), sp, l.Scale(), sk.textColor(l, p, st))
				if fg := sk.textColor(l, p, st); !colorUnset(fg) {
					return fg
				}
				// The art is the skin's but the ink is not stated: the base
				// engine's answer for this role is still the right one.
				return skinBaseForeground(l, role, st)
			}
		}
	}
	return under(l).Face(l, ctx, b, role, st)
}

// skinBaseForeground is the label colour the base look would have used,
// without painting its face over the skin's art.
func skinBaseForeground(l *Classic, role Role, st ControlState) paintengine2d.Color {
	_, _, fg := l.faceColors(role, st)
	return fg
}

func (skinEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	sk := skinFor(l)
	if sk == nil || !sk.checkLike(l, ctx, box, st, checked, "check.mark") {
		under(l).CheckIndicator(l, ctx, box, st, checked)
	}
}

func (skinEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	sk := skinFor(l)
	if sk == nil || !sk.checkLike(l, ctx, box, st, selected, "radio.mark") {
		under(l).RadioIndicator(l, ctx, box, st, selected)
	}
}

// checkLike paints a check box or radio: the "check" well for its state,
// then the mark over it when it is on. A skin that draws the on state into
// the well itself needs no mark part — the well's "checked" art is enough.
func (sk *Skin) checkLike(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, on bool, mark string) bool {
	well := st
	if on {
		well |= StateChecked
	}
	drew := sk.draw(l, ctx, box, "check", well)
	if !drew {
		return false
	}
	if on {
		state := "normal"
		if st.Disabled() {
			state = "disabled"
		}
		ink := sk.textColor(l, sk.part("check"), well)
		if colorUnset(ink) {
			ink = l.palette.TextOnAccent
		}
		sk.drawState(l, ctx, box, mark, state, ink)
	}
	return true
}

func (skinEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	sk := skinFor(l)
	if sk != nil && sk.drawState(l, ctx, b, skinArrowPart(dir), "normal", col) {
		return
	}
	under(l).Arrow(l, ctx, b, dir, col)
}

func skinArrowPart(dir Direction) string {
	switch dir {
	case DirUp:
		return "arrow.up"
	case DirLeft:
		return "arrow.left"
	case DirRight:
		return "arrow.right"
	}
	return "arrow.down"
}

func (skinEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	sk := skinFor(l)
	name := "expander.shut"
	if expanded {
		name = "expander.open"
	}
	if sk != nil && sk.drawState(l, ctx, b, name, "normal", col) {
		return
	}
	under(l).Expander(l, ctx, b, expanded, col)
}

func (skinEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	sk := skinFor(l)
	if sk != nil && sk.draw(l, ctx, b, "menu", StateHovered) {
		return
	}
	under(l).MenuHighlight(l, ctx, b, attachBottom)
}

func (skinEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	if sk := skinFor(l); sk != nil && sk.has("menu") {
		st := StateNone
		if hot {
			st = StateHovered
		}
		if c := sk.textColor(l, sk.part("menu"), st); !colorUnset(c) {
			return c
		}
	}
	return under(l).MenuTextColor(l, hot)
}

func (skinEngine) FieldFocusRing(l *Classic) bool { return under(l).FieldFocusRing(l) }

// ---- the focus ring -------------------------------------------------------

// DrawFocusRing paints the skin's ring sprite, or the base engine's ring.
//
// It always paints one or the other. A skin may say what focus looks like;
// it may not say that focus looks like nothing, because a control nobody can
// see the focus on is a control nobody can drive from a keyboard.
func (skinEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	sk := skinFor(l)
	if sk != nil && sk.drawState(l, ctx, b, "focus", "normal", l.palette.Focus) {
		return
	}
	under(l).DrawFocusRing(l, ctx, b)
}

// ---- whole controls the roles cannot express ------------------------------

func (skinEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	sk := skinFor(l)
	if sk == nil || !sk.has("slider.track") {
		under(l).DrawSlider(l, ctx, b, st, t)
		return
	}
	thumb := l.metrics.Thumb
	if thumb <= 0 {
		thumb = l.S(16)
	}
	trackH := l.metrics.SliderH
	if trackH <= 0 || trackH > b.Dy() {
		trackH = min(b.Dy(), l.S(6))
	}
	track := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-trackH)*0.5, b.Dx(), trackH)
	sk.draw(l, ctx, track, "slider.track", st)

	x0, x1 := b.Min.X+thumb*0.5, b.Max.X-thumb*0.5
	if x1 < x0 {
		x0, x1 = b.Min.X+b.Dx()*0.5, b.Min.X+b.Dx()*0.5
	}
	cx := x0 + (x1-x0)*clamp01(t)
	if sk.has("slider.fill") && cx > track.Min.X {
		sk.draw(l, ctx, paintengine2d.Rect{Min: track.Min, Max: paintengine2d.Pt(cx, track.Max.Y)}, "slider.fill", st)
	}
	knob := paintengine2d.XYWH(cx-thumb*0.5, b.Min.Y+(b.Dy()-thumb)*0.5, thumb, thumb)
	if !sk.draw(l, ctx, knob, "slider.thumb", st) {
		sk.draw(l, ctx, knob, "thumb", st)
	}
	if st.Focused() && !st.Disabled() {
		l.DrawFocusRing(ctx, b)
	}
}

func (skinEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	sk := skinFor(l)
	if sk == nil || !sk.has("progress.back") {
		under(l).DrawProgressBar(l, ctx, b, st, t, indeterminate, phase)
		return
	}
	sk.draw(l, ctx, b, "progress.back", st)
	if !sk.has("progress.fill") {
		return
	}
	fill := b
	if indeterminate {
		// One travelling block, the width the era's busy bars used: a third
		// of the track, wrapped, so the bar reads as working rather than as
		// stuck at a value it does not have.
		w := b.Dx() / 3
		x := b.Min.X + (b.Dx()+w)*clamp01(phase) - w
		fill = paintengine2d.Rect{
			Min: paintengine2d.Pt(max(b.Min.X, x), b.Min.Y),
			Max: paintengine2d.Pt(min(b.Max.X, x+w), b.Max.Y),
		}
	} else {
		fill.Max.X = b.Min.X + b.Dx()*clamp01(t)
	}
	if fill.Dx() > 0.5 {
		sk.draw(l, ctx, fill, "progress.fill", st)
	}
}

func (skinEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	sk := skinFor(l)
	if sk == nil || !sk.has("switch.track") {
		under(l).DrawSwitch(l, ctx, b, st, on, label)
		return
	}
	w, h := l.metrics.SwitchW, l.metrics.SwitchH
	if w <= 0 {
		w = l.S(36)
	}
	if h <= 0 || h > b.Dy() {
		h = min(b.Dy(), l.S(20))
	}
	track := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-h)*0.5, w, h)
	tst := st
	if on {
		tst |= StateChecked
	}
	sk.draw(l, ctx, track, "switch.track", tst)

	knob := h - l.S(4)
	kx := track.Min.X + l.S(2)
	if on {
		kx = track.Max.X - knob - l.S(2)
	}
	sk.draw(l, ctx, paintengine2d.XYWH(kx, track.Min.Y+(h-knob)*0.5, knob, knob), "switch.knob", tst)

	if label != "" {
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, max(0, b.Max.X-track.Max.X-l.S(8)), b.Dy())
		ink := sk.textColor(l, sk.part("switch.track"), tst)
		if colorUnset(ink) {
			ink = l.palette.Text
			if st.Disabled() {
				ink = l.palette.TextMuted
			}
		}
		l.drawFittedText(ctx, l.body, label, lb, ink, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		l.DrawFocusRing(ctx, track.Inset(-l.S(2)))
	}
}

// DrawTab paints every tab from art, not only the selected and hovered ones.
//
// The base look deliberately leaves an idle tab's face unpainted — a flat
// look's unselected tab *is* the strip behind it. A skin's tab strip is a
// picture, so every tab needs its cell drawn or the row reads as one
// selected tab floating on a bar.
func (skinEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	sk := skinFor(l)
	if sk == nil || !sk.has("tab") {
		under(l).DrawTab(l, ctx, b, st, label, selected)
		return
	}
	if selected {
		st |= StateChecked
	}
	p := sk.part("tab")
	sk.draw(l, ctx, b, "tab", st)

	font, ink := l.body, sk.textColor(l, p, st)
	if !selected {
		font = l.muted
	}
	if colorUnset(ink) {
		ink = l.palette.Text
		if !selected {
			ink = l.palette.TextMuted
		}
	}
	l.drawFittedText(ctx, font, label, b, ink, AlignCenter, 8)
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(l.S(2)))
	}
}

// DrawWindowBackground paints the skin's window art behind everything.
//
// It obeys the engine rule that a window background must depend only on
// window coordinates — the art is anchored to the window's own box, so any
// sub-rect of it repaints identically under a clip and partial redraw stays
// correct.
func (skinEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	sk := skinFor(l)
	if sk != nil && sk.draw(l, ctx, b, "window", StateNone) {
		return
	}
	under(l).DrawWindowBackground(l, ctx, b)
}

func (skinEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	sk := skinFor(l)
	if sk != nil && sk.draw(l, ctx, b, "menu.frame", StateNone) {
		return
	}
	under(l).DrawMenuFrame(l, ctx, b)
}

func (skinEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	sk := skinFor(l)
	if sk == nil || !sk.has("tooltip") {
		under(l).DrawTooltip(l, ctx, b, text)
		return
	}
	sk.draw(l, ctx, b, "tooltip", StateNone)
	ink := sk.textColor(l, sk.part("tooltip"), StateNone)
	if colorUnset(ink) {
		ink = l.palette.Text
	}
	pad := l.S(8)
	l.drawFittedText(ctx, l.body, text, b.Inset(pad), ink, AlignCenter, 0)
}

// ControlFont is the face a skin labels a role in: its text role's size and
// weight where it states one. Widgets measure labels with the face the
// engine names, so a skin whose buttons are bold gets buttons wide enough
// for bold text instead of clipped ones.
func (skinEngine) ControlFont(l *Classic, role Role) *Font {
	sk := skinFor(l)
	if sk != nil {
		if name, ok := skinRolePart[role]; ok {
			if t := sk.textFor(name); t != nil && (t.Size > 0 || t.Bold) {
				size := l.metrics.FontSize
				if t.Size > 0 {
					size = t.Size * l.Scale() / sk.Design.Scale
				}
				w := WeightRegular
				if t.Bold {
					w = WeightBold
				}
				col := t.Color
				if colorUnset(col) {
					col = l.palette.Text
				}
				return BakeFamily(l.uiFamily, w, size, col)
			}
		}
	}
	return under(l).ControlFont(l, role)
}

// textFor is a part's text role, or the skin's "control" default.
func (sk *Skin) textFor(part string) *SkinText {
	if p := sk.part(part); p != nil && p.Text != nil {
		return p.Text
	}
	return sk.Text["control"]
}

// ---- forwarded ------------------------------------------------------------
//
// Everything below is the base pack's engine, unchanged. They are written
// out rather than inherited from BaseEngine because inheriting would give
// the *stock* painter, not the pack the skin stands on — a skin over Windows
// 95 would lose Win95's bevels on every control it did not draw, which is
// exactly the fallback this format promises not to be.

// ScrollBarStyle gives a skinned bar a gutter to live in.
//
// A modern base pack hands back a transient overlay bar — a sliver that
// appears while the view scrolls and fades out. That is right for a look
// whose thumb is a rounded rectangle of one colour and wrong for a skin,
// whose thumb is a picture someone drew: an overlay bar shows a few pixels
// of it and a transient one hides it entirely. So a skin with thumb art gets
// the classic arrangement, in its own "scroll" metric, with step buttons
// when it asks for them ("scrollArrows": 1).
func (skinEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	sk := skinFor(l)
	if sk == nil || !sk.has("thumb") {
		return under(l).ScrollBarStyle(l)
	}
	s := ScrollBarStyle{MinThumb: 24}
	if l.P("scrollArrows", 0) != 0 {
		s.Arrows = ArrowsEnds
	}
	return s
}

func (skinEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	underFor(l, "thumb", "track", "button").DrawScrollBarParts(l, ctx, p, vertical, st)
}

func (skinEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	return under(l).GroupBoxInsets(l, hasTitle)
}

func (skinEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	underFor(l, "panel").DrawGroupBox(l, ctx, b, title, raised)
}

func (skinEngine) WindowFrameInsets(l *Classic) Insets { return under(l).WindowFrameInsets(l) }

func (skinEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	underFor(l, "caption", "panel").DrawWindowFrame(l, ctx, b, title, st)
}

func (skinEngine) TabOutset(l *Classic) Insets {
	if sk := skinFor(l); sk != nil && sk.has("tab") {
		// The skin's tab art is a cell of its own: it does not overlap its
		// neighbours the way a drawn tab's border does.
		return Insets{}
	}
	return under(l).TabOutset(l)
}

func (skinEngine) TabOverlap(l *Classic) float32 {
	if sk := skinFor(l); sk != nil && sk.has("tab") {
		return 0
	}
	return under(l).TabOverlap(l)
}

func (skinEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return under(l).SpinBoxStyle(l) }

func (skinEngine) ViewBackground(l *Classic, st ControlState) paintengine2d.Color {
	return under(l).ViewBackground(l, st)
}

func (skinEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	sk := skinFor(l)
	if sk != nil && sk.draw(l, ctx, b, "panel", StateNone) {
		return
	}
	under(l).DrawTabPane(l, ctx, b)
}

func (skinEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return under(l).WindowCloseRect(l, b)
}

func (skinEngine) ToolBarInsets(l *Classic) Insets { return under(l).ToolBarInsets(l) }

func (skinEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	under(l).ItemFocus(l, ctx, b, st)
}

func (skinEngine) ViewFrameInsets(l *Classic) Insets { return under(l).ViewFrameInsets(l) }

func (skinEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	under(l).DrawViewFrame(l, ctx, b, st)
}

func (skinEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	return under(l).PopupShadow(l, kind)
}

func (skinEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	under(l).DrawPopupShadow(l, ctx, b, kind)
}

func (skinEngine) StyleHint(l *Classic, h StyleHint) int { return under(l).StyleHint(l, h) }

func (skinEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	underFor(l, "panel").DrawPanel(l, ctx, b, raised)
}

func (skinEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	underFor(l, "button").DrawButton(l, ctx, b, st, label)
}

func (skinEngine) DrawLabel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align) {
	under(l).DrawLabel(l, ctx, b, text, col, align)
}

func (skinEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	underFor(l, "check").DrawCheckbox(l, ctx, b, st, checked, label)
}

func (skinEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	underFor(l, "field").DrawTextField(l, ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

func (skinEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	underFor(l, "thumb", "track").DrawScrollBar(l, ctx, track, thumb, st)
}

func (skinEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	underFor(l, "splitter").DrawSplitter(l, ctx, b, vertical, st)
}

func (skinEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	underFor(l, "row").DrawListRow(l, ctx, b, st, label)
}

func (skinEngine) DrawOverlay(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	under(l).DrawOverlay(l, ctx, b)
}

func (skinEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	underFor(l, "bar").DrawMenuBar(l, ctx, b)
}

func (skinEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	underFor(l, "menu", "bar").DrawMenuTitle(l, ctx, b, st, label, underline, open)
}

func (skinEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	underFor(l, "menu").DrawMenuItem(l, ctx, b, st, row)
}

func (skinEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	underFor(l, "bar").DrawTabBar(l, ctx, b)
}

func (skinEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	underFor(l, "row").DrawTreeRow(l, ctx, b, st, expanded, leaf, depth, label, bold)
}

func (skinEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	underFor(l, "bar").DrawStatusBar(l, ctx, b, parts)
}

func (skinEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	underFor(l, "bar").DrawToolBar(l, ctx, b)
}

func (skinEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	underFor(l, "tool").DrawToolButton(l, ctx, b, st, label, icon)
}

func (skinEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	underFor(l, "check").DrawRadio(l, ctx, b, st, selected, label)
}

func (skinEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	underFor(l, "combo").DrawComboBox(l, ctx, b, st, text, open)
}

func (skinEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	underFor(l, "bar").DrawTitleBar(l, ctx, b, title, subtitle)
}

func (skinEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	under(l).DrawMessageIcon(l, ctx, b, icon)
}

func (skinEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	underFor(l, "button", "bar").DrawTableHeader(l, ctx, b, st, label, sorted, asc)
}

func (skinEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	underFor(l, "row").DrawTableCell(l, ctx, b, st, label, align, face)
}

func (skinEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	underFor(l, "field", "button").DrawSpinner(l, ctx, b, st, upHover, downHover, upPress, downPress)
}

func (skinEngine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	underFor(l, "field").DrawTextArea(l, ctx, b, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face)
}

func (skinEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	underFor(l, "button", "bar").DrawAccordionHeader(l, ctx, b, st, title, expanded)
}

func (skinEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	under(l).DrawSeparator(l, ctx, b, vertical)
}

// clamp01 keeps a fraction inside [0, 1].
func clamp01(t float32) float32 {
	if t < 0 || math.IsNaN(float64(t)) {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}
