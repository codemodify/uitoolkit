package platform

import (
	"testing"
	"time"
)

func TestX11ClassifyDevice(t *testing.T) {
	cases := []struct {
		name  string
		props []string
		want  x11DeviceKind
	}{
		{"SynPS/2 Synaptics TouchPad", nil, x11DeviceTouchpad},
		{"ELAN0676:00 04F3:3195 Touchpad", []string{"libinput Tapping Enabled", "libinput Accel Speed"}, x11DeviceTouchpad},
		// Named like a mouse, but the driver says it taps.
		{"PNP0C50:00 0911:5288", []string{"libinput Tapping Enabled"}, x11DeviceTouchpad},
		{"bcm5974", []string{"Synaptics Finger"}, x11DeviceTouchpad},
		{"Logitech USB Optical Mouse", []string{"libinput Accel Speed", "libinput Natural Scrolling Enabled"}, x11DeviceMouse},
		{"Logitech MX Master", []string{"Evdev Wheel Emulation"}, x11DeviceMouse},
		// Xwayland's one pointer for every device, and XTest's.
		{"xwayland-pointer:17", []string{"libinput Accel Speed"}, x11DeviceUnknown},
		{"Virtual core XTEST pointer", nil, x11DeviceUnknown},
		{"Some Device", nil, x11DeviceUnknown},
	}
	for _, c := range cases {
		if got := x11ClassifyDevice(c.name, c.props); got != c.want {
			t.Errorf("%q %v: %v, want %v", c.name, c.props, got, c.want)
		}
	}
}

// scrollRig is an x11Scroller with a vertical and a horizontal valuator
// (numbers 2 and 3, 15 per notch as xf86-input-libinput has them) on each
// of three devices: a touchpad (4), a mouse (5) and one that cannot say (6).
func scrollRig() *x11Scroller {
	x := newX11Scroller()
	vals := func() []x11Valuator {
		return []x11Valuator{{number: 2, vertical: true, increment: 15}, {number: 3, increment: 15}}
	}
	x.setDevice(4, x11DeviceTouchpad, 15, vals())
	x.setDevice(5, x11DeviceMouse, 0, vals())
	x.setDevice(6, x11DeviceUnknown, 0, vals())
	return x
}

// A wheel is notches: Scroll ±1 per notch, never precise — three lines in
// a widget, as on Wayland — and a high-resolution wheel's eighths are
// eighths of that. It never glides.
func TestX11WheelIsNotchesAndNeverGlides(t *testing.T) {
	x := scrollRig()
	t0 := time.Unix(1000, 0)
	// The first value only starts the valuator.
	if _, _, _, ok := x.motion(t0, 5, map[int]float64{2: 300, 3: 0}); ok {
		t.Fatal("the first value scrolled")
	}
	for i := 1; i <= 6; i++ {
		dx, dy, precise, ok := x.motion(t0.Add(time.Duration(i)*8*time.Millisecond), 5, map[int]float64{2: 300 + float64(i)*15})
		if !ok || precise || dx != 0 || dy != 1 {
			t.Fatalf("notch %d: %v %v precise %v ok %v", i, dx, dy, precise, ok)
		}
	}
	dx, dy, precise, _ := x.motion(t0.Add(60*time.Millisecond), 5, map[int]float64{2: 390 + 15.0/8, 3: -15})
	if precise || dy != 0.125 || dx != -1 {
		t.Fatalf("an eighth of a notch and a notch left: %v %v precise %v", dx, dy, precise)
	}
	// However fast it spun, nothing is left to glide.
	if !x.wakeAt().IsZero() {
		t.Fatal("a wheel asked for a glide")
	}
	if _, _, ok := x.tick(t0.Add(time.Second)); ok {
		t.Fatal("a wheel glided")
	}
}

