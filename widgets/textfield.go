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

// TextField is a single-line editor. Platform IME (XIM / text-input-v3)
// feeds preedit and commit through IMETarget.
type TextField struct {
	widget.Base
	Text        string
	Placeholder string
	// PreferredWidth is the 1x design width the field asks for, in place
	// of the default 180.
	//
	// A field cannot fold, so a layout gives it what it asks for even
	// where there is less, and the 180 was a floor under any form holding
	// one however narrow everything beside it was willing to be — a form
	// of secret fields asking for less still could not go below it.
	// [SecretField], [SecretArea] and [TokenField] have had this lever;
	// a plain field had not.
	PreferredWidth float32
	OnChange       func(string)
	// OnInput fires only for text the user put there — typing, backspace,
	// a paste, a cut, an IME commit, a screen reader's edit — and never
	// for SetText from the app. It fires after OnChange
	// (widgets/oninput.go).
	OnInput     func(string)
	OnSubmit    func(string)
	OnEscape    func()
	OnFocusLost func()
	Accept      func(string) bool
	Mono        bool
	Password    bool // paint bullets; Text stays the real value
	// Frameless paints the text alone: the field sits inside a frame its
	// parent drew (a spin box's field shares its frame with the buttons).
	Frameless bool
	// Clearable puts a small cross at the field's trailing end while
	// there is something to clear, and clicking it empties the field.
	//
	// It is opt-in, and that is the whole of the decision. A field that
	// grew one on its own would grow one everywhere: in a password box,
	// where a stray click throws away a typed secret with no undo; in a
	// form that already has a Reset, where it is a second, narrower
	// answer to a question the form has answered; in the one-character
	// boxes of a date or a licence key, where there is no room for it
	// and no work in clearing one character. None of those is a thing
	// the widget can see, and all of them are things the page knows. It
	// is not a look's decision either: whether a field can be emptied in
	// one click is a fact about what the field is for, and it must not
	// appear and disappear as the user changes theme.
	//
	// Four things turn it off whatever this says, because in each of
	// them showing it would be a bug rather than a preference:
	// [TextField.Password] (a bullet field must never offer to throw the
	// secret away), [TextField.Frameless] (the frame is the parent's and
	// so is that corner — a spin box's arrows are already there), a
	// disabled field, and a field too narrow to hold the cross and still
	// show text.
	Clearable bool
	// clearHot and clearDown are the pointer's state over that cross. It
	// is a button inside another control, so it keeps a button's
	// contract: it acts on the release, and only if the release lands
	// where the press did.
	clearHot   bool
	clearDown  bool
	caret      int
	selA, selB int
	blinkOn    bool
	dragging   bool
	// dragSel is a press inside the selection waiting to become a drag of
	// that text (drag.go); dragAt is where it landed, so a press that
	// stays a click still moves the caret there. selfDrop records a drop
	// of our own drag back into this same widget.
	dragSel      bool
	selfDrop     bool
	dragAt       int
	scrollX      float32
	preedit      string
	preeditCaret int
	user         userEdit
}

func NewTextField(text, placeholder string, on func(string)) *TextField {
	t := &TextField{Text: text, Placeholder: placeholder, OnChange: on, blinkOn: true}
	t.Init(t)
	t.SetWantsFocus(true)
	t.caret = runeCount(text)
	t.selA, t.selB = t.caret, t.caret
	return t
}

// NewMonoTextField is a single-line editor in the LookAndFeel mono role
// (JetBrains Mono). Use for paths, logs, and code-ish values.
func NewMonoTextField(text, placeholder string, on func(string)) *TextField {
	t := NewTextField(text, placeholder, on)
	t.Mono = true
	return t
}

// NewPasswordField is a single-line editor that paints a bullet per rune.
func NewPasswordField(placeholder string, on func(string)) *TextField {
	t := NewTextField("", placeholder, on)
	t.Password = true
	return t
}

