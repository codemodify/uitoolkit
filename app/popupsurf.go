package app

import (
	"log"
	"math"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Popups on surfaces of their own.
//
// A menu, a combo box's list, a submenu and a tooltip used to be layers the
// window painted over its own content, which is why a menu could never be
// larger than its window: Minim's 275x116 strip cut its skin menu off at the
// strip's edge. Now each is a surface of its own — an xdg_popup on Wayland,
// an override-redirect window on X11 — which the window system places
// against the screen's edges rather than the window's (platform/popup.go).
//
// Nothing else about a popup changes. The component is still the window's
// popup layer (w.popup and its cascade, w.tooltip), so focus, the keyboard,
// the accessibility tree and every widget's own code are exactly what they
// were; the backend hands the popup's input to the window with positions
// already in the window's coordinates, so hit-testing, hover and the click
// outside that dismisses work unchanged. What moved is only *where the
// pixels go*: the window no longer paints these layers, and each popup
// paints its own surface, shadow and all, from the same component at the
// place the window system gave it.
//
// Where a backend cannot open popups — headless and offscreen windows, a
// compositor that refuses one — the window keeps painting them inside
// itself, as it always did, and UITK_POPUPS=layer makes that the rule for
// comparison.

// popLayer is one of the window's popups on a surface of its own.
type popLayer struct {
	c    widget.Component
	kind style.PopupKind
	surf platform.PopupSurface
	// place is what the surface was last placed with and box the bounds
	// the component had then: the widget re-placing it (a list that grew)
	// is how a popup learns it has to move.
	place platform.PopupPlacement
	box   paintengine2d.Rect
	frame platform.Frame
	dirty bool
	// sil is the popup's silhouette, where it has one.
	sil popSilhouette
}

// popSilhouette is a popup's outline worked out for its size: an explicit
// one (the component's own hit shape, or its look's PopupShapeOf), or —
// only where the popup is glass and the blur behind has to stop at its
// rounded corners — the coverage of what it paints.
type popSilhouette struct {
	key    popShapeKey
	shape  *platform.Shape
	raster *platform.ShapeRaster
	// derived: the outline was read from what the popup paints, so there
	// is nothing to cut out of it — it is its own edge already.
	derived bool
	eraser  *paintengine2d.Image
	wipe    *paintengine2d.Path
	shade   *paintengine2d.Image
}

type popShapeKey struct {
	look  style.LookAndFeel
	own   *platform.Shape
	w, h  int
	glass bool
	solid bool
}

// popupsOnSurfaces reports whether this window's popups are surfaces of
// their own: the backend can open them, UITK_POPUPS does not say otherwise,
// no compositor has refused one, and the window is not itself a status
// menu (whose popup is the whole window).
func (w *Window) popupsOnSurfaces() bool {
	if w == nil || w.statusMenu || w.popsRefused || w.Closed() {
		return false
	}
	o, ok := w.surf.(platform.PopupOpener)
	return ok && o.PopupsSupported() && platform.PopupSurfacesAllowed()
}

// PopupSurfaces reports whether the window's menus, lists and tooltips open
// as surfaces of their own — free to run past the window's edges — rather
// than being drawn inside it. It is false headless, under UITK_POPUPS=layer,
// and after the window system has refused a popup.
func (w *Window) PopupSurfaces() bool { return w.popupsOnSurfaces() }

// PopupArea is the box the window's popups may use, in its device pixels
// (widget.PopupAreaHost): the work area of the monitor it is on where the
// backend knows it (X11, the offscreen desktop), and on Wayland — where a
// client never learns where its window is — a box the size of the screen
// around the window, which is only the first guess: the compositor has the
// last word and slides or flips the popup itself. ok is false while popups
// are drawn inside the window, which then keeps them inside itself.
func (w *Window) PopupArea() (paintengine2d.Rect, bool) {
	if !w.popupsOnSurfaces() {
		return paintengine2d.Rect{}, false
	}
	win := w.windowBox()
	sc := max(w.scale, 1)
	if a, ok := platform.SurfacePopupWorkArea(w.surf); ok && a.W > 0 && a.H > 0 {
		return paintengine2d.XYWH(win.Min.X+float32(a.X)*sc, win.Min.Y+float32(a.Y)*sc,
			float32(a.W)*sc, float32(a.H)*sc), true
	}
	bw, bh := 0, 0
	if b, ok := w.surf.(interface{ ConfigureBounds() (int, int) }); ok {
		bw, bh = b.ConfigureBounds()
	}
	if bw < 1 || bh < 1 {
		bw, bh = 1920, 1080
	}
	sw, sh := float32(bw)*sc, float32(bh)*sc
	cx, cy := (win.Min.X+win.Max.X)/2, (win.Min.Y+win.Max.Y)/2
	return paintengine2d.XYWH(cx-sw/2, cy-sh/2, sw, sh), true
}

// popChain is what should be on popup surfaces now: the popup layer and
// its open submenus, parent first.
func (w *Window) popChain() []widget.Component {
	if w.popup == nil {
		return nil
	}
	var chain []widget.Component
	widget.WalkCascade(w.popup, func(c widget.Component) {
		if c.Visible() {
			chain = append(chain, c)
		}
	})
	return chain
}

// syncPopups brings the popup surfaces in line with the popup layers:
// every open menu and submenu, and the tooltip, gets a surface, and every
// surface whose component went away is closed — a submenu before the menu
// it hangs from, which the window system insists on.
func (w *Window) syncPopups() {
	if !w.popupsOnSurfaces() {
		if len(w.pops) > 0 || w.tipPop != nil {
			w.closePops(0)
			w.closeTipPop()
		}
		return
	}
	chain := w.popChain()
	keep := 0
	for keep < len(chain) && keep < len(w.pops) && w.pops[keep].c == chain[keep] && !w.pops[keep].surf.Closed() {
		keep++
	}
	w.closePops(keep)
	for i, pl := range w.pops {
		if !w.followPop(pl) {
			// Moved by closing and opening again: a submenu goes with it.
			w.closePops(i)
			break
		}
	}
	for i := len(w.pops); i < len(chain); i++ {
		var parent platform.Surface = w.surf
		if i > 0 {
			parent = w.pops[i-1].surf
		}
		pl := w.openPop(chain[i], style.PopupMenu, parent)
		if pl == nil {
			w.refusePopups()
			return
		}
		w.pops = append(w.pops, pl)
	}
	switch {
	case w.tooltip == nil || !w.tooltip.Visible():
		w.closeTipPop()
	case w.tipPop != nil && w.tipPop.c == w.tooltip && !w.tipPop.surf.Closed() && w.followPop(w.tipPop):
	default:
		w.closeTipPop()
		if w.tipPop = w.openPop(w.tooltip, style.PopupTooltip, w.surf); w.tipPop == nil {
			w.refusePopups()
		}
	}
}

// openPop opens a surface for popup component c, hanging from parent, and
// moves c to where the window system put it. nil means the backend refused.
func (w *Window) openPop(c widget.Component, kind style.PopupKind, parent platform.Surface) *popLayer {
	opener, ok := parent.(platform.PopupOpener)
	if !ok {
		return nil
	}
	pl := &popLayer{c: c, kind: kind}
	pl.place = w.popupPlacement(c)
	pl.frame = w.popFrame(pl)
	role := platform.PopupRoleMenu
	if kind == style.PopupTooltip {
		role = platform.PopupRoleTooltip
	}
	s, err := opener.OpenPopup(platform.PopupOptions{
		Parent: parent, Placement: pl.place, Role: role, Frame: pl.frame,
	})
	if err != nil || s == nil {
		return nil
	}
	pl.surf = s
	w.adoptPop(pl)
	return pl
}

// followPop moves a popup's surface after its component: the widget placed
// it again (a list that grew, a submenu whose parent moved). false means the
// backend cannot move a popup that is up, and it has to be opened afresh.
func (w *Window) followPop(pl *popLayer) bool {
	if pl.c.Bounds() == pl.box {
		return true
	}
	pl.place = w.popupPlacement(pl.c)
	if f := w.popFrame(pl); !f.Same(pl.frame) {
		pl.frame = f
		pl.surf.SetFrame(f)
	}
	if !pl.surf.Reposition(pl.place) {
		return false
	}
	w.adoptPop(pl)
	return true
}

// adoptPop moves the popup's component to where the window system put its
// surface — flipped above a combo box near the bottom of the screen, slid
// left of a screen edge, shrunk to what fits (the list then scrolls) — so
// the widget hit-tests and paints where the user sees it.
func (w *Window) adoptPop(pl *popLayer) {
	placed := pl.surf.Placed()
	o := pl.surf.Origin()
	m := pl.frame.Margin
	sc := max(w.scale, 1)
	b := pl.c.Bounds()
	// The component fills the visible box exactly — whole logical pixels,
	// which at a fractional scale is up to a device pixel or two more than
	// it measured — so no transparent seam is left along its far edges.
	wd, ht := b.Dx(), b.Dy()
	if placed.W > 0 {
		wd = float32(platform.DevicePixels(placed.W, sc))
	}
	if placed.H > 0 {
		ht = float32(platform.DevicePixels(placed.H, sc))
	}
	nb := paintengine2d.XYWH(o.X+float32(m.Left), o.Y+float32(m.Top), wd, ht)
	if nb != b {
		pl.c.Arrange(nb)
	}
	pl.box = pl.c.Bounds()
	if f := w.popFrame(pl); !f.Same(pl.frame) {
		pl.frame = f
		pl.surf.SetFrame(f)
	}
	pl.dirty = true
}

// popupPlacement is c's place in the window system's terms: what it hangs
// from (widget.PopupAnchorOf, which the Place* functions leave behind) and
// which way it went, in logical pixels relative to the window's visible
// box, with the adjustments Qt and GTK allow each kind of popup — a
// drop-down flips above and slides sideways, a submenu flips to the other
// side and slides up, anything at a point may do both.
func (w *Window) popupPlacement(c widget.Component) platform.PopupPlacement {
	b := c.Bounds()
	a, ok := widget.PopupAnchorOf(c)
	if !ok {
		a = widget.PopupAnchor{Rect: paintengine2d.XYWH(b.Min.X, b.Min.Y, 0, 0), Side: widget.PopupAt}
	}
	win := w.windowBox().Min
	sc := max(w.scale, 1)
	lx := func(v float32) int { return int(math.Round(float64((v - win.X) / sc))) }
	ly := func(v float32) int { return int(math.Round(float64((v - win.Y) / sc))) }
	ar := a.Rect
	anchor := platform.FrameRect{X: lx(ar.Min.X), Y: ly(ar.Min.Y)}
	anchor.W = max(lx(ar.Max.X)-anchor.X, 1)
	anchor.H = max(ly(ar.Max.Y)-anchor.Y, 1)
	p := platform.PopupPlacement{
		Anchor: anchor,
		W:      max(int(math.Ceil(float64(b.Dx()/sc)-1e-3)), 1),
		H:      max(int(math.Ceil(float64(b.Dy()/sc)-1e-3)), 1),
	}
	const (
		top, bottom = platform.EdgeTop, platform.EdgeBottom
		left, right = platform.EdgeLeft, platform.EdgeRight
	)
	switch a.Side {
	case widget.PopupBelow:
		p.AnchorEdge, p.Gravity = bottom|left, bottom|right
		p.Adjust = platform.AdjustSlideX | platform.AdjustFlipY | platform.AdjustResizeY
	case widget.PopupAbove:
		p.AnchorEdge, p.Gravity = top|left, top|right
		p.Adjust = platform.AdjustSlideX | platform.AdjustFlipY | platform.AdjustResizeY
	case widget.PopupRight:
		p.AnchorEdge, p.Gravity = top|right, bottom|right
		p.Adjust = platform.AdjustFlipX | platform.AdjustSlideY | platform.AdjustResizeY
	case widget.PopupLeft:
		p.AnchorEdge, p.Gravity = top|left, bottom|left
		p.Adjust = platform.AdjustFlipX | platform.AdjustSlideY | platform.AdjustResizeY
	default:
		// At a point: the anchor's far corner is where the popup starts
		// (a tooltip's anchor is the pointer grown by the gap to it), and
		// a 0x0 anchor is a point with a pixel before it.
		if ar.Dx() < 1 {
			anchor.X, anchor.W = lx(ar.Max.X)-1, 1
		}
		if ar.Dy() < 1 {
			anchor.Y, anchor.H = ly(ar.Max.Y)-1, 1
		}
		p.Anchor = anchor
		p.AnchorEdge, p.Gravity = bottom|right, bottom|right
		p.Adjust = platform.AdjustFlipX | platform.AdjustFlipY | platform.AdjustSlideX |
			platform.AdjustSlideY | platform.AdjustResizeY
	}
	return p
}

// popMargin is the band around a popup its shadow is painted in: the
// look's reach for the kind, whole logical pixels, and none on a screen
// that composites nothing, where alpha counts for nothing.
func (w *Window) popMargin(lk style.LookAndFeel, kind style.PopupKind) platform.FrameInsets {
	if w.state.Solid {
		return platform.FrameInsets{}
	}
	in := style.PopupShadowOf(lk, kind)
	sc := max(w.scale, 1)
	edge := func(v float32) int {
		if v <= 0 {
			return 0
		}
		return platform.FrameMargin(v, sc)
	}
	return platform.FrameInsets{Top: edge(in.Top), Right: edge(in.Right), Bottom: edge(in.Bottom), Left: edge(in.Left)}
}

// popGlass reports whether a popup in lk is real glass here: the look's
// materials are translucent and the desktop blurs behind a surface.
func (w *Window) popGlass(lk style.LookAndFeel) bool {
	return !w.state.Solid && style.WantsGlass(lk) && platform.SurfaceBlurBehind(w.surf)
}

// popFrame is what the window system is told about a popup's surface: the
// margin its shadow lives in, where it takes input — its silhouette, the
// whole box, or nowhere at all for a tooltip, which every press goes
// through — and the glass behind it. Nothing is ever claimed opaque: a
// menu's rounded corners are transparent, and so is its tint where it is
// glass.
func (w *Window) popFrame(pl *popLayer) platform.Frame {
	lk := w.layerLook(pl.c)
	m := w.popMargin(lk, pl.kind)
	b := pl.c.Bounds()
	f := platform.Frame{Margin: m, Alpha: !w.state.Solid, Opaque: []platform.FrameRect{}}
	box := platform.FrameRect{X: m.Left, Y: m.Top, W: max(int(b.Dx()), 1), H: max(int(b.Dy()), 1)}
	sil := w.popShape(pl)
	switch {
	case pl.kind == style.PopupTooltip:
		f.Shape = []platform.FrameRect{}
	case sil != nil && !pl.sil.derived:
		f.Shape = platform.OffsetRects(sil.Rects, m.Left, m.Top)
	default:
		f.Shape = []platform.FrameRect{box}
	}
	if w.popGlass(lk) {
		if sil != nil {
			f.Blur = platform.OffsetRects(sil.Rects, m.Left, m.Top)
		} else {
			f.Blur = []platform.FrameRect{box}
		}
	}
	return f
}

// popShape is the popup's silhouette at its current size, or nil for the
// plain box: the component's own hit shape where it has one, else its
// look's (style.PopupShapeOf), else — only for glass, whose blur must stop
// where the popup's rounded corners do — the coverage of what it paints.
// It is worked out once per look and size, not per frame.
func (w *Window) popShape(pl *popLayer) *platform.ShapeRaster {
	lk := w.layerLook(pl.c)
	b := pl.c.Bounds()
	key := popShapeKey{look: lk, w: int(b.Dx()), h: int(b.Dy()), glass: w.popGlass(lk), solid: w.state.Solid}
	if key.w < 1 || key.h < 1 {
		return nil
	}
	if hs, ok := pl.c.(widget.HitShaper); ok {
		key.own = hs.HitShape()
	}
	if pl.sil.key == key {
		return pl.sil.raster
	}
	pl.sil = popSilhouette{key: key}
	shape := key.own
	if shape == nil {
		shape = platform.NewShapeSilhouette(style.PopupShapeOf(lk, paintengine2d.XYWH(0, 0, b.Dx(), b.Dy()), pl.kind))
	}
	if shape == nil && key.glass && pl.kind != style.PopupTooltip {
		shape = w.paintedShape(pl.c)
		pl.sil.derived = shape != nil
	}
	if shape == nil {
		return nil
	}
	pl.sil.shape = shape
	pl.sil.raster = shape.Raster(key.w, key.h)
	return pl.sil.raster
}

// paintedShape is the outline of what c paints: its alpha, read from one
// render of it on nothing. A translucent tint still counts as the popup —
// only what the look leaves entirely unpainted, its rounded corners, is
// outside.
func (w *Window) paintedShape(c widget.Component) *platform.Shape {
	b := c.Bounds()
	iw, ih := int(b.Dx()), int(b.Dy())
	if iw < 1 || ih < 1 || iw*ih > 4096*4096 {
		return nil
	}
	img := paintengine2d.NewImage(iw, ih)
	ctx := paintengine2d.NewContext(img)
	if ctx == nil {
		return nil
	}
	ctx.Clear(paintengine2d.Transparent)
	ctx.Translate(-b.Min.X, -b.Min.Y)
	prev := style.SetLayerBackdrop(style.BackdropGlass)
	widget.PaintTree(c, ctx, nil)
	style.SetLayerBackdrop(prev)
	mask := make([]uint8, iw*ih)
	stride := img.RowStride()
	for y := 0; y < ih; y++ {
		row := img.Pix[y*stride:]
		for x := 0; x < iw; x++ {
			mask[y*iw+x] = uint8(min(int(row[x*4+3])*4, 255))
		}
	}
	return platform.NewShapeMask(mask, iw, ih, iw)
}

// closePops closes the popup surfaces from index from on, the deepest
// submenu first.
func (w *Window) closePops(from int) {
	for i := len(w.pops) - 1; i >= from && i >= 0; i-- {
		_ = w.pops[i].surf.Close()
	}
	if from < len(w.pops) {
		w.pops = w.pops[:from]
	}
}

func (w *Window) closeTipPop() {
	if w.tipPop != nil {
		_ = w.tipPop.surf.Close()
		w.tipPop = nil
	}
}

// refusePopups falls back to drawing popups inside the window for the rest
// of its life: the backend refused one, and a menu cut off at the window's
// edge is better than a menu that never appears. The popups up now are
// placed again inside the window.
func (w *Window) refusePopups() {
	if w.popsRefused {
		return
	}
	log.Printf("uitoolkit: the window system refused a popup surface; drawing popups inside the window")
	w.popsRefused = true
	w.closePops(0)
	w.closeTipPop()
	for _, c := range w.popChain() {
		widget.ClampToSurface(c, c)
	}
	if w.tooltip != nil {
		widget.ClampToSurface(w.tooltip, w.tooltip)
	}
	w.fullInvalidate()
}

// popLayerOf is the popup surface c paints into, if it is on one.
func (w *Window) popLayerOf(c widget.Component) *popLayer {
	if c == nil || (len(w.pops) == 0 && w.tipPop == nil) {
		return nil
	}
	r := c
	for r.Parent() != nil {
		r = r.Parent()
	}
	for _, pl := range w.pops {
		if pl.c == r {
			return pl
		}
	}
	if w.tipPop != nil && w.tipPop.c == r {
		return w.tipPop
	}
	return nil
}

// onPopSurface reports whether layer root r is painted on a popup surface
// rather than by the window — including one about to get its surface on
// the next frame, which the window must not paint in the meantime.
func (w *Window) onPopSurface(r widget.Component) bool {
	if r == nil || !w.popupsOnSurfaces() {
		return false
	}
	if r == w.tooltip {
		return true
	}
	for c := w.popup; c != nil; c = widget.CascadeOf(c) {
		if c == r {
			return true
		}
	}
	return false
}

// popupPlaced adopts a place the window system chose on its own.
func (w *Window) popupPlaced(s platform.Surface) {
	for _, pl := range w.allPops() {
		if pl.surf == s {
			w.adoptPop(pl)
			return
		}
	}
}

// popupDone answers the window system taking a popup down (a click in
// another application): a menu is dismissed, a tooltip hidden.
func (w *Window) popupDone(s platform.Surface) {
	if w.tipPop != nil && w.tipPop.surf == s {
		w.dismissTooltip()
		return
	}
	for _, pl := range w.pops {
		if pl.surf == s {
			w.DismissPopup()
			return
		}
	}
}

func (w *Window) allPops() []*popLayer {
	if w.tipPop == nil {
		return w.pops
	}
	return append(append([]*popLayer(nil), w.pops...), w.tipPop)
}

// popsDirty reports whether a popup surface has to be painted.
func (w *Window) popsDirty() bool {
	for _, pl := range w.allPops() {
		if pl.dirty {
			return true
		}
	}
	return false
}

// markPopsDirty repaints every popup surface on the next frame.
func (w *Window) markPopsDirty() {
	for _, pl := range w.allPops() {
		pl.dirty = true
	}
}

// paintPops paints and presents every popup surface that needs it. A popup
// is small, so it is painted whole: its surface is its own, and there is
// nothing under it to keep.
func (w *Window) paintPops() {
	for _, pl := range w.allPops() {
		if !pl.dirty {
			continue
		}
		pl.dirty = false
		if perfOn() {
			t0 := time.Now()
			w.paintPop(pl)
			perfFrame(t0, "popup")
			continue
		}
		w.paintPop(pl)
	}
}

// paintPop paints one popup into its surface: the shadow, the component at
// the place it has in the window's coordinates (moved into the surface by
// the surface's origin), and then its silhouette cut out, as a shaped
// window's is.
func (w *Window) paintPop(pl *popLayer) {
	ctx := platform.NewPaintContext(pl.surf)
	if ctx == nil {
		return
	}
	lk := w.layerLook(pl.c)
	o := pl.surf.Origin()
	sil := w.popShape(pl)
	backdrop := style.BackdropNone
	if w.popGlass(lk) {
		backdrop = style.BackdropGlass
	}
	prev := style.SetLayerBackdrop(backdrop)
	defer style.SetLayerBackdrop(prev)
	ctx.Clear(paintengine2d.Transparent)
	ctx.Save()
	ctx.Translate(-o.X, -o.Y)
	b := pl.c.Bounds()
	if sil != nil && !pl.sil.derived {
		w.paintPopShade(ctx, pl, lk, sil)
	} else if !w.state.Solid {
		w.paintShadow(ctx, pl.c, pl.kind, nil)
	}
	widget.PaintTree(pl.c, ctx, nil)
	ctx.Restore()
	if sil != nil && !pl.sil.derived {
		off := paintengine2d.Pt(b.Min.X-o.X, b.Min.Y-o.Y)
		if pl.sil.wipe == nil {
			pl.sil.wipe = shapeWipePath(sil, off)
		}
		if pl.sil.eraser == nil {
			pl.sil.eraser = shapeEraserImage(sil)
		}
		punchSilhouette(ctx, sil, off, pl.sil.wipe, pl.sil.eraser, nil)
	}
	_ = pl.surf.Present(nil)
}

// paintPopShade is a shaped popup's shadow: its silhouette blurred into the
// margin, in the colour of the look's own popup shadow — a rectangle's
// shadow under a disc would give the disc away.
func (w *Window) paintPopShade(ctx *paintengine2d.Context, pl *popLayer, lk style.LookAndFeel, r *platform.ShapeRaster) {
	m := pl.frame.Margin
	if m.Zero() {
		return
	}
	if pl.sil.shade == nil {
		col := probePopupShadowColor(lk, pl.kind)
		pl.sil.shade = buildShadeImage(r, m, col)
	}
	img := pl.sil.shade
	if img == nil {
		return
	}
	b := pl.c.Bounds()
	dst := paintengine2d.XYWH(b.Min.X-float32(m.Left), b.Min.Y-float32(m.Top), float32(img.Width), float32(img.Height))
	ctx.DrawImageRectPaint(img, paintengine2d.XYWH(0, 0, float32(img.Width), float32(img.Height)), dst,
		paintengine2d.Paint{Color: paintengine2d.White, Filter: paintengine2d.FilterNearest})
}

// probePopupShadowColor reads the densest pixel of the look's own popup
// shadow, the colour a blurred silhouette is tinted with.
func probePopupShadowColor(lk style.LookAndFeel, kind style.PopupKind) paintengine2d.Color {
	in := style.PopupShadowOf(lk, kind)
	ml, mt := max(int(math.Ceil(float64(in.Left))), 1), max(int(math.Ceil(float64(in.Top))), 1)
	pw, ph := ml*2+16, mt*2+16
	img := paintengine2d.NewImage(pw, ph)
	ctx := paintengine2d.NewContext(img)
	if ctx == nil {
		return paintengine2d.Transparent
	}
	ctx.Clear(paintengine2d.Transparent)
	style.DrawPopupShadowOf(lk, ctx, paintengine2d.XYWH(float32(ml), float32(mt), float32(pw-2*ml), float32(ph-2*mt)), kind)
	c := img.NRGBAAt(ml-1, ph/2)
	return paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255)
}
