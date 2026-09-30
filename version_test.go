package uitoolkit

import (
	"os"
	"regexp"
	"testing"
)

// Version must match the newest release in release-notes.md. It was
// left at 0.22.1 through the whole 0.22.2 release, so every application
// that shows the toolkit's version showed the wrong one — and nothing
// anywhere would have said so.
func TestVersionMatchesTheNewestReleaseNote(t *testing.T) {
	b, err := os.ReadFile("release-notes.md")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^## ([0-9]+\.[0-9]+\.[0-9]+)$`).FindSubmatch(b)
	if m == nil {
		t.Fatal("no version heading in release-notes.md")
	}
	if got, want := Version, string(m[1]); got != want {
		t.Errorf("Version is %q, but the newest release note is %q — bump it with the release", got, want)
	}
}
