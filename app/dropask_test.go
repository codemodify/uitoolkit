package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// askTarget takes files and allows every action, so a drag over it has
// something to ask about. It records what the drop performed.
type askTarget struct {
	widget.Base
	action platform.DragAction
	paths  []string
}

func (t *askTarget) DropTypes() []string { return []string{"text/uri-list"} }

func (t *askTarget) Drop(e widget.DropEvent) bool {
	t.action, t.paths = e.Action, e.Paths
	return len(e.Paths) > 0
}

func (t *askTarget) DropActionFor(platform.DragAction) platform.DragAction {
	return platform.DragCopy | platform.DragMove | platform.DragLink
}

func (t *askTarget) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(400, 300))
}
func (t *askTarget) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }
func (t *askTarget) Paint(*paintengine2d.Context) {}

// askRig is a window with that target in it, and the offscreen surface a
// drag is simulated on.
type askRig struct {
	a      *Application
	w      *Window
	off    *platform.Offscreen
	target *askTarget
}

func newAskRig(t *testing.T) *askRig {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	target := &askTarget{}
	target.Init(target)
	w.SetContent(target)
	a.PumpOnce()
	off, ok := w.surf.(*platform.Offscreen)
	if !ok {
		t.Skip("not an offscreen surface")
	}
	return &askRig{a: a, w: w, off: off, target: target}
}

// dropAsking simulates a drop whose source allows all three actions and
// asks for the user to choose.
func (r *askRig) dropAsking() {
	r.dropWith(platform.DragCopy|platform.DragMove|platform.DragLink|platform.DragAsk, platform.DragAsk)
}

func (r *askRig) dropWith(offered, action platform.DragAction) {
	r.target.action, r.target.paths = platform.DragNone, nil
	r.off.SimulateDropAction(paintengine2d.Pt(100, 150), map[string][]byte{
		"text/uri-list": []byte("file:///tmp/a.txt\r\n"),
	}, offered, action)
	r.a.PumpOnce()
}

// menuTexts is what the menu that is up offers, in order.
func (r *askRig) menuTexts(t *testing.T) []string {
	t.Helper()
	pop, ok := r.w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		t.Fatal("no menu is up")
	}
	var out []string
	for _, it := range pop.Items {
		if it.Separator {
			continue
		}
		out = append(out, it.Text)
	}
	return out
}

func (r *askRig) key(k platform.Key) {
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: k})
	r.a.PumpOnce()
}

// A drag that offers more than one action and asks for the choice puts
// the Copy / Move / Link menu under the pointer, and nothing is dropped
// until it is answered.
func TestDropAsksWhichActionToPerform(t *testing.T) {
	r := newAskRig(t)
	r.dropAsking()
	if got, want := r.menuTexts(t), []string{"Copy Here", "Move Here", "Link Here", "Cancel"}; !sameStrings(got, want) {
		t.Fatalf("the menu offers %q, want %q", got, want)
	}
	if r.target.paths != nil {
		t.Fatal("nothing may be dropped before the user has answered")
	}
	if taken := r.off.DropTaken(); taken != nil {
		t.Fatal("the source must not be released while the menu is up")
	}
	// Down, Down, Enter: the second choice is a move.
	r.key(platform.KeyDown)
	r.key(platform.KeyDown)
	r.key(platform.KeyReturn)
	if r.w.Popup() != nil {
		t.Fatal("the menu goes down when a choice is made")
	}
	if len(r.target.paths) != 1 || r.target.paths[0] != "/tmp/a.txt" {
		t.Fatalf("dropped %q", r.target.paths)
	}
	if r.target.action != platform.DragMove {
		t.Fatalf("the drop performed %v, want a move", r.target.action)
	}
	if taken := r.off.DropTaken(); taken == nil || !*taken {
		t.Fatal("the drop should be finished as taken")
	}
	// The action the source is told is the one the user picked.
	if _, a := r.off.DragAccepted(); a != platform.DragMove {
		t.Fatalf("the source was told %v, want a move", a)
	}
}

