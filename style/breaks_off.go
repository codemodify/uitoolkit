//go:build !font_complex

package style

// lineBreaks is nil in a build with no line-breaking algorithm in it:
// [Font.Wrap] then breaks at spaces, which is the toolkit's contract
// (docs/contracts.md) and is wrong only for scripts that do not use
// them. See docs/fonts.md for the font_complex tag.
func lineBreaks([]rune) []bool { return nil }
