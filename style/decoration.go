package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// Window frames. A window whose frame uitoolkit draws (a client-side frame,
// docs/decorations.md) asks its look for the frame's measurements
// ([DecorationOf]) and paints, in z-order, the border and the caption band
// ([DrawDecorationOf]), then its title bar with the window title
// ([DrawCaptionTitleOf]) and the caption buttons ([DrawCaptionButtonOf]).
//
// An engine paints its era's frame by implementing [DecorationEngine]. Every
// other engine gets its in-app window frame (DrawWindowFrame,
// WindowCloseRect, Face) adapted to the job, so every pack has a frame in its
// own look from the start; looks that are not engine looks get a plain one.

// CaptionButton is a window caption button. The values match
// platform.CaptionButton's.
type CaptionButton uint8

const (
	CaptionClose CaptionButton = iota + 1
	CaptionMinimize
	// CaptionMaximize shows "restore" while the window is maximized.
	CaptionMaximize
	// CaptionMenu opens the window menu (Windows' control-menu box, Motif's
	// window menu button, KDE's "M").
	CaptionMenu
)

// Edges is a set of window edges. The values match platform.Edges'.
type Edges uint8

const (
	EdgeTop Edges = 1 << iota
	EdgeBottom
	EdgeLeft
	EdgeRight
)

// DecorationState is the state of a window frame.
type DecorationState struct {
	// Active paints the active window's frame; otherwise the backdrop
	// (inactive) one. It follows the desktop's idea of the active window
	// (xdg_toplevel's activated, _NET_WM_STATE_FOCUSED), not keyboard focus.
	Active bool
	// Maximized: no border, and the maximize button restores.
	Maximized bool
	// Tiled are the edges against a screen edge or a tiled neighbour
	// (square corners and no shadow there once frames have them).
	Tiled Edges
	// Custom: the app's own title bar (tabs, a tool bar) fills the caption,
	// not just the window title.
	Custom bool
}

// DecorationSpec is a look's window frame in one state, in device pixels.
type DecorationSpec struct {
	// Border is the visible frame around the window (zero when maximized).
	Border Insets
	// Caption is the caption's height: a stacked frame's strip, the least
	// height of a merged frame's title bar.
	Caption float32
	// Stacked frames keep the look's own caption — the window title and
	// the caption buttons, as the classic desktops drew it (Windows 95 to
	// XP, the Mac, Motif) — in a strip above the app's title bar, which
	// becomes the row under it. Otherwise the frame is merged: the app's
	// title bar is the caption, with the buttons at its sides (GTK's header
	// bars, Windows 11, macOS, Chromium, SourceGit).
	Stacked bool
	// Button is one caption button's box (Y 0: the caption's full height,
	// less ButtonPad.Top, as Windows 10 and 11 and Chromium size theirs).
	// ButtonGap is the gap between two buttons, ButtonPad the room between
	// the buttons and the caption's edges: Top from its top, Left and Right
	// from its sides.
	Button    paintengine2d.Point
	ButtonGap float32
	ButtonPad Insets
	// CloseButton is the close button's box where it differs from Button
	// (Windows 7's wide red one; zero: Button), CloseGap extra room between
	// it and its neighbour (Windows 95 to 2000 keep 2px).
	CloseButton paintengine2d.Point
	CloseGap    float32
	// CenterButtons centres the buttons in a caption taller than Caption
	// (GNOME, KDE, macOS); otherwise they keep ButtonPad.Top from the top
	// (Windows).
	CenterButtons bool
	// CenterTitle: the look centres the window title on the window (the
	// Mac, GNOME, Motif), so the title's box is kept clear of both button
	// groups alike; otherwise it is the space between them.
	CenterTitle bool
	// Layout is the look's own caption-button layout in GNOME's syntax
	// ("close,minimize,maximize:" on the Mac), used when the user prefers
	// the theme's layout to the desktop's (look.json "captionButtons");
	// empty: the desktop's.
	Layout string
	// Radius (the outer corners, top-left clockwise) and Shadow (how far
	// the frame's drop shadow reaches) are what the look wants once frames
	// can be translucent (docs/decorations.md, Phase 3). Frames are square
	// and shadowless until then.
	Radius [4]float32
	Shadow Insets
}

