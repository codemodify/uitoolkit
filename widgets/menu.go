package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// MenuItem is one row in a Menu / PopupMenu (dropdown and context share this).
type MenuItem struct {
	Text       string
	Shortcut   string
	Disabled   bool
	Separator  bool
	Icon       style.ToolIcon // optional gutter glyph; check/radio wins when on
	Checkable  bool           // click / keyboard toggles Checked
	Checked    bool
	RadioGroup string // exclusive with siblings that share the name
	Submenu    []*MenuItem
	OnClick    func()
}

// HasSubmenu reports whether this row opens a child popup.
func (it *MenuItem) HasSubmenu() bool {
	return it != nil && len(it.Submenu) > 0
}

// Item is an enabled command row.
func Item(text string, on func()) *MenuItem { return &MenuItem{Text: text, OnClick: on} }

// ItemAccel is a command row with a shortcut hint (display only).
func ItemAccel(text, shortcut string, on func()) *MenuItem {
	return &MenuItem{Text: text, Shortcut: shortcut, OnClick: on}
}

// ItemIcon is a command row with a leading gutter icon.
func ItemIcon(icon style.ToolIcon, text string, on func()) *MenuItem {
	return &MenuItem{Icon: icon, Text: text, OnClick: on}
}

// ItemIconAccel is ItemIcon plus a shortcut hint.
func ItemIconAccel(icon style.ToolIcon, text, shortcut string, on func()) *MenuItem {
	return &MenuItem{Icon: icon, Text: text, Shortcut: shortcut, OnClick: on}
}

// Sep is a horizontal rule between items.
func Sep() *MenuItem { return &MenuItem{Separator: true} }

// CheckItem is a checkable row (click / Return / Space toggles Checked).
func CheckItem(text string, checked bool, on func()) *MenuItem {
	return &MenuItem{Text: text, Checkable: true, Checked: checked, OnClick: on}
}

// RadioItem is an exclusive choice in group (empty group is still exclusive
// among other empty-group radios in the same popup — prefer a named group).
func RadioItem(text, group string, checked bool, on func()) *MenuItem {
	if group == "" {
		group = "radio"
	}
	return &MenuItem{Text: text, RadioGroup: group, Checked: checked, OnClick: on}
}

// Submenu is a cascade parent row. Hover or click opens items to the right.
func Submenu(text string, items ...*MenuItem) *MenuItem {
	return &MenuItem{Text: text, Submenu: items}
}

func (it *MenuItem) isRadio() bool {
	return it != nil && it.RadioGroup != ""
}

// Menu is a titled drop-down attached to a MenuBar.
type Menu struct {
	Title string
	Items []*MenuItem
}

// NewMenu builds a named menu.
func NewMenu(title string, items ...*MenuItem) *Menu {
	return &Menu{Title: title, Items: items}
}

// MenuBar is a horizontal strip of menu titles.
type MenuBar struct {
	widget.Base
	menus  []*Menu
	open   int
	hover  int
	press  int
	focus  int
	keyNav bool
}

// NewMenuBar constructs a menu bar.
func NewMenuBar(menus ...*Menu) *MenuBar {
	m := &MenuBar{menus: menus, open: -1, hover: -1, press: -1, focus: 0}
	m.Init(m)
	m.SetWantsFocus(true)
	return m
}

// Menus returns the attached menus.
func (m *MenuBar) Menus() []*Menu { return m.menus }

// OpenIndex is the open menu, or -1.
func (m *MenuBar) OpenIndex() int { return m.open }

// HoverIndex is the title under the pointer, or -1.
func (m *MenuBar) HoverIndex() int { return m.hover }

func (m *MenuBar) RetainsPointer() bool { return true }

func (m *MenuBar) barH() float32 {
	h := m.Look().Metrics().MenuBarH
	if h <= 0 {
		h = 28
	}
	return h
}

func (m *MenuBar) titlesWidth() float32 {
	f := m.Look().Font()
	x := float32(8)
	for _, menu := range m.menus {
		if menu == nil {
			continue
		}
		label, _, _ := ParseMnemonic(menu.Title)
		x += f.Advance(label) + 20
	}
	return x
}

