package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// **A window can be laid out at once.** It is otherwise laid out at its
// next frame, which is too late for anything that wants to read the
// laid-out window before it is drawn — a picture of it taken the moment
// it is shown, for a card turning over or a cross-fade — and an
// application had to measure and arrange the layers itself to get boxes.
func TestAWindowCanBeLaidOutAtOnce(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	body := widgets.NewButton("Body", nil)
	w.SetContent(widgets.NewColumn(body))
	// Nothing has drawn yet, so nothing has a box.
	if got := body.Bounds(); got != (paintengine2d.Rect{}) {
		t.Fatalf("the button already has a box: %v", got)
	}

	w.Layout()
	box := body.Bounds()
	if box.Dx() <= 0 || box.Dy() <= 0 {
		t.Fatalf("after Layout the button's box is %v", box)
	}
	// And what the frame would have done is done: the window counts as
	// laid out, so the frame does not do it again.
	if !w.laid {
		t.Error("the window still says it needs laying out")
	}
	// A window laid out for the first time by Layout must also have been
	// fully invalidated, or the frame after it paints a damage list
	// written when nothing had a size.
	if !w.full {
		t.Error("the first Layout did not invalidate the window")
	}
	// Twice is the same answer.
	w.Layout()
	if got := body.Bounds(); got != box {
		t.Errorf("a second Layout moved the button to %v from %v", got, box)
	}
	// The frame that follows paints what was laid out, in the same boxes.
	w.frame()
	if got := body.Bounds(); got != box {
		t.Errorf("the frame moved the button to %v from %v", got, box)
	}
	// A relayout asked for and then taken at once reaches the new size,
	// and what the window shows next is the new tree: laying out by hand
	// throws away the recorded display list, exactly as the frame does,
	// or the frame after it replays a scene of the tree that was.
	shown := w.surf.Buffer().Clone()
	w.SetContent(widgets.NewColumn(widgets.NewLabel("A different page altogether")))
	w.Layout()
	if w.Content().Bounds().Dx() <= 0 {
		t.Error("the new content was not laid out")
	}
	w.frame()
	if n := diffPixels(t, shown, w.surf.Buffer().Clone()); n == 0 {
		t.Error("the window is still showing the page it had: the recorded scene was replayed")
	}
}
