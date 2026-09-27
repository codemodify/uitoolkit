package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// needPack skips the test when pack is not in this build. Theme engines
// are chosen at build time (docs/engines.md), so a test written about one
// engine's pack has nothing to assert in a build that left it out — and
// saying so is not the same as failing.
func needPack(t *testing.T, name string) {
	t.Helper()
	if _, ok := style.LoadTheme(name); !ok {
		t.Skipf("the %q pack is not in this build", name)
	}
}

// packBuilt reports whether pack name is in this build, for a test that
// walks a table of packs and wants to skip the rows it has not got.
func packBuilt(name string) bool {
	_, ok := style.LoadTheme(name)
	return ok
}
