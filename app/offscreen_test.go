package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestOffscreenGalleryPaints(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("OK", nil)
	btn.Primary = true
	w.SetContent(widgets.NewPanel("Test", widgets.NewLabel("Hello"), btn))
	img := w.Capture()
	if img == nil || img.Width != 320 {
		t.Fatalf("image %+v", img)
	}
	if countOpaque(img, 20) < 200 {
		t.Fatalf("expected painted pixels, n=%d", countOpaque(img, 20))
	}
}

func TestTabFocusCycles(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	b1 := widgets.NewButton("One", nil)
	b2 := widgets.NewButton("Two", nil)
	w.SetContent(widgets.NewRow(b1, b2))
	a.PumpOnce()
	w.RequestFocus(b1)
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
	if w.Focus() != b2 {
		t.Fatalf("tab to two, got %v", name(w.Focus()))
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
	if w.Focus() != b1 {
		t.Fatalf("wrap to one, got %v", name(w.Focus()))
	}
}

func TestHitRoutesToButton(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 160, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	b := widgets.NewButton("Hit", func() { n++ })
	w.SetContent(widgets.NewPad(20, b))
	a.PumpOnce()
	c := widget.HitRoot(w.Content(), paintengine2d.Pt(40, 40))
	if c == nil {
		t.Fatal("expected hit")
	}
	bb := b.Bounds()
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(20+bb.Min.X+8, 20+bb.Min.Y+8), Button: platform.ButtonLeft})
	w.dispatch(platform.Event{Kind: platform.EventMouseUp, Pos: paintengine2d.Pt(20+bb.Min.X+8, 20+bb.Min.Y+8), Button: platform.ButtonLeft})
	if n != 1 {
		t.Fatalf("clicks %d hit=%T bounds=%+v", n, c, bb)
	}
}

func TestScrollWheelBubblesFromChild(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	col := widgets.NewColumn()
	for i := 0; i < 40; i++ {
		col.Add(widgets.NewLabel("Row content for scroll"))
	}
	sv := widgets.NewScrollView(col)
	w.SetContent(sv)
	a.PumpOnce()
	if sv.OffsetY != 0 {
		t.Fatalf("start offset %v", sv.OffsetY)
	}
	w.Inject(platform.Event{
		Kind:   platform.EventScroll,
		Pos:    paintengine2d.Pt(80, 80),
		Scroll: paintengine2d.Pt(0, 140),
	})
	a.PumpOnce()
	if sv.OffsetY < 40 {
		t.Fatalf("wheel should bubble to ScrollView, offset=%v", sv.OffsetY)
	}
}

func name(c widget.Component) string {
	if c == nil {
		return "<nil>"
	}
	return c.Name()
}

func countOpaque(img *paintengine2d.Image, minA uint8) int {
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			_, _, _, a := img.PremulAt(x, y)
			if a >= minA {
				n++
			}
		}
	}
	return n
}
