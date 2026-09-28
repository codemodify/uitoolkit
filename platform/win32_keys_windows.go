//go:build windows

package platform

// Win32 keyboard input: virtual-key codes to [Key], and WM_CHAR to the
// printable runes [EventText] carries.
//
// The split is Windows' own and it matches the toolkit's. WM_KEYDOWN
// names a *key* — a position on the keyboard, before any layout,
// modifier or dead key has had its say — and is what a shortcut, an
// arrow or Escape is about. WM_CHAR names a *character*, after the
// layout, the Shift, the AltGr and any dead-key composition, and is what
// a text field wants. Both are emitted for a letter, which is correct:
// Ctrl+S reads the first and inserts nothing, a plain "s" reads the
// second. The pump already calls TranslateMessage, which is what turns
// the first into the second.

var procGetKeyState = user32.NewProc("GetKeyState")

const (
	vkBack     = 0x08
	vkTab      = 0x09
	vkReturn   = 0x0D
	vkShift    = 0x10
	vkControl  = 0x11
	vkMenu     = 0x12 // Alt
	vkEscape   = 0x1B
	vkSpace    = 0x20
	vkPrior    = 0x21 // Page Up
	vkNext     = 0x22 // Page Down
	vkEnd      = 0x23
	vkHome     = 0x24
	vkLeft     = 0x25
	vkUp       = 0x26
	vkRight    = 0x27
	vkDown     = 0x28
	vkDelete   = 0x2E
	vkLWin     = 0x5B
	vkRWin     = 0x5C
	vkApps     = 0x5D // the context-menu key
	vkF1       = 0x70
	vkF12      = 0x7B
	vkOemComma = 0xBC
)

// winKey maps a virtual-key code to the toolkit's key.
func winKey(vk uintptr) Key {
	switch vk {
	case vkEscape:
		return KeyEscape
	case vkTab:
		return KeyTab
	case vkReturn:
		return KeyReturn
	case vkBack:
		return KeyBackspace
	case vkDelete:
		return KeyDelete
	case vkLeft:
		return KeyLeft
	case vkRight:
		return KeyRight
	case vkUp:
		return KeyUp
	case vkDown:
		return KeyDown
	case vkHome:
		return KeyHome
	case vkEnd:
		return KeyEnd
	case vkPrior:
		return KeyPageUp
	case vkNext:
		return KeyPageDown
	case vkSpace:
		return KeySpace
	case vkApps:
		return KeyMenu
	case vkMenu:
		// Either Alt on its own: a window shows its mnemonic underlines
		// while it is held, which is why the key has a name at all.
		return KeyAlt
	case '3':
		return Key3
	case vkOemComma:
		return KeyComma
	}
	if vk >= 'A' && vk <= 'Z' {
		// Virtual-key codes for letters are the ASCII capitals, and are
		// the *unshifted* key whatever the Shift state — which is what
		// KeyA..KeyZ mean.
		return KeyA + Key(vk-'A')
	}
	if vk >= vkF1 && vk <= vkF12 {
		return KeyF1 + Key(vk-vkF1)
	}
	return KeyUnknown
}

// winMods reads the modifier keys.
//
// GetKeyState rather than the message's own wParam: a key message says
// nothing about which modifiers were down with it, and GetKeyState
// answers for the message being dispatched rather than for this instant,
// so it stays right even when the queue runs behind the hardware.
func winMods() Modifiers {
	down := func(vk uintptr) bool {
		r, _, _ := procGetKeyState.Call(vk)
		return r&0x8000 != 0
	}
	var m Modifiers
	if down(vkShift) {
		m |= ModShift
	}
	if down(vkControl) {
		m |= ModCtrl
	}
	if down(vkMenu) {
		m |= ModAlt
	}
	if down(vkLWin) || down(vkRWin) {
		m |= ModSuper
	}
	return m
}

// winChar turns a WM_CHAR into the rune it stands for, or false for one
// that is not text.
//
// wParam is a UTF-16 code unit, not a rune: anything outside the basic
// plane arrives as two messages, a high surrogate and then a low one,
// and only the pair means anything. The first is held on the surface
// until the second comes.
//
// Control characters are dropped. Windows sends WM_CHAR for Escape,
// Return, Tab and every Ctrl+letter as well, and those are keys rather
// than text — a text field that took them would insert a glyph for
// Ctrl+S. The same filter the other backends use: a printable rune is
// >= 32 and not 127.
func (s *winSurface) winChar(w uintptr) (rune, bool) {
	u := uint16(w)
	switch {
	case u >= 0xD800 && u <= 0xDBFF: // high surrogate: wait for its pair
		s.hiSurrogate = u
		return 0, false
	case u >= 0xDC00 && u <= 0xDFFF: // low surrogate
		hi := s.hiSurrogate
		s.hiSurrogate = 0
		if hi == 0 {
			return 0, false
		}
		return rune(hi-0xD800)<<10 | rune(u-0xDC00) + 0x10000, true
	}
	s.hiSurrogate = 0
	r := rune(u)
	if r < 32 || r == 127 {
		return 0, false
	}
	return r, true
}
