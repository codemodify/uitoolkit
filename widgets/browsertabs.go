package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// BrowserTab is one document tab of a BrowserTabs strip.
type BrowserTab struct {
	Title string
	// Tip is the tab's tool tip; empty: the title, when it does not fit.
	Tip string
	// NoClose hides the tab's close button (a window's last tab it keeps).
	NoClose bool
	// Data is the app's own (the document the tab shows).
	Data any
}

// BrowserTabs is a strip of document tabs, browser style: Chromium's and
// Firefox's tab strips, SourceGit's repository tabs, Dolphin's folder tabs.
// Give it to a HeaderBar as the centre and it is the window's title bar:
// the tabs and the new-tab button are controls, the rest of the strip is
// caption (dragging it moves the window, double-clicking maximizes,
// right-clicking shows OnContextMenu's menu or the window menu).
//
// The look paints the tabs (style.DrawBrowserTabOf: SourceGit's outlined
// tab with concave feet over the tool bar below it in the web looks, the
// look's own notebook tab elsewhere), the strip under them and, on the
// selected tab and the one under the pointer, a close button; a "+" after
// the last tab asks for a new one (OnNew). Tabs share the width equally, at
// most MaxTabWidth; below MinTabWidth each the strip scrolls instead (the
// wheel, two arrow buttons at its end), keeping the selected tab in view.
// A tab dragged along the strip moves to where it is dropped (OnReorder).
// Titles that do not fit are elided and shown in full as the tab's tool
// tip. With the keyboard focus, Left and Right choose the tab next to the
// selected one, Home and End the first and the last; Shortcut gives an app
// the browser shortcuts (Ctrl+Tab, Ctrl+W, Ctrl+T …) in one call.
type BrowserTabs struct {
	widget.Base
	tabs []BrowserTab
	sel  int
	// OnSelect reports a change of the selected tab: a click, the keyboard,
	// Select, or the selected tab closing.
	OnSelect func(i int)
	// OnClose asks to close tab i (its close button, a middle click,
	// CloseTab): the app removes it with RemoveTab, and may ask first. Without it
	// the strip removes the tab itself.
	OnClose func(i int)
	// OnNew asks for a new tab (the "+" button, NewTab); the strip shows the
	// button only when it is set.
	OnNew func()
	// OnReorder reports that the user dragged tab from to index to (the
	// strip has moved it; the selection follows it).
	OnReorder func(from, to int)
	// OnContextMenu runs for a right-click on tab i, or on the strip's
	// caption space (i -1), at window point at; it reports whether it
	// showed a menu (else the window menu shows for the caption space).
	OnContextMenu func(i int, at paintengine2d.Point) bool
	// MaxTabWidth and MinTabWidth bound a tab's width (design pixels;
	// 0: 200 and 80).
	MaxTabWidth, MinTabWidth float32

	hover, press stripHit
	scroll       float32
	drag         tabDrag
}

// stripPart is what a point of the strip is.
type stripPart uint8

const (
	partNone stripPart = iota
	partTab
	partClose
	partNew
	partPrev
	partNext
)

// stripHit is a part of the strip and the tab it belongs to.
type stripHit struct {
	part stripPart
	tab  int
}

// tabDrag follows a tab pressed and dragged along the strip.
type tabDrag struct {
	armed, active bool
	tab           int
	// from is where the press was; grab how far into the tab.
	from paintengine2d.Point
	grab float32
	// x is the dragged tab's left edge now.
	x float32
}

// NewBrowserTabs makes a strip with a tab for each title, the first
// selected.
func NewBrowserTabs(titles ...string) *BrowserTabs {
	t := &BrowserTabs{hover: stripHit{tab: -1}, press: stripHit{tab: -1}}
	t.Init(t)
	t.SetWantsFocus(true)
	t.SetFocusVisibleOnly(true)
	for _, s := range titles {
		t.tabs = append(t.tabs, BrowserTab{Title: s})
	}
	if len(t.tabs) == 0 {
		t.sel = -1
	}
	return t
}

// ---- the model --------------------------------------------------------------

// Len is the number of tabs.
func (t *BrowserTabs) Len() int { return len(t.tabs) }

