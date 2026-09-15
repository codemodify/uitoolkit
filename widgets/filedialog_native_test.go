package widgets

import (
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// timerHost runs AfterFunc callbacks at once (the UI goroutine in a test).
type nowHost struct{ host }

func (h *nowHost) AfterFunc(d time.Duration, fn func()) func() { fn(); return func() {} }

// With Native set, the desktop's dialog answers through the portal and
// OnPick gets its path; without a portal the toolkit's dialog shows.
func TestNativeFileDialog(t *testing.T) {
	defer func(old func(platform.FileChooserOptions, func([]string)) bool) { openNative = old }(openNative)
	var asked platform.FileChooserOptions
	openNative = func(o platform.FileChooserOptions, done func([]string)) bool {
		asked = o
		done([]string{"/tmp/picked.txt"})
		return true
	}
	from := NewLabel("x")
	from.SetLook(style.DarkLook())
	from.SetHost(&nowHost{})
	var picked string
	fd := ShowFileDialog(from, FileDialogOptions{Title: "Attach file", Native: true, Filter: "*.txt;*.md",
		OnPick: func(p string) { picked = p }})
	if fd != nil || picked != "/tmp/picked.txt" {
		t.Fatalf("native: dialog %v, picked %q", fd, picked)
	}
	if asked.Title != "Attach file" || len(asked.Filters) != 1 || len(asked.Filters[0].Patterns) != 2 {
		t.Fatalf("options %+v", asked)
	}
	// Cancelled.
	openNative = func(o platform.FileChooserOptions, done func([]string)) bool { done(nil); return true }
	cancelled := false
	ShowFileDialog(from, FileDialogOptions{Native: true, OnCancel: func() { cancelled = true }})
	if !cancelled {
		t.Fatal("a cancelled native dialog calls OnCancel")
	}
	// No portal: the toolkit's dialog.
	openNative = func(platform.FileChooserOptions, func([]string)) bool { return false }
	if fd := ShowFileDialog(from, FileDialogOptions{Native: true}); fd == nil {
		t.Fatal("without a portal the toolkit's dialog shows")
	}
}
