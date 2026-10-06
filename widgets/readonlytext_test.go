package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Text to select and copy but not to edit — a file path in a settings page, a
// key, an address — had no widget. A Label cannot be selected; a TextField had
// no read-only mode, and refusing through Accept was not watertight; a
// NewTextView is read-only and selectable but was MinRows tall whatever it
// held, so a path wider than the box wrapped onto a second line hidden behind
// a scroll bar.

func readOnlyField(t *testing.T, text string) *TextField {
	t.Helper()
	f := NewTextField("", "", nil)
	f.Text = text
	f.ReadOnly = true
	f.SetLook(style.DarkLook())
	f.SetHost(&host{})
	f.Arrange(paintengine2d.XYWH(0, 0, 300, 28))
	f.SelectAll()
	return f
}

// Every way a user can change the text is refused — including the two that
// went round Accept, which is what made the Accept trick a trap.
func TestAReadOnlyFieldRefusesEveryEdit(t *testing.T) {
	const path = "/home/person/.local/share/vault/secrets.enc"
	for _, tc := range []struct {
		what string
		do   func(f *TextField)
	}{
		{"typing", func(f *TextField) { f.TextInput('x') }},
		{"Backspace with nothing selected", func(f *TextField) {
			f.selA, f.selB = f.caret, f.caret
			f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
		}},
		{"Delete with nothing selected", func(f *TextField) {
			f.caret, f.selA, f.selB = 0, 0, 0
			f.KeyPress(widget.KeyEvent{Key: platform.KeyDelete})
		}},
		{"Backspace over a selection", func(f *TextField) {
			f.SelectAll()
			f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
		}},
		{"an IME's delete-surrounding", func(f *TextField) { f.IMEDeleteSurrounding(3, 0) }},
		{"an IME commit", func(f *TextField) { f.IMECommit("xyz") }},
	} {
		f := readOnlyField(t, path)
		f.caret = runeCount(f.Text)
		tc.do(f)
		if f.Text != path {
			t.Errorf("%s changed the text to %q", tc.what, f.Text)
		}
	}
}

// But it is not disabled: it takes the focus, the caret moves, and the text
// can be selected and copied. That is the whole reason for it.
func TestAReadOnlyFieldCanStillBeSelectedAndCopied(t *testing.T) {
	const path = "/home/person/.local/share/vault/secrets.enc"
	f := readOnlyField(t, path)
	if !f.Enabled() {
		t.Error("a read-only field is disabled")
	}
	if got := f.SelectedText(); got != path {
		t.Errorf("Ctrl+A selected %q, want the whole text", got)
	}
	// Caret movement still works, which is how a selection is made by hand.
	f.caret, f.selA, f.selB = 0, 0, 0
	f.KeyPress(widget.KeyEvent{Key: platform.KeyRight, Mods: platform.ModShift})
	if f.SelectedText() == "" {
		t.Error("Shift+Right selected nothing")
	}
	// And the cross never appears on one.
	f.Clearable = true
	if f.clearShows() {
		t.Error("a read-only field offered a clear cross")
	}
}

// An ordinary field is unaffected: this is a new refusal, not a new default.
func TestAnOrdinaryFieldStillEdits(t *testing.T) {
	f := NewTextField("", "", nil)
	f.Text = "abc"
	f.SetLook(style.DarkLook())
	f.SetHost(&host{})
	f.caret = 3
	f.TextInput('d')
	if f.Text != "abcd" {
		t.Errorf("an ordinary field typed to %q", f.Text)
	}
	f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if f.Text != "abc" {
		t.Errorf("Backspace left %q", f.Text)
	}
}

// A view that fits its text is as tall as what it holds, not as tall as
// MinRows: one line for a path, two when it wraps.
func TestATextAreaCanFitItsText(t *testing.T) {
	const path = "/home/person/.local/share/comms-mail/accounts/work/secrets.enc"
	fit := NewTextView(path, "")
	fit.MinRows, fit.FitRows = 1, true
	fit.SetLook(style.DarkLook())
	fit.SetHost(&host{})

	wide := fit.Measure(layout.Loose(900, 600))
	narrow := fit.Measure(layout.Loose(220, 600))
	if narrow.Y <= wide.Y {
		t.Errorf("fitted to %.0f at 900 wide and %.0f at 220: a wrapped path needs more room", wide.Y, narrow.Y)
	}
	// One line at a width that holds it, rather than MinRows' three.
	plain := NewTextView(path, "")
	plain.MinRows = 3
	plain.SetLook(style.DarkLook())
	plain.SetHost(&host{})
	if wide.Y >= plain.Measure(layout.Loose(900, 600)).Y {
		t.Errorf("fitting measured %.0f and MinRows=3 measured %.0f: fitting saved nothing",
			wide.Y, plain.Measure(layout.Loose(900, 600)).Y)
	}
	// And MinRows is still the floor.
	floor := NewTextView("one line", "")
	floor.MinRows, floor.FitRows = 4, true
	floor.SetLook(style.DarkLook())
	floor.SetHost(&host{})
	if got, want := floor.Measure(layout.Loose(900, 600)).Y, plain.Measure(layout.Loose(900, 600)).Y; got <= want {
		t.Errorf("MinRows=4 fitted to %.0f, which is not above three rows' %.0f", got, want)
	}
}

// MaxRows is the ceiling, and past it the view scrolls as it always did.
func TestFitRowsStopsAtMaxRows(t *testing.T) {
	long := ""
	for i := 0; i < 60; i++ {
		long += "a line of a message that goes on\n"
	}
	v := NewTextView(long, "")
	v.MinRows, v.FitRows, v.MaxRows = 1, true, 6
	v.SetLook(style.DarkLook())
	v.SetHost(&host{})
	got := v.Measure(layout.Loose(400, 4000)).Y
	capped := float32(6)*v.lineH() + v.fieldPad()*2
	if got != capped {
		t.Errorf("fitted to %.1f with MaxRows=6, want %.1f", got, capped)
	}
}

// And measuring it changes nothing about what it shows (ScrollView's bug).
func TestMeasuringAFittedViewWritesNothingDown(t *testing.T) {
	v := NewTextView("a short line of text that is nonetheless long enough to wrap in a narrow box", "")
	v.MinRows, v.FitRows = 1, true
	v.SetLook(style.DarkLook())
	v.SetHost(&host{})
	v.Arrange(paintengine2d.XYWH(0, 0, 400, 80))
	before := len(v.Lines())
	v.Measure(layout.Loose(60, 600))
	if after := len(v.Lines()); after != before {
		t.Errorf("a narrow measure left the view showing %d lines, was %d", after, before)
	}
}
