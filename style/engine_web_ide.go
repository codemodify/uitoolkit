package style

import "github.com/codemodify/paintengine2d"

// The code editors on the web engine: VS Code's 2026 themes and JetBrains'
// Islands (see engine_web.go for the engine and its params). What they add:
//
//   - "tabStyle" 3, editor tabs (VS Code): boxed tabs on the strip's colour
//     ("tabStrip" [titleBar]) over its bottom hairline, a hairline ("tabSep"
//     [border0]) at each tab's end, the selected tab in the editor's colour
//     ("tabActive" [tabPane]) with a "tabLine" px line of the "tabLine"
//     colour along its top, open to the editor below;
//   - "tabStyle" 4, pill tabs (Islands): the selected tab a "tabPillH" [28]
//     tall pill of "tabRadius" [radius + 2] corners in "tabPill" [segOn] with
//     a hairline of "tabPillBorder" [none], the others bare labels that take
//     a pill of the wash under the pointer;
//   - "island" [0], px: list, tree, table and sidebar frames, cards and tab
//     views are islands — panels rounded by viewRadius (cardRadius for
//     cards) inside a band this wide, their outline the fill. The band is
//     left to what is behind: the main window's colour between islands, the
//     island itself where one holds another (IntelliJ's tool windows hold
//     frameless trees), so nested islands merge. A tab view's strip is its
//     island's top, a group box's heading sits inside its island (a tool
//     window's title);
//   - "scrollRadius" [a pill], the scroll thumb's corner (VS Code's square
//     sliders: 0);
//   - "openRing" [0] 1: an open combo keeps the focus ring, as Swing rings
//     whatever holds focus (Islands).

// The editors' tab styles.
const (
	webTabEditor = 3
	webTabPill   = 4
)

// webIDESet is a look's editor additions, built once per look.
type webIDESet struct {
	island, scrollR, pillH, pillR                 float32
	openRing                                      bool
	tabStrip, tabActive, tabSep, pill, pillBorder paintengine2d.Color
}

type webIDEKey struct{}

func webIDE(l *Classic) *webIDESet {
	return l.Memo(webIDEKey{}, func() any { return webIDEBuild(l) }).(*webIDESet)
}

func webIDEBuild(l *Classic) *webIDESet {
	c := webColors(l)
	return &webIDESet{
		island:     max(l.P("island", 0), 0),
		scrollR:    l.P("scrollRadius", -1),
		pillH:      l.P("tabPillH", 28),
		pillR:      l.P("tabRadius", c.radius+2),
		openRing:   l.P("openRing", 0) != 0,
		tabStrip:   l.X("tabStrip", c.titleBar),
		tabActive:  l.X("tabActive", c.tabPane),
		tabSep:     l.X("tabSep", c.border0),
		pill:       l.X("tabPill", c.segOn),
		pillBorder: l.X("tabPillBorder", paintengine2d.Color{}),
	}
}

// ---- tabs ---------------------------------------------------------------------------------------

// ideTabBar paints the strip under editor and pill tabs, and a tab view's
// top where the look has islands (any tab style but the browser's); it
// reports whether it did.
func (c *webSet) ideTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) bool {
	if c.tabStyle == webTabBrowser {
		return false
	}
	if band := c.islandBand(l); band > 0 {
		// The island's top: rounded above, open into the pane's part below.
		top := paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X+band, b.Min.Y+band), Max: paintengine2d.Pt(b.Max.X-band, b.Max.Y)}
		if top.Dx() >= 2 && top.Dy() >= 2 {
			r := min(c.rad(l, c.viewR), top.Dx()*0.5, top.Dy())
			ctx.DrawPath(RoundRectPath(top, r, r, 0, 0), paintengine2d.Fill(c.tabPane))
		}
		return true
	}
	switch c.tabStyle {
	case webTabEditor:
		d := webIDE(l)
		px := c.px(l)
		ctx.DrawRect(b, paintengine2d.Fill(d.tabStrip))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(d.tabSep))
	case webTabPill:
		ctx.DrawRect(b, paintengine2d.Fill(c.tabPane))
	default:
		return false
	}
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