// Two fingers are pixels; when they lift — the scroll goes quiet — the
// scroll glides on from the velocity just before, slowing down, and comes
// to rest.
func TestX11FingerScrollGlides(t *testing.T) {
	x := scrollRig()
	x.scale = 1
	t0 := time.Unix(2000, 0)
	x.motion(t0, 4, map[int]float64{2: 0})
	v := 0.0
	var at time.Time
	for i := 1; i <= 8; i++ {
		// 0.8 of a notch every 10 ms: 12 px, 1200 px/s.
		v += 0.8 * 15
		at = t0.Add(time.Duration(i) * 10 * time.Millisecond)
		dx, dy, precise, ok := x.motion(at, 4, map[int]float64{2: v})
		if !ok || !precise || dx != 0 || dy < 11.99 || dy > 12.01 {
			t.Fatalf("finger step %d: %v %v precise %v", i, dx, dy, precise)
		}
	}
	// Still moving: no glide yet.
	if _, _, ok := x.tick(at.Add(20 * time.Millisecond)); ok {
		t.Fatal("glided while the fingers were still on the pad")
	}
	lift := at.Add(x11LiftGap)
	if w := x.wakeAt(); !w.Equal(lift) {
		t.Fatalf("wakes at %v, want the lift at %v", w, lift)
	}
	// The glide: every step shorter than the last, all of it about v·τ.
	now := lift
	total, last, steps := float32(0), float32(1e9), 0
	for ; steps < 500; steps++ {
		dx, dy, ok := x.tick(now)
		if !ok {
			if steps == 0 {
				t.Fatal("no glide after the fingers lifted")
			}
			if x.wakeAt().IsZero() {
				break
			}
			now = x.wakeAt()
			continue
		}
		if dx != 0 || dy <= 0 {
			t.Fatalf("glide step %d: %v %v", steps, dx, dy)
		}
		// Per unit of time, since the first step covers the gap too.
		if steps > 1 && dy > last+0.01 {
			t.Fatalf("the glide sped up at step %d: %v after %v", steps, dy, last)
		}
		last = dy
		total += dy
		now = now.Add(kinFrame)
	}
	if total < 250 || total > 500 {
		t.Fatalf("glided %v px, want about 1200 × 0.325", total)
	}
	if !x.wakeAt().IsZero() {
		t.Fatal("still gliding after coming to rest")
	}
}

// A new touch — the fingers scrolling again — stops a glide at once, and so
// do a click and the pointer leaving (stop).
func TestX11GlideStopsOnTouch(t *testing.T) {
	for _, how := range []string{"scroll", "stop"} {
		x := scrollRig()
		t0 := time.Unix(3000, 0)
		x.motion(t0, 4, map[int]float64{2: 0})
		v := 0.0
		for i := 1; i <= 6; i++ {
			v += 15
			x.motion(t0.Add(time.Duration(i)*10*time.Millisecond), 4, map[int]float64{2: v})
		}
		lift := t0.Add(60*time.Millisecond + x11LiftGap)
		if _, _, ok := x.tick(lift); !ok {
			t.Fatal("no glide")
		}
		switch how {
		case "scroll":
			x.motion(lift.Add(5*time.Millisecond), 4, map[int]float64{2: v + 0.1})
		case "stop":
			x.stop()
		}
		if x.kin.running {
			t.Fatalf("%s: the glide went on", how)
		}
		if _, _, ok := x.tick(lift.Add(20 * time.Millisecond)); ok {
			t.Fatalf("%s: a step after the touch", how)
		}
	}
	// A wheel notch in a glide stops it too.
	x := scrollRig()
	t0 := time.Unix(4000, 0)
	x.motion(t0, 4, map[int]float64{2: 0})
	for i := 1; i <= 6; i++ {
		x.motion(t0.Add(time.Duration(i)*10*time.Millisecond), 4, map[int]float64{2: float64(i) * 15})
	}
	lift := t0.Add(60*time.Millisecond + x11LiftGap)
	x.tick(lift)
	x.motion(lift, 5, map[int]float64{2: 0})
	x.motion(lift.Add(time.Millisecond), 5, map[int]float64{2: 15})
	if x.kin.running {
		t.Fatal("a wheel notch did not stop the glide")
	}
}

// A slow scroll, or fingers held still before they lift, does not glide.
func TestX11NoGlideWhenSlow(t *testing.T) {
	x := scrollRig()
	t0 := time.Unix(5000, 0)
	x.motion(t0, 4, map[int]float64{2: 0})
	for i := 1; i <= 6; i++ {
		x.motion(t0.Add(time.Duration(i)*20*time.Millisecond), 4, map[int]float64{2: float64(i) * 0.03})
	}
	if _, _, ok := x.tick(t0.Add(time.Second)); ok {
		t.Fatal("a slow scroll glided")
	}
}

