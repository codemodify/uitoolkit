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
	// the four choosers over the preview and the preview's own sample
	// combo box.
	if a11ytest.Count(tree, a11y.RoleList) < 1 || a11ytest.Count(tree, a11y.RoleComboBox) < 6 {
		t.Errorf("settings: lists %d, combos %d",
			a11ytest.Count(tree, a11y.RoleList), a11ytest.Count(tree, a11y.RoleComboBox))
	}
	// Every chooser Settings owns is named, so a screen reader says what
	// is being chosen rather than reading anonymous combo boxes — and a
	// chooser on a tool bar has nothing but its name to say it by.
	for _, name := range []string{"Decade", "Window corners", "Icons", "Icon size", "Paint renderer"} {
		if a11ytest.Find(tree, a11y.RoleComboBox, name) == nil {
			t.Errorf("settings: no %s chooser in the tree", name)
		}
	}
	// The words in front of them are in the tree as the static text they
	// are, and each is the beginning of the name of the chooser it
	// stands for: a screen reader reads out what the screen says. They
	// are promised now, which they were not while these three rode on a
	// tool bar inside the preview: a bar sheds its words when it runs out
	// of room, and at the 720x520 minimum it shed all three.
	for _, word := range settingWords {
		if a11ytest.Find(tree, a11y.RoleLabel, word) == nil {
			t.Errorf("settings: the word %q is not on the page", word)
		}
	}
	// The four options over the preview are check boxes, and each is in
	// the tree under the name that says in full what the short word on it
	// means: the word is what the eye gets, the name is what the ear
	// gets, and the one contains the other.
	for _, name := range []string{
		"Animations",
		"OS open/save dialogs: the desktop's own Open and Save",
		"OS window borders: the desktop's title bar and borders",
		"OS colors: follow the desktop's light or dark mode and its accent",
	} {
		if a11ytest.Find(tree, a11y.RoleCheckBox, name) == nil {
			t.Errorf("settings: no %q check box in the tree", name)
		}
	}
	// And the fifth is gone with its box: where the caption buttons go
	// follows "OS window borders" now.
	if a11ytest.Find(tree, a11y.RoleCheckBox, "Theme buttons: the caption buttons where the theme puts them") != nil {
		t.Error("settings: the Theme buttons check box is back in the tree")
	}
	// The theme browser is a named list and the search field beside it is
	// a named text field. Nothing on the screen says what the column is
	// any more — the Theme group box and the paragraph over the search
	// field are gone, and so is the Settings heading over both — so these
	// two names are the whole of what a screen reader has to go on, and
	// a11y.Check (which Audit runs) fails an unnamed list outright.
	if a11ytest.Find(tree, a11y.RoleList, "Themes") == nil {
		t.Error("settings: no theme browser in the tree")
	}
	if a11ytest.Find(tree, a11y.RoleTextField, "Search themes") == nil {
		t.Error("settings: the search field is not named in the tree")
	}
	// And nothing is left claiming to name them: a group called Theme
	// would now be a legend over the whole column.
	if a11ytest.Find(tree, a11y.RoleGroup, "Theme") != nil {
		t.Error("settings: the Theme group box is back in the tree")
	}

	// Export is on the page and named; Apply is at the foot.
	for _, name := range []string{"Export current theme…", "Apply"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: no %s button in the tree", name)
		}
	}
	// Neither Delete is one of them any more, and a screen reader must
	// not be told about a button the page does not have.
	for _, gone := range []string{"Delete theme…", "Delete icon set…"} {
		if a11ytest.Find(tree, a11y.RoleButton, gone) != nil {
			t.Errorf("settings: %s is back in the tree", gone)
		}
	}

	// The preview is a live application, not a picture: every control on
	// the tab it opens on is in the tree, named, with a box of its own.
	for _, r := range []a11y.Role{a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleRadioButton, a11y.RoleSlider, a11y.RoleProgressBar} {
		if a11ytest.Count(tree, r) == 0 {
			t.Errorf("settings: the preview application has no %s", r)
		}
	}
	// The preview's tool bar is named buttons, so the set it is drawn in
	// is not a mystery to a screen reader either.
	for _, name := range []string{"New", "Save", "Paste", "Send"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: the preview's tool bar has no %s button", name)
		}
	}
	// And it is the only bar in the tree: the Appearance bar that stood
	// over the sample's menu bar for a release is gone, and the three
	// choosers that were on it are on the page.
	if n := a11ytest.Count(tree, a11y.RoleToolBar); n != 1 {
		t.Errorf("settings: %d tool bars in the tree, want the sample's own", n)
	}
	if a11ytest.Find(tree, a11y.RoleToolBar, "Tools") == nil {
		t.Error("settings: no Tools bar in the tree")
	}
	if a11ytest.Find(tree, a11y.RoleToolBar, "Appearance") != nil {
		t.Error("settings: the Appearance bar is back inside the preview")
	}

	// Nothing on this page is behind a scroll offset: the column does not
	// scroll, so there is no foot of it to be lost from the tree, and the
	// one thing that does scroll is the theme list, whose rows a screen
	// reader reaches through the list itself.
	if sv := findScrollView(s.Content()); sv != nil {
		t.Error("settings: the page is back in a scroll view")
	}
	list := findThemeListAny(s.Content())
	if list == nil || list.MaxOffset() <= 0 {
		t.Fatal("settings: the theme list does not scroll its 129 packs")
	}
	list.ScrollTo(list.MaxOffset())
	a.PumpOnce()
	tree = a11ytest.Audit(t, "settings list scrolled", s.AccessibleTree())
	for _, name := range []string{"Export current theme…", "Apply"} {
		if a11ytest.Find(tree, a11y.RoleButton, name) == nil {
			t.Errorf("settings: scrolling the list lost %s from the tree", name)
		}
	}
	// The options and the choosers are beside the column, not in it:
	// scrolling the list cannot take them anywhere.
	if a11ytest.Find(tree, a11y.RoleCheckBox, "Animations") == nil {
		t.Error("settings: the options left the tree when the list scrolled")
	}
	if a11ytest.Find(tree, a11y.RoleComboBox, "Icons") == nil {
		t.Error("settings: the icon chooser left the tree when the list scrolled")
	}
}
