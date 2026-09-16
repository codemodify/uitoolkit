package platform

// XDND — freedesktop's X11 drag-and-drop protocol — in pure Go: the
// client messages the two sides exchange, the version they settle on, the
// action atoms, and the search for the window under the pointer. The Xlib
// calls that carry all this are in x11_linux.go; the wire format lives
// here so it can be tested without a display.
//
// A drag is a conversation between the *source* (the window the drag
// started in) and the *target* (the toplevel under the pointer). The
// source sends XdndEnter once, XdndPosition as the pointer moves, and
// either XdndDrop or XdndLeave; the target answers every XdndPosition
// with an XdndStatus, and a drop with an XdndFinished. The data itself
// never travels in these messages: it is an ordinary X selection,
// XdndSelection, converted at the timestamp the source named.

// XDNDVersion is the version this toolkit speaks, in XdndAware and in the
// XdndEnter it sends. XDNDMinVersion is the oldest it talks to: version 3
// moved XdndAware to the toplevel window, and the spec sets 3 as the
// floor for a compliant implementation.
const (
	XDNDVersion    = 5
	XDNDMinVersion = 3
)

// XDNDAtomNames are every atom XDND needs, in one list so the backend can
// intern them in a single pass. The order is the XDNDAtom constants'.
var XDNDAtomNames = []string{
	"XdndAware",
	"XdndSelection",
	"XdndEnter",
	"XdndPosition",
	"XdndStatus",
	"XdndLeave",
	"XdndDrop",
	"XdndFinished",
	"XdndTypeList",
	"XdndActionCopy",
	"XdndActionMove",
	"XdndActionLink",
	"XdndActionAsk",
	"XdndActionPrivate",
	"XdndActionList",
	"XdndProxy",
}

// XDNDAtom indexes XDNDAtomNames.
type XDNDAtom int

const (
	XAAware XDNDAtom = iota
	XASelection
	XAEnter
	XAPosition
	XAStatus
	XALeave
	XADrop
	XAFinished
	XATypeList
	XAActionCopy
	XAActionMove
	XAActionLink
	XAActionAsk
	XAActionPrivate
	XAActionList
	XAProxy
)

// xdndAtomCount is how many there are, so a backend can hold them in a
// fixed array. TestXDNDAtomNamesMatchTheIndex keeps it and the names in
// step.
const xdndAtomCount = int(XAProxy) + 1

// XDNDMessage is one XDND client message's five data words as they go on
// the wire (ClientMessage, format 32).
type XDNDMessage [5]uint32

// XDNDRect is XdndStatus's "do not send another XdndPosition until the
// pointer leaves here" rectangle, in root coordinates. An empty one asks
// for a message on every move, which is what a toolkit whose targets are
// smaller than a window wants.
type XDNDRect struct{ X, Y, W, H int }

// packPoint and packSize are XDND's two-shorts-in-a-word packing,
// (x << 16) | y. Root coordinates are signed: a monitor left of or above
// the origin has negative ones.
func packPoint(x, y int) uint32 {
	return uint32(uint16(int16(x)))<<16 | uint32(uint16(int16(y)))
}

func unpackPoint(v uint32) (x, y int) {
	return int(int16(v >> 16)), int(int16(v))
}

// XDNDNegotiateVersion is the version two sides speak: the lower of the
// two, as the spec requires. It reports false when the target is too old
// (or advertises nothing at all).
func XDNDNegotiateVersion(ours, theirs int) (int, bool) {
	if theirs < XDNDMinVersion {
		return 0, false
	}
	if theirs < ours {
		return theirs, true
	}
	return ours, true
}

// EncodeXdndEnter announces a drag to the target: the source window, the
// version both sides speak, and the types offered. Only the first three
// types fit in the message; when there are more, bit 0 says so and the
// target reads the whole list from the source's XdndTypeList property.
func EncodeXdndEnter(source uint32, version int, types []uint32) XDNDMessage {
	var m XDNDMessage
	m[0] = source
	m[1] = uint32(version) << 24
	if len(types) > 3 {
		m[1] |= 1
	}
	for i := 0; i < 3 && i < len(types); i++ {
		m[2+i] = types[i]
	}
	return m
}

// DecodeXdndEnter reads an XdndEnter. more says the types in the message
// are only the first three and XdndTypeList on the source window has them
// all; types drops the unused (None) slots.
func DecodeXdndEnter(m XDNDMessage) (source uint32, version int, more bool, types []uint32) {
	source = m[0]
	version = int(m[1] >> 24)
	more = m[1]&1 != 0
	for _, t := range m[2:5] {
		if t != 0 {
			types = append(types, t)
		}
	}
	return
}

// EncodeXdndPosition tells the target where the pointer is, in root
// coordinates, with the timestamp the data must be converted at and the
// action the source asks for.
func EncodeXdndPosition(source uint32, x, y int, when uint32, action uint32) XDNDMessage {
	return XDNDMessage{source, 0, packPoint(x, y), when, action}
}

