package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A supplied raster must not be masked by a generic theme name.
//
// StatusIcon documents its order as Image, then Path, then Name, but the SNI
// property export substituted "application-default-icon" whenever Name was
// empty — without asking whether the caller had supplied a picture. The
// StatusNotifierItem specification has hosts *prefer* the name over the
// pixmap, so an image-only icon went on the bus asking the host to show a
// generic icon instead of the application's own. A player worked around it by
// passing a name no installed theme has.

func TestASuppliedPictureExportsNoIconName(t *testing.T) {
	logo := paintengine2d.NewImage(32, 32)
	logo.Clear(paintengine2d.RGB(1, 0, 0))

	png := filepath.Join(t.TempDir(), "logo.png")
	writeTestPNG(t, png, logo)

	for _, tc := range []struct {
		what string
		icon StatusIcon
	}{
		{"an Image", StatusIcon{Image: logo}},
		{"a Path", StatusIcon{Path: png}},
		{"both", StatusIcon{Image: logo, Path: png}},
	} {
		if got := statusIconName(tc.icon); got != "" {
			t.Errorf("%s exported IconName %q, which a host may show instead of the raster", tc.what, got)
		}
		// And the raster really is there: an empty name with no pixmap
		// would be a tray icon that shows nothing at all.
		if pix := iconPixmapsOf(tc.icon); len(pix) == 0 || pix[0].W == 0 {
			t.Errorf("%s exported no pixmap", tc.what)
		}
	}
}

// A name the caller gave still wins — that is what Name is for.
func TestAnExplicitIconNameIsStillExported(t *testing.T) {
	logo := paintengine2d.NewImage(32, 32)
	logo.Clear(paintengine2d.RGB(0, 1, 0))
	if got := statusIconName(StatusIcon{Name: "media-player-music", Image: logo}); got != "media-player-music" {
		t.Errorf("IconName %q, want the one the caller gave", got)
	}
}

// With no picture at all the generic name stays, so a host that ignores
// IconPixmap shows something rather than a blank.
func TestNoPictureStillFallsBackToTheGenericName(t *testing.T) {
	if got := statusIconName(StatusIcon{}); got != genericStatusIconName {
		t.Errorf("an empty icon exported %q, want the generic name", got)
	}
	// A Path that does not load is not a picture: there is no raster for a
	// name to mask, so the generic one is still the better answer.
	missing := filepath.Join(t.TempDir(), "nothing-here.png")
	if got := statusIconName(StatusIcon{Path: missing}); got != genericStatusIconName {
		t.Errorf("an unloadable path exported %q, want the generic name", got)
	}
	// An image with no pixels is not a picture either.
	if got := statusIconName(StatusIcon{Image: paintengine2d.NewImage(0, 0)}); got != genericStatusIconName {
		t.Errorf("an empty image exported %q, want the generic name", got)
	}
}

// A notification carries no pixmap, so it still gets a name to show.
func TestANotificationStillHasAnIconName(t *testing.T) {
	logo := paintengine2d.NewImage(32, 32)
	logo.Clear(paintengine2d.RGB(0, 0, 1))
	if got := notifyStatusIconName(StatusIcon{Image: logo}); got != genericStatusIconName {
		t.Errorf("a notification for an image-only item got %q, want something to show", got)
	}
	if got := notifyStatusIconName(StatusIcon{Name: "mail-unread"}); got != "mail-unread" {
		t.Errorf("a notification got %q, want the caller's name", got)
	}
}

// writeTestPNG puts a real PNG on disk, so the Path case is a path that
// actually loads — which is the difference the fix turns on.
func writeTestPNG(t *testing.T, path string, img *paintengine2d.Image) {
	t.Helper()
	b := notifyPNG(img)
	if len(b) == 0 {
		t.Fatal("could not encode the test image")
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
