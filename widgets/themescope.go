package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ThemeScope runs its child subtree in its own look while the rest of the
// window keeps the window's: a live theme preview in Settings, a themed
// pane in a designer. The look is given at 1x; the scope applies the
// window's display scale itself. Popups opened from inside (combo lists,
// context menus) inherit the scope's look.
type ThemeScope struct {
	widget.Base
	child  widget.Component
	base   style.LookAndFeel
	scaled style.LookAndFeel
	scale  float32
}

// NewThemeScope wraps child in look (a 1x LookAndFeel, e.g. a theme pack's
// Look()).
func NewThemeScope(look style.LookAndFeel, child widget.Component) *ThemeScope {
	s := &ThemeScope{base: look}
	s.Init(s)
	s.SetChild(child)
	return s
}

// SetChild replaces the scoped subtree.
func (s *ThemeScope) SetChild(child widget.Component) {
	if s.child != nil {
		s.Base.Remove(s.child)
	}
	s.child = child
	if child != nil {
		s.Base.Add(child)
	}
	s.RequestLayout()
}

// Child is the scoped subtree.
func (s *ThemeScope) Child() widget.Component { return s.child }

// SetTheme switches the scope to another 1x look: the subtree relayouts
// (metrics differ between themes) and repaints.
func (s *ThemeScope) SetTheme(look style.LookAndFeel) {
	s.base = look
	s.scaled = nil
	s.RequestLayout()
	s.Invalidate()
}

// Theme is the 1x look the scope was given.
func (s *ThemeScope) Theme() style.LookAndFeel { return s.base }

// Look is the scope's look at the window's display scale. Children resolve
// their look through their parents, so the whole subtree paints with it.
func (s *ThemeScope) Look() style.LookAndFeel {
	if s.base == nil {
		return s.Base.Look()
	}
	sc := float32(1)
	if h := s.Host(); h != nil {
		sc = h.Scale()
	}
	if sc <= 0 {
		sc = 1
	}
	if s.scaled == nil || s.scale != sc {
		s.scaled = style.WithScale(s.base, sc)
		s.scale = sc
	}
	return s.scaled
}

// RequestLayout asks the window for a relayout (no-op when detached).
func (s *ThemeScope) RequestLayout() {
	if h := s.Host(); h != nil {
		h.RequestLayout()
	}
}

func (s *ThemeScope) Measure(c layout.Constraints) paintengine2d.Point {
	if s.child == nil {
		return c.Constrain(paintengine2d.Pt(0, 0))
	}
	return s.child.Measure(c)
}

func (s *ThemeScope) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	if s.child != nil {
		s.child.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
	}
}

// Paint fills the scope with its theme's window background, so the preview
// is not seen against the surrounding theme.
//
// A scope whose child cuts itself to a shape (widget.PaintClipper) — an
// in-app window cut to its look's silhouette — fills only that shape: the
// child fills the whole scope, and a background painted beyond its outline
// would put back the rectangle the outline took away.
func (s *ThemeScope) Paint(ctx *paintengine2d.Context) {
	if pc, ok := s.child.(widget.PaintClipper); ok {
		ctx.Save()
		defer ctx.Restore()
		pc.PaintClip(ctx)
	}
	ctx.DrawRect(s.LocalBounds(), paintengine2d.Fill(s.Look().Palette().Background))
}
