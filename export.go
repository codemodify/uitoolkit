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
