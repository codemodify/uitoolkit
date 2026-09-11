package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Label is static text. `\n` is always a hard line break. When Wrap is
// set (or WrapWidth > 0), the body word-wraps to the measure / arrange
// width instead of clipping mid-sentence.
type Label struct {
	widget.Base
	Text      string
	Color     paintengine2d.Color
	Align     style.Align
	Title     bool
	Mono      bool
	Wrap      bool
	WrapWidth float32
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

// WithWrap enables word-wrap at width (0 = use the layout max / arranged box).
func (l *Label) WithWrap(width float32) *Label {
	l.Wrap = true
	l.WrapWidth = width
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

func (l *Label) wrapEnabled() bool { return l.Wrap || l.WrapWidth > 0 }

func (l *Label) wrapMax(limit float32) float32 {
	maxW := limit
	if l.WrapWidth > 0 && (maxW <= 0 || l.WrapWidth < maxW) {
		maxW = l.WrapWidth
	}
	return maxW
}

func (l *Label) lineHeight() float32 {
	return l.font().Height() + 2
}

// VisualLines is the wrapped layout at width (0 = arranged box or WrapWidth).
func (l *Label) VisualLines(width float32) []style.TextLine {
	if width <= 0 {
		if b := l.LocalBounds(); b.Dx() > 2 {
			width = b.Dx() - 2
		} else {
			width = l.wrapMax(0)
		}
	}
	if width <= 0 {
		width = 1e6
	}
	return layoutArea(l.font(), l.Text, width, l.wrapEnabled())
}

func (l *Label) Measure(c layout.Constraints) paintengine2d.Point {
	maxW := float32(0)
	if c.HasMaxW() {
		maxW = c.MaxW - 2
		if maxW < 4 {
			maxW = 4
		}
	}
	maxW = l.wrapMax(maxW)
	lines := layoutArea(l.font(), l.Text, maxW, l.wrapEnabled() && maxW > 0)
	if len(lines) == 0 {
		lines = []style.TextLine{{}}
	}
	f := l.font()
	var w float32
	for _, ln := range lines {
		if adv := f.Advance(ln.Text); adv > w {
			w = adv
		}
	}
	h := float32(len(lines)) * l.lineHeight()
	return c.Constrain(paintengine2d.Pt(w+2, h+2))
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
	width := b.Dx() - 2
	if width < 4 {
		width = 4
	}
	lines := layoutArea(f, l.Text, width, l.wrapEnabled())
	if len(lines) == 0 {
		return
	}
	lh := l.lineHeight()
	total := float32(len(lines)) * lh
	y := b.Min.Y
	if len(lines) == 1 && b.Dy() > lh {
		y = b.Min.Y + (b.Dy()-lh)*0.5
	} else if total < b.Dy() {
		y = b.Min.Y + 1
	}
	for _, ln := range lines {
		tw := f.Advance(ln.Text)
		x := b.Min.X
		switch l.Align {
		case style.AlignCenter:
			x = b.Min.X + (b.Dx()-tw)*0.5
		case style.AlignEnd:
			x = b.Max.X - tw - 2
		}
		if ln.Text != "" {
			f.Draw(ctx, ln.Text, paintengine2d.Pt(x, y), col)
		}
		y += lh
	}
}

func near(a, b paintengine2d.Color) bool {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return dr*dr+dg*dg+db*db < 0.002
}
