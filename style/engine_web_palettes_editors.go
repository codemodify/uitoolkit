package style

// The older editor palettes as web packs (research §1.3), each palette's
// colours in the UI roles its own scheme gives them where it gives any:
// Gruvbox and Solarized on the palettes' neutral web shape, Atom's One
// themes on Atom's 3px corners with its boxed editor tabs.

// pmenuParams are the palettes' shape with the menu's current item in a
// colour of its own (Vim's PmenuSel), its shortcut in the item's text.
var pmenuParams = webWith(paletteParams, map[string]float32{"menuHighlight": 1})

func editorPaletteSpecs() []webSpec {
	var out []webSpec
	out = append(out, gruvboxSpecs()...)
	out = append(out, solarizedSpecs()...)
	out = append(out, oneSpecs()...)
	return out
}

// ---- Gruvbox ------------------------------------------------------------------------------------

// gruvboxSpecs follow gruvbox.vim's own groups (morhetz/gruvbox): bg0 for
// panes, bg1 for raised controls and bars, bg2 for popup menus (Pmenu),
// bg3 for splits, separators and the visual selection, fg1 for text and
// fg4 for secondary text; the menu's current item in blue (PmenuSel), blue
// the accent; the bright accents on the dark mode, the faded ones on the
// light. Fields sit in the hard-contrast bg0 (an inference).
func gruvboxSpecs() []webSpec {
	type gv struct {
		bg0h, bg0, bg1, bg2, bg3, bg4, fg1, fg4, gray string
		red, green, yellow, blue                      string
	}
	spec := func(name, label, summary string, dark bool, g gv) webSpec {
		s := paletteSpec(name, label, "Gruvbox", 2012, dark, summary, paletteRoles{
			window: g.bg0, field: g.bg0h, sidebar: g.bg0, chrome: g.bg1, popup: g.bg2,
			raised: g.bg1, raisedHover: g.bg2, raisedPress: g.bg3,
			border: g.bg4, divider: g.bg3,
			text: g.fg1, text2: g.fg4, disabled: g.gray,
			accent: g.blue, onAccent: g.bg0, focus: g.blue,
			sel: g.bg2, hover: g.fg1 + "14", selection: g.bg3,
			scroll: g.bg3, danger: g.red, warning: g.yellow, success: g.green, link: g.blue, knobOff: g.fg4,
		}, pmenuParams)
		// PmenuSel: the current menu item on blue.
		s.c["menuHover"], s.c["menuText"] = g.blue, g.bg0
		return s
	}
	return []webSpec{
		spec("gruvbox", "Gruvbox Dark", "Gruvbox's dark mode: warm #282828 panes, #ebdbb2 cream text, #504945 menus with the current item on blue #83a598.", true, gv{
			bg0h: "#1d2021", bg0: "#282828", bg1: "#3c3836", bg2: "#504945", bg3: "#665c54", bg4: "#7c6f64", fg1: "#ebdbb2", fg4: "#a89984", gray: "#928374",
			red: "#fb4934", green: "#b8bb26", yellow: "#fabd2f", blue: "#83a598"}),
		spec("gruvbox-light", "Gruvbox Light", "Gruvbox's light mode: #fbf1c7 panes, #3c3836 text, #d5c4a1 menus, the faded blue #076678 as the accent.", false, gv{
			bg0h: "#f9f5d7", bg0: "#fbf1c7", bg1: "#ebdbb2", bg2: "#d5c4a1", bg3: "#bdae93", bg4: "#a89984", fg1: "#3c3836", fg4: "#7c6f64", gray: "#928374",
			red: "#9d0006", green: "#79740e", yellow: "#b57614", blue: "#076678"}),
	}
}

// ---- Solarized ----------------------------------------------------------------------------------

