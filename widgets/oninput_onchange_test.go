package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// An application that normalises a value from inside OnChange is making a
// programmatic change, not a user one.
//
// OnChange ran with the user mark still up, so the application's own write was
// counted as input: one key press produced two OnInput calls, and the last of
// them carried the value the control had already stopped holding. A player's
// fader would act on 51 after the application had settled on 60.
func TestNormalisingInsideOnChangeIsNotInput(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetLook(style.DarkLook())
	s.Step = 1
	var changes, inputs []float32
	s.OnChange = func(v float32) {
		changes = append(changes, v)
		if v == 51 {
			s.SetValue(60)
		}
	}
	s.OnInput = func(v float32) { inputs = append(inputs, v) }

	s.KeyPress(widget.KeyEvent{Key: platform.KeyRight})

	if got := fmt.Sprint(inputs); got != "[60]" {
		t.Errorf("OnInput saw %s, want one call carrying the value the slider ended up with", got)
	}
	if got := fmt.Sprint(changes); got != "[51 60]" {
		t.Errorf("OnChange saw %s, want both the user's value and the normalised one", got)
	}
	if s.Value != 60 {
		t.Errorf("the slider holds %g, not the normalised 60", s.Value)
	}
}

// And the same from inside OnInput, which was fixed first and must stay fixed:
// one input call, the normalisation after it.
func TestNormalisingInsideOnInputIsNotInput(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetLook(style.DarkLook())
	s.Step = 1
	var inputs []float32
	s.OnInput = func(v float32) {
		inputs = append(inputs, v)
		if v == 51 {
			s.SetValue(60)
		}
	}

	s.KeyPress(widget.KeyEvent{Key: platform.KeyRight})

	if got := fmt.Sprint(inputs); got != "[51]" {
		t.Errorf("OnInput saw %s, want the one user value", got)
	}
	if s.Value != 60 {
		t.Errorf("the slider holds %g, not the normalised 60", s.Value)
	}
}

// A handler that puts the value back has ended the edit where it began, so
// there is no input to report: an application that rejects a change must not
// be told the user made one.
func TestAnOnChangeThatRejectsTheEditReportsNoInput(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetLook(style.DarkLook())
	s.Step = 1
	var inputs []float32
	s.OnChange = func(v float32) {
		if v != 50 {
			s.SetValue(50)
		}
	}
	s.OnInput = func(v float32) { inputs = append(inputs, v) }

	s.KeyPress(widget.KeyEvent{Key: platform.KeyRight})

	if len(inputs) != 0 {
		t.Errorf("OnInput saw %v after the change was put back", inputs)
	}
	if s.Value != 50 {
		t.Errorf("the slider holds %g, not the 50 it was put back to", s.Value)
	}
}

// An ordinary user edit still reports itself, which is the whole point: a test
// that only checked the cases above would pass on a control that never fired
// OnInput at all.
func TestAnOrdinaryEditStillReportsInput(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetLook(style.DarkLook())
	s.Step = 1
	var inputs []float32
	s.OnInput = func(v float32) { inputs = append(inputs, v) }

	s.KeyPress(widget.KeyEvent{Key: platform.KeyRight})

	if got := fmt.Sprint(inputs); got != "[51]" {
		t.Errorf("OnInput saw %s, want [51]", got)
	}
}

// And a value the application sets is never input, whatever its handlers do.
func TestAProgrammaticSetIsNeverInput(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetLook(style.DarkLook())
	var inputs []float32
	s.OnChange = func(v float32) {
		if v == 70 {
			s.SetValue(80)
		}
	}
	s.OnInput = func(v float32) { inputs = append(inputs, v) }

	s.SetValue(70)

	if len(inputs) != 0 {
		t.Errorf("OnInput saw %v for a value the application set", inputs)
	}
}

// The same contract on the controls that carry a bool and a string, since the
// report asked for every control that shares it.
func TestNormalisingInsideOnChangeIsNotInputOnOtherControls(t *testing.T) {
	t.Run("checkbox", func(t *testing.T) {
		c := NewCheckbox("Wrap", false, nil)
		c.SetLook(style.DarkLook())
		var inputs []bool
		// A box that refuses to be ticked: the application puts it back.
		c.OnChange = func(v bool) {
			if v {
				c.SetChecked(false)
			}
		}
		c.OnInput = func(v bool) { inputs = append(inputs, v) }
		c.user.did(func() { c.SetChecked(true) })
		if len(inputs) != 0 {
			t.Errorf("OnInput saw %v after the tick was put back", inputs)
		}
	})
	t.Run("text field", func(t *testing.T) {
		f := NewTextField("", "", nil)
		f.SetLook(style.DarkLook())
		var inputs []string
		f.OnChange = func(s string) {
			if s == "abc" {
				f.SetText("ABC")
			}
		}
		f.OnInput = func(s string) { inputs = append(inputs, s) }
		f.user.did(func() { f.SetText("abc") })
		if got := fmt.Sprint(inputs); got != "[ABC]" {
			t.Errorf("OnInput saw %s, want the text the field ended up with", got)
		}
	})
}
