package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// HeaderBar is a window's title-bar content: leading items, a centre (tabs,
// a search field, or nothing — then its free space moves the window) and
// trailing items, in one row. Give it to Window.SetTitleBar.
//
// When uitoolkit draws the window's frame (a client-side frame) the window
// places the header bar in its caption and the look's frame decides how the
// two meet: a merged frame (GTK's header bars, Windows 11, macOS, Chromium)
// makes the row the caption, with the caption buttons at the sides the
// desktop puts them; a stacked frame (Windows 95 and XP, the classic Mac,
// Motif) keeps the era's caption — the window title and the buttons — in a
// strip and puts the row under it. Under the desktop's own frame the header
// bar is the window's first row and shows no caption buttons. Either way its
// free space is caption: pressing there moves the window, double-clicking
// maximizes, right-clicking shows the window menu (see
// widget.CaptionHitTester).
type HeaderBar struct {
	widget.Base
	row    *FlexBox
	free   *Spacer
	center widget.Component
	lead   *WindowControls
	trail  *WindowControls
	framed bool
	// custom: the app gave the header bar items of its own.
	custom bool
	// strip is a stacked frame's caption strip height in the last layout
	// (0: merged, or no frame).
	strip float32
	// ShowTitle paints the window's title in the free space when the header
	// bar has no centre (a stacked frame's strip always shows it).
	ShowTitle bool
	// OnContextMenu, when set, runs for a right-click on the header bar's
	// caption space (at, window device pixels) instead of the window menu,
	// and reports whether it showed a menu (Chromium's tab-strip menu).
	OnContextMenu func(at paintengine2d.Point) bool
}

// NewHeaderBar builds a header bar: start items on the left, center in the
// middle taking the free width (nil: free space that moves the window), end
// items on the right. Items keep their natural width, 8 px apart, centred
// vertically, like any row.
func NewHeaderBar(start []widget.Component, center widget.Component, end []widget.Component) *HeaderBar {
	h := &HeaderBar{free: NewSpacer(), center: center}
	h.Init(h)
	h.custom = len(start) > 0 || center != nil || len(end) > 0
	h.lead = NewWindowControls()
	h.trail = NewWindowControls()
	h.lead.SetLeading(true)
	h.lead.SetVisible(false)
	h.trail.SetVisible(false)
	h.row = NewRow().WithGap(8).WithAlign(layout.AlignCenter)
	for _, c := range start {
		h.row.Add(c)
	}
	mid := widget.Component(h.free)
	if center != nil {
		mid = center
	}
	h.row.AddFlex(mid, 1)
	for _, c := range end {
		h.row.Add(c)
	}
	h.Add(h.lead)
	h.Add(h.row)
	h.Add(h.trail)
	return h
}

// Center is the centre component (nil for free space).
func (h *HeaderBar) Center() widget.Component { return h.center }

// Row is the row holding the items (tests, inspectors).
func (h *HeaderBar) Row() *FlexBox { return h.row }

// Controls are the caption buttons on the left and the right.
func (h *HeaderBar) Controls() (left, right *WindowControls) { return h.lead, h.trail }

// Framed reports whether the header bar is the caption of a frame the
// toolkit draws (caption buttons shown).
func (h *HeaderBar) Framed() bool { return h.framed }

// Custom reports whether the header bar holds items of the app's (not just
// the window title).
func (h *HeaderBar) Custom() bool { return h.custom }

// SetWindowControls is how the window tells the header bar about its
// frame: with framed set it shows the caption buttons of layout; without,
// it shows none.
func (h *HeaderBar) SetWindowControls(l platform.ButtonLayout, framed bool) {
	h.framed = framed
	h.lead.SetButtons(l.Left)
	h.trail.SetButtons(l.Right)
	h.lead.SetVisible(framed && len(l.Left) > 0)
	h.trail.SetVisible(framed && len(l.Right) > 0)
	h.RequestLayout()
	h.Invalidate()
}

// DecorationState is the state the window's frame paints in: the window's
// active, maximized and tiled states, and whether the header bar holds the
// app's own items.
func (h *HeaderBar) DecorationState() style.DecorationState {
	fh, _ := h.Host().(widget.FrameHost)
	return frameState(fh, h.custom)
}

// spec is the look's frame in the window's current state.
func (h *HeaderBar) spec() style.DecorationSpec {
	return style.DecorationOf(h.Look(), h.DecorationState())
}

