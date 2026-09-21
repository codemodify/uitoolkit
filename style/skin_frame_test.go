package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The frame a skin states, beyond one border and one caption for every
// window: a frame per window role, a caption band inset differently from
// the content under it, a gap between the two, a side for the caption
// buttons, a title set on a tab, and silhouette rects pinned to the far
// edges. Everything here is design pixels in, whole device pixels out.

// skinFrameDoc has a caption band from the white mark, a variant that gives
// the "tabbed" role a band of its own from the sliced face, and the rest of
// the new window keys.
const skinFrameDoc = `{
  "skin": 1, "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": {
    "face": { "sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4, 4, 4, 4] },
    "mark": { "sheet": "chrome", "at": [16, 0, 8, 8] }
  },
  "text": { "caption": { "color": "#ffffff", "case": "upper" },
            "tab": { "color": "#000000", "size": 9 } },
  "parts": {
    "caption": { "states": { "normal": "mark" }, "text": "caption" },
    "window": { "states": { "normal": "mark" } }
  },
  "window": {
    "border": { "caption": [2, 4, 0, 4], "content": [0, 20, 10, 20] },
    "captionGap": 6,
    "caption": 20,
    "buttons": "left",
    "shape": [
      { "at": [0, 0, 0, 20], "stretchX": true },
      { "at": [0, 0, 60, 30], "fromRight": true, "fromBottom": true }
    ],
    "variants": {
      "tabbed": {
        "caption": 12,
        "title": { "align": "start", "inset": 18 },
        "parts": {
          "caption": { "states": { "normal": "face" }, "text": "tab" },
          "caption.title": { "states": { "normal": "mark" } }
        }
      }
    }
  }
}`

// frameTestLook installs skinFrameDoc as a user skin under a temporary
// config dir and returns its look at scale.
func frameTestLook(t *testing.T, scale float32) (*Classic, *Skin) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	installSkin(t, "frames", skinFrameDoc)
	p, ok := LoadTheme("frames")
	if !ok {
		t.Fatal("the frames skin does not load")
	}
	sk, _ := LoadSkin("frames")
	return WithScale(p.Look(), scale).(*Classic), sk
}

func TestSkinWindowReadsTheNewKeys(t *testing.T) {
	sk := loadTestSkin(t, skinFrameDoc)
	w := sk.Window
	if !w.Split || w.Border != (Insets{Top: 2, Right: 4, Left: 4}) || w.ContentBorder != (Insets{Right: 20, Bottom: 10, Left: 20}) {
		t.Fatalf("border %+v content %+v split %v", w.Border, w.ContentBorder, w.Split)
	}
	if w.CaptionGap != 6 || w.Buttons != ButtonsLeft {
		t.Fatalf("gap %g buttons %v", w.CaptionGap, w.Buttons)
	}
	if !w.Shape[1].FromRight || !w.Shape[1].FromBottom {
		t.Fatalf("anchors %+v", w.Shape[1])
	}
	v := w.Variants["tabbed"]
	if v == nil {
		t.Fatal("no tabbed variant")
	}
	// A variant is resolved at load: what it does not state is the
	// window's, what it does replaces it.
	if v.Caption != 12 || v.CaptionGap != 6 || !v.Split || v.Buttons != ButtonsLeft {
		t.Fatalf("variant %+v", v)
	}
	if !v.Title.Start || v.Title.Inset != 18 {
		t.Fatalf("variant title %+v", v.Title)
	}
	if v.Parts["caption"].art("normal") != sk.Sprites["face"] {
		t.Fatal("the variant does not rebind its caption")
	}
	if !sk.Text["caption"].Upper || sk.Text["tab"].Upper {
		t.Fatal(`"case": "upper" is not read, or leaks into a role that did not say it`)
	}
}

