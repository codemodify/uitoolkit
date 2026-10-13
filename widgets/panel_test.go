package widgets

import (
	"github.com/codemodify/paintengine2d"
	"testing"

	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// **A panel's floor holds its chrome.**
//
// A panel answered no MinWidth of its own, so the floor a window took
// from it was its body's alone — without the frame, the padding or the
// title it puts round that body. secretvault sized its window from a
// window panel and had to add the difference itself, measured as the
// panel's laid-out width less its content's (item 89).
func TestAPanelsFloorHoldsItsChrome(t *testing.T) {
	lk := style.DarkLook()
	for _, tc := range []struct {
		name  string
		build func() *Panel
	}{
		{"plain", func() *Panel { return NewPanel("") }},
		{"titled", func() *Panel { return NewPanel("Grants") }},
		{"window", func() *Panel { p := NewPanel("Grants"); p.Window = true; return p }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A splitter: unbounded it measures a fixed 320x200, so
			// MinWidthOf answers through its single-child
			// pass-through rule rather than by probing the panel —
			// which is the path that dropped the chrome, and the one
			// secretvault's editor took.
			left, right := NewButton("Items", nil), NewButton("Editor", nil)
			body := NewSplitter(SplitColumns, left, right)
			body.SetLook(lk)
			p := tc.build()
			p.SetLook(lk)
			p.Add(body)
			p.Content().AddFlex(body, 1)
			p.Arrange(paintengine2d.XYWH(0, 0, 800, 400))

			floor := widget.MinWidthOf(p)
			inner := widget.MinWidthOf(body)
			if floor < inner {
				t.Fatalf("the panel's floor is %v and its body's %v", floor, inner)
			}
			// The chrome is the difference, and a panel that draws some
			// must ask for it.
			in := p.insets()
			if want := inner + in.Left + in.Right; floor != want {
				t.Errorf("the floor is %v, want the body's %v and the %v of chrome round it", floor, inner, in.Left+in.Right)
			}
			if tc.name != "plain" && in.Left+in.Right <= 0 {
				t.Skip("this look draws no chrome round a panel")
			}
		})
	}
}
