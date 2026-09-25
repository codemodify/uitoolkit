package settingsapp

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

func openSettings(t *testing.T, w, h int) (*app.Application, *app.Window) {
	t.Helper()
	a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	win, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: w, Height: h, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { win.Close() })
	win.SetContent(SettingsApp(a, win))
	a.PumpOnce()
	return a, win
}

// The staged theme shows in the preview scope while Settings keeps the
// applied look; Apply persists it and switches Settings.
func TestSettingsAppAppliesAndPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}.Normalize()
	if err := style.SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 780)
	if findMenuBarOutsidePreview(w.Content()) {
		t.Fatal("settings must not have a menu bar (the preview app may)")
	}
	// One page: the corners, the icon set and the size its glyphs are
	// drawn at are all on it, with no navigating to do first.
	for _, opt := range []string{"Theme shape", "Round", "Square", "Small", "Medium", "Large", "Classic", "Sharp"} {
		if findCombo(w.Content(), opt) == nil {
			t.Fatalf("the chooser offering %q is not on the page", opt)
		}
	}
	// And so is every switch the Appearance page used to hold.
	for _, opt := range []string{"Animations", "Follow the desktop's colours",
		"Use the desktop's file dialogs", "Use system title bar and borders",
		"Place window buttons as the theme does"} {
		if findSwitch(w.Content(), opt) == nil {
			t.Fatalf("the %q switch is not on the page", opt)
		}
	}
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("look %+v", got)
	}

	clickTheme(t, w, "Classic 95 Dark")
	a.PumpOnce()
	staged := previewAppearance(t, w)
	if staged.Name != "dark" || staged.Corners != style.CornersSquare || staged.Icons != style.IconSetSharp {
		t.Fatalf("preview should show dark and keep corners/icons: %+v", staged)
	}
	if a.Look().Name() != "light" {
		t.Fatalf("Settings keeps the applied look while previewing, got %s", a.Look().Name())
	}
	if style.LoadAppearance() != want {
		t.Fatalf("staging must not write prefs: %+v", style.LoadAppearance())
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
	if a.Look().Name() != "dark" {
		t.Fatalf("Apply switches Settings too, got %s", a.Look().Name())
	}
	if findApply(w.Content()).Enabled() {
		t.Fatal("Apply should disable once saved matches staged")
	}

	pickCombo(t, w, "Round")
	a.PumpOnce()
	if p := previewAppearance(t, w); p.Corners != style.CornersRound || p.Name != "dark" {
		t.Fatalf("corners preview %+v", p)
	}
	clickApply(t, w)
	a.PumpOnce()
	if saved = style.LoadAppearance(); saved.Name != "dark" || saved.Corners != style.CornersRound || saved.Icons != style.IconSetSharp {
		t.Fatalf("corners apply %+v", saved)
	}

	// With Revert gone, Apply is the only writer: a staged pack shows in
	// the preview and stays out of look.json until it is applied.
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	if p := previewAppearance(t, w); p.Name != "win95" {
		t.Fatalf("the preview should show the staged pack: %+v", p)
	}
	if got := style.LoadAppearance().Name; got != "dark" {
		t.Fatalf("staging wrote look.json: %s", got)
	}
	if !findApply(w.Content()).Enabled() {
		t.Fatal("Apply should be enabled again with a pack staged")
	}
}

func TestSettingsApplyNotifiesOtherApp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance().WithPalette(style.ThemeDark)); err != nil {
		t.Fatal(err)
	}
	settingsApp, sw := openSettings(t, 1024, 780)

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
	a, w := openSettings(t, 1024, 780)
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
		t.Fatalf("export keeps corners/icons in look.json: %s", raw)
	}
	pack, ok := style.LoadTheme("ocean")
	if !ok || pack.Source != style.ThemeSourceUser || pack.Palette != style.ThemeLight {
		t.Fatalf("load exported %+v ok=%v", pack, ok)
	}
	// The pack it wrote is in the browser the Export button sits under,
	// as "User · ocean", and it is what is staged.
	if !listed(w.Content(), "ocean") {
		t.Fatal("exported pack not listed in the theme browser")
	}
	if got := previewAppearance(t, w).Name; got != "ocean" {
		t.Fatalf("export staged %q", got)
	}
}

func TestSettingsDeleteUserTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}); err != nil {
		t.Fatal(err)
	}
	if _, err := style.ExportAppearance("ocean", style.LoadAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 780)
	clickTheme(t, w, "ocean")
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if style.LoadAppearance().Name != "ocean" {
		t.Fatalf("apply ocean %+v", style.LoadAppearance())
	}

	// Delete is under the browser that staged the pack, and it is alive
	// because what is staged is the user's own.
	del := findButton(w.Content(), "Delete theme…")
	if del == nil || !del.Enabled() {
		t.Fatalf("Delete should be live for a staged user theme: %v", del)
	}
	clickNamed(t, w.Content(), "Delete theme…")
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
	// It goes grey rather than away once a built-in pack is staged: a
	// button that came and went would move the whole column under it
	// every time a pack was picked.
	if del := findButton(w.Content(), "Delete theme…"); del == nil || del.Enabled() {
		t.Fatalf("Delete should go grey after falling back to a builtin: %v", del)
	}
	if listed(w.Content(), "ocean") {
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
	a, w := openSettings(t, 1024, 780)
	pickCombo(t, w, "Lucide")
	a.PumpOnce()
	if p := previewAppearance(t, w); p.Icons != style.IconSetLucide {
		t.Fatalf("preview icons %+v", p)
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
	// The chooser is the only list of icon sets there is now, and it
	// offers every one on disk: the two drawn ones and the premiere set
	// that was copied in.
	combo := findCombo(w.Content(), "Lucide")
	if combo == nil {
		t.Fatal("no icon chooser")
	}
	for _, want := range []string{"Classic", "Sharp", "Lucide"} {
		found := false
		for _, it := range combo.Items {
			if it == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the icon chooser does not offer %q (has %v)", want, combo.Items)
		}
	}
}

func TestSettingsIconSizeApplyWritesLookJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 780)
	pickCombo(t, w, "Large")
	a.PumpOnce()
	if p := previewAppearance(t, w); p.IconSize != style.IconSizeLarge {
		t.Fatalf("preview size %+v", p)
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

func TestSettingsThemeListScrollsAllBuiltins(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 820, 560)

	packs := style.ListBuiltinThemes()
	if len(packs) < 8 {
		t.Fatalf("expected era packs, got %d", len(packs))
	}
	list := findThemeList(w.Content())
	if list == nil {
		t.Fatal("missing theme picker")
	}
	if list.Count != len(packs) {
		t.Fatalf("theme list count=%d want %d", list.Count, len(packs))
	}
	for _, p := range packs {
		found := false
		for i := 0; i < list.Count; i++ {
			if rowName(list.ItemText(i)) == p.Display() {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing theme %q", p.Display())
		}
	}
	if list.MaxOffset() <= 0 {
		t.Fatalf("theme list should overflow at 820×560, view=%v rows=%d", list.LocalBounds().Dy(), list.Count)
	}
	if _, thumb := list.ScrollTrack(); thumb.Empty() {
		t.Fatal("overflow theme list should show scrollbar chrome")
	}
	last := packs[len(packs)-1]
	list.ScrollTo(list.MaxOffset())
	off := list.OffsetY
	list.OnSelect(list.Count - 1)
	a.PumpOnce()
	if p := previewAppearance(t, w); p.Name != last.Name {
		t.Fatalf("preview last theme %+v want %s", p, last.Name)
	}
	// Picking a theme below the fold keeps the list where it was (it used
	// to jump back to the top — real-hardware finding).
	if list = findThemeList(w.Content()); list == nil || list.OffsetY < off-1 {
		t.Fatalf("theme list scrolled back after selection: offset %v want %v", list.OffsetY, off)
	}
	apply := findApply(w.Content())
	if apply == nil || !apply.Enabled() {
		t.Fatal("Apply should stay reachable and enabled after staging")
	}
	if nestedInScroll(apply) {
		t.Fatal("Apply must stay pinned outside the scroll region")
	}

	// Decade filter.
	pickCombo(t, w, "1990s")
	a.PumpOnce()
	list = findThemeListAny(w.Content())
	if list == nil || list.Count == 0 {
		t.Fatal("1990s filter shows nothing")
	}
	for i := 0; i < list.Count; i++ {
		if !strings.HasPrefix(list.ItemText(i), "199") {
			t.Fatalf("1990s filter shows %q", list.ItemText(i))
		}
	}

	// The five sections do not fit in the window, so the column of
	// choices scrolls — and the paths the old About page held are at the
	// foot of it, reachable by scrolling and not by navigating.
	choices := findScrollView(w.Content())
	if choices == nil {
		t.Fatal("the column of choices should scroll")
	}
	if choices.MaxOffset() <= 0 {
		t.Fatalf("the column should overflow at 820×560, content=%v view=%v",
			choices.ContentHeight(), choices.LocalBounds().Dy())
	}
	choices.ScrollTo(choices.MaxOffset())
	a.PumpOnce()
	if !findLabelWith(w.Content(), "Prefs file") {
		t.Error("the prefs path is not at the foot of the column")
	}
	if findApply(w.Content()) == nil || nestedInScroll(findApply(w.Content())) {
		t.Fatal("Apply must stay pinned outside the column")
	}
	// And it stays pinned on a window short enough that the column is
	// almost all scrollbar.
	w.Inject(platform.Event{Kind: platform.EventResize, Width: 720, Height: 520})
	a.PumpOnce()
	if findApply(w.Content()) == nil || nestedInScroll(findApply(w.Content())) {
		t.Fatal("Apply must stay pinned at the minimum window size")
	}
	if got := findScrollView(w.Content()); got == nil || got.MaxOffset() <= 0 {
		t.Fatal("the column should still scroll at 720×520")
	}
}

func TestSettingsAppPaintsPreview(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 1024, Height: 780, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsAppStaged(a, w, "win98"))
	img := w.Capture()
	if img == nil {
		t.Fatal("capture")
	}
	ink := 0
	for y := 80; y < img.Height-40; y++ {
		for x := 300; x < img.Width-20; x++ {
			_, _, _, al := img.PremulAt(x, y)
			if al > 30 {
				ink++
			}
		}
	}
	if ink < 800 {
		t.Fatalf("right pane looks empty (ink=%d)", ink)
	}
	if p := previewAppearance(t, w); p.Name != "win98" {
		t.Fatalf("staged preview %+v", p)
	}
	if a.Look().Name() == "win98" {
		t.Fatal("staging must not re-theme Settings")
	}
	if findApply(w.Content()) == nil || !findApply(w.Content()).Enabled() {
		t.Fatal("Apply should be enabled with a staged theme")
	}
	if findThemeList(w.Content()) == nil {
		t.Fatal("missing theme picker")
	}
}

