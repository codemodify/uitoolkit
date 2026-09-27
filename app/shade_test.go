package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Rolling the window up leaves its title bar, pins the height so the desktop
// cannot push it open again, and rolling it back down gives it exactly the
// height it went up with.
func TestShadeRollsTheWindowUpAndBack(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	w0, h0 := r.w.Size()
	if !r.w.CanShade() {
		t.Fatal("a window with a frame of the toolkit's can roll up")
	}
	if !r.w.SetShaded(true) {
		t.Fatal("SetShaded(true)")
	}
	r.a.PumpOnce()
	if !r.w.Shaded() {
		t.Fatal("not rolled up")
	}
	sw, sh := r.w.Size()
	if sw != w0 {
		t.Errorf("width changed: %d, was %d", sw, w0)
	}
	if sh >= h0 {
		t.Fatalf("height %d did not come down from %d", sh, h0)
	}
	// What is left is the caption and the look's border, nothing more.
	capH := r.w.shadedHeight()
	if sh != capH {
		t.Errorf("rolled up to %d, the caption is %d", sh, capH)
	}
	// The pin is on the wire: a window manager clamps a resize to the
	// stated minimum, so without it the window would spring open.
	if l := platform.SurfaceSizeLimits(r.w.Surface()); l.MinHeight != sh || l.MaxHeight != sh {
		t.Errorf("height not pinned while rolled up: %+v", l)
	}
	// Rolled up, there is no vertical resize to be had.
	if e := r.w.resizeEdgesAt(r.edge(platform.EdgeBottom, r.win().Min.X+40)); e&platform.EdgeBottom != 0 {
		t.Error("a rolled-up window offers a bottom resize edge")
	}
	if !r.w.SetShaded(false) {
		t.Fatal("SetShaded(false)")
	}
	r.a.PumpOnce()
	if r.w.Shaded() {
		t.Fatal("still rolled up")
	}
	if _, h := r.w.Size(); h != h0 {
		t.Errorf("came back to %d, went up from %d", h, h0)
	}
	if l := platform.SurfaceSizeLimits(r.w.Surface()); l.MaxHeight != 0 {
		t.Errorf("the pin outlived the roll-up: %+v", l)
	}
}

// A resize while the window is rolled up changes its width and leaves the
// height it will come back to alone.
func TestShadeSurvivesAResizeWhileRolledUp(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	_, h0 := r.w.Size()
	r.w.SetShaded(true)
	r.a.PumpOnce()
	sw, sh := r.w.Size()
	r.w.dispatch(platform.Event{Kind: platform.EventResize, Width: sw + 120, Height: sh})
	r.a.PumpOnce()
	r.w.SetShaded(false)
	r.a.PumpOnce()
	if _, h := r.w.Size(); h != h0 {
		t.Errorf("came back to %d, went up from %d", h, h0)
	}
}

// A window can never be left rolled up with no title bar to roll it back
// down with: handing the frame to the desktop, or being maximized or put
// full screen, gives the window its height back.
func TestShadeIsUndoneWhenItCannotBeUndone(t *testing.T) {
	for _, tc := range []struct {
		name string
		do   func(r *frameRig)
	}{
		{"the desktop takes the frame back", func(r *frameRig) {
			r.o.SimulateDecorations(platform.DecorationsServer)
			r.a.PumpOnce()
		}},
		{"maximized", func(r *frameRig) {
			r.o.SetMaximized(true)
			r.a.PumpOnce()
		}},
		{"full screen", func(r *frameRig) {
			r.o.SetFullscreen(true)
			r.a.PumpOnce()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newFrameRig(t, platform.DecorationsClient)
			_, h0 := r.w.Size()
			if !r.w.SetShaded(true) {
				t.Fatal("did not roll up")
			}
			r.a.PumpOnce()
			tc.do(r)
			if r.w.Shaded() {
				t.Fatal("left rolled up with no way back")
			}
			if _, h := r.w.Size(); h != h0 {
				t.Errorf("height %d, was %d", h, h0)
			}
			if l := platform.SurfaceSizeLimits(r.w.Surface()); l.MaxHeight != 0 {
				t.Errorf("the pin was left on: %+v", l)
			}
		})
	}
}

// A window whose frame is the desktop's does not roll up: the wheel over
// that title bar never reaches the client at all, and the desktop's own
// setting governs it.
func TestShadeNeedsTheToolkitsOwnFrame(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsServer)
	if r.w.CanShade() {
		t.Error("claimed to roll up under the desktop's frame")
	}
	if r.w.SetShaded(true) || r.w.Shaded() {
		t.Error("rolled up under the desktop's frame")
	}
}

