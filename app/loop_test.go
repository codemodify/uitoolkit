package app

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestIdleNeedsNoPaint(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewButton("OK", nil))
	a.PumpOnce()
	if w.needsPaint() {
		t.Fatal("idle after first frame")
	}
	if a.waitTimeout(time.Now(), time.Time{}) != -1 {
		t.Fatal("no caret / tip / anim should block on display")
	}
}

func TestBlinkOnlyInvalidatesCaret(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("Go", nil)
	tf := widgets.NewTextField("hi", "", nil)
	w.SetContent(widgets.NewRow(btn, tf))
	a.PumpOnce()
	w.RequestFocus(btn)
	a.PumpOnce()
	if w.wantsBlink() {
		t.Fatal("button is not a caret")
	}
	w.toggleBlink()
	if !w.dirty.Empty() || w.full {
		t.Fatal("button blink must not dirty")
	}
	w.RequestFocus(tf)
	a.PumpOnce()
	if !w.wantsBlink() {
		t.Fatal("text field should blink")
	}
	w.toggleBlink()
	if w.dirty.Empty() && !w.full {
		t.Fatal("caret blink should dirty the field")
	}
}

func TestListHoverDoesNotFullInvalidate(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	list := widgets.NewListView(20, func(i int) string { return "row" }, nil)
	w.SetContent(list)
	a.PumpOnce()
	ww, hh := w.surf.Size()
	full := float32(ww * hh)
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(40, 40)})
	if w.full {
		t.Fatal("hover must not full-invalidate")
	}
	if w.dirty.Empty() {
		t.Fatal("hover should dirty the row")
	}
	var area float32
	for _, r := range w.dirty.Rects {
		area += r.Dx() * r.Dy()
	}
	if area > full/3 {
		t.Fatalf("hover dirty area %v of %v", area, full)
	}
}

func TestWaitTimeoutHonorsCaret(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 60, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	tf := widgets.NewTextField("x", "", nil)
	w.SetContent(tf)
	a.PumpOnce()
	w.RequestFocus(tf)
	now := time.Now()
	next := now.Add(200 * time.Millisecond)
	d := a.waitTimeout(now, next)
	if d < 150*time.Millisecond || d > 200*time.Millisecond {
		t.Fatalf("caret wait %v", d)
	}
}
