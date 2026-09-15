package app

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// Options configure an Application.
type Options struct {
	Look     style.LookAndFeel
	Scale    float32
	Headless bool
	// Backend is "x11", "wayland", "offscreen", or "" for auto
	// (WAYLAND_DISPLAY → DISPLAY → offscreen). Headless always
	// uses offscreen. Unavailable names fall through.
	Backend string
	// WatchLook reloads $XDG_CONFIG_HOME/uitoolkit/look.json when the
	// file changes and applies it with SetLook(WithAppearance(...)).
	// New turns this on automatically when Look is nil (PreferredLook).
	// Set true when passing PreferredLook() explicitly so Mail / gallery
	// pick up Settings → Apply without a restart. Tests that pass
	// DarkLook / LightLook stay static unless this is set.
	WatchLook bool
	// DisableLookWatch skips the default watcher (Look == nil). Settings
	// uses this so picker changes preview locally until Apply writes.
	DisableLookWatch bool
}

// Application owns the run loop and open windows.
type Application struct {
	mu    sync.Mutex
	look  style.LookAndFeel
	base  style.LookAndFeel
	scale float32
	// autoScale is set when no explicit Options.Scale / env override was
	// given, so each Window may take its display scale from its own
	// surface (per-monitor DPI) instead of the process-wide guess.
	autoScale        bool
	headless         bool
	backend          platform.Backend
	windows          []*Window
	quit             atomic.Bool
	onQuit           func()
	watchLook        bool
	lookWatch        *lookFileStamp
	trays            []platform.StatusItem
	looping          bool
	posted           []func()
	statusMenu       *Window
	hidingStatusMenu bool
	// desktopStop ends the watch on the desktop's appearance preferences;
	// schemeForced is set when ColorSchemeEnv stands in for the desktop;
	// following is set while the look follows its light / dark preference.
	desktopStop  func()
	schemeForced bool
	accentForced bool
	following    bool
	lookHooks    []*func()
}

// New constructs an application. Default look is PreferredLook
// (XDG appearance, else dark Classic). When Look is nil, New also
// watches look.json so Settings → Apply updates running windows.
// Scale <= 0 means detect (env, then Xft.dpi on X11). Headless
// stays 1× unless UITK_SCALE / GDK_SCALE / QT_SCALE_FACTOR is set.
func New(opts Options) *Application {
	// The installed-font index loads while the display comes up.
	style.PrefetchSystemFonts()
	watch := opts.WatchLook
	preferred := opts.Look == nil
	var ap style.Appearance
	if preferred {
		ap = style.LoadAppearance()
		style.SetReduceMotion(ap.ReduceMotion)
		watch = true
	}
	if opts.DisableLookWatch {
		watch = false
	}
	var backend platform.Backend
	if opts.Backend != "" {
		backend = platform.Select(opts.Backend, opts.Headless)
	} else {
		backend = platform.Default(opts.Headless)
	}
	auto := false
	if opts.Scale <= 0 {
		if env := platform.ScaleFromEnv(); env > 0 {
			opts.Scale = env
		} else if opts.Headless {
			opts.Scale = 1
			auto = true
		} else {
			opts.Scale = platform.DetectScale()
			auto = true
		}
	}
	a := &Application{
		scale:     opts.Scale,
		autoScale: auto,
		headless:  opts.Headless,
		backend:   backend,
		watchLook: watch,
	}
	// Ask the desktop for its preferences before the look is built: a
	// theme that follows its light / dark mode starts in the right one.
	a.watchDesktop()()
	if preferred {
		a.following = ap.FollowDesktop
		opts.Look = ap.Look()
	}
	a.base = lookAtScale(opts.Look, 1)
	a.look = lookAtScale(a.base, opts.Scale)
	if watch {
		a.lookWatch = newLookFileStamp()
	}
	return a
}

// Look is the default theme for new windows.
func (a *Application) Look() style.LookAndFeel { return a.look }