// pillTab is one of Islands' editor tabs: the selected one a pill in its
// hairline, the others bare labels that take a pill of the wash under the
// pointer. In an island the pills sit in the part below the band, the first
// clear of the island's corner.
func (c *webSet) pillTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	px := c.px(l)
	if b.Dx() < 8*px || b.Dy() < 8*px {
		return
	}
	d := webIDE(l)
	band := c.islandBand(l)
	h := min(snap(l.S(d.pillH)), b.Dy()-band-2*px)
	in, lead := snap(l.S(2)), float32(0)
	if st.First() && band > 0 {
		lead = min(2*band, snap(b.Dx()*0.1))
	}
	pill := paintengine2d.XYWH(b.Min.X+in+lead, snap((b.Min.Y+band+b.Max.Y-h)*0.5), b.Dx()-2*in-lead, h)
	r := min(c.rad(l, d.pillR), pill.Dy()*0.5)
	fg, f := c.tabText2, c.tabFace
	switch {
	case selected:
		c.box(l, ctx, pill, r, c.disabled(d.pill, st), c.disabled(d.pillBorder, st))
		fg, f = c.tabText, c.tabSelFace
	case (st.Hovered() || st.Pressed()) && !st.Disabled():
		a := float32(1)
		if st.Pressed() {
			a = 1.8
		}
		c.box(l, ctx, pill, r, webA(c.wash, a), paintengine2d.Color{})
		fg = c.text
	}
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, f, label, pill, fg, AlignCenter, l.S(10))
	if st.Focused() && !st.Disabled() {
		c.focusRing(l, ctx, b, pill, r)
	}
}

// ---- islands ------------------------------------------------------------------------------------

// islandBand is the band round an island in device pixels (0: the look has
// no islands).
func (c *webSet) islandBand(l *Classic) float32 {
	if v := webIDE(l).island; v > 0 {
		return max(snap(l.S(v)), c.px(l))
	}
	return 0
}

// island paints an island into b: a panel in fill, rounded r, inside the
// band, which it leaves to what is behind.
func (c *webSet) island(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, fill paintengine2d.Color) {
	in := winSnap(b).Inset(c.islandBand(l))
	if in.Dx() < 2 || in.Dy() < 2 || fill.A <= 0 {
		return
	}
	rr := min(c.rad(l, r), in.Dx()*0.5, in.Dy()*0.5)
	ctx.DrawRoundRect(in, rr, rr, paintengine2d.Fill(fill))
}

// islandView paints a list, tree, table or sidebar frame as an island and
// reports whether the look has islands.
func (c *webSet) islandView(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) bool {
	if c.islandBand(l) <= 0 {
		return false
	}
	fill := c.field
	if st.Sidebar() {
		fill = c.sidebar
	}
	c.island(l, ctx, b, c.viewR, fill)
	return true
}

// islandCard paints a card (a group box, a raised panel) as an island and
// reports whether the look has islands.
func (c *webSet) islandCard(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) bool {
	if c.islandBand(l) <= 0 {
		return false
	}
	c.island(l, ctx, b, c.cardR, c.card)
	return true
}

// islandGroup paints a group box as an island with its heading inside, at
// the top like a tool window's title, and reports whether the look has
// islands.
func (c *webSet) islandGroup(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string) bool {
	if !c.islandCard(l, ctx, b) {
		return false
	}
	if title != "" {
		in := c.islandBand(l) + l.metrics.Pad
		hb := paintengine2d.XYWH(b.Min.X+in, b.Min.Y+in-snap(l.metrics.Pad*0.5), b.Dx()-2*in, webHeadH(l)-l.S(4))
		l.drawFittedText(ctx, c.bold, title, hb, c.text, AlignStart, 0)
	}
	return true
}

// islandPane paints a tab view as an island (its tab bar paints the top)
// and reports whether the look has islands.
func (c *webSet) islandPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) bool {
	if c.islandBand(l) <= 0 || c.tabStyle == webTabBrowser {
		return false
	}
	c.island(l, ctx, b, c.viewR, c.tabPane)
	return true
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
	return append(vscodeSpecs(), islandsSpecs()...)
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
			summary: "VS Code's Light 2026: #fafafd chrome around a white editor, 4px buttons and inputs, 1px focus borders inside, a black line atop the active tab."},
		{name: "vscode-night", label: "VS Code Dark 2026", era: "VS Code", lineage: "VS Code", year: 2026, dark: true, c: dark, p: p, m: m,
			summary: "VS Code's Dark 2026: a #121314 editor in #191a1b chrome, #297aa0 buttons, the #3994bc line atop the active tab, white-alpha selections."},
	}
}

