package skinart

import "github.com/codemodify/paintengine2d"

// Nocturne is the toolkit's reference skin: amber on charcoal, drawn from
// paths and gradients so it is exact at every scale.
//
// It exists to answer "what does a good skin look like on this toolkit", and
// its design brief was the one the format is built around — nothing here is
// a fixed-size picture of a window. Every face is a nine-slice with fixed
// corners and a stretchable middle, so a button is as wide as its label in
// any language, a panel is as big as its content, and 1.75 is exact rather
// than approximate. The whole skin is one sheet of 56×30 cells.
//
// The look: a face lit from above (a one-pixel highlight inside the top
// edge, a vertical gradient, a dark hairline round it), fields sunk into the
// same surface with the light reversed, and one warm accent that marks
// exactly three things — the default button, what is switched on, and where
// the keyboard is.

// Nocturne's palette. Named for what they are rather than what they look
// like, so the drawing below reads as a description of a surface.
const (
	nocInk     = "#0a0c10" // the hairline round every face
	nocWindow0 = "#171a22" // window background, top
	nocWindow1 = "#101319" // window background, bottom
	nocSurface = "#1b1f29" // bars, panels, menus
	nocPanel   = "#191d26"

	nocFace0 = "#333a49" // a resting face, top
	nocFace1 = "#252a35" // and bottom
	nocHot0  = "#3f4859"
	nocHot1  = "#2e3542"
	nocDown0 = "#191d25"
	nocDown1 = "#232832"
	nocOff0  = "#22262e" // disabled
	nocOff1  = "#1e222a"

	nocSunk  = "#0d0f15" // a field's well
	nocSunkE = "#2c3343" // and its rim

	nocAccent = "#e8a33d"
	nocAccHi  = "#f6bd63"
	nocAccLo  = "#c9822a"
	nocOnAcc  = "#1a1305"

	nocText = "#e6e9f2"
	nocDim  = "#9aa2b6"
	nocGone = "#5b6273"

	nocLight = "#ffffff20" // the one-pixel highlight inside a top edge
	nocDark  = "#00000060" // the shadow inside a sunken top edge
)

// Nocturne cell geometry. One size for every face so the sheet is a grid and
// the corners of every control match each other.
const (
	nocW = 56 // a face cell
	nocH = 30
	nocR = 5 // corner radius
	nocG = 2 // gap between cells, so no bilinear tap can reach a neighbour
)

// nocSlice is the nine-slice of a face cell: enough to hold the radius, the
// border and the highlight, and no more, so the middle carries the stretch.
var nocSlice = [4]int{9, 12, 9, 12}

