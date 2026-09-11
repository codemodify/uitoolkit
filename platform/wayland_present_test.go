package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestParseWaylandPresent(t *testing.T) {
	cases := map[string]string{
		"":         WaylandPresentAuto,
		"auto":     WaylandPresentAuto,
		"AUTO":     WaylandPresentAuto,
		"shm":      WaylandPresentSHM,
		"SHM":      WaylandPresentSHM,
		"dmabuf":   WaylandPresentDmabuf,
		" DMABuf ": WaylandPresentDmabuf,
		"other":    WaylandPresentAuto,
	}
	for in, want := range cases {
		if got := parseWaylandPresent(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestWaylandPresentPrefEnv(t *testing.T) {
	t.Setenv(EnvWaylandPresent, "shm")
	if WaylandPresentPref() != WaylandPresentSHM {
		t.Fatal(WaylandPresentPref())
	}
	t.Setenv(EnvWaylandPresent, "")
	if WaylandPresentPref() != WaylandPresentAuto {
		t.Fatal(WaylandPresentPref())
	}
}

func TestDRMFourcc(t *testing.T) {
	if drmFormatARGB8888 != 0x34325241 {
		t.Fatalf("ARGB8888 %#x", drmFormatARGB8888)
	}
	if drmFormatXRGB8888 != 0x34325258 {
		t.Fatalf("XRGB8888 %#x", drmFormatXRGB8888)
	}
	if !dmabufFormatUsable(drmFormatARGB8888) || !dmabufFormatUsable(drmFormatXRGB8888) {
		t.Fatal("usable")
	}
	if !dmabufNeedsSwizzle(drmFormatARGB8888) || dmabufNeedsSwizzle(drmFormatABGR8888) {
		t.Fatal("swizzle")
	}
	if !modifierCPUFriendly(drmModLinear) || !modifierCPUFriendly(drmModInvalid) {
		t.Fatal("mod")
	}
	if modifierCPUFriendly(1) {
		t.Fatal("tiled")
	}
}

func TestPickDmabufFormat(t *testing.T) {
	_, _, ok := pickDmabufFormat(nil)
	if ok {
		t.Fatal("empty")
	}
	fmt, mod, ok := pickDmabufFormat([]dmaFmtMod{
		{drmFormatNV12ish(), 0},
		{drmFormatARGB8888, 0x200000000000001},
		{drmFormatARGB8888, drmModLinear},
		{drmFormatXRGB8888, drmModInvalid},
	})
	if !ok || fmt != drmFormatXRGB8888 || mod != drmModInvalid {
		t.Fatalf("prefer opaque XRGB, got %#x %#x ok=%v", fmt, mod, ok)
	}
	fmt, mod, ok = pickDmabufFormat([]dmaFmtMod{
		{drmFormatARGB8888, drmModLinear},
		{drmFormatXRGB8888, drmModLinear},
	})
	if !ok || fmt != drmFormatXRGB8888 || mod != drmModLinear {
		t.Fatalf("xrgb-over-argb %#x %#x", fmt, mod)
	}
	fmt, mod, ok = pickDmabufFormat([]dmaFmtMod{{drmFormatXRGB8888, drmModInvalid}})
	if !ok || fmt != drmFormatXRGB8888 || mod != drmModInvalid {
		t.Fatalf("xrgb %#x %#x", fmt, mod)
	}
	_, _, ok = pickDmabufFormat([]dmaFmtMod{{drmFormatARGB8888, 0x200000000000001}})
	if ok {
		t.Fatal("tiled-only should be rejected")
	}
	fmt, mod, ok = pickDmabufFormat([]dmaFmtMod{{drmFormatARGB8888, drmModLinear}})
	if !ok || fmt != drmFormatARGB8888 || mod != drmModLinear {
		t.Fatalf("argb-only %#x %#x", fmt, mod)
	}
	if dmabufFormatOpaque(drmFormatXRGB8888) == dmabufFormatOpaque(drmFormatARGB8888) {
		t.Fatal("opaque fourcc")
	}
}

func TestWaylandAutoIsSHM(t *testing.T) {
	t.Setenv(EnvWaylandPresent, "")
	if waylandWantDmabuf() {
		t.Fatal("auto must not opt into dmabuf")
	}
	t.Setenv(EnvWaylandPresent, WaylandPresentAuto)
	if waylandWantDmabuf() {
		t.Fatal("auto")
	}
	t.Setenv(EnvWaylandPresent, WaylandPresentSHM)
	if waylandWantDmabuf() {
		t.Fatal("shm")
	}
	t.Setenv(EnvWaylandPresent, WaylandPresentDmabuf)
	if !waylandWantDmabuf() {
		t.Fatal("dmabuf opt-in")
	}
}

func drmFormatNV12ish() uint32 { return 0x3231564e }

func TestImageRawAndCopyRect(t *testing.T) {
	img := paintengine2d.NewImage(4, 2)
	pix, w, h, stride := imageRaw(img)
	if w != 4 || h != 2 || stride != 16 || len(pix) < 32 {
		t.Fatalf("raw %d×%d stride=%d len=%d", w, h, stride, len(pix))
	}
	// premul RGBA red
	img.Pix[0], img.Pix[1], img.Pix[2], img.Pix[3] = 200, 10, 20, 220
	img.Pix[16], img.Pix[17], img.Pix[18], img.Pix[19] = 1, 2, 3, 4

	dst := make([]byte, 2*16)
	copyImageRect(dst, 16, img, paintengine2d.XYWH(0, 0, 4, 2), true, false)
	if dst[0] != 20 || dst[1] != 10 || dst[2] != 200 || dst[3] != 220 {
		t.Fatalf("swizzle %+v", dst[:4])
	}
	if dst[16] != 3 || dst[18] != 1 {
		t.Fatalf("row1 %+v", dst[16:20])
	}

	dst2 := make([]byte, 2*16)
	copyImageRect(dst2, 16, img, paintengine2d.XYWH(0, 0, 1, 1), false, false)
	if dst2[0] != 200 || dst2[2] != 20 {
		t.Fatalf("memcpy %+v", dst2[:4])
	}
}

func TestCopyImageRectPaddedStride(t *testing.T) {
	img := paintengine2d.NewImage(2, 2)
	img.Pix[0], img.Pix[1], img.Pix[2], img.Pix[3] = 9, 8, 7, 6
	dst := make([]byte, 2*32)
	copyImageRect(dst, 32, img, paintengine2d.XYWH(0, 0, 2, 1), true, false)
	if dst[0] != 7 || dst[2] != 9 {
		t.Fatalf("%+v", dst[:4])
	}
	// padding after the two pixels must stay zero
	if dst[8] != 0 {
		t.Fatal("wrote into padding")
	}
}

func TestCopyImageRectOpaqueUI(t *testing.T) {
	img := paintengine2d.NewImage(8, 4)
	ctx := paintengine2d.NewContext(img)
	// Known-opaque UI fill: premul RGBA with A=255. After the Wayland
	// present swizzle every destination alpha/padding byte must be 0xFF
	// (XRGB8888 / forced-opaque ARGB), otherwise Mutter composites a
	// fully transparent window.
	ctx.Clear(paintengine2d.RGB(0.20, 0.40, 0.80))
	if !pixmapHasOpaque(img) {
		t.Fatal("source should be opaque")
	}
	for _, swizzle := range []bool{true, false} {
		dst := make([]byte, 8*4*4)
		copyImageRect(dst, 32, img, paintengine2d.XYWH(0, 0, 8, 4), swizzle, true)
		if destAlphaAllZero(dst, len(dst)) {
			t.Fatalf("swizzle=%v dest alpha all zero", swizzle)
		}
		for i := 3; i < len(dst); i += 4 {
			if dst[i] != 0xff {
				t.Fatalf("swizzle=%v dest[%d] alpha=%d (want 0xff)", swizzle, i, dst[i])
			}
		}
		// Damage-only: a 2×2 box must not touch the rest of the buffer.
		partial := make([]byte, 8*4*4)
		copyImageRect(partial, 32, img, paintengine2d.XYWH(1, 1, 2, 2), swizzle, true)
		if destAlphaAllZero(partial[1*32+1*4:], 8) {
			t.Fatalf("swizzle=%v damaged pixels blank", swizzle)
		}
		if partial[0] != 0 || partial[3] != 0 {
			t.Fatalf("swizzle=%v copied outside damage", swizzle)
		}
	}
	// XRGB swizzle of opaque red-ish: B,G,R,0xFF
	r, g, b, a := img.Pix[0], img.Pix[1], img.Pix[2], img.Pix[3]
	if a == 0 {
		t.Fatal("clear alpha")
	}
	one := make([]byte, 4)
	copyImageRect(one, 4, img, paintengine2d.XYWH(0, 0, 1, 1), true, true)
	if one[0] != b || one[1] != g || one[2] != r || one[3] != 0xff {
		t.Fatalf("XRGB swizzle got %v want BGRA(%d,%d,%d,255)", one, b, g, r)
	}
}

func TestCopyPresentRowMatchesBytes(t *testing.T) {
	src := []byte{10, 20, 30, 40, 50, 60, 70, 80}
	var got, want [8]byte
	copyPresentRow(got[:], src, true, false)
	for i := 0; i < 8; i += 4 {
		want[i+0], want[i+1], want[i+2], want[i+3] = src[i+2], src[i+1], src[i+0], src[i+3]
	}
	if got != want {
		t.Fatalf("swizzle got %v want %v", got, want)
	}
	copyPresentRow(got[:], src, true, true)
	if got[3] != 0xff || got[7] != 0xff || got[0] != 30 || got[2] != 10 {
		t.Fatalf("opaque %v", got)
	}
}

func BenchmarkCopyImageRectOpaque(b *testing.B) {
	img := paintengine2d.NewImage(1000, 760)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.RGB(0.2, 0.4, 0.8))
	dst := make([]byte, 1000*760*4)
	r := paintengine2d.XYWH(0, 0, 1000, 760)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copyImageRect(dst, 4000, img, r, true, true)
	}
}
