package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// A tab's chrome is drawn once. Giving a tab a mark or a close button adds
// those marks and moves its word; it must not change the tab's own outline.
//
// The strip used to place an off-centre label by painting the whole tab a
// second time, shifted, and clipping to the room the label had. That assumed
// a tab's face looks the same wherever it is put. A slanted tab — NeXT's, so
// Window Maker's, and most of the older packs — is a trapezoid, and the
// second pass painted its flat body over the tab's own slope: the trailing
// edge vanished and stray diagonals were left across the strip.
//
// So: paint a strip whose tabs carry marks, paint the same strip whose tabs
// carry none, and compare every pixel that is not inside a mark or a word.
// The chrome either survived the marks or it did not.
func TestATabsMarksDoNotRepaintItsChrome(t *testing.T) {
	const W, H = 900, 48
	for _, pack := range []string{"wmaker-default", "win95", "breeze-night", "adwaita", "linear", "beos"} {
		pack := pack
		t.Run(pack, func(t *testing.T) {
			p, ok := style.LoadTheme(pack)
			if !ok {
				t.Skipf("%s is not in this build", pack)
			}
			titles := []string{"Mail", "News", "Docs"}

			build := func(marks bool) (*paintengine2d.Image, *BrowserTabs) {
				tabs := NewBrowserTabs()
				for i, title := range titles {
					tab := BrowserTab{Title: title, NoClose: !marks || i == 0}
					if marks {
						tab.Icon = style.IconMail
					}
					tabs.AddTab(tab)
				}
				tabs.Select(0)
				tabs.SetLook(p.Look())
				tabs.SetHost(&host{})
				tabs.Arrange(paintengine2d.XYWH(0, 0, W, H))
				img := paintengine2d.NewImage(W, H)
				ctx := paintengine2d.NewContext(img)
				ctx.Clear(paintengine2d.RGB(0.18, 0.2, 0.23))
				tabs.Paint(ctx)
				return img, tabs
			}

			bare, _ := build(false)
			marked, tabs := build(true)

			// Compare the tab's edges — the zone its outline is drawn in, at
			// each end and along the top and foot. A word never reaches
			// there: the label box excludes the ends, and the title is
			// elided to fit it. Everything in between is where the words
			// legitimately differ between the two strips, so it is left out.
			g := tabs.geom()
			var zones []paintengine2d.Rect
			for i := range titles {
				s := tabs.slotOf(i, g)
				end := max(style.TabContentInsetOf(p.Look(), s), 3)
				zones = append(zones,
					paintengine2d.XYWH(s.Min.X, s.Min.Y, end, s.Dy()),
					paintengine2d.XYWH(s.Max.X-end, s.Min.Y, end, s.Dy()),
					paintengine2d.XYWH(s.Min.X, s.Min.Y, s.Dx(), 2),
					paintengine2d.XYWH(s.Min.X, s.Max.Y-2, s.Dx(), 2))
			}
			inZone := func(x, y int) bool {
				pt := paintengine2d.Pt(float32(x)+0.5, float32(y)+0.5)
				for _, r := range zones {
					if !r.Empty() && pt.X >= r.Min.X && pt.X < r.Max.X && pt.Y >= r.Min.Y && pt.Y < r.Max.Y {
						return true
					}
				}
				return false
			}

			diff := 0
			for y := 0; y < H; y++ {
				for x := 0; x < W; x++ {
					if !inZone(x, y) {
						continue
					}
					ar, ag, ab, aa := bare.PremulAt(x, y)
					br, bg, bb, ba := marked.PremulAt(x, y)
					if ar != br || ag != bg || ab != bb || aa != ba {
						diff++
					}
				}
			}
			if diff > 0 {
				t.Errorf("%d pixels of chrome changed when the tabs were given marks", diff)
			}
		})
	}
}