// SetLook swaps the theme on the app and every open window. The look is
// kept as an unscaled base; each window rebuilds it at that window's own
// display scale, so a theme toggle never drops (or doubles) HiDPI metrics
// and a window on a 2× monitor keeps 2× metrics.
func (a *Application) SetLook(l style.LookAndFeel) {
	if l == nil {
		return
	}
	a.base = lookAtScale(l, 1)
	a.look = lookAtScale(a.base, a.scale)
	for _, w := range a.Windows() {
		w.applyLook(a.base)
	}
	for _, fn := range append([]*func(){}, a.lookHooks...) {
		(*fn)()
	}
}

// OnLookChange runs fn on the UI goroutine after every change of the app's
// look: a theme applied in Settings, the desktop turning dark. Content that
// caches something drawn in the old look (a theme preview) rebuilds there.
// remove unregisters fn.
func (a *Application) OnLookChange(fn func()) (remove func()) {
	if a == nil || fn == nil {
		return func() {}
	}
	p := &fn
	a.lookHooks = append(a.lookHooks, p)
	return func() {
		for i, q := range a.lookHooks {
			if q == p {
				a.lookHooks = append(a.lookHooks[:i:i], a.lookHooks[i+1:]...)
				return
			}
		}
	}
}

// lookScaleOf is the display scale a look was built at (1 when unknown).
// Classic carries this explicitly; inferring it from FontSize would confuse
// display scale with density.
func lookScaleOf(look style.LookAndFeel) float32 {
	if c, ok := look.(*style.Classic); ok {
		if s := c.Scale(); s > 0 {
			return s
		}
	}
	return 1
}

// lookAtScale rebuilds look at an absolute display scale. style.WithScale
// multiplies onto whatever scale the look already carries, so the ratio is
// what gets applied; metrics are rebuilt from the pack defaults either way.
func lookAtScale(look style.LookAndFeel, scale float32) style.LookAndFeel {
	if look == nil || scale <= 0 {
		return look
	}
	cur := lookScaleOf(look)
	if cur <= 0 {
		cur = 1
	}
	if d := scale / cur; d < 0.999 || d > 1.001 {
		return style.WithScale(look, d)
	}
	return look
}

// Scale is the application display scale. Individual windows may differ
// when the backend reports a per-surface scale (see Window.Scale).
func (a *Application) Scale() float32 { return a.scale }

// windowScale is the display scale for a surface. An explicit Options.Scale
// (or UITK_SCALE / GDK_SCALE / QT_SCALE_FACTOR) pins every window; otherwise
// the surface reports its own output scale so a window on a HiDPI monitor
// gets HiDPI metrics even when another window does not.
func (a *Application) windowScale(surf platform.Surface) float32 {
	if a == nil {
		return 1
	}
	if !a.autoScale || surf == nil {
		return a.scale
	}
	if s := surf.Scale(); s > 0 {
		return s
	}
	return a.scale
}

// BackendName is "x11", "wayland", "offscreen", or a stub.
func (a *Application) BackendName() string { return a.backend.Name() }

// NewWindow opens a surface and attaches a retained widget host.
func (a *Application) NewWindow(opts platform.WindowOptions) (*Window, error) {
	if opts.Headless || a.headless {
		opts.Headless = true
	}
	if opts.Width < 1 {
		opts.Width = 800
	}
	if opts.Height < 1 {
		opts.Height = 560
	}
	surf, err := a.backend.NewSurface(opts)
	if err != nil {
		return nil, err
	}
	w := newWindow(a, surf, opts)
	a.mu.Lock()
	a.windows = append(a.windows, w)
	a.mu.Unlock()
	return w, nil
}

// Windows returns open windows (snapshot).
func (a *Application) Windows() []*Window {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]*Window, len(a.windows))
	copy(out, a.windows)
	return out
}

const caretBlinkPeriod = 530 * time.Millisecond

