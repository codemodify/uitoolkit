package players

import (
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The pieces of interface all three players share.
//
// There are only four, and the line between what is here and what is in each
// app is worth stating: **anything a player of any era would have drawn the
// same way** is here — the transport marks, the analyser, the clock that
// drives them, the keyboard. Everything about *how a particular player
// looked* is in that player's own package, because that is the whole subject
// of the demo and sharing it would be sharing the thing being demonstrated.
//
// All of it is ordinary toolkit code. A glyph button is a component with a
// name, a keyboard route, an accessibility node and a focus ring; what the
// skin changes is the face painted under the glyph, and nothing else.

// ---- the marks a transport row is made of ------------------------------------

// Glyph is one of the marks. They are drawn from paths rather than taken
// from an icon set, because none of the toolkit's seven icon sets has a
// transport row in it and because these are the shapes a tape machine's
// keys had before any of this was software.
type Glyph uint8

// Every glyph the three players use.
const (
	GlyphNone Glyph = iota
	GlyphPlay
	GlyphPause
	GlyphStop
	GlyphPrev
	GlyphNext
	GlyphEject
	GlyphShuffle
	GlyphRepeat
	GlyphRepeatOne
	GlyphVolume
	GlyphMute
	GlyphList
	GlyphSliders
	GlyphShrink
	GlyphGrow
	GlyphSkin
)

// DrawGlyph paints g centred in b, in col. weight is the stroke width for
// the glyphs that are drawn as lines; the solid ones ignore it.
//
// Everything is worked out from one side length rather than from b's own
// width and height, so a glyph in a box that is not square is centred and
// round rather than stretched into an oval.
func DrawGlyph(ctx *paintengine2d.Context, b paintengine2d.Rect, g Glyph, col paintengine2d.Color, weight float32) {
	s := min(b.Dx(), b.Dy())
	if s <= 0 || col.A <= 0 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2
	// u is the glyph's unit: half of the side it is drawn inside, which is
	// a little smaller than half the box so the mark has air round it.
	u := s * 0.30
	if weight <= 0 {
		weight = max(s*0.1, 1)
	}
	fill := paintengine2d.Fill(col)
	stroke := paintengine2d.StrokePaint(col, weight)
	stroke.Stroke.Cap = paintengine2d.CapRound
	stroke.Stroke.Join = paintengine2d.JoinRound

	tri := func(x, y, h float32, left bool) {
		p := paintengine2d.NewPath()
		w := h * 0.86
		if left {
			p.MoveTo(x+w/2, y-h/2)
			p.LineTo(x-w/2, y)
			p.LineTo(x+w/2, y+h/2)
		} else {
			p.MoveTo(x-w/2, y-h/2)
			p.LineTo(x+w/2, y)
			p.LineTo(x-w/2, y+h/2)
		}
		p.Close()
		ctx.DrawPath(p, fill)
	}
	bar := func(x, y, w, h float32) {
		ctx.DrawRect(paintengine2d.XYWH(x-w/2, y-h/2, w, h), fill)
	}

	switch g {
	case GlyphPlay:
		tri(cx+u*0.12, cy, u*2, false)
	case GlyphPause:
		bar(cx-u*0.5, cy, u*0.52, u*1.9)
		bar(cx+u*0.5, cy, u*0.52, u*1.9)
	case GlyphStop:
		ctx.DrawRect(paintengine2d.XYWH(cx-u*0.85, cy-u*0.85, u*1.7, u*1.7), fill)
	case GlyphPrev:
		tri(cx+u*0.35, cy, u*1.7, true)
		bar(cx-u*0.82, cy, u*0.4, u*1.7)
	case GlyphNext:
		tri(cx-u*0.35, cy, u*1.7, false)
		bar(cx+u*0.82, cy, u*0.4, u*1.7)
	case GlyphEject:
		p := paintengine2d.NewPath()
		p.MoveTo(cx-u, cy+u*0.15)
		p.LineTo(cx, cy-u)
		p.LineTo(cx+u, cy+u*0.15)
		p.Close()
		ctx.DrawPath(p, fill)
		bar(cx, cy+u*0.75, u*2, u*0.4)
	case GlyphShuffle:
		// Two lines that cross, with an arrow head on each: the mark every
		// player settled on once shuffling was a button rather than a menu.
		p := paintengine2d.NewPath()
		p.MoveTo(cx-u, cy-u*0.6)
		p.LineTo(cx+u*0.35, cy+u*0.6)
		p.MoveTo(cx-u, cy+u*0.6)
		p.LineTo(cx+u*0.35, cy-u*0.6)
		ctx.DrawPath(p, stroke)
		tri(cx+u*0.72, cy-u*0.6, u*0.8, false)
		tri(cx+u*0.72, cy+u*0.6, u*0.8, false)
	case GlyphRepeat, GlyphRepeatOne:
		// A circle that does not quite close, with one arrow head at the
		// break. The rectangular two-headed loop every icon set draws is
		// four strokes and two triangles, and at nineteen pixels — which is
		// exactly what a compact player's toggles are — all six land on the
		// same three rows and read as a box with a dent in it.
		r := u * 0.86
		p := paintengine2d.NewPath()
		p.AddArc(paintengine2d.Pt(cx, cy), r, r, -1.15, 5.5)
		ctx.DrawPath(p, stroke)
		tri(cx+r*0.42, cy-r*0.92, u*0.95, false)
		if g == GlyphRepeatOne {
			ctx.DrawRect(paintengine2d.XYWH(cx-weight/2, cy-u*0.4, weight, u*0.8), fill)
		}
	case GlyphVolume, GlyphMute:
		p := paintengine2d.NewPath()
		p.MoveTo(cx-u*0.9, cy-u*0.32)
		p.LineTo(cx-u*0.42, cy-u*0.32)
		p.LineTo(cx+u*0.1, cy-u*0.95)
		p.LineTo(cx+u*0.1, cy+u*0.95)
		p.LineTo(cx-u*0.42, cy+u*0.32)
		p.LineTo(cx-u*0.9, cy+u*0.32)
		p.Close()
		ctx.DrawPath(p, fill)
		if g == GlyphMute {
			x := paintengine2d.NewPath()
			x.MoveTo(cx+u*0.42, cy-u*0.45)
			x.LineTo(cx+u*1.0, cy+u*0.45)
			x.MoveTo(cx+u*1.0, cy-u*0.45)
			x.LineTo(cx+u*0.42, cy+u*0.45)
			ctx.DrawPath(x, stroke)
			break
		}
		wave := paintengine2d.NewPath()
		wave.MoveTo(cx+u*0.45, cy-u*0.45)
		wave.QuadTo(cx+u*0.8, cy, cx+u*0.45, cy+u*0.45)
		ctx.DrawPath(wave, stroke)
	case GlyphList:
		for i := float32(-1); i <= 1; i++ {
			y := cy + i*u*0.7
			ctx.DrawRect(paintengine2d.XYWH(cx-u, y-weight/2, weight, weight), fill)
			ctx.DrawRect(paintengine2d.XYWH(cx-u*0.5, y-weight/2, u*1.5, weight), fill)
		}
	case GlyphSliders:
		for i := float32(-1); i <= 1; i++ {
			x := cx + i*u*0.72
			ctx.DrawRect(paintengine2d.XYWH(x-weight/2, cy-u, weight, u*2), fill)
			// The knob sits at a different height on each, so the mark
			// reads as a set of faders rather than as three lines.
			ky := cy + u*0.55*[]float32{-1, 0.35, -0.2}[int(i)+1]
			ctx.DrawRect(paintengine2d.XYWH(x-u*0.42, ky-weight, u*0.84, weight*2), fill)
		}
	case GlyphShrink, GlyphGrow:
		box := paintengine2d.XYWH(cx-u, cy-u*0.8, u*2, u*1.6)
		ctx.DrawRect(box, stroke)
		if g == GlyphShrink {
			ctx.DrawRect(paintengine2d.XYWH(cx-u*0.5, cy-weight/2, u, weight), fill)
			break
		}
		ctx.DrawRect(paintengine2d.XYWH(cx-u*0.5, cy-weight/2, u, weight), fill)
		ctx.DrawRect(paintengine2d.XYWH(cx-weight/2, cy-u*0.5, weight, u), fill)
	case GlyphSkin:
		// A square with one corner turned up: the mark for "the picture
		// over the controls", which is all a skin is.
		p := paintengine2d.NewPath()
		p.MoveTo(cx-u, cy-u)
		p.LineTo(cx+u*0.35, cy-u)
		p.LineTo(cx+u, cy-u*0.35)
		p.LineTo(cx+u, cy+u)
		p.LineTo(cx-u, cy+u)
		p.Close()
		ctx.DrawPath(p, stroke)
		q := paintengine2d.NewPath()
		q.MoveTo(cx+u*0.35, cy-u)
		q.LineTo(cx+u*0.35, cy-u*0.35)
		q.LineTo(cx+u, cy-u*0.35)
		ctx.DrawPath(q, stroke)
	}
}

// ---- a transport button ------------------------------------------------------

// GlyphButton is a transport control: one of the look's faces with a glyph
// painted on it.
//
// It is an ordinary component and that is the whole point of it. It takes
// the focus, Return and Space work it, it names itself in the accessibility
// tree, and ShapeRole hands the look the face it paints — so a skin whose
// art for that face is a disc decides where the button really is, and a
// press in the corner of its box falls through to whatever is behind.
type GlyphButton struct {
	widget.Base
	Glyph Glyph
	// Label is what the button is called: its accessible name and its
	// tooltip. A glyph has no text to fall back on, so this is not
	// optional and a11y.Check says so when it is missing.
	Label string
	// Role is the face it paints. RoleTool is the transport row's — a mark
	// on the shell that grows a face under the pointer; RoleButton is a
	// push button with a face of its own at rest.
	Role style.Role
	// Checked draws the button in its on state (shuffle, repeat, the
	// playlist toggle) and says so in the accessibility tree.
	Checked bool
	// Toggle makes it a toggle for a screen reader's benefit rather than a
	// button that happens to look pressed.
	Toggle  bool
	OnClick func()

	hovered, pressed, outside bool
}

// NewGlyphButton is a transport button on the tool face.
func NewGlyphButton(g Glyph, label string, on func()) *GlyphButton {
	b := &GlyphButton{Glyph: g, Label: label, Role: style.RoleTool, OnClick: on}
	b.Init(b)
	b.SetWantsFocus(true)
	b.SetFocusVisibleOnly(true)
	b.SetAccessibleName(label)
	return b
}

// Tooltip is the button's label, since its face has no words on it.
func (b *GlyphButton) Tooltip() string { return b.Label }

// ShapeRole is the face the button paints, so a skin that gives that face a
// silhouette decides where the pointer has to be.
func (b *GlyphButton) ShapeRole() style.Role { return b.Role }

// SetGlyph swaps the mark — play becomes pause — and repaints.
func (b *GlyphButton) SetGlyph(g Glyph, label string) {
	if b.Glyph == g && b.Label == label {
		return
	}
	b.Glyph, b.Label = g, label
	b.SetAccessibleName(label)
	b.Invalidate()
}

// SetChecked turns the button on or off.
func (b *GlyphButton) SetChecked(on bool) {
	if b.Checked == on {
		return
	}
	b.Checked = on
	b.Invalidate()
}

// PaintState is the state the face is painted in.
func (b *GlyphButton) PaintState() style.ControlState {
	st := b.State()
	if b.hovered && !b.outside {
		st |= style.StateHovered
	}
	if b.pressed && !b.outside {
		st |= style.StatePressed
	}
	if b.Checked {
		st |= style.StateChecked
	}
	return st
}

func (b *GlyphButton) Measure(c layout.Constraints) paintengine2d.Point {
	h := b.Look().Metrics().ControlH
	return c.Constrain(paintengine2d.Pt(h, h))
}

func (b *GlyphButton) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *GlyphButton) Paint(ctx *paintengine2d.Context) {
	lk, r := b.Look(), b.LocalBounds()
	st := b.PaintState()
	if b.Role == style.RoleButton {
		lk.DrawButton(ctx, r, st, "")
	} else {
		lk.DrawToolButton(ctx, r, st, "", style.IconNone)
	}
	DrawGlyph(ctx, r, b.Glyph, GlyphInk(lk, st), style.Dip(lk, 1.6))
}