// ---- helpers --------------------------------------------------------------------

// rowName is the theme name of a browser row ("1995  ·  Windows 95" →
// "Windows 95"); other lists' rows come back unchanged.
func rowName(text string) string {
	if i := strings.LastIndex(text, "·"); i >= 0 {
		return strings.TrimSpace(text[i+len("·"):])
	}
	return text
}

// repoRoot is the checkout this test runs from, found by walking up to
// the go.mod rather than counting "..": the app has moved once already.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		up := filepath.Dir(dir)
		if up == dir {
			t.Fatal("no go.mod above the test's directory")
		}
		dir = up
	}
}

func installSettingsIconSet(t *testing.T, xdg, name string) {
	t.Helper()
	src := filepath.Join(repoRoot(t), "icons", name)
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

// previewAppearance is the appearance the live preview is painting.
func previewAppearance(t *testing.T, w *app.Window) style.Appearance {
	t.Helper()
	var scope *widgets.ThemeScope
	widget.Walk(w.Content(), func(c widget.Component) {
		if s, ok := c.(*widgets.ThemeScope); ok {
			scope = s
		}
	})
	if scope == nil {
		t.Fatal("no theme preview")
	}
	return style.LookAppearance(scope.Theme())
}

func insidePreview(c widget.Component) bool {
	for p := c; p != nil; p = p.Parent() {
		if _, ok := p.(*widgets.ThemeScope); ok {
			return true
		}
	}
	return false
}

func findMenuBarOutsidePreview(root widget.Component) bool {
	found := false
	widget.Walk(root, func(c widget.Component) {
		if _, ok := c.(*widgets.MenuBar); ok && !insidePreview(c) {
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
		if b, ok := c.(*widgets.Button); ok && b.Text == text && !insidePreview(c) {
			btn = b
		}
	})
	return btn
}

// findCombo is the Settings combo (outside the preview) offering item.
func findCombo(root widget.Component, item string) *widgets.ComboBox {
	var cb *widgets.ComboBox
	widget.Walk(root, func(c widget.Component) {
		if box, ok := c.(*widgets.ComboBox); ok && !insidePreview(c) {
			for _, it := range box.Items {
				if it == item {
					cb = box
				}
			}
		}
	})
	return cb
}

func pickCombo(t *testing.T, w *app.Window, item string) {
	t.Helper()
	cb := findCombo(w.Content(), item)
	if cb == nil || cb.OnChange == nil {
		t.Fatalf("no combo offering %q", item)
	}
	for i, it := range cb.Items {
		if it == item {
			cb.OnChange(i)
			return
		}
	}
}

func listed(root widget.Component, name string) bool {
	found := false
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil && !insidePreview(c) {
			for i := 0; i < l.Count; i++ {
				if rowName(l.ItemText(i)) == name {
					found = true
				}
			}
		}
	})
	return found
}