// islandsSpecs are JetBrains' Islands themes, the IntelliJ Platform's
// default since 2025.3, as their theme files set them (intellij-community,
// Apache-2.0): the tool windows and the editor 10px-rounded islands inside
// a 3px band of the main window's colour, 4px controls with a 2px focus ring
// beyond a gap, pill editor tabs, rounded selections inset 12px from an
// island's side, Inter 13.
func islandsSpecs() []webSpec {
	p := map[string]float32{
		// Arcs are diameters: 8 (4px) for controls and selections, 20
		// (10px) for islands, 12 (6px) for the tab pills.
		"radius": 4, "fieldRadius": 4, "comboRadius": 4, "overlayRadius": 8, "tipRadius": 6, "cardRadius": 10,
		"viewRadius": 10, "windowRadius": 10, "checkRadius": 3, "rowRadius": 4, "sideRadius": 4, "menuRowRadius": 4, "menuInset": 8,
		// Component.focusWidth 2 round a gap the background's colour (the
		// focused border), on whatever holds focus.
		"focusStyle": webFocusOutside, "focusWidth": 2, "focusGap": 1, "openRing": 1,
		"primaryStyle": 0, "buttonWeight": 400,
		"tabStyle": webTabPill, "tabRadius": 6, "tabPillH": 28, "tabWeight": 400,
		"checkStyle": 1, "checkSize": 16, "radioStyle": 1, "radioDot": 6, "knob": 12, "trackH": 4, "progressH": 4,
		// Popup rows inset 8; tree and list rows 8 inside the view's own
		// room, 12 from the island's side.
		"menuHighlight": 0, "rowInset": 8, "sideInset": 8,
		// The inactive selection is the neutral grey.
		"listSelOff": 1.2, "listOffWash": 1, "sideSelOff": 1.2, "sideOffWash": 1, "tableSelOff": 1.2, "tableOffWash": 1,
		"captionStyle": 1, "toolWash": 1, "seg": 2, "headerWeight": 400, "headerSep": 1, "accordion": 1,
		// Island.borderWidth 6: a 3px band on each side; thin idle scroll
		// thumbs.
		"island": 3, "scrollIdle": 4, "scrollInset": 2,
		"menuShadow": 0.22, "menuShadowY": 4, "menuShadowBlur": 20,
		"tipShadow": 0.15, "tipShadowY": 2, "tipShadowBlur": 8,
		"dialogShadow": 0.32, "dialogShadowY": 10, "dialogShadowBlur": 40,
	}
	// 28px controls inside the 3px focus room; 24px rows and toggle cells;
	// the 40px main tool bar and tab strip; the 26×16 switch of 2026.3.
	m := ChromeMetrics{
		FontSize: 13, Radius: 4, RadiusSmall: 4,
		ControlH: 34, FieldH: 34, ComboH: 34, Checkbox: 24, Radio: 24,
		MenuItemH: 28, MenuBarH: 32, TabH: 40, RowH: 24, TitleBar: 40, HeaderH: 28,
		ProgressH: 8, SliderH: 22, Thumb: 14, Scroll: 10, Pad: 12, FieldPad: 12,
		ToolBarH: 40, StatusBarH: 26, SpinnerW: 20, SwitchW: 26, SwitchH: 16,
		ViewFrame: 1, Border: 1, FocusWidth: 2,
	}
	// The islands (tool windows, the editor, fields) on the main window's
	// colour, which also fills the tool bar, title and status bars and
	// draws their borders; component borders, selections, the tab pill and
	// the status colours as the themes set them. Buttons, check box borders,
	// the switch's off track and the scroll thumbs are steps of the 16-grey
	// ramp (inferred).
	light := map[string]string{
		"window": "#e9eaee", "field": "#ffffff", "sidebar": "#ffffff", "subPanel": "#ffffff", "titleBar": "#e9eaee",
		"toolBar": "#e9eaee", "card": "#ffffff", "tabPane": "#ffffff", "header": "#ffffff",
		"popup": "#ffffff", "popupBorder": "#d1d3d9", "dialog": "#f7f8f9",
		"border0": "#e9eaee", "border1": "#d1d3d9", "border2": "#e9eaee", "checkBorder": "#9fa2a8",
		"text": "#000000", "text2": "#73767c", "textDis": "#9fa2a8", "link": "#2f5eb9", "headerText": "#73767c",
		"accent": "#3871e1", "accentHover": "#2f5eb9", "accentPress": "#2e4d89", "onAccent": "#ffffff",
		"focus": "#3871e1", "fieldFocus": "#f7f8f9",
		"btn": "#ffffff", "btnHover": "#f7f8f9", "btnPress": "#e9eaee", "btnBorder": "#d1d3d9", "btnText": "#000000",
		"wash": "#00000012", "sel": "#d0dffe", "menuHover": "#d0dffe", "menuText": "#000000",
		"tip": "#ffffff", "tipText": "#000000", "tipBorder": "#d1d3d9",
		"checkOff": "#ffffff", "checkOn": "#3871e1", "checkMark": "#ffffff",
		"switchOff": "#c3c5cb", "switchOn": "#3871e1", "knobOff": "#ffffff", "knobOn": "#ffffff",
		"track": "#dddfe4", "progress": "#3871e1", "progressTrack": "#dddfe4",
		"scrollThumb": "#b5b7bd", "scrollThumbHot": "#8b8e94",
		"tabPill": "#e3ebfe", "tabPillBorder": "#a7c5ff", "tabText": "#000000", "tabText2": "#000000",
		"danger": "#c54e58", "success": "#338555", "warning": "#a56906",
		"selection": "#d0dffe",
	}
	dark := map[string]string{
		"window": "#26282c", "field": "#191a1c", "sidebar": "#191a1c", "subPanel": "#191a1c", "titleBar": "#26282c",
		"toolBar": "#26282c", "card": "#191a1c", "tabPane": "#191a1c", "header": "#191a1c",
		"popup": "#26282c", "popupBorder": "#33353b", "dialog": "#191a1c",
		"border0": "#26282c", "border1": "#40434a", "border2": "#33353b", "checkBorder": "#5f6269",
		"text": "#d1d3d9", "text2": "#73767c", "textDis": "#4c4f56", "link": "#538af9", "headerText": "#8b8e94",
		"accent": "#3871e1", "accentHover": "#538af9", "accentPress": "#2f5eb9", "onAccent": "#ffffff",
		"focus": "#3871e1", "fieldFocus": "#191a1c",
		"btn": "#191a1c", "btnHover": "#212326", "btnPress": "#26282c", "btnBorder": "#40434a", "btnText": "#d1d3d9",
		"wash": "#ffffff17", "sel": "#2a4371", "menuHover": "#2a4371", "menuText": "#d1d3d9",
		"tip": "#33353b", "tipText": "#d1d3d9", "tipBorder": "#40434a",
		"checkOff": "#191a1c", "checkOn": "#3871e1", "checkMark": "#ffffff",
		"switchOff": "#4c4f56", "switchOn": "#3871e1", "knobOff": "#d1d3d9", "knobOn": "#ffffff",
		"track": "#40434a", "progress": "#3871e1", "progressTrack": "#40434a",
		"scrollThumb": "#4c4f56", "scrollThumbHot": "#73767c",
		"tabPill": "#233558", "tabPillBorder": "#2e4d89", "tabText": "#d1d3d9", "tabText2": "#d1d3d9",
		"danger": "#f57e84", "success": "#6db083", "warning": "#d59637",
		"selection": "#2a4371",
	}
	return []webSpec{
		{name: "islands", label: "Islands Light", era: "Islands", lineage: "JetBrains", year: 2025, c: light, p: p, m: m,
			summary: "JetBrains' Islands Light: white 10px-rounded islands on #e9eaee, 4px controls, the #3871e1 accent, pill editor tabs, rounded selections inset 12px."},
		{name: "islands-night", label: "Islands Dark", era: "Islands", lineage: "JetBrains", year: 2025, dark: true, c: dark, p: p, m: m,
			summary: "JetBrains' Islands Dark: #191a1c islands on #26282c with 6px between them, #2a4371 selections, #233558 tab pills, the #3871e1 accent."},
	}
}
