package settingsapp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The page is one column of choices, one block of settings and one
// preview of what they make, and whether
// that works is a question about pixels: the column has to stay readable
// and the preview has to stay the biggest thing on the page at the
// smallest window Settings opens to and at a fractional display scale.
// This builds it everywhere it has to hold — four packs, two scales, the
// default window and the 720x520 minimum — which fails if any of them
// cannot be laid out at all, and writes the stills when UITK_SHOTS names
// a directory, because the rest of the question is answered by looking:
//
//	UITK_SHOTS=/tmp/shots tools/testenv.sh go test ./cmd/uitoolkit-settings/... -run PageHolds
//
// UITK_PAGE opens them at a section (-page's names); nothing is below
// the fold any more, so it changes nothing about what is drawn, but the
// shots are taken through the same path the flag takes.
func TestSettingsPageHoldsAtEverySize(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("UITK_SHOTS")
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	page := os.Getenv("UITK_PAGE")
	for _, pack := range []string{"win95", "breeze", "adwaita", "breeze-night"} {
		for _, scale := range []float32{1, 1.75} {
			for _, size := range [][2]int{{1024, 860}, {720, 520}} {
				a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: scale, DisableLookWatch: true})
				w, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: size[0], Height: size[1], Headless: true})
				if err != nil {
					t.Fatal(err)
				}
				w.SetContent(SettingsAppOpen(a, w, pack, page))
				a.PumpOnce()
				// The preview is far the biggest thing on its side of the
				// splitter at every one of these sizes: that is the
				// promise the icon strip was moved out of this pane to
				// keep, and the reason the settings over it are one
				// folding row rather than a stack of switches with a line
				// of prose each. It is 82% of the pane at 1024x860 and
				// 62% at the 720x520 minimum, where the row folds onto
				// three lines.
				//
				// It went up, not down, when three choosers came out
				// of the preview and onto this page: the line they cost
				// the pane is 30 px and the group box the three paths
				// gave up under it was 34. Two rows of settings instead
				// of one folding row would have cost a fourth line at the
				// minimum and left the preview 53%, which is the whole
				// reason they are one row.
				split := settingsSplit(t, w)
				pane := split.PaneB()
				box := previewScope(t, w).LocalBounds()
				if box.Dy() < pane.Dy()*0.6 {
					t.Errorf("%s at %g×, %dx%d: the preview is %v of a %v pane",
						pack, scale, size[0], size[1], box.Dy(), pane.Dy())
				}
				// Everything in that block is whole at every one of these
				// sizes. A check box does not elide and the row does not
				// shed: it folds, and what it must never do is hand one
				// of them a width its word does not fit in or push one
				// past the pane's right edge. The three words in front of
				// the choosers are promised now, which they were not on
				// the bar inside the preview — that bar shed them from
				// the right and at this minimum dropped all three; a
				// wrapping row keeps what it carries and takes a line
				// instead.
				row := optionsRow(t, w)
				for _, word := range []string{"Animations", "OS colors", "OS open/save dialogs", "OS window borders"} {
					box := findOption(w.Content(), word)
					if box == nil {
						t.Fatalf("%s at %g×, %dx%d: no %q option", pack, scale, size[0], size[1], word)
					}
					if box.Bounds().Dx() < box.Measure(layout.Unbounded()).X-0.51 {
						t.Errorf("%s at %g×, %dx%d: the %q option is squeezed to %v of %v",
							pack, scale, size[0], size[1], word, box.Bounds().Dx(), box.Measure(layout.Unbounded()).X)
					}
					if box.Bounds().Max.X > row.LocalBounds().Dx()+0.51 {
						t.Errorf("%s at %g×, %dx%d: the %q option ends at %v in a row %v wide",
							pack, scale, size[0], size[1], word, box.Bounds().Max.X, row.LocalBounds().Dx())
					}
				}
				words := 0
				for _, word := range settingWords {
					if findRowLabel(w.Content(), word) != nil {
						words++
					}
				}
				if words != len(settingWords) {
					t.Errorf("%s at %g×, %dx%d: %d of the %d words in front of the choosers are on the page",
						pack, scale, size[0], size[1], words, len(settingWords))
				}
				for _, name := range []string{"Window corners", "Icons", "Icon size", "Paint renderer"} {
					cb := namedCombo(w.Content(), name)
					if cb == nil {
						t.Fatalf("%s at %g×, %dx%d: no %q chooser", pack, scale, size[0], size[1], name)
					}
					if cb.Bounds().Dx() < cb.Measure(layout.Unbounded()).X-0.51 {
						t.Errorf("%s at %g×, %dx%d: the %q chooser is squeezed to %v of %v",
							pack, scale, size[0], size[1], name, cb.Bounds().Dx(), cb.Measure(layout.Unbounded()).X)
					}
					if x := widget.DeviceOrigin(cb).X + cb.LocalBounds().Dx(); x > widget.DeviceOrigin(row).X+row.LocalBounds().Dx()+0.51 {
						t.Errorf("%s at %g×, %dx%d: the %q chooser ends at %v in a row that ends at %v",
							pack, scale, size[0], size[1], name, x, widget.DeviceOrigin(row).X+row.LocalBounds().Dx())
					}
				}
				// The fold: two lines at 1024x860, where the four options
				// fill the first and the four choosers fall onto the
				// second all by themselves, and three at the 720x520
				// minimum. Four would be a line too many — see above.
				//
				// It survived "OS borders" growing into "OS window
				// borders" (63 px) and a fourth chooser (136 px with its
				// word and the gap before it) because the order of the
				// row changed with them: the three narrow options lead,
				// so the widest folds down beside the corners instead of
				// leaving the icons a line of their own. Left as it was,
				// the minimum took four lines and the preview 55% of its
				// pane.
				lines := rowLines(row)
				want := 3
				if size[0] > 800 {
					want = 2
				}
				if lines > want {
					t.Errorf("%s at %g×, %dx%d: the settings stand on %d lines in a %v pane, want at most %d",
						pack, scale, size[0], size[1], lines, row.LocalBounds().Dx(), want)
				}
				// And the browser fills its own column, the way the
				// preview fills the pane beside it: the list runs from
				// the decade filter to the Export at the foot, and how
				// many of the 131 packs that shows is the whole of what
				// the column is for. It was held to 252 logical pixels
				// — nine rows at scale 1, five at 1.75 — with about 250
				// px of nothing under the buttons.
				col := browserColumn(t, w)
				list := findThemeListAny(w.Content())
				export := findButton(w.Content(), "Export current theme…")
				if list == nil || export == nil {
					t.Fatalf("%s at %g×, %dx%d: the column is not the browser", pack, scale, size[0], size[1])
				}
				if gap := export.Bounds().Min.Y - list.Bounds().Max.Y; gap > 16*scale {
					t.Errorf("%s at %g×, %dx%d: %v px of nothing between the list and Export",
						pack, scale, size[0], size[1], gap)
				}
				if tail := col.LocalBounds().Dy() - export.Bounds().Max.Y; tail > 16*scale {
					t.Errorf("%s at %g×, %dx%d: %v px of nothing under Export",
						pack, scale, size[0], size[1], tail)
				}
				lo, hi := list.VisibleRange()
				if hi-lo < 4 {
					t.Errorf("%s at %g×, %dx%d: the browser shows %d of %d packs",
						pack, scale, size[0], size[1], hi-lo, list.Count)
				}
				t.Logf("%s at %g×, %dx%d: the settings are %.0f px wide on %d line(s), all %d words shown; preview %.0f%% of the pane; "+
					"the browser shows %d of %d packs in %.0f px, %.0f%% of a %.0f px column",
					pack, scale, size[0], size[1], row.LocalBounds().Dx(), lines, words, 100*box.Dy()/pane.Dy(),
					hi-lo, list.Count, list.LocalBounds().Dy(), 100*list.LocalBounds().Dy()/col.LocalBounds().Dy(), col.LocalBounds().Dy())
				if dir != "" {
					name := fmt.Sprintf("settings-%s%s-%gx-%dx%d.png", page, pack, scale, size[0], size[1])
					if err := w.WritePNG(filepath.Join(dir, name)); err != nil {
						t.Fatal(err)
					}
				}
				w.Close()
			}
		}
	}
}
