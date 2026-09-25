package widgets_test

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// longTip is the kind of sentence the Settings check boxes carry: an
// explanation, not a label. Before tooltips wrapped, a tip this long
// measured as one 900-pixel line, was cut to the window and elided at
// "KDE's and GNOME's own Open and Sa…".
const longTip = "KDE's and GNOME's own Open and Save dialogs, through the XDG portal, instead of the themed ones."

// tipLooks are the eras a wrapped tip has to look right in: the classic
// bevel, a KDE and a GNOME of today, the Mac's gel, and a skin, whose
// bubble is a picture with the text centred in it.
var tipLooks = []string{"win95", "breeze", "adwaita", "aqua", "cassette"}

// hoverTip opens a headless window in pack at scale with one button
// carrying tip, rests the pointer on it and shows the bubble.
func hoverTip(t *testing.T, pack string, scale float32, w, h int, tip string) (*uitoolkit.Application, *uitoolkit.Window, *widgets.TooltipBubble) {
	t.Helper()
	var btn *widgets.Button
	a, win := shotWindow(t, pack, scale, w, h, func() widget.Component {
		btn = widgets.NewButton("Hover me", nil)
		btn.Tip = tip
		return widgets.NewPad(16, widgets.NewColumn(btn))
	})
	t.Cleanup(func() { win.Close() })
	win.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: widget.DeviceBounds(btn).Center()})
	a.PumpOnce()
	win.RevealTooltip()
	a.PumpOnce()
	bubble, _ := win.Tooltip().(*widgets.TooltipBubble)
	if bubble == nil {
		t.Fatalf("%s: no tooltip bubble", pack)
	}
	return a, win, bubble
}

// A long tip wraps, stays inside the window, and every line of it fits the
// bubble it is drawn in — in every era, at every scale. One line of it used
// to be all anyone ever saw.
func TestTooltipWrapsInEveryLook(t *testing.T) {
	for _, pack := range tipLooks {
		for _, scale := range []float32{1, 1.75} {
			name := pack
			if scale != 1 {
				name += "-1.75x"
			}
			t.Run(name, func(t *testing.T) {
				_, win, bubble := hoverTip(t, pack, scale, 520, 300, longTip)
				lines := bubble.Lines()
				if len(lines) < 2 {
					t.Fatalf("a 95-character tip did not wrap: %q", lines)
				}
				if got := strings.Join(lines, " "); got != longTip {
					t.Fatalf("the wrap lost text:\n got %q\nwant %q", got, longTip)
				}
				ts := style.TooltipStyleOf(win.Look())
				b := bubble.Bounds()
				for _, ln := range lines {
					if w := ts.Face.Advance(ln); w > b.Dx()-ts.Pad*2 {
						t.Errorf("line %q is %.1f wide in a %.1f bubble", ln, w, b.Dx()-ts.Pad*2)
					}
				}
				// As tall as its lines, and no taller.
				if want := ts.Face.Height() * float32(len(lines)); b.Dy() < want {
					t.Errorf("bubble %.1f tall for %d lines of %.1f", b.Dy(), len(lines), ts.Face.Height())
				}
				pw, ph := win.PixelSize()
				if b.Min.X < 0 || b.Min.Y < 0 || b.Max.X > float32(pw) || b.Max.Y > float32(ph) {
					t.Errorf("bubble %v is outside the %dx%d window", b, pw, ph)
				}
				writeShot(t, win, "tooltip-"+name)
			})
		}
	}
}

// The measure is the tip's own face, not a pixel count: the same sentence
// wraps to the same words at 1x and at 1.75x, and the bubble is the same
// shape, scaled.
func TestTooltipMeasureIsTheFontNotPixels(t *testing.T) {
	_, _, one := hoverTip(t, "breeze", 1, 900, 400, longTip)
	_, _, big := hoverTip(t, "breeze", 1.75, 1575, 700, longTip)
	if a, b := one.Lines(), big.Lines(); strings.Join(a, "|") != strings.Join(b, "|") {
		t.Fatalf("the same tip broke differently at 1.75x:\n1x   %q\n1.75 %q", a, b)
	}
	if r := big.Bounds().Dx() / one.Bounds().Dx(); r < 1.7 || r > 1.8 {
		t.Fatalf("the 1.75x bubble is %.2f times the 1x one", r)
	}
}

// A window narrower than the look's measure is the tighter limit: the tip
// wraps to the window rather than running out of it.
func TestTooltipWrapsToASmallWindow(t *testing.T) {
	_, win, bubble := hoverTip(t, "adwaita", 1, 260, 220, longTip)
	pw, ph := win.PixelSize()
	b := bubble.Bounds()
	if b.Max.X > float32(pw) || b.Min.X < 0 {
		t.Fatalf("bubble %v is outside the %d px window", b, pw)
	}
	if len(bubble.Lines()) < 4 {
		t.Fatalf("a 260px window should take more than %d lines: %q", len(bubble.Lines()), bubble.Lines())
	}
	if b.Max.Y > float32(ph) {
		t.Fatalf("bubble %v is below the %d px window", b, ph)
	}
	writeShot(t, win, "tooltip-narrow")
}

// The bubble hands the engine its lines, newline-separated, and the engine
// paints all of them: the window under a two-line tip has ink below the
// first line's baseline.
func TestTooltipPaintsEveryLine(t *testing.T) {
	_, win, bubble := hoverTip(t, "win95", 1, 520, 300, longTip)
	if len(bubble.Lines()) < 2 {
		t.Fatal("expected a wrapped tip")
	}
	off, ok := win.Surface().(*platform.Offscreen)
	if !ok {
		t.Skip("not an offscreen window")
	}
	img := off.Buffer()
	ts := style.TooltipStyleOf(win.Look())
	b := bubble.Bounds()
	// The band the second line sits in, inside the bubble.
	y0 := int(b.Min.Y + ts.Face.Height())
	y1 := int(b.Min.Y + ts.Face.Height()*2)
	ink := 0
	for y := y0; y < y1 && y < img.Height; y++ {
		for x := int(b.Min.X); x < int(b.Max.X) && x < img.Width; x++ {
			if r, g, bl, _ := img.PremulAt(x, y); r < 200 || g < 200 || bl < 200 {
				ink++
			}
		}
	}
	if ink < 20 {
		t.Fatalf("the second line of the tip was not painted (%d dark pixels in %v)", ink, paintengine2d.XYWH(b.Min.X, float32(y0), b.Dx(), float32(y1-y0)))
	}
}
