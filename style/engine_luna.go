package style

import (
	"math"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/codemodify/paintengine2d"
)

// lunaEngine paints Windows XP "Luna" (2001), the visual style uxtheme drew
// from luna.msstyles: push buttons are 3px-rounded white-to-beige gradients
// in a dark 1px border with an orange inner glow when hot and a blue glow
// on the default button; check boxes and radios are 13px wells with a green
// tick / dot; scroll bars are 17px gutters with rounded arrow buttons and a
// gripped thumb; tabs wear an orange top edge when hot or selected;
// progress bars fill with 8px chunks; list headers light an orange bottom
// edge; in-app windows carry the rounded gradient caption with its red
// close button.
//
// Menus and tool bars speak the hot-track language the era's applications
// shipped: a pale fill with a 1px border across icon gutter and label, a
// shaded gutter, command bars with a dotted grip. The flavour is per pack:
// Luna Blue, Olive Green and Silver use Office 2003 (#ffeec2 hot items in
// the scheme's border colour, a gradient gutter and bands, orange pressed
// and checked buttons); Royale and Royale Noir use Office XP (a pale tint
// of the highlight in a highlight border, flat gutter and bars) — Office
// 2003's own Royale table is exactly that look.
//
// Colours are the shipped values: luna.msstyles (the NormalColor, HomeStead
// and Metallic INIs and bitmaps) for controls and captions, the
// Royale / Royale Noir artwork for those schemes, and the Office 2003
// colour tables (WinForms' ProfessionalColorTable carries them) for menus
// and command bars. Royale Noir keeps its black glossy chrome but, unlike
// the leaked original, paints a dark client area: it is the dark pack.
//
// Pack data:
//
//	params  "scheme"    0 Blue (NormalColor), 1 Olive Green (HomeStead),
//	                    2 Silver (Metallic), 3 Royale, 4 Royale Noir.
//	                    Every colour defaults to that scheme's value.
//	        "vertical"  1 = scroll / combo buttons shade top-to-bottom
//	                    (Silver, Royale) instead of diagonally.
//	extra   any colour key of the scheme tables (see lunaBlue) overrides
//	        that one colour, e.g. "btnBorder", "hotFill", "menuBorder",
//	        "groupText"; "caption" + "caption2" replace the caption
//	        gradient. Other gradients ("g." keys) stay the scheme's.
//
// The shared palette carries what widgets paint themselves: Selection is
// the scheme highlight, MenuHover / MenuHoverBorder / MenuGutter are the
// menu hot-track and gutter the engine paints, Divider is the command-bar
// separator.
type lunaEngine struct{ BaseEngine }

func init() {
	RegisterEngine(lunaEngine{})
	for _, p := range lunaPacks() {
		RegisterPack(p)
	}
}

func (lunaEngine) ID() string { return "luna" }

// DefaultMetrics are XP's proportions at the toolkit's 16px UI font (XP
// used 11px Tahoma): 17px scroll bars, 13px check boxes and radios, 3px
// corners.
func (lunaEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Radius:    3, RadiusSmall: 3,
		ControlH: 30, FieldH: 28, ComboH: 28,
		Checkbox: 13, Radio: 13,
		MenuItemH: 26, MenuBarH: 26, TabH: 28, RowH: 22,
		TitleBar: 30, HeaderH: 26, ProgressH: 18, SliderH: 26, Thumb: 11,
		Scroll: 17, Pad: 10, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 26, SpinnerW: 17,
	}
}

// ---- scheme tables ------------------------------------------------------------

// lunaScheme maps a key to a colour ("#rrggbb", "#rrggbbaa"), a gradient
// ("g." keys: "offset #colour, …") or a flag ("f." keys). lunaBlue holds
// every key; the other schemes list what differs.
type lunaScheme map[string]string

// lunaBlue is Luna Blue (NormalColor): luna.msstyles Blue\*.bmp and
// [SysMetrics], with Office 2003's Blue command-bar table.
var lunaBlue = lunaScheme{
	// System colours.
	"shadow": "#aca899", "hi": "#ffffff",
	"gray": "#aca899", "disText": "#a1a192",
	"info": "#ffffe1", "infoText": "#000000", "infoBorder": "#000000",

	// Push button (button.bmp: normal, hot, pressed, disabled, defaulted).
	"btnBorder": "#003c74", "btnShade": "#cfc8b8", "btnSides": "#00000000",
	"btnPressSide": "#d8d4cb", "btnDisBorder": "#c9c7ba", "btnDisFace": "#f5f4ea",
	"g.btnFace":  "0 #ffffff, .28 #f6f6f3, .83 #f0f0ea, .89 #ecebe6, .94 #e2dfd6, 1 #d6d0c5",
	"g.btnPress": "0 #d1ccc1, .06 #dcd8cf, .12 #e4e3dc, .5 #e2e1d9, .88 #e2e2da, .94 #eae9e3, 1 #f2f1ee",
	"hotTop":     "#fff0cf", "hot2": "#fdd889", "hotOutA": "#fedf9a", "hotOutB": "#f9b435",
	"hotInA": "#fcd279", "hotInB": "#f8b535", "hotBot1": "#f8b330", "hotBot2": "#e59700",
	"defTop": "#cee7ff", "def2": "#bcd4f6", "defOutA": "#bad3f5", "defOutB": "#89ade4",
	"defInA": "#bcd4f6", "defInB": "#89ade4", "defBot1": "#89ade4", "defBot2": "#6982ee",

	// Check box and radio (CheckBox13.bmp, RadioButton13.bmp).
	"chkBorder": "#1c5180", "chkWell0": "#dcdcd7", "chkWell1": "#ffffff",
	"chkHot0": "#fff0cf", "chkHot1": "#f8b330", "chkHotWell": "#e7e7e3",
	"chkPress0": "#b0b0a7", "chkPress1": "#f1efdf",
	"chkDisBorder": "#cac8bb", "chkDisWell": "#ffffff",
	"tick": "#21a121", "tickDis": "#cac8bb",
	"g.dot": "0 #8be588, .45 #38b935, 1 #139210",

	// Edit and combo box ([edit], [Combobox]).
	"fieldBorder": "#7f9db9", "fieldDisBorder": "#c9c7ba", "fieldDis": "#ebebe4",
	"comboBorder": "#7f9db9", "comboDis": "#f5f4ea",

	// Scroll arrows, combo and spin buttons (ScrollArrows.bmp, ComboButton.bmp):
	// normal, hot, pressed, disabled.
	"sbRing": "#ffffff", "sbEdge": "#b7caf5", "g.sbFace": "0 #dae6fe, 1 #aec8f7", "sbGlyph": "#4d6185",
	"sbHotRing": "#ffffff", "sbHotEdge": "#97aee0", "g.sbHotFace": "0 #f6ffff, 1 #b9dafb", "sbHotGlyph": "#4d6185",
	"sbPressRing": "#ffffff", "sbPressEdge": "#8592d9", "g.sbPressFace": "0 #6e94f1, 1 #d2deeb", "sbPressGlyph": "#4d6185",
	"sbDisRing": "#ffffff", "sbDisEdge": "#e8e8df", "g.sbDisFace": "0 #f1f1ec, 1 #eaeae2", "sbDisGlyph": "#c9c9c2",
	"sbShadowR": "#a0b5d3", "sbShadowB": "#7c9fd3",
	"f.vertical": "0",
	// Scroll thumb (ScrollThumbVertical.bmp) and its gripper.
	"thRing": "#ffffff", "thEdge": "#b4c8f6", "g.thFace": "0 #c8d6fb, .5 #c5d5fc, .8 #b3cffc, 1 #b5cdfa",
	"thHotRing": "#ffffff", "thHotEdge": "#acceff", "g.thHotFace": "0 #d8e8ff, .6 #d6e7ff, 1 #cae0ff",
	"thPressRing": "#ffffff", "thPressEdge": "#839ad3", "g.thPressFace": "0 #a8bdf5, .5 #a8c0fa, 1 #91b3f7",
	"gripLt": "#eef4fe", "gripDk": "#8cb0f8", "gripHotLt": "#fcfdff", "gripHotDk": "#9cc5ff",
	"gripPressLt": "#cfddfd", "gripPressDk": "#839ed8",
	// Scroll shaft (ScrollShaftVertical.bmp), normal and pressed.
	"trackEdge": "#eeede5", "track0": "#f3f1ec", "track1": "#fefefb",
	"trackPressEdge": "#d7d5c2", "trackPress0": "#e3ded3", "trackPress1": "#fdfdf6",

	// Tabs (tabItem.bmp, TabPaneEdge.bmp, TabBackground.bmp).
	"tabBorder": "#91a7b4", "tabHotBorder": "#9db1ba", "tabShade": "#d6d0c0",
	"g.tabFace": "0 #fefefe, .12 #fdfdfc, .5 #f5f5f1, .88 #f0f0ea, 1 #ecebe6",
	"tabHot1":   "#e68b2c", "tabHot2": "#ffc83c", "tabHot3": "#ffc73c",
	"pane": "#fcfcfe", "pane2": "#f4f3ee", "paneBorder": "#919b9c",
	"paneShadow1": "#d0cebf", "paneShadow2": "#e3e0d0",

	// Progress (ProgressTrack.bmp, ProgressChunk.bmp).
	"progBorder": "#686868", "progIn": "#bebebe", "progIn2": "#efefef", "progTrough": "#ffffff",
	"g.chunk": "0 #abedac, .09 #94e995, .18 #7be37d, .27 #66df68, .36 #4eda50, .45 #35d538, .55 #28d22b, .64 #2ed331, .73 #41d744, .82 #5bdd5d, .91 #75e277, 1 #8ee790",

	// Trackbar (TrackbarDown13.bmp, sliderTrack.bmp): thumb frame, face and
	// the accent (top band, point edges) for normal, hot, pressed, disabled.
	"thumbBorder": "#b5c4cd", "thumbBorderDk": "#778892", "thumbBorderLo": "#91a6b2",
	"thumbFace": "#f3f3ef", "thumbFaceHi": "#f7f7f4", "thumbFaceLo": "#c3c3c0",
	"thumbAcc0": "#6fd16e", "thumbAcc1": "#47c446", "thumbAcc2": "#21b81f", "thumbAccDk": "#1a9118",
	"thumbHot0": "#fbcb74", "thumbHot1": "#fac158", "thumbHot2": "#f9b435", "thumbHotDk": "#c48e2a",
	"thumbPress0": "#66b65b", "thumbPress1": "#48a73b", "thumbPress2": "#229512", "thumbPressDk": "#1b750e",
	"thumbDis0": "#d2d0c6", "thumbDis1": "#ceccc0", "thumbDis2": "#ceccc0", "thumbDisDk": "#b9b6a9",
	"thumbDisBorder": "#ccc9ba", "thumbDisFace": "#f5f4ea",
	"grooveDk": "#9d9c99", "grooveLt": "#ffffff", "groove": "#ecebe4",

	// List view header (ListViewHeader.bmp).
	"hdrFace": "#ebeadb", "hdrLow1": "#e2decd", "hdrLow2": "#d6d2c2", "hdrLow3": "#cbc7b8",
	"hdrDivDk": "#c7c5b2", "hdrDivLt": "#ffffff", "hdrHotFace": "#faf8f3",
	"hdrHot1": "#f8a900", "hdrHot2": "#f6c456", "hdrHot3": "#f8b31f",
	"hdrPressFace": "#dedfd8", "hdrPressDk": "#a5a597", "hdrPressShade": "#c1c2b8", "hdrArrow": "#aca899",

	// Tree view (treeExpandCollapse.bmp).
	"expBorder": "#7898b5", "expTop": "#ffffff", "expBottom": "#c1b8a7", "expSign": "#000000", "treeLine": "#aca899",

	// Group box ([button.groupbox]).
	"groupBorder": "#d0d0bf", "groupText": "#0046d5",

	// Status bar (StatusBackground.bmp, StatusPane.bmp, ResizeGrip2.bmp).
	"statTop": "#959385", "stat1": "#c0bfb6",
	"g.status":  "0 #d8d7cc, .15 #eeede0, .6 #ebeadb, .85 #e3e2d2, 1 #dad8c5",
	"statSepDk": "#cbc7b5", "statSepLt": "#ffffff", "gripDot": "#b8b4a1", "gripDotLt": "#ffffff",

	// Explorer bar group header ([ExplorerBar.NormalGroupHead]).
	"xbHead0": "#ffffff", "xbHead1": "#c6d3f0", "xbText": "#215dc6", "xbHotText": "#428eff",
	"xbBtnRing": "#a5bbe4", "xbBtn0": "#ffffff", "xbBtn1": "#c3d1f0",

	// Caption and frame (FrameCaption.bmp, frameLeft.bmp, frameBottom.bmp):
	// frame lines run outer → inner.
	"g.caption":    "0 #0058ee, .036 #3f97ff, .071 #278dff, .107 #0073fd, .143 #0365f1, .179 #005ae7, .214 #0054e3, .286 #0050e2, .5 #0051e5, .571 #0055f1, .679 #005cf9, .75 #0065fd, .821 #026afe, .893 #0060fc, .929 #005cf9, .964 #0046e0, 1 #0043cf",
	"g.captionOff": "0 #688de0, .036 #98b2e8, .071 #9bb7ea, .107 #8aace7, .143 #7ea2e4, .179 #7a9ae0, .25 #7996de, .357 #7993de, .5 #7a96e0, .607 #7d99e3, .714 #7fa2e6, .821 #81a7e8, .857 #82a9e9, .929 #80a5e7, .964 #7d9be3, 1 #7a93df",
	"frame0":       "#0019cf", "frame1": "#0731d9", "frame2": "#1355e7", "frame3": "#166aee", "frame4": "#0855dd",
	"frameOff0": "#5b68cd", "frameOff1": "#7480dc", "frameOff2": "#7589de", "frameOff3": "#758cdd", "frameOff4": "#758cdc",
	"bottom0": "#00138c", "bottom1": "#001ea0", "bottom2": "#002ebf", "bottom3": "#003ddd", "bottom4": "#0048f1",
	"bottomOff0": "#5b68cd", "bottomOff1": "#6571d2", "bottomOff2": "#6f7ed8", "bottomOff3": "#7589de", "bottomOff4": "#758cdc",
	"capText": "#ffffff", "capShadow": "#0a1883", "capOffText": "#d8e4f8", "capOffShadow": "#00000000",
	// Close button (CloseButton.bmp: normal, hot, pressed, inactive).
	"closeRing": "#ffffff", "closeRingOff": "#bdcbef", "closeGlyph": "#ffffff", "closeGlyphEdge": "#00000000",
	"g.close":      "0 #e45f3e, .05 #e8795f, .15 #e97c62, .3 #e46446, .45 #e35c3a, .6 #e6623c, .75 #e7653c, .85 #e65d32, .9 #e2552a, .95 #d2451e, 1 #ae3110",
	"g.closeHot":   "0 #ff6f5f, .1 #ff8b7d, .3 #ff7868, .45 #ff7664, .65 #ff9279, .75 #ff967c, .85 #ff8b71, .9 #fd7e64, .95 #f16750, 1 #d34936",
	"g.closePress": "0 #752511, .1 #9c3116, .2 #b8391a, .5 #c03e1c, .8 #c74520, 1 #c13f1d",
	"g.closeOff":   "0 #ae768c, .1 #af819c, .3 #ad7590, .5 #ae728b, .7 #b17a8e, .9 #a86f81, 1 #97667b",

	// Office 2003 menus and command bars (ProfessionalColorTable, Blue).
	"menuBorder": "#002d96", "menuBg": "#f6f6f6", "menuSep": "#6a8ccb", "menuDisText": "#8d8d8d",
	"hotFill": "#ffeec2", "hotBorder": "#000080", "menuCheck": "#ffc06f", "menuCheckHot": "#fe803e",
	"g.gutter":   "0 #e3efff, .34 #cbe1fc, .66 #cbe1fc, 1 #7ba4e0",
	"g.selGrad":  "0 #ffffde, .5 #ffe1ac, 1 #ffcb88",
	"g.downGrad": "0 #fe803e, .5 #ffb16d, 1 #ffdf9a",
	"g.chkGrad":  "0 #ffdf9a, .5 #ffc374, 1 #ffa64c",
	"g.openGrad": "0 #e3efff, .5 #a1c5f9, 1 #7ba4e0",
	"g.tb":       "0 #e3efff, .45 #cbe1fc, .55 #cbe1fc, 1 #7ba4e0",
	"tbShadow":   "#3b619c", "tbGripDk": "#274176", "tbGripLt": "#ffffff", "tbSepDk": "#6a8ccb", "tbSepLt": "#f1f9ff",
}

