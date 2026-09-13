package uitest

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Norm is one desktop-toolkit chrome/layout contract.
type Norm struct {
	ID      string
	Control string
	Peer    string
	Want    string
	Check   func() error
}

// CompareRow is a printable checklist line.
type CompareRow struct {
	ID      string
	Control string
	Peer    string
	Want    string
	Err     error
}

func (r CompareRow) String() string {
	if r.Err != nil {
		return fmt.Sprintf("FAIL  %-22s  %s  (%s)", r.Control, r.Want, r.Err)
	}
	return fmt.Sprintf("ok    %-22s  %s", r.Control, r.Want)
}

// ChromeNorms is the Avalonia / Qt / GTK behavior map used by tests and
// `go run ./cmd/uitest-driver -compare`.
func ChromeNorms() []Norm {
	return []Norm{
		{
			ID: "button-focus-visible", Control: "Button",
			Peer:  "GTK :focus-visible · Qt WA_KeyboardFocusChange · Avalonia :focus-visible",
			Want:  "mouse click does not leave a focus ring; Tab does",
			Check: checkButtonFocusVisible,
		},
		{
			ID: "checkbox-click-release", Control: "CheckBox",
			Peer:  "Qt QCheckBox clicked · GTK GtkCheckButton · Avalonia CheckBox",
			Want:  "toggle on release inside bounds; drag-off cancels; disabled ignores keys",
			Check: checkCheckboxClick,
		},
		{
			ID: "radio-click-release", Control: "Radio",
			Peer:  "Qt QRadioButton · GTK GtkCheckButton · Avalonia RadioButton",
			Want:  "select on complete click, not press-and-drag-off",
			Check: checkRadioClick,
		},
		{
			ID: "toggle-vs-action", Control: "ToolToggle",
			Peer:  "Avalonia ToggleButton (SourceGit view modes) · Qt QToolButton checkable · GTK GtkToggleButton",
			Want:  "off well + on filled chrome differ from a flat ToolButton",
			Check: checkToggleChrome,
		},
		{
			ID: "toolbar-gaps", Control: "ToolBar",
			Peer:  "Qt QToolBar spacing · GTK toolbar box · Avalonia CommandBar spacing",
			Want:  "icon↔label and adjacent tools keep ToolItemGap; labels do not collide",
			Check: checkToolBarGaps,
		},
		{
			ID: "toolbar-mouse-focus", Control: "ToolBar",
			Peer:  "GTK / Qt toolbuttons suppress ring after pointer activate",
			Want:  "mouse click clears keyboard focus chrome",
			Check: checkToolBarMouseFocus,
		},
		{
			ID: "menubar-dismiss-focus", Control: "MenuBar",
			Peer:  "Qt QMenuBar · GTK GtkPopoverMenuBar · Avalonia Menu",
			Want:  "Close / dismiss drops leftover title focus ring",
			Check: checkMenuBarDismiss,
		},
		{
			ID: "popup-leave-highlight", Control: "PopupMenu",
			Peer:  "GTK menu prelight · Avalonia MenuItem hover",
			Want:  "pointer leave clears row highlight unless keyboard nav is active",
			Check: checkPopupLeave,
		},
		{
			ID: "menu-hover-bordered", Control: "PopupMenu",
			Peer:  "Office XP / Win32 menu hot-track · Qt QMenu highlight · GTK menu prelight",
			Want:  "hover highlight is a bordered fill across icon gutter + label, not text invert",
			Check: checkMenuHoverBordered,
		},
		{
			ID: "menubar-title-hover", Control: "MenuBar",
			Peer:  "Office XP / Qt QMenuBar · GTK GtkPopoverMenuBar",
			Want:  "only the hovered (or open) title paints XP highlight; siblings stay idle",
			Check: checkMenuBarTitleHover,
		},
		{
			ID: "textfield-inactive-sel", Control: "TextField",
			Peer:  "Qt QLineEdit inactive · GTK GtkEntry · Avalonia TextBox",
			Want:  "unfocused selection is de-emphasized vs the live accent band",
			Check: checkTextFieldInactiveSel,
		},
		{
			ID: "label-clip", Control: "Label",
			Peer:  "Qt QLabel elide · GTK GtkLabel · Avalonia TextBlock clip",
			Want:  "arranged narrower than measure does not paint past bounds",
			Check: checkLabelClip,
		},
		{
			ID: "button-label-clip", Control: "Button",
			Peer:  "Qt QPushButton · GTK GtkButton · Avalonia Button",
			Want:  "constrained width clips / fits the label",
			Check: checkButtonClip,
		},
		{
			ID: "combo-popup-clear", Control: "ComboBox",
			Peer:  "Qt QComboBox · GTK GtkDropDown · Avalonia ComboBox",
			Want:  "popup sits below the field and fits long labels",
			Check: checkComboPopup,
		},
		{
			ID: "scroll-overflow-thumb", Control: "ScrollView",
			Peer:  "Qt QScrollBar · GTK GtkScrolledWindow · Avalonia ScrollViewer",
			Want:  "overflow shows a thumb in-track; fitting content hides it; wheel clamps",
			Check: checkScrollOverflow,
		},
		{
			ID: "list-scroll-clip", Control: "ListView",
			Peer:  "Qt QListView · GTK GtkListView · Avalonia ListBox",
			Want:  "scrolled rows clip to the viewport (no header/body bleed)",
			Check: checkListScrollClip,
		},
		{
			ID: "table-header-flush", Control: "TableView",
			Peer:  "Qt QTableView · GTK GtkColumnView · Avalonia DataGrid",
			Want:  "first row flush under sticky header; scroll does not bleed into header",
			Check: checkTableHeaderFlush,
		},
		{
			ID: "table-column-resize", Control: "TableView",
			Peer:  "Qt QHeaderView resize · GTK GtkColumnView · Avalonia DataGrid",
			Want:  "header divider drag resizes a column and restores the pointer",
			Check: checkTableColumnResize,
		},
		{
			ID: "splitter-clip-cursor", Control: "Splitter",
			Peer:  "Qt QSplitter · GTK GtkPaned · Avalonia GridSplitter",
			Want:  "exclusive panes; children clipped; cursor returns to pointer after drag",
			Check: checkSplitterClipCursor,
		},
		{
			ID: "combo-field-height", Control: "ComboBox",
			Peer:  "Qt QComboBox · GTK GtkDropDown · Avalonia ComboBox",
			Want:  "closed height is ComboH; Escape dismisses; Down opens and navigates",
			Check: checkComboKeyboard,
		},
		{
			ID: "field-toolbar-height", Control: "TextField",
			Peer:  "Qt QLineEdit · GTK GtkEntry · Avalonia TextBox",
			Want:  "TextField / NumberField measure FieldHeight (ComboH); spinner is one tab stop",
			Check: checkFieldHeights,
		},
		{
			ID: "combo-focus-visible", Control: "ComboBox",
			Peer:  "GTK :focus-visible · Avalonia :focus-visible",
			Want:  "mouse click does not leave a focus ring; Tab does",
			Check: checkComboFocusVisible,
		},
		{
			ID: "switch-click-release", Control: "Switch",
			Peer:  "GTK GtkSwitch · Qt Quick Switch · Avalonia ToggleSwitch",
			Want:  "toggle on release inside bounds; drag-off cancels; disabled ignores keys",
			Check: checkSwitchClick,
		},
		{
			ID: "slider-click-disabled", Control: "Slider",
			Peer:  "Qt QSlider · GTK GtkScale · Avalonia Slider",
			Want:  "click sets value; clamp at ends; disabled ignores drag; focus-visible; muted chrome",
			Check: checkSliderChrome,
		},
		{
			ID: "progress-metrics", Control: "ProgressBar",
			Peer:  "Qt QProgressBar · GTK GtkProgressBar · Avalonia ProgressBar",
			Want:  "value clamped 0..1; height is ProgressH; disabled fill is muted",
			Check: checkProgressChrome,
		},
		{
			ID: "tabs-click-focus", Control: "TabBar",
			Peer:  "Qt QTabBar · GTK GtkNotebook · Avalonia TabControl",
			Want:  "select on release; disabled ignores keys; mouse click clears focus ring",
			Check: checkTabBarChrome,
		},
		{
			ID: "dialog-buttons", Control: "MessageBox",
			Peer:  "Qt QMessageBox · GTK GtkAlertDialog · Avalonia dialog",
			Want:  "primary Yes/OK; Escape cancels; click-release on Yes",
			Check: checkDialogButtons,
		},
		{
			ID: "menu-gutter-icons", Control: "PopupMenu",
			Peer:  "Office XP gutter · Qt QMenu · GTK menu",
			Want:  "icons and checks paint in the gutter; width follows the widest label",
			Check: checkMenuGutterIcons,
		},
		{
			ID: "menu-onscreen-clamp", Control: "PopupMenu",
			Peer:  "Qt QMenu · GTK GtkPopover · Avalonia ContextMenu",
			Want:  "right-edge / Help-near-edge menus stay on-screen at intrinsic width",
			Check: checkMenuOnScreen,
		},
	}
}

