package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// tabsRig is a 700×34 strip of three tabs with every callback recorded.
type tabsRig struct {
	t                                  *testing.T
	s                                  *BrowserTabs
	selected, closed, reordered, menus []string
	news                               int
}

func newTabsRig(t *testing.T, titles ...string) *tabsRig {
	r := &tabsRig{t: t}
	if len(titles) == 0 {
		titles = []string{"uitoolkit", "paintengine2d", "notes"}
	}
	s := NewBrowserTabs(titles...)
	s.OnSelect = func(i int) { r.selected = append(r.selected, s.Tab(i).Title) }
	s.OnClose = func(i int) { r.closed = append(r.closed, s.Tab(i).Title); s.RemoveTab(i) }
	s.OnNew = func() { r.news++; s.Select(s.AddTab(BrowserTab{Title: "New"})) }
	s.OnReorder = func(from, to int) { r.reordered = append(r.reordered, fmt.Sprint(from, "→", to)) }
	s.OnContextMenu = func(i int, _ paintengine2d.Point) bool { r.menus = append(r.menus, fmt.Sprint(i)); return true }
	s.SetHost(newFrameHost())
	arrange(s, 700, 34)
	r.s = s
	return r
}

func (r *tabsRig) center(i int) paintengine2d.Point {
	g := r.s.geom()
	s := g.slots[i]
	return paintengine2d.Pt((s.Min.X+s.Max.X)/2-10, (s.Min.Y+s.Max.Y)/2)
}

func (r *tabsRig) closeOf(i int) paintengine2d.Point {
	c := r.s.closeRect(r.s.geom().slots[i])
	return paintengine2d.Pt((c.Min.X+c.Max.X)/2, (c.Min.Y+c.Max.Y)/2)
}

func (r *tabsRig) press(p paintengine2d.Point, b platform.MouseButton) {
	r.s.MouseMove(widget.MouseEvent{Pos: p})
	r.s.MousePress(widget.MouseEvent{Pos: p, Button: b})
}

