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
	// Validate vets the value when the accepting button is pressed.
	// Returning an error keeps the dialog up and shows the error under
	// the field, with what the user typed still there.
	//
	// Without it a prompt could only dismiss and report, so a value the
	// caller had to refuse — "a folder with that name already exists" —
	// meant a second dialog complaining, then the prompt opened again
	// with the text put back. The refusal belongs where the value was
	// typed.
	//
	// It runs on the UI goroutine, so it is for a check the caller can
	// make at once: empty, malformed, already in a list the caller
	// holds. A check that has to ask something else — a server, a
	// daemon, a file system — is [MessageBoxInput.ValidateAsync].
	Validate func(string) error
	// ValidateAsync vets the value when the check cannot answer at once:
	// the mail server that has to be asked whether a folder name is
	// taken, the vault daemon that has to be asked whether a key
	// unlocks.
	//
	// The dialog stays up with the accepting button busy, and ignores
	// further presses, until done is called. done(nil) closes the dialog
	// with its accepting result, so OnResult runs exactly as it would
	// have; done(err) puts the reason under the field, re-enables the
	// button, and leaves what the user typed where it is.
	//
	// It is called on the UI goroutine and done may be called from any
	// goroutine — hand it to the application's dispatcher if the answer
	// arrives on another. done is safe to call more than once; the
	// second call does nothing.
	//
	// Set this or [MessageBoxInput.Validate], not both; Validate wins.
	ValidateAsync func(value string, done func(error))
	// AcceptLabel names the accepting button ("Rename", "Create") in
	// place of the stock OK or Yes. Every desktop's guidelines say a
	// button names its action.
	AcceptLabel string
}

// MessageBox is the card inside a modal Overlay.
type MessageBox struct {
	widget.Base
	opts     MessageBoxOptions
	result   MessageResult
	done     bool
	overlay  *Overlay
	field    *TextField
	errLabel *Label
	checking bool
	primary  *Button
	text     string
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
				mb.primary.SetEnabled(!mb.checking && mb.acceptable())
			}
		})
		mb.field.Password = in.Password
		// The initial value is a suggestion — a folder's current name, a
		// default file name — and typing replaces it rather than appending
		// to it, which is what every rename dialog does.
		mb.field.SelectAll()
		body.Add(mb.field)
		// The refusal goes under the field, where the value was typed.
		// It is hidden until there is one, so it takes no room in a
		// dialog that never refuses anything.
		mb.errLabel = NewLabel("")
		mb.errLabel.Wrap = true
		mb.errLabel.Tone = ToneDanger
		mb.errLabel.SetVisible(false)
		body.Add(mb.errLabel)
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
				mb.accept(mb.acceptResult())
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
		if primary && mb.opts.Input != nil && mb.opts.Input.AcceptLabel != "" {
			label = mb.opts.Input.AcceptLabel
		}
		b := NewButton(label, func() { mb.accept(res) })
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

// accept runs the caller's check before dismissing, where there is one
// and the result is the accepting one. A refused value keeps the dialog
// up with the reason under the field.
func (mb *MessageBox) accept(res MessageResult) {
	in := mb.opts.Input
	if mb.field == nil || in == nil || res != mb.acceptResult() {
		mb.finish(res)
		return
	}
	if mb.checking {
		// A check is already on its way. A second press must not start
		// another, and must not close the dialog under the first.
		return
	}
	if in.Validate != nil {
		if err := in.Validate(mb.field.Text); err != nil {
			mb.SetInputError(err.Error())
			return
		}
		mb.finish(res)
		return
	}
	if in.ValidateAsync == nil {
		mb.finish(res)
		return
	}
	mb.setChecking(true)
	answered := false
	in.ValidateAsync(mb.field.Text, func(err error) {
		// Once. A callback invoked twice must not close a dialog the
		// user has since reopened, or report a result twice.
		if answered {
			return
		}
		answered = true
		mb.setChecking(false)
		if err != nil {
			mb.SetInputError(err.Error())
			return
		}
		mb.finish(res)
	})
}

// setChecking puts the dialog in and out of its waiting state: the
// accepting button greys and stops responding while an asynchronous
// check is out, so the user can see that something is happening and a
// second press cannot start a second check.
func (mb *MessageBox) setChecking(v bool) {
	if mb == nil || mb.checking == v {
		return
	}
	mb.checking = v
	if mb.primary != nil {
		mb.primary.SetEnabled(!v && mb.acceptable())
	}
	if v {
		mb.SetInputError("")
	}
	if mb.overlay != nil {
		mb.overlay.Invalidate()
	}
}

// Checking reports whether an asynchronous check is out.
func (mb *MessageBox) Checking() bool { return mb != nil && mb.checking }

// Close finishes the dialog with res, exactly as pressing the matching
// button would: it dismisses, records the result and runs OnResult.
//
// It is how a dialog is closed by something other than the user — an
// answer that arrived, a vault that locked, a window shutting down.
// Dismissing the overlay directly takes the dialog off the screen
// without any of that, so OnResult never runs and the caller waiting on
// it waits forever.
//
// Closing a dialog that has already closed does nothing.
func (mb *MessageBox) Close(res MessageResult) {
	if mb == nil {
		return
	}
	mb.checking = false
	mb.finish(res)
}

// SetInputError shows a message under the input field and keeps the
// dialog up, or clears it when msg is empty. It is what
// [MessageBoxInput.Validate] uses, and what a caller whose check is a
// round trip calls when the answer arrives.
func (mb *MessageBox) SetInputError(msg string) {
	if mb == nil || mb.errLabel == nil {
		return
	}
	mb.errLabel.Text = msg
	mb.errLabel.SetVisible(msg != "")
	if mb.field != nil {
		mb.field.RequestFocus()
	}
	if mb.overlay != nil {
		mb.overlay.RequestLayout()
		mb.overlay.Invalidate()
	}
}

// InputError is the message currently shown under the field.
func (mb *MessageBox) InputError() string {
	if mb == nil || mb.errLabel == nil {
		return ""
	}
	return mb.errLabel.Text
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
	return PromptFor(from, title, label, MessageBoxInput{Text: initial}, on)
}

// PromptFor is [Prompt] with the field configured: a named accept button
// ("Rename", "Create"), a required value, a password field, or a
// Validate that refuses one without closing the dialog.
func PromptFor(from widget.Component, title, label string, in MessageBoxInput, on func(string, bool)) *MessageBox {
	var mb *MessageBox
	mb = NewMessageBox(MessageBoxOptions{
		Title: title, Message: label, Kind: MessageQuestion, Buttons: ButtonsOKCancel,
		Input: &in,
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