func (m *MenuBar) Measure(c layout.Constraints) paintengine2d.Point {
	// Intrinsic title strip only. Do not expand to MaxW: a Row/Flex parent
	// decides growth (Mail puts M on the left of a chrome row). A column
	// AlignStretch still arranges the bar to the full cross-axis width.
	w := m.titlesWidth()
	if w < 1 {
		w = 1
	}
	return c.Constrain(paintengine2d.Pt(w, m.barH()))
}

func (m *MenuBar) Arrange(r paintengine2d.Rect) { m.SetBounds(r) }

func (m *MenuBar) titleRects() []paintengine2d.Rect {
	f := m.Look().Font()
	h := m.LocalBounds().Dy()
	x := float32(4)
	out := make([]paintengine2d.Rect, len(m.menus))
	for i, menu := range m.menus {
		label, _, _ := ParseMnemonic(menu.Title)
		tw := f.Advance(label) + 20
		out[i] = paintengine2d.XYWH(x, 0, tw, h)
		x += tw
	}
	return out
}

func (m *MenuBar) titleAt(x float32) int {
	for i, r := range m.titleRects() {
		if x >= r.Min.X && x < r.Max.X {
			return i
		}
	}
	return -1
}

func (m *MenuBar) Paint(ctx *paintengine2d.Context) {
	lk := m.Look()
	lk.DrawMenuBar(ctx, m.LocalBounds())
	ctx.Save()
	ctx.ClipRect(m.LocalBounds())
	rects := m.titleRects()
	for i, menu := range m.menus {
		// The bar's widget StateHovered is true for any pointer on the
		// strip (including empty space). Titles must not inherit it or
		// every label paints the XP fill — a full-bar wash.
		st := m.State()
		st &^= style.StateHovered | style.StatePressed
		if i == m.hover {
			st |= style.StateHovered
		}
		if i == m.press {
			st |= style.StatePressed
		}
		if i != m.focus || !m.keyNav || i == m.open {
			st &^= style.StateFocused
		}
		label, _, idx := ParseMnemonic(menu.Title)
		lk.DrawMenuTitle(ctx, rects[i], st, label, idx, i == m.open)
	}
	ctx.Restore()
}

// KeyboardChrome is true when a title is painting a keyboard focus ring.
func (m *MenuBar) KeyboardChrome() bool {
	return m.keyNav && m.focus >= 0 && m.open < 0
}

// TitleRect is the local box of menu title i.
func (m *MenuBar) TitleRect(i int) paintengine2d.Rect {
	rects := m.titleRects()
	if i < 0 || i >= len(rects) {
		return paintengine2d.Rect{}
	}
	return rects[i]
}

func (m *MenuBar) invalidateTitle(i int) {
	rects := m.titleRects()
	if i < 0 || i >= len(rects) {
		return
	}
	m.InvalidateRect(rects[i].Inset(-1))
}

func (m *MenuBar) MouseMove(e widget.MouseEvent) bool {
	i := m.titleAt(e.Pos.X)
	if i != m.hover {
		old := m.hover
		m.hover = i
		m.invalidateTitle(old)
		m.invalidateTitle(i)
	}
	if m.open >= 0 && i >= 0 && i != m.open {
		m.Open(i)
	}
	return true
}

func (m *MenuBar) MouseExit() {
	m.hover = -1
	m.press = -1
	m.Base.MouseExit()
}

func (m *MenuBar) FocusLost() {
	m.keyNav = false
	m.focus = -1
	m.Base.FocusLost()
}

func (m *MenuBar) MousePress(e widget.MouseEvent) bool {
	if !m.Enabled() || e.Button == platform.ButtonRight {
		return false
	}
	m.keyNav = false
	m.RequestFocus()
	i := m.titleAt(e.Pos.X)
	m.press = i
	if i < 0 {
		return true
	}
	m.focus = i
	if m.open == i {
		m.Close()
		return true
	}
	m.Open(i)
	return true
}

func (m *MenuBar) MouseRelease(widget.MouseEvent) bool {
	m.press = -1
	m.Invalidate()
	return true
}

