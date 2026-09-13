package platform

import (
	"time"
	"unicode/utf8"
)

// KeyFromKeysymFallback maps a keysym to a toolkit [Key], falling back to
// the layout-independent keysym when the active layout has no mapping.
//
// sym is the keysym the active group/level produced; base is the same
// physical key evaluated at group 0, level 0 (US-QWERTY-ish on every sane
// keymap). With a Cyrillic or Greek layout active, Ctrl+C reports
// Cyrillic_es / Greek_psi, which no toolkit Key covers — shortcuts and
// accelerators would silently die. Qt, GTK, and Chromium all resolve
// shortcuts against the layout-independent symbol for exactly this reason.
func KeyFromKeysymFallback(sym, base uint64) Key {
	if k := KeyFromKeysym(sym); k != KeyUnknown {
		return k
	}
	if base == 0 || base == sym {
		return KeyUnknown
	}
	return KeyFromKeysym(base)
}

// latin1ToUTF8 re-encodes ISO-8859-1 bytes as UTF-8.
//
// XLookupString (the no-XIC X11 path) returns Latin-1, not UTF-8: byte
// 0xE9 means é. Feeding those bytes to a UTF-8 decoder yields U+FFFD, so
// every accented character typed without a running input method arrived
// as a replacement char. Bytes < 0x80 are already correct UTF-8, so an
// ASCII-only buffer is returned unchanged.
func latin1ToUTF8(b []byte) []byte {
	ascii := true
	for _, c := range b {
		if c >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return b
	}
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = utf8.AppendRune(out, rune(c))
	}
	return out
}

// ShouldArmKeyRepeat reports whether a pressed key starts the Wayland
// auto-repeat timer. Modifiers, and any key the keymap marks as
// non-repeating, must not: holding Shift to extend a selection otherwise
// produced a KeyDown stream at the repeat rate and woke the run loop
// 30+ times a second. rate/delay come from wl_keyboard.repeat_info; a
// zero rate means the compositor disabled repeat entirely.
func ShouldArmKeyRepeat(keyRepeats bool, rate, delay int) bool {
	return keyRepeats && rate > 0 && delay > 0
}

// repeatInterval is the gap between synthetic repeats for a rate in keys
// per second, clamped so a hostile rate cannot spin the loop.
func repeatInterval(rate int) time.Duration {
	if rate <= 0 {
		return 0
	}
	d := time.Second / time.Duration(rate)
	if d < 10*time.Millisecond {
		d = 10 * time.Millisecond
	}
	if d > time.Second {
		d = time.Second
	}
	return d
}
