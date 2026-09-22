package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// Drawing a sprite into a box.
//
// This is the whole painting surface of a skin: one function that puts a
// named sprite in a rect at a display scale. Everything the engine does is a
// call to it.
//
// Nine-slice is not optional here. A toolkit button is not a fixed 23×18 —
// it is as wide as its label in whatever language the user reads — so a
// skin's art has to survive being stretched. The corners keep their size and
// snap to whole device pixels (a 1px highlight in a corner survives; the
// same highlight stretched across a whole button does not), the edges grow
// along one axis, and the middle stretches or tiles. It is the single
// feature that separates a skin for a *toolkit* from a skin for one app's
// fixed window, and it is why this format can do what WinAmp's could not.

// skinDraw paints sprite sp into b at the look's display scale.
//
// tint, when set, replaces the art's colour and keeps its alpha as coverage
// — how a single-colour glyph (an arrow, a tick) follows the label colour
// instead of being redrawn per state.
func (sk *Skin) skinDraw(ctx *paintengine2d.Context, b paintengine2d.Rect, sp *SkinSprite, scale float32, tint paintengine2d.Color) bool {
	if sk == nil || ctx == nil || sp == nil || b.Empty() {
		return false
	}
	v := sk.variant(sp, sk.assetTarget(sp, scale))
	if v == nil {
		return false
	}
	paint := paintengine2d.Paint{Filter: paintengine2d.FilterBilinear}
	if sp.Sheet != nil && sp.Sheet.Pixelated {
		paint.Filter = paintengine2d.FilterNearest
	}
	if sp.Tint && !colorUnset(tint) {
		paint.Color = tint
	}
	if !sp.Sliced() {
		// An unsliced sprite that tiles is a texture: a dither, a hatch, a
		// grille, a window background. It repeats at its own size rather
		// than stretching, which is the whole reason a skin says "tile".
		if sp.Middle == FillTile {
			return tilePiece(ctx, v.pieces[pieceWhole], b, sk.texelScale(sp, v, scale), true, true, paint)
		}
		return blitPiece(ctx, v.pieces[pieceWhole], sk.fitBox(b, v, sp, scale), paint)
	}
	return sk.drawSlice(ctx, b, v, sp, scale, paint)
}

// panelDraw paints a sprite that is a piece of a panel — an app's sprite
// (DrawSkinSprite) or a layout's art (DrawSkinSlot, DrawSkinLayout) — into
// b. It is skinDraw, with one addition for pixel art.
//
// A pixel sprite drawn at its own size — a panel's key, a digit, a whole
// face — is magnified by the display scale itself, nearest, on the device
// grid (drawPixelGrid), rather than drawn at a whole multiple and centred:
// a panel's pieces are placed by design coordinates and have to meet, and a
// key drawn at 1× in the middle of a hole drawn at 1.75 does not meet
// anything. Its slices, if it has any, are for boxes of other sizes; at its
// own size there is nothing for them to absorb. A control face keeps the
// whole-multiple rule (skinDraw), which is what a pixel skin's buttons,
// fields and checks were drawn for.
func (sk *Skin) panelDraw(ctx *paintengine2d.Context, b paintengine2d.Rect, sp *SkinSprite, scale float32, tint paintengine2d.Color) bool {
	if sk == nil || ctx == nil || sp == nil || b.Empty() {
		return false
	}
	if sp.Sheet == nil || !sp.Sheet.Pixelated || !sk.ownSize(b, sp, scale) {
		return sk.skinDraw(ctx, b, sp, scale, tint)
	}
	v := sk.variant(sp, sk.assetTarget(sp, scale))
	if v == nil {
		return false
	}
	paint := paintengine2d.Paint{Filter: paintengine2d.FilterNearest}
	if sp.Tint && !colorUnset(tint) {
		paint.Color = tint
	}
	return drawPixelGrid(ctx, v.pieces[pieceWhole], b, paint)
}

// ownSize reports whether b is the sprite's own size at the display scale,
// to within half a device pixel either way.
func (sk *Skin) ownSize(b paintengine2d.Rect, sp *SkinSprite, scale float32) bool {
	s := scale / sk.Design.Scale
	dw, dh := b.Dx()-sp.W*s, b.Dy()-sp.H*s
	return dw > -0.5 && dw < 0.5 && dh > -0.5 && dh < 0.5
}

