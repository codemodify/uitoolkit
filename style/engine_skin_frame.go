package style

import (
	"math"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// The window frame a skin paints (see engine_skin.go).
//
// Every engine in this package states its own top-level frame, and a skin is
// no exception: its caption is where a skinned app is most obviously
// *itself*. The skin's "window" block gives the border, the caption height
// and the button layout in design pixels; the "caption" part gives the band
// its art and the "button" part gives the caption buttons theirs, so a skin
// that says nothing about windows still gets its base pack's frame and a
// skin that says everything gets its own.
//
// The frame need not be a rectangle. The "window" block's "shape" is the
// silhouette the window is cut to, and WindowShape at the bottom of this
// file is what hands it to the frame — under the same rules that drop the
// corners and the shadow.

// skinCaptionMin is the shortest caption a skin may ask for: the frame tests
// require at least 8 device pixels of caption on every pack at every scale,
// and a caption below that cannot hold a button anyway.
const skinCaptionMin = 8

func (skinEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	sk := skinFor(l)
	base := skinBaseDecoration(l, st)
	w := sk.windowFor(st.Role)
	if w == nil {
		return base
	}
	s := l.Scale() / sk.Design.Scale
	spec := base
	if w.Layout != "" {
		spec.Layout = w.Layout
	}
	// The border and caption are the skin's own geometry, snapped to whole
	// device pixels: a frame measured in fractions leaks a seam between the
	// border and the client at 1.25 and 1.75.
	whole := func(in Insets) Insets {
		return Insets{
			Top:    skinWhole(in.Top * s),
			Right:  skinWhole(in.Right * s),
			Bottom: skinWhole(in.Bottom * s),
			Left:   skinWhole(in.Left * s),
		}
	}
	if !w.Border.Zero() || w.Split {
		spec.Border = whole(w.Border)
	}
	if w.Split {
		spec.ContentBorder, spec.Split = whole(w.ContentBorder), true
	}
	spec.CaptionGap = skinWhole(w.CaptionGap * s)
	spec.ButtonSide = w.Buttons
	if w.Caption > 0 {
		spec.Caption = max(skinWhole(w.Caption*s), l.S(skinCaptionMin))
		// A caption the skin resized has to resize its buttons with it, or
		// the art and the hit boxes disagree.
		side := max(skinWhole(spec.Caption*0.62), l.S(skinCaptionMin))
		spec.Button = paintengine2d.Pt(side, 0)
		spec.CloseButton = paintengine2d.Point{}
		spec.ButtonPad = Insets{Right: skinWhole(spec.Caption * 0.2), Left: skinWhole(spec.Caption * 0.2)}
		if full := float32(math.Round(float64(l.S(24)))); spec.Caption < full {
			// A button with no height of its own is the caption's height,
			// and a frame gives such a button at least 24 pixels to be —
			// which, under a band shorter than that, grows the band rather
			// than the button. A skin that states a short caption means
			// it, so its buttons are square and stood in the middle of it.
			spec.Button.Y = side
			spec.ButtonPad.Top = float32(math.Floor(float64(spec.Caption-side) * 0.5))
		}
		spec.ButtonGap = skinWhole(spec.Caption * 0.12)
		spec.CenterButtons = true
	}
	if w.Radius != [4]float32{} {
		spec.Radius = [4]float32{
			w.Radius[0] * s, w.Radius[1] * s,
			w.Radius[2] * s, w.Radius[3] * s,
		}
	}
	if sk.framePart("caption", st.Role).art("normal") != nil {
		// A skin that paints its own caption band keeps the band stacked
		// above the app's title bar: the art is a picture of a title bar,
		// and merging it under an app's own header would draw two.
		spec.Stacked = true
	}
	return spec
}

// windowFor is the frame the skin gives a window the app calls role: that
// role's variant where the skin states one, its one window block otherwise
// (nil for a skin that says nothing about windows).
//
// A variant is chosen by what the app says the window *is*, never by
// anything the skin could work out for itself, so a skin cannot dress a
// window the app did not name — and an app that names none gets exactly the
// frame every window got before variants existed.
func (sk *Skin) windowFor(role string) *SkinWindow {
	if sk == nil || sk.Window == nil {
		return nil
	}
	if role != "" {
		if v := sk.Window.Variants[role]; v != nil {
			return v
		}
	}
	return sk.Window
}

// framePart is one of the frame's parts for a window of role: the variant's
// rebinding where it has one, the skin's own part otherwise.
func (sk *Skin) framePart(name, role string) *SkinPart {
	if w := sk.windowFor(role); w != nil {
		if p := w.Parts[name]; p != nil {
			return p
		}
	}
	return sk.part(name)
}

// skinBaseDecoration is the base pack's frame, which a skin starts from.
// A base engine with no frame of its own gives the plain one, exactly as it
// would if the base pack were selected directly.
func skinBaseDecoration(l *Classic, st DecorationState) DecorationSpec {
	return skinUnderFrame(l).Decoration(l, st)
}

// skinUnderFrame is the base pack's frame engine, or the plain frame when
// the base look has none of its own — the same answer DecorationOf gives
// when that pack is selected directly.
func skinUnderFrame(l *Classic) DecorationEngine {
	if e, ok := under(l).(DecorationEngine); ok {
		return e
	}
	return plainDecoration{lk: l}
}

func (skinEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	sk := skinFor(l)
	band := sk.framePart("caption", st.Role)
	if sk == nil || band.art("normal") == nil {
		skinBaseDrawDecoration(l, ctx, f, st)
		return
	}
	cs := StateNone
	if !st.Active {
		cs = StateInactive
	}
	// The border is painted first, from the caption part's own art where the
	// skin drew a window frame, so the band and the sides meet.
	if !st.Maximized {
		spec := skinEngine{}.Decoration(l, st)
		if !spec.Border.Zero() || spec.Split || spec.CaptionGap > 0 {
			if !sk.drawPart(l, ctx, f.Window, sk.framePart("window", st.Role), cs) {
				col := l.palette.Border
				drawFrameBorder(ctx, f.Window, spec.Border, col)
			}
		}
	}
	if !sk.drawPart(l, ctx, f.Caption, band, cs) {
		skinBaseDrawDecoration(l, ctx, f, st)
	}
}

func skinBaseDrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	skinUnderFrame(l).DrawDecoration(l, ctx, f, st)
}

