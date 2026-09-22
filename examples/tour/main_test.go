package main

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/style"
)

func pngSize(t *testing.T, path string) (int, int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width, cfg.Height
}

func devicePx(v int, s float32) int { return int(math.Round(float64(float32(v) * s))) }

// -shot draws the same 1180 x 820 page at every scale: window sizes are
// logical, so at 1.75 the still is 2065 x 1435 device pixels — the page
// drawn finer, not a page 1.75 times bigger (1180*1.75 logical, which is
// what asking in device pixels gave). -sheet puts the pages on one picture,
// each at half its still.
func TestShotsAreThePageDesignAtEveryScale(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	tabs, shapes := demo.TourPageIndex("tabs"), demo.TourPageIndex("shapes")
	for _, s := range []float32{1, 1.75} {
		dir := t.TempDir()
		a := uitoolkit.New(uitoolkit.Options{Headless: true, Scale: s})
		if err := writeShots(a, dir, []int{tabs, shapes}); err != nil {
			t.Fatal(err)
		}
		wantW, wantH := devicePx(shotW, s), devicePx(shotH, s)
		for _, name := range []string{"tour-tabs.png", "tour-shapes.png"} {
			if w, h := pngSize(t, filepath.Join(dir, name)); w != wantW || h != wantH {
				t.Errorf("%.2fx: %s is %dx%d, want %dx%d", s, name, w, h, wantW, wantH)
			}
		}

		sheet := filepath.Join(dir, "sheet.png")
		if err := writeSheet(a, sheet, []int{tabs, shapes}); err != nil {
			t.Fatal(err)
		}
		w, h := pngSize(t, sheet)
		// Two half-size pages side by side, and the margins and gaps round
		// them — at most three gaps' worth of 18 logical pixels each, and
		// a little for rounding.
		pages := 2 * (wantW / 2)
		if w < pages || w > pages+devicePx(3*18, s)+4 {
			t.Errorf("%.2fx: the sheet is %d wide for two pages of %d", s, w, wantW/2)
		}
		if h < wantH/2 || h > wantH {
			t.Errorf("%.2fx: the sheet is %d high for one row of pages %d high", s, h, wantH/2)
		}
	}
}
