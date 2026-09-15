package style

import "github.com/codemodify/paintengine2d"

// webEngine paints the flat design systems of today's apps and web sites —
// SourceGit (Avalonia's Fluent theme as SourceGit restyles it), GitHub's
// Primer, shadcn/ui, Vercel's Geist and Linear — and the editor palettes
// people carry between apps (Catppuccin, Nord, Dracula, Tokyo Night, Rosé
// Pine). They share one idiom that no older engine draws together: flat
// faces in a 1px hairline with small corner radii, a keyboard focus ring
// (outside the control, inside it, or Avalonia's dotted adorner), underlined
// or segmented tabs, 16px check boxes, pill switches, menus of rounded rows on
// a rounded popover and overlay scroll bars. Every difference between them is
// a number or a colour, so each design system is a pack: the shapes below
// are params, the colours "extra" keys.
//
// Params (design pixels at 1× unless said; default in brackets):
//
//   - corners: "radius" [6] buttons, combos, tool buttons; "fieldRadius"
//     [radius] text fields (SourceGit's are square); "comboRadius" [radius];
//     "overlayRadius" [8] menus and popovers; "tipRadius" [radius];
//     "cardRadius" [8] cards and group boxes; "viewRadius" [radius] list,
//     tree and table frames; "windowRadius" [overlayRadius] in-app windows;
//     "checkRadius" [4]; "rowRadius" [6] inset list rows; "sideRadius"
//     [rowRadius] sidebar rows; "menuRowRadius" [rowRadius];
//     "switchShape" 0 pill [0] or 1 a rounded rect of "switchRadius";
//   - lines: "hairline" [1] border width; "restShadow" [0] the alpha of a
//     resting drop shadow under buttons and fields (it needs the focus
//     margin to show); "pressScale" [1] (SourceGit's buttons shrink to 0.98);
//     "disabledAlpha" [0.5];
//   - focus: "focusStyle" 0 a ring outside the face, in a margin every
//     control keeps, 1 [default] a band inside the face over its border, 2
//     Avalonia's dotted adorner (SourceGit); "focusWidth" [2], "focusGap" [0]
//     (the gap between face and an outside ring), "focusAlpha" [1];
//     "hoverBorder" [0] 1: fields, combos and toggles take the accent border
//     under the pointer; "checkFocus" [0] 1: check boxes and radios show focus
//     with a 2px accent border (filled when checked), as SourceGit's do;
//   - buttons: "primaryStyle" 0 [default] the accent, 1 solid foreground
//     (shadcn, Geist), 2 the pack's own "primary" colour (Primer's green);
//     "buttonWeight" [500]; "primaryFirst" [0] 1: dialogs put the default
//     button first;
//   - tabs: "tabStyle" 0 [default] underline, 1 segmented (a raised knob on a
//     track), 2 browser (title-bar document tabs, see [BrowserTabEngine]);
//     "tabLine" [2] underline width, "tabLineGap" [0] its distance from the
//     bottom, "tabFit" [0] 1: the underline spans the label only,
//     "tabAccent" [0] 1: the selected label in the accent, "tabDim" [0] the
//     alpha of unselected labels in the text colour (0: the muted colour),
//     "tabWeight" [500], "tabSelWeight" [tabWeight] the selected label's,
//     "tabGrow" [0] px added to the label size, "tabHover" [0] 1: a wash
//     under the hovered tab, "tabBar" [1] a hairline under the strip;
//   - toggles: "checkStyle" 0 unfilled with an accent tick (SourceGit) or 1
//     [default] filled with a white tick; "checkSize" [16]; "radioSize"
//     [checkSize]; "radioStyle" 0 ring and dot or 1 [default] filled with a
//     centre dot; "radioDot" [6]; "knob" [track height − 4] switch knob;
//     "trackH" [4] slider track; "thumbStyle" 0 [default] a filled disc, 1 a
//     white disc in a border; "progressH" [4];
//   - menus: "menuHighlight" 0 [default] a neutral wash, 1 the accent;
//     "menuInset" [4] highlight inset from the popover's sides; "menuSep" [0]
//     1: separators start at the label column;
//   - rows: "rowInset" [0] list and tree rows are boxed this far in from the
//     sides (0: full width), "sideInset" [8] the same for sidebars, "rowGap"
//     [0] the box's vertical inset; consecutive selected rows join into one
//     box ([StateSelectedAbove]); per kind (list, side, table) "<k>Sel",
//     "<k>SelHover", "<k>SelOff" [1] the alpha of "sel" on a selected row in
//     a focused view, under the pointer and without focus, "<k>OffWash" [0]
//     1: an unfocused selection is the neutral wash instead, "<k>Hover" [1]
//     the hover wash's alpha; "rowBar" [0] width of an accent bar at the left
//     of a selected row; "expander" 0 triangles, 1 [default] chevrons;
//   - scroll bars: transient overlay bars of the scroll metric's width,
//     "scrollIdle" [2] the thin idle thumb, "scrollInset" [2], "scrollArrows"
//     [0] 1: arrow cells while the bar is expanded;
//   - windows: "captionStyle" 0 a caption strip in the title bar colour with
//     a centred bold title and a square close cell of "captionButton" [46]
//     that turns red, 1 [default] a web dialog header with an icon close
//     button; "captionCenter", "captionLine" [1 for style 0];
//   - tool buttons: "toolWash" [1] 0: no face, the glyph brightens from
//     "toolAlpha" [1] under the pointer (SourceGit); "seg" [1] a free toggle
//     tool button: 0 an accent pill, 1 a raised knob, 2 a wash;
//   - headers and accordions: "headerWeight" [600], "headerCenter" [0],
//     "headerSep" [1] a hairline between header cells;
//     "accordion" 0 SourceGit's group header (chevron at the left, bold muted
//     label), 1 [default] a web accordion (chevron at the right, divider);
//   - shadows: "menuShadow", "menuShadowY", "menuShadowBlur",
//     "menuShadowSpread"; the same for "tipShadow" and "dialogShadow"
//     (alpha, offset, blur — twice a CSS blur radius — and spread); "shadow"
//     [1] scales them all, 0 turns them off;
//   - fonts: "tipSize" [body] tool tip text size;
//   - "hoverFade" [0] ms of the hover cross-fade ([HintHoverFadeMs]);
//   - "accentFollows" [0] 1: the look takes the desktop's accent colour.
//
// The text size is the pack's "fontSize" metric (13px SourceGit and Linear,
// 14px the others); controls are sized in the same pixels.
//
// Colours ("extra"; unset keys derive from the palette): surfaces "window",
// "titleBar", "toolBar", "subPanel", "sidebar", "card", "dialog", "header",
// "tabPane", "popup", "popupBorder", "windowBorder", "field", "fieldDis";
// lines "border0" (structure: splitters, cards, the title bar's edge),
// "border1" (inputs), "border2" (separators, button borders); text "text",
// "text2", "textDis", "onAccent", "link", "headerText"; "accent",
// "accentHover", "accentPress", "focus", "fieldFocus" (a focused field's
// border), "fieldRing" (with the outside focus style, the halo round a
// focused field in place of the focus ring: Geist's); buttons "btn",
// "btnHover", "btnPress", "btnBorder", "btnText", "primary",
// "primaryHover", "primaryPress", "primaryBorder", "onPrimary"; "wash" (the
// neutral hover wash, translucent), "sel" (row selection), "menuHover",
// "menuText", "rowBar"; "tip", "tipText", "tipBorder"; "checkOff",
// "checkOn", "checkMark", "checkBorder"; "switchOff", "switchOffBorder",
// "switchOn", "knobOff", "knobOn", "knobBorder" (a ring round the knob,
// the accent when on); "track", "trackFill", "thumbFill", "thumbBorder",
// "progress", "progressTrack"; "scrollThumb", "scrollThumbHot",
// "scrollTrack"; "tabLine", "tabText", "tabText2", "tabSep" (between
// browser tabs), "segTrack", "segOn", "segOnBorder", "segOnText";
// "headerLine"; "captionHover", "close", "onClose"; "danger", "success",
// "warning".
type webEngine struct{ BaseEngine }

