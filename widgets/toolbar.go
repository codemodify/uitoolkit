package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ToolItem is one thing on a ToolBar: a tool button (text, icon, or
// both), the rule between two groups, a word that names the control
// beside it ([ToolLabel]), a control of the application's own
// ([ToolWidget]), or the free space that pushes what follows to the
// right ([ToolStretch]).
type ToolItem struct {
	Text     string
	Icon     style.ToolIcon
	Disabled bool
	Toggle   bool
	Down     bool
	Sep      bool
	Tip      string
	OnClick  func()
	// Widget is a control the bar carries instead of a tool button — a
	// combo box, a search field, a progress bar — laid out on the bar,
	// drawn on its background and focused in its own right. Use
	// [ToolWidget].
	Widget widget.Component
	// Stretch is free space that eats what the bar has spare, so the
	// items after it sit at its right-hand end. Use [ToolStretch].
	Stretch bool
	// Label makes Text a word on the bar rather than a button: the name
	// of the control that follows it. Use [ToolLabel].
	Label bool
}

// ToolText is a labeled tool button.
func ToolText(text string, on func()) *ToolItem {
	return &ToolItem{Text: text, OnClick: on}
}

// ToolIconBtn is an icon tool button. Text is optional (shown beside the icon).
func ToolIconBtn(icon style.ToolIcon, text string, on func()) *ToolItem {
	return &ToolItem{Icon: icon, Text: text, OnClick: on}
}

// ToolToggle is a sticky tool button.
func ToolToggle(text string, down bool, on func()) *ToolItem {
	return &ToolItem{Text: text, Toggle: true, Down: down, OnClick: on}
}

// ToolDivider is a vertical rule between tool groups.
func ToolDivider() *ToolItem { return &ToolItem{Sep: true} }

// ToolLabel is a word on the bar that says what the control after it
// sets — Qt's QToolBar with a QLabel in front of a combo box, the "Zoom"
// of every drawing program. The bar draws it in its own text, with half
// a gap after it so it reads as belonging to the control it names and
// not to the one before it.
//
// It is not a button: nothing hovers it, the bar's arrow keys step over
// it, and it carries no command. It is also droppable — a bar too narrow
// for everything sheds it along with its tools, because the control it
// names keeps its tooltip and its accessible name and so is still usable
// without the word.
func ToolLabel(text string) *ToolItem { return &ToolItem{Text: text, Label: true} }

// ToolWidget puts a control of the application's own on the bar — a
// combo box, a search field — where Qt's QToolBar::addWidget and GTK's
// tool items put one. It keeps its own size, is centred in the bar's
// height, draws on the bar's background and takes the focus as any other
// control does; the bar's own arrow keys walk its buttons and step over
// it.
func ToolWidget(c widget.Component) *ToolItem { return &ToolItem{Widget: c} }

// ToolStretch is the free space of a bar: everything after it is pushed
// to the bar's right-hand end. A bar with one fills the width it is
// given instead of only the width of its items, and when its items no
// longer fit it drops the last of the tools and words before the stretch
// rather than letting what comes after fall off the end — the controls a
// bar carries are the application's, and losing one silently is worse
// than showing one tool button fewer.
func ToolStretch() *ToolItem { return &ToolItem{Stretch: true} }

// ToolBar is a horizontal strip of tool buttons.
type ToolBar struct {
	widget.Base
	items  []*ToolItem
	hover  int
	press  int
	focus  int
	keyNav bool
	fades  []stateFade // one per tool: hover cross-fades
}

// NewToolBar constructs a toolbar.
// FocusOnClick is false: a tool button runs its command and leaves focus
// with the document or field (Win32 / Qt toolbars).
func (t *ToolBar) FocusOnClick() bool { return false }

func NewToolBar(items ...*ToolItem) *ToolBar {
	t := &ToolBar{items: items, hover: -1, press: -1, focus: firstTool(items)}
	t.Init(t)
	t.SetWantsFocus(true)
	for _, it := range items {
		if it != nil && it.Widget != nil {
			t.Add(it.Widget)
		}
	}
	return t
}

// isTool reports whether it is a tool button: the bar paints it, hovers
// it, and its arrow keys walk it. A separator, a label, a control and
// the free space are none of those.
func (it *ToolItem) isTool() bool {
	return it != nil && !it.Sep && !it.Stretch && !it.Label && it.Widget == nil
}

// Items returns the tool buttons.
func (t *ToolBar) Items() []*ToolItem { return t.items }

// Tooltip is the hovered tool button's tip, if any.
func (t *ToolBar) Tooltip() string {
	if t.hover >= 0 && t.hover < len(t.items) && t.items[t.hover] != nil {
		return t.items[t.hover].Tip
	}
	return ""
}

// ItemRect is the local box of item i.
func (t *ToolBar) ItemRect(i int) paintengine2d.Rect {
	rects := t.itemRects()
	if i < 0 || i >= len(rects) {
		return paintengine2d.Rect{}
	}
	return rects[i]
}

// KeyboardChrome is true when a tool is painting a keyboard focus ring.
func (t *ToolBar) KeyboardChrome() bool {
	return t.keyNav && t.focus >= 0
}

// ItemCenter is the local midpoint of item i.
func (t *ToolBar) ItemCenter(i int) paintengine2d.Point {
	rects := t.itemRects()
	if i < 0 || i >= len(rects) {
		return paintengine2d.Point{}
	}
	r := rects[i]
	return paintengine2d.Pt((r.Min.X+r.Max.X)*0.5, (r.Min.Y+r.Max.Y)*0.5)
}

// Hover highlights item i (screenshots / tests). i < 0 clears.
func (t *ToolBar) invalidateItem(i int) {
	rects := t.itemRects()
	if i < 0 || i >= len(rects) {
		return
	}
	t.InvalidateRect(rects[i].Inset(-1))
}

func (t *ToolBar) Hover(i int) {
	if t.hover == i {
		return
	}
	old := t.hover
	t.hover = i
	t.invalidateItem(old)
	t.invalidateItem(i)
}

func firstTool(items []*ToolItem) int {
	for i, it := range items {
		if it.isTool() && !it.Disabled {
			return i
		}
	}
	return 0
}

func (t *ToolBar) barH() float32 {
	h := t.Look().Metrics().ToolBarH
	if h <= 0 {
		h = 36
	}
	return h
}

func (t *ToolBar) toolBtnW(h float32) float32 {
	btn := t.Look().Metrics().ToolBtn
	if btn <= 0 {
		btn = h - style.Dip(t.Look(), 6)
		if btn < 8 {
			btn = 8
		}
	}
	return btn
}

func (t *ToolBar) itemBoxW(it *ToolItem, btn float32) float32 {
	if it == nil || it.Sep {
		return 8
	}
	pad, iconSide, iconGap := style.ToolButtonChromeFor(t.Look(), btn)
	if it.Text == "" {
		if it.Icon != style.IconNone {
			w := pad*2 + iconSide
			if w < btn {
				return btn
			}
			return w
		}
		return btn
	}
	textW := t.Look().Font().InkWidth(it.Text)
	if adv := style.ControlFontOf(t.Look(), style.RoleTool).Advance(it.Text); adv > textW {
		textW = adv
	}
	w := pad*2 + textW
	if it.Icon != style.IconNone {
		w += iconSide + iconGap
	}
	return w
}

// labelFont is what a [ToolLabel] is drawn in: the face the look labels
// a tool button with, so a word on the bar and the word under an icon
// are the same text.
func (t *ToolBar) labelFont() *style.Font {
	return style.ControlFontOf(t.Look(), style.RoleTool)
}

// gapAfter is the space the bar leaves after item i. It is the bar's own
// gap everywhere but after a [ToolLabel], which keeps half of one so the
// word sits with the control it names rather than midway between two.
func (t *ToolBar) gapAfter(i int) float32 {
	if i >= 0 && i < len(t.items) && t.items[i] != nil && t.items[i].Label {
		return style.ToolItemGap * 0.5
	}
	return style.ToolItemGap
}

// natural is what each item asks for on a bar btn wide in its buttons:
// a tool button's box, a separator's rule, a control's own measurement,
// and nothing at all for the free space, which takes what is left over.
func (t *ToolBar) natural(btn float32) (w, h []float32) {
	w = make([]float32, len(t.items))
	h = make([]float32, len(t.items))
	for i, it := range t.items {
		switch {
		case it == nil || it.Stretch:
		case it.Widget != nil:
			sz := it.Widget.Measure(layout.Unbounded())
			w[i], h[i] = sz.X, sz.Y
		case it.Label:
			f := t.labelFont()
			w[i], h[i] = f.Advance(it.Text), f.Height()
		case it.Sep:
			w[i], h[i] = 8, 0
		default:
			w[i], h[i] = t.itemBoxW(it, btn), btn
		}
	}
	return w, h
}

func (t *ToolBar) hasStretch() bool {
	for _, it := range t.items {
		if it != nil && it.Stretch {
			return true
		}
	}
	return false
}

// contentW is the intrinsic strip width (items + padding). ToolBar must
// not expand to MaxW: a Row/Flex parent decides growth via Flex weights —
// except that a bar with a [ToolStretch] has been told where its right
// edge is, and takes the width it is offered.
func (t *ToolBar) contentW(h float32) float32 {
	w, _ := t.natural(t.toolBtnW(h))
	in := style.ToolBarInsetsOf(t.Look())
	x := in.Left
	for i := range t.items {
		x += w[i] + t.gapAfter(i)
	}
	return x + in.Right
}

// barHeight is the bar's own height, or as much more as the tallest
// control it carries needs.
func (t *ToolBar) barHeight() float32 {
	h := t.barH()
	in := style.ToolBarInsetsOf(t.Look())
	_, ih := t.natural(t.toolBtnW(h))
	for i, it := range t.items {
		if it != nil && (it.Widget != nil || it.Label) {
			if want := ih[i] + in.Top + in.Bottom; want > h {
				h = want
			}
		}
	}
	return h
}

func (t *ToolBar) Measure(c layout.Constraints) paintengine2d.Point {
	h := t.barHeight()
	w := t.contentW(h)
	if t.hasStretch() && c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (t *ToolBar) Arrange(r paintengine2d.Rect) {
	t.SetBounds(r)
	rects := t.itemRects()
	for i, it := range t.items {
		if it == nil || it.Widget == nil {
			continue
		}
		if i < len(rects) {
			it.Widget.Arrange(rects[i])
		}
	}
}

func (t *ToolBar) itemRects() []paintengine2d.Rect {
	h := t.LocalBounds().Dy()
	btn := t.toolBtnW(h)
	in := style.ToolBarInsetsOf(t.Look())
	w, ih := t.natural(btn)
	y := (h - btn) * 0.5
	if y < 2 {
		y = 2
		btn = h - 4
	}
	total := in.Left + in.Right
	for i := range t.items {
		total += w[i] + t.gapAfter(i)
	}
	out := make([]paintengine2d.Rect, len(t.items))
	stretch, drop := float32(0), map[int]bool{}
	if n := t.stretches(); n > 0 {
		avail := t.LocalBounds().Dx()
		// The last item pays no gap after it: the bar's right inset is
		// where it ends, which is what the free space measures against.
		used := total - t.gapAfter(len(t.items)-1)
		// Too narrow: the tools and the words nearest the free space give
		// way, one at a time, so that what the free space pins to the
		// right edge — the application's own controls — is still whole
		// and still on the bar.
		for used > avail {
			i := t.lastBeforeStretch(drop)
			if i < 0 {
				break
			}
			drop[i] = true
			used -= w[i] + t.gapAfter(i)
		}
		if avail > used {
			stretch = (avail - used) / float32(n)
		}
	}
	x := in.Left
	for i, it := range t.items {
		switch {
		case drop[i]:
			out[i] = paintengine2d.Rect{}
		case it != nil && it.Stretch:
			out[i] = paintengine2d.XYWH(x, y, stretch, 0)
		case it != nil && (it.Widget != nil || it.Label):
			iy := (h - ih[i]) * 0.5
			if iy < 1 {
				iy = 1
			}
			out[i] = paintengine2d.XYWH(x, iy, w[i], min(ih[i], h-2))
		case it == nil || it.Sep:
			out[i] = paintengine2d.XYWH(x, 6, 8, h-12)
		default:
			out[i] = paintengine2d.XYWH(x, y, w[i], btn)
		}
		if it != nil && it.Stretch {
			x += stretch + t.gapAfter(i)
			continue
		}
		if drop[i] {
			continue
		}
		x += w[i] + t.gapAfter(i)
	}
	return out
}

func (t *ToolBar) stretches() int {
	n := 0
	for _, it := range t.items {
		if it != nil && it.Stretch {
			n++
		}
	}
	return n
}

// lastBeforeStretch is the last tool, rule or word ahead of the bar's
// free space that is still on the bar: the first thing to go when the
// bar is narrower than its items. The bar shortens from the right, and
// a control is never what goes.
func (t *ToolBar) lastBeforeStretch(drop map[int]bool) int {
	end := len(t.items)
	for i, it := range t.items {
		if it != nil && it.Stretch {
			end = i
			break
		}
	}
	for i := end - 1; i >= 0; i-- {
		if it := t.items[i]; !drop[i] && (it == nil || it.Widget == nil) {
			return i
		}
	}
	return -1
}

func (t *ToolBar) itemAt(p paintengine2d.Point) int {
	for i, r := range t.itemRects() {
		if t.items[i].isTool() && !r.Empty() && r.Contains(p) {
			return i
		}
	}
	return -1
}

func (t *ToolBar) Paint(ctx *paintengine2d.Context) {
	lk := t.Look()
	if !inMergedCaption(t) {
		// In a title bar the toolkit draws, the caption is the bar.
		lk.DrawToolBar(ctx, t.LocalBounds())
	}
	rects := t.itemRects()
	if len(t.fades) != len(t.items) {
		t.fades = make([]stateFade, len(t.items))
	}
	grouped := style.ToolGroupsOf(lk)
	if grouped {
		t.paintGroups(ctx, rects)
	}
	for i, it := range t.items {
		if it == nil || it.Widget != nil || it.Stretch || rects[i].Empty() {
			continue // a control paints itself; free space and a dropped tool paint nothing
		}
		if it.Label {
			f := t.labelFont()
			b := rects[i]
			show := it.Text
			if f.Advance(show) > b.Dx() {
				show = f.Fit(show, b.Dx())
			}
			f.Draw(ctx, show, paintengine2d.Pt(b.Min.X, b.Min.Y+(b.Dy()-f.Height())*0.5), lk.Palette().Text)
			continue
		}
		if it.Sep {
			if grouped {
				continue // the space between two groups
			}
			b := rects[i]
			x := (b.Min.X + b.Max.X) * 0.5
			ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, 1, b.Dy()), paintengine2d.Fill(lk.Palette().Divider))
			continue
		}
		// The bar's own hover/press cover the whole strip; each tool takes
		// them only from the index under the pointer. Tools in a bar are
		// auto-raise: flat until the pointer is over them.
		st := t.State()&^(style.StateHovered|style.StatePressed) | style.StateAutoRaise
		if i != t.focus || !t.keyNav {
			st &^= style.StateFocused
		}
		if i == t.hover {
			st |= style.StateHovered
		}
		if i == t.press {
			st |= style.StatePressed
		}
		if it.Toggle {
			st |= style.StateToggle
		}
		if it.Down {
			st |= style.StateChecked
		}
		if it.Disabled {
			st |= style.StateDisabled
		}
		r := rects[i]
		t.fades[i].paint(t, ctx, r, st, func(ctx *paintengine2d.Context, st style.ControlState) {
			lk.DrawToolButton(ctx, r, st, it.Text, it.Icon)
		})
	}
}

