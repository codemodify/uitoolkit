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
	app *Application
	// altHeld: the Alt key is down (mnemonic underlines show in looks
	// that hide them otherwise).
	altHeld bool
	// dropOver is the drop target a drag from another app is over.
	dropOver widget.Component
	// dropAction is what the drag over the window now would do on a drop
	// (the source is told); dragArm is a press here that could become a
	// drag of our own.
	dropAction platform.DragAction
	dragArm    dragGesture
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
	// initialFocus is the component the window focuses when it first
	// opens (SetInitialFocus), and openFocused that it has had its one
	// chance to (focusOnOpen).
	initialFocus widget.Component
	openFocused  bool
	scale        float32
	tipHover     widget.Component
	tipSince     time.Time
	tipPos       paintengine2d.Point
	tipDelay     time.Duration
	clock        func() time.Time
	lastTip      string
	animPeriod   time.Duration
	layers       *widget.SceneCache
	scene        *paintengine2d.Scene
	// paths keeps recorded shapes across frames, so a steady UI does not
	// clone every path it records each frame.
	paths *paintengine2d.PathCache
	// inactive: the window lost keyboard focus (selections dim, GTK's
	// backdrop). Windows start active; offscreen ones never change.
	inactive   bool
	cursor     platform.Cursor
	paints     int
	closeHides bool
	// onCloseRequest, when set, decides what a desktop close does: false
	// keeps the window (a floating dock panel hides itself instead).
	onCloseRequest func() bool
	statusMenu     bool
	// sweeping guards dropDeadRefs against re-entry: clearing focus runs
	// FocusLost, which widgets may answer by dismissing another layer.
	sweeping        bool
	statusMenuArmed bool
	statusMenuArmAt time.Time
	// state is what the desktop says about the window (maximized, tiled,
	// activated, suspended); stateKnown is set once it has said, from when
	// on its Activated — not keyboard focus — drives the active look.
	state      platform.WindowState
	stateKnown bool

	// opts are the options the window was made with (the decoration policy
	// runs again whenever its inputs change).
	opts platform.WindowOptions
	// titleBar is the app's title bar (SetTitleBar); caption the header bar
	// laid out on top: titleBar itself, wrapCaption around it, or the
	// defaultCaption of a toolkit-drawn frame (frame.go).
	titleBar       widget.Component
	caption        *widgets.HeaderBar
	wrapCaption    *widgets.HeaderBar
	defaultCaption *widgets.HeaderBar
	// decor is the decoration mode in effect, caps what the desktop can
	// do for the window, geom the frame's last layout, sysFrame what the
	// window system was last told about the frame (the margin holding the
	// shadow, the resize band in it, the corner radii, alpha, and the
	// window's silhouette) and shadow the cached nine-patch that margin is
	// painted from.
	decor    platform.Decorations
	caps     platform.WMCaps
	geom     frameGeom
	sysFrame platform.Frame
	shadow   frameShadow
	// shape is the window's silhouette (SetShape) and shapeFn the callback
	// that rebuilds it at every size (SetShapeFunc); only one is ever set.
	// shapeOpaque is whether its interior may be claimed solid, glass
	// whether the app asked for the desktop behind to be blurred, and
	// eraser the cached inverse-coverage image the silhouette is punched
	// out with (eraserFor is the rasterisation it was built from).
	shape       *platform.Shape
	shapeFn     func(paintengine2d.Point, float32) *platform.Shape
	shapeOpaque bool
	glass       bool
	eraser      *paintengine2d.Image
	eraserFor   *platform.ShapeRaster
	// wipe is the silhouette's uncovered region as one path (wipeFor is
	// the rasterisation it was built from, wipeAt where it was placed).
	wipe    *paintengine2d.Path
	wipeFor *platform.ShapeRaster
	wipeAt  paintengine2d.Point
	// shapeShade is the blurred silhouette a shaped window casts instead
	// of the nine-patch shadow, and glassTint the colour an app chose for
	// its glass (the look's own when it is fully transparent).
	shapeShade shapeShadow
	glassTint  paintengine2d.Color
	// shapeCur is what shapeFn last built, for the size and scale in
	// shapeCurW / shapeCurH / shapeCurScale: a fresh Shape every frame
	// would rasterise itself afresh every frame.
	shapeCur             *platform.Shape
	shapeCurW, shapeCurH int
	shapeCurScale        float32
	// lookShapeCur is the silhouette the look declared for the frame state
	// in lookShapeKey — nil shape and all, which is what every look that
	// frames a rectangle says (app/shape.go, lookShape).
	lookShapeCur *lookShapeMemo
	lookShapeKey lookShapeKey
	capPress     captionGesture
	capClick     captionClick
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
	w.opts = opts
	w.decor = platform.SurfaceDecorations(surf)
	if f, ok := surf.(platform.FrameSurface); ok {
		w.caps = f.Capabilities()
	}
	w.rebuildCaption()
	return w
}

