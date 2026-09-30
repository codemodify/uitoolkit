package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// MenuButton is a push button that drops a menu: an application menu
// behind a hamburger, an overflow behind ⋯, a split action's arrow.
//
// It is [IconButton] plus the things a menu *button* does that a button
// which happens to call ShowContextMenu does not:
//
//   - the menu opens on **press**, not on release, so it is there under
//     the pointer that opened it and a drag can run straight into it;
//   - the button stays **down** while its menu is open, which is what
//     says where the menu came from;
//   - pressing it again **closes** the menu rather than reopening it
//     under itself;
//   - its items' **accelerators work**, as a menu bar's do. This is the
//     one that bites: an application menu moved from a MenuBar to a
//     Button loses Ctrl+Q, and nothing says so — the shortcut simply
//     stops working.
//
// The menu is a plain slice of [MenuItem], the same as a menu bar's or a
// context menu's, so the three share their items and their submenus.
type MenuButton struct {
	IconButton
	// Items are the rows. They may be changed between openings; the menu
	// is built each time it is shown.
	Items []*MenuItem
	// Build is asked for the items instead, when the menu depends on
	// what is true at the moment it opens — a "Reopen" list, a set of
	// check marks. Items is used when Build is nil.
	Build func() []*MenuItem

	pop *PopupMenu
}

// NewMenuButton builds a menu button showing icon, named by name.
func NewMenuButton(icon style.ToolIcon, name string, items ...*MenuItem) *MenuButton {
	b := &MenuButton{Items: items}
	b.Icon = icon
	b.Action = name
	b.Tip = name
	b.Init(b)
	b.SetWantsFocus(true)
	b.SetFocusVisibleOnly(true)
	b.SetAccessibleName(name)
	if icon != style.IconNone {
		b.Content = b.paintIcon
	}
	b.Toggle = true
	return b
}

// NewTextMenuButton is a menu button that shows a word rather than a
// mark — the File, Edit and View of a menu bar rendered as buttons,
// which is what an application that draws its own title bar has instead
// of a [MenuBar].
func NewTextMenuButton(text string, items ...*MenuItem) *MenuButton {
	b := NewMenuButton(style.IconNone, text, items...)
	b.Text = text
	b.Tip = ""
	return b
}

// Measure is a square while the button is a mark alone, and a button's
// own width once it has a name.
//
// It embeds [IconButton], which is square by definition — but a menu
// button is as often "File" as it is a hamburger, and a named one
// squeezed into a square shows "F…".
func (b *MenuButton) Measure(c layout.Constraints) paintengine2d.Point {
	if b.Text == "" {
		return b.IconButton.Measure(c)
	}
	return b.Button.Measure(c)
}

// items are the rows this opening should show.
func (b *MenuButton) items() []*MenuItem {
	if b.Build != nil {
		return b.Build()
	}
	return b.Items
}

// Open shows the menu under the button, and reports whether it opened.
func (b *MenuButton) Open() bool {
	if b.pop != nil {
		return true
	}
	items := b.items()
	if len(items) == 0 {
		return false
	}
	// Under the button's bottom-left, which is where every desktop puts
	// the menu of a button rather than at the pointer: the menu belongs
	// to the control, not to the click.
	o := widget.DeviceOrigin(b)
	at := paintengine2d.Pt(o.X, o.Y+b.Bounds().Dy())
	pop := ShowContextMenu(b, at, items...)
	if pop == nil {
		return false
	}
	b.pop = pop
	b.Checked = true
	prev := pop.OnDismiss
	pop.OnDismiss = func() {
		b.pop = nil
		b.Checked = false
		b.Invalidate()
		if prev != nil {
			prev()
		}
	}
	b.Invalidate()
	return true
}

// Close takes the menu down if it is up.
func (b *MenuButton) Close() {
	if b.pop == nil {
		return
	}
	widget.DismissPopup(b)
}

// IsOpen reports whether the menu is showing.
func (b *MenuButton) IsOpen() bool { return b.pop != nil }

// MousePress opens the menu rather than waiting for the release, and a
// second press closes it instead of reopening it underneath itself.
func (b *MenuButton) MousePress(e widget.MouseEvent) bool {
	if !b.Enabled() {
		return false
	}
	b.MarkPointerFocus()
	if b.pop != nil {
		b.Close()
		return true
	}
	b.RequestFocus()
	b.Open()
	return true
}

// MouseRelease does nothing: the press opened the menu, and releasing
// over the button must not close it again. A drag that ran into the menu
// is the popup's business by then.
func (b *MenuButton) MouseRelease(widget.MouseEvent) bool { return true }

// KeyPress opens the menu on the keys every desktop opens one with.
func (b *MenuButton) KeyPress(e widget.KeyEvent) bool {
	if !b.Enabled() {
		return false
	}
	switch e.Key {
	case platform.KeySpace, platform.KeyReturn, platform.KeyDown:
		if b.pop != nil {
			return true
		}
		b.MarkKeyboardFocus()
		return b.Open()
	case platform.KeyEscape:
		if b.pop != nil {
			b.Close()
			return true
		}
	}
	return false
}

// HandleAccelerator runs a matching shortcut from this button's menu,
// which is what a menu bar does for its own and what a button calling
// ShowContextMenu never did. The window walks for this.
func (b *MenuButton) HandleAccelerator(key platform.Key, mods platform.Modifiers) bool {
	if !b.Enabled() || !b.Visible() {
		return false
	}
	return runMenuAccelerator(b.items(), key, mods)
}
