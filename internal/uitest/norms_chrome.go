package uitest

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func checkScrollOverflow() error {
	col := widgets.NewColumn()
	for i := 0; i < 40; i++ {
		col.Add(widgets.NewLabel(fmt.Sprintf("row-%02d ----------------", i)))
	}
	sv := widgets.NewScrollView(col)
	s := Mount(sv, paintengine2d.XYWH(0, 0, 180, 90))
	if sv.MaxOffset() <= 0 {
		return fmt.Errorf("expected overflow max=%v content=%v", sv.MaxOffset(), sv.ContentHeight())
	}
	track, thumb := sv.ScrollTrack()
	if thumb.Empty() {
		return fmt.Errorf("overflow thumb missing track=%+v", track)
	}
	if err := CheckThumbInTrack(track, thumb); err != nil {
		return err
	}
	frac := thumb.Dy() / track.Dy()
	if frac > 0.95 {
		return fmt.Errorf("thumb fills track (frac=%v) — not sized to content", frac)
	}
	s.Wheel(paintengine2d.Pt(40, 40), 1e6)
	if err := CheckClamped(sv.OffsetY, sv.MaxOffset()); err != nil {
		return err
	}
	if sv.OffsetY != sv.MaxOffset() {
		return fmt.Errorf("wheel past end offset=%v max=%v", sv.OffsetY, sv.MaxOffset())
	}
	sv.ScrollTo(-80)
	if sv.OffsetY != 0 {
		return fmt.Errorf("neg offset %v", sv.OffsetY)
	}

	fit := widgets.NewScrollView(widgets.NewLabel("fits"))
	_ = Mount(fit, paintengine2d.XYWH(0, 0, 180, 90))
	_, fitThumb := fit.ScrollTrack()
	if !fitThumb.Empty() {
		return fmt.Errorf("fitting content still shows thumb %+v", fitThumb)
	}
	return nil
}

func checkListScrollClip() error {
	lv := widgets.NewListView(30, func(i int) string { return fmt.Sprintf("====ROW %02d====", i) }, nil)
	lv.RowHeight = 20
	lv.Selected = 0
	s := Mount(lv, paintengine2d.XYWH(0, 0, 200, 80))
	if err := CheckInkInsideBounds(s, lv, 3); err != nil {
		return fmt.Errorf("idle: %w", err)
	}
	if lv.MaxOffset() <= 0 {
		return fmt.Errorf("expected overflow")
	}
	lv.ScrollTo(lv.LocalBounds().Dy() + 8)
	if lv.OffsetY <= 0 {
		return fmt.Errorf("expected scroll")
	}
	lo, hi := lv.VisibleRange()
	if lo < 1 || hi <= lo {
		return fmt.Errorf("window lo=%d hi=%d", lo, hi)
	}
	img := s.Paint()
	// Selected row 0 is off-screen — no accent wash at the top of the viewport.
	ar, ag, ab := RGB8(lv.Look().Palette().Accent)
	top := paintengine2d.XYWH(4, 1, 90, 5)
	if n := countNearRGB(img, top, ar, ag, ab, 36); n > 12 {
		return fmt.Errorf("selected row bled into top after scroll (near=%d offset=%v)", n, lv.OffsetY)
	}
	return CheckInkInsideBounds(s, lv, 3)
}

func checkTableHeaderFlush() error {
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "HHHHHHHHHH"}}, 16,
		func(row, col int) string { return fmt.Sprintf("====ROW %02d====", row) }, nil)
	tv.RowHeight = 22
	tv.Selected = 0
	s := Mount(tv, paintengine2d.XYWH(0, 0, 260, 140))
	row0 := tv.RowBounds(0)
	hh := tv.HeaderHeight()
	if row0.Empty() {
		return fmt.Errorf("row0 empty")
	}
	if d := row0.Min.Y - hh; d < -1 || d > 2 {
		return fmt.Errorf("header gap: first row y=%v headerH=%v delta=%v", row0.Min.Y, hh, d)
	}
	img0 := s.Paint()
	tv.ScrollTo(hh + tv.RowBounds(0).Dy())
	if tv.OffsetY <= 0 {
		return fmt.Errorf("expected scroll")
	}
	img1 := s.Paint()
	diff := 0
	for y := 1; y < int(hh)-1 && y < img0.Height && y < img1.Height; y++ {
		for x := 4; x < 80 && x < img0.Width && x < img1.Width; x++ {
			ar, ag, ab, _ := img0.PremulAt(x, y)
			br, bg, bb, _ := img1.PremulAt(x, y)
			if abs(int(ar)-int(br))+abs(int(ag)-int(bg))+abs(int(ab)-int(bb)) > 48 {
				diff++
			}
		}
	}
	if diff > 30 {
		return fmt.Errorf("header bleed on scroll: %d pixels changed", diff)
	}
	return nil
}

