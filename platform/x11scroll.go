package platform

import (
	"math"
	"strings"
	"time"
)

// X11 smooth scrolling (XInput 2.1) and the glide after it.
//
// An X server with XInput 2.1 reports scrolling as motion on a device's
// scroll valuators: a running total per axis, and an increment that is one
// wheel notch. What the valuator does not say is where the scroll came
// from — XInput has no axis source, and no "the fingers lifted" either —
// so the backend works both out here, as GTK and Qt do on X11:
//
//   - The device. A touchpad is one whose XInput properties say so (the
//     libinput driver gives only touchpads "libinput Tapping Enabled";
//     synaptics has "Synaptics Finger") or whose name does. A device with
//     scroll valuators and neither is a mouse — except where the device
//     cannot say at all: Xwayland has one pointer for every physical
//     device, so there the scroll itself is looked at.
//   - The scroll, for such a device: a wheel moves in whole 120ths of a
//     notch (that is what a high-resolution wheel reports, and Xwayland
//     passes value120 on as it is), while two fingers move by an
//     accelerated fraction of a pixel that almost never lands on one.
//     A run of scrolling that has shown a finger's fraction stays a
//     finger's until it pauses.
//   - The lift. Fingers on a pad report every few milliseconds while they
//     move; a scroll that has gone quiet for x11LiftGap has ended, and it
//     ended when it went quiet. The glide starts from the velocity just
//     before then, with the same curve as Wayland's (kinetic).
//
// A wheel notch is Scroll ±1 with ScrollPrecise false — three lines in
// every widget, as on Wayland — and a wheel never glides. A finger's
// scroll is ScrollPrecise pixels: pxPerNotch per notch, which is the
// driver's own "libinput Scrolling Pixel Distance" where it has one and
// Xwayland's 10 (it divides wl_pointer.axis by 10) where it does not.

// x11LiftGap is how long a finger scroll may go quiet before the fingers
// are taken to have lifted. Touchpads report at 60–140 Hz while fingers
// move.
const x11LiftGap = 60 * time.Millisecond

// x11BurstGap is how long a pause ends a run of scrolling for telling a
// finger's from a wheel's.
const x11BurstGap = 150 * time.Millisecond

// x11PxPerNotch is a finger scroll's pixels per scroll increment where the
// driver does not say: Xwayland's divisor.
const x11PxPerNotch = 10

// x11DeviceKind is what a scroll device is known to be.
type x11DeviceKind int8

const (
	x11DeviceUnknown x11DeviceKind = iota
	x11DeviceMouse
	x11DeviceTouchpad
)

// x11ClassifyDevice says what a pointer device is from its name and the
// names of its XInput properties.
func x11ClassifyDevice(name string, props []string) x11DeviceKind {
	for _, p := range props {
		switch p {
		case "libinput Tapping Enabled", "Synaptics Finger", "Synaptics Two-Finger Scrolling":
			return x11DeviceTouchpad
		}
	}
	n := strings.ToLower(name)
	for _, w := range []string{"touchpad", "trackpad", "synaptics", "glidepoint", "clickpad"} {
		if strings.Contains(n, w) {
			return x11DeviceTouchpad
		}
	}
	if strings.HasPrefix(n, "xwayland-") || strings.Contains(n, "xtest") {
		// One device for every mouse and pad on the Wayland side (or a
		// synthetic one): it cannot say.
		return x11DeviceUnknown
	}
	for _, p := range props {
		if strings.HasPrefix(p, "libinput ") || strings.HasPrefix(p, "Evdev ") {
			return x11DeviceMouse
		}
	}
	return x11DeviceUnknown
}

// x11Valuator is one scroll valuator of a device.
type x11Valuator struct {
	number    int
	vertical  bool
	increment float64
	last      float64
	have      bool
}

// x11ScrollDevice is a pointer device with scroll valuators.
type x11ScrollDevice struct {
	kind       x11DeviceKind
	pxPerNotch float32
	vals       []x11Valuator
}

// x11Scroller turns valuator motion into scroll events and keeps a finger
// scroll gliding after the fingers lift. It knows nothing about X itself:
// the backend feeds it (x11_xi2_linux.go), and tests drive it with times
// of their own.
type x11Scroller struct {
	devices map[int]*x11ScrollDevice
	kin     kinetic
	// scale converts a finger scroll's pixels to device pixels (the
	// display scale the toolkit draws at).
	scale float32

	// The current run of scrolling: when it last moved, and whether it
	// has shown itself to be fingers.
	lastAt  time.Time
	fingers bool
	// pending is a finger scroll that has not been seen to end yet.
	pending bool
}

