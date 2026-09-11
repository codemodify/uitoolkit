package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestPopupDismissAndMenu(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 480, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	picked := 0
	mb := widgets.NewMenuBar(widgets.NewMenu("&File",
		widgets.Item("About", func() { picked++ }),
	))
	w.SetContent(widgets.NewColumn(mb, widgets.NewLabel("body")))
	a.PumpOnce()
	mb.Open(0)
	a.PumpOnce()
	if w.Popup() == nil {
		t.Fatal("expected popup")
	}
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok {
		t.Fatalf("popup %T", w.Popup())
	}
	w.RequestFocus(pop)
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
	a.PumpOnce()
	if picked != 1 {
		t.Fatalf("picked %d", picked)
	}
	if w.Popup() != nil {
		t.Fatal("popup should close after pick")
	}

	mb.Open(0)
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(400, 200), Button: platform.ButtonLeft})
	if w.Popup() != nil {
		t.Fatal("outside click should dismiss")
	}
}

func TestTabOrderIncludesNewChrome(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 500, Height: 360, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	mb := widgets.NewMenuBar(widgets.NewMenu("&File", widgets.Item("Quit", nil)))
	tree := widgets.NewTreeView(widgets.NewTreeNode("root", widgets.NewTreeNode("leaf")))
	page := widgets.NewLabel("page")
	tv := widgets.NewTabView(widgets.Tab{Title: "Tree", Content: tree}, widgets.Tab{Title: "Page", Content: page})
	field := widgets.NewTextField("x", "", nil)
	w.SetContent(widgets.NewColumn(mb, tv, field))
	a.PumpOnce()
	list := widget.Focusables(w.Content())
	var kinds []string
	for _, c := range list {
		switch c.(type) {
		case *widgets.MenuBar:
			kinds = append(kinds, "menu")
		case *widgets.TabBar:
			kinds = append(kinds, "tabs")
		case *widgets.TreeView:
			kinds = append(kinds, "tree")
		case *widgets.TextField:
			kinds = append(kinds, "field")
		case *widgets.Label:
			kinds = append(kinds, "label")
		}
	}
	joined := ""
	for _, k := range kinds {
		joined += k + ","
	}
	if joined != "menu,tabs,tree,field," {
		t.Fatalf("tab order %s", joined)
	}
	tv.Select(1)
	list = widget.Focusables(w.Content())
	for _, c := range list {
		if c == tree {
			t.Fatal("hidden tree should leave tab order")
		}
	}
}

func TestMessageBoxResult(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 480, Height: 320, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("host"))
	a.PumpOnce()
	got := widgets.ResultNone
	mb := widgets.ShowMessageBox(w.Content(), widgets.MessageBoxOptions{
		Title: "Confirm", Message: "Proceed?", Kind: widgets.MessageQuestion,
		Buttons: widgets.ButtonsYesNo, OnResult: func(r widgets.MessageResult) { got = r },
	})
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("expected overlay")
	}
	var yes *widgets.Button
	widget.Walk(w.Overlay(), func(c widget.Component) {
		if b, ok := c.(*widgets.Button); ok && b.Text == "Yes" {
			yes = b
		}
	})
	if yes == nil {
		t.Fatal("yes")
	}
	yes.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 8)})
	yes.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(8, 8)})
	a.PumpOnce()
	if got != widgets.ResultYes {
		t.Fatalf("got %v", got)
	}
	if w.Overlay() != nil {
		t.Fatal("overlay should close")
	}
	_ = mb
}

func TestComboBoxOpensPopup(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	n := -1
	cb := widgets.NewComboBox([]string{"Alpha", "Beta", "Gamma"}, 0, func(i int) { n = i })
	w.SetContent(widgets.NewPad(20, cb))
	a.PumpOnce()
	cb.Open()
	a.PumpOnce()
	if w.Popup() == nil || !cb.Opened() {
		t.Fatal("expected combo popup")
	}
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok {
		t.Fatalf("popup %T", w.Popup())
	}
	w.RequestFocus(pop)
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	a.PumpOnce()
	if n != 1 || cb.Text() != "Beta" {
		t.Fatalf("n=%d text=%q", n, cb.Text())
	}
	if w.Popup() != nil {
		t.Fatal("popup should close")
	}
}

func TestEscapeDismissesMessageBox(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("host"))
	a.PumpOnce()
	got := widgets.ResultNone
	widgets.ShowMessageBox(w.Content(), widgets.MessageBoxOptions{
		Title: "X", Message: "Y", Buttons: widgets.ButtonsOKCancel,
		OnResult: func(r widgets.MessageResult) { got = r },
	})
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	a.PumpOnce()
	if got != widgets.ResultCancel {
		t.Fatalf("escape %v", got)
	}
	if w.Overlay() != nil {
		t.Fatal("overlay")
	}
}

func TestAltOpensMenu(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	mb := widgets.NewMenuBar(widgets.NewMenu("&File", widgets.Item("About", nil)))
	w.SetContent(mb)
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyF, Mods: platform.ModAlt})
	a.PumpOnce()
	if w.Popup() == nil {
		t.Fatal("Alt+F should open File")
	}
	if mb.OpenIndex() != 0 {
		t.Fatalf("open %d", mb.OpenIndex())
	}
}
