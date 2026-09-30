package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Every shipped pack's status colours must be readable *as text* on its
// own background. Measured before this existed: 225 of the 405 pairs
// were not, the worst near 1.6:1 — a colour you can see is there and
// cannot read.
func TestEveryPackHasReadableStatusInk(t *testing.T) {
	packs := ListThemes()
	if len(packs) == 0 {
		t.Skip("no packs in this build")
	}
	for _, p := range packs {
		lk := Appearance{Name: p.Name, Theme: p.Palette}.Look()
		pal := lk.Palette()
		for _, tc := range []struct {
			role string
			ink  paintengine2d.Color
		}{
			{"danger", pal.DangerInk()},
			{"success", pal.SuccessInk()},
			{"warning", pal.WarningInk()},
		} {
			if got := ContrastRatio(tc.ink, pal.Background); got < MinInkContrast {
				t.Errorf("%s: %s ink is %.2f:1 on the background, want %.1f",
					p.Name, tc.role, got, MinInkContrast)
			}
		}
	}
}

// The declared colours are era-faithful on purpose — Clearlooks really
// did use #c4a000 — and Ink must not write them back into the palette.
// A fill is still the pack's own colour.
func TestInkDoesNotChangeTheDeclaredPalette(t *testing.T) {
	packs := ListThemes()
	if len(packs) == 0 {
		t.Skip("no packs in this build")
	}
	for _, p := range packs {
		lk := Appearance{Name: p.Name, Theme: p.Palette}.Look()
		pal := lk.Palette()
		before := [3]paintengine2d.Color{pal.Danger, pal.Success, pal.Warning}
		_, _, _ = pal.DangerInk(), pal.SuccessInk(), pal.WarningInk()
		after := [3]paintengine2d.Color{pal.Danger, pal.Success, pal.Warning}
		if before != after {
			t.Fatalf("%s: asking for the ink changed the palette", p.Name)
		}
	}
}

// A colour that already reads is handed back untouched, so a pack that
// chose well keeps exactly what it chose.
func TestInkLeavesReadableColoursAlone(t *testing.T) {
	bg := paintengine2d.RGB(1, 1, 1)
	for _, c := range []paintengine2d.Color{
		paintengine2d.RGB(0, 0, 0),
		paintengine2d.RGB(0.5, 0, 0),
		paintengine2d.RGB(0, 0.25, 0),
	} {
		if ContrastRatio(c, bg) < MinInkContrast {
			t.Fatalf("test setup: %v does not read on white", c)
		}
		if got := ReadableInk(c, bg); got != c {
			t.Errorf("ReadableInk(%v, white) = %v, want it untouched", c, got)
		}
	}
}

// Hue survives the lift: a pack's red stays that pack's red rather than
// becoming a generic one, which is the whole reason for moving lightness
// instead of mixing toward black.
func TestInkKeepsTheHue(t *testing.T) {
	for _, tc := range []struct {
		name string
		want paintengine2d.Color
		bg   paintengine2d.Color
	}{
		{"clearlooks warning on light", Hex("#c4a000"), Hex("#ededed")},
		{"a dark pack's red", Hex("#8b0000"), Hex("#2b2b2e")},
		{"a washed green on white", Hex("#7ad37a"), Hex("#ffffff")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ReadableInk(tc.want, tc.bg)
			if r := ContrastRatio(got, tc.bg); r < MinInkContrast {
				t.Fatalf("ink %v is %.2f:1, want %.1f", got, r, MinInkContrast)
			}
			wh, _, _ := flatHSL(tc.want)
			gh, _, _ := flatHSL(got)
			if d := wh - gh; d > 1 || d < -1 {
				t.Errorf("hue moved from %.1f to %.1f", wh, gh)
			}
		})
	}
}

// Alpha is carried through: a translucent status colour stays
// translucent.
func TestInkKeepsAlpha(t *testing.T) {
	c := paintengine2d.Color{R: 0.9, G: 0.85, B: 0.1, A: 0.5}
	if got := ReadableInk(c, paintengine2d.RGB(1, 1, 1)); got.A != 0.5 {
		t.Errorf("alpha %v, want 0.5", got.A)
	}
}

