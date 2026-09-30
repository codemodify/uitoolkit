package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func captionWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Width: 480, Height: 300, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("content"))
	a.PumpOnce()
	if w.Caption() == nil {
		t.Skip("no toolkit-drawn caption in this configuration")
	}
	return a, w
}

func shownIn(w *Window) map[platform.CaptionButton]bool {
	out := map[platform.CaptionButton]bool{}
	c := w.Caption()
	for _, side := range []*widgets.WindowControls{c.LeadControls(), c.TrailControls()} {
		if side == nil {
			continue
		}
		for _, b := range side.Shown() {
			out[b] = true
		}
	}
	return out
}

// A look and the desktop decide which buttons *can* be there; a window
// decides which of those it wants. A tool window with no maximize, a
// dialog with only a close.
func TestWindowHidesACaptionButton(t *testing.T) {
	a, w := captionWindow(t)
	before := shownIn(w)
	if !before[platform.CaptionClose] {
		t.Skip("this look shows no close button to hide")
	}

	w.SetCaptionButtonVisible(platform.CaptionClose, false)
	a.PumpOnce()
	if shownIn(w)[platform.CaptionClose] {
		t.Error("the close button is still there")
	}
	if !w.CaptionButtonHidden(platform.CaptionClose) {
		t.Error("the window does not report it hidden")
	}

	w.SetCaptionButtonVisible(platform.CaptionClose, true)
	a.PumpOnce()
	if !shownIn(w)[platform.CaptionClose] {
		t.Error("the close button did not come back")
	}
}

// The application's own buttons sit beside the window's, on the side it
// asks for, and run their callback when pressed.
func TestWindowCaptionActions(t *testing.T) {
	a, w := captionWindow(t)
	profile, extensions := 0, 0
	w.SetCaptionActions(
		widgets.CaptionAction{Icon: style.IconUser, Name: "Profile", OnClick: func() { profile++ }},
		widgets.CaptionAction{Icon: style.IconMore, Name: "Extensions", Lead: true, OnClick: func() { extensions++ }},
	)
	a.PumpOnce()

	c := w.Caption()
	lead, trail := c.LeadControls(), c.TrailControls()
	if lead == nil || trail == nil {
		t.Fatal("no window controls")
	}
	if got := len(lead.Actions()); got != 1 {
		t.Errorf("%d lead actions, want 1", got)
	}
	if got := len(trail.Actions()); got != 1 {
		t.Errorf("%d trailing actions, want 1", got)
	}
	// They are in what the side paints and hit-tests.
	found := 0
	for _, side := range []*widgets.WindowControls{lead, trail} {
		for _, b := range side.Shown() {
			if b >= 128 {
				found++
			}
		}
	}
	if found != 2 {
		t.Errorf("%d actions reached the painted set, want 2", found)
	}

	// And pressing one runs it.
	for _, side := range []*widgets.WindowControls{lead, trail} {
		for i, b := range side.Shown() {
			if b >= 128 {
				side.ActivateForTest(i)
			}
		}
	}
	if profile != 1 || extensions != 1 {
		t.Errorf("callbacks ran profile=%d extensions=%d", profile, extensions)
	}
}