// Tab is tab i (the zero tab when out of range).
func (t *BrowserTabs) Tab(i int) BrowserTab {
	if i < 0 || i >= len(t.tabs) {
		return BrowserTab{}
	}
	return t.tabs[i]
}

// SetTab replaces tab i (a new title, the document it shows now).
func (t *BrowserTabs) SetTab(i int, tab BrowserTab) {
	if i < 0 || i >= len(t.tabs) {
		return
	}
	t.tabs[i] = tab
	t.Invalidate()
}

// SetTitle retitles tab i.
func (t *BrowserTabs) SetTitle(i int, title string) {
	if i >= 0 && i < len(t.tabs) && t.tabs[i].Title != title {
		t.tabs[i].Title = title
		t.Invalidate()
	}
}

// Selected is the selected tab's index (-1 without tabs).
func (t *BrowserTabs) Selected() int { return t.sel }

// Select selects tab i, reveals it, and reports it to OnSelect when it was
// not selected.
func (t *BrowserTabs) Select(i int) {
	if i < 0 || i >= len(t.tabs) {
		return
	}
	changed := i != t.sel
	t.sel = i
	t.reveal(i)
	t.Invalidate()
	if changed && t.OnSelect != nil {
		t.OnSelect(i)
	}
}

// AddTab appends tab and returns its index (it is not selected).
func (t *BrowserTabs) AddTab(tab BrowserTab) int {
	t.InsertTab(len(t.tabs), tab)
	return len(t.tabs) - 1
}

// InsertTab puts tab at index i (the selection stays on its tab).
func (t *BrowserTabs) InsertTab(i int, tab BrowserTab) {
	i = min(max(i, 0), len(t.tabs))
	t.tabs = append(t.tabs, BrowserTab{})
	copy(t.tabs[i+1:], t.tabs[i:])
	t.tabs[i] = tab
	switch {
	case t.sel < 0:
		t.sel = 0
	case i <= t.sel:
		t.sel++
	}
	t.drag = tabDrag{}
	t.RequestLayout()
	t.Invalidate()
}

// RemoveTab takes tab i out. Closing the selected tab selects the one after
// it (the one before, when it was the last), as browsers do, and reports
// that to OnSelect.
func (t *BrowserTabs) RemoveTab(i int) {
	if i < 0 || i >= len(t.tabs) {
		return
	}
	t.tabs = append(t.tabs[:i], t.tabs[i+1:]...)
	t.drag = tabDrag{}
	t.hover, t.press = stripHit{tab: -1}, stripHit{tab: -1}
	t.RequestLayout()
	t.Invalidate()
	switch {
	case len(t.tabs) == 0:
		t.sel = -1
	case i < t.sel:
		t.sel--
	case i == t.sel:
		t.sel = min(i, len(t.tabs)-1)
		t.reveal(t.sel)
		if t.OnSelect != nil {
			t.OnSelect(t.sel)
		}
	}
	t.clampScroll()
}

// MoveTab puts tab from at index to; the selection stays on its tab.
func (t *BrowserTabs) MoveTab(from, to int) {
	n := len(t.tabs)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return
	}
	tab := t.tabs[from]
	if from < to {
		copy(t.tabs[from:to], t.tabs[from+1:to+1])
	} else {
		copy(t.tabs[to+1:from+1], t.tabs[to:from])
	}
	t.tabs[to] = tab
	switch {
	case t.sel == from:
		t.sel = to
	case from < t.sel && t.sel <= to:
		t.sel--
	case to <= t.sel && t.sel < from:
		t.sel++
	}
	t.Invalidate()
}

// CloseTab is the user closing tab i: OnClose, or RemoveTab without it.
func (t *BrowserTabs) CloseTab(i int) {
	if i < 0 || i >= len(t.tabs) || t.tabs[i].NoClose {
		return
	}
	if t.OnClose != nil {
		t.OnClose(i)
		return
	}
	t.RemoveTab(i)
}

// NewTab is the user asking for a new tab (OnNew).
func (t *BrowserTabs) NewTab() {
	if t.OnNew != nil {
		t.OnNew()
	}
}

