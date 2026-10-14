package style

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
