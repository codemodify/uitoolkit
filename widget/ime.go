package widget

import "github.com/codemodify/paintengine2d"

// IMETarget is implemented by text widgets that show composition.
type IMETarget interface {
	IMEPreedit(s string, caret int)
	IMECommit(s string)
	IMEReset()
	IMEDeleteSurrounding(beforeBytes, afterBytes int)
	IMESurrounding() (text string, cursor, anchor int)
	IMECaretRect() paintengine2d.Rect
}
