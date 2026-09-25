package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// metroEngine paints the flat Windows of Windows 8 (2012) and Windows 10
// (2015), from the documented part and state names of their desktop visual
// style, the Windows 8 and 10 design guidance and the shipped control
// colours, as numbers. Everything is flat: 1px (or 2px) borders, no
// gradients, no rounded corners.
//
//   - push buttons are #e1e1e1 in a 1px #adadad border; hot is a pale blue
//     in the accent border, pressed a deeper blue, the default button
//     carries a 2px accent border;
//   - check boxes are 13px squares in a dark 1px border with a thin tick;
//     radios are circles with a solid dot; hot turns border and mark to the
//     accent, pressed fills them pale blue;
//   - edit boxes are white in a #7a7a7a line that turns accent on focus;
//   - progress bars fill with Windows' green, flat;
//   - menus are light grey with a solid blue hot row, list selection is
//     Explorer's box.
//
// Windows 8 keeps the Windows 7 layout in this flat dress: dotted focus
// rectangles, 17px scroll bars with their arrow buttons always shown, the
// pointed trackbar thumb, a rectangular toggle switch, solid blue list
// selection and coloured window frames with a centred title.
//
// Windows 10 adds the accent-coloured focus (a 2px accent border on buttons
// and combo boxes, an accent line round labels), thin scroll bars that show
// a slim indicator until the pointer is over them and then their track and
// arrow buttons, the pill toggle switch, the slider's rectangular accent
// thumb on a 2px track, Explorer's light accent selection, white title bars
// and a red close button on hover. Windows 10 Dark (the 2018 dark mode) is
// the same shapes on dark greys.
//
// Pack data: params "scheme" 0 = Windows 10, 1 = Windows 8, 2 = Windows 10
// Dark; every colour key of metroWin10 can be overridden through "extra".
type metroEngine struct{ BaseEngine }

func init() {
	RegisterEngine(metroEngine{})
	for _, p := range metroPacks() {
		RegisterPack(p)
	}
}

func (metroEngine) ID() string { return "metro" }

// DefaultMetrics are the Windows 8 / 10 desktop proportions at the
// toolkit's 16px UI font: 23px buttons become 30, check boxes stay 13px,
// the toggle switch is 44×20, everything is square.
func (metroEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1, Square: true,
		ControlH: 30, FieldH: 28, ComboH: 28,
		Checkbox: 13, Radio: 13,
		MenuItemH: 26, MenuBarH: 26, TabH: 28, RowH: 24,
		TitleBar: 32, HeaderH: 26, ProgressH: 18, SliderH: 28, Thumb: 8,
		Scroll: 17, Pad: 10, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 26, SpinnerW: 17, SwitchW: 44, SwitchH: 20,
	}
}

// ---- colour tables -----------------------------------------------------------------------

// metroScheme maps a colour key to "#rrggbb". metroWin10 holds every key;
// the other schemes list what differs.
type metroScheme map[string]string

var metroWin10 = metroScheme{
	// Window face, text, accent (the Windows 10 default blue).
	"face": "#f0f0f0", "text": "#000000", "gray": "#6d6d6d", "disText": "#838383",
	"field": "#ffffff", "accent": "#0078d7", "accentLt": "#429ce3", "onAccent": "#ffffff",

	// Push buttons: fill and border per state; the default button's border.
	"btn": "#e1e1e1", "btnBorder": "#adadad", "btnHot": "#e5f1fb", "btnHotBorder": "#0078d7",
	"btnPress": "#cce4f7", "btnPressBorder": "#005499", "btnDis": "#cccccc", "btnDisBorder": "#bfbfbf",
	"btnDefBorder": "#0078d7",

	// Check boxes and radios: fill, border and mark per state.
	"chk": "#ffffff", "chkBorder": "#333333", "chkMark": "#333333",
	"chkHot": "#ffffff", "chkHotBorder": "#0078d7", "chkHotMark": "#0078d7",
	"chkPress": "#cce4f7", "chkPressBorder": "#005499", "chkPressMark": "#005499",
	"chkDis": "#ffffff", "chkDisBorder": "#cccccc", "chkDisMark": "#cccccc",

	// Edit boxes and item views.
	"edBorder": "#7a7a7a", "edHot": "#171717", "edFocus": "#0078d7", "edDisBorder": "#cccccc", "edDis": "#f0f0f0",
	"viewBorder": "#828790",

	// Scroll bars: track, thumb per state, arrow glyphs and buttons, the
	// thin indicator Windows 10 shows while the pointer is away.
	"sbTrack": "#f0f0f0", "sbThumb": "#c2c2c2", "sbThumbHot": "#a6a6a6", "sbThumbPress": "#606060",
	"sbGlyph": "#606060", "sbGlyphHot": "#000000", "sbGlyphPress": "#ffffff", "sbGlyphDis": "#bfbfbf",
	"sbBtnHot": "#dadada", "sbBtnPress": "#606060", "sbIndicator": "#8a8a8a",

	// Tabs and the page under them.
	"tab": "#f0f0f0", "tabHot": "#d8eaf9", "tabBorder": "#d9d9d9", "pane": "#ffffff",

	// Progress bar.
	"prog": "#06b025", "progTrack": "#e6e6e6", "progBorder": "#bcbcbc",

	// Slider: the 2px track and the accent thumb per state (Windows 10);
	// the groove of Windows 8's trackbar.
	"slTrack": "#999999", "slTrackHot": "#666666", "slThumb": "#0078d7", "slThumbHot": "#171717",
	"slThumbPress": "#cccccc", "slDis": "#cccccc", "groove": "#e7eaea", "grooveBorder": "#d6d6d6",

	// Toggle switch (off border / knob, hot, pressed, disabled).
	"swOff": "#333333", "swOffHot": "#000000", "swPress": "#666666", "swDis": "#cccccc", "swKnobOn": "#ffffff",

	// Explorer selection: hover, selected, selected + hot border, selected
	// in an unfocused view, the keyboard focus line.
	"hov": "#e5f3ff", "selFill": "#cce8ff", "selBorder": "#99d1ff", "off": "#d9d9d9", "focusBorder": "#99d1ff",

	// Menus and the menu bar.
	"menuBg": "#f2f2f2", "menuBorder": "#cccccc", "menuHot": "#91c9f7", "menuHotBorder": "#91c9f7",
	"menuSep": "#cccccc", "menuSepLt": "#f2f2f2", "gutterLine": "#f2f2f2", "gutterLineLt": "#f2f2f2",
	"menuDisText": "#6d6d6d",
	"mbar":        "#ffffff", "mbHot": "#e5f3ff", "mbHotBorder": "#cce8ff", "mbOpen": "#cce8ff", "mbOpenBorder": "#99d1ff",

	// Tree glyphs.
	"exp": "#a6a6a6", "expOpen": "#595959",

	// List-view header.
	"hdr": "#ffffff", "hdrDiv": "#e5e5e5", "hdrHot": "#d9ebf9", "hdrHotBorder": "#bcdcf4",
	"hdrPress": "#bcdcf4", "hdrPressBorder": "#7eb4ea", "hdrText": "#4c607a", "hdrArrow": "#a1a1a1",

	// Tool tip, group box, status bar, tool bar, separators.
	"tip": "#ffffff", "tipBorder": "#767676", "tipText": "#575757",
	"groupBorder": "#dcdcdc", "groupText": "#000000",
	"status": "#f0f0f0", "statTop": "#dadada", "statSep": "#dadada",
	"tb": "#f0f0f0", "tbBorder": "#dadada", "toolHot": "#e5f3ff", "toolHotBorder": "#cce8ff",
	"toolPress": "#cce8ff", "toolPressBorder": "#99d1ff",
	"etchDk": "#dadada", "etchLt": "#f0f0f0",

	// Window frame: title bar, border, caption text, close button.
	"frame": "#ffffff", "frameOff": "#ffffff", "frameBorder": "#7a7a7a", "frameBorderOff": "#aaaaaa",
	"capText": "#000000", "capTextOff": "#999999",
	"close": "#ffffff", "closeOff": "#ffffff", "closeHot": "#e81123", "closePress": "#f1707a",
	"closeGlyph": "#000000", "closeGlyphHot": "#ffffff", "closeGlyphOff": "#999999",

	// Message icons.
	"msgError": "#e81123", "msgInfo": "#0078d7", "msgWarn": "#fcc300", "msgWarnGlyph": "#000000",
}

