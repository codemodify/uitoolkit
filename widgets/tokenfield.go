package widgets

import (
	"math"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// DefaultTokenSeparators are the characters that end a token as it is
// typed. Return always does, whatever this says.
const DefaultTokenSeparators = ",;"

// defaultTokenFieldWidth is the 1x design width a chip field asks for
// when it is measured without one — a text field's width, since that is
// what it stands in for.
const defaultTokenFieldWidth = 320

// Token is one chip in a [TokenField]: a label and, unless Removable is
// off, a cross that takes it out.
//
// It is a widget rather than a rectangle the field draws, so that it has
// its own box for hit testing, its own hover, and its own place in the
// accessibility tree — a screen reader reads the recipients of a mail as
// items, which is what they are.
type Token struct {
	widget.Base
	Text string
	// Color tints the chip. The look's accent when it is not set.
	Color paintengine2d.Color
	// Removable draws the cross and answers clicks on it. A field of
	// tokens the user may not take out — a read-only summary — turns it
	// off, and then the chip is a label in a capsule.
	Removable bool
	OnRemove  func()
	// OnPress is the click anywhere but the cross. The field uses it to
	// put the keyboard back in its editor.
	OnPress func()
	hover   bool
	crossIn bool
	down    bool
}

// NewToken makes a removable chip.
func NewToken(text string) *Token {
	t := &Token{Text: text, Removable: true}
	t.Init(t)
	return t
}

func (t *Token) pad() float32 { return style.Dip(t.Look(), 8) }
func (t *Token) crossW() float32 {
	if !t.Removable {
		return 0
	}
	return style.Dip(t.Look(), 16)
}

func (t *Token) Measure(c layout.Constraints) paintengine2d.Point {
	lk := t.Look()
	f := lk.Font()
	h := f.Height() + style.Dip(lk, 6)
	if min := style.Dip(lk, 20); h < min {
		h = min
	}
	// Whole pixels, rounded up. A component's bounds are snapped to the
	// pixel grid by SetBounds, so a chip that asks for a fractional width
	// can be handed back up to a pixel less than it asked for — and Paint
	// answers a box a hair too small by eliding the text. Asking for the
	// pixel above means the box is never narrower than the label needs,
	// and a chip stops cutting its own text short.
	w := t.pad()*2 + f.Advance(t.Text) + t.crossW()
	return c.Constrain(paintengine2d.Pt(ceilPx(w), h))
}

func (t *Token) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

// crossRect is the cross's box, at the chip's trailing end.
func (t *Token) crossRect() paintengine2d.Rect {
	if !t.Removable {
		return paintengine2d.Rect{}
	}
	b := t.LocalBounds()
	w := t.crossW()
	return paintengine2d.XYWH(b.Max.X-w-style.Dip(t.Look(), 2), b.Min.Y, w, b.Dy())
}

// tokenRadius is a chip's corner radius: a capsule, unless the look's
// own corner radius is smaller, so a square-cornered era gets square
// chips rather than a shape it never drew.
//
// Zero is a radius, and the commonest one here — 41 of the packs draw
// square corners. Testing for a radius above zero read every one of them
// as "no opinion" and gave them the capsule, which is the exact shape
// the test was written to keep out of them.
func tokenRadius(height, lookRadius float32) float32 {
	rad := height * 0.5
	if lookRadius >= 0 && lookRadius < rad {
		return lookRadius
	}
	return rad
}

func (t *Token) Paint(ctx *paintengine2d.Context) {
	lk := t.Look()
	p := lk.Palette()
	b := t.LocalBounds()
	col := t.Color
	if col == (paintengine2d.Color{}) {
		col = p.Accent
	}
	rad := tokenRadius(b.Dy(), lk.Metrics().Radius)
	alpha := float32(0.18)
	if t.hover {
		alpha = 0.26
	}
	ctx.DrawRoundRect(b, rad, rad, paintengine2d.Fill(col.WithAlpha(alpha)))
	ctx.DrawRoundRect(b.Inset(0.5), rad, rad, paintengine2d.StrokePaint(col.WithAlpha(0.55), 1))

	f := lk.Font()
	room := b.Dx() - t.pad()*2 - t.crossW()
	label := t.Text
	if f.Advance(label) > room {
		label = f.Fit(label, room)
	}
	fg := p.Text
	if !t.Enabled() {
		fg = p.TextMuted
	}
	f.Draw(ctx, label, paintengine2d.Pt(b.Min.X+t.pad(), b.Min.Y+(b.Dy()-f.Height())*0.5), fg)

	if t.Removable {
		cc := p.TextMuted
		if t.crossIn || t.down {
			cc = p.Text
		}
		box := t.crossRect()
		if t.down {
			box = box.Inset(style.Dip(lk, 1))
		}
		style.DrawCaptionGlyph(ctx, box, style.CaptionClose, false,
			cc, style.Dip(lk, 7), max(style.Dip(lk, 1.25), 1))
	}
}

func (t *Token) MouseMove(e widget.MouseEvent) bool {
	in := t.crossRect().Contains(e.Pos)
	if !t.hover || in != t.crossIn {
		t.hover, t.crossIn = true, in
		t.Invalidate()
	}
	return true
}

func (t *Token) MouseExit() {
	t.hover, t.crossIn, t.down = false, false, false
	t.Invalidate()
	t.Base.MouseExit()
}

func (t *Token) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.MarkPointerFocus()
	t.down = t.Removable && t.crossRect().Contains(e.Pos)
	if !t.down && t.OnPress != nil {
		t.OnPress()
	}
	t.Invalidate()
	return true
}

