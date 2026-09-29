package widgets

import (
	"strings"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// SecretField is a passphrase field whose contents never become a Go
// string.
//
// A [TextField] with Password set paints bullets, and that is all it
// does: the value is the exported string field Text, rebuilt on every
// keystroke, and Ctrl+C, Ctrl+X and a drag of the selection all carry
// the plaintext out. A Go string cannot be wiped — every copy lives
// until the collector reaches it and then lingers in freed memory — so a
// widget that holds a passphrase must not be built out of one, and must
// not share a code path with a widget that is.
//
// So this is a widget of its own rather than a mode of that one. Its
// contents live in a `[]byte` edited in place; when the buffer has to
// grow, the old array is zeroed before it is dropped; nothing in here
// converts it to a string, including the painting (see
// [style.Font.DrawBytes]), the accessibility tree and the input method.
//
// **Refused**: copy, cut, dragging the selection out, publishing it to
// the X11 PRIMARY selection, and middle-click paste into it. Every one
// of those is a way for a passphrase to leave the field by accident.
//
// **Allowed**: typing, Backspace and Delete, Ctrl+V from the clipboard,
// select-all then type over, and Home, End and the arrow keys.
//
// The input method is off while it has the focus, because
// [widget.IMETarget] is what asks for one and this deliberately does not
// implement it: an input method learns and suggests what was typed
// through it.
//
// What a caller gets out is [SecretField.Bytes], which is a copy the
// caller wipes, and [SecretField.Len], which is a rune count and no
// content — enough for a strength meter.
type SecretField struct {
	widget.Base
	Placeholder string
	// Reveal paints the characters instead of bullets: an application's
	// own "show" toggle, off by default. The revealed text is drawn
	// straight from the buffer and is still never a string.
	Reveal bool
	// Mono paints in the monospaced face, which is what a generated
	// passphrase wants.
	Mono bool
	// OnChange fires when the contents change. It is handed the field
	// rather than the value: a callback taking a string would be the
	// leak this widget exists to close.
	OnChange func(*SecretField)
	OnSubmit func(*SecretField)
	OnEscape func()

	buf        []byte // UTF-8, edited in place, wiped on growth and on Wipe
	caret      int    // byte offset, on a rune boundary
	selA, selB int    // byte offsets, unordered
	scrollX    float32
	blinkOn    bool
	dragging   bool
}

// NewSecretField builds an empty passphrase field.
func NewSecretField(placeholder string) *SecretField {
	f := &SecretField{Placeholder: placeholder, blinkOn: true}
	f.Init(f)
	f.SetWantsFocus(true)
	return f
}

// Bytes is a copy of the contents. The caller owns it and should wipe it
// ([SecretField.WipeBytes] is the same loop) when it is done.
func (f *SecretField) Bytes() []byte {
	out := make([]byte, len(f.buf))
	copy(out, f.buf)
	return out
}

// Len is how many runes are in the field, and nothing about them. It is
// what a strength meter and an "at least 12 characters" rule read.
func (f *SecretField) Len() int { return utf8.RuneCount(f.buf) }

// Empty reports whether anything has been typed.
func (f *SecretField) Empty() bool { return len(f.buf) == 0 }

// Wipe zeroes the field's buffer and empties it. A form calls it as soon
// as it has taken the passphrase, and again when it is dismissed.
func (f *SecretField) Wipe() {
	WipeBytes(f.buf)
	f.buf = f.buf[:0]
	f.caret, f.selA, f.selB, f.scrollX = 0, 0, 0, 0
	f.changed()
}

// WipeBytes zeroes b. It is here so a caller has the toolkit's own wipe
// for the copy [SecretField.Bytes] handed it, rather than writing the
// loop again and getting it subtly wrong.
func WipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func (f *SecretField) changed() {
	f.ensureCaretVisible()
	f.Invalidate()
	if f.OnChange != nil {
		f.OnChange(f)
	}
}

// ---- the buffer ---------------------------------------------------------
//
// Every edit goes through replace, and replace is the only thing that
// allocates. When the buffer has to grow it takes a new array, copies,
// and zeroes the old one before letting go of it — otherwise the
// passphrase would be left in whatever the allocator hands out next.

func (f *SecretField) replace(a, b int, ins []byte) {
	if a > b {
		a, b = b, a
	}
	a, b = clampByte(a, len(f.buf)), clampByte(b, len(f.buf))
	need := len(f.buf) - (b - a) + len(ins)
	if need > cap(f.buf) {
		grown := make([]byte, 0, roundUpCap(need))
		grown = append(grown, f.buf[:a]...)
		grown = append(grown, ins...)
		grown = append(grown, f.buf[b:]...)
		WipeBytes(f.buf[:cap(f.buf)])
		f.buf = grown
	} else {
		// It fits in the array that is already there, so nothing is
		// allocated and nothing is abandoned. The slice is lengthened
		// first where the edit grows it, because copy writes nothing
		// past the length however much room the array has.
		old := len(f.buf)
		tail := old - b
		if need > old {
			f.buf = f.buf[:need]
		}
		// copy is a memmove, so the tail may slide either way over
		// itself.
		copy(f.buf[a+len(ins):a+len(ins)+tail], f.buf[b:b+tail])
		copy(f.buf[a:a+len(ins)], ins)
		if need < old {
			// Whatever the shrink left past the end is still the old
			// secret, sitting in an array this field goes on using.
			WipeBytes(f.buf[need:old])
			f.buf = f.buf[:need]
		}
	}
	f.caret = a + len(ins)
	f.selA, f.selB = f.caret, f.caret
	f.changed()
}

func roundUpCap(n int) int {
	c := 64
	for c < n {
		c *= 2
	}
	return c
}

func clampByte(i, n int) int {
	if i < 0 {
		return 0
	}
	if i > n {
		return n
	}
	return i
}

func (f *SecretField) hasSel() bool { return f.selA != f.selB }

func (f *SecretField) selRange() (int, int) {
	a, b := f.selA, f.selB
	if a > b {
		a, b = b, a
	}
	return a, b
}

// prevRune and nextRune step the caret over whole runes, never into one.
func (f *SecretField) prevRune(i int) int {
	if i <= 0 {
		return 0
	}
	_, n := utf8.DecodeLastRune(f.buf[:i])
	return i - n
}

func (f *SecretField) nextRune(i int) int {
	if i >= len(f.buf) {
		return len(f.buf)
	}
	_, n := utf8.DecodeRune(f.buf[i:])
	return i + n
}

// ---- layout and painting ------------------------------------------------

func (f *SecretField) font() *style.Font {
	if f.Mono {
		return f.Look().MonoFont()
	}
	return f.Look().Font()
}

func (f *SecretField) fieldPad() float32 {
	p := f.Look().Metrics().FieldPad
	if p <= 0 {
		p = 8
	}
	return p
}

// mask is the bullets that stand in for the contents: one per rune, and
// a string made of nothing but bullets, so it can go anywhere a string
// goes — the engine's own field painter, the accessibility tree — while
// the contents cannot.
func (f *SecretField) mask() string {
	if n := f.Len(); n > 0 {
		return strings.Repeat("•", n)
	}
	return ""
}

// maskOffset maps a byte offset in the buffer to a rune index in the
// mask, which is what the engine's caret and selection are counted in.
func (f *SecretField) maskOffset(i int) int {
	return utf8.RuneCount(f.buf[:clampByte(i, len(f.buf))])
}

func (f *SecretField) Measure(c layout.Constraints) paintengine2d.Point {
	h := style.FieldHeight(f.Look().Metrics())
	return c.Constrain(paintengine2d.Pt(style.Dip(f.Look(), 180), h))
}

func (f *SecretField) Arrange(r paintengine2d.Rect) { f.SetBounds(r) }

func (f *SecretField) blink() bool {
	if !f.Focused() {
		return false
	}
	if h := f.Host(); h != nil {
		if b, ok := h.(interface{ CaretBlink() bool }); ok {
			return b.CaretBlink()
		}
	}
	return f.blinkOn
}

func (f *SecretField) Paint(ctx *paintengine2d.Context) {
	lk := f.Look()
	b := f.LocalBounds()
	st := f.State()
	if !f.Reveal {
		// The ordinary case, and the one that matters: the engine paints
		// the whole field — its well, its bevel, its capsule, the caret
		// and the selection — from a string of bullets. Every era is
		// exactly as right as it is for a password field today, and the
		// contents never reach any of that code.
		a, e := f.selRange()
		lk.DrawTextField(ctx, b, st, f.mask(), f.Placeholder,
			f.maskOffset(f.caret), f.maskOffset(a), f.maskOffset(e),
			f.blink(), f.scrollX, f.font())
		return
	}
	// Revealed. The frame is still the engine's, drawn with no text in
	// it; the text is drawn here, from the buffer, through the path that
	// caches nothing ([style.Font.DrawBytes]). It is not handed to
	// DrawTextField, because that would put the passphrase in the
	// face's shape cache for as long as the face lives.
	lk.DrawTextField(ctx, b, st, "", "", -1, 0, 0, false, 0, f.font())
	f.paintRevealed(ctx)
}

func (f *SecretField) paintRevealed(ctx *paintengine2d.Context) {
	lk := f.Look()
	p := lk.Palette()
	fc := f.font()
	b := f.LocalBounds()
	pad := f.fieldPad()
	inner := b.Inset(1)
	inner.Min.X += pad
	inner.Max.X -= pad
	if inner.Dx() <= 0 {
		return
	}
	y := inner.Min.Y + (inner.Dy()-fc.Height())*0.5
	x := inner.Min.X - f.scrollX

	ctx.Save()
	ctx.ClipRect(inner)
	if a, e := f.selRange(); a != e && f.Focused() {
		x0 := x + fc.CaretXBytes(f.buf, a)
		x1 := x + fc.CaretXBytes(f.buf, e)
		ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, fc.Height()),
			paintengine2d.Fill(p.Highlight))
	}
	fg := p.Text
	if !f.Enabled() {
		fg = p.TextMuted
	}
	fc.DrawBytes(ctx, f.buf, paintengine2d.Pt(x, y), fg)
	if f.blink() {
		cx := snapHalf(x + fc.CaretXBytes(f.buf, f.caret))
		ctx.DrawRect(paintengine2d.XYWH(cx, y, 1, fc.Height()), paintengine2d.Fill(p.Text))
	}
	ctx.Restore()
}

