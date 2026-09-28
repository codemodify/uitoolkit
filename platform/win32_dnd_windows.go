//go:build windows

package platform

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Taking a drop on Win32: OLE drag and drop, which is COM, built here
// without cgo — a vtable of syscall.NewCallback entries and an object
// whose first word points at it.
//
// The shape of the problem, and why this is not a straight translation.
// IDropTarget::DragEnter and ::DragOver must answer *now*, with the
// effect the window would have if the pointer let go here; the toolkit
// answers later, because [EventDragMotion] goes on the queue and the
// application replies with AcceptDrag when it gets round to it. There is
// no way to make a synchronous question wait for an asynchronous answer
// without running the toolkit's loop inside Windows' — re-entering the
// whole toolkit from inside a COM call, which is the thing the window
// procedure is written not to do.
//
// So the answer given is the *last* one the application settled on, and
// the motion is queued for the next turn. A drag that has just arrived
// over a window is offered whatever that window last agreed to take, and
// is corrected a frame later. One frame of a cursor showing the wrong
// verb is the whole cost, and it is the cost every toolkit bridging these
// two models pays.

var (
	ole32            = syscall.NewLazyDLL("ole32.dll")
	procOleInit      = ole32.NewProc("OleInitialize")
	procRegDragDrop  = ole32.NewProc("RegisterDragDrop")
	procRevDragDrop  = ole32.NewProc("RevokeDragDrop")
	procReleaseStg   = ole32.NewProc("ReleaseStgMedium")
	procDragQuery    = shell32.NewProc("DragQueryFileW")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")

	oleOnce sync.Once

	winDropMu sync.Mutex
	// winDropBy maps a COM object's address back to its surface: the
	// methods are given `this` and nothing else.
	winDropBy = map[uintptr]*winSurface{}
)

const (
	cfUnicodeText   = 13
	cfHDrop         = 15
	tymedHGlobal    = 1
	dvAspectContent = 1

	dropEffectNone = 0
	dropEffectCopy = 1
	dropEffectMove = 2
	dropEffectLink = 4

	sOK = 0
)

// winIDropTargetVtbl is IDropTarget's vtable: IUnknown's three, then the
// four of its own, in the order the interface declares them.
type winIDropTargetVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	DragEnter      uintptr
	DragOver       uintptr
	DragLeave      uintptr
	Drop           uintptr
}

// winDropTarget is the COM object itself. The vtable pointer must be the
// first word: that is what "a COM interface pointer" means.
type winDropTarget struct {
	vtbl *winIDropTargetVtbl
	refs int32
}

type winFormatEtc struct {
	Format uint16
	_      [3]uint16
	Ptd    uintptr
	Aspect uint32
	Index  int32
	Tymed  uint32
	_      uint32
}

type winStgMedium struct {
	Tymed  uint32
	_      uint32
	Handle uintptr
	Unk    uintptr
}

// comCall calls slot n of the vtable of the COM object at iface.
func comCall(iface uintptr, slot int, args ...uintptr) uintptr {
	obj := lparamAs[struct{ vtbl *[64]uintptr }](iface)
	if obj == nil || obj.vtbl == nil {
		return ^uintptr(0)
	}
	r, _, _ := syscall.SyscallN(obj.vtbl[slot], append([]uintptr{iface}, args...)...)
	return r
}

var winDropVtbl = winIDropTargetVtbl{
	QueryInterface: syscall.NewCallback(func(this, riid, ppv uintptr) uintptr {
		// Every caller of this object already holds the one interface it
		// has; answering E_NOINTERFACE for anything else is honest and is
		// all RegisterDragDrop needs.
		if ppv != 0 {
			*lparamAs[uintptr](ppv) = 0
		}
		return 0x80004002 // E_NOINTERFACE
	}),
	AddRef:  syscall.NewCallback(func(this uintptr) uintptr { return 2 }),
	Release: syscall.NewCallback(func(this uintptr) uintptr { return 1 }),
	DragEnter: syscall.NewCallback(func(this, data, keys, ptLo, ptHi, effect uintptr) uintptr {
		return winDropEvent(this, data, keys, ptLo, ptHi, effect, EventDragMotion)
	}),
	DragOver: syscall.NewCallback(func(this, keys, ptLo, ptHi, effect uintptr) uintptr {
		return winDropEvent(this, 0, keys, ptLo, ptHi, effect, EventDragMotion)
	}),
	DragLeave: syscall.NewCallback(func(this uintptr) uintptr {
		if s := winDropSurface(this); s != nil {
			s.dragMimes = nil
			s.push(Event{Kind: EventDragLeave})
		}
		return sOK
	}),
	Drop: syscall.NewCallback(func(this, data, keys, ptLo, ptHi, effect uintptr) uintptr {
		return winDropEvent(this, data, keys, ptLo, ptHi, effect, EventDrop)
	}),
}

