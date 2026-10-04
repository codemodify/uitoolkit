package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A skinned player has one primary window and two satellite panels — an
// equaliser and a playlist — that are dragged about on their own, carry a close
// control and no minimize control, stay with the window they belong to, and get
// no entry of their own in the desktop's window list.
//
// Only the last two needed anything new. A role said what kind of window this
// was and nothing said *whose*, and nothing at all said "leave this out of the
// window list" — so an application had to reach past the toolkit to its X11
// window and set WM_TRANSIENT_FOR and the skip states itself, re-applying them
// whenever the native window changed underneath it.

func panelRig(t *testing.T) (*Application, *Window, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	main, err := a.NewWindow(platform.WindowOptions{Title: "Player", Width: 275, Height: 116, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(main.Close)
	main.SetContent(widgets.NewColumn(widgets.NewLabel("player")))
	panel, err := a.NewWindow(platform.WindowOptions{
		Title: "Playlist", Width: 275, Height: 200, Headless: true,
		Role: platform.RoleUtility, Owner: main.Surface(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(panel.Close)
	panel.SetContent(widgets.NewColumn(widgets.NewLabel("playlist")))
	a.PumpOnce()
	return a, main, panel
}

// A panel opened as a utility window belongs to its primary window and is kept
// out of the window list, both from the options rather than afterwards: a
// window manager reads the type and the owner when it *maps* the window.
func TestAUtilityPanelIsOwnedAndOutOfTheWindowList(t *testing.T) {
	_, main, panel := panelRig(t)
	o := panel.Surface().(*platform.Offscreen)

	if got := o.WindowRole(); got != platform.RoleUtility {
		t.Errorf("the panel's role is %v, want utility", got)
	}
	if got := o.Owner(); got != main.Surface() {
		t.Errorf("the panel's owner is %v, want the primary window", got)
	}
	if !o.SkipsTaskbar() {
		t.Error("the panel asked for a place in the window list")
	}
	// And the primary window is not a panel.
	m := main.Surface().(*platform.Offscreen)
	if m.Owner() != nil || m.SkipsTaskbar() || m.WindowRole() != platform.RoleNormal {
		t.Errorf("the primary window is owned=%v skipping=%v role=%v",
			m.Owner(), m.SkipsTaskbar(), m.WindowRole())
	}
}

// The role implies the taskbar policy, so a window that stops being a panel is
// listed again — unless the application asked for it on its own account, which
// outlives the role.
func TestTheTaskbarPolicyFollowsTheRoleUnlessItWasAskedFor(t *testing.T) {
	a, _, panel := panelRig(t)
	o := panel.Surface().(*platform.Offscreen)

	if !panel.SetWindowRole(platform.RoleNormal) {
		t.Fatal("the panel could not become an ordinary window")
	}
	a.PumpOnce()
	if o.SkipsTaskbar() {
		t.Error("a window that is no longer a panel is still out of the window list")
	}

	// Asked for in its own right now.
	if !panel.SetSkipTaskbar(true) {
		t.Fatal("the window could not be kept out of the list")
	}
	if !panel.SetWindowRole(platform.RoleUtility) || !panel.SetWindowRole(platform.RoleNormal) {
		t.Fatal("the role could not be changed")
	}
	if !o.SkipsTaskbar() {
		t.Error("a change of role undid what the application had asked for")
	}
}

// An owner can be given and taken away afterwards, and the two owners no
// backend accepts are refused: the window itself, and nothing at all is not a
// refusal but a release.
func TestAnOwnerCanBeGivenAndTakenAway(t *testing.T) {
	_, main, panel := panelRig(t)
	o := panel.Surface().(*platform.Offscreen)

	if panel.SetOwner(panel) {
		t.Error("a window was allowed to own itself")
	}
	if !panel.SetOwner(nil) {
		t.Error("an owner could not be given up")
	}
	if o.Owner() != nil {
		t.Errorf("the panel still belongs to %v", o.Owner())
	}
	if !panel.SetOwner(main) {
		t.Error("an owner could not be given")
	}
	if o.Owner() != main.Surface() {
		t.Error("the owner did not take")
	}
}

// A desktop that cannot do either says so, and the call answers false rather
// than reporting success for something that did not happen. This is Wayland's
// answer about the window list, and it is the whole reason the capability
// exists: an application has to be able to tell the user.
func TestADesktopThatCannotSaysSo(t *testing.T) {
	_, main, panel := panelRig(t)
	o := panel.Surface().(*platform.Offscreen)

	if !panel.FrameCaps().Has(platform.FrameSkipTaskbar) {
		t.Fatal("the simulated desktop should start able to do this")
	}
	o.SimulateSkipTaskbar(false)
	if panel.FrameCaps().Has(platform.FrameSkipTaskbar) {
		t.Error("the capability is still claimed")
	}
	if panel.SetSkipTaskbar(true) {
		t.Error("a desktop that cannot keep a window out of the list reported that it had")
	}

	o.SimulateOwner(false)
	if panel.FrameCaps().Has(platform.FrameOwner) {
		t.Error("the owner capability is still claimed")
	}
	if panel.SetOwner(main) {
		t.Error("a desktop that cannot own windows reported that it had")
	}
}

// The role's own name, which an application puts in a log or a diagnostic.
func TestRoleNames(t *testing.T) {
	for r, want := range map[platform.WindowRole]string{
		platform.RoleNormal:  "normal",
		platform.RoleDialog:  "dialog",
		platform.RoleUtility: "utility",
	} {
		if got := r.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", r, got, want)
		}
		if got := r.SkipsTaskbar(); got != (r == platform.RoleUtility) {
			t.Errorf("%s.SkipsTaskbar() = %v", want, got)
		}
	}
}