func findThemeList(root widget.Component) *widgets.ListView {
	var list *widgets.ListView
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil && !insidePreview(c) {
			for i := 0; i < l.Count; i++ {
				if strings.Contains(l.ItemText(i), "Classic 95") {
					list = l
					return
				}
			}
		}
	})
	return list
}

// findThemeListAny is the theme browser whatever its filter shows. It is
// the only list Settings owns now that the pages sidebar and the packs
// lists have gone; the preview's own lists are excluded as ever.
func findThemeListAny(root widget.Component) *widgets.ListView {
	var list *widgets.ListView
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil && !insidePreview(c) && l.Count > 0 {
			list = l
		}
	})
	return list
}

// clickTheme selects a theme in the browser by its display name (rows are
// "1995  ·  Windows 95"; an exact suffix wins over a longer name).
func clickTheme(t *testing.T, w *app.Window, label string) {
	t.Helper()
	var found *widgets.ListView
	idx := -1
	widget.Walk(w.Content(), func(c widget.Component) {
		if l, ok := c.(*widgets.ListView); ok && l.ItemText != nil && l.OnSelect != nil && !insidePreview(c) {
			for i := 0; i < l.Count; i++ {
				if rowName(l.ItemText(i)) == label {
					found, idx = l, i
				}
			}
		}
	})
	if found == nil {
		t.Fatalf("no theme %q", label)
	}
	found.OnSelect(idx)
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
	var btn *widgets.Button
	widget.Walk(root, func(c widget.Component) {
		if b, ok := c.(*widgets.Button); ok && b.Text == text {
			btn = b
		}
	})
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

func clickApply(t *testing.T, w *app.Window) {
	t.Helper()
	apply := findApply(w.Content())
	if apply == nil || apply.OnClick == nil {
		t.Fatal("no Apply button")
	}
	apply.OnClick()
}

func findSwitch(root widget.Component, text string) *widgets.Switch {
	var sw *widgets.Switch
	widget.Walk(root, func(c widget.Component) {
		if s, ok := c.(*widgets.Switch); ok && s.Text == text && !insidePreview(c) {
			sw = s
		}
	})
	return sw
}

// findLabelWith reports whether any label outside the preview holds part.
func findLabelWith(root widget.Component, part string) bool {
	found := false
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.Label); ok && strings.Contains(l.Text, part) && !insidePreview(c) {
			found = true
		}
	})
	return found
}

// previewPanelTitle is the caption over the live preview: "Preview — " and
// the pack it is drawing.
func previewPanelTitle(t *testing.T, w *app.Window) string {
	t.Helper()
	var box *widgets.Panel
	widget.Walk(w.Content(), func(c widget.Component) {
		if p, ok := c.(*widgets.Panel); ok && p.Window && box == nil {
			box = p
		}
	})
	if box == nil {
		t.Fatal("no preview panel")
	}
	return box.Title
}

