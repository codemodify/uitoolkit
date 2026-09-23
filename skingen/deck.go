package skingen

import "github.com/codemodify/paintengine2d"

// Deck is the toolkit's shaped skin: a window that is not a rectangle, and
// controls whose hit area is their own artwork.
//
// Nocturne shows what a skin looks like and Cassette shows what a pixel
// sheet does at a fractional scale. Deck is here for the half neither can
// reach: a skin declares its window's silhouette (the "shape" block) and the
// toolkit cuts the window to it, so the desktop shows through beside it and
// a click there lands on whatever is behind.
//
// The outline is a deck's: a full-width shoulder across the top, with the
// title plate inlaid in it, and a narrower body under it — so the desktop
// steps in under both shoulders and the corners are round. It is stated as
// two rounded rectangles that stretch with the window, which is why it is
// exact at 1.25, 1.5 and 1.75 instead of resampled, and why the window can
// be dragged to any size at all.
//
// The controls are the other half. Deck's push button is a stadium: fully
// round caps, a flat middle that stretches, and nothing in the corners of
// its box. The corners are not the button, and with the art's own alpha
// deciding, a press there falls through to whatever is behind it — which is
// what a round skinned button looked like it should do since the first one
// was drawn.
//
// It is a *partial* skin on purpose. It binds the window, the caption, the
// push button, the field, the panel and the bars, and leaves everything else
// to breeze-night: a skin is always a partial override, and a shaped skin is
// no different.

// Deck's palette: graphite and one cold accent.
const (
	deckInk    = "#070a0e" // the hairline round every face
	deckShell0 = "#2f3740" // the shell, lit from above
	deckShell1 = "#191e25"
	deckPlate0 = "#12171d" // the title plate inlaid in the shoulder
	deckPlate1 = "#0a0e13"

	deckFace0 = "#39424e" // a resting face, top
	deckFace1 = "#272e38" // and bottom
	deckHot0  = "#48535f"
	deckHot1  = "#323a45"
	deckDown0 = "#1a1f26"
	deckDown1 = "#242a33"
	deckOff0  = "#242931"
	deckOff1  = "#1f242b"

	deckSunk  = "#0b0f14" // a field's well
	deckSunkE = "#333c47"

	deckAccent = "#3fd0c9"
	deckAccHi  = "#82f0eb"
	deckAccLo  = "#28a8a2"
	deckOnAcc  = "#042120"

	deckText = "#e8eef2"
	deckDim  = "#94a2ad"
	deckGone = "#5a646f"

	deckLight = "#ffffff1f" // the highlight inside a lit top edge
	deckDark  = "#00000066" // the shadow inside a sunken one
)

// Deck's geometry, in design pixels. The button cell is exactly twice as
// wide as it is tall so the sheet shows the stadium whole: its two caps are
// half-circles of the cell's height, and the nine-slice keeps them at that
// size however wide the button gets.
const (
	deckW   = 72 // a face cell
	deckH   = 36
	deckCap = 18 // the stadium's cap: half the cell's height
	deckG   = 2  // gap between cells

	// The frame. The shoulder is the caption band's row and runs the whole
	// width; the body is drawn in from it on both sides by the waist, and
	// the border is a little wider still, so the content keeps a margin of
	// shell between it and the cut rather than running to the edge of it.
	deckShoulder = 46
	deckWaist    = 18
	deckBezel    = 26
	deckRadTop   = 24
	deckRadBot   = 22
)

// deckSlice is a stadium's nine-slice: the caps fixed, the middle carrying
// every bit of the stretch, and no fixed rows at all — a stadium is the same
// all the way down.
var deckSlice = [4]int{0, deckCap, 0, deckCap}

