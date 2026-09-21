package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// pressBox records the mouse events it is told about. takes is what its
// MousePress answers: a component that takes the press stops the walk up.
type pressBox struct {
	widget.Base
	name     string
	takes    bool
	presses  int
	releases int
	moves    int
	at       []paintengine2d.Point
}

func newPressBox(name string, takes bool) *pressBox {
	b := &pressBox{name: name, takes: takes}
	b.Init(b)
	return b
}

func (b *pressBox) Measure(c layout.Constraints) paintengine2d.Point {
	return paintengine2d.Pt(c.MaxW, c.MaxH)
}

func (b *pressBox) MousePress(e widget.MouseEvent) bool {
	b.presses++
	b.at = append(b.at, e.Pos)
	return b.takes
}

func (b *pressBox) MouseRelease(widget.MouseEvent) bool { b.releases++; return b.takes }
func (b *pressBox) MouseMove(widget.MouseEvent) bool    { b.moves++; return b.takes }

// pressRig is a 400×300 window with a panel that fills it and a child that
// fills the panel's lower half — the shape of a player's face: a container
// with pictures on it.
type pressRig struct {
	a     *Application
	w     *Window
	panel *pressBox
	child *pressBox
}

func newPressRig(t *testing.T, panelTakes, childTakes bool) *pressRig {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	r := &pressRig{}
	r.a = New(Options{Look: style.DarkLook(), Headless: true})
	w, err := r.a.NewWindow(platform.WindowOptions{Title: "Press", Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	r.w = w
	r.panel = newPressBox("panel", panelTakes)
	r.child = newPressBox("child", childTakes)
	r.panel.Add(r.child)
	w.SetContent(r.panel)
	r.a.PumpOnce()
	r.child.SetBounds(paintengine2d.XYWH(0, 150, 400, 150))
	return r
}

func (r *pressRig) press(x, y float32) {
	r.w.dispatch(platform.Event{Kind: platform.EventMouseDown,
		Pos: paintengine2d.Pt(x, y), Button: platform.ButtonLeft})
}

func (r *pressRig) release(x, y float32) {
	r.w.dispatch(platform.Event{Kind: platform.EventMouseUp,
		Pos: paintengine2d.Pt(x, y), Button: platform.ButtonLeft})
}

func (r *pressRig) move(x, y float32) {
	r.w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(x, y)})
}

// TestPressBubblesToAContainer is the defect: Window.hit found the deepest
// component, told that one alone, and an ancestor never heard about a press
// its child ignored — so a panel of labels could not drag its window.
func TestPressBubblesToAContainer(t *testing.T) {
	r := newPressRig(t, true, false)
	r.press(200, 200)
	if r.child.presses != 1 {
		t.Fatalf("the child under the pointer is told first: %d presses", r.child.presses)
	}
	if r.panel.presses != 1 {
		t.Fatalf("the press did not reach the container: %d presses", r.panel.presses)
	}
	// Each one hears it in its own coordinates.
	if got := r.child.at[0]; got != paintengine2d.Pt(200, 50) {
		t.Errorf("child heard %v, want the point in its own space", got)
	}
	if got := r.panel.at[0]; got != paintengine2d.Pt(200, 200) {
		t.Errorf("panel heard %v, want the point in its own space", got)
	}
	// The taker becomes the capture: the gesture is its, wherever the
	// pointer goes and wherever it is let go.
	r.move(390, 10)
	if r.panel.moves != 1 {
		t.Errorf("the container did not keep the pointer: %d moves", r.panel.moves)
	}
	r.release(390, 10)
	if r.panel.releases != 1 {
		t.Errorf("the container did not get the release: %d", r.panel.releases)
	}
	if r.child.releases != 0 {
		t.Errorf("the child that ignored the press got the release: %d", r.child.releases)
	}
}

