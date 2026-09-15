package style

import "github.com/codemodify/paintengine2d"

// The code editors on the web engine: VS Code's 2026 themes (see
// engine_web.go for the engine and its params). What they add:
//
//   - "tabStyle" 3, editor tabs (VS Code): boxed tabs on the strip's colour
//     ("tabStrip" [titleBar]) over its bottom hairline, a hairline ("tabSep"
//     [border0]) at each tab's end, the selected tab in the editor's colour
//     ("tabActive" [tabPane]) with a "tabLine" px line of the "tabLine"
//     colour along its top, open to the editor below;
//   - "scrollRadius" [a pill], the scroll thumb's corner (VS Code's square
//     sliders: 0).

// The editors' tab style.
const webTabEditor = 3

// webIDESet is a look's editor additions, built once per look.
type webIDESet struct {
	scrollR                     float32
	tabStrip, tabActive, tabSep paintengine2d.Color
}

type webIDEKey struct{}

func webIDE(l *Classic) *webIDESet {
	return l.Memo(webIDEKey{}, func() any { return webIDEBuild(l) }).(*webIDESet)
}

func webIDEBuild(l *Classic) *webIDESet {
	c := webColors(l)
	return &webIDESet{
		scrollR:   l.P("scrollRadius", -1),
		tabStrip:  l.X("tabStrip", c.titleBar),
		tabActive: l.X("tabActive", c.tabPane),
		tabSep:    l.X("tabSep", c.border0),
	}
}

// ---- tabs ---------------------------------------------------------------------------------------

// ideTabBar paints the strip under editor tabs and reports whether it did.
func (c *webSet) ideTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) bool {
	if c.tabStyle != webTabEditor {
		return false
	}
	d := webIDE(l)
	px := c.px(l)
	ctx.DrawRect(b, paintengine2d.Fill(d.tabStrip))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(d.tabSep))
	return true
}

// editorTab is one of VS Code's editor tabs: the selected one the editor's
// colour, its top line across it and its bottom open to the editor; the
// others the strip under a muted label that brightens under the pointer; a
// hairline at each tab's end.
func (c *webSet) editorTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	px := c.px(l)
	if b.Dx() < 4*px || b.Dy() < 4*px {
		return
	}
	d := webIDE(l)
	fg, f := c.tabText2, c.tabFace
	switch {
	case selected:
		ctx.DrawRect(b, paintengine2d.Fill(d.tabActive))
		fg, f = c.tabText, c.tabSelFace
	case st.Hovered() || st.Pressed():
		fg = c.text
	}
	if st.Disabled() {
		fg = c.textDis
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-px, b.Min.Y, px, b.Dy()), paintengine2d.Fill(d.tabSep))
	if selected {
		h := max(snap(l.S(c.tabLineW)), px)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-px, h), paintengine2d.Fill(c.disabled(c.tabLine, st)))
	}
	l.drawFittedText(ctx, f, label, b, fg, AlignCenter, l.S(12))
	if st.Focused() && !st.Disabled() {
		fb := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-px, b.Dy())
		c.focusRing(l, ctx, fb, fb, 0)
	}
}

// ---- scroll bars --------------------------------------------------------------------------------

// thumbR is a scroll thumb's corner: a pill, or the pack's "scrollRadius".
func (c *webSet) thumbR(l *Classic, t paintengine2d.Rect) float32 {
	half := min(t.Dx(), t.Dy()) * 0.5
	if r := webIDE(l).scrollR; r >= 0 {
		return min(c.rad(l, r), half)
	}
	return half
}

// ---- packs --------------------------------------------------------------------------------------

// ideSpecs are the editors' packs.
func ideSpecs() []webSpec {
	return vscodeSpecs()
}