// drawPixelGrid magnifies a pixel-art piece into b, nearest, with every
// texel landing on whole device pixels on one grid shared by everything
// drawn this way in the same window.
//
// At a whole scale that is plain doubling. At 1.25, 1.5 and 1.75 it cannot
// be even — a texel at 1.75 covers two device pixels three times in four
// and one the fourth, because 1.75 pixels do not exist — so the question is
// only *which* texels get the narrow column, and the answer here is the
// same everywhere: a texel whose left edge is at device x covers the pixels
// whose centres fall in [x, x + k). A panel's pieces all have boxes on the
// same design grid, so a key's texels continue its face's exactly, with no
// double-width or missing column where one sprite meets the next — which is
// what made the panels look uneven before, when each piece was fitted to a
// rounded box of its own with a stretch of its own.
//
// The picture is magnified here, on the CPU, into an image whose pixels are
// the device's, and blitted one to one at a whole pixel: the same answer on
// the CPU and the GPU (which would antialias a quad's fractional edge), and
// the same answer for a pixel whose centre falls exactly on a texel edge —
// which at 1.25, 1.5 and 1.75 is most of them, since design coordinates
// times those scales are exact binary fractions.
func drawPixelGrid(ctx *paintengine2d.Context, img *paintengine2d.Image, b paintengine2d.Rect, paint paintengine2d.Paint) bool {
	if ctx == nil || img == nil || b.Empty() {
		return false
	}
	paint.Filter = paintengine2d.FilterNearest
	m := ctx.Matrix()
	if !m.IsTranslation() {
		ctx.DrawImageRectPaint(img, pieceRect(img), b, paint)
		return true
	}
	// The box in device pixels, and the texel size along each axis.
	x0, y0 := b.Min.X+m.E, b.Min.Y+m.F
	kx, ky := b.Dx()/float32(img.Width), b.Dy()/float32(img.Height)
	g := pixelGridImage(img, x0, y0, kx, ky)
	if g == nil {
		return false
	}
	dst := paintengine2d.XYWH(float32(g.px)-m.E, float32(g.py)-m.F, float32(g.img.Width), float32(g.img.Height))
	ctx.DrawImageRectPaint(g.img, pieceRect(g.img), dst, paint)
	return true
}

// pixelGrid is a piece magnified onto the device grid: the image, and the
// device pixel its top-left pixel goes on.
type pixelGrid struct {
	img    *paintengine2d.Image
	px, py int
}

// pixelGridKey identifies one magnification: the piece, the texel size, and
// where on the device grid the piece starts, as the fraction of a pixel —
// which is all that changes the picture; the whole part only moves it.
type pixelGridKey struct {
	piece  *paintengine2d.Image
	kx, ky float32
	fx, fy int32 // 1/4096ths of a pixel
}

// pixelGridMax caps the magnifications kept at once. A panel has a few
// hundred pieces at a handful of phases; the cap is reached only by an app
// that moves pixel sprites by fractions every frame.
const pixelGridMax = 1024

// pixelGridImage is img magnified by (kx, ky) with its top left at device
// (x0, y0), from the cache when it has been made before.
func pixelGridImage(img *paintengine2d.Image, x0, y0, kx, ky float32) *pixelGrid {
	if kx <= 0 || ky <= 0 {
		return nil
	}
	ix, fx := splitPixel(x0)
	iy, fy := splitPixel(y0)
	key := pixelGridKey{piece: img, kx: kx, ky: ky, fx: fx, fy: fy}
	skinCache.mu.Lock()
	skinCache.syncGen()
	g, ok := skinCache.grids[key]
	skinCache.mu.Unlock()
	if !ok {
		g = magnifyOnGrid(img, float64(fx)/4096, float64(fy)/4096, float64(kx), float64(ky))
		skinCache.mu.Lock()
		if len(skinCache.grids) >= pixelGridMax {
			skinCache.grids = map[pixelGridKey]*pixelGrid{}
		}
		skinCache.grids[key] = g
		skinCache.mu.Unlock()
	}
	if g == nil {
		return nil
	}
	return &pixelGrid{img: g.img, px: g.px + ix, py: g.py + iy}
}

// splitPixel is v's whole pixel and its fraction in 1/4096ths, rounded so a
// value a float's breadth off a fraction lands on it.
func splitPixel(v float32) (int, int32) {
	q := int64(math.Round(float64(v) * 4096))
	whole := q >> 12 // floor, negatives included
	return int(whole), int32(q - whole<<12)
}

