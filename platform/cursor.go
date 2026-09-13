package platform

// Cursor is a pointer shape for Window.SetCursor / Surface.SetCursor.
// Backends apply a host-provided image: compositor cursor-shape or
// XCURSOR theme on Wayland, Xcursor / X font cursors on X11,
// LoadCursorW on Win32, NSCursor on AppKit. Offscreen records the
// logical value only. No backend draws a homemade cursor bitmap.
type Cursor int

const (
	CursorDefault Cursor = iota
	CursorColResize
	CursorRowResize
	CursorText
)

func (c Cursor) String() string {
	switch c {
	case CursorDefault:
		return "default"
	case CursorColResize:
		return "col-resize"
	case CursorRowResize:
		return "row-resize"
	case CursorText:
		return "text"
	default:
		return "unknown"
	}
}

// wp_cursor_shape_device_v1.shape (cursor-shape-v1).
const (
	wlShapeDefault   uint32 = 1
	wlShapePointer   uint32 = 4
	wlShapeText      uint32 = 9
	wlShapeEWResize  uint32 = 26
	wlShapeNSResize  uint32 = 27
	wlShapeColResize uint32 = 30
	wlShapeRowResize uint32 = 31
)

func waylandCursorShape(c Cursor) uint32 {
	switch c {
	case CursorColResize:
		return wlShapeColResize
	case CursorRowResize:
		return wlShapeRowResize
	case CursorText:
		return wlShapeText
	default:
		return wlShapeDefault
	}
}

func waylandThemeCursorNames(c Cursor) []string {
	switch c {
	case CursorColResize:
		return []string{"col-resize", "ew-resize", "sb_h_double_arrow", "split_h"}
	case CursorRowResize:
		return []string{"row-resize", "ns-resize", "sb_v_double_arrow", "split_v"}
	case CursorText:
		return []string{"text", "xterm", "ibeam", "IBeam"}
	default:
		return []string{"left_ptr", "default", "arrow", "top_left_arrow"}
	}
}

func x11ThemeCursorNames(c Cursor) []string {
	switch c {
	case CursorColResize:
		return []string{"sb_h_double_arrow", "col-resize", "ew-resize"}
	case CursorRowResize:
		return []string{"sb_v_double_arrow", "row-resize", "ns-resize"}
	case CursorText:
		return []string{"xterm", "text", "ibeam"}
	default:
		return []string{"left_ptr", "default", "arrow"}
	}
}

func x11FontCursorShape(c Cursor) uint {
	switch c {
	case CursorColResize:
		return 108 // XC_sb_h_double_arrow
	case CursorRowResize:
		return 116 // XC_sb_v_double_arrow
	case CursorText:
		return 152 // XC_xterm
	default:
		return 68 // XC_left_ptr
	}
}

const (
	winIDCArrow  = 32512
	winIDCIbeam  = 32513
	winIDCSizeWE = 32644
	winIDCSizeNS = 32645
)

func win32CursorID(c Cursor) uintptr {
	switch c {
	case CursorColResize:
		return winIDCSizeWE
	case CursorRowResize:
		return winIDCSizeNS
	case CursorText:
		return winIDCIbeam
	default:
		return winIDCArrow
	}
}

func darwinCursorKind(c Cursor) int {
	switch c {
	case CursorColResize:
		return 1
	case CursorRowResize:
		return 2
	case CursorText:
		return 3
	default:
		return 0
	}
}

// CursorSurface is implemented by backends that can change the pointer shape.
type CursorSurface interface {
	SetCursor(c Cursor)
}

// SetCursor applies c when the surface implements CursorSurface.
func SetCursor(s Surface, c Cursor) {
	if cs, ok := s.(CursorSurface); ok {
		cs.SetCursor(c)
	}
}
