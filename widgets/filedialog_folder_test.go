package widgets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A folder chooser lists folders and only folders: a file is not an
// answer to "which folder", and one that can be clicked but not picked
// is how a chooser looks broken.
func TestFolderModeListsFoldersOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Maildir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	fd := NewFileDialog(FileDialogOptions{Mode: FileOpenFolder, Path: dir})
	if len(fd.Entries()) != 1 || fd.Entries()[0].Name != "Maildir" || !fd.Entries()[0].Dir {
		t.Fatalf("entries %+v", fd.Entries())
	}
	// With nothing selected the answer is the folder being shown, which
	// is what every folder chooser returns.
	var got string
	fd.opts.OnPick = func(p string) { got = p }
	fd.finish(true)
	if got != dir {
		t.Fatalf("picked %q, want %q", got, dir)
	}
}

// Selecting a folder and pressing Choose returns that folder, not the
// one being listed.
func TestFolderModeReturnsTheSelectedFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	var got string
	fd := NewFileDialog(FileDialogOptions{
		Mode: FileOpenFolder, Path: dir, OnPick: func(p string) { got = p },
	})
	fd.table.Selected = 0
	fd.onSelect(0)
	fd.finish(true)
	if want := filepath.Join(dir, "Archive"); got != want {
		t.Fatalf("picked %q, want %q", got, want)
	}
}

// The title and the button say what the dialog does. "Open" on a folder
// chooser reads as "enter this folder", which is the other thing the
// same click can mean.
func TestFolderModeLabels(t *testing.T) {
	fd := NewFileDialog(FileDialogOptions{Mode: FileOpenFolder, Path: t.TempDir()})
	if fd.opts.Title != "Choose folder" {
		t.Fatalf("title %q", fd.opts.Title)
	}
	if fd.hintText() != "Folders only" {
		t.Fatalf("hint %q", fd.hintText())
	}
	var choose bool
	widget.Walk(fd.overlay, func(c widget.Component) {
		if b, ok := c.(*Button); ok && b.Text == "Choose" {
			choose = true
		}
	})
	if !choose {
		t.Fatal("no Choose button")
	}
}

// Name suggests a file name without moving the listing: Path stays the
// folder, and the field comes up holding the two joined.
func TestSaveNameSuggestsAFileName(t *testing.T) {
	dir := t.TempDir()
	fd := NewFileDialog(FileDialogOptions{Mode: FileSave, Path: dir, Name: "Invoice.eml"})
	want := filepath.Join(dir, "Invoice.eml")
	if fd.path.Text != want {
		t.Fatalf("field %q, want %q", fd.path.Text, want)
	}
	if fd.dir != dir {
		t.Fatalf("listing %q, want %q", fd.dir, dir)
	}
	if fd.Error() != "" {
		t.Fatalf("error %q — the listing went to the file, not its folder", fd.Error())
	}
}

// And a caller with only one string to give: a Save Path that names a
// file, existing or not, is split rather than listed as a folder. That
// was the bug — the dialog showed an empty list and no error.
func TestSavePathNamingAFileIsSplit(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Invoice.eml", "does-not-exist-yet.eml"} {
		if name == "Invoice.eml" {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		full := filepath.Join(dir, name)
		fd := NewFileDialog(FileDialogOptions{Mode: FileSave, Path: full})
		if fd.dir != dir {
			t.Fatalf("%s: listing %q, want %q", name, fd.dir, dir)
		}
		if fd.path.Text != full {
			t.Fatalf("%s: field %q, want %q", name, fd.path.Text, full)
		}
		if fd.Error() != "" {
			t.Fatalf("%s: error %q", name, fd.Error())
		}
	}
}

// The split is for Save on a real filesystem only. An Open dialog, a
// stubbed one and a path whose parent is not a directory keep the Path
// they were given — guessing there would move a listing the caller chose.
func TestSavePathSplitStaysOutOfTheWay(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Invoice.eml")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		what string
		opts FileDialogOptions
	}{
		{"open mode", FileDialogOptions{Mode: FileOpen, Path: file}},
		{"a stubbed listing", FileDialogOptions{Mode: FileSave, Path: file, Entries: []FileInfo{{Name: "a"}}}},
		{"an explicit Name", FileDialogOptions{Mode: FileSave, Path: dir, Name: "Other.eml"}},
		{"no parent to list", FileDialogOptions{Mode: FileSave, Path: "/no/such/thing.eml"}},
	}
	for _, c := range cases {
		want := c.opts.Path
		fd := NewFileDialog(c.opts)
		if fd.dir != want {
			t.Fatalf("%s: listing %q, want %q", c.what, fd.dir, want)
		}
	}
}

// The portal has both natively: `directory` for the folder chooser and
// `current_name` for the suggested name.
func TestNativeFolderAndName(t *testing.T) {
	defer func(old func(platform.FileChooserOptions, func([]string)) bool) { openNative = old }(openNative)
	var asked platform.FileChooserOptions
	openNative = func(o platform.FileChooserOptions, done func([]string)) bool {
		asked = o
		done([]string{"/home/ada/Mail"})
		return true
	}
	from := NewLabel("x")
	from.SetLook(style.DarkLook())
	from.SetHost(&nowHost{})

	var picked string
	ShowFileDialog(from, FileDialogOptions{
		Mode: FileOpenFolder, Native: true, Title: "Choose a mailbox",
		OnPick: func(p string) { picked = p },
	})
	if !asked.Directory || asked.Save || picked != "/home/ada/Mail" {
		t.Fatalf("folder: %+v picked %q", asked, picked)
	}

	dir := t.TempDir()
	ShowFileDialog(from, FileDialogOptions{
		Mode: FileSave, Native: true, Path: dir, Name: "Invoice.eml", OnPick: func(string) {},
	})
	if !asked.Save || asked.Directory || asked.Name != "Invoice.eml" || asked.Folder != dir {
		t.Fatalf("save: %+v", asked)
	}
}
