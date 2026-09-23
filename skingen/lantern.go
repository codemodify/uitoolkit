package skingen

import "github.com/codemodify/paintengine2d"

// Lantern is the skin the shaped player wears: matte charcoal with one
// indigo light in it, and a window whose skirt is swept round on one side
// and square on the other.
//
// Where Marquee is glass and Minim is pixels, Lantern is matte — no gloss
// anywhere, faces that are a single flat tone inside a soft rim, and all the
// contrast carried by one accent. It is the look a player of the last decade
// arrived at once the cabinets went away.
//
// Its silhouette is one rectangle, and that is the interesting part. It is
// the window's own box with an asymmetric radius — fifty-two design pixels
// at the bottom left and ten at the bottom right — so the desktop sweeps in
// under one corner and barely touches the other. A skin whose "shape" is the
// plain window rounded no more than the frame already rounds it is *not*
// used at all (the frame does that job, and a shape would cost the window
// its resize band); this one asks for four times the frame's radius on one
// corner, so it is a real outline and the toolkit cuts to it.
//
// The corners it leaves alone are the top two, on purpose. Which side a
// window's caption buttons sit on is the *desktop's* choice, not a skin's —
// style.CaptionButtonsDesktop is the default and a skin's "layout" is only
// consulted when the user has asked for the look's own — so a silhouette
// that bit into either top corner would eat a close button on half the
// desktops it ran on. Every shape here keeps out of the caption's way.
//
// It is a *partial* skin, deliberately. It binds seventeen parts and leaves
// tabs, splitters, switches and tree disclosures to breeze-night, because a
// skin is always a partial override and an app that can drop its skin
// entirely — which is the switch the demo wearing this one has — should show
// what a half-drawn one looks like too. Nothing about it is broken; the
// parts it does not draw are simply its base pack's.

// Lantern's palette: charcoal, and one indigo light.
const (
	lnInk    = "#08080e" // the hairline round every face
	lnShell0 = "#2b2b38" // the shell, lit from above
	lnShell1 = "#15151d"
	lnBand0  = "#23232f" // the header band
	lnBand1  = "#191922"

	lnFace0 = "#33333f" // a resting face
	lnFace1 = "#26262f"
	lnHot0  = "#41415a"
	lnHot1  = "#2f2f3f"
	lnDown  = "#1a1a23"
	lnOff   = "#232330"

	lnWell = "#0d0d14" // a field's well
	lnRim  = "#3a3a4c"

	lnAccent = "#8b7cf0"
	lnAccHi  = "#bcb1fb"
	lnAccLo  = "#5b4cc6"
	lnOnAcc  = "#0c0720"

	lnText = "#e9e9f4"
	lnDim  = "#9a9ab2"
	lnGone = "#5c5c72"

	lnLight = "#ffffff16" // the highlight inside a lit top edge
	lnDark  = "#00000066" // the shadow inside a sunken one
)

// Lantern's cell grid and its one radius. Everything here is rounded by the
// same amount, which is most of why it reads as matte rather than moulded.
const (
	lnW   = 72 // a face cell
	lnH   = 30
	lnRad = 6
)

// lnSlice keeps the radius and the hairline at every edge.
var lnSlice = [4]int{10, 10, 10, 10}

// The frame, in design pixels. The two bottom radii are the skirt: the
// sweep on the left and the near-square on the right. The border is wider
// than the sweep reaches at the content's own corner, which is what keeps
// the content inside the cut.
const (
	lnCaption  = 34
	lnRadTop   = 12
	lnSweep    = 52 // the bottom-left corner
	lnClip     = 10 // and the bottom-right one
	lnBezel    = 12 // the border at the right and the top
	lnBezelLft = 18 // and at the left, where the sweep comes in
	lnFoot     = 20 // and at the bottom
)

