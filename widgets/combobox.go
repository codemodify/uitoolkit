package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ComboBox is a closed field that drops a list of choices.
type ComboBox struct {
	widget.Base
	Items       []string
	Selected    int
	Placeholder string
	OnChange    func(int)
	open        bool
	hovered     bool
	fade        stateFade // hover / focus cross-fade (the look's HintHoverFadeMs)
}

// NewComboBox builds a drop-down. selected < 0 means none.
func NewComboBox(items []string, selected int, on func(int)) *ComboBox {
	if selected >= len(items) {
		selected = -1
	}
	c := &ComboBox{Items: items, Selected: selected, OnChange: on}
	c.Init(c)
	c.SetWantsFocus(true)
	c.SetFocusVisibleOnly(true)
	return c
}

func (c *ComboBox) RetainsPointer() bool { return true }

// Text is the selected label, or empty.
func (c *ComboBox) Text() string {
	if c.Selected >= 0 && c.Selected < len(c.Items) {
		return c.Items[c.Selected]
	}
	return ""
}

// Select sets the current choice.
func (c *ComboBox) Select(i int) {
	if i < -1 || i >= len(c.Items) {
		return
	}
	if c.Selected == i {
		c.Close()
		return
	}
	c.Selected = i
	c.Invalidate()
	c.Close()
	if c.OnChange != nil {
		c.OnChange(i)
	}
}

func (c *ComboBox) Measure(cons layout.Constraints) paintengine2d.Point {
	lk := c.Look()
	m := lk.Metrics()
	w := float32(160)
	f := style.ControlFontOf(lk, style.RoleCombo)
	for _, s := range c.Items {
		tw := f.Advance(s)
		if ink := f.InkWidth(s); ink > tw {
			tw = ink
		}
		tw += 44
		if tw > w {
			w = tw
		}
	}
	if c.Placeholder != "" {
		tw := f.Advance(c.Placeholder)
		if ink := f.InkWidth(c.Placeholder); ink > tw {
			tw = ink
		}
		tw += 44
		if tw > w {
			w = tw
		}
	}
	return cons.Constrain(paintengine2d.Pt(w, style.ComboHeight(m)))
}

func (c *ComboBox) Arrange(r paintengine2d.Rect) { c.SetBounds(r) }

func (c *ComboBox) Paint(ctx *paintengine2d.Context) {
	st := c.State()
	if c.hovered {
		st |= style.StateHovered
	}
	show := c.Text()
	if show == "" {
		show = c.Placeholder
	}
	lk, r := c.Look(), c.LocalBounds()
	c.fade.paint(c, ctx, r, st, func(ctx *paintengine2d.Context, st style.ControlState) { lk.DrawComboBox(ctx, r, st, show, c.open) })
}

func (c *ComboBox) MouseEnter() { c.hovered = true; c.Base.MouseEnter() }
func (c *ComboBox) MouseExit()  { c.hovered = false; c.Base.MouseExit() }

func (c *ComboBox) MousePress(e widget.MouseEvent) bool {
	if !c.Enabled() || e.Button == platform.ButtonRight {
		return false
	}
	c.MarkPointerFocus()
	c.RequestFocus()
	if c.open {
		c.Close()
	} else {
		c.Open()
	}
	return true
}

func (c *ComboBox) KeyPress(e widget.KeyEvent) bool {
	if !c.Enabled() {
		return false
	}
	c.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeyDown, platform.KeySpace:
		if !c.open {
			c.Open()
		}
		return true
	case platform.KeyUp:
		if c.open {
			return true
		}
		if c.Selected > 0 {
			c.Select(c.Selected - 1)
		}
		return true
	case platform.KeyReturn:
		if c.open {
			c.Close()
		} else {
			c.Open()
		}
		return true
	case platform.KeyEscape:
		if c.open {
			c.Close()
			return true
		}
	case platform.KeyHome:
		if len(c.Items) > 0 {
			c.Select(0)
		}
		return true
	case platform.KeyEnd:
		if len(c.Items) > 0 {
			c.Select(len(c.Items) - 1)
		}
		return true
	}
	return false
}

// Open drops the choice list.
func (c *ComboBox) Open() {
	if c.open || !c.Enabled() || len(c.Items) == 0 {
		return
	}
	items := make([]*MenuItem, len(c.Items))
	for i, s := range c.Items {
		i, s := i, s
		items[i] = Item(s, func() { c.Select(i) })
		items[i].Checked = i == c.Selected
	}
	pop := NewPopupMenu(items...)
	if c.Selected >= 0 {
		pop.focus = c.Selected
		pop.hover = c.Selected
	}
	pop.OnPick = func(it *MenuItem) {
		c.open = false
		widget.DismissPopup(c)
		if it != nil && it.OnClick != nil {
			it.OnClick()
		}
	}
	pop.OnDismiss = func() {
		if c.open {
			c.open = false
			c.Invalidate()
		}
	}
	pop.RestoreFocusTo(c)
	o := widget.DeviceOrigin(c)
	b := c.LocalBounds()
	anchor := paintengine2d.XYWH(o.X, o.Y, b.Dx(), b.Dy())
	widget.PlacePopupForAnchor(c, pop, anchor, b.Dx(), 2)
	if widget.ShowPopup(c, pop) {
		c.open = true
		c.Invalidate()
		pop.RequestFocus()
	}
}

// Close dismisses the drop-down.
func (c *ComboBox) Close() {
	if !c.open {
		return
	}
	c.open = false
	widget.DismissPopup(c)
	c.Invalidate()
}

// Opened reports whether the list is showing.
func (c *ComboBox) Opened() bool { return c.open }
