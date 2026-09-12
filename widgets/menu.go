package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// MenuItem is one row in a Menu / PopupMenu.
type MenuItem struct {
	Text      string
	Shortcut  string
	Disabled  bool
	Separator bool
	Checked   bool
	OnClick   func()
}

// Item is an enabled command row.
func Item(text string, on func()) *MenuItem { return &MenuItem{Text: text, OnClick: on} }

// ItemAccel is a command row with a shortcut hint (display only).
func ItemAccel(text, shortcut string, on func()) *MenuItem {
	return &MenuItem{Text: text, Shortcut: shortcut, OnClick: on}
}

// Sep is a horizontal rule between items.
func Sep() *MenuItem { return &MenuItem{Separator: true} }

// CheckItem is a toggleable row.
func CheckItem(text string, checked bool, on func()) *MenuItem {
	return &MenuItem{Text: text, Checked: checked, OnClick: on}
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
	w := m.titlesWidth()
	if w < 200 {
		w = 200
	}
	if c.HasMaxW() {
		w = c.MaxW
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
		st := m.State()
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
		if m.open == i {
			m.open = -1
		}
		m.keyNav = false
		m.focus = -1
		m.Invalidate()
	}
	origin := widget.DeviceOrigin(m)
	tb := rects[i]
	anchor := paintengine2d.XYWH(origin.X+tb.Min.X, origin.Y+tb.Min.Y, tb.Dx(), tb.Dy())
	widget.PlacePopupForAnchor(m, pop, anchor, 0, 1)
	if widget.ShowPopup(m, pop) {
		m.open = i
		m.Invalidate()
		pop.RequestFocus()
	}
}

// Close dismisses the open menu.
func (m *MenuBar) Close() {
	m.open = -1
	m.keyNav = false
	m.focus = -1
	widget.DismissPopup(m)
	m.Invalidate()
}

// PopupMenu is a floating list of MenuItems (drop-down or context menu).
type PopupMenu struct {
	widget.Base
	Items     []*MenuItem
	OnPick    func(*MenuItem)
	OnDismiss func()
	OffsetY   float32
	hover     int
	press     int
	focus     int
	keyNav    bool
	vbar      scrollDrag
}

