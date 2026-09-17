package style

import (
	"bytes"
	"io/fs"
	"math"
	"sync"
	"time"

	"github.com/codemodify/paintengine2d"
)

// A skin's art, from the file on disk to the nine pieces a button is blitted
// from. Three things happen here and each is a quality decision, not an
// optimisation:
//
//   - **Sheets are chosen upward.** The asset used is the smallest one at or
//     above the target scale, drawn down. Art is never enlarged. This is the
//     same rule style/iconset.go applies to @2x icons, and it is why a skin
//     looks right at 1.75 instead of soft.
//   - **Every piece is cut into an image of its own.** paintengine2d's
//     bilinear sampler reads the two texels around each sample and returns
//     transparent outside the *image* — not outside the source rect — so a
//     sub-rect blit of a sheet bleeds its neighbour along every seam. Cutting
//     each sprite (and each of a nine-slice's nine pieces) into its own image
//     removes that class of artefact entirely, and costs a few hundred small
//     images per skin.
//   - **Downscale only, so edges stay put.** With the asset at or above the
//     target the source step is ≥ 1 texel per device pixel, so the first and
//     last samples land inside the art and the edge does not fade. Upscaling
//     would sample outside it and halo; the format's validation warns about
//     a skin with no asset at or above the display's scale rather than
//     quietly enlarging one.

// skinAssetTTL is how long a cached file stat is trusted, matching
// iconCacheTTL: within a frame nothing is re-stat'ed, and a re-saved PNG
// shows up about a second later without a restart.
const skinAssetTTL = iconCacheTTL

// skinPiece names one of the nine parts of a sliced sprite, in reading
// order. An unsliced sprite is the single piece pieceWhole.
type skinPiece uint8

const (
	pieceTopLeft skinPiece = iota
	pieceTop
	pieceTopRight
	pieceLeft
	pieceMiddle
	pieceRight
	pieceBottomLeft
	pieceBottom
	pieceBottomRight
	pieceWhole
	pieceCount
)

// skinVariantKey identifies one rendering of one sprite: the sprite itself
// and the sheet scale it was cut from.
type skinVariantKey struct {
	sprite *SkinSprite
	scale  float32
}

// skinVariant is a sprite cut at one sheet scale, ready to blit.
type skinVariant struct {
	// scale is the sheet scale the pieces came from, in design pixels per
	// piece pixel: a 2× sheet of art drawn on a 1× grid gives 2.
	scale float32
	// pieces are the nine (or one) images. A piece with no pixels is nil.
	pieces [pieceCount]*paintengine2d.Image
	// slice is the sprite's insets in *piece* pixels (design × scale), so
	// the painter never re-derives them.
	slice  Insets
	ok     bool
	warned bool
}

// skinAssets caches decoded sheets and cut sprites for every loaded skin.
// It follows the icon cache's generation so the existing look watcher —
// which already bumps it on every reload — drops a skin's art too, and
// stats files on the same TTL so an edited PNG applies without a restart.
type skinAssets struct {
	mu    sync.Mutex
	gen   uint64
	files map[string]*skinFileEntry
	cuts  map[skinVariantKey]*skinVariant
}

type skinFileEntry struct {
	img     *paintengine2d.Image
	mod     int64
	size    int64
	checked int64 // unix nanos
}

var skinCache = &skinAssets{
	files: map[string]*skinFileEntry{},
	cuts:  map[skinVariantKey]*skinVariant{},
}

// sync drops everything when the process-wide asset generation has moved
// (InvalidateIconCache, which the look watcher calls on every reload).
// Callers hold c.mu.
func (c *skinAssets) syncGen() {
	if gen := IconGeneration(); gen != c.gen {
		c.gen = gen
		c.files = map[string]*skinFileEntry{}
		c.cuts = map[skinVariantKey]*skinVariant{}
	}
}

