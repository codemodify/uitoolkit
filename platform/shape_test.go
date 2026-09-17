package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A shape and the rectangles it becomes: the silhouette a window system is
// told about. Everything here is headless — no cgo, no display — because
// the rasterisation is the part that has to be right at every scale.

// ringPath is the spike's proof shape: a rounded body with a circular hole
// in the middle, filled even-odd. At w*h device pixels.
func ringPath(w, h float32) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	p.AddRoundRect(paintengine2d.XYWH(0, 0, w, h), w*0.06, h*0.06)
	p.AddCircle(paintengine2d.Pt(w/2, h/2), min(w, h)*0.29)
	return p
}

func TestMaskRectsMergesIdenticalRows(t *testing.T) {
	// A solid block is one rectangle however tall it is: rows with the
	// same runs are merged, which is what keeps a circle's rectangle count
	// near its height rather than far past it.
	mask := make([]uint8, 8*6)
	for y := 1; y < 5; y++ {
		for x := 2; x < 6; x++ {
			mask[y*8+x] = 255
		}
	}
	got := MaskRects(mask, 8, 6, 8, 128)
	want := []FrameRect{{X: 2, Y: 1, W: 4, H: 4}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("solid block: %+v, want %+v", got, want)
	}
}

func TestMaskRectsSplitsARowIntoRuns(t *testing.T) {
	// Two runs in a row are two rectangles, and a row that differs starts
	// a new group: this is the hole case in miniature.
	mask := make([]uint8, 6*3)
	set := func(y int, xs ...int) {
		for _, x := range xs {
			mask[y*6+x] = 255
		}
	}
	set(0, 0, 1, 2, 3, 4, 5)
	set(1, 0, 1, 4, 5) // a hole in the middle
	set(2, 0, 1, 2, 3, 4, 5)
	got := MaskRects(mask, 6, 3, 6, 128)
	want := []FrameRect{
		{X: 0, Y: 0, W: 6, H: 1},
		{X: 0, Y: 1, W: 2, H: 1},
		{X: 4, Y: 1, W: 2, H: 1},
		{X: 0, Y: 2, W: 6, H: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("runs: %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rect %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestMaskRectsRejectsABadMask(t *testing.T) {
	if r := MaskRects(nil, 4, 4, 4, 128); r != nil {
		t.Fatalf("no mask: %+v", r)
	}
	if r := MaskRects(make([]uint8, 4), 4, 4, 2, 128); r != nil {
		t.Fatalf("stride below the width: %+v", r)
	}
	if r := MaskRects(make([]uint8, 3), 4, 4, 4, 128); r != nil {
		t.Fatalf("mask too short: %+v", r)
	}
}

func TestShapeRasterisesAtEveryScale(t *testing.T) {
	// The real test of a shape: a 500x500 logical window at each scale the
	// desktop offers. The shape is built in device pixels, so the hole has
	// to come out the right size and in the right place at 1.75 as at 1.
	const logical = 500
	for _, sc := range []float32{1, 1.25, 1.5, 1.75, 2} {
		dev := int(float32(logical) * sc)
		sh := NewShapeEvenOdd(ringPath(float32(dev), float32(dev)))
		if sh == nil {
			t.Fatalf("scale %v: no shape", sc)
		}
		r := sh.Raster(dev, dev)
		if r == nil || r.Empty() {
			t.Fatalf("scale %v: empty raster", sc)
		}
		if r.W != dev || r.H != dev || len(r.Mask) != dev*dev {
			t.Fatalf("scale %v: raster %dx%d mask %d", sc, r.W, r.H, len(r.Mask))
		}
		// The body is there and the hole is not.
		mid := dev / 2
		if !r.Contains(4, mid) {
			t.Fatalf("scale %v: the left edge of the body is not in the shape", sc)
		}
		if r.Contains(mid, mid) {
			t.Fatalf("scale %v: the middle of the hole is in the shape", sc)
		}
		// The hole's radius scales exactly: 0.29 of the window, within a
		// pixel of rasterisation either way.
		want := float32(dev) * 0.29
		var edge int
		for x := mid; x < dev; x++ {
			if r.Contains(x, mid) {
				edge = x
				break
			}
		}
		if got := float32(edge - mid); got < want-1.5 || got > want+1.5 {
			t.Fatalf("scale %v: the hole reaches %v px, want %v", sc, got, want)
		}
		// The silhouette covers the whole surface's bounds and no more.
		if r.Bounds.X != 0 || r.Bounds.Y != 0 || r.Bounds.W != dev || r.Bounds.H != dev {
			t.Fatalf("scale %v: bounds %+v", sc, r.Bounds)
		}
		// A shape this size stays well inside what a compositor takes: the
		// spike measured 628 rectangles for an 875 px ring.
		if len(r.Rects) > dev {
			t.Fatalf("scale %v: %d rectangles for %d rows", sc, len(r.Rects), dev)
		}
		if r.Clamped {
			t.Fatalf("scale %v: clamped", sc)
		}
	}
}

func TestShapeRasterIsCachedAndAllocationFree(t *testing.T) {
	sh := NewShapeEvenOdd(ringPath(400, 400))
	first := sh.Raster(400, 400)
	if first == nil {
		t.Fatal("no raster")
	}
	if again := sh.Raster(400, 400); again != first {
		t.Fatal("the same size rasterised twice")
	}
	// Repeating it allocates nothing at all: a window hands its shape over
	// every frame, and rasterising a 400 px ring costs tens of
	// milliseconds.
	if n := testing.AllocsPerRun(50, func() { sh.Raster(400, 400) }); n != 0 {
		t.Fatalf("%v allocations a repeat", n)
	}
	// A shape shared between windows of different sizes keeps both warm,
	// rather than thrashing one against the other.
	a, b := sh.Raster(300, 300), sh.Raster(200, 200)
	if sh.Raster(400, 400) != first || sh.Raster(300, 300) != a || sh.Raster(200, 200) != b {
		t.Fatal("the cache dropped a size it still had room for")
	}
}

func TestShapeOpaqueIsOnlyWhatIsFullyCovered(t *testing.T) {
	// The opaque region may never hold a half-covered pixel: a compositor
	// skips what is behind it, and an antialiased edge claimed solid is a
	// hard seam around the window.
	sh := NewShape(ShapeEllipsePath(paintengine2d.XYWH(0, 0, 200, 200)))
	r := sh.Raster(200, 200)
	for _, b := range r.Opaque {
		for _, p := range [][2]int{{b.X, b.Y}, {b.X + b.W - 1, b.Y + b.H - 1}} {
			if c := r.Coverage(p[0], p[1]); c != 255 {
				t.Fatalf("opaque rect %+v holds a pixel with coverage %d", b, c)
			}
		}
	}
	if len(r.Opaque) == 0 {
		t.Fatal("a disc has a solid middle")
	}
	// And the opaque region is inside the input region, never past it.
	for _, b := range r.Opaque {
		if !r.Contains(b.X, b.Y) {
			t.Fatalf("opaque rect %+v starts outside the shape", b)
		}
	}
}

// ShapeEllipsePath is the ellipse filling r as a path (the test's own
// helper: ShapeEllipse hands back a Shape, and this needs the path).
func ShapeEllipsePath(r paintengine2d.Rect) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	p.AddEllipse(paintengine2d.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2), r.Dx()/2, r.Dy()/2)
	return p
}

func TestShapeFromAMaskResamples(t *testing.T) {
	// A skin's silhouette arrives as coverage at the size it was drawn
	// for. Half of a 40x40 mask filled, asked for at 100x100, is still
	// half filled — and the boundary lands where it should.
	const n = 40
	mask := make([]uint8, n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n/2; x++ {
			mask[y*n+x] = 255
		}
	}
	sh := NewShapeMask(mask, n, n, n)
	if sh == nil {
		t.Fatal("no shape from the mask")
	}
	r := sh.Raster(100, 100)
	if !r.Contains(10, 50) || r.Contains(90, 50) {
		t.Fatalf("the mask's half did not survive the resample: %+v", r.Rects)
	}
	// The edge is within a pixel of halfway.
	var edge int
	for x := 0; x < 100; x++ {
		if !r.Contains(x, 50) {
			edge = x
			break
		}
	}
	if edge < 49 || edge > 51 {
		t.Fatalf("the edge is at %d, want about 50", edge)
	}
	// At its own size it is a straight copy, not a resample.
	if own := sh.Raster(n, n); !own.Contains(0, 0) || own.Contains(n-1, 0) {
		t.Fatalf("at its own size: %+v", own.Rects)
	}
}

func TestNewShapeMaskRejectsABadMask(t *testing.T) {
	if NewShapeMask(nil, 4, 4, 4) != nil {
		t.Fatal("no mask")
	}
	if NewShapeMask(make([]uint8, 16), 4, 4, 2) != nil {
		t.Fatal("stride below the width")
	}
	if NewShape(paintengine2d.NewPath()) != nil {
		t.Fatal("an empty path is no shape")
	}
	if NewShape(nil) != nil {
		t.Fatal("no path")
	}
}

func TestShapeFromAnImageTakesItsAlpha(t *testing.T) {
	// The skin case: the silhouette and the artwork are the same file.
	img := paintengine2d.NewImage(8, 8)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	ctx.DrawRect(paintengine2d.XYWH(2, 2, 4, 4), paintengine2d.Fill(paintengine2d.RGBA(0, 0.5, 1, 1)))
	sh := NewShapeImage(img)
	if sh == nil {
		t.Fatal("no shape from the image")
	}
	r := sh.Raster(8, 8)
	if !r.Contains(3, 3) || r.Contains(0, 0) || r.Contains(7, 7) {
		t.Fatalf("the image's alpha is not the shape: %+v", r.Rects)
	}
	// An 8-bit coverage image works the same way (a skin's mask file).
	m := paintengine2d.NewImageA8(8, 8)
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			m.Pix[y*m.RowStride()+x] = 255
		}
	}
	if ar := NewShapeImage(m).Raster(8, 8); !ar.Contains(3, 3) || ar.Contains(0, 0) {
		t.Fatalf("A8: %+v", ar.Rects)
	}
}