// GlyphInk is the colour a mark is drawn in for a control state.
//
// It deliberately does not try to read the face underneath. A skin's faces
// are pictures and the engine does not publish the colour of one, so a
// glyph that guessed would be unreadable on whichever skin guessed wrong;
// what it does instead is stay the label colour and let the *accent* carry
// "this is switched on", which every skin here draws a dark face for.
func GlyphInk(lk style.LookAndFeel, st style.ControlState) paintengine2d.Color {
	pal := lk.Palette()
	switch {
	case st.Disabled():
		return pal.TextMuted
	case st.Checked():
		return pal.Accent
	case st.Backdrop():
		return pal.TextMuted
	}
	return pal.Text
}

func (b *GlyphButton) MouseEnter() { b.hovered = true; b.Base.MouseEnter() }

func (b *GlyphButton) MouseExit() {
	b.hovered, b.pressed = false, false
	b.Base.MouseExit()
}

func (b *GlyphButton) MousePress(widget.MouseEvent) bool {
	if !b.Enabled() {
		return false
	}
	b.MarkPointerFocus()
	b.RequestFocus()
	b.pressed = true
	b.Invalidate()
	return true
}

// MouseMove tracks a press dragged off the button and back on, exactly as
// widgets.Button does: it pops up, and releasing outside does not click.
func (b *GlyphButton) MouseMove(e widget.MouseEvent) bool {
	if !b.pressed {
		return false
	}
	if out := !b.LocalBounds().Contains(e.Pos); out != b.outside {
		b.outside = out
		b.Invalidate()
	}
	return true
}

