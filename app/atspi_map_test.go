package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/a11y"
)

func has(w [2]uint32, bit uint) bool { return w[bit/32]&(1<<(bit%32)) != 0 }

func TestATSPIStates(t *testing.T) {
	n := &a11y.Node{Role: a11y.RoleTreeItem, State: a11y.StateSelectable | a11y.StateExpandable}
	w := atspiStates(n)
	for _, bit := range []uint{atspiStateEnabled, atspiStateSensitive, atspiStateVisible, atspiStateShowing,
		atspiStateSelectable, atspiStateExpandable, atspiStateCollapsed} {
		if !has(w, bit) {
			t.Errorf("tree item: missing state %d", bit)
		}
	}
	n.State |= a11y.StateExpanded | a11y.StateDisabled | a11y.StateOffscreen
	w = atspiStates(n)
	if has(w, atspiStateCollapsed) || has(w, atspiStateEnabled) || has(w, atspiStateShowing) || !has(w, atspiStateExpanded) {
		t.Errorf("expanded, disabled, offscreen: %v", w)
	}
	// Bits past 31 land in the second word.
	w = atspiStates(&a11y.Node{Role: a11y.RoleButton, State: a11y.StateDefault | a11y.StateHasPopup})
	if !has(w, atspiStateIsDefault) || !has(w, atspiStateHasPopup) || w[1] == 0 {
		t.Errorf("default button with a popup: %v", w)
	}
	// A focused window is active in AT-SPI.
	w = atspiStates(&a11y.Node{Role: a11y.RoleWindow, State: a11y.StateFocused})
	if has(w, atspiStateFocused) || !has(w, atspiStateActive) {
		t.Errorf("window: %v", w)
	}
	if !has(atspiStates(&a11y.Node{Role: a11y.RoleTextField}), atspiStateSingleLine) {
		t.Error("an entry is single-line")
	}
}

func TestATSPIRolesCovered(t *testing.T) {
	for r := a11y.RoleUnknown; r <= a11y.RoleTitleBar; r++ {
		v, ok := atspiRoles[r]
		if !ok {
			t.Errorf("no AT-SPI role for %s", r)
			continue
		}
		if atspiRoleNames[v] == "" {
			t.Errorf("no AT-SPI name for %s (%d)", r, v)
		}
	}
	if atspiRole(a11y.RoleTitleBar) != 104 || a11y.RoleTitleBar.String() != "title bar" {
		t.Error("title bar is ATSPI_ROLE_TITLE_BAR (104)")
	}
	if atspiActionName(&a11y.Node{Role: a11y.RoleCheckBox}, a11y.ActionDefault) != "toggle" ||
		atspiActionName(&a11y.Node{Role: a11y.RoleButton}, a11y.ActionDefault) != "click" {
		t.Error("action names")
	}
}
