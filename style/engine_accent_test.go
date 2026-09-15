package style

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// accCase is a pack whose engine takes the desktop's accent.
type accCase struct {
	pack string
	// own is the pack's own accent, the one its platform shipped: applied as
	// the desktop's accent it gives the pack back. Empty: the platform had
	// no accent there and the look keeps its colours.
	own string
	// floor is the contrast the platform keeps for text and marks on its
	// accent.
	floor float64
}

var accCases = []accCase{
	{"fluent", "#0078d4", 4.5}, {"fluent-night", "#0078d4", 4.5},
	{"win8", "#6ba5e7", 3}, {"win10", "#0078d7", 3}, {"win10-night", "#0078d7", 3},
	{"adwaita", "#3584e4", 3}, {"adwaita-night", "#3584e4", 3}, {"adwaita-gtk3", "", 0},
	{"yosemite", "", 0}, {"bigsur", "#007aff", 2.2}, {"bigsur-night", "#007aff", 2.2},
	{"tahoe", "#0088ff", 2.2}, {"tahoe-night", "#0091ff", 2.2}, {"adwaita48", "#3584e4", 3}, {"adwaita48-night", "#3584e4", 3},
	{"material", "#6200ee", 4.5}, {"material-night", "#6200ee", 4.5},
	{"material3", "#6750a4", 4.5}, {"material3-night", "#6750a4", 4.5},
	{"flatlaf", "#2675bf", 2.2}, {"flatlaf-night", "#4b6eaf", 3}, {"flatlaf-darcula", "#4b6eaf", 3},
	{"fusion", "#308cc6", 3}, {"fusion-night", "#2a82da", 3},
	{"oxygen", "#43ace8", 2.2},
	{"aero", "#74b8fc", 0}, {"aero-basic", "", 0},
	{"sourcegit", "#0078d7", 3}, {"sourcegit-night", "#0078d7", 3}, {"primer", "", 0}, {"catppuccin-mocha", "", 0},
}

// accSamples are desktop accents: Ubuntu's orange, GNOME's yellow 2 and
// green 4 (pale and mid-tone), GNOME 47's purple.
var accSamples = []string{"#e95420", "#f6d32d", "#26a269", "#9141ac"}

// accLook is the pack's look as an app that follows the desktop builds it
// while the desktop's accent is accent.
func accLook(t *testing.T, pack string, accent paintengine2d.Color, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(pack)
	if !ok {
		t.Fatalf("pack %q not registered", pack)
	}
	SetDesktopAccent(accent, true)
	defer SetDesktopAccent(paintengine2d.Color{}, false)
	a := p.Appearance()
	a.FollowDesktop = true
	var lk LookAndFeel = a.Look()
	if scale != 1 {
		lk = WithScale(lk, scale)
	}
	return lk.(*Classic)
}

// accPlain is the pack's own look.
func accPlain(t *testing.T, pack string, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(pack)
	if !ok {
		t.Fatalf("pack %q not registered", pack)
	}
	var lk LookAndFeel = p.Look()
	if scale != 1 {
		lk = WithScale(lk, scale)
	}
	return lk.(*Classic)
}

// accHex prints a colour for failure messages.
func accHex(c paintengine2d.Color) string {
	q := func(v float32) int { return int(math.Round(float64(clamp1(v)) * 255)) }
	if c.A < 1 {
		return fmt.Sprintf("#%02x%02x%02x%02x", q(c.R), q(c.G), q(c.B), q(c.A))
	}
	return fmt.Sprintf("#%02x%02x%02x", q(c.R), q(c.G), q(c.B))
}

// accHue is c's Oklch hue and chroma.
func accHue(c paintengine2d.Color) (h, chroma float64) {
	_, chroma, h = okLCh(c)
	return h, chroma
}

// accTakers are the engines whose looks follow the desktop's accent: the
// platforms that let the user pick one (and Breeze, the first).
var accTakers = []string{"adwaita", "aero", "breeze", "flatlaf", "fluent", "fusion", "macos", "material", "metro", "oxygen", "web"}

