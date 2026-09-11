package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// rowCacheCap drops off-screen row groups once the map grows past this.
const rowCacheCap = 256

// rowSceneCache keeps recorded virtualized rows so scroll can change a
// parent transform without re-recording every visible row (Qt Quick ListView).
type rowSceneCache struct {
	nodes        map[uint64]*paintengine2d.GroupNode
	sig          map[uint64]uint64
	ox, oy, w, h float32
}

func (c *rowSceneCache) ready(ox, oy, w, h float32) {
	if c.nodes == nil || c.ox != ox || c.oy != oy || c.w != w || c.h != h {
		c.nodes = make(map[uint64]*paintengine2d.GroupNode)
		c.sig = make(map[uint64]uint64)
		c.ox, c.oy, c.w, c.h = ox, oy, w, h
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

func visualSig(selected, hovered bool, extra uint64, parts ...string) uint64 {
	h := uint64(14695981039346656037)
	mix := func(v uint64) {
		h ^= v
		h *= 1099511628211
	}
	if selected {
		mix(1)
	}
	if hovered {
		mix(2)
	}
	mix(extra)
	for _, p := range parts {
		for i := 0; i < len(p); i++ {
			mix(uint64(p[i]))
		}
		mix(0xff)
	}
	return h
}

func bits32(f float32) uint64 { return uint64(math.Float32bits(f)) }

// recordScrollingRows paints visible rows in content space (y = i*rowH)
// under Translation(0, y0-offsetY). Unchanged rows are Attached.
func recordScrollingRows(
	rec *paintengine2d.Recorder,
	cache *rowSceneCache,
	groupID uint64,
	offsetY, y0 float32,
	lo, hi int,
	rowID func(i int) uint64,
	sigOf func(i int) uint64,
	paint func(i int),
) {
	if rec == nil || cache == nil || lo >= hi {
		return
	}
	keep := make(map[uint64]struct{}, hi-lo)
	rec.BeginGroup(groupID, paintengine2d.Translation(0, y0-offsetY))
	for i := lo; i < hi; i++ {
		id := rowID(i)
		keep[id] = struct{}{}
		s := sigOf(i)
		if g := cache.hit(id, s); g != nil {
			rec.Attach(g)
			continue
		}
		g := rec.BeginGroup(id, paintengine2d.Identity())
		paint(i)
		rec.EndGroup()
		if g != nil {
			cache.store(id, s, g)
		}
	}
	rec.EndGroup()
	cache.prune(keep)
}