// Following the desktop is staged like any other option: the preview
// shows the sibling for the desktop's scheme and says so; Apply saves it.
func TestSettingsFollowDesktop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(app.ColorSchemeEnv, "dark")
	defer style.SetDesktopColorScheme(style.SchemeNoPreference)
	if err := style.SaveAppearance(style.Appearance{Name: "breeze", Theme: style.ThemeLight}); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 780)
	const label = "Follow the desktop's colours"
	sw := findSwitch(w.Content(), label)
	if sw == nil {
		t.Fatal("no follow-the-desktop switch")
	}
	if sw.On {
		t.Fatal("switch on before it was chosen")
	}
	sw.OnChange(true)
	a.PumpOnce()
	if got := previewAppearance(t, w).Name; got != "breeze-night" {
		t.Fatalf("preview %s, want breeze-night", got)
	}
	// The preview's caption names the pack it is really drawing, which is
	// where the user sees what the desktop's scheme did to their choice —
	// and the switch and the caption are now on the same page, a hand's
	// width apart.
	if got := previewPanelTitle(t, w); got != "Preview — Breeze Dark" {
		t.Fatalf("the preview is captioned %q", got)
	}
	clickApply(t, w)
	a.PumpOnce()
	saved := style.LoadAppearance()
	if !saved.FollowDesktop || saved.Name != "breeze" {
		t.Fatalf("saved %+v", saved)
	}
	if c, ok := a.Look().(*style.Classic); !ok || c.Pack() != "breeze-night" {
		t.Fatalf("applied look %v", a.Look())
	}
	if sw := findSwitch(w.Content(), label); sw == nil || !sw.On {
		t.Fatal("switch should show the saved choice")
	}
}

// "Use system title bar and borders" writes look.json's decorations, and
// running apps switch their windows at once.
func TestSettingsSystemTitleBarSwitch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w := openSettings(t, 1024, 780)
	const label = "Use system title bar and borders"
	sw := findSwitch(w.Content(), label)
	if sw == nil {
		t.Fatal("no system title bar switch")
	}
	if sw.On {
		t.Fatal("the toolkit's frame is the default for windows with a title bar")
	}
	sw.OnChange(true)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance().Decorations; got != style.DecorationsSystem {
		t.Fatalf("saved decorations %q", got)
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil || !strings.Contains(string(raw), `"decorations": "system"`) {
		t.Fatalf("look.json %s %v", raw, err)
	}
	if sw := findSwitch(w.Content(), label); sw == nil || !sw.On {
		t.Fatal("switch should show the saved choice")
	}
	findSwitch(w.Content(), label).OnChange(false)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance().Decorations; got != style.DecorationsAuto {
		t.Fatalf("back to auto: %q", got)
	}
}

// "Place window buttons as the theme does" writes look.json's
// captionButtons.
func TestSettingsThemeCaptionButtonsSwitch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w := openSettings(t, 1024, 780)
	const label = "Place window buttons as the theme does"
	sw := findSwitch(w.Content(), label)
	if sw == nil || sw.On {
		t.Fatalf("switch %v: the desktop's layout is the default", sw)
	}
	sw.OnChange(true)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance().CaptionButtons; got != style.CaptionButtonsTheme {
		t.Fatalf("saved captionButtons %q", got)
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil || !strings.Contains(string(raw), `"captionButtons": "theme"`) {
		t.Fatalf("look.json %s %v", raw, err)
	}
	findSwitch(w.Content(), label).OnChange(false)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if raw, _ := os.ReadFile(style.AppearancePath()); strings.Contains(string(raw), "captionButtons") {
		t.Fatalf("the desktop's layout is left out of look.json: %s", raw)
	}
}

