package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Aero's glass is its frame's alone, and only the Glass flavour asks.
func TestAeroGlassIsTheFrames(t *testing.T) {
	lk := lookNamed(t, "aero")
	d := DecorationOf(lk, DecorationState{Active: true})
	if !d.Glass || !d.GlassFrame {
		t.Fatalf("aero: glass %v frame %v", d.Glass, d.GlassFrame)
	}
	if !GlassFrameOnly(lk) {
		t.Fatal("GlassFrameOnly")
	}
	if d := DecorationOf(lk, DecorationState{Active: true, Solid: true}); d.Glass || d.GlassFrame {
		t.Fatal("an uncomposited screen keeps no glass")
	}
	if p, ok := LoadTheme("aero-basic"); ok && WantsGlass(p.Look()) {
		t.Fatal("Windows 7 Basic has no glass")
	}
}

// With real glass the frame is translucent over the desktop; without it the
// painted frame is what it always was, opaque.
func TestAeroFrameKeepsItsAlphaOnlyOverRealGlass(t *testing.T) {
	lk := lookNamed(t, "aero").(*Classic)
	alphaAt := func(glass bool) uint32 {
		SetGlassAvailable(glass)
		defer SetGlassAvailable(false)
		img := paintengine2d.NewImage(400, 300)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.White)
		st := DecorationState{Active: true}
		win := paintengine2d.XYWH(0, 0, 400, 300)
		cap := paintengine2d.XYWH(8, 0, 384, 30)
		DrawDecorationOf(lk, ctx, DecorationFrame{Window: win, Caption: cap}, st)
		_, _, _, a := img.At(200, 12).RGBA()
		return a >> 8
	}
	if a := alphaAt(false); a != 255 {
		t.Fatalf("painted glass alpha %d, want opaque", a)
	}
	if a := alphaAt(true); a > 200 || a == 0 {
		t.Fatalf("real glass caption alpha %d, want translucent", a)
	}
}

// A menu's material follows what is behind the layer it is on.
func TestLayerMaterial(t *testing.T) {
	tint := paintengine2d.RGBA(1, 1, 1, 0.8)
	bg := paintengine2d.RGBA(0, 0, 0, 1)
	defer SetLayerBackdrop(BackdropWindow)
	if c, blur := LayerMaterial(tint, bg); !blur || c != tint {
		t.Fatalf("in the window: %v %v", c, blur)
	}
	SetLayerBackdrop(BackdropGlass)
	if c, blur := LayerMaterial(tint, bg); blur || c != tint {
		t.Fatalf("over glass: %v %v", c, blur)
	}
	SetLayerBackdrop(BackdropNone)
	if c, blur := LayerMaterial(tint, bg); blur || c.A != 1 || c.R < 0.79 || c.R > 0.81 {
		t.Fatalf("over nothing: %v %v", c, blur)
	}
}

// Big Sur's menu bar and sidebar are flattened without real glass, and keep
// their material's alpha with it.
func TestBigSurMaterialsKeepAlphaOverRealGlass(t *testing.T) {
	lk := lookNamed(t, "bigsur").(*Classic)
	bar := func(glass bool) uint32 {
		SetGlassAvailable(glass)
		defer SetGlassAvailable(false)
		img := paintengine2d.NewImage(200, 40)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Transparent)
		lk.DrawMenuBar(ctx, paintengine2d.XYWH(0, 0, 200, 40))
		_, _, _, a := img.At(100, 10).RGBA()
		return a >> 8
	}
	if a := bar(false); a != 255 {
		t.Fatalf("flattened menu bar alpha %d", a)
	}
	if a := bar(true); a == 255 || a == 0 {
		t.Fatalf("glass menu bar alpha %d, want the material's own", a)
	}
	SetGlassAvailable(true)
	side := lk.ViewBackground(StateSidebar)
	SetGlassAvailable(false)
	if side.A >= 1 {
		t.Fatalf("sidebar over real glass %v: still flattened", side)
	}
	if c := lk.ViewBackground(StateSidebar); c.A < 1 {
		t.Fatalf("sidebar without glass %v: not flattened", c)
	}
}