func (b *GlyphButton) MouseRelease(e widget.MouseEvent) bool {
	was := b.pressed
	b.pressed, b.outside = false, false
	b.Invalidate()
	if was && b.LocalBounds().Contains(e.Pos) && b.Enabled() {
		b.fire()
	}
	return true
}

func (b *GlyphButton) KeyPress(e widget.KeyEvent) bool {
	if !b.Enabled() {
		return false
	}
	if e.Key == platform.KeyReturn || e.Key == platform.KeySpace {
		b.MarkKeyboardFocus()
		b.fire()
		return true
	}
	return false
}

func (b *GlyphButton) fire() {
	if b.OnClick != nil {
		b.OnClick()
	}
}

// Describe puts the button in the accessibility tree as what it is: a
// button, or a toggle when it is one, with the label its glyph does not
// have.
func (b *GlyphButton) Describe(n *a11y.Node) {
	n.Role = a11y.RoleButton
	if b.Toggle {
		n.Role = a11y.RoleToggleButton
	}
	if n.Name == "" {
		n.Name = b.Label
	}
	if b.Checked {
		n.State |= a11y.StateChecked | a11y.StatePressed
	}
	n.Actions = n.Actions.With(a11y.ActionDefault)
}

// AccessibleAction runs the button from the accessibility tree, which is
// what a screen reader's "press" does.
func (b *GlyphButton) AccessibleAction(a a11y.Action) bool {
	if a != a11y.ActionDefault || !b.Enabled() {
		return false
	}
	b.fire()
	return true
}