// SelectNext and SelectPrev select the tab after / before the selected one, wrapping
// round (Ctrl+Tab, Ctrl+Shift+Tab).
func (t *BrowserTabs) SelectNext() { t.step(1) }
func (t *BrowserTabs) SelectPrev() { t.step(-1) }

func (t *BrowserTabs) step(d int) {
	if n := len(t.tabs); n > 0 {
		t.Select(((t.sel+d)%n + n) % n)
	}
}

// Shortcut runs the browser shortcuts an app gives its tab strip, from the
// app's own key handler (a window's content root): Ctrl+Tab and
// Ctrl+PageDown the next tab, Ctrl+Shift+Tab and Ctrl+PageUp the previous,
// Ctrl+W and Ctrl+F4 close the selected tab, Ctrl+T a new one. It reports
// whether it took the key.
func (t *BrowserTabs) Shortcut(e widget.KeyEvent) bool {
	if !e.Mods.Ctrl() || e.Mods.Alt() || !t.Enabled() {
		return false
	}
	switch e.Key {
	case platform.KeyTab:
		if e.Mods.Shift() {
			t.SelectPrev()
		} else {
			t.SelectNext()
		}
	case platform.KeyPageDown:
		t.SelectNext()
	case platform.KeyPageUp:
		t.SelectPrev()
	case platform.KeyW, platform.KeyF4:
		if e.Mods.Shift() {
			return false
		}
		t.CloseTab(t.sel)
	case platform.KeyT:
		if e.Mods.Shift() || t.OnNew == nil {
			return false
		}
		t.NewTab()
	default:
		return false
	}
	return true
}

// ---- geometry ---------------------------------------------------------------

// stripGeom is the strip laid out: every tab's slot (scrolled), the tab
// viewport, the new-tab and scroll buttons, and the tab box's height.
type stripGeom struct {
	slots      []paintengine2d.Rect
	view       paintengine2d.Rect
	newBtn     paintengine2d.Rect
	prev, next paintengine2d.Rect
	overflow   bool
	maxScroll  float32
	tabW, tabH float32
}

func (t *BrowserTabs) dip(v float32) float32 {
	return float32(math.Round(float64(style.Dip(t.Look(), v))))
}

// tabHeight is a tab's natural height.
func (t *BrowserTabs) tabHeight() float32 {
	h := t.Look().Metrics().TabH
	if h <= 0 {
		h = t.dip(30)
	}
	return float32(math.Ceil(float64(h)))
}

// onCaption reports whether the strip is in a merged frame's caption, where
// the look's caption band is its background.
func (t *BrowserTabs) onCaption() bool {
	for p := t.Parent(); p != nil; p = p.Parent() {
		if hb, ok := p.(*HeaderBar); ok {
			return hb.Framed() && !hb.Stacked()
		}
	}
	return false
}

