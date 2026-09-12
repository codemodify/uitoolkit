package app

import (
	"os"
	"time"

	"github.com/codemodify/uitoolkit/style"
)

// lookWatchInterval is how often Run re-stats look.json while idle.
// PumpOnce always checks once. Polling keeps the watcher dependency-free
// (no fsnotify) and fits the existing waitDisplay timeout.
const lookWatchInterval = 300 * time.Millisecond

// lookFileStamp is the last-seen contents of look.json. Size+mtime is not
// enough: dark/round/classic and light/square/sharp JSON can be the same
// length, and overlay clocks may not bump mtime.
type lookFileStamp struct {
	path string
	raw  string
}

func newLookFileStamp() *lookFileStamp {
	s := &lookFileStamp{path: style.AppearancePath()}
	s.raw = readLookFile(s.path)
	return s
}

func readLookFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *lookFileStamp) changed() bool {
	raw := readLookFile(s.path)
	if raw == s.raw {
		return false
	}
	s.raw = raw
	return true
}

// WatchingLook reports whether this application reloads look.json at runtime.
func (a *Application) WatchingLook() bool { return a.watchLook }

// ReloadPreferredLook applies XDG look.json (theme pack name) onto the
// current look, keeping display scale and density (WithAppearance).
// Always SetLook when the file changed: skipping on Appearance equality
// can no-op if LookAppearance is stale or a pack name matches while
// palette / corners / icons do not.
func (a *Application) ReloadPreferredLook() {
	if a == nil || a.look == nil {
		return
	}
	next := style.LoadAppearance()
	a.SetLook(style.WithAppearance(a.look, next))
}

func (a *Application) pollLookFile() {
	if a == nil || !a.watchLook || a.lookWatch == nil {
		return
	}
	if !a.lookWatch.changed() {
		return
	}
	a.ReloadPreferredLook()
}
