package tourapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func packOfLook(t *testing.T, lk style.LookAndFeel) string {
	t.Helper()
	c, ok := lk.(*style.Classic)
	if !ok {
		t.Fatalf("not a Classic look: %T", lk)
	}
	return c.Pack()
}

// Three packs in one window, and the still that says so. The shot is
// written when UITK_SHOTS names a directory:
//
//	UITK_SHOTS=/tmp/shots tools/testenv.sh go test ./examples/uitoolkit-sample-tour/... -run CascadePage
func TestTourCascadePageHoldsThreePacks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	for _, window := range []string{"luna", "breeze6", "tahoe"} {
		t.Run(window, func(t *testing.T) {
			a, win := tourWindow(t, window, 1, pageCascade)
			defer win.Close()
			p := stateOf(t, win).page(pageCascade).(*cascadePage)

			p.outerPack, p.innerPack = indexOfPick(t, "aqua"), indexOfPick(t, "win95")
			p.retheme()
			a.PumpOnce()
			a.PumpOnce()

			if got := packOfLook(t, win.Look()); got != window {
				t.Fatalf("the window is in %s", got)
			}
			if got := packOfLook(t, p.outer.Look()); got != "aqua" {
				t.Fatalf("the pane is in %s", got)
			}
			if got := packOfLook(t, p.inner.Look()); got != "win95" {
				t.Fatalf("the inner pane is in %s", got)
			}
			if got := packOfLook(t, p.plain.Look()); got != window {
				t.Fatalf("the pane that states nothing is in %s, not the window's %s", got, window)
			}
			// A menu opened from the inner pane is on a surface of its
			// own and outside the window's tree entirely, and it is
			// still in the inner pane's pack.
			menu, ok := widget.Find[*widgets.Button](p.inner)
			if !ok {
				t.Fatal("no button in the inner pane")
			}
			menu.OnClick()
			a.PumpOnce()
			pop := win.Popup()
			if pop == nil {
				t.Fatal("no menu")
			}
			if got := packOfLook(t, pop.Look()); got != "win95" {
				t.Fatalf("the menu opened in %s, not the inner pane's win95", got)
			}
			if dir := os.Getenv("UITK_SHOTS"); dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := win.WritePNG(filepath.Join(dir, "tour-cascade-"+window+".png")); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// The corner policy is the application's and reaches every level: two
// panes in packs that never mention corners both go square with the
// window.
func TestTourCascadeInheritsCorners(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	_, win := tourWindow(t, "aqua", 1, pageCascade)
	defer win.Close()
	p := stateOf(t, win).page(pageCascade).(*cascadePage)
	p.outerPack, p.innerPack = indexOfPick(t, "luna"), indexOfPick(t, "tahoe")
	p.retheme()

	p.t.apply(func(ap *style.Appearance) { ap.Corners = style.CornersSquare })
	for what, c := range map[string]widget.Component{"the pane": p.outer, "the inner pane": p.inner} {
		if got := style.LookAppearance(c.Look()).Corners; got != style.CornersSquare {
			t.Fatalf("%s did not inherit square corners: %s", what, got)
		}
		if r := c.Look().Metrics().Radius; r != 0 {
			t.Fatalf("%s still has a radius of %v", what, r)
		}
	}
	if got := style.LookAppearance(p.outer.Look()).Name; got != "luna" {
		t.Fatalf("inheriting the corners must not change the pack: %s", got)
	}
}

func indexOfPick(t *testing.T, name string) int {
	t.Helper()
	for i, n := range cascadePicks {
		if n == name {
			return i
		}
	}
	t.Fatalf("%q is not one of the page's packs", name)
	return 0
}
