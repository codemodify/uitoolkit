package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// CardBadge is a colored chip on a CardList row (tags).
type CardBadge struct {
	Label string
	Color paintengine2d.Color
}

// CardContent is one multi-line card (Thunderbird-style message row).
type CardContent struct {
	Title    string // correspondent
	Subtitle string // subject
	Meta     string // date / size
	Snippet  string // body preview
	Badges   []CardBadge
	Bold     bool
	Starred  bool
}

// CardList is a virtualized multi-line card list. Gallery and Mail share it.
type CardList struct {
	widget.Base
	Count      int
	CardHeight float32
	Selected   int
	Card       func(i int) CardContent
	OnSelect   func(i int)
	OnContext  func(i int, windowPos paintengine2d.Point)
	OffsetY    float32
	hovered    int
	vbar       scrollDrag
	rows       rowSceneCache
}

// NewCardList builds a card list. selected starts at -1.
func NewCardList(count int, card func(int) CardContent, on func(int)) *CardList {
	l := &CardList{Count: count, CardHeight: 68, Selected: -1, Card: card, OnSelect: on, hovered: -1}
	l.Init(l)
	l.SetWantsFocus(true)
	return l
}

func (l *CardList) Measure(c layout.Constraints) paintengine2d.Point {
	h := float32(l.Count) * l.rowH()
	if h < l.rowH()*3 {
		h = l.rowH() * 3
	}
	if c.HasMaxH() && h > c.MaxH {
		h = c.MaxH
	}
	w := float32(240)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (l *CardList) Arrange(r paintengine2d.Rect) { l.SetBounds(r); l.clamp() }

func (l *CardList) rowH() float32 {
	h := l.CardHeight
	if h <= 0 {
		h = 68
	}
	lk := l.Look()
	if lk == nil {
		return h
	}
	font := float32(16)
	if f := lk.Font(); f != nil {
		font = f.Height()
	}
	pad := lk.Metrics().RowPad
	if pad <= 0 {
		pad = 6
	}
	min := font*3 + pad*3 + 12
	if min > h {
		h = min
	}
	if style.LookScale(lk) > 1.01 {
		if scaled := style.Dip(lk, l.CardHeight); scaled > h {
			h = scaled
		}
	}
	return h
}

func (l *CardList) contentH() float32 { return float32(l.Count) * l.rowH() }

// MaxOffset is max(0, content − viewport).
func (l *CardList) MaxOffset() float32 {
	return layout.MaxScroll(l.contentH(), l.LocalBounds().Dy())
}

func (l *CardList) clamp() {
	l.OffsetY = layout.ClampScroll(l.OffsetY, l.contentH(), l.LocalBounds().Dy())
}

func (l *CardList) vparts() style.ScrollParts {
	return vScrollParts(l.Look(), l.LocalBounds(), l.contentH(), l.OffsetY)
}

func (l *CardList) vaxis() scrollAxis {
	return scrollAxis{
		vertical: true,
		parts:    l.vparts,
		get:      func() (float32, float32) { return l.OffsetY, l.MaxOffset() },
		set:      func(y float32) { l.OffsetY = y; l.clamp(); l.Invalidate() },
		steps:    func() (float32, float32) { return l.rowH(), l.LocalBounds().Dy() * 0.9 },
	}
}

// rowsW is the row width: the view minus the gutter a visible bar takes.
func (l *CardList) rowsW() float32 {
	return l.LocalBounds().Dx() - scrollGutter(l.Look(), l.MaxOffset() > 0)
}

func (l *CardList) scrollTrack() (track, thumb paintengine2d.Rect) {
	sp := l.vparts()
	return sp.Track, sp.Thumb
}

// VisibleRange is the half-open [lo, hi) window of cards that Paint draws.
func (l *CardList) VisibleRange() (lo, hi int) { return l.visibleRange() }

// ScrollTrack is the overflow bar geometry (empty thumb when content fits).
func (l *CardList) ScrollTrack() (track, thumb paintengine2d.Rect) { return l.scrollTrack() }

// ScrollTo sets OffsetY (clamped) without requiring a wheel event.
func (l *CardList) ScrollTo(y float32) {
	l.OffsetY = y
	l.clamp()
	l.Invalidate()
}

func (l *CardList) visibleRange() (lo, hi int) {
	rh := l.rowH()
	if rh <= 0 {
		return 0, 0
	}
	lo = int(l.OffsetY / rh)
	hi = int((l.OffsetY+l.LocalBounds().Dy())/rh) + 1
	if lo < 0 {
		lo = 0
	}
	if hi > l.Count {
		hi = l.Count
	}
	return
}

func (l *CardList) cardAt(i int) CardContent {
	if l.Card == nil || i < 0 || i >= l.Count {
		return CardContent{}
	}
	return l.Card(i)
}

func (l *CardList) Paint(ctx *paintengine2d.Context) {
	l.clamp()
	b := l.LocalBounds()
	lk := l.Look()
	rw := l.rowsW()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
	rh := l.rowH()
	lo, hi := l.visibleRange()
	if rec, ok := ctx.Device().(*paintengine2d.Recorder); ok {
		o := rowOrigin(ctx)
		l.rows.ready(o.X, o.Y, rw, rh, lookSig(lk))
		recordScrollingRows(rec, ctx, &l.rows, l.ID()^(3<<32), b, rw, rh, l.OffsetY, 0, lo, hi,
			func(i int) uint64 { return l.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 {
				c := l.cardAt(i)
				sig := newRowSig(i == l.Selected, i == l.hovered, bits32(rw))
				for _, part := range cardSigParts(c) {
					sig.str(part)
				}
				return sig.sum()
			},
			func(i int) {
				paintCard(lk, ctx, paintengine2d.XYWH(0, 0, rw, rh), l.cardAt(i), i == l.Selected, i == l.hovered)
			},
		)
	} else {
		ctx.Save()
		ctx.ClipRect(b)
		for i := lo; i < hi; i++ {
			y := float32(i)*rh - l.OffsetY
			paintCard(lk, ctx, paintengine2d.XYWH(0, y, rw, rh), l.cardAt(i), i == l.Selected, i == l.hovered)
		}
		ctx.Restore()
	}
	l.vbar.paint(ctx, lk, l.vparts(), true)
	if l.Focused() {
		lk.DrawFocusRing(ctx, b)
	}
}

func cardSigParts(c CardContent) []string {
	parts := []string{c.Title, c.Subtitle, c.Meta, c.Snippet}
	if c.Starred {
		parts = append(parts, "★")
	}
	if c.Bold {
		parts = append(parts, "B")
	}
	for _, b := range c.Badges {
		parts = append(parts, b.Label)
	}
	return parts
}

func paintCard(lk style.LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, c CardContent, selected, hovered bool) {
	p := lk.Palette()
	m := lk.Metrics()
	if selected {
		ctx.DrawRoundRect(b.Inset(2), 6, 6, paintengine2d.Fill(p.Accent.WithAlpha(0.26)))
	} else if hovered {
		ctx.DrawRoundRect(b.Inset(2), 6, 6, paintengine2d.Fill(p.Highlight))
	}
	pad := m.RowPad + 6
	if pad < 8 {
		pad = 8
	}
	inner := b.Inset(pad)
	titleFace := lk.Font()
	if c.Bold && lk.BoldFont() != nil {
		titleFace = lk.BoldFont()
	}
	muted := lk.MutedFont()
	if muted == nil {
		muted = lk.Font()
	}
	x := inner.Min.X
	y := inner.Min.Y
	if c.Starred {
		lk.Font().Draw(ctx, "★", paintengine2d.Pt(x, y), p.Accent)
		x += 16
	}
	metaW := float32(0)
	if c.Meta != "" && muted != nil {
		metaW = muted.Advance(c.Meta)
		muted.Draw(ctx, c.Meta, paintengine2d.Pt(inner.Max.X-metaW, y), p.TextMuted)
	}
	title := c.Title
	if title == "" {
		title = "(no sender)"
	}
	avail := inner.Max.X - x - metaW - 10
	if avail < 20 {
		avail = 20
	}
	title = titleFace.Fit(title, avail)
	titleFace.Draw(ctx, title, paintengine2d.Pt(x, y), p.Text)
	y += titleFace.Height() + 2

	subFace := titleFace
	sub := c.Subtitle
	if sub == "" {
		sub = "(no subject)"
	}
	sub = subFace.Fit(sub, inner.Dx())
	subFace.Draw(ctx, sub, paintengine2d.Pt(inner.Min.X, y), p.Text)
	y += subFace.Height() + 2

	if c.Snippet != "" && muted != nil {
		snip := muted.Fit(c.Snippet, inner.Dx())
		muted.Draw(ctx, snip, paintengine2d.Pt(inner.Min.X, y), p.TextMuted)
		y += muted.Height() + 3
	}
	bx := inner.Min.X
	bh := float32(16)
	if y+bh > inner.Max.Y {
		return
	}
	for _, badge := range c.Badges {
		if badge.Label == "" {
			continue
		}
		col := badge.Color
		if col == (paintengine2d.Color{}) {
			col = p.Accent
		}
		tw := muted.Advance(badge.Label) + 12
		if bx+tw > inner.Max.X {
			break
		}
		chip := paintengine2d.XYWH(bx, y, tw, bh)
		ctx.DrawRoundRect(chip, 8, 8, paintengine2d.Fill(col.WithAlpha(0.85)))
		fg := p.TextOnAccent
		lk.OnAccentFont().Draw(ctx, badge.Label, paintengine2d.Pt(chip.Min.X+6, chip.Min.Y+(bh-lk.OnAccentFont().Height())*0.5), fg)
		bx += tw + 6
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+8, b.Max.Y-1, b.Dx()-16, 1), paintengine2d.Fill(p.Divider.WithAlpha(0.55)))
}

func (l *CardList) indexAt(y float32) int {
	i := int((y + l.OffsetY) / l.rowH())
	if i < 0 || i >= l.Count {
		return -1
	}
	return i
}

func (l *CardList) rowRect(i int) paintengine2d.Rect {
	if i < 0 {
		return paintengine2d.Rect{}
	}
	rh := l.rowH()
	y := float32(i)*rh - l.OffsetY
	return paintengine2d.XYWH(0, y, l.LocalBounds().Dx(), rh)
}

// Invalidate drops retained card scenes so Starred / text changes
// repaint on the next frame.
func (l *CardList) Invalidate() {
	l.rows.reset()
	l.Base.Invalidate()
}

func (l *CardList) invalidateRow(i int) {
	if r := l.rowRect(i); !r.Empty() {
		l.InvalidateRect(r.Inset(-1))
	}
}

func (l *CardList) MouseEnter() {}

func (l *CardList) MouseMove(e widget.MouseEvent) bool {
	if handled, dirty := l.vbar.move(e.Pos, l.vaxis()); handled || dirty {
		if dirty {
			l.Invalidate()
		}
		if handled {
			return true
		}
	}
	h := l.indexAt(e.Pos.Y)
	if h != l.hovered {
		old := l.hovered
		l.hovered = h
		l.invalidateRow(old)
		l.invalidateRow(h)
	}
	return true
}

func (l *CardList) MouseExit() {
	old := l.hovered
	l.hovered = -1
	l.vbar.exit()
	l.invalidateRow(old)
}

func (l *CardList) MouseRelease(widget.MouseEvent) bool {
	if l.vbar.release() {
		l.Invalidate()
		return true
	}
	return false
}

func (l *CardList) MousePress(e widget.MouseEvent) bool {
	l.RequestFocus()
	if l.vbar.press(l, e.Pos, l.vaxis()) {
		l.Invalidate()
		return true
	}
	i := l.indexAt(e.Pos.Y)
	if i >= 0 {
		l.Selected = i
		l.Invalidate()
		if l.OnSelect != nil {
			l.OnSelect(i)
		}
	}
	if e.Button == platform.ButtonRight && l.OnContext != nil {
		o := widget.DeviceOrigin(l)
		l.OnContext(i, paintengine2d.Pt(o.X+e.Pos.X, o.Y+e.Pos.Y))
	}
	return true
}

// MouseWheel scrolls, and reports false when it cannot: an unscrollable or
// already-at-the-edge view must let the wheel bubble to an outer scroll pane
// instead of swallowing it.
func (l *CardList) MouseWheel(e widget.MouseEvent) bool {
	if l.MaxOffset() <= 0 {
		return false
	}
	before := l.OffsetY
	l.OffsetY += wheelDelta(e.Scroll.Y, l.rowH())
	l.clamp()
	if l.OffsetY == before {
		return false
	}
	l.Invalidate()
	return true
}

func (l *CardList) KeyPress(e widget.KeyEvent) bool {
	if !l.Enabled() || l.Count <= 0 {
		return false
	}
	next := l.Selected
	page := int(l.LocalBounds().Dy()/l.rowH()) - 1
	if page < 1 {
		page = 1
	}
	switch e.Key {
	case platform.KeyDown:
		next++
	case platform.KeyUp:
		next--
	case platform.KeyPageDown:
		next += page
	case platform.KeyPageUp:
		next -= page
	case platform.KeyHome:
		next = 0
	case platform.KeyEnd:
		next = l.Count - 1
	case platform.KeyReturn, platform.KeySpace:
		if l.Selected >= 0 && l.OnSelect != nil {
			l.OnSelect(l.Selected)
		}
		return true
	default:
		return false
	}
	if next < 0 {
		next = 0
	}
	if next >= l.Count {
		next = l.Count - 1
	}
	if next != l.Selected {
		l.Selected = next
		l.ensureVisible(next)
		l.Invalidate()
		if l.OnSelect != nil {
			l.OnSelect(next)
		}
	}
	return true
}

func (l *CardList) ensureVisible(i int) {
	rh := l.rowH()
	top := float32(i) * rh
	bot := top + rh
	view := l.LocalBounds().Dy()
	if top < l.OffsetY {
		l.OffsetY = top
	}
	if bot > l.OffsetY+view {
		l.OffsetY = bot - view
	}
	l.clamp()
}
