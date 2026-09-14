package widgets

import (
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Scrollbar geometry comes from the look (style.ScrollGeometry): thickness,
// overlay vs gutter, and where the step buttons sit are theme decisions —
// Win95 has an arrow at each end, Aqua groups them, Breeze has none.

// Auto-repeat timing for step buttons and track paging (Windows defaults).
const (
	scrollRepeatDelay = 400 * time.Millisecond
	scrollRepeatEvery = 50 * time.Millisecond
)

// scrollAxis is how a widget exposes one scrollable axis to its bar.
type scrollAxis struct {
	vertical bool
	parts    func() style.ScrollParts        // current geometry
	get      func() (offset, maxOff float32) // current offset / limit
	set      func(offset float32)            // scroll (clamps, invalidates)
	steps    func() (line, page float32)     // arrow step / track page
}

// scrollDrag drives one scrollbar: thumb drag, track paging, and
// auto-repeating step buttons.
type scrollDrag struct {
	active bool             // thumb drag in progress
	grab   float32          // pointer offset inside the thumb at press
	over   bool             // pointer over the bar
	hot    style.ScrollPart // part under the pointer
	down   style.ScrollPart // part held (step / page auto-repeat)
	stop   func()           // cancels a pending auto-repeat
}

// vScrollParts lays out a vertical bar at the right edge of view.
func vScrollParts(lk style.LookAndFeel, view paintengine2d.Rect, content, offset float32) style.ScrollParts {
	return style.ScrollGeometry(lk, view, true, content, view.Dy(), offset, false)
}

// hScrollParts lays out a horizontal bar at the bottom edge of view.
func hScrollParts(lk style.LookAndFeel, view paintengine2d.Rect, content, offset float32) style.ScrollParts {
	return style.ScrollGeometry(lk, view, false, content, view.Dx(), offset, false)
}

// scrollGutter is the width content gives up to a visible bar.
func scrollGutter(lk style.LookAndFeel, overflow bool) float32 {
	if !overflow {
		return 0
	}
	return style.ScrollGutter(lk)
}

// overflowBarSize is the bar thickness and the gap to the edge (layout
// helpers that reserve room for a bar next to their content).
func overflowBarSize(lk style.LookAndFeel) (bar, gap float32) {
	s := style.ScrollBarStyleOf(lk)
	if s.Overlay {
		return s.Thickness, s.Inset
	}
	return s.Thickness, 0
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

// press handles a button press on the bar. owner is the widget (timers,
// focus); it reports whether the press landed on the bar.
func (d *scrollDrag) press(owner widget.Component, pos paintengine2d.Point, ax scrollAxis) bool {
	if ax.parts == nil {
		return false
	}
	sp := ax.parts()
	part := sp.HitTest(pos, ax.vertical)
	switch part {
	case style.ScrollThumbPart:
		d.active = true
		d.down = part
		if ax.vertical {
			d.grab = pos.Y - sp.Thumb.Min.Y
		} else {
			d.grab = pos.X - sp.Thumb.Min.X
		}
		return true
	case style.ScrollDec, style.ScrollInc, style.ScrollPageDec, style.ScrollPageInc:
		d.down = part
		d.hot = part
		d.step(ax, part, pos)
		d.cancel()
		d.schedule(owner, ax, part, pos, scrollRepeatDelay)
		return true
	}
	// A press on an empty (nothing to scroll) bar still belongs to it.
	return !sp.Bar.Empty() && sp.Bar.Contains(pos) && sp.Thumb.Empty() && !sp.Track.Empty()
}

// step applies one arrow step or page for part.
func (d *scrollDrag) step(ax scrollAxis, part style.ScrollPart, pos paintengine2d.Point) bool {
	off, maxOff := ax.get()
	line, page := ax.steps()
	next := off
	switch part {
	case style.ScrollDec:
		next = off - line
	case style.ScrollInc:
		next = off + line
	case style.ScrollPageDec, style.ScrollPageInc:
		// Page only while the pointer is still beyond the thumb: holding
		// the track stops when the thumb arrives under the pointer.
		sp := ax.parts()
		if sp.HitTest(pos, ax.vertical) != part {
			return false
		}
		if part == style.ScrollPageDec {
			next = off - page
		} else {
			next = off + page
		}
	}
	next = clampOff(next, maxOff)
	if next == off {
		return false
	}
	ax.set(next)
	return true
}

func (d *scrollDrag) schedule(owner widget.Component, ax scrollAxis, part style.ScrollPart, pos paintengine2d.Point, after time.Duration) {
	d.stop = widget.After(owner, after, func() {
		if d.down != part {
			return
		}
		if !d.step(ax, part, pos) {
			return
		}
		d.schedule(owner, ax, part, pos, scrollRepeatEvery)
	})
}

func (d *scrollDrag) cancel() {
	if d.stop != nil {
		d.stop()
		d.stop = nil
	}
}

// move tracks hover and drags the thumb. handled reports the pointer is
// on (or dragging) the bar; dirty that the bar needs a repaint.
func (d *scrollDrag) move(pos paintengine2d.Point, ax scrollAxis) (handled, dirty bool) {
	if ax.parts == nil {
		return false, false
	}
	sp := ax.parts()
	over := !sp.Bar.Empty() && sp.Bar.Contains(pos)
	hot := style.ScrollNone
	if over {
		hot = sp.HitTest(pos, ax.vertical)
	}
	if over != d.over || hot != d.hot {
		d.over, d.hot = over, hot
		dirty = true
	}
	if !d.active {
		return over || d.down != style.ScrollNone, dirty
	}
	_, maxOff := ax.get()
	start := pos.X - d.grab
	if ax.vertical {
		start = pos.Y - d.grab
	}
	ax.set(style.ScrollOffsetForThumb(sp, ax.vertical, start, maxOff))
	return true, true
}

// release ends a drag or auto-repeat; true when the bar owned the press.
func (d *scrollDrag) release() bool {
	owned := d.active || d.down != style.ScrollNone
	d.active = false
	d.down = style.ScrollNone
	d.cancel()
	return owned
}

// exit clears hover when the pointer leaves the widget.
func (d *scrollDrag) exit() {
	d.over = false
	d.hot = style.ScrollNone
}

// paint draws the bar from its geometry (nothing when there is nothing to
// scroll).
func (d *scrollDrag) paint(ctx *paintengine2d.Context, lk style.LookAndFeel, sp style.ScrollParts, vertical bool) {
	if ctx == nil || lk == nil || sp.Thumb.Empty() {
		return
	}
	pressed := d.down
	if d.active {
		pressed = style.ScrollThumbPart
	}
	style.DrawScrollBarParts(lk, ctx, sp, vertical, style.ScrollState{Hot: d.hot, Pressed: pressed, Hovered: d.over})
}

func wheelDelta(scrollY, line float32) float32 {
	dy := scrollY
	if dy > -8 && dy < 8 && dy != 0 {
		dy *= line * 3
	}
	return dy
}
