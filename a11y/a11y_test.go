package a11y

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestCheckFindsProblems(t *testing.T) {
	box := paintengine2d.XYWH(0, 0, 80, 24)
	root := &Node{ID: 1, Role: RoleWindow, Children: []*Node{
		{ID: 2, Role: RoleButton, Name: "Save", Bounds: box, State: StateFocused},
		{ID: 3, Role: RoleButton, Bounds: box},                                                 // no name
		{ID: 2, Role: RoleLabel, Name: "dup"},                                                  // duplicate id
		{ID: 4, Role: RoleSlider, Name: "Volume", Bounds: box, HasRange: true, Min: 5, Max: 1}, // backwards
		{ID: 5, Role: RoleCheckBox, Name: "Wrap", State: StateFocused},                         // second focus, no box
		{ID: 6, Role: RoleListItem, State: StateOffscreen},                                     // fine: offscreen, not interactive
	}}
	var got []string
	for _, p := range Check(root) {
		got = append(got, p.String())
	}
	all := strings.Join(got, "\n")
	for _, want := range []string{"has no name", "shares its id", "range runs backwards", "a second focused node", "has no box"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing %q in:\n%s", want, all)
		}
	}
	if len(got) != 5 {
		t.Errorf("%d problems:\n%s", len(got), all)
	}
}

func TestRolesAndActions(t *testing.T) {
	if RoleButton.String() != "button" || RoleLink.String() != "link" || Role(250).String() != "unknown" {
		t.Fatal("role names")
	}
	var a Actions
	a = a.With(ActionDefault).With(ActionExpand)
	if !a.Has(ActionExpand) || a.Has(ActionCollapse) {
		t.Fatal("action set")
	}
	n := &Node{ID: 1, Children: []*Node{{ID: 2, Children: []*Node{{ID: 3}}}}}
	if n.Find(3) == nil || n.Find(9) != nil {
		t.Fatal("find")
	}
}
