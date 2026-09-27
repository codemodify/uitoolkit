package platform

import "testing"

func TestKeyFromKeysym(t *testing.T) {
	if keyFromKeysym(0xffe9) != KeyAlt || keyFromKeysym(0xffea) != KeyAlt {
		t.Fatal("Alt_L / Alt_R")
	}
	if keyFromKeysym(0xff1b) != KeyEscape {
		t.Fatal("Escape")
	}
	if keyFromKeysym('a') != KeyA || keyFromKeysym('A') != KeyA {
		t.Fatal("A")
	}
	if keyFromKeysym(0xff0d) != KeyReturn {
		t.Fatal("Return")
	}
	if keyFromKeysym(0x20) != KeySpace {
		t.Fatal("space")
	}
	if keyFromKeysym(0x33) != Key3 || keyFromKeysym(0x23) != KeyHash {
		t.Fatal("hash")
	}
}
