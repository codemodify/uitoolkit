package widgets_test

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Regression map — basic UI bugs that must fail CI before a tag.
//
//	blank-after-first-page  TestListViewScenePaintsPastFirstPage, TestTableViewHeaderFlushAndClip,
//	                        TestCardListScenePaintsPastFirstPage, TestTreeViewPaintsAfterFirstPage
//	header overlap scroll   TestTableViewHeaderFlushAndClip, TestTableViewHeaderClipsBodyWhenScrolled
//	header gap at top       TestTableViewHeaderFlushAndClip, TestTableViewFirstRowFlushUnderHeader
//	splitter overlap        TestSplitterArrangeAfterDragExclusive, TestSplitterPanesExclusiveAtRatios
//	stuck resize cursor     TestSplitterRestoresPointerCursor, TestSplitterCursorReturnsAfterDrag
//	unclamped scroll        TestScrollClampWheelStopsAtEnd, TestScrollersClampOffset
//	type into read-only     TestTextViewReadOnlyAndScrollbar, TestTextAreaReadOnlyRejectsInput
//	menu clip / truncate    TestPopupMenuFitsLongLabelsAndManyItems,
//	                        TestPopupMenuVIPLabelNotClipped,
//	                        TestMenuBarDropdownFitsLabelsAndShortcuts,
//	                        TestMenuBarHelpNearRightEdgeFitsAboutMail
//	combo overlap / clip    TestComboBoxPopupClearsFieldAndFitsLabels
//	toolbar steals MaxW     TestToolBarLeavesRoomForFlexSibling
//	focus-visible / chrome  internal/uitest.TestDesktopChromeNorms (docs/compare.md)
//	label/button clip       TestDesktopChromeNorms/label-clip, button-label-clip
//	toggle vs action        TestDesktopChromeNorms/toggle-vs-action
//	menu leftover focus     TestDesktopChromeNorms/menubar-dismiss-focus, popup-leave-highlight

func TestSplitterPanesExclusiveAtRatios(t *testing.T) {
	left := widgets.NewLabel("AAAA pane A chrome")
	right := widgets.NewLabel("BBBB pane B chrome that must not overlap A")
	split := widgets.NewSplitter(true, left, right)
	s := uitest.Mount(split, paintengine2d.XYWH(0, 0, 400, 180))
	for _, ratio := range []float32{0.08, 0.25, 0.5, 0.75, 0.92} {
		split.Ratio = ratio
		s.Layout()
		a, b := split.PaneA(), split.PaneB()
		if err := uitest.CheckExclusive(a, b); err != nil {
			t.Fatalf("ratio %v: %v", ratio, err)
		}
		if a.Max.X > b.Min.X+0.01 {
			t.Fatalf("ratio %v A not exclusive of B A.max=%v B.min=%v", ratio, a.Max.X, b.Min.X)
		}
	}
}

func TestSplitterCursorReturnsAfterDrag(t *testing.T) {
	split := widgets.NewSplitter(true, widgets.NewLabel("left"), widgets.NewLabel("right"))
	split.Ratio = 0.4
	s := uitest.Mount(split, paintengine2d.XYWH(0, 0, 400, 160))
	a, b := split.PaneA(), split.PaneB()
	sash := paintengine2d.Pt((a.Max.X+b.Min.X)*0.5, 40)
	s.MouseMove(sash)
	if s.Cursor() != platform.CursorColResize {
		t.Fatalf("hover sash cursor=%v want col-resize", s.Cursor())
	}
	s.MousePress(sash, platform.ButtonLeft)
	s.MouseMove(paintengine2d.Pt(300, 40))
	if s.Cursor() != platform.CursorColResize {
		t.Fatalf("during drag cursor=%v", s.Cursor())
	}
	// Release in pane A, then leave the sash — stuck-cursor bug stays col-resize.
	s.MouseRelease(paintengine2d.Pt(40, 40))
	s.MouseMove(paintengine2d.Pt(24, 40))
	if s.Cursor() != platform.CursorDefault {
		t.Fatalf("after release off sash cursor=%v want default", s.Cursor())
	}
}

