package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The editors' packs: VS Code's 2026 themes.
func TestWebEditorPacks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range []struct {
		name, label, lineage string
		year                 int
		dark                 bool
		ui                   string
	}{
		{"vscode", "VS Code Light 2026", "VS Code", 2026, false, "Segoe WPC"},
		{"vscode-night", "VS Code Dark 2026", "VS Code", 2026, true, "Segoe WPC"},
	} {
		p, ok := LoadTheme(c.name)
		if !ok {
			t.Fatalf("%s not registered", c.name)
		}
		if p.Label != c.label || p.Lineage != c.lineage || p.Year != c.year || p.Tokens.Engine != "web" || (p.Tokens.Family == ThemeDark) != c.dark {
			t.Errorf("%s: %q %q %d engine %s family %s", c.name, p.Label, p.Lineage, p.Year, p.Tokens.Engine, p.Tokens.Family)
		}
		lk := winLook(t, c.name, 1)
		if got := lk.Metrics().FontSize; got != 13 {
			t.Errorf("%s: %vpx text, want 13", c.name, got)
		}
		if f := withEraFonts(p.Tokens, c.name).Fonts.UI; len(f) == 0 || f[0] != c.ui {
			t.Errorf("%s: UI faces %v, want %s first", c.name, f, c.ui)
		}
	}
}

// VS Code's editor tabs: the active tab in the editor's colour with its
// top line, open to the editor below; the others on the strip over its
// bottom hairline; a hairline at each tab's end.
func TestVSCodeEditorTabs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range []string{"vscode", "vscode-night"} {
		lk := winLook(t, n, 1)
		c, d := webColors(lk), webIDE(lk)
		tab := paintengine2d.XYWH(10, 10, 100, 35)
		paint := func(selected bool) *paintengine2d.Image {
			return webPaint(lk, 160, 55, func(ctx *paintengine2d.Context) {
				lk.DrawTabBar(ctx, paintengine2d.XYWH(0, 10, 160, 35))
				lk.DrawTab(ctx, tab, StateNone, "main.go", selected)
			})
		}
		sel, other := paint(true), paint(false)
		if !winNear(sel, 14, 10, c.tabLine) || !winNear(sel, 14, 11, d.tabActive) {
			t.Errorf("%s: the active tab has no 1px line along its top", n)
		}
		if !winNear(sel, 14, 44, d.tabActive) || !winNear(sel, 140, 44, d.tabSep) || !winNear(sel, 140, 30, d.tabStrip) {
			t.Errorf("%s: the active tab is not open to the editor beside the strip's hairline", n)
		}
		if !winNear(sel, 109, 30, d.tabSep) || !winNear(other, 109, 30, d.tabSep) {
			t.Errorf("%s: no hairline at a tab's end", n)
		}
		if winNear(other, 14, 10, c.tabLine) || !winNear(other, 14, 30, d.tabStrip) || !winNear(other, 14, 44, d.tabSep) {
			t.Errorf("%s: an inactive tab is not the bare strip", n)
		}
	}
	// Focus is VS Code's 1px border inside the control.
	lk := winLook(t, "vscode", 1)
	c := webColors(lk)
	b := paintengine2d.XYWH(10, 10, 120, 26)
	foc := webPaint(lk, 140, 46, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, StateFocused, "") })
	if !winNear(foc, 60, 10, c.focus) || winNear(foc, 60, 11, c.focus) || !winNear(foc, 60, 9, c.window) {
		t.Errorf("vscode: a focused button has no 1px focus border inside its edge")
	}
}

// Scroll thumbs: VS Code's are square, the others pills.
func TestWebScrollRadius(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p := ScrollParts{Bar: paintengine2d.XYWH(10, 10, 10, 100), Track: paintengine2d.XYWH(10, 10, 10, 100), Thumb: paintengine2d.XYWH(10, 30, 10, 40)}
	for n, square := range map[string]bool{"vscode-night": true, "vscode": true, "shadcn": false, "linear": false} {
		lk := winLook(t, n, 1)
		c := webColors(lk)
		img := webPaint(lk, 30, 120, func(ctx *paintengine2d.Context) { DrawScrollBarParts(lk, ctx, p, true, ScrollState{Hovered: true}) })
		thumb := webOver(c.window, c.scrollThumb)
		// The wide thumb's top-left corner pixel.
		x := int(10 + snap(c.scrollInset))
		if got := winNear(img, x, 30, thumb); got != square {
			t.Errorf("%s: thumb corner %v, square %v", n, pixelColor(img, x, 30), square)
		}
		if !winNear(img, 15, 50, thumb) {
			t.Errorf("%s: no thumb at its middle", n)
		}
	}
}

// The four tab styles paint four ways.
func TestWebTabStylesDiffer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	b := paintengine2d.XYWH(10, 10, 120, 32)
	var imgs []*paintengine2d.Image
	for _, ts := range []float32{webTabUnderline, webTabSegmented, webTabBrowser, webTabEditor} {
		lk := webLookWith(t, "primer", 1, map[string]float32{"tabStyle": ts})
		imgs = append(imgs, webPaint(lk, 140, 52, func(ctx *paintengine2d.Context) {
			lk.DrawTabBar(ctx, paintengine2d.XYWH(0, 10, 140, 32))
			lk.DrawTab(ctx, b, StateNone, "Tab", true)
		}))
	}
	for i := range imgs {
		for j := i + 1; j < len(imgs); j++ {
			if winSame(imgs[i], imgs[j]) {
				t.Errorf("tab styles %d and %d paint alike", i, j)
			}
		}
	}
}
