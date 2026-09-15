package app

import (
	"os"
	"strings"
	"time"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// ColorSchemeEnv overrides the desktop's light / dark preference for the
// process (UITK_COLOR_SCHEME=dark ./app), to try a theme's dark sibling
// without switching the desktop. It matters only while the appearance
// follows the desktop.
const ColorSchemeEnv = "UITK_COLOR_SCHEME"

// desktopReadTimeout bounds how long New waits for the desktop portal; a
// slower answer arrives later and restyles the app.
const desktopReadTimeout = 300 * time.Millisecond

func styleScheme(s platform.ColorScheme) style.ColorScheme {
	switch s {
	case platform.SchemeDark:
		return style.SchemeDark
	case platform.SchemeLight:
		return style.SchemeLight
	}
	return style.SchemeNoPreference
}

// watchDesktop reads the desktop's appearance preferences and follows
// them for the life of the app: reduced motion always applies, the light /
// dark preference restyles an app whose appearance follows the desktop.
// Headless and offscreen apps never ask the desktop (screenshots and tests
// stay the same on every machine); ColorSchemeEnv still applies to them.
// The first read runs in the background; wait returns once it is in.
func (a *Application) watchDesktop() (wait func()) {
	if env := strings.TrimSpace(os.Getenv(ColorSchemeEnv)); env != "" {
		style.SetDesktopColorScheme(style.ParseColorScheme(env))
		a.schemeForced = true
	}
	if a.headless || a.backend == nil || a.backend.Name() == "offscreen" {
		return func() {}
	}
	type first struct {
		prefs platform.DesktopPrefs
		ok    bool
		stop  func()
	}
	ch := make(chan first, 1)
	go func() {
		prefs, ok, stop := platform.WatchDesktopPrefs(desktopReadTimeout, func(p platform.DesktopPrefs) {
			a.Post(func() { a.desktopPrefsChanged(p) })
		})
		ch <- first{prefs, ok, stop}
	}()
	return func() {
		r := <-ch
		a.desktopStop = r.stop
		if r.ok {
			style.SetDesktopReduceMotion(r.prefs.ReducedMotion)
			if !a.schemeForced {
				style.SetDesktopColorScheme(styleScheme(r.prefs.ColorScheme))
			}
		}
	}
}

// desktopPrefsChanged applies a change of the desktop's preferences on the
// UI goroutine.
func (a *Application) desktopPrefsChanged(p platform.DesktopPrefs) {
	style.SetDesktopReduceMotion(p.ReducedMotion)
	s := styleScheme(p.ColorScheme)
	if a.schemeForced || style.DesktopColorScheme() == s {
		return
	}
	style.SetDesktopColorScheme(s)
	if a.following {
		a.ReloadPreferredLook()
	}
}

// DesktopColorScheme is the desktop's light / dark preference
// (ColorSchemeEnv when set; no preference in a headless app).
func (a *Application) DesktopColorScheme() style.ColorScheme {
	return style.DesktopColorScheme()
}

// ApplyAppearance switches every window to ap (theme, corners, icons,
// motion, and following the desktop's light / dark preference) without
// saving it; SaveAppearance persists it for every app.
func (a *Application) ApplyAppearance(ap style.Appearance) {
	if a == nil {
		return
	}
	style.SetReduceMotion(ap.ReduceMotion)
	a.following = ap.FollowDesktop
	if a.look == nil {
		a.SetLook(ap.Look())
		return
	}
	a.SetLook(style.WithAppearance(a.look, ap))
}
