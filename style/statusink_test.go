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

// A mid-grey background leaves nowhere for a colour to go. An unreadable
// coloured word is worse than a readable plain one, so it falls back
// rather than returning something that still cannot be read.
func TestInkOnAHopelessBackground(t *testing.T) {
	bg := paintengine2d.RGB(0.45, 0.45, 0.45)
	got := ReadableInk(Hex("#c4a000"), bg)
	best := ContrastRatio(paintengine2d.RGB(0, 0, 0), bg)
	if w := ContrastRatio(paintengine2d.RGB(1, 1, 1), bg); w > best {
		best = w
	}
	if r := ContrastRatio(got, bg); r < best-0.01 {
		t.Errorf("ink is %.2f:1 where black or white would be %.2f:1", r, best)
	}
}
