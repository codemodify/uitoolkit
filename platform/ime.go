package platform

import "unicode/utf8"

// IMECaretRect is a caret box in the focused widget's local pixels.
type IMECaretRect struct {
	X, Y, W, H int
}

// IMESurface is implemented by backends that drive an OS input method.
type IMESurface interface {
	SetIMECursor(x, y, w, h int)
}

// DesktopSurface is optional chrome control (EWMH / xdg-shell).
type DesktopSurface interface {
	SetFullscreen(on bool)
	SetMaximized(on bool)
}

// SetFullscreen asks the window manager / compositor when the surface
// implements DesktopSurface.
func SetFullscreen(s Surface, on bool) {
	if d, ok := s.(DesktopSurface); ok {
		d.SetFullscreen(on)
	}
}

// SetMaximized asks the window manager / compositor when supported.
func SetMaximized(s Surface, on bool) {
	if d, ok := s.(DesktopSurface); ok {
		d.SetMaximized(on)
	}
}

// ApplyPreeditDraw applies an XIM-style incremental preedit update.
// chgFirst/chgLen are rune indices into old; insert replaces that span.
func ApplyPreeditDraw(old string, chgFirst, chgLen int, insert string) string {
	runes := []rune(old)
	n := len(runes)
	if chgFirst < 0 {
		chgFirst = 0
	}
	if chgFirst > n {
		chgFirst = n
	}
	end := chgFirst + chgLen
	if end < chgFirst {
		end = chgFirst
	}
	if end > n {
		end = n
	}
	return string(runes[:chgFirst]) + insert + string(runes[end:])
}

// DeleteSurroundingUTF8 removes before/after bytes around a rune caret.
// Returns the new string and the new rune caret.
func DeleteSurroundingUTF8(s string, caretRunes, beforeBytes, afterBytes int) (string, int) {
	if caretRunes < 0 {
		caretRunes = 0
	}
	runes := []rune(s)
	if caretRunes > len(runes) {
		caretRunes = len(runes)
	}
	head := string(runes[:caretRunes])
	tail := string(runes[caretRunes:])
	for beforeBytes > 0 && head != "" {
		_, sz := utf8.DecodeLastRuneInString(head)
		if sz <= 0 {
			break
		}
		head = head[:len(head)-sz]
		beforeBytes -= sz
		caretRunes--
	}
	for afterBytes > 0 && tail != "" {
		_, sz := utf8.DecodeRuneInString(tail)
		if sz <= 0 {
			break
		}
		tail = tail[sz:]
		afterBytes -= sz
	}
	if caretRunes < 0 {
		caretRunes = 0
	}
	return head + tail, caretRunes
}

// ComposeVisual inserts preedit at caret for painting. selA/selB cover the
// preedit span so LookAndFeel can highlight it; visCaret is inside that span.
func ComposeVisual(text string, caret int, preedit string, preeditCaret int) (vis string, visCaret, selA, selB int) {
	runes := []rune(text)
	if caret < 0 {
		caret = 0
	}
	if caret > len(runes) {
		caret = len(runes)
	}
	if preedit == "" {
		return text, caret, caret, caret
	}
	pre := []rune(preedit)
	if preeditCaret < 0 {
		preeditCaret = 0
	}
	if preeditCaret > len(pre) {
		preeditCaret = len(pre)
	}
	vis = string(runes[:caret]) + preedit + string(runes[caret:])
	selA, selB = caret, caret+len(pre)
	visCaret = caret + preeditCaret
	return vis, visCaret, selA, selB
}

// InsertAtRune inserts s at rune index i.
func InsertAtRune(text string, i int, s string) string {
	runes := []rune(text)
	if i < 0 {
		i = 0
	}
	if i > len(runes) {
		i = len(runes)
	}
	return string(runes[:i]) + s + string(runes[i:])
}
