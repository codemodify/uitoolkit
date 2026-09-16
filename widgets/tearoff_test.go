package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// tearHost is a window that can carry a torn-off window, and records what
// a tear-off asked it for.
type tearHost struct {
	frameHost
	carries bool
	drag    *widget.Drag
	tear    *widget.TearOff
	window  widget.TearOffWindow
	started int
	// refuse makes StartTearOff fail, as a second drag would.
	refuse bool
}

func (h *tearHost) DragsWindows() bool { return h.carries }

func (h *tearHost) StartTearOff(d *widget.Drag, t *widget.TearOff) bool {
	if h.refuse {
		return false
	}
	h.drag, h.tear, h.started = d, t, h.started+1
	if h.carries {
		// A desktop that carries the window opens it as the drag starts,
		// as app.Window does.
		h.window = t.Open()
	}
	return true
}

// tornWindow is the window a tear-off opens, standing in for app.Window.
type tornWindow struct {
	host
	closed int
}

func (w *tornWindow) Surface() platform.Surface { return nil }
func (w *tornWindow) Close()                    { w.closed++ }

// tearRig is a strip whose tabs tear off into tornWindows.
type tearRig struct {
	*tabsRig
	host *tearHost
	torn []BrowserTab
	win  *tornWindow
	// merged is what OnMergeTab was given.
	mergedAt  int
	mergedTab BrowserTab
	merges    int
}

func newTearRig(t *testing.T, carries bool) *tearRig {
	base := newTabsRig(t)
	r := &tearRig{tabsRig: base, host: &tearHost{carries: carries}, mergedAt: -1}
	r.host.active = true
	base.s.SetHost(r.host)
	base.s.OnTearOff = func(i int, tab BrowserTab) widget.TearOffWindow {
		r.torn = append(r.torn, tab)
		r.win = &tornWindow{}
		return r.win
	}
	base.s.OnMergeTab = func(at int, tab BrowserTab, _ widget.DropEvent) bool {
		r.mergedAt, r.mergedTab, r.merges = at, tab, r.merges+1
		base.s.InsertTab(at, tab)
		return true
	}
	arrange(base.s, 700, 34)
	return r
}

// dragTabTo presses tab i and drags the pointer to p (strip-local).
func (r *tearRig) dragTabTo(i int, p paintengine2d.Point) {
	from := r.center(i)
	r.press(from, platform.ButtonLeft)
	r.s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(from.X+20, from.Y), Button: platform.ButtonLeft})
	r.s.MouseMove(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
}

// A tab dragged along the strip is being reordered; the same drag pulled
// clear of the strip takes the tab out of the window.
func TestTabDraggedOutOfTheStripTearsOff(t *testing.T) {
	r := newTearRig(t, true)
	c := r.center(1)

	// Inside the strip, and just outside it: still a reorder.
	r.dragTabTo(1, paintengine2d.Pt(c.X+40, c.Y))
	if r.host.started != 0 || r.s.Len() != 3 {
		t.Fatalf("a drag along the strip reorders: started=%d len=%d", r.host.started, r.s.Len())
	}
	r.s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(c.X+40, 34+8), Button: platform.ButtonLeft})
	if r.host.started != 0 {
		t.Fatal("a tab wobbling at the edge of the strip stays in it")
	}

	// Pulled well below the strip: out it goes.
	r.s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(c.X+40, 34+40), Button: platform.ButtonLeft})
	if r.host.started != 1 {
		t.Fatalf("the tab should have torn off: started=%d", r.host.started)
	}
	if got := r.titles(); len(got) != 2 || got[0] != "uitoolkit" || got[1] != "notes" {
		t.Fatalf("the tab leaves the strip at once: %v", got)
	}
	if len(r.torn) != 1 || r.torn[0].Title != "paintengine2d" {
		t.Fatalf("the window was asked for %v", r.torn)
	}

	// What the drag offers: the private type first, the tab itself in the
	// payload, and a move — a tab is in one window or the other.
	d := r.host.drag
	if len(d.Types) == 0 || d.Types[0] != TabMimeType {
		t.Fatalf("types %v", d.Types)
	}
	if b, ok := d.Read(TabMimeType); !ok || string(b) != "paintengine2d" {
		t.Fatalf("the private type carries the title: %q ok=%v", b, ok)
	}
	if tab, ok := d.Payload.(BrowserTab); !ok || tab.Title != "paintengine2d" {
		t.Fatalf("payload %#v", d.Payload)
	}
	if !d.Allowed().Has(platform.DragMove) || d.Preferred != platform.DragMove {
		t.Fatalf("actions %v preferred %v", d.Actions, d.Preferred)
	}
	if d.Image == nil {
		t.Fatal("a torn-off tab draws a picture, for the desktop that cannot carry the window")
	}
	if d.Source != widget.Component(r.s) {
		t.Fatal("the drag names the strip it came from")
	}
}

