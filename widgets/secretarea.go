package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// SecretArea is [SecretField] over several lines: a PEM block, an
// OpenSSH private key, a recovery phrase.
//
// It is the same widget with a second dimension rather than a second
// implementation, and that is the point. Everything that makes a secret
// field a secret field — the byte buffer edited in place, the wipe when
// it is outgrown or shortened, the refusal of copy, cut, drag-out and
// the PRIMARY selection, the input method staying off, the tests that
// read the source for a `string(` conversion — belongs to the embedded
// [SecretField] and is not written twice here. A second implementation
// of secret handling is the last thing this toolkit should have.
//
// What is added: newlines are kept rather than dropped (typed, and
// pasted), the caret moves by line as well as by rune, and the value is
// drawn as one masked line per line of text.
//
// A multi-line secret is usually pasted rather than typed, and usually
// long, so it wraps nothing: a line too wide to show scrolls sideways,
// which keeps a key's structure — its header, its base64 body, its
// footer — the shape the user recognises.
type SecretArea struct {
	SecretField
	// Rows is how many lines of the value are shown at once, which is
	// what the widget measures to. Zero means [DefaultSecretAreaRows].
	Rows int

	scrollY float32
	// goalX is the x the caret is aiming for while it moves up and down,
	// so a run through a short line does not drag it in — the behaviour
	// every text editor has.
	goalX    float32
	goalLive bool
}

// DefaultSecretAreaRows is how tall a secret area is when Rows is unset:
// enough for a PEM block's beginning, some body and its end to be on
// screen together.
const DefaultSecretAreaRows = 6

// NewSecretArea builds an empty multi-line passphrase field.
func NewSecretArea(placeholder string) *SecretArea {
	a := &SecretArea{}
	a.Placeholder = placeholder
	a.blinkOn = true
	a.CapsHint = true
	a.Init(a)
	a.SetWantsFocus(true)
	return a
}

func (a *SecretArea) rows() int {
	if a.Rows > 0 {
		return a.Rows
	}
	return DefaultSecretAreaRows
}

