package style

import (
	"fmt"
	"sync"

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

// otFace is a parsed bundled TTF used to rasterize glyphs into a white atlas.
type otFace struct {
	font *sfnt.Font
	src  []byte
	name string
	mu   sync.Mutex
	buf  sfnt.Buffer
}

type otAtlas struct {
	face   *otFace
	size   float32
	ascent float32
	pad    int
	packer shelfPacker
	mu     sync.Mutex
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
	buf := append([]byte(nil), src...)
	f, err := sfnt.Parse(buf)
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
	switch family {
	case FamilyMono, "jetbrains mono", "mono":
		if w >= WeightBold {
			return monoBold, nil
		}
		return monoReg, nil
	default:
		if w >= WeightBold {
			return uiBold, nil
		}
		return uiReg, nil
	}
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

const otPreload = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~\u2026\u00b7\u2013\u2014\u201c\u201d\u2018\u2019\u20ac"

func newOTAtlas(face *otFace, size float32) *otAtlas {
	if size < 8 {
		size = 8
	}
	ascent, _ := face.metrics(size)
	side := 512
	if size >= 22 {
		side = 768
	}
	a := &otAtlas{
		face:   face,
		size:   size,
		ascent: ascent,
		pad:    1,
		packer: shelfPacker{w: side, h: side, x: 1, y: 1, pad: 1},
	}
	return a
}

func (a *otAtlas) bake(preload string) (*paintengine2d.FontAtlas, error) {
	img := paintengine2d.NewImage(a.packer.w, a.packer.h)
	atlas := &paintengine2d.FontAtlas{
		Image: img,
		Cells: make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, 128),
	}
	if preload == "" {
		preload = otPreload
	}
	for _, r := range preload {
		if _, err := a.rasterize(atlas, r); err != nil {
			return nil, err
		}
	}
	return atlas, nil
}

func (a *otAtlas) rasterize(atlas *paintengine2d.FontAtlas, r rune) (paintengine2d.AtlasCell, error) {
	if atlas.Cells != nil {
		if c, ok := atlas.Cells[paintengine2d.GlyphID(r)]; ok {
			return c, nil
		}
	}
	tf := a.face
	ppem := ppem26(a.size)
	tf.mu.Lock()
	gid, err := tf.font.GlyphIndex(&tf.buf, r)
	if err != nil {
		tf.mu.Unlock()
		return paintengine2d.AtlasCell{}, err
	}
	if gid == 0 && r != ' ' && r != 0 {
		tf.mu.Unlock()
		if p, adv := symbolFallback(r, a.size); p != nil {
			return a.blitPath(atlas, r, p, adv)
		}
		if r > 127 {
			return paintengine2d.AtlasCell{}, nil
		}
		return paintengine2d.AtlasCell{}, fmt.Errorf("missing glyph %q", string(r))
	}
	adv, err := tf.font.GlyphAdvance(&tf.buf, gid, ppem, font.HintingNone)
	if err != nil {
		adv = 0
	}
	segs, err := tf.font.LoadGlyph(&tf.buf, gid, ppem, nil)
	tf.mu.Unlock()
	if err != nil {
		return paintengine2d.AtlasCell{}, err
	}
	advance := fx32(adv)
	if r == ' ' && advance < a.size*0.2 {
		advance = a.size * 0.3
	}
	if len(segs) == 0 {
		return a.blitPath(atlas, r, nil, advance)
	}
	return a.blitPath(atlas, r, segmentsPath(segs), advance)
}

// blitPath rasterizes p into the white atlas. Font space is baseline origin,
// Y down (same as sfnt). Engine DrawPath does the AA; we only pack the sheet.
func (a *otAtlas) blitPath(atlas *paintengine2d.FontAtlas, r rune, p *paintengine2d.Path, advance float32) (paintengine2d.AtlasCell, error) {
	if p == nil || p.Empty() {
		cell := paintengine2d.AtlasCell{Advance: advance}
		atlas.Cells[paintengine2d.GlyphID(r)] = cell
		return cell, nil
	}
	b := p.Bounds()
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
	tmp := paintengine2d.NewImage(gw, gh)
	ctx := paintengine2d.NewContext(tmp)
	ctx.Translate(-xmin+pad, -ymin+pad)
	ctx.DrawPath(p, paintengine2d.Fill(paintengine2d.White))

	ax, ay, ok := a.pack(atlas, gw, gh)
	if !ok {
		return paintengine2d.AtlasCell{}, fmt.Errorf("atlas full for %q", string(r))
	}
	blitGlyph(atlas.Image, ax, ay, tmp)
	atlas.Image.TouchRect(paintengine2d.XYWH(float32(ax), float32(ay), float32(gw), float32(gh)))
	cell := paintengine2d.AtlasCell{
		Src:     paintengine2d.XYWH(float32(ax), float32(ay), float32(gw), float32(gh)),
		Bearing: paintengine2d.Pt(xmin-pad, a.ascent+ymin-pad),
		Advance: advance,
	}
	atlas.Cells[paintengine2d.GlyphID(r)] = cell
	return cell, nil
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

type shelfPacker struct {
	w, h, x, y, rowH, pad int
}

func (a *otAtlas) pack(atlas *paintengine2d.FontAtlas, gw, gh int) (x, y int, ok bool) {
	for try := 0; try < 6; try++ {
		x, y, ok = a.packer.add(gw, gh)
		if ok {
			return x, y, true
		}
		a.grow(atlas)
	}
	return 0, 0, false
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

func (a *otAtlas) grow(atlas *paintengine2d.FontAtlas) {
	old := atlas.Image
	nw, nh := old.Width, old.Height*2
	if nw < 64 {
		nw = 64
	}
	img := paintengine2d.NewImage(nw, nh)
	for y := 0; y < old.Height; y++ {
		for x := 0; x < old.Width; x++ {
			r, g, b, al := old.PremulAt(x, y)
			img.SetColor(x, y, paintengine2d.FromPremul8(r, g, b, al))
		}
	}
	atlas.Image = img
	a.packer.h = nh
	a.packer.w = nw
}