// geom lays the strip out in its bounds.
func (t *BrowserTabs) geom() stripGeom {
	lk := t.Look()
	b := t.LocalBounds()
	var g stripGeom
	g.tabH = min(t.tabHeight(), b.Dy())
	top := b.Dy() - g.tabH
	out := style.BrowserTabOutsetOf(lk)
	pad := max(float32(math.Ceil(float64(out.Left))), t.dip(2))
	maxW := t.dip(200)
	if t.MaxTabWidth > 0 {
		maxW = t.dip(t.MaxTabWidth)
	}
	minW := t.dip(80)
	if t.MinTabWidth > 0 {
		minW = t.dip(t.MinTabWidth)
	}
	minW = min(minW, maxW)
	btn := min(g.tabH, t.dip(28))
	gap := t.dip(4)
	// A little caption is always left at the end, to grab the window by.
	reserve := t.dip(24)
	newW := float32(0)
	if t.OnNew != nil {
		newW = gap + btn
	}
	n := len(t.tabs)
	ov := style.TabOverlapOf(lk)
	room := b.Dx() - pad - newW - reserve
	g.tabW = maxW
	if n > 0 {
		g.tabW = min(maxW, max(minW, (room+ov*float32(n-1))/float32(n)))
		g.tabW = float32(math.Floor(float64(g.tabW)))
	}
	total := float32(n)*g.tabW - ov*float32(max(n-1, 0))
	arrows := float32(0)
	if n > 0 && total > room+0.5 {
		g.overflow = true
		arrows = 2 * t.dip(22)
		room -= arrows
	}
	g.view = paintengine2d.XYWH(0, 0, max(pad+room, 0), b.Dy())
	if !g.overflow {
		g.view.Max.X = min(pad+total+out.Right, b.Dx())
	}
	g.maxScroll = max(total-room, 0)
	x := pad - min(t.scroll, g.maxScroll)
	g.slots = make([]paintengine2d.Rect, n)
	for i := range g.slots {
		g.slots[i] = paintengine2d.XYWH(x, top, g.tabW, g.tabH)
		x += g.tabW - ov
	}
	end := pad + total
	if g.overflow {
		end = g.view.Max.X
		aw := arrows * 0.5
		g.prev = paintengine2d.XYWH(end, top, aw, g.tabH)
		g.next = paintengine2d.XYWH(end+aw, top, aw, g.tabH)
		end += arrows
	}
	if t.OnNew != nil {
		g.newBtn = paintengine2d.XYWH(end+gap, top+float32(math.Round(float64(g.tabH-btn)*0.5)), btn, btn)
	}
	return g
}

// closeRect is tab slot s's close button.
func (t *BrowserTabs) closeRect(s paintengine2d.Rect) paintengine2d.Rect {
	side := min(t.dip(18), s.Dy()-t.dip(4))
	return paintengine2d.XYWH(s.Max.X-t.dip(6)-side, s.Min.Y+float32(math.Round(float64(s.Dy()-side)*0.5)), side, side)
}

// hasClose reports whether tab i shows its close button now: the selected
// tab and the one under the pointer, when wide enough.
func (t *BrowserTabs) hasClose(i int, g stripGeom) bool {
	if i < 0 || i >= len(t.tabs) || t.tabs[i].NoClose || g.tabW < t.dip(56) {
		return false
	}
	return i == t.sel || i == t.hover.tab
}

// closable reports whether tab i keeps room for a close button.
func (t *BrowserTabs) closable(i int, g stripGeom) bool {
	return i >= 0 && i < len(t.tabs) && !t.tabs[i].NoClose && g.tabW >= t.dip(56)
}

// slotOf is where tab i paints now: its slot, or under the pointer while
// dragged, the others making room.
func (t *BrowserTabs) slotOf(i int, g stripGeom) paintengine2d.Rect {
	if !t.drag.active || i < 0 || i >= len(g.slots) {
		if i >= 0 && i < len(g.slots) {
			return g.slots[i]
		}
		return paintengine2d.Rect{}
	}
	d := t.drag.tab
	if i == d {
		s := g.slots[d]
		return paintengine2d.XYWH(t.drag.x, s.Min.Y, s.Dx(), s.Dy())
	}
	to := t.dropIndex(g)
	step := g.slots[min(1, len(g.slots)-1)].Min.X - g.slots[0].Min.X
	switch {
	case d < i && i <= to:
		return g.slots[i].Translate(paintengine2d.Pt(-step, 0))
	case to <= i && i < d:
		return g.slots[i].Translate(paintengine2d.Pt(step, 0))
	}
	return g.slots[i]
}

// dropIndex is where the dragged tab lands: the slot its centre is over.
func (t *BrowserTabs) dropIndex(g stripGeom) int {
	n := len(g.slots)
	if n == 0 {
		return -1
	}
	cx := t.drag.x + g.tabW*0.5
	best, dist := 0, float32(math.MaxFloat32)
	for i, s := range g.slots {
		if d := float32(math.Abs(float64((s.Min.X+s.Max.X)*0.5 - cx))); d < dist {
			best, dist = i, d
		}
	}
	return best
}

