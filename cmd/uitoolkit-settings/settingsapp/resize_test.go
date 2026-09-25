package settingsapp

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// settingsSplit is the page's splitter: the column of choices left, the
// two previews right.
func settingsSplit(t *testing.T, w *app.Window) *widgets.Splitter {
	t.Helper()
	var split *widgets.Splitter
	widget.Walk(w.Content(), func(c widget.Component) {
		if s, ok := c.(*widgets.Splitter); ok && s.AccessibleName() == "Settings and preview" {
			split = s
		}
	})
	if split == nil {
		t.Fatal("no Settings and preview splitter")
	}
	return split
}

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-4 }

// The column of choices keeps its share of a window resized live, not
// only on the next rebuild — and once the user has dragged the sash, the
// split is theirs through every resize after.
func TestSettingsChoicesFollowALiveResize(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, w := openSettings(t, 1024, 860)
	split := settingsSplit(t, w)
	if want := defaultChoicesRatio(w); !near(split.Ratio, want) {
		t.Fatalf("at 1024: ratio %v, want %v", split.Ratio, want)
	}
	for _, size := range [][2]int{{1400, 900}, {760, 600}, {1024, 860}} {
		w.Inject(platform.Event{Kind: platform.EventResize, Width: size[0], Height: size[1]})
		a.PumpOnce()
		if lw, lh := w.Size(); lw != size[0] || lh != size[1] {
			t.Fatalf("window is %dx%d, want %v", lw, lh, size)
		}
		if settingsSplit(t, w) != split {
			t.Fatal("a resize rebuilt the page")
		}
		if want := defaultChoicesRatio(w); !near(split.Ratio, want) {
			t.Errorf("at %v: ratio %v, want %v", size, split.Ratio, want)
		}
		// About 300 logical pixels across, within the ratio's limits: an
		// option row is a label, a control and a line of prose.
		if got := split.PaneA().Dx(); got < 260 || got > 340 {
			t.Errorf("at %v: the browser is %v px wide", size, got)
		}
	}

	// Drag the sash to the right: the resizes after that leave it be.
	o := widget.DeviceOrigin(split)
	pa, pb := split.PaneA(), split.PaneB()
	sash := paintengine2d.Pt(o.X+(pa.Max.X+pb.Min.X)*0.5, o.Y+200)
	dest := paintengine2d.Pt(sash.X+120, sash.Y)
	w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: sash, Button: platform.ButtonLeft})
	a.PumpOnce()
	w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: dest, Button: platform.ButtonLeft})
	a.PumpOnce()
	w.Inject(platform.Event{Kind: platform.EventMouseUp, Pos: dest, Button: platform.ButtonLeft})
	a.PumpOnce()
	dragged := split.Ratio
	if near(dragged, defaultChoicesRatio(w)) {
		t.Fatalf("the drag did not move the sash (ratio %v)", dragged)
	}
	for _, size := range [][2]int{{1300, 880}, {800, 640}} {
		w.Inject(platform.Event{Kind: platform.EventResize, Width: size[0], Height: size[1]})
		a.PumpOnce()
		if !near(split.Ratio, dragged) {
			t.Errorf("at %v: the dragged ratio %v became %v", size, dragged, split.Ratio)
		}
	}
	// And a rebuild — Apply is what builds the page again — keeps the
	// user's split too.
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if got := settingsSplit(t, w).Ratio; !near(got, dragged) {
		t.Errorf("after a rebuild the dragged ratio %v became %v", dragged, got)
	}
}
