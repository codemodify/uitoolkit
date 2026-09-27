package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The theme cascade, resolved in a tree of real widgets: the nearest
// level wins, a level inherits what it does not state, and the display
// scale is nobody's to override.

// packOf is the engine a component ended up painting through, which is
// the thing a wrong cascade gets wrong.
func packOf(t *testing.T, c widget.Component) string {
	t.Helper()
	lk, ok := c.Look().(*style.Classic)
	if !ok {
		t.Fatalf("%T resolved to %T, not a Classic look", c, c.Look())
	}
	return lk.Pack()
}

// rig lays a tree out under a window in pack at scale.
func rig(t *testing.T, pack string, scale float32, root widget.Component) *fakeWindow {
	t.Helper()
	p, ok := style.LoadTheme(pack)
	if !ok {
		t.Skipf("the %q pack is not in this build", pack)
	}
	// fakeWindow answers 1 for Scale; the display scale that matters to
	// the cascade is the one the window's *look* was built at, which is
	// what a scope derives from.
	h := &fakeWindow{look: style.WithScale(p.Look(), scale)}
	root.SetHost(h)
	root.Measure(layout.Loose(600, 400))
	root.Arrange(paintengine2d.XYWH(0, 0, 600, 400))
	return h
}

// A themed component takes the whole subtree with it and leaves its
// siblings alone — the tab-in-Metal case, whole.
func TestCascadeSubtreeAndSiblings(t *testing.T) {
	inner := NewButton("Inside", nil)
	themed := NewPanel("Themed", NewColumn(inner))
	outside := NewButton("Outside", nil)
	root := NewColumn(outside, themed)
	rig(t, "luna", 1, root)

	themed.SetTheme(style.ThemeOverride{Pack: "metal-ocean"})

	if got := packOf(t, inner); got != "metal-ocean" {
		t.Fatalf("inside the scope: %s", got)
	}
	if got := packOf(t, themed); got != "metal-ocean" {
		t.Fatalf("the scope itself: %s", got)
	}
	if got := packOf(t, outside); got != "luna" {
		t.Fatalf("a sibling of the scope: %s", got)
	}
	if got := packOf(t, root); got != "luna" {
		t.Fatalf("the window's own tree: %s", got)
	}
}

// A single control is a scope of one.
func TestCascadeOneControl(t *testing.T) {
	odd := NewButton("Odd", nil)
	row := NewRow(NewButton("Plain", nil), odd)
	rig(t, "win95", 1, row)
	odd.SetTheme(style.ThemeOverride{Pack: "aqua"})
	if got := packOf(t, odd); got != "aqua" {
		t.Fatalf("the themed button: %s", got)
	}
	if got := packOf(t, row.Children()[0]); got != "win95" {
		t.Fatalf("the button beside it: %s", got)
	}
}

// A tab view whose content is in another pack, with the strip left in
// the window's — the owner's example.
func TestCascadeTabContent(t *testing.T) {
	body := NewButton("In the tab", nil)
	tv := NewTabView(Tab{Title: "One", Content: NewColumn(body)}, Tab{Title: "Two", Content: NewLabel("two")})
	rig(t, "luna", 1, tv)
	widget.SetTheme(tv.Page(), style.ThemeOverride{Pack: "metal-steel"})

	if got := packOf(t, body); got != "metal-steel" {
		t.Fatalf("the tab's content: %s", got)
	}
	if got := packOf(t, tv.Bar()); got != "luna" {
		t.Fatalf("the tab strip: %s", got)
	}
	if got := packOf(t, tv); got != "luna" {
		t.Fatalf("the tab view's own pane: %s", got)
	}
}

// Nearest wins: a scope inside a scope inside a scope.
func TestCascadeNestedNearestWins(t *testing.T) {
	leaf := NewButton("Leaf", nil)
	third := NewPanel("Third", NewColumn(leaf))
	second := NewPanel("Second", NewColumn(third))
	first := NewPanel("First", NewColumn(second))
	rig(t, "win95", 1, first)

	first.SetTheme(style.ThemeOverride{Pack: "luna"})
	second.SetTheme(style.ThemeOverride{Pack: "aqua"})
	third.SetTheme(style.ThemeOverride{Pack: "metal-ocean"})

	for _, tc := range []struct {
		what string
		c    widget.Component
		want string
	}{
		{"outermost scope", first, "luna"},
		{"middle scope", second, "aqua"},
		{"innermost scope", third, "metal-ocean"},
		{"a leaf under all three", leaf, "metal-ocean"},
	} {
		if got := packOf(t, tc.c); got != tc.want {
			t.Fatalf("%s: %s, want %s", tc.what, got, tc.want)
		}
	}
}

