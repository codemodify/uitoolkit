package style

import "github.com/codemodify/paintengine2d"

// Sprites an app paints itself.
//
// A skin binds its art to a fixed table of parts, and that is the whole of
// what the engine paints from it. An app that lays out a fixed panel of its
// own — a player's display, with its digits, its indicators and its wells —
// has no part to ask for, and until now had no way to reach the art at all:
// it could only paint in the palette's colours and hope the skin's pictures
// agreed with them.
//
// These two functions are that way. They name a sprite by the manifest's own
// name and paint it by exactly the rules a part is painted by — the asset
// chosen upward, nine-slice with whole-pixel edges, nearest sampling at
// whole multiples for a pixelated sheet, the tint for a glyph — so a panel an
// app assembles from sprites is as sharp at 1.75 as a skinned button is.
//
// They add nothing to the format. A sprite nobody binds was always allowed;
// it is simply one only an app that knows its name will paint. Everything
// else a skin promises still holds: a look that is not a skin, or a skin
// without that sprite, answers false and the app paints its own, which is
// what makes a panel drawn this way a partial override like everything else.

// DrawSkinSprite paints the named sprite of the look's skin into b.
//
// tint colours a sprite the manifest marks "tint" (a white glyph drawn in the
// caller's ink); it is ignored for any other sprite, and the zero colour
// leaves a tinted sprite white. It reports false, having painted nothing,
// when the look is not a skin or its skin has no sprite by that name.
func DrawSkinSprite(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, name string, tint paintengine2d.Color) bool {
	l, sk := lookSkin(lk)
	if sk == nil || ctx == nil {
		return false
	}
	sp := sk.Sprites[name]
	if sp == nil {
		return false
	}
	return sk.skinDraw(ctx, b, sp, l.Scale(), tint)
}

// SkinSpriteSize is the named sprite's size in the skin's design pixels, and
// whether the look's skin has it at all. An app that sets a line of text in
// a skin's own bitmap alphabet measures its advances with this.
func SkinSpriteSize(lk LookAndFeel, name string) (w, h float32, ok bool) {
	_, sk := lookSkin(lk)
	if sk == nil {
		return 0, 0, false
	}
	sp := sk.Sprites[name]
	if sp == nil {
		return 0, 0, false
	}
	s := sk.Design.Scale
	if s <= 0 {
		s = 1
	}
	return sp.W / s, sp.H / s, true
}

// lookSkin is the look as the engine sees it and the skin it wears, or nil
// for a look that is not a skin.
func lookSkin(lk LookAndFeel) (*Classic, *Skin) {
	l, ok := classicOf(lk)
	if !ok {
		return nil, nil
	}
	return l, skinFor(l)
}