// paintGroups paints the look's group chrome under each run of buttons
// between separators (macOS Tahoe's glass capsules).
func (t *ToolBar) paintGroups(ctx *paintengine2d.Context, rects []paintengine2d.Rect) {
	lk := t.Look()
	var run paintengine2d.Rect
	n := 0
	flush := func() {
		if n > 0 {
			style.DrawToolGroupOf(lk, ctx, run, n)
		}
		run, n = paintengine2d.Rect{}, 0
	}
	for i, it := range t.items {
		if !it.isTool() || rects[i].Empty() {
			flush()
			continue
		}
		if n == 0 {
			run = rects[i]
		} else {
			run = run.Union(rects[i])
		}
		n++
	}
	flush()
}

func (t *ToolBar) MouseMove(e widget.MouseEvent) bool {
	i := t.itemAt(e.Pos)
	if i != t.hover {
		old := t.hover
		t.hover = i
		t.invalidateItem(old)
		t.invalidateItem(i)
	}
	return true
}

func (t *ToolBar) MouseExit() {
	t.hover = -1
	t.press = -1
	t.Base.MouseExit()
}

func (t *ToolBar) FocusLost() {
	t.keyNav = false
	t.focus = -1
	t.Base.FocusLost()
}

func (t *ToolBar) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() || e.Button == platform.ButtonRight {
		return false
	}
	t.keyNav = false
	t.RequestFocus()
	i := t.itemAt(e.Pos)
	t.press = i
	if i >= 0 {
		t.focus = i
	}
	t.Invalidate()
	return true
}

