// Package themesheet renders a "theme sheet": every control of a
// LookAndFeel in every interesting state, laid out on one deterministic
// offscreen image. Engine authors use it to check their work
// (cmd/uitk-themesheet), tests use it for goldens and the contract test,
// and Settings can use it for previews.
package themesheet

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// Width and Height are the sheet size at 1x.
const (
	Width  = 1180
	Height = 1070
)

// sheet is the painter state: a context, the look and a 1x→device scale.
type sheet struct {
	ctx *paintengine2d.Context
	lk  style.LookAndFeel
	sc  float32
	p   style.Palette
}

func (s *sheet) u(v float32) float32 { return v * s.sc }

func (s *sheet) r(x, y, w, h float32) paintengine2d.Rect {
	return paintengine2d.XYWH(s.u(x), s.u(y), s.u(w), s.u(h))
}

// caption writes a small state label above a cell.
func (s *sheet) caption(x, y float32, text string) {
	f := s.lk.MutedFont()
	if f == nil {
		return
	}
	f.Draw(s.ctx, text, paintengine2d.Pt(s.u(x), s.u(y)), s.p.TextMuted)
}

// heading writes a section title.
func (s *sheet) heading(x, y float32, text string) {
	f := s.lk.BoldFont()
	if f == nil {
		f = s.lk.Font()
	}
	f.Draw(s.ctx, text, paintengine2d.Pt(s.u(x), s.u(y)), s.p.Text)
}

// Render paints the sheet for lk. The look should already carry the
// display scale (style.WithScale); the image is Width×Height times that
// scale.
func Render(lk style.LookAndFeel, title string) *paintengine2d.Image {
	sc := float32(1)
	if c, ok := lk.(*style.Classic); ok {
		sc = c.Scale()
	}
	img := paintengine2d.NewImage(int(Width*sc+0.5), int(Height*sc+0.5))
	ctx := paintengine2d.NewContext(img)
	s := &sheet{ctx: ctx, lk: lk, sc: sc, p: lk.Palette()}
	ctx.Clear(s.p.Background)
	if bl, ok := lk.(style.WindowBackgroundLook); ok {
		bl.DrawWindowBackground(ctx, paintengine2d.XYWH(0, 0, float32(img.Width), float32(img.Height)))
	}
	s.heading(16, 10, title)

	// Left column.
	s.buttons(16, 40)
	s.toggles(16, 190)
	s.fields(16, 375)
	s.scrollbars(16, 570)
	s.ranges(16, 715)
	s.bars(16, 850)
	// Right column.
	s.tabsMenus(610, 40)
	s.rows(610, 400)
	s.frames(610, 745)
	return img
}

var (
	stN   = style.StateNone
	stH   = style.StateHovered
	stP   = style.StatePressed | style.StateHovered
	stF   = style.StateFocused
	stD   = style.StateDisabled
	stDef = style.StatePrimary
)

type cell struct {
	name string
	st   style.ControlState
}

func (s *sheet) buttons(x, y float32) {
	s.heading(x, y, "Push buttons")
	cells := []cell{{"normal", stN}, {"hover", stH}, {"pressed", stP}, {"focused", stF}, {"disabled", stD}, {"default", stDef}, {"default+focus", stDef | stF}}
	for i, c := range cells {
		cx := x + float32(i)*82
		s.caption(cx, y+22, c.name)
		s.lk.DrawButton(s.ctx, s.r(cx, y+42, 76, 28), c.st, "Button")
	}
	s.caption(x, y+76, "tool: normal / hover / pressed / toggle / toggle on / on+hover / disabled")
	tools := []style.ControlState{stN, stH, stP, style.StateToggle, style.StateToggle | style.StateChecked, style.StateToggle | style.StateChecked | stH, stD}
	icons := []style.ToolIcon{style.IconNew, style.IconOpen, style.IconSave, style.IconCut, style.IconCopy, style.IconPaste, style.IconSave}
	for i, st := range tools {
		s.lk.DrawToolButton(s.ctx, s.r(x+float32(i)*44, y+94, 40, 34), st, "", icons[i])
	}
	s.lk.DrawToolButton(s.ctx, s.r(x+312, y+94, 90, 34), stH, "Fetch", style.IconOpen)
}

