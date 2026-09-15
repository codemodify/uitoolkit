package style

// shadcn/ui, Vercel's Geist and Linear as web packs (see engine_web.go and
// engine_web_packs.go).

// shadcnSpecs are shadcn/ui's classic new-york style (registry v4, 2025) in
// the neutral base colour: the OKLCH tokens converted to sRGB.
func shadcnSpecs() []webSpec {
	p := map[string]float32{
		// --radius 10px: rounded-md 8 for controls, lg 10 for dialogs and the
		// tab track, sm 6 for menu rows, xl 14 for cards; 4px check boxes.
		"radius": 8, "overlayRadius": 8, "tipRadius": 8, "cardRadius": 14, "viewRadius": 10, "windowRadius": 10,
		"checkRadius": 4, "rowRadius": 6, "menuInset": 4, "restShadow": 0.05,
		// focus-visible: border-ring and a 3px ring of the ring colour at 50%
		// outside the control.
		"focusStyle": webFocusOutside, "focusWidth": 3, "focusGap": 0, "focusAlpha": 0.5,
		"primaryStyle": 1, "buttonWeight": 500,
		// Tabs are a segmented TabsList: a muted track, the active trigger a
		// raised card.
		"tabStyle": webTabSegmented, "tabWeight": 500,
		"checkStyle": 1, "checkSize": 16, "radioStyle": 0, "radioDot": 8,
		"knob": 16, "trackH": 6, "thumbStyle": 1, "progressH": 8,
		"menuHighlight": 0, "rowInset": 8, "sideInset": 8, "listHover": 0.5, "sideHover": 0.5, "tableHover": 0.5,
		"captionStyle": 1, "captionLine": 0, "toolWash": 1, "seg": 2,
		"headerWeight": 500, "headerSep": 0, "accordion": 1,
		// shadow-md under menus, shadow-lg round dialogs; tips flat.
		"menuShadow": 0.1, "menuShadowY": 4, "menuShadowBlur": 12, "menuShadowSpread": -1,
		"tipShadow": 0, "dialogShadow": 0.1, "dialogShadowY": 10, "dialogShadowBlur": 30, "dialogShadowSpread": -3,
		"tipSize": 12, "hoverFade": 150,
	}
	// Controls are 36px (h-9) inside the 3px ring's room.
	m := ChromeMetrics{
		FontSize: 14, Radius: 8, RadiusSmall: 6,
		ControlH: 42, FieldH: 42, ComboH: 42, Checkbox: 22, Radio: 22,
		MenuItemH: 32, MenuBarH: 36, TabH: 36, RowH: 32, TitleBar: 52, HeaderH: 40,
		ProgressH: 12, SliderH: 24, Thumb: 16, Scroll: 10, Pad: 16, FieldPad: 12,
		ToolBarH: 44, StatusBarH: 28, SpinnerW: 24, SwitchW: 32, SwitchH: 18,
		ViewFrame: 1, Border: 1, FocusWidth: 3,
	}
	light := map[string]string{
		"window": "#ffffff", "field": "#ffffff", "sidebar": "#fafafa", "subPanel": "#fafafa", "titleBar": "#ffffff",
		"toolBar": "#ffffff", "card": "#ffffff", "tabPane": "#ffffff", "header": "#ffffff",
		"popup": "#ffffff", "popupBorder": "#e5e5e5", "dialog": "#ffffff",
		"border0": "#e5e5e5", "border1": "#e5e5e5", "border2": "#e5e5e5", "checkBorder": "#e5e5e5",
		"text": "#0a0a0a", "text2": "#737373", "textDis": "#858585", "link": "#171717", "headerText": "#0a0a0a",
		"accent": "#171717", "onAccent": "#fafafa", "focus": "#a1a1a1", "fieldFocus": "#a1a1a1",
		"btn": "#ffffff", "btnHover": "#f5f5f5", "btnPress": "#f5f5f5", "btnBorder": "#e5e5e5", "btnText": "#0a0a0a",
		"primary": "#171717", "primaryHover": "#2e2e2e", "primaryPress": "#454545", "onPrimary": "#fafafa",
		"wash": "#f5f5f5", "sel": "#f5f5f5", "menuHover": "#f5f5f5", "menuText": "#171717",
		"tip": "#0a0a0a", "tipText": "#ffffff", "tipBorder": "#00000000",
		"checkOff": "#ffffff", "checkOn": "#171717", "checkMark": "#fafafa",
		"switchOff": "#e5e5e5", "switchOn": "#171717", "knobOff": "#ffffff", "knobOn": "#ffffff",
		"track": "#f5f5f5", "trackFill": "#171717", "thumbFill": "#ffffff", "thumbBorder": "#171717",
		"progress": "#171717", "progressTrack": "#d1d1d1",
		"scrollThumb": "#e5e5e5", "scrollThumbHot": "#a1a1a1",
		"tabText2": "#0a0a0a", "segTrack": "#f5f5f5", "segOn": "#ffffff", "segOnText": "#0a0a0a",
		"danger": "#e7000b", "success": "#16a34a", "warning": "#f59e0b",
		"selection": "#171717", "selectionText": "#fafafa", "overlay": "#00000080",
	}
	dark := map[string]string{
		"window": "#0a0a0a", "field": "#141414", "sidebar": "#171717", "subPanel": "#171717", "titleBar": "#0a0a0a",
		"toolBar": "#0a0a0a", "card": "#171717", "tabPane": "#0a0a0a", "header": "#0a0a0a",
		"popup": "#171717", "popupBorder": "#ffffff1a", "dialog": "#171717",
		"border0": "#ffffff1a", "border1": "#ffffff26", "border2": "#ffffff1a", "checkBorder": "#ffffff26",
		"text": "#fafafa", "text2": "#a1a1a1", "textDis": "#828282", "link": "#e5e5e5", "headerText": "#fafafa",
		"accent": "#e5e5e5", "onAccent": "#171717", "focus": "#737373", "fieldFocus": "#737373",
		"btn": "#141414", "btnHover": "#1d1d1d", "btnPress": "#1d1d1d", "btnBorder": "#ffffff26", "btnText": "#fafafa",
		"primary": "#e5e5e5", "primaryHover": "#cfcfcf", "primaryPress": "#b9b9b9", "onPrimary": "#171717",
		"wash": "#262626", "sel": "#262626", "menuHover": "#262626", "menuText": "#fafafa",
		"tip": "#fafafa", "tipText": "#0a0a0a", "tipBorder": "#00000000",
		"checkOff": "#141414", "checkOn": "#e5e5e5", "checkMark": "#171717",
		"switchOff": "#272727", "switchOn": "#e5e5e5", "knobOff": "#fafafa", "knobOn": "#171717",
		"track": "#262626", "trackFill": "#e5e5e5", "thumbFill": "#ffffff", "thumbBorder": "#e5e5e5",
		"progress": "#e5e5e5", "progressTrack": "#353535",
		"scrollThumb": "#ffffff1a", "scrollThumbHot": "#737373",
		"tabText2": "#a1a1a1", "segTrack": "#262626", "segOn": "#2e2e2e", "segOnBorder": "#ffffff26", "segOnText": "#fafafa",
		"danger": "#ff6467", "success": "#22c55e", "warning": "#f59e0b",
		"selection": "#e5e5e5", "selectionText": "#171717", "overlay": "#00000080",
	}
	return []webSpec{
		{name: "shadcn", label: "shadcn/ui", era: "shadcn/ui", lineage: "shadcn/ui", year: 2025, c: light, p: p, m: m,
			summary: "shadcn/ui's new-york style in neutral: near-black primary buttons, #e5e5e5 hairlines, a 3px grey focus ring outside, segmented tabs."},
		{name: "shadcn-night", label: "shadcn/ui Dark", era: "shadcn/ui", lineage: "shadcn/ui", year: 2025, dark: true, c: dark, p: p, m: m,
			summary: "shadcn/ui's dark neutral theme: #0a0a0a with white-alpha hairlines and near-white primary buttons."},
	}
}

