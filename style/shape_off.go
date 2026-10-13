//go:build !font_complex

package style

// complexLayout is the shaping path, which this build has not got: the
// caller lays the string out a rune at a time instead. See
// docs/fonts.md for what the font_complex tag adds and what it costs.
func complexLayout(*Font, string) (shapedRun, bool) { return shapedRun{}, false }