// Deck builds the skin's plan.
func Deck() *Plan {
	sh := &Sheet{Name: "chrome"}
	l := &lay{sh: sh, gap: deckG, faceW: deckW, faceH: deckH, faceSlice: deckSlice}
	p := &Plan{
		Name:    "deck",
		Label:   "Deck",
		Year:    2026,
		Lineage: "uitoolkit",
		Summary: "The toolkit's shaped skin: a window cut to a deck's outline, with the desktop beside it, and round buttons that take the pointer only on their ink.",
		Family:  "dark",
		Base:    "breeze-night",
		Sheets:  []*Sheet{sh},
	}

	// ---- the push button: a stadium ---------------------------------------
	//
	// Every one of these leaves the four corners of its cell empty, and that
	// emptiness is the point: it is what the toolkit reads to decide where
	// the button is.
	stadium := func(c0, c1, edge, hi string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			deckPill(ctx, w, h, Hex(c0), Hex(c1), Hex(edge), Hex(hi))
		}
	}
	l.row(deckH)
	l.face("button.normal", stadium(deckFace0, deckFace1, deckInk, deckLight))
	l.face("button.hover", stadium(deckHot0, deckHot1, deckInk, "#ffffff2e"))
	l.face("button.pressed", func(ctx *paintengine2d.Context, w, h float32) {
		deckPillSunk(ctx, w, h, Hex(deckDown0), Hex(deckDown1), Hex(deckInk))
	})
	l.face("button.disabled", stadium(deckOff0, deckOff1, "#0f1319", "#00000000"))
	l.face("button.focus", func(ctx *paintengine2d.Context, w, h float32) {
		deckPill(ctx, w, h, Hex(deckFace0), Hex(deckFace1), Hex(deckInk), Hex(deckLight))
		deckPillRing(ctx, w, h, 2, Hex(deckAccent).WithAlpha(0.9))
	})
	l.face("button.default", stadium(deckAccHi, deckAccLo, "#1d7873", "#ffffff5a"))
	l.face("button.checked", func(ctx *paintengine2d.Context, w, h float32) {
		deckPillSunk(ctx, w, h, Hex("#123734"), Hex("#18453f"), Hex("#1d7873"))
	})

	// ---- fields, panels and bars ------------------------------------------
	l.row(deckH)
	l.face("field.normal", func(ctx *paintengine2d.Context, w, h float32) {
		deckWell(ctx, w, h, Hex(deckSunk), Hex(deckSunkE))
	})
	l.face("field.focus", func(ctx *paintengine2d.Context, w, h float32) {
		deckWell(ctx, w, h, Hex(deckSunk), Hex(deckAccent))
	})
	l.face("field.disabled", func(ctx *paintengine2d.Context, w, h float32) {
		deckWell(ctx, w, h, Hex("#0d1116"), Hex("#262c35"))
	})

	l.row(32)
	l.cell("panel.normal", 40, 32, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 6, 6, paintengine2d.Fill(Hex("#1e242c")))
		outline(ctx, paintengine2d.XYWH(0, 0, w, h), 6, 1, Hex("#343d48"))
	})
	l.cell("bar.normal", 40, 32, [4]int{2, 4, 2, 4}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, Hex("#252c35")), stop(1, Hex("#1c222a")))
		px(ctx, 0, h-1, w, 1, Hex(deckInk))
	})
	// The focus ring: a cap-radius outline that fits round the stadium as
	// well as round anything square, because a skin may re-draw the ring but
	// never remove it.
	l.cell("focus.ring", 40, 40, [4]int{16, 16, 16, 16}, "none", false, func(ctx *paintengine2d.Context, w, h float32) {
		outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), 14, 2, Hex(deckAccent).WithAlpha(0.9))
	})

	// ---- the frame --------------------------------------------------------
	//
	// The window art is the shell: one gradient across the whole window box,
	// which the silhouette then cuts to the deck's outline. It carries no rim
	// of its own — the outline's own edge is the rim, and a line drawn at the
	// art's edge would be cut away with the pixels beside it.
	l.row(64)
	l.cell("window.normal", 48, 64, [4]int{8, 8, 8, 8}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		vgrad(ctx, paintengine2d.XYWH(0, 0, w, h), 0, stop(0, Hex(deckShell0)), stop(0.35, Hex("#232932")), stop(1, Hex(deckShell1)))
	})
	// The title plate, inlaid in the shoulder: darker than the shell it sits
	// in, with the accent line under an active window's — the one place the
	// skin says which window has the keyboard.
	l.cell("caption.normal", 96, deckShoulder, [4]int{16, 16, 5, 16}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		deckPlate(ctx, w, h, Hex(deckAccent))
	})
	l.cell("caption.inactive", 96, deckShoulder, [4]int{16, 16, 5, 16}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
		deckPlate(ctx, w, h, paintengine2d.Color{})
	})

	// The caption buttons are the same stadium stood on end. A caption
	// button's box is as tall as the band, which is taller than it is wide,
	// so a disc drawn there would be stretched into an ellipse; a stadium
	// sliced across its waist keeps its round ends whatever the band's
	// height turns out to be.
	l.row(28)
	for _, s := range []struct{ name, c0, c1, edge string }{
		{"capbtn.normal", "#39424e", "#272e38", deckInk},
		{"capbtn.hover", "#4c5866", "#333c47", deckInk},
		{"capbtn.pressed", "#1c2128", "#262d36", deckInk},
	} {
		cell := s
		l.cell(cell.name, 24, 28, [4]int{12, 0, 12, 0}, "", false, func(ctx *paintengine2d.Context, w, h float32) {
			deckCapsule(ctx, w, h, Hex(cell.c0), Hex(cell.c1), Hex(cell.edge))
		})
	}
	l.close()

	// ---- bindings ---------------------------------------------------------

	p.Text = []TextRole{
		{Name: "control", Color: deckText, Hover: "#ffffff", Pressed: deckDim,
			Disabled: deckGone, Checked: deckAccHi, Default: deckOnAcc},
		{Name: "caption", Color: deckText, Disabled: deckDim, Bold: true},
	}
	p.Parts = []PartBinding{
		{Part: "button", Text: "control", States: [][2]string{
			{"normal", "button.normal"}, {"hover", "button.hover"},
			{"pressed", "button.pressed"}, {"disabled", "button.disabled"},
			{"focus", "button.focus"}, {"checked", "button.checked"},
			{"default", "button.default"},
		}},
		{Part: "field", Text: "control", States: [][2]string{
			{"normal", "field.normal"}, {"focus", "field.focus"},
			{"disabled", "field.disabled"},
		}},
		{Part: "panel", States: [][2]string{{"normal", "panel.normal"}}},
		{Part: "bar", States: [][2]string{{"normal", "bar.normal"}}},
		{Part: "focus", States: [][2]string{{"normal", "focus.ring"}}},
		{Part: "window", States: [][2]string{{"normal", "window.normal"}}},
		{Part: "caption", Text: "caption", States: [][2]string{
			{"normal", "caption.normal"}, {"inactive", "caption.inactive"},
		}},
		{Part: "caption.button", Text: "control", States: [][2]string{
			{"normal", "capbtn.normal"}, {"hover", "capbtn.hover"},
			{"pressed", "capbtn.pressed"},
		}},
	}
	p.Colors = map[string]string{
		"background": "#1e242c", "surface": "#232932", "surfaceAlt": "#1b2028",
		"border": "#343d48", "divider": "#2a323c",
		"text": deckText, "textMuted": deckDim, "textOnAccent": deckOnAcc,
		"accent": deckAccent, "accentHover": deckAccHi, "accentPress": deckAccLo,
		"field": deckSunk, "fieldBorder": deckSunkE,
		"focus": deckAccent, "selection": "#3fd0c959",
		"track": deckSunk, "thumb": "#3b4553",
		"menuHover": "#3fd0c93a", "menuHoverBorder": deckAccent, "menuGutter": "#1d232b",
		"highlight": "#ffffff12", "shadow": "#00000080",
		"bevelLight": "#3d4756", "bevelDark": deckInk,
		"danger": "#e0705e", "success": "#76c89b", "warning": "#e0b24a",
	}
	p.Metrics = map[string]float32{
		"radius": 8, "radiusSmall": 6, "controlH": 36, "fieldH": 36, "comboH": 36,
		"checkbox": 20, "radio": 20, "scroll": 12, "thumb": 20, "sliderH": 8,
		"progressH": 10, "switchW": 44, "switchH": 24, "focusWidth": 2, "border": 1,
		"rowH": 28, "tabH": 34, "menuItemH": 28, "titleBar": deckShoulder,
	}
	// The frame, and the outline the window is cut to.
	//
	// The waist is how far the body is drawn in from the shoulder above it,
	// and the bezel is the border, eight pixels wider, so the content keeps
	// a margin of shell inside the cut. The two rectangles overlap by six
	// pixels so their union has no seam.
	p.Window = &WindowSpec{
		Border:  [4]int{0, deckBezel, deckBezel, deckBezel},
		Caption: deckShoulder,
		Layout:  ":minimize,maximize,close",
		Radius:  [4]int{deckRadTop, deckRadTop, 0, 0},
		Shape: []ShapeRect{
			{At: [4]int{0, 0, 0, deckShoulder}, Radius: [4]int{deckRadTop, deckRadTop, 0, 0}, StretchX: true},
			{At: [4]int{deckWaist, deckShoulder - 6, deckWaist, 0},
				Radius: [4]int{0, 0, deckRadBot, deckRadBot}, StretchX: true, StretchY: true},
		},
	}
	return p
}

