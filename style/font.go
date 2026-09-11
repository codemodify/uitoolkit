package style

import (
	"sync"

	"github.com/codemodify/paintengine2d"
)

// Font is a baked glyph atlas painted through paintengine2d. Glyph ink is
// produced by the engine (bitmap atlas + nearest blit), not a second rasterizer.
type Font struct {
	Atlas   *paintengine2d.FontAtlas
	Size    float32
	Ascent  float32
	Descent float32
	Color   paintengine2d.Color
}

func (f *Font) Measure(text string) paintengine2d.Point {
	if f == nil {
		return paintengine2d.Point{}
	}
	return paintengine2d.Pt(f.Advance(text), f.Height())
}

func (f *Font) Height() float32 {
	if f == nil {
		return 0
	}
	if f.Descent > 0 {
		return f.Ascent + f.Descent
	}
	return f.Size + 2
}

func (f *Font) Advance(text string) float32 {
	if f == nil || f.Atlas == nil {
		return 0
	}
	run := paintengine2d.NullShaper{}.Shape(text, f.Atlas)
	var w float32
	for _, g := range run.Glyphs {
		if cell, ok := f.Atlas.Cell(g.ID); ok {
			adv := cell.Advance
			if adv <= 0 {
				adv = cell.Src.Dx()
			}
			if g.X+adv > w {
				w = g.X + adv
			}
		}
	}
	return w
}

func (f *Font) IndexAt(text string, x float32) int {
	if f == nil || x <= 0 {
		return 0
	}
	var acc float32
	i := 0
	for _, r := range text {
		adv := f.runeAdvance(r)
		if x < acc+adv*0.5 {
			return i
		}
		acc += adv
		i++
	}
	return i
}

func (f *Font) runeAdvance(r rune) float32 {
	if f == nil || f.Atlas == nil {
		return 0
	}
	if cell, ok := f.Atlas.Cell(paintengine2d.GlyphID(r)); ok {
		if cell.Advance > 0 {
			return cell.Advance
		}
		return cell.Src.Dx()
	}
	return 0
}

func (f *Font) CaretX(text string, i int) float32 {
	if f == nil || i <= 0 {
		return 0
	}
	n := 0
	var acc float32
	for _, r := range text {
		if n >= i {
			break
		}
		acc += f.runeAdvance(r)
		n++
	}
	return acc
}

func (f *Font) Draw(ctx *paintengine2d.Context, text string, origin paintengine2d.Point, tint paintengine2d.Color) {
	if f == nil || f.Atlas == nil || text == "" {
		return
	}
	col := tint
	if col == (paintengine2d.Color{}) {
		col = f.Color
	}
	if col == (paintengine2d.Color{}) {
		col = paintengine2d.White
	}
	run := paintengine2d.NullShaper{}.Shape(text, f.Atlas)
	ctx.DrawGlyphs(run, origin, paintengine2d.Paint{Color: col, Filter: paintengine2d.FilterNearest})
}

type fontKey struct {
	size int
}

var fontCache sync.Map

// BakeFont scales paintengine2d's 5×7 UI atlas (plus extra punct) to a shared
// white sheet. Theme color is applied at draw time via Paint.Color (v0.7.2 tint).
func BakeFont(size float32, col paintengine2d.Color) *Font {
	if size < 8 {
		size = 8
	}
	scale := int(size/8 + 0.5)
	if scale < 2 {
		scale = 2
	}
	if scale > 4 {
		scale = 4
	}
	key := fontKey{size: scale}
	if v, ok := fontCache.Load(key); ok {
		f := *v.(*Font)
		f.Color = col
		return &f
	}
	f := bakeScaled(scale, paintengine2d.White)
	fontCache.Store(key, f)
	out := *f
	out.Color = col
	return &out
}