func (m *MenuBar) KeyPress(e widget.KeyEvent) bool {
	if !m.Enabled() || len(m.menus) == 0 {
		return false
	}
	m.keyNav = true
	if m.focus < 0 {
		m.focus = 0
	}
	switch e.Key {
	case platform.KeyLeft:
		m.focus = (m.focus - 1 + len(m.menus)) % len(m.menus)
		if m.open >= 0 {
			m.Open(m.focus)
		} else {
			m.Invalidate()
		}
		return true
	case platform.KeyRight:
		m.focus = (m.focus + 1) % len(m.menus)
		if m.open >= 0 {
			m.Open(m.focus)
		} else {
			m.Invalidate()
		}
		return true
	case platform.KeyDown, platform.KeyReturn, platform.KeySpace:
		if m.open == m.focus {
			m.Close()
		} else {
			m.Open(m.focus)
		}
		return true
	}
	return false
}

// HandleAlt opens the menu whose mnemonic matches key.
func (m *MenuBar) HandleAlt(key platform.Key) bool {
	if !m.Enabled() || !m.Visible() {
		return false
	}
	for i, menu := range m.menus {
		_, k, _ := ParseMnemonic(menu.Title)
		if k != platform.KeyUnknown && k == key {
			m.keyNav = true
			m.RequestFocus()
			m.focus = i
			m.Open(i)
			return true
		}
	}
	return false
}

// Open drops the i-th menu.
func (m *MenuBar) Open(i int) {
	if i < 0 || i >= len(m.menus) {
		return
	}
	rects := m.titleRects()
	pop := NewPopupMenu(m.menus[i].Items...)
	pop.OnPick = func(it *MenuItem) {
		m.open = -1
		m.keyNav = false
		m.focus = -1
		widget.DismissPopup(m)
		if it != nil && it.OnClick != nil {
			it.OnClick()
		}
	}
	pop.OnDismiss = func() {
		// Switching menus (Left / Right / hover) shows the next popup, which
		// dismisses this one. Only the still-current menu may clear bar state,
		// or the arrow keys would reset focus and stick on one title.
		if m.open != i {
			return
		}
		m.open = -1
		m.keyNav = false
		m.focus = -1
		m.Invalidate()
	}
	origin := widget.DeviceOrigin(m)
	tb := rects[i]
	anchor := paintengine2d.XYWH(origin.X+tb.Min.X, origin.Y+tb.Min.Y, tb.Dx(), tb.Dy())
	// Flush under the title so DrawMenuTitle(open) shares an edge with the popup.
	widget.PlacePopupForAnchor(m, pop, anchor, 0, 0)
	pop.RestoreFocusTo(m)
	prev := m.open
	// Set before ShowPopup: the outgoing popup's OnDismiss runs inside it and
	// must see that a sibling menu is now the current one.
	m.open = i
	if widget.ShowPopup(m, pop) {
		m.Invalidate()
		pop.RequestFocus()
		return
	}
	m.open = prev
}

// Close dismisses the open menu.
func (m *MenuBar) Close() {
	m.open = -1
	m.keyNav = false
	m.focus = -1
	widget.DismissPopup(m)
	m.Invalidate()
}

var _ widget.CascadeHost = (*PopupMenu)(nil)

// PopupMenu is a floating list of MenuItems (drop-down or context menu).
type PopupMenu struct {
	widget.Base
	Items      []*MenuItem
	OnPick     func(*MenuItem)
	OnDismiss  func()
	OffsetY    float32
	hover      int
	press      int
	focus      int
	keyNav     bool
	vbar       scrollDrag
	cascade    *PopupMenu
	cascadeIdx int
	parentMenu *PopupMenu
	restore    widget.Component
	dead       bool
	lay        menuLayout
}

// menuLayout caches the per-item geometry for one popup. Without it every
// rowBounds walks the item list (rowTop) and re-measures every label
// (contentSize through innerWidth), making Paint O(n^2) in item count.
type menuLayout struct {
	valid bool
	itemH float32
	padT  float32
	padB  float32
	tops  []float32 // len(Items)+1 offsets from the frame top
	size  paintengine2d.Point
}

// NewPopupMenu builds a popup from items.
func NewPopupMenu(items ...*MenuItem) *PopupMenu {
	p := &PopupMenu{Items: items, hover: -1, press: -1, cascadeIdx: -1, focus: firstEnabled(items)}
	p.Init(p)
	p.SetWantsFocus(true)
	return p
}

func firstEnabled(items []*MenuItem) int {
	for i, it := range items {
		if it != nil && !it.Separator && !it.Disabled {
			return i
		}
	}
	return 0
}