// Only the engines of platforms with an accent take it; the historical
// looks keep their fixed palettes.
func TestAccentTakenOnlyWhereThePlatformHadOne(t *testing.T) {
	var got []string
	for _, id := range EngineIDs() {
		e, _ := EngineByID(id)
		if _, ok := e.(AccentEngine); ok {
			got = append(got, id)
		}
	}
	sort.Strings(got)
	if strings.Join(got, " ") != strings.Join(accTakers, " ") {
		t.Fatalf("engines taking the accent: %v, want %v", got, accTakers)
	}
	for _, n := range []string{"win95", "luna", "aqua", "motif", "platinum", "next", "amiga13", "os2warp", "beos", "keramik", "plastik", "clearlooks", "bluecurve"} {
		if p, ok := LoadTheme(n); ok && TakesAccent(p) {
			t.Errorf("%s (a historical look) takes the accent", n)
		}
	}
}

// accSnapshot copies what a pack registered, maps included.
type accSnapshot struct {
	tok    ThemeTokens
	extra  map[string]paintengine2d.Color
	params map[string]float32
}

func accSnap(pack string) accSnapshot {
	p, _ := LoadTheme(pack)
	s := accSnapshot{tok: p.Tokens, extra: map[string]paintengine2d.Color{}, params: map[string]float32{}}
	for k, v := range p.Tokens.Extra {
		s.extra[k] = v
	}
	for k, v := range p.Tokens.Params {
		s.params[k] = v
	}
	return s
}

// Following an accent never writes into the registered pack.
func TestAccentLeavesTheRegisteredPackAlone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range accCases {
		before := accSnap(c.pack)
		for _, s := range accSamples {
			accLook(t, c.pack, Hex(s), 1)
		}
		p, _ := LoadTheme(c.pack)
		if p.Tokens.Palette != before.tok.Palette || p.Tokens.Hot != before.tok.Hot || p.Tokens.Pressed != before.tok.Pressed ||
			p.Tokens.Selected != before.tok.Selected || p.Tokens.Focus != before.tok.Focus || p.Tokens.Disabled != before.tok.Disabled {
			t.Errorf("%s: the registered palette or chrome changed", c.pack)
		}
		if len(p.Tokens.Extra) != len(before.extra) || len(p.Tokens.Params) != len(before.params) {
			t.Errorf("%s: the registered maps grew: extra %d→%d, params %d→%d", c.pack, len(before.extra), len(p.Tokens.Extra), len(before.params), len(p.Tokens.Params))
		}
		for k, v := range before.extra {
			if p.Tokens.Extra[k] != v {
				t.Errorf("%s: registered extra %q changed to %s", c.pack, k, accHex(p.Tokens.Extra[k]))
			}
		}
		for k, v := range before.params {
			if p.Tokens.Params[k] != v {
				t.Errorf("%s: registered param %q changed", c.pack, k)
			}
		}
	}
}

// accCell is one control painted into a w×h rect (1× design pixels).
type accCell struct {
	name string
	w, h float32
	draw func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
}