func TestScaleRectsGrowsInputAndShrinksOpaque(t *testing.T) {
	// A wl_region speaks logical pixels. The input region has to round
	// outwards, or the window goes deaf along its edge; the opaque region
	// has to round inwards, or it claims a translucent pixel solid. At a
	// fractional scale those are different rectangles, which is the whole
	// point.
	in := []FrameRect{{X: 3, Y: 3, W: 10, H: 10}} // device px
	grown := ScaleRects(in, 1.75, true)
	shrunk := ScaleRects(in, 1.75, false)
	if len(grown) != 1 || len(shrunk) != 1 {
		t.Fatalf("grown %+v shrunk %+v", grown, shrunk)
	}
	g, s := grown[0], shrunk[0]
	if g.X > s.X || g.X+g.W < s.X+s.W {
		t.Fatalf("the grown rect %+v does not hold the shrunk one %+v", g, s)
	}
	// Grown covers every device pixel the shape did, once scaled back.
	if float32(g.X)*1.75 > 3 || float32(g.X+g.W)*1.75 < 13 {
		t.Fatalf("grown %+v loses a device pixel", g)
	}
	// Shrunk claims none the shape did not.
	if float32(s.X)*1.75 < 3 || float32(s.X+s.W)*1.75 > 13 {
		t.Fatalf("shrunk %+v claims a device pixel outside the shape", s)
	}
	// Scale 1 is the identity, and an empty rect is dropped rather than
	// handed over as a zero-sized region.
	if got := ScaleRects(in, 1, true); len(got) != 1 || got[0] != in[0] {
		t.Fatalf("scale 1: %+v", got)
	}
	if got := ScaleRects([]FrameRect{{X: 0, Y: 0, W: 1, H: 1}}, 4, false); len(got) != 0 {
		t.Fatalf("a rect that shrinks to nothing: %+v", got)
	}
}

