package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// An application can tint its own chrome, and the engine follows.
//
// Every browser and editor paints its own title bar and tool bar — it is
// how Chromium wears the desktop's accent — and none could do it here:
// the surfaces were a pack's to state and an application's to live with.
// The look-like-chromium sample came out neutral grey beside a real
// Chromium whose tab strip was #D2E2FC, and nothing in the API closed
// that gap.
func TestAnApplicationCanTintItsOwnChrome(t *testing.T) {
	lk := Appearance{Name: DefaultThemeName, Theme: ThemeLight}.Look()
	before := lk.X("titleBar", paintengine2d.Color{})

	blue := paintengine2d.RGB(0.82, 0.886, 0.988)
	tinted, _ := WithChrome(lk, map[string]paintengine2d.Color{"titleBar": blue}).(*Classic)
	if tinted == nil {
		t.Fatal("WithChrome did not return a look")
	}
	if got := tinted.X("titleBar", paintengine2d.Color{}); got != blue {
		t.Errorf("titleBar is %v, want the stated %v", got, blue)
	}
	if lk.X("titleBar", paintengine2d.Color{}) != before {
		t.Error("tinting changed the look it was given — it must return a new one")
	}

	// Everything else about the look survives: a tint is a tint, not a
	// different pack.
	if tinted.Pack() != lk.Pack() || tinted.Palette().Background != lk.Palette().Background {
		t.Error("tinting changed the pack or its palette")
	}

	// A zero colour means "leave the pack's", so a caller may build the
	// map from fields that are not all set.
	z, _ := WithChrome(lk, map[string]paintengine2d.Color{"titleBar": {}}).(*Classic)
	if got := z.X("titleBar", paintengine2d.Color{}); got != before {
		t.Errorf("a zero colour overwrote the pack's %v with %v", before, got)
	}

	// Safe to wrap unconditionally.
	if WithChrome(nil, map[string]paintengine2d.Color{"a": blue}) != nil {
		t.Error("a nil look came back non-nil")
	}
	if got := WithChrome(lk, nil); got != lk {
		t.Error("an empty tint should return the very same look, so caches keyed on it survive")
	}
}
