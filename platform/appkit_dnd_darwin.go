//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "appkit_darwin.h"
*/
import "C"

import (
	"bytes"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// Drag and drop on AppKit: the source half ([DragSurface]) and the
// target half ([DropNegotiator] and [DropReceiver]).
//
// Types cross as MIME strings, which is what the toolkit speaks. The
// Objective-C side maps the two macOS has names of its own for — text
// and file URLs — and puts everything else on the pasteboard under the
// MIME string itself, which NSPasteboard allows. So a toolkit-specific
// type travels between two uitoolkit windows with nothing registered
// anywhere, and text and files still arrive from Finder and from every
// other application.
//
// **The drop is read inside the callback, not after it.** A dragging
// pasteboard belongs to the drag and the drag is over the moment
// -performDragOperation: returns, so the data is copied out there and
// [ReceiveDrop] is served from that copy. XDND and wl_data_offer both
// let a target read afterwards; AppKit does not, and a backend that
// assumed it could would read an empty pasteboard exactly when a user
// dropped something big enough to be worth dropping.

// AcceptDrag says what this window would do with the drag over it
// ([DropNegotiator]).
//
// mime "" or action [DragNone] refuses, and AppKit shows the "no drop"
// pointer for it — the answer goes into the view, and -draggingUpdated:
// returns it for every step of the drag.
func (s *akSurface) AcceptDrag(mime string, allowed, action DragAction) {
	if s == nil || s.win == nil {
		return
	}
	s.dragAccept = mime != "" && action != DragNone
	s.dragAction = action
	a := action
	if !s.dragAccept {
		a = DragNone
	}
	C.uitk_ak_set_drop_answer(s.win, C.int(a))
}

// ReceiveDrop is the dropped data as mime ([DropReceiver]).
//
// It answers from what was copied out of the pasteboard when the drop
// arrived; see the note above on why it cannot read it now.
func (s *akSurface) ReceiveDrop(mime string) ([]byte, bool) {
	if s == nil {
		return nil, false
	}
	b, ok := s.dropData[mime]
	return b, ok && len(b) > 0
}

// FinishDrop completes the drop ([DropReceiver]).
//
// There is nothing to tell AppKit: -performDragOperation: already
// answered, and it had to, because it is synchronous. This drops the
// copied payload, which is the only thing still owed.
func (s *akSurface) FinishDrop(ok bool) {
	if s == nil {
		return
	}
	s.dropData = nil
	s.dragMimes = nil
	if s.win != nil {
		C.uitk_ak_set_drop_answer(s.win, 0)
	}
}

// StartDrag begins a drag from the press being handled ([DragSurface]).
func (s *akSurface) StartDrag(p DragPayload) bool {
	if s == nil || s.win == nil || s.Closed() || len(p.Types) == 0 {
		return false
	}
	item := C.uitk_ak_drag_new()
	if item == nil {
		return false
	}
	any := false
	for _, mime := range p.Types {
		if p.Data == nil {
			break
		}
		b, ok := p.Data(mime)
		if !ok || len(b) == 0 {
			continue
		}
		cm := C.CString(mime)
		put := C.uitk_ak_drag_add(item, cm, unsafe.Pointer(&b[0]), C.int(len(b))) != 0
		C.free(unsafe.Pointer(cm))
		// Only a payload that actually reached the pasteboard counts: a
		// drag whose every type was refused is a drag carrying nothing,
		// and starting it would show the user a gesture that cannot drop.
		any = any || put
	}
	if !any {
		C.uitk_ak_drag_free(item)
		return false
	}
	// Every window of this process is told it may be offered these types.
	// A view hears nothing about a type it has not registered, and the
	// window a tab is dropped on cannot have known the type in advance.
	akMu.Lock()
	wins := make([]*akSurface, 0, len(akBySID))
	for _, o := range akBySID {
		wins = append(wins, o)
	}
	akMu.Unlock()
	for _, mime := range p.Types {
		cm := C.CString(mime)
		for _, o := range wins {
			if o.win != nil && !o.Closed() {
				C.uitk_ak_register_drag_types(o.win, cm)
			}
		}
		C.free(unsafe.Pointer(cm))
	}

	actions := p.Actions
	if actions == DragNone {
		actions = DragCopy
	}
	var icon *C.uchar
	var iw, ih C.int
	if p.Icon != nil && len(p.Icon.Pix) > 0 {
		icon = (*C.uchar)(unsafe.Pointer(&p.Icon.Pix[0]))
		iw, ih = C.int(p.Icon.Width), C.int(p.Icon.Height)
	}
	started := C.uitk_ak_drag_start(s.win, item, icon, iw, ih,
		C.double(p.Hotspot.X), C.double(p.Hotspot.Y), C.int(actions)) != 0
	// The session retains what it needs off the item; releasing the
	// reference taken by uitk_ak_drag_new is this side's to do either
	// way, and leaking it on the way out of a refused drag would be a
	// pasteboard item per attempt.
	C.uitk_ak_drag_free(item)
	if started {
		s.dragging = true
	}
	return started
}

// CancelDrag stops a drag early ([DragSurface]).
//
// AppKit has no way to call one off: NSDraggingSession runs its own
// loop and ends when the button comes up. Escape during a drag is
// AppKit's own business and it cancels there, which is the behaviour a
// user expects — so this reports the truth rather than pretending, the
// same answer [WindowFrame.StartResize] gives for the same kind of
// reason.
func (s *akSurface) CancelDrag() {}

// Dragging reports whether a drag started here is still running
// ([DragSurface]).
func (s *akSurface) Dragging() bool {
	if s == nil || s.win == nil {
		return false
	}
	return C.uitk_ak_dragging(s.win) != 0
}

// akDropTypes is the MIME types the drag now over the window offers.
func (s *akSurface) akDropTypes() []string {
	if s == nil || s.win == nil {
		return nil
	}
	var n C.int
	p := C.uitk_ak_drop_types(s.win, &n)
	if p == nil || n <= 0 {
		return nil
	}
	defer C.free(unsafe.Pointer(p))
	// The whole block in one piece, split in Go: the splitting is the
	// part that can be wrong, and in Go it can be tested — cgo is not
	// allowed in a _test.go file, so anything that takes a *C.char
	// cannot be.
	return parseMimeList(C.GoBytes(unsafe.Pointer(p), n))
}

// parseMimeList splits a NUL-separated list of MIME names. A trailing
// NUL leaves no empty entry, and neither does a doubled one.
func parseMimeList(b []byte) []string {
	var out []string
	for _, f := range bytes.Split(b, []byte{0}) {
		if len(f) > 0 {
			out = append(out, string(f))
		}
	}
	return out
}

// akTakeDrop copies every offered type out of the dragging pasteboard,
// which is only readable while the drop callback is on the stack.
func (s *akSurface) akTakeDrop(mimes []string) map[string][]byte {
	out := map[string][]byte{}
	for _, m := range mimes {
		cm := C.CString(m)
		var n C.int
		p := C.uitk_ak_drop_data(s.win, cm, &n)
		C.free(unsafe.Pointer(cm))
		if p == nil || n <= 0 {
			continue
		}
		out[m] = C.GoBytes(p, n)
		C.free(p)
	}
	return out
}

// akDragEvent turns one of the Objective-C side's drag reports into the
// toolkit's event. It runs inside the AppKit callback, which is why the
// drop reads the pasteboard here.
func (s *akSurface) akDragEvent(kind C.int, at paintengine2d.Point, allowed, action DragAction) {
	switch kind {
	case C.UITK_AK_DRAG_MOTION:
		s.dragMimes = s.akDropTypes()
		s.push(Event{
			Kind: EventDragMotion, Pos: at,
			Mimes: s.dragMimes, Actions: allowed,
			Action: ModifierDragAction(s.mods, allowed, action),
		})
	case C.UITK_AK_DRAG_LEAVE:
		s.dragMimes, s.dropData = nil, nil
		s.push(Event{Kind: EventDragLeave})
	case C.UITK_AK_DROP:
		s.dragMimes = s.akDropTypes()
		s.dropData = s.akTakeDrop(s.dragMimes)
		s.push(Event{
			Kind: EventDrop, Pos: at,
			Mimes: s.dragMimes, Actions: allowed,
			Action: ModifierDragAction(s.mods, allowed, action),
		})
	case C.UITK_AK_DRAG_END:
		s.dragging = false
		s.push(Event{Kind: EventDragEnd, Action: action, Dropped: action != DragNone})
	}
}

var (
	_ DragSurface    = (*akSurface)(nil)
	_ DropNegotiator = (*akSurface)(nil)
	_ DropReceiver   = (*akSurface)(nil)
)

// dragTypeOfMime is the pasteboard type a MIME travels as, and mimeOfDragType
// the reverse — empty for a type that is not the toolkit's.
//
// A MIME is not a valid UTI (every one has a '/' in it) and macOS stores
// nothing under an invalid one, so a custom type is carried under an encoded
// name and turned back when the pasteboard is enumerated. These two are that
// encoding's only seam, and the only way to hold the write side and the read
// side to each other without a live drag session.
func dragTypeOfMime(mime string) string {
	cm := C.CString(mime)
	defer C.free(unsafe.Pointer(cm))
	t := C.uitk_ak_drag_type_of_mime(cm)
	if t == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(t))
	return C.GoString(t)
}

func mimeOfDragType(t string) string {
	ct := C.CString(t)
	defer C.free(unsafe.Pointer(ct))
	m := C.uitk_ak_mime_of_drag_type(ct)
	if m == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(m))
	return C.GoString(m)
}

// newDragItem, addDragPayload and freeDragItem are the pasteboard item a drag
// carries. addDragPayload reports whether the payload reached it.
func newDragItem() unsafe.Pointer { return C.uitk_ak_drag_new() }

func freeDragItem(item unsafe.Pointer) { C.uitk_ak_drag_free(item) }

func addDragPayload(item unsafe.Pointer, mime string, b []byte) bool {
	if len(b) == 0 {
		return false
	}
	cm := C.CString(mime)
	defer C.free(unsafe.Pointer(cm))
	return C.uitk_ak_drag_add(item, cm, unsafe.Pointer(&b[0]), C.int(len(b))) != 0
}
