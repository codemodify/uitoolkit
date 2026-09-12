package style

// ToolIcon is a stock glyph drawn by LookAndFeel.
// File icon sets use one SVG per id (new.svg, open.svg, …) under
// ~/.config/uitoolkit/icons/<set>/. Classic / Sharp remain drawn fallbacks.
type ToolIcon int

const (
	IconNone ToolIcon = iota
	IconNew
	IconOpen
	IconSave
	IconCut
	IconCopy
	IconPaste
	IconUndo
	IconRedo
	IconSearch
	IconInfo
	IconWarning
	IconError
	IconQuestion
)

// toolIconFiles maps each chrome action to its SVG basename (no extension).
var toolIconFiles = []struct {
	icon ToolIcon
	name string
}{
	{IconNew, "new"},
	{IconOpen, "open"},
	{IconSave, "save"},
	{IconCut, "cut"},
	{IconCopy, "copy"},
	{IconPaste, "paste"},
	{IconUndo, "undo"},
	{IconRedo, "redo"},
	{IconSearch, "search"},
	{IconInfo, "info"},
	{IconWarning, "warning"},
	{IconError, "error"},
	{IconQuestion, "question"},
}

// AllToolIcons is every ToolIcon that has a file name (not IconNone).
func AllToolIcons() []ToolIcon {
	out := make([]ToolIcon, len(toolIconFiles))
	for i, e := range toolIconFiles {
		out[i] = e.icon
	}
	return out
}

// ToolIconName is the action id used for SVG files (open, new, cut, …).
func ToolIconName(icon ToolIcon) string {
	for _, e := range toolIconFiles {
		if e.icon == icon {
			return e.name
		}
	}
	return ""
}

// ToolIconFileName is the SVG file for icon (open.svg). Empty for IconNone.
func ToolIconFileName(icon ToolIcon) string {
	if name := ToolIconName(icon); name != "" {
		return name + ".svg"
	}
	return ""
}

// ToolIconByName maps a file stem ("open") to a ToolIcon.
func ToolIconByName(name string) (ToolIcon, bool) {
	for _, e := range toolIconFiles {
		if e.name == name {
			return e.icon, true
		}
	}
	return IconNone, false
}
