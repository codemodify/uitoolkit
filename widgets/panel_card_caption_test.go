package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// TestRaisedPanelKeepsFileDialogCaptionInsideCard is the win8 file dialog
// report: the dialog's title was drawn half above the card's top edge,
// clipped, because a titled Panel took the look's group box path and a
// group box hangs its title on the top border, outside the bounds it was
// given. A raised panel is a card, so its caption belongs inside it, and
// the room for it has to be in the panel's own insets — otherwise the body
// starts under the caption whatever the look draws.
func TestRaisedPanelKeepsFileDialogCaptionInsideCard(t *testing.T) {
	for _, pack := range style.ListBuiltinThemes() {
		for _, scale := range []float32{1, 1.75} {
			name := fmt.Sprintf("%s@%g", pack.Name, scale)
			t.Run(name, func(t *testing.T) {
				lk := style.WithScale(pack.Look(), scale)
				want := lk.Metrics().TitleBar
				if want <= 0 {
					t.Fatalf("%s: look has no title bar metric", name)
				}
				p := NewPanel("Open file", NewLabel("body"))
				p.Raised = true
				p.SetLook(lk)
				if got := p.insets().Top; got < want {
					t.Errorf("%s: raised panel reserves %.2f at the top, the look's caption is %.2f tall: the title is drawn outside the card", name, got, want)
				}
				// An untitled card needs no band, and a plain (not raised)
				// panel is still the look's group box.
				q := NewPanel("Group", NewLabel("body"))
				q.SetLook(lk)
				if g, ok := lk.(style.GroupBoxLook); ok {
					if q.insets() != g.GroupBoxInsets(true) {
						t.Errorf("%s: a plain titled panel no longer uses the look's group box insets", name)
					}
				}
			})
		}
	}
}

// TestRaisedPanelCaptionBandMatchesReservedInset: the band Paint draws and
// the room insets reserve are the same metric, so the body sits exactly
// below the caption at fractional scale too, with no half-pixel overlap.
func TestRaisedPanelCaptionBandMatchesReservedInset(t *testing.T) {
	for _, pack := range style.ListBuiltinThemes() {
		for _, scale := range []float32{1, 1.25, 1.75, 2} {
			lk := style.WithScale(pack.Look(), scale)
			p := NewPanel("Open file", NewLabel("body"))
			p.Raised = true
			p.SetLook(lk)
			m := lk.Metrics()
			if got, want := p.insets().Top, m.Pad+m.TitleBar; got != want {
				t.Errorf("%s @%g: inset top %.3f, caption band + pad %.3f", pack.Name, scale, got, want)
			}
		}
	}
}
