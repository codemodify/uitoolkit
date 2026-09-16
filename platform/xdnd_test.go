package platform

import (
	"reflect"
	"testing"
)

func TestXdndEnterRoundTrip(t *testing.T) {
	types := []uint32{101, 102, 103}
	m := EncodeXdndEnter(0xdead, 5, types)
	if m[1]>>24 != 5 {
		t.Fatalf("version in the high byte: got %#x", m[1])
	}
	if m[1]&1 != 0 {
		t.Fatal("three types must not set the more bit")
	}
	src, version, more, got := DecodeXdndEnter(m)
	if src != 0xdead || version != 5 || more {
		t.Fatalf("got source %#x version %d more %v", src, version, more)
	}
	if !reflect.DeepEqual(got, types) {
		t.Fatalf("types: got %v want %v", got, types)
	}
}

func TestXdndEnterMoreThanThreeTypes(t *testing.T) {
	m := EncodeXdndEnter(7, 5, []uint32{1, 2, 3, 4, 5})
	_, _, more, got := DecodeXdndEnter(m)
	if !more {
		t.Fatal("more than three types must set bit 0 so the target reads XdndTypeList")
	}
	if !reflect.DeepEqual(got, []uint32{1, 2, 3}) {
		t.Fatalf("only the first three ride in the message: got %v", got)
	}
}

func TestXdndEnterUnusedSlotsAreNone(t *testing.T) {
	m := EncodeXdndEnter(7, 5, []uint32{42})
	if m[3] != 0 || m[4] != 0 {
		t.Fatalf("unused type slots must be None: %v", m)
	}
	_, _, more, got := DecodeXdndEnter(m)
	if more || !reflect.DeepEqual(got, []uint32{42}) {
		t.Fatalf("got more %v types %v", more, got)
	}
}

func TestXdndPositionRoundTrip(t *testing.T) {
	m := EncodeXdndPosition(9, 640, 480, 12345, 77)
	src, x, y, when, action := DecodeXdndPosition(m)
	if src != 9 || x != 640 || y != 480 || when != 12345 || action != 77 {
		t.Fatalf("got %d %d,%d %d %d", src, x, y, when, action)
	}
	if m[1] != 0 {
		t.Fatal("the reserved word must be zero")
	}
}

// A monitor left of or above the origin gives negative root coordinates;
// XDND packs two shorts in a word, so they have to survive as signed.
func TestXdndPositionNegativeRootCoordinates(t *testing.T) {
	m := EncodeXdndPosition(9, -300, -17, 1, 0)
	_, x, y, _, _ := DecodeXdndPosition(m)
	if x != -300 || y != -17 {
		t.Fatalf("got %d,%d want -300,-17", x, y)
	}
}

func TestXdndStatusRoundTrip(t *testing.T) {
	r := XDNDRect{X: 10, Y: 20, W: 300, H: 40}
	m := EncodeXdndStatus(0xbeef, true, r, 55)
	target, accept, wantPos, gotR, action := DecodeXdndStatus(m)
	if target != 0xbeef || !accept || gotR != r || action != 55 {
		t.Fatalf("got %#x %v %+v %d", target, accept, gotR, action)
	}
	if wantPos {
		t.Fatal("bit 1 was not set")
	}
}

// A refused drop must name no action: a source that saw one would show
// the user a drop cursor for a drop that will not happen.
func TestXdndStatusRefusedCarriesNoAction(t *testing.T) {
	m := EncodeXdndStatus(1, false, XDNDRect{}, 55)
	if m[4] != 0 {
		t.Fatalf("refused status must send action None, got %d", m[4])
	}
	if m[1]&1 != 0 {
		t.Fatal("refused status must clear the accept bit")
	}
	_, accept, _, _, action := DecodeXdndStatus(m)
	if accept || action != 0 {
		t.Fatalf("got accept %v action %d", accept, action)
	}
}

func TestXdndStatusWantPositionBit(t *testing.T) {
	m := EncodeXdndStatus(1, true, XDNDRect{}, 5)
	m[1] |= 2
	_, accept, wantPos, _, _ := DecodeXdndStatus(m)
	if !accept || !wantPos {
		t.Fatalf("got accept %v wantPos %v", accept, wantPos)
	}
}

func TestXdndDropAndLeaveRoundTrip(t *testing.T) {
	src, when := DecodeXdndDrop(EncodeXdndDrop(3, 999))
	if src != 3 || when != 999 {
		t.Fatalf("drop: got %d %d", src, when)
	}
	if got := DecodeXdndLeave(EncodeXdndLeave(3)); got != 3 {
		t.Fatalf("leave: got %d", got)
	}
}

func TestXdndFinishedRoundTrip(t *testing.T) {
	target, accepted, action := DecodeXdndFinished(EncodeXdndFinished(8, true, 44), 5)
	if target != 8 || !accepted || action != 44 {
		t.Fatalf("got %d %v %d", target, accepted, action)
	}
	if _, accepted, action = DecodeXdndFinished(EncodeXdndFinished(8, false, 44), 5); accepted || action != 0 {
		t.Fatalf("a refused drop reports nothing performed: got %v %d", accepted, action)
	}
}

