//go:build linux && cgo

package platform

import (
	"os"
	"testing"
)

// A satellite panel's policy, read back from the X server.
//
// The report that asked for this had probed a toolkit window with xprop and
// found "type DIALOG, no skip-taskbar state, and no transient parent": the role
// was all the toolkit could say, so an application kept its own X11 adapter to
// set WM_TRANSIENT_FOR and the skip states, and to put them back whenever the
// native window changed underneath it. These read the same properties the
// report read.

func x11Window(t *testing.T, opts WindowOptions) *x11Surface {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	if opts.Width == 0 {
		opts.Width, opts.Height = 200, 140
	}
	b := x11Backend{}
	raw, err := b.NewSurface(opts)
	if err != nil {
		t.Skip("no X11 surface: ", err)
	}
	s, ok := raw.(*x11Surface)
	if !ok {
		t.Fatalf("not an X11 surface: %T", raw)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// transientFor is WM_TRANSIENT_FOR, or 0. It is a WINDOW-typed property, not
// an ATOM one: asking for the wrong type answers "nothing", which reads just
// like a property nobody wrote.
func transientFor(s *x11Surface) uint64 { return s.propWindow("WM_TRANSIENT_FOR") }

// A utility window says what kind of window it is, whose it is, and that it is
// not one to list — all three before it is mapped, which is when a window
// manager reads them.
func TestAUtilityWindowSaysItsTypeOwnerAndTaskbarPolicy(t *testing.T) {
	main := x11Window(t, WindowOptions{Title: "uitoolkit-main"})
	panel := x11Window(t, WindowOptions{
		Title: "uitoolkit-panel", Role: RoleUtility, Owner: main,
	})
	c := panel.conn

	wantType := c.atomNamed("_NET_WM_WINDOW_TYPE_UTILITY")
	if wantType == 0 {
		t.Skip("the server has no utility window type")
	}
	if got := panel.propAtoms("_NET_WM_WINDOW_TYPE"); len(got) != 1 || got[0] != wantType {
		t.Errorf("_NET_WM_WINDOW_TYPE holds %v, want the utility type", got)
	}
	if got := transientFor(panel); got != uint64(main.win) {
		t.Errorf("WM_TRANSIENT_FOR is %d, want the primary window %d", got, uint64(main.win))
	}
	state := panel.propAtoms("_NET_WM_STATE")
	for _, name := range []string{"_NET_WM_STATE_SKIP_TASKBAR", "_NET_WM_STATE_SKIP_PAGER"} {
		want := c.atomNamed(name)
		found := false
		for _, a := range state {
			found = found || a == want
		}
		if !found {
			t.Errorf("_NET_WM_STATE is missing %s", name)
		}
	}
}

// And an ordinary window claims none of it, which is what makes the test above
// about the role rather than about every window.
func TestAnOrdinaryWindowClaimsNoneOfIt(t *testing.T) {
	s := x11Window(t, WindowOptions{Title: "uitoolkit-ordinary"})
	if got := transientFor(s); got != 0 {
		t.Errorf("an ordinary window belongs to %d", got)
	}
	if got := s.propAtoms("_NET_WM_STATE"); len(got) != 0 {
		t.Errorf("an ordinary window claims %d states", len(got))
	}
	if want := s.conn.atomNamed("_NET_WM_WINDOW_TYPE_NORMAL"); want != 0 {
		got := s.propAtoms("_NET_WM_WINDOW_TYPE")
		if len(got) != 1 || got[0] != want {
			t.Errorf("_NET_WM_WINDOW_TYPE holds %v, want the normal type", got)
		}
	}
}

// An owner is given up as well as given, and the property goes rather than
// being set to the None window — which is a different thing and one a window
// manager reads as "belongs to the root".
func TestAnOwnerIsGivenUpByRemovingTheProperty(t *testing.T) {
	main := x11Window(t, WindowOptions{Title: "uitoolkit-main-2"})
	panel := x11Window(t, WindowOptions{Title: "uitoolkit-panel-2", Owner: main})
	if transientFor(panel) == 0 {
		t.Fatal("the owner did not take, so there is nothing to give up")
	}
	if !panel.SetOwner(nil) {
		t.Fatal("the owner could not be given up")
	}
	if got := transientFor(panel); got != 0 {
		t.Errorf("WM_TRANSIENT_FOR still holds %d", got)
	}
}

// A window of another backend, and the window itself, are owners no X11 window
// can have: a transient-for pointing at either is a lie the server would
// accept and the window manager would act on.
func TestAnImpossibleOwnerIsRefused(t *testing.T) {
	s := x11Window(t, WindowOptions{Title: "uitoolkit-refuse"})
	if s.SetOwner(s) {
		t.Error("a window was allowed to own itself")
	}
	off := NewOffscreen(WindowOptions{Width: 10, Height: 10})
	if s.SetOwner(off) {
		t.Error("an X11 window took an offscreen window as its owner")
	}
	if got := transientFor(s); got != 0 {
		t.Errorf("a refused owner was written anyway: %d", got)
	}
}

// A window that becomes a satellite panel after it was made leaves the list,
// and one that stops being a panel comes back. The window is mapped by then,
// so the states go as a client message and not as a property — which is the
// mistake that makes a state change look like it took and do nothing.
func TestBecomingAPanelLeavesTheListAndCeasingToBePutsItBack(t *testing.T) {
	s := x11Window(t, WindowOptions{Title: "uitoolkit-later"})
	if !s.FrameCaps().Has(FrameSkipTaskbar) {
		t.Skip("this window manager does not support the skip states")
	}
	s.Show()
	deadlineDrain(t, s)

	if !s.SetWindowRole(RoleUtility) {
		t.Fatal("the window could not become a panel")
	}
	if !s.task.skipping() {
		t.Error("a window that became a panel is still in the list")
	}
	if !s.SetWindowRole(RoleNormal) {
		t.Fatal("the window could not stop being a panel")
	}
	if s.task.skipping() {
		t.Error("a window that stopped being a panel is still out of the list")
	}
}

// deadlineDrain pumps the connection a few times so a map and its configure
// have arrived.
func deadlineDrain(t *testing.T, s *x11Surface) {
	t.Helper()
	for i := 0; i < 20; i++ {
		x11Mu.Lock()
		s.conn.drainLocked()
		x11Mu.Unlock()
	}
}

// A panel hidden and shown again is still out of the list.
//
// A window manager removes _NET_WM_STATE when a window is withdrawn, as the
// EWMH tells it to, so the states a panel claimed went with the first hide. The
// X window id does not change, so nothing on this side had changed and nothing
// noticed: the application that found this could only watch the window and ask
// again, once a second.
func TestAPanelHiddenAndShownAgainIsStillOutOfTheList(t *testing.T) {
	main := x11Window(t, WindowOptions{Title: "uitoolkit-main-3"})
	panel := x11Window(t, WindowOptions{
		Title: "uitoolkit-panel-3", Role: RoleUtility, Owner: main,
	})
	c := panel.conn
	want := c.atomNamed("_NET_WM_STATE_SKIP_TASKBAR")
	if want == 0 {
		t.Skip("the server has no skip-taskbar state")
	}
	has := func() bool {
		for _, a := range panel.propAtoms("_NET_WM_STATE") {
			if a == want {
				return true
			}
		}
		return false
	}
	panel.Show()
	deadlineDrain(t, panel)
	if !has() {
		t.Skip("this window manager does not keep the state on a mapped window")
	}
	was := panel.win

	panel.Hide()
	deadlineDrain(t, panel)
	panel.Show()
	deadlineDrain(t, panel)

	if panel.win != was {
		t.Fatalf("the window was re-created (%d then %d), which is a different bug", uint64(was), uint64(panel.win))
	}
	if !has() {
		t.Error("the panel came back in the window list it asked to be left out of")
	}
}

// And a re-created window — a frame that gains a shadow destroys the X window
// and makes another — keeps all three.
func TestARecreatedPanelKeepsItsTypeOwnerAndPolicy(t *testing.T) {
	main := x11Window(t, WindowOptions{Title: "uitoolkit-main-4"})
	panel := x11Window(t, WindowOptions{
		Title: "uitoolkit-panel-4", Role: RoleUtility, Owner: main,
	})
	if !panel.conn.composited {
		t.Skip("no compositing manager: a window is never re-created")
	}
	was := panel.win
	panel.SetFrame(Frame{Margin: FrameInsets{Left: 8, Right: 8, Top: 8, Bottom: 8}, Alpha: true})
	if panel.win == was {
		t.Fatal("the window was not re-created, so this proves nothing")
	}

	c := panel.conn
	if want := c.atomNamed("_NET_WM_WINDOW_TYPE_UTILITY"); want != 0 {
		got := panel.propAtoms("_NET_WM_WINDOW_TYPE")
		if len(got) != 1 || got[0] != want {
			t.Errorf("the new window's type is %v, not utility", got)
		}
	}
	if got := transientFor(panel); got != uint64(main.win) {
		t.Errorf("the new window belongs to %d, want the primary window %d", got, uint64(main.win))
	}
	want := c.atomNamed("_NET_WM_STATE_SKIP_TASKBAR")
	found := false
	for _, a := range panel.propAtoms("_NET_WM_STATE") {
		found = found || a == want
	}
	if want != 0 && !found {
		t.Error("the new window is in the window list the old one was out of")
	}
}
