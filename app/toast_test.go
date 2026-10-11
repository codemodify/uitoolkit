package app

import (
	"strings"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// toastRig is a window with a button in it.
//
// A note's fade and its timeout are the window's own timers, which are
// real timers (Window.AfterFunc), so the test lets real time pass and
// pumps until what it is waiting for has happened rather than counting
// frames: a fade is some number of one-frame steps, and how many run per
// pump is the machine's business.
type toastRig struct {
	a   *Application
	w   *Window
	btn *widgets.Button
}

func newToastRig(t *testing.T) *toastRig {
	t.Helper()
	r := &toastRig{}
	r.a = New(Options{Look: style.DarkLook(), Headless: true})
	w, err := r.a.NewWindow(platform.WindowOptions{Width: 500, Height: 360, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	r.w = w
	r.btn = widgets.NewButton("Copy", nil)
	w.SetContent(widgets.NewColumn(r.btn))
	r.a.PumpOnce()
	w.frame()
	return r
}

// until pumps the window until cond, and fails if it never comes.
func (r *toastRig) until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		r.a.PumpOnce()
		if cond() {
			r.w.frame()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(4 * time.Millisecond)
	}
}

// settled waits for every note to have finished fading in.
func (r *toastRig) settled(t *testing.T) {
	t.Helper()
	r.until(t, "the notes to be up", func() bool {
		notes := widgets.Toasts(r.btn)
		if len(notes) == 0 {
			return false
		}
		for _, n := range notes {
			if n.Opacity() < 1 || n.Bounds().Empty() {
				return false
			}
		}
		return true
	})
}

// **A window can say something for a few seconds.** A status bar was the
// only place it had for what just happened, and a status bar never goes
// and takes a row of the layout for ever.
func TestAToastAppearsOverTheContentAndGoes(t *testing.T) {
	r := newToastRig(t)
	before := r.w.surf.Buffer().Clone()

	note := widgets.ShowToast(r.btn, "Copied", widgets.ToastInfo, 120*time.Millisecond)
	if note == nil {
		t.Fatal("no note")
	}
	r.settled(t)
	if r.w.ToastLayer() == nil {
		t.Fatal("the window has no toast layer")
	}
	if got := len(widgets.Toasts(r.btn)); got != 1 {
		t.Fatalf("%d notes up, want 1", got)
	}
	// It is over the content, near its bottom edge, and inside it.
	g, box := r.w.geom, note.Bounds()
	if box.Empty() {
		t.Fatal("the note has no box")
	}
	if box.Max.Y > g.content.Max.Y || box.Min.Y < g.content.Min.Y+g.content.Dy()*0.5 {
		t.Errorf("the note is at %v, want it against the bottom of the content %v", box, g.content)
	}
	if box.Min.X < g.content.Min.X || box.Max.X > g.content.Max.X {
		t.Errorf("the note runs outside the content: %v in %v", box, g.content)
	}
	// And it is painted.
	withNote := r.w.surf.Buffer().Clone()
	if n := diffPixels(t, before, withNote); n < 200 {
		t.Errorf("only %d pixels changed: the note is not being painted", n)
	}

	// It goes by itself, and the window comes back to what it was.
	r.until(t, "the note to go", func() bool { return len(widgets.Toasts(r.btn)) == 0 })
	if r.w.ToastLayer() != nil {
		t.Error("the empty layer was left on the window")
	}
	if n := diffPixels(t, before, r.w.surf.Buffer().Clone()); n != 0 {
		t.Errorf("%d pixels differ after the note went", n)
	}
}

// A note is read, not used: every press goes through it to the content
// underneath, which is what keeps it out of the way.
func TestAPressGoesThroughAToast(t *testing.T) {
	r := newToastRig(t)
	clicked := 0
	r.btn.OnClick = func() { clicked++ }
	note := widgets.ShowToast(r.btn, "Copied", widgets.ToastInfo, -1)
	r.settled(t)

	at := paintengine2d.Pt(note.Bounds().Min.X+4, note.Bounds().Min.Y+4)
	if h := r.w.hit(at); h != nil && widget.LiveUnder(h, r.w.ToastLayer()) {
		t.Errorf("a press on the note landed on %T, want whatever is under it", h)
	}
	// And the button still works while a note is up.
	b := r.btn.Bounds()
	mid := paintengine2d.Pt(b.Min.X+b.Dx()*0.5, b.Min.Y+b.Dy()*0.5)
	r.w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: mid, Button: platform.ButtonLeft})
	r.w.dispatch(platform.Event{Kind: platform.EventMouseUp, Pos: mid, Button: platform.ButtonLeft})
	if clicked != 1 {
		t.Errorf("the button under the note was clicked %d times", clicked)
	}
}

