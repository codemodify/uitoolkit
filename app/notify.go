package app

import (
	"os"
	"strings"

	"github.com/codemodify/uitoolkit/platform"
)

// NewNotifier makes the application's desktop notifier
// (platform.Notifier): a notification's clicks come back on the UI
// goroutine. A headless application sends nothing unless UITK_NOTIFY names
// a backend, so tests and screenshot runs never put a toast on a desktop.
func (a *Application) NewNotifier(opts platform.NotifierOptions) *platform.Notifier {
	if a != nil {
		opts.Dispatch = a.Post
		if a.headless {
			switch strings.ToLower(strings.TrimSpace(os.Getenv(platform.EnvNotify))) {
			case "portal", "fdo":
			default:
				opts.Stub = true
			}
		}
	}
	return platform.NewNotifier(opts)
}

// OpenURI opens uri with the desktop's application for it
// (platform.OpenURI), as this window asks: a chooser the desktop shows is
// the window's child. It returns at once; done, if not nil, hears how it
// went on the UI goroutine.
func (w *Window) OpenURI(uri string, done func(error)) {
	opts := platform.OpenURIOptions{ParentWindow: w.PortalParent()}
	go func() {
		err := platform.OpenURI(uri, opts)
		if done != nil && w.app != nil {
			w.app.Post(func() { done(err) })
		}
	}()
}