// metroWin8 is Windows 8's desktop: the #3399ff highlight, light-blue hot
// states, Windows 7's menus and coloured window frames.
var metroWin8 = metroScheme{
	"accent": "#3399ff", "accentLt": "#7eb4ea",
	"btn": "#eaeaea", "btnBorder": "#acacac", "btnHot": "#e4f0fc", "btnHotBorder": "#7eb4ea",
	"btnPress": "#cfe6fc", "btnPressBorder": "#569de5", "btnDis": "#f4f4f4", "btnDisBorder": "#adb2b5",
	"btnDefBorder": "#3399ff",
	"chkBorder":    "#707070", "chkMark": "#212121", "chkHot": "#f3f9ff", "chkHotBorder": "#3399ff", "chkHotMark": "#212121",
	"chkPress": "#d9ecff", "chkPressBorder": "#007acc", "chkPressMark": "#212121", "chkDisBorder": "#bcbcbc", "chkDisMark": "#bcbcbc",
	"edBorder": "#abadb3", "edHot": "#7eb4ea", "edFocus": "#569de5", "edDisBorder": "#d9d9d9",
	"sbTrack": "#e8e8ec", "sbThumb": "#cdcdcd", "sbIndicator": "#cdcdcd",
	"tab": "#f0f0f0", "tabHot": "#e4f0fc", "tabBorder": "#acacac",
	"groove": "#e7eaea", "grooveBorder": "#d6d6d6",
	"swOff": "#333333", "swOffHot": "#000000", "swKnobOn": "#000000",
	"hov": "#e5f3ff", "selFill": "#3399ff", "selBorder": "#3399ff", "off": "#d9d9d9", "focusBorder": "#000000",
	"menuBg": "#f0f0f0", "menuBorder": "#979797", "menuHot": "#d5e1f2", "menuHotBorder": "#a3bde3",
	"menuSep": "#e0e0e0", "menuSepLt": "#ffffff", "gutterLine": "#e2e3e3", "gutterLineLt": "#ffffff",
	"mbar": "#f5f6f7", "mbHot": "#d5e7f8", "mbHotBorder": "#7eb4ea", "mbOpen": "#b8d6f5", "mbOpenBorder": "#569de5",
	"hdrHot": "#d9ebf9", "hdrHotBorder": "#bcdcf4", "hdrPress": "#bcdcf4", "hdrPressBorder": "#569de5",
	"status": "#f0f0f0", "statTop": "#d7d7d7", "statSep": "#d7d7d7",
	"toolHot": "#e4f0fc", "toolHotBorder": "#7eb4ea", "toolPress": "#cfe6fc", "toolPressBorder": "#569de5",
	"etchDk": "#a0a0a0", "etchLt": "#ffffff",
	"frame": "#6ba5e7", "frameOff": "#ebebeb", "frameBorder": "#4f89c8", "frameBorderOff": "#bcbcbc",
	"capText": "#000000", "capTextOff": "#6d6d6d",
	"close": "#c75050", "closeOff": "#bcbcbc", "closeHot": "#e04343", "closePress": "#993d3d",
	"closeGlyph": "#ffffff", "closeGlyphHot": "#ffffff", "closeGlyphOff": "#ffffff",
	"msgError": "#e33e2c", "msgInfo": "#3399ff",
}

// metroWin10Dark is Windows 10's dark mode (2018).
var metroWin10Dark = metroScheme{
	"face": "#202020", "text": "#ffffff", "gray": "#9a9a9a", "disText": "#6d6d6d",
	"field": "#191919", "accentLt": "#429ce3",
	"btn": "#333333", "btnBorder": "#4d4d4d", "btnHot": "#333d47", "btnHotBorder": "#0078d7",
	"btnPress": "#1c3f5e", "btnPressBorder": "#429ce3", "btnDis": "#2b2b2b", "btnDisBorder": "#3d3d3d",
	"chk": "#191919", "chkBorder": "#cccccc", "chkMark": "#ffffff",
	"chkHot": "#191919", "chkHotBorder": "#429ce3", "chkHotMark": "#429ce3",
	"chkPress": "#1c3f5e", "chkPressBorder": "#0078d7", "chkPressMark": "#ffffff",
	"chkDis": "#191919", "chkDisBorder": "#5d5d5d", "chkDisMark": "#5d5d5d",
	"edBorder": "#9a9a9a", "edHot": "#cccccc", "edFocus": "#0078d7", "edDisBorder": "#444444", "edDis": "#262626",
	"viewBorder": "#444444",
	"sbTrack":    "#171717", "sbThumb": "#4d4d4d", "sbThumbHot": "#7a7a7a", "sbThumbPress": "#a6a6a6",
	"sbGlyph": "#999999", "sbGlyphHot": "#ffffff", "sbGlyphPress": "#000000", "sbGlyphDis": "#4d4d4d",
	"sbBtnHot": "#373737", "sbBtnPress": "#a6a6a6", "sbIndicator": "#7a7a7a",
	"tab": "#202020", "tabHot": "#2d3a47", "tabBorder": "#3d3d3d", "pane": "#191919",
	"progTrack": "#333333", "progBorder": "#555555",
	"slTrack": "#7a7a7a", "slTrackHot": "#a6a6a6", "slThumbHot": "#f2f2f2", "slThumbPress": "#767676", "slDis": "#444444",
	"groove": "#333333", "grooveBorder": "#555555",
	"swOff": "#cccccc", "swOffHot": "#ffffff", "swPress": "#999999", "swDis": "#5d5d5d",
	"hov": "#2d2d2d", "selFill": "#0d4878", "selBorder": "#1f7fd9", "off": "#3d3d3d", "focusBorder": "#429ce3",
	"menuBg": "#2b2b2b", "menuBorder": "#414141", "menuHot": "#414141", "menuHotBorder": "#414141",
	"menuSep": "#555555", "menuSepLt": "#2b2b2b", "gutterLine": "#2b2b2b", "gutterLineLt": "#2b2b2b",
	"menuDisText": "#6d6d6d",
	"mbar":        "#202020", "mbHot": "#2d2d2d", "mbHotBorder": "#3d3d3d", "mbOpen": "#414141", "mbOpenBorder": "#555555",
	"exp": "#8a8a8a", "expOpen": "#cccccc",
	"hdr": "#191919", "hdrDiv": "#3d3d3d", "hdrHot": "#2d2d2d", "hdrHotBorder": "#3d3d3d",
	"hdrPress": "#3d3d3d", "hdrPressBorder": "#555555", "hdrText": "#cccccc", "hdrArrow": "#8a8a8a",
	"tip": "#2b2b2b", "tipBorder": "#767676", "tipText": "#ffffff",
	"groupBorder": "#3d3d3d", "groupText": "#ffffff",
	"status": "#202020", "statTop": "#3d3d3d", "statSep": "#3d3d3d",
	"tb": "#202020", "tbBorder": "#3d3d3d", "toolHot": "#2d2d2d", "toolHotBorder": "#3d3d3d",
	"toolPress": "#0d4878", "toolPressBorder": "#1f7fd9",
	"etchDk": "#3d3d3d", "etchLt": "#202020",
	"frame": "#2b2b2b", "frameOff": "#2b2b2b", "frameBorder": "#555555", "frameBorderOff": "#3d3d3d",
	"capText": "#ffffff", "capTextOff": "#8a8a8a",
	"close": "#2b2b2b", "closeOff": "#2b2b2b", "closeGlyph": "#ffffff", "closeGlyphOff": "#8a8a8a",
	"msgError": "#ff4a4a", "msgInfo": "#429ce3", "msgWarn": "#fcc300",
}

var metroSchemes = []metroScheme{nil, metroWin8, metroWin10Dark}

// ---- resolved colour set ----------------------------------------------------------------------

// metroSet is a look's resolved Metro colours, built once per look.
type metroSet struct {
	win10 bool

	face, text, gray, disText, field, accent, accentLt, onAccent paintengine2d.Color
	selText                                                      paintengine2d.Color

	btn [5][2]paintengine2d.Color // normal, hot, pressed, disabled, default: fill, border
	chk [4][3]paintengine2d.Color // normal, hot, pressed, disabled: fill, border, mark
	ed  [4]paintengine2d.Color    // border: normal, hot, focused, disabled

	edDis, viewBorder paintengine2d.Color

	sbTrack, sbBtnHot, sbBtnPress, sbIndicator paintengine2d.Color
	sbThumb                                    [3]paintengine2d.Color // normal, hot, pressed
	sbGlyph                                    [4]paintengine2d.Color // normal, hot, pressed, disabled

	tab, tabHot, tabBorder, pane paintengine2d.Color
	prog, progTrack, progBorder  paintengine2d.Color

	slTrack, slTrackHot, slDis, groove, grooveBorder paintengine2d.Color
	slThumb                                          [3]paintengine2d.Color // normal, hot, pressed

	swOff, swOffHot, swPress, swDis, swKnobOn paintengine2d.Color

	hov, selFill, selBorder, off, focusBorder paintengine2d.Color
	selSolid                                  bool // Windows 8: the solid highlight with white text
	selFillText, offText                      paintengine2d.Color

	menuBg, menuBorder, menuHot, menuHotBorder, menuSep, menuSepLt paintengine2d.Color
	gutterLine, gutterLineLt, menuDisText, menuHotText             paintengine2d.Color
	mbar, mbHot, mbHotBorder, mbOpen, mbOpenBorder                 paintengine2d.Color

	exp, expOpen paintengine2d.Color

	hdr, hdrDiv, hdrHot, hdrHotBorder, hdrPress, hdrPressBorder, hdrText, hdrArrow paintengine2d.Color

	tip, tipBorder, tipText, groupBorder, groupText paintengine2d.Color
	status, statTop, statSep                        paintengine2d.Color
	tb, tbBorder, toolHot, toolHotBorder            paintengine2d.Color
	toolPress, toolPressBorder, etchDk, etchLt      paintengine2d.Color

	frame, frameOff, frameBorder, frameBorderOff, capText, capTextOff paintengine2d.Color
	close, closeOff, closeHot, closePress                             paintengine2d.Color
	closeGlyph, closeGlyphHot, closeGlyphOff                          paintengine2d.Color

	msgError, msgInfo, msgWarn, msgWarnGlyph paintengine2d.Color
}

type metroKey struct{}

// metroColors is the look's resolved colour set (built once per look).
func metroColors(l *Classic) *metroSet {
	return l.Memo(metroKey{}, func() any { return metroBuild(l) }).(*metroSet)
}