func snapHalf(v float32) float32 { return float32(int(v)) }

func (f *SecretField) ensureCaretVisible() {
	fc := f.font()
	inner := f.LocalBounds().Dx() - f.fieldPad()*2
	if inner <= 0 {
		return
	}
	cx := f.caretX()
	if cx-f.scrollX > inner-2 {
		f.scrollX = cx - inner + 2
	}
	if cx-f.scrollX < 0 {
		f.scrollX = cx
	}
	if f.scrollX < 0 {
		f.scrollX = 0
	}
	if w := f.runWidth(fc); w-f.scrollX < inner-2 {
		f.scrollX = max(0, w-inner+2)
	}
}

// caretX and runWidth measure in whichever of the two the field is
// showing, so the caret sits where the user sees it.
func (f *SecretField) caretX() float32 {
	if f.Reveal {
		return f.font().CaretXBytes(f.buf, f.caret)
	}
	return f.font().CaretX(f.mask(), f.maskOffset(f.caret))
}

func (f *SecretField) runWidth(fc *style.Font) float32 {
	if f.Reveal {
		return fc.AdvanceBytes(f.buf)
	}
	return fc.Advance(f.mask())
}

// offsetAt is the byte offset a click at local x lands on.
func (f *SecretField) offsetAt(x float32) int {
	fc := f.font()
	at := x - f.LocalBounds().Min.X - f.fieldPad() + f.scrollX
	if f.Reveal {
		return fc.IndexAtBytes(f.buf, at)
	}
	// Hidden: the bullets are what is on screen, so the hit test runs
	// over them and the rune index it gives is turned back into a byte
	// offset in the buffer.
	i := fc.IndexAt(f.mask(), at)
	off := 0
	for n := 0; n < i && off < len(f.buf); n++ {
		off = f.nextRune(off)
	}
	return off
}

