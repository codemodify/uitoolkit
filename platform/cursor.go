package platform

// Cursor is a pointer shape for Window.SetCursor / Surface.SetCursor.
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
