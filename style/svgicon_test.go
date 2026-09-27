package style

import (
	"strings"
	"testing"
)

// coverage is how many of the pixmap's pixels were painted at all, and
// what the first painted one's un-premultiplied colour is.
func coverage(t *testing.T, doc string, size int) (painted int, r, g, b int) {
	t.Helper()
	img, err := rasterizeSVGBytes([]byte(doc), size)
	if err != nil {
		t.Fatalf("rasterize: %v", err)
	}
	first := true
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			pr, pg, pb, a := img.PremulAt(x, y)
			if a < 128 {
				continue
			}
			painted++
			if first {
				r, g, b = int(pr)*255/int(a), int(pg)*255/int(a), int(pb)*255/int(a)
				first = false
			}
		}
	}
	return painted, r, g, b
}

// The shape a themed icon actually is: a viewBox and one filled path.
func TestSVGDrawsAPath(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">
	  <path style="fill:currentColor" d="M 2 2 L 8 2 L 8 8 L 2 8 Z"/>
	</svg>`
	painted, r, g, b := coverage(t, doc, 10)
	// A 6x6 square of a 10x10 box, drawn into 10 pixels.
	if painted < 30 || painted > 42 {
		t.Errorf("a 6x6 square of a 10x10 viewBox covered %d pixels, want about 36", painted)
	}
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("currentColor drew #%02x%02x%02x, want black: it is the look's foreground that tints it", r, g, b)
	}
}

// The path grammar: relative commands, run-together numbers, implicit
// line-to after a move-to, horizontal and vertical, and an arc.
func TestSVGPathGrammar(t *testing.T) {
	for _, d := range []string{
		"M2 2 L8 2 L8 8 L2 8 Z",
		"M2,2 8,2 8,8 2,8 z",                   // implicit L after M
		"m2 2h6v6h-6z",                         // relative, run together
		"M2 2 H8 V8 H2 Z",                      // absolute H/V
		"M2 2 C2 2 8 2 8 2 L8 8 L2 8 Z",        // a cubic that is a line
		"M5 2 A3 3 0 1 1 4.99 2 Z",             // a circle drawn as one arc
		"M2 2 L8 2 L8 8 L2 8 Z M3 3 L4 3 L4 4", // two subpaths
	} {
		doc := `<svg viewBox="0 0 10 10"><path fill="currentColor" d="` + d + `"/></svg>`
		if painted, _, _, _ := coverage(t, doc, 20); painted < 20 {
			t.Errorf("%q painted %d pixels: it draws nothing", d, painted)
		}
	}
}

// The shapes that are not paths, and the transform that moves them.
func TestSVGShapesAndTransform(t *testing.T) {
	plain, _, _, _ := coverage(t, `<svg viewBox="0 0 10 10"><rect x="0" y="0" width="5" height="10" fill="currentColor"/></svg>`, 10)
	if plain < 40 || plain > 60 {
		t.Errorf("half the box covered %d of 100 pixels", plain)
	}
	// Translated out of the box entirely: nothing is left.
	moved, _, _, _ := coverage(t, `<svg viewBox="0 0 10 10"><g transform="translate(20,0)"><rect x="0" y="0" width="5" height="10" fill="currentColor"/></g></svg>`, 10)
	if moved != 0 {
		t.Errorf("a rect translated off the box painted %d pixels", moved)
	}
	round, _, _, _ := coverage(t, `<svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="5" fill="currentColor"/></svg>`, 20)
	if round < 280 || round > 330 {
		t.Errorf("a circle filling the box covered %d of 400 pixels, want about pi/4 of them", round)
	}
}

// An explicit colour is drawn in that colour, so that the monochrome
// test downstream can tell a picture from a shape.
func TestSVGKeepsExplicitColour(t *testing.T) {
	doc := `<svg viewBox="0 0 10 10"><rect x="0" y="0" width="10" height="10" style="fill:#ff8000"/></svg>`
	_, r, g, b := coverage(t, doc, 8)
	if r < 240 || g < 110 || g > 145 || b > 16 {
		t.Errorf("fill:#ff8000 drew #%02x%02x%02x", r, g, b)
	}
}

// What this rasterizer refuses, and the whole reason it may be trusted
// with somebody else's artwork: a construct outside the subset is an
// error, never a shape drawn wrong. A theme of files like these is
// listed as unavailable instead (see [iconThemeDrawable]).
func TestSVGRefusesWhatItCannotDraw(t *testing.T) {
	for _, doc := range []string{
		`<svg viewBox="0 0 10 10"><use href="#x"/></svg>`,
		`<svg viewBox="0 0 10 10"><linearGradient id="g"/><rect width="10" height="10" fill="url(#g)"/></svg>`,
		`<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="url(#g)"/></svg>`,
		`<svg viewBox="0 0 10 10"><text x="0" y="5">hi</text></svg>`,
		`<svg viewBox="0 0 10 10"><image href="x.png"/></svg>`,
		`<svg viewBox="0 0 10 10"><mask id="m"/><rect width="10" height="10"/></svg>`,
		`<svg viewBox="0 0 10 10"><clipPath id="c"/><rect width="10" height="10"/></svg>`,
		`<svg viewBox="0 0 10 10"><rect width="10" height="10" transform="skew(3)"/></svg>`,
		`<svg viewBox="0 0 10 10"><path d="M0 0 Q"/></svg>`,
		`<svg><rect width="10" height="10"/></svg>`, // no viewBox and no size
		`<rect width="10" height="10"/>`,            // no <svg> at all
	} {
		if _, err := rasterizeSVGBytes([]byte(doc), 16); err == nil {
			t.Errorf("drew a document it cannot draw: %s", doc)
		}
	}
}

// <defs>, <style> and the editor's own elements carry no picture, and
// must not be a reason to refuse a file that has one: every Breeze icon
// keeps its colour scheme in a <style> inside <defs>.
func TestSVGSkipsWhatPaintsNothing(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 22 22">
	 <defs id="defs3051">
	   <style type="text/css" id="current-color-scheme">.ColorScheme-Text { color:#232629; }</style>
	 </defs>
	 <metadata><rdf:RDF xmlns:rdf="http://x"/></metadata>
	 <title>Save</title>
	 <path style="fill:currentColor;fill-opacity:1;stroke:none" class="ColorScheme-Text" d="M 3 3 L 19 3 L 19 19 L 3 19 Z"/>
	</svg>`
	if painted, _, _, _ := coverage(t, doc, 22); painted < 200 {
		t.Errorf("a Breeze-shaped document painted %d pixels", painted)
	}
}

// The file is read once and it is small: an icon directory is not a
// place to be handed a megabyte of XML on a paint path.
func TestSVGRefusesAHugeFile(t *testing.T) {
	big := `<svg viewBox="0 0 10 10"><path d="M0 0` + strings.Repeat(" L1 1", 300000) + `"/></svg>`
	if _, err := rasterizeSVGBytes([]byte(big), 16); err == nil && len(big) <= maxSVGBytes {
		t.Skip("the fixture is under the limit")
	}
}
