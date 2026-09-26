package app

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// What a window tells the desktop about dressing it: the colours of the
// desktop's own frame, and the window's icon (platform/windowdress.go).

// EnvDecorationPalette set to 0 keeps the desktop's frame in the desktop's
// own colours: no window names a colour scheme for it. A testing switch,
// and the escape hatch for a user who wants every title bar alike.
const EnvDecorationPalette = "UITK_DECORATION_PALETTE"

// syncDecorationPalette tells a desktop that paints its own frame in a
// colour scheme of the window's choosing (KWin) to paint the window's in
// its look's: a Luna window under KWin's frame gets a blue Breeze title
// bar, as a KDE app with its own colour scheme gets its colours. It runs
// whenever the window's look or its frame changes; the surface puts the
// palette on the wire only when it changed.
//
// The scheme goes to the desktop even while the toolkit draws the frame,
// so a live switch to the desktop's frame (Settings' "OS window borders") comes up
// in the right colours at once.
func (w *Window) syncDecorationPalette() {
	if w == nil || w.surf == nil || w.opts.Popup || os.Getenv(EnvDecorationPalette) == "0" {
		return
	}
	ps, ok := w.surf.(platform.DecorationPaletteSurface)
	if !ok || !ps.DecorationPaletteSupported() {
		return
	}
	path, err := w.app.decorationPaletteFile(w.look)
	if err != nil {
		if !w.app.paletteWarned {
			w.app.paletteWarned = true
			log.Printf("uitoolkit: the desktop's frame keeps its own colours: %v", err)
		}
		return
	}
	ps.SetDecorationPalette(path)
}

// decorationPaletteFile is the KDE colour scheme of lk, written once under
// the user's cache directory ($XDG_CACHE_HOME/uitoolkit/colors) and named
// after its contents: a desktop that caches a scheme by its path (KWin
// does) is never shown a stale one, and every window and every app in the
// same look shares one file. Nothing is written under the user's config
// directory, where KDE keeps the schemes the user chose.
func (a *Application) decorationPaletteFile(lk style.LookAndFeel) (string, error) {
	if lk == nil {
		return "", fmt.Errorf("no look")
	}
	// Remembered by look (a look is a pointer, or at least comparable, in
	// every case but a curiosity): the scheme is painted from the frame,
	// which is worth doing once.
	memo := reflect.TypeOf(lk).Comparable()
	if memo {
		a.mu.Lock()
		path, ok := a.palettes[lk]
		a.mu.Unlock()
		if ok {
			return path, nil
		}
	}
	name := "uitoolkit"
	if n := lk.Name(); n != "" {
		name = "uitoolkit " + n
	}
	data := style.KDEColorScheme(lk, name)
	h := fnv.New64a()
	h.Write(data)
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "uitoolkit", "colors")
	path := filepath.Join(dir, fmt.Sprintf("uitk-%016x.colors", h.Sum64()))
	if _, err := os.Stat(path); err != nil {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		// Written aside and renamed into place: KWin may read the file the
		// moment another window names it.
		tmp, err := os.CreateTemp(dir, ".uitk-*.colors")
		if err != nil {
			return "", err
		}
		_, werr := tmp.Write(data)
		cerr := tmp.Close()
		if werr != nil || cerr != nil {
			os.Remove(tmp.Name())
			return "", fmt.Errorf("writing %s: %w", path, errors.Join(werr, cerr))
		}
		if err := os.Rename(tmp.Name(), path); err != nil {
			os.Remove(tmp.Name())
			return "", err
		}
	}
	if memo {
		a.mu.Lock()
		if a.palettes == nil {
			a.palettes = map[style.LookAndFeel]string{}
		}
		a.palettes[lk] = path
		a.mu.Unlock()
	}
	return path, nil
}

// SetIcon gives every window of the application its icon — the one the
// desktop shows in the window's title bar, its task switcher and its task
// bar — as square images at the sizes the app has (16 to 64 pixels, and a
// large one for switchers that show it big); the desktop picks the size it
// needs. Windows that set their own ([Window.SetIcon]) keep it. Without an
// icon a desktop falls back on the application's desktop entry, which a
// program run from its build tree has none of.
//
// Wayland states it with xdg-toplevel-icon-v1, X11 with _NET_WM_ICON.
func (a *Application) SetIcon(images ...*paintengine2d.Image) {
	if a == nil {
		return
	}
	a.icon = images
	for _, w := range a.Windows() {
		if w.icon == nil {
			w.applyIcon()
		}
	}
}

// Icon is the application's icon (SetIcon).
func (a *Application) Icon() []*paintengine2d.Image { return a.icon }

// SetIcon gives this window an icon of its own instead of the
// application's ([Application.SetIcon]); none goes back to the
// application's.
func (w *Window) SetIcon(images ...*paintengine2d.Image) {
	if w == nil {
		return
	}
	w.icon = images
	w.applyIcon()
}

// applyIcon hands the window's icon — its own, else the application's —
// to its surface.
func (w *Window) applyIcon() {
	if w == nil || w.surf == nil || w.opts.Popup {
		return
	}
	icon := w.icon
	if icon == nil {
		icon = w.app.icon
	}
	if icon == nil && !w.iconSent {
		return
	}
	w.iconSent = platform.SurfaceSetIcon(w.surf, icon) && icon != nil
}
