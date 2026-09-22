package app

import (
	"fmt"
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The window frames of the eras whose in-app window (DrawWindowFrame) was
// measured off the original pixel by pixel are held to it: the frame a
// window wears puts its title, its boxes and its bevels where the in-app
// frame of the same look puts them.

// fidelityWindow is a window in pack at scale whose frame the toolkit draws,
// with the caption buttons in layout and the default caption (the title
// alone), captured.
func fidelityWindow(t *testing.T, pack string, scale float32, layout, title string) (*Window, *paintengine2d.Image) {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	p, ok := style.LoadTheme(pack)
	if !ok {
		t.Fatalf("no %s pack", pack)
	}
	a := New(Options{Look: p.Look(), Headless: true, Scale: scale})
	prefs := platform.DefaultTitleBarPrefs("")
	prefs.Layout = platform.ParseButtonLayout(layout)
	a.SetTitleBarPrefs(prefs)
	w, err := a.NewWindow(platform.WindowOptions{Title: title, Width: 420, Height: 240, Headless: true, Decorations: platform.DecorationsClient})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewColumn(widgets.NewLabel("Body")))
	a.PumpOnce()
	img := w.Capture()
	if img == nil {
		t.Fatal("no capture")
	}
	return w, img
}

// inAppFrame is pack's in-app window frame, w by h device pixels, active,
// with a close and a zoom button and the title.
func inAppFrame(t *testing.T, lk style.LookAndFeel, w, h int, title string) *paintengine2d.Image {
	t.Helper()
	c, ok := lk.(*style.Classic)
	if !ok {
		t.Fatal("not an engine look")
	}
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	c.Engine().DrawWindowFrame(c, ctx, paintengine2d.XYWH(0, 0, float32(w), float32(h)), title,
		style.WindowState{Active: true, CanClose: true, Maximizable: true})
	return img
}

// lum is pixel (x, y)'s grey level, 0 to 255.
func lum(img *paintengine2d.Image, x, y int) int {
	p := img.NRGBAAt(x, y)
	return (int(p.R)*3 + int(p.G)*6 + int(p.B)) / 10
}

// inkSpan is the first and last column in rows [y0, y1) and columns
// [x0, x1) holding a pixel darker than dark: a line of black text.
func inkSpan(img *paintengine2d.Image, x0, x1, y0, y1, dark int) (first, last int) {
	first, last = -1, -1
	for x := max(x0, 0); x < min(x1, img.Width); x++ {
		for y := max(y0, 0); y < min(y1, img.Height); y++ {
			if lum(img, x, y) < dark {
				if first < 0 {
					first = x
				}
				last = x
				break
			}
		}
	}
	return first, last
}

func within(a, b, tol float32) bool { return float32(math.Abs(float64(a-b))) <= tol }

// BeOS: the tab keeps R5's gaps — the title 19 pixels after the close box
// and 17 before the zoom box — so the tab a window wears is exactly as wide
// as the in-app window's, and its title sits in the same place on it.
func TestBeOSTabKeepsR5Gaps(t *testing.T) {
	for _, sc := range []float32{1, 1.75} {
		t.Run(fmt.Sprint(sc), func(t *testing.T) {
			const title = "Tracker"
			w, img := fidelityWindow(t, "beos", sc, "close:maximize", title)
			lk := w.Look()
			hb := w.Caption()
			lead, trail := hb.Controls()
			spec := style.DecorationOf(lk, hb.DecorationState())
			u := spec.ButtonPad.Left / 4 // the tab's cell: R5's close box sits four in
			wr := w.WindowRect()
			tab := w.captionStrip()
			if !spec.CaptionFits || tab.Dx() >= wr.Dx() {
				t.Fatalf("the tab %v fills the window %v", tab, wr)
			}
			ref := inAppFrame(t, lk, int(wr.Dx()), int(wr.Dy()), title)
			// The in-app tab's width: its top row is the tab's dark line, the
			// panel grey beside it.
			refTab := 0
			for x := 0; x < ref.Width && lum(ref, x, 0) < 190; x++ {
				refTab = x + 1
			}
			if !within(tab.Max.X-wr.Min.X, float32(refTab), u) {
				t.Errorf("tab %v wide, R5's %d", tab.Max.X-wr.Min.X, refTab)
			}
			// The title's ink on each, from the tab's left corner.
			th := int(spec.Caption - 5*u)
			box := 18 * u
			rf, rl := inkSpan(ref, int(box)+1, refTab-int(box)-1, 2, th-1, 60)
			ox, oy := int(wr.Min.X), int(wr.Min.Y)
			ff, fl := inkSpan(img, int(widget.DeviceBounds(lead).Max.X)+1, int(widget.DeviceBounds(trail).Min.X)-1, oy+2, oy+th-1, 60)
			if rf < 0 || ff < 0 {
				t.Fatalf("no title ink: in-app %d, frame %d", rf, ff)
			}
			if !within(float32(ff-ox), float32(rf), u) || !within(float32(fl-ox), float32(rl), u) {
				t.Errorf("title ink %d..%d from the tab's corner, R5's %d..%d", ff-ox, fl-ox, rf, rl)
			}
			// And by R5's numbers: 19 after the close box's right edge.
			if gap := float32(ff) - widget.DeviceBounds(lead).Max.X; gap < 19*u-u || gap > 19*u+3*u {
				t.Errorf("title %v after the close box, want R5's %v", gap, 19*u)
			}
		})
	}
}