func TestScaleRectsRoundsNegativeCoordinatesTheRightWay(t *testing.T) {
	// A shape stated in the visible window's coordinates reaches left of
	// and above the surface origin once the shadow margin is taken off.
	// Truncating towards zero would round those the wrong way and lose a
	// row of pixels along the top and left edges.
	in := []FrameRect{{X: -5, Y: -5, W: 10, H: 10}}
	g := ScaleRects(in, 2, true)[0]
	if g.X != -3 || g.Y != -3 {
		t.Fatalf("grown %+v: -5/2 should floor to -3", g)
	}
	s := ScaleRects(in, 2, false)[0]
	if s.X != -2 || s.Y != -2 {
		t.Fatalf("shrunk %+v: -5/2 should ceil to -2", s)
	}
}

func TestOffsetAndBoundsOfRects(t *testing.T) {
	in := []FrameRect{{X: 1, Y: 2, W: 3, H: 4}, {X: 10, Y: 0, W: 2, H: 2}}
	if got := BoundsOfRects(in); got != (FrameRect{X: 1, Y: 0, W: 11, H: 6}) {
		t.Fatalf("bounds %+v", got)
	}
	if got := BoundsOfRects(nil); got != (FrameRect{}) {
		t.Fatalf("bounds of nothing %+v", got)
	}
	got := OffsetRects(in, 5, -1)
	if got[0] != (FrameRect{X: 6, Y: 1, W: 3, H: 4}) || got[1] != (FrameRect{X: 15, Y: -1, W: 2, H: 2}) {
		t.Fatalf("offset %+v", got)
	}
	// No move, no copy.
	if same := OffsetRects(in, 0, 0); &same[0] != &in[0] {
		t.Fatal("a zero offset copied the list")
	}
	if !RectsContain(in, 11, 1) || RectsContain(in, 5, 5) {
		t.Fatal("RectsContain")
	}
}