func TestScrollersClampOffset(t *testing.T) {
	lv := widgets.NewListView(40, func(i int) string { return fmt.Sprintf("row-%02d", i) }, nil)
	lv.RowHeight = 16
	_ = uitest.Mount(lv, paintengine2d.XYWH(0, 0, 160, 64))
	lv.ScrollTo(-80)
	if lv.OffsetY != 0 {
		t.Fatalf("list neg offset %v", lv.OffsetY)
	}
	lv.ScrollTo(1e6)
	if lv.OffsetY != lv.MaxOffset() || lv.MaxOffset() <= 0 {
		t.Fatalf("list past-end %v max=%v", lv.OffsetY, lv.MaxOffset())
	}
	if _, thumb := lv.ScrollTrack(); thumb.Empty() {
		t.Fatal("overflow list thumb")
	}
	if err := uitest.CheckThumbInTrack(lv.ScrollTrack()); err != nil {
		t.Fatal(err)
	}

	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "Col"}}, 30, func(row, col int) string {
		return fmt.Sprintf("R%02d", row)
	}, nil)
	tv.RowHeight = 18
	_ = uitest.Mount(tv, paintengine2d.XYWH(0, 0, 200, 90))
	tv.ScrollTo(-40)
	if tv.OffsetY != 0 {
		t.Fatalf("table neg %v", tv.OffsetY)
	}
	tv.ScrollTo(1e6)
	if tv.OffsetY != tv.MaxOffset() || tv.MaxOffset() <= 0 {
		t.Fatalf("table past-end %v max=%v", tv.OffsetY, tv.MaxOffset())
	}

	cards := widgets.NewCardList(20, func(i int) widgets.CardContent {
		return widgets.CardContent{Title: fmt.Sprintf("Card %d", i), Subtitle: "sub", Meta: "now", Snippet: "snip"}
	}, nil)
	cards.CardHeight = 48
	_ = uitest.Mount(cards, paintengine2d.XYWH(0, 0, 240, 120))
	cards.ScrollTo(1e6)
	if cards.OffsetY != cards.MaxOffset() || cards.MaxOffset() <= 0 {
		t.Fatalf("cards %v max=%v", cards.OffsetY, cards.MaxOffset())
	}

	root := widgets.NewTreeNode("root")
	for i := 0; i < 40; i++ {
		root.Children = append(root.Children, widgets.NewTreeNode(fmt.Sprintf("n%d", i)))
	}
	root.Expanded = true
	tree := widgets.NewTreeView(root)
	_ = uitest.Mount(tree, paintengine2d.XYWH(0, 0, 180, 80))
	tree.ScrollTo(1e6)
	if tree.OffsetY != tree.MaxOffset() || tree.MaxOffset() <= 0 {
		t.Fatalf("tree %v max=%v", tree.OffsetY, tree.MaxOffset())
	}

	col := widgets.NewColumn()
	for i := 0; i < 40; i++ {
		col.Add(widgets.NewLabel("row"))
	}
	sv := widgets.NewScrollView(col)
	_ = uitest.Mount(sv, paintengine2d.XYWH(0, 0, 160, 80))
	sv.ScrollTo(-10)
	if sv.OffsetY != 0 {
		t.Fatalf("sv neg %v", sv.OffsetY)
	}
	sv.ScrollTo(1e6)
	if sv.OffsetY != sv.MaxOffset() || sv.MaxOffset() <= 0 {
		t.Fatalf("sv %v max=%v", sv.OffsetY, sv.MaxOffset())
	}
}

func TestTreeViewPaintsAfterFirstPage(t *testing.T) {
	root := widgets.NewTreeNode("root")
	root.Expanded = true
	for i := 0; i < 50; i++ {
		root.Children = append(root.Children, widgets.NewTreeNode(fmt.Sprintf("====NODE %02d====", i)))
	}
	tree := widgets.NewTreeView(root)
	s := uitest.Mount(tree, paintengine2d.XYWH(0, 0, 220, 90))
	if tree.MaxOffset() <= 0 {
		t.Fatal("expected overflow")
	}
	tree.ScrollTo(tree.LocalBounds().Dy() + 24)
	lo, hi := tree.VisibleRange()
	if lo < 1 || hi <= lo {
		t.Fatalf("window lo=%d hi=%d", lo, hi)
	}
	img := s.Paint()
	field := uitest.FieldColor(tree.Look())
	painted := uitest.CountNonColor(img, tree.LocalBounds().Inset(4), field, 18)
	if painted < 20 {
		t.Fatalf("tree blank after first page: %d lo=%d hi=%d", painted, lo, hi)
	}
}

