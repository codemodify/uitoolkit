package skingen_test

// The tests in this file are the outside author's point of view: they use
// nothing the exported API does not offer, and they import style the way an
// outside package would. If one of them stops compiling, the promise the
// package doc makes — that a skin somebody else writes can be made the way
// the shipped ones are — has been broken.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/skingen"
	"github.com/codemodify/uitoolkit/style"
)

// smallestPlan is the worked example in the package doc and in docs/skins.md,
// kept here so the three cannot drift apart: the least a plan can say and
// still produce a skin the loader accepts.
func smallestPlan() *skingen.Plan {
	return &skingen.Plan{
		Name: "mine", Label: "Mine", Base: "breeze-night",
		Sheets: []*skingen.Sheet{{Name: "chrome", W: 96, H: 32, Cells: []skingen.Cell{{
			Name: "button.normal", X: 0, Y: 0, W: 32, H: 32, Slice: [4]int{6, 6, 6, 6},
			Draw: func(ctx *paintengine2d.Context, w, h float32) {
				ctx.DrawRoundRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), 6, 6,
					paintengine2d.Fill(skingen.Hex("#40444c")))
			},
		}}}},
		Parts: []skingen.PartBinding{{Part: "button", States: [][2]string{{"normal", "button.normal"}}}},
	}
}

// The smallest plan there is writes a pack the loader reads: two sheets at
// the two required scales, a manifest beside them, and a button that is the
// author's. Everything else is the base pack's, which is what a partial skin
// means.
func TestTheSmallestPlanWritesASkinThatLoads(t *testing.T) {
	dir := t.TempDir()
	p := smallestPlan()
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, f := range []string{
		skingen.SkinManifestName,
		filepath.Join("art", "chrome.png"),
		filepath.Join("art", "chrome@2x.png"),
	} {
		if _, err := os.Stat(filepath.Join(dir, p.Name, f)); err != nil {
			t.Errorf("Write did not leave %s: %v", f, err)
		}
	}

	sk, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name)
	if err != nil {
		t.Fatalf("the smallest plan does not load: %v", err)
	}
	if sk.Label != "Mine" || sk.Base != "breeze-night" {
		t.Errorf("loaded %q on %q, want Mine on breeze-night", sk.Label, sk.Base)
	}
	if sk.Parts["button"] == nil {
		t.Error("the button the plan binds did not reach the skin")
	}

	// It lints. A one-part skin is terse, not wrong: the lint has things to
	// say about a button with no pressed face, and nothing to say about the
	// pack itself. What must not appear is a complaint about the art or the
	// manifest, because the generator wrote both.
	for _, w := range style.LintSkin(sk) {
		if w.Key == "sheets.chrome.2x" || w.Key == "sprites" {
			t.Errorf("the generator's own output warns: %s", w)
		}
	}
}

// Render is the half of the generator an author may want on its own — a
// preview, a contact sheet, a plan drawn at a scale nobody ships. It draws
// the sheet the plan asked for, at the size the scale makes it.
func TestRenderDrawsTheSheetAtAnyScale(t *testing.T) {
	sh := smallestPlan().Sheets[0]
	for _, c := range []struct {
		scale float32
		w, h  int
	}{{1, 96, 32}, {2, 192, 64}, {1.5, 144, 48}} {
		img := skingen.Render(sh, c.scale)
		if img.Width != c.w || img.Height != c.h {
			t.Errorf("at %gx the sheet is %d×%d, want %d×%d", c.scale, img.Width, img.Height, c.w, c.h)
		}
	}
	// And it drew something: an empty sheet is the failure mode that looks
	// like success until the app is running.
	img := skingen.Render(sh, 1)
	ink := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 0 {
				ink++
			}
		}
	}
	if ink == 0 {
		t.Error("Render drew nothing at all")
	}
}

// Manifest is the other half, for an author who renders the art some other
// way, or who wants to read what a plan would say before writing anything.
func TestManifestIsTheSkinTheLoaderWouldRead(t *testing.T) {
	doc, err := skingen.Manifest(smallestPlan())
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if len(doc) == 0 || doc[len(doc)-1] != '\n' {
		t.Error("the manifest should be a text file, newline and all")
	}
	// Twice is the same bytes: the manifest is ordered, so a pack diffs one
	// line per change instead of wholesale.
	again, err := skingen.Manifest(smallestPlan())
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if string(doc) != string(again) {
		t.Error("two manifests of the same plan differ")
	}
}