// InvalidateSkinCache drops every decoded sheet and cut sprite. The look
// watcher reaches it through InvalidateIconCache; tests and the lint command
// call it directly.
func InvalidateSkinCache() {
	skinCache.mu.Lock()
	skinCache.files = map[string]*skinFileEntry{}
	skinCache.cuts = map[skinVariantKey]*skinVariant{}
	skinCache.gen = IconGeneration()
	skinCache.mu.Unlock()
	invalidateSkinPacks()
}

// sheetImage decodes one file of a sheet, caching hits and misses.
//
// The cache key is the skin's id *and* its directory plus the file. Both
// halves are load-bearing: every skin names its main sheet art/chrome.png,
// and a built-in skin has no directory at all, so keying on the path alone
// makes one embedded skin paint from another's sheet.
func (sk *Skin) sheetImage(file string) *paintengine2d.Image {
	if sk == nil || sk.fsys == nil || file == "" {
		return nil
	}
	key := sk.Name + "\x00" + sk.Dir + "\x00" + file
	c := skinCache
	c.mu.Lock()
	defer c.mu.Unlock()
	c.syncGen()

	now := time.Now().UnixNano()
	e, ok := c.files[key]
	if ok && now-e.checked < int64(skinAssetTTL) {
		return e.img
	}
	// An embedded skin's files never change; anything on disk is re-stat'ed
	// on the TTL so a re-saved sheet applies live.
	if st, err := fs.Stat(sk.fsys, file); err == nil {
		mod, size := st.ModTime().UnixNano(), st.Size()
		if ok && e.mod == mod && e.size == size {
			e.checked = now
			return e.img
		}
		raw, err := fs.ReadFile(sk.fsys, file)
		if err != nil {
			c.files[key] = &skinFileEntry{mod: mod, size: size, checked: now}
			return nil
		}
		img, err := paintengine2d.DecodePNG(bytes.NewReader(raw))
		if err != nil || img == nil || img.Width < 1 || img.Height < 1 {
			c.files[key] = &skinFileEntry{mod: mod, size: size, checked: now}
			return nil
		}
		// A sheet whose pixels changed invalidates every cut taken from it.
		// Sheets are compared by identity, not by file name, so re-saving
		// one skin's art/chrome.png does not throw away another's.
		if ok && e.img != nil {
			for k := range c.cuts {
				if k.sprite != nil && sk.owns(k.sprite.Sheet, file) {
					delete(c.cuts, k)
				}
			}
		}
		c.files[key] = &skinFileEntry{img: img, mod: mod, size: size, checked: now}
		return img
	}
	c.files[key] = &skinFileEntry{checked: now}
	return nil
}

// owns reports whether sh is this skin's own sheet and reads file. Identity,
// not name: every skin calls its main sheet "chrome".
func (sk *Skin) owns(sh *SkinSheet, file string) bool {
	if sk == nil || sh == nil {
		return false
	}
	if sk.Sheets[sh.Name] != sh {
		return false
	}
	for _, f := range sh.Files {
		if f == file {
			return true
		}
	}
	return false
}

// pickScale is the sheet's asset for a target scale: the smallest one at or
// above it, else the largest there is. Art is drawn down, never up.
func (sh *SkinSheet) pickScale(target float32) (float32, string, bool) {
	if sh == nil || len(sh.scales) == 0 {
		return 0, "", false
	}
	if target <= 0 {
		target = 1
	}
	for _, s := range sh.scales {
		if s >= target-0.001 {
			return s, sh.Files[s], true
		}
	}
	s := sh.scales[len(sh.scales)-1]
	return s, sh.Files[s], true
}

// Covers reports whether the sheet has an asset at or above scale — what
// validation warns about, and what a skin author needs to know before
// shipping.
func (sh *SkinSheet) Covers(scale float32) bool {
	if sh == nil || len(sh.scales) == 0 {
		return false
	}
	return sh.scales[len(sh.scales)-1] >= scale-0.001
}