// Picking an icon set shows what it draws, at the head of the Themes
// page, beside the theme it is being chosen for. The strip is in a theme
// scope of its own, so it follows the staged set and the staged size
// without Apply, and its caption names the set it is showing.
func TestSettingsIconSetPreview(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	installSettingsIconSet(t, dir, "lucide")
	a, w := openSettings(t, 1024, 780)

	box := findPanelTitled(w.Content(), "Preview — Classic")
	if box == nil {
		t.Fatal("no icon preview for the selected set on the Themes page")
	}
	// Every stock icon is in it, named by the action it stands for.
	shown := iconStripIcons(box)
	for _, want := range style.AllToolIcons() {
		if !shown[want] {
			t.Errorf("the icon preview has no %s", want.Label())
		}
	}
	scope := scopeAround(t, box)
	if style.LookAppearance(scope.Theme()).Icons != style.IconSetClassic {
		t.Fatal("the preview is not drawing the staged icon set")
	}

	// Picking another set repaints it where it stands and renames it.
	pickCombo(t, w, "Lucide")
	a.PumpOnce()
	if style.LookAppearance(scopeAround(t, box).Theme()).Icons != style.IconSetLucide {
		t.Fatal("the icon preview did not follow the staged set")
	}
	if box.Title != "Preview — Lucide" {
		t.Fatalf("the icon preview is captioned %q", box.Title)
	}
	if style.LoadAppearance().Icons != style.IconSetClassic {
		t.Fatal("previewing an icon set wrote look.json")
	}

	// The size the glyphs are drawn at is staged beside the set, and the
	// strip shows that size rather than a fixed one: the bars grow with
	// it. (Loose tool buttons do not — their icon is capped by the
	// control height — which is why the strip is made of bars.)
	before := iconStripBar(t, box).Bounds().Dy()
	pickCombo(t, w, "Large")
	a.PumpOnce()
	if got := style.LookAppearance(scopeAround(t, box).Theme()).IconSize; got != style.IconSizeLarge {
		t.Fatalf("the icon preview is drawing size %v", got)
	}
	if after := iconStripBar(t, box).Bounds().Dy(); after <= before {
		t.Errorf("the strip did not grow with the icon size (%v then %v)", before, after)
	}

	// Staging a theme leaves the icon set alone: a pack's theme.json
	// carries no icon preference — the field is read and ignored, look.json
	// owns icons — so picking a pack changes only the surface under the
	// strip.
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	if box.Title != "Preview — Lucide" {
		t.Fatalf("staging a theme renamed the icon preview to %q", box.Title)
	}

	// And the icons area follows a set staged by something other than its
	// own chooser — Apply restages the whole appearance — with the
	// chooser showing it and the strip drawing it.
	clickApply(t, w)
	a.PumpOnce()
	if findPanelTitled(w.Content(), "Preview — Lucide") == nil {
		t.Error("the icon strip lost the applied set across a rebuild")
	}
	if combo := findCombo(w.Content(), "Lucide"); combo == nil || combo.Items[combo.Selected] != "Lucide" {
		t.Error("the icon chooser does not show the applied set")
	}
}

// iconStripIcons is every icon the preview strip draws.
func iconStripIcons(box widget.Component) map[style.ToolIcon]bool {
	out := map[style.ToolIcon]bool{}
	widget.Walk(box, func(c widget.Component) {
		bar, ok := c.(*widgets.ToolBar)
		if !ok {
			return
		}
		for _, it := range bar.Items() {
			if it != nil && it.Icon != style.IconNone {
				out[it.Icon] = true
			}
		}
	})
	return out
}

// iconStripBar is the first of the strip's tool bars, whose height is the
// size the staged set is being drawn at.
func iconStripBar(t *testing.T, box widget.Component) *widgets.ToolBar {
	t.Helper()
	var bar *widgets.ToolBar
	widget.Walk(box, func(c widget.Component) {
		if b, ok := c.(*widgets.ToolBar); ok && bar == nil {
			bar = b
		}
	})
	if bar == nil {
		t.Fatal("the icon preview has no tool bar")
	}
	return bar
}

// The line beside Apply — "Applied — every uitoolkit app is using this
// look" — is gone from every page; Apply being enabled says it instead.
func TestSettingsHasNoAppliedLine(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, w := openSettings(t, 1024, 780)
	// The phrases in full: the temporary config path the Files section
	// prints carries this test's own name, so a loose match on "Applied"
	// would find itself.
	for _, phrase := range []string{"every uitoolkit app is using this look", "Staged, not applied"} {
		if findLabelWith(w.Content(), phrase) {
			t.Errorf("the applied/staged line is back (%q)", phrase)
		}
	}
}

