package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// aeroEngine paints Windows Vista (2007) and Windows 7 (2009) "Aero", the
// visual style of the aero.msstyles era, from the documented part and state
// names (BP_PUSHBUTTON, SBP_THUMBBTNVERT, TABP_TABITEM, LVP_LISTITEM…), the
// Vista and 7 UX guidelines and the colours of the shipped controls, as
// numbers:
//
//   - push buttons are glass: a 3px-rounded #707070 border, a two-tone face
//     (a light top half over a darker bottom half) and a white inner
//     highlight; hot turns the face and border blue, pressed deepens the
//     blue and shades the inner top edge; the default button keeps a blue
//     border and a light-blue inner glow (static: Windows pulsed it);
//   - check boxes and radios are 13px wells with a diagonal grey gradient
//     that glows blue when hot; the tick is dark blue-grey and the radio
//     dot a small blue orb;
//   - scroll bars are 17px shafts; thumbs are rounded glass with a
//     four-line grip; arrow buttons show only their glyph until the pointer
//     is over the bar, when they show their chrome (Windows 7's hover
//     state) and turn blue under the pointer;
//   - drop-down lists are glass buttons with the small black arrow;
//   - tabs are glass with rounded tops; the selected one is white, taller
//     and wider, and opens into the white page;
//   - progress bars fill with the glossy green gradient (its moving shine
//     painted as a static highlight);
//   - menus have the light gutter with its etched edge and a rounded
//     blue-glass hot item; tree expanders are the hollow and filled Vista
//     triangles; list, tree and table selections are Explorer's rounded
//     light-blue boxes (a paler box for hover, grey when the view is not
//     focused); tool tips fade from white to #e4e5f0.
//
// In-app windows get the glass frame: a light blue translucent-looking
// caption with a sheen, the title in black over a soft white glow and the
// red close button. Windows 7 Basic ("aero-basic", params "glass" 0) has
// the same controls under an opaque frame.
//
// Pack data: every colour key of aeroBase can be overridden through the
// pack's "extra" map; params "glass" (1 = Aero Glass frame, 0 = Basic).
type aeroEngine struct{ BaseEngine }

func init() {
	RegisterEngine(aeroEngine{})
	for _, p := range aeroPacks() {
		RegisterPack(p)
	}
}

func (aeroEngine) ID() string { return "aero" }

// DefaultMetrics are Windows 7's proportions at the toolkit's 16px UI font
// (Windows used 9pt Segoe UI, 12px): 23px buttons become 30, 17px scroll
// bars and 13px check boxes stay, corners are 3px.
func (aeroEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Radius:    3, RadiusSmall: 3,
		ControlH: 30, FieldH: 28, ComboH: 28,
		Checkbox: 13, Radio: 13,
		MenuItemH: 26, MenuBarH: 26, TabH: 28, RowH: 24,
		TitleBar: 30, HeaderH: 26, ProgressH: 18, SliderH: 26, Thumb: 11,
		Scroll: 17, Pad: 10, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 26, SpinnerW: 17,
	}
}

// ---- colour table ----------------------------------------------------------------

// aeroScheme maps a colour key to "#rrggbb". aeroBase holds every key;
// aeroBasic lists what Windows 7 Basic changes (the window frame only).
type aeroScheme map[string]string

var aeroBase = aeroScheme{
	// Window face, text and the classic highlight (COLOR_HIGHLIGHT).
	"face": "#f0f0f0", "text": "#000000", "gray": "#6d6d6d", "disText": "#838383",
	"field": "#ffffff", "sel": "#3399ff", "selText": "#ffffff",

	// Push buttons: border and the two-tone face (top half, bottom half).
	"btnBorder": "#707070", "btnTop0": "#f2f2f2", "btnTop1": "#ebebeb", "btnBot0": "#dddddd", "btnBot1": "#cfcfcf",
	"hotBorder": "#3c7fb1", "hotTop0": "#eaf6fd", "hotTop1": "#d9f0fc", "hotBot0": "#bee6fd", "hotBot1": "#a7d9f5",
	"pressBorder": "#2c628b", "pressTop0": "#e5f4fc", "pressTop1": "#c4e5f6", "pressBot0": "#98d1ef", "pressBot1": "#68b3db",
	"disBorder": "#adb2b5", "disFace": "#f4f4f4", "defGlow": "#4cc4f8",

	// Check boxes and radios: border per state, the white ring, the
	// diagonal well per state, the tick and the radio orb.
	"chkBorder": "#8e8f8f", "chkHotBorder": "#3c7fb1", "chkPressBorder": "#2c628b", "chkDisBorder": "#bcbcbc",
	"chkRing":  "#f4f4f4",
	"chkWell0": "#cbcfd5", "chkWell1": "#f7f7f7", "chkHot0": "#b1dffd", "chkHot1": "#e9f7fe",
	"chkPress0": "#7ec3ee", "chkPress1": "#dcf1fd", "chkDis0": "#f4f4f4", "chkDis1": "#f4f4f4",
	"tick": "#4a5f97", "tickDis": "#aeb3b9",
	"dot0": "#e3f6ff", "dot1": "#63bdf2", "dot2": "#1f5fae", "dotRim": "#1b4f8c", "dotDis": "#aeb3b9",

	// Edit boxes: top edge, sides and bottom edge per state; list and tree
	// views are framed in viewBorder.
	"edTop": "#abadb3", "edSide": "#e2e3ea", "edBot": "#e3e9ef",
	"edHotTop": "#5794bf", "edHotSide": "#b7d5ea", "edHotBot": "#c7e2f1",
	"edFocTop": "#3d7bad", "edFocSide": "#a4c9e3", "edFocBot": "#b7d9ed",
	"edDisBorder": "#afafaf", "edDis": "#f4f4f4", "viewBorder": "#828790",

	// Scroll bars: the shaft across the bar, a paged shaft, the thumb (also
	// the arrow buttons' chrome), its grip and the arrow glyphs.
	"sbTrack0": "#e3e3e3", "sbTrack1": "#eeeeee", "sbTrack2": "#f5f5f5",
	"sbPage0": "#c7c7c7", "sbPage1": "#d3d3d3", "sbPage2": "#dcdcdc",
	"thBorder": "#979797", "thTop0": "#f5f5f5", "thTop1": "#ebebeb", "thBot0": "#dbdbdb", "thBot1": "#cfcfcf",
	"grip": "#838383", "gripLt": "#fcfcfc", "gripHot": "#2c628b", "gripPress": "#1b4467",
	"sbGlyph": "#4d4d4d", "sbGlyphHot": "#000000", "sbGlyphDis": "#bfbfbf",

	// Tabs and the page under them.
	"tabBorder": "#898c95", "pane": "#ffffff",

	// Progress bar: trough, the green fill (top to bottom) and its edge.
	"progBorder": "#bcbcbc", "progT0": "#dadada", "progT1": "#ebebeb", "progT2": "#f6f6f6",
	"prog0": "#b3eda8", "prog1": "#5dd457", "prog2": "#17b615", "prog3": "#3fc73b", "progEdge": "#2c9d29",

	// Trackbar groove.
	"groove": "#e7eaea", "grooveBorder": "#b6babf",

	// Explorer's rounded boxes: hover, selected, selected + hot, selected in
	// an unfocused view, the keyboard focus rectangle.
	"hovBorder": "#b8d6fb", "hov0": "#fafbfd", "hov1": "#ebf3fd",
	"selBorder": "#84acdd", "sel0": "#dcebfc", "sel1": "#c1dbfc",
	"selHotBorder": "#7da2ce", "selHot0": "#d3e6fc", "selHot1": "#b3d3fa",
	"offBorder": "#d9d9d9", "off0": "#f8f8f8", "off1": "#e5e5e5",
	"focusBorder": "#7da2ce",

	// Menus: frame, background, the gutter and its etched edge,
	// separators, the hot item; the menu bar and its hot and open titles.
	"menuBg": "#f0f0f0", "menuBorder": "#979797", "menuInner": "#f8f8f8",
	"gutter0": "#f7f7f7", "gutter1": "#ececec", "gutterLine": "#e2e3e3", "gutterLineLt": "#ffffff",
	"menuSep": "#e0e0e0", "menuSepLt": "#ffffff", "menuDisText": "#6d6d6d",
	"menuHotBorder": "#aecff7", "menuHot0": "#f1f7fe", "menuHot1": "#dcebfc",
	"mbar0": "#fdfdfd", "mbar1": "#e8ecf1",
	"mbHotBorder": "#b8d6fb", "mbHot0": "#f9fbfd", "mbHot1": "#e3eefb",
	"mbOpenBorder": "#8fa8c9", "mbOpen0": "#e4ecf7", "mbOpen1": "#cedcf0",

	// Tree expanders: the filled (open) and hollow (closed) triangles.
	"expOpen": "#595959", "expOpenEdge": "#262626", "expClosed": "#a6a6a6", "expClosedFill": "#ffffff",
	"expHot": "#1cc4f7", "expHotEdge": "#1ba1c9", "expHotFill": "#e5f8fe",

	// List-view header.
	"hdrTop": "#ffffff", "hdrBot0": "#f7f8fa", "hdrBot1": "#f1f2f4", "hdrDiv": "#e3e5e8", "hdrLine": "#d5d5d5",
	"hdrHotBorder": "#88cbeb", "hdrHot0": "#e3f7ff", "hdrHot1": "#bdedff",
	"hdrPressBorder": "#61a4c4", "hdrPress0": "#bce4f9", "hdrPress1": "#8dd6f7",
	"hdrText": "#4c607a", "hdrArrow": "#97a5b8",

	// Tool tip.
	"tipBorder": "#767676", "tip0": "#ffffff", "tip1": "#e4e5f0", "tipText": "#575757",

	// Group box, the dialog's main instruction, Explorer's group headers.
	"groupBorder": "#d5dfe5", "groupText": "#000000", "instr": "#003399", "groupHead": "#1e3287", "groupLine": "#e2e8f4",

	// Status bar and its size grip.
	"stat0": "#fafbfc", "stat1": "#e9ecef", "statTop": "#d6dbe3", "statSep": "#d6dbe3", "statSepLt": "#ffffff",
	"gripDot": "#b3bcc8", "gripDotLt": "#ffffff",

	// Tool bar band and its grip.
	"tb0": "#ffffff", "tb1": "#f4f6f9", "tb2": "#e3e8ef", "tbBorder": "#d1d9e4", "tbGrip": "#a7b1c0", "tbGripLt": "#ffffff",

	// Etched separators (button shadow over button highlight).
	"etchDk": "#a0a0a0", "etchLt": "#ffffff",

	// Window frame: the glass (active, inactive), its edges, the caption
	// text, and the red close button.
	"frameEdge": "#4d5d72", "clientEdge": "#8190a6",
	"glass0": "#c9daee", "glass1": "#a7c1e1", "glass2": "#bdd1ea",
	"glassOff0": "#e0e8f2", "glassOff1": "#d0dbe9", "glassOff2": "#d9e3ef",
	"capText": "#000000", "capTextOff": "#4a4a4a",
	"closeBorder": "#6e2a21", "close0": "#e8a493", "close1": "#d98f7d", "close2": "#c75a3f", "close3": "#d06a4f",
	"closeHot0": "#f6b9a8", "closeHot1": "#eb846a", "closeHot2": "#d83d25", "closeHot3": "#e56346",
	"closePress0": "#d18e80", "closePress1": "#b6503b", "closePress2": "#9a230f", "closePress3": "#ad4029",
	"closeOffBorder": "#8d9db3", "closeOff0": "#eef2f7", "closeOff1": "#e0e7f0", "closeOff2": "#d3dce8", "closeOff3": "#dae2ec",
	"closeGlyph": "#ffffff", "closeGlyphEdge": "#5e1d14", "closeGlyphOff": "#4a5a70",
}

// aeroBasic is Windows 7 Basic: opaque, flatter frame colours.
var aeroBasic = aeroScheme{
	"glass0": "#c3d5ee", "glass1": "#b1c8e8", "glass2": "#bdd0ec",
	"glassOff0": "#dde6f2", "glassOff1": "#d3ddec", "glassOff2": "#d9e2ef",
	"frameEdge": "#6d7f97", "clientEdge": "#8e9db2",
}

