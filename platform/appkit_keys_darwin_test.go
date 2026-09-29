//go:build darwin && cgo

package platform

import "testing"

// The key is the layout's, before modifiers: what a shortcut is about.
func TestAkKeyIsTheUnshiftedKey(t *testing.T) {
	for _, c := range []struct {
		name string
		r    rune
		want Key
	}{
		{"escape", akEscape, KeyEscape},
		{"tab", akTab, KeyTab},
		{"return", akReturn, KeyReturn},
		{"keypad enter", akEnter, KeyReturn},
		{"the key labelled Delete", akBackspace, KeyBackspace},
		{"forward delete", akFwdDelete, KeyDelete},
		{"left", akLeftArrow, KeyLeft},
		{"right", akRightArrow, KeyRight},
		{"up", akUpArrow, KeyUp},
		{"down", akDownArrow, KeyDown},
		{"home", akHome, KeyHome},
		{"end", akEnd, KeyEnd},
		{"page up", akPageUp, KeyPageUp},
		{"page down", akPageDown, KeyPageDown},
		{"space", akSpace, KeySpace},
		{"menu", akMenuKey, KeyMenu},
		{"a", 'a', KeyA},
		{"z", 'z', KeyZ},
		{"A folds to a", 'A', KeyA},
		{"Z folds to z", 'Z', KeyZ},
		{"3", '3', Key3},
		{"hash", '#', KeyHash},
		{"comma", ',', KeyComma},
		{"F1", akF1, KeyF1},
		{"F12", akF12, KeyF12},
		{"F5", akF1 + 4, KeyF5},
		{"nothing at all", 0, KeyUnknown},
		{"a character with no key", 'ø', KeyUnknown},
	} {
		if got := akKey(c.r); got != c.want {
			t.Errorf("%s (%#04x): key %v, want %v", c.name, c.r, got, c.want)
		}
	}
}

// Apple's Delete key is backspace and its Fn+Delete is forward delete.
// Getting these the wrong way round deletes the wrong character, which
// is the kind of bug nobody reports as a key mapping bug.
func TestAkDeleteKeysAreNotSwapped(t *testing.T) {
	if akKey(akBackspace) == akKey(akFwdDelete) {
		t.Fatal("backspace and forward delete map to the same key")
	}
	if akKey(0x7F) != KeyBackspace {
		t.Errorf("0x7F is %v, want KeyBackspace: it is the key labelled Delete", akKey(0x7F))
	}
}

// Command is Super and Control is Ctrl. This is the mapping the backend
// deliberately does *not* fudge — see akMods — so it is worth a test
// that says so, to make a later change to it a deliberate one.
func TestAkModsAreFaithful(t *testing.T) {
	for _, c := range []struct {
		name  string
		flags uint64
		want  Modifiers
	}{
		{"none", 0, 0},
		{"shift", akModShift, ModShift},
		{"control is Ctrl", akModControl, ModCtrl},
		{"option is Alt", akModOption, ModAlt},
		{"command is Super, not Ctrl", akModCommand, ModSuper},
		{"command and shift", akModCommand | akModShift, ModSuper | ModShift},
		{"all four", akModShift | akModControl | akModOption | akModCommand,
			ModShift | ModCtrl | ModAlt | ModSuper},
		// Caps Lock is a modifier the toolkit knows, because a
		// passphrase prompt has to be able to say it is on. It is a
		// *lock* rather than a chord key: Chord() drops it, so a
		// shortcut still matches with Caps Lock down.
		{"caps lock alone", akModCapsLock, ModCapsLock},
		{"command with caps lock", akModCommand | akModCapsLock, ModSuper | ModCapsLock},
		// Function and the rest are flags too, and none of those is a
		// modifier the toolkit knows.
		{"function alone", 1 << 23, 0},
	} {
		if got := akMods(c.flags); got != c.want {
			t.Errorf("%s: mods %v, want %v", c.name, got, c.want)
		}
	}
	// And a chord is the chord whatever the locks are doing.
	if got := akMods(akModCommand | akModCapsLock).Chord(); got != ModSuper {
		t.Errorf("Chord() with caps lock = %v, want just Super", got)
	}
}

// What a text field should be given, and what it should not.
func TestAkTextTakesOnlyText(t *testing.T) {
	for _, c := range []struct {
		name  string
		chars string
		flags uint64
		want  string
	}{
		{"a letter", "s", 0, "s"},
		{"a shifted letter", "S", akModShift, "S"},
		{"option types a glyph", "ø", akModOption, "ø"},
		{"option-e-e composes", "é", akModOption, "é"},
		{"space is text", " ", 0, " "},
		{"return is not", "\r", 0, ""},
		{"tab is not", "\t", 0, ""},
		{"escape is not", "\x1b", 0, ""},
		{"backspace is not", "\x7f", 0, ""},
		{"control-s is not", "\x13", akModControl, ""},
		{"an arrow is not", string(rune(akLeftArrow)), 0, ""},
		{"F1 is not", string(rune(akF1)), 0, ""},
		{"command-s is a shortcut, not an s", "s", akModCommand, ""},
		{"command-shift-s likewise", "S", akModCommand | akModShift, ""},
		{"nothing", "", 0, ""},
		{"a surrogate pair arrives whole", "\U0001F600", 0, "\U0001F600"},
	} {
		if got := akText(c.chars, c.flags); got != c.want {
			t.Errorf("%s: text %q, want %q", c.name, got, c.want)
		}
	}
}

// Option is how a Mac types ø, so it must not suppress text the way
// Command does. This is the pair the filter is most likely to confuse.
func TestAkOptionTypesButCommandDoesNot(t *testing.T) {
	if got := akText("ø", akModOption); got != "ø" {
		t.Errorf("Option+o gave %q, want the character it types", got)
	}
	if got := akText("o", akModCommand); got != "" {
		t.Errorf("Command+o gave %q, want nothing: it is a shortcut", got)
	}
}