func TestShapeClampsAPathologicalMask(t *testing.T) {
	// A dithered mask would rasterise to a rectangle a pixel: a mistake in
	// the mask, not a silhouette anyone meant. The shape reports its
	// bounding box instead of handing a compositor hundreds of thousands
	// of rectangles, and says so.
	const n = 400
	mask := make([]uint8, n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if (x+y)%2 == 0 {
				mask[y*n+x] = 255
			}
		}
	}
	r := NewShapeMask(mask, n, n, n).Raster(n, n)
	if !r.Clamped {
		t.Fatalf("a checkerboard made %d rectangles and was not clamped", len(r.Rects))
	}
	if len(r.Rects) != 1 || r.Rects[0] != r.Bounds {
		t.Fatalf("clamped to %+v, want the bounds %+v", r.Rects, r.Bounds)
	}
	// The mask itself is still exact: Capture and the shadow use it.
	if r.Coverage(0, 0) != 255 || r.Coverage(1, 0) != 0 {
		t.Fatal("the clamp damaged the coverage mask")
	}
}

func TestFrameSameComparesRegions(t *testing.T) {
	// Frame stopped being comparable with == when it grew regions, and
	// every backend decides whether to send anything by comparing.
	a := Frame{Alpha: true, Shape: []FrameRect{{W: 2, H: 2}}}
	b := Frame{Alpha: true, Shape: []FrameRect{{W: 2, H: 2}}}
	if !a.Same(b) {
		t.Fatal("equal frames differ")
	}
	b.Shape = []FrameRect{{W: 3, H: 2}}
	if a.Same(b) {
		t.Fatal("different shapes are the same")
	}
	// "No region" and "the empty region" are different requests: the first
	// leaves the surface's default, the second says nothing is there.
	none := Frame{}
	empty := Frame{Shape: []FrameRect{}}
	if none.Same(empty) {
		t.Fatal("a nil shape is not an empty shape")
	}
	if !none.Zero() || empty.Zero() {
		t.Fatal("Zero should tell an unshaped frame from an empty-shaped one")
	}
	if !(Frame{Blur: []FrameRect{}}).Same(Frame{Blur: []FrameRect{}}) {
		t.Fatal("two empty blur regions differ")
	}
}

func TestShapeRasterisesInStripsWithoutASeam(t *testing.T) {
	// A shape larger than one scratch strip is drawn a strip at a time, so
	// its peak memory does not follow its size. The joins must leave
	// nothing behind: a seam would be a row of pixels the compositor is
	// told the window does not cover, straight across it.
	const n = 900
	strip := shapeScratchBytes / (n * 4)
	if strip >= n {
		t.Fatalf("a %d px shape fits in one strip (%d rows): nothing joins", n, strip)
	}
	p := paintengine2d.NewPath()
	p.AddRect(paintengine2d.XYWH(0, 0, n, n))
	r := NewShape(p).Raster(n, n)
	// A full rectangle is one merged rectangle, however many strips it
	// took: every row has the same run, seam or no seam.
	if len(r.Rects) != 1 || r.Rects[0] != (FrameRect{X: 0, Y: 0, W: n, H: n}) {
		t.Fatalf("a full square rasterised to %+v", r.Rects)
	}
	for y := 0; y < n; y++ {
		if r.Mask[y*n+n/2] != 255 {
			t.Fatalf("row %d of the mask is not covered (strip height %d)", y, strip)
		}
	}
	// And a shape with a hole keeps the hole exactly where it belongs
	// across a join.
	ring := NewShapeEvenOdd(ringPath(n, n)).Raster(n, n)
	if ring.Contains(n/2, n/2) {
		t.Fatal("the hole closed up")
	}
	for y := strip - 2; y <= strip+2 && y < n; y++ {
		if !ring.Contains(2, y) {
			t.Fatalf("row %d at a strip join lost the body", y)
		}
	}
}
