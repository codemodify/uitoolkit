package a11ytest

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
)

func tree() *a11y.Node {
	box := paintengine2d.XYWH(0, 0, 100, 40)
	return &a11y.Node{ID: 1, Role: a11y.RoleWindow, Name: "W", Bounds: box, Children: []*a11y.Node{
		{ID: 2, Role: a11y.RoleList, Name: "Themes", Bounds: box, Children: []*a11y.Node{
			{ID: 1 << 25, Role: a11y.RoleListItem, Name: "Dark", Bounds: box},
			{ID: 1<<25 | 1, Role: a11y.RoleListItem, Name: "Light", Bounds: box},
		}},
		{ID: 3, Role: a11y.RoleButton, Name: "Apply", Bounds: box, State: a11y.StateFocusable},
	}}
}

// A clean tree passes, and the tree comes back for the caller to go on
// asking questions of.
func TestAuditPassesACleanTree(t *testing.T) {
	root := tree()
	if got := Audit(t, "settings", root); got != root {
		t.Error("Audit did not hand the tree back")
	}
}

func TestCountAndFind(t *testing.T) {
	root := tree()
	if got := Count(root, a11y.RoleListItem); got != 2 {
		t.Errorf("Count(list item) = %d, want 2", got)
	}
	if got := Count(root, a11y.RoleTable); got != 0 {
		t.Errorf("Count(table) = %d, want 0", got)
	}
	if got := Find(root, a11y.RoleListItem, "Light"); got == nil || got.ID != 1<<25|1 {
		t.Errorf("Find(list item, Light) = %v", got)
	}
	if got := Find(root, a11y.RoleButton, "Cancel"); got != nil {
		t.Errorf("Find found a button that is not there: %v", got)
	}
	if Count(nil, a11y.RoleButton) != 0 || Find(nil, a11y.RoleButton, "x") != nil {
		t.Error("a nil tree must be empty rather than a panic")
	}
}
