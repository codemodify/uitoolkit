package widgets

import (
	"errors"
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// A prompt could only dismiss and report, so a value the caller had to
// refuse — "a folder with that name already exists" — meant a second
// dialog complaining and then the prompt opened again with the text put
// back. The refusal belongs where the value was typed.
func TestPromptValidateKeepsTheDialogUp(t *testing.T) {
	var got string
	var closed bool
	mb := NewMessageBox(MessageBoxOptions{
		Title: "New folder", Message: "Name", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{
			Text: "Drafts",
			Validate: func(s string) error {
				if s == "Drafts" {
					return errors.New("a folder with that name exists")
				}
				return nil
			},
		},
		OnResult: func(MessageResult) { closed = true },
	})
	mb.SetHost(&scaleHost{look: style.WithScale(style.DarkLook(), 1)})

	mb.accept(ResultOK)
	if closed {
		t.Fatal("the dialog closed on a refused value")
	}
	if mb.InputError() != "a folder with that name exists" {
		t.Errorf("error shown: %q", mb.InputError())
	}
	if mb.Text() != "Drafts" {
		t.Errorf("the typed value was lost: %q", mb.Text())
	}

	// Fix it and it goes through.
	mb.field.SetText("Archive")
	mb.accept(ResultOK)
	if !closed {
		t.Fatal("the dialog did not close on an accepted value")
	}
	got = mb.Text()
	if got != "Archive" {
		t.Errorf("value %q", got)
	}
}

// Cancel is never validated: refusing to let someone out of a dialog is
// not what a check is for.
func TestPromptValidateDoesNotBlockCancel(t *testing.T) {
	closed := false
	mb := NewMessageBox(MessageBoxOptions{
		Title: "New folder", Message: "Name", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{
			Text:     "x",
			Validate: func(string) error { return errors.New("no") },
		},
		OnResult: func(MessageResult) { closed = true },
	})
	mb.SetHost(&scaleHost{look: style.WithScale(style.DarkLook(), 1)})
	mb.accept(ResultCancel)
	if !closed {
		t.Error("Cancel was refused by the value check")
	}
}

// A button names its action, which is what every desktop's guidelines
// say.
func TestPromptAcceptLabel(t *testing.T) {
	mb := NewMessageBox(MessageBoxOptions{
		Title: "Rename", Message: "Name", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{Text: "a", AcceptLabel: "Rename"},
	})
	if mb.primary == nil || mb.primary.Text != "Rename" {
		t.Errorf("the accepting button says %q", mb.primary.Text)
	}

	plain := NewMessageBox(MessageBoxOptions{
		Title: "x", Message: "y", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{Text: "a"},
	})
	if plain.primary.Text != "OK" {
		t.Errorf("without a label it should stay OK, got %q", plain.primary.Text)
	}
}

// The error line takes no room until there is one.
func TestPromptErrorLineIsHiddenUntilNeeded(t *testing.T) {
	mb := NewMessageBox(MessageBoxOptions{
		Title: "x", Message: "y", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{Text: "a"},
	})
	if mb.errLabel == nil {
		t.Fatal("no error line was made")
	}
	if mb.errLabel.Visible() {
		t.Error("the error line is visible before there is an error")
	}
	mb.SetInputError("bad")
	if !mb.errLabel.Visible() || mb.InputError() != "bad" {
		t.Error("the error line did not appear")
	}
	mb.SetInputError("")
	if mb.errLabel.Visible() {
		t.Error("the error line stayed after being cleared")
	}
}

// A check that has to ask a server cannot answer on the UI goroutine.
// The documented route before this was to return a non-nil error to keep
// the dialog up — errors.New("") happened to work, because an empty
// message hides the label — and then dismiss the overlay by hand, which
// took the dialog off the screen without running OnResult at all. It
// worked; it was not a pattern anyone would find.
func TestPromptValidateAsync(t *testing.T) {
	var answer func(error)
	var result MessageResult
	closed := 0

	mb := NewMessageBox(MessageBoxOptions{
		Title: "New folder", Message: "Name", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{
			Text:          "Drafts",
			ValidateAsync: func(_ string, done func(error)) { answer = done },
		},
		OnResult: func(r MessageResult) { result, closed = r, closed+1 },
	})
	mb.SetHost(&scaleHost{look: style.WithScale(style.DarkLook(), 1)})

	mb.accept(ResultOK)
	if answer == nil {
		t.Fatal("the check was never started")
	}
	if !mb.Checking() {
		t.Error("the dialog is not marked as waiting")
	}
	if mb.primary.Enabled() {
		t.Error("the accepting button is still live while a check is out")
	}
	if closed != 0 {
		t.Fatal("the dialog closed before the answer came")
	}

	// A second press must not start a second check nor close anything.
	first := answer
	mb.accept(ResultOK)
	if closed != 0 {
		t.Error("a second press closed the dialog")
	}

	// The server refuses.
	first(errors.New("a folder with that name exists"))
	if closed != 0 {
		t.Error("a refusal closed the dialog")
	}
	if mb.InputError() != "a folder with that name exists" {
		t.Errorf("error shown: %q", mb.InputError())
	}
	if mb.Checking() {
		t.Error("still waiting after the answer")
	}
	if !mb.primary.Enabled() {
		t.Error("the accepting button was not re-enabled")
	}
	if mb.Text() != "Drafts" {
		t.Errorf("the typed value was lost: %q", mb.Text())
	}

	// Try again, and this time it is accepted.
	mb.field.SetText("Archive")
	mb.accept(ResultOK)
	answer(nil)
	if closed != 1 {
		t.Fatalf("the dialog reported closing %d times", closed)
	}
	if result != ResultOK {
		t.Errorf("result %v, want OK — OnResult must run as it would have", result)
	}
	if mb.Text() != "Archive" {
		t.Errorf("value %q", mb.Text())
	}
}

// done called twice must not report a second result.
func TestPromptValidateAsyncAnswersOnce(t *testing.T) {
	var answer func(error)
	closed := 0
	mb := NewMessageBox(MessageBoxOptions{
		Title: "x", Message: "y", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{
			Text:          "a",
			ValidateAsync: func(_ string, done func(error)) { answer = done },
		},
		OnResult: func(MessageResult) { closed++ },
	})
	mb.SetHost(&scaleHost{look: style.WithScale(style.DarkLook(), 1)})
	mb.accept(ResultOK)
	answer(nil)
	answer(nil)
	answer(errors.New("late"))
	if closed != 1 {
		t.Errorf("OnResult ran %d times, want 1", closed)
	}
}

// Cancel is never checked: a check is not a reason to refuse someone the
// way out.
func TestPromptValidateAsyncDoesNotBlockCancel(t *testing.T) {
	started := false
	closed := 0
	mb := NewMessageBox(MessageBoxOptions{
		Title: "x", Message: "y", Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{
			Text:          "a",
			ValidateAsync: func(string, func(error)) { started = true },
		},
		OnResult: func(MessageResult) { closed++ },
	})
	mb.SetHost(&scaleHost{look: style.WithScale(style.DarkLook(), 1)})
	mb.accept(ResultCancel)
	if started {
		t.Error("Cancel started a check")
	}
	if closed != 1 {
		t.Error("Cancel was refused")
	}
}

// Close finishes a dialog from outside a button — an answer that
// arrived, a vault that locked — and runs OnResult, which dismissing the
// overlay by hand does not.
func TestMessageBoxClose(t *testing.T) {
	var got MessageResult
	closed := 0
	mb := NewMessageBox(MessageBoxOptions{
		Title: "x", Message: "y", Buttons: ButtonsOKCancel,
		OnResult: func(r MessageResult) { got, closed = r, closed+1 },
	})
	mb.SetHost(&scaleHost{look: style.WithScale(style.DarkLook(), 1)})

	mb.Close(ResultCancel)
	if closed != 1 || got != ResultCancel {
		t.Errorf("closed %d times with %v", closed, got)
	}
	mb.Close(ResultOK)
	if closed != 1 {
		t.Error("closing an already-closed dialog reported again")
	}
}