// Before version 5 XdndFinished had no "accepted" bit. The spec tells the
// source to proceed as if it were set, whatever the bit actually holds.
func TestXdndFinishedBeforeVersion5CountsAsTaken(t *testing.T) {
	var m XDNDMessage
	m[0] = 8
	m[2] = 44
	_, accepted, action := DecodeXdndFinished(m, 4)
	if !accepted || action != 44 {
		t.Fatalf("v4 finished: got accepted %v action %d", accepted, action)
	}
}

func TestXdndNegotiateVersion(t *testing.T) {
	for _, tc := range []struct {
		theirs int
		want   int
		ok     bool
	}{
		{5, 5, true},
		{6, XDNDVersion, true}, // never speak above our own
		{3, 3, true},
		{2, 0, false}, // below the spec's floor
		{0, 0, false}, // no XdndAware at all
	} {
		got, ok := XDNDNegotiateVersion(XDNDVersion, tc.theirs)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("theirs %d: got %d %v want %d %v", tc.theirs, got, ok, tc.want, tc.ok)
		}
	}
}

func TestXDNDActionsMapping(t *testing.T) {
	tab := XDNDActions{Copy: 10, Move: 11, Link: 12, Ask: 13, Private: 14}
	for _, tc := range []struct {
		atom uint32
		want DragAction
	}{
		{10, DragCopy},
		{11, DragMove},
		{12, DragLink},
		{0, DragNone},
		// Ask and Private are not offered: the spec lets a target fall
		// back on copying rather than refuse.
		{13, DragCopy},
		{14, DragCopy},
		{999, DragCopy},
	} {
		if got := tab.Action(tc.atom); got != tc.want {
			t.Fatalf("atom %d: got %v want %v", tc.atom, got, tc.want)
		}
	}
	if got := tab.Atom(DragMove); got != 11 {
		t.Fatalf("Atom(move): got %d", got)
	}
	if got := tab.Atom(DragNone); got != 0 {
		t.Fatalf("Atom(none) must be None, got %d", got)
	}
}

func TestXDNDActionsList(t *testing.T) {
	tab := XDNDActions{Copy: 10, Move: 11, Link: 12}
	got := tab.List(DragCopy|DragMove|DragLink, DragMove)
	want := []uint32{11, 10, 12} // preferred first, then copy, move, link
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := tab.List(DragCopy, DragNone); !reflect.DeepEqual(got, []uint32{10}) {
		t.Fatalf("one action: got %v", got)
	}
}

// ---- the search for the window under the pointer --------------------------------

// fakeTree is an X window tree in a table: children bottom-most first,
// geometry in the parent's coordinates, XdndAware and XdndProxy.
type fakeTree struct {
	kids    map[uint32][]uint32
	geom    map[uint32][4]int
	unmap   map[uint32]bool
	aware   map[uint32]int
	proxies map[uint32]uint32
}

func (f fakeTree) Children(win uint32) []uint32 { return f.kids[win] }

func (f fakeTree) Geometry(win uint32) (x, y, w, h int, mapped bool) {
	g := f.geom[win]
	return g[0], g[1], g[2], g[3], !f.unmap[win]
}

func (f fakeTree) Aware(win uint32) (int, uint32) { return f.aware[win], f.proxies[win] }

// A reparenting window manager puts its frame between the root and the
// client, and the frame carries no XdndAware: the search has to descend
// into it rather than give up at the root's children.
func newFrameTree() fakeTree {
	return fakeTree{
		kids: map[uint32][]uint32{
			1:  {10, 20}, // root: window 10 below window 20
			10: {11},
			20: {21},
		},
		geom: map[uint32][4]int{
			10: {0, 0, 400, 300},     // frame
			11: {4, 24, 392, 272},    // client inside it
			20: {200, 100, 400, 300}, // a second frame, overlapping
			21: {4, 24, 392, 272},
		},
		aware: map[uint32]int{11: 5, 21: 5},
	}
}

func TestXDNDFindTargetDescendsThroughTheFrame(t *testing.T) {
	tree := newFrameTree()
	got := XDNDFindTarget(tree, 1, 100, 200, nil)
	if got.Window != 11 {
		t.Fatalf("got window %d want the client 11", got.Window)
	}
	if got.Version != 5 || got.Send() != 11 {
		t.Fatalf("got version %d send %d", got.Version, got.Send())
	}
}

func TestXDNDFindTargetTakesTheTopmostWindow(t *testing.T) {
	tree := newFrameTree()
	// (250,150) is inside both frames; window 20 is later in the child
	// list, so it is on top.
	got := XDNDFindTarget(tree, 1, 250, 150, nil)
	if got.Window != 21 {
		t.Fatalf("got window %d want the topmost client 21", got.Window)
	}
}

