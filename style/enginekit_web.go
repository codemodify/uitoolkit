package style

// Shared parts of the web-derived engines — the modern flat looks that
// several design systems (Primer, shadcn, Geist, Linear, VS Code) are
// variations of. The specs and the small helpers that describe a band, a
// header row or an overlay are common to all of them, so they are not
// any one system's.
//
// No build tag, so everything here is in every build.

import (
	"maps"

	"github.com/codemodify/paintengine2d"
)

// webA is c with its alpha multiplied by a.
func webA(c paintengine2d.Color, a float32) paintengine2d.Color { return c.WithAlpha(c.A * a) }

// band fills the ring between the round rects outer (radius ro) and inner.
func webBand(ctx *paintengine2d.Context, outer paintengine2d.Rect, ro float32, inner paintengine2d.Rect, ri float32, col paintengine2d.Color) {
	if outer.Empty() || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	p.AddRoundRect(outer, min(ro, outer.Dx()*0.5, outer.Dy()*0.5), min(ro, outer.Dx()*0.5, outer.Dy()*0.5))
	if !inner.Empty() {
		p.AddRoundRect(inner, min(ri, inner.Dx()*0.5, inner.Dy()*0.5), min(ri, inner.Dx()*0.5, inner.Dy()*0.5))
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
}

// webOver is col flattened over the opaque base.
func webOver(base, col paintengine2d.Color) paintengine2d.Color {
	if col.A >= 1 {
		return col
	}
	return Mix(base, col, col.A)
}

// webHeadH is the band a group's heading takes above its card.
func webHeadH(l *Classic) float32 { return snap(l.BoldFont().Height() + l.S(8)) }

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

// webChrome sets the chrome states the base painters read from the palette.
func webChrome(tok *ThemeTokens) {
	p := tok.Palette
	tok.Hot = ChromeState{Fill: p.MenuHover, Border: p.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: Mix(p.MenuHover, p.Text, 0.06), Border: p.Accent}
	tok.Selected = ChromeState{Fill: webOver(p.Field, p.Selection), Border: p.Accent}
	tok.Focus = ChromeState{Fill: p.Field, Border: p.Focus}
}

func webDark1(c paintengine2d.Color) paintengine2d.Color { return flatLighten(c, -28.5/255*100) }

// webLight1 and webDark1 are Avalonia's accent shades (its
// SystemAccentColorLight1 and Dark1, which SourceGit's accent buttons use
// under the pointer and pressed): the accent's HSL lightness moved up 39 and
// down 28.5 of 255.
func webLight1(c paintengine2d.Color) paintengine2d.Color { return flatLighten(c, 39.0/255*100) }

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

func editorPaletteSpecs() []webSpec {
	var out []webSpec
	out = append(out, gruvboxSpecs()...)
	out = append(out, solarizedSpecs()...)
	out = append(out, oneSpecs()...)
	return out
}

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

// catppuccinFlavour is one flavour's 26 colours (catppuccin/palette 1.8).
type catppuccinFlavour struct {
	name, label, summary                                   string
	dark                                                   bool
	red, yellow, green, blue, lavender                     string
	text, subtext1, subtext0, overlay2, overlay1, overlay0 string
	surface2, surface1, surface0, base, mantle, crust      string
}

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

// oneParams are Atom's UI metrics on the palettes' shape: 3px corners,
// boxed editor tabs, the tree's full-width selection.
var oneParams = webWith(paletteParams, map[string]float32{
	"radius": 3, "fieldRadius": 3, "comboRadius": 3, "overlayRadius": 3, "tipRadius": 3, "cardRadius": 3,
	"viewRadius": 3, "windowRadius": 3, "checkRadius": 3, "menuRowRadius": 3,
	"rowRadius": 0, "sideRadius": 0, "rowInset": 0, "sideInset": 0,
	"tabStyle": webTabEditor, "tabLine": 1,
})

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

// pmenuParams are the palettes' shape with the menu's current item in a
// colour of its own (Vim's PmenuSel), its shortcut in the item's text.
var pmenuParams = webWith(paletteParams, map[string]float32{"menuHighlight": 1})

// webWith is base with over's entries on top.
func webWith[V any](base, over map[string]V) map[string]V {
	out := make(map[string]V, len(base)+len(over))
	maps.Copy(out, base)
	maps.Copy(out, over)
	return out
}

// Focus styles.
const (
	webFocusOutside = 0
	webFocusInside  = 1
	webFocusDotted  = 2
)

// Tab styles.
const (
	webTabUnderline = 0
	webTabSegmented = 1
	webTabBrowser   = 2
)

// The editors' tab styles.
const (
	webTabEditor = 3
	webTabPill   = 4
)