// metroSchemeIndex is the pack's scheme: 0 Windows 10, 1 Windows 8, 2
// Windows 10 Dark (the default for a dark pack without the param).
func metroSchemeIndex(l *Classic) int {
	def := float32(0)
	if Luma(l.palette.Background) < 0.5 {
		def = 2 // a dark pack that names no scheme gets the dark tables
	}
	i := int(l.P("scheme", def))
	if i < 0 || i >= len(metroSchemes) {
		i = 0
	}
	return i
}

func metroBuild(l *Classic) *metroSet {
	idx := metroSchemeIndex(l)
	sc := metroSchemes[idx]
	col := func(k string) paintengine2d.Color {
		v, ok := sc[k]
		if !ok {
			v, ok = metroWin10[k]
		}
		if !ok {
			v = "#ff00ff" // a missing key shows (and the table test fails)
		}
		return l.X(k, Hex(v))
	}
	s := &metroSet{win10: idx != 1, selSolid: idx == 1}
	s.face, s.text, s.gray, s.disText = col("face"), col("text"), col("gray"), col("disText")
	s.field = col("field")
	s.accent, s.accentLt = col("accent"), col("accentLt")
	s.onAccent = ReadableOn(s.accent, 3, col("onAccent"), s.text)
	s.selText = s.onAccent

	s.btn = [5][2]paintengine2d.Color{
		{col("btn"), col("btnBorder")}, {col("btnHot"), col("btnHotBorder")},
		{col("btnPress"), col("btnPressBorder")}, {col("btnDis"), col("btnDisBorder")},
		{col("btn"), col("btnDefBorder")},
	}
	s.chk = [4][3]paintengine2d.Color{
		{col("chk"), col("chkBorder"), col("chkMark")}, {col("chkHot"), col("chkHotBorder"), col("chkHotMark")},
		{col("chkPress"), col("chkPressBorder"), col("chkPressMark")}, {col("chkDis"), col("chkDisBorder"), col("chkDisMark")},
	}
	s.ed = [4]paintengine2d.Color{col("edBorder"), col("edHot"), col("edFocus"), col("edDisBorder")}
	s.edDis, s.viewBorder = col("edDis"), col("viewBorder")

	s.sbTrack, s.sbBtnHot, s.sbBtnPress, s.sbIndicator = col("sbTrack"), col("sbBtnHot"), col("sbBtnPress"), col("sbIndicator")
	s.sbThumb = [3]paintengine2d.Color{col("sbThumb"), col("sbThumbHot"), col("sbThumbPress")}
	s.sbGlyph = [4]paintengine2d.Color{col("sbGlyph"), col("sbGlyphHot"), col("sbGlyphPress"), col("sbGlyphDis")}

	s.tab, s.tabHot, s.tabBorder, s.pane = col("tab"), col("tabHot"), col("tabBorder"), col("pane")
	s.prog, s.progTrack, s.progBorder = col("prog"), col("progTrack"), col("progBorder")
	s.slTrack, s.slTrackHot, s.slDis = col("slTrack"), col("slTrackHot"), col("slDis")
	s.groove, s.grooveBorder = col("groove"), col("grooveBorder")
	s.slThumb = [3]paintengine2d.Color{col("slThumb"), col("slThumbHot"), col("slThumbPress")}
	s.swOff, s.swOffHot, s.swPress, s.swDis, s.swKnobOn = col("swOff"), col("swOffHot"), col("swPress"), col("swDis"), col("swKnobOn")

	s.hov, s.selFill, s.selBorder, s.off, s.focusBorder = col("hov"), col("selFill"), col("selBorder"), col("off"), col("focusBorder")
	s.selFillText = ReadableOn(s.selFill, 4.5, s.text, s.onAccent)
	s.offText = ReadableOn(s.off, 4.5, s.text, s.onAccent)

	s.menuBg, s.menuBorder, s.menuHot, s.menuHotBorder = col("menuBg"), col("menuBorder"), col("menuHot"), col("menuHotBorder")
	s.menuSep, s.menuSepLt, s.gutterLine, s.gutterLineLt = col("menuSep"), col("menuSepLt"), col("gutterLine"), col("gutterLineLt")
	s.menuDisText = col("menuDisText")
	s.menuHotText = ReadableOn(s.menuHot, 4.5, s.text, s.onAccent)
	s.mbar, s.mbHot, s.mbHotBorder, s.mbOpen, s.mbOpenBorder = col("mbar"), col("mbHot"), col("mbHotBorder"), col("mbOpen"), col("mbOpenBorder")

	s.exp, s.expOpen = col("exp"), col("expOpen")
	s.hdr, s.hdrDiv, s.hdrHot, s.hdrHotBorder = col("hdr"), col("hdrDiv"), col("hdrHot"), col("hdrHotBorder")
	s.hdrPress, s.hdrPressBorder, s.hdrText, s.hdrArrow = col("hdrPress"), col("hdrPressBorder"), col("hdrText"), col("hdrArrow")

	s.tip, s.tipBorder, s.tipText = col("tip"), col("tipBorder"), col("tipText")
	s.groupBorder, s.groupText = col("groupBorder"), col("groupText")
	s.status, s.statTop, s.statSep = col("status"), col("statTop"), col("statSep")
	s.tb, s.tbBorder, s.toolHot, s.toolHotBorder = col("tb"), col("tbBorder"), col("toolHot"), col("toolHotBorder")
	s.toolPress, s.toolPressBorder = col("toolPress"), col("toolPressBorder")
	s.etchDk, s.etchLt = col("etchDk"), col("etchLt")

	s.frame, s.frameOff, s.frameBorder, s.frameBorderOff = col("frame"), col("frameOff"), col("frameBorder"), col("frameBorderOff")
	s.capText, s.capTextOff = col("capText"), col("capTextOff")
	s.close, s.closeOff, s.closeHot, s.closePress = col("close"), col("closeOff"), col("closeHot"), col("closePress")
	s.closeGlyph, s.closeGlyphHot, s.closeGlyphOff = col("closeGlyph"), col("closeGlyphHot"), col("closeGlyphOff")
	s.msgError, s.msgInfo, s.msgWarn, s.msgWarnGlyph = col("msgError"), col("msgInfo"), col("msgWarn"), col("msgWarnGlyph")
	return s
}

// btnIndex picks a push button state: 0 normal, 1 hot, 2 pressed, 3
// disabled, 4 default.
func (c *metroSet) btnIndex(st ControlState) int {
	switch {
	case st.Disabled():
		return 3
	case st.Pressed():
		return 2
	case st.Hovered():
		return 1
	case st.Primary() || (st.Focused() && c.win10):
		return 4
	}
	return 0
}

// ---- parts ------------------------------------------------------------------------------------

// flatBox fills b in a bw-wide square border: the Metro face. It stays
// square whatever the Corners pref says — the flat era had no radii.
func (c *metroSet) flatBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, fill, border paintengine2d.Color, bw float32) paintengine2d.Rect {
	b = winSnap(b)
	if b.Dx() < 2*bw || b.Dy() < 2*bw {
		return paintengine2d.Rect{}
	}
	if fill.A > 0 {
		ctx.DrawRect(b.Inset(bw), paintengine2d.Fill(fill))
	}
	winBorder(ctx, b, bw, border)
	return b.Inset(bw)
}

// pushButton paints a push button face for st and returns the label colour.
func (c *metroSet) pushButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	lw := winPx(l)
	i := c.btnIndex(st)
	bw := lw
	if i == 4 || (c.win10 && st.Focused() && !st.Disabled()) {
		bw = 2 * lw // the default (and, on Windows 10, focused) button
	}
	border := c.btn[i][1]
	if bw > lw {
		border = c.btn[4][1]
	}
	c.flatBox(l, ctx, b, c.btn[i][0], border, bw)
	if i == 3 {
		return c.disText
	}
	return ReadableOn(c.btn[i][0], 4.5, c.text, c.onAccent)
}

// chkIndex picks the check box row: 0 normal, 1 hot, 2 pressed, 3 disabled.
func metroChkIndex(st ControlState) int { return aeroChkIndex(st) }

func (e metroEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := metroColors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	switch role {
	case RoleButton, RoleCombo:
		return c.pushButton(l, ctx, b, st)
	case RoleTool:
		return c.toolFace(l, ctx, b, st)
	case RoleField:
		c.edit(l, ctx, b, st)
		if !st.Disabled() {
			fg = l.fieldText()
		}
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, false)
	case RoleRow:
		return c.row(l, ctx, winSnap(b), st)
	case RoleMenu:
		e.MenuHighlight(l, ctx, b, false)
		return c.menuHotText
	case RoleThumb:
		i := 0
		switch {
		case st.Pressed():
			i = 2
		case st.Hovered():
			i = 1
		}
		ctx.DrawRect(winSnap(b), paintengine2d.Fill(c.sbThumb[i]))
	case RoleTrack:
		ctx.DrawRect(b, paintengine2d.Fill(c.sbTrack))
	case RoleTab:
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
	}
	return fg
}

// edit paints an edit box: the field in a 1px line (dark grey, black when
// hot, the accent when focused).
func (c *metroSet) edit(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	i, fill := 0, c.field
	switch {
	case st.Disabled():
		i, fill = 3, c.edDis
	case st.Focused():
		i = 2
	case st.Hovered():
		i = 1
	}
	c.flatBox(l, ctx, b, fill, c.ed[i], winPx(l))
}

