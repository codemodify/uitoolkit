package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A shaped window: the silhouette it tells the window system about, the
// states that drop it, what a screenshot of it is, where its damage goes,
// and the glass it asks for and does not always get. All headless, through
// the Offscreen backend, which answers for shapes and glass exactly as the
// two real backends do.

// shapeRig is an undecorated window with a ring silhouette — a rounded body
// with a hole through the middle, the spike's proof shape.
type shapeRig struct {
	a *Application
	w *Window
	o *platform.Offscreen
}

func newShapeRig(t *testing.T, width, height int) *shapeRig {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	r := &shapeRig{}
	r.a = New(Options{Headless: true})
	w, err := r.a.NewWindow(platform.WindowOptions{
		Title: "Shaped", Width: width, Height: height, Headless: true,
		Decorations: platform.DecorationsNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.w = w
	r.o = w.Surface().(*platform.Offscreen)
	w.SetContent(widgets.NewColumn(widgets.NewButton("Body", nil)))
	r.a.PumpOnce()
	return r
}

// ring is the test silhouette at size s: a rounded rectangle with a
// circular hole a little under a third of the way across.
func ring(size paintengine2d.Point, _ float32) *platform.Shape {
	p := paintengine2d.NewPath()
	p.AddRoundRect(paintengine2d.XYWH(0, 0, size.X, size.Y), size.X*0.06, size.Y*0.06)
	p.AddCircle(paintengine2d.Pt(size.X/2, size.Y/2), min(size.X, size.Y)*0.29)
	return platform.NewShapeEvenOdd(p)
}

func TestAnOrdinaryWindowIsUntouchedByShapes(t *testing.T) {
	// The regression guard. A window that asks for no shape and no glass
	// must state exactly what it stated before shapes existed: no shape,
	// no opaque override, no blur — and for an undecorated window, the
	// zero frame, which is the "say nothing at all" path in both backends.
	r := newShapeRig(t, 400, 300)
	r.a.PumpOnce()
	f := r.w.sysFrame
	if f.Shape != nil || f.Opaque != nil || f.Blur != nil {
		t.Fatalf("an unshaped window stated regions: %+v", f)
	}
	if !f.Zero() {
		t.Fatalf("an undecorated unshaped window asked for something: %+v", f)
	}
	if r.w.Shape() != nil || r.w.Glass() {
		t.Fatal("a window has no shape and no glass by default")
	}
	// And nothing it paints goes through the silhouette path.
	if r.w.shapeRaster() != nil {
		t.Fatal("an unshaped window rasterised a shape")
	}
	if r.w.seeThrough() {
		t.Fatal("an opaque unshaped window started see-through")
	}
	// Its screenshot is the plain buffer, as it always was.
	img := r.w.Capture()
	if img == nil || img.Width != 400 || img.Height != 300 {
		t.Fatalf("capture %v", img)
	}
}

func TestAShapedWindowStatesItsSilhouette(t *testing.T) {
	r := newShapeRig(t, 400, 400)
	r.w.SetShapeFunc(ring)
	r.a.PumpOnce()

	f := r.w.sysFrame
	if f.Shape == nil {
		t.Fatal("no shape reached the window system")
	}
	if !f.Alpha {
		t.Fatal("a shaped window needs an alpha channel")
	}
	// The hole is really a hole: the middle is in no rectangle at all.
	if platform.RectsContain(f.Shape, 200, 200) {
		t.Fatal("the middle of the hole is in the input region")
	}
	if !platform.RectsContain(f.Shape, 4, 200) {
		t.Fatal("the body is not in the input region")
	}
	// Nothing is claimed solid unless the app says so.
	if f.Opaque == nil || len(f.Opaque) != 0 {
		t.Fatalf("opaque %+v, want the empty region", f.Opaque)
	}
	r.w.SetShapeOpaque(true)
	r.a.PumpOnce()
	if f = r.w.sysFrame; len(f.Opaque) == 0 {
		t.Fatal("a window that says its shape is solid claims nothing")
	}
	for _, b := range f.Opaque {
		if !platform.RectsContain(f.Shape, b.X, b.Y) {
			t.Fatalf("opaque rect %+v is outside the silhouette", b)
		}
	}
}

func TestAShapeSurvivesResizeAndScale(t *testing.T) {
	r := newShapeRig(t, 400, 400)
	r.w.SetShapeFunc(ring)
	r.a.PumpOnce()
	if got := r.w.shapeRaster(); got == nil || got.W != 400 {
		t.Fatalf("raster %+v", got)
	}
	// A resize rebuilds it at the new size, hole and all.
	_ = r.o.Resize(600, 500)
	r.a.PumpOnce()
	got := r.w.shapeRaster()
	if got == nil || got.W != 600 || got.H != 500 {
		t.Fatalf("after a resize: %+v", got)
	}
	if got.Contains(300, 250) {
		t.Fatal("the hole did not move to the middle of the new size")
	}
	if !got.Contains(4, 250) {
		t.Fatal("the body is gone after the resize")
	}
	if f := r.w.sysFrame; f.Shape == nil || platform.RectsContain(f.Shape, 300, 250) {
		t.Fatal("the window system was not told the new silhouette")
	}
}

func TestMaximizeTileAndFullscreenDropTheShape(t *testing.T) {
	// A maximized, full-screen or tiled window fills a box the desktop
	// chose and shares its edges: a silhouette there would leave gaps the
	// desktop did not expect, exactly as a corner radius would — and those
	// are already dropped in these states.
	for _, tc := range []struct {
		name string
		st   platform.WindowState
	}{
		{"maximized", platform.WindowState{Maximized: true}},
		{"fullscreen", platform.WindowState{Fullscreen: true}},
		{"tiled", platform.WindowState{Tiled: platform.EdgeLeft}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newShapeRig(t, 400, 400)
			r.w.SetShapeFunc(ring)
			r.a.PumpOnce()
			if r.w.sysFrame.Shape == nil {
				t.Fatal("no shape to drop")
			}
			st := tc.st
			st.Activated = true
			r.o.SimulateWindowState(st)
			r.a.PumpOnce()
			if r.w.shapeRaster() != nil {
				t.Fatalf("%s kept its silhouette", tc.name)
			}
			if f := r.w.sysFrame; f.Shape != nil {
				t.Fatalf("%s still states a shape: %+v", tc.name, f.Shape)
			}
			// And restoring brings it back, hole and all.
			r.o.SimulateWindowState(platform.WindowState{Activated: true})
			r.a.PumpOnce()
			if r.w.shapeRaster() == nil || r.w.sysFrame.Shape == nil {
				t.Fatalf("%s did not get its silhouette back", tc.name)
			}
		})
	}
}

func TestCaptureCropsToTheShape(t *testing.T) {
	r := newShapeRig(t, 400, 400)
	// A disc inset from the window's edge: the crop has something to do.
	r.w.SetShapeFunc(func(size paintengine2d.Point, _ float32) *platform.Shape {
		return platform.ShapeEllipse(paintengine2d.XYWH(50, 50, size.X-100, size.Y-100))
	})
	r.a.PumpOnce()
	img := r.w.Capture()
	if img == nil {
		t.Fatal("no capture")
	}
	if img.Width != 300 || img.Height != 300 {
		t.Fatalf("capture %dx%d, want the silhouette's 300x300 bounds", img.Width, img.Height)
	}
	// Its corners are outside the disc, so they are see-through.
	for _, p := range [][2]int{{0, 0}, {299, 0}, {0, 299}, {299, 299}} {
		if _, _, _, a := img.PremulAt(p[0], p[1]); a != 0 {
			t.Fatalf("corner %v has alpha %d: the crop did not follow the shape", p, a)
		}
	}
	// And its middle is the window, painted.
	if _, _, _, a := img.PremulAt(150, 150); a == 0 {
		t.Fatal("the middle of the disc is see-through")
	}
}

func TestDamageStaysInsideTheShape(t *testing.T) {
	r := newShapeRig(t, 400, 400)
	r.w.SetShapeFunc(func(size paintengine2d.Point, _ float32) *platform.Shape {
		return platform.ShapeEllipse(paintengine2d.XYWH(100, 100, size.X-200, size.Y-200))
	})
	r.a.PumpOnce()
	r.w.frame()
	// Something changed in a corner the window does not own.
	r.w.dirty.Reset()
	r.w.full = false
	r.w.dirty.Add(paintengine2d.XYWH(0, 0, 400, 400))
	rects := r.w.paintRects()
	if len(rects) == 0 {
		t.Fatal("no damage at all")
	}
	for _, b := range rects {
		if b.Min.X < 100 || b.Min.Y < 100 || b.Max.X > 300 || b.Max.Y > 300 {
			t.Fatalf("damage %+v reaches outside the silhouette's bounds", b)
		}
	}
}

func TestGlassAsksOnlyWhereTheDesktopCanBlur(t *testing.T) {
	r := newShapeRig(t, 400, 300)
	r.w.SetGlass(true)
	r.a.PumpOnce()
	// The simulated desktop cannot blur: the window asks for nothing, and
	// the look keeps whatever it paints for itself.
	if r.w.GlassAvailable() {
		t.Fatal("the plain simulated desktop should not blur")
	}
	if f := r.w.sysFrame; f.Blur != nil {
		t.Fatalf("asked a desktop that cannot blur: %+v", f.Blur)
	}
	if style.GlassAvailable() {
		t.Fatal("style was told the desktop can blur")
	}
	// Switch the desktop's blur on and it asks.
	r.o.SimulateGlass(true)
	r.w.shapeChanged()
	r.a.PumpOnce()
	if !r.w.GlassAvailable() {
		t.Fatal("the desktop says it blurs and the window disagrees")
	}
	f := r.w.sysFrame
	if len(f.Blur) == 0 {
		t.Fatal("glass was available and the window asked for none")
	}
	if !f.Alpha {
		t.Fatal("a glassy window needs an alpha channel")
	}
	if !style.GlassAvailable() {
		t.Fatal("style was not told the desktop can blur")
	}
	// And the window's own fill goes translucent, or the blur behind it
	// would never show.
	if fill := r.w.windowFill(); fill.A >= 1 {
		t.Fatalf("glass fill %v is opaque", fill)
	}
	// Taking it away again withdraws the request.
	r.o.SimulateGlass(false)
	r.w.shapeChanged()
	r.a.PumpOnce()
	if f := r.w.sysFrame; f.Blur != nil {
		t.Fatalf("blur survived the desktop losing the capability: %+v", f.Blur)
	}
}

func TestGlassBlursOnlyTheShape(t *testing.T) {
	r := newShapeRig(t, 400, 400)
	r.o.SimulateGlass(true)
	r.w.SetShapeFunc(ring)
	r.w.SetGlass(true)
	r.a.PumpOnce()
	f := r.w.sysFrame
	if len(f.Blur) == 0 {
		t.Fatal("no blur region")
	}
	// The hole is not blurred: there is no window there to see through.
	if platform.RectsContain(f.Blur, 200, 200) {
		t.Fatal("the hole is in the blur region")
	}
	if !platform.RectsContain(f.Blur, 4, 200) {
		t.Fatal("the body is not in the blur region")
	}
}

func TestShapeAndGlassSurviveALookWithAFrame(t *testing.T) {
	// A framed window keeps a margin for its shadow, and a shape is stated
	// in the visible window's coordinates: the two have to agree, or the
	// silhouette lands a margin's width off.
	r := &shapeRig{}
	t.Setenv(platform.EnvDecorations, "")
	r.a = New(Options{Look: bigSurLook(t), Headless: true})
	r.a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := r.a.NewWindow(platform.WindowOptions{
		Title: "Shaped", Width: 400, Height: 400, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.w, r.o = w, w.Surface().(*platform.Offscreen)
	w.SetTitleBar(widgets.NewHeaderBar(nil, nil, nil))
	w.SetContent(widgets.NewColumn(widgets.NewButton("Body", nil)))
	r.o.SimulateWindowState(platform.WindowState{Activated: true})
	r.a.PumpOnce()
	m := r.w.sysFrame.Margin
	if m.Zero() {
		t.Skip("this look's frame keeps no margin")
	}
	w.SetShapeFunc(ring)
	r.a.PumpOnce()
	f := r.w.sysFrame
	if f.Shape == nil {
		t.Fatal("no shape")
	}
	// The silhouette sits inside the surface, moved by the margin: its
	// hole is at the middle of the *window*, not of the surface.
	ww, hh := r.w.shapeSize()
	if platform.RectsContain(f.Shape, m.Left+ww/2, m.Top+hh/2) {
		t.Fatal("the hole is not at the middle of the visible window")
	}
	if !platform.RectsContain(f.Shape, m.Left+4, m.Top+hh/2) {
		t.Fatal("the body is not where the visible window is")
	}
	// The shape takes the resize band's job over.
	if f.Input != (platform.FrameInsets{}) {
		t.Fatalf("a shaped window kept a resize band: %+v", f.Input)
	}
	// And the shadow it casts follows the silhouette rather than a
	// rounded rectangle.
	if p := r.w.shapeShadowPatch(r.w.shapeRaster()); p == nil || p.img == nil {
		t.Fatal("a framed shaped window casts no shadow")
	}
}

func TestShapedWindowPaintsItsHoleThrough(t *testing.T) {
	r := newShapeRig(t, 200, 200)
	r.w.SetShapeFunc(ring)
	r.a.PumpOnce()
	img := r.w.CaptureSurface()
	if img == nil {
		t.Fatal("no buffer")
	}
	// The middle of the hole is fully see-through, and the body is not.
	if _, _, _, a := img.PremulAt(100, 100); a != 0 {
		t.Fatalf("the hole has alpha %d", a)
	}
	if _, _, _, a := img.PremulAt(6, 100); a == 0 {
		t.Fatal("the body is see-through")
	}
	// Outside the rounded corner too.
	if _, _, _, a := img.PremulAt(0, 0); a != 0 {
		t.Fatalf("the corner outside the silhouette has alpha %d", a)
	}
}
