package skinart

import "github.com/codemodify/paintengine2d"

// Cassette is the toolkit's pixel skin: a six-colour interface drawn on a
// whole-pixel grid, the way a machine with a palette and no antialiasing
// would have drawn it.
//
// It is here to exercise the half of the format Nocturne never touches. A
// pixelated sheet is drawn nearest-neighbour at *integer* multiples only, so
// at a 1.75 display the art is drawn at 2× and fitted while the layout, the
// text and the hit regions stay at the true 1.75 — a crisp 2× sprite in a
// 1.75 window reads as pixel art, and a bilinear 1.75 enlargement of a 1×
// one reads as a mistake. Its 2× sheet is the 1× sheet with every pixel
// doubled, which is the only honest way to publish one.
//
// Everything is hard: two-pixel bevels, square corners, flat faces, no
// gradient anywhere. The only curve in the whole skin is the one the eye
// draws between two stair-stepped pixels.

// Cassette's palette: six colours and their two washes, the whole skin.
const (
	casInk    = "#101820" // outline, shadow, text on light
	casDeep   = "#1b2a33" // window
	casMid    = "#2c3e4f" // face
	casMidHi  = "#3d5366" // face, lit edge
	casMidLo  = "#16222b" // face, shadowed edge
	casSteel  = "#5a7d8c" // rims, disabled ink
	casMint   = "#8fb9a8" // the cool accent: selection, progress
	casCream  = "#f2e9c9" // text on dark, highlights
	casRust   = "#d95f45" // the warm accent: what is on, where the keyboard is
	casRustLo = "#a6412c"
)

// Cassette's cell grid. Small numbers on purpose: this is art drawn at 1×,
// and a 2px bevel is 2px.
const (
	casW = 32 // a face cell
	casH = 20
)

// casSlice keeps the 2px bevel and one pixel of face either side of it.
var casSlice = [4]int{4, 5, 4, 5}

