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
	Text  string
	Color paintengine2d.Color
	Align style.Align
	Title bool
	Mono  bool
	// Wrap breaks long lines at spaces to fit the label's width (QLabel's
	// wordWrap, GtkLabel's wrap): the label grows taller instead of
	// eliding. It measures to the width its parent offers.
	Wrap bool

	wrapKey labelWrapKey
	wrapped []string
}

// labelWrapKey is what a wrapped layout depends on.
type labelWrapKey struct {
	text string
	w    float32
	f    *style.Font
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

// layoutLines is the text's lines at width w: its newlines, and with Wrap
// the breaks that fit w (w <= 0: unbounded).
func (l *Label) layoutLines(f *style.Font, w float32) []string {
	if !l.Wrap || w <= 0 {
		return l.lines()
	}
	k := labelWrapKey{l.Text, w, f}
	if l.wrapped != nil && l.wrapKey == k {
		return l.wrapped
	}
	var out []string
	for _, para := range l.lines() {
		out = append(out, wrapText(f, para, w)...)
	}
	l.wrapKey, l.wrapped = k, out
	return out
}

// wrapText breaks s at spaces into lines no wider than w; a word wider
// than w keeps a line of its own (elided when painted).
func wrapText(f *style.Font, s string, w float32) []string {
	if f.Advance(s) <= w {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	line := words[0]
	for _, word := range words[1:] {
		if cand := line + " " + word; f.Advance(cand) <= w {
			line = cand
			continue
		}
		out = append(out, line)
		line = word
	}
	return append(out, line)
}

func (l *Label) Measure(c layout.Constraints) paintengine2d.Point {
	f := l.font()
	wrapW := float32(-1)
	if l.Wrap && c.HasMaxW() {
		wrapW = c.MaxW - 2
	}
	lines := l.layoutLines(f, wrapW)
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
	lines := l.layoutLines(f, b.Dx()-2)
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
