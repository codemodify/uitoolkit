package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ScrollView clips a larger child and paints a vertical scrollbar.
type ScrollView struct {
	widget.Base
	OffsetY float32
	// OffsetX is the horizontal scroll position, used only when
	// [ScrollView.Horizontal] is set.
	OffsetX float32
	// Horizontal lets the view scroll sideways as well as up and down,
	// for content that has no narrower form: a table with more columns
	// than fit, a wide diagram, an image at its own size.
	//
	// It is off by default, and deliberately. A scroll view's usual job
	// is to give its child the width it has and let it fold — a form
	// that does not fit should drop its labels above its fields or fold
	// with [Wrap], not be scrolled sideways, which is a poor way to read
	// anything laid out in columns. Turning this on means "this content
	// genuinely cannot be narrower", and the child is then measured at
	// its natural width rather than the view's.
	Horizontal bool
	// ShrinkToContent measures to what is inside instead of taking all
	// the height it is offered.
	//
	// A scroll view's whole job is normally to fill a space and scroll
	// what does not fit, so that is the default. It is the wrong answer
	// for a panel that should be as tall as its contents *up to* a limit
	// — a message header that grows with an invitation or a list of
	// attachments and must not push the body out — which otherwise has
	// to be wrapped in a box that measures the child again and reads
	// [ScrollView.ContentHeight] back.
	//
	// With [ScrollView.MaxHeight] it is "as tall as what is in it, up to
	// N, and scrolling past that".
	ShrinkToContent bool
	// MaxHeight caps the height this view asks for, in 1x design pixels
	// (0: no cap). It applies whether or not ShrinkToContent is set: on
	// its own it means "no taller than N however much room there is".
	MaxHeight float32
	// MinHeight floors it, in 1x design pixels (0: none). It is for a
	// ShrinkToContent view that must not collapse to nothing when it is
	// empty.
	MinHeight    float32
	child        widget.Component
	content      paintengine2d.Point
	bar          scrollDrag
	sceneOff     float32
	sceneOffX    float32
	draggingH    bool
	contentDirty bool
}

var _ widget.SceneLayer = (*ScrollView)(nil)

func NewScrollView(child widget.Component) *ScrollView {
	s := &ScrollView{}
	s.Init(s)
	s.SetManagesChildren(true)
	s.SetWantsFocus(true)
	s.SetFocusVisibleOnly(true)
	if child != nil {
		s.SetChild(child)
	}
	return s
}

func (s *ScrollView) SetChild(c widget.Component) {
	s.ClearChildren()
	s.child = c
	if c != nil {
		s.Base.Add(c)
	}
}

// MaxOffset is the largest legal OffsetY.
func (s *ScrollView) MaxOffset() float32 { return s.maxOff() }

// ScrollTo sets OffsetY (clamped) and relayouts the child.
func (s *ScrollView) ScrollTo(y float32) {
	s.OffsetY = y
	s.clamp()
	if !s.Bounds().Empty() {
		s.Arrange(s.Bounds())
	}
	s.Invalidate()
}

// ScrollBy adds dy pixels to the offset.
func (s *ScrollView) ScrollBy(dy float32) { s.ScrollTo(s.OffsetY + dy) }

// MinWidth implements [widget.MinWidther]. A scroll view scrolls up and
// down, so it makes nothing narrower: its minimum is its child's, plus
// the gutter its bar takes.
func (s *ScrollView) MinWidth() float32 {
	if s.Horizontal {
		// It scrolls sideways, so its child's width is not a floor on
		// anything: the view can be as narrow as it likes.
		return s.gutter()
	}
	w := widget.MinWidthOf(s.child)
	if w <= 0 {
		return 0
	}
	return w + s.Look().Metrics().Scroll
}