// Platinum: the title is centred on the whole bar, as Mac OS 8 centred it,
// not in the room between the close box and the collapse and zoom boxes.
func TestPlatinumTitleCentresOnTheBar(t *testing.T) {
	for _, sc := range []float32{1, 1.75} {
		t.Run(fmt.Sprint(sc), func(t *testing.T) {
			const title = "Untitled"
			w, img := fidelityWindow(t, "platinum", sc, "close:minimize,maximize", title)
			lk := w.Look()
			hb := w.Caption()
			wr := w.WindowRect()
			band, _ := hb.FrameParts()
			band = band.Translate(widget.DeviceOrigin(hb))
			lead, trail := hb.Controls()
			// Clear of the boxes' black outlines, whichever frame drew them.
			x0, x1 := widget.DeviceBounds(lead).Max.X+lk.(*style.Classic).S(8), widget.DeviceBounds(trail).Min.X-lk.(*style.Classic).S(8)
			// The title's middle rows: clear of the black line under the bar,
			// which the in-app bar puts elsewhere at a fractional scale.
			cy, ht := band.Min.Y+(band.Dy()-3*style.Dip(lk, 1))*0.5, lk.(*style.Classic).S(3)
			y0, y1 := int(cy-ht), int(cy+ht)
			ff, fl := inkSpan(img, int(x0), int(x1), y0, y1, 40)
			if ff < 0 {
				t.Fatal("no title ink")
			}
			mid := (float32(ff) + float32(fl+1)) * 0.5
			if !within(mid, (wr.Min.X+wr.Max.X)*0.5, 2*lk.(*style.Classic).S(1)) {
				t.Errorf("title's centre %v, the window's %v", mid, (wr.Min.X+wr.Max.X)*0.5)
			}
			ref := inAppFrame(t, lk, int(wr.Dx()), int(wr.Dy()), title)
			oy := int(wr.Min.Y)
			rf, rl := inkSpan(ref, int(x0-wr.Min.X), int(x1-wr.Min.X), y0-oy, y1-oy, 40)
			ox := int(wr.Min.X)
			if tol := lk.(*style.Classic).S(1) + 1; !within(float32(ff-ox), float32(rf), tol) || !within(float32(fl-ox), float32(rl), tol) {
				t.Errorf("title ink %d..%d, the in-app window's %d..%d", ff-ox, fl-ox, rf, rl)
			}
		})
	}
}

// Window Maker: the bar between the tiles is a piece of its own, bevelled
// — its highlight down the seam beside the miniaturize tile — as the in-app
// window draws it.
func TestWindowMakerMiddleSectionHighlight(t *testing.T) {
	for _, sc := range []float32{1, 1.75} {
		t.Run(fmt.Sprint(sc), func(t *testing.T) {
			w, img := fidelityWindow(t, "wmaker-default", sc, "minimize:close", "Window")
			hb := w.Caption()
			lead, trail := hb.Controls()
			band, _ := hb.FrameParts()
			band = band.Translate(widget.DeviceOrigin(hb))
			y := int(band.Min.Y + band.Dy()*0.5)
			seam := int(widget.DeviceBounds(lead).Max.X)
			end := int(widget.DeviceBounds(trail).Min.X)
			wr := w.WindowRect()
			ref := inAppFrame(t, w.Look(), int(wr.Dx()), int(wr.Dy()), "Window")
			ox, oy := int(wr.Min.X), int(wr.Min.Y)
			// The in-app window places its tiles on its own grid: find its
			// seams where a tile's black right edge meets the middle's
			// highlight, and where the middle's black edge meets the close
			// tile's.
			refSeam, refEnd := -1, -1
			for x := 1; x < ref.Width-1; x++ {
				if lum(ref, x-1, y-oy) < 12 && lum(ref, x, y-oy) > 120 {
					if refSeam < 0 {
						refSeam = x
					}
					refEnd = x
				}
			}
			if refSeam < 0 || refEnd <= refSeam {
				t.Fatalf("no seams in the in-app window (%d, %d)", refSeam, refEnd)
			}
			// The tile's edge and the middle's bevel on either side of it.
			for d := -2; d <= 2; d++ {
				for _, s := range [][2]int{{seam, refSeam}, {end, refEnd}} {
					got, want := lum(img, s[0]+d, y), lum(ref, s[1]+d, y-oy)
					if got-want > 12 || want-got > 12 {
						t.Errorf("%+d from the seam at %d: %d, the in-app window's %d", d, s[0]-ox, got, want)
					}
				}
			}
			// The highlight is lighter than the bar beside it.
			if lum(img, seam, y) <= lum(img, seam+3, y) {
				t.Errorf("no highlight down the seam: %d beside %d", lum(img, seam, y), lum(img, seam+3, y))
			}
		})
	}
}
