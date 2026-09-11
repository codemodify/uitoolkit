package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

func drawPreeditBar(ctx *paintengine2d.Context, look style.LookAndFeel, b paintengine2d.Rect, pad float32, text string, a, b0 int, scrollX float32, multiline bool, face *style.Font) {
	if a == b0 || look == nil {
		return
	}
	if a > b0 {
		a, b0 = b0, a
	}
	f := face
	if f == nil {
		f = look.Font()
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	if multiline {
		inner = paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+pad, b.Dx()-pad*2, b.Dy()-pad*2)
	}
	ctx.Save()
	ctx.ClipRect(inner)
	x0 := inner.Min.X - scrollX + f.CaretX(text, a)
	x1 := inner.Min.X - scrollX + f.CaretX(text, b0)
	if x1 < x0 {
		x1 = x0
	}
	y := inner.Min.Y + (inner.Dy()+f.Height())*0.5 + 1
	if multiline {
		y = inner.Max.Y - 3
	}
	col := look.Palette().Accent
	ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, 1.6), paintengine2d.Fill(col))
	ctx.Restore()
}
