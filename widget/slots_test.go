package widget

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// Slots places an app's components on a skin layout's slots and does
// nothing else: the order, the names and the keyboard are the app's.

// slotSkinLook installs a skin whose one sprite is a disc and whose one
// layout, "deck.probe", has a play slot painted with it and a stop slot
// painted by its control.
func slotSkinLook(t *testing.T) style.LookAndFeel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := filepath.Join(style.SkinsDir(), "slotprobe")
	if err := os.MkdirAll(filepath.Join(dir, "art"), 0o755); err != nil {
		t.Fatal(err)
	}
	for file, n := range map[string]float32{"sheet.png": 1, "sheet2.png": 2} {
		img := paintengine2d.NewImage(int(24*n), int(24*n))
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		ctx.DrawCircle(paintengine2d.Pt(12*n, 12*n), 12*n, paintengine2d.Fill(paintengine2d.RGB(0.8, 0.8, 0.9)))
		var buf bytes.Buffer
		if err := img.WritePNG(&buf); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "art", file), buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	doc := `{"skin": 1, "base": "breeze-night",
		"sheets": {"chrome": {"1x": "art/sheet.png", "2x": "art/sheet2.png"}},
		"sprites": {"disc": {"at": [0, 0, 24, 24]}},
		"layouts": {"deck.probe": {"size": [200, 100], "slots": {
			"play": {"at": [20, 30, 24, 24], "art": "disc"},
			"stop": {"at": [60, 30, 40, 20]}}}}}`
	if err := os.WriteFile(filepath.Join(dir, style.SkinFile), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	style.InvalidateSkinCache()
	p, ok := style.LoadTheme("slotprobe")
	if !ok {
		t.Fatal("the probe skin does not load")
	}
	return p.Look()
}

// namedBox is a focusable component with a name, for the tab order and
// the accessibility tree.
type namedBox struct{ Base }

func newNamed(name string) *namedBox {
	n := &namedBox{}
	n.Init(n)
	n.SetWantsFocus(true)
	n.SetAccessibleName(name)
	return n
}

func (n *namedBox) Describe(node *a11y.Node) { node.Role = a11y.RoleButton }

func TestSlotsPlaceWhatIsBoundAndNothingElse(t *testing.T) {
	lk := slotSkinLook(t)
	root := newShapedBox(paintengine2d.XYWH(0, 0, 400, 200))
	root.SetLook(lk)
	// Added in one order, bound in another: the order they were added in
	// is the tab order and the tree order, whatever the slots say.
	play, stop, spare := newNamed("Play"), newNamed("Stop"), newNamed("Spare")
	root.Add(stop)
	root.Add(play)
	root.Add(spare)
	before := treeOf(root)

	s := NewSlots("deck.probe").Bind("play", play).Bind("stop", stop).Bind("no-such-slot", spare)
	if !s.Available(lk) {
		t.Fatal("the layout is not available")
	}
	box := root.LocalBounds()
	unplaced := s.Arrange(lk, box)
	if len(unplaced) != 1 || unplaced[0] != Component(spare) {
		t.Fatalf("unplaced %v, want only the component whose slot the skin leaves out", unplaced)
	}
	if !spare.Visible() {
		t.Fatal("the skin hid a component: whether a control exists is the app's decision")
	}
	for name, c := range map[string]Component{"play": play, "stop": stop} {
		want, _ := style.SkinSlotRect(lk, "deck.probe", name, box)
		if c.Bounds() != want {
			t.Errorf("%s at %v, want its slot %v", name, c.Bounds(), want)
		}
	}
	if after := treeOf(root); after != before {
		t.Errorf("placing changed the tree:\n before %s\n after  %s", before, after)
	}
	if FirstFocusable(root) != Component(stop) {
		t.Error("the first control on the keyboard is not the first one added")
	}
	if sz, ok := s.Measure(lk, layout.Unbounded()); !ok || sz.X <= 0 {
		t.Errorf("measure %v %v", sz, ok)
	}
}

func TestSlotsInALookWithNoLayoutPlaceNothing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := style.LoadTheme("breeze-night")
	if !ok {
		t.Skip("no breeze-night")
	}
	a := newNamed("A")
	a.SetBounds(paintengine2d.XYWH(1, 2, 3, 4))
	s := NewSlots("deck.probe").Bind("play", a)
	if s.Available(p.Look()) {
		t.Fatal("a look that is not a skin has the layout")
	}
	if got := s.Arrange(p.Look(), paintengine2d.XYWH(0, 0, 100, 100)); len(got) != 1 || a.Bounds() != paintengine2d.XYWH(1, 2, 3, 4) {
		t.Fatalf("with no layout, Arrange moved %v or returned %v", a.Bounds(), got)
	}
}

// A control in a slot with art takes the pointer on that art: the slot's
// silhouette is what an ArtShape component answers.
type slotKey struct {
	Base
	layout, slot string
}

func (k *slotKey) ArtShape(lk style.LookAndFeel, size paintengine2d.Point) *style.Silhouette {
	return style.SkinSlotShape(lk, size, k.layout, k.slot)
}

func TestAControlInASlotTakesTheSlotArtsShape(t *testing.T) {
	lk := slotSkinLook(t)
	root := newShapedBox(paintengine2d.XYWH(0, 0, 400, 200))
	root.SetLook(lk)
	key := &slotKey{layout: "deck.probe", slot: "play"}
	key.Init(key)
	root.Add(key)
	NewSlots("deck.probe").Bind("play", key).Arrange(lk, root.LocalBounds())
	b := key.Bounds()
	if HitRoot(root, paintengine2d.Pt(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)) != Component(key) {
		t.Fatal("the middle of the key is not the key")
	}
	if HitRoot(root, paintengine2d.Pt(b.Min.X+1, b.Min.Y+1)) == Component(key) {
		t.Error("the corner of a round key's slot is the key")
	}
}

// treeOf is the accessibility tree under root as one line: roles and names
// in tree order.
func treeOf(root Component) string {
	n := &a11y.Node{}
	AccessibleTree(n, root)
	var out string
	var walk func(*a11y.Node)
	walk = func(n *a11y.Node) {
		out += n.Role.String() + ":" + n.Name + " "
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}