// Presented records the opener as the focus anchor when one was not set
// explicitly (widget.Presenter, called from widget.ShowPopup).
func (p *PopupMenu) Presented(from widget.Component) {
	if p.restore == nil {
		p.restore = from
	}
}

// RestoreFocusTo records the component that opened this popup. When the popup
// is dismissed while it still holds focus, focus returns there instead of
// staying on a detached node that would keep handling keys.
func (p *PopupMenu) RestoreFocusTo(anchor widget.Component) { p.restore = anchor }

// Dead reports whether this popup has been dismissed. A dead popup ignores
// every input event: the window may still hold a reference to it (focus,
// pointer capture) for the rest of the current event.
func (p *PopupMenu) Dead() bool { return p.dead }

// Invalidate also drops the cached item geometry so label / item changes are
// measured again on the next paint.
func (p *PopupMenu) Invalidate() {
	p.lay.valid = false
	p.Base.Invalidate()
}

func (p *PopupMenu) markDead() {
	if p.dead {
		return
	}
	p.dead = true
	p.restoreFocus()
}

func (p *PopupMenu) restoreFocus() {
	widget.RestoreFocus(p, p.restore, p.keyNav)
}

func (p *PopupMenu) itemH() float32 {
	lk := p.Look()
	h := float32(28)
	if lk != nil && lk.Metrics().MenuItemH > 0 {
		h = lk.Metrics().MenuItemH
	}
	if lk != nil {
		if f := lk.Font(); f != nil {
			if min := f.Height() + 16; min > h {
				h = min
			}
		}
	}
	return h
}

func (p *PopupMenu) rowH(it *MenuItem) float32 {
	if it != nil && it.Separator {
		return 8
	}
	return p.itemH()
}

func (p *PopupMenu) chrome() style.MenuChrome {
	return style.MenuChromeFor(p.Look())
}

func menuTextWidth(f *style.Font, text string) float32 {
	if text == "" {
		return 0
	}
	if f != nil {
		if w := f.InkWidth(text); w > 0 {
			return w
		}
		if w := f.Advance(text); w > 0 {
			return w
		}
		if em := f.Size; em > 0 {
			return float32(len([]rune(text))) * em * 0.62
		}
	}
	return float32(len([]rune(text))) * 10
}

// layoutInfo returns the cached item geometry, recomputing it when the
// metrics that feed it changed (density, theme, item count).
func (p *PopupMenu) layoutInfo() *menuLayout {
	ch := p.chrome()
	ih := p.itemH()
	if p.lay.valid && p.lay.itemH == ih && p.lay.padT == ch.PadT && p.lay.padB == ch.PadB &&
		len(p.lay.tops) == len(p.Items)+1 {
		return &p.lay
	}
	p.lay.itemH = ih
	p.lay.padT = ch.PadT
	p.lay.padB = ch.PadB
	p.lay.size = p.measureContent(ch)
	p.lay.valid = true
	return &p.lay
}

func (p *PopupMenu) contentSize() paintengine2d.Point { return p.layoutInfo().size }

func (p *PopupMenu) measureContent(ch style.MenuChrome) paintengine2d.Point {
	lk := p.Look()
	f := (*style.Font)(nil)
	if lk != nil {
		f = lk.Font()
	}
	var maxLabel, maxAccel float32
	hasSub := false
	h := ch.PadT + ch.PadB
	if cap(p.lay.tops) >= len(p.Items)+1 {
		p.lay.tops = p.lay.tops[:0]
	} else {
		p.lay.tops = make([]float32, 0, len(p.Items)+1)
	}
	cur := ch.PadT
	for _, it := range p.Items {
		p.lay.tops = append(p.lay.tops, cur)
		cur += p.rowH(it)
		h += p.rowH(it)
		if it == nil || it.Separator {
			continue
		}
		if it.HasSubmenu() {
			hasSub = true
		}
		label, _, _ := ParseMnemonic(it.Text)
		if tw := menuTextWidth(f, label); tw > maxLabel {
			maxLabel = tw
		}
		if it.Shortcut != "" {
			if tw := menuTextWidth(f, it.Shortcut); tw > maxAccel {
				maxAccel = tw
			}
		}
	}
	w := ch.FrameWidth(maxLabel, maxAccel)
	if hasSub {
		w += ch.SubmenuArrow
	}
	if min := style.Dip(lk, 80); w < min {
		w = min
	}
	p.lay.tops = append(p.lay.tops, cur)
	return paintengine2d.Pt(w, h)
}

