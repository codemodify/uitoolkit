package settingsapp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/a11y/a11ytest"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The two new things on this page are drawn rather than laid out, so
// the rest of what they are is answered by looking at them: the cross
// at the end of the theme search field, which the widget strokes itself
// over whatever well, bevel or capsule the pack put there, and the
// About card, which has to hold together at the 720x580 minimum.
//
// Settings is built *in* the pack here rather than in the applied look,
// which is the opposite of what page_shot_test.go does and is the whole
// point: what is being looked at is a field's own chrome, and the field
// is drawn in whatever look Settings is wearing. Win95's sunken well
// and Aqua's gel capsule are the two that decorate a field most and
// leave the least room at its trailing end.
//
//	UITK_SHOTS=/tmp/shots tools/testenv.sh go test ./cmd/uitoolkit-settings/... -run AboutAndClear
func TestAboutAndClearButtonHold(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("UITK_SHOTS")
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, pack := range []string{"win95", "aqua", "breeze", "adwaita", "breeze-night"} {
		p, ok := style.LoadTheme(pack)
		if !ok {
			t.Fatalf("unknown pack %q", pack)
		}
		for _, scale := range []float32{1, 1.75} {
			for _, size := range [][2]int{{1024, 860}, {720, 580}} {
				where := fmt.Sprintf("%s at %g×, %dx%d", pack, scale, size[0], size[1])
				a := uitoolkit.New(uitoolkit.Options{Look: p.Look(), Headless: true, Scale: scale, DisableLookWatch: true})
				w, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: size[0], Height: size[1], Headless: true})
				if err != nil {
					t.Fatal(err)
				}
				w.SetContent(SettingsApp(a, w))
				a.PumpOnce()

				// Something to clear, and long enough that it would run
				// under the cross if nothing stopped it.
				search := searchField(t, w)
				search.SetText("windows classic 95 high contrast")
				a.PumpOnce()
				if !search.Clearable {
					t.Fatalf("%s: the search field is not clearable", where)
				}

				cross := clearBox(t, w)
				field := widget.LocalToWindow(search, search.LocalBounds())
				if !field.Contains(cross.Min) || cross.Max.X > field.Max.X+0.51 {
					t.Errorf("%s: the cross %v is not inside the field %v", where, cross, field)
				}
				// It grows with the window and is never too small to
				// hit: 10 logical pixels is the floor below which the
				// widget draws none at all.
				if side := cross.Dx(); side < 10*scale-0.51 {
					t.Errorf("%s: the cross is %v across", where, side)
				}
				if dir != "" {
					name := fmt.Sprintf("clear-%s-%gx-%dx%d.png", pack, scale, size[0], size[1])
					if err := w.WritePNG(filepath.Join(dir, name)); err != nil {
						t.Fatal(err)
					}
				}

				// And the card over the same window.
				about := findButton(w.Content(), "About")
				if about == nil {
					t.Fatalf("%s: no About button", where)
				}
				about.OnClick()
				a.PumpOnce()
				if w.Overlay() == nil {
					t.Fatalf("%s: About opened nothing", where)
				}
				if err := checkAboutCard(w); err != nil {
					t.Errorf("%s: %v", where, err)
				}
				// Everything the card says is in the tree and named, at
				// every size: a dialog that elides its own version is
				// worse than no dialog.
				tree := a11ytest.Audit(t, "about "+where, w.AccessibleTree())
				if a11ytest.Find(tree, a11y.RoleButton, "Close") == nil {
					t.Errorf("%s: the About card has no Close in the tree", where)
				}
				if dir != "" {
					name := fmt.Sprintf("about-%s-%gx-%dx%d.png", pack, scale, size[0], size[1])
					if err := w.WritePNG(filepath.Join(dir, name)); err != nil {
						t.Fatal(err)
					}
				}
				w.Close()
			}
		}
	}
}

// clearBox is where the clear button sits in the window, taken from the
// accessibility tree because that is the box a screen reader — and an
// end-to-end driver clicking by name — will aim at.
func clearBox(t *testing.T, w *app.Window) paintengine2d.Rect {
	t.Helper()
	n := a11ytest.Find(w.AccessibleTree(), a11y.RoleButton, "Clear Search themes")
	if n == nil {
		t.Fatal("the clear button is not in the accessibility tree")
	}
	return n.Bounds
}

// checkAboutCard is the part of the picture a picture cannot be asked:
// that the card is inside its window and was not shrunk to nothing to
// get there.
func checkAboutCard(w *app.Window) error {
	ov := w.Overlay()
	var card widget.Component
	for _, c := range ov.Children() {
		card = c
	}
	if card == nil {
		return fmt.Errorf("the overlay has no card")
	}
	b, in := card.Bounds(), ov.LocalBounds()
	if b.Min.X < -0.51 || b.Min.Y < -0.51 || b.Max.X > in.Dx()+0.51 || b.Max.Y > in.Dy()+0.51 {
		return fmt.Errorf("the card %v hangs out of a %v window", b, in)
	}
	if b.Dx() < in.Dx()*0.4 || b.Dy() < 120 {
		return fmt.Errorf("the card is %vx%v in a %v window", b.Dx(), b.Dy(), in)
	}
	return nil
}
