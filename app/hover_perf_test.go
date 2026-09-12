package app

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func hoverChromeWindow(t testing.TB, w, h int) (*Application, *Window, *widgets.MenuBar, *widgets.ToolBar) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	win, err := a.NewWindow(platform.WindowOptions{Width: w, Height: h, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	items := make([]*widgets.MenuItem, 0, 16)
	for i := 0; i < 16; i++ {
		items = append(items, widgets.Item(fmt.Sprintf("Command %02d", i+1), nil))
	}
	mb := widgets.NewMenuBar(widgets.NewMenu("&File", items...))
	tb := widgets.NewToolBar(
		widgets.ToolText("Get", nil),
		widgets.ToolText("Write", nil),
		widgets.ToolDivider(),
		widgets.ToolText("Reply", nil),
		widgets.ToolText("Forward", nil),
	)
	list := widgets.NewListView(48, func(i int) string { return fmt.Sprintf("row %02d body", i) }, nil)
	tree := widgets.NewTreeView(widgets.NewTreeNode("Inbox"), widgets.NewTreeNode("Sent"), widgets.NewTreeNode("Filters"))
	win.SetContent(widgets.NewColumn(mb, tb, widgets.NewSplitter(true, tree, list)))
	a.PumpOnce()
	return a, win, mb, tb
}

func popupItemPos(pop *widgets.PopupMenu, i int) paintengine2d.Point {
	r := pop.ItemBounds(i)
	o := widget.DeviceOrigin(pop)
	return paintengine2d.Pt(o.X+(r.Min.X+r.Max.X)*0.5, o.Y+(r.Min.Y+r.Max.Y)*0.5)
}

func dirtyArea(w *Window) float32 {
	var area float32
	for _, r := range w.dirty.Rects {
		area += r.Dx() * r.Dy()
	}
	return area
}

func TestMenuHoverDoesNotFullInvalidate(t *testing.T) {
	a, w, mb, _ := hoverChromeWindow(t, 720, 480)
	mb.Open(0)
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		t.Fatal("popup")
	}
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, 0)})
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, 1)})
	if w.full {
		t.Fatal("menu hover must not full-invalidate")
	}
	if w.dirty.Empty() {
		t.Fatal("menu hover should dirty the old and new row")
	}
	ww, hh := w.surf.Size()
	full := float32(ww * hh)
	area := dirtyArea(w)
	if area > full/8 {
		t.Fatalf("menu hover dirty area %v of %v", area, full)
	}
	row := pop.ItemBounds(0)
	if area > row.Dx()*row.Dy()*5 {
		t.Fatalf("menu hover dirty %v larger than ~2 rows (row %v×%v)", area, row.Dx(), row.Dy())
	}
}

func TestToolBarHoverDoesNotFullInvalidate(t *testing.T) {
	a, w, _, tb := hoverChromeWindow(t, 720, 480)
	_ = a
	c0 := tb.ItemCenter(0)
	c1 := tb.ItemCenter(3)
	o := widget.DeviceOrigin(tb)
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+c0.X, o.Y+c0.Y)})
	if w.full {
		t.Fatal("toolbar hover must not full-invalidate")
	}
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+c1.X, o.Y+c1.Y)})
	if w.full {
		t.Fatal("toolbar hover must not full-invalidate")
	}
	ww, hh := w.surf.Size()
	if dirtyArea(w) > float32(ww*hh)/6 {
		t.Fatalf("toolbar hover dirty %v", dirtyArea(w))
	}
}

func TestComboPopupHoverDirtiesRows(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 280, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	cb := widgets.NewComboBox([]string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}, 0, nil)
	w.SetContent(widgets.NewPad(24, cb))
	a.PumpOnce()
	cb.Open()
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok {
		t.Fatal("combo popup")
	}
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, 1)})
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, 3)})
	if w.full {
		t.Fatal("combo hover must not full-invalidate")
	}
	ww, hh := w.surf.Size()
	if dirtyArea(w) > float32(ww*hh)/6 {
		t.Fatalf("combo hover dirty %v", dirtyArea(w))
	}
}

func BenchmarkMenuHover(b *testing.B) {
	a, w, mb, _ := hoverChromeWindow(b, 800, 560)
	mb.Open(0)
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		b.Fatal("popup")
	}
	n := len(pop.Items)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, i%n)})
		a.PumpOnce()
	}
}

// BenchmarkMenuHoverFullScene is the v0.13 path: Clear+DrawScene the
// whole window on every hover. Kept so the dirty-rect win stays visible.
func BenchmarkMenuHoverFullScene(b *testing.B) {
	a, w, mb, _ := hoverChromeWindow(b, 800, 560)
	mb.Open(0)
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		b.Fatal("popup")
	}
	n := len(pop.Items)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, i%n)})
		w.full = true
		a.PumpOnce()
	}
}
