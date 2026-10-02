package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// HeaderBar.StartWidth reserves room at the start of the bar so an
// application's own items line up with something below them — a sidebar the
// chrome runs along with.
//
// It used to hold only where the application's bar *is* the caption. Under a
// pack whose era stacks the frame (KDE 1, Windows 95, Metal: their own title
// strip with the application's row beneath it) the row was placed at the
// frame's inset and StartWidth was dropped, so a mail client's menu and
// buttons sat at the window's left edge, over the pane they were asked to
// begin after. The field's doc stated no exception.
func TestStartWidthHoldsUnderAStackedFrameToo(t *testing.T) {
	// One of each: a stacked era and a merged one.
	for _, pack := range []string{"kde1", "win95", "breeze-night", "adwaita"} {
		t.Run(pack, func(t *testing.T) {
			r := framedRig(t, pack, 1280, 800)
			const start = 240

			lead := widgets.NewButton("Fetch", nil)
			hb := widgets.NewHeaderBar([]widget.Component{lead}, nil, nil)
			hb.ShowTitle = false
			hb.StartWidth = start
			r.w.SetTitleBar(hb)
			r.a.PumpOnce()

			if r.w.Caption() == nil {
				t.Skip("no toolkit-drawn caption here")
			}
			got := widget.DeviceOrigin(lead).X + lead.Bounds().Min.X
			// The item begins at the reserved width, not before it. An era
			// that puts window controls past it may push it further along,
			// which is the documented floor rather than a failure.
			if got < start {
				stacked := style.DecorationOf(r.w.Look(), hb.DecorationState()).Stacked
				t.Errorf("the bar's first item starts at %g, before the %d asked for (stacked=%v)",
					got, start, stacked)
			}
		})
	}
}
