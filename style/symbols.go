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

func fallbackPaperclip(size float32) (*paintengine2d.Path, float32) {
	p := paintengine2d.NewPath()
	h := size * 0.76
	w := size * 0.30
	x := size * 0.12
	y := -size * 0.80
	p.AddRoundRect(paintengine2d.XYWH(x, y, w, h), w*0.48, w*0.48)
	p.AddRoundRect(paintengine2d.XYWH(x+w*0.28, y+h*0.16, w*0.52, h*0.64), w*0.26, w*0.26)
	return p, size * 0.62
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
