package app

// What an app may tell the frame about one of its windows.
//
// A look states one frame for every window it dresses, and that is almost
// always right. Two things are the app's to say instead, because only the
// app knows them: what a window *is*, so a look that dresses its windows
// differently — a skin whose equaliser has a tab for a header and no title
// band — can give it the frame it drew for it; and how tall a caption the
// window wants, for a compact mode that has no room for the look's band.
//
// Neither changes behaviour. The window keeps its title, its caption
// buttons, its keyboard and its accessibility node; only the frame's
// painting and its measurements follow.

// SetFrameRole names what this window is to the app — "equaliser",
// "playlist" — so a look that states a frame for that role dresses it in
// that frame (a skin's window variants, docs/skins.md). A look with no
// frame for the role, and every look that is not a skin, gives the window
// its one frame, exactly as it does with no role at all. "" drops it.
func (w *Window) SetFrameRole(role string) {
	if w == nil || w.frameRole == role {
		return
	}
	w.frameRole = role
	w.frameChanged()
}

// FrameRole is the role given to SetFrameRole.
func (w *Window) FrameRole() string {
	if w == nil {
		return ""
	}
	return w.frameRole
}

// SetCaptionHeight asks for a caption h logical pixels tall instead of the
// look's own, as a compact mode that has no room for the look's band does.
// The caption's buttons shrink to stand in it; the band never drops below
// the eight pixels every frame keeps. 0 gives the caption back to the look.
func (w *Window) SetCaptionHeight(h float32) {
	if w == nil {
		return
	}
	h = max(h, 0)
	if w.captionAsk == h {
		return
	}
	w.captionAsk = h
	w.frameChanged()
}

// CaptionHeight is the caption height given to SetCaptionHeight (0: the
// look's own).
func (w *Window) CaptionHeight() float32 {
	if w == nil {
		return 0
	}
	return w.captionAsk
}

// frameChanged re-reads the frame after something the frame is asked with
// changed: the caption, its buttons and the silhouette are the look's
// answer to the window's state, and the state has moved.
func (w *Window) frameChanged() {
	if w.caption != nil {
		w.rebuildCaption()
	}
	w.RequestLayout()
}
