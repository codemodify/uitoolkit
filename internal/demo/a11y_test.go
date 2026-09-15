package demo

import (
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// auditWindow checks a window's accessibility tree and reports what a
// screen reader would stumble over.
func auditWindow(t *testing.T, name string, w *app.Window) *a11y.Node {
	t.Helper()
	tree := w.AccessibleTree()
	if len(tree.Children) == 0 {
		t.Fatalf("%s: empty accessibility tree", name)
	}
	var lines []string
	for _, p := range a11y.Check(tree) {
		lines = append(lines, p.String())
	}
	if len(lines) > 0 {
		t.Errorf("%s: %d accessibility problems:\n%s", name, len(lines), strings.Join(lines, "\n"))
	}
	return tree
}

func count(tree *a11y.Node, role a11y.Role) int {
	n := 0
	tree.Walk(func(x *a11y.Node) bool {
		if x.Role == role {
			n++
		}
		return true
	})
	return n
}

// Every control in the gallery and in Settings has a name, a box and a
// unique ID, and the tree carries the controls a screen reader lists.
func TestAppsAreAccessible(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})

	g, err := a.NewWindow(platform.WindowOptions{Title: "Gallery", Width: 1280, Height: 860, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	g.SetContent(Gallery(a, g, false))
	a.PumpOnce()
	tree := auditWindow(t, "gallery", g)
	// Every page of every tab view.
	var tabs []*widgets.TabView
	widget.Walk(g.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok {
			tabs = append(tabs, tv)
		}
	})
	for _, tv := range tabs {
		for i := range tv.Bar().Titles {
			tv.Select(i)
			a.PumpOnce()
			auditWindow(t, "gallery tab "+tv.Bar().Titles[i], g)
		}
		tv.Select(0)
	}
	for _, r := range []a11y.Role{a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleMenuBar, a11y.RoleTabList, a11y.RoleTab} {
		if count(tree, r) == 0 {
			t.Errorf("gallery: no %s", r)
		}
	}

	s, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: 1024, Height: 860, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.SetContent(SettingsApp(a, s))
	a.PumpOnce()
	tree = auditWindow(t, "settings", s)
	if count(tree, a11y.RoleList) < 2 || count(tree, a11y.RoleComboBox) == 0 || count(tree, a11y.RoleSwitch) < 2 {
		t.Errorf("settings: lists %d, combos %d, switches %d", count(tree, a11y.RoleList), count(tree, a11y.RoleComboBox), count(tree, a11y.RoleSwitch))
	}
	// The page list is a sidebar list whose selected item is the page.
	var themes *a11y.Node
	tree.Walk(func(n *a11y.Node) bool {
		if n.Role == a11y.RoleListItem && n.Name == "Themes" {
			themes = n
		}
		return themes == nil
	})
	if themes == nil || !themes.State.Has(a11y.StateSelected) {
		t.Fatalf("settings: the Themes page item %+v", themes)
	}
	// Actions reach the widgets: pick the second page through the tree.
	var packs *a11y.Node
	tree.Walk(func(n *a11y.Node) bool {
		if n.Role == a11y.RoleListItem && n.Name == "Packs & icons" {
			packs = n
		}
		return packs == nil
	})
	if packs == nil || !s.AccessibleAction(packs.ID, a11y.ActionDefault) {
		t.Fatal("settings: selecting a page through the tree")
	}
	a.PumpOnce()
	found := false
	s.AccessibleTree().Walk(func(n *a11y.Node) bool {
		if n.Role == a11y.RoleListItem && n.Name == "Packs & icons" && n.State.Has(a11y.StateSelected) {
			found = true
		}
		return true
	})
	if !found {
		t.Fatal("settings: the page did not change")
	}
}

// The other sample apps pass the audit too.
func TestSampleAppsAreAccessible(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	for name, build := range map[string]func(*app.Window) widget.Component{
		"files": FilesApp, "notes": NotesApp, "inspector": InspectorApp,
	} {
		w, err := a.NewWindow(platform.WindowOptions{Title: name, Width: 1040, Height: 700, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(build(w))
		a.PumpOnce()
		auditWindow(t, name, w)
		w.Close()
	}
}
