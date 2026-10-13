package style

// DirectionRun is a stretch of a string that reads one way, as the
// bidirectional algorithm (UAX #9) divides it.
//
// Start and End are rune indices into the string. A list of them is in
// the order the runs are *drawn*, left to right, which is not the order
// they are written in: the pieces of "abc שלום abc" are drawn in the
// order they are read only for the Latin.
type DirectionRun struct {
	Start, End int
	RTL        bool
}
