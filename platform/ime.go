package platform

import "unicode/utf8"

// IMECaretRect is a caret box in the focused widget's local pixels.
type IMECaretRect struct {
	X, Y, W, H int
}

// IMESurface is implemented by backends that drive an OS input method.
type IMESurface interface {
	SetIMECursor(x, y, w, h int)
	// SetIMEEnabled turns the OS IME on when a text widget has focus.
	// Wayland uses this for zwp_text_input_v3 enable/disable instead of
	// enabling on every keyboard enter (which starved EventText on
	// compositors that enter text-input without ever committing).
	SetIMEEnabled(on bool)
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

// ShouldEmitXKBText reports whether a Wayland key should produce EventText
// from xkb/compose. zwp_text_input_v3 being entered (textActive) is not a
// reason to suppress: many compositors never send commit or preedit for
// Latin, which left TextFields with KeyDown only (Add Account, Quick Filter).
// Skip only while an IME preedit is in progress, or for key-up / Ctrl.
func ShouldEmitXKBText(pressed, ctrl, imeComposing bool) bool {
	return pressed && !ctrl && !imeComposing
}

// PairXKBText records xkb UTF-8 and drops it when IME already committed the
// same string for this key (IME-then-key order).
func PairXKBText(utf8, lastIME string) (emit, lastXKB, lastIMEOut string) {
	if utf8 == "" {
		return "", "", lastIME
	}
	if utf8 == lastIME {
		return "", "", ""
	}
	return utf8, utf8, ""
}

// PairIMECommit drops IME commit text that xkb already inserted as EventText
// (key-then-IME order on Latin compositors that both emit utf8 and commit).
func PairIMECommit(commit, lastXKB string) (emit, lastXKBOut, lastIME string) {
	if commit == "" {
		return "", lastXKB, ""
	}
	if commit == lastXKB {
		return "", "", ""
	}
	return commit, "", commit
}

// textEvents turns UTF-8 bytes into EventText for each printable rune.
func textEvents(b []byte, mods Modifiers) []Event {
	var out []Event
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r >= 32 && r != 127 {
			out = append(out, Event{Kind: EventText, Rune: r, Mods: mods})
		}
		if size < 1 {
			break
		}
		b = b[size:]
	}
	return out
}
