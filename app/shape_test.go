package app

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
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

// ringFace paints what examples/shapes paints: the silhouette filled, then
// a rim stroked along it, both even-odd. It is here so the edge test sees
// the same pixels a real shaped app puts on the boundary.
type ringFace struct{ widget.Base }

func newRingFace() *ringFace {
	f := &ringFace{}
	f.Init(f)
	return f
}

func (f *ringFace) Paint(ctx *paintengine2d.Context) {
	b := f.LocalBounds()
	w, h := b.Dx(), b.Dy()
	path, rule := ring(paintengine2d.Pt(w, h), 1).Path()
	pal := f.Look().Palette()
	fill := paintengine2d.Fill(pal.Background.WithAlpha(0.93))
	fill.FillRule, fill.AntiAlias = rule, true
	ctx.DrawPath(path, fill)
	rim := paintengine2d.StrokePaint(pal.Accent, max(min(w, h)*0.012, 2))
	rim.FillRule, rim.AntiAlias = rule, true
	ctx.DrawPath(path, rim)
}

func TestNothingSurvivesInsideTheCut(t *testing.T) {
	// The edge test. Every pixel the silhouette does not cover has to end
	// up *exactly* transparent — not nearly — and no pixel anywhere in the
	// cut may keep more than its own coverage.
	//
	// What this guards against is a thread of half-erased pixels along the
	// boundary: it showed up as a stipple just inside the hole, visible at
	// 1x against a saturated background, because the cut was blended away
	// with the coverage mask instead of the uncovered part being wiped.
	// A blended draw does not promise to touch every sample of a pixel;
	// a rectangle wipe does.
	for _, n := range []int{200, 300, 420} {
		r := newShapeRig(t, n, n)
		r.w.SetContent(newRingFace())
		r.w.SetShapeFunc(ring)
		r.a.PumpOnce()
		r.w.frame()
		// And again after a partial repaint across the hole's edge, which
		// is what a hover or an activation change does to a live window.
		r.w.dirty.Reset()
		r.w.full = false
		r.w.dirty.Add(paintengine2d.XYWH(0, float32(n)/3, float32(n), float32(n)/3))
		r.w.frame()

		img, ras := r.w.surf.Buffer(), r.w.shapeRaster()
		if img == nil || ras == nil {
			t.Fatalf("n=%d: nothing painted", n)
		}
		var survivors, notClear int
		var first string
		for y := 0; y < ras.H; y++ {
			for x := 0; x < ras.W; x++ {
				c := ras.Mask[y*ras.W+x]
				_, _, _, a := img.PremulAt(x, y)
				if c == 0 && a != 0 {
					notClear++
					if first == "" {
						first = fmt.Sprintf("(%d,%d) is uncovered but alpha %d", x, y, a)
					}
				}
				if int(a) > int(c) {
					survivors++
					if first == "" {
						first = fmt.Sprintf("(%d,%d) coverage %d but alpha %d", x, y, c, a)
					}
				}
			}
		}
		if notClear > 0 || survivors > 0 {
			t.Fatalf("n=%d: %d uncovered pixels are not transparent and %d keep more than their coverage: %s",
				n, notClear, survivors, first)
		}
	}
}

// wipeWatcher is a device that records what erased what: the path of every
// dest-out fill (a wipe, which writes every sample of every pixel it
// covers) and the box of every image blit (a blend through the coverage
// mask, which does not promise to).
type wipeWatcher struct {
	*paintengine2d.Recorder
	wipes []*paintengine2d.Path
	blits []paintengine2d.Rect
}

func (w *wipeWatcher) Fill(p *paintengine2d.Path, m paintengine2d.Matrix, paint paintengine2d.Paint, c paintengine2d.Clip) {
	if paint.Blend == paintengine2d.BlendDestOut {
		q := p.Clone()
		q.Transform(m)
		w.wipes = append(w.wipes, q)
	}
	w.Recorder.Fill(p, m, paint, c)
}

