package widgets

import (
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ComboBox is a closed field that drops a list of choices. SetEditable
// lets the user type a value of their own.
type ComboBox struct {
	widget.Base
	Items       []string
	Selected    int
	Placeholder string
	OnChange    func(int)
	// OnEdit reports the text of an editable combo box as it is typed;
	// OnSubmit reports it when Return is pressed.
	OnEdit   func(text string)
	OnSubmit func(text string)
	// NoCompletion stops an editable combo box completing typed text
	// inline from Items (QComboBox completes by default).
	NoCompletion bool
	// Tip is the hover help, for a box whose own label does not say what
	// it chooses — one on a tool bar, where there is no room for a label
	// beside it.
	Tip string
	// MinWidth is the narrowest the box measures itself, in 1x pixels.
	// It is 160 by default, which is a form's field: a box in a column of
	// them should not be narrower than its neighbours whatever it lists.
	// A box on a tool bar sets it small — there it is one item among
	// many, and 160 pixels of empty field pushes the tools off the end.
	MinWidth   float32
	open       bool
	hovered    bool
	fade       stateFade // hover / focus cross-fade (the look's HintHoverFadeMs)
	field      *TextField
	typed      string // the text as typed, without the completion
	completing bool
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

// Tooltip is the hover help text.
func (c *ComboBox) Tooltip() string { return c.Tip }

// Text is the selected label (an editable box's typed text), or empty.
func (c *ComboBox) Text() string {
	if c.field != nil {
		return c.field.Text
	}
	if c.Selected >= 0 && c.Selected < len(c.Items) {
		return c.Items[c.Selected]
	}
	return ""
}

// Editable reports whether the text is typed into.
func (c *ComboBox) Editable() bool { return c.field != nil }

// SetEditable makes the combo box's text editable (QComboBox's
// setEditable, GTK's combo box with an entry): a field inside takes
// typing, the arrow and Down drop the list, and typed text completes
// inline from Items unless NoCompletion is set.
func (c *ComboBox) SetEditable(on bool) {
	if on == (c.field != nil) {
		return
	}
	if !on {
		c.Remove(c.field)
		c.field = nil
		c.SetWantsFocus(true)
		c.RequestLayout()
		return
	}
	f := NewTextField(c.Text(), c.Placeholder, nil)
	f.Frameless = true
	f.OnChange = c.edited
	f.OnSubmit = func(s string) {
		if c.OnSubmit != nil {
			c.OnSubmit(s)
		}
	}
	c.field = f
	c.typed = f.Text
	c.Add(f)
	// The field takes the keyboard focus for the box.
	c.SetWantsFocus(false)
	c.RequestLayout()
}

// SetText sets an editable combo box's text (and the matching choice).
func (c *ComboBox) SetText(s string) {
	if c.field == nil {
		return
	}
	c.completing = true
	c.field.SetText(s)
	c.completing = false
	c.typed = s
	c.Selected = c.indexOf(s)
	c.Invalidate()
}

func (c *ComboBox) indexOf(s string) int {
	for i, it := range c.Items {
		if it == s {
			return i
		}
	}
	return -1
}

// edited follows the field: the matching choice, and inline completion
// when the text grew at its end.
func (c *ComboBox) edited(s string) {
	if c.completing {
		return
	}
	grew := len(s) > len(c.typed) && strings.HasPrefix(s, c.typed)
	c.typed = s
	f := c.field
	if grew && !c.NoCompletion && s != "" && f.caret == runeCount(s) {
		low := strings.ToLower(s)
		for _, it := range c.Items {
			if len(it) > len(s) && strings.HasPrefix(strings.ToLower(it), low) {
				// Keep what was typed; select the completed rest, so the
				// next key replaces it.
				full := s + string([]rune(it)[runeCount(s):])
				c.completing = true
				f.SetText(full)
				c.completing = false
				f.selA, f.selB, f.caret = runeCount(full), runeCount(s), runeCount(s)
				f.Invalidate()
				s = full
				break
			}
		}
	}
	c.Selected = c.indexOf(s)
	c.Invalidate()
	if c.OnEdit != nil {
		c.OnEdit(s)
	}
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
	if c.field != nil && i >= 0 {
		c.SetText(c.Items[i])
		c.field.selA, c.field.selB, c.field.caret = 0, runeCount(c.field.Text), runeCount(c.field.Text)
	}
	c.Invalidate()
	c.Close()
	if c.OnChange != nil {
		c.OnChange(i)
	}
}

func (c *ComboBox) Measure(cons layout.Constraints) paintengine2d.Point {
	lk := c.Look()
	m := lk.Metrics()
	floor := float32(160)
	if c.MinWidth > 0 {
		floor = c.MinWidth
	}
	w := style.Dip(lk, floor)
	f := style.ControlFontOf(lk, style.RoleCombo)
	for _, s := range c.Items {
		tw := f.Advance(s)
		if ink := f.InkWidth(s); ink > tw {
			tw = ink
		}
		tw += style.Dip(lk, 44)
		if tw > w {
			w = tw
		}
	}
	if c.Placeholder != "" {
		tw := f.Advance(c.Placeholder)
		if ink := f.InkWidth(c.Placeholder); ink > tw {
			tw = ink
		}
		tw += style.Dip(lk, 44)
		if tw > w {
			w = tw
		}
	}
	return cons.Constrain(paintengine2d.Pt(w, style.ComboHeight(m)))
}

func (c *ComboBox) Arrange(r paintengine2d.Rect) {
	c.SetBounds(r)
	if c.field != nil {
		c.field.Arrange(style.ComboTextRectOf(c.Look(), c.LocalBounds()))
	}
}

func (c *ComboBox) Paint(ctx *paintengine2d.Context) {
	st := c.State()
	if c.hovered {
		st |= style.StateHovered
	}
	show := c.Text()
	if show == "" {
		show = c.Placeholder
	}
	if c.field != nil {
		// The face only: the field inside draws the text.
		st |= style.StateEditable
		if c.field.Focused() {
			st |= style.StateFocused
		}
		show = ""
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
	if c.field != nil {
		c.field.RequestFocus()
	} else {
		c.RequestFocus()
	}
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
		if c.field != nil && !c.open {
			return false
		}
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
		if c.field != nil {
			return false
		}
		if len(c.Items) > 0 {
			c.Select(0)
		}
		return true
	case platform.KeyEnd:
		if c.field != nil {
			return false
		}
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
	if c.field != nil {
		pop.RestoreFocusTo(c.field)
	} else {
		pop.RestoreFocusTo(c)
	}
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
