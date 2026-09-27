package platform

import "testing"

// Every backend answers for itself. The offscreen one is the interesting
// answer: no desktop behind it, but it places windows, because it is
// pretending and nothing contradicts it. The old code could not say that
// — "offscreen" was one word meaning both.
func TestBackendCapsAreDeclaredNotGuessedFromTheName(t *testing.T) {
	c := OffscreenBackend{}.Caps()
	if c.Has(BackendDesktop) {
		t.Error("the offscreen backend has no desktop behind it")
	}
	if !c.Has(BackendScreenPlace) {
		t.Error("the offscreen backend places windows where they ask")
	}
	if got := c.String(); got != "screen-place" {
		t.Errorf("caps print as %q", got)
	}
	if got := BackendCaps(0).String(); got != "none" {
		t.Errorf("no capabilities print as %q", got)
	}
	if got := (BackendDesktop | BackendScreenPlace).String(); got != "desktop screen-place" {
		t.Errorf("both print as %q", got)
	}
}

// A nil backend does nothing. An application without one is headless
// before it has opened anything, and the three callers that used to read
// `a.backend == nil || a.backend.Name() == "offscreen"` now ask one
// question instead of two.
func TestBackendCapsOfNilIsNothing(t *testing.T) {
	if got := BackendCapsOf(nil); got != 0 {
		t.Fatalf("a nil backend answered %v", got)
	}
	if BackendCapsOf(nil).Has(BackendDesktop) {
		t.Fatal("a nil backend must not claim a desktop")
	}
}

// Has is all-of, not any-of: asking for two capabilities and getting one
// is no.
func TestBackendCapsHasIsAllOf(t *testing.T) {
	c := BackendDesktop
	if c.Has(BackendDesktop | BackendScreenPlace) {
		t.Fatal("Has must want every bit asked for")
	}
	if !c.Has(BackendDesktop) || !c.Has(0) {
		t.Fatal("Has must grant the bits that are there")
	}
}

// ScreenPlacementAvailable follows the capability rather than the
// backend's name, and the test hook still overrides it.
func TestScreenPlacementFollowsTheCapability(t *testing.T) {
	// The default backend under test is offscreen, which places.
	if !ScreenPlacementAvailable() {
		t.Error("the offscreen desktop places windows")
	}
	restore := SimulateScreenPlacement(false)
	if ScreenPlacementAvailable() {
		t.Error("the simulation hook no longer overrides the answer")
	}
	restore()
	if !ScreenPlacementAvailable() {
		t.Error("the answer did not come back")
	}
}
