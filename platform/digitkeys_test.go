package platform

import "testing"

// Ctrl+1 and Ctrl+2 are how every browser steps to its first and
// second tab, and for years this toolkit had `Key3` and no others: it,
// `#` and `,` are the three characters an accelerator happened to need
// once, and the enum grew by demand. A password manager asked for its
// three parts on Ctrl+1, 2 and 3, found that only the third worked,
// and settled for Ctrl+Page Up and Down.
func TestEveryDigitIsAKey(t *testing.T) {
	for r := '0'; r <= '9'; r++ {
		k := DigitKey(r)
		if k == KeyUnknown {
			t.Errorf("%q is no key", r)
			continue
		}
		if got := KeyChar(k); got != r {
			t.Errorf("%q maps to a key whose character is %q", r, got)
		}
		if got, ok := DigitRune(k); !ok || got != r {
			t.Errorf("DigitRune of %q gave %q, %v", r, got, ok)
		}
	}
	if DigitKey('a') != KeyUnknown || DigitKey('/') != KeyUnknown {
		t.Error("a letter or a slash came back as a digit")
	}
	// In order, so Key0+n is the nth — Key3 used to sit on its own
	// between KeySpace and KeyHash.
	for n := 0; n <= 9; n++ {
		if Key0+Key(n) != DigitKey(rune('0'+n)) {
			t.Fatalf("the digits are not in order at %d", n)
		}
	}
	if Key3 != Key0+3 {
		t.Error("Key3 is not the fourth digit")
	}
}

// The keysym table serves both Linux backends: X11 reads it from the
// server and Wayland from xkbcommon, and both come here. The keypad's
// digits are the same keys — Num Lock is the person's business, not a
// shortcut's.
func TestTheKeysymTableKnowsTheDigits(t *testing.T) {
	for n := 0; n <= 9; n++ {
		want := Key0 + Key(n)
		if got := keyFromKeysym(uint64(0x0030 + n)); got != want {
			t.Errorf("keysym for %d gave %v, want %v", n, got, want)
		}
		if got := keyFromKeysym(uint64(0xffb0 + n)); got != want {
			t.Errorf("the keypad's %d gave %v, want %v", n, got, want)
		}
	}
	// And the three that were there before it.
	if keyFromKeysym(0x0023) != KeyHash || keyFromKeysym(0x002c) != KeyComma || keyFromKeysym(0x0020) != KeySpace {
		t.Error("a character key that worked before does not now")
	}
}