// lunaOlive is Luna Olive Green (HomeStead): green-bordered cream buttons,
// copper hot glow and progress, olive scroll buttons with white glyphs.
var lunaOlive = lunaScheme{
	"btnBorder": "#376206", "btnShade": "#d8c9a8", "btnPressSide": "#e4d4bf",
	"btnDisBorder": "#cac4b8", "btnDisFace": "#f6f2e9",
	"g.btnFace":  "0 #fffff6, .28 #faf9e9, .83 #f6f3e0, .89 #f3eedb, .94 #ece1c9, 1 #e3d1b8",
	"g.btnPress": "0 #dfcdb4, .06 #e7d9c3, .12 #eee6d2, .5 #ece3cc, .88 #eae2ca, .94 #f2ecd8, 1 #f8f4e4",
	"hotTop":     "#fcc595", "hot2": "#edbe96", "hotOutA": "#eec9a5", "hotOutB": "#e39152",
	"hotInA": "#ebb88b", "hotInB": "#e3914f", "hotBot1": "#e3914f", "hotBot2": "#cf7225",
	"defTop": "#c2d18f", "def2": "#b1cb80", "defOutA": "#b1cb7d", "defOutB": "#90c154",
	"defInA": "#b1cb80", "defInB": "#90c154", "defBot1": "#90c154", "defBot2": "#a8a766",
	"fieldBorder": "#a4b97f", "comboBorder": "#a4b97f",
	"sbRing": "#fafafa", "sbEdge": "#8e997d", "g.sbFace": "0 #cbd7ba, .15 #a5b78e, .6 #a0b086, 1 #95a775", "sbGlyph": "#ffffff",
	"sbHotRing": "#fafafa", "sbHotEdge": "#9dab77", "g.sbHotFace": "0 #dae8b9, .15 #c9d5aa, .6 #c6d39b, 1 #c3d096", "sbHotGlyph": "#ffffff",
	"sbPressRing": "#fafafa", "sbPressEdge": "#768361", "g.sbPressFace": "0 #879770, .15 #98aa80, .6 #98ab77, 1 #95aa72", "sbPressGlyph": "#ffffff",
	"sbShadowR": "#9bb383", "sbShadowB": "#83ab5a",
	"f.vertical": "1",
	"thEdge":     "#8da271", "g.thFace": "0 #a6b594, .4 #a4b48b, 1 #95a775",
	"thHotEdge": "#bccc94", "g.thHotFace": "0 #c8d5aa, 1 #c3d096",
	"thPressEdge": "#7e9164", "g.thPressFace": "0 #98aa80, 1 #94a972",
	"gripLt": "#d0dfac", "gripDk": "#8c9d73", "gripHotLt": "#ebf5d4", "gripHotDk": "#b6c68e",
	"gripPressLt": "#b9d097", "gripPressDk": "#7a8b63",
	"tabBorder": "#9bac9c", "tabHotBorder": "#9bac9c", "tabShade": "#d9cfae",
	"g.tabFace": "0 #fffff6, .12 #fffff3, .5 #f9f6e6, .88 #f5f2e0, 1 #f2ecdb",
	"tabHot1":   "#cf7225", "tabHot2": "#e3914f", "tabHot3": "#e39658",
	"g.chunk":   "0 #ebb593, .09 #e9a67f, .18 #e89c6a, .27 #e68c55, .36 #e48245, .45 #e57e3f, .55 #e4844a, .64 #e6935e, .73 #e89f72, .82 #eaaa85, .91 #ebb89a, 1 #ecc7ae",
	"thumbAcc0": "#96ca65", "thumbAcc1": "#82bf49", "thumbAcc2": "#67b221", "thumbAccDk": "#538e1a",
	"thumbHot0": "#f8b174", "thumbHot1": "#f6a358", "thumbHot2": "#f48f35", "thumbHotDk": "#c0712a",
	"thumbPress0": "#92b658", "thumbPress1": "#7ca638", "thumbPress2": "#5e930d", "thumbPressDk": "#4c750b",
	"hdrHot1": "#e39658", "hdrHot2": "#e3914f", "hdrHot3": "#cf7225",
	"expBorder": "#8e997d",
	"groupText": "#99540a",
	"xbHead1":   "#c7dca9", "xbText": "#294600", "xbHotText": "#5b8a1a", "xbBtnRing": "#a9bf89", "xbBtn1": "#d8e6c4",
	"g.caption":    "0 #8ba169, .036 #eaf5c9, .071 #dbe3b2, .107 #bec98f, .143 #b8c58d, .179 #abba83, .214 #a8b680, .357 #a7b580, .5 #aab883, .571 #adbd85, .607 #b0c088, .643 #b3c48b, .679 #b5c68d, .714 #bac88e, .75 #bbc98f, .857 #c2cd95, .893 #bbc98f, .929 #bac88e, .964 #a3b27f, 1 #96a867",
	"g.captionOff": "0 #d6d9bc, .036 #f1f2db, .071 #ebebd1, .107 #dfdec1, .143 #dbdcbf, .179 #d4d7bb, .357 #d3d6ba, .5 #d4d7bb, .607 #d7dbbd, .714 #dcdec0, .821 #dddfc1, .857 #e0e1c3, .929 #dcdec0, .964 #d0d5b9, 1 #cbceb6",
	"frame0":       "#758d5e", "frame1": "#8ba169", "frame2": "#abbd85", "frame3": "#abbd85", "frame4": "#a4b27f",
	"frameOff0": "#c8d0b7", "frameOff1": "#d0d6bd", "frameOff2": "#dbdfc5", "frameOff3": "#e0e2c8", "frameOff4": "#d6d8be",
	"bottom0": "#5e764f", "bottom1": "#899b6d", "bottom2": "#a3ae7e", "bottom3": "#bdc891", "bottom4": "#cbd798",
	"bottomOff0": "#c8d0b7", "bottomOff1": "#d0d6bd", "bottomOff2": "#dbdfc5", "bottomOff3": "#e0e2c8", "bottomOff4": "#d6d8be",
	"capShadow": "#41400a", "capOffText": "#ffffff", "closeRingOff": "#ffffff",
	"g.close":      "0 #d56f4d, .1 #db876c, .2 #dc896e, .35 #d57354, .5 #d36b49, .7 #d7744c, .85 #d56c42, .9 #d0653a, .95 #c1552e, 1 #9f3f1e",
	"g.closeHot":   "0 #f37448, .1 #f08966, .3 #f27c52, .5 #f27e52, .7 #f19363, .85 #f18c5a, .95 #e66d3a, 1 #cb5024",
	"g.closePress": "0 #9b6040, .15 #c5754f, .35 #b9552f, .6 #ba5027, .85 #be5329, 1 #b84e26",
	"g.closeOff":   "0 #bc9b9d, .1 #bfa3aa, .5 #bc989b, .8 #bf9f9f, .95 #b89996, 1 #ab8e91",
	// Office 2003 HomeStead.
	"menuBorder": "#758d5e", "menuBg": "#f4f4ee", "menuSep": "#608058", "hotBorder": "#3f5d38",
	"g.gutter":   "0 #ffffed, .34 #cedca7, .66 #cedca7, 1 #b5c48f",
	"g.openGrad": "0 #edf0d6, .5 #bac98f, 1 #b5c48f",
	"g.tb":       "0 #ffffed, .45 #cedca7, .55 #cedca7, 1 #b5c48f",
	"tbShadow":   "#608058", "tbGripDk": "#515e33", "tbSepDk": "#608058", "tbSepLt": "#f4f7de",
}

// lunaSilver is Luna Silver (Metallic): lavender-grey buttons and scroll
// buttons with dark outlines, a silver caption with a black title.
var lunaSilver = lunaScheme{
	"shadow": "#9d9da1", "chkDisWell": "#e0dfe3",
	"btnShade": "#00000000", "btnSides": "#ffffff", "btnPressSide": "#00000000",
	"btnDisBorder": "#c4c3bf", "btnDisFace": "#f1f1ed",
	"g.btnFace":   "0 #ffffff, .11 #fdfdfd, .22 #fdfdfd, .28 #f8fcfd, .39 #f4f5fd, .44 #f0f1fc, .5 #e9ebf5, .56 #e3e5f0, .61 #dedfec, .67 #d9dae7, .78 #d6d7e7, .83 #d2d1e4, .89 #cdccdf, .94 #c6c5d7, 1 #c6c5d7",
	"g.btnPress":  "0 #ffffff, .06 #acabbd, .17 #b8b7ca, .33 #bfc0cd, .44 #c9cbd6, .56 #d6d7e2, .67 #e5e6ef, .78 #f5f7fd, .89 #ffffff, 1 #ffffff",
	"fieldBorder": "#a5acb2",
	"sbRing":      "#9495a2", "sbEdge": "#ffffff", "g.sbFace": "0 #ffffff, .15 #fbfcfc, .3 #ececf1, .45 #d9dae4, .6 #cccddb, 1 #cbccda", "sbGlyph": "#3f3d3d",
	"sbHotRing": "#5b6665", "sbHotEdge": "#ffffff", "g.sbHotFace": "0 #ffffff, .35 #fbfbff, .5 #e2e3f0, .65 #d1d2e4, 1 #cfd1e3", "sbHotGlyph": "#202020",
	"sbPressRing": "#5b6665", "sbPressEdge": "#ffffff", "g.sbPressFace": "0 #bfc2db, .15 #ced4e8, .3 #e9edf3, .45 #ffffff, 1 #ffffff", "sbPressGlyph": "#202020",
	"sbShadowR": "#00000000", "sbShadowB": "#00000000",
	"f.vertical": "1",
	"thRing":     "#9495a2", "thEdge": "#ffffff", "g.thFace": "0 #f7f7f9, .25 #ececf1, .5 #e3e2eb, .75 #d4d5e0, 1 #cccddb",
	"thHotRing": "#5b6665", "thHotEdge": "#ffffff", "g.thHotFace": "0 #fefeff, .5 #e5e5ee, 1 #c9cbdc",
	"thPressRing": "#434848", "thPressEdge": "#ffffff", "g.thPressFace": "0 #c5c7d8, .4 #e0e0eb, .7 #f6f5f9, 1 #ffffff",
	"gripLt": "#ffffff", "gripDk": "#8e95a2", "gripHotLt": "#ffffff", "gripHotDk": "#8e95a2",
	"gripPressLt": "#ffffff", "gripPressDk": "#8e95a2",
	"trackEdge": "#e5e6ee", "track0": "#eceef3", "track1": "#fbfbfe",
	"trackPressEdge": "#c2c3d7", "trackPress0": "#d3d7e3", "trackPress1": "#f6f6fd",
	"tabHotBorder": "#99a0a3", "tabShade": "#00000000",
	"g.tabFace": "0 #ffffff, .3 #faf9fe, .5 #eaeaf9, .7 #dfdff1, .85 #cccde2, 1 #bebed8",
	"pane2":     "#f0efea",
	"g.chunk":   "0 #95b38e, .09 #99c88e, .18 #a4d498, .27 #c3e3ba, .36 #a4d498, .45 #8dbc82, .55 #83ae76, .64 #76a66a, .73 #83ae76, .82 #83ae76, .91 #8fbc82, 1 #98c88c",
	"thumbFace": "#d4d3e1", "thumbFaceHi": "#ffffff", "thumbFaceLo": "#b7b6c4",
	"hdrFace": "#f9fafd", "hdrLow1": "#e6e7ef", "hdrLow2": "#d1d2de", "hdrLow3": "#bdbece",
	"hdrDivDk": "#b5b6c8", "hdrDivLt": "#fefefe", "hdrHotFace": "#fefefe",
	"hdrPressFace": "#ececf3", "hdrPressDk": "#808099", "hdrPressShade": "#b9b9c8",
	"expBorder": "#9495a2", "expBottom": "#c5cfd9",
	"groupBorder": "#bfb8bf",
	"statTop":     "#8a8b8f", "stat1": "#b9b9bc",
	"g.status": "0 #d0d0d4, .15 #e5e5e9, .6 #e0e0e5, .85 #d1d2d7, 1 #cccdd2",
	"xbHead1":  "#d6d4e8", "xbText": "#3e3c52", "xbHotText": "#6c6a89", "xbBtnRing": "#b1b7bb", "xbBtn1": "#e2e1ef",
	"g.caption":    "0 #66667e, .036 #a8a7bf, .071 #ffffff, .107 #d7d8e2, .143 #bcbccf, .179 #a8a7bf, .214 #a4a3be, .25 #acabc4, .357 #b4b6c7, .5 #bdc0ce, .607 #cbcdd8, .714 #dcdee5, .786 #eaebf0, .857 #f7f7f9, .893 #ffffff, .929 #e4e3e3, .964 #bcbdcd, 1 #9898a7",
	"g.captionOff": "0 #babac5, .036 #eceef5, .071 #ffffff, .107 #eceef5, .143 #e4e5ed, .214 #d7d7e3, .357 #dedfea, .5 #e6e8ed, .643 #f4f3f5, .786 #fbfbfc, .893 #ffffff, .929 #fdfffc, .964 #eaedf5, 1 #cccbd9",
	"frame0":       "#66667e", "frame1": "#fbfcfd", "frame2": "#fbfcfd", "frame3": "#a8a9bb", "frame4": "#66667e",
	"frameOff0": "#c9c9d7", "frameOff1": "#fdfffc", "frameOff2": "#fdfffc", "frameOff3": "#e6e9e4", "frameOff4": "#c9c9d7",
	"bottom0": "#66667e", "bottom1": "#fbfcfd", "bottom2": "#a8a9bb", "bottom3": "#a8a9bb", "bottom4": "#66667e",
	"bottomOff0": "#c9c9d7", "bottomOff1": "#fdfffc", "bottomOff2": "#e6e9e4", "bottomOff3": "#e6e9e4", "bottomOff4": "#c9c9d7",
	"capText": "#0e1010", "capShadow": "#c7c2d1", "capOffText": "#a2a1a1",
	"closeRing": "#a63944", "closeRingOff": "#c19095", "closeGlyphEdge": "#461e1a",
	"g.close":      "0 #f9e9b6, .05 #f2b89e, .15 #eb988b, .25 #e97e78, .4 #e36c6c, .6 #de6767, .75 #d65f63, .9 #cd595e, 1 #c6555c",
	"g.closeHot":   "0 #f9e9b6, .05 #ffcba4, .15 #ffad8f, .3 #fe8a71, .5 #fb806b, .75 #f47664, .9 #ea6a5d, 1 #df6157",
	"g.closePress": "0 #c23f4b, .05 #fae9b6, .15 #eb695c, .45 #f57668, .7 #fe8a71, .85 #ffc7a2, 1 #f9e9b6",
	"g.closeOff":   "0 #f6efd8, .1 #eed4c9, .3 #e6b4b2, .6 #e2adad, .9 #d8a4a6, 1 #d4a1a4",
	// Office 2003 Metallic.
	"menuBorder": "#7c7c94", "menuBg": "#fdfaff", "menuSep": "#6e6d8f", "hotBorder": "#4b4b6f",
	"g.gutter":   "0 #f9f9ff, .34 #e1e2ec, .66 #e1e2ec, 1 #9391b0",
	"g.openGrad": "0 #e8e9f2, .5 #b8b9ca, 1 #acaac2",
	"g.tb":       "0 #f9f9ff, .45 #e1e2ec, .55 #e1e2ec, 1 #9391b0",
	"tbShadow":   "#7c7c94", "tbGripDk": "#545475", "tbSepDk": "#6e6d8f", "tbSepLt": "#ffffff",
}

