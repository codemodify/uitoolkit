package app

import (
	"sync/atomic"
	"testing"
	"time"
)

// After a burst of allocation the loop hands the free heap back once, when
// it has stayed idle for trimIdle; activity pushes the trim back.
func TestTrimAfterIdle(t *testing.T) {
	var trims atomic.Int32
	old := trimHeap
	trimHeap = func() { trims.Add(1) }
	defer func() { trimHeap = old }()
	settle := func() {
		for i := 0; i < 100 && trims.Load() == 0; i++ {
			time.Sleep(time.Millisecond)
		}
	}

	a := &Application{}
	t0 := time.Unix(1000, 0)
	a.maybeTrim(t0)
	if !a.trimDeadline().IsZero() {
		t.Fatal("nothing owed, nothing scheduled")
	}
	a.owesTrim()
	a.maybeTrim(t0) // arms the deadline
	if d := a.trimDeadline(); !d.Equal(t0.Add(trimIdle)) {
		t.Fatalf("deadline %v, want trimIdle after the burst", d)
	}
	a.maybeTrim(t0.Add(trimIdle / 2))
	a.stirred() // an event: the trim waits for quiet again
	a.maybeTrim(t0.Add(trimIdle - time.Millisecond))
	a.maybeTrim(t0.Add(trimIdle + time.Millisecond))
	settle()
	if trims.Load() != 0 {
		t.Fatal("trimmed while the app was still busy")
	}
	a.maybeTrim(t0.Add(2*trimIdle + time.Millisecond))
	settle()
	if trims.Load() != 1 {
		t.Fatalf("%d trims after an idle trimIdle, want 1", trims.Load())
	}
	a.maybeTrim(t0.Add(10 * trimIdle))
	time.Sleep(5 * time.Millisecond)
	if trims.Load() != 1 || !a.trimDeadline().IsZero() {
		t.Fatal("a trim is owed only once per burst")
	}
}
