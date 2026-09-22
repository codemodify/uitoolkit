package app

import "github.com/codemodify/uitoolkit/a11y"

// AT-SPI2's numbering of roles and states (AtspiRole, AtspiStateType in
// the published D-Bus interface), for the Linux adapter.

// AT-SPI roles used here.
const (
	atspiRoleAlert          = 2
	atspiRoleCalendar       = 5
	atspiRoleCheckBox       = 7
	atspiRoleCheckMenuItem  = 8
	atspiRoleComboBox       = 11
	atspiRoleDesktopFrame   = 14
	atspiRoleDialog         = 16
	atspiRoleFrame          = 23
	atspiRoleImage          = 27
	atspiRoleInternalFrame  = 28
	atspiRoleLabel          = 29
	atspiRoleList           = 31
	atspiRoleListItem       = 32
	atspiRoleMenu           = 33
	atspiRoleMenuBar        = 34
	atspiRoleMenuItem       = 35
	atspiRolePageTab        = 37
	atspiRolePageTabList    = 38
	atspiRolePanel          = 39
	atspiRolePasswordText   = 40
	atspiRoleProgressBar    = 42
	atspiRolePushButton     = 43
	atspiRoleRadioButton    = 44
	atspiRoleRadioMenuItem  = 45
	atspiRoleScrollBar      = 48
	atspiRoleScrollPane     = 49
	atspiRoleSeparator      = 50
	atspiRoleSlider         = 51
	atspiRoleSpinButton     = 52
	atspiRoleSplitPane      = 53
	atspiRoleStatusBar      = 54
	atspiRoleTable          = 55
	atspiRoleTableCell      = 56
	atspiRoleTableColHeader = 57
	atspiRoleText           = 61
	atspiRoleToggleButton   = 62
	atspiRoleToolBar        = 63
	atspiRoleToolTip        = 64
	atspiRoleTree           = 65
	atspiRoleUnknown        = 67
	atspiRoleApplication    = 75
	atspiRoleEntry          = 79
	atspiRoleHeading        = 83
	atspiRoleLink           = 88
	atspiRoleTableRow       = 90
	atspiRoleTreeItem       = 91
	atspiRoleGrouping       = 99
	atspiRoleTitleBar       = 104
	atspiRoleSwitch         = 130
)

var atspiRoles = map[a11y.Role]uint32{
	a11y.RoleUnknown:       atspiRoleUnknown,
	a11y.RoleWindow:        atspiRoleFrame,
	a11y.RoleDialog:        atspiRoleDialog,
	a11y.RoleAlert:         atspiRoleAlert,
	a11y.RoleGroup:         atspiRoleGrouping,
	a11y.RolePane:          atspiRolePanel,
	a11y.RoleButton:        atspiRolePushButton,
	a11y.RoleToggleButton:  atspiRoleToggleButton,
	a11y.RoleCheckBox:      atspiRoleCheckBox,
	a11y.RoleRadioButton:   atspiRoleRadioButton,
	a11y.RoleSwitch:        atspiRoleSwitch,
	a11y.RoleLabel:         atspiRoleLabel,
	a11y.RoleHeading:       atspiRoleHeading,
	a11y.RoleTextField:     atspiRoleEntry,
	a11y.RolePasswordField: atspiRolePasswordText,
	a11y.RoleTextArea:      atspiRoleText,
	a11y.RoleComboBox:      atspiRoleComboBox,
	a11y.RoleSpinButton:    atspiRoleSpinButton,
	a11y.RoleSlider:        atspiRoleSlider,
	a11y.RoleProgressBar:   atspiRoleProgressBar,
	a11y.RoleScrollBar:     atspiRoleScrollBar,
	a11y.RoleScrollPane:    atspiRoleScrollPane,
	a11y.RoleList:          atspiRoleList,
	a11y.RoleListItem:      atspiRoleListItem,
	a11y.RoleTree:          atspiRoleTree,
	a11y.RoleTreeItem:      atspiRoleTreeItem,
	a11y.RoleTable:         atspiRoleTable,
	a11y.RoleRow:           atspiRoleTableRow,
	a11y.RoleCell:          atspiRoleTableCell,
	a11y.RoleColumnHeader:  atspiRoleTableColHeader,
	a11y.RoleTabList:       atspiRolePageTabList,
	a11y.RoleTab:           atspiRolePageTab,
	a11y.RoleTabPanel:      atspiRolePanel,
	a11y.RoleMenuBar:       atspiRoleMenuBar,
	a11y.RoleMenu:          atspiRoleMenu,
	a11y.RoleMenuItem:      atspiRoleMenuItem,
	a11y.RoleCheckMenuItem: atspiRoleCheckMenuItem,
	a11y.RoleRadioMenuItem: atspiRoleRadioMenuItem,
	a11y.RoleSeparator:     atspiRoleSeparator,
	a11y.RoleToolBar:       atspiRoleToolBar,
	a11y.RoleStatusBar:     atspiRoleStatusBar,
	a11y.RoleSplitter:      atspiRoleSplitPane,
	a11y.RoleImage:         atspiRoleImage,
	a11y.RoleCalendar:      atspiRoleCalendar,
	a11y.RoleToolTip:       atspiRoleToolTip,
	a11y.RoleLink:          atspiRoleLink,
	a11y.RoleTitleBar:      atspiRoleTitleBar,
	a11y.RoleDesktopPane:   atspiRoleDesktopFrame,
	a11y.RoleInternalFrame: atspiRoleInternalFrame,
}

