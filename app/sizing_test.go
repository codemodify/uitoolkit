package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// fixedRig is a window whose size is its design: 275 by 116, the compact
// player's strip, with a frame the toolkit draws.
func fixedRig(t *testing.T) (*Application, *Window, *platform.Offscreen) {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Strip", Width: 275, Height: 116, Headless: true,
		Sizing: platform.SizingFixed, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewColumn(widgets.NewLabel("strip")))
	a.PumpOnce()
	return a, w, w.Surface().(*platform.Offscreen)
}

// TestFixedWindowHasNoResizeBand is the defect the rig found: a press eight
// pixels below a fixed 275×116 window's top edge landed in the frame's
// resize band and grew the window to 275×308. A fixed window's edges are
// the app's, and every way of resizing it is refused.
func TestFixedWindowHasNoResizeBand(t *testing.T) {
	_, w, o := fixedRig(t)
	if w.Resizable() {
		t.Fatal("a window opened SizingFixed is not resizable")
	}
	win := w.WindowRect()
	// The four edges and a corner, inside and out: the band a shadowed
	// frame keeps in its margin and the one a flat frame keeps inside.
	for _, p := range []paintengine2d.Point{
		{X: win.Min.X - 2, Y: win.Min.Y - 2},
		{X: (win.Min.X + win.Max.X) / 2, Y: win.Min.Y - 2},
		{X: (win.Min.X + win.Max.X) / 2, Y: win.Min.Y + 8},
		{X: win.Min.X - 2, Y: (win.Min.Y + win.Max.Y) / 2},
		{X: win.Max.X + 1, Y: (win.Min.Y + win.Max.Y) / 2},
		{X: (win.Min.X + win.Max.X) / 2, Y: win.Max.Y + 1},
		{X: win.Max.X + 1, Y: win.Max.Y + 1},
	} {
		if got, e := w.NonClientHit(p); got == RegionResize {
			t.Errorf("at %v the frame offers a resize band (%v)", p, e)
		}
	}
	// A press on the band does not hand the desktop a resize either.
	w.dispatch(platform.Event{Kind: platform.EventMouseDown,
		Pos: paintengine2d.Pt(win.Max.X+1, win.Max.Y+1), Button: platform.ButtonLeft})
	w.dispatch(platform.Event{Kind: platform.EventMouseUp,
		Pos: paintengine2d.Pt(win.Max.X+1, win.Max.Y+1), Button: platform.ButtonLeft})
	if n := len(o.FrameCalls().Resizes); n != 0 {
		t.Errorf("a fixed window started %d desktop resizes", n)
	}
	if w.StartResize(platform.EdgeBottom | platform.EdgeRight) {
		t.Error("StartResize succeeded on a fixed window")
	}
	// And the window system was told, which is what makes the desktop's
	// own frame refuse: minimum and maximum are the window's size.
	l := platform.SurfaceSizeLimits(w.Surface())
	if !l.Fixed() || l.MinWidth != 275 || l.MinHeight != 116 {
		t.Errorf("limits %+v, want 275x116 on both ends", l)
	}
	// Maximizing is a resize: the action and its caption button both go.
	if w.FrameCaps().Can(platform.CapMaximize) {
		t.Error("a fixed window offers maximize")
	}
	w.ToggleMaximize()
	if n := len(o.FrameCalls().Maximizes); n != 0 {
		t.Errorf("a fixed window asked to be maximized %d times", n)
	}
	left, right := w.Caption().Controls()
	for _, c := range []*widgets.WindowControls{left, right} {
		for _, b := range c.Shown() {
			if b == platform.CaptionMaximize {
				t.Error("the caption draws a maximize button on a fixed window")
			}
		}
	}
}

// TestFixedWindowPinFollowsTheApp: the application may resize its own fixed
// window (a player that folds), and the pin goes with it.
func TestFixedWindowPinFollowsTheApp(t *testing.T) {
	a, w, _ := fixedRig(t)
	w.SetSize(468, 104)
	a.PumpOnce()
	if got, want := platform.SurfaceSizeLimits(w.Surface()), (platform.SizeLimits{
		MinWidth: 468, MinHeight: 104, MaxWidth: 468, MaxHeight: 104}); got != want {
		t.Errorf("limits %+v, want %+v", got, want)
	}
	if ww, hh := w.Size(); ww != 468 || hh != 104 {
		t.Errorf("window %dx%d, want 468x104", ww, hh)
	}
}

// TestSetResizableUnpinsTheWindow: the policy is not frozen at open.
func TestSetResizableUnpinsTheWindow(t *testing.T) {
	a, w, _ := fixedRig(t)
	if !w.SetResizable(true) {
		t.Fatal("SetResizable(true) refused")
	}
	a.PumpOnce()
	if !w.Resizable() {
		t.Fatal("still fixed after SetResizable(true)")
	}
	if l := platform.SurfaceSizeLimits(w.Surface()); l.Fixed() {
		t.Errorf("limits still pinned: %+v", l)
	}
	win := w.WindowRect()
	if got, _ := w.NonClientHit(paintengine2d.Pt(win.Max.X+1, win.Max.Y+1)); got != RegionResize {
		t.Errorf("no resize band after SetResizable(true): %v", got)
	}
	if !w.FrameCaps().Can(platform.CapMaximize) {
		t.Error("maximize still refused after SetResizable(true)")
	}
}

// TestResizableWindowKeepsItsBand: the default is unchanged — a window that
// says nothing about sizing resizes as it always did.
func TestResizableWindowKeepsItsBand(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	if !r.w.Resizable() {
		t.Fatal("a window that asked for nothing is resizable")
	}
	if l := platform.SurfaceSizeLimits(r.w.Surface()); l.Fixed() {
		t.Errorf("an ordinary window is pinned: %+v", l)
	}
	win := r.win()
	if got, e := r.w.NonClientHit(paintengine2d.Pt(win.Max.X+1, win.Max.Y+1)); got != RegionResize ||
		e != platform.EdgeBottom|platform.EdgeRight {
		t.Errorf("bottom-right corner: %v %v", got, e)
	}
}
