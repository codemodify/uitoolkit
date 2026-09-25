package settingsapp

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Settings previews a theme in a panel that stands in for a window, and a
// shaped skin's windows are not rectangles: the preview is cut to the
// skin's own silhouette, so Deck shows its shoulder and waist and Minim
// Silver its round corners there as on the desktop.
func TestThePreviewHasTheSkinsSilhouette(t *testing.T) {
	type probe struct {
		name string
		dx   float32 // from the panel's left edge (negative: from its right)
		dy   float32 // from its top (negative: from its bottom)
		cut  bool    // the silhouette leaves this point to what is behind
	}
	cases := map[string][]probe{
		// Deck's body is drawn in 18 design pixels under a full-width
		// shoulder: beside the body near the foot is outside the window,
		// the middle is inside.
		"deck": {{"beside the body", 4, -8, true}, {"beside the body, right", -4, -8, true}, {"the middle", 0.5, 0.5, false}},
		// Minim Silver is round at all four corners.
		"minim-silver": {{"the top left corner", 1, 1, true}, {"the bottom right corner", -1, -1, true}, {"the middle", 0.5, 0.5, false}},
		// A theme whose windows are rectangles paints its corners.
		"breeze-night": {{"the top left corner", 1, 1, false}, {"the bottom right corner", -1, -1, false}},
	}
	for pack, probes := range cases {
		t.Run(pack, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
			w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 1024, Height: 780, Headless: true})
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			w.SetContent(SettingsAppStaged(a, w, pack))
			a.PumpOnce()
			var preview *widgets.Panel
			widget.Walk(w.Content(), func(c widget.Component) {
				if p, ok := c.(*widgets.Panel); ok && p.Window && preview == nil {
					preview = p
				}
			})
			if preview == nil {
				t.Fatal("no preview window panel")
			}
			img := w.Capture()
			o := widget.DeviceOrigin(preview)
			b := preview.Bounds()
			// What is behind the preview: the pixel just left of it.
			bg := func(y int) [4]uint8 {
				r, g, bl, al := img.PremulAt(int(o.X)-2, y)
				return [4]uint8{r, g, bl, al}
			}
			for _, p := range probes {
				x, y := p.dx, p.dy
				switch {
				case x > 0 && x < 1:
					x *= b.Dx()
				case x < 0:
					x += b.Dx()
				}
				switch {
				case y > 0 && y < 1:
					y *= b.Dy()
				case y < 0:
					y += b.Dy()
				}
				px, py := int(o.X+x), int(o.Y+y)
				r, g, bl, al := img.PremulAt(px, py)
				behind := [4]uint8{r, g, bl, al} == bg(py)
				if behind != p.cut {
					t.Errorf("%s: %s (%d, %d) is %d,%d,%d,%d; what is behind is %v; cut=%v, want %v",
						pack, p.name, px, py, r, g, bl, al, bg(py), behind, p.cut)
				}
			}
		})
	}
}