// atspiRoleNames are AT-SPI's names for the roles used here (what
// GetRoleName answers; a screen reader may speak them).
var atspiRoleNames = map[uint32]string{
	atspiRoleAlert: "alert", atspiRoleCalendar: "calendar", atspiRoleCheckBox: "check box",
	atspiRoleCheckMenuItem: "check menu item", atspiRoleComboBox: "combo box", atspiRoleDialog: "dialog",
	atspiRoleFrame: "frame", atspiRoleDesktopFrame: "desktop frame", atspiRoleInternalFrame: "internal frame", atspiRoleImage: "image", atspiRoleLabel: "label", atspiRoleList: "list",
	atspiRoleListItem: "list item", atspiRoleMenu: "menu", atspiRoleMenuBar: "menu bar",
	atspiRoleMenuItem: "menu item", atspiRolePageTab: "page tab", atspiRolePageTabList: "page tab list",
	atspiRolePanel: "panel", atspiRolePasswordText: "password text", atspiRoleProgressBar: "progress bar",
	atspiRolePushButton: "push button", atspiRoleRadioButton: "radio button",
	atspiRoleRadioMenuItem: "radio menu item", atspiRoleScrollBar: "scroll bar",
	atspiRoleScrollPane: "scroll pane", atspiRoleSeparator: "separator", atspiRoleSlider: "slider",
	atspiRoleSpinButton: "spin button", atspiRoleSplitPane: "split pane", atspiRoleStatusBar: "status bar",
	atspiRoleTable: "table", atspiRoleTableCell: "table cell", atspiRoleTableColHeader: "table column header",
	atspiRoleText: "text", atspiRoleToggleButton: "toggle button", atspiRoleToolBar: "tool bar",
	atspiRoleToolTip: "tool tip", atspiRoleTree: "tree", atspiRoleUnknown: "unknown",
	atspiRoleApplication: "application", atspiRoleEntry: "entry", atspiRoleHeading: "heading",
	atspiRoleLink: "link", atspiRoleTableRow: "table row", atspiRoleTreeItem: "tree item",
	atspiRoleGrouping: "grouping", atspiRoleSwitch: "switch", atspiRoleTitleBar: "title bar",
}

func atspiRole(r a11y.Role) uint32 {
	if v, ok := atspiRoles[r]; ok {
		return v
	}
	return atspiRoleUnknown
}

