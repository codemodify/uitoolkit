package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// A frame with a shadow: the window is opaque to its last pixel, the
// margin around it holds nothing but the shadow, the corners are cut out,
// and everything the window tells the outside world — its size, its
// popups, its caret, its accessible boxes, its screenshots — is about the
// window, not about the buffer the shadow lives in.

// shadowRig is a framed window in a look whose frame has a shadow and
// rounded corners.
func shadowRig(t *testing.T, look style.LookAndFeel) *frameRig {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	r := &frameRig{}
	r.a = New(Options{Look: look, Headless: true})
	r.a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := r.a.NewWindow(platform.WindowOptions{
		Title: "Frame", Width: 400, Height: 300, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.w = w
	r.o = w.Surface().(*platform.Offscreen)
	r.hb = widgets.NewHeaderBar(nil, nil, nil)
	r.body = widgets.NewButton("Body", nil)
	w.SetContent(widgets.NewColumn(r.body))
	w.SetTitleBar(r.hb)
	r.o.SimulateWindowState(platform.WindowState{Activated: true})
	r.a.PumpOnce()
	return r
}

// bigSurLook is a look whose frame is rounded on all four corners and has
// a large shadow.
func bigSurLook(t *testing.T) style.LookAndFeel {
	t.Helper()
	pack, ok := style.LoadTheme("bigsur")
	if !ok {
		t.Skip("no bigsur pack")
	}
	return pack.Look()
}

func TestFrameAlphaIsOneInsideTheWindow(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	img := r.w.CaptureSurface()
	if img == nil {
		t.Fatal("no buffer")
	}
	g := r.w.geom
	if g.margin.Zero() || !g.shadow {
		t.Fatalf("the look drops no shadow: %+v", g.margin)
	}
	win := r.w.WindowRect()
	rad := int(g.radius[0] + 1)
	// Every pixel of the window is opaque, bar the squares the rounded
	// corners are cut out of.
	x0, y0, x1, y1 := int(win.Min.X), int(win.Min.Y), int(win.Max.X), int(win.Max.Y)
	inCorner := func(x, y int) bool {
		nearX := x < x0+rad || x >= x1-rad
		nearY := y < y0+rad || y >= y1-rad
		return nearX && nearY
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			_, _, _, a := img.PremulAt(x, y)
			if a == 255 || inCorner(x, y) {
				continue
			}
			t.Fatalf("window pixel %d,%d has alpha %d: a window's content is opaque", x, y, a)
		}
	}
	// The corners really are cut out: what is left there is the shadow
	// curving round them, never the window's own opaque content.
	if _, _, _, a := img.PremulAt(x0, y0); a > 120 {
		t.Fatalf("the top-left corner pixel has alpha %d: the corner is not cut out", a)
	}
	if _, _, _, a := img.PremulAt(x1-1, y1-1); a > 120 {
		t.Fatalf("the bottom-right corner pixel has alpha %d: the corner is not cut out", a)
	}
	if _, _, _, a := img.PremulAt(0, 0); a != 0 {
		t.Fatalf("the buffer's own corner has alpha %d, want 0 (nothing but shadow lives there)", a)
	}
}

func TestFrameShadowLivesOnlyInTheMargin(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	img := r.w.CaptureSurface()
	g := r.w.geom
	win := r.w.WindowRect()
	// Something is painted in the margin — that is the shadow.
	var painted int
	sw, sh := r.w.SurfaceSize()
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			if float32(x) >= win.Min.X && float32(x) < win.Max.X && float32(y) >= win.Min.Y && float32(y) < win.Max.Y {
				continue
			}
			if _, _, _, a := img.PremulAt(x, y); a > 0 {
				painted++
			}
		}
	}
	if painted == 0 {
		t.Fatal("no shadow in the margin")
	}
	// It fades out: the outermost ring of the buffer is (nearly) clear.
	for x := 0; x < sw; x++ {
		if _, _, _, a := img.PremulAt(x, 0); a > 24 {
			t.Fatalf("the shadow reaches the buffer's top edge at %d (alpha %d): the margin is too small", x, a)
		}
	}
	// And a solid frame has neither margin nor shadow.
	r.o.SimulateCompositing(false)
	r.a.PumpOnce()
	if g = r.w.geom; !g.margin.Zero() || g.shadow || g.radius != [4]float32{} {
		t.Fatalf("without a compositing manager the frame is solid: %+v", g)
	}
	if f := r.o.Frame(); f.Alpha {
		t.Fatalf("a solid frame needs no alpha: %+v", f)
	}
	if sw2, _ := r.w.SurfaceSize(); float32(sw2) != r.w.WindowRect().Dx() {
		t.Fatal("a solid frame fills its surface")
	}
	// The resize band moves back inside the window.
	win = r.w.WindowRect()
	if got, e := r.w.NonClientHit(paintengine2d.Pt(win.Min.X+1, win.Min.Y+100)); got != RegionResize || e != platform.EdgeLeft {
		t.Fatalf("solid left edge: %v %v", got, e)
	}
}

