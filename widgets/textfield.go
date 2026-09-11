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
	caret       int
	selA, selB  int
	blinkOn     bool
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
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(s)
	}
}

func (t *TextField) Measure(c layout.Constraints) paintengine2d.Point {
	h := t.Look().Metrics().ControlH
	return c.Constrain(paintengine2d.Pt(180, h))
}

func (t *TextField) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

func (t *TextField) Paint(ctx *paintengine2d.Context) {
	t.Look().DrawTextField(ctx, t.LocalBounds(), t.State(), t.Text, t.Placeholder, t.caret, t.selA, t.selB, t.blinkOn && t.Focused())
}

func (t *TextField) FocusGained() {
	t.blinkOn = true
	t.Invalidate()
}

func (t *TextField) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
	f := t.Look().Font()
	x := e.Pos.X - 8
	t.caret = f.IndexAt(t.Text, x)
	t.selA, t.selB = t.caret, t.caret
	t.Invalidate()
	return true
}

func (t *TextField) MouseMove(e widget.MouseEvent) bool {
	if e.Button == platform.ButtonLeft || t.selDragging(e) {
		f := t.Look().Font()
		t.selB = f.IndexAt(t.Text, e.Pos.X-8)
		t.caret = t.selB
		t.Invalidate()
		return true
	}
	return false
}

func (t *TextField) selDragging(e widget.MouseEvent) bool { return false }

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
		if t.caret > 0 {
			t.caret--
		}
		if !e.Mods.Shift() {
			t.selA, t.selB = t.caret, t.caret
		} else {
			t.selB = t.caret
		}
		t.Invalidate()
		return true
	case platform.KeyRight:
		if t.caret < runeCount(t.Text) {
			t.caret++
		}
		if !e.Mods.Shift() {
			t.selA, t.selB = t.caret, t.caret
		} else {
			t.selB = t.caret
		}
		t.Invalidate()
		return true
	case platform.KeyHome:
		t.caret = 0
		if !e.Mods.Shift() {
			t.selA, t.selB = 0, 0
		} else {
			t.selB = 0
		}
		t.Invalidate()
		return true
	case platform.KeyEnd:
		t.caret = runeCount(t.Text)
		if !e.Mods.Shift() {
			t.selA, t.selB = t.caret, t.caret
		} else {
			t.selB = t.caret
		}
		t.Invalidate()
		return true
	case platform.KeyA:
		if e.Mods.Ctrl() {
			t.selA, t.selB = 0, runeCount(t.Text)
			t.caret = t.selB
			t.Invalidate()
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

func (t *TextField) hasSel() bool { return t.selA != t.selB }

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
