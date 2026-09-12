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

// lookFileStamp is the last-seen look.json. Idle polls Stat size+mtime
// and only ReadFile when those change (Mail WatchLook used to read the
// file every 300ms). Content is still compared after a meta change so
// same-length theme swaps apply.
type lookFileStamp struct {
	path string
	raw  string
	size int64
	mod  time.Time
}

func newLookFileStamp() *lookFileStamp {
	s := &lookFileStamp{path: style.AppearancePath()}
	s.refreshMeta()
	s.raw = readLookFile(s.path)
	return s
}

func (s *lookFileStamp) refreshMeta() {
	if s == nil {
		return
	}
	st, err := os.Stat(s.path)
	if err != nil {
		s.size = 0
		s.mod = time.Time{}
		return
	}
	s.size = st.Size()
	s.mod = st.ModTime()
}

func readLookFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *lookFileStamp) changed() bool {
	st, err := os.Stat(s.path)
	if err != nil {
		if s.raw == "" && s.size == 0 {
			return false
		}
		s.raw = ""
		s.size = 0
		s.mod = time.Time{}
		return true
	}
	if st.Size() == s.size && st.ModTime().Equal(s.mod) && time.Since(st.ModTime()) > time.Second {
		return false
	}
	s.size = st.Size()
	s.mod = st.ModTime()
	raw := readLookFile(s.path)
	if raw == s.raw {
		return false
	}
	s.raw = raw
	return true
}

// WatchingLook reports whether this application reloads look.json at runtime.
func (a *Application) WatchingLook() bool { return a.watchLook }

// ReloadPreferredLook applies XDG look.json (theme pack + icon set) onto
// the current look, keeping display scale and density (WithAppearance).
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
