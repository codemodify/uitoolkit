//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

// Starting a drag on Win32: DoDragDrop, and the three COM objects it
// wants — IDataObject for what is being carried, IEnumFORMATETC for
// listing it, IDropSource for deciding when the drag ends.
//
// The one thing to know before reading it. **DoDragDrop blocks.** It runs
// a modal loop of its own until the user drops or gives up, and it
// returns the action that happened. The toolkit's contract is the other
// way round: StartDrag says whether a drag *began* and the drag ends
// later with an [EventDragEnd].
//
// The two are bridged by starting the drag on the *next* turn of the
// loop rather than inside StartDrag: StartDrag records the payload,
// answers true, and Poll runs DoDragDrop before it drains the queue. So
// the application's call returns at once and the blocking happens where
// the toolkit already expects to be — inside its own loop — instead of
// half way down an event handler.
//
// What that costs, stated plainly: for the length of the drag the
// application's loop is inside DoDragDrop, so the source window does not
// repaint. Windows pumps messages during its modal loop, so the window
// procedure runs and everything it queues is waiting afterwards, but
// nothing drains that queue until the drag ends. A Win32 application that
// paints from WM_PAINT does not have this problem and one that paints
// from its own loop does; that is a consequence of the loop being the
// toolkit's, and the alternative — re-entering the toolkit from inside
// IDropSource::GiveFeedback, which Windows calls throughout — is the
// re-entrancy the window procedure exists to avoid.

var (
	procDoDragDrop  = ole32.NewProc("DoDragDrop")
	procGlobalAlloc = kernel32.NewProc("GlobalAlloc")
	procGlobalFree  = kernel32.NewProc("GlobalFree")

	winSrcMu sync.Mutex
	winSrcBy = map[uintptr]*winDragSource{}
)

const (
	gmemMoveable           = 0x0002
	sFalse                 = 1
	dragDropSCancel        = 0x00040101 // DRAGDROP_S_CANCEL
	dragDropSDrop          = 0x00040100 // DRAGDROP_S_DROP
	dragDropSUseDefault    = 0x00040102
	eNotImpl               = 0x80004001
	eFail                  = 0x80004005
	dvENoTymed             = 0x80040069
	dvEFormatEtc           = 0x80040064
	oleEAdviseNotSupported = 0x80040003
)

// winDragSource carries one drag: the payload, and the three COM objects
// Windows is handed. Each object's first word is its vtable, which is
// what makes the Go struct's address usable as an interface pointer.
type winDragSource struct {
	dataVtbl *winIDataObjectVtbl
	data     struct{ vtbl *winIDataObjectVtbl }
	srcVtbl  *winIDropSourceVtbl
	src      struct{ vtbl *winIDropSourceVtbl }

	payload DragPayload
	formats []uint16 // the clipboard formats this drag offers
	surface *winSurface
}

type winIDataObjectVtbl struct {
	QueryInterface, AddRef, Release    uintptr
	GetData, GetDataHere, QueryGetData uintptr
	GetCanonicalFormatEtc, SetData     uintptr
	EnumFormatEtc, DAdvise, DUnadvise  uintptr
	EnumDAdvise                        uintptr
}

type winIDropSourceVtbl struct {
	QueryInterface, AddRef, Release uintptr
	QueryContinueDrag, GiveFeedback uintptr
}

func winSourceOf(this uintptr) *winDragSource {
	winSrcMu.Lock()
	defer winSrcMu.Unlock()
	return winSrcBy[this]
}

// winMimeFormat is the clipboard format a toolkit mime maps to, and
// winFormatMime is the way back. Only the two formats every Windows
// application understands are offered; a drag of anything else would be
// carried but never recognised.
func winMimeFormat(mime string) (uint16, bool) {
	switch mime {
	case "text/uri-list":
		return cfHDrop, true
	case "text/plain;charset=utf-8", "text/plain", "UTF8_STRING":
		return cfUnicodeText, true
	}
	return 0, false
}

func winFormatMime(cf uint16) string {
	if cf == cfHDrop {
		return "text/uri-list"
	}
	return "text/plain;charset=utf-8"
}

