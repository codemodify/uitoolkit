package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// A tab's word and mark sit on the face the look drew for it, not on the
// slot the strip handed out.
//
// NeXT's tabs — so Window Maker's and OpenStep's — draw the tab behind
// the pane lower than the one in front. Placing the title from the slot
// centred it on the strip instead, so the tops of the words crossed the
// unselected tab's own top edge.
func TestATabsWordSitsOnTheFaceTheLookDrew(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, pack := range []string{"wmaker-default", "next", "openstep"} {
		p, ok := style.LoadTheme(pack)
		if !ok {
			continue
		}
		tabs := NewBrowserTabs()
		tabs.AddTab(BrowserTab{Title: "Mail", Icon: style.IconInbox, NoClose: true})
		tabs.AddTab(BrowserTab{Title: "News", Icon: style.IconMail})
		tabs.Select(0)
		tabs.SetLook(p.Look())
		tabs.SetHost(&host{})
		tabs.Arrange(paintengine2d.XYWH(0, 0, 800, 48))
		g := tabs.geom()

		sel, unsel := tabs.slotOf(0, g), tabs.slotOf(1, g)
		selFace := style.TabFaceOf(p.Look(), sel, tabs.tabState(0), true)
		unselFace := style.TabFaceOf(p.Look(), unsel, tabs.tabState(1), false)
		if unselFace.Min.Y <= selFace.Min.Y {
			t.Skipf("%s does not draw the tab behind lower", pack)
		}

		// The unselected tab's label box starts inside its own face, not
		// at the top of the strip.
		lb := tabs.labelBox(1, unsel, g)
		if lb.Min.Y < unselFace.Min.Y {
			t.Errorf("%s: the label box starts at %g, above the face at %g",
				pack, lb.Min.Y, unselFace.Min.Y)
		}
		if ir := tabs.iconRect(1, unsel); !ir.Empty() && ir.Min.Y < unselFace.Min.Y {
			t.Errorf("%s: the mark starts at %g, above the face at %g",
				pack, ir.Min.Y, unselFace.Min.Y)
		}
	}
}
