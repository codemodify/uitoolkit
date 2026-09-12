package app

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/uitest"
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
	if pop.HighlightedIndex() != 1 {
		t.Fatalf("hover index %d want 1", pop.HighlightedIndex())
	}
	if w.full {
		t.Fatal("menu hover must not full-invalidate the window")
	}
	if w.dirty.Empty() {
		t.Fatal("menu hover should dirty the popup")
	}
	pb := pop.Bounds()
	if dirtyArea(w) < pb.Dx()*pb.Dy()*0.8 {
		t.Fatalf("hover dirty %v must cover the popup %v", dirtyArea(w), pb.Dx()*pb.Dy())
	}
}

func TestMenuHoverPaintFollowsPointer(t *testing.T) {
	a, w, mb, _ := hoverChromeWindow(t, 720, 480)
	mb.Open(0)
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		t.Fatal("popup")
	}
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, 0)})
	if pop.HighlightedIndex() != 0 {
		t.Fatalf("hover 0: %d", pop.HighlightedIndex())
	}
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: popupItemPos(pop, 1)})
	if pop.HighlightedIndex() != 1 {
		t.Fatalf("item0 → item1: %d", pop.HighlightedIndex())
	}
	a.PumpOnce()
	img := w.surf.Buffer()
	if img == nil {
		t.Fatal("buffer")
	}
	lk := pop.Look()
	o := widget.DeviceOrigin(pop)
	r0 := pop.ItemBounds(0).Translate(o)
	r1 := pop.ItemBounds(1).Translate(o)
	if err := uitest.CheckMenuHoverBordered(img, r1, lk); err != nil {
		t.Fatalf("item1 highlight after dirty present: %v", err)
	}
	p := style.ResolveMenuChrome(lk.Palette())
	ch := style.MenuChromeFor(lk)
	gx := int(r0.Min.X + ch.CheckCol()*0.45)
	gy := int((r0.Min.Y + r0.Max.Y) * 0.5)
	if uitest.ColorDist(img, gx, gy, p.MenuHover) <= uitest.ColorDist(img, gx, gy, p.MenuGutter) {
		t.Fatal("item0 kept MenuHover after moving to item1 (stale dirty row)")
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

func TestPopupOpenDoesNotFullInvalidate(t *testing.T) {
	a, w, mb, _ := hoverChromeWindow(t, 720, 480)
	mb.Open(0)
	if w.full {
		t.Fatal("opening a menu must not full-invalidate the window")
	}
	if w.dirty.Empty() {
		t.Fatal("open should dirty the popup")
	}
	ww, hh := w.surf.Size()
	if dirtyArea(w) > float32(ww*hh)*0.55 {
		t.Fatalf("popup open dirty %v of %dx%d", dirtyArea(w), ww, hh)
	}
	_ = a
}

func TestSplitterDragDoesNotRequestLayout(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 640, Height: 400, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	left := widgets.NewListView(40, func(i int) string { return fmt.Sprintf("L%d", i) }, nil)
	right := widgets.NewListView(40, func(i int) string { return fmt.Sprintf("R%d", i) }, nil)
	split := widgets.NewSplitter(true, left, right)
	split.Ratio = 0.4
	w.SetContent(split)
	a.PumpOnce()
	sashX := split.PaneA().Max.X + 1
	o := widget.DeviceOrigin(split)
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(o.X+sashX, o.Y+40), Button: platform.ButtonLeft})
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+280, o.Y+40), Button: platform.ButtonLeft})
	if w.full {
		t.Fatal("splitter drag must not full-invalidate")
	}
	if !w.laid {
		t.Fatal("splitter drag must not RequestLayout the window")
	}
	ww, hh := w.surf.Size()
	if dirtyArea(w) > float32(ww*hh)*1.05 {
		t.Fatalf("drag dirty %v", dirtyArea(w))
	}
}

func TestSplitterDragKeepsBakedLayers(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 640, Height: 400, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	split := widgets.NewSplitter(true,
		widgets.NewListView(40, func(i int) string { return fmt.Sprintf("L%d", i) }, nil),
		widgets.NewListView(40, func(i int) string { return fmt.Sprintf("R%d", i) }, nil),
	)
	split.Ratio = 0.4
	w.SetContent(split)
	a.PumpOnce()
	o := widget.DeviceOrigin(split)
	sashX := split.PaneA().Max.X + 1
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(o.X+sashX, o.Y+40), Button: platform.ButtonLeft})
	a.PumpOnce()
	if !split.HasBakedPanes() {
		t.Fatal("first drag frame should BakeGroup both panes")
	}
	a0, b0 := split.BakedPaneLayers()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+280, o.Y+40), Button: platform.ButtonLeft})
	if w.full {
		t.Fatal("splitter drag must not full-invalidate")
	}
	a.PumpOnce()
	if !split.HasBakedPanes() {
		t.Fatal("drag should keep baked layers")
	}
	a1, b1 := split.BakedPaneLayers()
	if a0 != a1 || b0 != b1 {
		t.Fatal("drag re-baked panes (should Xform only)")
	}
	_ = a
}

