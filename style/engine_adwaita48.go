package style

import "github.com/codemodify/paintengine2d"

// adwaita48Engine paints libadwaita as GNOME 48 and later draw it
// (libadwaita 1.7 to 1.10): the adwaita engine's era for packs with param
// "era" 48. The "adwaita" packs keep GNOME 42–47's proportions, as the
// atlas shows them. What GNOME 48 changed, taken as numbers from its
// release notes and stylesheet values:
//
//   - buttons, entries, drop-downs, spin buttons, tabs, popover-menu and
//     sidebar rows round from 6 to 9px, popovers, dialogs and windows from
//     12 to 15px (adwEraR); cards and boxed lists keep 12px, check boxes
//     their 6;
//   - popovers lose their 1px border for a ring and two soft shadows
//     (1.10); dialogs float on 1.8's lighter three-layer shadow;
//   - the interface reads in Adwaita Sans (Inter with one variant frozen)
//     and Adwaita Mono, in place of Cantarell and Source Code Pro.
//
// Colours, washes, sizes and the focus ring are GNOME 47's.
type adwaita48Engine struct{ adwaitaEngine }

func init() {
	for _, p := range adwaita48Packs() {
		RegisterPack(p)
	}
}

// EngineFor paints packs with param "era" 48 with adwaita48Engine.
func (adwaitaEngine) EngineFor(t ThemeTokens) Engine {
	if t.Params["era"] >= 48 {
		return adwaita48Engine{}
	}
	return nil
}

// adwEraR is GNOME 42–47's design radius r as the look's GNOME draws it:
// GNOME 48 rounded the 6px controls, rows and menus to 9px and the 12px
// popovers, dialogs and windows to 15px.
func adwEraR(l *Classic, r float32) float32 {
	if l != nil && l.P("era", 0) >= 48 {
		switch r {
		case 6:
			return 9
		case 12:
			return 15
		}
	}
	return r
}

// adwCardR is the 12px corner of cards, boxed lists and frames, which GNOME
// 48 kept, for a shape of size b.
func adwCardR(l *Classic, b paintengine2d.Rect) float32 {
	return min(l.rx(12), adwPill(l, b))
}

// adw48Shadows are libadwaita 1.10's box-shadows as DropShadow layers (a CSS
// blur radius fades over twice its length): popovers a 1px ring of black at
// 5% under two soft shadows, tool tips 1.6's, dialogs 1.8's three layers.
// The dark style doubles them, as the engine's GNOME 47 shadows do.
func adw48Shadows(l *Classic, kind PopupKind) []adwShadow {
	k := l.P("shadow", 1)
	if k <= 0 {
		return nil
	}
	dark := adwColors(l).dark
	black := func(a float32) paintengine2d.Color {
		if dark {
			a *= 2
		}
		return paintengine2d.RGBA(0, 0, 0, min(a*k, 1))
	}
	switch kind {
	case PopupMenu:
		return []adwShadow{{black(0.05), l.S(2), l.S(28), l.S(3)}, {black(0.09), l.S(1), l.S(10), l.S(1)}, {black(0.05), 0, 0, l.S(1)}}
	case PopupTooltip:
		return []adwShadow{{black(0.12), l.S(1), l.S(6), 0}}
	case PopupDialog:
		return []adwShadow{{black(0.03), 0, l.S(28), l.S(2)}, {black(0.10), 0, l.S(10), l.S(2)}, {black(0.05), 0, 0, l.S(1)}}
	}
	return nil
}

func (adwaita48Engine) PopupShadow(l *Classic, kind PopupKind) Insets {
	var out Insets
	for _, s := range adw48Shadows(l, kind) {
		out = out.Max(ShadowReach(0, s.dy, s.blur, s.spread))
	}
	return out
}

func (adwaita48Engine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	r := adwR(l, 12, b)
	if kind == PopupTooltip {
		r = adwR(l, 6, b)
	}
	for _, s := range adw48Shadows(l, kind) {
		DropShadow(ctx, b, r, s.col, 0, s.dy, s.blur, s.spread)
	}
}

// DrawMenuFrame is a 1.10 popover: the popover colour in 15px corners; its
// edge is the ring its shadow draws.
func (adwaita48Engine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := adwColors(l)
	b = adwSnap(b)
	if b.Empty() {
		return
	}
	r := adwR(l, 12, b)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.popover))
}

// ---- packs --------------------------------------------------------------------------------------

func adwaita48Pack(name, label, summary string, fam ThemeName, scheme int) ThemePack {
	p := adwaitaPack(name, label, 2025, summary, fam, scheme, adwaitaPalette(scheme), ChromeMetrics{})
	p.Tokens.Params["era"] = 48
	return p
}

func adwaita48Packs() []ThemePack {
	return []ThemePack{
		adwaita48Pack("adwaita48", "Adwaita (GNOME 48)",
			"libadwaita as GNOME 48 draws it: 9px controls and rows, 15px popovers and dialogs, a ring for the popover's edge, Adwaita Sans.",
			ThemeLight, 0),
		adwaita48Pack("adwaita48-night", "Adwaita Dark (GNOME 48)",
			"GNOME 48's dark style: the rounder libadwaita shapes on #222226 windows, #1d1d20 views and #36363a popovers.",
			ThemeDark, 1),
	}
}
