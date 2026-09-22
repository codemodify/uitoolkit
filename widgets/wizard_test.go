package widgets

import (
	"errors"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// wizardSample is an account set-up: a required name, an e-mail address
// that must hold an @, optional settings, and a proxy page that only
// shows when a proxy is asked for.
type wizardSample struct {
	w      *Wizard
	name   *TextField
	email  *TextField
	proxy  *Checkbox
	host   *TextField
	finish int
	cancel int
}

func newWizardSample() *wizardSample {
	s := &wizardSample{}
	s.name = NewTextField("", "Full name", nil)
	s.email = NewTextField("", "Address", nil)
	s.proxy = NewCheckbox("Connect through a proxy", false, nil)
	s.host = NewTextField("", "Proxy host", nil)
	s.w = NewWizard("New account",
		&WizardPage{Title: "Welcome", Subtitle: "This assistant sets up an account.", Content: NewLabel("Press Next to begin.")},
		&WizardPage{Title: "Account", Subtitle: "Who you are.", Content: NewColumn(s.name, s.email).WithGap(8),
			Complete: func() bool { return strings.TrimSpace(s.name.Text) != "" },
			Validate: func() error {
				if !strings.Contains(s.email.Text, "@") {
					return errors.New("The address needs an @.")
				}
				return nil
			}},
		&WizardPage{Title: "Connection", Subtitle: "How to reach the server.", Content: s.proxy, Optional: true},
		&WizardPage{Title: "Proxy", Content: s.host, Skip: func() bool { return !s.proxy.Checked }},
		&WizardPage{Title: "Done", Subtitle: "Everything is ready.", Content: NewLabel("Press Finish.")},
	)
	s.w.OnFinish = func() { s.finish++ }
	s.w.OnCancel = func() bool { s.cancel++; return true }
	return s
}

func wizardHarness(t *testing.T, st WizardStyle) (*wizardSample, *lookHost) {
	t.Helper()
	s := newWizardSample()
	s.w.Style = st
	hs := &lookHost{lk: style.LightLook()}
	s.w.SetHost(hs)
	s.w.Arrange(paintengine2d.XYWH(0, 0, 640, 440))
	return s, hs
}

func TestWizardNavigationAndValidation(t *testing.T) {
	s, _ := wizardHarness(t, WizardClassic)
	w := s.w
	var changes [][2]int
	w.OnPageChange = func(from, to int) { changes = append(changes, [2]int{from, to}) }
	if w.bBack.Enabled() || !w.bNext.Visible() || w.bFinish.Visible() {
		t.Fatal("first page: no Back, Next not Finish")
	}
	if !w.Next() || w.Current() != 1 {
		t.Fatal("next")
	}
	// Incomplete: Next is greyed and does nothing.
	w.UpdateButtons()
	if w.bNext.Enabled() || w.Next() {
		t.Fatal("an incomplete page must hold Next back")
	}
	s.name.SetText("Ada")
	w.UpdateButtons()
	if !w.bNext.Enabled() {
		t.Fatal("complete page enables Next")
	}
	// Invalid: Next stays, with the page's message.
	s.email.SetText("ada.example.com")
	if w.Next() || w.Current() != 1 || w.ErrorText() != "The address needs an @." || !w.errLabel.Visible() {
		t.Fatalf("validation: page %d, %q", w.Current(), w.ErrorText())
	}
	s.email.SetText("ada@example.com")
	if !w.Next() || w.ErrorText() != "" {
		t.Fatal("valid page moves on and clears the message")
	}
	// Optional page: Skip shows and moves on without validation; the
	// proxy page is left out while no proxy is asked for.
	if !w.bSkip.Visible() {
		t.Fatal("skip on an optional page")
	}
	if !w.SkipPage() || w.CurrentPage().Title != "Done" {
		t.Fatalf("skip went to %q", w.CurrentPage().Title)
	}
	if !w.bFinish.Visible() || w.bNext.Visible() {
		t.Fatal("the last page shows Finish")
	}
	// Back retraces the way taken.
	w.Back()
	if w.CurrentPage().Title != "Connection" {
		t.Fatalf("back to %q", w.CurrentPage().Title)
	}
	s.proxy.Checked = true
	w.Next()
	if w.CurrentPage().Title != "Proxy" {
		t.Fatalf("a choice brings the proxy page in: %q", w.CurrentPage().Title)
	}
	if got := w.History(); len(got) != 3 || got[2] != 2 {
		t.Fatalf("history %v", got)
	}
	w.Next()
	if !w.Finish() || !w.Finished() || s.finish != 1 {
		t.Fatal("finish")
	}
	if w.Next() || w.Back() || w.bFinish.Enabled() {
		t.Fatal("a finished wizard does not move")
	}
	if len(changes) != 6 {
		t.Fatalf("page changes %v", changes)
	}
	w.Restart()
	if w.Current() != 0 || w.Finished() || len(w.History()) != 0 {
		t.Fatal("restart")
	}
}

func TestWizardSkippedPagesInSteps(t *testing.T) {
	s, _ := wizardHarness(t, WizardModern)
	w := s.w
	w.UpdateButtons()
	if w.steps.Count != 4 || !w.steps.Visible() {
		t.Fatalf("%d steps listed, want the proxy page left out", w.steps.Count)
	}
	s.proxy.Checked = true
	w.UpdateButtons()
	if w.steps.Count != 5 {
		t.Fatal("the proxy page joins the steps")
	}
	if !strings.Contains(w.stepText(2), "(optional)") {
		t.Fatalf("optional step %q", w.stepText(2))
	}
	w.Next()
	if w.steps.Selected != 1 {
		t.Fatalf("current step %d", w.steps.Selected)
	}
	// A commit page cannot be gone back past.
	w.pages[2].Commit = true
	s.name.SetText("Ada")
	s.email.SetText("a@b")
	w.Next()
	w.UpdateButtons()
	if w.bNext.Text != "Apply" {
		t.Fatalf("commit label %q", w.bNext.Text)
	}
	w.Next()
	if w.bBack.Enabled() || w.Back() {
		t.Fatal("back past a commit page")
	}
}

func TestWizardKeyboard(t *testing.T) {
	s, hs := wizardHarness(t, WizardClassic)
	w := s.w
	bubble := func(k platform.Key) {
		for c := hs.Focus(); c != nil; c = c.Parent() {
			if c.KeyPress(widget.KeyEvent{Key: k}) {
				return
			}
		}
		w.KeyPress(widget.KeyEvent{Key: k})
	}
	bubble(platform.KeyReturn)
	if w.Current() != 1 || hs.Focus() != s.name {
		t.Fatalf("Return: page %d, focus %T", w.Current(), hs.Focus())
	}
	// Return in an incomplete page's field does not move on.
	bubble(platform.KeyReturn)
	if w.Current() != 1 {
		t.Fatal("Return past an incomplete page")
	}
	s.name.SetText("Ada")
	s.email.SetText("ada@example.com")
	bubble(platform.KeyReturn)
	if w.Current() != 2 {
		t.Fatal("Return on a complete page")
	}
	// Alt+B goes back, Alt+N forward again.
	if !w.HandleAlt(platform.KeyB) || w.Current() != 1 || !w.HandleAlt(platform.KeyN) || w.Current() != 2 {
		t.Fatalf("access keys: page %d", w.Current())
	}
	bubble(platform.KeyEscape)
	if !w.Cancelled() || s.cancel != 1 {
		t.Fatal("Escape cancels")
	}
	// OnCancel can keep the wizard going.
	s2, _ := wizardHarness(t, WizardModern)
	s2.w.OnCancel = func() bool { return false }
	s2.w.KeyPress(widget.KeyEvent{Key: platform.KeyEscape})
	if s2.w.Cancelled() {
		t.Fatal("a refused cancel")
	}
	// Buttons are in the Tab order; the steps list is not.
	for _, c := range widget.Focusables(w) {
		if c == widget.Component(w.steps) {
			t.Fatal("the steps list takes focus")
		}
	}
}

func TestWizardLayoutAndAccessibility(t *testing.T) {
	for _, st := range []WizardStyle{WizardClassic, WizardModern} {
		s, hs := wizardHarness(t, st)
		w := s.w
		w.OnHelp = func(int) {}
		w.UpdateButtons()
		w.Arrange(w.Bounds())
		left, right := w.bHelp.Bounds(), w.bNext.Bounds()
		if !w.bHelp.Visible() || left.Min.X >= right.Min.X {
			t.Fatalf("style %d: help at the left", st)
		}
		back, cancel := w.bBack.Bounds(), w.bCancel.Bounds()
		if st == WizardClassic && cancel.Min.X < right.Max.X {
			t.Fatal("classic: Cancel after Next")
		}
		if st == WizardModern && (cancel.Min.X > back.Min.X || !w.steps.Visible()) {
			t.Fatal("modern: Cancel at the left, steps shown")
		}
		root := &a11y.Node{ID: 1, Role: a11y.RoleWindow, Name: "w", Bounds: paintengine2d.XYWH(0, 0, 640, 440)}
		col := NewColumn(w)
		col.SetHost(hs)
		widget.AccessibleTree(root, col)
		var dlg *a11y.Node
		root.Walk(func(n *a11y.Node) bool {
			if n.Role == a11y.RoleDialog {
				dlg = n
			}
			return true
		})
		if dlg == nil || dlg.Name != "New account" || !strings.HasPrefix(dlg.Description, "Step 1 of 4") {
			t.Fatalf("dialog %+v", dlg)
		}
		for _, p := range a11y.Check(root) {
			t.Error(p)
		}
	}
}