// Lantern builds the skin's plan.
func Lantern() *Plan {
	sh := &Sheet{Name: "chrome"}
	l := &lay{sh: sh, gap: 2, faceW: lnW, faceH: lnH, faceSlice: lnSlice}
	p := &Plan{
		Name:    "lantern",
		Label:   "Lantern",
		Year:    2026,
		Lineage: "uitoolkit",
		Summary: "Matte charcoal with one indigo light, and a window with a notch bitten out of its top-right corner — a deliberately partial skin over breeze-night.",
		Family:  "dark",
		Base:    "breeze-night",
		Sheets:  []*Sheet{sh},
	}

	// ---- faces ------------------------------------------------------------
	matte := func(c0, c1, edge, hi string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			lnPlate(ctx, w, h, Hex(c0), Hex(c1), Hex(edge), Hex(hi))
		}
	}
	l.row(lnH)
	l.face("button.normal", matte(lnFace0, lnFace1, lnInk, lnLight))
	l.face("button.hover", matte(lnHot0, lnHot1, lnInk, "#ffffff24"))
	l.face("button.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		lnSunk(ctx, w, h, Hex(lnDown), Hex("#20202a"), Hex(lnInk))
	})
	l.face("button.disabled", matte(lnOff, "#1e1e28", "#101017", "#00000000"))
	l.face("button.focus", func(ctx *paintengine2d.Context, w, h float32) {
		lnPlate(ctx, w, h, Hex(lnFace0), Hex(lnFace1), Hex(lnInk), Hex(lnLight))
		outline(ctx, paintengine2d.XYWH(1.5, 1.5, w-3, h-3), lnRad-1, 2, Hex(lnAccent))
	})
	l.face("button.default", matte(lnAccent, lnAccLo, "#3d3196", "#ffffff44"))
	l.face("button.checked", func(ctx *paintengine2d.Context, w, h float32) {
		lnSunk(ctx, w, h, Hex("#2a2258"), Hex("#332a68"), Hex(lnAccLo))
	})

	l.row(lnH)
	l.face("field.normal", func(ctx *paintengine2d.Context, w, h float32) {
		lnWellFace(ctx, w, h, Hex(lnWell), Hex(lnRim))
	})
	l.face("field.focus", func(ctx *paintengine2d.Context, w, h float32) {
		lnWellFace(ctx, w, h, Hex(lnWell), Hex(lnAccent))
	})
	l.face("field.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		lnWellFace(ctx, w, h, Hex("#101017"), Hex("#2b2b39"))
	})
	l.face("combo.normal", matte(lnFace0, lnFace1, lnInk, lnLight))
	l.face("combo.hover", matte(lnHot0, lnHot1, lnInk, "#ffffff24"))
	l.face("combo.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		lnSunk(ctx, w, h, Hex(lnDown), Hex("#20202a"), Hex(lnInk))
	})

	// A tool button is nothing at rest: the transport row of this look is a
	// line of glyphs on the shell, and a row of boxes would be a toolbar.
	l.row(lnH)
	l.face("tool.normal", func(ctx *paintengine2d.Context, w, h float32) {})
	l.face("tool.hover", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), lnRad, lnRad,
			paintengine2d.Fill(Hex("#ffffff14")))
	})
	l.face("tool.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), lnRad, lnRad,
			paintengine2d.Fill(Hex("#00000055")))
	})
	l.face("tool.checked", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), lnRad, lnRad,
			paintengine2d.Fill(Hex("#2a2258")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), lnRad, 1, Hex(lnAccLo))
	})
	l.face("row.hover", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(Hex("#232331")))
	})
	l.face("row.checked", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(Hex("#2a2258")))
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 3, h), paintengine2d.Fill(Hex(lnAccent)))
	})
	l.face("menu.hover", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 4, 4,
			paintengine2d.Fill(Hex("#332a68")))
	})

	l.row(lnH)
	l.face("panel.normal", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 8, 8, paintengine2d.Fill(Hex("#1f1f29")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 8, 1, Hex("#32323f"))
	})
	l.face("bar.normal", func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, Hex(lnBand0)), stop(1, Hex(lnBand1)))
		px(ctx, 0, h-1, w, 1, Hex(lnInk))
	})
	l.face("menu.frame", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 8, 8, paintengine2d.Fill(Hex("#20202b")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 8, 1, Hex("#3a3a4c"))
	})
	l.face("tooltip.normal", func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 6, 6, paintengine2d.Fill(Hex("#12121a")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 6, 1, Hex(lnAccLo))
	})

	// ---- the frame --------------------------------------------------------
	//
	// The window art is the shell, one gradient over the whole box, which
	// the silhouette then cuts. It carries no rim: the outline's own edge is
	// the rim, and a line at the art's edge would be cut away with it.
	l.row(72)
	l.cell("window.normal", 56, 72, [4]int{10, 10, 10, 10}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0,
			stop(0, Hex(lnShell0)), stop(0.4, Hex("#1f1f29")), stop(1, Hex(lnShell1)))
	})
	l.cell("caption.normal", 96, lnCaption, [4]int{10, 14, 4, 14}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnBandArt(ctx, w, h, Hex(lnAccent))
	})
	l.cell("caption.inactive", 96, lnCaption, [4]int{10, 14, 4, 14}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnBandArt(ctx, w, h, paintengine2d.Color{})
	})
	// The caption buttons are discs, and the caption is short, so a disc is
	// what a square box at this height actually is — which is also how the
	// toolkit works out where the pointer really has to be.
	l.row(20)
	for _, s := range []struct{ name, fill, edge string }{
		{"capbtn.normal", "#3a3a4c", lnInk},
		{"capbtn.hover", "#5b4cc6", "#3d3196"},
		{"capbtn.pressed", "#2a2258", "#1b1540"},
	} {
		cell := s
		l.cell(cell.name, 20, 20, [4]int{9, 0, 9, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
			r := min(w, h) / 2
			ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), r, r, paintengine2d.Fill(Hex(cell.fill)))
			outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, Hex(cell.edge))
		})
	}

	// ---- the small parts --------------------------------------------------

	l.row(18)
	l.cell("thumb.normal", 26, 18, [4]int{7, 7, 7, 7}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnPlate(ctx, w, h, Hex(lnFace0), Hex(lnFace1), Hex(lnInk), Hex(lnLight))
	})
	l.cell("thumb.hover", 26, 18, [4]int{7, 7, 7, 7}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnPlate(ctx, w, h, Hex(lnHot0), Hex(lnHot1), Hex(lnInk), Hex("#ffffff24"))
	})
	l.cell("track.normal", 26, 18, [4]int{7, 7, 7, 7}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnWellFace(ctx, w, h, Hex("#121219"), Hex("#2b2b39"))
	})
	// The seek bar: a shallow slot with square-cut ends, because the fill
	// that runs in it is square-cut too and a round fill in a round slot
	// leaves a crescent of track at either end.
	l.cell("slot.normal", 26, 8, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 3, 3, paintengine2d.Fill(Hex(lnWell)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 3, 1, Hex("#2f2f3e"))
	})
	l.cell("slot.fill", 26, 8, [4]int{3, 3, 3, 3}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 3, 3, paintengine2d.Fill(Hex(lnAccent)))
		px(ctx, 1, 1, w-2, 1, Hex(lnAccHi))
	})
	l.cell("knob.normal", 14, 18, [4]int{7, 0, 7, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnKnob(ctx, w, h, Hex("#e2e2ef"), Hex("#7b7b96"))
	})
	l.cell("knob.disabled", 14, 18, [4]int{7, 0, 7, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnKnob(ctx, w, h, Hex("#5a5a6d"), Hex("#2f2f3d"))
	})

	l.row(18)
	l.cell("check.normal", 18, 18, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnWellFace(ctx, w, h, Hex(lnWell), Hex(lnRim))
	})
	l.cell("check.hover", 18, 18, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnWellFace(ctx, w, h, Hex(lnWell), Hex(lnAccent))
	})
	l.cell("check.checked", 18, 18, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 4, 4, paintengine2d.Fill(Hex(lnAccent)))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 4, 1, Hex("#3d3196"))
	})
	l.cell("check.disabled", 18, 18, [4]int{6, 6, 6, 6}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		lnWellFace(ctx, w, h, Hex("#101017"), Hex("#2b2b39"))
	})
	l.glyph("mark.check", 18, func(ctx *paintengine2d.Context, w, h float32) {
		p := paintengine2d.NewPath()
		p.MoveTo(4.5, 9.5)
		p.LineTo(7.5, 12.5)
		p.LineTo(13.5, 5.5)
		st := paintengine2d.StrokePaint(paintengine2d.RGB(1, 1, 1), 2.2)
		st.Stroke.Cap = paintengine2d.CapRound
		st.Stroke.Join = paintengine2d.JoinRound
		ctx.DrawPath(p, st)
	})
	l.glyph("mark.radio", 18, func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawCircle(paintengine2d.Pt(w/2, h/2), 3.2, paintengine2d.Fill(paintengine2d.RGB(1, 1, 1)))
	})

	l.row(18)
	for _, d := range []string{"down", "up", "left", "right"} {
		dir := d
		l.glyph("arrow."+dir, 18, func(ctx *paintengine2d.Context, w, h float32) {
			chevron(ctx, w, h, dir, 1.8, paintengine2d.RGB(1, 1, 1))
		})
	}
	l.cell("focus.ring", 22, 22, [4]int{8, 8, 8, 8}, "none", false, func(ctx *paintengine2d.Context, w, h float32) {
		outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), 6, 2, Hex(lnAccent))
	})
	l.close()

	// ---- bindings ---------------------------------------------------------

	p.Text = []TextRole{
		{Name: "control", Color: lnText, Hover: "#ffffff", Pressed: lnDim,
			Disabled: lnGone, Checked: lnAccHi, Default: lnOnAcc},
		{Name: "caption", Color: lnText, Disabled: lnDim, Bold: true},
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
		{Part: "row", Text: "control", States: [][2]string{
			{"normal", "row.hover"}, {"hover", "row.hover"}, {"checked", "row.checked"},
		}},
		{Part: "menu", Text: "control", States: [][2]string{
			{"normal", "menu.hover"}, {"hover", "menu.hover"},
		}},
		{Part: "panel", States: [][2]string{{"normal", "panel.normal"}}},
		{Part: "bar", States: [][2]string{{"normal", "bar.normal"}}},
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
		{Part: "focus", States: [][2]string{{"normal", "focus.ring"}}},
	}
	p.Colors = map[string]string{
		"background": "#1f1f29", "surface": "#26262f", "surfaceAlt": "#1b1b24",
		"border": "#32323f", "divider": "#2a2a36",
		"text": lnText, "textMuted": lnDim, "textOnAccent": lnOnAcc,
		"accent": lnAccent, "accentHover": lnAccHi, "accentPress": lnAccLo,
		"field": lnWell, "fieldBorder": lnRim,
		"focus": lnAccent, "selection": "#8b7cf055",
		"track": "#121219", "thumb": "#3a3a4c",
		"menuHover": "#332a68", "menuHoverBorder": lnAccent, "menuGutter": "#1b1b24",
		"highlight": "#ffffff12", "shadow": "#00000088",
		"bevelLight": "#43435a", "bevelDark": lnInk,
		"danger": "#e0705e", "success": "#76c89b", "warning": "#e0b24a",
	}
	p.Metrics = map[string]float32{
		"radius": lnRad, "radiusSmall": 4, "controlH": 30, "fieldH": 30, "comboH": 30,
		"checkbox": 18, "radio": 18, "scroll": 12, "thumb": 18, "sliderH": 8,
		"progressH": 8, "switchW": 40, "switchH": 22, "focusWidth": 2, "border": 1,
		"rowH": 26, "tabH": 30, "menuItemH": 26, "titleBar": lnCaption,
	}
	// The frame, and the skirt: one rect, the window's own box, with four
	// times the frame's radius on one bottom corner and a tenth of the
	// window's height on neither top one.
	p.Window = &WindowSpec{
		Border:  [4]int{0, lnBezel, lnFoot, lnBezelLft},
		Caption: lnCaption,
		Layout:  ":minimize,maximize,close",
		Radius:  [4]int{lnRadTop, lnRadTop, 0, 0},
		Shape: []ShapeRect{
			{At: [4]int{0, 0, 0, 0}, Radius: [4]int{lnRadTop, lnRadTop, lnClip, lnSweep},
				StretchX: true, StretchY: true},
		},
	}
	return p
}