func (t *TextField) SetText(s string) {
	if t.Text == s {
		return
	}
	t.Text = s
	// The caret goes to the end, which is where every toolkit puts it
	// after a programmatic set — Qt, GTK and every browser — and where
	// a caller who has just replaced the whole text means it to be.
	// It used only to be clamped *down*, so setting the text of a field
	// whose caret was at 0 left it in front of what had just been put
	// there, and the next keystroke typed behind it.
	t.caret = runeCount(s)
	t.selA, t.selB = t.caret, t.caret
	t.ensureCaretVisible()
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(s)
	}
	if t.user.is() && t.OnInput != nil {
		t.OnInput(s)
	}
}

// SetSelection sets the caret and the [a,b] rune range (order independent).
func (t *TextField) SetSelection(a, b int) {
	n := runeCount(t.Text)
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	if a > n {
		a = n
	}
	if b > n {
		b = n
	}
	t.selA, t.selB, t.caret = a, b, b
	t.ensureCaretVisible()
	t.Invalidate()
}

// Caret is the rune index of the insertion point.
func (t *TextField) Caret() int { return t.caret }

// Selection returns the unordered selection anchors.
func (t *TextField) Selection() (a, b int) { return t.selA, t.selB }

// SelectAll selects the whole value and puts the caret at its end, which
// is what Ctrl+A does and what a dialog does to an initial value it
// expects to be replaced rather than edited.
func (t *TextField) SelectAll() {
	t.selA, t.selB = 0, runeCount(t.Text)
	t.caret = t.selB
	t.Invalidate()
}

func (t *TextField) SetCaretBlink(on bool) { t.blinkOn = on }

func (t *TextField) Measure(c layout.Constraints) paintengine2d.Point {
	h := style.FieldHeight(t.Look().Metrics())
	w := float32(180)
	if t.PreferredWidth > 0 {
		w = t.PreferredWidth
	}
	return c.Constrain(paintengine2d.Pt(style.Dip(t.Look(), w), h))
}

// MinWidth is what the field asks for: it has no narrower form, so the
// width it measures at is also the width below which it stops working.
func (t *TextField) MinWidth() float32 {
	w := float32(180)
	if t.PreferredWidth > 0 {
		w = t.PreferredWidth
	}
	return style.Dip(t.Look(), w)
}

func (t *TextField) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

// ---- the clear button ---------------------------------------------------------
//
// The cross is drawn by the widget, not by an engine, and it is the one
// piece of chrome in this file that no look is asked about.
//
// A DrawFieldClear on the engine interface would be thirty-odd
// implementations of a sixteen-pixel × — thirty-odd chances for a pack
// to draw it a pixel out, thirty-odd golden files to keep — and there
// is nothing for those implementations to be faithful to. Win95 has no
// clear button. Motif has none, NeXT has none, Platinum has none: the
// affordance was invented for Mac OS X's search field and the web's
// input[type=search], long after most of these eras ended, so an engine
// asked to draw one in its era's manner has no era to draw from. What a
// look does own here it already gives: the field's own frame around it,
// [style.Palette].Text and TextMuted for the strokes, and the look's
// scale for their weight.
//
// So this uses [style.DrawCaptionGlyph] with [style.CaptionClose] — the
// same painter, the same rounding to whole device pixels, that draws
// the × on a browser tab's close button ([BrowserTabs]) and on a window
// caption. One cross, drawn one way, crisp at 1x and at 1.75x.
//
// A typed [style.ToolIcon] was the other candidate and is the wrong
// shape for this. The premiere PNG sets do ship close and x stems, but
// no ToolIcon reaches them, and adding one means a constant, a label,
// an entry in the freedesktop name table, artwork in five sets at two
// densities and both drawn-vector fallbacks — because three tests walk
// AllToolIcons and demand ink from every set for every icon. That is
// the price of an icon an application asks for by name. This is not
// one: it is a part of a control, like a combo's arrow, and parts of
// controls are drawn, not fetched.

