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

func scrollFixture(t *testing.T, rows int) (*Application, *Window, *widgets.ScrollView, *widgets.Button) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 220, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	col := widgets.NewColumn()
	for i := 0; i < rows; i++ {
		col.Add(widgets.NewButton(fmt.Sprintf("row %d", i), nil))
	}
	sv := widgets.NewScrollView(col)
	header := widgets.NewButton("HEADER", nil)
	w.SetContent(widgets.NewColumn(header, sv))
	a.PumpOnce()
	return a, w, sv, header
}

// Scrolling a retained scene layer must produce exactly the pixels a full
// re-record produces: no content bleeding over the widget above the
// viewport, and no blank band where rows scrolled in from off-screen.
func TestRetainedScrollMatchesFullRecord(t *testing.T) {
	a, w, sv, _ := scrollFixture(t, 40)
	for _, dy := range []float32{12, 48, 60, 97, 200} {
		sv.ScrollBy(dy)
		a.PumpOnce()
		got := w.surf.Buffer().Clone()
		want := fullRepaint(w)
		if n := diffPixels(t, got, want); n != 0 {
			t.Fatalf("scroll to %v: %d pixels differ from a full record", sv.OffsetY, n)
		}
	}
}

// The retained path must actually be taken, otherwise the test above would
// pass trivially.
func TestRetainedScrollReusesContent(t *testing.T) {
	a, w, sv, _ := scrollFixture(t, 40)
	sv.ScrollBy(48)
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Reused < 1 {
		t.Fatalf("scroll should reuse recorded content, scene=%+v", w.Scene())
	}
}

// A row invalidated in the same burst as a scroll must not leave the other
// rows attached at their old positions.
func TestScrollPlusRowInvalidateMatchesFullRecord(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 220, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	col := widgets.NewColumn()
	var rows []*widgets.Button
	for i := 0; i < 40; i++ {
		b := widgets.NewButton(fmt.Sprintf("row %d", i), nil)
		rows = append(rows, b)
		col.Add(b)
	}
	sv := widgets.NewScrollView(col)
	w.SetContent(widgets.NewColumn(widgets.NewButton("HEADER", nil), sv))
	a.PumpOnce()

	sv.ScrollBy(60)
	rows[2].Invalidate()
	a.PumpOnce()
	got := w.surf.Buffer().Clone()
	if n := diffPixels(t, got, fullRepaint(w)); n != 0 {
		t.Fatalf("scroll + row invalidate: %d pixels differ from a full record", n)
	}
}

// movingBox is a widget that can be re-arranged without a relayout, the
// way a scroll offset moves content.
type movingBox struct {
	widget.Base
	col paintengine2d.Color
}

func newMovingBox(c paintengine2d.Color) *movingBox {
	b := &movingBox{col: c}
	b.Init(b)
	return b
}

func (b *movingBox) Paint(ctx *paintengine2d.Context) {
	ctx.DrawRect(b.LocalBounds(), paintengine2d.Fill(b.col))
}

type mover struct{ widget.Base }

func (m *mover) Arrange(r paintengine2d.Rect) { m.SetBounds(r) }

// A cached group belongs to the position it was recorded at. Moving the
// widget without a relayout must not replay it at the old spot.
func TestCachedGroupFollowsWidgetMove(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	host := &mover{}
	host.Init(host)
	box := newMovingBox(paintengine2d.RGB(30, 200, 90))
	host.Add(box)
	w.SetContent(host)
	host.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	box.Arrange(paintengine2d.XYWH(10, 10, 60, 60))
	a.PumpOnce()
	want := box.col.NRGBA()
	if got := w.surf.Buffer().NRGBAAt(20, 20); got != want {
		t.Fatalf("box not painted at its first position: %v", got)
	}

	// Move it without a relayout — the way a scroll offset moves content.
	// Only the parent is invalidated, so the box itself stays clean and
	// the recording reuses its cached group.
	box.Arrange(paintengine2d.XYWH(100, 100, 60, 60))
	host.Invalidate()
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Reused < 1 {
		t.Fatalf("a clean moved child should be reused, scene=%+v", w.Scene())
	}
	if got := w.surf.Buffer().NRGBAAt(120, 120); got != want {
		t.Fatalf("moved box missing at its new position: %v", got)
	}
	if got := w.surf.Buffer().NRGBAAt(20, 20); got == want {
		t.Fatal("cached group replayed at the widget's old position")
	}
	if n := diffPixels(t, w.surf.Buffer().Clone(), fullRepaint(w)); n != 0 {
		t.Fatalf("moved-widget frame differs from a full record in %d pixels", n)
	}
}