func init() {
	RegisterEngine(webEngine{})
	for _, p := range webPacks() {
		RegisterPack(p)
	}
}

func (webEngine) ID() string { return "web" }

// DefaultMetrics are a neutral web look at 14px text (Radix Themes' sizes):
// 32px controls, rows and menu items, 16px check boxes, 6px corners. The
// packs set their own in their own pixels.
func (webEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		FontSize: 14, ViewFrame: 1,
		Radius: 6, RadiusSmall: 4,
		ControlH: 32, FieldH: 32, ComboH: 32,
		Checkbox: 16, Radio: 16,
		MenuItemH: 32, MenuBarH: 32, TabH: 36, RowH: 32,
		TitleBar: 44, HeaderH: 32, ProgressH: 12, SliderH: 20, Thumb: 16,
		Scroll: 10, Pad: 12, FieldPad: 10, FocusWidth: 2, Border: 1,
		ToolBarH: 40, StatusBarH: 28, SpinnerW: 22, SwitchW: 36, SwitchH: 20,
	}
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

// Row kinds: plain lists and trees, sidebars, tables.
const (
	webList = iota
	webSide
	webTable
)

// webRows is how one kind of item view marks its rows.
type webRows struct {
	sel, selHover, selOff, hover float32
	offWash                      bool
	inset, gap, radius           float32
}

