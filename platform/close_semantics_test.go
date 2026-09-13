package platform

import (
	"sync"
	"testing"
)

// TestEventCloseIsARequest pins the contract every backend now follows:
// EventClose is a request from the window manager / compositor, and the
// surface stays usable until the application calls Close.
func TestEventCloseIsARequest(t *testing.T) {
	s := NewOffscreen(WindowOptions{Width: 16, Height: 16})
	s.Inject(Event{Kind: EventClose})
	evs := s.Poll()
	if len(evs) != 1 || evs[0].Kind != EventClose {
		t.Fatalf("Poll = %+v, want one EventClose", evs)
	}
	if s.Closed() {
		t.Fatal("EventClose must not close the surface: an app that hides to tray keeps using it")
	}
	if !SurfaceVisible(s) {
		t.Fatal("surface should still be visible after a close request")
	}
	// The app vetoes the close and hides instead.
	HideSurface(s)
	if s.Closed() {
		t.Fatal("Hide must not close the surface")
	}
	if SurfaceVisible(s) {
		t.Fatal("Hide should clear visibility")
	}
	RaiseSurface(s)
	if !SurfaceVisible(s) {
		t.Fatal("Raise should map the surface again")
	}
	// Only Close closes.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !s.Closed() {
		t.Fatal("Closed must be true after Close")
	}
}

// TestPollDrainsQueueAfterClose guards the other half of the contract: a
// queued EventClose must survive the surface being closed between Wait
// and Poll, or a run loop can miss the event entirely.
func TestPollDrainsQueueAfterClose(t *testing.T) {
	s := NewOffscreen(WindowOptions{Width: 8, Height: 8})
	s.Inject(Event{Kind: EventKeyDown, Key: KeyEscape})
	s.Inject(Event{Kind: EventClose})
	_ = s.Close()
	evs := s.Poll()
	if len(evs) != 2 {
		t.Fatalf("Poll after Close = %d events, want 2", len(evs))
	}
	if evs[1].Kind != EventClose {
		t.Fatalf("last event = %v, want EventClose", evs[1].Kind)
	}
}

// TestInvokeStatusSerializes checks tray callbacks without a Dispatch
// hook cannot run concurrently (run with -race to see the difference).
func TestInvokeStatusSerializes(t *testing.T) {
	var (
		wg      sync.WaitGroup
		n       int
		inside  bool
		overlap bool
		mu      sync.Mutex
	)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			invokeStatus(nil, func() {
				mu.Lock()
				if inside {
					overlap = true
				}
				inside = true
				mu.Unlock()
				n++ // unsynchronized on purpose: -race proves serialization
				mu.Lock()
				inside = false
				mu.Unlock()
			})
		}()
	}
	wg.Wait()
	if overlap {
		t.Fatal("tray callbacks ran concurrently without a Dispatch hook")
	}
	if n != 16 {
		t.Fatalf("ran %d callbacks, want 16", n)
	}
}

// TestInvokeStatusUsesDispatch verifies the UI-thread hook still wins.
func TestInvokeStatusUsesDispatch(t *testing.T) {
	var posted int
	invokeStatus(func(fn func()) { posted++; fn() }, func() {})
	if posted != 1 {
		t.Fatalf("dispatch called %d times, want 1", posted)
	}
	invokeStatus(func(fn func()) { posted++ }, nil)
	if posted != 1 {
		t.Fatalf("nil callback must not dispatch (posted=%d)", posted)
	}
}