// geistSpecs are Vercel's Geist design system (2023; the colour page's raw
// sRGB values of 2026).
func geistSpecs() []webSpec {
	p := map[string]float32{
		// Materials: 6px controls, 12px menus, modals and cards.
		"radius": 6, "overlayRadius": 12, "tipRadius": 6, "cardRadius": 12, "viewRadius": 6, "windowRadius": 12,
		"checkRadius": 4, "rowRadius": 6, "menuInset": 4,
		// Focus: a 2px gap in the page colour, then a 2px ring; a focused
		// input darkens its ring and takes a 3px wash beyond it instead.
		"focusStyle": webFocusOutside, "focusWidth": 2, "focusGap": 2,
		"primaryStyle": 1, "buttonWeight": 500,
		"tabStyle": webTabUnderline, "tabLine": 2, "tabFit": 1, "tabWeight": 400, "tabSelWeight": 500, "tabBar": 1,
		"checkStyle": 1, "radioStyle": 1, "radioDot": 6, "knob": 10, "trackH": 4, "thumbStyle": 1, "progressH": 6,
		"menuHighlight": 0, "rowInset": 8, "sideInset": 8,
		"captionStyle": 1, "toolWash": 1, "seg": 1, "headerWeight": 500, "headerSep": 0, "accordion": 1,
		// Rings plus soft drops: the menu's 0 16px 24px -8px, the tool
		// tip's small shadow, the modal's larger one.
		"menuShadow": 0.06, "menuShadowY": 16, "menuShadowBlur": 48, "menuShadowSpread": -8,
		"tipShadow": 0.08, "tipShadowY": 2, "tipShadowBlur": 8,
		"dialogShadow": 0.1, "dialogShadowY": 24, "dialogShadowBlur": 64, "dialogShadowSpread": -8,
		"hoverFade": 150,
	}
	// Medium (36px) controls inside the 4px focus room; 36px menu rows.
	m := ChromeMetrics{
		FontSize: 14, Radius: 6, RadiusSmall: 6,
		ControlH: 44, FieldH: 44, ComboH: 44, Checkbox: 24, Radio: 24,
		MenuItemH: 36, MenuBarH: 36, TabH: 40, RowH: 36, TitleBar: 52, HeaderH: 40,
		ProgressH: 10, SliderH: 24, Thumb: 16, Scroll: 10, Pad: 16, FieldPad: 12,
		ToolBarH: 48, StatusBarH: 28, SpinnerW: 24, SwitchW: 28, SwitchH: 14,
		ViewFrame: 1, Border: 1, FocusWidth: 2,
	}
	light := map[string]string{
		"window": "#ffffff", "field": "#ffffff", "sidebar": "#fafafa", "subPanel": "#fafafa", "titleBar": "#ffffff",
		"toolBar": "#ffffff", "card": "#ffffff", "tabPane": "#ffffff", "header": "#fafafa",
		"popup": "#ffffff", "popupBorder": "#00000014", "dialog": "#ffffff",
		"border0": "#eaeaea", "border1": "#00000014", "border2": "#eaeaea", "checkBorder": "#00000057",
		"text": "#171717", "text2": "#4d4d4d", "textDis": "#8f8f8f", "link": "#0067d6", "headerText": "#4d4d4d",
		"accent": "#0070f3", "onAccent": "#ffffff", "focus": "#0070f3", "fieldFocus": "#00000057", "fieldRing": "#00000029",
		"btn": "#ffffff", "btnHover": "#f2f2f2", "btnPress": "#ebebeb", "btnBorder": "#eaeaea", "btnText": "#171717",
		"primary": "#171717", "primaryHover": "#383838", "primaryPress": "#4d4d4d", "onPrimary": "#ffffff",
		"wash": "#0000000d", "sel": "#00000014", "menuHover": "#0000000d",
		"tip": "#171717", "tipText": "#ffffff", "tipBorder": "#00000000",
		"checkOff": "#ffffff", "checkOn": "#171717", "checkMark": "#ffffff",
		"switchOff": "#ebebeb", "switchOffBorder": "#00000014", "switchOn": "#0070f3", "knobOff": "#ffffff", "knobOn": "#ffffff",
		"track": "#ebebeb", "trackFill": "#171717", "thumbFill": "#ffffff", "thumbBorder": "#00000057",
		"progress": "#171717", "progressTrack": "#ebebeb",
		"scrollThumb": "#00000033", "scrollThumbHot": "#00000057",
		"tabLine": "#171717", "tabText": "#171717", "tabText2": "#4d4d4d",
		"segTrack": "#f2f2f2", "segOn": "#ffffff", "segOnBorder": "#00000014",
		"danger": "#ca2a30", "success": "#297c3b", "warning": "#a35200",
		"selection": "#0070f333", "overlay": "#00000066",
	}
	dark := map[string]string{
		"window": "#0a0a0a", "field": "#0a0a0a", "sidebar": "#000000", "subPanel": "#000000", "titleBar": "#0a0a0a",
		"toolBar": "#0a0a0a", "card": "#0a0a0a", "tabPane": "#0a0a0a", "header": "#000000",
		"popup": "#0a0a0a", "popupBorder": "#ffffff25", "dialog": "#0a0a0a",
		"border0": "#2e2e2e", "border1": "#ffffff25", "border2": "#2e2e2e", "checkBorder": "#ffffff81",
		"text": "#ededed", "text2": "#a0a0a0", "textDis": "#8f8f8f", "link": "#52a9ff", "headerText": "#a0a0a0",
		"accent": "#0070f3", "onAccent": "#ffffff", "focus": "#52a9ff", "fieldFocus": "#ffffff81", "fieldRing": "#ffffff3d",
		"btn": "#0a0a0a", "btnHover": "#1a1a1a", "btnPress": "#1f1f1f", "btnBorder": "#2e2e2e", "btnText": "#ededed",
		"primary": "#ededed", "primaryHover": "#cccccc", "primaryPress": "#a0a0a0", "onPrimary": "#0a0a0a",
		"wash": "#ffffff0f", "sel": "#ffffff1a", "menuHover": "#ffffff0f",
		"tip": "#ededed", "tipText": "#0a0a0a", "tipBorder": "#00000000",
		"checkOff": "#0a0a0a", "checkOn": "#ededed", "checkMark": "#0a0a0a",
		"switchOff": "#1f1f1f", "switchOffBorder": "#ffffff25", "switchOn": "#0070f3", "knobOff": "#ededed", "knobOn": "#ffffff",
		"track": "#2e2e2e", "trackFill": "#ededed", "thumbFill": "#0a0a0a", "thumbBorder": "#ffffff81",
		"progress": "#ededed", "progressTrack": "#2e2e2e",
		"scrollThumb": "#ffffff33", "scrollThumbHot": "#ffffff81",
		"tabLine": "#ededed", "tabText": "#ededed", "tabText2": "#a0a0a0",
		"segTrack": "#1a1a1a", "segOn": "#0a0a0a", "segOnBorder": "#ffffff25",
		"danger": "#ff6369", "success": "#63c174", "warning": "#f1a10d",
		"selection": "#0070f366", "overlay": "#00000099",
	}
	return []webSpec{
		{name: "geist", label: "Geist", era: "Geist", lineage: "Vercel", year: 2023, c: light, p: p, m: m,
			summary: "Vercel's Geist: gray-1000 primary buttons, alpha-ring hairlines, a 2px blue ring beyond a 2px gap, 2px underlined tabs."},
		{name: "geist-night", label: "Geist Dark", era: "Geist", lineage: "Vercel", year: 2023, dark: true, c: dark, p: p, m: m,
			summary: "Geist's dark theme: #0a0a0a on black sidebars, near-white primary buttons and the #52a9ff focus ring."},
	}
}

