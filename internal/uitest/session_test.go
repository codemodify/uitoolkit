package uitest

import (
	"testing"

	"github.com/codemodify/paintengine2d"
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