// hitAt is the part of the strip at local point p.
func (t *BrowserTabs) hitAt(p paintengine2d.Point) stripHit {
	g := t.geom()
	none := stripHit{tab: -1}
	if !t.LocalBounds().Contains(p) {
		return none
	}
	switch {
	case g.newBtn.Contains(p):
		return stripHit{part: partNew, tab: -1}
	case g.prev.Contains(p):
		return stripHit{part: partPrev, tab: -1}
	case g.next.Contains(p):
		return stripHit{part: partNext, tab: -1}
	}
	if !g.view.Contains(p) {
		return none
	}
	// Later tabs win where neighbours overlap; the selected one over all.
	order := make([]int, 0, len(t.tabs))
	if t.sel >= 0 && t.sel < len(t.tabs) {
		order = append(order, t.sel)
	}
	for i := len(t.tabs) - 1; i >= 0; i-- {
		if i != t.sel {
			order = append(order, i)
		}
	}
	for _, i := range order {
		s := t.slotOf(i, g)
		if !s.Contains(p) {
			continue
		}
		if t.closable(i, g) && t.closeRect(s).Inset(-1).Contains(p) {
			return stripHit{part: partClose, tab: i}
		}
		return stripHit{part: partTab, tab: i}
	}
	return none
}

// reveal scrolls tab i fully into view.
func (t *BrowserTabs) reveal(i int) {
	g := t.geom()
	if !g.overflow || i < 0 || i >= len(g.slots) {
		return
	}
	s := g.slots[i]
	if s.Min.X < g.view.Min.X {
		t.scroll -= g.view.Min.X - s.Min.X + t.dip(2)
	} else if s.Max.X > g.view.Max.X {
		t.scroll += s.Max.X - g.view.Max.X + t.dip(2)
	}
	t.clampScroll()
}

func (t *BrowserTabs) clampScroll() {
	t.scroll = min(max(t.scroll, 0), t.geom().maxScroll)
}

// ---- layout and painting --------------------------------------------------

