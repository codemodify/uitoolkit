package platform

import "sync"

// In-memory clipboard, always updated by ClipboardSet. On Linux with CGO
// this also owns the OS selection: Wayland wl_data_device (and primary
// when offered) or X11 CLIPBOARD + PRIMARY, so Ctrl+C/X/V and
// middle-click talk to the session.
//
// Threading: call ClipboardGet / ClipboardPrimaryGet / ClipboardSet from
// the goroutine that runs the event loop, like any other [Surface] call.
// They talk to the same X11 / Wayland connection the loop does, and the
// read path dispatches incoming protocol events (which run this package's
// C callbacks) while it waits for the selection owner to answer. The
// in-memory fallback is mutex-guarded, but the native path is not
// goroutine-safe.
//
// Reads are bounded: a selection owner that never answers costs a short
// timeout, not a hung UI.
var (
	clipMu sync.Mutex
	clip   string
)

// ClipboardGet returns CLIPBOARD text (from the session when a native
// backend is live, otherwise the in-process buffer). Call it from the UI
// goroutine; see the threading note above.
func ClipboardGet() string {
	if s, ok := clipboardNativeGet(); ok {
		return s
	}
	clipMu.Lock()
	defer clipMu.Unlock()
	return clip
}

// ClipboardPrimaryGet returns the X11 PRIMARY selection, or the in-process
// buffer when there is no native clipboard.
func ClipboardPrimaryGet() string {
	if s, ok := clipboardNativePrimaryGet(); ok {
		return s
	}
	clipMu.Lock()
	defer clipMu.Unlock()
	return clip
}

// ClipboardSet stores text for later ClipboardGet and, when a native
// backend is live, offers it to the session as both CLIPBOARD and
// PRIMARY. Call it from the UI goroutine; see the threading note above.
//
// Ownership is a promise to serve the data on request. Both backends
// keep answering after the last window closes — X11 from a background
// drainer, Wayland from the connection the clipboard holds open — so
// another application's paste cannot hang on a selection nobody is
// listening for.
func ClipboardSet(s string) {
	clipMu.Lock()
	clip = s
	clipMu.Unlock()
	clipboardNativeSet(s)
}
