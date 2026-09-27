package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
)

// Who draws the window's drop shadow, and who runs a resize from its
// edges, is the window system's to say — and until FrameSystemShadow
// existed the contract could not say it. Frame.Margin assumed the client
// owned the shadow, because on Wayland and X11 it does: the window grows
// by an invisible band and paints one into it.
//
// Windows' DWM and macOS's AppKit draw it themselves, around the window.
// A toolkit that reserved a margin there would give every window two
// shadows and line it up with nothing else on the desktop — and it would
// do so *because the boundary told it to*, which is why this is settled
// before there is a backend to be wrong.

// The Linux answer, unchanged: the client reserves a margin for its own
// shadow and a band inside it for the resize edges.
func TestClientOwnsItsShadowByDefault(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	f := r.w.wantDecorFrame()
	if f.Margin.Zero() {
		t.Fatal("a look with a shadow reserved no margin for it")
	}
	if f.Input.Zero() {
		t.Fatal("a resizable window reserved no band for its resize edges")
	}
	if !f.Alpha {
		t.Error("a window with a shadow needs an alpha channel")
	}
}

// Where the window system draws the shadow, the toolkit reserves nothing
// and paints nothing. The margin going to zero is what turns the painting
// off too, because geom.shadow is !margin.Zero().
func TestSystemShadowLeavesNoMarginAndPaintsNothing(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	r.o.SimulateSystemShadow(true)

	f := r.w.wantDecorFrame()
	if !f.Margin.Zero() {
		t.Fatalf("the window reserved %v for a shadow the desktop draws", f.Margin)
	}
	if !f.Input.Zero() {
		t.Fatalf("the resize band %v reaches into a margin that is not there", f.Input)
	}
	// And nothing paints one: the geometry the window lays out with has
	// no shadow in it.
	r.a.PumpOnce()
	if r.w.geom.shadow {
		t.Error("the window still paints a shadow of its own")
	}
	// The corners are still the look's — the window system draws the
	// shadow, not the window.
	if f.Radius == [4]float32{} {
		t.Error("the rounded corners went with the shadow")
	}
	if !f.Alpha {
		t.Error("a window with rounded corners still needs alpha")
	}
}

// macOS sets both bits: a resizable NSWindow already has resize borders
// AppKit manages, and there is no public API to begin a resize from an
// edge — so a band of the toolkit's would be unnecessary and unreachable.
// The shadow may still be the client's on a platform that wanted that, so
// the bits are independent.
func TestSystemResizeBandLeavesNoInputBand(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	r.o.SimulateSystemResizeBand(true)

	f := r.w.wantDecorFrame()
	if !f.Input.Zero() {
		t.Fatalf("the toolkit reserved %v for edges the desktop resizes", f.Input)
	}
	// The shadow is still ours here: only the band was taken away.
	if f.Margin.Zero() {
		t.Fatal("the margin went with the band; they are separate capabilities")
	}
}

// Both bits print, so a diagnostic says who owns what.
func TestOwnershipCapsPrint(t *testing.T) {
	c := platform.FrameSystemShadow | platform.FrameSystemResizeBand
	if got := c.String(); got != "system-shadow system-resize-band" {
		t.Fatalf("caps print as %q", got)
	}
}
