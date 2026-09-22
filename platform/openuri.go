package platform

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

// OpenURIOptions configure [OpenURI].
type OpenURIOptions struct {
	// ParentWindow is the window asking, as the portal names it
	// ([PortalParenter]: "x11:<id>", "wayland:<handle>"): a chooser the
	// desktop shows is then its child. Empty: none.
	ParentWindow string
	// Ask lets the user pick the application instead of opening the
	// default one (the portal's "Open with…").
	Ask bool
	// ActivationToken is an xdg-activation token for the application
	// that opens, so that the compositor lets it take the focus.
	ActivationToken string
}

// ErrNoOpener: neither the OpenURI portal nor xdg-open could open it.
var ErrNoOpener = errors.New("platform: nothing to open the link with")

// EnvOpenURI set to "xdg-open" skips the portal and opens links with
// xdg-open; "off" opens nothing (tests, kiosks).
const EnvOpenURI = "UITK_OPENURI"

// localPath says whether uri names a local file — a path, or a file:// URI
// — and which.
func localPath(uri string) (string, bool) {
	if strings.HasPrefix(uri, "/") {
		return filepath.Clean(uri), true
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") {
		return "", false
	}
	return u.Path, u.Path != ""
}

// validOpenURI refuses what must never be handed to an opener: nothing at
// all, and anything a command line would read as an option.
func validOpenURI(uri string) bool {
	uri = strings.TrimSpace(uri)
	return uri != "" && !strings.HasPrefix(uri, "-")
}
