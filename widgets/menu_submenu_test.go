package widgets_test

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestPopupMenuSubmenuOpensAndActivates(t *testing.T) {
	picked := ""
	root := widgets.NewLabel("host")
	s := uitest.Mount(root, paintengine2d.XYWH(0, 0, 640, 400))
	pop := widgets.NewPopupMenu(
		widgets.Submenu("&View",
			widgets.RadioItem("&Compact", "density", false, func() { picked = "Compact" }),
			widgets.RadioItem("&Relaxed", "density", false, func() { picked = "Relaxed" }),
		),
		widgets.Sep(),
		widgets.Item("Quit", func() { picked = "Quit" }),
	)
	widget.PreparePopup(root, pop)
	widget.PlacePopup(pop, paintengine2d.Pt(24, 24), 360, 480)
	if !widget.ShowPopup(root, pop) {
		t.Fatal("show parent")
	}
	s.Host.RequestFocus(pop)

	view := pop.ItemBounds(0)
	if view.Empty() {
		t.Fatal("view row")
	}
	o := widget.DeviceOrigin(pop)
	mid := paintengine2d.Pt(o.X+(view.Min.X+view.Max.X)*0.5, o.Y+(view.Min.Y+view.Max.Y)*0.5)
	s.MouseMove(mid)
	sub := pop.CascadeMenu()
	if sub == nil || len(sub.Items) < 2 {
		t.Fatalf("submenu did not open: %+v", pop.Cascade())
	}
	if sub.Bounds().Min.X+1 < pop.Bounds().Max.X-2 {
		t.Fatalf("submenu %+v should sit to the right of parent %+v", sub.Bounds(), pop.Bounds())
	}

	row := sub.ItemBounds(0)
	so := widget.DeviceOrigin(sub)
	pt := paintengine2d.Pt(so.X+(row.Min.X+row.Max.X)*0.5, so.Y+(row.Min.Y+row.Max.Y)*0.5)
	s.MouseMove(pt)
	s.MousePress(pt, platform.ButtonLeft)
	s.MouseRelease(pt)
	if picked != "Compact" {
		t.Fatalf("activate got %q", picked)
	}
	if s.Host.Popup() != nil {
		t.Fatal("parent should dismiss with the submenu pick")
	}
}

func TestPopupMenuSubmenuClickDoesNotDismissParent(t *testing.T) {
	n := 0
	root := widgets.NewLabel("host")
	s := uitest.Mount(root, paintengine2d.XYWH(0, 0, 640, 400))
	pop := widgets.NewPopupMenu(
		widgets.Submenu("View", widgets.Item("Compact", func() { n++ })),
		widgets.Item("Quit", nil),
	)
	widget.PreparePopup(root, pop)
	widget.PlacePopup(pop, paintengine2d.Pt(16, 16), 360, 480)
	if !widget.ShowPopup(root, pop) {
		t.Fatal("show")
	}
	r := pop.ItemBounds(0)
	o := widget.DeviceOrigin(pop)
	pt := paintengine2d.Pt(o.X+(r.Min.X+r.Max.X)*0.5, o.Y+(r.Min.Y+r.Max.Y)*0.5)
	s.MouseMove(pt)
	s.MousePress(pt, platform.ButtonLeft)
	s.MouseRelease(pt)
	if n != 0 {
		t.Fatal("submenu parent row should not fire a command")
	}
	if pop.CascadeMenu() == nil {
		t.Fatal("clicking View should keep the child open")
	}
	if s.Host.Popup() != pop {
		t.Fatal("parent popup dismissed")
	}
}

func TestSubmenuConstructor(t *testing.T) {
	it := widgets.Submenu("View", widgets.Item("Compact", nil), widgets.Sep())
	if it.Text != "View" || !it.HasSubmenu() || len(it.Submenu) != 2 {
		t.Fatalf("%+v", *it)
	}
	if it.OnClick != nil || it.Checkable {
		t.Fatal("submenu row is not a command")
	}
}
