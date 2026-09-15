package style

import (
	"sort"
	"strings"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// Engine paints the controls of one theme family — a "shape language" such
// as Win95 bevels, Luna gradients, Aqua gel, Motif shadows or NeXT greys. It
// is to a theme pack what a QStyle subclass is to a QPalette: the pack
// supplies colours and metrics, the engine decides what a button *is*.
//
// A [Classic] look owns the palette, metrics, fonts and tokens; every Draw*
// call on it is routed to its engine with the look as the first argument.
// Engines embed [BaseEngine] (the stock painter) and override only what
// their era did differently.
//
// Dispatch rule: BaseEngine never calls its own methods directly — every
// part (Face, CheckIndicator, Arrow, ...) and every nested control
// (DrawFocusRing, ...) is re-dispatched through l.Engine(). So an engine that
// overrides a single part, for example CheckIndicator, changes the checkbox,
// the menu check column and anything else that paints that part, while
// inheriting all layout and text handling.
type Engine interface {
	// ID is the registry key ("base", "win95", "luna", "aqua", ...).
	ID() string
	// DefaultMetrics are the engine's native geometry at 1x (control
	// heights, scrollbar thickness, indicator sizes, radii). A pack's own
	// metrics override these field by field; zero means "no opinion".
	DefaultMetrics() ChromeMetrics

	// ---- parts --------------------------------------------------------

	// Face paints a control face (fill, bevel, border, gloss) for role in
	// state st and returns the foreground (label/glyph) colour to use on it.
	Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color
	// CheckIndicator paints the checkbox box and mark into box.
	CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool)
	// RadioIndicator paints the radio button indicator into box.
	RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool)
	// Arrow paints a direction glyph (combo drop arrow, spinner, scroll
	// arrows, submenu arrow, sort indicator) centred in b.
	Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color)
	// Expander paints a tree / accordion disclosure glyph (+/- box,
	// triangle, chevron) centred in b.
	Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color)
	// MenuHighlight paints the hot item of a menu (and an open menu-bar
	// title when attachBottom). Its text colour comes from MenuTextColor.
	MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool)
	// MenuTextColor is the label colour on a hot menu item / title.
	MenuTextColor(l *Classic, hot bool) paintengine2d.Color
	// FieldFocusRing reports whether focused text fields, text areas and
	// combos draw a focus ring (Win95 and Motif showed only the caret).
	FieldFocusRing(l *Classic) bool

	// ---- scrollbars ---------------------------------------------------

	// ScrollBarStyle describes thickness, overlay vs gutter, arrow buttons.
	ScrollBarStyle(l *Classic) ScrollBarStyle
	// DrawScrollBarParts paints a whole scrollbar from its geometry.
	DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState)

	// ---- frames ---------------------------------------------------------

	// GroupBoxInsets is the frame thickness around a group box body and the
	// title band height (top includes the title when hasTitle).
	GroupBoxInsets(l *Classic, hasTitle bool) Insets
	// DrawGroupBox paints a titled frame (Panel / Card / group box).
	DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool)
	// WindowFrameInsets is the chrome around an in-app window / dialog body
	// (top includes the caption bar).
	WindowFrameInsets(l *Classic) Insets
	// DrawWindowFrame paints an in-app window or dialog: frame, caption bar
	// with title and caption buttons.
	DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState)
	// DrawWindowBackground paints the window's background (pinstripes,
	// brushed metal, textures). It must depend only on window coordinates:
	// partial redraw repaints any sub-rect of it under a clip.
	DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	// TabOutset grows the selected tab's rect so it can overlap its
	// neighbours; it is painted after them. Zero for most looks.
	TabOutset(l *Classic) Insets
	// DrawTabPane paints the page under a tab bar (TabView).
	DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	// WindowCloseRect is where DrawWindowFrame put the close button for a
	// frame of bounds b (empty when there is none) — hit-testing uses it.
	WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect
	// PopupShadow is how far the drop shadow of a floating layer (menu,
	// list, tooltip, dialog) reaches outside its bounds; zero for none.
	PopupShadow(l *Classic, kind PopupKind) Insets
	// DrawPopupShadow paints that shadow around b, before the layer paints
	// over b. It must stay within b grown by PopupShadow.
	DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind)

	// ---- behaviour ------------------------------------------------------

	// StyleHint answers look-dependent behaviour questions (Qt's
	// QStyle::styleHint): dialog button order, tab alignment, ...
	StyleHint(l *Classic, h StyleHint) int

	// ---- whole controls (same shape as LookAndFeel, plus the look) ------

	DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool)
	DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string)
	DrawLabel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align)
	DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string)
	DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32)
	DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font)
	DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState)
	DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState)
	DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string)
	DrawOverlay(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool)
	DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow)
	DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool)
	DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool)
	DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string)
	DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect)
	DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon)
	DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32)
	DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string)
	DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool)
	DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string)
	DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon)
	DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool)
	DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string, align Align, face *Font)
	DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool)
	DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string)
	DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font)
	DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string)
	DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool)
	DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool)
}

