package widgets

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/widget"
)

// How each widget describes itself to assistive technology (see package
// a11y). Views whose items are not components list them as item nodes.

func tipDescription(n *a11y.Node, tip string) {
	if n.Description == "" {
		n.Description = tip
	}
}

func nameOr(n *a11y.Node, s string) {
	if n.Name == "" {
		n.Name = s
	}
}

func rangeNode(n *a11y.Node, min, max, now, step float64) {
	n.HasRange, n.Min, n.Max, n.Now, n.Step = true, min, max, now, step
}

// item builds the node of item i of view c, with a local box.
func item(c widget.Component, i int, role a11y.Role, name string, local paintengine2d.Rect) *a11y.Node {
	n := &a11y.Node{ID: widget.ItemID(c, i), Role: role, Name: name, Bounds: widget.LocalToWindow(c, local)}
	if !local.Overlaps(c.LocalBounds()) {
		n.State |= a11y.StateOffscreen
	}
	return n
}

// ---- controls ----------------------------------------------------------------

func (b *Button) Describe(n *a11y.Node) {
	n.Role = a11y.RoleButton
	nameOr(n, b.Text)
	tipDescription(n, b.Tip)
	if b.Primary {
		n.State |= a11y.StateDefault
	}
	n.Actions = n.Actions.With(a11y.ActionDefault)
}

func (b *Button) AccessibleAction(item int, a a11y.Action) bool {
	if a != a11y.ActionDefault || !b.Enabled() || b.OnClick == nil {
		return false
	}
	b.OnClick()
	return true
}

func (c *Checkbox) Describe(n *a11y.Node) {
	n.Role = a11y.RoleCheckBox
	nameOr(n, c.Text)
	n.State |= a11y.StateCheckable
	if c.Checked {
		n.State |= a11y.StateChecked
	}
	n.Actions = n.Actions.With(a11y.ActionDefault)
}

func (c *Checkbox) AccessibleAction(item int, a a11y.Action) bool {
	if a != a11y.ActionDefault || !c.Enabled() {
		return false
	}
	c.Checked = !c.Checked
	c.Invalidate()
	if c.OnChange != nil {
		c.OnChange(c.Checked)
	}
	return true
}

func (r *RadioButton) Describe(n *a11y.Node) {
	n.Role = a11y.RoleRadioButton
	nameOr(n, r.Text)
	n.State |= a11y.StateCheckable
	if r.Selected {
		n.State |= a11y.StateChecked
	}
	if r.group != nil {
		n.Index, n.Count = r.index+1, len(r.group.buttons)
	}
	n.Actions = n.Actions.With(a11y.ActionDefault)
}

func (s *Switch) Describe(n *a11y.Node) {
	n.Role = a11y.RoleSwitch
	nameOr(n, s.Text)
	n.State |= a11y.StateCheckable
	if s.On {
		n.State |= a11y.StateChecked
	}
	n.Actions = n.Actions.With(a11y.ActionDefault)
}

func (s *Switch) AccessibleAction(item int, a a11y.Action) bool {
	if a != a11y.ActionDefault || !s.Enabled() {
		return false
	}
	s.SetOn(!s.On)
	if s.OnChange != nil {
		s.OnChange(s.On)
	}
	return true
}

func (l *Label) Describe(n *a11y.Node) {
	n.Role = a11y.RoleLabel
	if l.Title {
		n.Role = a11y.RoleHeading
	}
	nameOr(n, l.Text)
}

func (t *TextField) Describe(n *a11y.Node) {
	n.Role = a11y.RoleTextField
	n.State |= a11y.StateEditable
	if t.Password {
		n.Role = a11y.RolePasswordField
	} else {
		n.Value = t.Text
	}
	if cb, ok := t.Parent().(*ComboBox); ok && n.Name == "" {
		// An editable combo box's entry is known by the box's name.
		n.Name = cb.AccessibleName()
	}
	if n.Name == "" {
		// A placeholder stands in for a missing label, as browsers do.
		n.Name = t.Placeholder
	}
	n.Caret, n.SelStart, n.SelEnd = t.caret, min(t.selA, t.selB), max(t.selA, t.selB)
}