func newX11Scroller() *x11Scroller {
	return &x11Scroller{devices: map[int]*x11ScrollDevice{}, scale: 1}
}

// setDevice records (or replaces) a device's scroll valuators.
func (x *x11Scroller) setDevice(id int, kind x11DeviceKind, pxPerNotch float32, vals []x11Valuator) {
	if len(vals) == 0 {
		delete(x.devices, id)
		return
	}
	if pxPerNotch <= 0 {
		pxPerNotch = x11PxPerNotch
	}
	x.devices[id] = &x11ScrollDevice{kind: kind, pxPerNotch: pxPerNotch, vals: vals}
}

// hasScroll reports whether device id scrolls through valuators — so its
// wheel buttons are emulated, and ignored.
func (x *x11Scroller) hasScroll(id int) bool { return x.devices[id] != nil }

// reset forgets every valuator's last value: after the pointer entered a
// window or the device changed, the next value is a new start, not a
// scroll by the difference.
func (x *x11Scroller) reset() {
	for _, d := range x.devices {
		for i := range d.vals {
			d.vals[i].have = false
		}
	}
}

// isWheelStep reports whether a scroll of notches is a whole number of
// 120ths of a notch — a wheel's, high-resolution or not.
func isWheelStep(notches float64) bool {
	v := notches * 120
	return math.Abs(v-math.Round(v)) < 1e-3
}

// motion takes one motion event's valuators (number → absolute value) from
// device id at now, and returns the scroll it makes: dx, dy, and whether it
// is precise (pixels) rather than notches. ok is false when it scrolled
// nothing.
func (x *x11Scroller) motion(now time.Time, id int, values map[int]float64) (dx, dy float32, precise, ok bool) {
	d := x.devices[id]
	if d == nil {
		return 0, 0, false, false
	}
	var nx, ny float64
	for i := range d.vals {
		v := &d.vals[i]
		val, in := values[v.number]
		if !in {
			continue
		}
		prev, had := v.last, v.have
		v.last, v.have = val, true
		if !had || v.increment == 0 {
			continue
		}
		n := (val - prev) / v.increment
		if v.vertical {
			ny += n
		} else {
			nx += n
		}
	}
	if nx == 0 && ny == 0 {
		return 0, 0, false, false
	}
	if now.Sub(x.lastAt) > x11BurstGap {
		x.fingers = false
	}
	x.lastAt = now
	switch d.kind {
	case x11DeviceTouchpad:
		x.fingers = true
	case x11DeviceMouse:
		x.fingers = false
	default:
		if !isWheelStep(nx) || !isWheelStep(ny) {
			x.fingers = true
		}
	}
	if !x.fingers {
		// A wheel: notches, and a notch stops a glide.
		x.kin.stop()
		x.pending = false
		return float32(nx), float32(ny), false, true
	}
	k := d.pxPerNotch * x.scale
	dx, dy = float32(nx)*k, float32(ny)*k
	x.kin.sample(now, dx, dy)
	x.pending = true
	return dx, dy, true, true
}

// stop ends a glide and forgets a pending finger scroll (a button press,
// the pointer leaving).
func (x *x11Scroller) stop() {
	x.kin.stop()
	x.pending = false
}

// wakeAt is when tick has something to do: the moment a quiet finger
// scroll counts as lifted, or the glide's next step. Zero for never.
func (x *x11Scroller) wakeAt() time.Time {
	if x.pending {
		return x.lastAt.Add(x11LiftGap)
	}
	if x.kin.running {
		return x.kin.next
	}
	return time.Time{}
}

// tick starts a glide once a finger scroll has gone quiet, and steps a
// glide in progress. It returns the scroll to deliver now (precise
// pixels); ok is false when there is none.
func (x *x11Scroller) tick(now time.Time) (dx, dy float32, ok bool) {
	if x.pending && now.Sub(x.lastAt) >= x11LiftGap {
		x.pending = false
		// The fingers lifted when the scroll went quiet: the velocity
		// is the one just before, and the glide has been under way
		// since then.
		if !x.kin.release(x.lastAt) {
			return 0, 0, false
		}
	}
	if !x.kin.running || now.Before(x.kin.next) {
		return 0, 0, false
	}
	dx, dy, ok = x.kin.step(now)
	if !ok || (dx == 0 && dy == 0) {
		return 0, 0, false
	}
	return dx, dy, true
}