// Role names the part of a control a [Engine.Face] paints.
type Role = chromeRole

// Face roles.
const (
	RoleButton   Role = roleButton   // push button
	RoleTool     Role = roleTool     // tool button / toggle
	RoleField    Role = roleField    // text field, text area
	RoleCheck    Role = roleCheck    // checkbox / radio indicator well
	RoleRow      Role = roleRow      // list / tree / table row highlight
	RoleTab      Role = roleTab      // tab
	RoleThumb    Role = roleThumb    // scrollbar thumb
	RoleTrack    Role = roleTrack    // scrollbar / slider track
	RoleMenu     Role = roleMenu     // menu hot item
	RoleCombo    Role = roleCombo    // combo box field
	RoleSplitter Role = roleSplitter // splitter handle
	RoleBar      Role = roleBar      // menu bar, tool bar, status bar, tab strip
	RolePanel    Role = rolePanel    // panel / group box surface
)

// Direction is an arrow glyph orientation.
type Direction uint8

const (
	DirDown Direction = iota
	DirUp
	DirLeft
	DirRight
)

// Insets is per-edge chrome thickness.
type Insets struct{ Top, Right, Bottom, Left float32 }

// Apply shrinks r by the insets.
func (in Insets) Apply(r paintengine2d.Rect) paintengine2d.Rect {
	return paintengine2d.XYWH(r.Min.X+in.Left, r.Min.Y+in.Top, r.Dx()-in.Left-in.Right, r.Dy()-in.Top-in.Bottom)
}

// StyleHint names a look-dependent behaviour (see Engine.StyleHint).
type StyleHint int

const (
	// HintDialogPrimaryFirst: 1 puts the default dialog button first
	// (Windows, KDE: "OK Cancel"); 0 puts it last (Mac, GNOME: "Cancel OK").
	HintDialogPrimaryFirst StyleHint = iota
	// HintTabsCentered: 1 centres tabs over their pane (Aqua segmented).
	HintTabsCentered
)

// LookHint reads a style hint from any look (0 for non-engine looks).
func LookHint(lk LookAndFeel, h StyleHint) int {
	if c, ok := lk.(*Classic); ok && c != nil {
		return c.eng().StyleHint(c, h)
	}
	return 0
}

// WindowState is the state of an in-app window / dialog frame.
type WindowState struct {
	Active      bool
	CloseHot    bool
	ClosePress  bool
	CanClose    bool
	Maximizable bool
}

// ---- scrollbar geometry -------------------------------------------------

// ArrowPlacement says where a scrollbar's step buttons sit.
type ArrowPlacement uint8

const (
	// ArrowsNone: thumb only (modern overlay bars, Breeze, Fluent).
	ArrowsNone ArrowPlacement = iota
	// ArrowsEnds: one button at each end (Win95, Motif, Metal, XP).
	ArrowsEnds
	// ArrowsTogetherEnd: both buttons at the bottom / right (Aqua, Platinum option).
	ArrowsTogetherEnd
	// ArrowsTogetherStart: both buttons at the top / left.
	ArrowsTogetherStart
)

// ScrollBarStyle is an engine's scrollbar policy at 1x (the look scales it).
type ScrollBarStyle struct {
	// Thickness of the bar across its axis. 0 = Metrics.Scroll.
	Thickness float32
	// Overlay bars float inside the content with Inset padding and do not
	// reserve layout space; classic bars reserve a Thickness-wide gutter.
	Overlay bool
	// Inset is the gap between an overlay bar and the view edge.
	Inset float32
	// Arrows places the step buttons; ArrowLen is each button's length
	// along the axis (0 = Thickness).
	Arrows   ArrowPlacement
	ArrowLen float32
	// MinThumb is the shortest thumb (0 = 24).
	MinThumb float32
	// EndPad is extra padding at both ends of an overlay track (0 = 4).
	EndPad float32
}