// ---- resolved colour set ----------------------------------------------------------

// aeroGlass is one state of a glass face: border, inner highlight ring and
// the face gradient.
type aeroGlass struct {
	border, hi paintengine2d.Color
	face       []paintengine2d.GradientStop
}

// aeroBox is one of Explorer's rounded boxes: border and face gradient
// (with a white inner ring).
type aeroBox struct {
	border paintengine2d.Color
	face   []paintengine2d.GradientStop
}

// aeroSet is a look's resolved Aero colours, built once per look.
type aeroSet struct {
	glass bool

	face, text, gray, disText, field, sel, selText paintengine2d.Color

	btn, btnHot, btnPress, btnDis, btnDef aeroGlass
	defGlow                               [2]paintengine2d.Color

	chkBorder         [4]paintengine2d.Color // normal, hot, pressed, disabled
	chkWell           [4][]paintengine2d.GradientStop
	chkRing           paintengine2d.Color
	tick, tickDis     paintengine2d.Color
	dot               []paintengine2d.GradientStop
	dotRim, dotDis    paintengine2d.Color
	ed                [4][3]paintengine2d.Color // normal, hot, focused, disabled
	edDis, viewBorder paintengine2d.Color

	track, page                     []paintengine2d.GradientStop
	thumb, thumbHot, thumbPress     aeroGlass
	grip                            [3][2]paintengine2d.Color
	sbGlyph, sbGlyphHot, sbGlyphDis paintengine2d.Color

	tabBorder, pane paintengine2d.Color

	progBorder, progEdge  paintengine2d.Color
	trough, prog, shine   []paintengine2d.GradientStop
	marqL, marqR          []paintengine2d.GradientStop
	groove, grooveBorder  paintengine2d.Color
	knobOn                []paintengine2d.GradientStop
	hov, selBox, selHot   aeroBox
	off, menuHot          aeroBox
	mbHot, mbOpen         aeroBox
	focusBorder, innerHi  paintengine2d.Color
	pressShade, pressSide paintengine2d.Color

	menuBg, menuBorder, menuInner          paintengine2d.Color
	gutterLine, gutterLineLt               paintengine2d.Color
	menuSep, menuSepLt, menuDisText        paintengine2d.Color
	gutter, mbar                           []paintengine2d.GradientStop
	expOpen, expOpenEdge, expClosed, expCF paintengine2d.Color
	expHot, expHotEdge, expHotFill         paintengine2d.Color

	hdrStops, hdrHotStops, hdrPressStops                []paintengine2d.GradientStop
	hdrDiv, hdrLine, hdrHotBorder, hdrPressBorder       paintengine2d.Color
	hdrText, hdrArrow                                   paintengine2d.Color
	tipBorder, tipText                                  paintengine2d.Color
	tip                                                 []paintengine2d.GradientStop
	groupBorder, groupText, instr, groupHead, groupLine paintengine2d.Color
	groupFade                                           []paintengine2d.GradientStop
	status                                              []paintengine2d.GradientStop
	statTop, statSep, statSepLt, gripDot, gripDotLt     paintengine2d.Color
	tb                                                  []paintengine2d.GradientStop
	tbBorder, tbGrip, tbGripLt                          paintengine2d.Color
	etchDk, etchLt                                      paintengine2d.Color
	frameEdge, clientEdge, capText, capTextOff          paintengine2d.Color
	glassGrad, glassOff, sheen                          []paintengine2d.GradientStop
	close, closeHot, closePress, closeOff               aeroGlass
	closeGlyph, closeGlyphEdge, closeGlyphOff, capGlow  paintengine2d.Color
	errDisc, infoDisc, warnFace, gloss                  []paintengine2d.GradientStop
	errRim, infoRim, warnRim                            paintengine2d.Color
}

type aeroKey struct{}

// aeroColors is the look's resolved colour set (built once per look).
func aeroColors(l *Classic) *aeroSet {
	return l.Memo(aeroKey{}, func() any { return aeroBuild(l) }).(*aeroSet)
}

func aeroBuild(l *Classic) *aeroSet {
	glass := l.P("glass", 1) != 0
	col := func(k string) paintengine2d.Color {
		v, ok := aeroBase[k]
		if !glass {
			if b, ok2 := aeroBasic[k]; ok2 {
				v, ok = b, true
			}
		}
		if !ok {
			v = "#ff00ff" // a missing key shows (and the table test fails)
		}
		return l.X(k, Hex(v))
	}
	two := func(a, b string) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col(a)), Stop(1, col(b))}
	}
	// tone is the glass face: the top half fades a→b, the bottom half c→d.
	tone := func(a, b, c, d string) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col(a)), Stop(0.48, col(b)), Stop(0.48, col(c)), Stop(1, col(d))}
	}
	white := paintengine2d.RGB(1, 1, 1)
	s := &aeroSet{glass: glass}
	s.face, s.text, s.gray, s.disText = col("face"), col("text"), col("gray"), col("disText")
	s.field, s.sel = col("field"), col("sel")
	s.selText = ReadableOn(s.sel, 3, col("selText"), s.text)

	s.btn = aeroGlass{border: col("btnBorder"), hi: white.WithAlpha(0.72), face: tone("btnTop0", "btnTop1", "btnBot0", "btnBot1")}
	s.btnHot = aeroGlass{border: col("hotBorder"), hi: white.WithAlpha(0.6), face: tone("hotTop0", "hotTop1", "hotBot0", "hotBot1")}
	s.btnPress = aeroGlass{border: col("pressBorder"), face: tone("pressTop0", "pressTop1", "pressBot0", "pressBot1")}
	s.btnDis = aeroGlass{border: col("disBorder"), face: two("disFace", "disFace")}
	s.btnDef = aeroGlass{border: col("hotBorder"), face: s.btn.face}
	g := col("defGlow")
	s.defGlow = [2]paintengine2d.Color{g.WithAlpha(0.8), g.WithAlpha(0.3)}
	s.innerHi = white.WithAlpha(0.55)
	s.pressShade, s.pressSide = paintengine2d.RGBA(0, 0, 0, 0.2), paintengine2d.RGBA(0, 0, 0, 0.08)

	s.chkBorder = [4]paintengine2d.Color{col("chkBorder"), col("chkHotBorder"), col("chkPressBorder"), col("chkDisBorder")}
	s.chkWell = [4][]paintengine2d.GradientStop{two("chkWell0", "chkWell1"), two("chkHot0", "chkHot1"), two("chkPress0", "chkPress1"), two("chkDis0", "chkDis1")}
	s.chkRing, s.tick, s.tickDis = col("chkRing"), col("tick"), col("tickDis")
	s.dot = []paintengine2d.GradientStop{Stop(0, col("dot0")), Stop(0.45, col("dot1")), Stop(1, col("dot2"))}
	s.dotRim, s.dotDis = col("dotRim"), col("dotDis")
	s.ed = [4][3]paintengine2d.Color{
		{col("edTop"), col("edSide"), col("edBot")},
		{col("edHotTop"), col("edHotSide"), col("edHotBot")},
		{col("edFocTop"), col("edFocSide"), col("edFocBot")},
		{col("edDisBorder"), col("edDisBorder"), col("edDisBorder")},
	}
	s.edDis, s.viewBorder = col("edDis"), col("viewBorder")

	s.track = []paintengine2d.GradientStop{Stop(0, col("sbTrack0")), Stop(0.2, col("sbTrack1")), Stop(1, col("sbTrack2"))}
	s.page = []paintengine2d.GradientStop{Stop(0, col("sbPage0")), Stop(0.2, col("sbPage1")), Stop(1, col("sbPage2"))}
	s.thumb = aeroGlass{border: col("thBorder"), hi: white.WithAlpha(0.7), face: tone("thTop0", "thTop1", "thBot0", "thBot1")}
	s.thumbHot = aeroGlass{border: col("hotBorder"), hi: white.WithAlpha(0.6), face: s.btnHot.face}
	s.thumbPress = aeroGlass{border: col("pressBorder"), hi: white.WithAlpha(0.25), face: s.btnPress.face}
	s.grip = [3][2]paintengine2d.Color{{col("grip"), col("gripLt")}, {col("gripHot"), col("gripLt")}, {col("gripPress"), col("gripLt")}}
	s.sbGlyph, s.sbGlyphHot, s.sbGlyphDis = col("sbGlyph"), col("sbGlyphHot"), col("sbGlyphDis")

	s.tabBorder, s.pane = col("tabBorder"), col("pane")

	s.progBorder, s.progEdge = col("progBorder"), col("progEdge")
	s.trough = []paintengine2d.GradientStop{Stop(0, col("progT0")), Stop(0.3, col("progT1")), Stop(1, col("progT2"))}
	s.prog = []paintengine2d.GradientStop{Stop(0, col("prog0")), Stop(0.45, col("prog1")), Stop(0.5, col("prog2")), Stop(1, col("prog3"))}
	s.shine = []paintengine2d.GradientStop{Stop(0, white.WithAlpha(0)), Stop(0.5, white.WithAlpha(0.32)), Stop(1, white.WithAlpha(0))}
	mid := col("prog1")
	s.marqL = []paintengine2d.GradientStop{Stop(0, mid.WithAlpha(0)), Stop(1, mid)}
	s.marqR = []paintengine2d.GradientStop{Stop(0, mid), Stop(1, mid.WithAlpha(0))}
	s.groove, s.grooveBorder = col("groove"), col("grooveBorder")
	s.knobOn = s.prog

	s.hov = aeroBox{border: col("hovBorder"), face: two("hov0", "hov1")}
	s.selBox = aeroBox{border: col("selBorder"), face: two("sel0", "sel1")}
	s.selHot = aeroBox{border: col("selHotBorder"), face: two("selHot0", "selHot1")}
	s.off = aeroBox{border: col("offBorder"), face: two("off0", "off1")}
	s.menuHot = aeroBox{border: col("menuHotBorder"), face: two("menuHot0", "menuHot1")}
	s.mbHot = aeroBox{border: col("mbHotBorder"), face: two("mbHot0", "mbHot1")}
	s.mbOpen = aeroBox{border: col("mbOpenBorder"), face: two("mbOpen0", "mbOpen1")}
	s.focusBorder = col("focusBorder")

	s.menuBg, s.menuBorder, s.menuInner = col("menuBg"), col("menuBorder"), col("menuInner")
	s.gutterLine, s.gutterLineLt = col("gutterLine"), col("gutterLineLt")
	s.menuSep, s.menuSepLt, s.menuDisText = col("menuSep"), col("menuSepLt"), col("menuDisText")
	s.gutter, s.mbar = two("gutter0", "gutter1"), two("mbar0", "mbar1")
	s.expOpen, s.expOpenEdge, s.expClosed, s.expCF = col("expOpen"), col("expOpenEdge"), col("expClosed"), col("expClosedFill")
	s.expHot, s.expHotEdge, s.expHotFill = col("expHot"), col("expHotEdge"), col("expHotFill")

	s.hdrStops = []paintengine2d.GradientStop{Stop(0, col("hdrTop")), Stop(0.4, col("hdrTop")), Stop(0.4, col("hdrBot0")), Stop(1, col("hdrBot1"))}
	s.hdrHotStops = two("hdrHot0", "hdrHot1")
	s.hdrPressStops = two("hdrPress0", "hdrPress1")
	s.hdrDiv, s.hdrLine = col("hdrDiv"), col("hdrLine")
	s.hdrHotBorder, s.hdrPressBorder = col("hdrHotBorder"), col("hdrPressBorder")
	s.hdrText, s.hdrArrow = col("hdrText"), col("hdrArrow")
	s.tipBorder, s.tipText, s.tip = col("tipBorder"), col("tipText"), two("tip0", "tip1")
	s.groupBorder, s.groupText, s.instr = col("groupBorder"), col("groupText"), col("instr")
	s.groupHead, s.groupLine = col("groupHead"), col("groupLine")
	s.groupFade = []paintengine2d.GradientStop{Stop(0, s.groupLine), Stop(1, s.groupLine.WithAlpha(0))}
	s.status = two("stat0", "stat1")
	s.statTop, s.statSep, s.statSepLt = col("statTop"), col("statSep"), col("statSepLt")
	s.gripDot, s.gripDotLt = col("gripDot"), col("gripDotLt")
	s.tb = []paintengine2d.GradientStop{Stop(0, col("tb0")), Stop(0.5, col("tb1")), Stop(1, col("tb2"))}
	s.tbBorder, s.tbGrip, s.tbGripLt = col("tbBorder"), col("tbGrip"), col("tbGripLt")
	s.etchDk, s.etchLt = col("etchDk"), col("etchLt")

	s.frameEdge, s.clientEdge = col("frameEdge"), col("clientEdge")
	s.capText, s.capTextOff = col("capText"), col("capTextOff")
	s.glassGrad = []paintengine2d.GradientStop{Stop(0, col("glass0")), Stop(0.5, col("glass1")), Stop(1, col("glass2"))}
	s.glassOff = []paintengine2d.GradientStop{Stop(0, col("glassOff0")), Stop(0.5, col("glassOff1")), Stop(1, col("glassOff2"))}
	s.sheen = []paintengine2d.GradientStop{Stop(0, white.WithAlpha(0.5)), Stop(1, white.WithAlpha(0))}
	closeHi := white.WithAlpha(0.35)
	s.close = aeroGlass{border: col("closeBorder"), hi: closeHi, face: tone("close0", "close1", "close2", "close3")}
	s.closeHot = aeroGlass{border: col("closeBorder"), hi: closeHi, face: tone("closeHot0", "closeHot1", "closeHot2", "closeHot3")}
	s.closePress = aeroGlass{border: col("closeBorder"), face: tone("closePress0", "closePress1", "closePress2", "closePress3")}
	s.closeOff = aeroGlass{border: col("closeOffBorder"), hi: white.WithAlpha(0.5), face: tone("closeOff0", "closeOff1", "closeOff2", "closeOff3")}
	s.closeGlyph, s.closeGlyphEdge, s.closeGlyphOff = col("closeGlyph"), col("closeGlyphEdge"), col("closeGlyphOff")
	s.capGlow = white.WithAlpha(0.78)

	s.errDisc = []paintengine2d.GradientStop{Stop(0, Hex("#ffb4a3")), Stop(0.5, Hex("#e8412c")), Stop(1, Hex("#a51b0b"))}
	s.infoDisc = []paintengine2d.GradientStop{Stop(0, Hex("#cfe8ff")), Stop(0.5, Hex("#3e8ee5")), Stop(1, Hex("#15469e"))}
	s.warnFace = []paintengine2d.GradientStop{Stop(0, Hex("#fff39a")), Stop(1, Hex("#f2b10b"))}
	s.gloss = []paintengine2d.GradientStop{Stop(0, white.WithAlpha(0.75)), Stop(1, white.WithAlpha(0.05))}
	s.errRim, s.infoRim, s.warnRim = Hex("#7d1206"), Hex("#123c86"), Hex("#a36a00")
	return s
}