// A level inherits every part it does not state: a scope that names only
// the corners keeps the window's pack, and one that names only the pack
// keeps the window's corners, icons and pinned typeface.
func TestCascadeInheritsWhatItDoesNotState(t *testing.T) {
	inner := NewButton("Inner", nil)
	scope := NewPanel("", NewColumn(inner))
	root := NewColumn(scope)

	base := style.Appearance{
		Name: "luna", Corners: style.CornersSquare, Icons: style.IconSetSharp,
		IconSize: style.IconSizeLarge, FontUI: "Liberation Sans",
	}.Look()
	h := &fakeWindow{look: base}
	root.SetHost(h)

	scope.SetTheme(style.ThemeOverride{Corners: style.CornersRound})
	got := style.LookAppearance(inner.Look())
	if got.Name != "luna" {
		t.Fatalf("a corners-only level changed the pack: %s", got.Name)
	}
	if got.Corners != style.CornersRound {
		t.Fatalf("corners: %s", got.Corners)
	}
	if got.Icons != style.IconSetSharp || got.IconSize != style.IconSizeLarge {
		t.Fatalf("icons were not inherited: %s %s", got.Icons, got.IconSize)
	}
	if got.FontUI != "Liberation Sans" {
		t.Fatalf("the pinned typeface was not inherited: %q", got.FontUI)
	}
	if inner.Look().Metrics().Radius <= 0 {
		t.Fatal("round corners did not reach the metrics")
	}

	scope.SetTheme(style.ThemeOverride{Pack: "metal-ocean"})
	got = style.LookAppearance(inner.Look())
	if got.Name != "metal-ocean" {
		t.Fatalf("pack: %s", got.Name)
	}
	if got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp || got.IconSize != style.IconSizeLarge {
		t.Fatalf("a pack-only level dropped the inherited parts: %+v", got)
	}
	if got.FontUI != "Liberation Sans" {
		t.Fatalf("a pack-only level dropped the pinned typeface: %q", got.FontUI)
	}
}

// The display scale is the display's: it is not in a ThemeOverride, and
// a scope at any depth paints at the window's.
func TestCascadeScaleIsTheDisplays(t *testing.T) {
	leaf := NewButton("Leaf", nil)
	inner := NewPanel("", NewColumn(leaf))
	outer := NewPanel("", NewColumn(inner))
	rig(t, "luna", 2, outer)
	outer.SetTheme(style.ThemeOverride{Pack: "aqua"})
	inner.SetTheme(style.ThemeOverride{Pack: "win95", Corners: style.CornersSquare})

	for _, c := range []widget.Component{outer, inner, leaf} {
		lk, ok := c.Look().(*style.Classic)
		if !ok {
			t.Fatalf("%T", c.Look())
		}
		if lk.Scale() != 2 {
			t.Fatalf("%T is at %gx, the display is at 2x", c, lk.Scale())
		}
	}
}

// Clearing a level hands the subtree back to the window.
func TestCascadeClearing(t *testing.T) {
	inner := NewButton("Inner", nil)
	scope := NewPanel("", NewColumn(inner))
	rig(t, "luna", 1, scope)
	scope.SetTheme(style.ThemeOverride{Pack: "aqua"})
	if got := packOf(t, inner); got != "aqua" {
		t.Fatalf("themed: %s", got)
	}
	scope.SetTheme(style.ThemeOverride{})
	if got := packOf(t, inner); got != "luna" {
		t.Fatalf("cleared: %s", got)
	}
	if !widget.ThemeOf(scope).Empty() {
		t.Fatal("the level should be gone")
	}
}

// A level that states what the window already says does not build a
// second look: the subtree keeps the window's pointer, and with it every
// cache keyed on it.
func TestCascadeSamePackKeepsTheLook(t *testing.T) {
	inner := NewButton("Inner", nil)
	scope := NewPanel("", NewColumn(inner))
	h := rig(t, "luna", 1, scope)
	scope.SetTheme(style.ThemeOverride{Pack: "luna"})
	if inner.Look() != h.Look() {
		t.Fatal("a level that changes nothing should resolve to the same look")
	}
}

