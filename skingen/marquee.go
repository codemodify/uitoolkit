package skingen

import "github.com/codemodify/paintengine2d"

// Marquee is the skin the big player wears: brushed steel and glass over a
// deep blue display, drawn from paths and gradients so it is exact at every
// scale.
//
// This is the other half of the pair Minim starts. Minim is a front panel
// at eighteen pixels a control; Marquee is the cabinet the same maker
// shipped three years later, when a window could afford a gradient, a gloss
// and a corner radius — every face is a rounded plate with the light on its
// top half and a hairline round it, and the accent is the blue a display of
// that era glowed.
//
// Its silhouette is two rects. A shallow brow steps the top corners in, and
// the body under it carries a bottom radius four times the top one, so the
// cabinet sits on a dome rather than on a rectangle. The brow cuts only the
// top six design pixels, which is *above* where a caption button starts: a framed caption centres its buttons, so the first row one of them
// occupies is about a fifth of the caption's height down, and a silhouette
// that bit deeper than that would eat the corner of a close button. Which
// side those buttons sit on is the desktop's choice and not a skin's, so
// every shape here stays out of both top corners.
//
// The compact mode of the app that wears it is not in here at all. A skin
// declares one window shape, and the second silhouette — the flat stadium
// the player collapses to — is the app's own, set with SetShapeFunc. That
// is the division the toolkit draws: the look says what its windows look
// like, and an app that knows it has two shapes says so itself.

// Marquee's palette: steel, glass and one blue light.
const (
	mqInk    = "#05070c" // the hairline round every face
	mqShell0 = "#3d4859" // the cabinet, lit from above
	mqShell1 = "#171d29"
	mqDeep0  = "#151b28" // the display well
	mqDeep1  = "#070a11"

	mqFace0 = "#4d5a6e" // a resting face
	mqFace1 = "#29334a"
	mqHot0  = "#61708a"
	mqHot1  = "#333f59"
	mqDown0 = "#181f2d"
	mqDown1 = "#242d3f"
	mqOff0  = "#2c3342"
	mqOff1  = "#232937"

	mqAccent = "#4fa8ff" // the blue a display of this era glowed
	mqAccHi  = "#a5d6ff"
	mqAccLo  = "#2a6cbd"
	mqOnAcc  = "#04121f"

	mqText = "#e9eff8"
	mqDim  = "#93a2b8"
	mqGone = "#5a6579"

	mqLight = "#ffffff26" // the highlight inside a lit top edge
	mqDark  = "#00000070" // the shadow inside a sunken one
	mqGlass = "#ffffff1c" // the gloss over the top half of a face
)

// Marquee's cell grid. The face cell is wide enough that a nine-slice's
// middle is most of it, which is what keeps a gradient smooth when a button
// is stretched to three times the cell's width.
const (
	mqW   = 84 // a face cell
	mqH   = 32
	mqRad = 7 // the radius every plate is rounded by
)

// mqSlice keeps the corner radius, the hairline and the gloss's turn.
var mqSlice = [4]int{12, 12, 12, 12}

// The frame, in design pixels.
const (
	mqCaption = 40
	mqBrow    = 10  // how deep the brow is
	mqBrowIn  = 60  // and how far in it holds the top corners
	mqRadTop  = 12  // the frame's own corners
	mqRadBot  = 48  // the dome the cabinet stands on
	mqBezel   = 16  // the border: content keeps this much shell round it
	mqFoot    = 26  // and this much at the bottom, clear of the dome
	mqDisplay = 112 // the display well's own cell height
)