// CompareReport runs every chrome norm (driver -compare / tests).
func CompareReport() []CompareRow {
	var out []CompareRow
	for _, n := range ChromeNorms() {
		out = append(out, CompareRow{
			ID: n.ID, Control: n.Control, Peer: n.Peer, Want: n.Want, Err: n.Check(),
		})
	}
	return out
}

func checkButtonFocusVisible() error {
	btn := widgets.NewButton("OK", nil)
	other := widgets.NewButton("Next", nil)
	row := widgets.NewRow(btn, other).WithGap(8)
	s := Mount(row, paintengine2d.XYWH(0, 0, 280, 40))
	pt := paintengine2d.Pt(20, 20)
	s.MousePress(pt, platform.ButtonLeft)
	s.MouseRelease(pt)
	if btn.State().Focused() {
		return fmt.Errorf("mouse click left StateFocused")
	}
	s.Tab(true)
	if s.Host.Focus() != other {
		return fmt.Errorf("Tab did not move focus")
	}
	if !other.State().Focused() {
		return fmt.Errorf("Tab focus did not paint StateFocused")
	}
	return nil
}

func checkCheckboxClick() error {
	n := 0
	c := widgets.NewCheckbox("X", false, func(bool) { n++ })
	s := Mount(c, paintengine2d.XYWH(0, 0, 160, 32))
	s.MousePress(paintengine2d.Pt(8, 16), platform.ButtonLeft)
	if c.Checked || n != 0 {
		return fmt.Errorf("toggled on press")
	}
	s.MouseRelease(paintengine2d.Pt(8, 16))
	if !c.Checked || n != 1 {
		return fmt.Errorf("did not toggle on release")
	}
	s.MousePress(paintengine2d.Pt(8, 16), platform.ButtonLeft)
	c.MouseExit()
	s.MouseRelease(paintengine2d.Pt(200, 16))
	if !c.Checked || n != 1 {
		return fmt.Errorf("drag-off still toggled")
	}
	c.SetEnabled(false)
	c.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if !c.Checked || n != 1 {
		return fmt.Errorf("disabled checkbox toggled from keyboard")
	}
	return nil
}

