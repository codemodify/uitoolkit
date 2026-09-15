package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Segmented is a row of joined buttons, one of them chosen — Aqua's
// segmented control, GTK's linked toggle buttons, WinUI's SegmentedControl;
// the usual switch between views ("List | Cards | Columns"). Each segment is
// the look's own toggle tool button inside one frame, so every theme draws
// the choice pressed in its own way. Left / Right move the choice.
type Segmented struct {
	widget.Base
	Segments []string
	Selected int
	OnChange func(int)
	hot      int
	down     int
}

// NewSegmented chooses selected among segments.
func NewSegmented(segments []string, selected int, on func(int)) *Segmented {
	s := &Segmented{Segments: segments, Selected: selected, OnChange: on, hot: -1, down: -1}
	s.Init(s)
	s.SetWantsFocus(true)
	return s
}

// segW is each segment's width: the widest label plus padding, every segment as
// wide as the widest (a segmented control reads as one object).
func (s *Segmented) segW() float32 {
	lk := s.Look()
	f := style.ControlFontOf(lk, style.RoleButton)
	var w float32
	for _, t := range s.Segments {
		if a := f.Advance(t); a > w {
			w = a
		}
	}
	return w + lk.Metrics().Pad*2 + style.Dip(lk, 12)
}

func (s *Segmented) Measure(c layout.Constraints) paintengine2d.Point {
	lk := s.Look()
	return c.Constrain(paintengine2d.Pt(s.segW()*float32(len(s.Segments)), lk.Metrics().ControlH))
}

func (s *Segmented) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

// segRect is segment i in local coordinates.
func (s *Segmented) segRect(i int) paintengine2d.Rect {
	b := s.LocalBounds()
	n := len(s.Segments)
	if n == 0 {
		return paintengine2d.Rect{}
	}
	w := b.Dx() / float32(n)
	return paintengine2d.XYWH(float32(i)*w, 0, w, b.Dy())
}

func (s *Segmented) Paint(ctx *paintengine2d.Context) {
	lk := s.Look()
	n := len(s.Segments)
	if n == 0 {
		return
	}
	b := s.LocalBounds()
	p := lk.Palette()
	r := lk.Metrics().RadiusSmall
	// One object: a frame round the whole control, dividers at the seams,
	// and each segment a toggle tool button, which every look draws pressed
	// in when chosen (the toolbar toggle groups of the Win95 era, GTK's
	// linked buttons today).
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(p.Border, 1))
	base := s.State() &^ (style.StateHovered | style.StatePressed | style.StateFocused)
	inset := style.Dip(lk, 1)
	for i, label := range s.Segments {
		cell := s.segRect(i)
		st := base | style.StateToggle
		if i == s.Selected {
			st |= style.StateChecked
		}
		if i == s.hot {
			st |= style.StateHovered
			if i == s.down {
				st |= style.StatePressed
			}
		}
		if i == 0 {
			st |= style.StateFirst
		}
		if i == n-1 {
			st |= style.StateLast
		}
		lk.DrawToolButton(ctx, cell.Inset(inset), st, label, style.IconNone)
		if i > 0 {
			ctx.DrawRect(paintengine2d.XYWH(cell.Min.X-0.5, b.Min.Y+inset*3, 1, b.Dy()-inset*6), paintengine2d.Fill(p.Divider))
		}
		if i == s.Selected && s.Focused() {
			lk.DrawFocusRing(ctx, cell.Inset(inset*3))
		}
	}
}

func (s *Segmented) hit(p paintengine2d.Point) int {
	for i := range s.Segments {
		if s.segRect(i).Contains(p) {
			return i
		}
	}
	return -1
}

func (s *Segmented) choose(i int) {
	if i < 0 || i >= len(s.Segments) || i == s.Selected {
		return
	}
	s.Selected = i
	s.Invalidate()
	if s.OnChange != nil {
		s.OnChange(i)
	}
}

func (s *Segmented) MouseMove(e widget.MouseEvent) bool {
	if h := s.hit(e.Pos); h != s.hot {
		s.hot = h
		s.Invalidate()
	}
	return true
}

func (s *Segmented) MouseExit() {
	s.hot, s.down = -1, -1
	s.Invalidate()
	s.Base.MouseExit()
}

func (s *Segmented) MousePress(e widget.MouseEvent) bool {
	if e.Button != platform.ButtonLeft || !s.Enabled() {
		return false
	}
	s.RequestFocus()
	s.down = s.hit(e.Pos)
	s.hot = s.down
	s.Invalidate()
	return true
}

func (s *Segmented) MouseRelease(e widget.MouseEvent) bool {
	down := s.down
	s.down = -1
	s.Invalidate()
	if down >= 0 && s.hit(e.Pos) == down {
		s.choose(down)
	}
	return true
}

func (s *Segmented) KeyPress(e widget.KeyEvent) bool {
	switch e.Key {
	case platform.KeyLeft:
		s.choose(s.Selected - 1)
	case platform.KeyRight:
		s.choose(s.Selected + 1)
	case platform.KeyHome:
		s.choose(0)
	case platform.KeyEnd:
		s.choose(len(s.Segments) - 1)
	default:
		return false
	}
	return true
}
