package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/widget"
)

// contentViewMinusVBar shrinks b so a right-hand overflow track is not
// included in a Context.Scroll blit (the thumb would smear).
func contentViewMinusVBar(b, track paintengine2d.Rect) paintengine2d.Rect {
	if !track.Empty() && track.Min.X > b.Min.X && track.Min.X < b.Max.X {
		b.Max.X = track.Min.X
	}
	return b
}

func contentViewMinusHBar(b, track paintengine2d.Rect) paintengine2d.Rect {
	if !track.Empty() && track.Min.Y > b.Min.Y && track.Min.Y < b.Max.Y {
		b.Max.Y = track.Min.Y
	}
	return b
}

// pixelScrollEnabled gates Context.Scroll + strip-only damage.
// v0.14.1–v0.14.3 left vacated holes on Wayland/HiDPI (logical vs
// buffer px, GPU present tiles, DrawSceneDamage skipping the blit
// middle). Keep off until a scaled blit + strip replay is proven.
const pixelScrollEnabled = false

// commitAxisScroll applies a clamped offset. The pixel-scroll blit is
// disabled; the viewport is fully invalidated so the next frame
// repaints every visible row (hover still uses small dirty rects).
func commitAxisScroll(c widget.Component, view paintengine2d.Rect, old, want, max float32, apply func(float32), vertical bool) {
	want = clampOff(want, max)
	if want == old {
		return
	}
	if apply != nil {
		apply(want)
	}
	dx, dy := float32(0), float32(0)
	if vertical {
		dy = old - want
	} else {
		dx = old - want
	}
	if tryPixelScroll(c, view, dx, dy) {
		if strip := exposedStrip(view, dx, dy); !strip.Empty() {
			c.InvalidateRect(strip)
		}
		return
	}
	c.Invalidate()
}

func commitScrollXY(c widget.Component, view paintengine2d.Rect, oldX, oldY, wantX, wantY, maxX, maxY float32, apply func(x, y float32)) {
	wantX = clampOff(wantX, maxX)
	wantY = clampOff(wantY, maxY)
	if wantX == oldX && wantY == oldY {
		return
	}
	if apply != nil {
		apply(wantX, wantY)
	}
	dx := oldX - wantX
	dy := oldY - wantY
	if tryPixelScroll(c, view, dx, dy) {
		if strip := exposedStrip(view, dx, 0); !strip.Empty() {
			c.InvalidateRect(strip)
		}
		if strip := exposedStrip(view, 0, dy); !strip.Empty() {
			c.InvalidateRect(strip)
		}
		return
	}
	c.Invalidate()
}

func tryPixelScroll(c widget.Component, view paintengine2d.Rect, dx, dy float32) bool {
	if !pixelScrollEnabled {
		return false
	}
	if c == nil || view.Empty() {
		return false
	}
	if dx == 0 && dy == 0 {
		return true
	}
	if (dx != 0 && abs32(dx) >= view.Dx()) || (dy != 0 && abs32(dy) >= view.Dy()) {
		return false
	}
	if !pixelDeltaOK(dx) || !pixelDeltaOK(dy) {
		return false
	}
	h := c.Host()
	ps, ok := h.(widget.PixelScroller)
	if !ok || ps == nil {
		return false
	}
	return ps.ScrollPixels(c, view, dx, dy)
}

func exposedStrip(view paintengine2d.Rect, dx, dy float32) paintengine2d.Rect {
	if view.Empty() {
		return paintengine2d.Rect{}
	}
	switch {
	case dy < 0:
		h := -dy
		if h > view.Dy() {
			h = view.Dy()
		}
		return paintengine2d.XYWH(view.Min.X, view.Max.Y-h, view.Dx(), h)
	case dy > 0:
		h := dy
		if h > view.Dy() {
			h = view.Dy()
		}
		return paintengine2d.XYWH(view.Min.X, view.Min.Y, view.Dx(), h)
	case dx < 0:
		w := -dx
		if w > view.Dx() {
			w = view.Dx()
		}
		return paintengine2d.XYWH(view.Max.X-w, view.Min.Y, w, view.Dy())
	case dx > 0:
		w := dx
		if w > view.Dx() {
			w = view.Dx()
		}
		return paintengine2d.XYWH(view.Min.X, view.Min.Y, w, view.Dy())
	default:
		return paintengine2d.Rect{}
	}
}

func pixelDeltaOK(v float32) bool {
	i := int(v)
	if v < 0 && float32(i) != v {
		i--
	}
	d := v - float32(i)
	if d < 0 {
		d = -d
	}
	return d <= 1e-4
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
