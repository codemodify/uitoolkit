package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// glassWindow is a window with a narrow sidebar beside a wide document.
func glassWindow(t *testing.T) (*Application, *Window, *widgets.Panel) {
	t.Helper()
	a := New(Options{Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	side := widgets.NewPanel("", widgets.NewLabel("sidebar"))
	body := widgets.NewPanel("", widgets.NewLabel("document"))
	row := widgets.NewRow(side, body)
	row.AddFlex(side, 1)
	row.AddFlex(body, 3)
	win.SetContent(row)
	win.SetGlass(true)
	a.PumpOnce()
	win.Capture()
	return a, win, side
}

// **Glass can be confined to one component.** A sidebar of glass beside
// an opaque document is the usual shape of a window that asks for blur
// at all; blurring the whole surface for the sidebar's sake would put
// the desktop behind the document too.
//
// It is the box computation that is tested rather than the region the
// display server is handed, because an offscreen window never asks for
// blur at all and a test through the frame skips instead of running —
// which is what the first version of this did on every machine.
func TestGlassCanBeConfinedToOneComponent(t *testing.T) {
	_, win, side := glassWindow(t)

	if _, ok := win.glassUnderBox(platform.FrameInsets{}); ok {
		t.Error("a window with no component named confined its glass to one")
	}

	win.SetGlassUnder(side)
	win.Capture()
	box, ok := win.glassUnderBox(platform.FrameInsets{})
	if !ok {
		t.Fatal("the glass is not confined to the sidebar")
	}
	want := widget.DeviceBounds(side)
	if want.Dx() <= 0 {
		t.Fatal("the sidebar was never laid out, so this proves nothing")
	}
	if float32(box.W) < want.Dx()-2 || float32(box.W) > want.Dx()+2 {
		t.Errorf("the glass box is %d wide and the sidebar %.1f", box.W, want.Dx())
	}
	if float32(box.X) < want.Min.X-2 || float32(box.X) > want.Min.X+2 {
		t.Errorf("the glass box starts at %d and the sidebar at %.1f", box.X, want.Min.X)
	}
	// It is the sidebar, not the window: a quarter of the width here.
	if box.W >= 400 {
		t.Errorf("the glass box is %d wide on a 400 window: it is not confined", box.W)
	}

	win.SetGlassUnder(nil)
	if _, ok := win.glassUnderBox(platform.FrameInsets{}); ok {
		t.Error("clearing it left the glass confined")
	}
}

// The region follows the component rather than being copied once: a
// window that is resized must not leave the glass where the sidebar
// used to be.
func TestTheConfinedGlassFollowsTheLayout(t *testing.T) {
	_, win, side := glassWindow(t)
	win.SetGlassUnder(side)
	win.Capture()
	before, ok := win.glassUnderBox(platform.FrameInsets{})
	if !ok {
		t.Fatal("the glass is not confined to the sidebar")
	}

	win.dispatch(platform.Event{Kind: platform.EventResize, Width: 800, Height: 300})
	win.Capture()
	after, ok := win.glassUnderBox(platform.FrameInsets{})
	if !ok {
		t.Fatal("after the resize the glass is not confined to anything")
	}
	if after.W <= before.W {
		t.Errorf("the window doubled in width and the glass went from %d to %d: it is not following the layout",
			before.W, after.W)
	}
}

// The frame margin is part of the surface, so the box is stated inside
// it: a window with a toolkit-drawn border would otherwise blur a strip
// of its own frame.
func TestTheConfinedGlassSitsInsideTheFrame(t *testing.T) {
	_, win, side := glassWindow(t)
	win.SetGlassUnder(side)
	win.Capture()
	plain, _ := win.glassUnderBox(platform.FrameInsets{})
	inset, _ := win.glassUnderBox(platform.FrameInsets{Left: 7, Top: 9})
	if inset.X != plain.X+7 || inset.Y != plain.Y+9 {
		t.Errorf("under a 7x9 margin the box is at (%d,%d), want (%d,%d)",
			inset.X, inset.Y, plain.X+7, plain.Y+9)
	}
}
