package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The looks with a sidebar style of their own: macOS's source list,
// libadwaita's navigation sidebar, WinUI's NavigationView pane and
// Material's navigation drawer.
var sidebarPackNames = []string{
	"yosemite", "bigsur", "bigsur-night",
	"adwaita", "adwaita-night",
	"fluent", "fluent-night",
	"material", "material-night", "material3", "material3-night",
}

// sidebarSelected are the states of a selected row: in a focused view, in
// one without focus, and in an inactive window.
var sidebarSelected = []ControlState{
	StateChecked, StateChecked | StateFocused, StateChecked | StateHovered | StateFocused,
	StateChecked | StateInactive, StateChecked | StateInactive | StateBackdrop,
}

// sidebarRowImage paints a row as a view does: the view's background under
// it, then the row. tree paints a tree row instead of a list row.
func sidebarRowImage(lk *Classic, b paintengine2d.Rect, st ControlState, label string, tree bool) *paintengine2d.Image {
	img := paintengine2d.NewImage(int(b.Max.X), int(b.Max.Y))
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(b, paintengine2d.Fill(ViewBackgroundOf(lk, st)))
	if tree {
		lk.DrawTreeRow(ctx, b, st, false, true, 0, label, false)
	} else {
		lk.DrawListRow(ctx, b, st, label)
	}
	return img
}

// sidebarDiff counts the pixels where two images differ beyond rounding
// (colour or coverage). Some platforms' panes sit close to their views:
// WinUI's dark Mica (#202020) is four levels off its list field (#1c1c1c).
func sidebarDiff(a, b *paintengine2d.Image) int {
	n := 0
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			r1, g1, b1, a1 := a.PremulAt(x, y)
			r2, g2, b2, a2 := b.PremulAt(x, y)
			if absDiff8(r1, r2)+absDiff8(g1, g2)+absDiff8(b1, b2)+absDiff8(a1, a2) > 3 {
				n++
			}
		}
	}
	return n
}

func absDiff8(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// pixelColor is the opaque colour of an image pixel.
func pixelColor(img *paintengine2d.Image, x, y int) paintengine2d.Color {
	r, g, b, _ := img.PremulAt(x, y)
	return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
}

// sidebarLabelColor is the colour a sidebar row of st labels itself in: the
// value the engine's own row painter returns (painted into a scratch image).
func sidebarLabelColor(t *testing.T, lk *Classic, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	t.Helper()
	ctx := paintengine2d.NewContext(paintengine2d.NewImage(int(b.Max.X), int(b.Max.Y)))
	switch lk.Engine().ID() {
	case "macos":
		return macColors(lk).sideRow(lk, ctx, b, st)
	case "adwaita":
		_, fg := adwColors(lk).sideRow(lk, ctx, b, st)
		return fg
	case "fluent":
		return fluentColors(lk).itemText(st)
	case "material":
		_, fg := mdColors(lk).drawerRow(lk, ctx, b, st)
		return fg
	}
	t.Fatalf("%s: engine %q has no sidebar style", lk.Pack(), lk.Engine().ID())
	return paintengine2d.Color{}
}

// A selected sidebar row paints differently from a plain one, in the list
// and the tree (what the row sits on counts: the pane); a row at rest
// differs where the pane does (Material 2's drawer rests on the surface).
func TestSidebarRowsDifferFromPlainRows(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	b := paintengine2d.XYWH(0, 0, 220, 32)
	for _, n := range sidebarPackNames {
		lk := lunaLook(t, n, 1)
		paneDiffers := ViewBackgroundOf(lk, StateSidebar) != ViewBackgroundOf(lk, StateNone)
		for _, st := range []ControlState{StateChecked, StateChecked | StateFocused, StateChecked | StateInactive, StateNone} {
			if st == StateNone && !paneDiffers {
				continue
			}
			for _, tree := range []bool{false, true} {
				plain := sidebarRowImage(lk, b, st, "Themes", tree)
				side := sidebarRowImage(lk, b, st|StateSidebar, "Themes", tree)
				if d := sidebarDiff(plain, side); d < 200 {
					t.Errorf("%s tree=%v state %b: a sidebar row paints like a plain row (%d pixels differ)", n, tree, st, d)
				}
			}
		}
	}
	// GNOME 3's sidebars were plain views; that look keeps them so.
	lk := lunaLook(t, "adwaita-gtk3", 1)
	if d := sidebarDiff(sidebarRowImage(lk, b, StateChecked, "Themes", false), sidebarRowImage(lk, b, StateChecked|StateSidebar, "Themes", false)); d != 0 {
		t.Errorf("adwaita-gtk3: a sidebar row changed (%d pixels)", d)
	}
}

// A selected sidebar row's label reads at 4.5:1 on the selection it sits
// on, focused or not, active window or not; the label is painted.
func TestSidebarSelectionTextReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	b := paintengine2d.XYWH(0, 0, 220, 32)
	for _, n := range sidebarPackNames {
		lk := lunaLook(t, n, 1)
		for _, st := range sidebarSelected {
			st |= StateSidebar
			for _, tree := range []bool{false, true} {
				bare := sidebarRowImage(lk, b, st, "", tree)
				// The fill under the label: inside the selection box at its
				// right end, clear of the text.
				fill := pixelColor(bare, 196, 16)
				if fill == pixelColor(sidebarRowImage(lk, b, st&^StateChecked, "", tree), 196, 16) {
					t.Errorf("%s tree=%v state %b: no selection painted", n, tree, st)
					continue
				}
				fg := sidebarLabelColor(t, lk, b, st)
				if r := ContrastRatio(fg, fill); r < 4.5 {
					t.Errorf("%s tree=%v state %b: label %s on selection %s is %.2f:1", n, tree, st, colorHexPadded(fg), colorHexPadded(fill), r)
				}
				// The label is really painted in a contrasting colour.
				img := sidebarRowImage(lk, b, st, "Themes", tree)
				best := 0.0
				for y := 6; y < 26; y++ {
					for x := 4; x < 150; x++ {
						best = max(best, ContrastRatio(pixelColor(img, x, y), fill))
					}
				}
				if best < 3 {
					t.Errorf("%s tree=%v state %b: the label's ink reaches only %.2f:1 on the selection", n, tree, st, best)
				}
			}
		}
	}
}