// The published plans are the reason the package is public: an author forks
// one rather than opening a blank sheet. So each call has to hand back a
// plan of its own — a fork that renamed Nocturne and then found the shipped
// Nocturne renamed too would be a trap.
func TestAShippedPlanCanBeForkedWithoutDisturbingIt(t *testing.T) {
	fork := skingen.Nocturne()
	fork.Name, fork.Label = "nocturne-mine", "Nocturne, mine"
	fork.Colors = map[string]string{"accent": "#ff0000"}

	fresh := skingen.Nocturne()
	if fresh.Name != "nocturne" || fresh.Label == fork.Label {
		t.Fatalf("forking a plan changed the shipped one: %q / %q", fresh.Name, fresh.Label)
	}
	if fresh.Colors["accent"] == "#ff0000" {
		t.Error("the fork's palette leaked into the shipped plan")
	}

	dir := t.TempDir()
	if err := skingen.Write(dir, fork); err != nil {
		t.Fatalf("Write the fork: %v", err)
	}
	sk, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, fork.Name)), fork.Name)
	if err != nil {
		t.Fatalf("the fork does not load: %v", err)
	}
	if sk.Label != "Nocturne, mine" {
		t.Errorf("the fork loaded as %q", sk.Label)
	}
}

// Every plan the toolkit ships is named, labelled and backed by a pack, and
// every one of them loads. Plans() is a menu an author picks a starting
// point from, so a plan in it that did not load would be a dead end.
func TestEveryShippedPlanIsAWholeSkin(t *testing.T) {
	plans := skingen.Plans()
	if len(plans) != 8 {
		t.Errorf("Plans() has %d plans, expected the eight that ship", len(plans))
	}
	dir := t.TempDir()
	seen := map[string]bool{}
	for _, p := range plans {
		if p.Name == "" || p.Label == "" || p.Base == "" {
			t.Errorf("plan %q: name, label and base are all required", p.Name)
		}
		if seen[p.Name] {
			t.Errorf("two plans are called %q", p.Name)
		}
		seen[p.Name] = true
		if err := skingen.Write(dir, p); err != nil {
			t.Fatalf("%s: %v", p.Name, err)
		}
		if _, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name); err != nil {
			t.Errorf("%s does not load: %v", p.Name, err)
		}
	}
}

// Hex is the one drawing helper the package exports, because a plan forked
// into another package states its palette in hex strings on every line.
func TestHexReadsThePaletteNotationThePlansUse(t *testing.T) {
	for _, c := range []struct {
		in         string
		r, g, b, a float32
	}{
		{"#fff", 1, 1, 1, 1},
		{"#000000", 0, 0, 0, 1},
		{"#40444c", 0x40 / 255.0, 0x44 / 255.0, 0x4c / 255.0, 1},
		{"#ffffff80", 1, 1, 1, 0x80 / 255.0},
	} {
		got := skingen.Hex(c.in)
		if got.R != c.r || got.G != c.g || got.B != c.b || got.A != c.a {
			t.Errorf("Hex(%q) = %v, want %g %g %g %g", c.in, got, c.r, c.g, c.b, c.a)
		}
	}
}

// The pixel alphabet the two panel skins print their displays in is public
// as a pair: the runes it draws, and the sprite name each one is published
// under. An app setting a line needs both, and so does anyone drawing a
// display of their own.
func TestThePixelFontNamesEveryGlyphItDraws(t *testing.T) {
	runes := skingen.PixFontRunes()
	if len(runes) == 0 {
		t.Fatal("the pixel face draws no runes at all")
	}
	if got := skingen.PixFontSprite('A'); got != "font.41" {
		t.Errorf("PixFontSprite('A') = %q, want font.41", got)
	}

	cells := map[string]bool{}
	for _, sh := range skingen.MinimClassic().Sheets {
		for _, c := range sh.Cells {
			cells[c.Name] = true
		}
	}
	for _, r := range runes {
		if !cells[skingen.PixFontSprite(r)] {
			t.Errorf("%q is in the face but no cell draws %s", r, skingen.PixFontSprite(r))
		}
	}
}
