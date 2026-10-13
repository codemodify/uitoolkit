# Complex text: what a spike found

A spike, not a feature. Nothing in the toolkit calls any of this; it
exists to turn three guesses into measurements before anyone commits to
[the text contract](contracts.md) changing. Build it with
`-tags font_complex`; leave the tag off and the toolkit is exactly what
it was.

    tools/testenv.sh go test -tags font_complex ./style/

## The three questions

**1. A shaper answers in glyph indices. How much of the toolkit assumes
runes?**

Less than feared. `paintengine2d.GlyphID` is already an opaque `uint32`
whose own documentation says *"bitmap fonts typically use the rune value;
a shaper may assign packed IDs"* — the renderer never cared. The toolkit
is what fills it with runes, and it does so in exactly **one function**:
`Font.layout` in `style/font.go`, which walks a string, looks each rune up
in the sheet and emits positioned glyphs. Everything else — `Fit`,
`Prefix`, `Wrap`, `CaretX`, `IndexAt`, `Draw` — reads the `shapedRun` that
function returns and never sees a rune.

So one sheet can hold both numberings, with one bit to say which:

```go
const shapedGIDBit = paintengine2d.GlyphID(1) << 31
```

Runes stop at `0x10FFFF`, so the top bit is free. Glyph index 65 and
`'A'` then live in the same atlas without colliding, which matters
because a window draws a shaped label and a rune-mapped icon from the
same face.

**2. Can the existing rasterizer draw a glyph the shaper chose?**

Yes, unchanged. `atlasDraft.rasterize` already goes rune → glyph index →
outline; the shaper has simply done the first step. Dropping it is four
lines (`blitGID`), and `x/image/font/sfnt` rasterizes by index happily —
including glyphs that have no rune at all, which is the whole point:

| | glyphs the runes map to | glyphs the shaper picked |
| --- | --- | --- |
| `->` in JetBrains Mono | 934, 1318 | **1751, 841** |
| `سلام` in Noto Naskh Arabic | 32, 66, 8, 72 | **72, 11, 69, 35** |

The Arabic row is the case the current toolkit cannot represent at all:
four runes become four *joining forms*, in visual order, none of them the
glyph its rune maps to. There is no rune to key them by.

Both rows are tests (`style/shape_complex_test.go`), and the second one
draws: the shaped run goes through the ordinary sheet, the ordinary
packer and `Context.DrawGlyphs`, and leaves ink.

**3. What does it weigh?**

Measured here, stripped (`-ldflags "-s -w"`), each binary linking what
the row says:

| | size | delta |
| --- | --- | --- |
| hello world (the Go floor) | 1.44 MB | |
| `x/image/font/sfnt`, parse + rasterize | 1.77 MB | +0.33 |
| **a** — `style`, as the toolkit is today | 5.19 MB | |
| **c** — a + go-text's OpenType parser | 6.26 MB | **+1.07** |
| **b** — c + harfbuzz + `language` | 7.09 MB | **+0.83** |
| **f** — b + the line/word segmenter | 7.14 MB | +0.05 |
| `cmd/uitoolkit-settings`, default | 13.04 MB | |
| the same, `-tags font_complex` | 13.12 MB | +0.08 |

Three things to read out of that:

- **The shaper itself is the cheap half.** Harfbuzz and its script tables
  are 0.83 MB. Go-text's *font parser* is 1.07 MB, and it is only on the
  bill because the spike keeps both parsers: a real implementation would
  parse each face once with go-text and drop `sfnt`, which gives 0.33 MB
  back. Call the honest end state **+1.6 MB**, and the spike's
  as-it-stands **+1.9 MB**.
- **Nobody who does not ask for it pays.** The tag as wired costs 0.08 MB
  because the linker drops a shaper nothing calls; off, it costs nothing
  at all.
- **Bidi and segmentation are rounding errors** next to the shaper
  (+0.05 MB for the segmenter here, +0.10 for bidi when measured on its
  own). The earlier estimate of +1.35 MB for all three was measuring the
  shaper against a main that already linked go-text's parser; in place,
  with the toolkit's own parser still there, it is the number above.

And the rule from [fonts.md](fonts.md) survives intact: this is all
*code*. One Noto Sans CJK family is 18.6 MB of font data. Supporting
every script on earth costs 1.6 MB as long as the faces stay the
machine's.

## What a real implementation still has to do

The spike proves the bottom of the stack. Above it:

- **`Font.layout` rewritten around clusters.** It currently builds `xs`,
  one x per rune, which `CaretX`, `IndexAt` and `Fit` index directly. A
  cluster covers several runes and may be several glyphs, so `xs` keeps
  its shape — one entry per rune — but is filled from cluster boundaries
  rather than from the pen after each glyph.
- **Runs, not strings.** One string can be several scripts, directions
  and faces. UAX #9 splits a paragraph into runs; the segmenter splits
  for line breaking; `shapeOf` already holds *several* `GlyphRun`s (one
  per lending face, from `font_fallback`), so the shape of the answer is
  there.
- **The shaping cache.** `shapeOf` memoizes by string; shaped output is
  bigger and the key would want the script and direction in it.
- **RTL in the widgets.** Shaping puts glyphs in visual order; selection,
  caret movement, and hit-testing in `TextArea` and `RichText` assume
  logical order equals visual order. This is the largest piece of work in
  the list and none of it is in the spike.
- **One parser.** Keep `sfnt` and every face is parsed twice; drop it and
  the outline path, the metrics and `symbolFallback` move to go-text.

Not touched at all: vertical text (UAX #50), colour glyph formats
(COLR/CPAL, CBDT, sbix — see the emoji note in [fonts.md](fonts.md)),
and IME.

## The dependency

`github.com/go-text/typesetting v0.3.5` — Unlicense OR BSD-3-Clause, with
the harfbuzz port under MIT. Pulling it in moved `golang.org/x/image`
from v0.18.0 to v0.23.0 and `golang.org/x/text` from v0.16.0 to v0.21.0,
which the default build takes whether or not the tag is on.
