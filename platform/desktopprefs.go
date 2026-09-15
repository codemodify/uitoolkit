package platform

// ColorScheme is the desktop's light / dark preference: the freedesktop
// portal's org.freedesktop.appearance color-scheme, which GTK 4, Qt 6 and
// the browsers follow.
type ColorScheme int

const (
	// SchemeNoPreference: the desktop does not say.
	SchemeNoPreference ColorScheme = iota
	// SchemeDark: the desktop prefers dark.
	SchemeDark
	// SchemeLight: the desktop prefers light.
	SchemeLight
)

// DesktopPrefs are the desktop's appearance preferences, as the XDG
// desktop portal publishes them (org.freedesktop.appearance): GNOME's
// Style, Plasma's colour scheme and accent, and both desktops' animation
// and contrast settings.
type DesktopPrefs struct {
	ColorScheme ColorScheme
	// ReducedMotion: the user asked for fewer animations.
	ReducedMotion bool
	// HighContrast: the user asked for higher contrast.
	HighContrast bool
	// Accent is the accent colour (sRGB, 0..1) when HasAccent is set.
	Accent    [3]float64
	HasAccent bool
	// ButtonLayout and the Titlebar* actions are GNOME's window-manager
	// preferences (org.gnome.desktop.wm.preferences: button-layout,
	// action-double-click-titlebar, …) as the GNOME and GTK portal backends
	// publish them; DoubleClickTime (ms) and DragThreshold (px) are
	// org.gnome.desktop.peripherals.mouse's. Empty / zero when the desktop
	// does not say (KDE's portal publishes none of them).
	ButtonLayout        string
	TitlebarDoubleClick string
	TitlebarMiddleClick string
	TitlebarRightClick  string
	DoubleClickTime     int
	DragThreshold       int
}

// parseScheme maps the portal's value (0 none, 1 dark, 2 light).
func parseScheme(v uint32) ColorScheme {
	switch v {
	case 1:
		return SchemeDark
	case 2:
		return SchemeLight
	}
	return SchemeNoPreference
}