func TestSkinWindowRefusalsNameTheirKey(t *testing.T) {
	head := `{"skin": 1, "sheets": {"chrome": {"1x": "art/sheet.png"}},
		"sprites": {"mark": {"at": [16, 0, 8, 8]}},`
	cases := []struct{ name, window, wantKey string }{
		{"a gap below the caption band", `{"border": {"caption": [0, 0, 2, 0], "content": [0, 0, 0, 0]}}`, "window.border.caption"},
		{"a gap above the content", `{"border": {"caption": [0, 0, 0, 0], "content": [3, 0, 0, 0]}}`, "window.border.content"},
		{"half a split border", `{"border": {"caption": [0, 0, 0, 0]}}`, "window.border"},
		{"a negative gap", `{"captionGap": -1}`, "window.captionGap"},
		{"a side that is not one", `{"buttons": "top"}`, "window.buttons"},
		{"a title alignment that is not one", `{"title": {"align": "end"}}`, "window.title.align"},
		{"stretching and pinned", `{"shape": [{"at": [0, 0, 10, 10], "stretchX": true, "fromRight": true}]}`, "window.shape[0]"},
		{"a variant rebinding a control", `{"variants": {"eq": {"parts": {"button": {"states": {"normal": "mark"}}}}}}`, "window.variants.eq.parts.button"},
		{"a variant with variants", `{"variants": {"eq": {"variants": {"x": {}}}}}`, "window.variants.eq.variants"},
		{"frame parts outside a variant", `{"parts": {"caption": {"states": {"normal": "mark"}}}}`, "window.parts"},
		{"a bad variant key", `{"variants": {"eq": {"captin": 3}}}`, "window.variants.eq"},
	}
	for _, c := range cases {
		_, err := LoadSkinFS(skinTestFS(t, head+`"window": `+c.window+`}`), "probe")
		if err == nil {
			t.Errorf("%s: loaded", c.name)
			continue
		}
		se, ok := err.(*SkinError)
		if !ok || se.Key != c.wantKey {
			t.Errorf("%s: %v (key %q), want key %q", c.name, err, keyOf(err), c.wantKey)
		}
	}
	_, err := LoadSkinFS(skinTestFS(t, head+`"text": {"caption": {"case": "title"}}}`), "probe")
	if se, ok := err.(*SkinError); !ok || se.Key != "text.caption.case" {
		t.Errorf("a case that is not one: %v", err)
	}
}

func keyOf(err error) string {
	if se, ok := err.(*SkinError); ok {
		return se.Key
	}
	return ""
}

// A window the app gives a role wears that role's frame; a role the skin
// does not know, and no role at all, wear the window's own.
func TestAWindowWearsItsRolesFrame(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		lk, _ := frameTestLook(t, scale)
		whole := func(v float32) float32 { return float32(int(v*scale + 0.5)) }
		own := DecorationOf(lk, DecorationState{Active: true})
		tab := DecorationOf(lk, DecorationState{Active: true, Role: "tabbed"})
		stranger := DecorationOf(lk, DecorationState{Active: true, Role: "no-such-role"})
		if own.Caption != whole(20) || stranger.Caption != own.Caption {
			t.Errorf("%gx: own caption %g, unknown role %g, want %g", scale, own.Caption, stranger.Caption, whole(20))
		}
		if tab.Caption != max(whole(12), whole(8)) {
			t.Errorf("%gx: tabbed caption %g, want %g", scale, tab.Caption, whole(12))
		}
		if !tab.Stacked || tab.Button.Y <= 0 || tab.ButtonPad.Top+tab.Button.Y > tab.Caption {
			t.Errorf("%gx: tabbed buttons %v at %g do not stand in a %g band", scale, tab.Button, tab.ButtonPad.Top, tab.Caption)
		}
		// The split border, the gap and the side are whole device pixels,
		// and the variant keeps the ones it did not restate.
		for _, d := range []DecorationSpec{own, tab} {
			if !d.Split || d.ContentBorder.Left != whole(20) || d.ContentBorder.Bottom != whole(10) || d.Border.Left != whole(4) {
				t.Errorf("%gx: border %+v content %+v", scale, d.Border, d.ContentBorder)
			}
			if d.CaptionGap != whole(6) || d.ButtonSide != ButtonsLeft {
				t.Errorf("%gx: gap %g side %v", scale, d.CaptionGap, d.ButtonSide)
			}
		}
		// Maximized, there is nothing to inset from and nothing to leave a
		// gap in.
		m := DecorationOf(lk, DecorationState{Active: true, Maximized: true})
		if !m.Border.Zero() || m.Split || m.CaptionGap != 0 {
			t.Errorf("%gx: maximized frame kept border %+v split %v gap %g", scale, m.Border, m.Split, m.CaptionGap)
		}
	}
}

// The variant's own caption art is what is painted for a window of that
// role: the fixture's sliced face has red corners, the window's own caption
// is the solid white mark.
func TestAVariantPaintsItsOwnCaption(t *testing.T) {
	lk, _ := frameTestLook(t, 1)
	paint := func(role string) *paintengine2d.Image {
		img := paintengine2d.NewImage(120, 60)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		f := DecorationFrame{Window: paintengine2d.XYWH(0, 0, 120, 60), Caption: paintengine2d.XYWH(0, 0, 120, 12)}
		DrawDecorationOf(lk, ctx, f, DecorationState{Active: true, Role: role})
		return img
	}
	if r, g, b, _ := paint("").PremulAt(1, 1); r < 250 || g < 250 || b < 250 {
		t.Errorf("the window's own caption is %d,%d,%d at its corner, want the white mark", r, g, b)
	}
	if r, g, b, _ := paint("tabbed").PremulAt(1, 1); r < 250 || g > 5 || b > 5 {
		t.Errorf("the tabbed caption is %d,%d,%d at its corner, want the face's red corner", r, g, b)
	}
}