func checkTableColumnResize() error {
	tv := widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Width: 100, MinWidth: 40},
		{Title: "Size", Width: 100, MinWidth: 40},
	}, 6, func(row, col int) string { return "x" }, nil)
	s := Mount(tv, paintengine2d.XYWH(0, 0, 280, 140))
	w0 := tv.ColumnWidths()
	if len(w0) < 2 {
		return fmt.Errorf("widths")
	}
	edge := w0[0]
	pt := paintengine2d.Pt(edge, tv.HeaderHeight()*0.5)
	if tv.ColumnDividerAt(pt.X) != 0 {
		return fmt.Errorf("divider at x=%v got %d (w0=%v)", pt.X, tv.ColumnDividerAt(pt.X), w0[0])
	}
	s.MouseMove(pt)
	if s.Cursor() != platform.CursorColResize {
		return fmt.Errorf("hover divider cursor=%v", s.Cursor())
	}
	s.MousePress(pt, platform.ButtonLeft)
	if tv.ResizingColumn() != 0 {
		return fmt.Errorf("resize col %d", tv.ResizingColumn())
	}
	s.MouseMove(paintengine2d.Pt(edge+40, pt.Y))
	w1 := tv.ColumnWidths()
	if w1[0] < w0[0]+20 {
		return fmt.Errorf("col0 did not grow %v -> %v", w0[0], w1[0])
	}
	s.MouseRelease(paintengine2d.Pt(24, tv.HeaderHeight()+20))
	s.MouseMove(paintengine2d.Pt(24, tv.HeaderHeight()+20))
	if tv.ResizingColumn() != -1 {
		return fmt.Errorf("still resizing %d", tv.ResizingColumn())
	}
	if s.Cursor() != platform.CursorDefault {
		return fmt.Errorf("after resize cursor=%v", s.Cursor())
	}
	return nil
}

type bleedBox struct {
	widget.Base
	col paintengine2d.Color
}

func newBleedBox(c paintengine2d.Color) *bleedBox {
	b := &bleedBox{col: c}
	b.Init(b)
	return b
}

func (b *bleedBox) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(80, 80))
}
func (b *bleedBox) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }
func (b *bleedBox) Paint(ctx *paintengine2d.Context) {
	ctx.DrawRect(paintengine2d.XYWH(-800, -80, 2400, 400), paintengine2d.Fill(b.col))
}

