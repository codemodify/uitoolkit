package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// stackedLook is a pack whose frame keeps its own title strip above the
// application's bar — the classic desktops. Every build has one; which
// one it is does not matter to this test.
func stackedLook(t *testing.T) style.LookAndFeel {
	t.Helper()
	for _, p := range style.ListThemes() {
		lk := style.Appearance{Name: p.Name, Theme: p.Palette}.Look()
		if style.DecorationOf(lk, style.DecorationState{Active: true}).Stacked {
			return lk
		}
	}
	t.Skip("no stacked pack in this build")
	return nil
}

// An application whose title bar *is* its caption says so, and a classic
// look stops putting its own strip above it.
//
// Chromium's tabs are its title bar on every desktop it runs on; it does
// not grow a second caption because the system theme is an old one. The
// look-like-chromium sample did exactly that: it asked for client
// decorations, set the tab strip as the title bar and still came out with
// a title row above the tabs, because the pack's era said stacked and
// nothing could say otherwise.
func TestCaptionMergedBeatsAStackedLook(t *testing.T) {
	lk := stackedLook(t)
	const w, h = 900, 200

	build := func(s style.CaptionStyle) *HeaderBar {
		hb := NewHeaderBar(nil, NewLabel("TABS"), nil)
		hb.ShowTitle = false
		hb.SetHost(&fakeWindow{look: lk})
		hb.SetWindowControls(platform.ButtonLayout{
			Right: []platform.CaptionButton{platform.CaptionMinimize, platform.CaptionMaximize, platform.CaptionClose},
		}, true)
		hb.SetCaptionStyle(s)
		return hb
	}

	era := build(style.CaptionFollowsLook)
	if !era.Stacked() {
		t.Fatal("the era's own frame is not stacked — wrong pack picked")
	}
	era.Arrange(paintengine2d.XYWH(0, 0, w, era.Measure(layout.Constraints{MaxW: w, MaxH: h}).Y))
	if _, bar := era.FrameParts(); bar.Dy() <= 0 {
		t.Error("a stacked frame gave the app no row under its strip")
	}

	merged := build(style.CaptionMerged)
	if merged.Stacked() {
		t.Fatal("CaptionMerged still reports a stacked frame")
	}
	mh := merged.Measure(layout.Constraints{MaxW: w, MaxH: h}).Y
	merged.Arrange(paintengine2d.XYWH(0, 0, w, mh))
	caption, bar := merged.FrameParts()
	if bar.Dy() > 0 {
		t.Errorf("a merged caption still has a row under a strip: %v", bar)
	}
	if caption.Dx() < w-0.5 {
		t.Errorf("a merged caption is %v wide, want the window's %v", caption.Dx(), w)
	}

	// One row, not two: the whole point is that the strip is gone.
	eh := era.Measure(layout.Constraints{MaxW: w, MaxH: h}).Y
	if mh >= eh {
		t.Errorf("merged is %v tall and stacked is %v — merging saved no row", mh, eh)
	}

	// The window buttons are in that one row, which is what makes it the
	// caption rather than a toolbar.
	if !merged.TrailControls().Visible() {
		t.Error("a merged caption has no window buttons")
	}
	if tb := merged.TrailControls().Bounds(); tb.Max.Y > mh+0.5 || tb.Dy() <= 0 {
		t.Errorf("the window buttons are not inside the one row: %v in %v", tb, mh)
	}

	// And the other way: a window can demand the classic strip too.
	if got := build(style.CaptionStacked); !got.Stacked() {
		t.Error("CaptionStacked did not keep the strip")
	}
}

// A pack that says its buttons are centred gets them centred, whether the
// band is taller than its caption or its button is shorter than the band.
//
// The flag used to answer only the first of those, so every pack that
// wanted centring worked the second out by hand into ButtonPad.Top and a
// pack that set the flag alone got nothing — its buttons hung from the
// top with the slack beneath them. That is what a browser sample's
// caption buttons sitting high in their band turned out to be.
func TestCenteredCaptionButtonsAreCentredInTheBand(t *testing.T) {
	for _, p := range style.ListThemes() {
		lk := style.Appearance{Name: p.Name, Theme: p.Palette}.Look()
		spec := style.DecorationOf(lk, style.DecorationState{Active: true})
		bh := max(spec.Button.Y, spec.CloseButton.Y)
		if !spec.CenterButtons || bh <= 0 || spec.Caption <= 0 {
			continue // full-height buttons and era looks that hang from the top
		}
		hb := NewHeaderBar(nil, NewLabel("x"), nil)
		hb.SetHost(&fakeWindow{look: lk})
		hb.SetWindowControls(platform.ButtonLayout{
			Right: []platform.CaptionButton{platform.CaptionClose},
		}, true)
		band := spec.Caption
		hb.Arrange(paintengine2d.XYWH(0, 0, 400, band))
		if !hb.TrailControls().Visible() {
			continue
		}
		btn := hb.TrailControls().Bounds()
		if btn.Dy() <= 0 {
			continue
		}
		// The gap above the button and the gap below it agree.
		above := btn.Min.Y
		below := band - btn.Max.Y
		if d := above - below; d > 1.5 || d < -1.5 {
			t.Errorf("%s: button sits %v from the top and %v from the bottom of a %v band",
				p.Name, above, below, band)
		}
	}
}
