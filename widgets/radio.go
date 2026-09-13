package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// RadioButton is one exclusive choice. Use RadioGroup for a set.
type RadioButton struct {
	widget.Base
	Text     string
	Selected bool
	OnChange func(bool)
	hovered  bool
	pressed  bool
	group    *RadioGroup
	index    int
}

// NewRadio builds a standalone radio (selects on press; does not uncheck itself).
func NewRadio(text string, selected bool, on func(bool)) *RadioButton {
	r := &RadioButton{Text: text, Selected: selected, OnChange: on, index: -1}
	r.Init(r)
	r.SetWantsFocus(true)
	r.SetFocusVisibleOnly(true)
	return r
}

func (r *RadioButton) SetSelected(v bool) {
	if r.Selected == v {
		return
	}
	r.Selected = v
	r.Invalidate()
	if r.OnChange != nil {
		r.OnChange(v)
	}
}

func (r *RadioButton) Measure(c layout.Constraints) paintengine2d.Point {
	lk := r.Look()
	m := lk.Metrics()
	side := m.Radio
	if side <= 0 {
		side = m.Checkbox
	}
	w := side + 8 + lk.Font().Advance(r.Text) + 4
	return c.Constrain(paintengine2d.Pt(w, m.ControlH))
}

func (r *RadioButton) Arrange(b paintengine2d.Rect) { r.SetBounds(b) }

func (r *RadioButton) Paint(ctx *paintengine2d.Context) {
	st := r.State()
	if r.hovered {
		st |= style.StateHovered
	}
	r.Look().DrawRadio(ctx, r.LocalBounds(), st, r.Selected, r.Text)
}

func (r *RadioButton) MouseEnter() { r.hovered = true; r.Base.MouseEnter() }
func (r *RadioButton) MouseExit() {
	r.hovered = false
	r.pressed = false
	r.Base.MouseExit()
}

func (r *RadioButton) MousePress(widget.MouseEvent) bool {
	if !r.Enabled() {
		return false
	}
	r.MarkPointerFocus()
	r.RequestFocus()
	r.pressed = true
	r.Invalidate()
	return true
}

func (r *RadioButton) MouseRelease(e widget.MouseEvent) bool {
	was := r.pressed
	r.pressed = false
	r.Invalidate()
	if was && r.Enabled() && r.LocalBounds().Contains(e.Pos) {
		r.choose()
	}
	return true
}

func (r *RadioButton) KeyPress(e widget.KeyEvent) bool {
	if !r.Enabled() {
		return false
	}
	r.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeySpace, platform.KeyReturn:
		r.choose()
		return true
	case platform.KeyUp, platform.KeyLeft:
		if r.group != nil {
			r.group.move(-1)
			return true
		}
	case platform.KeyDown, platform.KeyRight:
		if r.group != nil {
			r.group.move(1)
			return true
		}
	}
	return false
}

func (r *RadioButton) choose() {
	if r.group != nil {
		r.group.Select(r.index)
		return
	}
	r.SetSelected(true)
}

// RadioGroup is a column of exclusive RadioButtons.
type RadioGroup struct {
	widget.Base
	selected int
	OnChange func(int)
	buttons  []*RadioButton
	col      *FlexBox
}

// NewRadioGroup builds radios from labels. selected < 0 means none.
func NewRadioGroup(labels []string, selected int, on func(int)) *RadioGroup {
	g := &RadioGroup{selected: selected, OnChange: on}
	g.Init(g)
	g.col = NewColumn().WithGap(4)
	g.Base.Add(g.col)
	for i, label := range labels {
		rb := NewRadio(label, i == selected, nil)
		rb.group = g
		rb.index = i
		g.buttons = append(g.buttons, rb)
		g.col.Add(rb)
	}
	if selected >= 0 && selected < len(g.buttons) {
		g.selected = selected
	} else {
		g.selected = -1
	}
	return g
}

// Buttons returns the radios in order.
func (g *RadioGroup) Buttons() []*RadioButton { return g.buttons }

// Selected is the current index, or -1.
func (g *RadioGroup) Selected() int { return g.selected }

// Select picks index i.
func (g *RadioGroup) Select(i int) {
	if i < 0 || i >= len(g.buttons) {
		return
	}
	if g.selected == i && g.buttons[i].Selected {
		return
	}
	g.selected = i
	for n, rb := range g.buttons {
		rb.Selected = n == i
		rb.Invalidate()
	}
	g.Invalidate()
	if g.OnChange != nil {
		g.OnChange(i)
	}
}

func (g *RadioGroup) move(dir int) {
	if len(g.buttons) == 0 {
		return
	}
	i := g.selected
	if i < 0 {
		i = 0
	} else {
		i += dir
	}
	if i < 0 {
		i = 0
	}
	if i >= len(g.buttons) {
		i = len(g.buttons) - 1
	}
	g.Select(i)
	g.buttons[i].RequestFocus()
	// Arrow keys moved focus, so the new radio must paint its focus ring
	// (radios are focus-visible-only).
	widget.MarkKeyboardFocus(g.buttons[i])
}

func (g *RadioGroup) Measure(c layout.Constraints) paintengine2d.Point {
	return g.col.Measure(c)
}

func (g *RadioGroup) Arrange(r paintengine2d.Rect) {
	g.SetBounds(r)
	g.col.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
}