// toolFace is a flat tool button: nothing until hot, then Explorer's pale
// box; pressed and latched buttons take the selection box.
func (c *metroSet) toolFace(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	if st.Disabled() {
		return c.disText
	}
	lw := winPx(l)
	var fill, border paintengine2d.Color
	switch {
	case st.Pressed(), st.Checked() && st.Hovered():
		fill, border = c.toolPress, c.toolPressBorder
	case st.Checked():
		fill, border = c.toolPress, c.toolPressBorder
	case st.Hovered():
		fill, border = c.toolHot, c.toolHotBorder
	case st.Focused():
		if c.win10 {
			c.flatBox(l, ctx, b, paintengine2d.Color{}, c.accent, lw)
			return c.text
		}
		fill, border = c.toolHot, c.toolHotBorder
	default:
		return c.text
	}
	c.flatBox(l, ctx, b, fill, border, lw)
	return ReadableOn(fill, 4.5, c.text, c.onAccent)
}

// row paints a list row's selection (Windows 10: Explorer's light accent
// box; Windows 8: the solid highlight) and returns the label colour.
func (c *metroSet) row(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	lw := winPx(l)
	fg := l.fieldText()
	switch {
	case st.Disabled():
		if st.Checked() {
			ctx.DrawRect(b, paintengine2d.Fill(c.off))
		}
		return c.gray
	case st.Checked() && (st.Inactive() || st.Backdrop()):
		ctx.DrawRect(b, paintengine2d.Fill(c.off))
		return c.offText
	case st.Checked():
		ctx.DrawRect(b, paintengine2d.Fill(c.selFill))
		if st.Hovered() && !c.selSolid {
			winBorder(ctx, b, lw, c.selBorder)
		}
		return c.selFillText
	case st.Hovered():
		ctx.DrawRect(b, paintengine2d.Fill(c.hov))
	}
	return fg
}

func (e metroEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := metroColors(l)
	box = winSnap(box)
	lw := winPx(l)
	if box.Dx() < 5*lw || box.Dy() < 5*lw {
		return
	}
	k := c.chk[metroChkIndex(st)]
	winWell(ctx, box, lw, k[1], k[0])
	if checked || st.Checked() {
		w := box.Dx() * 0.125
		if !c.win10 {
			w = box.Dx() * 0.16
		}
		winTick(ctx, box.Inset(box.Dx()*0.04), k[2], max(w, lw*1.2))
	}
}

func (e metroEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := metroColors(l)
	box = winSnap(box)
	lw := winPx(l)
	r := min(box.Dx(), box.Dy()) * 0.5
	if r < 3*lw {
		return
	}
	ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
	k := c.chk[metroChkIndex(st)]
	ctx.DrawCircle(ctr, r, paintengine2d.Fill(k[1]))
	ctx.DrawCircle(ctr, r-lw, paintengine2d.Fill(k[0]))
	if selected || st.Checked() {
		ctx.DrawCircle(ctr, r*0.42, paintengine2d.Fill(k[2]))
	}
}

// Arrow is the solid Windows glyph (8×4 at 1x), fitted into b.
func (metroEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	u := min(min(b.Dx(), b.Dy())/10, l.S(1))
	winGlyph(ctx, b, dir, u, col)
}

// Expander: Windows 8 kept Vista's triangles (flat); Windows 10 draws thin
// chevrons.
func (metroEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := metroColors(l)
	if !c.win10 {
		winVistaTriangle(l, ctx, b, expanded, c.expOpen, c.expOpen, c.exp, c.field)
		return
	}
	dir, gc := DirRight, c.exp
	if expanded {
		dir, gc = DirDown, c.expOpen
	}
	winChevron(l, ctx, b, dir, gc)
}

// MenuHighlight: Windows 10 fills the hot row with a solid pale blue;
// Windows 8 boxes it. An open menu-bar title is Explorer's selected box.
func (metroEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := metroColors(l)
	lw := winPx(l)
	if attachBottom {
		c.flatBox(l, ctx, b, c.mbOpen, c.mbOpenBorder, lw)
		return
	}
	c.flatBox(l, ctx, b, c.menuHot, c.menuHotBorder, lw)
}

func (metroEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := metroColors(l)
	if hot {
		return c.menuHotText
	}
	return c.text
}

// Fields show focus with their accent line (painted by Face).
func (metroEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is Windows 8's dotted rectangle, Windows 10's accent line.
func (metroEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := metroColors(l)
	lw := winPx(l)
	b = winSnap(b).Inset(lw)
	if c.win10 {
		winBorder(ctx, b, lw, c.accent)
		return
	}
	winDots(ctx, b, c.text, lw)
}

// focusMark rings a label or control with the look's focus.
func (c *metroSet) focusMark(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	lw := winPx(l)
	if c.win10 {
		winBorder(ctx, winSnap(b), lw, c.accent)
		return
	}
	winDots(ctx, b, c.text, lw)
}

// ---- scroll bars ----------------------------------------------------------------------------------

// ScrollBarStyle: Windows 8's 17px bar with an arrow button at each end;
// Windows 10's thin 12px bar whose buttons show on hover.
func (metroEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	if metroColors(l).win10 {
		return ScrollBarStyle{Thickness: 12, Arrows: ArrowsEnds, ArrowLen: 12, MinThumb: 16}
	}
	return ScrollBarStyle{Thickness: 17, Arrows: ArrowsEnds, MinThumb: 17}
}

func (e metroEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := metroColors(l)
	bar := winSnap(p.Bar)
	if bar.Empty() {
		return
	}
	lw := winPx(l)
	hover := !st.Disabled && (st.Hovered || st.Hot != ScrollNone || st.Pressed != ScrollNone)
	if c.win10 && !hover {
		// At rest Windows 10 shows only a slim indicator along the edge.
		if p.Thumb.Empty() || st.Disabled {
			return
		}
		w := snap(l.S(3))
		t := winSnap(p.Thumb)
		if vertical {
			ctx.DrawRect(paintengine2d.XYWH(bar.Max.X-w-lw, t.Min.Y, w, t.Dy()), paintengine2d.Fill(c.sbIndicator))
		} else {
			ctx.DrawRect(paintengine2d.XYWH(t.Min.X, bar.Max.Y-w-lw, t.Dx(), w), paintengine2d.Fill(c.sbIndicator))
		}
		return
	}
	ctx.DrawRect(bar, paintengine2d.Fill(c.sbTrack))
	if pg := winPagedPart(p, vertical, st); !pg.Empty() {
		ctx.DrawRect(winSnap(pg), paintengine2d.Fill(c.sbBtnHot))
	}
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		b = winSnap(b)
		g := c.sbGlyph[0]
		switch {
		case st.Disabled:
			g = c.sbGlyph[3]
		case st.Pressed == part:
			ctx.DrawRect(b, paintengine2d.Fill(c.sbBtnPress))
			g = c.sbGlyph[2]
		case st.Hot == part:
			ctx.DrawRect(b, paintengine2d.Fill(c.sbBtnHot))
			g = c.sbGlyph[1]
		}
		u := min(l.S(1), min(b.Dx(), b.Dy())/10)
		if c.win10 {
			u *= 0.75
		}
		winGlyph(ctx, b, dir, u, g)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec)
	arrow(p.Inc, inc, ScrollInc)
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	i := 0
	switch {
	case st.Pressed == ScrollThumbPart:
		i = 2
	case st.Hot == ScrollThumbPart:
		i = 1
	}
	in := lw
	if c.win10 {
		in = 2 * lw
	}
	t := winSnap(p.Thumb)
	if vertical {
		t = paintengine2d.XYWH(bar.Min.X+in, t.Min.Y, bar.Dx()-2*in, t.Dy())
	} else {
		t = paintengine2d.XYWH(t.Min.X, bar.Min.Y+in, t.Dx(), bar.Dy()-2*in)
	}
	ctx.DrawRect(t, paintengine2d.Fill(c.sbThumb[i]))
}

func (e metroEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), winScrollState(st))
}

// ---- frames ------------------------------------------------------------------------------------------

