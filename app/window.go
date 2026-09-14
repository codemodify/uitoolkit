package app

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
	closed     atomic.Bool
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
	closeHides bool
	statusMenu bool
	// sweeping guards dropDeadRefs against re-entry: clearing focus runs
	// FocusLost, which widgets may answer by dismissing another layer.
	sweeping        bool
	statusMenuArmed bool
	statusMenuArmAt time.Time
}

func newWindow(a *Application, surf platform.Surface, opts platform.WindowOptions) *Window {
	w := &Window{
		app: a, surf: surf, full: true,
		tipDelay: 450 * time.Millisecond,
		layers:   widget.NewSceneCache(),
	}
	w.scale = a.windowScale(surf)
	w.look = lookAtScale(a.base, w.scale)
	w.dirty.Pad = 1
	_ = opts
	return w
}

// applyLook rebuilds this window's theme from the application base look at
// the window's own display scale.
func (w *Window) applyLook(base style.LookAndFeel) {
	if w == nil || base == nil {
		return
	}
	w.look = lookAtScale(base, w.scale)
	w.RequestLayout()
}

// syncScale re-reads the surface display scale (a move to another monitor,
// a Wayland fractional-scale change) and rebuilds the look when it moved.
// Reports whether the scale changed.
func (w *Window) syncScale() bool {
	if w == nil || w.app == nil {
		return false
	}
	next := w.app.windowScale(w.surf)
	if next <= 0 || (next/w.scale > 0.999 && next/w.scale < 1.001) {
		return false
	}
	w.scale = next
	w.look = lookAtScale(w.app.base, next)
	w.laid = false
	w.dropScene()
	w.fullInvalidate()
	return true
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

// SetCursor applies the host pointer shape (X11, Wayland, Win32, AppKit, offscreen).
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
	old := w.root
	w.root = c
	if c != nil {
		c.SetHost(w)
	}
	if old != nil && old != c {
		if d, ok := old.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	w.dropDeadRefs()
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
	// After Dismissed: an overlay restores focus to its anchor there, and
	// sweeping first would undo that.
	w.dropDeadRefs()
	w.laid = false
	w.fullInvalidate()
}

func (w *Window) Overlay() widget.Component { return w.overlay }

func (w *Window) SetPopup(c widget.Component) {
	old := w.popup
	w.popup = c
	if c != nil {
		c.SetHost(w)
		w.dismissTooltip()
	}
	if old != nil && old != c {
		if d, ok := old.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	w.dropDeadRefs()
	w.fullInvalidate()
}

func (w *Window) Popup() widget.Component { return w.popup }

func (w *Window) DismissPopup() {
	old := w.popup
	if old == nil {
		return
	}
	w.popup = nil
	if d, ok := old.(widget.Dismisser); ok {
		d.Dismissed()
	}
	w.dropDeadRefs()
	w.fullInvalidate()
}

// dropDeadRefs clears window state that points at components no longer
// reachable from any live layer. Call it after a layer field changes and
// after the outgoing layer's Dismissed has run — popups and overlays hand
// focus back to their anchor in there, and sweeping earlier would undo it.
func (w *Window) dropDeadRefs() {
	if w == nil || w.sweeping {
		return
	}
	w.sweeping = true
	defer func() { w.sweeping = false }()
	roots := []widget.Component{w.popup, w.overlay, w.root}
	widget.ClearFocusOutside(w, roots...)
	if !widget.LiveUnder(w.hover, roots...) {
		if w.hover != nil {
			w.hover.MouseExit()
		}
		w.hover = nil
	}
	if !widget.LiveUnder(w.capture, roots...) {
		w.capture = nil
	}
	if !widget.LiveUnder(w.tipHover, roots...) {
		w.tipHover = nil
		w.lastTip = ""
		w.HideTooltip()
	}
	w.syncIMECursor()
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

// dismissTooltip hides the bubble and stops the hover timer from bringing it
// straight back. Escape, a click, and focus loss mean "no tip for this
// hover"; only moving the pointer (or hovering different chrome) re-arms it.
func (w *Window) dismissTooltip() {
	if w == nil {
		return
	}
	w.HideTooltip()
	w.tipHover = nil
	w.lastTip = ""
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
	if w == nil || w.Closed() {
		return false
	}
	return !w.laid || w.full || !w.dirty.Empty()
}

// AfterFunc runs fn on the UI thread after d (widget.Timers). The returned
// stop cancels it if it has not run yet.
func (w *Window) AfterFunc(d time.Duration, fn func()) (stop func()) {
	if w == nil || fn == nil {
		return func() {}
	}
	var cancelled atomic.Bool
	t := time.AfterFunc(d, func() {
		if cancelled.Load() {
			return
		}
		w.app.Post(func() {
			if cancelled.Load() || w.Closed() {
				return
			}
			fn()
		})
	})
	return func() {
		cancelled.Store(true)
		t.Stop()
	}
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
	w.armStatusMenuIfDue()
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

const statusMenuArmDelay = 180 * time.Millisecond

func (w *Window) armStatusMenuIfDue() {
	if w == nil || !w.statusMenu || w.statusMenuArmed || w.statusMenuArmAt.IsZero() {
		return
	}
	if !time.Now().Before(w.statusMenuArmAt) {
		w.statusMenuArmed = true
	}
}

func (w *Window) dispatch(ev platform.Event) {
	switch ev.Kind {
	case platform.EventClose:
		if w.statusMenu {
			if w.app != nil {
				w.app.hideStatusMenu()
			} else {
				w.Hide()
			}
			return
		}
		if w.closeHides {
			w.Hide()
			return
		}
		w.Close()
	case platform.EventResize:
		_ = w.surf.Resize(ev.Width, ev.Height)
		// A resize is also how a move between monitors reaches us, so
		// re-read the surface display scale here.
		if !w.syncScale() {
			w.laid = false
			w.dropScene()
			w.fullInvalidate()
		}
	case platform.EventExpose:
		w.dirty.Add(paintengine2d.XYWH(ev.Pos.X, ev.Pos.Y, float32(ev.Width), float32(ev.Height)))
	case platform.EventFocusOut:
		w.resetIME()
		w.dismissTooltip()
		w.capture = nil
		// The pointer is no longer ours: leave no widget stuck in its
		// hover state behind an alt-tab.
		if w.hover != nil {
			w.hover.MouseExit()
			w.hover = nil
		}
		w.armStatusMenuIfDue()
		if w.statusMenu && w.statusMenuArmed {
			if w.app != nil {
				w.app.hideStatusMenu()
			} else {
				w.Hide()
			}
		}
	case platform.EventFocusIn:
		// Toolkit status menus arm FocusOut-dismiss after a short delay
		// so map/focus churn on Wayland does not kill the first frame.
		w.syncIMECursor()
	case platform.EventIMEPreedit:
		if t, ok := w.keyTarget().(widget.IMETarget); ok {
			t.IMEPreedit(ev.Text, ev.IMECaret)
			w.syncIMECursor()
		}
	case platform.EventIMECommit:
		target := w.keyTarget()
		if t, ok := target.(widget.IMETarget); ok {
			if ev.IMEDelBefore > 0 || ev.IMEDelAfter > 0 {
				t.IMEDeleteSurrounding(ev.IMEDelBefore, ev.IMEDelAfter)
			}
			if ev.Text != "" {
				t.IMECommit(ev.Text)
			}
			w.syncIMECursor()
		} else if target != nil {
			for _, r := range ev.Text {
				if r >= 32 && r != 127 {
					target.TextInput(r)
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
	case platform.EventPointerLeave:
		w.pointerLeft()
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
			leaf := widget.CascadeLeaf(w.popup)
			if leaf.KeyPress(widget.KeyEvent{Key: ev.Key, Mods: ev.Mods}) {
				return
			}
			// The popup owns the keyboard while it is up; bubbling on
			// would deliver the same key to it a second time (focus is
			// normally the popup itself) and then leak it to the content
			// behind the menu.
			if w.popup != nil {
				return
			}
		}
		w.bubbleKey(widget.KeyEvent{Key: ev.Key, Mods: ev.Mods})
	case platform.EventKeyUp:
		if t := w.keyTarget(); t != nil {
			t.KeyRelease(widget.KeyEvent{Key: ev.Key, Mods: ev.Mods})
		}
	case platform.EventText:
		if t := w.keyTarget(); t != nil {
			t.TextInput(ev.Rune)
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
		if h := widget.HitCascade(w.popup, p); h != nil {
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
		w.dismissTooltip()
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

// keyTarget is the component allowed to receive keys or text right now. A
// modal overlay contains the keyboard: nothing outside it may be typed into.
func (w *Window) keyTarget() widget.Component {
	if w == nil {
		return nil
	}
	return widget.KeyTarget(w.focus, w.overlay)
}

func (w *Window) bubbleKey(e widget.KeyEvent) {
	start := w.keyTarget()
	if start == nil {
		return
	}
	for c := start; c != nil; c = c.Parent() {
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
			w.dismissTooltip()
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
	w.dismissTooltip()
	if w.popup != nil {
		if widget.HitCascade(w.popup, ev.Pos) == nil {
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

// pointerLeft clears hover and pending tooltips when the pointer leaves the
// window. A drag in progress keeps its capture: the release still comes.
func (w *Window) pointerLeft() {
	if w.capture != nil {
		return
	}
	if w.hover != nil {
		w.hover.MouseExit()
		w.hover = nil
	}
	w.dismissTooltip()
	w.tipHover = nil
	w.lastTip = ""
	w.SetCursor(platform.CursorDefault)
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
	moved := w.tipPos != ev.Pos
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
		} else if moved && w.tipHover == nil && w.tooltip == nil {
			// Re-arm after a dismissal (Escape, click): real pointer
			// motion over the same widget is a fresh hover.
			w.tipHover = t
			w.tipSince = w.now()
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
	if w.statusMenu && w.popup != nil {
		w.popup.Arrange(box)
	}
	// A relayout is when subtrees appear and disappear, so this is where a
	// component that was removed without going through a layer swap stops
	// being the focus / hover / capture target.
	w.dropDeadRefs()
	w.laid = true
}

// EnvFullFrame forces a full repaint and a full present every frame. It is
// the escape hatch for a compositor that mishandles damage: set
// UITK_PAINT_FULLFRAME=1 to get the v0.14 behaviour back.
const EnvFullFrame = "UITK_PAINT_FULLFRAME"

var fullFrameOnce struct {
	once sync.Once
	on   bool
}

// fullFramePaint reports whether partial redraw is disabled by the
// environment.
func fullFramePaint() bool {
	fullFrameOnce.once.Do(func() {
		switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvFullFrame))) {
		case "", "0", "off", "false", "no":
			fullFrameOnce.on = false
		default:
			fullFrameOnce.on = true
		}
	})
	return fullFrameOnce.on
}

// paintRects is the damage to repaint and present this frame, clipped to the
// surface. A nil result means "repaint and present everything".
func (w *Window) paintRects() []paintengine2d.Rect {
	if w.full || fullFramePaint() || w.dirty.Empty() {
		return nil
	}
	ww, hh := w.surf.Size()
	if ww < 1 || hh < 1 {
		return nil
	}
	// Copy: the returned rects outlive w.dirty.Reset and are handed to the
	// device (GPUDevice keeps them until its next present).
	out := paintengine2d.Damage{Pad: w.dirty.Pad, MaxRects: w.dirty.MaxRects}
	for _, r := range w.dirty.Rects {
		out.Add(r)
	}
	out.ClipTo(paintengine2d.XYWH(0, 0, float32(ww), float32(hh)))
	if out.Empty() {
		return nil
	}
	return out.Rects
}

func (w *Window) frame() {
	if w.Closed() {
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
	rects := w.paintRects()
	if platform.WantScene() {
		w.frameScene(rects)
	} else {
		w.frameImmediate(rects)
	}
	// A nil list presents the whole surface; otherwise only the boxes we
	// actually repainted are uploaded / swapped.
	_ = w.surf.Present(rects)
	w.paints++
	w.dirty.Reset()
	w.full = false
}

// paintLayers draws content, overlay, popup cascade and tooltip in z-order.
// dirty (device pixels) only skips subtrees that cannot contribute; the
// caller's clip is what makes the result correct.
func (w *Window) paintLayers(ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
	if w.root != nil {
		widget.PaintTree(w.root, ctx, dirty)
	}
	if w.overlay != nil {
		widget.PaintTree(w.overlay, ctx, dirty)
	}
	if w.popup != nil {
		widget.PaintCascade(w.popup, ctx, dirty)
	}
	if w.tooltip != nil {
		widget.PaintTree(w.tooltip, ctx, dirty)
	}
}

func (w *Window) frameImmediate(rects []paintengine2d.Rect) {
	ctx := platform.NewPaintContext(w.surf)
	if ctx == nil {
		return
	}
	bg := w.look.Palette().Background
	if len(rects) == 0 {
		ctx.Clear(bg)
		w.paintLayers(ctx, nil)
		return
	}
	// One pass per dirty box with the device clip pinned to that box. The
	// whole tree is walked in z-order inside it, so an overlapping sibling
	// above the invalidated widget is repainted over it instead of being
	// skipped (which used to leave the lower widget on top).
	for _, r := range rects {
		var one paintengine2d.Damage
		one.Add(r)
		ctx.Save()
		ctx.ClipDeviceRect(r)
		ctx.DrawRect(r, paintengine2d.Fill(bg))
		w.paintLayers(ctx, &one)
		ctx.Restore()
	}
}

func (w *Window) frameScene(rects []paintengine2d.Rect) {
	ww, hh := w.surf.Size()
	rec := paintengine2d.NewRecorder(ww, hh)
	rec.Clear(w.look.Palette().Background)
	ctx := paintengine2d.NewContextDevice(rec)
	// The recording is always a complete display list for the window;
	// partial redraw happens at replay, where the engine clips every op to
	// the dirty box. Recording a subset would make the next partial replay
	// paint from a scene that never had the clean widgets in it.
	if w.root != nil {
		widget.RecordTree(w.root, rec, ctx, nil, w.layers, false)
	}
	if w.overlay != nil {
		widget.RecordTree(w.overlay, rec, ctx, nil, w.layers, true)
	}
	if w.popup != nil {
		widget.RecordCascade(w.popup, rec, ctx, nil, w.layers, true)
	}
	if w.tooltip != nil {
		widget.RecordTree(w.tooltip, rec, ctx, nil, w.layers, true)
	}
	w.layers.EndFrame()
	w.scene = rec.Finish()
	dev := platform.SurfaceDevice(w.surf)
	if dev == nil {
		return
	}
	if len(rects) == 0 {
		paintengine2d.DrawScene(w.scene, dev)
		return
	}
	dmg := paintengine2d.Damage{Rects: rects}
	paintengine2d.DrawSceneDamage(w.scene, dev, &dmg)
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

// holdsNothing reports whether this window must not keep the run loop
// alive: toolkit status-menu chrome that is currently hidden is not an
// application window.
func (w *Window) holdsNothing() bool {
	return w != nil && w.statusMenu && !w.Visible()
}

// Close destroys the surface.
func (w *Window) Close() {
	if w == nil || !w.closed.CompareAndSwap(false, true) {
		return
	}
	if w.app != nil && w.app.statusMenu == w {
		w.app.statusMenu = nil
	}
	w.popup = nil
	w.overlay = nil
	w.root = nil
	w.tooltip = nil
	w.focus = nil
	w.hover = nil
	w.capture = nil
	w.tipHover = nil
	w.dropScene()
	w.scene = nil
	_ = w.surf.Close()
	if w.app != nil {
		w.app.remove(w)
	}
}

// Closed reports whether Close has run.
func (w *Window) Closed() bool { return w == nil || w.closed.Load() }

// SetCloseHides maps the window-manager close button to Hide (close-to-tray).
func (w *Window) SetCloseHides(on bool) { w.closeHides = on }

// Raise maps and activates the native window (X11 _NET_ACTIVE_WINDOW).
func (w *Window) Raise() {
	if w == nil || w.Closed() {
		return
	}
	platform.RaiseSurface(w.surf)
	w.fullInvalidate()
}

// Show maps a hidden window.
func (w *Window) Show() {
	if w == nil || w.Closed() {
		return
	}
	platform.RaiseSurface(w.surf)
}

// Hide unmaps / minimizes without destroying the surface.
func (w *Window) Hide() {
	if w == nil || w.Closed() {
		return
	}
	platform.HideSurface(w.surf)
}

// Visible reports whether the surface is mapped.
func (w *Window) Visible() bool {
	if w == nil || w.Closed() {
		return false
	}
	return platform.SurfaceVisible(w.surf)
}

// Idle is used by tests that want a timestamp.
func (w *Window) Idle() time.Time { return time.Now() }