// ---- text a display needs and a look does not offer --------------------------

// ScaledFace is one of a look's faces at another size: the pack's own family
// and weight, baked bigger or smaller.
//
// A pack states one body size and one mono size, and the clock on a player's
// display is neither — it is the one number the thing is read by from across
// a room, and at the body size it is a caption. The family and the weight
// still come from the pack, so an era pack's display is set in that era's
// face; only the size is the app's.
func ScaledFace(f *style.Font, size float32) *style.Font {
	if f == nil || size <= 0 {
		return f
	}
	if size >= f.Size-0.01 && size <= f.Size+0.01 {
		return f
	}
	return style.BakeFamily(f.Family, f.Weight, size, f.Color)
}

// DrawTextIn paints one line in a box with a face of the caller's choosing,
// aligned across the box and centred down it.
//
// LookAndFeel.DrawLabel does this with the look's body face and no other, so
// anything set in the mono face or at a display size has to place its own
// baseline — and Font.Draw's origin is the *top of the em box*, not the
// baseline, which is the thing that is easy to get wrong and looks like a
// layout bug when it is.
func DrawTextIn(ctx *paintengine2d.Context, f *style.Font, b paintengine2d.Rect, text string, col paintengine2d.Color, align style.Align) {
	if f == nil || text == "" {
		return
	}
	w := f.Advance(text)
	if w > b.Dx() {
		text = f.Fit(text, b.Dx())
		w = f.Advance(text)
	}
	x := b.Min.X
	switch align {
	case style.AlignCenter:
		x += (b.Dx() - w) / 2
	case style.AlignEnd:
		x += b.Dx() - w
	}
	f.Draw(ctx, text, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())/2), col)
}