func (metroEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is a square 1px light-grey frame open behind the title.
func (metroEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := metroColors(l)
	winGroupFrame(l, ctx, b, title, raised, 0, c.groupBorder, paintengine2d.Color{}, c.pane, c.groupText)
}

// metroCaptionH is the in-app caption: Windows 10's 30px title bar (31 on
// Windows 8) at the toolkit's font.
func metroCaptionH(l *Classic) float32 {
	h := l.body.Height() + l.S(12)
	if m := l.S(32); h < m {
		h = m
	}
	return snap(h)
}

// metroFrameW is the window border: Windows 10's hairline, Windows 8's
// 8px coloured frame.
func metroFrameW(l *Classic) float32 {
	if metroColors(l).win10 {
		return winPx(l)
	}
	return snap(l.S(7))
}

func (metroEngine) WindowFrameInsets(l *Classic) Insets {
	fw := metroFrameW(l)
	return Insets{Top: metroCaptionH(l), Right: fw, Bottom: fw, Left: fw}
}

// WindowCloseRect: Windows 10's 46px caption button fills the title bar's
// height at the right; Windows 8's red button hangs from the top edge.
func (metroEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	capH := metroCaptionH(l)
	if c.win10 {
		h := capH - lw
		w := min(snap(h*46/30), snap(b.Dx()*0.3))
		return paintengine2d.XYWH(b.Max.X-lw-w, b.Min.Y+lw, w, h)
	}
	h := snap(capH * 0.62)
	w := min(snap(h*46/20), snap(b.Dx()*0.3))
	return paintengine2d.XYWH(b.Max.X-metroFrameW(l)-w, b.Min.Y+lw, w, h)
}

// Windows dialogs put the default button first: "OK  Cancel".
func (metroEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintHoverFadeMs {
		return 150 // Windows 10 controls fade their hover
	}
	if h == HintDialogPrimaryFirst {
		return 1
	}
	if h == HintMnemonics {
		return MnemonicsOnAlt // Windows hides the underlines until Alt
	}
	return 0
}

// DrawWindowFrame: Windows 10's window is a white title bar over a
// hairline border with the title at the left; Windows 8's is a thick flat
// coloured frame with the title centred and the red close button.
func (e metroEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	capH := metroCaptionH(l)
	fw := metroFrameW(l)
	if b.Dx() < 4*capH*0.5 || b.Dy() < capH+2*fw {
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		return
	}
	frame, border, tc := c.frame, c.frameBorder, c.capText
	if !st.Active {
		frame, border, tc = c.frameOff, c.frameBorderOff, c.capTextOff
	}
	winWell(ctx, b, lw, border, frame)
	client := paintengine2d.XYWH(b.Min.X+fw, b.Min.Y+capH, b.Dx()-2*fw, b.Dy()-capH-fw)
	if !c.win10 {
		ctx.DrawRect(client.Inset(-lw), paintengine2d.Fill(border))
	}
	ctx.DrawRect(client, paintengine2d.Fill(c.face))
	right := b.Max.X - fw
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		c.closeButton(l, ctx, cb, st)
		right = cb.Min.X - l.S(4)
	}
	if title == "" {
		return
	}
	bar := paintengine2d.XYWH(b.Min.X+fw, b.Min.Y+lw, right-b.Min.X-fw, capH-lw)
	if c.win10 {
		l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(bar.Min.X+l.S(8), bar.Min.Y, bar.Dx()-l.S(8), bar.Dy()), tc, AlignStart, 0)
		return
	}
	// Windows 8 centres the title over the whole caption.
	full := paintengine2d.XYWH(b.Min.X+fw, bar.Min.Y, b.Dx()-2*fw, bar.Dy())
	tw := l.body.Advance(title)
	if full.Min.X+(full.Dx()+tw)*0.5 > right {
		l.drawFittedText(ctx, l.body, title, bar, tc, AlignCenter, 8)
		return
	}
	l.drawFittedText(ctx, l.body, title, full, tc, AlignCenter, 8)
}

// closeButton paints the caption close button: Windows 10's flat button
// that turns red when hot; Windows 8's red rectangle.
func (c *metroSet) closeButton(l *Classic, ctx *paintengine2d.Context, cb paintengine2d.Rect, st WindowState) {
	fill, glyph := c.close, c.closeGlyph
	switch {
	case !st.Active:
		fill, glyph = c.closeOff, c.closeGlyphOff
	case st.ClosePress:
		fill, glyph = c.closePress, c.closeGlyphHot
	case st.CloseHot:
		fill, glyph = c.closeHot, c.closeGlyphHot
	}
	cb = winSnap(cb)
	ctx.DrawRect(cb, paintengine2d.Fill(fill))
	lw := winPx(l)
	side := snap(min(cb.Dy()*0.36, l.S(10)))
	gb := paintengine2d.XYWH(snap((cb.Min.X+cb.Max.X-side)*0.5), snap((cb.Min.Y+cb.Max.Y-side)*0.5), side, side)
	w := lw
	if !c.win10 {
		w = max(lw*1.6, cb.Dy()*0.1)
	}
	winCross(ctx, gb, glyph, w)
}

// PopupShadow: small soft shadows under menus and tool tips, a wide one
// round windows (Windows 10); Windows 8 floated flat with a slight shadow.
func (metroEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	sp, ok := metroShadow(l, kind)
	if !ok {
		return Insets{}
	}
	return ShadowReach(sp.dx, sp.dy, sp.blur, sp.spread)
}

func (metroEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if sp, ok := metroShadow(l, kind); ok {
		DropShadow(ctx, b, sp.r, sp.col, sp.dx, sp.dy, sp.blur, sp.spread)
	}
}

func metroShadow(l *Classic, kind PopupKind) (winShadow, bool) {
	dark := Luma(l.palette.Background) < 0.5
	a := float32(1)
	if dark {
		a = 1.6
	}
	switch kind {
	case PopupDialog:
		return winShadowSpec(l, 0.34*a, 0, 0, l.S(4), l.S(18), 0)
	case PopupTooltip:
		return winShadowSpec(l, 0.2*a, 0, l.S(1), l.S(1), l.S(4), 0)
	}
	return winShadowSpec(l, 0.24*a, 0, l.S(2), l.S(2), l.S(6), -l.S(1))
}

// TabOutset: the selected tab is 2px wider on each side (the common
// controls kept the XP metrics).
func (metroEngine) TabOutset(l *Classic) Insets {
	return Insets{Left: snap(l.S(2)), Right: snap(l.S(2))}
}

// TabOverlap: neighbouring tabs share one border line.
func (metroEngine) TabOverlap(l *Classic) float32 { return winPx(l) }

// DrawTabPane is the page under the tabs in its light-grey line.
func (metroEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := metroColors(l)
	winWell(ctx, winSnap(b), winPx(l), c.tabBorder, c.pane)
}

func (metroEngine) ViewFrameInsets(l *Classic) Insets {
	lw := winPx(l)
	return Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}
}

// DrawViewFrame is the list box's field in its 1px line.
func (metroEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := metroColors(l)
	border, fill := c.viewBorder, c.field
	if st.Disabled() {
		border, fill = c.ed[3], c.edDis
	}
	winWell(ctx, winSnap(b), winPx(l), border, fill)
}

// ItemFocus: Windows 10's light accent line round the current row;
// Windows 8's dotted rectangle (white on the solid highlight).
func (metroEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := metroColors(l)
	lw := winPx(l)
	b = winSnap(b)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	if c.win10 {
		winBorder(ctx, b, lw, c.focusBorder)
		return
	}
	col := c.text
	if st.Checked() && !st.Inactive() && !st.Backdrop() {
		col = c.selFillText
	}
	winDots(ctx, b, col, lw)
}

// ---- controls ---------------------------------------------------------------------------------------

func (e metroEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := metroColors(l)
	fg := c.pushButton(l, ctx, b, st)
	l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, 8)
	if st.Focused() && !st.Disabled() && !c.win10 {
		lw := winPx(l)
		winDots(ctx, winSnap(b).Inset(3*lw), c.text, lw)
	}
}

func (e metroEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	metroToggleLabel(l, ctx, b, winSnap(box), st, label)
}

func (e metroEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	metroToggleLabel(l, ctx, b, winSnap(box), st, label)
}

// metroToggleLabel draws a check box / radio caption and its focus mark.
func metroToggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := metroColors(l)
	lw := winPx(l)
	if label == "" {
		if st.Focused() {
			c.focusMark(l, ctx, box.Inset(-lw).Intersect(b))
		}
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		c.focusMark(l, ctx, labelFocusRect(l.body, label, lb, b))
	}
}

func (e metroEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	if !st.Disabled() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	e.Face(l, ctx, b, RoleField, st)
	winDisabledText(l, ctx, b, text, placeholder, scrollX, face, metroColors(l).disText)
}

// DrawComboBox is a drop-down list: a flat button with the arrow at the
// right (Windows 10: a thin chevron). Windows 8 highlights a focused
// list's text; Windows 10 gives it the accent border.
func (e metroEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	fst := st
	if open {
		fst |= StatePressed
	}
	if !c.win10 {
		fst &^= StateFocused
	}
	fg := c.pushButton(l, ctx, b, fst)
	aw := min(snap(l.S(17)), snap(b.Dx()*0.5))
	ab := paintengine2d.XYWH(b.Max.X-aw-lw, b.Min.Y, aw, b.Dy())
	glyph := c.sbGlyph[0]
	if c.win10 {
		glyph = c.text
	}
	if st.Disabled() {
		glyph = c.disText
	}
	if c.win10 {
		winChevron(l, ctx, ab, DirDown, glyph)
	} else {
		metroEngine{}.Arrow(l, ctx, ab, DirDown, glyph)
	}
	tb := paintengine2d.XYWH(b.Min.X+2*lw, b.Min.Y+3*lw, ab.Min.X-b.Min.X-2*lw, b.Dy()-6*lw)
	if !c.win10 && st.Focused() && !open && !st.Disabled() && !st.Editable() {
		tb = winSnap(tb)
		ctx.DrawRect(tb, paintengine2d.Fill(c.accent))
		fg = c.onAccent
		winDots(ctx, tb, c.onAccent, lw)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(tb.Min.X+l.S(4), tb.Min.Y, tb.Dx()-l.S(5), tb.Dy()), fg, AlignStart, 0)
}

// DrawSpinner is the up-down control: two flat buttons.
func (e metroEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(h paintengine2d.Rect, dir Direction, hover, press bool) {
		i := 0
		switch {
		case st.Disabled():
			i = 3
		case press:
			i = 2
		case hover:
			i = 1
		}
		c.flatBox(l, ctx, h, c.btn[i][0], c.btn[i][1], lw)
		g := c.sbGlyph[0]
		if st.Disabled() {
			g = c.sbGlyph[3]
		}
		winGlyph(ctx, h, dir, min(l.S(1), min(h.Dx(), h.Dy())/8), g)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y+lw), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress)
}

