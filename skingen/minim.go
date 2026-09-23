package skingen

import "github.com/codemodify/paintengine2d"

// Minim is the pixel skin the compact player wears: a hi-fi front panel in
// graphite, with a phosphor-green display sunk into it.
//
// Cassette already exercises the pixelated path, so Minim is not here to
// prove that again — it is here because the compact player is 275 by 116
// design pixels and every one of them has to earn its place. Where Cassette
// is six flat colours and two-pixel bevels, Minim is one-pixel bevels with
// the four corner pixels of every face knocked out, which is how a front
// panel of this kind was drawn when a control was eighteen pixels tall and
// a rounded corner cost four of them.
//
// The other half of it is the shape. The window's bottom corners step in
// twice and the bottom band steps in again, so the compact player stands on
// a chin rather than on a rectangle — stated as three stretching rects, so
// it is the same outline at 1, 1.25, 1.75 and 2 rather than a resampled
// picture of one, and correct however the window is sized.
//
// It is a deliberately complete skin, unlike Deck: a compact player is
// mostly *not* buttons — it is sliders, a list, a scroll bar and a display —
// and a skin that left those to its base pack would be a picture of a
// player with somebody else's controls in it.

// Minim's palette: a panel, a phosphor and one warm light.
const (
	minInk    = "#080c0a" // the outline round every face
	minShell  = "#262e2b" // the panel itself
	minShellH = "#3e4a45"
	minShellL = "#161c1a"

	minFace  = "#313c38" // a resting face
	minFaceH = "#4c5a54"
	minFaceL = "#1c2421"
	minHot   = "#3d4a45"
	minDown  = "#1f2825"
	minOff   = "#272f2c"

	minWellC = "#04140c" // the display well
	minWellR = "#16362a"

	minPhos = "#63f0a4" // the phosphor: readings, fills, selection
	minPhoL = "#2a9160"
	minPhoD = "#123f2a"

	minAmber = "#f0b455" // the warm light: what is switched on
	minAmbLo = "#a8752c"

	minText = "#d6e8dd"
	minDim  = "#87a294"
	minGone = "#4f615a"
)

// Minim's cell grid. Small numbers on purpose: this is art drawn at 1×, and
// a one-pixel bevel is one pixel.
const (
	minW = 30 // a face cell
	minH = 18
)

// minSlice keeps the outline, the bevel and one pixel of face at each edge,
// which is exactly the three pixels a chamfered corner is made of.
var minSlice = [4]int{3, 3, 3, 3}

// The frame, in design pixels. The chin is what the silhouette cuts the
// window down to at the bottom, in two steps; the border is wider than the
// deepest step so the content never runs into the cut.
const (
	minCaption = 18
	minChin    = 16 // the depth of the bottom band that steps in
	minStep    = 8  // the first step's width
	minWaist   = 24 // the second step's width
)

