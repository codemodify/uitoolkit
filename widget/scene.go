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

// cachedGroup is one widget's recorded group plus the state it was recorded
// under. Reuse is only valid when that state still holds: a group records
// its draw ops (and their clips) in the space it was recorded in, so a
// widget that moved needs at least a corrective transform, and one that
// resized needs a fresh recording because its clip changed.
type cachedGroup struct {
	node *paintengine2d.GroupNode
	// origin is the device (or content-space) origin at record time.
	origin paintengine2d.Point
	// clip is the accumulated device clip at record time. Reuse is only
	// sound when the whole thing — ops, their clips, and the ancestor
	// clip they were taken under — moved rigidly, so this must still
	// match after the move.
	clip paintengine2d.Rect
	// size is the arranged size at record time.
	size paintengine2d.Point
	// space is 0 for the window device space and otherwise the ID of the
	// scene layer whose content space this was recorded in. A group may
	// only be reused in the space it was recorded in.
	space uint64
	// content marks a scene-layer child recorded in its own content space
	// with no viewport clip baked into the ops, so it can be re-attached
	// under any scroll transform.
	content bool
	gen     uint32
}

// SceneCache stores recorded widget groups between frames.
type SceneCache struct {
	layers map[uint64]*cachedGroup
	dirty  map[uint64]bool
	gen    uint32
	reused int
}

// NewSceneCache constructs an empty layer cache.
func NewSceneCache() *SceneCache {
	return &SceneCache{
		layers: make(map[uint64]*cachedGroup),
		dirty:  make(map[uint64]bool),
	}
}

// Invalidate marks one widget's recorded group stale.
func (c *SceneCache) Invalidate(id uint64) {
	if c == nil {
		return
	}
	if c.dirty == nil {
		c.dirty = make(map[uint64]bool)
	}
	c.dirty[id] = true
}

// Reset drops every cached group (resize / theme / full invalidate).
// Maps are cleared in place so a full paint does not reallocate buckets.
func (c *SceneCache) Reset() {
	if c == nil {
		return
	}
	if c.layers != nil {
		clear(c.layers)
	} else {
		c.layers = make(map[uint64]*cachedGroup)
	}
	if c.dirty != nil {
		clear(c.dirty)
	} else {
		c.dirty = make(map[uint64]bool)
	}
	c.reused = 0
}

// Len is the number of retained groups (tests / inspector).
func (c *SceneCache) Len() int {
	if c == nil {
		return 0
	}
	return len(c.layers)
}

// Reused is how many groups the last recording attached instead of
// re-recording.
func (c *SceneCache) Reused() int {
	if c == nil {
		return 0
	}
	return c.reused
}

// EndFrame closes a recording pass: groups that were not visited this frame
// belong to widgets that are hidden or detached, so their recorded ops (and
// any images they hold) are dropped, and their stale dirty flags with them.
// Without this a removed subtree keeps every ancestor permanently dirty.
func (c *SceneCache) EndFrame() {
	if c == nil {
		return
	}
	for id, ent := range c.layers {
		if ent == nil || ent.gen != c.gen {
			delete(c.layers, id)
			delete(c.dirty, id)
		}
	}
	c.gen++
}

func (c *SceneCache) dirtyID(id uint64) bool {
	return c != nil && c.dirty != nil && c.dirty[id]
}

func (c *SceneCache) get(id uint64) *cachedGroup {
	if c == nil || c.layers == nil {
		return nil
	}
	return c.layers[id]
}

func (c *SceneCache) store(ent *cachedGroup) {
	if c == nil || ent == nil || ent.node == nil {
		return
	}
	if c.layers == nil {
		c.layers = make(map[uint64]*cachedGroup)
	}
	ent.gen = c.gen
	c.layers[ent.node.ID] = ent
	delete(c.dirty, ent.node.ID)
}

// touch marks ent as still in use this frame.
func (c *SceneCache) touch(ent *cachedGroup) {
	if c == nil || ent == nil {
		return
	}
	ent.gen = c.gen
	c.reused++
}

// markLive keeps the groups of a subtree that was attached wholesale. Its
// widgets are not visited by the recording, but they are still on screen, so
// EndFrame must not evict them.
func (c *SceneCache) markLive(n Component) {
	if c == nil || n == nil || len(c.layers) == 0 {
		return
	}
	if ent := c.layers[n.ID()]; ent != nil {
		ent.gen = c.gen
	}
	if sl, ok := n.(SceneLayer); ok {
		if ch := sl.SceneChild(); ch != nil {
			c.markLive(ch)
		}
		return
	}
	for _, ch := range n.Children() {
		c.markLive(ch)
	}
}

