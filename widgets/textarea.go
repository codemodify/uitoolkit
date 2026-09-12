package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// TextArea is a multi-line editor with wrap (default) or horizontal scroll.
// Set ReadOnly (or use NewTextView) for a scrollable message/log pane that
// does not accept keyboard text input.
type TextArea struct {
	widget.Base
	Text         string
	Placeholder  string
	Wrap         bool
	ReadOnly     bool
	MinRows      int
	OnChange     func(string)
	OnFocusLost  func()
	Accept       func(string) bool
	Mono         bool
	caret        int
	selA, selB   int
	blinkOn      bool
	dragging     bool
	scrollX      float32
	scrollY      float32
	lines        []style.TextLine
	preferX      float32
	havePref     bool
	preedit      string
	preeditCaret int
	vbar         scrollDrag
	hbar         scrollDrag
}

// NewTextArea builds a wrapping multi-line field.
func NewTextArea(text, placeholder string, on func(string)) *TextArea {
	t := &TextArea{Text: text, Placeholder: placeholder, OnChange: on, Wrap: true, blinkOn: true, MinRows: 3}
	t.Init(t)
	t.SetWantsFocus(true)
	t.caret = runeCount(text)
	t.selA, t.selB = t.caret, t.caret
	return t
}

// NewMonoTextArea is a multi-line editor in the LookAndFeel mono role
// (JetBrains Mono). Use for code, Inspector dumps, and logs.
func NewMonoTextArea(text, placeholder string, on func(string)) *TextArea {
	t := NewTextArea(text, placeholder, on)
	t.Mono = true
	return t
}

// NewTextView is a wrapping, scrollable, read-only text pane (message view).
func NewTextView(text, placeholder string) *TextArea {
	t := NewTextArea(text, placeholder, nil)
	t.ReadOnly = true
	t.Wrap = true
	return t
}

// NewMonoTextView is a read-only mono TextView (raw source, logs).
func NewMonoTextView(text, placeholder string) *TextArea {
	t := NewTextView(text, placeholder)
	t.Mono = true
	return t
}

func (t *TextArea) editable() bool { return t.Enabled() && !t.ReadOnly }

func (t *TextArea) SetText(s string) {
	if t.Text == s {
		return
	}
	t.Text = s
	n := runeCount(s)
	if t.caret > n {
		t.caret = n
	}
	t.selA, t.selB = t.caret, t.caret
	t.havePref = false
	t.relayout()
	t.ensureCaretVisible()
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(s)
	}
}

// SetSelection sets the caret and the [a,b] rune range (order independent).
func (t *TextArea) SetSelection(a, b int) {
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
	t.havePref = false
	t.relayout()
	t.ensureCaretVisible()
	t.Invalidate()
}

func (t *TextArea) Caret() int { return t.caret }

func (t *TextArea) Selection() (a, b int) { return t.selA, t.selB }

func (t *TextArea) SetCaretBlink(on bool) { t.blinkOn = on }

// InvalidateCaret dirties only the insertion bar (blink / hide).
func (t *TextArea) InvalidateCaret() {
	t.InvalidateRect(t.IMECaretRect().Inset(-2))
}

func (t *TextArea) Lines() []style.TextLine {
	t.relayout()
	return t.lines
}

func (t *TextArea) fieldPad() float32 {
	p := t.Look().Metrics().FieldPad
	if p <= 0 {
		p = 8
	}
	return p
}

func (t *TextArea) font() *style.Font {
	if t.Mono {
		return t.Look().MonoFont()
	}
	return t.Look().Font()
}

func (t *TextArea) lineH() float32 {
	h := t.font().Height() + 2
	if h < 12 {
		h = 18
	}
	return h
}

func (t *TextArea) minRows() int {
	if t.MinRows > 0 {
		return t.MinRows
	}
	return 3
}

