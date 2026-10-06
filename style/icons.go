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
	if icon == IconStarFilled || icon == IconDot {
		// Pure geometry with no house style to match, and no set ships
		// either of them: a filled star is a filled star. Drawing them
		// here rather than per set is what makes them work everywhere
		// instead of falling back to the missing-icon mark in the four
		// sets that have no such file.
		drawScaledIcon(ctx, b, func(ctx *paintengine2d.Context, db paintengine2d.Rect) { drawClassicIcon(ctx, db, icon, col) })
		return
	}
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
		if drawAppIcon(ctx, b, icon, col) {
			return
		}
		drawScaledIcon(ctx, b, func(ctx *paintengine2d.Context, db paintengine2d.Rect) { drawClassicIcon(ctx, db, icon, col) })
		return
	}
	if IsFileIconSet(set) {
		if DrawFileToolIcon(ctx, b, icon, col, set) {
			return
		}
		// The set has no file for this one. Where the toolkit can draw
		// the mark itself it does — a set that predates an id is behind,
		// not broken, and a person who copied their icons last month
		// should not get a box because the toolkit gained an id since.
		// A stem-only icon, which nothing anywhere can draw, and a set
		// that was never installed, which the person needs to be told
		// about, both fall to the missing-icon mark (see loadFileIcon).
		// An application's own icon is drawn by the application, which is
		// the point of registering one: the set has no file for a stem it
		// has never heard of, and the mark is still better than a box.
		if drawAppIcon(ctx, b, icon, col) {
			return
		}
		if Drawable(icon) && fileIconSetInstalled(set) {
			drawScaledIcon(ctx, b, func(ctx *paintengine2d.Context, db paintengine2d.Rect) { drawClassicIcon(ctx, db, icon, col) })
			return
		}
		drawEmbeddedNoIcon(ctx, b, col)
		return
	}
	if drawAppIcon(ctx, b, icon, col) {
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
		// A paperclip is one stroke: down the outer right, a wide U at
		// the bottom, up the outer left, a tighter hook over the top,
		// and back down the inside, stopping short.
		//
		// Drawn wide with tight bends it read as a rounded box at the
		// 20px a message list actually uses, so it is narrow, the bottom
		// U is open, and the inner stroke ends well clear of it.
		clip := paintengine2d.NewPath()
		clip.MoveTo(cx+3.4, b.Min.Y+7.5)
		clip.LineTo(cx+3.4, b.Min.Y+16.0)
		clip.QuadTo(cx, b.Min.Y+21.2, cx-3.4, b.Min.Y+16.0)
		clip.LineTo(cx-3.4, b.Min.Y+6.4)
		clip.QuadTo(cx-1.0, b.Min.Y+2.0, cx+1.1, b.Min.Y+6.4)
		clip.LineTo(cx+1.1, b.Min.Y+15.2)
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
	case IconTrash:
		// A bin: lid, handle, body, two ribs.
		ctx.DrawPath(lineP(b.Min.X+2.6, b.Min.Y+5.6, b.Max.X-2.6, b.Min.Y+5.6), stroke)
		ctx.DrawPath(lineP(cx-2.4, b.Min.Y+3.4, cx+2.4, b.Min.Y+3.4), stroke)
		body := paintengine2d.NewPath()
		body.MoveTo(b.Min.X+4.4, b.Min.Y+5.6)
		body.LineTo(b.Min.X+5.4, b.Max.Y-2.4)
		body.LineTo(b.Max.X-5.4, b.Max.Y-2.4)
		body.LineTo(b.Max.X-4.4, b.Min.Y+5.6)
		ctx.DrawPath(body, stroke)
		ctx.DrawPath(lineP(cx-1.8, b.Min.Y+8.2, cx-1.5, b.Max.Y-4.6), stroke)
		ctx.DrawPath(lineP(cx+1.8, b.Min.Y+8.2, cx+1.5, b.Max.Y-4.6), stroke)
	case IconArchive:
		// A box with a lid and a slot: the archive of every mail client.
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2.4, b.Min.Y+3.2, w-4.8, 3.4),
			paintengine2d.StrokePaint(col, 1.5))
		box := paintengine2d.NewPath()
		box.MoveTo(b.Min.X+3.8, b.Min.Y+6.6)
		box.LineTo(b.Min.X+3.8, b.Max.Y-2.6)
		box.LineTo(b.Max.X-3.8, b.Max.Y-2.6)
		box.LineTo(b.Max.X-3.8, b.Min.Y+6.6)
		ctx.DrawPath(box, stroke)
		ctx.DrawPath(lineP(cx-2.2, b.Min.Y+9.4, cx+2.2, b.Min.Y+9.4), stroke)
	case IconJunk:
		// A circle with a bar through it: "no", which is what junk means
		// as an action.
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), w*0.36, paintengine2d.StrokePaint(col, 1.6))
		d := w * 0.36 * 0.72
		ctx.DrawPath(lineP(cx-d, cy-d, cx+d, cy+d), stroke)
	case IconTag:
		// A label with its eyelet.
		tag := paintengine2d.NewPath()
		tag.MoveTo(cx+w*0.30, b.Min.Y+2.6)
		tag.LineTo(b.Min.X+2.6, b.Min.Y+2.6)
		tag.LineTo(b.Min.X+2.6, cy+h*0.06)
		tag.LineTo(cx+w*0.10, b.Max.Y-2.6)
		tag.LineTo(b.Max.X-2.6, cy-h*0.06)
		tag.Close()
		ctx.DrawPath(tag, stroke)
		ctx.DrawCircle(paintengine2d.Pt(b.Min.X+5.8, b.Min.Y+5.8), 1.5, paintengine2d.StrokePaint(col, 1.4))
	case IconFolder:
		fol := paintengine2d.NewPath()
		fol.MoveTo(b.Min.X+2.4, b.Max.Y-3.0)
		fol.LineTo(b.Min.X+2.4, b.Min.Y+4.6)
		fol.LineTo(b.Min.X+2.4+w*0.30, b.Min.Y+4.6)
		fol.LineTo(b.Min.X+2.4+w*0.30+2.0, b.Min.Y+6.8)
		fol.LineTo(b.Max.X-2.4, b.Min.Y+6.8)
		fol.LineTo(b.Max.X-2.4, b.Max.Y-3.0)
		fol.Close()
		ctx.DrawPath(fol, stroke)
	case IconReplyAll:
		// Two arrows into one line: reply, with a second head behind it.
		for _, dx := range []float32{0, 3.4} {
			head := paintengine2d.NewPath()
			head.MoveTo(b.Min.X+5.0+dx, cy-3.2)
			head.LineTo(b.Min.X+1.8+dx, cy)
			head.LineTo(b.Min.X+5.0+dx, cy+3.2)
			ctx.DrawPath(head, stroke)
		}
		tail := paintengine2d.NewPath()
		tail.MoveTo(b.Min.X+5.2, cy)
		tail.LineTo(b.Max.X-6.0, cy)
		tail.QuadTo(b.Max.X-2.4, cy, b.Max.X-2.4, cy+4.4)
		ctx.DrawPath(tail, stroke)
	case IconSettings:
		// A cog: a toothed outline around a hub. Thin rays out of a ring
		// read as a sun, so the teeth have width and are part of one
		// closed path — which is what makes the silhouette a gear.
		inner, outer := w*0.30, w*0.44
		const teeth = 8
		gear := paintengine2d.NewPath()
		for i := 0; i < teeth*2; i++ {
			a0 := float64(i)*math.Pi/teeth - math.Pi/(teeth*2)
			a1 := float64(i+1)*math.Pi/teeth - math.Pi/(teeth*2)
			r := inner
			if i%2 == 0 {
				r = outer
			}
			x0, y0 := cx+float32(math.Cos(a0))*r, cy+float32(math.Sin(a0))*r
			x1, y1 := cx+float32(math.Cos(a1))*r, cy+float32(math.Sin(a1))*r
			if i == 0 {
				gear.MoveTo(x0, y0)
			} else {
				gear.LineTo(x0, y0)
			}
			gear.LineTo(x1, y1)
		}
		gear.Close()
		ctx.DrawPath(gear, paintengine2d.StrokePaint(col, 1.4))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), w*0.14, paintengine2d.StrokePaint(col, 1.4))
	case IconExternalLink:
		// A box with a corner missing and an arrow leaving it.
		out := paintengine2d.NewPath()
		out.MoveTo(cx+0.6, b.Min.Y+3.0)
		out.LineTo(b.Min.X+3.0, b.Min.Y+3.0)
		out.LineTo(b.Min.X+3.0, b.Max.Y-3.0)
		out.LineTo(b.Max.X-3.0, b.Max.Y-3.0)
		out.LineTo(b.Max.X-3.0, cy-0.6)
		ctx.DrawPath(out, stroke)
		arr := paintengine2d.NewPath()
		arr.MoveTo(cx-0.4, cy+0.4)
		arr.LineTo(b.Max.X-2.6, b.Min.Y+2.6)
		arr.MoveTo(cx+2.8, b.Min.Y+2.6)
		arr.LineTo(b.Max.X-2.6, b.Min.Y+2.6)
		arr.LineTo(b.Max.X-2.6, b.Min.Y+6.8)
		ctx.DrawPath(arr, stroke)
	case IconEye:
		eye := paintengine2d.NewPath()
		eye.MoveTo(b.Min.X+2.0, cy)
		eye.QuadTo(cx, cy-h*0.30, b.Max.X-2.0, cy)
		eye.QuadTo(cx, cy+h*0.30, b.Min.X+2.0, cy)
		ctx.DrawPath(eye, stroke)
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), w*0.13, paintengine2d.StrokePaint(col, 1.5))
	case IconUser:
		ctx.DrawCircle(paintengine2d.Pt(cx, b.Min.Y+h*0.34), w*0.17, paintengine2d.StrokePaint(col, 1.6))
		sh := paintengine2d.NewPath()
		sh.MoveTo(b.Min.X+4.4, b.Max.Y-2.6)
		sh.QuadTo(cx, b.Max.Y-8.4, b.Max.X-4.4, b.Max.Y-2.6)
		ctx.DrawPath(sh, stroke)
	case IconBell:
		// The dome sits high and narrow and the skirt flares wide at the
		// bottom: a dome of even width reads as a lampshade.
		bell := paintengine2d.NewPath()
		bell.MoveTo(b.Min.X+3.0, b.Max.Y-6.0)
		bell.QuadTo(b.Min.X+5.6, b.Max.Y-7.4, b.Min.X+5.6, cy-1.6)
		bell.QuadTo(b.Min.X+5.6, b.Min.Y+3.0, cx, b.Min.Y+3.0)
		bell.QuadTo(b.Max.X-5.6, b.Min.Y+3.0, b.Max.X-5.6, cy-1.6)
		bell.QuadTo(b.Max.X-5.6, b.Max.Y-7.4, b.Max.X-3.0, b.Max.Y-6.0)
		bell.Close()
		ctx.DrawPath(bell, stroke)
		// The clapper, and the little stud on top.
		clap := paintengine2d.NewPath()
		clap.MoveTo(cx-1.8, b.Max.Y-5.2)
		clap.QuadTo(cx, b.Max.Y-2.4, cx+1.8, b.Max.Y-5.2)
		ctx.DrawPath(clap, stroke)
		ctx.DrawPath(lineP(cx, b.Min.Y+1.6, cx, b.Min.Y+3.0), stroke)
	case IconSend:
		// A paper plane.
		plane := paintengine2d.NewPath()
		plane.MoveTo(b.Max.X-2.2, b.Min.Y+2.2)
		plane.LineTo(b.Min.X+2.2, cy+0.6)
		plane.LineTo(cx-0.4, cy+1.6)
		plane.LineTo(b.Max.X-4.6, b.Max.Y-2.2)
		plane.Close()
		ctx.DrawPath(plane, stroke)
		ctx.DrawPath(lineP(b.Max.X-2.2, b.Min.Y+2.2, cx-0.4, cy+1.6), stroke)
	case IconClose:
		d := w * 0.26
		ctx.DrawPath(lineP(cx-d, cy-d, cx+d, cy+d), stroke)
		ctx.DrawPath(lineP(cx+d, cy-d, cx-d, cy+d), stroke)
	case IconQuit:
		// A door with an arrow leaving it.
		door := paintengine2d.NewPath()
		door.MoveTo(cx-0.6, b.Min.Y+2.6)
		door.LineTo(b.Min.X+2.8, b.Min.Y+2.6)
		door.LineTo(b.Min.X+2.8, b.Max.Y-2.6)
		door.LineTo(cx-0.6, b.Max.Y-2.6)
		ctx.DrawPath(door, stroke)
		arr := paintengine2d.NewPath()
		arr.MoveTo(cx-1.0, cy)
		arr.LineTo(b.Max.X-2.8, cy)
		arr.MoveTo(b.Max.X-6.0, cy-3.2)
		arr.LineTo(b.Max.X-2.8, cy)
		arr.LineTo(b.Max.X-6.0, cy+3.2)
		ctx.DrawPath(arr, stroke)
	case IconMore:
		// Three dots in a row, which is what every desktop draws an
		// overflow as.
		r := w * 0.065
		for _, dx := range []float32{-w * 0.22, 0, w * 0.22} {
			ctx.DrawCircle(paintengine2d.Pt(cx+dx, cy), r, fill)
		}
	case IconMenu:
		// A hamburger: three full-width bars.
		bw := w * 0.58
		bh := max(h*0.055, 1.4)
		for _, dy := range []float32{-h * 0.18, 0, h * 0.18} {
			ctx.DrawRect(paintengine2d.XYWH(cx-bw*0.5, cy+dy-bh*0.5, bw, bh), fill)
		}
	case IconLayout, IconColumns, IconRows, IconTable, IconCards:
		// A frame with the divisions each one names, drawn from one
		// outline so the family reads as a family.
		out := b.Inset(3)
		lw := max(h*0.055, 1.3)
		ctx.DrawRoundRect(out, 2, 2, paintengine2d.StrokePaint(col, lw))
		vx := out.Min.X + out.Dx()*0.38
		hy := out.Min.Y + out.Dy()*0.38
		switch icon {
		case IconColumns:
			ctx.DrawRect(paintengine2d.XYWH(vx, out.Min.Y, lw, out.Dy()), fill)
		case IconRows:
			ctx.DrawRect(paintengine2d.XYWH(out.Min.X, hy, out.Dx(), lw), fill)
		case IconLayout:
			// A side bar and a panel under the rest.
			ctx.DrawRect(paintengine2d.XYWH(vx, out.Min.Y, lw, out.Dy()), fill)
			ctx.DrawRect(paintengine2d.XYWH(vx, out.Max.Y-out.Dy()*0.32, out.Max.X-vx, lw), fill)
		case IconTable:
			ctx.DrawRect(paintengine2d.XYWH(out.Min.X, hy, out.Dx(), lw), fill)
			for _, f := range []float32{0.36, 0.68} {
				x := out.Min.X + out.Dx()*f
				ctx.DrawRect(paintengine2d.XYWH(x, out.Min.Y, lw, out.Dy()), fill)
			}
		case IconCards:
			ctx.DrawRect(paintengine2d.XYWH(out.Min.X+out.Dx()*0.5-lw*0.5, out.Min.Y, lw, out.Dy()), fill)
			ctx.DrawRect(paintengine2d.XYWH(out.Min.X, out.Min.Y+out.Dy()*0.5-lw*0.5, out.Dx(), lw), fill)
		}
	case IconArrowLeft, IconArrowRight, IconArrowUp, IconArrowDown:
		// A shaft and a head, in the direction asked for.
		var dx, dy float32
		switch icon {
		case IconArrowLeft:
			dx = -1
		case IconArrowRight:
			dx = 1
		case IconArrowUp:
			dy = -1
		default:
			dy = 1
		}
		r := w * 0.30
		tipX, tipY := cx+dx*r, cy+dy*r
		ctx.DrawPath(lineP(cx-dx*r, cy-dy*r, tipX, tipY), stroke)
		head := paintengine2d.NewPath()
		// The two barbs are the shaft turned a quarter each way.
		head.MoveTo(tipX-dx*r*0.55-dy*r*0.55, tipY-dy*r*0.55+dx*r*0.55)
		head.LineTo(tipX, tipY)
		head.LineTo(tipX-dx*r*0.55+dy*r*0.55, tipY-dy*r*0.55-dx*r*0.55)
		ctx.DrawPath(head, stroke)
	case IconInbox:
		// A tray: the box, and the lip mail drops behind.
		tray := paintengine2d.NewPath()
		tray.MoveTo(b.Min.X+2.6, cy-h*0.22)
		tray.LineTo(b.Min.X+2.6, b.Max.Y-3.0)
		tray.LineTo(b.Max.X-2.6, b.Max.Y-3.0)
		tray.LineTo(b.Max.X-2.6, cy-h*0.22)
		ctx.DrawPath(tray, stroke)
		lip := paintengine2d.NewPath()
		lip.MoveTo(b.Min.X+2.6, cy-h*0.22)
		lip.LineTo(b.Min.X+w*0.32, cy-h*0.22)
		lip.LineTo(b.Min.X+w*0.38, cy+h*0.04)
		lip.LineTo(b.Max.X-w*0.38, cy+h*0.04)
		lip.LineTo(b.Max.X-w*0.32, cy-h*0.22)
		lip.LineTo(b.Max.X-2.6, cy-h*0.22)
		ctx.DrawPath(lip, stroke)
		ctx.DrawPath(lineP(cx, b.Min.Y+2.6, cx, cy-h*0.30), stroke)
	case IconPlus:
		d := w * 0.26
		ctx.DrawPath(lineP(cx-d, cy, cx+d, cy), stroke)
		ctx.DrawPath(lineP(cx, cy-d, cx, cy+d), stroke)
	case IconClose2:
		d := w * 0.24
		ctx.DrawPath(lineP(cx-d, cy-d, cx+d, cy+d), stroke)
		ctx.DrawPath(lineP(cx+d, cy-d, cx-d, cy+d), stroke)
	case IconLock:
		body := paintengine2d.XYWH(cx-w*0.24, cy-h*0.04, w*0.48, h*0.32)
		ctx.DrawRoundRect(body, 2, 2, paintengine2d.StrokePaint(col, 1.5))
		sh := paintengine2d.NewPath()
		sh.MoveTo(cx-w*0.14, cy-h*0.04)
		sh.LineTo(cx-w*0.14, cy-h*0.18)
		sh.QuadTo(cx, cy-h*0.36, cx+w*0.14, cy-h*0.18)
		sh.LineTo(cx+w*0.14, cy-h*0.04)
		ctx.DrawPath(sh, stroke)
	case IconSync:
		// A ring open at the top with an arrowhead on one end: reload,
		// which is not redo.
		r := w * 0.30
		arc := paintengine2d.NewPath()
		arc.MoveTo(cx+r*0.30, cy-r)
		arc.QuadTo(cx-r*1.35, cy-r*1.05, cx-r, cy)
		arc.QuadTo(cx-r*0.9, cy+r*1.35, cx, cy+r)
		arc.QuadTo(cx+r*1.35, cy+r*0.9, cx+r, cy-r*0.10)
		ctx.DrawPath(arc, stroke)
		head := paintengine2d.NewPath()
		head.MoveTo(cx+r*0.30, cy-r*1.55)
		head.LineTo(cx+r*0.95, cy-r*0.92)
		head.LineTo(cx+r*0.05, cy-r*0.55)
		head.Close()
		ctx.DrawPath(head, fill)
	case IconStarFilled:
		ctx.DrawPath(starPath(cx, cy, w*0.44, w*0.18), fill)
	case IconDot:
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), w*0.22, fill)
	case IconPrint:
		// A printer: the sheet going in, the body, the sheet coming out.
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+5.4, b.Min.Y+2.2, w-10.8, 3.6),
			paintengine2d.StrokePaint(col, 1.5))
		ctx.DrawRoundRect(paintengine2d.XYWH(b.Min.X+2.4, b.Min.Y+5.8, w-4.8, h*0.30), 1.5, 1.5,
			paintengine2d.StrokePaint(col, 1.5))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+5.4, b.Max.Y-6.6, w-10.8, 4.4),
			paintengine2d.StrokePaint(col, 1.5))
	default:
		// A stem-only icon ([IconByStem]) in a drawn set: the toolkit has
		// no vector for it, and drawing nothing would leave a button with
		// an invisible mark on it. The missing-icon square says so.
		drawNoIconMark(ctx, b, col)
	}
}