func (s *ScrollView) Measure(c layout.Constraints) paintengine2d.Point {
	g := s.gutter()
	if s.child != nil {
		cw := c.MaxW
		if c.HasMaxW() {
			cw = c.MaxW - g
			if cw < 0 {
				cw = 0
			}
		}
		if s.Horizontal {
			// The child keeps whatever width it wants; the view scrolls
			// to the rest of it.
			cw = -1
		}
		s.content = s.child.Measure(layout.Constraints{MaxW: cw, MaxH: -1})
	}
	lk := s.Look()
	w, h := s.content.X+g, style.Dip(lk, 160)
	switch {
	case s.ShrinkToContent:
		h = s.content.Y
	case c.HasMaxH():
		h = c.MaxH
	}
	if s.MinHeight > 0 {
		h = max(h, style.Dip(lk, s.MinHeight))
	}
	if s.MaxHeight > 0 {
		h = min(h, style.Dip(lk, s.MaxHeight))
	}
	// The cap is this view's, not a licence to overflow the box it was
	// given: a parent that offered less still wins.
	if c.HasMaxH() && h > c.MaxH {
		h = c.MaxH
	}
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (s *ScrollView) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	if s.child == nil {
		return
	}
	cw := r.Dx() - s.gutter()
	if cw < 0 {
		cw = 0
	}
	if s.Horizontal {
		// Measured at its own width, then laid out at least as wide as
		// the view so a child narrower than the pane still fills it.
		s.content = s.child.Measure(layout.Unbounded())
		if s.content.X < cw {
			s.content.X = cw
		}
	} else {
		s.content = s.child.Measure(layout.Constraints{MinW: cw, MaxW: cw, MaxH: -1})
		s.content.X = cw
	}
	if s.content.Y < r.Dy()-s.hGutter() {
		s.content.Y = r.Dy() - s.hGutter()
	}
	s.clamp()
	s.child.Arrange(paintengine2d.XYWH(-s.OffsetX, -s.OffsetY, s.content.X, s.content.Y))
}

// gutter is the width the child gives up to the scrollbar (always
// reserved, so content does not reflow when it starts to overflow).
func (s *ScrollView) gutter() float32 { return style.ScrollGutter(s.Look()) }

// Reveal scrolls so c (a descendant) is fully visible, with a small margin
// for its focus ring (widget.Revealer).
func (s *ScrollView) Reveal(c widget.Component) {
	if c == nil || s.child == nil {
		return
	}
	// c's box in the scroll view's coordinates.
	top := float32(0)
	for p := widget.Component(c); p != nil && p != widget.Component(s); p = p.Parent() {
		top += p.Bounds().Min.Y
	}
	bot := top + c.Bounds().Dy()
	margin := float32(8)
	view := s.LocalBounds().Dy()
	switch {
	case top-margin < 0:
		s.ScrollTo(s.OffsetY + top - margin)
	case bot+margin > view:
		s.ScrollTo(s.OffsetY + bot + margin - view)
	}
}

func (s *ScrollView) maxOff() float32 {
	return layout.MaxScroll(s.content.Y, s.viewH())
}

func (s *ScrollView) clamp() {
	s.OffsetY = layout.ClampScroll(s.OffsetY, s.content.Y, s.viewH())
	if s.Horizontal {
		s.OffsetX = layout.ClampScroll(s.OffsetX, s.content.X, s.viewW())
	} else {
		s.OffsetX = 0
	}
}

// hGutter is the height given up to the horizontal bar, zero when the
// view does not scroll sideways.
func (s *ScrollView) hGutter() float32 {
	if !s.Horizontal {
		return 0
	}
	return style.ScrollGutter(s.Look())
}

// viewW and viewH are the box the content is seen through, with the
// bars' gutters taken off.
func (s *ScrollView) viewW() float32 { return max(s.LocalBounds().Dx()-s.gutter(), 0) }
func (s *ScrollView) viewH() float32 { return max(s.LocalBounds().Dy()-s.hGutter(), 0) }

// MaxOffsetX is the largest legal OffsetX.
func (s *ScrollView) MaxOffsetX() float32 {
	if !s.Horizontal {
		return 0
	}
	return layout.MaxScroll(s.content.X, s.viewW())
}

