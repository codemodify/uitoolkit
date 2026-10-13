//go:build font_complex || font_emoji

// Shaping, behind the font_complex tag: the pieces that talk to the
// shaper. [complexLayout] in shape_complex_layout.go is what uses them,
// and docs/complex-text.md is the whole picture — what the tag adds,
// what it costs, and what is still missing.
package style

import (
	"bytes"
	"fmt"
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
	// byRune says GID holds a rune, not a glyph index: the shaper had
	// nothing for this cluster and the rune path's own answers — the
	// drawn star or arrow of symbols.go, or the .notdef box — are
	// better than an empty cell. They are baked and keyed by rune, as
	// they always were.
	byRune bool
}

// key is where this glyph lives in a sheet.
func (g shapedGlyph) key() paintengine2d.GlyphID {
	if g.byRune {
		return paintengine2d.GlyphID(g.GID)
	}
	return shapedGIDBit | paintengine2d.GlyphID(g.GID)
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
	g, err := parseGotext(f.src, f.index)
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

// parseGotext reads one face of a font file.
//
// A collection has to be said out loud: every Noto CJK family ships as
// one — four languages' worth of Han in a single .ttc — and ParseTTF
// refuses the file outright. Falling back to "no shaper for this face"
// there meant every borrowed glyph in a shaping build was a box, which
// is what the plain fallback build was written to stop.
func parseGotext(src []byte, index int) (*gofont.Face, error) {
	if len(src) >= 4 && string(src[:4]) == "ttcf" {
		faces, err := gofont.ParseTTC(bytes.NewReader(src))
		if err != nil {
			return nil, err
		}
		if index < 0 || index >= len(faces) {
			index = 0
		}
		if len(faces) == 0 {
			return nil, fmt.Errorf("style: %d faces in the collection", len(faces))
		}
		return faces[index], nil
	}
	return gofont.ParseTTF(bytes.NewReader(src))
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
