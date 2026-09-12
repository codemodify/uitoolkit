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

func TestComboHeightMatchesToolbarNotControl(t *testing.T) {
	m := DefaultMetrics()
	h := ComboHeight(m)
	if h != m.ComboH || h != 30 {
		t.Fatalf("ComboHeight %v want ComboH 30", h)
	}
	if h >= m.ControlH {
		t.Fatalf("ComboHeight %v should be shorter than ControlH %v", h, m.ControlH)
	}
	look := DarkLook().Metrics()
	if ComboHeight(look) != 30 {
		t.Fatalf("DarkLook ComboHeight %v; icon-size ToolBtn %v must not grow combo", ComboHeight(look), look.ToolBtn)
	}
	scaled := ScaleMetrics(m, 2)
	if ComboHeight(scaled) != scaled.ComboH {
		t.Fatalf("scaled ComboHeight %v want %v", ComboHeight(scaled), scaled.ComboH)
	}
}
