# Fonts: what ships, what comes from the machine, what it costs

Everything about type in this toolkit comes down to one decision nobody
has to make twice: **the toolkit embeds two Latin faces and nothing
else, and borrows the rest from the machine it runs on.** This page says
what that means for an application, what the build tags change, and
where the megabytes actually are.

## What ships in your binary

Four files, two families, two weights each:

| file | size | role |
| --- | --- | --- |
| `TitilliumWeb-Regular.ttf` | 0.06 MB | `style.FamilyUI`, the interface face |
| `TitilliumWeb-Bold.ttf` | 0.06 MB | |
| `JetBrainsMono-Regular.ttf` | 0.26 MB | `style.FamilyMono`, the monospace face |
| `JetBrainsMono-Bold.ttf` | 0.26 MB | |

**0.65 MB in total**, and they do not cover the same ground:

| | Titillium Web | JetBrains Mono |
| --- | --- | --- |
| Latin, accented Latin | yes | yes |
| Greek, Cyrillic | **no** | yes |
| CJK | **no** | **no** |
| `✓ ✗ → ← └` and box drawing | **no** | yes |

They are the floor, not the ceiling: a pack that asks for Tahoma or
Lucida Grande gets it when the machine has it, and falls back to these
when it does not. That is why a window looks like Windows 2000 on a
machine with Tahoma installed and like Windows 2000 in Titillium on one
without.

## How a face is chosen

Every pack states a chain of family names — the era's own face first,
then the look-alikes a machine is likely to have:

```go
segoe = []string{"Segoe UI", "Selawik", "Open Sans", "Noto Sans", "Liberation Sans"}
```

The user's own choice (`look.json`'s `fontUI` / `fontMono`, the font
page in Settings) is **prepended** to that chain, and
`style.ResolveFont` walks it and takes the first family that is bundled
or installed. Nothing is ever *matched* — the toolkit asks `fc-list`
what exists and compares names, and never asks `fc-match`, which is
required to answer and would hand back Noto Sans for every family on
earth.

`UITK_SYSTEM_FONTS=0` turns installed fonts off for a run: every pack
then reads the bundled two. It is what makes a screenshot reproducible,
and what a container or an appliance sets when "it looks different on
that machine" is a defect rather than a feature. One value is special —
`0`; anything else, including unset, leaves installed fonts on.

## A rune the face has not got

By default it is a box, and the box is the same box for every missing
rune. Two different CJK titles are then the same row of boxes, pixel for
pixel — a list showing two identical rows for two different things,
which is how a music player found this.

```sh
go build -tags font_fallback ./...
```

borrows instead: a rune the chosen face lacks is taken from the first
family in `style.FallbackFamilies()` that has it, baked to the borrowing
face's baseline so it sits on the line. The built-in list names the
usual CJK and wide-coverage families; an application that knows better
says so:

```go
style.SetFallbackFamilies("Noto Sans CJK JP", "Noto Sans")
```

which matters more than it looks: the Han glyphs of the SC, TC, JP and
KR families genuinely differ, and a Go string carries no language to
choose by. Naming none puts the built-in list back.

**This is fallback and nothing else.** No shaping, so Arabic stays
unjoined and Indic unreordered; no bidi, so Hebrew stays in logical
order; no line breaking for scripts without spaces. Scripts that need
none of those — CJK, Cyrillic, Greek — are what it is for.

For the rest there is a second tag:

```sh
go build -tags font_complex ./...
```

which reads the face's own tables: ligatures and joining forms, GPOS
kerning, bidirectional ordering, and the Unicode line breaking
algorithm. It implies `font_fallback`, and costs 2.16 MB of code and no
font data. [complex-text.md](complex-text.md) is what it does,
what it costs and what it still does not do; [the text
contract](contracts.md) is what a build without it promises.

## Embedding your own

An application embeds whatever it likes and names it in the chain:

```go
//go:embed fonts/AcmeSans-Regular.ttf
var acme embed.FS
```

Today the toolkit has no call to register that face, so the practical
route is to install it with the application, or ship it to the user's
font directory, and name it in `look.json`'s `fontUI`. A registration
API is a known gap rather than a documented capability — say so rather
than working around it quietly.

## Where the megabytes are

Measured on `cmd/uitoolkit-settings`:

| | size | note |
| --- | --- | --- |
| stripped, default engines | **13.03 MB** | what an application ships |
| **unstripped** | 18.71 MB | `-ldflags "-s -w"` saves **5.68 MB** |
| every theme engine (`theme_engine_all`) | 21.31 MB | **+8.3 MB** over the default |
| `-tags font_fallback` | +0.01 MB | ten kilobytes of code; the faces are the machine's |
| `-tags font_complex` | +2.16 MB | shaping, bidi and line breaking; still no font data |
| the four embedded faces | 0.65 MB of the above | |
| *one* Noto Sans CJK family | **18.6 MB** | which is why it is not in there |

So, to make a binary smaller, in the order the bytes come off:

1. **Strip it.** `-ldflags "-s -w"` — 5.68 MB, and the single biggest
   thing most applications are not doing.
2. **Name only the engines you ship.** `theme_engine_all` is 8.2 MB over
   the default; three engines is 0.45 MB over it. See
   [engines.md](engines.md).
3. **Leave the icon packs you do not use.** Only a small subset is
   embedded; the five premiere packs on disk (`icons/`) are rendered
   assets an application opts into.
4. **Do not reach for fonts.** The embedded faces are 0.65 MB of a
   13 MB binary — 5% — and dropping them would leave a machine with no
   installed fonts drawing nothing at all. `font_complex` is +2.16 MB of
   *code* and no font data: the scripts come from the machine either
   way ([complex-text.md](complex-text.md)).

The rule underneath all of it: **binary size scales with font data, not
with the number of scripts supported.** Shaping is an algorithm over
tables the font carries, so it does not grow when you add a language.
Fonts are 18 MB a family, so they are never ours to ship.
