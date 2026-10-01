// Package diag collects what the toolkit could not do the way an
// application asked for it.
//
// The toolkit has several layers entitled to override an application's
// stated intent: the look's era decides how a title bar meets the frame,
// build tags decide which theme engines exist, the person's icon
// directory decides which artwork is found, the compositor decides
// whether the toolkit may draw a frame at all. Each of those is a
// reasonable rule. Together they meant that an application could state
// something clearly, be overruled by a layer it had never heard of, and
// get no signal at all — a wrong pixel and nothing to pull on.
//
// So every override says so, in one place, in a shape that can be acted
// on: what was asked, what happened instead, and the exact call or build
// flag that gets what was asked. Findings are *collected* as well as
// logged, which is the part that matters for a test:
//
//	a := uitoolkit.New(uitoolkit.Options{...})
//	win, _ := a.NewWindow(...)
//	for _, f := range a.Diagnostics() {
//	    t.Errorf("%v", f)
//	}
//
// An application that asserts on this cannot regress into the confusion
// silently, which no amount of documentation achieves.
//
// It is not an error channel. A finding is never a reason to stop; it is
// a reason to know. Anything that must fail is returned as an error.
package diag

import (
	"fmt"
	"log"
	"sync"
)

// Level says whether the application lost something it asked for.
type Level uint8

const (
	// Note is the toolkit doing something reasonable that a developer
	// may nonetheless be surprised by. Nothing was asked for and denied:
	// a classic look putting a title bar in a row under its own strip is
	// the era being honoured, and the note is there for the developer
	// wondering why their tab strip is not the caption.
	Note Level = iota
	// Warn is something the application stated and did not get: a theme
	// pack whose engine is not in the build, a client frame a compositor
	// would not give.
	Warn
)

func (l Level) String() string {
	if l == Warn {
		return "warn"
	}
	return "note"
}

// Finding is one thing the toolkit did other than what was asked.
//
// Every field is written for the person reading the line. Asked and Got
// are concrete — a pack's name, a mode, a stem — and Fix is a call they
// can paste or a flag they can build with, never "see the docs".
type Finding struct {
	Level Level
	// Area is the part of the toolkit the finding came from: "caption",
	// "theme", "icons", "decorations", "tray", "x11", "text".
	Area string
	// Asked is what the application stated, in its own terms.
	Asked string
	// Got is what happened instead, and why.
	Got string
	// Fix is the exact call, field or build flag that gets what was
	// asked. Empty where nothing the application can do would change it.
	Fix string
}

func (f Finding) String() string {
	s := fmt.Sprintf("uitk %s: asked %s, got %s", f.Area, f.Asked, f.Got)
	if f.Fix != "" {
		s += " — " + f.Fix
	}
	return s
}

var state struct {
	mu    sync.Mutex
	seen  map[Finding]bool
	found []Finding
	quiet bool
}

// maxFindings caps what is kept. A finding repeats only if it is not
// identical to one already held, so reaching this means a program is
// generating genuinely distinct findings in a loop and keeping more
// would be a leak rather than a service.
const maxFindings = 256

// Report records a finding, once per identical finding, and logs it
// unless logging is off. It is safe from any goroutine and cheap enough
// for a paint path, because the second and later reports of the same
// finding do nothing but take a lock and a map lookup.
func Report(f Finding) {
	if f.Area == "" {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.seen == nil {
		state.seen = map[Finding]bool{}
	}
	if state.seen[f] {
		return
	}
	state.seen[f] = true
	if len(state.found) < maxFindings {
		state.found = append(state.found, f)
	}
	if !state.quiet {
		log.Print(f.String())
	}
}

// Findings is everything reported so far, oldest first.
func Findings() []Finding {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]Finding(nil), state.found...)
}

// SetLogging turns the log line on or off. Findings are collected either
// way, so a program that shows them in its own interface can silence the
// log without losing them. On by default.
func SetLogging(on bool) {
	state.mu.Lock()
	state.quiet = !on
	state.mu.Unlock()
}

// Reset drops everything, so a test can assert on a clean slate.
func Reset() {
	state.mu.Lock()
	state.seen, state.found = nil, nil
	state.mu.Unlock()
}