// ScrollToX sets OffsetX (clamped) and relayouts the child.
func (s *ScrollView) ScrollToX(x float32) {
	if !s.Horizontal {
		return
	}
	s.OffsetX = x
	s.clamp()
	if !s.Bounds().Empty() {
		s.Arrange(s.Bounds())
	}
	s.Invalidate()
}

// ScrollByX adds dx pixels to the horizontal offset.
func (s *ScrollView) ScrollByX(dx float32) { s.ScrollToX(s.OffsetX + dx) }

// ContentWidth is the arranged child width used for clamp and thumb.
func (s *ScrollView) ContentWidth() float32 { return s.content.X }

func (s *ScrollView) hparts() style.ScrollParts {
	if !s.Horizontal {
		return style.ScrollParts{}
	}
	return style.ScrollGeometry(s.Look(), s.LocalBounds(), false, s.content.X, s.viewW(), s.OffsetX, true)
}

func (s *ScrollView) haxis() scrollAxis {
	return scrollAxis{
		parts: s.hparts,
		get:   func() (float32, float32) { return s.OffsetX, s.MaxOffsetX() },
		set:   s.ScrollToX,
		steps: func() (float32, float32) { return s.lineStep(), max(s.viewW()*0.9, 24) },
	}
}

func (s *ScrollView) lineStep() float32 {
	h := s.Look().Font().Height()
	if h < 12 {
		h = 18
	}
	return h + 4
}

func (s *ScrollView) pageStep() float32 {
	h := s.viewH() * 0.9
	if h < 24 {
		h = 24
	}
	return h
}

func (s *ScrollView) vparts() style.ScrollParts {
	return style.ScrollGeometry(s.Look(), s.LocalBounds(), true, s.content.Y, s.viewH(), s.OffsetY, s.Horizontal)
}

func (s *ScrollView) vaxis() scrollAxis {
	return scrollAxis{
		vertical: true,
		parts:    s.vparts,
		get:      func() (float32, float32) { return s.OffsetY, s.maxOff() },
		set:      s.ScrollTo,
		steps:    func() (float32, float32) { return s.lineStep(), s.pageStep() },
	}
}

func (s *ScrollView) thumb() (track, thumb paintengine2d.Rect) {
	sp := s.vparts()
	return sp.Track, sp.Thumb
}

// ScrollTrack is the overflow bar geometry (empty thumb when content fits).
func (s *ScrollView) ScrollTrack() (track, thumb paintengine2d.Rect) { return s.thumb() }

// ContentHeight is the arranged child height used for clamp / thumb.
func (s *ScrollView) ContentHeight() float32 { return s.content.Y }

func (s *ScrollView) SceneChild() widget.Component { return s.child }

func (s *ScrollView) RetainScene() (paintengine2d.Matrix, bool) {
	if s.contentDirty || s.child == nil {
		return paintengine2d.Identity(), false
	}
	return paintengine2d.Translation(s.sceneOffX-s.OffsetX, s.sceneOff-s.OffsetY), true
}

func (s *ScrollView) MarkSceneChildDirty() { s.contentDirty = true }

func (s *ScrollView) NoteSceneRecorded() {
	s.sceneOff, s.sceneOffX = s.OffsetY, s.OffsetX
	s.contentDirty = false
}

func (s *ScrollView) Paint(ctx *paintengine2d.Context) {
	b := s.LocalBounds()
	lk := s.Look()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Background.WithAlpha(0.15)))
	if !widget.Recording(ctx) {
		ctx.Save()
		ctx.ClipRect(b)
		if s.child != nil {
			widget.PaintTree(s.child, ctx, nil)
		}
		ctx.Restore()
	}
	s.bar.paint(s, ctx, lk, s.vparts(), true, s.OffsetY)
	if s.Horizontal {
		s.bar.paint(s, ctx, lk, s.hparts(), false, s.OffsetX)
	}
	if s.State().Focused() {
		lk.DrawFocusRing(ctx, b)
	}
}

