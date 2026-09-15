package style

import (
	"math"
	"testing"
)

// The published Material 3 baseline scheme (seed #6750A4, 2021 roles).
var mdBaseline = map[bool]map[string]string{
	false: {
		"primary": "#6750a4", "onPrimary": "#ffffff", "primaryContainer": "#eaddff", "onPrimaryContainer": "#21005d",
		"secondary": "#625b71", "onSecondary": "#ffffff", "secondaryContainer": "#e8def8", "onSecondaryContainer": "#1d192b",
		"tertiary": "#7d5260", "onTertiary": "#ffffff", "tertiaryContainer": "#ffd8e4", "onTertiaryContainer": "#31111d",
		"error": "#b3261e", "onError": "#ffffff", "errorContainer": "#f9dedc", "onErrorContainer": "#410e0b",
		"background": "#fffbfe", "onBackground": "#1c1b1f", "surface": "#fffbfe", "onSurface": "#1c1b1f",
		"surfaceVariant": "#e7e0ec", "onSurfaceVariant": "#49454f", "outline": "#79747e", "outlineVariant": "#cac4d0",
		"inverseSurface": "#313033", "inverseOnSurface": "#f4eff4", "inversePrimary": "#d0bcff",
	},
	true: {
		"primary": "#d0bcff", "onPrimary": "#381e72", "primaryContainer": "#4f378b", "onPrimaryContainer": "#eaddff",
		"secondary": "#ccc2dc", "onSecondary": "#332d41", "secondaryContainer": "#4a4458", "onSecondaryContainer": "#e8def8",
		"tertiary": "#efb8c8", "onTertiary": "#492532", "tertiaryContainer": "#633b48", "onTertiaryContainer": "#ffd8e4",
		"error": "#f2b8b5", "onError": "#601410", "errorContainer": "#8c1d18", "onErrorContainer": "#f9dedc",
		"background": "#1c1b1f", "onBackground": "#e6e1e5", "surface": "#1c1b1f", "onSurface": "#e6e1e5",
		"surfaceVariant": "#49454f", "onSurfaceVariant": "#cac4d0", "outline": "#938f99", "outlineVariant": "#49454f",
		"inverseSurface": "#e6e1e5", "inverseOnSurface": "#313033", "inversePrimary": "#6750a4",
	},
}

// mdLooseRoles are the roles the Lab approximation reproduces less closely.
var mdLooseRoles = map[string]bool{
	"onPrimaryContainer": true, "onPrimary": true, "primaryContainer": true,
	"error": true, "onError": true, "errorContainer": true, "onErrorContainer": true,
}

func mdSchemeRoles(s mdScheme) map[string]string {
	return map[string]string{
		"primary": colorHexPadded(s.primary), "onPrimary": colorHexPadded(s.onPrimary),
		"primaryContainer": colorHexPadded(s.primaryContainer), "onPrimaryContainer": colorHexPadded(s.onPrimaryContainer),
		"secondary": colorHexPadded(s.secondary), "onSecondary": colorHexPadded(s.onSecondary),
		"secondaryContainer": colorHexPadded(s.secondaryContainer), "onSecondaryContainer": colorHexPadded(s.onSecondaryContainer),
		"tertiary": colorHexPadded(s.tertiary), "onTertiary": colorHexPadded(s.onTertiary),
		"tertiaryContainer": colorHexPadded(s.tertiaryContainer), "onTertiaryContainer": colorHexPadded(s.onTertiaryContainer),
		"error": colorHexPadded(s.errorC), "onError": colorHexPadded(s.onError),
		"errorContainer": colorHexPadded(s.errorContainer), "onErrorContainer": colorHexPadded(s.onErrorContainer),
		"background": colorHexPadded(s.background), "onBackground": colorHexPadded(s.onBackground),
		"surface": colorHexPadded(s.surface), "onSurface": colorHexPadded(s.onSurface),
		"surfaceVariant": colorHexPadded(s.surfaceVariant), "onSurfaceVariant": colorHexPadded(s.onSurfaceVariant),
		"outline": colorHexPadded(s.outline), "outlineVariant": colorHexPadded(s.outlineVariant),
		"inverseSurface": colorHexPadded(s.inverseSurface), "inverseOnSurface": colorHexPadded(s.inverseOnSurface),
		"inversePrimary": colorHexPadded(s.inversePrimary),
	}
}

// The CIELAB approximation of HCT reproduces the baseline scheme from its
// seed: the roles controls are painted with within ΔE 6 of the published
// values, the rest (the darkest primary and the error tones, where CAM16
// and Lab chroma part ways) within ΔE 12, and every tone within 3 of L*.
func TestMaterialTonalPaletteMatchesBaseline(t *testing.T) {
	core := mdCorePalette(Hex("#6750a4"))
	for _, dark := range []bool{false, true} {
		got := mdSchemeRoles(mdSchemeFrom(core, dark))
		worst, worstRole := 0.0, ""
		for role, want := range mdBaseline[dark] {
			g := Hex(got[role])
			w := Hex(want)
			d := mdDeltaE(g, w)
			if d > worst {
				worst, worstRole = d, role
			}
			limit := 6.0
			if mdLooseRoles[role] {
				limit = 12
			}
			if d > limit {
				t.Errorf("dark=%v %s = %s, published %s (ΔE %.1f)", dark, role, got[role], want, d)
			}
			if dl := math.Abs(mdToLab(g).l - mdToLab(w).l); dl > 3 {
				t.Errorf("dark=%v %s tone off by %.1f", dark, role, dl)
			}
		}
		t.Logf("dark=%v: worst ΔE %.2f (%s)", dark, worst, worstRole)
	}
}

// A palette's tones are ordered by lightness, and every tone is in gamut.
func TestMaterialTonesAreMonotonic(t *testing.T) {
	for _, seed := range []string{"#6750a4", "#0061a4", "#006e1c", "#b3261e", "#795548", "#808080", "#ff00ff", "#00ffff"} {
		core := mdCorePalette(Hex(seed))
		for _, pal := range []mdPalette{core.primary, core.secondary, core.tertiary, core.neutral, core.variant, core.err} {
			prev := -1.0
			for tone := 0.0; tone <= 100; tone += 5 {
				c := pal.tone(tone)
				l := mdToLab(c).l
				if l < prev-0.01 {
					t.Fatalf("seed %s hue %.0f: tone %v has L* %.2f below the previous %.2f", seed, pal.h, tone, l, prev)
				}
				if math.Abs(l-tone) > 1 {
					t.Fatalf("seed %s hue %.0f: tone %v came out at L* %.2f", seed, pal.h, tone, l)
				}
				prev = l
			}
		}
	}
}
