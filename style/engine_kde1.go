package style

import "github.com/codemodify/paintengine2d"

// kde1Engine paints KDE 1 (1998), the first K Desktop Environment, on
// Qt 1.x. Its widgets are not KDE's own and not Motif's: KDE 1 shipped
// with Qt 1's Windows 95 look-alike, and said so in its own Display
// Settings handbook — "Draw widgets in the style of Windows 95 … If this
// option is not selected, they will have a Motif style" — with Windows as
// the default of the two. So this engine embeds the win95 painter, which
// already draws that shape language, and changes only what KDE 1 did not
// take from Windows:
//
//   - the window. Windows ran the caption bar the full width of the frame
//     and put raised button boxes on it; kwm sank a *coloured strip* into
//     a four-pixel raised grey frame and left the buttons outside it, flat
//     line-art on the grey — the window menu and the sticky pin at the
//     left, iconify, maximize and close at the right, with a gap before
//     close. The strip's colour runs *across* its width: the focused
//     window blends #000080 into black, the rest blend #808080 into the
//     window grey, and the bold title sits at the left over the deep end;
//   - the bevel shades. Qt derived them from the button colour rather than
//     reading Windows' system colours, so the inner light is #d8d8d8 and
//     the inner shadow #606060 against Windows' #dfdfdf and #808080;
//   - the typeface. KDE 1's general font was Helvetica 12 and its title
//     font the same in bold, where Windows read in MS Sans Serif 8.
//
// Everything else — the four-colour bevel, the dotted focus rectangle, the
// navy selection with white text, the dithered scroll troughs with an
// arrow at each end, the chamfered tabs — is engine_win95.go's, because
// that is what a KDE 1 application looked like.
//
// Sources. No KDE or Qt code, pixmap or data is used; these are documented
// values and pixels measured on published screenshots:
//
//   - kde.org/announcements/1-2-3/1.0/ — KDE 1.0, 12 July 1998, on Qt 1;
//     en.wikipedia.org/wiki/K_Desktop_Environment_1 — kwm was its window
//     manager;
//   - KDE 1's Display Settings handbook (the "Style" page) for the two
//     widget styles and which one is the default, and its kdisplay page
//     for the twelve-colour scheme with a "blend" colour for each title
//     bar; KDE 1's KWM control-module handbook for the title-bar look
//     ("Shaded Horizontally: fills the bar with a color gradient from side
//     to side", the default), the left title alignment, the shaded title
//     frame, and the five button functions with at most three a side;
//   - the colour scheme KDE shipped to reproduce this look, KDEOne.kcsrc:
//     background #c0c0c0, foreground black, windowBackground white,
//     selectBackground #000080 on white, activeBackground #000080 with
//     activeBlend #000000, inactiveBackground #808080 with inactiveBlend
//     #c0c0c0, contrast 7 — the same values KDE 1 itself fell back to;
//   - kwm's metrics: a four-pixel border, a twenty-pixel title row with a
//     two-pixel separation (an eighteen-pixel strip), twenty-pixel
//     buttons and a two-pixel gap before close;
//   - Wikimedia Commons "KDE Beta3.png" and "KDE 1.1.jpg". Measured on the
//     first: the push-button bevel #ffffff / #d8d8d8 over #c0c0c0 over
//     #606060 / #000000, the title strip sunken into the frame behind a
//     dark line above and a light line below, and the caption glyphs
//     outside the strip on the grey. Measured on the second: the active
//     strip running #000080 at the left into black at the right and the
//     inactive one #808080 into #c0c0c0.
//
// Judgement calls. The much-reproduced "KDE 1.0.png" screenshot is *not*
// in the default scheme — its #d6ceb9 widgets are KDE 1's shipped "Desert
// red" (background 214,205,187), so it was not used for colour. And the
// inactive title is written in black here: KDEOne.kcsrc asks for #c0c0c0
// on the #808080 end of the blend, which is a 2.6:1 label, while the
// KDE 1.1 screenshot shows a dark one.
//
// Pack data and params are engine_win95.go's.
type kde1Engine struct{ win95Engine }

