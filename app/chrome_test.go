package app

import (
	"testing"
	"time"

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

func TestTooltipDelayAndEscape(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("Save", nil)
	btn.Tip = "Write the file"
	w.SetContent(widgets.NewPad(20, btn))
	a.PumpOnce()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	w.SetClock(func() time.Time { return now })
	w.SetTooltipDelay(400 * time.Millisecond)
	bb := btn.Bounds()
	pos := paintengine2d.Pt(20+bb.Min.X+8, 20+bb.Min.Y+8)
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: pos})
	a.PumpOnce()
	if w.Tooltip() != nil {
		t.Fatal("tooltip too early")
	}
	now = now.Add(500 * time.Millisecond)
	a.PumpOnce()
	if w.Tooltip() == nil {
		t.Fatal("expected tooltip")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	if w.Tooltip() != nil {
		t.Fatal("esc should hide tooltip")
	}
}

func TestEscapeClosesPopupThenOverlay(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 480, Height: 320, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("host"))
	a.PumpOnce()
	cancelled := 0
	widgets.ShowFileDialog(w.Content(), widgets.FileDialogOptions{
		Title: "Open", Path: "/stub", Entries: []widgets.FileInfo{{Name: "a.go"}},
		OnCancel: func() { cancelled++ },
	})
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("overlay")
	}
	cb := widgets.NewComboBox([]string{"A", "B"}, 0, nil)
	// popup on top of overlay should still dismiss first
	widgets.ShowContextMenu(w.Content(), paintengine2d.Pt(40, 40), widgets.Item("X", nil))
	a.PumpOnce()
	if w.Popup() == nil {
		t.Fatal("popup")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	if w.Popup() != nil {
		t.Fatal("esc popup")
	}
	if w.Overlay() == nil {
		t.Fatal("overlay should remain")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	a.PumpOnce()
	if w.Overlay() != nil {
		t.Fatal("esc overlay")
	}
	if cancelled != 1 {
		t.Fatalf("cancel %d", cancelled)
	}
	_ = cb
}

func TestNumberFieldKeysBubbleFromInnerField(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 360, Height: 160, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	nf := widgets.NewNumberField(0, 20, 5, 1, nil)
	w.SetContent(widgets.NewPad(16, nf))
	a.PumpOnce()
	w.RequestFocus(nf.Field())
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyUp})
	if nf.Value != 6 {
		t.Fatalf("bubble up %v", nf.Value)
	}
}

func TestExpanderRequestsLayout(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	child := widgets.NewLabel("secret")
	exp := widgets.NewExpander("More", false, child)
	w.SetContent(widgets.NewPad(8, exp))
	a.PumpOnce()
	if child.Bounds().Dy() > 1 {
		t.Fatalf("collapsed body should be empty, got %v", child.Bounds().Dy())
	}
	exp.SetExpanded(true)
	a.PumpOnce()
	if child.Bounds().Dy() <= 1 {
		t.Fatalf("expand should relayout body, got %v", child.Bounds().Dy())
	}
	if !child.Visible() {
		t.Fatal("expanded child hidden")
	}
}

func TestAccordionExclusiveYieldsWindowFocus(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 360, Height: 280, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	inner := widgets.NewSwitch("Hidden", true, nil)
	secA := widgets.NewExpander("A", true, inner)
	secB := widgets.NewExpander("B", false, widgets.NewLabel("bb"))
	w.SetContent(widgets.NewAccordion(true, secA, secB))
	a.PumpOnce()
	w.RequestFocus(inner)
	if w.Focus() != inner {
		t.Fatal("inner focus")
	}
	secB.SetExpanded(true)
	a.PumpOnce()
	if w.Focus() == inner {
		t.Fatal("focus stuck in collapsed accordion body")
	}
	if secA.Expanded {
		t.Fatal("exclusive should close A")
	}
}

func TestTabOrderIncludesTextAreaSwitchAccordion(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 320, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	ta := widgets.NewTextArea("x", "", nil)
	sw := widgets.NewSwitch("On", false, nil)
	inner := widgets.NewSwitch("Nested", true, nil)
	acc := widgets.NewAccordion(true, widgets.NewExpander("More", true, inner))
	w.SetContent(widgets.NewColumn(ta, sw, acc))
	a.PumpOnce()
	var kinds []string
	for _, c := range widget.Focusables(w.Content()) {
		switch c.(type) {
		case *widgets.TextArea:
			kinds = append(kinds, "area")
		case *widgets.Switch:
			kinds = append(kinds, "switch")
		}
	}
	joined := ""
	for _, k := range kinds {
		joined += k + ","
	}
	if joined != "area,switch,switch," {
		t.Fatalf("tab order %s", joined)
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

// Leaving the window must clear hover and cancel the pending tooltip. There
// used to be no leave event at all: buttons stayed hot and tooltips popped
// up after the pointer was gone (Wayland and X11).
func TestPointerLeaveClearsHoverAndTooltip(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("Save", nil)
	btn.Tip = "Write the file"
	w.SetContent(widgets.NewPad(20, btn))
	a.PumpOnce()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	w.SetClock(func() time.Time { return now })
	w.SetTooltipDelay(400 * time.Millisecond)
	bb := btn.Bounds()
	pos := paintengine2d.Pt(20+bb.Min.X+8, 20+bb.Min.Y+8)
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: pos})
	if !btn.State().Hovered() {
		t.Fatal("button should be hovered")
	}
	w.dispatch(platform.Event{Kind: platform.EventPointerLeave})
	if btn.State().Hovered() {
		t.Fatal("hover must clear when the pointer leaves the window")
	}
	now = now.Add(time.Second)
	a.PumpOnce()
	if w.Tooltip() != nil {
		t.Fatal("no tooltip may appear after the pointer left")
	}
}
