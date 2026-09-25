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
// page — including the Themes page, where the whole showcase sits under
// the preview — and the tree is enough to drive the app: a screen reader
// can see which page is selected and pick another.
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
	if a11ytest.Count(tree, a11y.RoleList) < 2 || a11ytest.Count(tree, a11y.RoleComboBox) == 0 || a11ytest.Count(tree, a11y.RoleSwitch) < 2 {
		t.Errorf("settings: lists %d, combos %d, switches %d",
			a11ytest.Count(tree, a11y.RoleList), a11ytest.Count(tree, a11y.RoleComboBox), a11ytest.Count(tree, a11y.RoleSwitch))
	}

	// The page list is a sidebar list whose selected item is the page.
	themes := a11ytest.Find(tree, a11y.RoleListItem, "Themes")
	if themes == nil || !themes.State.Has(a11y.StateSelected) {
		t.Fatalf("settings: the Themes page item %+v", themes)
	}
	// Actions reach the widgets: pick the second page through the tree.
	packs := a11ytest.Find(tree, a11y.RoleListItem, "Packs & icons")
	if packs == nil || !s.AccessibleAction(packs.ID, a11y.ActionDefault) {
		t.Fatal("settings: selecting a page through the tree")
	}
	a.PumpOnce()
	if got := a11ytest.Find(s.AccessibleTree(), a11y.RoleListItem, "Packs & icons"); got == nil || !got.State.Has(a11y.StateSelected) {
		t.Fatal("settings: the page did not change")
	}

	// Every page, including the Themes page with the preview and the
	// whole showcase under it.
	for _, name := range settingsPages {
		clickSettingsNav(t, s, name)
		a.PumpOnce()
		a11ytest.Audit(t, "settings "+name, s.AccessibleTree())
	}
	clickSettingsNav(t, s, "Themes")
	a.PumpOnce()
	tree = a11ytest.Audit(t, "settings themes", s.AccessibleTree())
	for _, r := range []a11y.Role{a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleTable, a11y.RoleTree, a11y.RoleSlider} {
		if a11ytest.Count(tree, r) == 0 {
			t.Errorf("settings: the showcase under the preview has no %s", r)
		}
	}
}
