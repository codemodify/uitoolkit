package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// flatArt is a painter that covers the whole box in one colour, which is
// what a key's sprite does: any pixel that is not that colour afterwards was
// drawn over the art, and the focus ring is the only thing allowed to.
func flatArt(seen *int, st *style.ControlState) ButtonPainter {
	return func(ctx *paintengine2d.Context, b paintengine2d.Rect, s style.ControlState) bool {
		*seen++
		*st = s
		ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGB(0.2, 0.2, 0.2)))
		return true
	}
}

// sameImage reports whether two paints of the same box came out identical.
func sameImage(a, b *paintengine2d.Image) bool {
	if a == nil || b == nil || a.Width != b.Width || a.Height != b.Height {
		return false
	}
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			r1, g1, b1, a1 := a.PremulAt(x, y)
			r2, g2, b2, a2 := b.PremulAt(x, y)
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				return false
			}
		}
	}
	return true
}

func TestButtonPainterTakesOverTheLook(t *testing.T) {
	seen := 0
	var got style.ControlState
	b := NewButton("Play", nil)
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 90, 30))
	plain := immediatePaint(b, 90, 30)
	b.Painter = flatArt(&seen, &got)
	art := immediatePaint(b, 90, 30)
	if seen != 1 {
		t.Fatalf("the painter was consulted %d times, want once", seen)
	}
	if sameImage(plain, art) {
		t.Error("the painter drew but the button still looks like the look's face")
	}
	if got.Focused() {
		t.Error("an unfocused button was painted focused")
	}
}

// A painter that has nothing for this look answers false, and the look's
// own face is drawn — the fallback that makes a half-skinned app coherent.
func TestButtonPainterThatRefusesFallsBackToTheFace(t *testing.T) {
	b := NewButton("Play", nil)
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 90, 30))
	plain := immediatePaint(b, 90, 30)
	b.Painter = func(*paintengine2d.Context, paintengine2d.Rect, style.ControlState) bool { return false }
	if !sameImage(plain, immediatePaint(b, 90, 30)) {
		t.Error("a painter that took nothing over changed the button")
	}
}

// The ring is the one thing the art does not get to leave out.
func TestPaintedButtonKeepsItsFocusRing(t *testing.T) {
	seen := 0
	var got style.ControlState
	for _, c := range []struct {
		name  string
		paint func(p ButtonPainter, focus bool) *paintengine2d.Image
	}{
		{"Button", func(p ButtonPainter, focus bool) *paintengine2d.Image {
			b := NewButton("Play", nil)
			b.SetHost(&host{})
			b.Painter = p
			b.Arrange(paintengine2d.XYWH(0, 0, 90, 30))
			if focus {
				b.RequestFocus()
				b.MarkKeyboardFocus()
			}
			return immediatePaint(b, 90, 30)
		}},
		{"ToolButton", func(p ButtonPainter, focus bool) *paintengine2d.Image {
			b := NewToolButton("Play", style.IconNone, nil)
			b.SetHost(&host{})
			b.Painter = p
			b.Arrange(paintengine2d.XYWH(0, 0, 90, 30))
			if focus {
				b.RequestFocus()
				b.MarkKeyboardFocus()
			}
			return immediatePaint(b, 90, 30)
		}},
	} {
		rest := c.paint(flatArt(&seen, &got), false)
		focused := c.paint(flatArt(&seen, &got), true)
		if !got.Focused() {
			t.Errorf("%s: the painter was not told the button has the keyboard", c.name)
		}
		if sameImage(rest, focused) {
			t.Errorf("%s: a focused painted button looks exactly like a resting one — the ring is gone", c.name)
		}
	}
}

func TestToolButtonPainterTakesOverTheLook(t *testing.T) {
	seen := 0
	var got style.ControlState
	b := NewToolButton("Rec", style.IconNone, nil)
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 40, 30))
	plain := immediatePaint(b, 40, 30)
	b.Painter = flatArt(&seen, &got)
	if seen != 0 {
		t.Fatalf("the painter ran before the button was painted")
	}
	if sameImage(plain, immediatePaint(b, 40, 30)) {
		t.Error("the painter drew but the tool button still looks like the look's face")
	}
	if seen != 1 {
		t.Fatalf("the painter was consulted %d times, want once", seen)
	}
}

// leftHalf is the silhouette of art that only fills the left half of its box.
func leftHalf(size paintengine2d.Point) *style.Silhouette {
	return style.SilhouetteOfRects(style.SilhouetteRect{Rect: paintengine2d.XYWH(0, 0, size.X/2, size.Y)})
}