// chkIndex picks the check box / radio / glass state row: normal, hot,
// pressed, disabled.
func aeroChkIndex(st ControlState) int {
	switch {
	case st.Disabled():
		return 3
	case st.Pressed():
		return 2
	case st.Hovered():
		return 1
	}
	return 0
}

// ---- parts: glass faces and Explorer boxes -------------------------------------------

// paintGlass paints a glass face into b: the rounded border, the two-tone
// face (across = the gradient runs left to right, for vertical thumbs) and
// the white inner highlight. It returns the face rect inside the border.
func (c *aeroSet) paintGlass(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, g *aeroGlass, r float32, across bool) paintengine2d.Rect {
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return paintengine2d.Rect{}
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(g.border))
	in := b.Inset(lw)
	ri := max(r-lw, 0)
	paint := VGradient(in, g.face...)
	if across {
		paint = HGradient(in, g.face...)
	}
	ctx.DrawRoundRect(in, ri, ri, paint)
	if g.hi.A > 0 && in.Dx() > 3*lw && in.Dy() > 3*lw {
		winRing(ctx, in, ri, lw, paintengine2d.Fill(g.hi))
	}
	return in
}

// pressShadow shades the inner top and left edges of a pressed face.
func (c *aeroSet) pressShadow(l *Classic, ctx *paintengine2d.Context, in paintengine2d.Rect, r float32) {
	lw := winPx(l)
	if in.Dx() < 3*lw || in.Dy() < 3*lw {
		return
	}
	ctx.Save()
	ctx.ClipRoundRect(in, r, r)
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(c.pressShade))
	p := paintengine2d.NewPath()
	p.AddRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, in.Dx(), lw))
	p.AddRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+2*lw, lw, in.Dy()-2*lw))
	ctx.DrawPath(p, paintengine2d.Fill(c.pressSide))
	ctx.Restore()
}

// pushButton paints a push button face for st and returns the label colour.
func (c *aeroSet) pushButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	r := l.rx(3)
	lw := winPx(l)
	switch {
	case st.Disabled():
		c.paintGlass(l, ctx, b, &c.btnDis, r, false)
		return c.disText
	case st.Pressed():
		in := c.paintGlass(l, ctx, b, &c.btnPress, r, false)
		c.pressShadow(l, ctx, in, max(r-lw, 0))
	case st.Hovered():
		c.paintGlass(l, ctx, b, &c.btnHot, r, false)
	case st.Primary() || st.Focused():
		// The default button: a blue border and the light-blue inner glow
		// (Windows animated it between this and the hot face).
		in := c.paintGlass(l, ctx, b, &c.btnDef, r, false)
		c.glow(l, ctx, in, max(r-lw, 0))
	default:
		c.paintGlass(l, ctx, b, &c.btn, r, false)
	}
	return c.text
}

// glow paints the default button's two-line inner glow inside in.
func (c *aeroSet) glow(l *Classic, ctx *paintengine2d.Context, in paintengine2d.Rect, r float32) {
	lw := winPx(l)
	if in.Dx() < 5*lw || in.Dy() < 5*lw {
		return
	}
	winRing(ctx, in, r, lw, paintengine2d.Fill(c.defGlow[0]))
	winRing(ctx, in.Inset(lw), max(r-lw, 0), lw, paintengine2d.Fill(c.defGlow[1]))
}

// paintBox paints one of Explorer's rounded boxes (hover, selection, menu
// hot item) into b.
func (c *aeroSet) paintBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, bx *aeroBox) {
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return
	}
	r := l.rx(3)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(bx.border))
	in := b.Inset(lw)
	ri := max(r-lw, 0)
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, bx.face...))
	if in.Dx() > 3*lw && in.Dy() > 3*lw {
		winRing(ctx, in, ri, lw, paintengine2d.Fill(c.innerHi))
	}
}

// rowBox picks the Explorer box of a row state (nil when the row is plain).
func (c *aeroSet) rowBox(st ControlState) *aeroBox {
	switch {
	case st.Disabled():
		if st.Checked() {
			return &c.off
		}
		return nil
	case st.Checked() && (st.Inactive() || st.Backdrop()):
		return &c.off
	case st.Checked() && st.Hovered():
		return &c.selHot
	case st.Checked():
		return &c.selBox
	case st.Hovered():
		return &c.hov
	}
	return nil
}

// shaft fills a scroll bar shaft with its gradient across the bar.
func (c *aeroSet) shaft(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, stops []paintengine2d.GradientStop) {
	if b.Empty() {
		return
	}
	if vertical {
		ctx.DrawRect(b, HGradient(b, stops...))
		return
	}
	ctx.DrawRect(b, VGradient(b, stops...))
}

// edit paints an edit box: white, a 1px border darker along the top, blue
// when hot or focused.
func (c *aeroSet) edit(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	b = winSnap(b)
	lw := winPx(l)
	if b.Empty() {
		return
	}
	i := 0
	fill := c.field
	switch {
	case st.Disabled():
		i, fill = 3, c.edDis
	case st.Focused():
		i = 2
	case st.Hovered():
		i = 1
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return
	}
	e := c.ed[i]
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(e[0]))
	sides := paintengine2d.NewPath()
	sides.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, lw, b.Dy()-2*lw))
	sides.AddRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+lw, lw, b.Dy()-2*lw))
	ctx.DrawPath(sides, paintengine2d.Fill(e[1]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(e[2]))
}

// ---- parts --------------------------------------------------------------------------------

func (e aeroEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := aeroColors(l)
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
		if bx := c.rowBox(st); bx != nil {
			c.paintBox(l, ctx, b, bx)
		}
		if !st.Disabled() {
			fg = l.fieldText()
		}
	case RoleMenu:
		e.MenuHighlight(l, ctx, b, false)
	case RoleThumb:
		g := &c.thumb
		switch aeroChkIndex(st) {
		case 1:
			g = &c.thumbHot
		case 2:
			g = &c.thumbPress
		}
		c.paintGlass(l, ctx, b, g, l.rx(2), b.Dy() > b.Dx())
	case RoleTrack:
		c.shaft(ctx, winSnap(b), b.Dy() >= b.Dx(), c.track)
	case RoleTab:
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
	}
	return fg
}

// toolFace is a tool button: flat until hot, then a blue-glass box; pressed
// and latched buttons take the selection box.
func (c *aeroSet) toolFace(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	if st.Disabled() {
		return c.disText
	}
	switch {
	case st.Pressed(), st.Checked() && st.Hovered():
		c.paintBox(l, ctx, b, &c.selHot)
		lw := winPx(l)
		in := winSnap(b).Inset(lw)
		c.pressShadow(l, ctx, in, max(l.rx(3)-lw, 0))
	case st.Checked():
		c.paintBox(l, ctx, b, &c.selBox)
	case st.Hovered() || st.Focused():
		c.paintBox(l, ctx, b, &c.menuHot)
	}
	return c.text
}

func (e aeroEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := aeroColors(l)
	box = winSnap(box)
	lw := winPx(l)
	if box.Dx() < 5*lw || box.Dy() < 5*lw {
		return
	}
	i := aeroChkIndex(st)
	ctx.DrawRect(box, paintengine2d.Fill(c.chkBorder[i]))
	in := box.Inset(lw)
	ctx.DrawRect(in, paintengine2d.Fill(c.chkRing))
	well := in.Inset(lw)
	ctx.DrawRect(well, DGradient(well, c.chkWell[i]...))
	if checked || st.Checked() {
		tick := c.tick
		if st.Disabled() {
			tick = c.tickDis
		}
		winTick(ctx, box, tick, box.Dx()*2/13)
	}
}

func (e aeroEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := aeroColors(l)
	box = winSnap(box)
	lw := winPx(l)
	r := min(box.Dx(), box.Dy()) * 0.5
	if r < 3*lw {
		return
	}
	cx, cy := (box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5
	ctr := paintengine2d.Pt(cx, cy)
	i := aeroChkIndex(st)
	ctx.DrawCircle(ctr, r, paintengine2d.Fill(c.chkBorder[i]))
	ctx.DrawCircle(ctr, r-lw, paintengine2d.Fill(c.chkRing))
	rw := r - 2*lw
	ctx.DrawCircle(ctr, rw, DGradient(paintengine2d.XYWH(cx-rw, cy-rw, 2*rw, 2*rw), c.chkWell[i]...))
	if !selected && !st.Checked() {
		return
	}
	rd := r * 0.44
	if st.Disabled() {
		ctx.DrawCircle(ctr, rd, paintengine2d.Fill(c.dotDis))
		return
	}
	ctx.DrawCircle(ctr, rd, paintengine2d.Fill(c.dotRim))
	ctx.DrawCircle(ctr, rd-lw*0.75, paintengine2d.Radial(paintengine2d.RadialGradient{
		Center: paintengine2d.Pt(cx-rd*0.3, cy-rd*0.35), Radius: rd * 1.3, Stops: c.dot,
	}))
}

// Arrow is the small solid Aero glyph (8×4 at 1x), fitted into b.
func (aeroEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	u := s / 10
	if m := l.S(1); u > m {
		u = m
	}
	winGlyph(ctx, b, dir, u, col)
}

// Expander is Vista's tree glyph: a hollow grey triangle pointing right
// when closed, a filled dark one pointing down-right when open.
func (aeroEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	aeroExpander(l, ctx, b, expanded, false)
}

// aeroExpander is the Vista triangle; under the pointer it turns blue.
func aeroExpander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded, hot bool) {
	c := aeroColors(l)
	if hot {
		winVistaTriangle(l, ctx, b, expanded, c.expHot, c.expHotEdge, c.expHot, c.expHotFill)
		return
	}
	winVistaTriangle(l, ctx, b, expanded, c.expOpen, c.expOpenEdge, c.expClosed, c.expCF)
}

