package app

import (
	"os"
	"time"

	"github.com/codemodify/uitoolkit/style"
)

// lookWatchInterval is how often Run re-stats the watched files while idle.
// PumpOnce always checks once. Polling keeps the watcher dependency-free
// (no fsnotify) and fits the existing waitDisplay timeout.
const lookWatchInterval = 300 * time.Millisecond

// lookSettleWindow keeps reading a file whose stamp matches but whose mtime
// is within this distance of now, so a coarse overlay clock cannot hide a
// same-second rewrite. The distance is absolute: a file stamped in the
// future (restored backup, VM clock skew) must not pin the watcher into
// re-reading on every poll.
const lookSettleWindow = time.Second

// fileStamp is one watched file: size + mtime, with the content kept so a
// same-length rewrite still counts as a change.
type fileStamp struct {
	path string
	raw  string
	size int64
	mod  time.Time
}

func newFileStamp(path string) *fileStamp {
	s := &fileStamp{path: path}
	s.refresh()
	return s
}

func (s *fileStamp) refresh() {
	if s == nil {
		return
	}
	st, err := os.Stat(s.path)
	if err != nil {
		s.size = 0
		s.mod = time.Time{}
		s.raw = ""
		return
	}
	s.size = st.Size()
	s.mod = st.ModTime()
	s.raw = readLookFile(s.path)
}

// changed re-stats the file and reports whether its content moved.
func (s *fileStamp) changed() bool {
	if s == nil || s.path == "" {
		return false
	}
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
	if st.Size() == s.size && st.ModTime().Equal(s.mod) && absDuration(time.Since(st.ModTime())) > lookSettleWindow {
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

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// lookFileStamp watches look.json plus the theme.json of the pack it
// currently selects, so editing a user pack applies without a restart.
// Idle polls stat size+mtime and only ReadFile when those change (Mail
// WatchLook used to read the file every 300ms).
type lookFileStamp struct {
	// path is look.json, the primary watched file.
	path  string
	look  *fileStamp
	pack  *fileStamp
	reads int
}

func newLookFileStamp() *lookFileStamp {
	s := &lookFileStamp{path: style.AppearancePath()}
	s.look = newFileStamp(s.path)
	s.reads++
	s.followPack()
	return s
}

// followPack re-points the pack stamp at the theme.json backing the
// appearance look.json currently selects (empty for a builtin pack).
func (s *lookFileStamp) followPack() {
	if s == nil {
		return
	}
	path := style.ThemeSourceFile(style.LoadAppearance().Name)
	if path == "" {
		s.pack = nil
		return
	}
	if s.pack != nil && s.pack.path == path {
		return
	}
	s.pack = newFileStamp(path)
	s.reads++
}

func readLookFile(path string) string {
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// refreshMeta re-reads the stamps of every watched file (used after the
// caller changes mtimes behind the watcher's back).
func (s *lookFileStamp) refreshMeta() {
	if s == nil {
		return
	}
	s.look.refresh()
	if s.pack != nil {
		s.pack.refresh()
	}
}

// changed reports whether look.json or the selected pack file moved.
func (s *lookFileStamp) changed() bool {
	if s == nil {
		return false
	}
	hit := false
	if s.look.changed() {
		hit = true
		s.reads++
	}
	if s.pack != nil && s.pack.changed() {
		hit = true
		s.reads++
	}
	if hit {
		s.followPack()
	}
	return hit
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
	// Icon files may have changed with the set; drop cached stats so the
	// new set is picked up on the next paint.
	style.InvalidateIconCache()
	next := style.LoadAppearance()
	style.SetReduceMotion(next.ReduceMotion)
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
