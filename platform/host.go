package platform

// HostWindow is an optional Surface capability: raise / show / hide the
// native window. Offscreen implements it as a visibility flag. X11 maps
// and sends _NET_ACTIVE_WINDOW. Wayland Hide drops the xdg_toplevel
// role (and may set_minimized first); Show remaps the role and requests
// xdg_activation_v1 when the compositor advertises it.
type HostWindow interface {
	Raise()
	Show()
	Hide()
	Visible() bool
}

// HostMover is an optional Surface capability: move the native window
// in root/screen coordinates (X11). Wayland and offscreen no-op.
type HostMover interface {
	Move(x, y int)
}

// RaiseSurface shows and raises s when it implements [HostWindow].
func RaiseSurface(s Surface) {
	if s == nil {
		return
	}
	if h, ok := s.(HostWindow); ok {
		h.Show()
		h.Raise()
	}
}

// HideSurface unmaps / minimizes s when supported.
func HideSurface(s Surface) {
	if s == nil {
		return
	}
	if h, ok := s.(HostWindow); ok {
		h.Hide()
	}
}

// SurfaceVisible reports true when s has no hide protocol or is mapped.
func SurfaceVisible(s Surface) bool {
	if s == nil {
		return false
	}
	if h, ok := s.(HostWindow); ok {
		return h.Visible()
	}
	return !s.Closed()
}

// MoveSurface places s at root/screen (x, y) when the backend can.
func MoveSurface(s Surface, x, y int) {
	if s == nil {
		return
	}
	if m, ok := s.(HostMover); ok {
		m.Move(x, y)
	}
}