// webShadow is one drop shadow recipe (1× values).
type webShadow struct {
	a, dy, blur, spread float32
}

// webSet is a look's resolved colours, shapes and faces, built once per look.
type webSet struct {
	dark bool

	window, titleBar, toolBar, subPanel, sidebar, card, dialog, header, tabPane paintengine2d.Color
	popup, popupBorder, windowBorder, field, fieldDis                           paintengine2d.Color
	border0, border1, border2                                                   paintengine2d.Color
	text, text2, textDis, onAccent, link, headerText                            paintengine2d.Color
	accent, accentHover, accentPress, focus, fieldFocus, fieldRing              paintengine2d.Color
	btn, btnHover, btnPress, btnBorder, btnText                                 paintengine2d.Color
	primary, primaryHover, primaryPress, primaryBorder, onPrimary               paintengine2d.Color
	wash, sel, menuHover, menuText, rowBar                                      paintengine2d.Color
	tip, tipText, tipBorder                                                     paintengine2d.Color
	checkOff, checkOn, checkMark, checkBorder                                   paintengine2d.Color
	switchOff, switchOffBorder, switchOn, knobOff, knobOn, knobBorder           paintengine2d.Color
	track, trackFill, thumbFill, thumbBorder, progress, progressTrack           paintengine2d.Color
	scrollThumb, scrollThumbHot, scrollTrack                                    paintengine2d.Color
	tabLine, tabText, tabText2, segTrack, segOn, segOnText                      paintengine2d.Color
	captionHover, close, onClose, danger, success, warning                      paintengine2d.Color

	radius, fieldR, comboR, overlayR, tipR, cardR, viewR, windowR float32
	checkR, menuRowR, switchR, menuInset                          float32
	hair, restShadow, pressScale, disA                            float32
	focusStyle                                                    int
	focusW, focusGap, focusA                                      float32
	hoverBorder, checkFocus, primaryFirst                         bool
	primaryStyle, tabStyle                                        int
	tabLineW, tabLineGap                                          float32
	tabFit, tabHover, tabBar                                      bool
	checkStyle, radioStyle, thumbStyle                            int
	checkSize, radioSize, radioDot, knob, trackH, progressH       float32
	pill, menuAccent, menuSepLabel, triangles                     bool
	rows                                                          [3]webRows
	rowBarW                                                       float32
	scrollIdle, scrollInset                                       float32
	scrollArrows                                                  bool
	captionStyle                                                  int
	captionBtn                                                    float32
	captionCenter, captionLine, toolWash, headerCenter            bool
	toolAlpha                                                     float32
	seg, accordion                                                int
	shadows                                                       [3]webShadow
	shadowK                                                       float32
	takesAccent                                                   bool

	bold, btnFace, tabFace, tabSelFace, headFace, tipFace, smallFace *Font
}

type webKey struct{}

// webColors is the look's resolved set (built once per look).
func webColors(l *Classic) *webSet {
	return l.Memo(webKey{}, func() any { return webBuild(l) }).(*webSet)
}

