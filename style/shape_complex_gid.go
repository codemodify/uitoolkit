//go:build font_complex || font_emoji

package style

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"

	"github.com/codemodify/paintengine2d"
)

// ensureGID bakes the glyph at index gid into the sheet, keyed by index
// rather than by rune. It is rasterize() with its first three lines gone:
// the rune → index lookup is what the shaper has already done, and the
// glyph it chose may have no rune at all.
func (a *otAtlas) ensureGID(gid uint32) *paintengine2d.FontAtlas {
	if a == nil {
		return nil
	}
	id := shapedGIDBit | paintengine2d.GlyphID(gid)
	cur := a.atlas()
	if cur != nil {
		if _, ok := cur.Cell(id); ok {
			return cur
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	cur = a.atlas()
	if cur != nil {
		if _, ok := cur.Cell(id); ok {
			return cur
		}
	}
	d := a.draft(cur)
	if _, ok := d.cells[id]; ok {
		return a.publish(d)
	}
	tf := a.face
	ppem := ppem26(a.size)
	tf.mu.Lock()
	adv, err := tf.font.GlyphAdvance(&tf.buf, sfnt.GlyphIndex(gid), ppem, font.HintingNone)
	if err != nil {
		adv = 0
	}
	segs, err := tf.font.LoadGlyph(&tf.buf, sfnt.GlyphIndex(gid), ppem, nil)
	// Under the lock, because the segments are the Buffer's own
	// memory: see the note in rasterize.
	var p *paintengine2d.Path
	if err == nil && len(segs) > 0 {
		p = segmentsPath(segs)
	}
	tf.mu.Unlock()
	if err != nil {
		return a.publish(d)
	}
	advance := fx32(adv)
	if tf.embolden > 0 {
		advance += tf.embolden * a.size
	}
	d.blitGID(id, p, advance)
	return a.publish(d)
}

// ensureGIDs bakes a whole run's glyphs in one draft, which is what a
// string of shaped text wants: ensureGID publishes a sheet per glyph,
// and a word of Arabic is a dozen of them.
func (a *otAtlas) ensureGIDs(gids []uint32) *paintengine2d.FontAtlas {
	if a == nil {
		return nil
	}
	cur := a.atlas()
	missing := false
	for _, gid := range gids {
		if cur == nil {
			missing = true
			break
		}
		if _, ok := cur.Cell(shapedGIDBit | paintengine2d.GlyphID(gid)); !ok {
			missing = true
			break
		}
	}
	if !missing {
		return cur
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	cur = a.atlas()
	d := a.draft(cur)
	for _, gid := range gids {
		id := shapedGIDBit | paintengine2d.GlyphID(gid)
		if _, ok := d.cells[id]; ok {
			continue
		}
		p, advance, ok := a.outlineOf(gid)
		if !ok {
			continue
		}
		d.blitGID(id, p, advance)
	}
	return a.publish(d)
}

// outlineOf is the glyph at index gid, rasterized at this sheet's size.
func (a *otAtlas) outlineOf(gid uint32) (*paintengine2d.Path, float32, bool) {
	tf := a.face
	ppem := ppem26(a.size)
	tf.mu.Lock()
	adv, err := tf.font.GlyphAdvance(&tf.buf, sfnt.GlyphIndex(gid), ppem, font.HintingNone)
	if err != nil {
		adv = 0
	}
	segs, err := tf.font.LoadGlyph(&tf.buf, sfnt.GlyphIndex(gid), ppem, nil)
	// Under the lock, because the segments are the Buffer's own
	// memory: see the note in rasterize.
	var p *paintengine2d.Path
	if err == nil && len(segs) > 0 {
		p = segmentsPath(segs)
	}
	tf.mu.Unlock()
	if err != nil {
		return nil, 0, false
	}
	advance := fx32(adv)
	if tf.embolden > 0 {
		advance += tf.embolden * a.size
	}
	return p, advance, true
}

// hasGlyph reports whether the face has a glyph of its own for r. It is
// [Font.covers] without the baking: the segmenter asks it for every rune
// of every string, and a sheet is not the place to answer from.
func (f *otFace) hasGlyph(r rune) bool {
	if f == nil || f.font == nil {
		return false
	}
	f.mu.Lock()
	gid, err := f.font.GlyphIndex(&f.buf, r)
	f.mu.Unlock()
	return err == nil && gid != 0
}

// blitGID is blitPath keyed by glyph index instead of by rune. That it is
// four lines, and the only untagged machinery it needs is bakeCell, is the
// spike's main finding: the sheet, the packer and the renderer do not care
// what a key means.
func (d *atlasDraft) blitGID(id paintengine2d.GlyphID, p *paintengine2d.Path, advance float32) {
	cell, err := d.bakeCell(p, advance)
	if err != nil {
		return
	}
	d.cells[id] = cell
}
