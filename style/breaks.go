package style

// LineBreaks marks every rune a line may legally break *before*, by the
// Unicode line breaking algorithm (UAX #14), for a wrapper that cannot
// use [Font.Wrap] — a text editor, which has to keep the offsets of
// every line it makes.
//
// It is nil in a build without font_complex, and nil for text whose
// only break opportunities are the ones a space already gives. Both
// mean "carry on with the space rule", so a caller tests it once and
// keeps its own cheap path.
func LineBreaks(runes []rune) []bool { return lineBreaks(runes) }
