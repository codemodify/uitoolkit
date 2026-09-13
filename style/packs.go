package style

import "sync"

// Era names for Settings grouping (oldest → newest).
const (
	EraClassic95 = "Classic 95"
	EraMotif     = "Motif / CDE"
	EraNext      = "NeXT"
	EraLuna      = "Luna"
	EraAqua      = "Aqua"
	EraFusion    = "Fusion"
	EraBreeze    = "Breeze"
	EraFluent    = "Fluent"
	EraMaterial  = "Material"
	EraFlatLaf   = "FlatLaf"
)

var (
	eraPacksOnce sync.Once
	eraPacksBy   map[string]ThemePack
	eraPackOrder []string
)

func eraPackIndex() map[string]ThemePack {
	eraPacksOnce.Do(func() {
		eraPacksBy = map[string]ThemePack{}
		for _, p := range allEraPacks() {
			p.Source = ThemeSourceBuiltin
			p.Tokens = p.Tokens.Resolve()
			p.Palette = p.Tokens.Family
			eraPacksBy[p.Name] = p
			eraPackOrder = append(eraPackOrder, p.Name)
		}
	})
	return eraPacksBy
}

func builtinEraOrder() []string {
	eraPackIndex()
	return append([]string(nil), eraPackOrder...)
}

func builtinEraPack(name string) (ThemePack, bool) {
	p, ok := eraPackIndex()[name]
	return p, ok
}

// AllBuiltinThemeNames is the embedded era-pack id list (stable order).
func AllBuiltinThemeNames() []string {
	return builtinEraOrder()
}

func allEraPacks() []ThemePack {
	return []ThemePack{
		packClassic95Dark(),
		packClassic95Light(),
		packMotif(),
		packCDE(),
		packCDECrimson(),
		packNext(),
		packNextNight(),
		packLuna(),
		packLunaNight(),
		packAqua(),
		packAquaNight(),
		packFusion(),
		packFusionNight(),
		packBreeze(),
		packBreezeNight(),
		packFluent(),
		packFluentNight(),
		packMaterial(),
		packMaterialNight(),
		packFlatLaf(),
		packFlatLafNight(),
	}
}

func eraPack(name, label, era string, family ThemeName, bevel BevelStyle, tok ThemeTokens) ThemePack {
	tok.Family = family
	tok.Bevel = bevel
	tok.Era = era
	return ThemePack{
		Name:    name,
		Label:   label,
		Source:  ThemeSourceBuiltin,
		Palette: family,
		Era:     era,
		Tokens:  tok,
	}
}

func packClassic95Dark() ThemePack {
	return eraPack("dark", "Classic 95 Dark", EraClassic95, ThemeDark, BevelClassic3D, ThemeTokens{
		Metrics:  ChromeMetrics{BevelDepth: 1, Scroll: 16, Elevation: 0},
		Palette:  Dark(),
		Hot:      ChromeState{Fill: hexColor("#000080"), Border: hexColor("#000040")},
		Pressed:  ChromeState{Fill: hexColor("#000060"), Border: hexColor("#000040")},
		Selected: ChromeState{Fill: hexColor("#000080"), Border: hexColor("#000040")},
	})
}

func packClassic95Light() ThemePack {
	return eraPack("light", "Classic 95 Light", EraClassic95, ThemeLight, BevelClassic3D, ThemeTokens{
		Metrics:  ChromeMetrics{BevelDepth: 1, Scroll: 16, Elevation: 0},
		Palette:  Light(),
		Hot:      ChromeState{Fill: hexColor("#000080"), Border: hexColor("#000040")},
		Pressed:  ChromeState{Fill: hexColor("#000060"), Border: hexColor("#000040")},
		Selected: ChromeState{Fill: hexColor("#000080"), Border: hexColor("#000040")},
	})
}

