package style

import "testing"

// restoreMotion puts the process-wide motion state back after a test.
func restoreMotion(t *testing.T) {
	t.Helper()
	user, desktop, follow := ReduceMotion(), DesktopReducesMotion(), FollowsDesktopMotion()
	t.Cleanup(func() {
		SetReduceMotion(user)
		SetDesktopReduceMotion(desktop)
		SetFollowDesktopMotion(follow)
	})
}

// **The toolkit decides its own motion.** The desktop's reduced-motion
// setting is recorded but not obeyed, unless the application asks for it:
// a Plasma session with its animations switched off used to turn off
// every animation in every uitoolkit app with nothing able to say
// otherwise, including an app written to demonstrate one.
func TestMotionIsTheToolkitsOwnCall(t *testing.T) {
	restoreMotion(t)
	for _, tc := range []struct {
		name                  string
		env                   string
		user, desktop, follow bool
		want                  bool
	}{
		{"nobody minds", "", false, false, false, true},

		// The headline: the desktop asked for less and is ignored.
		{"the desktop asked for less", "", false, true, false, true},
		{"...and is obeyed where the app asked for that", "", false, true, true, false},
		{"...and following with nothing to follow changes nothing", "", false, false, true, true},

		// The application's own preference is always obeyed: it is the
		// app's, not the session's.
		{"the app turned motion off", "", true, false, false, false},
		{"the app turned it off while following", "", true, true, true, false},

		// The variable is over everything, both ways.
		{"0 with nobody minding", "0", false, false, false, false},
		{"0 over everything", "0", true, true, true, false},
		{"1 over the app's own preference", "1", true, false, false, true},
		{"1 over a followed desktop", "1", false, true, true, true},
		{"1 with spaces round it", " 1 ", true, true, true, true},

		// Anything else is not an instruction, so the rest stands.
		{"a word is not an instruction", "yes", true, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(AnimationsEnv, tc.env)
			SetReduceMotion(tc.user)
			SetDesktopReduceMotion(tc.desktop)
			SetFollowDesktopMotion(tc.follow)
			if got := Animations(); got != tc.want {
				t.Errorf("Animations() = %v with %s=%q, app=%v, desktop=%v, following=%v; want %v",
					got, AnimationsEnv, tc.env, tc.user, tc.desktop, tc.follow, tc.want)
			}
		})
	}
}

// The two preferences keep reporting themselves whatever the variable
// says: an application that draws its own motion still has to be able to
// ask what the user wanted, and the override is about this process's
// animations rather than about rewriting the preference.
func TestTheAnimationsVariableDoesNotRewriteThePreferences(t *testing.T) {
	restoreMotion(t)
	t.Setenv(AnimationsEnv, "1")
	SetReduceMotion(true)
	SetDesktopReduceMotion(true)
	if !ReduceMotion() {
		t.Error("the user's reduce-motion preference was lost")
	}
	if !DesktopReducesMotion() {
		t.Error("the desktop's reduced-motion setting was lost")
	}
}
