package platform

import "testing"

func TestKeyFromKeysym(t *testing.T) {
	if KeyFromKeysym(0xffe9) != KeyAlt || KeyFromKeysym(0xffea) != KeyAlt {
		t.Fatal("Alt_L / Alt_R")
	}
	if KeyFromKeysym(0xff1b) != KeyEscape {
		t.Fatal("Escape")
	}
	if KeyFromKeysym('a') != KeyA || KeyFromKeysym('A') != KeyA {
		t.Fatal("A")
	}
	if KeyFromKeysym(0xff0d) != KeyReturn {
		t.Fatal("Return")
	}
	if KeyFromKeysym(0x20) != KeySpace {
		t.Fatal("space")
	}
	if KeyFromKeysym(0x33) != Key3 || KeyFromKeysym(0x23) != KeyHash {
		t.Fatal("hash")
	}
}
