package platform

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// The identity a Linux desktop files this application's windows under.
//
// On Wayland it is `xdg_toplevel.set_app_id`, on X11 the `WM_CLASS`
// hint, and a desktop uses it for everything that treats windows as
// belonging to a program: which task-bar button they share, which icon
// they wear, which `.desktop` file they match, which window rules apply.
//
// Both were hardcoded to "uitoolkit". Every application built with this
// toolkit therefore claimed the same identity, and a desktop did exactly
// what it was told — a mail client and a password vault, two unrelated
// programs, shared one task-bar entry and one icon with no way for the
// user to separate them.
//
// Windows and macOS have no such bug to fix: they identify a program by
// its executable, which is already distinct. So this is a Linux-shaped
// problem with a Linux-shaped answer, not a seam the other backends owe
// an implementation.
//
// The default is the executable's own name, which is right far more often
// than a constant is. An application that ships a `.desktop` file should
// set the id to match it — that is the only way the desktop can tie the
// two together.
var appID struct {
	mu sync.Mutex
	id string
}

// SetAppID sets the identity this application's windows carry. It takes
// effect for windows created afterwards; a desktop reads the id when a
// window is mapped and does not re-read it.
//
// The name is sanitized to what the protocols allow. An empty string
// restores the default.
func SetAppID(id string) {
	appID.mu.Lock()
	appID.id = sanitizeAppID(id)
	appID.mu.Unlock()
}

// AppID is the identity this application's windows carry: what
// [SetAppID] was given, or the executable's own name.
func AppID() string {
	appID.mu.Lock()
	id := appID.id
	appID.mu.Unlock()
	if id != "" {
		return id
	}
	return defaultAppID()
}

// AppClass is AppID in the capitalized form X11's WM_CLASS carries as its
// class, beside the id itself as the instance name.
func AppClass() string {
	id := AppID()
	if id == "" {
		return "Uitoolkit"
	}
	r := []rune(id)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

var defaultOnce struct {
	sync.Once
	id string
}

// defaultAppID is the executable's base name, which is what a desktop
// would guess anyway and what distinguishes two programs built with this
// toolkit from each other.
func defaultAppID() string {
	defaultOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil || exe == "" {
			exe = os.Args[0]
		}
		name := sanitizeAppID(filepath.Base(exe))
		if name == "" {
			name = "uitoolkit"
		}
		defaultOnce.id = name
	})
	return defaultOnce.id
}

// sanitizeAppID keeps what both protocols accept and a desktop will match
// on: letters, digits, dot, dash and underscore. A go-build temporary
// binary's name ("exe", "main") is left alone — it is still better than
// every program sharing one id.
func sanitizeAppID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".exe")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), ".-_")
	if len(out) > 128 {
		out = out[:128]
	}
	return out
}
