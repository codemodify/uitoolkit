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
	fades    []stateFade // one per tab: hover cross-fades
	// enabled and shown are the per-tab flags, by index, grown only when
	// something turns one off — a bar nobody has called SetTabEnabled or
	// SetTabVisible on carries neither slice and behaves as it always did.
	// Both read as true past their end, so growing Titles adds tabs that
	// are on.
	enabled []bool
	shown   []bool
}

// TabEnabled reports whether tab i can be selected. Tabs are enabled.
func (t *TabBar) TabEnabled(i int) bool {
	return i >= 0 && i < len(t.Titles) && (i >= len(t.enabled) || t.enabled[i])
}

// TabVisible reports whether tab i is in the strip. Tabs are visible.
func (t *TabBar) TabVisible(i int) bool {
	return i >= 0 && i < len(t.Titles) && (i >= len(t.shown) || t.shown[i])
}

// SetTabEnabled greys a tab out. It stays in the strip — the page exists
// and is not available, which is different from not being there — and it
// cannot be clicked, arrowed onto or scrolled onto.
func (t *TabBar) SetTabEnabled(i int, v bool) { t.setFlag(&t.enabled, i, v) }

// SetTabVisible takes a tab out of the strip altogether, for a page that
// does not apply at all: a mail with no HTML part has no HTML tab, and a
// disabled one would only invite a click that can never work.
func (t *TabBar) SetTabVisible(i int, v bool) { t.setFlag(&t.shown, i, v) }

func (t *TabBar) setFlag(slot *[]bool, i int, v bool) {
	if i < 0 || i >= len(t.Titles) {
		return
	}
	for len(*slot) < len(t.Titles) {
		*slot = append(*slot, true)
	}
	if (*slot)[i] == v {
		return
	}
	(*slot)[i] = v
	// The selected tab may have just become one that cannot be selected.
	if !t.selectable(t.Selected) {
		if n := t.nearest(t.Selected, 0); n >= 0 {
			t.Selected = n
			if t.OnSelect != nil {
				t.OnSelect(n)
			}
		}
	}
	t.Invalidate()
	t.RequestLayout()
}

// selectable is "in range, shown and enabled".
func (t *TabBar) selectable(i int) bool { return t.TabVisible(i) && t.TabEnabled(i) }

// nearest is the tab Select(i) lands on. With dir 0 it looks forward from
// i and then backward, which makes Select(0) mean "the first tab there
// actually is" and Select(len-1) the last. With dir -1 or +1 it looks
// only that way, which is what an arrow key means: Left at the leftmost
// selectable tab must stay, not wrap round to the right.
func (t *TabBar) nearest(i, dir int) int {
	n := len(t.Titles)
	if n == 0 {
		return -1
	}
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	if dir == 0 {
		for j := i; j < n; j++ {
			if t.selectable(j) {
				return j
			}
		}
		for j := i - 1; j >= 0; j-- {
			if t.selectable(j) {
				return j
			}
		}
		return -1
	}
	for j := i; j >= 0 && j < n; j += dir {
		if t.selectable(j) {
			return j
		}
	}
	return -1
}

// step moves one selectable tab along (an arrow key, a wheel notch).
func (t *TabBar) step(dir int) {
	if i := t.nearest(t.Selected+dir, dir); i >= 0 {
		t.Select(i)
	}
}

// NewTabBar constructs a tab strip.
func NewTabBar(titles ...string) *TabBar {
	t := &TabBar{Titles: titles, hover: -1, press: -1}
	t.Init(t)
	t.SetWantsFocus(true)
	t.SetFocusVisibleOnly(true)
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
	lk := t.Look()
	f := style.ControlFontOf(lk, style.RoleTab)
	h := t.LocalBounds().Dy()
	margin := style.Dip(lk, 4)
	x := margin
	// Neighbours overlap by the look's tab overlap, so two tabs share one
	// border line (each tab still paints only inside its own rect).
	ov := style.TabOverlapOf(lk)
	// Hidden tabs take no width and get an empty rect, which is what
	// every reader below treats as "not there".
	vis := make([]int, 0, n)
	for i := range t.Titles {
		if t.TabVisible(i) {
			vis = append(vis, i)
		}
	}
	if len(vis) == 0 {
		return make([]paintengine2d.Rect, n)
	}
	ws := make([]float32, len(vis))
	minW := style.Dip(lk, 56)
	var total float32
	for k, i := range vis {
		ws[k] = max(f.Advance(t.Titles[i])+style.Dip(lk, 28), minW)
		total += ws[k]
	}
	total -= ov * float32(len(vis)-1)
	// Too many for the strip: tabs give up width above a narrower floor
	// (their labels elide), then the strip slides to keep the selected tab
	// in view (Qt's QTabBar, GTK's notebook).
	avail := t.LocalBounds().Dx() - 2*margin
	if total > avail && avail > 0 {
		floor := style.Dip(lk, 40)
		var spare float32
		for _, w := range ws {
			spare += w - floor
		}
		if cut := total - avail; spare > 0 {
			k := min(cut/spare, 1)
			for i := range ws {
				ws[i] -= (ws[i] - floor) * k
			}
		}
	}
	out := make([]paintengine2d.Rect, n)
	for k, w := range ws {
		out[vis[k]] = paintengine2d.XYWH(x, 0, w, h)
		x += w
		if k < len(vis)-1 {
			x -= ov
		}
	}
	if end := x + margin; end > t.LocalBounds().Dx() && t.Selected >= 0 && t.Selected < n && out[t.Selected].Dx() > 0 {
		sel := out[t.Selected]
		shift := float32(0)
		if over := sel.Max.X + margin - t.LocalBounds().Dx(); over > 0 {
			shift = -over
		}
		for i := range out {
			if out[i].Dx() > 0 {
				out[i] = out[i].Translate(paintengine2d.Pt(shift, 0))
			}
		}
	}
	// Some looks centre the strip over its page (Aqua's segmented tabs).
	if style.LookHint(lk, style.HintTabsCentered) == 1 {
		if shift := (t.LocalBounds().Dx() - (x + style.Dip(lk, 4))) * 0.5; shift > 0 {
			for i := range out {
				if out[i].Dx() > 0 {
					out[i] = out[i].Translate(paintengine2d.Pt(shift, 0))
				}
			}
		}
	}
	return out
}

