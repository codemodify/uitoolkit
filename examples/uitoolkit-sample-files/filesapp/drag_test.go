package filesapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// A drop from another application is listed, not copied: the sample adds
// rows and touches no filesystem. So it must refuse a Move.
//
// Returning true tells the source the target took the data, and a source
// that negotiated Move deletes its original on the strength of it. The
// sample advertises Move because its *own* rows move between folders;
// answering one from outside would destroy the user's files to produce a
// row in a demo.
func TestExternalMoveDropIsRefused(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "keepme.txt")
	if err := os.WriteFile(file, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	places := []filePlace{{Label: "A"}, {Label: "B"}}

	move := widget.DropEvent{Paths: []string{file}, Action: platform.DragMove}
	if filesDropInto(places, 1, move) {
		t.Fatal("the sample acknowledged an external Move it cannot perform; the source would delete the original")
	}
	if n := len(places[1].Rows); n != 0 {
		t.Fatalf("a refused drop still added %d row(s)", n)
	}

	// A copy is what it can actually do, and still does.
	cp := widget.DropEvent{Paths: []string{file}, Action: platform.DragCopy}
	if !filesDropInto(places, 1, cp) {
		t.Fatal("an external Copy was refused")
	}
	if n := len(places[1].Rows); n != 1 {
		t.Fatalf("a copy added %d row(s), want 1", n)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("the original is gone: %v", err)
	}
}
