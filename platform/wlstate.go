package platform

// This file holds the display-independent state machines the Wayland
// backend drives from its //export callbacks. Keeping them here (no cgo,
// no build tag) makes them unit-testable on a machine with no compositor.

// clipCache is one selection's local text cache.
//
// The cache may only answer a read while this process owns a live
// selection source. Without that rule the Wayland backend returned its
// own stale text forever: wl_data_source.cancelled destroyed the source
// (another app took the selection) but left the string in place, so
// Ctrl+V kept pasting what this app copied ten minutes ago. X11 gets this
// right through SelectionClear; this is the same invariant.
type clipCache struct {
	text  string
	owned bool
}

// set records text this process is now offering.
func (c *clipCache) set(s string) {
	c.text = s
	c.owned = true
}

// invalidate drops the cache: the selection is no longer ours
// (wl_data_source.cancelled, a foreign selection event, or shutdown).
func (c *clipCache) invalidate() {
	c.text = ""
	c.owned = false
}

// get returns the cached text when this process still owns the selection.
func (c *clipCache) get() (string, bool) {
	if !c.owned {
		return "", false
	}
	return c.text, true
}

// selectionChanged folds a wl_data_device.selection / primary selection
// event into the cache. ours is true when this process still holds a live
// source object for that selection; anything else means the selection
// moved to another client and the cache is stale.
func (c *clipCache) selectionChanged(ours bool) {
	if !ours {
		c.invalidate()
	}
}

// pickPresentSlot returns the first buffer slot the compositor has
// released. Reusing a slot that is still busy (the old fallback to slot
// 0) let the CPU blit into memory the compositor was still scanning out,
// which tears and, with linux-dmabuf, can be read mid-write by the GPU.
// The caller must skip the frame when this reports false.
func pickPresentSlot(busy []bool) (int, bool) {
	for i, b := range busy {
		if !b {
			return i, true
		}
	}
	return 0, false
}

// outputSet tracks wl_output scale factors by registry name, and which
// outputs a surface currently overlaps (wl_surface.enter / leave).
//
// The previous code kept a single wl_output proxy and a single outScale,
// so on a mixed-DPI desktop the last global announced won and a window on
// the 1x monitor was rendered (and set_buffer_scale'd) for 2x. The
// convention every toolkit follows is: a surface's integer scale is the
// maximum scale of the outputs it currently touches.
type outputSet struct {
	scale   map[uint32]int32
	entered map[uint32]bool
	// boxes is where each output sits on the desktop and how large it is
	// (see outputBox). Only the connection's set fills it; a surface's
	// set only ever tracks entered.
	boxes map[uint32]outputBox
}

func newOutputSet() *outputSet {
	return &outputSet{scale: map[uint32]int32{}, entered: map[uint32]bool{}, boxes: map[uint32]outputBox{}}
}

// setScale records wl_output.scale for a registry name.
func (o *outputSet) setScale(name uint32, scale int32) {
	if o == nil || scale < 1 {
		return
	}
	o.scale[name] = scale
}

// remove forgets an output that left the registry.
func (o *outputSet) remove(name uint32) {
	if o == nil {
		return
	}
	delete(o.scale, name)
	delete(o.entered, name)
	delete(o.boxes, name)
}

// enter / leave track wl_surface.enter and wl_surface.leave.
func (o *outputSet) enter(name uint32) {
	if o != nil {
		o.entered[name] = true
	}
}

func (o *outputSet) leave(name uint32) {
	if o != nil {
		delete(o.entered, name)
	}
}

// maxScale is the scale a surface should render at: the largest scale
// among the outputs it overlaps. Before the first enter (or after the
// last leave) it falls back to the largest scale on the desktop, which
// keeps a not-yet-mapped window from starting out blurry.
func (o *outputSet) maxScale() int32 {
	if o == nil {
		return 1
	}
	return o.maxScaleEntered(o.entered)
}

// maxScaleEntered resolves the scale for a surface whose entered output
// names are tracked separately from this set's scales: the connection
// owns the scales (one per wl_output), each surface owns the names it
// currently overlaps.
func (o *outputSet) maxScaleEntered(entered map[uint32]bool) int32 {
	if o == nil {
		return 1
	}
	best := int32(0)
	for name := range entered {
		if s := o.scale[name]; s > best {
			best = s
		}
	}
	if best > 0 {
		return best
	}
	for _, s := range o.scale {
		if s > best {
			best = s
		}
	}
	if best < 1 {
		best = 1
	}
	return best
}