// ButtonBox is the box of caption button k in s: the close button's own
// when it has one.
func (s DecorationSpec) ButtonBox(k CaptionButton) paintengine2d.Point {
	if k == CaptionClose && s.CloseButton.X > 0 {
		return s.CloseButton
	}
	return s.Button
}

// DecorationFrame is where a window frame's parts are, in device pixels.
type DecorationFrame struct {
	// Window is the visible window; the border runs just inside it.
	Window paintengine2d.Rect
	// Caption is the caption: a merged frame's whole title bar, a stacked
	// frame's strip.
	Caption paintengine2d.Rect
	// Bar is the app's title-bar row under a stacked frame's strip (empty
	// when there is none).
	Bar paintengine2d.Rect
}

// DecorationEngine is an optional engine hook: a top-level window's frame
// in the engine's own style (Win95's gradient caption, Luna's blue one,
// Aqua's traffic lights). Engines without it get their in-app window frame
// adapted (DrawWindowFrame, WindowCloseRect).
type DecorationEngine interface {
	// Decoration is the frame's measurements in state st.
	Decoration(l *Classic, st DecorationState) DecorationSpec
	// DrawDecoration paints the border inside f.Window and the caption
	// band f.Caption (a unified look may paint f.Bar too), under the title
	// bar; the window paints the rest.
	DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState)
	// DrawCaptionTitle paints the window title in b, the caption's free
	// space between the button groups (left, centred, …: the look's way).
	DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState)
	// DrawCaptionButton paints caption button k in its box b, in control
	// state cs (StateHovered, StatePressed).
	DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState)
}

// decorationFor is the frame painter of lk: its engine's own, the base
// look's plain frame, or the adapter over the engine's in-app window frame.
func decorationFor(lk LookAndFeel) (*Classic, DecorationEngine) {
	c, ok := lk.(*Classic)
	if !ok || c == nil {
		return nil, plainDecoration{lk: lk}
	}
	if e, ok := c.eng().(DecorationEngine); ok {
		return c, e
	}
	if c.eng().ID() == baseEngine.ID() {
		return c, plainDecoration{lk: c}
	}
	return c, frameAdapter{}
}

// DecorationOf is lk's window frame in state st, in device pixels.
func DecorationOf(lk LookAndFeel, st DecorationState) DecorationSpec {
	if lk == nil {
		return DecorationSpec{}
	}
	c, e := decorationFor(lk)
	s := e.Decoration(c, st)
	if st.Maximized {
		s.Border = Insets{}
	}
	return s
}

// DrawDecorationOf paints lk's window frame: the border and the caption band.
func DrawDecorationOf(lk LookAndFeel, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	if lk == nil || ctx == nil || f.Window.Empty() {
		return
	}
	c, e := decorationFor(lk)
	e.DrawDecoration(c, ctx, f, st)
}

// DrawCaptionTitleOf paints the window title in the caption's free space b.
func DrawCaptionTitleOf(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	if lk == nil || ctx == nil || title == "" || b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	c, e := decorationFor(lk)
	e.DrawCaptionTitle(c, ctx, b, title, st)
}

// DrawCaptionButtonOf paints caption button k in its box b.
func DrawCaptionButtonOf(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	if lk == nil || ctx == nil || b.Empty() {
		return
	}
	c, e := decorationFor(lk)
	e.DrawCaptionButton(c, ctx, b, k, cs, st)
}

// NativeDecoration reports whether lk's engine paints window frames itself
// (implements DecorationEngine); the others are adapted from their in-app
// window frames, or plain.
func NativeDecoration(lk LookAndFeel) bool {
	c, ok := lk.(*Classic)
	if !ok || c == nil {
		return false
	}
	_, native := c.eng().(DecorationEngine)
	return native
}

// ---- glyphs ---------------------------------------------------------------

