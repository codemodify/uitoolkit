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
		t.Fatal("old triad radio labels must be gone")
	}
	if findRadio(w.Content(), "Round") == nil || findRadio(w.Content(), "Square") == nil {
		t.Fatal("corners should be a separate Round / Square control")
	}
	if findRadio(w.Content(), "Small") == nil || findRadio(w.Content(), "Medium") == nil || findRadio(w.Content(), "Large") == nil {
		t.Fatal("icon size should be a separate Small / Medium / Large control")
	}
	if !findLabel(w.Content(), "Built-in") || !findLabel(w.Content(), "User") {
		t.Fatal("Theme and Icons lists should split Built-in vs User")
	}
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("look %+v", got)
	}
	if style.LoadAppearance() != want.Normalize() {
		t.Fatalf("prefs %+v", style.LoadAppearance())
	}

	clickTheme(t, w, "Classic 95 Dark")
	a.PumpOnce()
	if a.Look().Name() != "dark" {
		t.Fatalf("preview theme %s", a.Look().Name())
	}
	live := style.LookAppearance(a.Look())
	if live.Name != "dark" || live.Corners != style.CornersSquare || live.Icons != style.IconSetSharp {
		t.Fatalf("preview should keep corners/icons %+v", live)
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
	if saved.Name != "dark" || saved.Theme != style.ThemeDark || saved.Corners != style.CornersSquare || saved.Icons != style.IconSetSharp {
		t.Fatalf("applied prefs %+v", saved)
	}
	if findApply(w.Content()).Enabled() {
		t.Fatal("Apply should disable once saved matches staged")
	}

	clickCorners(t, w, "Round")
	a.PumpOnce()
	if style.LookAppearance(a.Look()).Corners != style.CornersRound || style.LookAppearance(a.Look()).Name != "dark" {
		t.Fatalf("corners preview %+v", style.LookAppearance(a.Look()))
	}
	clickApply(t, w)
	a.PumpOnce()
	saved = style.LoadAppearance()
	if saved.Name != "dark" || saved.Corners != style.CornersRound || saved.Icons != style.IconSetSharp {
		t.Fatalf("corners apply %+v", saved)
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

	clickTheme(t, sw, "Classic 95 Light")
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
	if style.LoadAppearance().Name != "light" || style.LoadAppearance().Theme != style.ThemeLight {
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
	if strings.Contains(string(raw), `"corners"`) || strings.Contains(string(raw), `"icons"`) {
		t.Fatalf("export should be palette only: %s", raw)
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
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil {
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

func TestSettingsDeleteUserTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}); err != nil {
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
	if findButton(w.Content(), "Delete") != nil {
		t.Fatal("Delete must not appear for built-in themes")
	}

	if _, err := style.ExportAppearance("ocean", style.LoadAppearance()); err != nil {
		t.Fatal(err)
	}
	w.SetContent(SettingsApp(a, w))
	a.PumpOnce()
	clickTheme(t, w, "ocean")
	a.PumpOnce()
	if findButton(w.Content(), "Delete") == nil {
		t.Fatal("Delete should appear for a selected user theme")
	}
	clickApply(t, w)
	a.PumpOnce()
	if style.LoadAppearance().Name != "ocean" {
		t.Fatalf("apply ocean %+v", style.LoadAppearance())
	}

	clickNamed(t, w.Content(), "Delete")
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("delete should confirm")
	}
	clickNamed(t, w.Overlay(), "Yes")
	a.PumpOnce()

	if _, err := os.Stat(style.ThemeFile("ocean")); !os.IsNotExist(err) {
		t.Fatalf("user theme still on disk: %v", err)
	}
	if _, ok := style.LoadTheme("ocean"); ok {
		t.Fatal("deleted theme still loads")
	}
	live := style.LookAppearance(a.Look())
	if live.Name != "light" || live.Theme != style.ThemeLight || live.Corners != style.CornersSquare || live.Icons != style.IconSetSharp {
		t.Fatalf("fallback %+v", live)
	}
	if style.LoadAppearance() != live {
		t.Fatalf("look.json left dangling %+v", style.LoadAppearance())
	}
	if findButton(w.Content(), "Delete") != nil {
		t.Fatal("Delete should hide after falling back to a builtin")
	}
	listed := false
	widget.Walk(w.Content(), func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil {
			for i := 0; i < l.Count; i++ {
				if l.ItemText(i) == "ocean" {
					listed = true
				}
			}
		}
	})
	if listed {
		t.Fatal("deleted pack still listed")
	}
}

func TestSettingsIconSetApplyWritesLookJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	installSettingsIconSet(t, dir, "lucide")
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
	clickIconSet(t, w, "Lucide")
	a.PumpOnce()
	if style.LookAppearance(a.Look()).Icons != style.IconSetLucide {
		t.Fatalf("preview icons %+v", style.LookAppearance(a.Look()))
	}
	if style.LoadAppearance().Icons != style.IconSetClassic {
		t.Fatal("unapplied icon change leaked")
	}
	clickApply(t, w)
	a.PumpOnce()
	got := style.LoadAppearance()
	if got.Name != style.DefaultThemeName || got.Icons != style.IconSetLucide {
		t.Fatalf("applied %+v", got)
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"icons": "lucide"`) {
		t.Fatalf("look.json: %s", raw)
	}
}

func TestSettingsIconSizeApplyWritesLookJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
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
	clickIconSize(t, w, "Large")
	a.PumpOnce()
	if style.LookAppearance(a.Look()).IconSize != style.IconSizeLarge {
		t.Fatalf("preview size %+v", style.LookAppearance(a.Look()))
	}
	if style.LoadAppearance().IconSize != style.IconSizeMedium {
		t.Fatal("unapplied icon size leaked")
	}
	clickApply(t, w)
	a.PumpOnce()
	got := style.LoadAppearance()
	if got.IconSize != style.IconSizeLarge || got.Icons != style.IconSetClassic {
		t.Fatalf("applied %+v", got)
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"iconSize": "large"`) {
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

func TestSettingsThemeListScrollsAllBuiltins(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a := uitoolkit.New(uitoolkit.Options{Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 720, Height: 520, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsApp(a, w))
	a.PumpOnce()

	packs := style.ListBuiltinThemes()
	if len(packs) < 8 {
		t.Fatalf("expected era packs, got %d", len(packs))
	}
	list := findThemeList(w.Content())
	if list == nil {
		t.Fatal("missing theme picker")
	}
	if list.Count != len(packs) {
		t.Fatalf("theme list count=%d want %d (one Built-in list)", list.Count, len(packs))
	}
	seen := map[string]bool{}
	for i := 0; i < list.Count; i++ {
		seen[list.ItemText(i)] = true
	}
	for _, p := range packs {
		if !seen[p.Display()] {
			t.Fatalf("missing theme %q", p.Display())
		}
	}
	if list.MaxOffset() <= 0 {
		t.Fatalf("theme list should overflow at 720×520, view=%v rows=%d", list.LocalBounds().Dy(), list.Count)
	}
	if _, thumb := list.ScrollTrack(); thumb.Empty() {
		t.Fatal("overflow theme list should show scrollbar chrome")
	}
	last := packs[len(packs)-1]
	list.ScrollTo(list.MaxOffset())
	lo, hi := list.VisibleRange()
	if lastIdx := list.Count - 1; lastIdx < lo || lastIdx >= hi {
		t.Fatalf("last theme not in view after scroll lo=%d hi=%d count=%d", lo, hi, list.Count)
	}
	if list.OnSelect == nil {
		t.Fatal("theme list has no OnSelect")
	}
	list.OnSelect(list.Count - 1)
	a.PumpOnce()
	live := style.LookAppearance(a.Look())
	if live.Name != last.Name {
		t.Fatalf("preview last theme %+v want %s", live, last.Name)
	}
	apply := findApply(w.Content())
	if apply == nil || !apply.Enabled() {
		t.Fatal("Apply should stay reachable and enabled after staging")
	}
	if nestedInScroll(apply) {
		t.Fatal("Apply must stay pinned outside the scroll region")
	}
	if findRadio(w.Content(), "Round") == nil || findRadio(w.Content(), "Square") == nil {
		t.Fatal("Corners options must stay reachable")
	}
	if findRadio(w.Content(), "Small") == nil || findRadio(w.Content(), "Large") == nil {
		t.Fatal("Icon size options must stay reachable")
	}
	if findIconList(w.Content()) == nil {
		t.Fatal("missing icon picker")
	}

	clickSettingsNav(t, w, "About")
	a.PumpOnce()
	if findApply(w.Content()) == nil {
		t.Fatal("Apply missing on About")
	}
	if nestedInScroll(findApply(w.Content())) {
		t.Fatal("Apply must stay pinned on About")
	}
	aboutScroll := findScrollView(w.Content())
	if aboutScroll == nil {
		t.Fatal("About should scroll")
	}
	if aboutScroll.MaxOffset() <= 0 {
		t.Fatalf("About should overflow at 720×520, content=%v view=%v", aboutScroll.ContentHeight(), aboutScroll.LocalBounds().Dy())
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

func findLabel(root widget.Component, text string) bool {
	found := false
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.Label); ok && l.Text == text {
			found = true
		}
	})
	return found
}

func findThemeList(root widget.Component) *widgets.ListView {
	var list *widgets.ListView
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil {
			for i := 0; i < l.Count; i++ {
				if l.ItemText(i) == "dark" || l.ItemText(i) == "light" ||
					strings.Contains(l.ItemText(i), "Classic 95") {
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
			hasClassic, hasSharp := false, false
			for i := 0; i < l.Count; i++ {
				if l.ItemText(i) == "Classic" {
					hasClassic = true
				}
				if l.ItemText(i) == "Sharp" {
					hasSharp = true
				}
			}
			if hasClassic && hasSharp {
				list = l
			}
		}
	})
	return list
}

func clickTheme(t *testing.T, w *app.Window, label string) {
	t.Helper()
	var found *widgets.ListView
	var idx int
	widget.Walk(w.Content(), func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil && l.OnSelect != nil {
			for i := 0; i < l.Count; i++ {
				if strings.EqualFold(l.ItemText(i), label) || strings.Contains(strings.ToLower(l.ItemText(i)), strings.ToLower(label)) {
					found = l
					idx = i
				}
			}
		}
	})
	if found == nil {
		t.Fatalf("no theme %q", label)
	}
	found.OnSelect(idx)
}

func clickIconSize(t *testing.T, w *app.Window, label string) {
	t.Helper()
	rb := findRadio(w.Content(), label)
	if rb == nil {
		t.Fatalf("no icon size radio %q", label)
	}
	rb.MousePress(widget.MouseEvent{})
	rb.MouseRelease(widget.MouseEvent{})
}

func clickCorners(t *testing.T, w *app.Window, label string) {
	t.Helper()
	rb := findRadio(w.Content(), label)
	if rb == nil {
		t.Fatalf("no corners radio %q", label)
	}
	rb.MousePress(widget.MouseEvent{})
	rb.MouseRelease(widget.MouseEvent{})
}

func clickExportLook(t *testing.T, w *app.Window) {
	t.Helper()
	btn := findButton(w.Content(), "Export current theme…")
	if btn == nil || btn.OnClick == nil {
		t.Fatal("no Export current theme…")
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

func nestedInScroll(c widget.Component) bool {
	if c == nil {
		return false
	}
	for p := c.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*widgets.ScrollView); ok {
			return true
		}
	}
	return false
}

func findScrollView(root widget.Component) *widgets.ScrollView {
	var sv *widgets.ScrollView
	widget.Walk(root, func(c widget.Component) {
		if s, ok := c.(*widgets.ScrollView); ok {
			sv = s
		}
	})
	return sv
}

func clickSettingsNav(t *testing.T, w *app.Window, name string) {
	t.Helper()
	var nav *widgets.ListView
	widget.Walk(w.Content(), func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil && l.OnSelect != nil {
			for i := 0; i < l.Count; i++ {
				if l.ItemText(i) == "Appearance" {
					nav = l
					return
				}
			}
		}
	})
	if nav == nil {
		t.Fatal("missing Settings nav")
	}
	for i := 0; i < nav.Count; i++ {
		if nav.ItemText(i) == name {
			nav.OnSelect(i)
			return
		}
	}
	t.Fatalf("no settings section %q", name)
}

func clickApply(t *testing.T, w *app.Window) {
	t.Helper()
	apply := findApply(w.Content())
	if apply == nil || apply.OnClick == nil {
		t.Fatal("no Apply button")
	}
	apply.OnClick()
}
