package app

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// frameRig is a 600×400 offscreen window whose frame the toolkit draws, with
// a header bar: a two-button tool bar on the left, free space in the middle,
// a label on the right.
type frameRig struct {
	a      *Application
	w      *Window
	o      *platform.Offscreen
	tools  *widgets.ToolBar
	label  *widgets.Label
	hb     *widgets.HeaderBar
	body   *widgets.Button
	now    time.Time
	clicks int
}

func newFrameRig(t *testing.T, deco platform.Decorations) *frameRig {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	r := &frameRig{now: time.Unix(1000, 0)}
	r.a = New(Options{Look: style.DarkLook(), Headless: true})
	r.a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := r.a.NewWindow(platform.WindowOptions{Title: "Frame", Width: 600, Height: 400, Headless: true, Decorations: deco})
	if err != nil {
		t.Fatal(err)
	}
	r.w = w
	r.o = w.Surface().(*platform.Offscreen)
	w.SetClock(func() time.Time { return r.now })
	r.tools = widgets.NewToolBar(widgets.ToolText("Fetch", func() { r.clicks++ }), widgets.ToolText("Write", nil))
	r.label = widgets.NewLabel("Inbox")
	r.hb = widgets.NewHeaderBar([]widget.Component{r.tools}, nil, []widget.Component{r.label})
	r.body = widgets.NewButton("Body", nil)
	w.SetContent(widgets.NewColumn(r.body))
	w.SetTitleBar(r.hb)
	r.a.PumpOnce()
	return r
}

func (r *frameRig) ev(kind platform.EventKind, x, y float32, b platform.MouseButton) {
	r.w.dispatch(platform.Event{Kind: kind, Pos: paintengine2d.Pt(x, y), Button: b})
}

func (r *frameRig) click(x, y float32) {
	r.ev(platform.EventMouseDown, x, y, platform.ButtonLeft)
	r.ev(platform.EventMouseUp, x, y, platform.ButtonLeft)
}

