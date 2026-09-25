package settingsapp

import (
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The right-hand pane is the preview and the two things that stand over
// and under it: the row of five options, the application window in the
// staged pack, and where the files are. The window is the biggest thing
// in the pane at every size, switches when another pack is picked, and
// never re-themes Settings itself. The whole widget gallery used to sit
// under it in a second splitter (it is the tour's three showcase pages
// now) and a strip of stock icons sat above it for a while; this is what
// says neither has crept back.
func TestSettingsPreviewIsTheWholeRightPane(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a := uitoolkit.New(uitoolkit.Options{Look: style.LightLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "settings", Width: 1024, Height: 860, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(SettingsAppStaged(a, w, "win95"))
	a.PumpOnce()

	preview := previewScope(t, w)
	if got := style.LookAppearance(preview.Theme()).Name; got != "win95" {
		t.Fatalf("the preview draws the staged pack, got %s", got)
	}
	if a.Look().Name() == "win95" {
		t.Fatal("the preview must not re-theme Settings itself")
	}
	// One scope on the whole page: the preview. The strip of icons that
	// had one of its own is gone — the preview's tool bar is drawn in
	// the staged set and shows it better than a strip could.
	split := settingsSplit(t, w)
	if n := len(allScopes(w.Content())); n != 1 {
		t.Fatalf("the page has %d theme scopes, want only the preview", n)
	}
	if n := len(allScopes(browserColumn(t, w))); n != 0 {
		t.Fatalf("the browser column holds %d theme scopes, want none", n)
	}
	// The preview takes what the two rows beside it leave, and it is far
	// the biggest thing in the pane: everything else in here is one row.
	pane, box := split.PaneB(), preview.LocalBounds()
	if box.Dx() > pane.Dx()+1 {
		t.Errorf("the preview is %v wide in a %v pane", box.Dx(), pane.Dx())
	}
	if box.Dy() < pane.Dy()*0.6 {
		t.Errorf("the preview is %v of a %v pane", box.Dy(), pane.Dy())
	}
	// Over it, the five options — the four the column held and the one
	// that says where the colours come from; under it, the three paths.
	// Both are Settings' own, drawn in the applied look, outside the
	// preview's scope.
	follow := findOption(w.Content(), "OS colors")
	if follow == nil {
		t.Fatal("no desktop-colours option")
	}
	if insidePreview(follow) {
		t.Error("the colours option is inside the preview's theme scope")
	}
	files := findPanelTitled(w.Content(), "Files")
	if files == nil {
		t.Fatal("no Files block")
	}
	if inBrowserColumn(w.Content(), files) || inBrowserColumn(w.Content(), follow) {
		t.Error("the options row or Files is back in the browser column")
	}
	if widget.DeviceOrigin(follow).Y >= widget.DeviceOrigin(preview).Y {
		t.Error("the options row is not over the preview")
	}
	// And the column beside them is the theme browser and nothing else.
	for _, word := range []string{"Animations", "System file dialogs", "System frames", "Theme buttons"} {
		box := findOption(w.Content(), word)
		if box == nil {
			t.Fatalf("no %q option on the page", word)
		}
		if inBrowserColumn(w.Content(), box) {
			t.Errorf("the %q option is back in the browser column", word)
		}
	}
	if widget.DeviceOrigin(files).Y <= widget.DeviceOrigin(preview).Y {
		t.Error("the Files block is not under the preview")
	}
	// The showcase's own controls are the mark the gallery leaves.
	if findButton(w.Content(), "Primary action") != nil {
		t.Error("the widget gallery is back under the preview")
	}

	clickTheme(t, w, "Aqua")
	a.PumpOnce()
	if got := style.LookAppearance(preview.Theme()).Name; got != "aqua" {
		t.Fatalf("the preview is still on %s after picking Aqua", got)
	}
	if a.Look().Name() == "aqua" {
		t.Fatal("staging must not re-theme Settings")
	}
}

// Staging a theme does not rebuild the page: the search field keeps its
// caret and the focus, the theme list stays where the user scrolled it,
// and the preview is repainted where it stands rather than built again.
func TestSettingsStagingKeepsThePage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1100, 860)
	search := findSearchField(t, w)
	list := findThemeList(w.Content())
	preview := previewScope(t, w)
	if list == nil {
		t.Fatal("missing theme list")
	}
	w.RequestFocus(search)
	list.ScrollTo(list.MaxOffset() * 0.5)
	a.PumpOnce()
	listOff := list.OffsetY
	if listOff <= 0 {
		t.Fatalf("the theme list was not scrolled: %v", listOff)
	}

	clickTheme(t, w, "Aqua")
	a.PumpOnce()
	if findSearchField(t, w) != search {
		t.Fatal("staging rebuilt the search field, so the caret and the typing would be gone")
	}
	if w.Focus() != widget.Component(search) {
		t.Fatalf("the focus left the search field: %T", w.Focus())
	}
	if previewScope(t, w) != preview {
		t.Fatal("staging rebuilt the preview instead of repainting it")
	}
	if got := style.LookAppearance(preview.Theme()).Name; got != "aqua" {
		t.Fatalf("the preview was not repainted: %s", got)
	}
	if list.OffsetY != listOff {
		t.Fatalf("the theme list jumped: %v want %v", list.OffsetY, listOff)
	}

	// Apply does build the page again — every option has to show what it
	// now holds — and the list comes back down among the packs, on the
	// one that was staged, rather than at the top of 129 rows.
	clickApply(t, w)
	a.PumpOnce()
	got := findThemeList(w.Content())
	if got == nil || got.OffsetY <= 0 {
		t.Fatalf("the theme list went back to the top across Apply: %v", got)
	}
	if rowName(got.ItemText(got.Selected)) != "Aqua" {
		t.Fatalf("the applied pack is not the selected row: %q", got.ItemText(got.Selected))
	}
}

// The search field narrows the browser to the packs it names, and
// emptying it brings them all back.
func TestSettingsThemeSearch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)
	search := findSearchField(t, w)
	list := findThemeList(w.Content())
	all := list.Count
	if all < 50 {
		t.Fatalf("expected every built-in pack, got %d", all)
	}

	search.SetText("aqua")
	a.PumpOnce()
	if list.Count == 0 || list.Count >= all {
		t.Fatalf("searching aqua left %d of %d rows", list.Count, all)
	}
	for i := 0; i < list.Count; i++ {
		pack, ok := style.LoadTheme(packNamed(t, rowName(list.ItemText(i))))
		if !ok || !matchesQuery(pack, "aqua") {
			t.Fatalf("row %q does not match the search", list.ItemText(i))
		}
	}
	// A pack is found by what it is as well as by its name.
	search.SetText("kde")
	a.PumpOnce()
	if list.Count == 0 {
		t.Fatal("searching kde found nothing")
	}
	search.SetText("")
	a.PumpOnce()
	if list.Count != all {
		t.Fatalf("emptying the search left %d of %d rows", list.Count, all)
	}
	// Return stages the first pack the search found.
	search.SetText("hot dog")
	a.PumpOnce()
	search.OnSubmit(search.Text)
	a.PumpOnce()
	if got := previewAppearance(t, w).Name; !strings.Contains(got, "hotdog") {
		t.Fatalf("Return staged %q", got)
	}
}