// Marquee builds the skin's plan.
func Marquee() *Plan {
	sh := &Sheet{Name: "chrome"}
	l := &lay{sh: sh, gap: 2, faceW: mqW, faceH: mqH, faceSlice: mqSlice}
	p := &Plan{
		Name:    "marquee",
		Label:   "Marquee",
		Year:    2026,
		Lineage: "uitoolkit",
		Summary: "Brushed steel and glass over a deep blue display, on a domed cabinet — the skin the big player demo wears.",
		Family:  "dark",
		Base:    "breeze-night",
		Sheets:  []*Sheet{sh},
	}

	// ---- faces ------------------------------------------------------------
	plate := func(c0, c1, edge, hi string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			mqPlate(ctx, w, h, Hex(c0), Hex(c1), Hex(edge), Hex(hi))
		}
	}
	l.row(mqH)
	l.face("button.normal", plate(mqFace0, mqFace1, mqInk, mqLight))
	l.face("button.hover", plate(mqHot0, mqHot1, mqInk, "#ffffff36"))
	l.face("button.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		mqSunk(ctx, w, h, Hex(mqDown0), Hex(mqDown1), Hex(mqInk))
	})
	l.face("button.disabled", plate(mqOff0, mqOff1, "#0d1119", "#00000000"))
	l.face("button.focus", func(ctx *paintengine2d.Context, w, h float32) {
		mqPlate(ctx, w, h, Hex(mqFace0), Hex(mqFace1), Hex(mqInk), Hex(mqLight))
		outline(ctx, paintengine2d.XYWH(1.5, 1.5, w-3, h-3), mqRad-1, 2, Hex(mqAccent).WithAlpha(0.95))
	})
	l.face("button.default", plate(mqAccHi, mqAccLo, "#1d4d85", "#ffffff5c"))
	l.face("button.checked", func(ctx *paintengine2d.Context, w, h float32) {
		mqSunk(ctx, w, h, Hex("#12304f"), Hex("#193f66"), Hex("#1d4d85"))
	})

	l.row(mqH)
	l.face("field.normal", func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex(mqDeep1), Hex("#3a465c"))
	})
	l.face("field.focus", func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex(mqDeep1), Hex(mqAccent))
	})
	l.face("field.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex("#0d1119"), Hex("#2a3243"))
	})
	l.face("combo.normal", plate(mqFace0, mqFace1, mqInk, mqLight))
	l.face("combo.hover", plate(mqHot0, mqHot1, mqInk, "#ffffff36"))
	l.face("combo.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		mqSunk(ctx, w, h, Hex(mqDown0), Hex(mqDown1), Hex(mqInk))
	})

	// A tool button is nothing until the pointer finds it: a transport bar
	// of this era was a row of glyphs on the cabinet, and a row of outlined
	// boxes would have been a different machine.
	l.row(mqH)
	l.face("tool.normal", func(ctx *paintengine2d.Context, w, h float32) {})
	l.face("tool.hover", func(ctx *paintengine2d.Context, w, h float32) {
		mqPlate(ctx, w, h, Hex(mqHot0), Hex(mqHot1), Hex("#ffffff1a"), Hex("#ffffff30"))
	})
	l.face("tool.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		mqSunk(ctx, w, h, Hex(mqDown0), Hex(mqDown1), Hex("#ffffff14"))
	})
	l.face("tool.checked", func(ctx *paintengine2d.Context, w, h float32) {
		mqSunk(ctx, w, h, Hex("#12304f"), Hex("#193f66"), Hex(mqAccLo))
	})
	l.face("row.hover", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(Hex("#243048")))
	})
	l.face("row.checked", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, Hex("#2a5a94")), stop(1, Hex("#1c3f6c")))
		px(ctx, 0, 0, w, 1, Hex("#4a86c9"))
	})

	l.row(mqH)
	l.face("tab.normal", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 2.5, w-1, h-2), 6, stop(0, Hex("#333d51")), stop(1, Hex("#212938")))
		px(ctx, 0, h-1, w, 1, Hex(mqInk))
	})
	l.face("tab.hover", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 2.5, w-1, h-2), 6, stop(0, Hex("#414d65")), stop(1, Hex("#283243")))
		px(ctx, 0, h-1, w, 1, Hex(mqInk))
	})
	l.face("tab.checked", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 0.5, w-1, h), 6, stop(0, Hex("#55637c")), stop(1, Hex("#2d3850")))
		ctx.DrawRect(paintengine2d.XYWH(2, 0, w-4, 2), paintengine2d.Fill(Hex(mqAccent)))
	})
	l.face("menu.hover", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 4, stop(0, Hex("#39628f")), stop(1, Hex("#23446b")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 4, 1, Hex("#4f86c4"))
	})

	l.row(mqH)
	l.face("panel.normal", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 8, stop(0, Hex("#2b3347")), stop(1, Hex("#1e2433")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 8, 1, Hex("#3a4358"))
	})
	l.face("bar.normal", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0,
			stop(0, Hex("#404b5e")), stop(0.5, Hex("#2c3546")), stop(0.52, Hex("#262e3e")), stop(1, Hex("#1b2230")))
		px(ctx, 0, 0, w, 1, Hex("#5a6880"))
		px(ctx, 0, h-1, w, 1, Hex(mqInk))
	})
	l.face("menu.frame", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 8, 8, paintengine2d.Fill(Hex("#262e3e")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 8, 1, Hex("#4a5568"))
	})
	l.face("tooltip.normal", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 6, 6, paintengine2d.Fill(Hex("#0d1420")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 6, 1, Hex(mqAccLo))
	})
	l.face("splitter.normal", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, Hex("#333c4e")), stop(1, Hex("#232a38")))
		for _, d := range [][2]float32{{-4, -4}, {1, -4}, {-4, 1}, {1, 1}} {
			ctx.DrawRoundRect(paintengine2d.XYWH(w/2+d[0], h/2+d[1], 3, 3), 1, 1, paintengine2d.Fill(Hex("#5c6a82")))
		}
	})

	// ---- the frame --------------------------------------------------------
	//
	// The window art is the cabinet: one gradient across the whole box,
	// which the silhouette then cuts to the brow and the dome. It carries no
	// rim of its own — the outline's edge is the rim, and a line drawn at
	// the art's edge would be cut away with the pixels beside it.
	l.row(96)
	l.cell("window.normal", 64, 96, [4]int{12, 12, 12, 12}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0,
			stop(0, Hex(mqShell0)), stop(0.28, Hex("#2a3243")), stop(1, Hex(mqShell1)))
	})
	// The display well, as its own part of the cabinet: sunk, cold, with a
	// hairline of sky along its top edge.
	l.cell("display.normal", 96, mqDisplay, [4]int{14, 14, 14, 14}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqDisplayWell(ctx, w, h)
	})
	l.cell("caption.normal", 128, mqCaption, [4]int{12, 20, 8, 20}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqBand(ctx, w, h, Hex(mqAccent))
	})
	l.cell("caption.inactive", 128, mqCaption, [4]int{12, 20, 8, 20}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqBand(ctx, w, h, paintengine2d.Color{})
	})

	// The caption buttons are the same plate at a caption button's
	// proportions: taller than they are wide, so a radius rather than a cap.
	l.row(30)
	for _, s := range []struct{ name, c0, c1, hi string }{
		{"capbtn.normal", "#4a5668", "#2a3345", mqLight},
		{"capbtn.hover", "#5f6e88", "#354158", "#ffffff3a"},
		{"capbtn.pressed", "#1a2130", "#252e40", "#00000000"},
	} {
		cell := s
		l.cell(cell.name, 30, 30, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
			mqPlate(ctx, w, h, Hex(cell.c0), Hex(cell.c1), Hex(mqInk), Hex(cell.hi))
		})
	}

	// ---- the small parts --------------------------------------------------

	l.row(20)
	l.cell("thumb.normal", 28, 20, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqPlate(ctx, w, h, Hex(mqFace0), Hex(mqFace1), Hex(mqInk), Hex(mqLight))
	})
	l.cell("thumb.hover", 28, 20, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqPlate(ctx, w, h, Hex(mqHot0), Hex(mqHot1), Hex(mqInk), Hex("#ffffff36"))
	})
	l.cell("track.normal", 28, 20, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex("#141a26"), Hex("#333c4e"))
	})
	l.cell("slot.normal", 28, 10, [4]int{4, 6, 4, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex(mqDeep1), Hex("#39445a"))
	})
	l.cell("slot.fill", 28, 10, [4]int{4, 6, 4, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 4, stop(0, Hex(mqAccHi)), stop(1, Hex(mqAccLo)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 4, 1, Hex("#1c4a80"))
	})
	l.cell("knob.normal", 16, 22, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqKnob(ctx, w, h, Hex("#d6dfec"), Hex("#8794a9"))
	})
	l.cell("knob.disabled", 16, 22, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqKnob(ctx, w, h, Hex("#5d6678"), Hex("#333b49"))
	})

	l.row(20)
	l.cell("check.normal", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex(mqDeep1), Hex("#3a465c"))
	})
	l.cell("check.hover", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex(mqDeep1), Hex(mqAccent))
	})
	l.cell("check.checked", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 4, stop(0, Hex(mqAccHi)), stop(1, Hex(mqAccLo)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 4, 1, Hex("#1c4a80"))
	})
	l.cell("check.disabled", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex("#0d1119"), Hex("#2a3243"))
	})
	l.glyph("mark.check", 20, func(ctx *paintengine2d.Context, w, h float32) {
		p := paintengine2d.NewPath()
		p.MoveTo(5, 10.5)
		p.LineTo(8.5, 14)
		p.LineTo(15, 6.5)
		st := paintengine2d.StrokePaint(paintengine2d.RGB(1, 1, 1), 2.4)
		st.Stroke.Cap = paintengine2d.CapRound
		st.Stroke.Join = paintengine2d.JoinRound
		ctx.DrawPath(p, st)
	})
	l.glyph("mark.radio", 20, func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawCircle(paintengine2d.Pt(w/2, h/2), 3.6, paintengine2d.Fill(paintengine2d.RGB(1, 1, 1)))
	})

	l.row(20)
	for _, d := range []string{"down", "up", "left", "right"} {
		dir := d
		l.glyph("arrow."+dir, 20, func(ctx *paintengine2d.Context, w, h float32) {
			chevron(ctx, w, h, dir, 2, paintengine2d.RGB(1, 1, 1))
		})
	}
	l.glyph("expander.open", 20, func(ctx *paintengine2d.Context, w, h float32) {
		chevron(ctx, w, h, "down", 2, paintengine2d.RGB(1, 1, 1))
	})
	l.glyph("expander.shut", 20, func(ctx *paintengine2d.Context, w, h float32) {
		chevron(ctx, w, h, "right", 2, paintengine2d.RGB(1, 1, 1))
	})
	l.cell("focus.ring", 24, 24, [4]int{9, 9, 9, 9}, "none", false, func(ctx *paintengine2d.Context, w, h float32) {
		outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), 7, 2, Hex(mqAccent).WithAlpha(0.95))
	})

	l.row(22)
	l.cell("switch.off", 40, 22, [4]int{6, 12, 6, 12}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex(mqDeep1), Hex("#3a465c"))
	})
	l.cell("switch.on", 40, 22, [4]int{6, 12, 6, 12}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0.5, 0.5, w-1, h-1), h/2, stop(0, Hex(mqAccHi)), stop(1, Hex(mqAccLo)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), h/2, 1, Hex("#1c4a80"))
	})
	l.cell("switch.disabled", 40, 22, [4]int{6, 12, 6, 12}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		mqWell(ctx, w, h, Hex("#0d1119"), Hex("#2a3243"))
	})
	l.close()

	// ---- bindings ---------------------------------------------------------

	p.Text = []TextRole{
		{Name: "control", Color: mqText, Hover: "#ffffff", Pressed: mqDim,
			Disabled: mqGone, Checked: mqAccHi, Default: mqOnAcc},
		{Name: "caption", Color: mqText, Disabled: mqDim, Bold: true},
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
		"background": "#242b3a", "surface": "#2b3347", "surfaceAlt": "#1e2433",
		"border": "#3a4358", "divider": "#2a3243",
		"text": mqText, "textMuted": mqDim, "textOnAccent": mqOnAcc,
		"accent": mqAccent, "accentHover": mqAccHi, "accentPress": mqAccLo,
		"field": mqDeep1, "fieldBorder": "#3a465c",
		"focus": mqAccent, "selection": "#4fa8ff4d",
		"track": "#141a26", "thumb": "#3f4a5e",
		"menuHover": "#23446b", "menuHoverBorder": "#4f86c4", "menuGutter": "#1e2433",
		"highlight": "#ffffff1a", "shadow": "#00000088",
		"bevelLight": "#5a6880", "bevelDark": mqInk,
		"danger": "#e0705e", "success": "#76c89b", "warning": "#e0b24a",
	}
	p.Metrics = map[string]float32{
		"radius": mqRad, "radiusSmall": 5, "controlH": 32, "fieldH": 32, "comboH": 32,
		"checkbox": 20, "radio": 20, "scroll": 14, "thumb": 22, "sliderH": 10,
		"progressH": 10, "switchW": 40, "switchH": 22, "focusWidth": 2, "border": 1,
		"rowH": 28, "tabH": 32, "menuItemH": 28, "titleBar": mqCaption,
	}
	// The frame, and the outline the cabinet is cut to: a brow across the
	// top that steps the corners in, and the body under it standing on a
	// dome. The two overlap by four design pixels so the union has no seam.
	p.Window = &WindowSpec{
		Border:  [4]int{0, mqBezel, mqFoot, mqBezel},
		Caption: mqCaption,
		Layout:  ":minimize,maximize,close",
		Radius:  [4]int{mqRadTop, mqRadTop, 0, 0},
		Shape: []ShapeRect{
			{At: [4]int{mqBrowIn, 0, mqBrowIn, mqBrow}, Radius: [4]int{10, 10, 0, 0},
				StretchX: true, StretchY: true},
			{At: [4]int{0, mqBrow - 4, 0, 0}, Radius: [4]int{mqRadTop, mqRadTop, mqRadBot, mqRadBot},
				StretchX: true, StretchY: true},
		},
	}
	return p
}