func packMotif() ThemePack {
	return eraPack("motif", "Motif", EraMotif, ThemeLight, BevelClassic3D, ThemeTokens{
		Metrics: ChromeMetrics{BevelDepth: 2, Scroll: 18, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#aeb2c3"),
			Surface:         hexColor("#aeb2c3"),
			SurfaceAlt:      hexColor("#9aa0b4"),
			Overlay:         hexAlpha("#202028", 0.40),
			Border:          hexColor("#555566"),
			Divider:         hexColor("#7a8090"),
			Text:            hexColor("#000000"),
			TextMuted:       hexColor("#404050"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#000082"),
			AccentHover:     hexColor("#1a1a9a"),
			AccentPress:     hexColor("#000060"),
			Danger:          hexColor("#a02020"),
			Success:         hexColor("#206020"),
			Warning:         hexColor("#8a6a10"),
			Track:           hexColor("#9aa0b4"),
			Thumb:           hexColor("#aeb2c3"),
			Field:           hexColor("#d6d6de"),
			FieldBorder:     hexColor("#404050"),
			Focus:           hexColor("#000000"),
			Selection:       hexAlpha("#000082", 0.90),
			Shadow:          hexAlpha("#000000", 0.40),
			Highlight:       hexAlpha("#ffffff", 0.70),
			MenuHover:       hexColor("#000082"),
			MenuHoverBorder: hexColor("#000050"),
			MenuGutter:      hexColor("#9aa0b4"),
			BevelLight:      hexColor("#e8e8f0"),
			BevelDark:       hexColor("#404050"),
		},
		Hot:      ChromeState{Fill: hexColor("#000082"), Border: hexColor("#000050")},
		Pressed:  ChromeState{Fill: hexColor("#000060"), Border: hexColor("#000050")},
		Selected: ChromeState{Fill: hexColor("#000082"), Border: hexColor("#000050")},
	})
}

func packCDE() ThemePack {
	return eraPack("cde", "CDE Charcoal", EraMotif, ThemeLight, BevelClassic3D, ThemeTokens{
		Metrics: ChromeMetrics{BevelDepth: 2, Scroll: 18, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#8a8580"),
			Surface:         hexColor("#9a9590"),
			SurfaceAlt:      hexColor("#8a8580"),
			Overlay:         hexAlpha("#201810", 0.40),
			Border:          hexColor("#4a4540"),
			Divider:         hexColor("#6a6560"),
			Text:            hexColor("#000000"),
			TextMuted:       hexColor("#3a3530"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#5a9a9a"),
			AccentHover:     hexColor("#6aaeae"),
			AccentPress:     hexColor("#3a7878"),
			Danger:          hexColor("#a04040"),
			Success:         hexColor("#3a7848"),
			Warning:         hexColor("#b09030"),
			Track:           hexColor("#8a8580"),
			Thumb:           hexColor("#9a9590"),
			Field:           hexColor("#c4bfb8"),
			FieldBorder:     hexColor("#4a4540"),
			Focus:           hexColor("#000000"),
			Selection:       hexAlpha("#5a9a9a", 0.88),
			Shadow:          hexAlpha("#000000", 0.40),
			Highlight:       hexAlpha("#ffffff", 0.55),
			MenuHover:       hexColor("#5a9a9a"),
			MenuHoverBorder: hexColor("#2a5a5a"),
			MenuGutter:      hexColor("#7a7570"),
			BevelLight:      hexColor("#d8d4ce"),
			BevelDark:       hexColor("#3a3530"),
		},
		Hot:      ChromeState{Fill: hexColor("#5a9a9a"), Border: hexColor("#2a5a5a")},
		Pressed:  ChromeState{Fill: hexColor("#3a7878"), Border: hexColor("#2a5a5a")},
		Selected: ChromeState{Fill: hexColor("#5a9a9a"), Border: hexColor("#2a5a5a")},
	})
}

func packCDECrimson() ThemePack {
	return eraPack("cde-crimson", "CDE Crimson", EraMotif, ThemeLight, BevelClassic3D, ThemeTokens{
		Metrics: ChromeMetrics{BevelDepth: 2, Scroll: 18, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#9a7070"),
			Surface:         hexColor("#b09090"),
			SurfaceAlt:      hexColor("#9a7070"),
			Overlay:         hexAlpha("#301010", 0.42),
			Border:          hexColor("#5a3030"),
			Divider:         hexColor("#7a4848"),
			Text:            hexColor("#000000"),
			TextMuted:       hexColor("#402020"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#a03030"),
			AccentHover:     hexColor("#c04040"),
			AccentPress:     hexColor("#701818"),
			Danger:          hexColor("#c02020"),
			Success:         hexColor("#3a7848"),
			Warning:         hexColor("#c0a030"),
			Track:           hexColor("#9a7070"),
			Thumb:           hexColor("#b09090"),
			Field:           hexColor("#e0c8c8"),
			FieldBorder:     hexColor("#5a3030"),
			Focus:           hexColor("#000000"),
			Selection:       hexAlpha("#a03030", 0.88),
			Shadow:          hexAlpha("#000000", 0.40),
			Highlight:       hexAlpha("#ffffff", 0.50),
			MenuHover:       hexColor("#a03030"),
			MenuHoverBorder: hexColor("#501010"),
			MenuGutter:      hexColor("#8a6060"),
			BevelLight:      hexColor("#e8d0d0"),
			BevelDark:       hexColor("#401818"),
		},
		Hot:      ChromeState{Fill: hexColor("#a03030"), Border: hexColor("#501010")},
		Pressed:  ChromeState{Fill: hexColor("#701818"), Border: hexColor("#501010")},
		Selected: ChromeState{Fill: hexColor("#a03030"), Border: hexColor("#501010")},
	})
}

func packNext() ThemePack {
	return eraPack("next", "NeXT", EraNext, ThemeDark, BevelClassic3D, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 0, RadiusSmall: 0, BevelDepth: 1, Scroll: 14, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#4a4a4a"),
			Surface:         hexColor("#4a4a4a"),
			SurfaceAlt:      hexColor("#3a3a3a"),
			Overlay:         hexAlpha("#000000", 0.50),
			Border:          hexColor("#000000"),
			Divider:         hexColor("#2a2a2a"),
			Text:            hexColor("#ffffff"),
			TextMuted:       hexColor("#c0c0c0"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#000000"),
			AccentHover:     hexColor("#1a1a1a"),
			AccentPress:     hexColor("#000000"),
			Danger:          hexColor("#c04040"),
			Success:         hexColor("#40a060"),
			Warning:         hexColor("#d0b040"),
			Track:           hexColor("#3a3a3a"),
			Thumb:           hexColor("#6a6a6a"),
			Field:           hexColor("#ffffff"),
			FieldBorder:     hexColor("#000000"),
			Focus:           hexColor("#000000"),
			Selection:       hexColor("#000000"),
			Shadow:          hexAlpha("#000000", 0.55),
			Highlight:       hexAlpha("#ffffff", 0.35),
			MenuHover:       hexColor("#000000"),
			MenuHoverBorder: hexColor("#ffffff"),
			MenuGutter:      hexColor("#3a3a3a"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#000000"),
		},
		Hot:      ChromeState{Fill: hexColor("#000000"), Border: hexColor("#ffffff")},
		Pressed:  ChromeState{Fill: hexColor("#1a1a1a"), Border: hexColor("#000000")},
		Selected: ChromeState{Fill: hexColor("#000000"), Border: hexColor("#ffffff")},
	})
}

func packNextNight() ThemePack {
	return eraPack("next-night", "NeXT Night", EraNext, ThemeDark, BevelClassic3D, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 0, RadiusSmall: 0, BevelDepth: 1, Scroll: 14, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#2a2a2a"),
			Surface:         hexColor("#323232"),
			SurfaceAlt:      hexColor("#262626"),
			Overlay:         hexAlpha("#000000", 0.55),
			Border:          hexColor("#000000"),
			Divider:         hexColor("#1a1a1a"),
			Text:            hexColor("#f0f0f0"),
			TextMuted:       hexColor("#a0a0a0"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#d0d0d0"),
			AccentHover:     hexColor("#ffffff"),
			AccentPress:     hexColor("#a0a0a0"),
			Danger:          hexColor("#c04040"),
			Success:         hexColor("#40a060"),
			Warning:         hexColor("#d0b040"),
			Track:           hexColor("#1e1e1e"),
			Thumb:           hexColor("#5a5a5a"),
			Field:           hexColor("#1a1a1a"),
			FieldBorder:     hexColor("#000000"),
			Focus:           hexColor("#ffffff"),
			Selection:       hexColor("#000000"),
			Shadow:          hexAlpha("#000000", 0.60),
			Highlight:       hexAlpha("#ffffff", 0.18),
			MenuHover:       hexColor("#000000"),
			MenuHoverBorder: hexColor("#c0c0c0"),
			MenuGutter:      hexColor("#222222"),
			BevelLight:      hexColor("#8a8a8a"),
			BevelDark:       hexColor("#000000"),
		},
		Hot:      ChromeState{Fill: hexColor("#000000"), Border: hexColor("#c0c0c0")},
		Pressed:  ChromeState{Fill: hexColor("#101010"), Border: hexColor("#000000")},
		Selected: ChromeState{Fill: hexColor("#000000"), Border: hexColor("#c0c0c0")},
	})
}

func packLuna() ThemePack {
	return eraPack("luna", "Luna", EraLuna, ThemeLight, BevelLunaHottrack, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 2, RadiusSmall: 1, BevelDepth: 1, Scroll: 16, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#ece9d8"),
			Surface:         hexColor("#ece9d8"),
			SurfaceAlt:      hexColor("#ece9d8"),
			Overlay:         hexAlpha("#2a2820", 0.35),
			Border:          hexColor("#aca899"),
			Divider:         hexColor("#d6d2c2"),
			Text:            hexColor("#000000"),
			TextMuted:       hexColor("#5a5848"),
			TextOnAccent:    hexColor("#000000"),
			Accent:          hexColor("#316ac5"),
			AccentHover:     hexColor("#4b82d6"),
			AccentPress:     hexColor("#1e4a9a"),
			Danger:          hexColor("#c02828"),
			Success:         hexColor("#2a7a40"),
			Warning:         hexColor("#c09020"),
			Track:           hexColor("#d6d2c2"),
			Thumb:           hexColor("#c1d2ee"),
			Field:           hexColor("#ffffff"),
			FieldBorder:     hexColor("#7a7870"),
			Focus:           hexColor("#316ac5"),
			Selection:       hexAlpha("#c1d2ee", 0.95),
			Shadow:          hexAlpha("#000000", 0.22),
			Highlight:       hexAlpha("#ffffff", 0.55),
			MenuHover:       hexColor("#c1d2ee"),
			MenuHoverBorder: hexColor("#316ac5"),
			MenuGutter:      hexColor("#d6d2c2"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#aca899"),
		},
		Hot:      ChromeState{Fill: hexColor("#c1d2ee"), Border: hexColor("#316ac5")},
		Pressed:  ChromeState{Fill: hexColor("#a7c0e9"), Border: hexColor("#316ac5")},
		Selected: ChromeState{Fill: hexColor("#c1d2ee"), Border: hexColor("#316ac5")},
	})
}

func packLunaNight() ThemePack {
	return eraPack("luna-night", "Luna Night", EraLuna, ThemeDark, BevelLunaHottrack, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 2, RadiusSmall: 1, BevelDepth: 1, Scroll: 16, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#2b2d30"),
			Surface:         hexColor("#32343a"),
			SurfaceAlt:      hexColor("#2b2d30"),
			Overlay:         hexAlpha("#000000", 0.50),
			Border:          hexColor("#1a1c20"),
			Divider:         hexColor("#3a3c42"),
			Text:            hexColor("#e8e8ea"),
			TextMuted:       hexColor("#9aa0a8"),
			TextOnAccent:    hexColor("#e8e8ea"),
			Accent:          hexColor("#7aa2e3"),
			AccentHover:     hexColor("#8eb4f0"),
			AccentPress:     hexColor("#4a78c0"),
			Danger:          hexColor("#d05050"),
			Success:         hexColor("#4aaa68"),
			Warning:         hexColor("#d0b040"),
			Track:           hexColor("#232428"),
			Thumb:           hexColor("#3d5a80"),
			Field:           hexColor("#1e2024"),
			FieldBorder:     hexColor("#4a4e56"),
			Focus:           hexColor("#7aa2e3"),
			Selection:       hexAlpha("#3d5a80", 0.92),
			Shadow:          hexAlpha("#000000", 0.45),
			Highlight:       hexAlpha("#ffffff", 0.08),
			MenuHover:       hexColor("#3d5a80"),
			MenuHoverBorder: hexColor("#7aa2e3"),
			MenuGutter:      hexColor("#262830"),
			BevelLight:      hexColor("#5a5e66"),
			BevelDark:       hexColor("#101214"),
		},
		Hot:      ChromeState{Fill: hexColor("#3d5a80"), Border: hexColor("#7aa2e3")},
		Pressed:  ChromeState{Fill: hexColor("#2e4666"), Border: hexColor("#7aa2e3")},
		Selected: ChromeState{Fill: hexColor("#3d5a80"), Border: hexColor("#7aa2e3")},
	})
}