var winDataVtbl = winIDataObjectVtbl{
	QueryInterface: syscall.NewCallback(func(this, riid, ppv uintptr) uintptr {
		if ppv != 0 {
			*lparamAs[uintptr](ppv) = this
		}
		return sOK
	}),
	AddRef:  syscall.NewCallback(func(this uintptr) uintptr { return 2 }),
	Release: syscall.NewCallback(func(this uintptr) uintptr { return 1 }),
	GetData: syscall.NewCallback(func(this, pfmt, pmed uintptr) uintptr {
		d := winSourceOf(this)
		f := lparamAs[winFormatEtc](pfmt)
		m := lparamAs[winStgMedium](pmed)
		if d == nil || f == nil || m == nil {
			return eFail
		}
		if f.Tymed&tymedHGlobal == 0 {
			return dvENoTymed
		}
		h, ok := d.global(f.Format)
		if !ok {
			return dvEFormatEtc
		}
		*m = winStgMedium{Tymed: tymedHGlobal, Handle: h}
		return sOK
	}),
	GetDataHere: syscall.NewCallback(func(this, pfmt, pmed uintptr) uintptr { return eNotImpl }),
	QueryGetData: syscall.NewCallback(func(this, pfmt uintptr) uintptr {
		d := winSourceOf(this)
		f := lparamAs[winFormatEtc](pfmt)
		if d == nil || f == nil {
			return eFail
		}
		for _, cf := range d.formats {
			if cf == f.Format {
				return sOK
			}
		}
		return dvEFormatEtc
	}),
	GetCanonicalFormatEtc: syscall.NewCallback(func(this, in, out uintptr) uintptr { return eNotImpl }),
	SetData:               syscall.NewCallback(func(this, a, b, c uintptr) uintptr { return eNotImpl }),
	EnumFormatEtc:         syscall.NewCallback(func(this, dir, ppenum uintptr) uintptr { return eNotImpl }),
	DAdvise:               syscall.NewCallback(func(this, a, b, c, d uintptr) uintptr { return oleEAdviseNotSupported }),
	DUnadvise:             syscall.NewCallback(func(this, a uintptr) uintptr { return oleEAdviseNotSupported }),
	EnumDAdvise:           syscall.NewCallback(func(this, a uintptr) uintptr { return oleEAdviseNotSupported }),
}

var winSrcVtbl = winIDropSourceVtbl{
	QueryInterface: syscall.NewCallback(func(this, riid, ppv uintptr) uintptr {
		if ppv != 0 {
			*lparamAs[uintptr](ppv) = this
		}
		return sOK
	}),
	AddRef:  syscall.NewCallback(func(this uintptr) uintptr { return 2 }),
	Release: syscall.NewCallback(func(this uintptr) uintptr { return 1 }),
	// Escape gives up; letting the left button go drops. Both are what
	// Windows itself does, and saying so here rather than answering
	// S_OK keeps the drag from outliving the gesture that began it.
	QueryContinueDrag: syscall.NewCallback(func(this, esc, keys uintptr) uintptr {
		const mkLButton = 0x0001
		if esc != 0 {
			return dragDropSCancel
		}
		if keys&mkLButton == 0 {
			return dragDropSDrop
		}
		return sOK
	}),
	GiveFeedback: syscall.NewCallback(func(this, effect uintptr) uintptr {
		// Let Windows pick the cursor for the effect it has settled on.
		return dragDropSUseDefault
	}),
}

// global is an HGLOBAL holding this drag's payload in format cf, made
// fresh each time: GetData gives the medium away, and whoever took it
// frees it.
func (d *winDragSource) global(cf uint16) (uintptr, bool) {
	if d.payload.Data == nil {
		return 0, false
	}
	b, ok := d.payload.Data(winFormatMime(cf))
	if !ok {
		return 0, false
	}
	switch cf {
	case cfUnicodeText:
		u, err := syscall.UTF16FromString(string(b))
		if err != nil {
			return 0, false
		}
		return winGlobalOf(unsafe.Slice((*byte)(unsafe.Pointer(&u[0])), len(u)*2)), true
	case cfHDrop:
		return winHDropOf(string(b))
	}
	return 0, false
}

// winGlobalOf copies bytes into a moveable HGLOBAL.
func winGlobalOf(b []byte) uintptr {
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(b)))
	if h == 0 {
		return 0
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return 0
	}
	copy(unsafe.Slice(lparamAs[byte](p), len(b)), b)
	procGlobalUnlock.Call(h)
	return h
}

// winHDropOf builds a CF_HDROP from a uri-list: a DROPFILES header, then
// the paths as UTF-16, each NUL-terminated, the list ended by one more.
func winHDropOf(uriList string) (uintptr, bool) {
	paths := winPathsFromURIs(uriList)
	if len(paths) == 0 {
		return 0, false
	}
	var names []uint16
	for _, p := range paths {
		u, err := syscall.UTF16FromString(p)
		if err != nil {
			continue
		}
		names = append(names, u...) // UTF16FromString already NUL-terminates
	}
	names = append(names, 0) // and one more to end the list
	const hdr = 20           // DROPFILES: DWORD pFiles, POINT pt, BOOL fNC, BOOL fWide
	b := make([]byte, hdr+len(names)*2)
	put32 := func(off int, v uint32) {
		b[off] = byte(v)
		b[off+1] = byte(v >> 8)
		b[off+2] = byte(v >> 16)
		b[off+3] = byte(v >> 24)
	}
	put32(0, hdr) // the names start straight after the header
	put32(16, 1)  // fWide: the names are UTF-16
	copy(b[hdr:], unsafe.Slice((*byte)(unsafe.Pointer(&names[0])), len(names)*2))
	h := winGlobalOf(b)
	return h, h != 0
}