func TestTableViewFirstRowFlushUnderHeader(t *testing.T) {
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "HHHHHHHHHH"}}, 12,
		func(row, col int) string { return fmt.Sprintf("====ROW %02d====", row) }, nil)
	tv.RowHeight = 22
	s := uitest.Mount(tv, paintengine2d.XYWH(0, 0, 260, 140))
	if tv.OffsetY != 0 {
		t.Fatalf("start offset %v", tv.OffsetY)
	}
	row0 := tv.RowBounds(0)
	hh := tv.HeaderHeight()
	if row0.Empty() {
		t.Fatal("row0 empty")
	}
	if d := row0.Min.Y - hh; d < -1 || d > 2 {
		t.Fatalf("header gap: first row y=%v headerH=%v delta=%v", row0.Min.Y, hh, d)
	}
	img := s.Paint()
	field := uitest.FieldColor(tv.Look())
	band := paintengine2d.XYWH(8, hh+2, 180, row0.Dy()-4)
	painted := uitest.CountNonColor(img, band, field, 16)
	if painted < 10 {
		t.Fatalf("large gap under header: non-field pixels in first-row band=%d band=%+v", painted, band)
	}
}

func TestTableViewHeaderClipsBodyWhenScrolled(t *testing.T) {
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "HHHHHHHHHH"}}, 20,
		func(row, col int) string { return fmt.Sprintf("====ROW %02d====", row) }, nil)
	tv.RowHeight = 22
	tv.Selected = 0
	s := uitest.Mount(tv, paintengine2d.XYWH(0, 0, 260, 140))
	hh := tv.HeaderHeight()
	img0 := s.Paint()

	tv.ScrollTo(hh + tv.RowBounds(0).Dy())
	if tv.OffsetY <= 0 {
		t.Fatal("expected scroll")
	}
	row0 := tv.RowBounds(0)
	img1 := s.Paint()
	// Sticky header must not pick up selected-row / body pixels (bleed).
	diff := 0
	for y := 1; y < int(hh)-1 && y < img0.Height && y < img1.Height; y++ {
		for x := 4; x < 80 && x < img0.Width && x < img1.Width; x++ {
			ar, ag, ab, _ := img0.PremulAt(x, y)
			br, bg, bb, _ := img1.PremulAt(x, y)
			dr, dg, db := int(ar)-int(br), int(ag)-int(bg), int(ab)-int(bb)
			if dr < 0 {
				dr = -dr
			}
			if dg < 0 {
				dg = -dg
			}
			if db < 0 {
				db = -db
			}
			if dr+dg+db > 48 {
				diff++
			}
		}
	}
	if diff > 30 {
		t.Fatalf("header bleed on scroll-down: %d header pixels changed (row0=%+v offset=%v)", diff, row0, tv.OffsetY)
	}
	field := uitest.FieldColor(tv.Look())
	band := paintengine2d.XYWH(8, hh+2, 180, tv.RowBounds(1).Dy()-4)
	if uitest.CountNonColor(img1, band, field, 16) < 8 {
		t.Fatal("body under header empty after scroll")
	}
}

func TestTextAreaReadOnlyRejectsInput(t *testing.T) {
	view := widgets.NewTextView("plain message body\nline two", "view")
	s := uitest.Mount(view, paintengine2d.XYWH(0, 0, 240, 80))
	s.Focus(view)
	before := view.Text
	if view.TextInput('x') {
		t.Fatal("read-only TextView accepted TextInput")
	}
	s.Type("typed into view")
	view.KeyPress(widget.KeyEvent{Key: platform.KeyA})
	view.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	view.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if view.Text != before {
		t.Fatalf("read-only mutated %q -> %q", before, view.Text)
	}

	edit := widgets.NewTextArea("hello", "", nil)
	s = uitest.Mount(edit, paintengine2d.XYWH(0, 0, 240, 80))
	s.Focus(edit)
	edit.SetSelection(5, 5)
	if !edit.TextInput('!') {
		t.Fatal("editable TextArea rejected input")
	}
	s.Type("ok")
	if edit.Text != "hello!ok" {
		t.Fatalf("editable %q", edit.Text)
	}
}