// solarizedSpecs follow Solarized's own roles (altercation/solarized):
// base03 (base3) backgrounds, base02 (base2) background highlights for
// raised surfaces and selections, base1 (base01) emphasised content as the
// UI's text for its contrast, base0 (base00) body text as the secondary
// text, base01 (base1) comments for disabled text and, at half strength,
// separators; blue as the accent. The menu's current item is base2 on
// base01 (base02 on base1), as solarized.vim's PmenuSel shows it.
func solarizedSpecs() []webSpec {
	const (
		base03, base02, base01, base00 = "#002b36", "#073642", "#586e75", "#657b83"
		base0, base1, base2, base3     = "#839496", "#93a1a1", "#eee8d5", "#fdf6e3"
		yellow, red, blue, green       = "#b58900", "#dc322f", "#268bd2", "#859900"
	)
	dark := paletteSpec("solarized", "Solarized Dark", "Solarized", 2011, true,
		"Solarized Dark: base03 #002b36 panes with base02 highlights, base1 text, blue #268bd2 as the accent.",
		paletteRoles{
			window: base03, field: base03, sidebar: base03, chrome: base02, popup: base02,
			raised: base02, raisedHover: base01 + "80", raisedPress: base01,
			border: base01, divider: base01 + "80",
			text: base1, text2: base0, disabled: base01,
			accent: blue, onAccent: base3, focus: blue,
			sel: base02, hover: base1 + "14", selection: base01,
			scroll: base01, danger: red, warning: yellow, success: green, link: blue, knobOff: base0,
		}, pmenuParams)
	dark.c["menuHover"], dark.c["menuText"] = base01, base2
	light := paletteSpec("solarized-light", "Solarized Light", "Solarized", 2011, false,
		"Solarized Light: base3 #fdf6e3 panes with base2 highlights, base01 text, the same blue #268bd2 as the accent.",
		paletteRoles{
			window: base3, field: base3, sidebar: base3, chrome: base2, popup: base2,
			raised: base2, raisedHover: base1 + "80", raisedPress: base1,
			border: base1, divider: base1 + "80",
			text: base01, text2: base00, disabled: base1,
			accent: blue, onAccent: base3, focus: blue,
			sel: base2, hover: base01 + "14", selection: base1,
			scroll: base1, danger: red, warning: yellow, success: green, link: blue, knobOff: base00,
		}, pmenuParams)
	light.c["menuHover"], light.c["menuText"] = base1, base02
	return []webSpec{dark, light}
}

// ---- One Dark and One Light ---------------------------------------------------------------------

// oneParams are Atom's UI metrics on the palettes' shape: 3px corners,
// boxed editor tabs, the tree's full-width selection.
var oneParams = webWith(paletteParams, map[string]float32{
	"radius": 3, "fieldRadius": 3, "comboRadius": 3, "overlayRadius": 3, "tipRadius": 3, "cardRadius": 3,
	"viewRadius": 3, "windowRadius": 3, "checkRadius": 3, "menuRowRadius": 3,
	"rowRadius": 0, "sideRadius": 0, "rowInset": 0, "sideInset": 0,
	"tabStyle": webTabEditor, "tabLine": 1,
})

// oneSpecs are Atom's One Dark and One Light UI themes (atom/atom, the
// themes' LESS in HSL computed to hex): the tree, tab bar and status bar on
// the app colour round the pane's, raised buttons, the accent for focus and
// the default button, list selections in the button's hover colour with the
// highlight text; secondary and disabled text and the status colours are
// the syntax themes' greys and hues.
func oneSpecs() []webSpec {
	dark := paletteSpec("onedark", "One Dark", "One", 2014, true,
		"Atom's One Dark: #282c34 panes in #21252b chrome, #9da5b4 text, 3px controls, boxed tabs, the #4d78cc accent.",
		paletteRoles{
			window: "#282c34", field: "#1b1d23", sidebar: "#21252b", chrome: "#21252b", popup: "#21252b",
			raised: "#353b45", raisedHover: "#3a3f4b", raisedPress: "#3e4451",
			border: "#181a1f", divider: "#181a1f",
			text: "#9da5b4", text2: "#828997", disabled: "#5c6370",
			accent: "#4d78cc", onAccent: "#ffffff", focus: "#568af2",
			sel: "#3a3f4b", hover: "#9da5b414", selection: "#3e4451",
			scroll: "#9da5b440", danger: "#ff6347", warning: "#e2c08d", success: "#73c990", link: "#568af2", knobOff: "#9da5b4",
		}, oneParams)
	dark.lineage = "Atom"
	dark.c["menuText"], dark.c["tabText"], dark.c["tabLine"] = "#d7dae0", "#d7dae0", "#00000000"
	light := paletteSpec("onelight", "One Light", "One", 2014, false,
		"Atom's One Light: #fafafa panes in #eaeaeb chrome, #424243 text, 3px controls, boxed tabs, the #556de8 accent.",
		paletteRoles{
			window: "#fafafa", field: "#ffffff", sidebar: "#eaeaeb", chrome: "#eaeaeb", popup: "#eaeaeb",
			raised: "#ffffff", raisedHover: "#f2f2f2", raisedPress: "#e5e5e6",
			border: "#dbdbdc", divider: "#dbdbdc",
			text: "#424243", text2: "#696c77", disabled: "#a0a1a7",
			accent: "#556de8", onAccent: "#ffffff", focus: "#556de8",
			sel: "#dbdbdc", hover: "#42424312", selection: "#e5e5e6",
			scroll: "#42424340", danger: "#e45649", warning: "#c18401", success: "#50a14f", link: "#556de8", knobOff: "#696c77",
		}, oneParams)
	light.lineage = "Atom"
	light.c["tabLine"] = "#00000000"
	light.c["btnBorder"] = "#dbdbdc"
	return []webSpec{dark, light}
}
