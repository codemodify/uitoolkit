package style

import "github.com/codemodify/paintengine2d"

// DrawToolIcon paints a stock ToolIcon in the chosen glyph set.
// File sets (lucide / phosphor / tabler / heroicons / material-symbols /
// user dirs) load a tinted PNG from ~/.config/uitoolkit/icons/<set>/<action>.png
// (or name@2x.png). Missing files fall back to the drawn classic set.
// Classic / Sharp stay in-process vectors.
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
	if IsFileIconSet(set) {
		if DrawFileToolIcon(ctx, b, icon, col, set) {
			return
		}
		set = FallbackIcons(set)
	}
	switch set {
	case IconSetSharp:
		drawSharpIcon(ctx, b, icon, col)
	default:
		drawClassicIcon(ctx, b, icon, col)
	}
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
