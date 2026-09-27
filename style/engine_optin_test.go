package style

import (
	"strings"
	"testing"
)

// Theme engines are opt-in at build time, and the default is everything.
//
//	go build ./...                                     all engines (unchanged)
//	go build -tags theme_engines_pick ./...            only the always-built ones
//	go build -tags "theme_engines_pick,\
//	                theme_engine_motif,theme_engine_aqua" ./...   plus those two
//
// The default matters more than the mechanism. An opt-in scheme where a
// plain build gives you nothing fails the wrong way: a consumer who
// forgets the flag gets a toolkit with no themes and no compile error to
// say so. Here a forgotten flag gives you *more* than you asked for,
// never less, and only a deliberate theme_engines_pick takes anything
// away.
//
// Build tags cannot contain hyphens — //go:build theme-engine-motif is a
// syntax error — so the names use underscores.

// A theme naming an engine that was not built must fall back, not fail.
// Packs name their engine by string (theme.json "engine"), so the string
// can outlive the code: that is the whole point of leaving an engine out.
func TestUnbuiltEngineFallsBackToBase(t *testing.T) {
	if _, ok := EngineByID("no-such-engine-was-ever-built"); ok {
		t.Fatal("an engine that does not exist was found")
	}
	e := engineFor(ThemeTokens{Engine: "no-such-engine-was-ever-built"})
	if e == nil {
		t.Fatal("a token set naming an unbuilt engine resolved to nothing")
	}
	if e.ID() != baseEngine.ID() {
		t.Fatalf("fell back to %q, want the base engine %q", e.ID(), baseEngine.ID())
	}
}

// The always-built engines are the ones the core itself reaches into —
// core files use snap and winSnap and w95 from win95 and aero, and the
// skin and base engines outright — so they carry no tag and no
// theme_engines_pick build can drop them. If this list ever shrinks, the
// engines above it became separable and can be tagged.
func TestAlwaysBuiltEnginesAreThere(t *testing.T) {
	for _, id := range []string{"base"} {
		if _, ok := EngineByID(id); !ok {
			t.Errorf("%q must be built into every configuration", id)
		}
	}
	if len(EngineIDs()) == 0 {
		t.Fatal("no engines at all")
	}
	if len(ListThemes()) == 0 {
		t.Fatal("no theme packs at all")
	}
}

// Every id is lower case and has no spaces, because a build tag is made
// from it (theme_engine_<id>) and a pack names it in JSON.
func TestEngineIDsAreTagSafe(t *testing.T) {
	for _, id := range EngineIDs() {
		if id != strings.ToLower(id) || strings.ContainsAny(id, " \t") {
			t.Errorf("engine id %q is not tag-safe", id)
		}
	}
}

// needEngine skips the test when engine id was not built into this
// configuration.
//
// Asking for the *pack* is not enough, and the difference is the
// interesting part: core carries legacy era packs that an engine
// overrides when it is present (see [RegisterPack]). Leave the engine
// out and the name still resolves — to the legacy pack, painted by the
// base engine. That is the intended degradation, a rougher version of
// the theme rather than a missing one, but it means a test that wants
// what the *engine* draws has to say so.
func needEngine(t *testing.T, id string) {
	t.Helper()
	if _, ok := EngineByID(id); !ok {
		t.Skipf("the %q engine is not in this build (theme_engine_%s)", id, id)
	}
}

// packBuilt reports whether pack name is in this build. Tests that walk a
// table of packs use it to skip the rows whose engine was left out, so
// the same table serves every configuration instead of one table per
// build.
func packBuilt(name string) bool { return packRegistered(name) }

// packRegistered reports whether name is a pack this build registered,
// by that exact name. It is stricter than LoadTheme, which follows
// aliases and so can answer yes by resolving to a different pack.
func packRegistered(name string) bool {
	for _, n := range AllBuiltinThemeNames() {
		if n == name {
			return true
		}
	}
	return false
}
