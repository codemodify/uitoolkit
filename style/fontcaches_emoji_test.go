//go:build font_emoji

package style

import gofont "github.com/go-text/typesetting/font"

func init() {
	derivedFontCacheResets = append(derivedFontCacheResets, func() {
		emojiFaces.mu.Lock()
		emojiFaces.m = map[string]*gofont.Face{}
		emojiFaces.mu.Unlock()

		emojiSheets.mu.Lock()
		emojiSheets.m = map[emojiKey]*emojiAtlas{}
		emojiSheets.mu.Unlock()
	})
}
