package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// An application that wants no window-menu button on any of its windows had
// nowhere to say so.
//
// The preference is the user's — look.json's "hideWindowMenu" — and
// ApplyAppearance re-applies it from the file on every change Settings makes,
// so a call to SetHideWindowMenu was undone by the next one. The per-window
// SetCaptionButtonVisible has to be repeated for every window the application
// opens and cannot reach the ones the toolkit opens for it. A mail client
// worked around it by setting the flag again from OnLookChange after every
// look, which works only because ApplyAppearance happens to end in SetLook.

func TestAnApplicationsWindowMenuChoiceSurvivesTheUsersFile(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	a.SetHideWindowMenu(true)
	if !a.HideWindowMenu() {
		t.Fatal("the application's choice did not take")
	}
	if !a.HideWindowMenuPinned() {
		t.Error("the choice is not reported as the application's")
	}

	// Everything Settings does ends in one of these.
	ap := a.Appearance()
	ap.HideWindowMenu = false
	a.ApplyAppearance(ap)
	if !a.HideWindowMenu() {
		t.Error("the user's file overwrote the application's choice")
	}

	// And again, because the mail client's symptom was that it came back
	// on the *second* apply.
	a.ApplyAppearance(ap)
	if !a.HideWindowMenu() {
		t.Error("a second apply overwrote the application's choice")
	}
}

// The other way round too: an application that wants the button kept is not
// overruled by a file that hides it.
func TestAnApplicationCanKeepTheWindowMenuButton(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	a.SetHideWindowMenu(false)
	ap := a.Appearance()
	ap.HideWindowMenu = true
	a.ApplyAppearance(ap)
	if a.HideWindowMenu() {
		t.Error("the user's file hid a button the application asked to keep")
	}
}

// An application that says nothing follows the user, which is the default and
// must stay that way.
func TestWithoutAChoiceTheUsersFileStillRules(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	if a.HideWindowMenuPinned() {
		t.Fatal("a fresh application claims to have chosen")
	}
	ap := a.Appearance()
	ap.HideWindowMenu = true
	a.ApplyAppearance(ap)
	if !a.HideWindowMenu() {
		t.Error("the user's file did not reach an application that pinned nothing")
	}
	ap.HideWindowMenu = false
	a.ApplyAppearance(ap)
	if a.HideWindowMenu() {
		t.Error("the user's file could not put the button back")
	}
}