// Hidden and detached widgets must not keep their recorded groups (and the
// images in them) alive, nor keep their ancestors permanently dirty.
func TestSceneCacheDropsUnvisitedGroups(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	col := widgets.NewColumn()
	for i := 0; i < 6; i++ {
		col.Add(widgets.NewButton(fmt.Sprintf("b%d", i), nil))
	}
	w.SetContent(col)
	a.PumpOnce()
	before := w.layers.Len()
	if before < 6 {
		t.Fatalf("expected a group per widget, got %d", before)
	}
	col.ClearChildren()
	w.RequestLayout()
	a.PumpOnce()
	a.PumpOnce()
	if after := w.layers.Len(); after >= before {
		t.Fatalf("removed subtree kept %d groups (was %d)", after, before)
	}
}

// An invisible child's stale dirty flag must not force its ancestors to
// re-record forever.
func TestInvisibleChildDoesNotDirtyAncestors(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 160, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	hidden := widgets.NewButton("hidden", nil)
	col := widgets.NewColumn(widgets.NewLabel("stays"), hidden)
	w.SetContent(col)
	a.PumpOnce()
	hidden.SetVisible(false)
	w.RequestLayout()
	a.PumpOnce()
	w.Invalidate(nil, paintengine2d.Rect{})
	a.PumpOnce()
	if w.Scene() == nil || w.Scene().Reused < 1 {
		t.Fatalf("a hidden child should not keep ancestors dirty, scene=%+v", w.Scene())
	}
}

// rowReset is a widget that can drop its retained row recordings.
type rowReset interface{ Invalidate() }

// fullRowRepaint re-renders with every row recording thrown away, so the
// result is an independent reference for a scrolled frame.
func fullRowRepaint(w *Window, v rowReset) *paintengine2d.Image {
	v.Invalidate()
	w.layers.Reset()
	return fullRepaint(w)
}

// Virtualized rows are recorded in row-local space under a band group whose
// clip is the viewport. Scrolling must therefore keep producing exactly the
// pixels a from-scratch recording produces — rows must not escape the
// viewport (or a table's sticky header), and none may be missing.
func TestScrolledListMatchesFullRecord(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	list := widgets.NewListView(200, func(i int) string { return fmt.Sprintf("row %03d body", i) }, nil)
	w.SetContent(widgets.NewColumn(widgets.NewButton("HEADER", nil), list))
	a.PumpOnce()
	for _, off := range []float32{7, 28, 29, 140, 999} {
		list.OffsetY = off
		list.Base.Invalidate()
		a.PumpOnce()
		got := w.surf.Buffer().Clone()
		if n := diffPixels(t, got, fullRowRepaint(w, list)); n != 0 {
			t.Fatalf("list offset %v: %d pixels differ from a full record", off, n)
		}
	}
}

func TestScrolledTableMatchesFullRecord(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 420, Height: 260, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	cols := []widgets.TableColumn{{Title: "From", Width: 140}, {Title: "Subject", Width: 180}, {Title: "Date", Width: 90}}
	table := widgets.NewTableView(cols, 200, func(r, c int) string {
		return fmt.Sprintf("c%d/%d", r, c)
	}, nil)
	w.SetContent(table)
	a.PumpOnce()
	for _, off := range []float32{9, 30, 31, 210, 1234} {
		table.OffsetY = off
		table.Base.Invalidate()
		a.PumpOnce()
		got := w.surf.Buffer().Clone()
		if n := diffPixels(t, got, fullRowRepaint(w, table)); n != 0 {
			t.Fatalf("table offset %v: %d pixels differ from a full record", off, n)
		}
	}
}

// Rows must stay under the sticky header while scrolling.
func TestScrolledTableKeepsStickyHeader(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 420, Height: 260, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	cols := []widgets.TableColumn{{Title: "From", Width: 140}, {Title: "Subject", Width: 180}}
	table := widgets.NewTableView(cols, 200, func(r, c int) string { return fmt.Sprintf("c%d/%d", r, c) }, nil)
	w.SetContent(table)
	a.PumpOnce()
	header := make([]uint8, 0, 420*4)
	snap := func() []uint8 {
		out := header[:0]
		for y := 0; y < 8; y++ {
			for x := 0; x < 420; x++ {
				c := w.surf.Buffer().NRGBAAt(x, y)
				out = append(out, c.R, c.G, c.B, c.A)
			}
		}
		return out
	}
	want := append([]uint8(nil), snap()...)
	for _, off := range []float32{13, 60, 400} {
		table.OffsetY = off
		table.Base.Invalidate()
		a.PumpOnce()
		if got := snap(); string(got) != string(want) {
			t.Fatalf("scrolling to %v changed the sticky header band", off)
		}
	}
}
