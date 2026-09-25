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
	// the three choosers on the preview's settings bar and the preview's
	// own sample combo box.
	if a11ytest.Count(tree, a11y.RoleList) < 1 || a11ytest.Count(tree, a11y.RoleComboBox) < 5 {
		t.Errorf("settings: lists %d, combos %d",
			a11ytest.Count(tree, a11y.RoleList), a11ytest.Count(tree, a11y.RoleComboBox))
	}
	// Every chooser Settings owns is named, so a screen reader says what
	// is being chosen rather than reading anonymous combo boxes — and a
	// chooser on a tool bar has nothing but its name to say it by.
	for _, name := range []string{"Decade", "Icons", "Icon size", "Window corners"} {
		if a11ytest.Find(tree, a11y.RoleComboBox, name) == nil {
			t.Errorf("settings: no %s chooser in the tree", name)
		}
	}
	// The words in front of them are in the tree as the static text they
	// are, and each is the beginning of the name of the chooser it
	// stands for: a screen reader reads out what the screen says.
	for _, word := range PreviewSettingsWords {
		if a11ytest.Find(tree, a11y.RoleLabel, word) == nil {
			t.Errorf("settings: the word %q is not on the preview's settings bar", word)
		}
	}
	// The five options over the preview are check boxes, and each is in
	// the tree under the name that says in full what the short word on it
	// means: the word is what the eye gets, the name is what the ear
	// gets, and the one contains the other.
	for _, name := range []string{
		"Animations",
		"Use the desktop's file dialogs",
		"System frame: the desktop's title bar and borders",
		"Theme buttons: the caption buttons where the theme puts them",
		"Desktop colours: follow the desktop's light or dark mode and its accent",
	} {
		if a11ytest.Find(tree, a11y.RoleCheckBox, name) == nil {
			t.Errorf("settings: no %q check box in the tree", name)
		}
	}
	// The theme browser is a named list, and so is the search beside it.
	if a11ytest.Find(tree, a11y.RoleList, "Themes") == nil {
		t.Error("settings: no theme browser in the tree")
	}

	// Export and Delete are on the page and named. Delete is there while
	// it is grey, because a built-in pack is staged: a screen reader that
	// could not find it at all would have no way to learn that removing a
	// pack is something Settings does.
	for _, name := range []string{"Export current theme…", "Delete theme…", "Apply"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: no %s button in the tree", name)
		}
	}
	// Delete icon set… is not one of them any more, and a screen reader
	// must not be told about a button the page does not have.
	if a11ytest.Find(tree, a11y.RoleButton, "Delete icon set…") != nil {
		t.Error("settings: Delete icon set… is back in the tree")
	}

	// The preview is a live application, not a picture: every control on
	// the tab it opens on is in the tree, named, with a box of its own.
	for _, r := range []a11y.Role{a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleRadioButton, a11y.RoleSlider, a11y.RoleProgressBar} {
		if a11ytest.Count(tree, r) == 0 {
			t.Errorf("settings: the preview application has no %s", r)
		}
	}
	// The preview's tool bar is named buttons, so the set it is drawn in
	// is not a mystery to a screen reader either, and both bars are in
	// the tree: the sample's tools and the settings over them.
	for _, name := range []string{"New", "Save", "Paste", "Send"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: the preview's tool bar has no %s button", name)
		}
	}
	if a11ytest.Count(tree, a11y.RoleToolBar) < 2 {
		t.Error("settings: the preview's two bars are not both in the tree")
	}
	for _, name := range []string{"Tools", "Appearance"} {
		if a11ytest.Find(tree, a11y.RoleToolBar, name) == nil {
			t.Errorf("settings: no %s bar in the tree", name)
		}
	}

	// Scrolling the column of choices does not take anything out of the
	// tree: what is under the fold in it is still there to be driven.
	scroll := findScrollView(s.Content())
	if scroll == nil {
		t.Fatal("settings: the column of choices does not scroll")
	}
	scroll.ScrollTo(scroll.MaxOffset())
	a.PumpOnce()
	tree = a11ytest.Audit(t, "settings scrolled", s.AccessibleTree())
	if a11ytest.Find(tree, a11y.RoleButton, "Delete theme…") == nil {
		t.Error("settings: scrolling the column lost the foot of it from the tree")
	}
	// The options and the choosers are beside the column, not in it:
	// scrolling it cannot take them anywhere.
	if a11ytest.Find(tree, a11y.RoleCheckBox, "Animations") == nil {
		t.Error("settings: the options left the tree when the column scrolled")
	}
	if a11ytest.Find(tree, a11y.RoleComboBox, "Icons") == nil {
		t.Error("settings: the icon chooser left the tree when the column scrolled")
	}
}