// Resolving is memoized: the same look comes back for the same tree, and
// a scope nested deep resolves to one look rather than one per level.
func TestCascadeResolvesOnce(t *testing.T) {
	leaf := NewButton("Leaf", nil)
	var top widget.Component = NewColumn(leaf)
	var scopes []*Panel
	for i := 0; i < 8; i++ {
		p := NewPanel("", top)
		scopes = append(scopes, p)
		top = NewColumn(p)
	}
	rig(t, "win95", 1, top)
	for _, p := range scopes {
		p.SetTheme(style.ThemeOverride{Pack: "aqua"})
	}
	first := leaf.Look()
	for i := 0; i < 50; i++ {
		if leaf.Look() != first {
			t.Fatal("a resolved look must be stable between calls")
		}
	}
	// Eight scopes all asking for aqua: the outermost derives one look
	// and the seven inside it change nothing, so they share it.
	if scopes[0].Look() != leaf.Look() {
		t.Fatal("nested scopes naming one pack should share one look")
	}
	if got := packOf(t, leaf); got != "aqua" {
		t.Fatalf("leaf: %s", got)
	}
}

// A component moved from one scope to another resolves in the new one.
func TestCascadeReparent(t *testing.T) {
	moved := NewButton("Moved", nil)
	a := NewPanel("", NewColumn(moved))
	b := NewPanel("", NewColumn())
	root := NewColumn(a, b)
	rig(t, "win95", 1, root)
	a.SetTheme(style.ThemeOverride{Pack: "luna"})
	b.SetTheme(style.ThemeOverride{Pack: "aqua"})
	if got := packOf(t, moved); got != "luna" {
		t.Fatalf("before the move: %s", got)
	}
	b.Content().Add(moved)
	if got := packOf(t, moved); got != "aqua" {
		t.Fatalf("after the move: %s", got)
	}
}

// SetLook still wins over everything: it is the way past the cascade.
func TestCascadeExplicitLookWins(t *testing.T) {
	inner := NewButton("Inner", nil)
	scope := NewPanel("", NewColumn(inner))
	rig(t, "win95", 1, scope)
	scope.SetTheme(style.ThemeOverride{Pack: "aqua"})
	exact, _ := style.LoadTheme("system7")
	inner.SetLook(exact.Look())
	if got := packOf(t, inner); got != "system7" {
		t.Fatalf("an exact look must beat the cascade: %s", got)
	}
}

// A combo box inside a themed pane opens its list in that pane's theme,
// and the list is *measured* in it too — packs differ in their row
// heights, and a list measured in one pack and painted in another is the
// wrong size.
func TestCascadePopupFromScope(t *testing.T) {
	combo := NewComboBox([]string{"one", "two", "three"}, 0, nil)
	scope := NewPanel("", NewColumn(combo))
	h := rig(t, "win95", 1, NewColumn(scope))
	scope.SetTheme(style.ThemeOverride{Pack: "aqua"})

	combo.Open()
	pop := h.popup
	if pop == nil {
		t.Fatal("the combo did not open a list")
	}
	lk, ok := pop.Look().(*style.Classic)
	if !ok || lk.Pack() != "aqua" {
		t.Fatalf("the list is in %v, not the pane's aqua", pop.Look())
	}
	if pop.Look() != combo.Look() {
		t.Fatal("the list should have exactly the combo's look")
	}
}

// A menu opened from a themed pane is in that theme, and so is every
// submenu that cascades out of it — a submenu hangs from no parent, so
// it is the case a naive cascade gets wrong.
func TestCascadeMenuAndSubmenuFromScope(t *testing.T) {
	anchor := NewButton("Anchor", nil)
	scope := NewPanel("", NewColumn(anchor))
	rig(t, "win95", 1, NewColumn(scope))
	scope.SetTheme(style.ThemeOverride{Pack: "metal-ocean"})

	pop := ShowContextMenu(anchor, paintengine2d.Pt(10, 10),
		Item("Plain", nil),
		Submenu("More", Item("Deeper", nil)))
	if pop == nil {
		t.Fatal("no context menu")
	}
	if got := packOf(t, pop); got != "metal-ocean" {
		t.Fatalf("the menu: %s", got)
	}
	pop.openCascade(1)
	sub := pop.CascadeMenu()
	if sub == nil {
		t.Fatal("no submenu")
	}
	if got := packOf(t, sub); got != "metal-ocean" {
		t.Fatalf("the submenu: %s", got)
	}
}

// A popup reused by its widget does not keep the theme of a scope it has
// since left.
func TestCascadePopupForgetsAnOldTheme(t *testing.T) {
	combo := NewComboBox([]string{"one", "two"}, 0, nil)
	scope := NewPanel("", NewColumn(combo))
	h := rig(t, "win95", 1, NewColumn(scope))

	scope.SetTheme(style.ThemeOverride{Pack: "aqua"})
	combo.Open()
	if got := packOf(t, h.popup); got != "aqua" {
		t.Fatalf("themed open: %s", got)
	}
	combo.Close()

	scope.SetTheme(style.ThemeOverride{})
	combo.Open()
	if got := packOf(t, h.popup); got != "win95" {
		t.Fatalf("after the scope was cleared: %s", got)
	}
}

