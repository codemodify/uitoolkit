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

func TestPackXPixel16RGB565(t *testing.T) {
	// A 16-bpp TrueColor visual: red 0xf800, green 0x07e0, blue 0x001f.
	var dst [2]byte
	packXPixelN(dst[:], 2, 0xff, 0x00, 0x00, 0xff, false, 0xf800, 0x07e0, 0x001f)
	if got := uint16(dst[0]) | uint16(dst[1])<<8; got != 0xf800 {
		t.Fatalf("red = %#04x, want 0xf800", got)
	}
	packXPixelN(dst[:], 2, 0x00, 0xff, 0x00, 0xff, false, 0xf800, 0x07e0, 0x001f)
	if got := uint16(dst[0]) | uint16(dst[1])<<8; got != 0x07e0 {
		t.Fatalf("green = %#04x, want 0x07e0", got)
	}
	packXPixelN(dst[:], 2, 0xff, 0xff, 0xff, 0xff, false, 0xf800, 0x07e0, 0x001f)
	if got := uint16(dst[0]) | uint16(dst[1])<<8; got != 0xffff {
		t.Fatalf("white = %#04x, want 0xffff", got)
	}
	// MSB-first server byte order.
	packXPixelN(dst[:], 2, 0xff, 0x00, 0x00, 0xff, true, 0xf800, 0x07e0, 0x001f)
	if dst[0] != 0xf8 || dst[1] != 0x00 {
		t.Fatalf("msb red = %#02x%02x, want f800", dst[0], dst[1])
	}
}

func TestPackXPixelNShortBuffer(t *testing.T) {
	var one [1]byte
	packXPixelN(one[:], 2, 1, 2, 3, 4, false, 0xf800, 0x07e0, 0x001f) // must not panic
	var three [3]byte
	packXPixelN(three[:], 4, 1, 2, 3, 4, false, 0xff0000, 0xff00, 0xff)
	packXPixelN(three[:], 3, 1, 2, 3, 4, false, 0xff0000, 0xff00, 0xff) // unsupported size
}

func TestMaskWidth(t *testing.T) {
	cases := map[uint32]uint{0xf800: 5, 0x07e0: 6, 0x001f: 5, 0xff0000: 8, 0: 8}
	for mask, want := range cases {
		if got := maskWidth(mask); got != want {
			t.Fatalf("maskWidth(%#x) = %d, want %d", mask, got, want)
		}
	}
}