func subtreeDirty(c Component, cache *SceneCache) bool {
	if c == nil || cache == nil || !c.Visible() {
		// An invisible subtree contributes nothing to the recording, so
		// its stale dirty flags must not force ancestors to re-record.
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

// recordState is the per-recording context threaded through recordNode.
type recordState struct {
	cache *SceneCache
	// space is the coordinate space the current ops are recorded in: 0 for
	// the window, otherwise the scene layer whose content space we are in.
	space uint64
}

// RecordTree walks c into rec (retained nodes). Clean widgets reuse the
// cached group. SceneLayer children can be reused with only a transform.
//
// dirty is accepted for API compatibility and deliberately unused: the
// recording must always be a complete display list for the window, because
// partial redraw is applied when the scene is replayed
// ([paintengine2d.DrawSceneDamage]). Recording a subset would make the next
// damaged replay paint from a scene that never contained the clean widgets.
func RecordTree(c Component, rec *paintengine2d.Recorder, ctx *paintengine2d.Context, dirty *paintengine2d.Damage, cache *SceneCache, fullContent bool) {
	if rec == nil || ctx == nil {
		return
	}
	_ = dirty
	if cache != nil {
		cache.reused = 0
	}
	recordNode(c, rec, ctx, &recordState{cache: cache}, fullContent)
}

// originOf is the translation part of the context matrix. Recording only
// ever translates, so this is the node's origin in the current space.
func originOf(ctx *paintengine2d.Context) (paintengine2d.Point, bool) {
	m := ctx.Matrix()
	if !m.IsTranslation() {
		return paintengine2d.Point{}, false
	}
	return paintengine2d.Pt(m.E, m.F), true
}

func recordNode(c Component, rec *paintengine2d.Recorder, ctx *paintengine2d.Context, st *recordState, fullContent bool) {
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
	defer ctx.Restore()
	ctx.Translate(b.Min.X, b.Min.Y)
	ctx.ClipRect(local)
	if ctx.ClipEmpty() || (!fullContent && ctx.QuickReject(local)) {
		return
	}
	cache := st.cache
	origin, translated := originOf(ctx)
	size := paintengine2d.Pt(local.Dx(), local.Dy())
	clip := ctx.DeviceClipBounds()

	if cache != nil && translated {
		if ent := cache.get(c.ID()); ent != nil && !ent.content && ent.space == st.space &&
			ent.size == size && !cache.dirtyID(c.ID()) && !subtreeDirty(c, cache) {
			// The recorded ops (and their clips) are in the space the
			// widget occupied when they were taken. A widget that only
			// moved is still valid under a corrective translation — but
			// only when its clip moved with it, so a widget sliding
			// under a clip that stayed put re-records instead.
			if dx, dy, ok := reuseDelta(ent, origin, clip); ok {
				attachMoved(rec, ent, dx, dy)
				cache.touch(ent)
				cache.markLive(c)
				return
			}
		}
	}

	g := rec.BeginGroup(c.ID(), paintengine2d.Identity())
	c.Paint(ctx)
	if sl, ok := c.(SceneLayer); ok {
		if ch := sl.SceneChild(); ch != nil {
			recordSceneChild(sl, ch, local, origin, translated, rec, ctx, st)
		}
	} else {
		// Walk children even when ManagesChildren (Splitter). Immediate
		// PaintTree already drew those panes; Recording skips that so a
		// nested SceneLayer (ScrollView) can record its child here.
		for _, ch := range c.Children() {
			recordNode(ch, rec, ctx, st, fullContent)
		}
	}
	rec.EndGroup()
	if cache != nil && g != nil {
		cache.store(&cachedGroup{node: g, origin: origin, clip: clip, size: size, space: st.space})
	}
}

// reuseDelta is how far a cached group moved, and whether reusing it is
// sound. It is only sound when the clip it was recorded under translated by
// exactly the same amount: the recorded ops carry that clip, so a rigid move
// keeps them correct while a slide under a fixed clip does not.
func reuseDelta(ent *cachedGroup, origin paintengine2d.Point, clip paintengine2d.Rect) (dx, dy float32, ok bool) {
	dx, dy = origin.X-ent.origin.X, origin.Y-ent.origin.Y
	want := ent.clip.Translate(paintengine2d.Pt(dx, dy))
	if !nearRect(want, clip) {
		return 0, 0, false
	}
	return dx, dy, true
}

func nearRect(a, b paintengine2d.Rect) bool {
	return nearF(a.Min.X, b.Min.X) && nearF(a.Min.Y, b.Min.Y) &&
		nearF(a.Max.X, b.Max.X) && nearF(a.Max.Y, b.Max.Y)
}

func nearF(a, b float32) bool {
	d := a - b
	return d < 1.0/64 && d > -1.0/64
}

// attachMoved splices a cached group back in, correcting for a widget that
// moved since it was recorded.
func attachMoved(rec *paintengine2d.Recorder, ent *cachedGroup, dx, dy float32) {
	if dx == 0 && dy == 0 {
		rec.Attach(ent.node)
		return
	}
	rec.Attach(&paintengine2d.GroupNode{
		ID:       ent.node.ID,
		Xform:    paintengine2d.Translation(dx, dy),
		Children: ent.node.Children,
	})
}

// sceneContentBudget caps how much off-screen content is recorded for a
// retained scroll layer, in viewports. Past that the layer is recorded
// clipped to its viewport and re-recorded every frame instead.
const sceneContentBudget = 8

// sceneContentFloor is the height (device px) below which the budget never
// bites, so short content is always retained.
const sceneContentFloor = 4096

// recordSceneChild records the content subtree of a scene layer.
//
// The content is recorded in its own space, with no viewport clip baked
// into the ops, into a group whose Xform places it and whose group Clip is
// the viewport. Sliding the content is then a new Xform on a clip that
// stays put — and rows that were off-screen when the content was recorded
// are in the group, so scrolling them into view shows them instead of
// background.
func recordSceneChild(sl SceneLayer, ch Component, viewport paintengine2d.Rect, origin paintengine2d.Point, translated bool, rec *paintengine2d.Recorder, ctx *paintengine2d.Context, st *recordState) {
	cache := st.cache
	cb := ch.Bounds()
	cw, chh := cb.Dx(), cb.Dy()
	_, retain := sl.RetainScene()
	budget := viewport.Dy() * sceneContentBudget
	if budget < sceneContentFloor {
		budget = sceneContentFloor
	}
	if !translated || cache == nil || cw < 1 || chh < 1 || chh > budget {
		// Not retainable: record inline, clipped to the viewport. Correct
		// because this path re-records every frame.
		// No cache for this subtree: its ops carry the viewport clip, so
		// a later frame must not slide them under it.
		ctx.Save()
		ctx.ClipRect(viewport)
		recordNode(ch, rec, ctx, &recordState{}, true)
		ctx.Restore()
		sl.NoteSceneRecorded()
		return
	}

	// Device-space viewport for the group clip. Group clips resolve under
	// the matrix accumulated to the group's parent, which is the identity
	// chain of enclosing widget groups, so this is device space.
	clip := paintengine2d.XYWH(origin.X, origin.Y, viewport.Dx(), viewport.Dy())
	// Where the content's top-left currently sits in device space.
	place := paintengine2d.Translation(origin.X+cb.Min.X, origin.Y+cb.Min.Y)

	ent := cache.get(ch.ID())
	size := paintengine2d.Pt(cw, chh)
	if retain && ent != nil && ent.content && ent.space == st.space && ent.size == size && !subtreeDirty(ch, cache) {
		g := &paintengine2d.GroupNode{ID: ch.ID(), Xform: place, Children: ent.node.Children}
		g.SetClipRect(clip)
		rec.Attach(g)
		cache.touch(ent)
		cache.markLive(ch)
		return
	}

	// Re-record the content in its own space: a fresh recorder sized to the
	// content, so nothing outside the viewport is culled away. The context
	// is pre-shifted by the content's current position so the recording
	// starts at (0,0) and is therefore independent of the scroll offset —
	// otherwise a partially scrolled first row would be cut off by the
	// sub-recorder's own canvas.
	sub := paintengine2d.NewRecorder(ceilInt(cw), ceilInt(chh))
	subCtx := paintengine2d.NewContextDevice(sub)
	subCtx.Translate(-cb.Min.X, -cb.Min.Y)
	space := ch.ID()
	if sc, ok := sl.(Component); ok {
		space = sc.ID()
	}
	subState := &recordState{cache: cache, space: space}
	recordNode(ch, sub, subCtx, subState, true)
	root := sub.Finish()
	node := &paintengine2d.GroupNode{ID: ch.ID()}
	if root != nil && root.Root != nil {
		node.Children = root.Root.Children
	}
	cache.store(&cachedGroup{node: node, size: size, space: st.space, content: true})
	sl.NoteSceneRecorded()

	g := &paintengine2d.GroupNode{ID: ch.ID(), Xform: place, Children: node.Children}
	g.SetClipRect(clip)
	rec.Attach(g)
}

func ceilInt(v float32) int {
	n := int(v)
	if float32(n) < v {
		n++
	}
	if n < 1 {
		n = 1
	}
	return n
}
