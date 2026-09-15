package style

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/fonts"
)

// Weight selects Regular or Bold of a bundled family.
type Weight int

const (
	WeightRegular Weight = 400
	WeightBold    Weight = 700
)

const (
	FamilyUI   = fonts.FamilyUI
	FamilyMono = fonts.FamilyMono
	// DefaultFontFamily is the LookAndFeel UI face (Titillium Web).
	DefaultFontFamily = FamilyUI
	// DefaultMonoFamily is the LookAndFeel mono / code face (JetBrains Mono).
	// mononoki is not a default role — optional later theme only.
	DefaultMonoFamily = FamilyMono
)

// FontRole is a LookAndFeel typeface slot.
type FontRole int

const (
	// RoleUI is labels, buttons, menus, and proportional fields (Titillium Web).
	RoleUI FontRole = iota
	// RoleMono is code, Inspector, logs, and monospace fields (JetBrains Mono).
	RoleMono
)

// FamilyFor returns the locked family name for a role.
func FamilyFor(role FontRole) string {
	if role == RoleMono {
		return FamilyMono
	}
	return FamilyUI
}

// FontFor returns Classic's UI or mono face. Other looks fall back to Font().
func FontFor(l LookAndFeel, role FontRole) *Font {
	if l == nil {
		return nil
	}
	if role == RoleMono {
		return l.MonoFont()
	}
	return l.Font()
}

// otFace is a parsed TTF / OTF used to rasterize glyphs into a white atlas.
type otFace struct {
	font *sfnt.Font
	src  []byte
	name string
	mu   sync.Mutex
	buf  sfnt.Buffer
	// embolden draws the outline heavier by this fraction of the size: a
	// bold synthesized from a regular face (a variable font's bold instance,
	// which sfnt cannot draw).
	embolden float32
}

var (
	facesOnce sync.Once
	facesErr  error
	uiReg     *otFace
	uiBold    *otFace
	monoReg   *otFace
	monoBold  *otFace
)

// LoadEmbeddedFonts parses Titillium Web and JetBrains Mono from the module.
// Tests must call this (or BakeFont) and fail if it returns an error.
func LoadEmbeddedFonts() error {
	facesOnce.Do(func() {
		var err error
		if uiReg, err = parseFace(fonts.FileTitilliumRegular, fonts.TitilliumRegular); err != nil {
			facesErr = err
			return
		}
		if uiBold, err = parseFace(fonts.FileTitilliumBold, fonts.TitilliumBold); err != nil {
			facesErr = err
			return
		}
		if monoReg, err = parseFace(fonts.FileJetBrainsRegular, fonts.JetBrainsRegular); err != nil {
			facesErr = err
			return
		}
		if monoBold, err = parseFace(fonts.FileJetBrainsBold, fonts.JetBrainsBold); err != nil {
			facesErr = err
			return
		}
		if uiReg.name == "" {
			uiReg.name = FamilyUI
		}
		if monoReg.name == "" {
			monoReg.name = FamilyMono
		}
	})
	return facesErr
}

func parseFace(name string, load func() ([]byte, error)) (*otFace, error) {
	src, err := load()
	if err != nil {
		return nil, err
	}
	// Copy: sfnt keeps a view of the bytes.
	return parseFaceBytes(name, append([]byte(nil), src...), 0)
}

// parseFaceBytes parses face index of buf (a single font or a collection).
// sfnt keeps a view of buf.
func parseFaceBytes(name string, buf []byte, index int) (*otFace, error) {
	var f *sfnt.Font
	var err error
	if len(buf) >= 4 && string(buf[:4]) == "ttcf" {
		var c *sfnt.Collection
		if c, err = sfnt.ParseCollection(buf); err == nil {
			f, err = c.Font(index)
		}
	} else {
		f, err = sfnt.Parse(buf)
	}
	if err != nil {
		return nil, fmt.Errorf("style: parse %s: %w", name, err)
	}
	tf := &otFace{font: f, src: buf}
	var b sfnt.Buffer
	if n, err := f.Name(&b, sfnt.NameIDFamily); err == nil {
		tf.name = n
	}
	return tf, nil
}

func faceFor(family string, w Weight) (*otFace, error) {
	if err := LoadEmbeddedFonts(); err != nil {
		return nil, err
	}
	switch bundledFamily(family) {
	case FamilyMono:
		if w >= WeightBold {
			return monoBold, nil
		}
		return monoReg, nil
	case FamilyUI:
	default:
		// An installed face (a pack's era font); not installed, the
		// bundled UI face stands in.
		if f := systemOTFace(family, w); f != nil {
			return f, nil
		}
	}
	if w >= WeightBold {
		return uiBold, nil
	}
	return uiReg, nil
}