func (t *BrowserTabs) Measure(c layout.Constraints) paintengine2d.Point {
	h := t.tabHeight()
	if t.onCaption() {
		// Room above the tabs in a caption: the band shows over them.
		h += t.dip(6)
	}
	w := float32(len(t.tabs))*t.dip(160) + t.dip(60)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (t *BrowserTabs) Arrange(r paintengine2d.Rect) {
	t.SetBounds(r)
	t.clampScroll()
}

// FillsCaption: in a header bar the strip takes the caption's whole height,
// its tabs standing on the content below.
func (t *BrowserTabs) FillsCaption() bool { return true }

// tabState is tab i's state for the look.
func (t *BrowserTabs) tabState(i int) style.ControlState {
	st := t.State() &^ (style.StateHovered | style.StatePressed | style.StateFocused)
	if (t.hover.tab == i && t.hover.part == partTab) || (t.drag.active && t.drag.tab == i) {
		st |= style.StateHovered
	}
	if t.press.tab == i && t.press.part == partTab {
		st |= style.StatePressed
	}
	if i == 0 {
		st |= style.StateFirst
	}
	if i == len(t.tabs)-1 {
		st |= style.StateLast
	}
	return st
}

// label is tab i's title as it fits its slot s, and whether it was elided.
func (t *BrowserTabs) label(i int, s paintengine2d.Rect, g stripGeom) (string, bool) {
	title := t.tabs[i].Title
	f := style.ControlFontOf(t.Look(), style.RoleTab)
	if f == nil || title == "" {
		return title, false
	}
	room := s.Dx() - 2*t.dip(12)
	if t.closable(i, g) {
		// Room for the close button on both sides keeps the label centred.
		room = s.Dx() - 2*(s.Max.X-t.closeRect(s).Min.X+t.dip(4))
	}
	room = max(room, t.dip(8))
	if f.Advance(title) <= room {
		return title, false
	}
	return f.Fit(title, room), true
}

func (t *BrowserTabs) Paint(ctx *paintengine2d.Context) {
	lk := t.Look()
	b := t.LocalBounds()
	g := t.geom()
	if !t.onCaption() {
		style.DrawBrowserTabBarOf(lk, ctx, b)
	}
	out := style.BrowserTabOutsetOf(lk)
	ctx.Save()
	ctx.ClipRect(g.view)
	paint := func(i int, sel bool) {
		s := t.slotOf(i, g)
		r := s
		if sel {
			r = paintengine2d.Rect{Min: paintengine2d.Pt(s.Min.X-out.Left, s.Min.Y-out.Top), Max: paintengine2d.Pt(s.Max.X+out.Right, s.Max.Y+out.Bottom)}.Intersect(b)
		}
		text, _ := t.label(i, s, g)
		style.DrawBrowserTabOf(lk, ctx, r, t.tabState(i), text, sel)
		if t.hasClose(i, g) {
			t.paintClose(ctx, i, t.closeRect(s), sel)
		}
		if sel && t.State().Focused() {
			lk.DrawFocusRing(ctx, s.Inset(t.dip(3)))
		}
	}
	for i := range t.tabs {
		if i != t.sel && !(t.drag.active && i == t.drag.tab) {
			paint(i, false)
		}
	}
	if t.sel >= 0 && t.sel < len(t.tabs) && !(t.drag.active && t.sel == t.drag.tab) {
		paint(t.sel, true)
	}
	if t.drag.active {
		paint(t.drag.tab, t.drag.tab == t.sel)
	}
	ctx.Restore()
	if g.overflow {
		t.paintButton(ctx, g.prev, partPrev)
		t.paintButton(ctx, g.next, partNext)
	}
	if !g.newBtn.Empty() {
		t.paintButton(ctx, g.newBtn, partNew)
	}
}

// partState is the pointer's state over a button of the strip.
func (t *BrowserTabs) partState(part stripPart, tab int) style.ControlState {
	st := style.StateAutoRaise
	if !widget.WindowActive(t) {
		st |= style.StateBackdrop
	}
	if !t.Enabled() {
		return st | style.StateDisabled
	}
	if t.hover.part == part && t.hover.tab == tab {
		st |= style.StateHovered
	}
	if t.press.part == part && t.press.tab == tab {
		st |= style.StatePressed | style.StateHovered
	}
	return st
}

// paintClose paints tab i's close button: a glyph in the label's colour,
// the tool face under the pointer.
func (t *BrowserTabs) paintClose(ctx *paintengine2d.Context, i int, cb paintengine2d.Rect, sel bool) {
	lk := t.Look()
	st := t.partState(partClose, i)
	fg := lk.Palette().TextMuted
	if sel || t.hover.tab == i {
		fg = lk.Palette().Text
	}
	if st.Hovered() || st.Pressed() {
		fg = style.DrawFaceOf(lk, ctx, cb, style.RoleTool, st)
	}
	style.DrawCaptionGlyph(ctx, cb, style.CaptionClose, false, fg, style.Dip(lk, 8), max(style.Dip(lk, 1.25), 1))
}

// paintButton paints the new-tab or a scroll button.
func (t *BrowserTabs) paintButton(ctx *paintengine2d.Context, r paintengine2d.Rect, part stripPart) {
	lk := t.Look()
	st := t.partState(part, -1)
	fg := style.DrawFaceOf(lk, ctx, r, style.RoleTool, st)
	switch part {
	case partNew:
		s := t.dip(12)
		lw := max(t.dip(1.5), 1)
		x := float32(math.Round(float64(r.Min.X + (r.Dx()-s)*0.5)))
		y := float32(math.Round(float64(r.Min.Y + (r.Dy()-s)*0.5)))
		c := float32(math.Round(float64(s-lw) * 0.5))
		p := paintengine2d.NewPath()
		p.AddRect(paintengine2d.XYWH(x, y+c, s, lw))
		p.AddRect(paintengine2d.XYWH(x+c, y, lw, s))
		ctx.DrawPath(p, paintengine2d.Fill(fg))
	case partPrev:
		style.DrawArrowOf(lk, ctx, r.Inset(r.Dx()*0.25), style.DirLeft, fg)
	case partNext:
		style.DrawArrowOf(lk, ctx, r.Inset(r.Dx()*0.25), style.DirRight, fg)
	}
}

// ---- input --------------------------------------------------------------------

// CaptionAt: the strip is caption where there is no tab or button.
func (t *BrowserTabs) CaptionAt(p paintengine2d.Point) bool { return t.hitAt(p).part == partNone }

// CaptionMenu implements widget.CaptionMenuer: the strip's own menu for a
// right-click on its caption space.
func (t *BrowserTabs) CaptionMenu(p paintengine2d.Point) bool {
	return t.OnContextMenu != nil && t.OnContextMenu(-1, p)
}

// FocusOnClick is false: clicking a tab leaves the keyboard focus where it
// was (in the page), as browsers do; Tab reaches the strip.
func (t *BrowserTabs) FocusOnClick() bool { return false }

// Tooltip: the full title of a hovered tab whose title is elided (or its
// own tip), and the buttons' names.
func (t *BrowserTabs) Tooltip() string {
	switch t.hover.part {
	case partClose:
		return "Close Tab"
	case partNew:
		return "New Tab"
	case partPrev:
		return "Scroll Tabs Left"
	case partNext:
		return "Scroll Tabs Right"
	case partTab:
		i := t.hover.tab
		if i < 0 || i >= len(t.tabs) {
			return ""
		}
		if t.tabs[i].Tip != "" {
			return t.tabs[i].Tip
		}
		g := t.geom()
		if _, elided := t.label(i, g.slots[i], g); elided {
			return t.tabs[i].Title
		}
	}
	return ""
}

func (t *BrowserTabs) setHover(h stripHit) {
	if h != t.hover {
		t.hover = h
		t.Invalidate()
	}
}

func (t *BrowserTabs) MouseMove(e widget.MouseEvent) bool {
	if t.drag.armed && !t.drag.active {
		if d := e.Pos.Sub(t.drag.from); d.X*d.X+d.Y*d.Y >= t.dip(8)*t.dip(8) {
			t.drag.active = true
			t.press = stripHit{tab: -1}
		}
	}
	if t.drag.active {
		g := t.geom()
		lo := g.view.Min.X
		hi := max(g.view.Max.X-g.tabW, lo)
		t.drag.x = min(max(e.Pos.X-t.drag.grab, lo), hi)
		t.Invalidate()
		return true
	}
	t.setHover(t.hitAt(e.Pos))
	return true
}

func (t *BrowserTabs) MouseExit() {
	t.hover = stripHit{tab: -1}
	if !t.drag.active {
		t.press = stripHit{tab: -1}
	}
	t.Invalidate()
	t.Base.MouseExit()
}

func (t *BrowserTabs) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	h := t.hitAt(e.Pos)
	switch e.Button {
	case platform.ButtonLeft:
		t.press = h
		if h.part == partTab {
			// Browsers select on the press; the press may become a drag.
			t.Select(h.tab)
			g := t.geom()
			t.drag = tabDrag{armed: true, tab: h.tab, from: e.Pos, grab: e.Pos.X - g.slots[h.tab].Min.X, x: g.slots[h.tab].Min.X}
		}
		t.Invalidate()
		return h.part != partNone
	case platform.ButtonMiddle:
		if h.part == partTab || h.part == partClose {
			t.CloseTab(h.tab)
			return true
		}
	case platform.ButtonRight:
		if h.part == partTab || h.part == partClose {
			if t.OnContextMenu != nil {
				return t.OnContextMenu(h.tab, widget.DeviceOrigin(t).Add(e.Pos))
			}
			return true
		}
	}
	return false
}