// DrawScrollingText paints one line that slides when it is wider than its
// box, and paints it plainly when it fits.
//
// It slides *back and forth with a rest at each end* rather than looping
// round. A loop is what a player of this era did and it is one line of
// arithmetic, but two thirds of every still frame of one is a title cut in
// half with the start of the next copy beside it — and a demo whose whole
// job is to be looked at should be readable in a still. The rest at each end
// also gives the eye the beginning of the title back once a cycle, which is
// the part anyone is actually trying to read.
//
// The phase comes from the transport's own position rather than from a wall
// clock, so the same moment is always the same picture: a screenshot is
// comparable with the one before it, and the scroll does not speed up when
// the window happens to repaint more often.
func DrawScrollingText(ctx *paintengine2d.Context, lk style.LookAndFeel, f *style.Font, b paintengine2d.Rect, text string, col paintengine2d.Color, pos time.Duration) {
	if f == nil || text == "" || b.Dx() <= 0 {
		return
	}
	w := f.Advance(text)
	if w <= b.Dx() {
		DrawTextIn(ctx, f, b, text, col, style.AlignStart)
		return
	}
	over := w - b.Dx()
	// Seconds: a rest, a slide at about twenty-six design pixels a second,
	// a rest, and the slide back.
	const rest = 2.2
	slide := float64(over) / float64(style.Dip(lk, 26))
	if slide < 0.4 {
		slide = 0.4
	}
	cycle := 2 * (rest + slide)
	t := pos.Seconds()
	t -= cycle * float64(int(t/cycle))
	var k float64 // 0 at the start of the line, 1 at its end
	switch {
	case t < rest:
		k = 0
	case t < rest+slide:
		k = (t - rest) / slide
	case t < 2*rest+slide:
		k = 1
	default:
		k = 1 - (t-2*rest-slide)/slide
	}
	y := b.Min.Y + (b.Dy()-f.Height())/2
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, text, paintengine2d.Pt(b.Min.X-over*float32(k), y), col)
	ctx.Restore()
}

// ---- a fader -----------------------------------------------------------------

// Fader is a slider stood on end.
//
// The toolkit's widgets.Slider is horizontal — it measures its height from
// the look's slider metric and draws along its width — and an equaliser is
// the one control that has to be vertical, because ten of them side by side
// *is* the control and a row of horizontal ones is a form. So this is a
// component of its own rather than a flag on that one, and it paints through
// the same three parts a slider does (slider.track, slider.fill,
// slider.thumb) by asking the look to draw a slider into a box it has
// rotated, which is what keeps a skinned fader skinned.
type Fader struct {
	widget.Base
	Min, Max, Value float32
	// Step is one press of an arrow key, and Page one of Page Up or Down.
	Step, Page float32
	// Label is the fader's accessible name: "62 Hz", "Preamp".
	Label    string
	OnChange func(float32)
	// Format turns the value into what a screen reader reads and what a
	// tooltip shows ("+4.5 dB"). Nil reads the bare number.
	Format func(float32) string

	hovered, drag bool
}

// NewFader is a fader over a range.
func NewFader(min, max, value float32, label string, on func(float32)) *Fader {
	if max <= min {
		max = min + 1
	}
	f := &Fader{Min: min, Max: max, Value: value, Label: label, OnChange: on}
	f.Step = (max - min) / 24
	f.Page = (max - min) / 4
	f.Init(f)
	f.SetWantsFocus(true)
	f.SetFocusVisibleOnly(true)
	f.SetAccessibleName(label)
	return f
}

// SetValue moves the fader, clamped, and tells its owner.
func (f *Fader) SetValue(v float32) {
	if v < f.Min {
		v = f.Min
	}
	if v > f.Max {
		v = f.Max
	}
	if v == f.Value {
		return
	}
	f.Value = v
	f.Invalidate()
	if f.OnChange != nil {
		f.OnChange(v)
	}
}

// Set moves the fader without telling its owner — for a preset that moves
// ten of them at once and reports the change itself.
func (f *Fader) Set(v float32) {
	on := f.OnChange
	f.OnChange = nil
	f.SetValue(v)
	f.OnChange = on
}

// Tooltip is the label and the reading.
func (f *Fader) Tooltip() string {
	if f.Format == nil {
		return f.Label
	}
	return f.Label + "  " + f.Format(f.Value)
}

func (f *Fader) t() float32 {
	if f.Max <= f.Min {
		return 0
	}
	return (f.Value - f.Min) / (f.Max - f.Min)
}

func (f *Fader) Measure(c layout.Constraints) paintengine2d.Point {
	lk := f.Look()
	return c.Constrain(paintengine2d.Pt(lk.Metrics().SliderH+style.Dip(lk, 8), style.Dip(lk, 64)))
}

func (f *Fader) Arrange(r paintengine2d.Rect) { f.SetBounds(r) }