func (r *tabsRig) click(p paintengine2d.Point) {
	r.press(p, platform.ButtonLeft)
	r.s.MouseRelease(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
}

func (r *tabsRig) titles() []string {
	var out []string
	for i := 0; i < r.s.Len(); i++ {
		out = append(out, r.s.Tab(i).Title)
	}
	return out
}

func TestBrowserTabsLayoutAndCaption(t *testing.T) {
	r := newTabsRig(t)
	g := r.s.geom()
	if len(g.slots) != 3 || g.overflow || g.tabW != 200 {
		t.Fatalf("three tabs at their widest: %+v", g)
	}
	if g.newBtn.Empty() || g.newBtn.Min.X < g.slots[2].Max.X {
		t.Fatalf("the new-tab button %v follows the last tab %v", g.newBtn, g.slots[2])
	}
	if r.s.CaptionAt(r.center(1)) || r.s.CaptionAt(r.closeOf(0)) {
		t.Fatal("tabs and their close buttons are controls")
	}
	nb := g.newBtn
	if r.s.CaptionAt(paintengine2d.Pt((nb.Min.X+nb.Max.X)/2, (nb.Min.Y+nb.Max.Y)/2)) {
		t.Fatal("the new-tab button is a control")
	}
	if !r.s.CaptionAt(paintengine2d.Pt(690, 17)) || !r.s.CaptionAt(paintengine2d.Pt(300, 1)) && g.tabH < 34 {
		t.Fatal("the rest of the strip is caption")
	}
	// Without a new-tab handler there is no button.
	r.s.OnNew = nil
	if !r.s.geom().newBtn.Empty() {
		t.Fatal("no OnNew, no new-tab button")
	}
}

func TestBrowserTabsSelectCloseNew(t *testing.T) {
	r := newTabsRig(t)
	r.click(r.center(1))
	if r.s.Selected() != 1 || len(r.selected) != 1 || r.selected[0] != "paintengine2d" {
		t.Fatalf("select: %d %v", r.s.Selected(), r.selected)
	}
	// The selected tab shows its close button; closing it selects the next.
	if !r.s.hasClose(1, r.s.geom()) {
		t.Fatal("the selected tab has a close button")
	}
	r.click(r.closeOf(1))
	if got := r.titles(); len(got) != 2 || got[1] != "notes" || r.s.Selected() != 1 || r.selected[len(r.selected)-1] != "notes" {
		t.Fatalf("after close: %v sel %d %v", got, r.s.Selected(), r.selected)
	}
	// Closing the last tab while it is selected selects the one before.
	r.s.CloseTab(1)
	if r.s.Selected() != 0 || r.s.Len() != 1 {
		t.Fatalf("closing the last: sel %d len %d", r.s.Selected(), r.s.Len())
	}
	// A middle click closes a tab; closing one left of the selection keeps
	// the selected tab (its index moves).
	r.s.AddTab(BrowserTab{Title: "b"})
	r.s.AddTab(BrowserTab{Title: "c"})
	r.s.Select(2)
	r.press(r.center(0), platform.ButtonMiddle)
	if got := r.titles(); len(got) != 2 || got[0] != "b" || r.s.Selected() != 1 {
		t.Fatalf("middle click: %v sel %d", got, r.s.Selected())
	}
	// The new-tab button.
	g := r.s.geom()
	r.click(paintengine2d.Pt((g.newBtn.Min.X+g.newBtn.Max.X)/2, (g.newBtn.Min.Y+g.newBtn.Max.Y)/2))
	if r.news != 1 || r.s.Len() != 3 || r.s.Selected() != 2 {
		t.Fatalf("new tab: news %d len %d sel %d", r.news, r.s.Len(), r.s.Selected())
	}
	// A tab that keeps its place has no close button.
	r.s.SetTab(0, BrowserTab{Title: "home", NoClose: true})
	r.s.Select(0)
	if r.s.hasClose(0, r.s.geom()) {
		t.Fatal("NoClose")
	}
	r.s.CloseTab(0)
	if r.s.Tab(0).Title != "home" {
		t.Fatal("a NoClose tab does not close")
	}
	// Without OnClose the strip removes the tab itself.
	r.s.OnClose = nil
	r.s.CloseTab(1)
	if r.s.Len() != 2 {
		t.Fatalf("removed by the strip: %v", r.titles())
	}
}

func TestBrowserTabsDragReorders(t *testing.T) {
	r := newTabsRig(t)
	from := r.center(0)
	r.press(from, platform.ButtonLeft)
	// Less than the threshold: still a click on the tab.
	r.s.MouseMove(widget.MouseEvent{Pos: from.Add(paintengine2d.Pt(3, 0))})
	if r.s.drag.active {
		t.Fatal("a 3 px wobble is no drag")
	}
	to := r.center(2)
	r.s.MouseMove(widget.MouseEvent{Pos: to})
	if !r.s.drag.active {
		t.Fatal("dragging")
	}
	// The others make room while it moves.
	g := r.s.geom()
	if s := r.s.slotOf(1, g); s.Min.X >= g.slots[1].Min.X {
		t.Fatalf("tab 1 slides left under the drag: %v", s)
	}
	r.s.MouseRelease(widget.MouseEvent{Pos: to, Button: platform.ButtonLeft})
	if got := r.titles(); got[2] != "uitoolkit" || got[0] != "paintengine2d" || len(r.reordered) != 1 || r.reordered[0] != "0→2" {
		t.Fatalf("reorder: %v %v", got, r.reordered)
	}
	if r.s.Selected() != 2 {
		t.Fatalf("the selection follows the dragged tab: %d", r.s.Selected())
	}
	// A drag clamps to the strip.
	r.press(r.center(2), platform.ButtonLeft)
	r.s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(-400, 17)})
	if r.s.drag.x < r.s.geom().view.Min.X {
		t.Fatalf("dragged past the start: %v", r.s.drag.x)
	}
	r.s.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(-400, 17), Button: platform.ButtonLeft})
	if r.titles()[0] != "uitoolkit" {
		t.Fatalf("dropped at the start: %v", r.titles())
	}
}

