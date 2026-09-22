//go:build linux && cgo

package platform

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// gestureEvents is the gestures among evs.
func gestureEvents(evs []Event) []Event {
	var out []Event
	for _, ev := range evs {
		if ev.Kind == EventGesture {
			out = append(out, ev)
		}
	}
	return out
}

// A compositor's pinch, swipe and hold (zwp_pointer_gestures_v1, read off
// the wire by the real client code) reach the window as EventGesture: at
// the pointer, with the fingers, the scale relative to the begin, the
// rotation and the movement, and the end or cancel the compositor said. A
// hold's begin stops a glide.
func TestWaylandGesturesOnTheWire(t *testing.T) {
	f := startWlFake(t, "wl_seat:5", "zwp_pointer_gestures_v1:3")
	s := fakeSurface(t)
	s.conn.roundtrip()
	s.conn.roundtrip()
	ptr, pinch, swipe, hold := f.first("wl_pointer"), f.first("zwp_pointer_gesture_pinch_v1"),
		f.first("zwp_pointer_gesture_swipe_v1"), f.first("zwp_pointer_gesture_hold_v1")
	if ptr == 0 || pinch == 0 || swipe == 0 || hold == 0 {
		t.Fatalf("pointer %d pinch %d swipe %d hold %d:\n%s", ptr, pinch, swipe, hold, strings.Join(f.requests(), "\n"))
	}
	surf := f.toplevel()
	f.emit(ptr, 0, uint32(1), surf, fixed(40), fixed(30)) // enter
	f.emit(ptr, 5)                                        // frame
	f.emit(pinch, 0, uint32(2), uint32(100), surf, uint32(2))
	f.emit(pinch, 1, uint32(110), fixed(3), fixed(-1), fixed(1.5), fixed(10))
	f.emit(pinch, 1, uint32(120), fixed(0), fixed(0), fixed(2), fixed(-4))
	f.emit(pinch, 2, uint32(3), uint32(130), uint32(0))
	f.emit(swipe, 0, uint32(4), uint32(140), surf, uint32(3))
	f.emit(swipe, 1, uint32(150), fixed(-25), fixed(2))
	f.emit(swipe, 2, uint32(5), uint32(160), uint32(1)) // cancelled
	s.conn.kin.sample(s.conn.kin.last, 0, 1)
	s.conn.kin.running = true
	f.emit(hold, 0, uint32(6), uint32(170), surf, uint32(2))
	f.emit(hold, 1, uint32(7), uint32(180), uint32(0))
	s.conn.roundtrip()
	if s.conn.kin.running {
		t.Fatal("fingers resting on the pad did not stop the glide")
	}
	got := gestureEvents(s.Poll())
	at := paintengine2d.Pt(40, 30)
	want := []Event{
		{Gesture: GesturePinch, Phase: GestureBegin, Fingers: 2, Scale: 1},
		{Gesture: GesturePinch, Phase: GestureUpdate, Fingers: 2, Scale: 1.5, Rotation: 10, Delta: paintengine2d.Pt(3, -1)},
		{Gesture: GesturePinch, Phase: GestureUpdate, Fingers: 2, Scale: 2, Rotation: -4},
		{Gesture: GesturePinch, Phase: GestureEnd, Fingers: 2, Scale: 2},
		{Gesture: GestureSwipe, Phase: GestureBegin, Fingers: 3, Scale: 1},
		{Gesture: GestureSwipe, Phase: GestureUpdate, Fingers: 3, Scale: 1, Delta: paintengine2d.Pt(-25, 2)},
		{Gesture: GestureSwipe, Phase: GestureCancel, Fingers: 3, Scale: 1},
		{Gesture: GestureHold, Phase: GestureBegin, Fingers: 2, Scale: 1},
		{Gesture: GestureHold, Phase: GestureEnd, Fingers: 2, Scale: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("%d gesture events, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Gesture != w.Gesture || g.Phase != w.Phase || g.Fingers != w.Fingers || g.Scale != w.Scale ||
			g.Rotation != w.Rotation || g.Delta != w.Delta || g.Pos != at {
			t.Fatalf("event %d: %v %v fingers %d scale %v rot %v delta %v at %v; want %+v",
				i, g.Gesture, g.Phase, g.Fingers, g.Scale, g.Rotation, g.Delta, g.Pos, w)
		}
	}
}

// Without the global there are no gesture objects, and nothing breaks.
func TestWaylandNoGesturesWithoutTheGlobal(t *testing.T) {
	f := startWlFake(t, "wl_seat:5")
	s := fakeSurface(t)
	s.conn.roundtrip()
	s.conn.roundtrip()
	if f.first("wl_pointer") == 0 {
		t.Fatal("no pointer")
	}
	if s.conn.gest.objs[0] != nil || f.first("zwp_pointer_gesture_pinch_v1") != 0 {
		t.Fatal("gesture objects without the global")
	}
}

// PortalParent exports the toplevel with xdg-foreign and names it by the
// compositor's handle, "wayland:<handle>" — once for as long as the window
// keeps its role; a hidden window names nothing, and the export goes with
// the role.
func TestWaylandPortalParentExportsTheToplevel(t *testing.T) {
	f := startWlFake(t, "zxdg_exporter_v2:1")
	s := fakeSurface(t)
	if got := s.PortalParent(); got != "wayland:handle-1" {
		t.Fatalf("PortalParent %q, want wayland:handle-1\n%s", got, strings.Join(f.requests(), "\n"))
	}
	if got := s.PortalParent(); got != "wayland:handle-1" {
		t.Fatalf("asked again: %q", got)
	}
	s.Hide()
	if got := s.PortalParent(); got != "" {
		t.Fatalf("a hidden window named itself %q", got)
	}
	s.conn.roundtrip()
	log := f.requests()
	if !inOrder(log, "bind zxdg_exporter_v2", "export_toplevel on a wl_surface", "exported.destroy") {
		t.Fatalf("requests:\n%s", strings.Join(log, "\n"))
	}
	if n := strings.Count(strings.Join(log, "\n"), "export_toplevel"); n != 1 {
		t.Fatalf("exported %d times, want once", n)
	}
}

// A compositor without xdg-foreign: no parent, and the dialog opens on its
// own as before.
func TestWaylandPortalParentNeedsTheExporter(t *testing.T) {
	startWlFake(t)
	s := fakeSurface(t)
	if got := s.PortalParent(); got != "" {
		t.Fatalf("PortalParent %q with no exporter", got)
	}
	if wlPortalParent("") != "" || wlPortalParent("abc") != "wayland:abc" {
		t.Fatal("wlPortalParent")
	}
}
