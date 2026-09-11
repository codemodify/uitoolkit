package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

func TestCardListPaints(t *testing.T) {
	look := style.DarkLook()
	h := &scaledHost{look: look}
	cards := NewCardList(4, func(i int) CardContent {
		return CardContent{
			Title: "Ada Lovelace", Subtitle: "Welcome", Meta: "3:04 PM",
			Snippet: "This is the uitoolkit Mail dogfood",
			Badges:  []CardBadge{{Label: "Important", Color: paintengine2d.RGB(0.8, 0.2, 0.2)}},
			Bold:    i == 0, Starred: i == 0,
		}
	}, nil)
	cards.SetHost(h)
	cards.Arrange(paintengine2d.XYWH(0, 0, 360, 280))
	if cards.rowH() < 40 {
		t.Fatalf("card height %v", cards.rowH())
	}
	img := paintengine2d.NewImage(360, 280)
	ctx := paintengine2d.NewContext(img)
	cards.Paint(ctx)
}
