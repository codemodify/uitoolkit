package style

import (
	"sync"

	"github.com/codemodify/paintengine2d"
)

// AccentEngine is an engine whose looks take the user's accent colour, the
// way Windows 10 and 11, macOS, Plasma, GNOME and Material You recolour
// their controls around one. Accented returns tok recoloured around
// accent: the selection, the default button, focus and checked marks, and
// whatever the platform derives from them. It must not write to tok's maps
// (they belong to the registered pack); [CloneTokenMaps] gives it its own.
type AccentEngine interface {
	Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens
}

var desktopAccent struct {
	sync.RWMutex
	c  paintengine2d.Color
	ok bool
}

// SetDesktopAccent records the desktop's accent colour for the process
// (ok false: the desktop names none). Looks that follow the desktop take
// it when their engine is an [AccentEngine]; the app package keeps it
// current.
func SetDesktopAccent(c paintengine2d.Color, ok bool) {
	desktopAccent.Lock()
	desktopAccent.c, desktopAccent.ok = c, ok
	desktopAccent.Unlock()
}

// DesktopAccent is the colour last recorded by [SetDesktopAccent].
func DesktopAccent() (paintengine2d.Color, bool) {
	desktopAccent.RLock()
	defer desktopAccent.RUnlock()
	return desktopAccent.c, desktopAccent.ok
}

// TakesAccent reports whether the pack is recoloured around an accent:
// its engine takes one, and for this pack (not GNOME 3's Adwaita, not
// Yosemite, not Windows 7 Basic) it changes something. Two probe accents,
// so a pack whose own accent is one of them still answers.
func TakesAccent(p ThemePack) bool {
	ae, ok := engineFor(p.Tokens).(AccentEngine)
	if !ok {
		return false
	}
	tok := p.Tokens.Resolve()
	for _, probe := range []string{"#e95420", "#26a269"} {
		if !sameTokens(ae.Accented(tok, Hex(probe)), tok) {
			return true
		}
	}
	return false
}

// sameTokens compares what an AccentEngine may change.
func sameTokens(a, b ThemeTokens) bool {
	if a.Palette != b.Palette || a.Hot != b.Hot || a.Pressed != b.Pressed || a.Selected != b.Selected || a.Focus != b.Focus {
		return false
	}
	if len(a.Extra) != len(b.Extra) || len(a.Params) != len(b.Params) {
		return false
	}
	for k, v := range a.Extra {
		if w, ok := b.Extra[k]; !ok || w != v {
			return false
		}
	}
	for k, v := range a.Params {
		if w, ok := b.Params[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// withDesktopAccent recolours tok around the desktop's accent when its
// engine takes one.
func withDesktopAccent(tok ThemeTokens) ThemeTokens {
	c, ok := DesktopAccent()
	if !ok {
		return tok
	}
	if ae, ok := engineFor(tok).(AccentEngine); ok {
		return ae.Accented(tok, c.WithAlpha(1))
	}
	return tok
}

// CloneTokenMaps gives tok its own Extra and Params maps, so an
// [AccentEngine] can change them without touching the registered pack.
func CloneTokenMaps(tok ThemeTokens) ThemeTokens {
	extra := make(map[string]paintengine2d.Color, len(tok.Extra)+8)
	for k, v := range tok.Extra {
		extra[k] = v
	}
	params := make(map[string]float32, len(tok.Params)+2)
	for k, v := range tok.Params {
		params[k] = v
	}
	tok.Extra, tok.Params = extra, params
	return tok
}
