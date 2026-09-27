package widget

import (
	"sync/atomic"

	"github.com/codemodify/uitoolkit/style"
)

// The theme cascade, in the tree.
//
// [style.ThemeOverride] says what one level of the cascade states;
// this file is where a component carries one and where a component's
// look is worked out from the levels above it. The rule is CSS's: the
// nearest level wins, and a level inherits what it does not state.
//
// **The hot path.** Look is called several times per widget per layout
// and per paint — every draw call asks the look for its palette, its
// metrics and its fonts — so resolution cannot walk the tree. It does
// not: each component remembers the look it resolved to, and the memo
// is thrown away wholesale by bumping one process-wide counter whenever
// anything that could change an answer changes (a window's look, a
// component's theme, a reparent). A resolve is then a nil check, a load
// and a compare, and a subtree nested a hundred scopes deep answers in
// the same time as one at the root: each node resolves once against its
// parent's already-resolved answer, so the whole tree costs one pass,
// not one pass per node.
//
// The counter is deliberately blunt. A finer scheme — invalidating only
// the subtree that changed — would save a single re-resolve per node
// after a theme switch, at the price of a correctness bug every time a
// mutation site is forgotten. A stale look is a wrong-looking window
// across 131 packs; a redundant re-resolve is a few nanoseconds once.

// lookGen is bumped whenever any component could resolve to a different
// look. Everything that reads it is on the UI goroutine; it is atomic
// because [LooksChanged] may be called from a look watcher's callback
// before it posts, and because the race detector should stay quiet.
var lookGen atomic.Uint64

// LooksChanged throws away every component's memoized look, so the next
// [Component.Look] works it out again. Call it after anything that
// changes what a component would resolve to and that this package
// cannot see for itself — an application or a window swapping its look,
// a scope switching the look it imposes.
//
// It is one atomic increment: it costs nothing to call it too often, and
// a great deal to call it too rarely.
func LooksChanged() { lookGen.Add(1) }

// themed is implemented by Base, so a free function can reach a
// component's own level of the cascade without [Component] having to
// carry it (and without colliding with a widget that already has a
// method called SetTheme, such as widgets.ThemeScope).
type themed interface {
	themeSlot() **style.ThemeOverride
}

func (b *Base) themeSlot() **style.ThemeOverride { return &b.theme }

// SetTheme gives c its own level of the cascade: c and everything under
// it draw in the pack t names, in the corners, icons and typefaces it
// names, and inherit from the window — or from the nearest scope above
// — whatever t leaves empty. The zero [style.ThemeOverride] clears it.
//
// It works on any component, because every component embeds [Base]:
// one button, a panel, the content of a tab. Most callers can write
// c.SetTheme(…) directly; this is for a Component held behind the
// interface, and for the one widget whose own SetTheme means something
// else.
//
//	widget.SetTheme(tabs.Page(), style.ThemeOverride{Pack: "metal-ocean"})
func SetTheme(c Component, t style.ThemeOverride) {
	s, ok := c.(themed)
	if !ok {
		return
	}
	if t.Empty() {
		*s.themeSlot() = nil
	} else {
		cp := t
		*s.themeSlot() = &cp
	}
	LooksChanged()
	c.Invalidate()
	if h := c.Host(); h != nil {
		// Packs disagree about control heights, paddings and radii, so a
		// theme is a relayout and not only a repaint.
		h.RequestLayout()
	}
}

// ThemeOf is the level c states for itself — not the level it resolves
// to, which is [Component.Look]. The zero value means c states nothing.
func ThemeOf(c Component) style.ThemeOverride {
	if s, ok := c.(themed); ok {
		if t := *s.themeSlot(); t != nil {
			return *t
		}
	}
	return style.ThemeOverride{}
}

// SetTheme gives this component and its subtree their own level of the
// cascade. See the free function [SetTheme], which is the same thing for
// a Component held behind the interface.
func (b *Base) SetTheme(t style.ThemeOverride) { SetTheme(b.me(), t) }

// resolveLook is the component's look: the whole cascade, memoized.
//
// The order, strongest first:
//
//  1. an exact look put on this component with [Base.SetLook] — the way
//     past the cascade, for a preview that must show one particular
//     look (widgets.ThemeScope) and for a popup handed the look of the
//     widget that opened it;
//  2. this component's own [style.ThemeOverride], written over the look
//     that reaches it from above;
//  3. the parent's look — parents before the host, so a subtree can run
//     in a theme of its own;
//  4. the window's;
//  5. a dark Classic, for a component that is not in a window yet.
//
// Levels compose by composition rather than by flattening: a scope
// derives from its parent's *resolved* look, which may itself be
// derived, so a scope inside a scope inside a scope is three one-step
// derivations and the nearest one wins. Each is memoized, so the chain
// is walked once and not once per node.
func (b *Base) resolveLook() style.LookAndFeel {
	if b.look != nil {
		return b.look
	}
	gen := lookGen.Load()
	if b.lookMemo != nil && b.lookGen == gen {
		return b.lookMemo
	}
	l := b.inheritedLook()
	if b.theme != nil {
		l = style.Themed(l, *b.theme)
	}
	b.lookMemo, b.lookGen = l, gen
	return l
}

// inheritedLook is what reaches this component from above, before its
// own level is written over it.
func (b *Base) inheritedLook() style.LookAndFeel {
	if b.parent != nil {
		return b.parent.Look()
	}
	if b.host != nil {
		return b.host.Look()
	}
	return style.DarkLook()
}

// LookGeneration is the number of times every memoized look has been
// thrown away. It is here for tests and for diagnosing a window that
// re-resolves when it should not: a steady window laying out and
// painting must not move it.
func LookGeneration() uint64 { return lookGen.Load() }