// Several notes stack, newest nearest the edge, and none of them overlaps
// another.
func TestNotesStack(t *testing.T) {
	r := newToastRig(t)
	first := widgets.ShowToast(r.btn, "Unlocked", widgets.ToastSuccess, -1)
	second := widgets.ShowToast(r.btn, "Copied", widgets.ToastInfo, -1)
	third := widgets.ShowToast(r.btn, "Could not reach the server", widgets.ToastError, -1)
	r.settled(t)

	if got := len(widgets.Toasts(r.btn)); got != 3 {
		t.Fatalf("%d notes up, want 3", got)
	}
	// Newest at the bottom, each above the one after it.
	if !(third.Bounds().Min.Y > second.Bounds().Min.Y && second.Bounds().Min.Y > first.Bounds().Min.Y) {
		t.Errorf("the pile is in the wrong order: %v %v %v",
			first.Bounds(), second.Bounds(), third.Bounds())
	}
	if third.Bounds().Min.Y < second.Bounds().Max.Y {
		t.Errorf("two notes overlap: %v and %v", second.Bounds(), third.Bounds())
	}
	// Taken down together.
	widgets.DismissToasts(r.btn)
	r.a.PumpOnce()
	if got := len(widgets.Toasts(r.btn)); got != 0 || r.w.ToastLayer() != nil {
		t.Errorf("%d notes left, layer %v", got, r.w.ToastLayer())
	}
}

// A note about something still happening stays until it is taken down.
func TestANoteWithNoTimeoutStays(t *testing.T) {
	r := newToastRig(t)
	note := widgets.ShowToast(r.btn, "Waiting for the request…", widgets.ToastInfo, -1)
	r.settled(t)
	// Long enough that a note with the default timeout would have gone.
	for i := 0; i < 60; i++ {
		r.a.PumpOnce()
		time.Sleep(5 * time.Millisecond)
	}
	if got := len(widgets.Toasts(r.btn)); got != 1 {
		t.Fatalf("%d notes after a third of a second, want the one that was asked to stay", got)
	}
	note.Dismiss()
	r.until(t, "the note to go", func() bool { return len(widgets.Toasts(r.btn)) == 0 })
}

// **Assistive technology is told what a note says.** Nothing moved the
// focus, so a screen reader hears nothing at all unless the window says
// it — this is the toolkit's word for a polite live region.
func TestANoteIsAnnounced(t *testing.T) {
	r := newToastRig(t)
	widgets.ShowToast(r.btn, "Vault locked", widgets.ToastWarning, -1)
	said := r.a.takeAnnouncements()
	if len(said) != 1 || said[0] != "Vault locked" {
		t.Fatalf("announcements %q", said)
	}
	// Taken, they are gone: an adapter reads each announcement once.
	if again := r.a.takeAnnouncements(); len(again) != 0 {
		t.Errorf("the queue still holds %q", again)
	}
	// And the note is in the accessibility tree while it is up, as the
	// window's status.
	r.settled(t)
	var found *a11y.Node
	r.w.AccessibleTree().Walk(func(n *a11y.Node) bool {
		if n.Role == a11y.RoleStatusBar && strings.Contains(n.Name, "Vault locked") {
			found = n
		}
		return true
	})
	if found == nil {
		t.Error("the note is not in the accessibility tree")
	}
}

// Under reduced motion a note appears and goes at once rather than
// fading: the toolkit's own policy, not the desktop's.
func TestANoteDoesNotFadeUnderReducedMotion(t *testing.T) {
	style.SetReduceMotion(true)
	t.Cleanup(func() { style.SetReduceMotion(false) })
	r := newToastRig(t)
	note := widgets.ShowToast(r.btn, "Copied", widgets.ToastInfo, 0)
	r.a.PumpOnce()
	if got := note.Opacity(); got != 1 {
		t.Errorf("the note is at %g before any frame, want it there at once", got)
	}
}