func (t *BrowserTabs) MouseRelease(e widget.MouseEvent) bool {
	was := t.press
	t.press = stripHit{tab: -1}
	if t.drag.active {
		g := t.geom()
		from, to := t.drag.tab, t.dropIndex(g)
		t.drag = tabDrag{}
		if to >= 0 && to != from {
			t.MoveTab(from, to)
			t.reveal(to)
			if t.OnReorder != nil {
				t.OnReorder(from, to)
			}
		}
		t.setHover(t.hitAt(e.Pos))
		t.Invalidate()
		return true
	}
	t.drag = tabDrag{}
	if e.Button != platform.ButtonLeft {
		return false
	}
	h := t.hitAt(e.Pos)
	if h == was {
		switch h.part {
		case partClose:
			t.CloseTab(h.tab)
		case partNew:
			t.NewTab()
		case partPrev, partNext:
			d := t.geom().tabW
			if h.part == partPrev {
				d = -d
			}
			t.scroll += d
			t.clampScroll()
		}
	}
	t.setHover(t.hitAt(e.Pos))
	t.Invalidate()
	return true
}

// MouseWheel scrolls an overflowing strip.
func (t *BrowserTabs) MouseWheel(e widget.MouseEvent) bool {
	g := t.geom()
	if !g.overflow {
		return false
	}
	d := e.Scroll.Y
	if e.Scroll.X != 0 {
		d = e.Scroll.X
	}
	if !e.Precise {
		d *= g.tabW * 0.5
	}
	t.scroll += d
	t.clampScroll()
	t.Invalidate()
	return true
}