// DrawCaptionGlyph draws a caption-button glyph of side s centred in b: an ✕
// (close), a bar (minimize), a square (maximize), two overlapping squares
// (restore, while maximized) or a small window (the window menu), in lines
// lw wide on whole device pixels, so it stays crisp at any scale.
func DrawCaptionGlyph(ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, maximized bool, col paintengine2d.Color, s, lw float32) {
	if ctx == nil || b.Empty() || col.A <= 0 {
		return
	}
	round := func(v float32) float32 { return float32(math.Round(float64(v))) }
	s = max(round(s), 3)
	lw = max(round(lw), 1)
	x0 := round(b.Min.X + (b.Dx()-s)*0.5)
	y0 := round(b.Min.Y + (b.Dy()-s)*0.5)
	g := paintengine2d.XYWH(x0, y0, s, s)
	fill := paintengine2d.Fill(col)
	frame := func(q paintengine2d.Rect) {
		p := paintengine2d.NewPath()
		p.AddRect(paintengine2d.XYWH(q.Min.X, q.Min.Y, q.Dx(), lw))
		p.AddRect(paintengine2d.XYWH(q.Min.X, q.Max.Y-lw, q.Dx(), lw))
		p.AddRect(paintengine2d.XYWH(q.Min.X, q.Min.Y+lw, lw, q.Dy()-2*lw))
		p.AddRect(paintengine2d.XYWH(q.Max.X-lw, q.Min.Y+lw, lw, q.Dy()-2*lw))
		ctx.DrawPath(p, fill)
	}
	switch k {
	case CaptionClose:
		p := paintengine2d.NewPath()
		p.MoveTo(g.Min.X, g.Min.Y)
		p.LineTo(g.Max.X, g.Max.Y)
		p.MoveTo(g.Max.X, g.Min.Y)
		p.LineTo(g.Min.X, g.Max.Y)
		ctx.DrawPath(p, paintengine2d.StrokePaint(col, lw*1.15))
	case CaptionMinimize:
		ctx.DrawRect(paintengine2d.XYWH(g.Min.X, round(g.Min.Y+s*0.5), s, lw), fill)
	case CaptionMaximize:
		if !maximized {
			frame(g)
			return
		}
		// Restore: a square in front of another, the back one showing its
		// top and right edges.
		d := max(lw*2, round(s*0.2))
		back := paintengine2d.XYWH(g.Min.X+d, g.Min.Y, s-d, s-d)
		front := paintengine2d.XYWH(g.Min.X, g.Min.Y+d, s-d, s-d)
		p := paintengine2d.NewPath()
		p.AddRect(paintengine2d.XYWH(back.Min.X, back.Min.Y, back.Dx(), lw))
		p.AddRect(paintengine2d.XYWH(back.Max.X-lw, back.Min.Y+lw, lw, back.Dy()-lw))
		p.AddRect(paintengine2d.XYWH(back.Min.X, back.Min.Y+lw, lw, front.Min.Y-back.Min.Y-lw))
		p.AddRect(paintengine2d.XYWH(front.Max.X, back.Max.Y-lw, back.Max.X-front.Max.X-lw, lw))
		ctx.DrawPath(p, fill)
		frame(front)
	case CaptionMenu:
		w := paintengine2d.XYWH(g.Min.X, round(g.Min.Y+s*0.1), s, round(s*0.8))
		frame(w)
		ctx.DrawRect(paintengine2d.XYWH(w.Min.X, w.Min.Y, w.Dx(), max(lw*2, round(s*0.25))), fill)
	}
}

// ---- the plain frame --------------------------------------------------------

// plainDecoration is the frame of the base look and of looks that are not
// engine looks: a hairline border, the caption on the window background
// (a rule under a caption that holds only the title), the title centred,
// flat 40×30 buttons over the look's tool face with generic glyphs, close
// turning red under the pointer (Windows, Breeze, Chromium).
type plainDecoration struct{ lk LookAndFeel }

func (d plainDecoration) Decoration(_ *Classic, st DecorationState) DecorationSpec {
	lk := d.lk
	r := func(v float32) float32 { return float32(math.Round(float64(Dip(lk, v)))) }
	return DecorationSpec{
		Border:  Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
		Caption: r(32),
		Button:  paintengine2d.Pt(r(40), 0),
		Layout:  "",
		Shadow:  PopupShadowOf(lk, PopupDialog),
	}
}

func (d plainDecoration) DrawDecoration(_ *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	pal := d.lk.Palette()
	if !st.Custom && !f.Caption.Empty() {
		ctx.DrawRect(paintengine2d.XYWH(f.Caption.Min.X, f.Caption.Max.Y-1, f.Caption.Dx(), 1), paintengine2d.Fill(pal.Divider))
	}
	if !st.Maximized {
		drawFrameBorder(ctx, f.Window, Insets{Top: 1, Right: 1, Bottom: 1, Left: 1}, pal.Border)
	}
}

