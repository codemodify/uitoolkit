package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// scrollDrag is shared thumb/track interaction for overflow bars.
type scrollDrag struct {
	active bool
	grab   float32
	over   bool
}

func overflowBarSize(lk style.LookAndFeel) (bar, gap float32) {
	bar = float32(10)
	if lk != nil {
		if v := lk.Metrics().Scroll; v > 0 {
			bar = v
		}
	}
	return bar, 2
}

func vScrollThumb(b paintengine2d.Rect, content, offset, bar, gap float32) (track, thumb paintengine2d.Rect) {
	if b.Empty() || bar <= 0 {
		return
	}
	h := b.Dy()
	if h < 16 {
		return
	}
	track = paintengine2d.XYWH(b.Max.X-bar-gap, b.Min.Y+4, bar, h-8)
	maxOff := layout.MaxScroll(content, h)
	if maxOff <= 0 || content <= 0 {
		return track, paintengine2d.Rect{}
	}
	frac := h / content
	th := track.Dy() * frac
	if th < 24 {
		th = 24
	}
	if th > track.Dy() {
		th = track.Dy()
	}
	ty := track.Min.Y
	if track.Dy() > th {
		ty += (track.Dy() - th) * (offset / maxOff)
	}
	thumb = paintengine2d.XYWH(track.Min.X, ty, track.Dx(), th)
	return
}

func hScrollThumb(b paintengine2d.Rect, content, offset, bar, gap float32) (track, thumb paintengine2d.Rect) {
	if b.Empty() || bar <= 0 {
		return
	}
	w := b.Dx()
	if w < 16 {
		return
	}
	track = paintengine2d.XYWH(b.Min.X+4, b.Max.Y-bar-gap, w-8, bar)
	maxOff := layout.MaxScroll(content, w)
	if maxOff <= 0 || content <= 0 {
		return track, paintengine2d.Rect{}
	}
	frac := w / content
	tw := track.Dx() * frac
	if tw < 24 {
		tw = 24
	}
	if tw > track.Dx() {
		tw = track.Dx()
	}
	tx := track.Min.X
	if track.Dx() > tw {
		tx += (track.Dx() - tw) * (offset / maxOff)
	}
	thumb = paintengine2d.XYWH(tx, track.Min.Y, tw, track.Dy())
	return
}

func paintOverflowBar(ctx *paintengine2d.Context, lk style.LookAndFeel, track, thumb paintengine2d.Rect, hovered, pressed bool) {
	if ctx == nil || lk == nil || thumb.Empty() {
		return
	}
	st := style.ControlState(0)
	if hovered {
		st |= style.StateHovered
	}
	if pressed {
		st |= style.StatePressed
	}
	lk.DrawScrollBar(ctx, track, thumb, st)
}

func clampOff(offset, maxOff float32) float32 {
	if offset < 0 {
		return 0
	}
	if offset > maxOff {
		return maxOff
	}
	return offset
}

func (d *scrollDrag) press(pos paintengine2d.Point, track, thumb paintengine2d.Rect, alongY bool, offset, maxOff, page float32) (float32, bool) {
	if !thumb.Empty() && thumb.Contains(pos) {
		d.active = true
		if alongY {
			d.grab = pos.Y - thumb.Min.Y
		} else {
			d.grab = pos.X - thumb.Min.X
		}
		return offset, true
	}
	if track.Empty() || !track.Contains(pos) || thumb.Empty() {
		return offset, false
	}
	if alongY {
		if pos.Y < thumb.Min.Y {
			return clampOff(offset-page, maxOff), true
		}
		return clampOff(offset+page, maxOff), true
	}
	if pos.X < thumb.Min.X {
		return clampOff(offset-page, maxOff), true
	}
	return clampOff(offset+page, maxOff), true
}

func (d *scrollDrag) move(pos paintengine2d.Point, track, thumb paintengine2d.Rect, alongY bool, maxOff float32) (offset float32, apply, handled, hoverDirty bool) {
	over := !track.Empty() && track.Contains(pos)
	if over != d.over {
		d.over = over
		hoverDirty = true
	}
	if !d.active {
		return 0, false, over, hoverDirty
	}
	var span, t float32
	if alongY {
		span = track.Dy() - thumb.Dy()
		if span <= 0 {
			return 0, true, true, hoverDirty
		}
		t = (pos.Y - d.grab - track.Min.Y) / span
	} else {
		span = track.Dx() - thumb.Dx()
		if span <= 0 {
			return 0, true, true, hoverDirty
		}
		t = (pos.X - d.grab - track.Min.X) / span
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * maxOff, true, true, hoverDirty
}

// applyScrollHover updates thumb-drag offset or track hover. invalidateAll
// runs when the offset changes; invalidateBar when only the thumb chrome
// changes. Every move over the track used to call invalidateAll.
func applyScrollHover(d *scrollDrag, pos paintengine2d.Point, track, thumb paintengine2d.Rect, alongY bool, maxOff float32, setOff func(float32), invalidateAll, invalidateBar func()) bool {
	if d == nil {
		return false
	}
	off, apply, handled, hoverDirty := d.move(pos, track, thumb, alongY, maxOff)
	if apply {
		if setOff != nil {
			setOff(off)
		}
		// setOff owns invalidation when the widget blits (List/Table/Tree/
		// TextArea). CardList and other fallbacks still pass invalidateAll.
		if invalidateAll != nil {
			invalidateAll()
		}
	} else if hoverDirty && invalidateBar != nil {
		invalidateBar()
	}
	return apply || handled
}

func invalidateOverflowTrack(c interface{ InvalidateRect(paintengine2d.Rect) }, track paintengine2d.Rect) {
	if c == nil || track.Empty() {
		return
	}
	c.InvalidateRect(track.Inset(-2))
}

func invalidateOverflowThumbs(c interface{ InvalidateRect(paintengine2d.Rect) }, oldThumb, newThumb paintengine2d.Rect) {
	if c == nil {
		return
	}
	if !oldThumb.Empty() {
		c.InvalidateRect(oldThumb.Inset(-2))
	}
	if !newThumb.Empty() && (newThumb.Min != oldThumb.Min || newThumb.Max != oldThumb.Max) {
		c.InvalidateRect(newThumb.Inset(-2))
	}
}

func (d *scrollDrag) release() bool {
	if !d.active {
		return false
	}
	d.active = false
	return true
}

func wheelDelta(scrollY, line float32) float32 {
	dy := scrollY
	if dy > -8 && dy < 8 && dy != 0 {
		dy *= line * 3
	}
	return dy
}