func (metroEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.tabBorder))
}

// DrawTab is a flat tab in a 1px line: the face, pale blue when hot; the
// selected tab is the page colour, taller and wider, open into the page.
func (metroEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	u2 := snap(l.S(2))
	t := b
	if !selected {
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+u2, b.Dx(), b.Dy()-u2-lw)
	}
	if t.Dx() < 4*lw || t.Dy() < 4*lw {
		return
	}
	fill := c.tab
	switch {
	case selected:
		fill = c.pane
	case !st.Disabled() && (st.Hovered() || st.Pressed()):
		fill = c.tabHot
	}
	ctx.DrawRect(t, paintengine2d.Fill(c.tabBorder))
	ctx.DrawRect(paintengine2d.XYWH(t.Min.X+lw, t.Min.Y+lw, t.Dx()-2*lw, t.Dy()-lw), paintengine2d.Fill(fill))
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	lb := paintengine2d.XYWH(b.Min.X, b.Min.Y+u2, b.Dx(), b.Dy()-u2-lw)
	if selected {
		lb = lb.Translate(paintengine2d.Pt(0, -lw))
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, 8)
	if st.Focused() && selected {
		c.focusMark(l, ctx, winLabelBox(l, l.body, label, lb, AlignCenter))
	}
}

func (metroEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := metroColors(l)
	if !raised {
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		return
	}
	winWell(ctx, winSnap(b), winPx(l), c.groupBorder, c.pane)
}

func (metroEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(metroColors(l).mbar))
}

// DrawMenuTitle: a pale blue box when hot, Explorer's selected box when
// open; a keyboard-focused title takes the focus mark.
func (e metroEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := metroColors(l)
	lw := winPx(l)
	hb := winSnap(b).Inset(lw)
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.gray
	case open:
		c.flatBox(l, ctx, hb, c.mbOpen, c.mbOpenBorder, lw)
	case st.Pressed() || st.Hovered():
		c.flatBox(l, ctx, hb, c.mbHot, c.mbHotBorder, lw)
	case st.Focused():
		c.focusMark(l, ctx, hb)
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
}

// DrawMenuFrame: Windows 10's light grey popup in a grey line; Windows 8
// keeps the Windows 7 gutter edge.
func (metroEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	winWell(ctx, b, lw, c.menuBorder, c.menuBg)
	if c.win10 {
		return
	}
	in := b.Inset(lw)
	gx := snap(b.Min.X + MenuChromeFor(l).GutterW())
	if gx-in.Min.X > 2*lw && gx-in.Min.X < in.Dx()*0.55 {
		ctx.DrawRect(paintengine2d.XYWH(gx, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.gutterLine))
		ctx.DrawRect(paintengine2d.XYWH(gx+lw, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.gutterLineLt))
	}
}

// DrawMenuItem: the hot row across the popup (square), thin checks in the
// gutter, separators from the label column.
func (e metroEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := metroColors(l)
	lw := winPx(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0 := snap(ch.LabelMinX(b.Min.X) - l.S(4))
		if !c.win10 {
			x0 = snap(b.Min.X - ch.PadL + ch.GutterW() + 2*lw + l.S(4))
		}
		x1 := snap(b.Max.X + ch.PadR - 2*lw)
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y-lw, x1-x0, lw), paintengine2d.Fill(c.menuSep))
			if !c.win10 {
				ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, lw), paintengine2d.Fill(c.menuSepLt))
			}
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	fg := c.text
	if hot {
		inset := 2 * lw
		if c.win10 {
			inset = lw
		}
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL+inset, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*inset, b.Dy())
		e.MenuHighlight(l, ctx, hb, false)
		fg = c.menuHotText
	}
	if st.Disabled() {
		fg = c.menuDisText
	}
	if row.Checked || row.Radio || row.Icon != IconNone {
		gw := ch.CheckCol()
		side := min(b.Dy()-4*lw, gw-2*lw)
		box := winSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
		switch {
		case row.Checked && row.Radio:
			ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
			ctx.DrawCircle(ctr, box.Dx()*0.17, paintengine2d.Fill(fg))
		case row.Checked && row.Icon == IconNone:
			winTick(ctx, box.Inset(box.Dx()*0.1), fg, max(box.Dx()*0.09, lw*1.2))
		case row.Checked:
			if !c.win10 {
				c.flatBox(l, ctx, box, c.selFill.WithAlpha(0.25), c.selBorder, lw)
			} else {
				winBorder(ctx, box, lw, c.selBorder)
			}
			l.drawToolIcon(ctx, box.Inset(2*lw), row.Icon, fg)
		case !row.Radio:
			l.drawToolIcon(ctx, box.Inset(lw), row.Icon, fg)
		}
	}
	winMenuText(l, ctx, b, ch, row, fg, fg, func(ab paintengine2d.Rect) {
		if c.win10 {
			winChevron(l, ctx, ab, DirRight, fg)
			return
		}
		metroEngine{}.Arrow(l, ctx, ab, DirRight, fg)
	})
}

// DrawProgressBar: Windows' green, flat, in a light trough with a grey
// line; the marquee slides a green block.
func (metroEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	winWell(ctx, b, lw, c.progBorder, c.progTrack)
	area := b.Inset(lw)
	fill := paintengine2d.Fill(c.prog)
	if st.Disabled() {
		fill = paintengine2d.Fill(c.prog.WithAlpha(0.35))
	}
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := max(area.Dx()*0.25, l.S(24))
		x := area.Min.X + (area.Dx()+span)*phase - span
		seg := paintengine2d.XYWH(x, area.Min.Y, span, area.Dy()).Intersect(area)
		if !seg.Empty() {
			ctx.DrawRect(seg, fill)
		}
		return
	}
	if w := snap(area.Dx() * clamp1(t)); w >= lw {
		ctx.DrawRect(paintengine2d.XYWH(area.Min.X, area.Min.Y, w, area.Dy()), fill)
	}
}

// DrawSlider: Windows 10's 2px track, accent to the left of its
// rectangular accent thumb (black when hot, grey when dragged); Windows 8's
// flat groove and pointed thumb.
func (metroEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	t = clamp1(t)
	if !c.win10 {
		metroTrackbar(l, c, ctx, b, st, t)
		return
	}
	tw, th := snap(l.S(8)), min(snap(l.S(24)), b.Dy())
	if b.Dx() < tw+2*lw || th < 6*lw {
		return
	}
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	x0, x1 := b.Min.X+tw*0.5, b.Max.X-tw*0.5
	tx := snap(x0 + (x1-x0)*t - tw*0.5)
	tr := 2 * lw
	rest, fill, thumb := c.slTrack, c.accent, c.slThumb[0]
	switch {
	case st.Disabled():
		rest, fill, thumb = c.slDis, c.slDis, c.slDis
	case st.Pressed():
		rest, thumb = c.slTrackHot, c.slThumb[2]
	case st.Hovered():
		rest, thumb = c.slTrackHot, c.slThumb[1]
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, cy-lw, tx-b.Min.X, tr), paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(tx+tw, cy-lw, b.Max.X-tx-tw, tr), paintengine2d.Fill(rest))
	ctx.DrawRect(paintengine2d.XYWH(tx, snap(cy-th*0.5), tw, th), paintengine2d.Fill(thumb))
	if st.Focused() && !st.Disabled() {
		winBorder(ctx, b, lw, c.accent)
	}
}

// metroTrackbar is Windows 8's trackbar: a flat groove and the pointed
// thumb in the button colours; focus is the dotted rectangle.
func metroTrackbar(l *Classic, c *metroSet, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	lw := winPx(l)
	tw, th := snap(l.S(11)), min(snap(l.S(19)), b.Dy()-2*lw)
	if tw < 5*lw || th < 8*lw || b.Dx() < tw+2*lw {
		return
	}
	ty := snap(b.Min.Y + (b.Dy()-th)*0.5)
	gy := snap(ty + th*0.4 - 2*lw)
	winWell(ctx, paintengine2d.XYWH(b.Min.X+lw, gy, b.Dx()-2*lw, 4*lw), lw, c.grooveBorder, c.groove)
	x0, x1 := b.Min.X+tw*0.5, b.Max.X-tw*0.5
	tx := snap(x0 + (x1-x0)*t - tw*0.5)
	pt := snap(min(l.S(5), th*0.35))
	k := c.btn[aeroChkIndex(st)]
	outer := paintengine2d.XYWH(tx, ty, tw, th)
	ctx.DrawPath(winPointer(outer, pt, 0), paintengine2d.Fill(k[1]))
	ctx.DrawPath(winPointer(outer.Inset(lw), pt-lw*0.4, 0), paintengine2d.Fill(k[0]))
	if st.Focused() && !st.Disabled() {
		winDots(ctx, b, c.text, lw)
	}
}

