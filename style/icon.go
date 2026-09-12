package style

// ToolIcon is a stock glyph drawn by LookAndFeel (no bitmap assets).
// Classic and Sharp icon sets share this enum; see DrawToolIcon / IconSetName.
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
