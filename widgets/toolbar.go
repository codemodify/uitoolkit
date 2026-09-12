package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ToolItem is one control on a ToolBar (text, icon, or both).
type ToolItem struct {
	Text     string
	Icon     style.ToolIcon
	Disabled bool
	Toggle   bool
	Down     bool
	Sep      bool
	Tip      string
	OnClick  func()
}

// ToolText is a labeled tool button.
func ToolText(text string, on func()) *ToolItem {
	return &ToolItem{Text: text, OnClick: on}
}

// ToolIconBtn is an icon tool button. Text is optional (shown beside the icon).
func ToolIconBtn(icon style.ToolIcon, text string, on func()) *ToolItem {
	return &ToolItem{Icon: icon, Text: text, OnClick: on}
}

// ToolToggle is a sticky tool button.
func ToolToggle(text string, down bool, on func()) *ToolItem {
	return &ToolItem{Text: text, Toggle: true, Down: down, OnClick: on}
}

// ToolDivider is a vertical rule between tool groups.
func ToolDivider() *ToolItem { return &ToolItem{Sep: true} }

// ToolBar is a horizontal strip of tool buttons.
type ToolBar struct {
	widget.Base
	items  []*ToolItem
	hover  int
	press  int
	focus  int
	keyNav bool
}

// NewToolBar constructs a toolbar.
func NewToolBar(items ...*ToolItem) *ToolBar {
	t := &ToolBar{items: items, hover: -1, press: -1, focus: firstTool(items)}
	t.Init(t)
	t.SetWantsFocus(true)
	return t
}

// Items returns the tool buttons.
func (t *ToolBar) Items() []*ToolItem { return t.items }

// Tooltip is the hovered tool button's tip, if any.
func (t *ToolBar) Tooltip() string {
	if t.hover >= 0 && t.hover < len(t.items) && t.items[t.hover] != nil {
		return t.items[t.hover].Tip
	}
	return ""
}

// ItemRect is the local box of item i.
func (t *ToolBar) ItemRect(i int) paintengine2d.Rect {
	rects := t.itemRects()
	if i < 0 || i >= len(rects) {
		return paintengine2d.Rect{}
	}
	return rects[i]
}

// KeyboardChrome is true when a tool is painting a keyboard focus ring.
func (t *ToolBar) KeyboardChrome() bool {
	return t.keyNav && t.focus >= 0
}

// ItemCenter is the local midpoint of item i.
func (t *ToolBar) ItemCenter(i int) paintengine2d.Point {
	rects := t.itemRects()
	if i < 0 || i >= len(rects) {
		return paintengine2d.Point{}
	}
	r := rects[i]
	return paintengine2d.Pt((r.Min.X+r.Max.X)*0.5, (r.Min.Y+r.Max.Y)*0.5)
}

// Hover highlights item i (screenshots / tests). i < 0 clears.
func (t *ToolBar) invalidateItem(i int) {
	rects := t.itemRects()
	if i < 0 || i >= len(rects) {
		return
	}
	t.InvalidateRect(rects[i].Inset(-1))
}

func (t *ToolBar) Hover(i int) {
	if t.hover == i {
		return
	}
	old := t.hover
	t.hover = i
	t.invalidateItem(old)
	t.invalidateItem(i)
}

func firstTool(items []*ToolItem) int {
	for i, it := range items {
		if it != nil && !it.Sep && !it.Disabled {
			return i
		}
	}
	return 0
}

func (t *ToolBar) barH() float32 {
	h := t.Look().Metrics().ToolBarH
	if h <= 0 {
		h = 36
	}
	return h
}

func (t *ToolBar) toolBtnW(h float32) float32 {
	btn := t.Look().Metrics().ToolBtn
	if btn <= 0 {
		btn = h - 6
		if btn < 8 {
			btn = 8
		}
	}
	return btn
}

func (t *ToolBar) itemBoxW(it *ToolItem, btn float32) float32 {
	if it == nil || it.Sep {
		return 8
	}
	pad, iconSide, iconGap := style.ToolButtonChromeFor(t.Look(), btn)
	if it.Text == "" {
		if it.Icon != style.IconNone {
			w := pad*2 + iconSide
			if w < btn {
				return btn
			}
			return w
		}
		return btn
	}
	textW := t.Look().Font().InkWidth(it.Text)
	if adv := t.Look().Font().Advance(it.Text); adv > textW {
		textW = adv
	}
	w := pad*2 + textW
	if it.Icon != style.IconNone {
		w += iconSide + iconGap
	}
	return w
}

// contentW is the intrinsic strip width (items + padding). ToolBar must
// not expand to MaxW: a Row/Flex parent decides growth via Flex weights.
func (t *ToolBar) contentW() float32 {
	btn := t.toolBtnW(t.barH())
	x := float32(6)
	for _, it := range t.items {
		if it == nil || it.Sep {
			x += 8 + style.ToolItemGap
			continue
		}
		x += t.itemBoxW(it, btn) + style.ToolItemGap
	}
	return x + 6
}

