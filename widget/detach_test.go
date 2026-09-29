package widget

import "testing"

// Add hands the host down a subtree, so a component that has ever been
// added holds a pointer to the window it was added to. Clearing only the
// parent left that pointer live after the component came off the tree:
// whatever referenced the component kept the window reachable, and the
// component itself could not tell it had left the screen.
func TestRemoveClearsTheHostOfTheWholeSubtree(t *testing.T) {
	h := &stubHost{}
	root, mid, leaf := &Base{}, &Base{}, &Base{}
	root.Init(root)
	mid.Init(mid)
	leaf.Init(leaf)
	mid.Add(leaf)
	root.Add(mid)
	root.SetHost(h)

	if mid.Host() != h || leaf.Host() != h {
		t.Fatal("Add did not hand the host down")
	}
	root.Remove(mid)
	if mid.Parent() != nil {
		t.Error("parent survived Remove")
	}
	if mid.Host() != nil {
		t.Error("the removed child still has its host")
	}
	if leaf.Host() != nil {
		t.Error("the removed child's own child still has the host")
	}
}

func TestClearChildrenClearsTheHost(t *testing.T) {
	h := &stubHost{}
	root, a, b := &Base{}, &Base{}, &Base{}
	root.Init(root)
	a.Init(a)
	b.Init(b)
	root.Add(a)
	root.Add(b)
	root.SetHost(h)

	root.ClearChildren()
	for i, ch := range []*Base{a, b} {
		if ch.Host() != nil {
			t.Errorf("child %d still has its host", i)
		}
		if ch.Parent() != nil {
			t.Errorf("child %d still has its parent", i)
		}
	}
}

// Moving a child from one parent to another goes through Remove, so the
// host has to come back on the other side or a re-parent would strand it.
func TestReparentKeepsTheHost(t *testing.T) {
	h := &stubHost{}
	root, from, to, leaf := &Base{}, &Base{}, &Base{}, &Base{}
	for _, b := range []*Base{root, from, to, leaf} {
		b.Init(b)
	}
	from.Add(leaf)
	root.Add(from)
	root.Add(to)
	root.SetHost(h)

	to.Add(leaf)
	if leaf.Parent() != Component(to) {
		t.Fatal("the child did not move")
	}
	if leaf.Host() != h {
		t.Error("a re-parented child lost its host")
	}
}

// Removing something that is not a child must not detach it from the
// parent it does have.
func TestRemoveOfAStrangerDoesNothing(t *testing.T) {
	h := &stubHost{}
	root, mine, other := &Base{}, &Base{}, &Base{}
	for _, b := range []*Base{root, mine, other} {
		b.Init(b)
	}
	root.Add(mine)
	root.SetHost(h)
	other.Init(other)
	root.Add(other)
	stranger := &Base{}
	stranger.Init(stranger)
	root.Remove(stranger)

	if mine.Host() != h || other.Host() != h {
		t.Error("removing a stranger detached the real children")
	}
}