// accCells are the controls that wear an accent, in the states that show it.
func accCells() []accCell {
	var cells []accCell
	add := func(n string, w, h float32, f func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)) {
		cells = append(cells, accCell{n, w, h, f})
	}
	for _, st := range []ControlState{
		StateNone, StateHovered, StatePressed | StateHovered, StateFocused, StateDisabled, StatePrimary,
		StatePrimary | StateHovered, StatePrimary | StatePressed, StatePrimary | StateFocused, StateToggle | StateChecked,
		StateBackdrop | StatePrimary,
	} {
		st := st
		sn := fmt.Sprintf("%#x", uint32(st))
		add("button "+sn, 90, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawButton(ctx, b, st, "Button")
		})
		add("tool "+sn, 90, 34, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawToolButton(ctx, b, st, "Fetch", IconOpen)
		})
		for _, on := range []bool{false, true} {
			on := on
			add(fmt.Sprintf("check %s %v", sn, on), 110, 24, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawCheckbox(ctx, b, st, on, "Check")
			})
			add(fmt.Sprintf("radio %s %v", sn, on), 110, 24, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawRadio(ctx, b, st, on, "Radio")
			})
			add(fmt.Sprintf("switch %s %v", sn, on), 120, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawSwitch(ctx, b, st, on, "Switch")
			})
			add(fmt.Sprintf("tab %s %v", sn, on), 96, 32, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawTab(ctx, b, st, "Tab", on)
			})
			add(fmt.Sprintf("combo %s %v", sn, on), 130, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawComboBox(ctx, b, st, "Choice", on)
			})
		}
		add("slider "+sn, 110, 28, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawSlider(ctx, b, st, 0.6) })
		add("progress "+sn, 110, 18, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawProgressBar(ctx, b, st, 0.7, false, 0)
		})
		add("busy "+sn, 110, 18, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawProgressBar(ctx, b, st, 0, true, 0.4)
		})
		add("field "+sn, 160, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTextField(ctx, b, st, "Ada Lovelace", "", 3, 0, 3, true, 0, nil)
		})
		add("menu "+sn, 200, 26, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			ch := MenuChromeFor(lk)
			row := paintengine2d.XYWH(b.Min.X+ch.PadL, b.Min.Y, b.Dx()-ch.PadL-ch.PadR, b.Dy())
			lk.DrawMenuItem(ctx, row, st, MenuRow{Label: "Word wrap", Checked: true, Shortcut: "Ctrl+W"})
		})
		add("menu title "+sn, 60, 26, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawMenuTitle(ctx, b, st, "File", 0, st.Pressed())
		})
		add("header "+sn, 90, 26, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTableHeader(ctx, b, st, "Name", true, true)
		})
	}
	for _, st := range []ControlState{
		StateChecked, StateChecked | StateFocused, StateChecked | StateHovered, StateHovered, StateFocused,
		StateChecked | StateInactive, StateChecked | StateBackdrop, StateChecked | StateDisabled,
	} {
		st := st
		sn := fmt.Sprintf("%#x", uint32(st))
		add("list "+sn, 170, 26, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawListRow(ctx, b, st, "List row")
		})
		add("tree "+sn, 170, 26, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTreeRow(ctx, b, st, true, false, 1, "Archives", false)
		})
		add("cell "+sn, 80, 26, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTableCell(ctx, b, st|StateFirst, "Cell", AlignStart, nil)
		})
		add("item focus "+sn, 170, 26, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawItemFocus(ctx, b, st) })
		add("view frame "+sn, 160, 90, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawViewFrame(ctx, b, st) })
	}
	add("text area", 200, 60, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawTextArea(ctx, b, StateFocused, []TextLine{{Text: "A text area", Start: 0, End: 11}, {Text: "two lines", Start: 12, End: 21}}, 5, 2, 15, true, 0, 0, "", nil)
	})
	for _, ws := range []WindowState{{Active: true, CanClose: true}, {CanClose: true}, {Active: true, CanClose: true, CloseHot: true}} {
		ws := ws
		add(fmt.Sprintf("window %+v", ws), 280, 160, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawWindowFrame(ctx, b, "Dialog", ws)
		})
	}
	add("group", 240, 120, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawGroupBox(ctx, b, "Group", false)
	})
	add("tooltip", 160, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTooltip(ctx, b, "A tip") })
	add("focus ring", 120, 34, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawFocusRing(ctx, b) })
	add("accordion", 150, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawAccordionHeader(ctx, b, StateHovered, "Section", true)
	})
	add("splitter", 12, 90, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawSplitter(ctx, b, true, StateHovered)
	})
	add("label", 150, 24, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawLabel(ctx, b, "Accent label", lk.Palette().Accent, AlignStart)
	})
	for _, ss := range []ScrollState{{}, {Hot: ScrollThumbPart, Hovered: true}, {Pressed: ScrollThumbPart}} {
		ss := ss
		add(fmt.Sprintf("scroll %+v", ss), 17, 120, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
			DrawScrollBarParts(lk, ctx, ScrollGeometry(lk, b, true, 500, b.Dy(), 100, false), true, ss)
		})
	}
	return cells
}

// accPaint paints a cell of lk into an image with a margin round it.
func accPaint(lk *Classic, c accCell) (*paintengine2d.Image, paintengine2d.Rect) {
	sc := lk.Scale()
	m := 8 * sc
	img := paintengine2d.NewImage(int(c.w*sc+2*m), int(c.h*sc+2*m))
	b := paintengine2d.XYWH(m, m, c.w*sc, c.h*sc)
	c.draw(lk, paintengine2d.NewContext(img), b)
	return img, b
}