// Nocturne builds the skin's plan.
func Nocturne() *Plan {
	sh := &Sheet{Name: "chrome"}
	l := &lay{sh: sh, gap: nocG}
	p := &Plan{
		Name:    "nocturne",
		Label:   "Nocturne",
		Year:    2026,
		Lineage: "uitoolkit",
		Summary: "The toolkit's own skin: amber on charcoal, drawn from paths so every corner is exact at 1.75 as well as at 1 and 2.",
		Family:  "dark",
		Base:    "breeze-night",
		Sheets:  []*Sheet{sh},
	}

	// ---- faces ------------------------------------------------------------
	//
	// A raised face and its states. The pressed face inverts the gradient and
	// swaps the highlight for a shadow, which is the whole trick: the eye
	// reads "lit from above" and "lit from below" as "out" and "in".
	raised := func(c0, c1, hi, edge string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			nocFace(ctx, w, h, hex(c0), hex(c1), hex(edge), hex(hi), nocR)
		}
	}

	l.row(nocH)
	l.face("button.normal", raised(nocFace0, nocFace1, nocLight, nocInk))
	l.face("button.hover", raised(nocHot0, nocHot1, "#ffffff2c", nocInk))
	l.face("button.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		nocSunkFace(ctx, w, h, hex(nocDown0), hex(nocDown1), hex(nocInk), nocR)
	})
	l.face("button.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		nocFace(ctx, w, h, hex(nocOff0), hex(nocOff1), hex("#0f1116"), paintengine2d.Color{}, nocR)
	})
	l.face("button.focus", func(ctx *paintengine2d.Context, w, h float32) {
		nocFace(ctx, w, h, hex(nocFace0), hex(nocFace1), hex(nocLight), hex(nocInk), nocR)
		outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), nocR-1, 2, hex(nocAccent).WithAlpha(0.85))
	})
	l.face("button.default", func(ctx *paintengine2d.Context, w, h float32) {
		nocFace(ctx, w, h, hex(nocAccHi), hex(nocAccLo), hex("#a6661c"), hex("#ffffff55"), nocR)
	})
	l.face("button.checked", func(ctx *paintengine2d.Context, w, h float32) {
		nocSunkFace(ctx, w, h, hex("#3a2a12"), hex("#4a3517"), hex("#8a5f1e"), nocR)
	})

	// Fields are the same surface with the light reversed: sunk, with the
	// shadow inside the top edge and no highlight at all.
	l.row(nocH)
	l.field("field.normal", nocSunkE, false)
	l.field("field.hover", "#3b4356", false)
	l.field("field.focus", nocAccent, true)
	l.face("field.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0, 0, w, h)
		ctx.DrawRoundRect(b, nocR, nocR, paintengine2d.Fill(hex("#14171e")))
		outline(ctx, b, nocR, 1, hex("#23272f"))
	})

	// A combo box is a field with a button's face on it: the era's own
	// compromise, and the one that still reads as "you can open this".
	l.row(nocH)
	l.face("combo.normal", raised(nocFace0, nocFace1, nocLight, nocInk))
	l.face("combo.hover", raised(nocHot0, nocHot1, "#ffffff2c", nocInk))
	l.face("combo.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		nocSunkFace(ctx, w, h, hex(nocDown0), hex(nocDown1), hex(nocInk), nocR)
	})
	l.face("combo.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		nocFace(ctx, w, h, hex(nocOff0), hex(nocOff1), hex("#0f1116"), paintengine2d.Color{}, nocR)
	})
	l.face("combo.focus", func(ctx *paintengine2d.Context, w, h float32) {
		nocFace(ctx, w, h, hex(nocFace0), hex(nocFace1), hex(nocLight), hex(nocInk), nocR)
		outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), nocR-1, 2, hex(nocAccent).WithAlpha(0.85))
	})

	// A tool button is flat until it is touched — the toolbar convention
	// since Windows 95's rebar — so its resting state is nothing at all.
	l.row(nocH)
	l.face("tool.normal", func(ctx *paintengine2d.Context, w, h float32) {})
	l.face("tool.hover", func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
		vgrad(ctx, b, nocR, stop(0, hex("#ffffff16")), stop(1, hex("#ffffff08")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), nocR, 1, hex("#ffffff1a"))
	})
	l.face("tool.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0, 0, w, h), nocR, nocR, paintengine2d.Fill(hex("#00000055")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), nocR, 1, hex("#00000066"))
	})
	l.face("tool.disabled", func(ctx *paintengine2d.Context, w, h float32) {})
	l.face("tool.checked", func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0, 0, w, h)
		ctx.DrawRoundRect(b, nocR, nocR, paintengine2d.Fill(hex(nocAccent).WithAlpha(0.20)))
		outline(ctx, b, nocR, 1, hex(nocAccent).WithAlpha(0.55))
	})

	// Tabs: the selected one is a raised face with its bottom edge open, so
	// it joins the pane below instead of floating over it.
	l.row(nocH)
	l.face("tab.normal", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRect(paintengine2d.XYWH(0, h-1, w, 1), paintengine2d.Fill(hex(nocInk)))
	})
	l.face("tab.hover", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h-1), nocR, stop(0, hex("#ffffff12")), stop(1, hex("#ffffff04")))
		ctx.DrawRect(paintengine2d.XYWH(0, h-1, w, 1), paintengine2d.Fill(hex(nocInk)))
	})
	l.face("tab.checked", func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0.5, 0.5, w-1, h+nocR)
		ctx.DrawRoundRectCorners(b, nocR, nocR, 0, 0, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(0, 0), End: paintengine2d.Pt(0, h),
			Stops: []paintengine2d.GradientStop{stop(0, hex("#2c3341")), stop(1, hex(nocSurface))},
		}))
		p := paintengine2d.NewPath()
		p.MoveTo(0.5, h)
		p.LineTo(0.5, nocR)
		p.QuadTo(0.5, 0.5, nocR, 0.5)
		p.LineTo(w-nocR, 0.5)
		p.QuadTo(w-0.5, 0.5, w-0.5, nocR)
		p.LineTo(w-0.5, h)
		ctx.DrawPath(p, paintengine2d.StrokePaint(hex(nocInk), 1))
		// The accent bar along the top is the only mark that says which tab
		// you are on; it reads at a glance and it survives a colour-blind
		// eye, which a hue change alone does not.
		ctx.DrawRect(paintengine2d.XYWH(nocR, 1, w-nocR*2, 2), paintengine2d.Fill(hex(nocAccent)))
	})

	// ---- surfaces ---------------------------------------------------------

	l.row(nocH)
	l.face("menu.hover", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0, 0, w, h), nocR, nocR, paintengine2d.Fill(hex(nocAccent).WithAlpha(0.22)))
	})
	l.face("row.hover", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0, 0, w, h), 4, 4, paintengine2d.Fill(hex("#ffffff0e")))
	})
	l.face("row.checked", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0, 0, w, h), 4, 4, paintengine2d.Fill(hex(nocAccent).WithAlpha(0.26)))
		ctx.DrawRect(paintengine2d.XYWH(0, 2, 2, h-4), paintengine2d.Fill(hex(nocAccent)))
	})
	l.face("panel.normal", func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0, 0, w, h)
		ctx.DrawRoundRect(b, nocR+1, nocR+1, paintengine2d.Fill(hex(nocPanel)))
		outline(ctx, b, nocR+1, 1, hex("#262c38"))
	})
	l.face("bar.normal", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, hex("#20242f")), stop(1, hex(nocSurface)))
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, 1), paintengine2d.Fill(hex(nocLight)))
		ctx.DrawRect(paintengine2d.XYWH(0, h-1, w, 1), paintengine2d.Fill(hex(nocInk)))
	})
	l.face("menu.frame", func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0, 0, w, h)
		ctx.DrawRoundRect(b, nocR+2, nocR+2, paintengine2d.Fill(hex("#1d2129")))
		outline(ctx, b, nocR+2, 1, hex("#333b4a"))
	})
	// A splitter is drawn into a box that may be tall and thin or short and
	// wide, and Face is not told which. So its art is a grip that reads the
	// same either way: a flat seam with a small cluster of dots at its
	// centre, which the nine-slice keeps centred at any length.
	l.face("splitter.normal", func(ctx *paintengine2d.Context, w, h float32) {
		nocGrip(ctx, w, h, hex("#4a5366"))
	})
	l.face("splitter.hover", func(ctx *paintengine2d.Context, w, h float32) {
		nocGrip(ctx, w, h, hex(nocAccent))
	})
	l.face("tooltip.normal", func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0, 0, w, h)
		ctx.DrawRoundRect(b, 4, 4, paintengine2d.Fill(hex("#2a303c")))
		outline(ctx, b, 4, 1, hex("#434c5e"))
	})

	// The window background and the caption band. Both stretch the whole
	// width of a window, so they are gradients with nothing in them that a
	// stretch would distort.
	l.row(48)
	l.cell("window.normal", 64, 48, [4]int{2, 2, 2, 2}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, hex(nocWindow0)), stop(1, hex(nocWindow1)))
	})
	l.cell("caption.normal", 96, 34, [4]int{10, 12, 6, 12}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocCaption(ctx, w, h, hex("#262d3a"), hex("#1a1e27"), hex(nocAccent).WithAlpha(0.55))
	})
	l.cell("caption.inactive", 96, 34, [4]int{10, 12, 6, 12}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocCaption(ctx, w, h, hex("#1d212a"), hex("#171a22"), hex("#00000000"))
	})

	// ---- small parts ------------------------------------------------------

	l.row(22)
	l.cell("thumb.normal", 40, 22, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), (h-2)/2, stop(0, hex("#454e60")), stop(1, hex("#333a48")))
	})
	l.cell("thumb.hover", 40, 22, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), (h-2)/2, stop(0, hex("#576274")), stop(1, hex("#434b5c")))
	})
	l.cell("track.normal", 40, 22, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
		ctx.DrawRoundRect(b, (h-1)/2, (h-1)/2, paintengine2d.Fill(hex(nocSunk)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), h/2, 1, hex("#242a36"))
	})
	l.cell("fill.normal", 40, 22, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), (h-2)/2, stop(0, hex(nocAccHi)), stop(1, hex(nocAccLo)))
	})
	l.cell("knob.normal", 22, 22, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocKnob(ctx, w, h, hex("#eef1f8"), hex("#b9c0d0"))
	})
	l.cell("knob.disabled", 22, 22, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocKnob(ctx, w, h, hex("#767d8d"), hex("#565d6b"))
	})

	// Check boxes and radios. The well is a field; the mark is a glyph the
	// engine tints, so one drawing serves every state.
	l.row(20)
	l.cell("check.normal", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocWell(ctx, w, h, 4, hex(nocSunk), hex(nocSunkE))
	})
	l.cell("check.hover", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocWell(ctx, w, h, 4, hex("#12151c"), hex("#3d465a"))
	})
	l.cell("check.checked", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
		vgrad(ctx, b, 4, stop(0, hex(nocAccHi)), stop(1, hex(nocAccLo)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 4, 1, hex("#9c6620"))
	})
	l.cell("check.disabled", 20, 20, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocWell(ctx, w, h, 4, hex("#14171e"), hex("#262b35"))
	})
	l.cell("radio.normal", 20, 20, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocRadio(ctx, w, h, hex(nocSunk), hex(nocSunkE))
	})
	l.cell("radio.hover", 20, 20, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocRadio(ctx, w, h, hex("#12151c"), hex("#3d465a"))
	})
	l.cell("radio.checked", 20, 20, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocRadio(ctx, w, h, hex(nocAccent), hex("#9c6620"))
	})
	l.cell("radio.disabled", 20, 20, [4]int{0, 0, 0, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		nocRadio(ctx, w, h, hex("#14171e"), hex("#262b35"))
	})
	l.glyph("mark.check", 20, func(ctx *paintengine2d.Context, w, h float32) {
		p := paintengine2d.NewPath()
		p.MoveTo(w*0.26, h*0.52)
		p.LineTo(w*0.44, h*0.70)
		p.LineTo(w*0.76, h*0.31)
		st := paintengine2d.StrokePaint(paintengine2d.RGB(1, 1, 1), 2.4)
		st.Stroke.Cap, st.Stroke.Join = paintengine2d.CapRound, paintengine2d.JoinRound
		ctx.DrawPath(p, st)
	})
	l.glyph("mark.radio", 20, func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawCircle(paintengine2d.Pt(w/2, h/2), min(w, h)*0.19, paintengine2d.Fill(paintengine2d.RGB(1, 1, 1)))
	})

	// Arrows and disclosure glyphs, drawn white and tinted at paint time so
	// one drawing follows whatever colour the label is in.
	l.row(20)
	for _, d := range []string{"down", "up", "left", "right"} {
		dir := d
		l.glyph("arrow."+dir, 20, func(ctx *paintengine2d.Context, w, h float32) {
			chevron(ctx, w, h, dir, 1.8, paintengine2d.RGB(1, 1, 1))
		})
	}
	l.glyph("expander.open", 20, func(ctx *paintengine2d.Context, w, h float32) {
		chevron(ctx, w, h, "down", 1.8, paintengine2d.RGB(1, 1, 1))
	})
	l.glyph("expander.shut", 20, func(ctx *paintengine2d.Context, w, h float32) {
		chevron(ctx, w, h, "right", 1.8, paintengine2d.RGB(1, 1, 1))
	})

	// The focus ring. A skin may re-draw it; it may never remove it, so this
	// sprite is the only thing standing between a skin and a keyboard trap.
	// It is drawn as a frame with an empty middle, so it lays over whatever
	// face it marks instead of covering it.
	l.cell("focus.ring", 24, 24, [4]int{9, 9, 9, 9}, "none", false, func(ctx *paintengine2d.Context, w, h float32) {
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 6, 1, hex(nocAccent).WithAlpha(0.25))
		outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), 5, 2, hex(nocAccent))
	})

	// Switches: a pill track that takes the accent when it is on.
	l.row(24)
	l.cell("switch.off", 44, 24, [4]int{11, 11, 11, 11}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
		ctx.DrawRoundRect(b, (h-1)/2, (h-1)/2, paintengine2d.Fill(hex("#12151c")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), h/2, 1, hex("#394152"))
	})
	l.cell("switch.on", 44, 24, [4]int{11, 11, 11, 11}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
		vgrad(ctx, b, (h-1)/2, stop(0, hex(nocAccHi)), stop(1, hex(nocAccLo)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), h/2, 1, hex("#9c6620"))
	})
	l.cell("switch.disabled", 44, 24, [4]int{11, 11, 11, 11}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
		ctx.DrawRoundRect(b, (h-1)/2, (h-1)/2, paintengine2d.Fill(hex("#15181f")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), h/2, 1, hex("#262b35"))
	})
	l.close()

	// ---- bindings ---------------------------------------------------------

	p.Text = []TextRole{
		{Name: "control", Color: nocText, Hover: "#f2f5fb", Pressed: nocDim, Disabled: nocGone, Checked: nocText},
		{Name: "onAccent", Color: nocOnAcc, Hover: nocOnAcc, Pressed: nocOnAcc, Disabled: nocGone, Checked: nocOnAcc},
		{Name: "field", Color: nocText, Disabled: nocGone},
		{Name: "caption", Color: nocText, Disabled: nocGone},
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
			{"pressed", "tool.pressed"}, {"disabled", "tool.disabled"},
			{"checked", "tool.checked"},
		}},
		{Part: "field", Text: "field", States: [][2]string{
			{"normal", "field.normal"}, {"hover", "field.hover"},
			{"focus", "field.focus"}, {"disabled", "field.disabled"},
		}},
		{Part: "combo", Text: "control", States: [][2]string{
			{"normal", "combo.normal"}, {"hover", "combo.hover"},
			{"pressed", "combo.pressed"}, {"disabled", "combo.disabled"},
			{"focus", "combo.focus"},
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
		{Part: "splitter", States: [][2]string{
			{"normal", "splitter.normal"}, {"hover", "splitter.hover"},
		}},
		{Part: "bar", States: [][2]string{{"normal", "bar.normal"}}},
		{Part: "menu.frame", States: [][2]string{{"normal", "menu.frame"}}},
		{Part: "tooltip", Text: "control", States: [][2]string{{"normal", "tooltip.normal"}}},
		{Part: "window", States: [][2]string{{"normal", "window.normal"}}},
		{Part: "caption", Text: "caption", States: [][2]string{
			{"normal", "caption.normal"}, {"inactive", "caption.inactive"},
		}},
		{Part: "thumb", States: [][2]string{
			{"normal", "thumb.normal"}, {"hover", "thumb.hover"},
		}},
		{Part: "track", States: [][2]string{{"normal", "track.normal"}}},
		{Part: "slider.track", States: [][2]string{{"normal", "track.normal"}}},
		{Part: "slider.fill", States: [][2]string{{"normal", "fill.normal"}}},
		{Part: "slider.thumb", States: [][2]string{
			{"normal", "knob.normal"}, {"disabled", "knob.disabled"},
		}},
		{Part: "progress.back", States: [][2]string{{"normal", "track.normal"}}},
		{Part: "progress.fill", States: [][2]string{{"normal", "fill.normal"}}},
		{Part: "switch.track", Text: "control", States: [][2]string{
			{"normal", "switch.off"}, {"checked", "switch.on"}, {"disabled", "switch.disabled"},
		}},
		{Part: "switch.knob", States: [][2]string{
			{"normal", "knob.normal"}, {"disabled", "knob.disabled"},
		}},
		{Part: "check", Text: "onAccent", States: [][2]string{
			{"normal", "check.normal"}, {"hover", "check.hover"},
			{"checked", "check.checked"}, {"disabled", "check.disabled"},
		}},
		{Part: "check.mark", States: [][2]string{
			{"normal", "mark.check"}, {"disabled", "mark.check"},
		}},
		{Part: "radio.mark", States: [][2]string{
			{"normal", "mark.radio"}, {"disabled", "mark.radio"},
		}},
		{Part: "arrow.down", States: [][2]string{{"normal", "arrow.down"}}},
		{Part: "arrow.up", States: [][2]string{{"normal", "arrow.up"}}},
		{Part: "arrow.left", States: [][2]string{{"normal", "arrow.left"}}},
		{Part: "arrow.right", States: [][2]string{{"normal", "arrow.right"}}},
		{Part: "expander.open", States: [][2]string{{"normal", "expander.open"}}},
		{Part: "expander.shut", States: [][2]string{{"normal", "expander.shut"}}},
		{Part: "focus", States: [][2]string{{"normal", "focus.ring"}}},
	}

	// What the art does not cover: the colours widgets read directly (label
	// ink, text selection, the window's own fill under a partial redraw) and
	// the geometry the art was drawn for.
	p.Colors = map[string]string{
		"background": nocWindow1, "surface": nocSurface, "surfaceAlt": nocPanel,
		"border": "#2b3140", "divider": "#242a36",
		"text": nocText, "textMuted": nocDim, "textOnAccent": nocOnAcc,
		"accent": nocAccent, "accentHover": nocAccHi, "accentPress": nocAccLo,
		"field": nocSunk, "fieldBorder": nocSunkE,
		"focus": nocAccent, "selection": "#e8a33d59",
		"track": nocSunk, "thumb": "#3b4354",
		"menuHover": "#e8a33d3a", "menuHoverBorder": nocAccent, "menuGutter": "#1d222b",
		"highlight": "#ffffff12", "shadow": "#00000073",
		"bevelLight": "#3a4151", "bevelDark": nocInk,
		"danger": "#e06a5a", "success": "#79c08a", "warning": nocAccent,
	}
	p.Metrics = map[string]float32{
		"radius": 5, "radiusSmall": 4, "controlH": 30, "fieldH": 30, "comboH": 30,
		"checkbox": 20, "radio": 20, "scroll": 12, "thumb": 20, "sliderH": 8,
		"progressH": 10, "switchW": 44, "switchH": 24, "focusWidth": 2, "border": 1,
		"rowH": 28, "tabH": 32, "menuItemH": 28, "titleBar": 34,
	}
	p.Window = &WindowSpec{
		Border:  [4]int{1, 1, 1, 1},
		Caption: 34,
		Radius:  [4]int{7, 7, 0, 0},
		// The silhouette this skin would like: the whole window, with the
		// caption's own corners. It is inert until the toolkit can cut a
		// window to a shape (see style/engine_skin_frame.go), and it is
		// stated now so the manifest is complete and the resolver is tested.
		Shape: []ShapeRect{
			{At: [4]int{0, 0, 0, 0}, Radius: [4]int{7, 7, 0, 0}, StretchX: true, StretchY: true},
		},
	}
	return p
}

