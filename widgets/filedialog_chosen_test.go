package widgets

import (
	"os"
	"path/filepath"
	"testing"
)

// A Save dialog must return the name the user typed, not the row they
// happened to select first.
//
// Selecting a row writes that row's full path into the path field, so
// for a plain selection the two agree. They disagree exactly when the
// user typed — select old.txt, type new.txt over it, press Save — and
// the dialog used to read the row, hand back old.txt, and let the
// caller overwrite the file the user was careful not to name.
func TestSaveDialogReturnsTheTypedNameNotTheSelectedRow(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"old.txt", "other.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var got string
	fd := NewFileDialog(FileDialogOptions{
		Mode: FileSave, Path: dir,
		OnPick: func(p string) { got = p },
	})

	// Select old.txt, the way a user browsing to the right folder does.
	sel := -1
	for i, e := range fd.Entries() {
		if e.Name == "old.txt" {
			sel = i
		}
	}
	if sel < 0 {
		t.Fatalf("old.txt is not in the listing: %+v", fd.Entries())
	}
	fd.table.Selected = sel
	fd.onSelect(sel)
	if want := filepath.Join(dir, "old.txt"); fd.path.Text != want {
		t.Fatalf("selecting a row put %q in the path field, want %q", fd.path.Text, want)
	}

	// Then type a new name over it and accept.
	want := filepath.Join(dir, "new.txt")
	fd.path.SetText(want)
	fd.finish(true)

	if got != want {
		t.Fatalf("Save returned %q, want %q — the typed name has to win", got, want)
	}
}

// With nothing typed, the selected row is still what the dialog returns.
func TestOpenDialogStillReturnsTheSelectedRow(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "only.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got string
	fd := NewFileDialog(FileDialogOptions{
		Mode: FileOpen, Path: dir,
		OnPick: func(p string) { got = p },
	})
	sel := -1
	for i, e := range fd.Entries() {
		if e.Name == "only.txt" {
			sel = i
		}
	}
	if sel < 0 {
		t.Skip("no listing")
	}
	fd.table.Selected = sel
	fd.onSelect(sel)
	fd.finish(true)
	if want := filepath.Join(dir, "only.txt"); got != want {
		t.Fatalf("Open returned %q, want %q", got, want)
	}
}