// vscodeSpecs are VS Code's default themes since 1.113 (March 2026), named
// "Light 2026" and "Dark 2026" in 1.114, as their theme files set them
// (release 1.137, MIT) on its base sizes: 13px workbench text, 4px buttons
// and inputs, 1px focus borders drawn inside, editor tabs with a line along
// the active one's top, full-width 22px rows, 8px menus of 24px rows, 10px
// square scroll sliders.
func vscodeSpecs() []webSpec {
	p := map[string]float32{
		// Corner tokens: buttons, inputs and selects small (4), menus,
		// hovers and widgets large (8) with medium (6) rows, dialogs
		// xLarge (12); 3px check boxes; flat full-width rows.
		"radius": 4, "fieldRadius": 4, "comboRadius": 4, "overlayRadius": 8, "tipRadius": 8, "cardRadius": 6,
		"viewRadius": 0, "windowRadius": 12, "checkRadius": 3, "rowRadius": 0, "sideRadius": 0, "menuRowRadius": 6, "menuInset": 4,
		// focusBorder: 1px, at offset −1 (inside).
		"focusStyle": webFocusInside, "focusWidth": 1,
		"primaryStyle": 0, "buttonWeight": 400,
		// Editor tabs, a 1px tab.activeBorderTop.
		"tabStyle": webTabEditor, "tabLine": 1, "tabWeight": 400,
		// 18px check boxes with the check in the checkbox foreground; the
		// 2px progress bar.
		"checkStyle": 0, "checkSize": 18, "radioStyle": 0, "radioSize": 18, "radioDot": 8, "progressH": 2,
		"menuHighlight": 0, "rowInset": 0, "sideInset": 0,
		// Lists: the selection and hover washes; an unfocused selection the
		// neutral one (list.inactiveSelectionBackground).
		"listOffWash": 1, "sideOffWash": 1, "tableOffWash": 1,
		// 10px sliders, square, whole while shown.
		"scrollIdle": 10, "scrollInset": 0, "scrollRadius": 0,
		"captionStyle": 1, "toolWash": 1, "seg": 2,
		"headerWeight": 400, "headerSep": 1, "accordion": 0,
		// Shadow tokens: menus and widgets lg, hovers md, dialogs xl.
		"menuShadow": 0.2, "menuShadowY": 2, "menuShadowBlur": 16,
		"tipShadow": 0.12, "tipShadowY": 2, "tipShadowBlur": 8,
		"dialogShadow": 0.3, "dialogShadowY": 8, "dialogShadowBlur": 40,
	}
	m := ChromeMetrics{
		FontSize: 13, Radius: 4, RadiusSmall: 2,
		ControlH: 26, FieldH: 26, ComboH: 26, Checkbox: 22, Radio: 22,
		MenuItemH: 24, MenuBarH: 30, TabH: 35, RowH: 22, TitleBar: 35, HeaderH: 24,
		ProgressH: 8, SliderH: 20, Thumb: 14, Scroll: 10, Pad: 10, FieldPad: 6,
		ToolBarH: 35, StatusBarH: 22, SpinnerW: 18, SwitchW: 32, SwitchH: 18,
		ViewFrame: 1, Border: 1, FocusWidth: 1,
	}
	// The side bar, title bar, status bar, panel and tab strip share one
	// colour round the editor's. Secondary buttons, disabled text, text
	// selection and the warning and success colours are the colour
	// registry's defaults (the secondary buttons inferred from Modern's).
	light := map[string]string{
		"window": "#fafafd", "field": "#ffffff", "sidebar": "#fafafd", "subPanel": "#fafafd", "titleBar": "#fafafd",
		"toolBar": "#fafafd", "card": "#ffffff", "tabPane": "#ffffff", "header": "#fafafd",
		"popup": "#fafafd", "popupBorder": "#e4e5e6", "dialog": "#fafafd",
		"border0": "#f0f1f2", "border1": "#d8d8d866", "border2": "#f0f1f2", "checkBorder": "#868686",
		"text": "#202020", "text2": "#606060", "textDis": "#adadaf", "link": "#0069cc", "headerText": "#606060",
		"accent": "#0069cc", "accentHover": "#0063c1", "accentPress": "#005bb2", "onAccent": "#ffffff",
		"focus": "#0069cc", "fieldFocus": "#0069cc",
		"btn": "#e4e5e6", "btnHover": "#d8d9da", "btnPress": "#cecfd0", "btnBorder": "#00000000", "btnText": "#202020",
		"wash": "#00000014", "sel": "#00000025", "menuHover": "#0069cc1a", "menuText": "#202020",
		"tip": "#fafafd", "tipText": "#202020", "tipBorder": "#e4e5e6",
		"checkOff": "#eaeaea", "checkOn": "#202020", "checkMark": "#202020",
		"progressTrack": "#00000000", "scrollThumb": "#646464c0", "scrollThumbHot": "#646464d0",
		"tabStrip": "#fafafd", "tabActive": "#ffffff", "tabSep": "#f0f1f2", "tabLine": "#000000",
		"tabText": "#202020", "tabText2": "#606060",
		"danger": "#ad0707", "success": "#388a34", "warning": "#bf8803",
		"selection": "#add6ff",
	}
	dark := map[string]string{
		"window": "#191a1b", "field": "#191a1b", "sidebar": "#191a1b", "subPanel": "#191a1b", "titleBar": "#191a1b",
		"toolBar": "#191a1b", "card": "#121314", "tabPane": "#121314", "header": "#191a1b",
		"popup": "#202122", "popupBorder": "#2a2b2c", "dialog": "#202122",
		"border0": "#2a2b2c", "border1": "#333536", "border2": "#2a2b2c", "checkBorder": "#707070",
		"text": "#bfbfbf", "text2": "#8c8c8c", "textDis": "#6c6c6c", "link": "#48a0c7", "headerText": "#8c8c8c",
		"accent": "#297aa0", "accentHover": "#2b7da3", "accentPress": "#256e90", "onAccent": "#ffffff",
		"focus": "#3994bcb3", "fieldFocus": "#3994bcb3",
		"btn": "#2a2b2c", "btnHover": "#313233", "btnPress": "#38393a", "btnBorder": "#00000000", "btnText": "#bfbfbf",
		"wash": "#ffffff14", "sel": "#ffffff22", "menuHover": "#3994bc26", "menuText": "#bfbfbf",
		"tip": "#202122", "tipText": "#bfbfbf", "tipBorder": "#2a2b2c",
		"checkOff": "#242526", "checkOn": "#bfbfbf", "checkMark": "#bfbfbf",
		"progressTrack": "#00000000", "scrollThumb": "#a8a9aa85", "scrollThumbHot": "#a8a9aa90",
		"tabStrip": "#191a1b", "tabActive": "#121314", "tabSep": "#2a2b2c", "tabLine": "#3994bc",
		"tabText": "#bfbfbf", "tabText2": "#8c8c8c",
		"danger": "#f48771", "success": "#89d185", "warning": "#cca700",
		"selection": "#264f78",
	}
	return []webSpec{
		{name: "vscode", label: "VS Code Light 2026", era: "VS Code", lineage: "VS Code", year: 2026, c: light, p: p, m: m,
			summary: "VS Code's Light 2026: #fafafd chrome round a white editor, 4px buttons and inputs, 1px focus borders inside, a black line atop the active tab."},
		{name: "vscode-night", label: "VS Code Dark 2026", era: "VS Code", lineage: "VS Code", year: 2026, dark: true, c: dark, p: p, m: m,
			summary: "VS Code's Dark 2026: a #121314 editor in #191a1b chrome, #297aa0 buttons, the #3994bc line atop the active tab, white-alpha selections."},
	}
}
