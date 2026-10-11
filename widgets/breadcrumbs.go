package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Crumb is one step of a path: where the window is, one level of it.
type Crumb struct {
	// Text is the step's name, which is what is read and what is shown.
	Text string
	// Icon is a mark before the text, [style.IconNone] for none: what
	// kind of thing this step is (a drive, a folder, a vault).
	Icon style.ToolIcon
	// OnClick is what pressing the step does — go there, usually.
	OnClick func()
	// Menu is what the step offers instead of going anywhere: the things
	// it could be changed to, as a drop-down under it. It is called when
	// the step is pressed, so a list that depends on what is on disk is
	// read then rather than when the crumbs were built.
	//
	// A step with a Menu and an OnClick offers the menu; the click is
	// what a step with no menu does.
	Menu func() []*MenuItem
}

// Breadcrumbs is a path of steps, each a button: Vault › personal ›
// Logins. The last step is where the window is; the ones before it are
// what it is inside, and any of them can be pressed to go there or to
// open what it could be changed to.
//
// It is the chrome a window wears instead of a title that says the same
// thing in one string, and it is a *control*: every step is reachable
// from the keyboard, and the path is read out as a row of buttons rather
// than as one long name.
//
// Two shapes, because the two eras draw this differently and neither is
// wrong. The default is words with a chevron between them, which is what
// a file manager and a browser's address bar do. [Breadcrumbs.Arrows]
// draws each step as a slab with a point, the next one's notch over the
// last one's point, which is what a wizard's progress and a shell's
// path-bar do — and the press in the notch goes to the step before, as
// the shape says it should.
type Breadcrumbs struct {
	widget.Base
	// Crumbs are the steps, outermost first.
	Crumbs []Crumb
	// Arrows draws the steps as pointed slabs instead of words with
	// chevrons between them.
	Arrows bool
	// OnPick is called with the index of a step that was pressed, after
	// its own OnClick. It is the simple way to use this: one handler and
	// no closures per step.
	OnPick func(i int)

	hover    int
	press    int
	focus    int
	keyNav   bool
	rects    []paintengine2d.Rect
	shown    []int // which crumbs the rects belong to
	hidden   []int // the ones there was no room for
	overflow bool
}

// NewBreadcrumbs is a path of steps, outermost first.
func NewBreadcrumbs(crumbs ...Crumb) *Breadcrumbs {
	b := &Breadcrumbs{Crumbs: crumbs, hover: -1, press: -1}
	b.Init(b)
	b.SetWantsFocus(true)
	b.SetFocusVisibleOnly(true)
	return b
}

// SetCrumbs replaces the path.
func (b *Breadcrumbs) SetCrumbs(crumbs ...Crumb) {
	b.Crumbs = crumbs
	if b.focus >= len(crumbs) {
		b.focus = max(len(crumbs)-1, 0)
	}
	b.rects, b.shown, b.hidden = nil, nil, nil
	b.RequestLayout()
	b.Invalidate()
}

// Count is how many steps the path has.
func (b *Breadcrumbs) Count() int { return len(b.Crumbs) }

// FocusOnClick is false: pressing a step moves somewhere, and the
// keyboard should stay in whatever the window was working in — the same
// rule a tool bar follows.
func (b *Breadcrumbs) FocusOnClick() bool { return false }

// ---- measuring ---------------------------------------------------------------

// crumbMetrics are the lengths the row is laid out by, at this look.
func (b *Breadcrumbs) crumbMetrics() (h, pad, gap, iconSide, sep float32) {
	lk := b.Look()
	m := lk.Metrics()
	h = m.ControlH
	if h <= 0 {
		h = style.Dip(lk, 24)
	}
	pad, iconSide, gap = style.ToolButtonChromeFor(lk, h)
	sep = style.Dip(lk, 14)
	if b.Arrows {
		// Nothing is drawn between two slabs: they interlock, the next
		// one's notch over the last one's point, so the row *overlaps*
		// by exactly the depth of that wedge. A gap there would be a
		// background-coloured arrowhead between every two steps.
		sep = -b.point(h)
		pad += b.point(h)
	}
	return h, pad, gap, iconSide, sep
}