// accDiff is the first pixel where a and b differ by more than tol (0–255).
func accDiff(a, b *paintengine2d.Image, tol int) (x, y int, ok bool) {
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			r0, g0, b0, a0 := a.PremulAt(x, y)
			r1, g1, b1, a1 := b.PremulAt(x, y)
			d := func(p, q uint8) bool { return int(p) > int(q)+tol || int(q) > int(p)+tol }
			if d(r0, r1) || d(g0, g1) || d(b0, b1) || d(a0, a1) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// A pack's own accent gives the pack back: every control paints as the
// plain pack does, within one step of 255 on every channel, and the looks
// that took no accent paint as they are with any.
func TestAccentOwnAccentGivesThePackBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cells := accCells()
	for _, c := range accCases {
		accent := c.own
		if accent == "" {
			accent = accSamples[0]
		}
		plain, lk := accPlain(t, c.pack, 1), accLook(t, c.pack, Hex(accent), 1)
		pa, pb := plain.Palette(), lk.Palette()
		for _, f := range [][3]any{
			{"accent", pa.Accent, pb.Accent}, {"selection", pa.Selection, pb.Selection}, {"focus", pa.Focus, pb.Focus},
			{"text on accent", pa.TextOnAccent, pb.TextOnAccent}, {"accent hover", pa.AccentHover, pb.AccentHover},
			{"accent press", pa.AccentPress, pb.AccentPress}, {"menu hover", pa.MenuHover, pb.MenuHover},
			{"background", pa.Background, pb.Background}, {"surface", pa.Surface, pb.Surface},
		} {
			if a, b := f[1].(paintengine2d.Color), f[2].(paintengine2d.Color); !accentSame(a, b) {
				t.Errorf("%s with %s: palette %s %s, pack %s", c.pack, accent, f[0], accHex(b), accHex(a))
			}
		}
		ta, tb := plain.Tokens(), lk.Tokens()
		for _, s := range [][3]any{{"hot", ta.Hot, tb.Hot}, {"pressed", ta.Pressed, tb.Pressed}, {"selected", ta.Selected, tb.Selected}, {"focus", ta.Focus, tb.Focus}, {"disabled", ta.Disabled, tb.Disabled}} {
			a, b := s[1].(ChromeState), s[2].(ChromeState)
			if !accentSame(a.Fill, b.Fill) || !accentSame(a.Border, b.Border) {
				t.Errorf("%s with %s: %s state %s/%s, pack %s/%s", c.pack, accent, s[0], accHex(b.Fill), accHex(b.Border), accHex(a.Fill), accHex(a.Border))
			}
		}
		for _, cell := range cells {
			ia, _ := accPaint(plain, cell)
			ib, _ := accPaint(lk, cell)
			if x, y, bad := accDiff(ia, ib, 1); bad {
				r0, g0, b0, a0 := ia.PremulAt(x, y)
				r1, g1, b1, a1 := ib.PremulAt(x, y)
				t.Errorf("%s with %s: %s differs at (%d,%d): %d,%d,%d,%d vs the pack's %d,%d,%d,%d",
					c.pack, accent, cell.name, x, y, r1, g1, b1, a1, r0, g0, b0, a0)
				break
			}
		}
	}
}

// accShown is the colour the look paints its accent in, where its platform
// put it, and what that colour should be for the desktop accent a.
type accShown struct {
	got, want paintengine2d.Color
	what      string
}

