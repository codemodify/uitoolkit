package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// overflowBox paints far outside its arranged rect (tests pane clip).
type overflowBox struct {
	widget.Base
	col paintengine2d.Color
}

func newOverflowBox(c paintengine2d.Color) *overflowBox {
	b := &overflowBox{col: c}
	b.Init(b)
	return b
}

func (b *overflowBox) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(80, 80))
}

func (b *overflowBox) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *overflowBox) Paint(ctx *paintengine2d.Context) {
	ctx.DrawRect(paintengine2d.XYWH(-800, -80, 2400, 400), paintengine2d.Fill(b.col))
}

func TestSplitterArrangeAfterDragExclusive(t *testing.T) {
	left := NewLabel("Date Size columns live here")
	right := NewLabel("Subject: a very long preview header that must not paint over the list")
	split := NewSplitter(true, left, right)
	split.Ratio = 0.5
	split.SetHost(&host{})
	box := paintengine2d.XYWH(0, 0, 400, 200)
	split.Measure(layout.Tight(400, 200))
	split.Arrange(box)

	a0, b0 := split.PaneA(), split.PaneB()
	if a0.Empty() || b0.Empty() {
		t.Fatalf("empty panes A=%+v B=%+v", a0, b0)
	}
	if a0.Overlaps(b0) {
		t.Fatalf("initial overlap A=%+v B=%+v", a0, b0)
	}
	if left.Bounds() != a0 || right.Bounds() != b0 {
		t.Fatalf("child bounds left=%+v want A=%+v right=%+v want B=%+v", left.Bounds(), a0, right.Bounds(), b0)
	}

	div := split.divider()
	split.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(div.Min.X+1, 20), Button: platform.ButtonLeft})
	split.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(280, 20), Button: platform.ButtonLeft})
	split.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(280, 20)})

	a1, b1 := split.PaneA(), split.PaneB()
	if a1.Overlaps(b1) {
		t.Fatalf("after drag overlap A=%+v B=%+v", a1, b1)
	}
	if a1.Max.X > b1.Min.X+0.01 {
		t.Fatalf("A not exclusive of B: A.max=%v B.min=%v", a1.Max.X, b1.Min.X)
	}
	if left.Bounds() != a1 || right.Bounds() != b1 {
		t.Fatalf("re-arrange left=%+v A=%+v right=%+v B=%+v", left.Bounds(), a1, right.Bounds(), b1)
	}
	if a1.Dx() < a0.Dx() {
		t.Fatalf("expected first pane to grow, before=%v after=%v", a0.Dx(), a1.Dx())
	}
}

func TestSplitterClipsOverflowPaint(t *testing.T) {
	red := paintengine2d.RGB(0.9, 0.1, 0.1)
	blue := paintengine2d.RGB(0.1, 0.2, 0.9)
	split := NewSplitter(true, newOverflowBox(red), newOverflowBox(blue))
	split.Ratio = 0.45
	split.SetHost(&host{})
	split.SetLook(style.DarkLook())
	split.Arrange(paintengine2d.XYWH(0, 0, 300, 120))

	img := paintengine2d.NewImage(300, 120)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, 300, 120), paintengine2d.Fill(paintengine2d.RGB(0, 0, 0)))
	widget.PaintTree(split, ctx, nil)

	a, b := split.PaneA(), split.PaneB()
	ax := int((a.Min.X + a.Max.X) * 0.5)
	bx := int((b.Min.X + b.Max.X) * 0.5)
	y := 40
	ar, ag, ab, _ := img.PremulAt(ax, y)
	br, bg, bb, _ := img.PremulAt(bx, y)
	if ar < 80 || ag > 80 || ab > 80 {
		t.Fatalf("pane A should be red, got %d %d %d at %d,%d A=%+v", ar, ag, ab, ax, y, a)
	}
	if br > 80 || bb < 80 {
		t.Fatalf("pane B should be blue, got %d %d %d at %d,%d B=%+v", br, bg, bb, bx, y, b)
	}
	// Just inside A, near the sash: must not be the sibling blue.
	edge := int(a.Max.X) - 2
	if edge > 1 {
		er, eg, eb, _ := img.PremulAt(edge, y)
		if eb > er+20 && eb > 80 {
			t.Fatalf("B painted into A at x=%d: %d %d %d", edge, er, eg, eb)
		}
	}
}

