//go:build !linux

package app

// startA11y: no accessibility adapter outside Linux yet.
func (a *Application) startA11y() {}
