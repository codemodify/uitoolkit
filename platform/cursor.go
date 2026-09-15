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
	// The eight window-edge resize shapes (a client-side frame's resize
	// border): north, south, east, west and the corners.
	CursorResizeN
	CursorResizeS
	CursorResizeE
	CursorResizeW
	CursorResizeNE
	CursorResizeNW
	CursorResizeSE
	CursorResizeSW
	// CursorMove is the four-way move shape; CursorGrab / CursorGrabbing
	// are the open and closed hand.
	CursorMove
	CursorGrab
	CursorGrabbing
)

var cursorNames = [...]string{
	CursorDefault: "default", CursorColResize: "col-resize", CursorRowResize: "row-resize",
	CursorText: "text", CursorResizeN: "n-resize", CursorResizeS: "s-resize",
	CursorResizeE: "e-resize", CursorResizeW: "w-resize", CursorResizeNE: "ne-resize",
	CursorResizeNW: "nw-resize", CursorResizeSE: "se-resize", CursorResizeSW: "sw-resize",
	CursorMove: "move", CursorGrab: "grab", CursorGrabbing: "grabbing",
}

func (c Cursor) String() string {
	if c >= 0 && int(c) < len(cursorNames) {
		return cursorNames[c]
	}
	return "unknown"
}

// wp_cursor_shape_device_v1.shape (cursor-shape-v1).
const (
	wlShapeDefault   uint32 = 1
	wlShapePointer   uint32 = 4
	wlShapeText      uint32 = 9
	wlShapeMove      uint32 = 13
	wlShapeGrab      uint32 = 16
	wlShapeGrabbing  uint32 = 17
	wlShapeEResize   uint32 = 18
	wlShapeNResize   uint32 = 19
	wlShapeNEResize  uint32 = 20
	wlShapeNWResize  uint32 = 21
	wlShapeSResize   uint32 = 22
	wlShapeSEResize  uint32 = 23
	wlShapeSWResize  uint32 = 24
	wlShapeWResize   uint32 = 25
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
	case CursorResizeN:
		return wlShapeNResize
	case CursorResizeS:
		return wlShapeSResize
	case CursorResizeE:
		return wlShapeEResize
	case CursorResizeW:
		return wlShapeWResize
	case CursorResizeNE:
		return wlShapeNEResize
	case CursorResizeNW:
		return wlShapeNWResize
	case CursorResizeSE:
		return wlShapeSEResize
	case CursorResizeSW:
		return wlShapeSWResize
	case CursorMove:
		return wlShapeMove
	case CursorGrab:
		return wlShapeGrab
	case CursorGrabbing:
		return wlShapeGrabbing
	default:
		return wlShapeDefault
	}
}

// edgeThemeNames are the cursor-theme names of the resize and move shapes,
// the CSS / freedesktop name first, then the legacy X cursor-font name.
var edgeThemeNames = map[Cursor][]string{
	CursorResizeN:  {"n-resize", "top_side", "size_ver"},
	CursorResizeS:  {"s-resize", "bottom_side", "size_ver"},
	CursorResizeE:  {"e-resize", "right_side", "size_hor"},
	CursorResizeW:  {"w-resize", "left_side", "size_hor"},
	CursorResizeNE: {"ne-resize", "top_right_corner", "size_bdiag"},
	CursorResizeNW: {"nw-resize", "top_left_corner", "size_fdiag"},
	CursorResizeSE: {"se-resize", "bottom_right_corner", "size_fdiag"},
	CursorResizeSW: {"sw-resize", "bottom_left_corner", "size_bdiag"},
	CursorMove:     {"move", "fleur", "size_all", "all-scroll"},
	CursorGrab:     {"grab", "openhand", "hand1"},
	CursorGrabbing: {"grabbing", "closedhand", "fleur"},
}

func waylandThemeCursorNames(c Cursor) []string {
	if names, ok := edgeThemeNames[c]; ok {
		return names
	}
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
	if names, ok := edgeThemeNames[c]; ok {
		// The legacy name first: every X cursor theme ships it.
		return append([]string{names[1], names[0]}, names[2:]...)
	}
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
	case CursorResizeN:
		return 138 // XC_top_side
	case CursorResizeS:
		return 16 // XC_bottom_side
	case CursorResizeE:
		return 96 // XC_right_side
	case CursorResizeW:
		return 70 // XC_left_side
	case CursorResizeNE:
		return 136 // XC_top_right_corner
	case CursorResizeNW:
		return 134 // XC_top_left_corner
	case CursorResizeSE:
		return 14 // XC_bottom_right_corner
	case CursorResizeSW:
		return 12 // XC_bottom_left_corner
	case CursorMove, CursorGrab, CursorGrabbing:
		return 52 // XC_fleur
	default:
		return 68 // XC_left_ptr
	}
}

const (
	winIDCArrow    = 32512
	winIDCIbeam    = 32513
	winIDCSizeNWSE = 32642
	winIDCSizeNESW = 32643
	winIDCSizeWE   = 32644
	winIDCSizeNS   = 32645
	winIDCSizeAll  = 32646
	winIDCHand     = 32649
)

func win32CursorID(c Cursor) uintptr {
	switch c {
	case CursorColResize, CursorResizeE, CursorResizeW:
		return winIDCSizeWE
	case CursorRowResize, CursorResizeN, CursorResizeS:
		return winIDCSizeNS
	case CursorResizeNW, CursorResizeSE:
		return winIDCSizeNWSE
	case CursorResizeNE, CursorResizeSW:
		return winIDCSizeNESW
	case CursorMove, CursorGrabbing:
		return winIDCSizeAll
	case CursorGrab:
		return winIDCHand
	case CursorText:
		return winIDCIbeam
	default:
		return winIDCArrow
	}
}

// darwinCursorKind picks an NSCursor: AppKit has left/right and up/down
// resize cursors but no public diagonal ones, so corners keep the arrow.
func darwinCursorKind(c Cursor) int {
	switch c {
	case CursorColResize, CursorResizeE, CursorResizeW:
		return 1
	case CursorRowResize, CursorResizeN, CursorResizeS:
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
