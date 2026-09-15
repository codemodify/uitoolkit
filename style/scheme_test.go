package style

import (
	"strings"
	"testing"
)

func TestSchemeVariantPairs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range []struct {
		name   string
		scheme ColorScheme
		want   string
	}{
		{"breeze", SchemeDark, "breeze-night"},
		{"breeze-night", SchemeLight, "breeze"},
		{"breeze", SchemeLight, "breeze"},
		{"breeze-night", SchemeDark, "breeze-night"},
		{"win95", SchemeDark, "win95-dark"},
		{"win95-dark", SchemeLight, "win95"},
		{"win98", SchemeDark, "win95-dark"},
		{"luna-olive", SchemeDark, "luna-night"},
		{"luna-night", SchemeLight, "luna"},
		{"aqua-graphite", SchemeDark, "aqua-night"},
		{"adwaita", SchemeDark, "adwaita-night"},
		{"wmaker-steelbluesilk", SchemeDark, "wmaker-night"},
		{"wmaker-night", SchemeLight, "wmaker-default"},
		{"cde-desert", SchemeDark, "cde-charcoal"},
		{"cde-charcoal", SchemeLight, "cde"},
		{"light", SchemeDark, "dark"},
		{"dark", SchemeLight, "light"},
		{"luna-dark", SchemeDark, "luna-night"}, // a legacy alias
		// No sibling: the pack stays as chosen.
		{"os2warp", SchemeDark, "os2warp"},
		{"amiga13", SchemeLight, "amiga13"},
		{"win-highcontrast", SchemeLight, "win-highcontrast"},
		{"breeze", SchemeNoPreference, "breeze"},
		{"no-such-pack", SchemeDark, "no-such-pack"},
	} {
		if got := SchemeVariant(c.name, c.scheme); got != c.want {
			t.Errorf("SchemeVariant(%q, %v) = %q, want %q", c.name, c.scheme, got, c.want)
		}
	}
}

// Every sibling is in the scheme asked for, and every -night pack comes
// back to its light pack.
func TestSchemeVariantEveryPack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, p := range ListBuiltinThemes() {
		for _, s := range []ColorScheme{SchemeDark, SchemeLight} {
			got := SchemeVariant(p.Name, s)
			if got == p.Name {
				continue
			}
			q, ok := LoadTheme(got)
			if !ok {
				t.Fatalf("%s → %s: not a pack", p.Name, got)
			}
			if packIsDark(q) != (s == SchemeDark) {
				t.Errorf("%s → %s is not %v", p.Name, got, s)
			}
		}
		if base, ok := strings.CutSuffix(p.Name, "-night"); ok {
			if _, ok := LoadTheme(base); ok && SchemeVariant(p.Name, SchemeLight) != base {
				t.Errorf("%s in light = %s, want %s", p.Name, SchemeVariant(p.Name, SchemeLight), base)
			}
			if _, ok := LoadTheme(base); ok && SchemeVariant(base, SchemeDark) != p.Name {
				t.Errorf("%s in dark = %s, want %s", base, SchemeVariant(base, SchemeDark), p.Name)
			}
		}
	}
}

func TestEffectiveFollowsDesktop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer SetDesktopColorScheme(SchemeNoPreference)
	a := Appearance{Name: "breeze", Theme: ThemeLight}
	SetDesktopColorScheme(SchemeDark)
	if got := a.Effective().Name; got != "breeze" {
		t.Fatalf("not following: %s", got)
	}
	a.FollowDesktop = true
	eff := a.Effective()
	if eff.Name != "breeze-night" || eff.Theme != ThemeDark {
		t.Fatalf("following dark: %s %s", eff.Name, eff.Theme)
	}
	if got := a.Look().Pack(); got != "breeze-night" {
		t.Fatalf("look pack %s", got)
	}
	SetDesktopColorScheme(SchemeLight)
	if got := a.Look().Pack(); got != "breeze" {
		t.Fatalf("light desktop: look pack %s", got)
	}
	// WithAppearance (a running app reloading look.json) follows too.
	SetDesktopColorScheme(SchemeDark)
	if got := WithAppearance(DarkLook(), a).(*Classic).Pack(); got != "breeze-night" {
		t.Fatalf("WithAppearance pack %s", got)
	}
}

func TestFollowDesktopPref(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := Appearance{Name: "adwaita", Theme: ThemeLight, FollowDesktop: true}
	if err := SaveAppearance(a); err != nil {
		t.Fatal(err)
	}
	got := LoadAppearance()
	if !got.FollowDesktop || got.Name != "adwaita" {
		t.Fatalf("round trip %+v", got)
	}
	// UITK_THEME asks for one pack: it is shown as it is.
	t.Setenv(ThemeEnv, "win95")
	if got := LoadAppearance(); got.FollowDesktop || got.Name != "win95" {
		t.Fatalf("env override %+v", got)
	}
}

func TestParseColorScheme(t *testing.T) {
	for in, want := range map[string]ColorScheme{"dark": SchemeDark, "Light": SchemeLight, "prefer-dark": SchemeDark, "": SchemeNoPreference, "blue": SchemeNoPreference} {
		if got := ParseColorScheme(in); got != want {
			t.Errorf("ParseColorScheme(%q) = %v", in, got)
		}
	}
}
