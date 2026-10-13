//go:build font_complex

// Shaping, behind the font_complex tag. A SPIKE: it proves what the real
// thing costs and where it lands, and nothing in the toolkit calls it yet.
//
// What it is here to answer:
//
//  1. A shaper's output is glyph *indices*, not runes. Everything in this
//     package keys the atlas by rune. How much has to change?
//  2. Can the existing rasterizer draw a glyph the shaper chose — one with
//     no rune of its own, like an Arabic medial form or a ligature?
//  3. What does the binary weigh with it in?
//
// The answers are in docs/complex-text-spike.md.
package style

import (
	"bytes"
	"sync"

	"github.com/go-text/typesetting/di"
	gofont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"github.com/codemodify/paintengine2d"
)

// shapedGIDBit marks an atlas key as a glyph index rather than a rune.
//
// The one real collision in the design: an atlas is keyed by
// [paintengine2d.GlyphID], the toolkit stores runes in it, and a shaper
// answers in the font's own glyph numbering — where 65 is whatever glyph
// the font put at index 65, not 'A'. Runes stop at 0x10FFFF, so the top
// bit is free to say which numbering a key is in, and one sheet can hold
// both.
const shapedGIDBit = paintengine2d.GlyphID(1) << 31

// shapedGlyph is one glyph the shaper placed, in pixels.
type shapedGlyph struct {
	GID     uint32
	Advance float32
	X, Y    float32 // offset from the pen, y down
	Cluster int
}

// gotextFaces parses each otFace's bytes a second time, for the shaper.
//
// A real implementation would not: go-text can rasterize as well, and the
// toolkit would drop x/image/font/sfnt rather than carry both parses of
// every face. Here it keeps the spike additive.
var gotextFaces = struct {
	mu sync.Mutex
	m  map[*otFace]*gofont.Face
}{m: map[*otFace]*gofont.Face{}}

func (f *otFace) gotext() *gofont.Face {
	if f == nil || len(f.src) == 0 {
		return nil
	}
	gotextFaces.mu.Lock()
	defer gotextFaces.mu.Unlock()
	if g, ok := gotextFaces.m[f]; ok {
		return g
	}
	g, err := gofont.ParseTTF(bytes.NewReader(f.src))
	if err != nil {
		g = nil
	}
	gotextFaces.m[f] = g
	return g
}

// scriptDirection is the direction a script is written in. A paragraph of
// mixed text needs UAX #9 (the bidi package, +0.10 MB) to split into runs
// before this is asked; one run has one direction.
func scriptDirection(s language.Script) di.Direction {
	switch s {
	case language.Arabic, language.Hebrew, language.Syriac, language.Thaana,
		language.Nko, language.Adlam:
		return di.DirectionRTL
	}
	return di.DirectionLTR
}

var shaperPool = sync.Pool{New: func() any { return &shaping.HarfbuzzShaper{} }}

// shapeComplex runs the face's own OpenType tables over text: GSUB for the
// substitutions (joining forms, ligatures, Indic reordering) and GPOS for
// the positioning (kerning, mark attachment).
func (f *otFace) shapeComplex(text string, size float32) []shapedGlyph {
	g := f.gotext()
	if g == nil || text == "" {
		return nil
	}
	runes := []rune(text)
	in := shaping.Input{
		Text:     runes,
		RunStart: 0,
		RunEnd:   len(runes),
		Face:     g,
		Size:     fixed.Int26_6(size * 64),
		Script:   language.LookupScript(runes[0]),
		Language: language.NewLanguage("en"),
	}
	in.Direction = scriptDirection(in.Script)
	sh := shaperPool.Get().(*shaping.HarfbuzzShaper)
	defer shaperPool.Put(sh)
	out := sh.Shape(in)
	glyphs := make([]shapedGlyph, 0, len(out.Glyphs))
	for _, gl := range out.Glyphs {
		glyphs = append(glyphs, shapedGlyph{
			GID:     uint32(gl.GlyphID),
			Advance: fx32(gl.Advance),
			X:       fx32(gl.XOffset),
			Y:       -fx32(gl.YOffset),
			Cluster: gl.TextIndex(),
		})
	}
	return glyphs
}

// glyphRun turns shaped glyphs into something the renderer already draws:
// one [paintengine2d.GlyphRun] over an atlas keyed by glyph index. The pen
// walks the advances the shaper gave, which is the whole of positioning
// for a horizontal run.
func (a *otAtlas) glyphRun(glyphs []shapedGlyph) paintengine2d.GlyphRun {
	run := paintengine2d.GlyphRun{}
	var pen float32
	for _, g := range glyphs {
		at := a.ensureGID(g.GID)
		if at == nil {
			break
		}
		run.Atlas = at
		run.Glyphs = append(run.Glyphs, paintengine2d.Glyph{
			ID: shapedGIDBit | paintengine2d.GlyphID(g.GID),
			X:  pen + g.X,
			Y:  g.Y,
		})
		pen += g.Advance
	}
	return run
}
