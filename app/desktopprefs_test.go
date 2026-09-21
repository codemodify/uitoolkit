package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
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

// UITK_ACCENT stands in for the desktop's accent; the portal's accent
// arriving later restyles an app that follows the desktop.
func TestAppTakesDesktopAccent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(AccentEnv, "#e95420")
	defer style.SetDesktopAccent(paintengine2d.Color{}, false)
	if err := style.SaveAppearance(style.Appearance{Name: "breeze", Theme: style.ThemeLight, FollowDesktop: true}); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	if got := a.Look().Palette().Selection; got != style.Hex("#e95420") {
		t.Fatalf("selection %v, want UITK_ACCENT", got)
	}
	// A portal accent (as SettingChanged would bring it).
	a.accentForced = false
	a.desktopPrefsChanged(platform.DesktopPrefs{HasAccent: true, Accent: [3]float64{0, 0.5, 0}})
	if got := a.Look().Palette().Selection; got != paintengine2d.RGB(0, 0.5, 0) {
		t.Fatalf("selection %v after the desktop's accent changed", got)
	}
}

// Switching pack the way an app does — read the appearance back, change
// the pack, apply it — keeps every preference that belongs to the
// application rather than to the look. style.LookAppearance alone brings
// them back at their defaults, which is how a theme switch used to reset
// the frame, the caption buttons, motion and the file dialogs.
func TestAppearanceRoundTripsTheApplicationsPreferences(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer style.SetReduceMotion(false)
	defer style.SetNativeDialogs(false)
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})

	ap := a.Appearance()
	ap.Decorations = style.DecorationsSystem
	ap.CaptionButtons = style.CaptionButtonsTheme
	ap.ReduceMotion = true
	ap.NativeDialogs = true
	a.ApplyAppearance(ap)
	if a.Decorations() != style.DecorationsSystem {
		t.Fatalf("Decorations() = %q after applying the desktop's frame", a.Decorations())
	}

	next := a.Appearance()
	next.Name = "win95"
	a.ApplyAppearance(next)

	got := a.Appearance()
	if got.Name != "win95" {
		t.Errorf("the pack is %q, want win95", got.Name)
	}
	if got.Decorations != style.DecorationsSystem || a.Decorations() != style.DecorationsSystem {
		t.Errorf("changing pack reset the frame preference to %q", got.Decorations)
	}
	if got.CaptionButtons != style.CaptionButtonsTheme || a.CaptionButtons() != style.CaptionButtonsTheme {
		t.Errorf("changing pack reset the caption buttons to %q", got.CaptionButtons)
	}
	if !got.ReduceMotion || !style.ReduceMotion() {
		t.Error("changing pack turned reduced motion off")
	}
	if !got.NativeDialogs || !style.NativeDialogs() {
		t.Error("changing pack turned the desktop's file dialogs off")
	}
}