func webBuild(l *Classic) *webSet {
	pal := l.palette
	x, p := l.X, l.P
	c := &webSet{dark: Luma(pal.Background) < 0.5}
	clear := paintengine2d.Color{}

	// Surfaces and lines.
	c.window = x("window", pal.Background)
	c.field = x("field", pal.Field)
	c.text = x("text", pal.Text)
	c.text2 = x("text2", pal.TextMuted)
	c.textDis = x("textDis", Mix(c.window, c.text, 0.4))
	c.titleBar = x("titleBar", c.window)
	c.toolBar = x("toolBar", c.window)
	c.subPanel = x("subPanel", c.window)
	c.sidebar = x("sidebar", c.window)
	c.card = x("card", c.window)
	c.popup = x("popup", pal.SurfaceAlt)
	c.dialog = x("dialog", c.popup)
	c.header = x("header", c.field)
	c.tabPane = x("tabPane", c.window)
	c.border0 = x("border0", pal.Divider)
	c.border1 = x("border1", pal.FieldBorder)
	c.border2 = x("border2", pal.Divider)
	c.popupBorder = x("popupBorder", c.border2)
	c.windowBorder = x("windowBorder", c.border1)
	c.fieldDis = x("fieldDis", c.window)
	c.headerText = x("headerText", c.text2)

	// Accent and focus.
	c.accent = x("accent", pal.Accent)
	c.accentHover = x("accentHover", pal.AccentHover)
	c.accentPress = x("accentPress", pal.AccentPress)
	c.onAccent = x("onAccent", pal.TextOnAccent)
	c.link = x("link", c.accent)
	c.focus = x("focus", pal.Focus)
	c.fieldFocus = x("fieldFocus", c.accent)
	c.fieldRing = x("fieldRing", clear)

	// Buttons.
	c.btn = x("btn", c.field)
	c.btnHover = x("btnHover", Mix(c.btn, c.text, 0.04))
	c.btnPress = x("btnPress", Mix(c.btn, c.text, 0.08))
	c.btnBorder = x("btnBorder", c.border2)
	c.btnText = x("btnText", c.text)
	c.primaryStyle = int(p("primaryStyle", 0))
	c.primary, c.primaryHover, c.primaryPress, c.onPrimary = c.accent, c.accentHover, c.accentPress, c.onAccent
	if c.primaryStyle == 1 {
		c.primary, c.onPrimary = c.text, c.window
		c.primaryHover, c.primaryPress = Mix(c.text, c.window, 0.1), Mix(c.text, c.window, 0.18)
	}
	c.primary = x("primary", c.primary)
	c.primaryHover = x("primaryHover", c.primaryHover)
	c.primaryPress = x("primaryPress", c.primaryPress)
	c.onPrimary = x("onPrimary", c.onPrimary)
	c.primaryBorder = x("primaryBorder", clear)

	// Washes, selections, menus.
	c.wash = x("wash", c.text.WithAlpha(0.08))
	c.sel = x("sel", c.accent)
	c.menuAccent = p("menuHighlight", 0) == 1
	if c.menuAccent {
		c.menuHover, c.menuText = x("menuHover", c.accent), x("menuText", c.onAccent)
	} else {
		c.menuHover, c.menuText = x("menuHover", c.wash), x("menuText", c.text)
	}
	c.rowBar = x("rowBar", c.accent)
	c.tip = x("tip", c.popup)
	c.tipText = x("tipText", c.text)
	c.tipBorder = x("tipBorder", c.popupBorder)

	// Toggles and ranges.
	c.checkStyle = int(p("checkStyle", 1))
	c.checkOff = x("checkOff", c.field)
	c.checkOn = x("checkOn", c.accent)
	mark := c.onAccent
	if c.checkStyle == 0 {
		mark = c.accent
	}
	c.checkMark = x("checkMark", mark)
	c.checkBorder = x("checkBorder", c.border1)
	c.switchOn = x("switchOn", c.accent)
	c.switchOff = x("switchOff", Mix(c.window, c.text, 0.18))
	c.switchOffBorder = x("switchOffBorder", clear)
	c.knobOn = x("knobOn", c.onAccent)
	c.knobOff = x("knobOff", c.field)
	c.knobBorder = x("knobBorder", clear)
	c.track = x("track", c.text.WithAlpha(0.2))
	c.trackFill = x("trackFill", c.accent)
	c.thumbFill = x("thumbFill", c.accent)
	c.thumbBorder = x("thumbBorder", clear)
	c.progress = x("progress", c.accent)
	c.progressTrack = x("progressTrack", c.text.WithAlpha(0.12))
	c.scrollThumb = x("scrollThumb", c.text2)
	c.scrollThumbHot = x("scrollThumbHot", c.text)
	c.scrollTrack = x("scrollTrack", clear)

	// Tabs and segments.
	c.tabLine = x("tabLine", c.accent)
	c.tabText = c.text
	if p("tabAccent", 0) == 1 {
		c.tabText = c.accent
	}
	c.tabText = x("tabText", c.tabText)
	c.tabText2 = c.text2
	if d := p("tabDim", 0); d > 0 && d < 1 {
		c.tabText2 = c.text.WithAlpha(c.text.A * d)
	}
	c.tabText2 = x("tabText2", c.tabText2)
	c.segTrack = x("segTrack", c.wash)
	c.segOn = x("segOn", c.window)
	c.segOnText = x("segOnText", c.text)

	// Window chrome and status.
	c.captionHover = x("captionHover", paintengine2d.RGBA(0, 0, 0, 0.25))
	c.close = x("close", Hex("#c42b1c"))
	c.onClose = x("onClose", paintengine2d.RGB(1, 1, 1))
	c.danger, c.success, c.warning = x("danger", pal.Danger), x("success", pal.Success), x("warning", pal.Warning)

	// Shapes.
	c.radius = p("radius", 6)
	c.fieldR = p("fieldRadius", c.radius)
	c.comboR = p("comboRadius", c.radius)
	c.overlayR = p("overlayRadius", 8)
	c.tipR = p("tipRadius", c.radius)
	c.cardR = p("cardRadius", 8)
	c.viewR = p("viewRadius", c.radius)
	c.windowR = p("windowRadius", c.overlayR)
	c.checkR = p("checkRadius", 4)
	c.switchR = p("switchRadius", c.radius)
	c.menuInset = p("menuInset", 4)
	c.hair = max(p("hairline", 1), 0.5)
	c.restShadow = p("restShadow", 0)
	c.pressScale = p("pressScale", 1)
	c.disA = p("disabledAlpha", 0.5)
	c.focusStyle = int(p("focusStyle", webFocusInside))
	c.focusW = p("focusWidth", 2)
	c.focusGap = p("focusGap", 0)
	c.focusA = p("focusAlpha", 1)
	c.hoverBorder = p("hoverBorder", 0) != 0
	c.checkFocus = p("checkFocus", 0) != 0
	c.primaryFirst = p("primaryFirst", 0) != 0
	c.tabStyle = int(p("tabStyle", webTabUnderline))
	c.tabLineW = p("tabLine", 2)
	c.tabLineGap = p("tabLineGap", 0)
	c.tabFit = p("tabFit", 0) != 0
	c.tabHover = p("tabHover", 0) != 0
	c.tabBar = p("tabBar", 1) != 0
	c.checkSize = p("checkSize", 16)
	c.radioSize = p("radioSize", c.checkSize)
	c.radioStyle = int(p("radioStyle", 1))
	c.radioDot = p("radioDot", 6)
	c.pill = p("switchShape", 0) == 0
	c.knob = p("knob", 0)
	c.trackH = p("trackH", 4)
	c.thumbStyle = int(p("thumbStyle", 0))
	c.progressH = p("progressH", 4)
	c.menuSepLabel = p("menuSep", 0) == 1
	rowR := p("rowRadius", 6)
	c.menuRowR = p("menuRowRadius", rowR)
	gap := p("rowGap", 0)
	for i, k := range [3]string{"list", "side", "table"} {
		r := &c.rows[i]
		r.sel = p(k+"Sel", 1)
		r.selHover = p(k+"SelHover", r.sel)
		r.selOff = p(k+"SelOff", r.sel)
		r.offWash = p(k+"OffWash", 0) != 0
		r.hover = p(k+"Hover", 1)
		r.radius, r.gap = rowR, gap
	}
	c.rows[webList].inset = p("rowInset", 0)
	c.rows[webSide].inset = p("sideInset", 8)
	c.rows[webSide].radius = p("sideRadius", rowR)
	c.rows[webTable].radius, c.rows[webTable].gap = 0, 0
	c.rowBarW = p("rowBar", 0)
	c.triangles = p("expander", 1) == 0
	c.scrollIdle = p("scrollIdle", 2)
	c.scrollInset = p("scrollInset", 2)
	c.scrollArrows = p("scrollArrows", 0) != 0
	c.captionStyle = int(p("captionStyle", 1))
	c.captionBtn = p("captionButton", 46)
	strip := float32(0)
	if c.captionStyle == 0 {
		strip = 1
	}
	c.captionCenter = p("captionCenter", strip) != 0
	c.captionLine = p("captionLine", strip) != 0
	c.toolWash = p("toolWash", 1) != 0
	c.toolAlpha = p("toolAlpha", 1)
	c.seg = int(p("seg", 1))
	c.headerCenter = p("headerCenter", 0) != 0
	c.accordion = int(p("accordion", 1))
	c.shadowK = p("shadow", 1)
	sh := func(k string, a, dy, blur, spread float32) webShadow {
		return webShadow{a: p(k, a), dy: p(k+"Y", dy), blur: p(k+"Blur", blur), spread: p(k+"Spread", spread)}
	}
	c.shadows[PopupMenu] = sh("menuShadow", 0.16, 4, 16, 0)
	c.shadows[PopupTooltip] = sh("tipShadow", 0.12, 2, 8, 0)
	c.shadows[PopupDialog] = sh("dialogShadow", 0.25, 12, 40, 0)
	c.takesAccent = p("accentFollows", 0) != 0

	// Faces.
	c.bold = l.BoldFont()
	c.btnFace = l.WeightFont(Weight(p("buttonWeight", 500)))
	tw := Weight(p("tabWeight", 500))
	if grow := p("tabGrow", 0); grow != 0 {
		c.tabFace = BakeFamily(l.UIFamily(), tw, l.metrics.FontSize+l.S(grow), pal.Text)
	} else {
		c.tabFace = l.WeightFont(tw)
	}
	c.tabSelFace = c.tabFace
	if sw := Weight(p("tabSelWeight", float32(tw))); sw != tw {
		c.tabSelFace = BakeFamily(l.UIFamily(), sw, l.metrics.FontSize+l.S(p("tabGrow", 0)), pal.Text)
	}
	c.headFace = l.WeightFont(Weight(p("headerWeight", 600)))
	c.tipFace = l.body
	if ts := p("tipSize", 0); ts > 0 {
		c.tipFace = BakeFamily(l.UIFamily(), WeightRegular, l.S(ts), pal.Text)
	}
	c.smallFace = BakeFamily(l.UIFamily(), WeightRegular, max(l.metrics.FontSize-l.S(1), 6), pal.Text)
	return c
}

