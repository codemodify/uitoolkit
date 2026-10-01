package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// FieldBox is a text field's well with controls inside it.
//
// A browser's address bar is the case it exists for: the site-information
// mark sits inside the pill at its leading end, the text runs between, and
// the bookmark star sits inside at the trailing end — one well, three
// things in it. A search box with a magnifier, a password field with a
// reveal, an amount with its currency: all the same shape.
//
// [TextField.Clearable] solves only half of it, and only at the trailing
// end, because the cross takes its room out of the *string* rather than
// out of the box. Nothing can do that at the leading end: where the text
// begins is the engine's, not the widget's, and twenty-five engines draw
// a field. So the well is drawn by this box and the field inside it is
// [TextField.Frameless] — the frame once, around everything.
//
//	addr := widgets.NewTextField(url, "Search or enter address", nil)
//	addr.Frameless = true
//	omnibox := widgets.NewFieldBox(
//	    []widget.Component{widgets.ToolIconBtn(style.IconLock, "", onInfo).Widget},
//	    addr,
//	    []widget.Component{star},
//	)
//
// The well is painted in the field's own state, so it takes the focus
// ring when the field inside it has the focus: one control to look at,
// not three.
type FieldBox struct {
	widget.Base
	// MinHeight is the least the well may be, as a 1x design length.
	// Zero takes the look's field height, which is what an ordinary form
	// wants; a browser's address capsule is taller than the fields on a
	// dialog and says so.
	MinHeight float32
	row       *FlexBox
	field     widget.Component
}

// NewFieldBox puts lead, field and trail in one well, with field taking
// the width the others leave.
func NewFieldBox(lead []widget.Component, field widget.Component, trail []widget.Component) *FieldBox {
	kids := make([]widget.Component, 0, len(lead)+len(trail)+1)
	kids = append(kids, lead...)
	kids = append(kids, field)
	kids = append(kids, trail...)
	f := &FieldBox{row: NewRow(kids...).WithGap(2), field: field}
	f.row.AddFlex(field, 1)
	if t, ok := field.(*TextField); ok {
		// The box is the frame now; a second one inside it would draw a
		// well in a well.
		t.Frameless = true
	}
	f.Init(f)
	f.Base.Add(f.row)
	return f
}

// Field is the control the well is drawn around.
func (f *FieldBox) Field() widget.Component { return f.field }

// Content is the row inside the well, for an application adding to it.
func (f *FieldBox) Content() *FlexBox { return f.row }

// pad is the room between the well's edge and what is in it: the look's
// own field padding, halved, because the things inside a field sit closer
// to its edge than text does — a browser's lock mark is nearly against
// the pill.
func (f *FieldBox) pad() float32 {
	p := f.Look().Metrics().FieldPad
	if p <= 0 {
		p = 8
	}
	return float32(int(p*0.5 + 0.5))
}

func (f *FieldBox) Measure(c layout.Constraints) paintengine2d.Point {
	p := f.pad()
	sz := f.row.Measure(c.Inset(p*2, 0))
	h := max(sz.Y, style.FieldHeight(f.Look().Metrics()))
	if f.MinHeight > 0 {
		h = max(h, style.Dip(f.Look(), f.MinHeight))
	}
	return c.Constrain(paintengine2d.Pt(sz.X+p*2, h))
}

func (f *FieldBox) Arrange(r paintengine2d.Rect) {
	f.SetBounds(r)
	p := f.pad()
	b := f.LocalBounds()
	f.row.Arrange(paintengine2d.XYWH(b.Min.X+p, b.Min.Y, max(b.Dx()-p*2, 0), b.Dy()))
}

// state is the well's state, taken from the field inside it so the ring
// and the hover follow the thing the person is actually typing in.
func (f *FieldBox) state() style.ControlState {
	st := f.State()
	if c, ok := f.field.(interface{ State() style.ControlState }); ok {
		st = c.State()
	}
	if widget.FocusWithin(f) {
		st |= style.StateFocused
	}
	return st
}

// Paint draws the well and nothing else: the children paint over it.
//
// An empty string rather than the field's own, because the field inside
// draws its own text, its caret and its selection. Asking the engine for
// the frame alone is what keeps this working across every look without
// another seam: a field with nothing in it *is* the well.
func (f *FieldBox) Paint(ctx *paintengine2d.Context) {
	f.Look().DrawTextField(ctx, f.LocalBounds(), f.state(), "", "", -1, 0, 0, false, 0, f.Look().Font())
}