func testLook(scale float32) style.LookAndFeel {
	look := style.LookAndFeel(style.DarkLook())
	if scale > 1 {
		look = style.WithScale(look, scale)
	}
	return look
}

func TestPopupMenuFitsLongLabelsAndManyItems(t *testing.T) {
	items := []*widgets.MenuItem{
		widgets.Item("Reply", nil),
		widgets.Item("Forward", nil),
		widgets.Sep(),
		widgets.Item("Mark as Read", nil),
		widgets.Item("Mark as Unread", nil),
		widgets.Item("Star", nil),
		widgets.Sep(),
		widgets.Item("Tag · Important", nil),
		widgets.Item("Mute Thread", nil),
		widgets.Item("Add sender to VIP", nil),
		widgets.Item("Archive", nil),
		widgets.Item("Junk", nil),
		widgets.Item("Delete", nil),
	}
	for _, scale := range []float32{1, 2} {
		h := uitest.NewHost()
		h.SetLook(testLook(scale))
		h.SetScale(scale)
		root := widgets.NewLabel("anchor")
		_ = uitest.MountHost(h, root, paintengine2d.XYWH(0, 0, 640, 480))
		pop := widgets.ShowContextMenu(root, paintengine2d.Pt(40, 40), items...)
		if pop == nil {
			t.Fatalf("scale %v: ShowContextMenu failed", scale)
		}
		if err := uitest.CheckMenuFitsItems(pop); err != nil {
			t.Fatalf("scale %v: %v bounds=%+v content=%+v", scale, err, pop.Bounds(), pop.ContentSize())
		}
		f := pop.Look().Font()
		ch := style.MenuChromeFor(pop.Look())
		for _, label := range []string{"Mark as Read", "Mark as Unread", "Tag · Important", "Add sender to VIP"} {
			need := f.Advance(label) + ch.PadL + ch.CheckCol() + ch.ItemPad + ch.PadR
			if pop.LocalBounds().Dx()+0.5 < need {
				t.Fatalf("scale %v width %v truncates %q need %v", scale, pop.LocalBounds().Dx(), label, need)
			}
		}
		last := pop.ItemBounds(len(pop.Items) - 1)
		if pop.MaxOffset() <= 0 && last.Max.Y > pop.LocalBounds().Dy()+1 {
			t.Fatalf("scale %v last item clipped %+v menuH=%v", scale, last, pop.LocalBounds().Dy())
		}
	}
}

