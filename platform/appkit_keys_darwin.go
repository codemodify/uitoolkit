//go:build darwin && cgo

package platform

// AppKit keyboard input: an NSEvent to [Key], and its characters to the
// runes [EventText] carries.
//
// The split is the same one Windows makes and the toolkit keeps: a key
// is a position on the keyboard before any modifier or dead key has had
// its say, and is what a shortcut, an arrow or Escape is about; a
// character is what comes out after the layout, the Shift and the
// Option, and is what a text field wants. AppKit hands both on the one
// event — charactersIgnoringModifiers and characters — so unlike Win32
// there is no second message to wait for.
//
// The key comes from charactersIgnoringModifiers rather than from
// NSEvent's keyCode, which is a hardware position. A Dvorak typist
// pressing the key labelled S should get KeyS, and does: the layout has
// already been applied to charactersIgnoringModifiers, and the
// modifiers, which is the whole point, have not. Win32's virtual-key
// codes are layout-mapped in exactly the same way, so the two backends
// mean the same thing by KeyA.
//
// The keys that are not characters at all arrive there too, as the
// private-use code points AppKit reserves for them (NSUpArrowFunctionKey
// and its neighbours, 0xF700 up). They are listed below rather than
// imported from C so that this file, like its Win32 twin, is pure Go and
// can be tested without a window.

// AppKit's function-key code points (NSEvent.h). Private-use area, so
// they can never collide with a character a layout produces.
const (
	akUpArrow    = 0xF700
	akDownArrow  = 0xF701
	akLeftArrow  = 0xF702
	akRightArrow = 0xF703
	akF1         = 0xF704
	akF12        = 0xF70F
	akFwdDelete  = 0xF728
	akHome       = 0xF729
	akEnd        = 0xF72B
	akPageUp     = 0xF72C
	akPageDown   = 0xF72D
	akMenuKey    = 0xF735
)

// The control characters AppKit sends for the keys that predate the
// private-use block.
const (
	akEscape    = 0x1B
	akTab       = 0x09
	akReturn    = 0x0D
	akEnter     = 0x03 // the keypad's, and Control-Return
	akBackspace = 0x7F // the key labelled Delete on an Apple keyboard
	akSpace     = 0x20
)

// NSEventModifierFlags (NSEvent.h). Listed rather than imported for the
// same reason as the key codes.
const (
	// NSEventModifierFlagCapsLock and NumericPad. Caps Lock is a latch
	// and reports while it is on; NumericPad is set while a key from the
	// numeric keypad is involved, which is the nearest thing macOS has
	// to a Num Lock state — the Mac has had no Num Lock key since the
	// Extended Keyboard.
	akModCapsLock = 1 << 16
	akModShift    = 1 << 17
	akModControl  = 1 << 18
	akModOption   = 1 << 19
	akModCommand  = 1 << 20
	akModNumLock  = 1 << 21
)

// akKey maps a code point from charactersIgnoringModifiers to the
// toolkit's key. 0 (an event with no characters at all, which a modifier
// press is) gives KeyUnknown.
func akKey(r rune) Key {
	switch r {
	case akEscape:
		return KeyEscape
	case akTab:
		return KeyTab
	case akReturn, akEnter:
		return KeyReturn
	case akBackspace:
		return KeyBackspace
	case akFwdDelete:
		return KeyDelete
	case akLeftArrow:
		return KeyLeft
	case akRightArrow:
		return KeyRight
	case akUpArrow:
		return KeyUp
	case akDownArrow:
		return KeyDown
	case akHome:
		return KeyHome
	case akEnd:
		return KeyEnd
	case akPageUp:
		return KeyPageUp
	case akPageDown:
		return KeyPageDown
	case akSpace:
		return KeySpace
	case akMenuKey:
		return KeyMenu
	case '3':
		return Key3
	case '#':
		return KeyHash
	case ',':
		return KeyComma
	}
	// Letters are the unshifted key, which is what KeyA..KeyZ mean, so an
	// upper-case character from a Shifted or Caps-Locked keyboard folds
	// back down. charactersIgnoringModifiers drops the Shift for most
	// layouts, but not for Caps Lock, which it keeps.
	if r >= 'A' && r <= 'Z' {
		r += 'a' - 'A'
	}
	if r >= 'a' && r <= 'z' {
		return KeyA + Key(r-'a')
	}
	if r >= akF1 && r <= akF12 {
		return KeyF1 + Key(r-akF1)
	}
	return KeyUnknown
}

// akMods reads an NSEvent's modifierFlags.
//
// Faithfully: Command is ModSuper and Control is ModCtrl, because this
// layer says what was pressed and those are what was pressed. That a Mac
// user reaches for Command where the same shortcut is Ctrl elsewhere is
// true and important, and it is a question about what a *shortcut* is,
// not about what the keyboard did — answering it here, by handing the
// toolkit a ModCtrl nobody pressed, would make Mods lie to everything
// else that reads them.
func akMods(flags uint64) Modifiers {
	var m Modifiers
	if flags&akModShift != 0 {
		m |= ModShift
	}
	if flags&akModControl != 0 {
		m |= ModCtrl
	}
	if flags&akModOption != 0 {
		m |= ModAlt
	}
	if flags&akModCommand != 0 {
		m |= ModSuper
	}
	if flags&akModCapsLock != 0 {
		m |= ModCapsLock
	}
	if flags&akModNumLock != 0 {
		m |= ModNumLock
	}
	return m
}

// akText is the text an NSEvent's characters insert, or "" for a key
// that is not text.
//
// The same filter the other backends use — a printable rune is >= 32 and
// not 127 — which drops Escape, Tab, Return and every Control+letter,
// all of which AppKit puts in characters and none of which a text field
// should insert. AppKit's function keys fall out too: they live in the
// private-use area, and a text field asked to insert one would show the
// empty box the font has there.
//
// Command held means a shortcut rather than typing, and produces no
// text. Option held does not: Option is how a Mac types ø and é, and the
// character it produces is exactly what the field should get.
func akText(chars string, flags uint64) string {
	if flags&akModCommand != 0 {
		return ""
	}
	out := make([]rune, 0, len(chars))
	for _, r := range chars {
		if r < 32 || r == 127 || (r >= 0xF700 && r <= 0xF8FF) {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