// center of c in window coordinates.
func center(c widget.Component) paintengine2d.Point {
	b := widget.DeviceBounds(c)
	return paintengine2d.Pt((b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2)
}

func (r *frameRig) freeSpace() paintengine2d.Point {
	// Between the tool bar and the label.
	tb, lb := widget.DeviceBounds(r.tools), widget.DeviceBounds(r.label)
	return paintengine2d.Pt((tb.Max.X+lb.Min.X)/2, (tb.Min.Y+tb.Max.Y)/2)
}

// win is the visible window inside the surface. A look with a drop shadow
// keeps a margin around it, so every frame coordinate hangs off this box
// rather than off the buffer's corner.
func (r *frameRig) win() paintengine2d.Rect { return r.w.WindowRect() }

// outside is a point dx, dy device pixels outside the window's edge — in
// the shadow, where the resize band of a shadowed frame lives.
func (r *frameRig) edge(side platform.Edges, along float32) paintengine2d.Point {
	win := r.win()
	switch side {
	case platform.EdgeLeft:
		return paintengine2d.Pt(win.Min.X-2, along)
	case platform.EdgeRight:
		return paintengine2d.Pt(win.Max.X+1, along)
	case platform.EdgeTop:
		return paintengine2d.Pt(along, win.Min.Y-2)
	}
	return paintengine2d.Pt(along, win.Max.Y+1)
}

func (r *frameRig) button(b platform.CaptionButton) paintengine2d.Point {
	left, right := r.hb.Controls()
	for _, c := range []*widgets.WindowControls{left, right} {
		if rr := c.ButtonRect(b); !rr.Empty() {
			o := widget.DeviceOrigin(c)
			return paintengine2d.Pt(o.X+(rr.Min.X+rr.Max.X)/2, o.Y+(rr.Min.Y+rr.Max.Y)/2)
		}
	}
	return paintengine2d.Pt(-1, -1)
}

func TestFrameHitTestTable(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	w := r.w
	if w.Decorations() != platform.DecorationsClient || !r.hb.Framed() {
		t.Fatalf("mode %v framed %v", w.Decorations(), r.hb.Framed())
	}
	fetch := r.tools.ItemRect(0).Translate(widget.DeviceOrigin(r.tools))
	toolGap := widget.DeviceBounds(r.tools)
	win := r.win()
	if win.Dx() != 600 || win.Dy() != 400 {
		t.Fatalf("the window keeps the size the app asked for: %v", win)
	}
	if sw, sh := w.SurfaceSize(); float32(sw) <= win.Dx() || float32(sh) <= win.Dy() {
		t.Fatalf("the surface grows by the shadow's margin: %dx%d for %v", sw, sh, win)
	}
	cases := []struct {
		name  string
		p     paintengine2d.Point
		want  NonClientRegion
		edges platform.Edges
	}{
		{"top-left corner", paintengine2d.Pt(win.Min.X-2, win.Min.Y-2), RegionResize, platform.EdgeTop | platform.EdgeLeft},
		{"top edge", r.edge(platform.EdgeTop, 300), RegionResize, platform.EdgeTop},
		{"left edge", r.edge(platform.EdgeLeft, 200), RegionResize, platform.EdgeLeft},
		{"right edge", r.edge(platform.EdgeRight, 200), RegionResize, platform.EdgeRight},
		{"bottom edge", r.edge(platform.EdgeBottom, 300), RegionResize, platform.EdgeBottom},
		{"bottom-right corner", paintengine2d.Pt(win.Max.X+1, win.Max.Y+1), RegionResize, platform.EdgeBottom | platform.EdgeRight},
		{"bottom-left corner along the left edge", paintengine2d.Pt(win.Min.X-1, win.Max.Y-10), RegionResize, platform.EdgeBottom | platform.EdgeLeft},
		{"the far margin is not the window's", paintengine2d.Pt(win.Min.X-14, 200), RegionClient, 0},
		{"the band is outside: the close button keeps its corner", paintengine2d.Pt(win.Max.X-3, win.Min.Y+2), RegionClose, 0},
		{"close button inside the window", paintengine2d.Pt(win.Max.X-10, win.Min.Y+6), RegionClose, 0},
		{"close button", r.button(platform.CaptionClose), RegionClose, 0},
		{"maximize button", r.button(platform.CaptionMaximize), RegionMaximize, 0},
		{"minimize button", r.button(platform.CaptionMinimize), RegionMinimize, 0},
		{"tool button", paintengine2d.Pt((fetch.Min.X+fetch.Max.X)/2, (fetch.Min.Y+fetch.Max.Y)/2), RegionClient, 0},
		{"tool bar between buttons", paintengine2d.Pt(toolGap.Max.X-2, (toolGap.Min.Y+toolGap.Max.Y)/2), RegionCaption, 0},
		{"free space", r.freeSpace(), RegionCaption, 0},
		{"label", center(r.label), RegionCaption, 0},
		{"content", center(r.body), RegionClient, 0},
		{"content empty space", paintengine2d.Pt(300, 300), RegionClient, 0},
	}
	for _, c := range cases {
		got, e := w.NonClientHit(c.p)
		if got != c.want || e != c.edges {
			t.Errorf("%s at %v: %v %v, want %v %v", c.name, c.p, got, e, c.want, c.edges)
		}
	}
	// The caption sits inside the hairline border, the content below it.
	if g := w.geom; g.border != (style.Insets{Top: 1, Right: 1, Bottom: 1, Left: 1}) ||
		g.caption.Min != paintengine2d.Pt(win.Min.X+1, win.Min.Y+1) || g.content.Min.Y != g.caption.Max.Y ||
		w.Content().Bounds() != g.content {
		t.Fatalf("geometry %+v content %v", w.geom, w.Content().Bounds())
	}

	// Maximized: no border and no resize band; the close button reaches
	// the screen's corner.
	r.o.SimulateWindowState(platform.WindowState{Maximized: true, Activated: true})
	r.a.PumpOnce()
	if got, _ := w.NonClientHit(paintengine2d.Pt(599, 0)); got != RegionClose {
		t.Errorf("maximized corner: %v", got)
	}
	if got, _ := w.NonClientHit(paintengine2d.Pt(2, 200)); got != RegionClient {
		t.Errorf("maximized left edge: %v", got)
	}
	if w.geom.border != (style.Insets{}) || w.geom.caption.Min != (paintengine2d.Point{}) {
		t.Errorf("maximized geometry %+v", w.geom)
	}
	// Tiled on the left (a quick tile): the margin, the shadow and the
	// corners go on those edges, and only the free right edge resizes.
	r.o.SimulateWindowState(platform.WindowState{Activated: true, Tiled: platform.EdgeLeft | platform.EdgeTop | platform.EdgeBottom})
	r.a.PumpOnce()
	if m := w.geom.margin; m.Left != 0 || m.Top != 0 || m.Bottom != 0 || m.Right == 0 {
		t.Errorf("tiled margins %+v", m)
	}
	win = r.win()
	if win.Min != (paintengine2d.Point{}) {
		t.Errorf("a window tiled to the left starts at the buffer's corner: %v", win)
	}
	if got, _ := w.NonClientHit(paintengine2d.Pt(win.Min.X+2, 200)); got != RegionClient {
		t.Errorf("tiled left edge: %v", got)
	}
	if got, e := w.NonClientHit(r.edge(platform.EdgeRight, 200)); got != RegionResize || e != platform.EdgeRight {
		t.Errorf("tiled right edge: %v %v", got, e)
	}
	if got, e := w.NonClientHit(paintengine2d.Pt(win.Max.X+1, win.Max.Y-2)); got != RegionResize || e != platform.EdgeRight {
		t.Errorf("a corner with a tiled side resizes the free side: %v %v", got, e)
	}
	// Full screen: no frame, no caption gestures.
	r.o.SimulateWindowState(platform.WindowState{Fullscreen: true})
	r.a.PumpOnce()
	if got, _ := w.NonClientHit(r.freeSpace()); got != RegionClient || r.hb.Framed() {
		t.Errorf("full screen: %v framed %v", got, r.hb.Framed())
	}
}

// Under the desktop's frame the header bar is the window's first row: its
// free space still moves the window, but there are no caption buttons and
// no resize band, and a dialog covers it like the rest.
func TestFrameServerModeHeaderBarIsFirstRow(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsServer)
	w := r.w
	if w.Decorations() != platform.DecorationsServer || r.hb.Framed() {
		t.Fatalf("mode %v framed %v", w.Decorations(), r.hb.Framed())
	}
	left, right := r.hb.Controls()
	if left.Visible() || right.Visible() {
		t.Fatal("no caption buttons under the desktop's frame")
	}
	hb := r.hb.Bounds()
	if hb.Min != (paintengine2d.Point{}) || hb.Dx() != 600 || w.Content().Bounds().Min.Y != hb.Max.Y {
		t.Fatalf("header %v content %v", hb, w.Content().Bounds())
	}
	if got, _ := w.NonClientHit(paintengine2d.Pt(0, 0)); got == RegionResize {
		t.Fatal("no resize band under the desktop's frame")
	}
	if got, _ := w.NonClientHit(r.freeSpace()); got != RegionCaption {
		t.Fatalf("free space: %v", got)
	}
	widgets.Info(w.Content(), "Hi", "x", nil)
	r.a.PumpOnce()
	if w.Overlay() == nil || w.Overlay().Bounds() != paintengine2d.XYWH(0, 0, 600, 400) {
		t.Fatalf("overlay %v", w.Overlay())
	}
	p := r.freeSpace()
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseMove, p.X+40, p.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseUp, p.X+40, p.Y, platform.ButtonLeft)
	if r.o.FrameCalls().Moves != 0 {
		t.Fatal("a dialog covers the first row: no move")
	}
}

