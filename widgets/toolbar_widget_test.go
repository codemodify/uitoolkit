package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A bar can carry a control of the application's own, and the free space
// before it pins that control to the bar's right-hand end: the icon
// choosers on the Settings preview's tool bar are laid out this way.
func TestToolBarCarriesAControlAtItsRightEnd(t *testing.T) {
	combo := NewComboBox([]string{"Classic", "Lucide"}, 0, nil)
	bar := NewToolBar(
		ToolIconBtn(style.IconNew, "", nil),
		ToolIconBtn(style.IconSave, "", nil),
		ToolStretch(),
		ToolWidget(combo),
	)
	bar.SetHost(&fakeWindow{look: style.DarkLook()})

	// A bar with free space in it takes the width it is offered rather
	// than only the width of its items: its right edge is where its
	// controls hang from.
	if got := bar.Measure(layout.Loose(600, 100)).X; got != 600 {
		t.Errorf("a bar with a stretch measured %v wide of 600 offered", got)
	}
	bar.Arrange(paintengine2d.XYWH(0, 0, 600, bar.Measure(layout.Loose(600, 100)).Y))
	box := combo.Bounds()
	if box.Empty() {
		t.Fatal("the control on the bar was never arranged")
	}
	if in := style.ToolBarInsetsOf(bar.Look()); box.Max.X < 600-in.Right-1 {
		t.Errorf("the control ends at %v, not at the bar's right edge (600 less %v)", box.Max.X, in.Right)
	}
	// It is on the bar, not under it: the bar is as tall as the control
	// it carries when that is taller than a row of tool buttons.
	if box.Min.Y < 0 || box.Max.Y > bar.LocalBounds().Dy() {
		t.Errorf("the control at %v is not inside the bar %v", box, bar.LocalBounds())
	}
	// The bar's own keys walk its buttons and step over the control; the
	// control is a focus stop in its own right (it is a child).
	if got := lastTool(bar.items); got != 1 {
		t.Errorf("the last tool the bar walks is item %d, want the Save button", got)
	}
	if n := len(widget.Focusables(bar)); n < 2 {
		t.Errorf("the bar and its control are %d focus stops", n)
	}
}

// A bar too narrow for everything drops the tools nearest the free space
// rather than letting the controls after it fall off the end: half a
// combo box off the edge of a window is a control the user cannot use,
// while one tool button fewer is a tool button fewer.
func TestToolBarDropsToolsBeforeControls(t *testing.T) {
	combo := NewComboBox([]string{"Classic"}, 0, nil)
	bar := NewToolBar(
		ToolIconBtn(style.IconNew, "", nil),
		ToolIconBtn(style.IconOpen, "", nil),
		ToolIconBtn(style.IconSave, "", nil),
		ToolStretch(),
		ToolWidget(combo),
	)
	bar.SetHost(&fakeWindow{look: style.DarkLook()})
	h := bar.Measure(layout.Loose(600, 100)).Y
	wide := bar.Measure(layout.Unbounded()).X

	bar.Arrange(paintengine2d.XYWH(0, 0, wide, h))
	rects := bar.itemRects()
	for i := 0; i < 3; i++ {
		if rects[i].Empty() {
			t.Fatalf("tool %d is gone from a bar wide enough for everything", i)
		}
	}
	// One tool button short of the width it wants: the tool nearest the
	// free space goes, the first stays, and the combo box is still whole.
	bar.Arrange(paintengine2d.XYWH(0, 0, wide-40, h))
	rects = bar.itemRects()
	if !rects[2].Empty() {
		t.Error("the tool nearest the free space stayed on a bar too narrow for it")
	}
	if rects[0].Empty() {
		t.Error("the bar dropped its first tool before its last")
	}
	if box := combo.Bounds(); box.Dx() < combo.Measure(layout.Unbounded()).X {
		t.Errorf("the control was squeezed to %v of the %v it asked for", box.Dx(), combo.Measure(layout.Unbounded()).X)
	}
	// A dropped tool is not in the accessibility tree either: it is not
	// on screen, and a name a screen reader reads out with no box to
	// point at is worse than nothing.
	for _, n := range bar.AccessibleItems() {
		if n.Bounds.Empty() {
			t.Errorf("a dropped tool is still in the tree: %q", n.Name)
		}
	}
}

// A bar with no free space in it is the bar every other application has:
// it measures its items and no more.
func TestToolBarWithoutStretchKeepsItsWidth(t *testing.T) {
	bar := NewToolBar(ToolText("One", nil), ToolText("Two", nil))
	bar.SetHost(&fakeWindow{look: style.DarkLook()})
	if got, want := bar.Measure(layout.Loose(600, 100)).X, bar.contentW(bar.barHeight()); got != want {
		t.Errorf("a plain bar measured %v wide, want its content %v", got, want)
	}
}