func (t *TextArea) Describe(n *a11y.Node) {
	n.Role = a11y.RoleTextArea
	n.State |= a11y.StateMultiLine
	if t.ReadOnly {
		n.State |= a11y.StateReadOnly
	} else {
		n.State |= a11y.StateEditable
	}
	n.Value = t.Text
	if n.Name == "" {
		n.Name = t.Placeholder
	}
	n.Caret, n.SelStart, n.SelEnd = t.caret, min(t.selA, t.selB), max(t.selA, t.selB)
}

func (c *ComboBox) Describe(n *a11y.Node) {
	n.Role = a11y.RoleComboBox
	n.State |= a11y.StateHasPopup | a11y.StateExpandable
	if c.open {
		n.State |= a11y.StateExpanded
	}
	n.Value = c.Text()
	if c.field != nil {
		n.State |= a11y.StateEditable
		if c.field.Focused() {
			n.State |= a11y.StateFocused | a11y.StateFocusable
		}
	}
	if n.Name == "" {
		n.Name = c.Placeholder
	}
	n.Actions = n.Actions.With(a11y.ActionDefault).With(a11y.ActionShowMenu)
}

func (f *NumberField) Describe(n *a11y.Node) {
	n.Role = a11y.RoleSpinButton
	n.State |= a11y.StateEditable
	rangeNode(n, f.Min, f.Max, f.Value, f.Step)
	n.Value = strconv.FormatFloat(f.Value, 'f', max(f.Decimals, 0), 64)
	tipDescription(n, f.Tip)
	// The inner field takes the keyboard; the spin button is what has it.
	if f.field != nil && f.field.Focused() {
		n.State |= a11y.StateFocused | a11y.StateFocusable
	}
	n.Actions = n.Actions.With(a11y.ActionIncrement).With(a11y.ActionDecrement)
}

// AccessibleLeaf: the spin button's text field is part of it.
func (f *NumberField) AccessibleLeaf() bool { return true }

func (s *Slider) Describe(n *a11y.Node) {
	n.Role = a11y.RoleSlider
	n.State |= a11y.StateHorizontal
	step := float64(s.Max-s.Min) / 100
	rangeNode(n, float64(s.Min), float64(s.Max), float64(s.Value), step)
	n.Value = strconv.FormatFloat(float64(s.Value), 'f', -1, 32)
	n.Actions = n.Actions.With(a11y.ActionIncrement).With(a11y.ActionDecrement)
}

func (p *ProgressBar) Describe(n *a11y.Node) {
	n.Role = a11y.RoleProgressBar
	if p.Indeterminate {
		return // busy: no reading
	}
	rangeNode(n, 0, 1, float64(p.Value), 0)
	n.Value = fmt.Sprintf("%.0f%%", p.Value*100)
}

func (c *ColorButton) Describe(n *a11y.Node) {
	n.Role = a11y.RoleButton
	n.State |= a11y.StateHasPopup
	r, g, b := c.Color.R, c.Color.G, c.Color.B
	n.Value = fmt.Sprintf("#%02x%02x%02x", int(r*255+0.5), int(g*255+0.5), int(b*255+0.5))
	nameOr(n, "Colour")
	n.Actions = n.Actions.With(a11y.ActionDefault)
}

func (d *DateField) Describe(n *a11y.Node) {
	n.Role = a11y.RoleComboBox
	n.State |= a11y.StateHasPopup | a11y.StateExpandable
	if d.open {
		n.State |= a11y.StateExpanded
	}
	f := d.Format
	if f == "" {
		f = "2006-01-02"
	}
	if !d.Value.IsZero() {
		n.Value = d.Value.Format(f)
	}
}

func (c *Calendar) Describe(n *a11y.Node) {
	n.Role = a11y.RoleCalendar
	if !c.Selected.IsZero() {
		n.Value = c.Selected.Format("Monday, 2 January 2006")
	}
}

func (p *Picture) Describe(n *a11y.Node) { n.Role = a11y.RoleImage }

// ---- containers ----------------------------------------------------------------

func (p *Panel) Describe(n *a11y.Node) {
	n.Role = a11y.RoleGroup
	if p.Window {
		n.Role = a11y.RoleDialog
	}
	nameOr(n, p.Title)
}

