package platform

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// This file reads the desktop's title-bar conventions — which caption
// buttons go on which side, what double-, middle- and right-clicking the
// title bar does, the double-click time and the drag threshold — so a
// toolkit-drawn title bar behaves like the desktop's own. Sources, by
// desktop: KDE Plasma's kwinrc and kdeglobals (the settings portal carries
// none of KWin's keys), GNOME's org.gnome.desktop.wm.preferences through
// the settings portal, GTK's settings.ini, then defaults. Only documented
// setting names and values are used.

// CaptionButton is a button a title bar can show.
type CaptionButton uint8

const (
	CaptionNone CaptionButton = iota
	CaptionClose
	CaptionMinimize
	CaptionMaximize
	// CaptionMenu opens the window menu (KDE's "M" button, GTK's "icon").
	CaptionMenu
	// CaptionSpacer is a gap between buttons.
	CaptionSpacer
)

func (b CaptionButton) String() string {
	switch b {
	case CaptionClose:
		return "close"
	case CaptionMinimize:
		return "minimize"
	case CaptionMaximize:
		return "maximize"
	case CaptionMenu:
		return "icon"
	case CaptionSpacer:
		return "spacer"
	}
	return ""
}

// ButtonLayout is the caption buttons at each end of the title bar, both
// lists in left-to-right order.
type ButtonLayout struct {
	Left, Right []CaptionButton
}

// String is the layout in GNOME's syntax ("icon:minimize,maximize,close").
func (l ButtonLayout) String() string {
	side := func(bs []CaptionButton) string {
		names := make([]string, 0, len(bs))
		for _, b := range bs {
			names = append(names, b.String())
		}
		return strings.Join(names, ",")
	}
	return side(l.Left) + ":" + side(l.Right)
}

// Has reports whether b is on either side.
func (l ButtonLayout) Has(b CaptionButton) bool {
	for _, x := range append(append([]CaptionButton(nil), l.Left...), l.Right...) {
		if x == b {
			return true
		}
	}
	return false
}

// ParseButtonLayout reads GNOME's button-layout / GTK's
// gtk-decoration-layout syntax: button names separated by commas, a colon
// between the left and the right side ("appmenu:close",
// "close,minimize:"). minimize, maximize and close are the buttons; icon
// and menu become the window-menu button; spacer is a gap; anything else
// (appmenu, which GNOME Shell no longer shows) is skipped, and a button is
// kept only where it first appears. Without a colon every button is on
// the right.
func ParseButtonLayout(s string) ButtonLayout {
	left, right, found := strings.Cut(strings.TrimSpace(s), ":")
	if !found {
		left, right = "", left
	}
	seen := map[CaptionButton]bool{}
	side := func(part string) []CaptionButton {
		var out []CaptionButton
		for _, name := range strings.Split(part, ",") {
			b := captionButtonNamed(strings.TrimSpace(name))
			if b == CaptionNone || (b != CaptionSpacer && seen[b]) {
				continue
			}
			seen[b] = true
			out = append(out, b)
		}
		return out
	}
	return ButtonLayout{Left: side(left), Right: side(right)}
}

func captionButtonNamed(name string) CaptionButton {
	switch strings.ToLower(name) {
	case "close":
		return CaptionClose
	case "minimize":
		return CaptionMinimize
	case "maximize":
		return CaptionMaximize
	case "icon", "menu":
		return CaptionMenu
	case "spacer":
		return CaptionSpacer
	}
	return CaptionNone
}

// KDEButtonLayout reads KWin's ButtonsOnLeft / ButtonsOnRight letters
// ([org.kde.kdecoration2] in kwinrc): M window menu, I minimize, A
// maximize, X close, _ a spacer. KWin's other buttons (N application
// menu, S on all desktops, H help, F keep above, B keep below, L shade, E
// exclude from capture) have no toolkit equivalent and are skipped.
func KDEButtonLayout(left, right string) ButtonLayout {
	seen := map[CaptionButton]bool{}
	side := func(letters string) []CaptionButton {
		var out []CaptionButton
		for _, r := range letters {
			var b CaptionButton
			switch r {
			case 'M':
				b = CaptionMenu
			case 'I':
				b = CaptionMinimize
			case 'A':
				b = CaptionMaximize
			case 'X':
				b = CaptionClose
			case '_':
				b = CaptionSpacer
			default:
				continue
			}
			if b != CaptionSpacer && seen[b] {
				continue
			}
			seen[b] = true
			out = append(out, b)
		}
		return out
	}
	return ButtonLayout{Left: side(left), Right: side(right)}
}

