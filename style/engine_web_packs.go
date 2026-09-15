package style

import (
	"maps"

	"github.com/codemodify/paintengine2d"
)

// The web engine's packs: each design system as data (see engine_web.go
// for the keys). Colours are the systems' own published values; where a
// value is inferred it says so.

// webSpec is one web pack: its identity, colours (the engine's keys), params
// and metrics in the pack's own pixels.
type webSpec struct {
	name, label, era, lineage, summary string
	year                               int
	dark                               bool
	c                                  map[string]string
	p                                  map[string]float32
	m                                  ChromeMetrics
}

// webWith is base with over's entries on top.
func webWith[V any](base, over map[string]V) map[string]V {
	out := make(map[string]V, len(base)+len(over))
	maps.Copy(out, base)
	maps.Copy(out, over)
	return out
}

// webHex formats a derived colour for a pack table.
func webHex(c paintengine2d.Color) string { return colorHexPadded(c) }

// webPack builds a pack: every colour goes to "extra" for the engine, and the
// shared palette the widgets read is filled from the same roles.
func webPack(s webSpec) ThemePack {
	extra := make(map[string]paintengine2d.Color, len(s.c))
	for k, v := range s.c {
		if c, ok := ParseHexColor(v); ok {
			extra[k] = markExplicitColor(c)
		}
	}
	get := func(k string, def paintengine2d.Color) paintengine2d.Color {
		if c, ok := extra[k]; ok {
			return c
		}
		return def
	}
	window := get("window", Hex("#ffffff"))
	if _, ok := extra["window"]; !ok && s.dark {
		window = Hex("#1e1e1e")
	}
	text := get("text", Contrast(window))
	field := get("field", window)
	popup := get("popup", window)
	accent := get("accent", Hex("#0078d7"))
	border1 := get("border1", Mix(window, text, 0.35))
	border2 := get("border2", Mix(window, text, 0.15))
	flat := func(base, c paintengine2d.Color) paintengine2d.Color { return webOver(base, c) }
	wash := get("wash", text.WithAlpha(0.08))
	menuHover := get("menuHover", wash)
	fam := ThemeLight
	shadow, overlay := paintengine2d.RGBA(0, 0, 0, 0.18), paintengine2d.RGBA(0, 0, 0, 0.35)
	if s.dark {
		fam = ThemeDark
		shadow, overlay = paintengine2d.RGBA(0, 0, 0, 0.5), paintengine2d.RGBA(0, 0, 0, 0.55)
	}
	pal := Palette{
		Background: window, Surface: window, SurfaceAlt: popup,
		Overlay: get("overlay", overlay),
		Border:  flat(window, border1), Divider: flat(window, border2),
		Text: text, TextMuted: get("text2", Mix(window, text, 0.6)), TextOnAccent: get("onAccent", Contrast(accent)),
		Accent: accent, AccentHover: get("accentHover", accent), AccentPress: get("accentPress", accent),
		Danger: get("danger", Hex("#d13438")), Success: get("success", Hex("#107c10")), Warning: get("warning", Hex("#c19c00")),
		Track: flat(window, get("progressTrack", text.WithAlpha(0.12))), Thumb: flat(window, get("scrollThumb", text.WithAlpha(0.5))),
		Field: field, FieldBorder: flat(field, border1),
		Focus: get("focus", accent), Selection: get("selection", accent),
		Shadow: get("shadow", shadow), Highlight: wash,
		MenuHover: flat(popup, menuHover), MenuHoverBorder: flat(popup, menuHover), MenuGutter: popup,
		BevelLight: field, BevelDark: flat(window, border1),
	}
	tok := ThemeTokens{
		Engine: "web", Bevel: BevelNone, Family: fam,
		Palette: pal, Extra: extra, Params: maps.Clone(s.p), Metrics: s.m,
	}
	webChrome(&tok)
	return ThemePack{
		Name: s.name, Label: s.label, Year: s.year, Lineage: s.lineage, Summary: s.summary,
		Era: s.era, Palette: fam, Tokens: tok,
	}
}

