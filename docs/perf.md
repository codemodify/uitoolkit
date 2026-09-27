# Paint performance

This document is layered, newest section last, and each section is a
record of what was measured on the day it names. **The two sections
below the v0.15.0 one supersede it**: the toolkit is on paintengine2d
v0.11.0, `Window.paintRects` does hand `DrawSceneDamage` a damage list
and `Surface.Present` a rect list (with `UITK_PAINT_FULLFRAME=1` and
`tools/perf/oracle.sh` there to prove the two agree), and the numbers to
compare against are in [the 0.20.0 review](#the-0200-review-2026-09-27).
The v0.15.0 section's **hard ban** on bringing back pixel `Context.Scroll`
and on skipping `Surface.Present` still stands.

## v0.15.0

v0.14.0–v0.14.6 tried to go faster with dirty-rect clipping, pixel
`Context.Scroll`, `DrawSceneDamage`, and partial GPU present. That was
a bad idea: black first frames, HiDPI scale blow-up, vacated list
holes, stuck menu highlights, and bold→thin glyphs on hover.

**v0.14.7 restored the v0.13.8 paint path.** **v0.15.0** is the safer
second pass: less work per **full** paint, idle coalescing, and
alloc cuts. Dirty semantics stay the v0.13.8 model — dirty marks
which widgets to **re-record**; `DrawScene` always replays the whole
graph; `Surface.Present(nil)` always presents the full buffer.

### The model v0.15.0 shipped

1. Dirty marks which widgets to **re-record** into the retained scene.
2. A full present of an unchanged tree **keeps** recorded groups.
   Layout, look, and a new content tree still `Reset` the scene cache.
3. `DrawScene` always replays the **whole** graph.
4. `Surface.Present(nil)` always presents the **full** buffer.
5. paintengine2d **v0.9.0** — no `DrawSceneDamage` / `BakeGroup` wiring.
   (Points 3, 4 and 5 no longer describe the toolkit: see the note at the
   top of this file.)

### Headless profile (CI, `CGO_ENABLED=0`)

```bash
CGO_ENABLED=0 go test -bench='BenchmarkSettingsFullPaint|BenchmarkShowcaseFullPaint|BenchmarkListHover|BenchmarkListScroll' -benchmem ./internal/apptest
CGO_ENABLED=0 go test -bench=BenchmarkFontAdvanceCached -benchmem ./style
CGO_ENABLED=0 go test -bench=BenchmarkLookWatchSettled -benchmem ./app
```

Wayland dogfood (`UITK_SCENE=auto` with a real application + pprof / perf)
is still the place to confirm HiDPI; these numbers are CPU `DrawScene`
of the offscreen pixmap.

Median of 3 runs on the agent host (Xeon, `CGO_ENABLED=0`):

| Bench | v0.14.7 | v0.15.0 |
| --- | --- | --- |
| Mail full paint (1280×800, historical — Mail now lives in [comms-mail](https://github.com/codemodify/comms-mail); `BenchmarkSettingsFullPaint` is this repo's heavy window) | 63.8 ms · 1305 KB · 8555 allocs · 48 reused | 62.6 ms · 36 KB · 87 allocs · 1 reused (root attach) |
| Gallery full paint (1100×720) | 18.1 ms · 106 KB · 690 allocs · 0 reused | 18.1 ms · 35 KB · 84 allocs · 1 reused |
| List hover | 778 µs · 20 KB · 142 allocs | 777 µs · 18 KB · 113 allocs |
| List scroll | 1.23 ms · 70 KB · 508 allocs | 1.22 ms · 54 KB · 305 allocs |
| `Font.Advance` warm string | 671 ns · 760 B · 6 allocs | 165 ns · 0 B · 0 allocs |
| WatchLook settled file | 2740 ns · 1000 B · 6 allocs | 629 ns · 288 B · 2 allocs |

`DrawScene` raster of the full mailbox still dominates wall time.
The win is recording: v0.14.7 dropped the scene cache on every full
invalidate (only virtualized rows survived), so Mail rebuilt ~8k
nodes per present. v0.15.0 attaches the recorded root.

### What v0.15.0 shipped

1. **Less work per full paint** — `fullInvalidate` no longer
   `SceneCache.Reset`. `frame` drops the cache only after Measure /
   Arrange (resize, `RequestLayout`, `SetLook`, `SetContent`). Hover
   still `Invalidate`s and does **not** `RequestLayout`. `Font`
   reuses `NullShaper` runs (cap 512) for Advance / InkWidth / Draw.
   Row keep-maps and scene-cache maps are pooled / `clear`d.
2. **Idle coalescing** — `Run` / `PumpOnce` drain every surface
   (up to 64 `Poll`s) then paint each window once. WatchLook is
   Stat-first again: idle polls size+mtime and `ReadFile`s only on
   a meta change (1s settle window for coarse overlay clocks).
3. **Alloc cuts** — no paintengine2d bump. `Recorder` has no `Reset`
   in v0.9.0, so recorders are not pooled. Present stays `nil`
   (no `[]Rect` copies).

## Hard ban (until 1× + 2× goldens exist for the *changed* path)

`internal/uitest.TestChromeStripGolden` locks a button + checkbox +
menu-bar strip at **1× and 2×** (`testdata/chrome-strip-1x.png`,
`chrome-strip-2x.png`, plus `PaintHash`). That is a *control-strip*
guard, not a license to bring back dirty-rect present.

Do **not** ship again:

- `Context.Scroll` blit + strip-only damage
- `DrawSceneDamage` that skips ops
- skipping `Surface.Present` on GPU
- ClearRect of a hover box that contains static text, unless a golden
  test at 1× **and** 2× shows identical glyphs vs a full paint

Dirty-rect / `DrawSceneDamage` / pixel scroll stay banned until those
full-frame 1×+2× goldens exist for the scene they would clip. Update
strip goldens with `UITK_UPDATE_GOLDEN=1`.

## Memory, start-up, frames and idle (2026-09-22)

Measured with `tools/perf/measure.sh`, which drives the nested-KWin rig
(`tools/e2e`) on the real GPU and prints the table below in one command:

```bash
tools/perf/measure.sh 31 1.75     # every app, at a real 1.75 output scale
tools/perf/measure.sh 31 1        # the same at 1x
tools/perf/oracle.sh 31           # partial redraw against UITK_PAINT_FULLFRAME=1
```

The machine: Arch, KDE Plasma 6 / KWin 6.7.5, kernel 7.2.6, Intel Core Ultra
7 265U with its Arrow Lake graphics (Mesa 26.2.3), Go 1.27.1. The instance is
a 1280×860 logical screen at whichever scale is asked for — 2240×1505 device
pixels at 1.75 — so every window opens in the same place and the scripted
pointer lands on the same controls at both scales. Each app runs alone, and
the load average stands beside its row in the results file.

What each column is. **own** is settled `Private_Dirty` from
`/proc/<pid>/smaps_rollup` 10 s after the launch, as on 2026-09-15, and the
median of three launches: one launch lands 2–3 MB either side of another,
depending on where the heap falls and when the collector last ran. **start**
is from the `exec` to the end of the first painted frame. The interaction
columns are the median frame time (`app/perflog.go` times every frame): a
hover sweep of 72 pointer moves across the window, a scroll of eight wheel
notches each way, a menu or popup opened and closed three times, and two
theme switches (`look.json` to Breeze Plasma 6 and back to Metal Ocean).
**idle** is CPU time and wake-ups (context switches over every thread) in 10
quiet seconds. **GPU tex** is the most image-texture memory one window's GPU
device held.

| app | own 1× | own 1.75 | 1.75 on `dev` | 1.75 on 2026-09-15 | start | hover | scroll | menu | theme | idle 10 s | GPU tex |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Settings | 33.3 | 31.3 | 32.8 | 29.8 | 124 ms | 1.9 ms | 2.3 ms | 1.2 ms | 4.8 ms | 0 ms / 14 | 0.4 MB |
| gallery | 31.1 | 29.5 | 39.4 | 36.4 | 115 ms | 1.5 ms | 3.3 ms | 1.2 ms | 1.6 ms | 0 ms / 197 | 0.8 MB |
| Mail | 31.0 | 30.7 | 32.0 | 30.8 | 163 ms | 2.0 ms | 6.6 ms | 1.6 ms | 14.9 ms | 0 ms / 60 | 0.8 MB |
| Files | 28.3 | 31.0 | 30.3 | 27.8 | 108 ms | 1.9 ms | 1.9 ms | 0.8 ms | 8.1 ms | 0 ms / 0 | 0.5 MB |
| Notes | 27.3 | 28.8 | 27.3 | 28.2 | 98 ms | 0.8 ms | 0.8 ms | 0.9 ms | 0.9 ms | 0 ms / 5 | 0.7 MB |
| Inspector | 26.6 | 28.6 | 27.0 | 27.1 | 93 ms | 1.6 ms | 1.8 ms | 1.1 ms | 3.3 ms | 0 ms / 0 | 0.4 MB |
| tour | 30.5 | 32.0 | 30.0 | — | 107 ms | 2.4 ms | 2.3 ms | 1.2 ms | 13.0 ms | 0 ms / 0 | 0.5 MB |
| Minim | 43.4 | 47.2 | 47.7 | — | 121 ms | 2.8 ms | 2.9 ms | 3.1 ms | 3.4 ms | 1950 ms / 26357 | 1.0 MB |
| Marquee | 38.8 | 56.4 | 66.3 | — | 194 ms | 3.7 ms | 3.8 ms | 3.9 ms | 3.6 ms | 730 ms / 15043 | 9.4 MB |
| Lantern | 45.0 | 63.3 | 80.8 | — | 198 ms | 2.1 ms | 2.0 ms | 1.8 ms | 2.0 ms | 820 ms / 16426 | 16.7 MB |
| rich text | 30.6 | 33.7 | 32.0 | — | 119 ms | 1.8 ms | 1.8 ms | 1.3 ms | 2.1 ms | 10 ms / 11 | 3.1 MB |
| MDI | 29.6 | 29.3 | 31.4 | — | 110 ms | 1.2 ms | 1.4 ms | 0.8 ms | 0.8 ms | 0 ms / 19 | 1.4 MB |
| wizard | 25.8 | 27.4 | 29.0 | — | 86 ms | 2.1 ms | — | — | 1.5 ms | 0 ms / 0 | 0.2 MB |

Memory in MB. The 2026-09-15 column is that day's code built and measured
again today, not the figures RESUME recorded then: they were taken on
another Mesa and another Go, and only a same-day build compares.

**Mail, Minim, Marquee and Lantern are no longer in this repository** — they
moved to [comms-mail](https://github.com/codemodify/comms-mail) and
[media-player-music](https://github.com/codemodify/media-player-music) on
2026-09-23 — and `tools/perf/apps.txt` no longer lists them, so a rerun
produces the other rows only. Their numbers are left here because they were
measured on one day beside the rest and because the conclusions below rest
on them: the players were the only apps in the set that animate, and Mail
was the heaviest window.

**An app that animates nothing is idle.** Every app but the three players
now takes no CPU at all in ten quiet seconds. The wake-ups left are the
loop's own: Mail's tray safety net (once a second), the gallery's busy bar
looking whether it has been scrolled back into sight (twice a second), and a
caret blinking out its ten seconds. The players animate by design — they are
playing — and that is what their CPU and their wake-ups are.

**A scroll costs about the same as a hover**, a menu less (its surface is
its own), and a theme switch is the expensive frame: everything is measured,
arranged and recorded again (Mail 15 ms, the tour 13 ms). A frame's budget
at 60 Hz is 16 ms.

### The partial-redraw oracle

`tools/perf/oracle.sh 31` runs every app twice — as it paints, and with
`UITK_PAINT_FULLFRAME=1` — through the same scripted input, and compares
the compositor's screenshot after each step. Every app but the three
players is clean: the odd step differs in four pixels by one or two levels
in 255, in symmetric pairs at an antialiased corner's edge, and the same
four turn up on `dev`. The players animate on their own clock, so their two
runs are never the same frame; their differences are the spectrum and the
clock, not the redraw.

### What this pass changed

| | before | after |
| --- | --- | --- |
| Files, 30 menus opened and closed | 30.8 → 38.4 MB | 29.0 → 30.9 MB |
| Notes idle, 10 s (a caret blinking) | 1315 wake-ups, 10–30 ms CPU | 0 and 0 |
| the gallery idle, 10 s | 32 109 wake-ups, 2.0 s CPU, 320 frames | 174, 0, 0 |
| Notes, 5 s of a blinking caret | 629 wake-ups | 369 |
| Lantern | 80.8 MB | 63.3 MB |
| Marquee | 66.3 MB | 56.4 MB |
| Notes in Breeze (desktop fonts) | 30.5 MB | 28.7 MB |
| Minim's frame (three animating windows) | 2.93 ms | 2.80 ms |

### Measuring it again

Three things will make an app look several MB heavier than it is, and all
three fooled this pass before the script took care of them:

1. **A binary in a tmpfs.** Every page of a tmpfs file is dirty, so a binary
   under `/tmp` counts its whole text and rodata as the process's own
   memory: 12.6 MB for Settings. Keep the binaries on a disk.
2. **A binary just built.** Its pages sit dirty in the page cache until the
   filesystem writes them back (about 30 s on btrfs), and count the same
   way: 17 MB for Files. `measure.sh` builds everything first, then `sync`s.
3. **The rig's own wait.** `tools/e2e/run.sh` waits 0.4 s for the previous
   app to go when it finds its pid file, which would otherwise be counted as
   start-up. `measure.sh` removes the file itself.

### CPU or GPU is now a preference, not only a variable

`UITK_PAINT=cpu|gpu|auto` picks the paint device and always has; from
0.20.0 the same three are saved in `look.json` as `"renderer"` and chosen
in Settings' **Renderer** box, with the variable still winning wherever it
is set. Nothing about the two paths changed — the numbers above stand —
but two things are worth knowing when measuring:

- **A run is one device.** `platform.PaintPref` is read when a surface
  binds its device, so a preference applied while an app runs reaches the
  windows it opens afterwards, not the ones already up. Measure a
  renderer by starting the app under it (the variable is the quickest
  way), not by switching it in Settings mid-run.
- **Ask, do not assume.** `auto` and `gpu` both fall back to the CPU when
  EGL will not start, and a table that recorded the preference instead of
  the device would be quietly wrong. `app.Application.PaintBackend()`
  (and `platform.SurfaceBackend`) reports what a window really presents
  through; `UITK_PERF_LOG` already carries the window's GPU texture
  bytes, which is zero on the CPU path.

`UITK_PAINT_MSAA=0` is unchanged and stays an environment variable: it is
a knob on the GPU path alone, it is invisible on a machine that came up
on the CPU, and it costs four times the pixels on llvmpipe — which is
exactly when you want it, and exactly not a thing to put on a settings
row that is already at its fold
([settings.md](settings.md#uitk_paint_msaa-is-not-on-this-page)).

`UITK_PERF_LOG=<file>` (`app/perflog.go`) is what the script reads: a line
per painted frame with the time it took and the window's GPU texture bytes,
a line per idle trim, and, on `SIGUSR1`, a heap profile and every
goroutine's stack beside it (`SIGUSR2`: five seconds of CPU profile).

## The 0.20.0 review (2026-09-27)

A second pass over start-up, idle, steady-state drawing, memory, binary
size and the shape of the widget layer, on the branch `perf/review-2`.
Four things changed, all in `style`:

1. the built-in skins are resolved before any of them is registered, so
   the era-pack index is built once at init instead of eight times;
2. `DottedRect` fills a focus ring's four edges instead of drawing 282
   one-pixel rects, and `Bumps` fills a row of dots at a time;
3. `DottedRect`, `Bumps` and `Pinstripes` borrow their paths from a pool;
4. a shaped run cached against the published glyph sheet is returned
   without walking the string.

Everything else this section names was measured and deliberately left
alone, or is a decision for the owner rather than a fix.

### What the numbers were taken on, and what they are worth

Same machine as the 2026-09-22 table (Arch, KDE Plasma 6 / KWin 6.7.5,
kernel 7.2.6, Intel Core Ultra 7 265U with Arrow Lake graphics, Mesa
26.2.3), Go 1.27.1 — but **not a quiet one**: another agent was building,
testing and running a nested-KWin rig of its own throughout, at load
averages between 4 and 15. That is the fourth measurement trap, and it is
worse than the other three because it does not bias a number, it widens
it: the same benchmark ran 258 µs and 405 µs ten minutes apart. The last
of its measurements were taken after the neighbour went quiet, and the
tables say which.

What that costs, and what it does not:

- **Deterministic counters are unaffected** and carry most of the
  conclusions here: `B/op` and `allocs/op` from `-benchmem`, bytes and
  allocations from `GODEBUG=inittrace=1`, `Scene.Nodes`, draw-op counts,
  section sizes in the binary. Every one of those repeats exactly.
- **Wall clock needs interleaving.** Where a time is quoted as a
  before/after it was taken by running the two builds alternately, many
  times, and comparing medians and quartiles — never two runs an hour
  apart. Where the interleaved bands overlap, this says so.
- **The rig table below is an absolute baseline, not an A/B.** One
  launch's `Private_Dirty` sits 2–3 MB either side of another's even on a
  quiet machine, and start-up on a loaded one varies by 4×: Files
  measured 87 ms on one pass and 358 ms on the other with no code between
  them. Read the rig for idle, for frame times and for the order of
  magnitude of memory, not for a 1 MB delta.

A fifth trap, new and specific to the paint benchmarks:

- **`-benchtime` decides what a paint benchmark measures.** At
  `-benchtime=200x` the window, look and font sheets built in the
  fixture are amortized over too few iterations, and the collector barely
  runs; a change that cuts the garbage per frame by five sixths then
  looks like no change at all, or like a small loss. The regime a real
  application is in is the one where the allocation rate matters, so the
  frame benchmarks here are quoted at `-benchtime=4000x`, and the before
  and after were run alternately at the same count.

### Start-up: what a process pays before main

`GODEBUG=inittrace=1` on a binary that imports `app`, `style` and
`widgets` and does nothing:

| | before | after |
| --- | --- | --- |
| `style` package init | 11 898 896 B, 63 892 allocs | 5 184 464 B, 34 901 allocs |
| whole process at main | 12 003 KB, 64 469 mallocs | 5 445 KB, 35 479 mallocs |
| collections before main | 5 | 2 |
| exec → exit, wall | 15.7 ms (p25 14.7, p75 16.5) | 11.4 ms (p25 10.5, p75 12.4) |
| exec → exit, CPU | 22 ms | 13 ms |

Medians of 61 alternated runs. Against a first frame that lands between
79 ms and 150 ms in the rig, package init was 10–13 ms of it and is now
6–8.

The cause was not the 129 packs or the 33 engines as such. Building every
engine's packs — every `…Packs()` function in `style`, called back to
back — is 1.0 ms, 0.8 MB and 1200 allocations all told, and the 33
`RegisterEngine` calls are a map insert each. What cost was **the
built-in skins' registration doing the whole job eight times**: a skin's
`Pack` resolves it against its base pack, which builds the era-pack index
(129 packs, each `Resolve`d, plus the embedded starters parsed out of the
binary), and `RegisterPack` invalidates the index the moment it is
handed one. Resolving all eight skins before registering any of them
builds it once. That is the first commit, and it is the whole of the
table above.

What is left, for whoever looks next: `loadBuiltinSkins` is 2.2 MB in
22 400 allocations — 42% of the bytes and 64% of the allocations still
spent in `style`'s init — and it happens whether or not the application
will ever paint a skin.

### Steady-state drawing: the focus ring was 90% of a hovered frame's recording

`internal/apptest`, `CGO_ENABLED=0`, `-benchtime=4000x`, three rounds of
the two builds run alternately, on a machine at load 3 — every figure
below is the three values, not a mean:

| bench | before | after |
| --- | --- | --- |
| list hover, time | 245.1, 245.2, 248.0 µs | 152.9, 152.8, 154.3 µs |
| list hover, garbage | 189.9 KB/op, 88 allocs | 30.0 KB/op, 69 allocs |
| list scroll, time | 500.9, 505.9, 501.0 µs | 435.9, 432.8, 453.0 µs |
| list scroll, garbage | 280.1 KB/op, 90 allocs | 18.4 KB/op, 60 allocs |
| Settings full paint 1280×800 | 5.51, 5.39, 5.31 ms | 5.36, 5.44, 5.33 ms |
| showcase full paint 1100×720 | 8.64, 8.83, 8.77 ms | 8.64, 8.54, 8.83 ms |

A hover frame is 37% quicker and makes a sixth of the garbage; a scroll
frame 12% quicker and a fifteenth. The two full paints do not move at
all, and their bands overlap completely: they are rasterizing, and none
of this pass touched that.

The recorded scene of a hovered 280×220 list window went from **533
re-recorded nodes a frame to 51** — the rows themselves were already
being attached from `rowSceneCache` rather than rebuilt, so what was
left was almost all one ring. 482 of those 533 were its dots:
`DottedRect` drew each dot with its own `DrawRect`, so a ring round a list
row was 282 draw ops, and `style.DrawFocusRing` — which every engine ends
in — was 29% of the frame's CPU in a profile. `Pinstripes` two functions
below it already filled all its stripes from one path, with a comment
saying why. Now `DottedRect` does the same, an edge at a time, and
`Bumps` a row at a time; all three borrow their path from a pool.

Four paths and not one, because of the sixth trap:

- **Fewer draw ops is not free on the CPU rasterizer.** paintengine2d
  takes a multi-rect fill through an analytic fast path guarded by
  `pixelDisjointRects`, whose sweep sorts the boxes by their top edge and
  stops at the first that begins below the current one — so a run of
  rects that all share a row never stops early, and 130 dots in one path
  cost 8450 overlap tests. Filling the whole ring from one path made the
  immediate CPU device 3.3× slower (33 µs → 110 µs); splitting it into
  four one-pixel-thick edges did not help, because the cost is the sweep
  and not the bounding box. Sorting by X within a row would make it
  linear; that is upstream.

So the trade was measured on both devices, replaying the same recorded
window scene. `tools/perf/replay` is that measurement, kept:
`paintengine2d.NewGPUDevice` renders surfacelessly, so it needs a DRM
render node but no compositor and no display, and it runs through
`tools/testenv.sh` like everything else.

```bash
tools/testenv.sh go run ./tools/perf/replay all
```

| hovered 280×220 list window | nodes re-recorded | CPU replay | GPU replay |
| --- | --- | --- | --- |
| before | 533 | 200–208 µs | 262–266 µs |
| after | 51 | 243–249 µs | 209–214 µs |

Recording the ring itself fell from 92 µs / 167 KB / 1170 allocations to
19 µs / 25.8 KB / 26. On the GPU — which is what `UITK_PAINT=auto`, the
default, picks whenever EGL starts — both halves win. On the CPU the
recorder saves about what the rasterizer loses, and the end-to-end
benchmark above says the frame still comes out ahead once the collector
is in the picture.

`Bumps` over a 320×200 panel, which is public API no app in this
repository calls: 16 000 draw ops and 10.3 MB to record become 200 and
752 KB, against 2.9 ms of rasterizing becoming 3.9.

`style/shapes_bench_test.go` holds the four micro-benchmarks, recording
and rasterizing, so the next person can see both sides:

```bash
tools/testenv.sh env CGO_ENABLED=0 go test -run=X -bench='DottedRect|Bumps' -benchmem -count=6 ./style
tools/testenv.sh env CGO_ENABLED=0 go test -run=X -bench='List|FullPaint' -benchmem -benchtime=4000x -count=5 ./internal/apptest
```

Everything in this section that is a `go test` went through
`tools/testenv.sh`, which is not optional: a bare run reaches the
desktop it runs on.

### Measuring text: the warm path walked the string

`Font.Advance` is what a layout pass is made of, and every call went
through `otAtlas.ensure`, which asks the published glyph sheet whether it
holds each rune of the string — a map lookup a rune, on a run that may
have been shaped a thousand frames ago. A run cached against the sheet
that is still published needs none of that: cells are only ever added to
a sheet, and a new sheet is published as a new pointer, so pointer
identity is enough to know the cached run is still good. `shapeOf` now
checks the shape cache first.

| | before | after |
| --- | --- | --- |
| `BenchmarkFontAdvanceCached` (28 characters, warm) | 173–330 ns | 15.9–17.2 ns |
| Settings' Measure + Arrange pass (1280×800) | 0.48–0.88 ms | 0.12–0.19 ms |
| showcase's Measure + Arrange pass (1100×720) | 0.15–0.25 ms | 0.10–0.15 ms |

Three interleaved runs each. It does not move a full paint —
`BenchmarkSettingsFullPaint` sits at 5.8–6.0 ms either way — because a
full paint is rasterizing, not measuring; it pays on the frames that
relayout, which are resize, theme change and content swap, and those are
already the slowest frames there are.

For scale, one Measure + Arrange of Settings puts 8946 child
measurements through `layout.Flex`, for a tree of 85 components 16 deep.

### What a frame is actually made of

A counting `paintengine2d.Device` replaying a recorded scene says what
each window's ops are:

| window | ops | one-glyph blits | opaque aligned rects (memset) | path fills | other |
| --- | --- | --- | --- | --- | --- |
| Settings 1280×800 | 1029 | 770 (112 Kpx) | 144 (3.87 Mpx) | 65 (1.74 Mpx) | 23 strokes, 14 gradient rects (104 Kpx), 11 off-grid rects, 1 translucent rect (405 Kpx) |
| showcase 1100×720 | 1622 | 1440 (186 Kpx) | 100 (0.81 Mpx) | 38 (1.58 Mpx) | 23 strokes, 3 translucent rects (856 Kpx) |
| list 280×220 | 99 | 72 (11 Kpx) | 18 (124 Kpx) | 8 | — |

(Each window as it opens, with nothing hovered and no focus ring drawn.)

And what replaying each of them costs, three runs of
`tools/perf/replay all 31` on a machine at load 3.7. Its op counts are
of the scene graph and so run a little above the table above, which
counts the ops that reached the device — the rest are clipped away on
the way:

| window | ops | CPU replay | GPU replay |
| --- | --- | --- | --- |
| showcase 1100×720 | 1679 | 8.60–8.75 ms | 1.58–1.67 ms |
| Settings 1280×800 | 1029 | 5.11–5.43 ms | 2.28–2.42 ms |
| list 280×220, hovered | 104 | 246–260 µs | 212–223 µs |
| list 280×220 | 99 | 209–218 µs | 206–232 µs |

**The GPU path is 2.2× to 5.3× faster than the CPU rasterizer on the
windows big enough to matter**, and indistinguishable on a small one,
where the fixed cost of a frame is the whole of it. That is the ratio a
`renderer` setting of `cpu` gives up, and it is the reason the draw-op
trade above was resolved in the GPU's favour.

Two things fall out of that, and neither is a bug in this repository:

1. **Text is one draw op a glyph.** `Context.DrawGlyphs` issues one
   `Blit` per glyph, and the GPU device batches consecutive opaque rect
   fills but not blits — so three quarters of Settings' ops, and seven
   eighths of the showcase's, are one-glyph GPU draw calls. A batched
   glyph-run op in the `Device` interface is the single biggest
   structural saving available to the GPU path, and it is an upstream API
   change.
2. **A window is painted about six times over.** Settings puts 6.1 Mpx
   into a 1.02 Mpx window, the showcase 3.25 Mpx into 0.79. Most of it is
   the memset path, which is why it is survivable, but nested containers
   each filling their own opaque background is what that overdraw is.

### Idle, memory and frames on the rig

`tools/perf/measure.sh 31 1`, nested KWin, 1280×860 logical at 1×, on the
GPU, on the branch. The load average beside each row is what the machine
was doing at the time; see the caveat above before reading a delta out of
this table.

| app | load | start | own MB | rss MB | idle 10 s | hover | scroll | menu | theme | gor | GPU tex |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Settings | 8.5 | 139 ms | 27.9 | 83.4 | 0 ms / 0 | 2.5 ms | 4.6 ms | 1.1 ms | — | 7 | 0.2 MB |
| Files | 6.4 | 87 ms | 28.0 | 82.4 | 0 ms / 0 | 2.7 ms | 2.0 ms | 1.0 ms | 16.8 ms | 8 | 0.3 MB |
| Notes | 4.9 | 79 ms | 25.6 | 79.8 | 0 ms / 6 | 1.1 ms | 0.9 ms | 1.1 ms | 0.7 ms | 8 | 0.3 MB |
| Inspector | 5.0 | 91 ms | 25.5 | 79.2 | 0 ms / 0 | 2.9 ms | 1.3 ms | 2.3 ms | 6.7 ms | 8 | 0.4 MB |
| tour | 6.7 | 134 ms | 25.4 | 82.4 | 0 ms / 145 | 3.8 ms | 3.0 ms | 1.5 ms | 13.6 ms | 8 | 0.3 MB |
| rich text | 10.2 | 114 ms | 29.2 | 84.7 | 0 ms / 39 | 2.2 ms | 2.5 ms | 1.6 ms | 2.2 ms | 8 | 1.2 MB |
| MDI | 4.1 | 89 ms | 28.4 | 83.8 | 0 ms / 33 | 1.3 ms | 1.1 ms | 0.9 ms | 1.4 ms | 8 | 0.6 MB |
| wizard | 7.7 | 150 ms | 24.0 | 79.1 | 0 ms / 0 | 2.5 ms | — | — | 2.6 ms | 8 | 0.2 MB |

The frame columns are medians. Settings' theme cell is empty rather than
zero: it recorded no frame inside the marked stretch on either pass, and
it is the one app in the set that owns a theme UI of its own, so the
script writing `look.json` underneath it is not the event the other rows
measure. The same pass over binaries built from
`dev` (`PERF_BIN=… PERF_TAG=base`) came out at 24.8–30.4 MB of own
memory and 80–358 ms of start-up against this pass's 24.0–29.2 MB and
79–150 ms — overlapping bands on a machine this busy, which is why the
start-up claim above rests on the `inittrace` A/B and not on this.

`gor` is the goroutine count. Every app trimmed its idle heap about 2.1 s
after the last activity, as designed.

**Idle is still free.** Every app takes 0 ms of CPU in ten quiet seconds.
The wake-ups that remain are the caret's: the run loop blinks a focused
field for ten seconds after the last input and then stops, and the idle
window is measured from the launch, which is why the tour (whose first
page opens on a field) shows 145 and Files, whose first focus is a list,
shows none. An app left alone for a minute wakes for nothing at all — the
loop's wait is `-1` (block on the Wayland fd) when no deadline is owed.

**A theme switch is still the expensive frame** and the only measurement
in the set that goes past a 60 Hz budget — Files 16.8 ms, the tour
13.6 — because it re-measures, re-arranges and re-records everything.
The layout part of that is now 0.1–0.2 ms (above); the rest is the
recording and the raster.

### The widget layer's shape

- **Layout does not run on a frame.** `Window.frame` measures and
  arranges only when `!w.laid`, which is a resize, `RequestLayout`,
  `SetLook` or `SetContent`. Hover and scroll never relayout.
- **Hover and scroll are O(visible), not O(tree).** A `ListView` of 80
  rows and one of 2000 record the same 886 ops, and rows are attached
  from `rowSceneCache` by content signature rather than re-recorded. The
  earlier `TreeView.flatten` cache does the same for trees.
- **A layout pass measures a subtree once per ancestor.** `layout.Flex`
  measures each child in `Measure` and measures it again in `Arrange`,
  and there is no measure cache on `widget.Base`, so every `Arrange` down
  the tree re-measures everything under it. Settings is 85 components 16
  deep and one pass makes **8946** child measurements through
  `layout.Flex`; the showcase is 225 components 13 deep and makes 14 238.
  A measure cache keyed on the constraints would make that O(nodes). It
  is worth 0.1–0.2 ms a relayout today (see the table above), which is
  why this pass made `Advance` cheap instead of touching the layout
  contract before a tag.
- **`SceneCache` reuse asks `subtreeDirty`**, which walks a whole subtree
  per cache probe, so a re-record along a dirty path costs O(nodes ×
  depth) map lookups as well. Settings is 106 groups at depth 17; it does
  not register.

### Memory: what is retained and what is bounded

Every cache in the toolkit was read for a bound. The shaped-run cache is
512 entries (flushed whole, not LRU), the font cache is an LRU on a byte
budget of live glyph sheets, `paintengine2d`'s path cache sweeps on a
frame clock and caps at 8192, the skin hit and pixel-grid caches have
maxima, `rowSceneCache` caps at 256 rows, `RichText`'s layout cache
prunes at `2·blocks+256`, the window shadow patches are reference-counted
by the windows that use them, and `Classic.Memo` is keyed by empty
structs — about 35 entries for the life of a look.

The one cache with no bound at all is **`style.iconsCache.entries`**: one
decoded image per icon file path, dropped only when the whole map is
replaced by `InvalidateIconCache` on a theme or look change. No
application measured here comes close to mattering (Files settles at
28 MB with 0.3 MB of GPU texture), but nothing stops an app that draws an
icon for every MIME type in a large directory.

Start-up allocates 12 MB and retains 1.6 MB of it after a collection, so
the init cost was churn rather than footprint; the branch now allocates
5.4 MB there.

The three locks this pass touched — the shaped-run cache read on the
warm path, the path pool, the skin registry's two-pass init — were
checked under the detector:

```bash
tools/testenv.sh go test -race -count=1 -timeout=40m ./app/... ./widget/... ./widgets/... ./internal/... ./style/
```

No data race in any of them. `style` takes 1070 s under `-race` against
88 s without, which is over the `go test` default timeout of 10 minutes,
so that run needs `-timeout` raised or it fails on the clock rather than
on a finding.

### Binary size: the engines are the binary

The Notes sample, `go build` with no flags:

| | bytes |
| --- | --- |
| whole file | 28.58 MB |
| `-ldflags=-s -w` | 20.30 MB |
| `.text` | 10.59 MB |
| `.gopclntab` | 7.19 MB |
| DWARF | 6.79 MB |
| `.rodata` | 1.88 MB |
| `.go.type` | 1.25 MB |

**The `style` package is 5.84 MB of the 10.03 MB of machine code** — 58%
of it — at roughly 100–240 KB an engine. Embedded assets are not the
story: the four TTFs are 668 KB, the eight skins' manifests and art
687 KB, the Lucide icons 74 KB and the three starter packs 12 KB, 1.44 MB
in all.

Because every engine registers itself from `init()`, the linker cannot
drop one, and an application that uses a single theme still carries all
33. Rebuilding Notes with 25 of them removed (a measurement, not a
proposal): **28.58 MB → 18.50 MB**, `.text` 10.59 → 5.93 and
`.gopclntab` 7.19 → 4.64.

### Handed back, not changed

Five things this pass measured and deliberately left for the owner,
because each is a decision about the API or about a trade rather than a
fix:

1. **The engine registry defeats dead-code elimination.** 10.1 MB of a
   28.6 MB binary, measured. Making the engines opt-in — a blank-import
   package of them all as the default, or build tags — is an API change
   and a packaging decision, not a patch.
2. **The eight built-in skins are parsed at init**, 2.2 MB in 22 400
   allocations, whether or not the application will ever paint one. They
   cannot simply be made lazy, because every application resolves the
   default pack at start-up and that builds the index the skins are in;
   deferring them means taking skins *out* of the era-pack index and
   finding them on the way through `LoadTheme` instead, which changes
   what the theme browser lists and when.
3. **`style.iconsCache.entries` has no bound.** A budget like the font
   cache's is the obvious answer; what the budget should be is a
   decision.
4. **`layout.Flex` has no measure cache**, which is the 8946
   measurements above. Adding one means deciding what invalidates it,
   and that is the layout contract.
5. **Text is one draw op a glyph, and the GPU batches rects but not
   blits** — three quarters of Settings' draw calls. That is a
   paintengine2d `Device` change, as is `pixelDisjointRects`' quadratic
   sweep over same-row rects.
