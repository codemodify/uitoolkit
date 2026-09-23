# Paint performance (v0.15.0)

v0.14.0–v0.14.6 tried to go faster with dirty-rect clipping, pixel
`Context.Scroll`, `DrawSceneDamage`, and partial GPU present. That was
a bad idea: black first frames, HiDPI scale blow-up, vacated list
holes, stuck menu highlights, and bold→thin glyphs on hover.

**v0.14.7 restored the v0.13.8 paint path.** **v0.15.0** is the safer
second pass: less work per **full** paint, idle coalescing, and
alloc cuts. Dirty semantics stay the v0.13.8 model — dirty marks
which widgets to **re-record**; `DrawScene` always replays the whole
graph; `Surface.Present(nil)` always presents the full buffer.

## Current model

1. Dirty marks which widgets to **re-record** into the retained scene.
2. A full present of an unchanged tree **keeps** recorded groups.
   Layout, look, and a new content tree still `Reset` the scene cache.
3. `DrawScene` always replays the **whole** graph.
4. `Surface.Present(nil)` always presents the **full** buffer.
5. paintengine2d **v0.9.0** — no `DrawSceneDamage` / `BakeGroup` wiring.

## Headless profile (CI, `CGO_ENABLED=0`)

```bash
CGO_ENABLED=0 go test -bench='BenchmarkSettingsFullPaint|BenchmarkGalleryFullPaint|BenchmarkListHover|BenchmarkListScroll' -benchmem ./internal/apptest
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

## What shipped

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

`UITK_PERF_LOG=<file>` (`app/perflog.go`) is what the script reads: a line
per painted frame with the time it took and the window's GPU texture bytes,
a line per idle trim, and, on `SIGUSR1`, a heap profile and every
goroutine's stack beside it (`SIGUSR2`: five seconds of CPU profile).
