package style

import (
	"fmt"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// Font is a baked glyph atlas painted through paintengine2d.
// Default UI/mono faces are OpenType outlines (Titillium Web / JetBrains Mono)
// rasterized with the engine's scanline AA into a white sheet; Paint.Color tints.
//
// A Font value is a cheap handle: BakeFamily hands out copies that share one
// copy-on-write glyph sheet (see [Font.GlyphAtlas]), so copies are safe to use
// from different goroutines.
type Font struct {
	Size    float32
	Ascent  float32
	Descent float32
	Color   paintengine2d.Color
	Family  string
	Weight  Weight
	Outline bool // true when built from TTF outlines (not the 5×7 bitmap)
	// static is the immutable sheet of a bitmap face (ot == nil).
	static *paintengine2d.FontAtlas
	ot     *otAtlas
	shaped *shapeCache
}

// GlyphAtlas is the face's current glyph sheet.
//
// The OpenType path publishes a new immutable sheet whenever a rune is baked,
// so the returned pointer is a snapshot: safe to read from any goroutine and
// never mutated after publication. Re-read it (do not cache it) after drawing
// text that may introduce new runes.
func (f *Font) GlyphAtlas() *paintengine2d.FontAtlas {
	if f == nil {
		return nil
	}
	if f.ot != nil {
		if a := f.ot.atlas(); a != nil {
			return a
		}
	}
	return f.static
}

// atlasBytes is the live sheet size (font cache budget accounting).
func (f *Font) atlasBytes() int64 {
	if f == nil {
		return 0
	}
	if f.ot != nil {
		return f.ot.atlasBytes()
	}
	if f.static != nil && f.static.Image != nil {
		return int64(f.static.Image.Width) * int64(f.static.Image.Height) * 4
	}
	return 0
}

// shapeCacheCap drops shaped runs once the map grows past this. Labels
// and menu titles reuse the same strings; list rows with unique text miss.
const shapeCacheCap = 512

type shapeCache struct {
	mu sync.Mutex
	m  map[string]shapedRun
}

type shapedRun struct {
	run paintengine2d.GlyphRun
	adv float32
	ink float32
}

func newShapeCache() *shapeCache {
	return &shapeCache{m: make(map[string]shapedRun, 32)}
}

// Fit returns text, or a prefix plus an ellipsis, that fits in maxW.
func (f *Font) Fit(text string, maxW float32) string {
	if f == nil || text == "" {
		return ""
	}
	if maxW <= 0 {
		return ""
	}
	if f.Advance(text) <= maxW {
		return text
	}
	const ell = "…"
	ew := f.Advance(ell)
	if maxW <= ew {
		return ell
	}
	budget := maxW - ew
	n := 0
	var acc float32
	atlas := f.ensure(text) // one publication for the whole string
	for _, r := range text {
		adv := advanceIn(atlas, r)
		if acc+adv > budget {
			break
		}
		acc += adv
		n++
	}
	if n <= 0 {
		return ell
	}
	return string([]rune(text)[:n]) + ell
}

func (f *Font) Measure(text string) paintengine2d.Point {
	if f == nil {
		return paintengine2d.Point{}
	}
	return paintengine2d.Pt(f.Advance(text), f.Height())
}

func (f *Font) Height() float32 {
	if f == nil {
		return 0
	}
	if f.Descent > 0 {
		return f.Ascent + f.Descent
	}
	return f.Size + 2
}

func (f *Font) Advance(text string) float32 {
	if f == nil {
		return 0
	}
	return f.shapeOf(text).adv
}

// InkWidth is the painted AABB width of text (advance plus last-glyph
// bearing / atlas pad). Menus use this so DrawGlyphs cannot clip a stem.
func (f *Font) InkWidth(text string) float32 {
	if f == nil {
		return 0
	}
	if text == "" {
		return 0
	}
	return f.shapeOf(text).ink
}

func (f *Font) IndexAt(text string, x float32) int {
	if f == nil || x <= 0 {
		return 0
	}
	var acc float32
	i := 0
	atlas := f.ensure(text)
	for _, r := range text {
		adv := advanceIn(atlas, r)
		if x < acc+adv*0.5 {
			return i
		}
		acc += adv
		i++
	}
	return i
}

// advanceIn is the advance of r in an already-baked sheet.
func advanceIn(atlas *paintengine2d.FontAtlas, r rune) float32 {
	cell, ok := atlas.Cell(paintengine2d.GlyphID(r))
	if !ok {
		return 0
	}
	if cell.Advance > 0 {
		return cell.Advance
	}
	return cell.Src.Dx()
}

// cellFor is the atlas cell for r, baking it on first use.
func (f *Font) cellFor(r rune) (paintengine2d.AtlasCell, bool) {
	if f == nil {
		return paintengine2d.AtlasCell{}, false
	}
	atlas := f.GlyphAtlas()
	if atlas == nil {
		return paintengine2d.AtlasCell{}, false
	}
	if c, ok := atlas.Cell(paintengine2d.GlyphID(r)); ok {
		return c, true
	}
	if f.ot == nil {
		return paintengine2d.AtlasCell{}, false
	}
	return f.ot.ensureRune(r).Cell(paintengine2d.GlyphID(r))
}

func (f *Font) runeAdvance(r rune) float32 {
	cell, ok := f.cellFor(r)
	if !ok {
		return 0
	}
	if cell.Advance > 0 {
		return cell.Advance
	}
	return cell.Src.Dx()
}

func (f *Font) CaretX(text string, i int) float32 {
	if f == nil || i <= 0 {
		return 0
	}
	n := 0
	var acc float32
	atlas := f.ensure(text)
	for _, r := range text {
		if n >= i {
			break
		}
		acc += advanceIn(atlas, r)
		n++
	}
	return acc
}

// ensure bakes every rune of text and returns the sheet to shape against.
func (f *Font) ensure(text string) *paintengine2d.FontAtlas {
	if f == nil {
		return nil
	}
	if f.ot == nil {
		return f.static
	}
	return f.ot.ensure(text)
}

func (f *Font) shapeOf(text string) shapedRun {
	if f == nil || text == "" {
		return shapedRun{}
	}
	atlas := f.ensure(text)
	if atlas == nil {
		return shapedRun{}
	}
	shape := func() shapedRun {
		return measureShaped(snapRun(paintengine2d.NullShaper{}.Shape(text, atlas), atlas), atlas)
	}
	if f.shaped == nil {
		return shape()
	}
	f.shaped.mu.Lock()
	defer f.shaped.mu.Unlock()
	if hit, ok := f.shaped.m[text]; ok {
		// Re-point a run shaped against an earlier snapshot: cells are only
		// ever added, so the positions still hold, and the superseded sheet
		// stops being pinned by the cache.
		if hit.run.Atlas != atlas {
			hit.run.Atlas = atlas
			f.shaped.m[text] = hit
		}
		return hit
	}
	hit := shape()
	if f.shaped.m == nil {
		f.shaped.m = make(map[string]shapedRun, 32)
	}
	if len(f.shaped.m) >= shapeCacheCap {
		clear(f.shaped.m)
	}
	f.shaped.m[text] = hit
	return hit
}

// snapRun rounds every glyph onto the pixel grid.
//
// Cells are rasterized with a fractional bearing and NullShaper accumulates
// fractional advances, so an unsnapped run lands the atlas blit between
// pixels: the engine then resamples the sheet (a 16px stem measured 255/2px
// crisp drops to ~191/3px) and loses its fast 1:1 path. Rounding the sum of
// position and bearing keeps every glyph within half a pixel of its ideal
// spot and makes the blit integer-aligned.
func snapRun(run paintengine2d.GlyphRun, atlas *paintengine2d.FontAtlas) paintengine2d.GlyphRun {
	if len(run.Glyphs) == 0 || atlas == nil {
		return run
	}
	out := run
	out.Glyphs = make([]paintengine2d.Glyph, len(run.Glyphs))
	for i, g := range run.Glyphs {
		if cell, ok := atlas.Cell(g.ID); ok {
			g.X = snapCoord(g.X+cell.Bearing.X) - cell.Bearing.X
			g.Y = snapCoord(g.Y+cell.Bearing.Y) - cell.Bearing.Y
		}
		out.Glyphs[i] = g
	}
	return out
}

func measureShaped(run paintengine2d.GlyphRun, atlas *paintengine2d.FontAtlas) shapedRun {
	var w float32
	for _, g := range run.Glyphs {
		if atlas == nil {
			break
		}
		cell, ok := atlas.Cell(g.ID)
		if !ok {
			continue
		}
		adv := cell.Advance
		if adv <= 0 {
			adv = cell.Src.Dx()
		}
		if g.X+adv > w {
			w = g.X + adv
		}
	}
	inkW := w
	ink := run.Bounds(paintengine2d.Pt(0, 0))
	if !ink.Empty() && ink.Max.X > w {
		inkW = ink.Max.X
	}
	return shapedRun{run: run, adv: w, ink: inkW}
}

// Draw paints text at origin (baseline-top of the em box) tinted with col.
//
// Under a pure translation the origin is snapped to whole device pixels and
// the sheet is blitted 1:1 with FilterNearest — crisp stems and the engine's
// fast blit. A scaled or rotated context resamples, so it keeps bilinear.
func (f *Font) Draw(ctx *paintengine2d.Context, text string, origin paintengine2d.Point, tint paintengine2d.Color) {
	if f == nil || ctx == nil || text == "" {
		return
	}
	run := f.shapeOf(text).run
	if len(run.Glyphs) == 0 {
		return
	}
	col := tint
	if col == (paintengine2d.Color{}) {
		col = f.Color
	}
	if col == (paintengine2d.Color{}) {
		col = paintengine2d.White
	}
	filter := paintengine2d.FilterNearest
	if m := ctx.Matrix(); m.IsTranslation() {
		origin = paintengine2d.Pt(snapCoord(origin.X+m.E)-m.E, snapCoord(origin.Y+m.F)-m.F)
	} else {
		filter = paintengine2d.FilterBilinear
	}
	ctx.DrawGlyphs(run, origin, paintengine2d.Paint{Color: col, Filter: filter})
}

type fontKey struct {
	family string
	weight Weight
	size   int
}

// fontCacheBudget caps the glyph sheets the size cache keeps alive. Sheets
// grow as new runes are baked, so the budget is re-checked on every bake and
// the least recently used faces are dropped (a Font already handed out stays
// valid — only the cache entry goes).
//
// A var, not a const, so tests can shrink it.
var fontCacheBudget int64 = 48 << 20

type fontCacheEntry struct {
	font *Font
	used uint64
}

var fontCache = struct {
	mu   sync.Mutex
	m    map[fontKey]*fontCacheEntry
	tick uint64
}{m: map[fontKey]*fontCacheEntry{}}

func cachedFont(key fontKey) *Font {
	fontCache.mu.Lock()
	defer fontCache.mu.Unlock()
	e, ok := fontCache.m[key]
	if !ok {
		return nil
	}
	fontCache.tick++
	e.used = fontCache.tick
	return e.font
}

// storeFont publishes f and returns the face callers should use: a
// concurrent bake of the same key wins so every caller shares one sheet.
func storeFont(key fontKey, f *Font) *Font {
	fontCache.mu.Lock()
	defer fontCache.mu.Unlock()
	fontCache.tick++
	if e, ok := fontCache.m[key]; ok {
		e.used = fontCache.tick
		return e.font
	}
	fontCache.m[key] = &fontCacheEntry{font: f, used: fontCache.tick}
	evictFontsLocked(key)
	return f
}

// evictFontsLocked drops least-recently-used faces until the live sheets fit
// the budget. keep is never evicted (it was just requested).
func evictFontsLocked(keep fontKey) {
	total := int64(0)
	for _, e := range fontCache.m {
		total += e.font.atlasBytes()
	}
	for total > fontCacheBudget && len(fontCache.m) > 1 {
		var (
			victim fontKey
			oldest uint64
			found  bool
		)
		for k, e := range fontCache.m {
			if k == keep {
				continue
			}
			if !found || e.used < oldest {
				oldest, victim, found = e.used, k, true
			}
		}
		if !found {
			break
		}
		total -= fontCache.m[victim].font.atlasBytes()
		delete(fontCache.m, victim)
	}
}

// fontCacheStats is the live cache size (entries, glyph sheet bytes).
func fontCacheStats() (entries int, bytes int64) {
	fontCache.mu.Lock()
	defer fontCache.mu.Unlock()
	for _, e := range fontCache.m {
		bytes += e.font.atlasBytes()
	}
	return len(fontCache.m), bytes
}

// BakeFont builds the default UI face (Titillium Web Regular) at size.
// Glyphs are OpenType outlines rasterized through paintengine2d — not the
// 5×7 bitmap atlas. Theme color is applied at draw time via Paint.Color.
func BakeFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyUI, WeightRegular, size, col)
}

