package platform

// FileChooserOptions configure the desktop's own file dialog.
type FileChooserOptions struct {
	Title       string
	AcceptLabel string // the default button's label ("Attach")
	Save        bool
	Name        string // the suggested file name when saving
	Folder      string // where to start
	Multiple    bool
	Directory   bool // pick a folder
	Filters     []FileFilter
	// ParentWindow is the portal's parent id ("x11:1a00003",
	// "wayland:<exported handle>"); empty: no parent.
	ParentWindow string
}

// PortalParenter is implemented by surfaces that can name themselves as a
// portal dialog's parent (X11 windows; Wayland needs xdg-foreign).
type PortalParenter interface {
	PortalParent() string
}

// FileFilter is one entry of a dialog's type list ("Images", "*.png").
type FileFilter struct {
	Name     string
	Patterns []string
}