func TestHoverUsesDrawSceneDamage(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 220, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	list := widgets.NewListView(40, func(i int) string { return fmt.Sprintf("row %02d", i) }, nil)
	w.SetContent(list)
	a.PumpOnce()
	o := widget.DeviceOrigin(list)
	rh := list.RowHeight
	if rh < 8 {
		rh = 28
	}
	rowPos := func(i int) paintengine2d.Point {
		return paintengine2d.Pt(o.X+40, o.Y+rh*float32(i)+rh*0.5)
	}
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: rowPos(0)})
	a.PumpOnce()
	before := cloneSurface(w)
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: rowPos(1)})
	if w.full {
		t.Fatal("list hover must not full-invalidate")
	}
	if !chromeDamageOnly(&w.dirty) {
		t.Fatal("list row hover must stay on the dirty DrawSceneDamage path")
	}
	if w.scene == nil {
		t.Fatal("WantScene should keep a retained graph")
	}
	dirty := append([]paintengine2d.Rect(nil), w.dirty.Rects...)
	a.PumpOnce()
	after := cloneSurface(w)
	if before == nil || after == nil {
		t.Fatal("buffer")
	}
	var d paintengine2d.Damage
	d.Pad = 2
	for _, r := range dirty {
		d.Add(r.Inset(-2))
	}
	changed := 0
	outside := 0
	for y := 0; y < before.Height && y < after.Height; y++ {
		for x := 0; x < before.Width && x < after.Width; x++ {
			ar, ag, ab, aa := before.PremulAt(x, y)
			br, bg, bb, ba := after.PremulAt(x, y)
			if ar == br && ag == bg && ab == bb && aa == ba {
				continue
			}
			if d.Overlaps(paintengine2d.XYWH(float32(x), float32(y), 1, 1)) {
				changed++
				continue
			}
			outside++
		}
	}
	if changed == 0 {
		t.Fatal("hover should repaint the dirty rows")
	}
	if outside > 40 {
		t.Fatalf("DrawSceneDamage leaked %d px outside dirty", outside)
	}
}

func TestListScrollbarHoverDirtiesBar(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 280, Height: 220, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	list := widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d", i) }, nil)
	w.SetContent(list)
	a.PumpOnce()
	track, thumb := list.ScrollTrack()
	if thumb.Empty() {
		t.Fatal("need overflow thumb")
	}
	o := widget.DeviceOrigin(list)
	pos := paintengine2d.Pt(o.X+(track.Min.X+track.Max.X)*0.5, o.Y+(track.Min.Y+track.Max.Y)*0.5)
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: pos})
	if w.full {
		t.Fatal("scrollbar hover must not full-invalidate")
	}
	area := dirtyArea(w)
	if area <= 0 {
		t.Fatal("bar hover should dirty the track")
	}
	lb := list.Bounds()
	if area > lb.Dx()*lb.Dy()*0.45 {
		t.Fatalf("bar hover dirty %v of list %v", area, lb.Dx()*lb.Dy())
	}
	_ = a
}

func BenchmarkSplitterDrag(b *testing.B) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 800, Height: 560, Headless: true})
	if err != nil {
		b.Fatal(err)
	}
	split := widgets.NewSplitter(true,
		widgets.NewListView(48, func(i int) string { return fmt.Sprintf("L%d", i) }, nil),
		widgets.NewListView(48, func(i int) string { return fmt.Sprintf("R%d", i) }, nil),
	)
	w.SetContent(split)
	a.PumpOnce()
	o := widget.DeviceOrigin(split)
	sashX := split.PaneA().Max.X + 1
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(o.X+sashX, o.Y+80), Button: platform.ButtonLeft})
	a.PumpOnce()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := o.X + 180 + float32(i%80)
		w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(x, o.Y+80), Button: platform.ButtonLeft})
		a.PumpOnce()
	}
}

func BenchmarkListRowHover(b *testing.B) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 360, Headless: true})
	if err != nil {
		b.Fatal(err)
	}
	list := widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d body", i) }, nil)
	w.SetContent(list)
	a.PumpOnce()
	o := widget.DeviceOrigin(list)
	rh := list.Bounds().Dy() / 12
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		y := o.Y + 20 + float32(i%10)*rh
		w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+40, y)})
		a.PumpOnce()
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
