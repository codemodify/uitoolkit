package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// comboWheelOff restores the process-wide preference after a test that
// turns it on. It is a process flag, like style.Animations, so a test
// that leaves it set changes every test that runs after it.
func comboWheelOff(t *testing.T) {
	t.Helper()
	was := style.ComboWheel()
	t.Cleanup(func() { style.SetComboWheel(was) })
	style.SetComboWheel(false)
}

func wheelDown() widget.MouseEvent { return widget.MouseEvent{Scroll: paintengine2d.Pt(0, 1)} }
func wheelUp() widget.MouseEvent   { return widget.MouseEvent{Scroll: paintengine2d.Pt(0, -1)} }

func newTestCombo(h *host) *ComboBox {
	cb := NewComboBox([]string{"one", "two", "three"}, 1, nil)
	cb.SetHost(h)
	cb.Arrange(paintengine2d.XYWH(0, 0, 180, 30))
	return cb
}

// Off is the default, and off means the notch is not ours: a combo box
// that is only being scrolled past keeps its value and lets whatever is
// scrolling behind it scroll.
func TestComboWheelIsOffUntilAskedFor(t *testing.T) {
	comboWheelOff(t)
	h := &host{}
	cb := newTestCombo(h)
	if cb.MouseWheel(wheelDown()) {
		t.Error("a combo box swallowed the wheel with the preference off")
	}
	if cb.Selected != 1 {
		t.Errorf("the selection moved to %d with the preference off", cb.Selected)
	}
}

func TestComboWheelStepsWhenAskedFor(t *testing.T) {
	comboWheelOff(t)
	style.SetComboWheel(true)
	h := &host{}
	var got []int
	cb := newTestCombo(h)
	cb.OnChange = func(i int) { got = append(got, i) }

	if !cb.MouseWheel(wheelDown()) {
		t.Fatal("the wheel down did nothing with the preference on")
	}
	if cb.Selected != 2 {
		t.Errorf("wheel down selected %d, want the next item", cb.Selected)
	}
	if !cb.MouseWheel(wheelUp()) {
		t.Fatal("the wheel up did nothing")
	}
	if !cb.MouseWheel(wheelUp()) {
		t.Fatal("the second wheel up did nothing")
	}
	if cb.Selected != 0 {
		t.Errorf("two notches up from the last item is %d, want the first", cb.Selected)
	}
	// Every step is a change the application hears about, once each.
	if len(got) != 3 {
		t.Errorf("OnChange fired %d times for three notches: %v", len(got), got)
	}
}

// The rule that keeps the option honest. A combo box with nowhere left
// to go hands the notch back, so a page behind it goes on scrolling
// instead of stopping dead the moment the pointer crosses a drop-down.
func TestComboWheelDoesNotEatTheNotchAtItsEnds(t *testing.T) {
	comboWheelOff(t)
	style.SetComboWheel(true)
	h := &host{}
	cb := newTestCombo(h)

	cb.Select(0)
	if cb.MouseWheel(wheelUp()) {
		t.Error("the first item swallowed a wheel up")
	}
	if cb.Selected != 0 {
		t.Errorf("a wheel up at the first item selected %d", cb.Selected)
	}
	cb.Select(len(cb.Items) - 1)
	if cb.MouseWheel(wheelDown()) {
		t.Error("the last item swallowed a wheel down")
	}
	if cb.Selected != len(cb.Items)-1 {
		t.Errorf("a wheel down at the last item selected %d", cb.Selected)
	}
}

// And the same thing said where it matters: a scroll view with a combo
// box in it scrolls under the pointer, whether the preference is off or
// the box has run out of items.
func TestComboWheelLeavesTheScrollViewScrolling(t *testing.T) {
	comboWheelOff(t)
	h := &host{}
	cb := NewComboBox([]string{"one", "two"}, 0, nil)
	tall := NewColumn(NewLabel("a"), cb, NewLabel("b")).WithGap(8)
	for i := 0; i < 40; i++ {
		tall.Add(NewLabel("filler"))
	}
	sv := NewScrollView(tall)
	sv.SetHost(h)
	sv.Measure(layout.Loose(240, 120))
	sv.Arrange(paintengine2d.XYWH(0, 0, 240, 120))
	if sv.MaxOffset() <= 0 {
		t.Fatal("the fixture does not scroll")
	}

	// The window hands the notch to the deepest component and walks up,
	// so what matters is that the combo declines it.
	for _, on := range []bool{false, true} {
		style.SetComboWheel(on)
		cb.Select(len(cb.Items) - 1) // nowhere further to go on a wheel down
		before := sv.OffsetY
		if cb.MouseWheel(wheelDown()) {
			t.Fatalf("preference %v: the combo took a notch it had no item for", on)
		}
		if !sv.MouseWheel(wheelDown()) {
			t.Fatalf("preference %v: the scroll view did not take the notch the combo passed on", on)
		}
		if sv.OffsetY <= before {
			t.Errorf("preference %v: the view did not scroll (%v then %v)", on, before, sv.OffsetY)
		}
	}
}