// ContentSize is the intrinsic box for all items + separators + frame pad.
func (p *PopupMenu) ContentSize() paintengine2d.Point { return p.contentSize() }

func (p *PopupMenu) Measure(c layout.Constraints) paintengine2d.Point {
	sz := p.contentSize()
	if c.HasMaxH() && sz.Y > c.MaxH {
		sz.X += style.ScrollGutter(p.Look())
	}
	out := c.Constrain(sz)
	if out.Y+0.5 < sz.Y && out.X < sz.X {
		out.X = sz.X
		if c.HasMaxW() && out.X > c.MaxW {
			out.X = c.MaxW
		}
	}
	return out
}

func (p *PopupMenu) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	p.clamp()
}

func (p *PopupMenu) contentH() float32 { return p.layoutInfo().size.Y }

// MaxOffset is max(0, content − viewport) when the popup was clamped.
func (p *PopupMenu) MaxOffset() float32 {
	return layout.MaxScroll(p.contentH(), p.LocalBounds().Dy())
}

func (p *PopupMenu) clamp() {
	p.OffsetY = layout.ClampScroll(p.OffsetY, p.contentH(), p.LocalBounds().Dy())
}

func (p *PopupMenu) vparts() style.ScrollParts {
	return vScrollParts(p.Look(), p.LocalBounds(), p.contentH(), p.OffsetY)
}

func (p *PopupMenu) vaxis() scrollAxis {
	return scrollAxis{
		vertical: true,
		parts:    p.vparts,
		get:      func() (float32, float32) { return p.OffsetY, p.MaxOffset() },
		set:      func(y float32) { p.OffsetY = y; p.clamp(); p.Invalidate() },
		steps: func() (float32, float32) {
			return p.Look().Metrics().MenuItemH, p.LocalBounds().Dy() * 0.9
		},
	}
}

func (p *PopupMenu) scrollTrack() (track, thumb paintengine2d.Rect) {
	sp := p.vparts()
	return sp.Track, sp.Thumb
}

// ScrollTrack is the overflow bar (empty thumb when every item fits).
func (p *PopupMenu) ScrollTrack() (track, thumb paintengine2d.Rect) { return p.scrollTrack() }

func (p *PopupMenu) innerWidth() float32 {
	ch := p.chrome()
	w := p.LocalBounds().Dx() - ch.PadL - ch.PadR
	if p.MaxOffset() > 0 {
		w -= style.ScrollGutter(p.Look())
	}
	if w < 0 {
		w = 0
	}
	return w
}

func (p *PopupMenu) rowTop(i int) float32 {
	tops := p.layoutInfo().tops
	if i < 0 || i >= len(tops) {
		if len(tops) == 0 {
			return p.chrome().PadT
		}
		return tops[len(tops)-1]
	}
	return tops[i]
}

func (p *PopupMenu) rowAt(y float32) int {
	tops := p.layoutInfo().tops
	y += p.OffsetY
	for i := 0; i+1 < len(tops); i++ {
		if y >= tops[i] && y < tops[i+1] {
			return i
		}
	}
	return -1
}

func (p *PopupMenu) rowBounds(i int) paintengine2d.Rect {
	if i < 0 || i >= len(p.Items) {
		return paintengine2d.Rect{}
	}
	ch := p.chrome()
	return paintengine2d.XYWH(ch.PadL, p.rowTop(i)-p.OffsetY, p.innerWidth(), p.rowH(p.Items[i]))
}

// ItemBounds is the arranged row in popup-local coordinates (scroll applied).
func (p *PopupMenu) ItemBounds(i int) paintengine2d.Rect { return p.rowBounds(i) }

// LabelBounds is the text column for item i (excludes the shortcut column).
func (p *PopupMenu) LabelBounds(i int) paintengine2d.Rect {
	row := p.rowBounds(i)
	if row.Empty() || i < 0 || i >= len(p.Items) || p.Items[i] == nil {
		return paintengine2d.Rect{}
	}
	ch := p.chrome()
	x0 := ch.LabelMinX(row.Min.X)
	x1 := ch.LabelMaxX(row.Max.X)
	if r := p.ArrowBounds(i); !r.Empty() {
		x1 = r.Min.X
	}
	if r := p.ShortcutBounds(i); !r.Empty() {
		x1 = r.Min.X - ch.AccelGap
	}
	if x1 < x0 {
		x1 = x0
	}
	return paintengine2d.XYWH(x0, row.Min.Y, x1-x0, row.Dy())
}

