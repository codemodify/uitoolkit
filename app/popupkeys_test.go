package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// suggestList stands in for a completion drop-down: it takes the keys
// that are about the list and refuses everything else.
type suggestList struct {
	widget.Base
	took []platform.Key
}

func newSuggestList() *suggestList {
	s := &suggestList{}
	s.Init(s)
	return s
}

func (s *suggestList) KeyPress(e widget.KeyEvent) bool {
	switch e.Key {
	case platform.KeyUp, platform.KeyDown, platform.KeyReturn:
		s.took = append(s.took, e.Key)
		return true
	}
	return false
}

func (s *suggestList) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

// typeAheadHost is a window with a focused field and a popup over it.
func typeAheadHost(t *testing.T, pass bool) (*Application, *Window, *widgets.TextField, *suggestList) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 420, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	f := widgets.NewTextField("", "Search", nil)
	w.SetContent(widgets.NewColumn(f))
	a.PumpOnce()
	w.RequestFocus(f)
	a.PumpOnce()

	list := newSuggestList()
	list.Arrange(paintengine2d.XYWH(0, 40, 200, 80))
	widget.SetPopupKeysPass(list, pass)
	if !widget.ShowPopup(f, list) {
		t.Fatal("no popup")
	}
	a.PumpOnce()
	return a, w, f, list
}

// A non-capturing popup takes the keys that are about the list and lets
// the rest reach the field the user is typing in — which is the whole
// point of one, and was impossible while every popup owned the keyboard.
func TestNonCapturingPopupPassesKeysToTheField(t *testing.T) {
	a, w, f, list := typeAheadHost(t, true)

	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDown})
	a.PumpOnce()
	if len(list.took) != 1 || list.took[0] != platform.KeyDown {
		t.Fatalf("the list did not take Down: %v", list.took)
	}

	// Typing goes to the field, and the popup stays up.
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'a'})
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'd'})
	a.PumpOnce()
	if f.Text != "ad" {
		t.Fatalf("field %q", f.Text)
	}
	if w.Popup() == nil {
		t.Fatal("the popup went away while typing")
	}

	// A caret key the list does not want moves the caret in the field.
	f.SetSelection(1, 1)
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyLeft})
	a.PumpOnce()
	if f.Caret() != 0 {
		t.Fatalf("caret %d — Left did not reach the field", f.Caret())
	}
	// Backspace reaches it too, which is the key an empty field lets go.
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEnd})
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyBackspace})
	a.PumpOnce()
	if f.Text != "a" {
		t.Fatalf("after Backspace: %q", f.Text)
	}
}

// The default is unchanged: a popup owns the keyboard, because a menu is
// where the keyboard is while it is open.
func TestCapturingPopupStillOwnsTheKeyboard(t *testing.T) {
	a, w, f, _ := typeAheadHost(t, false)
	f.SetText("ab")
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyBackspace})
	a.PumpOnce()
	if f.Text != "ab" {
		t.Fatalf("a key leaked past the popup: %q", f.Text)
	}
	if widget.PopupKeysPass(w.Popup()) {
		t.Fatal("PopupKeysPass says a plain popup lets keys through")
	}
}
