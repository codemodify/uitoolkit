package widgets

import (
	"math"
	"strconv"
	"sync"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// RichTextBar is the formatting tool bar for a RichText, as a word
// processor has: the block style (body text or a heading), the text size,
// bold, italic, underline, strikethrough and monospace, text and
// highlight colours, bulleted and numbered lists and their levels,
// alignment, a link, clearing the formatting, and undo and redo. Its
// buttons follow the caret — Bold shows pressed in bold text — and never
// take the keyboard focus from the document, which gets it back after a
// choice in one of the bar's lists. It is optional and separate: put it
// above the editor, in a header bar, or leave it out.
type RichTextBar struct {
	widget.Base
	ed     *RichText
	row    *Wrap
	block  *ComboBox
	size   *ComboBox
	fg, bg *colorTool
	groups []*rtToolGroup
	sync   bool
}

// barBlockKinds are the block style list's entries.
var barBlockKinds = []struct {
	name  string
	kind  richtext.Kind
	level int
}{
	{"Body text", richtext.Paragraph, 0},
	{"Heading 1", richtext.Heading, 1},
	{"Heading 2", richtext.Heading, 2},
	{"Heading 3", richtext.Heading, 3},
	{"Heading 4", richtext.Heading, 4},
}

// barSizes are the size list's entries, in logical pixels; 0 is the
// block's own.
var barSizes = []float32{0, 10, 11, 12, 13, 14, 16, 18, 20, 24, 28, 32, 40}

// NewRichTextBar is a formatting bar for ed.
func NewRichTextBar(ed *RichText) *RichTextBar {
	b := &RichTextBar{ed: ed}
	b.Init(b)
	names := make([]string, len(barBlockKinds))
	for i, k := range barBlockKinds {
		names[i] = k.name
	}
	b.block = NewComboBox(names, 0, func(i int) {
		if b.sync || i < 0 || i >= len(barBlockKinds) {
			return
		}
		k := barBlockKinds[i]
		kind, _ := ed.Document().BlockKind()
		if k.kind == richtext.Paragraph && kind.IsList() {
			// "Body text" on a list item keeps it an item; a heading
			// takes it out of the list.
			ed.RequestFocus()
			return
		}
		ed.SetKind(k.kind, k.level)
		ed.RequestFocus()
	})
	b.block.SetAccessibleName("Paragraph style")
	sizes := make([]string, len(barSizes))
	for i, s := range barSizes {
		if s == 0 {
			sizes[i] = "Auto"
		} else {
			sizes[i] = strconv.Itoa(int(s))
		}
	}
	b.size = NewComboBox(sizes, 0, func(i int) {
		if b.sync || i < 0 || i >= len(barSizes) {
			return
		}
		ed.SetSize(barSizes[i])
		ed.RequestFocus()
	})
	b.size.SetEditable(true)
	b.size.NoCompletion = true
	b.size.OnSubmit = func(s string) {
		if v, err := strconv.ParseFloat(s, 32); err == nil && v >= 4 && v <= 400 {
			ed.SetSize(float32(v))
		} else if s == "" || s == "Auto" {
			ed.SetSize(0)
		}
		ed.RequestFocus()
	}
	b.size.SetAccessibleName("Text size")
	doc := func() *richtext.Doc { return ed.Document() }
	st := func() richtext.Style { return doc().SelectionStyle() }
	kindIs := func(k richtext.Kind) func() bool {
		return func() bool { kk, _ := doc().BlockKind(); return kk == k }
	}
	alignIs := func(a richtext.Align) func() bool {
		return func() bool { return doc().Block(doc().Selection().Caret.Block).Align == a }
	}
	chars := newToolGroup("Character formatting",
		rtTool{glyph: glyphBold, name: "Bold", key: "Ctrl+B", run: ed.ToggleBold, down: func() bool { return st().Bold }},
		rtTool{glyph: glyphItalic, name: "Italic", key: "Ctrl+I", run: ed.ToggleItalic, down: func() bool { return st().Italic }},
		rtTool{glyph: glyphUnderline, name: "Underline", key: "Ctrl+U", run: ed.ToggleUnderline, down: func() bool { return st().Underline }},
		rtTool{glyph: glyphStrike, name: "Strikethrough", key: "Alt+Shift+5", run: ed.ToggleStrike, down: func() bool { return st().Strike }},
		rtTool{glyph: glyphMono, name: "Monospace", run: ed.ToggleMono, down: func() bool { return st().Mono }},
	)
	b.fg = newColorTool(glyphTextColor, paintengine2d.Color{}, func(c paintengine2d.Color) {
		if !b.sync {
			ed.SetColor(c)
			ed.RequestFocus()
		}
	})
	b.fg.SetAccessibleName("Text colour")
	b.bg = newColorTool(glyphHighlight, paintengine2d.RGB(1, 0.92, 0.23), func(c paintengine2d.Color) {
		if !b.sync {
			ed.SetHighlight(c)
			ed.RequestFocus()
		}
	})
	b.bg.SetAccessibleName("Highlight colour")
	lists := newToolGroup("Lists",
		rtTool{glyph: glyphBullets, name: "Bulleted list", key: "Ctrl+Shift+8", run: func() { ed.ToggleList(richtext.Bullet) }, down: kindIs(richtext.Bullet)},
		rtTool{glyph: glyphNumbers, name: "Numbered list", key: "Ctrl+Shift+7", run: func() { ed.ToggleList(richtext.Numbered) }, down: kindIs(richtext.Numbered)},
		rtTool{glyph: glyphOutdent, name: "Decrease indent", key: "Ctrl+[", run: func() { ed.Indent(-1) }, enabled: func() bool { k, _ := doc().BlockKind(); return k.IsList() }},
		rtTool{glyph: glyphIndent, name: "Increase indent", key: "Ctrl+]", run: func() { ed.Indent(1) }, enabled: func() bool { k, _ := doc().BlockKind(); return k.IsList() }},
	)
	align := newToolGroup("Alignment",
		rtTool{glyph: glyphAlignLeft, name: "Align left", key: "Ctrl+Shift+L", run: func() { ed.SetAlign(richtext.AlignLeft) }, down: alignIs(richtext.AlignLeft)},
		rtTool{glyph: glyphAlignCenter, name: "Centre", key: "Ctrl+Shift+E", run: func() { ed.SetAlign(richtext.AlignCenter) }, down: alignIs(richtext.AlignCenter)},
		rtTool{glyph: glyphAlignRight, name: "Align right", key: "Ctrl+Shift+R", run: func() { ed.SetAlign(richtext.AlignRight) }, down: alignIs(richtext.AlignRight)},
	)
	insert := newToolGroup("Insert",
		rtTool{glyph: glyphLink, name: "Link", key: "Ctrl+K", run: ed.RequestLink, down: func() bool { return doc().LinkAt(doc().Selection().Caret) != "" }},
		rtTool{glyph: glyphClear, name: "Clear formatting", key: "Ctrl+\\", run: ed.ClearFormat},
	)
	history := newToolGroup("History",
		rtTool{icon: style.IconUndo, name: "Undo", key: "Ctrl+Z", run: ed.Undo, enabled: func() bool { return doc().CanUndo() }},
		rtTool{icon: style.IconRedo, name: "Redo", key: "Ctrl+Shift+Z", run: ed.Redo, enabled: func() bool { return doc().CanRedo() }},
	)
	b.groups = []*rtToolGroup{chars, lists, align, insert, history}
	for _, g := range b.groups {
		g.ed = ed
	}
	// On a narrow window the bar folds onto a second line rather than
	// running off the edge.
	b.row = NewWrap(widthBox(130, b.block), widthBox(86, b.size), chars, b.fg, b.bg, lists, align, insert, history)
	b.row.Gap, b.row.LineGap = 6, 4
	b.Base.Add(b.row)
	ed.Listen(b.refresh)
	b.refresh()
	return b
}

// Editor is the editor the bar formats.
func (b *RichTextBar) Editor() *RichText { return b.ed }

// refresh brings the bar's lists and buttons in step with the caret.
func (b *RichTextBar) refresh() {
	d := b.ed.Document()
	b.sync = true
	defer func() { b.sync = false }()
	kind, level := d.BlockKind()
	sel := 0
	for i, k := range barBlockKinds {
		if k.kind == kind && (k.kind != richtext.Heading || k.level == level) {
			sel = i
		}
	}
	if b.block.Selected != sel {
		b.block.Select(sel)
	}
	st := d.SelectionStyle()
	si := -1
	for i, s := range barSizes {
		if s == st.Size {
			si = i
		}
	}
	switch {
	case si >= 0 && b.size.Selected != si:
		b.size.Select(si)
	case si < 0:
		b.size.SetText(strconv.FormatFloat(float64(st.Size), 'f', -1, 32))
	}
	if st.Color.A > 0 {
		b.fg.Color = st.Color
	} else {
		b.fg.Color = b.Look().Palette().Text
	}
	for _, g := range b.groups {
		g.Invalidate()
	}
	b.fg.Invalidate()
}

func (b *RichTextBar) barH() float32 {
	h := b.Look().Metrics().ToolBarH
	if h <= 0 {
		h = 36
	}
	return h
}

func (b *RichTextBar) Measure(c layout.Constraints) paintengine2d.Point {
	in := style.ToolBarInsetsOf(b.Look())
	maxW := float32(-1)
	if c.HasMaxW() {
		maxW = max(c.MaxW-in.Left-in.Right, 0)
	}
	sz := b.row.Measure(layout.Constraints{MaxW: maxW, MaxH: -1})
	h := max(b.barH(), sz.Y+in.Top+in.Bottom)
	w := sz.X + in.Left + in.Right
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *RichTextBar) Arrange(r paintengine2d.Rect) {
	b.SetBounds(r)
	in := style.ToolBarInsetsOf(b.Look())
	lb := b.LocalBounds()
	w := max(lb.Dx()-in.Left-in.Right, 0)
	sz := b.row.Measure(layout.Loose(w, lb.Dy()))
	y := (lb.Dy() - sz.Y) * 0.5
	b.row.Arrange(paintengine2d.XYWH(in.Left, float32(math.Round(float64(y))), w, sz.Y))
}

// widthBox holds its child to w logical pixels wide (the bar's lists,
// which would otherwise take a form's width).
type fixedWidth struct {
	widget.Base
	w float32
}

func widthBox(w float32, child widget.Component) *fixedWidth {
	b := &fixedWidth{w: w}
	b.Init(b)
	b.Add(child)
	return b
}

func (b *fixedWidth) Measure(c layout.Constraints) paintengine2d.Point {
	w := style.Dip(b.Look(), b.w)
	if c.HasMaxW() {
		w = min(w, c.MaxW)
	}
	sz := b.Children()[0].Measure(layout.Constraints{MaxW: w, MaxH: c.MaxH})
	return c.Constrain(paintengine2d.Pt(w, sz.Y))
}

func (b *fixedWidth) Arrange(r paintengine2d.Rect) {
	b.SetBounds(r)
	b.Children()[0].Arrange(b.LocalBounds())
}

func (b *RichTextBar) Paint(ctx *paintengine2d.Context) {
	if !inMergedCaption(b) {
		b.Look().DrawToolBar(ctx, b.LocalBounds())
	}
}

// Describe: the bar is a tool bar.
func (b *RichTextBar) Describe(n *a11y.Node) {
	n.Role = a11y.RoleToolBar
	if n.Name == "" {
		n.Name = "Formatting"
	}
}

// CaptionAt: the bar's own space moves a window when it is a title bar.
func (b *RichTextBar) CaptionAt(paintengine2d.Point) bool { return true }

// ---- tool groups -------------------------------------------------------

// rtGlyph is a picture a format button draws for itself.
type rtGlyph uint8

const (
	glyphNone rtGlyph = iota
	glyphBold
	glyphItalic
	glyphUnderline
	glyphStrike
	glyphMono
	glyphBullets
	glyphNumbers
	glyphOutdent
	glyphIndent
	glyphAlignLeft
	glyphAlignCenter
	glyphAlignRight
	glyphLink
	glyphClear
	glyphTextColor
	glyphHighlight
)

// rtTool is one button of a tool group.
type rtTool struct {
	glyph   rtGlyph
	icon    style.ToolIcon
	name    string
	key     string
	run     func()
	down    func() bool
	enabled func() bool
}

func (t rtTool) isDown() bool    { return t.down != nil && t.down() }
func (t rtTool) isEnabled() bool { return t.enabled == nil || t.enabled() }

// rtToolGroup is a run of format buttons: flat until the pointer is over
// them, pressed while their attribute is on, walked with the arrow keys.
type rtToolGroup struct {
	widget.Base
	name   string
	tools  []rtTool
	ed     *RichText
	hover  int
	press  int
	focus  int
	keyNav bool
}

func newToolGroup(name string, tools ...rtTool) *rtToolGroup {
	g := &rtToolGroup{name: name, tools: tools, hover: -1, press: -1}
	g.Init(g)
	g.SetWantsFocus(true)
	return g
}

// FocusOnClick is false: a click formats the document and leaves the
// keyboard with it.
func (g *rtToolGroup) FocusOnClick() bool { return false }

func (g *rtToolGroup) side() float32 { return toolSide(g.Look()) }

// toolSide is a format button's side in lk: the look's tool button.
func toolSide(lk style.LookAndFeel) float32 {
	btn := lk.Metrics().ToolBtn
	if btn <= 0 {
		h := lk.Metrics().ToolBarH
		if h <= 0 {
			h = 36
		}
		btn = h - style.Dip(lk, 6)
	}
	return float32(math.Round(float64(max(btn, style.Dip(lk, 20)))))
}

func (g *rtToolGroup) Measure(c layout.Constraints) paintengine2d.Point {
	s := g.side()
	n := float32(len(g.tools))
	return c.Constrain(paintengine2d.Pt(n*s+(n-1)*style.ToolItemGap, s))
}

func (g *rtToolGroup) Arrange(r paintengine2d.Rect) { g.SetBounds(r) }

func (g *rtToolGroup) rect(i int) paintengine2d.Rect {
	s := g.side()
	return paintengine2d.XYWH(float32(i)*(s+style.ToolItemGap), (g.LocalBounds().Dy()-s)*0.5, s, s)
}

func (g *rtToolGroup) indexAt(p paintengine2d.Point) int {
	for i := range g.tools {
		if g.rect(i).Contains(p) {
			return i
		}
	}
	return -1
}

func (g *rtToolGroup) state(i int) style.ControlState {
	st := g.State()&^(style.StateHovered|style.StatePressed|style.StateFocused) | style.StateAutoRaise | style.StateToggle
	t := g.tools[i]
	if i == g.hover {
		st |= style.StateHovered
	}
	if i == g.press && i == g.hover {
		st |= style.StatePressed
	}
	if t.isDown() {
		st |= style.StateChecked
	}
	if !t.isEnabled() || !g.ed.editable() {
		st |= style.StateDisabled
	}
	if g.Focused() && g.keyNav && i == g.focus {
		st |= style.StateFocused
	}
	return st
}

// glyphColors remembers, per look and state, the colour a glyph drawn on
// a tool face takes.
var glyphColors = struct {
	mu sync.Mutex
	m  map[glyphKey]paintengine2d.Color
}{m: map[glyphKey]paintengine2d.Color{}}

type glyphKey struct {
	lk style.LookAndFeel
	st style.ControlState
}

// glyphColor is the colour a picture drawn on the look's tool face in
// state st takes. Every engine picks its own label colour for a tool
// button — System 7 inverts a pressed one, Luna tints it — so the face is
// painted once off screen and the glyph takes whichever of the look's
// text colours reads best on it.
func glyphColor(lk style.LookAndFeel, st style.ControlState) paintengine2d.Color {
	p := lk.Palette()
	if st.Disabled() {
		return p.TextMuted
	}
	k := glyphKey{lk, st}
	glyphColors.mu.Lock()
	defer glyphColors.mu.Unlock()
	if c, ok := glyphColors.m[k]; ok {
		return c
	}
	const side = 28
	img := paintengine2d.NewImage(side, side)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	lk.DrawToolButton(ctx, paintengine2d.XYWH(0, 0, side, side), st, "", style.IconNone)
	r, g, b, a := img.PremulAt(side/2, side/2)
	col := p.Text
	if a > 128 {
		face := paintengine2d.RGB(float32(r)/float32(a), float32(g)/float32(a), float32(b)/float32(a))
		col = style.ReadableOn(face, 4.5, p.Text, p.TextOnAccent, paintengine2d.RGB(0, 0, 0), paintengine2d.RGB(1, 1, 1))
	}
	if len(glyphColors.m) > 512 {
		clear(glyphColors.m)
	}
	glyphColors.m[k] = col
	return col
}

func (g *rtToolGroup) Paint(ctx *paintengine2d.Context) {
	lk := g.Look()
	if style.ToolGroupsOf(lk) && len(g.tools) > 0 {
		style.DrawToolGroupOf(lk, ctx, g.rect(0).Union(g.rect(len(g.tools)-1)), len(g.tools))
	}
	for i, t := range g.tools {
		r := g.rect(i)
		st := g.state(i)
		if t.icon != style.IconNone {
			lk.DrawToolButton(ctx, r, st, "", t.icon)
			continue
		}
		lk.DrawToolButton(ctx, r, st, "", style.IconNone)
		drawGlyph(ctx, lk, r, t.glyph, glyphColor(lk, st))
	}
}

// drawGlyph draws a format button's picture in b.
func drawGlyph(ctx *paintengine2d.Context, lk style.LookAndFeel, b paintengine2d.Rect, gl rtGlyph, col paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	u := s / 20 // a twentieth of the button
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	lw := max(1, float32(math.Round(float64(u*1.3))))
	fill := paintengine2d.Fill(col)
	letter := func(ch string, bold bool, slant bool) (x, top, w, h float32, f *style.Font) {
		w8 := style.WeightRegular
		if bold {
			w8 = style.WeightBold
		}
		fam := lk.Font().Family
		f = style.BakeFamily(fam, w8, float32(math.Round(float64(s*0.56))), col)
		w, h = f.Advance(ch), f.Ascent
		x = float32(math.Round(float64(cx - w*0.5)))
		top = float32(math.Round(float64(cy - f.Height()*0.5)))
		if slant {
			ctx.Save()
			base := top + f.Ascent
			ctx.Transform(paintengine2d.Matrix{A: 1, C: -0.22, D: 1, E: 0.22 * base})
			f.Draw(ctx, ch, paintengine2d.Pt(x, top), col)
			ctx.Restore()
		} else {
			f.Draw(ctx, ch, paintengine2d.Pt(x, top), col)
		}
		return
	}
	hline := func(x0, y, x1 float32) {
		ctx.DrawRect(paintengine2d.XYWH(float32(math.Round(float64(x0))), float32(math.Round(float64(y))), float32(math.Round(float64(x1-x0))), lw), fill)
	}
	switch gl {
	case glyphBold:
		letter("B", true, false)
	case glyphItalic:
		letter("I", false, true)
	case glyphUnderline:
		x, top, w, _, f := letter("U", false, false)
		hline(x-u, top+f.Ascent+u*2, x+w+u)
	case glyphStrike:
		x, top, w, _, f := letter("S", false, false)
		hline(x-u*2, top+f.Ascent*0.62, x+w+u*2)
	case glyphMono:
		f := style.BakeFamily(lk.MonoFont().Family, style.WeightRegular, float32(math.Round(float64(s*0.42))), col)
		t := "</>"
		f.Draw(ctx, t, paintengine2d.Pt(float32(math.Round(float64(cx-f.Advance(t)*0.5))), float32(math.Round(float64(cy-f.Height()*0.5)))), col)
	case glyphBullets, glyphNumbers:
		for k := 0; k < 3; k++ {
			y := cy + float32(k-1)*u*5
			if gl == glyphBullets {
				ctx.DrawCircle(paintengine2d.Pt(cx-u*6, y), u*1.3, fill)
			} else {
				f := style.BakeFamily(lk.Font().Family, style.WeightBold, float32(math.Round(float64(s*0.3))), col)
				n := strconv.Itoa(k + 1)
				f.Draw(ctx, n, paintengine2d.Pt(float32(math.Round(float64(cx-u*6-f.Advance(n)*0.5))), float32(math.Round(float64(y-f.Height()*0.5)))), col)
			}
			hline(cx-u*3, y-lw*0.5, cx+u*7)
		}
	case glyphOutdent, glyphIndent:
		for k := 0; k < 4; k++ {
			y := cy + (float32(k)-1.5)*u*4
			x0 := cx - u*7
			if k == 1 || k == 2 {
				x0 = cx - u*1
			}
			hline(x0, y-lw*0.5, cx+u*7)
		}
		p := paintengine2d.NewPath()
		tx := cx - u*7
		if gl == glyphIndent {
			p.MoveTo(tx, cy-u*3)
			p.LineTo(tx+u*4, cy)
			p.LineTo(tx, cy+u*3)
		} else {
			p.MoveTo(tx+u*4, cy-u*3)
			p.LineTo(tx, cy)
			p.LineTo(tx+u*4, cy+u*3)
		}
		p.Close()
		ctx.DrawPath(p, fill)
	case glyphAlignLeft, glyphAlignCenter, glyphAlignRight:
		widths := []float32{14, 9, 14, 9}
		for k, w := range widths {
			y := cy + (float32(k)-1.5)*u*4
			x0 := cx - u*7
			switch gl {
			case glyphAlignCenter:
				x0 = cx - u*w*0.5
			case glyphAlignRight:
				x0 = cx + u*7 - u*w
			}
			hline(x0, y-lw*0.5, x0+u*w)
		}
	case glyphLink:
		for _, dx := range []float32{-3.2, 3.2} {
			r := paintengine2d.XYWH(cx+dx*u-u*4.5, cy-u*2.6, u*9, u*5.2)
			ctx.DrawRoundRect(r, u*2.6, u*2.6, paintengine2d.StrokePaint(col, lw))
		}
	case glyphTextColor:
		letter("A", false, false)
	case glyphHighlight:
		// A marker's nib.
		p := paintengine2d.NewPath()
		p.MoveTo(cx-u*5, cy+u*3)
		p.LineTo(cx+u*1, cy-u*6)
		p.LineTo(cx+u*5, cy-u*3)
		p.LineTo(cx-u*1, cy+u*6)
		p.Close()
		ctx.DrawPath(p, paintengine2d.StrokePaint(col, lw))
		q := paintengine2d.NewPath()
		q.MoveTo(cx-u*5, cy+u*3)
		q.LineTo(cx-u*1, cy+u*6)
		q.LineTo(cx-u*6, cy+u*7)
		q.Close()
		ctx.DrawPath(q, fill)
	case glyphClear:
		x, top, w, _, f := letter("T", false, false)
		ctx.DrawLine(paintengine2d.Pt(x-u, top+f.Ascent+u), paintengine2d.Pt(x+w+u*2, top+u), paintengine2d.StrokePaint(col, lw))
	}
}

func (g *rtToolGroup) invalidateTool(i int) {
	if i >= 0 && i < len(g.tools) {
		g.InvalidateRect(g.rect(i).Inset(-2))
	}
}

func (g *rtToolGroup) MouseMove(e widget.MouseEvent) bool {
	if i := g.indexAt(e.Pos); i != g.hover {
		g.invalidateTool(g.hover)
		g.hover = i
		g.invalidateTool(i)
	}
	return true
}

func (g *rtToolGroup) MouseExit() {
	g.invalidateTool(g.hover)
	g.hover, g.press = -1, -1
	g.Base.MouseExit()
}

func (g *rtToolGroup) MousePress(e widget.MouseEvent) bool {
	i := g.indexAt(e.Pos)
	if i < 0 || e.Button != platform.ButtonLeft {
		return false
	}
	g.press = i
	g.keyNav = false
	g.invalidateTool(i)
	return true
}

func (g *rtToolGroup) MouseRelease(e widget.MouseEvent) bool {
	i := g.press
	g.press = -1
	g.invalidateTool(i)
	if i >= 0 && i == g.indexAt(e.Pos) {
		g.activate(i)
	}
	return true
}

// activate runs tool i.
func (g *rtToolGroup) activate(i int) bool {
	if i < 0 || i >= len(g.tools) {
		return false
	}
	t := g.tools[i]
	if !t.isEnabled() || !g.ed.editable() || t.run == nil {
		return false
	}
	t.run()
	g.Invalidate()
	return true
}

func (g *rtToolGroup) KeyPress(e widget.KeyEvent) bool {
	n := len(g.tools)
	switch e.Key {
	case platform.KeyLeft:
		g.focus = (g.focus + n - 1) % n
	case platform.KeyRight:
		g.focus = (g.focus + 1) % n
	case platform.KeyHome:
		g.focus = 0
	case platform.KeyEnd:
		g.focus = n - 1
	case platform.KeySpace, platform.KeyReturn:
		g.activate(g.focus)
		return true
	default:
		return false
	}
	g.keyNav = true
	g.Invalidate()
	return true
}

func (g *rtToolGroup) MarkKeyboardFocus() {
	g.keyNav = true
	g.Base.MarkKeyboardFocus()
}

// Tooltip names the hovered button and its shortcut.
func (g *rtToolGroup) Tooltip() string {
	if g.hover < 0 || g.hover >= len(g.tools) {
		return ""
	}
	t := g.tools[g.hover]
	if t.key != "" {
		return t.name + " (" + t.key + ")"
	}
	return t.name
}

// Describe: a group of buttons named for what they format.
func (g *rtToolGroup) Describe(n *a11y.Node) {
	n.Role = a11y.RoleGroup
	if n.Name == "" {
		n.Name = g.name
	}
}

// AccessibleItems are the buttons: toggles for the attributes that stay
// on, push buttons for the rest.
func (g *rtToolGroup) AccessibleItems() []*a11y.Node {
	out := make([]*a11y.Node, len(g.tools))
	o := widget.DeviceOrigin(g)
	for i, t := range g.tools {
		n := &a11y.Node{ID: widget.ItemID(g, i), Role: a11y.RoleButton, Name: t.name, Shortcut: t.key,
			Bounds: g.rect(i).Translate(o), Index: i + 1, Count: len(g.tools)}
		if t.down != nil {
			n.Role = a11y.RoleToggleButton
			if t.isDown() {
				n.State |= a11y.StatePressed
			}
		}
		if !t.isEnabled() {
			n.State |= a11y.StateDisabled
		}
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out[i] = n
	}
	return out
}

// AccessibleFocusItem is the button the keyboard is on.
func (g *rtToolGroup) AccessibleFocusItem() int {
	if g.Focused() {
		return g.focus
	}
	return -1
}

// AccessibleAction presses a button.
func (g *rtToolGroup) AccessibleAction(item int, act a11y.Action) bool {
	if act != a11y.ActionDefault {
		return false
	}
	return g.activate(item)
}

// ---- colour tools ------------------------------------------------------

// colorTool is a tool button for a colour: a glyph over a bar of the
// colour it applies, dropping the colour picker down (ColorButton's).
type colorTool struct {
	ColorButton
	glyph rtGlyph
	hover bool
}

func newColorTool(gl rtGlyph, col paintengine2d.Color, on func(paintengine2d.Color)) *colorTool {
	c := &colorTool{glyph: gl}
	c.Color, c.OnChange = col, on
	c.Init(c)
	c.SetWantsFocus(true)
	return c
}

// FocusOnClick is false: the picker takes the keyboard while it is open,
// and the document has it back after.
func (c *colorTool) FocusOnClick() bool { return false }

func (c *colorTool) side() float32 { return toolSide(c.Look()) }

func (c *colorTool) Measure(cons layout.Constraints) paintengine2d.Point {
	s := c.side()
	return cons.Constrain(paintengine2d.Pt(s, s))
}

func (c *colorTool) MouseEnter() { c.hover = true; c.Base.MouseEnter() }
func (c *colorTool) MouseExit()  { c.hover = false; c.Base.MouseExit() }

// Tooltip names the tool.
func (c *colorTool) Tooltip() string { return c.AccessibleName() }

func (c *colorTool) Paint(ctx *paintengine2d.Context) {
	lk := c.Look()
	b := c.LocalBounds()
	s := c.side()
	r := paintengine2d.XYWH(0, (b.Dy()-s)*0.5, s, s)
	st := c.State()&^style.StateHovered | style.StateAutoRaise
	if c.hover {
		st |= style.StateHovered
	}
	if c.open {
		st |= style.StatePressed
	}
	if !c.KeyNav() {
		st &^= style.StateFocused
	}
	lk.DrawToolButton(ctx, r, st, "", style.IconNone)
	fg := glyphColor(lk, st&^style.StateFocused)
	u := s / 20
	bar := paintengine2d.XYWH(r.Min.X+u*4, r.Max.Y-u*6, s-u*8, max(u*3, 2))
	sw := c.Color
	if sw.A == 0 {
		sw = fg
	}
	ctx.DrawRect(bar, paintengine2d.Fill(sw))
	ctx.DrawRect(bar.Inset(-0.5), paintengine2d.StrokePaint(fg.WithAlpha(0.35), 1))
	glyphBox := paintengine2d.XYWH(r.Min.X, r.Min.Y-u*2.5, s, s)
	drawGlyph(ctx, lk, glyphBox, c.glyph, fg)
}
