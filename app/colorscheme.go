package app

import (
	"os"
	"strings"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// ColorSchemeEnv overrides the desktop's light / dark preference for the
// process (UITK_COLOR_SCHEME=dark ./app), to try a theme's dark sibling
// without switching the desktop. It matters only while the appearance
// follows the desktop.
const ColorSchemeEnv = "UITK_COLOR_SCHEME"

func styleScheme(s platform.ColorScheme) style.ColorScheme {
	switch s {
	case platform.SchemeDark:
		return style.SchemeDark
	case platform.SchemeLight:
		return style.SchemeLight
	}
	return style.SchemeNoPreference
}

// followDesktop starts or stops tracking the desktop's light / dark
// preference: on while the appearance follows the desktop, off otherwise.
// Headless applications never ask the desktop (screenshots and tests stay
// the same on every machine); ColorSchemeEnv still applies to them.
func (a *Application) followDesktop(on bool) {
	if !on {
		if a.schemeStop != nil {
			a.schemeStop()
			a.schemeStop = nil
		}
		return
	}
	if a.schemeStop != nil {
		return
	}
	if !a.readDesktopScheme() {
		return
	}
	a.schemeStop = platform.WatchColorScheme(func(s platform.ColorScheme) {
		a.Post(func() { a.desktopSchemeChanged(styleScheme(s)) })
	})
}

// readDesktopScheme records the desktop's light / dark preference from
// ColorSchemeEnv or, with a display, the desktop itself; it reports
// whether the desktop is worth watching for changes.
func (a *Application) readDesktopScheme() bool {
	a.schemeRead = true
	if env := strings.TrimSpace(os.Getenv(ColorSchemeEnv)); env != "" {
		style.SetDesktopColorScheme(style.ParseColorScheme(env))
		return false
	}
	if a.headless {
		return false
	}
	if s, ok := platform.DesktopColorScheme(); ok {
		style.SetDesktopColorScheme(styleScheme(s))
	}
	return true
}

// DesktopColorScheme is the desktop's light / dark preference. An app that
// does not follow it asks once, on the first call.
func (a *Application) DesktopColorScheme() style.ColorScheme {
	if a != nil && a.schemeStop == nil && !a.schemeRead {
		a.readDesktopScheme()
	}
	return style.DesktopColorScheme()
}

// desktopSchemeChanged restyles every window when the desktop turns light
// or dark (on the UI goroutine).
func (a *Application) desktopSchemeChanged(s style.ColorScheme) {
	if style.DesktopColorScheme() == s {
		return
	}
	style.SetDesktopColorScheme(s)
	if a.schemeStop != nil {
		a.ReloadPreferredLook()
	}
}

// ApplyAppearance switches every window to ap (theme, corners, icons,
// motion, and following the desktop's light / dark preference) without
// saving it; SaveAppearance persists it for every app.
func (a *Application) ApplyAppearance(ap style.Appearance) {
	if a == nil {
		return
	}
	style.SetReduceMotion(ap.ReduceMotion)
	a.followDesktop(ap.FollowDesktop)
	if a.look == nil {
		a.SetLook(ap.Look())
		return
	}
	a.SetLook(style.WithAppearance(a.look, ap))
}
