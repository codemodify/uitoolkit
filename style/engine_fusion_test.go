//go:build theme_engine_all || theme_engine_fusion

package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var fusionPackNames = []string{"fusion", "fusion-night"}

func TestFusionPacksRegisteredInYearOrder(t *testing.T) {
	kdeCheckPacks(t, "fusion", "Qt", 2012, map[string]string{"fusion": "Fusion", "fusion-night": "Fusion Dark"})
	// The dark pack replaces the legacy base-engine "fusion-night"; there
	// is no second dark Fusion under another name.
	if _, ok := kdePositions()["fusion-dark"]; ok {
		t.Fatal("no pack may be named fusion-dark: the dark pack is fusion-night")
	}
	if p, _ := LoadTheme("fusion-night"); !strings.Contains(p.Summary, "Not historical") {
		t.Fatalf("fusion-night must say Qt's 2012 Fusion had no dark palette: %q", p.Summary)
	}
	pos := kdePositions()
	if !(pos["oxygen"] < pos["fusion"] && pos["fusion"] < pos["breeze"]) {
		t.Fatalf("Oxygen (2008), Fusion (2012) and Breeze (2014) out of year order: %v %v %v", pos["oxygen"], pos["fusion"], pos["breeze"])
	}
}

func TestFusionStyleHintsAndScrollBars(t *testing.T) {
	for _, n := range fusionPackNames {
		lk := mustLook(t, n)
		// Qt on Linux: KDE button-box layout, accept before reject.
		if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != 0 {
			t.Fatalf("%s: want OK before Cancel, left-aligned tabs and left-aligned form labels", n)
		}
		s := ScrollBarStyleOf(lk)
		if s.Arrows != ArrowsEnds || s.Overlay || s.Thickness != 14 {
			t.Fatalf("%s scroll bar %+v: want a 14px bar with an arrow button at each end", n, s)
		}
		if o := lk.TabOutset(); o.Right <= 0 {
			t.Fatalf("%s: the selected tab overlaps its neighbour, got outset %+v", n, o)
		}
	}
}

// Fusion's shades follow Qt's documented HSV-value arithmetic.
func TestFusionColourArithmetic(t *testing.T) {
	for _, c := range []struct {
		got  paintengine2d.Color
		want string
	}{
		{darkerPct(Hex("#efefef"), 140), "#ababab"}, // the outline of the light palette
		{lighterPct(Hex("#efefef"), 150), "#ffffff"},
		{lighterPct(Hex("#308cc6"), 100), "#308cc6"},
		{withLightness(Hex("#ff0000"), 0.25), "#800000"},
	} {
		if colorHexPadded(c.got) != c.want {
			t.Fatalf("got %s want %s", colorHexPadded(c.got), c.want)
		}
	}
	for _, s := range []string{"#308cc6", "#2a82da", "#353535", "#ffffdc", "#7f7f7f"} {
		c := Hex(s)
		h, sat, v := hsvOf(c)
		if got := colorHexPadded(hsvColor(h, sat, v, 1)); got != s {
			t.Fatalf("HSV round trip of %s gave %s", s, got)
		}
	}
}

func TestFusionCloseButtonAgreesWithPaint(t *testing.T) {
	for _, n := range fusionPackNames {
		kdeCheckClose(t, n)
	}
}

func TestFusionPaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range fusionPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			exerciseEngine(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}

// ---- helpers shared by the Fusion, Oxygen and Breeze tests -----------------------------

// flagged below

// kdeCheckPacks checks that every pack in labels is registered once, paints
// with engine, and carries its label, year and lineage.

// kdeCheckClose checks that the close button sits at the right of the
// title bar and that WindowCloseRect covers exactly what the button paints:
// a frame with the button and one without differ only inside the rect.

// kdeExtras paints what exerciseEngine does not reach — the current item's
// focus mark, item views' frames, rows in every item state, the tab pane
// and the window background — and fails when one panics, leaves saved
// states or paints outside its rect.