func packAqua() ThemePack {
	return eraPack("aqua", "Aqua", EraAqua, ThemeLight, BevelSoftShadow, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 8, RadiusSmall: 6, BevelDepth: 0, Scroll: 11, Elevation: 1},
		Palette: Palette{
			Background:      hexColor("#ececec"),
			Surface:         hexColor("#f6f6f6"),
			SurfaceAlt:      hexColor("#e8e8ea"),
			Overlay:         hexAlpha("#202028", 0.32),
			Border:          hexColor("#c0c0c6"),
			Divider:         hexColor("#d4d4d8"),
			Text:            hexColor("#1a1a1e"),
			TextMuted:       hexColor("#5a5a62"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#4c8bf5"),
			AccentHover:     hexColor("#6aa0f8"),
			AccentPress:     hexColor("#2f6ed4"),
			Danger:          hexColor("#e24b4b"),
			Success:         hexColor("#2fa05a"),
			Warning:         hexColor("#e0a020"),
			Track:           hexColor("#d8d8dc"),
			Thumb:           hexColor("#b0b0b6"),
			Field:           hexColor("#ffffff"),
			FieldBorder:     hexColor("#b8b8be"),
			Focus:           hexColor("#7ab8ff"),
			Selection:       hexAlpha("#4c8bf5", 0.28),
			Shadow:          hexAlpha("#202028", 0.22),
			Highlight:       hexAlpha("#ffffff", 0.75),
			MenuHover:       hexColor("#d6e6ff"),
			MenuHoverBorder: hexColor("#4c8bf5"),
			MenuGutter:      hexColor("#e8e8ea"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#b0b0b6"),
		},
		Hot:      ChromeState{Fill: hexColor("#e8f0ff"), Border: hexColor("#7ab8ff")},
		Pressed:  ChromeState{Fill: hexColor("#d0e0ff"), Border: hexColor("#4c8bf5")},
		Selected: ChromeState{Fill: hexAlpha("#4c8bf5", 0.22), Border: hexColor("#4c8bf5")},
	})
}

