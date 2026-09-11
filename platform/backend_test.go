package platform

import "testing"

func TestDefaultHeadlessIsOffscreen(t *testing.T) {
	if Default(true).Name() != "offscreen" {
		t.Fatalf("got %q", Default(true).Name())
	}
	if Select("x11", true).Name() != "offscreen" {
		t.Fatal("headless select")
	}
	if Select("offscreen", false).Name() != "offscreen" {
		t.Fatal("explicit offscreen")
	}
}