func checkRadioClick() error {
	r := widgets.NewRadio("Solo", false, nil)
	s := Mount(r, paintengine2d.XYWH(0, 0, 140, 32))
	s.MousePress(paintengine2d.Pt(8, 16), platform.ButtonLeft)
	if r.Selected {
		return fmt.Errorf("selected on press")
	}
	s.MouseRelease(paintengine2d.Pt(8, 16))
	if !r.Selected {
		return fmt.Errorf("not selected on release")
	}
	return nil
}

func checkToggleChrome() error {
	action := widgets.NewToolBar(widgets.ToolText("Cards", nil))
	off := widgets.NewToolBar(widgets.ToolToggle("Cards", false, nil))
	on := widgets.NewToolBar(widgets.ToolToggle("Cards", true, nil))
	box := paintengine2d.XYWH(0, 0, 140, 36)
	sa, so, sn := Mount(action, box), Mount(off, box), Mount(on, box)
	pa, po, pn := sa.Paint(), so.Paint(), sn.Paint()
	if ColorDiff(pa, po, 40) < 40 {
		return fmt.Errorf("toggle-off matches flat action")
	}
	if ColorDiff(po, pn, 40) < 40 {
		return fmt.Errorf("toggle-on matches toggle-off")
	}
	return nil
}

