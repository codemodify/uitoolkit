package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The scroll bar's own strip belongs to the bar, in every view that has
// one. The list, the tree and the table are covered in drag_test.go,
// where the drag that started this is; these two are the rest of them.
//
// The menu is the one that bites: it acts on the *release*, so a click
// that visibly missed every item ran the command level with it.
func TestAClickBesideTheMenuScrollBarRunsNothing(t *testing.T) {
	ran := -1
	items := make([]*MenuItem, 0, 30)
	for i := 0; i < 30; i++ {
		i := i
		items = append(items, Item(fmt.Sprintf("Appearance %d", i+1), func() { ran = i }))
	}
	p := NewPopupMenu(items...)
	p.SetLook(style.DarkLook())
	p.SetHost(&host{})
	p.Arrange(paintengine2d.XYWH(0, 0, 240, 108))
	if p.MaxOffset() <= 0 {
		t.Skip("the menu fits its box; there is nothing to scroll")
	}
	if p.vparts().Bar.Empty() {
		t.Skip("this look draws no scroll bar")
	}

	click := func(x, y float32) {
		at := paintengine2d.Pt(x, y)
		p.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
		p.MouseRelease(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	}
	y := p.LocalBounds().Dy() / 2
	ch := p.chrome()
	for x := ch.PadL + p.innerWidth(); x < p.LocalBounds().Dx(); x++ {
		ran = -1
		click(x, y)
		if ran >= 0 {
			t.Errorf("a click at x=%v, beside the bar, ran item %d", x, ran)
		}
	}

	t.Run("and the highlight is the bar's, not an item's", func(t *testing.T) {
		p.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(20, y)})
		if p.hover < 0 {
			t.Fatal("the pointer on the items highlighted nothing; the test proves nothing")
		}
		bar := p.vparts().Bar
		p.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(bar.Min.X+bar.Dx()/2, y)})
		if p.hover >= 0 {
			t.Errorf("the pointer on the bar left item %d highlighted", p.hover)
		}
	})

	// And an item still runs, or the test proves nothing. Last, because
	// activating closes the menu.
	ran = -1
	click(20, y)
	if ran < 0 {
		t.Error("a click on an item ran nothing")
	}
}

func TestAPressBesideTheCardScrollBarSelectsNoCard(t *testing.T) {
	l := NewCardList(40, func(i int) CardContent {
		return CardContent{Title: "Card", Subtitle: "subject", Meta: "today"}
	}, nil)
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
	if l.vparts().Bar.Empty() {
		t.Skip("this look draws no scroll bar")
	}

	// Released, every time: a press left on the bar keeps the bar
	// tracking the pointer, and the moves after it are the bar's.
	press := func(x float32) {
		at := paintengine2d.Pt(x, 70)
		l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
		l.MouseRelease(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	}
	press(20)
	if l.Selected < 0 {
		t.Fatal("a press on the cards selected nothing; the test proves nothing")
	}
	for x := l.rowsW(); x < l.LocalBounds().Dx(); x++ {
		l.Selected = -1
		press(x)
		if l.Selected >= 0 {
			t.Errorf("a press at x=%v, beside the bar, selected card %d", x, l.Selected)
		}
	}

	t.Run("and the highlight is the bar's, not a card's", func(t *testing.T) {
		move := func(x float32) {
			l.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(x, 70)})
		}
		move(20)
		if l.hovered < 0 {
			t.Fatal("the pointer on the cards highlighted nothing; the test proves nothing")
		}
		for x := l.rowsW(); x < l.LocalBounds().Dx(); x++ {
			move(x)
			if l.hovered >= 0 {
				t.Errorf("x=%v, beside the bar, highlighted card %d", x, l.hovered)
			}
			move(20)
		}
	})
}
