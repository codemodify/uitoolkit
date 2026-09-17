package style

import (
	"math"

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
	if sk == nil || sk.Window == nil {
		return base
	}
	w := sk.Window
	s := l.Scale() / sk.Design.Scale
	spec := base
	if w.Layout != "" {
		spec.Layout = w.Layout
	}
	// The border and caption are the skin's own geometry, snapped to whole
	// device pixels: a frame measured in fractions leaks a seam between the
	// border and the client at 1.25 and 1.75.
	if !w.Border.Zero() {
		spec.Border = Insets{
			Top:    skinWhole(w.Border.Top * s),
			Right:  skinWhole(w.Border.Right * s),
			Bottom: skinWhole(w.Border.Bottom * s),
			Left:   skinWhole(w.Border.Left * s),
		}
	}
	if w.Caption > 0 {
		spec.Caption = max(skinWhole(w.Caption*s), l.S(skinCaptionMin))
		// A caption the skin resized has to resize its buttons with it, or
		// the art and the hit boxes disagree.
		side := skinWhole(spec.Caption * 0.62)
		spec.Button = paintengine2d.Pt(max(side, l.S(skinCaptionMin)), 0)
		spec.CloseButton = paintengine2d.Point{}
		spec.ButtonPad = Insets{Right: skinWhole(spec.Caption * 0.2), Left: skinWhole(spec.Caption * 0.2)}
		spec.ButtonGap = skinWhole(spec.Caption * 0.12)
		spec.CenterButtons = true
	}
	if w.Radius != [4]float32{} {
		spec.Radius = [4]float32{
			w.Radius[0] * s, w.Radius[1] * s,
			w.Radius[2] * s, w.Radius[3] * s,
		}
	}
	if sk.has("caption") {
		// A skin that paints its own caption band keeps the band stacked
		// above the app's title bar: the art is a picture of a title bar,
		// and merging it under an app's own header would draw two.
		spec.Stacked = true
	}
	return spec
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
	if sk == nil || !sk.has("caption") {
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
		if !spec.Border.Zero() {
			if !sk.draw(l, ctx, f.Window, "window", cs) {
				col := l.palette.Border
				drawFrameBorder(ctx, f.Window, spec.Border, col)
			}
		}
	}
	if !sk.draw(l, ctx, f.Caption, "caption", cs) {
		skinBaseDrawDecoration(l, ctx, f, st)
	}
}

func skinBaseDrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	skinUnderFrame(l).DrawDecoration(l, ctx, f, st)
}

func (skinEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	sk := skinFor(l)
	if sk == nil || !sk.has("caption") {
		skinUnderFrame(l).DrawCaptionTitle(l, ctx, b, title, st)
		return
	}
	cs := StateNone
	if !st.Active {
		cs = StateInactive
	}
	col := sk.textColor(l, sk.part("caption"), cs)
	if colorUnset(col) {
		col = l.palette.Text
		if !st.Active {
			col = l.palette.TextMuted
		}
	}
	// A skin's title is real text in a real font, not a bitmap alphabet: it
	// has to hold a window name in any language the user reads, and it has
	// to be legible to a magnifier.
	captionTitle(l, ctx, l.bold, b, title, col, true, l.S(8))
}

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
	part := "button"
	for _, candidate := range []string{"caption.button", "tool"} {
		if sk.has(candidate) {
			part = candidate
			break
		}
	}
	painted := sk.draw(l, ctx, b, part, cs)
	if painted || sk.has("caption") {
		col := sk.textColor(l, sk.part(part), cs)
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
	if sk == nil || sk.Window == nil {
		return nil
	}
	if sp := sk.Window.ShapeArt; sp != nil {
		// The art *is* the outline: cut at the sheet scale the window's own
		// size asks for, and resampled from there, exactly as the same
		// sprite is when it is painted.
		if v := sk.variant(sp, sk.assetTarget(sp, l.Scale())); v != nil {
			return &Silhouette{Mask: v.pieces[pieceWhole]}
		}
		return nil
	}
	return SilhouetteOfRects(sk.skinWindowRects(f.Window, l.Scale())...)
}

// skinWindowRects resolves a skin's silhouette against a window box, in
// device pixels. SkinWindowRects exports it so the lint command can show an
// author what their shape resolves to.
func (sk *Skin) skinWindowRects(win paintengine2d.Rect, scale float32) []SilhouetteRect {
	if sk == nil || sk.Window == nil || len(sk.Window.Shape) == 0 {
		return nil
	}
	s := scale / sk.Design.Scale
	out := make([]SilhouetteRect, 0, len(sk.Window.Shape))
	for _, r := range sk.Window.Shape {
		x := win.Min.X + r.X*s
		y := win.Min.Y + r.Y*s
		w, h := r.W*s, r.H*s
		if r.StretchX {
			// W is a margin from the right edge, not a width.
			w = win.Dx() - r.X*s - r.W*s
		}
		if r.StretchY {
			h = win.Dy() - r.Y*s - r.H*s
		}
		if w <= 0 || h <= 0 {
			continue
		}
		out = append(out, SilhouetteRect{
			Rect: paintengine2d.XYWH(x, y, w, h),
			// A corner radius is art, not geometry: it is scaled and then
			// clamped to the box it rounds, so a small window keeps a
			// sensible outline instead of an hourglass.
			Radius: skinCorners(r.Radius, s, w, h),
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
