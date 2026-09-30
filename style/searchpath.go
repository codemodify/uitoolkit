package style

import (
	"path/filepath"
	"sync"
)

// Where the toolkit looks for icon sets, themes and skins.
//
// ~/.config/uitoolkit/{icons,themes,skins} belongs to the **user**. It is
// the toolkit's equivalent of ~/.icons and ~/.themes: somewhere a person
// drops a set they like, once, and every uitoolkit program on the machine
// picks it up. Nothing but the person who owns the account should write
// there.
//
// An application that ships art of its own — its own icon set, a skin
// that is part of its identity, a theme it wants available — must
// therefore not copy anything into it. Two applications doing that
// overwrite each other, last one wins, and neither can tell: a program
// built against a newer toolkit installs stems an older program's copy
// then removes, and the older program's icons start coming back as the
// missing-icon placeholder with nothing anywhere to say why.
//
// So an application keeps its own art wherever it likes — beside the
// binary, in its own config directory, embedded and unpacked to a
// temporary directory — and calls [AddSearchPath] with that directory at
// start-up. It is private to that process.
//
// A registered directory has the same shape as the user's:
//
//	<dir>/icons/<set>/*.png
//	<dir>/themes/<name>/theme.json
//	<dir>/skins/<name>/skin.json
//
// **The user's own comes first, file by file.** Where the person has
// installed a set, their copy of a given icon wins; where they have
// nothing — a stem their copy predates, or a set they never installed —
// the application's answers. That keeps theming the user's to decide
// while letting a program rely on art it ships, which are the two things
// that were in tension.

var searchPaths struct {
	mu   sync.RWMutex
	dirs []string
	gen  uint64
}

// AddSearchPath registers a directory the application ships its own icon
// sets, themes or skins in, searched after the user's own.
//
// Call it before the first window is made. Registering the same
// directory twice is ignored, so it is safe to call from an init or from
// several packages that each need their own art.
//
// The directory is laid out like ~/.config/uitoolkit: an icons/, themes/
// or skins/ subdirectory, each holding named folders. Any of the three
// may be missing.
func AddSearchPath(dir string) {
	if dir == "" {
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	searchPaths.mu.Lock()
	defer searchPaths.mu.Unlock()
	for _, d := range searchPaths.dirs {
		if d == abs {
			return
		}
	}
	searchPaths.dirs = append(searchPaths.dirs, abs)
	searchPaths.gen++
	// The caches key on this, so a path registered after something has
	// already been drawn is picked up rather than ignored.
	InvalidateIconCache()
}

// SearchPaths are the directories registered with [AddSearchPath], in
// the order they were registered.
func SearchPaths() []string {
	searchPaths.mu.RLock()
	defer searchPaths.mu.RUnlock()
	out := make([]string, len(searchPaths.dirs))
	copy(out, searchPaths.dirs)
	return out
}

// searchGeneration changes whenever a path is registered, so caches
// keyed on it are dropped.
func searchGeneration() uint64 {
	searchPaths.mu.RLock()
	defer searchPaths.mu.RUnlock()
	return searchPaths.gen
}

// kindDirs are the directories to look in for one kind of thing
// ("icons", "themes", "skins"), the user's first.
func kindDirs(kind string) []string {
	searchPaths.mu.RLock()
	extra := make([]string, len(searchPaths.dirs))
	copy(extra, searchPaths.dirs)
	searchPaths.mu.RUnlock()

	out := make([]string, 0, len(extra)+1)
	out = append(out, filepath.Join(ConfigDir(), kind))
	for _, d := range extra {
		out = append(out, filepath.Join(d, kind))
	}
	return out
}

// IconSearchDirs are the icons/ directories, the user's first.
func IconSearchDirs() []string { return kindDirs("icons") }

// ThemeSearchDirs are the themes/ directories, the user's first.
func ThemeSearchDirs() []string { return kindDirs("themes") }

// SkinSearchDirs are the skins/ directories, the user's first.
func SkinSearchDirs() []string { return kindDirs("skins") }

// ResetSearchPathsForTest drops every registered path. Tests only.
func ResetSearchPathsForTest() {
	searchPaths.mu.Lock()
	searchPaths.dirs = nil
	searchPaths.gen++
	searchPaths.mu.Unlock()
	InvalidateIconCache()
}
