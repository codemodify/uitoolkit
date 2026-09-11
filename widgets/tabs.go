package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// TabBar is a strip of titled tabs that reports selection.
type TabBar struct {
	widget.Base
	Titles   []string
	Selected int
	OnSelect func(int)
	hover    int
	press    int
}

// NewTabBar constructs a tab strip.
func NewTabBar(titles ...string) *TabBar {
	t := &TabBar{Titles: titles, hover: -1, press: -1}
	t.Init(t)
	t.SetWantsFocus(true)
	return t
}

func (t *TabBar) barH() float32 {
	h := t.Look().Metrics().TabH
	if h <= 0 {
		h = 30
	}
	return h
}

func (t *TabBar) Measure(c layout.Constraints) paintengine2d.Point {
	w := float32(160)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, t.barH()))
}

func (t *TabBar) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

func (t *TabBar) tabRects() []paintengine2d.Rect {
	n := len(t.Titles)
	if n == 0 {
		return nil
	}
	f := t.Look().Font()
	h := t.LocalBounds().Dy()
	x := float32(4)
	out := make([]paintengine2d.Rect, n)
	for i, title := range t.Titles {
		w := f.Advance(title) + 28
		if w < 56 {
			w = 56
		}
		out[i] = paintengine2d.XYWH(x, 0, w, h)
		x += w
	}
	return out
}

func (t *TabBar) indexAt(x float32) int {
	for i, r := range t.tabRects() {
		if x >= r.Min.X && x < r.Max.X {
			return i
		}
	}
	return -1
}

func (t *TabBar) Paint(ctx *paintengine2d.Context) {
	lk := t.Look()
	lk.DrawTabBar(ctx, t.LocalBounds())
	for i, title := range t.Titles {
		st := t.State()
		if i == t.hover {
			st |= style.StateHovered
		}
		if i == t.press {
			st |= style.StatePressed
		}
		if i != t.Selected {
			st &^= style.StateFocused
		}
		lk.DrawTab(ctx, t.tabRects()[i], st, title, i == t.Selected)
	}
}

func (t *TabBar) MouseMove(e widget.MouseEvent) bool {
	i := t.indexAt(e.Pos.X)
	if i != t.hover {
		t.hover = i
		t.Invalidate()
	}
	return true
}

func (t *TabBar) MouseExit() {
	t.hover = -1
	t.press = -1
	t.Base.MouseExit()
}

func (t *TabBar) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
	t.press = t.indexAt(e.Pos.X)
	t.Invalidate()
	return true
}

func (t *TabBar) MouseRelease(e widget.MouseEvent) bool {
	i := t.indexAt(e.Pos.X)
	was := t.press
	t.press = -1
	if was >= 0 && i == was {
		t.Select(i)
	}
	t.Invalidate()
	return true
}

func (t *TabBar) MouseWheel(e widget.MouseEvent) bool {
	if e.Scroll.Y > 0 {
		t.Select(t.Selected + 1)
		return true
	}
	if e.Scroll.Y < 0 {
		t.Select(t.Selected - 1)
		return true
	}
	return false
}

func (t *TabBar) KeyPress(e widget.KeyEvent) bool {
	switch e.Key {
	case platform.KeyLeft:
		t.Select(t.Selected - 1)
		return true
	case platform.KeyRight:
		t.Select(t.Selected + 1)
		return true
	case platform.KeyHome:
		t.Select(0)
		return true
	case platform.KeyEnd:
		t.Select(len(t.Titles) - 1)
		return true
	}
	return false
}

// Select changes the current tab.
func (t *TabBar) Select(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(t.Titles) {
		i = len(t.Titles) - 1
	}
	if i < 0 || i == t.Selected {
		return
	}
	t.Selected = i
	t.Invalidate()
	if t.OnSelect != nil {
		t.OnSelect(i)
	}
}

// TabPage hosts one visible child (content swap).
type TabPage struct {
	widget.Base
}

