//go:build linux && cgo

package app

import (
	"os"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Two things a client-decorated X11 window got wrong about its own geometry.
// Both need a real X server — the frame margin is what they are about, and an
// offscreen window has none — so they run under tools/test-display.sh.

func framedWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		t.Skip("a Wayland toplevel has no position of its own")
	}
	t.Setenv(platform.EnvDecorations, "")
	// A look whose client frame keeps a shadow: the margin is what both of
	// these are about, and a look without one has none to get wrong.
	look := style.DarkLook()
	if pack, ok := style.LoadTheme("breeze"); ok {
		look = pack.Look()
	}
	a := New(Options{Look: look})
	a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "geometry", Width: 520, Height: 340,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Skip("no X11 window: ", err)
	}
	t.Cleanup(func() { w.Close(); a.Quit() })
	w.SetContent(widgets.NewColumn(widgets.NewLabel("geometry")))
	// The caption is what makes the frame the toolkit's, and so what gives
	// the surface a margin at all.
	w.SetTitleBar(widgets.NewHeaderBar(nil, nil, []widget.Component{widgets.NewLabel("geometry")}))
	return a, w
}

// A position asked for before the first layout is where the window ends up.
//
// Move subtracts the frame's margin and Position adds it, and the margin
// arrives with the first layout — the title bar being set after the window is
// made. So a Move made before it was read back with a margin it was never
// written with, and the window the user sees stood 24 logical pixels across
// and 12 down from where it was put, under Breeze at 1.75. An application had
// to repeat an unchanged request to make it take.
func TestAPositionAskedForBeforeLayoutSurvivesTheFrame(t *testing.T) {
	a, w := framedWindow(t)
	const wantX, wantY = 120, 75

	w.Move(wantX, wantY)
	// The first layout is what publishes the margin.
	a.PumpOnce()
	a.PumpOnce()

	x, y, ok := w.Position()
	if !ok {
		t.Skip("the server does not say where the window is")
	}
	if w.geom.margin.Zero() {
		t.Skip("this look gives the window no shadow: there is no margin to lose")
	}
	if x != wantX || y != wantY {
		t.Errorf("the window is at (%d,%d), not the (%d,%d) it was moved to", x, y, wantX, wantY)
	}
}

// A resize the window manager grants exactly still reaches the layout.
//
// X11's Resize sizes the backing image at once and then asks the server, and
// the configure that comes back becomes an EventResize only where the native
// size differs from that image — so a request granted precisely was silent.
// The cached geometry stayed with it: a window asked for 700x450 went on
// reporting 520x340, and every hit test with it.
func TestAnExactlyGrantedResizeReachesTheLayout(t *testing.T) {
	a, w := framedWindow(t)
	a.PumpOnce()
	const wantW, wantH = 700, 450

	w.SetSize(wantW, wantH)
	a.PumpOnce()
	a.PumpOnce()

	gw, gh := w.Size()
	if gw != wantW || gh != wantH {
		t.Errorf("the window reports %dx%d after being sized to %dx%d", gw, gh, wantW, wantH)
	}
	// And the rectangle widgets are laid out in agrees.
	if box := w.WindowRect(); int(box.Dx()) < wantW || int(box.Dy()) < wantH {
		t.Errorf("the window rect is %v, smaller than the %dx%d asked for", box, wantW, wantH)
	}
}
