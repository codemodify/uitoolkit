package settingsapp

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/a11y/a11ytest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// Every control in Settings has a name, a box and a unique ID — the live
// application inside the preview included — and the tree is enough to
// drive the app. There is one page now, so there is nothing to navigate
// to first: everything the old four pages held has to be in this one
// tree.
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

	// The theme browser and the preview's own lists; the decade filter,
	// the corners, the two icon choosers and the preview's combo box.
	if a11ytest.Count(tree, a11y.RoleList) < 1 || a11ytest.Count(tree, a11y.RoleComboBox) < 5 {
		t.Errorf("settings: lists %d, combos %d",
			a11ytest.Count(tree, a11y.RoleList), a11ytest.Count(tree, a11y.RoleComboBox))
	}
	// Every chooser Settings owns is named, so a screen reader says what
	// is being chosen rather than reading three anonymous combo boxes.
	for _, name := range []string{"Decade", "Corners", "Icons", "Icon size"} {
		if a11ytest.Find(tree, a11y.RoleComboBox, name) == nil {
			t.Errorf("settings: no %s chooser in the tree", name)
		}
	}
	// The options are switches, and a screen reader has to find them as
	// switches: four behaviours and the one that says where the colours
	// come from.
	if n := a11ytest.Count(tree, a11y.RoleSwitch); n < 5 {
		t.Errorf("settings: the page has %d switches, want the five options", n)
	}
	// The theme browser is a named list, and so is the search beside it.
	if a11ytest.Find(tree, a11y.RoleList, "Themes") == nil {
		t.Error("settings: no theme browser in the tree")
	}

	// Export and Delete are on the page and named. Delete is there while
	// it is grey, because a built-in pack is staged: a screen reader that
	// could not find it at all would have no way to learn that removing a
	// pack is something Settings does.
	for _, name := range []string{"Export current theme…", "Delete theme…", "Delete icon set…", "Apply"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: no %s button in the tree", name)
		}
	}

	// The preview is a live application, not a picture: every control on
	// the tab it opens on is in the tree, named, with a box of its own.
	for _, r := range []a11y.Role{a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleRadioButton, a11y.RoleSlider, a11y.RoleProgressBar} {
		if a11ytest.Count(tree, r) == 0 {
			t.Errorf("settings: the preview application has no %s", r)
		}
	}
	// The icon strip beside it is tool bars of named buttons, so the
	// preview of an icon set is not a mystery to a screen reader.
	// (Information, Warning, Error and Question are the strip's alone:
	// New and Save are in the previewed application's tool bar too.)
	for _, name := range []string{"Information", "Warning", "Error", "Question"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: the icon preview has no %s button", name)
		}
	}

	// Scrolling the column of choices does not take anything out of the
	// tree: the sections below the fold are still there to be driven.
	scroll := findScrollView(s.Content())
	if scroll == nil {
		t.Fatal("settings: the column of choices does not scroll")
	}
	scroll.ScrollTo(scroll.MaxOffset())
	a.PumpOnce()
	tree = a11ytest.Audit(t, "settings scrolled", s.AccessibleTree())
	if a11ytest.Find(tree, a11y.RoleComboBox, "Icons") == nil {
		t.Error("settings: scrolling the column lost the icon chooser from the tree")
	}
}
