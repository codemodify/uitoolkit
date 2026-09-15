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
// places the header bar in its caption and adds the caption buttons at the
// sides the desktop puts them; under the desktop's own frame it is the
// window's first row and shows no caption buttons. Either way its free space
// is caption: pressing there moves the window, double-clicking maximizes,
// right-clicking shows the window menu (see widget.CaptionHitTester).
type HeaderBar struct {
	widget.Base
	row    *FlexBox
	free   *Spacer
	center widget.Component
	lead   *WindowControls
	trail  *WindowControls
	framed bool
	// ShowTitle paints the window's title in the free space when the header
	// bar has no centre.
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
	h.lead = NewWindowControls()
	h.trail = NewWindowControls()
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

// minCaption is the least caption height of a toolkit-drawn frame: room for
// the caption buttons and a line of title.
func (h *HeaderBar) minCaption() float32 {
	lk := h.Look()
	m := max(h.lead.MinHeight(), float32(math.Round(float64(style.Dip(lk, 32)))))
	if f := lk.BoldFont(); f != nil {
		m = max(m, float32(math.Ceil(float64(f.Height()+style.Dip(lk, 12)))))
	}
	return m
}

func (h *HeaderBar) Measure(c layout.Constraints) paintengine2d.Point {
	lw, rw := h.controlsW()
	inner := c
	if c.HasMaxW() {
		inner.MaxW = max(0, c.MaxW-lw-rw)
		inner.MinW = max(0, c.MinW-lw-rw)
	}
	sz := h.row.Measure(inner)
	if h.framed {
		sz.Y = max(sz.Y, h.minCaption())
	}
	return c.Constrain(paintengine2d.Pt(sz.X+lw+rw, sz.Y))
}

func (h *HeaderBar) Arrange(r paintengine2d.Rect) {
	h.SetBounds(r)
	lw, rw := h.controlsW()
	w, ht := r.Dx(), r.Dy()
	if h.lead.Visible() {
		h.lead.Arrange(paintengine2d.XYWH(0, 0, h.lead.Measure(layout.Unbounded()).X, ht))
	} else {
		h.lead.Arrange(paintengine2d.Rect{})
	}
	if h.trail.Visible() {
		tw := h.trail.Measure(layout.Unbounded()).X
		h.trail.Arrange(paintengine2d.XYWH(w-tw, 0, tw, ht))
	} else {
		h.trail.Arrange(paintengine2d.Rect{})
	}
	h.row.Arrange(paintengine2d.XYWH(lw, 0, max(0, w-lw-rw), ht))
}

// Paint draws the title, when shown, centred in the free space.
func (h *HeaderBar) Paint(ctx *paintengine2d.Context) {
	if !h.ShowTitle || h.center != nil {
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
	col := lk.Palette().Text
	if !widget.WindowActive(h) {
		col = lk.Palette().TextMuted
	}
	lk.DrawLabel(ctx, h.titleRect(), title, col, style.AlignCenter)
}

// titleRect is where the title goes: the free space's width, the row's
// height (the free space itself is centred with no height of its own).
func (h *HeaderBar) titleRect() paintengine2d.Rect {
	rb, fb := h.row.Bounds(), h.free.Bounds()
	return paintengine2d.XYWH(rb.Min.X+fb.Min.X, rb.Min.Y, fb.Dx(), rb.Dy())
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
	if !h.ShowTitle || h.center != nil {
		return nil
	}
	t, ok := h.Host().(interface{ Title() string })
	if !ok || t.Title() == "" {
		return nil
	}
	return []*a11y.Node{item(h, 0, a11y.RoleLabel, t.Title(), h.titleRect())}
}
