//go:build !font_fallback && !font_complex

package style

// The default build borrows nothing: one face draws everything, and a
// rune it has not got is a box. [SetFallbackFamilies] still takes its
// names — an application that sets them need not know which build it is
// in — and nothing reads them.
//
// Build with `-tags font_fallback` for the other behaviour. It costs no
// bytes; what it costs is that text measurement then depends on the
// faces the machine has, which is why it is a choice and not the
// default.
func fallbackFor(*Font, rune) *Font { return nil }

// fallbackChanged has nothing to throw away: nothing was decided.
func fallbackChanged() {}
