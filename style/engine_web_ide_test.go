package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The editors' packs: VS Code's 2026 themes and JetBrains' Islands.
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
		{"islands", "Islands Light", "JetBrains", 2025, false, "Inter"},
		{"islands-night", "Islands Dark", "JetBrains", 2025, true, "Inter"},
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
	for n, square := range map[string]bool{"vscode-night": true, "vscode": true, "islands": false, "shadcn": false} {
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

// Islands' editor tabs: the selected one a pill in its hairline, the others
// bare labels with a wash pill under the pointer; the first pill clear of
// the island's corner.
func TestIslandsPillTabs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range []string{"islands", "islands-night"} {
		lk := winLook(t, n, 1)
		c, d := webColors(lk), webIDE(lk)
		tab := paintengine2d.XYWH(4, 0, 80, 40)
		paint := func(st ControlState, selected bool) *paintengine2d.Image {
			return webPaint(lk, 200, 40, func(ctx *paintengine2d.Context) {
				lk.DrawTabBar(ctx, paintengine2d.XYWH(0, 0, 200, 40))
				lk.DrawTab(ctx, tab, st, "Tab", selected)
			})
		}
		sel := paint(StateNone, true)
		if !winNear(sel, 6, 22, d.pillBorder) || !winNear(sel, 10, 22, d.pill) || !winNear(sel, 10, 5, c.tabPane) {
			t.Errorf("%s: the selected tab is not a %s pill in %s: %v %v", n, colorHexPadded(d.pill), colorHexPadded(d.pillBorder), pixelColor(sel, 6, 22), pixelColor(sel, 10, 22))
		}
		if !winNear(paint(StateNone, false), 10, 22, c.tabPane) {
			t.Errorf("%s: an unselected tab has a pill", n)
		}
		if !winNear(paint(StateHovered, false), 10, 22, webOver(c.tabPane, c.wash)) {
			t.Errorf("%s: a hovered tab has no wash pill", n)
		}
		if first := paint(StateFirst, true); winNear(first, 8, 22, d.pill) || !winNear(first, 16, 22, d.pill) {
			t.Errorf("%s: the first pill is not clear of the island's corner", n)
		}
	}
}

// Islands: views, sidebars, cards and tab views are 10px-rounded panels
// inside a 3px band left to what is behind, so an island shows the window
// round it and merges into an island that holds it.
func TestIslandsFrames(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	behind := Hex("#ff00ff")
	for _, sc := range []float32{1, 2} {
		lk := winLook(t, "islands", sc)
		c := webColors(lk)
		band, r := snap(3*sc), 10*sc
		// The band and the room the 10px corner needs inside the hairline.
		if in := lk.ViewFrameInsets(); in.Left != 7*sc {
			t.Errorf("islands@%gx: view insets %v, want %v", sc, in.Left, 7*sc)
		}
		frame := func(draw func(ctx *paintengine2d.Context, b paintengine2d.Rect)) *paintengine2d.Image {
			w, h := int(200*sc), int(120*sc)
			img := paintengine2d.NewImage(w, h)
			ctx := paintengine2d.NewContext(img)
			ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(w), float32(h)), paintengine2d.Fill(behind))
			draw(ctx, paintengine2d.XYWH(0, 0, float32(w), float32(h)))
			return img
		}
		mid := int(60 * sc)
		for what, img := range map[string]*paintengine2d.Image{
			"view": frame(func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawViewFrame(ctx, b, StateNone) }),
			"sidebar": frame(func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawViewFrame(ctx, b, StateSidebar)
			}),
			"card":     frame(func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawGroupBox(ctx, b, "", false) }),
			"tab pane": frame(func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTabPane(ctx, b) }),
		} {
			in := int(band)
			if !winNear(img, in-1, mid, behind) || winNear(img, in, mid, behind) {
				t.Errorf("islands@%gx %s: the band is not %v wide and left to what is behind", sc, what, band)
			}
			if !winNear(img, in, in, behind) || winNear(img, in+int(r), in+int(r), behind) {
				t.Errorf("islands@%gx %s: the panel's corner is not rounded", sc, what)
			}
		}
		// The tab strip is the island's top: the band above and at the
		// sides, rounded above, open below.
		bar := frame(func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTabBar(ctx, paintengine2d.XYWH(0, 0, b.Dx(), 40*sc))
		})
		in := int(band)
		if !winNear(bar, mid, in-1, behind) || !winNear(bar, mid, in, c.tabPane) || !winNear(bar, in, int(40*sc)-1, c.tabPane) || winNear(bar, in, in, c.tabPane) {
			t.Errorf("islands@%gx: the tab strip is not the island's top", sc)
		}
		// A group's heading sits inside its island; its body clears both.
		g := lk.GroupBoxInsets(true)
		if g.Left != 15*sc || g.Top <= g.Left {
			t.Errorf("islands@%gx: group insets %+v", sc, g)
		}
	}
	// Selections: rounded boxes inset 8px in the view's own room, 12 from
	// the island's side.
	lk := winLook(t, "islands", 1)
	c := webColors(lk)
	row := webPaint(lk, 200, 24, func(ctx *paintengine2d.Context) {
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 200, 24), paintengine2d.Fill(c.field))
		lk.DrawListRow(ctx, paintengine2d.XYWH(0, 0, 200, 24), StateChecked, "")
	})
	sel := webOver(c.field, c.sel)
	if winNear(row, 6, 12, sel) || !winNear(row, 14, 12, sel) || winNear(row, 8, 0, sel) {
		t.Errorf("islands: a selected row is not a rounded box inset 8px")
	}
	// Focus: a 2px ring beyond a 1px gap round the face.
	b := paintengine2d.XYWH(10, 10, 120, 34)
	foc := webPaint(lk, 140, 54, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, StateFocused, "") })
	if !winNear(foc, 60, 10, c.focus) || !winNear(foc, 60, 11, c.focus) || !winNear(foc, 60, 12, c.window) || !winNear(foc, 60, 13, c.btnBorder) {
		t.Errorf("islands: a focused button has no 2px ring a pixel clear of its face")
	}
	// An open combo keeps the ring: Swing rings whatever holds focus.
	for open, want := range map[bool]bool{true: true, false: false} {
		img := webPaint(lk, 140, 54, func(ctx *paintengine2d.Context) { lk.DrawComboBox(ctx, b, StateNone, "Open", open) })
		if got := winNear(img, 60, 10, c.focus); got != want {
			t.Errorf("islands: combo open %v rings %v", open, got)
		}
	}
}

// The five tab styles paint five ways.
func TestWebTabStylesDiffer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	b := paintengine2d.XYWH(10, 10, 120, 32)
	var imgs []*paintengine2d.Image
	for _, ts := range []float32{webTabUnderline, webTabSegmented, webTabBrowser, webTabEditor, webTabPill} {
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
