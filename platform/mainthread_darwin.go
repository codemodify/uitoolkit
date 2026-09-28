//go:build darwin

package platform

import "runtime"

// AppKit requires NSApplication and every NSWindow to be touched from
// the process's *first* thread — not a consistent thread, that one. A
// window made anywhere else does not draw, and running the event loop
// elsewhere is undefined.
//
// Locking in an init function pins the goroutine that runs package
// initialisation, which is the main goroutine on the main thread, for
// the life of the program. [app] does the same for the same reason;
// this is here as well because package platform is usable on its own,
// and a backend whose correctness depends on a rule enforced one layer
// up is not a backend, it is a trap. See mainthread_windows.go, where
// the same omission cost four runs in five.
func init() { runtime.LockOSThread() }
