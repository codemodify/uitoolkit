package style

// ToolIcon is a stock glyph drawn by LookAndFeel.
// File icon sets use one PNG per id (new.png, open.png, …) under
// ~/.config/uitoolkit/icons/<set>/. Classic / Sharp remain drawn fallbacks
// only when no file set is installed. HiDPI uses name@2x.png (48×48) when
// the destination is large enough.
//
// Premiere packs ship a wide stem vocabulary beyond these typed ids
// (see ShippedIconStems / icons/STEMS.txt). Fetch / Write already use
// IconDownload / IconPen — new enum values are added only when Mail or
// Settings need a typed chrome action.
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
	IconDownload
	IconPen
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
	{IconDownload, "download"},
	{IconPen, "pen"},
}

// shippedIconStems is the wide PNG vocabulary rendered by icons/render.sh
// into every premiere set. Keep in sync with icons/STEMS.txt.
var shippedIconStems = []string{
	"new", "open", "save", "cut", "copy", "paste", "undo", "redo",
	"search", "filter", "settings", "preferences", "quit", "close", "more",
	"chevron-up", "chevron-down", "chevron-left", "chevron-right",
	"arrow-up", "arrow-down", "arrow-left", "arrow-right",
	"mail", "mail-open", "inbox", "send", "reply", "reply-all", "forward",
	"attach", "download", "upload", "trash", "archive", "junk", "tag", "star", "flag",
	"pen", "pencil", "compose", "fetch", "sync",
	"folder", "folder-plus", "user", "users",
	"bell", "bell-off", "eye", "eye-off", "lock", "unlock",
	"check", "x", "plus", "minus", "edit", "trash-2",
	"info", "warning", "error", "question", "help",
	"home", "calendar", "clock", "link", "external-link",
	"list", "layout", "columns", "rows", "table", "cards",
	"sun", "moon",
	"no-icon",
}

// ShippedIconStems is every PNG stem committed in the five premiere packs.
func ShippedIconStems() []string {
	out := make([]string, len(shippedIconStems))
	copy(out, shippedIconStems)
	return out
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

func toolIconAliases(icon ToolIcon) []string {
	switch icon {
	case IconQuestion:
		return []string{"help"}
	case IconMail:
		return []string{"inbox", "mail-open"}
	case IconDownload:
		return []string{"fetch"}
	case IconPen:
		return []string{"pencil", "compose"}
	default:
		return nil
	}
}

func toolIconStems(icon ToolIcon) []string {
	name := ToolIconName(icon)
	stems := make([]string, 0, 4)
	if name != "" {
		stems = append(stems, name)
	}
	for _, a := range toolIconAliases(icon) {
		if a == "" || a == name {
			continue
		}
		stems = append(stems, a)
	}
	return stems
}

func stemFileCandidates(name string, destW float32) []string {
	if name == "" {
		return nil
	}
	lo, hi := name+".png", name+"@2x.png"
	if destW >= iconHiDPIMin {
		return []string{hi, lo}
	}
	return []string{lo, hi}
}

func toolIconFileCandidates(icon ToolIcon, destW float32) []string {
	var out []string
	for _, name := range toolIconStems(icon) {
		out = append(out, stemFileCandidates(name, destW)...)
	}
	return out
}

const noIconStem = "no-icon"

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
