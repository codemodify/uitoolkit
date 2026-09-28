//go:build darwin && cgo

package platform

/*
#include "appkit_darwin.h"
*/
import "C"

// Input methods on AppKit: the view is an NSTextInputClient, so a
// Japanese, Pinyin or Hangul method composes into the window the way it
// does into any Mac application, and a dead key on a US-International
// layout produces the character it is for.
//
// The shape of it is the same as the Linux backends': the platform
// reports preedit and commit, and the toolkit does the editing. What
// differs is where the text comes from. Before this, keyDown: read
// [event characters] and pushed it as EventText — which is right for
// plain typing and wrong for everything else, because with an input
// method active those characters are the *raw keystrokes*: a user
// typing にほん would have got "nihon" inserted as they went.
//
// Now keyDown: sends the key and then hands the event to
// -interpretKeyEvents:, and the text arrives from -insertText: or
// -setMarkedText: afterwards. The key is still sent first and
// unconditionally, because a shortcut, an arrow or Escape is about the
// key whatever the input method then does with the event.

// SetIMEEnabled turns the input method on for this window
// ([IMESurface]).
//
// macOS has no per-window switch — there is only which responder the
// input context is serving — so this activates or deactivates that
// context, and on the way off discards anything half-composed. Without
// that a preedit abandoned in one field reappears in the next.
func (s *akSurface) SetIMEEnabled(on bool) {
	if s == nil || s.win == nil || s.Closed() {
		return
	}
	C.uitk_ak_set_ime_enabled(s.win, cbool(on))
}

// SetIMECursor says where the caret is, so the candidate window opens
// beside it rather than in the corner of the screen ([IMESurface]).
//
// In device pixels of the content area, y down, which is the unit
// every other position in this backend is in; the Objective-C side
// turns it into the screen rectangle -firstRectForCharacterRange:
// answers with.
func (s *akSurface) SetIMECursor(x, y, w, h int) {
	if s == nil || s.win == nil || s.Closed() {
		return
	}
	C.uitk_ak_set_ime_cursor(s.win, C.int(x), C.int(y), C.int(w), C.int(h))
}

var _ IMESurface = (*akSurface)(nil)
