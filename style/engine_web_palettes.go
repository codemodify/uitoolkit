package style

// The editor palettes as web packs: each palette's colours placed in the UI
// roles its own style guide, docs, spec or reference port gives them
// (research §1.3), on a neutral web shape (6px controls, 8px popovers, a 2px
// focus band, underlined tabs, pill switches).

// paletteRoles are a palette's colours by UI role.
type paletteRoles struct {
	window, field, sidebar, chrome, popup string // surfaces
	raised, raisedHover, raisedPress      string // buttons and other raised controls
	border, divider                       string // input borders, separators
	text, text2, disabled                 string
	accent, onAccent, focus               string
	sel, hover, selection                 string // list selection, row hover, text selection
	scroll                                string
	danger, warning, success, link        string
	knobOff                               string
}

// paletteParams are the shared shapes of the palette packs.
var paletteParams = map[string]float32{
	"radius": 6, "overlayRadius": 8, "tipRadius": 6, "cardRadius": 8, "viewRadius": 6, "windowRadius": 10,
	"checkRadius": 4, "rowRadius": 6, "menuInset": 4,
	"focusStyle": webFocusInside, "focusWidth": 2,
	"primaryStyle": 0, "buttonWeight": 500,
	"tabStyle": webTabUnderline, "tabLine": 2, "tabWeight": 500, "tabBar": 1,
	"checkStyle": 1, "radioStyle": 1, "radioDot": 6, "trackH": 4, "progressH": 4,
	"menuHighlight": 0, "rowInset": 4, "sideInset": 6,
	"captionStyle": 1, "toolWash": 1, "seg": 1, "headerWeight": 600, "headerSep": 0, "accordion": 1,
	"menuShadow": 0.2, "menuShadowY": 6, "menuShadowBlur": 24,
	"tipShadow": 0.15, "tipShadowY": 2, "tipShadowBlur": 8,
	"dialogShadow": 0.3, "dialogShadowY": 12, "dialogShadowBlur": 48,
	"hoverFade": 120,
}

// paletteMetrics are the palette packs' sizes at 14px text (room for the
// check boxes' focus band round the 16px box).
var paletteMetrics = ChromeMetrics{
	FontSize: 14, Radius: 6, RadiusSmall: 6,
	ControlH: 32, FieldH: 32, ComboH: 32, Checkbox: 22, Radio: 22,
	MenuItemH: 30, MenuBarH: 32, TabH: 36, RowH: 30, TitleBar: 44, HeaderH: 30,
	ProgressH: 10, SliderH: 20, Thumb: 16, Scroll: 10, Pad: 12, FieldPad: 10,
	ToolBarH: 40, StatusBarH: 28, SpinnerW: 22, SwitchW: 36, SwitchH: 20,
	ViewFrame: 1, Border: 1, FocusWidth: 2,
}

// paletteSpec is a palette pack from its roles.
func paletteSpec(name, label, era string, year int, dark bool, summary string, r paletteRoles, p map[string]float32) webSpec {
	c := map[string]string{
		"window": r.window, "field": r.field, "sidebar": r.sidebar, "titleBar": r.chrome, "toolBar": r.chrome,
		"subPanel": r.sidebar, "card": r.window, "tabPane": r.window, "header": r.sidebar,
		"popup": r.popup, "popupBorder": r.divider, "dialog": r.window,
		"border0": r.divider, "border1": r.border, "border2": r.divider, "checkBorder": r.border,
		"text": r.text, "text2": r.text2, "textDis": r.disabled, "link": r.link, "headerText": r.text2,
		"accent": r.accent, "onAccent": r.onAccent, "focus": r.focus, "fieldFocus": r.focus,
		"btn": r.raised, "btnHover": r.raisedHover, "btnPress": r.raisedPress, "btnBorder": "#00000000", "btnText": r.text,
		"wash": r.hover, "sel": r.sel, "menuHover": r.raisedHover,
		"tip": r.popup, "tipText": r.text, "tipBorder": r.divider,
		"checkOff": r.field, "checkOn": r.accent, "checkMark": r.onAccent,
		"switchOff": r.raisedHover, "switchOn": r.accent, "knobOff": r.knobOff, "knobOn": r.onAccent,
		"track": r.raisedHover, "trackFill": r.accent, "thumbFill": r.accent,
		"progress": r.accent, "progressTrack": r.raised,
		"scrollThumb": r.scroll, "scrollThumbHot": r.text2,
		"tabLine": r.accent, "tabText": r.text, "tabText2": r.text2,
		"segTrack": r.raised, "segOn": r.window, "segOnText": r.text, "segOnBorder": r.divider,
		"danger": r.danger, "success": r.success, "warning": r.warning,
		"selection": r.selection,
	}
	if p == nil {
		p = paletteParams
	}
	return webSpec{name: name, label: label, era: era, lineage: era, year: year, dark: dark, summary: summary, c: c, p: p, m: paletteMetrics}
}