func checkToolBarGaps() error {
	tb := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconOpen, "Get Messages", nil),
		widgets.ToolIconBtn(style.IconNew, "Write", nil),
		widgets.ToolDivider(),
		widgets.ToolToggle("Cards", false, nil),
		widgets.ToolToggle("Classic", true, nil),
	)
	s := Mount(tb, paintengine2d.XYWH(0, 0, 640, 36))
	sz := tb.Measure(layout.Unbounded())
	s.Relayout(paintengine2d.XYWH(0, 0, sz.X, 36))
	return CheckToolBarGaps(tb)
}

func checkToolBarMouseFocus() error {
	tb := widgets.NewToolBar(widgets.ToolText("Delete", nil), widgets.ToolText("Next", nil))
	s := Mount(tb, paintengine2d.XYWH(0, 0, 280, 36))
	pt := tb.ItemCenter(0)
	s.MousePress(pt, platform.ButtonLeft)
	s.MouseRelease(pt)
	if tb.KeyboardChrome() {
		return fmt.Errorf("mouse left keyboard chrome")
	}
	return nil
}

func checkMenuBarDismiss() error {
	mb := widgets.NewMenuBar(widgets.NewMenu("&Help", widgets.Item("About", nil)))
	s := Mount(mb, paintengine2d.XYWH(0, 0, 400, 28))
	if !mb.HandleAlt(platform.KeyH) {
		return fmt.Errorf("alt+H")
	}
	mb.Close()
	if mb.KeyboardChrome() {
		return fmt.Errorf("Close left keyboard chrome")
	}
	_ = s
	return nil
}

func checkMenuHoverBordered() error {
	if err := checkMenuHoverOnLook(style.DarkLook()); err != nil {
		return fmt.Errorf("dark: %w", err)
	}
	if err := checkMenuHoverOnLook(style.LightLook()); err != nil {
		return fmt.Errorf("light: %w", err)
	}
	if pack, ok := style.LoadTheme("luna"); ok {
		if err := checkMenuHoverOnLook(pack.Look()); err != nil {
			return fmt.Errorf("luna: %w", err)
		}
	}
	return checkMenuBarOpenTitle(style.DarkLook())
}

func checkMenuBarTitleHover() error {
	if err := checkMenuBarTitleHoverOnLook(style.LightLook()); err != nil {
		return fmt.Errorf("light: %w", err)
	}
	return checkMenuBarTitleHoverOnLook(style.DarkLook())
}

