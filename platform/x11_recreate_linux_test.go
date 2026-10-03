//go:build linux && cgo

package platform

import (
	"os"
	"testing"
)

// An X window's visual is fixed when it is created. A frame with a shadow
// needs an alpha channel, so setting one on a window that has not got a
// 32-bit visual destroys that window and makes another — and every property
// the window manager had been told about it belongs to the window, not to
// this process, so it goes too.
//
// Nothing else in the suite reaches an X server: tools/testenv.sh unsets
// DISPLAY, which is the right default for everything that does not need one
// and the reason this class of bug kept shipping. These run under the
// nested-compositor rig (tools/e2e/start.sh) and skip everywhere else.

// recreatableSurface is a surface whose window a frame can force onto
// another visual, or a skip saying why not.
func recreatableSurface(t *testing.T, opts WindowOptions) *x11Surface {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	opts.Width, opts.Height = 240, 160
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
	if !s.conn.composited {
		// Without a compositing manager there is no ARGB visual to move to
		// and the window is never re-created, so there is nothing here to
		// lose.
		t.Skip("no compositing manager: a window is never re-created")
	}
	return s
}

// shadowFrame needs an alpha channel, which is what moves the window.
var shadowFrame = Frame{Margin: FrameInsets{Left: 8, Right: 8, Top: 8, Bottom: 8}, Alpha: true}

// recreate gives the surface a frame with a shadow and insists the window
// really was made again — a test that checked properties on the *same*
// window would pass without proving anything.
func recreateWindow(t *testing.T, s *x11Surface) {
	t.Helper()
	was := s.win
	s.SetFrame(shadowFrame)
	if s.win == was {
		t.Fatalf("the window was not re-created (still %d): this test would prove nothing", uint64(was))
	}
}

// has reports whether the property holds the atom of this name.
func propHas(t *testing.T, s *x11Surface, prop, atom string) bool {
	t.Helper()
	want := s.conn.atomNamed(atom)
	if want == 0 {
		t.Skipf("the server has no %s", atom)
	}
	for _, a := range s.propAtoms(prop) {
		if a == want {
			return true
		}
	}
	return false
}

// A dialog is still a dialog after its window has been re-created.
//
// _NET_WM_WINDOW_TYPE went with the old window, so a dialog that grew a
// shadow stopped being a dialog to the window manager — no centring on its
// parent, no dialog frame, not kept above what it belongs to — and the
// application was never told, because from its own side nothing had changed.
func TestADialogKeepsItsRoleWhenItsWindowIsRecreated(t *testing.T) {
	s := recreatableSurface(t, WindowOptions{Title: "uitoolkit-role", Role: RoleDialog})
	if !propHas(t, s, "_NET_WM_WINDOW_TYPE", "_NET_WM_WINDOW_TYPE_DIALOG") {
		t.Fatal("the window was not a dialog before the frame was set")
	}
	recreateWindow(t, s)
	if !propHas(t, s, "_NET_WM_WINDOW_TYPE", "_NET_WM_WINDOW_TYPE_DIALOG") {
		t.Error("_NET_WM_WINDOW_TYPE_DIALOG is gone from the new window")
	}
}

// An ordinary window says what it is after its window has been re-created.
// The report that found this probed a plain window, not a dialog: the
// property is absent on the new window either way, and a window manager that
// reads it to decide placement has nothing to read.
func TestAnOrdinaryWindowKeepsItsTypeWhenItsWindowIsRecreated(t *testing.T) {
	s := recreatableSurface(t, WindowOptions{Title: "uitoolkit-role-normal"})
	recreateWindow(t, s)
	if !propHas(t, s, "_NET_WM_WINDOW_TYPE", "_NET_WM_WINDOW_TYPE_NORMAL") {
		t.Errorf("_NET_WM_WINDOW_TYPE holds %d atoms, not NORMAL", len(s.propAtoms("_NET_WM_WINDOW_TYPE")))
	}
}

// A maximized window comes back maximized, a full-screen one full-screen, and
// one kept above still kept above.
//
// The states are set here rather than asked of a window manager because the
// bug is in the handover: what a window says about itself before it is mapped
// is all a window manager has to go on, and a re-created window said nothing.
func TestARecreatedWindowKeepsTheStateItWasIn(t *testing.T) {
	for _, tc := range []struct {
		name string
		st   WindowState
		want []string
	}{
		{"maximized", WindowState{Maximized: true},
			[]string{"_NET_WM_STATE_MAXIMIZED_HORZ", "_NET_WM_STATE_MAXIMIZED_VERT"}},
		{"fullscreen", WindowState{Fullscreen: true},
			[]string{"_NET_WM_STATE_FULLSCREEN"}},
		{"kept above", WindowState{KeepAbove: true},
			[]string{"_NET_WM_STATE_ABOVE"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := recreatableSurface(t, WindowOptions{Title: "uitoolkit-state"})
			// What the window manager would have told us it did.
			x11Mu.Lock()
			s.state = tc.st
			x11Mu.Unlock()
			recreateWindow(t, s)
			for _, w := range tc.want {
				if !propHas(t, s, "_NET_WM_STATE", w) {
					t.Errorf("the new window lost %s", w)
				}
			}
		})
	}
}

// A window in no particular state claims none. Writing _NET_WM_STATE
// unconditionally would have a restored window telling the window manager it
// was maximized, which is a worse bug than the one being fixed.
func TestARecreatedWindowClaimsNoStateItWasNotIn(t *testing.T) {
	s := recreatableSurface(t, WindowOptions{Title: "uitoolkit-state-none"})
	x11Mu.Lock()
	s.state = WindowState{}
	x11Mu.Unlock()
	recreateWindow(t, s)
	if got := s.propAtoms("_NET_WM_STATE"); len(got) != 0 {
		t.Errorf("_NET_WM_STATE holds %d atoms on a window that was in no state", len(got))
	}
}

// A re-created window still takes drops. XdndAware is a property like the
// others and went the same way; it is re-applied, and until now nothing said
// so, because nothing in the suite reached an X server.
func TestARecreatedWindowStillTakesDrops(t *testing.T) {
	s := recreatableSurface(t, WindowOptions{Title: "uitoolkit-xdnd"})
	recreateWindow(t, s)
	if got := s.propAtoms("XdndAware"); len(got) != 1 || got[0] == 0 {
		t.Error("XdndAware is gone from the new window: it would take no drops")
	}
}