func (d plainDecoration) DrawCaptionTitle(_ *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	col := d.lk.Palette().Text
	if !st.Active {
		col = d.lk.Palette().TextMuted
	}
	d.lk.DrawLabel(ctx, b, title, col, AlignCenter)
}

func (d plainDecoration) DrawCaptionButton(_ *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	drawPlainCaptionButton(d.lk, ctx, b, k, cs, st)
}

// drawPlainCaptionButton is a flat caption button over lk's tool face with
// a generic glyph; close turns red under the pointer, and a backdrop
// window's glyphs are dimmed.
func drawPlainCaptionButton(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	hot := cs.Hovered() || cs.Pressed()
	var fg paintengine2d.Color
	if k == CaptionClose && hot {
		bg := lk.Palette().Danger
		if cs.Pressed() {
			bg = bg.Lerp(paintengine2d.RGB(0, 0, 0), 0.2)
		}
		ctx.DrawRect(b, paintengine2d.Fill(bg))
		fg = paintengine2d.RGB(1, 1, 1)
	} else {
		fs := (cs & (StateHovered | StatePressed)) | StateAutoRaise
		if !st.Active {
			fs |= StateBackdrop
		}
		fg = DrawFaceOf(lk, ctx, b, RoleTool, fs)
	}
	if !st.Active && !hot {
		fg = fg.WithAlpha(fg.A * 0.55)
	}
	DrawCaptionGlyph(ctx, b, k, st.Maximized, fg, Dip(lk, 10), Dip(lk, 1))
}

