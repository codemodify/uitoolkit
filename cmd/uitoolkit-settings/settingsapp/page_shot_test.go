package settingsapp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/uitoolkit"
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
// UITK_PAGE opens them at a section (-page's names), for the parts of
// the column below the fold.
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
				// keep, and the reason the two rows that share it with
				// it — the colours switch and the three paths — are one
				// row each.
				split := settingsSplit(t, w)
				box := previewScope(t, w).LocalBounds()
				if pane := split.PaneB(); box.Dy() < pane.Dy()*0.6 {
					t.Errorf("%s at %g×, %dx%d: the preview is %v of a %v pane",
						pack, scale, size[0], size[1], box.Dy(), pane.Dy())
				}
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