func packAquaNight() ThemePack {
	return eraPack("aqua-night", "Aqua Night", EraAqua, ThemeDark, BevelSoftShadow, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 8, RadiusSmall: 6, BevelDepth: 0, Scroll: 11, Elevation: 1},
		Palette: Palette{
			Background:      hexColor("#1e1e22"),
			Surface:         hexColor("#2a2a30"),
			SurfaceAlt:      hexColor("#24242a"),
			Overlay:         hexAlpha("#000000", 0.50),
			Border:          hexColor("#3a3a44"),
			Divider:         hexColor("#32323a"),
			Text:            hexColor("#f0f0f4"),
			TextMuted:       hexColor("#9a9aa4"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#5b9cff"),
			AccentHover:     hexColor("#7ab0ff"),
			AccentPress:     hexColor("#3a78d8"),
			Danger:          hexColor("#e24b4b"),
			Success:         hexColor("#3cb86a"),
			Warning:         hexColor("#e0a020"),
			Track:           hexColor("#1a1a1e"),
			Thumb:           hexColor("#5a5a66"),
			Field:           hexColor("#1a1a1e"),
			FieldBorder:     hexColor("#3a3a44"),
			Focus:           hexColor("#7ab8ff"),
			Selection:       hexAlpha("#5b9cff", 0.32),
			Shadow:          hexAlpha("#000000", 0.50),
			Highlight:       hexAlpha("#ffffff", 0.08),
			MenuHover:       hexColor("#2a3a58"),
			MenuHoverBorder: hexColor("#5b9cff"),
			MenuGutter:      hexColor("#222228"),
			BevelLight:      hexColor("#4a4a54"),
			BevelDark:       hexColor("#101014"),
		},
		Hot:      ChromeState{Fill: hexColor("#2a3a58"), Border: hexColor("#5b9cff")},
		Pressed:  ChromeState{Fill: hexColor("#223048"), Border: hexColor("#5b9cff")},
		Selected: ChromeState{Fill: hexAlpha("#5b9cff", 0.28), Border: hexColor("#5b9cff")},
	})
}