func winDropSurface(this uintptr) *winSurface {
	winDropMu.Lock()
	defer winDropMu.Unlock()
	return winDropBy[this]
}

// winDropEvent is the body of DragEnter, DragOver and Drop: read what is
// on offer, queue the event, and answer with what the window last agreed
// to take.
//
// The POINTL is passed by value, which on this calling convention arrives
// as two words: x in the low, y in the high. It is in *screen*
// coordinates, like the wheel's and unlike a button's.
func winDropEvent(this, data, keys, ptLo, ptHi, effect uintptr, kind EventKind) uintptr {
	s := winDropSurface(this)
	if s == nil {
		if effect != 0 {
			*lparamAs[uint32](effect) = dropEffectNone
		}
		return sOK
	}
	if data != 0 {
		s.dragMimes = winDragTypes(data)
		s.dragData = data
	}
	ev := Event{
		Kind:    kind,
		Pos:     s.screenPoint(ptLo&0xFFFF | (ptHi&0xFFFF)<<16),
		Mimes:   append([]string(nil), s.dragMimes...),
		Actions: winDragActions(keys),
		Mods:    winMods(),
	}
	ev.Action = s.dragAction
	if ev.Action == DragNone {
		ev.Action = DragCopy
	}
	if kind == EventDrop {
		ev.Dropped = true
		s.dragPayload = winDragRead(data)
	}
	s.push(ev)
	if effect != 0 {
		*lparamAs[uint32](effect) = winEffectOf(s.dragAccept, s.dragAction)
	}
	if kind == EventDrop {
		s.dragMimes, s.dragData = nil, 0
	}
	return sOK
}

// winDragActions is what the source is offering, narrowed by the keys the
// user holds — the same convention every desktop shares, which Windows
// leaves to the target to apply.
func winDragActions(keys uintptr) DragAction {
	const mkCtrl, mkShift = 0x0008, 0x0004
	switch {
	case keys&mkCtrl != 0 && keys&mkShift != 0:
		return DragLink
	case keys&mkShift != 0:
		return DragMove
	case keys&mkCtrl != 0:
		return DragCopy
	}
	return DragCopy | DragMove | DragLink
}

func winEffectOf(accepted bool, a DragAction) uint32 {
	if !accepted {
		return dropEffectNone
	}
	switch {
	case a&DragMove != 0:
		return dropEffectMove
	case a&DragLink != 0:
		return dropEffectLink
	default:
		return dropEffectCopy
	}
}

// winDragTypes is the mime names for what an IDataObject holds, in the
// toolkit's vocabulary rather than Windows'.
func winDragTypes(data uintptr) []string {
	var out []string
	if winHasFormat(data, cfHDrop) {
		out = append(out, "text/uri-list")
	}
	if winHasFormat(data, cfUnicodeText) {
		out = append(out, "text/plain;charset=utf-8", "text/plain")
	}
	return out
}

func winFormat(cf uint16) winFormatEtc {
	return winFormatEtc{Format: cf, Aspect: dvAspectContent, Index: -1, Tymed: tymedHGlobal}
}

// winHasFormat asks IDataObject::QueryGetData, slot 5.
func winHasFormat(data uintptr, cf uint16) bool {
	f := winFormat(cf)
	return comCall(data, 5, uintptr(unsafe.Pointer(&f))) == sOK
}

// winDragRead pulls the payload out, files first.
func winDragRead(data uintptr) map[string][]byte {
	out := map[string][]byte{}
	if data == 0 {
		return out
	}
	if b, ok := winGetMedium(data, cfHDrop, winFilesFromHDrop); ok {
		out["text/uri-list"] = b
	}
	if b, ok := winGetMedium(data, cfUnicodeText, winTextFromGlobal); ok {
		out["text/plain;charset=utf-8"] = b
		out["text/plain"] = b
	}
	return out
}

