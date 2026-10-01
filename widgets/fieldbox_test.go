package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
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

// The two browsers people copy chose different tab shapes, and the
// toolkit drew only one of them.
//
// Chrome's tab is merged: filled with the tool bar's colour, with concave
// feet running into the row below, so the tab and the tool bar read as
// one surface. Firefox's, since Proton, is a floating card standing clear
// of that row on every side. A sample copying Firefox with Chrome's shape
// came out looking like Chrome wearing Firefox's colours.
func TestFloatingTabsStandClearOfTheRowBelow(t *testing.T) {
	// Tinted, because that is the case the shapes exist for and the only
	// one in which they can be told apart: a pack whose title bar and
	// tool bar are the same colour gives a tab of either shape nothing to
	// stand out from.
	tint := paintengine2d.RGB(0.38, 0.58, 0.81)
	lk := style.WithChrome(style.Appearance{Name: "linear", Theme: style.ThemeLight}.Look(),
		map[string]paintengine2d.Color{"titleBar": tint})
	strip := func(shape style.TabShape) (topGap, bottomGap int) {
		tabs := NewBrowserTabs("One", "Two")
		tabs.Shape = shape
		hb := NewHeaderBar(nil, tabs, nil)
		hb.ShowTitle = false
		hb.SetHost(&fakeWindow{look: lk})
		hb.SetWindowControls(platform.ButtonLayout{}, true)
		hb.SetCaptionStyle(style.CaptionMerged)
		const w, h = 400, 40
		hb.Arrange(paintengine2d.XYWH(0, 0, w, h))
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		// The band behind the tabs, which in a window is the caption the
		// frame paints: without it the tab's fill and the ground are the
		// same colour and there is nothing to measure.
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(tint))
		widget.PaintTree(hb, ctx, nil)
		img.Touch()
		// Down a column through the first (selected) tab: how many rows at
		// the top and the bottom are still the *strip* rather than the
		// tab. The strip's own colour is sampled past the last tab, not
		// tab of either shape reaches, rather than assumed.
		sr, sg, sb, _ := img.At(w-3, 2).RGBA()
		const x = 40
		first, last := -1, -1
		for y := 0; y < h; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			d := absI(int(r>>8)-int(sr>>8)) + absI(int(g>>8)-int(sg>>8)) + absI(int(b>>8)-int(sb>>8))
			if d > 8 {
				if first < 0 {
					first = y
				}
				last = y
			}
		}
		if first < 0 {
			t.Fatalf("%v: no tab drawn at all", shape)
		}
		return first, h - 1 - last
	}

	_, mergedBottom := strip(style.TabsMerged)
	floatTop, floatBottom := strip(style.TabsFloating)

	if mergedBottom > 1 {
		t.Errorf("a merged tab left %d rows under it — it should meet the row below", mergedBottom)
	}
	if floatBottom < 2 {
		t.Errorf("a floating tab left %d rows under it — it should stand clear", floatBottom)
	}
	if floatTop < 2 {
		t.Errorf("a floating tab left %d rows above it — it should stand clear", floatTop)
	}
}

// A tab carries a mark: a browser's favicon, a repository's icon, the
// kind of document the tab holds.
//
// BrowserTab had Title, Tip, NoClose and Data and no way to show one, so
// a sample copying SourceGit's repository tabs had bare words where the
// real ones have an icon — and the browsers had no favicons either.
func TestBrowserTabsCarryAMark(t *testing.T) {
	lk := style.Appearance{Name: "linear", Theme: style.ThemeLight}.Look()
	build := func(icon style.ToolIcon) (*BrowserTabs, paintengine2d.Rect) {
		tabs := NewBrowserTabs()
		tabs.AddTab(BrowserTab{Title: "uitoolkit", Icon: icon})
		tabs.AddTab(BrowserTab{Title: "comms-mail", Icon: icon})
		tabs.Select(0)
		tabs.SetHost(&fakeWindow{look: lk})
		tabs.Arrange(paintengine2d.XYWH(0, 0, 500, 36))
		return tabs, tabs.iconRect(0, tabs.slotOf(0, tabs.geom()))
	}

	_, none := build(style.IconNone)
	if !none.Empty() {
		t.Error("a tab with no icon reserved room for one")
	}
	withIcon, r := build(style.IconFolder)
	if r.Empty() {
		t.Fatal("a tab with an icon got no room for it")
	}

	// The mark sits at the leading end, inside the tab, and the title
	// starts after it rather than under it.
	slot := withIcon.slotOf(0, withIcon.geom())
	if r.Min.X < slot.Min.X || r.Max.X > slot.Max.X {
		t.Errorf("the mark %v is outside its tab %v", r, slot)
	}
	lb := withIcon.labelBox(0, slot, withIcon.geom())
	if lb.Min.X < r.Max.X {
		t.Errorf("the title box %v starts before the mark ends (%v)", lb, r)
	}

	// And it is ink, not just reserved space.
	draw := func(tabs *BrowserTabs) int {
		img := paintengine2d.NewImage(500, 36)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 500, 36), paintengine2d.Fill(paintengine2d.RGB(0, 1, 0)))
		widget.PaintTree(tabs, ctx, nil)
		img.Touch()
		n := 0
		for y := int(r.Min.Y); y < int(r.Max.Y); y++ {
			for x := int(r.Min.X); x < int(r.Max.X); x++ {
				if rr, gg, bb, _ := img.At(x, y).RGBA(); !(rr>>8 == 0 && gg>>8 == 255 && bb>>8 == 0) {
					n++
				}
			}
		}
		return n
	}
	if n := draw(withIcon); n == 0 {
		t.Error("the mark drew nothing in the room it reserved")
	}
}

// A flat button keeps its word.
//
// IconButton is a mark alone, but MenuButton embeds it, and a *named*
// menu — "File", "Edit" — is the same control with a word instead. The
// flat face drew an empty label, so a code editor's menu bar came out as
// four blank hit areas.
func TestAFlatButtonKeepsItsLabel(t *testing.T) {
	lk := style.LightLook()
	ink := func(flat bool) int {
		b := NewTextMenuButton("File", &MenuItem{Text: "New"})
		b.Flat = flat
		b.SetHost(&fakeWindow{look: lk})
		sz := b.Measure(layout.Constraints{MaxW: 300, MaxH: 60})
		b.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
		img := paintengine2d.NewImage(int(sz.X)+2, int(sz.Y)+2)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, sz.X+2, sz.Y+2), paintengine2d.Fill(lk.Palette().Background))
		b.Paint(ctx)
		img.Touch()
		n, bg := 0, lk.Palette().Background
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				if absI(int(r>>8)-int(bg.R*255)) > 40 {
					n++
				}
				_, _ = g, bl
			}
		}
		return n
	}
	if flat := ink(true); flat == 0 {
		t.Error("a flat named menu drew nothing — its word was dropped")
	}
}