func packFusion() ThemePack {
	return eraPack("fusion", "Fusion", EraFusion, ThemeLight, BevelClassic3D, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 3, RadiusSmall: 2, BevelDepth: 1, Scroll: 12, ControlH: 30, ComboH: 26, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#efefef"),
			Surface:         hexColor("#f6f6f6"),
			SurfaceAlt:      hexColor("#e4e4e4"),
			Overlay:         hexAlpha("#202020", 0.32),
			Border:          hexColor("#a0a0a0"),
			Divider:         hexColor("#c4c4c4"),
			Text:            hexColor("#202020"),
			TextMuted:       hexColor("#5a5a5a"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#4a90d9"),
			AccentHover:     hexColor("#62a4e4"),
			AccentPress:     hexColor("#3278c0"),
			Danger:          hexColor("#c04040"),
			Success:         hexColor("#2a8a48"),
			Warning:         hexColor("#c09020"),
			Track:           hexColor("#d4d4d4"),
			Thumb:           hexColor("#b0b0b0"),
			Field:           hexColor("#ffffff"),
			FieldBorder:     hexColor("#8a8a8a"),
			Focus:           hexColor("#4a90d9"),
			Selection:       hexAlpha("#4a90d9", 0.30),
			Shadow:          hexAlpha("#000000", 0.20),
			Highlight:       hexAlpha("#ffffff", 0.70),
			MenuHover:       hexColor("#cde4f8"),
			MenuHoverBorder: hexColor("#4a90d9"),
			MenuGutter:      hexColor("#e4e4e4"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#8a8a8a"),
		},
		Hot:      ChromeState{Fill: hexColor("#e8f2fc"), Border: hexColor("#4a90d9")},
		Pressed:  ChromeState{Fill: hexColor("#d0e4f6"), Border: hexColor("#3278c0")},
		Selected: ChromeState{Fill: hexAlpha("#4a90d9", 0.28), Border: hexColor("#4a90d9")},
	})
}

func packFusionNight() ThemePack {
	return eraPack("fusion-night", "Fusion Night", EraFusion, ThemeDark, BevelClassic3D, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 3, RadiusSmall: 2, BevelDepth: 1, Scroll: 12, ControlH: 30, ComboH: 26, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#323232"),
			Surface:         hexColor("#3c3c3c"),
			SurfaceAlt:      hexColor("#363636"),
			Overlay:         hexAlpha("#000000", 0.50),
			Border:          hexColor("#202020"),
			Divider:         hexColor("#2a2a2a"),
			Text:            hexColor("#eeeeee"),
			TextMuted:       hexColor("#aaaaaa"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#3d8fd9"),
			AccentHover:     hexColor("#56a2e4"),
			AccentPress:     hexColor("#2a70b0"),
			Danger:          hexColor("#d05050"),
			Success:         hexColor("#40a860"),
			Warning:         hexColor("#d0b040"),
			Track:           hexColor("#2a2a2a"),
			Thumb:           hexColor("#5a5a5a"),
			Field:           hexColor("#2a2a2a"),
			FieldBorder:     hexColor("#1a1a1a"),
			Focus:           hexColor("#3d8fd9"),
			Selection:       hexAlpha("#3d8fd9", 0.38),
			Shadow:          hexAlpha("#000000", 0.45),
			Highlight:       hexAlpha("#ffffff", 0.08),
			MenuHover:       hexColor("#2a4a6a"),
			MenuHoverBorder: hexColor("#3d8fd9"),
			MenuGutter:      hexColor("#303030"),
			BevelLight:      hexColor("#5a5a5a"),
			BevelDark:       hexColor("#1a1a1a"),
		},
		Hot:      ChromeState{Fill: hexColor("#2a4a6a"), Border: hexColor("#3d8fd9")},
		Pressed:  ChromeState{Fill: hexColor("#1e3850"), Border: hexColor("#3d8fd9")},
		Selected: ChromeState{Fill: hexAlpha("#3d8fd9", 0.35), Border: hexColor("#3d8fd9")},
	})
}

func packBreeze() ThemePack {
	return eraPack("breeze", "Breeze", EraBreeze, ThemeLight, BevelNone, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 4, RadiusSmall: 3, BevelDepth: 0, Scroll: 8, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#eff0f1"),
			Surface:         hexColor("#fcfcfc"),
			SurfaceAlt:      hexColor("#e8eaed"),
			Overlay:         hexAlpha("#232629", 0.30),
			Border:          hexColor("#c0c4c8"),
			Divider:         hexColor("#dcdfe3"),
			Text:            hexColor("#232629"),
			TextMuted:       hexColor("#5a6168"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#3daee9"),
			AccentHover:     hexColor("#55bdf2"),
			AccentPress:     hexColor("#2180b8"),
			Danger:          hexColor("#da4453"),
			Success:         hexColor("#27ae60"),
			Warning:         hexColor("#f67400"),
			Track:           hexColor("#dcdfe3"),
			Thumb:           hexColor("#b0b6bc"),
			Field:           hexColor("#ffffff"),
			FieldBorder:     hexColor("#b0b6bc"),
			Focus:           hexColor("#3daee9"),
			Selection:       hexAlpha("#3daee9", 0.28),
			Shadow:          hexAlpha("#232629", 0.16),
			Highlight:       hexAlpha("#3daee9", 0.12),
			MenuHover:       hexColor("#d8eef8"),
			MenuHoverBorder: hexColor("#3daee9"),
			MenuGutter:      hexColor("#e8eaed"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#b0b6bc"),
		},
		Hot:      ChromeState{Fill: hexColor("#e8f4fb"), Border: hexColor("#3daee9")},
		Pressed:  ChromeState{Fill: hexColor("#d0eaf6"), Border: hexColor("#2180b8")},
		Selected: ChromeState{Fill: hexAlpha("#3daee9", 0.24), Border: hexColor("#3daee9")},
		Focus:    ChromeState{Fill: hexAlpha("#3daee9", 0.12), Border: hexColor("#3daee9")},
	})
}

