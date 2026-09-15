package platform

import (
	"math"
	"time"
)

// kinetic keeps a touchpad scroll going after the fingers lift, slowing
// down as GTK's and the phones' kinetic scrolling does: the release
// velocity decays exponentially.
type kinetic struct {
	samples [8]kinSample
	n       int
	vx, vy  float32 // pixels per second
	last    time.Time
	next    time.Time
	running bool
}

type kinSample struct {
	t      time.Time
	dx, dy float32
}

const (
	// kinWindow is how far back the release velocity looks.
	kinWindow = 100 * time.Millisecond
	// kinTau is the decay's time constant.
	kinTau = 0.325
	// kinMinSpeed starts, and stops, a fling (pixels per second).
	kinMinSpeed = 60
	// kinFrame is how often a fling scrolls.
	kinFrame = 16 * time.Millisecond
)

// sample records a finger scroll; it stops a fling in progress.
func (k *kinetic) sample(now time.Time, dx, dy float32) {
	k.running = false
	k.samples[k.n%len(k.samples)] = kinSample{now, dx, dy}
	k.n++
}

// release starts a fling from the scroll's recent velocity (the fingers
// lifted); it reports whether one started.
func (k *kinetic) release(now time.Time) bool {
	if k.n == 0 {
		// The other axis's stop: the fling is already set.
		return k.running
	}
	var sx, sy float32
	first := now
	for i := 0; i < len(k.samples) && i < k.n; i++ {
		s := k.samples[i]
		if now.Sub(s.t) > kinWindow {
			continue
		}
		sx += s.dx
		sy += s.dy
		if s.t.Before(first) {
			first = s.t
		}
	}
	k.n = 0
	span := now.Sub(first).Seconds()
	if span < 0.02 {
		span = 0.02
	}
	k.vx, k.vy = sx/float32(span), sy/float32(span)
	if math.Hypot(float64(k.vx), float64(k.vy)) < kinMinSpeed {
		k.running = false
		return false
	}
	k.running = true
	k.last = now
	k.next = now.Add(kinFrame)
	return true
}

// stop ends a fling (a click, a new scroll, the pointer leaving).
func (k *kinetic) stop() {
	k.running = false
	k.n = 0
}

// step is how far the fling scrolls from its last step to now; ok is false
// once it has come to rest.
func (k *kinetic) step(now time.Time) (dx, dy float32, ok bool) {
	if !k.running {
		return 0, 0, false
	}
	dt := now.Sub(k.last).Seconds()
	if dt <= 0 {
		return 0, 0, true
	}
	decay := math.Exp(-dt / kinTau)
	// The distance covered while the speed decays from v to v·decay.
	f := float32(kinTau * (1 - decay))
	dx, dy = k.vx*f, k.vy*f
	k.vx *= float32(decay)
	k.vy *= float32(decay)
	k.last = now
	k.next = now.Add(kinFrame)
	if math.Hypot(float64(k.vx), float64(k.vy)) < kinMinSpeed {
		k.running = false
	}
	return dx, dy, true
}