// field places a text field's well: sunk, rimmed, and with a second accent
// rim where the keyboard is.
func (l *lay) field(name, rim string, focused bool) {
	l.face(name, func(ctx *paintengine2d.Context, w, h float32) {
		nocWell(ctx, w, h, nocR, hex(nocSunk), hex(rim))
		if focused {
			outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), nocR-1, 2, hex(nocAccent).WithAlpha(0.85))
		}
	})
}

// ---- Nocturne's own idioms ------------------------------------------------

// nocFace is a raised face: a vertical gradient, a hairline round it, and a
// one-pixel highlight inside the top edge.
func nocFace(ctx *paintengine2d.Context, w, h float32, c0, c1, edge, hi paintengine2d.Color, r float32) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, r, stop(0, c0), stop(1, c1))
	if hi.A > 0 {
		topLight(ctx, b, r, hi)
	}
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, edge)
}

// nocSunkFace is the same surface pressed in: the gradient reversed and a
// shadow where the highlight was.
func nocSunkFace(ctx *paintengine2d.Context, w, h float32, c0, c1, edge paintengine2d.Color, r float32) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, r, stop(0, c0), stop(1, c1))
	innerShadow(ctx, b, r, 3, hex(nocDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, edge)
}

// nocWell is a field's well: flat, dark, rimmed, shadowed under its top edge.
func nocWell(ctx *paintengine2d.Context, w, h float32, r float32, fill, rim paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
	innerShadow(ctx, b, r, 3, hex(nocDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, rim)
}

// nocRadio is the round form of the same well.
func nocRadio(ctx *paintengine2d.Context, w, h float32, fill, rim paintengine2d.Color) {
	c := paintengine2d.Pt(w/2, h/2)
	rad := min(w, h)/2 - 1
	ctx.DrawCircle(c, rad, paintengine2d.Fill(fill))
	ctx.DrawCircle(c, rad, paintengine2d.StrokePaint(rim, 1))
}

// nocKnob is a slider's and a switch's knob: a lit disc with a rim.
func nocKnob(ctx *paintengine2d.Context, w, h float32, top, bottom paintengine2d.Color) {
	c := paintengine2d.Pt(w/2, h/2)
	rad := min(w, h)/2 - 1.5
	ctx.DrawCircle(c, rad+1, paintengine2d.Fill(hex("#00000055")))
	ctx.DrawCircle(c, rad, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, h/2-rad), End: paintengine2d.Pt(0, h/2+rad),
		Stops: []paintengine2d.GradientStop{stop(0, top), stop(1, bottom)},
	}))
}