func packBreezeNight() ThemePack {
	return eraPack("breeze-night", "Breeze Night", EraBreeze, ThemeDark, BevelNone, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 4, RadiusSmall: 3, BevelDepth: 0, Scroll: 8, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#2a2e32"),
			Surface:         hexColor("#31363b"),
			SurfaceAlt:      hexColor("#2a2e32"),
			Overlay:         hexAlpha("#000000", 0.50),
			Border:          hexColor("#1a1e22"),
			Divider:         hexColor("#3a4046"),
			Text:            hexColor("#eff0f1"),
			TextMuted:       hexColor("#a0a6ac"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#3daee9"),
			AccentHover:     hexColor("#55bdf2"),
			AccentPress:     hexColor("#2180b8"),
			Danger:          hexColor("#da4453"),
			Success:         hexColor("#27ae60"),
			Warning:         hexColor("#f67400"),
			Track:           hexColor("#1e2226"),
			Thumb:           hexColor("#5a626a"),
			Field:           hexColor("#23282e"),
			FieldBorder:     hexColor("#1a1e22"),
			Focus:           hexColor("#3daee9"),
			Selection:       hexAlpha("#3daee9", 0.32),
			Shadow:          hexAlpha("#000000", 0.45),
			Highlight:       hexAlpha("#3daee9", 0.14),
			MenuHover:       hexColor("#1e4a62"),
			MenuHoverBorder: hexColor("#3daee9"),
			MenuGutter:      hexColor("#262a30"),
			BevelLight:      hexColor("#4a5056"),
			BevelDark:       hexColor("#12161a"),
		},
		Hot:      ChromeState{Fill: hexColor("#1e4a62"), Border: hexColor("#3daee9")},
		Pressed:  ChromeState{Fill: hexColor("#163848"), Border: hexColor("#2180b8")},
		Selected: ChromeState{Fill: hexAlpha("#3daee9", 0.30), Border: hexColor("#3daee9")},
		Focus:    ChromeState{Fill: hexAlpha("#3daee9", 0.14), Border: hexColor("#3daee9")},
	})
}

func packFluent() ThemePack {
	return eraPack("fluent", "Fluent", EraFluent, ThemeLight, BevelFluentAccent, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 4, RadiusSmall: 4, BevelDepth: 0, Scroll: 8, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#f3f3f3"),
			Surface:         hexColor("#ffffff"),
			SurfaceAlt:      hexColor("#f9f9f9"),
			Overlay:         hexAlpha("#202020", 0.28),
			Border:          hexColor("#e0e0e0"),
			Divider:         hexColor("#ebebeb"),
			Text:            hexColor("#1a1a1a"),
			TextMuted:       hexColor("#5a5a5a"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#0067c0"),
			AccentHover:     hexColor("#1975c9"),
			AccentPress:     hexColor("#005aa8"),
			Danger:          hexColor("#c42b1c"),
			Success:         hexColor("#0f7b0f"),
			Warning:         hexColor("#9d5d00"),
			Track:           hexColor("#e8e8e8"),
			Thumb:           hexColor("#8a8a8a"),
			Field:           hexColor("#ffffff"),
			FieldBorder:     hexColor("#8a8a8a"),
			Focus:           hexColor("#0067c0"),
			Selection:       hexAlpha("#0067c0", 0.18),
			Shadow:          hexAlpha("#000000", 0.12),
			Highlight:       hexColor("#e8e8e8"),
			MenuHover:       hexColor("#e8e8e8"),
			MenuHoverBorder: hexColor("#0067c0"),
			MenuGutter:      hexColor("#f3f3f3"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#d0d0d0"),
		},
		Hot:      ChromeState{Fill: hexColor("#e8e8e8"), Border: hexColor("#c0c0c0")},
		Pressed:  ChromeState{Fill: hexColor("#d8d8d8"), Border: hexColor("#0067c0")},
		Selected: ChromeState{Fill: hexAlpha("#0067c0", 0.12), Border: hexColor("#0067c0")},
		Focus:    ChromeState{Fill: hexColor("#ffffff"), Border: hexColor("#0067c0")},
	})
}