func TestScrollClampWheelStopsAtEnd(t *testing.T) {
	tv := NewTableView([]TableColumn{{Title: "S"}}, 8, func(row, col int) string { return "row" }, nil)
	tv.RowHeight = 20
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 200, 80))
	mx := tv.MaxOffset()
	if mx <= 0 {
		t.Fatalf("expected overflow max=%v content=%v body=%v", mx, tv.contentH(), tv.bodyH())
	}
	for i := 0; i < 80; i++ {
		tv.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 40)})
	}
	if tv.OffsetY != mx {
		t.Fatalf("table past-end offset=%v max=%v", tv.OffsetY, mx)
	}
	_, thumb := tv.scrollTrack()
	if thumb.Empty() {
		t.Fatal("overflow table should show a thumb")
	}

	lv := NewListView(30, func(i int) string { return "x" }, nil)
	lv.RowHeight = 16
	lv.SetHost(&host{})
	lv.Arrange(paintengine2d.XYWH(0, 0, 120, 64))
	for i := 0; i < 80; i++ {
		lv.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 30)})
	}
	if lv.OffsetY != lv.MaxOffset() || lv.MaxOffset() <= 0 {
		t.Fatalf("list offset=%v max=%v", lv.OffsetY, lv.MaxOffset())
	}

	sv := NewScrollView(NewLabel("a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl"))
	sv.SetHost(&host{})
	col := NewColumn()
	for i := 0; i < 40; i++ {
		col.Add(NewLabel("row"))
	}
	sv.SetChild(col)
	sv.Arrange(paintengine2d.XYWH(0, 0, 160, 100))
	for i := 0; i < 80; i++ {
		sv.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 40)})
	}
	if sv.OffsetY != sv.MaxOffset() || sv.MaxOffset() <= 0 {
		t.Fatalf("scrollview offset=%v max=%v", sv.OffsetY, sv.MaxOffset())
	}
	_, th := sv.thumb()
	if th.Empty() {
		t.Fatal("scrollview thumb")
	}
}

func TestTextViewReadOnlyAndScrollbar(t *testing.T) {
	body := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12"
	tv := NewTextView(body, "")
	tv.MinRows = 2
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 180, 56))
	if !tv.ReadOnly {
		t.Fatal("TextView must be ReadOnly")
	}
	before := tv.Text
	if tv.TextInput('x') {
		t.Fatal("read-only must reject text input")
	}
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyA})
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if tv.Text != before {
		t.Fatalf("edited read-only %q", tv.Text)
	}
	if tv.MaxOffset() <= 0 {
		t.Fatalf("expected vertical overflow, max=%v h=%v", tv.MaxOffset(), tv.contentH())
	}
	_, thumb := tv.scrollTrackV()
	if thumb.Empty() {
		t.Fatal("overflow TextView should paint a thumb")
	}
	for i := 0; i < 40; i++ {
		tv.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 80)})
	}
	if tv.scrollY != tv.MaxOffset() {
		t.Fatalf("textview past-end y=%v max=%v", tv.scrollY, tv.MaxOffset())
	}

	edit := NewTextArea("hello", "", nil)
	edit.SetHost(&host{})
	edit.Arrange(paintengine2d.XYWH(0, 0, 180, 80))
	edit.SetSelection(5, 5)
	if !edit.TextInput('!') {
		t.Fatal("editable TextArea should accept input")
	}
	if edit.Text != "hello!" {
		t.Fatalf("compose %q", edit.Text)
	}
}
