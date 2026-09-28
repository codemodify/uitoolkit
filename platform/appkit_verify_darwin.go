//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "appkit_darwin.h"
*/
import "C"

import (
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// Looking at what a window is showing, and putting input into it.
//
// These are here rather than in the test beside them because Go does
// not allow cgo in a _test.go file: a test that needs C has to reach it
// through the package. They are unexported, and nothing in the toolkit
// calls them — appkit_window_darwin_test.go is the only caller.
//
// They are worth having anyway. On macOS a test cannot see a window the
// ordinary way: reading the screen back wants Screen Recording
// permission, which a run over ssh does not have, and screencapture(1)
// does not fail without it — it returns a clean desktop with every
// window silently missing, which reads exactly like a window that never
// opened. That cost a diagnostic detour once. A layer render needs no
// permission and checks what a present can actually get wrong.

// akReadback is what the window's layer is showing, w by h device
// pixels; nil when it could not be read.
func akReadback(s *akSurface, w, h int) *paintengine2d.Image {
	if s == nil || s.win == nil || w <= 0 || h <= 0 {
		return nil
	}
	img := paintengine2d.NewImage(w, h)
	if len(img.Pix) == 0 {
		return nil
	}
	if C.uitk_ak_readback(s.win, (*C.uchar)(unsafe.Pointer(&img.Pix[0])),
		C.int(w), C.int(h)) == 0 {
		return nil
	}
	return img
}

// akPostKey puts a synthetic key in the application's own queue. chars
// is what it types and bare is the key before modifiers, which is the
// same split -[UitkView keyDown:] reads off a real NSEvent.
func akPostKey(s *akSurface, chars, bare string, mods Modifiers, down bool) {
	if s == nil || s.win == nil {
		return
	}
	c, b := C.CString(chars), C.CString(bare)
	defer C.free(unsafe.Pointer(c))
	defer C.free(unsafe.Pointer(b))
	d := 0
	if down {
		d = 1
	}
	C.uitk_ak_post_key(s.win, c, b, C.uint64_t(akFlags(mods)), C.int(d))
}

// akPostMouse puts a synthetic press, release or move in the queue, at
// x, y in AppKit's window coordinates: points, y up from the bottom
// left. Not the toolkit's convention, on purpose — see the header.
func akPostMouse(s *akSurface, kind EventKind, x, y float64, button MouseButton, mods Modifiers) {
	if s == nil || s.win == nil {
		return
	}
	var k C.int
	switch kind {
	case EventMouseDown:
		k = C.UITK_AK_MOUSE_DOWN
	case EventMouseUp:
		k = C.UITK_AK_MOUSE_UP
	default:
		k = C.UITK_AK_MOUSE_MOVE
	}
	C.uitk_ak_post_mouse(s.win, k, C.double(x), C.double(y),
		C.int(button), C.uint64_t(akFlags(mods)))
}

// akIMEMark, akIMEInsert and akIMEUnmark drive the view's
// NSTextInputClient the way an input method would. A real one cannot
// be scripted from a test, and this is the path it takes.
func akIMEMark(s *akSurface, text string, caretUTF16 int) {
	akIME(s, 1, text, caretUTF16)
}
func akIMEInsert(s *akSurface, text string) { akIME(s, 2, text, 0) }
func akIMEUnmark(s *akSurface)              { akIME(s, 3, "", 0) }

func akIME(s *akSurface, what int, text string, caret int) {
	if s == nil || s.win == nil {
		return
	}
	c := C.CString(text)
	defer C.free(unsafe.Pointer(c))
	C.uitk_ak_ime_simulate(s.win, C.int(what), c, C.int(caret))
}

// akContentSize is what AppKit says the content area is, in device
// pixels — the number the buffer is supposed to match.
func akContentSize(s *akSurface) (int, int) {
	if s == nil || s.win == nil {
		return 0, 0
	}
	var w, h C.int
	C.uitk_ak_content_size(s.win, &w, &h)
	return int(w), int(h)
}

// akFlags is akMods backwards: the NSEventModifierFlags that would
// produce these modifiers, so a test can say ModSuper and mean Command.
func akFlags(m Modifiers) uint64 {
	var f uint64
	if m&ModShift != 0 {
		f |= akModShift
	}
	if m&ModCtrl != 0 {
		f |= akModControl
	}
	if m&ModAlt != 0 {
		f |= akModOption
	}
	if m&ModSuper != 0 {
		f |= akModCommand
	}
	return f
}
