package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/widget"
)

func TestTextFieldIMEPreeditCommit(t *testing.T) {
	tf := NewTextField("ab", "", nil)
	tf.SetHost(&host{})
	tf.Arrange(paintengine2d.XYWH(0, 0, 200, 32))
	tf.SetSelection(2, 2)
	tf.IMEPreedit("に", 1)
	if tf.Preedit() != "に" {
		t.Fatalf("preedit %q", tf.Preedit())
	}
	vis, caret, a, b := tf.visual()
	if vis != "abに" || caret != 3 || a != 2 || b != 3 {
		t.Fatalf("visual %q %d [%d,%d]", vis, caret, a, b)
	}
	tf.IMECommit("に")
	if tf.Text != "abに" || tf.Preedit() != "" {
		t.Fatalf("commit %q pre=%q", tf.Text, tf.Preedit())
	}
}

func TestTextFieldIMEResetOnFocusLost(t *testing.T) {
	tf := NewTextField("", "", nil)
	tf.SetHost(&host{})
	tf.IMEPreedit("x", 1)
	tf.FocusLost()
	if tf.Preedit() != "" {
		t.Fatal("reset")
	}
}

func TestTextAreaIMECommit(t *testing.T) {
	ta := NewTextArea("hi", "", nil)
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 220, 80))
	ta.SetSelection(2, 2)
	ta.IMEPreedit("ん", 1)
	ta.IMECommit("ん")
	if ta.Text != "hiん" {
		t.Fatalf("%q", ta.Text)
	}
}

func TestTextFieldIMEDeleteSurrounding(t *testing.T) {
	tf := NewTextField("hello", "", nil)
	tf.SetHost(&host{})
	tf.SetSelection(5, 5)
	tf.IMEDeleteSurrounding(2, 0)
	if tf.Text != "hel" {
		t.Fatalf("%q", tf.Text)
	}
}

func TestIMETargetSatisfied(t *testing.T) {
	var _ widget.IMETarget = (*TextField)(nil)
	var _ widget.IMETarget = (*TextArea)(nil)
}
