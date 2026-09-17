package style

import (
	"github.com/codemodify/paintengine2d"
)

// The silhouette of a skinned control.
//
// A skin's face is a picture, and a picture knows where it is. So the answer
// to "does this press belong to the button" is a read of the art the button
// was painted from: a round face takes the pointer where it is round and the
// corners of its box fall through to whatever is behind, the same rule a
// shaped window follows one layer up (docs/shapes.md, widget/shape.go).
//
// The mask is taken by *painting* the sprite into a scratch the size of the
// control rather than by reading the sheet: a face is a nine-slice, so where
// its ink lands depends on how its corners and middles were fitted to this
// particular box, and only the painter knows that. It costs one image the
// size of the control, once per sprite and size, and never on a frame — the
// cut and the cache below are shared with the sheets.

// skinHitCoverage is how much of a pixel the art must cover for the pixel to
// be the control's.
//
// It is deliberately near the bottom rather than at half. The question this
// answers is "did the art mark this pixel at all", because what it is for is
// refusing the *empty* corners of a round face; adjudicating translucency is
// not its job, and a threshold at half would make a face drawn as a pale
// wash — a ghost button, a glassy toggle — entirely unclickable. Rounding
// this far out also rounds an antialiased edge outwards, which is the right
// way to round for input: never lose a pixel the control covers.
const skinHitCoverage uint8 = 32

// skinHitMax is the largest face this bothers with, in pixels. A control
// bigger than a 512-pixel square is a panel, a bar or a window background,
// and cutting input out of one of those is neither what a skin author means
// nor worth an image that size.
const skinHitMax = 512 * 512

// skinHitCacheMax caps the derived masks kept at once. A skin has a dozen
// faces at a handful of sizes, so the cap is only ever reached by an app
// that resizes a skinned control through hundreds of widths — and then the
// oldest answers are the least likely to be asked for again.
const skinHitCacheMax = 256

// skinHitKey identifies one derived mask: the sprite, the box it was fitted
// to, and the display scale that fitted it.
type skinHitKey struct {
	sprite *SkinSprite
	w, h   int
	scale  float32
}

// ControlShape is the silhouette of the face this skin paints for role in
// box b, or nil for the whole box — which is the answer for a part the skin
// does not bind, for art that fills its box solidly (the usual case, and the
// cheapest), and for art that could not be drawn at all.
//
// The last of those is deliberate: a sheet that failed to load leaves a
// control that is square, not one that cannot be clicked.
func (skinEngine) ControlShape(l *Classic, b paintengine2d.Rect, role Role) *Silhouette {
	sk := skinFor(l)
	if sk == nil {
		return nil
	}
	name, ok := skinRolePart[role]
	if !ok {
		return nil
	}
	p := sk.part(name)
	// The resting face, always: see ControlShapeEngine on why a hit area may
	// not follow the pointer.
	sp := p.art("normal")
	if sp == nil {
		return nil
	}
	w, h := int(b.Dx()+0.5), int(b.Dy()+0.5)
	if w < 1 || h < 1 || w*h > skinHitMax {
		return nil
	}
	mask := sk.hitMask(l, p, sp, w, h)
	if mask == nil {
		return nil
	}
	return &Silhouette{Mask: mask}
}

// hitMask is the coverage of one face at one size, from the cache when it
// has been asked for before. A nil answer is cached too: "this face fills
// its box" is the common case and re-deriving it would be the expensive way
// to learn nothing.
func (sk *Skin) hitMask(l *Classic, p *SkinPart, sp *SkinSprite, w, h int) *paintengine2d.Image {
	key := skinHitKey{sprite: sp, w: w, h: h, scale: l.Scale()}

	skinCache.mu.Lock()
	skinCache.syncGen()
	if m, ok := skinCache.hits[key]; ok {
		skinCache.mu.Unlock()
		return m
	}
	skinCache.mu.Unlock()

	// Painting happens outside the lock, as the sheet cut does: two
	// goroutines racing on a first hit test may both paint, and either
	// answer is the same picture.
	m := sk.paintHitMask(l, p, sp, w, h)

	skinCache.mu.Lock()
	if len(skinCache.hits) >= skinHitCacheMax {
		skinCache.hits = map[skinHitKey]*paintengine2d.Image{}
	}
	skinCache.hits[key] = m
	skinCache.mu.Unlock()
	return m
}

// paintHitMask draws the face into a scratch of its own size and hands back
// its coverage, or nil where there is nothing useful to say.
func (sk *Skin) paintHitMask(l *Classic, p *SkinPart, sp *SkinSprite, w, h int) *paintengine2d.Image {
	scratch := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(scratch)
	if ctx == nil {
		return nil
	}
	ctx.Clear(paintengine2d.Transparent)
	// The same box the painter uses, padding and all, so the silhouette and
	// the picture are the same shape in the same place.
	box := sk.box(l, paintengine2d.XYWH(0, 0, float32(w), float32(h)), p)
	if !sk.skinDraw(ctx, box, sp, l.Scale(), paintengine2d.Color{}) {
		return nil
	}
	mask := paintengine2d.NewImageA8(w, h)
	src, dst := scratch.RowStride(), mask.RowStride()
	in, out := 0, 0
	for y := 0; y < h; y++ {
		row := scratch.Pix[y*src:]
		to := mask.Pix[y*dst:]
		for x := 0; x < w; x++ {
			if row[x*4+3] >= skinHitCoverage { // premultiplied RGBA8: alpha last
				to[x] = 255
				in++
				continue
			}
			to[x] = 0
			out++
		}
	}
	if out == 0 || in == 0 {
		// Solid to its edges, or nothing at all. Both mean "the box", and
		// saying so costs a comparison instead of a rasterisation.
		return nil
	}
	mask.Touch()
	return mask
}
