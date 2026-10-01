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
	// The marks a list view puts beside a row, added when a mail client
	// needed them and found that a cell could only hold characters: a
	// paperclip, a star, a flag, reply and forward, a check and a muted
	// bell. Every one of them resolves to a PNG the five shipped icon
	// sets already carry ([ShippedIconStems]), so they are icons in the
	// full sense rather than drawn-only stand-ins.
	IconAttach
	IconStar
	IconFlag
	IconReply
	IconForward
	IconCheck
	IconMute
	// The actions a mail client's menus and toolbars are made of. Every
	// one of these was already a PNG in all five shipped sets with no id
	// to name it by, so an application could see the art and not reach
	// it; IconPrint is the one that had to be drawn, since no set carried
	// a printer.
	IconTrash
	IconArchive
	IconJunk
	IconTag
	IconFolder
	IconReplyAll
	IconSettings
	IconExternalLink
	IconEye
	IconUser
	IconBell
	IconSend
	IconClose
	IconQuit
	IconPrint
	// A filled star and a dot: the two marks a list uses to say "this one
	// is flagged" and "this one is unread". They are pure geometry with
	// no house style to match, so the toolkit draws them itself in every
	// set — a filled star is a filled star in all five packs, and none of
	// them ships one.
	IconStarFilled
	IconDot
	// The two marks a button that opens a menu wears. Every desktop
	// draws an overflow as ⋯ and an application menu as a hamburger, and
	// both shipped only as stems — so IconByStem gave them back with no
	// vector, and in a drawn set (which every pack uses unless the user
	// picks otherwise) they came out as the missing-icon mark.
	IconMore
	IconMenu
	// The layout family, which an application shell's "toggle the panel /
	// the sidebar / the second editor" buttons are made of. They shipped
	// as stems with no typed id, so in a drawn set — which every pack
	// uses unless the user picks otherwise — they were the missing-icon
	// mark.
	IconLayout
	IconColumns
	IconRows
	IconTable
	IconCards
	// The four arrows. Back and forward are the chrome of a browser, a
	// wizard and a file manager alike, and they were stem-only too.
	IconArrowLeft
	IconArrowRight
	IconArrowUp
	IconArrowDown
	// A tray, which is what an inbox is drawn as everywhere.
	IconInbox
	// A plus and a cross, which every tab strip and every list with an
	// "add" needs.
	IconPlus
	IconClose2
	// A padlock, for a secure address and a locked vault.
	IconLock
	// A circular arrow. Reload is not redo: a browser's reload button is
	// a ring, and IconRedo is the right-angled arrow that undoes an undo.
	IconSync
)

var toolIconLabels = [...]string{
	IconNew: "New", IconOpen: "Open", IconSave: "Save", IconCut: "Cut",
	IconCopy: "Copy", IconPaste: "Paste", IconUndo: "Undo", IconRedo: "Redo",
	IconSearch: "Search", IconInfo: "Information", IconWarning: "Warning",
	IconError: "Error", IconQuestion: "Question", IconMail: "Mail",
	IconDownload: "Download", IconPen: "Edit",
	IconAttach: "Attachment", IconStar: "Star", IconFlag: "Flag",
	IconReply: "Reply", IconForward: "Forward", IconCheck: "Done",
	IconMute:  "Muted",
	IconTrash: "Delete", IconArchive: "Archive", IconJunk: "Junk",
	IconTag: "Tag", IconFolder: "Folder", IconReplyAll: "Reply All",
	IconSettings: "Settings", IconExternalLink: "Open in browser",
	IconEye: "Show", IconUser: "Contact", IconBell: "Notify",
	IconSend: "Send", IconClose: "Close", IconQuit: "Quit",
	IconPrint: "Print", IconStarFilled: "Starred", IconDot: "Unread",
	IconMore: "More", IconMenu: "Menu",
	IconLayout: "Layout", IconColumns: "Columns", IconRows: "Rows",
	IconTable: "Table", IconCards: "Cards",
	IconArrowLeft: "Back", IconArrowRight: "Forward",
	IconArrowUp: "Up", IconArrowDown: "Down",
	IconInbox: "Inbox", IconPlus: "Add", IconClose2: "Close", IconLock: "Secure",
	IconSync: "Reload",
}