func paletteSpecs() []webSpec {
	var out []webSpec
	out = append(out, catppuccinSpecs()...)
	out = append(out, nordSpecs()...)
	out = append(out, draculaSpecs()...)
	out = append(out, tokyoNightSpecs()...)
	out = append(out, rosePineSpecs()...)
	out = append(out, editorPaletteSpecs()...)
	return out
}

// ---- Catppuccin ---------------------------------------------------------------------------------

// catppuccinFlavour is one flavour's 26 colours (catppuccin/palette 1.8).
type catppuccinFlavour struct {
	name, label, summary                                   string
	dark                                                   bool
	red, yellow, green, blue, lavender                     string
	text, subtext1, subtext0, overlay2, overlay1, overlay0 string
	surface2, surface1, surface0, base, mantle, crust      string
}

// catppuccinSpecs follow the style guide: Base for panes, Mantle and Crust
// for secondary panes (sidebars, bars), Surface 0–2 for raised controls and
// popovers, Overlay 0 for inactive borders and Lavender for the active one,
// Text and Subtext for labels, Overlay 2 at 25% for selections, Blue as the
// accent with Base on it; Red, Yellow and Green for errors, warnings and
// success.
func catppuccinSpecs() []webSpec {
	flavours := []catppuccinFlavour{
		{name: "catppuccin-latte", label: "Catppuccin Latte", dark: false,
			summary: "Catppuccin's light flavour: Base #eff1f5 panes, Mantle sidebars, Blue #1e66f5 accent, Lavender focus.",
			red:     "#d20f39", yellow: "#df8e1d", green: "#40a02b", blue: "#1e66f5", lavender: "#7287fd",
			text: "#4c4f69", subtext1: "#5c5f77", subtext0: "#6c6f85", overlay2: "#7c7f93", overlay1: "#8c8fa1", overlay0: "#9ca0b0",
			surface2: "#acb0be", surface1: "#bcc0cc", surface0: "#ccd0da", base: "#eff1f5", mantle: "#e6e9ef", crust: "#dce0e8"},
		{name: "catppuccin-frappe", label: "Catppuccin Frappé", dark: true,
			summary: "Catppuccin Frappé: the lightest dark flavour, #303446 Base with Surface popovers and Blue #8caaee.",
			red:     "#e78284", yellow: "#e5c890", green: "#a6d189", blue: "#8caaee", lavender: "#babbf1",
			text: "#c6d0f5", subtext1: "#b5bfe2", subtext0: "#a5adce", overlay2: "#949cbb", overlay1: "#838ba7", overlay0: "#737994",
			surface2: "#626880", surface1: "#51576d", surface0: "#414559", base: "#303446", mantle: "#292c3c", crust: "#232634"},
		{name: "catppuccin-macchiato", label: "Catppuccin Macchiato", dark: true,
			summary: "Catppuccin Macchiato: #24273a Base, deeper Mantle and Crust bars, Blue #8aadf4 with Lavender focus.",
			red:     "#ed8796", yellow: "#eed49f", green: "#a6da95", blue: "#8aadf4", lavender: "#b7bdf8",
			text: "#cad3f5", subtext1: "#b8c0e0", subtext0: "#a5adcb", overlay2: "#939ab7", overlay1: "#8087a2", overlay0: "#6e738d",
			surface2: "#5b6078", surface1: "#494d64", surface0: "#363a4f", base: "#24273a", mantle: "#1e2030", crust: "#181926"},
		{name: "catppuccin-mocha", label: "Catppuccin Mocha", dark: true,
			summary: "Catppuccin Mocha, the darkest flavour: #1e1e2e Base, #11111b Crust bars, Blue #89b4fa and Lavender #b4befe.",
			red:     "#f38ba8", yellow: "#f9e2af", green: "#a6e3a1", blue: "#89b4fa", lavender: "#b4befe",
			text: "#cdd6f4", subtext1: "#bac2de", subtext0: "#a6adc8", overlay2: "#9399b2", overlay1: "#7f849c", overlay0: "#6c7086",
			surface2: "#585b70", surface1: "#45475a", surface0: "#313244", base: "#1e1e2e", mantle: "#181825", crust: "#11111b"},
	}
	var out []webSpec
	for _, f := range flavours {
		out = append(out, paletteSpec(f.name, f.label, "Catppuccin", 2021, f.dark, f.summary, paletteRoles{
			window: f.base, field: f.base, sidebar: f.mantle, chrome: f.crust, popup: f.surface0,
			raised: f.surface0, raisedHover: f.surface1, raisedPress: f.surface2,
			border: f.overlay0, divider: f.surface1,
			text: f.text, text2: f.subtext1, disabled: f.overlay1,
			accent: f.blue, onAccent: f.base, focus: f.lavender,
			sel: f.overlay2 + "40", hover: f.overlay2 + "1f", selection: f.overlay2 + "4d",
			scroll: f.overlay0, danger: f.red, warning: f.yellow, success: f.green, link: f.blue, knobOff: f.subtext0,
		}, nil))
	}
	return out
}

