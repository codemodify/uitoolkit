package app

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"
)

// EnvPerfLog names a file the run loop appends one line to for every frame
// it paints, for tools/perf/measure.sh:
//
//	frame <end, unix ns> <µs spent> <window|popup>
//	trim <unix ns>
//
// With it set, SIGUSR1 also writes a heap profile (after a collection) to
// <file>.heap.<n>.pb.gz and a line with the goroutine count and the live
// heap, and <file>.goroutines.<n>.txt with every goroutine's stack. Unset,
// the loop takes no timings at all.
const EnvPerfLog = "UITK_PERF_LOG"

var perf struct {
	once sync.Once
	mu   sync.Mutex
	f    *os.File
}

// perfOn opens the log the first time it is asked and reports whether
// frames are being timed.
func perfOn() bool {
	perf.once.Do(func() {
		path := os.Getenv(EnvPerfLog)
		if path == "" {
			return
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return
		}
		perf.f = f
		go perfDumps(path)
	})
	return perf.f != nil
}

// perfFrame logs one painted frame that began at t0.
func perfFrame(t0 time.Time, what string) {
	end := time.Now()
	perf.mu.Lock()
	fmt.Fprintf(perf.f, "frame %d %d %s\n", end.UnixNano(), end.Sub(t0).Microseconds(), what)
	perf.mu.Unlock()
}

// perfMark logs a moment worth lining the frames up against: "trim <unix ns>"
// for an idle trim.
func perfMark(what string) {
	perf.mu.Lock()
	fmt.Fprintf(perf.f, "%s %d\n", what, time.Now().UnixNano())
	perf.mu.Unlock()
}

// perfDumps writes a heap profile and the goroutines on every SIGUSR1.
func perfDumps(path string) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	for n := 1; ; n++ {
		<-ch
		runtime.GC()
		if f, err := os.Create(fmt.Sprintf("%s.heap.%d.pb.gz", path, n)); err == nil {
			_ = pprof.WriteHeapProfile(f)
			f.Close()
		}
		if f, err := os.Create(fmt.Sprintf("%s.goroutines.%d.txt", path, n)); err == nil {
			_ = pprof.Lookup("goroutine").WriteTo(f, 2)
			f.Close()
		}
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		perf.mu.Lock()
		fmt.Fprintf(perf.f, "dump %d goroutines %d heap-live %d heap-sys %d heap-released %d stack %d other %d\n",
			n, runtime.NumGoroutine(), ms.HeapAlloc, ms.HeapSys, ms.HeapReleased, ms.StackSys,
			ms.Sys-ms.HeapSys-ms.StackSys)
		perf.mu.Unlock()
	}
}
