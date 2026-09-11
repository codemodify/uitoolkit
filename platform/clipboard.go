package platform

import "sync"

// In-memory clipboard. OS clipboard is out of scope for v0.1 (documented).
// TextField copy / cut / paste stubs talk to this buffer.
var (
	clipMu sync.Mutex
	clip   string
)

// ClipboardGet returns the last ClipboardSet value (may be empty).
func ClipboardGet() string {
	clipMu.Lock()
	defer clipMu.Unlock()
	return clip
}

// ClipboardSet stores text for later ClipboardGet.
func ClipboardSet(s string) {
	clipMu.Lock()
	clip = s
	clipMu.Unlock()
}
