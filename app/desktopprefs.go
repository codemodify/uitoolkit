package app

import (
	"os"
	"strings"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// ColorSchemeEnv overrides the desktop's light / dark preference for the
// process (UITK_COLOR_SCHEME=dark ./app), to try a theme's dark sibling
// without switching the desktop. It matters only while the appearance
// follows the desktop.
const ColorSchemeEnv = "UITK_COLOR_SCHEME"

// AccentEnv overrides the desktop's accent colour for the process
// (UITK_ACCENT=#e95420 ./app). Like the desktop's, it recolours only
// appearances that follow the desktop, and only in themes whose engine
// takes an accent.
const AccentEnv = "UITK_ACCENT"

// desktopReadTimeout bounds how long New waits for the desktop portal; a
// slower answer arrives later and restyles the app.
const desktopReadTimeout = 300 * time.Millisecond

func accentOf(p platform.DesktopPrefs) (paintengine2d.Color, bool) {
	if !p.HasAccent {
		return paintengine2d.Color{}, false
	}
	return paintengine2d.RGB(float32(p.Accent[0]), float32(p.Accent[1]), float32(p.Accent[2])), true
}

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
	if env := strings.TrimSpace(os.Getenv(AccentEnv)); env != "" {
		c, ok := style.ParseHexColor(env)
		style.SetDesktopAccent(c, ok)
		a.accentForced = true
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
			a.desktop = r.prefs
			style.SetDesktopReduceMotion(r.prefs.ReducedMotion)
			if !a.schemeForced {
				style.SetDesktopColorScheme(styleScheme(r.prefs.ColorScheme))
			}
			if !a.accentForced {
				style.SetDesktopAccent(accentOf(r.prefs))
			}
		}
	}
}

// desktopPrefsChanged applies a change of the desktop's preferences on the
// UI goroutine.
func (a *Application) desktopPrefsChanged(p platform.DesktopPrefs) {
	was := a.desktop
	a.desktop = p
	if was.ButtonLayout != p.ButtonLayout || was.TitlebarDoubleClick != p.TitlebarDoubleClick ||
		was.TitlebarMiddleClick != p.TitlebarMiddleClick || was.TitlebarRightClick != p.TitlebarRightClick ||
		was.DoubleClickTime != p.DoubleClickTime || was.DragThreshold != p.DragThreshold {
		a.reloadTitleBarPrefs()
	}
	style.SetDesktopReduceMotion(p.ReducedMotion)
	changed := false
	if s := styleScheme(p.ColorScheme); !a.schemeForced && style.DesktopColorScheme() != s {
		style.SetDesktopColorScheme(s)
		changed = true
	}
	if !a.accentForced {
		c, ok := accentOf(p)
		if was, wasOK := style.DesktopAccent(); ok != wasOK || (ok && was != c) {
			style.SetDesktopAccent(c, ok)
			changed = true
		}
	}
	if changed && a.following {
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
	style.SetNativeDialogs(ap.NativeDialogs)
	if !a.headless {
		a.setDecorationsPref(ap.Decorations)
	}
	a.following = ap.FollowDesktop
	if a.look == nil {
		a.SetLook(ap.Look())
		return
	}
	a.SetLook(style.WithAppearance(a.look, ap))
}