// winGetMedium calls IDataObject::GetData (slot 3) and hands the medium
// to read, then releases it however the call went.
func winGetMedium(data uintptr, cf uint16, read func(h uintptr) []byte) ([]byte, bool) {
	f := winFormat(cf)
	var m winStgMedium
	if comCall(data, 3, uintptr(unsafe.Pointer(&f)), uintptr(unsafe.Pointer(&m))) != sOK {
		return nil, false
	}
	defer procReleaseStg.Call(uintptr(unsafe.Pointer(&m)))
	if m.Handle == 0 {
		return nil, false
	}
	return read(m.Handle), true
}

// winFilesFromHDrop is the dropped files as a text/uri-list: DragQueryFileW
// with 0xFFFFFFFF asks how many there are, then each in turn.
func winFilesFromHDrop(h uintptr) []byte {
	n, _, _ := procDragQuery.Call(h, ^uintptr(0), 0, 0)
	var b strings.Builder
	for i := uintptr(0); i < n; i++ {
		need, _, _ := procDragQuery.Call(h, i, 0, 0)
		if need == 0 {
			continue
		}
		buf := make([]uint16, need+1)
		procDragQuery.Call(h, i, uintptr(unsafe.Pointer(&buf[0])), need+1)
		b.WriteString(winFileURI(syscall.UTF16ToString(buf)))
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

// winFileURI turns C:\a b\c.txt into file:///C:/a%20b/c.txt, which is
// what a uri-list holds and what every other backend hands over.
func winFileURI(path string) string {
	var b strings.Builder
	b.WriteString("file:///")
	for _, r := range strings.ReplaceAll(path, `\`, "/") {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '/', r == '-', r == '_', r == '.', r == '~', r == ':':
			b.WriteRune(r)
		default:
			for _, c := range []byte(string(r)) {
				b.WriteByte('%')
				const hex = "0123456789ABCDEF"
				b.WriteByte(hex[c>>4])
				b.WriteByte(hex[c&0x0F])
			}
		}
	}
	return b.String()
}

// winTextFromGlobal reads a CF_UNICODETEXT HGLOBAL as UTF-8.
func winTextFromGlobal(h uintptr) []byte {
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return nil
	}
	defer procGlobalUnlock.Call(h)
	// The text is NUL-terminated; walk to it rather than trusting a size.
	u := lparamAs[[1 << 20]uint16](ptr)
	n := 0
	for n < len(u) && u[n] != 0 {
		n++
	}
	return []byte(syscall.UTF16ToString(u[:n]))
}

// ---- the surface's side ---------------------------------------------------

// registerDrop makes this window a drop target. OLE wants the thread in a
// single-threaded apartment, which is what OleInitialize gives it, and
// which is only meaningful because the run loop is pinned to one thread
// (mainthread_windows.go).
func (s *winSurface) registerDrop() {
	if s.hwnd == 0 {
		return
	}
	oleOnce.Do(func() { procOleInit.Call(0) })
	s.dropTarget = &winDropTarget{vtbl: &winDropVtbl, refs: 1}
	this := uintptr(unsafe.Pointer(s.dropTarget))
	winDropMu.Lock()
	winDropBy[this] = s
	winDropMu.Unlock()
	if r, _, _ := procRegDragDrop.Call(s.hwnd, this); r != sOK {
		winDropMu.Lock()
		delete(winDropBy, this)
		winDropMu.Unlock()
		s.dropTarget = nil
	}
}

func (s *winSurface) revokeDrop() {
	if s.dropTarget == nil {
		return
	}
	procRevDragDrop.Call(s.hwnd)
	winDropMu.Lock()
	delete(winDropBy, uintptr(unsafe.Pointer(s.dropTarget)))
	winDropMu.Unlock()
	s.dropTarget = nil
}

// AcceptDrag records what the window would do with the drag now
// ([DropNegotiator]). It cannot answer Windows on the spot — the question
// was asked and answered before this call — so it sets what the *next*
// DragOver will say, which is the one-frame lag this file opens with.
func (s *winSurface) AcceptDrag(mime string, allowed, action DragAction) {
	if s == nil {
		return
	}
	s.dragAccept = mime != "" && action != DragNone
	s.dragAction = action
}

// DragData is what was dropped, by mime type: the payload read while the
// drop was being delivered, because an IDataObject is only valid for the
// length of the call that handed it over.
func (s *winSurface) DragData(mime string) ([]byte, bool) {
	if s == nil || s.dragPayload == nil {
		return nil, false
	}
	b, ok := s.dragPayload[mime]
	return b, ok
}

var _ DropNegotiator = (*winSurface)(nil)
