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
	menus []*Menu
	open  int
	hover int
	press int
	focus int
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

func (m *MenuBar) Measure(c layout.Constraints) paintengine2d.Point {
	w := float32(200)
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
	rects := m.titleRects()
	for i, menu := range m.menus {
		st := m.State()
		if i == m.hover {
			st |= style.StateHovered
		}
		if i == m.press {
			st |= style.StatePressed
		}
		if i != m.focus {
			st &^= style.StateFocused
		}
		label, _, idx := ParseMnemonic(menu.Title)
		lk.DrawMenuTitle(ctx, rects[i], st, label, idx, i == m.open)
	}
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

func (m *MenuBar) MousePress(e widget.MouseEvent) bool {
	if !m.Enabled() || e.Button == platform.ButtonRight {
		return false
	}
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
		widget.DismissPopup(m)
		if it != nil && it.OnClick != nil {
			it.OnClick()
		}
	}
	pop.OnDismiss = func() {
		if m.open == i {
			m.open = -1
			m.Invalidate()
		}
	}
	origin := widget.DeviceOrigin(m)
	tb := rects[i]
	widget.PlacePopup(pop, paintengine2d.Pt(origin.X+tb.Min.X, origin.Y+tb.Max.Y-1), 360, 480)
	if widget.ShowPopup(m, pop) {
		widget.ClampToSurface(m, pop)
		m.open = i
		m.Invalidate()
		pop.RequestFocus()
	}
}

// Close dismisses the open menu.
func (m *MenuBar) Close() {
	m.open = -1
	widget.DismissPopup(m)
	m.Invalidate()
}

// PopupMenu is a floating list of MenuItems (drop-down or context menu).
type PopupMenu struct {
	widget.Base
	Items     []*MenuItem
	OnPick    func(*MenuItem)
	OnDismiss func()
	hover     int
	press     int
	focus     int
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
	h := p.Look().Metrics().MenuItemH
	if h <= 0 {
		h = 26
	}
	return h
}

func (p *PopupMenu) rowH(it *MenuItem) float32 {
	if it != nil && it.Separator {
		return 8
	}
	return p.itemH()
}

func (p *PopupMenu) Measure(c layout.Constraints) paintengine2d.Point {
	f := p.Look().Font()
	var w, h float32
	w = 160
	for _, it := range p.Items {
		if it == nil || it.Separator {
			h += p.rowH(it)
			continue
		}
		label, _, _ := ParseMnemonic(it.Text)
		tw := f.Advance(label) + 36
		if it.Shortcut != "" {
			tw += f.Advance(it.Shortcut) + 24
		}
		if tw > w {
			w = tw
		}
		h += p.rowH(it)
	}
	h += 8
	w += 8
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (p *PopupMenu) Arrange(r paintengine2d.Rect) { p.SetBounds(r) }

func (p *PopupMenu) rowAt(y float32) int {
	cur := float32(4)
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
	cur := float32(4)
	b := p.LocalBounds()
	for n, it := range p.Items {
		rh := p.rowH(it)
		if n == i {
			return paintengine2d.XYWH(4, cur, b.Dx()-8, rh)
		}
		cur += rh
	}
	return paintengine2d.Rect{}
}

func (p *PopupMenu) Paint(ctx *paintengine2d.Context) {
	lk := p.Look()
	lk.DrawMenuFrame(ctx, p.LocalBounds())
	for i, it := range p.Items {
		if it == nil {
			continue
		}
		st := style.StateNone
		if it.Disabled {
			st |= style.StateDisabled
		}
		if i == p.hover || i == p.focus {
			st |= style.StateHovered
		}
		if i == p.press {
			st |= style.StatePressed
		}
		label, _, idx := ParseMnemonic(it.Text)
		lk.DrawMenuItem(ctx, p.rowBounds(i), st, label, it.Shortcut, idx, it.Separator, it.Checked)
	}
}

func (p *PopupMenu) invalidateRow(i int) {
	r := p.rowBounds(i)
	if !r.Empty() {
		p.InvalidateRect(r.Inset(-1))
	}
}

func (p *PopupMenu) MouseMove(e widget.MouseEvent) bool {
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

func (p *PopupMenu) MousePress(e widget.MouseEvent) bool {
	p.press = p.rowAt(e.Pos.Y)
	p.Invalidate()
	return true
}

func (p *PopupMenu) MouseRelease(e widget.MouseEvent) bool {
	i := p.rowAt(e.Pos.Y)
	p.press = -1
	p.Invalidate()
	if i >= 0 && i < len(p.Items) {
		p.activate(i)
	}
	return true
}

func (p *PopupMenu) KeyPress(e widget.KeyEvent) bool {
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
	widget.PlacePopup(pop, origin, 360, 480)
	if widget.ShowPopup(from, pop) {
		widget.ClampToSurface(from, pop)
		pop.RequestFocus()
		return pop
	}
	return nil
}
