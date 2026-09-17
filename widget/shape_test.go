package widget

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// A component's own silhouette, and the promise that not asking for one
// changes nothing: every widget in the toolkit and all 121 packs rely on a
// press anywhere in a component's box being that component's.

// shapedBox is the smallest component that can carry a silhouette.
type shapedBox struct{ Base }

func newShapedBox(b paintengine2d.Rect) *shapedBox {
	s := &shapedBox{}
	s.Init(s)
	s.SetBounds(b)
	return s
}

func TestBoundingBoxIsStillTheDefault(t *testing.T) {
	// The regression guard for every widget that never heard of shapes.
	root := newShapedBox(paintengine2d.XYWH(0, 0, 100, 100))
	if root.HitShape() != nil {
		t.Fatal("a component has no silhouette by default")
	}
	for _, p := range [][2]float32{{0, 0}, {99, 99}, {50, 50}, {0, 99}} {
		if got := HitRoot(root, paintengine2d.Pt(p[0], p[1])); got == nil {
			t.Fatalf("a press at %v missed a component with no silhouette", p)
		}
	}
	if got := HitRoot(root, paintengine2d.Pt(100, 100)); got != nil {
		t.Fatal("a press outside the box hit it")
	}
	if root.Transparent() {
		t.Fatal("a component is not see-through by default")
	}
}

func TestAHitShapeDecidesWhatIsTheComponents(t *testing.T) {
	root := newShapedBox(paintengine2d.XYWH(0, 0, 100, 100))
	root.SetHitShape(platform.ShapeEllipse(paintengine2d.XYWH(0, 0, 100, 100)))
	// The middle is the component's; the corners are not, and fall through
	// to whatever is behind — here, nothing.
	if HitRoot(root, paintengine2d.Pt(50, 50)) == nil {
		t.Fatal("the middle of the disc is not the component's")
	}
	for _, p := range [][2]float32{{1, 1}, {98, 1}, {1, 98}, {98, 98}} {
		if got := HitRoot(root, paintengine2d.Pt(p[0], p[1])); got != nil {
			t.Fatalf("the corner at %v is outside the disc but was hit", p)
		}
	}
}

func TestAHitShapeCoversItsChildrenToo(t *testing.T) {
	// Outside a component's silhouette is not its subtree's either: the
	// press goes past the whole branch, exactly as a press outside a
	// shaped window goes past the whole window.
	root := newShapedBox(paintengine2d.XYWH(0, 0, 100, 100))
	child := newShapedBox(paintengine2d.XYWH(0, 0, 20, 20))
	root.Add(child)
	if got := HitRoot(root, paintengine2d.Pt(2, 2)); got != Component(child) {
		t.Fatalf("with no silhouette the child takes the corner: %v", got)
	}
	root.SetHitShape(platform.ShapeEllipse(paintengine2d.XYWH(0, 0, 100, 100)))
	if got := HitRoot(root, paintengine2d.Pt(2, 2)); got != nil {
		t.Fatalf("the child took a press outside its parent's silhouette: %v", got)
	}
	// And inside the silhouette the child still wins.
	child.SetBounds(paintengine2d.XYWH(40, 40, 20, 20))
	if got := HitRoot(root, paintengine2d.Pt(50, 50)); got != Component(child) {
		t.Fatalf("inside the silhouette the child should still win: %v", got)
	}
}

func TestAHitShapeFuncFollowsTheSize(t *testing.T) {
	root := newShapedBox(paintengine2d.XYWH(0, 0, 100, 100))
	var built int
	root.SetHitShapeFunc(func(size paintengine2d.Point) *platform.Shape {
		built++
		return platform.ShapeEllipse(paintengine2d.XYWH(0, 0, size.X, size.Y))
	})
	if HitRoot(root, paintengine2d.Pt(50, 50)) == nil {
		t.Fatal("the middle is not the component's")
	}
	if HitRoot(root, paintengine2d.Pt(1, 1)) != nil {
		t.Fatal("the corner is the component's")
	}
	// Asking again does not rebuild: a fresh Shape each time would
	// rasterise itself each time.
	was := built
	for i := 0; i < 20; i++ {
		HitRoot(root, paintengine2d.Pt(50, 50))
	}
	if built != was {
		t.Fatalf("the silhouette was rebuilt %d times for one size", built-was+1)
	}
	// A resize does rebuild it, at the new size.
	root.SetBounds(paintengine2d.XYWH(0, 0, 200, 40))
	if HitRoot(root, paintengine2d.Pt(100, 20)) == nil {
		t.Fatal("the middle of the resized ellipse is not the component's")
	}
	if built == was {
		t.Fatal("a resize did not rebuild the silhouette")
	}
	// Dropping it gives the whole box back.
	root.SetHitShapeFunc(nil)
	if HitRoot(root, paintengine2d.Pt(1, 1)) == nil {
		t.Fatal("dropping the silhouette did not give the corner back")
	}
}