func (t *TabBar) indexAt(x float32) int {
	// Later tabs win where neighbours overlap: they paint over the shared
	// border.
	rects := t.tabRects()
	for i := len(rects) - 1; i >= 0; i-- {
		if r := rects[i]; r.Dx() > 0 && x >= r.Min.X && x < r.Max.X {
			return i
		}
	}
	return -1
}

func (t *TabBar) Paint(ctx *paintengine2d.Context) {
	lk := t.Look()
	b := t.LocalBounds()
	lk.DrawTabBar(ctx, b)
	rects := t.tabRects()
	state := func(i int) style.ControlState {
		// The bar's own hover/press describe the whole strip; each tab takes
		// hover and press only from the index under the pointer.
		st := t.State() &^ (style.StateHovered | style.StatePressed)
		if i == t.hover {
			st |= style.StateHovered
		}
		if i == t.press {
			st |= style.StatePressed
		}
		if i != t.Selected {
			st &^= style.StateFocused
		}
		if !t.TabEnabled(i) {
			st |= style.StateDisabled
			st &^= style.StateHovered | style.StatePressed
		}
		// First and last are about the strip as it is drawn, so a hidden
		// tab does not leave its neighbour looking like the middle of a
		// row that starts with it.
		if i == t.firstShown() {
			st |= style.StateFirst
		}
		if i == t.lastShown() {
			st |= style.StateLast
		}
		return st
	}
	if len(t.fades) != len(t.Titles) {
		t.fades = make([]stateFade, len(t.Titles))
	}
	for i, title := range t.Titles {
		if i != t.Selected && rects[i].Dx() > 0 {
			r := rects[i]
			t.fades[i].paint(t, ctx, r, state(i), func(ctx *paintengine2d.Context, st style.ControlState) {
				lk.DrawTab(ctx, r, st, title, false)
			})
		}
	}
	// The selected tab paints last, grown by the look's outset, so it can
	// overlap its neighbours (Win95, XP, Platinum and Motif tabs do).
	if i := t.Selected; i >= 0 && i < len(t.Titles) && rects[i].Dx() > 0 {
		r := rects[i]
		out := style.TabOutsetOf(lk)
		r = paintengine2d.XYWH(r.Min.X-out.Left, r.Min.Y-out.Top, r.Dx()+out.Left+out.Right, r.Dy()+out.Top+out.Bottom).Intersect(b)
		lk.DrawTab(ctx, r, state(i), t.Titles[i], true)
	}
}

func (t *TabBar) firstShown() int {
	for i := range t.Titles {
		if t.TabVisible(i) {
			return i
		}
	}
	return -1
}

func (t *TabBar) lastShown() int {
	for i := len(t.Titles) - 1; i >= 0; i-- {
		if t.TabVisible(i) {
			return i
		}
	}
	return -1
}

func (t *TabBar) invalidateTab(i int) {
	rects := t.tabRects()
	if i < 0 || i >= len(rects) {
		return
	}
	t.InvalidateRect(rects[i].Inset(-1))
}