// ShortcutBounds is the right-aligned accelerator column, or empty.
func (p *PopupMenu) ShortcutBounds(i int) paintengine2d.Rect {
	if i < 0 || i >= len(p.Items) || p.Items[i] == nil || p.Items[i].Shortcut == "" {
		return paintengine2d.Rect{}
	}
	row := p.rowBounds(i)
	if row.Empty() {
		return paintengine2d.Rect{}
	}
	f := (*style.Font)(nil)
	if lk := p.Look(); lk != nil {
		f = lk.Font()
	}
	ch := p.chrome()
	tw := menuTextWidth(f, p.Items[i].Shortcut)
	right := ch.LabelMaxX(row.Max.X)
	if r := p.ArrowBounds(i); !r.Empty() {
		right = r.Min.X
	}
	return paintengine2d.XYWH(right-tw, row.Min.Y, tw, row.Dy())
}

// ArrowBounds is the trailing submenu chevron column, or empty.
func (p *PopupMenu) ArrowBounds(i int) paintengine2d.Rect {
	if i < 0 || i >= len(p.Items) || !p.Items[i].HasSubmenu() {
		return paintengine2d.Rect{}
	}
	row := p.rowBounds(i)
	if row.Empty() {
		return paintengine2d.Rect{}
	}
	ch := p.chrome()
	aw := ch.SubmenuArrow
	if aw < 1 {
		aw = 10
	}
	return paintengine2d.XYWH(ch.ArrowMinX(row.Max.X), row.Min.Y, aw, row.Dy())
}

// Cascade is the open child popup, if any (widget.CascadeHost).
func (p *PopupMenu) Cascade() widget.Component {
	if p == nil || p.cascade == nil {
		return nil
	}
	return p.cascade
}

// CascadeMenu is the open child PopupMenu, or nil.
func (p *PopupMenu) CascadeMenu() *PopupMenu {
	if p == nil {
		return nil
	}
	return p.cascade
}

func (p *PopupMenu) ensureItemVisible(i int) {
	if i < 0 || i >= len(p.Items) {
		return
	}
	top := p.rowTop(i)
	bot := top + p.rowH(p.Items[i])
	view := p.LocalBounds().Dy()
	if view <= 0 {
		return
	}
	ch := p.chrome()
	if top < p.OffsetY+ch.PadT {
		p.OffsetY = top - ch.PadT
	}
	if bot > p.OffsetY+view-ch.PadB {
		p.OffsetY = bot - view + ch.PadB
	}
	p.clamp()
}

func (p *PopupMenu) Paint(ctx *paintengine2d.Context) {
	p.clamp()
	lk := p.Look()
	b := p.LocalBounds()
	lk.DrawMenuFrame(ctx, b)
	ctx.Save()
	ctx.ClipRect(b.Inset(2))
	for i, it := range p.Items {
		if it == nil {
			continue
		}
		st := style.StateNone
		if it.Disabled {
			st |= style.StateDisabled
		}
		if i == p.hover || (p.keyNav && i == p.focus) {
			st |= style.StateHovered
		}
		if i == p.press {
			st |= style.StatePressed
		}
		label, _, idx := ParseMnemonic(it.Text)
		lk.DrawMenuItem(ctx, p.rowBounds(i), st, style.MenuRow{
			Label: label, Shortcut: it.Shortcut, Underline: idx,
			Separator: it.Separator, Checked: it.Checked, Radio: it.isRadio(),
			Submenu: it.HasSubmenu(), Icon: it.Icon,
		})
	}
	ctx.Restore()
	p.vbar.paint(ctx, lk, p.vparts(), true)
}

func (p *PopupMenu) invalidateRow(i int) {
	r := p.rowBounds(i)
	if !r.Empty() {
		p.InvalidateRect(r.Inset(-1))
	}
}