// drawNoIconMark is the placeholder a drawn set uses for an icon it has
// no vector for: a dashed square with a cross in it, which is the same
// mark the file sets ship as no-icon.png.
func drawNoIconMark(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	stroke := iconStroke(col, 1.5, paintengine2d.CapRound, paintengine2d.JoinRound)
	r := b.Inset(2.5)
	ctx.DrawRoundRect(r, 2, 2, paintengine2d.StrokePaint(col.WithAlpha(col.A*0.7), 1.4))
	d := r.Dx() * 0.26
	cx, cy := (r.Min.X+r.Max.X)*0.5, (r.Min.Y+r.Max.Y)*0.5
	ctx.DrawPath(lineP(cx-d, cy-d, cx+d, cy+d), stroke)
	ctx.DrawPath(lineP(cx+d, cy-d, cx-d, cy+d), stroke)
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
	default:
		// A shape this set has no drawing of its own for. The classic
		// vector is a better answer than an empty box, and better than a
		// placeholder: it is the same action, drawn in the other hand.
		drawClassicIcon(ctx, b, icon, col)
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

// lineP is a two-point path, which most of the drawn icons are made of.
func lineP(x0, y0, x1, y1 float32) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	p.MoveTo(x0, y0)
	p.LineTo(x1, y1)
	return p
}

// IconSizeOf is the chrome icon size a look draws marks at (look.json
// "iconSize"), in 1x design pixels — what a tool button and a browser
// tab's mark are sized from, so anything else drawing a mark on chrome
// can agree with them.
func IconSizeOf(look LookAndFeel) float32 {
	if c, ok := look.(*Classic); ok && c != nil {
		return IconSizePixels(c.IconSize())
	}
	return IconSizePixels(IconSizeMedium)
}