// ---- Deck's own idioms ------------------------------------------------------

// deckPill is the stadium every push button is: a vertical gradient inside
// fully round caps, a hairline round it, and a highlight inside the top edge.
func deckPill(ctx *paintengine2d.Context, w, h float32, c0, c1, edge, hi paintengine2d.Color) {
	r := h / 2
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, r, stop(0, c0), stop(1, c1))
	if hi.A > 0 {
		topLight(ctx, b, r, hi)
	}
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, edge)
}

// deckPillSunk is the same stadium pressed in: the gradient turned over and
// a shadow where the highlight was.
func deckPillSunk(ctx *paintengine2d.Context, w, h float32, c0, c1, edge paintengine2d.Color) {
	r := h / 2
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, r, stop(0, c0), stop(1, c1))
	innerShadow(ctx, b, r, 4, Hex(deckDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, edge)
}

// deckPillRing is the accent ring round a focused stadium.
func deckPillRing(ctx *paintengine2d.Context, w, h, width float32, col paintengine2d.Color) {
	outline(ctx, paintengine2d.XYWH(1, 1, w-2, h-2), h/2-1, width, col)
}

// deckWell is a field: the same surface sunk, rimmed, and rounded far less
// than a button so the two never read as the same control.
func deckWell(ctx *paintengine2d.Context, w, h float32, fill, rim paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	ctx.DrawRoundRect(b, 7, 7, paintengine2d.Fill(fill))
	innerShadow(ctx, b, 7, 4, Hex(deckDark))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), 7, 1, rim)
}