// Paint asks the look for a horizontal slider inside a rotated frame, so a
// fader is painted from the same sprites a slider is and needs no art of its
// own. The value is turned over with the frame: up is more.
func (f *Fader) Paint(ctx *paintengine2d.Context) {
	lk, b := f.Look(), f.LocalBounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	st := f.State()
	if f.hovered {
		st |= style.StateHovered
	}
	if f.drag {
		st |= style.StatePressed
	}
	ctx.Save()
	// Rotate about the box's centre, then draw the slider in the box with
	// its width and height swapped.
	cx, cy := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2
	ctx.Translate(cx, cy)
	ctx.Rotate(-90 * 3.14159265 / 180)
	ctx.Translate(-cy, -cx)
	turned := paintengine2d.XYWH(b.Min.Y, b.Min.X, b.Dy(), b.Dx())
	lk.DrawSlider(ctx, turned, st, f.t())
	ctx.Restore()
	if f.Focused() {
		lk.DrawFocusRing(ctx, b)
	}
}

func (f *Fader) MouseEnter() { f.hovered = true; f.Base.MouseEnter() }

func (f *Fader) MouseExit() {
	f.hovered, f.drag = false, false
	f.Base.MouseExit()
}

func (f *Fader) MousePress(e widget.MouseEvent) bool {
	if !f.Enabled() {
		return false
	}
	f.MarkPointerFocus()
	f.RequestFocus()
	f.drag = true
	f.setFromY(e.Pos.Y)
	return true
}

func (f *Fader) MouseMove(e widget.MouseEvent) bool {
	if !f.drag {
		return false
	}
	f.setFromY(e.Pos.Y)
	return true
}

func (f *Fader) MouseRelease(widget.MouseEvent) bool {
	f.drag = false
	f.Invalidate()
	return true
}

// setFromY puts the value where the pointer is, with the track's ends
// trimmed by half a thumb so the two extremes are reachable.
func (f *Fader) setFromY(y float32) {
	b := f.LocalBounds()
	pad := min(style.Dip(f.Look(), 8), b.Dy()/4)
	span := b.Dy() - 2*pad
	if span <= 0 {
		return
	}
	t := 1 - clamp01((y-pad)/span)
	f.SetValue(f.Min + t*(f.Max-f.Min))
}

func (f *Fader) KeyPress(e widget.KeyEvent) bool {
	if !f.Enabled() {
		return false
	}
	step, page := f.Step, f.Page
	if step <= 0 {
		step = (f.Max - f.Min) / 24
	}
	if page <= 0 {
		page = (f.Max - f.Min) / 4
	}
	switch e.Key {
	case platform.KeyUp:
		f.MarkKeyboardFocus()
		f.SetValue(f.Value + step)
	case platform.KeyDown:
		f.MarkKeyboardFocus()
		f.SetValue(f.Value - step)
	case platform.KeyPageUp:
		f.MarkKeyboardFocus()
		f.SetValue(f.Value + page)
	case platform.KeyPageDown:
		f.MarkKeyboardFocus()
		f.SetValue(f.Value - page)
	case platform.KeyHome:
		f.MarkKeyboardFocus()
		f.SetValue(f.Max)
	case platform.KeyEnd:
		f.MarkKeyboardFocus()
		f.SetValue(f.Min)
	default:
		return false
	}
	return true
}

// Describe puts the fader in the tree as the vertical slider it is, with a
// reading a screen reader can say out loud.
func (f *Fader) Describe(n *a11y.Node) {
	n.Role = a11y.RoleSlider
	n.State |= a11y.StateVertical
	if n.Name == "" {
		n.Name = f.Label
	}
	n.HasRange = true
	n.Min, n.Max, n.Now = float64(f.Min), float64(f.Max), float64(f.Value)
	n.Step = float64(f.Step)
	if f.Format != nil {
		n.Value = f.Format(f.Value)
	}
	n.Actions = n.Actions.With(a11y.ActionIncrement).With(a11y.ActionDecrement)
}

// AccessibleAction moves the fader from the accessibility tree.
func (f *Fader) AccessibleAction(a a11y.Action) bool {
	switch a {
	case a11y.ActionIncrement:
		f.SetValue(f.Value + f.Step)
	case a11y.ActionDecrement:
		f.SetValue(f.Value - f.Step)
	default:
		return false
	}
	return true
}

// ---- the analyser ------------------------------------------------------------

