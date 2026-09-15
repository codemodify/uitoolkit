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

// Check audits a tree the way an accessibility linter does: controls
// without a name (a screen reader would say only "button"), duplicate IDs,
// more than one focused node, ranges that run backwards, visible controls
// without a box. Apps can call it in their tests.
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
		if n.Role.Interactive() && n.Name == "" && !offscreen {
			out = append(out, Problem{n, "has no name"})
		}
		if n.HasRange && n.Min > n.Max {
			out = append(out, Problem{n, fmt.Sprintf("range runs backwards (%g > %g)", n.Min, n.Max)})
		}
		if n.Role.Interactive() && !offscreen && n.Bounds.Empty() {
			out = append(out, Problem{n, "has no box"})
		}
		return true
	})
	return out
}
