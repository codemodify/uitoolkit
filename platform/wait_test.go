package platform

import (
	"testing"
	"time"
)

func TestWaitDisplayOffscreen(t *testing.T) {
	s := NewOffscreen(WindowOptions{Width: 8, Height: 8})
	if WaitDisplay(s, 0) {
		t.Fatal("empty queue, timeout 0")
	}
	s.Inject(Event{Kind: EventExpose})
	if !s.Wait(0) {
		t.Fatal("queued event should wake")
	}
	start := time.Now()
	_ = WaitDisplay(s, 5*time.Millisecond)
	if time.Since(start) > 80*time.Millisecond {
		t.Fatal("offscreen wait should not block long")
	}
}

func TestWaitMillis(t *testing.T) {
	if waitMillis(-1) != -1 {
		t.Fatal("infinite")
	}
	if waitMillis(0) != 0 {
		t.Fatal("zero")
	}
	if waitMillis(1500*time.Microsecond) < 1 {
		t.Fatal("sub-ms rounds up")
	}
}