// MenuHighlight is the rounded blue-glass hot item; an open menu-bar title
// is the deeper pushed box.
func (aeroEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := aeroColors(l)
	if attachBottom {
		c.paintBox(l, ctx, b, &c.mbOpen)
		return
	}
	c.paintBox(l, ctx, b, &c.menuHot)
}

// Menu labels stay black on the pale blue glass.
func (aeroEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return aeroColors(l).text
}

// Fields show focus with their blue border (painted by Face).
func (aeroEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is the dotted focus rectangle Windows 7 kept.
func (aeroEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	winDots(ctx, winSnap(b).Inset(winPx(l)), aeroColors(l).text, winPx(l))
}

// ---- scroll bars -----------------------------------------------------------------------

func (aeroEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 17, Arrows: ArrowsEnds, MinThumb: 12}
}

func (e aeroEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := aeroColors(l)
	bar := winSnap(p.Bar)
	if bar.Empty() {
		return
	}
	c.shaft(ctx, bar, vertical, c.track)
	if pg := winPagedPart(p, vertical, st); !pg.Empty() {
		c.shaft(ctx, winSnap(pg), vertical, c.page)
	}
	hover := !st.Disabled && (st.Hovered || st.Hot != ScrollNone || st.Pressed != ScrollNone)
	r := l.rx(2)
	lw := winPx(l)
	u := l.S(1)
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		glyph := c.sbGlyph
		switch {
		case st.Disabled:
			glyph = c.sbGlyphDis
		case st.Pressed == part:
			in := c.paintGlass(l, ctx, b, &c.thumbPress, r, vertical)
			c.pressShadow(l, ctx, in, max(r-lw, 0))
			glyph = c.sbGlyphHot
		case st.Hot == part:
			c.paintGlass(l, ctx, b, &c.thumbHot, r, vertical)
			glyph = c.sbGlyphHot
		case hover:
			// Windows 7 shows the buttons' chrome only while the pointer
			// is over the bar; at rest they are just their glyphs.
			c.paintGlass(l, ctx, b, &c.thumb, r, vertical)
		}
		gu := min(u, min(b.Dx(), b.Dy())/10)
		winGlyph(ctx, b, dir, gu, glyph)
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
	gi := 0
	g := &c.thumb
	switch {
	case st.Pressed == ScrollThumbPart:
		gi, g = 2, &c.thumbPress
	case st.Hot == ScrollThumbPart:
		gi, g = 1, &c.thumbHot
	}
	in := c.paintGlass(l, ctx, p.Thumb, g, r, vertical)
	// The grip: four short ridges, a dark line over a light one.
	along, across := in.Dy(), in.Dx()
	if !vertical {
		along, across = in.Dx(), in.Dy()
	}
	if along < l.S(14) || across < l.S(7) {
		return
	}
	gl := snap(across * 0.5)
	cx, cy := snap((in.Min.X+in.Max.X)*0.5), snap((in.Min.Y+in.Max.Y)*0.5)
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for k := 0; k < 4; k++ {
		o := float32(k*2-4) * lw
		if vertical {
			x0 := cx - snap(gl*0.5)
			dk.AddRect(paintengine2d.XYWH(x0, cy+o, gl, lw))
			lt.AddRect(paintengine2d.XYWH(x0+lw, cy+o+lw, gl, lw))
		} else {
			y0 := cy - snap(gl*0.5)
			dk.AddRect(paintengine2d.XYWH(cx+o, y0, lw, gl))
			lt.AddRect(paintengine2d.XYWH(cx+o+lw, y0+lw, lw, gl))
		}
	}
	ctx.DrawPath(lt, paintengine2d.Fill(c.grip[gi][1]))
	ctx.DrawPath(dk, paintengine2d.Fill(c.grip[gi][0]))
}

// DrawScrollBar is the thumb-in-track fallback for callers without
// scroll bar geometry.
func (e aeroEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), winScrollState(st))
}

// ---- frames ----------------------------------------------------------------------------------

func (aeroEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is BP_GROUPBOX: a 3px-rounded light blue-grey frame with a
// white line inside it, open behind the black title; a raised one is a
// white card.
func (aeroEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := aeroColors(l)
	winGroupFrame(l, ctx, b, title, raised, l.rx(3), c.groupBorder, c.etchLt, c.pane, c.groupText)
}

// aeroCaptionH is the in-app caption: Windows 7's 30px glass caption at
// the toolkit's font.
func aeroCaptionH(l *Classic) float32 {
	h := l.body.Height() + l.S(12)
	if m := l.S(30); h < m {
		h = m
	}
	return snap(h)
}

// aeroFrameW is the glass border around the client area.
func aeroFrameW(l *Classic) float32 { return snap(l.S(7)) }

func (aeroEngine) WindowFrameInsets(l *Classic) Insets {
	fw := aeroFrameW(l)
	return Insets{Top: aeroCaptionH(l), Right: fw, Bottom: fw, Left: fw}
}

// WindowCloseRect is the wide red close button hanging from the top edge at
// the right of the caption (43×19 at 96 DPI, grown with the caption).
func (aeroEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = winSnap(b)
	capH := aeroCaptionH(l)
	h := snap(capH * 0.62)
	w := snap(h * 43 / 19)
	if lim := snap(b.Dx() * 0.3); w > lim {
		w = lim
	}
	right := b.Max.X - aeroFrameW(l) + snap(l.S(2))
	return paintengine2d.XYWH(right-w, b.Min.Y+winPx(l), w, h)
}

// Windows dialogs put the default button first: "OK  Cancel".
func (aeroEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDefaultPulseMs {
		return 2400 // the default button breathes
	}
	if h == HintHoverFadeMs {
		return 200 // Vista and 7 buttons glow in
	}
	if h == HintDialogPrimaryFirst {
		return 1
	}
	if h == HintMnemonics {
		return MnemonicsOnAlt // Windows hides the underlines until Alt
	}
	return 0
}

// DrawWindowFrame is the Aero window: a dark outer edge, the glass (a light
// blue gradient with a sheen and two diagonal streaks; opaque and plain on
// Windows 7 Basic), the client area set in a thin line, the title in black
// over a soft white glow, and the red glass close button.
func (e aeroEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	capH := aeroCaptionH(l)
	fw := aeroFrameW(l)
	if b.Dx() < 4*fw || b.Dy() < capH+2*fw {
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		return
	}
	r := l.rx(6)
	ctx.DrawRoundRectCorners(b, r, r, r*0.5, r*0.5, paintengine2d.Fill(c.frameEdge))
	in := b.Inset(lw)
	ri := max(r-lw, 0)
	glass := RoundRectPath(in, ri, ri, ri*0.5, ri*0.5)
	grad := c.glassGrad
	if !st.Active {
		grad = c.glassOff
	}
	if c.glass && st.Active {
		// Aero Glass: what lies behind the frame, blurred, through the
		// tinted glass.
		ctx.Save()
		ctx.ClipPath(glass)
		ctx.BackdropBlur(in, l.S(6))
		ctx.Restore()
		tint := make([]paintengine2d.GradientStop, len(grad))
		for i, s := range grad {
			tint[i] = paintengine2d.GradientStop{Offset: s.Offset, Color: s.Color.WithAlpha(s.Color.A * 0.78)}
		}
		grad = tint
	}
	ctx.DrawPath(glass, VGradient(in, grad...))
	if c.glass {
		ctx.Save()
		ctx.ClipPath(glass)
		sh := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), capH*0.55)
		ctx.DrawRect(sh, VGradient(sh, c.sheen...))
		// Two soft diagonal streaks across the glass.
		streak := paintengine2d.NewPath()
		for _, f := range [2]float32{0.18, 0.62} {
			x := in.Min.X + in.Dx()*f
			w := capH * 0.9
			streak.MoveTo(x, in.Min.Y)
			streak.LineTo(x+w, in.Min.Y)
			streak.LineTo(x+w-in.Dy()*0.35, in.Max.Y)
			streak.LineTo(x-in.Dy()*0.35, in.Max.Y)
			streak.Close()
		}
		ctx.DrawPath(streak, paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.1)))
		ctx.Restore()
	}
	winRing(ctx, in, ri, lw, paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.35)))
	client := paintengine2d.XYWH(b.Min.X+fw, b.Min.Y+capH, b.Dx()-2*fw, b.Dy()-capH-fw)
	ctx.DrawRect(client.Inset(-lw), paintengine2d.Fill(c.clientEdge))
	ctx.DrawRect(client, paintengine2d.Fill(c.face))
	right := client.Max.X
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		c.closeButton(l, ctx, cb, st)
		right = cb.Min.X - l.S(6)
	}
	if title == "" {
		return
	}
	f := l.body
	bar := paintengine2d.XYWH(b.Min.X+lw, b.Min.Y+lw, b.Dx()-2*lw, capH-lw)
	tb := paintengine2d.XYWH(client.Min.X+l.S(4), bar.Min.Y, right-client.Min.X-l.S(4), bar.Dy())
	if tb.Dx() < l.S(8) {
		return
	}
	if c.glass {
		// The caption text sits on a blurred white blob so it reads on
		// any glass colour.
		tw := min(f.Advance(title), tb.Dx())
		h := f.Height()
		gr := paintengine2d.XYWH(tb.Min.X-l.S(2), bar.Min.Y+(bar.Dy()-h)*0.5+l.S(2), tw+l.S(4), h-l.S(4))
		ctx.Save()
		ctx.ClipRect(bar)
		DropShadow(ctx, gr, gr.Dy()*0.5, c.capGlow, 0, 0, l.S(12), 0)
		ctx.Restore()
	}
	tc := c.capText
	if !st.Active {
		tc = c.capTextOff
	}
	l.drawFittedText(ctx, f, title, tb, tc, AlignStart, 0)
}

// closeButton paints the caption close button: red glass with the white ×
// (outlined in dark red); a quiet grey-blue glass in an inactive window.
func (c *aeroSet) closeButton(l *Classic, ctx *paintengine2d.Context, cb paintengine2d.Rect, st WindowState) {
	g := &c.close
	switch {
	case !st.Active:
		g = &c.closeOff
	case st.ClosePress:
		g = &c.closePress
	case st.CloseHot:
		g = &c.closeHot
	}
	r := l.rx(3)
	cb = winSnap(cb)
	in := c.paintGlass(l, ctx, cb, g, r, false)
	if in.Empty() {
		return
	}
	lw := winPx(l)
	side := snap(cb.Dy() * 0.42)
	gb := paintengine2d.XYWH(snap((cb.Min.X+cb.Max.X-side)*0.5), snap((cb.Min.Y+cb.Max.Y-side)*0.5), side, side)
	w := max(cb.Dy()*0.13, lw*1.5)
	if !st.Active {
		winCross(ctx, gb, c.closeGlyphOff, w)
		return
	}
	winCross(ctx, gb, c.closeGlyphEdge.WithAlpha(0.75), w+2*lw)
	winCross(ctx, gb, c.closeGlyph, w)
}

// PopupShadow: Vista's soft shadows — small down-right ones under menus
// and tool tips, a large one around windows.
func (aeroEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	sp, ok := aeroShadow(l, kind)
	if !ok {
		return Insets{}
	}
	return ShadowReach(sp.dx, sp.dy, sp.blur, sp.spread)
}

func (aeroEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if sp, ok := aeroShadow(l, kind); ok {
		DropShadow(ctx, b, sp.r, sp.col, sp.dx, sp.dy, sp.blur, sp.spread)
	}
}

func aeroShadow(l *Classic, kind PopupKind) (winShadow, bool) {
	switch kind {
	case PopupDialog:
		return winShadowSpec(l, 0.42, l.rx(6), 0, l.S(3), l.S(14), l.S(1))
	case PopupTooltip:
		return winShadowSpec(l, 0.24, l.rx(3), l.S(1), l.S(1), l.S(4), 0)
	}
	return winShadowSpec(l, 0.3, 0, l.S(2), l.S(2), l.S(5), -l.S(1))
}

