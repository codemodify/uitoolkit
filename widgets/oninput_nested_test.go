package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// OnInput fires once for one piece of user input, whatever the callback
// does to the control.
//
// An application that normalises a value by calling SetValue from inside
// OnInput is making a programmatic change — but the user-edit flag stayed
// up for the whole callback, so the nested SetValue looked like more user
// input and OnInput ran again. One key press, two callbacks. A callback
// that always adjusts the value recursed until the value stopped moving.
func TestOnInputFiresOncePerUserAction(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.Step = 1
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	calls := 0
	s.OnInput = func(v float32) {
		calls++
		if v == 51 {
			s.SetValue(60) // the application's own normalisation
		}
	}
	s.KeyPress(widget.KeyEvent{Key: platform.KeyRight})

	if calls != 1 {
		t.Errorf("one key press gave %d OnInput calls, want 1", calls)
	}
	if s.Value != 60 {
		t.Errorf("the value is %v, want the 60 the callback set", s.Value)
	}
}

// And the same for the other controls that have the pair: all of them
// shared the flag, so all of them shared the bug.
func TestOnInputFiresOncePerUserActionOnACheckbox(t *testing.T) {
	c := NewCheckbox("x", false, nil)
	c.SetLook(style.DarkLook())
	c.SetHost(&host{})
	calls := 0
	c.OnInput = func(on bool) {
		calls++
		c.SetChecked(!on) // an application vetoing the change
	}
	c.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if calls != 1 {
		t.Errorf("one press gave %d OnInput calls, want 1", calls)
	}
}
