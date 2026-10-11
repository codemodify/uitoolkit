//go:build theme_engine_all || theme_engine_macos_tahoe

package style

import "github.com/codemodify/paintengine2d"

// Lumen is the toolkit's own glass, as against a reproduction of
// somebody's.
//
// [tahoeEngine] already draws the right *shapes* — capsules of chrome, a
// floating sidebar, rounded everything — because that is what a glass
// interface is made of. What it does not draw is a glass *material*, and
// it should not: macOS 26's tool bar glass is clear and its menus are
// 85% opaque over a backdrop blurred flat, and Tahoe is a reproduction
// of macOS 26. Rendered over content, chrome that clear is an outline
// you read straight through.
//
// Lumen is the same engine with a material on it. Three numbers do all
// of it, and all three have to move together or the effect disappears:
//
//   - **a tint you can see** — a third of an alpha rather than an
//     eighth, so a label on the chrome stays legible over whatever
//     scrolls under it;
//   - **a blur that frosts without flattening** — a few pixels, not
//     fourteen. Fourteen leaves a flat colour, and a flat colour cannot
//     be bent; a few leaves the structure and gains the legibility;
//   - **an edge with thickness** — which Tahoe's chrome already has
//     (paintengine2d's BackdropRefract), and which is the thing
//     that makes a pane read as glass rather than as a frosted sticker.
//
// None of it is visible without content behind it. A pane over a flat
// window background has nothing to frost, nothing to bend and nothing to
// be translucent of, whatever these numbers say — which is a layout
// decision (widgets.FloatOver) rather than a palette one, and the
// reason this pack ships with a sample rather than alone.
//
// It rides with the Tahoe engine's build tag: a pack is data, and this
// one is Tahoe's own tokens with its glass keys moved.

// lumenGlass is the material, over the pack it is built from.
func lumenGlass(dark bool) map[string]string {
	if dark {
		return map[string]string{
			// A third of an alpha of white: enough to carry a label over
			// a photograph, little enough to be glass.
			"glass": "#ffffff52",
			// The specular, brighter than Tahoe's because there is more
			// surface for it to run along.
			"glassRim":  "#ffffff8c",
			"glassRim2": "#ffffff1a",
			"glassEdge": "#00000073",
			// Menus lighten too: at Tahoe's d9 over a backdrop blurred
			// flat, a menu is a grey rectangle with a rim.
			"menu": "#2c2c2ebf",
		}
	}
	return map[string]string{
		"glass":     "#ffffffa6",
		"glassRim":  "#ffffff",
		"glassRim2": "#ffffff73",
		"glassEdge": "#00000026",
		"menu":      "#f6f6f8c9",
	}
}

// lumenPack is tahoePack with the glass material over it, and the two
// blur radii that go with it.
//
// The material goes in Tokens.Extra, which is the seam the engine reads
// a pack's overrides through (Classic.X, falling back to the engine's
// own table). Handing tahoePack a different base map does nothing: the
// engine does not read the pack's map, it reads its own — which is the
// first way this was written and rendered identically to Tahoe.
func lumenPack(name, label, summary string, fam ThemeName, base map[string]string) ThemePack {
	p := tahoePack(name, label, summary, fam, base)
	if p.Tokens.Extra == nil {
		p.Tokens.Extra = map[string]paintengine2d.Color{}
	}
	for k, v := range lumenGlass(fam == ThemeDark) {
		p.Tokens.Extra[k] = Hex(v)
	}
	p.Lineage, p.Era, p.Year = "uitoolkit", "Glass", 2026
	// Seven pixels frosts the chrome without flattening it; eight does
	// the same for a menu, where Tahoe uses fourteen and leaves nothing
	// behind it to see.
	p.Tokens.Params["glassBlur"] = 7
	p.Tokens.Params["menuBlur"] = 8
	return p
}

// Two shapes of the same material.
//
// Round is the capsule: every pane a pill, which is the shape Apple
// chose for Liquid Glass and carries all the way down to a two-button
// tool group. Square is the same glass with the corner only taken off
// ([CornersSoft], a 3px hint) — a rectangle that does not come to a
// point, which is where most interfaces actually sit and which a palette
// of two positions had no way to say.
//
// They differ in [ThemePack.Corners] and nothing else: same engine, same
// tint, same blur, same bevel. That is the point of the pair — it is a
// shape preference, not a second look.
func lumenPacks() []ThemePack {
	const (
		roundLight = "The toolkit's own glass, in Apple's shape: every pane a capsule. Chrome you see the content through, frosted a little, bent at its edges, lit from wherever the pointer is."
		roundDark  = "Lumen over charcoal: capsules of glass, a third of an alpha of white over whatever scrolls beneath them."
		edgeLight  = "Lumen with the corner only taken off: the same glass, drawn as rectangles with a 3px hint rather than pills."
		edgeDark   = "Lumen Square over charcoal: the same glass as the capsules, squared off to a hint."
	)
	round := []ThemePack{
		lumenPack("lumen", "Lumen", roundLight, ThemeLight, tahoeLight),
		lumenPack("lumen-night", "Lumen Dark", roundDark, ThemeDark, tahoeDark),
	}
	square := []ThemePack{
		lumenPack("lumen-square", "Lumen Square", edgeLight, ThemeLight, tahoeLight),
		lumenPack("lumen-square-night", "Lumen Square Dark", edgeDark, ThemeDark, tahoeDark),
	}
	for i := range square {
		square[i].Corners = CornersSoft
	}
	return append(round, square...)
}

func init() {
	for _, p := range lumenPacks() {
		RegisterPack(p)
	}
}
