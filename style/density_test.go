package style

import "testing"

func TestLookScaleAndDip(t *testing.T) {
	if LookScale(nil) != 1 || Dip(nil, 10) != 10 {
		t.Fatal("nil look")
	}
	if LookScale(DarkLook()) != 1 {
		t.Fatalf("1x %v", LookScale(DarkLook()))
	}
	hi := WithScale(DarkLook(), 2)
	if LookScale(hi) != 2 {
		t.Fatalf("2x %v", LookScale(hi))
	}
	if Dip(hi, 28) != 56 {
		t.Fatalf("dip %v", Dip(hi, 28))
	}
}

func TestFittedRowHeightHiDPI(t *testing.T) {
	one := FittedRowHeight(DarkLook(), 28)
	if one != 28 {
		t.Fatalf("1x should keep design height, got %v", one)
	}
	hi := WithScale(DarkLook(), 2)
	h := FittedRowHeight(hi, 28)
	fh := hi.Font().Height()
	if h < fh+hi.Metrics().RowPad {
		t.Fatalf("2x row %v too short for font %v", h, fh)
	}
	if h < 50 {
		t.Fatalf("2x row %v should grow from 28", h)
	}
	if FittedRowHeight(hi, 26) <= 28 {
		t.Fatal("tree default must grow on HiDPI")
	}
}

func TestApplyDensityRowHeights(t *testing.T) {
	def := ApplyDensity(DefaultMetrics(), DensityDefault)
	cmp := ApplyDensity(DefaultMetrics(), DensityCompact)
	rel := ApplyDensity(DefaultMetrics(), DensityRelaxed)
	if cmp.RowH >= def.RowH {
		t.Fatalf("compact row %v >= default %v", cmp.RowH, def.RowH)
	}
	if rel.RowH <= def.RowH {
		t.Fatalf("relaxed row %v <= default %v", rel.RowH, def.RowH)
	}
	if ParseDensity("compact") != DensityCompact || DensityRelaxed.String() != "relaxed" {
		t.Fatal("parse/string")
	}
	look := WithDensity(DarkLook(), DensityCompact)
	if look.Metrics().RowH != cmp.RowH {
		t.Fatalf("look row %v", look.Metrics().RowH)
	}
	if look.Font() == nil || look.Font().Size < 12 {
		t.Fatal("compact font")
	}
}

func TestBoldFontMatchesBodySize(t *testing.T) {
	for _, lk := range []LookAndFeel{DarkLook(), WithScale(DarkLook(), 2)} {
		if lk.BoldFont() == nil || lk.BoldFont().Weight != WeightBold {
			t.Fatal("bold face")
		}
		if lk.BoldFont().Size != lk.Font().Size {
			t.Fatalf("bold %v body %v", lk.BoldFont().Size, lk.Font().Size)
		}
		if lk.TitleFont().Size <= lk.Font().Size {
			t.Fatal("title should stay larger than body")
		}
	}
}
