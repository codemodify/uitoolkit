package apptest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/internal/mail"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Options configure a driver run.
type Options struct {
	// Apps is "gallery", "mail", or "all" (default).
	Apps string
	// Short skips compose and extra resize/scroll passes.
	Short bool
}

// Result is one scripted step.
type Result struct {
	App  string
	Step string
	Err  error
}

func (r Result) String() string {
	if r.Err != nil {
		return fmt.Sprintf("FAIL %s/%s: %v", r.App, r.Step, r.Err)
	}
	return fmt.Sprintf("ok   %s/%s", r.App, r.Step)
}

// Run exercises gallery and/or the in-memory Mail UI. It never opens the
// user's mail.json or a live IMAP daemon — see docs/testing.md.
func Run(opts Options) []Result {
	var out []Result
	apps := strings.ToLower(strings.TrimSpace(opts.Apps))
	if apps == "" || apps == "all" {
		out = append(out, runGallery(opts)...)
		out = append(out, runMail(opts)...)
		return out
	}
	for _, a := range strings.Split(apps, ",") {
		switch strings.TrimSpace(a) {
		case "gallery":
			out = append(out, runGallery(opts)...)
		case "mail":
			out = append(out, runMail(opts)...)
		default:
			out = append(out, Result{App: a, Step: "select", Err: fmt.Errorf("unknown app %q (gallery|mail|all)", a)})
		}
	}
	return out
}

// Failed reports whether any step failed.
func Failed(rs []Result) bool {
	for _, r := range rs {
		if r.Err != nil {
			return true
		}
	}
	return false
}

func step(out *[]Result, app, name string, fn func() error) {
	err := fn()
	*out = append(*out, Result{App: app, Step: name, Err: err})
}

func checkTree(root widget.Component) error {
	if errs := uitest.TreeInvariants(root); len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func runGallery(opts Options) []Result {
	var out []Result
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "gallery-driver", Width: 1000, Height: 760, Headless: true,
	})
	if err != nil {
		return []Result{{App: "gallery", Step: "window", Err: err}}
	}
	defer w.Close()
	w.SetContent(demo.Gallery(a, w, false))
	a.PumpOnce()

	step(&out, "gallery", "construct", func() error { return checkTree(w.Content()) })
	sizes := [][2]int{{1000, 760}, {860, 560}}
	if !opts.Short {
		sizes = append(sizes, [2]int{1200, 800})
	}
	for _, sz := range sizes {
		ww, hh := sz[0], sz[1]
		step(&out, "gallery", fmt.Sprintf("resize-%dx%d", ww, hh), func() error {
			w.Inject(platform.Event{Kind: platform.EventResize, Width: ww, Height: hh})
			a.PumpOnce()
			return checkTree(w.Content())
		})
	}
	step(&out, "gallery", "drag-splitters", func() error {
		return dragSplitters(a, w)
	})
	step(&out, "gallery", "scroll-lists", func() error {
		return scrollCollections(a, w, opts.Short)
	})
	step(&out, "gallery", "select-rows", func() error {
		return selectRows(a, w)
	})
	return out
}