// Minim builds the skin's plan.
func Minim() *Plan {
	sh := &Sheet{Name: "chrome", Pixelated: true}
	l := &lay{sh: sh, gap: 2, faceW: minW, faceH: minH, faceSlice: minSlice}
	p := &Plan{
		Name:    "minim",
		Label:   "Minim",
		Year:    2026,
		Lineage: "uitoolkit",
		Summary: "A pixel front panel with a phosphor display, and a window that stands on a stepped chin — the skin the compact player demo wears.",
		Family:  "dark",
		Base:    "win95",
		Sheets:  []*Sheet{sh},
	}

	// ---- faces ------------------------------------------------------------
	//
	// Out and in are the same two colours the other way round, which is the
	// whole grammar of a bevelled panel; the corners are empty in both,
	// which is the whole grammar of this one.
	out := func(fill string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			minPlate(ctx, w, h, Hex(fill), Hex(minFaceH), Hex(minFaceL))
		}
	}
	in := func(fill string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			minPlate(ctx, w, h, Hex(fill), Hex(minFaceL), Hex(minFaceH))
		}
	}

	l.row(minH)
	l.face("button.normal", out(minFace))
	l.face("button.hover", out(minHot))
	l.face("button.pressed", in(minDown))
	l.face("button.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minOff), Hex("#3a443f"), Hex("#1e2623"))
	})
	l.face("button.focus", func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minFace), Hex(minFaceH), Hex(minFaceL))
		minDots(ctx, 3, 3, w-6, h-6, Hex(minPhos))
	})
	l.face("button.default", func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minPhoL), Hex(minPhos), Hex(minPhoD))
	})
	l.face("button.checked", func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minPhoD), Hex(minPhoL), Hex("#0a1f15"))
	})

	l.row(minH)
	l.face("field.normal", func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex(minWellC), Hex(minWellR))
	})
	l.face("field.focus", func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex(minWellC), Hex(minPhos))
	})
	l.face("field.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex("#141a18"), Hex("#2c3833"))
	})
	l.face("combo.normal", out(minFace))
	l.face("combo.hover", out(minHot))
	l.face("combo.pressed", in(minDown))

	// A tool button is nothing at rest — the transport row of a compact
	// player is a strip of glyphs on the panel, not a row of boxes — and
	// grows a face only under the pointer.
	l.row(minH)
	l.face("tool.normal", func(ctx *paintengine2d.Context, w, h float32) {})
	l.face("tool.hover", func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minHot), Hex(minFaceH), Hex(minFaceL))
	})
	l.face("tool.pressed", in(minDown))
	// A tool that is on is sunk with a phosphor rule under it rather than
	// lit up: the glyph on it is drawn in the accent by the app, and an
	// accent glyph on an accent face is a glyph nobody can read.
	l.face("tool.checked", func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex("#1b2320"), Hex(minFaceL), Hex(minFaceH))
		px(ctx, 2, h-3, w-4, 1, Hex(minPhos))
	})
	l.face("row.hover", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex("#1d2622"))
	})
	l.face("row.checked", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex(minPhoD))
		px(ctx, 0, 0, 2, h, Hex(minPhos))
	})

	l.row(minH)
	l.face("tab.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 1, 2, w-2, h-3, Hex("#28322e"))
		px(ctx, 1, 2, w-2, 1, Hex(minFaceH))
		px(ctx, 0, h-1, w, 1, Hex(minInk))
	})
	l.face("tab.hover", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 1, 2, w-2, h-3, Hex(minHot))
		px(ctx, 1, 2, w-2, 1, Hex(minFaceH))
		px(ctx, 0, h-1, w, 1, Hex(minInk))
	})
	l.face("tab.checked", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 1, 0, w-2, h, Hex(minFace))
		px(ctx, 1, 0, w-2, 2, Hex(minPhos))
		px(ctx, 1, 2, 1, h-2, Hex(minFaceH))
		px(ctx, w-2, 2, 1, h-2, Hex(minFaceL))
	})
	l.face("menu.hover", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex(minPhoD))
		px(ctx, 0, 0, w, 1, Hex(minPhoL))
	})
	l.face("panel.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex("#222a27"))
		bevel(ctx, w, h, 1, Hex(minShellH), Hex(minShellL))
	})
	l.face("bar.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex(minShell))
		px(ctx, 0, 0, w, 1, Hex(minShellH))
		px(ctx, 0, h-1, w, 1, Hex(minInk))
	})

	l.row(minH)
	l.face("menu.frame", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex(minShell))
		bevel(ctx, w, h, 1, Hex(minShellH), Hex(minInk))
	})
	l.face("tooltip.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex(minWellC))
		bevel(ctx, w, h, 1, Hex(minPhoL), Hex(minInk))
	})
	// Orientation-neutral, like the other two pixel skins: Face is not told
	// which way a splitter runs, so the grip is a cluster and not a line.
	l.face("splitter.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, Hex(minShell))
		for _, d := range [][2]float32{{-2, -2}, {1, -2}, {-2, 1}, {1, 1}} {
			px(ctx, w/2+d[0], h/2+d[1], 1, 1, Hex(minShellH))
		}
	})

	// ---- the frame --------------------------------------------------------
	//
	// The panel is a two-colour dither, which is what a machine with a
	// palette and no gradient did; it tiles, so the pattern stays on the
	// grid however big the window is.
	l.row(16)
	l.cell("window.normal", 16, 16, [4]int{}, "tile", false, func(ctx *paintengine2d.Context, w, h float32) {
		minDither(ctx, w, h, Hex(minShell), Hex("#2b3430"))
	})
	l.cell("caption.normal", 48, minCaption, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minBand(ctx, w, h, Hex("#1d2622"), Hex(minPhos))
	})
	l.cell("caption.inactive", 48, minCaption, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minBand(ctx, w, h, Hex("#1d2622"), paintengine2d.Color{})
	})
	l.cell("capbtn.normal", 14, 12, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minFace), Hex(minFaceH), Hex(minFaceL))
	})
	l.cell("capbtn.hover", 14, 12, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minHot), Hex("#5a6a63"), Hex(minFaceL))
	})
	l.cell("capbtn.pressed", 14, 12, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minDown), Hex(minFaceL), Hex(minFaceH))
	})

	// ---- the small parts --------------------------------------------------

	l.row(14)
	l.cell("thumb.normal", 20, 14, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minFace), Hex(minFaceH), Hex(minFaceL))
	})
	l.cell("thumb.hover", 20, 14, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minPlate(ctx, w, h, Hex(minHot), Hex("#5a6a63"), Hex(minFaceL))
	})
	l.cell("track.normal", 20, 14, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex("#151c19"), Hex("#2e3a35"))
	})
	// A slider's track is the slot a compact player's seek bar ran in: two
	// pixels deep with the phosphor showing through behind it.
	l.cell("slot.normal", 20, 8, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex(minWellC), Hex("#2e3a35"))
	})
	l.cell("slot.fill", 20, 8, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 1, 1, w-2, h-2, Hex(minPhoL))
		px(ctx, 1, 1, w-2, 1, Hex(minPhos))
		px(ctx, 1, h-2, w-2, 1, Hex(minPhoD))
	})
	l.cell("knob.normal", 10, 16, [4]int{}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minKnob(ctx, w, h, Hex("#c9d8d0"), Hex("#ffffff"), Hex("#5b6a63"))
	})
	l.cell("knob.disabled", 10, 16, [4]int{}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minKnob(ctx, w, h, Hex("#53605a"), Hex("#6f7d76"), Hex("#2c3632"))
	})

	l.row(16)
	l.cell("check.normal", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex(minWellC), Hex("#2e3a35"))
	})
	l.cell("check.hover", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex(minWellC), Hex(minPhos))
	})
	l.cell("check.checked", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex(minPhoD), Hex(minPhos))
	})
	l.cell("check.disabled", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex("#141a18"), Hex("#2c3833"))
	})
	l.glyph("mark.check", 16, func(ctx *paintengine2d.Context, w, h float32) {
		// Placed by hand: a stroked path at sixteen pixels is mush, and the
		// tick of this era was eight pixels and a diagonal.
		c := paintengine2d.RGB(1, 1, 1)
		for _, q := range [][2]float32{{3, 8}, {4, 9}, {5, 10}, {6, 9}, {7, 8}, {8, 7}, {9, 6}, {10, 5}} {
			px(ctx, q[0], q[1], 2, 2, c)
		}
	})
	l.glyph("mark.radio", 16, func(ctx *paintengine2d.Context, w, h float32) {
		c := paintengine2d.RGB(1, 1, 1)
		px(ctx, 6, 5, 4, 6, c)
		px(ctx, 5, 6, 6, 4, c)
	})

	l.row(16)
	for _, d := range []string{"down", "up", "left", "right"} {
		dir := d
		l.glyph("arrow."+dir, 16, func(ctx *paintengine2d.Context, w, h float32) {
			minArrow(ctx, dir, paintengine2d.RGB(1, 1, 1))
		})
	}
	l.glyph("expander.open", 16, func(ctx *paintengine2d.Context, w, h float32) {
		minArrow(ctx, "down", paintengine2d.RGB(1, 1, 1))
	})
	l.glyph("expander.shut", 16, func(ctx *paintengine2d.Context, w, h float32) {
		minArrow(ctx, "right", paintengine2d.RGB(1, 1, 1))
	})
	// A skin may re-draw the focus ring and may never remove it. This one is
	// the dotted rectangle every machine with a keyboard had, in phosphor.
	l.cell("focus.ring", 16, 16, [4]int{4, 4, 4, 4}, "none", false, func(ctx *paintengine2d.Context, w, h float32) {
		minDots(ctx, 0, 0, w, h, Hex(minPhos))
	})

	l.row(16)
	l.cell("switch.off", 26, 16, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex(minWellC), Hex("#2e3a35"))
	})
	l.cell("switch.on", 26, 16, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 1, 1, w-2, h-2, Hex(minPhoL))
		bevel(ctx, w, h, 1, Hex(minPhoD), Hex(minPhos))
	})
	l.cell("switch.disabled", 26, 16, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		minWell(ctx, w, h, Hex("#141a18"), Hex("#2c3833"))
	})
	l.close()

	// ---- bindings ---------------------------------------------------------

	p.Text = []TextRole{
		{Name: "control", Color: minText, Hover: "#ffffff", Pressed: minText,
			Disabled: minGone, Checked: minPhos, Default: "#07170f"},
		{Name: "onLight", Color: minInk, Disabled: minGone},
		{Name: "caption", Color: minPhos, Disabled: minDim, Bold: true},
	}
	p.Parts = []PartBinding{
		{Part: "button", Text: "control", States: [][2]string{
			{"normal", "button.normal"}, {"hover", "button.hover"},
			{"pressed", "button.pressed"}, {"disabled", "button.disabled"},
			{"focus", "button.focus"}, {"checked", "button.checked"},
			{"default", "button.default"},
		}},
		{Part: "tool", Text: "control", States: [][2]string{
			{"normal", "tool.normal"}, {"hover", "tool.hover"},
			{"pressed", "tool.pressed"}, {"checked", "tool.checked"},
		}},
		{Part: "field", Text: "control", States: [][2]string{
			{"normal", "field.normal"}, {"focus", "field.focus"}, {"disabled", "field.disabled"},
		}},
		{Part: "combo", Text: "control", States: [][2]string{
			{"normal", "combo.normal"}, {"hover", "combo.hover"}, {"pressed", "combo.pressed"},
		}},
		{Part: "tab", Text: "control", States: [][2]string{
			{"normal", "tab.normal"}, {"hover", "tab.hover"}, {"checked", "tab.checked"},
		}},
		{Part: "row", Text: "control", States: [][2]string{
			{"normal", "row.hover"}, {"hover", "row.hover"}, {"checked", "row.checked"},
		}},
		{Part: "menu", Text: "control", States: [][2]string{
			{"normal", "menu.hover"}, {"hover", "menu.hover"},
		}},
		{Part: "panel", States: [][2]string{{"normal", "panel.normal"}}},
		{Part: "bar", States: [][2]string{{"normal", "bar.normal"}}},
		{Part: "splitter", States: [][2]string{{"normal", "splitter.normal"}}},
		{Part: "menu.frame", States: [][2]string{{"normal", "menu.frame"}}},
		{Part: "tooltip", Text: "control", States: [][2]string{{"normal", "tooltip.normal"}}},
		{Part: "window", States: [][2]string{{"normal", "window.normal"}}},
		{Part: "caption", Text: "caption", States: [][2]string{
			{"normal", "caption.normal"}, {"inactive", "caption.inactive"},
		}},
		{Part: "caption.button", Text: "control", States: [][2]string{
			{"normal", "capbtn.normal"}, {"hover", "capbtn.hover"}, {"pressed", "capbtn.pressed"},
		}},
		{Part: "thumb", States: [][2]string{{"normal", "thumb.normal"}, {"hover", "thumb.hover"}}},
		{Part: "track", States: [][2]string{{"normal", "track.normal"}}},
		{Part: "slider.track", States: [][2]string{{"normal", "slot.normal"}}},
		{Part: "slider.fill", States: [][2]string{{"normal", "slot.fill"}}},
		{Part: "slider.thumb", States: [][2]string{
			{"normal", "knob.normal"}, {"disabled", "knob.disabled"},
		}},
		{Part: "progress.back", States: [][2]string{{"normal", "slot.normal"}}},
		{Part: "progress.fill", States: [][2]string{{"normal", "slot.fill"}}},
		{Part: "switch.track", States: [][2]string{
			{"normal", "switch.off"}, {"checked", "switch.on"}, {"disabled", "switch.disabled"},
		}},
		{Part: "switch.knob", States: [][2]string{
			{"normal", "knob.normal"}, {"disabled", "knob.disabled"},
		}},
		{Part: "check", Text: "control", States: [][2]string{
			{"normal", "check.normal"}, {"hover", "check.hover"},
			{"checked", "check.checked"}, {"disabled", "check.disabled"},
		}},
		{Part: "check.mark", States: [][2]string{{"normal", "mark.check"}, {"disabled", "mark.check"}}},
		{Part: "radio.mark", States: [][2]string{{"normal", "mark.radio"}, {"disabled", "mark.radio"}}},
		{Part: "arrow.down", States: [][2]string{{"normal", "arrow.down"}}},
		{Part: "arrow.up", States: [][2]string{{"normal", "arrow.up"}}},
		{Part: "arrow.left", States: [][2]string{{"normal", "arrow.left"}}},
		{Part: "arrow.right", States: [][2]string{{"normal", "arrow.right"}}},
		{Part: "expander.open", States: [][2]string{{"normal", "expander.open"}}},
		{Part: "expander.shut", States: [][2]string{{"normal", "expander.shut"}}},
		{Part: "focus", States: [][2]string{{"normal", "focus.ring"}}},
	}
	p.Colors = map[string]string{
		"background": minShell, "surface": "#2b3430", "surfaceAlt": "#222a27",
		"border": minInk, "divider": "#1d2622",
		"text": minText, "textMuted": minDim, "textOnAccent": "#07170f",
		"accent": minPhos, "accentHover": "#8cf7bd", "accentPress": minPhoL,
		"field": minWellC, "fieldBorder": minWellR,
		"focus": minPhos, "selection": "#63f0a455",
		"track": "#151c19", "thumb": minFace,
		"menuHover": minPhoD, "menuHoverBorder": minPhos, "menuGutter": "#222a27",
		"highlight": minShellH, "shadow": "#00000088",
		"bevelLight": minFaceH, "bevelDark": minFaceL,
		"danger": "#e0705e", "success": minPhos, "warning": minAmber,
	}
	p.Metrics = map[string]float32{
		"radius": 0, "radiusSmall": 0,
		"controlH": 18, "fieldH": 18, "comboH": 18,
		"checkbox": 16, "radio": 16, "scroll": 14, "thumb": 14, "sliderH": 8,
		"progressH": 8, "switchW": 26, "switchH": 16, "focusWidth": 1, "border": 1,
		"rowH": 16, "tabH": 18, "menuItemH": 18, "titleBar": minCaption, "bevelDepth": 1,
	}
	// The frame, and the chin the window stands on.
	//
	// Three rects, each stretching with the window, so the outline is the
	// same at every scale and at every size. The first is the whole window
	// down to the top of the chin; the other two step in twice below it.
	// The cut is at the *bottom* corners on purpose: the caption buttons
	// live in the top ones, and a silhouette that ate a corner of the close
	// button would be a skin with a bug in it rather than a shape.
	p.Window = &WindowSpec{
		Border:  [4]int{0, 4, minChin + 4, 4},
		Caption: minCaption,
		Layout:  ":minimize,close",
		Shape: []ShapeRect{
			{At: [4]int{0, 0, 0, minChin}, StretchX: true, StretchY: true},
			{At: [4]int{minStep, 0, minStep, minChin / 2}, StretchX: true, StretchY: true},
			{At: [4]int{minWaist, 0, minWaist, 0}, StretchX: true, StretchY: true},
		},
	}
	return p
}

