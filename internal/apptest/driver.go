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

func checkPopup(w *app.Window) error {
	if w == nil || w.Popup() == nil {
		return nil
	}
	if errs := uitest.TreeInvariants(w.Popup()); len(errs) > 0 {
		return errs[0]
	}
	if pop, ok := w.Popup().(*widgets.PopupMenu); ok {
		return uitest.CheckMenuFitsItems(pop)
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
	step(&out, "gallery", "combo-popup", func() error { return openGalleryCombo(a, w) })
	step(&out, "gallery", "form-popups", func() error { return driveFormPopups(a, w) })
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

	// Mail's chrome row is the window's title bar: check it with the content.
	checkWindow := func() error {
		if err := checkTree(w.Content()); err != nil {
			return err
		}
		if tb := w.TitleBar(); tb != nil {
			return checkTree(tb)
		}
		return nil
	}
	step(&out, "mail", "construct", checkWindow)
	step(&out, "mail", "resize", func() error {
		w.Inject(platform.Event{Kind: platform.EventResize, Width: 1100, Height: 720})
		a.PumpOnce()
		return checkWindow()
	})
	step(&out, "mail", "drag-splitters", func() error { return dragSplitters(a, w) })
	step(&out, "mail", "scroll-lists", func() error { return scrollCollections(a, w, opts.Short) })
	step(&out, "mail", "select-rows", func() error { return selectRows(a, w) })
	step(&out, "mail", "context-menu", func() error { return openMailContextMenu(a, w) })
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
		// Leave the sash in the middle of pane A (not the thread-list
		// header, which now sits at the pane's top-left after the list
		// toolbar moved onto the M row).
		pa = sp.PaneA()
		away := paintengine2d.Pt(o.X+(pa.Min.X+pa.Max.X)*0.5, o.Y+(pa.Min.Y+pa.Max.Y)*0.5)
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

// driveFormPopups opens the Form tab's date field and colour button from
// the keyboard, as a user would: each drop-down must sit below its field,
// Escape must close it and focus must come back to the field.
func driveFormPopups(a *app.Application, w *app.Window) error {
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok {
			tv.Select(4)
		}
	})
	a.PumpOnce()
	var df *widgets.DateField
	var cbtn *widgets.ColorButton
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.DateField:
			df = v
		case *widgets.ColorButton:
			cbtn = v
		}
	})
	if df == nil || cbtn == nil {
		return fmt.Errorf("form tab: date field %v, colour button %v", df != nil, cbtn != nil)
	}
	key := func(k platform.Key, mods platform.Modifiers) {
		w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: k, Mods: mods})
		w.Inject(platform.Event{Kind: platform.EventKeyUp, Key: k, Mods: mods})
		a.PumpOnce()
	}
	for _, f := range []widget.Component{df, cbtn} {
		w.RequestFocus(f)
		a.PumpOnce()
		if f == widget.Component(df) {
			key(platform.KeyDown, platform.ModAlt)
		} else {
			key(platform.KeySpace, 0)
		}
		pop := w.Popup()
		if pop == nil {
			return fmt.Errorf("%T: the keyboard did not open its drop-down", f)
		}
		fb, pb := widget.DeviceBounds(f), pop.Bounds()
		if pb.Min.Y < fb.Max.Y-0.5 && pb.Max.Y > fb.Min.Y+0.5 {
			return fmt.Errorf("%T drop-down %v overlaps its field %v", f, pb, fb)
		}
		if err := checkTree(pop); err != nil {
			return err
		}
		key(platform.KeyEscape, 0)
		if w.Popup() != nil {
			return fmt.Errorf("%T: Escape left the drop-down open", f)
		}
		if w.Focus() != f {
			return fmt.Errorf("%T: focus went to %T after Escape, want the field back", f, w.Focus())
		}
	}
	return nil
}

func openGalleryCombo(a *app.Application, w *app.Window) error {
	var cb *widgets.ComboBox
	widget.Walk(w.Content(), func(c widget.Component) {
		if v, ok := c.(*widgets.ComboBox); ok && cb == nil && len(v.Items) > 0 {
			cb = v
		}
	})
	if cb == nil {
		return fmt.Errorf("gallery has no ComboBox")
	}
	cb.Open()
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		return fmt.Errorf("expected combo popup, got %T", w.Popup())
	}
	field := widget.DeviceBounds(cb)
	pb := pop.Bounds()
	if pb.Min.Y < field.Max.Y-0.5 && pb.Max.Y > field.Min.Y+0.5 {
		return fmt.Errorf("combo popup %+v overlaps field %+v", pb, field)
	}
	if err := uitest.CheckMenuFitsItems(pop); err != nil {
		return err
	}
	cb.Close()
	a.PumpOnce()
	return nil
}

func mailMessageMenuItems() []*widgets.MenuItem {
	return []*widgets.MenuItem{
		widgets.Item("Reply", nil),
		widgets.Item("Forward", nil),
		widgets.Sep(),
		widgets.Item("Mark as Read", nil),
		widgets.Item("Mark as Unread", nil),
		widgets.Item("Star", nil),
		widgets.Sep(),
		widgets.Item("Tag · Important", nil),
		widgets.Item("Mute Thread", nil),
		widgets.Item("Add sender to VIP", nil),
		widgets.Item("Archive", nil),
		widgets.Item("Junk", nil),
		widgets.Item("Delete", nil),
	}
}

func openMailContextMenu(a *app.Application, w *app.Window) error {
	var table *widgets.TableView
	var cards *widgets.CardList
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && table == nil && tv.RowCount > 0 {
			table = tv
		}
		if cl, ok := c.(*widgets.CardList); ok && cards == nil && cl.Count > 0 {
			cards = cl
		}
	})
	switch {
	case table != nil:
		o := widget.DeviceOrigin(table)
		p := paintengine2d.Pt(o.X+48, o.Y+table.HeaderHeight()+10)
		w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: p, Button: platform.ButtonRight})
	case cards != nil:
		o := widget.DeviceOrigin(cards)
		p := paintengine2d.Pt(o.X+48, o.Y+24)
		w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: p, Button: platform.ButtonRight})
	default:
		// Demo list can be empty (first-run / filter); still size the Mail
		// message menu — including “Add sender to VIP”.
		if widgets.ShowContextMenu(w.Content(), paintengine2d.Pt(40, 80), mailMessageMenuItems()...) == nil {
			return fmt.Errorf("ShowContextMenu failed")
		}
	}
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		return fmt.Errorf("expected mail context menu, got %T", w.Popup())
	}
	if err := uitest.CheckMenuFitsItems(pop); err != nil {
		return err
	}
	if err := checkPopup(w); err != nil {
		return err
	}
	w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	a.PumpOnce()
	if w.Popup() != nil {
		return fmt.Errorf("escape should dismiss context menu")
	}
	return nil
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