func (s *sheet) toggles(x, y float32) {
	s.heading(x, y, "Check boxes, radios, switches")
	cells := []cell{{"off", stN}, {"hover", stH}, {"pressed", stP}, {"focused", stF}, {"disabled", stD}}
	for i, c := range cells {
		cx := x + float32(i)*112
		s.caption(cx, y+22, c.name)
		s.lk.DrawCheckbox(s.ctx, s.r(cx, y+40, 108, 22), c.st, false, "Check")
		s.lk.DrawCheckbox(s.ctx, s.r(cx, y+64, 108, 22), c.st, true, "Checked")
		s.lk.DrawRadio(s.ctx, s.r(cx, y+88, 108, 22), c.st, false, "Radio")
		s.lk.DrawRadio(s.ctx, s.r(cx, y+112, 108, 22), c.st, true, "Selected")
	}
	s.lk.DrawSwitch(s.ctx, s.r(x, y+136, 120, 26), stN, false, "Off")
	s.lk.DrawSwitch(s.ctx, s.r(x+130, y+136, 120, 26), stN, true, "On")
	s.lk.DrawSwitch(s.ctx, s.r(x+260, y+136, 120, 26), stH|stF, true, "Focus")
	s.lk.DrawSwitch(s.ctx, s.r(x+390, y+136, 120, 26), stD, true, "Disabled")
}

func (s *sheet) fields(x, y float32) {
	s.heading(x, y, "Fields, combos, spinners")
	s.caption(x, y+22, "empty")
	s.lk.DrawTextField(s.ctx, s.r(x, y+40, 180, 28), stN, "", "Placeholder", 0, 0, 0, false, 0, nil)
	s.caption(x+190, y+22, "focused + selection + caret")
	s.lk.DrawTextField(s.ctx, s.r(x+190, y+40, 180, 28), stF, "Ada Lovelace", "", 3, 0, 3, true, 0, nil)
	s.caption(x+380, y+22, "disabled")
	s.lk.DrawTextField(s.ctx, s.r(x+380, y+40, 180, 28), stD, "Read only", "", 0, 0, 0, false, 0, nil)
	s.caption(x, y+74, "combo / hover / open / focused")
	s.lk.DrawComboBox(s.ctx, s.r(x, y+92, 130, 28), stN, "Choice", false)
	s.lk.DrawComboBox(s.ctx, s.r(x+140, y+92, 130, 28), stH, "Hover", false)
	s.lk.DrawComboBox(s.ctx, s.r(x+280, y+92, 130, 28), stN, "Open", true)
	s.lk.DrawComboBox(s.ctx, s.r(x+420, y+92, 130, 28), stF, "Focused", false)
	s.caption(x, y+126, "spinner up hover / down pressed / disabled")
	s.lk.DrawSpinner(s.ctx, s.r(x, y+144, 20, 28), stN, true, false, false, false)
	s.lk.DrawSpinner(s.ctx, s.r(x+30, y+144, 20, 28), stN, false, true, false, true)
	s.lk.DrawSpinner(s.ctx, s.r(x+60, y+144, 20, 28), stD, false, false, false, false)
	lines := []style.TextLine{{Text: "A text area", Start: 0, End: 11}, {Text: "with two lines", Start: 12, End: 26}}
	s.lk.DrawTextArea(s.ctx, s.r(x+100, y+126, 220, 60), stF, lines, 5, 2, 6, true, 0, 0, "", nil)
	s.lk.DrawTextArea(s.ctx, s.r(x+330, y+126, 230, 60), stN, nil, 0, 0, 0, false, 0, 0, "Placeholder", nil)
}

func (s *sheet) scrollbars(x, y float32) {
	s.heading(x, y, "Scrollbars")
	s.caption(x+120, y+2, "normal · thumb hot · pressed · arrow hot · arrow pressed · none")
	states := []style.ScrollState{
		{},
		{Hot: style.ScrollThumbPart, Hovered: true},
		{Pressed: style.ScrollThumbPart, Hot: style.ScrollThumbPart, Hovered: true},
		{Hot: style.ScrollInc, Hovered: true},
		{Pressed: style.ScrollDec, Hot: style.ScrollDec, Hovered: true},
	}
	for i, st := range states {
		view := s.r(x+float32(i)*60, y+22, 50, 100)
		parts := style.ScrollGeometry(s.lk, view, true, 400, view.Dy(), 80*s.sc, false)
		style.DrawScrollBarParts(s.lk, s.ctx, parts, true, st)
	}
	view := s.r(x+300, y+22, 50, 100)
	parts := style.ScrollGeometry(s.lk, view, true, 100, view.Dy()+10, 0, false)
	style.DrawScrollBarParts(s.lk, s.ctx, parts, true, style.ScrollState{Disabled: true})
	h := s.r(x+360, y+22, 220, 50)
	hp := style.ScrollGeometry(s.lk, h, false, 900, h.Dx(), 300*s.sc, false)
	style.DrawScrollBarParts(s.lk, s.ctx, hp, false, style.ScrollState{Hot: style.ScrollThumbPart, Hovered: true})
}