// A drag of the caption goes to the app first when it asks for it: taken,
// the desktop is never asked to move the window (a floating dock panel's
// window carries itself back over its host instead); refused, the desktop
// moves it as it would have. A click is still a click.
func TestCaptionDragGoesToTheAppFirst(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	var got []paintengine2d.Point
	take := true
	r.w.SetOnCaptionDrag(func(at paintengine2d.Point) bool {
		got = append(got, at)
		return take
	})
	p := r.freeSpace()
	r.click(p.X, p.Y)
	if len(got) != 0 {
		t.Fatal("a click on the caption ran the drag hook")
	}
	r.now = r.now.Add(time.Second)
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseMove, p.X+20, p.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseUp, p.X+20, p.Y, platform.ButtonLeft)
	if len(got) != 1 || got[0] != paintengine2d.Pt(p.X+20, p.Y) {
		t.Fatalf("the hook saw %v, want one drag at %v", got, paintengine2d.Pt(p.X+20, p.Y))
	}
	if n := r.o.FrameCalls().Moves; n != 0 {
		t.Fatalf("the app took the drag, and the desktop was still asked to move the window %d times", n)
	}
	take = false
	r.now = r.now.Add(time.Second)
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseMove, p.X+20, p.Y, platform.ButtonLeft)
	if n := r.o.FrameCalls().Moves; len(got) != 2 || n != 1 {
		t.Fatalf("refused, the drag should be the desktop's move: hook %d, moves %d", len(got), n)
	}
}

func TestCaptionDragThresholdClickAndDoubleClick(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	p := r.freeSpace()
	// A press that moves less than the threshold is a click: no move.
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseMove, p.X+3, p.Y+2, platform.ButtonLeft)
	r.ev(platform.EventMouseUp, p.X+3, p.Y+2, platform.ButtonLeft)
	if r.o.FrameCalls().Moves != 0 {
		t.Fatal("a click started a move")
	}
	// Past the threshold (8 px): one move, started while the button is
	// down; later motion does not start another.
	r.now = r.now.Add(time.Second)
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonLeft)
	if r.w.capture == nil {
		t.Fatal("the caption holds the pointer while pressed")
	}
	r.ev(platform.EventMouseMove, p.X+20, p.Y, platform.ButtonLeft)
	// The desktop owns the pointer now: capture and hover are gone (the
	// compositor sends a pointer leave; the release goes to it).
	if r.w.capture != nil || r.w.hover != nil || r.w.capPress.armed {
		t.Fatalf("capture %v hover %v armed %v", r.w.capture, r.w.hover, r.w.capPress.armed)
	}
	r.ev(platform.EventMouseMove, p.X+40, p.Y, platform.ButtonLeft)
	if n := r.o.FrameCalls().Moves; n != 1 {
		t.Fatalf("moves %d", n)
	}
	// A double click (two presses within the double-click time) runs the
	// double-click action: maximize, then restore.
	r.now = r.now.Add(time.Second)
	r.click(p.X, p.Y)
	r.now = r.now.Add(200 * time.Millisecond)
	r.click(p.X+2, p.Y)
	r.a.PumpOnce()
	if !r.w.WindowState().Maximized || r.o.FrameCalls().Moves != 1 {
		t.Fatalf("double click: state %+v moves %d", r.w.WindowState(), r.o.FrameCalls().Moves)
	}
	r.now = r.now.Add(time.Second)
	r.click(p.X, p.Y)
	r.now = r.now.Add(100 * time.Millisecond)
	r.click(p.X, p.Y)
	r.a.PumpOnce()
	if r.w.WindowState().Maximized {
		t.Fatal("a second double click restores")
	}
	// Two clicks further apart than the double-click time are two clicks.
	r.now = r.now.Add(time.Second)
	r.click(p.X, p.Y)
	r.now = r.now.Add(600 * time.Millisecond)
	r.click(p.X, p.Y)
	r.a.PumpOnce()
	if got := r.o.FrameCalls().Maximizes; len(got) != 2 {
		t.Fatalf("slow clicks maximized: %v", got)
	}
	// A click on the content in between breaks a double click.
	r.now = r.now.Add(time.Second)
	r.click(p.X, p.Y)
	b := center(r.body)
	r.click(b.X, b.Y)
	r.click(p.X, p.Y)
	if got := r.o.FrameCalls().Maximizes; len(got) != 2 {
		t.Fatalf("caption-content-caption is no double click: %v", got)
	}
}

// A drag that starts on a tool button presses the button; it never moves
// the window.
func TestDragFromControlDoesNotMove(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	fetch := r.tools.ItemRect(0).Translate(widget.DeviceOrigin(r.tools))
	c := paintengine2d.Pt((fetch.Min.X+fetch.Max.X)/2, (fetch.Min.Y+fetch.Max.Y)/2)
	r.ev(platform.EventMouseDown, c.X, c.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseMove, c.X+30, c.Y+2, platform.ButtonLeft)
	r.ev(platform.EventMouseUp, c.X+30, c.Y+2, platform.ButtonLeft)
	if r.o.FrameCalls().Moves != 0 {
		t.Fatal("dragging a tool button moved the window")
	}
	r.click(c.X, c.Y)
	if r.clicks != 1 {
		t.Fatalf("tool clicks %d", r.clicks)
	}
}