// ---- Minim's own idioms -----------------------------------------------------

// minPlate is every raised face: a flat fill inside a one-pixel outline,
// with a one-pixel bevel inside that and the four corner pixels left empty.
//
// The empty corners are the skin's signature and they cost nothing: the
// fill is drawn as two overlapping rects rather than one, so no pixel is
// ever painted and then painted over.
func minPlate(ctx *paintengine2d.Context, w, h float32, fill, hi, lo paintengine2d.Color) {
	px(ctx, 1, 0, w-2, h, fill)
	px(ctx, 0, 1, w, h-2, fill)
	ink := Hex(minInk)
	px(ctx, 1, 0, w-2, 1, ink)
	px(ctx, 1, h-1, w-2, 1, ink)
	px(ctx, 0, 1, 1, h-2, ink)
	px(ctx, w-1, 1, 1, h-2, ink)
	px(ctx, 1, 1, w-2, 1, hi)
	px(ctx, 1, 1, 1, h-2, hi)
	px(ctx, 1, h-2, w-2, 1, lo)
	px(ctx, w-2, 1, 1, h-2, lo)
}

// minWell is the same plate sunk: the bevel turned over, a rim colour of
// its own, and the corners empty like everything else here.
func minWell(ctx *paintengine2d.Context, w, h float32, fill, rim paintengine2d.Color) {
	px(ctx, 1, 0, w-2, h, fill)
	px(ctx, 0, 1, w, h-2, fill)
	ink := Hex(minInk)
	px(ctx, 1, 0, w-2, 1, ink)
	px(ctx, 1, h-1, w-2, 1, ink)
	px(ctx, 0, 1, 1, h-2, ink)
	px(ctx, w-1, 1, 1, h-2, ink)
	px(ctx, 1, 1, w-2, 1, rim)
	px(ctx, 1, 1, 1, h-2, rim)
	px(ctx, 1, h-2, w-2, 1, rim)
	px(ctx, w-2, 1, 1, h-2, rim)
}