// magnifyOnGrid lays img's texels on the device grid: texel i along an axis
// starts at f + i·k, and a device pixel is the texel its centre falls in.
// px, py are relative to the whole pixel f is a fraction of.
func magnifyOnGrid(img *paintengine2d.Image, fx, fy, kx, ky float64) *pixelGrid {
	cols, c0 := gridSpans(img.Width, fx, kx)
	rows, r0 := gridSpans(img.Height, fy, ky)
	if len(cols) == 0 || len(rows) == 0 {
		return nil
	}
	out := paintengine2d.NewImage(len(cols), len(rows))
	stride := out.RowStride()
	for j, ty := range rows {
		o := j * stride
		for i, tx := range cols {
			r, g, b, a := img.PremulAt(tx, ty)
			out.Pix[o+i*4+0], out.Pix[o+i*4+1], out.Pix[o+i*4+2], out.Pix[o+i*4+3] = r, g, b, a
		}
	}
	out.Touch()
	return &pixelGrid{img: out, px: c0, py: r0}
}

// gridSpans maps n texels starting at device f with k pixels each onto
// device pixels: for each pixel from the first one returned, the texel its
// centre is in. A centre exactly on a texel edge belongs to the texel that
// starts there — decided in texel units with a margin far below a pixel, so
// two pieces that meet decide it the same way.
func gridSpans(n int, f, k float64) ([]int, int) {
	const eps = 1e-6
	texel := func(p int) int { return int(math.Floor((float64(p)+0.5-f)/k + eps)) }
	first := int(math.Floor(f - 1))
	for texel(first) < 0 {
		first++
	}
	var out []int
	for p := first; ; p++ {
		t := texel(p)
		if t >= n {
			break
		}
		out = append(out, t)
	}
	return out, first
}

// texelScale is how many device pixels one texel of the chosen asset covers.
// Pixelated art takes the whole part of it, so its pixels stay square.
func (sk *Skin) texelScale(sp *SkinSprite, v *skinVariant, scale float32) float32 {
	k := scale / v.scale
	if sp.Sheet != nil && sp.Sheet.Pixelated {
		if n := float32(math.Floor(float64(k))); n >= 1 {
			return n
		}
	}
	return k
}

// assetTarget is the sheet scale to pick for a sprite drawn at a display
// scale. Ordinary art wants the asset at or above the display scale. A
// pixelated sheet wants a whole multiple: a 2× sheet shown at 1.75 is
// nearest-sampled at 2 and fitted, because a crisp 2× sprite reads as pixel
// art while a 1.75 enlargement of a 1× one reads as a mistake.
func (sk *Skin) assetTarget(sp *SkinSprite, scale float32) float32 {
	if sp.Sheet != nil && sp.Sheet.Pixelated {
		n := float32(math.Floor(float64(scale)))
		if n < 1 {
			n = 1
		}
		return n
	}
	return scale
}

// fitBox is where an unsliced sprite is drawn inside b.
//
// An ordinary sprite fills b. A pixelated one is drawn at a whole multiple
// of its own texels, centred and snapped to the device grid, so its pixels
// stay square at 1.25, 1.5 and 1.75 — the art keeps its integer scale while
// layout, text and hit-testing stay at the true fractional one.
func (sk *Skin) fitBox(b paintengine2d.Rect, v *skinVariant, sp *SkinSprite, scale float32) paintengine2d.Rect {
	if sp.Sheet == nil || !sp.Sheet.Pixelated {
		return b
	}
	img := v.pieces[pieceWhole]
	if img == nil {
		return b
	}
	n := float32(math.Floor(float64(scale / v.scale)))
	if n < 1 {
		n = 1
	}
	src := pieceRect(img)
	w, h := src.Dx()*n, src.Dy()*n
	if w > b.Dx() || h > b.Dy() {
		return b // the box is smaller than one whole multiple: fill it
	}
	x := float32(math.Round(float64(b.Min.X + (b.Dx()-w)*0.5)))
	y := float32(math.Round(float64(b.Min.Y + (b.Dy()-h)*0.5)))
	return paintengine2d.XYWH(x, y, w, h)
}