func (p *PopupMenu) MouseMove(e widget.MouseEvent) bool {
	if p.dead {
		return false
	}
	if handled, dirty := p.vbar.move(e.Pos, p.vaxis()); handled {
		if dirty {
			p.Invalidate()
		}
		return true
	}
	p.keyNav = false
	i := p.rowAt(e.Pos.Y)
	if i != p.hover {
		old := p.hover
		p.hover = i
		if i >= 0 && p.Items[i] != nil && !p.Items[i].Separator {
			p.focus = i
		}
		p.invalidateRow(old)
		p.invalidateRow(i)
	}
	p.syncCascade(i)
	return true
}

func (p *PopupMenu) MouseExit() {
	if p.cascade != nil {
		p.hover = p.cascadeIdx
		return
	}
	p.hover = -1
	p.vbar.exit()
	p.Invalidate()
}

// HighlightedIndex is the pointer-hot or keyboard-current row, or -1.
func (p *PopupMenu) HighlightedIndex() int {
	if p.hover >= 0 {
		return p.hover
	}
	if p.keyNav {
		return p.focus
	}
	return -1
}

func (p *PopupMenu) MousePress(e widget.MouseEvent) bool {
	if p.dead {
		return false
	}
	if p.vbar.press(p, e.Pos, p.vaxis()) {
		p.Invalidate()
		return true
	}
	p.press = p.rowAt(e.Pos.Y)
	p.Invalidate()
	return true
}

func (p *PopupMenu) MouseRelease(e widget.MouseEvent) bool {
	if p.dead {
		return false
	}
	if p.vbar.release() {
		p.press = -1
		p.Invalidate()
		return true
	}
	i := p.rowAt(e.Pos.Y)
	p.press = -1
	p.Invalidate()
	if i >= 0 && i < len(p.Items) {
		p.activate(i)
	}
	return true
}

func (p *PopupMenu) MouseWheel(e widget.MouseEvent) bool {
	if p.dead || p.MaxOffset() <= 0 {
		return false
	}
	before := p.OffsetY
	p.OffsetY += wheelDelta(e.Scroll.Y, p.itemH())
	p.clamp()
	if p.OffsetY == before {
		return false
	}
	p.Invalidate()
	return true
}

func (p *PopupMenu) KeyPress(e widget.KeyEvent) bool {
	if p.dead {
		return false
	}
	p.keyNav = true
	if p.focus < 0 {
		p.focus = firstEnabled(p.Items)
	}
	switch e.Key {
	case platform.KeyUp:
		p.moveFocus(-1)
		return true
	case platform.KeyDown:
		p.moveFocus(1)
		return true
	case platform.KeyHome:
		p.focus = firstEnabled(p.Items)
		p.hover = p.focus
		p.Invalidate()
		return true
	case platform.KeyEnd:
		p.focus = lastEnabled(p.Items)
		p.hover = p.focus
		p.Invalidate()
		return true
	case platform.KeyRight:
		if p.focus >= 0 && p.focus < len(p.Items) && p.Items[p.focus].HasSubmenu() {
			p.openCascade(p.focus)
			if p.cascade != nil {
				p.cascade.keyNav = true
				p.cascade.RequestFocus()
			}
			return true
		}
	case platform.KeyLeft:
		if p.parentMenu != nil {
			parent := p.parentMenu
			parent.closeCascade()
			parent.keyNav = true
			parent.RequestFocus()
			return true
		}
	case platform.KeyReturn, platform.KeySpace:
		p.activate(p.focus)
		return true
	case platform.KeyEscape:
		widget.DismissPopup(p)
		return true
	}
	// Type-ahead must not eat application accelerators: while a popup is open
	// every key comes here first, so Ctrl+C over a context menu would
	// otherwise activate the first item starting with "c".
	if e.Mods.Ctrl() || e.Mods.Alt() {
		return false
	}
	if i := p.indexForKey(e.Key); i >= 0 {
		p.activate(i)
		return true
	}
	return false
}

func (p *PopupMenu) indexForKey(key platform.Key) int {
	r, ok := platform.KeyRune(key)
	if !ok {
		return -1
	}
	mnemonic := -1
	letter := -1
	for i, it := range p.Items {
		if it == nil || it.Separator || it.Disabled {
			continue
		}
		label, k, _ := ParseMnemonic(it.Text)
		if k == key {
			if mnemonic >= 0 {
				return -1
			}
			mnemonic = i
		}
		runes := []rune(label)
		if letter < 0 && len(runes) > 0 && foldLetter(runes[0]) == r {
			letter = i
		}
	}
	if mnemonic >= 0 {
		return mnemonic
	}
	return letter
}

