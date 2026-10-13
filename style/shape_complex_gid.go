//go:build font_complex

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
	tf.mu.Unlock()
	if err != nil {
		return a.publish(d)
	}
	advance := fx32(adv)
	if tf.embolden > 0 {
		advance += tf.embolden * a.size
	}
	var p *paintengine2d.Path
	if len(segs) > 0 {
		p = segmentsPath(segs)
	}
	d.blitGID(id, p, advance)
	return a.publish(d)
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