// TitleAction is what clicking the title bar does.
type TitleAction uint8

const (
	TitleNone TitleAction = iota
	TitleToggleMaximize
	// TitleToggleMaximizeVertically / Horizontally maximize one way only;
	// xdg-shell cannot, so on Wayland they toggle-maximize.
	TitleToggleMaximizeVertically
	TitleToggleMaximizeHorizontally
	TitleMinimize
	// TitleLower sends the window behind the others (X11 only).
	TitleLower
	// TitleMenu shows the window menu.
	TitleMenu
	TitleClose
)

func (a TitleAction) String() string {
	switch a {
	case TitleToggleMaximize:
		return "toggle-maximize"
	case TitleToggleMaximizeVertically:
		return "toggle-maximize-vertically"
	case TitleToggleMaximizeHorizontally:
		return "toggle-maximize-horizontally"
	case TitleMinimize:
		return "minimize"
	case TitleLower:
		return "lower"
	case TitleMenu:
		return "menu"
	case TitleClose:
		return "close"
	}
	return "none"
}

// ParseTitleAction reads GNOME's action-*-titlebar and GTK's
// gtk-titlebar-*-click values. toggle-shade has no toolkit equivalent and
// reads as none; ok is false for an unknown value.
func ParseTitleAction(s string) (TitleAction, bool) {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(s), `"'`)) {
	case "toggle-maximize":
		return TitleToggleMaximize, true
	case "toggle-maximize-vertically":
		return TitleToggleMaximizeVertically, true
	case "toggle-maximize-horizontally":
		return TitleToggleMaximizeHorizontally, true
	case "minimize":
		return TitleMinimize, true
	case "lower":
		return TitleLower, true
	case "menu":
		return TitleMenu, true
	case "none", "toggle-shade":
		return TitleNone, true
	}
	return TitleNone, false
}

// kdeDoubleClickAction reads KWin's [Windows] TitlebarDoubleClickCommand.
func kdeDoubleClickAction(s string) (TitleAction, bool) {
	switch strings.TrimSpace(s) {
	case "Maximize":
		return TitleToggleMaximize, true
	case "Maximize (vertical only)":
		return TitleToggleMaximizeVertically, true
	case "Maximize (horizontal only)":
		return TitleToggleMaximizeHorizontally, true
	case "Minimize":
		return TitleMinimize, true
	case "Lower":
		return TitleLower, true
	case "Close":
		return TitleClose, true
	case "Nothing", "Shade", "OnAllDesktops", "Keep above", "Keep below":
		return TitleNone, true
	}
	return TitleNone, false
}

// kdeMouseAction reads KWin's [MouseBindings] CommandActiveTitlebar2 /
// CommandActiveTitlebar3 (middle and right click on an active window's
// title bar).
func kdeMouseAction(s string) (TitleAction, bool) {
	switch strings.TrimSpace(s) {
	case "Operations menu":
		return TitleMenu, true
	case "Minimize":
		return TitleMinimize, true
	case "Lower", "Toggle raise and lower":
		return TitleLower, true
	case "Close":
		return TitleClose, true
	case "Maximize":
		return TitleToggleMaximize, true
	case "Nothing", "Raise", "Activate and raise", "Activate", "Activate and lower", "Shade", "Move", "Resize":
		return TitleNone, true
	}
	return TitleNone, false
}

// TitleBarPrefs are the desktop's title-bar conventions.
type TitleBarPrefs struct {
	Layout      ButtonLayout
	DoubleClick TitleAction
	MiddleClick TitleAction
	RightClick  TitleAction
	// DoubleClickTime is the longest gap between the clicks of a double
	// click; DragThreshold how far (logical px) a pressed pointer moves
	// before a press becomes a drag.
	DoubleClickTime time.Duration
	DragThreshold   float32
	// Source says where the layout came from ("kwinrc", "portal", "gtk",
	// "default"), for diagnostics.
	Source string
}

