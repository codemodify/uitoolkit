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

func bandInk(img *paintengine2d.Image, y0, y1, x0, x1 int) int {
	if img == nil || y1 <= y0 || x1 <= x0 {
		return 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x0 < 0 {
		x0 = 0
	}
	if y1 > img.Height {
		y1 = img.Height
	}
	if x1 > img.Width {
		x1 = img.Width
	}
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, a := img.PremulAt(x, y)
			if a > 8 && int(r)+int(g)+int(b) > 40 {
				n++
			}
		}
	}
	return n
}

func bandContrast(img *paintengine2d.Image, y0, y1, x0, x1 int) int {
	if img == nil || y1 <= y0 || x1 <= x0 {
		return 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x0 < 0 {
		x0 = 0
	}
	if y1 > img.Height {
		y1 = img.Height
	}
	if x1 > img.Width {
		x1 = img.Width
	}
	minS, maxS := 255*3, 0
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, a := img.PremulAt(x, y)
			if a < 8 {
				continue
			}
			s := int(r) + int(g) + int(b)
			if s < minS {
				minS = s
			}
			if s > maxS {
				maxS = s
			}
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return maxS - minS
}

func assertViewportFilled(t *testing.T, img *paintengine2d.Image, o paintengine2d.Point, view paintengine2d.Rect) {
	t.Helper()
	if img == nil {
		t.Fatal("buffer")
	}
	x0 := int(o.X + view.Min.X + 8)
	x1 := int(o.X + view.Max.X - 16)
	if x1 <= x0 {
		x1 = x0 + 8
	}
	yTop := int(o.Y + view.Min.Y + 2)
	yBot := int(o.Y + view.Max.Y - 2)
	ink := bandInk(img, yTop, yBot, x0, x1)
	if ink < 40 {
		t.Fatalf("viewport empty after scroll, ink=%d", ink)
	}
	// Sliding windows catch a vacated hole (Mail: white middle + orphan
	// bottom row) without depending on fitted row height. One flat
	// window can be padding; three in a row is a missing strip.
	const step = 16
	consec, maxConsec, painted := 0, 0, 0
	for y := yTop; y+8 < yBot; y += step {
		if bandContrast(img, y, y+step, x0, x1) < 25 {
			consec++
			if consec > maxConsec {
				maxConsec = consec
			}
			continue
		}
		painted++
		consec = 0
	}
	if painted < 3 {
		t.Fatalf("expected painted row bands, got %d", painted)
	}
	if maxConsec >= 3 {
		t.Fatalf("unpainted gap %d×%d px in viewport (scroll hole)", maxConsec, step)
	}
}

func TestChromeDamageOnly(t *testing.T) {
	var hover paintengine2d.Damage
	hover.Add(paintengine2d.XYWH(10, 40, 400, 28))
	if !chromeDamageOnly(&hover) {
		t.Fatal("row hover must stay on the dirty path")
	}
	var bar paintengine2d.Damage
	bar.Add(paintengine2d.XYWH(260, 8, 12, 200))
	if !chromeDamageOnly(&bar) {
		t.Fatal("scrollbar track is thin chrome")
	}
	var list paintengine2d.Damage
	list.Add(paintengine2d.XYWH(200, 80, 400, 360))
	if chromeDamageOnly(&list) {
		t.Fatal("list viewport must full-replay")
	}
	if chromeDamageOnly(nil) {
		t.Fatal("empty damage is not chrome")
	}
}

func TestListScrollFillsViewport(t *testing.T) {
	a, w := scrollHeadless(t, 280, 220)
	list := widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d unique text", i) }, nil)
	w.SetContent(list)
	a.PumpOnce()
	const dy = 40
	list.ScrollTo(dy)
	if list.OffsetY != dy {
		t.Fatalf("offset %v", list.OffsetY)
	}
	if !w.presentExtra.Empty() {
		t.Fatal("pixel scroll blit must stay off")
	}
	lb := list.Bounds()
	area := dirtyArea(w)
	if area < lb.Dx()*lb.Dy()*0.8 {
		t.Fatalf("scroll dirty %v must cover the list viewport %v", area, lb.Dx()*lb.Dy())
	}
	if chromeDamageOnly(&w.dirty) {
		t.Fatal("list scroll dirty must take the full-replay path")
	}
	a.PumpOnce()
	o := widget.DeviceOrigin(list)
	assertViewportFilled(t, w.surf.Buffer(), o, list.LocalBounds())
	off := list.OffsetY
	a.PumpOnce()
	if list.OffsetY != off {
		t.Fatalf("offset jumped %v → %v", off, list.OffsetY)
	}
	assertViewportFilled(t, w.surf.Buffer(), o, list.LocalBounds())
}

func TestTableScrollFillsBody(t *testing.T) {
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
	if !w.presentExtra.Empty() {
		t.Fatal("table pixel scroll blit must stay off")
	}
	body := tv.Bounds().Dx() * tv.BodyHeight()
	if dirtyArea(w) < body*0.7 {
		t.Fatalf("table scroll dirty %v must cover the body %v", dirtyArea(w), body)
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
		t.Fatalf("sticky header moved on scroll: %d px", diff)
	}
	bodyView := paintengine2d.XYWH(0, tv.HeaderHeight(), tv.LocalBounds().Dx(), tv.BodyHeight())
	assertViewportFilled(t, after, o, bodyView)
}

func TestCardScrollFillsViewport(t *testing.T) {
	a, w := scrollHeadless(t, 320, 280)
	cards := widgets.NewCardList(40, func(i int) widgets.CardContent {
		return widgets.CardContent{
			Title: fmt.Sprintf("from-%02d correspondent", i), Subtitle: "subject line", Meta: "now",
		}
	}, nil)
	w.SetContent(cards)
	a.PumpOnce()
	cards.ScrollTo(72)
	if cards.OffsetY != 72 {
		t.Fatalf("offset %v", cards.OffsetY)
	}
	if !w.presentExtra.Empty() {
		t.Fatal("card scroll must not blit")
	}
	b := cards.Bounds()
	if dirtyArea(w) < b.Dx()*b.Dy()*0.8 {
		t.Fatalf("card scroll dirty %v must cover the viewport %v", dirtyArea(w), b.Dx()*b.Dy())
	}
	a.PumpOnce()
	assertViewportFilled(t, w.surf.Buffer(), widget.DeviceOrigin(cards), cards.LocalBounds())
}

func TestTreeScrollDirtiesViewport(t *testing.T) {
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
	if !w.presentExtra.Empty() {
		t.Fatal("tree pixel scroll blit must stay off")
	}
	b := tree.Bounds()
	if dirtyArea(w) < b.Dx()*b.Dy()*0.8 {
		t.Fatalf("tree scroll dirty %v must cover %v", dirtyArea(w), b.Dx()*b.Dy())
	}
	a.PumpOnce()
	assertViewportFilled(t, w.surf.Buffer(), widget.DeviceOrigin(tree), tree.LocalBounds())
}

func TestTextAreaScrollDirtiesViewport(t *testing.T) {
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
	if !w.presentExtra.Empty() {
		t.Fatal("textarea pixel scroll blit must stay off")
	}
	box := ta.Bounds()
	if dirtyArea(w) < box.Dx()*box.Dy()*0.8 {
		t.Fatalf("textarea scroll dirty %v must cover %v", dirtyArea(w), box.Dx()*box.Dy())
	}
	a.PumpOnce()
	assertViewportFilled(t, w.surf.Buffer(), widget.DeviceOrigin(ta), ta.LocalBounds())
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

func TestListWheelScrollsFullViewport(t *testing.T) {
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
	if !w.presentExtra.Empty() {
		t.Fatal("wheel must not Context.Scroll")
	}
	lb := list.Bounds()
	if dirtyArea(w) < lb.Dx()*lb.Dy()*0.8 {
		t.Fatalf("wheel dirty %v must cover the list", dirtyArea(w))
	}
	a.PumpOnce()
	assertViewportFilled(t, w.surf.Buffer(), widget.DeviceOrigin(list), list.LocalBounds())
}

func TestScrollPixelsDisabled(t *testing.T) {
	_, w := scrollHeadless(t, 200, 160)
	if w.ScrollPixels(widgets.NewLabel("x"), paintengine2d.XYWH(0, 0, 80, 80), 0, 16) {
		t.Fatal("ScrollPixels must refuse the blit")
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
