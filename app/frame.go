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
// in plain looks).
//
// A look with a drop shadow puts the visible window inside a larger
// surface: the margin around it holds the shadow, the resize handles live
// in it, and the rest of it clicks through (frameshadow.go paints it, and
// platform.Frame tells the window system where the window really is). A
// look without one keeps the window's geometry the whole surface and its
// resize handles inside its edges (the border, at least 4 px, and the top
// of the caption).

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
	// window is the visible window inside the surface: the whole surface
	// for a frame without a shadow, and the surface less margin for one
	// with (margin holds the shadow and the resize band).
	window paintengine2d.Rect
	margin platform.FrameInsets
	// input is how far into margin a press still reaches the window.
	input platform.FrameInsets
	// radius are the visible window's corner radii (top-left clockwise);
	// shadow says the look drops one into the margin.
	radius [4]float32
	shadow bool
	// border is the look's frame around a framed window (none when
	// maximized).
	border style.Insets
	// caption is the title bar's box (empty without one); content the
	// box of the content and, in a framed window, of the overlay.
	caption paintengine2d.Rect
	content paintengine2d.Rect
}

// translucent reports whether the frame needs an alpha channel: it has a
// shadow to drop in the margin, or corners to cut out of the window.
func (g frameGeom) translucent() bool {
	return g.framed && (g.shadow || g.radius != [4]float32{})
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
// the toolkit's frame for a window with a caption of its own (a title bar,
// or a caption whose drags the app takes) where the desktop can move and
// resize it on request; the desktop's frame otherwise.
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

// SetOnCaptionDrag gives the app a drag of the window's caption before the
// desktop's interactive move gets it. fn runs when a left press on caption
// space passes the drag threshold, with the pointer in window coordinates;
// returning true says the app took the drag — a floating dock panel's
// window starts a drag that carries it back over its host
// ([dock.Host], app.DockWindows) — and false leaves it the desktop's move.
// nil removes it.
//
// Only a caption the toolkit draws can do this: a desktop's own title bar
// moves its window without asking the client. So a window with a hook
// gets the toolkit's frame under the default decoration policy, as a
// window with a title bar of its own does; the user's "Use system title
// bar and borders", WindowOptions.Decorations and UITK_DECORATIONS still
// have the last word, and under the desktop's frame the hook never runs.
func (w *Window) SetOnCaptionDrag(fn func(at paintengine2d.Point) bool) {
	if w == nil {
		return
	}
	had := w.onCaptionDrag != nil
	w.onCaptionDrag = fn
	if had != (fn != nil) {
		w.syncDecorations()
	}
}

// ownsCaption reports whether the window has a caption of its own to
// show — a title bar, or a caption whose drags the app takes — which is
// what earns it the toolkit's frame under the default policy.
func (w *Window) ownsCaption() bool { return w.titleBar != nil || w.onCaptionDrag != nil }

// syncDecorations asks the surface for the mode the policy picks now and
// rebuilds the caption for the mode in effect.
func (w *Window) syncDecorations() {
	want := w.app.resolveDecorations(w.opts, w.ownsCaption(), w.surf)
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
		hb.SetWindowControls(w.buttonLayout(hb), framed)
	}
	if hb != w.caption {
		old := w.caption
		w.caption = hb
		if old != nil && old != hb {
			old.SetHost(nil)
		}
		w.capPress, w.capClick = captionGesture{}, captionClick{}
	}
	// The caption's buttons are half of what a fitted caption measures, and
	// therefore half of what a look's silhouette is stated against.
	w.lookShapeCur = nil
	w.laid = false
	w.dropScene()
	w.fullInvalidate()
}