// applyLook rebuilds this window's theme from the application base look at
// the window's own display scale.
func (w *Window) applyLook(base style.LookAndFeel) {
	if w == nil || base == nil {
		return
	}
	w.look = lookAtScale(base, w.scale)
	if w.caption != nil {
		// The frame is the new look's (and, with the theme's button
		// layout, so are the caption buttons' places).
		w.rebuildCaption()
	}
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

// App is the application this window belongs to, so a component with its
// window in hand can open another one — a tab torn out of its strip goes
// into a window of the same application.
func (w *Window) App() *Application {
	if w == nil {
		return nil
	}
	return w.app
}

func (w *Window) Look() style.LookAndFeel   { return w.look }
func (w *Window) Scale() float32            { return w.scale }
func (w *Window) Focus() widget.Component   { return w.focus }
func (w *Window) Surface() platform.Surface { return w.surf }
func (w *Window) Title() string             { return w.surf.Title() }

// SurfaceSize is the whole buffer in device pixels — the visible window
// plus the invisible margin a frame the toolkit draws keeps for its shadow.
// Size is the window itself, which is what an app means by "the window".
func (w *Window) SurfaceSize() (int, int) { return w.surf.Size() }

// Size is the visible window in device pixels: the surface less the frame's
// margin (the same box the desktop moves, snaps and tiles).
func (w *Window) Size() (int, int) {
	box := w.WindowRect()
	return int(box.Dx()), int(box.Dy())
}

// SetSize asks for a visible window this many device pixels across. It is
// [Window.Size]'s other half: the size of the *window* the user sees, not of
// the surface, which is larger by whatever margin a frame the toolkit draws
// is keeping for its shadow. An app that wants a 468 by 96 window asks for
// 468 by 96 and never has to know the margin exists.
//
// It is a request like every other window-geometry call. A desktop may give
// a different size, and a maximized, tiled or full-screen window keeps the
// one it was given; the answer arrives as an ordinary resize.
func (w *Window) SetSize(width, height int) {
	if w == nil || w.Closed() || width < 1 || height < 1 {
		return
	}
	_ = w.surf.Resize(width, height)
}

// WindowRect is the visible window inside the surface, in surface device
// pixels (widget.WindowRecter): popups, menus and tooltips stay inside it,
// never in the margin, where a compositor may clip them and clicks fall
// through to the window behind.
func (w *Window) WindowRect() paintengine2d.Rect {
	if w == nil {
		return paintengine2d.Rect{}
	}
	return w.windowBox()
}

// SetTitle sets the window's title: the desktop's title bar and task bar
// show it, and so does a title bar the toolkit draws.
func (w *Window) SetTitle(s string) {
	if w.surf.Title() == s {
		return
	}
	w.surf.SetTitle(s)
	if w.caption != nil {
		w.caption.Invalidate()
	}
}

// SetFullscreen asks the native backend (EWMH / xdg-shell) when available.
func (w *Window) SetFullscreen(on bool) { platform.SetFullscreen(w.surf, on) }

// SetMaximized asks the native backend when available.
func (w *Window) SetMaximized(on bool) { platform.SetMaximized(w.surf, on) }

// WindowState is what the desktop last said about the window: maximized,
// full screen, tiled edges, activated, suspended.
func (w *Window) WindowState() platform.WindowState {
	if w == nil {
		return platform.WindowState{}
	}
	return w.state
}

// Minimize iconifies the window when the desktop can (a caption button,
// a title-bar action). Unlike Hide it keeps the window in the taskbar.
func (w *Window) Minimize() {
	if w == nil || w.Closed() {
		return
	}
	if f, ok := w.surf.(platform.FrameSurface); ok {
		f.Minimize()
	}
}

// ToggleMaximize maximizes the window, or restores a maximized one. A
// window the desktop may not resize is not maximized either.
func (w *Window) ToggleMaximize() {
	if w == nil || w.Closed() || !w.Resizable() {
		return
	}
	w.SetMaximized(!w.state.Maximized)
}

// Resizable reports whether the user may resize the window. A window opened
// with platform.SizingFixed may not: the desktop was told its minimum and
// its maximum are the same, and the frame the toolkit draws offers no
// resize band for it either.
func (w *Window) Resizable() bool {
	if w == nil {
		return false
	}
	return platform.SurfaceSizing(w.surf) != platform.SizingFixed
}

// SetResizable pins the window to its current size, or lets the user resize
// it again. It is [platform.WindowOptions.Sizing] after the window exists —
// for an app whose window is a design in one mode and free in another — and
// it reports whether the backend could. A fixed window may still be resized
// by the application (SetSize), which takes its pin with it.
func (w *Window) SetResizable(on bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	sz := platform.SizingFixed
	if on {
		sz = platform.SizingResizable
	}
	if !platform.SetSurfaceSizing(w.surf, sz) {
		return false
	}
	// The resize band and the maximize button both go with it.
	if f, ok := w.surf.(platform.FrameSurface); ok {
		w.caps = f.Capabilities()
	}
	w.rebuildCaption()
	return true
}

// windowStateChanged adopts the desktop's new state for the window.
func (w *Window) windowStateChanged(st platform.WindowState) {
	was := w.state
	w.state, w.stateKnown = st, true
	w.setActive(st.Activated)
	if st.Activated && !was.Activated {
		// Back from another window: the desktop's title-bar settings may
		// have changed there.
		w.app.refreshTitleBarPrefs()
	}
	switch {
	case was.Fullscreen != st.Fullscreen:
		// No frame at all in full screen.
		w.rebuildCaption()
	case was.Maximized != st.Maximized || was.Tiled != st.Tiled || was.Constrained != st.Constrained ||
		was.Solid != st.Solid:
		// The frame (borders, resize edges, the restore glyph, the margin
		// its shadow lives in) depends on these.
		w.laid = false
		w.dropScene()
		w.fullInvalidate()
	}
}

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
	w.app.owesTrim()
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

// Active reports whether the window has keyboard focus (widget.ActiveHost).
func (w *Window) Active() bool { return !w.inactive }

// setActive repaints everything when the window gains or loses keyboard
// focus: selections, focus marks and default buttons follow it.
func (w *Window) setActive(active bool) {
	if w.inactive == !active {
		return
	}
	w.inactive = !active
	w.dropScene()
	w.fullInvalidate()
}

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
	roots := w.layerRoots()
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

// layerRoots are the window's live layers below the tooltip, top first.
func (w *Window) layerRoots() []widget.Component {
	roots := []widget.Component{w.popup, w.overlay}
	if w.caption != nil {
		roots = append(roots, w.caption)
	}
	return append(roots, w.root)
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
	if c.Parent() == nil && c != w.root && c != widget.Component(w.caption) && (local.Empty() || local == c.LocalBounds()) {
		// A floating layer shown, moved or hidden: its drop shadow lies
		// outside it and must be repainted too.
		lk := w.layerLook(c)
		r = style.PopupShadowOf(lk, style.PopupMenu).Max(style.PopupShadowOf(lk, style.PopupTooltip)).Grow(r)
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
	// Containers that paint by where the focus is rather than by holding
	// it — a dock panel's title bar — hear about every move. The caption
	// is a concrete type, so it only joins the roots when there is one.
	roots := []widget.Component{w.root, w.overlay, w.popup}
	if w.caption != nil {
		roots = append(roots, w.caption)
	}
	widget.NotifyFocusMoved(c, roots...)
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
	if w.state.Suspended {
		return
	}
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
	if w == nil || w.focus == nil || w.state.Suspended {
		// A suspended window is not visible: no caret to blink (and no
		// wake-ups for it).
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
	w.app.stirred()
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
		if w.onCloseRequest != nil && !w.onCloseRequest() {
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
	case platform.EventWindowState:
		w.windowStateChanged(ev.State)
	case platform.EventDecorations:
		w.decorationsChanged(ev.Decor)
	case platform.EventCapabilities:
		w.capsChanged(ev.Caps)
	case platform.EventFocusOut:
		w.setAltHeld(false)
		if !w.stateKnown {
			// Without the desktop's activated state, keyboard focus is the
			// best guess at the active look.
			w.setActive(false)
		}
		w.resetIME()
		w.dismissTooltip()
		w.capture = nil
		// A press waiting to become a drag dies with the focus: the
		// button went up somewhere we will never hear about.
		w.dragArm = dragGesture{}
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
		if !w.stateKnown {
			w.setActive(true)
			w.app.refreshTitleBarPrefs()
		}
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
	case platform.EventDragMotion:
		w.dragMotion(ev)
	case platform.EventDragLeave:
		w.dragLeave()
	case platform.EventDrop:
		w.drop(ev)
	case platform.EventDragEnd:
		w.dragEnded(ev.Action, ev.Dropped)
	case platform.EventKeyDown:
		if w.dragKey(ev) {
			return
		}
		if ev.Key == platform.KeyAlt {
			w.setAltHeld(true)
		}
		if ev.Key == platform.KeyTab {
			// Ctrl+Tab goes to the widgets and the app first (a tab strip
			// switches tabs with it); plain Tab, and a Ctrl+Tab nobody
			// takes, move the focus.
			if ev.Mods.Ctrl() && w.popup == nil && (w.bubbleKey(keyEvent(ev)) || w.accelerator(ev.Key, ev.Mods)) {
				return
			}
			w.tab(!ev.Mods.Shift())
			return
		}
		if ev.Key == platform.KeyEscape {
			if w.dismissEscape() {
				return
			}
		}
		if ev.Key == platform.KeyF4 && ev.Mods.Alt() && !ev.Mods.Ctrl() && w.geom.framed {
			// The toolkit draws the frame: Alt+F4 closes, as the desktop's
			// frame would (a compositor that owns the shortcut never sends
			// it).
			w.RequestClose()
			return
		}
		if ev.Mods.Alt() {
			// While a modal overlay is up its mnemonics are the only ones:
			// Alt+F used to open the main menu above a dialog.
			if w.overlay != nil {
				if handleAlt(w.overlay, ev.Key) {
					return
				}
			} else if w.caption != nil && handleAlt(w.caption, ev.Key) || w.root != nil && handleAlt(w.root, ev.Key) {
				return
			}
		}
		if w.popup != nil {
			leaf := widget.CascadeLeaf(w.popup)
			if leaf.KeyPress(keyEvent(ev)) {
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
		if w.bubbleKey(keyEvent(ev)) {
			return
		}
		// Keys nobody took run menu accelerators (Ctrl+N, F1, Ctrl+Q …).
		w.accelerator(ev.Key, ev.Mods)
	case platform.EventKeyUp:
		if ev.Key == platform.KeyAlt {
			w.setAltHeld(false)
		}
		if t := w.keyTarget(); t != nil {
			t.KeyRelease(keyEvent(ev))
		}
	case platform.EventText:
		if t := w.keyTarget(); t != nil {
			t.TextInput(ev.Rune)
		}
	}
}

// accelerator runs the first menu accelerator for key, but not under a
// modal overlay; the title bar's come first (it is the window's top row).
// It reports whether one ran.
func (w *Window) accelerator(key platform.Key, mods platform.Modifiers) bool {
	if w.overlay != nil {
		return false
	}
	if w.caption != nil && handleAccel(w.caption, key, mods) {
		return true
	}
	return w.root != nil && handleAccel(w.root, key, mods)
}

// AltHeld reports whether the Alt key is down in the window.
func (w *Window) AltHeld() bool { return w != nil && w.altHeld }

// setAltHeld records the Alt key and, in looks that show mnemonic
// underlines only while it is held, repaints them.
func (w *Window) setAltHeld(v bool) {
	if w.altHeld == v {
		return
	}
	w.altHeld = v
	if style.LookHint(w.look, style.HintMnemonics) == style.MnemonicsOnAlt {
		// Every label with a mnemonic draws differently now: re-record,
		// since a full present only replays the retained scene.
		w.dropScene()
		w.fullInvalidate()
	}
}

// CaretBlink is the current caret pulse (tests / themed fields).
func (w *Window) CaretBlink() bool { return w.blink }

func (w *Window) bubbleWheel(ev platform.Event) {
	for c := w.hit(ev.Pos); c != nil; c = c.Parent() {
		if !c.Enabled() || !c.Visible() {
			continue
		}
		if c.MouseWheel(widget.MouseEvent{Pos: local(c, ev.Pos), Scroll: ev.Scroll, Mods: ev.Mods, Precise: ev.ScrollPrecise}) {
			// Content moved under a still pointer: re-hit-test so the row
			// (or widget) now under it becomes hot, not the one that
			// scrolled away.
			if w.capture == nil {
				w.rehover(ev.Pos, ev.Mods)
			}
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
	if w.caption != nil {
		if h := widget.HitRoot(w.caption, p); h != nil {
			if w.overlay != nil {
				// A modal dialog is up: the title bar's own controls are
				// the app's and inert like the rest; its caption buttons
				// are the window's and keep working.
				if _, ok := h.(*widgets.WindowControls); !ok {
					return nil
				}
			}
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

// keyEvent is a platform key event as the widgets see it: the key, the
// modifiers, and the character the key stands for where it stands for one,
// so a shortcut table written over letters fires. A backend that already
// put a character on the event keeps its own.
func keyEvent(ev platform.Event) widget.KeyEvent {
	r := ev.Rune
	if r == 0 {
		r = platform.KeyChar(ev.Key)
	}
	return widget.KeyEvent{Key: ev.Key, Rune: r, Mods: ev.Mods}
}

// bubbleKey offers a key to the focused widget and its ancestors and
// reports whether one of them took it.
func (w *Window) bubbleKey(e widget.KeyEvent) bool {
	start := w.keyTarget()
	if start == nil {
		return false
	}
	var top widget.Component
	for c := start; c != nil; c = c.Parent() {
		if c.KeyPress(e) {
			return true
		}
		top = c
	}
	// The content root is the window's key handler (an app's shortcuts):
	// keys from the title bar, the row above it, reach it too.
	if w.caption != nil && top == widget.Component(w.caption) && w.root != nil && w.overlay == nil {
		return w.root.KeyPress(e)
	}
	return false
}

// focusOnClick reports whether a click should focus c. Widgets that take
// keyboard focus only through Tab / mnemonics (a menu bar, a tool bar —
// Qt's Qt::TabFocus policy) implement FocusOnClick() false, so clicking
// them leaves focus with the field the user was typing in.
func focusOnClick(c widget.Component) bool {
	if f, ok := c.(interface{ FocusOnClick() bool }); ok {
		return f.FocusOnClick()
	}
	return true
}

// handleAccel runs the first menu accelerator under root that matches.
func handleAccel(root widget.Component, key platform.Key, mods platform.Modifiers) bool {
	handled := false
	widget.Walk(root, func(c widget.Component) {
		if handled {
			return
		}
		if a, ok := c.(interface {
			HandleAccelerator(platform.Key, platform.Modifiers) bool
		}); ok && a.HandleAccelerator(key, mods) {
			handled = true
		}
	})
	return handled
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
	if (w.popup == nil || widget.HitCascade(w.popup, ev.Pos) == nil) && w.frameMouseDown(ev) {
		return
	}
	t := w.hit(ev.Pos)
	w.capture = t
	// A drag is armed from the deepest component: startDragFrom walks up
	// from there on its own, so a row still starts the drag its list
	// defines whichever of the two takes the press.
	w.armDrag(t, ev.Pos)
	if t != nil && t.WantsFocus() && focusOnClick(t) {
		w.RequestFocus(t)
	}
	// A click on inert chrome (or on a widget with a tab-only focus policy)
	// keeps focus where it was.
	if t != nil {
		w.bubblePress(t, ev)
	}
}

// bubblePress offers the press to the component under the pointer and then
// to its ancestors until one takes it (widget.Component.MousePress states
// the contract). The taker becomes the capture, so the moves and the
// release follow the gesture rather than the pixel it started on; when
// nobody takes it the capture stays where it has always been, on the
// deepest component hit.
func (w *Window) bubblePress(from widget.Component, ev platform.Event) {
	for c := from; c != nil; c = c.Parent() {
		// The component under the pointer is always told, disabled or
		// not — it has always been, and widgets do their own refusing.
		// Above it, a disabled or hidden container is skipped and the
		// walk goes on, as the wheel's does.
		if c != from && (!c.Enabled() || !c.Visible()) {
			continue
		}
		lp := local(c, ev.Pos)
		if !c.MousePress(widget.MouseEvent{Pos: lp, Button: ev.Button, Mods: ev.Mods}) {
			continue
		}
		// Only move a capture nobody touched: a handler that handed the
		// pointer to the desktop (StartMove, StartResize, the window
		// menu) has already let go of it, and must not get it back.
		if c != from && w.capture == from {
			w.capture = c
		}
		w.syncCursor(c, lp)
		return
	}
	w.syncCursor(from, local(from, ev.Pos))
}

func (w *Window) mouseUp(ev platform.Event) {
	if w.frameMouseUp() {
		return
	}
	if w.dragPointerUp(ev) {
		return
	}
	t := w.capture
	if t == nil {
		t = w.hit(ev.Pos)
	}
	if t != nil {
		lp := local(t, ev.Pos)
		t.MouseRelease(widget.MouseEvent{Pos: lp, Button: ev.Button, Mods: ev.Mods})
	}
	captured := w.capture != nil
	w.capture = nil
	if captured {
		// A drag that ended over another widget hands hover to it (the
		// pressed button used to stay hot until the pointer moved again).
		w.rehover(ev.Pos, ev.Mods)
	}
	hit := w.hit(ev.Pos)
	if hit != nil {
		w.syncCursor(hit, local(hit, ev.Pos))
	} else {
		w.SetCursor(platform.CursorDefault)
	}
}

// rehover re-hit-tests hover at pos after the content moved under a still
// pointer (wheel scroll) or a capture ended. Unlike real motion it never
// arms a tooltip.
func (w *Window) rehover(pos paintengine2d.Point, mods platform.Modifiers) {
	t := w.hit(pos)
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
		lp := local(t, pos)
		t.MouseMove(widget.MouseEvent{Pos: lp, Mods: mods})
		w.syncCursor(t, lp)
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
	if w.frameMouseMove(ev) {
		return
	}
	if w.dragPointerMove(ev) {
		return
	}
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
	var list []widget.Component
	switch {
	case w.popup != nil:
		list = widget.Focusables(w.popup)
	case w.overlay != nil:
		list = widget.Focusables(w.overlay)
	default:
		// The title bar's controls come first: it is the window's top row.
		if w.caption != nil {
			list = widget.Focusables(w.caption)
		}
		list = append(list, widget.Focusables(w.root)...)
	}
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
	widget.RevealFocus(list[idx])
}

func (w *Window) layout() {
	// The margin is part of the surface: ask for it first, so this layout
	// (and the frame painted from it) already has the buffer it needs.
	w.applyFrame()
	ww, hh := w.surf.Size()
	box := paintengine2d.XYWH(0, 0, float32(ww), float32(hh))
	g := w.layoutFrame(box)
	if w.root != nil {
		_ = w.root.Measure(layout.Tight(g.content.Dx(), g.content.Dy()))
		w.root.Arrange(g.content)
	}
	if w.overlay != nil {
		// A modal dialog dims the content; in a frame the toolkit draws,
		// the caption stays out of it (the window can still be moved,
		// minimized and closed, as with a macOS sheet).
		ob := box
		if g.framed {
			ob = g.content
		}
		_ = w.overlay.Measure(layout.Tight(ob.Dx(), ob.Dy()))
		w.overlay.Arrange(ob)
	}
	if w.statusMenu && w.popup != nil {
		w.popup.Arrange(box)
	}
	// A relayout is when subtrees appear and disappear, so this is where a
	// component that was removed without going through a layer swap stops
	// being the focus / hover / capture target.
	w.dropDeadRefs()
	w.focusOnOpen()
	w.laid = true
}

// focusOnOpen gives the keyboard somewhere to go the first time the window
// is laid out with content in it.
//
// A window used to open with nothing focused, so every key that bubbles
// from the focus — which is every shortcut a widget or a container
// defines — reached nobody at all until the user clicked or pressed Tab.
// Qt and GTK both focus the first widget in the tab chain when a window is
// shown, and this is that.
//
// It never takes focus from an app that placed it itself: a window whose
// content called RequestFocus before its first frame keeps that focus, and
// this runs once, so a later Escape or content swap that leaves the window
// unfocused is left alone. SetInitialFocus names the component to start on
// where the first one in the tab order is the wrong one.
//
// Where the focus lands, it lands the way a click's does rather than a
// Tab's: the ring a look shows only after keyboard navigation stays hidden
// until the user actually uses the keyboard (GTK's :focus-visible), while
// a field that always shows its caret shows it, which is what a dialog
// that opens on a text field should do.
func (w *Window) focusOnOpen() {
	if w == nil || w.openFocused || w.statusMenu || w.opts.Popup {
		return
	}
	if w.root == nil && w.overlay == nil {
		// Nothing to focus yet: an app that calls SetContent after the
		// first frame still gets its turn.
		return
	}
	w.openFocused = true
	if w.focus != nil {
		return
	}
	target := w.initialFocus
	if target == nil {
		// The caption is deliberately not searched: a window that opened
		// with its own close button focused would be absurd. The content
		// is the app, and a modal overlay is the app while it is up.
		scope := w.root
		if w.overlay != nil {
			scope = w.overlay
		}
		target = firstOpenFocus(scope)
	}
	if target == nil {
		return
	}
	w.RequestFocus(target)
	widget.MarkPointerFocus(target)
}

// firstOpenFocus is the component a freshly opened window starts on: the
// first in the tab order that a *click* would also focus.
//
// Chrome that takes focus only from the keyboard — a menu bar, a tool bar,
// a tab strip, everything with FocusOnClick false (Qt's Qt::TabFocus) — is
// reached with Tab, F10 or a mnemonic and is never where a window starts,
// so a window that holds nothing else opens with no focus at all. Its menu
// accelerators and Alt mnemonics work either way: neither goes through the
// focus.
func firstOpenFocus(root widget.Component) widget.Component {
	for _, c := range widget.Focusables(root) {
		if focusOnClick(c) {
			return c
		}
	}
	return nil
}

// SetInitialFocus names the component the window focuses when it first
// opens, instead of the first one in the tab order. It must be in the
// window's content (or its overlay) by the time the first frame is laid
// out; nil restores the default.
//
// An app that places focus itself — RequestFocus before the first frame —
// does not need this: the window only chooses when nothing else has.
func (w *Window) SetInitialFocus(c widget.Component) {
	if w == nil {
		return
	}
	w.initialFocus = c
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
	clip := paintengine2d.XYWH(0, 0, float32(ww), float32(hh))
	if g := w.geom; g.framed && !g.margin.Zero() {
		// The margin holds the shadow and nothing else: it changes only
		// with the window's size or state, both of which repaint in full.
		// A hover or a caret never touches it.
		clip = clip.Intersect(g.window)
	}
	if r := w.shapeRaster(); r != nil {
		// Damage stays inside the silhouette. Its bounding box is as far
		// as this goes on purpose: clipping each damage box against a
		// silhouette's own rectangles would turn one box into as many as
		// the shape has rows, and presenting hundreds of slivers costs the
		// compositor far more than repainting a corner the window does not
		// own — whose pixels are punched transparent anyway.
		win := w.windowBox()
		b := r.Bounds
		clip = clip.Intersect(paintengine2d.XYWH(
			win.Min.X+float32(b.X), win.Min.Y+float32(b.Y), float32(b.W), float32(b.H)))
	}
	out.ClipTo(clip)
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
		// A backdrop blur can widen what is repainted.
		rects = w.frameScene(rects)
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
	if w.caption != nil {
		w.paintDecoration(ctx)
		widget.PaintTree(w.caption, ctx, dirty)
	}
	if w.root != nil {
		widget.PaintTree(w.root, ctx, dirty)
	}
	if w.overlay != nil {
		widget.PaintTree(w.overlay, ctx, dirty)
	}
	if w.popup != nil {
		widget.WalkCascade(w.popup, func(c widget.Component) {
			w.paintShadow(ctx, c, style.PopupMenu, dirty)
			widget.PaintTree(c, ctx, dirty)
		})
	}
	if w.tooltip != nil {
		w.paintShadow(ctx, w.tooltip, style.PopupTooltip, dirty)
		widget.PaintTree(w.tooltip, ctx, dirty)
	}
}

// layerLook is the look a floating layer paints with: its own (inherited
// from the widget that opened it) or the window's.
func (w *Window) layerLook(c widget.Component) style.LookAndFeel {
	if l, ok := c.(interface{ Look() style.LookAndFeel }); ok {
		if lk := l.Look(); lk != nil {
			return lk
		}
	}
	return w.look
}

// paintShadow drops the look's shadow under a floating layer, just before
// the layer paints over it. dirty (device pixels) skips it when no dirty box
// reaches the shadow.
func (w *Window) paintShadow(ctx *paintengine2d.Context, c widget.Component, kind style.PopupKind, dirty *paintengine2d.Damage) {
	if c == nil || !c.Visible() {
		return
	}
	lk := w.layerLook(c)
	out := style.PopupShadowOf(lk, kind)
	if out.Zero() {
		return
	}
	b := c.Bounds()
	reach := out.Grow(b)
	if dirty != nil && !dirty.Empty() && !dirty.Overlaps(ctx.Matrix().TransformRect(reach)) {
		return
	}
	ctx.Save()
	ctx.ClipRect(reach)
	style.DrawPopupShadowOf(lk, ctx, b, kind)
	ctx.Restore()
}

// paintBackground lets the look paint the window background (Aqua
// pinstripes, brushed metal); the flat clear already covers the rest.
func (w *Window) paintBackground(ctx *paintengine2d.Context, full paintengine2d.Rect) {
	if w.wantsGlass() {
		// The window is a pane over the blurred desktop: the look's own
		// background is opaque — a gradient, a texture, a flat fill — and
		// painting it here would hide every bit of the blur. The glass
		// tint windowFill already laid down is the background now.
		return
	}
	if bl, ok := w.look.(style.WindowBackgroundLook); ok {
		bl.DrawWindowBackground(ctx, full)
	}
}

// windowBox is the visible window inside the surface: the whole surface
// unless a frame of ours keeps a margin for its shadow.
func (w *Window) windowBox() paintengine2d.Rect {
	ww, hh := w.surf.Size()
	full := paintengine2d.XYWH(0, 0, float32(ww), float32(hh))
	if g := w.geom; g.framed && !g.margin.Zero() {
		return full.Intersect(g.window)
	}
	return full
}

// clearColor is what a frame starts from: the look's background, or
// nothing at all where the margin around the window must stay see-through.
func (w *Window) clearColor() paintengine2d.Color {
	if w.seeThrough() {
		return paintengine2d.Transparent
	}
	return w.look.Palette().Background
}

// seeThrough reports whether the surface starts fully transparent: there is
// a margin to keep clear for the shadow, a corner to cut, a silhouette
// whose outside is not the window, or glass to let the desktop through.
func (w *Window) seeThrough() bool {
	return w.geom.translucent() || w.shapeRaster() != nil || w.wantsGlass()
}

// windowFill is the colour the window's own box starts from: the look's
// background, or the look's glass tint where the desktop is really being
// blurred behind it (an opaque fill over a blurred desktop would show none
// of the blur).
func (w *Window) windowFill() paintengine2d.Color {
	if !w.wantsGlass() {
		return w.look.Palette().Background
	}
	if w.glassTint.A > 0 {
		return w.glassTint
	}
	return style.GlassTint(w.look, glassAlpha)
}

// glassAlpha is how much of the blurred desktop shows through a window
// whose look did not name a glass colour of its own. It sits where Fluent's
// acrylic, Big Sur's vibrancy and Tahoe's glass each sit.
const glassAlpha = 0.78

// fillWindow paints box with the window's fill. An opaque fill is drawn as
// it always was; a translucent one has to replace what is under it rather
// than blend over it, or every frame would stack another coat of tint on
// the last.
func (w *Window) fillWindow(ctx *paintengine2d.Context, box paintengine2d.Rect, fill paintengine2d.Color) {
	if fill.A < 1 {
		// A translucent fill has to replace what is under it, not blend
		// over it, or every frame would stack another coat of tint on the
		// last: take the box down to nothing first.
		ctx.DrawRect(box, paintengine2d.Paint{
			Color: paintengine2d.White,
			Blend: paintengine2d.BlendDestOut,
		})
	}
	ctx.DrawRect(box, paintengine2d.Fill(fill))
}

func (w *Window) frameImmediate(rects []paintengine2d.Rect) {
	ctx := platform.NewPaintContext(w.surf)
	if ctx == nil {
		return
	}
	bg := w.windowFill()
	win := w.windowBox()
	if len(rects) == 0 {
		ctx.Clear(w.clearColor())
		ctx.Save()
		ctx.ClipRect(win)
		if w.seeThrough() {
			w.fillWindow(ctx, win, bg)
		}
		w.paintBackground(ctx, win)
		w.paintLayers(ctx, nil)
		ctx.Restore()
		w.paintFrameCorners(ctx, nil)
		w.paintWindowShape(ctx, nil)
		w.paintFrameShadow(ctx, nil)
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
		ctx.ClipRect(win)
		w.fillWindow(ctx, r, bg)
		w.paintBackground(ctx, win)
		w.paintLayers(ctx, &one)
		ctx.Restore()
		// A box that reaches a corner, or any part of the silhouette's
		// edge, takes its bite out again.
		ctx.Save()
		ctx.ClipDeviceRect(r)
		w.paintFrameCorners(ctx, &one)
		w.paintWindowShape(ctx, &one)
		ctx.Restore()
	}
}

func (w *Window) frameScene(rects []paintengine2d.Rect) []paintengine2d.Rect {
	ww, hh := w.surf.Size()
	rec := paintengine2d.NewRecorder(ww, hh)
	if w.paths == nil {
		w.paths = paintengine2d.NewPathCache()
	}
	rec.UsePathCache(w.paths)
	defer w.paths.EndFrame()
	rec.Clear(w.clearColor())
	ctx := paintengine2d.NewContextDevice(rec)
	win := w.windowBox()
	ctx.Save()
	ctx.ClipRect(win)
	if w.seeThrough() {
		// The clear left the whole surface see-through for the shadow (or
		// for the desktop behind a shaped or glassy window); the window
		// itself starts from the look's background again.
		w.fillWindow(ctx, win, w.windowFill())
	}
	w.paintBackground(ctx, win)
	// The recording is always a complete display list for the window;
	// partial redraw happens at replay, where the engine clips every op to
	// the dirty box. Recording a subset would make the next partial replay
	// paint from a scene that never had the clean widgets in it.
	if w.caption != nil {
		w.paintDecoration(ctx)
		widget.RecordTree(w.caption, rec, ctx, nil, w.layers, false)
	}
	if w.root != nil {
		widget.RecordTree(w.root, rec, ctx, nil, w.layers, false)
	}
	if w.overlay != nil {
		widget.RecordTree(w.overlay, rec, ctx, nil, w.layers, true)
	}
	if w.popup != nil {
		widget.WalkCascade(w.popup, func(c widget.Component) {
			w.paintShadow(ctx, c, style.PopupMenu, nil)
			widget.RecordTree(c, rec, ctx, nil, w.layers, true)
		})
	}
	if w.tooltip != nil {
		w.paintShadow(ctx, w.tooltip, style.PopupTooltip, nil)
		widget.RecordTree(w.tooltip, rec, ctx, nil, w.layers, true)
	}
	ctx.Restore()
	// Last, outside the window's clip: the corners (or the whole
	// silhouette) are cut out of what was painted and the shadow goes into
	// the margin around it.
	w.paintFrameCorners(ctx, nil)
	w.paintWindowShape(ctx, nil)
	w.paintFrameShadow(ctx, nil)
	w.layers.EndFrame()
	w.scene = rec.Finish()
	dev := platform.SurfaceDevice(w.surf)
	if dev == nil {
		return rects
	}
	if len(rects) == 0 {
		paintengine2d.DrawScene(w.scene, dev)
		return rects
	}
	dmg := paintengine2d.Damage{Rects: rects}
	paintengine2d.DrawSceneDamage(w.scene, dev, &dmg)
	return dmg.Rects
}

// Scene is the last retained graph (tests / inspector).
func (w *Window) Scene() *paintengine2d.Scene { return w.scene }

// Capture paints a full frame and returns a clone of the pixmap — the
// visible window only: the invisible margin a frame keeps for its shadow is
// cropped away, so a screenshot is the window the user sees, whatever the
// look does outside it.
func (w *Window) Capture() *paintengine2d.Image {
	img := w.CaptureSurface()
	if img == nil {
		return nil
	}
	g := w.geom
	if r := w.shapeRaster(); r != nil {
		return w.cropToShape(img, r)
	}
	if !g.framed || g.margin.Zero() {
		return img
	}
	x0, y0, x1, y1 := g.window.IntBounds()
	return img.SubImage(x0, y0, x1, y1)
}

// cropToShape is Capture's crop for a shaped window: the silhouette's
// bounding box rather than the whole window (a shape that does not fill its
// window would otherwise come back with a transparent margin), with the
// coverage mask applied to the alpha channel so every pixel outside the
// silhouette is see-through whatever the paint path left there.
func (w *Window) cropToShape(img *paintengine2d.Image, r *platform.ShapeRaster) *paintengine2d.Image {
	win := w.windowBox()
	ox, oy := int(win.Min.X), int(win.Min.Y)
	b := r.Bounds
	out := img.SubImage(ox+b.X, oy+b.Y, ox+b.X+b.W, oy+b.Y+b.H)
	if out == nil || out.Width < 1 || out.Height < 1 {
		return out
	}
	stride := out.RowStride()
	for y := 0; y < out.Height; y++ {
		row := out.Pix[y*stride:]
		src := r.Mask[(y+b.Y)*r.W:]
		for x := 0; x < out.Width; x++ {
			c := uint32(src[x+b.X])
			if c == 255 {
				continue
			}
			// Premultiplied: every channel scales with the coverage.
			i := x * 4
			for k := 0; k < 4; k++ {
				row[i+k] = uint8((uint32(row[i+k])*c + 127) / 255)
			}
		}
	}
	return out
}

// CaptureSurface paints a full frame and returns the whole buffer, margin
// and all: the shadow and the see-through corners as the compositor gets
// them (the frames sheet and the frame tests look at these).
func (w *Window) CaptureSurface() *paintengine2d.Image {
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
	w.app.owesTrim()
	if w.app != nil && w.app.statusMenu == w {
		w.app.statusMenu = nil
	}
	w.popup = nil
	w.overlay = nil
	w.root = nil
	w.caption = nil
	w.titleBar = nil
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

// SetOnCloseRequest hands the desktop's close button to the app: fn runs
// before the window would close and returning false keeps it open (a
// document with unsaved work, a floating dock panel that hides itself
// instead). Alt+F4 and the window menu's Close go through it too, since
// they ask the same way. nil restores the default.
func (w *Window) SetOnCloseRequest(fn func() bool) { w.onCloseRequest = fn }

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

// PortalParent names the window as the parent of a desktop portal dialog
// ("x11:<id>"; empty where the platform cannot say).
func (w *Window) PortalParent() string {
	if p, ok := w.surf.(platform.PortalParenter); ok {
		return p.PortalParent()
	}
	return ""
}