// winPathsFromURIs is the other direction from winFileURI: a uri-list
// back to Windows paths, skipping anything that is not a local file.
func winPathsFromURIs(list string) []string {
	var out []string
	for _, line := range splitLines(list) {
		if line == "" || line[0] == '#' {
			continue
		}
		p, ok := winPathFromURI(line)
		if ok {
			out = append(out, p)
		}
	}
	return out
}

func winPathFromURI(uri string) (string, bool) {
	const pfx = "file:///"
	if len(uri) <= len(pfx) || uri[:len(pfx)] != pfx {
		return "", false
	}
	s := uri[len(pfx):]
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, ok1 := unhex(s[i+1])
			lo, ok2 := unhex(s[i+2])
			if ok1 && ok2 {
				b = append(b, hi<<4|lo)
				i += 2
				continue
			}
		}
		if s[i] == '/' {
			b = append(b, '\\')
			continue
		}
		b = append(b, s[i])
	}
	return string(b), len(b) > 0
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := s[start:i]
			if n := len(line); n > 0 && line[n-1] == '\r' {
				line = line[:n-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	return out
}

// ---- the surface's side ---------------------------------------------------

// StartDrag begins a drag of this window's own ([DragSurface]).
//
// It records the payload and answers at once; the drag itself starts on
// the next turn of the loop, because DoDragDrop blocks until it is over
// and the contract here is that StartDrag returns and an EventDragEnd
// follows.
func (s *winSurface) StartDrag(p DragPayload) bool {
	if s == nil || s.hwnd == 0 || s.closed || s.drag != nil || s.pendingDrag != nil {
		return false
	}
	d := &winDragSource{payload: p, surface: s}
	for _, m := range p.Types {
		if cf, ok := winMimeFormat(m); ok && !containsFormat(d.formats, cf) {
			d.formats = append(d.formats, cf)
		}
	}
	if len(d.formats) == 0 {
		return false
	}
	d.data.vtbl, d.src.vtbl = &winDataVtbl, &winSrcVtbl
	s.pendingDrag = d
	return true
}

func containsFormat(fs []uint16, cf uint16) bool {
	for _, f := range fs {
		if f == cf {
			return true
		}
	}
	return false
}

// CancelDrag stops one that has not started yet; once DoDragDrop is
// running only Escape ends it, which QueryContinueDrag answers.
func (s *winSurface) CancelDrag() {
	if s != nil {
		s.pendingDrag = nil
	}
}

// Dragging reports whether a drag started here is still running.
func (s *winSurface) Dragging() bool { return s != nil && s.drag != nil }

// runPendingDrag is Poll's half: it runs the modal drag and pushes the
// EventDragEnd that says what the target did.
func (s *winSurface) runPendingDrag() {
	d := s.pendingDrag
	if d == nil {
		return
	}
	s.pendingDrag, s.drag = nil, d

	dataThis := uintptr(unsafe.Pointer(&d.data))
	srcThis := uintptr(unsafe.Pointer(&d.src))
	winSrcMu.Lock()
	winSrcBy[dataThis], winSrcBy[srcThis] = d, d
	winSrcMu.Unlock()

	allowed := d.payload.Actions
	if allowed == 0 {
		allowed = DragCopy
	}
	var effect uint32
	procDoDragDrop.Call(dataThis, srcThis, uintptr(winEffectsOf(allowed)), uintptr(unsafe.Pointer(&effect)))

	winSrcMu.Lock()
	delete(winSrcBy, dataThis)
	delete(winSrcBy, srcThis)
	winSrcMu.Unlock()
	s.drag = nil

	act := winActionOf(effect)
	s.push(Event{Kind: EventDragEnd, Action: act, Dropped: act != DragNone})
}

func winEffectsOf(a DragAction) uint32 {
	var e uint32
	if a&DragCopy != 0 {
		e |= dropEffectCopy
	}
	if a&DragMove != 0 {
		e |= dropEffectMove
	}
	if a&DragLink != 0 {
		e |= dropEffectLink
	}
	return e
}

func winActionOf(e uint32) DragAction {
	switch {
	case e&dropEffectMove != 0:
		return DragMove
	case e&dropEffectLink != 0:
		return DragLink
	case e&dropEffectCopy != 0:
		return DragCopy
	}
	return DragNone
}

var _ DragSurface = (*winSurface)(nil)