func (skinEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	sk := skinFor(l)
	band := sk.framePart("caption", st.Role)
	if sk == nil || band.art("normal") == nil {
		skinUnderFrame(l).DrawCaptionTitle(l, ctx, b, title, st)
		return
	}
	cs := StateNone
	if !st.Active {
		cs = StateInactive
	}
	col := sk.textColor(l, band, cs)
	if colorUnset(col) {
		col = l.palette.Text
		if !st.Active {
			col = l.palette.TextMuted
		}
	}
	// A skin's title is real text in a real font, not a bitmap alphabet: it
	// has to hold a window name in any language the user reads, and it has
	// to be legible to a magnifier. Its size is the caption text role's when
	// the role states one — a fourteen-pixel band cannot hold a title set at
	// the body size — and it is bold either way, as a caption always was.
	f := l.bold
	t := band.Text
	if t == nil {
		t = sk.Text["control"]
	}
	if t != nil && t.Size > 0 {
		f = BakeFamily(l.uiFamily, WeightBold, t.Size*l.Scale()/sk.Design.Scale, col)
	}
	if t != nil && t.Upper {
		// The skin's choice, and only its painting: the window's title is
		// still the app's, and so is what a screen reader reads.
		title = strings.ToUpper(title)
	}
	var place SkinTitle
	if w := sk.windowFor(st.Role); w != nil {
		place = w.Title
	}
	pad := l.S(8)
	plate := sk.framePart("caption.title", st.Role)
	if sp := plate.art(skinStateName(cs)); sp != nil && f != nil && title != "" {
		// The plate behind the title: the gap a ribbed or grooved band
		// leaves for the words, as wide as the words are. It is the one
		// piece of a caption that depends on the title, which is why it is
		// a part of its own rather than something a fixed slice of the
		// band could say — the band cannot know how long the title is.
		// Set from the start, it is a tab: the title on a tongue of the
		// window, as far in as the skin says.
		//
		// The plate's slices are its ends and the words go in its middle:
		// a plate that is a tab with a long curve at its right end says so
		// in its right slice, and the words stop short of the curve.
		k := l.Scale() / sk.Design.Scale
		room := skinWhole(l.S(captionTitleRoom))
		in := Insets{
			Top:    skinWhole(sp.Slice.Top * k),
			Right:  max(room, skinWhole(sp.Slice.Right*k)),
			Bottom: skinWhole(sp.Slice.Bottom * k),
			Left:   max(room, skinWhole(sp.Slice.Left*k)),
		}
		x0 := b.Min.X
		if place.Start {
			x0 += place.Inset * k
		}
		avail := b.Max.X - x0 - in.Left - in.Right
		if !place.Start {
			avail = b.Dx() - pad - in.Left - in.Right
		}
		show := title
		if f.Advance(show) > avail {
			show = f.Fit(show, max(avail, 4))
		}
		// Whole pixels, and never narrower than the words: the text is fitted
		// to the plate's middle again as it is set, and a middle rounded a
		// hair short of the words would lose their last letter to an ellipsis.
		w := float32(math.Ceil(float64(f.Advance(show)))) + in.Left + in.Right
		x := b.Min.X + (b.Dx()-w)*0.5
		if place.Start {
			x = x0
		}
		x = float32(math.Round(float64(x)))
		box := paintengine2d.XYWH(x, b.Min.Y, w, b.Dy())
		sk.drawPart(l, ctx, box, plate, cs)
		captionTitle(l, ctx, f, in.Apply(box), show, col, true, 0)
		return
	}
	if place.Start {
		inset := place.Inset * l.Scale() / sk.Design.Scale
		captionTitle(l, ctx, f, paintengine2d.XYWH(b.Min.X+inset, b.Min.Y, max(b.Dx()-inset, 0), b.Dy()), title, col, false, 0)
		return
	}
	captionTitle(l, ctx, f, b, title, col, true, pad)
}

