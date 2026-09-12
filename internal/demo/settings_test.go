package demo

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestSettingsAppAppliesAndPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}
	if err := style.SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: 1, DisableLookWatch: true})
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
	if findMenuBar(w.Content()) {
		t.Fatal("settings must not have a menu bar")
	}
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("look %+v", got)
	}
	if style.LoadAppearance() != want.Normalize() {
		t.Fatalf("prefs %+v", style.LoadAppearance())
	}

	clickRadio(t, w, "Dark graphite")
	a.PumpOnce()
	if a.Look().Name() != "dark" {
		t.Fatalf("preview theme %s", a.Look().Name())
	}
	if style.LoadAppearance() != want.Normalize() {
		t.Fatalf("toggle must not write prefs: %+v", style.LoadAppearance())
	}
	if findApply(w.Content()) == nil || !findApply(w.Content()).Enabled() {
		t.Fatal("Apply should be enabled after a staged change")
	}

	clickApply(t, w)
	a.PumpOnce()
	saved := style.LoadAppearance()
	if saved.Theme != style.ThemeDark || saved.Corners != style.CornersSquare || saved.Icons != style.IconSetSharp {
		t.Fatalf("applied prefs %+v", saved)
	}
	if findApply(w.Content()).Enabled() {
		t.Fatal("Apply should disable once saved matches staged")
	}
}

func TestSettingsApplyNotifiesOtherApp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	settingsApp := uitoolkit.New(uitoolkit.Options{Headless: true, Scale: 1, DisableLookWatch: true})
	sw, err := settingsApp.NewWindow(platform.WindowOptions{Title: "settings", Width: 960, Height: 780, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer sw.Close()
	sw.SetContent(SettingsApp(settingsApp, sw))
	settingsApp.PumpOnce()

	listener := app.New(app.Options{Headless: true, Scale: 1})
	lw, err := listener.NewWindow(platform.WindowOptions{Title: "mail", Width: 320, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer lw.Close()
	lw.SetContent(widgets.NewLabel("other app"))
	listener.PumpOnce()
	if listener.Look().Name() != "dark" {
		t.Fatalf("listener start %s", listener.Look().Name())
	}

	clickRadio(t, sw, "Light paper")
	settingsApp.PumpOnce()
	listener.PumpOnce()
	if style.LoadAppearance().Theme != style.ThemeDark {
		t.Fatal("unapplied preview leaked to look.json")
	}
	if listener.Look().Name() != "dark" {
		t.Fatal("listener must stay dark until Apply")
	}

	clickApply(t, sw)
	settingsApp.PumpOnce()
	listener.PumpOnce()
	if style.LoadAppearance().Theme != style.ThemeLight {
		t.Fatalf("apply wrote %+v", style.LoadAppearance())
	}
	if listener.Look().Name() != "light" {
		t.Fatalf("listener after apply %s", listener.Look().Name())
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
	if findMenuBar(w.Content()) {
		t.Fatal("settings must not have a menu bar")
	}
	if findApply(w.Content()) == nil {
		t.Fatal("missing Apply")
	}
}

func findMenuBar(root widget.Component) bool {
	found := false
	widget.Walk(root, func(c widget.Component) {
		if _, ok := c.(*widgets.MenuBar); ok {
			found = true
		}
	})
	return found
}

func findApply(root widget.Component) *widgets.Button {
	var apply *widgets.Button
	widget.Walk(root, func(c widget.Component) {
		if b, ok := c.(*widgets.Button); ok && b.Text == "Apply" {
			apply = b
		}
	})
	return apply
}

func clickRadio(t *testing.T, w *app.Window, label string) {
	t.Helper()
	var rb *widgets.RadioButton
	widget.Walk(w.Content(), func(c widget.Component) {
		if r, ok := c.(*widgets.RadioButton); ok && r.Text == label {
			rb = r
		}
	})
	if rb == nil {
		t.Fatalf("no radio %q", label)
	}
	rb.MousePress(widget.MouseEvent{})
}

func clickApply(t *testing.T, w *app.Window) {
	t.Helper()
	apply := findApply(w.Content())
	if apply == nil || apply.OnClick == nil {
		t.Fatal("no Apply button")
	}
	apply.OnClick()
}
