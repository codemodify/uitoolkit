//go:build skipfixture

// Package skipfixture2 exists only to share a test name with
// [skipfixture], so that the reporter can be checked for keeping two
// packages' logs apart.
//
// Two packages may each have a TestOuter, and keying what was said by
// test name alone handed the second one the first one's reason: this
// package's child says nothing at all, and was reported with the
// other package's words (a video player, the release after the
// reporter started finding reasons at all).
package skipfixture2

import "testing"

func TestOuter(t *testing.T) {
	// The same name as a child of [skipfixture] that skips *with* a
	// reason. This one logs nothing, so the only line it could be
	// reported with is the other package's.
	t.Run("shared", func(t *testing.T) { t.SkipNow() })
}
