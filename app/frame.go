package app

import (
	"math"
	"os"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The window frame layer: who draws the frame (the desktop, the toolkit or
// nobody), the title bar an app puts in it (SetTitleBar), the frame's
// geometry and its hit test (resize edges, caption, caption buttons,
// client), and the caption gestures that hand a move, a resize or the
// window menu to the desktop. The look paints the frame (style.DecorationOf
// and friends: Win95's gradient caption, Aqua's traffic lights, a hairline
// in plain looks). Frames are opaque and square for now: the window's
// geometry is the whole surface and the resize handles sit inside its edges
// (the border, at least 4 px, and the top of the caption).

// NonClientRegion is what a point of a window is to the window system, the
// region kinds every platform understands (Win32's WM_NCHITTEST codes,
// Windows App SDK's NonClientRegionKind).
type NonClientRegion uint8

const (
	// RegionClient is the app's: widgets handle it.
	RegionClient NonClientRegion = iota
	// RegionCaption is title-bar space: a press moves the window.
	RegionCaption
	// RegionResize is a resize edge or corner (see the Edges returned).
	RegionResize
	// RegionClose / Minimize / Maximize / Menu are the caption buttons;
	// widgets handle them, like client space.
	RegionClose
	RegionMinimize
	RegionMaximize
	RegionMenu
)

func (r NonClientRegion) String() string {
	return [...]string{"client", "caption", "resize", "close", "minimize", "maximize", "menu"}[r]
}

// frameGeom is the window's frame layout, in device pixels.
type frameGeom struct {
	// framed: the toolkit draws the frame (caption buttons, border,
	// resize edges).
	framed bool
	// border is the look's frame around a framed window (none when
	// maximized).
	border style.Insets
	// caption is the title bar's box (empty without one); content the
	// box of the content and, in a framed window, of the overlay.
	caption paintengine2d.Rect
	content paintengine2d.Rect
}

// captionGesture is a left press on caption space, waiting to become a move
// (past the drag threshold) or a click.
type captionGesture struct {
	armed bool
	pos   paintengine2d.Point
}

// captionClick is the last caption click, for double-click detection: a
// double click counts only when both presses hit the caption.
type captionClick struct {
	ok  bool
	at  time.Time
	pos paintengine2d.Point
}

// SetTitleBar puts c in the window's title bar: the caption of a frame the
// toolkit draws, with the caption buttons at the sides the desktop puts
// them, or — under the desktop's frame — the first row of the window. A
// widgets.HeaderBar is used as it is; any other component becomes the
// centre of one. Its caption space (widget.CaptionHitTester) moves the
// window, double-clicking it runs the desktop's double-click action and
// right-clicking shows the window menu. nil restores the default: the
// desktop's frame, or a caption with the window title where the toolkit
// draws the frame. GTK's gtk_window_set_titlebar.
//
// With the default decoration policy a window with a title bar gets the
// toolkit's frame where the desktop can move and resize it on request
// (Wayland, X11 stacking window managers); the user's "Use system title bar
// and borders" preference, WindowOptions.Decorations and UITK_DECORATIONS
// override it.
func (w *Window) SetTitleBar(c widget.Component) {
	if w == nil || w.titleBar == c {
		return
	}
	w.titleBar = c
	w.wrapCaption = nil
	w.syncDecorations()
	w.dropDeadRefs()
}

// TitleBar is the component given to SetTitleBar (nil for none).
func (w *Window) TitleBar() widget.Component { return w.titleBar }

// Caption is the header bar the window lays out on top: the title bar, the
// header bar wrapping it, or the default caption of a toolkit-drawn frame
// (nil when there is none).
func (w *Window) Caption() *widgets.HeaderBar { return w.caption }

// Decorations is the decoration mode in effect: Client when the toolkit
// draws the frame, Server when the desktop does, None for no frame.
func (w *Window) Decorations() platform.Decorations { return w.decor }

// FrameCaps is what the desktop can do for the window (widget.FrameHost).
func (w *Window) FrameCaps() platform.WMCaps { return w.caps }

// offscreen reports whether the application paints without a display
// (tests, screenshots): no frame is chosen for it unless asked explicitly.
func (a *Application) offscreen() bool {
	return a == nil || a.headless || a.backend == nil || a.backend.Name() == "offscreen"
}

// resolveDecorations is the decoration policy, first match wins:
// UITK_DECORATIONS; WindowOptions.Decorations; no frame of the toolkit's
// for an offscreen window; the user's preference (look.json "decorations");
// the toolkit's frame for a window with a title bar where the desktop can
// move and resize it on request; the desktop's frame otherwise.
func (a *Application) resolveDecorations(opts platform.WindowOptions, titleBar bool, surf platform.Surface) platform.Decorations {
	if opts.Popup {
		return platform.DecorationsNone
	}
	if d, ok := platform.DecorationsFromEnv(); ok {
		return d
	}
	if opts.Decorations != platform.DecorationsAuto {
		return opts.Decorations
	}
	if a.offscreen() || opts.Headless {
		return platform.DecorationsServer
	}
	switch a.decorPref {
	case style.DecorationsSystem:
		return platform.DecorationsServer
	case style.DecorationsToolkit:
		return platform.DecorationsClient
	}
	if titleBar {
		if f, ok := surf.(platform.FrameSurface); ok && f.SuitsClientFrame() {
			return platform.DecorationsClient
		}
	}
	return platform.DecorationsServer
}

// syncDecorations asks the surface for the mode the policy picks now and
// rebuilds the caption for the mode in effect.
func (w *Window) syncDecorations() {
	want := w.app.resolveDecorations(w.opts, w.titleBar != nil, w.surf)
	if f, ok := w.surf.(platform.FrameSurface); ok {
		f.RequestDecorations(want)
		w.decor = f.Decorations()
		w.caps = f.Capabilities()
	} else {
		w.decor = platform.DecorationsServer
	}
	w.rebuildCaption()
}

// framed reports whether the toolkit draws the window's frame now (a full
// screen window has none).
func (w *Window) framed() bool {
	return w.decor == platform.DecorationsClient && !w.state.Fullscreen
}

// rebuildCaption picks the header bar the window lays out on top and tells
// it about the frame (caption buttons in the desktop's layout, or none).
func (w *Window) rebuildCaption() {
	framed := w.framed()
	var hb *widgets.HeaderBar
	switch tb := w.titleBar.(type) {
	case nil:
		if framed {
			if w.defaultCaption == nil {
				w.defaultCaption = widgets.NewHeaderBar(nil, nil, nil)
				w.defaultCaption.ShowTitle = true
			}
			hb = w.defaultCaption
		}
	case *widgets.HeaderBar:
		hb = tb
	default:
		if w.wrapCaption == nil || w.wrapCaption.Center() != tb {
			w.wrapCaption = widgets.NewHeaderBar(nil, tb, nil)
		}
		hb = w.wrapCaption
	}
	if hb != nil {
		hb.SetHost(w)
		hb.SetWindowControls(w.app.TitleBarPrefs().Layout, framed)
	}
	if hb != w.caption {
		old := w.caption
		w.caption = hb
		if old != nil && old != hb {
			old.SetHost(nil)
		}
		w.capPress, w.capClick = captionGesture{}, captionClick{}
	}
	w.laid = false
	w.dropScene()
	w.fullInvalidate()
}

// decorationsChanged adopts the mode the desktop answered.
func (w *Window) decorationsChanged(d platform.Decorations) {
	if d == w.decor {
		return
	}
	w.decor = d
	w.rebuildCaption()
}

// capsChanged adopts what the desktop can do now (caption buttons follow).
func (w *Window) capsChanged(c platform.WMCaps) {
	if c == w.caps {
		return
	}
	w.caps = c
	w.rebuildCaption()
}

// frameBorder is the look's border around a framed window, whole device
// pixels (none when maximized).
func (w *Window) frameBorder() style.Insets {
	if w.caption == nil || w.state.Maximized {
		return style.Insets{}
	}
	b := style.DecorationOf(w.look, w.caption.DecorationState()).Border
	px := func(v float32) float32 { return max(float32(math.Round(float64(v))), 0) }
	return style.Insets{Top: px(b.Top), Right: px(b.Right), Bottom: px(b.Bottom), Left: px(b.Left)}
}

// layoutFrame arranges the caption inside full and returns the geometry.
func (w *Window) layoutFrame(full paintengine2d.Rect) frameGeom {
	g := frameGeom{content: full}
	if w.caption == nil {
		w.geom = g
		return g
	}
	g.framed = w.framed()
	inner := full
	if g.framed {
		g.border = w.frameBorder()
		inner = g.border.Apply(full)
	}
	sz := w.caption.Measure(layout.Loose(inner.Dx(), inner.Dy()))
	h := min(float32(math.Ceil(float64(sz.Y)-1e-3)), inner.Dy())
	g.caption = paintengine2d.XYWH(inner.Min.X, inner.Min.Y, inner.Dx(), h)
	g.content = paintengine2d.XYWH(inner.Min.X, inner.Min.Y+h, inner.Dx(), max(0, inner.Dy()-h))
	w.caption.Arrange(g.caption)
	w.geom = g
	return g
}

// NonClientHit is what window point p (device pixels) is to the window
// system — a resize edge, caption, a caption button or the app's own space —
// from the last layout: the resize band, then the caption buttons, then
// caption. (Chromium tests its buttons first, but they sit below its top
// border; these run to the frame's edge, so the band keeps the outer
// pixels of a restored window. Maximized there is no band, and the buttons
// reach the screen's edge.) A future Win32 backend answers WM_NCHITTEST with
// it.
func (w *Window) NonClientHit(p paintengine2d.Point) (NonClientRegion, platform.Edges) {
	if w == nil || w.caption == nil || w.state.Fullscreen {
		return RegionClient, 0
	}
	if w.popup != nil && widget.HitCascade(w.popup, p) != nil {
		return RegionClient, 0
	}
	if w.geom.framed {
		if e := w.resizeEdgesAt(p); e != 0 {
			return RegionResize, e
		}
		switch w.captionButtonAt(p) {
		case platform.CaptionClose:
			return RegionClose, 0
		case platform.CaptionMinimize:
			return RegionMinimize, 0
		case platform.CaptionMaximize:
			return RegionMaximize, 0
		case platform.CaptionMenu:
			return RegionMenu, 0
		}
	}
	if w.geom.caption.Contains(p) {
		if hit := widget.HitRoot(w.caption, p); hit != nil && widget.IsCaption(hit, w.caption, p) {
			return RegionCaption, 0
		}
	}
	return RegionClient, 0
}

// captionButtonAt is the caption button under p, if any.
func (w *Window) captionButtonAt(p paintengine2d.Point) platform.CaptionButton {
	left, right := w.caption.Controls()
	for _, c := range []*widgets.WindowControls{left, right} {
		if c == nil || !c.Visible() {
			continue
		}
		o := widget.DeviceOrigin(c)
		if b := c.ButtonAt(paintengine2d.Pt(p.X-o.X, p.Y-o.Y)); b != platform.CaptionNone {
			return b
		}
	}
	return platform.CaptionNone
}

// resizeEdgesAt is the resize edge under p: a 4 px band inside the left,
// right and bottom edges and the top 4 px of the caption, reaching 16 px
// along each edge from a corner for the corners; none while maximized or
// on a tiled or constrained edge.
func (w *Window) resizeEdgesAt(p paintengine2d.Point) platform.Edges {
	st := w.state
	if !w.geom.framed || st.Maximized || st.Fullscreen {
		return 0
	}
	ww, hh := w.surf.Size()
	W, H := float32(ww), float32(hh)
	bd := w.geom.border
	band := max(bd.Left, bd.Right, bd.Top, bd.Bottom, float32(math.Round(float64(style.Dip(w.look, 4)))))
	corner := float32(math.Round(float64(style.Dip(w.look, 16))))
	if p.X < 0 || p.Y < 0 || p.X >= W || p.Y >= H {
		return 0
	}
	var e platform.Edges
	switch {
	case p.X < band:
		e |= platform.EdgeLeft
	case p.X >= W-band:
		e |= platform.EdgeRight
	}
	switch {
	case p.Y < band:
		e |= platform.EdgeTop
	case p.Y >= H-band:
		e |= platform.EdgeBottom
	}
	if e == platform.EdgeTop || e == platform.EdgeBottom {
		if p.X < corner {
			e |= platform.EdgeLeft
		} else if p.X >= W-corner {
			e |= platform.EdgeRight
		}
	}
	if e == platform.EdgeLeft || e == platform.EdgeRight {
		if p.Y < corner {
			e |= platform.EdgeTop
		} else if p.Y >= H-corner {
			e |= platform.EdgeBottom
		}
	}
	// A tiled edge touches a screen edge or a neighbour: it does not resize.
	e &^= st.Tiled | st.Constrained
	if !e.Valid() {
		return 0
	}
	return e
}

// dragThreshold is how far (device px) a caption press moves before it
// becomes a window move.
func (w *Window) dragThreshold() float32 {
	t := w.app.TitleBarPrefs().DragThreshold
	if t <= 0 {
		t = 8
	}
	return t * max(w.scale, 1)
}

func near(a, b paintengine2d.Point, d float32) bool {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx+dy*dy <= d*d
}

// frameMouseDown handles a press on the frame (resize band, caption) and
// reports whether it did; caption buttons and the app's own space go on to
// the widgets.
func (w *Window) frameMouseDown(ev platform.Event) bool {
	if w.caption == nil {
		return false
	}
	r, edges := w.NonClientHit(ev.Pos)
	switch r {
	case RegionResize:
		w.dismissOpenPopup()
		switch ev.Button {
		case platform.ButtonLeft:
			// Resizes start on the press (only moves wait for a drag).
			w.StartResize(edges)
		case platform.ButtonRight:
			w.capture = w.caption
			w.ShowWindowMenu(ev.Pos)
		}
		return true
	case RegionCaption:
		if w.overlay != nil && !w.geom.framed {
			// Under the desktop's frame the title bar is the first content
			// row: a modal dialog covers it like the rest.
			return false
		}
		w.dismissOpenPopup()
		w.captionPress(ev)
		return true
	}
	// The app's own space: a later caption press is no double click.
	w.capClick = captionClick{}
	return false
}

// dismissOpenPopup takes a popup down before a frame gesture.
func (w *Window) dismissOpenPopup() {
	if w.popup != nil {
		w.DismissPopup()
	}
}

// captionPress runs a press on caption space: a left press arms a move
// (started once the pointer passes the drag threshold, so clicks and double
// clicks still work) or completes a double click; middle and right presses
// run the desktop's title-bar actions.
func (w *Window) captionPress(ev platform.Event) {
	// The caption holds the pointer until the release, so a menu opened
	// under it on the press does not take the release as a pick.
	w.capture = w.caption
	p := w.app.TitleBarPrefs()
	switch ev.Button {
	case platform.ButtonLeft:
		now := w.now()
		if w.capClick.ok && now.Sub(w.capClick.at) <= p.DoubleClickTime && near(ev.Pos, w.capClick.pos, w.dragThreshold()) {
			w.capClick, w.capPress = captionClick{}, captionGesture{}
			w.runTitleAction(p.DoubleClick, ev.Pos)
			return
		}
		w.capClick = captionClick{ok: true, at: now, pos: ev.Pos}
		w.capPress = captionGesture{armed: true, pos: ev.Pos}
	case platform.ButtonMiddle:
		w.capClick = captionClick{}
		w.runTitleAction(p.MiddleClick, ev.Pos)
	case platform.ButtonRight:
		w.capClick = captionClick{}
		// The component's own menu (a tab strip's), then the header bar's,
		// then the desktop's action.
		for c := widget.HitRoot(w.caption, ev.Pos); c != nil; c = c.Parent() {
			if m, ok := c.(widget.CaptionMenuer); ok && m.CaptionMenu(ev.Pos) {
				return
			}
			if c == widget.Component(w.caption) {
				break
			}
		}
		if w.caption.OnContextMenu != nil && w.caption.OnContextMenu(ev.Pos) {
			return
		}
		w.runTitleAction(p.RightClick, ev.Pos)
	}
}

// frameMouseMove follows the pointer over the frame: an armed caption press
// becomes a move past the drag threshold; over the resize band the pointer
// takes the edge's resize shape. It reports whether it handled ev.
func (w *Window) frameMouseMove(ev platform.Event) bool {
	if w.capPress.armed {
		if !near(ev.Pos, w.capPress.pos, w.dragThreshold()) {
			w.capPress.armed = false
			// A drag is no click: it does not start a double click.
			w.capClick = captionClick{}
			w.StartMove()
		}
		return true
	}
	if w.caption == nil || w.capture != nil {
		return false
	}
	if r, e := w.NonClientHit(ev.Pos); r == RegionResize {
		if w.hover != nil {
			w.hover.MouseExit()
			w.hover = nil
		}
		w.HideTooltip()
		w.tipHover = nil
		w.SetCursor(e.ResizeCursor())
		return true
	}
	return false
}

// frameMouseUp ends a caption press that never became a move: a click.
func (w *Window) frameMouseUp() bool {
	if !w.capPress.armed {
		return false
	}
	w.capPress.armed = false
	w.capture = nil
	return true
}

// StartMove hands the pointer press being handled to the desktop for an
// interactive move (from a press handler, like Avalonia's BeginMoveDrag or
// Qt's startSystemMove); false when the desktop cannot. The window lets go of
// the pointer: the release goes to the desktop.
func (w *Window) StartMove() bool {
	f, ok := w.surf.(platform.FrameSurface)
	if !ok || !f.StartSystemMove() {
		return false
	}
	w.releasePointer()
	return true
}

// StartResize is StartMove for an interactive resize from edges.
func (w *Window) StartResize(edges platform.Edges) bool {
	f, ok := w.surf.(platform.FrameSurface)
	if !ok || !f.StartSystemResize(edges) {
		return false
	}
	w.releasePointer()
	return true
}

// releasePointer forgets the press, capture and hover once the desktop has
// taken the pointer over: the release (and the motion) go to the desktop
// until the gesture ends.
func (w *Window) releasePointer() {
	w.capPress = captionGesture{}
	w.capture = nil
	if w.hover != nil {
		w.hover.MouseExit()
		w.hover = nil
	}
	w.dismissTooltip()
	w.SetCursor(platform.CursorDefault)
}

// ShowWindowMenu shows the desktop's window menu at p (window device
// pixels), or the toolkit's own (Restore / Maximize, Minimize, Close) where
// the desktop has none.
func (w *Window) ShowWindowMenu(p paintengine2d.Point) {
	if w == nil || w.Closed() {
		return
	}
	if f, ok := w.surf.(platform.FrameSurface); ok && f.ShowWindowMenu(p) {
		w.releasePointer()
		return
	}
	w.showToolkitWindowMenu(p)
}

// showToolkitWindowMenu is the window menu for desktops without one.
func (w *Window) showToolkitWindowMenu(p paintengine2d.Point) *widgets.PopupMenu {
	anchor := w.captionLayer()
	if anchor == nil {
		anchor = w.root
	}
	if anchor == nil {
		return nil
	}
	caps := w.caps
	var items []*widgets.MenuItem
	switch {
	case w.state.Maximized:
		items = append(items, widgets.Item("&Restore", w.ToggleMaximize))
	case caps.Can(platform.CapMaximize):
		items = append(items, widgets.Item("Ma&ximize", w.ToggleMaximize))
	}
	if caps.Can(platform.CapMinimize) {
		items = append(items, widgets.Item("Mi&nimize", w.Minimize))
	}
	if len(items) > 0 {
		items = append(items, widgets.Sep())
	}
	items = append(items, widgets.ItemAccel("&Close", "Alt+F4", w.RequestClose))
	return widgets.ShowContextMenu(anchor, p, items...)
}

// runTitleAction runs a title-bar click action at p.
func (w *Window) runTitleAction(a platform.TitleAction, p paintengine2d.Point) {
	switch a {
	case platform.TitleToggleMaximize:
		w.ToggleMaximize()
	case platform.TitleToggleMaximizeVertically, platform.TitleToggleMaximizeHorizontally:
		if m, ok := w.surf.(platform.AxisMaximizer); ok {
			m.ToggleMaximizeAxis(a == platform.TitleToggleMaximizeVertically)
			return
		}
		// xdg-shell maximizes both ways or not at all.
		w.ToggleMaximize()
	case platform.TitleMinimize:
		w.Minimize()
	case platform.TitleLower:
		if l, ok := w.surf.(platform.Lowerer); ok {
			l.Lower()
		}
	case platform.TitleMenu:
		w.ShowWindowMenu(p)
	case platform.TitleClose:
		w.RequestClose()
	}
}

// RequestClose asks the window to close, as the desktop's close button does:
// a window that hides on close (SetCloseHides, a status menu) stays. It runs
// on the next turn of the loop, after the event being handled.
func (w *Window) RequestClose() {
	if w == nil || w.Closed() || w.app == nil {
		return
	}
	w.app.Post(func() {
		if !w.Closed() {
			w.dispatch(platform.Event{Kind: platform.EventClose})
		}
	})
}

// paintDecoration paints the look's frame of a framed window — its border
// and the caption band — under the title bar.
func (w *Window) paintDecoration(ctx *paintengine2d.Context) {
	g := w.geom
	if !g.framed || w.caption == nil {
		return
	}
	ww, hh := w.surf.Size()
	f := style.DecorationFrame{Window: paintengine2d.XYWH(0, 0, float32(ww), float32(hh))}
	cap, bar := w.caption.FrameParts()
	o := widget.DeviceOrigin(w.caption)
	f.Caption = cap.Translate(o)
	if !bar.Empty() {
		f.Bar = bar.Translate(o)
	}
	style.DrawDecorationOf(w.look, ctx, f, w.caption.DecorationState())
}

// ---- the desktop's title-bar conventions ------------------------------------

// titleBarPrefsCache holds the desktop's title-bar conventions and the
// config files they were read from, to re-read them when one changes.
type titleBarPrefsCache struct {
	prefs  platform.TitleBarPrefs
	ok     bool
	pinned bool
	stamps map[string]fileMeta
}

type fileMeta struct {
	size int64
	mod  time.Time
}

func statMeta(path string) fileMeta {
	st, err := os.Stat(path)
	if err != nil {
		return fileMeta{}
	}
	return fileMeta{size: st.Size(), mod: st.ModTime()}
}

// TitleBarPrefs are the desktop's title-bar conventions the toolkit's frame
// follows: the caption-button layout, the click actions, the double-click
// time and the drag threshold (read from kwinrc on KDE Plasma, the settings
// portal on GNOME, GTK's settings.ini elsewhere; defaults offscreen).
func (a *Application) TitleBarPrefs() platform.TitleBarPrefs {
	if a == nil {
		return platform.DefaultTitleBarPrefs("")
	}
	if !a.tbar.ok {
		a.loadTitleBarPrefs()
	}
	return a.tbar.prefs
}

// SetTitleBarPrefs overrides the desktop's title-bar conventions for every
// window (an app that wants its own button layout, tests).
func (a *Application) SetTitleBarPrefs(p platform.TitleBarPrefs) {
	a.tbar = titleBarPrefsCache{prefs: p, ok: true, pinned: true}
	a.titleBarPrefsChanged()
}

func (a *Application) loadTitleBarPrefs() {
	if a.offscreen() {
		// Screenshots and tests look the same on every machine.
		a.tbar = titleBarPrefsCache{prefs: platform.DefaultTitleBarPrefs(""), ok: true}
		return
	}
	stamps := map[string]fileMeta{}
	for _, f := range platform.TitleBarConfigFiles() {
		stamps[f] = statMeta(f)
	}
	a.tbar = titleBarPrefsCache{prefs: platform.ReadTitleBarPrefs(a.desktop), ok: true, stamps: stamps}
}

// refreshTitleBarPrefs re-reads the conventions when a config file they
// come from changed (called as a window becomes active: the user changed
// the setting in another window).
func (a *Application) refreshTitleBarPrefs() {
	if a == nil || !a.tbar.ok || a.tbar.pinned || a.offscreen() {
		return
	}
	for f, was := range a.tbar.stamps {
		if statMeta(f) != was {
			a.reloadTitleBarPrefs()
			return
		}
	}
}

// reloadTitleBarPrefs reads the conventions again and applies them.
func (a *Application) reloadTitleBarPrefs() {
	if a.tbar.pinned {
		return
	}
	was := a.tbar
	a.loadTitleBarPrefs()
	if was.ok && was.prefs.Layout.String() == a.tbar.prefs.Layout.String() {
		return // actions and timing apply on the next click anyway
	}
	a.titleBarPrefsChanged()
}

// titleBarPrefsChanged re-lays every caption out.
func (a *Application) titleBarPrefsChanged() {
	for _, w := range a.Windows() {
		if !w.Closed() && w.caption != nil {
			w.rebuildCaption()
		}
	}
}

// setDecorationsPref applies the user's decorations preference (look.json).
func (a *Application) setDecorationsPref(p style.DecorationsPref) {
	p = style.ParseDecorationsPref(string(p))
	if p == a.decorPref {
		return
	}
	a.decorPref = p
	for _, w := range a.Windows() {
		if !w.Closed() {
			w.syncDecorations()
		}
	}
}
