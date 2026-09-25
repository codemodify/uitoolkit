package tourapp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Page eight: the three things that are invisible until they are wrong.
//
// A window is a tree of accessible nodes whether or not anyone is
// reading it; a linter says whether a screen reader would find anything
// to trip over; Tab walks the same controls in the same order a screen
// reader lists them; and motion and scale are the desktop's to decide,
// not the app's. This page shows the tree of the window it is in — which
// means it shows itself, and any mistake on any other page of the tour
// shows up here the moment that page is open.

func init() {
	p := &tourPages[pageAccess]
	p.title = "What a screen reader sees"
	p.proof = "This window's accessibility tree, as it stands, with the linter's verdict on it — " +
		"plus the tab order, the motion preference and the display scale, which are the desktop's to set."
	p.try = "Press Tab a few times, then \"Read the window again\"."
	p.build = buildAccessPage
}

type accessPage struct {
	t     *tourState
	tree  *widgets.TreeView
	probs *widgets.ListView
	found []string
	order *widgets.TextArea
	facts *widgets.TextArea
	busy  *widgets.ProgressBar
	// motion is the animations switch; it shows the application's
	// preference, which another tour window may change.
	motion *widgets.Switch
	// scope shows the same controls at a scale of their own.
	scope     *widgets.ThemeScope
	scaleBox  *widgets.Panel
	scaleWant float32
}

var tourScales = []float32{1, 1.25, 1.5, 1.75, 2}

func buildAccessPage(t *tourState) widget.Component {
	p := &accessPage{t: t, scaleWant: 1.75}
	t.own(pageAccess, p)

	p.tree = widgets.NewTreeView()
	p.tree.SetAccessibleName("Accessibility tree")
	p.probs = widgets.NewListView(0, func(i int) string {
		if i < 0 || i >= len(p.found) {
			return ""
		}
		return p.found[i]
	}, nil)
	p.probs.SetAccessibleName("What the linter found")
	p.probs.RowHeight = 22

	read := widgets.NewButton("Read the window again", func() {
		p.read()
		p.note("Read " + strconv.Itoa(countNodes(p.t.win.AccessibleTree())) + " nodes off this window.")
	})
	read.Primary = true

	treePanel := widgets.NewPanel("The tree, as it stands", p.tree)
	treePanel.Content().AddFlex(p.tree, 1)
	probPanel := widgets.NewPanel("a11y.Check", p.probs)
	probPanel.Content().AddFlex(p.probs, 1)

	trees := widgets.NewSplitter(widgets.SplitRows, treePanel, probPanel)
	// The linter's list keeps room for two rows of its verdict in the
	// roomier packs too.
	trees.Ratio = 0.62

	// ---- the keyboard -------------------------------------------------------

	p.order = widgets.NewMonoTextView("", "")
	p.order.SetAccessibleName("Tab order")
	p.order.MinRows = 9
	keys := widgets.NewPanel("The tab order",
		tourNote("Tab and Shift+Tab walk exactly this list, and a screen reader announces the same "+
			"controls in the same order. Escape unwinds floating chrome in one order everywhere: "+
			"tooltip, then popup, then overlay."),
		p.order,
	)
	keys.Content().AddFlex(p.order, 1)
	keys.Content().Spec.Gap = 7

	// ---- motion -------------------------------------------------------------

	p.busy = widgets.NewBusyBar(0)
	p.busy.SetAccessibleName("A busy indicator")
	motion := widgets.NewSwitch("Animations", style.Animations(), func(on bool) {
		// Through the appearance, so every tour window's switch and
		// readout follow.
		p.t.apply(func(ap *style.Appearance) { ap.ReduceMotion = !on })
		if on {
			p.note("Animations on: the bar travels, hovers fade, the default button breathes.")
		} else {
			p.note("Reduced motion: every state change is instant and the bar stops where it is.")
		}
		p.refresh()
	})
	motion.SetAccessibleName("Animations")
	p.motion = motion
	hoverMe := widgets.NewButton("Hover me", nil)
	hoverMe.Tip = "With animations on the hover fades in; with them off it snaps"
	defaultish := widgets.NewButton("A default button", nil)
	defaultish.Primary = true
	motionPanel := widgets.NewPanel("Motion",
		motion,
		p.busy,
		widgets.NewRow(hoverMe, defaultish).WithGap(8),
		tourNote("The switch here is the process-wide preference; Settings writes it to look.json for "+
			"every app, UITK_ANIMATIONS=0 forces it off for one run, and the desktop's own "+
			"reduced-motion setting turns it off without anyone asking. Screenshots are taken with it "+
			"off so that no frame is caught halfway."),
	)
	motionPanel.Content().Spec.Gap = 8

	// ---- scale ---------------------------------------------------------------

	names := make([]string, len(tourScales))
	for i, s := range tourScales {
		names[i] = strconv.FormatFloat(float64(s), 'f', 2, 32) + "x"
	}
	scalePick := widgets.NewSegmented(names, 3, func(i int) {
		p.scaleWant = tourScales[i]
		p.applyScale()
		p.note("The panel beside is drawn at " + names[i] + "; this window is still at its own scale.")
	})
	scalePick.SetAccessibleName("Preview scale")
	p.scaleBox = widgets.NewPanel("At "+names[3], widgets.NewLabel(""))
	scalePanel := widgets.NewPanel("Fractional scale",
		tourRow("Draw at", scalePick),
		p.scaleBox,
		tourNote("Every length in a look is a design pixel multiplied by the display's scale, so 1.25 "+
			"and 1.75 are as ordinary as 2 — nothing here is a bitmap scaled up. A window takes its "+
			"scale from the output it is on, so dragging the tour to a second monitor with a different "+
			"scale re-bakes its look at the new one."),
		tourNote("UITK_SCALE (or GDK_SCALE, or QT_SCALE_FACTOR) pins every window of a run; without "+
			"one each window follows its own surface."),
	)
	scalePanel.Content().Spec.Gap = 8
	scalePanel.Content().AddFlex(p.scaleBox, 1)

	panel, facts := tourReadout("The window, measured")
	p.facts = facts

	// The tree wants height of its own, so it is not in the scroll view
	// with the rest: a splitter gives the user the say over the share.
	top := widgets.NewColumn(widgets.NewRow(read).WithGap(6), trees).WithGap(8)
	top.AddFlex(trees, 1)
	rest := tourScroll("Keyboard, motion and scale", widgets.NewColumn(keys, motionPanel, scalePanel).WithGap(10))
	stage := widgets.NewSplitter(widgets.SplitRows, top, rest)
	stage.Ratio = 0.5

	p.applyScale()
	// A busy indicator needs the loop to wake; without this it would only
	// move when something else asked for a frame.
	t.win.RequestAnim(33 * time.Millisecond)
	t.onClose(func() { t.win.RequestAnim(0) })

	p.read()
	// Read again once this page has a box: a tree read before layout has
	// no bounds in it, and the linter rightly complains about every one
	// of them.
	return tourStage(newAfterLaidOut(stage, t.a.Post, func() {
		if !t.win.Closed() {
			p.read()
		}
	}), panel)
}

