package widgets

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.Local) }

// The month grid starts on the first weekday, arrow keys walk days and
// weeks, PageDown turns the month, and a click picks the day under it.
func TestCalendarGridAndKeys(t *testing.T) {
	var picked time.Time
	c := NewCalendar(day(2026, time.September, 14), func(d time.Time) { picked = d })
	c.Today = func() time.Time { return day(2026, time.September, 14) }
	c.SetHost(&fakeWindow{look: style.DarkLook()})
	c.Arrange(paintengine2d.XYWH(0, 0, 280, 260))
	// 1 September 2026 is a Tuesday: the Monday before, 31 August, leads.
	if got := c.dayAt(0); !got.Equal(day(2026, time.August, 31)) {
		t.Fatalf("first cell %v, want Mon 31 Aug 2026", got)
	}
	if i := c.cellOf(day(2026, time.September, 14)); i != 14 {
		t.Fatalf("14 Sep in cell %d, want 14", i)
	}
	c.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	c.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if !c.Selected.Equal(day(2026, time.September, 22)) {
		t.Fatalf("right, down -> %v, want 22 Sep", c.Selected)
	}
	c.KeyPress(widget.KeyEvent{Key: platform.KeyPageDown})
	if !c.Selected.Equal(day(2026, time.October, 22)) || c.Shown().Month() != time.October {
		t.Fatalf("page down -> %v (showing %v), want 22 Oct", c.Selected, c.Shown().Month())
	}
	// Click the cell of 5 October.
	r := c.cellRect(c.cellOf(day(2026, time.October, 5)))
	p := paintengine2d.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	c.MousePress(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	c.MouseRelease(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	if !picked.Equal(day(2026, time.October, 5)) {
		t.Fatalf("click picked %v, want 5 Oct", picked)
	}
	// It paints (every look draws its days as tool buttons).
	img := paintengine2d.NewImage(280, 260)
	c.Paint(paintengine2d.NewContext(img))
}

// The date field steps a day with Up / Down and drops a calendar down.
func TestDateFieldStepsAndOpens(t *testing.T) {
	var got time.Time
	f := NewDateField(day(2026, time.December, 31), func(d time.Time) { got = d })
	host := &fakeWindow{look: style.DarkLook()}
	f.SetHost(host)
	f.Arrange(paintengine2d.XYWH(0, 0, 160, 28))
	f.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if !got.Equal(day(2027, time.January, 1)) || f.text() != "2027-01-01" {
		t.Fatalf("down -> %v %q, want 2027-01-01", got, f.text())
	}
	f.KeyPress(widget.KeyEvent{Key: platform.KeyDown, Mods: platform.ModAlt})
	if !f.open || host.popup == nil {
		t.Fatal("Alt+Down should drop the calendar down")
	}
	f.pop.cal.OnSelect(day(2027, time.February, 3))
	if !f.Value.Equal(day(2027, time.February, 3)) || f.open {
		t.Fatalf("picking in the calendar: value %v open %v", f.Value, f.open)
	}
}
