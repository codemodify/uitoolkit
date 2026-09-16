package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestDragActionHasAndOne(t *testing.T) {
	both := DragCopy | DragMove
	if !both.Has(DragCopy) || !both.Has(DragMove) || both.Has(DragLink) {
		t.Fatalf("Has: %v", both)
	}
	if DragNone.Has(DragNone) {
		t.Fatal("nothing has DragNone")
	}
	if got := both.One(); got != DragCopy {
		t.Fatalf("One prefers copying: got %v", got)
	}
	if got := (DragMove | DragLink).One(); got != DragMove {
		t.Fatalf("One: got %v", got)
	}
	if got := DragNone.One(); got != DragNone {
		t.Fatalf("One of nothing: got %v", got)
	}
}

func TestDragActionString(t *testing.T) {
	for _, tc := range []struct {
		a    DragAction
		want string
	}{
		{DragCopy, "copy"},
		{DragMove, "move"},
		{DragLink, "link"},
		{DragNone, "none"},
		{DragMove | DragLink, "move"},
	} {
		if got := tc.a.String(); got != tc.want {
			t.Fatalf("%d: got %q want %q", tc.a, got, tc.want)
		}
	}
}

func TestNegotiateDragAction(t *testing.T) {
	all := DragCopy | DragMove | DragLink
	for _, tc := range []struct {
		name                        string
		offered, preferred, allowed DragAction
		want                        DragAction
	}{
		{"the source's preference wins", all, DragMove, all, DragMove},
		{"a preference the target refuses falls back", all, DragMove, DragCopy, DragCopy},
		{"no preference takes the best of both", all, DragNone, DragMove | DragLink, DragMove},
		{"nothing in common refuses", DragMove, DragMove, DragCopy, DragNone},
		{"a source offering nothing refuses", DragNone, DragNone, all, DragNone},
		{"a target allowing nothing refuses", all, DragCopy, DragNone, DragNone},
		{"one action both sides allow", DragCopy, DragNone, all, DragCopy},
		// A preference outside the offered set is meaningless; the
		// negotiation still has to land on something both allow.
		{"a preference not offered is ignored", DragCopy, DragLink, all, DragCopy},
	} {
		got := NegotiateDragAction(tc.offered, tc.preferred, tc.allowed)
		if got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestOffscreenCarriesAToplevel(t *testing.T) {
	src := NewOffscreen(WindowOptions{Width: 200, Height: 100})
	torn := NewOffscreen(WindowOptions{Width: 120, Height: 80})
	if !DragsToplevels(src) {
		t.Fatal("the offscreen desktop carries windows")
	}
	if AttachToplevel(src, torn, 10, 4) {
		t.Fatal("nothing to attach to before the drag starts")
	}
	if !src.StartDrag(DragPayload{Types: []string{"text/plain"}, Toplevel: true}) {
		t.Fatal("StartDrag")
	}
	if !AttachToplevel(src, torn, 10, 4) {
		t.Fatal("AttachToplevel")
	}
	if got, dx, dy := src.DragToplevel(); got != Surface(torn) || dx != 10 || dy != 4 {
		t.Fatalf("DragToplevel: got %v %d,%d", got, dx, dy)
	}
	src.Move(40, 30)
	src.SimulateDragOver(paintengine2d.Pt(60, 20))
	// The window hangs off the pointer by the offset, in root
	// coordinates: the source window's own position plus the pointer.
	if x, y, ok := torn.Position(); !ok || x != 40+60-10 || y != 30+20-4 {
		t.Fatalf("carried to %d,%d (ok=%v)", x, y, ok)
	}
	src.SimulateDragRelease()
	if !src.DragDropped() || src.DragEnded() != DragNone {
		t.Fatalf("a release over nothing is a drop that took nothing: dropped=%v action=%v",
			src.DragDropped(), src.DragEnded())
	}
}

func TestOffscreenWithoutToplevelDrag(t *testing.T) {
	src := NewOffscreen(WindowOptions{Width: 200, Height: 100})
	torn := NewOffscreen(WindowOptions{Width: 120, Height: 80})
	src.SetDragsToplevels(false)
	if DragsToplevels(src) {
		t.Fatal("this desktop has no toplevel drag")
	}
	if !src.StartDrag(DragPayload{Types: []string{"text/plain"}, Toplevel: true}) {
		t.Fatal("StartDrag")
	}
	if AttachToplevel(src, torn, 0, 0) {
		t.Fatal("a desktop without the protocol carries nothing")
	}
	src.CancelDrag()
	if src.DragDropped() {
		t.Fatal("a cancelled drag was never dropped")
	}
}

func TestSurfacePosition(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 100, Height: 100, X: 12, Y: 34})
	if x, y, ok := SurfacePosition(o); !ok || x != 12 || y != 34 {
		t.Fatalf("the window was opened at 12,34: got %d,%d (ok=%v)", x, y, ok)
	}
	MoveSurface(o, 200, 100)
	if x, y, _ := SurfacePosition(o); x != 200 || y != 100 {
		t.Fatalf("after a move: got %d,%d", x, y)
	}
	// A surface with no position at all — every Wayland toplevel — says
	// so rather than making one up.
	if _, _, ok := SurfacePosition(noPosition{o}); ok {
		t.Fatal("a surface that is not told its position must answer false")
	}
}

// noPosition is a Surface without the position capability, which is what
// a Wayland toplevel is.
type noPosition struct{ Surface }
