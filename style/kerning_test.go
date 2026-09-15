package style

import "testing"

// Pairs like "AV" and "To" sit closer than their glyphs' advances; carets,
// hit-testing and eliding follow the kerned layout that Draw paints.
func TestTextIsKerned(t *testing.T) {
	f := BakeFont(16, Hex("#000000"))
	for _, pair := range []string{"AV", "To", "Ty"} {
		a, b := string(pair[0]), string(pair[1])
		if got, plain := f.Advance(pair), f.Advance(a)+f.Advance(b); got >= plain-0.2 {
			t.Errorf("%q is %v wide, its glyphs %v: not kerned", pair, got, plain)
		}
	}
	text := "AVATAR Typography"
	run := f.shapeOf(text)
	if len(run.xs) != len([]rune(text))+1 {
		t.Fatalf("%d positions for %d runes", len(run.xs), len([]rune(text)))
	}
	for i := 0; i <= len([]rune(text)); i++ {
		if got, want := f.CaretX(text, i), run.xs[i]; got != want {
			t.Fatalf("caret %d at %v, the layout puts it at %v", i, got, want)
		}
	}
	// Clicking just right of a caret stop lands on it.
	for i := 1; i < len(run.xs)-1; i++ {
		if got := f.IndexAt(text, run.xs[i]+0.2); got != i {
			t.Fatalf("a click at stop %d (x=%v) hit %d", i, run.xs[i], got)
		}
	}
	if fit := f.Fit(text, f.Advance("AVATAR")+f.Advance("…")+0.5); fit != "AVATAR…" {
		t.Fatalf("Fit gave %q", fit)
	}
}
