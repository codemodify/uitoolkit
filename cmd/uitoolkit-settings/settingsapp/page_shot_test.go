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
)

// The page is one column of choices and one preview that sets what it
// shows, and whether
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
				// keep, and the reason the five options over it are one
				// folding row of check boxes rather than a stack of
				// switches with a line of prose each. It is 82% of the
				// pane at 1024x860 and 61% at the 720x520 minimum, where
				// the row folds onto two lines; five switches would have
				// folded onto three and left it 53%.
				split := settingsSplit(t, w)
				pane := split.PaneB()
				box := previewScope(t, w).LocalBounds()
				if box.Dy() < pane.Dy()*0.6 {
					t.Errorf("%s at %g×, %dx%d: the preview is %v of a %v pane",
						pack, scale, size[0], size[1], box.Dy(), pane.Dy())
				}
				// And the three settings it carries are whole and on
				// their bar at every one of these sizes. The words in
				// front of them are not promised — the bar sheds those
				// from the right when it runs out of room, and at the
				// 720x520 minimum all three go — but a chooser cut off
				// by the window frame would be a setting nobody can
				// reach.
				bar := previewSettings(t, w)
				words := 0
				for i, it := range bar.Items() {
					if it.Label && !bar.ItemRect(i).Empty() {
						words++
					}
				}
				for _, name := range []string{"Icons", "Icon size", "Window corners"} {
					cb := namedCombo(w.Content(), name)
					if cb == nil {
						t.Fatalf("%s at %g×, %dx%d: no %q chooser", pack, scale, size[0], size[1], name)
					}
					if cb.Bounds().Dx() < cb.Measure(layout.Unbounded()).X-0.51 {
						t.Errorf("%s at %g×, %dx%d: the %q chooser is squeezed to %v of %v",
							pack, scale, size[0], size[1], name, cb.Bounds().Dx(), cb.Measure(layout.Unbounded()).X)
					}
					if in := cb.Bounds(); in.Max.X > bar.LocalBounds().Dx()+0.51 {
						t.Errorf("%s at %g×, %dx%d: the %q chooser ends at %v on a bar %v wide",
							pack, scale, size[0], size[1], name, in.Max.X, bar.LocalBounds().Dx())
					}
				}
				// The five options over it are whole at every one of
				// these sizes too. A check box does not elide and the
				// row does not shed: it folds, and what it must never do
				// is hand one of them a width its word does not fit in
				// or push one past the pane's right edge.
				row := optionsRow(t, w)
				for _, word := range []string{"Animations", "File dialogs", "System frame", "Theme buttons", "Desktop colours"} {
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
				lines := rowLines(row)
				if lines > 2 {
					t.Errorf("%s at %g×, %dx%d: the five options stand on %d lines in a %v pane",
						pack, scale, size[0], size[1], lines, row.LocalBounds().Dx())
				}
				// And the browser fills its own column, the way the
				// preview fills the pane beside it: the list runs from
				// the decade filter to the Export at the foot, and how
				// many of the 129 packs that shows is the whole of what
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
				t.Logf("%s at %g×, %dx%d: bar %v wide, %d of 3 words shown; options on %d line(s), preview %.0f%% of the pane; "+
					"the browser shows %d of %d packs in %.0f px, %.0f%% of a %.0f px column",
					pack, scale, size[0], size[1], bar.LocalBounds().Dx(), words, lines, 100*box.Dy()/pane.Dy(),
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