// TabOutset: the selected tab is 2px wider on each side and overlaps its
// neighbours.
func (aeroEngine) TabOutset(l *Classic) Insets {
	return Insets{Left: snap(l.S(2)), Right: snap(l.S(2))}
}

// TabOverlap: neighbouring tabs share one border line.
func (aeroEngine) TabOverlap(l *Classic) float32 { return winPx(l) }

// DrawTabPane is the white property page in its tab border.
func (aeroEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aeroColors(l)
	b = winSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.tabBorder))
	ctx.DrawRect(b.Inset(winPx(l)), paintengine2d.Fill(c.pane))
}

// ViewFrameInsets: lists, trees and tables sit in a 1px frame.
func (aeroEngine) ViewFrameInsets(l *Classic) Insets {
	lw := winPx(l)
	return Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}
}

// DrawViewFrame is the list view's white well in its #828790 line.
func (aeroEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := aeroColors(l)
	border, fill := c.viewBorder, c.field
	if st.Disabled() {
		border, fill = c.ed[3][0], c.edDis
	}
	winWell(ctx, winSnap(b), winPx(l), border, fill)
}

// ItemFocus is Explorer's keyboard focus: a rounded blue line around the
// current row (over its selection box when it is selected).
func (aeroEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	r := l.rx(3)
	winRing(ctx, b, r, lw, paintengine2d.Fill(c.focusBorder))
}

// ---- controls ------------------------------------------------------------------------------

func (e aeroEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := aeroColors(l)
	fg := e.Face(l, ctx, b, RoleButton, st)
	l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, 8)
	if st.Focused() && !st.Disabled() {
		lw := winPx(l)
		winDots(ctx, winSnap(b).Inset(3*lw), c.text, lw)
	}
}

func (e aeroEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	c := aeroColors(l)
	winToggleLabel(l, ctx, b, winSnap(box), st, label, c.text, c.disText, c.text)
}

func (e aeroEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	c := aeroColors(l)
	winToggleLabel(l, ctx, b, winSnap(box), st, label, c.text, c.disText, c.text)
}

func (e aeroEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	if !st.Disabled() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	e.Face(l, ctx, b, RoleField, st)
	winDisabledText(l, ctx, b, text, placeholder, scrollX, face, aeroColors(l).gray)
}

// DrawComboBox is a drop-down list: a glass button with the small black
// arrow at its right; keyboard focus puts it in the default blue and rings
// the text with dots.
func (e aeroEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	fst := st
	if open {
		fst |= StatePressed
	}
	fg := c.pushButton(l, ctx, b, fst)
	aw := snap(l.S(17))
	if aw > b.Dx()*0.5 {
		aw = snap(b.Dx() * 0.5)
	}
	ab := paintengine2d.XYWH(b.Max.X-aw-lw, b.Min.Y, aw, b.Dy())
	glyph := c.text
	if st.Disabled() {
		glyph = c.sbGlyphDis
	}
	aeroEngine{}.Arrow(l, ctx, ab, DirDown, glyph)
	tb := paintengine2d.XYWH(b.Min.X+3*lw, b.Min.Y+3*lw, ab.Min.X-b.Min.X-3*lw, b.Dy()-6*lw)
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(tb.Min.X+l.S(3), tb.Min.Y, tb.Dx()-l.S(4), tb.Dy()), fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		winDots(ctx, winSnap(tb), c.text, lw)
	}
}

// DrawSpinner is the up-down control: two small glass buttons.
func (e aeroEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	r := l.rx(2)
	half := func(h paintengine2d.Rect, dir Direction, hover, press bool) {
		g := &c.thumb
		glyph := c.sbGlyph
		switch {
		case st.Disabled():
			g, glyph = &c.btnDis, c.sbGlyphDis
		case press:
			g, glyph = &c.thumbPress, c.sbGlyphHot
		case hover:
			g, glyph = &c.thumbHot, c.sbGlyphHot
		}
		in := c.paintGlass(l, ctx, h, g, r, false)
		if press && !st.Disabled() {
			c.pressShadow(l, ctx, in, max(r-lw, 0))
		}
		gu := min(l.S(1), min(h.Dx(), h.Dy())/8)
		winGlyph(ctx, h, dir, gu, glyph)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress)
}

// DrawTabBar is the strip the tabs stand on: the window face with the
// page's top edge along its bottom.
func (aeroEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.tabBorder))
}

// DrawTab is TABP_TABITEM: glass with rounded top corners (blue when hot);
// the selected tab is white, taller and wider, and opens into the page.
func (aeroEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	u2 := snap(l.S(2))
	t := b
	if !selected {
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+u2, b.Dx(), b.Dy()-u2-lw)
	}
	if t.Dx() < 6*lw || t.Dy() < 6*lw {
		return
	}
	hot := !selected && !st.Disabled() && (st.Hovered() || st.Pressed())
	border := c.tabBorder
	if hot {
		border = c.btnHot.border
	}
	r := l.rx(3)
	ctx.DrawRoundRectCorners(t, r, r, 0, 0, paintengine2d.Fill(border))
	in := paintengine2d.XYWH(t.Min.X+lw, t.Min.Y+lw, t.Dx()-2*lw, t.Dy()-lw)
	ri := max(r-lw, 0)
	inner := RoundRectPath(in, ri, ri, 0, 0)
	switch {
	case selected:
		ctx.DrawPath(inner, paintengine2d.Fill(c.pane))
	case st.Disabled():
		ctx.DrawPath(inner, paintengine2d.Fill(c.btnDis.face[0].Color))
	case hot:
		ctx.DrawPath(inner, VGradient(in, c.btnHot.face...))
	default:
		ctx.DrawPath(inner, VGradient(in, c.btn.face...))
	}
	if !selected && !st.Disabled() && in.Dx() > 3*lw {
		// The white inner highlight along the top and sides.
		ctx.Save()
		ctx.ClipPath(inner)
		hi := paintengine2d.NewPath()
		hi.AddRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw))
		hi.AddRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, lw, in.Dy()-lw))
		hi.AddRect(paintengine2d.XYWH(in.Max.X-lw, in.Min.Y+lw, lw, in.Dy()-lw))
		ctx.DrawPath(hi, paintengine2d.Fill(c.innerHi))
		ctx.Restore()
	}
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
		winDots(ctx, winLabelBox(l, l.body, label, lb, AlignCenter), c.text, lw)
	}
}

func (aeroEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := aeroColors(l)
	if !raised {
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		return
	}
	b = winSnap(b)
	r := l.rx(3)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.groupBorder))
	lw := winPx(l)
	ctx.DrawRoundRect(b.Inset(lw), max(r-lw, 0), max(r-lw, 0), paintengine2d.Fill(c.pane))
}

// DrawMenuBar is Windows 7's menu bar: a pale gradient to a blue-grey.
func (aeroEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, VGradient(b, aeroColors(l).mbar...))
}

// DrawMenuTitle: hot and keyboard-focused titles take a pale glass box, an
// open one the deeper pushed box.
func (e aeroEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := aeroColors(l)
	lw := winPx(l)
	hb := winSnap(b).Inset(lw)
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.gray
	case open:
		c.paintBox(l, ctx, hb, &c.mbOpen)
	case st.Pressed() || st.Hovered() || st.Focused():
		c.paintBox(l, ctx, hb, &c.mbHot)
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
}

// DrawMenuFrame is the Windows 7 popup: a grey border with a white inner
// line, the #f0f0f0 body and the light gradient gutter with its etched
// edge.
func (aeroEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	winWell(ctx, b, lw, c.menuBorder, c.menuInner)
	in := b.Inset(2 * lw)
	ctx.DrawRect(in, paintengine2d.Fill(c.menuBg))
	ch := MenuChromeFor(l)
	gx := snap(b.Min.X + ch.GutterW())
	if gw := gx - in.Min.X; gw > 2*lw && gw < in.Dx()*0.55 {
		g := paintengine2d.XYWH(in.Min.X, in.Min.Y, gw, in.Dy())
		ctx.DrawRect(g, HGradient(g, c.gutter...))
		ctx.DrawRect(paintengine2d.XYWH(gx, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.gutterLine))
		ctx.DrawRect(paintengine2d.XYWH(gx+lw, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.gutterLineLt))
	}
}

// DrawMenuItem is a Windows 7 menu row: the rounded glass hot item across
// the whole row, checks in a small glass box in the gutter, etched
// separators from the label column.
func (e aeroEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := aeroColors(l)
	lw := winPx(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0 := snap(b.Min.X - ch.PadL + ch.GutterW() + 2*lw + l.S(4))
		x1 := snap(b.Max.X + ch.PadR - 3*lw)
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y-lw, x1-x0, lw), paintengine2d.Fill(c.menuSep))
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, lw), paintengine2d.Fill(c.menuSepLt))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL+3*lw, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-6*lw, b.Dy())
		c.paintBox(l, ctx, hb, &c.menuHot)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.menuDisText
	}
	if row.Checked {
		gw := ch.CheckCol()
		side := min(b.Dy()-4*lw, gw-2*lw)
		box := winSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
		if !st.Disabled() {
			c.paintBox(l, ctx, box, &c.hov)
		}
		switch {
		case row.Icon != IconNone && !row.Radio:
			l.drawToolIcon(ctx, box.Inset(2*lw), row.Icon, fg)
		case row.Radio:
			ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
			ctx.DrawCircle(ctr, box.Dx()*0.16, paintengine2d.Fill(fg))
		default:
			winTick(ctx, box.Inset(box.Dx()*0.12), fg, box.Dx()*0.12)
		}
	} else if !row.Radio && row.Icon != IconNone {
		l.drawMenuGutter(ctx, b, ch, row, fg)
	}
	winMenuText(l, ctx, b, ch, row, fg, fg, func(ab paintengine2d.Rect) {
		aeroEngine{}.Arrow(l, ctx, ab, DirRight, fg)
	})
}

// DrawProgressBar is the Windows 7 meter: a rounded grey trough and the
// glossy green fill with its highlight (drawn in place: Windows swept it);
// the indeterminate bar slides a green block with soft ends.
func (aeroEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 6*lw || b.Dy() < 5*lw {
		return
	}
	r := l.rx(3)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.progBorder))
	in := b.Inset(lw)
	ri := max(r-lw, 0)
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.trough...))
	area := in.Inset(lw)
	if area.Empty() {
		return
	}
	ra := max(ri-lw, 0)
	fill := VGradient(area, c.prog...)
	if st.Disabled() {
		fill = fill.WithOpacity(0.4)
	}
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := max(area.Dx()*0.3, l.S(30))
		fade := span * 0.3
		x := area.Min.X + (area.Dx()+span)*phase - span
		ctx.Save()
		ctx.ClipRoundRect(area, ra, ra)
		core := paintengine2d.XYWH(x+fade, area.Min.Y, span-2*fade, area.Dy())
		ctx.DrawRect(core, fill)
		lf := paintengine2d.XYWH(x, area.Min.Y, fade, area.Dy())
		rt := paintengine2d.XYWH(core.Max.X, area.Min.Y, fade, area.Dy())
		ctx.DrawRect(lf, HGradient(lf, c.marqL...))
		ctx.DrawRect(rt, HGradient(rt, c.marqR...))
		ctx.Restore()
		return
	}
	t = clamp1(t)
	w := snap(area.Dx() * t)
	if w < lw {
		return
	}
	// The fill keeps the trough's rounded ends (no clip: the shape is the
	// fill's own path).
	re := float32(0)
	if w >= area.Dx()-ra {
		re = ra
	}
	fr := paintengine2d.XYWH(area.Min.X, area.Min.Y, w, area.Dy())
	ctx.DrawRoundRectCorners(fr, ra, re, re, ra, fill)
	if !st.Disabled() {
		sw := min(fr.Dx()*0.4, l.S(50))
		sx := fr.Min.X + fr.Dx()*0.6 - sw*0.5
		sb := paintengine2d.XYWH(sx, fr.Min.Y, sw, fr.Dy()).Intersect(paintengine2d.XYWH(fr.Min.X+ra, fr.Min.Y, fr.Dx()-ra-re, fr.Dy()))
		if !sb.Empty() {
			ctx.DrawRect(sb, HGradient(paintengine2d.XYWH(sx, fr.Min.Y, sw, fr.Dy()), c.shine...))
		}
		if re == 0 {
			ctx.DrawRect(paintengine2d.XYWH(fr.Max.X-lw, fr.Min.Y, lw, fr.Dy()), paintengine2d.Fill(c.progEdge.WithAlpha(0.6)))
		}
	}
}

