package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// keySpy records every key it is offered and takes the ones it is told
// to, so a test can read both *what* heard a key and in what order.
type keySpy struct {
	widget.Base
	name  string
	takes map[platform.Key]bool
	heard *[]string
}

func newKeySpy(name string, heard *[]string, takes ...platform.Key) *keySpy {
	s := &keySpy{name: name, heard: heard, takes: map[platform.Key]bool{}}
	for _, k := range takes {
		s.takes[k] = true
	}
	s.Init(s)
	s.SetWantsFocus(true)
	return s
}

func (s *keySpy) KeyPress(e widget.KeyEvent) bool {
	*s.heard = append(*s.heard, s.name)
	return s.takes[e.Key]
}

func (s *keySpy) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	for _, c := range s.Children() {
		c.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
	}
}

func keyRig(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	return a, w
}

// **A popup of widgets is not a menu: a key reaches what has the keyboard
// inside it.** A drop-down holding a list is focused onto that list by
// FocusOn and Tab moves the focus round its widgets — and until this the
// keys went to the popup and stopped there, so the list never heard Up,
// Down or Enter.
func TestAPopupsKeysReachWhatHasTheKeyboardInIt(t *testing.T) {
	a, w := keyRig(t)
	var heard []string
	anchor := widgets.NewLabel("content")
	w.SetContent(anchor)
	a.PumpOnce()

	pop := newKeySpy("popup", &heard, platform.KeyEscape)
	inner := newKeySpy("list", &heard, platform.KeyDown)
	pop.Add(inner)
	pop.Arrange(paintengine2d.XYWH(0, 40, 200, 80))
	if !widget.ShowPopup(anchor, pop) {
		t.Fatal("no popup")
	}
	w.RequestFocus(inner)
	a.PumpOnce()

	// A key the focused widget takes stops there.
	heard = nil
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDown})
	if len(heard) != 1 || heard[0] != "list" {
		t.Errorf("Down was heard by %v, want the list alone", heard)
	}
	// One it refuses climbs to the popup, as a key climbs to the window.
	heard = nil
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyF7})
	if len(heard) != 2 || heard[0] != "list" || heard[1] != "popup" {
		t.Errorf("F7 was heard by %v, want the list then the popup", heard)
	}
}

// A menu's keyboard is the menu itself, and that routing is untouched:
// with the focus on the popup, or on nothing inside it, the popup hears
// the key as it always did.
func TestAMenuPopupStillHearsItsOwnKeys(t *testing.T) {
	a, w := keyRig(t)
	var heard []string
	anchor := widgets.NewLabel("content")
	w.SetContent(anchor)
	a.PumpOnce()

	pop := newKeySpy("popup", &heard, platform.KeyDown)
	inner := newKeySpy("row", &heard)
	pop.Add(inner)
	pop.Arrange(paintengine2d.XYWH(0, 40, 200, 80))
	if !widget.ShowPopup(anchor, pop) {
		t.Fatal("no popup")
	}
	w.RequestFocus(pop)
	a.PumpOnce()

	heard = nil
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDown})
	if len(heard) != 1 || heard[0] != "popup" {
		t.Errorf("Down was heard by %v, want the popup alone", heard)
	}
}

// **A key with the keyboard on nothing reaches the content root**, which
// is where an application keeps the window's own keys. A window nobody
// has clicked in yet still answers Ctrl+Page Up and Escape.
func TestAKeyWithNoFocusReachesTheContentRoot(t *testing.T) {
	a, w := keyRig(t)
	var heard []string
	root := newKeySpy("root", &heard)
	// Nothing in the window takes focus, so nothing has the keyboard —
	// a window the user has not clicked in.
	root.SetWantsFocus(false)
	w.SetContent(root)
	a.PumpOnce()
	if w.Focus() != nil {
		t.Fatalf("the fixture starts with the keyboard on %T", w.Focus())
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDelete})
	if len(heard) != 1 || heard[0] != "root" {
		t.Errorf("Delete was heard by %v, want the content root", heard)
	}
}

// **And when what had the keyboard is taken out of the window, the
// window's keys keep working.** A page swapped for another detaches a
// subtree without telling the window, so the field went on pointing at
// it and every key climbed a chain that no longer reached the window:
// Escape, Delete and Ctrl+Page Up died until something was clicked.
func TestAKeyAfterTheFocusLeftTheWindowReachesTheRoot(t *testing.T) {
	a, w := keyRig(t)
	var heard []string
	root := newKeySpy("root", &heard)
	page := newKeySpy("page", &heard)
	field := newKeySpy("field", &heard, platform.KeyDelete)
	page.Add(field)
	root.Add(page)
	w.SetContent(root)
	a.PumpOnce()
	w.RequestFocus(field)
	a.PumpOnce()
	if w.Focus() != widget.Component(field) {
		t.Fatalf("the keyboard is on %T", w.Focus())
	}

	// The page is swapped for another: the old subtree comes off the tree.
	root.ClearChildren()
	root.Add(newKeySpy("page2", &heard))
	a.PumpOnce()

	heard = nil
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDelete})
	if len(heard) != 1 || heard[0] != "root" {
		t.Errorf("Delete was heard by %v, want the content root — and not the detached field", heard)
	}
	if w.Focus() != nil {
		t.Errorf("the keyboard is still on %T, want nothing", w.Focus())
	}
}

// Nothing leaks behind a modal dialog: with one up and the keyboard on
// nothing, the key goes to the dialog, which is the window's content
// while it is up, and never to what is behind it.
func TestAKeyWithNoFocusUnderAModalGoesToTheDialog(t *testing.T) {
	a, w := keyRig(t)
	var heard []string
	root := newKeySpy("root", &heard)
	w.SetContent(root)
	dialog := newKeySpy("dialog", &heard)
	dialog.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	w.PushOverlay(dialog)
	a.PumpOnce()
	w.RequestFocus(nil)

	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDelete})
	if len(heard) != 1 || heard[0] != "dialog" {
		t.Errorf("Delete was heard by %v, want the dialog alone", heard)
	}
}
