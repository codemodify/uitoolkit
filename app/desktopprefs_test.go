package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func lookPack(t *testing.T, a *Application) string {
	t.Helper()
	c, ok := a.Look().(*style.Classic)
	if !ok {
		t.Fatalf("look %T", a.Look())
	}
	return c.Pack()
}

// A theme that follows the desktop starts in the desktop's scheme and
// switches with it; one that does not stays put.
func TestAppFollowsDesktopScheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(ColorSchemeEnv, "dark")
	defer style.SetDesktopColorScheme(style.SchemeNoPreference)
	if err := style.SaveAppearance(style.Appearance{Name: "breeze", Theme: style.ThemeLight, FollowDesktop: true}); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	if got := lookPack(t, a); got != "breeze-night" {
		t.Fatalf("start: %s, want breeze-night", got)
	}
	w, err := a.NewWindow(platform.WindowOptions{Title: "follow", Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("x"))
	hooks := 0
	remove := a.OnLookChange(func() { hooks++ })
	// The desktop turns light (as the portal's SettingChanged would say).
	a.schemeForced = false
	a.desktopPrefsChanged(platform.DesktopPrefs{ColorScheme: platform.SchemeLight})
	if got := lookPack(t, a); got != "breeze" {
		t.Fatalf("after light: %s, want breeze", got)
	}
	if c, ok := w.Look().(*style.Classic); !ok || c.Pack() != "breeze" {
		t.Fatalf("window look %v", w.Look())
	}
	if hooks != 1 {
		t.Fatalf("OnLookChange ran %d times", hooks)
	}
	// The same scheme again is not a change.
	a.desktopPrefsChanged(platform.DesktopPrefs{ColorScheme: platform.SchemeLight})
	if hooks != 1 {
		t.Fatalf("repeat scheme restyled (%d)", hooks)
	}
	remove()
	a.ApplyAppearance(style.Appearance{Name: "win95", Theme: style.ThemeLight})
	if hooks != 1 {
		t.Fatal("removed hook ran")
	}
	if got := lookPack(t, a); got != "win95" {
		t.Fatalf("apply: %s", got)
	}
	// Not following: the desktop turning dark changes nothing.
	a.desktopPrefsChanged(platform.DesktopPrefs{ColorScheme: platform.SchemeDark})
	if got := lookPack(t, a); got != "win95" {
		t.Fatalf("not following, dark desktop: %s", got)
	}
}

// The desktop's reduced-motion setting stops animations in every app,
// whatever its theme.
func TestDesktopReducedMotion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "")
	defer style.SetDesktopReduceMotion(false)
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	if !style.Animations() {
		t.Fatal("animations should start on")
	}
	a.desktopPrefsChanged(platform.DesktopPrefs{ReducedMotion: true})
	if style.Animations() {
		t.Fatal("the desktop asked for reduced motion")
	}
	a.desktopPrefsChanged(platform.DesktopPrefs{})
	if !style.Animations() {
		t.Fatal("reduced motion lifted")
	}
}

func TestAppNotFollowingIgnoresScheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(ColorSchemeEnv, "dark")
	defer style.SetDesktopColorScheme(style.SchemeNoPreference)
	if err := style.SaveAppearance(style.Appearance{Name: "breeze", Theme: style.ThemeLight}); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	if got := lookPack(t, a); got != "breeze" {
		t.Fatalf("%s, want breeze", got)
	}
}