// What a sidebar's rows sit on differs from the field colour where the
// platform's did — macOS's source list, libadwaita's sidebar pane (darker
// than the white view, lighter than the dark one), Material 3's drawer —
// and stays the field where it did not: Material 2's drawer is the
// surface, GNOME 3's sidebar the view. The WinUI pane shows the window's
// Mica; macOS and libadwaita change the pane in an inactive window.
func TestSidebarViewBackground(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	differs := map[string]bool{
		"yosemite": true, "bigsur": true, "bigsur-night": true,
		"adwaita": true, "adwaita-night": true, "adwaita-gtk3": false,
		"fluent": true, "fluent-night": true,
		"material": false, "material-night": false, "material3": true, "material3-night": true,
	}
	backdrop := map[string]bool{"yosemite": true, "bigsur": true, "bigsur-night": true, "adwaita": true, "adwaita-night": true}
	for n, want := range differs {
		lk := lunaLook(t, n, 1)
		field := ViewBackgroundOf(lk, StateNone)
		if field != lk.Palette().Field {
			t.Errorf("%s: a plain view sits on %s, not the field %s", n, colorHexPadded(field), colorHexPadded(lk.Palette().Field))
		}
		side := ViewBackgroundOf(lk, StateSidebar)
		if got := side != field; got != want {
			t.Errorf("%s: sidebar background %s, field %s: differs=%v, want %v", n, colorHexPadded(side), colorHexPadded(field), got, want)
		}
		off := ViewBackgroundOf(lk, StateSidebar|StateBackdrop)
		if got := off != side; got != backdrop[n] {
			t.Errorf("%s: sidebar background in an inactive window %s (active %s): differs=%v, want %v", n, colorHexPadded(off), colorHexPadded(side), got, backdrop[n])
		}
		// The frame paints the pane the rows sit on, edge to edge.
		img := paintengine2d.NewImage(120, 80)
		DrawViewFrameOf(lk, paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 120, 80), StateSidebar)
		if want && (!nxNear(img, 1, 40, side) || !nxNear(img, 60, 40, side)) {
			t.Errorf("%s: the sidebar frame is not the pane %s", n, colorHexPadded(side))
		}
	}
	// libadwaita's pane is darker than the white view in the light style.
	if lk := lunaLook(t, "adwaita", 1); Luma(ViewBackgroundOf(lk, StateSidebar)) >= Luma(ViewBackgroundOf(lk, StateNone)) {
		t.Error("adwaita: the sidebar pane is not darker than the view")
	}
}

