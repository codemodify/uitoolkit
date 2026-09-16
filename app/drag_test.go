package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// dragBox is a component that drags a fixed payload out of itself and
// takes drops of the same type back in: the smallest thing that exercises
// both halves of the protocol.
type dragBox struct {
	widget.Base
	drag *widget.Drag
	// dropped is what landed here; action what the drop performed.
	dropped widget.DropEvent
	drops   int
	// allow narrows what a drop here may do (widget.DropActions).
	allow platform.DragAction
	// takes is whether Drop accepts what it is handed.
	takes bool
	over  int
	left  int
}

func newDragBox(d *widget.Drag) *dragBox {
	b := &dragBox{drag: d, takes: true}
	b.Init(b)
	return b
}

func (b *dragBox) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(200, 100))
}

func (b *dragBox) DragAt(paintengine2d.Point) *widget.Drag { return b.drag }

func (b *dragBox) DropTypes() []string { return []string{"text/uri-list", "text/plain"} }

func (b *dragBox) Drop(e widget.DropEvent) bool {
	if !b.takes {
		return false
	}
	b.dropped = e
	b.drops++
	return true
}

func (b *dragBox) DragOver(paintengine2d.Point) { b.over++ }
func (b *dragBox) DragLeave()                   { b.left++ }

func (b *dragBox) DropActionFor(offered platform.DragAction) platform.DragAction {
	if b.allow == platform.DragNone {
		return platform.DragCopy
	}
	return b.allow
}

// dragWindow is a window filled with one dragBox, and the offscreen
// surface under it.
func dragWindow(t *testing.T, d *widget.Drag) (*Application, *Window, *dragBox, *platform.Offscreen) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	box := newDragBox(d)
	// In a column, so the window keeps empty space below the box: a drag
	// has to have somewhere to be over nothing.
	w.SetContent(widgets.NewColumn(box))
	a.PumpOnce()
	off, ok := w.surf.(*platform.Offscreen)
	if !ok {
		t.Skip("not an offscreen surface")
	}
	return a, w, box, off
}

// press and move past the drag threshold: what the toolkit turns into a
// drag without the widget asking.
func pressAndDrag(a *Application, off *platform.Offscreen, from, to paintengine2d.Point) {
	off.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: from, Button: platform.ButtonLeft})
	off.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: to, Button: platform.ButtonLeft})
	a.PumpOnce()
}

// A press that moves past the threshold asks the component under it for a
// drag and hands it to the surface with everything it offers.
func TestPressBecomesADrag(t *testing.T) {
	d := widget.DragFiles("/home/ada/notes.txt")
	d.Preferred = platform.DragMove
	a, w, _, off := dragWindow(t, d)
	defer w.Close()

	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(21, 21))
	if off.Dragging() {
		t.Fatal("a press that barely moved is not a drag yet")
	}
	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	if !off.Dragging() {
		t.Fatal("past the threshold the press should have become a drag")
	}
	p, _ := off.DragOffer()
	if len(p.Types) == 0 || p.Types[0] != "text/uri-list" {
		t.Fatalf("offered types %q", p.Types)
	}
	if p.Preferred != platform.DragMove || !p.Actions.Has(platform.DragMove) {
		t.Fatalf("actions %v preferred %v", p.Actions, p.Preferred)
	}
	got, ok := off.DragData("text/uri-list")
	if !ok || string(got) != "file:///home/ada/notes.txt\r\n" {
		t.Fatalf("uri-list %q ok %v", got, ok)
	}
	if text, ok := off.DragData("text/plain"); !ok || string(text) != "/home/ada/notes.txt" {
		t.Fatalf("text %q ok %v", text, ok)
	}
	if _, ok := off.DragData("image/png"); ok {
		t.Fatal("a type the drag never offered must not be served")
	}
}

// While the drag is over a target the source is told what a drop would
// do; over nothing it is told the drop is refused.
func TestDragOverAnswersTheSource(t *testing.T) {
	a, w, box, off := dragWindow(t, widget.DragFiles("/tmp/a.txt"))
	defer w.Close()
	box.allow = platform.DragMove
	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))

	off.SimulateDragOver(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	mime, action := off.DragAccepted()
	if mime != "text/uri-list" || action != platform.DragMove {
		t.Fatalf("over the box: got %q %v", mime, action)
	}
	if box.over == 0 {
		t.Fatal("the target should have been told to highlight")
	}
	// Off the target entirely: the window still has to answer, or an XDND
	// source waits for an XdndStatus that never comes.
	empty := paintengine2d.Pt(20, box.Bounds().Max.Y+20)
	if empty.Y > 300 {
		t.Fatalf("the box fills the window: %v", box.Bounds())
	}
	off.SimulateDragOver(empty)
	a.PumpOnce()
	if mime, action = off.DragAccepted(); mime != "" || action != platform.DragNone {
		t.Fatalf("off the target: got %q %v", mime, action)
	}
	if box.left == 0 {
		t.Fatal("the target should have been told the drag left")
	}
}