// buttonLayout is where hb's caption buttons go: the desktop's layout, or
// the look's own when the user prefers it (look.json "captionButtons").
func (w *Window) buttonLayout(hb *widgets.HeaderBar) platform.ButtonLayout {
	if w.app.captionPref == style.CaptionButtonsTheme {
		st := hb.DecorationState()
		if l := style.DecorationOf(w.look, st).Layout; l != "" {
			return platform.ParseButtonLayout(l)
		}
	}
	return w.app.TitleBarPrefs().Layout
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

// frameSpec is the look's frame in the window's current state.
func (w *Window) frameSpec() style.DecorationSpec {
	if w.caption == nil {
		return style.DecorationSpec{}
	}
	return style.DecorationOf(w.look, w.caption.DecorationState())
}

// frameBorder is the look's border around a framed window, whole device
// pixels (none when maximized).
func (w *Window) frameBorder() style.Insets {
	if w.caption == nil || w.state.Maximized {
		return style.Insets{}
	}
	b := w.frameSpec().Border
	px := func(v float32) float32 { return max(float32(math.Round(float64(v))), 0) }
	return style.Insets{Top: px(b.Top), Right: px(b.Right), Bottom: px(b.Bottom), Left: px(b.Left)}
}

// resizeBandDip is how far outside the visible window a press still resizes
// it — the band inside the shadow (Chromium's 10-DIP kResizeBorder, GTK's
// 12px handle, SourceGit's 12px ring) — cornerBandDip how far a corner
// reaches along each edge (Chromium's kResizeAreaCornerSize), and
// insideBandDip the band a frame without a shadow keeps inside its own
// edges instead.
const (
	resizeBandDip = 10
	cornerBandDip = 16
	insideBandDip = 4
)

// resizeBand is the resize band and the corner zone in device pixels.
func (w *Window) resizeBand() (band, corner float32) {
	rnd := func(v float32) float32 { return float32(math.Round(float64(v))) }
	return rnd(style.Dip(w.look, resizeBandDip)), rnd(style.Dip(w.look, cornerBandDip))
}

// wantFrame is what the window system should know about the frame now: the
// margin the shadow needs (whole logical pixels, so the visible window
// starts on a pixel of the compositor's grid at any scale), how deep the
// resize band reaches into it, the corner radii and whether the buffer
// needs alpha. A window with no frame of the toolkit's, a maximized or
// full-screen one and one on an uncomposited X11 screen ask for nothing.
func (w *Window) wantFrame() platform.Frame {
	f := w.wantDecorFrame()
	// The silhouette and the glass go on last, and over the top of what
	// the look asked for: a shaped window's input region is its shape and
	// nothing else. A window with neither leaves f exactly as it was
	// before shapes existed.
	w.applyShapeToFrame(&f, f.Margin)
	return f
}

// wantDecorFrame is the frame the look's decoration asks for: the margin,
// the resize band, the corners and the alpha.
func (w *Window) wantDecorFrame() platform.Frame {
	if !w.framed() || w.caption == nil {
		return platform.Frame{}
	}
	spec := w.frameSpec()
	// DecorationOf has already dropped the shadow and the corners where
	// the window is maximized, tiled or uncomposited.
	band, _ := w.resizeBand()
	sc := max(w.scale, 1)
	edge := func(reach float32) int {
		if reach <= 0 {
			return 0
		}
		return platform.FrameMargin(max(reach, band), sc)
	}
	f := platform.Frame{Radius: spec.Radius}
	f.Margin = platform.FrameInsets{
		Top:    edge(spec.Shadow.Top),
		Right:  edge(spec.Shadow.Right),
		Bottom: edge(spec.Shadow.Bottom),
		Left:   edge(spec.Shadow.Left),
	}
	if !w.Resizable() {
		// Nothing to reach into the margin for: a fixed window's input
		// region is the window, and the shadow around it clicks through.
		band = 0
	}
	f.Input = platform.FrameInsets{
		Top:    min(f.Margin.Top, int(band)),
		Right:  min(f.Margin.Right, int(band)),
		Bottom: min(f.Margin.Bottom, int(band)),
		Left:   min(f.Margin.Left, int(band)),
	}
	f.Alpha = !f.Margin.Zero() || f.Radius != [4]float32{}
	return f
}

// applyFrame hands the frame to the window system before the window lays
// itself out: the surface grows by the margin at once, so this frame is
// painted at the size the compositor is told about in the same commit.
func (w *Window) applyFrame() {
	// The desktop's answer about glass changes while the app runs (KWin
	// drops it when desktop effects go off), and every look reads it
	// through style.GlassBehind, so it is refreshed here — once a layout,
	// beside everything else the window system has to say.
	if _, ok := w.surf.(platform.GlassSurface); ok {
		style.SetGlassAvailable(platform.SurfaceBlurBehind(w.surf))
	}
	f := w.wantFrame()
	if f.Same(w.sysFrame) {
		return
	}
	w.sysFrame = f
	platform.SetSurfaceFrame(w.surf, f)
	w.shadow.drop()
}

// layoutFrame arranges the caption inside the surface box full and returns
// the geometry.
func (w *Window) layoutFrame(full paintengine2d.Rect) frameGeom {
	g := frameGeom{content: full, window: full}
	if w.caption == nil {
		w.geom = g
		return g
	}
	g.framed = w.framed()
	inner := full
	if g.framed {
		m := w.sysFrame.Margin
		g.margin, g.input = m, w.sysFrame.Input
		g.radius = w.sysFrame.Radius
		g.shadow = !m.Zero()
		g.window = paintengine2d.Rect{
			Min: paintengine2d.Pt(full.Min.X+float32(m.Left), full.Min.Y+float32(m.Top)),
			Max: paintengine2d.Pt(full.Max.X-float32(m.Right), full.Max.Y-float32(m.Bottom)),
		}
		g.border = w.frameBorder()
		inner = w.innerBox(g.window, true)
	}
	g.caption = w.captionBox(inner, g.framed)
	h := g.caption.Dy()
	g.content = paintengine2d.XYWH(inner.Min.X, inner.Min.Y+h, inner.Dx(), max(0, inner.Dy()-h))
	w.caption.Arrange(g.caption)
	w.geom = g
	return g
}

// captionBox is where the caption band goes inside the window's inner box:
// as tall as the header bar measures and, normally, as wide as the window.
//
// A look whose frame is a tab rather than a band (DecorationSpec.CaptionFits
// — BeOS's) gets a caption only as wide as its contents instead, and the
// rest of the window's top edge is left to the silhouette to cut away. The
// two go together: DecorationOf drops the fitted caption wherever
// WindowShapeOf drops the silhouette, so there is never a narrow band on a
// window that really is a rectangle.
//
// It is a function of the look, the window's size and the header bar's
// content and nothing else, so the silhouette can ask for it before the
// layout that will use it.
func (w *Window) captionBox(inner paintengine2d.Rect, framed bool) paintengine2d.Rect {
	if w.caption == nil {
		return paintengine2d.Rect{}
	}
	sz := w.caption.Measure(layout.Loose(inner.Dx(), inner.Dy()))
	h := min(float32(math.Ceil(float64(sz.Y)-1e-3)), inner.Dy())
	width := inner.Dx()
	if framed && w.frameSpec().CaptionFits {
		if fit := w.caption.CaptionFitWidth(); fit > 0 {
			width = min(fit, inner.Dx())
		}
	}
	return paintengine2d.XYWH(inner.Min.X, inner.Min.Y, width, h)
}

// innerBox is the box a framed window's caption and content share: the
// visible window less the look's border.
func (w *Window) innerBox(win paintengine2d.Rect, framed bool) paintengine2d.Rect {
	if !framed {
		return win
	}
	return w.frameBorder().Apply(win)
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

// resizeEdgesAt is the resize edge under p. Without a shadow the band runs
// inside the window: the look's border, at least 4 px, and the top 4 px of
// the caption. With one it runs outside, in the margin (the shadow), where
// Chromium, GTK and SourceGit put theirs, and the margin beyond it is not
// the window's at all. Corners reach 16 px along each edge; there is no
// band at all while maximized or on a tiled or constrained edge, and none
// on a window whose size is fixed (platform.SizingFixed) — its whole edge
// is client space, so a press eight pixels in is the app's.
func (w *Window) resizeEdgesAt(p paintengine2d.Point) platform.Edges {
	st := w.state
	g := w.geom
	if !g.framed || st.Maximized || st.Fullscreen || !w.Resizable() {
		return 0
	}
	win := g.window
	bd := g.border
	inside := max(bd.Left, bd.Right, bd.Top, bd.Bottom, float32(math.Round(float64(style.Dip(w.look, insideBandDip)))))
	_, corner := w.resizeBand()
	// Each side's band: out from the visible window into the margin, or in
	// from its edge where there is no margin.
	out := func(v int) float32 { return float32(v) }
	l, r := out(g.input.Left), out(g.input.Right)
	t, b := out(g.input.Top), out(g.input.Bottom)
	if p.X < win.Min.X-l || p.Y < win.Min.Y-t || p.X >= win.Max.X+r || p.Y >= win.Max.Y+b {
		return 0
	}
	in := func(band float32) float32 {
		if band > 0 {
			return 0
		}
		return inside
	}
	var e platform.Edges
	switch {
	case p.X < win.Min.X+in(l):
		e |= platform.EdgeLeft
	case p.X >= win.Max.X-in(r):
		e |= platform.EdgeRight
	}
	switch {
	case p.Y < win.Min.Y+in(t):
		e |= platform.EdgeTop
	case p.Y >= win.Max.Y-in(b):
		e |= platform.EdgeBottom
	}
	if e == platform.EdgeTop || e == platform.EdgeBottom {
		if p.X < win.Min.X+corner {
			e |= platform.EdgeLeft
		} else if p.X >= win.Max.X-corner {
			e |= platform.EdgeRight
		}
	}
	if e == platform.EdgeLeft || e == platform.EdgeRight {
		if p.Y < win.Min.Y+corner {
			e |= platform.EdgeTop
		} else if p.Y >= win.Max.Y-corner {
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
			if w.onCaptionDrag != nil && w.onCaptionDrag(ev.Pos) {
				return true
			}
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
	w.dragArm = dragGesture{}
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
	f := style.DecorationFrame{Window: g.window}
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

// SetCaptionButtons puts the caption buttons of every frame the toolkit
// draws where the desktop's layout puts them (CaptionButtonsDesktop, the
// default) or where the look's own does (CaptionButtonsTheme): the user's
// look.json "captionButtons".
func (a *Application) SetCaptionButtons(p style.CaptionButtonsPref) {
	p = style.ParseCaptionButtonsPref(string(p))
	if a == nil || p == a.captionPref {
		return
	}
	a.captionPref = p
	a.titleBarPrefsChanged()
}

// CaptionButtons is where frames the toolkit draws put their caption
// buttons (see SetCaptionButtons).
func (a *Application) CaptionButtons() style.CaptionButtonsPref { return a.captionPref }

// Decorations is the user's preference for who draws the frame of the
// application's windows (look.json "decorations", or what ApplyAppearance
// last set). What a window actually got — the desktop has the last word —
// is [Window.Decorations].
func (a *Application) Decorations() style.DecorationsPref {
	if a == nil {
		return style.DecorationsAuto
	}
	return a.decorPref
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
