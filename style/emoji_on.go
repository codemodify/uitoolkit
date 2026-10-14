//go:build font_emoji

package style

import (
	"bytes"
	"image"
	"image/png"
	"sync"
	"sync/atomic"

	gofont "github.com/go-text/typesetting/font"

	"github.com/codemodify/paintengine2d"
)

// Colour emoji, behind the font_emoji tag.
//
// An emoji face does not draw outlines. Noto Color Emoji — the one on
// most Linux machines — has no `glyf` and no `CFF` table at all: every
// glyph is a PNG in `CBDT`, chosen from `CBLC` by pixel size. Apple's is
// `sbix`, also PNG. So the glyph the rest of the toolkit bakes, a path
// filled into a coverage mask, does not exist here, and the sheet it
// would be baked into cannot hold it: a glyph sheet is A8, one byte of
// coverage a pixel, tinted when it is drawn.
//
// Hence a sheet of its own, in colour, with the picture scaled into it
// and no tint — [Font.Draw] paints a sheet that carries its own colour
// with white, which multiplies to itself. The cells are keyed the way
// every shaped glyph is, by index with the top bit set, and the shaper
// chooses them: an emoji font's GSUB is what turns the five runes of a
// family emoji into one glyph (UTS #51), and that is why font_emoji
// implies font_complex rather than standing on its own.
//
// What it does not do: COLR/CPAL (layered outlines, and gradients in
// v1) and SVG glyphs. Both are parsed by go-text and neither is drawn
// here; a face that has only those falls through to the box, as before.

// defaultEmojiFamilies are the faces that carry emoji, in the order they
// are asked for: the platform's own first, then the ones a person
// installs. The list is walked only for a rune nothing else could draw.
var defaultEmojiFamilies = []string{
	"Noto Color Emoji",
	"Apple Color Emoji",
	"Segoe UI Emoji",
	"Twemoji Mozilla",
	"JoyPixels",
	"EmojiOne Color",
	"OpenMoji Color",
}

// emojiFaces caches the parsed face per family.
//
// Only the faces that were *found*. A family that is not installed is
// looked for again next time, because "not installed" is a fact about
// the machine and not about the toolkit: the installed-font index is
// built once per process and a test — or an application that writes a
// font file and asks again — can change what it holds. Remembering the
// absence made the answer permanent, and the emoji tests then passed
// alone and failed in the suite, which is the shape of a bug that is
// about to be shipped.
//
// The cost of not caching it is one walk of the family list for a rune
// nothing else could draw, and shaped runs are cached above this.
var emojiFaces = struct {
	mu sync.Mutex
	m  map[string]*gofont.Face
}{m: map[string]*gofont.Face{}}

func emojiFace(family string) *gofont.Face {
	emojiFaces.mu.Lock()
	defer emojiFaces.mu.Unlock()
	if g, ok := emojiFaces.m[family]; ok && g != nil {
		return g
	}
	// systemOTFace, not faceFor: faceFor never fails — a family the
	// machine has not got comes back as the bundled interface face,
	// which parses perfectly and has no emoji in it. Cached under the
	// name of the family that was asked for, that stand-in *became*
	// the answer for the rest of the process, and the first test to
	// draw a missing rune poisoned the ones after it.
	f := systemOTFace(family, WeightRegular)
	if f == nil {
		return nil
	}
	g := f.gotext()
	if g == nil {
		return nil
	}
	emojiFaces.m[family] = g
	return g
}

// emojiSheets is one sheet per face and size.
var emojiSheets = struct {
	mu sync.Mutex
	m  map[emojiKey]*emojiAtlas
}{m: map[emojiKey]*emojiAtlas{}}

type emojiKey struct {
	face *gofont.Face
	size int // quantized, as the font cache quantizes
	asc  int
}

// emojiSheetFor is the face that draws r in colour and the sheet its
// glyphs are baked into, or nil: a rune no emoji face has, or a build
// where none is installed, is the box it always was.
func emojiSheetFor(f *Font, r rune) (*gofont.Face, *emojiAtlas) {
	if f == nil {
		return nil, nil
	}
	for _, family := range defaultEmojiFamilies {
		g := emojiFace(family)
		if g == nil {
			continue
		}
		unlock := lockFace(g)
		gid, ok := g.NominalGlyph(r)
		unlock()
		if !ok || gid == 0 {
			continue
		}
		// A face that has the rune but no picture for it is no use
		// here: it would draw nothing at all, where the box at least
		// says a character is there.
		if _, ok := bitmapOf(g, gid, f.Size); !ok {
			continue
		}
		return g, emojiSheet(g, f)
	}
	return nil, nil
}

