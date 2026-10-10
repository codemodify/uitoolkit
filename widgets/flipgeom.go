package widgets

import "math"

// The projection a [FlipPanel] turns by: a flat face rotating about its
// own vertical axis, seen through a pinhole camera.
//
// Rotation about the *vertical* axis is the one case a 2D engine can draw
// exactly. Take a point on the face at (u, v), both in -1..1 from the
// centre. Turning by θ about the vertical axis and dividing by the depth
// gives
//
//	z(u)   = u·sinθ
//	s(u)   = d / (d + u·sinθ)
//	x(u)   = u·cosθ · s(u)
//	y(u,v) = v · s(u)
//
// — and the only term in y that varies with u is s(u). So every source
// *column* lands on a vertical line, scaled about the centre by a factor
// that depends on the column alone: no rotation, no shear, no perspective
// divide per pixel. A column is an axis-aligned rectangle scaled in y,
// which is the one thing [paintengine2d.Context.DrawImageRect] already
// draws. That is why this needs nothing from the engine.
//
// Cutting the face into n columns is therefore exact in y, and exact in x
// at every column edge; the approximation is only that s is held constant
// across one column's width instead of varying hyperbolically, which goes
// as the square of that width.
type flipGeom struct {
	// W, H are the face's box in its own (panel-local) space.
	W, H float32
	// D is the camera distance in half-widths, already clamped.
	D float32
	// Sin, Cos are of the turn angle, unscaled.
	Sin, Cos float32
	// K scales the whole projected frame so the face never leaves its box
	// (see flipGeomAt).
	K float32
}

// flipPerspective bounds a camera distance, in half-widths.
//
// It must stay above 1, the face's own half-width, or the near edge
// reaches the pinhole and the projection blows up. Below about 1.25
// nothing reads as a turn any more, and past 100 the projection is affine
// to within a pixel — the foreshortening this is all for has gone.
func flipPerspective(d float32) float32 {
	switch {
	case d != d || !(d > 0): // NaN, unset, negative
		return flipPerspectiveDefault
	case d < 1.25:
		return 1.25
	case d > 100:
		return 100
	}
	return d
}

// flipGeomAt is the projection of a face of w x h turned through angle
// (radians), normalised so the face always lands inside its own box.
//
// **The normalisation is what keeps the card inside the panel.** s is
// largest at the far edge, u = -sign(sinθ), where it is d/(d-|sinθ|) —
// greater than 1 for every angle but the two ends. Un-normalised, the
// card would be drawn taller than the box it lives in and the panel's own
// clip would cut it. Scaling the frame by k = (d-|sinθ|)/d makes that
// largest s exactly 1: at rest k is 1 and the face is its box to the
// pixel, and mid-turn the card draws smaller, which reads as it receding
// as it comes edge-on. Dashboard's widgets pulled back through their flip
// the same way.
func flipGeomAt(w, h, d, angle float32) flipGeom {
	sin, cos := math.Sincos(float64(angle))
	d = flipPerspective(d)
	return flipGeom{
		W: w, H: h, D: d,
		Sin: float32(sin), Cos: float32(cos),
		K: (d - float32(math.Abs(sin))) / d,
	}
}

// scaleAt is s(u): how much the column at u is scaled about the centre,
// normalisation included. It is never above 1, which is what guarantees
// the card fits.
func (g flipGeom) scaleAt(u float32) float32 {
	den := g.D + u*g.Sin
	if den <= 0 {
		return 0
	}
	return g.K * g.D / den
}

// xAt is where the column at u lands, in panel-local pixels.
func (g flipGeom) xAt(u float32) float32 {
	return g.W/2 + (g.W/2)*u*g.Cos*g.scaleAt(u)
}

// halfHAt is the column's half-height at u, in panel-local pixels.
func (g flipGeom) halfHAt(u float32) float32 {
	return (g.H / 2) * g.scaleAt(u)
}

// edgeOn reports that the projected face is under a pixel wide, where
// there is nothing left to draw: at θ = 90° cosθ is 0 and every column
// lands on the same vertical line.
func (g flipGeom) edgeOn() bool {
	return absF(g.xAt(1)-g.xAt(-1)) < 1
}

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
