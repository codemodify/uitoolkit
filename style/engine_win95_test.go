package style

import "testing"

func TestWin95PacksRegisteredInYearOrder(t *testing.T) {
	want := []string{"win-hotdog", "win95", "win-highcontrast", "win95-dark", "win98", "win2000"}
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	for _, n := range want {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		if p.Tokens.Engine != "win95" {
			t.Fatalf("%s engine %q", n, p.Tokens.Engine)
		}
		if lk := p.Look(); lk.Engine().ID() != "win95" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	if pos["win-hotdog"] > pos["win95"] || pos["win95"] > pos["win98"] || pos["win98"] > pos["win2000"] {
		t.Fatalf("packs not ordered by year: %v", pos)
	}
}

// Engine metrics set the default-density geometry; density still shifts it.
func TestWin95DensityShiftsEngineMetrics(t *testing.T) {
	p, _ := LoadTheme("win95")
	def := packMetrics(DensityDefault, p.Tokens, CornersTheme, IconSizeMedium)
	cmp := packMetrics(DensityCompact, p.Tokens, CornersTheme, IconSizeMedium)
	if def.RowH != 22 {
		t.Fatalf("win95 default row %v, want the engine's 22", def.RowH)
	}
	if cmp.RowH >= def.RowH {
		t.Fatalf("compact row %v not shorter than default %v", cmp.RowH, def.RowH)
	}
	if def.Radius != 0 || def.RadiusSmall != 0 {
		t.Fatalf("win95 is natively square, got radii %v/%v", def.Radius, def.RadiusSmall)
	}
	round := packMetrics(DensityDefault, p.Tokens, CornersRound, IconSizeMedium)
	if round.Radius == 0 {
		t.Fatal("an explicit Round override must still round the corners")
	}
}
