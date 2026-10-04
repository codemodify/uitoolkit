package platform

// taskbarPolicy is the one rule every backend that can keep a window out of
// the desktop's window list has to follow, written once.
//
// [RoleUtility] asks to be left out by its nature, so a window that becomes a
// satellite panel leaves the list and one that stops being a panel comes back
// — unless the application asked to be left out *in its own right*, which
// outlives any change of role. Without that distinction a window the
// application had deliberately kept out of the list would reappear in it the
// moment its role changed for some other reason, and the application would
// have no way to tell.
//
// It is here rather than in each backend because three of them have the rule
// and the fourth would have grown its own version of it: the states are
// _NET_WM_STATE_SKIP_TASKBAR on X11 and WS_EX_TOOLWINDOW on Win32, but the
// policy about *when* to set them is the same everywhere, and a policy written
// three times is a policy that drifts.
type taskbarPolicy struct {
	skip  bool
	asked bool
}

// init is the policy a window opens with.
func (p *taskbarPolicy) init(opts WindowOptions) {
	p.skip = opts.SkipTaskbar || opts.Role.SkipsTaskbar()
	p.asked = opts.SkipTaskbar
}

// ask records a request the application made in its own right, which from
// now on survives any change of role.
func (p *taskbarPolicy) ask(skip bool) { p.skip, p.asked = skip, true }

// roleChanged folds a change of role into the policy and reports whether the
// window's place in the list changed with it.
func (p *taskbarPolicy) roleChanged(was, now WindowRole) bool {
	if p.asked || was.SkipsTaskbar() == now.SkipsTaskbar() {
		return false
	}
	p.skip = now.SkipsTaskbar()
	return true
}

// skipping is whether the window asks to be left out of the list.
func (p *taskbarPolicy) skipping() bool { return p.skip }
