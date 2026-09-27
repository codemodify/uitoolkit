package app

import "os"

// perfDumpSignals: Windows has no SIGUSR1 / SIGUSR2 — the Go runtime maps
// only the ANSI C signals there — so the perf log still records frames but
// takes no profile dumps on demand. Asking for one from outside the process
// wants a named event or a debug channel, which is not this file's job.
func perfDumpSignals() (heap, cpu os.Signal) { return nil, nil }