// DrawSwitch: Windows 10's pill toggle (a 2px outline and dot when off,
// the accent with a white dot when on); Windows 8's rectangular switch
// with a black thumb block.
func (e metroEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := metroColors(l)
	lw := winPx(l)
	tw, th := l.metrics.SwitchW, min(l.metrics.SwitchH, b.Dy())
	track := winSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 8*lw || track.Dy() < 6*lw {
		return
	}
	bw := 2 * lw
	edge, knob := c.swOff, c.swOff
	var fill paintengine2d.Color
	switch {
	case st.Disabled():
		edge, knob = c.swDis, c.swDis
		if on {
			fill, knob = c.swDis, c.face
		}
	case st.Pressed():
		edge, fill, knob = c.swPress, c.swPress, c.swKnobOn
	case on && st.Hovered():
		edge, fill, knob = c.accentLt, c.accentLt, c.swKnobOn
	case on:
		edge, fill, knob = c.accent, c.accent, c.swKnobOn
	case st.Hovered():
		edge, knob = c.swOffHot, c.swOffHot
	}
	if c.win10 {
		r := track.Dy() * 0.5
		if fill.A > 0 {
			ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(fill))
		}
		winRing(ctx, track, r, bw, paintengine2d.Fill(edge))
		kr := snap(l.S(5))
		kx := track.Min.X + r
		if on {
			kx = track.Max.X - r
		}
		ctx.DrawCircle(paintengine2d.Pt(kx, (track.Min.Y+track.Max.Y)*0.5), kr, paintengine2d.Fill(knob))
	} else {
		// Windows 8: a 2px frame, a gap, the fill, and the thumb block
		// standing over the frame at the on or off end.
		winBorder(ctx, track, bw, edge)
		in := track.Inset(2 * bw)
		if fill.A > 0 && !in.Empty() {
			ctx.DrawRect(in, paintengine2d.Fill(fill))
		}
		kw := snap(l.S(11))
		kx := track.Min.X
		if on {
			kx = track.Max.X - kw
		}
		thumb := c.text
		if st.Disabled() {
			thumb = c.swDis
		}
		ctx.DrawRect(paintengine2d.XYWH(kx, track.Min.Y, kw, track.Dy()), paintengine2d.Fill(thumb))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	if label == "" {
		if st.Focused() {
			c.focusMark(l, ctx, track.Inset(-lw).Intersect(b))
		}
		return
	}
	lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		c.focusMark(l, ctx, labelFocusRect(l.body, label, lb, b))
	}
}

// DrawListRow is a list item: Windows 10's light accent box (bordered
// under the pointer), Windows 8's solid highlight; grey when the view is
// not focused.
func (e metroEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := metroColors(l)
	box := winSnap(b)
	fg := c.row(l, ctx, box, st)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, box, st)
	}
}

// DrawTreeRow is a navigation-pane row: chevrons (Windows 10) or flat
// Vista triangles (Windows 8), the selection across the row.
func (e metroEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := metroColors(l)
	box := winSnap(b)
	fg := c.row(l, ctx, box, st)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, fg)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	lx := x + indent + l.S(3)
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx-l.S(4), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, box, st)
	}
}

// DrawTableHeader is the flat list header: white, hairline dividers, a
// pale blue hot item and the sort chevron centred at the top.
func (metroEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Empty() {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	hot := st.Hovered() && !st.Disabled() && !pressed
	switch {
	case pressed:
		winWell(ctx, b, lw, c.hdrPressBorder, c.hdrPress)
	case hot:
		winWell(ctx, b, lw, c.hdrHotBorder, c.hdrHot)
	default:
		ctx.DrawRect(b, paintengine2d.Fill(c.hdr))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(c.hdrDiv))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx()-lw, lw), paintengine2d.Fill(c.hdrDiv))
	}
	if sorted {
		dir := DirDown
		if asc {
			dir = DirUp
		}
		winChevron(l, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), snap(l.S(8))), dir, c.hdrArrow)
	}
	fg := c.hdrText
	if st.Disabled() {
		fg = c.gray
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
}

func (metroEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := metroColors(l)
	fg := l.fieldText()
	switch {
	case st.Disabled():
		fg = c.gray
		if st.Checked() {
			ctx.DrawRect(winSnap(b), paintengine2d.Fill(c.off))
		}
	case st.Checked() && (st.Inactive() || st.Backdrop()):
		ctx.DrawRect(winSnap(b), paintengine2d.Fill(c.off))
		fg = c.offText
	case st.Checked():
		ctx.DrawRect(winSnap(b), paintengine2d.Fill(c.selFill))
		fg = c.selFillText
	case st.Hovered():
		ctx.DrawRect(winSnap(b), paintengine2d.Fill(c.hov))
	}
	winCellText(l, ctx, b, label, align, face, fg)
}

func (metroEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.tb))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.tbBorder))
}

func (metroEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := metroColors(l)
	fg := c.text
	if st.Toggle() || st.Hovered() || st.Pressed() || st.Focused() {
		fg = c.toolFace(l, ctx, winSnap(b).Inset(winPx(l)), st)
	}
	if st.Disabled() {
		fg = c.disText
	}
	winToolContent(l, ctx, b, label, icon, fg)
}

func (metroEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.status))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.statTop))
	winStatusParts(l, ctx, b, parts, c.text, c.statSep, paintengine2d.Color{})
	winGripDots(l, ctx, b, c.gray, paintengine2d.Color{})
}

// DrawTitleBar is a dialog heading: Windows 8 and 10 kept the main
// instruction, in the accent-dark blue.
func (metroEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := metroColors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	col := ReadableOn(c.face, 4.5, Hex("#003399"), c.text)
	winHeading(l, ctx, b, title, subtitle, col, c.gray, false)
}

// DrawAccordionHeader: a flat header with the expander at the left; hot
// takes Explorer's pale fill.
func (e metroEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := metroColors(l)
	lw := winPx(l)
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		c.flatBox(l, ctx, b, c.toolHot, c.toolHotBorder, lw)
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, c.text)
	fg := c.text
	if st.Disabled() {
		fg = c.gray
	}
	tb := paintengine2d.XYWH(b.Min.X+l.S(22), b.Min.Y, b.Dx()-l.S(28), b.Dy())
	l.drawFittedText(ctx, l.body, title, tb, fg, AlignStart, 0)
	if lx := tb.Min.X + l.body.Advance(title) + l.S(8); lx < b.Max.X-l.S(12) {
		ctx.DrawRect(paintengine2d.XYWH(snap(lx), snap((b.Min.Y+b.Max.Y)*0.5), snap(b.Max.X-l.S(8)-lx), lw), paintengine2d.Fill(c.etchDk))
	}
	if st.Focused() {
		c.focusMark(l, ctx, winLabelBox(l, l.body, title, tb, AlignStart))
	}
}

func (metroEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := metroColors(l)
	lt := c.etchLt
	if c.win10 {
		lt = paintengine2d.Color{}
	}
	winEtched(l, ctx, b, vertical, c.etchDk, lt)
}

func (metroEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	ctx.DrawRect(b, paintengine2d.Fill(metroColors(l).face))
}

// TooltipStyle: Windows 8 pads a tip by 4.
func (metroEngine) TooltipStyle(l *Classic) TooltipStyle {
	return l.tipStyle(l.body, l.tipPad(l.S(4)), AlignStart)
}

// DrawTooltip is the flat tool tip: white, a grey line, grey text.
func (metroEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := metroColors(l)
	tb := b
	b = winSnap(b)
	winWell(ctx, b, winPx(l), c.tipBorder, c.tip)
	pad := l.TooltipStyle().Pad
	// The bubble is sized to the text plus the padding: let the text use
	// the right padding rather than lose its last letters to rounding.
	l.drawTipText(ctx, paintengine2d.XYWH(tb.Min.X+pad, tb.Min.Y, tb.Dx()-pad-winPx(l), tb.Dy()), text, c.tipText)
}

// DrawMessageIcon: flat discs with a white glyph and the yellow triangle.
func (metroEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	c := metroColors(l)
	white := Hex("#ffffff")
	switch icon {
	case IconNone:
	case IconError:
		winFlatDisc(l, ctx, b, c.msgError, "×", white)
	case IconInfo:
		winFlatDisc(l, ctx, b, c.msgInfo, "i", white)
	case IconQuestion:
		winFlatDisc(l, ctx, b, c.msgInfo, "?", white)
	case IconWarning:
		winFlatTriangle(l, ctx, b, c.msgWarn, c.msgWarnGlyph)
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
	}
}

// ---- packs ----------------------------------------------------------------------------------------

func metroPal(face, surface, field, text, muted, accent, border, divider, hot, hotBorder string) Palette {
	a := Hex(accent)
	return Palette{
		Background: Hex(face), Surface: Hex(face), SurfaceAlt: Hex(surface),
		Border: Hex(border), Divider: Hex(divider),
		Text: Hex(text), TextMuted: Hex(muted), TextOnAccent: Hex("#ffffff"),
		Accent: a, AccentHover: Shade(a, 0.2), AccentPress: Shade(a, -0.25),
		Field: Hex(field), FieldBorder: Hex(border),
		Focus: a, Selection: a,
		Track: Hex(surface), Thumb: Hex("#c2c2c2"),
		Highlight: Hex(hot), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: Hex(hot), MenuHoverBorder: Hex(hotBorder), MenuGutter: Hex(surface),
		Danger: Hex("#e81123"), Success: Hex("#107c10"), Warning: Hex("#b36b00"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.35),
		BevelLight: Hex(field), BevelDark: Hex(border),
	}
}

