package uitest

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// Session is a mounted widget tree with injected input (no OS window).
type Session struct {
	Host    *Host
	Root    widget.Component
	Box     paintengine2d.Rect
	hover   widget.Component
	capture widget.Component
}

// Mount hosts root, Measures/Arranges it into box, and returns a session.
func Mount(root widget.Component, box paintengine2d.Rect) *Session {
	return MountHost(NewHost(), root, box)
}

// MountHost is Mount with a caller-owned host (shared look / cursor).
func MountHost(h *Host, root widget.Component, box paintengine2d.Rect) *Session {
	if h == nil {
		h = NewHost()
	}
	s := &Session{Host: h, Root: root, Box: box}
	h.SetSurfaceSize(int(box.Dx()), int(box.Dy()))
	if root != nil {
		root.SetHost(h)
	}
	s.Layout()
	return s
}

// Layout Measure+Arranges the tree into Box (Qt updateGeometry / GTK allocate).
func (s *Session) Layout() {
	if s == nil || s.Root == nil {
		return
	}
	_ = s.Root.Measure(layout.Tight(s.Box.Dx(), s.Box.Dy()))
	s.Root.Arrange(s.Box)
}

// Relayout assigns a new box and Measure/Arranges.
func (s *Session) Relayout(box paintengine2d.Rect) {
	s.Box = box
	if s.Host != nil {
		s.Host.SetSurfaceSize(int(box.Dx()), int(box.Dy()))
	}
	s.Layout()
}

// Hit is HitRoot in session (root-local) coordinates.
func (s *Session) Hit(p paintengine2d.Point) widget.Component {
	if s == nil {
		return nil
	}
	if s.Host != nil {
		if pop := s.Host.Popup(); pop != nil {
			if h := widget.HitRoot(pop, p); h != nil {
				return h
			}
		}
	}
	if s.Root == nil {
		return nil
	}
	return widget.HitRoot(s.Root, p)
}

func local(c widget.Component, p paintengine2d.Point) paintengine2d.Point {
	o := widget.DeviceOrigin(c)
	return paintengine2d.Pt(p.X-o.X, p.Y-o.Y)
}

func (s *Session) syncCursor(c widget.Component, p paintengine2d.Point) {
	if s == nil || s.Host == nil {
		return
	}
	cur := platform.CursorDefault
	if c != nil {
		if hint, ok := c.(widget.CursorHint); ok {
			cur = hint.CursorAt(local(c, p))
		}
	}
	widget.ApplyCursor(s.Host, cur)
}

// MouseMove injects a move (and enter/exit) at root-local p.
func (s *Session) MouseMove(p paintengine2d.Point) {
	s.mouseMove(p, platform.ButtonNone)
}

func (s *Session) mouseMove(p paintengine2d.Point, btn platform.MouseButton) {
	t := s.capture
	if t == nil {
		t = s.Hit(p)
	}
	if t != s.hover {
		if s.hover != nil {
			s.hover.MouseExit()
		}
		s.hover = t
		if t != nil {
			t.MouseEnter()
		}
	}
	if t != nil {
		lp := local(t, p)
		t.MouseMove(widget.MouseEvent{Pos: lp, Button: btn})
		s.syncCursor(t, p)
		return
	}
	s.syncCursor(nil, p)
}

// MousePress presses button at p (captures the hit target).
func (s *Session) MousePress(p paintengine2d.Point, button platform.MouseButton) {
	if button == platform.ButtonNone {
		button = platform.ButtonLeft
	}
	t := s.Hit(p)
	s.capture = t
	if t != nil {
		if t.WantsFocus() {
			s.Host.RequestFocus(t)
		}
		t.MousePress(widget.MouseEvent{Pos: local(t, p), Button: button})
	}
	s.syncCursor(t, p)
}

// MouseRelease releases the capture (or the hit) at p.
func (s *Session) MouseRelease(p paintengine2d.Point) {
	t := s.capture
	if t == nil {
		t = s.Hit(p)
	}
	if t != nil {
		t.MouseRelease(widget.MouseEvent{Pos: local(t, p), Button: platform.ButtonLeft})
	}
	s.capture = nil
	hit := s.Hit(p)
	if hit != s.hover {
		if s.hover != nil {
			s.hover.MouseExit()
		}
		s.hover = hit
		if hit != nil {
			hit.MouseEnter()
		}
	}
	s.syncCursor(hit, p)
}

// Drag presses at from, moves to to with the button held, then releases.
func (s *Session) Drag(from, to paintengine2d.Point) {
	s.MousePress(from, platform.ButtonLeft)
	s.mouseMove(to, platform.ButtonLeft)
	s.MouseRelease(to)
}

// Wheel injects a vertical scroll at p (positive dy = down / toward end).
func (s *Session) Wheel(p paintengine2d.Point, dy float32) {
	for c := s.Hit(p); c != nil; c = c.Parent() {
		if !c.Enabled() || !c.Visible() {
			continue
		}
		if c.MouseWheel(widget.MouseEvent{Pos: local(c, p), Scroll: paintengine2d.Pt(0, dy)}) {
			return
		}
	}
}

// Key sends a key to the focused widget (bubbles to parents).
func (s *Session) Key(key platform.Key) {
	s.KeyEvent(widget.KeyEvent{Key: key})
}

// KeyEvent sends a full key event to focus.
func (s *Session) KeyEvent(e widget.KeyEvent) {
	for c := s.Host.Focus(); c != nil; c = c.Parent() {
		if c.KeyPress(e) {
			return
		}
	}
}

// Type injects printable runes into the focused widget.
func (s *Session) Type(text string) {
	c := s.Host.Focus()
	if c == nil {
		return
	}
	for _, r := range text {
		if r >= 32 && r != 127 {
			c.TextInput(r)
		}
	}
}

// Focus requests focus on c.
func (s *Session) Focus(c widget.Component) {
	if s.Host != nil {
		s.Host.RequestFocus(c)
	}
}

// Cursor is the last pointer shape published to the host.
func (s *Session) Cursor() platform.Cursor {
	if s == nil || s.Host == nil {
		return platform.CursorDefault
	}
	return s.Host.Cursor()
}

// Paint rasterizes the tree into a new CPU image (GTK wait-for-draw analogue).
func (s *Session) Paint() *paintengine2d.Image {
	w := int(s.Box.Dx())
	h := int(s.Box.Dy())
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	if s.Root != nil {
		widget.PaintTree(s.Root, ctx, nil)
	}
	if s.Host != nil {
		if pop := s.Host.Popup(); pop != nil {
			widget.PaintTree(pop, ctx, nil)
		}
	}
	return img
}

// Record builds a retained scene (paintengine2d Recorder path).
func (s *Session) Record() *paintengine2d.Scene {
	w := int(s.Box.Dx())
	h := int(s.Box.Dy())
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	rec := paintengine2d.NewRecorder(w, h)
	ctx := paintengine2d.NewContextDevice(rec)
	if s.Root != nil {
		widget.PaintTree(s.Root, ctx, nil)
	}
	if s.Host != nil {
		if pop := s.Host.Popup(); pop != nil {
			widget.PaintTree(pop, ctx, nil)
		}
	}
	return rec.Finish()
}
