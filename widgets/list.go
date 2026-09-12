package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
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
	OnContext func(i int, windowPos paintengine2d.Point)
	OffsetY   float32
	hovered   int
	vbar      scrollDrag
	rows      rowSceneCache
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
	return style.FittedRowHeight(l.Look(), l.RowHeight)
}

func (l *ListView) contentH() float32 { return float32(l.Count) * l.rowH() }

// MaxOffset is max(0, content − viewport).
func (l *ListView) MaxOffset() float32 {
	return layout.MaxScroll(l.contentH(), l.LocalBounds().Dy())
}

func (l *ListView) clamp() {
	l.OffsetY = layout.ClampScroll(l.OffsetY, l.contentH(), l.LocalBounds().Dy())
}

func (l *ListView) scrollTrack() (track, thumb paintengine2d.Rect) {
	bar, gap := overflowBarSize(l.Look())
	return vScrollThumb(l.LocalBounds(), l.contentH(), l.OffsetY, bar, gap)
}

// VisibleRange is the half-open [lo, hi) window of rows that Paint draws.
func (l *ListView) VisibleRange() (lo, hi int) { return l.visibleRange() }

// ScrollTrack is the overflow bar geometry (empty thumb when content fits).
func (l *ListView) ScrollTrack() (track, thumb paintengine2d.Rect) { return l.scrollTrack() }

