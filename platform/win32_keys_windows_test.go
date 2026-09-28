//go:build windows

package platform

import "testing"

// The keyboard translation, which needs no window and no desktop.

func TestWinKeyMapsTheKeysAToolkitNames(t *testing.T) {
	for _, c := range []struct {
		vk   uintptr
		want Key
		why  string
	}{
		{0x1B, KeyEscape, "VK_ESCAPE"},
		{0x09, KeyTab, "VK_TAB"},
		{0x0D, KeyReturn, "VK_RETURN"},
		{0x08, KeyBackspace, "VK_BACK"},
		{0x2E, KeyDelete, "VK_DELETE"},
		{0x25, KeyLeft, "VK_LEFT"},
		{0x28, KeyDown, "VK_DOWN"},
		{0x21, KeyPageUp, "VK_PRIOR is Page Up, not Print"},
		{0x22, KeyPageDown, "VK_NEXT is Page Down"},
		{0x20, KeySpace, "VK_SPACE"},
		{0x5D, KeyMenu, "VK_APPS is the context-menu key"},
		{0x12, KeyAlt, "VK_MENU is Alt, despite the name"},
		{'A', KeyA, "letters are their ASCII capitals"},
		{'Z', KeyZ, "and the last of them"},
		{'3', Key3, ""},
		{0xBC, KeyComma, "VK_OEM_COMMA"},
		{0x70, KeyF1, "VK_F1"},
		{0x7B, KeyF12, "VK_F12"},
		{0x5B, KeyUnknown, "the Windows key is a modifier, not a key"},
		{0x00, KeyUnknown, ""},
	} {
		if got := winKey(c.vk); got != c.want {
			t.Errorf("winKey(%#x) = %v, want %v %s", c.vk, got, c.want, c.why)
		}
	}
	// Every letter, in order: an off-by-one here would silently swap the
	// whole alphabet and no single case above would show it.
	for i := uintptr(0); i < 26; i++ {
		if got, want := winKey('A'+i), KeyA+Key(i); got != want {
			t.Fatalf("winKey(%q) = %v, want %v", rune('A'+i), got, want)
		}
	}
}

// WM_CHAR is a UTF-16 code unit, and the ones that are not text have to be
// dropped: Windows sends it for Escape, Return, Tab and every Ctrl+letter
// as well, and a field that took them would insert a glyph for Ctrl+S.
func TestWinCharKeepsTextAndDropsControlCodes(t *testing.T) {
	s := &winSurface{}
	for _, c := range []struct {
		in   uintptr
		want rune
		ok   bool
	}{
		{'a', 'a', true},
		{' ', ' ', true},
		{'~', '~', true},
		{0x00E9, 'é', true},   // é, straight from a layout
		{0x20AC, '€', true},   // AltGr on many layouts
		{0x1B, 0, false},      // Escape
		{0x0D, 0, false},      // Return
		{0x09, 0, false},      // Tab
		{0x08, 0, false},      // Backspace
		{0x13, 0, false},      // Ctrl+S
		{0x7F, 0, false},      // Delete
	} {
		got, ok := s.winChar(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("winChar(%#04x) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

// Anything outside the basic plane arrives as two messages, and only the
// pair means anything.
func TestWinCharPairsSurrogates(t *testing.T) {
	s := &winSurface{}
	if r, ok := s.winChar(0xD83D); ok || r != 0 {
		t.Fatalf("a lone high surrogate produced %q, %v: it has to wait for its pair", r, ok)
	}
	r, ok := s.winChar(0xDE00)
	if !ok || r != 0x1F600 {
		t.Fatalf("the pair D83D DE00 gave %q (%#x), %v; want U+1F600", r, r, ok)
	}
	// A low surrogate with nothing before it is not text, and must not
	// leave the pairing state armed for the next keystroke.
	if r, ok := s.winChar(0xDE00); ok || r != 0 {
		t.Fatalf("an orphan low surrogate produced %q, %v", r, ok)
	}
	if r, ok := s.winChar('x'); !ok || r != 'x' {
		t.Fatalf("after an orphan surrogate, an ordinary key gave %q, %v", r, ok)
	}
}

// The measuring style decides how much room the frame takes, and a client
// frame takes none: the whole window is the client area (WM_NCCALCSIZE).
func TestWinAdjustStyleMeasuresNoFrameForAClientFrame(t *testing.T) {
	if got := winAdjustStyle(DecorationsClient); got != wsPopup {
		t.Errorf("a client frame measures with %#x, want WS_POPUP (%#x) so AdjustWindowRectEx adds nothing", got, uintptr(wsPopup))
	}
	for _, d := range []Decorations{DecorationsServer, DecorationsAuto} {
		if got := winAdjustStyle(d); got != wsOverlappedWindow {
			t.Errorf("decorations %v measure with %#x, want WS_OVERLAPPEDWINDOW", d, got)
		}
	}
}