// ---- helpers ------------------------------------------------------------------------------------

// webA is c with its alpha multiplied by a.
func webA(c paintengine2d.Color, a float32) paintengine2d.Color { return c.WithAlpha(c.A * a) }

// flat is col flattened over the window colour.
func (c *webSet) flat(col paintengine2d.Color) paintengine2d.Color { return webOver(c.window, col) }

// webOver is col flattened over the opaque base.
func webOver(base, col paintengine2d.Color) paintengine2d.Color {
	if col.A >= 1 {
		return col
	}
	return Mix(base, col, col.A)
}

// rad is a corner radius at display scale, 0 when the Corners pref squares
// the look.
func (c *webSet) rad(l *Classic, v float32) float32 {
	if v <= 0 || l.square() {
		return 0
	}
	return l.S(v)
}

// px is the hairline in device pixels (whole, at least one).
func (c *webSet) px(l *Classic) float32 {
	return max(snap(l.S(c.hair)), 1)
}

// ring is the focus ring's gap and width in device pixels.
func (c *webSet) ring(l *Classic) (gap, w float32) {
	return snap(l.S(c.focusGap)), max(snap(l.S(c.focusW)), 1)
}

// reach is the margin every control keeps round its face for a focus ring
// outside it (zero for the other focus styles).
func (c *webSet) reach(l *Classic) float32 {
	if c.focusStyle != webFocusOutside {
		return 0
	}
	g, w := c.ring(l)
	return g + w
}

