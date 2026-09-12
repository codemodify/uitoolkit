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
