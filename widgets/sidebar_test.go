package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// A sidebar view tells the look so, on its frame and every row; a tool in
// a tool bar is auto-raise.
func TestSidebarAndAutoRaiseStates(t *testing.T) {
	l := NewListView(3, func(i int) string { return "row" }, nil)
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	if l.rowState(0).Sidebar() || l.viewState().Sidebar() {
		t.Fatal("a plain list is not a sidebar")
	}
	l.Sidebar = true
	if !l.rowState(1).Sidebar() || !l.viewState().Sidebar() {
		t.Fatal("sidebar rows and frame carry StateSidebar")
	}
	tr := NewTreeView(&TreeNode{Label: "Inbox"})
	tr.SetLook(style.DarkLook())
	tr.SetHost(&host{})
	tr.Sidebar = true
	if !tr.rowState(tr.flatten()[0].node).Sidebar() {
		t.Fatal("sidebar tree rows carry StateSidebar")
	}
	// The new bits sit above the tree chain bits and leave them intact.
	st := style.TreeChain(0xffff) | style.StateSidebar | style.StateAutoRaise
	if !st.HasNextSibling(15) || !st.Sidebar() || !st.AutoRaise() {
		t.Fatalf("state bits collide: %#x", uint64(st))
	}
}