// lineStarts are the byte offsets each line begins at, the first being 0.
// A trailing newline starts a last, empty line, which is where the caret
// goes after it — so the count is one more than the newlines.
func (a *SecretArea) lineStarts() []int {
	starts := []int{0}
	for i, c := range a.buf {
		if c == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// lineAt is the index of the line holding byte offset off, and that
// line's start and end (the end excluding the newline).
func (a *SecretArea) lineAt(off int) (line, start, end int) {
	starts := a.lineStarts()
	line = len(starts) - 1
	for i, s := range starts {
		if off < s {
			line = i - 1
			break
		}
	}
	if line < 0 {
		line = 0
	}
	start = starts[line]
	end = len(a.buf)
	if line+1 < len(starts) {
		end = starts[line+1] - 1
	}
	return line, start, end
}

func (a *SecretArea) lineHeight() float32 { return a.font().Height() }

func (a *SecretArea) Measure(c layout.Constraints) paintengine2d.Point {
	lk := a.Look()
	pad := a.fieldPad()
	h := a.lineHeight()*float32(a.rows()) + pad*2
	return c.Constrain(paintengine2d.Pt(style.Dip(lk, 280), h))
}

func (a *SecretArea) Arrange(r paintengine2d.Rect) { a.SetBounds(r) }

// maskOfLine is the bullets standing in for one line of the buffer.
func (a *SecretArea) maskOfLine(start, end int) string {
	n := 0
	for i := start; i < end; {
		i = a.nextRune(i)
		n++
	}
	return bullets(n)
}

func (a *SecretArea) Paint(ctx *paintengine2d.Context) {
	lk := a.Look()
	b := a.LocalBounds()
	// The engine's own field chrome with nothing in it — its well, its
	// bevel, whatever this era draws — and the value painted here, so no
	// part of it reaches the engine's text path or its shape cache.
	lk.DrawTextField(ctx, b, a.State(), "", "", -1, 0, 0, false, 0, a.font())

	pad := a.fieldPad()
	inner := b.Inset(1)
	inner.Min.X += pad
	inner.Max.X -= pad
	inner.Min.Y += pad
	inner.Max.Y -= pad
	if inner.Dx() <= 0 || inner.Dy() <= 0 {
		return
	}
	ctx.Save()
	ctx.ClipRect(inner)
	defer ctx.Restore()

	p := lk.Palette()
	fc := a.font()
	lh := a.lineHeight()
	starts := a.lineStarts()

	if len(a.buf) == 0 && a.Placeholder != "" && !a.Focused() {
		fc.Draw(ctx, a.Placeholder, paintengine2d.Pt(inner.Min.X, inner.Min.Y), p.TextMuted)
		a.paintCapsHint(ctx)
		return
	}

	selA, selB := a.selRange()
	for i, start := range starts {
		end := len(a.buf)
		if i+1 < len(starts) {
			end = starts[i+1] - 1
		}
		y := inner.Min.Y + float32(i)*lh - a.scrollY
		if y+lh < inner.Min.Y || y > inner.Max.Y {
			continue
		}
		x := inner.Min.X - a.scrollX
		// The selection, under the text, in whichever of the two the
		// area is showing.
		if selA != selB && selB > start && selA <= end {
			s, e := max(selA, start), min(selB, end)
			x0 := x + a.xOf(fc, start, s)
			x1 := x + a.xOf(fc, start, e)
			if e == end && selB > end {
				// The newline is selected too: show it as a sliver, the
				// way every editor marks a selected line break.
				x1 += fc.Advance(" ")
			}
			ctx.DrawRect(paintengine2d.XYWH(x0, y, max(x1-x0, 0), lh),
				paintengine2d.Fill(p.Selection))
		}
		if a.Reveal {
			// Straight from the buffer, through the path that caches
			// nothing and makes no string of it.
			fc.DrawBytes(ctx, a.buf[start:end], paintengine2d.Pt(x, y), p.Text)
		} else {
			fc.Draw(ctx, a.maskOfLine(start, end), paintengine2d.Pt(x, y), p.Text)
		}
	}

	if a.blink() {
		line, start, _ := a.lineAt(a.caret)
		cx := inner.Min.X - a.scrollX + a.xOf(fc, start, a.caret)
		cy := inner.Min.Y + float32(line)*lh - a.scrollY
		ctx.DrawRect(paintengine2d.XYWH(cx, cy, max(style.Dip(lk, 1), 1), lh),
			paintengine2d.Fill(p.Text))
	}
	a.paintCapsHint(ctx)
}

// xOf is the x of byte offset off measured from the start of its line,
// in whichever of the mask and the value is on screen.
func (a *SecretArea) xOf(fc *style.Font, start, off int) float32 {
	if off <= start {
		return 0
	}
	if a.Reveal {
		return fc.AdvanceBytes(a.buf[start:off])
	}
	return fc.Advance(a.maskOfLine(start, off))
}

// offsetAtPoint is the byte offset a click at local p lands on.
func (a *SecretArea) offsetAtPoint(pt paintengine2d.Point) int {
	fc := a.font()
	pad := a.fieldPad()
	b := a.LocalBounds()
	lh := a.lineHeight()
	starts := a.lineStarts()

	row := int((pt.Y - b.Min.Y - pad - 1 + a.scrollY) / lh)
	if row < 0 {
		row = 0
	}
	if row >= len(starts) {
		row = len(starts) - 1
	}
	start := starts[row]
	end := len(a.buf)
	if row+1 < len(starts) {
		end = starts[row+1] - 1
	}
	at := pt.X - b.Min.X - pad - 1 + a.scrollX
	if a.Reveal {
		return start + fc.IndexAtBytes(a.buf[start:end], at)
	}
	n := fc.IndexAt(a.maskOfLine(start, end), at)
	off := start
	for k := 0; k < n && off < end; k++ {
		off = a.nextRune(off)
	}
	return off
}

func (a *SecretArea) MousePress(e widget.MouseEvent) bool {
	if !a.Enabled() {
		return false
	}
	a.RequestFocus()
	a.caret = a.offsetAtPoint(e.Pos)
	a.selA, a.selB = a.caret, a.caret
	a.dragging = true
	a.goalLive = false
	a.ensureCaretShown()
	a.Invalidate()
	return true
}

func (a *SecretArea) MouseMove(e widget.MouseEvent) bool {
	if !a.dragging {
		return false
	}
	a.caret = a.offsetAtPoint(e.Pos)
	a.selB = a.caret
	a.ensureCaretShown()
	a.Invalidate()
	return true
}

func (a *SecretArea) MouseRelease(widget.MouseEvent) bool {
	a.dragging = false
	return true
}

func (a *SecretArea) TextInput(r rune) bool {
	a.goalLive = false
	return a.SecretField.TextInput(r)
}

func (a *SecretArea) KeyPress(e widget.KeyEvent) bool {
	if !a.Enabled() {
		return false
	}
	switch e.Key {
	case platform.KeyReturn:
		// A newline is content here, not a submit. OnSubmit is still the
		// way out for a caller that wants one, on Ctrl+Return.
		if e.Mods.Ctrl() {
			if a.OnSubmit != nil {
				a.OnSubmit(&a.SecretField)
				return true
			}
			return false
		}
		s, t := a.selRange()
		a.replace(s, t, []byte{'\n'})
		a.goalLive = false
		a.ensureCaretShown()
		return true
	case platform.KeyUp, platform.KeyDown:
		a.moveByLine(e.Key == platform.KeyDown, e.Mods.Shift())
		return true
	case platform.KeyHome:
		_, start, _ := a.lineAt(a.caret)
		if e.Mods.Ctrl() {
			start = 0
		}
		a.moveTo(start, e.Mods.Shift())
		a.goalLive = false
		a.ensureCaretShown()
		return true
	case platform.KeyEnd:
		_, _, end := a.lineAt(a.caret)
		if e.Mods.Ctrl() {
			end = len(a.buf)
		}
		a.moveTo(end, e.Mods.Shift())
		a.goalLive = false
		a.ensureCaretShown()
		return true
	}
	handled := a.SecretField.KeyPress(e)
	if handled {
		a.goalLive = false
		a.ensureCaretShown()
	}
	return handled
}

// moveByLine moves the caret one line down or up, keeping the x it was
// aiming for so a run through a short line does not drag it in.
func (a *SecretArea) moveByLine(down, extend bool) {
	fc := a.font()
	line, start, _ := a.lineAt(a.caret)
	if !a.goalLive {
		a.goalX = a.xOf(fc, start, a.caret)
		a.goalLive = true
	}
	starts := a.lineStarts()
	to := line - 1
	if down {
		to = line + 1
	}
	if to < 0 || to >= len(starts) {
		return
	}
	ns := starts[to]
	ne := len(a.buf)
	if to+1 < len(starts) {
		ne = starts[to+1] - 1
	}
	off := ns
	if a.Reveal {
		off = ns + fc.IndexAtBytes(a.buf[ns:ne], a.goalX)
	} else {
		n := fc.IndexAt(a.maskOfLine(ns, ne), a.goalX)
		for k := 0; k < n && off < ne; k++ {
			off = a.nextRune(off)
		}
	}
	goal, live := a.goalX, a.goalLive
	a.moveTo(off, extend)
	a.goalX, a.goalLive = goal, live
	a.ensureCaretShown()
}

// ensureCaretShown scrolls in both directions so the caret is inside the
// visible box.
func (a *SecretArea) ensureCaretShown() {
	b := a.LocalBounds()
	if b.Empty() {
		return
	}
	pad := a.fieldPad()
	viewH := b.Dy() - pad*2 - 2
	viewW := b.Dx() - pad*2 - 2
	lh := a.lineHeight()

	line, start, _ := a.lineAt(a.caret)
	top := float32(line) * lh
	if top < a.scrollY {
		a.scrollY = top
	}
	if bottom := top + lh; bottom > a.scrollY+viewH {
		a.scrollY = bottom - viewH
	}
	if lines := float32(len(a.lineStarts())) * lh; a.scrollY > lines-viewH {
		a.scrollY = max(lines-viewH, 0)
	}
	if a.scrollY < 0 {
		a.scrollY = 0
	}

	cx := a.xOf(a.font(), start, a.caret)
	if cx < a.scrollX {
		a.scrollX = cx
	}
	if cx > a.scrollX+viewW {
		a.scrollX = cx - viewW
	}
	if a.scrollX < 0 {
		a.scrollX = 0
	}
}

// Paste keeps the newlines: a key is several lines and taking only the
// first would make a value that looks right and is not. The one-line
// field truncates at the first newline for exactly the same reason, and
// the two are the same rule applied to different shapes.
func (a *SecretArea) paste() {
	b := platform.ClipboardGetSecret()
	if len(b) == 0 {
		return
	}
	defer WipeBytes(b)
	s, e := a.selRange()
	a.replace(s, e, normalizeNewlines(b))
	a.goalLive = false
	a.ensureCaretShown()
}

// normalizeNewlines turns CRLF and lone CR into LF in place, returning
// the shortened slice. A key pasted from a Windows editor must not come
// out with a stray CR at the end of every line, and the buffer is edited
// rather than copied so no second array holds the secret.
func normalizeNewlines(b []byte) []byte {
	out := 0
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c == '\r' {
			if i+1 < len(b) && b[i+1] == '\n' {
				continue // the \n behind it is the newline
			}
			c = '\n'
		}
		b[out] = c
		out++
	}
	// Whatever is past the new end held bytes of the secret.
	WipeBytes(b[out:])
	return b[:out]
}

// Lines is how many lines the value has, and nothing about them — the
// counterpart of [SecretField.Len] for a key's shape.
func (a *SecretArea) Lines() int { return len(a.lineStarts()) }
