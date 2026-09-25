package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
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

// A bar can put a word in front of a control to say what it sets — the
// Settings preview's second bar is "Icons [ ] Size [ ] Corners [ ]" —
// and the word is text, not a button: nothing hovers it and the bar's
// arrow keys step over it.
func TestToolBarLabelsTheControlBesideIt(t *testing.T) {
	combo := NewComboBox([]string{"Classic", "Lucide"}, 0, nil)
	combo.MinWidth = 1
	bar := NewToolBar(
		ToolText("One", nil),
		ToolLabel("Icons"),
		ToolWidget(combo),
		ToolStretch(),
	)
	bar.SetHost(&fakeWindow{look: style.DarkLook()})
	h := bar.Measure(layout.Loose(600, 100)).Y
	bar.Arrange(paintengine2d.XYWH(0, 0, 600, h))
	rects := bar.itemRects()

	// It measures to its text, on the bar and centred in it.
	f := style.ControlFontOf(bar.Look(), style.RoleTool)
	if got, want := rects[1].Dx(), f.Advance("Icons"); got != want {
		t.Errorf("the word measured %v wide, want its text %v", got, want)
	}
	if rects[1].Min.Y < 0 || rects[1].Max.Y > h {
		t.Errorf("the word at %v is not inside the bar (height %v)", rects[1], h)
	}
	// It belongs to the control after it: the space between the two is
	// half what the bar leaves between anything else, so the eye pairs
	// them rather than pairing the word with the tool before it.
	before := rects[1].Min.X - rects[0].Max.X
	after := combo.Bounds().Min.X - rects[1].Max.X
	if after >= before {
		t.Errorf("the word is %v from its control and %v from the tool before it", after, before)
	}
	// It is not a tool: the arrow keys walk from the button to nothing
	// else, and the pointer finds no item over the word.
	if got := lastTool(bar.items); got != 0 {
		t.Errorf("the last tool the bar walks is item %d, want the One button", got)
	}
	if got := bar.itemAt(paintengine2d.Pt(rects[1].Min.X+1, h*0.5)); got != -1 {
		t.Errorf("the pointer found item %d over the word", got)
	}
	// A screen reader reads it as the static text it is, beside the
	// control's own name.
	var label *a11y.Node
	for _, n := range bar.AccessibleItems() {
		if n.Role == a11y.RoleLabel {
			label = n
		}
	}
	if label == nil || label.Name != "Icons" {
		t.Errorf("the word is not static text in the tree: %+v", label)
	}
}

// A bar too narrow for everything drops its words before its controls,
// the same way it drops its tools: a combo box the user cannot reach is
// worse than one whose name is only in its tooltip.
func TestToolBarDropsWordsBeforeControls(t *testing.T) {
	combo := NewComboBox([]string{"Classic"}, 0, nil)
	combo.MinWidth = 1
	bar := NewToolBar(
		ToolLabel("Icons"),
		ToolWidget(combo),
		ToolStretch(),
	)
	bar.SetHost(&fakeWindow{look: style.DarkLook()})
	h := bar.Measure(layout.Loose(600, 100)).Y
	wide := bar.Measure(layout.Unbounded()).X

	bar.Arrange(paintengine2d.XYWH(0, 0, wide, h))
	if bar.itemRects()[0].Empty() {
		t.Fatal("the word is gone from a bar wide enough for it")
	}
	bar.Arrange(paintengine2d.XYWH(0, 0, wide-20, h))
	if !bar.itemRects()[0].Empty() {
		t.Error("the word stayed on a bar too narrow for it")
	}
	if box := combo.Bounds(); box.Dx() < combo.Measure(layout.Unbounded()).X {
		t.Errorf("the control was squeezed to %v of the %v it asked for", box.Dx(), combo.Measure(layout.Unbounded()).X)
	}
	// And a dropped word is out of the tree with it: a name read out with
	// no box to point at is worse than nothing.
	for _, n := range bar.AccessibleItems() {
		if n.Role == a11y.RoleLabel {
			t.Errorf("a dropped word is still in the tree: %q", n.Name)
		}
	}
}
