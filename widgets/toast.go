package widgets

import (
	"strings"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ToastKind is what a note is about, which decides its mark and the ink
// of its edge. The words are the application's; this is only how loudly
// they are shown.
type ToastKind int

const (
	// ToastInfo is the plain note: Copied, Unlocked, Saved.
	ToastInfo ToastKind = iota
	// ToastSuccess is something that finished as it should.
	ToastSuccess
	// ToastWarning is something that worked with a reservation.
	ToastWarning
	// ToastError is something that did not work.
	ToastError
)

// ToastDuration is how long a note stays when the caller does not say:
// long enough to read a short sentence twice, which is what GTK, Android
// and macOS all land within a second of.
const ToastDuration = 4 * time.Second

// toastFade is how long a note takes to appear and to go. It is short
// enough not to delay the reading and long enough not to flash.
const (
	toastFadeIn  = 110 * time.Millisecond
	toastFadeOut = 180 * time.Millisecond
	toastFrame   = 16 * time.Millisecond
)

// Toast is a passing note: a line of text over the window's content that
// says what just happened and then goes.
//
// It is what a window has instead of a status bar, which never goes and
// takes a row of the layout for ever. A note is over the content, out of
// the pointer's way — nothing in it can be clicked, and a press goes
// through it to whatever is underneath — and assistive technology is told
// what it says as a polite announcement ([widget.Announce]), because
// nothing moved the focus and a screen reader would otherwise never hear
// of it.
//
// [ShowToast] is how one is shown. A Toast made by hand is an ordinary
// widget: it paints a note wherever it is put, and nothing fades or
// removes it.
type Toast struct {
	widget.Base
	// Text is what the note says. It is also what assistive technology
	// is told, so it is a sentence a person would want read.
	Text string
	Kind ToastKind
	// Timeout is how long the note stays once it is up, not counting the
	// fade. Zero keeps it until something takes it down, which is what a
	// note about something still happening wants ("Waiting for the
	// request…").
	Timeout time.Duration
	// Mark is the mark at the note's leading edge. It starts as the one
	// the kind asks for; IconNone draws none.
	Mark style.ToolIcon

	lines   []string
	lineFor string
	lineBox float32

	alpha       float32
	going       bool
	stopTimeout func()
	stopFade    func()
	done        func(*Toast)
}

// NewToast is a note that says text. Nothing shows or removes it; see
// [ShowToast].
func NewToast(text string, kind ToastKind) *Toast {
	t := &Toast{Text: text, Kind: kind, Timeout: ToastDuration, Mark: toastMark(kind), alpha: 1}
	t.Init(t)
	return t
}

// toastMark is the mark a kind wears.
func toastMark(kind ToastKind) style.ToolIcon {
	switch kind {
	case ToastSuccess:
		return style.IconCheck
	case ToastWarning:
		return style.IconWarning
	case ToastError:
		return style.IconError
	}
	return style.IconInfo
}

// ShowToast puts a note over the window from is in and returns it.
//
// The note appears, stays for d, and goes; d of zero is [ToastDuration],
// and a negative d keeps it up until [Toast.Dismiss] (or
// [DismissToasts]), which is what a note about something still happening
// wants. Several notes stack, newest nearest the edge.
//
// It returns nil when there is no window to put a note on, so a widget
// that is not mounted says nothing rather than panicking.
func ShowToast(from widget.Component, text string, kind ToastKind, d time.Duration) *Toast {
	if from == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	stack := toastStackFor(from)
	if stack == nil {
		return nil
	}
	t := NewToast(text, kind)
	switch {
	case d > 0:
		t.Timeout = d
	case d < 0:
		t.Timeout = 0
	}
	stack.add(t)
	// The window hears about it as a polite announcement: nothing moved
	// the focus, so this is the only way it is spoken.
	widget.Announce(from, text)
	return t
}

// DismissToasts takes down every note on from's window at once.
func DismissToasts(from widget.Component) {
	if s, ok := widget.ToastLayerOf(from).(*toastStack); ok {
		s.clear()
	}
}

// Toasts are the notes up on from's window, oldest first.
func Toasts(from widget.Component) []*Toast {
	s, ok := widget.ToastLayerOf(from).(*toastStack)
	if !ok {
		return nil
	}
	return s.notes()
}

// Dismiss fades the note out and takes it down. It is safe to call twice
// and safe on a note that is already going.
func (t *Toast) Dismiss() {
	if t == nil || t.going {
		return
	}
	t.going = true
	t.stop(&t.stopTimeout)
	t.stop(&t.stopFade)
	t.fade(0, toastFadeOut, t.finish)
}

// finish removes the note from the layer.
func (t *Toast) finish() {
	t.alpha = 0
	if t.done != nil {
		done := t.done
		t.done = nil
		done(t)
	}
}

// Opacity is how far the note has faded in, from 0 to 1. It is 1 for a
// note at rest, and under reduced motion it is only ever 0 or 1.
func (t *Toast) Opacity() float32 { return t.alpha }

// appear starts the note: it fades in, then waits out its timeout.
func (t *Toast) appear() {
	if !style.Animations() {
		t.alpha = 1
		t.arm()
		return
	}
	t.alpha = 0
	t.fade(1, toastFadeIn, t.arm)
}

// arm sets the note's own clock going, where it has one.
func (t *Toast) arm() {
	t.alpha = 1
	t.Invalidate()
	if t.Timeout <= 0 {
		return
	}
	t.stopTimeout = widget.After(t, t.Timeout, t.Dismiss)
}

// stop cancels one of the note's timers, if it is running.
func (t *Toast) stop(slot *func()) {
	if *slot != nil {
		(*slot)()
		*slot = nil
	}
}

// fade walks the note's opacity to a over d and then calls then.
//
// It counts steps rather than reading a clock. The steps are the window's
// own timers, so a note fades at the rate the window runs, and a test
// that moves the window's clock moves the fade with it; a second clock
// would have to be kept in agreement with the first for no gain. Each
// step asks for the one after it, so a note taken down — or a window
// closed — stops walking.
func (t *Toast) fade(to float32, d time.Duration, then func()) {
	arrive := func() {
		t.alpha = to
		t.Invalidate()
		if then != nil {
			then()
		}
	}
	if d <= 0 || !style.Animations() {
		arrive()
		return
	}
	steps := max(float32(d)/float32(toastFrame), 1)
	delta := (to - t.alpha) / steps
	var walk func()
	walk = func() {
		t.alpha += delta
		if (delta >= 0 && t.alpha >= to) || (delta <= 0 && t.alpha <= to) {
			arrive()
			return
		}
		t.Invalidate()
		t.stopFade = widget.After(t, toastFrame, walk)
	}
	t.stopFade = widget.After(t, toastFrame, walk)
}

// toastStyle is the measure a note's text is laid out by: the look's
// tooltip measure, which is the one surface every pack already has for a
// line of text over the window.
func (t *Toast) toastStyle() style.TooltipStyle { return style.TooltipStyleOf(t.Look()) }

// markSide is the room the mark takes, mark and gap, or zero for none.
func (t *Toast) markSide() (side, gap float32) {
	if t.Mark == style.IconNone {
		return 0, 0
	}
	lk := t.Look()
	_, side, gap = style.ToolButtonChromeFor(lk, style.Dip(lk, 24))
	return side, gap
}

func (t *Toast) wrap(maxW float32) paintengine2d.Point {
	ts := t.toastStyle()
	side, gap := t.markSide()
	lines, sz := ts.Wrap(t.Text, maxW-side-gap)
	t.lines, t.lineFor, t.lineBox = lines, t.Text, sz.X
	sz.X += side + gap
	if min := style.Dip(t.Look(), 28); sz.Y < min {
		sz.Y = min
	}
	return sz
}

func (t *Toast) Measure(c layout.Constraints) paintengine2d.Point {
	maxW := float32(0)
	if c.HasMaxW() {
		maxW = c.MaxW
	}
	return c.Constrain(t.wrap(maxW))
}

func (t *Toast) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

// HitTest is nil: a note is read, not used. Every press goes through it
// to the content underneath, which is what keeps it out of the way.
func (t *Toast) HitTest(paintengine2d.Point) widget.Component { return nil }

func (t *Toast) Paint(ctx *paintengine2d.Context) {
	b := t.LocalBounds()
	if b.Empty() || t.alpha <= 0 {
		return
	}
	if t.lines == nil || t.lineFor != t.Text || b.Dx() < t.lineBox-0.5 {
		t.wrap(b.Dx())
	}
	// The whole note fades as one piece: a layer, not per-draw alpha, so
	// the mark over the ground does not show through it.
	ctx.DrawLayer(b, t.alpha, func(ctx *paintengine2d.Context) { t.paintFace(ctx, b) })
}

func (t *Toast) paintFace(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	lk := t.Look()
	side, gap := t.markSide()
	text := b
	text.Min.X += side + gap
	// The look's own note surface, with its own text in it: a pack that
	// draws a bubble draws this one too.
	lk.DrawTooltip(ctx, b, strings.Join(t.lines, "\n"))
	if side > 0 {
		pad := t.toastStyle().Pad
		box := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+(b.Dy()-side)*0.5, side, side)
		style.DrawToolIcon(ctx, box, t.Mark, t.markInk(), style.IconSetOf(lk))
	}
	if t.Kind != ToastInfo {
		// A strip of the kind's own colour down the leading edge. It is
		// the one thing a tooltip surface does not say, and a note that
		// says something went wrong has to say so without the words.
		w := max(style.Dip(lk, 3), 2)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, w, b.Dy()), paintengine2d.Fill(t.markInk()))
	}
}

// markInk is the kind's colour, from the palette the look already has for
// saying these four things.
func (t *Toast) markInk() paintengine2d.Color {
	p := t.Look().Palette()
	switch t.Kind {
	case ToastSuccess:
		return p.Success
	case ToastWarning:
		return p.Warning
	case ToastError:
		return p.Danger
	}
	return p.Accent
}

// Describe is the note as assistive technology sees it: a status bar,
// named by what it says, which is the role a window's "what just
// happened" has had since Windows 95. It is announced when it appears
// ([widget.Announce]) as well; this is what a reader finds if it walks
// the window while the note is still up.
func (t *Toast) Describe(n *a11y.Node) {
	n.Role = a11y.RoleStatusBar
	n.Name = t.Text
}
