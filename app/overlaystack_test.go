package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func stackWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 500, Height: 400, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("behind"))
	a.PumpOnce()
	return a, w
}

// A window had one overlay, so a confirmation raised from inside a
// dialog replaced that dialog and ran its OnClose — which for a dialog
// that wipes its secret fields on close wiped them.
func TestOverlaysStack(t *testing.T) {
	a, w := stackWindow(t)

	firstClosed := 0
	first := widgets.NewMessageBox(widgets.MessageBoxOptions{
		Title: "Edit entry", Message: "the dialog underneath",
		Buttons:  widgets.ButtonsOKCancel,
		OnResult: func(widgets.MessageResult) { firstClosed++ },
	})
	first.Show(w.Content())
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("the first dialog did not show")
	}
	firstOverlay := w.Overlay()

	secondClosed := 0
	second := widgets.NewMessageBox(widgets.MessageBoxOptions{
		Title: "Are you sure?", Message: "raised from inside the first",
		Buttons:  widgets.ButtonsYesNo,
		OnResult: func(widgets.MessageResult) { secondClosed++ },
	})
	second.Show(first.Overlay())
	a.PumpOnce()

	if firstClosed != 0 {
		t.Error("showing a second dialog closed the first")
	}
	if got := len(w.Overlays()); got != 2 {
		t.Fatalf("%d overlays, want 2", got)
	}
	if w.Overlay() == firstOverlay {
		t.Error("the second dialog is not on top")
	}

	// Answering the confirmation returns to the dialog underneath.
	widget.DismissOverlay(second.Overlay())
	a.PumpOnce()
	if secondClosed != 1 {
		t.Errorf("the confirmation reported closing %d times", secondClosed)
	}
	if firstClosed != 0 {
		t.Error("closing the confirmation closed the dialog under it")
	}
	if got := len(w.Overlays()); got != 1 {
		t.Fatalf("%d overlays after the pop, want 1", got)
	}
	if w.Overlay() != firstOverlay {
		t.Error("the dialog underneath is not back on top")
	}

	// And the first still closes normally.
	widget.DismissOverlay(firstOverlay)
	a.PumpOnce()
	if firstClosed != 1 {
		t.Errorf("the dialog reported closing %d times", firstClosed)
	}
	if len(w.Overlays()) != 0 || w.Overlay() != nil {
		t.Error("the stack is not empty")
	}
}

// Escape closes the top one only.
func TestEscapeClosesTheTopDialogOnly(t *testing.T) {
	a, w := stackWindow(t)
	first := widgets.NewMessageBox(widgets.MessageBoxOptions{
		Title: "One", Message: "one", Buttons: widgets.ButtonsOKCancel,
	})
	first.Show(w.Content())
	a.PumpOnce()
	second := widgets.NewMessageBox(widgets.MessageBoxOptions{
		Title: "Two", Message: "two", Buttons: widgets.ButtonsOKCancel,
	})
	second.Show(first.Overlay())
	a.PumpOnce()

	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	a.PumpOnce()
	if got := len(w.Overlays()); got != 1 {
		t.Fatalf("%d overlays after Escape, want 1", got)
	}
}

// SetOverlay is still "there is one dialog": it clears the whole stack.
func TestSetOverlayReplacesTheWholeStack(t *testing.T) {
	a, w := stackWindow(t)
	closed := 0
	for i := 0; i < 3; i++ {
		mb := widgets.NewMessageBox(widgets.MessageBoxOptions{
			Title: "x", Message: "y", Buttons: widgets.ButtonsOK,
			OnResult: func(widgets.MessageResult) { closed++ },
		})
		if i == 0 {
			mb.Show(w.Content())
		} else {
			mb.Show(w.Overlay())
		}
		a.PumpOnce()
	}
	if len(w.Overlays()) != 3 {
		t.Fatalf("%d overlays", len(w.Overlays()))
	}
	w.SetOverlay(nil)
	a.PumpOnce()
	if len(w.Overlays()) != 0 {
		t.Errorf("%d overlays after SetOverlay(nil)", len(w.Overlays()))
	}
	if closed != 3 {
		t.Errorf("%d of 3 dialogs were told they closed", closed)
	}
}

// Before overlays stacked, DismissOverlay(from) closed the window's
// overlay whatever from was — a program passed its content's root to
// take down whatever was on screen as it quit. Popping only the overlay
// from is in silently broke that: a program quitting with a recovery key
// up stopped wiping it, and a cancelled file chooser stayed up.
func TestDismissFromOutsideEveryOverlayClosesTheTop(t *testing.T) {
	a, w := stackWindow(t)
	closed := 0
	mb := widgets.NewMessageBox(widgets.MessageBoxOptions{
		Title: "Recovery key", Message: "write this down",
		Buttons:  widgets.ButtonsOK,
		OnResult: func(widgets.MessageResult) { closed++ },
	})
	mb.Show(w.Content())
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("the dialog did not show")
	}

	// From the window's content, which is in no overlay at all.
	widget.DismissOverlay(w.Content())
	a.PumpOnce()
	if closed != 1 {
		t.Errorf("the dialog reported closing %d times", closed)
	}
	if w.Overlay() != nil || len(w.Overlays()) != 0 {
		t.Error("the dialog is still up")
	}
}

// And it takes the top one only, leaving what is under it.
func TestDismissFromOutsideTakesOnlyTheTop(t *testing.T) {
	a, w := stackWindow(t)
	first := widgets.NewMessageBox(widgets.MessageBoxOptions{
		Title: "One", Message: "one", Buttons: widgets.ButtonsOK,
	})
	first.Show(w.Content())
	a.PumpOnce()
	second := widgets.NewMessageBox(widgets.MessageBoxOptions{
		Title: "Two", Message: "two", Buttons: widgets.ButtonsOK,
	})
	second.Show(first.Overlay())
	a.PumpOnce()

	widget.DismissOverlay(w.Content())
	a.PumpOnce()
	if got := len(w.Overlays()); got != 1 {
		t.Errorf("%d overlays left, want 1", got)
	}
}