// clearShows reports whether the cross is on screen: the field asked
// for one, there is something to clear, and clearing it is a thing this
// field may do at all. See [TextField.Clearable] for the four refusals.
func (t *TextField) clearShows() bool {
	if !t.Clearable || t.Password || t.Frameless || !t.Enabled() || t.Text == "" {
		return false
	}
	b := t.LocalBounds()
	// Room for the cross and for text beside it. A field narrow enough
	// that the cross would be most of it keeps its whole width: one
	// character is quicker to erase than to aim at.
	return b.Dx() >= style.Dip(t.Look(), 72) && t.clearSide() >= style.Dip(t.Look(), 10)
}

// clearSide is the side of the square the cross is drawn in.
func (t *TextField) clearSide() float32 {
	return min(style.Dip(t.Look(), 16), t.LocalBounds().Dy()-style.Dip(t.Look(), 6))
}

// clearRect is that square, in local coordinates: against the trailing
// end of the field, inside the frame, vertically centred and snapped to
// whole pixels so the two strokes land on the same grid at every scale.
func (t *TextField) clearRect() paintengine2d.Rect {
	b, side := t.LocalBounds(), t.clearSide()
	x := b.Max.X - t.fieldPad()*0.5 - side
	y := b.Min.Y + float32(math.Round(float64(b.Dy()-side)*0.5))
	return paintengine2d.XYWH(float32(math.Round(float64(x))), y, side, side)
}

// clearRoom is how much of the text box the cross takes, which is what
// keeps the promise that it never sits on the text or on the caret at
// the end of it. Nothing else in this file knows the cross is there:
// the text is measured, scrolled and truncated against the box less
// this, so every engine — the ones that route their text through
// baseDrawTextField and the eight that lay it out themselves — ends up
// drawing a string that stops before the cross begins.
func (t *TextField) clearRoom() float32 {
	if !t.clearShows() {
		return 0
	}
	room := t.LocalBounds().Max.X - t.fieldPad() - (t.clearRect().Min.X - style.Dip(t.Look(), 3))
	return max(room, 0)
}

// clearName is what a screen reader calls the cross. It is the field's
// own name with the verb in front of it, the way a browser tab's close
// button is "Close <title>": "Clear" alone, on a page with four fields,
// names four different buttons the same.
func (t *TextField) clearName() string {
	for _, s := range []string{t.AccessibleName(), t.Placeholder} {
		if s = strings.TrimSpace(s); s != "" {
			return "Clear " + s
		}
	}
	return "Clear"
}

// clearNow empties the field as the user's own edit, so OnInput fires
// beside OnChange (widgets/oninput.go) — a click on this cross is the
// user typing nothing, not the application assigning a value.
//
// It does not touch the focus. A field that had the caret keeps it, at
// the start of the now-empty text; a field that did not is not given it,
// because what was clicked is a button and clicking a button does not
// put a caret anywhere. That is also why the press is answered here
// rather than falling through to MousePress's caret placement.
func (t *TextField) clearNow() {
	if t.Text == "" {
		return
	}
	t.IMEReset()
	t.user.did(func() { t.SetText("") })
	t.caret, t.selA, t.selB, t.scrollX = 0, 0, 0, 0
	t.Invalidate()
}

// paintClear strokes the cross: muted while it is only there, the
// field's own text colour under the pointer, so it reads as reachable
// without a face of its own — a field is already a sunken well or a
// gel capsule, and a second raised box inside it is one box too many.
func (t *TextField) paintClear(ctx *paintengine2d.Context) {
	lk := t.Look()
	col := lk.Palette().TextMuted
	if t.clearHot || t.clearDown {
		col = lk.Palette().Text
	}
	box := t.clearRect()
	if t.clearDown {
		box = box.Inset(style.Dip(lk, 1))
	}
	style.DrawCaptionGlyph(ctx, box, style.CaptionClose, false,
		col, style.Dip(lk, 8), max(style.Dip(lk, 1.25), 1))
}

