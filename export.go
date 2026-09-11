package uitoolkit

import (
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Re-exported types so a typical app imports this module once.

type (
	Application   = app.Application
	Options       = app.Options
	Window        = app.Window
	WindowOptions = platform.WindowOptions
	LookAndFeel   = style.LookAndFeel
	Palette       = style.Palette
	Component     = widget.Component
	Menu          = widgets.Menu
	MenuItem      = widgets.MenuItem
	Tab           = widgets.Tab
	TreeNode      = widgets.TreeNode
)

func New(opts Options) *Application { return app.New(opts) }

func DarkLook() *style.Classic  { return style.DarkLook() }
func LightLook() *style.Classic { return style.LightLook() }
func Dark() style.Palette       { return style.Dark() }
func Light() style.Palette      { return style.Light() }

func NewColumn(children ...widget.Component) *widgets.FlexBox {
	return widgets.NewColumn(children...)
}
func NewRow(children ...widget.Component) *widgets.FlexBox { return widgets.NewRow(children...) }
func NewStack(children ...widget.Component) *widgets.Stack { return widgets.NewStack(children...) }
func NewPad(v float32, child widget.Component) *widgets.Pad {
	return widgets.NewPad(v, child)
}
func NewPanel(title string, children ...widget.Component) *widgets.Panel {
	return widgets.NewPanel(title, children...)
}
func NewLabel(text string) *widgets.Label { return widgets.NewLabel(text) }
func NewTitle(text string) *widgets.Label { return widgets.NewTitle(text) }
func NewButton(text string, on func()) *widgets.Button {
	return widgets.NewButton(text, on)
}
func NewCheckbox(text string, checked bool, on func(bool)) *widgets.Checkbox {
	return widgets.NewCheckbox(text, checked, on)
}
func NewSlider(min, max, value float32, on func(float32)) *widgets.Slider {
	return widgets.NewSlider(min, max, value, on)
}
func NewTextField(text, placeholder string, on func(string)) *widgets.TextField {
	return widgets.NewTextField(text, placeholder, on)
}
func NewScrollView(child widget.Component) *widgets.ScrollView {
	return widgets.NewScrollView(child)
}
func NewListView(count int, text func(int) string, on func(int)) *widgets.ListView {
	return widgets.NewListView(count, text, on)
}
func NewSplitter(vertical bool, a, b widget.Component) *widgets.Splitter {
	return widgets.NewSplitter(vertical, a, b)
}
func NewOverlay(card widget.Component) *widgets.Overlay { return widgets.NewOverlay(card) }
func DialogCard(title, body string, actions ...widget.Component) *widgets.Panel {
	return widgets.DialogCard(title, body, actions...)
}
func NewMenuBar(menus ...*widgets.Menu) *widgets.MenuBar { return widgets.NewMenuBar(menus...) }
func NewMenu(title string, items ...*widgets.MenuItem) *widgets.Menu {
	return widgets.NewMenu(title, items...)
}
func NewMenuItem(text string, on func()) *widgets.MenuItem { return widgets.Item(text, on) }
func NewMenuItemAccel(text, shortcut string, on func()) *widgets.MenuItem {
	return widgets.ItemAccel(text, shortcut, on)
}
func MenuSep() *widgets.MenuItem { return widgets.Sep() }
func NewPopupMenu(items ...*widgets.MenuItem) *widgets.PopupMenu {
	return widgets.NewPopupMenu(items...)
}
func NewTabBar(titles ...string) *widgets.TabBar { return widgets.NewTabBar(titles...) }
func NewTabPage(child widget.Component) *widgets.TabPage {
	return widgets.NewTabPage(child)
}
func NewTabView(tabs ...widgets.Tab) *widgets.TabView { return widgets.NewTabView(tabs...) }
func NewTreeNode(label string, kids ...*widgets.TreeNode) *widgets.TreeNode {
	return widgets.NewTreeNode(label, kids...)
}
func NewTreeView(roots ...*widgets.TreeNode) *widgets.TreeView {
	return widgets.NewTreeView(roots...)
}
func NewStatusBar(parts ...string) *widgets.StatusBar { return widgets.NewStatusBar(parts...) }

// Layout constants.
const (
	AlignStart          = layout.AlignStart
	AlignCenter         = layout.AlignCenter
	AlignEnd            = layout.AlignEnd
	AlignStretch        = layout.AlignStretch
	JustifyStart        = layout.JustifyStart
	JustifyCenter       = layout.JustifyCenter
	JustifyEnd          = layout.JustifyEnd
	JustifySpaceBetween = layout.JustifySpaceBetween
)