func webPacks() []ThemePack {
	var out []ThemePack
	for _, s := range webSpecs() {
		out = append(out, webPack(s))
	}
	return out
}

// webSpecs lists every web pack.
func webSpecs() []webSpec {
	specs := []webSpec{sourcegitSpec(false), sourcegitSpec(true)}
	specs = append(specs, primerSpecs()...)
	specs = append(specs, shadcnSpecs()...)
	specs = append(specs, geistSpecs()...)
	specs = append(specs, linearSpecs()...)
	specs = append(specs, paletteSpecs()...)
	return specs
}

// ---- SourceGit ----------------------------------------------------------------------------------

// sourcegitSpec is SourceGit (v2026.20; the Avalonia UI of 2024 on): the
// Light and Dark dictionaries of its Themes.axaml mapped onto the engine's
// keys, Avalonia Fluent's resources where SourceGit inherits them (the
// accent and its Light1 / Dark1 shades, the 9.8% SystemListLow wash, the
// 3px and 5px corners), its Styles.axaml controls as params and its sizes at
// its own 13px Inter.
func sourcegitSpec(dark bool) webSpec {
	accent := Hex("#0078d7") // the OS accent's fallback
	c := map[string]string{
		"accent": "#0078d7", "accentHover": webHex(webLight1(accent)), "accentPress": webHex(webDark1(accent)),
		"onAccent": "#ffffff", "focus": "#0078d7", "fieldFocus": "#0078d7",
		"selection": "#0078d7", "selectionText": "#ffffff",
		"checkOff": "#00000000", "fieldDis": "#00000000", "switchOff": "#00000000",
		"close": "#ff0000", "captionHover": "#00000040",
		"danger": "#ff0000", "success": "#008000", "warning": "#ff8c00",
	}
	if dark {
		c = webWith(c, map[string]string{
			"window": "#252525", "windowBorder": "#606060", "titleBar": "#1f1f1f", "toolBar": "#2f2f2f",
			"popup": "#2b2b2b", "popupBorder": "#393939", "field": "#1c1c1c", "subPanel": "#272727",
			"border0": "#181818", "border1": "#7c7c7c", "border2": "#404040",
			"btn": "#303030", "btnHover": "#333333", "btnPress": "#333333", "btnBorder": "#404040",
			"text": "#dfdfdf", "text2": "#9f9f9f", "textDis": "#7c7c7c", "link": "#4daafc",
			"wash": "#ffffff19", "headerText": "#dfdfdfa6", "tabText2": "#dfdfdf8f",
			"switchOffBorder": "#ffffff99", "knobOff": "#ffffff", "track": "#ffffff66", "progressTrack": "#ffffff33",
			"scrollThumb": "#9f9f9f", "scrollThumbHot": "#dfdfdf", "onClose": "#dfdfdf",
			"tabPane": "#272727", "sidebar": "#252525", "card": "#252525", "header": "#1c1c1c", "tabSep": "#ffffff33",
		})
	} else {
		c = webWith(c, map[string]string{
			"window": "#f4f4f4", "windowBorder": "#999999", "titleBar": "#f2f0f0", "toolBar": "#f0f3f5",
			"popup": "#fafafa", "popupBorder": "#ffffff", "field": "#ffffff", "subPanel": "#fbfbfb",
			"border0": "#cfcfcf", "border1": "#898989", "border2": "#cfcfcf",
			"btn": "#f8f8f8", "btnHover": "#ffffff", "btnPress": "#ffffff", "btnBorder": "#cfcfcf",
			"text": "#1f1f1f", "text2": "#6f6f6f", "textDis": "#929292", "link": "#0000ee",
			"wash": "#00000019", "headerText": "#1f1f1fa6", "tabText2": "#1f1f1f8f",
			"switchOffBorder": "#00000099", "knobOff": "#000000", "track": "#00000066", "progressTrack": "#00000033",
			"scrollThumb": "#6f6f6f", "scrollThumbHot": "#1f1f1f", "onClose": "#1f1f1f",
			"tabPane": "#fbfbfb", "sidebar": "#f4f4f4", "card": "#f4f4f4", "header": "#ffffff", "tabSep": "#00000033",
		})
	}
	p := map[string]float32{
		// Corners: Fluent's ControlCornerRadius 3 and OverlayCornerRadius
		// 5; square text boxes; 4px tool tips, list frames and sidebar
		// rows; 3px menu rows; the 8px content card and Linux window.
		"radius": 3, "fieldRadius": 0, "comboRadius": 3, "overlayRadius": 5, "tipRadius": 4,
		"cardRadius": 8, "viewRadius": 4, "windowRadius": 8, "checkRadius": 2,
		"rowRadius": 3, "menuRowRadius": 3, "sideRadius": 4, "menuInset": 8,
		"pressScale": 0.98, "disabledAlpha": 0.6,
		// The dotted focus adorner; accent borders under the pointer; check
		// boxes and radios take a 2px accent border for keyboard focus.
		"focusStyle": webFocusDotted, "focusWidth": 1, "hoverBorder": 1, "checkFocus": 1,
		// Bold flat buttons; the accent primary; OK before Cancel.
		"primaryStyle": 0, "buttonWeight": 700, "primaryFirst": 1,
		// Tabs: bold labels a size up, FG1 at 56%, the selected one in the
		// accent over a 1px accent pipe 2px above the bottom.
		"tabStyle": webTabUnderline, "tabLine": 1, "tabLineGap": 2, "tabFit": 1, "tabAccent": 1,
		"tabDim": 0.56, "tabWeight": 700, "tabGrow": 1, "tabBar": 0,
		// A 16px box with a 12px accent tick and no fill; a 14px ring with a
		// 10px dot; Avalonia's 40×20 switch with a 10px knob; the 2px slider
		// track and 16px thumb; the 4px progress bar.
		"checkStyle": 0, "checkSize": 16, "radioSize": 14, "radioStyle": 0, "radioDot": 10,
		"switchShape": 0, "knob": 10, "trackH": 2, "thumbStyle": 0, "progressH": 4,
		"menuHighlight": 0, "menuSep": 1,
		// Lists: Fluent's full-width accent at 40% (60% under the pointer);
		// sidebars: 4px rows inset 6px, the accent at 65% (80%) with focus,
		// the 10% wash without, 5% under the pointer, joined when adjacent;
		// the commit table: the accent at 60% (80%) either way.
		"listSel": 0.4, "listSelHover": 0.6, "listSelOff": 0.4, "listHover": 1,
		"sideSel": 0.65, "sideSelHover": 0.8, "sideSelOff": 1, "sideOffWash": 1, "sideHover": 0.5, "sideInset": 6,
		"tableSel": 0.6, "tableSelHover": 0.8, "tableSelOff": 0.6, "tableHover": 1,
		"rowInset": 0, "expander": 0,
		// 8px overlay bars: a 2px idle thumb, arrow cells when expanded.
		"scrollIdle": 2, "scrollInset": 0, "scrollArrows": 1,
		// The dialog's 28px caption strip, 48px close cell.
		"captionStyle": 0, "captionButton": 48,
		// Icon buttons: no face, the glyph at 80% to 100%; accent pills
		// for checked segments.
		"toolWash": 0, "toolAlpha": 0.8, "seg": 0,
		"headerWeight": 700, "headerCenter": 1, "accordion": 0,
		// drop-shadow(0 0 6 #80000000) under menus, (0 0 8 #60000000)
		// under tips, (0 0 12 #60000000) round windows.
		"menuShadow": 0.5, "menuShadowY": 0, "menuShadowBlur": 12,
		"tipShadow": 0.376, "tipShadowY": 0, "tipShadowBlur": 16,
		"dialogShadow": 0.376, "dialogShadowY": 0, "dialogShadowBlur": 24,
		"accentFollows": 1,
	}
	m := ChromeMetrics{
		FontSize: 13, Radius: 3, RadiusSmall: 3,
		ControlH: 28, FieldH: 28, ComboH: 28, Checkbox: 16, Radio: 16,
		MenuItemH: 28, MenuBarH: 28, TabH: 30, RowH: 26, TitleBar: 28, HeaderH: 24,
		ProgressH: 10, SliderH: 20, Thumb: 16, Scroll: 8, Pad: 8, FieldPad: 5,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 20, SwitchW: 40, SwitchH: 20,
		ViewFrame: 1, Border: 1, FocusWidth: 1,
	}
	s := webSpec{
		name: "sourcegit", label: "SourceGit", era: "SourceGit", lineage: "SourceGit", year: 2024,
		summary: "SourceGit's Avalonia look: square fields, bold 3px flat buttons, the 1px accent tab pipe, unfilled check boxes with an accent tick, joined sidebar selections.",
		c:       c, p: p, m: m,
	}
	if dark {
		s.name, s.label, s.dark = "sourcegit-night", "SourceGit Dark", true
		s.summary = "SourceGit's dark theme: #252525 windows around #1c1c1c content, the same Avalonia controls and the OS accent."
	}
	return s
}