func (s *sheet) ranges(x, y float32) {
	s.heading(x, y, "Sliders and progress")
	sl := []struct {
		t  float32
		st style.ControlState
		n  string
	}{{0, stN, "0"}, {0.3, stH, "hover"}, {0.6, stP, "pressed"}, {1, stF, "focused"}, {0.5, stD, "disabled"}}
	for i, c := range sl {
		cx := x + float32(i)*112
		s.caption(cx, y+22, c.n)
		s.lk.DrawSlider(s.ctx, s.r(cx, y+38, 104, 24), c.st, c.t)
	}
	pr := []struct {
		t      float32
		ind    bool
		phase  float32
		st     style.ControlState
		n      string
		width  float32
		offset float32
	}{{0, false, 0, stN, "0%", 100, 0}, {0.35, false, 0, stN, "35%", 100, 110}, {1, false, 0, stN, "100%", 100, 220},
		{0, true, 0.2, stN, "busy", 100, 330}, {0.5, false, 0, stD, "disabled", 100, 440}}
	for _, c := range pr {
		s.caption(x+c.offset, y+72, c.n)
		s.lk.DrawProgressBar(s.ctx, s.r(x+c.offset, y+90, c.width, 18), c.st, c.t, c.ind, c.phase)
	}
}

func (s *sheet) tabsMenus(x, y float32) {
	s.heading(x, y, "Tabs")
	bar := s.r(x, y+22, 540, 32)
	s.lk.DrawTabBar(s.ctx, bar)
	// Adjacent tabs as a TabBar lays them out: the selected one (second, so
	// both neighbours show any overlap) paints last, grown by the outset.
	tabs := []struct {
		n   string
		st  style.ControlState
		sel bool
	}{{"Normal", stN, false}, {"Selected", stN, true}, {"Hover", stH, false}, {"Pressed", stP, false}, {"Disabled", stD, false}}
	tabRect := func(i int) paintengine2d.Rect {
		r := s.r(x+float32(i)*100, y+22, 100, 32)
		if tabs[i].sel {
			out := style.TabOutsetOf(s.lk)
			r = paintengine2d.XYWH(r.Min.X-out.Left, r.Min.Y-out.Top, r.Dx()+out.Left+out.Right, r.Dy()+out.Top+out.Bottom)
		}
		return r
	}
	tabState := func(i int) style.ControlState {
		st := tabs[i].st
		if i == 0 {
			st |= style.StateFirst
		}
		if i == len(tabs)-1 {
			st |= style.StateLast
		}
		return st
	}
	for i, t := range tabs {
		if !t.sel {
			s.lk.DrawTab(s.ctx, tabRect(i), tabState(i), t.n, false)
		}
	}
	for i, t := range tabs {
		if t.sel {
			s.lk.DrawTab(s.ctx, tabRect(i), tabState(i), t.n, true)
		}
	}
	s.lk.DrawTab(s.ctx, s.r(x+440, y+60, 96, 32), stF, "Focused", true)

	s.heading(x, y+100, "Menus")
	mb := s.r(x, y+122, 540, 26)
	s.lk.DrawMenuBar(s.ctx, mb)
	s.lk.DrawMenuTitle(s.ctx, s.r(x+4, y+122, 52, 26), stN, "File", 0, true)
	s.lk.DrawMenuTitle(s.ctx, s.r(x+60, y+122, 52, 26), stH, "Edit", 0, false)
	s.lk.DrawMenuTitle(s.ctx, s.r(x+116, y+122, 52, 26), stN, "View", 0, false)
	s.lk.DrawMenuTitle(s.ctx, s.r(x+172, y+122, 60, 26), stF, "Help", 0, false)
	frame := s.r(x+4, y+150, 230, 190)
	style.DrawPopupShadowOf(s.lk, s.ctx, frame, style.PopupMenu)
	s.lk.DrawMenuFrame(s.ctx, frame)
	ch := style.MenuChromeFor(s.lk)
	rows := []struct {
		st  style.ControlState
		row style.MenuRow
	}{
		{stN, style.MenuRow{Label: "New window", Shortcut: "Ctrl+N", Icon: style.IconNew, Underline: 0}},
		{stH, style.MenuRow{Label: "Open…", Shortcut: "Ctrl+O", Icon: style.IconOpen, Underline: 0}},
		{stD, style.MenuRow{Label: "Save", Shortcut: "Ctrl+S", Icon: style.IconSave}},
		{stN, style.MenuRow{Separator: true}},
		{stN, style.MenuRow{Label: "Word wrap", Checked: true}},
		{stN, style.MenuRow{Label: "Compact", Radio: true, Checked: true}},
		{stN, style.MenuRow{Label: "Relaxed", Radio: true}},
		{stN, style.MenuRow{Label: "Recent", Submenu: true}},
	}
	rowH := (frame.Dy() - ch.PadT - ch.PadB) / float32(len(rows))
	for i, rw := range rows {
		rb := paintengine2d.XYWH(frame.Min.X+ch.PadL, frame.Min.Y+ch.PadT+float32(i)*rowH, frame.Dx()-ch.PadL-ch.PadR, rowH)
		s.lk.DrawMenuItem(s.ctx, rb, rw.st, rw.row)
	}
	s.caption(x+250, y+156, "tooltip")
	tip := s.r(x+250, y+174, 150, 26)
	style.DrawPopupShadowOf(s.lk, s.ctx, tip, style.PopupTooltip)
	s.lk.DrawTooltip(s.ctx, tip, "A helpful tip")
	s.caption(x+250, y+210, "message icons")
	for i, ic := range []style.ToolIcon{style.IconInfo, style.IconWarning, style.IconError, style.IconQuestion} {
		s.lk.DrawMessageIcon(s.ctx, s.r(x+250+float32(i)*40, y+228, 32, 32), ic)
	}
}

