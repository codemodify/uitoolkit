package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var adw48PackNames = []string{"adwaita48", "adwaita48-night"}

func TestAdwaita48PacksRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	for n, w := range map[string]struct {
		label string
		fam   ThemeName
	}{"adwaita48": {"Adwaita (GNOME 48)", ThemeLight}, "adwaita48-night": {"Adwaita Dark (GNOME 48)", ThemeDark}} {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		if p.Tokens.Engine != "adwaita" || p.Label != w.label || p.Lineage != "GNOME" || p.Year != 2025 || p.Tokens.Family != w.fam {
			t.Fatalf("%s: engine %q label %q lineage %q year %d family %q", n, p.Tokens.Engine, p.Label, p.Lineage, p.Year, p.Tokens.Family)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if _, ok := p.Look().Engine().(adwaita48Engine); !ok {
			t.Fatalf("%s paints with %T, want GNOME 48's era", n, p.Look().Engine())
		}
		if f := withEraFonts(p.Tokens, n).Fonts; f.UI[0] != "Adwaita Sans" || f.Mono[0] != "Adwaita Mono" {
			t.Fatalf("%s reads in %v / %v, want Adwaita Sans and Adwaita Mono", n, f.UI, f.Mono)
		}
	}
	if pos["adwaita"] > pos["adwaita48"] {
		t.Fatal("GNOME 42's Adwaita sorts after GNOME 48's")
	}
	// The atlas keeps GNOME 42–47's Adwaita as it was.
	for _, n := range adwPackNames {
		if _, ok := mustLook(t, n).Engine().(adwaitaEngine); !ok {
			t.Fatalf("%s left GNOME 47", n)
		}
	}
}

func TestAdwaita48PaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, adw48PackNames)
	winPaintsInsideRect(t, adw48PackNames)
	for _, n := range adw48PackNames {
		for _, scale := range []float32{1, 1.75, 2} {
			lk := clLook(t, n, scale)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}

func TestAdwaita48CloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range adw48PackNames {
		winCheckClose(t, n)
	}
}

// GNOME 48 rounds buttons 9px where GNOME 47 rounded 6, and popovers 15px
// where it rounded 12; cards keep 12.
func TestAdwaita48Radii(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	old, now := clLook(t, "adwaita", 1), clLook(t, "adwaita48", 1)
	b := paintengine2d.XYWH(0, 0, 80, 34)
	for _, c := range []struct {
		lk   *Classic
		ctl  float32
		pop  float32
		card float32
	}{{old, 6, 12, 12}, {now, 9, 15, 12}} {
		if got := adwR(c.lk, 6, b); got != c.ctl {
			t.Errorf("%s: control radius %v, want %v", c.lk.Name(), got, c.ctl)
		}
		if got := adwR(c.lk, 12, paintengine2d.XYWH(0, 0, 200, 200)); got != c.pop {
			t.Errorf("%s: popover radius %v, want %v", c.lk.Name(), got, c.pop)
		}
		if got := adwCardR(c.lk, paintengine2d.XYWH(0, 0, 200, 200)); got != c.card {
			t.Errorf("%s: card radius %v, want %v", c.lk.Name(), got, c.card)
		}
	}
	// A button's corner shows it: the pixel two in from the corner is the
	// window under GNOME 48's 9px corner and the wash under GNOME 47's 6px.
	paint := func(lk *Classic) *paintengine2d.Image {
		img := paintengine2d.NewImage(100, 50)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 100, 50), paintengine2d.Fill(adwColors(lk).win))
		lk.DrawButton(ctx, paintengine2d.XYWH(10, 8, 80, 34), StateNone, "")
		return img
	}
	if winNear(paint(old), 11, 10, adwColors(old).win) || !winNear(paint(now), 11, 10, adwColors(now).win) {
		t.Errorf("the button corners do not go from 6 to 9px")
	}
}

// The 1.10 popover has no border line: its edge is the popover colour, and
// the ring is its shadow's.
func TestAdwaita48PopoverRing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := clLook(t, "adwaita48", 1)
	c := adwColors(lk)
	b := paintengine2d.XYWH(20, 20, 160, 100)
	img := paintengine2d.NewImage(200, 140)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, 200, 140), paintengine2d.Fill(c.win))
	lk.DrawPopupShadow(ctx, b, PopupMenu)
	lk.DrawMenuFrame(ctx, b)
	if !winNear(img, 100, 20, c.popover) {
		t.Errorf("the popover's top edge is %v, want its fill", pixelColor(img, 100, 20))
	}
	if winNear(img, 100, 19, c.win) {
		t.Errorf("no ring just outside the popover")
	}
	if in := lk.PopupShadow(PopupMenu); in.Top < 1 || in.Bottom <= in.Top {
		t.Errorf("popover shadow reach %+v", in)
	}
}

// Follow the desktop moves between the GNOME 48 twins.
func TestAdwaita48SchemeSiblings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := SchemeVariant("adwaita48", SchemeDark); got != "adwaita48-night" {
		t.Errorf("adwaita48 in the dark: %q", got)
	}
	if got := SchemeVariant("adwaita48-night", SchemeLight); got != "adwaita48" {
		t.Errorf("adwaita48-night in the light: %q", got)
	}
}
