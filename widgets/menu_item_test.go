package widgets_test

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestCheckItemIsCheckable(t *testing.T) {
	it := widgets.CheckItem("Wrap", true, nil)
	if !it.Checkable || !it.Checked || it.RadioGroup != "" {
		t.Fatalf("%+v", *it)
	}
}

func TestMenuItemCheckableTogglesOnActivate(t *testing.T) {
	n := 0
	it := widgets.CheckItem("Wrap", false, nil)
	pop := widgets.NewPopupMenu(it)
	pop.OnPick = func(*widgets.MenuItem) { n++ }
	s := uitest.Mount(pop, paintengine2d.XYWH(0, 0, 200, 80))
	if !pop.KeyPress(widget.KeyEvent{Key: platform.KeyReturn}) {
		t.Fatal("return")
	}
	if !it.Checked || n != 1 {
		t.Fatalf("first toggle checked=%v n=%d", it.Checked, n)
	}
	if !pop.KeyPress(widget.KeyEvent{Key: platform.KeySpace}) {
		t.Fatal("space")
	}
	if it.Checked || n != 2 {
		t.Fatalf("second toggle checked=%v n=%d", it.Checked, n)
	}
	_ = s
}

func TestMenuItemRadioGroupExclusive(t *testing.T) {
	a := widgets.RadioItem("Dark", "palette", true, nil)
	b := widgets.RadioItem("Light", "palette", false, nil)
	other := widgets.CheckItem("Toolbar", true, nil)
	pop := widgets.NewPopupMenu(a, b, other)
	pop.OnPick = func(*widgets.MenuItem) {}
	_ = uitest.Mount(pop, paintengine2d.XYWH(0, 0, 200, 120))
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if a.Checked || !b.Checked || !other.Checked {
		t.Fatalf("a=%v b=%v other=%v", a.Checked, b.Checked, other.Checked)
	}
	// Clicking the selected radio keeps it on.
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if a.Checked || !b.Checked {
		t.Fatalf("reselect a=%v b=%v", a.Checked, b.Checked)
	}
}

func TestMenuItemDisabledDoesNotToggle(t *testing.T) {
	it := widgets.CheckItem("Locked", false, nil)
	it.Disabled = true
	pop := widgets.NewPopupMenu(it)
	pop.OnPick = func(*widgets.MenuItem) {}
	_ = uitest.Mount(pop, paintengine2d.XYWH(0, 0, 180, 60))
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if it.Checked {
		t.Fatal("disabled toggled")
	}
}