// lunaRoyale is Royale (Media Center Edition 2005, "Energy Blue"): glossy
// blue captions split at half height, white-to-steel buttons, glossy
// progress, and Office XP menus (Office's Royale table is the XP look).
var lunaRoyale = lunaScheme{
	"shadow": "#a7a6aa", "gray": "#a7a6aa",
	"btnBorder": "#2d5082", "btnShade": "#00000000", "btnPressSide": "#00000000",
	"btnDisBorder": "#b4b7b4", "btnDisFace": "#f4f4f2",
	"g.btnFace":  "0 #f8fcff, .15 #fcfdfd, .35 #f2f9f9, .5 #e7eff5, .58 #d4dfed, .66 #c6d4e6, .76 #b8c8dc, .88 #aabcd1, 1 #9ab0d2",
	"g.btnPress": "0 #7994bb, .12 #93a8c6, .25 #abbcd2, .4 #bccbdf, .5 #cfdbeb, .6 #e6eef4, .75 #f1f8f9, .9 #fbfdfd, 1 #ffffff",
	"hotTop":     "#fff3a3", "hot2": "#f5e8a1", "hotOutA": "#faeca1", "hotOutB": "#dc802d",
	"hotInA": "#f5e8a1", "hotInB": "#e7b14b", "hotBot1": "#e7a743", "hotBot2": "#e78325",
	"sbRing": "#8599b1", "sbEdge": "#ffffff", "g.sbFace": "0 #ffffff, .12 #ffffff, .16 #c2d6ee, .85 #c2d6ee, 1 #bcd0e8", "sbGlyph": "#5b6473",
	"sbHotRing": "#52667e", "sbHotEdge": "#ffffff", "g.sbHotFace": "0 #ffffff, .16 #d3e7ff, 1 #c1d5ed", "sbHotGlyph": "#4b5463",
	"sbPressRing": "#52667e", "sbPressEdge": "#f2f7ff", "g.sbPressFace": "0 #c3d7ef, .25 #ffffff, 1 #ffffff", "sbPressGlyph": "#4b5463",
	"sbShadowR": "#00000000", "sbShadowB": "#00000000",
	"f.vertical": "1",
	"thRing":     "#8599b1", "thEdge": "#ffffff", "g.thFace": "0 #f2f7ff, .3 #e2efff, .6 #cbdff7, 1 #bdd1e9",
	"thHotRing": "#52667e", "thHotEdge": "#ffffff", "g.thHotFace": "0 #fefeff, .4 #eff6ff, 1 #bbcfe7",
	"thPressRing": "#364a62", "thPressEdge": "#ffffff", "g.thPressFace": "0 #b7cbe3, .5 #d0e4fc, 1 #fcfdff",
	"gripLt": "#ffffff", "gripDk": "#8599b1", "gripHotLt": "#ffffff", "gripHotDk": "#6b7f97",
	"gripPressLt": "#ffffff", "gripPressDk": "#52667e",
	"trackEdge": "#e2e0e6", "track0": "#edebef", "track1": "#faf9fb",
	"trackPressEdge": "#c6c4cb", "trackPress0": "#d8d6dd", "trackPress1": "#f3f2f5",
	"paneBorder": "#879bb3", "pane": "#fbfcff", "pane2": "#f1f0f4", "paneShadow1": "#cfcdd4", "paneShadow2": "#e1dfe5",
	"progBorder": "#bcbcbc", "progIn": "#d5d5d5", "progIn2": "#e1e1e1", "progTrough": "#ececec",
	"g.chunk": "0 #7ac381, .45 #5fbe64, .5 #39d145, 1 #18e621",
	"hdrFace": "#fcfcfc", "hdrLow1": "#f7f7f7", "hdrLow2": "#f0f0f0", "hdrLow3": "#e3e3e6",
	"hdrDivDk": "#d5d4da", "hdrDivLt": "#ffffff", "hdrHotFace": "#ffffff",
	"hdrPressFace": "#e6e5ea", "hdrPressDk": "#a7a6aa", "hdrPressShade": "#cfced4", "hdrArrow": "#a7a6aa",
	"expBorder": "#9fa6ad", "expBottom": "#dde2e7", "expSign": "#4c4c4c", "treeLine": "#a7a6aa",
	"groupBorder": "#c6c5cc",
	"statTop":     "#a7a6aa", "stat1": "#d6d5da",
	"g.status":     "0 #f3f2f5, .5 #ebe9ed, 1 #e2e0e6",
	"statSepDk":    "#c1c1c4",
	"g.caption":    "0 #2a4dab, .033 #99c6f1, .067 #83c2ed, .2 #76b8ec, .33 #6daae9, .43 #609ee7, .5 #5593de, .52 #3980d2, .6 #3677c9, .7 #3369c1, .8 #315fb7, .9 #2c55b1, .967 #2c55b1, 1 #254394",
	"g.captionOff": "0 #4d73c3, .033 #85b3e3, .067 #4f74b7, .3 #5580c0, .5 #5b91ce, .52 #6fa1d9, .7 #80b2e2, .9 #8ec0e6, .967 #90c3e6, 1 #79a6e3",
	"frame0":       "#2a4dab", "frame1": "#84b0df", "frame2": "#5e89cb", "frame3": "#3262bd", "frame4": "#254394",
	"frameOff0": "#4d73c3", "frameOff1": "#a2c0e7", "frameOff2": "#86a9dc", "frameOff3": "#6c93d3", "frameOff4": "#4d73c3",
	"bottom0": "#1e3a82", "bottom1": "#2c55b1", "bottom2": "#5e89cb", "bottom3": "#84b0df", "bottom4": "#3262bd",
	"bottomOff0": "#4d73c3", "bottomOff1": "#6c93d3", "bottomOff2": "#86a9dc", "bottomOff3": "#a2c0e7", "bottomOff4": "#6c93d3",
	"capShadow": "#0e2a6e", "capOffText": "#f8f8f8",
	"closeRing": "#4d73c3", "closeRingOff": "#285b94",
	"g.close":      "0 #e7aa9b, .1 #d78f7e, .3 #d98674, .45 #cf7b67, .5 #c74f32, .6 #cc593d, .75 #da755d, .9 #e8967e, 1 #eec4b9",
	"g.closeHot":   "0 #ffb8a8, .1 #f0907c, .45 #e46248, .5 #c73a1c, .7 #d4502f, .9 #e8866c, 1 #f2b6a6",
	"g.closePress": "0 #b95050, .1 #dd7676, .45 #b24949, .5 #9d3535, .9 #993131, 1 #ef8686",
	"g.closeOff":   "0 #aecbe7, .1 #88aad6, .45 #3a7ac9, .5 #649ee2, .9 #84b4e6, 1 #9ec2ea",
	// Office XP look (Office 2003's Royale table).
	"menuBorder": "#868588", "menuBg": "#fcfcfc", "menuSep": "#c1c1c4", "menuDisText": "#a7a6aa",
	"hotFill": "#c2cfe5", "hotBorder": "#335ea8", "menuCheck": "#e2e5ee", "menuCheckHot": "#99afd4",
	"g.gutter":   "0 #eeedf0, 1 #eeedf0",
	"g.selGrad":  "0 #c2cfe5, 1 #c2cfe5",
	"g.downGrad": "0 #99afd4, 1 #99afd4",
	"g.chkGrad":  "0 #e2e5ee, 1 #e2e5ee",
	"g.openGrad": "0 #fcfcfc, 1 #f5f4f6",
	"g.tb":       "0 #fcfcfc, .45 #f5f4f6, .55 #f5f4f6, 1 #ebe9ed",
	"tbShadow":   "#c9c7cd", "tbGripDk": "#a7a6aa", "tbSepDk": "#c1c1c4", "tbSepLt": "#ffffff",
}

// lunaNoir is Royale Noir (2005) as a dark scheme: the black glossy caption
// and orange-red close button of the original on graphite chrome, with the
// scheme's #5b83bd highlight and Office XP-style dark menus.
var lunaNoir = lunaScheme{
	"shadow": "#11151b", "hi": "#5c6572",
	"gray": "#6f7784", "disText": "#6f7784",
	"btnBorder": "#0a0c10", "btnShade": "#00000000", "btnSides": "#00000000", "btnPressSide": "#1a1e24",
	"btnDisBorder": "#5a626d", "btnDisFace": "#3d444f",
	"g.btnFace":  "0 #6b7684, .45 #4e5866, .5 #414a57, 1 #353d48",
	"g.btnPress": "0 #22272e, .5 #2b313a, 1 #3a424e",
	"defTop":     "#9ec0f0", "def2": "#7ea5de", "defOutA": "#7ea5de", "defOutB": "#4a72ac",
	"defInA": "#7ea5de", "defInB": "#4a72ac", "defBot1": "#4a72ac", "defBot2": "#36598f",
	"chkBorder": "#8a95a5", "chkWell0": "#181b21", "chkWell1": "#2e343e",
	"chkHot0": "#ffd28a", "chkHot1": "#e59700", "chkHotWell": "#252a32",
	"chkPress0": "#101317", "chkPress1": "#262b33",
	"chkDisBorder": "#4a515c", "chkDisWell": "#2b313b",
	"tick": "#5fd35d", "tickDis": "#5a616c",
	"g.dot":       "0 #a8f5a5, .45 #4ec24c, 1 #2a8f28",
	"fieldBorder": "#5d6877", "fieldDisBorder": "#3f4650", "fieldDis": "#2b313b",
	"comboBorder": "#5d6877", "comboDis": "#2b313b",
	"sbRing": "#0d1015", "sbEdge": "#6a7482", "g.sbFace": "0 #5a6472, .5 #465060, 1 #3a4350", "sbGlyph": "#dde3ea",
	"sbHotRing": "#0d1015", "sbHotEdge": "#8591a1", "g.sbHotFace": "0 #6c7888, 1 #4b5666", "sbHotGlyph": "#ffffff",
	"sbPressRing": "#0d1015", "sbPressEdge": "#2a3039", "g.sbPressFace": "0 #262b33, 1 #343b46", "sbPressGlyph": "#dde3ea",
	"sbDisRing": "#20252c", "sbDisEdge": "#333a44", "g.sbDisFace": "0 #2f353f, 1 #2b313a", "sbDisGlyph": "#5a616c",
	"sbShadowR": "#00000000", "sbShadowB": "#00000000",
	"f.vertical": "1",
	"thRing":     "#0d1015", "thEdge": "#7a8494", "g.thFace": "0 #5d6776, 1 #434c59",
	"thHotRing": "#0d1015", "thHotEdge": "#939eae", "g.thHotFace": "0 #6f7a8a, 1 #525c6a",
	"thPressRing": "#0d1015", "thPressEdge": "#3a414b", "g.thPressFace": "0 #2e343d, 1 #3e4652",
	"gripLt": "#8a95a5", "gripDk": "#1c2027", "gripHotLt": "#a3adbb", "gripHotDk": "#1c2027",
	"gripPressLt": "#6f7784", "gripPressDk": "#14171c",
	"trackEdge": "#1d2127", "track0": "#262b33", "track1": "#2d333c",
	"trackPressEdge": "#111418", "trackPress0": "#1b1f25", "trackPress1": "#23282f",
	"tabBorder": "#0d1015", "tabHotBorder": "#0d1015", "tabShade": "#00000000",
	"g.tabFace": "0 #525b69, .5 #444c59, 1 #3a424e",
	"pane":      "#383f4a", "pane2": "#2f353f", "paneBorder": "#0d1015",
	"paneShadow1": "#1d2127", "paneShadow2": "#262b32",
	"progBorder": "#0c0e11", "progIn": "#15181c", "progIn2": "#1b1f25", "progTrough": "#20252c",
	"g.chunk":     "0 #6fbf76, .45 #52b058, .5 #2fc03b, 1 #16d41f",
	"thumbBorder": "#0d1015", "thumbBorderDk": "#07090c", "thumbBorderLo": "#0d1015",
	"thumbFace": "#4e5866", "thumbFaceHi": "#6a7482", "thumbFaceLo": "#353d48",
	"thumbDis0": "#454c56", "thumbDis1": "#40464f", "thumbDis2": "#40464f", "thumbDisDk": "#30353d",
	"thumbDisBorder": "#262b32", "thumbDisFace": "#3a414b",
	"grooveDk": "#0c0e11", "grooveLt": "#4a525e", "groove": "#1f242b",
	"hdrFace": "#3d4450", "hdrLow1": "#373e49", "hdrLow2": "#313742", "hdrLow3": "#2a2f38",
	"hdrDivDk": "#22272e", "hdrDivLt": "#4d5663", "hdrHotFace": "#48505d",
	"hdrPressFace": "#30363f", "hdrPressDk": "#0d1015", "hdrPressShade": "#23282f", "hdrArrow": "#9aa3ae",
	"expBorder": "#7a8494", "expTop": "#5a6472", "expBottom": "#353d48", "expSign": "#e6e8ec", "treeLine": "#5d6877",
	"groupBorder": "#4d5663", "groupText": "#8fb4ff",
	"statTop": "#0d1015", "stat1": "#1d2127",
	"g.status":  "0 #3a414c, .5 #333a45, 1 #2b313b",
	"statSepDk": "#1d2127", "statSepLt": "#4a525e", "gripDot": "#6f7784", "gripDotLt": "#1d2127",
	"xbHead0": "#5b6677", "xbHead1": "#3e4859", "xbText": "#e6e8ec", "xbHotText": "#ffffff",
	"xbBtnRing": "#0d1015", "xbBtn0": "#6a7482", "xbBtn1": "#434c59",
	"g.caption":    "0 #1f2739, .033 #919195, .067 #66737c, .2 #5c6975, .33 #556271, .45 #4d5968, .5 #4b5767, .52 #334151, .7 #2c3748, .85 #263142, .967 #232a3d, 1 #161d2d",
	"g.captionOff": "0 #020307, .033 #363c44, .067 #111929, .45 #0e1422, .52 #040a17, .967 #040a17, 1 #2b333b",
	"frame0":       "#161d2d", "frame1": "#798188", "frame2": "#434c5a", "frame3": "#1a2430", "frame4": "#0d1015",
	"frameOff0": "#020307", "frameOff1": "#2b333b", "frameOff2": "#111929", "frameOff3": "#040a17", "frameOff4": "#020307",
	"bottom0": "#0b0f18", "bottom1": "#1a2430", "bottom2": "#434c5a", "bottom3": "#798188", "bottom4": "#161d2d",
	"bottomOff0": "#020307", "bottomOff1": "#040a17", "bottomOff2": "#111929", "bottomOff3": "#2b333b", "bottomOff4": "#020307",
	"capText": "#ffffff", "capShadow": "#000000", "capOffText": "#9aa3ae",
	"closeRing": "#434343", "closeRingOff": "#2c3b4e",
	"g.close":      "0 #f08f78, .1 #d58471, .45 #e46848, .5 #db3918, .7 #e95d38, .9 #e88f7a, 1 #f4af9f",
	"g.closeHot":   "0 #ffa088, .45 #f06a4a, .5 #e8401c, .9 #f09a84, 1 #f8bcac",
	"g.closePress": "0 #af8175, .45 #9d5d4e, .5 #973c26, .9 #b27d6f, 1 #b5958c",
	"g.closeOff":   "0 #2c3b4e, .1 #040c1d, 1 #040a17",
	"menuBorder":   "#0d1015", "menuBg": "#2b313b", "menuSep": "#4a525e", "menuDisText": "#6f7784",
	"hotFill": "#3c5a84", "hotBorder": "#6f95cf", "menuCheck": "#34507a", "menuCheckHot": "#2c4468",
	"g.gutter":   "0 #353f4d, 1 #353f4d",
	"g.selGrad":  "0 #3c5a84, 1 #3c5a84",
	"g.downGrad": "0 #2c4468, 1 #2c4468",
	"g.chkGrad":  "0 #34507a, 1 #34507a",
	"g.openGrad": "0 #3a414c, 1 #353c47",
	"g.tb":       "0 #4a5360, .45 #3c4450, .55 #3c4450, 1 #2e343e",
	"tbShadow":   "#0d1015", "tbGripDk": "#0d1015", "tbGripLt": "#6f7784", "tbSepDk": "#1d2127", "tbSepLt": "#4a525e",
}

// lunaSchemes is indexed by the "scheme" param.
var lunaSchemes = [...]lunaScheme{lunaBlue, lunaOlive, lunaSilver, lunaRoyale, lunaNoir}

// lunaKeyMiss counts lookups of keys missing from lunaBlue (a typo in the
// engine); the tests require it to stay zero.
var lunaKeyMiss atomic.Int32

// ---- resolved colour set --------------------------------------------------------

