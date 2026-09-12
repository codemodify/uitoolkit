package app

import (
	"fmt"
	"sync"
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
	mu        sync.Mutex
	look      style.LookAndFeel
	scale     float32
	headless  bool
	backend   platform.Backend
	windows   []*Window
	quit      bool
	onQuit    func()
	watchLook bool
	lookWatch *lookFileStamp
}

// New constructs an application. Default look is PreferredLook
// (XDG appearance, else dark Classic). When Look is nil, New also
// watches look.json so Settings → Apply updates running windows.
// Scale <= 0 means detect (env, then Xft.dpi on X11). Headless
// stays 1× unless UITK_SCALE / GDK_SCALE / QT_SCALE_FACTOR is set.
func New(opts Options) *Application {
	watch := opts.WatchLook
	if opts.Look == nil {
		opts.Look = style.PreferredLook()
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
	if opts.Scale <= 0 {
		if opts.Headless {
			opts.Scale = 1
			if s := platform.ScaleFromEnv(); s > 0 {
				opts.Scale = s
			}
		} else {
			opts.Scale = platform.DetectScale()
		}
	}
	if opts.Scale != 1 {
		opts.Look = style.WithScale(opts.Look, opts.Scale)
	}
	a := &Application{
		look:      opts.Look,
		scale:     opts.Scale,
		headless:  opts.Headless,
		backend:   backend,
		watchLook: watch,
	}
	if watch {
		a.lookWatch = newLookFileStamp()
	}
	return a
}

// Look is the default theme for new windows.
func (a *Application) Look() style.LookAndFeel { return a.look }

// SetLook swaps the theme on the app and every open window.
// Unscaled Classic looks are rebuilt with the application display scale
// so theme toggles do not drop HiDPI metrics.
func (a *Application) SetLook(l style.LookAndFeel) {
	if l == nil {
		return
	}
	l = applyScale(l, a.scale)
	a.look = l
	for _, w := range a.windows {
		w.look = l
		w.RequestLayout()
	}
}

func applyScale(look style.LookAndFeel, scale float32) style.LookAndFeel {
	if look == nil || scale <= 0 || scale == 1 {
		return look
	}
	if _, ok := look.(*style.Classic); !ok {
		return look
	}
	// Already HiDPI-scaled (density may have changed the 1× font size).
	if style.LookScale(look) > 1.01 {
		return look
	}
	return style.WithScale(look, scale)
}

// Scale is the display scale applied to layout metrics at the window.
func (a *Application) Scale() float32 { return a.scale }

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
func (a *Application) Run() error {
	if len(a.windows) == 0 {
		return fmt.Errorf("uitoolkit: Run with no windows")
	}
	var nextBlink time.Time
	for !a.quit {
		// Wayland / X11 / offscreen: poll look.json on every idle wake
		// (waitTimeout ≤ lookWatchInterval when WatchLook is on).
		a.pollLookFile()
		now := time.Now()
		if a.anyCaret() && (nextBlink.IsZero() || !now.Before(nextBlink)) {
			for _, w := range a.Windows() {
				w.toggleBlink()
			}
			nextBlink = now.Add(caretBlinkPeriod)
		}
		alive := 0
		for _, w := range a.Windows() {
			if w.surf.Closed() {
				w.Close()
				continue
			}
			w.pump()
			w.frame()
			alive++
		}
		a.reap()
		if alive == 0 {
			a.quit = true
			break
		}
		if a.quit {
			break
		}
		timeout := a.waitTimeout(time.Now(), nextBlink)
		if a.anyNeedsPaint() {
			timeout = 0
		}
		a.waitDisplay(timeout)
	}
	if a.onQuit != nil {
		a.onQuit()
	}
	return nil
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
		if _, ok := w.surf.(platform.DisplayWaiter); ok {
			platform.WaitDisplay(w.surf, timeout)
			return
		}
	}
	platform.WaitDisplay(nil, timeout)
}

// PumpOnce processes one frame on every window (tests / screenshots).
func (a *Application) PumpOnce() {
	a.pollLookFile()
	for _, w := range a.Windows() {
		w.pump()
		w.frame()
	}
}

// Quit requests the run loop to exit.
func (a *Application) Quit() { a.quit = true }

func (a *Application) reap() {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.windows[:0]
	for _, w := range a.windows {
		if !w.closed {
			out = append(out, w)
		}
	}
	a.windows = out
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