// MouseRelease removes the chip, on the button's contract: only if the
// release lands on the cross the press did.
func (t *Token) MouseRelease(e widget.MouseEvent) bool {
	was := t.down
	t.down = false
	t.Invalidate()
	if was && t.crossRect().Contains(e.Pos) && t.OnRemove != nil {
		t.OnRemove()
	}
	return true
}

// TokenField is a field of removable chips with a text editor at the end:
// the recipient field of every mail client written since Gmail, and the
// tag field of everything that has tags.
//
// What it is for is a list of short values the user builds by typing. A
// comma, a semicolon or Return ends one and starts the next; Backspace in
// an empty editor takes the last one back **into the editor**, so a
// mistyped address can be corrected rather than retyped; and leaving the
// field commits what is in the editor, because a typed address that
// vanished because nobody pressed comma is the bug this widget exists to
// stop.
type TokenField struct {
	widget.Base
	// Placeholder shows in the editor while the field is empty.
	Placeholder string
	// Separators are the characters that end a token as they are typed or
	// pasted. Empty means [DefaultTokenSeparators]; Return always ends one.
	Separators string
	// Accept vets a token before it goes in. Returning false drops it and
	// leaves the text in the editor, where the user can see what was
	// wrong with it.
	Accept func(string) bool
	// Unique drops a token the field already holds, compared exactly.
	Unique bool
	// Removable is the default for the chips this field makes.
	Removable bool
	// PreferredWidth is the 1x design width the field asks for when it is
	// measured without one, in place of [defaultTokenFieldWidth]. The
	// field wraps, so this is where it folds rather than a width it is
	// held to: given a width, it uses that one.
	PreferredWidth float32
	// Color tints every chip. Per-chip colours are set on the [Token].
	Color paintengine2d.Color
	// OnChange fires whenever the set of tokens changes, from typing, a
	// removed chip or SetTokens.
	OnChange func([]string)
	// OnSubmit is Return pressed with nothing pending in the editor — the
	// "I am done with this field" key.
	OnSubmit func()

	tokens    []*Token
	editor    *TextField
	splitting bool
}

// NewTokenField builds an empty chip field. on is OnChange.
func NewTokenField(placeholder string, on func([]string)) *TokenField {
	f := &TokenField{Placeholder: placeholder, Removable: true, OnChange: on}
	f.Init(f)
	f.editor = NewTextField("", placeholder, nil)
	f.editor.Frameless = true
	f.editor.OnInput = func(s string) { f.splitInput(s) }
	f.editor.OnSubmit = func(s string) {
		if strings.TrimSpace(s) != "" {
			if f.add(strings.TrimSpace(s)) {
				f.editor.SetText("")
			}
			return
		}
		if f.OnSubmit != nil {
			f.OnSubmit()
		}
	}
	// Leaving commits. Every mail client does this, and the ones that do
	// not lose addresses.
	f.editor.OnFocusLost = func() { f.Commit() }
	f.Base.Add(f.editor)
	return f
}

