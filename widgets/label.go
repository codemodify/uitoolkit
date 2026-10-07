package widgets

import (
	"math"
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
	Tone LabelTone
	// Icon is a mark drawn before the text, in the text's own colour and
	// scaled to its height.
	//
	// The toolkit's rule is that a mark is an icon and never a character
	// (docs/icons.md), because the bundled faces carry no arrows, no
	// check and no chevron — a "↑2" written as text draws a tofu box.
	// Until this field existed the only way to obey that rule was to
	// wrap the mark in a button, which is wrong for a status bar or a
	// caption, where nothing is clickable. A label with an icon and no
	// text is just the mark.
	Icon  style.ToolIcon
	Align style.Align
	Title bool
	Mono  bool
	// Bold is the look's bold face at the body's own size: a run-in
	// heading, the way GTK's `heading` class, Qt's QFont::setBold and
	// HTML's <strong> give one.
	//
	// Title is the *page* title's size, a step above the section titles it
	// would sit under, so a line naming the paragraph beneath it had
	// nowhere to go: an application had to draw the line itself with the
	// look's BoldFont and do its own wrapping, which is a label
	// reimplemented to change one thing about the face. A look with no
	// bold face of its own answers with the body face, so a label asking
	// for bold on a pack that has none reads as an ordinary one rather
	// than as a missing font.
	//
	// Title wins where both are set: a title is already the louder of the
	// two.
	Bold bool
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
	if l.Bold {
		return lk.BoldFont()
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
		// At the *narrowest* whole width this box can be handed, not at the
		// fractional width it is offered.
		//
		// Layout rounds each edge of a component's bounds to the nearest
		// pixel (widget.PixelRect), and which way each edge goes depends on
		// where the box sits — so a label offered 248.5 is handed 248 or
		// 249, and 248 is a width it was never measured at. At a fractional
		// scale that is the common case: a column's inner width is
		// fractional, the text is measured for six lines at 248.5 and wraps
		// to seven at 248, and the lines, centred in a box tall enough for
		// six, lose the first and the last. That is the symptom the icon
		// label fix was about, back again at a width nobody measured.
		//
		// Flooring here and rounding the answer up below are conservative in
		// the same direction: wrap for the worst width the box may be, and
		// ask for a width that still holds the longest line.
		wrapW = l.wrapWidth(f, floorPx(c.MaxW))
	}
	lines := l.layoutLines(f, wrapW)
	var w float32
	for _, line := range lines {
		w = max(w, f.Advance(line))
	}
	n := max(len(lines), l.MinLines)
	w += l.iconRoom(f)
	// A measured size is whole pixels, both ways and always.
	//
	// Layout rounds every component's bounds to the pixel grid
	// (widget.PixelRect) and it rounds to the *nearest* pixel — which is
	// right, because that is what makes neighbours that share an edge go on
	// sharing it. The consequence is that a fractional measurement can be
	// handed back smaller than it asked for: a label measured 238.12 wide is
	// given 238, and the longest line, which needed 208.12, no longer fits —
	// it wraps once more and the extra line is pushed out of a box measured
	// without it. The same in the other direction: a box measured 245.3 tall
	// is given 245, and the last line's descender is clipped.
	//
	// So the rule is the widget's, not layout's: measure in whole pixels, and
	// round up, so that rounding to nearest can never take anything away.
	// Grid has followed it for a child sized to its content all along
	// (ceilPx); this is the same rule where the text is.
	return c.Constrain(wholePx(paintengine2d.Pt(w+2, f.Height()*float32(n)+2)))
}

// iconRoom is what the mark takes out of the text's width, or zero.
// floorPx is the narrowest whole pixel width a box of this width may be
// handed once layout has rounded its edges (widget.PixelRect).
func floorPx(v float32) float32 { return float32(math.Floor(float64(v))) }

// wholePx rounds a measured size up to whole pixels on both axes, so that
// layout's round-to-nearest cannot hand the widget less than it measured.
func wholePx(p paintengine2d.Point) paintengine2d.Point {
	return paintengine2d.Pt(ceilPx(p.X), ceilPx(p.Y))
}

func (l *Label) iconRoom(f *style.Font) float32 {
	if l.Icon == style.IconNone {
		return 0
	}
	return l.iconSide(f) + l.iconGap(f)
}

// wrapWidth is the width the text really gets out of a box this wide: less
// the border, and less the room the mark takes.
//
// Measure and Paint have to agree on it exactly. Measure used to wrap at the
// whole width and add the mark's room afterwards, while Paint took the mark's
// room off first and wrapped at what was left — so a label measured for two
// lines painted three, and the lines, centred in a box only tall enough for
// two, lost the first and the last. A sender warning, a signed-and-encrypted
// line, a "Not available: ..." — all of them, at ordinary widths.
func (l *Label) wrapWidth(f *style.Font, avail float32) float32 {
	w := avail - 2 - l.iconRoom(f)
	if w < 4 {
		// The floor Paint elides at: a width with no room for text is not a
		// width to wrap at, and one character per line is not an answer.
		w = 4
	}
	return w
}

// NewIconLabel is a label that leads with a mark — a status bar's ahead
// and behind counts, a caption's lock, a row's severity.
func NewIconLabel(icon style.ToolIcon, text string) *Label {
	l := NewLabel(text)
	l.Icon = icon
	return l
}

func (l *Label) Arrange(r paintengine2d.Rect) { l.SetBounds(r) }

// iconSide and iconGap size the mark from the text it sits beside, not
// from the look's control metrics: a label is whatever height its font
// is, and an icon measured against a toolbar button would tower over a
// status line's small print.
func (l *Label) iconSide(f *style.Font) float32 { return float32(int(f.Height()*0.85 + 0.5)) }

func (l *Label) iconGap(f *style.Font) float32 { return float32(int(f.Height()*0.3 + 0.5)) }

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
	if l.Icon != style.IconNone {
		side, gap := l.iconSide(f), l.iconGap(f)
		// Centred on the first line, so a wrapping label's mark sits
		// beside the text it introduces rather than beside the middle
		// of the paragraph.
		top := b.Min.Y + (min(f.Height(), b.Dy())-side)*0.5
		style.DrawToolIcon(ctx, paintengine2d.XYWH(b.Min.X, top, side, side), l.Icon, col, style.IconSetOf(lk))
		b.Min.X += side + gap
	}
	// The same width Measure wrapped at. b has already lost the mark's room
	// above, so this is the full box again.
	lines := l.layoutLines(f, l.wrapWidth(f, l.LocalBounds().Dx()))
	th := f.Height()
	y := b.Min.Y + (b.Dy()-th*float32(len(lines)))*0.5
	if l.MinLines > len(lines) {
		y = b.Min.Y + 1
	}
	maxW := l.wrapWidth(f, l.LocalBounds().Dx())
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