// ScrollPart identifies a hit region of a scrollbar.
type ScrollPart uint8

const (
	ScrollNone      ScrollPart = iota
	ScrollDec                  // step button towards the start (up / left)
	ScrollInc                  // step button towards the end (down / right)
	ScrollPageDec              // track before the thumb
	ScrollPageInc              // track after the thumb
	ScrollThumbPart            // the thumb
)

// ScrollParts is the laid-out geometry of one scrollbar.
type ScrollParts struct {
	Bar   paintengine2d.Rect // whole bar area (gutter)
	Track paintengine2d.Rect // groove the thumb travels in
	Thumb paintengine2d.Rect // empty when nothing to scroll
	Dec   paintengine2d.Rect // step-back button (empty when none)
	Inc   paintengine2d.Rect // step-forward button (empty when none)
}

// HitTest maps a point to the part under it.
func (p ScrollParts) HitTest(pt paintengine2d.Point, vertical bool) ScrollPart {
	switch {
	case !p.Dec.Empty() && p.Dec.Contains(pt):
		return ScrollDec
	case !p.Inc.Empty() && p.Inc.Contains(pt):
		return ScrollInc
	case !p.Thumb.Empty() && p.Thumb.Contains(pt):
		return ScrollThumbPart
	case !p.Track.Empty() && p.Track.Contains(pt) && !p.Thumb.Empty():
		if vertical {
			if pt.Y < p.Thumb.Min.Y {
				return ScrollPageDec
			}
			return ScrollPageInc
		}
		if pt.X < p.Thumb.Min.X {
			return ScrollPageDec
		}
		return ScrollPageInc
	}
	return ScrollNone
}

// ScrollState is the interaction state of a scrollbar for painting.
type ScrollState struct {
	Hot      ScrollPart // part under the pointer
	Pressed  ScrollPart // part held down
	Disabled bool       // nothing to scroll
	Hovered  bool       // pointer anywhere over the bar
}

// Part converts a scrollbar part's interaction into a ControlState.
func (s ScrollState) Part(part ScrollPart) ControlState {
	st := StateNone
	if s.Disabled {
		return StateDisabled
	}
	if s.Hot == part {
		st |= StateHovered
	}
	if s.Pressed == part {
		st |= StatePressed | StateHovered
	}
	return st
}

// ScrollBarStyleOf resolves the scrollbar policy of any look, scaled to the
// look's display scale. Non-Classic looks get a thumb-only overlay bar.
func ScrollBarStyleOf(lk LookAndFeel) ScrollBarStyle {
	var s ScrollBarStyle
	var scale float32
	if c, ok := lk.(*Classic); ok && c != nil {
		s = c.eng().ScrollBarStyle(c)
		scale = c.Scale()
	} else {
		s = ScrollBarStyle{Overlay: true, Inset: 2}
		scale = LookScale(lk)
	}
	if scale <= 0 {
		scale = 1
	}
	if s.Thickness <= 0 {
		// Metrics.Scroll is already at display scale; bring it back to 1x
		// so every field below scales once.
		s.Thickness = 10
		if lk != nil && lk.Metrics().Scroll > 0 {
			s.Thickness = lk.Metrics().Scroll / scale
		}
	}
	s.Thickness *= scale
	s.Inset *= scale
	s.ArrowLen *= scale
	s.MinThumb *= scale
	s.EndPad *= scale
	if s.ArrowLen <= 0 && s.Arrows != ArrowsNone {
		s.ArrowLen = s.Thickness
	}
	if s.MinThumb <= 0 {
		s.MinThumb = 24 * scale
	}
	if s.EndPad <= 0 && s.Overlay {
		s.EndPad = 4 * scale
	}
	return s
}

// ScrollGutter is the layout space content gives up to a visible
// scrollbar across its axis, so no row text ever runs under the bar:
// the bar plus its inset for overlay bars, the bar for gutter bars.
func ScrollGutter(lk LookAndFeel) float32 {
	s := ScrollBarStyleOf(lk)
	if s.Overlay {
		return s.Thickness + s.Inset*2
	}
	return s.Thickness
}

