package widgets

import (
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
	OnChange    func(string)
	OnSubmit    func(string)
	OnEscape    func()
	OnFocusLost func()
	Accept      func(string) bool
	Mono        bool
	Password    bool // paint bullets; Text stays the real value
	// Frameless paints the text alone: the field sits inside a frame its
	// parent drew (a spin box's field shares its frame with the buttons).
	Frameless  bool
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
	n := runeCount(s)
	if t.caret > n {
		t.caret = n
	}
	t.selA, t.selB = t.caret, t.caret
	t.ensureCaretVisible()
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(s)
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

func (t *TextField) SetCaretBlink(on bool) { t.blinkOn = on }

func (t *TextField) Measure(c layout.Constraints) paintengine2d.Point {
	h := style.FieldHeight(t.Look().Metrics())
	return c.Constrain(paintengine2d.Pt(style.Dip(t.Look(), 180), h))
}

func (t *TextField) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

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
	t.Look().DrawTextField(ctx, t.LocalBounds(), st, text, t.Placeholder, caret, selA, selB, t.blink(), t.scrollX, t.font())
	if t.preedit != "" {
		drawPreeditBar(ctx, t.Look(), t.LocalBounds(), t.fieldPad(), text, selA, selB, t.scrollX, false, t.font())
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

func (t *TextField) indexAt(x float32) int {
	f := t.font()
	return f.IndexAt(t.displayText(), x-t.fieldPad()+t.scrollX)
}

func (t *TextField) ensureCaretVisible() {
	f := t.font()
	pad := t.fieldPad()
	inner := t.LocalBounds().Dx() - pad*2
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

func (t *TextField) MouseRelease(widget.MouseEvent) bool {
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
		}
		return true
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
			t.selA, t.selB = 0, runeCount(t.Text)
			t.caret = t.selB
			t.Invalidate()
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

func (t *TextField) replaceSel(s string) {
	a, b := t.selA, t.selB
	if a == b {
		a, b = t.caret, t.caret
	}
	if a > b {
		a, b = b, a
	}
	if a == b && s == "" {
		return
	}
	if t.Accept != nil && !t.Accept(t.previewReplace(s)) {
		return
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
}

func (t *TextField) changed() {
	t.ensureCaretVisible()
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(t.Text)
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