func foldLetter(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r - 'A' + 'a'
	}
	return r
}

func lastEnabled(items []*MenuItem) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i] != nil && !items[i].Separator && !items[i].Disabled {
			return i
		}
	}
	return 0
}

func (p *PopupMenu) moveFocus(dir int) {
	if len(p.Items) == 0 {
		return
	}
	i := p.focus
	for n := 0; n < len(p.Items); n++ {
		i = (i + dir + len(p.Items)) % len(p.Items)
		it := p.Items[i]
		if it != nil && !it.Separator && !it.Disabled {
			p.focus = i
			p.hover = i
			p.ensureItemVisible(i)
			p.Invalidate()
			return
		}
	}
}

func (p *PopupMenu) applyCheck(it *MenuItem) {
	if it == nil {
		return
	}
	if it.RadioGroup != "" {
		for _, o := range p.Items {
			if o != nil && o.RadioGroup == it.RadioGroup {
				o.Checked = o == it
			}
		}
		return
	}
	if it.Checkable {
		it.Checked = !it.Checked
	}
}

func (p *PopupMenu) activate(i int) {
	if p.dead || i < 0 || i >= len(p.Items) {
		return
	}
	it := p.Items[i]
	if it == nil || it.Separator || it.Disabled {
		return
	}
	if it.HasSubmenu() {
		p.openCascade(i)
		return
	}
	p.applyCheck(it)
	if p.OnPick != nil {
		p.OnPick(it)
		return
	}
	widget.DismissPopup(p)
	if it.OnClick != nil {
		it.OnClick()
	}
}

func (p *PopupMenu) Dismissed() {
	p.closeCascade()
	// Mark dead before the callback: OnDismiss may reopen a sibling popup and
	// this one must stop answering keys either way.
	p.markDead()
	if p.OnDismiss != nil {
		p.OnDismiss()
	}
}

func (p *PopupMenu) syncCascade(i int) {
	if i >= 0 && i < len(p.Items) && p.Items[i].HasSubmenu() && !p.Items[i].Disabled {
		p.openCascade(i)
		return
	}
	if i >= 0 {
		p.closeCascade()
	}
}

func (p *PopupMenu) openCascade(i int) {
	if i < 0 || i >= len(p.Items) {
		p.closeCascade()
		return
	}
	it := p.Items[i]
	if it == nil || it.Disabled || !it.HasSubmenu() {
		p.closeCascade()
		return
	}
	if p.cascade != nil && p.cascadeIdx == i {
		return
	}
	p.closeCascade()
	child := NewPopupMenu(it.Submenu...)
	child.parentMenu = p
	child.OnPick = p.OnPick
	child.RestoreFocusTo(p)
	widget.PreparePopup(p, child)
	origin := widget.DeviceOrigin(p)
	row := p.rowBounds(i)
	pb := widget.DeviceBounds(p)
	anchor := paintengine2d.XYWH(pb.Min.X, origin.Y+row.Min.Y, pb.Dx(), row.Dy())
	widget.PlacePopupBeside(p, child, anchor)
	p.cascade = child
	p.cascadeIdx = i
	p.hover = i
	child.Invalidate()
	p.Invalidate()
}

func (p *PopupMenu) closeCascade() {
	child := p.cascade
	if child == nil {
		return
	}
	p.cascade = nil
	p.cascadeIdx = -1
	child.parentMenu = nil
	child.closeCascade()
	// A closed cascade is unreachable but may still be the host focus (it was
	// opened with Right). Mark it dead and hand focus back to this menu.
	child.markDead()
	child.Invalidate()
	p.Invalidate()
}

// ShowContextMenu opens a popup at window-space origin.
func ShowContextMenu(from widget.Component, origin paintengine2d.Point, items ...*MenuItem) *PopupMenu {
	pop := NewPopupMenu(items...)
	widget.PreparePopup(from, pop)
	pop.RestoreFocusTo(from)
	widget.PlacePopup(pop, origin, 360, 480)
	if widget.ShowPopup(from, pop) {
		widget.ClampToSurface(from, pop)
		pop.RequestFocus()
		return pop
	}
	return nil
}