func TestFrameMarginsPerEdgeAndState(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	full := r.w.geom.margin
	if full.Zero() {
		t.Fatal("no margin")
	}
	// Maximized: the window is the screen's, margins and corners go.
	r.o.SimulateWindowState(platform.WindowState{Activated: true, Maximized: true})
	r.a.PumpOnce()
	if g := r.w.geom; !g.margin.Zero() || g.radius != [4]float32{} || g.shadow {
		t.Fatalf("maximized %+v", g)
	}
	// Full screen: no frame at all.
	r.o.SimulateWindowState(platform.WindowState{Activated: true, Fullscreen: true})
	r.a.PumpOnce()
	if f := r.o.Frame(); !f.Zero() {
		t.Fatalf("full screen %+v", f)
	}
	// Tiled to the left: no margin on the tiled edges, the free one keeps
	// its shadow; every corner beside a tiled edge is square.
	r.o.SimulateWindowState(platform.WindowState{
		Activated: true, Tiled: platform.EdgeLeft | platform.EdgeTop | platform.EdgeBottom,
	})
	r.a.PumpOnce()
	g := r.w.geom
	if g.margin.Left != 0 || g.margin.Top != 0 || g.margin.Bottom != 0 || g.margin.Right != full.Right {
		t.Fatalf("tiled margins %+v (free right edge was %d)", g.margin, full.Right)
	}
	if g.radius != [4]float32{} {
		t.Fatalf("a window tiled on three edges is square: %v", g.radius)
	}
	// Tiled on the left only: the right corners stay round.
	r.o.SimulateWindowState(platform.WindowState{Activated: true, Tiled: platform.EdgeLeft})
	r.a.PumpOnce()
	g = r.w.geom
	if g.radius[0] != 0 || g.radius[3] != 0 || g.radius[1] == 0 || g.radius[2] == 0 {
		t.Fatalf("one tiled edge squares off its own corners only: %v", g.radius)
	}
	if g.margin.Left != 0 || g.margin.Right == 0 {
		t.Fatalf("one tiled edge drops its own margin only: %+v", g.margin)
	}
}

func TestFrameShadowPatchIsCachedAcrossResizes(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	r.w.CaptureSurface()
	first := r.w.shadow.img
	if first == nil {
		t.Fatal("no shadow patch")
	}
	// A resize keeps it: the patch is a nine-patch, not a picture of the
	// window.
	r.w.dispatch(platform.Event{Kind: platform.EventResize, Width: 500, Height: 380})
	r.a.PumpOnce()
	r.w.CaptureSurface()
	if r.w.shadow.img != first {
		t.Fatal("the shadow was rasterised again for a new size")
	}
	// Losing focus paints a different shadow, so that one is rebuilt.
	r.o.SimulateWindowState(platform.WindowState{})
	r.a.PumpOnce()
	r.w.CaptureSurface()
	if r.w.shadow.img == first {
		t.Fatal("the backdrop shadow is the active one")
	}
}

func TestPartialRedrawNeverTouchesTheMargin(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	r.a.PumpOnce()
	win := r.w.WindowRect()
	// A widget repaint, the everyday case (a hover, a caret): its damage
	// stays inside the window.
	r.w.full = false
	r.w.dirty.Reset()
	r.w.Invalidate(r.body, r.body.LocalBounds())
	rects := r.w.paintRects()
	if len(rects) == 0 {
		t.Fatal("no damage")
	}
	for _, d := range rects {
		if d.Min.X < win.Min.X || d.Min.Y < win.Min.Y || d.Max.X > win.Max.X || d.Max.Y > win.Max.Y {
			t.Fatalf("damage %v reaches into the margin around %v", d, win)
		}
	}
	// A full repaint covers everything, shadow included.
	r.w.fullInvalidate()
	if got := r.w.paintRects(); got != nil {
		t.Fatalf("a full repaint presents the whole surface, got %v", got)
	}
}

