package filesapp

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func findTable(c widget.Component) *widgets.TableView {
	if t, ok := c.(*widgets.TableView); ok {
		return t
	}
	for _, k := range c.Children() {
		if t := findTable(k); t != nil {
			return t
		}
	}
	return nil
}

// Files goes back and forward through the folders a tab showed: Alt+Left
// and Alt+Right, the mouse's back and forward buttons, and a sideways
// three-finger swipe; each tab keeps its own history.
func TestFilesHistory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Files", Width: 1040, Height: 700, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	root := FilesApp(w).(*shortcutRoot)
	w.SetContent(root)
	a.PumpOnce()
	tabs, _ := w.TitleBar().(*widgets.HeaderBar)
	if tabs == nil {
		t.Fatal("no header bar")
	}
	if root.NavigateHistory(false) {
		t.Fatal("went back with no history")
	}
	// The second tab shows Projects; open the uitoolkit folder in it.
	strip := findStrip(w.TitleBar())
	strip.Select(1)
	// Spelled out once: what the desktop is told, prefix and all. The
	// rest of the test goes through WindowTitle, so a retitle is checked
	// for keeping whatever the prefix is.
	if w.Title() != "uitoolkit - Projects — Files" {
		t.Fatalf("title %q", w.Title())
	}
	table := findTable(root)
	table.OnActivate(0)
	a.PumpOnce()
	if w.Title() != WindowTitle("uitoolkit — Files") || strip.Tab(1).Title != "uitoolkit" {
		t.Fatalf("after opening a folder: %q, tab %q", w.Title(), strip.Tab(1).Title)
	}
	key := func(k platform.Key) {
		w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: k, Mods: platform.ModAlt})
		w.Inject(platform.Event{Kind: platform.EventKeyUp, Key: k, Mods: platform.ModAlt})
		a.PumpOnce()
	}
	key(platform.KeyLeft)
	if w.Title() != WindowTitle("Projects — Files") {
		t.Fatalf("Alt+Left: %q", w.Title())
	}
	key(platform.KeyRight)
	if w.Title() != WindowTitle("uitoolkit — Files") {
		t.Fatalf("Alt+Right: %q", w.Title())
	}
	// The mouse's back button, over the listing.
	at := paintengine2d.Pt(600, 300)
	w.Inject(platform.Event{Kind: platform.EventMouseDown, Button: platform.ButtonBack, Pos: at})
	w.Inject(platform.Event{Kind: platform.EventMouseUp, Button: platform.ButtonBack, Pos: at})
	a.PumpOnce()
	if w.Title() != WindowTitle("Projects — Files") {
		t.Fatalf("back button: %q", w.Title())
	}
	// Three fingers swiped left: forward.
	w.Inject(platform.Event{Kind: platform.EventGesture, Gesture: platform.GestureSwipe, Phase: platform.GestureBegin, Fingers: 3, Pos: at, Scale: 1})
	w.Inject(platform.Event{Kind: platform.EventGesture, Gesture: platform.GestureSwipe, Phase: platform.GestureUpdate, Fingers: 3, Pos: at, Scale: 1, Delta: paintengine2d.Pt(-150, 6)})
	w.Inject(platform.Event{Kind: platform.EventGesture, Gesture: platform.GestureSwipe, Phase: platform.GestureEnd, Fingers: 3, Pos: at, Scale: 1})
	a.PumpOnce()
	if w.Title() != WindowTitle("uitoolkit — Files") {
		t.Fatalf("swipe: %q", w.Title())
	}
	// The first tab has a history of its own: none.
	strip.Select(0)
	if root.NavigateHistory(false) {
		t.Fatal("the other tab's history leaked into this one")
	}
	strip.Select(1)
	if !root.NavigateHistory(false) || w.Title() != WindowTitle("Projects — Files") {
		t.Fatalf("the tab lost its history: %q", w.Title())
	}
}

func findStrip(c widget.Component) *widgets.BrowserTabs {
	if c == nil {
		return nil
	}
	if s, ok := c.(*widgets.BrowserTabs); ok {
		return s
	}
	for _, k := range c.Children() {
		if s := findStrip(k); s != nil {
			return s
		}
	}
	return nil
}

// Back and Forward on the tool bar are greyed while there is nowhere to go.
func TestFilesHistoryButtons(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Files", Width: 1040, Height: 700, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	root := FilesApp(w).(*shortcutRoot)
	w.SetContent(root)
	a.PumpOnce()
	var back, fwd *widgets.ToolItem
	var find func(widget.Component)
	find = func(c widget.Component) {
		if tb, ok := c.(*widgets.ToolBar); ok {
			for _, it := range tb.Items() {
				switch it.Text {
				case "‹ Back":
					back = it
				case "Forward ›":
					fwd = it
				}
			}
		}
		for _, k := range c.Children() {
			find(k)
		}
	}
	find(root)
	if back == nil || fwd == nil {
		t.Fatal("no Back / Forward")
	}
	if !back.Disabled || !fwd.Disabled {
		t.Fatalf("a fresh window: back disabled %v, forward disabled %v", back.Disabled, fwd.Disabled)
	}
	strip := findStrip(w.TitleBar())
	strip.Select(1)
	findTable(root).OnActivate(0)
	if back.Disabled || !fwd.Disabled {
		t.Fatal("after opening a folder Back should be live, Forward not")
	}
	root.NavigateHistory(false)
	if !back.Disabled || fwd.Disabled {
		t.Fatal("after going back Forward should be live, Back not")
	}
}
