package style

import "testing"

func TestThemeOverrideOverIsNearestWins(t *testing.T) {
	near := ThemeOverride{Pack: "aqua", Corners: CornersSquare}
	far := ThemeOverride{Pack: "luna", Icons: IconSetLucide, FontUI: "Cantarell"}
	got := near.Over(far)
	if got.Pack != "aqua" || got.Corners != CornersSquare {
		t.Fatalf("the nearer level must win: %+v", got)
	}
	if got.Icons != IconSetLucide || got.FontUI != "Cantarell" {
		t.Fatalf("what the nearer level does not state must be inherited: %+v", got)
	}
	if (ThemeOverride{}).Over(far) != far {
		t.Fatal("an empty level is the level above it")
	}
	if !(ThemeOverride{}).Empty() || (ThemeOverride{IconSize: IconSizeSmall}).Empty() {
		t.Fatal("Empty")
	}
}

func TestThemeOverrideOnAppearance(t *testing.T) {
	base := Appearance{
		Name: "win95", Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp,
		IconSize: IconSizeLarge, FontUI: "Cantarell", FollowDesktop: true,
		Decorations: DecorationsSystem, ReduceMotion: true,
	}.Normalize()

	got := ThemeOverride{Pack: "aqua"}.On(base)
	if got.Name != "aqua" {
		t.Fatalf("pack: %s", got.Name)
	}
	if got.Corners != CornersSquare || got.Icons != IconSetSharp || got.IconSize != IconSizeLarge {
		t.Fatalf("a pack-only level must inherit the rest: %+v", got)
	}
	if got.FontUI != "Cantarell" {
		t.Fatalf("typeface: %q", got.FontUI)
	}
	if got.FollowDesktop {
		t.Fatal("naming a pack stops the light / dark substitution")
	}
	// The preferences that never cascade pass through untouched, so On
	// can be applied to a whole application appearance.
	if got.Decorations != DecorationsSystem || !got.ReduceMotion {
		t.Fatalf("a level must not touch what does not cascade: %+v", got)
	}

	// "theme" is how a level takes the pack's own face back from one a
	// level above pinned.
	if back := (ThemeOverride{FontUI: "theme"}).On(base); back.FontUI != "" {
		t.Fatalf(`FontUI "theme" should mean the pack's own: %q`, back.FontUI)
	}
}

func TestThemedKeepsScaleAndDensity(t *testing.T) {
	pack, ok := LoadTheme("luna")
	if !ok {
		t.Fatal("luna")
	}
	base := WithDensity(WithScale(pack.Look(), 2), DensityCompact).(*Classic)
	got, ok := Themed(base, ThemeOverride{Pack: "aqua"}).(*Classic)
	if !ok {
		t.Fatalf("%T", got)
	}
	if got.Pack() != "aqua" {
		t.Fatalf("pack: %s", got.Pack())
	}
	if got.Scale() != base.Scale() {
		t.Fatalf("scale %g, want %g — a level of the cascade may not change it", got.Scale(), base.Scale())
	}
	if got.Density() != DensityCompact {
		t.Fatalf("density: %v", got.Density())
	}
}

// The derived look is memoized on (base, level): the same pair gives the
// same pointer, which is what lets everything keyed on a look be shared.
func TestThemedIsMemoized(t *testing.T) {
	base := DarkLook()
	over := ThemeOverride{Pack: "aqua"}
	first := Themed(base, over)
	if second := Themed(base, over); second != first {
		t.Fatal("the same base and the same level must give the same look")
	}
	if other := Themed(base, ThemeOverride{Pack: "luna"}); other == first {
		t.Fatal("a different level must give a different look")
	}
	if same := Themed(base, ThemeOverride{}); same != base {
		t.Fatal("an empty level is no level")
	}
}

// A look built through an Appearance remembers the parts it cannot be
// read back for, so a level of the cascade does not silently drop them.
func TestLookAppearanceKeepsPinnedFontsAndFollowing(t *testing.T) {
	ap := Appearance{Name: "luna", FontUI: "Cantarell", FontMono: "Hack", FollowDesktop: true}
	got := LookAppearance(ap.Look())
	if got.FontUI != "Cantarell" || got.FontMono != "Hack" {
		t.Fatalf("pinned typefaces: %q %q", got.FontUI, got.FontMono)
	}
	if !got.FollowDesktop {
		t.Fatal("following the desktop")
	}
	// And every derivation carries them.
	for what, lk := range map[string]LookAndFeel{
		"WithScale":    WithScale(ap.Look(), 2),
		"WithCorners":  WithCorners(ap.Look(), CornersSquare),
		"WithIcons":    WithIcons(ap.Look(), IconSetLucide),
		"WithIconSize": WithIconSize(ap.Look(), IconSizeLarge),
		"WithDensity":  WithDensity(ap.Look(), DensityCompact),
		"Themed":       Themed(ap.Look(), ThemeOverride{Corners: CornersSquare}),
	} {
		if a := LookAppearance(lk); a.FontUI != "Cantarell" || a.FontMono != "Hack" {
			t.Fatalf("%s dropped the pinned typefaces: %q %q", what, a.FontUI, a.FontMono)
		}
	}
	// A look nobody pinned anything in says so, rather than guessing.
	if a := LookAppearance(NewClassic("dark", Dark(), DefaultMetrics())); a.FontUI != "" || a.FontMono != "" {
		t.Fatalf("a hand-built look pinned nothing: %q %q", a.FontUI, a.FontMono)
	}
}
