package widgets

import (
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Label is static text. A newline starts another line (QLabel, GtkLabel);
// each line is aligned and, when too long, elided on its own.
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

// lines splits the text at its newlines.
func (l *Label) lines() []string {
	if !strings.Contains(l.Text, "\n") {
		return []string{l.Text}
	}
	return strings.Split(strings.ReplaceAll(l.Text, "\r\n", "\n"), "\n")
}

func (l *Label) Measure(c layout.Constraints) paintengine2d.Point {
	f := l.font()
	lines := l.lines()
	var w float32
	for _, line := range lines {
		w = max(w, f.Advance(line))
	}
	return c.Constrain(paintengine2d.Pt(w+2, f.Height()*float32(len(lines))+2))
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
	lines := l.lines()
	th := f.Height()
	y := b.Min.Y + (b.Dy()-th*float32(len(lines)))*0.5
	maxW := b.Dx() - 2
	if maxW < 4 {
		maxW = 4
	}
	ctx.Save()
	ctx.ClipRect(b)
	for _, show := range lines {
		if f.Advance(show) > maxW {
			show = f.Fit(show, maxW)
		}
		tw := f.Advance(show)
		x := b.Min.X
		switch l.Align {
		case style.AlignCenter:
			x = b.Min.X + (b.Dx()-tw)*0.5
		case style.AlignEnd:
			x = b.Max.X - tw - 2
		}
		f.Draw(ctx, show, paintengine2d.Pt(x, y), col)
		y += th
	}
	ctx.Restore()
}

func near(a, b paintengine2d.Color) bool {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return dr*dr+dg*dg+db*db < 0.002
}