func (w *wipeWatcher) Blit(src *paintengine2d.Image, srcRect, dstRect paintengine2d.Rect, m paintengine2d.Matrix, paint paintengine2d.Paint, c paintengine2d.Clip) {
	w.blits = append(w.blits, m.TransformRect(dstRect))
	w.Recorder.Blit(src, srcRect, dstRect, m, paint, c)
}

// wiped rasterises every recorded wipe, so the test asks which pixels were
// really taken out rather than trusting a bounding box: the wipe is one
// path of many rectangles and its bounds are the whole window.
func (w *wipeWatcher) wiped(width, height int) []uint8 {
	img := paintengine2d.NewImage(width, height)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	for _, p := range w.wipes {
		ctx.DrawPath(p, paintengine2d.Fill(paintengine2d.White))
	}
	out := make([]uint8, width*height)
	stride := img.RowStride()
	for y := 0; y < height; y++ {
		row := img.Pix[y*stride:]
		for x := 0; x < width; x++ {
			out[y*width+x] = row[x*4+3]
		}
	}
	return out
}

func TestUncoveredPixelsAreWipedNotBlended(t *testing.T) {
	// The mechanism behind TestNothingSurvivesInsideTheCut, pinned where a
	// CPU render cannot see it. A pixel the silhouette does not cover has
	// to be taken out by a wipe, which writes every sample of it; leaving
	// it to an image blit through the coverage mask is what put a stipple
	// along the hole's boundary on a multisampled GPU, and no amount of
	// rendering on the CPU would have shown it.
	r := newShapeRig(t, 300, 300)
	r.w.SetContent(newRingFace())
	r.w.SetShapeFunc(ring)
	r.a.PumpOnce()
	ras := r.w.shapeRaster()
	if ras == nil {
		t.Fatal("no silhouette")
	}
	watch := &wipeWatcher{Recorder: paintengine2d.NewRecorder(ras.W, ras.H)}
	ctx := paintengine2d.NewContextDevice(watch)
	r.w.paintWindowShape(ctx, nil)
	if len(watch.wipes) == 0 {
		t.Fatal("a ring has a hole to wipe and nothing was wiped")
	}
	wiped := watch.wiped(ras.W, ras.H)
	for y := 0; y < ras.H; y++ {
		for x := 0; x < ras.W; x++ {
			c := ras.Mask[y*ras.W+x]
			switch {
			case c == 0 && wiped[y*ras.W+x] != 255:
				t.Fatalf("uncovered pixel (%d,%d) is not wiped (wipe coverage %d)",
					x, y, wiped[y*ras.W+x])
			case c == 255 && wiped[y*ras.W+x] != 0:
				t.Fatalf("solidly covered pixel (%d,%d) is wiped away", x, y)
			}
		}
	}
}

// ---- the silhouette a look declares -----------------------------------------