// A button painted as a picture takes the pointer on the picture, and a
// press beside it falls through to whatever is behind.
func TestPaintedButtonTakesThePointerOnItsArt(t *testing.T) {
	panel := NewColumn()
	b := NewButton("Play", nil)
	b.Shaper = func(_ style.LookAndFeel, size paintengine2d.Point) *style.Silhouette { return leftHalf(size) }
	panel.Add(b)
	panel.SetHost(&host{})
	panel.Arrange(paintengine2d.XYWH(0, 0, 80, 40))
	b.Arrange(paintengine2d.XYWH(0, 0, 80, 40))
	if got := widget.HitRoot(panel, paintengine2d.Pt(10, 20)); got != widget.Component(b) {
		t.Errorf("the middle of the art is not the button: %T", got)
	}
	if got := widget.HitRoot(panel, paintengine2d.Pt(70, 20)); got == widget.Component(b) {
		t.Error("a press beside the art was taken by the button")
	}
}

func TestPaintedToolButtonTakesThePointerOnItsArt(t *testing.T) {
	panel := NewColumn()
	b := NewToolButton("Play", style.IconNone, nil)
	b.Shaper = func(_ style.LookAndFeel, size paintengine2d.Point) *style.Silhouette { return leftHalf(size) }
	panel.Add(b)
	panel.SetHost(&host{})
	panel.Arrange(paintengine2d.XYWH(0, 0, 80, 40))
	b.Arrange(paintengine2d.XYWH(0, 0, 80, 40))
	if got := widget.HitRoot(panel, paintengine2d.Pt(10, 20)); got != widget.Component(b) {
		t.Errorf("the middle of the art is not the tool button: %T", got)
	}
	if got := widget.HitRoot(panel, paintengine2d.Pt(70, 20)); got == widget.Component(b) {
		t.Error("a press beside the art was taken by the tool button")
	}
}

// A painted button is still a button: it clicks, it takes the keyboard, and
// a screen reader can press it.
func TestPaintedButtonStillBehavesLikeOne(t *testing.T) {
	n := 0
	b := NewToolButton("Play", style.IconNone, func() { n++ })
	b.Painter = func(*paintengine2d.Context, paintengine2d.Rect, style.ControlState) bool { return true }
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 40, 30))
	b.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, 15)})
	b.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(20, 15)})
	if n != 1 {
		t.Fatalf("a painted button did not click: n=%d", n)
	}
	if b.Tooltip() != "" {
		t.Errorf("an empty tip became %q", b.Tooltip())
	}
	b.Tip = "Play"
	if b.Tooltip() != "Play" {
		t.Errorf("the tip of a painted button is %q", b.Tooltip())
	}
}

// ---- the content painter -------------------------------------------------------

// mark is a content painter: a square in the middle of the face, which is
// what a transport glyph is.
func mark(seen *int, st *style.ControlState) ButtonContentPainter {
	return func(ctx *paintengine2d.Context, b paintengine2d.Rect, s style.ControlState) {
		*seen++
		*st = s
		side := b.Dy() / 3
		cx, cy := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2
		ctx.DrawRect(paintengine2d.XYWH(cx-side/2, cy-side/2, side, side),
			paintengine2d.Fill(paintengine2d.RGB(1, 0, 1)))
	}
}

// The content painter draws on the look's face and leaves it standing: the
// mark is there and the corner of the button is the look's own pixel.
func TestButtonContentDrawsOnTheLooksFace(t *testing.T) {
	seen := 0
	var got style.ControlState
	b := NewButton("", nil)
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 90, 30))
	plain := immediatePaint(b, 90, 30)
	b.Content = mark(&seen, &got)
	marked := immediatePaint(b, 90, 30)
	if seen != 1 {
		t.Fatalf("the content painter was consulted %d times, want once", seen)
	}
	if sameImage(plain, marked) {
		t.Error("the content painter drew and the button did not change")
	}
	r1, g1, b1, a1 := plain.PremulAt(3, 3)
	r2, g2, b2, a2 := marked.PremulAt(3, 3)
	if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
		t.Error("the look's own face was redrawn under the mark")
	}
}

// The content painter is told the state the face was drawn in, so a mark
// that is a different ink when the key is hot moves with it.
func TestButtonContentIsToldTheState(t *testing.T) {
	seen := 0
	var got style.ControlState
	b := NewToolButton("", style.IconNone, nil)
	b.Content = mark(&seen, &got)
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 40, 30))
	b.MouseEnter()
	immediatePaint(b, 40, 30)
	if seen == 0 || !got.Hovered() {
		t.Errorf("a hovered key painted its mark in %v (%d paints)", got, seen)
	}
}

// A Painter replaces the face, the mark and all: a picture of a key has its
// glyph painted in it already, so the content painter is not asked.
func TestButtonPainterLeavesTheContentPainterOut(t *testing.T) {
	seen, art := 0, 0
	var got style.ControlState
	b := NewButton("", nil)
	b.Content = mark(&seen, &got)
	b.Painter = flatArt(&art, &got)
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 90, 30))
	immediatePaint(b, 90, 30)
	if art != 1 || seen != 0 {
		t.Errorf("art painted %d times, content %d: the art paints the mark itself", art, seen)
	}
}
