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
// The source is the piece's inner rect — its replicated border is there for
// the sampler to read and never for the eye to see (skin_assets.go,
// cutPadded). Each piece is its own image, so no blit can reach another
// sprite on the sheet.
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