// With that line gone, a failure has to find another way to the user: a
// failed Apply puts up an error message box rather than passing in
// silence. (A read-only config directory is the cheapest real failure.)
func TestSettingsApplyFailureIsSaid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	a, w := openSettings(t, 1024, 780)
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	clickApply(t, w)
	a.PumpOnce()
	over := w.Overlay()
	if over == nil {
		t.Fatal("a failed Apply said nothing at all")
	}
	if findPanelTitled(over, "Apply failed") == nil {
		t.Error("the message box does not say what failed")
	}
	if !findLabelWith(over, dir) {
		t.Error("the message box does not say why")
	}
	if style.LoadAppearance().Name == "win95" {
		t.Fatal("the failed Apply was reported as having worked")
	}
}

// findPanelTitled is the panel with exactly this caption.
func findPanelTitled(root widget.Component, title string) *widgets.Panel {
	var box *widgets.Panel
	widget.Walk(root, func(c widget.Component) {
		if p, ok := c.(*widgets.Panel); ok && p.Title == title {
			box = p
		}
	})
	return box
}

func findToolButton(root widget.Component, name string) *widgets.ToolButton {
	var btn *widgets.ToolButton
	widget.Walk(root, func(c widget.Component) {
		if b, ok := c.(*widgets.ToolButton); ok && b.AccessibleName() == name {
			btn = b
		}
	})
	return btn
}

// scopeAround is the theme scope c is drawn inside.
func scopeAround(t *testing.T, c widget.Component) *widgets.ThemeScope {
	t.Helper()
	for p := c.Parent(); p != nil; p = p.Parent() {
		if sc, ok := p.(*widgets.ThemeScope); ok {
			return sc
		}
	}
	t.Fatalf("%T is in no theme scope", c)
	return nil
}

func pickListItem(t *testing.T, list *widgets.ListView, text string) {
	t.Helper()
	for i := 0; i < list.Count; i++ {
		if list.ItemText(i) == text {
			list.OnSelect(i)
			return
		}
	}
	t.Fatalf("no list row %q", text)
}

// -page names a section of the one page now, and the names of the four
// pages Settings used to have still resolve: they are in scripts, in the
// atlas tooling and in the docs of two releases.
func TestSettingsPageNamesStillResolve(t *testing.T) {
	for name, want := range map[string]int{
		"":                 sectionTheme,
		"themes":           sectionTheme,
		"packs":            sectionTheme,
		"packs & icons":    sectionTheme,
		"theme packs":      sectionTheme,
		"Theme":            sectionTheme,
		"appearance":       sectionShape,
		"icons":            sectionShape,
		"icon sets":        sectionShape,
		"Shape and weight": sectionShape,
		"about":            sectionFiles,
		"files":            sectionFiles,
		"behaviour":        sectionBehaviour,
		"behavior":         sectionBehaviour,
		"windows":          sectionBehaviour,
		"desktop":          sectionDesktop,
		"nonsense":         sectionTheme,
	} {
		if got := SettingsPage(name); got != want {
			t.Errorf("-page %q opens section %d (%s), want %d (%s)",
				name, got, settingsSections[got], want, settingsSections[want])
		}
	}
}

// And the name is not only resolved: the column of choices opens
// scrolled to that section, which is what -page did when it switched
// pages.
func TestSettingsPageOpensScrolledToItsSection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	open := func(page string) (*app.Application, *app.Window) {
		t.Helper()
		a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: 1, DisableLookWatch: true})
		w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 1024, Height: 780, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { w.Close() })
		w.SetContent(SettingsAppOpen(a, w, "", page))
		a.PumpOnce()
		return a, w
	}
	_, top := open("")
	if got := findScrollView(top.Content()).OffsetY; got != 0 {
		t.Errorf("with no -page the column opens at %v, want the top", got)
	}
	for _, page := range []string{"about", "behaviour", "appearance"} {
		_, w := open(page)
		sv := findScrollView(w.Content())
		if sv == nil {
			t.Fatalf("%s: no column of choices", page)
		}
		if sv.OffsetY <= 0 {
			t.Errorf("-page %s left the column at the top (%v)", page, sv.OffsetY)
		}
	}
	// The old About page is the Files section, and it is the foot of the
	// column, so -page about scrolls all the way down.
	_, w := open("about")
	sv := findScrollView(w.Content())
	if sv.OffsetY < sv.MaxOffset()-1 {
		t.Errorf("-page about stopped at %v of %v", sv.OffsetY, sv.MaxOffset())
	}
}
