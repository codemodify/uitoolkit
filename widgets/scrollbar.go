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

// Transient bars (style.ScrollBarStyle.Transient) stay a second after the
// view last scrolled or the pointer last moved over it, then fade out in
// a few steps.
const (
	transientRest  = time.Second
	transientStep  = 40 * time.Millisecond
	transientSteps = 5
)

// scrollNow is the clock transient bars rest by (tests replace it).
var scrollNow = time.Now

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

	// A transient bar's visibility: reveal runs from 0 (hidden) to 1,
	// woke is when the view last scrolled or the pointer last moved over
	// it, and resting reports a fade-out timer is pending.
	reveal  float32
	woke    time.Time
	resting bool
	lastOff float32
	painted bool
}

// transient reports whether lk's bars come and go over the content.
func transient(lk style.LookAndFeel) bool { return style.ScrollBarStyleOf(lk).Transient }

// wake shows a transient bar and restarts its rest; owner repaints and
// times the fade.
func (d *scrollDrag) wake(owner widget.Component) {
	if owner == nil || !transient(owner.Look()) {
		return
	}
	d.woke = scrollNow()
	if d.reveal < 1 {
		d.reveal = 1
		repaintBar(owner)
	}
	if !d.resting {
		d.resting = true
		d.rest(owner, transientRest)
	}
}

// rest waits out the rest period (restarted by every wake), then fades
// the bar out step by step. A hovered or held bar stays.
func (d *scrollDrag) rest(owner widget.Component, after time.Duration) {
	widget.After(owner, after, func() {
		if left := transientRest - scrollNow().Sub(d.woke); left > 0 {
			d.rest(owner, left)
			return
		}
		if d.over || d.active || d.down != style.ScrollNone {
			d.rest(owner, transientRest)
			return
		}
		d.reveal -= 1.0 / transientSteps
		if !style.Animations() {
			d.reveal = 0 // reduced motion: gone at once, no fade
		}
		if d.reveal <= 0.001 {
			d.reveal = 0
			d.resting = false
		} else {
			d.rest(owner, transientStep)
		}
		repaintBar(owner)
	})
}

// repaintBar repaints the owner without dropping its retained rows (the
// views' own Invalidate re-records every row).
func repaintBar(owner widget.Component) {
	if r, ok := owner.(interface{ InvalidateRect(paintengine2d.Rect) }); ok {
		r.InvalidateRect(owner.LocalBounds())
	}
}

// vScrollParts lays out a vertical bar at the right edge of view.
func vScrollParts(lk style.LookAndFeel, view paintengine2d.Rect, content, offset float32) style.ScrollParts {
	return style.ScrollGeometry(lk, view, true, content, view.Dy(), offset, false)
}

// scrollGutter is the width content gives up to a visible bar.
func scrollGutter(lk style.LookAndFeel, overflow bool) float32 {
	if !overflow {
		return 0
	}
	return style.ScrollGutter(lk)
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
	case style.ScrollDec, style.ScrollDecEnd, style.ScrollInc, style.ScrollPageDec, style.ScrollPageInc:
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
	case style.ScrollDec, style.ScrollDecEnd:
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
// on (or dragging) the bar; dirty that the bar needs a repaint. Pointer
// motion over a scrollable view shows its transient bar.
func (d *scrollDrag) move(owner widget.Component, pos paintengine2d.Point, ax scrollAxis) (handled, dirty bool) {
	if ax.parts == nil {
		return false, false
	}
	sp := ax.parts()
	if !sp.Thumb.Empty() {
		d.wake(owner)
	}
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
// scroll). offset is the axis' scroll offset: when it moved since the last
// paint, a transient bar shows; a hidden one draws nothing and a fading one
// draws at its reveal.
func (d *scrollDrag) paint(owner widget.Component, ctx *paintengine2d.Context, lk style.LookAndFeel, sp style.ScrollParts, vertical bool, offset float32) {
	if ctx == nil || lk == nil || sp.Thumb.Empty() {
		return
	}
	pressed := d.down
	if d.active {
		pressed = style.ScrollThumbPart
	}
	st := style.ScrollState{Hot: d.hot, Pressed: pressed, Hovered: d.over}
	if !transient(lk) {
		style.DrawScrollBarParts(lk, ctx, sp, vertical, st)
		return
	}
	if d.painted && offset != d.lastOff {
		d.wake(owner)
	}
	d.painted, d.lastOff = true, offset
	reveal := d.reveal
	if d.over || d.active || d.down != style.ScrollNone {
		reveal = 1
	}
	if reveal <= 0 {
		return
	}
	ctx.Save()
	ctx.SetAlpha(reveal)
	style.DrawScrollBarParts(lk, ctx, sp, vertical, st)
	ctx.Restore()
}

// wheelDelta is how far a wheel or touchpad event scrolls: a touchpad's
// pixels as they are (the content follows the fingers), wheel notches
// three lines each. Large non-precise values are pixels from backends that
// send them.
func wheelDelta(scrollY, line float32, precise bool) float32 {
	dy := scrollY
	if !precise && dy > -8 && dy < 8 && dy != 0 {
		dy *= line * 3
	}
	return dy
}