func checkSplitterClipCursor() error {
	red := paintengine2d.RGB(0.9, 0.1, 0.1)
	blue := paintengine2d.RGB(0.1, 0.2, 0.9)
	split := widgets.NewSplitter(widgets.SplitColumns, newBleedBox(red), newBleedBox(blue))
	split.Ratio = 0.4
	s := Mount(split, paintengine2d.XYWH(0, 0, 300, 120))
	a, b := split.PaneA(), split.PaneB()
	if err := CheckExclusive(a, b); err != nil {
		return err
	}
	img := s.Paint()
	ax := int((a.Min.X + a.Max.X) * 0.5)
	bx := int((b.Min.X + b.Max.X) * 0.5)
	ar, ag, ab, _ := img.PremulAt(ax, 40)
	br, bg, bb, _ := img.PremulAt(bx, 40)
	if ar < 80 || ag > 80 || ab > 80 {
		return fmt.Errorf("pane A not clipped red, got %d %d %d", ar, ag, ab)
	}
	if br > 80 || bb < 80 {
		return fmt.Errorf("pane B not clipped blue, got %d %d %d", br, bg, bb)
	}
	edge := int(a.Max.X) - 2
	if edge > 1 {
		er, eg, eb, _ := img.PremulAt(edge, 40)
		if eb > er+20 && eb > 80 {
			return fmt.Errorf("B painted into A at x=%d: %d %d %d", edge, er, eg, eb)
		}
	}
	sash := paintengine2d.Pt((a.Max.X+b.Min.X)*0.5, 40)
	s.MouseMove(sash)
	if s.Cursor() != platform.CursorColResize {
		return fmt.Errorf("sash cursor=%v", s.Cursor())
	}
	s.MousePress(sash, platform.ButtonLeft)
	s.MouseMove(paintengine2d.Pt(220, 40))
	s.MouseRelease(paintengine2d.Pt(30, 40))
	s.MouseMove(paintengine2d.Pt(24, 40))
	if s.Cursor() != platform.CursorDefault {
		return fmt.Errorf("after drag cursor=%v", s.Cursor())
	}
	if err := CheckExclusive(split.PaneA(), split.PaneB()); err != nil {
		return fmt.Errorf("after drag: %w", err)
	}
	return nil
}

func checkComboKeyboard() error {
	cb := widgets.NewComboBox([]string{"Alpha", "Beta", "Gamma Longer Label"}, 0, nil)
	root := widgets.NewPad(16, cb)
	s := Mount(root, paintengine2d.XYWH(0, 0, 420, 280))
	want := style.ComboHeight(cb.Look().Metrics())
	sz := cb.Measure(layout.Loose(400, 200))
	if sz.Y != want {
		return fmt.Errorf("combo height %v want ComboH %v", sz.Y, want)
	}
	s.Focus(cb)
	s.Key(platform.KeyDown)
	if !cb.Opened() {
		return fmt.Errorf("Down did not open")
	}
	pop, ok := s.Host.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		return fmt.Errorf("no popup")
	}
	field := widget.DeviceBounds(cb)
	pb := pop.Bounds()
	if pb.Min.Y < field.Max.Y-0.5 && pb.Max.Y > field.Min.Y+0.5 {
		return fmt.Errorf("popup %+v overlaps field %+v", pb, field)
	}
	before := pop.HighlightedIndex()
	s.Key(platform.KeyDown)
	if pop.HighlightedIndex() <= before {
		return fmt.Errorf("Down did not move highlight %d -> %d", before, pop.HighlightedIndex())
	}
	s.Key(platform.KeyEscape)
	if cb.Opened() || s.Host.Popup() != nil {
		return fmt.Errorf("Escape did not dismiss")
	}
	return nil
}

func checkFieldHeights() error {
	tf := widgets.NewTextField("", "filter", nil)
	nf := widgets.NewNumberField(0, 10, 3, 1, nil)
	row := widgets.NewRow(widgets.NewButton("A", nil), nf, widgets.NewButton("B", nil))
	s := Mount(row, paintengine2d.XYWH(0, 0, 480, 40))
	m := tf.Look().Metrics()
	want := style.FieldHeight(m)
	tsz := tf.Measure(layout.Loose(400, 200))
	nsz := nf.Measure(layout.Loose(400, 200))
	if tsz.Y != want {
		return fmt.Errorf("TextField height %v want FieldHeight %v", tsz.Y, want)
	}
	if nsz.Y != want {
		return fmt.Errorf("NumberField height %v want FieldHeight %v", nsz.Y, want)
	}
	if want >= m.ControlH {
		return fmt.Errorf("FieldHeight %v should be < ControlH %v", want, m.ControlH)
	}
	stops := widget.Focusables(row)
	nShell := 0
	nField := 0
	for _, c := range stops {
		switch c.(type) {
		case *widgets.NumberField:
			nShell++
		case *widgets.TextField:
			nField++
		}
	}
	if nShell != 0 {
		return fmt.Errorf("NumberField shell is a tab stop (want field only)")
	}
	if nField != 1 {
		return fmt.Errorf("NumberField field tab stops=%d", nField)
	}
	nf.SetEnabled(false)
	if nf.Field().Enabled() {
		return fmt.Errorf("disabled NumberField left inner editor enabled")
	}
	_ = s
	return nil
}