func TestFrameResizeEdges(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	left := r.edge(platform.EdgeLeft, 200)
	win := r.win()
	r.ev(platform.EventMouseMove, left.X, left.Y, platform.ButtonNone)
	if r.w.Cursor() != platform.CursorResizeW {
		t.Fatalf("cursor over the left edge: %v", r.w.Cursor())
	}
	r.ev(platform.EventMouseDown, left.X, left.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseDown, win.Max.X+1, win.Max.Y+1, platform.ButtonLeft)
	calls := r.o.FrameCalls()
	if len(calls.Resizes) != 2 || calls.Resizes[0] != platform.EdgeLeft || calls.Resizes[1] != platform.EdgeBottom|platform.EdgeRight {
		t.Fatalf("resizes %v", calls.Resizes)
	}
	if r.w.capture != nil || r.w.Cursor() != platform.CursorDefault {
		t.Fatal("the desktop owns the pointer after a resize starts")
	}
	// Back over the content, the content's cursor.
	b := center(r.body)
	r.ev(platform.EventMouseMove, b.X, b.Y, platform.ButtonNone)
	if r.w.Cursor() != platform.CursorDefault || r.w.hover != widget.Component(r.body) {
		t.Fatalf("content hover %v cursor %v", r.w.hover, r.w.Cursor())
	}
}

func TestCaptionRightClickWindowMenu(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	p := r.freeSpace()
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonRight)
	r.ev(platform.EventMouseUp, p.X, p.Y, platform.ButtonRight)
	if m := r.o.FrameCalls().Menus; len(m) != 1 || m[0] != p {
		t.Fatalf("desktop menu %v", m)
	}
	// Right-clicking a caption button asks for the window menu too.
	cb := r.button(platform.CaptionMaximize)
	r.ev(platform.EventMouseDown, cb.X, cb.Y, platform.ButtonRight)
	r.ev(platform.EventMouseUp, cb.X, cb.Y, platform.ButtonRight)
	if m := r.o.FrameCalls().Menus; len(m) != 2 {
		t.Fatalf("caption button menu %v", m)
	}
	// A desktop without a window menu: the toolkit's own, which the
	// release of the press that opened it does not pick from.
	r.o.SetWindowMenu(false)
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonRight)
	r.ev(platform.EventMouseUp, p.X, p.Y, platform.ButtonRight)
	pop, ok := r.w.Popup().(*widgets.PopupMenu)
	if !ok {
		t.Fatalf("toolkit window menu %T", r.w.Popup())
	}
	var names []string
	for _, it := range pop.Items {
		names = append(names, it.Text)
	}
	if len(names) != 4 || names[0] != "Ma&ximize" || names[1] != "Mi&nimize" || names[3] != "&Close" {
		t.Fatalf("menu %q", names)
	}
	if len(r.o.FrameCalls().Maximizes) != 0 {
		t.Fatal("the opening release picked an item")
	}
	// The header bar's own menu wins where the app gave one.
	r.w.DismissPopup()
	own := 0
	r.hb.OnContextMenu = func(paintengine2d.Point) bool { own++; return true }
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonRight)
	r.ev(platform.EventMouseUp, p.X, p.Y, platform.ButtonRight)
	if own != 1 || r.w.Popup() != nil {
		t.Fatalf("own menu %d popup %v", own, r.w.Popup())
	}
}

func TestCaptionButtons(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	_, right := r.hb.Controls()
	if got := right.Shown(); len(got) != 3 || got[0] != platform.CaptionMinimize || got[2] != platform.CaptionClose {
		t.Fatalf("default layout %v", got)
	}
	mn := r.button(platform.CaptionMinimize)
	r.click(mn.X, mn.Y)
	if r.o.FrameCalls().Minimizes != 1 {
		t.Fatal("minimize")
	}
	mx := r.button(platform.CaptionMaximize)
	r.click(mx.X, mx.Y)
	r.a.PumpOnce()
	if !r.w.WindowState().Maximized {
		t.Fatal("maximize")
	}
	// Maximized: the button restores.
	mx = r.button(platform.CaptionMaximize)
	r.ev(platform.EventMouseMove, mx.X, mx.Y, platform.ButtonNone)
	if right.Tooltip() != "Restore" {
		t.Fatalf("tooltip %q", right.Tooltip())
	}
	r.click(mx.X, mx.Y)
	r.a.PumpOnce()
	if r.w.WindowState().Maximized {
		t.Fatal("restore")
	}
	// The desktop cannot minimize: no minimize button.
	r.o.SimulateCapabilities(platform.CapMaximize | platform.CapWindowMenu)
	r.a.PumpOnce()
	if got := right.Shown(); len(got) != 2 || got[0] != platform.CaptionMaximize {
		t.Fatalf("without minimize %v", got)
	}
	// Close asks the window to close (on the next turn of the loop).
	cl := r.button(platform.CaptionClose)
	r.click(cl.X, cl.Y)
	if r.w.Closed() {
		t.Fatal("closed inside the click")
	}
	r.a.PumpOnce()
	if !r.w.Closed() {
		t.Fatal("close button")
	}
}

// Close honours close-to-tray like the desktop's close button.
func TestCaptionCloseHonoursCloseHides(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	r.w.SetCloseHides(true)
	cl := r.button(platform.CaptionClose)
	r.click(cl.X, cl.Y)
	r.a.PumpOnce()
	if r.w.Closed() || r.w.Visible() {
		t.Fatalf("closed %v visible %v", r.w.Closed(), r.w.Visible())
	}
}

func TestDesktopButtonLayout(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	p := platform.DefaultTitleBarPrefs("")
	p.Layout = platform.KDEButtonLayout("X", "IA")
	r.a.SetTitleBarPrefs(p)
	r.a.PumpOnce()
	left, right := r.hb.Controls()
	if got := left.Shown(); len(got) != 1 || got[0] != platform.CaptionClose {
		t.Fatalf("left %v", got)
	}
	if got := right.Shown(); len(got) != 2 || got[1] != platform.CaptionMaximize {
		t.Fatalf("right %v", got)
	}
	if cl := r.button(platform.CaptionClose); cl.X > 100 {
		t.Fatalf("close is on the left: %v", cl)
	}
	// The header bar's own items move over.
	if tb := widget.DeviceBounds(r.tools); tb.Min.X < widget.DeviceBounds(left).Max.X {
		t.Fatalf("tool bar %v overlaps the left buttons %v", tb, widget.DeviceBounds(left))
	}
	// The window-menu button opens the menu below itself.
	p.Layout = platform.ParseButtonLayout("icon:close")
	r.a.SetTitleBarPrefs(p)
	r.a.PumpOnce()
	mb := r.button(platform.CaptionMenu)
	r.click(mb.X, mb.Y)
	if m := r.o.FrameCalls().Menus; len(m) != 1 || m[0].Y <= mb.Y {
		t.Fatalf("window menu button %v at %v", m, mb)
	}
}