// minKnob is a slider's grip: a tall plate with a notch scored across its
// waist, so the eye finds the middle of it at a glance.
func minKnob(ctx *paintengine2d.Context, w, h float32, fill, hi, lo paintengine2d.Color) {
	minPlate(ctx, w, h, fill, hi, lo)
	px(ctx, 2, h/2-1, w-4, 1, lo)
	px(ctx, 2, h/2, w-4, 1, hi)
}

// minBand is the caption: a sunk strip with a phosphor rule along its
// bottom when the window has the keyboard, and nothing when it does not —
// the one place the skin says which window is the active one.
func minBand(ctx *paintengine2d.Context, w, h float32, fill, accent paintengine2d.Color) {
	px(ctx, 0, 0, w, h, fill)
	px(ctx, 0, 0, w, 1, Hex(minInk))
	px(ctx, 0, 1, w, 1, Hex("#2b3430"))
	if accent.A > 0 {
		px(ctx, 0, h-2, w, 1, accent)
	}
	px(ctx, 0, h-1, w, 1, Hex(minInk))
}

// minDither is a checkerboard of two close colours: the panel's texture,
// and the trick every machine that wanted a gradient and had no room for
// one reached for.
func minDither(ctx *paintengine2d.Context, w, h float32, a, b paintengine2d.Color) {
	px(ctx, 0, 0, w, h, a)
	for y := float32(0); y < h; y++ {
		for x := float32(0); x < w; x++ {
			if (int(x)+int(y))%2 == 0 {
				px(ctx, x, y, 1, 1, b)
			}
		}
	}
}

// minDots is the dotted focus rectangle: every other pixel, on the grid.
func minDots(ctx *paintengine2d.Context, x0, y0, w, h float32, col paintengine2d.Color) {
	for x := float32(0); x < w; x += 2 {
		px(ctx, x0+x, y0, 1, 1, col)
		px(ctx, x0+x, y0+h-1, 1, 1, col)
	}
	for y := float32(0); y < h; y += 2 {
		px(ctx, x0, y0+y, 1, 1, col)
		px(ctx, x0+w-1, y0+y, 1, 1, col)
	}
}

// minArrow is a stepped triangle drawn a row at a time, so every edge lands
// on a pixel instead of between two.
func minArrow(ctx *paintengine2d.Context, dir string, col paintengine2d.Color) {
	const n = 4
	for i := float32(0); i < n; i++ {
		long, short := 7-i*2, i
		switch dir {
		case "up":
			px(ctx, 4.5+short, 9-i, long, 1, col)
		case "down":
			px(ctx, 4.5+short, 6+i, long, 1, col)
		case "left":
			px(ctx, 9-i, 4.5+short, 1, long, col)
		case "right":
			px(ctx, 6+i, 4.5+short, 1, long, col)
		}
	}
}
