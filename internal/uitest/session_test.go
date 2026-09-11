package uitest

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestMountPaintAndWheel(t *testing.T) {
	lv := widgets.NewListView(40, func(i int) string { return "row" }, nil)
	lv.RowHeight = 16
	s := Mount(lv, paintengine2d.XYWH(0, 0, 120, 64))
	if lv.MaxOffset() <= 0 {
		t.Fatal("expected overflow")
	}
	s.Wheel(paintengine2d.Pt(20, 20), 80)
	if err := CheckClamped(lv.OffsetY, lv.MaxOffset()); err != nil {
		t.Fatal(err)
	}
	img := s.Paint()
	if img == nil || img.Width != 120 {
		t.Fatalf("paint %+v", img)
	}
	if s.Record() == nil {
		t.Fatal("record")
	}
}

func TestSessionContextMenuFitsItems(t *testing.T) {
	h := NewHost()
	h.SetLook(style.WithScale(style.DarkLook(), 2))
	root := widgets.NewLabel("host")
	s := MountHost(h, root, paintengine2d.XYWH(0, 0, 800, 600))
	items := []*widgets.MenuItem{
		widgets.Item("Reply", nil),
		widgets.Item("Forward", nil),
		widgets.Sep(),
		widgets.Item("Mark as Read", nil),
		widgets.Item("Mark as Unread", nil),
		widgets.Item("Tag · Important", nil),
		widgets.Item("Add sender to VIP", nil),
		widgets.Item("Archive", nil),
		widgets.Item("Junk", nil),
		widgets.Item("Delete", nil),
	}
	pop := widgets.ShowContextMenu(root, paintengine2d.Pt(20, 20), items...)
	if pop == nil || h.Popup() != pop {
		t.Fatal("expected hosted popup")
	}
	if err := CheckMenuFitsItems(pop); err != nil {
		t.Fatal(err)
	}
	img := s.Paint()
	if img == nil || img.Width != 800 {
		t.Fatalf("paint %+v", img)
	}
}
