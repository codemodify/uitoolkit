package app

import "runtime"

// The event loop runs on the process's main OS thread, and this is what
// pins it there.
//
// runtime.LockOSThread in an init function locks the goroutine that runs
// package initialisation — the main goroutine, on the main thread — so
// main.main, and the [Application.Run] it calls, stay on that thread for
// the life of the program. It is the same thing GLFW, Gio and every other
// Go toolkit that talks to a window system does, for the same reason.
//
// Why it is not enough to say "one goroutine": Go may move a goroutine to
// another OS thread at any function call. On Linux that has been survivable
// — Xlib and libwayland want serialised access rather than one particular
// thread — so the toolkit got away with the weaker rule. Two of the three
// platforms it is heading for will not allow it:
//
//   - **AppKit** requires NSApplication and every NSWindow to be touched
//     from the process's *first* thread. Not a consistent thread — that
//     one. A window created anywhere else does not draw, and
//     [NSApp run] elsewhere is undefined.
//   - **Win32** delivers messages to the thread that created the HWND, and
//     OLE drag-and-drop (DoDragDrop, IDropTarget) requires that thread to
//     have entered a single-threaded apartment.
//
// Doing it now costs Linux nothing — the main goroutine was already where
// the loop ran — and means the rule is in force before there is a backend
// that depends on it, rather than being discovered by a window that will
// not draw.
//
// The cost, stated plainly: the main thread is no longer available to the
// scheduler for other goroutines. For a program whose main goroutine runs
// a UI event loop that was already true.
func init() { runtime.LockOSThread() }