// accWhere lists, for a look built around the accent a, the colours its
// engine now paints with and what each should be.
func accWhere(t *testing.T, pack string, lk *Classic, a paintengine2d.Color) []accShown {
	t.Helper()
	switch lk.Engine().ID() {
	case "fluent":
		c := fluentColors(lk)
		// Windows' own shade of the accent: darker in the light theme,
		// lighter in the dark one.
		want := accentShift(a, fluentSystemAccent, Hex(fluentLight["accent"]))
		aL, _, _ := okLCh(a)
		fL, _, _ := okLCh(c.accent[0])
		if c.dark {
			want = accentShift(a, fluentSystemAccent, Hex(fluentDark["accent"]))
			if fL <= aL {
				t.Errorf("%s: the dark theme's fill %s is not lighter than the accent", pack, accHex(c.accent[0]))
			}
		} else if fL >= aL {
			t.Errorf("%s: the light theme's fill %s is not darker than the accent", pack, accHex(c.accent[0]))
		}
		return []accShown{{c.accent[0], want, "accent fill"}, {c.accent[1], want.WithAlpha(0xe6 / 255.0), "hover fill"},
			{c.accent[2], want.WithAlpha(0xcc / 255.0), "pressed fill"}, {lk.Palette().Selection, accentShift(a, fluentSystemAccent, Hex("#0067c0")), "text selection"}}
	case "metro":
		c := metroColors(lk)
		if !c.win10 {
			return []accShown{{c.frame, a, "window frame"}, {c.accent, Hex("#3399ff"), "Windows 8's fixed highlight"}}
		}
		return []accShown{{c.accent, a, "accent"}, {c.btn[4][1], a, "default button border"}, {c.ed[2], a, "focused edit line"},
			{c.slThumb[0], a, "slider thumb"}, {lk.Palette().Selection, a, "selection"}}
	case "adwaita":
		c := adwColors(lk)
		bg := adwNearestAccent(a)
		return []accShown{{c.accentBg, bg, "accent_bg_color"}, {c.accent, adwStandalone(bg, c.dark), "accent_color"}, {c.accentFg, Hex("#ffffff"), "accent_fg_color"}}
	case "macos":
		if _, ok := lk.Engine().(tahoeEngine); ok {
			c := tahoeColors(lk)
			sc := map[bool]macScheme{false: tahoeLight, true: tahoeDark}[c.dark]
			return []accShown{{c.accent, a, "accent"}, {c.menuHi, a, "menu highlight"},
				{c.sel, accentShift(a, Hex(sc["accent"]), Hex(sc["sel"])), "list selection"},
				{c.selOff, Hex(sc["selOff"]), "unemphasized selection (grey)"}}
		}
		c := macColors(lk)
		return []accShown{{c.accent, a, "accent"}, {c.accentStops[1].Color, a, "default button"}, {c.checkStops[1].Color, a, "checked box"},
			{c.selOff, Hex(map[bool]string{false: macBigSur["selOff"], true: macBigSurDark["selOff"]}[c.dark]), "unemphasized selection (grey)"}}
	case "material":
		c := mdColors(lk)
		if c.m3 {
			return []accShown{{lk.X("seed", paintengine2d.Color{}), a, "seed"}, {c.primary, mdSchemeFrom(mdCorePalette(a), c.dark).primary, "primary"}}
		}
		if c.dark {
			return []accShown{{c.primary, accentShift(a, md2Baseline, Hex("#bb86fc")), "dark primary (the 200)"}}
		}
		return []accShown{{c.primary, a, "primary"}, {c.primaryC, accentShift(a, md2Baseline, Hex("#3700b3")), "primary variant (the 700)"}}
	case "flatlaf":
		c := flatColors(lk)
		base2 := flatLighten(flatSaturate(a, 10), 6)
		if c.dark {
			base2 = flatLighten(flatSaturate(flatSpin(a, -8), 13), 5)
		}
		return []accShown{{c.sel, a, "selection"}, {c.accent, a, "accent"}, {c.slider, base2, "slider"}, {c.progress, base2, "progress"}}
	case "fusion":
		c := fusionColors(lk)
		return []accShown{{c.hl, a, "highlight"}, {lk.Palette().Selection, a, "selection"}}
	case "oxygen":
		c := oxygenColors(lk)
		return []accShown{{c.hl, a, "selection"}, {c.focus, accentShift(a, Hex("#43ace8"), Hex("#3aa7dd")), "focus decoration"},
			{c.hover, accentShift(a, Hex("#43ace8"), Hex("#6ed6ff")), "hover decoration"}}
	case "aero":
		c := aeroColors(lk)
		return []accShown{{c.glassGrad[1].Color, accentShift(a, aeroWindowColour, Hex(aeroBase["glass1"])), "glass"},
			{lk.Palette().Accent, Hex("#3399ff"), "controls' highlight"}}
	case "web":
		c := webColors(lk)
		return []accShown{{c.accent, a, "accent"}, {c.primary, a, "primary button"}, {c.sel, a, "selection"},
			{c.accentHover, webLight1(a), "hover (Light1)"}, {c.accentPress, webDark1(a), "pressed (Dark1)"}}
	}
	t.Fatalf("%s: no probe for engine %q", pack, lk.Engine().ID())
	return nil
}