func TestPopupMenuVIPLabelNotClipped(t *testing.T) {
	const vip = "Add sender to VIP"
	items := []*widgets.MenuItem{
		widgets.Item("Reply", nil),
		widgets.Item("Forward", nil),
		widgets.Sep(),
		widgets.Item("Mark as Read", nil),
		widgets.Item("Tag · Important", nil),
		widgets.Item("Mute Thread", nil),
		widgets.Item(vip, nil),
		widgets.Item("Archive", nil),
		widgets.Item("Junk", nil),
		widgets.Item("Delete", nil),
	}
	for _, scale := range []float32{1, 2} {
		for _, dens := range []style.Density{style.DensityDefault, style.DensityCompact, style.DensityRelaxed} {
			h := uitest.NewHost()
			look := style.WithDensity(style.DarkLook(), dens)
			if scale > 1 {
				look = style.WithScale(look, scale)
			}
			h.SetLook(look)
			h.SetScale(scale)
			root := widgets.NewLabel("host")
			s := uitest.MountHost(h, root, paintengine2d.XYWH(0, 0, 720, 520))
			pop := widgets.ShowContextMenu(root, paintengine2d.Pt(24, 24), items...)
			if pop == nil {
				t.Fatalf("scale %v density %s: no popup", scale, dens)
			}
			if err := uitest.CheckMenuFitsItems(pop); err != nil {
				t.Fatalf("scale %v density %s: %v", scale, dens, err)
			}
			idx := -1
			for i, it := range pop.Items {
				if it != nil && it.Text == vip {
					idx = i
					break
				}
			}
			if idx < 0 {
				t.Fatal(vip)
			}
			f := pop.Look().Font()
			ch := style.MenuChromeFor(pop.Look())
			adv := f.Advance(vip)
			ink := f.InkWidth(vip)
			if ink > adv {
				adv = ink
			}
			need := adv + ch.PadL + ch.CheckCol() + ch.ItemPad + ch.PadR + ch.Border
			if pop.LocalBounds().Dx()+0.5 < need {
				t.Fatalf("scale %v density %s width %v < need %v", scale, dens, pop.LocalBounds().Dx(), need)
			}
			lb := pop.LabelBounds(idx)
			item := pop.ItemBounds(idx)
			textMax := lb.Min.X + adv
			if lb.Dx()+0.5 < adv {
				t.Fatalf("scale %v density %s label col %v < advance %v", scale, dens, lb.Dx(), adv)
			}
			if textMax > item.Max.X+0.5 {
				t.Fatalf("scale %v density %s text maxX %v exceeds item %+v", scale, dens, textMax, item)
			}
			if textMax > pop.LocalBounds().Max.X+0.5 {
				t.Fatalf("scale %v density %s text maxX %v exceeds menu %+v", scale, dens, textMax, pop.LocalBounds())
			}
			img := s.Paint()
			if img == nil {
				t.Fatal("paint")
			}
			origin := widget.DeviceOrigin(pop)
			prefix := f.Advance("Add sender to VI")
			x0 := int(origin.X + lb.Min.X + prefix + 0.5)
			x1 := int(origin.X + textMax + 0.5)
			y0 := int(origin.Y + item.Min.Y + 2)
			y1 := int(origin.Y + item.Max.Y - 2)
			if x1 <= x0 {
				x1 = x0 + 1
			}
			inkN := 0
			for y := y0; y < y1 && y < img.Height; y++ {
				if y < 0 {
					continue
				}
				for x := x0; x < x1 && x < img.Width; x++ {
					if x < 0 {
						continue
					}
					_, _, _, a := img.PremulAt(x, y)
					if a > 20 {
						inkN++
					}
				}
			}
			if inkN < 4 {
				t.Fatalf("scale %v density %s: %q last glyph has no ink in x[%d,%d] y[%d,%d] (clipped?)", scale, dens, vip, x0, x1, y0, y1)
			}
			widget.DismissPopup(root)
		}
	}
}

func TestPopupMenuAtRightEdgeKeepsIntrinsicWidth(t *testing.T) {
	items := []*widgets.MenuItem{
		widgets.Item("Reply", nil),
		widgets.Item("Add sender to VIP", nil),
		widgets.Item("Delete", nil),
	}
	const winW, winH float32 = 360, 280
	h := uitest.NewHost()
	h.SetLook(style.DarkLook())
	root := widgets.NewLabel("anchor")
	_ = uitest.MountHost(h, root, paintengine2d.XYWH(0, 0, winW, winH))
	pop := widgets.ShowContextMenu(root, paintengine2d.Pt(winW-8, 40), items...)
	if pop == nil {
		t.Fatal("ShowContextMenu")
	}
	if err := uitest.CheckMenuFitsItems(pop); err != nil {
		t.Fatal(err)
	}
	want := pop.ContentSize().X
	got := pop.Bounds().Dx()
	if got+0.5 < want {
		t.Fatalf("right-edge placement shrank width %v < intrinsic %v bounds=%+v", got, want, pop.Bounds())
	}
	if pop.Bounds().Max.X > winW-3 {
		t.Fatalf("popup %+v past window %v", pop.Bounds(), winW)
	}
	if pop.Bounds().Min.X >= winW-8-0.5 {
		t.Fatalf("popup should translate left of cursor, got %+v", pop.Bounds())
	}
}

