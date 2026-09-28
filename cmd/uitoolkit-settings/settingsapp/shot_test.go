package settingsapp

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The Theme Atlas crops the preview panel out of `uitoolkit-settings -stage ID
// -screenshot` at a fixed rectangle, so it has to sit in the same place
// in every pack. These are those pixels in the default 1024x860 window at
// scale 1; a layout change that moves them has to be published with the
// new numbers (docs/settings.md, "Screenshot geometry").
//
// What the preview carries inside itself does not move them: the panel
// takes whatever the blocks over and under it leave, so the settings bar
// that stood at the head of the window for a release grew inside this
// rectangle rather than pushing it down. What is over and under it does
// move them, and the last three changes moved them all three ways.
//
// Three choosers came off that bar and onto the page, which gave the
// block over the panel a second line and pushed its top down 30 px
// (y 44 → 74); the three paths under it gave up the group box they were
// in, whose legend and frame were 34 px (101 → 67), so the panel grew
// from 647 to 651. Renaming "OS borders" to "OS borders" and
// adding a fourth chooser moved nothing, although between them they put
// 199 px more into that folding row: the row's order changed with them
// and it still folded onto two lines.
//
// The two typeface choosers are what moved it this time, and they moved
// it a long way down. Ten groups will not fold onto two lines in this
// pane, so the settings are two blocks now — what the toolkit does, then
// what it is drawn with — and the second block is 64 px of its own
// (y 74 → 138). Under the panel the three path lines became one, which
// gave 18 px back, so the panel is 633 tall where it was 651: 46 px went
// into the settings and 18 came back out of the paths. The preview is
// still far the biggest thing in its pane — 80% of it at this size and
// 65% at the 720x520 minimum, where it was 82% and 62% — because a
// block that folds onto two lines at both sizes costs the minimum less
// than a row that folded onto three.
//
// These numbers were re-measured in the eight packs below, and rendered
// through tools/atlas/render.sh's own crop, rather than assumed.
//
// They are measured in the *bundled* faces, which is why tools/test.sh and
// render.sh both pin UITK_SYSTEM_FONTS=0. A look reads in the era's
// typeface when the machine has it, and the panel then ends somewhere
// else: 648 here, 645 without, and the 3 px was this file passing on the
// machine it was written on and failing on macOS, which has no fontconfig
// at all. The bundled faces are on every machine, so the height below is
// the toolkit's and not the developer's.
//
// The column on the left cannot move them either, whatever is taken out
// of it: the splitter's ratio is worked out from the window's width and
// the column has no minimum that could push the sash. Stripping it to
// four bare controls — no group box, no heading, no version, no Delete
// theme… — and running the list to the foot of it left these numbers
// exactly where they were.
const (
	previewShotX = 317
	previewShotY = 116
	previewShotW = 697
	previewShotH = 645
)

func TestSettingsPreviewPanelKeepsItsPlace(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	for _, pack := range []string{"win95", "luna", "aqua", "breeze", "adwaita", "bigsur", "tahoe", "vscode-night"} {
		// The look Settings itself runs in, the way the command starts it:
		// the column of choices beside the preview is drawn in it, and the
		// splitter aims that column at 300 logical pixels, so where the
		// preview panel begins is Settings' own metrics' business. A
		// fixture look here and PreferredLook in the command would pin a
		// rectangle the atlas never renders.
		a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: 1, DisableLookWatch: true})
		w, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: 1024, Height: 860, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(SettingsAppStaged(a, w, pack))
		a.PumpOnce()
		scope := previewScope(t, w)
		o, b := widget.DeviceOrigin(scope), scope.LocalBounds()
		got := [4]int{int(o.X), int(o.Y), int(b.Dx()), int(b.Dy())}
		want := [4]int{previewShotX, previewShotY, previewShotW, previewShotH}
		if got != want {
			t.Errorf("%s: the preview panel is at %v, the Theme Atlas crops %v — "+
				"move the crop and say so in docs/settings.md", pack, got, want)
		}
		w.Close()
	}
}