// kern is the face's pair kerning (GPOS or kern table) for r0 then r1 at
// size, in pixels; 0 when the face has none for the pair.
func (f *otFace) kern(r0, r1 rune, size float32) float32 {
	if f == nil || f.font == nil {
		return 0
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	g0, err := f.font.GlyphIndex(&f.buf, r0)
	if err != nil || g0 == 0 {
		return 0
	}
	g1, err := f.font.GlyphIndex(&f.buf, r1)
	if err != nil || g1 == 0 {
		return 0
	}
	k, err := f.font.Kern(&f.buf, g0, g1, ppem26(size), font.HintingNone)
	if err != nil {
		return 0
	}
	return fx32(k)
}

func (f *otFace) metrics(size float32) (ascent, descent float32) {
	if f == nil || f.font == nil {
		return size * 0.8, size * 0.2
	}
	if size < 1 {
		size = 1
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.font.Metrics(&f.buf, ppem26(size), font.HintingNone)
	if err != nil {
		return size * 0.8, size * 0.2
	}
	return float32(m.Ascent) / 64, float32(m.Descent) / 64
}

func ppem26(size float32) fixed.Int26_6 {
	if size < 1 {
		size = 1
	}
	return fixed.Int26_6(size * 64)
}

func fx32(v fixed.Int26_6) float32 { return float32(v) / 64 }

const otPreload = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~…·–—“”‘’€"

const (
	// atlasSmallSide / atlasBigSide are the first sheet allocated for a size.
	atlasSmallSide = 512
	atlasBigSide   = 768
	// atlasMaxSide caps one (family, weight, size) sheet at 2048² = 16 MB.
	// A full sheet still measures text: further runes reuse the .notdef
	// advance, they just stop adding pixels.
	atlasMaxSide = 2048
)

// otAtlas bakes glyphs for one (face, size) into a copy-on-write
// [paintengine2d.FontAtlas].
//
// Readers (measure, shape, paint) take the published snapshot through
// atlas() and never lock: a published sheet is immutable. A writer clones
// the sheet (fresh Cells map and image), rasterizes the missing runes into
// the private draft, and stores the pointer atomically — so a paint loop in
// one goroutine can blit while another measures a rune that is not baked yet.
type otAtlas struct {
	face   *otFace
	size   float32
	ascent float32
	pad    int

	// mu serializes writers (draft + publish) only.
	mu        sync.Mutex
	packer    shelfPacker
	notdef    paintengine2d.AtlasCell
	hasNotdef bool
	full      bool

	cur   atomic.Pointer[paintengine2d.FontAtlas]
	bytes atomic.Int64

	// kerns caches pair kerning at this size (rune pair → pixels).
	kernMu sync.Mutex
	kerns  map[uint64]float32
}

// kernCap bounds the pair-kerning cache of one face and size.
const kernCap = 4096

// kern is the pair kerning between r0 and r1 at the atlas's size, in
// pixels (negative pulls them together).
func (a *otAtlas) kern(r0, r1 rune) float32 {
	if a == nil || a.face == nil {
		return 0
	}
	key := uint64(uint32(r0))<<32 | uint64(uint32(r1))
	a.kernMu.Lock()
	defer a.kernMu.Unlock()
	if v, ok := a.kerns[key]; ok {
		return v
	}
	v := a.face.kern(r0, r1, a.size)
	if a.kerns == nil || len(a.kerns) >= kernCap {
		a.kerns = make(map[uint64]float32, 64)
	}
	a.kerns[key] = v
	return v
}

func newOTAtlas(face *otFace, size float32) *otAtlas {
	if size < 8 {
		size = 8
	}
	ascent, _ := face.metrics(size)
	side := atlasSmallSide
	if size >= 22 {
		side = atlasBigSide
	}
	return &otAtlas{
		face:   face,
		size:   size,
		ascent: ascent,
		pad:    1,
		packer: shelfPacker{w: side, h: side, x: 1, y: 1, pad: 1},
	}
}

// atlas is the published snapshot. Never mutated after publication.
func (a *otAtlas) atlas() *paintengine2d.FontAtlas {
	if a == nil {
		return nil
	}
	return a.cur.Load()
}

// atlasBytes is the live sheet size, used by the font cache budget.
func (a *otAtlas) atlasBytes() int64 {
	if a == nil {
		return 0
	}
	return a.bytes.Load()
}

// atlasDraft is a private, writable copy of the sheet.
type atlasDraft struct {
	a      *otAtlas
	img    *paintengine2d.Image
	cells  map[paintengine2d.GlyphID]paintengine2d.AtlasCell
	packer shelfPacker
	full   bool
}

// draft clones from (nil for the first bake). Caller holds a.mu.
func (a *otAtlas) draft(from *paintengine2d.FontAtlas) *atlasDraft {
	d := &atlasDraft{a: a, packer: a.packer, full: a.full}
	if from == nil || from.Image == nil {
		d.img = paintengine2d.NewImage(a.packer.w, a.packer.h)
		d.cells = make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, 256)
		return d
	}
	d.img = cloneImage(from.Image, from.Image.Width, from.Image.Height)
	d.cells = make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, len(from.Cells)+8)
	for k, v := range from.Cells {
		d.cells[k] = v
	}
	return d
}