func TestXDNDFindTargetSkipsTheDragIcon(t *testing.T) {
	tree := newFrameTree()
	// An override-redirect icon window covering everything: without the
	// skip it would swallow every drop.
	tree.kids[1] = append(tree.kids[1], 99)
	tree.geom[99] = [4]int{0, 0, 2000, 2000}
	tree.aware[99] = 5
	if got := XDNDFindTarget(tree, 1, 100, 200, nil); got.Window != 99 {
		t.Fatalf("without the skip the icon is the target: got %d", got.Window)
	}
	got := XDNDFindTarget(tree, 1, 100, 200, func(w uint32) bool { return w == 99 })
	if got.Window != 11 {
		t.Fatalf("with the skip: got %d want 11", got.Window)
	}
}

func TestXDNDFindTargetIgnoresUnmappedWindows(t *testing.T) {
	tree := newFrameTree()
	tree.unmap = map[uint32]bool{20: true}
	if got := XDNDFindTarget(tree, 1, 250, 150, nil); got.Window != 11 {
		t.Fatalf("got %d want 11 (the mapped window under it)", got.Window)
	}
}

func TestXDNDFindTargetNothingThere(t *testing.T) {
	tree := newFrameTree()
	if got := XDNDFindTarget(tree, 1, 5000, 5000, nil); got.Valid() {
		t.Fatalf("got a target off every window: %+v", got)
	}
	// A window that takes no drops is not a target either.
	tree.aware = nil
	if got := XDNDFindTarget(tree, 1, 100, 200, nil); got.Valid() {
		t.Fatalf("got a target with no XdndAware: %+v", got)
	}
}

func TestXDNDFindTargetTooOldIsNoTarget(t *testing.T) {
	tree := newFrameTree()
	tree.aware[11] = 2 // below the spec's floor of 3
	if got := XDNDFindTarget(tree, 1, 100, 200, nil); got.Valid() {
		t.Fatalf("version 2 is not XDND: got %+v", got)
	}
}

func TestXDNDFindTargetSpeaksTheOlderVersion(t *testing.T) {
	tree := newFrameTree()
	tree.aware[11] = 3
	got := XDNDFindTarget(tree, 1, 100, 200, nil)
	if got.Window != 11 || got.Version != 3 {
		t.Fatalf("got window %d version %d want 11 at 3", got.Window, got.Version)
	}
}

func TestXDNDFindTargetFollowsTheProxy(t *testing.T) {
	tree := newFrameTree()
	tree.proxies = map[uint32]uint32{11: 77, 77: 77} // the proxy names itself
	tree.aware[77] = 5
	got := XDNDFindTarget(tree, 1, 100, 200, nil)
	if got.Window != 11 {
		t.Fatalf("the messages must still name the window under the pointer: got %d", got.Window)
	}
	if got.Proxy != 77 || got.Send() != 77 {
		t.Fatalf("they are sent to the proxy: got proxy %d send %d", got.Proxy, got.Send())
	}
}

// A proxy that does not name itself is left over from a crash; the spec
// says ignore the property rather than send into a dead window.
func TestXDNDFindTargetIgnoresAStaleProxy(t *testing.T) {
	tree := newFrameTree()
	tree.proxies = map[uint32]uint32{11: 77}
	tree.aware[77] = 5
	got := XDNDFindTarget(tree, 1, 100, 200, nil)
	if got.Window != 11 || got.Proxy != 0 || got.Send() != 11 {
		t.Fatalf("got %+v want the window itself", got)
	}
}

// A tree that loops (a window server race, or a mistake in a mock) must
// not hang the drag.
func TestXDNDFindTargetSurvivesALoop(t *testing.T) {
	tree := fakeTree{
		kids: map[uint32][]uint32{1: {2}, 2: {2}},
		geom: map[uint32][4]int{2: {0, 0, 100, 100}},
	}
	if got := XDNDFindTarget(tree, 1, 10, 10, nil); got.Valid() {
		t.Fatalf("got %+v", got)
	}
}

// The names and the indexes are two halves of one table: a name added to
// one without the other would intern the wrong atom for every message.
func TestXDNDAtomNamesMatchTheIndex(t *testing.T) {
	if len(XDNDAtomNames) != xdndAtomCount {
		t.Fatalf("%d names for %d atoms", len(XDNDAtomNames), xdndAtomCount)
	}
	for i, want := range map[XDNDAtom]string{
		XAAware: "XdndAware", XASelection: "XdndSelection", XAEnter: "XdndEnter",
		XAPosition: "XdndPosition", XAStatus: "XdndStatus", XALeave: "XdndLeave",
		XADrop: "XdndDrop", XAFinished: "XdndFinished", XATypeList: "XdndTypeList",
		XAActionCopy: "XdndActionCopy", XAActionMove: "XdndActionMove",
		XAActionLink: "XdndActionLink", XAActionAsk: "XdndActionAsk",
		XAActionPrivate: "XdndActionPrivate", XAActionList: "XdndActionList",
		XAProxy: "XdndProxy",
	} {
		if got := XDNDAtomNames[i]; got != want {
			t.Fatalf("atom %d is %q, want %q", int(i), got, want)
		}
	}
}
