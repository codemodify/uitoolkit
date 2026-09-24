package demo

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

// Settings draws the staged pack twice: the preview window on top and the
// whole widget gallery under it, both in that pack, both switching when
// another pack is picked.
func TestSettingsShowsGalleryUnderPreview(t *testing.T) {
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

	preview, gallery := previewScope(t, w), galleryScope(t, w)
	if style.LookAppearance(gallery.Theme()).Name != "win95" {
		t.Fatalf("the gallery draws the staged pack, got %+v", style.LookAppearance(gallery.Theme()))
	}
	if a.Look().Name() == "win95" {
		t.Fatal("the gallery must not re-theme Settings itself")
	}
	if py, gy := widget.DeviceOrigin(preview).Y, widget.DeviceOrigin(gallery).Y; gy <= py {
		t.Fatalf("the gallery sits under the preview: preview y=%v gallery y=%v", py, gy)
	}

	// Every one of the gallery's panes is there, and the whole of it is
	// reachable by scrolling.
	for _, title := range []string{"Buttons", "Fields", "ScrollView", "ListView", "TreeView", "TableView", "Form"} {
		if !hasPanel(gallery, title) {
			t.Errorf("the gallery pane is missing its %s panel", title)
		}
	}
	scroll := galleryScrollView(t, gallery)
	if scroll.MaxOffset() <= 0 {
		t.Fatalf("the gallery should scroll: content=%v view=%v", scroll.ContentHeight(), scroll.LocalBounds().Dy())
	}

	// Picking another pack repaints both scopes.
	clickTheme(t, w, "Aqua")
	a.PumpOnce()
	for _, sc := range allScopes(w.Content()) {
		if got := style.LookAppearance(sc.Theme()).Name; got != "aqua" {
			t.Fatalf("scope still on %s after picking Aqua", got)
		}
	}
	if a.Look().Name() == "aqua" {
		t.Fatal("staging must not re-theme Settings")
	}
}

// The window the examples draw and the pane Settings embeds are built
// from the same parts: the same controls, panels and views.
func TestGalleryWindowAndPaneShowTheSameControls(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	// The same host for both, so the comparison is about how the parts
	// are assembled and not about which controls the host greys out.
	host := GalleryHost{Root: func() widget.Component { return nil }}
	win := galleryInWindow(t, a, GalleryWindow(host))
	defer win.Close()
	pane := galleryInWindow(t, a, GalleryPane(host))
	defer pane.Close()

	got, want := census(a, pane), census(a, win)
	// The window owns chrome a pane inside Settings does not: its own
	// menu bar, title bar, tab bar and the heading over its left column.
	for _, key := range []string{"menubar", "titlebar", "tabview"} {
		delete(want.counts, key)
		delete(got.counts, key)
	}
	for _, text := range []string{"uitoolkit", "paintengine2d  ·  v" + uitoolkit.Version} {
		delete(want.texts["labels"], text)
	}
	for kind, wantSet := range want.texts {
		gotSet := got.texts[kind]
		for text := range wantSet {
			if !gotSet[text] {
				t.Errorf("%s: the pane is missing %q", kind, text)
			}
		}
		for text := range gotSet {
			if !wantSet[text] {
				t.Errorf("%s: only the pane has %q", kind, text)
			}
		}
	}
	for kind, n := range want.counts {
		if got.counts[kind] != n {
			t.Errorf("%s: the window has %d, the pane %d", kind, n, got.counts[kind])
		}
	}
	for kind, n := range got.counts {
		if _, ok := want.counts[kind]; !ok {
			t.Errorf("%s: only the pane has %d", kind, n)
		}
	}
}

