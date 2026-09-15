package style

import (
	"reflect"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var metroPackNames = []string{"win8", "win10", "win10-night"}

func TestMetroPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winCheckPacks(t, "metro", []winPackWant{
		{"win8", "Windows 8", 2012, ThemeLight},
		{"win10", "Windows 10", 2015, ThemeLight},
		{"win10-night", "Windows 10 Dark", 2018, ThemeDark},
	})
	// Metro is natively square: the "theme" Corners pref keeps it so.
	for _, n := range metroPackNames {
		if p, _ := LoadTheme(n); !p.Tokens.Metrics.Square {
			t.Fatalf("%s must be square", n)
		}
	}
}

func TestMetroColourTables(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for i, sc := range metroSchemes {
		for k := range sc {
			if _, ok := metroWin10[k]; !ok {
				t.Errorf("scheme %d: key %q is not in metroWin10", i, k)
			}
		}
	}
	for _, n := range metroPackNames {
		lk := winLook(t, n, 1)
		c := metroColors(lk)
		if c != metroColors(lk) {
			t.Fatalf("%s: colours are rebuilt per paint", n)
		}
		winNoSentinel(t, n, reflect.ValueOf(*c))
	}
	if metroColors(winLook(t, "win8", 1)).win10 || !metroColors(winLook(t, "win10-night", 1)).win10 {
		t.Fatal("scheme flags: win8 is Windows 8, win10-night is Windows 10")
	}
}

func TestMetroPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, metroPackNames)
}

func TestMetroPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsInsideRect(t, metroPackNames)
}

func TestMetroCloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metroPackNames {
		winCheckClose(t, n)
	}
}

func TestMetroStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metroPackNames {
		winCheckHints(t, n)
	}
}

func TestMetroLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metroPackNames {
		lk := winLook(t, n, 1)
		c := metroColors(lk)
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on face", c.text, c.face, 4.5},
			{"text on field", lk.fieldText(), c.field, 4.5},
			{"button label", c.text, c.btn[0][0], 4.5},
			{"hot button label", c.text, c.btn[1][0], 4.5},
			{"pressed button label", c.text, c.btn[2][0], 4.5},
			{"disabled label", c.disText, c.btn[3][0], 2},
			{"hot menu label", lk.Engine().MenuTextColor(lk, true), c.menuHot, 4.5},
			{"selected row", c.selFillText, c.selFill, 4.5},
			{"unfocused selection", c.offText, c.off, 4.5},
			{"hovered row", lk.fieldText(), c.hov, 4.5},
			{"tool tip", c.tipText, c.tip, 4.5},
			{"header", c.hdrText, c.hdr, 4.5},
			{"tab label", c.text, c.tab, 4.5},
			{"caption", c.capText, c.frame, 4.5},
			{"accent text", c.onAccent, c.accent, 3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Flat: square corners, no gradient — the corner pixel of a button is its
// border and the face is one colour top to bottom.
func TestMetroButtonsAreFlatAndSquare(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metroPackNames {
		lk := winLook(t, n, 1)
		c := metroColors(lk)
		img := paintengine2d.NewImage(100, 40)
		lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(5, 5, 90, 30), StateNone, "")
		if !winNear(img, 5, 5, c.btn[0][1]) {
			t.Errorf("%s: button corner is not its square border", n)
		}
		if !winNear(img, 12, 9, c.btn[0][0]) || !winNear(img, 12, 30, c.btn[0][0]) {
			t.Errorf("%s: button face is not flat", n)
		}
	}
}

// Windows 10's scroll bar is a thin indicator until the pointer is over it;
// then it shows its track and arrow buttons. Windows 8 always shows them.
func TestMetroThinScrollBarsShowArrowsOnHover(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metroPackNames {
		lk := winLook(t, n, 1)
		view := paintengine2d.XYWH(0, 0, 40, 200)
		p := ScrollGeometry(lk, view, true, 800, 200, 100, false)
		if p.Dec.Empty() || p.Inc.Empty() {
			t.Fatalf("%s: no arrow buttons %+v", n, p)
		}
		ink := func(st ScrollState) int {
			img := paintengine2d.NewImage(40, 200)
			DrawScrollBarParts(lk, paintengine2d.NewContext(img), p, true, st)
			k := 0
			for y := int(p.Dec.Min.Y); y < int(p.Dec.Max.Y); y++ {
				for x := int(p.Dec.Min.X); x < int(p.Dec.Max.X); x++ {
					if _, _, _, a := img.PremulAt(x, y); a != 0 {
						k++
					}
				}
			}
			return k
		}
		rest, hover := ink(ScrollState{}), ink(ScrollState{Hovered: true})
		thin := metroColors(lk).win10
		if thin && (rest != 0 || hover == 0) {
			t.Errorf("%s: arrow area ink at rest %d, on hover %d; want none, then the button", n, rest, hover)
		}
		if !thin && rest == 0 {
			t.Errorf("%s: Windows 8 arrows must show at rest", n)
		}
		if s := ScrollBarStyleOf(lk); thin != (s.Thickness < 16) {
			t.Errorf("%s: bar thickness %v", n, s.Thickness)
		}
	}
}

// Windows 10's slider thumb is a rectangle in the accent colour; its toggle
// switch is a pill.
func TestMetroWin10SliderAndSwitch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := winLook(t, "win10", 1)
	c := metroColors(lk)
	img := paintengine2d.NewImage(140, 40)
	lk.DrawSlider(paintengine2d.NewContext(img), paintengine2d.XYWH(10, 4, 120, 32), StateNone, 0.5)
	// The thumb (8×24 at the middle) is solid accent right up to its corner.
	if !winNear(img, 67, 9, c.accent) || !winNear(img, 72, 30, c.accent) {
		t.Fatalf("slider thumb is not the accent rectangle")
	}
	sw := paintengine2d.NewImage(80, 30)
	lk.DrawSwitch(paintengine2d.NewContext(sw), paintengine2d.XYWH(5, 5, 70, 20), StateNone, true, "")
	if _, _, _, a := sw.PremulAt(5, 5); a > 40 {
		t.Fatalf("toggle switch corner is painted (alpha %d): not a pill", a)
	}
	if !winNear(sw, 20, 15, c.accent) {
		t.Fatal("an on switch is the accent")
	}
}

// Windows 10 selects list rows with Explorer's light accent box; Windows 8
// with the solid highlight; both grey an unfocused view's selection.
func TestMetroListSelection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pick := func(lk *Classic, st ControlState) paintengine2d.Color {
		img := paintengine2d.NewImage(60, 24)
		lk.DrawListRow(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 60, 24), st, "")
		r, g, b, _ := img.PremulAt(30, 12)
		return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
	}
	w10, w8 := winLook(t, "win10", 1), winLook(t, "win8", 1)
	if got := pick(w10, StateChecked); !nearColor(got, Hex("#cce8ff")) {
		t.Errorf("win10 selection %s, want Explorer's #cce8ff", colorHexPadded(got))
	}
	if got := pick(w8, StateChecked); !nearColor(got, Hex("#3399ff")) {
		t.Errorf("win8 selection %s, want the #3399ff highlight", colorHexPadded(got))
	}
	for _, lk := range []*Classic{w10, w8} {
		if pick(lk, StateChecked|StateInactive) == pick(lk, StateChecked) {
			t.Errorf("%s: an unfocused view keeps its selection colour", lk.Pack())
		}
	}
}

func winNear(img *paintengine2d.Image, x, y int, c paintengine2d.Color) bool {
	r, g, b, _ := img.PremulAt(x, y)
	d := func(p uint8, q float32) int {
		v := int(p) - int(q*255+0.5)
		if v < 0 {
			v = -v
		}
		return v
	}
	return d(r, c.R)+d(g, c.G)+d(b, c.B) < 24
}
