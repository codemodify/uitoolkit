package app

import (
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Window is a widget host that paints into a platform.Surface.
type Window struct {
	app        *Application
	surf       platform.Surface
	root       widget.Component
	overlay    widget.Component
	popup      widget.Component
	tooltip    widget.Component
	look       style.LookAndFeel
	dirty      paintengine2d.Damage
	full       bool
	focus      widget.Component
	hover      widget.Component
	capture    widget.Component
	closed     bool
	blink      bool
	laid       bool
	scale      float32
	tipHover   widget.Component
	tipSince   time.Time
	tipPos     paintengine2d.Point
	tipDelay   time.Duration
	clock      func() time.Time
	lastTip    string
	animPeriod time.Duration
	layers     *widget.SceneCache
	scene      *paintengine2d.Scene
	cursor     platform.Cursor
	paints     int
}

func newWindow(a *Application, surf platform.Surface, opts platform.WindowOptions) *Window {
	w := &Window{
		app: a, surf: surf, look: a.look, scale: a.scale, full: true,
		tipDelay: 450 * time.Millisecond,
		layers:   widget.NewSceneCache(),
	}
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

// SetFullscreen asks the native backend (EWMH / xdg-shell) when available.
func (w *Window) SetFullscreen(on bool) { platform.SetFullscreen(w.surf, on) }

// SetMaximized asks the native backend when available.
func (w *Window) SetMaximized(on bool) { platform.SetMaximized(w.surf, on) }

// SetCursor applies the pointer shape on X11 / Wayland / offscreen.
func (w *Window) SetCursor(c platform.Cursor) {
	if w == nil {
		return
	}
	w.cursor = c
	platform.SetCursor(w.surf, c)
}

// Cursor is the last shape passed to SetCursor.
func (w *Window) Cursor() platform.Cursor { return w.cursor }

func (w *Window) syncCursor(target widget.Component, local paintengine2d.Point) {
	cur := platform.CursorDefault
	if target != nil {
		if h, ok := target.(widget.CursorHint); ok {
			cur = h.CursorAt(local)
		}
	}
	w.SetCursor(cur)
}

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
		w.HideTooltip()
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

func (w *Window) SetTooltip(c widget.Component) {
	old := w.tooltip
	w.tooltip = c
	if c != nil {
		c.SetHost(w)
		w.Invalidate(c, c.LocalBounds())
	}
	if old != nil && old != c {
		w.Invalidate(old, old.LocalBounds())
	}
}

func (w *Window) Tooltip() widget.Component { return w.tooltip }

func (w *Window) HideTooltip() {
	if w.tooltip == nil {
		return
	}
	old := w.tooltip
	w.tooltip = nil
	w.tipSince = w.now()
	w.Invalidate(old, old.LocalBounds())
}

// SetTooltipDelay overrides the hover rest time. Zero keeps the default 450ms.
func (w *Window) SetTooltipDelay(d time.Duration) {
	if d <= 0 {
		d = 450 * time.Millisecond
	}
	w.tipDelay = d
}

// SetClock injects time for tooltip tests. nil uses time.Now.
func (w *Window) SetClock(now func() time.Time) { w.clock = now }

func (w *Window) now() time.Time {
	if w.clock != nil {
		return w.clock()
	}
	return time.Now()
}

// RevealTooltip shows the current hover tip immediately (screenshots / tests).
func (w *Window) RevealTooltip() {
	if w.hover == nil {
		return
	}
	text := widget.TooltipText(w.hover)
	if text == "" {
		return
	}
	w.showTip(text, w.tipPos)
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
	if w.layers != nil {
		w.layers.Invalidate(c.ID())
	}
	for p := c.Parent(); p != nil; p = p.Parent() {
		if sl, ok := p.(widget.SceneLayer); ok && sl.SceneChild() != nil {
			sl.MarkSceneChildDirty()
		}
	}
}

func (w *Window) RequestFocus(c widget.Component) {
	if w.focus == c {
		return
	}
	if w.focus != nil {
		if t, ok := w.focus.(widget.IMETarget); ok {
			t.IMEReset()
		}
		w.focus.FocusLost()
	}
	w.focus = c
	if c != nil {
		c.FocusGained()
	}
	w.syncIMECursor()
}

func (w *Window) resetIME() {
	if t, ok := w.focus.(widget.IMETarget); ok {
		t.IMEReset()
	}
}

func (w *Window) syncIMECursor() {
	s, ok := w.surf.(platform.IMESurface)
	if !ok {
		return
	}
	if t, ok := w.focus.(widget.IMETarget); ok {
		s.SetIMEEnabled(true)
		r := t.IMECaretRect()
		o := widget.DeviceOrigin(w.focus)
		s.SetIMECursor(int(o.X+r.Min.X), int(o.Y+r.Min.Y), int(r.Dx()), int(r.Dy()))
		return
	}
	s.SetIMEEnabled(false)
	s.SetIMECursor(0, 0, 0, 0)
}

// RequestLayout marks the tree dirty so the next frame Measure/Arranges.
func (w *Window) RequestLayout() {
	w.laid = false
	w.fullInvalidate()
}

func (w *Window) fullInvalidate() {
	ww, hh := w.surf.Size()
	w.dirty.Reset()
	w.dirty.Add(paintengine2d.XYWH(0, 0, float32(ww), float32(hh)))
	w.full = true
}

// dropScene forgets retained groups. Layout, look, and a new content
// tree must re-record; a full present of an unchanged tree must not.
func (w *Window) dropScene() {
	if w.layers != nil {
		w.layers.Reset()
	}
}

func (w *Window) toggleBlink() {
	w.blink = !w.blink
	if w.focus == nil {
		return
	}
	if b, ok := w.focus.(interface{ SetCaretBlink(bool) }); ok {
		b.SetCaretBlink(w.blink)
		w.focus.Invalidate()
	}
}

func (w *Window) wantsBlink() bool {
	if w == nil || w.focus == nil {
		return false
	}
	_, ok := w.focus.(interface{ SetCaretBlink(bool) })
	return ok
}

func (w *Window) needsPaint() bool {
	if w == nil || w.closed {
		return false
	}
	return !w.laid || w.full || !w.dirty.Empty()
}

// RequestAnim asks Run to wake at least every d (busy indicators).
// d <= 0 clears the request so idle can sleep on the display fd.
func (w *Window) RequestAnim(d time.Duration) { w.animPeriod = d }

func (w *Window) tipDeadline(_ time.Time) (time.Time, bool) {
	if w.popup != nil || w.overlay != nil || w.tooltip != nil || w.tipHover == nil {
		return time.Time{}, false
	}
	if widget.TooltipText(w.tipHover) == "" {
		return time.Time{}, false
	}
	return w.tipSince.Add(w.tipDelay), true
}

const pumpBurstCap = 64

func (w *Window) pump() {
	for i := 0; i < pumpBurstCap; i++ {
		evs := w.surf.Poll()
		if len(evs) == 0 {
			return
		}
		for _, ev := range evs {
			w.dispatch(ev)
		}
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
	case platform.EventFocusOut:
		w.resetIME()
		w.HideTooltip()
		w.capture = nil
	case platform.EventFocusIn:
		w.syncIMECursor()
	case platform.EventIMEPreedit:
		if t, ok := w.focus.(widget.IMETarget); ok {
			t.IMEPreedit(ev.Text, ev.IMECaret)
			w.syncIMECursor()
		}
	case platform.EventIMECommit:
		if t, ok := w.focus.(widget.IMETarget); ok {
			if ev.IMEDelBefore > 0 || ev.IMEDelAfter > 0 {
				t.IMEDeleteSurrounding(ev.IMEDelBefore, ev.IMEDelAfter)
			}
			if ev.Text != "" {
				t.IMECommit(ev.Text)
			}
			w.syncIMECursor()
		} else if w.focus != nil {
			for _, r := range ev.Text {
				if r >= 32 && r != 127 {
					w.focus.TextInput(r)
				}
			}
		}
	case platform.EventIMECancel:
		w.resetIME()
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
			if w.dismissEscape() {
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
		w.bubbleKey(widget.KeyEvent{Key: ev.Key, Mods: ev.Mods})
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

func (w *Window) dismissEscape() bool {
	if w.tooltip != nil {
		w.HideTooltip()
		return true
	}
	if w.popup != nil {
		w.DismissPopup()
		return true
	}
	if w.overlay != nil {
		w.SetOverlay(nil)
		return true
	}
	return false
}

func (w *Window) bubbleKey(e widget.KeyEvent) {
	for c := w.focus; c != nil; c = c.Parent() {
		if c.KeyPress(e) {
			return
		}
	}
}

func (w *Window) showTip(text string, pos paintengine2d.Point) {
	if text == "" {
		return
	}
	bubble := widgets.NewTooltipBubble(text)
	bubble.SetHost(w)
	sz := bubble.Measure(layout.Loose(360, 80))
	origin := paintengine2d.Pt(pos.X+12, pos.Y+18)
	bubble.Arrange(paintengine2d.XYWH(origin.X, origin.Y, sz.X, sz.Y))
	widget.ClampToSurface(w.root, bubble)
	if w.root == nil && w.overlay != nil {
		widget.ClampToSurface(w.overlay, bubble)
	}
	w.SetTooltip(bubble)
}

func (w *Window) tickTips() {
	if w.popup != nil || w.overlay != nil {
		if w.tooltip != nil {
			w.HideTooltip()
		}
		return
	}
	if w.tooltip != nil {
		return
	}
	if w.tipHover == nil {
		return
	}
	text := widget.TooltipText(w.tipHover)
	if text == "" {
		return
	}
	if w.now().Sub(w.tipSince) < w.tipDelay {
		return
	}
	w.showTip(text, w.tipPos)
}

func (w *Window) mouseDown(ev platform.Event) {
	w.HideTooltip()
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
		lp := local(t, ev.Pos)
		t.MousePress(widget.MouseEvent{Pos: lp, Button: ev.Button, Mods: ev.Mods})
		w.syncCursor(t, lp)
	}
}

func (w *Window) mouseUp(ev platform.Event) {
	t := w.capture
	if t == nil {
		t = w.hit(ev.Pos)
	}
	if t != nil {
		lp := local(t, ev.Pos)
		t.MouseRelease(widget.MouseEvent{Pos: lp, Button: ev.Button, Mods: ev.Mods})
	}
	w.capture = nil
	hit := w.hit(ev.Pos)
	if hit != nil {
		w.syncCursor(hit, local(hit, ev.Pos))
	} else {
		w.SetCursor(platform.CursorDefault)
	}
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
		w.HideTooltip()
		w.tipHover = t
		w.tipSince = w.now()
	}
	w.tipPos = ev.Pos
	if t != nil {
		lp := local(t, ev.Pos)
		t.MouseMove(widget.MouseEvent{Pos: lp, Button: ev.Button, Mods: ev.Mods})
		w.syncCursor(t, lp)
		text := widget.TooltipText(t)
		if text != w.lastTip {
			w.HideTooltip()
			w.tipHover = t
			w.tipSince = w.now()
			w.lastTip = text
		}
	} else {
		w.lastTip = ""
		w.SetCursor(platform.CursorDefault)
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
	widget.MarkKeyboardFocus(list[idx])
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
		w.dropScene()
		w.fullInvalidate()
	}
	w.tickTips()
	if w.dirty.Empty() && !w.full {
		return
	}
	if platform.WantScene() {
		w.frameScene()
	} else {
		w.frameImmediate()
	}
	// Full present. DrawScene already painted the whole graph; partial
	// PresentRects left stale Wayland tiles in v0.14.x.
	_ = w.surf.Present(nil)
	w.paints++
	w.dirty.Reset()
	w.full = false
}

func (w *Window) frameImmediate() {
	ctx := platform.NewPaintContext(w.surf)
	if ctx == nil {
		return
	}
	var paintDirty *paintengine2d.Damage
	if w.full {
		ctx.Clear(w.look.Palette().Background)
		paintDirty = nil
	} else {
		paintDirty = &w.dirty
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
	if w.tooltip != nil {
		widget.PaintTree(w.tooltip, ctx, nil)
	}
}

func (w *Window) frameScene() {
	ww, hh := w.surf.Size()
	rec := paintengine2d.NewRecorder(ww, hh)
	rec.Clear(w.look.Palette().Background)
	ctx := paintengine2d.NewContextDevice(rec)
	var paintDirty *paintengine2d.Damage
	if !w.full {
		paintDirty = &w.dirty
	}
	if w.root != nil {
		widget.RecordTree(w.root, rec, ctx, paintDirty, w.layers, false)
	}
	if w.overlay != nil {
		widget.RecordTree(w.overlay, rec, ctx, nil, w.layers, true)
	}
	if w.popup != nil {
		widget.RecordTree(w.popup, rec, ctx, nil, w.layers, true)
	}
	if w.tooltip != nil {
		widget.RecordTree(w.tooltip, rec, ctx, nil, w.layers, true)
	}
	w.scene = rec.Finish()
	dev := platform.SurfaceDevice(w.surf)
	if dev == nil {
		return
	}
	// Full replay (v0.13.8). Do not call DrawSceneDamage / strip present.
	paintengine2d.DrawScene(w.scene, dev)
}

// Scene is the last retained graph (tests / inspector).
func (w *Window) Scene() *paintengine2d.Scene { return w.scene }

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