func runMail(opts Options) []Result {
	var out []Result
	dir, err := os.MkdirTemp("", "uitest-mail-")
	if err != nil {
		return []Result{{App: "mail", Step: "isolate", Err: err}}
	}
	defer os.RemoveAll(dir)
	if err := mail.IsolateTestEnv(dir); err != nil {
		return []Result{{App: "mail", Step: "isolate", Err: err}}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sock, stop, err := mail.StartDemo(ctx)
	if err != nil {
		return []Result{{App: "mail", Step: "startdemo", Err: err}}
	}
	defer stop()
	if !mail.IsDisposableMailSocket(sock) {
		return []Result{{App: "mail", Step: "socket", Err: fmt.Errorf("refusing non-disposable socket %s", sock)}}
	}

	cli, err := mail.DialWait(sock, 2*time.Second)
	if err != nil {
		return []Result{{App: "mail", Step: "dial", Err: err}}
	}
	defer cli.Close()
	st, err := cli.Status()
	if err != nil {
		return []Result{{App: "mail", Step: "status", Err: err}}
	}
	if err := mail.AssertMemoryBackend(st.Backend); err != nil {
		return []Result{{App: "mail", Step: "backend", Err: err}}
	}

	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "mail-driver", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		return []Result{{App: "mail", Step: "window", Err: err}}
	}
	defer w.Close()
	w.SetContent(mail.Open(a, w, cli, mail.AppOptions{ShowFilter: true}))
	a.PumpOnce()

	step(&out, "mail", "construct", func() error { return checkTree(w.Content()) })
	step(&out, "mail", "resize", func() error {
		w.Inject(platform.Event{Kind: platform.EventResize, Width: 1100, Height: 720})
		a.PumpOnce()
		return checkTree(w.Content())
	})
	step(&out, "mail", "drag-splitters", func() error { return dragSplitters(a, w) })
	step(&out, "mail", "scroll-lists", func() error { return scrollCollections(a, w, opts.Short) })
	step(&out, "mail", "select-rows", func() error { return selectRows(a, w) })
	step(&out, "mail", "readonly-preview", func() error { return typeIntoReadOnlyViews(a, w) })
	if !opts.Short {
		step(&out, "mail", "compose-editable", func() error {
			return driveCompose(a, cli)
		})
	}
	return out
}

func dragSplitters(a *app.Application, w *app.Window) error {
	var last error
	widget.Walk(w.Content(), func(c widget.Component) {
		sp, ok := c.(*widgets.Splitter)
		if !ok {
			return
		}
		o := widget.DeviceOrigin(sp)
		pa, pb := sp.PaneA(), sp.PaneB()
		var sash, aPt, bPt paintengine2d.Point
		if sp.Vertical {
			sash = paintengine2d.Pt(o.X+(pa.Max.X+pb.Min.X)*0.5, o.Y+24)
			aPt = paintengine2d.Pt(o.X+sp.LocalBounds().Dx()*0.25, sash.Y)
			bPt = paintengine2d.Pt(o.X+sp.LocalBounds().Dx()*0.75, sash.Y)
		} else {
			sash = paintengine2d.Pt(o.X+24, o.Y+(pa.Max.Y+pb.Min.Y)*0.5)
			aPt = paintengine2d.Pt(sash.X, o.Y+sp.LocalBounds().Dy()*0.25)
			bPt = paintengine2d.Pt(sash.X, o.Y+sp.LocalBounds().Dy()*0.75)
		}
		for _, dest := range []paintengine2d.Point{aPt, bPt, sash} {
			w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: sash, Button: platform.ButtonLeft})
			a.PumpOnce()
			w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: dest, Button: platform.ButtonLeft})
			a.PumpOnce()
			w.Inject(platform.Event{Kind: platform.EventMouseUp, Pos: dest, Button: platform.ButtonLeft})
			a.PumpOnce()
			if w.Cursor() != platform.CursorDefault && w.Cursor() != platform.CursorColResize && w.Cursor() != platform.CursorRowResize {
				last = fmt.Errorf("unexpected cursor %v", w.Cursor())
			}
			if err := checkTree(w.Content()); err != nil {
				last = err
			}
		}
		// Leave the sash so a stuck resize cursor would show.
		away := paintengine2d.Pt(o.X+8, o.Y+8)
		w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: away})
		a.PumpOnce()
		if w.Cursor() == platform.CursorColResize || w.Cursor() == platform.CursorRowResize {
			last = fmt.Errorf("stuck resize cursor after leaving sash: %v", w.Cursor())
		}
	})
	return last
}

