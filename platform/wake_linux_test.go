//go:build linux

package platform

import (
	"testing"
	"time"
)

func TestFDWakerUnblocksWait(t *testing.T) {
	w := newFDWaker()
	if w.FD() < 0 {
		t.Fatal("eventfd")
	}
	t.Cleanup(w.Close)
	done := make(chan bool, 1)
	go func() {
		done <- w.Wait(500 * time.Millisecond)
	}()
	time.Sleep(20 * time.Millisecond)
	w.Signal()
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("Wait should return true after Signal")
		}
	case <-time.After(time.Second):
		t.Fatal("Wait did not wake")
	}
}

func TestWakeLoopIncrements(t *testing.T) {
	before := LoopWakes()
	WakeLoop()
	if LoopWakes() <= before {
		t.Fatal("WakeLoop count")
	}
	if !waitLoopWake(50 * time.Millisecond) {
		t.Fatal("waitLoopWake after WakeLoop")
	}
}
