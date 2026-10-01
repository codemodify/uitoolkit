package style

import "github.com/codemodify/paintengine2d"

// WithChrome is lk with its window surfaces tinted by name — the title
// bar, the tool bar, the sidebar — exactly as a pack's theme.json "extra"
// map states them, and read back by engines through [Classic.X].
//
// It is how an application themes its own chrome, which every browser and
// editor does and none could do here. Chromium paints its tab strip in
// the desktop's accent and its tool bar a shade lighter, and fills the
// *selected tab* with the tool bar's colour so the two meet and read as
// one surface. An application states those two and the engine does the
// rest, because it was already reading them from the pack:
//
//	lk = style.WithChrome(lk, map[string]paintengine2d.Color{
//	    "titleBar": accent,
//	    "toolBar":  style.Mix(accent, white, 0.6),
//	})
//
// The names are the pack's own, so they are as engine-specific as a
// pack's art is. "titleBar", "toolBar", "window", "sidebar", "card" and
// "popup" are read by every engine built on the web look; an engine that
// does not know a name simply never asks for it, which is why an unknown
// one is ignored rather than an error.
//
// A tint carries its own ink. Where a surface is stated and the ink on it
// is not, the engine derives one with [ReadableInk]: the palette's text,
// walked away from the surface until it reads. So tinting a tab strip a
// deep blue does not leave a dark label stranded on it, and an untinted
// pack is unaffected, because the palette's text already reads on the
// palette's own surfaces.
//
// Derived is not always *right*, though — only safe. Firefox's tab strip
// takes the desktop's title-bar colour and its labels take the desktop's
// title-bar foreground, which is white whether or not dark would have
// read. That is a fact about the desktop, not about contrast, so an
// application that knows it states the ink beside the surface:
//
//	"titleBar": accent, "titleBarText": paintengine2d.RGB(1, 1, 1),
//
// "titleBarText" and "toolBarText" are the two the web look reads.
//
// It returns lk unchanged for an empty map, for a look this package did
// not make, and for a nil look — so it is safe to wrap unconditionally.
func WithChrome(lk LookAndFeel, tint map[string]paintengine2d.Color) LookAndFeel {
	if lk == nil || len(tint) == 0 {
		return lk
	}
	c, ok := lk.(*Classic)
	if !ok {
		return lk
	}
	tok := c.Tokens()
	extra := make(map[string]paintengine2d.Color, len(tok.Extra)+len(tint))
	for k, v := range tok.Extra {
		extra[k] = v
	}
	for k, v := range tint {
		// A zero colour means "leave the pack's", so that a caller can
		// build the map from fields some of which are unset without
		// having to prune it first.
		if v == (paintengine2d.Color{}) {
			continue
		}
		extra[k] = v
	}
	tok.Extra = extra
	return newClassic(c.Name(), c.Palette(), c.Metrics(), c.Corners(), c.Icons(), c.IconSize(), tok).
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale()).carry(c)
}
