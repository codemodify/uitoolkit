package app

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
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

func TestFullPresentReusesRecordedGroups(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 160, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewColumn(
		widgets.NewLabel("one"),
		widgets.NewLabel("two"),
		widgets.NewButton("OK", nil),
	))
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Nodes < 3 {
		t.Fatalf("first scene %+v", w.Scene())
	}
	w.Invalidate(nil, paintengine2d.Rect{})
	if !w.laid {
		t.Fatal("full present must not RequestLayout")
	}
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Reused < 1 {
		t.Fatalf("full present should keep groups, reused=%v scene=%+v", w.Scene().Reused, w.Scene())
	}
}

func TestLayoutDropsSceneCache(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("relayout"))
	a.PumpOnce()
	w.RequestLayout()
	a.PumpOnce()
	if w.Scene() == nil {
		t.Fatal("layout should still record")
	}
	if w.Scene().Reused != 0 {
		t.Fatalf("layout must re-record, reused=%d", w.Scene().Reused)
	}
}

func TestHoverDoesNotRequestLayout(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewColumn(
		widgets.NewLabel("static pane"),
		widgets.NewButton("OK", nil),
	))
	a.PumpOnce()
	if !w.laid {
		t.Fatal("first frame should layout")
	}
	o := widget.DeviceOrigin(w.Content())
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+20, o.Y+10)})
	if !w.laid {
		t.Fatal("hover must not RequestLayout")
	}
	if w.full {
		t.Fatal("hover must not full-invalidate")
	}
}

func TestPumpOnceCoalescesEventBurst(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 220, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	list := widgets.NewListView(20, func(i int) string { return "row" }, nil)
	w.SetContent(list)
	a.PumpOnce()
	before := w.paints
	o := widget.DeviceOrigin(list)
	for i := 0; i < 8; i++ {
		w.Inject(platform.Event{
			Kind: platform.EventMouseMove,
			Pos:  paintengine2d.Pt(o.X+40, o.Y+12+float32(i)*28),
		})
	}
	a.PumpOnce()
	if w.paints != before+1 {
		t.Fatalf("event burst painted %d frames, want 1", w.paints-before)
	}
	if !w.dirty.Empty() {
		t.Fatal("coalesced frame should consume dirty")
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

func TestSceneScrollReusesChild(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 160, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	col := widgets.NewColumn()
	for i := 0; i < 24; i++ {
		col.Add(widgets.NewLabel("row content for scroll reuse"))
	}
	sv := widgets.NewScrollView(col)
	w.SetContent(sv)
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Nodes < 4 {
		t.Fatalf("first scene %+v", w.Scene())
	}
	sv.ScrollBy(48)
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Reused < 1 {
		t.Fatalf("scroll should reuse child nodes, reused=%v scene=%+v", w.Scene().Reused, w.Scene())
	}
	img := w.Capture()
	if countOpaque(img, 20) < 200 {
		t.Fatal("scroll scene should still paint")
	}
}

func TestListScrollReusesRows(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	list := widgets.NewListView(80, func(i int) string { return "row content for list reuse" }, nil)
	w.SetContent(list)
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Nodes < 4 {
		t.Fatalf("first list scene %+v", w.Scene())
	}
	list.OffsetY += 56
	list.Invalidate()
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Nodes < 4 {
		t.Fatalf("scrolled list scene %+v", w.Scene())
	}
	img := w.Capture()
	if countOpaque(img, 20) < 200 {
		t.Fatal("list scene should still paint past the first page")
	}
	list.Invalidate()
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Reused < 1 {
		t.Fatalf("same offset should reuse row nodes, reused=%v scene=%+v", w.Scene().Reused, w.Scene())
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
