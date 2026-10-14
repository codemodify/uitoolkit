//go:build !font_complex && !font_emoji

package style

// complexLayout is the shaping path, which this build has not got: the
// caller lays the string out a rune at a time instead. See
// docs/fonts.md for what the font_complex tag adds and what it costs.
func complexLayout(*Font, string) (shapedRun, bool) { return shapedRun{}, false }

// shapes reports whether this build can draw a script that joins or
// reorders. It cannot, which is what makes [borrowable] the right rule
// here: lending Arabic letterforms a glyph at a time draws them
// unjoined, and a box is the honester answer.
func shapes() bool { return false }