// Cassette builds the skin's plan.
func Cassette() *Plan {
	sh := &Sheet{Name: "chrome", Pixelated: true}
	l := &lay{sh: sh, gap: 2, faceW: casW, faceH: casH, faceSlice: casSlice}
	p := &Plan{
		Name:    "cassette",
		Label:   "Cassette",
		Year:    2026,
		Lineage: "uitoolkit",
		Summary: "A pixel skin: six colours, two-pixel bevels and square corners, drawn at 1x and doubled exactly — the toolkit's test of the pixelated path.",
		Family:  "dark",
		Base:    "win95",
		Sheets:  []*Sheet{sh},
	}

	// ---- faces ------------------------------------------------------------
	//
	// Out and in are the same two colours the other way round. That is the
	// whole grammar of a bevelled interface, and it needs no gradient.
	out := func(face string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			casFace(ctx, w, h, hex(face), hex(casMidHi), hex(casMidLo))
		}
	}
	in := func(face string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			casFace(ctx, w, h, hex(face), hex(casMidLo), hex(casMidHi))
		}
	}

	l.row(casH)
	l.face("button.normal", out(casMid))
	l.face("button.hover", out("#384d60"))
	l.face("button.pressed", in("#243543"))
	l.face("button.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex("#243139"), hex("#31424e"), hex("#1a242c"))
	})
	l.face("button.focus", func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex(casMid), hex(casMidHi), hex(casMidLo))
		casDots(ctx, 2, 2, w-4, h-4, hex(casCream))
	})
	l.face("button.default", func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex(casRust), hex("#e8836c"), hex(casRustLo))
	})
	l.face("button.checked", in(casRustLo))

	l.row(casH)
	l.face("field.normal", func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex(casInk), hex(casSteel))
	})
	l.face("field.focus", func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex(casInk), hex(casRust))
	})
	l.face("field.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex("#19242c"), hex("#2f3d47"))
	})
	l.face("combo.normal", out(casMid))
	l.face("combo.hover", out("#384d60"))
	l.face("combo.pressed", in("#243543"))

	l.row(casH)
	l.face("tool.normal", func(ctx *paintengine2d.Context, w, h float32) {})
	l.face("tool.hover", func(ctx *paintengine2d.Context, w, h float32) {
		bevel(ctx, w, h, 1, hex(casMidHi), hex(casMidLo))
	})
	l.face("tool.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 1, 1, w-2, h-2, hex("#22313c"))
		bevel(ctx, w, h, 1, hex(casMidLo), hex(casMidHi))
	})
	l.face("tool.checked", func(ctx *paintengine2d.Context, w, h float32) {
		casHatch(ctx, w, h, hex("#22313c"), hex(casMid))
		bevel(ctx, w, h, 1, hex(casMidLo), hex(casMidHi))
	})
	l.face("tab.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 2, w, h-3, hex("#243543"))
		px(ctx, 0, 2, w, 1, hex(casMidHi))
		px(ctx, 0, h-1, w, 1, hex(casInk))
	})
	l.face("tab.hover", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 2, w, h-3, hex("#2e4253"))
		px(ctx, 0, 2, w, 1, hex(casMidHi))
		px(ctx, 0, h-1, w, 1, hex(casInk))
	})
	l.face("tab.checked", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casMid))
		px(ctx, 0, 0, w, 2, hex(casRust))
		px(ctx, 0, 2, 1, h-2, hex(casMidHi))
		px(ctx, w-1, 2, 1, h-2, hex(casMidLo))
	})

	l.row(casH)
	l.face("row.hover", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex("#243543"))
	})
	l.face("row.checked", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex("#3a5c52"))
		px(ctx, 0, 0, 2, h, hex(casMint))
	})
	l.face("menu.hover", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casRustLo))
	})
	l.face("panel.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex("#22313c"))
		bevel(ctx, w, h, 1, hex(casMidHi), hex(casMidLo))
	})
	l.face("bar.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casMid))
		px(ctx, 0, 0, w, 1, hex(casMidHi))
		px(ctx, 0, h-1, w, 1, hex(casInk))
	})
	l.face("menu.frame", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casMid))
		bevel(ctx, w, h, 1, hex(casMidHi), hex(casInk))
	})
	// Orientation-neutral, like Nocturne's: Face is not told which way a
	// splitter runs, so the grip is a dot cluster rather than a line.
	l.face("splitter.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casMid))
		for _, d := range [][2]float32{{-3, -3}, {1, -3}, {-3, 1}, {1, 1}} {
			px(ctx, w/2+d[0], h/2+d[1], 2, 2, hex(casMidHi))
		}
	})
	l.face("tooltip.normal", func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casCream))
		bevel(ctx, w, h, 1, hex("#ffffff"), hex(casSteel))
	})

	// The window is a two-colour dither: the era's answer to a gradient on a
	// machine that had no spare colours for one. It tiles, so the pattern
	// stays on the grid however big the window is.
	l.row(16)
	l.cell("window.normal", 16, 16, [4]int{0, 0, 0, 0}, "tile", false, func(ctx *paintengine2d.Context, w, h float32) {
		casDither(ctx, w, h, hex(casDeep), hex("#1f303a"))
	})
	l.cell("caption.normal", 48, 22, [4]int{4, 6, 3, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casRustLo))
		px(ctx, 0, 0, w, 1, hex("#e8836c"))
		px(ctx, 0, 1, w, 1, hex(casRust))
		px(ctx, 0, h-1, w, 1, hex(casInk))
	})
	l.cell("caption.inactive", 48, 22, [4]int{4, 6, 3, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex("#2c3e4f"))
		px(ctx, 0, 0, w, 1, hex(casMidHi))
		px(ctx, 0, h-1, w, 1, hex(casInk))
	})
	// This era put raised keys in its title bars, so Cassette does too.
	l.cell("capbtn.normal", 16, 14, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex(casMid), hex(casMidHi), hex(casMidLo))
	})
	l.cell("capbtn.pressed", 16, 14, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex("#243543"), hex(casMidLo), hex(casMidHi))
	})

	// ---- small parts ------------------------------------------------------

	l.row(16)
	l.cell("thumb.normal", 24, 16, [4]int{4, 4, 4, 4}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex(casMid), hex(casMidHi), hex(casMidLo))
	})
	l.cell("thumb.hover", 24, 16, [4]int{4, 4, 4, 4}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex("#384d60"), hex("#4a6479"), hex(casMidLo))
	})
	l.cell("track.normal", 24, 16, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casHatch(ctx, w, h, hex("#1b2a33"), hex("#22333e"))
		bevel(ctx, w, h, 1, hex(casMidLo), hex(casMidHi))
	})
	l.cell("fill.normal", 24, 16, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 0, 0, w, h, hex(casMint))
		px(ctx, 0, 0, w, 1, hex("#b6dbcb"))
		px(ctx, 0, h-1, w, 1, hex("#5f8879"))
	})
	l.cell("knob.normal", 12, 16, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex(casCream), hex("#ffffff"), hex(casSteel))
	})
	l.cell("knob.disabled", 12, 16, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casFace(ctx, w, h, hex("#5d6a72"), hex("#7c8a92"), hex("#3a464e"))
	})

	l.row(16)
	l.cell("check.normal", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex(casInk), hex(casSteel))
	})
	l.cell("check.hover", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex("#16222b"), hex(casCream))
	})
	l.cell("check.checked", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex(casRustLo), hex(casRust))
	})
	l.cell("check.disabled", 16, 16, [4]int{5, 5, 5, 5}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex("#19242c"), hex("#37454f"))
	})
	l.glyph("mark.check", 16, func(ctx *paintengine2d.Context, w, h float32) {
		// A hand-placed tick: seven pixels, the shape a 16×16 icon of the
		// era would have had, because a stroked path at this size is mush.
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
			casArrow(ctx, dir, paintengine2d.RGB(1, 1, 1))
		})
	}
	l.glyph("expander.open", 16, func(ctx *paintengine2d.Context, w, h float32) {
		casPlusMinus(ctx, false, paintengine2d.RGB(1, 1, 1))
	})
	l.glyph("expander.shut", 16, func(ctx *paintengine2d.Context, w, h float32) {
		casPlusMinus(ctx, true, paintengine2d.RGB(1, 1, 1))
	})
	// The focus mark of every machine that had one: a dotted rectangle.
	l.cell("focus.ring", 16, 16, [4]int{4, 4, 4, 4}, "none", false, func(ctx *paintengine2d.Context, w, h float32) {
		casDots(ctx, 0, 0, w, h, hex(casCream))
	})

	l.row(16)
	l.cell("switch.off", 28, 16, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex(casInk), hex(casSteel))
	})
	l.cell("switch.on", 28, 16, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		px(ctx, 1, 1, w-2, h-2, hex(casRust))
		bevel(ctx, w, h, 1, hex(casRustLo), hex("#e8836c"))
	})
	l.cell("switch.disabled", 28, 16, [4]int{4, 8, 4, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		casWell(ctx, w, h, hex("#19242c"), hex("#37454f"))
	})
	l.close()

	// ---- bindings ---------------------------------------------------------

	p.Text = []TextRole{
		{Name: "control", Color: casCream, Hover: "#ffffff", Pressed: casCream, Disabled: casSteel, Checked: casCream},
		{Name: "onDark", Color: casInk, Disabled: casSteel},
		{Name: "caption", Color: casCream, Disabled: "#9db0bb", Bold: true},
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
		{Part: "splitter", States: [][2]string{{"normal", "splitter.normal"}}},
		{Part: "bar", States: [][2]string{{"normal", "bar.normal"}}},
		{Part: "menu.frame", States: [][2]string{{"normal", "menu.frame"}}},
		{Part: "tooltip", Text: "onDark", States: [][2]string{{"normal", "tooltip.normal"}}},
		{Part: "window", States: [][2]string{{"normal", "window.normal"}}},
		{Part: "caption", Text: "caption", States: [][2]string{
			{"normal", "caption.normal"}, {"inactive", "caption.inactive"},
		}},
		{Part: "caption.button", Text: "control", States: [][2]string{
			{"normal", "capbtn.normal"}, {"pressed", "capbtn.pressed"},
		}},
		{Part: "thumb", States: [][2]string{{"normal", "thumb.normal"}, {"hover", "thumb.hover"}}},
		{Part: "track", States: [][2]string{{"normal", "track.normal"}}},
		{Part: "slider.track", States: [][2]string{{"normal", "track.normal"}}},
		{Part: "slider.fill", States: [][2]string{{"normal", "fill.normal"}}},
		{Part: "slider.thumb", States: [][2]string{
			{"normal", "knob.normal"}, {"disabled", "knob.disabled"},
		}},
		{Part: "progress.back", States: [][2]string{{"normal", "track.normal"}}},
		{Part: "progress.fill", States: [][2]string{{"normal", "fill.normal"}}},
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
		"background": casDeep, "surface": casMid, "surfaceAlt": "#22313c",
		"border": casInk, "divider": "#22313c",
		"text": casCream, "textMuted": "#a8bcc4", "textOnAccent": casInk,
		"accent": casRust, "accentHover": "#e8836c", "accentPress": casRustLo,
		"field": casInk, "fieldBorder": casSteel,
		"focus": casCream, "selection": "#8fb9a866",
		"track": "#1b2a33", "thumb": casMid,
		"menuHover": casRustLo, "menuHoverBorder": casRust, "menuGutter": "#22313c",
		"highlight": casMidHi, "shadow": "#00000088",
		"bevelLight": casMidHi, "bevelDark": casMidLo,
		"danger": casRust, "success": casMint, "warning": "#e0b24a",
	}
	p.Metrics = map[string]float32{
		"radius": 0, "radiusSmall": 0,
		"controlH": 26, "fieldH": 26, "comboH": 26,
		"checkbox": 16, "radio": 16, "scroll": 16, "thumb": 16, "sliderH": 8,
		"progressH": 12, "switchW": 28, "switchH": 16, "focusWidth": 1, "border": 1,
		"rowH": 22, "tabH": 26, "menuItemH": 22, "titleBar": 22, "bevelDepth": 2,
	}
	p.Window = &WindowSpec{
		Border:  [4]int{1, 2, 2, 2},
		Caption: 22,
		Layout:  ":minimize,maximize,close",
	}
	return p
}

