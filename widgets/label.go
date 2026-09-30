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
	Text string
	// Color overrides everything: a literal colour, in device terms,
	// which does not follow the look. Prefer Tone.
	Color paintengine2d.Color
	// Tone is what the text *means* — a warning, an error, a success —
	// resolved against the look's palette at paint time, so it follows a
	// theme change and stays readable on whatever ground the pack uses.
	//
	// It is the answer to a colour that is right in one theme and
	// invisible in another: the palette's status colours are era-faithful
	// and most of them do not reach 4.5:1 as text, so a label takes the
	// ink form of them ([style.Palette.Ink]) rather than the declared
	// one. An application that set Color from the palette itself got the
	// unreadable version, and had to re-set it on every look change.
	Tone  LabelTone
	Align style.Align
	Title bool
	Mono  bool
	// Wrap breaks long lines at spaces to fit the label's width (QLabel's
	// wordWrap, GtkLabel's wrap): the label grows taller instead of
	// eliding. It measures to the width its parent offers.
	Wrap bool
	// MinLines reserves at least this many lines, text at the top, so a
	// layout keeps its place as the text changes (Settings' theme
	// description above its preview).
	MinLines int

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

// For makes the label the caption of c, as Qt's buddy labels are: c is
// named after the label for assistive technology, unless it has a name.
func (l *Label) For(c widget.Component) *Label {
	if nm, ok := c.(interface {
		AccessibleName() string
		SetAccessibleName(string)
	}); ok && nm.AccessibleName() == "" {
		nm.SetAccessibleName(strings.TrimSuffix(strings.TrimSpace(l.Text), ":"))
	}
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
	// [style.Font.Wrap] keeps the text's own newlines, so it lays the
	// whole label out in one pass: greedy at spaces, and a word too long
	// for the width broken rather than left to be elided.
	out := f.Wrap(l.Text, w)
	if out == nil {
		out = []string{""}
	}
	l.wrapKey, l.wrapped = k, out
	return out
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
	n := max(len(lines), l.MinLines)
	return c.Constrain(paintengine2d.Pt(w+2, f.Height()*float32(n)+2))
}

func (l *Label) Arrange(r paintengine2d.Rect) { l.SetBounds(r) }

// MinWidth implements [widget.MinWidther].
//
// A label that wraps has a real floor: its longest word, below which a
// word would have to be broken. A label that does *not* wrap has almost
// none — it is a single line that clips, and giving it less than it
// would like costs the end of the text and nothing else. That is the
// difference the height-based probe cannot see: a one-line label is not
// taller when it is narrowed, so the probe reads it as a thing that
// cannot shrink, and a layout would then keep a whole sentence's width
// for something perfectly happy to be cut short.
//
// So a wrapping label is probed, and a plain one asks for a few pixels:
// enough that it is never squeezed out of existence, far too little to
// hold a layout open.
func (l *Label) MinWidth() float32 {
	if !l.Wrap {
		return style.Dip(l.Look(), 16)
	}
	return widget.MinWidthByProbe(l, l.Measure(layout.Unbounded()))
}

func (l *Label) Paint(ctx *paintengine2d.Context) {
	lk := l.Look()
	f := l.font()
	col := l.Color
	if col == (paintengine2d.Color{}) {
		col = l.Tone.colorIn(lk)
	}
	b := l.LocalBounds()
	lines := l.layoutLines(f, b.Dx()-2)
	th := f.Height()
	y := b.Min.Y + (b.Dy()-th*float32(len(lines)))*0.5
	if l.MinLines > len(lines) {
		y = b.Min.Y + 1
	}
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

// LabelTone is what a label's text means, which decides its colour.
type LabelTone uint8

const (
	// ToneNormal is ordinary text in the palette's text colour.
	ToneNormal LabelTone = iota
	// ToneMuted is secondary text — a hint, a caption, a unit.
	ToneMuted
	// ToneDanger, ToneWarning and ToneSuccess are the three status
	// meanings, in the readable ink form of the palette's status colours
	// rather than the declared ones, which are chosen for fills and are
	// mostly too faint to read as text.
	ToneDanger
	ToneWarning
	ToneSuccess
	// ToneAccent is the palette's accent, lifted to be readable.
	ToneAccent
)

func (t LabelTone) colorIn(lk style.LookAndFeel) paintengine2d.Color {
	p := lk.Palette()
	switch t {
	case ToneMuted:
		return p.TextMuted
	case ToneDanger:
		return p.DangerInk()
	case ToneWarning:
		return p.WarningInk()
	case ToneSuccess:
		return p.SuccessInk()
	case ToneAccent:
		return p.Ink(p.Accent)
	default:
		return p.Text
	}
}