// Staging a theme does not rebuild the page: the search field keeps its
// caret and the focus, and the theme list and the gallery stay where the
// user scrolled them.
func TestSettingsStagingKeepsThePage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1100, 860)
	search := findSearchField(t, w)
	list := findThemeList(w.Content())
	scroll := galleryScrollView(t, galleryScope(t, w))
	if list == nil {
		t.Fatal("missing theme list")
	}
	w.RequestFocus(search)
	list.ScrollTo(list.MaxOffset() * 0.5)
	scroll.ScrollTo(scroll.MaxOffset() * 0.5)
	a.PumpOnce()
	listOff, galleryOff := list.OffsetY, scroll.OffsetY
	if listOff <= 0 || galleryOff <= 0 {
		t.Fatalf("nothing was scrolled: list=%v gallery=%v", listOff, galleryOff)
	}

	clickTheme(t, w, "Aqua")
	a.PumpOnce()
	if findSearchField(t, w) != search {
		t.Fatal("staging rebuilt the search field, so the caret and the typing would be gone")
	}
	if w.Focus() != widget.Component(search) {
		t.Fatalf("the focus left the search field: %T", w.Focus())
	}
	if got := galleryScrollView(t, galleryScope(t, w)); got != scroll {
		t.Fatal("staging rebuilt the gallery")
	}
	if scroll.OffsetY != galleryOff {
		t.Fatalf("the gallery jumped: %v want %v", scroll.OffsetY, galleryOff)
	}
	if list.OffsetY != listOff {
		t.Fatalf("the theme list jumped: %v want %v", list.OffsetY, listOff)
	}

	// Apply does build the page again — every option has to show what it
	// now holds — and the gallery comes back where it was.
	clickApply(t, w)
	a.PumpOnce()
	if got := galleryScrollView(t, galleryScope(t, w)).OffsetY; got != galleryOff {
		t.Fatalf("the gallery lost its place across Apply: %v want %v", got, galleryOff)
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

// Defaults stages the appearance a fresh install has, and is off while
// that is already what is staged.
func TestSettingsDefaultsButton(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)
	def := findButton(w.Content(), "Defaults")
	if def == nil {
		t.Fatal("no Defaults button")
	}
	if def.Enabled() {
		t.Fatal("Defaults should be off when the defaults are already staged")
	}
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	if !findButton(w.Content(), "Defaults").Enabled() {
		t.Fatal("Defaults should come on once something else is staged")
	}
	findButton(w.Content(), "Defaults").OnClick()
	a.PumpOnce()
	if got := previewAppearance(t, w); got.Name != style.DefaultThemeName {
		t.Fatalf("Defaults staged %+v", got)
	}
	if style.LoadAppearance().Name != style.DefaultThemeName {
		t.Fatal("Defaults must not write look.json on its own")
	}
}

