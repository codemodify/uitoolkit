package platform

import "testing"

// Window geometry is logical pixels; a backend whose window system wants
// device pixels converts at its own boundary. The two halves have to be
// exact inverses at the scales people actually run, or a window would
// creep by a pixel every time its own resize came back to it.
func TestLogicalToDeviceRoundTrip(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		for _, v := range []int{1, 8, 96, 104, 116, 275, 468, 540, 780, 1281} {
			dev := DevicePixels(v, scale)
			if want := int(float32(v)*scale + 0.5); dev != want && dev != want-1 {
				t.Errorf("%.2fx: DevicePixels(%d) = %d, want about %d", scale, v, dev, want)
			}
			if back := LogicalPixels(dev, scale); back != v {
				t.Errorf("%.2fx: %d -> %d device -> %d logical", scale, v, dev, back)
			}
		}
	}
}

// The compact player: 275 by 116 design pixels is 275 by 116 at 1 and 481
// by 203 at 1.75 — the numbers the e2e report measured.
func TestTheStripsPixelsAtEveryScale(t *testing.T) {
	for _, c := range []struct {
		scale float32
		w, h  int
		wantW int
		wantH int
	}{
		{1, 275, 116, 275, 116},
		{1.25, 275, 116, 344, 145},
		{1.5, 275, 116, 413, 174},
		{1.75, 275, 116, 481, 203},
		{2, 275, 116, 550, 232},
	} {
		if got := DevicePixels(c.w, c.scale); got != c.wantW {
			t.Errorf("%.2fx: width %d, want %d", c.scale, got, c.wantW)
		}
		if got := DevicePixels(c.h, c.scale); got != c.wantH {
			t.Errorf("%.2fx: height %d, want %d", c.scale, got, c.wantH)
		}
	}
}

// An offscreen surface is a backend whose geometry is device pixels, like
// X11's: the window keeps the logical size it was asked for and the pixmap
// behind it is that times the scale.
func TestOffscreenSizeIsLogical(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		o := NewOffscreen(WindowOptions{Width: 275, Height: 116, Scale: scale})
		wantW, wantH := DevicePixels(275, scale), DevicePixels(116, scale)
		if w, h := o.Size(); w != wantW || h != wantH {
			t.Errorf("%.2fx: pixmap %dx%d, want %dx%d", scale, w, h, wantW, wantH)
		}
		if err := o.Resize(468, 104); err != nil {
			t.Fatal(err)
		}
		wantW, wantH = DevicePixels(468, scale), DevicePixels(104, scale)
		if w, h := o.Size(); w != wantW || h != wantH {
			t.Errorf("%.2fx: after Resize, pixmap %dx%d, want %dx%d", scale, w, h, wantW, wantH)
		}
		// The resize it reports is logical, so feeding it back changes
		// nothing.
		var ev Event
		for _, e := range o.Poll() {
			if e.Kind == EventResize {
				ev = e
			}
		}
		if ev.Width != 468 || ev.Height != 104 {
			t.Errorf("%.2fx: EventResize %dx%d, want 468x104", scale, ev.Width, ev.Height)
		}
	}
}
