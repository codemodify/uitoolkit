package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// **A view given its text before it is laid out opens at the top.**
//
// comms-mail's reading pane showed its first message 6.3 px down, the
// top of the first line cut off by the edge; every message opened after
// it was fine. The view is given the text before the window's first
// layout, while it is still 0 px tall — and `XYWH` canonicalises a
// negative size, so a box shorter than its own padding came back 2·pad
// *tall* rather than empty. The guard against "no room" therefore never
// fired, a 26 px line was scrolled into a 20 px box that does not
// exist, and once the view grew that offset was a legal scroll
// position and stayed.
func TestAViewGivenTextBeforeItIsLaidOutOpensAtTheTop(t *testing.T) {
	for _, h := range []float32{0, 5, 10, 15, 25, 30} {
		ta := NewTextArea("", "", nil)
		ta.SetLook(style.DarkLook())
		// Laid out too small, or not at all, and given its text there.
		ta.Arrange(paintengine2d.XYWH(0, 0, 280, h))
		// Long enough that the laid-out view can still scroll: with
		// content that fits, clampScroll zeroes any offset and the
		// test would pass whatever this code did.
		lines := ""
		for i := 0; i < 40; i++ {
			lines += "A line of the message that is long enough to matter.\n"
		}
		ta.SetText(lines)
		// Then the window lays out properly.
		ta.Arrange(paintengine2d.XYWH(0, 0, 280, 260))
		ta.relayout()
		if ta.scrollY != 0 {
			t.Errorf("given its text at %vpx tall, the view opens %v down", h, ta.scrollY)
		}
	}
}

// And the inner box of a view too short for its own padding is empty,
// rather than the padding turned inside out.
func TestAnInnerBoxIsNeverNegative(t *testing.T) {
	ta := NewTextArea("x", "", nil)
	ta.SetLook(style.DarkLook())
	for _, h := range []float32{0, 1, 5, 10} {
		ta.Arrange(paintengine2d.XYWH(0, 0, 280, h))
		if got := ta.inner().Dy(); got < 0 {
			t.Errorf("a %vpx view has an inner height of %v", h, got)
		} else if h < ta.fieldPad()*2 && got != 0 {
			t.Errorf("a %vpx view, shorter than its %vpx of padding, has %v of room inside", h, ta.fieldPad()*2, got)
		}
	}
}