// DrawSlider is the trackbar: a thin sunken groove and the pointed glass
// thumb (blue when hot, deeper when dragged); focus is a dotted rectangle
// around the control.
func (aeroEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	t = clamp1(t)
	tw, th := snap(l.S(11)), snap(l.S(19))
	if th > b.Dy()-2*lw {
		th = b.Dy() - 2*lw
	}
	if tw < 5*lw || th < 8*lw || b.Dx() < tw+2*lw {
		return
	}
	ty := snap(b.Min.Y + (b.Dy()-th)*0.5)
	gy := snap(ty + th*0.4 - 2*lw)
	groove := paintengine2d.XYWH(b.Min.X+lw, gy, b.Dx()-2*lw, 4*lw)
	winWell(ctx, groove, lw, c.grooveBorder, c.groove)
	x0 := b.Min.X + tw*0.5
	x1 := b.Max.X - tw*0.5
	tx := snap(x0 + (x1-x0)*t - tw*0.5)
	pt := snap(min(l.S(5), th*0.35))
	g := &c.thumb
	switch aeroChkIndex(st) {
	case 1:
		g = &c.thumbHot
	case 2:
		g = &c.thumbPress
	case 3:
		g = &c.btnDis
	}
	outer := paintengine2d.XYWH(tx, ty, tw, th)
	rr := l.rx(2)
	ctx.DrawPath(winPointer(outer, pt, rr), paintengine2d.Fill(g.border))
	in := outer.Inset(lw)
	inner := winPointer(in, pt-lw*0.4, max(rr-lw, 0))
	ctx.DrawPath(inner, HGradient(in, g.face...))
	if g.hi.A > 0 {
		ctx.Save()
		ctx.ClipPath(inner)
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(g.hi))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, lw, in.Dy()-pt), paintengine2d.Fill(g.hi))
		ctx.Restore()
	}
	if st.Focused() && !st.Disabled() {
		winDots(ctx, b, c.text, lw)
	}
}

// DrawSwitch has no Windows 7 original: a progress trough that fills green
// when on, with a glass knob.
func (e aeroEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := aeroColors(l)
	lw := winPx(l)
	tw, th := l.metrics.SwitchW, min(l.metrics.SwitchH, b.Dy())
	track := winSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 8*lw || track.Dy() < 6*lw {
		return
	}
	r := l.rx(3)
	ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(c.progBorder))
	in := track.Inset(lw)
	ri := max(r-lw, 0)
	if on && !st.Disabled() {
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.knobOn...))
	} else {
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.trough...))
	}
	kw := snap(track.Dy() * 1.1)
	kx := track.Min.X
	if on {
		kx = track.Max.X - kw
	}
	c.pushButton(l, ctx, paintengine2d.XYWH(kx, track.Min.Y, kw, track.Dy()), st&^(StateFocused|StatePrimary))
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	if label == "" {
		if st.Focused() {
			winDots(ctx, track.Inset(-lw).Intersect(b), c.text, lw)
		}
		return
	}
	lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		winDots(ctx, winSnap(labelFocusRect(l.body, label, lb, b)), c.text, lw)
	}
}

// DrawListRow is an Explorer list item: the rounded light-blue box when
// selected (paler under the pointer, grey in an unfocused view).
func (e aeroEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := aeroColors(l)
	lw := winPx(l)
	box := winSnap(b).Inset(lw)
	if bx := c.rowBox(st); bx != nil {
		c.paintBox(l, ctx, box, bx)
	}
	fg := l.fieldText()
	if st.Disabled() {
		fg = c.gray
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(7), b.Min.Y, b.Dx()-l.S(11), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, box, st)
	}
}

// DrawTreeRow is an Explorer navigation-pane row: Vista triangles, no
// connector lines, the selection box across the whole row.
func (e aeroEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := aeroColors(l)
	lw := winPx(l)
	box := winSnap(b).Inset(lw)
	if bx := c.rowBox(st); bx != nil {
		c.paintBox(l, ctx, box, bx)
	}
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		aeroExpander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, st.ExpanderHot())
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	fg := l.fieldText()
	if st.Disabled() {
		fg = c.gray
	}
	lx := x + indent + l.S(3)
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx-l.S(4), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, box, st)
	}
}

// DrawTableHeader is the Windows 7 list header: white over a pale grey
// half, light dividers, a blue glass hot item, a deeper pressed one, and
// the sort arrow centred at the top.
func (aeroEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Empty() {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	hot := st.Hovered() && !st.Disabled() && !pressed
	switch {
	case pressed, hot:
		border, stops := c.hdrHotBorder, c.hdrHotStops
		if pressed {
			border, stops = c.hdrPressBorder, c.hdrPressStops
		}
		ctx.DrawRect(b, paintengine2d.Fill(border))
		in := b.Inset(lw)
		ctx.DrawRect(in, VGradient(in, stops...))
		if pressed {
			c.pressShadow(l, ctx, in, 0)
		} else if in.Dx() > 3*lw && in.Dy() > 3*lw {
			winBorder(ctx, in, lw, c.innerHi)
		}
	default:
		ctx.DrawRect(b, VGradient(b, c.hdrStops...))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y, lw, b.Dy()-lw), paintengine2d.Fill(c.hdrDiv))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.hdrLine))
	}
	if sorted {
		dir := DirDown
		if asc {
			dir = DirUp
		}
		u := min(l.S(1), b.Dx()/12)
		winGlyph(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), snap(5*u)), dir, u, c.hdrArrow)
	}
	fg := c.hdrText
	if st.Disabled() {
		fg = c.gray
	}
	lb := b
	if pressed {
		lb = lb.Translate(paintengine2d.Pt(lw, lw))
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10), lb.Dy()), fg, AlignStart, 0)
}

// DrawTableCell paints one cell of an Explorer details row: the list
// item's rounded selection box runs across the row, each cell painting its
// part of it (the ends round on the first and last cell).
func (aeroEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := aeroColors(l)
	fg := l.fieldText()
	if st.Disabled() {
		fg = c.gray
	}
	if bx := c.rowBox(st); bx != nil {
		r := winSnap(b)
		ctx.Save()
		ctx.ClipRect(r)
		c.paintBox(l, ctx, CellSpan(r, st, l.S(8)).Inset(winPx(l)), bx)
		ctx.Restore()
	}
	winCellText(l, ctx, b, label, align, face, fg)
}

// ToolBarInsets keeps the rebar grip clear of the first button.
func (aeroEngine) ToolBarInsets(l *Classic) Insets { return Insets{Left: l.S(12), Right: l.S(6)} }

// DrawToolBar is a Windows 7 rebar band: a pale gradient, a hairline
// under it and the dotted grip at the left.
func (aeroEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, VGradient(b, c.tb...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.tbBorder))
	if b.Dx() < l.S(12) || b.Dy() < l.S(12) {
		return
	}
	d := 2 * lw
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	gx := snap(b.Min.X + l.S(4))
	for y := snap(b.Min.Y + l.S(6)); y+d <= b.Max.Y-l.S(6); y += 2 * d {
		lt.AddRect(paintengine2d.XYWH(gx+lw, y+lw, d, d))
		dk.AddRect(paintengine2d.XYWH(gx, y, d, d))
	}
	ctx.DrawPath(lt, paintengine2d.Fill(c.tbGripLt))
	ctx.DrawPath(dk, paintengine2d.Fill(c.tbGrip))
}

func (aeroEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := aeroColors(l)
	fg := c.text
	if st.Toggle() || st.Hovered() || st.Pressed() || st.Focused() {
		fg = c.toolFace(l, ctx, winSnap(b).Inset(winPx(l)), st)
	}
	if st.Disabled() {
		fg = c.disText
	}
	winToolContent(l, ctx, b, label, icon, fg)
}

// DrawStatusBar is a pale status bar with a hairline on top, etched pane
// dividers and the dotted size grip.
func (aeroEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, VGradient(b, c.status...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.statTop))
	winStatusParts(l, ctx, b, parts, c.text, c.statSep, c.statSepLt)
	winGripDots(l, ctx, b, c.gripDot, c.gripDotLt)
}

// DrawTitleBar is a dialog heading in the guidelines' main-instruction
// style: dark blue (#003399), larger when there is room.
func (aeroEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := aeroColors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	winHeading(l, ctx, b, title, subtitle, c.instr, c.gray, false)
}

// DrawAccordionHeader is an Explorer group header: the Vista triangle, the
// title in dark blue and a fading line after it; a pale glass box when hot.
func (e aeroEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := aeroColors(l)
	lw := winPx(l)
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		c.paintBox(l, ctx, winSnap(b).Inset(lw), &c.hov)
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, c.expOpen)
	fg := c.groupHead
	if st.Disabled() {
		fg = c.gray
	}
	tb := paintengine2d.XYWH(b.Min.X+l.S(22), b.Min.Y, b.Dx()-l.S(28), b.Dy())
	l.drawFittedText(ctx, l.body, title, tb, fg, AlignStart, 0)
	if lx := tb.Min.X + l.body.Advance(title) + l.S(8); lx < b.Max.X-l.S(12) {
		line := paintengine2d.XYWH(snap(lx), snap((b.Min.Y+b.Max.Y)*0.5), snap(b.Max.X-l.S(8)-lx), lw)
		ctx.DrawRect(line, HGradient(line, c.groupFade...))
	}
	if st.Focused() {
		winDots(ctx, winLabelBox(l, l.body, title, tb, AlignStart), fg, lw)
	}
}

// DrawSeparator is the dialog's etched line (SS_ETCHEDHORZ / VERT).
func (aeroEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := aeroColors(l)
	winEtched(l, ctx, b, vertical, c.etchDk, c.etchLt)
}

func (aeroEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	ctx.DrawRect(b, paintengine2d.Fill(aeroColors(l).face))
}

// TooltipStyle: Windows 7 pads a tip by 4.
func (aeroEngine) TooltipStyle(l *Classic) TooltipStyle {
	return l.tipStyle(l.body, l.tipPad(l.S(4)), AlignStart)
}

// DrawTooltip is the Windows 7 tool tip: rounded, white fading to a pale
// blue-grey, a grey border and grey text.
func (aeroEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := aeroColors(l)
	tb := b
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return
	}
	r := l.rx(3)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.tipBorder))
	in := b.Inset(lw)
	ctx.DrawRoundRect(in, max(r-lw, 0), max(r-lw, 0), VGradient(in, c.tip...))
	pad := l.TooltipStyle().Pad
	// The bubble is sized to the text plus the padding: let the text use
	// the right padding rather than lose its last letters to rounding.
	l.drawTipText(ctx, paintengine2d.XYWH(tb.Min.X+pad, tb.Min.Y, tb.Dx()-pad-winPx(l), tb.Dy()), text, c.tipText)
}

// DrawMessageIcon paints the Windows 7 message icons: glossy red and blue
// spheres with a white ×, "i" or "?", and the amber warning triangle.
func (aeroEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	c := aeroColors(l)
	switch icon {
	case IconNone:
	case IconError:
		winOrb(l, ctx, b, c.errDisc, c.errRim, c.gloss, "×", Hex("#ffffff"))
	case IconInfo:
		winOrb(l, ctx, b, c.infoDisc, c.infoRim, c.gloss, "i", Hex("#ffffff"))
	case IconQuestion:
		winOrb(l, ctx, b, c.infoDisc, c.infoRim, c.gloss, "?", Hex("#ffffff"))
	case IconWarning:
		winWarnTriangle(l, ctx, b, c.warnFace, c.warnRim, Hex("#000000"))
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
	}
}

// ---- packs --------------------------------------------------------------------------------