// Editor is the text field at the end of the chips.
func (f *TokenField) Editor() *TextField { return f.editor }

// Pending is what is typed and not yet a token.
func (f *TokenField) Pending() string { return f.editor.Text }

// Tokens is the committed values, in order.
func (f *TokenField) Tokens() []string {
	out := make([]string, len(f.tokens))
	for i, t := range f.tokens {
		out[i] = t.Text
	}
	return out
}

// Chips is the chip widgets, for a caller that wants to colour one.
func (f *TokenField) Chips() []*Token { return f.tokens }

// SetTokens replaces the whole list. It does not touch the editor.
func (f *TokenField) SetTokens(vals []string) {
	for _, t := range f.tokens {
		f.Base.Remove(t)
	}
	f.tokens = nil
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			f.appendToken(v)
		}
	}
	f.changed()
}

// AddToken commits one value, and reports whether it went in: Accept and
// Unique can both refuse it. (It is not called Add: that is the
// container's, and a field that took a component and a string through
// one name would be a trap.)
func (f *TokenField) AddToken(v string) bool { return f.add(strings.TrimSpace(v)) }

// RemoveAt takes token i out.
func (f *TokenField) RemoveAt(i int) {
	if i < 0 || i >= len(f.tokens) {
		return
	}
	f.Base.Remove(f.tokens[i])
	f.tokens = append(f.tokens[:i], f.tokens[i+1:]...)
	f.changed()
}

// Commit turns whatever is in the editor into a token. It is what
// leaving the field does, and what a form's Save should call before it
// reads Tokens.
func (f *TokenField) Commit() bool {
	s := strings.TrimSpace(f.editor.Text)
	if s == "" {
		return false
	}
	if f.add(s) {
		f.editor.SetText("")
		return true
	}
	return false
}

func (f *TokenField) seps() string {
	if f.Separators != "" {
		return f.Separators
	}
	return DefaultTokenSeparators
}

// splitInput takes the tokens out of text the user typed or pasted and
// leaves the tail — what comes after the last separator — in the editor.
//
// A part the field refuses stops the split: it and everything after it
// stay in the editor, minus the separator that tried to commit them, so
// the user can see what was wrong and fix it. Leaving the separator in
// would make the next keystroke try the same commit again and fail the
// same way, and the field would fight back at every letter.
func (f *TokenField) splitInput(s string) {
	if f.splitting {
		return
	}
	f.splitting = true
	defer func() { f.splitting = false }()

	seps := f.seps()
	if nextSep(s, 0, seps) < 0 {
		// Nothing but space in the editor is the space after the last
		// separator, not the start of the next value: "ada@x, grace@x"
		// must not leave " grace@x" being typed. A token cannot begin
		// with a space, so there is nothing else it could be.
		if s != "" && strings.TrimSpace(s) == "" {
			f.editor.SetText("")
		}
		return
	}
	i := 0
	for i < len(s) {
		j := nextSep(s, i, seps)
		if j < 0 {
			break
		}
		if part := strings.TrimSpace(s[i:j]); part != "" && !f.add(part) {
			f.editor.SetText(strings.TrimLeft(s[i:j]+s[j+1:], " \t"))
			return
		}
		i = j + 1
	}
	f.editor.SetText(strings.TrimLeft(s[i:], " \t"))
}