func checkComboFocusVisible() error {
	cb := widgets.NewComboBox([]string{"One", "Two"}, 0, nil)
	other := widgets.NewButton("Next", nil)
	row := widgets.NewRow(cb, other).WithGap(8)
	s := Mount(row, paintengine2d.XYWH(0, 0, 360, 40))
	pt := paintengine2d.Pt(widget.DeviceBounds(cb).Min.X+12, 20)
	s.MousePress(pt, platform.ButtonLeft)
	s.MouseRelease(pt)
	cb.Close()
	if cb.State().Focused() {
		return fmt.Errorf("mouse click left StateFocused")
	}
	s.Focus(cb)
	widget.MarkKeyboardFocus(cb)
	if !cb.State().Focused() {
		return fmt.Errorf("Tab focus did not paint StateFocused")
	}
	_ = other
	return nil
}

func checkSwitchClick() error {
	n := 0
	sw := widgets.NewSwitch("On", false, func(bool) { n++ })
	s := Mount(sw, paintengine2d.XYWH(0, 0, 160, 32))
	s.MousePress(paintengine2d.Pt(8, 16), platform.ButtonLeft)
	if sw.On || n != 0 {
		return fmt.Errorf("toggled on press")
	}
	s.MouseRelease(paintengine2d.Pt(8, 16))
	if !sw.On || n != 1 {
		return fmt.Errorf("did not toggle on release")
	}
	s.MousePress(paintengine2d.Pt(8, 16), platform.ButtonLeft)
	sw.MouseExit()
	s.MouseRelease(paintengine2d.Pt(200, 16))
	if !sw.On || n != 1 {
		return fmt.Errorf("drag-off still toggled")
	}
	sw.SetEnabled(false)
	sw.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if !sw.On || n != 1 {
		return fmt.Errorf("disabled switch toggled from keyboard")
	}
	return nil
}

func checkSliderChrome() error {
	sl := widgets.NewSlider(0, 100, 10, nil)
	other := widgets.NewButton("x", nil)
	row := widgets.NewRow(sl, other).WithGap(8)
	s := Mount(row, paintengine2d.XYWH(0, 0, 360, 36))
	sl.SetValue(-20)
	if sl.Value != 0 {
		return fmt.Errorf("clamp low %v", sl.Value)
	}
	sl.SetValue(200)
	if sl.Value != 100 {
		return fmt.Errorf("clamp high %v", sl.Value)
	}
	sl.SetValue(10)
	box := widget.DeviceBounds(sl)
	s.MousePress(paintengine2d.Pt(box.Max.X-8, (box.Min.Y+box.Max.Y)*0.5), platform.ButtonLeft)
	s.MouseRelease(paintengine2d.Pt(box.Max.X-8, (box.Min.Y+box.Max.Y)*0.5))
	if sl.Value < 70 {
		return fmt.Errorf("click near end value=%v", sl.Value)
	}
	if sl.State().Focused() {
		return fmt.Errorf("mouse click left StateFocused")
	}
	s.Tab(true)
	if s.Host.Focus() != other {
		return fmt.Errorf("Tab")
	}
	s.Tab(true)
	if !sl.State().Focused() {
		return fmt.Errorf("Tab focus did not paint StateFocused")
	}
	on := s.Paint()
	sl.SetEnabled(false)
	before := sl.Value
	s.MousePress(paintengine2d.Pt(box.Min.X+8, (box.Min.Y+box.Max.Y)*0.5), platform.ButtonLeft)
	s.MouseRelease(paintengine2d.Pt(box.Min.X+8, (box.Min.Y+box.Max.Y)*0.5))
	if sl.Value != before {
		return fmt.Errorf("disabled slider moved %v -> %v", before, sl.Value)
	}
	off := s.Paint()
	if ColorDiff(on, off, 28) < 8 {
		return fmt.Errorf("disabled slider chrome matches enabled")
	}
	return nil
}