func DecodeXdndPosition(m XDNDMessage) (source uint32, x, y int, when uint32, action uint32) {
	x, y = unpackPoint(m[2])
	return m[0], x, y, m[3], m[4]
}

// EncodeXdndStatus answers an XdndPosition: whether this target takes a
// drop here and what it would do with it. The rectangle is where the
// answer stays the same; an empty one asks for a message on every move.
// A refused drop must name no action.
func EncodeXdndStatus(target uint32, accept bool, r XDNDRect, action uint32) XDNDMessage {
	var m XDNDMessage
	m[0] = target
	if accept {
		m[1] = 1
	} else {
		action = 0
	}
	m[2] = packPoint(r.X, r.Y)
	m[3] = packPoint(r.W, r.H)
	m[4] = action
	return m
}

// DecodeXdndStatus reads a target's answer. wantPos is its bit 1: it
// wants an XdndPosition even while the pointer stays inside the
// rectangle.
func DecodeXdndStatus(m XDNDMessage) (target uint32, accept, wantPos bool, r XDNDRect, action uint32) {
	target = m[0]
	accept = m[1]&1 != 0
	wantPos = m[1]&2 != 0
	r.X, r.Y = unpackPoint(m[2])
	r.W, r.H = unpackPoint(m[3])
	action = m[4]
	if !accept {
		action = 0
	}
	return
}

// EncodeXdndLeave withdraws a drag from a target that will not get it.
func EncodeXdndLeave(source uint32) XDNDMessage { return XDNDMessage{source} }

func DecodeXdndLeave(m XDNDMessage) (source uint32) { return m[0] }

// EncodeXdndDrop completes a drag. when is the timestamp the target must
// convert XdndSelection at, so a second drag started before the first
// finished cannot hand over the wrong data.
func EncodeXdndDrop(source uint32, when uint32) XDNDMessage {
	return XDNDMessage{source, 0, when}
}

func DecodeXdndDrop(m XDNDMessage) (source uint32, when uint32) { return m[0], m[2] }

// EncodeXdndFinished releases the source: the target is done with the
// data. Version 5 added whether the drop was taken and which action ran;
// a refused drop names none.
func EncodeXdndFinished(target uint32, accepted bool, action uint32) XDNDMessage {
	var m XDNDMessage
	m[0] = target
	if accepted {
		m[1] = 1
	} else {
		action = 0
	}
	m[2] = action
	return m
}

// DecodeXdndFinished reads the target's last word. A source speaking a
// version below 5 gets no "accepted" bit, so version says which one this
// was and below 5 the drop counts as taken — that is what the spec tells
// a source to assume.
func DecodeXdndFinished(m XDNDMessage, version int) (target uint32, accepted bool, action uint32) {
	target = m[0]
	if version < 5 {
		return target, true, m[2]
	}
	accepted = m[1]&1 != 0
	action = m[2]
	if !accepted {
		action = 0
	}
	return
}

// XDNDActions maps XDND's action atoms — interned at run time, so their
// values are only known then — to [DragAction] and back.
type XDNDActions struct {
	Copy, Move, Link, Ask, Private uint32
}

// Action is the action an atom names. XdndActionAsk is [DragAsk] — the
// source asking for the user to be shown the choice, which is a question
// and not an action. An unknown atom and the Private action this toolkit
// does not offer count as a copy: the spec lets a target fall back on
// copying, and a drag that shows no action at all is worse for the user
// than one that copies.
func (t XDNDActions) Action(atom uint32) DragAction {
	switch {
	case atom == 0:
		return DragNone
	case atom == t.Move:
		return DragMove
	case atom == t.Link:
		return DragLink
	case t.Ask != 0 && atom == t.Ask:
		return DragAsk
	default:
		return DragCopy
	}
}

// Atom is the atom for one action. DragNone (and a set with no action in
// it) is None, which is what a refused drop must send.
func (t XDNDActions) Atom(a DragAction) uint32 {
	switch a.One() {
	case DragCopy:
		return t.Copy
	case DragMove:
		return t.Move
	case DragLink:
		return t.Link
	}
	return 0
}

// List is every atom for a set of actions, the source's preferred one
// first — what goes in the XdndActionList property when a drag offers
// more than one.
func (t XDNDActions) List(a, preferred DragAction) []uint32 {
	var out []uint32
	seen := func(atom uint32) bool {
		for _, o := range out {
			if o == atom {
				return true
			}
		}
		return false
	}
	add := func(one DragAction) {
		if !a.Has(one) {
			return
		}
		if atom := t.Atom(one); atom != 0 && !seen(atom) {
			out = append(out, atom)
		}
	}
	add(preferred.One())
	for _, one := range dragActionOrder {
		add(one)
	}
	return out
}