// ---- Nord ---------------------------------------------------------------------------------------

// nordSpecs follow Nord's docs: Polar Night nord0 for the dark background,
// nord1 for raised UI (bars, panels, popups, buttons, fields), nord2 for
// selections, nord3 for disabled UI, nord4 for text, the Frost nord8 as the
// accent; Snow Storm for the light scheme (nord6 background, nord4 raised UI
// and borders, nord5 selections, nord0 text) with the deeper Frost nord10 as
// its accent (nord8 is too pale there; an inference).
func nordSpecs() []webSpec {
	return []webSpec{
		paletteSpec("nord", "Nord Light", "Nord", 2016, false,
			"Nord's Snow Storm scheme: #eceff4 backgrounds, #d8dee9 raised UI and borders, Polar Night text, the Frost #5e81ac accent.",
			paletteRoles{
				window: "#eceff4", field: "#eceff4", sidebar: "#e5e9f0", chrome: "#e5e9f0", popup: "#eceff4",
				raised: "#e5e9f0", raisedHover: "#d8dee9", raisedPress: "#cdd4e0",
				border: "#b9c2d1", divider: "#d8dee9",
				text: "#2e3440", text2: "#4c566a", disabled: "#9aa3b2",
				accent: "#5e81ac", onAccent: "#eceff4", focus: "#5e81ac",
				sel: "#d8dee9", hover: "#2e34400f", selection: "#81a1c14d",
				scroll: "#4c566a80", danger: "#bf616a", warning: "#d08770", success: "#a3be8c", link: "#5e81ac", knobOff: "#4c566a",
			}, nil),
		paletteSpec("nord-night", "Nord", "Nord", 2016, true,
			"Nord's Polar Night: #2e3440 backgrounds, #3b4252 raised UI, #434c5e selections, the Frost #88c0d0 accent with dark text on it.",
			paletteRoles{
				window: "#2e3440", field: "#3b4252", sidebar: "#3b4252", chrome: "#3b4252", popup: "#3b4252",
				raised: "#3b4252", raisedHover: "#434c5e", raisedPress: "#4c566a",
				border: "#4c566a", divider: "#434c5e",
				text: "#d8dee9", text2: "#aeb5c1", disabled: "#616e88",
				accent: "#88c0d0", onAccent: "#2e3440", focus: "#88c0d0",
				sel: "#434c5e", hover: "#d8dee914", selection: "#434c5e",
				scroll: "#4c566a", danger: "#bf616a", warning: "#ebcb8b", success: "#a3be8c", link: "#88c0d0", knobOff: "#d8dee9",
			}, nil),
	}
}

// ---- Dracula and Alucard ------------------------------------------------------------------------

