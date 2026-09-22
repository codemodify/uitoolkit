// Package a11y is uitoolkit's accessibility model. Every widget describes
// itself as a node of a tree (its role, name, state, value and the actions
// it supports) and a platform adapter hands the tree to assistive
// technology: AT-SPI2 on Linux (Orca, accerciser), later UI Automation and
// NSAccessibility. The same split as AccessKit's: widgets never talk to a
// platform API.
//
// The tree is built on demand, when an adapter asks for it, so an app
// pays nothing while no assistive technology is running.
package a11y

import "github.com/codemodify/paintengine2d"

// Role is what a node is to assistive technology (ARIA's and AT-SPI's
// roles, narrowed to what a desktop toolkit shows).
type Role uint8

const (
	RoleUnknown Role = iota
	RoleWindow
	RoleDialog
	RoleAlert // a message box
	RoleGroup // a group box, a titled panel
	RolePane  // a plain container that is still worth naming
	RoleButton
	RoleToggleButton
	RoleCheckBox
	RoleRadioButton
	RoleSwitch
	RoleLabel
	RoleHeading
	RoleTextField
	RolePasswordField
	RoleTextArea
	RoleComboBox
	RoleSpinButton
	RoleSlider
	RoleProgressBar
	RoleScrollBar
	RoleScrollPane
	RoleList
	RoleListItem
	RoleTree
	RoleTreeItem
	RoleTable
	RoleRow
	RoleCell
	RoleColumnHeader
	RoleTabList
	RoleTab
	RoleTabPanel
	RoleMenuBar
	RoleMenu
	RoleMenuItem
	RoleCheckMenuItem
	RoleRadioMenuItem
	RoleSeparator
	RoleToolBar
	RoleStatusBar
	RoleSplitter
	RoleImage
	RoleCalendar
	RoleToolTip
	RoleLink
	// RoleTitleBar is a window's title bar when the toolkit draws it: its
	// content and the caption buttons.
	RoleTitleBar
	// RoleDesktopPane holds windows inside a window (an MDI area), and
	// RoleInternalFrame is one of them: AT-SPI's desktop frame and
	// internal frame, the roles for a multiple-document interface.
	RoleDesktopPane
	RoleInternalFrame
	roleCount
)

var roleNames = [roleCount]string{
	"unknown", "window", "dialog", "alert", "group", "pane", "button",
	"toggle button", "check box", "radio button", "switch", "label",
	"heading", "text field", "password field", "text area", "combo box",
	"spin button", "slider", "progress bar", "scroll bar", "scroll pane",
	"list", "list item", "tree", "tree item", "table", "row", "cell",
	"column header", "tab list", "tab", "tab panel", "menu bar", "menu",
	"menu item", "check menu item", "radio menu item", "separator",
	"tool bar", "status bar", "splitter", "image", "calendar", "tool tip",
	"link", "title bar", "desktop pane", "internal frame",
}

func (r Role) String() string {
	if r < roleCount {
		return roleNames[r]
	}
	return "unknown"
}

// Interactive reports whether the role is a control the user operates,
// which must carry a name.
func (r Role) Interactive() bool {
	switch r {
	case RoleButton, RoleToggleButton, RoleCheckBox, RoleRadioButton, RoleSwitch,
		RoleTextField, RolePasswordField, RoleTextArea, RoleComboBox, RoleSpinButton,
		RoleSlider, RoleTab, RoleMenuItem, RoleCheckMenuItem, RoleRadioMenuItem, RoleLink:
		return true
	}
	return false
}

// State is a set of a node's states.
type State uint32

const (
	StateFocusable State = 1 << iota
	StateFocused
	StateDisabled
	StateCheckable
	StateChecked
	// StateMixed: a check box that is neither on nor off.
	StateMixed
	// StatePressed: a toggle button that is down.
	StatePressed
	StateSelectable
	StateSelected
	StateExpandable
	StateExpanded
	StateEditable
	StateReadOnly
	StateMultiLine
	StateMultiSelectable
	StateModal
	// StateDefault: the button Return presses.
	StateDefault
	StateHasPopup
	// StateOffscreen: scrolled out of its view (still in the tree).
	StateOffscreen
	StateHorizontal
	StateVertical
)

// Has reports whether every state in q is set.
func (s State) Has(q State) bool { return s&q == q }

// Action is something assistive technology can ask a node to do.
type Action uint8

const (
	// ActionDefault is the node's main action: press a button, toggle a
	// check box, select an item, open a menu.
	ActionDefault Action = iota
	ActionFocus
	ActionIncrement
	ActionDecrement
	ActionExpand
	ActionCollapse
	ActionShowMenu
	ActionScrollIntoView
	actionCount
)

var actionNames = [actionCount]string{"default", "focus", "increment", "decrement", "expand", "collapse", "show menu", "scroll into view"}

func (a Action) String() string {
	if a < actionCount {
		return actionNames[a]
	}
	return "unknown"
}

// Actions is a set of actions.
type Actions uint16

// With adds a.
func (s Actions) With(a Action) Actions { return s | 1<<a }

// Has reports whether a is in the set.
func (s Actions) Has(a Action) bool { return s&(1<<a) != 0 }

// Node is one accessible object.
type Node struct {
	// ID is stable for the node's life: a component's ID, or one derived
	// from its view and the item's index.
	ID   uint64
	Role Role
	// Name is what the node is called ("Save", "First name"); Description
	// adds to it (a tooltip).
	Name, Description string
	// Value is a field's text, a combo box's choice, a slider's reading.
	Value string
	State State
	// Bounds is the node's box in window coordinates, device pixels.
	Bounds paintengine2d.Rect
	// A range: sliders, progress bars, spin buttons, scroll bars.
	HasRange            bool
	Min, Max, Now, Step float64
	// Position in a set of siblings (list items, tabs, radio buttons,
	// menu items), 1-based, and the set's size; Level is a tree item's
	// depth, 1-based. Zero: not given.
	Index, Count, Level int
	// Caret and selection in an editable text (character offsets).
	Caret, SelStart, SelEnd int
	// Shortcut is a menu item's accelerator ("Ctrl+S").
	Shortcut string
	Actions  Actions
	Children []*Node
}

// Walk visits n and its descendants depth first; returning false from fn
// skips a node's children.
func (n *Node) Walk(fn func(*Node) bool) {
	if n == nil || !fn(n) {
		return
	}
	for _, c := range n.Children {
		c.Walk(fn)
	}
}

// Find returns the node with the given ID in n's subtree.
func (n *Node) Find(id uint64) *Node {
	var hit *Node
	n.Walk(func(c *Node) bool {
		if hit != nil {
			return false
		}
		if c.ID == id {
			hit = c
			return false
		}
		return true
	})
	return hit
}