// A title set from the start sits on its plate as far in as the skin says,
// instead of centred on the band.
func TestATitleSetFromTheStartSitsOnItsTab(t *testing.T) {
	lk, _ := frameTestLook(t, 1)
	img := paintengine2d.NewImage(200, 12)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	DrawCaptionTitleOf(lk, ctx, paintengine2d.XYWH(0, 0, 200, 12), "Tab", DecorationState{Active: true, Role: "tabbed"})
	first := -1
	for x := 0; x < 200 && first < 0; x++ {
		if _, _, _, a := img.PremulAt(x, 6); a > 0 {
			first = x
		}
	}
	if first != 18 {
		t.Fatalf("the tab starts at x=%d, want the 18 the skin states", first)
	}
	if _, _, _, a := img.PremulAt(150, 6); a != 0 {
		t.Fatal("the tab runs on past the title: it is not as wide as its words")
	}
}

// "case": "upper" sets a role in capitals, and it is the skin's choice
// alone: a role that does not say so keeps the app's title as it is.
func TestTheCaptionCaseIsTheSkinsChoice(t *testing.T) {
	lk, _ := frameTestLook(t, 1)
	ink := func(title, role string) int {
		img := paintengine2d.NewImage(200, 20)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		DrawCaptionTitleOf(lk, ctx, paintengine2d.XYWH(0, 0, 200, 20), title, DecorationState{Active: true, Role: role})
		n := 0
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] > 128 {
				n++
			}
		}
		return n
	}
	// The window's own caption role is upper case: "minim" and "MINIM"
	// paint the same words.
	if a, b := ink("minim", ""), ink("MINIM", ""); a != b {
		t.Errorf("an upper-case role set %q in %d pixels and %q in %d", "minim", a, "MINIM", b)
	}
	// The tab's role is not: they differ.
	if a, b := ink("minim", "tabbed"), ink("MINIM", "tabbed"); a == b {
		t.Error("a role that did not ask for capitals set its title in them")
	}
}

// An app asks for its own caption height and gets it, in every look: the
// band is the height asked for in whole device pixels, and the buttons
// shrink to stand in it rather than growing it back.
func TestAnAppAsksForItsOwnCaptionHeight(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, pack := range []string{"breeze-night", "win95", "aqua", "marquee", "minim-silver"} {
		p, ok := LoadTheme(pack)
		if !ok {
			t.Fatalf("%s does not load", pack)
		}
		for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
			lk := WithScale(p.Look(), scale)
			for _, h := range []float32{14, 20, 40} {
				d := DecorationOf(lk, DecorationState{Active: true, Caption: h})
				want := max(float32(int(h*scale+0.5)), float32(int(8*scale+0.5)))
				if d.Caption != want {
					t.Errorf("%s@%gx asked %g: caption %g, want %g", pack, scale, h, d.Caption, want)
				}
				bh := max(d.Button.Y, d.CloseButton.Y)
				if bh > 0 && d.ButtonPad.Top+bh > d.Caption {
					t.Errorf("%s@%gx asked %g: a %g button %g down does not fit a %g band", pack, scale, h, bh, d.ButtonPad.Top, d.Caption)
				}
			}
		}
		if d0, d := DecorationOf(p.Look(), DecorationState{Active: true}), DecorationOf(p.Look(), DecorationState{Active: true, Caption: 0}); d0.Caption != d.Caption {
			t.Errorf("%s: asking for nothing changed the caption", pack)
		}
	}
}

// A rect pinned to the right or the bottom keeps its size and its margin
// from that edge whatever the window's size.
func TestShapeRectsPinToTheFarEdges(t *testing.T) {
	lk, sk := frameTestLook(t, 2)
	for _, win := range []paintengine2d.Rect{paintengine2d.XYWH(0, 0, 400, 300), paintengine2d.XYWH(0, 0, 900, 700)} {
		rects := skinShapeRects(sk.Window.Shape, win, lk.Scale()/sk.Design.Scale)
		if len(rects) != 2 {
			t.Fatalf("%d rects", len(rects))
		}
		foot := rects[1].Rect
		want := paintengine2d.XYWH(win.Max.X-120, win.Max.Y-60, 120, 60)
		if foot != want {
			t.Errorf("window %v: pinned rect %v, want %v", win, foot, want)
		}
	}
	// Both pins at once and a margin: fromRight's x is the room to the
	// right of the rect.
	r := SkinShapeRect{X: 10, Y: 5, W: 30, H: 20, FromRight: true}
	if got := r.resolve(paintengine2d.XYWH(100, 100, 200, 100), 1); got != paintengine2d.XYWH(260, 105, 30, 20) {
		t.Errorf("fromRight resolves to %v", got)
	}
}
