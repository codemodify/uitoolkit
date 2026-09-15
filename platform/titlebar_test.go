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
	l := KDEButtonLayout("MSE", "HIAX")
	if !sameButtons(l.Left, []CaptionButton{CaptionMenu}) ||
		!sameButtons(l.Right, []CaptionButton{CaptionMinimize, CaptionMaximize, CaptionClose}) {
		t.Errorf("defaults %v", l)
	}
	l = KDEButtonLayout("X", "IA")
	if !sameButtons(l.Left, []CaptionButton{CaptionClose}) || !sameButtons(l.Right, []CaptionButton{CaptionMinimize, CaptionMaximize}) {
		t.Errorf("close on the left %v", l)
	}
	l = KDEButtonLayout("XI_A", "NFBLHS")
	if !sameButtons(l.Left, []CaptionButton{CaptionClose, CaptionMinimize, CaptionSpacer, CaptionMaximize}) || len(l.Right) != 0 {
		t.Errorf("spacer and dropped buttons %v", l)
	}
}

func TestTitleActions(t *testing.T) {
	for in, want := range map[string]TitleAction{
		"toggle-maximize": TitleToggleMaximize, "toggle-maximize-vertically": TitleToggleMaximizeVertically,
		"toggle-maximize-horizontally": TitleToggleMaximizeHorizontally, "minimize": TitleMinimize,
		"lower": TitleLower, "menu": TitleMenu, "none": TitleNone, "toggle-shade": TitleNone, "'menu'": TitleMenu,
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
		"Lower": TitleLower, "Close": TitleClose, "Nothing": TitleNone, "Shade": TitleNone,
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
