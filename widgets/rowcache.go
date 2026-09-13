package widgets

import (
	"math"
	"sync"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

var rowKeepPool = sync.Pool{
	New: func() any { return make(map[uint64]struct{}, 32) },
}

// rowCacheCap drops off-screen row groups once the map grows past this.
const rowCacheCap = 256

// rowSceneCache keeps recorded virtualized rows so scroll can change a
// parent transform without re-recording every visible row (Qt Quick ListView).
//
// Rows are recorded in row-local space (each row group carries its own
// content offset as a transform) inside a band group whose transform is the
// scroll offset and whose group clip is the viewport. The scroll offset is
// therefore not part of the cache key: scrolling re-uses every row that was
// already recorded and only records the rows that just came into view.
type rowSceneCache struct {
	nodes        map[uint64]*paintengine2d.GroupNode
	sig          map[uint64]uint64
	ox, oy, w, h float32
	look         uint64
}

// ready invalidates the cache when anything that is baked into a recorded
// row changes: the widget's device origin (rows carry their clip from where
// they were recorded), the row width and height, or the theme.
func (c *rowSceneCache) ready(ox, oy, w, h float32, look uint64) {
	if c.nodes == nil || c.ox != ox || c.oy != oy || c.w != w || c.h != h || c.look != look {
		c.nodes = make(map[uint64]*paintengine2d.GroupNode)
		c.sig = make(map[uint64]uint64)
		c.ox, c.oy, c.w, c.h = ox, oy, w, h
		c.look = look
	}
}

func (c *rowSceneCache) reset() {
	*c = rowSceneCache{}
}

func (c *rowSceneCache) hit(id, sig uint64) *paintengine2d.GroupNode {
	if c == nil || c.nodes == nil || c.sig[id] != sig {
		return nil
	}
	return c.nodes[id]
}

func (c *rowSceneCache) store(id, sig uint64, g *paintengine2d.GroupNode) {
	if c.nodes == nil {
		c.nodes = make(map[uint64]*paintengine2d.GroupNode)
		c.sig = make(map[uint64]uint64)
	}
	c.nodes[id] = g
	c.sig[id] = sig
}

func (c *rowSceneCache) prune(keep map[uint64]struct{}) {
	if len(c.nodes) <= rowCacheCap {
		return
	}
	for id := range c.nodes {
		if _, ok := keep[id]; !ok {
			delete(c.nodes, id)
			delete(c.sig, id)
		}
	}
}

// visualSig is the convenience form of [rowSig] for callers with a fixed,
// small number of parts. Prefer rowSig in a per-row hot path: the variadic
// slice is what made TableView allocate per row per frame.
func visualSig(selected, hovered bool, extra uint64, parts ...string) uint64 {
	s := newRowSig(selected, hovered, extra)
	for _, p := range parts {
		s.str(p)
	}
	return s.sum()
}

func bits32(f float32) uint64 { return uint64(math.Float32bits(f)) }

// rowOrigin is where the painting widget sits in the space rows are
// recorded into. Rows bake the clip they were recorded under, so a widget
// that moved must record them again.
func rowOrigin(ctx *paintengine2d.Context) paintengine2d.Point {
	m := ctx.Matrix()
	return paintengine2d.Pt(m.E, m.F)
}

// rowIndexAt maps a local Y onto a painted virtual row (same formula as Paint:
// row i occupies [i*rowH − offsetY, (i+1)*rowH − offsetY)).
func rowIndexAt(y, offsetY, rowH float32, count int) int {
	if rowH <= 0 || count <= 0 {
		return -1
	}
	i := int(math.Floor(float64((y + offsetY) / rowH)))
	if i < 0 || i >= count {
		return -1
	}
	return i
}

func lookSig(lk style.LookAndFeel) uint64 {
	if lk == nil {
		return 0
	}
	h := bits32(lk.Metrics().FontSize) ^ bits32(lk.Metrics().RowH)
	for i, c := range lk.Name() {
		h ^= uint64(c) << uint(i%8)
	}
	return h
}

// rowSig accumulates a row's visual signature without allocating a slice of
// cell strings per row per frame.
type rowSig struct{ h uint64 }

func newRowSig(selected, hovered bool, extra uint64) rowSig {
	s := rowSig{h: 14695981039346656037}
	if selected {
		s.mix(1)
	}
	if hovered {
		s.mix(2)
	}
	s.mix(extra)
	return s
}

func (s *rowSig) mix(v uint64) {
	s.h ^= v
	s.h *= 1099511628211
}

// str folds a cell string in. Terminated so "ab","c" and "a","bc" differ.
func (s *rowSig) str(p string) {
	for i := 0; i < len(p); i++ {
		s.mix(uint64(p[i]))
	}
	s.mix(0xff)
}

func (s rowSig) sum() uint64 { return s.h }

// recordScrollingRows records the visible rows of a virtualized view.
//
// Each row is recorded at the origin (row-local space) and placed by its own
// group transform, so a row's recording does not depend on the scroll
// offset. All of them hang off a band group whose transform is the scroll
// offset and whose group clip is the viewport, in the band's parent space —
// the clip therefore stays put while the content slides under it, which is
// what lets a scrolled frame re-attach rows instead of re-recording them.
//
// viewport is in the painting widget's local coordinates; ctx supplies the
// matrix that maps it into the band's parent space.
func recordScrollingRows(
	rec *paintengine2d.Recorder,
	ctx *paintengine2d.Context,
	cache *rowSceneCache,
	groupID uint64,
	viewport paintengine2d.Rect,
	rowW, rowH float32,
	offsetY, y0 float32,
	lo, hi int,
	rowID func(i int) uint64,
	sigOf func(i int) uint64,
	paint func(i int),
) {
	if rec == nil || ctx == nil || cache == nil || lo >= hi {
		return
	}
	keep := rowKeepPool.Get().(map[uint64]struct{})
	clear(keep)
	defer func() {
		clear(keep)
		rowKeepPool.Put(keep)
	}()
	band := rec.BeginGroup(groupID, paintengine2d.Translation(0, y0-offsetY))
	if band != nil {
		band.SetClipRect(ctx.Matrix().TransformRect(viewport))
	}
	rowBox := paintengine2d.XYWH(0, 0, rowW, rowH)
	for i := lo; i < hi; i++ {
		id := rowID(i)
		keep[id] = struct{}{}
		s := sigOf(i)
		place := paintengine2d.Translation(0, float32(i)*rowH)
		if g := cache.hit(id, s); g != nil {
			g.Xform = place
			rec.Attach(g)
			continue
		}
		g := rec.BeginGroup(id, place)
		// Clip to the row itself so the clip baked into the row's draw
		// ops travels with the row instead of pinning it to wherever the
		// row happened to be recorded.
		ctx.Save()
		ctx.ClipRect(rowBox)
		paint(i)
		ctx.Restore()
		rec.EndGroup()
		if g != nil {
			cache.store(id, s, g)
		}
	}
	rec.EndGroup()
	cache.prune(keep)
}