// Tab walks the whole of Settings: the pages, the search field, the theme
// list, the gallery under the preview, and the buttons that act on it.
func TestSettingsKeyboardReachesEverything(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1100, 860)
	// Nothing is staged yet, so Apply, Revert and Defaults are off and
	// Tab steps over them, as it should.
	for _, name := range []string{"Apply", "Revert", "Defaults"} {
		for _, c := range focusRing(t, a, w) {
			if b, ok := c.(*widgets.Button); ok && b.Text == name {
				t.Errorf("Tab stops on %s while it is disabled", name)
			}
		}
	}
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	ring := focusRing(t, a, w)
	if len(ring) < 40 {
		t.Fatalf("the focus ring is only %d stops long", len(ring))
	}
	want := map[string]bool{
		"search":  false,
		"themes":  false,
		"pages":   false,
		"gallery": false,
		"apply":   false,
		"revert":  false,
		"default": false,
	}
	for _, c := range ring {
		switch v := c.(type) {
		case *widgets.TextField:
			if v.Placeholder == "Search themes" {
				want["search"] = true
			}
		case *widgets.ListView:
			if v.Count > 0 && v.ItemText(0) == settingsPages[0] {
				want["pages"] = true
			} else if v.Count > 0 && strings.Contains(v.ItemText(0), "·") {
				want["themes"] = true
			}
		case *widgets.Button:
			switch v.Text {
			case "Apply":
				want["apply"] = true
			case "Revert":
				want["revert"] = true
			case "Defaults":
				want["default"] = true
			case "Primary action":
				want["gallery"] = true
			}
		}
	}
	for part, found := range want {
		if !found {
			t.Errorf("Tab never reaches the %s", part)
		}
	}

	// Every page is reachable and keeps a ring of its own.
	for _, name := range settingsPages {
		clickSettingsNav(t, w, name)
		a.PumpOnce()
		if len(focusRing(t, a, w)) == 0 {
			t.Errorf("%s: nothing takes the focus", name)
		}
		if findApply(w.Content()) == nil {
			t.Errorf("%s: Apply is not on the page", name)
		}
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

// previewScope is the scope holding the preview application (the one with
// a menu bar); galleryScope is the one holding the widget gallery.
func previewScope(t *testing.T, w *app.Window) *widgets.ThemeScope {
	t.Helper()
	return scopeWith(t, w, "preview", func(c widget.Component) bool {
		_, ok := c.(*widgets.MenuBar)
		return ok
	})
}

func galleryScope(t *testing.T, w *app.Window) *widgets.ThemeScope {
	t.Helper()
	return scopeWith(t, w, "gallery", func(c widget.Component) bool {
		b, ok := c.(*widgets.Button)
		return ok && b.Text == "Primary action"
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

func galleryScrollView(t *testing.T, scope *widgets.ThemeScope) *widgets.ScrollView {
	t.Helper()
	sv, ok := scope.Child().(*widgets.ScrollView)
	if !ok {
		t.Fatalf("the gallery pane is a %T, not a scroll view", scope.Child())
	}
	return sv
}

func hasPanel(root widget.Component, title string) bool {
	found := false
	widget.Walk(root, func(c widget.Component) {
		if p, ok := c.(*widgets.Panel); ok && p.Title == title {
			found = true
		}
	})
	return found
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

func galleryInWindow(t *testing.T, a *app.Application, content widget.Component) *app.Window {
	t.Helper()
	w, err := a.NewWindow(platform.WindowOptions{Title: "gallery", Width: 1000, Height: 760, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(content)
	a.PumpOnce()
	return w
}

// galleryCensus is what a build of the gallery holds: the text on every
// control, and how many of each kind there are.
type galleryCensus struct {
	texts  map[string]map[string]bool
	counts map[string]int
}

// census walks a window, and every page of every tab view in it — a tab
// view keeps only the page it shows — and reports what it found.
func census(a *app.Application, w *app.Window) galleryCensus {
	out := galleryCensus{texts: map[string]map[string]bool{}, counts: map[string]int{}}
	add := func(kind, text string) {
		if out.texts[kind] == nil {
			out.texts[kind] = map[string]bool{}
		}
		out.texts[kind][text] = true
	}
	// The chrome outside the tab view is counted once, and each tab's
	// page once, so a kind of control that two tabs both hold (fields in
	// the form and in the wizard) counts twice, as the pane shows it.
	var visit func(c widget.Component, seen map[string]int, intoTabs bool)
	visit = func(c widget.Component, seen map[string]int, intoTabs bool) {
		if c == nil || !c.Visible() {
			return
		}
		switch v := c.(type) {
		case *widgets.Button:
			add("buttons", v.Text)
		case *widgets.Panel:
			add("panels", v.Title)
		case *widgets.Switch:
			add("switches", v.Text)
		case *widgets.Checkbox:
			add("checks", v.Text)
		case *widgets.Label:
			add("labels", v.Text)
		case *widgets.MenuBar:
			seen["menubar"]++
		case *widgets.TitleBar:
			seen["titlebar"]++
		case *widgets.TabView:
			seen["tabview"]++
		case *widgets.ListView:
			seen["list"]++
		case *widgets.TreeView:
			seen["tree"]++
		case *widgets.TableView:
			seen["table"]++
		case *widgets.ComboBox:
			seen["combo"]++
		case *widgets.Slider:
			seen["slider"]++
		case *widgets.ProgressBar:
			seen["progress"]++
		case *widgets.TextField:
			seen["field"]++
		case *widgets.TextArea:
			seen["area"]++
		case *widgets.ToolBar:
			seen["toolbar"]++
		case *widgets.StatusBar:
			seen["status"]++
		case *widgets.Form:
			seen["form"]++
		case *widgets.Accordion:
			seen["accordion"]++
		case *widgets.CardList:
			seen["cards"]++
		case *widgets.Segmented:
			seen["segmented"]++
		}
		if _, ok := c.(*widgets.TabView); ok && !intoTabs {
			return
		}
		for _, ch := range c.Children() {
			visit(ch, seen, intoTabs)
		}
	}
	chrome := map[string]int{}
	visit(w.Content(), chrome, false)
	for k, n := range chrome {
		out.counts[k] += n
	}
	var tabs []*widgets.TabView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok {
			tabs = append(tabs, tv)
		}
	})
	for _, tv := range tabs {
		for i := range tv.Bar().Titles {
			tv.Select(i)
			a.PumpOnce()
			page := map[string]int{}
			visit(tv.Page(), page, true)
			for k, n := range page {
				out.counts[k] += n
			}
		}
		tv.Select(0)
		a.PumpOnce()
	}
	return out
}
