package widgets

import (
	"math"
	"strconv"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// WizardPage is one step of a Wizard.
type WizardPage struct {
	// Title names the step (the header, the step list); Subtitle says in
	// a line what it is for.
	Title    string
	Subtitle string
	Content  widget.Component
	// Complete reports whether the page may be left forward yet: Next
	// (and Finish) stay greyed until it does, as QWizardPage.isComplete.
	// Call Wizard.UpdateButtons when what it reads changes; the wizard
	// also asks it every frame. Nil: always complete.
	Complete func() bool
	// Validate runs when Next or Finish is pressed; an error keeps the
	// page and shows its message on it (QWizardPage.validatePage).
	Validate func() error
	// Skip reports whether the page is left out right now: a step that
	// depends on an earlier choice (GtkAssistant's forward function,
	// QWizard's nextId). It is asked as the user moves, so a choice made
	// on one page adds or drops the pages after it.
	Skip func() bool
	// Optional pages offer Skip, which moves on without validating.
	Optional bool
	// Commit makes Next an Apply that cannot be taken back: past this
	// page, Back no longer returns to it (QWizard's commit page,
	// GtkAssistant's confirm page).
	Commit bool
	// OnEnter runs as the page is shown.
	OnEnter func()
}

// WizardStyle is how a Wizard is laid out.
type WizardStyle uint8

const (
	// WizardAuto follows the look: WizardClassic under Windows and KDE
	// looks, WizardModern under GNOME and Mac looks.
	WizardAuto WizardStyle = iota
	// WizardClassic is the Windows wizard (Wizard 97's interior pages):
	// a header band with the step's title and subtitle, the page under
	// it, "< Back  Next >  Cancel" at the foot.
	WizardClassic
	// WizardModern is GNOME's assistant and the Mac's setup assistant: a
	// sidebar of the steps with the current one marked, the step's title
	// over the page, "Cancel … Back  Next" at the foot.
	WizardModern
)

// WizardSteps says whether the list of steps shows.
type WizardSteps uint8

const (
	// StepsAuto shows the steps in the modern layout only.
	StepsAuto WizardSteps = iota
	StepsShown
	StepsHidden
)

// Wizard walks the user through pages with Back, Next, Finish and Cancel —
// Qt's QWizard, GTK's GtkAssistant. A page can hold Next back until it is
// complete, refuse to be left with a message, be left out when an earlier
// choice makes it moot, or be skipped by the user when it is optional.
// Return presses Next (Finish on the last page) and Escape Cancel, as a
// dialog's default and cancel buttons; Alt+B, N, F, S and H press Back,
// Next, Finish, Skip and Help. Help shows when OnHelp is set.
type Wizard struct {
	widget.Base
	// Title is the wizard's name (what assistive technology announces).
	Title string
	// Style picks the layout; Steps whether the list of steps shows.
	Style WizardStyle
	Steps WizardSteps
	// OnFinish runs when Finish is pressed on a valid last page.
	OnFinish func()
	// OnCancel runs when Cancel or Escape is pressed; returning false keeps
	// the wizard going (to ask first). Nil cancels.
	OnCancel func() bool
	// OnHelp, when set, shows Help and is told which page asked.
	OnHelp func(page int)
	// OnPageChange is told the page shown changed.
	OnPageChange func(from, to int)

	pages   []*WizardPage
	cur     int
	history []int
	done    bool
	cancel  bool

	steps    *ListView
	stepMap  []int // list row -> page
	header   *wizardHeader
	title    *Label
	subtitle *Label
	holder   *wizardPageBox
	errLabel *Label
	buttons  *FlexBox
	sep      *Separator
	bHelp    *Button
	bSkip    *Button
	bBack    *Button
	bNext    *Button
	bFinish  *Button
	bCancel  *Button
	laidFor  WizardStyle
	side     paintengine2d.Rect
	// byAlt: the page is changing on an access key, whose character is
	// still to arrive as text; the focus moves once it has gone by.
	byAlt bool
}

// NewWizard is a wizard over pages, showing the first.
func NewWizard(title string, pages ...*WizardPage) *Wizard {
	w := &Wizard{Title: title, pages: pages}
	w.Init(w)
	w.steps = NewListView(0, func(i int) string { return w.stepText(i) }, nil)
	w.steps.Sidebar = true
	w.steps.DisableTypeAhead = true
	w.steps.Frameless = true
	w.steps.SetWantsFocus(false)
	w.steps.SetAccessibleName("Steps")
	w.header = &wizardHeader{wiz: w}
	w.header.Init(w.header)
	w.title = NewTitle("")
	w.subtitle = NewLabel("")
	w.subtitle.Wrap = true
	w.holder = &wizardPageBox{wiz: w}
	w.holder.Init(w.holder)
	w.errLabel = NewLabel("")
	w.errLabel.Wrap = true
	w.errLabel.SetVisible(false)
	w.bHelp = NewButton("Help", func() {
		if w.OnHelp != nil {
			w.OnHelp(w.cur)
		}
	})
	w.bSkip = NewButton("Skip", func() { w.SkipPage() })
	w.bBack = NewButton("Back", func() { w.Back() })
	w.bNext = NewButton("Next", func() { w.Next() })
	w.bFinish = NewButton("Finish", func() { w.Finish() })
	w.bCancel = NewButton("Cancel", func() { w.Cancel() })
	w.bNext.Primary, w.bFinish.Primary = true, true
	w.sep = NewSeparator()
	for _, c := range []widget.Component{w.steps, w.header, w.title, w.subtitle, w.holder, w.errLabel, w.sep} {
		w.Add(c)
	}
	w.buttons = NewRow().WithGap(8)
	w.Add(w.buttons)
	w.show(0, false)
	return w
}

// Pages are the wizard's pages.
func (w *Wizard) Pages() []*WizardPage { return w.pages }

// Current is the index of the page shown.
func (w *Wizard) Current() int { return w.cur }

// CurrentPage is the page shown.
func (w *Wizard) CurrentPage() *WizardPage {
	if w.cur < 0 || w.cur >= len(w.pages) {
		return nil
	}
	return w.pages[w.cur]
}

// Finished reports whether Finish completed the wizard; Cancelled whether
// it was cancelled.
func (w *Wizard) Finished() bool  { return w.done }
func (w *Wizard) Cancelled() bool { return w.cancel }

// History is the pages visited to get here, in order, which Back walks.
func (w *Wizard) History() []int { return append([]int(nil), w.history...) }

// ShownStyle is the layout in use: Style, or the look's choice for
// WizardAuto.
func (w *Wizard) ShownStyle() WizardStyle { return w.style() }

// style is the layout in use.
func (w *Wizard) style() WizardStyle {
	if w.Style != WizardAuto {
		return w.Style
	}
	if style.LookHint(w.Look(), style.HintDialogPrimaryFirst) == 1 {
		return WizardClassic
	}
	return WizardModern
}

func (w *Wizard) showSteps() bool {
	switch w.Steps {
	case StepsShown:
		return true
	case StepsHidden:
		return false
	}
	return w.style() == WizardModern
}

// skipped reports whether page i is left out now.
func (w *Wizard) skipped(i int) bool {
	p := w.pages[i]
	return p.Skip != nil && p.Skip()
}

// nextPage is the first page after from that is not left out, or -1.
func (w *Wizard) nextPage(from int) int {
	for i := from + 1; i < len(w.pages); i++ {
		if !w.skipped(i) {
			return i
		}
	}
	return -1
}

// complete reports whether the page shown may be left forward.
func (w *Wizard) complete() bool {
	p := w.CurrentPage()
	return p == nil || p.Complete == nil || p.Complete()
}

// Next validates the page shown and moves to the next one (finishing on
// the last); it reports whether it moved.
func (w *Wizard) Next() bool {
	if w.done || !w.complete() || !w.validate() {
		return false
	}
	n := w.nextPage(w.cur)
	if n < 0 {
		return w.finish()
	}
	if w.CurrentPage().Commit {
		// Past a commit page there is no way back to it or before it.
		w.history = nil
	} else {
		w.history = append(w.history, w.cur)
	}
	w.show(n, true)
	return true
}

// Back returns to the page shown before this one.
func (w *Wizard) Back() bool {
	if w.done || len(w.history) == 0 {
		return false
	}
	prev := w.history[len(w.history)-1]
	w.history = w.history[:len(w.history)-1]
	w.show(prev, true)
	return true
}

// SkipPage moves past an optional page without validating it.
func (w *Wizard) SkipPage() bool {
	p := w.CurrentPage()
	if w.done || p == nil || !p.Optional {
		return false
	}
	n := w.nextPage(w.cur)
	if n < 0 {
		return false
	}
	w.history = append(w.history, w.cur)
	w.show(n, true)
	return true
}

// Finish validates the page shown and, on the last page, finishes.
func (w *Wizard) Finish() bool {
	if w.done || !w.complete() || !w.validate() {
		return false
	}
	return w.finish()
}

func (w *Wizard) finish() bool {
	if w.nextPage(w.cur) >= 0 {
		return false
	}
	w.done = true
	w.UpdateButtons()
	if w.OnFinish != nil {
		w.OnFinish()
	}
	return true
}

// Cancel asks OnCancel and gives the wizard up.
func (w *Wizard) Cancel() bool {
	if w.done {
		return false
	}
	if w.OnCancel != nil && !w.OnCancel() {
		return false
	}
	w.cancel = true
	return true
}

// Restart goes back to the first page with no history, for another run.
func (w *Wizard) Restart() {
	w.history, w.done, w.cancel = nil, false, false
	first := 0
	if len(w.pages) > 0 && w.skipped(0) {
		first = max(w.nextPage(0), 0)
	}
	w.show(first, true)
}

// validate runs the page's Validate and shows its message.
func (w *Wizard) validate() bool {
	p := w.CurrentPage()
	if p == nil || p.Validate == nil {
		w.setError("")
		return true
	}
	if err := p.Validate(); err != nil {
		w.setError(err.Error())
		return false
	}
	w.setError("")
	return true
}

func (w *Wizard) setError(msg string) {
	w.errLabel.Text = msg
	w.errLabel.SetVisible(msg != "")
	w.RequestLayout()
	w.Invalidate()
}

// ErrorText is the message the last failed validation showed.
func (w *Wizard) ErrorText() string { return w.errLabel.Text }

// show puts page i up.
func (w *Wizard) show(i int, focus bool) {
	if len(w.pages) == 0 {
		return
	}
	i = min(max(i, 0), len(w.pages)-1)
	from := w.cur
	w.cur = i
	p := w.pages[i]
	w.holder.set(p.Content)
	w.title.Text = p.Title
	w.subtitle.Text = p.Subtitle
	w.setError("")
	if p.OnEnter != nil {
		p.OnEnter()
	}
	w.UpdateButtons()
	w.RequestLayout()
	w.Invalidate()
	if focus {
		if w.byAlt {
			// Alt+N's "n" is still on its way as text: a field focused
			// now would take it.
			widget.After(w, 60*time.Millisecond, w.focusPage)
		} else {
			w.focusPage()
		}
	}
	if from != i && w.OnPageChange != nil {
		w.OnPageChange(from, i)
	}
}

// focusPage puts the keyboard on the page's first control, else on the
// button that moves on.
func (w *Wizard) focusPage() {
	if widget.FocusFirstIn(w.holder) {
		return
	}
	if w.bFinish.Visible() {
		widget.FocusFirstIn(w.bFinish)
	} else {
		widget.FocusFirstIn(w.bNext)
	}
}

// stepText is row i of the list of steps.
func (w *Wizard) stepText(i int) string {
	if i < 0 || i >= len(w.stepMap) {
		return ""
	}
	p := w.pages[w.stepMap[i]]
	s := strconv.Itoa(i+1) + ".  " + p.Title
	if p.Optional {
		s += " (optional)"
	}
	return s
}

// UpdateButtons brings the buttons and the list of steps in step with the
// page: Back where there is history, Next or Finish as the page stands,
// Skip on an optional page. Call it when a page's Complete would answer
// differently (the wizard also does, every frame).
func (w *Wizard) UpdateButtons() {
	classic := w.style() == WizardClassic
	if classic {
		w.bBack.Text, w.bNext.Text = "< Back", "Next >"
	} else {
		w.bBack.Text, w.bNext.Text = "Back", "Next"
	}
	p := w.CurrentPage()
	if p != nil && p.Commit {
		w.bNext.Text = "Apply"
	}
	last := w.nextPage(w.cur) < 0
	ok := !w.done && w.complete()
	w.bBack.SetEnabled(!w.done && len(w.history) > 0)
	w.bNext.SetVisible(!last)
	w.bFinish.SetVisible(last)
	w.bNext.SetEnabled(ok)
	w.bFinish.SetEnabled(ok)
	w.bSkip.SetVisible(p != nil && p.Optional && !last)
	w.bHelp.SetVisible(w.OnHelp != nil)
	w.bCancel.SetEnabled(!w.done)
	// The list of steps: the pages not left out, the current one chosen.
	w.stepMap = w.stepMap[:0]
	sel := -1
	for i := range w.pages {
		if w.skipped(i) && i != w.cur {
			continue
		}
		if i == w.cur {
			sel = len(w.stepMap)
		}
		w.stepMap = append(w.stepMap, i)
	}
	if w.steps.Count != len(w.stepMap) || w.steps.Selected != sel {
		w.steps.Count = len(w.stepMap)
		w.steps.SetSelectedRows(nil)
		w.steps.Selected = sel
		w.steps.Invalidate()
	}
	w.arrangeButtons(classic)
}

// arrangeButtons orders the buttons for the layout: Windows' "Help …
// < Back  Next >  Cancel", GNOME's and the Mac's "Help  Cancel … Back
// Next".
func (w *Wizard) arrangeButtons(classic bool) {
	if w.laidFor == w.style()+1 && len(w.buttons.Children()) > 0 {
		return
	}
	w.laidFor = w.style() + 1
	w.buttons.ClearChildren()
	sp := NewSpacer()
	if classic {
		for _, c := range []widget.Component{w.bHelp, sp, w.bSkip, w.bBack, w.bNext, w.bFinish, w.bCancel} {
			w.buttons.Add(c)
		}
	} else {
		for _, c := range []widget.Component{w.bHelp, w.bCancel, sp, w.bSkip, w.bBack, w.bNext, w.bFinish} {
			w.buttons.Add(c)
		}
	}
	w.buttons.AddFlex(sp, 1)
}

// ---- layout ------------------------------------------------------------

func (w *Wizard) dip(v float32) float32 { return float32(math.Round(float64(style.Dip(w.Look(), v)))) }

func (w *Wizard) sideW() float32 { return w.dip(190) }

func (w *Wizard) pad() float32 {
	p := w.Look().Metrics().Pad
	if p <= 0 {
		p = w.dip(12)
	}
	return max(p, w.dip(12))
}

func (w *Wizard) Measure(c layout.Constraints) paintengine2d.Point {
	pad := w.pad()
	var cw, ch float32
	for _, p := range w.pages {
		if p.Content == nil {
			continue
		}
		sz := p.Content.Measure(layout.Loose(w.dip(640), w.dip(480)))
		cw, ch = max(cw, sz.X), max(ch, sz.Y)
	}
	bw := w.buttons.Measure(layout.Unbounded())
	width := max(cw+2*pad, bw.X+2*pad, w.dip(420))
	height := ch + 2*pad + bw.Y + 2*pad
	if w.style() == WizardClassic {
		height += w.header.height()
	} else {
		height += w.dip(56)
	}
	if w.showSteps() {
		width += w.sideW()
	}
	return c.Constrain(paintengine2d.Pt(width, height))
}

func (w *Wizard) Arrange(r paintengine2d.Rect) {
	w.SetBounds(r)
	w.UpdateButtons()
	b := w.LocalBounds()
	pad := w.pad()
	classic := w.style() == WizardClassic
	bh := w.buttons.Measure(layout.Loose(b.Dx(), b.Dy())).Y
	foot := bh + 2*pad
	x0 := float32(0)
	top := float32(0)
	w.header.SetVisible(classic)
	w.title.SetVisible(!classic)
	w.subtitle.SetVisible(!classic && w.subtitle.Text != "")
	if classic {
		hh := w.header.height()
		w.header.Arrange(paintengine2d.XYWH(0, 0, b.Dx(), hh))
		top = hh
	}
	if w.showSteps() {
		w.steps.SetVisible(true)
		sb := paintengine2d.XYWH(0, top, w.sideW(), b.Dy()-top)
		if classic {
			sb.Max.Y -= foot
		}
		// The column is the sidebar's colour from edge to edge (Paint);
		// the list starts a margin down, level with the page's title.
		w.side = sb
		w.steps.Arrange(paintengine2d.Rect{Min: paintengine2d.Pt(sb.Min.X, sb.Min.Y+pad), Max: sb.Max})
		x0 = w.sideW()
	} else {
		w.steps.SetVisible(false)
		w.steps.Arrange(paintengine2d.Rect{})
		w.side = paintengine2d.Rect{}
	}
	// The button row runs under the page (under everything, classic).
	footX := x0
	if classic {
		footX = 0
	}
	w.sep.Arrange(paintengine2d.XYWH(footX, b.Dy()-foot, b.Dx()-footX, max(w.dip(1), 1)))
	w.buttons.Arrange(paintengine2d.XYWH(footX+pad, b.Dy()-foot+pad, b.Dx()-footX-2*pad, bh))
	// The page: its title and subtitle over it (modern), its error under.
	y := top + pad
	inner := b.Dx() - x0 - 2*pad
	if !classic {
		th := w.title.Measure(layout.Loose(inner, b.Dy())).Y
		w.title.Arrange(paintengine2d.XYWH(x0+pad, y, inner, th))
		y += th + w.dip(4)
		if w.subtitle.Visible() {
			sh := w.subtitle.Measure(layout.Loose(inner, b.Dy())).Y
			w.subtitle.Arrange(paintengine2d.XYWH(x0+pad, y, inner, sh))
			y += sh
		}
		y += w.dip(10)
	}
	bottom := b.Dy() - foot - pad
	if w.errLabel.Visible() {
		eh := w.errLabel.Measure(layout.Loose(inner, b.Dy())).Y
		w.errLabel.Arrange(paintengine2d.XYWH(x0+pad, bottom-eh, inner, eh))
		bottom -= eh + w.dip(8)
	}
	w.holder.Arrange(paintengine2d.XYWH(x0+pad, y, inner, max(bottom-y, 0)))
}

func (w *Wizard) Paint(ctx *paintengine2d.Context) {
	lk := w.Look()
	p := lk.Palette()
	ctx.DrawRect(w.LocalBounds(), paintengine2d.Fill(p.Background))
	if !w.side.Empty() {
		ctx.DrawRect(w.side, paintengine2d.Fill(style.ViewBackgroundOf(lk, w.steps.State()|style.StateSidebar)))
	}
	w.errLabel.Color = p.Danger
	// Complete may read fields the user is typing in: keep Next honest.
	w.UpdateButtons()
}

// ---- keyboard ----------------------------------------------------------

// KeyPress: Return presses Next (or Finish), Escape Cancel, as a dialog's
// default and cancel buttons do. Keys a control takes first (Return in a
// text area, a list's arrows) never get here.
func (w *Wizard) KeyPress(e widget.KeyEvent) bool {
	if e.Mods.Ctrl() || e.Mods.Alt() {
		return false
	}
	switch e.Key {
	case platform.KeyReturn:
		if w.bFinish.Visible() {
			w.Finish()
		} else {
			w.Next()
		}
		return true
	case platform.KeyEscape:
		w.Cancel()
		return true
	}
	return false
}

// HandleAlt gives the buttons the Windows wizard's access keys: Alt+B
// Back, Alt+N Next (or Apply), Alt+F Finish, Alt+S Skip, Alt+H Help.
func (w *Wizard) HandleAlt(key platform.Key) bool {
	press := func(b *Button) bool {
		if !b.Visible() || !b.Enabled() || b.OnClick == nil {
			return false
		}
		w.byAlt = true
		b.OnClick()
		w.byAlt = false
		return true
	}
	switch key {
	case platform.KeyB:
		return press(w.bBack)
	case platform.KeyN, platform.KeyA:
		return press(w.bNext)
	case platform.KeyF:
		return press(w.bFinish)
	case platform.KeyS:
		return press(w.bSkip)
	case platform.KeyH:
		return press(w.bHelp)
	}
	return false
}

// ---- accessibility -----------------------------------------------------

// Describe: a dialog named by the wizard's title, saying which step it is
// on.
func (w *Wizard) Describe(n *a11y.Node) {
	n.Role = a11y.RoleDialog
	if n.Name == "" {
		n.Name = w.Title
	}
	if p := w.CurrentPage(); p != nil {
		step := 0
		for i, pg := range w.stepMap {
			if pg == w.cur {
				step = i + 1
			}
		}
		n.Description = "Step " + strconv.Itoa(step) + " of " + strconv.Itoa(len(w.stepMap)) + ": " + p.Title
	}
}

// ---- parts -------------------------------------------------------------

// wizardPageBox holds the page shown.
type wizardPageBox struct {
	widget.Base
	wiz     *Wizard
	content widget.Component
}

func (b *wizardPageBox) set(c widget.Component) {
	if b.content == c {
		return
	}
	if b.content != nil {
		b.Remove(b.content)
	}
	b.content = c
	if c != nil {
		b.Add(c)
	}
	b.RequestLayout()
}

func (b *wizardPageBox) Measure(c layout.Constraints) paintengine2d.Point {
	if b.content == nil {
		return c.Constrain(paintengine2d.Point{})
	}
	return b.content.Measure(c)
}

// Arrange gives the page its natural height at the top (a field stays a
// field's height); content that wants the room — a list, a text area in a
// column with a flexible child — measures as tall as it is given.
func (b *wizardPageBox) Arrange(r paintengine2d.Rect) {
	b.SetBounds(r)
	if b.content != nil {
		lb := b.LocalBounds()
		sz := b.content.Measure(layout.Constraints{MinW: lb.Dx(), MaxW: lb.Dx(), MaxH: lb.Dy()})
		b.content.Arrange(paintengine2d.XYWH(0, 0, lb.Dx(), min(max(sz.Y, 0), lb.Dy())))
	}
}

// Describe: the page, a group named by its title.
func (b *wizardPageBox) Describe(n *a11y.Node) {
	n.Role = a11y.RoleGroup
	if p := b.wiz.CurrentPage(); p != nil && n.Name == "" {
		n.Name = p.Title
		n.Description = p.Subtitle
	}
}

// wizardHeader is the classic layout's header band: the step's title in
// bold and its subtitle, on the field colour (Wizard 97's white band),
// over a rule.
type wizardHeader struct {
	widget.Base
	wiz *Wizard
}

func (h *wizardHeader) height() float32 {
	lk := h.Look()
	w := h.wiz
	bf := lk.BoldFont()
	pad := w.dip(12)
	return float32(math.Ceil(float64(pad + bf.Height() + w.dip(4) + lk.Font().Height() + pad)))
}

func (h *wizardHeader) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(c.MaxW, h.height()))
}

