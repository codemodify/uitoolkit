package style

import "testing"

// With no look.json, a program follows the desktop — so a first run on a
// dark desktop is dark.
//
// It used to open white: the default pack is Plastik, which is light and
// had no dark sibling, and FollowDesktop defaulted to off. A passphrase
// prompt that flares up white on a dark desktop does it at the moment it
// matters most.
func TestFirstRunFollowsADarkDesktop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer SetDesktopColorScheme(DesktopColorScheme())

	SetDesktopColorScheme(SchemeDark)
	a := LoadAppearance()
	if !a.FollowDesktop {
		t.Fatal("a fresh appearance does not follow the desktop")
	}
	dark := a.Effective()
	if ParseTheme(string(dark.Theme)) != ThemeDark {
		t.Fatalf("on a dark desktop the look is %q (%s)", dark.Name, dark.Theme)
	}
	if RelLuminance(dark.Look().Palette().Background) > 0.35 {
		t.Fatalf("%q paints a light window on a dark desktop", dark.Name)
	}

	// And back: the saved choice is still Plastik, so a desktop that
	// turns light again gets Plastik rather than staying dark.
	SetDesktopColorScheme(SchemeLight)
	light := LoadAppearance().Effective()
	if ParseTheme(string(light.Theme)) != ThemeLight {
		t.Fatalf("on a light desktop the look is %q (%s)", light.Name, light.Theme)
	}
	if light.Name != DefaultTheme() {
		t.Fatalf("light is %q, want the default pack %q", light.Name, DefaultTheme())
	}
}

// A saved look.json is a choice, and a choice is not overridden: a user
// who turned following off keeps it off on a dark desktop.
func TestASavedLookDoesNotFollowUnlessItSaysSo(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer SetDesktopColorScheme(DesktopColorScheme())
	SetDesktopColorScheme(SchemeDark)

	saved := Appearance{Name: DefaultTheme()}.Normalize()
	saved.FollowDesktop = false
	if err := SaveAppearance(saved); err != nil {
		t.Fatal(err)
	}
	got := LoadAppearance()
	if got.FollowDesktop {
		t.Fatal("a saved look.json was made to follow the desktop")
	}
	if eff := got.Effective(); eff.Name != DefaultTheme() {
		t.Fatalf("the saved pack was swapped for %q", eff.Name)
	}
}

// The default pack has a dark sibling in every build, which is what
// makes the first-run case work without a starter fallback — a chosen
// pack with no sibling is still left alone (TestSchemeVariantPairs).
func TestTheDefaultPackHasADarkSibling(t *testing.T) {
	def := DefaultTheme()
	night := SchemeVariant(def, SchemeDark)
	if night == def {
		t.Fatalf("the default pack %q has no dark sibling in this build", def)
	}
	p, ok := LoadTheme(night)
	if !ok {
		t.Fatalf("%q does not load", night)
	}
	if ParseTheme(string(p.Palette)) != ThemeDark {
		t.Fatalf("%q is not dark", night)
	}
	if back := SchemeVariant(night, SchemeLight); back != def {
		t.Fatalf("%q comes back to %q, want %q", night, back, def)
	}
}