// ScrollGeometry lays out a scrollbar inside view (the scrolled area's
// bounds). vertical bars sit at the right edge, horizontal ones at the
// bottom. content and viewport are lengths along the axis; offset is the
// current scroll offset. reserveCorner shortens the bar by the other bar's
// thickness when both are shown.
func ScrollGeometry(lk LookAndFeel, view paintengine2d.Rect, vertical bool, content, viewport, offset float32, reserveCorner bool) ScrollParts {
	s := ScrollBarStyleOf(lk)
	var p ScrollParts
	if view.Empty() || s.Thickness <= 0 {
		return p
	}
	t := s.Thickness
	corner := float32(0)
	if reserveCorner {
		corner = t
	}
	if vertical {
		if s.Overlay {
			p.Bar = paintengine2d.XYWH(view.Max.X-t-s.Inset, view.Min.Y, t, view.Dy()-corner)
		} else {
			p.Bar = paintengine2d.XYWH(view.Max.X-t, view.Min.Y, t, view.Dy()-corner)
		}
	} else {
		if s.Overlay {
			p.Bar = paintengine2d.XYWH(view.Min.X, view.Max.Y-t-s.Inset, view.Dx()-corner, t)
		} else {
			p.Bar = paintengine2d.XYWH(view.Min.X, view.Max.Y-t, view.Dx()-corner, t)
		}
	}
	along := p.Bar.Dy()
	if !vertical {
		along = p.Bar.Dx()
	}
	if along < 16 {
		return ScrollParts{}
	}
	// Step buttons.
	a := s.ArrowLen
	if s.Arrows != ArrowsNone && along < a*2+8 {
		a = (along - 8) / 2
		if a < 6 {
			a = 0
		}
	}
	lo, hi := float32(0), along // track span along the axis, relative to bar start
	seg := func(from, n float32) paintengine2d.Rect {
		if vertical {
			return paintengine2d.XYWH(p.Bar.Min.X, p.Bar.Min.Y+from, p.Bar.Dx(), n)
		}
		return paintengine2d.XYWH(p.Bar.Min.X+from, p.Bar.Min.Y, n, p.Bar.Dy())
	}
	if a > 0 {
		switch s.Arrows {
		case ArrowsEnds:
			p.Dec = seg(0, a)
			p.Inc = seg(along-a, a)
			lo, hi = a, along-a
		case ArrowsTogetherEnd:
			p.Dec = seg(along-2*a, a)
			p.Inc = seg(along-a, a)
			hi = along - 2*a
		case ArrowsTogetherStart:
			p.Dec = seg(0, a)
			p.Inc = seg(a, a)
			lo = 2 * a
		}
	}
	if s.Overlay {
		lo += s.EndPad
		hi -= s.EndPad
	}
	if hi-lo < 4 {
		return p
	}
	p.Track = seg(lo, hi-lo)
	if viewport <= 0 || content <= viewport+0.5 {
		return p
	}
	maxOff := content - viewport
	if offset < 0 {
		offset = 0
	}
	if offset > maxOff {
		offset = maxOff
	}
	span := hi - lo
	th := span * viewport / content
	if th < s.MinThumb {
		th = s.MinThumb
	}
	if th > span {
		th = span
	}
	pos := lo + (span-th)*(offset/maxOff)
	p.Thumb = seg(pos, th)
	return p
}

// ScrollOffsetForThumb converts a thumb position (start of the thumb along
// the axis, in the same coordinates as p) back into a scroll offset in
// [0, maxOff].
func ScrollOffsetForThumb(p ScrollParts, vertical bool, thumbStart, maxOff float32) float32 {
	if p.Thumb.Empty() || maxOff <= 0 {
		return 0
	}
	var lo, span, th float32
	if vertical {
		lo, span, th = p.Track.Min.Y, p.Track.Dy(), p.Thumb.Dy()
	} else {
		lo, span, th = p.Track.Min.X, p.Track.Dx(), p.Thumb.Dx()
	}
	free := span - th
	if free <= 0 {
		return 0
	}
	f := (thumbStart - lo) / free
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	return f * maxOff
}

// DrawScrollBarParts paints a scrollbar on any look: Classic looks route to
// their engine; others fall back to track + thumb.
func DrawScrollBarParts(lk LookAndFeel, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	if lk == nil || ctx == nil {
		return
	}
	if c, ok := lk.(*Classic); ok && c != nil {
		c.eng().DrawScrollBarParts(c, ctx, p, vertical, st)
		return
	}
	if p.Thumb.Empty() {
		return
	}
	lk.DrawScrollBar(ctx, p.Track, p.Thumb, st.Part(ScrollThumbPart))
}

// ---- optional LookAndFeel extensions -----------------------------------

// GroupBoxLook is implemented by looks that paint titled frames themselves.
// Panel uses it when available and falls back to DrawPanel + DrawTitleBar.
type GroupBoxLook interface {
	GroupBoxInsets(hasTitle bool) Insets
	DrawGroupBox(ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool)
}