// face is a control's visible body inside its rect b: b on whole pixels,
// less the focus margin.
func (c *webSet) face(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = winSnap(b)
	m := c.reach(l)
	if m <= 0 || b.Dx() < 4*m || b.Dy() < 3*m {
		return b
	}
	return b.Inset(m)
}

// pressed shrinks a pressed face by the pack's press scale (SourceGit's
// 98%), about its centre.
func (c *webSet) pressed(f paintengine2d.Rect, st ControlState) paintengine2d.Rect {
	if c.pressScale >= 1 || c.pressScale <= 0 || !st.Pressed() || st.Disabled() {
		return f
	}
	dx, dy := f.Dx()*(1-c.pressScale)*0.5, f.Dy()*(1-c.pressScale)*0.5
	return paintengine2d.Rect{Min: paintengine2d.Pt(f.Min.X+dx, f.Min.Y+dy), Max: paintengine2d.Pt(f.Max.X-dx, f.Max.Y-dy)}
}

// box fills the round rect b and strokes its hairline just inside it.
func (c *webSet) box(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, fill, border paintengine2d.Color) {
	if b.Dx() < 1 || b.Dy() < 1 {
		return
	}
	r = min(r, b.Dx()*0.5, b.Dy()*0.5)
	if fill.A > 0 {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
	}
	if border.A > 0 {
		winRing(ctx, b, r, c.px(l), paintengine2d.Fill(border))
	}
}

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