func metroPack(name, label string, year int, summary string, fam ThemeName, scheme int, pal Palette) ThemePack {
	tok := ThemeTokens{
		Engine:  "metro",
		Bevel:   BevelNone,
		Family:  fam,
		Palette: pal,
		Params:  map[string]float32{"scheme": float32(scheme)},
		// Selected text is white on the accent (HighlightText).
		Extra: map[string]paintengine2d.Color{"selectionText": Hex("#ffffff")},
	}
	metroChrome(&tok)
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Windows", Summary: summary,
		Era: "Metro", Palette: fam, Tokens: tok,
	}
}

// metroChrome sets the chrome states Metro derives from its palette.
func metroChrome(tok *ThemeTokens) {
	pal := tok.Palette
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: Shade(pal.MenuHover, -0.08), Border: pal.Accent}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.12), Border: pal.Focus}
}

// metroAccentKeys are the colours of Windows 10 (scheme 0) and Windows 10
// Dark (scheme 2) that are its accent blue: the shades of it — the accent
// and its Light1, the push buttons' hot and pressed borders and the default
// button's, the check boxes' hot and pressed marks, the focused edit line,
// the slider thumb — and the washes of it over the face: the pale blues of
// the hot and pressed buttons and boxes, the hot tab, Explorer's selection,
// the menus, the header and the tool buttons (the dark theme's tinted hot
// and pressed faces; its other hot faces are greys and stay).
var metroAccentKeys = map[int]struct{ shades, washes []string }{
	0: {
		shades: []string{"accent", "accentLt", "btnHotBorder", "btnPressBorder", "btnDefBorder",
			"chkHotBorder", "chkHotMark", "chkPressBorder", "chkPressMark", "edFocus", "slThumb"},
		washes: []string{"btnHot", "btnPress", "chkPress", "tabHot", "hov", "selFill", "selBorder", "focusBorder",
			"menuHot", "menuHotBorder", "mbHot", "mbHotBorder", "mbOpen", "mbOpenBorder",
			"hdrHot", "hdrHotBorder", "hdrPress", "hdrPressBorder", "toolHot", "toolHotBorder", "toolPress", "toolPressBorder"},
	},
	2: {
		shades: []string{"accent", "accentLt", "btnHotBorder", "btnPressBorder", "btnDefBorder",
			"chkHotBorder", "chkHotMark", "chkPressBorder", "edFocus", "slThumb", "selBorder", "focusBorder", "toolPressBorder"},
		washes: []string{"btnHot", "btnPress", "chkPress", "tabHot", "selFill", "toolPress"},
	},
}

// Accented is the accent colour of Windows 10 and its dark mode: the accent
// blue and every shade of it that the controls wear take the user's accent.
// Shades (Light1, the pressed borders) make the step from the accent that
// Windows 10's made from its default #0078d7 (accentShift); the pale blues
// and the dark theme's tinted faces hold the same share of the accent over
// their grey (accentWash: #cce4f7 is the blue at 20% over white, and
// becomes any accent at 20%). The text and the switch knob on the accent
// stay white while they read at 3:1. Windows 8 coloured only its window
// frames: its active frame takes the accent as its window colour, the
// border the shade that the default frame's border is of it, and the
// controls keep Windows 8's fixed #3399ff highlight.
func (metroEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	def := float32(0)
	if accentDark(tok) {
		def = 2
	}
	idx := int(accentP(tok, "scheme", def))
	if idx < 0 || idx >= len(metroSchemes) {
		idx = 0
	}
	sc := metroSchemes[idx]
	own := func(k string) paintengine2d.Color {
		v, ok := sc[k]
		if !ok {
			v = metroWin10[k]
		}
		return accentX(tok, k, Hex(v))
	}
	tok = CloneTokenMaps(tok)
	if idx == 1 {
		frame := own("frame")
		tok.Extra["frame"] = accent
		tok.Extra["frameBorder"] = accentShift(accent, frame, own("frameBorder"))
		tok.Extra["capText"] = ReadableOn(accent, 3, own("capText"), Hex("#ffffff"))
		return tok
	}
	ref := own("accent")
	keys := metroAccentKeys[idx]
	for _, k := range keys.shades {
		tok.Extra[k] = accentShift(accent, ref, own(k))
	}
	for _, k := range keys.washes {
		tok.Extra[k] = accentWash(accent, ref, own(k))
	}
	p := &tok.Palette
	a := tok.Extra["accent"]
	p.Accent, p.Selection = a, a
	p.AccentHover, p.AccentPress = Shade(a, 0.2), Shade(a, -0.25)
	p.Focus = accentShift(accent, ref, p.Focus)
	if idx == 0 {
		// The hot menu row's pale blue.
		p.Highlight = accentWash(accent, ref, p.Highlight)
		p.MenuHover = accentWash(accent, ref, p.MenuHover)
		p.MenuHoverBorder = accentWash(accent, ref, p.MenuHoverBorder)
	}
	p.TextOnAccent = ReadableOn(a, 3, p.TextOnAccent, p.Text)
	// The switch's knob sits on the accent as its text does.
	tok.Extra["swKnobOn"] = ReadableOn(a, 3, own("swKnobOn"), p.Text)
	if c, ok := tok.Extra["selectionText"]; ok {
		tok.Extra["selectionText"] = ReadableOn(p.Selection, 3, c, p.Text)
	}
	metroChrome(&tok)
	return tok
}

func metroPacks() []ThemePack {
	win8 := metroPal("#f0f0f0", "#f0f0f0", "#ffffff", "#000000", "#6d6d6d", "#3399ff", "#acacac", "#d7d7d7", "#d5e1f2", "#a3bde3")
	win10 := metroPal("#f0f0f0", "#f0f0f0", "#ffffff", "#000000", "#6d6d6d", "#0078d7", "#adadad", "#dadada", "#91c9f7", "#91c9f7")
	dark := metroPal("#202020", "#2b2b2b", "#191919", "#ffffff", "#9a9a9a", "#0078d7", "#4d4d4d", "#3d3d3d", "#414141", "#414141")
	dark.Danger, dark.Success, dark.Warning = Hex("#ff6b6b"), Hex("#6ccb5f"), Hex("#fcc300")
	dark.Focus = Hex("#429ce3")
	dark.Shadow, dark.Overlay = paintengine2d.RGBA(0, 0, 0, 0.5), paintengine2d.RGBA(0, 0, 0, 0.5)
	return []ThemePack{
		metroPack("win8", "Windows 8", 2012, "The flat desktop of Windows 8: square 1px-bordered controls, dotted focus, solid blue selection, coloured window frames.", ThemeLight, 1, win8),
		metroPack("win10", "Windows 10", 2015, "Windows 10's flat desktop: accent-blue focus, thin scroll bars, pill toggles, rectangular slider thumbs, Explorer's light accent selection.", ThemeLight, 0, win10),
		metroPack("win10-night", "Windows 10 Dark", 2018, "Windows 10's 2018 dark mode: the same flat shapes on dark greys with the accent blue.", ThemeDark, 2, dark),
	}
}

// ---- helpers shared with the Fluent engine ------------------------------------------------------

// winChevron strokes a thin chevron (Segoe MDL2's) centred in b: 8 units
// across and 4 deep, one line thick.
func winChevron(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 {
		return
	}
	u := min(l.S(1), min(b.Dx(), b.Dy())/10)
	lw := max(winPx(l)*0.9, u)
	cx, cy := snap((b.Min.X+b.Max.X)*0.5), snap((b.Min.Y+b.Max.Y)*0.5)
	a, d := 4*u, 2*u
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-a, cy+d)
		p.LineTo(cx, cy-d)
		p.LineTo(cx+a, cy+d)
	case DirDown:
		p.MoveTo(cx-a, cy-d)
		p.LineTo(cx, cy+d)
		p.LineTo(cx+a, cy-d)
	case DirLeft:
		p.MoveTo(cx+d, cy-a)
		p.LineTo(cx-d, cy)
		p.LineTo(cx+d, cy+a)
	default:
		p.MoveTo(cx-d, cy-a)
		p.LineTo(cx+d, cy)
		p.LineTo(cx-d, cy+a)
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: lw, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// winFlatDisc paints a flat message-box disc with a glyph.
func winFlatDisc(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, fill paintengine2d.Color, glyph string, gc paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	if s < 4 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	r := s * 0.46
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(fill))
	if glyph == "×" {
		k := r * 0.36
		winCross(ctx, paintengine2d.XYWH(cx-k, cy-k, 2*k, 2*k), gc, r*0.16)
		return
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	w := f.Advance(glyph)
	f.Draw(ctx, glyph, paintengine2d.Pt(cx-w*0.5, cy-f.Height()*0.5), gc)
}

// winFlatTriangle paints a flat warning triangle with a "!".
func winFlatTriangle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, fill, gc paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	if s < 4 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	r := s * 0.47
	p := paintengine2d.NewPath()
	p.MoveTo(cx, cy-r)
	p.LineTo(cx+r, cy+r*0.82)
	p.LineTo(cx-r, cy+r*0.82)
	p.Close()
	ctx.DrawPath(p, paintengine2d.Paint{Color: fill, Style: paintengine2d.StyleStrokeAndFill,
		Stroke: paintengine2d.Stroke{Width: winPx(l), Join: paintengine2d.JoinRound, Cap: paintengine2d.CapRound, MiterLimit: 4}})
	f := l.bold
	if f == nil {
		f = l.body
	}
	w := f.Advance("!")
	f.Draw(ctx, "!", paintengine2d.Pt(cx-w*0.5, cy-f.Height()*0.5+r*0.2), gc)
}
