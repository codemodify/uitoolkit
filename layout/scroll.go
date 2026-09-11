package layout

// MaxScroll is the largest legal offset: max(0, content − viewport).
func MaxScroll(content, viewport float32) float32 {
	m := content - viewport
	if m < 0 {
		return 0
	}
	return m
}

// ClampScroll keeps offset in [0, MaxScroll(content, viewport)].
// There is no rubber-band / infinite past-the-end range.
func ClampScroll(offset, content, viewport float32) float32 {
	if offset < 0 {
		return 0
	}
	if mx := MaxScroll(content, viewport); offset > mx {
		return mx
	}
	return offset
}