func (t *TextArea) Measure(c layout.Constraints) paintengine2d.Point {
	h := float32(t.minRows())*t.lineH() + t.fieldPad()*2
	w := float32(220)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (t *TextArea) Arrange(r paintengine2d.Rect) {
	t.SetBounds(r)
	t.relayout()
	t.clampScroll()
}

func (t *TextArea) inner() paintengine2d.Rect {
	pad := t.fieldPad()
	b := t.LocalBounds()
	return paintengine2d.XYWH(pad, pad, b.Dx()-pad*2, b.Dy()-pad*2)
}

func (t *TextArea) relayout() {
	w := t.inner().Dx()
	if !t.Wrap {
		w = 1e6
	} else if w < 8 {
		w = 8
	}
	t.lines = layoutArea(t.font(), t.Text, w, t.Wrap)
}

func (t *TextArea) contentH() float32 {
	n := len(t.lines)
	if n < 1 {
		n = 1
	}
	return float32(n) * t.lineH()
}

func (t *TextArea) maxScrollY() float32 {
	return layout.MaxScroll(t.contentH(), t.inner().Dy())
}

func (t *TextArea) contentW() float32 {
	if t.Wrap {
		return t.inner().Dx()
	}
	var w float32
	f := t.font()
	for _, ln := range t.lines {
		if adv := f.Advance(ln.Text); adv > w {
			w = adv
		}
	}
	return w
}

func (t *TextArea) maxScrollX() float32 {
	if t.Wrap {
		return 0
	}
	return layout.MaxScroll(t.contentW(), t.inner().Dx())
}

// MaxOffset is the largest legal vertical scroll offset.
func (t *TextArea) MaxOffset() float32 { return t.maxScrollY() }

// OffsetY is the current vertical scroll (clamped on Arrange / input).
func (t *TextArea) OffsetY() float32 { return t.scrollY }

// ScrollTo sets the vertical offset (clamped).
func (t *TextArea) ScrollTo(y float32) {
	t.setScroll(t.scrollX, y)
}

func (t *TextArea) clampScroll() {
	t.scrollY = layout.ClampScroll(t.scrollY, t.contentH(), t.inner().Dy())
	t.scrollX = layout.ClampScroll(t.scrollX, t.contentW(), t.inner().Dx())
}

func (t *TextArea) scrollView() paintengine2d.Rect {
	inner := t.inner()
	vt, _ := t.scrollTrackV()
	ht, _ := t.scrollTrackH()
	return contentViewMinusHBar(contentViewMinusVBar(inner, vt), ht)
}

func (t *TextArea) setScroll(x, y float32) {
	oldX, oldY := t.scrollX, t.scrollY
	_, oldVT := t.scrollTrackV()
	_, oldHT := t.scrollTrackH()
	commitScrollXY(t, t.scrollView(), oldX, oldY, x, y, t.maxScrollX(), t.maxScrollY(), func(nx, ny float32) {
		t.scrollX, t.scrollY = nx, ny
	})
	if t.scrollX != oldX || t.scrollY != oldY {
		_, newVT := t.scrollTrackV()
		_, newHT := t.scrollTrackH()
		invalidateOverflowThumbs(t, oldVT, newVT)
		invalidateOverflowThumbs(t, oldHT, newHT)
	}
}

func (t *TextArea) scrollBy(dy, dx float32) {
	nx, ny := t.scrollX, t.scrollY+dy
	if !t.Wrap {
		nx += dx
	}
	t.setScroll(nx, ny)
}

func (t *TextArea) scrollTrackV() (track, thumb paintengine2d.Rect) {
	bar, gap := overflowBarSize(t.Look())
	return vScrollThumb(t.LocalBounds(), t.contentH(), t.scrollY, bar, gap)
}

func (t *TextArea) scrollTrackH() (track, thumb paintengine2d.Rect) {
	if t.Wrap {
		return
	}
	bar, gap := overflowBarSize(t.Look())
	return hScrollThumb(t.LocalBounds(), t.contentW(), t.scrollX, bar, gap)
}

func (t *TextArea) blink() bool {
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

func (t *TextArea) Paint(ctx *paintengine2d.Context) {
	t.relayout()
	t.clampScroll()
	text, caret, selA, selB := t.visual()
	w := t.inner().Dx()
	if !t.Wrap {
		w = 1e6
	} else if w < 8 {
		w = 8
	}
	lines := layoutArea(t.font(), text, w, t.Wrap)
	blink := t.blink() && t.editable()
	if t.ReadOnly {
		caret = -1
		blink = false
	}
	t.Look().DrawTextArea(ctx, t.LocalBounds(), t.State(), lines, caret, selA, selB, blink, t.scrollX, t.scrollY, t.Placeholder, t.font())
	vt, vth := t.scrollTrackV()
	paintOverflowBar(ctx, t.Look(), vt, vth, t.vbar.over, t.vbar.active)
	ht, hth := t.scrollTrackH()
	paintOverflowBar(ctx, t.Look(), ht, hth, t.hbar.over, t.hbar.active)
}

func (t *TextArea) visual() (text string, caret, selA, selB int) {
	if t.preedit == "" {
		return t.Text, t.caret, t.selA, t.selB
	}
	return platform.ComposeVisual(t.Text, t.caret, t.preedit, t.preeditCaret)
}

func (t *TextArea) IMEPreedit(s string, caret int) {
	if !t.editable() {
		return
	}
	t.preedit = s
	n := runeCount(s)
	if caret < 0 {
		caret = n
	}
	if caret > n {
		caret = n
	}
	t.preeditCaret = caret
	t.relayout()
	t.ensureCaretVisible()
	t.Invalidate()
}

func (t *TextArea) IMECommit(s string) {
	t.preedit = ""
	t.preeditCaret = 0
	if !t.editable() || s == "" {
		t.Invalidate()
		return
	}
	t.replaceSel(s)
}

func (t *TextArea) IMEReset() {
	if t.preedit == "" {
		return
	}
	t.preedit = ""
	t.preeditCaret = 0
	t.Invalidate()
}

func (t *TextArea) IMEDeleteSurrounding(before, after int) {
	t.preedit = ""
	t.preeditCaret = 0
	if !t.editable() {
		return
	}
	s, caret := platform.DeleteSurroundingUTF8(t.Text, t.caret, before, after)
	t.Text = s
	t.caret = caret
	t.selA, t.selB = caret, caret
	t.changed()
}

func (t *TextArea) IMESurrounding() (text string, cursor, anchor int) {
	a, b := t.selA, t.selB
	if a > b {
		a, b = b, a
	}
	return t.Text, t.caret, b
}

func (t *TextArea) IMECaretRect() paintengine2d.Rect {
	t.relayout()
	inner := t.inner()
	li := t.lineIndexOf(t.caret)
	y := inner.Min.Y + float32(li)*t.lineH() - t.scrollY
	cx := float32(0)
	if li >= 0 && li < len(t.lines) {
		ln := t.lines[li]
		cx = t.font().CaretX(ln.Text, t.caret-ln.Start) - t.scrollX
	}
	return paintengine2d.XYWH(inner.Min.X+cx, y, 2, t.lineH())
}

func (t *TextArea) Preedit() string { return t.preedit }

func (t *TextArea) FocusGained() {
	t.blinkOn = true
	t.Invalidate()
}

func (t *TextArea) FocusLost() {
	t.IMEReset()
	t.Invalidate()
	if t.OnFocusLost != nil {
		t.OnFocusLost()
	}
}

func (t *TextArea) lineIndexOf(caret int) int {
	if len(t.lines) == 0 {
		return 0
	}
	for i, ln := range t.lines {
		if caret < ln.End || i == len(t.lines)-1 {
			if caret >= ln.Start {
				return i
			}
		}
	}
	return len(t.lines) - 1
}

func (t *TextArea) indexAt(x, y float32) int {
	t.relayout()
	if len(t.lines) == 0 {
		return 0
	}
	inner := t.inner()
	li := int((y - inner.Min.Y + t.scrollY) / t.lineH())
	if li < 0 {
		li = 0
	}
	if li >= len(t.lines) {
		li = len(t.lines) - 1
	}
	ln := t.lines[li]
	col := t.font().IndexAt(ln.Text, x-inner.Min.X+t.scrollX)
	n := runeCount(ln.Text)
	if col > n {
		col = n
	}
	idx := ln.Start + col
	if idx > ln.End {
		idx = ln.End
	}
	if li < len(t.lines)-1 && idx == ln.End && ln.End > ln.Start+n {
		idx = ln.End - 1
	}
	return idx
}

func (t *TextArea) ensureCaretVisible() {
	t.relayout()
	inner := t.inner()
	if inner.Dy() <= 0 {
		return
	}
	li := t.lineIndexOf(t.caret)
	y := float32(li) * t.lineH()
	lh := t.lineH()
	if y < t.scrollY {
		t.scrollY = y
	}
	if y+lh > t.scrollY+inner.Dy() {
		t.scrollY = y + lh - inner.Dy()
	}
	if !t.Wrap && li >= 0 && li < len(t.lines) {
		ln := t.lines[li]
		cx := t.font().CaretX(ln.Text, t.caret-ln.Start)
		if cx-t.scrollX > inner.Dx()-2 {
			t.scrollX = cx - inner.Dx() + 2
		}
		if cx-t.scrollX < 0 {
			t.scrollX = cx
		}
	}
	t.clampScroll()
}

func (t *TextArea) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
	vt, vth := t.scrollTrackV()
	if off, ok := t.vbar.press(e.Pos, vt, vth, true, t.scrollY, t.maxScrollY(), t.inner().Dy()*0.9); ok {
		t.setScroll(t.scrollX, off)
		return true
	}
	ht, hth := t.scrollTrackH()
	if off, ok := t.hbar.press(e.Pos, ht, hth, false, t.scrollX, t.maxScrollX(), t.inner().Dx()*0.9); ok {
		t.setScroll(off, t.scrollY)
		return true
	}
	if e.Button == platform.ButtonMiddle {
		if !t.editable() {
			return true
		}
		t.havePref = false
		t.caret = t.indexAt(e.Pos.X, e.Pos.Y)
		t.selA, t.selB = t.caret, t.caret
		t.replaceSel(platform.ClipboardPrimaryGet())
		return true
	}
	t.dragging = true
	t.havePref = false
	t.caret = t.indexAt(e.Pos.X, e.Pos.Y)
	if e.Mods.Shift() {
		t.selB = t.caret
	} else {
		t.selA, t.selB = t.caret, t.caret
	}
	t.ensureCaretVisible()
	t.Invalidate()
	return true
}

func (t *TextArea) MouseMove(e widget.MouseEvent) bool {
	vt, vth := t.scrollTrackV()
	if applyScrollHover(&t.vbar, e.Pos, vt, vth, true, t.maxScrollY(), func(off float32) {
		t.setScroll(t.scrollX, off)
	}, nil, func() { invalidateOverflowTrack(t, vt) }) {
		return true
	}
	ht, hth := t.scrollTrackH()
	if applyScrollHover(&t.hbar, e.Pos, ht, hth, false, t.maxScrollX(), func(off float32) {
		t.setScroll(off, t.scrollY)
	}, nil, func() { invalidateOverflowTrack(t, ht) }) {
		return true
	}
	if !t.dragging && e.Button != platform.ButtonLeft {
		return false
	}
	t.selB = t.indexAt(e.Pos.X, e.Pos.Y)
	t.caret = t.selB
	t.ensureCaretVisible()
	t.Invalidate()
	return true
}

func (t *TextArea) MouseRelease(widget.MouseEvent) bool {
	t.dragging = false
	if t.vbar.release() || t.hbar.release() {
		vt, _ := t.scrollTrackV()
		ht, _ := t.scrollTrackH()
		invalidateOverflowTrack(t, vt)
		invalidateOverflowTrack(t, ht)
	}
	return true
}

func (t *TextArea) MouseExit() {
	was := t.vbar.over || t.hbar.over
	t.vbar.over = false
	t.hbar.over = false
	if !was {
		return
	}
	vt, _ := t.scrollTrackV()
	ht, _ := t.scrollTrackH()
	invalidateOverflowTrack(t, vt)
	invalidateOverflowTrack(t, ht)
}

func (t *TextArea) MouseWheel(e widget.MouseEvent) bool {
	dy := e.Scroll.Y
	if dy == 0 && e.Scroll.X == 0 {
		return false
	}
	t.scrollBy(wheelDelta(dy, t.lineH()), e.Scroll.X)
	return true
}

func (t *TextArea) TextInput(r rune) bool {
	if !t.editable() || r < 32 {
		return false
	}
	t.IMEReset()
	t.replaceSel(string(r))
	return true
}

func (t *TextArea) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() {
		return false
	}
	if t.preedit != "" && e.Key == platform.KeyEscape {
		t.IMEReset()
		return true
	}
	if t.ReadOnly {
		return t.readOnlyKey(e)
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
		t.applyNav(e.Mods.Shift(), false)
		return true
	case platform.KeyRight:
		if e.Mods.Ctrl() {
			t.caret = wordBoundary(t.Text, t.caret, 1)
		} else if t.hasSel() && !e.Mods.Shift() {
			t.caret = selMax(t.selA, t.selB)
		} else if t.caret < runeCount(t.Text) {
			t.caret++
		}
		t.applyNav(e.Mods.Shift(), false)
		return true
	case platform.KeyUp:
		t.moveVert(-1, e.Mods.Shift())
		return true
	case platform.KeyDown:
		t.moveVert(1, e.Mods.Shift())
		return true
	case platform.KeyHome:
		if e.Mods.Ctrl() {
			t.caret = 0
		} else {
			t.relayout()
			ln := t.lines[t.lineIndexOf(t.caret)]
			t.caret = ln.Start
		}
		t.applyNav(e.Mods.Shift(), false)
		return true
	case platform.KeyEnd:
		if e.Mods.Ctrl() {
			t.caret = runeCount(t.Text)
		} else {
			t.relayout()
			li := t.lineIndexOf(t.caret)
			ln := t.lines[li]
			t.caret = ln.Start + runeCount(ln.Text)
		}
		t.applyNav(e.Mods.Shift(), false)
		return true
	case platform.KeyPageUp:
		page := int(t.inner().Dy()/t.lineH()) - 1
		if page < 1 {
			page = 1
		}
		t.moveVert(-page, e.Mods.Shift())
		return true
	case platform.KeyPageDown:
		page := int(t.inner().Dy()/t.lineH()) - 1
		if page < 1 {
			page = 1
		}
		t.moveVert(page, e.Mods.Shift())
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
		t.replaceSel("\n")
		return true
	}
	return false
}