// publish swaps the draft in as the new snapshot. Caller holds a.mu.
func (a *otAtlas) publish(d *atlasDraft) *paintengine2d.FontAtlas {
	next := &paintengine2d.FontAtlas{Image: d.img, Cells: d.cells}
	a.packer = d.packer
	a.full = d.full
	a.cur.Store(next)
	a.bytes.Store(int64(d.img.Width) * int64(d.img.Height) * 4)
	return next
}

func (a *otAtlas) bake(preload string) (*paintengine2d.FontAtlas, error) {
	if preload == "" {
		preload = otPreload
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	d := a.draft(nil)
	for _, r := range preload {
		if err := d.rasterize(r); err != nil {
			return nil, err
		}
	}
	return a.publish(d), nil
}

func atlasHas(cur *paintengine2d.FontAtlas, r rune) bool {
	if cur == nil {
		return false
	}
	_, ok := cur.Cells[paintengine2d.GlyphID(r)]
	return ok
}

func atlasHasAll(cur *paintengine2d.FontAtlas, text string) bool {
	if cur == nil {
		return false
	}
	for _, r := range text {
		if _, ok := cur.Cells[paintengine2d.GlyphID(r)]; !ok {
			return false
		}
	}
	return true
}

// ensure bakes every rune of text the published sheet lacks and republishes
// once for the whole string. Returns the snapshot callers should shape with.
func (a *otAtlas) ensure(text string) *paintengine2d.FontAtlas {
	if a == nil {
		return nil
	}
	cur := a.atlas()
	if text == "" || atlasHasAll(cur, text) {
		return cur
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	cur = a.atlas()
	if atlasHasAll(cur, text) {
		return cur
	}
	d := a.draft(cur)
	for _, r := range text {
		if _, ok := d.cells[paintengine2d.GlyphID(r)]; ok {
			continue
		}
		// A rune that cannot be rasterized still gets a .notdef cell,
		// so it is never re-attempted on the next measure.
		_ = d.rasterize(r)
	}
	return a.publish(d)
}

// ensureRune is ensure for a single rune (no string allocation).
func (a *otAtlas) ensureRune(r rune) *paintengine2d.FontAtlas {
	if a == nil {
		return nil
	}
	cur := a.atlas()
	if atlasHas(cur, r) {
		return cur
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	cur = a.atlas()
	if atlasHas(cur, r) {
		return cur
	}
	d := a.draft(cur)
	_ = d.rasterize(r)
	return a.publish(d)
}

func (d *atlasDraft) rasterize(r rune) error {
	id := paintengine2d.GlyphID(r)
	if _, ok := d.cells[id]; ok {
		return nil
	}
	a := d.a
	tf := a.face
	ppem := ppem26(a.size)
	tf.mu.Lock()
	gid, err := tf.font.GlyphIndex(&tf.buf, r)
	if err != nil {
		tf.mu.Unlock()
		return err
	}
	if gid == 0 && r != ' ' && r != 0 {
		tf.mu.Unlock()
		if p, adv := symbolFallback(r, a.size); p != nil {
			return d.blitPath(r, p, adv)
		}
		// Unsupported rune: a .notdef box keeps the run measurable
		// (non-zero advance) and visible instead of silently empty.
		return d.storeNotdef(r)
	}
	adv, err := tf.font.GlyphAdvance(&tf.buf, gid, ppem, font.HintingNone)
	if err != nil {
		adv = 0
	}
	segs, err := tf.font.LoadGlyph(&tf.buf, gid, ppem, nil)
	tf.mu.Unlock()
	if err != nil {
		return d.storeNotdef(r)
	}
	advance := fx32(adv)
	if r == ' ' && advance < a.size*0.2 {
		advance = a.size * 0.3
	}
	if tf.embolden > 0 {
		advance += tf.embolden * a.size
	}
	if len(segs) == 0 {
		return d.blitPath(r, nil, advance)
	}
	return d.blitPath(r, segmentsPath(segs), advance)
}

// storeNotdef points r at the sheet's shared .notdef cell.
func (d *atlasDraft) storeNotdef(r rune) error {
	cell, err := d.notdefCell()
	if err != nil {
		return err
	}
	d.cells[paintengine2d.GlyphID(r)] = cell
	return nil
}

// notdefCell rasterizes the face's own .notdef glyph once per sheet (a drawn
// box when the face has no outline for gid 0) and reuses those pixels for
// every unsupported rune. Caller holds a.mu.
func (d *atlasDraft) notdefCell() (paintengine2d.AtlasCell, error) {
	a := d.a
	if a.hasNotdef {
		return a.notdef, nil
	}
	tf := a.face
	ppem := ppem26(a.size)
	var (
		advance float32
		p       *paintengine2d.Path
	)
	tf.mu.Lock()
	if adv, err := tf.font.GlyphAdvance(&tf.buf, 0, ppem, font.HintingNone); err == nil {
		advance = fx32(adv)
	}
	segs, err := tf.font.LoadGlyph(&tf.buf, 0, ppem, nil)
	tf.mu.Unlock()
	if err == nil && len(segs) > 0 {
		p = segmentsPath(segs)
	}
	if p == nil || p.Empty() {
		p, advance = notdefBox(a.size)
	}
	if advance <= 0 {
		advance = a.size * 0.55
	}
	cell, err := d.bakeCell(p, advance)
	if err != nil {
		return paintengine2d.AtlasCell{}, err
	}
	a.notdef = cell
	a.hasNotdef = true
	d.cells[paintengine2d.GlyphID(0)] = cell
	return cell, nil
}

// notdefBox is the drawn fallback box (four bars so any fill rule leaves it
// hollow). Font space is baseline origin, Y down.
func notdefBox(size float32) (*paintengine2d.Path, float32) {
	adv := size * 0.55
	w := adv - size*0.16
	if w < 2 {
		w = 2
	}
	h := size * 0.64
	x := size * 0.08
	th := size * 0.07
	if th < 1 {
		th = 1
	}
	if th > h/3 {
		th = h / 3
	}
	p := paintengine2d.NewPath()
	rect := func(rx, ry, rw, rh float32) {
		p.MoveTo(rx, ry)
		p.LineTo(rx+rw, ry)
		p.LineTo(rx+rw, ry+rh)
		p.LineTo(rx, ry+rh)
		p.Close()
	}
	rect(x, -h, w, th)      // top
	rect(x, -th, w, th)     // bottom
	rect(x, -h, th, h)      // left
	rect(x+w-th, -h, th, h) // right
	return p, adv
}

// blitPath rasterizes p into the white sheet and records the cell for r.
func (d *atlasDraft) blitPath(r rune, p *paintengine2d.Path, advance float32) error {
	id := paintengine2d.GlyphID(r)
	cell, err := d.bakeCell(p, advance)
	if err != nil {
		return err
	}
	d.cells[id] = cell
	return nil
}

// bakeCell packs p (may be nil / empty) and returns its atlas cell. Font
// space is baseline origin, Y down (same as sfnt). Engine DrawPath does the
// AA; we only pack the sheet.
func (d *atlasDraft) bakeCell(p *paintengine2d.Path, advance float32) (paintengine2d.AtlasCell, error) {
	a := d.a
	if p == nil || p.Empty() {
		return paintengine2d.AtlasCell{Advance: advance}, nil
	}
	b := p.Bounds()
	heavy := a.face.embolden * a.size
	if heavy > 0 {
		b = b.Inset(-heavy * 0.5)
	}
	xmin, ymin, xmax, ymax := b.Min.X, b.Min.Y, b.Max.X, b.Max.Y
	if xmax <= xmin {
		xmax = xmin + 1
	}
	if ymax <= ymin {
		ymax = ymin + 1
	}
	pad := float32(a.pad)
	gw := int(xmax-xmin+2*pad+1.5) + 1
	gh := int(ymax-ymin+2*pad+1.5) + 1
	if gw < 1 {
		gw = 1
	}
	if gh < 1 {
		gh = 1
	}
	ax, ay, ok := d.pack(gw, gh)
	if !ok {
		// Sheet is at its cap: keep the advance so layout still measures.
		d.full = true
		return paintengine2d.AtlasCell{Advance: advance}, nil
	}
	tmp := paintengine2d.NewImage(gw, gh)
	ctx := paintengine2d.NewContext(tmp)
	ctx.Translate(-xmin+pad, -ymin+pad)
	ink := paintengine2d.Fill(paintengine2d.White)
	if heavy > 0 {
		// Synthesized bold: the outline stroked round as well as filled.
		ink.Style = paintengine2d.StyleStrokeAndFill
		ink.Stroke = paintengine2d.Stroke{Width: heavy, Join: paintengine2d.JoinRound}
	}
	ctx.DrawPath(p, ink)
	blitGlyph(d.img, ax, ay, tmp)
	d.img.Bump()
	return paintengine2d.AtlasCell{
		Src:     paintengine2d.XYWH(float32(ax), float32(ay), float32(gw), float32(gh)),
		Bearing: paintengine2d.Pt(xmin-pad, a.ascent+ymin-pad),
		Advance: advance,
	}, nil
}

func segmentsPath(segs sfnt.Segments) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	for _, seg := range segs {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			p.MoveTo(fx32(seg.Args[0].X), fx32(seg.Args[0].Y))
		case sfnt.SegmentOpLineTo:
			p.LineTo(fx32(seg.Args[0].X), fx32(seg.Args[0].Y))
		case sfnt.SegmentOpQuadTo:
			p.QuadTo(
				fx32(seg.Args[0].X), fx32(seg.Args[0].Y),
				fx32(seg.Args[1].X), fx32(seg.Args[1].Y),
			)
		case sfnt.SegmentOpCubeTo:
			p.CubicTo(
				fx32(seg.Args[0].X), fx32(seg.Args[0].Y),
				fx32(seg.Args[1].X), fx32(seg.Args[1].Y),
				fx32(seg.Args[2].X), fx32(seg.Args[2].Y),
			)
		}
	}
	return p
}

func blitGlyph(dst *paintengine2d.Image, dx, dy int, src *paintengine2d.Image) {
	if dst == nil || src == nil {
		return
	}
	for y := 0; y < src.Height; y++ {
		sy := dy + y
		if sy < 0 || sy >= dst.Height {
			continue
		}
		for x := 0; x < src.Width; x++ {
			sx := dx + x
			if sx < 0 || sx >= dst.Width {
				continue
			}
			r, g, b, a := src.PremulAt(x, y)
			dst.SetColor(sx, sy, paintengine2d.FromPremul8(r, g, b, a))
		}
	}
}

// cloneImage copies src into a fresh w×h pixmap with row-slice copies.
// Growing keeps every packed cell at the same coordinates.
func cloneImage(src *paintengine2d.Image, w, h int) *paintengine2d.Image {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := paintengine2d.NewImage(w, h)
	if src == nil || src.Width == 0 || src.Height == 0 {
		return dst
	}
	rows := src.Height
	if h < rows {
		rows = h
	}
	ss, ds := src.RowStride(), dst.RowStride()
	n := src.Width * 4
	if m := dst.Width * 4; m < n {
		n = m
	}
	for y := 0; y < rows; y++ {
		copy(dst.Pix[y*ds:y*ds+n], src.Pix[y*ss:y*ss+n])
	}
	dst.Touch()
	return dst
}

type shelfPacker struct {
	w, h, x, y, rowH, pad int
}

// pack finds room for gw×gh, growing the draft sheet as needed.
func (d *atlasDraft) pack(gw, gh int) (x, y int, ok bool) {
	for {
		if x, y, ok = d.packer.add(gw, gh); ok {
			return x, y, true
		}
		if !d.grow() {
			return 0, 0, false
		}
	}
}

func (p *shelfPacker) add(gw, gh int) (x, y int, ok bool) {
	pad := p.pad
	if pad < 1 {
		pad = 1
	}
	if gw+pad > p.w || gh+pad > p.h {
		return 0, 0, false
	}
	if p.x+gw+pad > p.w {
		p.y += p.rowH
		p.x = pad
		p.rowH = 0
	}
	if p.y+gh+pad > p.h {
		return 0, 0, false
	}
	x, y = p.x, p.y
	p.x += gw + pad
	if gh+pad > p.rowH {
		p.rowH = gh + pad
	}
	return x, y, true
}

// grow doubles the shorter side (width first) up to atlasMaxSide. Cells keep
// their coordinates, so only the packer bounds change.
func (d *atlasDraft) grow() bool {
	w, h := d.img.Width, d.img.Height
	switch {
	case w <= h && w*2 <= atlasMaxSide:
		w *= 2
	case h*2 <= atlasMaxSide:
		h *= 2
	case w*2 <= atlasMaxSide:
		w *= 2
	default:
		return false
	}
	d.img = cloneImage(d.img, w, h)
	d.packer.w, d.packer.h = w, h
	return true
}

func snapCoord(v float32) float32 { return float32(math.Round(float64(v))) }