// Tab walks the whole of Settings. There is one page, so one ring has to
// hold all of it: the search field, the decade filter and the theme list
// down the column, the Export under them, the five options and every
// chooser beside them, the live preview under those, and Apply. A check
// box in a wrapping row is a stop like any other — a row that folded its
// last two onto a second line must not have folded them out of the ring.
//
// The column is four stops now and Export is the last of them: Delete
// theme… was the fifth and is gone.
func TestSettingsKeyboardReachesEverything(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1100, 860)
	// Nothing is staged yet, so Apply is off and Tab steps over it, as
	// it should.
	for _, name := range []string{"Apply"} {
		for _, c := range focusRing(t, a, w) {
			if b, ok := c.(*widgets.Button); ok && b.Text == name {
				t.Errorf("Tab stops on %s while it is disabled", name)
			}
		}
	}
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	ring := focusRing(t, a, w)
	// The whole widget gallery used to be in this ring and made it more
	// than forty stops; what is left is Settings' own chrome and the
	// preview application's.
	if len(ring) < 15 {
		t.Fatalf("the focus ring is only %d stops long", len(ring))
	}
	want := map[string]bool{
		"search":        false,
		"themes":        false,
		"decade":        false,
		"icon set":      false,
		"icon size":     false,
		"export":        false,
		"animations":    false,
		"colours":       false,
		"file dialogs":  false,
		"system frame":  false,
		"theme buttons": false,
		"preview":       false,
		"apply":         false,
	}
	for _, c := range ring {
		switch v := c.(type) {
		case *widgets.TextField:
			if v.Placeholder == "Search themes" {
				want["search"] = true
			}
		case *widgets.ListView:
			if v.Count > 0 && strings.Contains(v.ItemText(0), "·") {
				want["themes"] = true
			}
		case *widgets.ComboBox:
			switch v.AccessibleName() {
			case "Icons":
				want["icon set"] = true
			case "Icon size":
				want["icon size"] = true
			case "Decade":
				want["decade"] = true
			}
		case *widgets.Checkbox:
			// The five over the preview; the previewed application's own
			// check boxes are in this ring too and are not these.
			switch v.Text {
			case "Animations":
				want["animations"] = true
			case "OS colors":
				want["colours"] = true
			case "System file dialogs":
				want["file dialogs"] = true
			case "System frames":
				want["system frame"] = true
			case "Theme buttons":
				want["theme buttons"] = true
			}
		case *widgets.Button:
			switch v.Text {
			case "Apply":
				want["apply"] = true
			case "Export current theme…":
				want["export"] = true
			case "Delete theme…":
				t.Error("Tab stops on Delete theme…, which is off the page")
			case "Dialog…":
				// A control of the previewed application: the preview is
				// live, not a picture, so Tab reaches into it.
				want["preview"] = true
			}
		}
	}
	for part, found := range want {
		if !found {
			t.Errorf("Tab never reaches the %s", part)
		}
	}
	// The choosers on the preview's settings bar are stops of their own,
	// and the bar itself is one: a tool bar that carries a control must
	// not swallow it.
	if set, bar := namedCombo(w.Content(), "Icons"), previewSettings(t, w); set != nil {
		seen := false
		for _, c := range ring {
			if c == widget.Component(set) {
				seen = true
			}
		}
		if !seen {
			t.Error("Tab never reaches the icon chooser on the preview's settings bar")
		}
		if set.Parent() != widget.Component(bar) {
			t.Errorf("the icon chooser is not on the settings bar but in %T", set.Parent())
		}
	}
	// Apply is at the foot of the window, outside the browser column and
	// outside the splitter, so it is a stop at every window size.
	if inBrowserColumn(w.Content(), findApply(w.Content())) {
		t.Error("Apply is inside the browser column")
	}
}

