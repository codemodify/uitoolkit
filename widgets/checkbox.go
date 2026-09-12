package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Checkbox is a boolean toggle with a label.
type Checkbox struct {
	widget.Base
	Text     string
	Checked  bool
	OnChange func(bool)
	hovered  bool
	pressed  bool
}

func NewCheckbox(text string, checked bool, on func(bool)) *Checkbox {
	c := &Checkbox{Text: text, Checked: checked, OnChange: on}
	c.Init(c)
	c.SetWantsFocus(true)
	c.SetFocusVisibleOnly(true)
	return c
}

func (c *Checkbox) SetChecked(v bool) {
	if c.Checked == v {
		return
	}
	c.Checked = v
	c.Invalidate()
	if c.OnChange != nil {
		c.OnChange(v)
	}
}

func (c *Checkbox) Measure(cons layout.Constraints) paintengine2d.Point {
	lk := c.Look()
	m := lk.Metrics()
	w := m.Checkbox + 8 + lk.Font().Advance(c.Text) + 4
	return cons.Constrain(paintengine2d.Pt(w, m.ControlH))
}

func (c *Checkbox) Arrange(r paintengine2d.Rect) { c.SetBounds(r) }

func (c *Checkbox) Paint(ctx *paintengine2d.Context) {
	st := c.State()
	if c.hovered {
		st |= style.StateHovered
	}
	c.Look().DrawCheckbox(ctx, c.LocalBounds(), st, c.Checked, c.Text)
}

func (c *Checkbox) MouseEnter() { c.hovered = true; c.Base.MouseEnter() }
func (c *Checkbox) MouseExit() {
	c.hovered = false
	c.pressed = false
	c.Base.MouseExit()
}

func (c *Checkbox) MousePress(widget.MouseEvent) bool {
	if !c.Enabled() {
		return false
	}
	c.MarkPointerFocus()
	c.RequestFocus()
	c.pressed = true
	c.Invalidate()
	return true
}

func (c *Checkbox) MouseRelease(e widget.MouseEvent) bool {
	was := c.pressed
	c.pressed = false
	c.Invalidate()
	if was && c.Enabled() && c.LocalBounds().Contains(e.Pos) {
		c.SetChecked(!c.Checked)
	}
	return true
}

func (c *Checkbox) KeyPress(e widget.KeyEvent) bool {
	if !c.Enabled() {
		return false
	}
	c.MarkKeyboardFocus()
	if e.Key == platform.KeySpace || e.Key == platform.KeyReturn {
		c.SetChecked(!c.Checked)
		return true
	}
	return false
}
