package style

import "testing"

func TestScaleMetricsDoublesControls(t *testing.T) {
	m := ScaleMetrics(DefaultMetrics(), 2)
	base := DefaultMetrics()
	if m.ControlH != base.ControlH*2 || m.FontSize != base.FontSize*2 {
		t.Fatalf("scaled %+v vs %+v", m, base)
	}
	if ScaleMetrics(base, 1).ControlH != base.ControlH {
		t.Fatal("identity")
	}
}

func TestWithScaleRebuildsClassic(t *testing.T) {
	look := WithScale(DarkLook(), 2)
	if look.Metrics().ControlH != DefaultMetrics().ControlH*2 {
		t.Fatalf("look metrics %v", look.Metrics().ControlH)
	}
	if WithScale(DarkLook(), 1) == nil {
		t.Fatal("scale 1")
	}
}
