package widgets

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// A player adding a dozen tracks to a playlist opens one dialog, not twelve.
//
// platform.FileChooserOptions has had Multiple all along and its callback has
// always taken a []string; the widget's API dropped it — OnPick returns one
// path and the native path forwarded paths[0] — so an application opened or
// added one file per dialog and got the rest from the command line, a
// file-manager drop or a folder scan.
func multiDialog(t *testing.T, names ...string) (*FileDialog, string, *[]string) {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	fd := NewFileDialog(FileDialogOptions{
		Mode: FileOpen, Path: dir, Multiple: true,
		OnPickMany: func(p []string) { got = p },
	})
	return fd, dir, &got
}

func TestAMultipleFileDialogReturnsEveryChosenFile(t *testing.T) {
	fd, dir, got := multiDialog(t, "one.flac", "two.flac", "three.flac")
	rows := map[string]int{}
	for i, e := range fd.Entries() {
		rows[e.Name] = i
	}
	fd.table.Mode = SelectExtended
	fd.table.SetSelectedRows([]int{rows["one.flac"], rows["three.flac"]})
	fd.finish(true)

	want := []string{filepath.Join(dir, "one.flac"), filepath.Join(dir, "three.flac")}
	sort.Strings(*got)
	sort.Strings(want)
	if len(*got) != 2 || (*got)[0] != want[0] || (*got)[1] != want[1] {
		t.Errorf("chose %v, want %v", *got, want)
	}
	// Paths() says the same, for a caller that keeps the dialog.
	if len(fd.Paths()) != 2 {
		t.Errorf("Paths() = %v, want both", fd.Paths())
	}
}

// OnPick still fires, with the first of them, so code written before
// OnPickMany existed keeps working — and a dialog opened with Multiple by
// mistake is not silent.
func TestAMultipleFileDialogStillCallsOnPick(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "only.flac"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var one string
	var many []string
	fd := NewFileDialog(FileDialogOptions{
		Mode: FileOpen, Path: dir, Multiple: true,
		OnPick:     func(p string) { one = p },
		OnPickMany: func(p []string) { many = p },
	})
	fd.table.Selected = 0
	fd.onSelect(0)
	fd.finish(true)
	if one == "" {
		t.Error("OnPick was not called")
	}
	if len(many) != 1 || many[0] != one {
		t.Errorf("OnPickMany got %v and OnPick %q", many, one)
	}
}

// Saving and choosing a folder take one answer, whatever Multiple says: there
// is one file to save to, and one folder.
func TestMultipleIsIgnoredForSaveAndFolder(t *testing.T) {
	for _, mode := range []FileDialogMode{FileSave, FileOpenFolder} {
		dir := t.TempDir()
		for _, n := range []string{"a.txt", "b.txt"} {
			if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		var many []string
		fd := NewFileDialog(FileDialogOptions{
			Mode: mode, Path: dir, Multiple: true,
			OnPickMany: func(p []string) { many = p },
		})
		if fd.multiple() {
			t.Errorf("mode %v takes several files", mode)
		}
		if fd.table.Mode == SelectExtended {
			t.Errorf("mode %v offered a multiple selection", mode)
		}
		fd.finish(true)
		if len(many) != 1 {
			t.Errorf("mode %v answered with %v, want one path", mode, many)
		}
	}
}

// A dialog without Multiple is the dialog it always was.
func TestWithoutMultipleTheDialogAnswersWithOnePath(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var many []string
	fd := NewFileDialog(FileDialogOptions{
		Mode: FileOpen, Path: dir,
		OnPickMany: func(p []string) { many = p },
	})
	if fd.table.Mode == SelectExtended {
		t.Error("a single-file dialog offered a multiple selection")
	}
	fd.table.Selected = 0
	fd.onSelect(0)
	fd.finish(true)
	if len(many) != 1 {
		t.Errorf("answered with %v, want one path", many)
	}
}

// The native chooser is told, and every path it answers with is passed on.
// Forwarding paths[0] and dropping the rest is what the widget used to do.
func TestTheNativeChooserIsAskedForSeveralFiles(t *testing.T) {
	defer func(old func(platform.FileChooserOptions, func([]string)) bool) { openNative = old }(openNative)
	var asked platform.FileChooserOptions
	openNative = func(o platform.FileChooserOptions, done func([]string)) bool {
		asked = o
		done([]string{"/music/one.flac", "/music/two.flac", "/music/three.flac"})
		return true
	}
	from := NewLabel("x")
	from.SetLook(style.DarkLook())
	from.SetHost(&nowHost{})

	var many []string
	var one string
	ShowFileDialog(from, FileDialogOptions{
		Native: true, Multiple: true, Mode: FileOpen,
		OnPick:     func(p string) { one = p },
		OnPickMany: func(p []string) { many = p },
	})
	if !asked.Multiple {
		t.Error("the portal was not asked for several files")
	}
	if len(many) != 3 {
		t.Errorf("OnPickMany got %v, want all three", many)
	}
	if one != "/music/one.flac" {
		t.Errorf("OnPick got %q, want the first", one)
	}

	// And a Save dialog is never asked for several, whatever the caller set.
	asked = platform.FileChooserOptions{}
	ShowFileDialog(from, FileDialogOptions{Native: true, Multiple: true, Mode: FileSave})
	if asked.Multiple {
		t.Error("a Save dialog asked the portal for several files")
	}
}
