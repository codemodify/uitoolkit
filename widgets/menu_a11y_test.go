package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A menu row that advertises a default action can perform one.
//
// The rows said they could be activated and the menu implemented no way
// to do it: a screen reader or an automation tool asking got false and
// nothing happened, while a click or Return on the same row ran the item.
// A structural check of the accessibility tree cannot see this — the node
// is there, correctly described, and only the execution path is missing.
func TestAMenuRowCanPerformTheActionItAdvertises(t *testing.T) {
	ran := 0
	p := NewPopupMenu(
		&MenuItem{Text: "Run", OnClick: func() { ran++ }},
		&MenuItem{Separator: true},
		&MenuItem{Text: "Nope", Disabled: true, OnClick: func() { ran += 100 }},
	)
	p.SetLook(style.DarkLook())
	p.SetHost(&host{})

	actor, ok := any(p).(widget.AccessibleActor)
	if !ok {
		t.Fatal("a popup menu is not an AccessibleActor, so nothing can run its rows")
	}
	items := p.AccessibleItems()
	if len(items) != 3 {
		t.Fatalf("got %d accessible rows, want 3", len(items))
	}

	// The enabled row advertises the action and performs it.
	if !items[0].Actions.Has(a11y.ActionDefault) {
		t.Error("the enabled row does not advertise a default action")
	}
	if !actor.AccessibleAction(0, a11y.ActionDefault) {
		t.Error("the enabled row refused the action it advertises")
	}
	if ran != 1 {
		t.Errorf("the item ran %d times, want 1", ran)
	}

	// A separator and a disabled row neither advertise it nor run.
	if items[2].Actions.Has(a11y.ActionDefault) {
		t.Error("a disabled row advertises an action it will refuse")
	}
	if actor.AccessibleAction(2, a11y.ActionDefault) {
		t.Error("a disabled row performed an action")
	}
	if actor.AccessibleAction(1, a11y.ActionDefault) {
		t.Error("a separator performed an action")
	}
	if ran != 1 {
		t.Errorf("something else ran: %d", ran)
	}
}