// ScrollTo sets OffsetY (clamped) without requiring a wheel event.
func (l *ListView) ScrollTo(y float32) {
	l.OffsetY = y
	l.clamp()
	l.Invalidate()
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
	l.clamp()
	b := l.LocalBounds()
	lk := l.Look()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
	ctx.Save()
	ctx.ClipRect(b)
	rh := l.rowH()
	lo, hi := l.visibleRange()
	if rec, ok := ctx.Device().(*paintengine2d.Recorder); ok {
		ob := l.Bounds()
		l.rows.ready(ob.Min.X, ob.Min.Y, b.Dx(), rh, l.OffsetY, lookSig(lk))
		recordScrollingRows(rec, &l.rows, l.ID()^(1<<32), l.OffsetY, 0, lo, hi,
			func(i int) uint64 { return l.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 {
				label := ""
				if l.ItemText != nil {
					label = l.ItemText(i)
				}
				return visualSig(i == l.Selected, i == l.hovered, bits32(b.Dx()), label)
			},
			func(i int) {
				label := ""
				if l.ItemText != nil {
					label = l.ItemText(i)
				}
				y := float32(i)*rh - l.OffsetY
				lk.DrawListRow(ctx, paintengine2d.XYWH(0, y, b.Dx(), rh), i == l.Selected, i == l.hovered, label)
			},
		)
	} else {
		for i := lo; i < hi; i++ {
			y := float32(i)*rh - l.OffsetY
			row := paintengine2d.XYWH(0, y, b.Dx(), rh)
			if ctx.QuickReject(row) {
				continue
			}
			label := ""
			if l.ItemText != nil {
				label = l.ItemText(i)
			}
			lk.DrawListRow(ctx, row, i == l.Selected, i == l.hovered, label)
		}
	}
	ctx.Restore()
	track, thumb := l.scrollTrack()
	paintOverflowBar(ctx, lk, track, thumb, l.vbar.over, l.vbar.active)
	if l.Focused() {
		lk.DrawFocusRing(ctx, b.Inset(-2))
	}
}

func (l *ListView) indexAt(y float32) int {
	return rowIndexAt(y, l.OffsetY, l.rowH(), l.Count)
}

func (l *ListView) rowRect(i int) paintengine2d.Rect {
	if i < 0 {
		return paintengine2d.Rect{}
	}
	rh := l.rowH()
	y := float32(i)*rh - l.OffsetY
	return paintengine2d.XYWH(0, y, l.LocalBounds().Dx(), rh)
}

func (l *ListView) invalidateRow(i int) {
	if r := l.rowRect(i); !r.Empty() {
		l.InvalidateRect(r.Inset(-1))
	}
}

func (l *ListView) invalidateBar() {
	track, _ := l.scrollTrack()
	invalidateOverflowTrack(l, track)
}

func (l *ListView) MouseEnter() {}

func (l *ListView) MouseMove(e widget.MouseEvent) bool {
	track, thumb := l.scrollTrack()
	if applyScrollHover(&l.vbar, e.Pos, track, thumb, true, l.MaxOffset(), func(off float32) {
		l.OffsetY = off
		l.clamp()
	}, l.Invalidate, l.invalidateBar) {
		return true
	}
	h := l.indexAt(e.Pos.Y)
	if h != l.hovered {
		old := l.hovered
		l.hovered = h
		l.invalidateRow(old)
		l.invalidateRow(h)
	}
	return true
}

func (l *ListView) MouseExit() {
	old := l.hovered
	l.hovered = -1
	l.vbar.over = false
	l.invalidateRow(old)
}

func (l *ListView) MouseRelease(widget.MouseEvent) bool {
	if l.vbar.release() {
		l.Invalidate()
		return true
	}
	return false
}

func (l *ListView) MousePress(e widget.MouseEvent) bool {
	l.RequestFocus()
	track, thumb := l.scrollTrack()
	if off, ok := l.vbar.press(e.Pos, track, thumb, true, l.OffsetY, l.MaxOffset(), l.LocalBounds().Dy()*0.9); ok {
		l.OffsetY = off
		l.clamp()
		l.Invalidate()
		return true
	}
	i := l.indexAt(e.Pos.Y)
	if i >= 0 {
		old := l.Selected
		l.Selected = i
		l.invalidateRow(old)
		l.invalidateRow(i)
		if l.OnSelect != nil {
			l.OnSelect(i)
		}
	}
	if e.Button == platform.ButtonRight && l.OnContext != nil {
		o := widget.DeviceOrigin(l)
		l.OnContext(i, paintengine2d.Pt(o.X+e.Pos.X, o.Y+e.Pos.Y))
	}
	return true
}

func (l *ListView) MouseWheel(e widget.MouseEvent) bool {
	l.OffsetY += wheelDelta(e.Scroll.Y, l.rowH())
	l.clamp()
	l.Invalidate()
	return true
}

func (l *ListView) KeyPress(e widget.KeyEvent) bool {
	if !l.Enabled() || l.Count <= 0 {
		return false
	}
	next := l.Selected
	page := int(l.LocalBounds().Dy()/l.rowH()) - 1
	if page < 1 {
		page = 1
	}
	switch e.Key {
	case platform.KeyDown:
		next++
	case platform.KeyUp:
		next--
	case platform.KeyPageDown:
		next += page
	case platform.KeyPageUp:
		next -= page
	case platform.KeyHome:
		next = 0
	case platform.KeyEnd:
		next = l.Count - 1
	case platform.KeyReturn, platform.KeySpace:
		if l.Selected >= 0 && l.OnSelect != nil {
			l.OnSelect(l.Selected)
		}
		return true
	default:
		return false
	}
	if next < 0 {
		next = 0
	}
	if next >= l.Count {
		next = l.Count - 1
	}
	if next != l.Selected {
		old := l.Selected
		off := l.OffsetY
		l.Selected = next
		l.ensureVisible(next)
		if l.OffsetY != off {
			l.Invalidate()
		} else {
			l.invalidateRow(old)
			l.invalidateRow(next)
		}
		if l.OnSelect != nil {
			l.OnSelect(next)
		}
	}
	return true
}

func (l *ListView) ensureVisible(i int) {
	rh := l.rowH()
	top := float32(i) * rh
	bot := top + rh
	view := l.LocalBounds().Dy()
	if top < l.OffsetY {
		l.OffsetY = top
	}
	if bot > l.OffsetY+view {
		l.OffsetY = bot - view
	}
	l.clamp()
}
