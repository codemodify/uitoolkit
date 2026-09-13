package platform

import (
	"image"
	"image/png"
	"os"

	"github.com/codemodify/paintengine2d"
)

func resolveStatusImage(icon StatusIcon, size int) *paintengine2d.Image {
	if icon.Image != nil && icon.Image.Width > 0 && icon.Image.Height > 0 {
		return icon.Image
	}
	if icon.Path != "" {
		if img, err := loadStatusPNG(icon.Path); err == nil && img != nil {
			return img
		}
	}
	if size < 16 {
		size = 22
	}
	return paintengine2d.NewImage(size, size)
}

func loadStatusPNG(path string) (*paintengine2d.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return imageToPE(src), nil
}

func imageToPE(src image.Image) *paintengine2d.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := paintengine2d.NewImage(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// RGBA() is 16-bit; premul 8-bit for the engine.
			if a == 0 {
				continue
			}
			dst.SetColor(x, y, paintengine2d.RGBA(
				float32(r)/float32(a),
				float32(g)/float32(a),
				float32(bl)/float32(a),
				float32(a)/65535,
			))
		}
	}
	return dst
}

// sniARGB is StatusNotifierItem IconPixmap: network-endian ARGB32 with
// straight (non-premultiplied) colour channels.
//
// The spec, and every host that implements it (Plasma, AppIndicator,
// Waybar), treat the pixmap as plain ARGB32. Sending premultiplied
// channels darkened every semi-transparent pixel, which shows up as a
// dirty fringe around an anti-aliased icon.
func sniARGB(img *paintengine2d.Image) (w, h int, pix []byte) {
	if img == nil || img.Width < 1 || img.Height < 1 {
		return 0, 0, nil
	}
	w, h = img.Width, img.Height
	pix = make([]byte, w*h*4)
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.PremulAt(x, y)
			r, g, b = unpremul(r, a), unpremul(g, a), unpremul(b, a)
			pix[i+0] = a
			pix[i+1] = r
			pix[i+2] = g
			pix[i+3] = b
			i += 4
		}
	}
	return w, h, pix
}

// unpremul converts one premultiplied channel back to straight alpha.
func unpremul(c, a uint8) uint8 {
	if a == 0 {
		return 0
	}
	if a == 0xff || c >= a {
		if c > a {
			return 0xff
		}
		if a == 0xff {
			return c
		}
	}
	v := (int(c)*255 + int(a)/2) / int(a)
	if v > 255 {
		v = 255
	}
	return uint8(v)
}
