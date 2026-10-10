package widgets

import (
	"math"
	"testing"
)

const (
	geomW = 400
	geomH = 300
)

// closedForm is the projection written out independently of flipGeom, from
// the formulas in flipgeom.go's own doc comment. A test that asked flipGeom
// to agree with itself would pass however the projection was rewritten, so
// the maths is spelled out a second time here in float64 and the two are
// compared.
func closedForm(u, d, angle float64) (x, halfH float64) {
	sin, cos := math.Sincos(angle)
	k := (d - math.Abs(sin)) / d
	s := k * d / (d + u*sin)
	return geomW/2 + (geomW/2)*u*cos*s, (geomH / 2) * s
}

// The projection agrees with the closed form at every angle and column.
func TestFlipProjectionIsItsClosedForm(t *testing.T) {
	const d = flipPerspectiveDefault
	for deg := 0; deg <= 180; deg += 5 {
		angle := float64(deg) * math.Pi / 180
		g := flipGeomAt(geomW, geomH, d, float32(angle))
		for i := 0; i <= 20; i++ {
			u := 2*float64(i)/20 - 1
			wantX, wantH := closedForm(u, d, angle)
			if gotX := float64(g.xAt(float32(u))); math.Abs(gotX-wantX) > 1e-3 {
				t.Fatalf("%d°, u=%+.2f: x is %.4f, the formula says %.4f", deg, u, gotX, wantX)
			}
			if gotH := float64(g.halfHAt(float32(u))); math.Abs(gotH-wantH) > 1e-3 {
				t.Fatalf("%d°, u=%+.2f: half-height is %.4f, the formula says %.4f", deg, u, gotH, wantH)
			}
		}
	}
}

// At rest the face is its box to the pixel, and at a half turn it is its
// box mirrored. Anything else would mean a panel that does not sit still
// where it started, or lands off-centre when the turn ends.
func TestFlipAtRestIsExactlyTheBox(t *testing.T) {
	for _, tc := range []struct {
		name         string
		angle        float64
		leftU, randU float32
	}{
		{"flat", 0, -1, 1},
		{"half turn", math.Pi, 1, -1},
	} {
		g := flipGeomAt(geomW, geomH, 0, float32(tc.angle))
		if got := g.xAt(tc.leftU); math.Abs(float64(got)) > 1e-3 {
			t.Errorf("%s: the left edge is at %.4f, want 0", tc.name, got)
		}
		if got := g.xAt(tc.randU); math.Abs(float64(got-geomW)) > 1e-3 {
			t.Errorf("%s: the right edge is at %.4f, want %d", tc.name, got, geomW)
		}
		// Flat means flat: every column is full height, so there is no
		// foreshortening to see at either end of the turn.
		for i := 0; i <= 8; i++ {
			u := 2*float32(i)/8 - 1
			if got := g.halfHAt(u); math.Abs(float64(got-geomH/2)) > 1e-3 {
				t.Errorf("%s: u=%+.2f is %.4f high, want %d", tc.name, u, 2*got, geomH)
			}
		}
	}
}

// **The test that tells a perspective flip from an affine squeeze.** Mid
// turn the near edge is drawn taller than the far one, and by the ratio
// the projection says. An affine transform cannot do this at all: it maps
// parallel lines to parallel lines, so its columns are all one height and
// this fails.
func TestFlipForeshortensTheFarEdge(t *testing.T) {
	const d = flipPerspectiveDefault
	for _, deg := range []int{30, 45, 60, 120, 150} {
		angle := float64(deg) * math.Pi / 180
		g := flipGeomAt(geomW, geomH, d, float32(angle))
		// The sine stays positive right through the turn, so the near edge
		// is the same one throughout; it is the face that flips over.
		near, far := g.halfHAt(-1), g.halfHAt(1)
		if !(near > far) {
			t.Fatalf("%d°: near edge %.3f is not taller than far edge %.3f — this is an affine squeeze, not a turn", deg, near, far)
		}
		// d/(d-sin) against d/(d+sin): the normalisation is common to both
		// and cancels.
		sin := math.Abs(math.Sin(angle))
		want := (d + sin) / (d - sin)
		if got := float64(near / far); math.Abs(got-want) > 1e-3 {
			t.Errorf("%d°: the near edge is %.4f times the far one, the projection says %.4f", deg, got, want)
		}
	}
}