// TestPressStopsAtTheFirstTaker: a child that takes the press keeps it, and
// nothing above it acts on the same click.
func TestPressStopsAtTheFirstTaker(t *testing.T) {
	r := newPressRig(t, true, true)
	r.press(200, 200)
	if r.child.presses != 1 || r.panel.presses != 0 {
		t.Fatalf("child %d presses, container %d: the walk must stop at the first taker",
			r.child.presses, r.panel.presses)
	}
	r.release(200, 200)
	if r.child.releases != 1 || r.panel.releases != 0 {
		t.Errorf("release: child %d container %d", r.child.releases, r.panel.releases)
	}
}

// TestPressNobodyTakesKeepsTheOldCapture: when the walk finds no taker the
// capture is the deepest component hit, exactly as before bubbling — a
// widget that ignores presses still hears the release over it.
func TestPressNobodyTakesKeepsTheOldCapture(t *testing.T) {
	r := newPressRig(t, false, false)
	r.press(200, 200)
	if r.child.presses != 1 || r.panel.presses != 1 {
		t.Fatalf("both are offered it: child %d container %d", r.child.presses, r.panel.presses)
	}
	r.move(390, 10)
	r.release(390, 10)
	if r.child.releases != 1 || r.child.moves != 1 {
		t.Errorf("the deepest component keeps the capture: %d moves %d releases",
			r.child.moves, r.child.releases)
	}
	if r.panel.releases != 0 {
		t.Errorf("the container that took nothing got the release: %d", r.panel.releases)
	}
}

// focusBox wants focus and takes nothing, so a press on it bubbles.
type focusBox struct{ pressBox }

func newFocusBox(name string) *focusBox {
	b := &focusBox{pressBox{name: name}}
	b.Init(b)
	b.SetWantsFocus(true)
	return b
}

// TestPressFocusesWhatItLandsOnNotTheTaker: dragging a window by its face
// must not take the keyboard away from the control the user was on.
func TestPressFocusesWhatItLandsOnNotTheTaker(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Focus", Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	panel := newPressBox("panel", true)
	panel.SetWantsFocus(true)
	child := newFocusBox("child")
	panel.Add(child)
	w.SetContent(panel)
	a.PumpOnce()
	child.SetBounds(paintengine2d.XYWH(0, 150, 400, 150))

	w.dispatch(platform.Event{Kind: platform.EventMouseDown,
		Pos: paintengine2d.Pt(200, 200), Button: platform.ButtonLeft})
	if w.Focus() != widget.Component(child) {
		t.Fatalf("focus went to %v, want the component the press landed on", w.Focus())
	}
	if panel.presses != 1 {
		t.Fatalf("the container did not get the press it should have taken")
	}
}

// TestPressHandlerThatGivesUpThePointerKeepsNoCapture: a container whose
// MousePress hands the gesture to the desktop (StartMove, as a player's
// face does) has already let the pointer go — taking the press must not
// give it back.
func TestPressHandlerThatGivesUpThePointerKeepsNoCapture(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Move", Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	face := &movePanel{win: w}
	face.Init(face)
	child := newPressBox("label", false)
	face.Add(child)
	w.SetContent(face)
	a.PumpOnce()
	child.SetBounds(paintengine2d.XYWH(0, 150, 400, 150))

	w.dispatch(platform.Event{Kind: platform.EventMouseDown,
		Pos: paintengine2d.Pt(200, 200), Button: platform.ButtonLeft})
	if face.presses != 1 {
		t.Fatalf("the face did not get the press: %d", face.presses)
	}
	o := w.Surface().(*platform.Offscreen)
	if n := o.FrameCalls().Moves; n != 1 {
		t.Fatalf("the press did not start a window move: %d", n)
	}
	if w.capture != nil {
		t.Errorf("capture is %v, want none: the desktop has the pointer", w.capture)
	}
}

// movePanel is a player's face: its own presses move the window.
type movePanel struct {
	widget.Base
	win     *Window
	presses int
}

func (p *movePanel) Measure(c layout.Constraints) paintengine2d.Point {
	return paintengine2d.Pt(c.MaxW, c.MaxH)
}

func (p *movePanel) MousePress(widget.MouseEvent) bool {
	p.presses++
	p.win.StartMove()
	return true
}