func TestBrowserTabsOverflowScrolls(t *testing.T) {
	var titles []string
	for i := 0; i < 14; i++ {
		titles = append(titles, fmt.Sprintf("Folder %d", i))
	}
	r := newTabsRig(t, titles...)
	g := r.s.geom()
	if !g.overflow || g.tabW != 80 || g.prev.Empty() || g.next.Empty() || g.maxScroll <= 0 {
		t.Fatalf("fourteen tabs overflow at their narrowest: %+v", g)
	}
	r.s.Select(13)
	g = r.s.geom()
	if s := g.slots[13]; s.Max.X > g.view.Max.X+0.5 || s.Min.X < g.view.Min.X {
		t.Fatalf("the selected tab %v is revealed in %v", s, g.view)
	}
	before := r.s.scroll
	r.click(paintengine2d.Pt((g.prev.Min.X+g.prev.Max.X)/2, (g.prev.Min.Y+g.prev.Max.Y)/2))
	if r.s.scroll >= before {
		t.Fatalf("the scroll-left button: %v → %v", before, r.s.scroll)
	}
	r.s.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, -40)})
	if r.s.scroll != 0 {
		t.Fatalf("the wheel scrolls back to the start: %v", r.s.scroll)
	}
	// A tab left of the view is caption no more than it is visible: the
	// clipped part is neither tab nor caption for the pointer.
	if h := r.s.hitAt(paintengine2d.Pt(g.view.Max.X+1, 17)); h.part == partTab {
		t.Fatal("tabs outside the view take no clicks")
	}
}

func TestBrowserTabsKeysAndShortcuts(t *testing.T) {
	r := newTabsRig(t)
	key := func(k platform.Key, m platform.Modifiers) bool {
		return r.s.KeyPress(widget.KeyEvent{Key: k, Mods: m})
	}
	if !key(platform.KeyRight, 0) || r.s.Selected() != 1 {
		t.Fatal("Right")
	}
	key(platform.KeyEnd, 0)
	key(platform.KeyLeft, 0)
	if r.s.Selected() != 1 {
		t.Fatalf("End, Left: %d", r.s.Selected())
	}
	sc := func(k platform.Key, m platform.Modifiers) bool {
		return r.s.Shortcut(widget.KeyEvent{Key: k, Mods: m})
	}
	sc(platform.KeyTab, platform.ModCtrl)
	sc(platform.KeyTab, platform.ModCtrl)
	if r.s.Selected() != 0 {
		t.Fatalf("Ctrl+Tab wraps round: %d", r.s.Selected())
	}
	sc(platform.KeyTab, platform.ModCtrl|platform.ModShift)
	if r.s.Selected() != 2 {
		t.Fatalf("Ctrl+Shift+Tab: %d", r.s.Selected())
	}
	sc(platform.KeyPageUp, platform.ModCtrl)
	if r.s.Selected() != 1 {
		t.Fatalf("Ctrl+PageUp: %d", r.s.Selected())
	}
	if !sc(platform.KeyW, platform.ModCtrl) || r.s.Len() != 2 || r.closed[0] != "paintengine2d" {
		t.Fatalf("Ctrl+W: %v", r.closed)
	}
	if !sc(platform.KeyT, platform.ModCtrl) || r.news != 1 {
		t.Fatal("Ctrl+T")
	}
	if sc(platform.KeyW, 0) || sc(platform.KeyA, platform.ModCtrl) {
		t.Fatal("other keys are the app's")
	}
}

