package platform

import "sync"

// In-memory clipboard, always updated by ClipboardSet. On Linux with CGO
// this also owns the OS selection: Wayland wl_data_device (and primary
// when offered) or X11 CLIPBOARD + PRIMARY, so Ctrl+C/X/V and
// middle-click talk to the session.
var (
	clipMu sync.Mutex
	clip   string
)

// ClipboardGet returns CLIPBOARD text (OS when X11 owns or can convert,
// otherwise the in-process buffer).
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

// ClipboardSet stores text for later ClipboardGet and, when X11 is live,
// offers it as both CLIPBOARD and PRIMARY.
func ClipboardSet(s string) {
	clipMu.Lock()
	clip = s
	clipMu.Unlock()
	clipboardNativeSet(s)
}

func memoryClipboard() string {
	clipMu.Lock()
	defer clipMu.Unlock()
	return clip
}
