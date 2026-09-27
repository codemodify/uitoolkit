package richtext

import "testing"

// Removing every block leaves an empty paragraph behind, because a
// document is never empty. That has to be recorded as what happened, or
// undo puts the original blocks in front of a paragraph the history
// never knew about.
func TestUndoingAFullRemovalDoesNotLeaveABlankLine(t *testing.T) {
	d := NewPlain("hello")
	d.ReplaceBlocks(0, d.Len())
	d.Undo()
	if got := d.Text(); got != "hello" {
		t.Fatalf("undo gave %q, want %q", got, "hello")
	}
}

// A transaction is one undo step, and it must not swallow the step
// before it. Typing "a", then transacting "b" and "c", then undoing once
// has to leave "a" — the transaction's first edit used to merge into the
// typing step that preceded it, so the undo took the "a" away too.
func TestTransactionDoesNotAbsorbThePrecedingTyping(t *testing.T) {
	d := NewPlain("")
	d.InsertText("a")
	d.Transact(func() {
		d.InsertText("b")
		d.InsertText("c")
	})
	d.Undo()
	if got := d.Text(); got != "a" {
		t.Fatalf("after undoing the transaction the document is %q, want %q", got, "a")
	}
}
