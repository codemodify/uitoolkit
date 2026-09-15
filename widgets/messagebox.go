package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// MessageKind selects the modal icon and tone.
type MessageKind int

const (
	MessageInfo MessageKind = iota
	MessageWarning
	MessageError
	MessageQuestion
)

// MessageButtons is the stock action row.
type MessageButtons int

const (
	ButtonsOK MessageButtons = iota
	ButtonsOKCancel
	ButtonsYesNo
	ButtonsYesNoCancel
)

// MessageResult is the button that closed a message box.
type MessageResult int

const (
	ResultNone MessageResult = iota
	ResultOK
	ResultCancel
	ResultYes
	ResultNo
)

func (k MessageKind) Icon() style.ToolIcon {
	switch k {
	case MessageWarning:
		return style.IconWarning
	case MessageError:
		return style.IconError
	case MessageQuestion:
		return style.IconQuestion
	default:
		return style.IconInfo
	}
}

func (k MessageKind) Title() string {
	switch k {
	case MessageWarning:
		return "Warning"
	case MessageError:
		return "Error"
	case MessageQuestion:
		return "Question"
	default:
		return "Message"
	}
}

// MessageBoxOptions configures ShowMessageBox / NewMessageBox.
type MessageBoxOptions struct {
	Title    string
	Message  string
	Kind     MessageKind
	Buttons  MessageButtons
	OnResult func(MessageResult)
}

// MessageBox is the card inside a modal Overlay.
type MessageBox struct {
	widget.Base
	opts    MessageBoxOptions
	result  MessageResult
	done    bool
	overlay *Overlay
}

// NewMessageBox builds an overlay + card. Call Show on a host, or use ShowMessageBox.
func NewMessageBox(opts MessageBoxOptions) *MessageBox {
	if opts.Title == "" {
		opts.Title = opts.Kind.Title()
	}
	mb := &MessageBox{opts: opts, result: ResultNone}
	mb.Init(mb)

	icon := &messageIcon{kind: opts.Kind}
	icon.Init(icon)

	body := NewColumn(NewLabel(opts.Message)).WithGap(8)

	head := NewRow(icon, body).WithGap(12).WithAlign(layout.AlignStart)
	head.AddFlex(body, 1)

	actions := mb.actionButtons()
	row := NewRow(actions...).WithGap(8).WithJustify(layout.JustifyEnd)
	col := NewColumn(head, row).WithGap(16).WithPad(6)
	card := NewPanel(opts.Title, col)
	card.Window = true
	card.Raised = true
	card.OnClose = func() { mb.finish(mb.cancelResult()) }

	mb.overlay = NewOverlay(card)
	mb.overlay.Modal = true
	mb.overlay.OnClose = func() { mb.finish(mb.cancelResult()) }
	// The default button takes focus and Enter; the order follows the look.
	// actionButtons lists them the Mac / GNOME way, default last ("Cancel
	// No Yes"); Windows / KDE read the same row backwards ("Yes No Cancel").
	primary := actions[len(actions)-1]
	mb.overlay.InitialFocus = primary
	mb.overlay.OnPresented = func() {
		if style.LookHint(mb.overlay.Look(), style.HintDialogPrimaryFirst) == 0 {
			return
		}
		for _, c := range actions {
			row.Remove(c)
		}
		for i := len(actions) - 1; i >= 0; i-- {
			row.Add(actions[i])
		}
	}
	return mb
}

func (mb *MessageBox) actionButtons() []widget.Component {
	add := func(label string, primary bool, res MessageResult) *Button {
		b := NewButton(label, func() { mb.finish(res) })
		b.Primary = primary
		return b
	}
	switch mb.opts.Buttons {
	case ButtonsOKCancel:
		return []widget.Component{add("Cancel", false, ResultCancel), add("OK", true, ResultOK)}
	case ButtonsYesNo:
		return []widget.Component{add("No", false, ResultNo), add("Yes", true, ResultYes)}
	case ButtonsYesNoCancel:
		return []widget.Component{
			add("Cancel", false, ResultCancel),
			add("No", false, ResultNo),
			add("Yes", true, ResultYes),
		}
	default:
		return []widget.Component{add("OK", true, ResultOK)}
	}
}

func (mb *MessageBox) cancelResult() MessageResult {
	switch mb.opts.Buttons {
	case ButtonsOK:
		return ResultOK
	case ButtonsYesNo:
		return ResultNo
	default:
		return ResultCancel
	}
}

func (mb *MessageBox) finish(res MessageResult) {
	if mb.done {
		return
	}
	mb.done = true
	mb.result = res
	if mb.overlay != nil {
		widget.DismissOverlay(mb.overlay)
	}
	if mb.opts.OnResult != nil {
		mb.opts.OnResult(res)
	}
}

// Overlay is the dimmed host layer.
func (mb *MessageBox) Overlay() *Overlay { return mb.overlay }

// Result is the last button, or ResultNone.
func (mb *MessageBox) Result() MessageResult { return mb.result }

// Show mounts the modal on the window that hosts from.
func (mb *MessageBox) Show(from widget.Component) bool {
	if mb.overlay == nil {
		return false
	}
	return widget.ShowOverlay(from, mb.overlay)
}

// ShowMessageBox mounts a modal dialog on the window that hosts from.
func ShowMessageBox(from widget.Component, opts MessageBoxOptions) *MessageBox {
	mb := NewMessageBox(opts)
	if !mb.Show(from) {
		return mb
	}
	return mb
}

// Info is ShowMessageBox with an OK button.
func Info(from widget.Component, title, message string, on func()) *MessageBox {
	return ShowMessageBox(from, MessageBoxOptions{
		Title: title, Message: message, Kind: MessageInfo, Buttons: ButtonsOK,
		OnResult: func(MessageResult) {
			if on != nil {
				on()
			}
		},
	})
}

// Confirm is Yes/No. on is called with true for Yes.
func Confirm(from widget.Component, title, message string, on func(bool)) *MessageBox {
	return ShowMessageBox(from, MessageBoxOptions{
		Title: title, Message: message, Kind: MessageQuestion, Buttons: ButtonsYesNo,
		OnResult: func(r MessageResult) {
			if on != nil {
				on(r == ResultYes)
			}
		},
	})
}

// Warn is a warning OK dialog.
func Warn(from widget.Component, title, message string, on func()) *MessageBox {
	return ShowMessageBox(from, MessageBoxOptions{
		Title: title, Message: message, Kind: MessageWarning, Buttons: ButtonsOK,
		OnResult: func(MessageResult) {
			if on != nil {
				on()
			}
		},
	})
}

type messageIcon struct {
	widget.Base
	kind MessageKind
}

func (m *messageIcon) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(36, 36))
}

func (m *messageIcon) Arrange(r paintengine2d.Rect) { m.SetBounds(r) }

func (m *messageIcon) Paint(ctx *paintengine2d.Context) {
	m.Look().DrawMessageIcon(ctx, m.LocalBounds(), m.kind.Icon())
}
