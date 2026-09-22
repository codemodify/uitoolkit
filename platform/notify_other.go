//go:build !linux

package platform

// Notifier sends desktop notifications. Outside Linux there is no service
// for it yet (the tray item's toast is the way there): Send reports
// ErrNoNotifications.
type Notifier struct{ opts NotifierOptions }

// NewNotifier makes a notifier.
func NewNotifier(opts NotifierOptions) *Notifier { return &Notifier{opts: opts} }

// Backend is "stub" outside Linux.
func (n *Notifier) Backend() string { return "stub" }

// Send reports ErrNoNotifications outside Linux (a stub notifier sends
// nothing, and says it did).
func (n *Notifier) Send(note DesktopNotification) (string, error) {
	if n != nil && n.opts.Stub {
		return note.ID, nil
	}
	return "", ErrNoNotifications
}

// Withdraw does nothing outside Linux.
func (n *Notifier) Withdraw(id string) error { return nil }

// Close does nothing outside Linux.
func (n *Notifier) Close() {}
