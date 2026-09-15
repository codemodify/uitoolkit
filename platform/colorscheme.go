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
