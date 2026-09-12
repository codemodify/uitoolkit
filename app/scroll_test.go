package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func scrollHeadless(t testing.TB, w, h int) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	win, err := a.NewWindow(platform.WindowOptions{Width: w, Height: h, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	return a, win
}

func cloneSurface(w *Window) *paintengine2d.Image {
	img := w.surf.Buffer()
	if img == nil {
		return nil
	}
	return img.Clone()
}

func pixelShiftMatch(before, after *paintengine2d.Image, x, y0, y1, dy int) int {
	if before == nil || after == nil {
		return 0
	}
	match := 0
	for y := y0; y < y1; y++ {
		sy := y + dy
		if y < 0 || sy < 0 || y >= after.Height || sy >= before.Height {
			continue
		}
		if x < 0 || x >= before.Width || x >= after.Width {
			continue
		}
		ar, ag, ab, aa := after.PremulAt(x, y)
		br, bg, bb, ba := before.PremulAt(x, sy)
		if ar == br && ag == bg && ab == bb && aa == ba {
			match++
		}
	}
	return match
}

func TestListScrollBlitsAndDirtiesStrip(t *testing.T) {
	a, w := scrollHeadless(t, 280, 220)
	list := widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d unique", i) }, nil)
	w.SetContent(list)
	a.PumpOnce()
	before := cloneSurface(w)
	o := widget.DeviceOrigin(list)
	const dy = 40
	list.ScrollTo(dy)
	if list.OffsetY != dy {
		t.Fatalf("offset %v", list.OffsetY)
	}
	if w.full {
		t.Fatal("list scroll must not full-invalidate")
	}
	if w.presentExtra.Empty() {
		t.Fatal("Scroll should present the moved view")
	}
	area := dirtyArea(w)
	lb := list.Bounds()
	if area <= 0 {
		t.Fatal("scroll should dirty the exposed strip")
	}
	if area > lb.Dx()*lb.Dy()*0.7 {
		t.Fatalf("scroll dirty %v of list %v (want strip, not full list)", area, lb.Dx()*lb.Dy())
	}
	a.PumpOnce()
	after := cloneSurface(w)
	x := int(o.X + 24)
	y0 := int(o.Y + 8)
	y1 := int(o.Y + lb.Dy() - dy - 8)
	if got := pixelShiftMatch(before, after, x, y0, y1, dy); got < (y1-y0)*3/4 {
		t.Fatalf("blit match %d/%d at x=%d", got, y1-y0, x)
	}
}

func TestTableScrollKeepsHeaderAndDirtiesBody(t *testing.T) {
	a, w := scrollHeadless(t, 360, 240)
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "HHHHHHHHHH", Width: 220}}, 60,
		func(row, col int) string { return fmt.Sprintf("====ROW %02d====", row) }, nil)
	w.SetContent(tv)
	a.PumpOnce()
	before := cloneSurface(w)
	const dy = 36
	tv.ScrollTo(dy)
	if tv.OffsetY != dy {
		t.Fatalf("offset %v", tv.OffsetY)
	}
	if w.full {
		t.Fatal("table scroll must not full-invalidate")
	}
	if w.presentExtra.Empty() {
		t.Fatal("table Scroll should present the body")
	}
	area := dirtyArea(w)
	body := tv.Bounds().Dy() - tv.HeaderHeight()
	if area > tv.Bounds().Dx()*body*0.85 {
		t.Fatalf("table scroll dirty %v of body %v", area, tv.Bounds().Dx()*body)
	}
	a.PumpOnce()
	after := cloneSurface(w)
	o := widget.DeviceOrigin(tv)
	hh := int(tv.HeaderHeight())
	diff := 0
	for y := 1; y < hh-1; y++ {
		for x := 8; x < 80; x++ {
			ar, ag, ab, _ := before.PremulAt(int(o.X)+x, int(o.Y)+y)
			br, bg, bb, _ := after.PremulAt(int(o.X)+x, int(o.Y)+y)
			if abs8(ar, br)+abs8(ag, bg)+abs8(ab, bb) > 48 {
				diff++
			}
		}
	}
	if diff > 30 {
		t.Fatalf("sticky header moved on Scroll: %d px", diff)
	}
}

func TestTreeScrollBlitsStrip(t *testing.T) {
	a, w := scrollHeadless(t, 260, 200)
	root := widgets.NewTreeNode("root")
	root.Expanded = true
	for i := 0; i < 50; i++ {
		root.Children = append(root.Children, widgets.NewTreeNode(fmt.Sprintf("node %02d", i)))
	}
	tree := widgets.NewTreeView(root)
	w.SetContent(tree)
	a.PumpOnce()
	const dy = 32
	tree.ScrollTo(dy)
	if tree.OffsetY != dy {
		t.Fatalf("offset %v", tree.OffsetY)
	}
	if w.full {
		t.Fatal("tree scroll must not full-invalidate")
	}
	if w.presentExtra.Empty() {
		t.Fatal("tree Scroll should present the moved view")
	}
	area := dirtyArea(w)
	b := tree.Bounds()
	if area > b.Dx()*b.Dy()*0.7 {
		t.Fatalf("tree scroll dirty %v of %v", area, b.Dx()*b.Dy())
	}
}

func TestTextAreaScrollBlitsStrip(t *testing.T) {
	a, w := scrollHeadless(t, 280, 180)
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "line %02d of the document\n", i)
	}
	ta := widgets.NewTextView(b.String(), "")
	w.SetContent(ta)
	a.PumpOnce()
	const dy = 28
	ta.ScrollTo(dy)
	if ta.OffsetY() != dy {
		t.Fatalf("offset %v", ta.OffsetY())
	}
	if w.full {
		t.Fatal("textarea scroll must not full-invalidate")
	}
	if w.presentExtra.Empty() {
		t.Fatal("textarea Scroll should present the moved view")
	}
	area := dirtyArea(w)
	box := ta.Bounds()
	if area > box.Dx()*box.Dy()*0.75 {
		t.Fatalf("textarea scroll dirty %v of %v", area, box.Dx()*box.Dy())
	}
}

func TestResizeSetsSyncSize(t *testing.T) {
	a, w := scrollHeadless(t, 240, 160)
	w.SetContent(widgets.NewLabel("resize"))
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventResize, Width: 300, Height: 200})
	if !w.needSyncSize {
		t.Fatal("resize should request Context.SyncSize")
	}
	a.PumpOnce()
	if w.needSyncSize {
		t.Fatal("frame should consume SyncSize")
	}
}

func TestListWheelScrollsViaScroll(t *testing.T) {
	a, w := scrollHeadless(t, 280, 220)
	list := widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d", i) }, nil)
	w.SetContent(list)
	a.PumpOnce()
	o := widget.DeviceOrigin(list)
	w.dispatch(platform.Event{
		Kind:   platform.EventScroll,
		Pos:    paintengine2d.Pt(o.X+40, o.Y+40),
		Scroll: paintengine2d.Pt(0, 40),
	})
	if list.OffsetY <= 0 {
		t.Fatal("wheel should move offset")
	}
	if w.full {
		t.Fatal("wheel must not full-invalidate when Scroll blits")
	}
	if w.presentExtra.Empty() {
		t.Fatal("wheel should Context.Scroll the view")
	}
}

func BenchmarkListScroll(b *testing.B) {
	a, w := scrollHeadless(b, 400, 360)
	list := widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d body", i) }, nil)
	w.SetContent(list)
	a.PumpOnce()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			list.ScrollTo(40)
		} else {
			list.ScrollTo(0)
		}
		a.PumpOnce()
	}
}

func abs8(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}
