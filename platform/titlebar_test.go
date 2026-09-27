package platform

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseButtonLayout(t *testing.T) {
	cases := map[string]ButtonLayout{
		// GNOME's default: GNOME Shell shows no app menu, so only close.
		"appmenu:close": {Right: []CaptionButton{CaptionClose}},
		// GTK's default and kde-gtk-config's mapping of KWin's defaults.
		"menu:minimize,maximize,close": {Left: []CaptionButton{CaptionMenu}, Right: []CaptionButton{CaptionMinimize, CaptionMaximize, CaptionClose}},
		"icon:minimize,maximize,close": {Left: []CaptionButton{CaptionMenu}, Right: []CaptionButton{CaptionMinimize, CaptionMaximize, CaptionClose}},
		// macOS style, everything on the left.
		"close,minimize,maximize:": {Left: []CaptionButton{CaptionClose, CaptionMinimize, CaptionMaximize}},
		// Split between the sides, spacers kept, unknown names dropped.
		"close:spacer,foo,minimize,maximize": {Left: []CaptionButton{CaptionClose}, Right: []CaptionButton{CaptionSpacer, CaptionMinimize, CaptionMaximize}},
		// No colon: all on the right.
		"minimize,close": {Right: []CaptionButton{CaptionMinimize, CaptionClose}},
		// A button is kept where it first appears.
		"close:close,maximize": {Left: []CaptionButton{CaptionClose}, Right: []CaptionButton{CaptionMaximize}},
		" : ":                  {},
	}
	for in, want := range cases {
		got := ParseButtonLayout(in)
		if !sameButtons(got.Left, want.Left) || !sameButtons(got.Right, want.Right) {
			t.Errorf("ParseButtonLayout(%q) = %v, want %v", in, got, want)
		}
	}
	l := ParseButtonLayout("icon:minimize,maximize,close")
	if l.String() != "icon:minimize,maximize,close" || !l.Has(CaptionClose) || l.Has(CaptionSpacer) {
		t.Errorf("round trip %q", l.String())
	}
}

func sameButtons(a, b []CaptionButton) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func TestKDEButtonLayout(t *testing.T) {
	// KWin's defaults: MSE on the left, HIAX on the right.
	l := kdeButtonLayout("MSE", "HIAX")
	if !sameButtons(l.Left, []CaptionButton{CaptionMenu}) ||
		!sameButtons(l.Right, []CaptionButton{CaptionMinimize, CaptionMaximize, CaptionClose}) {
		t.Errorf("defaults %v", l)
	}
	l = kdeButtonLayout("X", "IA")
	if !sameButtons(l.Left, []CaptionButton{CaptionClose}) || !sameButtons(l.Right, []CaptionButton{CaptionMinimize, CaptionMaximize}) {
		t.Errorf("close on the left %v", l)
	}
	l = kdeButtonLayout("XI_A", "NFBLHS")
	if !sameButtons(l.Left, []CaptionButton{CaptionClose, CaptionMinimize, CaptionSpacer, CaptionMaximize}) ||
		!sameButtons(l.Right, []CaptionButton{CaptionKeepAbove}) {
		t.Errorf("spacer, keep above and dropped buttons %v", l)
	}
	// F is keep above; N, B, L, H and S still have no toolkit equivalent.
	if l := kdeButtonLayout("", "NBLHS"); len(l.Left) != 0 || len(l.Right) != 0 {
		t.Errorf("buttons with no equivalent %v", l)
	}
}

