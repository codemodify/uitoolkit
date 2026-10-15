//go:build skipfixture

// Package skipfixture is a test that skips on purpose, so that
// tools/skips.sh can be checked against a known answer.
//
// Behind a tag of its own so that an ordinary run never sees it: a
// fixture that skipped in every suite would be two more lines of
// noise in the very report it exists to verify. The root package's
// TestTheSkipReporterReportsSkipsAndFailures builds it with
// -tags skipfixture.
package skipfixture

import "testing"

func TestOuter(t *testing.T) {
	// A child that says something before it gives its reason. The
	// reporter kept the *first* line a test logged, so this one came
	// out as "setup talk, not a reason" (a music player, item 55).
	t.Run("logged", func(t *testing.T) {
		t.Log("setup talk, not a reason")
		t.Skip("the capability this needs is absent")
	})
	// And a child that skips without a word. Both are children, which
	// is the other half of it: go test indents a subtest's verdict, so
	// a grep for a line starting "--- SKIP" finds neither.
	t.Run("plain", func(t *testing.T) { t.Skip("nothing to say first") })
}

func TestThatRuns(t *testing.T) {}