// NewPopupMenu builds a popup from items.
func NewPopupMenu(items ...*MenuItem) *PopupMenu {
	p := &PopupMenu{Items: items, hover: -1, press: -1, focus: firstEnabled(items)}
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

func (p *PopupMenu) contentSize() paintengine2d.Point {
	lk := p.Look()
	ch := style.MenuChromeFor(lk)
	f := (*style.Font)(nil)
	if lk != nil {
		f = lk.Font()
	}
	var maxLabel, maxAccel float32
	h := ch.PadT + ch.PadB
	for _, it := range p.Items {
		h += p.rowH(it)
		if it == nil || it.Separator {
			continue
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
	if min := style.Dip(lk, 80); w < min {
		w = min
	}
	return paintengine2d.Pt(w, h)
}

// ContentSize is the intrinsic box for all items + separators + frame pad.
func (p *PopupMenu) ContentSize() paintengine2d.Point { return p.contentSize() }

func (p *PopupMenu) Measure(c layout.Constraints) paintengine2d.Point {
	sz := p.contentSize()
	if c.HasMaxH() && sz.Y > c.MaxH {
		bar, gap := overflowBarSize(p.Look())
		sz.X += bar + gap
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

func (p *PopupMenu) contentH() float32 { return p.contentSize().Y }

// MaxOffset is max(0, content − viewport) when the popup was clamped.
func (p *PopupMenu) MaxOffset() float32 {
	return layout.MaxScroll(p.contentH(), p.LocalBounds().Dy())
}

func (p *PopupMenu) clamp() {
	p.OffsetY = layout.ClampScroll(p.OffsetY, p.contentH(), p.LocalBounds().Dy())
}

func (p *PopupMenu) scrollTrack() (track, thumb paintengine2d.Rect) {
	bar, gap := overflowBarSize(p.Look())
	return vScrollThumb(p.LocalBounds(), p.contentH(), p.OffsetY, bar, gap)
}

// ScrollTrack is the overflow bar (empty thumb when every item fits).
func (p *PopupMenu) ScrollTrack() (track, thumb paintengine2d.Rect) { return p.scrollTrack() }

func (p *PopupMenu) innerWidth() float32 {
	ch := p.chrome()
	w := p.LocalBounds().Dx() - ch.PadL - ch.PadR
	if p.MaxOffset() > 0 {
		bar, gap := overflowBarSize(p.Look())
		w -= bar + gap
	}
	if w < 0 {
		w = 0
	}
	return w
}

func (p *PopupMenu) rowTop(i int) float32 {
	cur := p.chrome().PadT
	for n, it := range p.Items {
		if n == i {
			return cur
		}
		cur += p.rowH(it)
	}
	return cur
}

func (p *PopupMenu) rowAt(y float32) int {
	ch := p.chrome()
	cur := ch.PadT - p.OffsetY
	for i, it := range p.Items {
		rh := p.rowH(it)
		if y >= cur && y < cur+rh {
			return i
		}
		cur += rh
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
	return paintengine2d.XYWH(ch.LabelMaxX(row.Max.X)-tw, row.Min.Y, tw, row.Dy())
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
		lk.DrawMenuItem(ctx, p.rowBounds(i), st, label, it.Shortcut, idx, it.Separator, it.Checked)
	}
	ctx.Restore()
	track, thumb := p.scrollTrack()
	paintOverflowBar(ctx, lk, track, thumb, p.vbar.over, p.vbar.active)
}

func (p *PopupMenu) invalidateRow(i int) {
	r := p.rowBounds(i)
	if !r.Empty() {
		p.InvalidateRect(r.Inset(-1))
	}
}

func (p *PopupMenu) MouseMove(e widget.MouseEvent) bool {
	track, thumb := p.scrollTrack()
	if off, apply, handled, hoverDirty := p.vbar.move(e.Pos, track, thumb, true, p.MaxOffset()); handled {
		if apply {
			p.OffsetY = off
			p.clamp()
		}
		if apply || hoverDirty {
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
	return true
}

func (p *PopupMenu) MouseExit() {
	p.hover = -1
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
	track, thumb := p.scrollTrack()
	if off, ok := p.vbar.press(e.Pos, track, thumb, true, p.OffsetY, p.MaxOffset(), p.LocalBounds().Dy()*0.9); ok {
		p.OffsetY = off
		p.clamp()
		p.Invalidate()
		return true
	}
	p.press = p.rowAt(e.Pos.Y)
	p.Invalidate()
	return true
}

func (p *PopupMenu) MouseRelease(e widget.MouseEvent) bool {
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
	if p.MaxOffset() <= 0 {
		return false
	}
	p.OffsetY += wheelDelta(e.Scroll.Y, p.itemH())
	p.clamp()
	p.Invalidate()
	return true
}

func (p *PopupMenu) KeyPress(e widget.KeyEvent) bool {
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
	case platform.KeyReturn, platform.KeySpace:
		p.activate(p.focus)
		return true
	case platform.KeyEscape:
		widget.DismissPopup(p)
		return true
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

func (p *PopupMenu) activate(i int) {
	if i < 0 || i >= len(p.Items) {
		return
	}
	it := p.Items[i]
	if it == nil || it.Separator || it.Disabled {
		return
	}
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
	if p.OnDismiss != nil {
		p.OnDismiss()
	}
}

// ShowContextMenu opens a popup at window-space origin.
func ShowContextMenu(from widget.Component, origin paintengine2d.Point, items ...*MenuItem) *PopupMenu {
	pop := NewPopupMenu(items...)
	widget.PreparePopup(from, pop)
	widget.PlacePopup(pop, origin, 360, 480)
	if widget.ShowPopup(from, pop) {
		widget.ClampToSurface(from, pop)
		pop.RequestFocus()
		return pop
	}
	return nil
}
