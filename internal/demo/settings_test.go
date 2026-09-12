package demo

import (
	"os"
	"path/filepath"
	"strings"
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
	if findRadio(w.Content(), "Dark graphite") != nil || findRadio(w.Content(), "Round corners") != nil {
		t.Fatal("theme/corners/icons triad must be replaced by a theme picker")
	}
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("look %+v", got)
	}
	if style.LoadAppearance() != want.Normalize() {
		t.Fatalf("prefs %+v", style.LoadAppearance())
	}

	clickTheme(t, w, "Dark · Square · Sharp")
	a.PumpOnce()
	if a.Look().Name() != "dark" {
		t.Fatalf("preview theme %s", a.Look().Name())
	}
	if style.LookAppearance(a.Look()).Name != "dark-square-sharp" {
		t.Fatalf("preview pack %+v", style.LookAppearance(a.Look()))
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
	if saved.Name != "dark-square-sharp" || saved.Theme != style.ThemeDark || saved.Corners != style.CornersSquare || saved.Icons != style.IconSetSharp {
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

	clickTheme(t, sw, "Light · Round · Classic")
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
	if style.LoadAppearance().Name != "light-round-classic" || style.LoadAppearance().Theme != style.ThemeLight {
		t.Fatalf("apply wrote %+v", style.LoadAppearance())
	}
	if listener.Look().Name() != "light" {
		t.Fatalf("listener after apply %s", listener.Look().Name())
	}
}

func TestSettingsExportThemeByName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.Appearance{Theme: style.ThemeLight, Corners: style.CornersRound, Icons: style.IconSetClassic}); err != nil {
		t.Fatal(err)
	}
	a := uitoolkit.New(uitoolkit.Options{Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 960, Height: 780, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsApp(a, w))
	a.PumpOnce()

	clickExportLook(t, w)
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("export should ask for a name")
	}
	field := findOverlayField(w)
	if field == nil {
		t.Fatal("missing name field")
	}
	field.SetText("ocean")
	clickNamed(t, w.Overlay(), "Export")
	a.PumpOnce()

	raw, err := os.ReadFile(style.ThemeFile("ocean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"palette": "light"`) {
		t.Fatalf("exported pack: %s", raw)
	}
	pack, ok := style.LoadTheme("ocean")
	if !ok || pack.Source != style.ThemeSourceUser || pack.Palette != style.ThemeLight {
		t.Fatalf("load exported %+v ok=%v", pack, ok)
	}
	if findThemeList(w.Content()) == nil {
		t.Fatal("theme picker missing after export")
	}
	listed := false
	widget.Walk(w.Content(), func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.Count >= 8 && l.ItemText != nil {
			for i := 0; i < l.Count; i++ {
				if strings.Contains(l.ItemText(i), "ocean") {
					listed = true
				}
			}
		}
	})
	if !listed {
		t.Fatal("exported pack not listed")
	}
}

func TestSettingsIconSetApplyWritesLookJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	installSettingsIconSet(t, dir, "outline")
	a := uitoolkit.New(uitoolkit.Options{Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 960, Height: 780, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsApp(a, w))
	a.PumpOnce()
	if findIconList(w.Content()) == nil {
		t.Fatal("missing icon picker")
	}
	clickIconSet(t, w, "outline")
	a.PumpOnce()
	if style.LookAppearance(a.Look()).Icons != style.IconSetOutline {
		t.Fatalf("preview icons %+v", style.LookAppearance(a.Look()))
	}
	if style.LoadAppearance().Icons != style.IconSetClassic {
		t.Fatal("unapplied icon change leaked")
	}
	clickApply(t, w)
	a.PumpOnce()
	got := style.LoadAppearance()
	if got.Name != style.DefaultThemeName || got.Icons != style.IconSetOutline {
		t.Fatalf("applied %+v", got)
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"icons": "outline"`) {
		t.Fatalf("look.json: %s", raw)
	}
}

func installSettingsIconSet(t *testing.T, xdg, name string) {
	t.Helper()
	src := filepath.Join("..", "..", "icons", name)
	dst := filepath.Join(xdg, "uitoolkit", "icons", name)
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func clickIconSet(t *testing.T, w *app.Window, name string) {
	t.Helper()
	list := findIconList(w.Content())
	if list == nil || list.OnSelect == nil {
		t.Fatal("no icon picker")
	}
	for i := 0; i < list.Count; i++ {
		if strings.Contains(list.ItemText(i), name) {
			list.OnSelect(i)
			return
		}
	}
	t.Fatalf("no icon set %q", name)
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
	if findThemeList(w.Content()) == nil {
		t.Fatal("missing theme picker")
	}
	if findIconList(w.Content()) == nil {
		t.Fatal("missing icon picker")
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
	return findButton(root, "Apply")
}

func findButton(root widget.Component, text string) *widgets.Button {
	var btn *widgets.Button
	widget.Walk(root, func(c widget.Component) {
		if b, ok := c.(*widgets.Button); ok && b.Text == text {
			btn = b
		}
	})
	return btn
}

func findRadio(root widget.Component, label string) *widgets.RadioButton {
	var rb *widgets.RadioButton
	widget.Walk(root, func(c widget.Component) {
		if r, ok := c.(*widgets.RadioButton); ok && r.Text == label {
			rb = r
		}
	})
	return rb
}

func findThemeList(root widget.Component) *widgets.ListView {
	var list *widgets.ListView
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil && l.Count >= 8 {
			for i := 0; i < l.Count; i++ {
				if strings.Contains(l.ItemText(i), "Dark ·") {
					list = l
					return
				}
			}
		}
	})
	return list
}

func findIconList(root widget.Component) *widgets.ListView {
	var list *widgets.ListView
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil {
			for i := 0; i < l.Count; i++ {
				if strings.Contains(l.ItemText(i), "Classic  · builtin") {
					list = l
					return
				}
			}
		}
	})
	return list
}

func clickTheme(t *testing.T, w *app.Window, label string) {
	t.Helper()
	list := findThemeList(w.Content())
	if list == nil || list.ItemText == nil || list.OnSelect == nil {
		t.Fatal("no theme picker")
	}
	for i := 0; i < list.Count; i++ {
		if strings.Contains(list.ItemText(i), label) {
			list.OnSelect(i)
			return
		}
	}
	t.Fatalf("no theme %q", label)
}

func clickExportLook(t *testing.T, w *app.Window) {
	t.Helper()
	btn := findButton(w.Content(), "Export current look…")
	if btn == nil || btn.OnClick == nil {
		t.Fatal("no Export current look…")
	}
	btn.OnClick()
}

func clickNamed(t *testing.T, root widget.Component, text string) {
	t.Helper()
	btn := findButton(root, text)
	if btn == nil || btn.OnClick == nil {
		t.Fatalf("no button %q", text)
	}
	btn.OnClick()
}

func findOverlayField(w *app.Window) *widgets.TextField {
	var field *widgets.TextField
	if w.Overlay() == nil {
		return nil
	}
	widget.Walk(w.Overlay(), func(c widget.Component) {
		if tf, ok := c.(*widgets.TextField); ok {
			field = tf
		}
	})
	return field
}

func clickApply(t *testing.T, w *app.Window) {
	t.Helper()
	apply := findApply(w.Content())
	if apply == nil || apply.OnClick == nil {
		t.Fatal("no Apply button")
	}
	apply.OnClick()
}