// ---- input --------------------------------------------------------------

func (f *SecretField) MousePress(e widget.MouseEvent) bool {
	if !f.Enabled() {
		return false
	}
	// Middle click is the X11 PRIMARY paste, and PRIMARY is a selection
	// this field never publishes and never reads: a stray middle click
	// must not drop somebody else's clipboard into a passphrase, and the
	// gesture has no other meaning here.
	if e.Button == platform.ButtonMiddle {
		return true
	}
	f.MarkPointerFocus()
	f.RequestFocus()
	f.caret = f.offsetAt(e.Pos.X)
	f.selA, f.selB = f.caret, f.caret
	f.dragging = true
	f.ensureCaretVisible()
	f.Invalidate()
	return true
}

func (f *SecretField) MouseMove(e widget.MouseEvent) bool {
	if !f.dragging {
		return true
	}
	f.caret = f.offsetAt(e.Pos.X)
	f.selB = f.caret
	f.ensureCaretVisible()
	f.Invalidate()
	return true
}

func (f *SecretField) MouseRelease(widget.MouseEvent) bool {
	f.dragging = false
	// Note what does **not** happen here: a text field publishes its
	// selection to PRIMARY when a drag-select ends, which would put the
	// passphrase on the session's clipboard for any middle click to
	// paste. This one never does.
	return true
}