// enteredNames is the set of outputs this surface overlaps.
func (o *outputSet) enteredNames() map[uint32]bool {
	if o == nil {
		return nil
	}
	return o.entered
}

// anyEntered reports whether the surface has received at least one enter.
func (o *outputSet) anyEntered() bool {
	return o != nil && len(o.entered) > 0
}

// outputBox is one wl_output's place on the desktop, as wl_output tells
// it: geometry gives the top-left corner in the compositor's global
// coordinate space, mode gives the size in the monitor's own pixels. The
// logical size — the unit a layer surface's margins are in — is the mode
// divided by the output's scale.
type outputBox struct {
	// X, Y are wl_output.geometry: global compositor space.
	X, Y int
	// ModeW, ModeH are wl_output.mode: physical pixels.
	ModeW, ModeH int
	// haveGeom, haveMode: the compositor has sent that event. An output
	// missing either cannot be matched against a point.
	haveGeom, haveMode bool
}

// setGeom records wl_output.geometry for a registry name.
func (o *outputSet) setGeom(name uint32, x, y int) {
	if o == nil {
		return
	}
	b := o.boxes[name]
	b.X, b.Y, b.haveGeom = x, y, true
	o.boxes[name] = b
}

// setMode records wl_output.mode (the current one) for a registry name.
func (o *outputSet) setMode(name uint32, w, h int) {
	if o == nil || w < 1 || h < 1 {
		return
	}
	b := o.boxes[name]
	b.ModeW, b.ModeH, b.haveMode = w, h, true
	o.boxes[name] = b
}

// logicalBox is an output's rectangle in logical pixels: its global
// position and its mode divided by its scale. ok is false while either
// event is still missing.
func (o *outputSet) logicalBox(name uint32) (FrameRect, bool) {
	if o == nil {
		return FrameRect{}, false
	}
	b, ok := o.boxes[name]
	if !ok || !b.haveGeom || !b.haveMode {
		return FrameRect{}, false
	}
	sc := int(o.scale[name])
	if sc < 1 {
		sc = 1
	}
	return FrameRect{X: b.X, Y: b.Y, W: b.ModeW / sc, H: b.ModeH / sc}, true
}

// outputAt is the output whose logical rectangle contains the desktop
// point x, y, and that rectangle. It is how a layer surface picks the
// monitor to anchor to: zwlr_layer_surface_v1's margins are measured from
// the edges of one output, so a point in the desktop's global coordinates
// has to be turned into a point on a named output first.
//
// ok is false when no output claims the point — nothing has been told to
// us yet, the outputs overlap oddly, or the point is off the desktop. The
// caller then lets the compositor pick the output and uses the global
// point unchanged, which is right on a single-monitor desktop and the
// best guess anywhere else.
func (o *outputSet) outputAt(x, y int) (name uint32, box FrameRect, ok bool) {
	if o == nil {
		return 0, FrameRect{}, false
	}
	for n := range o.boxes {
		b, have := o.logicalBox(n)
		if !have || b.W < 1 || b.H < 1 {
			continue
		}
		if x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H {
			return n, b, true
		}
	}
	return 0, FrameRect{}, false
}

// layerMargins turns a point in the desktop's global coordinates into the
// top and left margins of a layer surface anchored to the top-left corner
// of the output that holds the point, and names that output. When no
// output claims the point the margins are the point itself and name is 0
// — the caller then passes no output and lets the compositor choose.
func (o *outputSet) layerMargins(x, y int) (name uint32, left, top int) {
	if n, b, ok := o.outputAt(x, y); ok {
		return n, x - b.X, y - b.Y
	}
	return 0, x, y
}

// layerWindowSize turns the size a layer surface was configured with —
// the whole surface, in logical pixels — into the window size the rest of
// the toolkit speaks, which leaves the frame's margin out.
func layerWindowSize(surfW, surfH int, m FrameInsets) (int, int) {
	w, h := surfW-m.Width(), surfH-m.Height()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}
