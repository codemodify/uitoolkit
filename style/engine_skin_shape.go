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
// to, where in that box it was drawn, and the display scale that fitted it.
type skinHitKey struct {
	sprite *SkinSprite
	w, h   int
	at     paintengine2d.Rect
	scale  float32
	// panel is whether the sprite is painted as a panel's piece
	// (panelDraw) rather than as a control's face.
	panel bool
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
	// The same box the painter uses, padding and all, so the silhouette and
	// the picture are the same shape in the same place.
	size := paintengine2d.Pt(b.Dx(), b.Dy())
	return sk.shapeOf(l, sp, size, sk.box(l, paintengine2d.XYWH(0, 0, size.X, size.Y), p), false)
}

// shapeOf is the silhouette of sprite sp drawn at at inside a box of size,
// or nil for the whole box. panel paints it as a panel's piece is painted
// (panelDraw), so the silhouette and the picture agree.
func (sk *Skin) shapeOf(l *Classic, sp *SkinSprite, size paintengine2d.Point, at paintengine2d.Rect, panel bool) *Silhouette {
	w, h := int(size.X+0.5), int(size.Y+0.5)
	if sp == nil || w < 1 || h < 1 || w*h > skinHitMax {
		return nil
	}
	mask := sk.hitMask(l, sp, w, h, at, panel)
	if mask == nil {
		return nil
	}
	return &Silhouette{Mask: mask}
}

// SkinSpriteShape is the silhouette of the named sprite as DrawSkinSprite
// paints it at at, inside a control whose box is size (device pixels,
// origin at the box's top left) — or nil for the whole box: the look is not
// a skin, it has no such sprite, the art fills the box, or it could not be
// drawn.
//
// It is the hit shape of a control an app paints from a sprite rather than
// from one of the look's faces. A round key a panel paints by name is round
// to the pointer too, exactly as a skinned push button is: the corners of
// its box fall through to whatever is behind.
func SkinSpriteShape(lk LookAndFeel, size paintengine2d.Point, at paintengine2d.Rect, name string) *Silhouette {
	l, sk := lookSkin(lk)
	if sk == nil {
		return nil
	}
	return sk.shapeOf(l, sk.Sprites[name], size, at, true)
}

// hitMask is the coverage of one face at one size, from the cache when it
// has been asked for before. A nil answer is cached too: "this face fills
// its box" is the common case and re-deriving it would be the expensive way
// to learn nothing.
func (sk *Skin) hitMask(l *Classic, sp *SkinSprite, w, h int, at paintengine2d.Rect, panel bool) *paintengine2d.Image {
	key := skinHitKey{sprite: sp, w: w, h: h, at: at, scale: l.Scale(), panel: panel}

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
	m := sk.paintHitMask(l, sp, w, h, at, panel)

	skinCache.mu.Lock()
	if len(skinCache.hits) >= skinHitCacheMax {
		skinCache.hits = map[skinHitKey]*paintengine2d.Image{}
	}
	skinCache.hits[key] = m
	skinCache.mu.Unlock()
	return m
}

// paintHitMask draws the sprite at at into a scratch the size of the
// control and hands back its coverage, or nil where there is nothing useful
// to say.
func (sk *Skin) paintHitMask(l *Classic, sp *SkinSprite, w, h int, at paintengine2d.Rect, panel bool) *paintengine2d.Image {
	scratch := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(scratch)
	if ctx == nil {
		return nil
	}
	ctx.Clear(paintengine2d.Transparent)
	draw := sk.skinDraw
	if panel {
		draw = sk.panelDraw
	}
	if !draw(ctx, at, sp, l.Scale(), paintengine2d.Color{}) {
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