// Hit testing, focus order and the accessibility tree do not know the
// cascade exists.
func TestCascadeIsInvisibleToInputAndAccess(t *testing.T) {
	a := NewButton("A", nil)
	b := NewButton("B", nil)
	scope := NewPanel("", NewColumn(b))
	root := NewColumn(a, scope)
	rig(t, "win95", 1, root)

	hitBefore := root.HitTest(paintengine2d.Pt(20, 20))
	focusBefore := len(widget.Focusables(root))
	var namesBefore []string
	widget.Walk(root, func(c widget.Component) { namesBefore = append(namesBefore, c.Name()) })

	scope.SetTheme(style.ThemeOverride{Pack: "aqua"})
	root.Measure(layout.Loose(600, 400))
	root.Arrange(paintengine2d.XYWH(0, 0, 600, 400))

	if got := root.HitTest(paintengine2d.Pt(20, 20)); got != hitBefore {
		t.Fatalf("hit testing changed: %T -> %T", hitBefore, got)
	}
	if got := len(widget.Focusables(root)); got != focusBefore {
		t.Fatalf("focus order changed: %d -> %d", focusBefore, got)
	}
	var namesAfter []string
	widget.Walk(root, func(c widget.Component) { namesAfter = append(namesAfter, c.Name()) })
	if len(namesAfter) != len(namesBefore) {
		t.Fatalf("the tree changed shape: %d -> %d nodes", len(namesBefore), len(namesAfter))
	}
}

// Every pack has to survive being a scope: the toolkit ships 131 of
// them, and a cascade that works in three is not a cascade.
func TestCascadeHoldsInEveryPack(t *testing.T) {
	inner := NewButton("Inner", nil)
	scope := NewPanel("", NewColumn(inner))
	rig(t, "metal-ocean", 1.5, NewColumn(scope))
	for _, pack := range style.ListThemes() {
		scope.SetTheme(style.ThemeOverride{Pack: pack.Name})
		lk, ok := inner.Look().(*style.Classic)
		if !ok {
			t.Fatalf("%s resolved to %T", pack.Name, inner.Look())
		}
		if lk.Pack() != pack.Name {
			t.Fatalf("%s resolved to pack %s", pack.Name, lk.Pack())
		}
		if lk.Scale() != 1.5 {
			t.Fatalf("%s lost the display scale: %g", pack.Name, lk.Scale())
		}
		if lk.Metrics().FontSize <= 0 {
			t.Fatalf("%s has no usable metrics", pack.Name)
		}
	}
}

// benchTree is a column nested depth deep with a leaf at the bottom, and
// a scope every eighth level when scopes is set.
func benchTree(depth int, scopes bool) (widget.Component, widget.Component) {
	leaf := NewButton("Leaf", nil)
	var node widget.Component = leaf
	for i := 0; i < depth; i++ {
		col := NewColumn(node)
		if scopes && i%8 == 7 {
			col.SetTheme(style.ThemeOverride{Pack: []string{"aqua", "luna", "win95", "system7"}[i%4]})
		}
		node = col
	}
	return node, leaf
}

// What one Look costs at the bottom of a deep tree — the call layout and
// paint make several times per widget per frame. Without the memo this
// walked one virtual call per level on every call.
func BenchmarkLookDeep(b *testing.B) {
	root, leaf := benchTree(64, false)
	root.SetHost(&fakeWindow{look: style.DarkLook()})
	_ = leaf.Look()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = leaf.Look()
	}
}

// The same, with eight nested scopes between the leaf and the window: a
// scope inside a scope inside a scope resolves once, not once per level.
func BenchmarkLookDeepScoped(b *testing.B) {
	root, leaf := benchTree(64, true)
	root.SetHost(&fakeWindow{look: style.DarkLook()})
	_ = leaf.Look()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = leaf.Look()
	}
}

// A modal dialog opened from a themed pane is in that pane's theme too:
// an overlay is a root of its own layer, so it cannot inherit by walking
// up any more than a popup can.
func TestCascadeDialogFromScope(t *testing.T) {
	anchor := NewButton("Open", nil)
	scope := NewPanel("", NewColumn(anchor))
	h := rig(t, "win95", 1, NewColumn(scope))
	scope.SetTheme(style.ThemeOverride{Pack: "aqua"})

	ShowMessageBox(anchor, MessageBoxOptions{Title: "Careful", Message: "Is it?"})
	if h.overlay == nil {
		t.Fatal("no overlay")
	}
	if got := packOf(t, h.overlay); got != "aqua" {
		t.Fatalf("the dialog: %s", got)
	}
}
