package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestKeramikPackRegisteredInYearOrder(t *testing.T) {
	kdeCheckPacks(t, "keramik", "KDE", 2002, map[string]string{"keramik": "Keramik"})
	pos := kdePositions()
	// Keramik (2002) comes before KDE's later looks: Plastik (2004) and
	// Oxygen (2008).
	if !(pos["keramik"] < pos["oxygen"]) {
		t.Fatalf("Keramik (2002) sorts after Oxygen (2008): %v %v", pos["keramik"], pos["oxygen"])
	}
	if p, ok := pos["plastik"]; ok && pos["keramik"] > p {
		t.Fatalf("Keramik (2002) sorts after Plastik (2004): %v %v", pos["keramik"], p)
	}
}

// The pack carries KDE 3.1's default colour scheme.
func TestKeramikCarriesTheKDE31Scheme(t *testing.T) {
	lk := mustLook(t, "keramik")
	p := lk.Palette()
	for _, c := range []struct {
		name string
		got  paintengine2d.Color
		want string
	}{
		{"background", p.Background, "#eae9e8"},
		{"button", p.SurfaceAlt, "#e6f0f9"},
		{"selection", p.Selection, "#a9d1ff"},
		{"selected text", p.TextOnAccent, "#030303"},
		{"base", p.Field, "#ffffff"},
		{"caption", lk.X("caption", paintengine2d.Color{}), "#3e91eb"},
		{"alternate", lk.X("alternate", paintengine2d.Color{}), "#eef6ff"},
	} {
		if got := colorHexPadded(c.got); got != c.want {
			t.Errorf("%s = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestKeramikStyleHintsAndScrollBars(t *testing.T) {
	lk := mustLook(t, "keramik")
	// KDE: OK before Cancel; Qt 3 left-aligned its form labels.
	if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != 0 {
		t.Fatal("Keramik: want OK before Cancel, left-aligned tabs and left-aligned form labels")
	}
	s := ScrollBarStyleOf(lk)
	if s.Arrows != ArrowsTripleEnd || s.Overlay || s.Thickness != 17 {
		t.Fatalf("Keramik scroll bar %+v: want a 17px bar with KDE 3's three step buttons", s)
	}
	// A back button at the top, back and forward together at the bottom.
	parts := ScrollGeometry(lk, paintengine2d.XYWH(0, 0, 200, 300), true, 900, 300, 0, false)
	if parts.Dec.Min.Y != 0 || parts.Track.Min.Y != parts.Dec.Max.Y || parts.DecEnd.Min.Y < 200 || parts.Inc.Min.Y <= parts.DecEnd.Min.Y {
		t.Fatalf("Keramik vertical bar parts %+v: want back at the top, back and forward together at the bottom", parts)
	}
	// KDE 3.1 menus cast no shadow.
	for _, k := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
		if !lk.PopupShadow(k).Zero() {
			t.Fatalf("Keramik popup kind %d has a shadow %+v", k, lk.PopupShadow(k))
		}
	}
}

// The two gel ramps land on KDE 3.1 as measured: the push button over
// #e6f0f9 (brightest #fbffff a sixth of the way down, a flat #d5dee6 lower
// half) and the menu highlight over #a9d1ff (#d6f0ff at the top, darkest
// #a6c3e4 at two thirds, lifting to #b6d2f2 at the bottom).
func TestKeramikGelsMatchKDE31(t *testing.T) {
	at := func(stops []paintengine2d.GradientStop, off float32) paintengine2d.Color {
		for i := 1; i < len(stops); i++ {
			if off <= stops[i].Offset {
				a, b := stops[i-1], stops[i]
				return Mix(a.Color, b.Color, (off-a.Offset)/(b.Offset-a.Offset))
			}
		}
		return stops[len(stops)-1].Color
	}
	btn := kmButtonStops(Hex("#e6f0f9"))
	sel := kmGelStops(Hex("#a9d1ff"))
	for _, c := range []struct {
		name  string
		stops []paintengine2d.GradientStop
		off   float32
		want  string
	}{
		{"button top band", btn, 0.14, "#fbffff"},
		{"button middle", btn, 0.52, "#dee7f0"},
		{"button lower half", btn, 0.8, "#d5dee6"},
		{"menu top", sel, 0.02, "#d6f0ff"},
		{"menu darkest", sel, 0.68, "#a6c3e4"},
		{"menu bottom", sel, 1, "#b6d2f2"},
	} {
		if got := at(c.stops, c.off); !kdeClose(got, Hex(c.want), 12) {
			t.Errorf("%s: %s, want about %s", c.name, colorHexPadded(got), c.want)
		}
	}
}

// The default button sits in a sunken ring: dark above it, light below.
func TestKeramikDefaultButtonRing(t *testing.T) {
	lk := mustLook(t, "keramik")
	img := paintengine2d.NewImage(120, 44)
	ctx := paintengine2d.NewContext(img)
	lk.DrawButton(ctx, paintengine2d.XYWH(4, 4, 110, 32), StatePrimary, "OK")
	lum := func(x, y int) int {
		r, g, b, _ := img.PremulAt(x, y)
		return int(r) + int(g) + int(b)
	}
	if top, bottom := lum(60, 4), lum(60, 35); top >= bottom {
		t.Fatalf("default ring: top %d not darker than bottom %d", top, bottom)
	}
}

func TestKeramikCloseButtonAgreesWithPaint(t *testing.T) {
	kdeCheckClose(t, "keramik")
}

func TestKeramikPaintsEveryControlInsideItsRect(t *testing.T) {
	p, _ := LoadTheme("keramik")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		aquaExercise(t, fmt.Sprintf("keramik@%gx", scale), lk)
		kdeExtras(t, fmt.Sprintf("keramik@%gx", scale), lk)
	}
}
