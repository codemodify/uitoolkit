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
	Application       = app.Application
	Options           = app.Options
	Window            = app.Window
	WindowOptions     = platform.WindowOptions
	LookAndFeel       = style.LookAndFeel
	Palette           = style.Palette
	Component         = widget.Component
	Menu              = widgets.Menu
	MenuItem          = widgets.MenuItem
	Tab               = widgets.Tab
	TreeNode          = widgets.TreeNode
	ToolItem          = widgets.ToolItem
	MessageKind       = widgets.MessageKind
	MessageButtons    = widgets.MessageButtons
	MessageResult     = widgets.MessageResult
	MessageBoxOptions = widgets.MessageBoxOptions
	TableColumn       = widgets.TableColumn
	FileInfo          = widgets.FileInfo
	FileDialogMode    = widgets.FileDialogMode
	FileDialogOptions = widgets.FileDialogOptions
	Density           = style.Density
	CardContent       = widgets.CardContent
	CardBadge         = widgets.CardBadge
	Cursor            = platform.Cursor
	Appearance        = style.Appearance
	ThemeName         = style.ThemeName
	CornerStyle       = style.CornerStyle
	IconSetName       = style.IconSetName
	ThemePack         = style.ThemePack
	ThemeSource       = style.ThemeSource
	IconSetInfo       = style.IconSetInfo
)

const (
	DensityDefault         = style.DensityDefault
	DensityCompact         = style.DensityCompact
	DensityRelaxed         = style.DensityRelaxed
	CursorDefault          = platform.CursorDefault
	CursorColResize        = platform.CursorColResize
	CursorRowResize        = platform.CursorRowResize
	CursorText             = platform.CursorText
	ThemeDark              = style.ThemeDark
	ThemeLight             = style.ThemeLight
	CornersRound           = style.CornersRound
	CornersSquare          = style.CornersSquare
	IconSetClassic         = style.IconSetClassic
	IconSetSharp           = style.IconSetSharp
	IconSetLucide          = style.IconSetLucide
	IconSetPhosphor        = style.IconSetPhosphor
	IconSetTabler          = style.IconSetTabler
	IconSetHeroicons       = style.IconSetHeroicons
	IconSetMaterialSymbols = style.IconSetMaterialSymbols
	ThemeSourceBuiltin     = style.ThemeSourceBuiltin
	ThemeSourceUser        = style.ThemeSourceUser
	DefaultThemeName       = style.DefaultThemeName
)

func New(opts Options) *Application { return app.New(opts) }

