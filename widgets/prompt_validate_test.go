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
