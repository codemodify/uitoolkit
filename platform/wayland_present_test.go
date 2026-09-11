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
	if !ok || fmt != drmFormatARGB8888 || mod != drmModLinear {
		t.Fatalf("got %#x %#x ok=%v", fmt, mod, ok)
	}
	fmt, mod, ok = pickDmabufFormat([]dmaFmtMod{{drmFormatXRGB8888, drmModInvalid}})
	if !ok || fmt != drmFormatXRGB8888 || mod != drmModInvalid {
		t.Fatalf("xrgb %#x %#x", fmt, mod)
	}
	_, _, ok = pickDmabufFormat([]dmaFmtMod{{drmFormatARGB8888, 0x200000000000001}})
	if ok {
		t.Fatal("tiled-only should be rejected")
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
	copyImageRect(dst, 16, img, paintengine2d.XYWH(0, 0, 4, 2), true)
	if dst[0] != 20 || dst[1] != 10 || dst[2] != 200 || dst[3] != 220 {
		t.Fatalf("swizzle %+v", dst[:4])
	}
	if dst[16] != 3 || dst[18] != 1 {
		t.Fatalf("row1 %+v", dst[16:20])
	}

	dst2 := make([]byte, 2*16)
	copyImageRect(dst2, 16, img, paintengine2d.XYWH(0, 0, 1, 1), false)
	if dst2[0] != 200 || dst2[2] != 20 {
		t.Fatalf("memcpy %+v", dst2[:4])
	}
}

func TestCopyImageRectPaddedStride(t *testing.T) {
	img := paintengine2d.NewImage(2, 2)
	img.Pix[0], img.Pix[1], img.Pix[2], img.Pix[3] = 9, 8, 7, 6
	dst := make([]byte, 2*32)
	copyImageRect(dst, 32, img, paintengine2d.XYWH(0, 0, 2, 1), true)
	if dst[0] != 7 || dst[2] != 9 {
		t.Fatalf("%+v", dst[:4])
	}
	// padding after the two pixels must stay zero
	if dst[8] != 0 {
		t.Fatal("wrote into padding")
	}
}
