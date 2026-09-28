//go:build windows

package platform

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// Input methods on Win32, through the Input Method Manager (imm32).
//
// The shape is the Linux backends' and macOS's: Windows reports what
// the method is composing and what the user chose, and the toolkit does
// the editing. Three messages carry it — WM_IME_STARTCOMPOSITION,
// WM_IME_COMPOSITION and WM_IME_ENDCOMPOSITION — and the composition
// string is read out of the input context with ImmGetCompositionStringW
// while the message is being handled, because it is the context's and
// not ours.
//
// **The default composition window is turned off.** Windows will draw
// the preedit itself, in a small box of its own near the caret, if the
// application does not say otherwise — and then the same text is on
// screen twice, once in the toolkit's text field and once in the IME's
// box. WM_IME_SETCONTEXT is where that is refused, by clearing
// ISC_SHOWUICOMPOSITIONWINDOW from its lParam before the default
// handler sees it.
//
// The candidate list is left to Windows: it is the *method's* window,
// every application on the desktop gets the same one, and a toolkit
// that drew its own would be drawing a list of characters it does not
// know how to choose between. What the toolkit owes it is the caret's
// position, so it opens beside the text rather than in the corner —
// that is [SetIMECursor], and it is ImmSetCandidateWindow.

var (
	imm32 = syscall.NewLazyDLL("imm32.dll")

	procImmGetContext         = imm32.NewProc("ImmGetContext")
	procImmReleaseContext     = imm32.NewProc("ImmReleaseContext")
	procImmGetCompositionStrW = imm32.NewProc("ImmGetCompositionStringW")
	procImmSetCompositionWnd  = imm32.NewProc("ImmSetCompositionWindow")
	procImmSetCandidateWindow = imm32.NewProc("ImmSetCandidateWindow")
	procImmAssociateContext   = imm32.NewProc("ImmAssociateContext")
	procImmNotifyIME          = imm32.NewProc("ImmNotifyIME")
)

const (
	wmImeStartComposition = 0x010D
	wmImeEndComposition   = 0x010E
	wmImeComposition      = 0x010F
	wmImeSetContext       = 0x0281

	// What ImmGetCompositionStringW is asked for.
	gcsCompStr   = 0x0008
	gcsCursorPos = 0x0080
	gcsResultStr = 0x0800

	// ISC_SHOWUICOMPOSITIONWINDOW: the IME's own preedit box, which the
	// toolkit draws itself.
	iscShowUIComposition = 0x80000000

	cfsPoint   = 0x0002
	cfsExclude = 0x0080

	// NI_COMPOSITIONSTR with CPS_CANCEL: throw away what is being
	// composed.
	niCompositionStr = 0x0015
	cpsCancel        = 0x0004
)

// winCompositionForm is COMPOSITIONFORM and winCandidateForm is
// CANDIDATEFORM: where the preedit and the candidate list go.
// COMPOSITIONFORM is DWORD, POINT, RECT with no padding between them:
// 4 + 8 is already 4-aligned, which is all a RECT of LONGs asks for.
type winCompositionForm struct {
	Style      uint32
	CurrentPos winPoint
	Area       winRect
}

type winCandidateForm struct {
	Index      uint32
	Style      uint32
	CurrentPos winPoint
	Area       winRect
}

// SetIMEEnabled turns the input method on for this window
// ([IMESurface]).
//
// A window with no input context associated gets no IME at all, which
// is what a window with no text field focused wants: the method's own
// key handling stays out of the way of shortcuts. The context is kept
// rather than destroyed, so turning it back on is the same one.
func (s *winSurface) SetIMEEnabled(on bool) {
	if s == nil || s.hwnd == 0 || s.closed {
		return
	}
	if on == s.imeOn {
		return
	}
	s.imeOn = on
	if on {
		if s.imeCtx != 0 {
			procImmAssociateContext.Call(s.hwnd, s.imeCtx)
			s.imeCtx = 0
		}
		return
	}
	// Anything half-composed goes with it, or the next field inherits
	// the last one's preedit.
	s.cancelComposition()
	old, _, _ := procImmAssociateContext.Call(s.hwnd, 0)
	s.imeCtx = old
}