// Stacked reports whether the header bar lays a stacked frame's caption
// strip above its row (see style.DecorationSpec.Stacked).
func (h *HeaderBar) Stacked() bool { return h.framed && h.spec().Stacked }

// inMergedCaption reports whether c is in the caption of a merged frame the
// toolkit draws: the look's caption band is its background, so bars there
// (a menu bar, a tab strip) paint none of their own.
func inMergedCaption(c widget.Component) bool {
	for p := c.Parent(); p != nil; p = p.Parent() {
		if hb, ok := p.(*HeaderBar); ok {
			return hb.Framed() && !hb.Stacked()
		}
	}
	return false
}

// FrameParts are the caption band and, under a stacked frame's strip, the
// title-bar row, in the header bar's own coordinates: the parts the look's
// frame paints under it.
func (h *HeaderBar) FrameParts() (caption, bar paintengine2d.Rect) {
	b := h.LocalBounds()
	if h.strip <= 0 {
		return b, paintengine2d.Rect{}
	}
	caption = paintengine2d.XYWH(0, 0, b.Dx(), min(h.strip, b.Dy()))
	if h.custom && b.Dy() > h.strip {
		bar = paintengine2d.XYWH(0, h.strip, b.Dx(), b.Dy()-h.strip)
	}
	return caption, bar
}

// controlsW is the room each side's caption buttons take, gap included.
func (h *HeaderBar) controlsW() (left, right float32) {
	gap := float32(math.Round(float64(style.Dip(h.Look(), 4))))
	if h.lead.Visible() {
		if w := h.lead.Measure(layout.Unbounded()).X; w > 0 {
			left = w + gap
		}
	}
	if h.trail.Visible() {
		if w := h.trail.Measure(layout.Unbounded()).X; w > 0 {
			right = w + gap
		}
	}
	return left, right
}

// minCaption is the least caption height of a toolkit-drawn frame: the
// look's caption, room for the caption buttons and a line of title.
func (h *HeaderBar) minCaption(s style.DecorationSpec) float32 {
	lk := h.Look()
	m := max(s.Caption, h.lead.MinHeight(), h.trail.MinHeight())
	if !s.Stacked {
		m = max(m, float32(math.Round(float64(style.Dip(lk, 32)))))
		if f := lk.BoldFont(); f != nil {
			m = max(m, float32(math.Ceil(float64(f.Height()+style.Dip(lk, 12)))))
		}
	}
	return float32(math.Ceil(float64(m) - 1e-3))
}

func (h *HeaderBar) Measure(c layout.Constraints) paintengine2d.Point {
	s := h.spec()
	if h.framed && s.Stacked {
		strip := h.minCaption(s)
		sz := paintengine2d.Pt(0, strip)
		if h.custom {
			inner := c.Inset(0, strip)
			inner.MinH = 0
			row := h.row.Measure(inner)
			sz = paintengine2d.Pt(row.X, strip+row.Y)
		}
		return c.Constrain(sz)
	}
	lw, rw := h.controlsW()
	inner := c
	if c.HasMaxW() {
		inner.MaxW = max(0, c.MaxW-lw-rw)
		inner.MinW = max(0, c.MinW-lw-rw)
	}
	sz := h.row.Measure(inner)
	if h.framed {
		sz.Y = max(sz.Y, h.minCaption(s))
	}
	return c.Constrain(paintengine2d.Pt(sz.X+lw+rw, sz.Y))
}

func (h *HeaderBar) Arrange(r paintengine2d.Rect) {
	h.SetBounds(r)
	w, ht := r.Dx(), r.Dy()
	s := h.spec()
	band := ht
	h.strip = 0
	if h.framed && s.Stacked {
		h.strip = min(h.minCaption(s), ht)
		band = h.strip
	}
	if h.lead.Visible() {
		h.lead.Arrange(paintengine2d.XYWH(0, 0, h.lead.Measure(layout.Unbounded()).X, band))
	} else {
		h.lead.Arrange(paintengine2d.Rect{})
	}
	if h.trail.Visible() {
		tw := h.trail.Measure(layout.Unbounded()).X
		h.trail.Arrange(paintengine2d.XYWH(w-tw, 0, tw, band))
	} else {
		h.trail.Arrange(paintengine2d.Rect{})
	}
	var rowBox paintengine2d.Rect
	switch {
	case h.strip > 0 && h.custom:
		rowBox = paintengine2d.XYWH(0, h.strip, w, max(ht-h.strip, 0))
	case h.strip > 0:
		// A stacked frame's caption holding only the title: no row.
	default:
		lw, rw := h.controlsW()
		rowBox = paintengine2d.XYWH(lw, 0, max(0, w-lw-rw), ht)
	}
	h.row.Arrange(rowBox)
	// A centre that fills the caption's height (a tab strip whose selected
	// tab meets the content below) gets all of it, not the centred slice
	// the row gives it.
	if f, ok := h.center.(interface{ FillsCaption() bool }); ok && f.FillsCaption() && !rowBox.Empty() {
		cb := h.center.Bounds()
		h.center.Arrange(paintengine2d.XYWH(cb.Min.X, 0, cb.Dx(), rowBox.Dy()))
	}
}