// A drop of our own drag on our own window serves the data from the drag
// itself and carries the payload, which never had a type at all.
func TestDropOfOurOwnDragCarriesThePayload(t *testing.T) {
	d := widget.DragFiles("/tmp/a.txt")
	d.Payload = []int{7, 8}
	var done []platform.DragAction
	d.Done = func(a platform.DragAction) { done = append(done, a) }
	a, w, box, off := dragWindow(t, d)
	defer w.Close()
	box.allow = platform.DragMove

	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	off.SimulateDragOver(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	off.SimulateDragDrop(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	a.PumpOnce() // the drag ends on the event after the drop

	if box.drops != 1 {
		t.Fatalf("drops %d", box.drops)
	}
	if got, ok := box.dropped.Payload.([]int); !ok || len(got) != 2 || got[0] != 7 {
		t.Fatalf("payload %#v", box.dropped.Payload)
	}
	if box.dropped.Source == nil {
		t.Fatal("an in-process drop names the component it came from")
	}
	if len(box.dropped.Paths) != 1 || box.dropped.Paths[0] != "/tmp/a.txt" {
		t.Fatalf("paths %q", box.dropped.Paths)
	}
	if box.dropped.Action != platform.DragMove {
		t.Fatalf("drop action %v", box.dropped.Action)
	}
	if len(done) != 1 || done[0] != platform.DragMove {
		t.Fatalf("the source is told the move ran: %v", done)
	}
	if off.Dragging() {
		t.Fatal("the drag should be over")
	}
}

// A target that refuses the drop leaves the source with nothing
// performed, so a move never removes the original.
func TestRefusedDropReportsNothingPerformed(t *testing.T) {
	d := widget.DragFiles("/tmp/a.txt")
	d.Actions, d.Preferred = platform.DragMove, platform.DragMove
	var done []platform.DragAction
	d.Done = func(a platform.DragAction) { done = append(done, a) }
	a, w, box, off := dragWindow(t, d)
	defer w.Close()
	box.takes = false

	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	off.SimulateDragOver(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	off.SimulateDragDrop(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	a.PumpOnce()
	if len(done) != 1 || done[0] != platform.DragNone {
		t.Fatalf("a refused drop performs nothing: %v", done)
	}
	if off.DragEnded() != platform.DragNone {
		t.Fatalf("the surface ended with %v", off.DragEnded())
	}
}

// A source that allows only a move over a target that only copies is a
// drop that cannot happen: nobody is shown an action they cannot have.
func TestDragWithNoCommonActionIsRefused(t *testing.T) {
	d := widget.DragFiles("/tmp/a.txt")
	d.Actions, d.Preferred = platform.DragLink, platform.DragLink
	a, w, box, off := dragWindow(t, d)
	defer w.Close()
	box.allow = platform.DragCopy

	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	off.SimulateDragOver(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	if mime, action := off.DragAccepted(); mime != "" || action != platform.DragNone {
		t.Fatalf("got %q %v, want the drop refused", mime, action)
	}
	off.SimulateDragDrop(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	if box.drops != 0 {
		t.Fatal("a drop with no action in common must not reach the target")
	}
}

// Cancelling leaves nothing behind: no drag running, no highlight, and
// the source told nothing was taken.
func TestCancelledDragLeavesNothingBehind(t *testing.T) {
	d := widget.DragFiles("/tmp/a.txt")
	var done []platform.DragAction
	d.Done = func(a platform.DragAction) { done = append(done, a) }
	a, w, box, off := dragWindow(t, d)
	defer w.Close()

	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	off.SimulateDragOver(paintengine2d.Pt(60, 40))
	a.PumpOnce()
	off.CancelDrag()
	a.PumpOnce()
	if off.Dragging() || w.Dragging() {
		t.Fatal("a cancelled drag is over on both sides")
	}
	if len(done) != 1 || done[0] != platform.DragNone {
		t.Fatalf("the source is told nothing was taken: %v", done)
	}
	if box.drops != 0 {
		t.Fatal("a cancelled drag drops nothing")
	}
	// And a fresh press can start another one.
	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	if !off.Dragging() {
		t.Fatal("a cancelled drag must not block the next one")
	}
}

// A drag marked local never reaches the desktop; the toolkit moves it
// between its own windows, and Escape cancels it there.
func TestLocalDragStaysInTheProcess(t *testing.T) {
	d := widget.DragFiles("/tmp/a.txt")
	d.Local = true
	d.Payload = "row-3"
	var done []platform.DragAction
	d.Done = func(a platform.DragAction) { done = append(done, a) }
	a, w, box, off := dragWindow(t, d)
	defer w.Close()

	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	if off.Dragging() {
		t.Fatal("a local drag must not be handed to the desktop")
	}
	if !w.Dragging() {
		t.Fatal("the toolkit should be moving it")
	}
	off.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(60, 40), Button: platform.ButtonLeft})
	a.PumpOnce()
	if box.over == 0 {
		t.Fatal("a local drag still highlights the target it is over")
	}
	off.Inject(platform.Event{Kind: platform.EventMouseUp, Pos: paintengine2d.Pt(60, 40), Button: platform.ButtonLeft})
	a.PumpOnce()
	if box.drops != 1 || box.dropped.Payload != "row-3" {
		t.Fatalf("drops %d payload %#v", box.drops, box.dropped.Payload)
	}
	if len(done) != 1 || done[0] != platform.DragCopy {
		t.Fatalf("done %v", done)
	}
	if w.Dragging() {
		t.Fatal("the drag is over")
	}
}

func TestLocalDragCancelsOnEscape(t *testing.T) {
	d := widget.DragFiles("/tmp/a.txt")
	d.Local = true
	var done []platform.DragAction
	d.Done = func(a platform.DragAction) { done = append(done, a) }
	a, w, box, off := dragWindow(t, d)
	defer w.Close()

	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	off.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(60, 40), Button: platform.ButtonLeft})
	a.PumpOnce()
	off.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	a.PumpOnce()
	if w.Dragging() {
		t.Fatal("Escape ends a local drag")
	}
	if box.left == 0 {
		t.Fatal("the highlight has to be taken back")
	}
	if len(done) != 1 || done[0] != platform.DragNone {
		t.Fatalf("done %v", done)
	}
	// The release after the cancel drops nothing.
	off.Inject(platform.Event{Kind: platform.EventMouseUp, Pos: paintengine2d.Pt(60, 40), Button: platform.ButtonLeft})
	a.PumpOnce()
	if box.drops != 0 {
		t.Fatalf("drops %d after a cancel", box.drops)
	}
}

// Only one drag runs at a time, as a desktop has one pointer.
func TestOneDragAtATime(t *testing.T) {
	a, w, _, off := dragWindow(t, widget.DragFiles("/tmp/a.txt"))
	defer w.Close()
	pressAndDrag(a, off, paintengine2d.Pt(20, 20), paintengine2d.Pt(80, 80))
	if !w.Dragging() {
		t.Fatal("the first drag should be running")
	}
	if w.StartDrag(widget.DragText("second")) {
		t.Fatal("a second drag must not start over the first")
	}
}

// A drag with nothing to offer never starts.
func TestEmptyDragNeverStarts(t *testing.T) {
	a, w, _, _ := dragWindow(t, nil)
	defer w.Close()
	_ = a
	if w.StartDrag(nil) || w.StartDrag(&widget.Drag{}) {
		t.Fatal("a drag offering no type is not a drag")
	}
	if widget.DragFiles() != nil || widget.DragText("") != nil {
		t.Fatal("no files and no text make no drag")
	}
}

// A drag of text offers every name text goes by, so an old X11 client
// that only knows STRING still takes it.
func TestDragTextOffersEveryTextType(t *testing.T) {
	d := widget.DragText("hello")
	for _, m := range []string{"text/plain;charset=utf-8", "UTF8_STRING", "text/plain", "STRING", "TEXT"} {
		if !d.Offers(m) {
			t.Fatalf("a text drag should offer %q", m)
		}
		if b, ok := d.Read(m); !ok || string(b) != "hello" {
			t.Fatalf("%q: got %q ok %v", m, b, ok)
		}
	}
	if d.Types[0] != "text/plain;charset=utf-8" {
		t.Fatalf("UTF-8 comes first, got %q", d.Types[0])
	}
}

// The drop zone widget takes a drag of ours the same way it takes one
// from a file manager.
func TestDropZoneTakesOurOwnDrag(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var got []string
	zone := widgets.NewDropZone(widgets.NewLabel("here"), func(p []string) { got = p })
	w.SetContent(zone)
	a.PumpOnce()
	off, ok := w.surf.(*platform.Offscreen)
	if !ok {
		t.Skip("not an offscreen surface")
	}
	if !w.StartDrag(widget.DragFiles("/tmp/x.png")) {
		t.Fatal("the drag should have started")
	}
	off.SimulateDragOver(paintengine2d.Pt(100, 100))
	a.PumpOnce()
	off.SimulateDragDrop(paintengine2d.Pt(100, 100))
	a.PumpOnce()
	if len(got) != 1 || got[0] != "/tmp/x.png" {
		t.Fatalf("paths %q", got)
	}
}