// framedRig is a window the toolkit frames, in a named pack: the shape of
// such a window is its look's business, not the app's.
func framedRig(t *testing.T, pack string, width, height int) *shapeRig {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := style.LoadTheme(pack)
	if !ok {
		t.Skipf("no %s pack", pack)
	}
	t.Setenv(platform.EnvDecorations, "")
	r := &shapeRig{}
	r.a = New(Options{Look: p.Look(), Headless: true})
	r.a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := r.a.NewWindow(platform.WindowOptions{
		Title: "Deck", Width: width, Height: height, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.w, r.o = w, w.Surface().(*platform.Offscreen)
	w.SetContent(widgets.NewColumn(widgets.NewButton("Body", nil)))
	r.o.SimulateWindowState(platform.WindowState{Activated: true})
	r.a.PumpOnce()
	return r
}

// A window that asks for no shape of its own is cut to the one its look
// declares, and tells the window system about it exactly as an app's own
// shape is told.
func TestAWindowTakesTheSilhouetteItsLookDeclares(t *testing.T) {
	r := framedRig(t, "deck", 500, 360)
	if r.w.Shape() == nil || r.w.sysFrame.Shape == nil {
		t.Fatal("a window in a shaped look states no silhouette")
	}
	m := r.w.sysFrame.Margin
	ww, hh := r.w.shapeSize()
	// Deck's outline is a full-width shoulder over a narrower body: the
	// middle of the window is the window, and the strip beside the body is
	// not — that is where the desktop shows through.
	if !platform.RectsContain(r.w.sysFrame.Shape, m.Left+ww/2, m.Top+hh/2) {
		t.Fatal("the middle of the window is not the window")
	}
	if !platform.RectsContain(r.w.sysFrame.Shape, m.Left+ww/2, m.Top+2) {
		t.Fatal("the shoulder is not the window")
	}
	if platform.RectsContain(r.w.sysFrame.Shape, m.Left+1, m.Top+hh-4) {
		t.Fatal("the strip beside the body is the window")
	}
	// A look's silhouette is a frame's, so a window the desktop frames is
	// the rectangle inside that frame.
	if !r.w.framed() {
		t.Fatal("the rig is not a framed window")
	}
}

// The states that drop an app's silhouette drop a look's, and the fitted
// caption that goes with it comes back with it.
func TestMaximizeDropsTheLooksSilhouette(t *testing.T) {
	for _, pack := range []string{"deck", "beos"} {
		t.Run(pack, func(t *testing.T) {
			r := framedRig(t, pack, 500, 360)
			if r.w.sysFrame.Shape == nil {
				t.Fatal("no silhouette to drop")
			}
			wide := r.w.geom.caption.Dx()
			r.o.SimulateWindowState(platform.WindowState{Activated: true, Maximized: true})
			r.a.PumpOnce()
			if r.w.shapeRaster() != nil || r.w.sysFrame.Shape != nil {
				t.Fatal("a maximized window kept its look's silhouette")
			}
			if got := r.w.geom.caption.Dx(); got <= wide {
				t.Errorf("maximized caption is %v wide, no wider than the fitted %v", got, wide)
			}
			r.o.SimulateWindowState(platform.WindowState{Activated: true})
			r.a.PumpOnce()
			if r.w.sysFrame.Shape == nil {
				t.Fatal("restoring did not bring the silhouette back")
			}
		})
	}
}

// BeOS's caption band is its tab, so the band is narrower than the window
// and the silhouette stops where the band does.
func TestTheBeOSCaptionIsItsTab(t *testing.T) {
	r := framedRig(t, "beos", 600, 400)
	g := r.w.geom
	if g.caption.Dx() >= g.window.Dx()*0.8 {
		t.Fatalf("the tab is %v wide on a %v window: it was not fitted", g.caption.Dx(), g.window.Dx())
	}
	m := r.w.sysFrame.Margin
	x := int(g.caption.Max.X - g.window.Min.X)
	if !platform.RectsContain(r.w.sysFrame.Shape, m.Left+x-2, m.Top+1) {
		t.Fatal("the tab's own last column is not the window")
	}
	sil := platform.OffsetRects(r.w.shapeRaster().Rects, m.Left, m.Top)
	if platform.RectsContain(sil, m.Left+x+8, m.Top+1) {
		t.Fatal("the top edge past the tab is the window")
	}
	// The step beside the tab is no edge of the window: no resize band
	// there, and no input.
	if e := r.w.shapeBandEdges(paintengine2d.Pt(float32(m.Left+x+3), float32(m.Top+4))); e != 0 {
		t.Fatalf("beside the tab: edges %v, want none", e)
	}
	if platform.RectsContain(r.w.sysFrame.Shape, m.Left+x+8, m.Top+1) {
		t.Fatal("the input region reaches past the tab")
	}
}

// An app that sets its own shape keeps it: a look has no idea what picture
// the app is drawing, so the app wins and dropping its shape gives the
// look's back.
func TestAnAppsOwnShapeWinsOverItsLooks(t *testing.T) {
	r := framedRig(t, "deck", 500, 360)
	looks := r.w.Shape()
	if looks == nil {
		t.Fatal("no look silhouette to start from")
	}
	r.w.SetShapeFunc(ring)
	r.a.PumpOnce()
	if got := r.w.Shape(); got == nil || got == looks {
		t.Fatal("the app's own shape did not win")
	}
	ww, hh := r.w.shapeSize()
	m := r.w.sysFrame.Margin
	if platform.RectsContain(r.w.sysFrame.Shape, m.Left+ww/2, m.Top+hh/2) {
		t.Fatal("the ring's hole is not there: this is not the app's shape")
	}
	r.w.SetShapeFunc(nil)
	r.a.PumpOnce()
	if r.w.Shape() == nil {
		t.Fatal("dropping the app's shape did not give the look's back")
	}
	if !platform.RectsContain(r.w.sysFrame.Shape, m.Left+ww/2, m.Top+hh/2) {
		t.Fatal("the hole survived the app's shape being dropped")
	}
}

// And the promise the other 121 packs rely on: a framed window in a look
// that declares nothing states no shape at all, and takes the path it took
// before any of this existed.
func TestAFramedWindowInAPlainLookStatesNoShape(t *testing.T) {
	r := framedRig(t, "breeze-night", 500, 360)
	if r.w.Shape() != nil || r.w.sysFrame.Shape != nil {
		t.Fatal("a look that declares no silhouette gave a window one")
	}
	if r.w.shapeRaster() != nil {
		t.Fatal("a window in a plain look rasterised a shape")
	}
	if r.w.sysFrame.Input == (platform.FrameInsets{}) && !r.w.sysFrame.Margin.Zero() {
		t.Fatal("a window with a shadow margin lost its resize band")
	}
}

// A shaped window the user may resize keeps a band along its silhouette's
// outer edge — outside it where there is room, a thin strip inside it —
// and a press there starts a resize from the edge it is on; a fixed window
// keeps none, and a hole is never an edge.
func TestShapedWindowKeepsAResizeBand(t *testing.T) {
	for _, fixed := range []bool{false, true} {
		t.Run(fmt.Sprint("fixed=", fixed), func(t *testing.T) {
			a := New(Options{Look: style.LightLook(), Headless: true})
			opts := platform.WindowOptions{Width: 400, Height: 400, Headless: true, Decorations: platform.DecorationsNone}
			if fixed {
				opts.Sizing = platform.SizingFixed
			}
			w, err := a.NewWindow(opts)
			if err != nil {
				t.Fatal(err)
			}
			w.SetContent(widgets.NewLabel(""))
			w.SetShapeFunc(ring)
			a.PumpOnce()
			o := w.Surface().(*platform.Offscreen)
			// The ring's rim at its left, half way down; its hole.
			left := paintengine2d.Pt(2, 200)
			hole := paintengine2d.Pt(200, 200)
			top := paintengine2d.Pt(200, 2)
			if fixed {
				if e := w.shapeBandEdges(left); e != 0 {
					t.Fatalf("a fixed window resizes from %v", e)
				}
				return
			}
			if e := w.shapeBandEdges(left); e != platform.EdgeLeft {
				t.Fatalf("left rim: %v", e)
			}
			if e := w.shapeBandEdges(top); e != platform.EdgeTop {
				t.Fatalf("top rim: %v", e)
			}
			if e := w.shapeBandEdges(hole); e != 0 {
				t.Fatalf("the hole resizes %v", e)
			}
			// The band takes presses: it is in the input region.
			if !platform.RectsContain(w.sysFrame.Shape, 1, 200) {
				t.Fatal("the band is not in the input region")
			}
			w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: left, Button: platform.ButtonLeft})
			a.PumpOnce()
			if rs := o.FrameCalls().Resizes; len(rs) != 1 || rs[0] != platform.EdgeLeft {
				t.Fatalf("resizes %v", rs)
			}
		})
	}
}