// applyScale redraws the preview panel at the chosen scale. A look is
// baked at a scale, so the preview is the same look built again rather
// than the same one stretched.
func (p *accessPage) applyScale() {
	look := style.WithScale(p.t.win.Look(), p.scaleWant/tourScale(p.t.win))
	child := skinPreviewTree()
	if p.scope == nil {
		p.scope = widgets.NewThemeScope(look, child)
		p.scaleBox.Content().ClearChildren()
		p.scaleBox.Content().AddFlex(p.scope, 1)
	} else {
		p.scope.SetChild(child)
		p.scope.SetTheme(look)
	}
	p.scaleBox.Title = "At " + strconv.FormatFloat(float64(p.scaleWant), 'f', 2, 32) + "x"
	p.scaleBox.RequestLayout()
	p.refresh()
}

// read rebuilds everything this page shows about the window it is in.
func (p *accessPage) read() {
	win := p.t.win
	root := win.AccessibleTree()

	p.tree.SetRoots([]*widgets.TreeNode{a11yTreeNode(root)})
	p.tree.Invalidate()

	p.found = p.found[:0]
	for _, pr := range a11y.Check(root) {
		p.found = append(p.found, pr.String())
	}
	if len(p.found) == 0 {
		p.found = append(p.found, "Nothing — every control has a name, a box and an id of its own.")
	}
	p.probs.Count = len(p.found)
	p.probs.Invalidate()

	var b strings.Builder
	list := widget.Focusables(win.Content())
	for i, c := range list {
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". ")
		b.WriteString(focusName(c))
		b.WriteByte('\n')
	}
	if len(list) == 0 {
		b.WriteString("nothing on this page takes the focus\n")
	}
	p.order.SetText(b.String())
	p.refresh()
}

func (p *accessPage) note(s string) {
	p.t.note(s)
	p.refresh()
}