func (o *Overlay) Describe(n *a11y.Node) {
	n.Role = a11y.RolePane
	if o.Modal {
		n.Role = a11y.RoleDialog
		n.State |= a11y.StateModal
	}
}

func (m *MessageBox) Describe(n *a11y.Node) {
	n.Role = a11y.RoleAlert
	nameOr(n, m.opts.Title)
	if n.Description == "" {
		n.Description = m.opts.Message
	}
}

func (s *ScrollView) Describe(n *a11y.Node) { n.Role = a11y.RoleScrollPane }

func (s *Splitter) Describe(n *a11y.Node) {
	n.Role = a11y.RoleSplitter
	if s.Vertical {
		n.State |= a11y.StateVertical
	} else {
		n.State |= a11y.StateHorizontal
	}
	rangeNode(n, 0, 1, float64(s.Ratio), 0.05)
}

func (e *Expander) Describe(n *a11y.Node) {
	n.Role = a11y.RoleToggleButton
	nameOr(n, e.Title)
	n.State |= a11y.StateExpandable
	if e.Expanded {
		n.State |= a11y.StateExpanded
	}
	n.Actions = n.Actions.With(a11y.ActionDefault)
}

func (s *StatusBar) Describe(n *a11y.Node) { n.Role = a11y.RoleStatusBar }

func (s *StatusBar) AccessibleItems() []*a11y.Node {
	var out []*a11y.Node
	for i, p := range s.parts {
		if p == "" {
			continue
		}
		out = append(out, item(s, i, a11y.RoleLabel, p, s.LocalBounds()))
	}
	return out
}

// ---- item views ----------------------------------------------------------------

func (l *ListView) Describe(n *a11y.Node) {
	n.Role = a11y.RoleList
	if l.Mode != SelectSingle {
		n.State |= a11y.StateMultiSelectable
	}
}

func (l *ListView) AccessibleItems() []*a11y.Node {
	out := make([]*a11y.Node, 0, l.Count)
	in := l.inner().Min
	for i := 0; i < l.Count; i++ {
		text := ""
		if l.ItemText != nil {
			text = l.ItemText(i)
		}
		n := item(l, i, a11y.RoleListItem, text, l.rowRect(i).Translate(in))
		n.State |= a11y.StateSelectable
		if l.IsSelected(i) {
			n.State |= a11y.StateSelected
		}
		n.Index, n.Count = i+1, l.Count
		n.Actions = n.Actions.With(a11y.ActionDefault).With(a11y.ActionScrollIntoView)
		out = append(out, n)
	}
	return out
}

func (l *ListView) AccessibleAction(i int, a a11y.Action) bool {
	if i < 0 || i >= l.Count || !l.Enabled() {
		return false
	}
	switch a {
	case a11y.ActionDefault:
		l.navigate(i, 0)
		return true
	case a11y.ActionScrollIntoView:
		l.ensureVisible(i)
		return true
	}
	return false
}

func (t *TreeView) Describe(n *a11y.Node) { n.Role = a11y.RoleTree }

// AccessibleItems: tree items nest as their nodes do; the collapsed
// branches' children are left out, as GTK and Qt do.
func (t *TreeView) AccessibleItems() []*a11y.Node {
	rows := t.flatten()
	in := t.inner().Min
	rh := t.rowH()
	index := make(map[*TreeNode]int, len(rows))
	for i, r := range rows {
		index[r.node] = i
	}
	var walk func(nodes []*TreeNode, level int) []*a11y.Node
	walk = func(nodes []*TreeNode, level int) []*a11y.Node {
		out := make([]*a11y.Node, 0, len(nodes))
		for k, tn := range nodes {
			i, shown := index[tn]
			if !shown {
				continue
			}
			local := paintengine2d.XYWH(in.X, in.Y+float32(i)*rh-t.OffsetY, t.rowsW(), rh)
			n := item(t, i, a11y.RoleTreeItem, tn.Label, local)
			n.Level, n.Index, n.Count = level, k+1, len(nodes)
			n.State |= a11y.StateSelectable
			if tn == t.Selected {
				n.State |= a11y.StateSelected
			}
			n.Actions = n.Actions.With(a11y.ActionDefault).With(a11y.ActionScrollIntoView)
			if len(tn.Children) > 0 {
				n.State |= a11y.StateExpandable
				if tn.Expanded {
					n.State |= a11y.StateExpanded
					n.Actions = n.Actions.With(a11y.ActionCollapse)
					n.Children = walk(tn.Children, level+1)
				} else {
					n.Actions = n.Actions.With(a11y.ActionExpand)
				}
			}
			out = append(out, n)
		}
		return out
	}
	return walk(t.Roots, 1)
}

