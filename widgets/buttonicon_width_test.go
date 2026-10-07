package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A button with a mark is one strip wider than one without, not two.
//
// The engine draws a push button's label and every engine centres it in the box
// it is handed, so a widget that wanted a mark at the leading edge could not
// move the label without taking the drawing away from the engine — and losing
// the era's treatment of it, which is Clearlooks' emboss, Win95's shadow, Aqua's
// gel text. So it reserved a strip at *each* end and let the centred label land
// between them: about 64 device pixels a button, half of it symmetry. Rows that
// fitted a narrow window stopped fitting as soon as their buttons got marks, and
// a mail client folded those rows to cope.
//
// The engine is handed the mark now, as DrawToolButton always was, and centres
// the label in style.ButtonLabelBox instead of in the button.
func TestAButtonWithAMarkIsOneStripWider(t *testing.T) {
	for _, pack := range []string{"", "luna", "win95", "adwaita", "aqua", "breeze"} {
		lk := style.DarkLook()
		if pack != "" {
			p, ok := style.LoadTheme(pack)
			if !ok {
				continue
			}
			lk = p.Look()
		}
		for _, label := range []string{"No", "Maybe", "Remove account"} {
			plain, marked := NewButton(label, nil), NewButton(label, nil)
			marked.Icon = style.IconSave
			for _, b := range []*Button{plain, marked} {
				b.SetHost(&fakeWindow{look: lk})
			}
			p := plain.Measure(layout.Unbounded())
			m := marked.Measure(layout.Unbounded())
			want := style.ButtonIconWidth(lk, p.Y)
			if got := m.X - p.X; got != want {
				t.Errorf("%s %q: a mark added %.0f, want one strip (%.0f)", pack, label, got, want)
			}
			// And one strip is what the era says, not a number the widget made up.
			_, side, gap := style.ToolButtonChromeFor(lk, p.Y)
			if want != side+gap {
				t.Errorf("%s: a strip is %.0f, want side %.0f + gap %.0f", pack, want, side, gap)
			}
		}
	}
}

// An icon-only button is still square-ish: the mark is centred and no label box
// is taken out of it.
func TestAMarkOnlyButtonIsNotWidenedByALabelBox(t *testing.T) {
	lk := style.DarkLook()
	b := NewButton("", nil)
	b.Icon = style.IconSave
	b.SetHost(&fakeWindow{look: lk})
	sz := b.Measure(layout.Unbounded())
	full := paintengine2d.XYWH(0, 0, sz.X, sz.Y)
	if box := style.ButtonLabelBox(lk, full, style.ButtonDraw{Icon: style.IconSave}); box != full {
		t.Errorf("a mark-only button reserved a label box: %v of %v", box, full)
	}
}