// drawFrameBorder fills the band of widths in just inside w.
func drawFrameBorder(ctx *paintengine2d.Context, w paintengine2d.Rect, in Insets, col paintengine2d.Color) {
	if w.Empty() || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	if in.Top > 0 {
		p.AddRect(paintengine2d.XYWH(w.Min.X, w.Min.Y, w.Dx(), in.Top))
	}
	if in.Bottom > 0 {
		p.AddRect(paintengine2d.XYWH(w.Min.X, w.Max.Y-in.Bottom, w.Dx(), in.Bottom))
	}
	if in.Left > 0 {
		p.AddRect(paintengine2d.XYWH(w.Min.X, w.Min.Y+in.Top, in.Left, w.Dy()-in.Top-in.Bottom))
	}
	if in.Right > 0 {
		p.AddRect(paintengine2d.XYWH(w.Max.X-in.Right, w.Min.Y+in.Top, in.Right, w.Dy()-in.Top-in.Bottom))
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// FrameParts are the rects a frame's painting stays inside: the caption band
// (with the top border above it) and the side and bottom borders. A frame
// that paints a whole in-app window clips to them, leaving the content and
// a stacked frame's title-bar row to the window.
func FrameParts(f DecorationFrame, border Insets) []paintengine2d.Rect {
	w := f.Window
	top := max(f.Caption.Max.Y, w.Min.Y+border.Top)
	out := []paintengine2d.Rect{paintengine2d.XYWH(w.Min.X, w.Min.Y, w.Dx(), top-w.Min.Y)}
	if border.Left > 0 {
		out = append(out, paintengine2d.XYWH(w.Min.X, top, border.Left, w.Max.Y-top))
	}
	if border.Right > 0 {
		out = append(out, paintengine2d.XYWH(w.Max.X-border.Right, top, border.Right, w.Max.Y-top))
	}
	if border.Bottom > 0 {
		out = append(out, paintengine2d.XYWH(w.Min.X+border.Left, w.Max.Y-border.Bottom, w.Dx()-border.Left-border.Right, border.Bottom))
	}
	return out
}

// ---- the adapter ------------------------------------------------------------

// frameAdapter paints a top-level frame with an engine's in-app window frame
// (DrawWindowFrame, WindowCloseRect), for the engines that have no frame of
// their own: a stacked frame whose strip is the in-app caption, the frame's
// borders around the window, the engine's own close button wherever the
// button layout puts it, and minimize, maximize and the window menu as push
// buttons with generic glyphs at the close button's size. Only the frame's
// parts are painted: the in-app window's body would cover the content.
type frameAdapter struct{}

// adapterProbe is the sample in-app window the adapter measures the engine's
// frame on.
func adapterProbe(l *Classic) paintengine2d.Rect {
	return paintengine2d.XYWH(0, 0, snap(l.S(640)), snap(l.S(480)))
}

// adapterGeom is the engine's in-app frame measured: its insets, the close
// button on the probe (empty when the engine has none) and whether it sits in
// the left half.
func adapterGeom(l *Classic) (in Insets, closeR paintengine2d.Rect, left bool) {
	e := l.eng()
	in = e.WindowFrameInsets(l)
	in = Insets{Top: snap(in.Top), Right: snap(in.Right), Bottom: snap(in.Bottom), Left: snap(in.Left)}
	probe := adapterProbe(l)
	closeR = e.WindowCloseRect(l, probe)
	if !closeR.Empty() {
		closeR = winSnap(closeR)
	}
	left = !closeR.Empty() && (closeR.Min.X+closeR.Max.X)*0.5 < (probe.Min.X+probe.Max.X)*0.5
	return in, closeR, left
}

func (frameAdapter) Decoration(l *Classic, st DecorationState) DecorationSpec {
	in, cr, left := adapterGeom(l)
	s := DecorationSpec{
		Stacked:   true,
		Border:    Insets{Right: in.Right, Bottom: in.Bottom, Left: in.Left},
		Caption:   in.Top,
		ButtonGap: max(snap(l.S(2)), 1),
		Layout:    adapterLayouts[l.eng().ID()],
		Radius:    [4]float32{l.metrics.RadiusSmall, l.metrics.RadiusSmall, 0, 0},
		Shadow:    l.eng().PopupShadow(l, PopupDialog),
	}
	if cr.Empty() {
		side := snap(max(min(s.Caption-l.S(6), l.S(22)), l.S(12)))
		s.Button = paintengine2d.Pt(side, side)
		pad := snap((s.Caption - side) * 0.5)
		s.ButtonPad = Insets{Top: pad, Left: pad, Right: pad}
		return s
	}
	probe := adapterProbe(l)
	s.Button = cr.Size()
	pt := max(cr.Min.Y-probe.Min.Y, 0)
	var ps float32
	if left {
		ps = max(cr.Min.X-(probe.Min.X+in.Left), 0)
	} else {
		ps = max(probe.Max.X-in.Right-cr.Max.X, 0)
	}
	if pt+s.Button.Y > s.Caption {
		pt = max(s.Caption-s.Button.Y, 0)
	}
	s.ButtonPad = Insets{Top: pt, Left: ps, Right: ps}
	return s
}

// adapterFrame is where the in-app frame goes to frame window w: w itself,
// or, while maximized, grown past the screen by its side and bottom borders,
// which are off the screen then, as on Windows. (The in-app frame's top edge,
// if it has one, stays: its thickness is not known.)
func adapterFrame(l *Classic, w paintengine2d.Rect, st DecorationState) paintengine2d.Rect {
	if !st.Maximized {
		return w
	}
	in, _, _ := adapterGeom(l)
	return paintengine2d.Rect{
		Min: paintengine2d.Pt(w.Min.X-in.Left, w.Min.Y),
		Max: paintengine2d.Pt(w.Max.X+in.Right, w.Max.Y+in.Bottom),
	}
}

func (a frameAdapter) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	e := l.eng()
	fr := adapterFrame(l, f.Window, st)
	ws := WindowState{Active: st.Active}
	for _, part := range FrameParts(f, a.Decoration(l, st).Border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		e.DrawWindowFrame(l, ctx, fr, "", ws)
		ctx.Restore()
	}
}

// DrawCaptionTitle lays the engine's own caption over the free space b, as if
// its in-app window were exactly that wide (its borders just outside b), so
// the engine puts the title where its caption would (at the left, centred)
// without running under the buttons at either end.
func (frameAdapter) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	in, _, _ := adapterGeom(l)
	fr := paintengine2d.XYWH(b.Min.X-in.Left, b.Min.Y, b.Dx()+in.Left+in.Right, max(in.Top*4, snap(l.S(240))))
	ctx.Save()
	ctx.ClipRect(b)
	l.eng().DrawWindowFrame(l, ctx, fr, title, WindowState{Active: st.Active})
	ctx.Restore()
}