func (t *TreeView) AccessibleAction(i int, a a11y.Action) bool {
	rows := t.flatten()
	if i < 0 || i >= len(rows) || !t.Enabled() {
		return false
	}
	n := rows[i].node
	switch a {
	case a11y.ActionDefault:
		t.selectNode(n)
		return true
	case a11y.ActionScrollIntoView:
		t.ensureVisible(n)
		return true
	case a11y.ActionExpand, a11y.ActionCollapse:
		want := a == a11y.ActionExpand
		if len(n.Children) == 0 || n.Expanded == want {
			return false
		}
		n.Expanded = want
		t.dropFlat()
		if t.OnToggle != nil {
			t.OnToggle(n)
		}
		t.Invalidate()
		return true
	}
	return false
}

func (t *TableView) Describe(n *a11y.Node) {
	n.Role = a11y.RoleTable
	if t.Mode != SelectSingle {
		n.State |= a11y.StateMultiSelectable
	}
}

// AccessibleItems: the column headers, then one row per record with its
// cells (item ids: headers first, then rows, then cells).
func (t *TableView) AccessibleItems() []*a11y.Node {
	widths := t.colWidths()
	in := t.inner().Min
	cols := len(t.Columns)
	out := make([]*a11y.Node, 0, cols+t.RowCount)
	x := in.X
	for c, col := range t.Columns {
		w := float32(0)
		if c < len(widths) {
			w = widths[c]
		}
		out = append(out, item(t, c, a11y.RoleColumnHeader, col.Title, paintengine2d.XYWH(x, in.Y, w, t.headerH())))
		x += w
	}
	for r := 0; r < t.RowCount; r++ {
		rr := t.rowRect(r).Translate(in)
		row := item(t, cols+r, a11y.RoleRow, "", rr)
		row.Index, row.Count = r+1, t.RowCount
		row.State |= a11y.StateSelectable
		if t.IsSelected(r) {
			row.State |= a11y.StateSelected
		}
		row.Actions = row.Actions.With(a11y.ActionDefault).With(a11y.ActionScrollIntoView)
		cx := rr.Min.X
		var said []string
		for c := 0; c < cols; c++ {
			w := float32(0)
			if c < len(widths) {
				w = widths[c]
			}
			text := ""
			if t.CellText != nil {
				text = t.CellText(r, c)
			}
			cell := item(t, cols+t.RowCount+r*cols+c, a11y.RoleCell, text, paintengine2d.XYWH(cx, rr.Min.Y, w, rr.Dy()))
			cell.Index, cell.Count = c+1, cols
			row.Children = append(row.Children, cell)
			cx += w
			if text != "" {
				said = append(said, text)
			}
		}
		// A row reads as its cells, as a screen reader speaks a row.
		row.Name = strings.Join(said, ", ")
		out = append(out, row)
	}
	return out
}

func (t *TableView) AccessibleAction(i int, a a11y.Action) bool {
	r := i - len(t.Columns)
	if r < 0 || r >= t.RowCount || !t.Enabled() {
		return false
	}
	switch a {
	case a11y.ActionDefault:
		t.navigate(r, 0)
		return true
	case a11y.ActionScrollIntoView:
		t.ensureVisible(r)
		return true
	}
	return false
}

func (l *CardList) Describe(n *a11y.Node) {
	n.Role = a11y.RoleList
	if l.Mode != SelectSingle {
		n.State |= a11y.StateMultiSelectable
	}
}

