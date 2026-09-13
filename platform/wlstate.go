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
}

func newOutputSet() *outputSet {
	return &outputSet{scale: map[uint32]int32{}, entered: map[uint32]bool{}}
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