// DrawCaptionButton paints close as the engine's own close button (its in-app
// window moved so the button lands on b) and the others as push buttons at
// its size with generic glyphs.
func (a frameAdapter) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	e := l.eng()
	if k == CaptionClose {
		_, cr, _ := adapterGeom(l)
		if !cr.Empty() {
			fr := adapterProbe(l).Translate(paintengine2d.Pt(b.Min.X-cr.Min.X, b.Min.Y-cr.Min.Y))
			ctx.Save()
			ctx.ClipRect(b)
			e.DrawWindowFrame(l, ctx, fr, "", WindowState{Active: st.Active, CanClose: true, CloseHot: cs.Hovered(), ClosePress: cs.Pressed()})
			ctx.Restore()
			return
		}
	}
	// A push button's face: every look paints one, with a label colour that
	// reads on it whatever the caption behind is.
	fs := cs & (StateHovered | StatePressed)
	if !st.Active {
		fs |= StateBackdrop
	}
	fg := e.Face(l, ctx, b, RoleButton, fs)
	side := min(b.Dx(), b.Dy())
	DrawCaptionGlyph(ctx, b, k, st.Maximized, fg, side*0.45, max(l.S(1), side/16))
}

// adapterLayouts are the caption buttons of the desktops the adapted engines
// imitate, for the "theme" button layout.
var adapterLayouts = map[string]string{
	"win31":      "icon:minimize,maximize",
	"motif":      "icon:minimize,maximize",
	"os2":        "icon:minimize,maximize",
	"next":       "minimize:close",
	"openlook":   "icon:",
	"amiga":      "close:maximize",
	"beos":       "close:maximize",
	"platinum":   "close:maximize",
	"system7":    "close:maximize",
	"keramik":    "icon:minimize,maximize,close",
	"plastik":    "icon:minimize,maximize,close",
	"oxygen":     "icon:minimize,maximize,close",
	"clearlooks": "icon:minimize,maximize,close",
	"bluecurve":  "icon:minimize,maximize,close",
	"flatlaf":    ":minimize,maximize,close",
	"metal":      ":minimize,maximize,close",
	"nimbus":     ":minimize,maximize,close",
	"material":   ":minimize,maximize,close",
	"fusion":     ":minimize,maximize,close",
}

// ---- shared painters --------------------------------------------------------

// flatCaption is a set of flat caption buttons (Windows 10 and 11,
// SourceGit, FlatLaf): nothing at rest but the glyph, a wash under the
// pointer and a deeper one pressed, a red close button with a white glyph.
type flatCaption struct {
	fg, fgOff                  paintengine2d.Color // the glyph; in a backdrop window
	hover, press               paintengine2d.Color
	close, closePress, onClose paintengine2d.Color
	// glyph sides for minimize, maximize and close, and the line width.
	min, max, cls, lw float32
}

func (f flatCaption) draw(ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	hot := cs.Hovered() || cs.Pressed()
	fg := f.fg
	if !st.Active && !hot {
		fg = f.fgOff
	}
	switch {
	case k == CaptionClose && cs.Pressed():
		ctx.DrawRect(b, paintengine2d.Fill(f.closePress))
		fg = f.onClose.WithAlpha(f.onClose.A * 0.8)
	case k == CaptionClose && hot:
		ctx.DrawRect(b, paintengine2d.Fill(f.close))
		fg = f.onClose
	case cs.Pressed():
		ctx.DrawRect(b, paintengine2d.Fill(f.press))
	case hot:
		ctx.DrawRect(b, paintengine2d.Fill(f.hover))
	}
	s := f.max
	switch k {
	case CaptionMinimize:
		s = f.min
	case CaptionClose:
		s = f.cls
	}
	DrawCaptionGlyph(ctx, b, k, st.Maximized, fg, s, f.lw)
}

// captionTitle draws a window title in b: f in col, aligned (centred on b,
// or at its left after pad), elided to fit.
func captionTitle(l *Classic, ctx *paintengine2d.Context, f *Font, b paintengine2d.Rect, title string, col paintengine2d.Color, center bool, pad float32) {
	if f == nil {
		f = l.body
	}
	if center {
		l.drawFittedText(ctx, f, title, b, col, AlignCenter, pad)
		return
	}
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad, b.Dy()), col, AlignStart, 0)
}