func checkProgressChrome() error {
	p := widgets.NewProgressBar(1.5)
	if p.Value != 1 {
		return fmt.Errorf("clamp high %v", p.Value)
	}
	p.SetValue(-1)
	if p.Value != 0 {
		return fmt.Errorf("clamp low %v", p.Value)
	}
	p.SetValue(0.6)
	s := Mount(p, paintengine2d.XYWH(0, 0, 200, 24))
	want := p.Look().Metrics().ProgressH
	sz := p.Measure(layout.Loose(200, 40))
	if want > 0 && sz.Y != want {
		return fmt.Errorf("height %v want ProgressH %v", sz.Y, want)
	}
	on := s.Paint()
	p.SetEnabled(false)
	off := s.Paint()
	if ColorDiff(on, off, 24) < 8 {
		return fmt.Errorf("disabled progress matches enabled")
	}
	return nil
}

func checkTabBarChrome() error {
	bar := widgets.NewTabBar("One", "Two", "Three")
	other := widgets.NewButton("x", nil)
	col := widgets.NewColumn(bar, other)
	s := Mount(col, paintengine2d.XYWH(0, 0, 360, 80))
	rects := tabRects(bar)
	if len(rects) < 3 {
		return fmt.Errorf("tab rects")
	}
	pt := paintengine2d.Pt((rects[1].Min.X+rects[1].Max.X)*0.5, rects[1].Min.Y+8)
	s.MousePress(pt, platform.ButtonLeft)
	if bar.Selected != 0 {
		return fmt.Errorf("selected on press")
	}
	s.MouseRelease(pt)
	if bar.Selected != 1 {
		return fmt.Errorf("not selected on release")
	}
	if bar.State().Focused() {
		return fmt.Errorf("mouse click left StateFocused")
	}
	bar.SetEnabled(false)
	bar.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if bar.Selected != 1 {
		return fmt.Errorf("disabled tab bar moved")
	}
	bar.SetEnabled(true)
	s.Focus(bar)
	widget.MarkKeyboardFocus(bar)
	if !bar.State().Focused() {
		return fmt.Errorf("keyboard focus did not paint StateFocused")
	}
	return nil
}

func tabRects(t *widgets.TabBar) []paintengine2d.Rect {
	// Reconstruct the same layout TabBar uses (font + 28 pad, min 56).
	n := len(t.Titles)
	if n == 0 {
		return nil
	}
	f := t.Look().Font()
	h := t.LocalBounds().Dy()
	x := float32(4)
	out := make([]paintengine2d.Rect, n)
	for i, title := range t.Titles {
		w := f.Advance(title) + 28
		if w < 56 {
			w = 56
		}
		out[i] = paintengine2d.XYWH(x, 0, w, h)
		x += w
	}
	return out
}

func checkDialogButtons() error {
	anchor := widgets.NewLabel("host")
	s := Mount(anchor, paintengine2d.XYWH(0, 0, 480, 320))
	got := widgets.ResultNone
	mb := widgets.ShowMessageBox(anchor, widgets.MessageBoxOptions{
		Title: "Confirm", Message: "Proceed?", Kind: widgets.MessageQuestion,
		Buttons: widgets.ButtonsYesNo, OnResult: func(r widgets.MessageResult) { got = r },
	})
	if mb == nil || s.Host.Overlay() == nil {
		return fmt.Errorf("no overlay")
	}
	s.Layout()
	var yes, no *widgets.Button
	widget.Walk(s.Host.Overlay(), func(c widget.Component) {
		b, ok := c.(*widgets.Button)
		if !ok {
			return
		}
		switch b.Text {
		case "Yes":
			yes = b
		case "No":
			no = b
		}
	})
	if yes == nil || no == nil {
		return fmt.Errorf("missing Yes/No")
	}
	if !yes.Primary || no.Primary {
		return fmt.Errorf("primary Yes=%v No=%v", yes.Primary, no.Primary)
	}
	pt := widget.DeviceBounds(yes)
	mid := paintengine2d.Pt((pt.Min.X+pt.Max.X)*0.5, (pt.Min.Y+pt.Max.Y)*0.5)
	s.MousePress(mid, platform.ButtonLeft)
	if got != widgets.ResultNone {
		return fmt.Errorf("fired on press")
	}
	s.MouseRelease(mid)
	if got != widgets.ResultYes {
		return fmt.Errorf("Yes click got %v", got)
	}
	if s.Host.Overlay() != nil {
		return fmt.Errorf("overlay stayed after Yes")
	}

	got = widgets.ResultNone
	mb = widgets.ShowMessageBox(anchor, widgets.MessageBoxOptions{
		Title: "Confirm", Message: "Cancel?", Kind: widgets.MessageQuestion,
		Buttons: widgets.ButtonsYesNo, OnResult: func(r widgets.MessageResult) { got = r },
	})
	s.Layout()
	if s.Host.Overlay() == nil {
		return fmt.Errorf("second overlay")
	}
	s.Focus(s.Host.Overlay())
	s.Key(platform.KeyEscape)
	if got != widgets.ResultNo {
		return fmt.Errorf("Escape cancel got %v (box=%v)", got, mb.Result())
	}
	return nil
}