// lunaGlow is the 2px inner glow of a push button: two top rows, the outer
// and inner side columns (vertical gradients) and two bottom rows.
type lunaGlow struct {
	top1, top2, bot1, bot2 paintengine2d.Color
	out, in                []paintengine2d.GradientStop
}

// lunaBtn is one state of an arrow button or scroll thumb: outer ring,
// inner edge, face gradient and glyph colour.
type lunaBtn struct {
	ring, edge, glyph paintengine2d.Color
	face              []paintengine2d.GradientStop
}

// lunaSet is a look's resolved Luna colours, built once per look.
type lunaSet struct {
	vertical bool

	face, text, gray, disText, field, sel, selText paintengine2d.Color
	shadow, hi, info, infoText, infoBorder         paintengine2d.Color

	btnBorder, btnShade, btnSides, btnPressSide, btnDisBorder, btnDisFace paintengine2d.Color
	btnFace, btnPress                                                     []paintengine2d.GradientStop
	hot, def                                                              lunaGlow

	chkBorder, chkDisBorder, chkDisWell, tick, tickDis paintengine2d.Color
	chkWell, chkHot, chkHotWell, chkPress, dot         []paintengine2d.GradientStop

	fieldBorder, fieldDisBorder, fieldDis, comboBorder, comboDis paintengine2d.Color

	arrow            [4]lunaBtn // normal, hot, pressed, disabled
	thumb            [3]lunaBtn // normal, hot, pressed
	grip             [3][2]paintengine2d.Color
	shadowR, shadowB paintengine2d.Color
	trackEdge        [2]paintengine2d.Color
	track            [2][]paintengine2d.GradientStop

	tabBorder, tabHotBorder, tabShade, tabHot1, tabHot2, tabHot3 paintengine2d.Color
	tabFace, pane                                                []paintengine2d.GradientStop
	paneFlat, paneBorder, paneShadow1, paneShadow2               paintengine2d.Color

	progBorder, progIn, progIn2, progTrough paintengine2d.Color
	chunk                                   []paintengine2d.GradientStop

	thumbBorder, thumbBorderDk, thumbBorderLo, thumbDisBorder, thumbDisFace paintengine2d.Color
	thumbFace, thumbDisGrad                                                 []paintengine2d.GradientStop
	thumbAcc                                                                [4][4]paintengine2d.Color
	grooveDk, grooveLt, groove                                              paintengine2d.Color

	hdrFace, hdrHotFace, hdrDivDk, hdrDivLt, hdrPressFace, hdrPressDk, hdrPressShade, hdrArrow paintengine2d.Color
	hdrLow, hdrHot                                                                             [3]paintengine2d.Color

	expBorder, expSign, treeLine paintengine2d.Color
	exp                          []paintengine2d.GradientStop

	groupBorder, groupText paintengine2d.Color

	statTop, stat1, statSepDk, statSepLt, gripDot, gripDotLt paintengine2d.Color
	status                                                   []paintengine2d.GradientStop

	xbText, xbHotText, xbBtnRing paintengine2d.Color
	xbHead, xbBtn                []paintengine2d.GradientStop

	caption, captionOff                   []paintengine2d.GradientStop
	frame, frameOff, bottom, bottomOff    [5]paintengine2d.Color
	capText, capShadow, capOff, capOffSh  paintengine2d.Color
	closeRing, closeRingOff, closeGlyph   paintengine2d.Color
	closeGlyphEdge                        paintengine2d.Color
	close, closeHot, closePress, closeOff []paintengine2d.GradientStop

	menuBorder, menuBg, menuSep, menuDisText, hotFill, hotBorder, menuCheck, menuCheckHot paintengine2d.Color
	gutter, selGrad, downGrad, chkGrad, openGrad, tb                                      []paintengine2d.GradientStop
	tbShadow, tbGripDk, tbGripLt, tbSepDk, tbSepLt                                        paintengine2d.Color

	// Labels resolved to read on the fills above (4.5:1).
	hotText, openText, selText2, downText, chkText, checkText, checkHotText paintengine2d.Color
}

type lunaKey struct{}

// lunaColors is the look's resolved colour set (built once per look).
func lunaColors(l *Classic) *lunaSet {
	return l.Memo(lunaKey{}, func() any { return lunaBuild(l) }).(*lunaSet)
}

// lunaSchemeOf is the scheme table a look paints with.
func lunaSchemeOf(l *Classic) lunaScheme {
	i := int(l.P("scheme", 0))
	if i < 0 || i >= len(lunaSchemes) {
		i = 0
	}
	return lunaSchemes[i]
}

func lunaBuild(l *Classic) *lunaSet {
	sc := lunaSchemeOf(l)
	raw := func(k string) string {
		if v, ok := sc[k]; ok {
			return v
		}
		v, ok := lunaBlue[k]
		if !ok {
			lunaKeyMiss.Add(1)
		}
		return v
	}
	col := func(k string) paintengine2d.Color { return l.X(k, Hex(raw(k))) }
	grad := func(k string) []paintengine2d.GradientStop { return lunaParseGrad(raw("g." + k)) }
	two := func(a, b paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, a), Stop(1, b)}
	}
	p := l.palette
	c := &lunaSet{
		vertical: l.P("vertical", float32(atoiOr(raw("f.vertical"), 0))) != 0,
		face:     p.SurfaceAlt,
		text:     p.Text,
		field:    p.Field,
		sel:      p.Selection,
	}
	if c.sel.A < 0.9 {
		c.sel = p.Accent
	}
	// Selected text is the scheme's highlight text unless it would not read
	// on the highlight (Olive Green's white on #93a070 is 2.8:1).
	c.selText = ReadableOn(c.sel, 3, p.TextOnAccent, p.Text)
	c.gray, c.disText = col("gray"), col("disText")
	c.shadow, c.hi = col("shadow"), col("hi")
	c.info, c.infoText, c.infoBorder = col("info"), col("infoText"), col("infoBorder")

	c.btnBorder, c.btnShade, c.btnSides = col("btnBorder"), col("btnShade"), col("btnSides")
	c.btnPressSide, c.btnDisBorder, c.btnDisFace = col("btnPressSide"), col("btnDisBorder"), col("btnDisFace")
	c.btnFace, c.btnPress = grad("btnFace"), grad("btnPress")
	glow := func(pre string) lunaGlow {
		return lunaGlow{
			top1: col(pre + "Top"), top2: col(pre + "2"), bot1: col(pre + "Bot1"), bot2: col(pre + "Bot2"),
			out: two(col(pre+"OutA"), col(pre+"OutB")), in: two(col(pre+"InA"), col(pre+"InB")),
		}
	}
	c.hot, c.def = glow("hot"), glow("def")

	c.chkBorder = col("chkBorder")
	c.chkDisBorder, c.chkDisWell = col("chkDisBorder"), col("chkDisWell")
	c.tick, c.tickDis = col("tick"), col("tickDis")
	c.chkWell = two(col("chkWell0"), col("chkWell1"))
	c.chkHot = two(col("chkHot0"), col("chkHot1"))
	c.chkPress = two(col("chkPress0"), col("chkPress1"))
	c.chkHotWell = two(col("chkWell0"), Mix(col("chkHotWell"), col("chkWell1"), 0.6))
	c.dot = grad("dot")

	c.fieldBorder, c.fieldDisBorder, c.fieldDis = col("fieldBorder"), col("fieldDisBorder"), col("fieldDis")
	c.comboBorder, c.comboDis = col("comboBorder"), col("comboDis")

	for i, pre := range [...]string{"sb", "sbHot", "sbPress", "sbDis"} {
		c.arrow[i] = lunaBtn{ring: col(pre + "Ring"), edge: col(pre + "Edge"), glyph: col(pre + "Glyph"), face: grad(pre + "Face")}
	}
	for i, pre := range [...]string{"th", "thHot", "thPress"} {
		c.thumb[i] = lunaBtn{ring: col(pre + "Ring"), edge: col(pre + "Edge"), face: grad(pre + "Face")}
	}
	for i, pre := range [...]string{"grip", "gripHot", "gripPress"} {
		c.grip[i] = [2]paintengine2d.Color{col(pre + "Lt"), col(pre + "Dk")}
	}
	c.shadowR, c.shadowB = col("sbShadowR"), col("sbShadowB")
	c.trackEdge[0], c.track[0] = col("trackEdge"), two(col("track0"), col("track1"))
	c.trackEdge[1], c.track[1] = col("trackPressEdge"), two(col("trackPress0"), col("trackPress1"))

	c.tabBorder, c.tabHotBorder, c.tabShade = col("tabBorder"), col("tabHotBorder"), col("tabShade")
	c.tabHot1, c.tabHot2, c.tabHot3 = col("tabHot1"), col("tabHot2"), col("tabHot3")
	c.tabFace = grad("tabFace")
	c.paneFlat = col("pane")
	c.pane = two(c.paneFlat, col("pane2"))
	c.paneBorder, c.paneShadow1, c.paneShadow2 = col("paneBorder"), col("paneShadow1"), col("paneShadow2")

	c.progBorder, c.progIn, c.progIn2, c.progTrough = col("progBorder"), col("progIn"), col("progIn2"), col("progTrough")
	c.chunk = grad("chunk")

	c.thumbBorder, c.thumbBorderDk, c.thumbBorderLo = col("thumbBorder"), col("thumbBorderDk"), col("thumbBorderLo")
	c.thumbDisBorder, c.thumbDisFace = col("thumbDisBorder"), col("thumbDisFace")
	c.thumbDisGrad = two(c.thumbDisFace, c.thumbDisFace)
	c.thumbFace = []paintengine2d.GradientStop{Stop(0, col("thumbFaceHi")), Stop(0.2, col("thumbFace")), Stop(0.7, col("thumbFace")), Stop(1, col("thumbFaceLo"))}
	for i, pre := range [...]string{"thumbAcc", "thumbHot", "thumbPress", "thumbDis"} {
		dk := pre + "Dk"
		if pre == "thumbAcc" {
			dk = "thumbAccDk"
		}
		c.thumbAcc[i] = [4]paintengine2d.Color{col(pre + "0"), col(pre + "1"), col(pre + "2"), col(dk)}
	}
	c.grooveDk, c.grooveLt, c.groove = col("grooveDk"), col("grooveLt"), col("groove")

	c.hdrFace, c.hdrHotFace = col("hdrFace"), col("hdrHotFace")
	c.hdrDivDk, c.hdrDivLt = col("hdrDivDk"), col("hdrDivLt")
	c.hdrPressFace, c.hdrPressDk, c.hdrPressShade, c.hdrArrow = col("hdrPressFace"), col("hdrPressDk"), col("hdrPressShade"), col("hdrArrow")
	c.hdrLow = [3]paintengine2d.Color{col("hdrLow1"), col("hdrLow2"), col("hdrLow3")}
	c.hdrHot = [3]paintengine2d.Color{col("hdrHot1"), col("hdrHot2"), col("hdrHot3")}

	c.expBorder, c.expSign, c.treeLine = col("expBorder"), col("expSign"), col("treeLine")
	c.exp = two(col("expTop"), col("expBottom"))

	c.groupBorder, c.groupText = col("groupBorder"), col("groupText")

	c.statTop, c.stat1, c.statSepDk, c.statSepLt = col("statTop"), col("stat1"), col("statSepDk"), col("statSepLt")
	c.gripDot, c.gripDotLt = col("gripDot"), col("gripDotLt")
	c.status = grad("status")

	c.xbText, c.xbHotText, c.xbBtnRing = col("xbText"), col("xbHotText"), col("xbBtnRing")
	c.xbHead = two(col("xbHead0"), col("xbHead1"))
	c.xbBtn = two(col("xbBtn0"), col("xbBtn1"))

	c.caption, c.captionOff = grad("caption"), grad("captionOff")
	if a, ok := l.tokens.Extra["caption"]; ok {
		c.caption = two(a, l.X("caption2", a))
	}
	for i := range c.frame {
		n := string(rune('0' + i))
		c.frame[i], c.frameOff[i] = col("frame"+n), col("frameOff"+n)
		c.bottom[i], c.bottomOff[i] = col("bottom"+n), col("bottomOff"+n)
	}
	c.capText, c.capShadow, c.capOff, c.capOffSh = col("capText"), col("capShadow"), col("capOffText"), col("capOffShadow")
	c.closeRing, c.closeRingOff = col("closeRing"), col("closeRingOff")
	c.closeGlyph, c.closeGlyphEdge = col("closeGlyph"), col("closeGlyphEdge")
	c.close, c.closeHot, c.closePress, c.closeOff = grad("close"), grad("closeHot"), grad("closePress"), grad("closeOff")

	c.menuBorder, c.menuBg, c.menuSep, c.menuDisText = col("menuBorder"), col("menuBg"), col("menuSep"), col("menuDisText")
	c.hotFill, c.hotBorder, c.menuCheck, c.menuCheckHot = col("hotFill"), col("hotBorder"), col("menuCheck"), col("menuCheckHot")
	c.gutter, c.selGrad, c.downGrad = grad("gutter"), grad("selGrad"), grad("downGrad")
	c.chkGrad, c.openGrad, c.tb = grad("chkGrad"), grad("openGrad"), grad("tb")
	c.tbShadow, c.tbGripDk, c.tbGripLt = col("tbShadow"), col("tbGripDk"), col("tbGripLt")
	c.tbSepDk, c.tbSepLt = col("tbSepDk"), col("tbSepLt")

	on := func(g []paintengine2d.GradientStop) paintengine2d.Color {
		return ReadableOn(g[len(g)/2].Color, 4.5, c.text, c.hi)
	}
	c.hotText = ReadableOn(c.hotFill, 4.5, c.text, c.hi)
	c.openText, c.selText2, c.downText, c.chkText = on(c.openGrad), on(c.selGrad), on(c.downGrad), on(c.chkGrad)
	c.checkText = ReadableOn(c.menuCheck, 4.5, c.text, c.hi)
	c.checkHotText = ReadableOn(c.menuCheckHot, 4.5, c.text, c.hi)
	return c
}

// lunaParseGrad reads "offset #colour, offset #colour, …".
func lunaParseGrad(s string) []paintengine2d.GradientStop {
	parts := strings.Split(s, ",")
	out := make([]paintengine2d.GradientStop, 0, len(parts))
	for _, part := range parts {
		f := strings.Fields(part)
		if len(f) != 2 {
			continue
		}
		at, err := strconv.ParseFloat(f[0], 32)
		if err != nil {
			continue
		}
		out = append(out, Stop(float32(at), Hex(f[1])))
	}
	if len(out) == 0 {
		out = append(out, Stop(0, Hex("#808080")), Stop(1, Hex("#808080")))
	}
	return out
}

func atoiOr(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return v
	}
	return def
}

// ---- drawing helpers --------------------------------------------------------------

// lunaPx is one line of the look: 1 device pixel at 1x, 2 at 2x.
func lunaPx(l *Classic) float32 {
	v := float32(int(l.S(1) + 0.5))
	if v < 1 {
		v = 1
	}
	return v
}

// lunaSnap puts a rect on the pixel grid so 1px lines stay crisp.
func lunaSnap(b paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(b.Max.X)-x0, snap(b.Max.Y)-y0)
}

// lunaFrame fills b with a lw-wide border of colour edge and returns the
// rect and radius left for the face.
func lunaFrame(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, edge paintengine2d.Color) (paintengine2d.Rect, float32) {
	if edge.A > 0 {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(edge))
	}
	ri := r - lw
	if ri < 0 {
		ri = 0
	}
	return b.Inset(lw), ri
}

// lunaBorder strokes a square lw-wide border inside b with four rects.
func lunaBorder(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, col paintengine2d.Color) {
	if b.Dx() < 2*lw || b.Dy() < 2*lw || col.A <= 0 {
		return
	}
	fill := paintengine2d.Fill(col)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), fill)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), fill)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, lw, b.Dy()-2*lw), fill)
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+lw, lw, b.Dy()-2*lw), fill)
}