// The card never leaves the box it was given, at any angle: the panel
// clips to its own bounds, so a projection that overflowed would be drawn
// cut off instead of smaller. This is what the normalisation in flipGeomAt
// is for.
func TestFlipNeverLeavesItsBox(t *testing.T) {
	for _, d := range []float32{1.25, 1.5, flipPerspectiveDefault, 10, 100} {
		for deg := 0; deg <= 180; deg++ {
			g := flipGeomAt(geomW, geomH, d, float32(float64(deg)*math.Pi/180))
			for i := 0; i <= 40; i++ {
				u := 2*float32(i)/40 - 1
				x, half := g.xAt(u), g.halfHAt(u)
				if x < -0.01 || x > geomW+0.01 {
					t.Fatalf("d=%.2f %d°: u=%+.2f lands at x=%.4f, outside 0..%d", d, deg, u, x, geomW)
				}
				if half > geomH/2+0.01 {
					t.Fatalf("d=%.2f %d°: u=%+.2f is %.4f high, taller than the box's %d", d, deg, u, 2*half, geomH)
				}
			}
		}
	}
}

// Columns tile: x is monotonic in u, so neighbouring strips meet without
// overlapping or leaving a gap. A projection whose x doubled back would
// draw columns over one another and the card would tear.
func TestFlipColumnsTile(t *testing.T) {
	for deg := 0; deg <= 180; deg += 3 {
		g := flipGeomAt(geomW, geomH, flipPerspectiveDefault, float32(float64(deg)*math.Pi/180))
		if deg == 90 {
			continue // edge on: every column lands on one line
		}
		rising := g.xAt(1) > g.xAt(-1)
		prev := g.xAt(-1)
		for i := 1; i <= 64; i++ {
			x := g.xAt(2*float32(i)/64 - 1)
			if rising && x < prev-1e-4 || !rising && x > prev+1e-4 {
				t.Fatalf("%d°: x doubles back at column %d (%.4f after %.4f)", deg, i, x, prev)
			}
			prev = x
		}
	}
}

// A camera distance is kept above the face's own half-width, where the
// near edge would reach the pinhole and the projection blow up.
func TestFlipPerspectiveIsClamped(t *testing.T) {
	for _, tc := range []struct {
		in, want float32
	}{
		{0, flipPerspectiveDefault},
		{-5, flipPerspectiveDefault},
		{float32(math.NaN()), flipPerspectiveDefault},
		{0.5, 1.25},
		{1, 1.25},
		{2.5, 2.5},
		{1e6, 100},
	} {
		if got := flipPerspective(tc.in); got != tc.want {
			t.Errorf("flipPerspective(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
	// The clamp is what keeps the projection finite: at the worst angle a
	// clamped distance still scales by a real, bounded number.
	g := flipGeomAt(geomW, geomH, 0.001, float32(math.Pi/2))
	if h := g.halfHAt(-1); !(h > 0) || h > geomH/2+0.01 {
		t.Errorf("a camera at the pinhole gives a half-height of %v", h)
	}
}

// Edge on there is nothing to draw, and the panel says so rather than
// drawing columns of no width.
func TestFlipSaysWhenItIsEdgeOn(t *testing.T) {
	if g := flipGeomAt(geomW, geomH, 0, float32(math.Pi/2)); !g.edgeOn() {
		t.Errorf("at 90° the face is not reported edge on (width %.4f)", g.xAt(1)-g.xAt(-1))
	}
	for _, deg := range []int{0, 30, 60, 120, 180} {
		if g := flipGeomAt(geomW, geomH, 0, float32(float64(deg)*math.Pi/180)); g.edgeOn() {
			t.Errorf("at %d° the face is reported edge on", deg)
		}
	}
}