func TestBrowserTabsTooltipsAndMenus(t *testing.T) {
	r := newTabsRig(t, "short", "a title far too long to fit in a tab two hundred pixels wide")
	r.s.MouseMove(widget.MouseEvent{Pos: r.center(0)})
	if tip := r.s.Tooltip(); tip != "" {
		t.Fatalf("a title that fits has no tip: %q", tip)
	}
	r.s.MouseMove(widget.MouseEvent{Pos: r.center(1)})
	if tip := r.s.Tooltip(); tip != r.s.Tab(1).Title {
		t.Fatalf("an elided title's tip: %q", tip)
	}
	r.s.MouseMove(widget.MouseEvent{Pos: r.closeOf(1)})
	if r.s.Tooltip() != "Close Tab" {
		t.Fatalf("close tip %q", r.s.Tooltip())
	}
	r.press(r.center(0), platform.ButtonRight)
	if !r.s.CaptionMenu(paintengine2d.Pt(690, 17)) || len(r.menus) != 2 || r.menus[0] != "0" || r.menus[1] != "-1" {
		t.Fatalf("context menus: %v", r.menus)
	}
}

func TestBrowserTabsAccessibility(t *testing.T) {
	r := newTabsRig(t)
	r.s.Select(1)
	var n a11y.Node
	r.s.Describe(&n)
	if n.Role != a11y.RoleTabList {
		t.Fatalf("role %v", n.Role)
	}
	items := r.s.AccessibleItems()
	if len(items) != 4 || items[3].Role != a11y.RoleButton || items[3].Name != "New Tab" {
		t.Fatalf("items %+v", items)
	}
	for i, it := range items[:3] {
		if it.Role != a11y.RoleTab || it.Name != r.s.Tab(i).Title || it.Index != i+1 || it.Count != 3 {
			t.Fatalf("tab %d: %+v", i, it)
		}
		if (i == 1) != it.State.Has(a11y.StateSelected) {
			t.Fatalf("tab %d selected %v", i, it.State.Has(a11y.StateSelected))
		}
		if len(it.Children) != 1 || it.Children[0].Role != a11y.RoleButton || it.Children[0].Name != "Close "+it.Name {
			t.Fatalf("tab %d close button %+v", i, it.Children)
		}
	}
	root := &a11y.Node{ID: 1, Role: a11y.RoleWindow, Name: "w", Children: []*a11y.Node{{ID: r.s.ID(), Role: a11y.RoleTabList, Name: "Tabs", Children: items}}}
	if probs := a11y.Check(root); len(probs) != 0 {
		t.Fatalf("a11y: %v", probs)
	}
	// Assistive technology selects a tab, presses a close button and the
	// new-tab button.
	if !r.s.AccessibleAction(2, a11y.ActionDefault) || r.s.Selected() != 2 {
		t.Fatal("select through AT-SPI")
	}
	if !r.s.AccessibleAction(3+0, a11y.ActionDefault) || r.s.Tab(0).Title != "paintengine2d" {
		t.Fatalf("close tab 0 through AT-SPI: %v", r.titles())
	}
	if !r.s.AccessibleAction(2*r.s.Len(), a11y.ActionDefault) || r.news != 1 {
		t.Fatal("new tab through AT-SPI")
	}
}

// In a header bar the strip takes the caption's whole height, so its tabs
// stand on the content below.
func TestBrowserTabsFillTheHeaderBar(t *testing.T) {
	s := NewBrowserTabs("a", "b")
	hb := NewHeaderBar([]widget.Component{NewButton("Menu", nil)}, s, nil)
	hb.SetHost(newFrameHost())
	hb.SetWindowControls(platform.ParseButtonLayout(":close"), true)
	sz := hb.Measure(layout.Loose(800, 400))
	hb.Arrange(paintengine2d.XYWH(0, 0, 800, sz.Y))
	if b := s.Bounds(); b.Min.Y != 0 || b.Dy() != sz.Y {
		t.Fatalf("strip %v in a %v tall header bar", b, sz.Y)
	}
	if !s.onCaption() {
		t.Fatal("in a merged frame's caption")
	}
}