// lunaChevron fills the XP arrow glyph — a 9×6 chevron three pixels thick,
// the shape of ScrollArrowGlyphs.bmp — pointing dir, centred in b, u
// pixels per glyph unit.
func lunaChevron(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, u float32, col paintengine2d.Color) {
	if col.A <= 0 || b.Empty() {
		return
	}
	// The up chevron in glyph units: apex (4.5,0), wings to x 0 and 9.
	pts := [6][2]float32{{4.5, 0}, {9, 4.5}, {8, 6}, {4.5, 2.5}, {1, 6}, {0, 4.5}}
	w, h := float32(9), float32(6)
	if dir == DirLeft || dir == DirRight {
		w, h = h, w
	}
	ox := snap((b.Min.X+b.Max.X)*0.5 - w*u*0.5)
	oy := snap((b.Min.Y+b.Max.Y)*0.5 - h*u*0.5)
	path := paintengine2d.NewPath()
	for i, p := range pts {
		x, y := p[0], p[1]
		switch dir {
		case DirDown:
			y = 6 - y
		case DirLeft:
			x, y = y, x
		case DirRight:
			x, y = 6-y, x
		}
		if i == 0 {
			path.MoveTo(ox+x*u, oy+y*u)
		} else {
			path.LineTo(ox+x*u, oy+y*u)
		}
	}
	path.Close()
	ctx.DrawPath(path, paintengine2d.Fill(col))
}

// lunaGlyphUnit is the chevron unit for a button of side s: 1px glyph units
// in a 17px scroll button, scaled with the button.
func lunaGlyphUnit(s float32) float32 {
	u := s / 17
	if u < 0.6 {
		u = 0.6
	}
	return u
}

// faceGrad is the gradient across a button face: diagonal (Blue, top-left
// light) or top-to-bottom (Olive, Silver, Royale).
func (c *lunaSet) faceGrad(b paintengine2d.Rect, stops []paintengine2d.GradientStop) paintengine2d.Paint {
	if c.vertical {
		return VGradient(b, stops...)
	}
	return DGradient(b, stops...)
}

// ---- parts: push button, arrow button, glow -----------------------------------------

// pushButton paints button.bmp: a 3px-rounded dark border, the face
// gradient with its shaded right edge, and the hot / default inner glow.
func (c *lunaSet) pushButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	b = lunaSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	lw := lunaPx(l)
	r := l.rx(3)
	if st.Disabled() {
		in, ri := lunaFrame(ctx, b, r, lw, c.btnDisBorder)
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(c.btnDisFace))
		return
	}
	in, ri := lunaFrame(ctx, b, r, lw, c.btnBorder)
	pressed := st.Pressed()
	if pressed {
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.btnPress...))
	} else {
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.btnFace...))
	}
	ctx.Save()
	ctx.ClipRoundRect(in, ri, ri)
	switch {
	case pressed && c.btnPressSide.A > 0:
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.btnPressSide))
	case !pressed && c.btnShade.A > 0:
		ctx.DrawRect(paintengine2d.XYWH(in.Max.X-lw, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.btnShade.WithAlpha(0.45)))
		ctx.DrawRect(paintengine2d.XYWH(in.Max.X-2*lw, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.btnShade.WithAlpha(0.2)))
	}
	if c.btnSides.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.btnSides))
		ctx.DrawRect(paintengine2d.XYWH(in.Max.X-lw, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.btnSides))
	}
	switch {
	case pressed:
	case st.Hovered():
		lunaGlowPaint(ctx, in, lw, &c.hot)
	case st.Primary() || st.Focused():
		lunaGlowPaint(ctx, in, lw, &c.def)
	}
	ctx.Restore()
}

// lunaGlowPaint draws a 2px glow inside the face rect in (clipped by the
// caller to the face's rounded corners).
func lunaGlowPaint(ctx *paintengine2d.Context, in paintengine2d.Rect, lw float32, g *lunaGlow) {
	if in.Dx() < 4*lw || in.Dy() < 4*lw {
		return
	}
	sy, sh := in.Min.Y+2*lw, in.Dy()-4*lw
	outer := paintengine2d.XYWH(in.Min.X, sy, lw, sh)
	inner := paintengine2d.XYWH(in.Min.X+lw, sy, lw, sh)
	ctx.DrawRect(outer, VGradient(outer, g.out...))
	ctx.DrawRect(outer.Translate(paintengine2d.Pt(in.Dx()-lw, 0)), VGradient(outer, g.out...))
	ctx.DrawRect(inner, VGradient(inner, g.in...))
	ctx.DrawRect(inner.Translate(paintengine2d.Pt(in.Dx()-3*lw, 0)), VGradient(inner, g.in...))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(g.top1))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, in.Dx(), lw), paintengine2d.Fill(g.top2))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Max.Y-2*lw, in.Dx(), lw), paintengine2d.Fill(g.bot1))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Max.Y-lw, in.Dx(), lw), paintengine2d.Fill(g.bot2))
}

// arrowButton paints a scroll arrow / combo / spin button or a thumb: a 1px
// ring, a 1px inner edge, the face gradient and (shadow) the soft 1px
// shadow ScrollArrows.bmp casts on the right and bottom. axis picks the
// face gradient of a thumb: 0 = button (diagonal or vertical per scheme),
// 1 = across a vertical thumb, 2 = across a horizontal thumb.
func (c *lunaSet) arrowButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, s *lunaBtn, shadow bool, axis int) paintengine2d.Rect {
	b = lunaSnap(b)
	lw := lunaPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return b
	}
	if shadow && c.shadowR.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+lw, lw, b.Dy()-2*lw), paintengine2d.Fill(c.shadowR))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+lw, b.Max.Y-lw, b.Dx()-2*lw, lw), paintengine2d.Fill(c.shadowB))
		b = paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-lw, b.Dy()-lw)
	}
	r := l.rx(2)
	in, ri := lunaFrame(ctx, b, r, lw, s.ring)
	in2, ri2 := lunaFrame(ctx, in, ri, lw, s.edge)
	var paint paintengine2d.Paint
	switch {
	case axis == 1:
		paint = HGradient(in2, s.face...)
	case axis == 2:
		paint = VGradient(in2, s.face...)
	default:
		paint = c.faceGrad(in2, s.face)
	}
	ctx.DrawRoundRect(in2, ri2, ri2, paint)
	return b
}

// ---- parts ------------------------------------------------------------------------

func (e lunaEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := lunaColors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	switch role {
	case RoleButton:
		c.pushButton(l, ctx, b, st)
	case RoleTool:
		return c.officeButton(l, ctx, b, st)
	case RoleField, RoleCombo:
		fill, border := c.field, c.fieldBorder
		if role == RoleCombo {
			border = c.comboBorder
		}
		if st.Disabled() {
			fill, border = c.fieldDis, c.fieldDisBorder
			if role == RoleCombo {
				fill = c.comboDis
			}
		}
		b = lunaSnap(b)
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		lunaBorder(ctx, b, lunaPx(l), border)
		if !st.Disabled() {
			fg = l.fieldText()
		}
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, false)
	case RoleRow:
		if st.Checked() {
			fill, fg := c.selFor(st)
			ctx.DrawRect(b, paintengine2d.Fill(fill))
			return fg
		}
		return l.fieldText()
	case RoleMenu:
		e.MenuHighlight(l, ctx, b, false)
	case RoleThumb:
		i := 0
		switch {
		case st.Pressed():
			i = 2
		case st.Hovered():
			i = 1
		}
		axis := 1
		if b.Dx() > b.Dy() {
			axis = 2
		}
		c.arrowButton(l, ctx, b, &c.thumb[i], false, axis)
	case RoleTrack:
		c.trackPaint(ctx, lunaSnap(b), b.Dy() >= b.Dx(), 0, lunaPx(l))
	case RoleTab:
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
	}
	return fg
}

// officeButton is an Office command-bar button: flat until hot, then the
// hot-track gradient in a 1px border; pressed and checked go orange
// (Office 2003) or deeper blue (Office XP).
func (c *lunaSet) officeButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	if st.Disabled() {
		return c.disText
	}
	var grad []paintengine2d.GradientStop
	fg := c.text
	switch {
	case st.Pressed(), st.Checked() && st.Hovered():
		grad, fg = c.downGrad, c.downText
	case st.Checked():
		grad, fg = c.chkGrad, c.chkText
	case st.Hovered() || st.Focused():
		grad, fg = c.selGrad, c.selText2
	default:
		return c.text
	}
	b = lunaSnap(b)
	ctx.DrawRect(b, VGradient(b, grad...))
	lunaBorder(ctx, b, lunaPx(l), c.hotBorder)
	return fg
}

func (e lunaEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := lunaColors(l)
	box = lunaSnap(box)
	lw := lunaPx(l)
	if box.Dx() < 4*lw {
		return
	}
	in := box.Inset(lw)
	tick := c.tick
	if st.Disabled() {
		ctx.DrawRect(box, paintengine2d.Fill(c.chkDisBorder))
		ctx.DrawRect(in, paintengine2d.Fill(c.chkDisWell))
		tick = c.tickDis
	} else {
		ctx.DrawRect(box, paintengine2d.Fill(c.chkBorder))
		switch {
		case st.Pressed():
			ctx.DrawRect(in, DGradient(in, c.chkPress...))
		case st.Hovered():
			ctx.DrawRect(in, DGradient(in, c.chkHot...))
			well := in.Inset(2 * lw)
			ctx.DrawRect(well, DGradient(well, c.chkHotWell...))
		default:
			ctx.DrawRect(in, DGradient(in, c.chkWell...))
		}
	}
	if checked || st.Checked() {
		// The 7×7 pixel tick of CheckBox13.bmp, three pixels thick.
		PixelTick(ctx, box.Inset(snap(box.Dx()*3/13)), tick)
	}
}

func (e lunaEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := lunaColors(l)
	box = lunaSnap(box)
	lw := lunaPx(l)
	cx, cy := (box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5
	r := box.Dx() * 0.5
	if box.Dy() < box.Dx() {
		r = box.Dy() * 0.5
	}
	if r < 3*lw {
		return
	}
	ctr := paintengine2d.Pt(cx, cy)
	disc := func(rad float32) paintengine2d.Rect { return paintengine2d.XYWH(cx-rad, cy-rad, rad*2, rad*2) }
	dot := c.tick
	if st.Disabled() {
		ctx.DrawCircle(ctr, r, paintengine2d.Fill(c.chkDisBorder))
		ctx.DrawCircle(ctr, r-lw, paintengine2d.Fill(c.chkDisWell))
		dot = c.tickDis
	} else {
		ctx.DrawCircle(ctr, r, paintengine2d.Fill(c.chkBorder))
		ri := r - lw
		switch {
		case st.Pressed():
			ctx.DrawCircle(ctr, ri, DGradient(disc(ri), c.chkPress...))
		case st.Hovered():
			ctx.DrawCircle(ctr, ri, DGradient(disc(ri), c.chkHot...))
			ctx.DrawCircle(ctr, ri-2*lw, DGradient(disc(ri-2*lw), c.chkWell...))
		default:
			ctx.DrawCircle(ctr, ri, DGradient(disc(ri), c.chkWell...))
		}
	}
	if selected || st.Checked() {
		rd := r * 0.37
		if st.Disabled() {
			ctx.DrawCircle(ctr, rd, paintengine2d.Fill(dot))
			return
		}
		ctx.DrawCircle(ctr, rd, paintengine2d.Radial(paintengine2d.RadialGradient{
			Center: paintengine2d.Pt(cx-rd*0.35, cy-rd*0.35), Radius: rd * 1.35, Stops: c.dot,
		}))
	}
}

// Arrow is the XP chevron glyph, fitted into b.
func (lunaEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	u := s / 11
	if m := l.S(1); u > m {
		u = m
	}
	lunaChevron(ctx, b, dir, u, col)
}

// Expander is the TreeView +/- box of treeExpandCollapse.bmp: 9px, a blue
// grey frame over a white-to-beige face, black sign.
func (lunaEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := lunaColors(l)
	lw := lunaPx(l)
	s := snap(l.S(9))
	box := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-s)*0.5), snap(b.Min.Y+(b.Dy()-s)*0.5), s, s)
	in, _ := lunaFrame(ctx, box, l.rx(1), lw, c.expBorder)
	ctx.DrawRect(in, VGradient(in, c.exp...))
	arm := snap(l.S(2))
	mid := snap(box.Min.Y + s*0.5 - lw*0.5)
	ctx.DrawRect(paintengine2d.XYWH(box.Min.X+arm, mid, s-2*arm, lw), paintengine2d.Fill(c.expSign))
	if !expanded {
		ctx.DrawRect(paintengine2d.XYWH(snap(box.Min.X+s*0.5-lw*0.5), box.Min.Y+arm, lw, s-2*arm), paintengine2d.Fill(c.expSign))
	}
}

// MenuHighlight is the Office hot-track: a pale fill in a 1px border.
func (lunaEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	if b.Dx() < 3 || b.Dy() < 3 {
		return
	}
	lw := lunaPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.hotFill))
	lunaBorder(ctx, b, lw, c.hotBorder)
	if attachBottom {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+lw, b.Max.Y-lw, b.Dx()-2*lw, lw), paintengine2d.Fill(c.hotFill))
	}
}

func (lunaEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := lunaColors(l)
	if hot {
		return c.hotText
	}
	return c.text
}

// Fields show only the caret when focused (XP drew no focus ring on edits).
func (lunaEngine) FieldFocusRing(l *Classic) bool { return false }

func (lunaEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	DottedRect(ctx, lunaSnap(b).Inset(snap(l.S(2))), lunaColors(l).text)
}

// ---- scrollbars -----------------------------------------------------------------

func (lunaEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 17, Arrows: ArrowsEnds, MinThumb: 10}
}

// trackPaint fills a scroll shaft: its 1px edges and the gradient across it
// (ScrollShaftVertical.bmp); pressed = 1 for a paged part.
func (c *lunaSet) trackPaint(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, pressed int, lw float32) {
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.trackEdge[pressed]))
	var in paintengine2d.Rect
	if vertical {
		in = paintengine2d.XYWH(b.Min.X+lw, b.Min.Y, b.Dx()-2*lw, b.Dy())
		ctx.DrawRect(in, HGradient(in, c.track[pressed]...))
		return
	}
	in = paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), b.Dy()-2*lw)
	ctx.DrawRect(in, VGradient(in, c.track[pressed]...))
}

func (e lunaEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := lunaColors(l)
	lw := lunaPx(l)
	bar := lunaSnap(p.Bar)
	c.trackPaint(ctx, bar, vertical, 0, lw)
	if (st.Pressed == ScrollPageDec || st.Pressed == ScrollPageInc) && !p.Thumb.Empty() {
		pg := p.Track
		if vertical {
			if st.Pressed == ScrollPageDec {
				pg.Max.Y = p.Thumb.Min.Y
			} else {
				pg.Min.Y = p.Thumb.Max.Y
			}
		} else if st.Pressed == ScrollPageDec {
			pg.Max.X = p.Thumb.Min.X
		} else {
			pg.Min.X = p.Thumb.Max.X
		}
		pg = lunaSnap(pg)
		if vertical {
			pg.Min.X, pg.Max.X = bar.Min.X, bar.Max.X
		} else {
			pg.Min.Y, pg.Max.Y = bar.Min.Y, bar.Max.Y
		}
		c.trackPaint(ctx, pg, vertical, 1, lw)
	}
	// Buttons leave the shaft's 1px edge on the leading side, like the
	// bitmaps (column 0 of ScrollArrows.bmp is the shaft).
	lead := func(b paintengine2d.Rect) paintengine2d.Rect {
		b = lunaSnap(b)
		if vertical {
			return paintengine2d.XYWH(b.Min.X+lw, b.Min.Y, b.Dx()-lw, b.Dy())
		}
		return paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), b.Dy()-lw)
	}
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		s := &c.arrow[0]
		switch {
		case st.Disabled:
			s = &c.arrow[3]
		case st.Pressed == part:
			s = &c.arrow[2]
		case st.Hot == part:
			s = &c.arrow[1]
		}
		face := c.arrowButton(l, ctx, lead(b), s, true, 0)
		side := face.Dx()
		if face.Dy() < side {
			side = face.Dy()
		}
		lunaChevron(ctx, face, dir, lunaGlyphUnit(side+lw), s.glyph)
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
	axis := 1
	if !vertical {
		axis = 2
	}
	face := c.arrowButton(l, ctx, lead(p.Thumb), &c.thumb[i], true, axis)
	// The gripper: four short ridges (a light line over a dark one)
	// centred on the thumb, when it is long enough to hold them.
	along, across := face.Dy(), face.Dx()
	if !vertical {
		along, across = face.Dx(), face.Dy()
	}
	if along < l.S(14) || across < l.S(9) {
		return
	}
	g := c.grip[i]
	gl := snap(l.S(6))
	cx, cy := snap((face.Min.X+face.Max.X)*0.5), snap((face.Min.Y+face.Max.Y)*0.5)
	for k := 0; k < 4; k++ {
		o := float32(k*2-4) * lw
		if vertical {
			x0, y := cx-snap(gl*0.5), cy+o
			ctx.DrawRect(paintengine2d.XYWH(x0, y, gl, lw), paintengine2d.Fill(g[0]))
			ctx.DrawRect(paintengine2d.XYWH(x0+lw, y+lw, gl, lw), paintengine2d.Fill(g[1]))
		} else {
			x, y0 := cx+o, cy-snap(gl*0.5)
			ctx.DrawRect(paintengine2d.XYWH(x, y0, lw, gl), paintengine2d.Fill(g[0]))
			ctx.DrawRect(paintengine2d.XYWH(x+lw, y0+lw, lw, gl), paintengine2d.Fill(g[1]))
		}
	}
}

