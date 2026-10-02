package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// A browser tab's title sits against its mark, not in the middle of the
// tab. Chrome, Firefox, Safari and VS Code all do it that way; the toolkit
// centred it because a browser tab is drawn by the same engine hook as a
// notebook tab, and a notebook tab is centred.
func TestABrowserTabsTitleIsSetAgainstItsMark(t *testing.T) {
	look := style.DarkLook()
	// The box the word is placed in is the same whichever way it is set;
	// what changes is where in it the word lands, which the look draws.
	// What this holds is that the box leaves room at both ends, which a
	// centred label never needed and a left-set one does.
	tabs := NewBrowserTabs()
	tabs.AddTab(BrowserTab{Title: "Mail", Icon: style.IconInbox, NoClose: true})
	tabs.SetLook(look)
	tabs.SetHost(&host{})
	tabs.Arrange(paintengine2d.XYWH(0, 0, 800, 44))
	g := tabs.geom()
	s := tabs.slotOf(0, g)
	lb := tabs.labelBox(0, s, g)

	if tabs.Align != style.AlignStart {
		t.Errorf("the default alignment is %v, want AlignStart — a browser sets its titles against the mark", tabs.Align)
	}
	if icon := tabs.iconRect(0, s); !icon.Empty() && lb.Min.X < icon.Max.X {
		t.Errorf("the label box starts at %g, inside the mark that ends at %g", lb.Min.X, icon.Max.X)
	}
	if lb.Min.X <= s.Min.X {
		t.Error("the label box starts at the tab's own edge, with no room for the word")
	}
	if lb.Max.X >= s.Max.X {
		t.Error("the label box runs to the tab's own edge")
	}
}