func (t *BrowserTabs) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() || e.Mods.Ctrl() || e.Mods.Alt() {
		return false
	}
	t.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeyLeft:
		t.Select(t.sel - 1)
	case platform.KeyRight:
		t.Select(t.sel + 1)
	case platform.KeyHome:
		t.Select(0)
	case platform.KeyEnd:
		t.Select(len(t.tabs) - 1)
	case platform.KeyMenu:
		if t.OnContextMenu == nil || t.sel < 0 {
			return false
		}
		s := t.geom().slots[t.sel]
		return t.OnContextMenu(t.sel, widget.DeviceOrigin(t).Add(paintengine2d.Pt(s.Min.X, s.Max.Y)))
	default:
		return false
	}
	return true
}

// ---- accessibility ------------------------------------------------------------

// Describe implements widget.Accessible: a page tab list.
func (t *BrowserTabs) Describe(n *a11y.Node) {
	n.Role = a11y.RoleTabList
	nameOr(n, "Tabs")
}

// AccessibleItems are the page tabs, each holding its close button, then
// the new-tab button. Items number the tabs 0…n-1, their close buttons
// n…2n-1, the new-tab button 2n.
func (t *BrowserTabs) AccessibleItems() []*a11y.Node {
	g := t.geom()
	n := len(t.tabs)
	out := make([]*a11y.Node, 0, n+1)
	for i, tab := range t.tabs {
		s := g.slots[i]
		it := item(t, i, a11y.RoleTab, tab.Title, s.Intersect(g.view))
		if s.Intersect(g.view).Empty() {
			it.Bounds = widget.LocalToWindow(t, s)
			it.State |= a11y.StateOffscreen
		}
		it.Index, it.Count = i+1, n
		it.State |= a11y.StateSelectable
		if i == t.sel {
			it.State |= a11y.StateSelected
		}
		it.Actions = it.Actions.With(a11y.ActionDefault)
		if !tab.NoClose {
			cb := item(t, n+i, a11y.RoleButton, "Close "+tab.Title, t.closeRect(s))
			cb.State |= it.State & a11y.StateOffscreen
			cb.Actions = cb.Actions.With(a11y.ActionDefault)
			it.Children = append(it.Children, cb)
		}
		out = append(out, it)
	}
	if !g.newBtn.Empty() {
		nb := item(t, 2*n, a11y.RoleButton, "New Tab", g.newBtn)
		nb.Actions = nb.Actions.With(a11y.ActionDefault)
		out = append(out, nb)
	}
	return out
}

// AccessibleAction selects a tab, presses a tab's close button or the
// new-tab button.
func (t *BrowserTabs) AccessibleAction(i int, a a11y.Action) bool {
	n := len(t.tabs)
	if a != a11y.ActionDefault || !t.Enabled() || i < 0 {
		return false
	}
	switch {
	case i < n:
		t.Select(i)
	case i < 2*n:
		if t.tabs[i-n].NoClose {
			return false
		}
		t.CloseTab(i - n)
	case i == 2*n && t.OnNew != nil:
		t.NewTab()
	default:
		return false
	}
	return true
}

// AccessibleFocusItem is the selected tab.
func (t *BrowserTabs) AccessibleFocusItem() int {
	if t.sel >= 0 && t.sel < len(t.tabs) {
		return t.sel
	}
	return -1
}
