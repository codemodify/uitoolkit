//go:build theme_engine_all || theme_engine_system7

package style

import ()

// Whole-pixel drawing shared by the retro engines: system7, win31,
// openlook, amiga, beos and os2.
//
// The desktops of the 1980s and early 1990s were drawn a pixel at a time on
// 1-bit and 4-bit screens. QuickDraw's round rects and ovals are staircases
// rather than anti-aliased curves, a grey is a dither, and a 3D edge is a
// row of whole pixels. These helpers build such shapes out of whole-pixel
// rectangles laid on a grid of cells, one cell being one pixel of the
// original screen and u device pixels of ours, so a 2x display shows the
// same drawing pixel-doubled. Every cell of one colour goes into a single
// path that fills in one draw; rows that repeat merge into taller rects, so
// a staircase corner or a dither costs a few dozen rectangles.
//
// Shapes are computed from geometry (the circle through pixel centres, run
// lengths) and patterns from their logic; no bitmap of any original system
// is reproduced.

// rpU is one design pixel (cell) in whole device pixels: 1 at 1x, 2 at 1.5x
// and 2x, so lines stay crisp at every scale.

// rpGrid lays whole cells over a rect: the device origin of cell (0, 0),
// the cell size u, and how many whole cells fit across and down.

// rpGridOf snaps b to the device grid (keeping pixels whose centres lie
// inside b) and fits whole cells of u into it from its top-left corner.

// rpOrigin is the device position of the context's user origin when the
// context only translates (widgets paint in their own coordinates under a
// translation that layout may leave fractional); zero otherwise.

// rpGridAt is rpGridOf snapped on the device's pixel grid rather than in
// user space, so a control whose origin falls between device pixels (a
// dialog centred on a half pixel) still draws in whole pixels.

// at is the device rect of cells [cx, cx+cw) × [cy, cy+ch).

// rect is the device rect of the whole grid.

// sub is the grid of cw × ch cells whose origin is cell (cx, cy) of g.

// inset is g shrunk by n cells on every side.

// centered is a cw × ch grid centred in g on whole cells.

// ok reports whether g holds at least w × h cells.

// rpPaths recycles the paths inks gather into: a path keeps its capacity
// across paints, so a repaint builds its staircases and dithers without
// allocating. (Backends never keep the path itself: recorders snapshot
// it, the GPU caches its tessellation by content.)

// rpInk gathers the whole-pixel rects of one colour into one path.

// add appends r (ignoring empty rects).

// cells appends cells [cx, cx+cw) × [cy, cy+ch) of g.

// fill paints everything gathered in c, in one draw, and empties the ink.

// frame appends a rectangular frame t cells thick around the cw × ch cells
// at (cx, cy), as four rects that never overlap.

// rpEdge appends a bevel of n one-cell rings around the cw × ch cells at
// (cx, cy): hi along the top and left, lo along the bottom and right, lo
// owning the top-right and bottom-left corner cells (the Windows and Amiga
// convention). Either ink may be nil.

// rpDotted appends the classic dotted focus rectangle just inside the
// cw × ch cells at (cx, cy): every other cell along each side.
// aboveGlyph emits the keep-above glyph n cells across with its top-left at
// cx, cy of g: a ceiling rule with the window held up under it, hollow while
// keep-above is off and solid while it is on. It is the 1-bit twin of
// DrawCaptionGlyph's, so the pixel-grid eras read the same as the rest.

// Too few cells to hollow out and still read as a box.

// ---- shapes ------------------------------------------------------------------

// rpInset is how many cells row y (0 = the outermost row) of a corner of
// radius r cuts from the straight edge: the circle through pixel centres,
// rounded to whole cells, as QuickDraw rasterised its round rects.

// rpInsetE is rpInset for an elliptical corner rx cells across and ry down.

