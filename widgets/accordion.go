package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// Expander is a disclosure header with a collapsible body.
type Expander struct {
	widget.Base
	Title    string
	Expanded bool
	OnToggle func(bool)
	head     *expanderHead
	body     widget.Component
}

// NewExpander builds a titled section. child may be nil.
func NewExpander(title string, expanded bool, child widget.Component) *Expander {
	e := &Expander{Title: title, Expanded: expanded}
	e.Init(e)
	e.head = newExpanderHead(e)
	e.Base.Add(e.head)
	if child != nil {
		e.body = child
		e.Base.Add(child)
		child.SetVisible(expanded)
	}
	return e
}

// Content is the expandable child, if any.
func (e *Expander) Content() widget.Component { return e.body }

func (e *Expander) SetExpanded(v bool) {
	if e.Expanded == v {
		return
	}
	e.Expanded = v
	if e.body != nil {
		if !v {
			e.yieldFocusFromBody()
		}
		e.body.SetVisible(v)
	}
	e.Invalidate()
	e.RequestLayout()
	if e.OnToggle != nil {
		e.OnToggle(v)
	}
}

func (e *Expander) yieldFocusFromBody() {
	h := e.Host()
	if h == nil || e.body == nil || e.head == nil {
		return
	}
	if widget.Contains(e.body, h.Focus()) {
		e.head.RequestFocus()
	}
}

func (e *Expander) headerH() float32 {
	h := e.Look().Metrics().AccordionH
	if h <= 0 {
		h = e.Look().Metrics().ControlH
	}
	if h <= 0 {
		h = 30
	}
	return h
}

func (e *Expander) bodyPad() float32 {
	p := e.Look().Metrics().Pad
	if p <= 0 {
		p = 10
	}
	return p
}

func (e *Expander) Measure(c layout.Constraints) paintengine2d.Point {
	hh := e.headerH()
	w := float32(160)
	if c.HasMaxW() {
		w = c.MaxW
	}
	h := hh
	if e.Expanded && e.body != nil && e.body.Visible() {
		pad := e.bodyPad()
		inner := c.Inset(pad, 4)
		if c.HasMaxW() {
			inner.MaxW = c.MaxW - pad
			if inner.MaxW < 0 {
				inner.MaxW = 0
			}
		}
		sz := e.body.Measure(inner)
		h += sz.Y + 6
		need := sz.X + pad
		if need > w {
			w = need
		}
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (e *Expander) Arrange(r paintengine2d.Rect) {
	e.SetBounds(r)
	hh := e.headerH()
	e.head.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), hh))
	if e.body == nil {
		return
	}
	if !e.Expanded {
		e.body.SetVisible(false)
		e.body.Arrange(paintengine2d.XYWH(0, hh, r.Dx(), 0))
		return
	}
	e.body.SetVisible(true)
	pad := e.bodyPad()
	e.body.Arrange(paintengine2d.XYWH(pad, hh+4, r.Dx()-pad, r.Dy()-hh-6))
}

// Accordion stacks Expander sections. Exclusive keeps one open at a time.
type Accordion struct {
	widget.Base
	Exclusive bool
	col       *FlexBox
	items     []*Expander
}

// NewAccordion stacks sections. exclusive closes others when one opens.
func NewAccordion(exclusive bool, items ...*Expander) *Accordion {
	a := &Accordion{Exclusive: exclusive, col: NewColumn().WithGap(0)}
	a.Init(a)
	a.Base.Add(a.col)
	for _, it := range items {
		a.AddSection(it)
	}
	return a
}

// Sections returns expanders in order.
func (a *Accordion) Sections() []*Expander { return a.items }

// AddSection appends an expander.
func (a *Accordion) AddSection(e *Expander) {
	if e == nil {
		return
	}
	prev := e.OnToggle
	e.OnToggle = func(open bool) {
		if a.Exclusive && open {
			a.closeOthers(e)
		}
		if prev != nil {
			prev(open)
		}
	}
	a.items = append(a.items, e)
	a.col.Add(e)
	if a.Exclusive && e.Expanded {
		a.closeOthers(e)
	}
}

func (a *Accordion) closeOthers(keep *Expander) {
	for _, o := range a.items {
		if o != keep && o.Expanded {
			o.SetExpanded(false)
		}
	}
}

func (a *Accordion) Measure(c layout.Constraints) paintengine2d.Point {
	return a.col.Measure(c)
}

func (a *Accordion) Arrange(r paintengine2d.Rect) {
	a.SetBounds(r)
	a.col.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
}

type expanderHead struct {
	widget.Base
	owner *Expander
}

func newExpanderHead(owner *Expander) *expanderHead {
	h := &expanderHead{owner: owner}
	h.Init(h)
	h.SetWantsFocus(true)
	h.SetFocusVisibleOnly(true)
	return h
}

func (h *expanderHead) Measure(c layout.Constraints) paintengine2d.Point {
	hh := h.owner.headerH()
	w := float32(160)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, hh))
}

func (h *expanderHead) Arrange(r paintengine2d.Rect) { h.SetBounds(r) }

func (h *expanderHead) Paint(ctx *paintengine2d.Context) {
	st := h.State()
	h.Look().DrawAccordionHeader(ctx, h.LocalBounds(), st, h.owner.Title, h.owner.Expanded)
}

func (h *expanderHead) MousePress(widget.MouseEvent) bool {
	if !h.Enabled() {
		return false
	}
	h.MarkPointerFocus()
	h.RequestFocus()
	h.owner.SetExpanded(!h.owner.Expanded)
	return true
}

func (h *expanderHead) KeyPress(e widget.KeyEvent) bool {
	if !h.Enabled() {
		return false
	}
	h.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeySpace, platform.KeyReturn:
		h.owner.SetExpanded(!h.owner.Expanded)
		return true
	case platform.KeyRight:
		if !h.owner.Expanded {
			h.owner.SetExpanded(true)
		}
		return true
	case platform.KeyLeft:
		if h.owner.Expanded {
			h.owner.SetExpanded(false)
		}
		return true
	}
	return false
}

var _ widget.Component = (*expanderHead)(nil)