// Sidebar rows, frames and (for macOS, whose tool buttons changed) free and
// tool-bar tool buttons paint inside their rects at 1× and 2×.
func TestSidebarPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	states := append([]ControlState{}, lunaStates...)
	states = append(states, sidebarSelected...)
	states = append(states, StatePressed|StateChecked, StateExpanderHot|StateHovered, StateDisabled|StateBackdrop)
	type call struct {
		name string
		w, h float32
		fn   func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState)
	}
	calls := []call{
		{"list row", 170, 32, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawListRow(ctx, r, st|StateSidebar, "Packs & icons")
		}},
		{"short row", 60, 18, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawListRow(ctx, r, st|StateSidebar, "Themes")
		}},
		{"tree row", 170, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTreeRow(ctx, r, st|StateSidebar, false, false, 1, "Archives", false)
			lk.DrawTreeRow(ctx, r, st|StateSidebar, true, true, 3, "A very long folder name", true)
		}},
		{"frame", 150, 90, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawViewFrame(ctx, r, st|StateSidebar)
		}},
		{"tool", 64, 34, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			if lk.Engine().ID() == "macos" {
				lk.DrawToolButton(ctx, r, st, "Tool", IconNone)
				lk.DrawToolButton(ctx, r, st|StateAutoRaise, "", IconOpen)
				lk.DrawToolButton(ctx, r, st|StateAutoRaise, "Fetch", IconOpen)
			}
		}},
	}
	for _, n := range append(sidebarPackNames, "adwaita-gtk3") {
		for _, sc := range []float32{1, 2} {
			lk := lunaLook(t, n, sc)
			for _, cl := range calls {
				for _, st := range states {
					img := paintengine2d.NewImage(int((cl.w+40)*sc), int((cl.h+40)*sc))
					ctx := paintengine2d.NewContext(img)
					r := paintengine2d.XYWH(20*sc, 20*sc, cl.w*sc, cl.h*sc)
					cl.fn(lk, ctx, r, st)
					if px := lunaOutside(img, r); px > 0 {
						t.Errorf("%s %gx %s state %b: %d pixels outside the rect", n, sc, cl.name, st, px)
					}
					if ctx.SaveCount() != 0 {
						t.Fatalf("%s %gx %s: %d saved states left", n, sc, cl.name, ctx.SaveCount())
					}
				}
			}
		}
	}
}

// macOS tool buttons: in a tool bar (StateAutoRaise) Big Sur's is borderless
// until the pointer is over it and Yosemite's latched one is the dark grey
// textured segment; a free tool button keeps the plain bezel, and a latched
// free one is drawn highlighted, in the accent.
func TestMacOSToolButtonsAutoRaise(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	b := paintengine2d.XYWH(0, 0, 64, 34)
	ink := func(lk *Classic, st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(64, 34)
		lk.DrawToolButton(paintengine2d.NewContext(img), b, st, "", IconNone)
		return img
	}
	for _, n := range macosPackNames {
		lk := macLook(t, n, 1)
		c := macColors(lk)
		if _, _, _, a := ink(lk, StateNone).PremulAt(32, 17); a == 0 {
			t.Errorf("%s: a free tool button lost its bezel", n)
		}
		if !nxNear(ink(lk, StateToggle|StateChecked), 32, 17, Mix(c.accentStops[0].Color, c.accentStops[1].Color, 0.5)) {
			t.Errorf("%s: a latched free tool button is not the accent", n)
		}
		bar := ink(lk, StateAutoRaise)
		_, _, _, a := bar.PremulAt(32, 17)
		switch {
		case c.bigSur && a != 0:
			t.Errorf("%s: a tool bar's button shows a face at rest", n)
		case !c.bigSur && a == 0:
			t.Errorf("%s: Yosemite's tool bar button lost its textured bezel", n)
		}
		if c.bigSur {
			if _, _, _, a := ink(lk, StateAutoRaise|StateHovered).PremulAt(32, 17); a == 0 {
				t.Errorf("%s: a tool bar's button shows no wash under the pointer", n)
			}
		} else if !nxNear(ink(lk, StateAutoRaise|StateToggle|StateChecked), 32, 17, c.toolOn) {
			t.Errorf("%s: Yosemite's latched tool bar button is not the dark grey segment", n)
		}
		// Hover shows on both kinds (the toolkit keeps tool buttons' hover).
		for _, auto := range []ControlState{StateNone, StateAutoRaise} {
			if sidebarDiff(ink(lk, auto), ink(lk, auto|StateHovered)) == 0 {
				t.Errorf("%s autoRaise=%v: no hover feedback", n, auto != 0)
			}
		}
	}
}

// Sanity: the packs named here exist and paint with the engines named.
func TestSidebarPacksExist(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	engines := map[string]bool{}
	for _, n := range sidebarPackNames {
		lk := lunaLook(t, n, 1)
		engines[lk.Engine().ID()] = true
	}
	for _, e := range []string{"macos", "adwaita", "fluent", "material"} {
		if !engines[e] {
			t.Errorf("no pack paints with %s", e)
		}
	}
}