// ---- Marquee's own idioms ---------------------------------------------------

// mqPlate is every raised face: a vertical gradient in a rounded rect, a
// gloss over its top half, a highlight inside its top edge and a hairline
// round the lot. It is the one idiom the whole skin is built from, which is
// what makes a cabinet look like one machine rather than a parts bin.
func mqPlate(ctx *paintengine2d.Context, w, h float32, c0, c1, edge, hi paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, mqRad, stop(0, c0), stop(1, c1))
	mqSheen(ctx, b, mqRad)
	if hi.A > 0 {
		topLight(ctx, b, mqRad, hi)
	}
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), mqRad, 1, edge)
}

// mqSunk is the same plate pressed in: the gradient turned over, a shadow
// where the gloss was, and no highlight at all.
func mqSunk(ctx *paintengine2d.Context, w, h float32, c0, c1, edge paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, mqRad, stop(0, c0), stop(1, c1))
	innerShadow(ctx, b, mqRad, 5, Hex(mqDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), mqRad, 1, edge)
}

// mqWell is a field or a track: sunk, rounded far less than a plate so the
// two never read as the same control, and rimmed in whatever colour says
// what state it is in.
func mqWell(ctx *paintengine2d.Context, w, h float32, fill, rim paintengine2d.Color) {
	r := min(h/2, 6)
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
	innerShadow(ctx, b, r, 4, Hex(mqDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, rim)
}

// mqSheen is the gloss of the era: the top half of a face lifted by a
// white wash that stops dead at the waist, which is what made a plastic
// button of 2001 look wet.
func mqSheen(ctx *paintengine2d.Context, b paintengine2d.Rect, radius float32) {
	ctx.Save()
	ctx.ClipRoundRect(b, radius, radius)
	top := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()*0.5)
	ctx.DrawRect(top, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(top.Min.X, top.Min.Y),
		End:   paintengine2d.Pt(top.Min.X, top.Max.Y),
		Stops: []paintengine2d.GradientStop{stop(0, Hex(mqGlass)), stop(1, Hex(mqGlass).WithAlpha(0))},
	}))
	ctx.Restore()
}