// DrawScrollBar is the thumb-in-track fallback for callers without
// scrollbar geometry.
func (e lunaEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	e.Face(l, ctx, track, RoleTrack, StateNone)
	if !thumb.Empty() {
		e.Face(l, ctx, thumb, RoleThumb, st)
	}
}

// ---- frames -----------------------------------------------------------------------

func (lunaEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is button.groupbox: a 3px-rounded 1px beige frame with the
// title in the scheme's group-box blue (brown on Olive Green). XP group
// boxes are transparent; a raised one is a card on the tab-pane face.
func (e lunaEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	f := l.body
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	frame := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	r := l.rx(3)
	if raised {
		in, ri := lunaFrame(ctx, frame, r, lw, c.groupBorder)
		ctx.DrawRoundRect(in, ri, ri, VGradient(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), l.S(300)), c.pane...))
	}
	var tx, tw float32
	if title != "" {
		tx = b.Min.X + l.S(8)
		tw = f.Advance(title) + l.S(6)
		if tw > b.Dx()-l.S(16) {
			tw = b.Dx() - l.S(16)
		}
	}
	if !raised || title != "" {
		// The frame, left open behind the title.
		stroke := paintengine2d.StrokePaint(c.groupBorder, lw)
		fr := frame.Inset(lw * 0.5)
		if title == "" {
			ctx.DrawRoundRect(fr, r, r, stroke)
		} else {
			gap := paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height())
			for _, clip := range [3]paintengine2d.Rect{
				paintengine2d.XYWH(b.Min.X, b.Min.Y, gap.Min.X-b.Min.X, b.Dy()),
				paintengine2d.XYWH(gap.Max.X, b.Min.Y, b.Max.X-gap.Max.X, b.Dy()),
				paintengine2d.XYWH(gap.Min.X, gap.Max.Y, gap.Dx(), b.Max.Y-gap.Max.Y),
			} {
				if clip.Empty() {
					continue
				}
				ctx.Save()
				ctx.ClipRect(clip)
				ctx.DrawRoundRect(fr, r, r, stroke)
				ctx.Restore()
			}
		}
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.groupText, AlignStart, 0)
	}
}

// lunaCaptionH is the in-app caption: XP's 25px bar plus its top edge at
// 11px Tahoma, grown with the toolkit font.
func lunaCaptionH(l *Classic) float32 {
	f := l.bold
	if f == nil {
		f = l.body
	}
	h := f.Height() + l.S(8)
	if m := l.S(26); h < m {
		h = m
	}
	return snap(h)
}

// lunaFrameW is the sizing frame: five lines, outer to inner.
func lunaFrameW(l *Classic) float32 { return 5 * lunaPx(l) }

func (lunaEngine) WindowFrameInsets(l *Classic) Insets {
	fw := lunaFrameW(l)
	return Insets{Top: lunaCaptionH(l), Right: fw, Bottom: fw, Left: fw}
}

// WindowCloseRect is CloseButton.bmp's place: 21px square at the top right
// of a 29px caption (offset -25, 5), scaled with the caption.
// PopupShadow: XP's "shadows under menus" — a small soft shadow down and to
// the right of menus and tooltips (windows got shadows only in Vista).
func (lunaEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	if kind == PopupDialog {
		return Insets{}
	}
	return ShadowReach(l.S(2), l.S(2), l.S(6), -l.S(1))
}

func (lunaEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if kind == PopupDialog {
		return
	}
	DropShadow(ctx, b, 0, paintengine2d.RGBA(0, 0, 0, 0.32), l.S(2), l.S(2), l.S(6), -l.S(1))
}

func (lunaEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = lunaSnap(b)
	h := lunaCaptionH(l)
	side := snap(h * 21 / 29)
	top := snap(h * 5 / 29)
	right := snap(h*4/29) + lunaPx(l)
	return paintengine2d.XYWH(b.Max.X-right-side, b.Min.Y+top, side, side)
}

// Windows dialogs put the default button first: "OK  Cancel".
func (lunaEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return 0
}

// DrawWindowFrame is the Luna window: the rounded caption gradient
// (FrameCaption.bmp), the five-line sizing frame, a bold title with its
// drop shadow and the red close button.
func (e lunaEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	capH := lunaCaptionH(l)
	if b.Dx() < 4*capH*0.5 || b.Dy() < capH {
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		return
	}
	side, bottom, capGrad := c.frame, c.bottom, c.caption
	text, shadow := c.capText, c.capShadow
	if !st.Active {
		side, bottom, capGrad = c.frameOff, c.bottomOff, c.captionOff
		text, shadow = c.capOff, c.capOffSh
	}
	r := l.rx(7)
	shape := RoundRectPath(b, r, r, 0, 0)
	ctx.Save()
	ctx.ClipPath(shape)
	// Sizing frame: nested lines outer → inner; the bottom rows on top.
	for i := 0; i < 5; i++ {
		f := float32(i) * lw
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+f, b.Min.Y, b.Dx()-2*f, b.Dy()-f), paintengine2d.Fill(side[i]))
	}
	for i := 0; i < 5; i++ {
		f := float32(i) * lw
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+f, b.Max.Y-f-lw, b.Dx()-2*f, lw), paintengine2d.Fill(bottom[i]))
	}
	// Caption, darkened toward its left and right ends like the bitmap.
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), capH)
	ctx.DrawRect(bar, VGradient(bar, capGrad...))
	for i, a := range [2]float32{0.85, 0.35} {
		f := float32(i) * lw
		edge := paintengine2d.Fill(side[i].WithAlpha(a))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X+f, bar.Min.Y+r*0.5, lw, bar.Dy()-r*0.5), edge)
		ctx.DrawRect(paintengine2d.XYWH(bar.Max.X-f-lw, bar.Min.Y+r*0.5, lw, bar.Dy()-r*0.5), edge)
	}
	ctx.Restore()
	fw := lunaFrameW(l)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+fw, b.Min.Y+capH, b.Dx()-2*fw, b.Dy()-capH-fw), paintengine2d.Fill(c.face))

	right := bar.Max.X - fw
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		c.closeButton(l, ctx, cb, st)
		right = cb.Min.X - l.S(4)
	}
	if title != "" {
		f := l.bold
		if f == nil {
			f = l.body
		}
		tb := paintengine2d.XYWH(bar.Min.X+l.S(8), bar.Min.Y+lw, right-bar.Min.X-l.S(8), bar.Dy()-lw)
		if shadow.A > 0 {
			l.drawFittedText(ctx, f, title, tb.Translate(paintengine2d.Pt(lw, lw)), shadow, AlignStart, 0)
		}
		l.drawFittedText(ctx, f, title, tb, text, AlignStart, 0)
	}
}

// closeButton paints CloseButton.bmp: a light ring, the red gradient face
// with a gloss in its upper left, and the white ×.
func (c *lunaSet) closeButton(l *Classic, ctx *paintengine2d.Context, cb paintengine2d.Rect, st WindowState) {
	lw := lunaPx(l)
	ring, grad := c.closeRing, c.close
	switch {
	case !st.Active:
		ring, grad = c.closeRingOff, c.closeOff
	case st.ClosePress:
		grad = c.closePress
	case st.CloseHot:
		grad = c.closeHot
	}
	r := l.rx(3)
	in, ri := lunaFrame(ctx, cb, r, lw, ring)
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, grad...))
	if st.Active && !st.ClosePress {
		gl := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx()*0.7, in.Dy()*0.6)
		ctx.Save()
		ctx.ClipRoundRect(in, ri, ri)
		ctx.DrawRoundRect(gl, gl.Dy()*0.5, gl.Dy()*0.5, paintengine2d.Radial(paintengine2d.RadialGradient{
			Center: gl.Min, Radius: gl.Dx(),
			Stops: lunaGloss,
		}))
		ctx.Restore()
	}
	g := cb.Inset(snap(cb.Dx() * 0.27))
	w := cb.Dx() * 0.13
	if w < lw*1.5 {
		w = lw * 1.5
	}
	if c.closeGlyphEdge.A > 0 {
		lunaCross(ctx, g, c.closeGlyphEdge, w+2*lw)
	}
	lunaCross(ctx, g, c.closeGlyph, w)
}

// Fixed gradients: the close button's gloss and the message-box icons.
var (
	lunaGloss     = []paintengine2d.GradientStop{Stop(0, paintengine2d.RGBA(1, 1, 1, 0.35)), Stop(1, paintengine2d.RGBA(1, 1, 1, 0))}
	lunaErrorDisc = []paintengine2d.GradientStop{Stop(0, Hex("#ff9f86")), Stop(0.45, Hex("#f04a26")), Stop(1, Hex("#b41c06"))}
	lunaInfoDisc  = []paintengine2d.GradientStop{Stop(0, Hex("#c9e0ff")), Stop(0.45, Hex("#3d86f0")), Stop(1, Hex("#1049b8"))}
	lunaWarnFace  = []paintengine2d.GradientStop{Stop(0, Hex("#fff3a0")), Stop(1, Hex("#f5b400"))}
)

// lunaCross fills the caption × (CloseGlyph.bmp): two bars w thick from
// corner to corner of b, as quads so the ends stay flat and whole.
func lunaCross(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, w float32) {
	if b.Empty() || w <= 0 {
		return
	}
	fill := paintengine2d.Fill(col)
	for _, d := range [2][4]float32{{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}, {b.Max.X, b.Min.Y, b.Min.X, b.Max.Y}} {
		x0, y0, x1, y1 := d[0], d[1], d[2], d[3]
		dx, dy := x1-x0, y1-y0
		n := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		nx, ny := -dy/n*w*0.5, dx/n*w*0.5
		p := paintengine2d.NewPath()
		p.MoveTo(x0+nx, y0+ny)
		p.LineTo(x1+nx, y1+ny)
		p.LineTo(x1-nx, y1-ny)
		p.LineTo(x0-nx, y0-ny)
		p.Close()
		ctx.DrawPath(p, fill)
	}
}

// ---- controls ---------------------------------------------------------------------

func (e lunaEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := lunaColors(l)
	fg := e.Face(l, ctx, b, RoleButton, st)
	l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, 8)
	if st.Focused() && !st.Disabled() {
		// Just inside the 1px border and 2px glow (ContentMargins 3).
		DottedRect(ctx, lunaSnap(b).Inset(3*lunaPx(l)), c.text)
	}
}

func (e lunaEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, lunaSnap(box), st, label)
}

func (e lunaEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, lunaSnap(box), st, label)
}

// toggleLabel draws a check box / radio caption with the Windows dotted
// focus rectangle around it.
func (lunaEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := lunaColors(l)
	if label == "" {
		if st.Focused() {
			DottedRect(ctx, box.Inset(-lunaPx(l)).Intersect(b), c.text)
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
		DottedRect(ctx, labelFocusRect(l.body, label, lb, b), c.text)
	}
}

func (e lunaEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	if !st.Disabled() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	// Disabled edits: #ebebe4 face, grey text, no caret or selection.
	c := lunaColors(l)
	e.Face(l, ctx, b, RoleField, st)
	pad := l.metrics.FieldPad
	if pad <= 0 {
		pad = 8
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	show := text
	if show == "" {
		show = placeholder
	}
	ctx.Save()
	ctx.ClipRect(inner)
	f := l.faceOrBody(face)
	f.Draw(ctx, show, paintengine2d.Pt(inner.Min.X-scrollX, inner.Min.Y+(inner.Dy()-f.Height())*0.5), c.disText)
	ctx.Restore()
}

// DrawComboBox is [Combobox]: a white field in the edit border with the
// drop-down button — a scroll-arrow button — inside it at the right. A
// focused drop-down list highlights its text like XP did.
func (e lunaEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	e.Face(l, ctx, b, RoleCombo, st)
	in := b.Inset(lw)
	bw := snap(in.Dy() * 0.79)
	if m := l.S(13); bw < m {
		bw = m
	}
	btn := paintengine2d.XYWH(in.Max.X-bw, in.Min.Y, bw, in.Dy())
	s := &c.arrow[0]
	switch {
	case st.Disabled():
		s = &c.arrow[3]
	case open || st.Pressed():
		s = &c.arrow[2]
	case st.Hovered():
		s = &c.arrow[1]
	}
	face := c.arrowButton(l, ctx, btn, s, false, 0)
	lunaChevron(ctx, face, DirDown, lunaGlyphUnit(face.Dx()+2*lw), s.glyph)
	tb := paintengine2d.XYWH(in.Min.X+l.S(2), in.Min.Y+l.S(2), btn.Min.X-in.Min.X-l.S(4), in.Dy()-l.S(4))
	tc := l.fieldText()
	switch {
	case st.Disabled():
		tc = c.disText
	case st.Focused() && !open:
		tb = lunaSnap(tb)
		ctx.DrawRect(tb, paintengine2d.Fill(c.sel))
		tc = c.selText
		DottedRect(ctx, tb, c.selText)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(tb.Min.X+l.S(3), tb.Min.Y, tb.Dx()-l.S(4), tb.Dy()), tc, AlignStart, 0)
}

// DrawSpinner is the up-down control: two stacked arrow buttons.
func (e lunaEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(r paintengine2d.Rect, dir Direction, hover, press bool) {
		s := &c.arrow[0]
		switch {
		case st.Disabled():
			s = &c.arrow[3]
		case press:
			s = &c.arrow[2]
		case hover:
			s = &c.arrow[1]
		}
		face := c.arrowButton(l, ctx, r, s, false, 0)
		lunaChevron(ctx, face, dir, lunaGlyphUnit(face.Dx()+2*lw)*0.8, s.glyph)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress)
}

// DrawTabBar is the strip the tabs stand on: the window face with the tab
// pane's top edge along its bottom.
func (lunaEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.paneBorder))
}

// DrawTab is tabItem.bmp: rounded top corners, a white-to-beige face and
// the orange top edge when hot or selected; the selected tab is taller and
// opens into the pane.
// TabOutset: XP's selected tab is 2px wider on each side than its slot.
func (lunaEngine) TabOutset(l *Classic) Insets {
	return Insets{Left: snap(l.S(2)), Right: snap(l.S(2))}
}

