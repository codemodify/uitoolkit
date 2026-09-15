package widgets

import (
	"slices"

	"github.com/codemodify/uitoolkit/platform"
)

// SelectionMode is how a list or table selects rows (Qt's
// QAbstractItemView::SelectionMode).
type SelectionMode uint8

const (
	// SelectSingle keeps one selected row, the current one (the default).
	SelectSingle SelectionMode = iota
	// SelectExtended is the desktop convention (Explorer, Finder,
	// Thunderbird): a click selects one row, Ctrl+click toggles one,
	// Shift+click and Shift+arrows extend a range from the anchor, Ctrl+A
	// selects everything.
	SelectExtended
	// SelectMulti toggles the clicked row; arrows move the current row and
	// Space toggles it.
	SelectMulti
)

// rowSelection is the selected rows of a multi-selecting view plus the
// anchor a Shift range extends from. The view's current row (its Selected
// field) is kept separately: it is where the keyboard is, selected or not.
type rowSelection struct {
	set    map[int]struct{}
	anchor int
}

func (s *rowSelection) has(i int) bool {
	_, ok := s.set[i]
	return ok
}

func (s *rowSelection) clear() {
	clear(s.set)
}

func (s *rowSelection) add(i int) {
	if s.set == nil {
		s.set = make(map[int]struct{})
	}
	s.set[i] = struct{}{}
}

func (s *rowSelection) toggle(i int) {
	if s.has(i) {
		delete(s.set, i)
	} else {
		s.add(i)
	}
}

func (s *rowSelection) addRange(a, b int) {
	if a > b {
		a, b = b, a
	}
	for i := a; i <= b; i++ {
		s.add(i)
	}
}

// rows is the selection in ascending order.
func (s *rowSelection) rows() []int {
	out := make([]int, 0, len(s.set))
	for i := range s.set {
		out = append(out, i)
	}
	slices.Sort(out)
	return out
}

// drop forgets rows at or past count (the model shrank).
func (s *rowSelection) drop(count int) {
	for i := range s.set {
		if i < 0 || i >= count {
			delete(s.set, i)
		}
	}
	if s.anchor >= count {
		s.anchor = count - 1
	}
}

// selectionClick applies a primary-button click on row i under mode and
// mods to the selection. It reports whether the selection changed.
func (s *rowSelection) click(mode SelectionMode, i int, mods platform.Modifiers) bool {
	before := len(s.set)
	had := s.has(i)
	switch mode {
	case SelectMulti:
		s.toggle(i)
		s.anchor = i
		return true
	case SelectExtended:
		switch {
		case mods.Shift():
			if !mods.Ctrl() {
				s.clear()
			}
			s.addRange(s.anchor, i)
			return true
		case mods.Ctrl():
			s.toggle(i)
			s.anchor = i
			return true
		}
	}
	s.clear()
	s.add(i)
	s.anchor = i
	return before != 1 || !had
}

// contextClick is a secondary-button press on row i: a row outside the
// selection becomes the selection (the menu acts on what it was opened
// over); inside it, the selection stays for the menu to act on.
func (s *rowSelection) contextClick(i int) bool {
	if s.has(i) {
		return false
	}
	s.clear()
	s.add(i)
	s.anchor = i
	return true
}

// moveTo applies keyboard navigation that landed on row i: Shift extends
// from the anchor (Extended), Ctrl moves without selecting (Extended), and
// plain navigation selects only i. Multi only moves the current row.
func (s *rowSelection) moveTo(mode SelectionMode, i int, mods platform.Modifiers) bool {
	switch mode {
	case SelectMulti:
		return false
	case SelectExtended:
		if mods.Shift() {
			s.clear()
			s.addRange(s.anchor, i)
			return true
		}
		if mods.Ctrl() {
			return false
		}
	}
	s.clear()
	s.add(i)
	s.anchor = i
	return true
}
