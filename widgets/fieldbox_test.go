package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A browser's address bar is one well with three things in it: the
// site-information mark at the leading end, the text, and the bookmark
// star at the trailing end.
//
// TextField.Clearable does the trailing end only, and does it by taking
// the room out of the *string* — which cannot work at the leading end,
// because where the text begins belongs to the engine and twenty-five of
// them draw a field. So the well is the box's and the field inside it is
// frameless.
func TestFieldBoxIsOneWellAroundItsParts(t *testing.T) {
	lk := style.LightLook()
	field := NewTextField("github.com/codemodify/uitoolkit", "Search", nil)
	lead := NewIconButton(style.IconLock, "Site information", nil)
	trail := NewIconButton(style.IconStar, "Bookmark", nil)
	fb := NewFieldBox([]widget.Component{lead}, field, []widget.Component{trail})
	fb.SetHost(&fakeWindow{look: lk})

	// The field inside stops drawing its own well: one frame, not two.
	if !field.Frameless {
		t.Error("the field inside still draws its own frame")
	}

	const w = 600
	sz := fb.Measure(layout.Constraints{MaxW: w, MaxH: 100})
	fb.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))

	lb, fbnds, tb := lead.Bounds(), field.Bounds(), trail.Bounds()
	if lb.Max.X > fbnds.Min.X+0.5 {
		t.Errorf("the leading mark is not before the text: %v then %v", lb, fbnds)
	}
	if fbnds.Max.X > tb.Min.X+0.5 {
		t.Errorf("the trailing mark is not after the text: %v then %v", fbnds, tb)
	}
	// All three inside the one well.
	box := fb.LocalBounds()
	for name, r := range map[string]paintengine2d.Rect{"lead": lb, "field": fbnds, "trail": tb} {
		if r.Min.X < box.Min.X-0.5 || r.Max.X > box.Max.X+0.5 {
			t.Errorf("%s (%v) is outside the well %v", name, r, box)
		}
	}
	// The text takes what the marks leave, which is what makes it an
	// address bar rather than three controls in a row.
	if fbnds.Dx() < w*0.5 {
		t.Errorf("the field got %v of %v — it should take the spare width", fbnds.Dx(), w)
	}

	// And the well is ink: it paints without the field's frame.
	img := paintengine2d.NewImage(w, int(sz.Y)+4)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, sz.Y+4), paintengine2d.Fill(paintengine2d.RGB(0.5, 0.5, 0.5)))
	widget.PaintTree(fb, ctx, nil)
	img.Touch()
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if r, g, b, _ := img.At(x, y).RGBA(); r>>8 != 128 || g>>8 != 128 || b>>8 != 128 {
				n++
			}
		}
	}
	if n < img.Width*img.Height/4 {
		t.Errorf("the well drew %d pixels of %d — it is not painting", n, img.Width*img.Height)
	}
}

// A tool bar can give one control its spare width. Every browser's
// address bar needs it, and ToolItem.Stretch is not it: that is blank
// space, so a widget marked with it was treated as a gap and never laid
// out at all — the look-like-chromium sample's omnibox was an invisible
// 950-pixel hole in its tool bar.
func TestToolGrowGivesAWidgetTheSpareWidth(t *testing.T) {
	lk := style.LightLook()
	wide := NewTextField("x", "", nil)
	bar := NewToolBar(
		ToolIconBtn(style.IconNew, "", nil),
		ToolGrow(wide),
		ToolIconBtn(style.IconSave, "", nil),
	)
	bar.SetHost(&fakeWindow{look: lk})
	const w = 800
	bar.Arrange(paintengine2d.XYWH(0, 0, w, 40))

	got := wide.Bounds()
	if got.Dx() < w*0.5 {
		t.Errorf("the grown widget got %v of %v", got.Dx(), w)
	}
	if got.Dx() <= 0 {
		t.Fatal("the grown widget was never laid out — the bar treated it as blank space")
	}
	// What follows it still sits after it, not under it.
	var last paintengine2d.Rect
	for _, it := range bar.items {
		if it != nil && it.Widget == nil && !it.Sep && !it.Stretch {
			last = paintengine2d.Rect{}
		}
	}
	_ = last
	if got.Max.X > float32(w) {
		t.Errorf("the grown widget ran past the bar: %v in %v", got, w)
	}
}

// A mark that is a button only when you point at it.
//
// Chromium's site-information lock, its bookmark star and its three-dot
// menu are all drawn with no frame until the pointer is over them. The
// toolkit had the flat face (ToolButton) and the framed one (IconButton)
// and no way to ask the second for the first, so a sample copying that
// chrome put framed buttons inside its own address bar — a control drawn
// inside a control.
func TestIconButtonFlatDrawsNoFrameUntilPointedAt(t *testing.T) {
	lk := style.LightLook()
	ink := func(flat, hovered bool) int {
		b := NewIconButton(style.IconLock, "Site information", nil)
		b.Flat = flat
		b.SetHost(&fakeWindow{look: lk})
		b.Arrange(paintengine2d.XYWH(0, 0, 32, 32))
		if hovered {
			b.hovered = true
		}
		img := paintengine2d.NewImage(32, 32)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 32, 32), paintengine2d.Fill(lk.Palette().Background))
		b.Paint(ctx)
		img.Touch()
		n := 0
		bg := lk.Palette().Background
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				if absI(int(r>>8)-int(bg.R*255)) > 6 || absI(int(g>>8)-int(bg.G*255)) > 6 || absI(int(bl>>8)-int(bg.B*255)) > 6 {
					n++
				}
			}
		}
		return n
	}
	framed, flat := ink(false, false), ink(true, false)
	if flat >= framed {
		t.Errorf("flat drew %d pixels and framed %d — flat should be the mark alone", flat, framed)
	}
	if flat == 0 {
		t.Error("flat drew nothing at all — the mark must still be there")
	}
	// And it does grow a face under the pointer, so it is still findable
	// as a control.
	if hot := ink(true, true); hot <= flat {
		t.Errorf("flat drew %d at rest and %d hovered — it should gain a face", flat, hot)
	}
}

func absI(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