// Three more refusals, each of them somebody scrolling rather than
// choosing. See ComboBox.MouseWheel.
func TestComboWheelRefusals(t *testing.T) {
	comboWheelOff(t)
	style.SetComboWheel(true)
	h := &host{}

	// A touchpad's two-finger scroll has no detents: one flick would
	// run through the whole list.
	cb := newTestCombo(h)
	if cb.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 12), Precise: true}) {
		t.Error("a precise (touchpad) scroll stepped the selection")
	}
	if cb.Selected != 1 {
		t.Errorf("a touchpad scroll moved the selection to %d", cb.Selected)
	}

	// Text somebody typed is not a choice they are one step away from.
	ed := newTestCombo(h)
	ed.SetEditable(true)
	if ed.MouseWheel(wheelDown()) {
		t.Error("an editable combo box stepped on the wheel")
	}

	// A disabled box changes nothing at all.
	dis := newTestCombo(h)
	dis.SetEnabled(false)
	if dis.MouseWheel(wheelDown()) {
		t.Error("a disabled combo box stepped on the wheel")
	}
}

// A box may overrule the preference either way, which is how a form
// that is scrolled past more often than it is answered opts out and a
// page whose combo box *is* its control opts in.
func TestComboWheelSelectOverridesThePreference(t *testing.T) {
	comboWheelOff(t)
	h := &host{}

	on := newTestCombo(h)
	on.WheelSelect = WheelSelectOn
	if !on.MouseWheel(wheelDown()) || on.Selected != 2 {
		t.Errorf("WheelSelectOn did not step with the preference off (selected %d)", on.Selected)
	}

	style.SetComboWheel(true)
	off := newTestCombo(h)
	off.WheelSelect = WheelSelectOff
	if off.MouseWheel(wheelDown()) {
		t.Error("WheelSelectOff stepped because the preference was on")
	}
	if off.Selected != 1 {
		t.Errorf("WheelSelectOff moved the selection to %d", off.Selected)
	}

	// And the zero value is "ask the preference", so a box nobody
	// touched follows the user.
	plain := newTestCombo(h)
	if plain.WheelSelect != WheelSelectPref {
		t.Errorf("a new combo box is %v, want WheelSelectPref", plain.WheelSelect)
	}
	if !plain.MouseWheel(wheelDown()) {
		t.Error("a box with no opinion ignored the preference that is on")
	}
}

// The open list is the other control, and it was never the contentious
// one: the popup scrolls like the list it is (PopupMenu.MouseWheel),
// and the face behind it does not step underneath it. The window hands
// a notch over the face of an open box to the box rather than to the
// popup (app.Window.hit), so this guard is the only thing standing
// between the two.
func TestComboWheelLeavesTheOpenListAlone(t *testing.T) {
	comboWheelOff(t)
	style.SetComboWheel(true)
	h := &host{}
	cb := newTestCombo(h)
	cb.open = true
	if cb.MouseWheel(wheelDown()) {
		t.Error("an open combo box stepped its selection on the wheel")
	}
	if cb.Selected != 1 {
		t.Errorf("an open combo box moved to %d on the wheel", cb.Selected)
	}
	cb.open = false
	if !cb.MouseWheel(wheelDown()) {
		t.Error("the same box, closed, ignored the wheel")
	}
}

// The preference is a process flag like style.Animations, and the app
// package sets it from look.json; this is the widget end of that wire.
func TestComboWheelPreferenceRoundTrips(t *testing.T) {
	comboWheelOff(t)
	if style.ComboWheel() {
		t.Fatal("the preference is on by default")
	}
	style.SetComboWheel(true)
	if !style.ComboWheel() {
		t.Error("SetComboWheel(true) did not take")
	}
}