// NewTabPage wraps child.
func NewTabPage(child widget.Component) *TabPage {
	p := &TabPage{}
	p.Init(p)
	if child != nil {
		p.Add(child)
	}
	return p
}

// SetContent replaces the page body.
func (p *TabPage) SetContent(c widget.Component) {
	p.ClearChildren()
	if c != nil {
		p.Add(c)
	}
}

func (p *TabPage) Measure(c layout.Constraints) paintengine2d.Point {
	var w, h float32
	for _, ch := range p.Children() {
		vis := ch.Visible()
		ch.SetVisible(true)
		sz := ch.Measure(c)
		ch.SetVisible(vis)
		if sz.X > w {
			w = sz.X
		}
		if sz.Y > h {
			h = sz.Y
		}
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (p *TabPage) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	box := paintengine2d.XYWH(0, 0, r.Dx(), r.Dy())
	for _, ch := range p.Children() {
		ch.Arrange(box)
	}
}

func (p *TabPage) Paint(ctx *paintengine2d.Context) {
	lk := p.Look()
	ctx.DrawRect(p.LocalBounds(), paintengine2d.Fill(lk.Palette().Background.WithAlpha(0.20)))
}

// Tab is one titled page for TabView.
type Tab struct {
	Title   string
	Content widget.Component
}

// TabView composes a TabBar and swapped TabPage content.
type TabView struct {
	widget.Base
	tabs     []Tab
	selected int
	bar      *TabBar
	page     *TabPage
	OnChange func(int)
}

// NewTabView builds a tabbed container.
func NewTabView(tabs ...Tab) *TabView {
	titles := make([]string, len(tabs))
	for i, tab := range tabs {
		titles[i] = tab.Title
	}
	tv := &TabView{tabs: tabs, bar: NewTabBar(titles...), page: NewTabPage(nil)}
	tv.Init(tv)
	tv.Base.Add(tv.bar)
	tv.Base.Add(tv.page)
	for i, tab := range tabs {
		if tab.Content != nil {
			tv.page.Add(tab.Content)
			tab.Content.SetVisible(i == 0)
		}
	}
	tv.bar.OnSelect = func(i int) { tv.Select(i) }
	return tv
}

// Bar is the tab strip.
func (t *TabView) Bar() *TabBar { return t.bar }

// Page is the content host (kept for API completeness; contents are children).
func (t *TabView) Page() *TabPage { return t.page }

// Selected is the current tab index.
func (t *TabView) Selected() int { return t.selected }

// Select shows tab i.
func (t *TabView) Select(i int) {
	if i < 0 || i >= len(t.tabs) {
		return
	}
	if t.selected == i && t.tabs[i].Content != nil && t.tabs[i].Content.Visible() {
		return
	}
	if t.selected >= 0 && t.selected < len(t.tabs) && t.tabs[t.selected].Content != nil {
		t.tabs[t.selected].Content.SetVisible(false)
	}
	t.selected = i
	t.bar.Selected = i
	if t.tabs[i].Content != nil {
		t.tabs[i].Content.SetVisible(true)
	}
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(i)
	}
}

func (t *TabView) Measure(c layout.Constraints) paintengine2d.Point {
	bar := t.bar.Measure(c)
	inner := c.Inset(0, bar.Y)
	sz := t.page.Measure(inner)
	w, h := sz.X, sz.Y+bar.Y
	if c.HasMaxW() {
		w = c.MaxW
	}
	if c.HasMaxH() {
		h = c.MaxH
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (t *TabView) Arrange(r paintengine2d.Rect) {
	t.SetBounds(r)
	bh := t.bar.barH()
	t.bar.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), bh))
	t.page.Arrange(paintengine2d.XYWH(0, bh, r.Dx(), r.Dy()-bh))
}

func (t *TabView) Paint(ctx *paintengine2d.Context) {
	t.Look().DrawPanel(ctx, t.LocalBounds(), false)
}
