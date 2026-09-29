package style

import (
	"sync"

	"github.com/codemodify/paintengine2d"
)

// Whether a pack paints a default button differently from an ordinary
// one, and what to do when it does not.
//
// [ControlState] carries Primary — "this is the button Enter presses" —
// and the core chrome answers it with the accent fill. An engine that
// paints its own buttons is free to ignore it, and several do: Plastik,
// KDE 3's default style and so the toolkit's own default pack, paints
// every button the same. A consent dialog that makes *Deny* the default,
// so that an Enter pressed without reading refuses, then shows nothing
// at all about which button that is.
//
// Which engines ignore it is not a list worth keeping by hand: there are
// 33 of them, a list would drift, and an engine that gains a default
// face would then get two marks. So it is **measured**: a button is
// drawn twice, once Primary and once not, into a small offscreen image,
// and the answer is whether the two differ. Once per pack, cached.
//
// That is the same instrument this repository uses for engine
// entanglement — ask the thing itself rather than maintain a table about
// it (docs/engines.md).

var primaryFace struct {
	mu sync.Mutex
	m  map[string]bool
}

// PrimaryFaceOf reports whether this look's engine paints a Primary
// button differently from an ordinary one.
//
// A false means the pack needs [DrawDefaultMark] to show which button
// Enter presses.
func PrimaryFaceOf(l LookAndFeel) bool {
	if l == nil {
		return true
	}
	// Keyed on the *pack*, not on Name — which is the palette family,
	// so every pack in a build would have shared one answer and the
	// first one measured would have decided for all of them.
	key := l.Name()
	if c, ok := l.(*Classic); ok {
		key = c.Pack()
	}
	primaryFace.mu.Lock()
	if got, ok := primaryFace.m[key]; ok {
		primaryFace.mu.Unlock()
		return got
	}
	primaryFace.mu.Unlock()

	got := measurePrimaryFace(l)

	primaryFace.mu.Lock()
	if primaryFace.m == nil {
		primaryFace.m = map[string]bool{}
	}
	primaryFace.m[key] = got
	primaryFace.mu.Unlock()
	return got
}

// measurePrimaryFace draws the same button twice and asks whether the
// engine drew two different things.
//
// It calls the **engine** rather than [Classic.DrawButton], and that is
// not an optimisation: DrawButton is where the ring is added, so
// measuring through it would ask this question about its own answer and
// recurse until the stack ran out. What is being measured is what the
// engine does with the flag, which is the question.
//
// Both renders are at 1x into the same size, with the same label, and
// differ only in StatePrimary. The tolerance is one pixel rather than
// zero, because a pack that shifts a bevel by a fraction and rounds it
// away has not marked anything a user can see.
func measurePrimaryFace(l LookAndFeel) bool {
	c, ok := l.(*Classic)
	if !ok {
		// Not this package's look, so it never goes through the wrapper
		// that would add a ring: whatever it paints is its own business.
		return true
	}
	const w, h = 72, 26
	draw := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(l.Palette().Background))
		c.eng().DrawButton(c, ctx, paintengine2d.XYWH(2, 2, w-4, h-4), st, "OK")
		img.Touch()
		return img
	}
	plain, prim := draw(StateNone), draw(StatePrimary)
	if plain.Width != prim.Width || plain.Height != prim.Height {
		return true
	}
	return visiblyDifferent(plain, prim) >= primaryFaceMinPixels
}

// primaryFaceMinPixels is how many visibly different pixels count as
// "this pack marks its default button".
//
// It is not one pixel, because a pack that shifts a bevel by a fraction
// and rounds it away has marked nothing a user can see. It is measured
// rather than guessed: across the 131 shipped packs the answers are
// **zero** for the two Amiga ones and **144 or more** for every other,
// so anything between the two separates them with room to spare.
const primaryFaceMinPixels = 24

// visiblyDifferent counts the pixels of a and b that differ by more than
// a rounding error — 4 of 255 on the mean channel, which is below what
// anyone sees and above what anti-aliasing shifts around.
func visiblyDifferent(a, b *paintengine2d.Image) int {
	n := 0
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			d := absDiff(r1, r2) + absDiff(g1, g2) + absDiff(b1, b2)
			if d/3/257 > 4 {
				n++
			}
		}
	}
	return n
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// DrawDefaultMark marks the default button for a pack whose engine does
// not, and does nothing for a pack that does.
//
// It is a plain rectangle just inside the button's edge, on whole
// pixels, in the palette's own text colour — which is deliberately not
// the accent and not rounded. The two packs that need it are Amiga
// Workbench's, drawn through the raster port at four colours a screen
// ([enginekit_pixel.go]), and a rounded accent ring would have put
// anti-aliasing and a fifth colour into a pack whose whole point is that
// it has neither. A heavier border is also what that era actually drew
// around a default button.
func DrawDefaultMark(l LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	if l == nil || ctx == nil || !st.Primary() || st.Disabled() || PrimaryFaceOf(l) {
		return
	}
	col := l.Palette().Text
	if colorUnset(col) {
		col = l.Palette().Accent
	}
	r := snapRectDevice(ctx, b.Inset(2))
	if r.Dx() < 3 || r.Dy() < 3 {
		return
	}
	paint := paintengine2d.Fill(col)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), 1), paint)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Max.Y-1, r.Dx(), 1), paint)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y, 1, r.Dy()), paint)
	ctx.DrawRect(paintengine2d.XYWH(r.Max.X-1, r.Min.Y, 1, r.Dy()), paint)
}

// snapRectDevice rounds a box onto whole **device** pixels and gives it
// back in the context's own coordinates.
//
// Snapping the local coordinates is not enough: layout leaves a control
// at a fractional origin and the context carries that as a translation,
// so a rectangle on whole local pixels still lands between device ones
// and is anti-aliased. That is how the first version of this put a fifth
// colour into a four-colour Workbench screen. It is the same correction
// the raster port makes (rpOrigin) and that Font.Draw makes for glyphs.
func snapRectDevice(ctx *paintengine2d.Context, b paintengine2d.Rect) paintengine2d.Rect {
	var ox, oy float32
	if m := ctx.Matrix(); m.A == 1 && m.D == 1 && m.B == 0 && m.C == 0 {
		ox, oy = m.E, m.F
	}
	return paintengine2d.Rect{
		Min: paintengine2d.Pt(snap(b.Min.X+ox)-ox, snap(b.Min.Y+oy)-oy),
		Max: paintengine2d.Pt(snap(b.Max.X+ox)-ox, snap(b.Max.Y+oy)-oy),
	}
}