func packFluentNight() ThemePack {
	return eraPack("fluent-night", "Fluent Night", EraFluent, ThemeDark, BevelFluentAccent, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 4, RadiusSmall: 4, BevelDepth: 0, Scroll: 8, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#202020"),
			Surface:         hexColor("#2c2c2c"),
			SurfaceAlt:      hexColor("#282828"),
			Overlay:         hexAlpha("#000000", 0.50),
			Border:          hexColor("#3a3a3a"),
			Divider:         hexColor("#333333"),
			Text:            hexColor("#ffffff"),
			TextMuted:       hexColor("#b0b0b0"),
			TextOnAccent:    hexColor("#000000"),
			Accent:          hexColor("#60cdff"),
			AccentHover:     hexColor("#7ad6ff"),
			AccentPress:     hexColor("#3ab4e8"),
			Danger:          hexColor("#ff99a4"),
			Success:         hexColor("#6ccb5f"),
			Warning:         hexColor("#fce100"),
			Track:           hexColor("#1a1a1a"),
			Thumb:           hexColor("#6a6a6a"),
			Field:           hexColor("#1c1c1c"),
			FieldBorder:     hexColor("#5a5a5a"),
			Focus:           hexColor("#60cdff"),
			Selection:       hexAlpha("#60cdff", 0.22),
			Shadow:          hexAlpha("#000000", 0.40),
			Highlight:       hexColor("#3a3a3a"),
			MenuHover:       hexColor("#3a3a3a"),
			MenuHoverBorder: hexColor("#60cdff"),
			MenuGutter:      hexColor("#242424"),
			BevelLight:      hexColor("#4a4a4a"),
			BevelDark:       hexColor("#101010"),
		},
		Hot:      ChromeState{Fill: hexColor("#3a3a3a"), Border: hexColor("#5a5a5a")},
		Pressed:  ChromeState{Fill: hexColor("#323232"), Border: hexColor("#60cdff")},
		Selected: ChromeState{Fill: hexAlpha("#60cdff", 0.18), Border: hexColor("#60cdff")},
		Focus:    ChromeState{Fill: hexColor("#2c2c2c"), Border: hexColor("#60cdff")},
	})
}

func packMaterial() ThemePack {
	return eraPack("material", "Material", EraMaterial, ThemeLight, BevelSoftShadow, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 12, RadiusSmall: 8, BevelDepth: 0, Scroll: 8, Elevation: 1},
		Palette: Palette{
			Background:      hexColor("#fef7ff"),
			Surface:         hexColor("#ffffff"),
			SurfaceAlt:      hexColor("#f3edf7"),
			Overlay:         hexAlpha("#1d1b20", 0.32),
			Border:          hexColor("#cac4d0"),
			Divider:         hexColor("#e7e0ec"),
			Text:            hexColor("#1d1b20"),
			TextMuted:       hexColor("#49454f"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#6750a4"),
			AccentHover:     hexColor("#7a66b5"),
			AccentPress:     hexColor("#4f3d86"),
			Danger:          hexColor("#b3261e"),
			Success:         hexColor("#386a20"),
			Warning:         hexColor("#7d5700"),
			Track:           hexColor("#e7e0ec"),
			Thumb:           hexColor("#79747e"),
			Field:           hexColor("#f7f2fa"),
			FieldBorder:     hexColor("#79747e"),
			Focus:           hexColor("#6750a4"),
			Selection:       hexAlpha("#6750a4", 0.18),
			Shadow:          hexAlpha("#1d1b20", 0.20),
			Highlight:       hexColor("#e8def8"),
			MenuHover:       hexColor("#e8def8"),
			MenuHoverBorder: hexColor("#6750a4"),
			MenuGutter:      hexColor("#f3edf7"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#cac4d0"),
		},
		Hot:      ChromeState{Fill: hexColor("#e8def8"), Border: hexColor("#6750a4")},
		Pressed:  ChromeState{Fill: hexColor("#d0c4e8"), Border: hexColor("#4f3d86")},
		Selected: ChromeState{Fill: hexColor("#e8def8"), Border: hexColor("#6750a4")},
		Focus:    ChromeState{Fill: hexAlpha("#6750a4", 0.10), Border: hexColor("#6750a4")},
	})
}

func packMaterialNight() ThemePack {
	return eraPack("material-night", "Material Night", EraMaterial, ThemeDark, BevelSoftShadow, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 12, RadiusSmall: 8, BevelDepth: 0, Scroll: 8, Elevation: 1},
		Palette: Palette{
			Background:      hexColor("#141218"),
			Surface:         hexColor("#1d1b20"),
			SurfaceAlt:      hexColor("#2b2930"),
			Overlay:         hexAlpha("#000000", 0.52),
			Border:          hexColor("#49454f"),
			Divider:         hexColor("#36323c"),
			Text:            hexColor("#e6e0e9"),
			TextMuted:       hexColor("#cac4d0"),
			TextOnAccent:    hexColor("#381e72"),
			Accent:          hexColor("#d0bcff"),
			AccentHover:     hexColor("#dccbff"),
			AccentPress:     hexColor("#b69df8"),
			Danger:          hexColor("#f2b8b5"),
			Success:         hexColor("#b5d99a"),
			Warning:         hexColor("#e8c47a"),
			Track:           hexColor("#2b2930"),
			Thumb:           hexColor("#79747e"),
			Field:           hexColor("#211f26"),
			FieldBorder:     hexColor("#938f99"),
			Focus:           hexColor("#d0bcff"),
			Selection:       hexAlpha("#d0bcff", 0.22),
			Shadow:          hexAlpha("#000000", 0.50),
			Highlight:       hexColor("#4a4458"),
			MenuHover:       hexColor("#4a4458"),
			MenuHoverBorder: hexColor("#d0bcff"),
			MenuGutter:      hexColor("#211f26"),
			BevelLight:      hexColor("#4a4458"),
			BevelDark:       hexColor("#0c0b10"),
		},
		Hot:      ChromeState{Fill: hexColor("#4a4458"), Border: hexColor("#d0bcff")},
		Pressed:  ChromeState{Fill: hexColor("#3a3646"), Border: hexColor("#d0bcff")},
		Selected: ChromeState{Fill: hexColor("#4a4458"), Border: hexColor("#d0bcff")},
		Focus:    ChromeState{Fill: hexAlpha("#d0bcff", 0.12), Border: hexColor("#d0bcff")},
	})
}

