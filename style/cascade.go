package style

import "sync"

// The theme cascade.
//
// A desktop saves one appearance in look.json; an application may want
// another; a pane inside that application may want a third. CSS answers
// this with a cascade — the nearest rule wins, everything it does not
// mention is inherited — and so does this.
//
// A [ThemeOverride] is one level of it: the parts an application, a
// subtree or a single control states for itself. Everything it leaves
// empty is inherited from the level above, and the levels are, from
// weakest to strongest:
//
//	look.json                      the desktop's saved appearance
//	app.Options.Theme              the application's own
//	any component's SetTheme       a subtree's
//	  … and again, nested          the nearest one wins
//	widget.Base.SetLook            an exact look, past the cascade
//
// **What cascades, and what does not.** CSS cascades every property
// independently because every property is independent. Here they are
// not: a pack is a bevel language, a palette, metrics, an engine and an
// era's typography that were drawn to go together, and half of Metal
// over half of Luna is not a theme, it is a bug. So the unit of a level
// is the *pack*, whole — and beside it the four preferences the toolkit
// already treats as independent of the pack, because a user sets them
// across packs in Settings: the corner policy, the icon set, the icon
// size and the two typefaces. Those five cascade one by one, the way
// CSS properties do. Nothing else does:
//
//   - **The display scale is not in here at all.** It belongs to the
//     monitor, not to the application and not to a pane. A derived look
//     is built from the look above it, which already carries the
//     window's scale and density, so a scope *cannot* change it: there
//     is no field for it and no path to one.
//   - Reduced motion, the desktop's file dialogs and the combo wheel are
//     process-wide behaviour, not appearance.
//   - Who draws the window frame, where its caption buttons go and which
//     device paints the surface belong to the window and to the
//     compositor, which need one answer per window rather than one per
//     widget.
//
// See docs/themes.md, "The cascade".

// ThemeOverride is one level of the theme cascade: the parts of an
// appearance a level states for itself. The zero value states nothing
// and inherits everything, so a widget that was never given a theme
// costs exactly what it did before the cascade existed.
//
// Every field is empty for "inherit". Where a field's own vocabulary has
// a word for "the pack's own" — [CornersTheme] is "theme", and "theme"
// (or "pack", or "default") is what the font chooser calls the pack's
// era typeface — that word is how a level takes the pack's own back from
// something a level above pinned.
type ThemeOverride struct {
	// Pack is a theme pack id ([ListThemes], [LoadTheme]): "luna",
	// "metal-ocean", "win95". It brings its palette with it.
	Pack string
	// Corners is the radius policy: "theme" for the pack's own shape,
	// "round", "square".
	Corners CornerStyle
	// Icons and IconSize are the chrome icon set and its draw size.
	Icons    IconSetName
	IconSize IconSize
	// FontUI and FontMono pin the two typefaces. "theme" takes the
	// pack's era face back from a family a level above pinned.
	FontUI, FontMono string
}

// Empty reports whether t states nothing, and so inherits everything.
func (t ThemeOverride) Empty() bool { return t == ThemeOverride{} }

// Over is t with base's parts filled in where t states nothing: t is the
// nearer level and wins field by field. It is how a nested scope is
// flattened without building a look for each level in between.
func (t ThemeOverride) Over(base ThemeOverride) ThemeOverride {
	if t.Pack == "" {
		t.Pack = base.Pack
	}
	if t.Corners == "" {
		t.Corners = base.Corners
	}
	if t.Icons == "" {
		t.Icons = base.Icons
	}
	if t.IconSize == "" {
		t.IconSize = base.IconSize
	}
	if t.FontUI == "" {
		t.FontUI = base.FontUI
	}
	if t.FontMono == "" {
		t.FontMono = base.FontMono
	}
	return t
}

// On is the appearance a inherited from the level above with t's own
// parts written over it. What t leaves empty comes back untouched —
// including the preferences that never cascade (the frame, the caption
// buttons, the renderer, reduced motion), which pass through so that
// [On] can be applied to a whole application appearance.
//
// Naming a pack stops the result following the desktop's light / dark
// preference: an application that asks for "luna" asked for Luna, not
// for whichever of Luna and Royale Noir matches the session.
func (t ThemeOverride) On(a Appearance) Appearance {
	if t.Empty() {
		return a
	}
	if t.Pack != "" {
		a.Name = t.Pack
		a.FollowDesktop = false
		if p, ok := LoadTheme(t.Pack); ok {
			a.Theme = p.Palette
		}
	}
	if t.Corners != "" {
		a.Corners = ParseCorners(string(t.Corners))
	}
	if t.Icons != "" {
		a.Icons = ParseIconSet(string(t.Icons))
	}
	if t.IconSize != "" {
		a.IconSize = ParseIconSize(string(t.IconSize))
	}
	if t.FontUI != "" {
		a.FontUI = NormalizeFontChoice(t.FontUI)
	}
	if t.FontMono != "" {
		a.FontMono = NormalizeFontChoice(t.FontMono)
	}
	return a
}

// themedKey is one derived look: the look it was derived from and what
// was written over it. Both halves are comparable, so the memo is a
// plain map.
type themedKey struct {
	base LookAndFeel
	over ThemeOverride
}

// themedLimit is how many derived looks are kept before the memo is
// dropped and filled again. A window holds a handful — one per themed
// subtree per scale — and Settings' browser walks 131 packs through a
// preview; the cap is what keeps the second case from pinning every
// pack's fonts, atlases and engine memo in memory for the life of the
// process.
const themedLimit = 64

var (
	themedMu    sync.Mutex
	themedLooks map[themedKey]LookAndFeel
)

// Themed is base with t written over it: a whole [LookAndFeel], built
// from the merged appearance, keeping base's display scale and density.
//
// It is the one place the cascade turns into a look, and it is memoized
// on (base, t): a subtree resolves to the *same* look pointer every
// time, which is what lets everything keyed on a look — an engine's
// derived paint data, a popup's shadow patch, a shaped window's
// silhouette — be shared rather than rebuilt per widget.
//
// base keeps its display scale because the merged appearance is applied
// with [WithAppearance], which rebuilds metrics at the scale and density
// the look already has. There is no way to pass another one, by design:
// see the note at the top of this file.
//
// A look that is not a [Classic] cannot be derived from and comes back
// unchanged.
func Themed(base LookAndFeel, t ThemeOverride) LookAndFeel {
	if base == nil || t.Empty() {
		return base
	}
	if _, ok := base.(*Classic); !ok {
		return base
	}
	// A level that states what the look above already says is not a
	// level at all: the subtree keeps the *same* look, and with it every
	// cache keyed on that pointer. A tab that asks for the pack the
	// window is already in costs nothing.
	was := LookAppearance(base)
	if t.On(was) == was {
		return base
	}
	key := themedKey{base: base, over: t}
	themedMu.Lock()
	if l, ok := themedLooks[key]; ok {
		themedMu.Unlock()
		return l
	}
	themedMu.Unlock()

	l := WithAppearance(base, t.On(was))

	themedMu.Lock()
	if themedLooks == nil || len(themedLooks) >= themedLimit {
		themedLooks = make(map[themedKey]LookAndFeel, themedLimit)
	}
	// Another goroutine may have built the same look while the lock was
	// down; either is correct, and keeping the one already published
	// keeps the pointer stable for whoever took it.
	if had, ok := themedLooks[key]; ok {
		themedMu.Unlock()
		return had
	}
	themedLooks[key] = l
	themedMu.Unlock()
	return l
}
