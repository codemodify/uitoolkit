package app

import "github.com/codemodify/uitoolkit/style"

// packBuilt reports whether pack name is in this build. Theme engines are
// chosen at build time (docs/engines.md), so a test that walks a table of
// packs skips the rows this build has not got.
func packBuilt(name string) bool {
	_, ok := style.LoadTheme(name)
	return ok
}