func TestTitleActions(t *testing.T) {
	for in, want := range map[string]TitleAction{
		"toggle-maximize": TitleToggleMaximize, "toggle-maximize-vertically": TitleToggleMaximizeVertically,
		"toggle-maximize-horizontally": TitleToggleMaximizeHorizontally, "minimize": TitleMinimize,
		"lower": TitleLower, "menu": TitleMenu, "none": TitleNone, "toggle-shade": TitleToggleShade, "'menu'": TitleMenu,
	} {
		if got, ok := ParseTitleAction(in); !ok || got != want {
			t.Errorf("ParseTitleAction(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := ParseTitleAction("fly"); ok {
		t.Error("unknown action parsed")
	}
	for in, want := range map[string]TitleAction{
		"Maximize": TitleToggleMaximize, "Maximize (vertical only)": TitleToggleMaximizeVertically,
		"Maximize (horizontal only)": TitleToggleMaximizeHorizontally, "Minimize": TitleMinimize,
		"Lower": TitleLower, "Close": TitleClose, "Nothing": TitleNone, "Shade": TitleToggleShade,
		"Keep above": TitleToggleKeepAbove, "Keep below": TitleNone,
	} {
		if got, ok := kdeDoubleClickAction(in); !ok || got != want {
			t.Errorf("kdeDoubleClickAction(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if a, ok := kdeMouseAction("Operations menu"); !ok || a != TitleMenu {
		t.Error("operations menu")
	}
	if a, ok := kdeMouseAction("Nothing"); !ok || a != TitleNone {
		t.Error("nothing")
	}
	if _, ok := kdeMouseAction(""); ok {
		t.Error("an unset binding is no action")
	}
}

func TestDefaultTitleBarPrefs(t *testing.T) {
	kde := DefaultTitleBarPrefs("KDE")
	if kde.Layout.String() != "icon:minimize,maximize,close" || kde.DoubleClick != TitleToggleMaximize ||
		kde.RightClick != TitleMenu || kde.MiddleClick != TitleNone || kde.DragThreshold != 10 {
		t.Errorf("KDE %+v", kde)
	}
	gnome := DefaultTitleBarPrefs("ubuntu:GNOME")
	if gnome.Layout.String() != ":close" || gnome.DragThreshold != 8 || gnome.DoubleClickTime != 400*time.Millisecond {
		t.Errorf("GNOME %+v", gnome)
	}
	other := DefaultTitleBarPrefs("")
	if other.Layout.String() != ":minimize,maximize,close" {
		t.Errorf("other %+v", other)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTitleBarPrefsFromKWin(t *testing.T) {
	user, sys := t.TempDir(), t.TempDir()
	dirs := []string{user, sys}
	// Nothing configured: KWin's defaults.
	p := titleBarPrefsFrom("KDE", dirs, DesktopPrefs{})
	if p.Layout.String() != "icon:minimize,maximize,close" || p.Source != "default" {
		t.Fatalf("empty %+v", p)
	}
	writeFile(t, filepath.Join(sys, "kwinrc"), "[org.kde.kdecoration2]\nButtonsOnLeft=XIA\n[Windows]\nTitlebarDoubleClickCommand=Minimize\n")
	writeFile(t, filepath.Join(user, "kwinrc"), `# user settings
[General]
Foo=bar

[org.kde.kdecoration2]
ButtonsOnLeft=X
ButtonsOnRight[$e]=IA
theme=Oxygen

[Windows]
TitlebarDoubleClickCommand=Maximize (vertical only)

[MouseBindings]
CommandActiveTitlebar2=Minimize
CommandActiveTitlebar3=Operations menu
`)
	writeFile(t, filepath.Join(user, "kdeglobals"), "[KDE]\nDoubleClickInterval=250\nStartDragDist=4\n")
	p = titleBarPrefsFrom("KDE", dirs, DesktopPrefs{ButtonLayout: "close,minimize:"})
	if p.Layout.String() != "close:minimize,maximize" || p.Source != "kwinrc" {
		t.Errorf("layout %q from %s", p.Layout.String(), p.Source)
	}
	if p.DoubleClick != TitleToggleMaximizeVertically || p.MiddleClick != TitleMinimize || p.RightClick != TitleMenu {
		t.Errorf("actions %+v", p)
	}
	if p.DoubleClickTime != 250*time.Millisecond || p.DragThreshold != 4 {
		t.Errorf("timing %+v", p)
	}
	// Only one side set: the other keeps KWin's default.
	writeFile(t, filepath.Join(user, "kwinrc"), "[org.kde.kdecoration2]\nButtonsOnRight=X\n")
	os.Remove(filepath.Join(sys, "kwinrc"))
	if p := titleBarPrefsFrom("KDE", dirs, DesktopPrefs{}); p.Layout.String() != "icon:close" {
		t.Errorf("one side %q", p.Layout.String())
	}
}

func TestTitleBarPrefsFromPortalAndGTK(t *testing.T) {
	user := t.TempDir()
	dirs := []string{user}
	writeFile(t, filepath.Join(user, "gtk-3.0", "settings.ini"), `[Settings]
gtk-decoration-layout=icon:minimize,maximize,close
gtk-titlebar-double-click=minimize
gtk-double-click-time=500
gtk-dnd-drag-threshold=12
`)
	p := titleBarPrefsFrom("XFCE", dirs, DesktopPrefs{})
	if p.Layout.String() != "icon:minimize,maximize,close" || p.Source != "gtk" || p.DoubleClick != TitleMinimize ||
		p.DoubleClickTime != 500*time.Millisecond || p.DragThreshold != 12 {
		t.Errorf("gtk %+v", p)
	}
	// GTK 4's file wins over GTK 3's.
	writeFile(t, filepath.Join(user, "gtk-4.0", "settings.ini"), "[Settings]\ngtk-decoration-layout=close:\n")
	if p := titleBarPrefsFrom("XFCE", dirs, DesktopPrefs{}); p.Layout.String() != "close:" {
		t.Errorf("gtk4 %q", p.Layout.String())
	}
	// The portal wins over files.
	portal := DesktopPrefs{ButtonLayout: "appmenu:minimize,close", TitlebarDoubleClick: "toggle-maximize",
		TitlebarRightClick: "none", DoubleClickTime: 300, DragThreshold: 6}
	p = titleBarPrefsFrom("GNOME", dirs, portal)
	if p.Layout.String() != ":minimize,close" || p.Source != "portal" || p.DoubleClick != TitleToggleMaximize ||
		p.RightClick != TitleNone || p.DoubleClickTime != 300*time.Millisecond || p.DragThreshold != 6 {
		t.Errorf("portal %+v", p)
	}
}

func TestParseINI(t *testing.T) {
	ini := parseINI(strings.NewReader("top=1\n[A]\nk = v \n; comment\n# comment\nName[de]=Hallo\nName=Hello\nFlag[$e]=$HOME\n[A][B]\nx=y\nbroken line\n"))
	if ini[""]["top"] != "1" || ini["A"]["k"] != "v" || ini["A"]["Name"] != "Hello" || ini["A"]["Flag"] != "$HOME" || ini["A][B"]["x"] != "y" {
		t.Errorf("ini %v", ini)
	}
}

func TestTitleBarConfigFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	t.Setenv("XDG_CONFIG_DIRS", "/etc/a:/etc/b")
	files := TitleBarConfigFiles()
	if len(files) == 0 || files[0] != "/cfg/kwinrc" {
		t.Fatalf("files %v", files)
	}
	joined := strings.Join(files, " ")
	for _, want := range []string{"/etc/b/kdeglobals", "/cfg/gtk-4.0/settings.ini"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %s in %v", want, files)
		}
	}
}

// The wheel over the title bar: KWin's Shade/Unshade is the one command the
// toolkit has, the rest read as none, and an unset key leaves the toolkit's
// own default (roll the window up) standing.
func TestTitlebarWheel(t *testing.T) {
	for in, want := range map[string]TitleWheel{
		"Shade/Unshade": TitleWheelShade, "Nothing": TitleWheelNone,
		"Raise/Lower": TitleWheelNone, "Maximize/Restore": TitleWheelNone,
		"Above/Below": TitleWheelNone, "Change Opacity": TitleWheelNone,
	} {
		if got, ok := kdeTitlebarWheel(in); !ok || got != want {
			t.Errorf("kdeTitlebarWheel(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := kdeTitlebarWheel("Spin"); ok {
		t.Error("unknown wheel command parsed")
	}
	if p := DefaultTitleBarPrefs("KDE"); p.Wheel != TitleWheelShade {
		t.Errorf("default wheel %v: a toolkit title bar rolls the window up", p.Wheel)
	}
}

// The keep-above button's name round-trips through the layout syntax.
func TestKeepAboveButtonName(t *testing.T) {
	if s := CaptionKeepAbove.String(); s != "keep-above" {
		t.Errorf("name %q", s)
	}
	l := ParseButtonLayout("keep-above:close")
	if !sameButtons(l.Left, []CaptionButton{CaptionKeepAbove}) || !sameButtons(l.Right, []CaptionButton{CaptionClose}) {
		t.Errorf("parsed %v", l)
	}
	if l.String() != "keep-above:close" {
		t.Errorf("round trip %q", l.String())
	}
	// The style package converts the values numerically, so they must line
	// up across the two packages' constants.
	if CaptionKeepAbove != 6 || CaptionSpacer != 5 {
		t.Errorf("caption button values moved: spacer %d, keep above %d (style.CaptionButton mirrors them)", CaptionSpacer, CaptionKeepAbove)
	}
}

// A rolled-up window is pinned to one height and cannot be resized
// vertically; unpinning gives its own limits back.
func TestShadePin(t *testing.T) {
	l := SizeLimits{MinWidth: 200, MinHeight: 120, MaxWidth: 900}
	if got := shadePin(l, 0); got != l {
		t.Errorf("unpinned %+v", got)
	}
	got := shadePin(l, 28)
	if got.MinHeight != 28 || got.MaxHeight != 28 {
		t.Errorf("pinned height %+v", got)
	}
	if got.MinWidth != 200 || got.MaxWidth != 900 {
		t.Errorf("the pin took the width with it: %+v", got)
	}
}

// _NET_WM_STATE_ABOVE is the keep-above state; Wayland never reports one.
func TestNetWMStateAbove(t *testing.T) {
	if st := netWMState(netStateAbove, true, false); !st.KeepAbove {
		t.Error("_NET_WM_STATE_ABOVE not decoded")
	}
	if st := netWMState(netStateMaxVert|netStateMaxHorz, true, false); st.KeepAbove {
		t.Error("keep above set without the atom")
	}
	if st := xdgStateFromMask(1 << 4); st.KeepAbove {
		t.Error("xdg-shell has no keep-above state to report")
	}
}

// The offscreen desktop keeps a window above only when a test says it can,
// and answers the request with the state, as a window manager does.
func TestOffscreenKeepAbove(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 300, Height: 200})
	if FrameCapsOf(o).Has(FrameKeepAbove) {
		t.Error("an offscreen window has no desktop to be stacked in")
	}
	if FrameOf(o).SetKeepAbove(true) {
		t.Error("asked a desktop that cannot")
	}
	o.SimulateKeepAbove(true)
	if !FrameOf(o).SetKeepAbove(true) || !o.WindowState().KeepAbove {
		t.Fatalf("keep above %+v", o.WindowState())
	}
	if calls := o.FrameCalls().Aboves; len(calls) != 1 || !calls[0] {
		t.Errorf("requests %v", calls)
	}
	if !FrameOf(o).SetShadedHeight(30) {
		t.Fatal("no shade pin")
	}
	if l := o.SizeLimits(); l.MinHeight != 30 || l.MaxHeight != 30 {
		t.Errorf("pinned limits %+v", l)
	}
	FrameOf(o).SetShadedHeight(0)
	if l := o.SizeLimits(); l.MinHeight == 30 && l.MaxHeight == 30 {
		t.Errorf("the pin outlived the roll-up: %+v", l)
	}
}