func init() {
	RegisterEngine(kde1Engine{})
	for _, p := range kde1Packs() {
		RegisterPack(p)
	}
}

func (kde1Engine) ID() string { return "kde1" }

// StyleHint: KDE's dialogs put the accepting button first ("OK Apply
// Cancel"), where Windows put it last.
func (kde1Engine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return win95Engine{}.StyleHint(l, h)
}

// ---- packs --------------------------------------------------------------------------------

func kde1Packs() []ThemePack {
	return []ThemePack{kde1Pack("kde1", "KDE 1", 1998,
		"KDE 1 on Qt 1: Windows 95 shapes in Helvetica, and kwm's title strip sunk into the frame, fading from navy to black.")}
}

func kde1Pack(name, label string, year int, summary string) ThemePack {
	pal := w95Palette("#c0c0c0", "#000000", "#ffffff", "#000000", "#000080", "#ffffff", "#808080", "#0000ff")
	tok := ThemeTokens{
		Engine:  "kde1",
		Bevel:   BevelClassic3D,
		Family:  ThemeLight,
		Palette: pal,
		Era:     EraKDE1,
		Extra: map[string]paintengine2d.Color{
			// Qt derived the bevel from the button colour: white and the
			// button lightened, the button halved and black.
			"hi": Hex("#ffffff"), "light": Hex("#d8d8d8"),
			"shadow": Hex("#606060"), "dk": Hex("#000000"),
			// kwm's blends (KDEOne.kcsrc).
			"caption": Hex("#000080"), "caption2": Hex("#000000"), "captionText": Hex("#ffffff"),
			"captionOff": Hex("#808080"), "captionOff2": Hex("#c0c0c0"), "captionOffText": Hex("#000000"),
			// Qt 1's tooltip.
			"info": Hex("#ffffe1"), "infoText": Hex("#000000"),
		},
	}
	tok.Hot = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "KDE", Summary: summary,
		Era: EraKDE1, Palette: ThemeLight, Tokens: tok,
	}
}

// ---- the one widget KDE 1 drew for itself -------------------------------------------------

// ToolBarInsets leaves room at the left for KToolBar's drag handle.
func (kde1Engine) ToolBarInsets(l *Classic) Insets {
	return Insets{Top: l.S(3), Right: l.S(4), Bottom: l.S(3), Left: l.S(14)}
}

// DrawToolBar is a Windows tool bar with KToolBar's handle at its left: a
// narrow strip of diagonal lines, each a light one beside a shadow one,
// which is the one piece of chrome KDE 1 drew that Windows 95 did not.
// Measured on the kfm tool bar of the KDE 1.1 screenshot.
func (e kde1Engine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	win95Engine{}.DrawToolBar(l, ctx, b)
	c := w95colors(l)
	u := max(snap(l.S(1)), 1)
	w := snap(l.S(9))
	if b.Dx() < w*3 || b.Dy() < 6*u {
		return
	}
	top, bot := snap(b.Min.Y+l.S(2)), snap(b.Max.Y-l.S(2))
	x0 := snap(b.Min.X + l.S(2))
	// Lines at 45°, stepped a unit at a time so they land on the pixel
	// grid the way an X11 hatch did, and clipped to the strip.
	hi, lo := paintengine2d.Fill(c.hi), paintengine2d.Fill(c.shadow)
	for start := x0 - (bot - top); start < x0+w; start += 4 * u {
		for k := float32(0); top+k*u < bot; k++ {
			y := top + k*u
			x := start + k*u
			if x >= x0 && x+u <= x0+w {
				ctx.DrawRect(paintengine2d.XYWH(x, y, u, u), hi)
			}
			if x+u >= x0 && x+2*u <= x0+w {
				ctx.DrawRect(paintengine2d.XYWH(x+u, y, u, u), lo)
			}
		}
	}
}