// fitClear cuts s where the cross begins. The engines draw the string
// they are handed, each clipping it to a text box of its own making, so
// the only way to keep every one of them off the cross is to hand them
// a string that does not reach it.
func (t *TextField) fitClear(s string, room float32) string {
	if room <= 0 || s == "" {
		return s
	}
	f := t.font()
	if f.Advance(s)-t.scrollX <= room {
		return s
	}
	n := runeCount(s)
	i := min(max(f.IndexAt(s, room+t.scrollX), 0), n)
	for i > 0 && f.CaretX(s, i)-t.scrollX > room {
		i--
	}
	return string([]rune(s)[:i])
}

func (t *TextField) fieldPad() float32 {
	p := t.Look().Metrics().FieldPad
	if p <= 0 {
		p = 8
	}
	return p
}

func (t *TextField) blink() bool {
	if !t.Focused() {
		return false
	}
	if h := t.Host(); h != nil {
		if b, ok := h.(interface{ CaretBlink() bool }); ok {
			return b.CaretBlink()
		}
	}
	return t.blinkOn
}

func (t *TextField) Paint(ctx *paintengine2d.Context) {
	text, caret, selA, selB := t.visual()
	st := t.State()
	if t.Frameless {
		st |= style.StateFrameless
	}
	// The cross takes its room out of the string, not out of the rect:
	// the field's frame is still drawn at its full width, so a look's
	// well, bevel or capsule is the shape it always was. See clearRoom.
	if room := t.clearRoom(); room > 0 {
		text = t.fitClear(text, t.LocalBounds().Dx()-t.fieldPad()*2-room)
		n := runeCount(text)
		caret, selA, selB = min(caret, n), min(selA, n), min(selB, n)
	}
	t.Look().DrawTextField(ctx, t.LocalBounds(), st, text, t.Placeholder, caret, selA, selB, t.blink(), t.scrollX, t.font())
	if t.preedit != "" {
		drawPreeditBar(ctx, t.Look(), t.LocalBounds(), t.fieldPad(), text, selA, selB, t.scrollX, false, t.font())
	}
	if t.clearShows() {
		t.paintClear(ctx)
	}
}

func (t *TextField) font() *style.Font {
	if t.Mono {
		return t.Look().MonoFont()
	}
	return t.Look().Font()
}

func (t *TextField) displayText() string {
	if !t.Password {
		return t.Text
	}
	return maskSecret(t.Text)
}

func maskSecret(s string) string {
	n := runeCount(s)
	if n == 0 {
		return ""
	}
	return strings.Repeat("•", n)
}

func (t *TextField) visual() (text string, caret, selA, selB int) {
	base := t.displayText()
	if t.preedit == "" {
		return base, t.caret, t.selA, t.selB
	}
	pre := t.preedit
	if t.Password {
		pre = maskSecret(t.preedit)
	}
	return platform.ComposeVisual(base, t.caret, pre, t.preeditCaret)
}

// IsSecret makes a password field a [widget.SecretTarget]: the window
// turns the input method off while it has the focus. Masking the preedit
// was never enough — the method is another process, and it sees and
// remembers what it was given whether or not this window drew it.
func (t *TextField) IsSecret() bool { return t.Password }

func (t *TextField) IMEPreedit(s string, caret int) {
	if !t.Enabled() {
		return
	}
	n := runeCount(s)
	if caret < 0 {
		caret = n
	}
	if caret > n {
		caret = n
	}
	if s == t.preedit && caret == t.preeditCaret {
		return // nothing changed: no repaint (IME done storms)
	}
	t.preedit = s
	t.preeditCaret = caret
	t.ensureCaretVisible()
	t.Invalidate()
}

func (t *TextField) IMECommit(s string) {
	t.preedit = ""
	t.preeditCaret = 0
	if s == "" {
		t.Invalidate()
		return
	}
	t.replaceSel(s)
}

func (t *TextField) IMEReset() {
	if t.preedit == "" {
		return
	}
	t.preedit = ""
	t.preeditCaret = 0
	t.Invalidate()
}

func (t *TextField) IMEDeleteSurrounding(before, after int) {
	t.preedit = ""
	t.preeditCaret = 0
	s, caret := platform.DeleteSurroundingUTF8(t.Text, t.caret, before, after)
	t.Text = s
	t.caret = caret
	t.selA, t.selB = caret, caret
	t.changed()
}