// ---- GitHub Primer ------------------------------------------------------------------------------

// primerSpecs are GitHub's Primer (@primer/primitives 11.10, the
// functional tokens of 2023 on) in its light, dark and dark dimmed themes.
func primerSpecs() []webSpec {
	p := map[string]float32{
		// Radius medium 6 for controls, large 12 for overlays and dialogs,
		// small 3 for check boxes; ActionList rows 6px, inset 8px.
		"radius": 6, "overlayRadius": 12, "tipRadius": 6, "cardRadius": 6, "viewRadius": 6, "windowRadius": 12,
		"checkRadius": 3, "rowRadius": 6, "menuInset": 8, "switchShape": 1, "switchRadius": 6,
		// The 2px focus outline at −2px (inside), a light line inside it on
		// the primary button.
		"focusStyle": webFocusInside, "focusWidth": 2,
		"primaryStyle": 2, "buttonWeight": 500,
		// UnderlineNav: a 2px coral bar across the item, semibold current
		// label, a wash under the pointer, a hairline under the strip.
		"tabStyle": webTabUnderline, "tabLine": 2, "tabWeight": 400, "tabSelWeight": 600, "tabHover": 1, "tabBar": 1,
		// 16px boxes filled with the accent, radios with a 4px accent ring.
		"checkStyle": 1, "checkSize": 16, "radioStyle": 1, "radioDot": 8,
		"knob": 22, "trackH": 4, "thumbStyle": 1, "progressH": 8,
		"menuHighlight": 0,
		// NavList / TreeView: the neutral wash and a 4px accent bar at the
		// current row's left, the same with or without focus.
		"rowInset": 8, "sideInset": 8, "rowBar": 4,
		"captionStyle": 1, "captionLine": 1, "toolWash": 1, "seg": 1,
		"headerWeight": 600, "headerSep": 0, "accordion": 1,
		// shadow-floating-small under menus; floating-large (0 40px 80px)
		// round dialogs, drawn at half its reach.
		"menuShadow": 0.12, "menuShadowY": 6, "menuShadowBlur": 36,
		"tipShadow": 0, "dialogShadow": 0.24, "dialogShadowY": 20, "dialogShadowBlur": 80,
		"hoverFade": 80,
	}
	m := ChromeMetrics{
		FontSize: 14, Radius: 6, RadiusSmall: 6,
		ControlH: 32, FieldH: 32, ComboH: 32, Checkbox: 22, Radio: 22,
		MenuItemH: 32, MenuBarH: 32, TabH: 40, RowH: 32, TitleBar: 48, HeaderH: 36,
		ProgressH: 12, SliderH: 20, Thumb: 16, Scroll: 10, Pad: 12, FieldPad: 12,
		ToolBarH: 40, StatusBarH: 28, SpinnerW: 22, SwitchW: 48, SwitchH: 24,
		ViewFrame: 1, Border: 1, FocusWidth: 2,
	}
	light := map[string]string{
		"window": "#ffffff", "field": "#ffffff", "subPanel": "#f6f8fa", "sidebar": "#f6f8fa", "titleBar": "#f6f8fa",
		"toolBar": "#f6f8fa", "card": "#ffffff", "tabPane": "#ffffff", "header": "#f6f8fa",
		"popup": "#ffffff", "popupBorder": "#d1d9e080", "dialog": "#ffffff",
		"border0": "#d1d9e0", "border1": "#d1d9e0", "border2": "#d1d9e0b3", "checkBorder": "#818b98",
		"text": "#1f2328", "text2": "#59636e", "textDis": "#818b98", "link": "#0969da", "headerText": "#59636e",
		"accent": "#0969da", "onAccent": "#ffffff", "focus": "#0969da", "fieldFocus": "#0969da",
		"btn": "#f6f8fa", "btnHover": "#eff2f5", "btnPress": "#e6eaef", "btnBorder": "#d1d9e0", "btnText": "#25292e",
		"primary": "#1f883d", "primaryHover": "#1c8139", "primaryPress": "#197935", "primaryBorder": "#1f232826", "onPrimary": "#ffffff",
		"wash": "#818b981a", "sel": "#818b981f", "menuHover": "#818b981a",
		"tip": "#25292e", "tipText": "#ffffff", "tipBorder": "#00000000",
		"checkOff": "#ffffff", "checkOn": "#0969da", "checkMark": "#ffffff",
		"switchOff": "#e6eaef", "switchOffBorder": "#d1d9e0", "switchOn": "#0969da", "knobOff": "#ffffff", "knobOn": "#ffffff", "knobBorder": "#d1d9e0",
		"track": "#d1d9e0", "thumbFill": "#ffffff", "thumbBorder": "#818b98",
		"progress": "#0969da", "progressTrack": "#d1d9e0",
		"scrollThumb": "#59636e80", "scrollThumbHot": "#59636e",
		"tabLine": "#fd8c73", "tabText2": "#1f2328",
		"segTrack": "#e6eaef", "segOn": "#ffffff", "segOnBorder": "#d1d9e0",
		"danger": "#d1242f", "success": "#1a7f37", "warning": "#9a6700",
		"selection": "#0969da33", "overlay": "#c8d1da66",
	}
	dark := map[string]string{
		"window": "#0d1117", "field": "#0d1117", "subPanel": "#151b23", "sidebar": "#151b23", "titleBar": "#151b23",
		"toolBar": "#151b23", "card": "#0d1117", "tabPane": "#0d1117", "header": "#151b23",
		"popup": "#010409", "popupBorder": "#3d444db3", "dialog": "#010409",
		"border0": "#3d444d", "border1": "#3d444d", "border2": "#3d444db3", "checkBorder": "#656c76",
		"text": "#f0f6fc", "text2": "#9198a1", "textDis": "#656c76", "link": "#4493f8", "headerText": "#9198a1",
		"accent": "#1f6feb", "onAccent": "#ffffff", "focus": "#1f6feb", "fieldFocus": "#1f6feb",
		"btn": "#212830", "btnHover": "#262c36", "btnPress": "#2a313c", "btnBorder": "#3d444d", "btnText": "#f0f6fc",
		"primary": "#238636", "primaryHover": "#29903b", "primaryPress": "#2e9a40", "primaryBorder": "#ffffff26", "onPrimary": "#ffffff",
		"wash": "#656c7633", "sel": "#656c7633", "menuHover": "#656c7633",
		"tip": "#3d444d", "tipText": "#ffffff", "tipBorder": "#00000000",
		"checkOff": "#0d1117", "checkOn": "#1f6feb", "checkMark": "#ffffff",
		"switchOff": "#010409", "switchOffBorder": "#3d444d", "switchOn": "#1f6feb", "knobOff": "#262c36", "knobOn": "#ffffff", "knobBorder": "#3d444d",
		"track": "#3d444d", "thumbFill": "#f0f6fc", "thumbBorder": "#656c76",
		"progress": "#1f6feb", "progressTrack": "#3d444d",
		"scrollThumb": "#9198a180", "scrollThumbHot": "#9198a1",
		"tabLine": "#f78166", "tabText2": "#f0f6fc",
		"segTrack": "#010409", "segOn": "#262c36", "segOnBorder": "#3d444d",
		"danger": "#f85149", "success": "#3fb950", "warning": "#d29922",
		"selection": "#1f6febb3", "overlay": "#21283066",
	}
	dimmed := map[string]string{
		"window": "#212830", "field": "#212830", "subPanel": "#262c36", "sidebar": "#262c36", "titleBar": "#262c36",
		"toolBar": "#262c36", "card": "#212830", "tabPane": "#212830", "header": "#262c36",
		"popup": "#2a313c", "popupBorder": "#3d444db3", "dialog": "#2a313c",
		"border0": "#3d444d", "border1": "#3d444d", "border2": "#3d444db3", "checkBorder": "#656c76",
		"text": "#d1d7e0", "text2": "#9198a1", "textDis": "#656c76", "link": "#478be6", "headerText": "#9198a1",
		"accent": "#316dca", "onAccent": "#f0f6fc", "focus": "#316dca", "fieldFocus": "#316dca",
		"btn": "#2a313c", "btnHover": "#2f3742", "btnPress": "#3d444d", "btnBorder": "#3d444d", "btnText": "#d1d7e0",
		"primary": "#347d39", "primaryHover": "#3b8640", "primaryPress": "#428f46", "primaryBorder": "#cdd9e526", "onPrimary": "#cdd9e5",
		"wash": "#656c7626", "sel": "#656c7633", "menuHover": "#656c7626",
		"tip": "#3d444d", "tipText": "#f0f6fc", "tipBorder": "#00000000",
		"checkOff": "#212830", "checkOn": "#316dca", "checkMark": "#f0f6fc",
		"switchOff": "#151b23", "switchOffBorder": "#3d444d", "switchOn": "#316dca", "knobOff": "#2a313c", "knobOn": "#cdd9e5", "knobBorder": "#3d444d",
		"track": "#3d444d", "thumbFill": "#d1d7e0", "thumbBorder": "#656c76",
		"progress": "#316dca", "progressTrack": "#3d444d",
		"scrollThumb": "#9198a180", "scrollThumbHot": "#9198a1",
		"tabLine": "#ec775c", "tabText2": "#d1d7e0",
		"segTrack": "#151b23", "segOn": "#2a313c", "segOnBorder": "#3d444d",
		"danger": "#e5534b", "success": "#57ab5a", "warning": "#c69026",
		"selection": "#316dcab3", "overlay": "#151b2366",
	}
	return []webSpec{
		{name: "primer", label: "Primer", era: "Primer", lineage: "GitHub", year: 2023, c: light, p: p, m: m,
			summary: "GitHub's Primer: 6px controls in #d1d9e0 hairlines, the green primary button, the coral underline under the current tab, rows marked with an accent bar."},
		{name: "primer-night", label: "Primer Dark", era: "Primer", lineage: "GitHub", year: 2023, dark: true, c: dark, p: p, m: m,
			summary: "GitHub's dark theme: Primer's shapes on #0d1117 with #3d444d hairlines and #1f6feb for focus and checks."},
		{name: "primer-dimmed", label: "Primer Dark Dimmed", era: "Primer", lineage: "GitHub", year: 2023, dark: true, c: dimmed, p: p, m: m,
			summary: "GitHub's dark dimmed theme: softer #212830 greys, muted green buttons and the #316dca accent."},
	}
}
