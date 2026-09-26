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
	// One page: the icon set, the size its glyphs are drawn at and the
	// shape of the window's corners are all on it, with no navigating to
	// do first — four choosers after the options, over the preview.
	for _, opt := range []string{"16", "24", "32", "Classic", "Sharp", "Theme shape", "Round", "Square"} {
		if findCombo(w.Content(), opt) == nil {
			t.Fatalf("the chooser offering %q is not on the page", opt)
		}
	}
	// And so is every option the Appearance page used to hold, in the row
	// over the preview, under the short word each one wears now.
	for _, opt := range []string{"Animations", "OS colors", "OS open/save dialogs", "OS window borders"} {
		if findOption(w.Content(), opt) == nil {
			t.Fatalf("the %q option is not on the page", opt)
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

	pickCorner(t, w, "Round")
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

// Delete theme… is gone, the call made for Delete icon set… the release
// before: a button that destroys a directory, standing under the browser
// grey for all 129 built-in packs and live for the handful of packs the
// user exported. A pack is a folder and rm -r removes it.
// style.DeleteUserTheme and style.AfterUserThemeDeleted stay — they are
// public toolkit API and an application may want the operation — and
// Settings does not call them any more. Export current theme… stays: it
// is the half of that pair that makes something.
func TestSettingsHasNoDeleteThemeButton(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}); err != nil {
		t.Fatal(err)
	}
	if _, err := style.ExportAppearance("ocean", style.LoadAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 780)
	if del := findButton(w.Content(), "Delete theme…"); del != nil {
		t.Error("Delete theme… is back under the browser")
	}
	// Not hiding behind a built-in pack being staged either: it was grey
	// for one of those and live for one of the user's own, so stage
	// theirs and look again.
	clickTheme(t, w, "ocean")
	a.PumpOnce()
	if got := previewAppearance(t, w).Name; got != "ocean" {
		t.Fatalf("the user's pack is staged as %q", got)
	}
	if del := findButton(w.Content(), "Delete theme…"); del != nil {
		t.Error("Delete theme… comes back when a user pack is staged")
	}
	// The pack is still on disk and still in the browser: only the
	// button went.
	if _, err := os.Stat(style.ThemeFile("ocean")); err != nil {
		t.Errorf("the user's pack left disk: %v", err)
	}
	if !listed(w.Content(), "ocean") {
		t.Error("the user's pack is not in the browser")
	}
	if findButton(w.Content(), "Export current theme…") == nil {
		t.Error("Export current theme… went with it")
	}
	// And nothing else that acts on a pack crept into the column: one
	// button under the list, and it is Export.
	widget.Walk(browserColumn(t, w), func(c widget.Component) {
		if b, ok := c.(*widgets.Button); ok && b.Text != "Export current theme…" {
			t.Errorf("the column carries a %q button", b.Text)
		}
	})
}