func (h *wizardHeader) Arrange(r paintengine2d.Rect) { h.SetBounds(r) }

func (h *wizardHeader) Paint(ctx *paintengine2d.Context) {
	lk := h.Look()
	p := lk.Palette()
	b := h.LocalBounds()
	bg := p.Field
	if bg.A == 0 {
		bg = p.Surface
	}
	ctx.DrawRect(b, paintengine2d.Fill(bg))
	lk.DrawSeparator(ctx, paintengine2d.XYWH(0, b.Dy()-max(h.wiz.dip(2), 2), b.Dx(), max(h.wiz.dip(2), 2)), false)
	pg := h.wiz.CurrentPage()
	if pg == nil {
		return
	}
	text := style.ReadableOn(bg, 4.5, p.Text)
	bf, f := lk.BoldFont(), lk.Font()
	pad := h.wiz.dip(12)
	x := h.wiz.dip(20)
	w := b.Dx() - x - pad
	bf.Draw(ctx, bf.Fit(pg.Title, w), paintengine2d.Pt(x, pad), text)
	if pg.Subtitle != "" {
		f.Draw(ctx, f.Fit(pg.Subtitle, w-h.wiz.dip(16)), paintengine2d.Pt(x+h.wiz.dip(16), pad+bf.Height()+h.wiz.dip(4)), text)
	}
}

// Describe: the header is the step's heading.
func (h *wizardHeader) Describe(n *a11y.Node) {
	n.Role = a11y.RoleHeading
	if pg := h.wiz.CurrentPage(); pg != nil {
		n.Name = pg.Title
		n.Description = pg.Subtitle
	}
}