func (p *accessPage) refresh() {
	if p.facts == nil {
		return
	}
	win := p.t.win
	if p.motion != nil && p.motion.On != style.Animations() {
		p.motion.On = style.Animations()
		p.motion.Invalidate()
	}
	root := win.AccessibleTree()
	focused := "nothing has the focus"
	root.Walk(func(n *a11y.Node) bool {
		if n.State.Has(a11y.StateFocused) {
			focused = n.Role.String() + " " + strconv.Quote(n.Name)
			return false
		}
		return true
	})
	bad := len(p.found)
	if bad == 1 && strings.HasPrefix(p.found[0], "Nothing") {
		bad = 0
	}
	p.facts.SetText(tourFacts(
		[2]string{"nodes", strconv.Itoa(countNodes(root))},
		[2]string{"problems", strconv.Itoa(bad)},
		[2]string{"focused", focused},
		[2]string{"focusable", strconv.Itoa(len(widget.Focusables(win.Content())))},
		[2]string{"", ""},
		[2]string{"buttons", strconv.Itoa(countRole(root, a11y.RoleButton))},
		[2]string{"tabs", strconv.Itoa(countRole(root, a11y.RoleTab))},
		[2]string{"lists", strconv.Itoa(countRole(root, a11y.RoleList))},
		[2]string{"fields", strconv.Itoa(countRole(root, a11y.RoleTextField))},
		[2]string{"", ""},
		[2]string{"animations", yesNo(style.Animations())},
		[2]string{"desktop asks", "reduced motion: " + yesNo(style.DesktopReducesMotion())},
		[2]string{"", ""},
		[2]string{"window scale", strconv.FormatFloat(float64(win.Scale()), 'f', 2, 32)},
		[2]string{"app scale", strconv.FormatFloat(float64(p.t.a.Scale()), 'f', 2, 32)},
		[2]string{"preview at", strconv.FormatFloat(float64(p.scaleWant), 'f', 2, 32)},
		[2]string{"", ""},
		[2]string{"bridge", "AT-SPI2 on Linux, on while a screen reader runs"},
		[2]string{"force it on", "UITK_A11Y=1"},
	))
}

// a11yTreeNode mirrors one accessibility node into a tree row, so the
// tree the toolkit publishes can be read in the app that publishes it.
func a11yTreeNode(n *a11y.Node) *widgets.TreeNode {
	if n == nil {
		return widgets.NewTreeNode("(no tree)")
	}
	label := n.Role.String()
	if n.Name != "" {
		label += "  " + strconv.Quote(n.Name)
	}
	if n.Value != "" {
		label += "  = " + strconv.Quote(clip(n.Value, 28))
	}
	if s := stateNames(n.State); s != "" {
		label += "  [" + s + "]"
	}
	kids := make([]*widgets.TreeNode, 0, len(n.Children))
	for _, c := range n.Children {
		kids = append(kids, a11yTreeNode(c))
	}
	return widgets.NewTreeNode(label, kids...)
}

var a11yStateNames = []struct {
	s    a11y.State
	name string
}{
	{a11y.StateFocusable, "focusable"}, {a11y.StateFocused, "focused"},
	{a11y.StateDisabled, "disabled"}, {a11y.StateCheckable, "checkable"},
	{a11y.StateChecked, "checked"}, {a11y.StateMixed, "mixed"},
	{a11y.StatePressed, "pressed"}, {a11y.StateSelectable, "selectable"},
	{a11y.StateSelected, "selected"}, {a11y.StateExpandable, "expandable"},
	{a11y.StateExpanded, "expanded"}, {a11y.StateEditable, "editable"},
	{a11y.StateReadOnly, "read-only"}, {a11y.StateMultiLine, "multi-line"},
	{a11y.StateMultiSelectable, "multi-selectable"}, {a11y.StateModal, "modal"},
	{a11y.StateDefault, "default"}, {a11y.StateHasPopup, "has-popup"},
	{a11y.StateOffscreen, "offscreen"}, {a11y.StateHorizontal, "horizontal"},
	{a11y.StateVertical, "vertical"},
}

func stateNames(s a11y.State) string {
	var out []string
	for _, x := range a11yStateNames {
		if s.Has(x.s) {
			out = append(out, x.name)
		}
	}
	return strings.Join(out, " ")
}

func countNodes(n *a11y.Node) int {
	c := 0
	n.Walk(func(*a11y.Node) bool { c++; return true })
	return c
}

func countRole(n *a11y.Node, r a11y.Role) int {
	c := 0
	n.Walk(func(x *a11y.Node) bool {
		if x.Role == r {
			c++
		}
		return true
	})
	return c
}

// focusName is what the tab order calls a control: the name and role a
// screen reader would say. It asks the widget to describe itself rather
// than reading the name the app set, because a stock control names
// itself from its own text and has no name set on it at all — and an
// control that describes itself as nothing is worth seeing here.
func focusName(c widget.Component) string {
	var n a11y.Node
	if d, ok := c.(interface{ Describe(*a11y.Node) }); ok {
		d.Describe(&n)
	}
	if n.Name == "" {
		if g, ok := c.(interface{ AccessibleName() string }); ok {
			n.Name = g.AccessibleName()
		}
	}
	if n.Name == "" {
		return goTypeName(c) + "  (" + n.Role.String() + ", unnamed)"
	}
	return clip(n.Name, 44) + "  (" + n.Role.String() + ")"
}

func goTypeName(v any) string {
	s := strings.TrimPrefix(fmt.Sprintf("%T", v), "*")
	if i := strings.LastIndexByte(s, '.'); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
