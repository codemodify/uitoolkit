package style

import "sync"

// Resetting what the installed-font index feeds.
//
// fakeFonts rebuilds the index from lines of its own, which is only
// half the job: several caches are *derived* from it and outlive it.
// A test that hands the index a colour emoji face leaves that face
// parsed and remembered, and the next test — pinned to the bundled
// faces, expecting a drawn paperclip — gets the real one instead. It
// passed alone and failed in the suite, twice in one evening.
//
// The tagged builds add their own caches to this list from an init.
var derivedFontCacheResets []func()

func resetDerivedFontCaches() {
	systemOTFaces.mu.Lock()
	systemOTFaces.m = map[sysFaceKey]*otFace{}
	systemOTFaces.mu.Unlock()
	for _, reset := range derivedFontCacheResets {
		reset()
	}
}

// realFonts runs a test against the machine's own installed faces.
//
// The suite pins them off (UITK_SYSTEM_FONTS=0) so that a pixel is the
// same number everywhere, and that is right for almost everything —
// but it is also why three defects in two days were found by
// applications and not here: a keycap taken by an outline face, a
// lender inside a .ttc, an emoji family that only one machine's font
// set could break. A test that needs the real set says so, resets the
// index around itself as fakeFonts does, and skips where the machine
// has nothing to say.
func realFonts(t interface {
	Setenv(string, string)
	Cleanup(func())
	Helper()
}) {
	t.Helper()
	t.Setenv(SystemFontsEnv, "1")
	sysIndex.once = sync.Once{}
	sysIndex.faces, sysIndex.names = nil, nil
	resetDerivedFontCaches()
	t.Cleanup(func() {
		sysIndex.once = sync.Once{}
		sysIndex.faces, sysIndex.names = nil, nil
		resetDerivedFontCaches()
	})
}