func TestFrameAccessibility(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	tree := r.w.AccessibleTree()
	if len(tree.Children) == 0 || tree.Children[0].Role != a11y.RoleTitleBar || tree.Children[0].Name != "Frame" {
		t.Fatalf("first child %+v", tree.Children)
	}
	find := func(name string) *a11y.Node {
		var hit *a11y.Node
		tree.Children[0].Walk(func(n *a11y.Node) bool {
			if n.Role == a11y.RoleButton && n.Name == name {
				hit = n
			}
			return hit == nil
		})
		return hit
	}
	for _, name := range []string{"Minimize", "Maximize", "Close"} {
		if n := find(name); n == nil || !n.Actions.Has(a11y.ActionDefault) || n.Bounds.Empty() {
			t.Fatalf("caption button %q: %+v", name, n)
		}
	}
	if probs := a11y.Check(tree); len(probs) != 0 {
		t.Fatalf("a11y problems: %v", probs)
	}
	// Assistive technology presses a caption button.
	if !r.w.AccessibleAction(find("Maximize").ID, a11y.ActionDefault) {
		t.Fatal("action")
	}
	r.a.PumpOnce()
	tree = r.w.AccessibleTree()
	if !r.w.WindowState().Maximized || find("Restore") == nil {
		t.Fatal("maximized through AT-SPI: the button is Restore now")
	}
	// The toolkit's default caption names the window too, with its title.
	r2 := newFrameRig(t, platform.DecorationsClient)
	r2.w.SetTitleBar(nil)
	r2.a.PumpOnce()
	tree = r2.w.AccessibleTree()
	if tree.Children[0].Role != a11y.RoleTitleBar || len(tree.Children[0].Children) == 0 ||
		tree.Children[0].Children[0].Role != a11y.RoleLabel || tree.Children[0].Children[0].Name != "Frame" {
		t.Fatalf("default caption %+v", tree.Children[0])
	}
}

// A modal dialog dims the content, not the caption: the window can still be
// moved and closed, but the title bar's own controls are inert.
func TestFrameModalKeepsCaption(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	widgets.Info(r.w.Content(), "Hi", "x", nil)
	r.a.PumpOnce()
	if ov := r.w.Overlay(); ov == nil || ov.Bounds() != r.w.geom.content {
		t.Fatalf("overlay %v content %v", r.w.Overlay(), r.w.geom.content)
	}
	fetch := r.tools.ItemRect(0).Translate(widget.DeviceOrigin(r.tools))
	c := paintengine2d.Pt((fetch.Min.X+fetch.Max.X)/2, (fetch.Min.Y+fetch.Max.Y)/2)
	r.click(c.X, c.Y)
	if r.clicks != 0 {
		t.Fatal("a tool in the title bar worked under a modal dialog")
	}
	p := r.freeSpace()
	r.ev(platform.EventMouseDown, p.X, p.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseMove, p.X+30, p.Y, platform.ButtonLeft)
	if r.o.FrameCalls().Moves != 1 {
		t.Fatal("the caption moves the window under a modal dialog")
	}
	mn := r.button(platform.CaptionMinimize)
	r.click(mn.X, mn.Y)
	if r.o.FrameCalls().Minimizes != 1 {
		t.Fatal("caption buttons work under a modal dialog")
	}
}

func TestFrameDefaultCaptionAndModeChanges(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Plain", Width: 400, Height: 300, Headless: true, Decorations: platform.DecorationsClient})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()
	hb := w.Caption()
	if hb == nil || !hb.ShowTitle || !hb.Framed() || w.Content().Bounds().Min.Y < 20 {
		t.Fatalf("default caption %v content %v", hb, w.Content().Bounds())
	}
	// The desktop answers with its own frame (KWin for a full-screen
	// window, a window rule): the toolkit's caption goes.
	o := w.Surface().(*platform.Offscreen)
	o.SimulateDecorations(platform.DecorationsServer)
	a.PumpOnce()
	if w.Caption() != nil || w.Content().Bounds() != paintengine2d.XYWH(0, 0, 400, 300) {
		t.Fatalf("server answer: caption %v content %v", w.Caption(), w.Content().Bounds())
	}
	o.SimulateDecorations(platform.DecorationsClient)
	a.PumpOnce()
	if w.Caption() == nil {
		t.Fatal("client again")
	}
	// A plain offscreen window keeps no frame (screenshots stay the same).
	plain, _ := a.NewWindow(platform.WindowOptions{Width: 200, Height: 100, Headless: true})
	plain.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()
	if plain.Caption() != nil || plain.Decorations() != platform.DecorationsServer {
		t.Fatalf("plain window %v %v", plain.Caption(), plain.Decorations())
	}
}

// fakeBackend stands for a real display backend (the policy only asks
// offscreen windows for no frame).
type fakeBackend struct{}

func (fakeBackend) Name() string { return "fake" }
func (fakeBackend) NewSurface(o platform.WindowOptions) (platform.Surface, error) {
	return platform.NewOffscreen(o), nil
}

