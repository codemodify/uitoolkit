package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// **A framed window can be a crisp rectangle.** Where the toolkit draws
// the frame, the look drops a soft shadow round the window into a
// transparent margin — the surface is bigger than the window it shows —
// and nothing an application said could turn it off: a window that wanted
// crisp edges had to give up the toolkit's frame, buttons and resizing
// altogether, or wear the shadow.
func TestAFramedWindowCanDropItsShadow(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	if !r.w.geom.framed {
		t.Fatal("the rig is not drawing its own frame")
	}
	if r.w.geom.margin.Zero() {
		t.Skip("this look drops no shadow, so there is none to turn off")
	}
	if !r.w.FrameShadow() {
		t.Error("a window starts without the look's shadow")
	}
	withShadow := r.w.geom

	r.w.SetFrameShadow(false)
	r.a.PumpOnce()
	r.w.frame()
	g := r.w.geom

	// The margin is gone: the surface is the window.
	if !g.margin.Zero() {
		t.Errorf("the margin is still %v", g.margin)
	}
	if g.shadow {
		t.Error("the frame still says it has a shadow")
	}
	ww, hh := r.w.surf.Size()
	if g.window != (paintengine2d.XYWH(0, 0, float32(ww), float32(hh))) {
		t.Errorf("the window is %v inside a %dx%d surface, want the whole of it", g.window, ww, hh)
	}
	if style.DecorationOf(r.w.look, r.w.caption.DecorationState()).Shadow != (style.Insets{}) {
		t.Error("the look is still asked for a shadow")
	}

	// Everything else stays: the border, the caption and its buttons.
	if g.border != withShadow.border {
		t.Errorf("the border changed from %v to %v", withShadow.border, g.border)
	}
	if g.caption.Dy() != withShadow.caption.Dy() {
		t.Errorf("the caption is %g tall, was %g", g.caption.Dy(), withShadow.caption.Dy())
	}
	if r.w.Caption() == nil || !r.w.Caption().Framed() {
		t.Error("the window lost its toolkit caption")
	}
	// And it still resizes: the band is inside the window's edge, which
	// is where it already is under a look that drops no shadow.
	if region, edges := r.w.NonClientHit(paintengine2d.Pt(float32(ww)-1, float32(hh)/2)); region != RegionResize || edges&platform.EdgeRight == 0 {
		t.Errorf("the right edge is %v/%v, want a resize edge", region, edges)
	}

	// And back again.
	r.w.SetFrameShadow(true)
	r.a.PumpOnce()
	r.w.frame()
	if r.w.geom.margin.Zero() {
		t.Error("the shadow did not come back")
	}
}