// nocGrip is a splitter's handle: a 2×2 cluster of dots on a flat seam,
// which reads as a grip whichever way the splitter runs.
func nocGrip(ctx *paintengine2d.Context, w, h float32, dot paintengine2d.Color) {
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(hex(nocSurface)))
	cx, cy := w/2, h/2
	for _, d := range [][2]float32{{-3, -3}, {1, -3}, {-3, 1}, {1, 1}} {
		ctx.DrawRect(paintengine2d.XYWH(cx+d[0], cy+d[1], 2, 2), paintengine2d.Fill(dot))
	}
}

// nocCaption is the title band: a gradient, a highlight along the top, a
// hairline along the bottom, and a thin accent line under an active window's
// — the one place the skin says "this window has the keyboard".
func nocCaption(ctx *paintengine2d.Context, w, h float32, c0, c1, accent paintengine2d.Color) {
	b := paintengine2d.XYWH(0, 0, w, h)
	ctx.DrawRoundRectCorners(b, 7, 7, 0, 0, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, 0), End: paintengine2d.Pt(0, h),
		Stops: []paintengine2d.GradientStop{stop(0, c0), stop(1, c1)},
	}))
	ctx.Save()
	ctx.ClipRoundRect(b, 7, 7)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, 1), paintengine2d.Fill(hex("#ffffff1e")))
	ctx.Restore()
	if accent.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(0, h-2, w, 1), paintengine2d.Fill(accent))
	}
	ctx.DrawRect(paintengine2d.XYWH(0, h-1, w, 1), paintengine2d.Fill(hex(nocInk)))
}