// Label is the action the icon stands for, in words ("Save"): what
// assistive technology says for a tool button that shows only the icon.
func (i ToolIcon) Label() string {
	if i > IconNone && int(i) < len(toolIconLabels) {
		return toolIconLabels[i]
	}
	return ""
}

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
	{IconAttach, "attach"},
	{IconStar, "star"},
	{IconFlag, "flag"},
	{IconReply, "reply"},
	{IconForward, "forward"},
	{IconCheck, "check"},
	{IconMute, "bell-off"},
	{IconTrash, "trash"},
	{IconArchive, "archive"},
	{IconJunk, "junk"},
	{IconTag, "tag"},
	{IconFolder, "folder"},
	{IconReplyAll, "reply-all"},
	{IconSettings, "settings"},
	{IconExternalLink, "external-link"},
	{IconEye, "eye"},
	{IconUser, "user"},
	{IconBell, "bell"},
	{IconSend, "send"},
	{IconClose, "close"},
	{IconQuit, "quit"},
	{IconPrint, "print"},
	{IconStarFilled, "star-filled"},
	{IconDot, "dot"},
	{IconMore, "more"},
	{IconMenu, "menu"},
	{IconLayout, "layout"},
	{IconColumns, "columns"},
	{IconRows, "rows"},
	{IconTable, "table"},
	{IconCards, "cards"},
	{IconArrowLeft, "arrow-left"},
	{IconArrowRight, "arrow-right"},
	{IconArrowUp, "arrow-up"},
	{IconArrowDown, "arrow-down"},
	{IconInbox, "inbox"},
	{IconPlus, "plus"},
	{IconClose2, "x"},
	{IconLock, "lock"},
	{IconSync, "sync"},
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
	"sun", "moon", "print", "menu",
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

// iconNative1x is the native width of a 1× ToolIcon PNG. Anything larger
// than that upscales the small asset, so @2x (48×48) is preferred as soon as
// the destination exceeds it — "large" chrome icons are 32 user-space px.
const iconNative1x = 24

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
	case IconAttach:
		return []string{"paperclip"}
	case IconMute:
		return []string{"bell"}
	default:
		return nil
	}
}