func TestDecorationsPolicy(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	a := &Application{backend: fakeBackend{}}
	surf := platform.NewOffscreen(platform.WindowOptions{})
	opts := platform.WindowOptions{}
	if d := a.resolveDecorations(opts, false, surf); d != platform.DecorationsServer {
		t.Errorf("a plain window keeps the desktop's frame: %v", d)
	}
	if d := a.resolveDecorations(opts, true, surf); d != platform.DecorationsClient {
		t.Errorf("a window with a title bar draws its own: %v", d)
	}
	surf.SetMoveResize(false)
	if d := a.resolveDecorations(opts, true, surf); d != platform.DecorationsServer {
		t.Errorf("a desktop that cannot move on request keeps its frame: %v", d)
	}
	surf.SetMoveResize(true)
	a.decorPref = style.DecorationsSystem
	if d := a.resolveDecorations(opts, true, surf); d != platform.DecorationsServer {
		t.Errorf("Use system title bar and borders: %v", d)
	}
	a.decorPref = style.DecorationsToolkit
	if d := a.resolveDecorations(opts, false, surf); d != platform.DecorationsClient {
		t.Errorf("toolkit frames for every window: %v", d)
	}
	a.decorPref = style.DecorationsAuto
	if d := a.resolveDecorations(platform.WindowOptions{Decorations: platform.DecorationsNone}, true, surf); d != platform.DecorationsNone {
		t.Errorf("the app's explicit mode: %v", d)
	}
	if d := a.resolveDecorations(platform.WindowOptions{Popup: true, Decorations: platform.DecorationsClient}, true, surf); d != platform.DecorationsNone {
		t.Errorf("popups have no frame: %v", d)
	}
	t.Setenv(platform.EnvDecorations, "server")
	if d := a.resolveDecorations(platform.WindowOptions{Decorations: platform.DecorationsClient}, true, surf); d != platform.DecorationsServer {
		t.Errorf("UITK_DECORATIONS wins: %v", d)
	}
	t.Setenv(platform.EnvDecorations, "")
	off := &Application{backend: platform.OffscreenBackend{}}
	if d := off.resolveDecorations(opts, true, surf); d != platform.DecorationsServer {
		t.Errorf("offscreen windows get no frame by default: %v", d)
	}
}

// The user's preference switches running windows at once.
func TestDecorationsPrefSwitchesLive(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	// Not headless: New reads look.json, so never the user's own.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{Look: style.DarkLook(), Backend: "offscreen"})
	a.backend = fakeBackend{}
	a.headless = false
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("x"))
	w.SetTitleBar(widgets.NewHeaderBar(nil, nil, nil))
	a.PumpOnce()
	if w.Decorations() != platform.DecorationsClient {
		t.Fatalf("auto with a title bar: %v", w.Decorations())
	}
	a.setDecorationsPref(style.DecorationsSystem)
	a.PumpOnce()
	if w.Decorations() != platform.DecorationsServer || w.Caption().Framed() {
		t.Fatalf("system: %v framed %v", w.Decorations(), w.Caption().Framed())
	}
	a.setDecorationsPref(style.DecorationsAuto)
	a.PumpOnce()
	if w.Decorations() != platform.DecorationsClient || !w.Caption().Framed() {
		t.Fatalf("back to auto: %v", w.Decorations())
	}
}

func TestFrameKeyboard(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	field := widgets.NewTextField("", "search", nil)
	got := 0
	root := &keyRoot{onKey: func(e widget.KeyEvent) bool {
		if e.Key == platform.KeyU && e.Mods.Ctrl() {
			got++
			return true
		}
		return false
	}}
	root.Init(root)
	root.Add(widgets.NewColumn(r.body))
	r.w.SetContent(root)
	r.w.SetTitleBar(widgets.NewHeaderBar([]widget.Component{r.tools}, field, nil))
	r.a.PumpOnce()
	// Tab: the title bar's controls first, then the content's.
	r.w.RequestFocus(nil)
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
	if r.w.Focus() != widget.Component(r.tools) {
		t.Fatalf("first tab stop %T", r.w.Focus())
	}
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
	if r.w.Focus() != widget.Component(r.body) {
		t.Fatalf("third tab stop %T", r.w.Focus())
	}
	// Keys from the title bar reach the content root's shortcuts.
	r.w.RequestFocus(field)
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyU, Mods: platform.ModCtrl})
	if got != 1 {
		t.Fatalf("shortcut from the title bar: %d", got)
	}
	// Alt+F4 closes a window whose frame the toolkit draws.
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyF4, Mods: platform.ModAlt})
	r.a.PumpOnce()
	if !r.w.Closed() {
		t.Fatal("Alt+F4")
	}
}

type keyRoot struct {
	widget.Base
	onKey func(widget.KeyEvent) bool
}

func (k *keyRoot) Measure(c layout.Constraints) paintengine2d.Point {
	return k.Children()[0].Measure(c)
}

func (k *keyRoot) Arrange(r paintengine2d.Rect) {
	k.SetBounds(r)
	k.Children()[0].Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
}

func (k *keyRoot) KeyPress(e widget.KeyEvent) bool { return k.onKey(e) }

// Suspended windows keep their frame; a plain title-bar component (not a
// HeaderBar) is wrapped in one and gains caption buttons.
func TestTitleBarWidgetIsWrapped(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	tb := widgets.NewTitleBar("Settings", "")
	r.w.SetTitleBar(tb)
	r.a.PumpOnce()
	if r.w.TitleBar() != widget.Component(tb) || r.w.Caption() == nil || r.w.Caption().Center() != widget.Component(tb) || !r.w.Caption().Framed() {
		t.Fatalf("wrapped %v", r.w.Caption())
	}
	if got, _ := r.w.NonClientHit(center(tb)); got != RegionCaption {
		t.Fatalf("a TitleBar is caption: %v", got)
	}
	// DragArea / NoDrag override what their content says.
	drag := widgets.DragArea(widgets.NewButton("Logo", nil))
	stop := widgets.NoDrag(widgets.NewLabel("Status"))
	r.w.SetTitleBar(widgets.NewHeaderBar([]widget.Component{drag}, nil, []widget.Component{stop}))
	r.a.PumpOnce()
	if got, _ := r.w.NonClientHit(center(drag)); got != RegionCaption {
		t.Fatalf("drag area: %v", got)
	}
	if got, _ := r.w.NonClientHit(center(stop)); got != RegionClient {
		t.Fatalf("no-drag label: %v", got)
	}
}

