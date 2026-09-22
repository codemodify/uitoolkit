package widgets

import (
	"image"
	_ "image/gif"  // decoders for LoadPicture
	_ "image/jpeg" //
	_ "image/png"  //
	"io"
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
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
//
// A Zoomable picture is an image viewer's: a touchpad pinch, or Ctrl and
// the wheel, zoom it about the point under the fingers or the pointer (the
// Fit size is 1×, up to MaxZoom), and once it is larger than its box the
// wheel, two fingers or a pinch's movement pan it — Gwenview's and Loupe's
// gestures.
type Picture struct {
	widget.Base
	img       *paintengine2d.Image
	Fit       ImageFit
	Pixelated bool
	Zoomable  bool
	// MaxZoom caps the zoom; zero is 8.
	MaxZoom float32
	// OnZoom hears the zoom change.
	OnZoom func(zoom float32)

	zoom, zoom0 float32
	// pan is how far the zoomed image's centre is from the fitted one's.
	pan paintengine2d.Point
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

func (p *Picture) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	p.clampPan()
}

// Zoom is the picture's zoom: 1 at its Fit size.
func (p *Picture) Zoom() float32 {
	if p.zoom <= 0 {
		return 1
	}
	return p.zoom
}

// SetZoom zooms about the centre of the box; 1 puts it back to its Fit.
func (p *Picture) SetZoom(z float32) {
	b := p.LocalBounds()
	p.zoomAt(paintengine2d.Pt(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2), z)
}

func (p *Picture) maxZoom() float32 {
	if p.MaxZoom > 1 {
		return p.MaxZoom
	}
	return 8
}

// shown is where the image is drawn: its Fit rectangle, zoomed and panned.
func (p *Picture) shown(b paintengine2d.Rect) (src, dst paintengine2d.Rect) {
	src, dst = fitRects(p.img, b, p.natural(), p.Fit)
	z := p.Zoom()
	if z == 1 && p.pan == (paintengine2d.Point{}) {
		return src, dst
	}
	cx, cy := dst.Min.X+dst.Dx()/2+p.pan.X, dst.Min.Y+dst.Dy()/2+p.pan.Y
	w, h := dst.Dx()*z, dst.Dy()*z
	return src, paintengine2d.XYWH(cx-w/2, cy-h/2, w, h)
}

// zoomAt zooms to z keeping the image's point under at where it is.
func (p *Picture) zoomAt(at paintengine2d.Point, z float32) {
	if p.img == nil {
		return
	}
	z = min(max(z, 1), p.maxZoom())
	b := p.LocalBounds()
	_, old := p.shown(b)
	_, fit := fitRects(p.img, b, p.natural(), p.Fit)
	if old.Dx() <= 0 || old.Dy() <= 0 {
		return
	}
	ux, uy := (at.X-old.Min.X)/old.Dx(), (at.Y-old.Min.Y)/old.Dy()
	w, h := fit.Dx()*z, fit.Dy()*z
	ox, oy := at.X-ux*w, at.Y-uy*h
	p.pan = paintengine2d.Pt(ox+w/2-(fit.Min.X+fit.Dx()/2), oy+h/2-(fit.Min.Y+fit.Dy()/2))
	changed := z != p.Zoom()
	p.zoom = z
	p.clampPan()
	p.Invalidate()
	if changed && p.OnZoom != nil {
		p.OnZoom(z)
	}
}

// clampPan keeps a zoomed image over its box: no gap at an edge while it
// is larger than the box that way, centred while it is smaller.
func (p *Picture) clampPan() {
	if p.img == nil || p.Zoom() == 1 {
		p.pan = paintengine2d.Point{}
		return
	}
	b := p.LocalBounds()
	_, fit := fitRects(p.img, b, p.natural(), p.Fit)
	z := p.Zoom()
	clamp := func(pan, size, lo, hi, centre float32) float32 {
		if size <= hi-lo {
			return (lo+hi)/2 - centre
		}
		// The image's centre may go as far as keeps both edges out.
		minC, maxC := hi-size/2, lo+size/2
		c := min(max(centre+pan, minC), maxC)
		return c - centre
	}
	p.pan.X = clamp(p.pan.X, fit.Dx()*z, b.Min.X, b.Max.X, fit.Min.X+fit.Dx()/2)
	p.pan.Y = clamp(p.pan.Y, fit.Dy()*z, b.Min.Y, b.Max.Y, fit.Min.Y+fit.Dy()/2)
}

// Gesture zooms a Zoomable picture with a pinch, about the fingers, and
// pans it with their movement.
func (p *Picture) Gesture(e widget.GestureEvent) bool {
	if !p.Zoomable || p.img == nil || e.Kind != platform.GesturePinch {
		return false
	}
	switch e.Phase {
	case platform.GestureBegin:
		p.zoom0 = p.Zoom()
	case platform.GestureUpdate:
		p.pan = p.pan.Add(e.Delta)
		p.zoomAt(e.Pos, p.zoom0*e.Scale)
	}
	return true
}

// MouseWheel zooms with Ctrl held, and pans a zoomed picture.
func (p *Picture) MouseWheel(e widget.MouseEvent) bool {
	if !p.Zoomable || p.img == nil {
		return false
	}
	if e.Mods.Ctrl() {
		steps := e.Scroll.Y
		if e.Precise {
			// Two fingers: a notch is about ten pixels of them.
			steps /= 10
		}
		p.zoomAt(e.Pos, p.Zoom()*float32(math.Pow(1.25, float64(-steps))))
		return true
	}
	if p.Zoom() == 1 {
		return false
	}
	d := e.Scroll
	if !e.Precise {
		line := style.Dip(p.Look(), 16) * 3
		d = paintengine2d.Pt(d.X*line, d.Y*line)
	}
	if e.Mods.Shift() && d.X == 0 {
		d = paintengine2d.Pt(d.Y, 0)
	}
	before := p.pan
	p.pan = paintengine2d.Pt(p.pan.X-d.X, p.pan.Y-d.Y)
	p.clampPan()
	if p.pan != before {
		p.Invalidate()
		return true
	}
	// At the edge: the scroll view around it moves instead.
	return false
}

func (p *Picture) Paint(ctx *paintengine2d.Context) {
	if p.img == nil || p.img.Width == 0 || p.img.Height == 0 {
		return
	}
	b := p.LocalBounds()
	src, dst := p.shown(b)
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