func TestMenuItemGutterFitsIcon(t *testing.T) {
	pop := widgets.NewPopupMenu(widgets.ItemIcon(style.IconSearch, "Find…", nil))
	s := uitest.Mount(pop, paintengine2d.XYWH(0, 0, 1, 1))
	sz := pop.Measure(layout.Unbounded())
	s.Relayout(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	ch := style.MenuChromeFor(pop.Look())
	need := style.IconSizePixels(style.LookIconSize(pop.Look()))
	if ch.CheckCol()+0.5 < need {
		t.Fatalf("gutter %v < icon %v", ch.CheckCol(), need)
	}
	row := pop.ItemBounds(0)
	if row.Dx()+0.5 < ch.CheckCol() {
		t.Fatalf("row %+v narrower than gutter %v", row, ch.CheckCol())
	}
}

func TestMenuItemCheckedPaintsGutter(t *testing.T) {
	on := widgets.CheckItem("On", true, nil)
	off := widgets.CheckItem("Off", false, nil)
	pop := widgets.NewPopupMenu(on, off)
	s := uitest.Mount(pop, paintengine2d.XYWH(0, 0, 220, 90))
	img := s.Paint()
	lk := pop.Look()
	ch := style.MenuChromeFor(lk)
	r0 := pop.ItemBounds(0).Translate(widget.DeviceOrigin(pop))
	r1 := pop.ItemBounds(1).Translate(widget.DeviceOrigin(pop))
	g0 := paintengine2d.XYWH(r0.Min.X, r0.Min.Y, ch.CheckCol(), r0.Dy()).Inset(2)
	g1 := paintengine2d.XYWH(r1.Min.X, r1.Min.Y, ch.CheckCol(), r1.Dy()).Inset(2)
	empty := style.ResolveMenuChrome(lk.Palette()).MenuGutter
	onInk := uitest.CountNonColor(img, g0, empty, 28)
	offInk := uitest.CountNonColor(img, g1, empty, 28)
	if onInk < 6 {
		t.Fatalf("checked gutter has no mark (ink=%d)", onInk)
	}
	if offInk >= onInk {
		t.Fatalf("unchecked gutter ink %d >= checked %d", offInk, onInk)
	}
}

func TestMenuItemIconPaintsGutter(t *testing.T) {
	plain := widgets.Item("Plain", nil)
	ic := widgets.ItemIcon(style.IconSearch, "Find", nil)
	pop := widgets.NewPopupMenu(plain, ic)
	s := uitest.Mount(pop, paintengine2d.XYWH(0, 0, 220, 90))
	img := s.Paint()
	ch := style.MenuChromeFor(pop.Look())
	r := pop.ItemBounds(1).Translate(widget.DeviceOrigin(pop))
	g := paintengine2d.XYWH(r.Min.X, r.Min.Y, ch.CheckCol(), r.Dy()).Inset(2)
	empty := style.ResolveMenuChrome(pop.Look().Palette()).MenuGutter
	if uitest.CountNonColor(img, g, empty, 28) < 6 {
		t.Fatal("icon gutter empty")
	}
}

func TestPopupMenuHoverFollowsPointer(t *testing.T) {
	pop := widgets.NewPopupMenu(
		widgets.Item("Keyboard", nil),
		widgets.Item("About Mail", nil),
		widgets.Item("Third", nil),
	)
	s := uitest.Mount(pop, paintengine2d.XYWH(0, 0, 240, 160))
	mid := func(i int) paintengine2d.Point {
		r := pop.ItemBounds(i)
		return paintengine2d.Pt((r.Min.X+r.Max.X)*0.5, (r.Min.Y+r.Max.Y)*0.5)
	}
	for i := 0; i < 3; i++ {
		r := pop.ItemBounds(i)
		if pop.HitTest(paintengine2d.Pt(r.Min.X+4, (r.Min.Y+r.Max.Y)*0.5)) != pop {
			t.Fatalf("hit-test missed row %d bounds %+v", i, r)
		}
		pop.MouseMove(widget.MouseEvent{Pos: mid(i)})
		if pop.HighlightedIndex() != i {
			t.Fatalf("hover index %d want %d", pop.HighlightedIndex(), i)
		}
	}
	pop.MouseMove(widget.MouseEvent{Pos: mid(0)})
	if pop.HighlightedIndex() != 0 {
		t.Fatalf("back to item0: %d", pop.HighlightedIndex())
	}
	n := s.Host.DamageCount()
	pop.MouseMove(widget.MouseEvent{Pos: mid(1)})
	if pop.HighlightedIndex() != 1 {
		t.Fatalf("item0 → item1 hover %d", pop.HighlightedIndex())
	}
	if s.Host.DamageCount() <= n {
		t.Fatal("hover change must invalidate the popup")
	}
	img := s.Paint()
	lk := pop.Look()
	r0 := pop.ItemBounds(0).Translate(widget.DeviceOrigin(pop))
	r1 := pop.ItemBounds(1).Translate(widget.DeviceOrigin(pop))
	if err := uitest.CheckMenuHoverBordered(img, r1, lk); err != nil {
		t.Fatalf("item1 should be highlighted: %v", err)
	}
	p := style.ResolveMenuChrome(lk.Palette())
	ch := style.MenuChromeFor(lk)
	gx := int(r0.Min.X + ch.CheckCol()*0.45)
	gy := int((r0.Min.Y + r0.Max.Y) * 0.5)
	if uitest.ColorDist(img, gx, gy, p.MenuHover) <= uitest.ColorDist(img, gx, gy, p.MenuGutter) {
		t.Fatal("item0 kept MenuHover after moving to item1")
	}
	pop.MouseExit()
	if pop.HighlightedIndex() >= 0 {
		t.Fatalf("leave left highlight %d", pop.HighlightedIndex())
	}
	same := s.Host.DamageCount()
	pop.MouseMove(widget.MouseEvent{Pos: mid(1)})
	pop.MouseMove(widget.MouseEvent{Pos: mid(1)})
	if s.Host.DamageCount() != same+1 {
		t.Fatalf("same-row move must not re-invalidate, extra=%d", s.Host.DamageCount()-(same+1))
	}
}

func BenchmarkPopupMenuHover(b *testing.B) {
	items := make([]*widgets.MenuItem, 20)
	for i := range items {
		items[i] = widgets.Item("Command", nil)
	}
	pop := widgets.NewPopupMenu(items...)
	s := uitest.Mount(pop, paintengine2d.XYWH(0, 0, 260, 640))
	_ = s
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := pop.ItemBounds(i % 20)
		pop.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt((r.Min.X+r.Max.X)*0.5, (r.Min.Y+r.Max.Y)*0.5)})
	}
}