func emojiSheet(g *gofont.Face, f *Font) *emojiAtlas {
	key := emojiKey{face: g, size: quantFont(f.Size), asc: quantFont(f.Ascent)}
	emojiSheets.mu.Lock()
	defer emojiSheets.mu.Unlock()
	if a, ok := emojiSheets.m[key]; ok {
		return a
	}
	a := &emojiAtlas{face: g, size: f.Size, ascent: f.Ascent}
	if a.ascent <= 0 {
		a.ascent = f.Size
	}
	emojiSheets.m[key] = a
	return a
}

// emojiAtlas is a colour glyph sheet: one face, one size, cells keyed by
// glyph index exactly as the outline sheets are.
type emojiAtlas struct {
	face   *gofont.Face
	size   float32
	ascent float32

	mu     sync.Mutex
	packer shelfPacker
	cur    atomic.Pointer[paintengine2d.FontAtlas]
}

// atlas is the published sheet, or nil before the first bake.
func (a *emojiAtlas) atlas() *paintengine2d.FontAtlas {
	if a == nil {
		return nil
	}
	return a.cur.Load()
}

const (
	emojiSheetStart = 256
	emojiSheetCap   = 2048
)

// ensureGIDs bakes every glyph of a run that is not on the sheet yet and
// publishes the result, the way [otAtlas.ensureGIDs] does for outlines.
func (a *emojiAtlas) ensureGIDs(gids []uint32) *paintengine2d.FontAtlas {
	if a == nil {
		return nil
	}
	cur := a.atlas()
	if !a.missing(cur, gids) {
		return cur
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	cur = a.atlas()
	if !a.missing(cur, gids) {
		return cur
	}

	var img *paintengine2d.Image
	cells := map[paintengine2d.GlyphID]paintengine2d.AtlasCell{}
	if cur == nil || cur.Image == nil {
		a.packer = shelfPacker{w: emojiSheetStart, h: emojiSheetStart, x: 1, y: 1, pad: 1}
		img = paintengine2d.NewImage(a.packer.w, a.packer.h)
	} else {
		img = cloneImage(cur.Image, cur.Image.Width, cur.Image.Height)
		for k, v := range cur.Cells {
			cells[k] = v
		}
	}
	for _, gid := range gids {
		id := shapedGIDBit | paintengine2d.GlyphID(gid)
		if _, ok := cells[id]; ok {
			continue
		}
		cell, ok := a.bake(&img, gid)
		if !ok {
			continue
		}
		cells[id] = cell
	}
	img.Bump()
	cells[colourSheetMark] = paintengine2d.AtlasCell{}
	sheet := &paintengine2d.FontAtlas{Image: img, Cells: cells}
	a.cur.Store(sheet)
	return sheet
}

// colourSheetMark is the key of a cell that is never drawn and says
// only "the colours in this sheet are the face's own".
//
// It lives in the sheet because that is what a cached run holds. The
// first version kept a set of the sheets that were in colour and
// deleted each one as it was superseded, which is tidy and wrong: a run
// shaped earlier still names the older sheet, so growing the atlas
// turned text that had been drawn in colour into text that was tinted
// again — found by a music player, which baked a second emoji and
// watched the first one change colour with the label.
//
// Nothing can collide with it: a rune key stops at 0x10FFFF and a glyph
// key is an OpenType index below 0x10000 with the top bit set.
const colourSheetMark = paintengine2d.GlyphID(1) << 30

// isColourSheet reports that a sheet carries its own colours and must
// be drawn as it is.
//
// Asked of the sheet, not of a table beside it. The image's format
// cannot answer — the toolkit's own bitmap font is RGBA as well and
// *is* tinted, being a white mask that happens to be kept in four
// bytes a pixel — so the sheet carries [colourSheetMark].
func isColourSheet(a *paintengine2d.FontAtlas) bool {
	if a == nil || a.Cells == nil {
		return false
	}
	_, ok := a.Cells[colourSheetMark]
	return ok
}

func (a *emojiAtlas) missing(cur *paintengine2d.FontAtlas, gids []uint32) bool {
	for _, gid := range gids {
		if cur == nil {
			return true
		}
		if _, ok := cur.Cell(shapedGIDBit | paintengine2d.GlyphID(gid)); !ok {
			return true
		}
	}
	return false
}

// bake scales one strike into the sheet, growing it if there is no room.
func (a *emojiAtlas) bake(img **paintengine2d.Image, gid uint32) (paintengine2d.AtlasCell, bool) {
	src, ok := bitmapOf(a.face, gofont.GID(gid), a.size)
	if !ok {
		return paintengine2d.AtlasCell{}, false
	}
	// As tall as the line's ascent, which puts the picture between the
	// baseline and the top of the text beside it, and as wide as its
	// own proportions make it.
	h := a.ascent
	if h < 4 {
		h = 4
	}
	w := h
	if b := src.Bounds(); b.Dy() > 0 {
		w = h * float32(b.Dx()) / float32(b.Dy())
	}
	gw, gh := int(w+1.5), int(h+1.5)
	x, y, ok := a.pack(img, gw, gh)
	if !ok {
		return paintengine2d.AtlasCell{}, false
	}
	ctx := paintengine2d.NewContext(*img)
	ctx.DrawImageRect(src,
		paintengine2d.XYWH(0, 0, float32(src.Width), float32(src.Height)),
		paintengine2d.XYWH(float32(x), float32(y), float32(gw), float32(gh)))
	return paintengine2d.AtlasCell{
		Src: paintengine2d.XYWH(float32(x), float32(y), float32(gw), float32(gh)),
		// From the top of the line, like every other cell on every
		// other sheet: an outline's bearing is `ascent + ymin`, and
		// the draw adds it to the origin, which is the line's top. A
		// bearing measured from the *baseline* put each emoji a whole
		// ascent above its line, where the window clipped it away.
		Bearing: paintengine2d.Pt(0, a.ascent-h),
		Advance: w + a.size*0.08,
	}, true
}

// pack finds room, doubling the sheet while it can.
func (a *emojiAtlas) pack(img **paintengine2d.Image, gw, gh int) (int, int, bool) {
	for {
		if x, y, ok := a.packer.add(gw, gh); ok {
			return x, y, true
		}
		if a.packer.h >= emojiSheetCap {
			return 0, 0, false
		}
		a.packer.h *= 2
		*img = cloneImage(*img, a.packer.w, a.packer.h)
	}
}

// bitmapOf is the face's own picture for one glyph at this size, as a
// premultiplied image. PNG is what CBDT and sbix carry in practice; the
// other formats a font may use are not decoded here.
func bitmapOf(g *gofont.Face, gid gofont.GID, size float32) (*paintengine2d.Image, bool) {
	ppem := uint16(size + 0.5)
	if ppem < 8 {
		ppem = 8
	}
	// Under the face's own lock: SetPpem *writes* to the face, and one
	// face serves every size the application draws at.
	unlock := lockFace(g)
	g.SetPpem(ppem, ppem)
	bm, ok := g.GlyphDataBitmap(gid)
	unlock()
	if !ok || len(bm.Data) == 0 || bm.Format != gofont.PNG {
		return nil, false
	}
	src, err := png.Decode(bytes.NewReader(bm.Data))
	if err != nil {
		return nil, false
	}
	return premultiplied(src), true
}

// premultiplied copies a decoded picture into the engine's own image,
// which is premultiplied RGBA like every other surface it draws.
func premultiplied(src image.Image) *paintengine2d.Image {
	b := src.Bounds()
	out := paintengine2d.NewImage(b.Dx(), b.Dy())
	for y := 0; y < b.Dy(); y++ {
		row := y * out.Stride
		for x := 0; x < b.Dx(); x++ {
			// RGBA() is already alpha-premultiplied, in 16 bits.
			r, g, bl, al := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			i := row + x*4
			out.Pix[i+0] = byte(r >> 8)
			out.Pix[i+1] = byte(g >> 8)
			out.Pix[i+2] = byte(bl >> 8)
			out.Pix[i+3] = byte(al >> 8)
		}
	}
	return out
}
