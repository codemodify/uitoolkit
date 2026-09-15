package app

import (
	"runtime/debug"
	"time"
)

// trimIdle is how long an app stays idle after a burst of allocation
// (start-up, a new window, a theme change, new content) before it hands
// its free heap back to the OS. Go keeps up to about twice the live heap
// between collections, and a GUI then sits idle holding the difference:
// Mail kept some 25 MB it no longer used.
const trimIdle = 2 * time.Second

// trimHeap returns free heap memory to the OS (a var so tests can count).
var trimHeap = debug.FreeOSMemory

// owesTrim notes a burst of allocation; the run loop trims once it has
// then stayed idle for trimIdle. Safe from any goroutine.
func (a *Application) owesTrim() {
	if a != nil {
		a.trimOwed.Store(true)
	}
}

// stirred records activity (an event, a painted frame), which pushes a
// pending trim back.
func (a *Application) stirred() {
	if a != nil {
		a.stir.Add(1)
	}
}

// maybeTrim runs on the loop goroutine every iteration. Activity pushes a
// pending trim to trimIdle from now; once the loop has stayed idle that
// long it trims, on a goroutine of its own so the UI never waits for the
// collection.
func (a *Application) maybeTrim(now time.Time) {
	if s := a.stir.Load(); s != a.stirSeen {
		a.stirSeen = s
		a.trimAt = now.Add(trimIdle)
		return
	}
	if !a.trimOwed.Load() {
		return
	}
	if a.trimAt.IsZero() {
		a.trimAt = now.Add(trimIdle)
		return
	}
	if now.Before(a.trimAt) {
		return
	}
	a.trimOwed.Store(false)
	a.trimAt = time.Time{}
	go trimHeap()
}

// trimDeadline is when the loop must wake to trim (zero: nothing owed).
func (a *Application) trimDeadline() time.Time {
	if !a.trimOwed.Load() {
		return time.Time{}
	}
	return a.trimAt
}
