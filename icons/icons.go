// Package icons makes window icons for the toolkit's own programs from its
// icon sets (README.md). The sets themselves are not embedded — an app
// copies the one it wants into the user's config directory — but a
// handful of Lucide glyphs are, so every example wears an icon in the
// desktop's title bar, task switcher and task bar
// (app.Application.SetIcon) with nothing to install.
package icons

import (
	"bytes"
	"embed"
	"math"

	"github.com/codemodify/paintengine2d"
)

// The glyphs the toolkit's programs wear, from the Lucide set (ISC, see
// lucide/LICENSE), at 48 pixels.
//
//go:embed lucide/LICENSE lucide/mail@2x.png lucide/folder@2x.png lucide/layout@2x.png lucide/search@2x.png lucide/sun@2x.png lucide/star@2x.png lucide/list@2x.png lucide/pencil@2x.png lucide/more@2x.png lucide/cards@2x.png lucide/columns@2x.png lucide/home@2x.png lucide/settings@2x.png
var glyphs embed.FS

// AppIconSizes are the sizes AppIcon draws: the title bar's 16 to 24
// pixels at 1x and 2x, the task bar's and the task switcher's 32 to 64,
// and 128 for a switcher that shows it big.
var AppIconSizes = []int{16, 22, 24, 32, 48, 64, 128}

// AppIcon is a window icon made from the Lucide glyph stem: the glyph in
// white on a rounded tile of colour bg, lit from the top, at every size in
// AppIconSizes. It is nil for a stem that is not embedded.
func AppIcon(stem string, bg paintengine2d.Color) []*paintengine2d.Image {
	mask := glyphMask(stem)
	if mask == nil {
		return nil
	}
	out := make([]*paintengine2d.Image, 0, len(AppIconSizes))
	for _, side := range AppIconSizes {
		out = append(out, appIconAt(mask, bg, side))
	}
	return out
}

// AppIconRGB is AppIcon on a tile of colour r, g, b (0 to 255).
func AppIconRGB(stem string, r, g, b uint8) []*paintengine2d.Image {
	return AppIcon(stem, paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255))
}

// Stems are the glyphs AppIcon can draw.
func Stems() []string {
	return []string{"mail", "folder", "layout", "search", "sun", "star", "list", "pencil", "more", "cards", "columns", "home", "settings"}
}

// glyphMask is the stem's 48-pixel glyph as white coverage (its alpha),
// premultiplied, ready to be tinted.
func glyphMask(stem string) *paintengine2d.Image {
	raw, err := glyphs.ReadFile("lucide/" + stem + "@2x.png")
	if err != nil {
		return nil
	}
	src, err := paintengine2d.DecodePNG(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	out := paintengine2d.NewImage(src.Width, src.Height)
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			_, _, _, a := src.PremulAt(x, y)
			i := y*out.RowStride() + x*4
			out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = a, a, a, a
		}
	}
	out.Touch()
	return out
}

// appIconAt draws the tile and the glyph side pixels square. The tile
// keeps a pixel clear at every edge (a desktop puts icons side by side),
// its corners a fifth of its side; the glyph takes three fifths of it,
// on whole pixels, and at the smallest sizes a little more so its strokes
// still read.
func appIconAt(mask *paintengine2d.Image, bg paintengine2d.Color, side int) *paintengine2d.Image {
	img := paintengine2d.NewImage(side, side)
	ctx := paintengine2d.NewContext(img)
	s := float32(side)
	inset := float32(math.Max(1, math.Round(float64(s)/32)))
	tile := paintengine2d.XYWH(inset, inset, s-2*inset, s-2*inset)
	p := paintengine2d.NewPath()
	p.AddRoundRect(tile, tile.Dx()*0.2, tile.Dx()*0.2)
	top, bottom := bg.Lerp(paintengine2d.RGB(1, 1, 1), 0.18), bg.Lerp(paintengine2d.RGB(0, 0, 0), 0.12)
	ctx.DrawPath(p, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: tile.Min,
		End:   paintengine2d.Pt(tile.Min.X, tile.Max.Y),
		Stops: []paintengine2d.GradientStop{{Offset: 0, Color: top}, {Offset: 1, Color: bottom}},
	}))
	frac := 0.6
	if side <= 24 {
		frac = 0.72
	}
	g := int(math.Round(float64(s) * frac))
	o := float32(math.Round(float64(side-g) * 0.5))
	glyph := resample(mask, g)
	ctx.DrawImageRectPaint(glyph, paintengine2d.XYWH(0, 0, float32(g), float32(g)), paintengine2d.XYWH(o, o, float32(g), float32(g)), paintengine2d.Paint{
		Color: paintengine2d.RGB(1, 1, 1),
	})
	return img
}

// resample is the square coverage mask m at side n, each pixel the area
// average of the source pixels under it. A filter that samples points
// instead drops a 48-pixel glyph's strokes altogether at 16.
func resample(m *paintengine2d.Image, n int) *paintengine2d.Image {
	out := paintengine2d.NewImage(n, n)
	if n <= 0 || m.Width <= 0 {
		return out
	}
	sc := float64(m.Width) / float64(n)
	// A stroke thinner than a pixel is spread over two at half strength;
	// strengthened, it reads as a line again rather than a grey smear —
	// what hinting does for a small font.
	boost := math.Min(2, math.Max(1, sc/2.2))
	// weights[i] is how much of source index j falls in destination i.
	span := func(i int) (j0, j1 int, w func(j int) float64) {
		a, b := float64(i)*sc, float64(i+1)*sc
		j0, j1 = int(math.Floor(a)), int(math.Ceil(b))
		return j0, min(j1, m.Width), func(j int) float64 {
			return math.Min(b, float64(j+1)) - math.Max(a, float64(j))
		}
	}
	for y := 0; y < n; y++ {
		y0, y1, wy := span(y)
		for x := 0; x < n; x++ {
			x0, x1, wx := span(x)
			var sum float64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					_, _, _, a := m.PremulAt(sx, sy)
					sum += float64(a) * wx(sx) * wy(sy)
				}
			}
			v := uint8(math.Min(255, math.Round(sum/(sc*sc)*boost)))
			i := y*out.RowStride() + x*4
			out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = v, v, v, v
		}
	}
	out.Touch()
	return out
}