func packFlatLaf() ThemePack {
	return eraPack("flatlaf", "FlatLaf", EraFlatLaf, ThemeLight, BevelNone, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 2, RadiusSmall: 0, BevelDepth: 0, Scroll: 10, ControlH: 26, ComboH: 24, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#f2f2f2"),
			Surface:         hexColor("#ffffff"),
			SurfaceAlt:      hexColor("#e8e8e8"),
			Overlay:         hexAlpha("#202020", 0.30),
			Border:          hexColor("#c4c4c4"),
			Divider:         hexColor("#d8d8d8"),
			Text:            hexColor("#000000"),
			TextMuted:       hexColor("#6a6a6a"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#4a88c7"),
			AccentHover:     hexColor("#5c96d0"),
			AccentPress:     hexColor("#3a70a8"),
			Danger:          hexColor("#c75450"),
			Success:         hexColor("#499c54"),
			Warning:         hexColor("#c29b4a"),
			Track:           hexColor("#d8d8d8"),
			Thumb:           hexColor("#b0b0b0"),
			Field:           hexColor("#ffffff"),
			FieldBorder:     hexColor("#c4c4c4"),
			Focus:           hexColor("#4a88c7"),
			Selection:       hexColor("#d5e1f2"),
			Shadow:          hexAlpha("#000000", 0.14),
			Highlight:       hexColor("#d5e1f2"),
			MenuHover:       hexColor("#d5e1f2"),
			MenuHoverBorder: hexColor("#4a88c7"),
			MenuGutter:      hexColor("#f2f2f2"),
			BevelLight:      hexColor("#ffffff"),
			BevelDark:       hexColor("#c4c4c4"),
		},
		Hot:      ChromeState{Fill: hexColor("#d5e1f2"), Border: hexColor("#4a88c7")},
		Pressed:  ChromeState{Fill: hexColor("#c0d4ec"), Border: hexColor("#3a70a8")},
		Selected: ChromeState{Fill: hexColor("#d5e1f2"), Border: hexColor("#4a88c7")},
		Focus:    ChromeState{Fill: hexColor("#ffffff"), Border: hexColor("#4a88c7")},
	})
}

func packFlatLafNight() ThemePack {
	return eraPack("flatlaf-night", "FlatLaf Night", EraFlatLaf, ThemeDark, BevelNone, ThemeTokens{
		Metrics: ChromeMetrics{Radius: 2, RadiusSmall: 0, BevelDepth: 0, Scroll: 10, ControlH: 26, ComboH: 24, Elevation: 0},
		Palette: Palette{
			Background:      hexColor("#3c3f41"),
			Surface:         hexColor("#4e5254"),
			SurfaceAlt:      hexColor("#3c3f41"),
			Overlay:         hexAlpha("#000000", 0.50),
			Border:          hexColor("#2b2b2b"),
			Divider:         hexColor("#323232"),
			Text:            hexColor("#bbbbbb"),
			TextMuted:       hexColor("#808080"),
			TextOnAccent:    hexColor("#ffffff"),
			Accent:          hexColor("#4a88c7"),
			AccentHover:     hexColor("#5c96d0"),
			AccentPress:     hexColor("#3a70a8"),
			Danger:          hexColor("#c75450"),
			Success:         hexColor("#499c54"),
			Warning:         hexColor("#c29b4a"),
			Track:           hexColor("#313335"),
			Thumb:           hexColor("#5a5d5f"),
			Field:           hexColor("#45494a"),
			FieldBorder:     hexColor("#646464"),
			Focus:           hexColor("#4a88c7"),
			Selection:       hexColor("#4b6eaf"),
			Shadow:          hexAlpha("#000000", 0.40),
			Highlight:       hexColor("#4b6eaf"),
			MenuHover:       hexColor("#4b6eaf"),
			MenuHoverBorder: hexColor("#4a88c7"),
			MenuGutter:      hexColor("#3c3f41"),
			BevelLight:      hexColor("#5a5d5f"),
			BevelDark:       hexColor("#2b2b2b"),
		},
		Hot:      ChromeState{Fill: hexColor("#4b6eaf"), Border: hexColor("#4a88c7")},
		Pressed:  ChromeState{Fill: hexColor("#3a5a90"), Border: hexColor("#4a88c7")},
		Selected: ChromeState{Fill: hexColor("#4b6eaf"), Border: hexColor("#4a88c7")},
		Focus:    ChromeState{Fill: hexColor("#45494a"), Border: hexColor("#4a88c7")},
	})
}

func aliasThemeName(name string) string {
	switch name {
	case "classic95", "classic95-light", "win95", "win98":
		return "light"
	case "classic95-dark", "win95-dark":
		return "dark"
	case "cde-charcoal", "charcoal":
		return "cde"
	case "next-dark":
		return "next-night"
	case "luna-dark":
		return "luna-night"
	case "aqua-dark":
		return "aqua-night"
	case "fusion-dark":
		return "fusion-night"
	case "breeze-dark", "adwaita", "adwaita-dark":
		if name == "adwaita" {
			return "breeze"
		}
		return "breeze-night"
	case "fluent-dark":
		return "fluent-night"
	case "material-dark":
		return "material-night"
	case "flatlaf-dark":
		return "flatlaf-night"
	default:
		return name
	}
}