func aeroPack(name, label string, year int, summary string, glass float32) ThemePack {
	pal := Palette{
		Background: Hex("#f0f0f0"), Surface: Hex("#f0f0f0"), SurfaceAlt: Hex("#f0f0f0"),
		Border: Hex("#898c95"), Divider: Hex("#d5dfe5"),
		Text: Hex("#000000"), TextMuted: Hex("#6d6d6d"), TextOnAccent: Hex("#ffffff"),
		Accent: Hex("#3399ff"), AccentHover: Hex("#5cadff"), AccentPress: Hex("#1f7fe0"),
		Field: Hex("#ffffff"), FieldBorder: Hex("#abadb3"),
		Focus: Hex("#000000"), Selection: Hex("#3399ff"),
		Track: Hex("#eeeeee"), Thumb: Hex("#dbdbdb"),
		Highlight: Hex("#e5f1fb"), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: Hex("#dcebfc"), MenuHoverBorder: Hex("#aecff7"), MenuGutter: Hex("#f1f1f1"),
		Danger: Hex("#c42b1c"), Success: Hex("#1f8a1f"), Warning: Hex("#9d6200"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.35),
		BevelLight: Hex("#ffffff"), BevelDark: Hex("#a0a0a0"),
	}
	tok := ThemeTokens{
		Engine:  "aero",
		Bevel:   BevelLunaHottrack,
		Family:  ThemeLight,
		Palette: pal,
		Params:  map[string]float32{"glass": glass},
		// Selected text is white on the #3399ff highlight (HighlightText).
		Extra: map[string]paintengine2d.Color{"selectionText": Hex("#ffffff")},
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: Hex("#c1dbfc"), Border: Hex("#7da2ce")}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.12), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Windows", Summary: summary,
		Era: "Aero", Palette: ThemeLight, Tokens: tok,
	}
}

func aeroPacks() []ThemePack {
	return []ThemePack{
		aeroPack("aero", "Aero", 2007, "Windows Vista and 7: glass captions, two-tone glassy buttons with a blue glow, Explorer's light-blue selection boxes.", 1),
		aeroPack("aero-basic", "Windows 7 Basic", 2009, "Windows 7 without Aero Glass: the same controls under opaque light-blue frames.", 0),
	}
}

// aeroWindowColour is Windows 7's default window colour, Sky (#74b8fc),
// which tints the glass to the pack's frame colours.
var aeroWindowColour = Hex("#74b8fc")

// aeroGlassKeys are the glass frame's gradient stops, active and inactive.
var aeroGlassKeys = [...]string{"glass0", "glass1", "glass2", "glassOff0", "glassOff1", "glassOff2"}

// Accented is the window colour of Windows Vista and 7: under Aero Glass it
// tinted the glass of the window frames and captions, and the glass takes
// the accent as its window colour — each stop makes the step from the
// accent that the default Sky glass makes from Sky (accentShift). The
// controls never followed the window colour and keep their blues, and
// Windows 7 Basic's opaque frames keep theirs.
func (aeroEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	if accentP(tok, "glass", 1) == 0 {
		return tok
	}
	own := func(k string) paintengine2d.Color { return accentX(tok, k, Hex(aeroBase[k])) }
	// The window colour this pack's glass is the tint of: Sky for the
	// shipped pack.
	ref := accentShift(own("glass1"), Hex(aeroBase["glass1"]), aeroWindowColour)
	tok = CloneTokenMaps(tok)
	for _, k := range aeroGlassKeys {
		tok.Extra[k] = accentShift(accent, ref, own(k))
	}
	return tok
}

// ---- helpers shared by the Windows engines (aero, metro, fluent) ----------------------

// winPx is one device line at the look's scale: 1 at 1x, 2 at 2x.
func winPx(l *Classic) float32 {
	v := float32(int(l.S(1) + 0.5))
	if v < 1 {
		v = 1
	}
	return v
}

// winSnap puts b on the pixel grid. Edges round half down, so a snapped
// rect never covers a pixel whose centre lies outside b, and rects that
// tile (table cells, rows) still meet without a gap.
func winSnap(b paintengine2d.Rect) paintengine2d.Rect {
	f := func(v float32) float32 { return float32(math.Ceil(float64(v) - 0.5 - 1e-3)) }
	x0, y0, x1, y1 := f(b.Min.X), f(b.Min.Y), f(b.Max.X), f(b.Max.Y)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return paintengine2d.Rect{Min: paintengine2d.Pt(x0, y0), Max: paintengine2d.Pt(x1, y1)}
}

// winRing fills the lw-wide band just inside the round rect b (radius r)
// with paint, leaving the inside untouched: one even-odd path.
func winRing(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, paint paintengine2d.Paint) {
	if b.Empty() {
		return
	}
	if b.Dx() <= 2*lw || b.Dy() <= 2*lw {
		ctx.DrawRoundRect(b, r, r, paint)
		return
	}
	p := paintengine2d.NewPath()
	p.AddRoundRect(b, r, r)
	ri := max(r-lw, 0)
	p.AddRoundRect(b.Inset(lw), ri, ri)
	paint.Style = paintengine2d.StyleFill
	paint.FillRule = paintengine2d.FillEvenOdd
	ctx.DrawPath(p, paint)
}

// winAddRoundRect appends a closed rect with per-corner radii (tl, tr, br,
// bl) to p, so several shapes can share one even-odd path.
func winAddRoundRect(p *paintengine2d.Path, b paintengine2d.Rect, tl, tr, br, bl float32) {
	if b.Empty() {
		return
	}
	lim := min(b.Dx(), b.Dy()) * 0.5
	tl, tr, br, bl = min(max(tl, 0), lim), min(max(tr, 0), lim), min(max(br, 0), lim), min(max(bl, 0), lim)
	const k = 0.5522847 // cubic circle constant
	p.MoveTo(b.Min.X+tl, b.Min.Y)
	p.LineTo(b.Max.X-tr, b.Min.Y)
	if tr > 0 {
		p.CubicTo(b.Max.X-tr+tr*k, b.Min.Y, b.Max.X, b.Min.Y+tr-tr*k, b.Max.X, b.Min.Y+tr)
	}
	p.LineTo(b.Max.X, b.Max.Y-br)
	if br > 0 {
		p.CubicTo(b.Max.X, b.Max.Y-br+br*k, b.Max.X-br+br*k, b.Max.Y, b.Max.X-br, b.Max.Y)
	}
	p.LineTo(b.Min.X+bl, b.Max.Y)
	if bl > 0 {
		p.CubicTo(b.Min.X+bl-bl*k, b.Max.Y, b.Min.X, b.Max.Y-bl+bl*k, b.Min.X, b.Max.Y-bl)
	}
	p.LineTo(b.Min.X, b.Min.Y+tl)
	if tl > 0 {
		p.CubicTo(b.Min.X, b.Min.Y+tl-tl*k, b.Min.X+tl-tl*k, b.Min.Y, b.Min.X+tl, b.Min.Y)
	}
	p.Close()
}

// winBorder fills a square lw-wide border just inside b (one path).
func winBorder(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, col paintengine2d.Color) {
	if b.Dx() < 2*lw || b.Dy() < 2*lw || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw))
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw))
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, lw, b.Dy()-2*lw))
	p.AddRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+lw, lw, b.Dy()-2*lw))
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// winWell fills b with fill inside an lw-wide square border.
func winWell(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, border, fill paintengine2d.Color) {
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	winBorder(ctx, b, lw, border)
}