// Run waits on the display connection and paints only when a window is
// dirty, a caret blinks, a tooltip is due, or (on Wayland) a key repeat
// fires. Idle gallery no longer wakes at 60 Hz.
// Post hands fn to the UI goroutine. It never runs fn on the caller's
// goroutine: the func is queued and the display wait is woken, and the queue
// is drained by Run and PumpOnce before the next pump.
//
// Posting before Run starts is fine — the queue is drained as the loop comes
// up. A func posted when no loop will ever run stays queued; callers outside
// a loop (tests, headless tools) flush it with DrainPosted.
//
// This is the only safe way for a background goroutine (a DBus tray
// callback, a network worker) to touch widgets.
func (a *Application) Post(fn func()) {
	if a == nil || fn == nil {
		return
	}
	a.mu.Lock()
	a.posted = append(a.posted, fn)
	looping := a.looping
	a.mu.Unlock()
	if looping {
		a.wakeUI()
	}
}

// Looping reports whether the run loop is pumping. Background callers use it
// to decide whether posted work will be drained promptly.
func (a *Application) Looping() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.looping
}

// DrainPosted runs everything queued by Post on the calling goroutine. Run
// and PumpOnce do this automatically; it is exported for headless callers
// and tests that never start a loop.
func (a *Application) DrainPosted() { a.runPosted() }

func (a *Application) wakeUI() {
	if a == nil {
		platform.WakeLoop()
		return
	}
	for _, w := range a.Windows() {
		if !w.Closed() {
			platform.WakeSurface(w.surf)
			return
		}
	}
	platform.WakeLoop()
}

func (a *Application) runPosted() {
	if a == nil {
		return
	}
	a.mu.Lock()
	fns := a.posted
	a.posted = nil
	a.mu.Unlock()
	for _, fn := range fns {
		if fn != nil {
			fn()
		}
	}
}

func (a *Application) Run() error {
	if len(a.Windows()) == 0 && !a.trayHolds() {
		return fmt.Errorf("uitoolkit: Run with no windows")
	}
	// Clear a quit left over from a previous Run so an application can be
	// restarted (a tray app that reopens its window after Quit).
	a.quit.Store(false)
	a.mu.Lock()
	a.looping = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.looping = false
		a.mu.Unlock()
	}()
	var nextBlink time.Time
	for !a.quit.Load() {
		// Wayland / X11 / offscreen: poll look.json on every idle wake
		// (waitTimeout ≤ lookWatchInterval when WatchLook is on).
		a.pollLookFile()
		a.runPosted()
		now := time.Now()
		if a.anyCaret() && (nextBlink.IsZero() || !now.Before(nextBlink)) {
			for _, w := range a.Windows() {
				w.toggleBlink()
			}
			nextBlink = now.Add(caretBlinkPeriod)
		}
		for _, w := range a.Windows() {
			// Closed() is true only for a surface the display server
			// destroyed. A user close request arrives as EventClose and
			// is answered by Window.dispatch, which may veto it
			// (close-to-tray, status menus).
			if w.surf.Closed() {
				w.Close()
				continue
			}
			w.pump()
		}
		// One frame per event burst: drain every surface first, then
		// paint. Per-window pump+frame used to present mid-burst.
		for _, w := range a.Windows() {
			if w.Closed() {
				continue
			}
			w.frame()
		}
		a.reap()
		// Count survivors after pump/frame: a window that closed itself
		// while draining its burst must not hold the loop open for
		// another iteration.
		alive := a.aliveWindows()
		if alive == 0 {
			if a.trayHolds() {
				timeout := a.waitTimeout(time.Now(), nextBlink)
				if timeout < 0 || timeout > 100*time.Millisecond {
					timeout = 100 * time.Millisecond
				}
				a.waitDisplay(timeout)
				continue
			}
			a.quit.Store(true)
			break
		}
		if a.quit.Load() {
			break
		}
		timeout := a.waitTimeout(time.Now(), nextBlink)
		if a.anyNeedsPaint() {
			timeout = 0
		}
		if a.trayHolds() && (timeout < 0 || timeout > 100*time.Millisecond) {
			timeout = 100 * time.Millisecond
		}
		a.waitDisplay(timeout)
	}
	if a.onQuit != nil {
		a.onQuit()
	}
	return nil
}