func (f *SecretField) Cursor() platform.Cursor { return platform.CursorText }

func (f *SecretField) TextInput(r rune) bool {
	if !f.Enabled() || r < 0x20 || r == 0x7f {
		return false
	}
	var tmp [4]byte
	n := utf8.EncodeRune(tmp[:], r)
	a, b := f.selRange()
	f.replace(a, b, tmp[:n])
	WipeBytes(tmp[:])
	return true
}

func (f *SecretField) KeyPress(e widget.KeyEvent) bool {
	if !f.Enabled() {
		return false
	}
	switch e.Key {
	case platform.KeyC, platform.KeyX:
		// Refused, and refused loudly enough to consume the key: a
		// passphrase does not go on the clipboard, and letting Ctrl+C
		// bubble would offer it to whatever is above this field.
		if e.Mods.Ctrl() {
			return true
		}
	case platform.KeyV:
		if e.Mods.Ctrl() {
			f.paste()
			return true
		}
	case platform.KeyA:
		if e.Mods.Ctrl() {
			f.selA, f.selB = 0, len(f.buf)
			f.caret = f.selB
			f.Invalidate()
			return true
		}
	case platform.KeyLeft:
		f.moveTo(f.prevRune(f.caret), e.Mods.Shift())
		return true
	case platform.KeyRight:
		f.moveTo(f.nextRune(f.caret), e.Mods.Shift())
		return true
	case platform.KeyHome:
		f.moveTo(0, e.Mods.Shift())
		return true
	case platform.KeyEnd:
		f.moveTo(len(f.buf), e.Mods.Shift())
		return true
	case platform.KeyBackspace:
		if f.hasSel() {
			a, b := f.selRange()
			f.replace(a, b, nil)
			return true
		}
		if f.caret > 0 {
			f.replace(f.prevRune(f.caret), f.caret, nil)
			return true
		}
		return false
	case platform.KeyDelete:
		if f.hasSel() {
			a, b := f.selRange()
			f.replace(a, b, nil)
			return true
		}
		if f.caret < len(f.buf) {
			f.replace(f.caret, f.nextRune(f.caret), nil)
			return true
		}
		return false
	case platform.KeyReturn:
		if f.OnSubmit != nil {
			f.OnSubmit(f)
			return true
		}
		return false
	case platform.KeyEscape:
		if f.OnEscape != nil {
			f.OnEscape()
			return true
		}
		return false
	}
	return false
}

func (f *SecretField) moveTo(i int, extend bool) {
	f.caret = clampByte(i, len(f.buf))
	if extend {
		f.selB = f.caret
	} else {
		f.selA, f.selB = f.caret, f.caret
	}
	f.ensureCaretVisible()
	f.Invalidate()
}

// paste takes the clipboard in as bytes. The string the platform hands
// back is the session's, not this field's — the toolkit did not make it
// and cannot wipe it — so what this can promise is that no *further*
// copy is kept: the bytes go into the buffer and the rest is dropped.
func (f *SecretField) paste() {
	s := platform.ClipboardGet()
	if s == "" {
		return
	}
	b := []byte(s)
	// A pasted passphrase is one line. Anything past a newline is the
	// next field's or the next line's, and silently taking it would make
	// a wrong passphrase that looks right.
	if i := indexNewline(b); i >= 0 {
		b = b[:i]
	}
	a, e := f.selRange()
	f.replace(a, e, b)
	WipeBytes(b)
}

func indexNewline(b []byte) int {
	for i, c := range b {
		if c == '\n' || c == '\r' {
			return i
		}
	}
	return -1
}

// SetCaretBlink is the host's blink phase, as a text field takes it.
func (f *SecretField) SetCaretBlink(on bool) { f.blinkOn = on }

// Selection is the byte range the user has selected, for a test or a
// caller that needs to know there is one. It is offsets, not content.
func (f *SecretField) Selection() (int, int) { return f.selRange() }

// Caret is the insertion point, as a byte offset.
func (f *SecretField) Caret() int { return f.caret }
