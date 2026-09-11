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