// The accent shows where each platform put it, in the shades it derived,
// and the historical halves of the engines (GNOME 3, Yosemite, Windows 7
// Basic) keep their colours.
func TestAccentShowsWhereThePlatformPutIt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range accCases {
		for _, s := range accSamples {
			a := Hex(s)
			lk := accLook(t, c.pack, a, 1)
			if c.own == "" {
				plain := accPlain(t, c.pack, 1)
				if lk.Palette() != plain.Palette() || len(lk.Tokens().Extra) != len(plain.Tokens().Extra) {
					t.Errorf("%s with %s: the colours changed; the platform had no accent there", c.pack, s)
				}
				continue
			}
			for _, w := range accWhere(t, c.pack, lk, a) {
				if !accentSame(w.got, w.want) {
					t.Errorf("%s with %s: %s is %s, want %s", c.pack, s, w.what, accHex(w.got), accHex(w.want))
				}
			}
			// The painted accent: the controls that wear it carry its hue.
			want := accWhere(t, c.pack, lk, a)[0].want
			wh, wc := accHue(want)
			n := 0
			for _, cell := range accCells() {
				if n >= 400 {
					break
				}
				img, _ := accPaint(lk, cell)
				for y := 0; y < img.Height; y++ {
					for x := 0; x < img.Width; x++ {
						r, g, b, al := img.PremulAt(x, y)
						if al < 200 {
							continue
						}
						h, ch := accHue(paintengine2d.RGB(float32(r)/float32(al), float32(g)/float32(al), float32(b)/float32(al)))
						if ch > 0.04 && math.Abs(okTurn(h, wh)) < 20 {
							n++
						}
					}
				}
			}
			if wc > 0.04 && n < 400 {
				t.Errorf("%s with %s: only %d painted pixels carry the accent's hue", c.pack, s, n)
			}
		}
	}
}

// Text and marks on the accent keep the contrast the platform kept: a pale
// accent (GNOME's yellow here) gets dark text.
func TestAccentTextStaysReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range accCases {
		if c.floor == 0 {
			continue
		}
		for _, s := range append([]string{c.own}, accSamples...) {
			lk := accLook(t, c.pack, Hex(s), 1)
			type pair struct {
				what   string
				fg, bg paintengine2d.Color
			}
			var pairs []pair
			switch lk.Engine().ID() {
			case "fluent":
				f := fluentColors(lk)
				pairs = []pair{{"accent button", f.onAccent[0], f.accent[0]}}
			case "metro":
				m := metroColors(lk)
				if m.win10 {
					pairs = []pair{{"text on accent", m.onAccent, m.accent}, {"selected text", lk.selectedText(lk.Palette().Selection), lk.Palette().Selection}}
				}
			case "adwaita":
				a := adwColors(lk)
				pairs = []pair{{"suggested action", a.accentFg, a.accentBg}}
			case "macos":
				if _, ok := lk.Engine().(tahoeEngine); ok {
					m := tahoeColors(lk)
					pairs = []pair{{"default button", m.onAccent, m.accent}, {"list selection", m.onAccent, m.sel}, {"menu highlight", m.onAccent, m.menuHi}}
					break
				}
				m := macColors(lk)
				pairs = []pair{{"default button", m.onAccent, m.accentStops[0].Color}, {"checked box", m.onAccent, m.checkStops[0].Color},
					{"list selection", m.onAccent, m.sel}, {"menu highlight", m.onAccent, m.menuHi}}
			case "material":
				m := mdColors(lk)
				pairs = []pair{{"contained button", m.onPrimary, m.primary}}
			case "flatlaf":
				f := flatColors(lk)
				pairs = []pair{{"selection", f.selText, f.sel}}
				if f.dark {
					pairs = append(pairs, pair{"default button", f.defText, f.def})
				}
			case "fusion":
				f := fusionColors(lk)
				pairs = []pair{{"highlighted text", f.hlText, f.hl}}
			case "oxygen":
				pairs = []pair{{"selection", lk.Palette().TextOnAccent, lk.Palette().Selection}}
			case "web":
				w := webColors(lk)
				pairs = []pair{{"primary button", w.onPrimary, w.primary}, {"text on accent", w.onAccent, w.accent}}
			}
			for _, p := range pairs {
				if r := ContrastRatio(p.fg, p.bg); r < c.floor-0.005 {
					t.Errorf("%s with %s: %s %s on %s is %.2f:1, under %.1f", c.pack, s, p.what, accHex(p.fg), accHex(p.bg), r, c.floor)
				}
			}
		}
	}
	// GNOME's yellow and a pale yellow on Windows 11 and FlatLaf: dark text.
	for _, n := range []string{"fluent", "flatlaf", "bigsur", "fusion", "material"} {
		lk := accLook(t, n, Hex("#f6d32d"), 1)
		if on := lk.Palette().TextOnAccent; RelLuminance(on) > 0.2 {
			t.Errorf("%s with a pale yellow accent: text on it is %s, want dark", n, accHex(on))
		}
	}
}

