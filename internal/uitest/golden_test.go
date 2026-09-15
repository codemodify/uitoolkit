package uitest

import (
	"os"
	"testing"
)

// Locked hashes for the chrome strip. Recomputed when UITK_UPDATE_GOLDEN=1
// prints the new values — do not silently accept a dirty-rect glyph change.
var chromeStripHash = map[float32]uint64{
	1: 0xfde69ea87b60058,
	2: 0x39c1284b414d83e1,
}

func TestChromeStripGolden(t *testing.T) {
	update := os.Getenv("UITK_UPDATE_GOLDEN") != ""
	for _, scale := range []float32{1, 2} {
		img := ChromeStrip(scale)
		if img == nil || img.Width < 8 {
			t.Fatalf("scale %v: empty strip", scale)
		}
		sum := PaintHash(img)
		path := GoldenPNGPath(scale)
		if update {
			if err := WriteGoldenPNG(img, path); err != nil {
				t.Fatalf("scale %v write: %v", scale, err)
			}
			t.Logf("UITK_UPDATE_GOLDEN scale=%v hash=0x%x png=%s", scale, sum, path)
			continue
		}
		if sum == 0 {
			t.Fatalf("scale %v: empty hash", scale)
		}
		if want := chromeStripHash[scale]; want != 0 && sum != want {
			t.Fatalf("scale %v hash 0x%x want 0x%x (dirty-rect / glyph regression? UITK_UPDATE_GOLDEN=1)", scale, sum, want)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("scale %v missing %s (run UITK_UPDATE_GOLDEN=1)", scale, path)
		}
		if err := CompareGoldenPNG(img, path, 6, 12); err != nil {
			t.Fatal(err)
		}
	}
}
