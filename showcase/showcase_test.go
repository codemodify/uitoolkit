package showcase_test

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/a11y/a11ytest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/showcase"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Every control in the showcase has a name, a box and a unique ID — on
// every page of every tab view, not only the one that opens — and the
// tree carries the controls a screen reader lists.
func TestShowcaseIsAccessible(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})

	g, err := a.NewWindow(platform.WindowOptions{Title: "Gallery", Width: 1280, Height: 860, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	g.SetContent(showcase.App(a, g, false))
	a.PumpOnce()
	tree := a11ytest.Audit(t, "showcase", g.AccessibleTree())

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
			a11ytest.Audit(t, "showcase tab "+tv.Bar().Titles[i], g.AccessibleTree())
		}
		tv.Select(0)
	}
	for _, r := range []a11y.Role{a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleMenuBar, a11y.RoleTabList, a11y.RoleTab} {
		if a11ytest.Count(tree, r) == 0 {
			t.Errorf("showcase: no %s", r)
		}
	}
}

// The pane is the same showcase without a window's chrome: a host that
// leaves Window, SwitchTheme and Quit out greys those controls rather
// than panicking, which is how Settings embeds it.
func TestPaneNeedsNoApplication(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.LightLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Pane", Width: 900, Height: 900, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var root widget.Component
	pane := showcase.Pane(showcase.Host{Light: true, Root: func() widget.Component { return root }})
	root = pane
	w.SetContent(pane)
	a.PumpOnce()
	a11ytest.Audit(t, "showcase pane", w.AccessibleTree())
}