// Every control of every accented look still paints inside its rect
// (aquaExercise, the KDE extras and the accent cells), with a dark and a
// pale accent at 1× and the pale one at 2×: the accent moves colours, never
// geometry.
func TestAccentLooksPaintInsideTheirRects(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range accCases {
		if c.own == "" {
			continue
		}
		for i, s := range accSamples[:2] {
			scales := []float32{1}
			if i == 1 {
				scales = append(scales, 2)
			}
			for _, sc := range scales {
				lk := accLook(t, c.pack, Hex(s), sc)
				name := fmt.Sprintf("%s+%s@%gx", c.pack, s, sc)
				// Fluent's floating bar sits a pixel in from the view's
				// edge, so it does not fit aquaExercise's bar-wide views
				// (its own check, winPaintsInsideRect, gives it a wider
				// one, as accCells does).
				if lk.Engine().ID() != "fluent" {
					aquaExercise(t, name, lk)
				}
				kdeExtras(t, name, lk)
				for _, cell := range accCells() {
					img, b := accPaint(lk, cell)
					if x, y, out := aquaOutside(img, b); out {
						t.Errorf("%s: %s painted outside its rect at (%d,%d)", name, cell.name, x, y)
					}
				}
				// Degenerate rects: nothing panics or leaves a saved state.
				img := paintengine2d.NewImage(64, 64)
				ctx := paintengine2d.NewContext(img)
				for _, r := range []paintengine2d.Rect{{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(0, 0, 5, 40), paintengine2d.XYWH(0, 0, 40, 5)} {
					for _, st := range lunaStates {
						lunaCalls(lk, ctx, r, st)
					}
					lunaStateless(lk, ctx, r)
				}
				if ctx.SaveCount() != 0 {
					t.Errorf("%s: %d saved states left", name, ctx.SaveCount())
				}
			}
		}
	}
}

// GNOME 47's nine accents are libadwaita's: each maps to itself, and its
// text-safe accent_color, derived in Oklab, is the documented one for the
// light and the dark style. Any other colour takes the nearest by hue.
func TestAdwaitaAccentsAreGNOMEs(t *testing.T) {
	for _, g := range [][3]string{
		{"#3584e4", "#0461be", "#81d0ff"}, {"#2190a4", "#007184", "#7bdff4"}, {"#3a944a", "#15772e", "#8de698"},
		{"#c88800", "#905300", "#ffc057"}, {"#ed5b00", "#b62200", "#ff9c5b"}, {"#e62d42", "#c00023", "#ff888c"},
		{"#d56199", "#a2326c", "#ffa0d8"}, {"#9141ac", "#8939a4", "#fba7ff"}, {"#6f8396", "#526678", "#bbd1e5"},
	} {
		bg := Hex(g[0])
		if got := adwNearestAccent(bg); !accentSame(got, bg) {
			t.Errorf("GNOME's %s maps to %s", g[0], accHex(got))
		}
		if got := adwStandalone(bg, false); !accentSame(got, Hex(g[1])) {
			t.Errorf("%s: light accent_color %s, GNOME's %s", g[0], accHex(got), g[1])
		}
		if got := adwStandalone(bg, true); !accentSame(got, Hex(g[2])) {
			t.Errorf("%s: dark accent_color %s, GNOME's %s", g[0], accHex(got), g[2])
		}
		if r := ContrastRatio(Hex("#ffffff"), bg); r < 3 {
			t.Errorf("%s: white on it is %.2f:1", g[0], r)
		}
	}
	for in, want := range map[string]string{
		"#e95420": "#ed5b00", "#f6d32d": "#c88800", "#26a269": "#3a944a", "#3daee9": "#3584e4",
		"#808080": "#6f8396", "#ff00ff": "#9141ac", "#0078d4": "#3584e4", "#1a5fb4": "#3584e4",
	} {
		if got := adwNearestAccent(Hex(in)); !accentSame(got, Hex(want)) {
			t.Errorf("desktop accent %s shows as %s, want %s", in, accHex(got), want)
		}
	}
}

// FlatLaf's formulas give its published values from its own base colours.
func TestFlatLafAccentFormulasGiveItsTables(t *testing.T) {
	for _, c := range []struct {
		pack string
		sc   flatScheme
		base string
	}{{"flatlaf", flatLight, "#2675bf"}, {"flatlaf-night", flatDark, "#4b6eaf"}} {
		p, _ := LoadTheme(c.pack)
		tok := flatlafEngine{}.Accented(p.Tokens.Resolve(), Hex(c.base))
		n := 0
		for k, v := range tok.Extra {
			want, ok := c.sc[k]
			if !ok {
				continue
			}
			n++
			if !accentSame(v, Hex(want)) {
				t.Errorf("%s: %s is %s, FlatLaf's %s", c.pack, k, accHex(v), want)
			}
		}
		if n < 20 {
			t.Errorf("%s: only %d accent properties derived", c.pack, n)
		}
	}
}

// The shade and wash steps: a colour's own accent gives it back, a wash
// of the accent over white stays that wash of any accent, and a shade keeps
// its place above or below the accent.
func TestAccentShadesAndWashes(t *testing.T) {
	ref := Hex("#0078d7")
	for _, tc := range []string{"#0078d7", "#429ce3", "#005499", "#cce4f7", "#e5f1fb", "#1c3f5e", "#0067c080", "#99d1ff"} {
		target := Hex(tc)
		if got := accentShift(ref, ref, target); !accentSame(got, target) {
			t.Errorf("accentShift(ref, ref, %s) = %s", tc, accHex(got))
		}
		if got := accentWash(ref, ref, target); !accentSame(got, target) {
			t.Errorf("accentWash(ref, ref, %s) = %s", tc, accHex(got))
		}
	}
	// #cce4f7 is Windows 10's accent at 20% over white.
	for _, s := range accSamples {
		a := Hex(s)
		want := Mix(Hex("#ffffff"), a, 0.2)
		if got := accentWash(a, ref, Hex("#cce4f7")); !accentSame(got, want) {
			t.Errorf("the 20%% wash of %s is %s, want %s", s, accHex(got), accHex(want))
		}
		aL, _, _ := okLCh(a)
		if l, _, _ := okLCh(accentShift(a, ref, Hex("#005499"))); l >= aL {
			t.Errorf("the pressed border of %s is not darker than it", s)
		}
		if l, _, _ := okLCh(accentShift(a, ref, Hex("#429ce3"))); l <= aL {
			t.Errorf("Light1 of %s is not lighter than it", s)
		}
	}
	// A grey accent (Windows' and macOS's graphite) gives grey shades.
	if _, ch := accHue(accentShift(Hex("#808080"), ref, Hex("#005499"))); ch > 0.01 {
		t.Errorf("a grey accent's shade has chroma %.3f", ch)
	}
}
