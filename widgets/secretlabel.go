package widgets

import (
	"strings"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// SecretLabel shows a value that must not become a Go string: bullets
// until it is revealed, the characters themselves when it is.
//
// It is the read-only half of [SecretField] and holds the same rule. A
// vault's item view shows a password beside a "show" toggle, and a TOTP
// code that changes every thirty seconds; both are secrets, and a
// [Label] would keep every one of them alive as an unwipeable string for
// as long as the collector felt like it.
//
// The value comes from a function rather than a field, so the widget
// holds nothing between paints unless it is asked to: a locked vault
// answers nil and the label is empty on the next frame without anyone
// having to clear it.
//
// Masked — which is the usual state — it keeps no bytes at all. It takes
// the rune count from what Value hands it and lets the bytes go, because
// a row of bullets needs a number and not a secret. It holds the value
// only while [SecretLabel.Reveal] is set, and drops it again the first
// time it is asked for anything with Reveal off, when it leaves its
// window, and on [SecretLabel.Hide]. Hide stays, for a vault that locks
// while a value is on the screen.
type SecretLabel struct {
	widget.Base
	// Value is asked for the bytes to show. It is called on paint and
	// on measure, so it should be cheap; nil or an empty answer is an
	// empty label. The widget copies what it is given and wipes its
	// copy — the caller may wipe its own the moment it returns.
	Value func() []byte
	// Reveal paints the characters instead of bullets.
	Reveal bool
	// Mono paints in the monospaced face, which is what a generated
	// password and a TOTP code both want.
	Mono bool
	// Align places the text in the label's box, as [Label.Align] does.
	Align style.Align
	// Color overrides the text colour.
	Color paintengine2d.Color
	// MaskLen is how many bullets stand for the value while it is
	// hidden, whatever its real length. Zero means one per rune.
	//
	// One per rune tells anyone looking at the screen how long the
	// secret is, and the label is measured for the wider of the bullets
	// and the text, so the box says it too. For a password that is a
	// meaningful part of it. Set a fixed count — eight is the usual
	// choice — and the screen says only that there is something there.
	//
	// A fixed mask is measured for itself alone, so revealing a longer
	// value makes the label grow. That is the trade: a box that does not
	// move, or a box that does not leak.
	MaskLen int
	// Lines splits the value at newlines and draws one line each, for a
	// PEM block or an OpenSSH private key. Off by default: a password
	// and a TOTP code are one line, and a label that grew a second one
	// would move the row under the reader's hand.
	Lines bool

	// buf is the value, held only while it is being shown in the clear.
	// Masked, the label needs no bytes at all — see refresh.
	buf []byte
	// runes is how many runes the value had when it was last asked for,
	// and had is whether there was one. They are all a masked label
	// needs, and they say nothing about what the value was.
	runes int
	had   bool
}

// NewSecretLabel shows what value returns.
func NewSecretLabel(value func() []byte) *SecretLabel {
	l := &SecretLabel{Value: value, Mono: true}
	l.Init(l)
	return l
}

// Hide drops and zeroes whatever the label last drew from. An
// application calls it when the vault locks; the next paint asks Value
// again, which by then answers nothing.
func (l *SecretLabel) Hide() {
	l.wipe()
	l.runes, l.had = 0, false
	l.Invalidate()
}

// Len is how many runes are shown, and nothing about them — enough to
// lay a row out, or to say "32 characters".
func (l *SecretLabel) Len() int {
	l.refresh()
	return l.runes
}

// wipe zeroes the buffer and empties it, keeping the capacity so a label
// redrawn every second does not leave a trail of old values behind it.
func (l *SecretLabel) wipe() {
	WipeBytes(l.buf)
	l.buf = l.buf[:0]
}

// SetHost wipes the buffer when the label leaves its window, so a page
// thrown away without [SecretLabel.Hide] does not keep what it drew. A
// label being mounted keeps whatever it has: it is about to paint it.
func (l *SecretLabel) SetHost(h widget.Host) {
	if h == nil {
		l.wipe()
		l.runes, l.had = 0, false
	}
	l.Base.SetHost(h)
}

// refresh pulls the current value in, over the top of the last one.
//
// The old bytes are wiped rather than left for the allocator, and the
// buffer is reused where it can be: a TOTP code redrawn every second
// must not leave a trail of old codes in freed memory.
func (l *SecretLabel) refresh() {
	var now []byte
	if l.Value != nil {
		now = l.Value()
	}
	// Masked, the label draws a row of bullets: it needs to know how many
	// runes there were and whether there were any, and nothing else. It
	// used to copy the value in regardless, so an application showing a
	// vault item's fields masked held a second copy of every one of them
	// for as long as the item was open, and had to remember to call Hide
	// on each label on every way the item could go. The count is taken
	// from the caller's bytes without keeping them, and anything the
	// buffer still holds from a spell of being revealed is wiped now.
	l.runes, l.had = utf8.RuneCount(now), len(now) > 0
	if !l.Reveal {
		l.wipe()
		return
	}
	if len(now) > cap(l.buf) {
		WipeBytes(l.buf[:cap(l.buf)])
		l.buf = make([]byte, 0, roundUpCap(len(now)))
	}
	old := len(l.buf)
	l.buf = l.buf[:len(now)]
	copy(l.buf, now)
	if len(now) < old {
		WipeBytes(l.buf[len(now):old])
	}
}

func (l *SecretLabel) font() *style.Font {
	if l.Mono {
		return l.Look().MonoFont()
	}
	return l.Look().Font()
}

// mask is one bullet per rune — a string of nothing but bullets, so it
// can be measured and drawn like any other text while the value cannot.
func (l *SecretLabel) mask() string {
	if l.MaskLen > 0 {
		if !l.had {
			return ""
		}
		return strings.Repeat("•", l.MaskLen)
	}
	if l.runes > 0 {
		return strings.Repeat("•", l.runes)
	}
	return ""
}

// lines are the value's lines, or the whole of it as one.
func (l *SecretLabel) valueLines() [][]byte {
	if !l.Lines {
		return [][]byte{l.buf}
	}
	var out [][]byte
	start := 0
	for i := 0; i <= len(l.buf); i++ {
		if i == len(l.buf) || l.buf[i] == '\n' {
			line := l.buf[start:i]
			// A trailing carriage return is not part of the line.
			if n := len(line); n > 0 && line[n-1] == '\r' {
				line = line[:n-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	return out
}

func (l *SecretLabel) Measure(c layout.Constraints) paintengine2d.Point {
	l.refresh()
	f := l.font()
	lines := l.valueLines()
	var w float32
	for _, ln := range lines {
		w = max(w, f.AdvanceBytes(ln))
	}
	if !l.Reveal {
		if l.MaskLen > 0 {
			// A fixed mask is measured for itself alone. Taking the
			// wider of the two would put the value's own width on the
			// screen, which is the length this exists to hide.
			w = f.Advance(l.mask())
		} else {
			// Bullets are not the same width as the text they stand
			// for, so the label is measured for what it is *showing*. A
			// box that changed size when the user pressed "show" would
			// move the row under their hand, so it is the wider of the
			// two.
			w = max(w, f.Advance(l.mask()))
		}
	}
	h := f.Height()*float32(len(lines)) + 2
	if !l.Reveal && l.MaskLen > 0 {
		// Hidden, a fixed mask is one line however many the value has.
		h = f.Height() + 2
	}
	return c.Constrain(paintengine2d.Pt(w+2, h))
}

func (l *SecretLabel) Arrange(r paintengine2d.Rect) { l.SetBounds(r) }

func (l *SecretLabel) Paint(ctx *paintengine2d.Context) {
	l.refresh()
	f := l.font()
	b := l.LocalBounds()
	col := l.Color
	if col == (paintengine2d.Color{}) {
		col = l.Look().Palette().Text
	}
	if !l.Enabled() {
		col = l.Look().Palette().TextMuted
	}
	if !l.Reveal {
		m := l.mask()
		y := b.Min.Y + (b.Dy()-f.Height())*0.5
		f.Draw(ctx, m, paintengine2d.Pt(l.textX(b, f.Advance(m)), y), col)
		return
	}
	// Revealed. Straight from the buffer, through the path that caches
	// nothing ([style.Font.DrawBytes]) — Font.Draw would keep the value
	// in the face's shaped-run cache, keyed by a string of it, for as
	// long as the face lives.
	ctx.Save()
	ctx.ClipRect(b)
	lines := l.valueLines()
	y := b.Min.Y + (b.Dy()-f.Height()*float32(len(lines)))*0.5
	for _, ln := range lines {
		f.DrawBytes(ctx, ln, paintengine2d.Pt(l.textX(b, f.AdvanceBytes(ln)), y), col)
		y += f.Height()
	}
	ctx.Restore()
}

func (l *SecretLabel) textX(b paintengine2d.Rect, w float32) float32 {
	switch l.Align {
	case style.AlignCenter:
		return b.Min.X + (b.Dx()-w)*0.5
	case style.AlignEnd:
		return b.Max.X - 1 - w
	default:
		return b.Min.X + 1
	}
}