// draculaSpecs follow the Dracula spec: Background for panes, Background
// Dark for sidebars and Darker for bars, the Floating colour for menus and
// popovers, Current Line's solid fallback for subtle borders, Selection,
// Comment for disabled text, Purple as the accent, and the spec's
// functional colours (the focus ring, error, warning, success, links).
func draculaSpecs() []webSpec {
	return []webSpec{
		paletteSpec("alucard", "Alucard", "Dracula", 2023, false,
			"Dracula's light variant: cream #fffbeb panes, #efeddc popovers, the Purple #644ac9 accent and the #815cd6 focus ring.",
			paletteRoles{
				window: "#fffbeb", field: "#fffbeb", sidebar: "#ceccc0", chrome: "#bcbab3", popup: "#efeddc",
				raised: "#efeddc", raisedHover: "#e2deca", raisedPress: "#deddcf",
				border: "#6c664b", divider: "#e2deca",
				text: "#1f1f1f", text2: "#4a4633", disabled: "#6c664b",
				accent: "#644ac9", onAccent: "#fffbeb", focus: "#815cd6",
				sel: "#cfcfde", hover: "#1f1f1f0d", selection: "#cfcfde",
				scroll: "#6c664b80", danger: "#de5735", warning: "#a39514", success: "#089108", link: "#0081d6", knobOff: "#6c664b",
			}, nil),
		paletteSpec("dracula", "Dracula", "Dracula", 2013, true,
			"Dracula: #282a36 panes, darker sidebars and bars, #343746 popovers, the Purple #bd93f9 accent and #815cd6 focus rings.",
			paletteRoles{
				window: "#282a36", field: "#282a36", sidebar: "#21222c", chrome: "#191a21", popup: "#343746",
				raised: "#343746", raisedHover: "#424450", raisedPress: "#44475a",
				border: "#6272a4", divider: "#353747",
				text: "#f8f8f2", text2: "#c5c6c8", disabled: "#6272a4",
				accent: "#bd93f9", onAccent: "#282a36", focus: "#815cd6",
				sel: "#44475a", hover: "#f8f8f20d", selection: "#44475a",
				scroll: "#6272a4", danger: "#de5735", warning: "#a39514", success: "#089108", link: "#0081d6", knobOff: "#f8f8f2",
			}, nil),
	}
}

// ---- Tokyo Night --------------------------------------------------------------------------------

// tokyoNightSpecs take the UI keys of the Tokyo Night VS Code theme (Night
// and Storm: editor and chrome colours, the #3d59a1 accent of buttons and the
// active tab, list selection and hover, input and focus borders) and, for
// Day, the palette of tokyonight.nvim's Day style.
func tokyoNightSpecs() []webSpec {
	return []webSpec{
		paletteSpec("tokyonight-day", "Tokyo Night Day", "Tokyo Night", 2021, false,
			"Tokyo Night Day: #e1e2e7 panes with #d0d5e3 sidebars and popups, blue #3760bf text, the #2e7de9 accent.",
			paletteRoles{
				window: "#e1e2e7", field: "#e1e2e7", sidebar: "#d0d5e3", chrome: "#d0d5e3", popup: "#d0d5e3",
				raised: "#d0d5e3", raisedHover: "#c4c8da", raisedPress: "#c1c9df",
				border: "#b4b5b9", divider: "#c1c9df",
				text: "#3760bf", text2: "#6172b0", disabled: "#8990b3",
				accent: "#2e7de9", onAccent: "#ffffff", focus: "#4094a3",
				sel: "#b7c1e3", hover: "#3760bf12", selection: "#b7c1e3",
				scroll: "#a8aecb", danger: "#c64343", warning: "#8c6c3e", success: "#587539", link: "#2e7de9", knobOff: "#6172b0",
			}, nil),
		paletteSpec("tokyonight-storm", "Tokyo Night Storm", "Tokyo Night", 2020, true,
			"Tokyo Night Storm: #24283b panes in #1f2335 chrome, #2c324a selections, the #3d59a1 accent.",
			paletteRoles{
				window: "#24283b", field: "#1b1e2e", sidebar: "#1f2335", chrome: "#1f2335", popup: "#1f2335",
				raised: "#292e42", raisedHover: "#2c324a", raisedPress: "#343a55",
				border: "#282e44", divider: "#1b1e2e",
				text: "#a9b1d6", text2: "#8089b3", disabled: "#545c7e",
				accent: "#3d59a1", onAccent: "#ffffff", focus: "#545c7e",
				sel: "#2c324a", hover: "#1b1e2e", selection: "#6f7bb635",
				scroll: "#545c7e80", danger: "#db4b4b", warning: "#e0af68", success: "#9ece6a", link: "#668ac4", knobOff: "#8089b3",
			}, nil),
		paletteSpec("tokyonight", "Tokyo Night", "Tokyo Night", 2020, true,
			"Tokyo Night: #1a1b26 panes in #16161e chrome with near-black borders, #202330 selections, the #3d59a1 accent.",
			paletteRoles{
				window: "#1a1b26", field: "#14141b", sidebar: "#16161e", chrome: "#16161e", popup: "#16161e",
				raised: "#1f2030", raisedHover: "#202330", raisedPress: "#292e42",
				border: "#0f0f14", divider: "#101014",
				text: "#a9b1d6", text2: "#787c99", disabled: "#545c7e",
				accent: "#3d59a1", onAccent: "#ffffff", focus: "#545c7e",
				sel: "#202330", hover: "#13131a", selection: "#515c7e40",
				scroll: "#545c7e80", danger: "#db4b4b", warning: "#e0af68", success: "#9ece6a", link: "#6183bb", knobOff: "#787c99",
			}, nil),
	}
}

