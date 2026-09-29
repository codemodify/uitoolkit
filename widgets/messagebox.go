package widgets

import (
	"strings"

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
	Title   string
	Message string
	Kind    MessageKind
	Buttons MessageButtons
	// Input, when set, puts a single-line field under the message and
	// makes this a prompt: the field takes the focus instead of the
	// default button, Return accepts, and [MessageBox.Text] reads the
	// value back. [Prompt] is the short way to one.
	Input    *MessageBoxInput
	OnResult func(MessageResult)
}

// MessageBoxInput is the field [MessageBoxOptions.Input] adds.
type MessageBoxInput struct {
	Text        string // the value the field starts with, selected
	Placeholder string
	Password    bool
	// Required greys the accepting button out while the field is empty,
	// and makes Return do nothing — a "name this folder" dialog must not
	// be able to return an empty name.
	Required bool
}

// MessageBox is the card inside a modal Overlay.
type MessageBox struct {
	widget.Base
	opts    MessageBoxOptions
	result  MessageResult
	done    bool
	overlay *Overlay
	field   *TextField
	primary *Button
	text    string
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
	if in := opts.Input; in != nil {
		mb.text = in.Text
		mb.field = NewTextField(in.Text, in.Placeholder, func(s string) {
			mb.text = s
			if mb.primary != nil && in.Required {
				mb.primary.SetEnabled(mb.acceptable())
			}
		})
		mb.field.Password = in.Password
		// The initial value is a suggestion — a folder's current name, a
		// default file name — and typing replaces it rather than appending
		// to it, which is what every rename dialog does.
		mb.field.SelectAll()
		body.Add(mb.field)
	}

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
	if mb.field != nil {
		// A prompt is there to be typed in: the field takes the focus,
		// and Return is what the default button would have been. Escape
		// is left alone — the field lets it bubble to the overlay, which
		// is the cancel the rest of this dialog already has.
		mb.overlay.InitialFocus = mb.field
		mb.field.OnSubmit = func(string) {
			if mb.acceptable() {
				mb.finish(mb.acceptResult())
			}
		}
		if mb.primary != nil {
			mb.primary.SetEnabled(mb.acceptable())
		}
	}
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
		if primary {
			mb.primary = b
		}
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

// acceptResult is what the default button of this button set returns.
func (mb *MessageBox) acceptResult() MessageResult {
	switch mb.opts.Buttons {
	case ButtonsYesNo, ButtonsYesNoCancel:
		return ResultYes
	default:
		return ResultOK
	}
}

// acceptable is false only for a Required field left empty.
func (mb *MessageBox) acceptable() bool {
	if mb.field == nil || mb.opts.Input == nil || !mb.opts.Input.Required {
		return true
	}
	return strings.TrimSpace(mb.field.Text) != ""
}

func (mb *MessageBox) finish(res MessageResult) {
	if mb.done {
		return
	}
	mb.done = true
	mb.result = res
	if mb.field != nil {
		mb.text = mb.field.Text
	}
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

// Text is what [MessageBoxOptions.Input]'s field holds. It is the empty
// string for a message box without one.
func (mb *MessageBox) Text() string { return mb.text }

// Field is the input field, or nil. It is here so a caller can reach the
// things a field has and a dialog cannot guess — Accept, Clearable, Mono.
func (mb *MessageBox) Field() *TextField { return mb.field }

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

// Prompt asks for one line of text. on is called with the field's value
// and true for OK, and with it and false for Cancel or Escape — the same
// shape as [Confirm], with the answer beside the yes or no.
//
// It is the "name this" dialog: a new folder, a rename, a saved search.
// Anything more than one field is a form, and a form is a window.
func Prompt(from widget.Component, title, label, initial string, on func(string, bool)) *MessageBox {
	var mb *MessageBox
	mb = NewMessageBox(MessageBoxOptions{
		Title: title, Message: label, Kind: MessageQuestion, Buttons: ButtonsOKCancel,
		Input: &MessageBoxInput{Text: initial},
		OnResult: func(r MessageResult) {
			if on != nil {
				on(mb.Text(), r == ResultOK)
			}
		},
	})
	mb.Show(from)
	return mb
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
