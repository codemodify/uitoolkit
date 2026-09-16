package demo

import "testing"

// Moving notes into a gap keeps their order and lands them where the
// caret was: before the row the gap belongs to, or at the end.
func TestMoveNotesLandsWhereTheCaretWas(t *testing.T) {
	base := []Note{{Title: "a"}, {Title: "b"}, {Title: "c"}, {Title: "d"}}
	all := []int{0, 1, 2, 3}
	for _, tc := range []struct {
		name  string
		rows  notesMove
		at    int
		want  string
		first int
	}{
		{"one note to the head", notesMove{2}, 0, "cabd", 0},
		{"one note to the end", notesMove{0}, 4, "bcda", 3},
		{"one note one place down", notesMove{0}, 2, "bacd", 1},
		{"a note into its own gap does not move", notesMove{1}, 1, "abcd", 1},
		{"two notes, in their own order", notesMove{3, 1}, 1, "abdc", 1},
		{"the whole list to the end", notesMove{0, 1, 2, 3}, 4, "abcd", 0},
	} {
		notes := append([]Note(nil), base...)
		got := moveNotes(&notes, all, tc.rows, tc.at)
		if titles(notes) != tc.want || got != tc.first {
			t.Errorf("%s: %q first=%d, want %q first=%d", tc.name, titles(notes), got, tc.want, tc.first)
		}
	}
	// Nothing to move is refused rather than silently rebuilding the list.
	notes := append([]Note(nil), base...)
	if got := moveNotes(&notes, all, nil, 0); got != -1 || titles(notes) != "abcd" {
		t.Fatalf("an empty move gave %d and %q", got, titles(notes))
	}
	if got := moveNotes(&notes, all, notesMove{9}, 0); got != -1 {
		t.Fatalf("a row that is not there gave %d", got)
	}
}

// A filtered list moves only what it shows, and the notes it hides stay
// where they are.
func TestMoveNotesThroughAFilter(t *testing.T) {
	notes := []Note{{Title: "a"}, {Title: "hidden"}, {Title: "b"}, {Title: "c"}}
	vis := []int{0, 2, 3} // "hidden" is filtered out
	// The last visible note to the head of the visible list.
	if got := moveNotes(&notes, vis, notesMove{2}, 0); got != 0 {
		t.Fatalf("moved to %d, want the head", got)
	}
	if titles(notes) != "cahiddenb" {
		t.Fatalf("got %q, want the hidden note left where it was", titles(notes))
	}
}

func titles(notes []Note) string {
	out := ""
	for _, n := range notes {
		out += n.Title
	}
	return out
}