// Escape puts the tab back where it was; a merge or a drop on the desktop
// leaves it out.
func TestTearOffEndingsPutTheTabBackOrLeaveItOut(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  widget.TearResult
		want []string
	}{
		{"cancelled", widget.TearCancelled, []string{"uitoolkit", "paintengine2d", "notes"}},
		{"kept", widget.TearKept, []string{"uitoolkit", "notes"}},
		{"merged", widget.TearMerged, []string{"uitoolkit", "notes"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newTearRig(t, true)
			c := r.center(1)
			r.dragTabTo(1, paintengine2d.Pt(c.X, 34+40))
			if r.host.started != 1 {
				t.Fatal("no tear-off")
			}
			r.host.tear.Done(tc.res, r.host.window)
			if got := r.titles(); len(got) != len(tc.want) || got[0] != tc.want[0] || got[1] != tc.want[1] {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// Where the desktop cannot carry a window the tab stays in the strip
// until the drag ends: an empty gap under the pointer for the length of
// the drag would be worse than none.
func TestTearOffWithoutACarriedWindowKeepsTheTab(t *testing.T) {
	r := newTearRig(t, false)
	c := r.center(1)
	r.dragTabTo(1, paintengine2d.Pt(c.X, 34+40))
	if r.host.started != 1 {
		t.Fatal("the drag still starts")
	}
	if got := r.titles(); len(got) != 3 {
		t.Fatalf("the tab stays until the drop: %v", got)
	}
	r.host.tear.Done(widget.TearKept, r.host.window)
	if got := r.titles(); len(got) != 2 || got[1] != "notes" {
		t.Fatalf("at the drop it leaves: %v", got)
	}
}

// A window's last tab never tears off: there would be nothing left
// behind, and dragging a window by its caption already means that.
func TestLastTabNeverTearsOff(t *testing.T) {
	r := newTearRig(t, true)
	r.s.RemoveTab(2)
	r.s.RemoveTab(1)
	c := r.center(0)
	r.dragTabTo(0, paintengine2d.Pt(c.X, 34+40))
	if r.host.started != 0 || r.s.Len() != 1 {
		t.Fatalf("started=%d len=%d", r.host.started, r.s.Len())
	}
}

// A drag that will not start leaves the strip as it was.
func TestTearOffThatCannotStartPutsTheTabBack(t *testing.T) {
	r := newTearRig(t, true)
	r.host.refuse = true
	c := r.center(1)
	r.dragTabTo(1, paintengine2d.Pt(c.X, 34+40))
	if got := r.titles(); len(got) != 3 || got[1] != "paintengine2d" {
		t.Fatalf("the tab goes back where it was: %v", got)
	}
}

// TearOffTab is the same thing from a menu item, without a drag.
func TestTearOffTabFromCode(t *testing.T) {
	r := newTearRig(t, true)
	if !r.s.TearOffTab(0, r.center(0)) {
		t.Fatal("TearOffTab")
	}
	if got := r.titles(); len(got) != 2 || got[0] != "paintengine2d" {
		t.Fatalf("%v", got)
	}
	if r.s.TearOffTab(9, paintengine2d.Point{}) {
		t.Fatal("there is no tab 9")
	}
}

// The strip takes tabs from other windows: the type it offers for them,
// where a drop lands, and what it does with one from another application.
func TestStripTakesATornOffTab(t *testing.T) {
	r := newTearRig(t, true)
	types := r.s.DropTypes()
	if len(types) == 0 || types[0] != TabMimeType {
		t.Fatalf("DropTypes %v", types)
	}
	r.s.OnDropTab = func(int, widget.DropEvent) bool { return true }
	if got := r.s.DropTypes(); len(got) != 2 || got[1] != "text/uri-list" {
		t.Fatalf("both halves, tabs first: %v", got)
	}

	// Dropped on the left half of tab 1: it goes in before it.
	e := widget.DropEvent{Pos: r.leftOf(1), Mime: TabMimeType, Payload: BrowserTab{Title: "torn", Data: 7}}
	if !r.s.Drop(e) {
		t.Fatal("the strip should take it")
	}
	if r.mergedAt != 1 || r.mergedTab.Title != "torn" || r.mergedTab.Data != 7 {
		t.Fatalf("merged at %d: %#v", r.mergedAt, r.mergedTab)
	}
	if got := r.titles(); got[1] != "torn" {
		t.Fatalf("%v", got)
	}

	// From another application there is no payload, only the title.
	e = widget.DropEvent{Pos: paintengine2d.Pt(690, 17), Mime: TabMimeType, Data: []byte("elsewhere\n")}
	if !r.s.Drop(e) {
		t.Fatal("a tab from another application is taken by its title")
	}
	if r.mergedAt != r.s.Len()-1 || r.mergedTab.Title != "elsewhere" || r.mergedTab.Data != nil {
		t.Fatalf("merged at %d: %#v", r.mergedAt, r.mergedTab)
	}
	// Nothing at all in the private type is not a tab.
	if r.s.Drop(widget.DropEvent{Pos: paintengine2d.Pt(20, 17), Mime: TabMimeType}) {
		t.Fatal("an empty tab drop takes nothing")
	}
}

// A file dropped on a tab still lands on that tab, and the two marks —
// the caret between tabs and the highlight on one — never show at once.
func TestStripMarksTabsAndFilesApart(t *testing.T) {
	r := newTearRig(t, true)
	took := -1
	r.s.OnDropTab = func(i int, _ widget.DropEvent) bool { took = i; return true }

	r.s.DragOverMime(r.center(1), TabMimeType)
	if r.s.dropAt != 1 || r.s.dropTab != -1 {
		t.Fatalf("a tab lands in a gap: dropAt=%d dropTab=%d", r.s.dropAt, r.s.dropTab)
	}
	r.s.DragOverMime(r.center(1), "text/uri-list")
	if r.s.dropTab != 1 || r.s.dropAt != -1 {
		t.Fatalf("a file lands on a tab: dropAt=%d dropTab=%d", r.s.dropAt, r.s.dropTab)
	}
	r.s.DragLeave()
	if r.s.dropTab != -1 || r.s.dropAt != -1 {
		t.Fatal("the marks go with the drag")
	}
	if !r.s.Drop(widget.DropEvent{Pos: r.center(1), Mime: "text/uri-list", Paths: []string{"/tmp/a"}}) || took != 1 {
		t.Fatalf("the file went to tab %d", took)
	}
	// The caret has a box to paint in for every gap, including the end.
	g := r.s.geom()
	for i := 0; i <= r.s.Len(); i++ {
		if r.s.caretRect(i, g).Empty() {
			t.Fatalf("no caret for gap %d", i)
		}
	}
}

// Where a drop lands, by the half of the tab the pointer is over.
func TestInsertIndexAtHalves(t *testing.T) {
	r := newTearRig(t, true)
	g := r.s.geom()
	for i := range g.slots {
		if got := r.s.insertIndexAt(r.leftOf(i)); got != i {
			t.Fatalf("left of tab %d: got %d", i, got)
		}
		if got := r.s.insertIndexAt(r.rightOf(i)); got != i+1 {
			t.Fatalf("right of tab %d: got %d", i, got)
		}
	}
	if got := r.s.insertIndexAt(paintengine2d.Pt(690, 17)); got != len(g.slots) {
		t.Fatalf("past the last tab: got %d", got)
	}
}

// leftOf and rightOf are points inside the two halves of tab i.
func (r *tearRig) leftOf(i int) paintengine2d.Point {
	s := r.s.geom().slots[i]
	return paintengine2d.Pt(s.Min.X+4, (s.Min.Y+s.Max.Y)/2)
}

func (r *tearRig) rightOf(i int) paintengine2d.Point {
	s := r.s.geom().slots[i]
	return paintengine2d.Pt(s.Max.X-4, (s.Min.Y+s.Max.Y)/2)
}

// The caret paints where the tab would go, in the pack's accent.
func TestInsertCaretPaints(t *testing.T) {
	r := newTearRig(t, true)
	r.s.DragOverMime(r.leftOf(2), TabMimeType)
	img := paintengine2d.NewImage(700, 34)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	widget.PaintTree(r.s, ctx, nil)
	box := r.s.caretRect(2, r.s.geom())
	x, y := int(box.Min.X+box.Dx()/2), int(box.Min.Y+box.Dy()/2)
	want := style.DarkLook().Palette().Accent.NRGBA()
	gr, gg, gb, ga := img.At(x, y).RGBA()
	wr, wg, wb, _ := want.RGBA()
	if ga == 0 || gr != wr || gg != wg || gb != wb {
		t.Fatalf("the caret at %d,%d is %v,%v,%v,%v, want the accent %v,%v,%v",
			x, y, gr, gg, gb, ga, wr, wg, wb)
	}
}
