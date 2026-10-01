package style

// TabShape is how a browser's tabs meet the strip they stand in.
//
// The two browsers people copy chose differently, and the difference is
// not decoration: it is the whole silhouette of the top of the window.
//
// Chrome's tab is a *merged* one. Its selected tab is filled with the
// tool bar's colour and has concave feet that run into the row below, so
// the tab, the tool bar and the page read as one continuous surface with
// a channel cut through the strip.
//
// Firefox's, since Proton, is a *floating* one. Its tabs are rounded
// rectangles with a margin on every side, standing clear of the tool bar
// rather than meeting it, so the strip reads as a tray with cards in it.
//
// The toolkit drew only the first, which is why a sample copying Firefox
// came out looking like Chrome wearing Firefox's colours.
type TabShape uint8

const (
	// TabsMerged is Chrome's: feet into the tool bar, one surface. The
	// default, because it is the older and commoner shape and because it
	// is what the toolkit has always drawn.
	TabsMerged TabShape = iota
	// TabsFloating is Firefox's: a rounded card standing clear of the
	// row below it.
	TabsFloating
)
