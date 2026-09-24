package apptest

import (
	"fmt"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Options configure a driver run.
type Options struct {
	// Apps is "gallery", "settings", or "all" (default).
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

// Run exercises the gallery and Settings — the two windows in this repo with
// enough chrome to be worth driving. Mail moved to its own repository and
// drives itself there; see docs/testing.md.
func Run(opts Options) []Result {
	var out []Result
	apps := strings.ToLower(strings.TrimSpace(opts.Apps))
	if apps == "" || apps == "all" {
		out = append(out, runGallery(opts)...)
		out = append(out, runSettings(opts)...)
		return out
	}
	for _, a := range strings.Split(apps, ",") {
		switch strings.TrimSpace(a) {
		case "gallery":
			out = append(out, runGallery(opts)...)
		case "settings":
			out = append(out, runSettings(opts)...)
		default:
			out = append(out, Result{App: a, Step: "select", Err: fmt.Errorf("unknown app %q (gallery|settings|all)", a)})
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

func runSettings(opts Options) []Result {
	var out []Result
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "settings-driver", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		return []Result{{App: "settings", Step: "window", Err: err}}
	}
	defer w.Close()
	w.SetContent(demo.SettingsApp(a, w))
	a.PumpOnce()

	// Settings carries its chrome in the title bar, as Mail did: check it
	// with the content.
	checkWindow := func() error {
		if err := checkTree(w.Content()); err != nil {
			return err
		}
		if tb := w.TitleBar(); tb != nil {
			return checkTree(tb)
		}
		return nil
	}
	step(&out, "settings", "construct", checkWindow)
	step(&out, "settings", "resize", func() error {
		w.Inject(platform.Event{Kind: platform.EventResize, Width: 1100, Height: 720})
		a.PumpOnce()
		return checkWindow()
	})
	step(&out, "settings", "drag-splitters", func() error { return dragSplitters(a, w) })
	step(&out, "settings", "scroll-lists", func() error { return scrollCollections(a, w, opts.Short) })
	step(&out, "settings", "select-rows", func() error { return selectRows(a, w) })
	step(&out, "settings", "context-menu", func() error { return openRowContextMenu(a, w) })
	step(&out, "settings", "readonly-preview", func() error { return typeIntoReadOnlyViews(a, w) })
	if !opts.Short {
		// Every page in turn: each one builds its own controls, and the
		// preview restages the look underneath them.
		step(&out, "settings", "pages", func() error { return walkSettingsPages(a) })
	}
	return out
}

// walkSettingsPages opens Settings once per page, in a fresh window, because
// a page is built when it is first shown and the driver is looking for what
// construction breaks.
func walkSettingsPages(a *app.Application) error {
	for _, page := range []string{"Themes", "Appearance", "Packs & icons", "About"} {
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "settings-page", Width: 1100, Height: 720, Headless: true,
		})
		if err != nil {
			return err
		}
		w.SetContent(demo.SettingsAppOpen(a, w, "", page))
		a.PumpOnce()
		err = checkTree(w.Content())
		w.Close()
		a.PumpOnce()
		if err != nil {
			return fmt.Errorf("page %q: %w", page, err)
		}
	}
	return nil
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
		if sp.Axis == widgets.SplitColumns {
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

func openRowContextMenu(a *app.Application, w *app.Window) error {
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
		// A list can be empty (a filter that matches nothing); still size a
		// menu long enough to have to fit its items.
		if widgets.ShowContextMenu(w.Content(), paintengine2d.Pt(40, 80), rowMenuItems()...) == nil {
			return fmt.Errorf("ShowContextMenu failed")
		}
	}
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		return fmt.Errorf("expected a context menu, got %T", w.Popup())
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

// rowMenuItems is a menu with enough items, separators and long labels to
// make a popup that has to be measured and flipped against a screen edge.
func rowMenuItems() []*widgets.MenuItem {
	return []*widgets.MenuItem{
		widgets.Item("Open", nil),
		widgets.Item("Open in New Window", nil),
		widgets.Sep(),
		widgets.Item("Copy", nil),
		widgets.Item("Duplicate", nil),
		widgets.Item("Rename…", nil),
		widgets.Sep(),
		widgets.Item("Add to Favourites", nil),
		widgets.Item("Properties…", nil),
		widgets.Item("Delete", nil),
	}
}
