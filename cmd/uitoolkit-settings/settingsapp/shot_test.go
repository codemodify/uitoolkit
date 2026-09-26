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
// takes whatever the two blocks over and under it leave, so the settings
// bar that stood at the head of the window for a release grew inside
// this rectangle rather than pushing it down. What is over and under it
// does move them, and the last two changes moved them both ways at once.
// Three choosers came off that bar and onto the page, which gave the
// block over the panel a second line and pushed its top down 30 px
// (y 44 → 74); the three paths under it gave up the group box they were
// in, whose legend and frame were 34 px (101 → 67). Four of those 34 are
// the difference: the panel is 651 tall where it was 647.
//
// Renaming "OS borders" to "OS window borders" and adding a fourth
// chooser did not move them either, which is not obvious: between them
// they put 199 px more into that folding row. The row's order changed
// with them (settings.go, optionsRow), so it still folds onto two lines
// at this size and still stands 56 px tall. These numbers were
// re-measured in the eight packs below, and rendered through
// tools/atlas/render.sh's own crop in three of them, rather than
// assumed.
//
// The column on the left cannot move them either, whatever is taken out
// of it: the splitter's ratio is worked out from the window's width and
// the column has no minimum that could push the sash. Stripping it to
// four bare controls — no group box, no heading, no version, no Delete
// theme… — and running the list to the foot of it left these numbers
// exactly where they were.
const (
	previewShotX = 317
	previewShotY = 74
	previewShotW = 697
	previewShotH = 651
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
