package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// DrawToolIcon paints a stock ToolIcon in the chosen glyph set.
// File sets (lucide / phosphor / tabler / heroicons / material-symbols /
// user dirs) load a tinted PNG from ~/.config/uitoolkit/icons/<set>/<action>.png
// (or name@2x.png). Missing stems use no-icon (pack or embedded), never a
// drawn classic scribble. Classic / Sharp stay in-process vectors when
// those sets are selected explicitly.
// DrawToolIconImage rasterizes icon onto a square pixmap (tray / SNI).
func DrawToolIconImage(icon ToolIcon, look LookAndFeel, size int) *paintengine2d.Image {
	if size < 16 {
		size = 22
	}
	img := paintengine2d.NewImage(size, size)
	ctx := paintengine2d.NewContext(img)
	plate := paintengine2d.RGB(0.18, 0.20, 0.24)
	ink := paintengine2d.RGB(0.92, 0.93, 0.95)
	set := IconSetClassic
	if look != nil {
		plate = look.Palette().Accent
		ink = look.Palette().TextOnAccent
		if ink == (paintengine2d.Color{}) {
			ink = paintengine2d.White
		}
		if c, ok := look.(*Classic); ok {
			set = c.Icons()
		}
	}
	b := paintengine2d.XYWH(0, 0, float32(size), float32(size))
	ctx.DrawRoundRect(b.Inset(1), 4, 4, paintengine2d.Fill(plate))
	if icon == IconNone {
		icon = IconInfo
	}
	DrawToolIcon(ctx, b.Inset(3), icon, ink, set)
	return img
}

func DrawToolIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color, set IconSetName) {
	if icon == IconNone || b.Empty() || ctx == nil {
		return
	}
	set = ParseIconSet(string(set))
	if IsSystemIconSet(set) {
		// An installed theme, drawn from the desktop's own files. It
		// falls through to the drawn set rather than to the no-icon
		// placeholder: a theme is somebody else's vocabulary, and an
		// action it has never heard of is a gap the toolkit can fill
		// itself, where a *toolkit* set missing one of its own stems is
		// a set that was installed wrong and should say so.
		if DrawSystemToolIcon(ctx, b, icon, col, set) {
			return
		}
		drawScaledIcon(ctx, b, func(ctx *paintengine2d.Context, db paintengine2d.Rect) { drawClassicIcon(ctx, db, icon, col) })
		return
	}
	if IsFileIconSet(set) {
		if DrawFileToolIcon(ctx, b, icon, col, set) {
			return
		}
		if drawEmbeddedNoIcon(ctx, b, col) {
			return
		}
		return
	}
	switch set {
	case IconSetSharp:
		drawScaledIcon(ctx, b, func(ctx *paintengine2d.Context, db paintengine2d.Rect) { drawSharpIcon(ctx, db, icon, col) })
	default:
		drawScaledIcon(ctx, b, func(ctx *paintengine2d.Context, db paintengine2d.Rect) { drawClassicIcon(ctx, db, icon, col) })
	}
}

// iconDesign is the box the vector icons are drawn for (a medium icon at
// 1x); other boxes scale the whole drawing, strokes included, so a 2x
// display gets 2x strokes, not hairlines.
const iconDesign = 24

func drawScaledIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, draw func(*paintengine2d.Context, paintengine2d.Rect)) {
	s := min(b.Dx(), b.Dy()) / iconDesign
	if s <= 0 {
		return
	}
	if s > 0.99 && s < 1.01 {
		draw(ctx, b)
		return
	}
	ctx.Save()
	ctx.Translate(b.Min.X, b.Min.Y)
	ctx.Scale(s, s)
	draw(ctx, paintengine2d.XYWH(0, 0, b.Dx()/s, b.Dy()/s))
	ctx.Restore()
}

func iconStroke(col paintengine2d.Color, width float32, cap paintengine2d.Cap, join paintengine2d.Join) paintengine2d.Paint {
	return paintengine2d.Paint{
		Color:  col,
		Style:  paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: width, Cap: cap, Join: join, MiterLimit: 4},
	}
}

func drawClassicIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color) {
	stroke := iconStroke(col, 1.6, paintengine2d.CapRound, paintengine2d.JoinRound)
	fill := paintengine2d.Fill(col)
	cx := (b.Min.X + b.Max.X) * 0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	w, h := b.Dx(), b.Dy()
	switch icon {
	case IconNew:
		page := paintengine2d.XYWH(b.Min.X+w*0.22, b.Min.Y+h*0.12, w*0.50, h*0.72)
		ctx.DrawRoundRect(page, 2, 2, paintengine2d.StrokePaint(col, 1.5))
		ctx.DrawRect(paintengine2d.XYWH(cx-2.4, cy-1, 4.8, 1.6), fill)
		ctx.DrawRect(paintengine2d.XYWH(cx-0.8, cy-3.2, 1.6, 4.8), fill)
	case IconOpen:
		p := paintengine2d.NewPath()
		p.MoveTo(b.Min.X+2, b.Max.Y-2.5)
		p.LineTo(b.Min.X+2, b.Min.Y+6)
		p.LineTo(b.Min.X+7, b.Min.Y+6)
		p.LineTo(b.Min.X+9.5, b.Min.Y+2.5)
		p.LineTo(b.Max.X-2, b.Min.Y+2.5)
		p.LineTo(b.Max.X-2, b.Max.Y-2.5)
		p.Close()
		ctx.DrawPath(p, stroke)
	case IconSave:
		box := b.Inset(2)
		ctx.DrawRoundRect(box, 2, 2, paintengine2d.StrokePaint(col, 1.5))
		ctx.DrawRect(paintengine2d.XYWH(box.Min.X+3, box.Min.Y, box.Dx()-6, 5), paintengine2d.StrokePaint(col, 1.2))
		ctx.DrawRect(paintengine2d.XYWH(box.Min.X+4, box.Max.Y-7, box.Dx()-8, 5), paintengine2d.StrokePaint(col, 1.2))
	case IconCut:
		ctx.DrawCircle(paintengine2d.Pt(b.Min.X+5, b.Max.Y-5), 2.4, stroke)
		ctx.DrawCircle(paintengine2d.Pt(b.Max.X-5, b.Max.Y-5), 2.4, stroke)
		p := paintengine2d.NewPath()
		p.MoveTo(b.Min.X+6, b.Max.Y-6)
		p.LineTo(b.Max.X-3, b.Min.Y+3)
		p.MoveTo(b.Max.X-6, b.Max.Y-6)
		p.LineTo(b.Min.X+3, b.Min.Y+3)
		ctx.DrawPath(p, stroke)
	case IconCopy:
		ctx.DrawRoundRect(paintengine2d.XYWH(b.Min.X+1.5, b.Min.Y+4, w*0.62, h*0.62), 2, 2, paintengine2d.StrokePaint(col, 1.4))
		ctx.DrawRoundRect(paintengine2d.XYWH(b.Min.X+6, b.Min.Y+1.5, w*0.62, h*0.62), 2, 2, paintengine2d.StrokePaint(col, 1.4))
	case IconPaste:
		ctx.DrawRoundRect(b.Inset(2.2), 2, 2, paintengine2d.StrokePaint(col, 1.5))
		ctx.DrawRoundRect(paintengine2d.XYWH(cx-4, b.Min.Y+1.2, 8, 4.2), 1.5, 1.5, fill)
	case IconUndo:
		p := paintengine2d.NewPath()
		p.MoveTo(b.Max.X-3, b.Min.Y+5)
		p.LineTo(b.Min.X+4, b.Min.Y+5)
		p.LineTo(b.Min.X+4, b.Max.Y-4)
		ctx.DrawPath(p, stroke)
		a := paintengine2d.NewPath()
		a.MoveTo(b.Min.X+1.5, b.Min.Y+5)
		a.LineTo(b.Min.X+5.5, b.Min.Y+2)
		a.LineTo(b.Min.X+5.5, b.Min.Y+8)
		a.Close()
		ctx.DrawPath(a, fill)
	case IconRedo:
		p := paintengine2d.NewPath()
		p.MoveTo(b.Min.X+3, b.Min.Y+5)
		p.LineTo(b.Max.X-4, b.Min.Y+5)
		p.LineTo(b.Max.X-4, b.Max.Y-4)
		ctx.DrawPath(p, stroke)
		a := paintengine2d.NewPath()
		a.MoveTo(b.Max.X-1.5, b.Min.Y+5)
		a.LineTo(b.Max.X-5.5, b.Min.Y+2)
		a.LineTo(b.Max.X-5.5, b.Min.Y+8)
		a.Close()
		ctx.DrawPath(a, fill)
	case IconSearch:
		ctx.DrawCircle(paintengine2d.Pt(cx-1.5, cy-1.5), w*0.28, stroke)
		p := paintengine2d.NewPath()
		p.MoveTo(cx+2.2, cy+2.2)
		p.LineTo(b.Max.X-1.5, b.Max.Y-1.5)
		ctx.DrawPath(p, stroke)
	case IconInfo:
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), w*0.42, stroke)
		ctx.DrawCircle(paintengine2d.Pt(cx, b.Min.Y+h*0.32), 1.15, fill)
		ctx.DrawRect(paintengine2d.XYWH(cx-0.85, b.Min.Y+h*0.44, 1.7, h*0.32), fill)
	case IconWarning:
		tri := paintengine2d.NewPath()
		tri.MoveTo(cx, b.Min.Y+1.5)
		tri.LineTo(b.Max.X-1.2, b.Max.Y-1.5)
		tri.LineTo(b.Min.X+1.2, b.Max.Y-1.5)
		tri.Close()
		ctx.DrawPath(tri, stroke)
		ctx.DrawRect(paintengine2d.XYWH(cx-0.85, b.Min.Y+h*0.38, 1.7, h*0.28), fill)
		ctx.DrawCircle(paintengine2d.Pt(cx, b.Max.Y-4.2), 1.1, fill)
	case IconError:
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), w*0.42, stroke)
		x := paintengine2d.NewPath()
		x.MoveTo(cx-3.2, cy-3.2)
		x.LineTo(cx+3.2, cy+3.2)
		x.MoveTo(cx+3.2, cy-3.2)
		x.LineTo(cx-3.2, cy+3.2)
		ctx.DrawPath(x, stroke)
	case IconQuestion:
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), w*0.42, stroke)
		q := paintengine2d.NewPath()
		q.MoveTo(cx-2.4, cy-2.6)
		q.LineTo(cx-0.4, cy-3.6)
		q.LineTo(cx+2.2, cy-2.2)
		q.LineTo(cx, cy+0.2)
		ctx.DrawPath(q, stroke)
		ctx.DrawCircle(paintengine2d.Pt(cx, cy+3.4), 1.05, fill)
	case IconMail:
		box := paintengine2d.XYWH(b.Min.X+1.6, b.Min.Y+h*0.28, w-3.2, h*0.50)
		ctx.DrawRoundRect(box, 1.5, 1.5, paintengine2d.StrokePaint(col, 1.5))
		flap := paintengine2d.NewPath()
		flap.MoveTo(box.Min.X+1.2, box.Min.Y+1.2)
		flap.LineTo(cx, box.Min.Y+h*0.22)
		flap.LineTo(box.Max.X-1.2, box.Min.Y+1.2)
		ctx.DrawPath(flap, stroke)
	case IconDownload:
		arrow := paintengine2d.NewPath()
		arrow.MoveTo(cx, b.Min.Y+1.8)
		arrow.LineTo(cx, b.Min.Y+h*0.58)
		arrow.MoveTo(cx-3.4, b.Min.Y+h*0.40)
		arrow.LineTo(cx, b.Min.Y+h*0.58)
		arrow.LineTo(cx+3.4, b.Min.Y+h*0.40)
		ctx.DrawPath(arrow, stroke)
		tray := paintengine2d.NewPath()
		tray.MoveTo(b.Min.X+2.2, b.Max.Y-6.2)
		tray.LineTo(b.Min.X+2.2, b.Max.Y-2.2)
		tray.LineTo(b.Max.X-2.2, b.Max.Y-2.2)
		tray.LineTo(b.Max.X-2.2, b.Max.Y-6.2)
		ctx.DrawPath(tray, stroke)
	case IconAttach:
		// A paperclip: a rounded hairpin, open at the bottom left, the
		// way every set has drawn one since Outlook.
		clip := paintengine2d.NewPath()
		clip.MoveTo(b.Min.X+5.4, b.Max.Y-6.2)
		clip.LineTo(b.Min.X+5.4, b.Min.Y+5.0)
		clip.QuadTo(cx, b.Min.Y+1.4, b.Max.X-5.4, b.Min.Y+5.0)
		clip.LineTo(b.Max.X-5.4, b.Max.Y-5.4)
		clip.QuadTo(cx, b.Max.Y-1.4, b.Min.X+7.6, b.Max.Y-5.4)
		clip.LineTo(b.Min.X+7.6, b.Min.Y+5.6)
		ctx.DrawPath(clip, stroke)
	case IconStar:
		ctx.DrawPath(starPath(cx, cy, w*0.44, w*0.18), stroke)
	case IconFlag:
		pole := paintengine2d.NewPath()
		pole.MoveTo(b.Min.X+4.2, b.Min.Y+2.2)
		pole.LineTo(b.Min.X+4.2, b.Max.Y-2.2)
		ctx.DrawPath(pole, stroke)
		cloth := paintengine2d.NewPath()
		cloth.MoveTo(b.Min.X+4.2, b.Min.Y+2.8)
		cloth.LineTo(b.Max.X-2.6, b.Min.Y+5.4)
		cloth.LineTo(b.Min.X+4.2, b.Min.Y+8.0)
		cloth.Close()
		ctx.DrawPath(cloth, stroke)
	case IconReply:
		ctx.DrawPath(replyPath(b, cx, cy, w, h, false), stroke)
	case IconForward:
		ctx.DrawPath(replyPath(b, cx, cy, w, h, true), stroke)
	case IconCheck:
		tick := paintengine2d.NewPath()
		tick.MoveTo(b.Min.X+2.8, cy+0.4)
		tick.LineTo(cx-1.4, b.Max.Y-3.4)
		tick.LineTo(b.Max.X-2.6, b.Min.Y+3.4)
		ctx.DrawPath(tick, stroke)
	case IconMute:
		bell := paintengine2d.NewPath()
		bell.MoveTo(b.Min.X+4.4, b.Max.Y-7.0)
		bell.LineTo(b.Min.X+4.4, cy-1.0)
		bell.QuadTo(cx, b.Min.Y+1.8, b.Max.X-4.4, cy-1.0)
		bell.LineTo(b.Max.X-4.4, b.Max.Y-7.0)
		bell.Close()
		ctx.DrawPath(bell, stroke)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+3.2, b.Max.Y-7.0, w-6.4, 1.4), fill)
		slash := paintengine2d.NewPath()
		slash.MoveTo(b.Min.X+2.4, b.Min.Y+2.4)
		slash.LineTo(b.Max.X-2.4, b.Max.Y-2.4)
		ctx.DrawPath(slash, stroke)
	case IconPen:
		nib := paintengine2d.NewPath()
		nib.MoveTo(b.Min.X+3.2, b.Max.Y-3.4)
		nib.LineTo(b.Max.X-5.4, b.Min.Y+4.2)
		nib.LineTo(b.Max.X-3.2, b.Min.Y+6.2)
		nib.LineTo(b.Min.X+5.4, b.Max.Y-1.6)
		nib.Close()
		ctx.DrawPath(nib, stroke)
		tip := paintengine2d.NewPath()
		tip.MoveTo(b.Min.X+3.2, b.Max.Y-3.4)
		tip.LineTo(b.Min.X+1.6, b.Max.Y-1.6)
		tip.LineTo(b.Min.X+5.4, b.Max.Y-1.6)
		tip.Close()
		ctx.DrawPath(tip, fill)
	}
}

func drawSharpIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color) {
	stroke := iconStroke(col, 1.7, paintengine2d.CapSquare, paintengine2d.JoinMiter)
	fill := paintengine2d.Fill(col)
	cx := (b.Min.X + b.Max.X) * 0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	w, h := b.Dx(), b.Dy()
	switch icon {
	case IconNew:
		page := paintengine2d.XYWH(b.Min.X+w*0.20, b.Min.Y+h*0.10, w*0.54, h*0.76)
		ctx.DrawRect(page, paintengine2d.StrokePaint(col, 1.6))
		ctx.DrawRect(paintengine2d.XYWH(cx-3.2, cy-0.7, 6.4, 1.4), fill)
		ctx.DrawRect(paintengine2d.XYWH(cx-0.7, cy-3.2, 1.4, 6.4), fill)
	case IconOpen:
		p := paintengine2d.NewPath()
		p.MoveTo(b.Min.X+1.5, b.Max.Y-1.5)
		p.LineTo(b.Min.X+1.5, b.Min.Y+5)
		p.LineTo(b.Min.X+7, b.Min.Y+5)
		p.LineTo(b.Min.X+7, b.Min.Y+1.5)
		p.LineTo(b.Max.X-1.5, b.Min.Y+1.5)
		p.LineTo(b.Max.X-1.5, b.Max.Y-1.5)
		p.Close()
		ctx.DrawPath(p, stroke)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+1.5, b.Min.Y+5, 5.5, 1.4), fill)
	case IconSave:
		box := b.Inset(1.8)
		ctx.DrawRect(box, paintengine2d.StrokePaint(col, 1.6))
		ctx.DrawRect(paintengine2d.XYWH(box.Min.X+3, box.Min.Y, box.Dx()-6, 4.5), fill)
		ctx.DrawRect(paintengine2d.XYWH(box.Min.X+4, box.Max.Y-6.5, box.Dx()-8, 4.5), paintengine2d.StrokePaint(col, 1.2))
	case IconCut:
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2.2, b.Max.Y-7.2, 4.4, 4.4), paintengine2d.StrokePaint(col, 1.4))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-6.6, b.Max.Y-7.2, 4.4, 4.4), paintengine2d.StrokePaint(col, 1.4))
		p := paintengine2d.NewPath()
		p.MoveTo(b.Min.X+4.4, b.Max.Y-7)
		p.LineTo(b.Max.X-2.5, b.Min.Y+2)
		p.MoveTo(b.Max.X-4.4, b.Max.Y-7)
		p.LineTo(b.Min.X+2.5, b.Min.Y+2)
		ctx.DrawPath(p, stroke)
	case IconCopy:
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+1.2, b.Min.Y+4.2, w*0.62, h*0.62), paintengine2d.StrokePaint(col, 1.5))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+6.2, b.Min.Y+1.2, w*0.62, h*0.62), paintengine2d.StrokePaint(col, 1.5))
	case IconPaste:
		ctx.DrawRect(b.Inset(2), paintengine2d.StrokePaint(col, 1.6))
		ctx.DrawRect(paintengine2d.XYWH(cx-4.2, b.Min.Y+1, 8.4, 4), fill)
	case IconUndo:
		p := paintengine2d.NewPath()
		p.MoveTo(b.Max.X-2.5, b.Min.Y+4.5)
		p.LineTo(b.Min.X+4, b.Min.Y+4.5)
		p.LineTo(b.Min.X+4, b.Max.Y-3)
		ctx.DrawPath(p, stroke)
		a := paintengine2d.NewPath()
		a.MoveTo(b.Min.X+1, b.Min.Y+4.5)
		a.LineTo(b.Min.X+5.8, b.Min.Y+1.6)
		a.LineTo(b.Min.X+5.8, b.Min.Y+7.4)
		a.Close()
		ctx.DrawPath(a, fill)
	case IconRedo:
		p := paintengine2d.NewPath()
		p.MoveTo(b.Min.X+2.5, b.Min.Y+4.5)
		p.LineTo(b.Max.X-4, b.Min.Y+4.5)
		p.LineTo(b.Max.X-4, b.Max.Y-3)
		ctx.DrawPath(p, stroke)
		a := paintengine2d.NewPath()
		a.MoveTo(b.Max.X-1, b.Min.Y+4.5)
		a.LineTo(b.Max.X-5.8, b.Min.Y+1.6)
		a.LineTo(b.Max.X-5.8, b.Min.Y+7.4)
		a.Close()
		ctx.DrawPath(a, fill)
	case IconSearch:
		side := w * 0.42
		ctx.DrawRect(paintengine2d.XYWH(cx-side*0.85, cy-side*0.85, side, side), paintengine2d.StrokePaint(col, 1.6))
		p := paintengine2d.NewPath()
		p.MoveTo(cx+side*0.25, cy+side*0.25)
		p.LineTo(b.Max.X-1.2, b.Max.Y-1.2)
		ctx.DrawPath(p, stroke)
	case IconInfo:
		ctx.DrawRect(b.Inset(2), paintengine2d.StrokePaint(col, 1.6))
		ctx.DrawRect(paintengine2d.XYWH(cx-1.1, b.Min.Y+h*0.28, 2.2, 2.2), fill)
		ctx.DrawRect(paintengine2d.XYWH(cx-1.0, b.Min.Y+h*0.46, 2.0, h*0.30), fill)
	case IconWarning:
		box := b.Inset(1.6)
		ctx.DrawRect(box, paintengine2d.StrokePaint(col, 1.6))
		ctx.DrawRect(paintengine2d.XYWH(cx-1.0, b.Min.Y+h*0.28, 2.0, h*0.32), fill)
		ctx.DrawRect(paintengine2d.XYWH(cx-1.1, b.Max.Y-h*0.26, 2.2, 2.2), fill)
	case IconError:
		ctx.DrawRect(b.Inset(2), paintengine2d.StrokePaint(col, 1.6))
		x := paintengine2d.NewPath()
		pad := w * 0.28
		x.MoveTo(b.Min.X+pad, b.Min.Y+pad)
		x.LineTo(b.Max.X-pad, b.Max.Y-pad)
		x.MoveTo(b.Max.X-pad, b.Min.Y+pad)
		x.LineTo(b.Min.X+pad, b.Max.Y-pad)
		ctx.DrawPath(x, stroke)
	case IconQuestion:
		ctx.DrawRect(b.Inset(2), paintengine2d.StrokePaint(col, 1.6))
		q := paintengine2d.NewPath()
		q.MoveTo(cx-2.6, cy-2.8)
		q.LineTo(cx-2.6, cy-3.8)
		q.LineTo(cx+2.6, cy-3.8)
		q.LineTo(cx+2.6, cy-1.2)
		q.LineTo(cx, cy-1.2)
		q.LineTo(cx, cy+1.2)
		ctx.DrawPath(q, stroke)
		ctx.DrawRect(paintengine2d.XYWH(cx-1.1, cy+2.6, 2.2, 2.2), fill)
	case IconMail:
		box := paintengine2d.XYWH(b.Min.X+1.4, b.Min.Y+h*0.26, w-2.8, h*0.52)
		ctx.DrawRect(box, paintengine2d.StrokePaint(col, 1.6))
		flap := paintengine2d.NewPath()
		flap.MoveTo(box.Min.X, box.Min.Y)
		flap.LineTo(cx, box.Min.Y+h*0.22)
		flap.LineTo(box.Max.X, box.Min.Y)
		ctx.DrawPath(flap, stroke)
	case IconDownload:
		ctx.DrawRect(paintengine2d.XYWH(cx-0.8, b.Min.Y+1.6, 1.6, h*0.46), fill)
		head := paintengine2d.NewPath()
		head.MoveTo(cx-3.6, b.Min.Y+h*0.38)
		head.LineTo(cx, b.Min.Y+h*0.58)
		head.LineTo(cx+3.6, b.Min.Y+h*0.38)
		head.Close()
		ctx.DrawPath(head, fill)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2, b.Max.Y-5.4, w-4, 1.5), fill)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2, b.Max.Y-5.4, 1.5, 3.8), fill)
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-3.5, b.Max.Y-5.4, 1.5, 3.8), fill)
	case IconAttach:
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+5.0, b.Min.Y+3.0, 1.6, h*0.56), fill)
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-6.6, b.Min.Y+3.0, 1.6, h*0.40), fill)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+5.0, b.Min.Y+3.0, w-11.6, 1.6), fill)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+5.0, b.Max.Y-5.4, w-11.6, 1.6), fill)
	case IconStar:
		ctx.DrawPath(starPath(cx, cy, w*0.44, w*0.18), fill)
	case IconFlag:
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+3.6, b.Min.Y+2.2, 1.6, h-4.4), fill)
		cloth := paintengine2d.NewPath()
		cloth.MoveTo(b.Min.X+5.2, b.Min.Y+2.6)
		cloth.LineTo(b.Max.X-2.4, b.Min.Y+2.6)
		cloth.LineTo(b.Max.X-2.4, b.Min.Y+8.2)
		cloth.LineTo(b.Min.X+5.2, b.Min.Y+8.2)
		cloth.Close()
		ctx.DrawPath(cloth, fill)
	case IconReply:
		ctx.DrawPath(replyPath(b, cx, cy, w, h, false), stroke)
	case IconForward:
		ctx.DrawPath(replyPath(b, cx, cy, w, h, true), stroke)
	case IconCheck:
		tick := paintengine2d.NewPath()
		tick.MoveTo(b.Min.X+2.6, cy)
		tick.LineTo(cx-1.6, b.Max.Y-3.6)
		tick.LineTo(b.Max.X-2.4, b.Min.Y+3.2)
		ctx.DrawPath(tick, iconStroke(col, 2.2, paintengine2d.CapButt, paintengine2d.JoinMiter))
	case IconMute:
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+4.4, cy-3.4, w-8.8, h*0.34), fill)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+3.2, b.Max.Y-7.2, w-6.4, 1.6), fill)
		slash := paintengine2d.NewPath()
		slash.MoveTo(b.Min.X+2.2, b.Min.Y+2.2)
		slash.LineTo(b.Max.X-2.2, b.Max.Y-2.2)
		ctx.DrawPath(slash, iconStroke(col, 2.0, paintengine2d.CapButt, paintengine2d.JoinMiter))
	case IconPen:
		body := paintengine2d.NewPath()
		body.MoveTo(b.Min.X+3.4, b.Max.Y-3.2)
		body.LineTo(b.Max.X-5.2, b.Min.Y+3.8)
		body.LineTo(b.Max.X-3.0, b.Min.Y+5.8)
		body.LineTo(b.Min.X+5.6, b.Max.Y-1.4)
		body.Close()
		ctx.DrawPath(body, stroke)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+1.4, b.Max.Y-3.2, 4.2, 1.6), fill)
	}
}