// A stacked frame (Windows 95: the era's caption strip with the title and
// the buttons) puts the app's title bar in a row under the strip; both are
// caption where nothing else is.
func TestStackedFrame(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	pack, ok := style.LoadTheme("win95")
	if !ok {
		t.Fatal("no win95 pack")
	}
	a := New(Options{Look: pack.Look(), Headless: true})
	a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := a.NewWindow(platform.WindowOptions{Title: "Stacked", Width: 600, Height: 400, Headless: true, Decorations: platform.DecorationsClient})
	if err != nil {
		t.Fatal(err)
	}
	o := w.Surface().(*platform.Offscreen)
	fetch := 0
	tools := widgets.NewToolBar(widgets.ToolText("Fetch", func() { fetch++ }))
	hb := widgets.NewHeaderBar([]widget.Component{tools}, nil, nil)
	body := widgets.NewButton("Body", nil)
	w.SetContent(widgets.NewColumn(body))
	w.SetTitleBar(hb)
	a.PumpOnce()
	w.Capture()
	spec := style.DecorationOf(w.Look(), hb.DecorationState())
	if !spec.Stacked || !hb.Stacked() {
		t.Fatalf("win95 frames are stacked: %+v", spec)
	}
	strip, bar := hb.FrameParts()
	if strip.Dy() < spec.Caption || bar.Empty() {
		t.Fatalf("strip %v bar %v caption %v", strip, bar, spec.Caption)
	}
	o0 := widget.DeviceOrigin(hb)
	stripDev := strip.Translate(o0)
	// The tool bar is in the row under the strip; the content under both.
	if tb := widget.DeviceBounds(tools); tb.Min.Y < stripDev.Max.Y {
		t.Fatalf("tool bar %v inside the strip %v", tb, stripDev)
	}
	if cb := w.Content().Bounds(); cb.Min.Y < widget.DeviceBounds(hb).Max.Y {
		t.Fatalf("content %v overlaps the title bar %v", cb, widget.DeviceBounds(hb))
	}
	// The border is the look's (Win95's four pixels), not a hairline.
	if w.geom.border.Left != spec.Border.Left || spec.Border.Left < 2 {
		t.Fatalf("border %+v spec %+v", w.geom.border, spec.Border)
	}
	_, right := hb.Controls()
	cl := right.ButtonRect(platform.CaptionClose).Translate(widget.DeviceOrigin(right))
	if cl.Empty() || cl.Max.Y > stripDev.Max.Y+0.5 {
		t.Fatalf("close %v outside the strip %v", cl, stripDev)
	}
	mid := func(r paintengine2d.Rect) paintengine2d.Point {
		return paintengine2d.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	}
	if got, _ := w.NonClientHit(mid(cl)); got != RegionClose {
		t.Fatalf("close button: %v", got)
	}
	if got, _ := w.NonClientHit(paintengine2d.Pt(200, mid(stripDev).Y)); got != RegionCaption {
		t.Fatalf("the strip's free space: %v", got)
	}
	tbar := widget.DeviceBounds(tools)
	if got, _ := w.NonClientHit(paintengine2d.Pt(400, mid(tbar).Y)); got != RegionCaption {
		t.Fatalf("the row's free space: %v", got)
	}
	fr := tools.ItemRect(0).Translate(widget.DeviceOrigin(tools))
	if got, _ := w.NonClientHit(mid(fr)); got != RegionClient {
		t.Fatalf("a tool in the row: %v", got)
	}
	// The strip shows the window title, for assistive technology too.
	tree := w.AccessibleTree()
	if tb := tree.Children[0]; tb.Role != a11y.RoleTitleBar || len(tb.Children) == 0 || tb.Children[0].Role != a11y.RoleLabel || tb.Children[0].Name != "Stacked" {
		t.Fatalf("title bar %+v", tree.Children[0])
	}
	// A drag from the strip moves the window; a click on the tool works.
	p := paintengine2d.Pt(200, mid(stripDev).Y)
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: p, Button: platform.ButtonLeft})
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: p.Add(paintengine2d.Pt(30, 0)), Button: platform.ButtonLeft})
	if o.FrameCalls().Moves != 1 {
		t.Fatal("the strip moves the window")
	}
	c := mid(fr)
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: c, Button: platform.ButtonLeft})
	w.dispatch(platform.Event{Kind: platform.EventMouseUp, Pos: c, Button: platform.ButtonLeft})
	if fetch != 1 {
		t.Fatalf("tool clicks %d", fetch)
	}
	// Maximized: no border, the strip at the top.
	o.SimulateWindowState(platform.WindowState{Activated: true, Maximized: true})
	a.PumpOnce()
	w.Capture()
	if w.geom.border != (style.Insets{}) || widget.DeviceBounds(hb).Min != (paintengine2d.Point{}) {
		t.Fatalf("maximized: border %+v title bar %v", w.geom.border, widget.DeviceBounds(hb))
	}
	// Without the app's own title bar the frame is the strip alone.
	w.SetTitleBar(nil)
	a.PumpOnce()
	w.Capture()
	if st, b := w.Caption().FrameParts(); !b.Empty() || st.Dy() != widget.DeviceBounds(w.Caption()).Dy() {
		t.Fatalf("default caption: strip %v bar %v", st, b)
	}
}