// captionTitleRoom is the room the caption.title plate leaves either side
// of the words, in design pixels, so they never touch the band. A plate that
// wants more draws it into its own end slices.
const captionTitleRoom = 4

func (skinEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	sk := skinFor(l)
	if sk == nil {
		skinUnderFrame(l).DrawCaptionButton(l, ctx, b, k, cs, st)
		return
	}
	if !st.Active {
		cs |= StateInactive
	}
	// A caption button takes its own art where the skin gives it any, then
	// the tool button's — flat until the pointer is on it, which is what
	// every frame of the last twenty years does — then the push button's,
	// for a skin whose frame wants raised keys. The glyph over it is the
	// toolkit's own, so a caption button says what it does at every scale
	// and in every skin: nobody has to guess which unlabelled square closes
	// the window.
	p := sk.framePart("caption.button", st.Role)
	if p.art("normal") == nil {
		p = sk.part("button")
		if sk.has("tool") {
			p = sk.part("tool")
		}
	}
	painted := sk.drawPart(l, ctx, b, p, cs)
	if painted || sk.framePart("caption", st.Role).art("normal") != nil {
		col := sk.textColor(l, p, cs)
		if colorUnset(col) {
			col = l.palette.Text
			if !st.Active {
				col = l.palette.TextMuted
			}
		}
		side := min(b.Dx(), b.Dy())
		DrawCaptionGlyph(ctx, b, k, st.Maximized, col, side*0.42, max(1, l.S(1)))
		return
	}
	skinUnderFrame(l).DrawCaptionButton(l, ctx, b, k, cs, st)
}

// skinWhole rounds a frame measurement to a whole device pixel, never below
// one where the skin asked for any: the frame tests require whole borders at
// every scale, and half a pixel of border is a seam.
func skinWhole(v float32) float32 {
	if v <= 0 {
		return 0
	}
	r := float32(math.Round(float64(v)))
	if r < 1 {
		return 1
	}
	return r
}

// ---- the silhouette -------------------------------------------------------

// WindowShape is the outline a skin's window really has, so the desktop
// shows through everywhere it is not.
//
// A skin says it in one of two ways and this reads both (skin.go,
// loadWindowShape): a union of rounded rects in design pixels, resolved
// against the window at the look's scale and therefore exact at every one;
// or the alpha channel of a sprite, for a skin whose window is one picture.
// A skin that says neither frames a rectangle, which is what both demo
// skins but Deck do.
//
// The rect form is the one to reach for and it was chosen over an SVG path
// string deliberately: nothing in the toolkit parses path strings, and a
// rect union is already what both consumers want — a compositor's input
// region is a rect list, and hit-testing one is a loop rather than a
// rasterisation.
func (skinEngine) WindowShape(l *Classic, f DecorationFrame, st DecorationState) *Silhouette {
	sk := skinFor(l)
	w := sk.windowFor(st.Role)
	if w == nil {
		return nil
	}
	if sp := w.ShapeArt; sp != nil {
		// The art *is* the outline: cut at the sheet scale the window's own
		// size asks for, and resampled from there, exactly as the same
		// sprite is when it is painted.
		if v := sk.variant(sp, sk.assetTarget(sp, l.Scale())); v != nil {
			return &Silhouette{Mask: v.pieces[pieceWhole]}
		}
		return nil
	}
	rects := skinShapeRects(w.Shape, f.Window, l.Scale()/sk.Design.Scale)
	if skinShapeIsTheFrame(rects, f.Window, DecorationOf(l, st).Radius) {
		return nil
	}
	return SilhouetteOfRects(rects...)
}