func (t *TextField) IMESurrounding() (text string, cursor, anchor int) {
	a, b := t.selA, t.selB
	if a > b {
		a, b = b, a
	}
	return t.Text, t.caret, b
}

func (t *TextField) IMECaretRect() paintengine2d.Rect {
	f := t.font()
	text, caret, _, _ := t.visual()
	pad := t.fieldPad()
	cx := f.CaretX(text, caret) - t.scrollX
	h := t.LocalBounds().Dy()
	if h < 8 {
		h = f.Height()
	}
	return paintengine2d.XYWH(pad+cx, 2, 2, h-4)
}

func (t *TextField) Preedit() string { return t.preedit }

func (t *TextField) FocusGained() {
	t.blinkOn = true
	t.Invalidate()
}

func (t *TextField) FocusLost() {
	t.IMEReset()
	t.Invalidate()
	if t.OnFocusLost != nil {
		t.OnFocusLost()
	}
}

func (t *TextField) MouseExit() {
	if t.clearHot {
		t.clearHot = false
		t.Invalidate()
	}
	t.Base.MouseExit()
}

func (t *TextField) indexAt(x float32) int {
	f := t.font()
	return f.IndexAt(t.displayText(), x-t.fieldPad()+t.scrollX)
}

func (t *TextField) ensureCaretVisible() {
	f := t.font()
	pad := t.fieldPad()
	// Less the cross's room, so the caret at the end of a long string
	// stops in front of it rather than under it.
	inner := t.LocalBounds().Dx() - pad*2 - t.clearRoom()
	if inner <= 0 {
		return
	}
	cx := f.CaretX(t.displayText(), t.caret)
	if cx-t.scrollX > inner-2 {
		t.scrollX = cx - inner + 2
	}
	if cx-t.scrollX < 0 {
		t.scrollX = cx
	}
	if t.scrollX < 0 {
		t.scrollX = 0
	}
	maxX := f.Advance(t.displayText())
	if cx > maxX {
		maxX = cx
	}
	maxScroll := maxX - inner + 2
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.scrollX > maxScroll {
		t.scrollX = maxScroll
	}
}

func (t *TextField) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	// The cross is answered before anything else in this method,
	// because everything else in this method belongs to the text: the
	// focus, the caret, the selection and the primary-selection paste.
	// A press on a button must move none of them.
	if e.Button == platform.ButtonLeft && t.clearShows() && t.clearRect().Contains(e.Pos) {
		t.clearDown, t.clearHot = true, true
		t.Invalidate()
		return true
	}
	t.RequestFocus()
	if e.Button == platform.ButtonMiddle {
		t.caret = t.indexAt(e.Pos.X)
		t.selA, t.selB = t.caret, t.caret
		t.replaceSel(platform.ClipboardPrimaryGet())
		return true
	}
	// A press inside the selection may be the start of a drag of that
	// text: the caret does not move and the selection stays until the
	// release, when it collapses if no drag took it.
	at := t.indexAt(e.Pos.X)
	if !e.Mods.Shift() && pressInSelection(at, t.selA, t.selB) {
		t.dragSel, t.selfDrop, t.dragAt = true, false, at
		return true
	}
	t.dragging = true
	t.caret = at
	if e.Mods.Shift() {
		t.selB = t.caret
	} else {
		t.selA, t.selB = t.caret, t.caret
	}
	t.ensureCaretVisible()
	t.Invalidate()
	return true
}

func (t *TextField) MouseMove(e widget.MouseEvent) bool {
	if hot := t.clearShows() && t.clearRect().Contains(e.Pos); hot != t.clearHot {
		t.clearHot = hot
		t.Invalidate()
	}
	if t.clearDown {
		// Held on the cross: the pointer may wander off it and back, and
		// it must not be extending a selection while it does.
		return true
	}
	if t.dragSel {
		// Waiting to see whether this press becomes a drag of the
		// selection; it must not extend it in the meantime.
		return true
	}
	if !t.dragging && e.Button != platform.ButtonLeft {
		return false
	}
	t.selB = t.indexAt(e.Pos.X)
	t.caret = t.selB
	t.ensureCaretVisible()
	t.Invalidate()
	return true
}

