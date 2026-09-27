//go:build linux

package platform

import (
	"syscall"
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

// Drain leaves the pipe empty: a poller watching its descriptor would
// otherwise find it readable for ever.
func TestFDWakerDrainEmpties(t *testing.T) {
	w := newFDWaker()
	defer w.Close()
	for i := 0; i < 3; i++ {
		w.Signal()
	}
	w.Drain()
	var buf [8]byte
	if n, err := syscall.Read(w.FD(), buf[:]); n > 0 || err != syscall.EAGAIN {
		t.Fatalf("pipe still holds data after Drain: n=%d err=%v", n, err)
	}
}

// A full wake pipe must not stop Signal returning. The pipe is
// O_NONBLOCK, but its write end is registered with the Go runtime
// poller, and os.File.Write on a registered descriptor parks the
// goroutine rather than answering EAGAIN. Nothing guarantees a drain —
// WakeSurface signals this pipe for every post while the X11 and
// offscreen waits never read it — so a run of posts fills the buffer and
// the next Signal blocks before it can wake the surface. Posting from
// the UI thread then deadlocks the loop against itself.
func TestWakeSignalNeverBlocksOnAFullPipe(t *testing.T) {
	w := newFDWaker()
	if w.FD() < 0 {
		t.Skip("no pipe")
	}
	// Fill it through the raw descriptor, so the state under test is the
	// one a run of undrained posts leaves behind.
	rc, err := w.w.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	filled := 0
	_ = rc.Write(func(fd uintptr) bool {
		buf := make([]byte, 4096)
		for {
			n, err := syscall.Write(int(fd), buf)
			if n <= 0 || err != nil {
				return true
			}
			filled += n
		}
	})
	if filled == 0 {
		t.Skip("could not fill the pipe")
	}

	done := make(chan struct{})
	go func() { w.Signal(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Signal blocked on a full pipe: a post would deadlock the loop")
	}

	// And it still wakes a waiter once there is room, so refusing to
	// block has not cost the wake itself.
	w.Drain()
	w.Signal()
	if !w.Wait(time.Second) {
		t.Fatal("a signalled waker did not wake")
	}
}
