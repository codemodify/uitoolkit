package uitest

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// PaintHash is a stable FNV-64 of premul RGBA at the current scale.
// Used as a light golden so dirty-rect / glyph regressions fail CI.
func PaintHash(img *paintengine2d.Image) uint64 {
	if img == nil || img.Width < 1 || img.Height < 1 {
		return 0
	}
	h := fnv.New64a()
	var buf [4]byte
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b, a := img.PremulAt(x, y)
			buf[0], buf[1], buf[2], buf[3] = r, g, b, a
			_, _ = h.Write(buf[:])
		}
	}
	return h.Sum64()
}

// ChromeStrip paints a button + checkbox + menu row (Office XP chrome).
func ChromeStrip(scale float32) *paintengine2d.Image {
	if scale < 1 {
		scale = 1
	}
	h := NewHost()
	look := style.LookAndFeel(style.DarkLook())
	if scale > 1 {
		look = style.WithScale(look, scale)
	}
	h.SetLook(look)
	h.SetScale(scale)
	btn := widgets.NewButton("OK", nil)
	chk := widgets.NewCheckbox("Wrap", true, nil)
	mb := widgets.NewMenuBar(widgets.NewMenu("&File", widgets.Item("New", nil)))
	row := widgets.NewRow(btn, chk, mb).WithGap(12).WithPadding(8, 8, 8, 8)
	w, ht := int(320*scale), int(48*scale)
	s := MountHost(h, row, paintengine2d.XYWH(0, 0, float32(w), float32(ht)))
	return s.Paint()
}

// GoldenPNGPath is testdata/chrome-strip-<scale>x.png next to this file.
func GoldenPNGPath(scale float32) string {
	name := "chrome-strip-1x.png"
	if scale >= 1.5 {
		name = "chrome-strip-2x.png"
	}
	return filepath.Join("testdata", name)
}

// WriteGoldenPNG writes img to testdata (UITK_UPDATE_GOLDEN=1).
func WriteGoldenPNG(img *paintengine2d.Image, path string) error {
	if img == nil {
		return fmt.Errorf("nil image")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return img.WritePNGFile(path)
}

// CompareGoldenPNG reports when img differs from the stored PNG by more
// than slop (per-channel L1) on more than maxDiff pixels.
func CompareGoldenPNG(img *paintengine2d.Image, path string, slop, maxDiff int) error {
	want, err := paintengine2d.DecodePNGFile(path)
	if err != nil {
		return fmt.Errorf("golden %s: %w", path, err)
	}
	if want.Width != img.Width || want.Height != img.Height {
		return fmt.Errorf("golden size %dx%d got %dx%d", want.Width, want.Height, img.Width, img.Height)
	}
	n := ColorDiff(img, want, slop)
	if n > maxDiff {
		return fmt.Errorf("golden %s: %d pixels differ (slop=%d)", path, n, slop)
	}
	return nil
}
