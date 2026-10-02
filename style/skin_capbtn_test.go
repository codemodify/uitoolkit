package style

import (
	"math"
	"testing"
)

// A skin that draws its own caption buttons is given a button its art fits:
// the art's proportions, scaled with the skin, stood in the middle of the
// band.
//
// The frame used to size a skin's caption button as a fraction of the
// caption and let it fill the band's whole height, which is right only for
// art drawn to fill a caption. None of the shipped skins is: Deck's button
// is a disc 24x28 in a 46-pixel caption and was drawn 29x46, a tall oval,
// and Nocturne's is 56x30 and was drawn 21 wide — a third of its width.
func TestASkinsCaptionButtonKeepsItsArtsShape(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, name := range []string{"nocturne", "cassette", "deck"} {
		t.Run(name, func(t *testing.T) {
			p, ok := LoadTheme(name)
			if !ok {
				t.Skipf("%s is not in this build", name)
			}
			for _, scale := range []float32{1, 1.25, 1.75, 2} {
				lk := WithScale(p.Look(), scale)
				c, _ := lk.(*Classic)
				sk := skinFor(c)
				if sk == nil {
					t.Fatal("not a skin")
				}
				art := skinCaptionButtonArt(sk, "")
				if art == nil {
					t.Skip("this skin leaves its caption buttons to the base pack")
				}
				d := DecorationOf(lk, DecorationState{Active: true})
				if d.Button.X <= 0 || d.Button.Y <= 0 {
					t.Fatalf("@%gx: the button has no size of its own: %v", scale, d.Button)
				}
				// The shape the art was drawn at, within a pixel of rounding.
				want := art.W / art.H
				got := d.Button.X / d.Button.Y
				if math.Abs(float64(got-want)) > 0.12 {
					t.Errorf("@%gx: button %gx%g is %.2f wide for its height; the art %gx%g is %.2f",
						scale, d.Button.X, d.Button.Y, got, art.W, art.H, want)
				}
				// And it stands inside the band rather than filling it.
				if d.Button.Y > d.Caption {
					t.Errorf("@%gx: a %g button does not fit a %g caption", scale, d.Button.Y, d.Caption)
				}
				if d.ButtonPad.Top+d.Button.Y > d.Caption {
					t.Errorf("@%gx: the button is pushed out of the band: pad %g + %g > %g",
						scale, d.ButtonPad.Top, d.Button.Y, d.Caption)
				}
			}
		})
	}
}
