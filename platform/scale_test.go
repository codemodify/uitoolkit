package platform

import (
	"testing"
)

func TestParseScale(t *testing.T) {
	if parseScale("") != 0 || parseScale("nope") != 0 {
		t.Fatal("empty")
	}
	if parseScale("2") != 2 {
		t.Fatalf("2 -> %v", parseScale("2"))
	}
	if parseScale("1.5") != 1.5 {
		t.Fatalf("1.5 -> %v", parseScale("1.5"))
	}
	if parseScale("1.02") != 1 {
		t.Fatalf("snap 1, got %v", parseScale("1.02"))
	}
	if parseScale("8") != 4 {
		t.Fatalf("clamp high %v", parseScale("8"))
	}
}

func TestScaleFromDPI(t *testing.T) {
	if scaleFromDPI(96) != 1 {
		t.Fatalf("96 dpi %v", scaleFromDPI(96))
	}
	if scaleFromDPI(192) != 2 {
		t.Fatalf("192 dpi %v", scaleFromDPI(192))
	}
	if scaleFromDPI(144) != 1.5 {
		t.Fatalf("144 dpi %v", scaleFromDPI(144))
	}
	if scaleFromDPI(20) != 0 {
		t.Fatal("tiny dpi")
	}
}

func TestScaleFromEnvUITK(t *testing.T) {
	t.Setenv("UITK_SCALE", "2")
	t.Setenv("GDK_SCALE", "3")
	if ScaleFromEnv() != 2 {
		t.Fatalf("UITK wins, got %v", ScaleFromEnv())
	}
	t.Setenv("UITK_SCALE", "")
	if ScaleFromEnv() != 3 {
		t.Fatalf("GDK %v", ScaleFromEnv())
	}
	t.Setenv("GDK_SCALE", "")
	t.Setenv("QT_SCALE_FACTOR", "")
	t.Setenv("GDK_DPI_SCALE", "1.5")
	if ScaleFromEnv() != 1.5 {
		t.Fatalf("GDK_DPI_SCALE %v", ScaleFromEnv())
	}
}

func TestAdoptDisplayScale(t *testing.T) {
	if AdoptDisplayScale(1, 2) != 2 {
		t.Fatal("surface 2× should replace a 1× guess")
	}
	if AdoptDisplayScale(2, 1) != 2 {
		t.Fatal("must not shrink an explicit scale")
	}
	if AdoptDisplayScale(2, 2) != 2 {
		t.Fatal("equal")
	}
	if AdoptDisplayScale(0, 1) != 1 {
		t.Fatal("zero current")
	}
}

func TestFitLogicalSizeRejectsBufferEcho(t *testing.T) {
	lw, lh := fitLogicalSize(2000, 1400, 1000, 700, 2)
	if lw != 1000 || lh != 700 {
		t.Fatalf("buffer echo became logical %dx%d", lw, lh)
	}
	lw, lh = fitLogicalSize(900, 600, 1000, 700, 2)
	if lw != 900 || lh != 600 {
		t.Fatalf("real logical resize %dx%d", lw, lh)
	}
	lw, lh = fitLogicalSize(800, 500, 800, 500, 1)
	if lw != 800 || lh != 500 {
		t.Fatalf("1x %dx%d", lw, lh)
	}
}