// IsKDE reports whether desktop (XDG_CURRENT_DESKTOP) is KDE Plasma.
func IsKDE(desktop string) bool { return desktopHas(desktop, "kde") }

// IsGNOME reports whether desktop (XDG_CURRENT_DESKTOP) is GNOME.
func IsGNOME(desktop string) bool { return desktopHas(desktop, "gnome") }

func desktopHas(desktop, name string) bool {
	for _, d := range strings.Split(desktop, ":") {
		if strings.EqualFold(strings.TrimSpace(d), name) {
			return true
		}
	}
	return false
}

// DefaultTitleBarPrefs are the conventions of desktop (XDG_CURRENT_DESKTOP)
// when it says nothing: KWin's defaults on KDE (window menu on the left;
// minimize, maximize, close on the right), GNOME's close-only layout,
// minimize, maximize and close elsewhere; double-click maximizes, a right
// click shows the window menu.
func DefaultTitleBarPrefs(desktop string) TitleBarPrefs {
	p := TitleBarPrefs{
		Layout:          ParseButtonLayout(":minimize,maximize,close"),
		DoubleClick:     TitleToggleMaximize,
		MiddleClick:     TitleNone,
		RightClick:      TitleMenu,
		DoubleClickTime: 400 * time.Millisecond,
		DragThreshold:   8,
		Source:          "default",
	}
	switch {
	case IsKDE(desktop):
		p.Layout = KDEButtonLayout("MSE", "HIAX")
		p.DragThreshold = 10 // Qt's and KDE's StartDragDist
	case IsGNOME(desktop):
		p.Layout = ParseButtonLayout("appmenu:close")
	}
	return p
}

// ReadTitleBarPrefs reads the conventions of the running desktop from its
// configuration files (XDG_CONFIG_HOME, XDG_CONFIG_DIRS). portal holds the
// settings portal's GNOME keys when the caller has them (WatchDesktopPrefs).
func ReadTitleBarPrefs(portal DesktopPrefs) TitleBarPrefs {
	return titleBarPrefsFrom(os.Getenv("XDG_CURRENT_DESKTOP"), configDirs(), portal)
}