// showsTitle reports whether the header bar paints the window title: a
// stacked frame's strip always does, the row's free space when ShowTitle is
// set and there is no centre.
func (h *HeaderBar) showsTitle() bool {
	return h.strip > 0 || h.ShowTitle && h.center == nil
}

// Paint draws the title, when shown, in the caption's free space.
func (h *HeaderBar) Paint(ctx *paintengine2d.Context) {
	if !h.showsTitle() {
		return
	}
	title := ""
	if t, ok := h.Host().(interface{ Title() string }); ok {
		title = t.Title()
	}
	if title == "" {
		return
	}
	lk := h.Look()
	if h.framed {
		style.DrawCaptionTitleOf(lk, ctx, h.titleRect(), title, h.DecorationState())
		return
	}
	col := lk.Palette().Text
	if !widget.WindowActive(h) {
		col = lk.Palette().TextMuted
	}
	lk.DrawLabel(ctx, h.titleRect(), title, col, style.AlignCenter)
}

// titleRect is where the title goes: a stacked frame's strip between the
// buttons, else the free space's width at the row's height (the free space
// itself is centred with no height of its own). A look that centres its
// title on the window gets a box the title's width centred on the window,
// or, where the buttons leave no room at the centre, pushed aside by them
// (as GNOME and the Mac do) rather than cut short.
func (h *HeaderBar) titleRect() paintengine2d.Rect {
	var r paintengine2d.Rect
	if h.strip > 0 {
		// The strip's own space, right up to the buttons (the look pads
		// its title).
		x0, x1 := float32(0), h.LocalBounds().Dx()
		if h.lead.Visible() {
			x0 = h.lead.Bounds().Max.X
		}
		if h.trail.Visible() {
			x1 = h.trail.Bounds().Min.X
		}
		r = paintengine2d.XYWH(x0, 0, max(x1-x0, 0), h.strip)
	} else {
		rb, fb := h.row.Bounds(), h.free.Bounds()
		r = paintengine2d.XYWH(rb.Min.X+fb.Min.X, rb.Min.Y, fb.Dx(), rb.Dy())
	}
	if h.framed && h.spec().CenterTitle {
		tw := min(h.titleWidth(), r.Dx())
		x := min(max((h.LocalBounds().Dx()-tw)*0.5, r.Min.X), r.Max.X-tw)
		r.Min.X, r.Max.X = x, x+tw
	}
	return r
}

// titleWidth is about what a look's centred title needs: the title in the
// bold font (the widest a title bar uses) and the looks' pads round it.
func (h *HeaderBar) titleWidth() float32 {
	title := ""
	if t, ok := h.Host().(interface{ Title() string }); ok {
		title = t.Title()
	}
	lk := h.Look()
	f := lk.BoldFont()
	if f == nil {
		f = lk.Font()
	}
	return f.Advance(title) + 2*style.Dip(lk, 12)
}

// CaptionAt: the header bar's own space is caption.
func (h *HeaderBar) CaptionAt(paintengine2d.Point) bool { return true }

// Describe implements widget.Accessible: the window's title bar where the
// toolkit draws the frame; under the desktop's frame (which has the real
// title bar) just the pane of the window's first row.
func (h *HeaderBar) Describe(n *a11y.Node) {
	if !h.framed {
		n.Role = a11y.RolePane
		return
	}
	n.Role = a11y.RoleTitleBar
	if t, ok := h.Host().(interface{ Title() string }); ok {
		n.Name = t.Title()
	}
}

// AccessibleItems: the title, as a label, when the header bar shows it.
func (h *HeaderBar) AccessibleItems() []*a11y.Node {
	if !h.showsTitle() {
		return nil
	}
	t, ok := h.Host().(interface{ Title() string })
	if !ok || t.Title() == "" {
		return nil
	}
	return []*a11y.Node{item(h, 0, a11y.RoleLabel, t.Title(), h.titleRect())}
}
