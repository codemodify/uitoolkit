package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// TextField is a single-line editor (IME is out of scope for v0.1).
type TextField struct {
	widget.Base
	Text        string
	Placeholder string
	OnChange    func(string)
	OnSubmit    func(string)
	OnFocusLost func()
	Accept      func(string) bool
	caret       int
	selA, selB  int
	blinkOn     bool
	dragging    bool
	scrollX     float32
}

func NewTextField(text, placeholder string, on func(string)) *TextField {
	t := &TextField{Text: text, Placeholder: placeholder, OnChange: on, blinkOn: true}
	t.Init(t)
	t.SetWantsFocus(true)
	t.caret = runeCount(text)
	t.selA, t.selB = t.caret, t.caret
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
	h := t.Look().Metrics().ControlH
	return c.Constrain(paintengine2d.Pt(180, h))
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
	t.Look().DrawTextField(ctx, t.LocalBounds(), t.State(), t.Text, t.Placeholder, t.caret, t.selA, t.selB, t.blink(), t.scrollX)
}

func (t *TextField) FocusGained() {
	t.blinkOn = true
	t.Invalidate()
}

func (t *TextField) FocusLost() {
	t.Invalidate()
	if t.OnFocusLost != nil {
		t.OnFocusLost()
	}
}

func (t *TextField) indexAt(x float32) int {
	f := t.Look().Font()
	return f.IndexAt(t.Text, x-t.fieldPad()+t.scrollX)
}

func (t *TextField) ensureCaretVisible() {
	f := t.Look().Font()
	pad := t.fieldPad()
	inner := t.LocalBounds().Dx() - pad*2
	if inner <= 0 {
		return
	}
	cx := f.CaretX(t.Text, t.caret)
	if cx-t.scrollX > inner-2 {
		t.scrollX = cx - inner + 2
	}
	if cx-t.scrollX < 0 {
		t.scrollX = cx
	}
	if t.scrollX < 0 {
		t.scrollX = 0
	}
}

func (t *TextField) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
	t.dragging = true
	t.caret = t.indexAt(e.Pos.X)
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
	t.dragging = false
	return true
}

func (t *TextField) TextInput(r rune) bool {
	if !t.Enabled() || r < 32 {
		return false
	}
	t.replaceSel(string(r))
	return true
}

func (t *TextField) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() {
		return false
	}
	switch e.Key {
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
		t.caret = 0
		t.applyNav(e.Mods.Shift())
		return true
	case platform.KeyEnd:
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
		if t.OnSubmit != nil {
			t.OnSubmit(t.Text)
		}
		return true
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
