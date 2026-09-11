package app

import (
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Window is a widget host that paints into a platform.Surface.
type Window struct {
	app     *Application
	surf    platform.Surface
	root    widget.Component
	overlay widget.Component
	popup   widget.Component
	look    style.LookAndFeel
	dirty   paintengine2d.Damage
	full    bool
	focus   widget.Component
	hover   widget.Component
	capture widget.Component
	closed  bool
	blink   bool
	laid    bool
	scale   float32
}

func newWindow(a *Application, surf platform.Surface, opts platform.WindowOptions) *Window {
	w := &Window{app: a, surf: surf, look: a.look, scale: a.scale, full: true}
	w.dirty.Pad = 1
	_ = opts
	return w
}

func (w *Window) Look() style.LookAndFeel   { return w.look }
func (w *Window) Scale() float32            { return w.scale }
func (w *Window) Focus() widget.Component   { return w.focus }
func (w *Window) Surface() platform.Surface { return w.surf }
func (w *Window) Title() string             { return w.surf.Title() }
func (w *Window) SurfaceSize() (int, int)   { return w.surf.Size() }

func (w *Window) SetTitle(s string) { w.surf.SetTitle(s) }

func (w *Window) SetContent(c widget.Component) {
	w.root = c
	if c != nil {
		c.SetHost(w)
	}
	w.laid = false
	w.fullInvalidate()
}

func (w *Window) Content() widget.Component { return w.root }

func (w *Window) SetOverlay(c widget.Component) {
	if w.overlay == c {
		return
	}
	old := w.overlay
	w.overlay = c
	if c != nil {
		c.SetHost(w)
	}
	if old != nil {
		if d, ok := old.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	w.laid = false
	w.fullInvalidate()
}

func (w *Window) Overlay() widget.Component { return w.overlay }

func (w *Window) SetPopup(c widget.Component) {
	if w.popup != nil && w.popup != c {
		if d, ok := w.popup.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	w.popup = c
	if c != nil {
		c.SetHost(w)
	}
	w.fullInvalidate()
}

func (w *Window) Popup() widget.Component { return w.popup }

func (w *Window) DismissPopup() {
	if w.popup == nil {
		return
	}
	if d, ok := w.popup.(widget.Dismisser); ok {
		d.Dismissed()
	}
	w.popup = nil
	w.fullInvalidate()
}

func (w *Window) Invalidate(c widget.Component, local paintengine2d.Rect) {
	if c == nil {
		w.fullInvalidate()
		return
	}
	dev := widget.DeviceBounds(c)
	r := local.Translate(dev.Min)
	if r.Empty() {
		r = dev
	}
	w.dirty.Add(r.Inset(-1))
}

func (w *Window) RequestFocus(c widget.Component) {
	if w.focus == c {
		return
	}
	if w.focus != nil {
		w.focus.FocusLost()
	}
	w.focus = c
	if c != nil {
		c.FocusGained()
	}
}

func (w *Window) fullInvalidate() {
	ww, hh := w.surf.Size()
	w.dirty.Reset()
	w.dirty.Add(paintengine2d.XYWH(0, 0, float32(ww), float32(hh)))
	w.full = true
}

func (w *Window) toggleBlink() {
	w.blink = !w.blink
	if w.focus != nil {
		if b, ok := w.focus.(interface{ SetCaretBlink(bool) }); ok {
			b.SetCaretBlink(w.blink)
		}
		w.focus.Invalidate()
	}
}

func (w *Window) pump() {
	for _, ev := range w.surf.Poll() {
		w.dispatch(ev)
	}
}

func (w *Window) dispatch(ev platform.Event) {
	switch ev.Kind {
	case platform.EventClose:
		w.Close()
	case platform.EventResize:
		_ = w.surf.Resize(ev.Width, ev.Height)
		w.laid = false
		w.fullInvalidate()
	case platform.EventExpose:
		w.dirty.Add(paintengine2d.XYWH(ev.Pos.X, ev.Pos.Y, float32(ev.Width), float32(ev.Height)))
	case platform.EventMouseDown:
		w.mouseDown(ev)
	case platform.EventMouseUp:
		w.mouseUp(ev)
	case platform.EventMouseMove:
		w.mouseMove(ev)
	case platform.EventScroll:
		w.bubbleWheel(ev)
	case platform.EventKeyDown:
		if ev.Key == platform.KeyTab {
			w.tab(!ev.Mods.Shift())
			return
		}
		if ev.Key == platform.KeyEscape {
			if w.popup != nil {
				w.DismissPopup()
				return
			}
			if w.overlay != nil {
				w.SetOverlay(nil)
				return
			}
		}
		if ev.Mods.Alt() && w.root != nil {
			if handleAlt(w.root, ev.Key) {
				return
			}
		}
		if w.popup != nil {
			if w.popup.KeyPress(widget.KeyEvent{Key: ev.Key, Mods: ev.Mods}) {
				return
			}
		}
		if w.focus != nil {
			w.focus.KeyPress(widget.KeyEvent{Key: ev.Key, Mods: ev.Mods})
		}
	case platform.EventKeyUp:
		if w.focus != nil {
			w.focus.KeyRelease(widget.KeyEvent{Key: ev.Key, Mods: ev.Mods})
		}
	case platform.EventText:
		if w.focus != nil {
			w.focus.TextInput(ev.Rune)
		}
	}
}

// CaretBlink is the current caret pulse (tests / themed fields).
func (w *Window) CaretBlink() bool { return w.blink }

func (w *Window) bubbleWheel(ev platform.Event) {
	for c := w.hit(ev.Pos); c != nil; c = c.Parent() {
		if !c.Enabled() || !c.Visible() {
			continue
		}
		if c.MouseWheel(widget.MouseEvent{Pos: local(c, ev.Pos), Scroll: ev.Scroll, Mods: ev.Mods}) {
			return
		}
	}
}

func (w *Window) hit(p paintengine2d.Point) widget.Component {
	if w.popup != nil {
		if h := widget.HitRoot(w.popup, p); h != nil {
			return h
		}
	}
	return w.hitContent(p)
}

func (w *Window) hitContent(p paintengine2d.Point) widget.Component {
	if w.overlay != nil {
		if h := widget.HitRoot(w.overlay, p); h != nil {
			return h
		}
	}
	return widget.HitRoot(w.root, p)
}

func handleAlt(root widget.Component, key platform.Key) bool {
	handled := false
	widget.Walk(root, func(c widget.Component) {
		if handled {
			return
		}
		if a, ok := c.(interface{ HandleAlt(platform.Key) bool }); ok {
			if a.HandleAlt(key) {
				handled = true
			}
		}
	})
	return handled
}

func local(c widget.Component, p paintengine2d.Point) paintengine2d.Point {
	o := widget.DeviceOrigin(c)
	return paintengine2d.Pt(p.X-o.X, p.Y-o.Y)
}

func (w *Window) mouseDown(ev platform.Event) {
	if w.popup != nil {
		if widget.HitRoot(w.popup, ev.Pos) == nil {
			under := w.hitContent(ev.Pos)
			if !widget.Retains(under) {
				w.DismissPopup()
				return
			}
		}
	}
	t := w.hit(ev.Pos)
	w.capture = t
	if t != nil && t.WantsFocus() {
		w.RequestFocus(t)
	} else if t == nil || !t.WantsFocus() {
		// click on inert chrome keeps focus unless it is the overlay dimmer
	}
	if t != nil {
		t.MousePress(widget.MouseEvent{Pos: local(t, ev.Pos), Button: ev.Button, Mods: ev.Mods})
	}
}

func (w *Window) mouseUp(ev platform.Event) {
	t := w.capture
	if t == nil {
		t = w.hit(ev.Pos)
	}
	if t != nil {
		t.MouseRelease(widget.MouseEvent{Pos: local(t, ev.Pos), Button: ev.Button, Mods: ev.Mods})
	}
	w.capture = nil
}

func (w *Window) mouseMove(ev platform.Event) {
	t := w.capture
	if t == nil {
		t = w.hit(ev.Pos)
	}
	if t != w.hover {
		if w.hover != nil {
			w.hover.MouseExit()
		}
		w.hover = t
		if t != nil {
			t.MouseEnter()
		}
	}
	if t != nil {
		t.MouseMove(widget.MouseEvent{Pos: local(t, ev.Pos), Button: ev.Button, Mods: ev.Mods})
	}
}

func (w *Window) tab(forward bool) {
	root := w.root
	if w.overlay != nil {
		root = w.overlay
	}
	if w.popup != nil {
		root = w.popup
	}
	list := widget.Focusables(root)
	if len(list) == 0 {
		return
	}
	idx := -1
	for i, c := range list {
		if c == w.focus {
			idx = i
			break
		}
	}
	if forward {
		idx++
		if idx >= len(list) {
			idx = 0
		}
	} else {
		if idx < 0 {
			idx = 0
		}
		idx--
		if idx < 0 {
			idx = len(list) - 1
		}
	}
	w.RequestFocus(list[idx])
}

func (w *Window) layout() {
	ww, hh := w.surf.Size()
	box := paintengine2d.XYWH(0, 0, float32(ww), float32(hh))
	if w.root != nil {
		_ = w.root.Measure(layout.Tight(box.Dx(), box.Dy()))
		w.root.Arrange(box)
	}
	if w.overlay != nil {
		_ = w.overlay.Measure(layout.Tight(box.Dx(), box.Dy()))
		w.overlay.Arrange(box)
	}
	w.laid = true
}

func (w *Window) frame() {
	if w.closed {
		return
	}
	if !w.laid {
		w.layout()
		w.fullInvalidate()
	}
	if w.dirty.Empty() && !w.full {
		return
	}
	img := w.surf.Buffer()
	if img == nil {
		return
	}
	ctx := paintengine2d.NewContext(img)
	var paintDirty *paintengine2d.Damage
	if w.full {
		ctx.Clear(w.look.Palette().Background)
		paintDirty = nil
	} else {
		paintDirty = &w.dirty
		// Clear each dirty box so leftover pixels do not ghost.
		for _, r := range w.dirty.Rects {
			ctx.Save()
			ctx.ClipRect(r)
			ctx.DrawRect(r, paintengine2d.Fill(w.look.Palette().Background))
			ctx.Restore()
		}
	}
	if w.root != nil {
		widget.PaintTree(w.root, ctx, paintDirty)
	}
	if w.overlay != nil {
		widget.PaintTree(w.overlay, ctx, nil)
	}
	if w.popup != nil {
		widget.PaintTree(w.popup, ctx, nil)
	}
	rects := append([]paintengine2d.Rect(nil), w.dirty.Rects...)
	if w.full {
		ww, hh := img.Width, img.Height
		rects = []paintengine2d.Rect{paintengine2d.XYWH(0, 0, float32(ww), float32(hh))}
	}
	_ = w.surf.Present(rects)
	w.dirty.Reset()
	w.full = false
}

// Capture paints a full frame and returns a clone of the pixmap.
func (w *Window) Capture() *paintengine2d.Image {
	w.laid = false
	w.fullInvalidate()
	w.frame()
	img := w.surf.Buffer()
	if img == nil {
		return nil
	}
	return img.Clone()
}

// WritePNG captures the current window and writes a PNG.
func (w *Window) WritePNG(path string) error {
	img := w.Capture()
	if img == nil {
		return errNoBuffer
	}
	return img.WritePNGFile(path)
}

var errNoBuffer = errString("uitoolkit: no window buffer")

type errString string

func (e errString) Error() string { return string(e) }

// Inject feeds a synthetic event (offscreen tests / scripted shots).
func (w *Window) Inject(ev platform.Event) {
	if o, ok := w.surf.(*platform.Offscreen); ok {
		o.Inject(ev)
		return
	}
	w.dispatch(ev)
}

// Close destroys the surface.
func (w *Window) Close() {
	if w.closed {
		return
	}
	w.closed = true
	_ = w.surf.Close()
	w.app.remove(w)
}

// Idle is used by tests that want a timestamp.
func (w *Window) Idle() time.Time { return time.Now() }