// mqKnob is a slider's grip: a pale plate with a score across its waist, so
// the eye finds its middle without being told.
func mqKnob(ctx *paintengine2d.Context, w, h float32, face, edge paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, 5, stop(0, face), stop(1, Hex("#96a3b8")))
	mqSheen(ctx, b, 5)
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), 5, 1, edge)
	ctx.DrawRect(paintengine2d.XYWH(3, h/2-1, w-6, 1), paintengine2d.Fill(Hex("#00000044")))
	ctx.DrawRect(paintengine2d.XYWH(3, h/2, w-6, 1), paintengine2d.Fill(Hex("#ffffff77")))
}

// mqDisplayWell is the screen the cabinet is built round: a cold gradient
// sunk into the shell, with a hairline of sky along its top edge and the
// shell's own shadow falling into it.
func mqDisplayWell(ctx *paintengine2d.Context, w, h float32) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, 8, stop(0, Hex(mqDeep0)), stop(0.6, Hex("#0b1119")), stop(1, Hex(mqDeep1)))
	innerShadow(ctx, b, 8, 10, Hex("#000000aa"))
	ctx.Save()
	ctx.ClipRoundRect(b, 8, 8)
	ctx.DrawRect(paintengine2d.XYWH(0, 1, w, 1), paintengine2d.Fill(Hex("#3b6da1")))
	ctx.Restore()
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), 8, 1, Hex("#0a0f18"))
}

// mqBand is the caption: brushed steel with a turn at its waist, a
// highlight along the top and the accent rule under an active window's —
// the one place the skin says which window has the keyboard.
func mqBand(ctx *paintengine2d.Context, w, h float32, accent paintengine2d.Color) {
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, 0), End: paintengine2d.Pt(0, h),
		Stops: []paintengine2d.GradientStop{
			stop(0, Hex("#4a5568")), stop(0.48, Hex("#303a4d")),
			stop(0.52, Hex("#28303f")), stop(1, Hex("#1a2029")),
		},
	}))
	px(ctx, 0, 0, w, 1, Hex("#6a788f"))
	if accent.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(10, h-4, w-20, 2), paintengine2d.Fill(accent))
	}
	px(ctx, 0, h-1, w, 1, Hex(mqInk))
}
