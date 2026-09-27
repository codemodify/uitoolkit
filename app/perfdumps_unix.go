//go:build !windows

package app

import (
	"os"
	"syscall"
)

// perfDumpSignals are the signals that ask the perf log for a profile:
// SIGUSR1 for a heap profile and the goroutines, SIGUSR2 for five seconds
// of CPU profile.
func perfDumpSignals() (heap, cpu os.Signal) { return syscall.SIGUSR1, syscall.SIGUSR2 }