func TestMenuBarDropdownFitsLabelsAndShortcuts(t *testing.T) {
	mb := widgets.NewMenuBar(widgets.NewMenu("&Message",
		widgets.ItemAccel("&New Message", "c", nil),
		widgets.ItemAccel("&Reply", "r", nil),
		widgets.ItemAccel("&Forward", "f", nil),
		widgets.Sep(),
		widgets.ItemAccel("Mark as Read", "M", nil),
		widgets.Item("Mark as Unread", nil),
		widgets.Item("Tag · Important", nil),
	))
	for _, scale := range []float32{1, 2} {
		h := uitest.NewHost()
		h.SetLook(testLook(scale))
		h.SetScale(scale)
		_ = uitest.MountHost(h, mb, paintengine2d.XYWH(0, 0, 900, 500))
		mb.Open(0)
		pop, ok := h.Popup().(*widgets.PopupMenu)
		if !ok || pop == nil {
			t.Fatalf("scale %v: menubar popup %T", scale, h.Popup())
		}
		if err := uitest.CheckMenuFitsItems(pop); err != nil {
			t.Fatalf("scale %v: %v", scale, err)
		}
		f := pop.Look().Font()
		for i, it := range pop.Items {
			if it == nil || it.Separator {
				continue
			}
			label, _, _ := widgets.ParseMnemonic(it.Text)
			lb := pop.LabelBounds(i)
			if f.Advance(label) > lb.Dx()+0.5 {
				t.Fatalf("scale %v label %q clipped in %+v", scale, label, lb)
			}
			if it.Shortcut == "" {
				continue
			}
			sb := pop.ShortcutBounds(i)
			if sb.Empty() || f.Advance(it.Shortcut) > sb.Dx()+0.5 {
				t.Fatalf("scale %v shortcut %q %+v", scale, it.Shortcut, sb)
			}
			if err := uitest.CheckExclusive(lb.Inset(0.5), sb.Inset(0.5)); err != nil {
				t.Fatalf("scale %v overlap %q / %q: %v", scale, label, it.Shortcut, err)
			}
		}
		widget.DismissPopup(mb)
	}
}

func TestMenuBarHelpNearRightEdgeFitsAboutMail(t *testing.T) {
	mb := widgets.NewMenuBar(
		widgets.NewMenu("&File", widgets.Item("Quit", nil)),
		widgets.NewMenu("&Edit", widgets.Item("Copy", nil)),
		widgets.NewMenu("&View", widgets.Item("Layout", nil)),
		widgets.NewMenu("&Go", widgets.Item("Inbox", nil)),
		widgets.NewMenu("&Message", widgets.Item("Reply", nil)),
		widgets.NewMenu("&Tools", widgets.Item("Prefs", nil)),
		widgets.NewMenu("&Help",
			widgets.Item("Keyboard", nil),
			widgets.Item("About Mail", nil),
		),
	)
	for _, scale := range []float32{1, 2} {
		h := uitest.NewHost()
		h.SetLook(testLook(scale))
		h.SetScale(scale)
		const winW, winH float32 = 360, 240
		_ = uitest.MountHost(h, mb, paintengine2d.XYWH(0, 0, winW, winH))
		help := len(mb.Menus()) - 1
		f := mb.Look().Font()
		tx := float32(4)
		var helpMinX float32
		for i, menu := range mb.Menus() {
			label, _, _ := widgets.ParseMnemonic(menu.Title)
			tw := f.Advance(label) + 20
			if i == help {
				helpMinX = tx
			}
			tx += tw
		}
		mb.Open(help)
		pop, ok := h.Popup().(*widgets.PopupMenu)
		if !ok || pop == nil {
			t.Fatalf("scale %v: Help popup %T", scale, h.Popup())
		}
		if err := uitest.CheckMenuFitsItems(pop); err != nil {
			t.Fatalf("scale %v: %v bounds=%+v", scale, err, pop.Bounds())
		}
		about := -1
		for i, it := range pop.Items {
			if it != nil && it.Text == "About Mail" {
				about = i
				break
			}
		}
		if about < 0 {
			t.Fatal("About Mail item")
		}
		lb := pop.LabelBounds(about)
		need := pop.Look().Font().Advance("About Mail")
		if need > lb.Dx()+0.5 {
			t.Fatalf("scale %v About Mail clipped in %+v need %v popup=%+v", scale, lb, need, pop.Bounds())
		}
		pb := pop.Bounds()
		if pb.Max.X > winW+0.5 {
			t.Fatalf("scale %v popup %+v past window width %v", scale, pb, winW)
		}
		remain := winW - (widget.DeviceOrigin(mb).X + helpMinX)
		if pb.Dx()+1 < need+30 {
			t.Fatalf("scale %v popup width %v cropped to the right-edge remainder %v", scale, pb.Dx(), remain)
		}
		if pb.Min.X > helpMinX+1 && pb.Max.X > winW-8 {
			t.Fatalf("scale %v popup should shift left of Help title x=%v, got %+v", scale, helpMinX, pb)
		}
		widget.DismissPopup(mb)
	}
}