// ---- Rosé Pine ----------------------------------------------------------------------------------

// rosePineSpecs follow rosepinetheme.com's roles: base for windows and side
// bars, surface for inputs, bars and popups, overlay for hovered and selected
// items and dialogs, highlight med for selections, highlight high for
// borders, text / subtle / muted for labels; love, gold, foam and pine for
// errors, warnings, information and success; iris (the links' colour) as the
// accent — a choice, the palette names none.
func rosePineSpecs() []webSpec {
	type rp struct{ name, label, summary, base, surface, overlay, muted, subtle, text, love, gold, pine, iris, hlLow, hlMed, hlHigh string }
	var out []webSpec
	for _, f := range []rp{
		{"rosepine-dawn", "Rosé Pine Dawn", "Rosé Pine's light variant: #faf4ed base, #fffaf3 surfaces, highlight-med selections, iris #907aa9.",
			"#faf4ed", "#fffaf3", "#f2e9e1", "#9893a5", "#797593", "#464261", "#b4637a", "#ea9d34", "#286983", "#907aa9", "#f4ede8", "#dfdad9", "#cecacd"},
		{"rosepine-moon", "Rosé Pine Moon", "Rosé Pine Moon: #232136 base, #2a273f surfaces, #393552 overlays, iris #c4a7e7.",
			"#232136", "#2a273f", "#393552", "#6e6a86", "#908caa", "#e0def4", "#eb6f92", "#f6c177", "#3e8fb0", "#c4a7e7", "#2a283e", "#44415a", "#56526e"},
		{"rosepine", "Rosé Pine", "Rosé Pine: #191724 base, #1f1d2e surfaces, #26233a overlays for hover and selection, iris #c4a7e7.",
			"#191724", "#1f1d2e", "#26233a", "#6e6a86", "#908caa", "#e0def4", "#eb6f92", "#f6c177", "#31748f", "#c4a7e7", "#21202e", "#403d52", "#524f67"},
	} {
		dark := f.name != "rosepine-dawn"
		out = append(out, paletteSpec(f.name, f.label, "Rosé Pine", 2021, dark, f.summary, paletteRoles{
			window: f.base, field: f.surface, sidebar: f.base, chrome: f.surface, popup: f.surface,
			raised: f.surface, raisedHover: f.overlay, raisedPress: f.hlMed,
			border: f.hlHigh, divider: f.hlMed,
			text: f.text, text2: f.subtle, disabled: f.muted,
			accent: f.iris, onAccent: f.base, focus: f.iris,
			sel: f.hlMed, hover: f.hlLow, selection: f.hlMed,
			scroll: f.hlHigh, danger: f.love, warning: f.gold, success: f.pine, link: f.iris, knobOff: f.subtle,
		}, nil))
	}
	return out
}