func (l *CardList) AccessibleItems() []*a11y.Node {
	out := make([]*a11y.Node, 0, l.Count)
	for i := 0; i < l.Count; i++ {
		cc := l.cardAt(i)
		n := item(l, i, a11y.RoleListItem, cc.Title, l.rowRect(i))
		n.Description = cc.Subtitle
		n.Index, n.Count = i+1, l.Count
		n.State |= a11y.StateSelectable
		if l.IsSelected(i) {
			n.State |= a11y.StateSelected
		}
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

// ---- bars ----------------------------------------------------------------------

func (t *TabBar) Describe(n *a11y.Node) { n.Role = a11y.RoleTabList }

func (t *TabBar) AccessibleItems() []*a11y.Node {
	rects := t.tabRects()
	out := make([]*a11y.Node, 0, len(t.Titles))
	for i, title := range t.Titles {
		var r paintengine2d.Rect
		if i < len(rects) {
			r = rects[i]
		}
		n := item(t, i, a11y.RoleTab, title, r)
		n.Index, n.Count = i+1, len(t.Titles)
		n.State |= a11y.StateSelectable
		if i == t.Selected {
			n.State |= a11y.StateSelected
		}
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

func (t *TabBar) AccessibleAction(i int, a a11y.Action) bool {
	if a != a11y.ActionDefault || i < 0 || i >= len(t.Titles) || !t.Enabled() {
		return false
	}
	t.Select(i)
	return true
}

func (p *TabPage) Describe(n *a11y.Node) { n.Role = a11y.RoleTabPanel }

func (s *Segmented) Describe(n *a11y.Node) { n.Role = a11y.RoleTabList }

func (s *Segmented) AccessibleItems() []*a11y.Node {
	out := make([]*a11y.Node, 0, len(s.Segments))
	for i, seg := range s.Segments {
		n := item(s, i, a11y.RoleTab, seg, s.segRect(i))
		n.Index, n.Count = i+1, len(s.Segments)
		n.State |= a11y.StateSelectable
		if i == s.Selected {
			n.State |= a11y.StateSelected
		}
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

func (s *Segmented) AccessibleAction(i int, a a11y.Action) bool {
	if a != a11y.ActionDefault || i < 0 || i >= len(s.Segments) || !s.Enabled() {
		return false
	}
	s.choose(i)
	return true
}

func (t *ToolBar) Describe(n *a11y.Node) { n.Role = a11y.RoleToolBar }

func (t *ToolBar) AccessibleItems() []*a11y.Node {
	rects := t.itemRects()
	var out []*a11y.Node
	for i, it := range t.items {
		if it == nil {
			continue
		}
		var r paintengine2d.Rect
		if i < len(rects) {
			r = rects[i]
		}
		if it.Sep {
			out = append(out, item(t, i, a11y.RoleSeparator, "", r))
			continue
		}
		role := a11y.RoleButton
		if it.Toggle {
			role = a11y.RoleToggleButton
		}
		name := it.Text
		if name == "" {
			name = it.Tip // an icon-only tool is known by its tooltip
		}
		if name == "" {
			name = it.Icon.Label() // or by what its icon stands for
		}
		n := item(t, i, role, name, r)
		if it.Tip != name {
			n.Description = it.Tip
		}
		if it.Toggle && it.Down {
			n.State |= a11y.StatePressed
		}
		if it.Disabled {
			n.State |= a11y.StateDisabled
		}
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

func (t *ToolBar) AccessibleAction(i int, a a11y.Action) bool {
	if a != a11y.ActionDefault || i < 0 || i >= len(t.items) || t.items[i] == nil || t.items[i].Disabled || t.items[i].Sep {
		return false
	}
	t.activate(i)
	return true
}

// ---- menus ---------------------------------------------------------------------

func (m *MenuBar) Describe(n *a11y.Node) { n.Role = a11y.RoleMenuBar }

func (m *MenuBar) AccessibleItems() []*a11y.Node {
	rects := m.titleRects()
	out := make([]*a11y.Node, 0, len(m.menus))
	for i, menu := range m.menus {
		if menu == nil {
			continue
		}
		n := item(m, i, a11y.RoleMenuItem, widget.PlainText(menu.Title), rects[i])
		n.State |= a11y.StateHasPopup
		if i == m.open {
			n.State |= a11y.StateExpanded
		}
		n.Index, n.Count = i+1, len(m.menus)
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

func (m *MenuBar) AccessibleAction(i int, a a11y.Action) bool {
	if a != a11y.ActionDefault || i < 0 || i >= len(m.menus) {
		return false
	}
	m.Open(i)
	return true
}

func (p *PopupMenu) Describe(n *a11y.Node) { n.Role = a11y.RoleMenu }

func (p *PopupMenu) AccessibleItems() []*a11y.Node {
	out := make([]*a11y.Node, 0, len(p.Items))
	for i, it := range p.Items {
		if it == nil {
			continue
		}
		if it.Separator {
			out = append(out, item(p, i, a11y.RoleSeparator, "", p.rowBounds(i)))
			continue
		}
		role := a11y.RoleMenuItem
		switch {
		case it.isRadio():
			role = a11y.RoleRadioMenuItem
		case it.Checkable:
			role = a11y.RoleCheckMenuItem
		}
		n := item(p, i, role, widget.PlainText(it.Text), p.rowBounds(i))
		n.Shortcut = it.Shortcut
		n.Index, n.Count = i+1, len(p.Items)
		if role != a11y.RoleMenuItem {
			n.State |= a11y.StateCheckable
			if it.Checked {
				n.State |= a11y.StateChecked
			}
		}
		if it.HasSubmenu() {
			n.State |= a11y.StateHasPopup
		}
		if it.Disabled {
			n.State |= a11y.StateDisabled
		}
		if i == p.focus || i == p.hover {
			n.State |= a11y.StateSelected
		}
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

// ---- the item with the keyboard focus ------------------------------------------

func (l *ListView) AccessibleFocusItem() int {
	if l.Selected >= 0 && l.Selected < l.Count {
		return l.Selected
	}
	return -1
}

func (t *TreeView) AccessibleFocusItem() int {
	if t.Selected == nil {
		return -1
	}
	return t.indexOf(t.Selected)
}

func (t *TableView) AccessibleFocusItem() int {
	if t.Selected >= 0 && t.Selected < t.RowCount {
		return len(t.Columns) + t.Selected
	}
	return -1
}

func (l *CardList) AccessibleFocusItem() int {
	if l.Selected >= 0 && l.Selected < l.Count {
		return l.Selected
	}
	return -1
}

func (t *TabBar) AccessibleFocusItem() int {
	if t.Selected >= 0 && t.Selected < len(t.Titles) {
		return t.Selected
	}
	return -1
}

func (s *Segmented) AccessibleFocusItem() int {
	if s.Selected >= 0 && s.Selected < len(s.Segments) {
		return s.Selected
	}
	return -1
}

func (t *ToolBar) AccessibleFocusItem() int {
	if t.keyNav && t.focus >= 0 && t.focus < len(t.items) {
		return t.focus
	}
	return -1
}

func (m *MenuBar) AccessibleFocusItem() int {
	if m.keyNav && m.focus >= 0 && m.focus < len(m.menus) {
		return m.focus
	}
	return -1
}

func (p *PopupMenu) AccessibleFocusItem() int {
	if p.focus >= 0 && p.focus < len(p.Items) {
		return p.focus
	}
	if p.hover >= 0 && p.hover < len(p.Items) {
		return p.hover
	}
	return -1
}

// AccessibleItemCount bounds the work of finding a focused item.
func (l *ListView) AccessibleItemCount() int  { return l.Count }
func (t *TableView) AccessibleItemCount() int { return t.RowCount * (len(t.Columns) + 1) }
func (l *CardList) AccessibleItemCount() int  { return l.Count }
func (t *TreeView) AccessibleItemCount() int  { return len(t.flatten()) }

// AccessibleSetText replaces the field's text (AT-SPI's EditableText),
// as typing would, and puts the caret at its end.
func (t *TextField) AccessibleSetText(s string) bool {
	if !t.Enabled() || (t.Accept != nil && !t.Accept(s)) {
		return false
	}
	t.caret = runeCount(s)
	t.SetText(s)
	return true
}

// AccessibleSetText replaces the area's text unless it is read-only.
func (t *TextArea) AccessibleSetText(s string) bool {
	if !t.Enabled() || t.ReadOnly || (t.Accept != nil && !t.Accept(s)) {
		return false
	}
	t.caret = runeCount(s)
	t.SetText(s)
	return true
}