// BakeTitleFont is Titillium Web Bold at size.
func BakeTitleFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyUI, WeightBold, size, col)
}

// BakeMonoFont is JetBrains Mono Regular at size (LookAndFeel mono role).
func BakeMonoFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyMono, WeightRegular, size, col)
}

// BakeMonoBoldFont is JetBrains Mono Bold at size.
func BakeMonoBoldFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyMono, WeightBold, size, col)
}

// BakeBitmapFont is the legacy 5×7 nearest-neighbor atlas. Default UI and
// mono faces never use this when the bundled OFL TTFs are present.
func BakeBitmapFont(scale int, col paintengine2d.Color) *Font {
	if scale < 1 {
		scale = 1
	}
	f := bakeScaled(scale, col)
	f.Family = "bitmap"
	f.Outline = false
	return f
}

// BakeFamily rasterizes a bundled OFL face. It panics only if the embedded
// TTF is missing or unloadable — that is a packaging bug, not a runtime
// fallback to the bitmap atlas.
func BakeFamily(family string, weight Weight, size float32, col paintengine2d.Color) *Font {
	if size < 8 {
		size = 8
	}
	key := fontKey{family: family, weight: weight, size: int(size*4 + 0.5)}
	if cached := cachedFont(key); cached != nil {
		f := *cached
		f.Color = col
		return &f
	}
	f, err := bakeOutline(family, weight, size)
	if err != nil {
		panic(err)
	}
	out := *storeFont(key, f)
	out.Color = col
	return &out
}