// ---- Lantern's own idioms ---------------------------------------------------

// lnPlate is every raised face: a shallow vertical gradient in a rounded
// rect, a one-pixel highlight inside the top edge and a hairline round it.
// No gloss — that is the whole difference between this look and Marquee's.
func lnPlate(ctx *paintengine2d.Context, w, h float32, c0, c1, edge, hi paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, lnRad, stop(0, c0), stop(1, c1))
	if hi.A > 0 {
		topLight(ctx, b, lnRad, hi)
	}
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), lnRad, 1, edge)
}

// lnSunk is the same plate pressed in: the gradient turned over and a
// shadow where the highlight was.
func lnSunk(ctx *paintengine2d.Context, w, h float32, c0, c1, edge paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, lnRad, stop(0, c0), stop(1, c1))
	innerShadow(ctx, b, lnRad, 4, Hex(lnDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), lnRad, 1, edge)
}

// lnWellFace is a field, a track or a check box's well: sunk, rounded less
// than a plate, and rimmed in whatever colour says its state.
func lnWellFace(ctx *paintengine2d.Context, w, h float32, fill, rim paintengine2d.Color) {
	r := min(h/2, 4)
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
	innerShadow(ctx, b, r, 3, Hex(lnDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, rim)
}

// lnKnob is a slider's grip: a pale stadium with a score across its waist.
func lnKnob(ctx *paintengine2d.Context, w, h float32, face, edge paintengine2d.Color) {
	r := min(w, h) / 2
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, r, stop(0, face), stop(1, Hex("#b4b4c8")))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, edge)
	ctx.DrawRect(paintengine2d.XYWH(3, h/2-1, w-6, 1), paintengine2d.Fill(Hex("#00000033")))
}

// lnBandArt is the header band: a shallow gradient with the accent rule
// under an active window's title and nothing under an inactive one's — the
// one place the skin says which window has the keyboard.
func lnBandArt(ctx *paintengine2d.Context, w, h float32, accent paintengine2d.Color) {
	vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, Hex(lnBand0)), stop(1, Hex(lnBand1)))
	px(ctx, 0, 0, w, 1, Hex("#3a3a4c"))
	if accent.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(0, h-2, w, 2), paintengine2d.Fill(accent))
	} else {
		px(ctx, 0, h-1, w, 1, Hex(lnInk))
	}
}
