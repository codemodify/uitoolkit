package a11y

import "fmt"

// Problem is something assistive technology would trip over.
type Problem struct {
	Node *Node
	What string
}

func (p Problem) String() string {
	return fmt.Sprintf("%s %q (id %d): %s", p.Node.Role, p.Node.Name, p.Node.ID, p.What)
}

// Check audits a tree the way an accessibility linter does: controls and
// item views without a name (a screen reader would say only "button", or
// only "list"), duplicate IDs, more than one focused node, ranges that run
// backwards, visible controls without a box, controls the keyboard cannot
// reach. Apps can call it in their tests.
func Check(root *Node) []Problem {
	var out []Problem
	seen := map[uint64]*Node{}
	focused := 0
	root.Walk(func(n *Node) bool {
		if other, dup := seen[n.ID]; dup && n.ID != 0 {
			out = append(out, Problem{n, fmt.Sprintf("shares its id with %s %q", other.Role, other.Name)})
		}
		seen[n.ID] = n
		if n.State.Has(StateFocused) && n.Role != RoleWindow {
			focused++
			if focused == 2 {
				out = append(out, Problem{n, "a second focused node"})
			}
		}
		offscreen := n.State.Has(StateOffscreen)
		if (n.Role.Interactive() || namedView(n.Role)) && n.Name == "" && !offscreen {
			out = append(out, Problem{n, "has no name"})
		}
		if n.HasRange && n.Min > n.Max {
			out = append(out, Problem{n, fmt.Sprintf("range runs backwards (%g > %g)", n.Min, n.Max)})
		}
		if n.Role.Interactive() && !offscreen && n.Bounds.Empty() {
			out = append(out, Problem{n, "has no box"})
		}
		// A control the keyboard cannot reach. Items (list rows, tabs,
		// tools, menu items) are reached through their view.
		if n.Role.Interactive() && !isItem(n.ID) && !n.State.Has(StateFocusable) && !n.State.Has(StateDisabled) {
			out = append(out, Problem{n, "cannot take the keyboard focus"})
		}
		return true
	})
	return out
}

// isItem reports whether id names an item of a view rather than a
// component (the widget package numbers items above bit 24).
func isItem(id uint64) bool { return id>>24 != 0 && id&(1<<24-1) != 0 && id>>63 == 0 }

// namedView reports whether the role is a view of items rather than a
// control the user operates. Role.Interactive says no to these, because
// what the user operates is the row inside them, and that is right for
// the focus and box checks. A name is another matter: a screen reader
// entering a list says its name before it starts reading rows, and a list
// with none is announced as "list", which says nothing about what is in
// it. Every such view an application shows is something the user has to
// be able to tell from the next one, so it must be called something.
func namedView(r Role) bool {
	switch r {
	case RoleList, RoleTree, RoleTable:
		return true
	}
	return false
}
