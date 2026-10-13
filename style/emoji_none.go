//go:build !font_emoji

package style

import "github.com/codemodify/paintengine2d"

// isColourSheet is false in every build without font_emoji: a glyph
// sheet is a coverage mask there, and [Font.Draw] tints all of them.
func isColourSheet(*paintengine2d.FontAtlas) bool { return false }
