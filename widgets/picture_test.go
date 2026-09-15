package widgets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A picture asks for its natural size, shrinks keeping its aspect ratio,
// and each fit mode places it as documented.
func TestPictureFits(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	p, err := LoadPicture(&buf)
	if err != nil {
		t.Fatal(err)
	}
	p.SetHost(&fakeWindow{look: style.DarkLook()})
	if sz := p.Measure(layout.Unbounded()); sz.X != 200 || sz.Y != 100 {
		t.Fatalf("natural size %v, want 200x100", sz)
	}
	if sz := p.Measure(layout.Loose(100, 400)); sz.X != 100 || sz.Y != 50 {
		t.Fatalf("shrunk size %v, want 100x50 (aspect kept)", sz)
	}
	box := paintengine2d.XYWH(0, 0, 100, 100)
	_, dst := fitRects(p.Image(), box, p.natural(), FitContain)
	if dst != paintengine2d.XYWH(0, 25, 100, 50) {
		t.Fatalf("contain: %v, want letterboxed 0,25 100x50", dst)
	}
	srcR, dst := fitRects(p.Image(), box, p.natural(), FitCover)
	if dst != box || srcR != paintengine2d.XYWH(50, 0, 100, 100) {
		t.Fatalf("cover: src %v dst %v, want the centre square", srcR, dst)
	}
	// It paints something inside its box.
	p.Arrange(box)
	img := paintengine2d.NewImage(100, 100)
	p.Paint(paintengine2d.NewContext(img))
	if _, _, b, a := img.At(50, 50).RGBA(); a == 0 || b == 0 {
		t.Fatal("the picture did not paint")
	}
	if _, _, _, a := img.At(50, 5).RGBA(); a != 0 {
		t.Fatal("contain should letterbox: the top band must stay empty")
	}
}