// The wheel over caption space rolls the window up and down; the wheel over
// what the app put in its title bar stays the app's.
func TestWheelOverTheCaptionRollsTheWindow(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	_, h0 := r.w.Size()
	wheel := func(at paintengine2d.Point, dy float32) {
		r.w.dispatch(platform.Event{Kind: platform.EventScroll, Pos: at, Scroll: paintengine2d.Pt(0, dy)})
		r.a.PumpOnce()
	}
	free := r.freeSpace()
	if reg, _ := r.w.NonClientHit(free); reg != RegionCaption {
		t.Fatalf("the test point is %v, not caption", reg)
	}
	wheel(free, -1) // up
	if !r.w.Shaded() {
		t.Fatal("the wheel up over the caption did not roll the window up")
	}
	wheel(free, 1) // down
	if r.w.Shaded() {
		t.Fatal("the wheel down did not roll it back")
	}
	if _, h := r.w.Size(); h != h0 {
		t.Errorf("height %d, was %d", h, h0)
	}
	// The app's own title-bar items keep their wheel.
	r.clicks = 0
	wheel(center(r.tools), -1)
	if r.w.Shaded() {
		t.Error("a wheel over the app's title-bar tool bar rolled the window up")
	}
	// And a desktop that says the wheel does nothing is obeyed.
	p := r.a.TitleBarPrefs()
	p.Wheel = platform.TitleWheelNone
	r.a.SetTitleBarPrefs(p)
	wheel(free, -1)
	if r.w.Shaded() {
		t.Error("rolled up although the desktop's wheel action is none")
	}
}

// The keep-above button stands where the window-menu button would, but only
// where the desktop can actually keep a window above the others — nobody
// trades a working button for a dead one.
func TestKeepAboveButtonTakesTheMenuSlot(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	r.a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs("KDE"))
	r.a.PumpOnce()
	shown := func() []platform.CaptionButton {
		l, rr := r.w.Caption().Controls()
		return append(append([]platform.CaptionButton(nil), l.Shown()...), rr.Shown()...)
	}
	has := func(bs []platform.CaptionButton, b platform.CaptionButton) bool {
		for _, x := range bs {
			if x == b {
				return true
			}
		}
		return false
	}
	// A desktop that cannot: the window-menu button stays exactly as it was.
	if bs := shown(); !has(bs, platform.CaptionMenu) || has(bs, platform.CaptionKeepAbove) {
		t.Fatalf("without keep-above: %v", bs)
	}
	r.o.SimulateKeepAbove(true)
	r.w.rebuildCaption()
	r.a.PumpOnce()
	bs := shown()
	if has(bs, platform.CaptionMenu) || !has(bs, platform.CaptionKeepAbove) {
		t.Fatalf("with keep-above: %v", bs)
	}
}

// Pressing the button toggles the state, and it is a toggle to a screen
// reader: named, checkable, and checked while it is on.
func TestKeepAboveButtonTogglesAndSaysSo(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	r.o.SimulateKeepAbove(true)
	r.a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs("KDE"))
	r.a.PumpOnce()
	var ctl *widgets.WindowControls
	l, rr := r.w.Caption().Controls()
	for _, c := range []*widgets.WindowControls{l, rr} {
		if !c.ButtonRect(platform.CaptionKeepAbove).Empty() {
			ctl = c
		}
	}
	if ctl == nil {
		t.Fatal("no keep-above button")
	}
	node := func() *a11y.Node {
		for _, n := range ctl.AccessibleItems() {
			if n.Name == "Keep Above Others" || n.Name == "Stop Keeping Above Others" {
				return n
			}
		}
		return nil
	}
	n := node()
	if n == nil {
		t.Fatal("the keep-above button is not in the accessibility tree")
	}
	if n.Role != a11y.RoleToggleButton || n.State&a11y.StateCheckable == 0 {
		t.Errorf("role %v state %v: it is a toggle", n.Role, n.State)
	}
	if n.State&a11y.StateChecked != 0 {
		t.Error("checked before it was pressed")
	}
	at := widget.DeviceOrigin(ctl).Add(ctl.ButtonRect(platform.CaptionKeepAbove).Center())
	r.click(at.X, at.Y)
	r.a.PumpOnce()
	if !r.w.KeepAbove() {
		t.Fatal("the button did not keep the window above")
	}
	if calls := r.o.FrameCalls().Aboves; len(calls) != 1 || !calls[0] {
		t.Errorf("requests %v", calls)
	}
	n = node()
	if n == nil || n.State&a11y.StateChecked == 0 || n.Name != "Stop Keeping Above Others" {
		t.Errorf("after the press: %+v", n)
	}
	r.click(at.X, at.Y)
	r.a.PumpOnce()
	if r.w.KeepAbove() {
		t.Error("the second press did not turn it off")
	}
}

// Where the desktop cannot keep a window above, nothing pretends it can.
func TestKeepAboveIsRefusedWhereItCannotWork(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	if r.w.CanKeepAbove() {
		t.Fatal("an offscreen window has no desktop to stack it")
	}
	if r.w.SetKeepAbove(true) || r.w.KeepAbove() {
		t.Error("kept above without a desktop that can")
	}
	if r.w.ToggleKeepAbove() {
		t.Error("toggled without a desktop that can")
	}
}