func (s *ScrollView) HitTest(local paintengine2d.Point) widget.Component {
	if !s.Visible() || !s.LocalBounds().Contains(local) {
		return nil
	}
	if sp := s.vparts(); !sp.Thumb.Empty() && sp.Bar.Contains(local) {
		return s
	}
	if sp := s.hparts(); !sp.Thumb.Empty() && sp.Bar.Contains(local) {
		return s
	}
	if s.child != nil {
		cb := s.child.Bounds()
		lp := paintengine2d.Pt(local.X-cb.Min.X, local.Y-cb.Min.Y)
		if hit := s.child.HitTest(lp); hit != nil {
			return hit
		}
	}
	return s
}

// MouseWheel scrolls, and reports false when it cannot: an unscrollable or
// already-at-the-edge view must let the wheel bubble to an outer scroll pane
// instead of swallowing it.
func (s *ScrollView) MouseWheel(e widget.MouseEvent) bool {
	dy, dx := e.Scroll.Y, e.Scroll.X
	if dy == 0 && dx == 0 {
		return false
	}
	// Shift turns a vertical wheel sideways, which is what every desktop
	// does and what a wheel with only one axis has to rely on.
	if s.Horizontal && dx == 0 && e.Mods.Shift() {
		dx, dy = dy, 0
	}
	beforeY, beforeX := s.OffsetY, s.OffsetX
	if dy != 0 && s.maxOff() > 0 {
		s.ScrollBy(wheelDelta(dy, s.lineStep(), e.Precise))
	}
	if dx != 0 && s.Horizontal && s.MaxOffsetX() > 0 {
		s.ScrollByX(wheelDelta(dx, s.lineStep(), e.Precise))
	}
	return s.OffsetY != beforeY || s.OffsetX != beforeX
}

func (s *ScrollView) MousePress(e widget.MouseEvent) bool {
	if !s.Enabled() {
		return false
	}
	if s.Horizontal && s.bar.press(s, e.Pos, s.haxis()) {
		s.draggingH = true
		s.MarkPointerFocus()
		s.RequestFocus()
		s.Invalidate()
		return true
	}
	if s.bar.press(s, e.Pos, s.vaxis()) {
		s.draggingH = false
		s.MarkPointerFocus()
		s.RequestFocus()
		s.Invalidate()
		return true
	}
	return false
}

func (s *ScrollView) MouseMove(e widget.MouseEvent) bool {
	// Whichever bar the press landed on: one drag at a time, and it
	// must keep following the bar it started on even when the pointer
	// wanders over the other.
	ax := s.vaxis()
	if s.draggingH {
		ax = s.haxis()
	}
	handled, dirty := s.bar.move(s, e.Pos, ax)
	if dirty {
		s.Invalidate()
	}
	return handled
}

func (s *ScrollView) MouseRelease(widget.MouseEvent) bool {
	s.draggingH = false
	if !s.bar.release() {
		return false
	}
	s.Invalidate()
	return true
}

func (s *ScrollView) MouseExit() {
	s.bar.exit()
	s.Base.MouseExit()
}

func (s *ScrollView) KeyPress(e widget.KeyEvent) bool {
	if !s.Enabled() {
		return false
	}
	s.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeyDown:
		s.ScrollBy(s.lineStep())
		return true
	case platform.KeyUp:
		s.ScrollBy(-s.lineStep())
		return true
	case platform.KeyPageDown:
		s.ScrollBy(s.pageStep())
		return true
	case platform.KeyPageUp:
		s.ScrollBy(-s.pageStep())
		return true
	case platform.KeyHome:
		s.ScrollTo(0)
		return true
	case platform.KeyEnd:
		s.ScrollTo(s.maxOff())
		return true
	case platform.KeyLeft:
		if !s.Horizontal {
			return false
		}
		s.ScrollByX(-s.lineStep())
		return true
	case platform.KeyRight:
		if !s.Horizontal {
			return false
		}
		s.ScrollByX(s.lineStep())
		return true
	}
	return false
}