// linearSpecs are Linear (2026): its app shell colours and, where the
// generator's output is not published, the text and border shades its site
// serves.
func linearSpecs() []webSpec {
	p := map[string]float32{
		// 4px controls, 8px inputs and menus; compact rounded tabs.
		"radius": 4, "fieldRadius": 8, "comboRadius": 6, "overlayRadius": 8, "tipRadius": 6, "cardRadius": 8,
		"viewRadius": 8, "windowRadius": 12, "checkRadius": 4, "rowRadius": 6, "menuInset": 4,
		// A 1px focus ring.
		"focusStyle": webFocusInside, "focusWidth": 1,
		"primaryStyle": 0, "buttonWeight": 500,
		"tabStyle": webTabSegmented, "tabWeight": 500,
		"checkStyle": 1, "radioStyle": 1, "radioDot": 6, "knob": 14, "trackH": 4, "progressH": 4,
		"menuHighlight": 0, "rowInset": 6, "sideInset": 8,
		// 6px scroll thumbs that widen to 10.
		"scrollIdle": 6, "scrollInset": 2,
		"captionStyle": 1, "toolWash": 1, "seg": 1, "headerWeight": 500, "headerSep": 0, "accordion": 1,
		"menuShadow": 0.28, "menuShadowY": 6, "menuShadowBlur": 36,
		"tipShadow": 0.18, "tipShadowY": 3, "tipShadowBlur": 12,
		"dialogShadow": 0.3, "dialogShadowY": 9, "dialogShadowBlur": 96,
		"hoverFade": 100,
	}
	m := ChromeMetrics{
		FontSize: 13, Radius: 4, RadiusSmall: 4,
		ControlH: 30, FieldH: 32, ComboH: 30, Checkbox: 20, Radio: 20,
		MenuItemH: 30, MenuBarH: 30, TabH: 32, RowH: 32, TitleBar: 44, HeaderH: 32,
		ProgressH: 10, SliderH: 20, Thumb: 14, Scroll: 10, Pad: 12, FieldPad: 12,
		ToolBarH: 44, StatusBarH: 28, SpinnerW: 22, SwitchW: 32, SwitchH: 18,
		ViewFrame: 1, Border: 1, FocusWidth: 1,
	}
	light := map[string]string{
		"window": "#f9f9fa", "field": "#ffffff", "sidebar": "#efeff0", "subPanel": "#f9f9fa", "titleBar": "#efeff0",
		"toolBar": "#f9f9fa", "card": "#ffffff", "tabPane": "#f9f9fa", "header": "#f9f9fa",
		"popup": "#ffffff", "popupBorder": "#e2e2e2", "dialog": "#ffffff",
		"border0": "#e2e2e2", "border1": "#dcdbdd", "border2": "#e9e8ea", "checkBorder": "#d4d4d6",
		"text": "#282a30", "text2": "#6f6e77", "textDis": "#86848d", "link": "#5e6ad2", "headerText": "#6f6e77",
		"accent": "#6d78d5", "onAccent": "#ffffff", "focus": "#6d78d5", "fieldFocus": "#6d78d5",
		"btn": "#ffffff", "btnHover": "#f4f2f4", "btnPress": "#eeedef", "btnBorder": "#dcdbdd", "btnText": "#282a30",
		"wash": "#0000000a", "sel": "#00000012", "menuHover": "#0000000a",
		"tip": "#ffffff", "tipText": "#282a30", "tipBorder": "#e2e2e2",
		"checkOff": "#ffffff", "checkOn": "#6d78d5", "checkMark": "#ffffff",
		"switchOff": "#e4e2e4", "switchOffBorder": "#dcdbdd", "switchOn": "#6d78d5", "knobOff": "#ffffff", "knobOn": "#ffffff",
		"track": "#e4e2e4", "progress": "#6d78d5", "progressTrack": "#e4e2e4",
		"scrollThumb": "#0000001a", "scrollThumbHot": "#00000033",
		"tabText2": "#6f6e77", "segTrack": "#00000000", "segOn": "#ffffff", "segOnBorder": "#e2e2e2", "segOnText": "#282a30",
		"danger": "#f34e52", "success": "#27a644", "warning": "#f0bf00",
		"selection": "#6d78d540",
	}
	dark := map[string]string{
		"window": "#121213", "field": "#1c1c1f", "sidebar": "#09090a", "subPanel": "#121213", "titleBar": "#09090a",
		"toolBar": "#121213", "card": "#1c1c1f", "tabPane": "#121213", "header": "#121213",
		"popup": "#1c1c1f", "popupBorder": "#23252a", "dialog": "#1c1c1f",
		"border0": "#212224", "border1": "#34343a", "border2": "#23252a", "checkBorder": "#3e3e44",
		"text": "#f7f8f8", "text2": "#8a8f98", "textDis": "#62666d", "link": "#828fff", "headerText": "#8a8f98",
		"accent": "#5e6ad2", "onAccent": "#ffffff", "focus": "#5e6ad2", "fieldFocus": "#5e6ad2",
		"btn": "#232326", "btnHover": "#28282c", "btnPress": "#2e2e33", "btnBorder": "#34343a", "btnText": "#f7f8f8",
		"wash": "#ffffff0d", "sel": "#ffffff12", "menuHover": "#ffffff0d",
		"tip": "#1c1c1f", "tipText": "#f7f8f8", "tipBorder": "#23252a",
		"checkOff": "#1c1c1f", "checkOn": "#5e6ad2", "checkMark": "#ffffff",
		"switchOff": "#28282c", "switchOffBorder": "#34343a", "switchOn": "#5e6ad2", "knobOff": "#8a8f98", "knobOn": "#ffffff",
		"track": "#28282c", "progress": "#5e6ad2", "progressTrack": "#28282c",
		"scrollThumb": "#ffffff1a", "scrollThumbHot": "#ffffff33",
		"tabText2": "#8a8f98", "segTrack": "#00000000", "segOn": "#28282c", "segOnBorder": "#34343a", "segOnText": "#f7f8f8",
		"danger": "#f34e52", "success": "#27a644", "warning": "#f0bf00",
		"selection": "#5e6ad259",
	}
	return []webSpec{
		{name: "linear", label: "Linear", era: "Linear", lineage: "Linear", year: 2026, c: light, p: p, m: m,
			summary: "Linear's light theme: warm #f9f9fa greys, 4px controls and 8px inputs, the indigo accent, compact rounded tabs."},
		{name: "linear-night", label: "Linear Dark", era: "Linear", lineage: "Linear", year: 2026, dark: true, c: dark, p: p, m: m,
			summary: "Linear's dark theme: #121213 content beside a #09090a sidebar, hairlines barely lighter, the #5e6ad2 indigo."},
	}
}
