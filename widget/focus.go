package widget

// MarkKeyboardFocus records keyboard-originated focus (Tab, arrows, mnemonics).
func MarkKeyboardFocus(c Component) {
	type hint interface{ MarkKeyboardFocus() }
	if v, ok := c.(hint); ok {
		v.MarkKeyboardFocus()
	}
}

// MarkPointerFocus records pointer-originated focus (mouse / touch).
func MarkPointerFocus(c Component) {
	type hint interface{ MarkPointerFocus() }
	if v, ok := c.(hint); ok {
		v.MarkPointerFocus()
	}
}
