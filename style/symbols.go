package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// symbolFallback returns a filled path for UI marks Titillium / JetBrains
// omit (★ 📎 ● 🔇). paintengine2d can paint any atlas cell; the hole is
// here — rasterize used to drop missing non-ASCII gids with no ink.
func symbolFallback(r rune, size float32) (*paintengine2d.Path, float32) {
	if size < 8 {
		size = 8
	}
	switch r {
	case '★':
		return fallbackStar(size)
	case '📎':
		return fallbackPaperclip(size)
	case '●':
		return fallbackBullet(size)
	case '🔇':
		return fallbackMuted(size)
	default:
		return nil, 0
	}
}

func fallbackStar(size float32) (*paintengine2d.Path, float32) {
	outer := size * 0.36
	inner := outer * 0.40
	cx := size * 0.40
	cy := -size * 0.40
	p := paintengine2d.NewPath()
	for i := 0; i < 10; i++ {
		ang := -math.Pi/2 + float64(i)*math.Pi/5
		rad := outer
		if i%2 == 1 {
			rad = inner
		}
		x := cx + rad*float32(math.Cos(ang))
		y := cy + rad*float32(math.Sin(ang))
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return p, size * 0.78
}

// fallbackPaperclip is 📎 for fonts without it: a wire loop (a capsule
// ring) with the inner wire standing in its hole. Glyphs bake with the
// non-zero rule, so the hole is wound the other way; with both contours
// clockwise the clip used to fill solid and read as a tofu box.
func fallbackPaperclip(size float32) (*paintengine2d.Path, float32) {
	p := paintengine2d.NewPath()
	h := size * 0.80
	w := size * 0.36
	x := size * 0.10
	y := -size * 0.84
	t := size * 0.075 // wire thickness
	if t < 1.2 {
		t = 1.2
	}
	p.AddRoundRect(paintengine2d.XYWH(x, y, w, h), w*0.5, w*0.5)
	hole := paintengine2d.XYWH(x+t, y+t, w-2*t, h-2*t)
	addRoundRectCCW(p, hole, hole.Dx()*0.5)
	// The inner wire: a bar from the upper third down towards the bottom.
	bx := x + w*0.5 - t*0.5
	p.AddRoundRect(paintengine2d.XYWH(bx, y+h*0.30, t, h*0.52), t*0.5, t*0.5)
	return p, size * 0.62
}

// addRoundRectCCW adds a counter-clockwise rounded rect (a hole under the
// non-zero fill rule when its outline is clockwise).
func addRoundRectCCW(p *paintengine2d.Path, b paintengine2d.Rect, r float32) {
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	if max := b.Dx() * 0.5; r > max {
		r = max
	}
	if max := b.Dy() * 0.5; r > max {
		r = max
	}
	const k = 0.5522847
	p.MoveTo(b.Min.X+r, b.Min.Y)
	p.CubicTo(b.Min.X+r-r*k, b.Min.Y, b.Min.X, b.Min.Y+r-r*k, b.Min.X, b.Min.Y+r)
	p.LineTo(b.Min.X, b.Max.Y-r)
	p.CubicTo(b.Min.X, b.Max.Y-r+r*k, b.Min.X+r-r*k, b.Max.Y, b.Min.X+r, b.Max.Y)
	p.LineTo(b.Max.X-r, b.Max.Y)
	p.CubicTo(b.Max.X-r+r*k, b.Max.Y, b.Max.X, b.Max.Y-r+r*k, b.Max.X, b.Max.Y-r)
	p.LineTo(b.Max.X, b.Min.Y+r)
	p.CubicTo(b.Max.X, b.Min.Y+r-r*k, b.Max.X-r+r*k, b.Min.Y, b.Max.X-r, b.Min.Y)
	p.Close()
}

func fallbackBullet(size float32) (*paintengine2d.Path, float32) {
	p := paintengine2d.NewPath()
	p.AddCircle(paintengine2d.Pt(size*0.26, -size*0.30), size*0.15)
	return p, size * 0.48
}

func fallbackMuted(size float32) (*paintengine2d.Path, float32) {
	p := paintengine2d.NewPath()
	s := size
	p.MoveTo(s*0.08, -s*0.48)
	p.LineTo(s*0.28, -s*0.48)
	p.LineTo(s*0.48, -s*0.68)
	p.LineTo(s*0.48, -s*0.12)
	p.LineTo(s*0.28, -s*0.32)
	p.LineTo(s*0.08, -s*0.32)
	p.Close()
	p.MoveTo(s*0.18, -s*0.72)
	p.LineTo(s*0.26, -s*0.78)
	p.LineTo(s*0.72, -s*0.08)
	p.LineTo(s*0.64, -s*0.02)
	p.Close()
	return p, s * 0.78
}
