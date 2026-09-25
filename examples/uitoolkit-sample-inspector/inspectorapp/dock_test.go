package inspectorapp

import (
	"os"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/dock"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// buildInspector puts the inspector in a headless window and hands back
// its dock host.
func buildInspector(t *testing.T) (*app.Application, *app.Window, *dock.Host) {
	t.Helper()
	a := uitoolkit.New(uitoolkit.Options{
		Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true,
	})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Inspector", Width: 1000, Height: 700, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(InspectorApp(w))
	a.PumpOnce()
	host, ok := widget.Find[*dock.Host](w.Content())
	if !ok {
		t.Fatal("the inspector has no dock host")
	}
	return a, w, host
}

// panelSide is the area a named panel sits in.
func panelSide(t *testing.T, h *dock.Host, name string) dock.Side {
	t.Helper()
	p := h.Panel(name)
	if p == nil {
		t.Fatalf("the inspector has no panel called %q", name)
	}
	for _, s := range []dock.Side{dock.SideLeft, dock.SideRight, dock.SideTop, dock.SideBottom} {
		found := false
		widget.Walk(h.Area(s), func(c widget.Component) {
			if c == widget.Component(p) {
				found = true
			}
		})
		if found {
			return s
		}
	}
	t.Fatalf("panel %q is in no area", name)
	return dock.SideLeft
}

func TestInspectorDocksItsPanels(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w, host := buildInspector(t)
	defer func() { w.Close(); _ = a }()
	for name, want := range map[string]dock.Side{
		"outline": dock.SideLeft, "properties": dock.SideRight, "log": dock.SideBottom,
	} {
		if got := panelSide(t, host, name); got != want {
			t.Errorf("%s starts in the %v area, want %v", name, got, want)
		}
	}
	if host.Centre() == nil {
		t.Error("the inspector has no central widget")
	}
}

func TestInspectorRemembersItsLayout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	a, w, host := buildInspector(t)
	// Move a panel, as a user dragging its title bar would.
	host.Dock(host.Panel("log"), dock.SideTop)
	a.PumpOnce()
	if got := panelSide(t, host, "log"); got != dock.SideTop {
		t.Fatalf("the log did not move: it is in the %v area", got)
	}
	path := dock.LayoutFile(LayoutName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no layout was written to %s: %v", path, err)
	}
	w.Close()

	// Start again against the same config dir, as a restart would.
	a2, w2, host2 := buildInspector(t)
	defer func() { w2.Close(); _ = a2 }()
	if got := panelSide(t, host2, "log"); got != dock.SideTop {
		t.Errorf("after a restart the log is in the %v area, want top", got)
	}
	if got := panelSide(t, host2, "outline"); got != dock.SideLeft {
		t.Errorf("after a restart the outline is in the %v area, want left", got)
	}
}

func TestInspectorResetsItsLayout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w, host := buildInspector(t)
	defer func() { w.Close(); _ = a }()
	host.Dock(host.Panel("log"), dock.SideTop)
	a.PumpOnce()
	if !host.ResetLayout() {
		t.Fatal("the inspector recorded no default layout")
	}
	a.PumpOnce()
	if got := panelSide(t, host, "log"); got != dock.SideBottom {
		t.Errorf("after a reset the log is in the %v area, want bottom", got)
	}
}

func TestInspectorPanelsClose(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w, host := buildInspector(t)
	defer func() { w.Close(); _ = a }()
	log := host.Panel("log")
	if !log.Close() {
		t.Fatal("the log panel refused to close")
	}
	a.PumpOnce()
	if !log.Closed() {
		t.Error("the log panel did not close")
	}
	log.Show()
	a.PumpOnce()
	if log.Closed() {
		t.Error("the log panel did not come back")
	}
}

// A saved layout that the running app cannot make sense of must not stop
// it coming up.
func TestInspectorSurvivesABrokenLayout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(dir+"/uitoolkit", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dock.LayoutFile(LayoutName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, w, host := buildInspector(t)
	defer func() { w.Close(); _ = a }()
	for _, name := range []string{"outline", "properties", "log"} {
		if host.Panel(name) == nil {
			t.Fatalf("panel %q went missing", name)
		}
		panelSide(t, host, name) // fails the test when it is in no area
	}
}

// Every window the inspector puts on the desktop carries the toolkit's
// prefix — the main window from main.go, and a panel's window from the
// dock's opener, whose name would otherwise be the panel's own title.
func TestInspectorFloatsUnderTheToolkitName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w, host := buildInspector(t)
	defer w.Close()
	log := host.Panel("log")
	if log == nil {
		t.Fatal("the inspector has no log panel")
	}
	if !log.Float() {
		t.Fatal("the log panel would not float")
	}
	a.PumpOnce()
	var titles []string
	for _, win := range a.Windows() {
		if win != w {
			titles = append(titles, win.Title())
		}
	}
	if len(titles) != 1 || titles[0] != "uitoolkit - Log" {
		t.Fatalf("the floating panel's windows are %q, want one called %q", titles, "uitoolkit - Log")
	}
	host.CloseFloating()
}