// Scales lists the scales the sheet was drawn at, ascending.
func (sh *SkinSheet) Scales() []float32 { return append([]float32(nil), sh.scales...) }

// variant is the sprite cut for a target scale, built once and cached.
func (sk *Skin) variant(sp *SkinSprite, target float32) *skinVariant {
	if sk == nil || sp == nil || sp.Sheet == nil {
		return nil
	}
	assetScale, file, ok := sp.Sheet.pickScale(target)
	if !ok {
		return nil
	}
	key := skinVariantKey{sprite: sp, scale: assetScale}

	skinCache.mu.Lock()
	skinCache.syncGen()
	if v, ok := skinCache.cuts[key]; ok {
		skinCache.mu.Unlock()
		if v.ok {
			return v
		}
		return nil
	}
	skinCache.mu.Unlock()

	// Decoding and cutting happen outside the lock: two goroutines racing
	// on a first paint may both cut, and the first stored wins — the cut is
	// a pure function of the sheet, so either is the same picture.
	v := sk.cut(sp, assetScale, file)

	skinCache.mu.Lock()
	if existing, ok := skinCache.cuts[key]; ok {
		skinCache.mu.Unlock()
		if existing.ok {
			return existing
		}
		return nil
	}
	skinCache.cuts[key] = v
	skinCache.mu.Unlock()
	if v.ok {
		return v
	}
	return nil
}

// cut slices one sprite out of its sheet at assetScale, into one image per
// nine-slice piece. Rects are rounded to whole texels: art is drawn on a
// grid and a half-texel cut is a blurred corner.
func (sk *Skin) cut(sp *SkinSprite, assetScale float32, file string) *skinVariant {
	v := &skinVariant{scale: assetScale}
	sheet := sk.sheetImage(file)
	if sheet == nil {
		return v
	}
	// The manifest's numbers are design pixels on the skin's own grid; the
	// asset is drawn at assetScale on that same grid.
	k := assetScale / sk.Design.Scale
	x0 := roundTexel(sp.X * k)
	y0 := roundTexel(sp.Y * k)
	x1 := roundTexel((sp.X + sp.W) * k)
	y1 := roundTexel((sp.Y + sp.H) * k)
	if x1 <= x0 || y1 <= y0 || x0 < 0 || y0 < 0 || x1 > sheet.Width || y1 > sheet.Height {
		return v
	}
	if !sp.Sliced() {
		v.pieces[pieceWhole] = sheet.SubImage(x0, y0, x1, y1)
		v.ok = v.pieces[pieceWhole] != nil
		return v
	}
	l := roundTexel(sp.Slice.Left * k)
	r := roundTexel(sp.Slice.Right * k)
	t := roundTexel(sp.Slice.Top * k)
	b := roundTexel(sp.Slice.Bottom * k)
	w, h := x1-x0, y1-y0
	// A slice that rounds to the whole sprite would leave no middle; give
	// the middle its texel back rather than refusing to draw.
	if l+r >= w {
		l, r = max(0, (w-1)/2), max(0, w-1-(w-1)/2)
	}
	if t+b >= h {
		t, b = max(0, (h-1)/2), max(0, h-1-(h-1)/2)
	}
	v.slice = Insets{Top: float32(t), Right: float32(r), Bottom: float32(b), Left: float32(l)}

	cols := [4]int{x0, x0 + l, x1 - r, x1}
	rows := [4]int{y0, y0 + t, y1 - b, y1}
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cx0, cx1 := cols[col], cols[col+1]
			cy0, cy1 := rows[row], rows[row+1]
			if cx1 <= cx0 || cy1 <= cy0 {
				continue
			}
			v.pieces[skinPiece(row*3+col)] = sheet.SubImage(cx0, cy0, cx1, cy1)
		}
	}
	for _, p := range v.pieces {
		if p != nil {
			v.ok = true
			break
		}
	}
	return v
}

// roundTexel snaps a design coordinate to a whole texel.
func roundTexel(v float32) int { return int(math.Round(float64(v))) }
