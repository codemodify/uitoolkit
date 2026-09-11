package platform

import "testing"

func TestMaskShift(t *testing.T) {
	if maskShift(0x000000ff) != 0 {
		t.Fatal("blue")
	}
	if maskShift(0x0000ff00) != 8 {
		t.Fatal("green")
	}
	if maskShift(0x00ff0000) != 16 {
		t.Fatal("red")
	}
	if maskShift(0) != 0 {
		t.Fatal("zero")
	}
}

func TestPackXPixelBGRA(t *testing.T) {
	var dst [4]byte
	packXPixel(dst[:], 0x11, 0x22, 0x33, 0x44, false, 0x00ff0000, 0x0000ff00, 0x000000ff)
	if dst != [4]byte{0x33, 0x22, 0x11, 0x44} {
		t.Fatalf("lsb BGRA got %x", dst)
	}
	packXPixel(dst[:], 0x11, 0x22, 0x33, 0x44, true, 0x00ff0000, 0x0000ff00, 0x000000ff)
	if dst != [4]byte{0x44, 0x11, 0x22, 0x33} {
		t.Fatalf("msb ARGB got %x", dst)
	}
}