func (t *TextField) MouseRelease(e widget.MouseEvent) bool {
	if t.clearDown {
		// A button's contract: it acts on the release, and only where
		// the press landed, so a press that turned out to be a mistake
		// can be taken back by letting go somewhere else.
		t.clearDown = false
		if t.clearShows() && t.clearRect().Contains(e.Pos) {
			t.clearNow()
		}
		t.Invalidate()
		return true
	}
	if t.dragSel {
		// The press inside the selection never became a drag: it was a
		// click, and a click puts the caret where it landed.
		t.dragSel = false
		t.caret = t.dragAt
		t.selA, t.selB = t.caret, t.caret
		t.ensureCaretVisible()
		t.Invalidate()
		return true
	}
	t.dragging = false
	return true
}

func (t *TextField) TextInput(r rune) bool {
	if !t.Enabled() || r < 32 {
		return false
	}
	t.IMEReset()
	t.replaceSel(string(r))
	return true
}

func (t *TextField) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() {
		return false
	}
	if t.preedit != "" && e.Key == platform.KeyEscape {
		t.IMEReset()
		return true
	}
	switch e.Key {
	case platform.KeyEscape:
		// The keyboard's way to the cross, and the convention of every
		// field that has one (NSSearchField, GtkSearchEntry, a
		// QLineEdit with a clear button): the first Escape empties the
		// field, and only once it is empty does Escape mean what the
		// page says it means — leave, close, cancel.
		if t.clearShows() {
			t.clearNow()
			return true
		}
		if t.OnEscape != nil {
			t.OnEscape()
			return true
		}
		return false
	case platform.KeyBackspace:
		if t.hasSel() {
			t.replaceSel("")
			return true
		}
		if t.caret > 0 {
			t.Text = dropRune(t.Text, t.caret-1)
			t.caret--
			t.selA, t.selB = t.caret, t.caret
			t.changed()
			return true
		}
		// Nothing to delete: the key bubbles, for the same reason Return
		// does below. A chip field's Backspace on an empty editor takes
		// back the last recipient, and a field that swallowed the key
		// with no visible effect would be the only thing stopping it.
		return false
	case platform.KeyDelete:
		if t.hasSel() {
			t.replaceSel("")
			return true
		}
		if t.caret < runeCount(t.Text) {
			t.Text = dropRune(t.Text, t.caret)
			t.changed()
		}
		return true
	case platform.KeyLeft:
		if e.Mods.Ctrl() {
			t.caret = wordBoundary(t.Text, t.caret, -1)
		} else if t.hasSel() && !e.Mods.Shift() {
			t.caret = selMin(t.selA, t.selB)
		} else if t.caret > 0 {
			t.caret--
		}
		t.applyNav(e.Mods.Shift())
		return true
	case platform.KeyRight:
		if e.Mods.Ctrl() {
			t.caret = wordBoundary(t.Text, t.caret, 1)
		} else if t.hasSel() && !e.Mods.Shift() {
			t.caret = selMax(t.selA, t.selB)
		} else if t.caret < runeCount(t.Text) {
			t.caret++
		}
		t.applyNav(e.Mods.Shift())
		return true
	case platform.KeyHome:
		// Ctrl+Home / Ctrl+End are document-level: a single-line field already
		// does the same thing with plain Home / End, so the Ctrl form bubbles
		// to the parent (NumberField min / max, an outer scroll pane).
		if e.Mods.Ctrl() {
			return false
		}
		t.caret = 0
		t.applyNav(e.Mods.Shift())
		return true
	case platform.KeyEnd:
		if e.Mods.Ctrl() {
			return false
		}
		t.caret = runeCount(t.Text)
		t.applyNav(e.Mods.Shift())
		return true
	case platform.KeyA:
		if e.Mods.Ctrl() {
			t.SelectAll()
			return true
		}
	case platform.KeyC:
		if e.Mods.Ctrl() {
			if s := t.SelectedText(); s != "" {
				platform.ClipboardSet(s)
			}
			return true
		}
	case platform.KeyX:
		if e.Mods.Ctrl() {
			if s := t.SelectedText(); s != "" {
				platform.ClipboardSet(s)
				t.replaceSel("")
			}
			return true
		}
	case platform.KeyV:
		if e.Mods.Ctrl() {
			t.replaceSel(platform.ClipboardGet())
			return true
		}
	case platform.KeyReturn:
		// A field with nothing to do on Return lets it bubble, so the
		// dialog or wizard around it presses its default button (a
		// QLineEdit's Return reaches QDialog's default button the same way).
		if t.OnSubmit != nil {
			t.OnSubmit(t.Text)
			return true
		}
		return false
	}
	return false
}

