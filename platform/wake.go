package platform

import (
	"sync/atomic"
	"time"
)

// loopWakeCount is incremented by [WakeLoop] so tests can assert Post
// actually pokes the display wait.
var loopWakeCount atomic.Int32

// WakeLoop interrupts a blocking [WaitDisplay] (eventfd on Linux,
// no-op sleep elsewhere) so [Application.Post] does not wait out the
// tray 100ms cap or a Wayland poll.
func WakeLoop() {
	loopWakeCount.Add(1)
	signalLoopWake()
}

// LoopWakes is the number of [WakeLoop] calls since process start.
func LoopWakes() int { return int(loopWakeCount.Load()) }

func waitOrSleep(timeout time.Duration) bool {
	if timeout == 0 {
		return false
	}
	if woken := waitLoopWake(timeout); woken {
		return true
	}
	return false
}