func DarkLook() *style.Classic  { return style.DarkLook() }
func LightLook() *style.Classic { return style.LightLook() }
func PreferredLook() style.LookAndFeel {
	return style.PreferredLook()
}
func LoadAppearance() style.Appearance { return style.LoadAppearance() }
func SaveAppearance(a style.Appearance) error {
	return style.SaveAppearance(a)
}
func AppearancePath() string { return style.AppearancePath() }
func ConfigDir() string      { return style.ConfigDir() }
func ThemesDir() string      { return style.ThemesDir() }
func IconsDir() string       { return style.IconsDir() }
func ListIconSets() []style.IconSetInfo {
	return style.ListIconSets()
}
func ListBuiltinIconSets() []style.IconSetInfo {
	return style.ListBuiltinIconSets()
}
func ListUserIconSets() []style.IconSetInfo {
	return style.ListUserIconSets()
}
func ListThemes() []style.ThemePack {
	return style.ListThemes()
}
func ListBuiltinThemes() []style.ThemePack {
	return style.ListBuiltinThemes()
}
func ListUserThemes() []style.ThemePack {
	return style.ListUserThemes()
}
func CanonicalStarterName(name string) string {
	return style.CanonicalStarterName(name)
}
func IsPremiereIconSet(name style.IconSetName) bool {
	return style.IsPremiereIconSet(name)
}
func LoadTheme(name string) (style.ThemePack, bool) {
	return style.LoadTheme(name)
}
func ExportTheme(name string, look style.LookAndFeel) (style.ThemePack, error) {
	return style.ExportTheme(name, look)
}
func ExportAppearance(name string, a style.Appearance) (style.ThemePack, error) {
	return style.ExportAppearance(name, a)
}
func StarterName(theme style.ThemeName) string {
	return style.StarterName(theme)
}
func SplitLookThemeName(name string) (string, style.CornerStyle, bool) {
	return style.SplitLookThemeName(name)
}
func LookAppearance(look style.LookAndFeel) style.Appearance {
	return style.LookAppearance(look)
}
func WithScale(look style.LookAndFeel, scale float32) style.LookAndFeel {
	return style.WithScale(look, scale)
}
func WithDensity(look style.LookAndFeel, d style.Density) style.LookAndFeel {
	return style.WithDensity(look, d)
}
func WithTheme(look style.LookAndFeel, theme style.ThemeName) style.LookAndFeel {
	return style.WithTheme(look, theme)
}
func WithCorners(look style.LookAndFeel, corners style.CornerStyle) style.LookAndFeel {
	return style.WithCorners(look, corners)
}
func WithIcons(look style.LookAndFeel, icons style.IconSetName) style.LookAndFeel {
	return style.WithIcons(look, icons)
}
func WithAppearance(look style.LookAndFeel, a style.Appearance) style.LookAndFeel {
	return style.WithAppearance(look, a)
}
func Dark() style.Palette  { return style.Dark() }
func Light() style.Palette { return style.Light() }

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
func NewMonoTextField(text, placeholder string, on func(string)) *widgets.TextField {
	return widgets.NewMonoTextField(text, placeholder, on)
}
func NewPasswordField(placeholder string, on func(string)) *widgets.TextField {
	return widgets.NewPasswordField(placeholder, on)
}
func NewScrollView(child widget.Component) *widgets.ScrollView {
	return widgets.NewScrollView(child)
}
func NewListView(count int, text func(int) string, on func(int)) *widgets.ListView {
	return widgets.NewListView(count, text, on)
}
func NewCardList(count int, card func(int) widgets.CardContent, on func(int)) *widgets.CardList {
	return widgets.NewCardList(count, card, on)
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
func NewToolBar(items ...*widgets.ToolItem) *widgets.ToolBar {
	return widgets.NewToolBar(items...)
}
func ToolText(text string, on func()) *widgets.ToolItem { return widgets.ToolText(text, on) }
func ToolIconBtn(icon style.ToolIcon, text string, on func()) *widgets.ToolItem {
	return widgets.ToolIconBtn(icon, text, on)
}
func ToolToggle(text string, down bool, on func()) *widgets.ToolItem {
	return widgets.ToolToggle(text, down, on)
}
func ToolDivider() *widgets.ToolItem { return widgets.ToolDivider() }
func NewComboBox(items []string, selected int, on func(int)) *widgets.ComboBox {
	return widgets.NewComboBox(items, selected, on)
}
func NewProgressBar(value float32) *widgets.ProgressBar { return widgets.NewProgressBar(value) }
func NewBusyBar(phase float32) *widgets.ProgressBar     { return widgets.NewBusyBar(phase) }
func NewRadio(text string, selected bool, on func(bool)) *widgets.RadioButton {
	return widgets.NewRadio(text, selected, on)
}
func NewRadioGroup(labels []string, selected int, on func(int)) *widgets.RadioGroup {
	return widgets.NewRadioGroup(labels, selected, on)
}
func NewTitleBar(title, subtitle string) *widgets.TitleBar {
	return widgets.NewTitleBar(title, subtitle)
}
func NewMessageBox(opts widgets.MessageBoxOptions) *widgets.MessageBox {
	return widgets.NewMessageBox(opts)
}
func ShowMessageBox(from widget.Component, opts widgets.MessageBoxOptions) *widgets.MessageBox {
	return widgets.ShowMessageBox(from, opts)
}
func Info(from widget.Component, title, message string, on func()) *widgets.MessageBox {
	return widgets.Info(from, title, message, on)
}
func Confirm(from widget.Component, title, message string, on func(bool)) *widgets.MessageBox {
	return widgets.Confirm(from, title, message, on)
}
func Warn(from widget.Component, title, message string, on func()) *widgets.MessageBox {
	return widgets.Warn(from, title, message, on)
}
func NewTableView(cols []widgets.TableColumn, rows int, cell func(row, col int) string, on func(int)) *widgets.TableView {
	return widgets.NewTableView(cols, rows, cell, on)
}
func NewNumberField(min, max, value, step float64, on func(float64)) *widgets.NumberField {
	return widgets.NewNumberField(min, max, value, step, on)
}
func NewSpinner(min, max, value, step float64, on func(float64)) *widgets.NumberField {
	return widgets.NewSpinner(min, max, value, step, on)
}
func NewTip(text string, child widget.Component) *widgets.TipWrap {
	return widgets.NewTip(text, child)
}
func NewTextArea(text, placeholder string, on func(string)) *widgets.TextArea {
	return widgets.NewTextArea(text, placeholder, on)
}
func NewMonoTextArea(text, placeholder string, on func(string)) *widgets.TextArea {
	return widgets.NewMonoTextArea(text, placeholder, on)
}
func NewTextView(text, placeholder string) *widgets.TextArea {
	return widgets.NewTextView(text, placeholder)
}
func NewMonoTextView(text, placeholder string) *widgets.TextArea {
	return widgets.NewMonoTextView(text, placeholder)
}

const (
	FamilyUI   = style.FamilyUI
	FamilyMono = style.FamilyMono
)

type FontRole = style.FontRole

const (
	RoleUI   = style.RoleUI
	RoleMono = style.RoleMono
)

func NewSwitch(text string, on bool, change func(bool)) *widgets.Switch {
	return widgets.NewSwitch(text, on, change)
}
func NewExpander(title string, expanded bool, child widget.Component) *widgets.Expander {
	return widgets.NewExpander(title, expanded, child)
}
func NewAccordion(exclusive bool, items ...*widgets.Expander) *widgets.Accordion {
	return widgets.NewAccordion(exclusive, items...)
}
func NewSeparator() *widgets.Separator  { return widgets.NewSeparator() }
func NewVSeparator() *widgets.Separator { return widgets.NewVSeparator() }
func NewSpacer() *widgets.Spacer        { return widgets.NewSpacer() }
func NewSpacerSize(w, h float32) *widgets.Spacer {
	return widgets.NewSpacerSize(w, h)
}
func NewFileDialog(opts widgets.FileDialogOptions) *widgets.FileDialog {
	return widgets.NewFileDialog(opts)
}
func ShowFileDialog(from widget.Component, opts widgets.FileDialogOptions) *widgets.FileDialog {
	return widgets.ShowFileDialog(from, opts)
}
func ReadDirEntries(path string) ([]widgets.FileInfo, error) {
	return widgets.ReadDirEntries(path)
}

const (
	IconNone     = style.IconNone
	IconNew      = style.IconNew
	IconOpen     = style.IconOpen
	IconSave     = style.IconSave
	IconCut      = style.IconCut
	IconCopy     = style.IconCopy
	IconPaste    = style.IconPaste
	IconUndo     = style.IconUndo
	IconRedo     = style.IconRedo
	IconSearch   = style.IconSearch
	IconInfo     = style.IconInfo
	IconWarning  = style.IconWarning
	IconError    = style.IconError
	IconQuestion = style.IconQuestion

	MessageInfo     = widgets.MessageInfo
	MessageWarning  = widgets.MessageWarning
	MessageError    = widgets.MessageError
	MessageQuestion = widgets.MessageQuestion

	ButtonsOK          = widgets.ButtonsOK
	ButtonsOKCancel    = widgets.ButtonsOKCancel
	ButtonsYesNo       = widgets.ButtonsYesNo
	ButtonsYesNoCancel = widgets.ButtonsYesNoCancel

	ResultNone   = widgets.ResultNone
	ResultOK     = widgets.ResultOK
	ResultCancel = widgets.ResultCancel
	ResultYes    = widgets.ResultYes
	ResultNo     = widgets.ResultNo

	FileOpen = widgets.FileOpen
	FileSave = widgets.FileSave
)

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