func scrollCollections(a *app.Application, w *app.Window, short bool) error {
	var last error
	passes := []float32{0, 0.45, 1, 0}
	if short {
		passes = []float32{0, 1, 0}
	}
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.ListView:
			if v.MaxOffset() <= 0 {
				return
			}
			for _, f := range passes {
				v.ScrollTo(v.MaxOffset() * f)
				a.PumpOnce()
				if err := checkTree(w.Content()); err != nil {
					last = err
				}
			}
		case *widgets.TableView:
			if v.MaxOffset() <= 0 {
				return
			}
			for _, f := range passes {
				v.ScrollTo(v.MaxOffset() * f)
				a.PumpOnce()
				if err := checkTree(w.Content()); err != nil {
					last = err
				}
			}
		case *widgets.CardList:
			if v.MaxOffset() <= 0 {
				return
			}
			for _, f := range passes {
				v.ScrollTo(v.MaxOffset() * f)
				a.PumpOnce()
				if err := checkTree(w.Content()); err != nil {
					last = err
				}
			}
		case *widgets.TreeView:
			if v.MaxOffset() <= 0 {
				return
			}
			for _, f := range passes {
				v.ScrollTo(v.MaxOffset() * f)
				a.PumpOnce()
				if err := checkTree(w.Content()); err != nil {
					last = err
				}
			}
		case *widgets.ScrollView:
			if v.MaxOffset() <= 0 {
				return
			}
			for _, f := range passes {
				v.ScrollTo(v.MaxOffset() * f)
				a.PumpOnce()
				if err := checkTree(w.Content()); err != nil {
					last = err
				}
			}
		}
	})
	return last
}

func selectRows(a *app.Application, w *app.Window) error {
	var last error
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.ListView:
			if v.Count == 0 {
				return
			}
			v.Selected = 0
			if v.OnSelect != nil {
				v.OnSelect(0)
			}
			if v.Count > 3 {
				v.Selected = 3
				if v.OnSelect != nil {
					v.OnSelect(3)
				}
			}
		case *widgets.TableView:
			if v.RowCount == 0 {
				return
			}
			v.Selected = 0
			if v.OnSelect != nil {
				v.OnSelect(0)
			}
		}
	})
	a.PumpOnce()
	if err := checkTree(w.Content()); err != nil {
		last = err
	}
	return last
}

func typeIntoReadOnlyViews(a *app.Application, w *app.Window) error {
	var last error
	widget.Walk(w.Content(), func(c widget.Component) {
		ta, ok := c.(*widgets.TextArea)
		if !ok || !ta.ReadOnly {
			return
		}
		before := ta.Text
		w.RequestFocus(ta)
		a.PumpOnce()
		w.Inject(platform.Event{Kind: platform.EventText, Rune: 'Z'})
		a.PumpOnce()
		if ta.Text != before {
			last = fmt.Errorf("read-only view accepted typing: %q -> %q", before, ta.Text)
		}
		if ta.TextInput('x') {
			last = fmt.Errorf("read-only TextInput succeeded (%q)", ta.Placeholder)
		}
	})
	return last
}

func driveCompose(a *app.Application, cli *mail.Client) error {
	if err := mail.AssertMemoryBackend(mustBackend(cli)); err != nil {
		return err
	}
	cw, err := mail.OpenCompose(a, cli, mail.ComposeOptions{})
	if err != nil {
		return err
	}
	defer cw.Close()
	a.PumpOnce()
	var body *widgets.TextArea
	widget.Walk(cw.Content(), func(c widget.Component) {
		if ta, ok := c.(*widgets.TextArea); ok && !ta.ReadOnly && body == nil {
			body = ta
		}
	})
	if body == nil {
		return fmt.Errorf("compose missing editable TextArea")
	}
	before := body.Text
	body.SetSelection(len([]rune(body.Text)), len([]rune(body.Text)))
	if !body.TextInput('x') {
		return fmt.Errorf("compose body rejected typing")
	}
	if body.Text == before {
		return fmt.Errorf("compose body did not change")
	}
	return checkTree(cw.Content())
}

func mustBackend(cli *mail.Client) string {
	st, err := cli.Status()
	if err != nil {
		return ""
	}
	return st.Backend
}