// ---- Cassette's own idioms ------------------------------------------------

// casFace is a flat face inside a two-pixel bevel: light where the light
// comes from, dark where it does not, and nothing in between.
func casFace(ctx *paintengine2d.Context, w, h float32, face, light, dark paintengine2d.Color) {
	px(ctx, 0, 0, w, h, face)
	bevel(ctx, w, h, 2, light, dark)
	// The outermost pixel of the dark edge is the outline, so faces that
	// touch each other do not share a lit edge.
	px(ctx, 0, h-1, w, 1, hex(casInk))
	px(ctx, w-1, 0, 1, h, hex(casInk))
}

// casWell is the same thing pressed in, with a rim colour of its own.
func casWell(ctx *paintengine2d.Context, w, h float32, fill, rim paintengine2d.Color) {
	px(ctx, 0, 0, w, h, fill)
	bevel(ctx, w, h, 1, hex(casMidLo), hex(casMidHi))
	px(ctx, 1, 1, w-2, 1, hex("#00000055"))
	px(ctx, 0, 0, w, 1, rim)
	px(ctx, 0, h-1, w, 1, rim)
	px(ctx, 0, 0, 1, h, rim)
	px(ctx, w-1, 0, 1, h, rim)
}

// casHatch is a one-pixel diagonal hatch: a texture with no colours to spare.
func casHatch(ctx *paintengine2d.Context, w, h float32, base, line paintengine2d.Color) {
	px(ctx, 0, 0, w, h, base)
	for y := float32(0); y < h; y++ {
		for x := float32(0); x < w; x++ {
			if int(x+y)%4 == 0 {
				px(ctx, x, y, 1, 1, line)
			}
		}
	}
}

