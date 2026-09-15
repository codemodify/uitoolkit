package widgets

import (
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// NumberField is a numeric TextField with stepper buttons (spinner). The
// look decides where the buttons go (style.SpinBoxStyle, Qt's CC_SpinBox):
// inside the field's frame, sharing it, for Windows, KDE and GNOME (GNOME
// side by side, "− +"); a stepper beside the field for Mac OS and Motif.
type NumberField struct {
	widget.Base
	Min, Max, Step, Value float64
	Decimals              int
	OnChange              func(float64)
	Tip                   string
	field                 *TextField
	syncing               bool
	upHover, downHover    bool
	upPress, downPress    bool
}

// NewNumberField builds a spinner. step <= 0 defaults to 1.
func NewNumberField(min, max, value, step float64, on func(float64)) *NumberField {
	if max < min {
		max = min
	}
	if step <= 0 {
		step = 1
	}
	n := &NumberField{Min: min, Max: max, Step: step, Value: clamp64(value, min, max), OnChange: on}
	if !isIntStep(step) {
		n.Decimals = 2
	}
	n.Init(n)
	// One tab stop: the inner editor. Spinner keys bubble from the field.
	n.SetWantsFocus(false)
	n.field = NewTextField(n.format(n.Value), "", n.onField)
	n.field.Accept = n.accept
	n.field.OnSubmit = func(string) { n.commit() }
	n.field.OnFocusLost = func() { n.commit() }
	n.Add(n.field)
	return n
}

// NewSpinner is an alias for NewNumberField.
func NewSpinner(min, max, value, step float64, on func(float64)) *NumberField {
	return NewNumberField(min, max, value, step, on)
}

func (n *NumberField) Tooltip() string { return n.Tip }

// Field is the inner editor.
func (n *NumberField) Field() *TextField { return n.field }

func (n *NumberField) SetEnabled(v bool) {
	n.Base.SetEnabled(v)
	if n.field != nil {
		n.field.SetEnabled(v)
	}
}

func (n *NumberField) SetValue(v float64) {
	v = clamp64(v, n.Min, n.Max)
	if n.Value == v && n.field != nil && n.field.Text == n.format(v) {
		return
	}
	n.Value = v
	n.syncing = true
	if n.field != nil {
		n.field.SetText(n.format(v))
	}
	n.syncing = false
	n.Invalidate()
	if n.OnChange != nil {
		n.OnChange(v)
	}
}

func (n *NumberField) Measure(c layout.Constraints) paintengine2d.Point {
	h := style.FieldHeight(n.Look().Metrics())
	return c.Constrain(paintengine2d.Pt(120, h))
}

func (n *NumberField) spinnerW() float32 {
	w := n.Look().Metrics().SpinnerW
	if w <= 0 {
		return 22
	}
	return w
}

// spinParts lays the spin box out for its look: the editor's rect, the
// button box and its up / down halves, and whether the buttons share the
// field's frame (then the editor is frameless and the box sits inside the
// frame).
func (n *NumberField) spinParts() (field, box, up, down paintengine2d.Rect, inside bool) {
	lk := n.Look()
	b := n.LocalBounds()
	ss := style.SpinBoxStyleOf(lk)
	sw := n.spinnerW()
	if !ss.Inside {
		box = paintengine2d.XYWH(b.Dx()-sw, 0, sw, b.Dy())
		field = paintengine2d.XYWH(0, 0, b.Dx()-sw, b.Dy())
	} else {
		// Inside the field's frame, clear of its border.
		fi := style.ViewFrameInsetsOf(lk)
		px := style.Dip(lk, 1)
		in := style.Insets{Top: max(fi.Top, px), Right: max(fi.Right, px), Bottom: max(fi.Bottom, px), Left: max(fi.Left, px)}
		w := sw
		if ss.Across {
			w = min(b.Dy()*2, b.Dx()*0.5)
		}
		box = paintengine2d.XYWH(b.Dx()-in.Right-w, in.Top, w, b.Dy()-in.Top-in.Bottom)
		field = paintengine2d.XYWH(0, 0, box.Min.X, b.Dy())
		inside = true
	}
	if ss.Across {
		mid := (box.Min.X + box.Max.X) * 0.5
		down = paintengine2d.Rect{Min: box.Min, Max: paintengine2d.Pt(mid, box.Max.Y)}
		up = paintengine2d.Rect{Min: paintengine2d.Pt(mid, box.Min.Y), Max: box.Max}
	} else {
		mid := (box.Min.Y + box.Max.Y) * 0.5
		up = paintengine2d.Rect{Min: box.Min, Max: paintengine2d.Pt(box.Max.X, mid)}
		down = paintengine2d.Rect{Min: paintengine2d.Pt(box.Min.X, mid), Max: box.Max}
	}
	return
}

func (n *NumberField) Arrange(r paintengine2d.Rect) {
	n.SetBounds(r)
	field, _, _, _, inside := n.spinParts()
	if n.field != nil {
		n.field.Frameless = inside
		n.field.Arrange(field)
	}
}

func (n *NumberField) spinnerBox() paintengine2d.Rect {
	_, box, _, _, _ := n.spinParts()
	return box
}

func (n *NumberField) Paint(ctx *paintengine2d.Context) {
	lk := n.Look()
	_, box, _, _, inside := n.spinParts()
	st := n.State()
	if inside {
		// One frame round text and buttons: the field's, drawn across the
		// whole box (focused while the editor inside is).
		fst := st
		if n.focusHere() {
			fst |= style.StateFocused
		}
		lk.DrawTextField(ctx, n.LocalBounds(), fst, "", "", 0, 0, 0, false, 0, nil)
		st |= style.StateFrameless
	}
	lk.DrawSpinner(ctx, box, st, n.upHover, n.downHover, n.upPress, n.downPress)
}

func (n *NumberField) HitTest(local paintengine2d.Point) widget.Component {
	if !n.Visible() {
		return nil
	}
	if !n.LocalBounds().Contains(local) {
		return nil
	}
	if n.spinnerBox().Contains(local) {
		return n
	}
	if n.field != nil {
		fb := n.field.Bounds()
		lp := paintengine2d.Pt(local.X-fb.Min.X, local.Y-fb.Min.Y)
		if hit := n.field.HitTest(lp); hit != nil {
			return hit
		}
	}
	return n
}

func (n *NumberField) MouseMove(e widget.MouseEvent) bool {
	_, _, ur, dr, _ := n.spinParts()
	up, down := ur.Contains(e.Pos), dr.Contains(e.Pos)
	if up != n.upHover || down != n.downHover {
		n.upHover, n.downHover = up, down
		n.Invalidate()
	}
	return true
}

func (n *NumberField) MouseExit() {
	n.upHover, n.downHover = false, false
	n.upPress, n.downPress = false, false
	n.Invalidate()
}

func (n *NumberField) MousePress(e widget.MouseEvent) bool {
	if !n.Enabled() {
		return false
	}
	n.MarkPointerFocus()
	if n.field != nil {
		n.field.RequestFocus()
	} else {
		n.RequestFocus()
	}
	_, _, ur, dr, _ := n.spinParts()
	switch {
	case ur.Contains(e.Pos):
		n.upPress = true
		n.nudge(1)
	case dr.Contains(e.Pos):
		n.downPress = true
		n.nudge(-1)
	default:
		return true
	}
	n.Invalidate()
	return true
}

func (n *NumberField) MouseRelease(widget.MouseEvent) bool {
	n.upPress, n.downPress = false, false
	n.Invalidate()
	return true
}

// MouseWheel steps the value only while the spinner has focus. An unfocused
// spinner must not change under a passing pointer, nor block the page scroll.
func (n *NumberField) MouseWheel(e widget.MouseEvent) bool {
	if !n.Enabled() || !n.focusHere() {
		return false
	}
	if e.Scroll.Y == 0 {
		return false
	}
	if e.Scroll.Y < 0 {
		n.nudge(1)
	} else if e.Scroll.Y > 0 {
		n.nudge(-1)
	}
	return true
}

// focusHere reports whether the spinner or its inner editor holds focus.
func (n *NumberField) focusHere() bool {
	if n.Focused() {
		return true
	}
	return n.field != nil && n.field.Focused()
}

func (n *NumberField) KeyPress(e widget.KeyEvent) bool {
	if !n.Enabled() {
		return false
	}
	mult := 1.0
	if e.Mods.Shift() {
		mult = 10
	}
	switch e.Key {
	case platform.KeyUp:
		n.nudge(mult)
		return true
	case platform.KeyDown:
		n.nudge(-mult)
		return true
	case platform.KeyPageUp:
		n.nudge(10)
		return true
	case platform.KeyPageDown:
		n.nudge(-10)
		return true
	case platform.KeyHome:
		// The inner editor consumes plain Home / End for its caret and leaves
		// the Ctrl form to bubble here, which is how Min / Max became
		// reachable at all. Both forms work when the spinner is the target.
		n.SetValue(n.Min)
		return true
	case platform.KeyEnd:
		n.SetValue(n.Max)
		return true
	}
	return false
}

// nudge parses whatever is typed and applies the step in a single SetValue,
// so stepping after typing reports one OnChange instead of two.
func (n *NumberField) nudge(dir float64) {
	v := n.Value
	if n.field != nil {
		if parsed, err := strconv.ParseFloat(n.field.Text, 64); err == nil {
			v = parsed
		}
	}
	n.SetValue(clamp64(v, n.Min, n.Max) + dir*n.Step)
}

func (n *NumberField) onField(s string) {
	if n.syncing {
		return
	}
	if s == "" || s == "-" || s == "." || s == "-." {
		return
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return
	}
	v = clamp64(v, n.Min, n.Max)
	if n.Value == v {
		return
	}
	n.Value = v
	n.Invalidate()
	if n.OnChange != nil {
		n.OnChange(v)
	}
}

func (n *NumberField) commit() {
	v := n.Value
	if n.field != nil {
		if parsed, err := strconv.ParseFloat(n.field.Text, 64); err == nil {
			v = parsed
		}
	}
	n.SetValue(v)
}

func (n *NumberField) accept(s string) bool {
	if s == "" || s == "-" {
		return true
	}
	if n.Decimals == 0 && (strings.Contains(s, ".") || strings.ContainsAny(s, "eE")) {
		return false
	}
	if strings.HasSuffix(s, ".") && n.Decimals != 0 {
		s = strings.TrimSuffix(s, ".")
		if s == "" || s == "-" {
			return true
		}
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func (n *NumberField) format(v float64) string {
	if n.Decimals > 0 {
		return strconv.FormatFloat(v, 'f', n.Decimals, 64)
	}
	if n.Decimals == 0 || isIntStep(n.Step) && v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func clamp64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func isIntStep(step float64) bool {
	return step == float64(int64(step))
}