func TestCaptureIsTheWindowAndCaptureSurfaceTheShadow(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	win := r.w.Capture()
	surf := r.w.CaptureSurface()
	if win == nil || surf == nil {
		t.Fatal("no capture")
	}
	if win.Width != 400 || win.Height != 300 {
		t.Fatalf("a screenshot is the window: %dx%d", win.Width, win.Height)
	}
	if surf.Width <= win.Width || surf.Height <= win.Height {
		t.Fatalf("the surface holds the margin too: %dx%d", surf.Width, surf.Height)
	}
	if w, h := r.w.Size(); w != 400 || h != 300 {
		t.Fatalf("Window.Size is the window: %dx%d", w, h)
	}
	// The crop starts at the window, not at the buffer: the title bar's
	// first pixel row is the same in both.
	m := r.w.geom.margin
	for _, x := range []int{10, 200, 390} {
		a1, b1, c1, d1 := win.PremulAt(x, 4)
		a2, b2, c2, d2 := surf.PremulAt(x+m.Left, 4+m.Top)
		if a1 != a2 || b1 != b2 || c1 != c2 || d1 != d2 {
			t.Fatalf("crop is off at x=%d: %v,%v,%v,%v vs %v,%v,%v,%v", x, a1, b1, c1, d1, a2, b2, c2, d2)
		}
	}
}

func TestPopupsAndTooltipsStayInsideTheWindow(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	win := r.w.WindowRect()
	// A menu asked for at the window's bottom-right corner is pulled back
	// inside the window, never into the margin.
	menu := widgets.ShowContextMenu(r.body, paintengine2d.Pt(win.Max.X-4, win.Max.Y-4),
		widgets.Item("One", nil), widgets.Item("Two", nil))
	if menu == nil {
		t.Fatal("no menu")
	}
	r.a.PumpOnce()
	b := menu.Bounds()
	if b.Min.X < win.Min.X || b.Min.Y < win.Min.Y || b.Max.X > win.Max.X || b.Max.Y > win.Max.Y {
		t.Fatalf("menu %v is not inside the window %v", b, win)
	}
	r.w.DismissPopup()
	// A tooltip, likewise.
	r.w.showTip("Tip", paintengine2d.Pt(win.Max.X-6, win.Max.Y-6))
	tip := r.w.Tooltip()
	if tip == nil {
		t.Fatal("no tooltip")
	}
	if tb := tip.Bounds(); tb.Max.X > win.Max.X || tb.Max.Y > win.Max.Y {
		t.Fatalf("tooltip %v is not inside the window %v", tb, win)
	}
}

func TestIMECursorAndAccessibleExtentsFollowTheMargin(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	field := widgets.NewTextField("", "", nil)
	r.w.SetContent(widgets.NewColumn(field))
	r.a.PumpOnce()
	r.w.RequestFocus(field)
	r.a.PumpOnce()
	x, y, _, _, on := r.o.IMECursor()
	if !on {
		t.Fatal("the input method is on for a focused field")
	}
	m := r.w.geom.margin
	fb := widget.DeviceBounds(field)
	if float32(x) < float32(m.Left) || float32(y) < float32(m.Top) {
		t.Fatalf("caret at %d,%d is not in surface coordinates (margin %+v, field %v)", x, y, m, fb)
	}
	if float32(x) < fb.Min.X-1 || float32(x) > fb.Max.X+1 {
		t.Fatalf("caret x=%d is outside the field %v", x, fb)
	}
	// Accessible boxes are relative to the window, so the margin is out.
	tree := r.w.AccessibleTree()
	if tree == nil || len(tree.Children) == 0 {
		t.Fatal("no accessible tree")
	}
	node := tree.Children[0]
	obj := &atspiObj{win: r.w, node: node}
	e := obj.extents()
	if e.X < 0 || e.Y < 0 {
		t.Fatalf("accessible extents %v are outside the window", e)
	}
	if want := int32(node.Bounds.Min.X - r.w.WindowRect().Min.X); e.X != want {
		t.Fatalf("accessible x %d, want %d (bounds %v, window %v)", e.X, want, node.Bounds, r.w.WindowRect())
	}
}

// The GPU path paints the same frame: the scene the window records is
// replayed onto a real GLES device, and the window is opaque inside with
// its corners cut out — the check that matters for an EGL surface with an
// alpha visual, where any pixel short of alpha 1 shows the desktop.
func TestFrameSceneOnGPUKeepsAlpha(t *testing.T) {
	if !paintengine2d.GPUAvailable() {
		t.Skip("no EGL/GLES")
	}
	r := shadowRig(t, bigSurLook(t))
	r.a.PumpOnce()
	scene := r.w.Scene()
	if scene == nil {
		t.Skip("no retained scene (UITK_SCENE=off)")
	}
	sw, sh := r.w.SurfaceSize()
	dev, err := paintengine2d.NewGPUDevice(sw, sh)
	if err != nil {
		t.Skipf("no GPU device: %v", err)
	}
	defer dev.Close()
	paintengine2d.DrawScene(scene, dev)
	img := dev.Image()
	if img == nil {
		t.Fatal("no GPU snapshot")
	}
	win := r.w.WindowRect()
	mid := func(x, y float32) (uint8, uint8, uint8, uint8) { return img.PremulAt(int(x), int(y)) }
	if _, _, _, a := mid(win.Min.X+win.Dx()/2, win.Min.Y+win.Dy()/2); a < 250 {
		t.Fatalf("the middle of the window has alpha %d on the GPU", a)
	}
	if _, _, _, a := mid(win.Min.X+2, win.Min.Y+win.Dy()/2); a < 250 {
		t.Fatalf("a window edge pixel has alpha %d on the GPU", a)
	}
	if _, _, _, a := mid(win.Min.X, win.Min.Y); a > 8 {
		t.Fatalf("the rounded corner has alpha %d on the GPU, want 0", a)
	}
	if _, _, _, a := mid(1, 1); a > 40 {
		t.Fatalf("the buffer's corner has alpha %d on the GPU: the margin is shadow only", a)
	}
}

