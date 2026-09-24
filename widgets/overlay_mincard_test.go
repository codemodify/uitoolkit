package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// TestOverlayCardFloorsAreScaled: the card's smallest size is compared
// against a size measured in device pixels, so the floors have to be
// scaled too. Unscaled, the file dialog's 520x380 is a cramped card on a
// HiDPI desktop and an oversized one at 1x.
func TestOverlayCardFloorsAreScaled(t *testing.T) {
	for _, scale := range []float32{1, 1.75, 2} {
		lk := style.WithScale(style.DarkLook(), scale)
		t.Run("mincard", func(t *testing.T) {
			ov := NewOverlay(NewPanel("Open file", NewLabel("x")))
			ov.MinCardW, ov.MinCardH = 520, 380
			ov.SetLook(lk)
			ov.Arrange(paintengine2d.XYWH(0, 0, 4000, 3000))
			b := ov.Card.Bounds()
			if wantW, wantH := style.Dip(lk, 520), style.Dip(lk, 380); b.Dx() < wantW-0.01 || b.Dy() < wantH-0.01 {
				t.Errorf("@%g: card %.1fx%.1f, floor %.1fx%.1f", scale, b.Dx(), b.Dy(), wantW, wantH)
			}
		})
		t.Run("stock", func(t *testing.T) {
			ov := NewOverlay(NewPanel("", NewLabel("x")))
			ov.SetLook(lk)
			ov.Arrange(paintengine2d.XYWH(0, 0, 4000, 3000))
			b := ov.Card.Bounds()
			if wantW, wantH := style.Dip(lk, 280), style.Dip(lk, 140); b.Dx() < wantW-0.01 || b.Dy() < wantH-0.01 {
				t.Errorf("@%g: card %.1fx%.1f, stock floor %.1fx%.1f", scale, b.Dx(), b.Dy(), wantW, wantH)
			}
		})
	}
}
