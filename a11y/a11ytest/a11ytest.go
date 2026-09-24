// Package a11ytest audits an accessibility tree from a test.
//
// [a11y.Check] is the audit itself and is what this package calls; what
// it adds is the part every application repeated around it — failing the
// test, naming the window that failed, and printing the problems one to a
// line so the failure reads as a list rather than a struct dump. It lives
// beside a11y rather than in it because a library must not import
// testing, in the way httptest lives beside net/http.
//
// A test of a window usually reads:
//
//	a.PumpOnce()
//	tree := a11ytest.Audit(t, "files", win.AccessibleTree())
//	if a11ytest.Count(tree, a11y.RoleButton) == 0 {
//		t.Error("no buttons in the tree")
//	}
//
// Audit reports rather than fatals, so one run lists everything a screen
// reader would stumble over instead of only the first thing.
package a11ytest

import (
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/a11y"
)

// Audit checks a window's accessibility tree and fails the test with what
// a screen reader would stumble over. name says which window, for a test
// that audits several. It returns the tree, so the caller can go on to
// ask what is in it.
//
// An empty tree is fatal: a window with no children means the test did
// not build or pump what it thought it did, and every later check would
// pass vacuously.
func Audit(t testing.TB, name string, tree *a11y.Node) *a11y.Node {
	t.Helper()
	if tree == nil || len(tree.Children) == 0 {
		t.Fatalf("%s: empty accessibility tree", name)
	}
	problems := a11y.Check(tree)
	if len(problems) == 0 {
		return tree
	}
	lines := make([]string, len(problems))
	for i, p := range problems {
		lines[i] = p.String()
	}
	t.Errorf("%s: %d accessibility problems:\n%s", name, len(lines), strings.Join(lines, "\n"))
	return tree
}

// Count is how many nodes in the tree carry the role: a test's way of
// saying "the table really is in there" after an audit found nothing
// wrong with what is.
func Count(tree *a11y.Node, role a11y.Role) int {
	n := 0
	if tree == nil {
		return 0
	}
	tree.Walk(func(x *a11y.Node) bool {
		if x.Role == role {
			n++
		}
		return true
	})
	return n
}

// Find is the first node with this role and name, or nil. Tests use it to
// reach a control the way assistive technology would — by what it is
// called — rather than by walking the widget tree.
func Find(tree *a11y.Node, role a11y.Role, name string) *a11y.Node {
	var found *a11y.Node
	if tree == nil {
		return nil
	}
	tree.Walk(func(n *a11y.Node) bool {
		if found == nil && n.Role == role && n.Name == name {
			found = n
		}
		return found == nil
	})
	return found
}