// drawSlice paints a nine-slice sprite into b.
//
// The fixed corners are drawn at the asset's own scale, snapped to whole
// device pixels; the stretchable middles absorb the fraction. A box too
// small for its own corners shrinks them proportionally rather than letting
// them overlap, so a sprite drawn into a 6px-tall track still looks like
// itself.
func (sk *Skin) drawSlice(ctx *paintengine2d.Context, b paintengine2d.Rect, v *skinVariant, sp *SkinSprite, scale float32, paint paintengine2d.Paint) bool {
	// Piece pixels → device pixels: the art's own scale, taken to the
	// display's.
	k := sk.texelScale(sp, v, scale)
	l := skinEdge(v.slice.Left * k)
	r := skinEdge(v.slice.Right * k)
	t := skinEdge(v.slice.Top * k)
	bt := skinEdge(v.slice.Bottom * k)

	x0 := float32(math.Round(float64(b.Min.X)))
	y0 := float32(math.Round(float64(b.Min.Y)))
	x3 := float32(math.Round(float64(b.Max.X)))
	y3 := float32(math.Round(float64(b.Max.Y)))
	w, h := x3-x0, y3-y0
	if w <= 0 || h <= 0 {
		return false
	}
	// Corners that no longer fit: keep their ratio, lose their size. The
	// alternative — overlapping them — draws one corner over another and
	// reads as a smear.
	if l+r > w {
		if s := w / (l + r); s > 0 {
			l, r = skinEdgeDown(l*s), skinEdgeDown(r*s)
		}
	}
	if t+bt > h {
		if s := h / (t + bt); s > 0 {
			t, bt = skinEdgeDown(t*s), skinEdgeDown(bt*s)
		}
	}
	cols := [4]float32{x0, x0 + l, x3 - r, x3}
	rows := [4]float32{y0, y0 + t, y3 - bt, y3}

	drew := false
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			img := v.pieces[skinPiece(row*3+col)]
			if img == nil {
				continue
			}
			cell := paintengine2d.Rect{
				Min: paintengine2d.Pt(cols[col], rows[row]),
				Max: paintengine2d.Pt(cols[col+1], rows[row+1]),
			}
			if cell.Dx() <= 0 || cell.Dy() <= 0 {
				continue
			}
			stretchX := col == 1
			stretchY := row == 1
			// FillNone leaves the centre empty and keeps the frame: that is
			// what a focus ring, an outline and a group box are. It does not
			// drop the edges — an edge is the frame.
			if stretchX && stretchY && sp.Middle == FillNone {
				continue
			}
			if (stretchX || stretchY) && sp.Middle == FillTile {
				if tilePiece(ctx, img, cell, k, stretchX, stretchY, paint) {
					drew = true
				}
				continue
			}
			if blitPiece(ctx, img, cell, paint) {
				drew = true
			}
		}
	}
	return drew
}

// blitPiece draws one cut piece into dst.
//
// Each piece is its own image, so no blit can reach another sprite on the
// sheet, and the sampler clamps at an image's edge, so a stretched piece
// keeps its edge colour to its edge (skin_assets.go, cutPiece).
func blitPiece(ctx *paintengine2d.Context, img *paintengine2d.Image, dst paintengine2d.Rect, paint paintengine2d.Paint) bool {
	if ctx == nil || img == nil || dst.Empty() {
		return false
	}
	ctx.DrawImageRectPaint(img, pieceRect(img), dst, paint)
	return true
}

// tilePiece repeats a piece at its own size across cell, along whichever
// axes stretch. A texture, a hatch or a grille tiles; a gradient does not,
// which is why the policy is the skin's to state.
//
// paintengine2d's ImagePattern always repeats on both axes, so the piece is
// tiled under a clip and the non-tiling axis is handled by drawing one row
// (or column) of tiles scaled to the cell across that axis.
func tilePiece(ctx *paintengine2d.Context, img *paintengine2d.Image, cell paintengine2d.Rect, k float32, tileX, tileY bool, paint paintengine2d.Paint) bool {
	if ctx == nil || img == nil || cell.Empty() {
		return false
	}
	src := pieceRect(img)
	tw, th := src.Dx()*k, src.Dy()*k
	if tw < 0.5 || th < 0.5 {
		return false
	}
	if !tileX {
		tw = cell.Dx()
	}
	if !tileY {
		th = cell.Dy()
	}
	ctx.Save()
	ctx.ClipRect(cell)
	for y := cell.Min.Y; y < cell.Max.Y-0.01; y += th {
		for x := cell.Min.X; x < cell.Max.X-0.01; x += tw {
			ctx.DrawImageRectPaint(img, src, paintengine2d.XYWH(x, y, tw, th), paint)
		}
	}
	ctx.Restore()
	return true
}

// skinEdge rounds a nine-slice's fixed edge to a whole device pixel,
// keeping at least one where the art asked for any: a hairline that rounds
// to nothing is a missing border, not a thin one.
func skinEdge(v float32) float32 {
	if v <= 0 {
		return 0
	}
	r := float32(math.Round(float64(v)))
	if r < 1 {
		return 1
	}
	return r
}

// skinEdgeDown rounds a shrunken edge down, so two corners can never add up
// to more than the box they are in.
func skinEdgeDown(v float32) float32 {
	if v <= 0 {
		return 0
	}
	return float32(math.Floor(float64(v)))
}