func (t *TextArea) moveVert(dir int, extend bool) {
	t.relayout()
	if len(t.lines) == 0 {
		return
	}
	li := t.lineIndexOf(t.caret)
	ln := t.lines[li]
	if !t.havePref {
		t.preferX = t.font().CaretX(ln.Text, t.caret-ln.Start)
		t.havePref = true
	}
	next := li + dir
	if next < 0 {
		t.caret = 0
	} else if next >= len(t.lines) {
		t.caret = runeCount(t.Text)
	} else {
		dst := t.lines[next]
		col := t.font().IndexAt(dst.Text, t.preferX)
		n := runeCount(dst.Text)
		if col > n {
			col = n
		}
		t.caret = dst.Start + col
		if t.caret > dst.End {
			t.caret = dst.End
		}
	}
	t.applyNav(extend, true)
}

func (t *TextArea) applyNav(extend, keepPref bool) {
	if !extend {
		t.selA, t.selB = t.caret, t.caret
	} else {
		t.selB = t.caret
	}
	if !keepPref {
		t.havePref = false
	}
	t.ensureCaretVisible()
	t.Invalidate()
}

func (t *TextArea) hasSel() bool { return t.selA != t.selB }

// SelectedText is the current selection, or empty if the caret is collapsed.
func (t *TextArea) SelectedText() string {
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

func (t *TextArea) previewReplace(s string) string {
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

func (t *TextArea) readOnlyKey(e widget.KeyEvent) bool {
	switch e.Key {
	case platform.KeyDown:
		t.scrollBy(t.lineH(), 0)
		return true
	case platform.KeyUp:
		t.scrollBy(-t.lineH(), 0)
		return true
	case platform.KeyPageDown:
		t.scrollBy(t.inner().Dy()*0.9, 0)
		return true
	case platform.KeyPageUp:
		t.scrollBy(-t.inner().Dy()*0.9, 0)
		return true
	case platform.KeyHome:
		if e.Mods.Ctrl() {
			t.scrollBy(-t.scrollY, 0)
			return true
		}
	case platform.KeyEnd:
		if e.Mods.Ctrl() {
			t.scrollBy(t.maxScrollY(), 0)
			return true
		}
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
	}
	return false
}

func (t *TextArea) replaceSel(s string) {
	if !t.editable() {
		return
	}
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
	t.havePref = false
	t.changed()
}

func (t *TextArea) changed() {
	t.relayout()
	t.ensureCaretVisible()
	t.Invalidate()
	if t.OnChange != nil {
		t.OnChange(t.Text)
	}
}

func layoutArea(f *style.Font, text string, maxW float32, wrap bool) []style.TextLine {
	runes := []rune(text)
	if len(runes) == 0 {
		return []style.TextLine{{Text: "", Start: 0, End: 0}}
	}
	if maxW < 4 {
		maxW = 4
	}
	var out []style.TextLine
	start := 0
	flush := func(end int, includeNL bool) {
		if end < start {
			end = start
		}
		srcEnd := end
		if includeNL && end < len(runes) && runes[end] == '\n' {
			srcEnd = end + 1
		}
		out = append(out, style.TextLine{
			Text:  string(runes[start:end]),
			Start: start,
			End:   srcEnd,
		})
		start = srcEnd
	}
	i := 0
	for i < len(runes) {
		if runes[i] == '\n' {
			flush(i, true)
			i = start
			continue
		}
		if wrap {
			adv := f.Advance(string(runes[start : i+1]))
			if adv > maxW && i > start {
				brk := lastWrap(runes, start, i)
				flush(brk, false)
				i = start
				continue
			}
		}
		i++
	}
	if start < len(runes) {
		flush(len(runes), false)
	}
	if runes[len(runes)-1] == '\n' {
		out = append(out, style.TextLine{Text: "", Start: len(runes), End: len(runes)})
	}
	return out
}

func lastWrap(runes []rune, start, i int) int {
	for j := i; j > start; j-- {
		if runes[j-1] == ' ' || runes[j-1] == '\t' {
			return j
		}
	}
	return i
}