func (t *TabBar) MouseMove(e widget.MouseEvent) bool {
	i := t.indexAt(e.Pos.X)
	if !t.TabEnabled(i) {
		// A disabled tab does not light up. Highlighting one is a promise
		// the click will not keep.
		i = -1
	}
	if i != t.hover {
		old := t.hover
		t.hover = i
		t.invalidateTab(old)
		t.invalidateTab(i)
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
	t.MarkPointerFocus()
	t.RequestFocus()
	t.press = -1
	if i := t.indexAt(e.Pos.X); t.selectable(i) {
		t.press = i
	}
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
	if !t.Enabled() {
		return false
	}
	if e.Scroll.Y > 0 {
		t.step(1)
		return true
	}
	if e.Scroll.Y < 0 {
		t.step(-1)
		return true
	}
	return false
}

func (t *TabBar) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeyLeft:
		t.step(-1)
		return true
	case platform.KeyRight:
		t.step(1)
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

// Select changes the current tab. A disabled or hidden tab cannot be
// selected: the nearest one that can be takes it — searched forward from
// i and then backward — so Select(0) means "the first tab there is".
func (t *TabBar) Select(i int) {
	if !t.Enabled() {
		return
	}
	i = t.nearest(i, 0)
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
	// Disabled greys the tab: the page exists and is not available now.
	Disabled bool
	// Hidden takes it out of the strip: the page does not apply at all —
	// no HTML part, so no HTML tab. A hidden tab is not a disabled one,
	// and showing a disabled tab for a page that will never exist only
	// invites a click that can never work.
	Hidden bool
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
			tab.Content.SetVisible(false)
		}
		if tab.Disabled {
			tv.bar.SetTabEnabled(i, false)
		}
		if tab.Hidden {
			tv.bar.SetTabVisible(i, false)
		}
	}
	// The first tab that can be selected, which is the first one unless
	// the caller hid or disabled it.
	tv.selected = tv.bar.nearest(0, 0)
	if tv.selected >= 0 {
		tv.bar.Selected = tv.selected
		if c := tabs[tv.selected].Content; c != nil {
			c.SetVisible(true)
		}
	}
	tv.bar.OnSelect = func(i int) { tv.Select(i) }
	return tv
}

// SetTabEnabled greys tab i out. A disabled tab cannot be selected, and
// selecting away from it happens here rather than being the caller's job.
func (t *TabView) SetTabEnabled(i int, v bool) { t.setTabFlag(i, v, false) }

// SetTabVisible takes tab i out of the strip, or puts it back.
func (t *TabView) SetTabVisible(i int, v bool) { t.setTabFlag(i, v, true) }

// TabEnabled reports whether tab i can be selected.
func (t *TabView) TabEnabled(i int) bool { return t.bar.TabEnabled(i) }

// TabVisible reports whether tab i is in the strip.
func (t *TabView) TabVisible(i int) bool { return t.bar.TabVisible(i) }

func (t *TabView) setTabFlag(i int, v, visibility bool) {
	if i < 0 || i >= len(t.tabs) {
		return
	}
	was := t.bar.Selected
	if visibility {
		t.tabs[i].Hidden = !v
		t.bar.SetTabVisible(i, v)
	} else {
		t.tabs[i].Disabled = !v
		t.bar.SetTabEnabled(i, v)
	}
	// The bar moves its own selection off a tab that can no longer hold
	// it; the pages have to follow. Its OnSelect is this view's Select,
	// so the common case is already done — this covers the one where the
	// bar changed Selected without firing (it fires only on a real move).
	if t.bar.Selected != was || t.selected != t.bar.Selected {
		t.Select(t.bar.Selected)
	}
}

// Bar is the tab strip.
func (t *TabView) Bar() *TabBar { return t.bar }

// Page is the content host (kept for API completeness; contents are children).
func (t *TabView) Page() *TabPage { return t.page }

// Selected is the current tab index.
func (t *TabView) Selected() int { return t.selected }

// Select shows tab i, or the nearest tab that is neither disabled nor
// hidden ([TabBar.Select]).
func (t *TabView) Select(i int) {
	i = t.bar.nearest(i, 0)
	if i < 0 || i >= len(t.tabs) {
		return
	}
	if t.selected == i && t.tabs[i].Content != nil && t.tabs[i].Content.Visible() {
		return
	}
	if t.selected >= 0 && t.selected < len(t.tabs) && t.tabs[t.selected].Content != nil {
		old := t.tabs[t.selected].Content
		if h := t.Host(); h != nil && widget.Contains(old, h.Focus()) {
			t.bar.RequestFocus()
			widget.MarkKeyboardFocus(t.bar)
		}
		old.SetVisible(false)
	}
	t.selected = i
	t.bar.Selected = i
	if t.tabs[i].Content != nil {
		t.tabs[i].Content.SetVisible(true)
	}
	t.Invalidate()
	t.RequestLayout()
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
	if tp, ok := t.Look().(style.TabPaneLook); ok {
		tp.DrawTabPane(ctx, t.LocalBounds())
	} else {
		t.Look().DrawPanel(ctx, t.LocalBounds(), false)
	}
}