func checkMenuBarTitleHoverOnLook(lk style.LookAndFeel) error {
	h := NewHost()
	h.SetLook(lk)
	mb := widgets.NewMenuBar(
		widgets.NewMenu("&File", widgets.Item("New", nil)),
		widgets.NewMenu("&Help", widgets.Item("About", nil)),
	)
	s := MountHost(h, mb, paintengine2d.XYWH(0, 0, 480, 40))
	file := mb.TitleRect(0)
	help := mb.TitleRect(1)
	if file.Empty() || help.Empty() {
		return fmt.Errorf("empty titles")
	}
	s.MouseMove(paintengine2d.Pt((help.Min.X+help.Max.X)*0.5, (help.Min.Y+help.Max.Y)*0.5))
	img := s.Paint()
	p := style.ResolveMenuChrome(lk.Palette())
	hr, hg, hb := RGB8(p.MenuHover)
	helpFill := countNearRGB(img, help.Inset(1), hr, hg, hb, 48)
	fileFill := countNearRGB(img, file.Inset(1), hr, hg, hb, 48)
	if helpFill < 8 {
		return fmt.Errorf("hovered Help title lacks MenuHover (near=%d)", helpFill)
	}
	if fileFill >= 8 && fileFill*2 >= helpFill {
		return fmt.Errorf("idle File title painted MenuHover (file=%d help=%d) — full-bar wash", fileFill, helpFill)
	}
	s.MouseMove(paintengine2d.Pt(mb.Bounds().Dx()-12, (help.Min.Y+help.Max.Y)*0.5))
	if mb.HoverIndex() != -1 {
		return fmt.Errorf("empty-bar hover index %d (file=%+v help=%+v)", mb.HoverIndex(), file, help)
	}
	return nil
}

func checkMenuHoverOnLook(lk style.LookAndFeel) error {
	h := NewHost()
	h.SetLook(lk)
	disabled := widgets.Item("Disabled", nil)
	disabled.Disabled = true
	pop := widgets.NewPopupMenu(
		widgets.Item("Paragraph…", nil),
		disabled,
		widgets.Sep(),
		widgets.Item("Two", nil),
	)
	s := MountHost(h, pop, paintengine2d.XYWH(0, 0, 220, 120))
	idle := s.Paint()
	p := style.ResolveMenuChrome(lk.Palette())
	row0 := menuRowDevice(pop, 0)
	ch := style.MenuChromeFor(lk)
	gx := int(row0.Min.X + ch.CheckCol()*0.45)
	gy := int((row0.Min.Y + row0.Max.Y) * 0.5)
	if ColorDist(idle, gx, gy, p.MenuGutter) >= ColorDist(idle, gx, gy, p.MenuHover) {
		return fmt.Errorf("idle gutter %d,%d looks like MenuHover", gx, gy)
	}
	if ColorDist(idle, gx, gy, p.MenuGutter) >= ColorDist(idle, gx, gy, p.SurfaceAlt) {
		return fmt.Errorf("idle gutter %d,%d matches menu body", gx, gy)
	}
	pt := paintengine2d.Pt((row0.Min.X+row0.Max.X)*0.5, (row0.Min.Y+row0.Max.Y)*0.5)
	s.MouseMove(pt)
	if pop.HighlightedIndex() != 0 {
		return fmt.Errorf("hover index %d", pop.HighlightedIndex())
	}
	hot := s.Paint()
	if err := CheckMenuHoverBordered(hot, row0, lk); err != nil {
		return err
	}
	if ColorDiff(idle, hot, 28) < 20 {
		return fmt.Errorf("hover paint matches idle (text invert?)")
	}
	row1 := menuRowDevice(pop, 1)
	s.MouseMove(paintengine2d.Pt((row1.Min.X+row1.Max.X)*0.5, (row1.Min.Y+row1.Max.Y)*0.5))
	off := s.Paint()
	dx := int(row1.Min.X + ch.CheckCol()*0.45)
	dy := int((row1.Min.Y + row1.Max.Y) * 0.5)
	if ColorDist(off, dx, dy, p.MenuHover) <= ColorDist(off, dx, dy, p.MenuGutter) {
		return fmt.Errorf("disabled gutter %d,%d closer to MenuHover than MenuGutter", dx, dy)
	}
	return nil
}