func checkMenuGutterIcons() error {
	on := widgets.CheckItem("On", true, nil)
	off := widgets.CheckItem("Off", false, nil)
	ic := widgets.ItemIcon(style.IconSearch, "Find a very long command label", nil)
	pop := widgets.NewPopupMenu(on, off, ic)
	s := Mount(pop, paintengine2d.XYWH(0, 0, 1, 1))
	sz := pop.Measure(layout.Unbounded())
	s.Relayout(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	if err := CheckMenuFitsItems(pop); err != nil {
		return err
	}
	ch := style.MenuChromeFor(pop.Look())
	need := style.IconSizePixels(style.LookIconSize(pop.Look()))
	if ch.CheckCol()+0.5 < need {
		return fmt.Errorf("gutter %v < icon %v", ch.CheckCol(), need)
	}
	img := s.Paint()
	lk := pop.Look()
	r0 := pop.ItemBounds(0).Translate(widget.DeviceOrigin(pop))
	r1 := pop.ItemBounds(1).Translate(widget.DeviceOrigin(pop))
	g0 := paintengine2d.XYWH(r0.Min.X, r0.Min.Y, ch.CheckCol(), r0.Dy()).Inset(2)
	g1 := paintengine2d.XYWH(r1.Min.X, r1.Min.Y, ch.CheckCol(), r1.Dy()).Inset(2)
	empty := style.ResolveMenuChrome(lk.Palette()).MenuGutter
	onInk := CountNonColor(img, g0, empty, 28)
	offInk := CountNonColor(img, g1, empty, 28)
	if onInk < 6 {
		return fmt.Errorf("checked gutter has no mark (ink=%d)", onInk)
	}
	if offInk >= onInk {
		return fmt.Errorf("unchecked gutter ink %d >= checked %d", offInk, onInk)
	}
	wide := pop.Look().Font().Advance("Find a very long command label")
	if pop.LocalBounds().Dx()+1 < wide+ch.PadL+ch.CheckCol()+ch.PadR {
		return fmt.Errorf("menu width %v ignores widest label need %v", pop.LocalBounds().Dx(), wide)
	}
	return nil
}

func checkMenuOnScreen() error {
	items := []*widgets.MenuItem{
		widgets.Item("Reply", nil),
		widgets.Item("Add sender to VIP", nil),
		widgets.Item("Delete", nil),
	}
	const winW, winH float32 = 360, 280
	h := NewHost()
	h.SetLook(style.DarkLook())
	root := widgets.NewLabel("anchor")
	_ = MountHost(h, root, paintengine2d.XYWH(0, 0, winW, winH))
	pop := widgets.ShowContextMenu(root, paintengine2d.Pt(winW-8, 40), items...)
	if pop == nil {
		return fmt.Errorf("ShowContextMenu")
	}
	if err := CheckMenuFitsItems(pop); err != nil {
		return err
	}
	if pop.Bounds().Dx()+0.5 < pop.ContentSize().X {
		return fmt.Errorf("right-edge shrank width %v < %v", pop.Bounds().Dx(), pop.ContentSize().X)
	}
	if pop.Bounds().Max.X > winW-3 {
		return fmt.Errorf("popup %+v past window %v", pop.Bounds(), winW)
	}
	return nil
}
