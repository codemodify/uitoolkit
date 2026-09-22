package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The palette's requests: none without KWin's global or for the palette
// already sent; the object once, when there is first a palette to send;
// going back to the desktop's colours is set_palette("") on it.
func TestPalettePlan(t *testing.T) {
	for _, c := range []struct {
		name         string
		global, have bool
		sent, want   string
		plan         palettePlan
	}{
		{"no global", false, false, "", "/c/a.colors", palettePlan{}},
		{"no global, even with an object", false, true, "/c/a.colors", "/c/b.colors", palettePlan{}},
		{"first palette", true, false, "", "/c/a.colors", palettePlan{create: true, set: true}},
		{"same palette", true, true, "/c/a.colors", "/c/a.colors", palettePlan{}},
		{"another palette", true, true, "/c/a.colors", "/c/b.colors", palettePlan{set: true}},
		{"back to the desktop's", true, true, "/c/a.colors", "", palettePlan{set: true}},
		{"nothing to go back from", true, false, "", "", palettePlan{}},
	} {
		if got := planPalette(c.global, c.have, c.sent, c.want); got != c.plan {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.plan)
		}
	}
}

// iconTestImage is a side×side icon whose pixel (x, y) is
// (x*40, y*40, 200, alpha) straight — alpha 255 on the first row, 128 on
// the rest.
func iconTestImage(side int) *paintengine2d.Image {
	im := paintengine2d.NewImage(side, side)
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			a := float32(1)
			if y > 0 {
				a = 128.0 / 255
			}
			im.SetColor(x, y, paintengine2d.RGBA(float32(x*40)/255, float32(y*40)/255, 200.0/255, a))
		}
	}
	return im
}

// _NET_WM_ICON: width, height, then the pixels row by row as ARGB with alpha
// in the high byte, not premultiplied; images smallest first, and anything
// not square left out.
func TestNetWMIconEncoding(t *testing.T) {
	two, three := iconTestImage(2), iconTestImage(3)
	got := netWMIcon([]*paintengine2d.Image{three, paintengine2d.NewImage(4, 2), two, iconTestImage(2)})
	want := []uint32{2, 2}
	for _, im := range []*paintengine2d.Image{two, three} {
		if im == three {
			want = append(want, 3, 3)
		}
		for y := 0; y < im.Height; y++ {
			for x := 0; x < im.Width; x++ {
				p := im.NRGBAAt(x, y)
				want = append(want, uint32(p.A)<<24|uint32(p.R)<<16|uint32(p.G)<<8|uint32(p.B))
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%d cardinals, want %d (2x2 and 3x3 with their sizes)", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cardinal %d: %#08x, want %#08x", i, got[i], want[i])
		}
	}
	// Straight, not premultiplied: the half-transparent row keeps its
	// colour.
	p := three.NRGBAAt(2, 1)
	// (The image keeps its pixels premultiplied, so blue comes back within
	// a step of 200.)
	if px := got[2+2*2+2+1*3+2]; px>>24 != uint32(p.A) || (px>>16)&0xff != uint32(p.R) || px&0xff < 198 {
		t.Fatalf("pixel (2,1) of the 3x3: %#08x, want alpha %d red %d blue about 200", px, p.A, p.R)
	}
}

// A wl_shm ARGB8888 icon buffer: little-endian pixels, blue first and alpha
// last in memory, premultiplied, rows packed.
func TestWaylandIconPixels(t *testing.T) {
	im := iconTestImage(3)
	b := wlIconPixels(im)
	if len(b) != 3*3*4 {
		t.Fatalf("%d bytes, want %d", len(b), 3*3*4)
	}
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			r, g, bl, a := im.PremulAt(x, y)
			i := (y*3 + x) * 4
			if b[i] != bl || b[i+1] != g || b[i+2] != r || b[i+3] != a {
				t.Fatalf("pixel (%d,%d): % x, want premultiplied b g r a % x", x, y, b[i:i+4], []byte{bl, g, r, a})
			}
		}
	}
	// Premultiplied: blue 200 at half alpha is about 100.
	if v := b[(1*3+0)*4]; v < 95 || v > 105 {
		t.Fatalf("blue at alpha 128: %d, want about 100 (premultiplied)", v)
	}
}