func checkMenuBarOpenTitle(lk style.LookAndFeel) error {
	h := NewHost()
	h.SetLook(lk)
	mb := widgets.NewMenuBar(widgets.NewMenu("&Format", widgets.Item("Paragraph…", nil)))
	root := widgets.NewColumn(mb)
	s := MountHost(h, root, paintengine2d.XYWH(0, 0, 420, 240))
	mb.Open(0)
	img := s.Paint()
	tr := mb.TitleRect(0)
	if tr.Empty() {
		return fmt.Errorf("empty title")
	}
	box := tr.Translate(widget.DeviceOrigin(mb))
	if err := CheckMenuHoverBordered(img, box, lk); err != nil {
		return fmt.Errorf("open title: %w", err)
	}
	pop, ok := h.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		return fmt.Errorf("no dropdown")
	}
	if pop.Bounds().Min.Y > box.Max.Y+1.5 {
		return fmt.Errorf("dropdown %+v not attached to title %+v", pop.Bounds(), box)
	}
	_ = s
	return nil
}

func menuRowDevice(p *widgets.PopupMenu, i int) paintengine2d.Rect {
	if p == nil {
		return paintengine2d.Rect{}
	}
	return p.ItemBounds(i).Translate(widget.DeviceOrigin(p))
}

func checkPopupLeave() error {
	pop := widgets.NewPopupMenu(widgets.Item("One", nil), widgets.Item("Two", nil))
	s := Mount(pop, paintengine2d.XYWH(0, 0, 180, 80))
	s.MouseMove(paintengine2d.Pt(40, 20))
	if pop.HighlightedIndex() < 0 {
		return fmt.Errorf("hover should highlight")
	}
	pop.MouseExit()
	if pop.HighlightedIndex() >= 0 {
		return fmt.Errorf("leave left highlight %d", pop.HighlightedIndex())
	}
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if pop.HighlightedIndex() < 0 {
		return fmt.Errorf("keyboard should highlight")
	}
	return nil
}

func checkTextFieldInactiveSel() error {
	tf := widgets.NewTextField("Hello world", "", nil)
	s := Mount(tf, paintengine2d.XYWH(0, 0, 200, 32))
	s.Focus(tf)
	tf.SetSelection(0, len([]rune("Hello world")))
	tf.FocusGained()
	hot := s.Paint()
	other := widgets.NewButton("x", nil)
	s.Focus(other)
	tf.FocusLost()
	cold := s.Paint()
	if ColorDiff(hot, cold, 24) < 8 {
		return fmt.Errorf("unfocused selection still matches live accent")
	}
	return nil
}

func checkLabelClip() error {
	l := widgets.NewLabel("A very long sidebar caption that must not bleed")
	s := Mount(l, paintengine2d.XYWH(0, 0, 220, 36))
	l.Arrange(paintengine2d.XYWH(4, 4, 64, 28))
	return CheckInkInsideBounds(s, l, 2)
}

func checkButtonClip() error {
	b := widgets.NewButton("Get Messages Now", nil)
	s := Mount(b, paintengine2d.XYWH(0, 0, 220, 40))
	b.Arrange(paintengine2d.XYWH(4, 4, 72, 32))
	return CheckInkInsideBounds(s, b, 4)
}

func checkComboPopup() error {
	cb := widgets.NewComboBox([]string{"QQ <j9@nchip.com>", "Work <you@example.com>"}, 0, nil)
	root := widgets.NewPad(16, cb)
	s := Mount(root, paintengine2d.XYWH(0, 0, 480, 280))
	cb.Open()
	pop, ok := s.Host.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		return fmt.Errorf("no popup")
	}
	field := widget.DeviceBounds(cb)
	pb := pop.Bounds()
	if pb.Min.Y < field.Max.Y-0.5 && pb.Max.Y > field.Min.Y+0.5 {
		return fmt.Errorf("popup %+v overlaps field %+v", pb, field)
	}
	return CheckMenuFitsItems(pop)
}
