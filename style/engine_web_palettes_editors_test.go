package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The editor palettes keep their colours in their own roles: Gruvbox's
// PmenuSel blue and Solarized's reversed base01 for the menu's current
// item, Atom's One themes on 3px corners with boxed tabs.
func TestWebEditorPalettes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range []struct {
		name, lineage, window, text, accent, menu string
		year                                      int
	}{
		{"gruvbox", "Gruvbox", "#282828", "#ebdbb2", "#83a598", "#83a598", 2012},
		{"gruvbox-light", "Gruvbox", "#fbf1c7", "#3c3836", "#076678", "#076678", 2012},
		{"solarized", "Solarized", "#002b36", "#93a1a1", "#268bd2", "#586e75", 2011},
		{"solarized-light", "Solarized", "#fdf6e3", "#586e75", "#268bd2", "#93a1a1", 2011},
		{"onedark", "Atom", "#282c34", "#9da5b4", "#4d78cc", "#3a3f4b", 2014},
		{"onelight", "Atom", "#fafafa", "#424243", "#556de8", "#f2f2f2", 2014},
	} {
		p, ok := LoadTheme(c.name)
		if !ok {
			t.Fatalf("%s not registered", c.name)
		}
		if p.Year != c.year || p.Lineage != c.lineage {
			t.Errorf("%s: year %d lineage %q", c.name, p.Year, p.Lineage)
		}
		w := webColors(winLook(t, c.name, 1))
		for what, got := range map[string][2]string{
			"window": {colorHexPadded(w.window), c.window}, "text": {colorHexPadded(w.text), c.text},
			"accent": {colorHexPadded(w.accent), c.accent}, "menu highlight": {colorHexPadded(w.menuHover), c.menu},
		} {
			if got[0] != got[1] {
				t.Errorf("%s: %s %s, want %s", c.name, what, got[0], got[1])
			}
		}
	}
	for _, n := range []string{"onedark", "onelight"} {
		lk := winLook(t, n, 1)
		c := webColors(lk)
		if c.radius != 3 || c.tabStyle != webTabEditor || c.rows[webSide].inset != 0 {
			t.Errorf("%s: radius %v, tab style %d, sidebar inset %v", n, c.radius, c.tabStyle, c.rows[webSide].inset)
		}
		// The active tab is the pane's colour, with no line along its top.
		img := webPaint(lk, 140, 50, func(ctx *paintengine2d.Context) {
			lk.DrawTabBar(ctx, paintengine2d.XYWH(0, 10, 140, 30))
			lk.DrawTab(ctx, paintengine2d.XYWH(10, 10, 100, 30), StateNone, "tab", true)
		})
		if !winNear(img, 14, 10, c.tabPane) || !winNear(img, 120, 20, webIDE(lk).tabStrip) {
			t.Errorf("%s: the active tab is not the pane's colour on the app's", n)
		}
	}
}
