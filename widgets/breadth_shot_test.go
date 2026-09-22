package widgets_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// breadthLooks are the extreme looks the rich-text editor, the MDI area
// and the wizard are checked in: the classic desktops, today's, a web
// look and a skin.
var breadthLooks = []string{"win95", "system7", "luna", "aqua", "breeze6", "adwaita48", "tahoe", "sourcegit", "deck"}

// shotWindow opens a headless window of w x h logical pixels in pack at
// scale, holding what build makes, laid out and painted.
func shotWindow(t *testing.T, pack string, scale float32, w, h int, build func() widget.Component) (*uitoolkit.Application, *uitoolkit.Window) {
	t.Helper()
	look := style.DarkLook()
	if p, ok := style.LoadTheme(pack); ok {
		look = p.Look()
	} else if pack != "" {
		t.Fatalf("unknown pack %q", pack)
	}
	a := uitoolkit.New(uitoolkit.Options{Look: look, Headless: true, Scale: scale, DisableLookWatch: true})
	win, err := a.NewWindow(platform.WindowOptions{Title: "shot", Width: w, Height: h, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	win.SetContent(build())
	a.PumpOnce()
	a.PumpOnce()
	return a, win
}

// writeShot saves the window when UITK_SHOTS names a directory: the
// stills the widgets were looked at in (see docs/widgets.md).
func writeShot(t *testing.T, win *uitoolkit.Window, name string) {
	t.Helper()
	dir := os.Getenv("UITK_SHOTS")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, strings.ReplaceAll(name, "/", "-")+".png")
	if err := win.WritePNG(path); err != nil {
		t.Fatal(err)
	}
}
