package app

import (
	"fmt"
	"os"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// diffPixels counts pixels that differ between two images.
func diffPixels(t *testing.T, a, b *paintengine2d.Image) int {
	t.Helper()
	if a == nil || b == nil {
		t.Fatal("nil image")
	}
	if a.Width != b.Width || a.Height != b.Height {
		t.Fatalf("size %dx%d vs %dx%d", a.Width, a.Height, b.Width, b.Height)
	}
	n := 0
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			if a.NRGBAAt(x, y) != b.NRGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}

// fullRepaint re-renders the whole window from scratch (fresh scene cache)
// so the result is the reference a partial frame must match.
func fullRepaint(w *Window) *paintengine2d.Image {
	w.laid = false
	w.fullInvalidate()
	w.frame()
	return w.surf.Buffer().Clone()
}

// overlapBox paints a solid rectangle so z-order is observable.
type overlapBox struct {
	widget.Base
	col paintengine2d.Color
}

func newOverlapBox(c paintengine2d.Color) *overlapBox {
	b := &overlapBox{col: c}
	b.Init(b)
	return b
}

func (b *overlapBox) Paint(ctx *paintengine2d.Context) {
	ctx.DrawRect(b.LocalBounds(), paintengine2d.Fill(b.col))
}

// overlapHost arranges two children that overlap, later child on top.
type overlapHost struct{ widget.Base }

func (h *overlapHost) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(200, 200))
}

func (h *overlapHost) Arrange(r paintengine2d.Rect) {
	h.SetBounds(r)
	ch := h.Children()
	ch[0].Arrange(paintengine2d.XYWH(0, 0, 120, 120))
	ch[1].Arrange(paintengine2d.XYWH(60, 60, 120, 120))
}

func newOverlapTree() (*overlapHost, *overlapBox, *overlapBox) {
	h := &overlapHost{}
	h.Init(h)
	lower := newOverlapBox(paintengine2d.RGB(220, 40, 40))
	upper := newOverlapBox(paintengine2d.RGB(40, 40, 220))
	h.Add(lower)
	h.Add(upper)
	return h, lower, upper
}

// A partial repaint must never let a lower sibling paint over the one
// stacked above it: the dirty box is repainted with the whole tree in
// z-order, not just the invalidated widget.
func TestPartialRepaintKeepsZOrder(t *testing.T) {
	for _, mode := range []string{"scene", "immediate"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "immediate" {
				t.Setenv(platform.EnvScene, "off")
			}
			a := New(Options{Look: style.DarkLook(), Headless: true})
			w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 200, Headless: true})
			if err != nil {
				t.Fatal(err)
			}
			h, lower, upper := newOverlapTree()
			w.SetContent(h)
			a.PumpOnce()
			want := upper.col.NRGBA()
			if got := w.surf.Buffer().NRGBAAt(90, 90); got != want {
				t.Fatalf("overlap pixel before %v, want the upper box %v", got, want)
			}
			// Dirty a corner of the lower box only.
			lower.InvalidateRect(paintengine2d.XYWH(0, 0, 12, 12))
			a.PumpOnce()
			if w.full {
				t.Fatal("a widget rect invalidate must stay partial")
			}
			if got := w.surf.Buffer().NRGBAAt(90, 90); got != want {
				t.Fatalf("partial repaint put the lower box on top: %v want %v", got, want)
			}
		})
	}
}