// The column is four bare controls, in this order and with nothing
// around them: the search field, the decade filter, the list of packs,
// and Export. The Theme group box that held them is gone and so is the
// paragraph over the search field; so are the "Settings" heading at the
// top of the column and the version label under it.
//
// What names the column is the two controls a screen reader would
// otherwise meet unnamed — the list is Themes and the field is Search
// themes — which is where those names belonged all along: a group box's
// legend names a group, not the list inside it, and a11y.Check fails an
// unnamed list whatever is written above it.
//
// And the list runs to the foot of the column. It was held to 252
// pixels, which left about 250 of nothing under the buttons; it takes
// that now, at every size.
func TestTheColumnIsBareControls(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{1024, 860}, {720, 520}} {
		_, w := openSettings(t, size[0], size[1])
		col := browserColumn(t, w)
		kids := col.Children()
		if len(kids) != 4 {
			t.Fatalf("at %v: the column has %d children, want search, filter, list, Export", size, len(kids))
		}
		if tf, ok := kids[0].(*widgets.TextField); !ok || tf.Placeholder != "Search themes" {
			t.Errorf("at %v: the column opens with %T, want the search field", size, kids[0])
		}
		if cb, ok := kids[1].(*widgets.ComboBox); !ok || cb.AccessibleName() != "Decade" {
			t.Errorf("at %v: the second control is %T, want the decade filter", size, kids[1])
		}
		list, ok := kids[2].(*widgets.ListView)
		if !ok {
			t.Fatalf("at %v: the third control is %T, want the theme list", size, kids[2])
		}
		export, ok := kids[3].(*widgets.Button)
		if !ok || export.Text != "Export current theme…" {
			t.Fatalf("at %v: the column ends in %T, want Export current theme…", size, kids[3])
		}
		// No group box around them, no heading over them, no version
		// under it.
		if p := findPanelTitled(w.Content(), "Theme"); p != nil {
			t.Errorf("at %v: the Theme group box is back", size)
		}
		widget.Walk(col, func(c widget.Component) {
			if p, ok := c.(*widgets.Panel); ok {
				t.Errorf("at %v: the column holds a %q panel", size, p.Title)
			}
			if l, ok := c.(*widgets.Label); ok {
				t.Errorf("at %v: the column holds the label %q", size, l.Text)
			}
		})
		if findLabelWith(w.Content(), "v"+uitoolkit.Version) {
			t.Errorf("at %v: the version label is back on the page", size)
		}
		// The two names a screen reader has instead of a legend.
		if got := list.AccessibleName(); got != "Themes" {
			t.Errorf("at %v: the theme list is called %q", size, got)
		}
		if got := kids[0].(*widgets.TextField).AccessibleName(); got != "Search themes" {
			t.Errorf("at %v: the search field is called %q", size, got)
		}
		// The list takes the foot of the column: the gap under it is the
		// column's own gap and no more, and what is under Export is the
		// column's padding.
		if gap := export.Bounds().Min.Y - list.Bounds().Max.Y; gap > 16 {
			t.Errorf("at %v: %v px of nothing between the list and Export", size, gap)
		}
		if tail := col.LocalBounds().Dy() - export.Bounds().Max.Y; tail > 16 {
			t.Errorf("at %v: %v px of nothing under Export", size, tail)
		}
		// And it is most of the column rather than a ninth of it.
		if share := list.LocalBounds().Dy() / col.LocalBounds().Dy(); share < 0.55 {
			t.Errorf("at %v: the list is %.0f%% of the column", size, 100*share)
		}
		// The whole of it scrolls inside the list, not under it.
		if list.MaxOffset() <= 0 {
			t.Errorf("at %v: the list of 131 packs does not overflow", size)
		}
		if sv := findScrollView(w.Content()); sv != nil {
			t.Errorf("at %v: the column is back in a scroll view", size)
		}
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
	pickCombo(t, w, "32")
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
	if inBrowserColumn(w.Content(), apply) {
		t.Fatal("Apply must stay pinned outside the column")
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

	// The list is the only thing on this page that scrolls: the column
	// around it does not, because there is nothing in it that a window
	// this size cannot hold — a field, a chooser, a button, and a list
	// that takes whatever they leave. Two scrollbars an inch apart was
	// what holding the list to 252 pixels inside a scrolling column
	// bought, and 250 pixels of nothing under the buttons was what it
	// cost.
	if sv := findScrollView(w.Content()); sv != nil {
		t.Error("the column is back in a scroll view")
	}
	if b := findButton(w.Content(), "Export current theme…"); b == nil || !inBrowserColumn(w.Content(), b) {
		t.Error("Export current theme… is not at the foot of the column")
	}
	if findButton(w.Content(), "Delete theme…") != nil {
		t.Error("Delete theme… is back under the browser")
	}
	for _, opt := range []string{"Animations", "OS colors"} {
		if box := findOption(w.Content(), opt); box == nil || inBrowserColumn(w.Content(), box) {
			t.Errorf("the %q option is in the browser column", opt)
		}
	}
	// The paths are not in this column at all: they are under the
	// preview, where nothing scrolls and they are always on screen.
	if !findLabelWith(w.Content(), "Prefs") {
		t.Error("the prefs path is not on the page")
	}
	if files := filesBlock(t, w); inBrowserColumn(w.Content(), files) {
		t.Error("the paths are in the browser column")
	}
	if findApply(w.Content()) == nil || inBrowserColumn(w.Content(), findApply(w.Content())) {
		t.Fatal("Apply must stay pinned outside the column")
	}
	// And it stays pinned on a window short enough that the list is
	// almost all scrollbar.
	w.Inject(platform.Event{Kind: platform.EventResize, Width: 720, Height: 520})
	a.PumpOnce()
	if findApply(w.Content()) == nil || inBrowserColumn(w.Content(), findApply(w.Content())) {
		t.Fatal("Apply must stay pinned at the minimum window size")
	}
	if got := findThemeListAny(w.Content()); got == nil || got.MaxOffset() <= 0 {
		t.Fatal("the theme list should still scroll at 720×520")
	}
	if sv := findScrollView(w.Content()); sv != nil {
		t.Error("the column scrolls at 720×520")
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

// findCombo is a Settings chooser offering item. Two of them — the icon
// set and its size — are on the preview's tool bar now, inside the
// preview's theme scope, so this can no longer tell Settings' own
// choosers from the previewed application's by where they are: it tells
// them apart by their name, which every chooser Settings owns has and
// the sample's "Combo box" does not.
func findCombo(root widget.Component, item string) *widgets.ComboBox {
	var cb *widgets.ComboBox
	widget.Walk(root, func(c widget.Component) {
		box, ok := c.(*widgets.ComboBox)
		if !ok || box.AccessibleName() == "" {
			return
		}
		for _, it := range box.Items {
			if it == item {
				cb = box
			}
		}
	})
	return cb
}

// namedCombo is the chooser a screen reader calls name.
func namedCombo(root widget.Component, name string) *widgets.ComboBox {
	var cb *widgets.ComboBox
	widget.Walk(root, func(c widget.Component) {
		if box, ok := c.(*widgets.ComboBox); ok && box.AccessibleName() == name {
			cb = box
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

// cornerPicker is the window-shape chooser: the third of the three
// settings that say what a pack is drawn with, beside the icon set and
// the icon size on the page over the preview.
func cornerPicker(t *testing.T, w *app.Window) *widgets.ComboBox {
	t.Helper()
	cb := namedCombo(w.Content(), "Window corners")
	if cb == nil {
		t.Fatal("no window corners chooser on the preview")
	}
	return cb
}

// pickCorner chooses a window shape the way the chooser does.
func pickCorner(t *testing.T, w *app.Window, text string) {
	t.Helper()
	cb := cornerPicker(t, w)
	for i, it := range cb.Items {
		if it == text {
			if cb.OnChange == nil {
				t.Fatalf("the %q corner style does nothing", text)
			}
			cb.OnChange(i)
			return
		}
	}
	t.Fatalf("no %q in the window corners chooser", text)
}

// cornerShown is what that chooser reads.
func cornerShown(t *testing.T, w *app.Window) string {
	t.Helper()
	cb := cornerPicker(t, w)
	if cb.Selected < 0 || cb.Selected >= len(cb.Items) {
		return ""
	}
	return cb.Items[cb.Selected]
}

// previewTools is the previewed application's own tool bar: New, Open,
// Save and the rest of the sample's commands. It is the only tool bar on
// the page — the bar of live settings that stood over the sample's menu
// bar for a release is gone, and what was on it is on the page now.
func previewTools(t *testing.T, w *app.Window) *widgets.ToolBar {
	t.Helper()
	var bars []*widgets.ToolBar
	widget.Walk(w.Content(), func(c widget.Component) {
		if b, ok := c.(*widgets.ToolBar); ok {
			bars = append(bars, b)
		}
	})
	if len(bars) != 1 {
		t.Fatalf("the page has %d tool bars, want the sample's own", len(bars))
	}
	return bars[0]
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

// browserColumn is the left-hand column: the theme list's parent, which
// is the column itself now that the list is a child of it rather than of
// a height box inside a group box inside a scroll view.
func browserColumn(t *testing.T, w *app.Window) widget.Component {
	t.Helper()
	list := findThemeListAny(w.Content())
	if list == nil {
		t.Fatal("no theme browser")
	}
	col := list.Parent()
	if col == nil {
		t.Fatal("the theme list is not in a column")
	}
	return col
}

// inBrowserColumn reports whether c is in that column. It was
// nestedInScroll: the column was a scroll view, so being inside that view
// was what "in the column" meant. The column does not scroll any more —
// the list takes what the three controls around it leave — so what says a
// thing is in the column is the column itself.
func inBrowserColumn(root, c widget.Component) bool {
	list := findThemeListAny(root)
	if list == nil || c == nil {
		return false
	}
	col := list.Parent()
	for p := c; p != nil; p = p.Parent() {
		if p == col {
			return true
		}
	}
	return false
}

// findScrollView is what says the column is not back in one: Settings owns
// no scroll view at all now.
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

// filesBlock is the three paths under the preview: the column the Prefs
// line stands in. They were a group box with "Files" on its legend until
// the legend and the frame were 34 pixels the preview wanted more — the
// same call the Theme legend lost in the column on the left.
func filesBlock(t *testing.T, w *app.Window) widget.Component {
	t.Helper()
	var found widget.Component
	widget.Walk(w.Content(), func(c widget.Component) {
		l, ok := c.(*widgets.Label)
		if !ok || l.Text != "Prefs" || insidePreview(c) {
			return
		}
		if row := l.Parent(); row != nil {
			found = row.Parent()
		}
	})
	if found == nil {
		t.Fatal("no block of paths under the preview")
	}
	return found
}

// findRowLabel is the word in front of a chooser: a label of Settings'
// own with exactly this text.
func findRowLabel(root widget.Component, text string) *widgets.Label {
	var lbl *widgets.Label
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.Label); ok && l.Text == text && !insidePreview(c) {
			lbl = l
		}
	})
	return lbl
}

// findOption is one of the four check boxes in the row over the preview,
// by the short word on it. Settings' own, never the sample's: the
// previewed application has check boxes of its own on its Controls tab.
func findOption(root widget.Component, text string) *widgets.Checkbox {
	var box *widgets.Checkbox
	widget.Walk(root, func(c widget.Component) {
		if b, ok := c.(*widgets.Checkbox); ok && b.Text == text && !insidePreview(c) {
			box = b
		}
	})
	return box
}

// optionsRow is the folding row the settings stand in: the four check
// boxes and, after them, the four choosers.
func optionsRow(t *testing.T, w *app.Window) *widgets.Wrap {
	t.Helper()
	var row *widgets.Wrap
	widget.Walk(w.Content(), func(c widget.Component) {
		if r, ok := c.(*widgets.Wrap); ok && row == nil && !insidePreview(c) {
			row = r
		}
	})
	if row == nil {
		t.Fatal("no row of options over the preview")
	}
	return row
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
	const label = "OS colors"
	sw := findOption(w.Content(), label)
	if sw == nil {
		t.Fatal("no desktop-colours option")
	}
	if sw.Checked {
		t.Fatal("ticked before it was chosen")
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
	if sw := findOption(w.Content(), label); sw == nil || !sw.Checked {
		t.Fatal("the option should show the saved choice")
	}
}

// "OS window borders" — "OS window borders: the desktop's title bar and
// borders" to a screen reader — writes look.json's decorations, and running apps switch
// their windows at once. It writes the caption buttons with them: see
// TestSettingsOSWindowBordersPlacesTheCaptionButtons.
func TestSettingsSystemTitleBarSwitch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w := openSettings(t, 1024, 780)
	const label = "OS window borders"
	sw := findOption(w.Content(), label)
	if sw == nil {
		t.Fatal("no system-frame option")
	}
	if sw.Checked {
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
	if sw := findOption(w.Content(), label); sw == nil || !sw.Checked {
		t.Fatal("the option should show the saved choice")
	}
	// Unticking is the other half of the box's own sentence — "the
	// toolkit draws the title bar and borders" — so it writes "toolkit",
	// not "auto". Auto leaves every window without a title bar of its own
	// (Settings' own window among them) with the desktop's frame, so the
	// box used to promise the theme's borders and change nothing at all.
	findOption(w.Content(), label).OnChange(false)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance().Decorations; got != style.DecorationsToolkit {
		t.Fatalf("unticked writes the toolkit's frame, not %q", got)
	}
	if got := a.Decorations(); got != style.DecorationsToolkit {
		t.Fatalf("the running app switched to %q", got)
	}
	raw, err = os.ReadFile(style.AppearancePath())
	if err != nil || !strings.Contains(string(raw), `"decorations": "toolkit"`) {
		t.Fatalf("look.json %s %v", raw, err)
	}
	if sw := findOption(w.Content(), label); sw == nil || sw.Checked {
		t.Fatal("the option should show unticked for the toolkit's frame")
	}
}

// Where a title bar the toolkit draws puts its caption buttons is not a
// box of its own any more. It was one — "Theme buttons" — and it asked a
// question about a title bar that only exists while "OS window borders" is
// unticked: with the desktop drawing the frame there is no toolkit
// caption to put buttons on, and with the toolkit drawing it the theme
// is the only thing on the page with an opinion about where they go. So
// the rule is implicit, and "OS window borders" writes both halves of it.
func TestSettingsOSWindowBordersPlacesTheCaptionButtons(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w := openSettings(t, 1024, 780)
	if box := findOption(w.Content(), "Theme buttons"); box != nil {
		t.Error("the Theme buttons box is back on the page")
	}
	// Unticked: the toolkit draws the frame, so the theme places the
	// buttons.
	sw := findOption(w.Content(), "OS window borders")
	if sw == nil {
		t.Fatal("no OS window borders option")
	}
	sw.OnChange(false)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance(); got.Decorations != style.DecorationsToolkit || got.CaptionButtons != style.CaptionButtonsTheme {
		t.Fatalf("unticked saved %q / %q", got.Decorations, got.CaptionButtons)
	}
	if got := a.CaptionButtons(); got != style.CaptionButtonsTheme {
		t.Fatalf("the running app places its buttons %q", got)
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil || !strings.Contains(string(raw), `"captionButtons": "theme"`) {
		t.Fatalf("look.json %s %v", raw, err)
	}
	// Ticked: the desktop draws the frame and the desktop's layout is
	// what its buttons are in, which is the default and so is left out of
	// the file altogether.
	findOption(w.Content(), "OS window borders").OnChange(true)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance(); got.Decorations != style.DecorationsSystem || got.CaptionButtons != style.CaptionButtonsDesktop {
		t.Fatalf("ticked saved %q / %q", got.Decorations, got.CaptionButtons)
	}
	if raw, _ := os.ReadFile(style.AppearancePath()); strings.Contains(string(raw), "captionButtons") {
		t.Fatalf("the desktop's layout is left out of look.json: %s", raw)
	}
}

// A look.json written before that box went keeps what it says. The
// preference is public API — style.CaptionButtonsPref, and
// app.Application.SetCaptionButtons for an application that wants to
// choose — so a file that asks for the theme's layout with the desktop's
// frame is not nonsense to be corrected on sight; it is a choice
// Settings no longer offers to make and does not silently unmake either.
// Touching "OS window borders" is what replaces it, because that is the box the
// rule now belongs to.
func TestSettingsKeepsACaptionButtonsPrefItDidNotWrite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	saved := style.DefaultAppearance()
	saved.Decorations, saved.CaptionButtons = style.DecorationsSystem, style.CaptionButtonsTheme
	if err := style.SaveAppearance(saved); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 780)
	if box := findOption(w.Content(), "OS window borders"); box == nil || !box.Checked {
		t.Fatal("the OS window borders box should open ticked for a saved system frame")
	}
	// Stage something else entirely and apply: the preference rides
	// through untouched, because what Apply writes is the appearance
	// Settings loaded with the one field a touched box changed.
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance(); got.Name != "win95" || got.CaptionButtons != style.CaptionButtonsTheme {
		t.Fatalf("applying a pack rewrote the caption buttons: %+v", got)
	}
	// And the box that owns the rule now is what changes it.
	findOption(w.Content(), "OS window borders").OnChange(false)
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance(); got.CaptionButtons != style.CaptionButtonsTheme || got.Decorations != style.DecorationsToolkit {
		t.Fatalf("after unticking: %q / %q", got.Decorations, got.CaptionButtons)
	}
}

// The preview's tool bar is the icon preview: it is drawn in the staged
// set at the staged size. The choosers that stage them are not on it and
// are not in that window at all — they are on the page above it, each
// behind the word that says what it sets. A strip of fifteen loose
// glyphs in the column of choices used to answer this question; a tool
// bar full of icons answers it by being the thing it is showing, and it
// goes on answering it from outside: the whole previewed window is drawn
// in the staged appearance through its theme scope, so the bar follows
// the chooser whether or not the chooser stands on it.
func TestSettingsIconSetPreview(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	installSettingsIconSet(t, dir, "lucide")
	a, w := openSettings(t, 1024, 780)

	bar := previewTools(t, w)
	scope := previewScope(t, w)
	if style.LookAppearance(scope.Theme()).Icons != style.IconSetClassic {
		t.Fatal("the preview is not drawn in the staged icon set")
	}
	// All four choosers are on the page, outside the previewed window.
	for _, name := range []string{"Icons", "Icon size", "Window corners"} {
		combo := namedCombo(w.Content(), name)
		if combo == nil {
			t.Fatalf("no %s chooser", name)
		}
		if insidePreview(combo) {
			t.Errorf("the %s chooser is back inside the preview", name)
		}
	}
	// And the sample's bar has its own commands, Paste included: the
	// choosers cost it that button when they rode on it.
	for _, icon := range []style.ToolIcon{style.IconNew, style.IconOpen, style.IconSave,
		style.IconCut, style.IconCopy, style.IconPaste, style.IconPen, style.IconMail} {
		if i := toolIndex(bar, icon); i < 0 || bar.ItemRect(i).Empty() {
			t.Errorf("the sample's tool bar has no %v button", icon)
		}
	}
	// Its free space is the last thing on it — nothing rides after it —
	// so a bar too narrow for its tools sheds them from the right rather
	// than cutting one in half at the window's edge.
	if i, n := toolIndex(bar, style.IconNone), len(bar.Items()); i != n-1 {
		t.Errorf("the sample's bar has free space at item %d of %d, want it last", i, n)
	}

	// Picking another set repaints the preview where it stands.
	pickCombo(t, w, "Lucide")
	a.PumpOnce()
	if style.LookAppearance(previewScope(t, w).Theme()).Icons != style.IconSetLucide {
		t.Fatal("the preview did not follow the staged set")
	}
	if previewScope(t, w) != scope {
		t.Fatal("staging an icon set rebuilt the page instead of repainting it")
	}
	if style.LoadAppearance().Icons != style.IconSetClassic {
		t.Fatal("previewing an icon set wrote look.json")
	}

	// The size the glyphs are drawn at is staged beside the set, and the
	// sample's bar shows that size rather than a fixed one: its buttons
	// grow with it. (A loose tool button does not — its icon is capped by
	// the control height — which is why the preview of a set is a bar.)
	before := bar.ItemRect(0).Dy()
	pickCombo(t, w, "32")
	a.PumpOnce()
	if got := style.LookAppearance(previewScope(t, w).Theme()).IconSize; got != style.IconSizeLarge {
		t.Fatalf("the preview is drawing size %v", got)
	}
	if after := previewTools(t, w).ItemRect(0).Dy(); after <= before {
		t.Errorf("the bar's buttons did not grow with the icon size (%v then %v)", before, after)
	}

	// Staging a theme leaves the icon set alone: a pack's theme.json
	// carries no icon preference — the field is read and ignored,
	// look.json owns icons — so picking a pack changes only the surface
	// the same glyphs are drawn on.
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	if got := namedCombo(w.Content(), "Icons"); got.Items[got.Selected] != "Lucide" {
		t.Fatalf("staging a theme moved the icon chooser to %q", got.Items[got.Selected])
	}

	// And the choosers follow a set staged by something other than
	// themselves — Apply restages the whole appearance — as the strip
	// they replaced did.
	clickApply(t, w)
	a.PumpOnce()
	if combo := namedCombo(w.Content(), "Icons"); combo == nil || combo.Items[combo.Selected] != "Lucide" {
		t.Error("the icon chooser does not show the applied set")
	}
	if got := style.LookAppearance(previewScope(t, w).Theme()); got.Icons != style.IconSetLucide {
		t.Errorf("the preview lost the applied set across a rebuild: %+v", got)
	}
}

// Delete icon set… is gone. It was the last thing left of the old Packs
// page: a button that destroys a directory, riding at the end of the
// icons path on the one part of this page that was meant to change
// nothing, acting on a set chosen two inches away on the preview's own
// bar. Nothing replaced it — style.DeleteUserIconSet is still there for
// an application that wants it, and a set is still a folder anyone can
// remove.
func TestSettingsHasNoDeleteIconSetButton(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	// A folder of the user's own: a premiere name (lucide) is a built-in
	// set wherever it is installed, and the button was only ever live for
	// one of these.
	installSettingsIconSet(t, dir, "lucide")
	mine := filepath.Join(dir, "uitoolkit", "icons", "mine")
	if err := os.MkdirAll(mine, 0o700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "uitoolkit", "icons", "lucide")
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(mine, e.Name()), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	a, w := openSettings(t, 1024, 780)

	if del := findButton(w.Content(), "Delete icon set…"); del != nil {
		t.Error("Delete icon set… is back on the page")
	}
	// Not hiding behind a user set being staged either: it was grey for a
	// built-in one and live for one of the user's own, so stage theirs
	// and look again.
	pickCombo(t, w, "mine")
	a.PumpOnce()
	if del := findButton(w.Content(), "Delete icon set…"); del != nil {
		t.Error("Delete icon set… comes back when a user set is staged")
	}
	// The paths are three lines and nothing that does anything: they are
	// the one part of the page that changes nothing.
	files := filesBlock(t, w)
	widget.Walk(files, func(c widget.Component) {
		if b, ok := c.(*widgets.Button); ok {
			t.Errorf("Files carries a %q button", b.Text)
		}
	})
	for _, name := range []string{"Prefs", "Themes", "Icons"} {
		if !findLabelWith(w.Content(), name) {
			t.Errorf("the %s path is not under the preview", name)
		}
	}
	// And the set itself is still staged and still choosable: only the
	// button went.
	if got := previewAppearance(t, w).Icons; got != style.IconSetName("mine") {
		t.Errorf("the user's set is staged as %q", got)
	}
}

// The window shape is chosen on the preview's settings bar, beside the
// icon set and the icon size, and what it shows follows the staged
// appearance whoever staged it. It was three radio items in the
// preview's View menu for a release, where nobody found it.
func TestSettingsCornersAreAChooserOnThePreviewsBar(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 780)
	checked := func() string { return cornerShown(t, w) }
	if got := checked(); got != "Theme shape" {
		t.Fatalf("the chooser opens on %q, want the pack's own shape", got)
	}
	pickCorner(t, w, "Square")
	a.PumpOnce()
	if p := previewAppearance(t, w); p.Corners != style.CornersSquare {
		t.Fatalf("the preview is drawn with %v corners", p.Corners)
	}
	if style.LoadAppearance().Corners != style.CornersTheme {
		t.Fatal("choosing a shape wrote look.json")
	}
	if got := checked(); got != "Square" {
		t.Fatalf("the chooser shows %q", got)
	}
	// Apply rebuilds the page: the chooser on the new preview shows what
	// was applied, not what it was built with.
	clickApply(t, w)
	a.PumpOnce()
	if got := checked(); got != "Square" {
		t.Fatalf("after Apply the chooser shows %q", got)
	}
	if style.LoadAppearance().Corners != style.CornersSquare {
		t.Fatalf("Apply did not write the shape: %+v", style.LoadAppearance())
	}
}

// toolIndex is the item of bar drawing icon; IconNone finds its free
// space, if it has any.
func toolIndex(bar *widgets.ToolBar, icon style.ToolIcon) int {
	for i, it := range bar.Items() {
		if icon == style.IconNone {
			if it.Stretch {
				return i
			}
			continue
		}
		if it.Icon == icon {
			return i
		}
	}
	return -1
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

// -page names a section of the one page now, and the names of the four
// pages Settings used to have still resolve: they are in scripts, in the
// atlas tooling and in the docs of two releases.
func TestSettingsPageNamesStillResolve(t *testing.T) {
	for name, want := range map[string]int{
		"":              sectionTheme,
		"themes":        sectionTheme,
		"packs":         sectionTheme,
		"packs & icons": sectionTheme,
		"theme packs":   sectionTheme,
		"Theme":         sectionTheme,
		"appearance":    sectionPreview,
		"icons":         sectionPreview,
		"icon sets":     sectionPreview,
		"corners":       sectionPreview,
		"Preview":       sectionPreview,
		"about":         sectionFiles,
		"files":         sectionFiles,
		"behaviour":     sectionBehaviour,
		"behavior":      sectionBehaviour,
		"windows":       sectionBehaviour,
		"desktop":       sectionBehaviour,
		"nonsense":      sectionTheme,
	} {
		if got := SettingsPage(name); got != want {
			t.Errorf("-page %q opens section %d (%s), want %d (%s)",
				name, got, settingsSections[got], want, settingsSections[want])
		}
	}
}

// And the name is not only resolved: what it names is on the page the
// moment the window opens. Nothing scrolls to it, because nothing on the
// page scrolls out of reach at all — the column that used to is four
// controls and a list that takes what they leave, and the pane beside it
// has never scrolled. The machinery that scrolled a column to a section
// went with the column's scrollbar: what cannot fire is not kept.
func TestSettingsPageOpensWhereItSays(t *testing.T) {
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
	for _, page := range []string{"", "theme", "behaviour", "desktop", "appearance", "about"} {
		_, w := open(page)
		if sv := findScrollView(w.Content()); sv != nil {
			t.Errorf("-page %q: the page scrolls; nothing it can name is under a fold", page)
		}
		// The one thing on the page that scrolls is the theme list, and
		// what scrolls it is the staged pack being row 30 of 129, not
		// anything -page said.
		if list := findThemeListAny(w.Content()); list == nil || list.Selected < 0 {
			t.Errorf("-page %q: the browser opened with nothing staged", page)
		}
	}
	// Each name's section is on screen, which is the whole of what -page
	// promises now.
	_, beh := open("behaviour")
	if box := findOption(beh.Content(), "Animations"); box == nil || box.LocalBounds().Empty() {
		t.Error("-page behaviour does not show the options")
	}
	_, colours := open("desktop")
	if box := findOption(colours.Content(), "OS colors"); box == nil || box.LocalBounds().Empty() {
		t.Error("-page desktop does not show the colours option")
	}
	_, files := open("about")
	if !findLabelWith(files.Content(), "Prefs") {
		t.Error("-page about does not show the paths")
	}
	if _, w := open("appearance"); namedCombo(w.Content(), "Icons") == nil {
		t.Error("-page appearance does not show the icon chooser")
	}
	_, themes := open("themes")
	if findThemeList(themes.Content()) == nil {
		t.Error("-page themes does not show the browser")
	}
}

// The settings stand in one folding row over the preview: the four
// on/off options the Behaviour panel held in the column — with the one
// that says where the colours come from, which was already up here on
// its own — and then the four choosers, three that came out of the
// previewed window and the renderer, which was never anywhere but
// UITK_PAINT.
//
// The options are check boxes rather than switches, and they wear a
// short word rather than the sentence each had in the column. Both are
// the price of standing over the preview: the pane is 453 logical pixels
// at the 720x520 minimum, a switch's pill is 42 of them before its word,
// and every line this row takes is a line off the window the page is
// about. The row folds — two lines at the default size, three at the
// minimum — and what it must never do is take enough of the pane for the
// preview to stop being the biggest thing in it.
func TestTheOptionsRowOverThePreview(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 860)
	row := optionsRow(t, w)
	// The words, in the order they are read.
	want := []string{"Animations", "OS colors", "OS open/save dialogs", "OS window borders"}
	var got []string
	widget.Walk(row, func(c widget.Component) {
		if b, ok := c.(*widgets.Checkbox); ok {
			got = append(got, b.Text)
		}
	})
	if len(got) != len(want) {
		t.Fatalf("the row carries %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("option %d is %q, want %q", i, got[i], want[i])
		}
	}
	// Every word is the start of what its option is called to a screen
	// reader, never a second name for it — the rule the preview's own
	// settings bar follows a hand's width below.
	names := map[string]string{
		"Animations":           "Animations",
		"OS open/save dialogs": "OS open/save dialogs: the desktop's own Open and Save",
		"OS window borders":    "OS window borders: the desktop's title bar and borders",
		"OS colors":            "OS colors: follow the desktop's light or dark mode and its accent",
	}
	for word, name := range names {
		box := findOption(w.Content(), word)
		if box == nil {
			t.Fatalf("no %q option on the page", word)
		}
		if box.AccessibleName() != name {
			t.Errorf("%q is called %q to a screen reader, want %q", word, box.AccessibleName(), name)
		}
		if !strings.Contains(strings.ToLower(name), strings.ToLower(word)) {
			t.Errorf("the box says %q and a screen reader says %q: two names for one control", word, name)
		}
		if box.AccessibleDescription() == "" {
			t.Errorf("the %q option says nothing about itself: the word alone is a riddle", word)
		}
	}
	// Settings' own, not the preview's, and outside the column that
	// scrolls: they must not be able to scroll away from the window they
	// are about.
	if insidePreview(row) || inBrowserColumn(w.Content(), row) {
		t.Error("the options row is inside the preview or in the column that scrolls")
	}
	if widget.DeviceOrigin(row).Y >= widget.DeviceOrigin(previewScope(t, w)).Y {
		t.Error("the options row is not over the preview")
	}
	// Nothing of Settings' own is a switch any more: five pills would not
	// fit, and nothing on this page is live before Apply.
	widget.Walk(w.Content(), func(c widget.Component) {
		if sw, ok := c.(*widgets.Switch); ok && !insidePreview(c) {
			t.Errorf("a %q switch is back on the page", sw.Text)
		}
	})
	// Two lines while the pane is wide, and they are the two lines the
	// arrangement is named for: the four options fill the first and the
	// four choosers fall onto the second by themselves. They are one
	// wrapping row rather than two rows of their own because two rows
	// each fold on their own account, which is a fourth line at the
	// 720x520 minimum and a preview that stops being what the pane is
	// for.
	if n := rowLines(row); n != 2 {
		t.Errorf("the settings stand on %d lines in a %v pane; they fold onto two", n, row.LocalBounds().Dx())
	}
	tops := map[float32]bool{}
	for _, word := range want {
		tops[findOption(w.Content(), word).Bounds().Min.Y] = true
	}
	if len(tops) != 1 {
		t.Errorf("the four options are on %d lines at 1024x860; they fit on one", len(tops))
	}
	for _, name := range []string{"Window corners", "Icons", "Icon size", "Paint renderer"} {
		cb := namedCombo(w.Content(), name)
		if cb == nil {
			t.Fatalf("no %q chooser on the page", name)
		}
		if insidePreview(cb) {
			t.Errorf("the %q chooser is inside the preview", name)
		}
		for top := range tops {
			if widget.DeviceOrigin(cb).Y < widget.DeviceOrigin(row).Y+top+1 {
				t.Errorf("the %q chooser shares the options' line at 1024x860", name)
			}
		}
	}
	// And every chooser has the word that says what it sets in front of
	// it, on the same line, with the word inside the name a screen reader
	// says.
	for i, name := range []string{"Window corners", "Icons", "Icon size", "Paint renderer"} {
		word := settingWords[i]
		lbl := findRowLabel(w.Content(), word)
		if lbl == nil {
			t.Fatalf("the word %q is not on the page", word)
		}
		cb := namedCombo(w.Content(), name)
		if widget.DeviceOrigin(lbl).X >= widget.DeviceOrigin(cb).X {
			t.Errorf("the word %q is not in front of the %q chooser", word, name)
		}
		if !strings.Contains(strings.ToLower(name), strings.ToLower(word)) {
			t.Errorf("the page says %q and a screen reader says %q: two names for one control", word, name)
		}
		if cb.Tip == "" {
			t.Errorf("the %q chooser says nothing about itself", name)
		}
	}
}

// rowLines is how many lines a wrapping row folded onto.
func rowLines(row *widgets.Wrap) int {
	tops := map[float32]bool{}
	for _, c := range row.Children() {
		tops[c.Bounds().Min.Y] = true
	}
	return len(tops)
}

// The preview carries one bar again, and it is the sample's own. There
// were two for a release: the sample's tools under its menu bar, where a
// tool bar belongs, and a bar of Settings' own above the menu bar, where
// no application has ever put one, carrying the icon set, the icon size
// and the corner style. Those three are settings of the page, so they
// are read where the page keeps its settings, and the window below is a
// sample again with nothing live in its chrome.
func TestThePreviewsOnlyBarIsTheSamples(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 860)
	var bars []*widgets.ToolBar
	widget.Walk(w.Content(), func(c widget.Component) {
		if b, ok := c.(*widgets.ToolBar); ok {
			bars = append(bars, b)
		}
	})
	if len(bars) != 1 {
		t.Fatalf("the page has %d tool bars, want the sample's own", len(bars))
	}
	tools := bars[0]
	if !insidePreview(tools) || tools.AccessibleName() != "Tools" {
		t.Errorf("the one tool bar is %q, inside the preview: %v", tools.AccessibleName(), insidePreview(tools))
	}
	// It is under the previewed window's menu bar, which is where a tool
	// bar belongs and where nothing of Settings' own stands over it.
	var menu *widgets.MenuBar
	widget.Walk(w.Content(), func(c widget.Component) {
		if m, ok := c.(*widgets.MenuBar); ok && menu == nil {
			menu = m
		}
	})
	if menu == nil {
		t.Fatal("the preview has no menu bar")
	}
	if widget.DeviceOrigin(tools).Y <= widget.DeviceOrigin(menu).Y {
		t.Error("the sample's tool bar is over its menu bar")
	}
	// No chooser of Settings' own rides on it, and none is anywhere in
	// that window. (The sample has a combo box of its own, "Choice", on
	// its Controls tab; that one is the point of a preview.)
	widget.Walk(previewScope(t, w), func(c widget.Component) {
		cb, ok := c.(*widgets.ComboBox)
		if !ok {
			return
		}
		switch cb.AccessibleName() {
		case "Icons", "Icon size", "Window corners", "Decade":
			t.Errorf("the %q chooser is back inside the preview", cb.AccessibleName())
		}
	})
	// And the window corners are nowhere in the sample's menus either:
	// they were three radio items in View for a release, where the owner
	// of this toolkit could not find them.
	widget.Walk(w.Content(), func(c widget.Component) {
		bar, ok := c.(*widgets.MenuBar)
		if !ok {
			return
		}
		for _, m := range bar.Menus() {
			for _, it := range m.Items {
				if strings.Contains(strings.ToLower(widget.PlainText(it.Text)), "corners") {
					t.Error("the window corners are back in the preview's View menu")
				}
			}
		}
	})
}

// The one thing in the previewed window that is not make-believe: File ▸
// Open… and Save… open the file dialog the staged "OS open/save dialogs"
// option asks for, so that the option can be looked at instead of read.
//
// What is checked here is which dialog was asked for, not which one
// appeared: there is no XDG portal on a test's private bus, so the
// desktop's dialog would fall back to the toolkit's and both branches
// would end in the same window.
func TestThePreviewOpensTheStagedFileDialog(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	asked := []bool{}
	var opts []widgets.FileDialogOptions
	defer swapPreviewDialog(t, func(from widget.Component, native bool, o widgets.FileDialogOptions) {
		asked = append(asked, native)
		opts = append(opts, o)
		if !insidePreview(from) {
			t.Error("the dialog opens from outside the preview, so it will not wear the staged pack")
		}
	})()
	a, w := openSettings(t, 1024, 860)

	// Unticked: the toolkit's own.
	openPreviewFile(t, w, "&Open…")
	a.PumpOnce()
	if len(asked) != 1 || asked[0] {
		t.Fatalf("with the box unticked the preview asked for native=%v", asked)
	}
	if opts[0].Mode != widgets.FileOpen || !strings.Contains(opts[0].Title, "preview") {
		t.Errorf("the dialog is %q, mode %v", opts[0].Title, opts[0].Mode)
	}
	// Ticked: the desktop's, from the staged setting and without an
	// Apply — which is the whole point, because Apply is what would make
	// it the applied one.
	findOption(w.Content(), "OS open/save dialogs").OnChange(true)
	a.PumpOnce()
	openPreviewFile(t, w, "&Save")
	a.PumpOnce()
	if len(asked) != 2 || !asked[1] {
		t.Fatalf("with the box ticked the preview asked for native=%v", asked)
	}
	if opts[1].Mode != widgets.FileSave {
		t.Errorf("Save opened a dialog in mode %v", opts[1].Mode)
	}
	if style.LoadAppearance().NativeDialogs {
		t.Error("previewing the dialog wrote look.json")
	}
	// The tool bar's Open and Save are the same command as the menu's.
	tools := previewTools(t, w)
	i := toolIndex(tools, style.IconOpen)
	if i < 0 || tools.Items()[i].OnClick == nil {
		t.Fatal("the sample's tool bar has no Open")
	}
	tools.Items()[i].OnClick()
	a.PumpOnce()
	if len(asked) != 3 {
		t.Error("the tool bar's Open does not open the dialog its menu item does")
	}
}

// And the staged setting wins over the applied one. style.NativeDialogs()
// is process-wide and holds what look.json said when Settings started,
// and widgets.ShowFileDialog ORs it — so a preview that went through the
// front door would show the desktop's dialog for an unticked box on a
// desktop whose dialogs are applied, which is a preview of the setting
// the user is trying to leave.
func TestThePreviewFileDialogIgnoresTheAppliedSetting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	saved := style.DefaultAppearance()
	saved.NativeDialogs = true
	if err := style.SaveAppearance(saved); err != nil {
		t.Fatal(err)
	}
	// A headless application built on an explicit look does not read
	// look.json into the process the way the command does, so say what
	// the command's start-up would have said: the desktop's dialogs are
	// the applied setting.
	style.SetNativeDialogs(true)
	t.Cleanup(func() { style.SetNativeDialogs(false) })
	a, w := openSettings(t, 1024, 860)
	if !style.NativeDialogs() {
		t.Fatal("the applied setting did not reach the process")
	}
	if box := findOption(w.Content(), "OS open/save dialogs"); box == nil || !box.Checked {
		t.Fatal("the option should open ticked")
	}
	// Untick it and ask the preview: the toolkit's own dialog, not the
	// portal's, and not a bare ShowFileDialog either.
	var native []bool
	done := swapPreviewDialog(t, func(_ widget.Component, n bool, _ widgets.FileDialogOptions) { native = append(native, n) })
	findOption(w.Content(), "OS open/save dialogs").OnChange(false)
	a.PumpOnce()
	openPreviewFile(t, w, "&Open…")
	a.PumpOnce()
	done()
	if len(native) != 1 || native[0] {
		t.Fatalf("the preview asked for native=%v while the applied setting says true", native)
	}

	// The real path, with nothing swapped: a themed dialog comes up in
	// the window even though style.NativeDialogs() is on, and it reads a
	// directory and writes nothing.
	openPreviewFile(t, w, "&Open…")
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("no dialog came up")
	}
	var fd *widgets.FileDialog
	widget.Walk(w.Overlay(), func(c widget.Component) {
		if d, ok := c.(*widgets.FileDialog); ok {
			fd = d
		}
	})
	if fd == nil {
		// The dialog is the overlay's card rather than a child of it in
		// some looks; the overlay being up is what matters, but a themed
		// dialog is what has to be up.
		if findOverlayButton(w, "Cancel") == nil {
			t.Fatal("the overlay is not the toolkit's own file dialog")
		}
	}
	if findOverlayButton(w, "Cancel") == nil {
		t.Error("the toolkit's file dialog has no Cancel")
	}
}

// swapPreviewDialog puts f in front of the preview's file dialog and
// gives back what restores it.
func swapPreviewDialog(t *testing.T, f func(widget.Component, bool, widgets.FileDialogOptions)) func() {
	t.Helper()
	was := showPreviewDialog
	showPreviewDialog = f
	restore := func() { showPreviewDialog = was }
	t.Cleanup(restore)
	return restore
}

// openPreviewFile picks an item of the previewed application's File menu.
func openPreviewFile(t *testing.T, w *app.Window, item string) {
	t.Helper()
	var bar *widgets.MenuBar
	widget.Walk(w.Content(), func(c widget.Component) {
		if m, ok := c.(*widgets.MenuBar); ok && bar == nil {
			bar = m
		}
	})
	if bar == nil {
		t.Fatal("the preview has no menu bar")
	}
	for _, m := range bar.Menus() {
		for _, it := range m.Items {
			if it.Text == item {
				if it.OnClick == nil {
					t.Fatalf("the preview's %q does nothing", item)
				}
				it.OnClick()
				return
			}
		}
	}
	t.Fatalf("no %q in the preview's menus", item)
}

func findOverlayButton(w *app.Window, text string) *widgets.Button {
	if w.Overlay() == nil {
		return nil
	}
	var btn *widgets.Button
	widget.Walk(w.Overlay(), func(c widget.Component) {
		if b, ok := c.(*widgets.Button); ok && b.Text == text {
			btn = b
		}
	})
	return btn
}

// A plain preview is make-believe all through: its File menu says what
// was asked of it and opens nothing. The Theme Atlas renders 129 tiles
// out of this window, and a tile is looked at rather than clicked.
func TestPlainPreviewOpensNothing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	opened := 0
	defer swapPreviewDialog(t, func(widget.Component, bool, widgets.FileDialogOptions) { opened++ })()
	a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: 1024, Height: 860, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsAppWith(a, w, SettingsOptions{Theme: "win95", PlainPreview: true}))
	a.PumpOnce()

	openPreviewFile(t, w, "&Open…")
	a.PumpOnce()
	if opened != 0 || w.Overlay() != nil {
		t.Errorf("a plain preview opened a file dialog (%d)", opened)
	}
	// It is the sample application otherwise: its own tool bar, whole.
	tools := previewTools(t, w)
	if i := toolIndex(tools, style.IconPaste); i < 0 {
		t.Error("the plain preview lost the sample's own tool bar")
	}
	// The four choosers are on the page, not in the window, so they are
	// there for a plain preview too — nothing the atlas crops holds them.
	for _, name := range []string{"Icons", "Icon size", "Window corners"} {
		cb := namedCombo(w.Content(), name)
		if cb == nil {
			t.Errorf("the %q chooser went with the plain preview", name)
		} else if insidePreview(cb) {
			t.Errorf("the %q chooser is inside the plain preview", name)
		}
	}
	// The panel it draws in is the same rectangle, so the atlas crops
	// both the same way.
	o, b := widget.DeviceOrigin(previewScope(t, w)), previewScope(t, w).LocalBounds()
	got := [4]int{int(o.X), int(o.Y), int(b.Dx()), int(b.Dy())}
	if want := [4]int{previewShotX, previewShotY, previewShotW, previewShotH}; got != want {
		t.Errorf("the plain preview panel is at %v, the atlas crops %v", got, want)
	}
	// And the page still works: the theme browser stages into it.
	clickTheme(t, w, "Aqua")
	a.PumpOnce()
	if got := previewAppearance(t, w).Name; got != "aqua" {
		t.Errorf("the plain preview did not follow the staged pack: %s", got)
	}
}