// casDither is a checkerboard of two close colours — the trick every
// 16-colour machine used where it wanted a gradient and had no room for one.
func casDither(ctx *paintengine2d.Context, w, h float32, a, b paintengine2d.Color) {
	px(ctx, 0, 0, w, h, a)
	for y := float32(0); y < h; y++ {
		for x := float32(0); x < w; x++ {
			if (int(x)+int(y))%2 == 0 {
				px(ctx, x, y, 1, 1, b)
			}
		}
	}
}

// casDots is the dotted focus rectangle: every other pixel, on the grid.
func casDots(ctx *paintengine2d.Context, x0, y0, w, h float32, col paintengine2d.Color) {
	for x := float32(0); x < w; x += 2 {
		px(ctx, x0+x, y0, 1, 1, col)
		px(ctx, x0+x, y0+h-1, 1, 1, col)
	}
	for y := float32(0); y < h; y += 2 {
		px(ctx, x0, y0+y, 1, 1, col)
		px(ctx, x0+w-1, y0+y, 1, 1, col)
	}
}

// casArrow is a stepped triangle, drawn a row at a time so every edge lands
// on a pixel.
func casArrow(ctx *paintengine2d.Context, dir string, col paintengine2d.Color) {
	const n = 4 // rows
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

// casPlusMinus is the tree disclosure of every file manager that had one: a
// boxed plus when shut, a boxed minus when open.
func casPlusMinus(ctx *paintengine2d.Context, shut bool, col paintengine2d.Color) {
	px(ctx, 3, 3, 10, 1, col)
	px(ctx, 3, 12, 10, 1, col)
	px(ctx, 3, 3, 1, 10, col)
	px(ctx, 12, 3, 1, 10, col)
	px(ctx, 5, 7, 6, 2, col)
	if shut {
		px(ctx, 7, 5, 2, 6, col)
	}
}
