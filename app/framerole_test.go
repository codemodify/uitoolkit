package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A window tells its frame two things only the app knows — what the window
// is, and how tall a caption it wants — and a skin's frame lays the window
// out with a gap under its caption and a narrower border round the content
// than round the band.

// twoPieceSkin is a skin whose window is a band over a narrower body with a
// slot of desktop between them, whose caption buttons are drawn at the
// left, and which gives a window called "tabbed" a thinner band.
const twoPieceSkin = `{
  "skin": 1, "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/sheet.png" } },
  "sprites": { "band": { "at": [0, 0, 8, 8] } },
  "parts": { "caption": { "states": { "normal": "band" } }, "window": { "states": { "normal": "band" } } },
  "window": {
    "caption": 20,
    "border": { "caption": [0, 4, 0, 4], "content": [0, 16, 12, 16] },
    "captionGap": 6,
    "buttons": "left",
    "variants": { "tabbed": { "caption": 12 } }
  }
}`

// skinRig is a framed window in a skin installed for the test.
func skinRig(t *testing.T, name, doc string, scale float32) *shapeRig {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(platform.EnvDecorations, "")
	dir := filepath.Join(style.SkinsDir(), name)
	if err := os.MkdirAll(filepath.Join(dir, "art"), 0o755); err != nil {
		t.Fatal(err)
	}
	img := paintengine2d.NewImage(8, 8)
	img.Clear(paintengine2d.RGB(0.3, 0.4, 0.5))
	var buf bytes.Buffer
	if err := img.WritePNG(&buf); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "art", "sheet.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, style.SkinFile), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	style.InvalidateSkinCache()
	p, ok := style.LoadTheme(name)
	if !ok {
		t.Fatalf("the %s skin does not load", name)
	}
	r := &shapeRig{}
	r.a = New(Options{Look: p.Look(), Headless: true, Scale: scale})
	r.a.SetTitleBarPrefs(platform.DefaultTitleBarPrefs(""))
	w, err := r.a.NewWindow(platform.WindowOptions{
		Title: "Two", Width: 400, Height: 300, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.w, r.o = w, w.Surface().(*platform.Offscreen)
	w.SetContent(widgets.NewColumn(widgets.NewButton("Body", nil)))
	r.o.SimulateWindowState(platform.WindowState{Activated: true})
	r.a.PumpOnce()
	return r
}

func TestTheContentSitsUnderTheGapInsideItsOwnBorder(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		r := skinRig(t, "twopiece", twoPieceSkin, scale)
		g := r.w.geom
		px := func(v float32) float32 { return float32(int(v*scale + 0.5)) }
		if got := g.caption.Min.X - g.window.Min.X; got != px(4) {
			t.Errorf("%gx: the band is %g in from the left, want %g", scale, got, px(4))
		}
		if got := g.content.Min.Y - g.caption.Max.Y; got != px(6) {
			t.Errorf("%gx: the content starts %g under the band, want the %g gap", scale, got, px(6))
		}
		if got := g.content.Min.X - g.window.Min.X; got != px(16) {
			t.Errorf("%gx: the content is %g in from the left, want %g", scale, got, px(16))
		}
		if got := g.window.Max.Y - g.content.Max.Y; got != px(12) {
			t.Errorf("%gx: the content stops %g above the bottom, want %g", scale, got, px(12))
		}
		// Maximized, the screen's edges are the window's: the content is
		// the box under the band, with no gap and no border.
		r.o.SimulateWindowState(platform.WindowState{Activated: true, Maximized: true})
		r.a.PumpOnce()
		g = r.w.geom
		if g.content.Min.Y != g.caption.Max.Y || g.content.Min.X != g.window.Min.X || g.content.Max.Y != g.window.Max.Y {
			t.Errorf("%gx: maximized content %v under caption %v in window %v", scale, g.content, g.caption, g.window)
		}
	}
}

func TestAWindowGivenARoleWearsItsFrame(t *testing.T) {
	r := skinRig(t, "twopiece", twoPieceSkin, 1)
	if got := r.w.geom.caption.Dy(); got != 20 {
		t.Fatalf("the window's own caption is %g tall, want 20", got)
	}
	r.w.SetFrameRole("tabbed")
	r.a.PumpOnce()
	if got := r.w.geom.caption.Dy(); got != 12 {
		t.Errorf("the tabbed window's caption is %g tall, want 12", got)
	}
	if r.w.FrameRole() != "tabbed" {
		t.Error("FrameRole does not answer what was set")
	}
	// A role the skin has no frame for is the window's own frame.
	r.w.SetFrameRole("stranger")
	r.a.PumpOnce()
	if got := r.w.geom.caption.Dy(); got != 20 {
		t.Errorf("an unknown role's caption is %g tall, want the window's 20", got)
	}
	r.w.SetFrameRole("")
	r.a.PumpOnce()
	if got := r.w.geom.caption.Dy(); got != 20 {
		t.Errorf("dropping the role left a %g caption", got)
	}
}

func TestAnAppAsksItsWindowForAThinnerCaption(t *testing.T) {
	for _, pack := range []string{"breeze-night", "win95", "marquee"} {
		t.Run(pack, func(t *testing.T) {
			r := framedRig(t, pack, 500, 360)
			own := r.w.geom.caption.Dy()
			r.w.SetCaptionHeight(16)
			r.a.PumpOnce()
			if got := r.w.geom.caption.Dy(); got != 16 {
				t.Errorf("asked for 16, the caption is %g (the look's is %g)", got, own)
			}
			if r.w.CaptionHeight() != 16 {
				t.Error("CaptionHeight does not answer what was asked")
			}
			r.w.SetCaptionHeight(0)
			r.a.PumpOnce()
			if got := r.w.geom.caption.Dy(); got != own {
				t.Errorf("giving the caption back left it %g, want the look's %g", got, own)
			}
		})
	}
}

// A skin that draws its caption buttons at one end gets them there, and
// keeps the desktop's choice of which buttons there are, the outermost one
// still outermost.
func TestASkinPutsItsCaptionButtonsOnItsSide(t *testing.T) {
	r := skinRig(t, "twopiece", twoPieceSkin, 1)
	r.a.SetTitleBarPrefs(platform.TitleBarPrefs{Layout: platform.ParseButtonLayout("menu:minimize,maximize,close")})
	l := r.w.buttonLayout(r.w.caption)
	if len(l.Right) != 0 {
		t.Fatalf("buttons left on the right: %v", l)
	}
	want := []platform.CaptionButton{platform.CaptionClose, platform.CaptionMenu, platform.CaptionMaximize, platform.CaptionMinimize}
	if len(l.Left) != 4 {
		t.Fatalf("left side %v", l.Left)
	}
	// The window menu was already on the left and stays where it was; the
	// moved three come round with close outermost.
	got := append([]platform.CaptionButton(nil), l.Left...)
	if got[0] != platform.CaptionMenu || got[3] != platform.CaptionMinimize || got[1] != platform.CaptionClose {
		t.Errorf("left side %v, want menu then close, maximize, minimize (%v in some order)", got, want)
	}
	right := buttonsToSide(platform.ParseButtonLayout("close,minimize:"), style.ButtonsRight)
	if len(right.Left) != 0 || len(right.Right) != 2 || right.Right[1] != platform.CaptionClose {
		t.Errorf("moved right: %v", right)
	}
	if same := buttonsToSide(platform.ParseButtonLayout(":close"), style.ButtonsDesktop); len(same.Right) != 1 {
		t.Errorf("the desktop's side moved: %v", same)
	}
}
