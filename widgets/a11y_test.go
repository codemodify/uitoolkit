package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func accTree(c widget.Component) *a11y.Node {
	root := &a11y.Node{Role: a11y.RoleWindow}
	widget.AccessibleTree(root, c)
	return root
}

// Tree items nest as their nodes do and expand and collapse through the
// accessibility actions; an icon-only tool is named for its icon.
func TestAccessibleItemsAndActions(t *testing.T) {
	archives := NewTreeNode("Archives", NewTreeNode("2025"), NewTreeNode("2026"))
	archives.Expanded = false
	tree := NewTreeView(NewTreeNode("Inbox"), archives)
	tree.SetLook(style.DarkLook())
	tree.SetHost(&host{})
	tree.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	root := accTree(tree)
	if len(root.Children) != 1 || root.Children[0].Role != a11y.RoleTree {
		t.Fatalf("tree node %+v", root.Children)
	}
	items := root.Children[0].Children
	if len(items) != 2 || items[1].Name != "Archives" || !items[1].State.Has(a11y.StateExpandable) || items[1].State.Has(a11y.StateExpanded) {
		t.Fatalf("items %+v", items)
	}
	_, idx := widget.SplitItemID(items[1].ID)
	if !tree.AccessibleAction(idx, a11y.ActionExpand) || !archives.Expanded {
		t.Fatal("expand through the tree")
	}
	items = accTree(tree).Children[0].Children
	if len(items[1].Children) != 2 || items[1].Children[1].Name != "2026" || items[1].Children[1].Level != 2 {
		t.Fatalf("expanded children %+v", items[1].Children)
	}
	if !tree.AccessibleAction(idx, a11y.ActionCollapse) || archives.Expanded {
		t.Fatal("collapse through the tree")
	}

	bar := NewToolBar(ToolIconBtn(style.IconSave, "", nil), ToolDivider(), ToolIconBtn(style.IconMail, "Send", nil))
	bar.SetLook(style.DarkLook())
	bar.SetHost(&host{})
	bar.Arrange(paintengine2d.XYWH(0, 0, 300, 40))
	tools := accTree(bar).Children[0].Children
	if len(tools) != 3 || tools[0].Name != "Save" || tools[1].Role != a11y.RoleSeparator || tools[2].Name != "Send" {
		t.Fatalf("tools %+v", tools)
	}
	if p := a11y.Check(accTree(bar)); len(p) != 0 {
		t.Fatalf("toolbar problems %v", p)
	}
}

// An explicit accessible name wins over the widget's own text.
func TestAccessibleNameWins(t *testing.T) {
	b := NewButton("…", nil)
	b.SetLook(style.DarkLook())
	b.SetHost(&host{})
	b.SetAccessibleName("More options")
	if n := accTree(b).Children[0]; n.Name != "More options" || n.Role != a11y.RoleButton {
		t.Fatalf("button %+v", n)
	}
}