// starPath is a five-pointed star, outer radius r and inner radius ri,
// point up. Both stock sets draw the same shape and differ only in
// whether it is stroked or filled, because a star is a star: the sets
// disagree about bevels and corners, not about this.
func starPath(cx, cy, r, ri float32) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	// Ten vertices, alternating outer and inner, starting at the top.
	const turn = 3.14159265 / 5
	for i := 0; i < 10; i++ {
		rad := r
		if i%2 == 1 {
			rad = ri
		}
		a := float64(float32(i)*turn) - 3.14159265/2
		x := cx + rad*float32(math.Cos(a))
		y := cy + rad*float32(math.Sin(a))
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return p
}

// replyPath is the curved arrow both reply and forward are drawn from:
// an arrowhead at one end and a line that turns down into the message it
// points back at. forward is the same shape the other way round, which
// is what every icon set does and why they are one function.
func replyPath(b paintengine2d.Rect, cx, cy, w, h float32, forward bool) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	tipX, backX := b.Min.X+2.4, b.Max.X-3.2
	headX := b.Min.X + 7.0
	if forward {
		tipX, backX = b.Max.X-2.4, b.Min.X+3.2
		headX = b.Max.X - 7.0
	}
	y := b.Min.Y + h*0.34
	p.MoveTo(headX, y-3.6)
	p.LineTo(tipX, y)
	p.LineTo(headX, y+3.6)
	p.MoveTo(tipX, y)
	p.LineTo(backX-(backX-tipX)*0.25, y)
	p.LineTo(backX, y+h*0.22)
	p.LineTo(backX, b.Max.Y-2.6)
	return p
}

// IconSetOf is the icon set a look draws with, for a widget that paints
// an icon itself rather than through the look. A look that is not this
// package's own answers with the classic drawn set, which every build
// has.
func IconSetOf(look LookAndFeel) IconSetName {
	if c, ok := look.(*Classic); ok {
		return c.Icons()
	}
	return IconSetClassic
}
