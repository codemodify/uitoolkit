package dock

import "testing"

// Closing a panel, collapsing one and switching the current tab are all
// part of the arrangement, so all three are worth saving. They called
// relayout, which only re-measures, so an application persisting its
// layout through OnLayoutChanged — the Inspector sample does — never
// heard about any of them.
func TestClosingAndCollapsingAnnounceALayoutChange(t *testing.T) {
	for _, c := range []struct {
		name string
		do   func(r *rig)
	}{
		{"closing a panel", func(r *rig) { r.tree.Close() }},
		{"reopening a panel", func(r *rig) { r.tree.Close(); r.tree.Show() }},
		{"collapsing a panel", func(r *rig) { r.tree.SetCollapsed(true) }},
		{"expanding a panel", func(r *rig) { r.tree.SetCollapsed(true); r.tree.SetCollapsed(false) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRig(t)
			n := 0
			r.host.OnLayoutChanged = func() { n++ }
			c.do(r)
			if n == 0 {
				t.Fatalf("%s told nobody; the layout would never be saved", c.name)
			}
		})
	}
}

// Selecting another tab changes which panel the saved layout calls
// current, so it is a layout change too.
func TestSelectingATabAnnouncesALayoutChange(t *testing.T) {
	r := newRig(t)
	second := NewPanel("second", "Second", nil)
	r.host.DockInto(r.tree, second)
	st := second.Stack()
	if st == nil || len(st.Panels()) < 2 {
		t.Skip("the two panels did not share a stack")
	}
	n := 0
	r.host.OnLayoutChanged = func() { n++ }
	for i, p := range st.Panels() {
		if p != st.Current() {
			st.Select(i)
			break
		}
	}
	if n == 0 {
		t.Fatal("selecting another tab told nobody")
	}
}

// ResetLayout applies the stored default, and the live splits used to
// alias its arrays: one sash drag afterwards rewrote the defaults
// themselves, so a later reset restored the dragged proportions rather
// than the original ones.
func TestAppliedLayoutDoesNotAliasItsWeights(t *testing.T) {
	r := newRig(t)
	saved := r.host.SaveLayout()
	before := saved.Weights.Cols
	if err := r.host.ApplyLayout(saved); err != nil {
		t.Fatal(err)
	}
	r.host.middle.moveSash(0, 300)
	if got := saved.Weights.Cols; got != before {
		t.Fatalf("moving a sash rewrote the layout it was applied from: %v, was %v", got, before)
	}
}

// NewPanel already sets DefaultFeatures, so a zero mask when a panel is
// docked is a caller who said SetFeatures(0) and meant it. Filling it in
// made a deliberately locked panel closable, movable, floatable and
// collapsible the moment it was docked.
func TestDockingKeepsAFeaturelessPanelFeatureless(t *testing.T) {
	h := NewHost(nil)
	p := NewPanel("locked", "Locked", nil)
	p.SetFeatures(0)
	h.Dock(p, SideLeft)
	if got := p.Features(); got != 0 {
		t.Fatalf("docking gave a locked panel features %v", got)
	}
}