// webDots is Avalonia's focus adorner: a dotted rectangle along the edge of
// b, dots of one line every three (the 1,2 dash), all in one path.
func webDots(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, lw float32) {
	b = winSnap(b)
	if b.Dx() < 2*lw || b.Dy() < 2*lw || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	step := 3 * lw
	for x := b.Min.X; x+lw <= b.Max.X+0.01; x += step {
		p.AddRect(paintengine2d.XYWH(x, b.Min.Y, lw, lw))
		p.AddRect(paintengine2d.XYWH(x, b.Max.Y-lw, lw, lw))
	}
	for y := b.Min.Y + step; y+lw <= b.Max.Y-lw+0.01; y += step {
		p.AddRect(paintengine2d.XYWH(b.Min.X, y, lw, lw))
		p.AddRect(paintengine2d.XYWH(b.Max.X-lw, y, lw, lw))
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// focusRing marks keyboard focus on a control whose face is f (corner r)
// inside its rect b: a ring round the face in the margin the control keeps
// (outside), a band over the face's edge (inside), or the dotted adorner.
func (c *webSet) focusRing(l *Classic, ctx *paintengine2d.Context, b, f paintengine2d.Rect, r float32) {
	col := webA(c.focus, c.focusA)
	switch c.focusStyle {
	case webFocusOutside:
		g, w := c.ring(l)
		bb := winSnap(b)
		m := g + w - 0.01
		if f.Min.X-bb.Min.X < m || bb.Max.X-f.Max.X < m || f.Min.Y-bb.Min.Y < m || bb.Max.Y-f.Max.Y < m {
			// No room round the face (a tool bar's button): over its edge.
			if f.Dx() > 2*w && f.Dy() > 2*w {
				winRing(ctx, f, min(r, f.Dx()*0.5, f.Dy()*0.5), w, paintengine2d.Fill(col))
			}
			return
		}
		inner := f.Inset(-g)
		outer := f.Inset(-(g + w))
		if r > 0 {
			webBand(ctx, outer, r+g+w, inner, r+g, col)
		} else {
			webBand(ctx, outer, 0, inner, 0, col)
		}
	case webFocusDotted:
		webDots(ctx, f, c.text, winPx(l))
	default:
		_, w := c.ring(l)
		if f.Dx() > 2*w && f.Dy() > 2*w {
			winRing(ctx, f, min(r, f.Dx()*0.5, f.Dy()*0.5), w, paintengine2d.Fill(col))
		}
	}
}

// fieldFocusRing marks a focused text field, combo or text area: with a
// "fieldRing" colour, a halo round the face (inside it the focused border
// darkens) where the CSS draws one ring outside the element's own; else the
// focus ring.
func (c *webSet) fieldFocusRing(l *Classic, ctx *paintengine2d.Context, b, f paintengine2d.Rect, r float32) {
	if c.fieldRing.A <= 0 || c.focusStyle != webFocusOutside {
		c.focusRing(l, ctx, b, f, r)
		return
	}
	bb := winSnap(b)
	m := c.reach(l) - 0.01
	if f.Min.X-bb.Min.X < m || bb.Max.X-f.Max.X < m || f.Min.Y-bb.Min.Y < m || bb.Max.Y-f.Max.Y < m {
		return // no room round the face: its focused border says it
	}
	// The element's 1px ring is the face's border, so the halo reaches one
	// hairline less than the whole margin.
	if w := c.reach(l) - c.px(l); w > 0 {
		webBand(ctx, f.Inset(-w), r+w, f, r, c.fieldRing)
	}
}

// webTick strokes the check mark filling g.
func webTick(ctx *paintengine2d.Context, g paintengine2d.Rect, col paintengine2d.Color, w float32) {
	if g.Empty() || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	p.MoveTo(g.Min.X+g.Dx()*0.18, g.Min.Y+g.Dy()*0.53)
	p.LineTo(g.Min.X+g.Dx()*0.41, g.Min.Y+g.Dy()*0.76)
	p.LineTo(g.Min.X+g.Dx()*0.84, g.Min.Y+g.Dy()*0.27)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// webChevron strokes a thin chevron w wide and h deep pointing dir, centred
// in b (combos, submenus, spinners, sort marks, tree expanders).
func webChevron(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, w, h, lw float32, col paintengine2d.Color) {
	if dir == DirLeft || dir == DirRight {
		w, h = h, w
	}
	if b.Dx() < 2 || b.Dy() < 2 || col.A <= 0 {
		return
	}
	if k := min(b.Dx()/w, b.Dy()/h); k < 1 {
		w, h = w*k, h*k
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-w*0.5, cy+h*0.5)
		p.LineTo(cx, cy-h*0.5)
		p.LineTo(cx+w*0.5, cy+h*0.5)
	case DirDown:
		p.MoveTo(cx-w*0.5, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
		p.LineTo(cx+w*0.5, cy-h*0.5)
	case DirLeft:
		p.MoveTo(cx+w*0.5, cy-h*0.5)
		p.LineTo(cx-w*0.5, cy)
		p.LineTo(cx+w*0.5, cy+h*0.5)
	default:
		p.MoveTo(cx-w*0.5, cy-h*0.5)
		p.LineTo(cx+w*0.5, cy)
		p.LineTo(cx-w*0.5, cy+h*0.5)
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: lw, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// webTriangle fills a small solid triangle s across pointing dir, centred in
// b (SourceGit's tree expanders and scroll arrows).
func webTriangle(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, s float32, col paintengine2d.Color) {
	s = min(s, b.Dx(), b.Dy())
	if s < 2 || col.A <= 0 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	h := s * 0.5
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-h, cy+h*0.5)
		p.LineTo(cx+h, cy+h*0.5)
		p.LineTo(cx, cy-h*0.5)
	case DirDown:
		p.MoveTo(cx-h, cy-h*0.5)
		p.LineTo(cx+h, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
	case DirLeft:
		p.MoveTo(cx+h*0.5, cy-h)
		p.LineTo(cx+h*0.5, cy+h)
		p.LineTo(cx-h*0.5, cy)
	default:
		p.MoveTo(cx-h*0.5, cy-h)
		p.LineTo(cx-h*0.5, cy+h)
		p.LineTo(cx+h*0.5, cy)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// label is the colour that reads on bg: the text colour when it does, else
// the text on the accent, else black or white.
func (c *webSet) label(bg paintengine2d.Color) paintengine2d.Color {
	return ReadableOn(bg, 4.5, c.text, c.onAccent)
}

// disabled fades a colour of a disabled control.
func (c *webSet) disabled(col paintengine2d.Color, st ControlState) paintengine2d.Color {
	if st.Disabled() {
		return webA(col, c.disA)
	}
	return col
}

// shadow is the drop shadow of kind at display scale.
func (c *webSet) shadow(l *Classic, kind PopupKind) (webShadow, bool) {
	if int(kind) >= len(c.shadows) || c.shadowK <= 0 {
		return webShadow{}, false
	}
	s := c.shadows[kind]
	if s.a <= 0 {
		return webShadow{}, false
	}
	return webShadow{a: min(s.a*c.shadowK, 1), dy: l.S(s.dy), blur: l.S(s.blur), spread: l.S(s.spread)}, true
}

// bool2f is 1 for true.
func bool2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