// SetIMECursor says where the caret is, so the candidate list opens
// beside it ([IMESurface]).
//
// In device pixels of the client area, which is the unit every other
// position in this backend is in and what Windows wants here too.
func (s *winSurface) SetIMECursor(x, y, w, h int) {
	if s == nil || s.hwnd == 0 || s.closed {
		return
	}
	himc, _, _ := procImmGetContext.Call(s.hwnd)
	if himc == 0 {
		return
	}
	defer procImmReleaseContext.Call(s.hwnd, himc)

	cf := winCompositionForm{Style: cfsPoint, CurrentPos: winPoint{X: int32(x), Y: int32(y)}}
	procImmSetCompositionWnd.Call(himc, uintptr(unsafe.Pointer(&cf)))

	// CFS_EXCLUDE, not CFS_POINT: it hands Windows the caret's whole
	// rectangle and says "do not cover this", so a candidate list near
	// the foot of the screen opens above the caret instead of over it.
	cand := winCandidateForm{
		Index: 0, Style: cfsExclude,
		CurrentPos: winPoint{X: int32(x), Y: int32(y)},
		Area:       winRect{Left: int32(x), Top: int32(y), Right: int32(x + w), Bottom: int32(y + h)},
	}
	procImmSetCandidateWindow.Call(himc, uintptr(unsafe.Pointer(&cand)))
}

// cancelComposition throws away what is being composed.
func (s *winSurface) cancelComposition() {
	himc, _, _ := procImmGetContext.Call(s.hwnd)
	if himc == 0 {
		return
	}
	procImmNotifyIME.Call(himc, niCompositionStr, cpsCancel, 0)
	procImmReleaseContext.Call(s.hwnd, himc)
	if s.composing {
		s.composing = false
		s.push(Event{Kind: EventIMECancel})
	}
}

// imeString reads one of the composition strings out of the context.
func imeString(himc uintptr, what uintptr) (string, bool) {
	n, _, _ := procImmGetCompositionStrW.Call(himc, what, 0, 0)
	size := int(int32(n))
	if size <= 0 {
		return "", false
	}
	buf := make([]uint16, size/2+1)
	procImmGetCompositionStrW.Call(himc, what,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(size))
	return syscall.UTF16ToString(buf[:size/2]), true
}

// wmImeComposition turns one WM_IME_COMPOSITION into the toolkit's
// events. It returns whether the message was handled here.
//
// A single message can carry both a result and a new composition — an
// input method that commits one word and starts the next on the same
// keystroke does exactly that — so the commit is looked for first and
// the preedit after it, rather than one or the other.
func (s *winSurface) wmImeComposition(lparam uintptr) bool {
	himc, _, _ := procImmGetContext.Call(s.hwnd)
	if himc == 0 {
		return false
	}
	defer procImmReleaseContext.Call(s.hwnd, himc)

	if lparam&gcsResultStr != 0 {
		if txt, ok := imeString(himc, gcsResultStr); ok {
			s.composing = false
			s.push(Event{Kind: EventIMECommit, Text: txt})
		}
	}
	if lparam&gcsCompStr != 0 {
		txt, _ := imeString(himc, gcsCompStr)
		if txt == "" {
			// An empty composition is the method saying it has nothing
			// left: the preedit goes away.
			if s.composing {
				s.composing = false
				s.push(Event{Kind: EventIMECancel})
			}
		} else {
			s.composing = true
			s.push(Event{Kind: EventIMEPreedit, Text: txt, IMECaret: imeCaret(himc, txt)})
		}
	}
	return true
}

// imeCaret is where the caret sits in the preedit, as a byte offset
// into its UTF-8.
//
// GCS_CURSORPOS counts UTF-16 code units, which is the unit Windows
// works in and not the one [Event.IMECaret] is in. The two part company
// on the first character outside the basic plane — which for an input
// method is not a corner case, since that is where the rarer CJK
// characters live.
func imeCaret(himc uintptr, preedit string) int {
	n, _, _ := procImmGetCompositionStrW.Call(himc, gcsCursorPos, 0, 0)
	return utf16ByteOffset(preedit, int(int32(n)))
}

// utf16ByteOffset turns a UTF-16 offset into s into a byte offset into
// its UTF-8. It is out here, away from the input context, because it is
// the part of imeCaret that can be wrong and the only part a test can
// reach without an input method running.
func utf16ByteOffset(s string, units int) int {
	if units <= 0 {
		return 0
	}
	u16 := utf16.Encode([]rune(s))
	if units >= len(u16) {
		return len(s)
	}
	// A caret that landed *between* a surrogate pair belongs before the
	// character, not inside it: decoding half a pair gives U+FFFD and a
	// byte offset into the middle of a rune.
	//
	// The test is for a **high** surrogate at the end of the span, not
	// for any surrogate. A low one there means the span ends after a
	// complete pair, which is a perfectly good place for a caret — and
	// treating that as a split is what this did first, putting the
	// caret one character early after every non-BMP character.
	if u16[units-1] >= 0xD800 && u16[units-1] < 0xDC00 {
		units--
	}
	return len(string(utf16.Decode(u16[:units])))
}
