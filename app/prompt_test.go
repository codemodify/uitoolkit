package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// promptHost is a window with a label in it, ready to be prompted on.
func promptHost(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 420, Height: 260, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("host"))
	a.PumpOnce()
	return a, w
}

func typeRunes(w *Window, s string) {
	for _, r := range s {
		w.dispatch(platform.Event{Kind: platform.EventText, Rune: r})
	}
}

// A prompt puts the caret in its field, not on its default button: the
// dialog exists to be typed in, and a user who has to press Tab first
// would type the first letter into a button.
func TestPromptFocusesItsFieldAndSelectsTheInitialValue(t *testing.T) {
	a, w := promptHost(t)
	mb := widgets.Prompt(w.Content(), "Rename", "Folder name", "Drafts", nil)
	a.PumpOnce()
	f := mb.Field()
	if f == nil {
		t.Fatal("no field")
	}
	if w.Focus() != widget.Component(f) {
		t.Fatalf("focus is %T, want the field", w.Focus())
	}
	if a, b := f.Selection(); a != 0 || b != len("Drafts") {
		t.Fatalf("selection %d..%d, want the whole initial value", a, b)
	}
	// Typing replaces it, which is what the selection is for.
	typeRunes(w, "Sent")
	a.PumpOnce()
	if f.Text != "Sent" {
		t.Fatalf("after typing over the selection: %q", f.Text)
	}
}

// Return is the default button when the field has the focus, and the
// text comes back with it.
func TestPromptReturnAccepts(t *testing.T) {
	a, w := promptHost(t)
	var got string
	var ok bool
	called := 0
	widgets.Prompt(w.Content(), "New folder", "Name", "", func(s string, k bool) {
		got, ok, called = s, k, called+1
	})
	a.PumpOnce()
	typeRunes(w, "Receipts")
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
	a.PumpOnce()
	if called != 1 || !ok || got != "Receipts" {
		t.Fatalf("on(%q, %v) called %d times", got, ok, called)
	}
	if w.Overlay() != nil {
		t.Fatal("the overlay is still up after Return")
	}
}

// Escape cancels. The field must let it bubble — it is not Clearable, so
// nothing inside the field wants it — and the overlay's cancel is the
// same one the Cancel button presses.
func TestPromptEscapeCancels(t *testing.T) {
	a, w := promptHost(t)
	var got string
	ok := true
	called := 0
	widgets.Prompt(w.Content(), "Rename", "Name", "Drafts", func(s string, k bool) {
		got, ok, called = s, k, called+1
	})
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	a.PumpOnce()
	if called != 1 || ok {
		t.Fatalf("on(%q, %v) called %d times", got, ok, called)
	}
	if w.Overlay() != nil {
		t.Fatal("the overlay is still up after Escape")
	}
}

// Required: an empty name is not an answer, so OK is grey and Return
// does nothing until there is something to return.
func TestPromptRequiredRefusesAnEmptyAnswer(t *testing.T) {
	a, w := promptHost(t)
	got := widgets.ResultNone
	mb := widgets.ShowMessageBox(w.Content(), widgets.MessageBoxOptions{
		Title: "New folder", Message: "Name", Kind: widgets.MessageQuestion,
		Buttons:  widgets.ButtonsOKCancel,
		Input:    &widgets.MessageBoxInput{Required: true},
		OnResult: func(r widgets.MessageResult) { got = r },
	})
	a.PumpOnce()
	var okBtn *widgets.Button
	widget.Walk(w.Overlay(), func(c widget.Component) {
		if b, is := c.(*widgets.Button); is && b.Text == "OK" {
			okBtn = b
		}
	})
	if okBtn == nil {
		t.Fatal("no OK button")
	}
	if okBtn.Enabled() {
		t.Fatal("OK is live with an empty required field")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
	a.PumpOnce()
	if got != widgets.ResultNone || w.Overlay() == nil {
		t.Fatalf("Return accepted an empty required field: %v", got)
	}
	// Whitespace is not a name either.
	typeRunes(w, "  ")
	a.PumpOnce()
	if okBtn.Enabled() {
		t.Fatal("OK is live for a field holding only spaces")
	}
	typeRunes(w, "Receipts")
	a.PumpOnce()
	if !okBtn.Enabled() {
		t.Fatal("OK is still grey with a name typed")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
	a.PumpOnce()
	if got != widgets.ResultOK || mb.Text() != "  Receipts" {
		t.Fatalf("got %v %q", got, mb.Text())
	}
}

// A message box without an Input is unchanged: no field, and Text is
// empty rather than something a caller might act on.
func TestMessageBoxWithoutAnInputHasNoField(t *testing.T) {
	a, w := promptHost(t)
	mb := widgets.ShowMessageBox(w.Content(), widgets.MessageBoxOptions{
		Title: "X", Message: "Y", Buttons: widgets.ButtonsOK,
	})
	a.PumpOnce()
	if mb.Field() != nil || mb.Text() != "" {
		t.Fatalf("field %v text %q", mb.Field(), mb.Text())
	}
	var n int
	widget.Walk(w.Overlay(), func(c widget.Component) {
		if _, is := c.(*widgets.TextField); is {
			n++
		}
	})
	if n != 0 {
		t.Fatalf("%d text fields in a plain message box", n)
	}
}