func (t *ToolBar) MouseRelease(e widget.MouseEvent) bool {
	i := t.itemAt(e.Pos)
	was := t.press
	t.press = -1
	t.Invalidate()
	if was >= 0 && was == i {
		t.activate(i)
	}
	return true
}

func (t *ToolBar) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() || len(t.items) == 0 {
		return false
	}
	t.keyNav = true
	if t.focus < 0 {
		t.focus = firstTool(t.items)
	}
	switch e.Key {
	case platform.KeyLeft:
		t.moveFocus(-1)
		return true
	case platform.KeyRight:
		t.moveFocus(1)
		return true
	case platform.KeyHome:
		t.focus = firstTool(t.items)
		t.Invalidate()
		return true
	case platform.KeyEnd:
		t.focus = lastTool(t.items)
		t.Invalidate()
		return true
	case platform.KeyReturn, platform.KeySpace:
		t.activate(t.focus)
		return true
	}
	return false
}

func lastTool(items []*ToolItem) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].isTool() && !items[i].Disabled {
			return i
		}
	}
	return 0
}

func (t *ToolBar) moveFocus(dir int) {
	if len(t.items) == 0 {
		return
	}
	i := t.focus
	for n := 0; n < len(t.items); n++ {
		i = (i + dir + len(t.items)) % len(t.items)
		it := t.items[i]
		if it.isTool() && !it.Disabled {
			t.focus = i
			t.Invalidate()
			return
		}
	}
}

func (t *ToolBar) activate(i int) {
	if i < 0 || i >= len(t.items) {
		return
	}
	it := t.items[i]
	if !it.isTool() || it.Disabled {
		return
	}
	if it.Toggle {
		it.Down = !it.Down
		t.Invalidate()
	}
	if it.OnClick != nil {
		it.OnClick()
	}
}