// AT-SPI states used here.
const (
	atspiStateActive          = 1
	atspiStateChecked         = 4
	atspiStateCollapsed       = 5
	atspiStateEditable        = 7
	atspiStateEnabled         = 8
	atspiStateExpandable      = 9
	atspiStateExpanded        = 10
	atspiStateFocusable       = 11
	atspiStateFocused         = 12
	atspiStateHorizontal      = 14
	atspiStateModal           = 16
	atspiStateMultiLine       = 17
	atspiStateMultiSelectable = 18
	atspiStatePressed         = 20
	atspiStateSelectable      = 22
	atspiStateSelected        = 23
	atspiStateSensitive       = 24
	atspiStateShowing         = 25
	atspiStateSingleLine      = 26
	atspiStateVertical        = 29
	atspiStateVisible         = 30
	atspiStateIndeterminate   = 32
	atspiStateIsDefault       = 39
	atspiStateCheckable       = 41
	atspiStateHasPopup        = 42
	atspiStateReadOnly        = 43
)

// atspiStates is n's state set as AT-SPI's two 32-bit words.
func atspiStates(n *a11y.Node) [2]uint32 {
	var w [2]uint32
	set := func(bit uint) { w[bit/32] |= 1 << (bit % 32) }
	s := n.State
	if !s.Has(a11y.StateDisabled) {
		set(atspiStateEnabled)
		set(atspiStateSensitive)
	}
	if !s.Has(a11y.StateOffscreen) {
		set(atspiStateVisible)
		set(atspiStateShowing)
	}
	for _, m := range []struct {
		a   a11y.State
		bit uint
	}{
		{a11y.StateFocusable, atspiStateFocusable},
		{a11y.StateFocused, atspiStateFocused},
		{a11y.StateCheckable, atspiStateCheckable},
		{a11y.StateChecked, atspiStateChecked},
		{a11y.StateMixed, atspiStateIndeterminate},
		{a11y.StatePressed, atspiStatePressed},
		{a11y.StateSelectable, atspiStateSelectable},
		{a11y.StateSelected, atspiStateSelected},
		{a11y.StateExpandable, atspiStateExpandable},
		{a11y.StateExpanded, atspiStateExpanded},
		{a11y.StateEditable, atspiStateEditable},
		{a11y.StateReadOnly, atspiStateReadOnly},
		{a11y.StateMultiLine, atspiStateMultiLine},
		{a11y.StateMultiSelectable, atspiStateMultiSelectable},
		{a11y.StateModal, atspiStateModal},
		{a11y.StateDefault, atspiStateIsDefault},
		{a11y.StateHasPopup, atspiStateHasPopup},
		{a11y.StateHorizontal, atspiStateHorizontal},
		{a11y.StateVertical, atspiStateVertical},
	} {
		if s.Has(m.a) {
			set(m.bit)
		}
	}
	if s.Has(a11y.StateExpandable) && !s.Has(a11y.StateExpanded) {
		set(atspiStateCollapsed)
	}
	switch n.Role {
	case a11y.RoleTextField, a11y.RolePasswordField:
		set(atspiStateSingleLine)
	case a11y.RoleWindow:
		if s.Has(a11y.StateFocused) {
			// A window is "active", not focused, in AT-SPI.
			w[atspiStateFocused/32] &^= 1 << (atspiStateFocused % 32)
			set(atspiStateActive)
		}
	}
	return w
}

// atspiActionName is the name AT-SPI clients (Orca) know an action by.
func atspiActionName(n *a11y.Node, a a11y.Action) string {
	switch a {
	case a11y.ActionDefault:
		switch n.Role {
		case a11y.RoleCheckBox, a11y.RoleSwitch, a11y.RoleToggleButton, a11y.RoleCheckMenuItem:
			return "toggle"
		case a11y.RoleListItem, a11y.RoleTreeItem, a11y.RoleRow, a11y.RoleTab:
			return "activate"
		case a11y.RoleComboBox:
			return "press"
		}
		return "click"
	case a11y.ActionExpand:
		return "expand"
	case a11y.ActionCollapse:
		return "collapse"
	case a11y.ActionIncrement:
		return "increment"
	case a11y.ActionDecrement:
		return "decrement"
	case a11y.ActionShowMenu:
		return "showmenu"
	case a11y.ActionScrollIntoView:
		return "scroll into view"
	case a11y.ActionFocus:
		return "focus"
	}
	return a.String()
}
