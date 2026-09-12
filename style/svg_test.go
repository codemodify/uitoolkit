package style

import "testing"

func TestParseSVGPathAndTintableFill(t *testing.T) {
	raw := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <path fill="currentColor" d="M4 4h16v16H4z"/>
  <path fill="currentColor" opacity="0.4" d="M8 8h8v8H8z"/>
</svg>`)
	doc, err := parseSVG(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.shapes) != 2 {
		t.Fatalf("shapes %d", len(doc.shapes))
	}
	if !doc.shapes[0].fill || doc.shapes[0].opacity < 0.99 {
		t.Fatalf("primary %+v", doc.shapes[0])
	}
	if doc.shapes[1].opacity < 0.35 || doc.shapes[1].opacity > 0.45 {
		t.Fatalf("duotone opacity %v", doc.shapes[1].opacity)
	}
}

func TestParseSVGStrokeOutline(t *testing.T) {
	raw := []byte(`<svg viewBox="0 0 24 24" fill="none">
  <circle fill="none" stroke="currentColor" stroke-width="1.75" cx="12" cy="12" r="8"/>
</svg>`)
	doc, err := parseSVG(raw)
	if err != nil || len(doc.shapes) != 1 {
		t.Fatalf("%v n=%d", err, len(doc.shapes))
	}
	if !doc.shapes[0].stroke || doc.shapes[0].fill {
		t.Fatalf("%+v", doc.shapes[0])
	}
}

func TestParsePathRelativeAndArc(t *testing.T) {
	p := parsePathData("M10 10 l 5 0 A 2 2 0 0 1 17 12 z")
	if p.Empty() {
		t.Fatal("empty")
	}
	if len(p.Verbs()) < 3 {
		t.Fatalf("verbs %d", len(p.Verbs()))
	}
}