// TitleBarConfigFiles are the files ReadTitleBarPrefs reads, so a caller can
// notice when one changes.
func TitleBarConfigFiles() []string {
	var out []string
	for _, dir := range configDirs() {
		for _, name := range []string{"kwinrc", "kdeglobals", filepath.Join("gtk-4.0", "settings.ini"), filepath.Join("gtk-3.0", "settings.ini")} {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out
}

// configDirs are the XDG configuration directories, most important first.
func configDirs() []string {
	var dirs []string
	if home := os.Getenv("XDG_CONFIG_HOME"); home != "" {
		dirs = append(dirs, home)
	} else if h, err := os.UserHomeDir(); err == nil && h != "" {
		dirs = append(dirs, filepath.Join(h, ".config"))
	}
	sys := os.Getenv("XDG_CONFIG_DIRS")
	if sys == "" {
		sys = "/etc/xdg"
	}
	for _, d := range strings.Split(sys, ":") {
		if d = strings.TrimSpace(d); d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// titleBarPrefsFrom resolves the conventions from dirs (most important
// first) and the portal's GNOME keys.
func titleBarPrefsFrom(desktop string, dirs []string, portal DesktopPrefs) TitleBarPrefs {
	p := DefaultTitleBarPrefs(desktop)
	if IsKDE(desktop) {
		kwin := readKConfig(dirs, "kwinrc")
		globals := readKConfig(dirs, "kdeglobals")
		deco := kwin["org.kde.kdecoration2"]
		left, okL := deco["ButtonsOnLeft"]
		right, okR := deco["ButtonsOnRight"]
		if okL || okR {
			if !okL {
				left = "MSE"
			}
			if !okR {
				right = "HIAX"
			}
			p.Layout = KDEButtonLayout(left, right)
			p.Source = "kwinrc"
		}
		if a, ok := kdeDoubleClickAction(kwin["Windows"]["TitlebarDoubleClickCommand"]); ok {
			p.DoubleClick = a
		}
		if a, ok := kdeMouseAction(kwin["MouseBindings"]["CommandActiveTitlebar2"]); ok {
			p.MiddleClick = a
		}
		if a, ok := kdeMouseAction(kwin["MouseBindings"]["CommandActiveTitlebar3"]); ok {
			p.RightClick = a
		}
		if ms, err := strconv.Atoi(globals["KDE"]["DoubleClickInterval"]); err == nil && ms > 0 {
			p.DoubleClickTime = time.Duration(ms) * time.Millisecond
		}
		if px, err := strconv.Atoi(globals["KDE"]["StartDragDist"]); err == nil && px > 0 {
			p.DragThreshold = float32(px)
		}
		return p
	}
	// GNOME and the rest: the portal's keys, else GTK's own settings.
	gtk := readGTKSettings(dirs)
	switch {
	case portal.ButtonLayout != "":
		p.Layout = ParseButtonLayout(portal.ButtonLayout)
		p.Source = "portal"
	case gtk["gtk-decoration-layout"] != "":
		p.Layout = ParseButtonLayout(gtk["gtk-decoration-layout"])
		p.Source = "gtk"
	}
	for _, x := range []struct {
		dst          *TitleAction
		portal, gtkK string
	}{
		{&p.DoubleClick, portal.TitlebarDoubleClick, "gtk-titlebar-double-click"},
		{&p.MiddleClick, portal.TitlebarMiddleClick, "gtk-titlebar-middle-click"},
		{&p.RightClick, portal.TitlebarRightClick, "gtk-titlebar-right-click"},
	} {
		if a, ok := ParseTitleAction(x.portal); ok {
			*x.dst = a
		} else if a, ok := ParseTitleAction(gtk[x.gtkK]); ok {
			*x.dst = a
		}
	}
	if portal.DoubleClickTime > 0 {
		p.DoubleClickTime = time.Duration(portal.DoubleClickTime) * time.Millisecond
	} else if ms, err := strconv.Atoi(gtk["gtk-double-click-time"]); err == nil && ms > 0 {
		p.DoubleClickTime = time.Duration(ms) * time.Millisecond
	}
	if portal.DragThreshold > 0 {
		p.DragThreshold = float32(portal.DragThreshold)
	} else if px, err := strconv.Atoi(gtk["gtk-dnd-drag-threshold"]); err == nil && px > 0 {
		p.DragThreshold = float32(px)
	}
	return p
}

// readKConfig reads a KDE configuration file from every dir, the most
// important last so it wins (KConfig's cascade): group → key → value.
func readKConfig(dirs []string, name string) map[string]map[string]string {
	out := map[string]map[string]string{}
	for i := len(dirs) - 1; i >= 0; i-- {
		f, err := os.Open(filepath.Join(dirs[i], name))
		if err != nil {
			continue
		}
		for group, kv := range parseINI(f) {
			if out[group] == nil {
				out[group] = map[string]string{}
			}
			for k, v := range kv {
				out[group][k] = v
			}
		}
		f.Close()
	}
	return out
}

// readGTKSettings reads [Settings] of GTK 4's, else GTK 3's settings.ini
// from the most important dir that has one.
func readGTKSettings(dirs []string) map[string]string {
	for _, dir := range dirs {
		for _, v := range []string{"gtk-4.0", "gtk-3.0"} {
			f, err := os.Open(filepath.Join(dir, v, "settings.ini"))
			if err != nil {
				continue
			}
			ini := parseINI(f)
			f.Close()
			if s := ini["Settings"]; len(s) > 0 {
				return s
			}
		}
	}
	return map[string]string{}
}

// parseINI reads a KConfig / GLib key file: [Group] headers (KConfig's
// nested [A][B] kept as "A][B"), key=value lines, # and ; comments. KConfig
// flags on a key ("Key[$e]") are dropped; localised keys ("Key[de]") are
// skipped.
func parseINI(r interface{ Read([]byte) (int, error) }) map[string]map[string]string {
	out := map[string]map[string]string{}
	group := ""
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			if end := strings.LastIndexByte(line, ']'); end > 0 {
				group = line[1:end]
			}
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if i := strings.IndexByte(key, '['); i >= 0 {
			if !strings.HasPrefix(key[i:], "[$") {
				continue
			}
			key = key[:i]
		}
		if out[group] == nil {
			out[group] = map[string]string{}
		}
		out[group][key] = strings.TrimSpace(val)
	}
	return out
}
