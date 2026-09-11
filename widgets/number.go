package widgets

import (
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// NumberField is a numeric TextField with stepper buttons (spinner).
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
	n.SetWantsFocus(true)
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
	h := n.Look().Metrics().ControlH
	return c.Constrain(paintengine2d.Pt(120, h))
}

func (n *NumberField) spinnerW() float32 {
	w := n.Look().Metrics().SpinnerW
	if w <= 0 {
		return 22
	}
	return w
}

func (n *NumberField) Arrange(r paintengine2d.Rect) {
	n.SetBounds(r)
	sw := n.spinnerW()
	if n.field != nil {
		n.field.Arrange(paintengine2d.XYWH(0, 0, r.Dx()-sw, r.Dy()))
	}
}

func (n *NumberField) spinnerBox() paintengine2d.Rect {
	b := n.LocalBounds()
	sw := n.spinnerW()
	return paintengine2d.XYWH(b.Dx()-sw, 0, sw, b.Dy())
}

func (n *NumberField) Paint(ctx *paintengine2d.Context) {
	n.Look().DrawSpinner(ctx, n.spinnerBox(), n.State(), n.upHover, n.downHover, n.upPress, n.downPress)
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
	sb := n.spinnerBox()
	mid := (sb.Min.Y + sb.Max.Y) * 0.5
	up := sb.Contains(e.Pos) && e.Pos.Y < mid
	down := sb.Contains(e.Pos) && e.Pos.Y >= mid
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
	n.RequestFocus()
	sb := n.spinnerBox()
	if !sb.Contains(e.Pos) {
		return true
	}
	mid := (sb.Min.Y + sb.Max.Y) * 0.5
	if e.Pos.Y < mid {
		n.upPress = true
		n.nudge(1)
	} else {
		n.downPress = true
		n.nudge(-1)
	}
	n.Invalidate()
	return true
}

func (n *NumberField) MouseRelease(widget.MouseEvent) bool {
	n.upPress, n.downPress = false, false
	n.Invalidate()
	return true
}

func (n *NumberField) MouseWheel(e widget.MouseEvent) bool {
	if !n.Enabled() {
		return false
	}
	if e.Scroll.Y < 0 {
		n.nudge(1)
	} else if e.Scroll.Y > 0 {
		n.nudge(-1)
	}
	return true
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
		n.SetValue(n.Min)
		return true
	case platform.KeyEnd:
		n.SetValue(n.Max)
		return true
	}
	return false
}

func (n *NumberField) nudge(dir float64) {
	n.commit()
	n.SetValue(n.Value + dir*n.Step)
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