// aliveWindows counts windows that can still hold the run loop open. A
// hidden status-menu window is chrome the toolkit owns, not an application
// window: it must not keep Run alive once the real windows are gone.
func (a *Application) aliveWindows() int {
	n := 0
	for _, w := range a.Windows() {
		if w.Closed() || w.holdsNothing() {
			continue
		}
		n++
	}
	return n
}

func (a *Application) anyCaret() bool {
	for _, w := range a.Windows() {
		if w.wantsBlink() {
			return true
		}
	}
	return false
}

func (a *Application) anyNeedsPaint() bool {
	for _, w := range a.Windows() {
		if w.needsPaint() {
			return true
		}
	}
	return false
}

func (a *Application) waitTimeout(now, nextBlink time.Time) time.Duration {
	var deadline time.Time
	if a.anyCaret() {
		if nextBlink.IsZero() {
			deadline = now.Add(caretBlinkPeriod)
		} else {
			deadline = nextBlink
		}
	}
	for _, w := range a.Windows() {
		if w.Closed() {
			continue
		}
		if d, ok := w.tipDeadline(now); ok {
			if deadline.IsZero() || d.Before(deadline) {
				deadline = d
			}
		}
		if t := platform.SurfaceWakeAt(w.surf); !t.IsZero() {
			if deadline.IsZero() || t.Before(deadline) {
				deadline = t
			}
		}
		if w.animPeriod > 0 {
			t := now.Add(w.animPeriod)
			if deadline.IsZero() || t.Before(deadline) {
				deadline = t
			}
		}
	}
	if a.watchLook {
		t := now.Add(lookWatchInterval)
		if deadline.IsZero() || t.Before(deadline) {
			deadline = t
		}
	}
	if deadline.IsZero() {
		return -1
	}
	wait := deadline.Sub(now)
	if wait < 0 {
		return 0
	}
	return wait
}

func (a *Application) waitDisplay(timeout time.Duration) {
	for _, w := range a.Windows() {
		if w.Closed() {
			continue
		}
		if _, ok := w.surf.(platform.DisplayWaiter); ok {
			platform.WaitDisplay(w.surf, timeout)
			return
		}
	}
	platform.WaitDisplay(nil, timeout)
}

// PumpOnce processes one event burst then one frame on every window
// (tests / screenshots).
func (a *Application) PumpOnce() {
	a.pollLookFile()
	a.runPosted()
	for _, w := range a.Windows() {
		if w.Closed() {
			continue
		}
		w.pump()
	}
	for _, w := range a.Windows() {
		if w.Closed() {
			continue
		}
		w.frame()
	}
}

// Quit requests the run loop to exit. Safe from any goroutine: the flag is
// atomic and the display wait is woken so an idle loop (blocked on the
// display fd with no caret or timer pending) leaves immediately.
func (a *Application) Quit() {
	if a == nil {
		return
	}
	a.quit.Store(true)
	a.wakeUI()
}

// Quitting reports whether Quit has been requested.
func (a *Application) Quitting() bool { return a != nil && a.quit.Load() }

// reap drops closed windows. A status-menu window is toolkit chrome; once
// the last application window is gone it is destroyed too, so its surface
// does not keep the process (or an X11 display wait) alive.
func (a *Application) reap() {
	a.mu.Lock()
	var orphan []*Window
	real := 0
	out := a.windows[:0]
	for _, w := range a.windows {
		if w.Closed() {
			continue
		}
		out = append(out, w)
		if !w.statusMenu {
			real++
		}
	}
	a.windows = out
	if real == 0 {
		for _, w := range a.windows {
			if w.statusMenu {
				orphan = append(orphan, w)
			}
		}
	}
	a.mu.Unlock()
	for _, w := range orphan {
		w.Close()
	}
	if len(orphan) == 0 {
		return
	}
	a.mu.Lock()
	out = a.windows[:0]
	for _, w := range a.windows {
		if !w.Closed() {
			out = append(out, w)
		}
	}
	a.windows = out
	a.mu.Unlock()
}

func (a *Application) remove(w *Window) {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.windows[:0]
	for _, x := range a.windows {
		if x != w {
			out = append(out, x)
		}
	}
	a.windows = out
}
