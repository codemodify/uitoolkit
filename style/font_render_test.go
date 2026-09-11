package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/fonts"
)

func TestDefaultFontsRender(t *testing.T) {
	if err := LoadEmbeddedFonts(); err != nil {
		t.Fatalf("LoadEmbeddedFonts: %v", err)
	}
	for _, name := range []string{
		fonts.FileTitilliumRegular, fonts.FileTitilliumBold,
		fonts.FileJetBrainsRegular, fonts.FileJetBrainsBold,
	} {
		b, err := fonts.Bytes(name)
		if err != nil {
			t.Fatalf("missing embedded %s: %v", name, err)
		}
		if len(b) < 1024 {
			t.Fatalf("%s too small to be a TTF (%d bytes)", name, len(b))
		}
	}

	ui := BakeFont(22, paintengine2d.White)
	mono := BakeMonoFont(22, paintengine2d.White)
	bold := BakeTitleFont(22, paintengine2d.White)

	if !ui.Outline || !mono.Outline || !bold.Outline {
		t.Fatal("default faces must be OpenType outlines, not the 5×7 bitmap atlas")
	}
	if ui.Family != FamilyUI {
		t.Fatalf("UI family %q want %q", ui.Family, FamilyUI)
	}
	if mono.Family != FamilyMono {
		t.Fatalf("mono family %q want %q", mono.Family, FamilyMono)
	}
	if bold.Weight != WeightBold {
		t.Fatalf("title weight %d", bold.Weight)
	}

	// Titillium is proportional: i is much narrower than W.
	if ui.Advance("i") >= ui.Advance("W")*0.55 {
		t.Fatalf("Titillium should be proportional: i=%v W=%v", ui.Advance("i"), ui.Advance("W"))
	}
	// JetBrains Mono is monospaced.
	if d := mono.Advance("i") - mono.Advance("W"); d > 1.5 || d < -1.5 {
		t.Fatalf("JetBrains Mono should be monospaced: i=%v W=%v", mono.Advance("i"), mono.Advance("W"))
	}

	type sample struct {
		name string
		f    *Font
		text string
	}
	for _, s := range []sample{
		{"titillium", ui, "Ag Quality — Titillium Web"},
		{"jetbrains", mono, "fn(x) => x  JetBrains Mono"},
		{"titillium-bold", bold, "Projects"},
	} {
		ink, fringe := rasterInk(s.f, s.text)
		if ink < 80 {
			t.Fatalf("%s: expected outline ink, got %d opaque pixels", s.name, ink)
		}
		if fringe < 12 {
			t.Fatalf("%s: expected anti-aliased fringe, got %d mid-alpha pixels (bitmap 5×7 is 1-bit)", s.name, fringe)
		}
	}

	look := DarkLook()
	if look.Font().Family != FamilyUI || !look.Font().Outline {
		t.Fatalf("Classic body is %q outline=%v", look.Font().Family, look.Font().Outline)
	}
	if look.TitleFont().Weight != WeightBold {
		t.Fatal("Classic title must be Titillium Bold")
	}
	if look.MonoFont().Family != FamilyMono || !look.MonoFont().Outline {
		t.Fatalf("Classic mono is %q outline=%v", look.MonoFont().Family, look.MonoFont().Outline)
	}
}

func TestLookAndFeelRoles(t *testing.T) {
	if FamilyFor(RoleUI) != "Titillium Web" || FamilyFor(RoleMono) != "JetBrains Mono" {
		t.Fatalf("locked families %q / %q", FamilyFor(RoleUI), FamilyFor(RoleMono))
	}
	m := DefaultMetrics()
	m.FontFamily = "mononoki"
	m.MonoFamily = "mononoki"
	look := NewClassic("dark", Dark(), m)
	if look.Font().Family != FamilyUI {
		t.Fatalf("UI role leaked %q (mononoki is not default)", look.Font().Family)
	}
	if look.MonoFont().Family != FamilyMono {
		t.Fatalf("mono role leaked %q (mononoki is not default)", look.MonoFont().Family)
	}
	if look.Metrics().FontFamily != FamilyUI || look.Metrics().MonoFamily != FamilyMono {
		t.Fatalf("metrics families %q / %q", look.Metrics().FontFamily, look.Metrics().MonoFamily)
	}
	if FontFor(look, RoleUI).Family != FamilyUI || FontFor(look, RoleMono).Family != FamilyMono {
		t.Fatal("FontFor roles")
	}
	bold := BakeMonoBoldFont(16, paintengine2d.White)
	if bold.Family != FamilyMono || bold.Weight != WeightBold || !bold.Outline {
		t.Fatalf("mono bold %q w=%d outline=%v", bold.Family, bold.Weight, bold.Outline)
	}
}

func TestBitmapFallbackIsNotDefault(t *testing.T) {
	ui := BakeFont(16, paintengine2d.White)
	bit := BakeBitmapFont(2, paintengine2d.White)
	if !ui.Outline {
		t.Fatal("BakeFont must stay on the OpenType path")
	}
	if bit.Outline {
		t.Fatal("BakeBitmapFont should be the 5×7 atlas")
	}
	_, uiFringe := rasterInk(ui, "Ag")
	_, bitFringe := rasterInk(bit, "Ag")
	if uiFringe <= bitFringe {
		t.Fatalf("outline fringe %d should exceed chunky bitmap fringe %d", uiFringe, bitFringe)
	}
}

func rasterInk(f *Font, text string) (ink, fringe int) {
	img := paintengine2d.NewImage(420, 48)
	ctx := paintengine2d.NewContext(img)
	f.Draw(ctx, text, paintengine2d.Pt(8, 10), paintengine2d.White)
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			_, _, _, a := img.PremulAt(x, y)
			switch {
			case a > 200:
				ink++
			case a > 24 && a < 200:
				fringe++
			}
		}
	}
	return ink, fringe
}
