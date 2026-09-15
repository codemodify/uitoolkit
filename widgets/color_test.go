package widgets

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func TestHSVRoundTrip(t *testing.T) {
	for _, c := range []paintengine2d.Color{
		paintengine2d.RGB(0.2, 0.4, 0.8), paintengine2d.RGB(1, 0, 0), paintengine2d.RGB(0.5, 0.5, 0.5), paintengine2d.RGB(0.9, 0.7, 0.1),
	} {
		h, s, v := toHSV(c)
		back := hsv(h, s, v)
		if math.Abs(float64(back.R-c.R)) > 1e-4 || math.Abs(float64(back.G-c.G)) > 1e-4 || math.Abs(float64(back.B-c.B)) > 1e-4 {
			t.Fatalf("%v -> hsv(%v,%v,%v) -> %v", c, h, s, v, back)
		}
	}
	if got := ColorHex(paintengine2d.RGB(1, 0.5, 0)); got != "#ff8000" {
		t.Fatalf("hex %q", got)
	}
}

// The button drops the picker down; a swatch picks and closes it, and a
// drag in the square picks saturation and value live.
func TestColorButtonPicks(t *testing.T) {
	var got paintengine2d.Color
	b := NewColorButton(paintengine2d.RGB(0.2, 0.4, 0.8), func(c paintengine2d.Color) { got = c })
	host := &fakeWindow{look: style.DarkLook()}
	b.SetHost(host)
	b.Arrange(paintengine2d.XYWH(0, 0, 150, 28))
	b.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	pop, ok := host.popup.(*colorPopup)
	if !ok || !b.open {
		t.Fatal("space should open the picker")
	}
	_, cell, grid, square, _ := pop.geometry()
	// The first swatch.
	pop.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(grid.Min.X+cell/2, grid.Min.Y+cell/2), Button: platform.ButtonLeft})
	if got != colorPalette[0] || b.open {
		t.Fatalf("swatch pick: got %v (open %v), want %v and closed", got, b.open, colorPalette[0])
	}
	b.Open()
	pop = host.popup.(*colorPopup)
	// Bottom-right of the square: full saturation, value zero -> black.
	pop.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(square.Max.X-0.01, square.Max.Y-0.01), Button: platform.ButtonLeft})
	pop.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(square.Max.X-0.01, square.Max.Y-0.01), Button: platform.ButtonLeft})
	if got.R > 0.01 || got.G > 0.01 || got.B > 0.01 {
		t.Fatalf("bottom of the square should be black, got %v", got)
	}
	img := paintengine2d.NewImage(400, 400)
	pop.Arrange(paintengine2d.XYWH(0, 0, 300, 300))
	pop.Paint(paintengine2d.NewContext(img))
}