// Browser tabs in the title bar (Chromium, SourceGit): the tabs are
// controls, the rest of the strip is caption, a right-click there shows the
// strip's own menu, and Ctrl+Tab reaches the app's shortcut handler.
func TestBrowserTabsInTheTitleBar(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	tabs := widgets.NewBrowserTabs("uitoolkit", "paintengine2d")
	menus := 0
	tabs.OnContextMenu = func(i int, _ paintengine2d.Point) bool { menus++; return i < 0 }
	switched := 0
	root := &keyRoot{onKey: func(e widget.KeyEvent) bool {
		if tabs.Shortcut(e) {
			switched++
			return true
		}
		return false
	}}
	root.Init(root)
	root.Add(widgets.NewColumn(r.body))
	r.w.SetContent(root)
	r.w.SetTitleBar(widgets.NewHeaderBar(nil, tabs, nil))
	r.a.PumpOnce()
	r.w.Capture()
	sb := widget.DeviceBounds(tabs)
	if hb := widget.DeviceBounds(r.w.Caption()); sb.Min.Y != hb.Min.Y || sb.Max.Y != hb.Max.Y {
		t.Fatalf("the strip %v fills the caption %v", sb, hb)
	}
	first := paintengine2d.Pt(sb.Min.X+60, sb.Max.Y-8)
	if got, _ := r.w.NonClientHit(first); got != RegionClient {
		t.Fatalf("a tab: %v", got)
	}
	empty := paintengine2d.Pt(sb.Max.X-20, sb.Max.Y-8)
	if got, _ := r.w.NonClientHit(empty); got != RegionCaption {
		t.Fatalf("the empty strip: %v", got)
	}
	// Clicking the second tab (tabs are 200 px wide here, the close button
	// at a tab's right end) selects it without moving the window.
	r.click(sb.Min.X+260, first.Y)
	if tabs.Selected() != 1 || r.o.FrameCalls().Moves != 0 {
		t.Fatalf("selected %d moves %d", tabs.Selected(), r.o.FrameCalls().Moves)
	}
	// The empty strip moves the window and has the strip's own menu.
	r.now = r.now.Add(time.Second)
	r.ev(platform.EventMouseDown, empty.X, empty.Y, platform.ButtonLeft)
	r.ev(platform.EventMouseMove, empty.X-30, empty.Y, platform.ButtonLeft)
	if r.o.FrameCalls().Moves != 1 {
		t.Fatal("the empty strip moves the window")
	}
	r.ev(platform.EventMouseDown, empty.X, empty.Y, platform.ButtonRight)
	r.ev(platform.EventMouseUp, empty.X, empty.Y, platform.ButtonRight)
	if menus != 1 || len(r.o.FrameCalls().Menus) != 0 {
		t.Fatalf("strip menu %d, window menus %v", menus, r.o.FrameCalls().Menus)
	}
	// Ctrl+Tab goes to the app (which switches tabs), plain Tab moves focus.
	r.w.RequestFocus(r.body)
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab, Mods: platform.ModCtrl})
	if switched != 1 || tabs.Selected() != 0 || r.w.Focus() != widget.Component(r.body) {
		t.Fatalf("Ctrl+Tab: switched %d selected %d focus %T", switched, tabs.Selected(), r.w.Focus())
	}
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
	if r.w.Focus() == widget.Component(r.body) {
		t.Fatal("Tab still moves the focus")
	}
	// Assistive technology sees the tabs inside the title bar.
	tree := r.w.AccessibleTree()
	found := false
	tree.Children[0].Walk(func(n *a11y.Node) bool {
		if n.Role == a11y.RoleTab && n.Name == "paintengine2d" {
			found = true
		}
		return true
	})
	if !found {
		t.Fatal("page tabs in the title bar's accessible tree")
	}
}

// "captionButtons": "theme" puts the caption buttons where the look does
// (the Mac's traffic lights on the left), and follows a change of look.
func TestThemeCaptionButtons(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	mac, ok := style.LoadTheme("bigsur")
	if !ok {
		t.Fatal("no bigsur pack")
	}
	r.a.SetLook(mac.Look())
	r.a.SetCaptionButtons(style.CaptionButtonsTheme)
	r.a.PumpOnce()
	left, right := r.hb.Controls()
	if got := left.Shown(); len(got) != 3 || got[0] != platform.CaptionClose || len(right.Shown()) != 0 {
		t.Fatalf("theme layout: left %v right %v", got, right.Shown())
	}
	// Another look, another layout: Windows 11 keeps them on the right.
	win11, _ := style.LoadTheme("fluent")
	r.a.SetLook(win11.Look())
	r.a.PumpOnce()
	if got := right.Shown(); len(got) != 3 || got[2] != platform.CaptionClose || len(left.Shown()) != 0 {
		t.Fatalf("fluent's layout: left %v right %v", left.Shown(), got)
	}
	// Back to the desktop's layout.
	r.a.SetLook(mac.Look())
	r.a.SetCaptionButtons(style.CaptionButtonsDesktop)
	r.a.PumpOnce()
	if len(left.Shown()) != 0 || len(right.Shown()) != 3 {
		t.Fatalf("desktop layout: left %v right %v", left.Shown(), right.Shown())
	}
}

// A new window title repaints the title bar the toolkit draws (a tab strip
// retitles the window as its selection changes).
func TestSetTitleRepaintsTheCaption(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	r.w.SetTitleBar(nil)
	r.a.PumpOnce()
	r.w.Capture()
	r.w.dirty.Reset()
	r.w.SetTitle("Renamed")
	hb := widget.DeviceBounds(r.w.Caption())
	covered := false
	for _, d := range r.w.dirty.Rects {
		if d.Intersect(hb) == hb {
			covered = true
		}
	}
	if !covered || r.w.Title() != "Renamed" {
		t.Fatalf("dirty %v caption %v", r.w.dirty.Rects, hb)
	}
	r.w.dirty.Reset()
	r.w.SetTitle("Renamed")
	if !r.w.dirty.Empty() {
		t.Fatal("the same title repaints nothing")
	}
}
