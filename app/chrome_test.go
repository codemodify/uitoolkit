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

// A modal message box stays until answered: clicks on the dimmer do not
// dismiss it (a double-click on "Confirm…" used to open and close it), the
// default button has focus, and the look orders the buttons.
func TestMessageBoxModalFocusAndOrder(t *testing.T) {
	for _, tc := range []struct {
		theme        string
		primaryFirst bool
	}{{"dark", false}, {"win95", true}} {
		pack, ok := style.LoadTheme(tc.theme)
		if !ok {
			t.Fatalf("theme %s", tc.theme)
		}
		a := New(Options{Look: pack.Look(), Headless: true})
		w, err := a.NewWindow(platform.WindowOptions{Width: 520, Height: 360, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(widgets.NewLabel("host"))
		a.PumpOnce()
		got := widgets.ResultNone
		widgets.ShowMessageBox(w.Content(), widgets.MessageBoxOptions{
			Title: "Confirm", Message: "Proceed?", Kind: widgets.MessageQuestion,
			Buttons: widgets.ButtonsYesNo, OnResult: func(r widgets.MessageResult) { got = r },
		})
		a.PumpOnce()
		ov := w.Overlay()
		if ov == nil {
			t.Fatal("overlay")
		}
		var buttons []*widgets.Button
		widget.Walk(ov, func(c widget.Component) {
			if b, ok := c.(*widgets.Button); ok {
				buttons = append(buttons, b)
			}
		})
		if len(buttons) != 2 {
			t.Fatalf("%s: %d buttons", tc.theme, len(buttons))
		}
		if first := buttons[0].Text; (first == "Yes") != tc.primaryFirst {
			t.Fatalf("%s: first button %q, primaryFirst=%v", tc.theme, first, tc.primaryFirst)
		}
		if f, ok := w.Focus().(*widgets.Button); !ok || f.Text != "Yes" {
			t.Fatalf("%s: default button must have focus, got %T %v", tc.theme, w.Focus(), w.Focus())
		}
		// Click on the dimmer, well outside the card.
		w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(4, 4), Button: platform.ButtonLeft})
		w.dispatch(platform.Event{Kind: platform.EventMouseUp, Pos: paintengine2d.Pt(4, 4), Button: platform.ButtonLeft})
		a.PumpOnce()
		if w.Overlay() == nil || got != widgets.ResultNone {
			t.Fatalf("%s: a click outside dismissed the modal box (result %v)", tc.theme, got)
		}
		w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
		a.PumpOnce()
		if got != widgets.ResultNo || w.Overlay() != nil {
			t.Fatalf("%s: Escape should answer No and close, got %v", tc.theme, got)
		}
	}
}

// menuFixture is a window with a text field and a File / Edit menu bar.
func menuFixture(t *testing.T) (*Application, *Window, *widgets.TextField, *widgets.MenuBar, map[string]int) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 520, Height: 320, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	fired := map[string]int{}
	bar := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&New", "Ctrl+N", func() { fired["new"]++ }),
			widgets.ItemAccel("&Help", "F1", func() { fired["help"]++ }),
		),
		widgets.NewMenu("&Edit",
			widgets.ItemAccel("&Copy", "Ctrl+C", func() { fired["copy"]++ }),
		),
	)
	field := widgets.NewTextField("hello", "", nil)
	w.SetContent(widgets.NewColumn(bar, field))
	a.PumpOnce()
	return a, w, field, bar, fired
}

// Shortcuts shown in menus work window-wide — except for keys the focused
// widget takes (Ctrl+C in a field copies text) and under a modal overlay.
func TestMenuAcceleratorsDispatch(t *testing.T) {
	a, w, field, _, fired := menuFixture(t)
	key := func(k platform.Key, m platform.Modifiers) {
		w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: k, Mods: m})
		a.PumpOnce()
	}
	key(platform.KeyN, platform.ModCtrl)
	key(platform.KeyF1, 0)
	if fired["new"] != 1 || fired["help"] != 1 {
		t.Fatalf("accelerators with no focus: %v", fired)
	}
	field.RequestFocus()
	key(platform.KeyN, platform.ModCtrl)
	if fired["new"] != 2 {
		t.Fatalf("Ctrl+N with a field focused: %v", fired)
	}
	key(platform.KeyC, platform.ModCtrl)
	if fired["copy"] != 0 {
		t.Fatal("Ctrl+C belongs to the focused text field")
	}
	key(platform.KeyN, platform.ModCtrl|platform.ModShift)
	if fired["new"] != 2 {
		t.Fatal("Ctrl+Shift+N must not match Ctrl+N")
	}
	widgets.Info(w.Content(), "Modal", "A modal box", nil)
	a.PumpOnce()
	key(platform.KeyN, platform.ModCtrl)
	if fired["new"] != 2 {
		t.Fatal("accelerators must not fire under a modal overlay")
	}
	key(platform.KeyF, platform.ModAlt)
	if w.Popup() != nil {
		t.Fatal("Alt+F must not open the main menu above a modal dialog")
	}
}

// Picking from a menu gives focus back to the widget that had it (it stayed
// on the bar and typing went nowhere).
func TestMenuPickReturnsFocus(t *testing.T) {
	a, w, field, bar, fired := menuFixture(t)
	field.RequestFocus()
	a.PumpOnce()
	edit := bar.TitleRect(1)
	o := widget.DeviceOrigin(bar)
	pos := paintengine2d.Pt(o.X+edit.Min.X+4, o.Y+edit.Min.Y+4)
	w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: pos, Button: platform.ButtonLeft})
	w.dispatch(platform.Event{Kind: platform.EventMouseUp, Pos: pos, Button: platform.ButtonLeft})
	a.PumpOnce()
	if w.Popup() == nil {
		t.Fatal("Edit did not open")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
	a.PumpOnce()
	if fired["copy"] != 1 {
		t.Fatalf("Copy not activated: %v", fired)
	}
	if w.Focus() != field {
		t.Fatalf("focus after the menu closed: %T, want the text field", w.Focus())
	}
}

// Left / Right walk the bar while a dropdown is open, and a keyboard-opened
// menu highlights its first item.
func TestMenuKeyboardWalksOpenMenus(t *testing.T) {
	a, w, _, bar, _ := menuFixture(t)
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyF, Mods: platform.ModAlt})
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || bar.OpenIndex() != 0 {
		t.Fatalf("Alt+F did not open File (open=%d)", bar.OpenIndex())
	}
	if pop.HighlightedIndex() != 0 {
		t.Fatalf("keyboard-opened menu should highlight the first item, got %d", pop.HighlightedIndex())
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyRight})
	a.PumpOnce()
	if bar.OpenIndex() != 1 {
		t.Fatalf("Right should open Edit, open=%d", bar.OpenIndex())
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyLeft})
	a.PumpOnce()
	if bar.OpenIndex() != 0 {
		t.Fatalf("Left should go back to File, open=%d", bar.OpenIndex())
	}
}
