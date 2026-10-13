//go:build (font_complex || font_emoji) && !font_emoji

package style

import (
	gofont "github.com/go-text/typesetting/font"

	"github.com/codemodify/paintengine2d"
)

// A build that shapes but was not asked for colour emoji. A rune no
// outline face has is the box it always was, whatever colour faces the
// machine has installed: see docs/fonts.md for the font_emoji tag.
//
// The type is declared so that the layout can name one sheet kind
// without a tag of its own; nothing ever makes one.

type emojiAtlas struct{}

func emojiSheetFor(*Font, rune) (*gofont.Face, *emojiAtlas) { return nil, nil }

func (*emojiAtlas) atlas() *paintengine2d.FontAtlas              { return nil }
func (*emojiAtlas) ensureGIDs([]uint32) *paintengine2d.FontAtlas { return nil }
