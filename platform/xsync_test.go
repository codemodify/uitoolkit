package platform

import (
	"testing"
	"time"
)

// The handshake: a request is answered by the first frame after the
// configure that resized the window, not by one before it; a configure
// that kept the size is answered at once; a later request replaces one
// still waiting; nothing is ever answered twice.
func TestSyncRequestStateMachine(t *testing.T) {
	t0 := time.Unix(100, 0)
	var r syncRequest
	if ok, _ := r.presented(); ok {
		t.Fatal("a frame answered with no request")
	}
	if ok, _ := r.configured(true, t0); ok {
		t.Fatal("a configure answered with no request")
	}

	r.request(syncValue(7, 0), t0)
	if ok, _ := r.presented(); ok {
		t.Fatal("a frame before the configure answered the request")
	}
	if ok, _ := r.configured(true, t0); ok {
		t.Fatal("a resize was answered before its frame")
	}
	ok, v := r.presented()
	if !ok || v != 7 {
		t.Fatalf("the frame at the new size answered %v %d, want 7", ok, v)
	}
	if ok, _ := r.presented(); ok {
		t.Fatal("a second frame answered the same request")
	}

	// A configure that kept the size: nothing to draw, answered at once.
	r.request(syncValue(8, 0), t0)
	if ok, v := r.configured(false, t0); !ok || v != 8 {
		t.Fatalf("an unchanged size answered %v %d, want 8 at once", ok, v)
	}
	if ok, _ := r.presented(); ok {
		t.Fatal("the frame after it answered again")
	}

	// The manager moved on: only the latest value is answered.
	r.request(syncValue(9, 0), t0)
	r.request(syncValue(10, 0), t0)
	r.configured(true, t0)
	if ok, v := r.presented(); !ok || v != 10 {
		t.Fatalf("answered %v %d, want the latest request, 10", ok, v)
	}
}

// A request nobody answers is answered after syncOverdue, whether its
// configure came (an app that did not repaint) or not; the deadline says
// when, so an idle loop wakes for it.
func TestSyncRequestOverdue(t *testing.T) {
	t0 := time.Unix(100, 0)
	var r syncRequest
	if !r.deadline(syncOverdue).IsZero() {
		t.Fatal("a deadline with nothing waiting")
	}
	r.request(syncValue(3, 0), t0)
	if d := r.deadline(syncOverdue); !d.Equal(t0.Add(syncOverdue)) {
		t.Fatalf("deadline %v, want %v", d, t0.Add(syncOverdue))
	}
	if ok, _ := r.overdue(t0.Add(syncOverdue/2), syncOverdue); ok {
		t.Fatal("answered before it was overdue")
	}
	r.configured(true, t0.Add(10*time.Millisecond))
	if ok, v := r.overdue(t0.Add(10*time.Millisecond+syncOverdue), syncOverdue); !ok || v != 3 {
		t.Fatalf("overdue answered %v %d, want 3", ok, v)
	}
	if ok, _ := r.presented(); ok {
		t.Fatal("the late frame answered again")
	}
}

// The value's halves: data.l[2] is the low word, data.l[3] the high one.
func TestSyncValue(t *testing.T) {
	if v := syncValue(0xdeadbeef, 0x1); v != 0x1deadbeef {
		t.Fatalf("%#x", v)
	}
	if lo, hi := uint32(syncValue(5, 9)), uint32(syncValue(5, 9)>>32); lo != 5 || hi != 9 {
		t.Fatalf("round trip %d %d", lo, hi)
	}
}
