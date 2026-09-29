//go:build darwin && cgo

package platform

// The seams the AppKit backend implements, asserted — see
// seams_linux.go for why the compiler rather than a table in the docs.
var (
	_ WindowFrame    = (*akSurface)(nil)
	_ WindowGeometry = (*akSurface)(nil)
	_ CursorSurface  = (*akSurface)(nil)
	_ PopupSurface   = (*akSurface)(nil)
	_ PopupOpener    = (*akSurface)(nil)
	_ PopupWorkArea  = (*akSurface)(nil)
	_ DragSurface    = (*akSurface)(nil)
	_ DropNegotiator = (*akSurface)(nil)
	_ DropReceiver   = (*akSurface)(nil)
	_ IMESurface     = (*akSurface)(nil)
	// macOS has both, and is the only one that does.
	_ SecureInputSurface    = (*akSurface)(nil)
	_ CaptureExcludeSurface = (*akSurface)(nil)
)