// The whole point of finding 7: a damaged frame must be pixel-identical to
// a full repaint, in both the retained and the immediate path.
func TestPartialFrameEqualsFullFrame(t *testing.T) {
	for _, mode := range []string{"scene", "immediate"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "immediate" {
				t.Setenv(platform.EnvScene, "off")
			}
			a := New(Options{Look: style.DarkLook(), Headless: true})
			w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 260, Headless: true})
			if err != nil {
				t.Fatal(err)
			}
			var buttons []*widgets.Button
			col := widgets.NewColumn()
			for i := 0; i < 6; i++ {
				b := widgets.NewButton(fmt.Sprintf("button %d", i), nil)
				buttons = append(buttons, b)
				col.Add(b)
			}
			list := widgets.NewListView(12, func(i int) string { return fmt.Sprintf("row %d", i) }, nil)
			w.SetContent(widgets.NewRow(col, list))
			a.PumpOnce()

			for i, b := range buttons {
				b.Invalidate()
				a.PumpOnce()
				if w.full {
					t.Fatalf("button %d invalidate went full-frame", i)
				}
				partial := w.surf.Buffer().Clone()
				full := fullRepaint(w)
				if n := diffPixels(t, partial, full); n != 0 {
					t.Fatalf("button %d: %d pixels differ between partial and full frame", i, n)
				}
			}

			list.Selected = 4
			list.Invalidate()
			a.PumpOnce()
			partial := w.surf.Buffer().Clone()
			if n := diffPixels(t, partial, fullRepaint(w)); n != 0 {
				t.Fatalf("list selection: %d pixels differ between partial and full frame", n)
			}
		})
	}
}

// A partial frame must present only the boxes it repainted.
func TestPartialFramePresentsOnlyDamage(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("OK", nil)
	w.SetContent(widgets.NewColumn(btn, widgets.NewLabel("static")))
	a.PumpOnce()
	if r := w.paintRects(); r != nil {
		t.Fatalf("clean window should have no damage, got %v", r)
	}
	btn.Invalidate()
	rects := w.paintRects()
	if len(rects) == 0 {
		t.Fatal("button invalidate produced no damage")
	}
	var area float32
	for _, r := range rects {
		area += r.Dx() * r.Dy()
	}
	if area > 400*300/3 {
		t.Fatalf("damage %v covers %v of 120000 px", rects, area)
	}
}

// UITK_PAINT_FULLFRAME is the field escape hatch.
func TestFullFrameEnvDisablesPartialRedraw(t *testing.T) {
	fullFrameOnce.once.Do(func() {})
	saved := fullFrameOnce.on
	fullFrameOnce.on = true
	defer func() { fullFrameOnce.on = saved }()
	_ = os.Getenv(EnvFullFrame)

	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("OK", nil)
	w.SetContent(btn)
	a.PumpOnce()
	btn.Invalidate()
	if r := w.paintRects(); r != nil {
		t.Fatalf("full-frame mode must present everything, got %v", r)
	}
}

// A translucent overlay must be composited exactly once per pixel. Damage
// boxes are coalesced so they do not overlap; if that ever stopped holding,
// a dimmer would darken twice where two dirty boxes met.
func TestPartialRepaintDoesNotDoubleBlend(t *testing.T) {
	for _, mode := range []string{"scene", "immediate"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "immediate" {
				t.Setenv(platform.EnvScene, "off")
			}
			a := New(Options{Look: style.DarkLook(), Headless: true})
			w, err := a.NewWindow(platform.WindowOptions{Width: 260, Height: 200, Headless: true})
			if err != nil {
				t.Fatal(err)
			}
			left := widgets.NewButton("left", nil)
			right := widgets.NewButton("right", nil)
			w.SetContent(widgets.NewColumn(widgets.NewRow(left, right), widgets.NewLabel("body")))
			w.SetOverlay(widgets.NewOverlay(widgets.NewLabel("dialog")))
			a.PumpOnce()

			// Two separate dirty boxes in one frame.
			left.Invalidate()
			right.Invalidate()
			a.PumpOnce()
			partial := w.surf.Buffer().Clone()
			if n := diffPixels(t, partial, fullRepaint(w)); n != 0 {
				t.Fatalf("%d pixels differ under a translucent overlay", n)
			}
		})
	}
}

// Damage boxes handed to the backend must never overlap.
func TestDamageRectsDoNotOverlap(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	var btns []*widgets.Button
	col := widgets.NewColumn()
	for i := 0; i < 5; i++ {
		b := widgets.NewButton("b", nil)
		btns = append(btns, b)
		col.Add(b)
	}
	w.SetContent(col)
	a.PumpOnce()
	for _, b := range btns {
		b.Invalidate()
	}
	rects := w.paintRects()
	for i := 0; i < len(rects); i++ {
		for j := i + 1; j < len(rects); j++ {
			if rects[i].Overlaps(rects[j]) {
				t.Fatalf("damage boxes overlap: %v and %v", rects[i], rects[j])
			}
		}
	}
}