// ---- helpers --------------------------------------------------------------------

// focusRing is every stop Tab visits, in order, until it comes back to
// where it started.
func focusRing(t *testing.T, a *app.Application, w *app.Window) []widget.Component {
	t.Helper()
	tab := func() {
		w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
		a.PumpOnce()
	}
	tab()
	first := w.Focus()
	if first == nil {
		return nil
	}
	ring := []widget.Component{first}
	for i := 0; i < 4000; i++ {
		tab()
		c := w.Focus()
		if c == nil || c == first {
			return ring
		}
		ring = append(ring, c)
	}
	t.Fatal("the focus ring never came back round")
	return nil
}

func allScopes(root widget.Component) []*widgets.ThemeScope {
	var out []*widgets.ThemeScope
	widget.Walk(root, func(c widget.Component) {
		if sc, ok := c.(*widgets.ThemeScope); ok {
			out = append(out, sc)
		}
	})
	return out
}

// previewScope is the scope holding the preview application: the one
// with a menu bar in it.
func previewScope(t *testing.T, w *app.Window) *widgets.ThemeScope {
	t.Helper()
	return scopeWith(t, w, "preview", func(c widget.Component) bool {
		_, ok := c.(*widgets.MenuBar)
		return ok
	})
}

func scopeWith(t *testing.T, w *app.Window, what string, pred func(widget.Component) bool) *widgets.ThemeScope {
	t.Helper()
	for _, sc := range allScopes(w.Content()) {
		found := false
		widget.Walk(sc, func(c widget.Component) {
			if pred(c) {
				found = true
			}
		})
		if found {
			return sc
		}
	}
	t.Fatalf("no %s scope", what)
	return nil
}

func findSearchField(t *testing.T, w *app.Window) *widgets.TextField {
	t.Helper()
	var field *widgets.TextField
	widget.Walk(w.Content(), func(c widget.Component) {
		if tf, ok := c.(*widgets.TextField); ok && tf.Placeholder == "Search themes" {
			field = tf
		}
	})
	if field == nil {
		t.Fatal("no theme search field")
	}
	return field
}

// packNamed is the pack id behind a display name in the browser.
func packNamed(t *testing.T, name string) string {
	t.Helper()
	for _, p := range style.ListThemes() {
		if p.Display() == name {
			return p.Name
		}
	}
	t.Fatalf("no pack called %q", name)
	return ""
}