func TestComboBoxPopupClearsFieldAndFitsLabels(t *testing.T) {
	cb := widgets.NewComboBox([]string{
		"QQ <j9@nchip.com>",
		"Work <you@example.com>",
		"+plus checked-looking label",
	}, 0, nil)
	root := widgets.NewPad(24, cb)
	for _, scale := range []float32{1, 2} {
		h := uitest.NewHost()
		h.SetLook(testLook(scale))
		h.SetScale(scale)
		s := uitest.MountHost(h, root, paintengine2d.XYWH(0, 0, 640, 400))
		s.Layout()
		cb.Open()
		pop, ok := h.Popup().(*widgets.PopupMenu)
		if !ok || pop == nil || !cb.Opened() {
			t.Fatalf("scale %v: combo popup", scale)
		}
		field := widget.DeviceBounds(cb)
		pb := pop.Bounds()
		if pb.Min.Y < field.Max.Y-0.5 && pb.Max.Y > field.Min.Y+0.5 {
			t.Fatalf("scale %v popup %+v overlaps field %+v", scale, pb, field)
		}
		if pb.Dx()+0.5 < field.Dx() {
			t.Fatalf("scale %v popup width %v < field %v", scale, pb.Dx(), field.Dx())
		}
		if err := uitest.CheckMenuFitsItems(pop); err != nil {
			t.Fatalf("scale %v: %v", scale, err)
		}
		f := pop.Look().Font()
		for i, it := range pop.Items {
			r := pop.ItemBounds(i)
			if r.Dy()+0.5 < f.Height() {
				t.Fatalf("scale %v row %d height %v < font %v", scale, i, r.Dy(), f.Height())
			}
			if it != nil && f.Advance(it.Text) > pop.LabelBounds(i).Dx()+0.5 {
				t.Fatalf("scale %v combo label %q clipped", scale, it.Text)
			}
		}
		cb.Close()
	}
}

func TestTreeInvariantsOnMountedTable(t *testing.T) {
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "A"}}, 25, func(row, col int) string {
		return fmt.Sprintf("%d", row)
	}, nil)
	s := uitest.Mount(tv, paintengine2d.XYWH(0, 0, 200, 96))
	s.Wheel(paintengine2d.Pt(40, 50), 80)
	if errs := uitest.TreeInvariants(tv); len(errs) > 0 {
		t.Fatal(errs)
	}
}

func TestToolBarLeavesRoomForFlexSibling(t *testing.T) {
	bar := widgets.NewToolBar(
		widgets.ToolText("Get Messages", nil),
		widgets.ToolText("Write", nil),
		widgets.ToolDivider(),
		widgets.ToolText("Cards", nil),
		widgets.ToolText("Classic", nil),
	)
	field := widgets.NewTextField("", "Quick Filter (subject, people, body)", nil)
	qf := widgets.NewRow(field).WithGap(8).WithPadding(8, 4, 8, 4)
	spacer := widgets.NewRow()
	row := widgets.NewRow(bar, spacer, qf).WithGap(8).WithPadding(4, 0, 8, 0)
	row.AddFlex(spacer, 1)
	const rowW float32 = 800
	uitest.Mount(row, paintengine2d.XYWH(0, 0, rowW, 40))

	if qf.Bounds().Dx() <= 100 {
		t.Fatalf("flex sibling crushed: qf width=%v toolbar=%v", qf.Bounds().Dx(), bar.Bounds().Dx())
	}
	if field.Bounds().Dx() <= 100 {
		t.Fatalf("filter field crushed: field width=%v", field.Bounds().Dx())
	}
	content := bar.Measure(layout.Unbounded())
	if bar.Bounds().Dx() > rowW*0.5 {
		t.Fatalf("toolbar width %v claimed half+ of row %v", bar.Bounds().Dx(), rowW)
	}
	if bar.Bounds().Dx() > content.X+1 {
		t.Fatalf("toolbar arranged %v > intrinsic %v", bar.Bounds().Dx(), content.X)
	}
}
