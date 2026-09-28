package platform

import "runtime"

// Win32 delivers a window's messages to the thread that created it, so
// that thread has to stay put. Go will otherwise move the main goroutine
// to another OS thread at any call, and then ShowWindow, SetWindowPos and
// every other windowing call becomes a cross-thread SendMessage to a
// thread that is not pumping — which does not fail, it blocks for ever.
//
// [app] pins the main thread too, for the same reason and with the longer
// argument. This is here as well because package platform is usable on
// its own — cmd/uitk-winsmoke uses it with no app — and a backend whose
// correctness depends on a rule enforced one layer up is not a backend,
// it is a trap. Measured before this existed: ShowWindow wedged in four
// runs out of five.
//
// Locking in an init function pins the goroutine that runs package
// initialisation, which is the main goroutine on the process's main
// thread, for the life of the program.
func init() { runtime.LockOSThread() }