// A mid-grey background is the hard case. What matters is that the
// result can be read — the first candidate that clears the bar wins,
// because keeping the colour is the point and a colour at 4.5:1 is not
// improved by being blacker.
//
// Only where no lightness on the hue reaches the bar at all does it fall
// back to black or white: an unreadable coloured word is worse than a
// readable plain one.
func TestInkOnAHopelessBackground(t *testing.T) {
	bg := paintengine2d.RGB(0.45, 0.45, 0.45)
	got := ReadableInk(Hex("#c4a000"), bg)
	if r := ContrastRatio(got, bg); r < MinInkContrast {
		// It could not reach the bar, so it must have taken the better
		// of black and white.
		best, which := ContrastRatio(paintengine2d.RGB(0, 0, 0), bg), "black"
		if w := ContrastRatio(paintengine2d.RGB(1, 1, 1), bg); w > best {
			best, which = w, "white"
		}
		if r < best-0.01 {
			t.Errorf("ink is %.2f:1 and did not fall back to %s at %.2f:1", r, which, best)
		}
		return
	}
	// It cleared the bar, so it must still be the colour it started as.
	if _, s, _ := flatHSL(got); s < 20 {
		t.Errorf("ink %v cleared the bar but lost its colour", got)
	}
}

// The ink must keep the pack's colour, not merely reach the contrast
// ratio. This is the test the first version of this file needed and did
// not have: the walk passed lightness as a fraction where flatFromHSL
// takes percent, so every candidate sat within one percent of black. On
// a light background the first already cleared 4.5:1 and the ink came
// out all but black; on a dark one none did, and the fallback made it
// white. Every pack's danger, warning and success ink was black or
// white — and the contrast test passed, because black on light and white
// on dark both read perfectly well.
//
// So contrast is not the property. Keeping the colour is, and that is
// what this asserts: a saturated status colour stays saturated.
func TestStatusInkKeepsThePacksColour(t *testing.T) {
	packs := ListThemes()
	if len(packs) == 0 {
		t.Skip("no packs in this build")
	}
	greyed := 0
	for _, p := range packs {
		lk := Appearance{Name: p.Name, Theme: p.Palette}.Look()
		pal := lk.Palette()
		for _, tc := range []struct {
			role string
			want paintengine2d.Color
			ink  paintengine2d.Color
		}{
			{"danger", pal.Danger, pal.DangerInk()},
			{"success", pal.Success, pal.SuccessInk()},
			{"warning", pal.Warning, pal.WarningInk()},
		} {
			_, ws, _ := flatHSL(tc.want)
			_, is, _ := flatHSL(tc.ink)
			if ws < 20 {
				continue // the pack chose a grey; keeping it grey is right
			}
			// A colour that had some in it must not come back grey.
			if is < ws*0.5 {
				greyed++
				t.Errorf("%s %s: %v (s=%.0f%%) came back as %v (s=%.0f%%) — the colour was lost",
					p.Name, tc.role, tc.want, ws, tc.ink, is)
			}
		}
	}
	if greyed > 0 {
		t.Logf("%d of %d status colours lost their hue", greyed, len(packs)*3)
	}
}

// And the two ends: nothing comes back pure black or pure white where
// the pack asked for a colour and one was available.
func TestStatusInkIsNotBlackOrWhite(t *testing.T) {
	packs := ListThemes()
	if len(packs) == 0 {
		t.Skip("no packs in this build")
	}
	black := paintengine2d.RGB(0, 0, 0)
	white := paintengine2d.RGB(1, 1, 1)
	for _, p := range packs {
		lk := Appearance{Name: p.Name, Theme: p.Palette}.Look()
		pal := lk.Palette()
		for _, tc := range []struct {
			role string
			want paintengine2d.Color
			ink  paintengine2d.Color
		}{
			{"danger", pal.Danger, pal.DangerInk()},
			{"success", pal.Success, pal.SuccessInk()},
			{"warning", pal.Warning, pal.WarningInk()},
		} {
			if _, s, _ := flatHSL(tc.want); s < 20 {
				continue
			}
			for _, plain := range []paintengine2d.Color{black, white} {
				if tc.ink.R == plain.R && tc.ink.G == plain.G && tc.ink.B == plain.B {
					t.Errorf("%s %s: %v came back as %v", p.Name, tc.role, tc.want, tc.ink)
				}
			}
		}
	}
}
