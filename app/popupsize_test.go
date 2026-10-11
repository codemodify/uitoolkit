package app

import (
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/diag"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// dipBox is a popup whose size is a design length in the look it is in:
// 100x40 at 1x, 200x80 on a screen at twice that. Every real popup is
// like this — its rows, its padding and its text all come out of the
// look — which is why measuring one before it has the window's look
// gives the wrong answer.
type dipBox struct {
	widget.Base
}

func newDipBox() *dipBox {
	d := &dipBox{}
	d.Init(d)
	return d
}

func (d *dipBox) Measure(layout.Constraints) paintengine2d.Point {
	return paintengine2d.Pt(style.Dip(d.Look(), 100), style.Dip(d.Look(), 40))
}

func (d *dipBox) Arrange(r paintengine2d.Rect) { d.SetBounds(r) }

func popupRig(t *testing.T, scale float32) (*Application, *Window, *widgets.Label) {
	t.Helper()
	// Findings are global and deduped, so each case starts from none.
	diag.Reset()
	t.Cleanup(diag.Reset)
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: scale})
	w, err := a.NewWindow(platform.WindowOptions{Width: 800, Height: 600, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	anchor := widgets.NewLabel("content")
	w.SetContent(anchor)
	a.PumpOnce()
	return a, w, anchor
}

// **A popup measured before it was shown was shown at the wrong size.**
// The look reaches a popup in ShowPopup — a popup is a root, so it has
// none of its own before that — so a caller who measured it to arrange
// it measured it in the toolkit's default look at scale 1. On a screen
// at twice that the popup was drawn at twice the size inside a box of
// the first: the words cut short, the last rows off the bottom.
func TestAPopupArrangedBeforeShowingIsGrownToItsSize(t *testing.T) {
	a, _, anchor := popupRig(t, 2)
	pop := newDipBox()

	// What a caller must do to arrange a popup: measure it, then place
	// it. Both happen before the popup has ever heard of the window.
	sz := pop.Measure(layout.Unbounded())
	if sz.X != 100 || sz.Y != 40 {
		t.Fatalf("with no look the box measures %v, want the 1x 100x40", sz)
	}
	pop.Arrange(paintengine2d.XYWH(20, 20, sz.X, sz.Y))
	if !widget.ShowPopup(anchor, pop) {
		t.Fatal("no popup")
	}
	a.PumpOnce()

	if got := pop.Bounds(); got.Dx() != 200 || got.Dy() != 80 {
		t.Errorf("the popup is %gx%g, want the 200x80 it is drawn at", got.Dx(), got.Dy())
	}
	// It grew where it stood.
	if got := pop.Bounds().Min; got.X != 20 || got.Y != 20 {
		t.Errorf("the popup moved to %v, want it left at 20,20", got)
	}
	// And it said so, with the call that avoids it.
	var note string
	for _, f := range a.Diagnostics() {
		if f.Area == "popup" {
			note = f.String()
		}
	}
	if !strings.Contains(note, "PreparePopup") {
		t.Errorf("the diagnostic was %q, want one naming PreparePopup", note)
	}
}

// Measuring after PreparePopup, which every Place* helper does, leaves
// nothing to grow — and a size somebody chose in the right look stands,
// smaller than the content or not: a panel holding a list that scrolls
// is a popup deliberately shorter than what is in it.
func TestAPreparedPopupKeepsTheSizeItWasGiven(t *testing.T) {
	a, _, anchor := popupRig(t, 2)
	pop := newDipBox()

	widget.PreparePopup(anchor, pop)
	sz := pop.Measure(layout.Unbounded())
	if sz.X != 200 || sz.Y != 80 {
		t.Fatalf("prepared, the box measures %v, want the 2x 200x80", sz)
	}
	// Deliberately shorter than its content.
	pop.Arrange(paintengine2d.XYWH(20, 20, sz.X, sz.Y/2))
	if !widget.ShowPopup(anchor, pop) {
		t.Fatal("no popup")
	}
	a.PumpOnce()

	if got := pop.Bounds(); got.Dx() != 200 || got.Dy() != 40 {
		t.Errorf("the popup is %gx%g, want the 200x40 it was given", got.Dx(), got.Dy())
	}
	for _, f := range a.Diagnostics() {
		if f.Area == "popup" {
			t.Errorf("a prepared popup reported %v", f)
		}
	}
}

// Growing stays inside the window: a popup at the bottom edge is moved
// up to fit rather than hanging off it, and one taller than the window
// keeps the room there is.
func TestAGrownPopupStaysInTheWindow(t *testing.T) {
	a, w, anchor := popupRig(t, 2)
	pop := newDipBox()
	sz := pop.Measure(layout.Unbounded())
	_, boxH := w.Surface().Size()
	pop.Arrange(paintengine2d.XYWH(10, float32(boxH)-sz.Y-6, sz.X, sz.Y))
	if !widget.ShowPopup(anchor, pop) {
		t.Fatal("no popup")
	}
	a.PumpOnce()
	if got := pop.Bounds(); got.Max.Y > float32(boxH) {
		t.Errorf("the popup ends at %g, past the window's %d", got.Max.Y, boxH)
	}
}

// Growing never shrinks. A popup arranged *larger* than it measures is a
// size somebody chose — room for what a list is about to be filled with,
// a panel sized to its widest page — and the look arriving is no reason
// to take it back. It is the box too small for its content that is the
// one case a wrong measurement can be told from a choice.
func TestAPopupArrangedLargerKeepsItsSize(t *testing.T) {
	a, _, anchor := popupRig(t, 2)
	pop := newDipBox()
	pop.Arrange(paintengine2d.XYWH(20, 20, 400, 300))
	if !widget.ShowPopup(anchor, pop) {
		t.Fatal("no popup")
	}
	a.PumpOnce()
	if got := pop.Bounds(); got.Dx() != 400 || got.Dy() != 300 {
		t.Errorf("the popup is %gx%g, want the 400x300 it was given", got.Dx(), got.Dy())
	}
}

// metricBox is a popup sized the way a real one is: out of the look's
// own numbers — a row's height, a control's height — rather than out of
// the display scale alone.
type metricBox struct {
	widget.Base
}

func newMetricBox() *metricBox {
	m := &metricBox{}
	m.Init(m)
	return m
}

func (d *metricBox) Measure(layout.Constraints) paintengine2d.Point {
	m := d.Look().Metrics()
	return paintengine2d.Pt(m.ControlH*4, m.RowH*3)
}

func (d *metricBox) Arrange(r paintengine2d.Rect) { d.SetBounds(r) }

// The scale is not the only thing that decides a size. A popup measured
// in one look and drawn in another at the *same* scale is just as wrong:
// a relaxed row is taller than a default one, and the last rows fall off
// the bottom all the same. What a look measures, not which object it is,
// is what the refit turns on.
func TestAPopupMeasuredInAnotherLookOfTheSameScaleIsGrown(t *testing.T) {
	diag.Reset()
	t.Cleanup(diag.Reset)
	relaxed := style.WithDensity(style.DarkLook(), style.DensityRelaxed)
	a := New(Options{Look: relaxed, Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 800, Height: 600, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	anchor := widgets.NewLabel("content")
	w.SetContent(anchor)
	a.PumpOnce()
	if style.LookScale(w.Look()) != style.LookScale(style.DarkLook()) {
		t.Fatal("the two looks must be at the same scale for this to be about metrics")
	}
	if w.Look().Metrics() == style.DarkLook().Metrics() {
		t.Fatal("the relaxed look measures the same as the default one")
	}

	pop := newMetricBox()
	sz := pop.Measure(layout.Unbounded())
	pop.Arrange(paintengine2d.XYWH(20, 20, sz.X, sz.Y))
	if !widget.ShowPopup(anchor, pop) {
		t.Fatal("no popup")
	}
	a.PumpOnce()

	m := w.Look().Metrics()
	if got := pop.Bounds(); got.Dx() != m.ControlH*4 || got.Dy() != m.RowH*3 {
		t.Errorf("the popup is %gx%g, want the %gx%g it is drawn at",
			got.Dx(), got.Dy(), m.ControlH*4, m.RowH*3)
	}
}