func bakeOutline(family string, weight Weight, size float32) (*Font, error) {
	face, err := faceFor(family, weight)
	if err != nil {
		return nil, err
	}
	ascent, descent := face.metrics(size)
	ot := newOTAtlas(face, size)
	ot.ascent = ascent
	atlas, err := ot.bake(otPreload)
	if err != nil {
		return nil, err
	}
	cell, ok := atlas.Cell(paintengine2d.GlyphID('A'))
	if !ok || cell.Src.Empty() || (ot.hasNotdef && cell.Src == ot.notdef.Src) {
		return nil, fmt.Errorf("style: %s did not rasterize A", family)
	}
	return &Font{
		Size:    size,
		Ascent:  ascent,
		Descent: descent,
		Family:  family,
		Weight:  weight,
		Outline: true,
		static:  atlas,
		ot:      ot,
		shaped:  newShapeCache(),
	}, nil
}

func bakeScaled(scale int, col paintengine2d.Color) *Font {
	base := paintengine2d.NewBitmapAtlas(col)
	extra := extraPunctAtlas(col)
	// Merge extra cells into a combined source by listing ids.
	ids := make([]paintengine2d.GlyphID, 0, 80)
	seen := map[paintengine2d.GlyphID]bool{}
	for id := range base.Cells {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for id := range extra.Cells {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	srcW, srcH := 6, 8
	dstW, dstH := srcW*scale, srcH*scale
	const cols = 16
	rows := (len(ids) + cols - 1) / cols
	img := paintengine2d.NewImage(cols*dstW, rows*dstH)
	ctx := paintengine2d.NewContext(img)
	cells := make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, len(ids))
	for i, id := range ids {
		cell, ok := base.Cell(id)
		srcImg := base.Image
		if !ok {
			cell, ok = extra.Cell(id)
			srcImg = extra.Image
			if !ok {
				continue
			}
		}
		colI := i % cols
		rowI := i / cols
		dx := float32(colI * dstW)
		dy := float32(rowI * dstH)
		dst := paintengine2d.XYWH(dx, dy, float32(dstW), float32(dstH))
		ctx.DrawImageRectPaint(srcImg, cell.Src, dst, paintengine2d.Paint{
			Color:  paintengine2d.White,
			Filter: paintengine2d.FilterNearest,
		})
		adv := float32(dstW) + 1
		cells[id] = paintengine2d.AtlasCell{
			Src:     dst,
			Advance: adv,
		}
	}
	return &Font{
		static:  &paintengine2d.FontAtlas{Image: img, Cells: cells},
		Size:    float32(dstH),
		Ascent:  float32(7 * scale),
		Descent: float32(scale + 2),
		Color:   col,
		shaped:  newShapeCache(),
	}
}

// extraPunctAtlas is a tiny 5×7 sheet for punctuation the engine atlas omits.
func extraPunctAtlas(fg paintengine2d.Color) *paintengine2d.FontAtlas {
	type bits [7]byte
	extra := map[rune]bits{
		',':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x18},
		';':  {0x00, 0x0C, 0x0C, 0x00, 0x0C, 0x0C, 0x18},
		'\'': {0x0C, 0x0C, 0x08, 0x00, 0x00, 0x00, 0x00},
		'"':  {0x1B, 0x1B, 0x12, 0x00, 0x00, 0x00, 0x00},
		'@':  {0x0E, 0x11, 0x17, 0x15, 0x17, 0x10, 0x0E},
		'#':  {0x0A, 0x1F, 0x0A, 0x0A, 0x1F, 0x0A, 0x00},
		'[':  {0x1C, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1C},
		']':  {0x07, 0x01, 0x01, 0x01, 0x01, 0x01, 0x07},
		'{':  {0x06, 0x08, 0x08, 0x10, 0x08, 0x08, 0x06},
		'}':  {0x0C, 0x02, 0x02, 0x01, 0x02, 0x02, 0x0C},
		'<':  {0x02, 0x04, 0x08, 0x10, 0x08, 0x04, 0x02},
		'>':  {0x08, 0x04, 0x02, 0x01, 0x02, 0x04, 0x08},
		'|':  {0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
		'\\': {0x10, 0x10, 0x08, 0x04, 0x04, 0x02, 0x01},
		'*':  {0x00, 0x15, 0x0E, 0x04, 0x0E, 0x15, 0x00},
		'&':  {0x0C, 0x12, 0x14, 0x08, 0x15, 0x12, 0x0D},
		'`':  {0x08, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
		'~':  {0x00, 0x00, 0x09, 0x16, 0x00, 0x00, 0x00},
	}
	const cw, ch, cols = 6, 8, 8
	n := len(extra)
	rows := (n + cols - 1) / cols
	img := paintengine2d.NewImage(cols*cw, rows*ch)
	cells := make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, n)
	i := 0
	for r, b := range extra {
		col := i % cols
		row := i / cols
		ox, oy := col*cw, row*ch
		for y := 0; y < 7; y++ {
			rowBits := b[y]
			for x := 0; x < 5; x++ {
				if rowBits&(1<<uint(4-x)) != 0 {
					img.SetColor(ox+x, oy+y, fg)
				}
			}
		}
		cells[paintengine2d.GlyphID(r)] = paintengine2d.AtlasCell{
			Src:     paintengine2d.XYWH(float32(ox), float32(oy), cw, ch),
			Advance: cw,
		}
		i++
	}
	return &paintengine2d.FontAtlas{Image: img, Cells: cells}
}

var (
	tintOnce sync.Once
	tintOK   bool
)

// GlyphTint reports whether paintengine2d blits apply RGB Color as a tint
// (v0.7.2+). uitoolkit requires this for themed white atlases.
func GlyphTint() bool {
	tintOnce.Do(func() {
		atlas := paintengine2d.NewBitmapAtlas(paintengine2d.White)
		img := paintengine2d.NewImage(20, 16)
		ctx := paintengine2d.NewContext(img)
		run := paintengine2d.NullShaper{}.Shape("I", atlas)
		ctx.DrawGlyphs(run, paintengine2d.Pt(1, 1), paintengine2d.Paint{
			Color:  paintengine2d.RGB(1, 0, 0),
			Filter: paintengine2d.FilterNearest,
		})
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				r, g, b, a := img.PremulAt(x, y)
				if a > 20 && int(r) > int(g)+24 && int(r) > int(b)+24 {
					tintOK = true
					return
				}
			}
		}
	})
	return tintOK
}