// point is how far a slab's point reaches past its box, which is also how
// deep the next one's notch is.
func (b *Breadcrumbs) point(h float32) float32 { return h * 0.34 }

// crumbWidth is how wide one step wants to be.
func (b *Breadcrumbs) crumbWidth(i int, pad, gap, iconSide float32) float32 {
	c := b.Crumbs[i]
	w := pad * 2
	if c.Icon != style.IconNone {
		w += iconSide
		if c.Text != "" {
			w += gap
		}
	}
	if c.Text != "" {
		w += style.ControlAdvanceOf(b.Look(), style.RoleTool, c.Text)
	}
	return w
}

func (b *Breadcrumbs) Measure(c layout.Constraints) paintengine2d.Point {
	h, pad, gap, iconSide, sep := b.crumbMetrics()
	var w float32
	for i := range b.Crumbs {
		if i > 0 {
			w += sep
		}
		w += b.crumbWidth(i, pad, gap, iconSide)
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

// MinWidth is the last step and an ellipsis: a path can always be shown
// as "… › where you are", which is the one step nobody can do without.
func (b *Breadcrumbs) MinWidth() float32 {
	if len(b.Crumbs) == 0 {
		return 0
	}
	_, pad, gap, iconSide, sep := b.crumbMetrics()
	w := b.crumbWidth(len(b.Crumbs)-1, pad, gap, iconSide)
	if len(b.Crumbs) > 1 {
		w += sep + b.ellipsisWidth(pad)
	}
	return w
}

// ellipsisWidth is the width of the step that stands for the ones there
// was no room for.
func (b *Breadcrumbs) ellipsisWidth(pad float32) float32 {
	return pad*2 + style.ControlAdvanceOf(b.Look(), style.RoleTool, crumbEllipsis)
}

// crumbEllipsis is what the dropped steps are shown as.
const crumbEllipsis = "…"

func (b *Breadcrumbs) Arrange(r paintengine2d.Rect) {
	b.SetBounds(r)
	b.layoutCrumbs(r.Dx())
}

// layoutCrumbs places the steps across width, dropping the leading ones
// where there is no room and standing them in with one ellipsis step.
//
// The *leading* ones, because the last step is where the window is: a
// path that cannot be shown whole is still useful read from the end, and
// is useless read from the start.
func (b *Breadcrumbs) layoutCrumbs(width float32) {
	h, pad, gap, iconSide, sep := b.crumbMetrics()
	b.rects, b.shown, b.hidden, b.overflow = nil, nil, nil, false
	if len(b.Crumbs) == 0 {
		return
	}
	widths := make([]float32, len(b.Crumbs))
	for i := range b.Crumbs {
		widths[i] = b.crumbWidth(i, pad, gap, iconSide)
	}
	// From the end back, taking what fits.
	first := 0
	var total float32
	for i := len(b.Crumbs) - 1; i >= 0; i-- {
		w := widths[i]
		if i < len(b.Crumbs)-1 {
			w += sep
		}
		if total+w > width && i < len(b.Crumbs)-1 {
			first = i + 1
			break
		}
		total += w
	}
	if first > 0 {
		b.overflow = true
		// Room for the ellipsis as well, taking more steps off the front
		// until it fits.
		need := b.ellipsisWidth(pad) + sep
		for first < len(b.Crumbs)-1 && total+need > width {
			total -= widths[first] + sep
			first++
		}
		for i := 0; i < first; i++ {
			b.hidden = append(b.hidden, i)
		}
	}
	x := float32(0)
	place := func(i int, w float32) {
		b.rects = append(b.rects, paintengine2d.XYWH(x, 0, w, h))
		b.shown = append(b.shown, i)
		x += w + sep
	}
	if b.overflow {
		place(crumbOverflow, b.ellipsisWidth(pad))
	}
	for i := first; i < len(b.Crumbs); i++ {
		place(i, widths[i])
	}
}

// crumbOverflow is the index the ellipsis step answers to.
const crumbOverflow = -2

// CrumbRect is where step i is, in the widget's own coordinates, or the
// empty rect for a step there was no room for.
func (b *Breadcrumbs) CrumbRect(i int) paintengine2d.Rect {
	for k, at := range b.shown {
		if at == i {
			return b.rects[k]
		}
	}
	return paintengine2d.Rect{}
}

// ---- painting ----------------------------------------------------------------

func (b *Breadcrumbs) Paint(ctx *paintengine2d.Context) {
	if len(b.rects) == 0 {
		b.layoutCrumbs(b.LocalBounds().Dx())
	}
	lk := b.Look()
	_, pad, gap, iconSide, _ := b.crumbMetrics()
	for k, i := range b.shown {
		r := b.rects[k]
		last := i == len(b.Crumbs)-1
		st := b.stateOf(i)
		if b.Arrows {
			b.paintSlab(ctx, r, st, last, k > 0)
		} else if st&(style.StateHovered|style.StatePressed) != 0 || last {
			// Flat until the pointer is on it, which is what a path bar
			// is: a row of words, not a row of buttons.
			lk.DrawToolButton(ctx, r, st|style.StateAutoRaise, "", style.IconNone)
		}
		b.paintCrumb(ctx, r, i, st, last, pad, gap, iconSide)
		if !b.Arrows && k < len(b.shown)-1 {
			// The chevron between two steps, in the look's own arrow.
			sepBox := paintengine2d.XYWH(r.Max.X, r.Min.Y, b.sepWidth(), r.Dy())
			style.DrawArrowOf(lk, ctx, sepBox.Inset(sepBox.Dx()*0.28), style.DirRight, lk.Palette().TextMuted)
		}
	}
	if b.Focused() && b.keyNav {
		if r := b.focusRect(); !r.Empty() {
			lk.DrawFocusRing(ctx, r.Inset(1))
		}
	}
}

func (b *Breadcrumbs) sepWidth() float32 {
	_, _, _, _, sep := b.crumbMetrics()
	return sep
}

// stateOf is the state step i is in.
func (b *Breadcrumbs) stateOf(i int) style.ControlState {
	st := b.State()
	// The row is one widget, so its own hover must not make every step
	// look hovered: only the step the pointer is actually on does.
	st &^= style.StateHovered
	if !b.Enabled() {
		return st
	}
	if b.press == i {
		st |= style.StatePressed
	} else if b.hover == i {
		st |= style.StateHovered
	}
	return st
}

// paintSlab draws one pointed slab: the path-bar shape, where the step
// the window is on is the accent and the rest are the window's own
// surface.
func (b *Breadcrumbs) paintSlab(ctx *paintengine2d.Context, r paintengine2d.Rect, st style.ControlState, last, notch bool) {
	p := b.Look().Palette()
	fill := p.SurfaceAlt
	switch {
	case last:
		fill = p.Accent
	case st.Pressed():
		fill = p.AccentPress
	case st.Hovered():
		fill = p.AccentHover
	}
	ctx.DrawPath(b.slabPath(r, notch), paintengine2d.Fill(fill))
}

// slabPath is the slab's outline: a rectangle with a point on its
// trailing edge and, where something is behind it, a notch of the same
// shape cut out of its leading one, so two of them interlock. The first
// slab has nothing behind it and is flat on that side, which is where a
// path begins.
func (b *Breadcrumbs) slabPath(r paintengine2d.Rect, notch bool) *paintengine2d.Path {
	tip := b.point(r.Dy())
	cy := (r.Min.Y + r.Max.Y) * 0.5
	p := paintengine2d.NewPath()
	p.MoveTo(r.Min.X, r.Min.Y)
	p.LineTo(r.Max.X-tip, r.Min.Y)
	p.LineTo(r.Max.X, cy)
	p.LineTo(r.Max.X-tip, r.Max.Y)
	p.LineTo(r.Min.X, r.Max.Y)
	if notch {
		p.LineTo(r.Min.X+tip, cy)
	}
	p.Close()
	return p
}

// paintCrumb draws one step's mark and words inside r.
func (b *Breadcrumbs) paintCrumb(ctx *paintengine2d.Context, r paintengine2d.Rect, i int, st style.ControlState, last bool, pad, gap, iconSide float32) {
	lk := b.Look()
	text, icon := crumbEllipsis, style.IconNone
	if i >= 0 {
		text, icon = b.Crumbs[i].Text, b.Crumbs[i].Icon
	}
	ink := lk.Palette().Text
	switch {
	case !b.Enabled():
		ink = lk.Palette().TextMuted
	case b.Arrows:
		ink = lk.Palette().TextOnAccent
		if !last && st&(style.StateHovered|style.StatePressed) == 0 {
			ink = lk.Palette().Text
		}
	case !last:
		ink = lk.Palette().TextMuted
	}
	x := r.Min.X + pad
	if icon != style.IconNone {
		box := paintengine2d.XYWH(x, r.Min.Y+(r.Dy()-iconSide)*0.5, iconSide, iconSide)
		style.DrawToolIcon(ctx, box, icon, ink, style.IconSetOf(lk))
		x = box.Max.X + gap
	}
	if text == "" {
		return
	}
	f := style.ControlFontOf(lk, style.RoleTool)
	if f == nil {
		return
	}
	shown := f.Fit(style.ControlLabelOf(lk, style.RoleTool, text), r.Max.X-pad-x)
	f.Draw(ctx, shown, paintengine2d.Pt(x, r.Min.Y+(r.Dy()-f.Height())*0.5), ink)
}

// focusRect is the box the focus ring goes round.
func (b *Breadcrumbs) focusRect() paintengine2d.Rect {
	if r := b.CrumbRect(b.focus); !r.Empty() {
		return r
	}
	if len(b.rects) > 0 {
		return b.rects[len(b.rects)-1]
	}
	return paintengine2d.Rect{}
}

// ---- the pointer -------------------------------------------------------------

// crumbAt is the step at p (widget coordinates), or the smallest int for
// none. In the slab shape a press in a notch belongs to the step before,
// which is what the shape says: the notch is the point of the one behind
// it showing through.
func (b *Breadcrumbs) crumbAt(p paintengine2d.Point) (int, bool) {
	if len(b.rects) == 0 {
		b.layoutCrumbs(b.LocalBounds().Dx())
	}
	// Backwards: a slab's point lies over the next one's notch, so the
	// later slab's box is tested first and the notch falls through it.
	for k := len(b.rects) - 1; k >= 0; k-- {
		r := b.rects[k]
		if !r.Contains(p) {
			continue
		}
		if b.Arrows && !inSlab(r, p, b.point(r.Dy()), k > 0) {
			continue
		}
		return b.shown[k], true
	}
	return 0, false
}

// inSlab reports whether p is inside the slab outline in r — outside the
// notch at its leading edge and inside the point at its trailing one.
func inSlab(r paintengine2d.Rect, p paintengine2d.Point, tip float32, notch bool) bool {
	if tip <= 0 {
		return true
	}
	cy := (r.Min.Y + r.Max.Y) * 0.5
	// How far from the middle line, as a fraction of the half height: the
	// notch and the point are the same wedge, deepest at the middle and
	// nothing at the top and bottom edges.
	off := abs32c(p.Y-cy) / max(r.Dy()*0.5, 1)
	if notch && p.X < r.Min.X+tip*(1-off) {
		return false // in the notch, which belongs to the step before
	}
	if p.X > r.Max.X-tip*off {
		return false // past the point
	}
	return true
}

func abs32c(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (b *Breadcrumbs) MouseMove(e widget.MouseEvent) bool {
	i, ok := b.crumbAt(e.Pos)
	if !ok {
		i = -1
	}
	if b.hover != i {
		b.hover = i
		b.Invalidate()
	}
	return false
}

func (b *Breadcrumbs) MouseExit() {
	if b.hover != -1 || b.press != -1 {
		b.hover, b.press = -1, -1
		b.Invalidate()
	}
	b.Base.MouseExit()
}

func (b *Breadcrumbs) MousePress(e widget.MouseEvent) bool {
	if !b.Enabled() || e.Button != platform.ButtonLeft {
		return false
	}
	i, ok := b.crumbAt(e.Pos)
	if !ok {
		return false
	}
	b.MarkPointerFocus()
	b.press = i
	b.keyNav = false
	if i >= 0 {
		b.focus = i
	}
	b.Invalidate()
	return true
}

func (b *Breadcrumbs) MouseRelease(e widget.MouseEvent) bool {
	i := b.press
	b.press = -1
	b.Invalidate()
	if i == 0 && !b.Enabled() {
		return false
	}
	if at, ok := b.crumbAt(e.Pos); !ok || at != i {
		return false
	}
	b.activate(i)
	return true
}

// activate opens what step i offers, or goes there.
func (b *Breadcrumbs) activate(i int) {
	if i == crumbOverflow {
		b.openOverflow()
		return
	}
	if i < 0 || i >= len(b.Crumbs) {
		return
	}
	c := b.Crumbs[i]
	if c.Menu != nil {
		if items := c.Menu(); len(items) > 0 {
			b.openMenu(i, items)
			return
		}
	}
	if c.OnClick != nil {
		c.OnClick()
	}
	if b.OnPick != nil {
		b.OnPick(i)
	}
}

// openMenu drops items under step i.
func (b *Breadcrumbs) openMenu(i int, items []*MenuItem) {
	r := b.CrumbRect(i)
	if r.Empty() {
		return
	}
	o := widget.DeviceOrigin(b)
	anchor := paintengine2d.XYWH(o.X+r.Min.X, o.Y+r.Min.Y, r.Dx(), r.Dy())
	pop := NewPopupMenu(items...)
	pop.RestoreFocusTo(b)
	widget.PlacePopupForAnchor(b, pop, anchor, r.Dx(), 2)
	widget.ShowPopup(b, pop)
}

// openOverflow offers the steps there was no room for, in their own
// order, so a path too long to show is still a path that can be walked.
func (b *Breadcrumbs) openOverflow() {
	var items []*MenuItem
	for _, i := range b.hidden {
		at := i
		c := b.Crumbs[i]
		items = append(items, &MenuItem{Text: c.Text, Icon: c.Icon, OnClick: func() { b.activate(at) }})
	}
	if len(items) == 0 {
		return
	}
	b.openMenu(crumbOverflow, items)
}

// ---- the keyboard ------------------------------------------------------------

// KeyPress moves along the path and opens a step, the way a tool bar's
// keys work: one tab stop for the row, the arrows inside it.
func (b *Breadcrumbs) KeyPress(e widget.KeyEvent) bool {
	if !b.Enabled() || len(b.Crumbs) == 0 {
		return false
	}
	switch e.Key {
	case platform.KeyLeft, platform.KeyRight, platform.KeyHome, platform.KeyEnd,
		platform.KeyReturn, platform.KeySpace, platform.KeyDown:
	default:
		return false
	}
	b.keyNav = true
	b.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeyLeft:
		b.focus = max(b.focus-1, 0)
	case platform.KeyRight:
		b.focus = min(b.focus+1, len(b.Crumbs)-1)
	case platform.KeyHome:
		b.focus = 0
	case platform.KeyEnd:
		b.focus = len(b.Crumbs) - 1
	case platform.KeyReturn, platform.KeySpace, platform.KeyDown:
		b.activate(b.focus)
		return true
	}
	b.Invalidate()
	return true
}

// FocusIndex is the step the keyboard is on inside the row.
func (b *Breadcrumbs) FocusIndex() int { return b.focus }