// Escape takes the menu down and refuses the drop: the source has to be
// told, or it waits for an answer that never comes.
func TestDropAskMenuCancels(t *testing.T) {
	r := newAskRig(t)
	r.dropAsking()
	r.key(platform.KeyEscape)
	if r.w.Popup() != nil {
		t.Fatal("Escape takes the menu down")
	}
	if r.target.paths != nil {
		t.Fatalf("a cancelled drop dropped %q", r.target.paths)
	}
	if taken := r.off.DropTaken(); taken == nil || *taken {
		t.Fatal("a cancelled drop is refused, and the source is told")
	}
}

// A modifier settles the question before the drop: the source sends the
// action it wants rather than asking, and no menu appears.
func TestModifierSkipsTheAskMenu(t *testing.T) {
	r := newAskRig(t)
	r.dropWith(platform.DragCopy|platform.DragMove|platform.DragLink|platform.DragAsk, platform.DragMove)
	if r.w.Popup() != nil {
		t.Fatal("a drag that names its action must not ask")
	}
	if r.target.action != platform.DragMove || len(r.target.paths) != 1 {
		t.Fatalf("the drop performed %v with %q", r.target.action, r.target.paths)
	}
}

// With one action there is nothing to ask about, so the drop just happens.
func TestOneActionIsNeverAsked(t *testing.T) {
	r := newAskRig(t)
	r.dropWith(platform.DragCopy|platform.DragAsk, platform.DragAsk)
	if r.w.Popup() != nil {
		t.Fatal("a drag with one action must not ask")
	}
	if r.target.action != platform.DragCopy || len(r.target.paths) != 1 {
		t.Fatalf("the drop performed %v with %q", r.target.action, r.target.paths)
	}
}

// While a drag that may be asked about is over the window, the window
// says it is willing to ask — which on Wayland is the only way the
// compositor ever offers the choice.
func TestWindowOffersToAskWhileTheDragIsOver(t *testing.T) {
	r := newAskRig(t)
	at := paintengine2d.Pt(100, 150)
	mimes := []string{"text/uri-list"}
	r.off.Inject(platform.Event{
		Kind: platform.EventDragMotion, Pos: at, Mimes: mimes,
		Actions: platform.DragCopy | platform.DragMove | platform.DragAsk,
	})
	r.a.PumpOnce()
	if allowed := r.off.DragAllowed(); !allowed.Asks() {
		t.Fatalf("the window allowed %v, which does not offer to ask", allowed)
	}
	// A source that never mentioned ask is never asked about.
	r.off.Inject(platform.Event{
		Kind: platform.EventDragMotion, Pos: at, Mimes: mimes,
		Actions: platform.DragCopy | platform.DragMove,
	})
	r.a.PumpOnce()
	if allowed := r.off.DragAllowed(); allowed.Asks() {
		t.Fatalf("the window offered to ask about a drag that never asked (%v)", allowed)
	}
}

// The set a drop puts to the user is what both sides allow, and there is
// nothing to put when they agree on one action or none.
func TestAskDropActionsNarrowsToBothSides(t *testing.T) {
	ev := func(offered, action platform.DragAction) platform.Event {
		return platform.Event{Kind: platform.EventDrop, Actions: offered, Action: action}
	}
	all := platform.DragCopy | platform.DragMove | platform.DragLink
	if got := dropAsks(ev(all|platform.DragAsk, platform.DragAsk), platform.DragCopy|platform.DragMove); got != platform.DragCopy|platform.DragMove {
		t.Fatalf("the choice is %v, want copy and move", got)
	}
	if got := dropAsks(ev(all|platform.DragAsk, platform.DragAsk), platform.DragLink); got != platform.DragNone {
		t.Fatalf("one action leaves nothing to ask, got %v", got)
	}
	if got := dropAsks(ev(all, platform.DragCopy), all); got != platform.DragNone {
		t.Fatalf("a drag that did not ask must not be asked about, got %v", got)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