// PopupKind says what is floating, for its drop shadow.
type PopupKind uint8

const (
	PopupMenu    PopupKind = iota // menus, combo and completion lists
	PopupTooltip                  // tooltips
	PopupDialog                   // in-app windows: dialogs, message boxes
)

// PopupShadowLook drops shadows under floating layers. The window paints
// them for its popup and tooltip layers; overlays paint their dialog's.
type PopupShadowLook interface {
	PopupShadow(kind PopupKind) Insets
	DrawPopupShadow(ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind)
}

// PopupShadow implements [PopupShadowLook].
func (l *Classic) PopupShadow(kind PopupKind) Insets { return l.eng().PopupShadow(l, kind) }

// DrawPopupShadow implements [PopupShadowLook].
func (l *Classic) DrawPopupShadow(ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	l.eng().DrawPopupShadow(l, ctx, b, kind)
}

// PopupShadowOf is a look's shadow reach for kind (zero when it has none).
func PopupShadowOf(lk LookAndFeel, kind PopupKind) Insets {
	if s, ok := lk.(PopupShadowLook); ok {
		return s.PopupShadow(kind)
	}
	return Insets{}
}

// DrawPopupShadowOf paints lk's shadow for a layer at b, if it has one.
func DrawPopupShadowOf(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if s, ok := lk.(PopupShadowLook); ok {
		s.DrawPopupShadow(ctx, b, kind)
	}
}

// Grow returns r grown by in on each side.
func (in Insets) Grow(r paintengine2d.Rect) paintengine2d.Rect {
	return paintengine2d.Rect{
		Min: paintengine2d.Pt(r.Min.X-in.Left, r.Min.Y-in.Top),
		Max: paintengine2d.Pt(r.Max.X+in.Right, r.Max.Y+in.Bottom),
	}
}

// Zero reports whether every side is zero or less.
func (in Insets) Zero() bool {
	return in.Left <= 0 && in.Top <= 0 && in.Right <= 0 && in.Bottom <= 0
}

// Max is the per-side maximum of two insets.
func (in Insets) Max(o Insets) Insets {
	return Insets{Left: max(in.Left, o.Left), Top: max(in.Top, o.Top), Right: max(in.Right, o.Right), Bottom: max(in.Bottom, o.Bottom)}
}

// TabOutsetLook grows the selected tab so it overlaps its neighbours.
type TabOutsetLook interface {
	TabOutset() Insets
}

// TabOutset implements [TabOutsetLook].
func (l *Classic) TabOutset() Insets { return l.eng().TabOutset(l) }

// TabOutsetOf is the selected-tab outset of any look (zero when the look
// has none), already at display scale.
func TabOutsetOf(lk LookAndFeel) Insets {
	if t, ok := lk.(TabOutsetLook); ok {
		return t.TabOutset()
	}
	return Insets{}
}

// WindowBackgroundLook paints a window background beyond a flat colour.
type WindowBackgroundLook interface {
	DrawWindowBackground(ctx *paintengine2d.Context, b paintengine2d.Rect)
}

// TabPaneLook paints the page of a TabView.
type TabPaneLook interface {
	DrawTabPane(ctx *paintengine2d.Context, b paintengine2d.Rect)
}

// DrawWindowBackground implements [WindowBackgroundLook].
func (l *Classic) DrawWindowBackground(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawWindowBackground(l, ctx, b)
}

// DrawTabPane implements [TabPaneLook].
func (l *Classic) DrawTabPane(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawTabPane(l, ctx, b)
}

// WindowFrameLook is implemented by looks that paint in-app window /
// dialog chrome (caption bar, frame, caption buttons).
type WindowFrameLook interface {
	WindowFrameInsets() Insets
	DrawWindowFrame(ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState)
	WindowCloseRect(b paintengine2d.Rect) paintengine2d.Rect
}

// ---- registry -------------------------------------------------------------

var (
	engineMu   sync.RWMutex
	engines    = map[string]Engine{}
	packMu     sync.Mutex
	registered []ThemePack
	packGen    int // bumps on every RegisterPack so indexes rebuild
)

// RegisterEngine adds an engine to the registry (call from init). A second
// engine with the same ID replaces the first.
func RegisterEngine(e Engine) {
	if e == nil || strings.TrimSpace(e.ID()) == "" {
		return
	}
	engineMu.Lock()
	engines[strings.ToLower(e.ID())] = e
	engineMu.Unlock()
}

