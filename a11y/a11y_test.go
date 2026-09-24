package a11y

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestCheckFindsProblems(t *testing.T) {
	box := paintengine2d.XYWH(0, 0, 80, 24)
	root := &Node{ID: 1, Role: RoleWindow, Children: []*Node{
		{ID: 2, Role: RoleButton, Name: "Save", Bounds: box, State: StateFocused | StateFocusable},
		{ID: 3, Role: RoleButton, Bounds: box, State: StateFocusable},                                                 // no name
		{ID: 2, Role: RoleLabel, Name: "dup"},                                                                         // duplicate id
		{ID: 4, Role: RoleSlider, Name: "Volume", Bounds: box, HasRange: true, Min: 5, Max: 1, State: StateFocusable}, // backwards
		{ID: 5, Role: RoleCheckBox, Name: "Wrap", State: StateFocused},                                                // second focus, no box
		{ID: 6, Role: RoleListItem, State: StateOffscreen},                                                            // fine: offscreen, not interactive
	}}
	var got []string
	for _, p := range Check(root) {
		got = append(got, p.String())
	}
	all := strings.Join(got, "\n")
	for _, want := range []string{"has no name", "shares its id", "range runs backwards", "a second focused node", "has no box", "cannot take the keyboard focus"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing %q in:\n%s", want, all)
		}
	}
	if len(got) != 6 {
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

// A list, a tree or a table with no name is announced as "list": the row
// inside it is what the user operates, so Role.Interactive says no to it,
// but it still has to be called something.
func TestCheckWantsItemViewsNamed(t *testing.T) {
	box := paintengine2d.XYWH(0, 0, 100, 40)
	for _, role := range []Role{RoleList, RoleTree, RoleTable} {
		root := &Node{ID: 1, Role: RoleWindow, Name: "W", Bounds: box, Children: []*Node{
			{ID: 2, Role: role, Bounds: box},
		}}
		problems := Check(root)
		if len(problems) != 1 || !strings.Contains(problems[0].String(), "has no name") {
			t.Errorf("an unnamed %s gave %v", role, problems)
		}
		root.Children[0].Name = "Inbox"
		if got := Check(root); len(got) != 0 {
			t.Errorf("a named %s gave %v", role, got)
		}
	}
	// A row inside one needs no name of its own beyond its text, and an
	// offscreen view is not on screen to be announced.
	off := &Node{ID: 1, Role: RoleWindow, Name: "W", Bounds: box, Children: []*Node{
		{ID: 2, Role: RoleList, Bounds: box, State: StateOffscreen},
	}}
	if got := Check(off); len(got) != 0 {
		t.Errorf("an offscreen list gave %v", got)
	}
}