// deckCapsule is a caption button: the stadium stood on end, lit from above
// with a hairline round it.
func deckCapsule(ctx *paintengine2d.Context, w, h float32, c0, c1, edge paintengine2d.Color) {
	r := w / 2
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	vgrad(ctx, b, r, stop(0, c0), stop(1, c1))
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), r, 1, edge)
}

// deckPlate is the title plate inlaid in the shoulder: a dark rounded slab
// with a highlight along its top, and an accent rule along its bottom when
// the window is the active one.
func deckPlate(ctx *paintengine2d.Context, w, h float32, accent paintengine2d.Color) {
	b := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
	ctx.DrawRoundRectCorners(b, 12, 12, 4, 4, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, 0), End: paintengine2d.Pt(0, h),
		Stops: []paintengine2d.GradientStop{stop(0, Hex(deckPlate0)), stop(1, Hex(deckPlate1))},
	}))
	ctx.Save()
	ctx.ClipRoundRect(b, 12, 12)
	ctx.DrawRect(paintengine2d.XYWH(0, 1, w, 1), paintengine2d.Fill(Hex("#ffffff18")))
	ctx.Restore()
	if accent.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(6, h-3, w-12, 2), paintengine2d.Fill(accent))
	}
	outline(ctx, paintengine2d.XYWH(0, 0, w, h), 12, 1, Hex(deckInk))
}