func bakeScaled(scale int, col paintengine2d.Color) *Font {
	base := paintengine2d.NewBitmapAtlas(col)
	extra := extraPunctAtlas(col)
	// Merge extra cells into a combined source by listing ids.
	ids := make([]paintengine2d.GlyphID, 0, 80)
	seen := map[paintengine2d.GlyphID]bool{}
	for id := range base.Cells {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for id := range extra.Cells {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	srcW, srcH := 6, 8
	dstW, dstH := srcW*scale, srcH*scale
	const cols = 16
	rows := (len(ids) + cols - 1) / cols
	img := paintengine2d.NewImage(cols*dstW, rows*dstH)
	ctx := paintengine2d.NewContext(img)
	cells := make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, len(ids))
	for i, id := range ids {
		cell, ok := base.Cell(id)
		srcImg := base.Image
		if !ok {
			cell, ok = extra.Cell(id)
			srcImg = extra.Image
			if !ok {
				continue
			}
		}
		colI := i % cols
		rowI := i / cols
		dx := float32(colI * dstW)
		dy := float32(rowI * dstH)
		dst := paintengine2d.XYWH(dx, dy, float32(dstW), float32(dstH))
		ctx.DrawImageRectPaint(srcImg, cell.Src, dst, paintengine2d.Paint{
			Color:  paintengine2d.White,
			Filter: paintengine2d.FilterNearest,
		})
		adv := float32(dstW) + 1
		cells[id] = paintengine2d.AtlasCell{
			Src:     dst,
			Advance: adv,
		}
	}
	return &Font{
		Atlas:   &paintengine2d.FontAtlas{Image: img, Cells: cells},
		Size:    float32(dstH),
		Ascent:  float32(7 * scale),
		Descent: float32(scale + 2),
		Color:   col,
	}
}

// extraPunctAtlas is a tiny 5×7 sheet for punctuation the engine atlas omits.
func extraPunctAtlas(fg paintengine2d.Color) *paintengine2d.FontAtlas {
	type bits [7]byte
	extra := map[rune]bits{
		',':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x18},
		';':  {0x00, 0x0C, 0x0C, 0x00, 0x0C, 0x0C, 0x18},
		'\'': {0x0C, 0x0C, 0x08, 0x00, 0x00, 0x00, 0x00},
		'"':  {0x1B, 0x1B, 0x12, 0x00, 0x00, 0x00, 0x00},
		'@':  {0x0E, 0x11, 0x17, 0x15, 0x17, 0x10, 0x0E},
		'#':  {0x0A, 0x1F, 0x0A, 0x0A, 0x1F, 0x0A, 0x00},
		'[':  {0x1C, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1C},
		']':  {0x07, 0x01, 0x01, 0x01, 0x01, 0x01, 0x07},
		'{':  {0x06, 0x08, 0x08, 0x10, 0x08, 0x08, 0x06},
		'}':  {0x0C, 0x02, 0x02, 0x01, 0x02, 0x02, 0x0C},
		'<':  {0x02, 0x04, 0x08, 0x10, 0x08, 0x04, 0x02},
		'>':  {0x08, 0x04, 0x02, 0x01, 0x02, 0x04, 0x08},
		'|':  {0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
		'\\': {0x10, 0x10, 0x08, 0x04, 0x04, 0x02, 0x01},
		'*':  {0x00, 0x15, 0x0E, 0x04, 0x0E, 0x15, 0x00},
		'&':  {0x0C, 0x12, 0x14, 0x08, 0x15, 0x12, 0x0D},
		'`':  {0x08, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
		'~':  {0x00, 0x00, 0x09, 0x16, 0x00, 0x00, 0x00},
	}
	const cw, ch, cols = 6, 8, 8
	n := len(extra)
	rows := (n + cols - 1) / cols
	img := paintengine2d.NewImage(cols*cw, rows*ch)
	cells := make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, n)
	i := 0
	for r, b := range extra {
		col := i % cols
		row := i / cols
		ox, oy := col*cw, row*ch
		for y := 0; y < 7; y++ {
			rowBits := b[y]
			for x := 0; x < 5; x++ {
				if rowBits&(1<<uint(4-x)) != 0 {
					img.SetColor(ox+x, oy+y, fg)
				}
			}
		}
		cells[paintengine2d.GlyphID(r)] = paintengine2d.AtlasCell{
			Src:     paintengine2d.XYWH(float32(ox), float32(oy), cw, ch),
			Advance: cw,
		}
		i++
	}
	return &paintengine2d.FontAtlas{Image: img, Cells: cells}
}

var (
	tintOnce sync.Once
	tintOK   bool
)

// GlyphTint reports whether paintengine2d blits apply RGB Color as a tint
// (v0.7.2+). uitoolkit requires this for themed white atlases.
func GlyphTint() bool {
	tintOnce.Do(func() {
		atlas := paintengine2d.NewBitmapAtlas(paintengine2d.White)
		img := paintengine2d.NewImage(20, 16)
		ctx := paintengine2d.NewContext(img)
		run := paintengine2d.NullShaper{}.Shape("I", atlas)
		ctx.DrawGlyphs(run, paintengine2d.Pt(1, 1), paintengine2d.Paint{
			Color:  paintengine2d.RGB(1, 0, 0),
			Filter: paintengine2d.FilterNearest,
		})
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				r, g, b, a := img.PremulAt(x, y)
				if a > 20 && int(r) > int(g)+24 && int(r) > int(b)+24 {
					tintOK = true
					return
				}
			}
		}
	})
	return tintOK
}
