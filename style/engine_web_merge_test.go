//go:build theme_engine_all || theme_engine_web

package style

import "testing"

// Chrome's tab shape works by filling the selected tab with the surface
// *below* the strip, so the tab, the row under it and the page read as one
// continuous surface with a channel cut through the strip.
//
// A pack that states the same colour for its title bar and its tool bar
// has nothing to merge into: the selected tab came out exactly the colour
// of the strip it stands in, and the only thing marking it was its own
// outline, which reads as a misdrawn box rather than a tab. Plenty of
// packs are like that.
func TestAMergedTabFillsWithWhatIsUnderTheStrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Packs whose tool bar is their title bar, with a window a shade
	// apart: these are the ones that showed it.
	for _, name := range []string{
		"gruvbox-light", "catppuccin-latte", "gruvbox", "catppuccin-mocha",
		"nord", "solarized-light",
	} {
		p, ok := LoadTheme(name)
		if !ok {
			continue
		}
		c := webColors(p.Look())
		if c.toolBar != c.titleBar {
			t.Logf("%s: tool bar and title bar differ now; nothing to fall back from", name)
			continue
		}
		fill, _ := c.mergedTabSurface()
		if fill == c.titleBar {
			t.Errorf("%s: the selected tab is filled with the strip's own colour, so nothing merges", name)
		}
		if fill != c.window {
			t.Errorf("%s: the selected tab is filled with %v, want the window's %v", name, fill, c.window)
		}
	}

	// A pack with a tool bar of its own keeps it: that is the case the
	// shape was drawn for and it must not change.
	if p, ok := LoadTheme("linear"); ok {
		c := webColors(p.Look())
		if c.toolBar == c.titleBar {
			t.Skip("linear no longer states a tool bar of its own")
		}
		if fill, _ := c.mergedTabSurface(); fill != c.toolBar {
			t.Errorf("linear: filled with %v, want its own tool bar %v", fill, c.toolBar)
		}
	}
}
