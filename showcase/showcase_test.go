package showcase_test

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/a11y/a11ytest"
	"github.com/codemodify/uitoolkit/app"
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

// The window the showcase draws and the pane it offers an embedder are
// built from the same parts: the same controls, panels and views. The
// tour shows the parts over three pages and Settings once put the pane
// under its theme preview; whoever embeds Pane next should get the whole
// showcase, not most of it.
func TestShowcaseWindowAndPaneShowTheSameControls(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	// The same host for both, so the comparison is about how the parts
	// are assembled and not about which controls the host greys out.
	host := showcase.Host{Root: func() widget.Component { return nil }}
	win := showcaseInWindow(t, a, showcase.Window(host))
	defer win.Close()
	pane := showcaseInWindow(t, a, showcase.Pane(host))
	defer pane.Close()

	got, want := census(a, pane), census(a, win)
	// The window owns chrome an embedded pane does not: its own menu bar,
	// title bar, tab bar and the heading over its left column.
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

func showcaseInWindow(t *testing.T, a *app.Application, content widget.Component) *app.Window {
	t.Helper()
	w, err := a.NewWindow(platform.WindowOptions{Title: "showcase", Width: 1000, Height: 760, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(content)
	a.PumpOnce()
	return w
}

// showcaseCensus is what a build of the showcase holds: the text on every
// control, and how many of each kind there are.
type showcaseCensus struct {
	texts  map[string]map[string]bool
	counts map[string]int
}

// census walks a window, and every page of every tab view in it — a tab
// view keeps only the page it shows — and reports what it found.
func census(a *app.Application, w *app.Window) showcaseCensus {
	out := showcaseCensus{texts: map[string]map[string]bool{}, counts: map[string]int{}}
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