func (t *ToolBar) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(t.contentW(), t.barH()))
}

func (t *ToolBar) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

func (t *ToolBar) itemRects() []paintengine2d.Rect {
	h := t.LocalBounds().Dy()
	btn := t.toolBtnW(h)
	x := float32(6)
	y := (h - btn) * 0.5
	if y < 2 {
		y = 2
		btn = h - 4
	}
	out := make([]paintengine2d.Rect, len(t.items))
	for i, it := range t.items {
		if it == nil || it.Sep {
			out[i] = paintengine2d.XYWH(x, 6, 8, h-12)
			x += 8 + style.ToolItemGap
			continue
		}
		w := t.itemBoxW(it, btn)
		out[i] = paintengine2d.XYWH(x, y, w, btn)
		x += w + style.ToolItemGap
	}
	return out
}

func (t *ToolBar) itemAt(p paintengine2d.Point) int {
	for i, r := range t.itemRects() {
		if r.Contains(p) {
			return i
		}
	}
	return -1
}

func (t *ToolBar) Paint(ctx *paintengine2d.Context) {
	lk := t.Look()
	lk.DrawToolBar(ctx, t.LocalBounds())
	rects := t.itemRects()
	for i, it := range t.items {
		if it == nil {
			continue
		}
		if it.Sep {
			b := rects[i]
			x := (b.Min.X + b.Max.X) * 0.5
			ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, 1, b.Dy()), paintengine2d.Fill(lk.Palette().Divider))
			continue
		}
		st := t.State()
		if i != t.focus || !t.keyNav {
			st &^= style.StateFocused
		}
		if i == t.hover {
			st |= style.StateHovered
		}
		if i == t.press {
			st |= style.StatePressed
		}
		if it.Toggle {
			st |= style.StateToggle
		}
		if it.Down {
			st |= style.StateChecked
		}
		if it.Disabled {
			st |= style.StateDisabled
		}
		lk.DrawToolButton(ctx, rects[i], st, it.Text, it.Icon)
	}
}

func (t *ToolBar) MouseMove(e widget.MouseEvent) bool {
	i := t.itemAt(e.Pos)
	if i != t.hover {
		old := t.hover
		t.hover = i
		t.invalidateItem(old)
		t.invalidateItem(i)
	}
	return true
}

func (t *ToolBar) MouseExit() {
	t.hover = -1
	t.press = -1
	t.Base.MouseExit()
}

func (t *ToolBar) FocusLost() {
	t.keyNav = false
	t.focus = -1
	t.Base.FocusLost()
}

func (t *ToolBar) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() || e.Button == platform.ButtonRight {
		return false
	}
	t.keyNav = false
	t.RequestFocus()
	i := t.itemAt(e.Pos)
	t.press = i
	if i >= 0 {
		t.focus = i
	}
	t.Invalidate()
	return true
}

func (t *ToolBar) MouseRelease(e widget.MouseEvent) bool {
	i := t.itemAt(e.Pos)
	was := t.press
	t.press = -1
	t.Invalidate()
	if was >= 0 && was == i {
		t.activate(i)
	}
	return true
}

func (t *ToolBar) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() || len(t.items) == 0 {
		return false
	}
	t.keyNav = true
	if t.focus < 0 {
		t.focus = firstTool(t.items)
	}
	switch e.Key {
	case platform.KeyLeft:
		t.moveFocus(-1)
		return true
	case platform.KeyRight:
		t.moveFocus(1)
		return true
	case platform.KeyHome:
		t.focus = firstTool(t.items)
		t.Invalidate()
		return true
	case platform.KeyEnd:
		t.focus = lastTool(t.items)
		t.Invalidate()
		return true
	case platform.KeyReturn, platform.KeySpace:
		t.activate(t.focus)
		return true
	}
	return false
}

func lastTool(items []*ToolItem) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i] != nil && !items[i].Sep && !items[i].Disabled {
			return i
		}
	}
	return 0
}

func (t *ToolBar) moveFocus(dir int) {
	if len(t.items) == 0 {
		return
	}
	i := t.focus
	for n := 0; n < len(t.items); n++ {
		i = (i + dir + len(t.items)) % len(t.items)
		it := t.items[i]
		if it != nil && !it.Sep && !it.Disabled {
			t.focus = i
			t.Invalidate()
			return
		}
	}
}

func (t *ToolBar) activate(i int) {
	if i < 0 || i >= len(t.items) {
		return
	}
	it := t.items[i]
	if it == nil || it.Sep || it.Disabled {
		return
	}
	if it.Toggle {
		it.Down = !it.Down
		t.Invalidate()
	}
	if it.OnClick != nil {
		it.OnClick()
	}
}