func (s *sheet) rows(x, y float32) {
	s.heading(x, y, "Lists, trees, tables")
	// Lists, trees and tables paint on the field (view) colour.
	s.ctx.DrawRect(s.r(x, y+22, 170, 110), paintengine2d.Fill(s.p.Field))
	s.lk.DrawListRow(s.ctx, s.r(x, y+24, 170, 24), false, false, "List row")
	s.lk.DrawListRow(s.ctx, s.r(x, y+48, 170, 24), false, true, "Hovered row")
	s.lk.DrawListRow(s.ctx, s.r(x, y+72, 170, 24), true, false, "Selected row")
	s.lk.DrawListRow(s.ctx, s.r(x, y+96, 170, 24), true, true, "Selected+hover")
	s.ctx.DrawRect(s.r(x+180, y+22, 170, 110), paintengine2d.Fill(s.p.Field))
	s.ctx.DrawRect(s.r(x+360, y+46, 180, 72), paintengine2d.Fill(s.p.Field))
	s.lk.DrawTreeRow(s.ctx, s.r(x+180, y+24, 170, 24), false, false, true, false, 0, "Inbox", true)
	s.lk.DrawTreeRow(s.ctx, s.r(x+180, y+48, 170, 24), false, true, false, false, 1, "Archives", false)
	s.lk.DrawTreeRow(s.ctx, s.r(x+180, y+72, 170, 24), true, false, false, true, 2, "2026", false)
	s.lk.DrawTreeRow(s.ctx, s.r(x+180, y+96, 170, 24), false, false, false, true, 1, "Sent", false)
	hx := x + 360
	s.lk.DrawTableHeader(s.ctx, s.r(hx, y+22, 60, 24), stN, "Name", false, false)
	s.lk.DrawTableHeader(s.ctx, s.r(hx+60, y+22, 60, 24), stH, "Hover", true, true)
	s.lk.DrawTableHeader(s.ctx, s.r(hx+120, y+22, 60, 24), stP, "Press", true, false)
	for i, sel := range []bool{false, true, false} {
		ry := y + 46 + float32(i)*24
		hov := i == 2
		s.lk.DrawTableCell(s.ctx, s.r(hx, ry, 60, 24), sel, hov, "Cell", style.AlignStart, nil)
		s.lk.DrawTableCell(s.ctx, s.r(hx+60, ry, 60, 24), sel, hov, "12", style.AlignEnd, nil)
		s.lk.DrawTableCell(s.ctx, s.r(hx+120, ry, 60, 24), sel, hov, "Row", style.AlignCenter, nil)
	}
	s.caption(x, y+150, "accordion: collapsed / expanded / hover / focused")
	s.lk.DrawAccordionHeader(s.ctx, s.r(x, y+168, 130, 28), stN, "Section", false)
	s.lk.DrawAccordionHeader(s.ctx, s.r(x+136, y+168, 130, 28), stN, "Open", true)
	s.lk.DrawAccordionHeader(s.ctx, s.r(x+272, y+168, 130, 28), stH, "Hover", false)
	s.lk.DrawAccordionHeader(s.ctx, s.r(x+408, y+168, 130, 28), stF, "Focused", true)
	s.caption(x, y+204, "splitter: vertical normal / hot, horizontal")
	s.lk.DrawSplitter(s.ctx, s.r(x, y+222, 10, 80), true, stN)
	s.lk.DrawSplitter(s.ctx, s.r(x+20, y+222, 10, 80), true, stH)
	s.lk.DrawSplitter(s.ctx, s.r(x+40, y+256, 120, 10), false, stN)
	s.lk.DrawSeparator(s.ctx, s.r(x+180, y+222, 10, 80), true)
	s.lk.DrawSeparator(s.ctx, s.r(x+200, y+256, 120, 10), false)
}

