package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// Measure answers a question; Arrange decides.
//
// ScrollView kept what the child measured as the size it scrolls, and the
// range, the clamp and the thumb all read it. So a measure *after* the layout
// left the view scrolling the content as it would be at that other width: a
// splitter asking its panes' minimum widths probes at a quarter of the natural
// width, and a parent may measure unbounded. A mail client's Security tab,
// scrolled to the end, read a range of 573 px before such a probe and 2979
// after it — the thumb half-way down at the end of the content, and the wheel
// going on into nothing.

// reflowing is a child whose height depends on the width it is given, which is
// what makes a probe at another width produce another answer.
func reflowing(t *testing.T) *Label {
	t.Helper()
	l := NewLabel(strings.Repeat("a paragraph that wraps, and wraps again, and goes on. ", 12))
	l.Wrap = true
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	return l
}

func TestMeasuringAScrollViewDoesNotChangeWhatItScrolls(t *testing.T) {
	s := NewScrollView(reflowing(t))
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 400, 120))
	s.ScrollTo(s.MaxOffset())

	laid, off := s.MaxOffset(), s.OffsetY
	if laid <= 0 {
		t.Fatal("the content does not overflow: there is no range to disturb")
	}

	for _, probe := range []struct {
		what string
		c    layout.Constraints
	}{
		{"a splitter's minimum-width probe", layout.Loose(100, 120)},
		{"a parent measuring unbounded", layout.Unbounded()},
		{"a narrow measure", layout.Loose(60, 120)},
	} {
		s.Measure(probe.c)
		if got := s.MaxOffset(); got != laid {
			t.Errorf("after %s the range is %.0f, want the laid-out %.0f", probe.what, got, laid)
		}
		if got := s.OffsetY; got != off {
			t.Errorf("after %s the offset moved to %.0f, want %.0f", probe.what, got, off)
		}
	}
}

// And the measure still answers correctly for the width it was asked about:
// not writing the value down is not the same as not working it out.
func TestAScrollViewStillMeasuresItsContent(t *testing.T) {
	s := NewScrollView(reflowing(t))
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	s.ShrinkToContent = true
	wide := s.Measure(layout.Loose(400, 4000)).Y
	narrow := s.Measure(layout.Loose(150, 4000)).Y
	if narrow <= wide {
		t.Errorf("measured %.0f tall at 150 wide and %.0f at 400: the narrow one must wrap to more", narrow, wide)
	}
}