// AnalyserView paints a Spectrum: a row of bars with a marker riding the
// top of each.
//
// It is not interactive and it is not in the tab order — a bar chart of
// nothing is not a control — but it *is* in the accessibility tree, as an
// image with a name, because a screen reader that met an unnamed blank
// rectangle would have nothing to say about it at all.
type AnalyserView struct {
	widget.Base
	Spectrum *Spectrum
	// Gap is the space between bars in design pixels, and MinBarW the
	// narrowest a bar may be before bands are dropped to make room.
	Gap, MinBarW float32
	// Peaks draws the markers. Off for the narrow analysers that have no
	// room for them.
	Peaks bool
	// Pixelated squares the bars off and puts every edge on a whole
	// device pixel, for the skin whose art is drawn that way.
	Pixelated bool
	// Tint overrides the accent the bars are drawn in.
	Tint paintengine2d.Color
}

// NewAnalyserView shows s.
func NewAnalyserView(s *Spectrum) *AnalyserView {
	a := &AnalyserView{Spectrum: s, Gap: 1, MinBarW: 2, Peaks: true}
	a.Init(a)
	a.SetAccessibleName("Spectrum analyser")
	return a
}

func (a *AnalyserView) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(style.Dip(a.Look(), 76), style.Dip(a.Look(), 32)))
}

func (a *AnalyserView) Arrange(r paintengine2d.Rect) { a.SetBounds(r) }

func (a *AnalyserView) Paint(ctx *paintengine2d.Context) {
	if a.Spectrum == nil || a.Spectrum.Bands() == 0 {
		return
	}
	lk, b := a.Look(), a.LocalBounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	col := a.Tint
	if col.A <= 0 {
		col = lk.Palette().Accent
	}
	gap := max(style.Dip(lk, a.Gap), 1)
	minW := max(style.Dip(lk, a.MinBarW), 1)

	// How many bands fit. The analyser is as wide as it is rather than as
	// wide as the model, so a narrow one shows fewer bands rather than a
	// row of slivers — and it averages the ones it drops in, so the
	// picture is the same shape at every width.
	n := a.Spectrum.Bands()
	for n > 1 && (b.Dx()-float32(n-1)*gap)/float32(n) < minW {
		n--
	}
	barW := (b.Dx() - float32(n-1)*gap) / float32(n)
	if barW <= 0 {
		return
	}
	per := float32(a.Spectrum.Bands()) / float32(n)
	fill := paintengine2d.Fill(col)
	peak := paintengine2d.Fill(style.Mix(col, lk.Palette().Text, 0.55))
	radius := float32(0)
	if !a.Pixelated {
		radius = min(barW/3, style.Dip(lk, 2))
	}
	for i := 0; i < n; i++ {
		lo, hi := int(float32(i)*per), int(float32(i+1)*per)
		if hi <= lo {
			hi = lo + 1
		}
		var v, p float32
		for j := lo; j < hi && j < a.Spectrum.Bands(); j++ {
			v += a.Spectrum.Bars[j]
			p += a.Spectrum.Peaks[j]
		}
		k := float32(hi - lo)
		v, p = v/k, p/k
		x := b.Min.X + float32(i)*(barW+gap)
		h := v * b.Dy()
		if a.Pixelated {
			x = snap(x)
			h = snap(h)
		}
		if h >= 1 {
			r := paintengine2d.XYWH(x, b.Max.Y-h, barW, h)
			if radius > 0 {
				ctx.DrawRoundRectCorners(r, radius, radius, 0, 0, fill)
			} else {
				ctx.DrawRect(r, fill)
			}
		}
		if !a.Peaks {
			continue
		}
		py := b.Max.Y - p*b.Dy()
		pt := max(style.Dip(lk, 1), 1)
		if a.Pixelated {
			py, pt = snap(py), snap(pt)
		}
		if py >= b.Min.Y && py+pt <= b.Max.Y {
			ctx.DrawRect(paintengine2d.XYWH(x, py, barW, pt), peak)
		}
	}
}

// Describe puts the analyser in the tree as a named image. It has no value
// to read out: "the bars are at 0.42, 0.61, 0.33" is not information, and a
// reader that tried to announce them every frame would be unusable.
func (a *AnalyserView) Describe(n *a11y.Node) {
	n.Role = a11y.RoleImage
	if n.Name == "" {
		n.Name = "Spectrum analyser"
	}
	n.Description = Disclaimer
}

func snap(v float32) float32 {
	f := float32(int(v))
	if v-f >= 0.5 {
		return f + 1
	}
	return f
}

