package demo

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

func TestSettingsAppAppliesAndPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}
	if err := style.SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 960, Height: 780, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsApp(a, w))
	a.PumpOnce()
	if w.Content() == nil {
		t.Fatal("no content")
	}
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("look %+v", got)
	}
	if style.LoadAppearance() != want.Normalize() {
		t.Fatalf("prefs %+v", style.LoadAppearance())
	}
	next := style.Appearance{Theme: style.ThemeDark, Corners: style.CornersRound, Icons: style.IconSetClassic}
	if err := style.SaveAppearance(next); err != nil {
		t.Fatal(err)
	}
	a.SetLook(next.Look())
	w.SetContent(SettingsApp(a, w))
	a.PumpOnce()
	if a.Look().Name() != "dark" {
		t.Fatalf("theme %s", a.Look().Name())
	}
	if a.Look().Metrics().Radius <= 0 {
		t.Fatal("round radius")
	}
}

func TestSettingsAppPaintsPreview(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 960, Height: 780, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsApp(a, w))
	img := w.Capture()
	if img == nil {
		t.Fatal("capture")
	}
	ink := 0
	for y := 80; y < img.Height-40; y++ {
		for x := 280; x < img.Width-20; x++ {
			_, _, _, al := img.PremulAt(x, y)
			if al > 30 {
				ink++
			}
		}
	}
	if ink < 800 {
		t.Fatalf("right pane looks empty (ink=%d)", ink)
	}
}