// nextSep is the index of the next separator in s at or after from, or
// -1, skipping any that falls inside a quoted string.
//
// A comma inside quotes is part of the value, not the end of it, which
// is the one piece of syntax every address list shares: `"Lovelace, Ada"
// <ada@x>, grace@x` is two recipients, and splitting on every comma made
// it three, two of them nonsense. RFC 5322 spells this a display-name
// quoted-string with backslash escapes, and the same rule reads a CSV
// field or any other quoted list, so the field does not need to know it
// is holding addresses.
//
// An unclosed quote is not an error here. The user is still typing, and
// a field that stopped accepting separators the moment a quote was
// opened would be a field that stopped working after a stray keystroke —
// so an opening quote with no partner leaves the rest of the text
// splitting normally.
func nextSep(s string, from int, seps string) int {
	quoted, closes := false, -1
	for i := from; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\\' && quoted:
			i++ // the next byte is escaped, whatever it is
		case c == '"':
			if quoted {
				quoted = false
				continue
			}
			// Only a quote that is closed hides what is between them.
			if closes = closingQuote(s, i); closes < 0 {
				continue
			}
			quoted = true
		case !quoted && strings.IndexByte(seps, c) >= 0:
			return i
		}
	}
	return -1
}

// closingQuote is the index of the quote that closes the one at i, or -1
// if it is never closed.
func closingQuote(s string, i int) int {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '"':
			return j
		}
	}
	return -1
}

func (f *TokenField) add(v string) bool {
	if v == "" {
		return false
	}
	if f.Accept != nil && !f.Accept(v) {
		return false
	}
	if f.Unique {
		for _, t := range f.tokens {
			if t.Text == v {
				return false
			}
		}
	}
	f.appendToken(v)
	f.changed()
	return true
}

func (f *TokenField) appendToken(v string) {
	t := NewToken(v)
	t.Removable = f.Removable
	t.Color = f.Color
	t.OnPress = func() { f.editor.RequestFocus() }
	t.OnRemove = func() {
		for i, c := range f.tokens {
			if c == t {
				f.RemoveAt(i)
				f.editor.RequestFocus()
				return
			}
		}
	}
	f.tokens = append(f.tokens, t)
	// The editor stays last, so the chips are in front of it.
	f.Base.Remove(f.editor)
	f.Base.Add(t)
	f.Base.Add(f.editor)
}

func (f *TokenField) changed() {
	f.Invalidate()
	f.RequestLayout()
	if f.OnChange != nil {
		f.OnChange(f.Tokens())
	}
}

// KeyPress catches the Backspace the editor let go: with nothing left to
// delete, it takes the last chip back into the editor rather than
// dropping it, so a mistyped address can be fixed.
func (f *TokenField) KeyPress(e widget.KeyEvent) bool {
	if !f.Enabled() || e.Key != platform.KeyBackspace || len(f.tokens) == 0 {
		return false
	}
	if f.editor.Text != "" {
		return false
	}
	last := len(f.tokens) - 1
	text := f.tokens[last].Text
	f.RemoveAt(last)
	f.editor.SetText(text)
	f.editor.RequestFocus()
	return true
}

// MousePress anywhere in the field that is not a chip puts the keyboard
// in the editor, which is the whole field's job as far as a click is
// concerned.
func (f *TokenField) MousePress(widget.MouseEvent) bool {
	if !f.Enabled() {
		return false
	}
	f.editor.RequestFocus()
	return true
}

func (f *TokenField) pad() style.Insets {
	p := style.Dip(f.Look(), 4)
	return style.Insets{Left: p, Top: p, Right: p, Bottom: p}
}

