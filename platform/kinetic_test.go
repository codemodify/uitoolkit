package platform

import (
	"testing"
	"time"
)

// A quick swipe keeps gliding and comes to rest; a slow drag, or one held
// still before lifting, does not fling.
func TestKineticFling(t *testing.T) {
	var k kinetic
	t0 := time.Unix(1000, 0)
	for i := 0; i < 6; i++ {
		k.sample(t0.Add(time.Duration(i)*10*time.Millisecond), 0, 12) // 1200 px/s
	}
	if !k.release(t0.Add(60 * time.Millisecond)) {
		t.Fatal("a quick swipe should fling")
	}
	// Wayland stops each axis separately: the second stop keeps the fling.
	if !k.release(t0.Add(60*time.Millisecond)) || !k.running {
		t.Fatal("the other axis's stop cancelled the fling")
	}
	total, steps := float32(0), 0
	now := t0.Add(60 * time.Millisecond)
	for steps < 1000 {
		now = now.Add(kinFrame)
		_, dy, ok := k.step(now)
		total += dy
		steps++
		if !ok || !k.running {
			break
		}
	}
	// It glides about v·τ (~1200 × 0.325 ≈ 390px), and stops within a
	// couple of seconds.
	if total < 250 || total > 450 {
		t.Fatalf("the fling glided %v px", total)
	}
	if time.Duration(steps)*kinFrame > 3*time.Second {
		t.Fatalf("the fling lasted %v steps", steps)
	}
	// Slow: no fling.
	for i := 0; i < 6; i++ {
		k.sample(t0.Add(time.Duration(i)*20*time.Millisecond), 0, 0.5)
	}
	if k.release(t0.Add(120 * time.Millisecond)) {
		t.Fatal("a slow drag should not fling")
	}
	// Held still for a while before lifting: the samples are too old.
	k.sample(t0, 0, 30)
	if k.release(t0.Add(500 * time.Millisecond)) {
		t.Fatal("lifting after a pause should not fling")
	}
	// A new scroll stops a fling.
	k.sample(t0, 0, 20)
	k.sample(t0.Add(10*time.Millisecond), 0, 20)
	k.release(t0.Add(20 * time.Millisecond))
	k.sample(t0.Add(30*time.Millisecond), 0, 1)
	if k.running {
		t.Fatal("touching the pad again stops the fling")
	}
}