func (s *sheet) bars(x, y float32) {
	s.heading(x, y, "Bars")
	s.lk.DrawTitleBar(s.ctx, s.r(x, y+22, 570, 34), "Title bar", "subtitle")
	tb := s.r(x, y+62, 570, 40)
	s.lk.DrawToolBar(s.ctx, tb)
	for i, ic := range []style.ToolIcon{style.IconNew, style.IconOpen, style.IconSave} {
		st := stN
		if i == 1 {
			st = stH
		}
		s.lk.DrawToolButton(s.ctx, s.r(x+4+float32(i)*40, y+65, 36, 34), st, "", ic)
	}
	s.lk.DrawSeparator(s.ctx, s.r(x+126, y+66, 8, 32), true)
	s.lk.DrawToolButton(s.ctx, s.r(x+138, y+65, 80, 34), style.StateToggle|style.StateChecked, "Snap", style.IconNone)
	s.lk.DrawStatusBar(s.ctx, s.r(x, y+108, 570, 26), []string{"Ready.", "Ln 1, Col 1", "v1.0"})
	s.caption(x, y+144, "overlay dimmer")
	s.lk.DrawButton(s.ctx, s.r(x, y+162, 120, 30), stN, "Behind")
	s.lk.DrawOverlay(s.ctx, s.r(x, y+160, 180, 40))
	s.lk.DrawLabel(s.ctx, s.r(x+200, y+162, 180, 30), "Label (accent)", s.p.Accent, style.AlignStart)
	s.lk.DrawLabel(s.ctx, s.r(x+380, y+162, 180, 30), "Label (danger)", s.p.Danger, style.AlignStart)
}

func (s *sheet) frames(x, y float32) {
	s.heading(x, y, "Frames")
	gb := s.r(x, y+22, 260, 150)
	if g, ok := s.lk.(style.GroupBoxLook); ok {
		g.DrawGroupBox(s.ctx, gb, "Group box", false)
		in := g.GroupBoxInsets(true).Apply(gb)
		s.lk.DrawCheckbox(s.ctx, paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), s.u(22)), stN, true, "Inside a group")
	} else {
		s.lk.DrawPanel(s.ctx, gb, false)
		s.lk.DrawTitleBar(s.ctx, paintengine2d.XYWH(gb.Min.X, gb.Min.Y, gb.Dx(), s.lk.Metrics().TitleBar), "Group box", "")
	}
	wf := s.r(x+280, y+22, 270, 150)
	if w, ok := s.lk.(style.WindowFrameLook); ok {
		style.DrawPopupShadowOf(s.lk, s.ctx, wf, style.PopupDialog)
		w.DrawWindowFrame(s.ctx, wf, "Dialog", style.WindowState{Active: true, CanClose: true, CloseHot: true})
		in := w.WindowFrameInsets().Apply(wf)
		f := s.lk.Font()
		if f != nil {
			f.Draw(s.ctx, "Save changes?", paintengine2d.Pt(in.Min.X+s.u(12), in.Min.Y+s.u(12)), s.p.Text)
		}
		s.lk.DrawButton(s.ctx, paintengine2d.XYWH(in.Max.X-s.u(170), in.Max.Y-s.u(40), s.u(76), s.u(28)), stDef|stF, "Save")
		s.lk.DrawButton(s.ctx, paintengine2d.XYWH(in.Max.X-s.u(88), in.Max.Y-s.u(40), s.u(76), s.u(28)), stN, "Cancel")
	}
	s.lk.DrawPanel(s.ctx, s.r(x, y+190, 260, 60), true)
	s.caption(x+8, y+196, "raised panel")
	s.lk.DrawPanel(s.ctx, s.r(x+280, y+190, 270, 60), false)
	s.caption(x+288, y+196, "flat panel")
}