// flow places the chips left to right, folding onto a new line when the
// next one will not fit, and gives the editor the rest of the last line
// — or a line of its own when what is left of that one is too narrow to
// type in. Rects are in the field's local space.
func (f *TokenField) flow(maxW float32) (chips []paintengine2d.Rect, editor paintengine2d.Rect, size paintengine2d.Point) {
	// Flow at a whole-pixel width, because that is what Arrange will
	// have: SetBounds snaps a component's bounds to the pixel grid, so a
	// measure taken at 300.4 and a layout done at 300 disagree — one
	// fits a child on the line that the other folds, and the difference
	// shows as a line's height of empty space under the last row, or a
	// row running past the edge. Rounding both to the same grid makes
	// the two passes answer the same question.
	if maxW > 0 {
		maxW = float32(math.Round(float64(maxW)))
	}
	lk := f.Look()
	in := f.pad()
	gap := style.Dip(lk, 4)
	avail := maxW - in.Left - in.Right
	if maxW < 0 {
		avail = -1
	}
	loose := layout.Constraints{MaxW: avail, MaxH: -1}

	chips = make([]paintengine2d.Rect, len(f.tokens))
	x, y, lineH, width := in.Left, in.Top, float32(0), float32(0)
	for i, t := range f.tokens {
		sz := t.Measure(loose)
		if avail > 0 && x > in.Left && x+sz.X > in.Left+avail {
			y += lineH + gap
			x, lineH = in.Left, 0
		}
		chips[i] = paintengine2d.XYWH(x, y, sz.X, sz.Y)
		x += sz.X + gap
		lineH = max(lineH, sz.Y)
		width = max(width, x-gap)
	}

	esz := f.editor.Measure(loose)
	minE := style.Dip(lk, 72)
	if avail > 0 && x > in.Left && in.Left+avail-x < minE {
		y += lineH + gap
		x, lineH = in.Left, 0
	}
	ew := esz.X
	if avail > 0 {
		ew = max(in.Left+avail-x, minE)
	}
	editor = paintengine2d.XYWH(x, y, ew, esz.Y)
	lineH = max(lineH, esz.Y)
	width = max(width, x+ew)
	return chips, editor, paintengine2d.Pt(width+in.Right, y+lineH+in.Bottom)
}

// naturalWidth is the width the field asks for when none is offered.
//
// A chip field wraps, so the width it would take with every chip on one
// line is not a width it *wants* — it is only the width at which it
// would not have to fold. Reporting that made a Form size its field
// column from it (a Grid measures its columns unbounded) and a Flex
// track never goes below its content, so three addresses pushed a Write
// window's form past the window's edge and the whole form was clipped.
//
// So it asks for a text field's width and wraps inside it, which is what
// a recipient row does in every mail client: the useful answer from a
// widget that can fold is its height at the width it is given, and this
// is the width at which that answer is computed when nobody has said.
// PreferredWidth overrides it; a chip is never split, so the widest chip
// is the floor.
func (f *TokenField) naturalWidth() float32 {
	lk := f.Look()
	w := style.Dip(lk, defaultTokenFieldWidth)
	if f.PreferredWidth > 0 {
		w = style.Dip(lk, f.PreferredWidth)
	}
	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	in := f.pad()
	for _, t := range f.tokens {
		if sz := t.Measure(loose); sz.X+in.Left+in.Right > w {
			w = sz.X + in.Left + in.Right
		}
	}
	return w
}

func (f *TokenField) Measure(c layout.Constraints) paintengine2d.Point {
	maxW := float32(-1)
	if c.HasMaxW() {
		maxW = c.MaxW
	} else {
		maxW = f.naturalWidth()
	}
	_, _, size := f.flow(maxW)
	if c.HasMaxW() {
		size.X = c.MaxW
	}
	// Never shorter than one text field: an empty chip field has to look
	// like the field it is.
	if h := style.FieldHeight(f.Look().Metrics()); size.Y < h {
		size.Y = h
	}
	return c.Constrain(size)
}

func (f *TokenField) Arrange(r paintengine2d.Rect) {
	f.SetBounds(r)
	chips, editor, _ := f.flow(r.Dx())
	for i, t := range f.tokens {
		t.Arrange(chips[i])
	}
	f.editor.Arrange(editor)
}

// Paint draws the look's own text-field chrome around the whole thing —
// the well, the bevel, the capsule, whatever this era's field is — with
// no text in it, because the chips and the frameless editor inside are
// the text. Doing it any other way would mean drawing a field by hand
// and having it match 33 engines by luck.
func (f *TokenField) Paint(ctx *paintengine2d.Context) {
	st := f.State()
	if widget.Contains(f, f.hostFocus()) {
		st |= style.StateFocused
	}
	f.Look().DrawTextField(ctx, f.LocalBounds(), st, "", "", -1, 0, 0, false, 0, f.Look().Font())
}

func (f *TokenField) hostFocus() widget.Component {
	if h := f.Host(); h != nil {
		return h.Focus()
	}
	return nil
}
