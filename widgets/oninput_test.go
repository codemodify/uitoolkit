package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// tally counts what each callback saw, so a test can say "OnChange twice,
// OnInput once" rather than "it fired".
type tally struct{ change, input int }

// press is a click on a control that takes one where the pointer is.
func press(c interface {
	MousePress(widget.MouseEvent) bool
	MouseRelease(widget.MouseEvent) bool
}, p paintengine2d.Point) {
	c.MousePress(widget.MouseEvent{Pos: p})
	c.MouseRelease(widget.MouseEvent{Pos: p})
}

func (n *tally) check(t *testing.T, who string, change, input int) {
	t.Helper()
	if n.change != change || n.input != input {
		t.Errorf("%s: OnChange %d (want %d), OnInput %d (want %d)", who, n.change, change, n.input, input)
	}
}

func TestSliderOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	s := NewSlider(0, 100, 0, func(float32) { n.change++ })
	s.OnInput = func(float32) { n.input++ }
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))

	s.SetValue(10) // the app, from a model
	n.check(t, "SetValue", 1, 0)

	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(100, 14)})
	n.check(t, "a press on the track", 2, 1)
	s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(150, 14)})
	n.check(t, "a drag", 3, 2)
	s.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(150, 14)})

	s.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	n.check(t, "Home", 4, 3)

	s.SetValue(42)
	n.check(t, "SetValue after a drag", 5, 3)
}

// OnChange comes first and both see the new value: that is the order the
// whole family keeps.
func TestOnInputComesAfterOnChange(t *testing.T) {
	var order []string
	var atChange, atInput float32
	s := NewSlider(0, 100, 0, nil)
	s.OnChange = func(v float32) { order = append(order, "change"); atChange = s.Value }
	s.OnInput = func(v float32) { order = append(order, "input"); atInput = s.Value }
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
	s.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	if len(order) != 2 || order[0] != "change" || order[1] != "input" {
		t.Fatalf("order %v, want change then input", order)
	}
	if atChange != 100 || atInput != 100 {
		t.Errorf("the value seen was %v / %v, want 100 from both", atChange, atInput)
	}
}

func TestNumberFieldOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	f := NewNumberField(0, 100, 5, 1, func(float64) { n.change++ })
	f.OnInput = func(float64) { n.input++ }
	f.SetHost(&host{})
	f.Arrange(paintengine2d.XYWH(0, 0, 120, 30))

	f.SetValue(9)
	n.check(t, "SetValue", 1, 0)

	f.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	n.check(t, "Up", 2, 1)

	f.Field().SetHost(&host{})
	f.Field().RequestFocus()
	f.Field().TextInput('7')
	if n.input != 2 {
		t.Errorf("typing in the editor reported OnInput %d times, want 2", n.input)
	}
}

func TestCheckboxOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	c := NewCheckbox("X", false, func(bool) { n.change++ })
	c.OnInput = func(bool) { n.input++ }
	c.SetHost(&host{})
	c.Arrange(paintengine2d.XYWH(0, 0, 120, 32))

	c.SetChecked(true)
	n.check(t, "SetChecked", 1, 0)

	press(c, paintengine2d.Pt(8, 16))
	n.check(t, "a click", 2, 1)

	c.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	n.check(t, "Space", 3, 2)

	c.AccessibleAction(0, a11y.ActionDefault)
	n.check(t, "a screen reader's press", 4, 3)
}

func TestSwitchOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	s := NewSwitch("X", false, func(bool) { n.change++ })
	s.OnInput = func(bool) { n.input++ }
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 120, 32))

	s.SetOn(true)
	n.check(t, "SetOn", 1, 0)

	press(s, paintengine2d.Pt(8, 16))
	n.check(t, "a click", 2, 1)

	s.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	n.check(t, "Space", 3, 2)

	// One flip, one OnChange: the accessibility action used to call the
	// app's handler a second time on top of the setter's.
	s.AccessibleAction(0, a11y.ActionDefault)
	n.check(t, "a screen reader's press", 4, 3)
}

func TestRadioOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	r := NewRadio("A", false, func(bool) { n.change++ })
	r.OnInput = func(bool) { n.input++ }
	r.SetHost(&host{})
	r.Arrange(paintengine2d.XYWH(0, 0, 120, 32))

	r.SetSelected(true)
	n.check(t, "SetSelected", 1, 0)

	r.SetSelected(false)
	press(r, paintengine2d.Pt(8, 16))
	n.check(t, "a click", 3, 1)

	r.SetSelected(false)
	r.AccessibleAction(0, a11y.ActionDefault)
	n.check(t, "a screen reader's press", 5, 2)
}

func TestRadioGroupOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	g := NewRadioGroup([]string{"A", "B", "C"}, 0, func(int) { n.change++ })
	g.OnInput = func(int) { n.input++ }
	g.SetHost(&host{})
	g.Arrange(paintengine2d.XYWH(0, 0, 120, 120))

	g.Select(1)
	n.check(t, "Select", 1, 0)

	// The radio the user pressed reports for itself too.
	own := 0
	g.Buttons()[2].OnInput = func(bool) { own++ }
	press(g.Buttons()[2], paintengine2d.Pt(8, 8))
	n.check(t, "a click on a radio", 2, 1)
	if own != 1 {
		t.Errorf("the pressed radio reported OnInput %d times, want 1", own)
	}

	// An arrow through the group is the user's too.
	g.Buttons()[2].KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	n.check(t, "an arrow key", 3, 2)
}

func TestTextFieldOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	f := NewTextField("Hi", "", func(string) { n.change++ })
	f.OnInput = func(string) { n.input++ }
	f.SetHost(&host{})
	f.Arrange(paintengine2d.XYWH(0, 0, 160, 32))

	f.SetText("Model")
	n.check(t, "SetText", 1, 0)

	f.TextInput('!')
	n.check(t, "typing", 2, 1)

	f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	n.check(t, "backspace", 3, 2)

	f.AccessibleSetText("Read aloud")
	n.check(t, "a screen reader's edit", 4, 3)
}

func TestTextAreaOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	a := NewTextArea("Hi", "", func(string) { n.change++ })
	a.OnInput = func(string) { n.input++ }
	a.SetHost(&host{})
	a.Arrange(paintengine2d.XYWH(0, 0, 200, 120))

	a.SetText("Model")
	n.check(t, "SetText", 1, 0)

	a.TextInput('!')
	n.check(t, "typing", 2, 1)

	a.AccessibleSetText("Read aloud")
	n.check(t, "a screen reader's edit", 3, 2)
}

func TestExpanderOnInputIsTheUsersOwn(t *testing.T) {
	var n tally
	e := NewExpander("Section", false, nil)
	e.OnToggle = func(bool) { n.change++ }
	e.OnInput = func(bool) { n.input++ }
	e.SetHost(&host{})
	e.Arrange(paintengine2d.XYWH(0, 0, 200, 40))

	e.SetExpanded(true)
	n.check(t, "SetExpanded", 1, 0)

	e.head.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 8)})
	n.check(t, "a click on the header", 2, 1)

	e.head.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	n.check(t, "Space", 3, 2)

	e.head.AccessibleAction(0, a11y.ActionDefault)
	n.check(t, "a screen reader's press", 4, 3)
}

// An exclusive accordion closing a section to open another is the
// accordion's doing, not the user's: only the section the user opened
// reports input.
func TestAccordionClosingOthersIsNotTheUsersInput(t *testing.T) {
	one := NewExpander("One", true, nil)
	two := NewExpander("Two", false, nil)
	closed, opened := 0, 0
	one.OnInput = func(bool) { closed++ }
	two.OnInput = func(bool) { opened++ }
	a := NewAccordion(true, one, two)
	a.SetHost(&host{})
	a.Arrange(paintengine2d.XYWH(0, 0, 200, 120))

	two.head.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 8)})
	if opened != 1 {
		t.Errorf("the section the user opened reported OnInput %d times, want 1", opened)
	}
	if closed != 0 {
		t.Errorf("the section the accordion closed reported OnInput %d times, want none", closed)
	}
	if one.Expanded {
		t.Error("the exclusive accordion did not close the other section")
	}
}

// The mark does not leak: a spin button writing its inner field is not the
// field's user, and an app setting a value from inside a user handler is
// still the app.
func TestTheUserMarkDoesNotLeak(t *testing.T) {
	f := NewNumberField(0, 100, 5, 1, nil)
	f.SetHost(&host{})
	f.Arrange(paintengine2d.XYWH(0, 0, 120, 30))
	inner := 0
	f.Field().OnInput = func(string) { inner++ }
	f.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if inner != 0 {
		t.Errorf("the spin button's own write reported %d edits of the inner field", inner)
	}

	other := NewSlider(0, 100, 0, nil)
	other.SetHost(&host{})
	other.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
	driven := 0
	other.OnInput = func(float32) { driven++ }
	c := NewCheckbox("Follow", false, nil)
	c.OnInput = func(bool) { other.SetValue(50) }
	c.SetHost(&host{})
	c.Arrange(paintengine2d.XYWH(0, 0, 120, 32))
	press(c, paintengine2d.Pt(8, 16))
	if driven != 0 {
		t.Errorf("a slider the app moved from a user handler reported %d inputs", driven)
	}
}
