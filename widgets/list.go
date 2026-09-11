package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// ListView is a virtualized fixed-height row list. Only visible rows paint.
type ListView struct {
	widget.Base
	Count     int
	RowHeight float32
	Selected  int
	ItemText  func(i int) string
	OnSelect  func(i int)
	OffsetY   float32
	hovered   int
}

func NewListView(count int, text func(int) string, on func(int)) *ListView {
	l := &ListView{Count: count, RowHeight: 28, Selected: -1, ItemText: text, OnSelect: on, hovered: -1}
	l.Init(l)
	l.SetWantsFocus(true)
	return l
}

func (l *ListView) Measure(c layout.Constraints) paintengine2d.Point {
	h := float32(l.Count) * l.rowH()
	if c.HasMaxH() && h > c.MaxH {
		h = c.MaxH
	}
	w := float32(200)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (l *ListView) Arrange(r paintengine2d.Rect) { l.SetBounds(r); l.clamp() }

func (l *ListView) rowH() float32 {
	if l.RowHeight <= 0 {
		return 28
	}
	return l.RowHeight
}

func (l *ListView) contentH() float32 { return float32(l.Count) * l.rowH() }

func (l *ListView) clamp() {
	mx := l.contentH() - l.LocalBounds().Dy()
	if mx < 0 {
		mx = 0
	}
	if l.OffsetY < 0 {
		l.OffsetY = 0
	}
	if l.OffsetY > mx {
		l.OffsetY = mx
	}
}

func (l *ListView) visibleRange() (lo, hi int) {
	rh := l.rowH()
	if rh <= 0 {
		return 0, 0
	}
	lo = int(l.OffsetY / rh)
	hi = int((l.OffsetY+l.LocalBounds().Dy())/rh) + 1
	if lo < 0 {
		lo = 0
	}
	if hi > l.Count {
		hi = l.Count
	}
	return
}

func (l *ListView) Paint(ctx *paintengine2d.Context) {
	b := l.LocalBounds()
	lk := l.Look()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
	ctx.Save()
	ctx.ClipRect(b)
	rh := l.rowH()
	lo, hi := l.visibleRange()
	for i := lo; i < hi; i++ {
		y := float32(i)*rh - l.OffsetY
		row := paintengine2d.XYWH(0, y, b.Dx(), rh)
		label := ""
		if l.ItemText != nil {
			label = l.ItemText(i)
		}
		lk.DrawListRow(ctx, row, i == l.Selected, i == l.hovered, label)
	}
	ctx.Restore()
}

func (l *ListView) indexAt(y float32) int {
	i := int((y + l.OffsetY) / l.rowH())
	if i < 0 || i >= l.Count {
		return -1
	}
	return i
}

func (l *ListView) MouseMove(e widget.MouseEvent) bool {
	h := l.indexAt(e.Pos.Y)
	if h != l.hovered {
		l.hovered = h
		l.Invalidate()
	}
	return true
}

func (l *ListView) MouseExit() { l.hovered = -1; l.Invalidate() }

func (l *ListView) MousePress(e widget.MouseEvent) bool {
	l.RequestFocus()
	i := l.indexAt(e.Pos.Y)
	if i >= 0 {
		l.Selected = i
		l.Invalidate()
		if l.OnSelect != nil {
			l.OnSelect(i)
		}
	}
	return true
}

func (l *ListView) MouseWheel(e widget.MouseEvent) bool {
	l.OffsetY += e.Scroll.Y
	l.clamp()
	l.Invalidate()
	return true
}