func (lunaEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	t := b
	if !selected {
		// Unselected tabs touch and sit 2px lower on the pane edge; the
		// selected one is grown by TabOutset and overlaps them.
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+snap(l.S(2)), b.Dx(), b.Dy()-snap(l.S(2))-lw)
	}
	if t.Dx() < 6*lw || t.Dy() < 6*lw {
		return
	}
	hot := !selected && !st.Disabled() && (st.Hovered() || st.Pressed())
	border := c.tabBorder
	switch {
	case selected:
		border = c.paneBorder
	case hot:
		border = c.tabHotBorder
	}
	r := l.rx(3)
	ctx.DrawPath(RoundRectPath(t, r, r, 0, 0), paintengine2d.Fill(border))
	in := paintengine2d.XYWH(t.Min.X+lw, t.Min.Y+lw, t.Dx()-2*lw, t.Dy()-lw)
	ri := r - lw
	if ri < 0 {
		ri = 0
	}
	inner := RoundRectPath(in, ri, ri, 0, 0)
	if selected {
		ctx.DrawPath(inner, paintengine2d.Fill(c.paneFlat))
	} else {
		ctx.DrawPath(inner, VGradient(in, c.tabFace...))
		if c.tabShade.A > 0 {
			ctx.DrawRect(paintengine2d.XYWH(in.Max.X-lw, in.Min.Y+ri, lw, in.Dy()-ri), paintengine2d.Fill(c.tabShade.WithAlpha(0.45)))
			ctx.DrawRect(paintengine2d.XYWH(in.Max.X-2*lw, in.Min.Y+ri, lw, in.Dy()-ri), paintengine2d.Fill(c.tabShade.WithAlpha(0.2)))
		}
	}
	if selected || hot {
		ctx.Save()
		ctx.ClipPath(RoundRectPath(t, r, r, 0, 0))
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X, t.Min.Y, t.Dx(), lw), paintengine2d.Fill(c.tabHot1))
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X, t.Min.Y+lw, t.Dx(), lw), paintengine2d.Fill(c.tabHot2))
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X+lw, t.Min.Y+2*lw, t.Dx()-2*lw, lw), paintengine2d.Fill(c.tabHot3))
		ctx.Restore()
	}
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	lb := paintengine2d.XYWH(t.Min.X, t.Min.Y+2*lw, t.Dx(), t.Dy()-2*lw)
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, 8)
	if st.Focused() && selected {
		f := l.body
		w := f.Advance(label) + l.S(6)
		if w > lb.Dx()-l.S(6) {
			w = lb.Dx() - l.S(6)
		}
		h := f.Height()
		DottedRect(ctx, lunaSnap(paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h)), c.text)
	}
}

// DrawPanel is Tab.Pane: the near-white property-page face (fading to the
// scheme's #f4f3ee over its first 300px) in a 1px frame; raised adds the
// pane's two-line shadow on the right and bottom.
func (lunaEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	if raised && b.Dx() > 8*lw && b.Dy() > 8*lw {
		// Outer light line, then the darker line against the pane.
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2*lw, b.Min.Y+2*lw, b.Dx()-2*lw, b.Dy()-2*lw), paintengine2d.Fill(c.paneShadow2))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+lw, b.Min.Y+lw, b.Dx()-2*lw, b.Dy()-2*lw), paintengine2d.Fill(c.paneShadow1))
		b = paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-2*lw, b.Dy()-2*lw)
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.paneBorder))
	in := b.Inset(lw)
	ctx.DrawRect(in, VGradient(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), l.S(300)), c.pane...))
}

func (lunaEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(lunaColors(l).face))
}

// DrawMenuTitle is an Office menu-bar item: hot = the hot-track gradient in
// a 1px border; open = the drop-down's gradient, framed on three sides so
// it joins the popup below.
func (e lunaEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.gray
	case open:
		hl := paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), b.Dy()-lw)
		ctx.DrawRect(hl, VGradient(hl, c.openGrad...))
		ctx.DrawRect(paintengine2d.XYWH(hl.Min.X, hl.Min.Y, hl.Dx(), lw), paintengine2d.Fill(c.menuBorder))
		ctx.DrawRect(paintengine2d.XYWH(hl.Min.X, hl.Min.Y, lw, hl.Dy()), paintengine2d.Fill(c.menuBorder))
		ctx.DrawRect(paintengine2d.XYWH(hl.Max.X-lw, hl.Min.Y, lw, hl.Dy()), paintengine2d.Fill(c.menuBorder))
		fg = c.openText
	case st.Pressed() || st.Hovered() || st.Focused():
		hl := paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), b.Dy()-2*lw)
		ctx.DrawRect(hl, VGradient(hl, c.selGrad...))
		lunaBorder(ctx, hl, lw, c.hotBorder)
		fg = c.selText2
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
}

// DrawMenuFrame is an Office drop-down: 1px border, light body and the
// icon gutter (Office 2003's horizontal gradient, Office XP's flat tone).
func (lunaEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.menuBorder))
	in := b.Inset(lw)
	ctx.DrawRect(in, paintengine2d.Fill(c.menuBg))
	ch := MenuChromeFor(l)
	gw := snap(b.Min.X+ch.GutterW()) - in.Min.X
	if gw > 2 && gw < in.Dx()*0.55 {
		g := paintengine2d.XYWH(in.Min.X, in.Min.Y, gw, in.Dy())
		ctx.DrawRect(g, HGradient(g, c.gutter...))
	}
}

// DrawMenuItem is an Office menu row: the hot-track across gutter and
// label, checked items boxed in the gutter, separators from the label
// column, shortcuts in the label colour.
func (e lunaEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := lunaColors(l)
	lw := lunaPx(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0 := snap(ch.LabelMinX(b.Min.X))
		x1 := snap(b.Max.X + ch.PadR - 2*lw)
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, lw), paintengine2d.Fill(c.menuSep))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		e.MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
	}
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.menuDisText
	case hot:
		fg = e.MenuTextColor(l, true)
	}
	switch {
	case row.Checked:
		// Office boxes a checked command in the gutter: the icon, else a
		// small tick or bullet, on the check colour in a 1px border.
		gw := ch.CheckCol()
		side := b.Dy() - 4*lw
		if side > gw-4*lw {
			side = gw - 4*lw
		}
		box := lunaSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
		fill, mark := c.menuCheck, c.checkText
		if hot {
			fill, mark = c.menuCheckHot, c.checkHotText
		}
		if st.Disabled() {
			mark = fg
		}
		ctx.DrawRect(box, paintengine2d.Fill(fill))
		lunaBorder(ctx, box, lw, c.hotBorder)
		switch {
		case row.Icon != IconNone && !row.Radio:
			l.drawToolIcon(ctx, box.Inset(2*lw), row.Icon, mark)
		case row.Radio:
			ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
			ctx.DrawCircle(ctr, box.Dx()*0.17, paintengine2d.Fill(mark))
		default:
			PixelTick(ctx, box.Inset(snap(box.Dx()*0.3)), mark)
		}
	case !row.Radio:
		// Unchecked radio items show nothing in Office; icons show as is.
		l.drawMenuGutter(ctx, b, ch, row, fg)
	}
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	labelRight := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = 10
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		drawMenuSubmenuArrow(ctx, ab, fg)
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
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), fg)
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

// DrawProgressBar is ProgressTrack.bmp with ProgressChunk.bmp: 8px chunks
// and 2px gaps in a rounded #686868 trough; indeterminate slides a block
// of three chunks.
func (lunaEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	if b.Dx() < 8*lw || b.Dy() < 6*lw {
		return
	}
	r := l.rx(3)
	in, ri := lunaFrame(ctx, b, r, lw, c.progBorder)
	ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(c.progTrough))
	ctx.Save()
	ctx.ClipRoundRect(in, ri, ri)
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(c.progIn))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, in.Dx(), lw), paintengine2d.Fill(c.progIn2))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.progIn))
	ctx.Restore()
	area := paintengine2d.XYWH(in.Min.X+lw, in.Min.Y+2*lw, in.Dx()-3*lw, in.Dy()-3*lw)
	if area.Empty() {
		return
	}
	cw := snap(area.Dy() * 8 / 12)
	if cw < 2*lw {
		cw = 2 * lw
	}
	gap := snap(l.S(2))
	paint := VGradient(area, c.chunk...)
	if st.Disabled() {
		paint = paint.WithOpacity(0.4)
	}
	chunks := func(x0, x1 float32) {
		for x := x0; x < x1; x += cw + gap {
			w := cw
			if x+w > x1 {
				w = x1 - x
			}
			if w >= lw {
				ctx.DrawRect(paintengine2d.XYWH(x, area.Min.Y, snap(w), area.Dy()), paint)
			}
		}
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		span := (cw + gap) * 3
		x := area.Min.X + snap((area.Dx()+span)*phase) - span
		lo, hi := x, x+span-gap
		if lo < area.Min.X {
			lo = area.Min.X
		}
		if hi > area.Max.X {
			hi = area.Max.X
		}
		if hi > lo {
			ctx.Save()
			ctx.ClipRect(paintengine2d.XYWH(lo, area.Min.Y, hi-lo, area.Dy()))
			chunks(x, x+span)
			ctx.Restore()
		}
		return
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	chunks(area.Min.X, area.Min.X+snap(area.Dx()*t))
}

// DrawSlider is the trackbar: a sunken 4px groove and the pointed thumb of
// TrackbarDown13.bmp — green-tipped, orange when hot, darker when pressed.
func (lunaEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	u := l.S(1)
	tw, th := snap(11*u), snap(21*u)
	if th > b.Dy()-2*lw {
		th = b.Dy() - 2*lw
	}
	if tw < 5*lw || th < 8*lw {
		return
	}
	x0 := b.Min.X + tw*0.5
	x1 := b.Max.X - tw*0.5
	ty := snap(b.Min.Y + (b.Dy()-th)*0.5)
	// Groove: sunken, dark top-left, white bottom-right (sliderTrack.bmp).
	gy := snap(ty + th*0.38 - 2*lw)
	groove := paintengine2d.XYWH(b.Min.X+lw, gy, b.Dx()-2*lw, 4*lw)
	ctx.DrawRect(groove, paintengine2d.Fill(c.grooveLt))
	ctx.DrawRect(paintengine2d.XYWH(groove.Min.X, groove.Min.Y, groove.Dx()-lw, 3*lw), paintengine2d.Fill(c.grooveDk))
	ctx.DrawRect(paintengine2d.XYWH(groove.Min.X+lw, groove.Min.Y+lw, groove.Dx()-2*lw, 2*lw), paintengine2d.Fill(c.groove))
	// Thumb: a rounded block over a point, in the thumb frame colours.
	tx := snap(x0 + (x1-x0)*t - tw*0.5)
	pt := snap(6 * u)
	if pt > th*0.4 {
		pt = snap(th * 0.4)
	}
	shape := func(r paintengine2d.Rect, p float32) *paintengine2d.Path {
		path := paintengine2d.NewPath()
		k := lw
		path.MoveTo(r.Min.X+k, r.Min.Y)
		path.LineTo(r.Max.X-k, r.Min.Y)
		path.LineTo(r.Max.X, r.Min.Y+k)
		path.LineTo(r.Max.X, r.Max.Y-p)
		path.LineTo((r.Min.X+r.Max.X)*0.5, r.Max.Y)
		path.LineTo(r.Min.X, r.Max.Y-p)
		path.LineTo(r.Min.X, r.Min.Y+k)
		path.Close()
		return path
	}
	outer := paintengine2d.XYWH(tx, ty, tw, th)
	acc := c.thumbAcc[0]
	border, faceStops := c.thumbBorder, c.thumbFace
	switch {
	case st.Disabled():
		acc = c.thumbAcc[3]
		border = c.thumbDisBorder
		faceStops = c.thumbDisGrad
	case st.Pressed():
		acc = c.thumbAcc[2]
	case st.Hovered() || st.Focused():
		acc = c.thumbAcc[1]
	}
	ctx.DrawPath(shape(outer, pt), paintengine2d.Fill(border))
	if !st.Disabled() {
		// The frame darkens toward the bottom right: the point's lower-left
		// edge is mid-grey, the right side and lower-right edge darkest.
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(outer.Min.X, outer.Max.Y-pt, outer.Dx()*0.5, pt))
		ctx.DrawPath(shape(outer, pt), paintengine2d.Fill(c.thumbBorderLo))
		ctx.Restore()
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(outer.Min.X+outer.Dx()*0.5, outer.Min.Y, outer.Dx()*0.5, outer.Dy()))
		ctx.DrawPath(shape(outer, pt), paintengine2d.Fill(c.thumbBorderDk))
		ctx.Restore()
	}
	in := paintengine2d.XYWH(tx+lw, ty+lw, tw-2*lw, th-2*lw)
	inner := shape(in, pt-lw*0.5)
	ctx.DrawPath(inner, HGradient(in, faceStops...))
	ctx.Save()
	ctx.ClipPath(inner)
	for i := 0; i < 3; i++ {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+float32(i)*lw, in.Dx(), lw), paintengine2d.Fill(acc[i]))
	}
	// Tinted point edges.
	tip := paintengine2d.Pt((in.Min.X+in.Max.X)*0.5, in.Max.Y)
	edgeL := paintengine2d.StrokePaint(acc[2], 1.4*lw)
	edgeR := paintengine2d.StrokePaint(acc[3], 1.4*lw)
	ctx.DrawLine(paintengine2d.Pt(in.Min.X, in.Max.Y-pt+lw*0.5), tip, edgeL)
	ctx.DrawLine(paintengine2d.Pt(in.Max.X, in.Max.Y-pt+lw*0.5), tip, edgeR)
	ctx.Restore()
	if st.Focused() {
		DottedRect(ctx, b, c.text)
	}
}

// DrawSwitch has no XP original: a field-framed slot that fills with the
// progress green when on, and a push-button knob.
func (e lunaEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := lunaColors(l)
	lw := lunaPx(l)
	tw, th := l.metrics.SwitchW, l.metrics.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	track := lunaSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	r := l.rx(3)
	border := c.fieldBorder
	if st.Disabled() {
		border = c.fieldDisBorder
	}
	in, ri := lunaFrame(ctx, track, r, lw, border)
	switch {
	case on && !st.Disabled():
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.chunk...))
	case st.Disabled():
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(c.fieldDis))
	default:
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(c.field))
	}
	kw := snap(track.Dy() * 0.9)
	kx := track.Min.X
	if on {
		kx = track.Max.X - kw
	}
	c.pushButton(l, ctx, paintengine2d.XYWH(kx, track.Min.Y, kw, track.Dy()), st&^StateFocused)
	if label == "" {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		DottedRect(ctx, labelFocusRect(l.body, label, lb, b), c.text)
	}
}

// selFor is the selection fill and text of a row: the highlight colour,
// or the button face with plain text when the view is unfocused (XP's
// list and tree views grey an inactive selection).
func (c *lunaSet) selFor(st ControlState) (fill, fg paintengine2d.Color) {
	if st.Inactive() {
		return c.face, c.text
	}
	return c.sel, c.selText
}

// ItemFocus is the dotted focus rectangle XP list and tree views kept.
func (lunaEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := lunaColors(l)
	col := c.text
	if st.Checked() && !st.Inactive() {
		col = c.selText
	}
	DottedRect(ctx, lunaSnap(b), col)
}

// DrawListRow is a ListView row: XP highlighted the selection in the
// scheme's highlight colour and did not hot-track.
func (lunaEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := lunaColors(l)
	fg := l.fieldText()
	if st.Checked() {
		var fill paintengine2d.Color
		fill, fg = c.selFor(st)
		ctx.DrawRect(lunaSnap(b), paintengine2d.Fill(fill))
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		lunaEngine{}.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a TreeView row: dotted connectors, the +/- box and the
// selection behind the label only.
func (e lunaEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	selected := st.Checked()
	c := lunaColors(l)
	lw := lunaPx(l)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(6)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	dot := paintengine2d.Fill(c.treeLine)
	ctx.Save()
	ctx.ClipRect(b)
	for d := 0; d <= depth; d++ {
		gx := snap(b.Min.X + pad + float32(d)*indent + indent*0.5)
		y1 := b.Max.Y
		if d == depth {
			y1 = cy
		}
		for y := snap(b.Min.Y); y < y1; y += 2 * lw {
			ctx.DrawRect(paintengine2d.XYWH(gx, y, lw, lw), dot)
		}
		if d == depth {
			for xx := gx; xx < x+indent+l.S(2); xx += 2 * lw {
				ctx.DrawRect(paintengine2d.XYWH(xx, cy, lw, lw), dot)
			}
		}
	}
	ctx.Restore()
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.expSign)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	lx := x + indent + l.S(4)
	fg := l.fieldText()
	lb := lunaSnap(paintengine2d.XYWH(lx-l.S(2), b.Min.Y+lw, f.Advance(label)+l.S(4), b.Dy()-2*lw)).Intersect(b)
	if selected {
		var fill paintengine2d.Color
		fill, fg = c.selFor(st)
		ctx.DrawRect(lb, paintengine2d.Fill(fill))
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, lb, st)
	}
}