// winDots is the Windows dotted focus rectangle just inside b: dots of one
// device line on every other one, all in one path.
func winDots(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, lw float32) {
	b = winSnap(b)
	if b.Dx() < 2*lw || b.Dy() < 2*lw || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	step := 2 * lw
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

// winGlyph fills the solid Windows arrow glyph — a stepped triangle 8
// units across and 4 deep, one device line per step — pointing dir,
// centred in b, u pixels per unit.
func winGlyph(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, u float32, col paintengine2d.Color) {
	if col.A <= 0 || b.Empty() || u <= 0 {
		return
	}
	n := int(4*u + 0.5) // steps, one device line each
	if n < 2 {
		n = 2
	}
	w := float32(2 * n) // across
	h := float32(n)     // deep
	vertical := dir == DirUp || dir == DirDown
	var x0, y0 float32
	if vertical {
		x0, y0 = snap((b.Min.X+b.Max.X-w)*0.5), snap((b.Min.Y+b.Max.Y-h)*0.5)
	} else {
		x0, y0 = snap((b.Min.X+b.Max.X-h)*0.5), snap((b.Min.Y+b.Max.Y-w)*0.5)
	}
	p := paintengine2d.NewPath()
	for k := 0; k < n; k++ {
		f := float32(k)
		switch dir {
		case DirDown:
			p.AddRect(paintengine2d.XYWH(x0+f, y0+f, w-2*f, 1))
		case DirUp:
			p.AddRect(paintengine2d.XYWH(x0+f, y0+h-1-f, w-2*f, 1))
		case DirRight:
			p.AddRect(paintengine2d.XYWH(x0+f, y0+f, 1, w-2*f))
		default:
			p.AddRect(paintengine2d.XYWH(x0+h-1-f, y0+f, 1, w-2*f))
		}
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// winTick strokes a check mark filling the box b (round caps: square caps
// on an open polyline draw a stray band in the engine).
func winTick(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, width float32) {
	if b.Empty() || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	p.MoveTo(b.Min.X+b.Dx()*0.24, b.Min.Y+b.Dy()*0.52)
	p.LineTo(b.Min.X+b.Dx()*0.42, b.Min.Y+b.Dy()*0.70)
	p.LineTo(b.Min.X+b.Dx()*0.77, b.Min.Y+b.Dy()*0.30)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: width, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// winCross fills an × from corner to corner of b, two bars w thick with
// flat ends.
func winCross(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, w float32) {
	if b.Empty() || w <= 0 || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	for _, d := range [2][4]float32{{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}, {b.Max.X, b.Min.Y, b.Min.X, b.Max.Y}} {
		x0, y0, x1, y1 := d[0], d[1], d[2], d[3]
		dx, dy := x1-x0, y1-y0
		n := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		nx, ny := -dy/n*w*0.5, dx/n*w*0.5
		p.MoveTo(x0+nx, y0+ny)
		p.LineTo(x1+nx, y1+ny)
		p.LineTo(x1-nx, y1-ny)
		p.LineTo(x0-nx, y0-ny)
		p.Close()
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// winVistaTriangle is the Vista / 7 tree glyph centred in b: a hollow
// triangle pointing right (closed), a filled one pointing down-right (open).
func winVistaTriangle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, open, openEdge, closed, closedFill paintengine2d.Color) {
	u := l.S(1)
	cx, cy := snap((b.Min.X+b.Max.X)*0.5), snap((b.Min.Y+b.Max.Y)*0.5)
	if expanded {
		s := snap(6 * u)
		x0, y0 := cx-snap(s*0.5), cy-snap(s*0.5)
		p := paintengine2d.NewPath()
		p.MoveTo(x0+s, y0)
		p.LineTo(x0+s, y0+s)
		p.LineTo(x0, y0+s)
		p.Close()
		ctx.DrawPath(p, paintengine2d.Fill(open))
		ctx.DrawPath(p, paintengine2d.Paint{Color: openEdge.WithAlpha(0.6), Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: u * 0.8, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
		return
	}
	h := snap(8 * u)
	w := h * 0.5
	x0, y0 := cx-snap(w*0.5), cy-h*0.5
	p := paintengine2d.NewPath()
	p.MoveTo(x0+u*0.5, y0+u*0.5)
	p.LineTo(x0+w, y0+h*0.5)
	p.LineTo(x0+u*0.5, y0+h-u*0.5)
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(closedFill))
	ctx.DrawPath(p, paintengine2d.Paint{Color: closed, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: u, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// winPointer is a trackbar thumb pointing down: a block with rounded top
// corners (r) ending in a point p deep.
func winPointer(b paintengine2d.Rect, pt, r float32) *paintengine2d.Path {
	r = min(r, b.Dx()*0.3)
	p := paintengine2d.NewPath()
	p.MoveTo(b.Min.X+r, b.Min.Y)
	p.LineTo(b.Max.X-r, b.Min.Y)
	if r > 0 {
		p.QuadTo(b.Max.X, b.Min.Y, b.Max.X, b.Min.Y+r)
	}
	p.LineTo(b.Max.X, b.Max.Y-pt)
	p.LineTo((b.Min.X+b.Max.X)*0.5, b.Max.Y)
	p.LineTo(b.Min.X, b.Max.Y-pt)
	p.LineTo(b.Min.X, b.Min.Y+r)
	if r > 0 {
		p.QuadTo(b.Min.X, b.Min.Y, b.Min.X+r, b.Min.Y)
	}
	p.Close()
	return p
}

// winPagedPart is the part of the track a held page press darkens.
func winPagedPart(p ScrollParts, vertical bool, st ScrollState) paintengine2d.Rect {
	if (st.Pressed != ScrollPageDec && st.Pressed != ScrollPageInc) || p.Thumb.Empty() {
		return paintengine2d.Rect{}
	}
	pg := p.Track
	if vertical {
		if st.Pressed == ScrollPageDec {
			pg.Max.Y = p.Thumb.Min.Y
		} else {
			pg.Min.Y = p.Thumb.Max.Y
		}
		pg.Min.X, pg.Max.X = p.Bar.Min.X, p.Bar.Max.X
	} else {
		if st.Pressed == ScrollPageDec {
			pg.Max.X = p.Thumb.Min.X
		} else {
			pg.Min.X = p.Thumb.Max.X
		}
		pg.Min.Y, pg.Max.Y = p.Bar.Min.Y, p.Bar.Max.Y
	}
	return pg
}

// winScrollState turns a thumb's ControlState into a ScrollState for the
// DrawScrollBar fallback.
func winScrollState(st ControlState) ScrollState {
	ss := ScrollState{Disabled: st.Disabled(), Hovered: st.Hovered()}
	switch {
	case st.Pressed():
		ss.Pressed, ss.Hot = ScrollThumbPart, ScrollThumbPart
	case st.Hovered():
		ss.Hot = ScrollThumbPart
	}
	return ss
}

// winShadow is a DropShadow recipe.
type winShadow struct {
	col                     paintengine2d.Color
	r, dx, dy, blur, spread float32
}

// winShadowSpec scales a shadow's alpha by the pack's "shadow" param (0
// turns shadows off).
func winShadowSpec(l *Classic, alpha, r, dx, dy, blur, spread float32) (winShadow, bool) {
	k := l.P("shadow", 1)
	if k <= 0 {
		return winShadow{}, false
	}
	return winShadow{col: paintengine2d.RGBA(0, 0, 0, min(alpha*k, 1)), r: r, dx: dx, dy: dy, blur: blur, spread: spread}, true
}

// winToggleLabel draws a check box / radio caption right of box, with the
// dotted focus rectangle around it (or around the box without a label).
func winToggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string, fg, dis, focus paintengine2d.Color) {
	lw := winPx(l)
	if label == "" {
		if st.Focused() {
			winDots(ctx, box.Inset(-lw).Intersect(b), focus, lw)
		}
		return
	}
	if st.Disabled() {
		fg = dis
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		winDots(ctx, labelFocusRect(l.body, label, lb, b), focus, lw)
	}
}

// winLabelBox is the focus rectangle around a label drawn in lb.
func winLabelBox(l *Classic, f *Font, label string, lb paintengine2d.Rect, align Align) paintengine2d.Rect {
	w := min(f.Advance(label)+l.S(6), lb.Dx()-l.S(2))
	h := min(f.Height()+l.S(2), lb.Dy())
	x := lb.Min.X - l.S(3)
	if align == AlignCenter {
		x = lb.Min.X + (lb.Dx()-w)*0.5
	}
	return paintengine2d.XYWH(x, lb.Min.Y+(lb.Dy()-h)*0.5, w, h).Intersect(lb)
}

// winDisabledText draws a disabled field's text (or placeholder) in col.
func winDisabledText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text, placeholder string, scrollX float32, face *Font, col paintengine2d.Color) {
	pad := l.metrics.FieldPad
	if pad <= 0 {
		pad = 8
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	if inner.Empty() {
		return
	}
	show := text
	if show == "" {
		show = placeholder
	}
	f := l.faceOrBody(face)
	ctx.Save()
	ctx.ClipRect(inner)
	f.Draw(ctx, show, paintengine2d.Pt(inner.Min.X-scrollX, inner.Min.Y+(inner.Dy()-f.Height())*0.5), col)
	ctx.Restore()
}

// winMenuText lays out a menu row's label, shortcut and submenu arrow.
func winMenuText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, ch MenuChrome, row MenuRow, fg, shortcut paintengine2d.Color, arrow func(ab paintengine2d.Rect)) {
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	labelRight := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = 10
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		arrow(ab)
		labelRight = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := f.InkWidth(row.Shortcut)
		if tw <= 0 {
			tw = f.Advance(row.Shortcut)
		}
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), shortcut)
		labelRight = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()))
	l.drawTextUnderline(ctx, f, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// winCellText draws a table cell's label with the stock padding rules.
func winCellText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, label string, align Align, face *Font, fg paintengine2d.Color) {
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
	avail := b.Dx() - pad*2
	if avail < 4 {
		avail = 4
	}
	label = f.Fit(label, avail)
	tw := f.Advance(label)
	x := b.Min.X + pad
	switch align {
	case AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case AlignEnd:
		x = b.Max.X - tw - pad
	}
	ctx.Save()
	ctx.ClipRect(b.Inset(1))
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// winToolContent draws a tool button's icon and label in fg.
func winToolContent(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, label string, icon ToolIcon, fg paintengine2d.Color) {
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		right := max(b.Max.X-pad, x)
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
		l.body.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-l.body.Height())*0.5), fg)
		ctx.Restore()
	}
}

// winStatusParts lays status bar parts out in equal slots (leaving room
// for the size grip), divided by an etched pair of lines.
func winStatusParts(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string, fg, sepDk, sepLt paintengine2d.Color) {
	if len(parts) == 0 {
		return
	}
	lw := winPx(l)
	slot := (b.Dx() - l.S(16)) / float32(len(parts))
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			sx := snap(x)
			y0 := b.Min.Y + snap(l.S(4))
			h := b.Max.Y - y0 - snap(l.S(4))
			dk.AddRect(paintengine2d.XYWH(sx-lw, y0, lw, h))
			lt.AddRect(paintengine2d.XYWH(sx, y0, lw, h))
		}
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(6), b.Min.Y+lw, slot-l.S(10), b.Dy()-lw), fg, AlignStart, 0)
	}
	if sepLt.A > 0 {
		ctx.DrawPath(lt, paintengine2d.Fill(sepLt))
	}
	ctx.DrawPath(dk, paintengine2d.Fill(sepDk))
}

// winGripDots paints the size grip: a triangle of dots, each with a light
// shadow, at the bottom right of b.
func winGripDots(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dot, lite paintengine2d.Color) {
	lw := winPx(l)
	d := 2 * lw
	step := snap(l.S(4))
	if b.Dy() < 3*step+d || b.Dx() < 3*step+d {
		return
	}
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for row := 0; row < 3; row++ {
		for k := 0; k <= row; k++ {
			x := b.Max.X - float32(k+1)*step
			y := b.Max.Y - float32(3-row)*step
			lt.AddRect(paintengine2d.XYWH(x+lw, y+lw, d, d))
			dk.AddRect(paintengine2d.XYWH(x, y, d, d))
		}
	}
	if lite.A > 0 {
		ctx.DrawPath(lt, paintengine2d.Fill(lite))
	}
	ctx.DrawPath(dk, paintengine2d.Fill(dot))
}

// winHeading draws a title (in the title font when it fits and big) and a
// muted subtitle after it.
func winHeading(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string, col, muted paintengine2d.Color, bold bool) {
	x := b.Min.X + l.metrics.Pad
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	} else if subtitle == "" && l.title != nil && b.Dy() >= l.title.Height()+l.S(6) {
		f = l.title
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), col, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), muted, AlignStart, 0)
	}
}

// winEtched is a two-line groove (dark over light) across the middle of b.
func winEtched(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, dk, lt paintengine2d.Color) {
	lw := winPx(l)
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - lw
		y0, h := snap(b.Min.Y+l.S(2)), snap(b.Dy()-l.S(4))
		ctx.DrawRect(paintengine2d.XYWH(x, y0, lw, h), paintengine2d.Fill(dk))
		if lt.A > 0 {
			ctx.DrawRect(paintengine2d.XYWH(x+lw, y0, lw, h), paintengine2d.Fill(lt))
		}
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - lw
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), lw), paintengine2d.Fill(dk))
	if lt.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+lw, b.Dx(), lw), paintengine2d.Fill(lt))
	}
}

// winGroupFrame is a group box frame of radius r in a line (with an inner
// light line when inner is set), left open behind the title; raised fills
// it with card.
func winGroupFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool, r float32, line, inner, card, text paintengine2d.Color) {
	b = winSnap(b)
	lw := winPx(l)
	f := l.body
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	frame := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if frame.Dx() < 4*lw || frame.Dy() < 4*lw {
		return
	}
	if raised {
		ctx.DrawRoundRect(frame, r, r, paintengine2d.Fill(card))
	}
	var gap paintengine2d.Rect
	if title != "" {
		tw := min(f.Advance(title)+l.S(6), b.Dx()-l.S(16))
		gap = paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, max(tw, 0), f.Height())
	}
	if !gap.Empty() {
		clip := paintengine2d.NewPath()
		clip.AddRect(b)
		clip.AddRect(gap)
		ctx.Save()
		ctx.ClipPathRule(clip, paintengine2d.FillEvenOdd)
	}
	if inner.A > 0 {
		winRing(ctx, frame.Inset(lw), max(r-lw, 0), lw, paintengine2d.Fill(inner))
	}
	winRing(ctx, frame, r, lw, paintengine2d.Fill(line))
	if !gap.Empty() {
		ctx.Restore()
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(gap.Min.X+l.S(3), gap.Min.Y, gap.Dx()-l.S(3), gap.Dy()), text, AlignStart, 0)
	}
}

// winOrb paints a glossy message-box sphere with a white glyph.
func winOrb(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, disc []paintengine2d.GradientStop, rim paintengine2d.Color, gloss []paintengine2d.GradientStop, glyph string, gc paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	if s < 4 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	r := s * 0.47
	lw := winPx(l)
	ctr := paintengine2d.Pt(cx, cy)
	ctx.DrawCircle(ctr, r, paintengine2d.Fill(rim))
	ctx.DrawCircle(ctr, r-lw, paintengine2d.Radial(paintengine2d.RadialGradient{
		Center: paintengine2d.Pt(cx-r*0.3, cy-r*0.4), Radius: r * 1.45, Stops: disc,
	}))
	if gloss != nil {
		hb := paintengine2d.XYWH(cx-r*0.62, cy-r*0.86, r*1.24, r*0.8)
		ctx.DrawOval(hb, VGradient(hb, gloss...))
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	if glyph == "×" {
		k := r * 0.42
		winCross(ctx, paintengine2d.XYWH(cx-k, cy-k, 2*k, 2*k), gc, r*0.24)
		return
	}
	w := f.Advance(glyph)
	f.Draw(ctx, glyph, paintengine2d.Pt(cx-w*0.5, cy-f.Height()*0.5), gc)
}

// winWarnTriangle paints the warning triangle with a "!".
func winWarnTriangle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, face []paintengine2d.GradientStop, rim, gc paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	if s < 4 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	r := s * 0.47
	lw := winPx(l)
	tri := func(k float32) *paintengine2d.Path {
		p := paintengine2d.NewPath()
		p.MoveTo(cx, cy-r+k*1.8)
		p.LineTo(cx+r-k*1.6, cy+r*0.82-k)
		p.LineTo(cx-r+k*1.6, cy+r*0.82-k)
		p.Close()
		return p
	}
	ctx.DrawPath(tri(0), paintengine2d.Paint{Color: rim, Style: paintengine2d.StyleStrokeAndFill,
		Stroke: paintengine2d.Stroke{Width: lw * 1.5, Join: paintengine2d.JoinRound, Cap: paintengine2d.CapRound, MiterLimit: 4}})
	ctx.DrawPath(tri(lw*1.3), VGradient(paintengine2d.XYWH(cx-r, cy-r, 2*r, 1.8*r), face...))
	f := l.bold
	if f == nil {
		f = l.body
	}
	w := f.Advance("!")
	f.Draw(ctx, "!", paintengine2d.Pt(cx-w*0.5, cy-f.Height()*0.5+r*0.2), gc)
}