// ---- the clock ---------------------------------------------------------------

// Pulse is the timer that moves a player: it advances the transport and the
// analyser by however much time really passed and repaints the windows that
// asked to be repainted.
//
// One timer for the whole app, not one per window. Three windows each
// waking on their own would advance the same transport three times in a
// frame, and the analyser's peak markers — which are the one part with
// memory — would fall three times as fast.
type Pulse struct {
	period time.Duration
	last   time.Time
	stop   func()
	step   func(dt time.Duration)
	// now is the clock, swappable so a test can drive a pulse without
	// waiting for one.
	now func() time.Time
}

// NewPulse makes a stopped pulse that calls step with the elapsed time.
func NewPulse(period time.Duration, step func(dt time.Duration)) *Pulse {
	if period <= 0 {
		period = 80 * time.Millisecond
	}
	return &Pulse{period: period, step: step, now: time.Now}
}

// SetClock replaces the clock a pulse measures with.
func (p *Pulse) SetClock(now func() time.Time) {
	if now != nil {
		p.now = now
	}
}

// Start begins waking on win's timer. Calling it twice is a no-op, so a
// player that starts the pulse whenever it starts playing need not track
// whether it is already running.
func (p *Pulse) Start(win *app.Window) {
	if p == nil || p.stop != nil || win == nil {
		return
	}
	p.last = p.now()
	var arm func()
	arm = func() {
		p.stop = win.AfterFunc(p.period, func() {
			p.stop = nil
			p.Tick()
			arm()
		})
	}
	arm()
}

// Stop ends the pulse. The elapsed time is not banked: a player that was
// paused for an hour does not jump an hour when it starts again, because
// Start resets the mark.
func (p *Pulse) Stop() {
	if p == nil || p.stop == nil {
		return
	}
	p.stop()
	p.stop = nil
}

// Running reports whether the pulse is armed.
func (p *Pulse) Running() bool { return p != nil && p.stop != nil }

// Tick advances by the time since the last tick and calls step. A test
// calls it directly.
func (p *Pulse) Tick() {
	if p == nil || p.step == nil {
		return
	}
	now := p.now()
	dt := now.Sub(p.last)
	p.last = now
	if dt < 0 {
		dt = 0
	}
	// A wake that was delayed past a few periods — the machine was asleep,
	// or another app had the GPU — is clamped. The transport is allowed to
	// know about the lost time (Tick handles any dt at all); the analyser's
	// peak markers are not, and a marker that fell for four seconds in one
	// frame is a row of bars with no markers on it.
	if limit := 8 * p.period; dt > limit {
		dt = limit
	}
	p.step(dt)
}

// ---- the keyboard ------------------------------------------------------------

// CommandFor maps a key press onto a transport command, or CmdNone.
//
// One table for all three players, because a person who learns the keys in
// one should not have to learn them again in the next. They are the keys
// these applications have always used: space plays and pauses, the arrows
// seek and set the volume, B and Z step (the two keys either side of the
// transport row on a keyboard of the era), and the single letters do the
// toggles.
//
// Nothing here takes a modifier, which is deliberate: a player's keys have
// to work while the focus is anywhere in its window, and a window whose
// bare letters are shortcuts must not also have a text field in it. None of
// the three does.
func CommandFor(e widget.KeyEvent) Command {
	if e.Mods.Ctrl() || e.Mods.Alt() {
		return CmdNone
	}
	switch e.Key {
	case platform.KeySpace:
		return CmdPlayPause
	case platform.KeyLeft:
		return CmdSeekBack
	case platform.KeyRight:
		return CmdSeekForward
	case platform.KeyUp:
		return CmdVolumeUp
	case platform.KeyDown:
		return CmdVolumeDown
	}
	switch e.Rune {
	case 'x', 'X':
		return CmdPlayPause
	case 'c', 'C':
		return CmdPlayPause
	case 'v', 'V':
		return CmdStop
	case 'b', 'B':
		return CmdNext
	case 'z', 'Z':
		return CmdPrev
	case 'm', 'M':
		return CmdMute
	case 's', 'S':
		return CmdShuffle
	case 'r', 'R':
		return CmdRepeat
	}
	return CmdNone
}

// KeyHelp is the line every player shows for its keys, so the three agree
// and a screenshot of one explains the others.
const KeyHelp = "Space play/pause · Z/B track · V stop · ←/→ seek · ↑/↓ volume · M mute · S shuffle · R repeat"