// MergeDragOffer is everything a drop may do: what the source listed in
// XdndActionList, plus the action it asked for in the XdndPosition that
// carried it.
//
// The two are merged rather than the list taken alone because the list is
// an optional property and the requested action is not: a source is free
// to ask for an action it never listed, and the spec has the target
// honour what it was asked for. Going by the list alone would narrow such
// a drag back to the list's first action and quietly ignore the modifier
// the user is holding. (KDE's own sources do list all three, so this
// costs them nothing.)
//
// A source that says nothing at all means a copy, which is what every
// drag did before the protocol grew actions. The question [DragAsk] asks
// is not an action and never survives.
func MergeDragOffer(listed, requested DragAction) DragAction {
	if out := (listed | requested).Actions(); out != DragNone {
		return out
	}
	return DragCopy
}

// XDNDTree is the X window tree a drag's source looks through for the
// target under the pointer. x11_linux.go implements it with XQueryTree
// and XGetWindowAttributes; the tests implement it with a table, which is
// the only way to test the search without a display.
type XDNDTree interface {
	// Children of win in stacking order, bottom-most first (X's own
	// order), each in win's coordinates.
	Children(win uint32) []uint32
	// Geometry of win in its parent's coordinates. mapped is false for a
	// window that is not on screen, which can never be a drop target.
	Geometry(win uint32) (x, y, w, h int, mapped bool)
	// Aware is win's XdndAware version (0: the window does not take
	// drops) and its XdndProxy (0: none).
	Aware(win uint32) (version int, proxy uint32)
}

// XDNDTarget is the window a drag is over: the one the messages name, the
// one they are actually sent to (they differ when the target set
// XdndProxy), and the protocol version to speak.
type XDNDTarget struct {
	Window  uint32
	Proxy   uint32
	Version int
}

// Valid reports that a target was found.
func (t XDNDTarget) Valid() bool { return t.Window != 0 }

// Send is the window XSendEvent addresses: the proxy when there is one.
func (t XDNDTarget) Send() uint32 {
	if t.Proxy != 0 {
		return t.Proxy
	}
	return t.Window
}

// XDNDFindTarget is the window at a root-coordinate point that takes XDND
// drops. It walks down from the root through the topmost mapped child
// containing the point, and stops at the first window with XdndAware — a
// reparenting window manager puts its frame above the client, and the
// frame has no XdndAware, so the search has to descend rather than look
// only at the root's children. skip (may be nil) leaves a window and
// everything under it out: a drag's own icon window sits above everything
// and would otherwise swallow every drop.
//
// The version reported is what both sides speak; a window whose XdndAware
// is older than the spec's floor is passed over as if it took no drops.
func XDNDFindTarget(t XDNDTree, root uint32, x, y int, skip func(uint32) bool) XDNDTarget {
	win := root
	// depth is a guard: a tree that loops (a window server race, or a
	// mocked tree in a test) must not hang the drag.
	for depth := 0; depth < 64; depth++ {
		if win != root {
			if got, ok := xdndAwareTarget(t, win); ok {
				return got
			}
		}
		next, ok := xdndChildAt(t, win, x, y, skip)
		if !ok {
			return XDNDTarget{}
		}
		x, y = next.x, next.y
		win = next.win
	}
	return XDNDTarget{}
}

// xdndAwareTarget reads XdndAware (through XdndProxy when one is set) and
// reports the target to talk to. The spec puts XdndAware on the proxy
// window, and guards against a stale property by requiring the proxy to
// name itself in its own XdndProxy.
func xdndAwareTarget(t XDNDTree, win uint32) (XDNDTarget, bool) {
	version, proxy := t.Aware(win)
	if proxy != 0 && proxy != win {
		if _, back := t.Aware(proxy); back != proxy {
			// Left over from a crash: the spec says ignore it.
			proxy = 0
		} else {
			version, _ = t.Aware(proxy)
		}
	} else {
		proxy = 0
	}
	v, ok := XDNDNegotiateVersion(XDNDVersion, version)
	if !ok {
		return XDNDTarget{}, false
	}
	return XDNDTarget{Window: win, Proxy: proxy, Version: v}, true
}

type xdndChild struct {
	win  uint32
	x, y int
}

// xdndChildAt is the topmost mapped child of win holding the point, with
// the point translated into it.
func xdndChildAt(t XDNDTree, win uint32, x, y int, skip func(uint32) bool) (xdndChild, bool) {
	kids := t.Children(win)
	for i := len(kids) - 1; i >= 0; i-- { // topmost first
		kid := kids[i]
		if skip != nil && skip(kid) {
			continue
		}
		kx, ky, kw, kh, mapped := t.Geometry(kid)
		if !mapped || kw <= 0 || kh <= 0 {
			continue
		}
		if x < kx || y < ky || x >= kx+kw || y >= ky+kh {
			continue
		}
		return xdndChild{win: kid, x: x - kx, y: y - ky}, true
	}
	return xdndChild{}, false
}