// Pointer, touch and drop coordinates are the surface's, so a frame's
// margin shifts them: a press at the window's own (x, y) lands on the
// widget the user sees there.
func TestPointerAndDropCoordinatesFollowTheMargin(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	win := r.w.WindowRect()
	m := r.w.geom.margin
	if m.Zero() {
		t.Fatal("no margin")
	}
	b := widget.DeviceBounds(r.body)
	if b.Min.X < win.Min.X || b.Min.Y < win.Min.Y {
		t.Fatalf("the content %v is not inside the window %v", b, win)
	}
	at := paintengine2d.Pt((b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2)
	r.w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: at})
	if r.w.hover != widget.Component(r.body) {
		t.Fatalf("hover %v, want the button under the pointer", r.w.hover)
	}
	// The same point measured from the window's corner, as an app would
	// think of it, is the margin away from the buffer's corner.
	local := paintengine2d.Pt(at.X-win.Min.X, at.Y-win.Min.Y)
	if int(at.X-local.X) != m.Left || int(at.Y-local.Y) != m.Top {
		t.Fatalf("window point %v is surface point %v with margin %+v", local, at, m)
	}
	// A drag from another app finds the target at the same point.
	drop := newDropBox()
	r.w.SetContent(drop)
	r.a.PumpOnce()
	db := widget.DeviceBounds(drop)
	over := paintengine2d.Pt((db.Min.X+db.Max.X)/2, (db.Min.Y+db.Max.Y)/2)
	r.w.dispatch(platform.Event{Kind: platform.EventDragMotion, Pos: over, Mimes: []string{"text/plain"}})
	if r.w.dropOver != widget.Component(drop) {
		t.Fatalf("drop target %v at %v (window %v)", r.w.dropOver, over, win)
	}
	// A point in the margin is outside the window: nothing there takes it.
	r.w.dispatch(platform.Event{Kind: platform.EventDragMotion, Pos: paintengine2d.Pt(2, 2), Mimes: []string{"text/plain"}})
	if r.w.dropOver != nil {
		t.Fatalf("the shadow's margin took a drop: %v", r.w.dropOver)
	}
}

// dropBox is a plain drop target for the coordinate tests.
type dropBox struct{ widget.Base }

func newDropBox() *dropBox {
	d := &dropBox{}
	d.Init(d)
	return d
}

func (d *dropBox) DropTypes() []string        { return []string{"text/plain"} }
func (d *dropBox) Drop(widget.DropEvent) bool { return true }

// Windows in the same look and state share one shadow patch, which lives
// as long as a window holds it.
func TestShadowPatchShared(t *testing.T) {
	key := frameShadowKey{margin: [4]int{7, 7, 7, 7}, active: true}
	builds := 0
	build := func(extX, extY *float32) *paintengine2d.Image {
		builds++
		*extX, *extY = 3, 3
		return paintengine2d.NewImage(8, 8)
	}
	a, _, _ := acquireShadowPatch(key, build)
	b, _, _ := acquireShadowPatch(key, build)
	if a != b || builds != 1 {
		t.Fatalf("two windows built %d patches", builds)
	}
	releaseShadowPatch(key)
	releaseShadowPatch(key)
	sharedShadows.mu.Lock()
	_, kept := sharedShadows.m[key]
	sharedShadows.mu.Unlock()
	if kept {
		t.Fatal("the patch outlived the windows that used it")
	}
	if c, _, _ := acquireShadowPatch(key, build); c == a || builds != 2 {
		t.Fatal("a released patch was handed out again")
	}
	releaseShadowPatch(key)
}

// Two windows at one scale hold one look, so what is cached on the look
// (and the shadow patch keyed by it) is not built twice.
func TestWindowsShareScaledLook(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1.75})
	w1, err := a.NewWindow(platform.WindowOptions{Width: 100, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w2, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 90, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	if w1.Look() != w2.Look() {
		t.Fatal("two windows at one scale built two looks")
	}
	a.SetLook(style.LightLook())
	if w1.Look() != w2.Look() || w1.Look() == nil {
		t.Fatal("after a theme change the windows hold different looks")
	}
}