// rpShape is a whole-pixel shape in a w × h cell box, symmetric about both
// axes: a rectangle whose corners are cut by a circle of radius r (a round
// rect; an oval or a stadium when r is half the short side) or, when r is 0,
// by a 45° chamfer of cut cells.

// top keeps the bottom corners square (tabs).

// rx, when set, makes the corners elliptical: rx across, r down
// (OPEN LOOK's buttons end in caps wider than they are tall).

// edge is the first filled cell of row y; the row spans [edge, w-edge).
// Rows outside the box report w (empty).

// rpRows gathers the one or two runs of each row of a shape and merges
// rows that repeat into taller rects.

// row adds row y (rows come in order; an empty row flushes first).

// fill appends the whole shape with its top-left cell at (ox, oy) of g.

// frameRuns is row y of the shape's one-cell outline (every filled cell
// with an empty 4-neighbour): one or two runs [a1, b1) and [a2, b2).

// frame appends the shape's one-cell outline, the thin staircase QuickDraw
// framed round rects with.

// frameDots appends every other cell of the outline — a dotted outline,
// the cells whose x+y is even in the grid's own coordinates.

// ringRuns is row y of the band t cells thick inside the shape's edge: the
// shape minus the concentric shape t cells smaller (QuickDraw's thick pen).

// ring appends the band t cells thick inside the shape's edge.

// ringDots appends every other cell of the band (x+y even): a band in
// the 50% grey of 1-bit screens.

// ---- masks ---------------------------------------------------------------------

// rpMask is a small one-bit cell bitmap (at most 64 × 64) that the engines
// compose their glyphs in — arrows, check marks, boxes, dots — from lines,
// spans and shapes, then draw as whole pixels. Bit x of rows[y] is cell
// (x, y).

// set turns cell (x, y) on (cells outside the mask are ignored).

// get reports cell (x, y).

// span turns on cells [x0, x1) of row y.

// rect turns on the w × h cells at (x, y).

// line turns on the cells of the Bresenham line from (x0, y0) to (x1, y1).

// shape turns on the cells of s with its top-left cell at (x, y).

// outline keeps the set cells that have an empty 4-neighbour.

// minus clears the cells set in o.

// or sets the cells set in o.

// turn rotates a glyph drawn pointing up so it points dir.

// emit appends the set cells with the mask's top-left cell at (ox, oy) of
// g: one rect per run, repeated rows merged.

// ---- patterns ------------------------------------------------------------------

// rpPat is an 8 × 8 one-bit pattern, one byte per row with the most
// significant bit leftmost — how QuickDraw, Windows and Intuition all
// described their fill patterns.

// rpMaxCells is where rpPatFill stops listing every cell of a checkerboard
// and fills it as the even-odd overlap of stripes instead (the CPU fills up
// to 4096 separate rects of one path analytically).

// rpPatFill paints the set cells of pat over r in col, in one path. The
// pattern is anchored to the device grid in cells of u, so pieces of one
// surface painted separately (a trough split by its thumb) keep one phase.

// Work on the device grid: the pattern's phase and every cell edge are
// whole device pixels wherever the control's origin lies.

// A checkerboard is the exclusive-or of every other column with
// every other row: a few hundred stripes filled even-odd instead of
// one rect per cell.
// rpGray: cell (x, y) is set when x+y is even

// ---- text in fields -------------------------------------------------------------

// rpFieldOpts styles the text of a whole-pixel text field or text area.

// text; placeholder and disabled text
// selection fill and its text
// selection fill without focus (unset: sel)

// grey disabled text the 1-bit way
// and placeholders too (1-bit looks have no grey)
// what the dither erases to
// an unfocused selection is outlined, not filled

// rpCaret paints the insertion point at x for a line of face f at y.

// A solid triangle five cells wide and six tall, its tip two cells
// above the baseline.

// rpFieldText paints the text of a single-line field inside inner: the
// selection, the text, the caret on whole pixels.

// rpAreaText paints the lines of a text area inside inner (selection per
// line, text, caret), on whole pixels.