// DrawTableHeader is ListViewHeader.bmp: a flat face over a three-line
// shade, divided by an etched line; hot trades the shade for three orange
// lines, pressed sinks.
func (e lunaEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	pressed := st.Pressed() && !st.Disabled()
	hot := st.Hovered() && !st.Disabled() && !pressed
	lb := b
	if pressed {
		ctx.DrawRect(b, paintengine2d.Fill(c.hdrPressFace))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.hdrPressShade))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), lw), paintengine2d.Fill(Mix(c.hdrPressShade, c.hdrPressFace, 0.5)))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(c.hdrPressDk))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.hdrPressDk))
		lb = lb.Translate(paintengine2d.Pt(lw, lw))
	} else {
		face, low := c.hdrFace, c.hdrLow
		if hot {
			face, low = c.hdrHotFace, c.hdrHot
		}
		ctx.DrawRect(b, paintengine2d.Fill(face))
		for i := 0; i < 3; i++ {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-float32(3-i)*lw, b.Dx(), lw), paintengine2d.Fill(low[i]))
		}
		if !hot {
			y0 := b.Min.Y + snap(b.Dy()*3/18)
			y1 := b.Max.Y - snap(b.Dy()*5/18)
			ctx.DrawRect(paintengine2d.XYWH(b.Max.X-2*lw, y0, lw, y1-y0), paintengine2d.Fill(c.hdrDivDk))
			ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, y0, lw, y1-y0), paintengine2d.Fill(c.hdrDivLt))
		}
	}
	aw := float32(0)
	if sorted {
		aw = l.S(14)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		ab := paintengine2d.XYWH(lb.Max.X-aw-l.S(4), lb.Min.Y, aw, lb.Dy()-3*lw)
		FillArrow(ctx, paintengine2d.XYWH(ab.Min.X+(ab.Dx()-l.S(8))*0.5, ab.Min.Y+(ab.Dy()-l.S(8))*0.5, l.S(8), l.S(8)), dir, c.hdrArrow)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10)-aw, lb.Dy()-2*lw), fg, AlignStart, 0)
}

func (lunaEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := lunaColors(l)
	fg := l.fieldText()
	if st.Checked() {
		var fill paintengine2d.Color
		fill, fg = c.selFor(st)
		ctx.DrawRect(lunaSnap(b), paintengine2d.Fill(fill))
	}
	f := l.faceOrBody(face)
	pad := tableCellPad(b.Dx(), f.Advance(label))
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

// DrawToolBar is an Office command bar: the vertical band gradient with
// rounded ends, its shadow line and the dotted grip at the left.
// ToolBarInsets keeps the rebar grip clear of the first button.
func (lunaEngine) ToolBarInsets(l *Classic) Insets { return Insets{Left: l.S(11), Right: l.S(6)} }

func (lunaEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	if b.Dx() < 8*lw || b.Dy() < 8*lw {
		return
	}
	r := l.rx(3)
	band := paintengine2d.XYWH(b.Min.X+lw, b.Min.Y+lw, b.Dx()-2*lw, b.Dy()-2*lw)
	ctx.DrawRoundRect(band, r, r, paintengine2d.Fill(c.tbShadow))
	body := paintengine2d.XYWH(band.Min.X, band.Min.Y, band.Dx()-lw, band.Dy()-lw)
	ctx.DrawRoundRect(body, r, r, VGradient(body, c.tb...))
	// Grip: a column of dots, each a dark pixel pair with a light shadow.
	d := 2 * lw
	gx := snap(body.Min.X + l.S(2))
	for y := snap(body.Min.Y + l.S(4)); y+d <= body.Max.Y-l.S(4); y += 2 * d {
		ctx.DrawRect(paintengine2d.XYWH(gx+lw, y+lw, d, d), paintengine2d.Fill(c.tbGripLt))
		ctx.DrawRect(paintengine2d.XYWH(gx, y, d, d), paintengine2d.Fill(c.tbGripDk))
	}
}

func (e lunaEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := lunaColors(l)
	fg := c.text
	if st.Toggle() || st.Hovered() || st.Pressed() || st.Focused() {
		fg = c.officeButton(l, ctx, b.Inset(1), st)
	}
	font := l.body
	if st.Disabled() {
		fg = c.disText
	}
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
		right := b.Max.X - pad
		if right < x {
			right = x
		}
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
		font.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-font.Height())*0.5), fg)
		ctx.Restore()
	}
}

// DrawStatusBar is StatusBackground.bmp: a dark top line over a soft
// gradient, panes split by etched lines, and the dotted size grip.
func (lunaEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	ctx.DrawRect(b, VGradient(b, c.status...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.statTop))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), lw), paintengine2d.Fill(c.stat1))
	grip := l.S(16)
	if len(parts) > 0 {
		slot := (b.Dx() - grip) / float32(len(parts))
		for i, s := range parts {
			x := b.Min.X + slot*float32(i)
			if i > 0 {
				sx := snap(x)
				y0 := b.Min.Y + 2*lw + snap(l.S(2))
				h := b.Max.Y - y0 - snap(l.S(3))
				ctx.DrawRect(paintengine2d.XYWH(sx-2*lw, y0, lw, h), paintengine2d.Fill(c.statSepDk))
				ctx.DrawRect(paintengine2d.XYWH(sx-lw, y0, lw, h), paintengine2d.Fill(c.statSepLt))
			}
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(6), b.Min.Y+2*lw, slot-l.S(10), b.Dy()-2*lw), c.text, AlignStart, 0)
		}
	}
	// Size grip: dots in a triangle (ResizeGrip2.bmp).
	d := 2 * lw
	step := snap(l.S(4))
	for row := 0; row < 3; row++ {
		for k := 0; k <= row; k++ {
			x := b.Max.X - float32(k+1)*step - lw
			y := b.Max.Y - float32(3-row)*step + lw
			ctx.DrawRect(paintengine2d.XYWH(x+lw, y+lw, d, d), paintengine2d.Fill(c.gripDotLt))
			ctx.DrawRect(paintengine2d.XYWH(x, y, d, d), paintengine2d.Fill(c.gripDot))
		}
	}
}

// DrawTitleBar is an Explorer-bar heading: white fading into the scheme's
// light tone, the title bold in the scheme's heading colour.
func (lunaEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	ctx.DrawRect(b, HGradient(b, c.xbHead...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.xbHead[1].Color))
	x := b.Min.X + l.metrics.Pad
	f := l.bold
	if f == nil {
		f = l.body
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), c.xbText, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), l.palette.TextMuted, AlignStart, 0)
	}
}

// DrawAccordionHeader is an Explorer-bar group head (NormalGroupHead.bmp):
// rounded top corners, white-to-blue band, bold title and the round
// double-chevron button at the right.
func (lunaEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := lunaColors(l)
	b = lunaSnap(b)
	lw := lunaPx(l)
	r := l.rx(3)
	ctx.DrawPath(RoundRectPath(b, r, r, 0, 0), HGradient(b, c.xbHead...))
	d := snap(b.Dy() - l.S(8))
	if m := snap(l.S(19)); d > m {
		d = m
	}
	btn := paintengine2d.XYWH(b.Max.X-d-l.S(4), snap(b.Min.Y+(b.Dy()-d)*0.5), d, d)
	ctr := paintengine2d.Pt((btn.Min.X+btn.Max.X)*0.5, (btn.Min.Y+btn.Max.Y)*0.5)
	ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(c.xbBtnRing))
	ctx.DrawCircle(ctr, d*0.5-lw, paintengine2d.Radial(paintengine2d.RadialGradient{
		Center: paintengine2d.Pt(ctr.X-d*0.15, ctr.Y-d*0.15), Radius: d * 0.6, Stops: c.xbBtn,
	}))
	fg := c.xbText
	if st.Hovered() && !st.Disabled() {
		fg = c.xbHotText
	}
	dir := DirDown
	if expanded {
		dir = DirUp
	}
	u := d / 19
	for k := 0; k < 2; k++ {
		off := (float32(k)*4 - 2) * u
		if dir == DirDown {
			off = -off
		}
		g := paintengine2d.XYWH(btn.Min.X, btn.Min.Y-off, btn.Dx(), btn.Dy())
		lunaChevron(ctx, g, dir, u*0.8, fg)
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	tb := paintengine2d.XYWH(b.Min.X+l.S(10), b.Min.Y, btn.Min.X-b.Min.X-l.S(14), b.Dy())
	l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
	if st.Focused() {
		DottedRect(ctx, labelFocusRect(f, title, tb, b), fg)
	}
}

// DrawSeparator is the dialog's etched line (SS_ETCHEDHORZ / VERT):
// button shadow over button highlight. Tool bars draw their own dividers
// in the command-bar colour (Palette.Divider).
func (lunaEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := lunaColors(l)
	lw := lunaPx(l)
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - lw
		y0, h := snap(b.Min.Y+l.S(2)), snap(b.Dy()-l.S(4))
		ctx.DrawRect(paintengine2d.XYWH(x, y0, lw, h), paintengine2d.Fill(c.shadow))
		ctx.DrawRect(paintengine2d.XYWH(x+lw, y0, lw, h), paintengine2d.Fill(c.hi))
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - lw
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), lw), paintengine2d.Fill(c.shadow))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+lw, b.Dx(), lw), paintengine2d.Fill(c.hi))
}

func (lunaEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	ctx.DrawRect(b, paintengine2d.Fill(lunaColors(l).face))
}

// DrawTooltip is the XP tooltip: #ffffe1 in a 1px black frame.
func (lunaEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := lunaColors(l)
	b = lunaSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.info))
	lunaBorder(ctx, b, lunaPx(l), c.infoBorder)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(4)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.infoText, AlignStart, 0)
}

// DrawMessageIcon paints the XP message-box icons: a red disc with a white
// ×, a yellow warning triangle, and the blue "i" / "?" discs.
func (lunaEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	if icon == IconNone {
		return
	}
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	ctr := paintengine2d.Pt(cx, cy)
	r := s * 0.47
	lw := lunaPx(l)
	disc := func(stops []paintengine2d.GradientStop, rim paintengine2d.Color) {
		ctx.DrawCircle(ctr, r, paintengine2d.Fill(rim))
		ctx.DrawCircle(ctr, r-lw, paintengine2d.Radial(paintengine2d.RadialGradient{
			Center: paintengine2d.Pt(cx-r*0.35, cy-r*0.4), Radius: r * 1.5, Stops: stops,
		}))
	}
	glyph := func(g string, col paintengine2d.Color, dy float32) {
		f := l.bold
		if f == nil {
			f = l.body
		}
		w := f.Advance(g)
		f.Draw(ctx, g, paintengine2d.Pt(cx-w*0.5, cy-f.Height()*0.5+dy), col)
	}
	switch icon {
	case IconError:
		disc(lunaErrorDisc, Hex("#8e1400"))
		lunaCross(ctx, paintengine2d.XYWH(cx-r*0.42, cy-r*0.42, r*0.84, r*0.84), Hex("#ffffff"), r*0.24)
	case IconWarning:
		tri := paintengine2d.NewPath()
		tri.MoveTo(cx, cy-r)
		tri.LineTo(cx+r, cy+r*0.8)
		tri.LineTo(cx-r, cy+r*0.8)
		tri.Close()
		ctx.DrawPath(tri, paintengine2d.Fill(Hex("#9c6a00")))
		in := paintengine2d.NewPath()
		k := lw * 1.6
		in.MoveTo(cx, cy-r+k*1.7)
		in.LineTo(cx+r-k*1.7, cy+r*0.8-k)
		in.LineTo(cx-r+k*1.7, cy+r*0.8-k)
		in.Close()
		ctx.DrawPath(in, VGradient(paintengine2d.XYWH(cx-r, cy-r, r*2, r*1.8), lunaWarnFace...))
		glyph("!", Hex("#000000"), r*0.2)
	case IconInfo:
		disc(lunaInfoDisc, Hex("#0b3a95"))
		glyph("i", Hex("#ffffff"), 0)
	case IconQuestion:
		disc(lunaInfoDisc, Hex("#0b3a95"))
		glyph("?", Hex("#ffffff"), 0)
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
	}
}

// ---- packs --------------------------------------------------------------------------

func lunaPack(name, label string, year int, summary string, fam ThemeName, scheme int, pal Palette) ThemePack {
	tok := ThemeTokens{
		Engine:  "luna",
		Bevel:   BevelLunaHottrack,
		Family:  fam,
		Era:     EraLuna,
		Palette: pal,
		Params:  map[string]float32{"scheme": float32(scheme)},
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: pal.AccentPress, Border: pal.MenuHoverBorder}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.12), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Windows", Summary: summary,
		Era: EraLuna, Palette: fam, Tokens: tok,
	}
}

// lunaPal is the shared palette of a scheme: the window face, text,
// highlight, the menu hot-track (so every widget sees the menu colours the
// engine paints) and the command-bar separator as the divider.
func lunaPal(face, shadow, text, muted, sel, selText, field, fieldBorder, border, divider, hot, hotBorder, gutter string) Palette {
	f := hexColor(face)
	return Palette{
		Background: f, Surface: f, SurfaceAlt: f,
		Border: hexColor(border), Divider: hexColor(divider),
		Text: hexColor(text), TextMuted: hexColor(muted), TextOnAccent: hexColor(selText),
		Accent: hexColor(sel), AccentHover: Shade(hexColor(sel), 0.15), AccentPress: Shade(hexColor(sel), -0.2),
		Field: hexColor(field), FieldBorder: hexColor(fieldBorder),
		Focus: hexColor(text), Selection: hexColor(sel),
		Track: Mix(f, hexColor(field), 0.5), Thumb: Shade(hexColor(sel), 0.6),
		Highlight: hexColor("#ffffff").WithAlpha(0.55), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: hexColor(hot), MenuHoverBorder: hexColor(hotBorder), MenuGutter: hexColor(gutter),
		Danger: hexColor("#c32a14"), Success: hexColor("#218a21"), Warning: hexColor("#a86a00"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.35),
		BevelLight: hexColor("#ffffff"), BevelDark: hexColor(shadow),
	}
}

func lunaPacks() []ThemePack {
	blue := lunaPal("#ece9d8", "#aca899", "#000000", "#716f64", "#316ac5", "#ffffff", "#ffffff", "#7f9db9", "#919b9c", "#6a8ccb", "#ffeec2", "#000080", "#cbe1fc")
	olive := lunaPal("#ece9d8", "#aca899", "#000000", "#716f64", "#93a070", "#ffffff", "#ffffff", "#a4b97f", "#919b9c", "#608058", "#ffeec2", "#3f5d38", "#cedca7")
	silver := lunaPal("#e0dfe3", "#9d9da1", "#000000", "#6b6a70", "#b2b4bf", "#000000", "#ffffff", "#a5acb2", "#919b9c", "#6e6d8f", "#ffeec2", "#4b4b6f", "#e1e2ec")
	royale := lunaPal("#ebe9ed", "#a7a6aa", "#000000", "#6e6d73", "#335ea8", "#ffffff", "#ffffff", "#7f9db9", "#879bb3", "#c1c1c4", "#c2cfe5", "#335ea8", "#eeedf0")
	noir := lunaPal("#313843", "#11151b", "#e6e8ec", "#9ba3ae", "#5b83bd", "#ffffff", "#1f242c", "#5d6877", "#0d1015", "#4a525e", "#3c5a84", "#6f95cf", "#353f4d")
	noir.Danger, noir.Success, noir.Warning = hexColor("#ff7a6a"), hexColor("#5fd35d"), hexColor("#f0b030")
	noir.Highlight, noir.Shadow = hexColor("#ffffff").WithAlpha(0.12), paintengine2d.RGBA(0, 0, 0, 0.5)
	noir.BevelLight, noir.Overlay = hexColor("#5c6572"), paintengine2d.RGBA(0, 0, 0, 0.5)
	return []ThemePack{
		lunaPack("luna", "Luna Blue", 2001, "Windows XP's default: blue captions, beige 3D face, orange hot glow, Office 2003 menus.", ThemeLight, 0, blue),
		lunaPack("luna-olive", "Luna Olive Green", 2001, "XP's HomeStead scheme: olive captions, green-framed buttons, copper progress.", ThemeLight, 1, olive),
		lunaPack("luna-silver", "Luna Silver", 2001, "XP's Metallic scheme: silver captions with black titles, lavender-grey chrome.", ThemeLight, 2, silver),
		lunaPack("luna-royale", "Royale", 2004, "Media Center's glossy Energy Blue captions, steel buttons and Office XP menus.", ThemeLight, 3, royale),
		lunaPack("luna-night", "Royale Noir", 2005, "Royale's black glossy variant, as a dark scheme with Office XP-style menus.", ThemeDark, 4, noir),
	}
}
