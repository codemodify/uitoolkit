package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A label stretched to its parent's width is measured at the width it is
// *offered*, and layout then rounds each edge of its box to the nearest pixel
// (widget.PixelRect) — so a label offered 248.5 is handed 248 or 249 depending
// on where the box sits, and 248 is a width it was never measured at. It wraps
// to one line more and the lines, centred in a box tall enough for one fewer,
// lose the first and the last.
//
// That is the symptom the icon-label fix was about, back at a width nobody
// measured. It only shows at fractional scales, where a column's inner width
// is fractional: a mail client found 24 cut labels and one hidden line in about
// 13,500 layouts, all at 1.25.

// wrapText is long enough to wrap at every width under test.
const wrapText = "This message came from a server that is not the one its sender claims, " +
	"which is usually a mistake and occasionally something to look at twice."

func TestAStretchedWrappingLabelFitsTheBoxItIsHanded(t *testing.T) {
	bad := 0
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		lk := style.WithScale(style.DarkLook(), scale)
		for w := float32(120); w <= 420; w += 0.5 {
			l := NewLabel(wrapText)
			l.Wrap = true
			l.SetHost(&fakeWindow{look: lk})
			got := l.Measure(layout.Loose(w, 4000))

			// Stretched to the parent's width, so the box comes from what it
			// was *offered*; layout rounds each edge, and the narrowest that
			// can produce is floor(w).
			handed := float32(int(w))
			l.Arrange(paintengine2d.XYWH(0, 0, handed, got.Y))
			f := l.font()
			painted := len(l.layoutLines(f, l.wrapWidth(f, l.LocalBounds().Dx())))
			need := f.Height()*float32(painted) + 2
			if l.LocalBounds().Dy()+0.01 < need {
				bad++
				if bad <= 4 {
					t.Errorf("%.2fx offered %.1f: measured a box %.0f tall, handed %.0f wide it paints %d lines needing %.1f",
						scale, w, got.Y, handed, painted, need)
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d widths cut a line", bad)
	}
}

// The same for a view that fits its text: a path measured as two lines and laid
// out a pixel narrower wraps to three, and the third is behind a scroll bar.
func TestAFitRowsViewFitsTheBoxItIsHanded(t *testing.T) {
	path := "/home/person/.local/share/comms-mail/accounts/work/secrets.enc"
	bad := 0
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		lk := style.WithScale(style.DarkLook(), scale)
		for w := float32(120); w <= 420; w += 0.5 {
			v := NewTextView(path, "")
			v.MinRows, v.FitRows = 1, true
			v.SetHost(&fakeWindow{look: lk})
			got := v.Measure(layout.Loose(w, 4000))
			handed := float32(int(w))
			v.Arrange(paintengine2d.XYWH(0, 0, handed, got.Y))
			rows := v.rowsFor(v.LocalBounds().Dx())
			// +0.01 because the division is float: a box measured for exactly
			// n rows must not read as n-1.
			fits := int((v.LocalBounds().Dy()-v.fieldPad()*2)/v.lineH() + 0.01)
			if rows > fits {
				bad++
				if bad <= 4 {
					t.Errorf("%.2fx offered %.1f: a view for %d rows holds %d", scale, w, rows, fits)
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d widths hid a line", bad)
	}
}
