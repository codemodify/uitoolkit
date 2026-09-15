package widgets

import (
	"fmt"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Calendar is a month view — QCalendarWidget, GtkCalendar, WPF's Calendar:
// the month and year between previous and next buttons, the weekday names,
// and six weeks of days painted as the look's tool buttons (the selected day
// pressed in, other months' days greyed, today ringed). Arrow keys move a
// day or a week, PageUp / PageDown a month, Home / End to the month's ends,
// Return picks; the wheel pages months.
type Calendar struct {
	widget.Base
	// Selected is the chosen date (its time of day is ignored).
	Selected time.Time
	// FirstWeekday starts each week row (Monday by default, as ISO 8601).
	FirstWeekday time.Weekday
	// OnSelect runs when the user picks a day (click, Return, Space).
	OnSelect func(time.Time)
	// Today is the clock marking today (nil: time.Now).
	Today func() time.Time

	shown     time.Time // first of the month on show
	hot, down int       // cell under the pointer / pressed: 0..41 days, calPrev, calNext
}

const (
	calNone = -1
	calPrev = -2
	calNext = -3
)

// NewCalendar shows the month of selected (today when zero).
func NewCalendar(selected time.Time, on func(time.Time)) *Calendar {
	c := &Calendar{FirstWeekday: time.Monday, OnSelect: on, hot: calNone, down: calNone}
	if selected.IsZero() {
		selected = time.Now()
	}
	c.Selected = dateOnly(selected)
	c.shown = monthOf(c.Selected)
	c.Init(c)
	c.SetWantsFocus(true)
	return c
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func monthOf(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

// Shown is the first day of the month on show.
func (c *Calendar) Shown() time.Time { return c.shown }

// ShowMonth shows the month of t without selecting anything.
func (c *Calendar) ShowMonth(t time.Time) {
	c.shown = monthOf(t)
	c.Invalidate()
}

func (c *Calendar) today() time.Time {
	if c.Today != nil {
		return dateOnly(c.Today())
	}
	return dateOnly(time.Now())
}

// firstCell is the date in the grid's top-left cell.
func (c *Calendar) firstCell() time.Time {
	off := (int(c.shown.Weekday()) - int(c.FirstWeekday) + 7) % 7
	return c.shown.AddDate(0, 0, -off)
}

// dayAt is the date of cell i (0..41).
func (c *Calendar) dayAt(i int) time.Time { return c.firstCell().AddDate(0, 0, i) }

// cellOf is the cell of date d, or -1 when it is not on show.
func (c *Calendar) cellOf(d time.Time) int {
	i := int(d.Sub(c.firstCell()).Hours()/24 + 0.5)
	if i < 0 || i > 41 {
		return -1
	}
	return i
}

// geometry is the header height, weekday row height and day cell size.
func (c *Calendar) geometry() (head, wk, cw, ch float32) {
	lk := c.Look()
	m := lk.Metrics()
	head = m.ControlH
	if head <= 0 {
		head = 28
	}
	wk = lk.MutedFont().Height() + 6
	b := c.LocalBounds()
	cw = b.Dx() / 7
	ch = (b.Dy() - head - wk - 4) / 6
	return
}

func (c *Calendar) Measure(cs layout.Constraints) paintengine2d.Point {
	lk := c.Look()
	m := lk.Metrics()
	head := m.ControlH
	if head <= 0 {
		head = 28
	}
	// Room for two digits inside the look's tool-button padding.
	cell := style.ControlFontOf(lk, style.RoleTool).Advance("00") + style.Dip(lk, 30)
	if min := m.ControlH * 1.4; cell < min {
		cell = min
	}
	rowH := m.ControlH
	if rowH <= 0 {
		rowH = 26
	}
	w := cell * 7
	h := head + lk.MutedFont().Height() + 6 + 4 + rowH*6
	return cs.Constrain(paintengine2d.Pt(w, h))
}

func (c *Calendar) Arrange(r paintengine2d.Rect) { c.SetBounds(r) }

// buttonRect is the previous (-1) or next (+1) month button.
func (c *Calendar) buttonRect(dir int) paintengine2d.Rect {
	head, _, _, _ := c.geometry()
	b := c.LocalBounds()
	if dir < 0 {
		return paintengine2d.XYWH(0, 0, head, head)
	}
	return paintengine2d.XYWH(b.Dx()-head, 0, head, head)
}

// cellRect is day cell i in local coordinates.
func (c *Calendar) cellRect(i int) paintengine2d.Rect {
	head, wk, cw, ch := c.geometry()
	return paintengine2d.XYWH(float32(i%7)*cw, head+wk+4+float32(i/7)*ch, cw, ch)
}

func (c *Calendar) hit(p paintengine2d.Point) int {
	if c.buttonRect(-1).Contains(p) {
		return calPrev
	}
	if c.buttonRect(1).Contains(p) {
		return calNext
	}
	for i := 0; i < 42; i++ {
		if c.cellRect(i).Contains(p) {
			return i
		}
	}
	return calNone
}

func (c *Calendar) Paint(ctx *paintengine2d.Context) {
	lk := c.Look()
	p := lk.Palette()
	head, wk, cw, _ := c.geometry()
	b := c.LocalBounds()
	st := c.State()
	// Header: ‹ month year ›.
	for _, dir := range []int{-1, 1} {
		r := c.buttonRect(dir)
		bs := st &^ (style.StateHovered | style.StatePressed | style.StateFocused)
		part := calPrev
		arrow := style.DirLeft
		if dir > 0 {
			part, arrow = calNext, style.DirRight
		}
		if c.hot == part {
			bs |= style.StateHovered
			if c.down == part {
				bs |= style.StatePressed
			}
		}
		lk.DrawToolButton(ctx, r, bs, "", style.IconNone)
		style.DrawArrowOf(lk, ctx, r.Inset(r.Dx()*0.25), arrow, p.Text)
	}
	title := fmt.Sprintf("%s %d", c.shown.Month(), c.shown.Year())
	tf := lk.BoldFont()
	if tf == nil {
		tf = lk.Font()
	}
	tf.Draw(ctx, title, paintengine2d.Pt((b.Dx()-tf.Advance(title))*0.5, (head-tf.Height())*0.5), p.Text)
	// Weekday names.
	mf := lk.MutedFont()
	for i := 0; i < 7; i++ {
		name := time.Weekday((int(c.FirstWeekday) + i) % 7).String()[:2]
		mf.Draw(ctx, name, paintengine2d.Pt(float32(i)*cw+(cw-mf.Advance(name))*0.5, head+3), p.TextMuted)
	}
	ctx.DrawRect(paintengine2d.XYWH(0, head+wk+1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
	// Days.
	today := c.today()
	focused := c.Focused() && widget.WindowActive(c)
	for i := 0; i < 42; i++ {
		d := c.dayAt(i)
		r := c.cellRect(i).Inset(1)
		ds := style.StateNone
		if d.Month() != c.shown.Month() {
			ds |= style.StateDisabled
		}
		if d.Equal(c.Selected) {
			ds |= style.StateToggle | style.StateChecked
			if focused {
				ds |= style.StateFocused
			}
		}
		if c.hot == i {
			ds |= style.StateHovered
			if c.down == i {
				ds |= style.StatePressed
			}
		}
		lk.DrawToolButton(ctx, r, ds, fmt.Sprint(d.Day()), style.IconNone)
		if d.Equal(today) {
			w := style.Dip(lk, 1)
			ctx.DrawRoundRect(r.Inset(w*1.5), lk.Metrics().RadiusSmall, lk.Metrics().RadiusSmall, paintengine2d.StrokePaint(p.Accent, w))
		}
	}
}

// pick selects d, shows its month and reports it.
func (c *Calendar) pick(d time.Time, report bool) {
	c.Selected = dateOnly(d)
	if monthOf(c.Selected) != c.shown {
		c.shown = monthOf(c.Selected)
	}
	c.Invalidate()
	if report && c.OnSelect != nil {
		c.OnSelect(c.Selected)
	}
}

func (c *Calendar) MouseMove(e widget.MouseEvent) bool {
	if h := c.hit(e.Pos); h != c.hot {
		c.hot = h
		c.Invalidate()
	}
	return true
}

func (c *Calendar) MouseExit() {
	c.hot, c.down = calNone, calNone
	c.Invalidate()
	c.Base.MouseExit()
}

func (c *Calendar) MousePress(e widget.MouseEvent) bool {
	if e.Button != platform.ButtonLeft {
		return false
	}
	c.RequestFocus()
	c.down = c.hit(e.Pos)
	c.hot = c.down
	c.Invalidate()
	return true
}

func (c *Calendar) MouseRelease(e widget.MouseEvent) bool {
	down := c.down
	c.down = calNone
	c.Invalidate()
	if down == calNone || c.hit(e.Pos) != down {
		return true
	}
	switch down {
	case calPrev:
		c.ShowMonth(c.shown.AddDate(0, -1, 0))
	case calNext:
		c.ShowMonth(c.shown.AddDate(0, 1, 0))
	default:
		c.pick(c.dayAt(down), true)
	}
	return true
}

func (c *Calendar) MouseWheel(e widget.MouseEvent) bool {
	switch {
	case e.Scroll.Y > 0:
		c.ShowMonth(c.shown.AddDate(0, 1, 0))
	case e.Scroll.Y < 0:
		c.ShowMonth(c.shown.AddDate(0, -1, 0))
	default:
		return false
	}
	return true
}

func (c *Calendar) KeyPress(e widget.KeyEvent) bool {
	d := c.Selected
	switch e.Key {
	case platform.KeyLeft:
		d = d.AddDate(0, 0, -1)
	case platform.KeyRight:
		d = d.AddDate(0, 0, 1)
	case platform.KeyUp:
		d = d.AddDate(0, 0, -7)
	case platform.KeyDown:
		d = d.AddDate(0, 0, 7)
	case platform.KeyPageUp:
		d = d.AddDate(0, -1, 0)
	case platform.KeyPageDown:
		d = d.AddDate(0, 1, 0)
	case platform.KeyHome:
		d = monthOf(d)
	case platform.KeyEnd:
		d = monthOf(d).AddDate(0, 1, -1)
	case platform.KeyReturn, platform.KeySpace:
		if c.OnSelect != nil {
			c.OnSelect(c.Selected)
		}
		return true
	default:
		return false
	}
	c.pick(d, false)
	return true
}

// DateField is a date entry with a drop-down calendar (QDateEdit with a
// popup, WPF's DatePicker): it looks like the look's combo box; Up / Down
// step a day, Alt+Down, F4, Space or a click open the calendar.
type DateField struct {
	widget.Base
	Value    time.Time
	OnChange func(time.Time)
	// Format renders the date (default "2006-01-02", ISO 8601).
	Format string
	open   bool
	pop    *calendarPopup
}

// NewDateField shows value (today when zero).
func NewDateField(value time.Time, on func(time.Time)) *DateField {
	if value.IsZero() {
		value = time.Now()
	}
	f := &DateField{Value: dateOnly(value), OnChange: on}
	f.Init(f)
	f.SetWantsFocus(true)
	return f
}

func (f *DateField) text() string {
	layoutStr := f.Format
	if layoutStr == "" {
		layoutStr = "2006-01-02"
	}
	return f.Value.Format(layoutStr)
}

func (f *DateField) Measure(c layout.Constraints) paintengine2d.Point {
	lk := f.Look()
	m := lk.Metrics()
	h := m.ComboH
	if h <= 0 {
		h = m.ControlH
	}
	w := style.ControlFontOf(lk, style.RoleCombo).Advance("0000-00-00") + m.Pad*2 + 40
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (f *DateField) Arrange(r paintengine2d.Rect) { f.SetBounds(r) }

func (f *DateField) Paint(ctx *paintengine2d.Context) {
	f.Look().DrawComboBox(ctx, f.LocalBounds(), f.State(), f.text(), f.open)
}

// set changes the value and reports it.
func (f *DateField) set(d time.Time) {
	d = dateOnly(d)
	if d.Equal(f.Value) {
		return
	}
	f.Value = d
	f.Invalidate()
	if f.OnChange != nil {
		f.OnChange(d)
	}
}

func (f *DateField) MousePress(e widget.MouseEvent) bool {
	if e.Button != platform.ButtonLeft {
		return false
	}
	f.RequestFocus()
	if f.open {
		f.Close()
	} else {
		f.Open()
	}
	return true
}

func (f *DateField) KeyPress(e widget.KeyEvent) bool {
	switch {
	case e.Key == platform.KeyUp && !e.Mods.Alt():
		f.set(f.Value.AddDate(0, 0, -1))
	case e.Key == platform.KeyDown && !e.Mods.Alt():
		f.set(f.Value.AddDate(0, 0, 1))
	case e.Key == platform.KeyDown, e.Key == platform.KeyF4, e.Key == platform.KeySpace:
		f.Open()
	default:
		return false
	}
	return true
}

// Open drops the calendar down under the field.
func (f *DateField) Open() {
	if f.open {
		return
	}
	cal := NewCalendar(f.Value, nil)
	pop := newCalendarPopup(cal, f)
	cal.OnSelect = func(d time.Time) {
		f.set(d)
		f.Close()
	}
	o := widget.DeviceOrigin(f)
	b := f.LocalBounds()
	anchor := paintengine2d.XYWH(o.X, o.Y, b.Dx(), b.Dy())
	widget.PlacePopupForAnchor(f, pop, anchor, 0, 2)
	if widget.ShowPopup(f, pop) {
		f.open = true
		f.pop = pop
		f.Invalidate()
		cal.RequestFocus()
	}
}

// Close takes the calendar down.
func (f *DateField) Close() {
	if !f.open {
		return
	}
	f.open = false
	widget.DismissPopup(f)
	f.Invalidate()
}

// calendarPopup is the drop-down frame around a Calendar.
type calendarPopup struct {
	widget.Base
	cal   *Calendar
	field *DateField
}

func newCalendarPopup(cal *Calendar, field *DateField) *calendarPopup {
	p := &calendarPopup{cal: cal, field: field}
	p.Init(p)
	p.Add(cal)
	return p
}

func (p *calendarPopup) pad() float32 { return style.Dip(p.Look(), 6) }

func (p *calendarPopup) Measure(c layout.Constraints) paintengine2d.Point {
	pad := p.pad()
	sz := p.cal.Measure(c.Inset(pad*2, pad*2))
	return c.Constrain(paintengine2d.Pt(sz.X+pad*2, sz.Y+pad*2))
}

func (p *calendarPopup) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	pad := p.pad()
	p.cal.Arrange(paintengine2d.XYWH(pad, pad, r.Dx()-pad*2, r.Dy()-pad*2))
}

func (p *calendarPopup) Paint(ctx *paintengine2d.Context) {
	p.Look().DrawMenuFrame(ctx, p.LocalBounds())
}

func (p *calendarPopup) KeyPress(e widget.KeyEvent) bool {
	if e.Key == platform.KeyEscape {
		p.field.Close()
		return true
	}
	return false
}

// Dismissed hands focus back to the field and clears its open state.
func (p *calendarPopup) Dismissed() {
	if p.field.open {
		p.field.open = false
		p.field.Invalidate()
	}
	widget.RestoreFocus(p, p.field, true)
}
