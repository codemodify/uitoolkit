//go:build !font_complex && !font_emoji

package style

// DirectionRuns is nil in a build with no bidirectional algorithm in it:
// every string reads one way, which is what this toolkit has always
// assumed. See docs/fonts.md for the font_complex tag.
func DirectionRuns(string) []DirectionRun { return nil }