func (t *TextField) applyNav(extend bool) {
	if !extend {
		t.selA, t.selB = t.caret, t.caret
	} else {
		t.selB = t.caret
	}
	t.ensureCaretVisible()
	t.Invalidate()
}

func (t *TextField) hasSel() bool { return t.selA != t.selB }

// SelectedText is the current selection, or empty if the caret is collapsed.
func (t *TextField) SelectedText() string {
	if !t.hasSel() {
		return ""
	}
	a, b := t.selA, t.selB
	if a > b {
		a, b = b, a
	}
	runes := []rune(t.Text)
	if a < 0 {
		a = 0
	}
	if b > len(runes) {
		b = len(runes)
	}
	return string(runes[a:b])
}

func (t *TextField) previewReplace(s string) string {
	a, b := t.selA, t.selB
	if a == b {
		a, b = t.caret, t.caret
	}
	if a > b {
		a, b = b, a
	}
	runes := []rune(t.Text)
	if a < 0 {
		a = 0
	}
	if b > len(runes) {
		b = len(runes)
	}
	return string(runes[:a]) + s + string(runes[b:])
}

// replaceSel puts s in place of the selection and reports whether it
// did. An Accept validator can refuse, and a caller that has told a drag
// source the drop succeeded needs to know the difference: the source
// removes its original on the strength of that answer.
func (t *TextField) replaceSel(s string) bool {
	a, b := t.selA, t.selB
	if a == b {
		a, b = t.caret, t.caret
	}
	if a > b {
		a, b = b, a
	}
	if a == b && s == "" {
		return false
	}
	if t.Accept != nil && !t.Accept(t.previewReplace(s)) {
		return false
	}
	runes := []rune(t.Text)
	if a < 0 {
		a = 0
	}
	if b > len(runes) {
		b = len(runes)
	}
	t.Text = string(runes[:a]) + s + string(runes[b:])
	t.caret = a + runeCount(s)
	t.selA, t.selB = t.caret, t.caret
	t.changed()
	return true
}

// changed is every edit the user makes — typing, backspace, a paste, a cut,
// an IME commit — and nothing the app does, which goes through SetText.
func (t *TextField) changed() {
	t.ensureCaretVisible()
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(t.Text)
	}
	if t.OnInput != nil {
		t.OnInput(t.Text)
	}
}

func runeCount(s string) int { return len([]rune(s)) }

func dropRune(s string, i int) string {
	r := []rune(s)
	if i < 0 || i >= len(r) {
		return s
	}
	return string(append(r[:i], r[i+1:]...))
}

func selMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func selMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func wordBoundary(s string, i, dir int) int {
	r := []rune(s)
	n := len(r)
	if n == 0 {
		return 0
	}
	if dir < 0 {
		if i <= 0 {
			return 0
		}
		i--
		for i > 0 && isWordSep(r[i]) {
			i--
		}
		for i > 0 && !isWordSep(r[i-1]) {
			i--
		}
		return i
	}
	for i < n && !isWordSep(r[i]) {
		i++
	}
	for i < n && isWordSep(r[i]) {
		i++
	}
	return i
}

func isWordSep(r rune) bool {
	return r == ' ' || r == '\t' || r == '/' || r == '-' || r == '_' || r == '.' || r == ','
}
