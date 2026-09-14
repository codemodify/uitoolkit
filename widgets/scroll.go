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
	OffsetY      float32
	child        widget.Component
	content      paintengine2d.Point
	bar          scrollDrag
	sceneOff     float32
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
		s.content = s.child.Measure(layout.Constraints{MaxW: cw, MaxH: -1})
	}
	w, h := s.content.X+g, float32(160)
	if c.HasMaxH() {
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
	s.content = s.child.Measure(layout.Constraints{MinW: cw, MaxW: cw, MaxH: -1})
	if s.content.Y < r.Dy() {
		s.content.Y = r.Dy()
	}
	s.clamp()
	s.child.Arrange(paintengine2d.XYWH(0, -s.OffsetY, cw, s.content.Y))
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
	return layout.MaxScroll(s.content.Y, s.LocalBounds().Dy())
}

func (s *ScrollView) clamp() {
	s.OffsetY = layout.ClampScroll(s.OffsetY, s.content.Y, s.LocalBounds().Dy())
}

func (s *ScrollView) lineStep() float32 {
	h := s.Look().Font().Height()
	if h < 12 {
		h = 18
	}
	return h + 4
}

func (s *ScrollView) pageStep() float32 {
	h := s.LocalBounds().Dy() * 0.9
	if h < 24 {
		h = 24
	}
	return h
}

func (s *ScrollView) vparts() style.ScrollParts {
	return vScrollParts(s.Look(), s.LocalBounds(), s.content.Y, s.OffsetY)
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
	return paintengine2d.Translation(0, s.sceneOff-s.OffsetY), true
}

func (s *ScrollView) MarkSceneChildDirty() { s.contentDirty = true }

func (s *ScrollView) NoteSceneRecorded() {
	s.sceneOff = s.OffsetY
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
	s.bar.paint(ctx, lk, s.vparts(), true)
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
	dy := e.Scroll.Y
	if dy == 0 && e.Scroll.X == 0 {
		return false
	}
	if s.maxOff() <= 0 {
		return false
	}
	before := s.OffsetY
	s.ScrollBy(wheelDelta(dy, s.lineStep()))
	return s.OffsetY != before
}

func (s *ScrollView) MousePress(e widget.MouseEvent) bool {
	if !s.Enabled() {
		return false
	}
	if s.bar.press(s, e.Pos, s.vaxis()) {
		s.MarkPointerFocus()
		s.RequestFocus()
		s.Invalidate()
		return true
	}
	return false
}

func (s *ScrollView) MouseMove(e widget.MouseEvent) bool {
	handled, dirty := s.bar.move(e.Pos, s.vaxis())
	if dirty {
		s.Invalidate()
	}
	return handled
}

func (s *ScrollView) MouseRelease(widget.MouseEvent) bool {
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
	}
	return false
}
