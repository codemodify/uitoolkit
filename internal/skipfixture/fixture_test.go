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
	// A child that talks on both sides of its reason: setup before,
	// and a cleanup that runs after the skip. The reporter guessed the
	// first line a test logged and this came out as "setup talk"; it
	// then guessed the last and this came out as "teardown talk". Both
	// guesses were reported by applications, a morning apart, and
	// there is nothing in the output to guess from — t.Skip's message
	// is an ordinary log line. So all three are reported.
	t.Run("logged", func(t *testing.T) {
		t.Cleanup(func() { t.Log("teardown talk, after the skip") })
		t.Log("setup talk, not a reason")
		t.Skip("the capability this needs is absent")
	})
	// A child that skips with one word, which is the ordinary case and
	// still reads as one line beside its name.
	t.Run("plain", func(t *testing.T) { t.Skip("nothing to say first") })
	// And one that gives no reason at all: t.SkipNow logs nothing, so
	// there is no line to find and the report has to say so rather
	// than reach for somebody else's.
	t.Run("silent", func(t *testing.T) { t.SkipNow() })
	// This one shares its name with a child of [skipfixture2], which
	// says nothing. Two packages may each have a test of the same
	// name, and what this one says must not be handed to that one.
	t.Run("shared", func(t *testing.T) { t.Skip("only this package is missing anything") })
}

func TestThatRuns(t *testing.T) {}

// attempts counts this package's runs of [TestRepeated], so that its
// second attempt can say nothing where its first said something.
var attempts int

// A test that speaks once and then goes quiet, which is what -count=2
// makes it possible to see: the same test runs twice inside one
// package block, and the second attempt was being reported with
// whatever the first one said (a video player, the release after
// packages were separated).
//
// Contrived on purpose. An ordinary test says the same thing on every
// attempt, which is exactly why nobody would notice the first
// attempt being quoted for the second.
func TestRepeated(t *testing.T) {
	attempts++
	if attempts == 1 {
		t.Skip("only the first attempt has anything to say")
	}
	t.SkipNow()
}
