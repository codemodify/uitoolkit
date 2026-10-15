# Complex text: shaping, direction and line breaking

*(Weighing this against the other build-time choices:
[build-tags.md](build-tags.md).)*

`-tags font_complex` makes the toolkit read the font's own tables
instead of looking each character up by itself. Leave the tag off and
nothing here applies: the toolkit is exactly what [the text
contract](contracts.md) describes, which is what every build before this
one was.

    go build -tags font_complex ./...
    tools/test-fonts.sh                 # the suite in this build and in font_fallback

It implies `font_fallback`: shaping chooses the *face* for each run of
text, and the fallback list is where the faces it may choose from are.

## What it does

| | without the tag | with it |
| --- | --- | --- |
| glyphs | one per rune, by code point | the face's GSUB: ligatures, joining forms, Indic reordering |
| spacing | the `kern` table, pairs only | GPOS: pair kerning, mark attachment |
| direction | left to right | UAX #9: a line that changes direction is ordered for reading |
| caret, click, selection | one rule for the whole line | per character, in its own direction |
| line breaking | at spaces | UAX #14: between Han characters, after a Thai word, and *not* before a closing bracket |
| the lending face | per rune | per run, through the segmenter |

Arabic سلام is the case worth stating plainly. Without the tag its four
runes are four isolated letters, each drawn as if alone. With it they are
four joining forms — four glyphs that *no rune maps to*, which is why
this needs the shaper and not a bigger table.

### Which characters are pictures

Under `-tags font_emoji` the toolkit has to decide, for every *position*
in a string, whether to draw a colour picture or an outline. It asks the
Unicode Character Database rather than a range, because a range cannot
answer it: ⚡ U+26A1 and ⌚ U+231A are pictures by default and sit below
the pictographic planes, while 🜁 U+1F701, 🨀 U+1FA00 and 🄰 U+1F130 sit
inside them and are text.

Two properties, generated into `style/emojidata.go` by
`tools/emoji/gen.go` from `emoji-data.txt` (236 ranges, about 5 KB of
source, and only in builds with the tag):

- **`Emoji_Presentation`** — a picture with nobody asking. `⚡` is;
  `❤` and `🏳` are not, and ask with U+FE0F.
- **`Extended_Pictographic`** — could be *part* of a picture, which is
  what a zero-width joiner needs to know about its neighbours.

On top of them, UTS #51's own rules: U+FE0F asks for the emoji form and
U+FE0E for the text form, the text form wins, and a sequence nobody
qualified is still composed — `🏳‍🌈` without the selector, and `1⃣`
without it, are each one picture, because every emoji font makes one of
them.

To refresh the table after a Unicode release:

```sh
go run tools/emoji/gen.go          # fetches the current emoji-data.txt
```

## What an application has to do

Nothing. Every widget that draws text goes through `style.Font`, and
that is where this lives. Two pieces of it are exported for an
application — or a widget — that places text itself:

```go
// The stretches of screen a stretch of text covers: one, nearly always.
spans := font.SelectionSpans(text, from, to)

// The runs of a string that read one way, in the order they are drawn.
// nil when the string reads one way, which is nearly every string.
runs := style.DirectionRuns(text)
```

`DirectionRuns` is for a line assembled from *several* strings, each
drawn by a call of its own: that is how [RichText](../widgets/richtext.go)
draws a line of styled fragments, and why it has to put its own
fragments in the order the rule gives
([widgets/richtext_bidi.go](../widgets/richtext_bidi.go)). Text drawn in
one call is already ordered by `Font.Draw`.

## How it fits together

    the paragraph's direction (UAX #9 P2, P3)
      → Segmenter.Split — bidi runs, then script, then face
        → harfbuzz per run — GSUB, GPOS, the glyphs and where they go
          → the runs ordered for display (UAX #9 L2)
            → one atlas group per face, and a caret table per rune

Two details are worth knowing if you work on it.

**The atlas key carries a bit.** A sheet is keyed by
`paintengine2d.GlyphID`, which the toolkit filled with runes and a shaper
fills with glyph indices — where 65 is whatever glyph the font put at
index 65, not `'A'`. Runes stop at `0x10FFFF`, so the top bit says which
numbering a key is in and one sheet holds both. It has to: a cluster the
shaper has no glyph for is handed *back* to the rune path, which has a
drawn star for `★` and a box for everything else.

**`xs` holds leading edges.** The caret table has one entry per rune plus
one, and an entry is where that rune *starts* — its left edge in text
that reads left to right and its right edge in text that does not. So
the table does not have to climb, and "where does this rune end" is no
longer "where the next one starts": the shaped layout fills a second
table, `ends`, and a layout that reads one way leaves it nil. That nil
is what every fast path tests.

## What it costs

Measured on this machine, stripped (`-ldflags "-s -w"`), each binary
linking what the row says:

| | size | delta |
| --- | --- | --- |
| hello world (the Go floor) | 1.44 MB | |
| `x/image/font/sfnt`, parse + rasterize | 1.77 MB | +0.33 |
| **a** — `style`, as the toolkit is without the tag | 5.19 MB | |
| **c** — a + go-text's OpenType parser | 6.26 MB | **+1.07** |
| **b** — c + harfbuzz + `language` | 7.09 MB | **+0.83** |
| **f** — b + the line/word segmenter | 7.14 MB | +0.05 |
| `cmd/uitoolkit-settings`, default | 13.05 MB | |
| the same, `-tags font_fallback` | 13.06 MB | +0.01 |
| the same, **`-tags font_complex`** | **15.21 MB** | **+2.16** |
| the same, with every theme engine too | 23.50 MB | |

**+2.16 MB** is the number to plan with: a whole application, built the
way one ships, with the shaper reachable from `main`. The rows above it
say where it goes — go-text's OpenType parser is the largest single
piece at 1.07 MB, and it is on the bill only because the toolkit still
parses every face twice, once with it and once with
`x/image/font/sfnt`. Dropping `sfnt` would give about 0.33 MB back. The
shaper itself, harfbuzz and its script tables, is 0.83 MB; bidi and
segmentation are rounding errors.

The rule from [fonts.md](fonts.md) is unchanged, and this is the proof of
it: supporting every script on earth costs two megabytes of *code*. One
Noto Sans CJK family is 18.6 MB of font data. The faces stay the
machine's.

## What is still missing

- **Vertical text** (UAX #50). The segmenter can resolve the
  orientation; nothing above it would know what to do with the answer.
- **COLR and SVG colour glyphs.** The PNG kind — `CBDT`/`CBLC` and
  `sbix`, which is what the emoji faces people have actually use — is
  drawn under `-tags font_emoji`, in a colour sheet of its own that
  [Font.Draw] leaves untinted. Layered outlines and SVG glyphs are
  parsed by go-text and not drawn.
- **Caret affinity.** At a change of direction one text position is two
  places on screen. The toolkit picks one: the leading edge of the
  character the index names.
- **Arrow keys move logically**, not visually: Left in a right-to-left
  run moves to the next character in the *string*, which is to the
  right. Both behaviours exist in the wild; this one is the one the
  toolkit already had.
- **One parser.** Keeping `sfnt` costs about 0.33 MB of the 2.16 above
  and parses every face twice.

## The dependency

`github.com/go-text/typesetting v0.3.5` — Unlicense OR BSD-3-Clause,
with the harfbuzz port under MIT. It moved `golang.org/x/image` from
v0.18.0 to v0.23.0 and `golang.org/x/text` from v0.16.0 to v0.21.0 for
every build, with the tag or without. That upgrade renders every bundled
face pixel for pixel as before, which was checked rather than assumed.