func TestTransparentRectsCollectsWhatDeclaredItself(t *testing.T) {
	root := newShapedBox(paintengine2d.XYWH(0, 0, 100, 100))
	a := newShapedBox(paintengine2d.XYWH(10, 10, 30, 30))
	b := newShapedBox(paintengine2d.XYWH(50, 50, 20, 20))
	root.Add(a)
	root.Add(b)
	if got := TransparentRects(root, nil); len(got) != 0 {
		t.Fatalf("nothing declared itself see-through: %+v", got)
	}
	b.SetTransparent(true)
	got := TransparentRects(root, nil)
	if len(got) != 1 {
		t.Fatalf("one component declared itself see-through: %+v", got)
	}
	if want := (paintengine2d.XYWH(50, 50, 20, 20)); got[0] != want {
		t.Fatalf("box %+v, want %+v", got[0], want)
	}
	// A hidden one is not collected: it paints nothing at all.
	a.SetTransparent(true)
	a.SetVisible(false)
	if got := TransparentRects(root, nil); len(got) != 1 {
		t.Fatalf("a hidden see-through component was collected: %+v", got)
	}
}

func TestShapeHitAnswersForAnyComponent(t *testing.T) {
	// ShapeHit is for a component that implements Component from scratch;
	// one that cannot say is its whole box.
	root := newShapedBox(paintengine2d.XYWH(0, 0, 100, 100))
	if !ShapeHit(root, paintengine2d.Pt(1, 1)) {
		t.Fatal("no silhouette should mean the whole box")
	}
	root.SetHitShape(platform.ShapeEllipse(paintengine2d.XYWH(0, 0, 100, 100)))
	if ShapeHit(root, paintengine2d.Pt(1, 1)) {
		t.Fatal("the corner is outside the disc")
	}
	if !ShapeHit(root, paintengine2d.Pt(50, 50)) {
		t.Fatal("the middle is inside the disc")
	}
}

// ---- the silhouette a look gives a face -------------------------------------

// facedBox names the face it paints, so its look may shape it.
type facedBox struct {
	Base
	role style.Role
}

func newFacedBox(b paintengine2d.Rect, role style.Role, lk style.LookAndFeel) *facedBox {
	f := &facedBox{role: role}
	f.Init(f)
	f.SetBounds(b)
	f.SetLook(lk)
	return f
}

func (f *facedBox) ShapeRole() style.Role { return f.role }

// deckLook is the shipped skin whose push-button art is a stadium: the one
// look in the toolkit that answers a control shape at all.
func deckLook(t *testing.T) style.LookAndFeel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := style.LoadTheme("deck")
	if !ok {
		t.Skip("the deck skin is not registered in this build")
	}
	return p.Look()
}

// A component that names the face it paints takes the pointer where its look
// says that face is. Deck's button is a stadium, so a square box at the
// control's height is a disc: the middle is the button and the corners of the
// box are not, and a press there falls through to whatever is behind.
func TestAComponentTakesTheShapeItsLookGivesItsFace(t *testing.T) {
	lk := deckLook(t)
	side := lk.Metrics().ControlH
	box := paintengine2d.XYWH(0, 0, side, side)
	root := newShapedBox(box)
	root.SetLook(lk)
	face := newFacedBox(box, style.RoleButton, lk)
	root.Add(face)

	mid := paintengine2d.Pt(side/2, side/2)
	if got := HitRoot(root, mid); got != Component(face) {
		t.Fatalf("the middle of the disc is not the button: %v", got)
	}
	for _, p := range [][2]float32{{1, 1}, {side - 2, 1}, {1, side - 2}, {side - 2, side - 2}} {
		got := HitRoot(root, paintengine2d.Pt(p[0], p[1]))
		if got == Component(face) {
			t.Errorf("corner %v is the button, but its art is a disc", p)
		}
		if got != Component(root) {
			t.Errorf("corner %v fell past the panel behind the button too: %v", p, got)
		}
	}
}

// The same component in a look that paints rectangles is its whole box —
// which is every pack in the toolkit but the skins, and was every pack
// before any of this existed.
func TestALookThatPaintsRectanglesLeavesTheBoxAlone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := style.LoadTheme("breeze-night")
	if !ok {
		t.Skip("no breeze-night")
	}
	box := paintengine2d.XYWH(0, 0, 40, 40)
	face := newFacedBox(box, style.RoleButton, p.Look())
	for _, pt := range [][2]float32{{1, 1}, {38, 1}, {20, 20}, {38, 38}} {
		if HitRoot(face, paintengine2d.Pt(pt[0], pt[1])) != Component(face) {
			t.Fatalf("a press at %v missed a control its look did not shape", pt)
		}
	}
}

// A component that sets a silhouette of its own keeps it: the look is only
// ever the fallback for a component that said nothing.
func TestAnOwnShapeWinsOverTheLooks(t *testing.T) {
	lk := deckLook(t)
	side := lk.Metrics().ControlH
	face := newFacedBox(paintengine2d.XYWH(0, 0, side, side), style.RoleButton, lk)
	face.SetHitShape(platform.ShapeRect(side, side))
	for _, pt := range [][2]float32{{1, 1}, {side - 2, side - 2}} {
		if HitRoot(face, paintengine2d.Pt(pt[0], pt[1])) != Component(face) {
			t.Fatalf("a press at %v missed: the component's own shape is its whole box", pt)
		}
	}
}
