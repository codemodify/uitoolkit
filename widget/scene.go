package widget

import "github.com/codemodify/paintengine2d"

// SceneLayer is a component that can keep a retained child subtree and
// only update a transform (scroll offset). Qt Quick Item / GSK transform node.
type SceneLayer interface {
	SceneChild() Component
	RetainScene() (extra paintengine2d.Matrix, retain bool)
	MarkSceneChildDirty()
	NoteSceneRecorded()
}

// SceneCache stores recorded widget groups between frames.
type SceneCache struct {
	Layers map[uint64]*paintengine2d.GroupNode
	Dirty  map[uint64]bool
	Reused int
}

// NewSceneCache constructs an empty layer cache.
func NewSceneCache() *SceneCache {
	return &SceneCache{
		Layers: make(map[uint64]*paintengine2d.GroupNode),
		Dirty:  make(map[uint64]bool),
	}
}

// Invalidate marks one widget's recorded group stale.
func (c *SceneCache) Invalidate(id uint64) {
	if c == nil {
		return
	}
	if c.Dirty == nil {
		c.Dirty = make(map[uint64]bool)
	}
	c.Dirty[id] = true
}

// Reset drops every cached group (resize / theme / full invalidate).
// Maps are cleared in place so a full paint does not reallocate buckets.
func (c *SceneCache) Reset() {
	if c == nil {
		return
	}
	if c.Layers != nil {
		clear(c.Layers)
	} else {
		c.Layers = make(map[uint64]*paintengine2d.GroupNode)
	}
	if c.Dirty != nil {
		clear(c.Dirty)
	} else {
		c.Dirty = make(map[uint64]bool)
	}
	c.Reused = 0
}

func (c *SceneCache) dirtyID(id uint64) bool {
	return c != nil && c.Dirty != nil && c.Dirty[id]
}

func subtreeDirty(c Component, cache *SceneCache) bool {
	if c == nil || cache == nil {
		return false
	}
	if cache.dirtyID(c.ID()) {
		return true
	}
	if sl, ok := c.(SceneLayer); ok {
		if ch := sl.SceneChild(); ch != nil {
			return subtreeDirty(ch, cache)
		}
		return false
	}
	for _, ch := range c.Children() {
		if subtreeDirty(ch, cache) {
			return true
		}
	}
	return false
}

// Recording reports whether ctx is capturing into a [paintengine2d.Recorder].
func Recording(ctx *paintengine2d.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Device().(*paintengine2d.Recorder)
	return ok
}

// RecordTree walks c into rec (retained nodes). Clean widgets reuse the
// cached group. SceneLayer children can be reused with only a transform.
func RecordTree(c Component, rec *paintengine2d.Recorder, ctx *paintengine2d.Context, dirty *paintengine2d.Damage, cache *SceneCache, fullContent bool) {
	if rec == nil || ctx == nil {
		return
	}
	if cache != nil {
		cache.Reused = 0
	}
	recordNode(c, rec, ctx, dirty, cache, fullContent)
}

func recordNode(c Component, rec *paintengine2d.Recorder, ctx *paintengine2d.Context, dirty *paintengine2d.Damage, cache *SceneCache, fullContent bool) {
	if c == nil || !c.Visible() {
		return
	}
	b := c.Bounds()
	local := c.LocalBounds()
	if local.Empty() && !b.Empty() {
		local = paintengine2d.XYWH(0, 0, b.Dx(), b.Dy())
	}
	if local.Empty() {
		return
	}
	ctx.Save()
	ctx.Translate(b.Min.X, b.Min.Y)
	ctx.ClipRect(local)
	if ctx.ClipEmpty() || (!fullContent && ctx.QuickReject(local)) {
		ctx.Restore()
		return
	}
	if cache != nil && cache.Layers[c.ID()] != nil && !subtreeDirty(c, cache) && !cache.dirtyID(c.ID()) {
		rec.Attach(cache.Layers[c.ID()])
		cache.Reused++
		ctx.Restore()
		return
	}

	g := rec.BeginGroup(c.ID(), paintengine2d.Identity())
	c.Paint(ctx)
	if sl, ok := c.(SceneLayer); ok {
		if ch := sl.SceneChild(); ch != nil {
			ctx.Save()
			ctx.ClipRect(local)
			extra, retain := sl.RetainScene()
			cached := (*paintengine2d.GroupNode)(nil)
			if cache != nil {
				cached = cache.Layers[ch.ID()]
			}
			if retain && cached != nil && !subtreeDirty(ch, cache) {
				rec.Attach(&paintengine2d.GroupNode{ID: ch.ID(), Xform: extra, Children: cached.Children})
				if cache != nil {
					cache.Reused++
				}
			} else {
				recordNode(ch, rec, ctx, nil, cache, true)
				sl.NoteSceneRecorded()
			}
			ctx.Restore()
		}
	} else if !c.ManagesChildren() {
		for _, ch := range c.Children() {
			recordNode(ch, rec, ctx, dirty, cache, fullContent)
		}
	}
	rec.EndGroup()
	if cache != nil && g != nil {
		cache.Layers[c.ID()] = g
		delete(cache.Dirty, c.ID())
	}
	ctx.Restore()
}
