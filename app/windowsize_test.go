package app

import (
	"math"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// scales are the display scales a shaped, skinned app is tested at: 1, the
// three fractional ones a KDE or GNOME user actually picks, and 2.
var scales = []float32{1, 1.25, 1.5, 1.75, 2}

func devPx(v int, s float32) int { return int(math.Round(float64(float32(v) * s))) }

// TestWindowSizeIsLogicalAtEveryScale is the defect: a window's size was
// device pixels on X11 and logical pixels on Wayland with nothing to
// reconcile them, so the same SetSize(275, 116) gave two different windows
// at 1.75. It is logical everywhere now — the same numbers give the same
// window — and the device pixels behind it grow with the scale.
func TestWindowSizeIsLogicalAtEveryScale(t *testing.T) {
	const dw, dh = 275, 116
	for _, s := range scales {
		t.Setenv(platform.EnvDecorations, "")
		a := New(Options{Look: style.DarkLook(), Headless: true, Scale: s})
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "Strip", Width: dw, Height: dh, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(widgets.NewColumn(widgets.NewLabel("strip")))
		a.PumpOnce()

		if got, gotH := w.Size(); got != dw || gotH != dh {
			t.Errorf("%.2fx: Size %dx%d, want the %dx%d it asked for", s, got, gotH, dw, dh)
		}
		wantW, wantH := devPx(dw, s), devPx(dh, s)
		if got, gotH := w.PixelSize(); got != wantW || gotH != wantH {
			t.Errorf("%.2fx: PixelSize %dx%d, want %dx%d", s, got, gotH, wantW, wantH)
		}
		// And the buffer is at least that: the surface is the window plus
		// whatever margin a frame keeps for its shadow.
		if bw, bh := w.SurfaceSize(); bw < wantW || bh < wantH {
			t.Errorf("%.2fx: surface %dx%d is smaller than the window %dx%d", s, bw, bh, wantW, wantH)
		}

		// SetSize speaks the same units, and a window resized to a design
		// it already has does not move.
		const cw, ch = 468, 104
		w.SetSize(cw, ch)
		a.PumpOnce()
		if got, gotH := w.Size(); got != cw || gotH != ch {
			t.Errorf("%.2fx: after SetSize, Size %dx%d, want %dx%d", s, got, gotH, cw, ch)
		}
		if got, gotH := w.PixelSize(); got != devPx(cw, s) || gotH != devPx(ch, s) {
			t.Errorf("%.2fx: after SetSize, PixelSize %dx%d, want %dx%d",
				s, got, gotH, devPx(cw, s), devPx(ch, s))
		}
		w.Close()
	}
}

// TestWindowSizeSurvivesTheRoundTrip: the resize the backend reports is in
// the same units, so a window that echoes its own resize event back (which
// is what Window.dispatch does) stays the size it was.
func TestWindowSizeSurvivesTheRoundTrip(t *testing.T) {
	for _, s := range scales {
		t.Setenv(platform.EnvDecorations, "")
		a := New(Options{Look: style.DarkLook(), Headless: true, Scale: s})
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "Echo", Width: 320, Height: 240, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(widgets.NewColumn(widgets.NewLabel("x")))
		a.PumpOnce()
		for i := 0; i < 3; i++ {
			ww, hh := w.Size()
			w.dispatch(platform.Event{Kind: platform.EventResize, Width: ww, Height: hh})
			a.PumpOnce()
		}
		if got, gotH := w.Size(); got != 320 || gotH != 240 {
			t.Errorf("%.2fx: %dx%d after three echoed resizes, want 320x240", s, got, gotH)
		}
		w.Close()
	}
}

// TestMinAndMaxAreLogicalToo: the limits handed to the window system are
// stated in the same units as the size they limit.
func TestMinAndMaxAreLogicalToo(t *testing.T) {
	for _, s := range scales {
		t.Setenv(platform.EnvDecorations, "")
		a := New(Options{Look: style.DarkLook(), Headless: true, Scale: s})
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "Limits", Width: 400, Height: 300, Headless: true,
			MinWidth: 200, MinHeight: 120, MaxWidth: 800, MaxHeight: 600})
		if err != nil {
			t.Fatal(err)
		}
		a.PumpOnce()
		want := platform.SizeLimits{MinWidth: 200, MinHeight: 120, MaxWidth: 800, MaxHeight: 600}
		if got := platform.SurfaceSizeLimits(w.Surface()); got != want {
			t.Errorf("%.2fx: limits %+v, want %+v", s, got, want)
		}
		w.Close()
	}
}
