package style

// Helpers that more than one engine needs, and that therefore belong to
// none of them.
//
// They are here because of what they are *for*, not where they were first
// written. snap lived in engine_win95.go, which made 1001 call sites
// across 36 other engines — and core — depend on the Windows 95 engine
// being compiled in. It is a pixel-grid rounding function; it has nothing
// to do with Windows 95, and while it lived there no build could leave
// win95 out.
//
// The rule, the same one the engine tests follow: **anything more than one
// engine needs belongs to all of them.** A helper that stays in the engine
// that happened to need it first is a build-time dependency nobody
// intended, and the only way to find them is to exclude an engine and read
// what the compiler says is missing.
//
// This file carries no build tag, so everything here is in every build.

// snap rounds to the pixel grid so 1px lines stay crisp.
func snap(v float32) float32 {
	if v < 0 {
		return float32(int(v - 0.5))
	}
	return float32(int(v + 0.5))
}
