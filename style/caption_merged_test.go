package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A look whose caption fits its title cannot stay fitted when the
// application's bar is the caption.
//
// BeOS's caption is a tab only as wide as its title and its buttons, and
// its silhouette leaves the rest of the top edge to the desktop. Merged,
// the application's bar is laid out across the window's whole width — so
// everything to the right of the tab fell outside the window and showed
// only its bottom few pixels. A mail client's menu, buttons and tabs were
// cut away to an 8-pixel sliver.
//
// DecorationOf already drops a fitted caption for a maximized or tiled
// window, for the same reason and in the same place.
func TestAMergedCaptionIsNotFittedToATitleItDoesNotDraw(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := LoadTheme("beos")
	if !ok {
		t.Skip("beos is not in this build")
	}
	lk := p.Look()

	if d := DecorationOf(lk, DecorationState{Active: true}); !d.CaptionFits {
		t.Skip("beos no longer fits its caption to its title")
	}
	d := DecorationOf(lk, DecorationState{Active: true, Merged: true})
	if d.CaptionFits {
		t.Error("a merged caption is still fitted to a title it does not draw")
	}
	// And the half of the silhouette that goes with it.
	f := DecorationFrame{
		Window:  paintengine2d.XYWH(0, 0, 1280, 800),
		// A narrow caption, which is what BeOS actually draws: with a
		// full-width one beTabCells finds the tab as wide as the window
		// and answers no silhouette either way, which would make the
		// assertion below prove nothing.
		Caption: paintengine2d.XYWH(0, 0, 300, 32),
	}
	if sil := WindowShapeOf(lk, f, DecorationState{Active: true, Merged: true}); sil != nil {
		t.Error("a merged window kept the outline that leaves its top edge to the desktop")
	}
	// Unmerged it keeps both: this is not a change to how BeOS draws.
	if sil := WindowShapeOf(lk, f, DecorationState{Active: true}); sil == nil {
		t.Error("beos lost its silhouette when nothing asked it to")
	}
}