// skinShapeIsTheFrame reports whether a resolved silhouette is the window's
// own box, rounded no more than the frame already rounds it.
//
// Such a silhouette describes nothing the frame does not do already, and it
// is not free: a shaped window states its input region as the shape and
// *replaces* the resize band its shadow margin held, so it is resized from
// inside its own edges instead of from the band outside them. A skin whose
// "shape" is the plain window — Nocturne's, written when nothing consumed
// the key — would trade that band for corners it already had.
//
// The radius is compared against the one in effect rather than the one the
// manifest states, which is what makes an uncomposited screen come out
// right: DecorationOf squares the corners there because alpha counts for
// nothing, so the silhouette stops being redundant and cuts them for real.
func skinShapeIsTheFrame(rects []SilhouetteRect, win paintengine2d.Rect, radius [4]float32) bool {
	if len(rects) != 1 {
		return false
	}
	r := rects[0]
	near := func(a, b float32) bool { return a-b < 0.5 && b-a < 0.5 }
	if !near(r.Rect.Min.X, win.Min.X) || !near(r.Rect.Min.Y, win.Min.Y) ||
		!near(r.Rect.Max.X, win.Max.X) || !near(r.Rect.Max.Y, win.Max.Y) {
		return false
	}
	for i, v := range r.Radius {
		if v > radius[i]+0.5 {
			return false
		}
	}
	return true
}

// skinWindowRects resolves a skin's silhouette against a window box, in
// device pixels. SkinWindowRects exports it so the lint command can show an
// author what their shape resolves to.
func (sk *Skin) skinWindowRects(win paintengine2d.Rect, scale float32) []SilhouetteRect {
	if sk == nil || sk.Window == nil || len(sk.Window.Shape) == 0 {
		return nil
	}
	return skinShapeRects(sk.Window.Shape, win, scale/sk.Design.Scale)
}

// skinShapeRects resolves a list of rects against a box at s device pixels
// per design pixel, dropping any a small box leaves no room for.
func skinShapeRects(shape []SkinShapeRect, box paintengine2d.Rect, s float32) []SilhouetteRect {
	out := make([]SilhouetteRect, 0, len(shape))
	for _, r := range shape {
		b := r.resolve(box, s)
		if b.Dx() <= 0 || b.Dy() <= 0 {
			continue
		}
		out = append(out, SilhouetteRect{
			Rect: b,
			// A corner radius is art, not geometry: it is scaled and then
			// clamped to the box it rounds, so a small window keeps a
			// sensible outline instead of an hourglass.
			Radius: skinCorners(r.Radius, s, b.Dx(), b.Dy()),
		})
	}
	return out
}

// skinCorners scales a rect's corner radii and keeps them inside the rect:
// two radii down one edge may never add up to more than that edge.
func skinCorners(r [4]float32, s, w, h float32) [4]float32 {
	out := [4]float32{r[0] * s, r[1] * s, r[2] * s, r[3] * s}
	k := float32(1)
	for _, p := range [4][3]float32{
		{out[0], out[1], w}, {out[3], out[2], w},
		{out[0], out[3], h}, {out[1], out[2], h},
	} {
		if sum := p[0] + p[1]; sum > p[2] && sum > 0 {
			k = min(k, p[2]/sum)
		}
	}
	if k < 1 {
		for i := range out {
			out[i] *= k
		}
	}
	return out
}

// SkinWindowRects is a skin's silhouette resolved against a window box at a
// display scale, as device-pixel rects. Empty for a skin with no shape, or
// for one whose silhouette is a sprite's alpha rather than rectangles.
func SkinWindowRects(sk *Skin, win paintengine2d.Rect, scale float32) []paintengine2d.Rect {
	rects := sk.skinWindowRects(win, scale)
	if len(rects) == 0 {
		return nil
	}
	out := make([]paintengine2d.Rect, len(rects))
	for i, r := range rects {
		out[i] = r.Rect
	}
	return out
}
