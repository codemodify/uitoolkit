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
// takes whatever the two rows over and under it leave, so the settings
// bar at the head of the window grew inside this rectangle rather than
// pushing it down. What is over and under it does move them, and the
// row of options that replaced the single colours switch moved them up
// and made the panel taller: the switch stood over a line of prose, the
// five check boxes fit on one line at this size, and the 22 pixels that
// line of prose was taking went to the preview.
const (
	previewShotX = 317
	previewShotY = 44
	previewShotW = 697
	previewShotH = 647
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
