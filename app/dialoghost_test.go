package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A file chooser opened from a window too small to hold it gets a window of
// its own.
//
// An overlay is held inside its host — 92% of its width, 88% of its height,
// so a modal cannot cover the window it belongs to — and a skinned player's
// window is 275 by 116 pixels. The chooser's card asks for 520 by 380 before
// its listing has a row in it, so the one shown there had a table of no
// height and its buttons below the window's own foot: not a smaller chooser,
// an unusable one. The same fallback is what any window gets when the
// desktop's own chooser is not there to ask.

func compactHost(t *testing.T, w, h int) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Player", Width: w, Height: h, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(win.Close)
	win.SetContent(widgets.NewColumn(widgets.NewLabel("player")))
	a.PumpOnce()
	return a, win
}

func TestAChooserTooBigForItsHostGetsAWindow(t *testing.T) {
	a, w := compactHost(t, 275, 116)
	before := len(a.windows)

	fd := widgets.ShowFileDialog(w.Content(), widgets.FileDialogOptions{
		Title: "Open", Path: "/stub", Entries: []widgets.FileInfo{{Name: "a.go"}},
	})
	a.PumpOnce()

	if fd == nil {
		t.Fatal("no dialog at all")
	}
	if w.Overlay() != nil {
		t.Error("the chooser was mounted in the window that cannot hold it")
	}
	if got := len(a.windows); got != before+1 {
		t.Fatalf("the application has %d windows, want one more than the %d it had", got, before)
	}
	// And it is a dialog belonging to the window that asked.
	dw := a.windows[len(a.windows)-1]
	o := dw.Surface().(*platform.Offscreen)
	if o.WindowRole() != platform.RoleDialog {
		t.Errorf("the chooser's window is a %v", o.WindowRole())
	}
	if o.Owner() != w.Surface() {
		t.Error("the chooser's window does not belong to the window that asked for it")
	}
	if cw, ch := dw.Size(); cw < 520 || ch < 380 {
		t.Errorf("the chooser's window is %dx%d, too small for the card it holds", cw, ch)
	}
}

// A window with room keeps the overlay, which is the lighter thing and what
// every desktop application does: a chooser in a window of its own is the
// fallback, not the new default.
func TestAChooserThatFitsStaysAnOverlay(t *testing.T) {
	a, w := compactHost(t, 900, 700)
	before := len(a.windows)

	widgets.ShowFileDialog(w.Content(), widgets.FileDialogOptions{
		Title: "Open", Path: "/stub", Entries: []widgets.FileInfo{{Name: "a.go"}},
	})
	a.PumpOnce()

	if w.Overlay() == nil {
		t.Error("a window with room for the chooser did not get the overlay")
	}
	if got := len(a.windows); got != before {
		t.Errorf("the application opened %d extra windows for a chooser that fits", got-before)
	}
}

// Choosing a file closes the window the chooser was in, and answers.
func TestFinishingAChooserInItsOwnWindowClosesIt(t *testing.T) {
	a, w := compactHost(t, 275, 116)
	var picked string
	done := false
	fd := widgets.ShowFileDialog(w.Content(), widgets.FileDialogOptions{
		Title: "Open", Path: "/stub", Entries: []widgets.FileInfo{{Name: "a.go"}},
		OnPick:   func(p string) { picked, done = p, true },
		OnCancel: func() { done = true },
	})
	a.PumpOnce()
	dw := a.windows[len(a.windows)-1]

	fd.Cancel()
	a.PumpOnce()

	if !done {
		t.Error("the caller was never told the chooser had finished")
	}
	if picked != "" {
		t.Errorf("a cancelled chooser picked %q", picked)
	}
	if !dw.Closed() {
		t.Error("the chooser's window is still open")
	}
}

// And the window's own close control ends the chooser, rather than leaving
// the application waiting for an answer that is never coming.
func TestClosingTheChoosersWindowCancelsIt(t *testing.T) {
	a, w := compactHost(t, 275, 116)
	cancels := 0
	widgets.ShowFileDialog(w.Content(), widgets.FileDialogOptions{
		Title: "Open", Path: "/stub", Entries: []widgets.FileInfo{{Name: "a.go"}},
		OnCancel: func() { cancels++ },
	})
	a.PumpOnce()
	dw := a.windows[len(a.windows)-1]

	dw.RequestClose()
	a.PumpOnce()

	if cancels != 1 {
		t.Errorf("the close control cancelled the chooser %d times, want once", cancels)
	}
	if !dw.Closed() {
		t.Error("the chooser's window is still open")
	}
}
