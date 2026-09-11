package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Label is static text.
type Label struct {
	widget.Base
	Text     string
	Color    paintengine2d.Color
	Align    style.Align
	Title    bool
	Mono     bool
	wrapHint float32
}

func NewLabel(text string) *Label {
	l := &Label{Text: text}
	l.Init(l)
	return l
}

func NewTitle(text string) *Label {
	l := NewLabel(text)
	l.Title = true
	return l
}

func (l *Label) SetText(s string) {
	if l.Text == s {
		return
	}
	l.Text = s
	l.Invalidate()
}

func (l *Label) font() *style.Font {
	lk := l.Look()
	if l.Title {
		return lk.TitleFont()
	}
	if l.Mono {
		return lk.MonoFont()
	}
	if l.Color != (paintengine2d.Color{}) && near(l.Color, lk.Palette().TextMuted) {
		return lk.MutedFont()
	}
	return lk.Font()
}

func (l *Label) Measure(c layout.Constraints) paintengine2d.Point {
	f := l.font()
	sz := f.Measure(l.Text)
	sz.X += 2
	sz.Y += 2
	return c.Constrain(sz)
}

func (l *Label) Arrange(r paintengine2d.Rect) { l.SetBounds(r) }

func (l *Label) Paint(ctx *paintengine2d.Context) {
	lk := l.Look()
	f := l.font()
	col := l.Color
	if col == (paintengine2d.Color{}) {
		col = lk.Palette().Text
	}
	b := l.LocalBounds()
	tw := f.Advance(l.Text)
	th := f.Height()
	x := b.Min.X
	switch l.Align {
	case style.AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case style.AlignEnd:
		x = b.Max.X - tw - 2
	}
	y := b.Min.Y + (b.Dy()-th)*0.5
	f.Draw(ctx, l.Text, paintengine2d.Pt(x, y), col)
}

func near(a, b paintengine2d.Color) bool {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return dr*dr+dg*dg+db*db < 0.002
}
