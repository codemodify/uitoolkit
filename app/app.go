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
}

// Application owns the run loop and open windows.
type Application struct {
	mu       sync.Mutex
	look     style.LookAndFeel
	scale    float32
	headless bool
	backend  platform.Backend
	windows  []*Window
	quit     bool
	onQuit   func()
}

// New constructs an application. Default look is dark Classic.
func New(opts Options) *Application {
	if opts.Look == nil {
		opts.Look = style.DarkLook()
	}
	if opts.Scale <= 0 {
		opts.Scale = 1
	}
	return &Application{
		look:     opts.Look,
		scale:    opts.Scale,
		headless: opts.Headless,
		backend:  platform.Default(opts.Headless),
	}
}

// Look is the default theme for new windows.
func (a *Application) Look() style.LookAndFeel { return a.look }

// SetLook swaps the theme on the app and every open window.
func (a *Application) SetLook(l style.LookAndFeel) {
	if l == nil {
		return
	}
	a.look = l
	for _, w := range a.windows {
		w.look = l
		w.fullInvalidate()
	}
}

// Scale is the display scale applied to layout metrics at the window.
func (a *Application) Scale() float32 { return a.scale }

// BackendName is "x11", "offscreen", or a stub.
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

// Run pumps events until Quit or the last window closes.
func (a *Application) Run() error {
	if len(a.windows) == 0 {
		return fmt.Errorf("uitoolkit: Run with no windows")
	}
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()
	blink := time.NewTicker(530 * time.Millisecond)
	defer blink.Stop()
	for !a.quit {
		select {
		case <-blink.C:
			for _, w := range a.Windows() {
				w.toggleBlink()
			}
		case <-ticker.C:
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
		}
	}
	if a.onQuit != nil {
		a.onQuit()
	}
	return nil
}

// PumpOnce processes one frame on every window (tests / screenshots).
func (a *Application) PumpOnce() {
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