// Where the device cannot say (Xwayland), the scroll does: whole 120ths of
// a notch are a wheel's, anything else a finger's, and a run that showed a
// finger stays one until it pauses.
func TestX11UnknownDeviceTellsFingersFromWheel(t *testing.T) {
	x := scrollRig()
	t0 := time.Unix(6000, 0)
	x.motion(t0, 6, map[int]float64{2: 0})
	_, dy, precise, _ := x.motion(t0.Add(10*time.Millisecond), 6, map[int]float64{2: 15})
	if precise || dy != 1 {
		t.Fatalf("a whole notch: %v precise %v", dy, precise)
	}
	_, dy, precise, _ = x.motion(t0.Add(20*time.Millisecond), 6, map[int]float64{2: 15 + 15.0*30/120})
	if precise || dy != 0.25 {
		t.Fatalf("a high-resolution quarter notch: %v precise %v", dy, precise)
	}
	// Fingers on Xwayland: surface pixels / 10, in accelerated fractions.
	base := 15 + 15.0*30/120
	_, dy, precise, _ = x.motion(t0.Add(30*time.Millisecond), 6, map[int]float64{2: base + 15*0.3710937})
	if !precise || dy < 3.7 || dy > 3.72 {
		t.Fatalf("a finger's fraction: %v precise %v", dy, precise)
	}
	// The next sample lands on a whole 120th by chance: still fingers.
	_, dy, precise, _ = x.motion(t0.Add(40*time.Millisecond), 6, map[int]float64{2: base + 15*0.3710937 + 15*0.25})
	if !precise || dy < 2.49 || dy > 2.51 {
		t.Fatalf("a finger's run turned into a wheel: %v precise %v", dy, precise)
	}
	// After a pause the device is a wheel again until it shows otherwise.
	x.stop()
	_, dy, precise, _ = x.motion(t0.Add(time.Second), 6, map[int]float64{2: base + 15*0.3710937 + 15*1.25})
	if precise || dy != 1 {
		t.Fatalf("after a pause, a notch: %v precise %v", dy, precise)
	}
}

// After the pointer enters a window (or the device changes) the valuators
// start again: the jump since the last value seen is not a scroll.
func TestX11ScrollResetOnEnter(t *testing.T) {
	x := scrollRig()
	t0 := time.Unix(7000, 0)
	x.motion(t0, 5, map[int]float64{2: 0})
	x.reset()
	if _, _, _, ok := x.motion(t0.Add(time.Second), 5, map[int]float64{2: 900}); ok {
		t.Fatal("scrolled by the jump across the reset")
	}
	if _, dy, _, ok := x.motion(t0.Add(time.Second+time.Millisecond), 5, map[int]float64{2: 915}); !ok || dy != 1 {
		t.Fatalf("after the reset: %v %v", dy, ok)
	}
	// Motion that moves no scroll valuator scrolls nothing; a device with
	// none is not a scroll device at all.
	if _, _, _, ok := x.motion(t0.Add(2*time.Second), 5, map[int]float64{0: 10, 1: 20}); ok {
		t.Fatal("pointer motion scrolled")
	}
	if x.hasScroll(9) || !x.hasScroll(5) {
		t.Fatal("hasScroll")
	}
}

// Where the server says the valuators stand (read at a device switch or an
// enter) is where the next scroll is measured from: the first notch after
// is a notch, not a starting point.
func TestX11ScrollSeededValues(t *testing.T) {
	x := scrollRig()
	t0 := time.Unix(8000, 0)
	x.reset()
	x.seed(5, 2, 42, true)
	x.seed(5, 3, 7, true)
	x.seed(9, 2, 1, true) // no such device: nothing
	_, dy, precise, ok := x.motion(t0, 5, map[int]float64{2: 57})
	if !ok || precise || dy != 1 {
		t.Fatalf("the first notch after a seed: %v precise %v ok %v", dy, precise, ok)
	}
	dx, _, _, ok := x.motion(t0.Add(time.Millisecond), 5, map[int]float64{3: 22})
	if !ok || dx != 1 {
		t.Fatalf("horizontal after a seed: %v %v", dx, ok)
	}
	// An unknown value seeds nothing: the next one is a start again.
	x.seed(5, 2, 0, false)
	if _, _, _, ok := x.motion(t0.Add(2*time.Millisecond), 5, map[int]float64{2: 500}); ok {
		t.Fatal("scrolled from an unknown start")
	}
}
