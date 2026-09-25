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
const (
	previewShotX = 445
	previewShotY = 10
	previewShotW = 569
	previewShotH = 782
)

func TestSettingsPreviewPanelKeepsItsPlace(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	for _, pack := range []string{"win95", "luna", "aqua", "breeze", "adwaita", "bigsur", "tahoe", "vscode-night"} {
		a := uitoolkit.New(uitoolkit.Options{Look: style.LightLook(), Headless: true, Scale: 1, DisableLookWatch: true})
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
