package style

// ToolIcon is a stock glyph drawn by LookAndFeel.
// File icon sets use one PNG per id (new.png, open.png, …) under
// ~/.config/uitoolkit/icons/<set>/. Classic / Sharp remain drawn fallbacks.
// HiDPI uses name@2x.png (48×48) when the destination is large enough.
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
	IconMail
)

// toolIconFiles maps each chrome action to its PNG basename (no extension).
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
	{IconMail, "mail"},
}

// AllToolIcons is every ToolIcon that has a file name (not IconNone).
func AllToolIcons() []ToolIcon {
	out := make([]ToolIcon, len(toolIconFiles))
	for i, e := range toolIconFiles {
		out[i] = e.icon
	}
	return out
}

// ToolIconName is the action id used for PNG files (open, new, cut, …).
func ToolIconName(icon ToolIcon) string {
	for _, e := range toolIconFiles {
		if e.icon == icon {
			return e.name
		}
	}
	return ""
}

// ToolIconFileName is the 24×24 PNG for icon (open.png). Empty for IconNone.
func ToolIconFileName(icon ToolIcon) string {
	if name := ToolIconName(icon); name != "" {
		return name + ".png"
	}
	return ""
}

// ToolIconHiDPIFileName is the 48×48 PNG (open@2x.png). Empty for IconNone.
func ToolIconHiDPIFileName(icon ToolIcon) string {
	if name := ToolIconName(icon); name != "" {
		return name + "@2x.png"
	}
	return ""
}

// iconHiDPIMin is the destination width (user-space px) at which @2x is
// preferred. Toolbar 1× is 24; message icons and 2× chrome are ≥36.
const iconHiDPIMin = 36

func toolIconFileCandidates(icon ToolIcon, destW float32) []string {
	stems := []string{ToolIconName(icon)}
	if icon == IconMail {
		stems = append(stems, "inbox", "mail-open")
	}
	var out []string
	for _, name := range stems {
		if name == "" {
			continue
		}
		lo, hi := name+".png", name+"@2x.png"
		if destW >= iconHiDPIMin {
			out = append(out, hi, lo)
		} else {
			out = append(out, lo, hi)
		}
	}
	return out
}

// ToolIconThemeName is a freedesktop icon-theme name for tray / SNI
// (IconName). File sets still use ToolIconName ("mail.png").
func ToolIconThemeName(icon ToolIcon) string {
	if icon == IconMail {
		return "mail-unread"
	}
	if name := ToolIconName(icon); name != "" {
		return name
	}
	return "application-default-icon"
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
