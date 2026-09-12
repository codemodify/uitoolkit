//go:build !linux

package platform

import "time"

func signalLoopWake() {}

func waitLoopWake(timeout time.Duration) bool {
	if timeout == 0 {
		return false
	}
	if timeout < 0 || timeout > 50*time.Millisecond {
		timeout = 50 * time.Millisecond
	}
	time.Sleep(timeout)
	return false
}

func loopWakeFD() int { return -1 }
