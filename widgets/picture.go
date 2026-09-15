package widgets

import (
	"image"
	_ "image/gif"  // decoders for LoadPicture
	_ "image/jpeg" //
	_ "image/png"  //
	"io"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ImageFit is how a picture fills its box (CSS object-fit, Qt's aspect
// ratio modes).
type ImageFit uint8

const (
	// FitContain shows the whole picture as large as fits, letterboxed.
	FitContain ImageFit = iota
	// FitCover fills the box, cropping what overflows.
	FitCover
	// FitStretch fills the box, ignoring the aspect ratio.
	FitStretch
	// FitNone keeps the natural size (times the display scale), centred.
	FitNone
)

// Picture shows a raster image — QLabel with a pixmap, GtkPicture, WPF's
// Image. It asks for the image's natural size (scaled for HiDPI), shrinks
// with its box and draws smoothly unless Pixelated (pixel art, icons at
// whole multiples).
type Picture struct {
	widget.Base
	img       *paintengine2d.Image
	Fit       ImageFit
	Pixelated bool
}

// NewPicture shows img (nil for none yet).
func NewPicture(img *paintengine2d.Image) *Picture {
	p := &Picture{img: img}
	p.Init(p)
	return p
}

// LoadPicture decodes a PNG, JPEG or GIF and shows it.
func LoadPicture(r io.Reader) (*Picture, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return NewPicture(paintengine2d.NewImageFromNRGBA(src)), nil
}

// Image is the picture shown.
func (p *Picture) Image() *paintengine2d.Image { return p.img }

// SetImage replaces the picture.
func (p *Picture) SetImage(img *paintengine2d.Image) {
	p.img = img
	p.RequestLayout()
	p.Invalidate()
}

// natural is the image size at the display scale.
func (p *Picture) natural() paintengine2d.Point {
	if p.img == nil {
		return paintengine2d.Point{}
	}
	s := style.LookScale(p.Look())
	if s <= 0 {
		s = 1
	}
	return paintengine2d.Pt(float32(p.img.Width)*s, float32(p.img.Height)*s)
}

func (p *Picture) Measure(c layout.Constraints) paintengine2d.Point {
	n := p.natural()
	if n.X <= 0 || n.Y <= 0 {
		return c.Constrain(paintengine2d.Point{})
	}
	w, h := n.X, n.Y
	// Shrink to the box keeping the aspect ratio (a picture never asks to
	// be cut in half).
	if c.HasMaxW() && w > c.MaxW {
		h *= c.MaxW / w
		w = c.MaxW
	}
	if c.HasMaxH() && h > c.MaxH {
		w *= c.MaxH / h
		h = c.MaxH
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (p *Picture) Arrange(r paintengine2d.Rect) { p.SetBounds(r) }

func (p *Picture) Paint(ctx *paintengine2d.Context) {
	if p.img == nil || p.img.Width == 0 || p.img.Height == 0 {
		return
	}
	b := p.LocalBounds()
	src, dst := fitRects(p.img, b, p.natural(), p.Fit)
	if dst.Empty() || src.Empty() {
		return
	}
	paint := paintengine2d.Paint{}
	if p.Pixelated {
		paint.Filter = paintengine2d.FilterNearest
	}
	ctx.Save()
	ctx.ClipRect(b)
	ctx.DrawImageRectPaint(p.img, src, dst, paint)
	ctx.Restore()
}

// fitRects is the source and destination rects that show img in b.
func fitRects(img *paintengine2d.Image, b paintengine2d.Rect, natural paintengine2d.Point, fit ImageFit) (src, dst paintengine2d.Rect) {
	iw, ih := float32(img.Width), float32(img.Height)
	src = paintengine2d.XYWH(0, 0, iw, ih)
	switch fit {
	case FitStretch:
		return src, b
	case FitNone:
		w, h := natural.X, natural.Y
		return src, paintengine2d.XYWH(b.Min.X+(b.Dx()-w)*0.5, b.Min.Y+(b.Dy()-h)*0.5, w, h)
	case FitCover:
		// Crop the source to the box's aspect ratio, centred.
		if b.Dx()*ih > b.Dy()*iw {
			ch := iw * b.Dy() / b.Dx()
			src = paintengine2d.XYWH(0, (ih-ch)*0.5, iw, ch)
		} else {
			cw := ih * b.Dx() / b.Dy()
			src = paintengine2d.XYWH((iw-cw)*0.5, 0, cw, ih)
		}
		return src, b
	}
	// FitContain: as large as fits, centred.
	k := b.Dx() / iw
	if kh := b.Dy() / ih; kh < k {
		k = kh
	}
	w, h := iw*k, ih*k
	return src, paintengine2d.XYWH(b.Min.X+(b.Dx()-w)*0.5, b.Min.Y+(b.Dy()-h)*0.5, w, h)
}
