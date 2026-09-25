package settingsapp

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/a11y/a11ytest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// Every control in Settings has a name, a box and a unique ID, on every
// page — including the live application inside the Themes page's preview
// — and the tree is enough to drive the app: a screen reader can see
// which page is selected and pick another.
func TestSettingsIsAccessible(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})

	s, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: 1024, Height: 860, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.SetContent(SettingsApp(a, s))
	a.PumpOnce()
	tree := a11ytest.Audit(t, "settings", s.AccessibleTree())
	// The Themes page: the pages sidebar and the theme browser, the
	// decade filter and the preview's own combo box.
	if a11ytest.Count(tree, a11y.RoleList) < 2 || a11ytest.Count(tree, a11y.RoleComboBox) < 2 {
		t.Errorf("settings: lists %d, combos %d",
			a11ytest.Count(tree, a11y.RoleList), a11ytest.Count(tree, a11y.RoleComboBox))
	}

	// The page list is a sidebar list whose selected item is the page.
	themes := a11ytest.Find(tree, a11y.RoleListItem, "Themes")
	if themes == nil || !themes.State.Has(a11y.StateSelected) {
		t.Fatalf("settings: the Themes page item %+v", themes)
	}
	// Actions reach the widgets: pick the second page through the tree.
	packs := a11ytest.Find(tree, a11y.RoleListItem, "Packs")
	if packs == nil || !s.AccessibleAction(packs.ID, a11y.ActionDefault) {
		t.Fatal("settings: selecting a page through the tree")
	}
	a.PumpOnce()
	if got := a11ytest.Find(s.AccessibleTree(), a11y.RoleListItem, "Packs"); got == nil || !got.State.Has(a11y.StateSelected) {
		t.Fatal("settings: the page did not change")
	}

	// Every page, the preview inside the Themes page included.
	for _, name := range settingsPages {
		clickSettingsNav(t, s, name)
		a.PumpOnce()
		a11ytest.Audit(t, "settings "+name, s.AccessibleTree())
	}

	// The options on Appearance are switches, and a screen reader has to
	// find them as switches.
	clickSettingsNav(t, s, "Appearance")
	a.PumpOnce()
	if n := a11ytest.Count(s.AccessibleTree(), a11y.RoleSwitch); n < 2 {
		t.Errorf("settings: Appearance has %d switches", n)
	}

	// The preview is a live application, not a picture: every control on
	// the tab it opens on is in the tree, named, with a box of its own.
	clickSettingsNav(t, s, "Themes")
	a.PumpOnce()
	tree = a11ytest.Audit(t, "settings themes", s.AccessibleTree())
	for _, r := range []a11y.Role{a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleRadioButton, a11y.RoleSlider, a11y.RoleProgressBar} {
		if a11ytest.Count(tree, r) == 0 {
			t.Errorf("settings: the preview application has no %s", r)
		}
	}
	// The icon strip at the head of Themes is tool bars of named buttons,
	// so the preview of an icon set is not a mystery to a screen reader,
	// and the two choosers beside it are combo boxes with names of their
	// own. (Information, Warning, Error and Question are the strip's
	// alone: New and Save are in the previewed application's tool bar
	// too.)
	for _, name := range []string{"Information", "Warning", "Error", "Question"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: the icon preview has no %s button", name)
		}
	}
	for _, name := range []string{"Icons", "Icon size"} {
		if a11ytest.Find(tree, a11y.RoleComboBox, name) == nil {
			t.Errorf("settings: the Themes page has no %s chooser", name)
		}
	}

	// Packs is still a page a screen reader can drive: the sets on disk
	// are there, and so is what deletes one.
	clickSettingsNav(t, s, "Packs")
	a.PumpOnce()
	packsTree := a11ytest.Audit(t, "settings packs", s.AccessibleTree())
	if a11ytest.Find(packsTree, a11y.RoleList, "Built-in") == nil {
		t.Error("settings: Packs has no built-in icon set list")
	}
}