func toolIconStems(icon ToolIcon) []string {
	name := StemOf(icon)
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
	if destW > iconNative1x {
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

// toolIconThemeNames maps each chrome action onto the names an installed
// freedesktop icon theme keeps it under, best first.
//
// They are the Icon Naming Specification's names and not this toolkit's
// own: a theme has "document-save", not "save", and asking it for "save"
// is asking for a file that no theme on earth ships. That is what this
// table is for, and it is why an installed theme can be an icon set at
// all (see icontheme.go) — and why the tray, which has always claimed to
// send a themed name, was in fact sending "new", "open" and "cut" to a
// status-notifier host that could not resolve one of them.
//
// A second and third name where the sets disagree: KDE's "edit-find" is
// GNOME's "system-search", Breeze's "document-edit" is Adwaita's
// "text-editor", and the download arrow is "browser-download" in one set
// and "go-down" in the plainest of the old ones.
var toolIconThemeNames = map[ToolIcon][]string{
	IconNew:      {"document-new"},
	IconOpen:     {"document-open", "folder-open"},
	IconSave:     {"document-save", "media-floppy"},
	IconCut:      {"edit-cut"},
	IconCopy:     {"edit-copy"},
	IconPaste:    {"edit-paste"},
	IconUndo:     {"edit-undo"},
	IconRedo:     {"edit-redo"},
	IconSearch:   {"edit-find", "system-search", "search"},
	IconInfo:     {"dialog-information", "gtk-dialog-info"},
	IconWarning:  {"dialog-warning", "gtk-dialog-warning"},
	IconError:    {"dialog-error", "gtk-dialog-error"},
	IconQuestion: {"dialog-question", "help-contents", "gtk-dialog-question"},
	IconMail:     {"mail-unread", "mail-message", "internet-mail"},
	IconDownload: {"browser-download", "document-save", "go-down"},
	IconPen:      {"document-edit", "gtk-edit", "text-editor", "accessories-text-editor"},
	IconAttach:   {"mail-attachment", "stock_attach"},
	IconStar:     {"starred", "rating", "bookmark-new"},
	IconFlag:     {"flag", "mail-mark-important", "emblem-important"},
	IconReply:    {"mail-reply-sender", "mail-replied"},
	IconForward:  {"mail-forward", "mail-forwarded"},
	IconCheck:    {"object-select", "emblem-ok", "gtk-apply"},
	IconMute:     {"audio-volume-muted", "notification-disabled"},

	IconTrash:        {"user-trash", "edit-delete", "delete"},
	IconArchive:      {"mail-archive", "package-x-generic", "archive-insert"},
	IconJunk:         {"mail-mark-junk", "dialog-error", "action-unavailable"},
	IconTag:          {"tag", "bookmark-new", "stock_bookmark"},
	IconFolder:       {"folder", "inode-directory"},
	IconReplyAll:     {"mail-reply-all", "mail-replied-all"},
	IconSettings:     {"preferences-system", "configure", "gtk-preferences"},
	IconExternalLink: {"link", "emblem-symbolic-link", "web-browser"},
	IconEye:          {"view-visible", "image-x-generic", "view-preview"},
	IconUser:         {"avatar-default", "user-identity", "stock_person"},
	IconBell:         {"preferences-desktop-notification", "notification-active"},
	IconSend:         {"mail-send", "document-send", "go-next"},
	IconClose:        {"window-close", "dialog-close", "gtk-close"},
	IconQuit:         {"application-exit", "system-log-out", "gtk-quit"},
	IconPrint:        {"document-print", "printer", "gtk-print"},
	IconStarFilled:   {"starred", "rating", "bookmark-new"},
	IconDot:          {"media-record", "dialog-information"},
	IconMore:         {"view-more-symbolic", "overflow-menu", "go-down"},
	IconMenu:         {"open-menu", "application-menu", "format-justify-fill"},
	IconLayout:       {"view-dual", "view-grid", "preferences-desktop"},
	IconColumns:      {"view-split-left-right", "view-column", "view-dual"},
	IconRows:         {"view-split-top-bottom", "view-list", "view-continuous"},
	IconTable:        {"view-grid", "x-office-spreadsheet", "table"},
	IconCards:        {"view-grid", "view-paged", "view-list-icons"},
	IconArrowLeft:    {"go-previous", "back", "draw-arrow-back"},
	IconArrowRight:   {"go-next", "forward", "draw-arrow-forward"},
	IconArrowUp:      {"go-up", "up"},
	IconArrowDown:    {"go-down", "down"},
	IconInbox:        {"mail-inbox", "mail-folder-inbox", "inbox"},
	IconPlus:         {"list-add", "add"},
	IconClose2:       {"window-close", "edit-delete", "list-remove"},
	IconLock:         {"channel-secure", "security-high", "lock"},
	IconSync:         {"view-refresh", "reload", "system-software-update"},
}

// ToolIconThemeNames are the freedesktop names an installed icon theme
// is asked for, best first. It is what the system icon sets look an
// action up by; [ToolIconThemeName] is the first of them.
func ToolIconThemeNames(icon ToolIcon) []string {
	if names, ok := toolIconThemeNames[icon]; ok {
		return names
	}
	return nil
}

// ToolIconThemeName is a freedesktop icon-theme name for tray / SNI
// (IconName). File sets still use ToolIconName ("mail.png").
func ToolIconThemeName(icon ToolIcon) string {
	if names := ToolIconThemeNames(icon); len(names) > 0 {
		return names[0]
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

// IconByStem is the icon for any stem the shipped sets carry, whether or
// not it has a typed id of its own.
//
// The typed ids are the actions the toolkit itself draws — the ones a
// drawn icon set has a vector for, so they work in every set including
// the two that are drawn rather than loaded. The shipped PNG vocabulary
// is wider than that ([ShippedIconStems]), and it was unreachable: an
// application could see "calendar", "clock", "users" and "sync" sitting
// in all five premiere packs with no way to name one.
//
// A stem with a typed id gives that id, so it behaves identically in
// every set. A stem without one gives an icon that resolves from a file
// set and falls back to the missing-icon mark in a drawn set — which is
// the honest answer, since the toolkit has no vector for it. ok is false
// for a stem nothing ships, so a caller can fall back rather than draw a
// placeholder.
func IconByStem(stem string) (ToolIcon, bool) {
	if stem == "" {
		return IconNone, false
	}
	if icon, ok := ToolIconByName(stem); ok {
		return icon, true
	}
	for i, s := range shippedIconStems {
		if s == stem {
			return stemIconBase + ToolIcon(i), true
		}
	}
	return IconNone, false
}

// stemIconBase is where the stem-only icons start, above every typed id.
// They are ToolIcon values so they travel through the same painting and
// caching as the rest; nothing switches on them, because the drawn sets
// have no vector for them and the file sets go by name.
const stemIconBase ToolIcon = 1 << 12

// Drawable reports whether the toolkit can draw this icon itself, which
// is true of every typed id and of no stem-only one.
//
// It is the difference between the two ways a file set can fail to have
// an icon. A set that is missing a *typed* stem is a set that has fallen
// behind — the toolkit gained an id after the person copied the set into
// their directory, which happens to everybody every time an id is added
// — and the toolkit has its own vector to draw meanwhile. A set missing
// a *stem-only* icon has nothing behind it anywhere, and the
// missing-icon mark is the only honest answer.
func Drawable(icon ToolIcon) bool { return icon != IconNone && icon < stemIconBase }

// StemOf is the file stem an icon resolves to, including the stem-only
// ones from [IconByStem]. It is "" for IconNone.
func StemOf(icon ToolIcon) string {
	if icon >= stemIconBase {
		if i := int(icon - stemIconBase); i >= 0 && i < len(shippedIconStems) {
			return shippedIconStems[i]
		}
		return ""
	}
	return ToolIconName(icon)
}