// EngineByID looks an engine up ("" and unknown IDs report false).
func EngineByID(id string) (Engine, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return nil, false
	}
	engineMu.RLock()
	e, ok := engines[id]
	engineMu.RUnlock()
	return e, ok
}

// EngineIDs lists registered engines, sorted.
func EngineIDs() []string {
	engineMu.RLock()
	out := make([]string, 0, len(engines))
	for id := range engines {
		out = append(out, id)
	}
	engineMu.RUnlock()
	sort.Strings(out)
	return out
}

// RegisterPack adds a built-in theme pack (call from init, typically next to
// the engine that paints it). A registered pack replaces a legacy era pack
// with the same Name, so an engine can take over "luna" or "motif".
func RegisterPack(p ThemePack) {
	if strings.TrimSpace(p.Name) == "" {
		return
	}
	p.Source = ThemeSourceBuiltin
	packMu.Lock()
	for i := range registered {
		if registered[i].Name == p.Name {
			registered[i] = p
			packGen++
			packMu.Unlock()
			return
		}
	}
	registered = append(registered, p)
	packGen++
	packMu.Unlock()
}

func registeredPacks() []ThemePack {
	packMu.Lock()
	defer packMu.Unlock()
	return append([]ThemePack(nil), registered...)
}

// Decade is the Settings grouping label for a year ("1990s"); 0 → "".
func Decade(year int) string {
	if year <= 0 {
		return ""
	}
	return itoa(year/10*10) + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// engineFor resolves the engine a token set names (BaseEngine fallback).
func engineFor(t ThemeTokens) Engine {
	if e, ok := EngineByID(t.Engine); ok {
		return e
	}
	return baseEngine
}

// eng is the look's engine (BaseEngine when unset or unknown).
func (l *Classic) eng() Engine {
	if l != nil && l.engine != nil {
		return l.engine
	}
	return baseEngine
}

// Engine is the engine painting this look.
func (l *Classic) Engine() Engine { return l.eng() }

// lookMemo caches values an engine derives from a look (resolved colour
// sets, gradients, paths). A look is immutable, so each value is built
// once; the lock only guards the first build (looks can be painted from
// tests in parallel).
type lookMemo struct {
	mu sync.Mutex
	m  map[any]any
}

// Memo returns the value cached under key for this look, building it on
// first use. Engines use it so painting does not re-derive colours:
//
//	c := l.Memo(w95Key{}, func() any { return w95build(l) }).(w95)
func (l *Classic) Memo(key any, build func() any) any {
	if l == nil {
		return build()
	}
	l.memo.mu.Lock()
	defer l.memo.mu.Unlock()
	if v, ok := l.memo.m[key]; ok {
		return v
	}
	if l.memo.m == nil {
		l.memo.m = map[any]any{}
	}
	v := build()
	l.memo.m[key] = v
	return v
}

// X reads an engine-specific colour from the pack's "extra" map, falling
// back to def when the pack does not define key.
func (l *Classic) X(key string, def paintengine2d.Color) paintengine2d.Color {
	if l == nil {
		return def
	}
	if c, ok := l.tokens.Extra[key]; ok {
		return c
	}
	return def
}

// P reads an engine-specific number from the pack's "params" map.
func (l *Classic) P(key string, def float32) float32 {
	if l == nil {
		return def
	}
	if v, ok := l.tokens.Params[key]; ok {
		return v
	}
	return def
}

// S scales a 1x design length to this look's display scale.
func (l *Classic) S(v float32) float32 {
	if l == nil {
		return v
	}
	return v * l.Scale()
}

// GroupBoxInsets implements [GroupBoxLook].
func (l *Classic) GroupBoxInsets(hasTitle bool) Insets { return l.eng().GroupBoxInsets(l, hasTitle) }

// DrawGroupBox implements [GroupBoxLook].
func (l *Classic) DrawGroupBox(ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	l.eng().DrawGroupBox(l, ctx, b, title, raised)
}

// WindowFrameInsets implements [WindowFrameLook].
func (l *Classic) WindowFrameInsets() Insets { return l.eng().WindowFrameInsets(l) }

// DrawWindowFrame implements [WindowFrameLook].
func (l *Classic) DrawWindowFrame(ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	l.eng().DrawWindowFrame(l, ctx, b, title, st)
}

// WindowCloseRect implements [WindowFrameLook].
func (l *Classic) WindowCloseRect(b paintengine2d.Rect) paintengine2d.Rect {
	return l.eng().WindowCloseRect(l, b)
}
