# Theme engines are opt-in

131 theme packs are drawn by 35 engines, and an application that wants
three of them should not carry all thirty-six. Engines are chosen at
**build time**, with build tags.

```sh
go build ./...                              # the default engine alone
go build -tags theme_engine_oxygen ./...    # Oxygen instead of it
go build -tags "theme_engine_oxygen,theme_engine_beos" ./...
go build -tags theme_engine_all ./...       # every engine there is
```

Four rules, and they compose:

1. **Name an engine and you get that engine.** `theme_engine_<id>`, as
   many as you like.
2. **Name none and you get the default**, `plastik` — KDE 3's default
   style — so a build always has a working look and a theme list that is
   not empty.
3. **Name one and the default steps aside.** It is there to be a sensible
   answer when nothing was asked for, not to be carried by everyone.
   `theme_engine_plastik` keeps it alongside the others.
4. `theme_engine_all` is every engine.

Measured on `cmd/uitoolkit-settings`, stripped (`-ldflags "-s -w"`):

| build | engines | size |
| --- | --- | --- |
| default | 1 | **12.71 MB** |
| `theme_engine_oxygen` | 1 | 12.88 MB |
| three engines | 3 | 13.29 MB |
| `theme_engine_all` | 35 | 21.15 MB |

**8.4 MB**, 40% of the binary, for an application that wants one look.

## How the tags are written

Every engine carries:

```go
//go:build theme_engine_all || theme_engine_oxygen
```

and the default engine carries the negation of all of them:

```go
//go:build theme_engine_all || theme_engine_plastik ||
//         (!theme_engine_adwaita && !theme_engine_aero && ... )
```

which is what makes rule 3 work: naming any engine makes one of those
terms false and the default drops out on its own.

An engine built *on* another brings it along, because that is what it
means to be built on it: `theme_engine_bluecurve` also builds Clearlooks,
`theme_engine_kde1` also builds Windows 95 (KDE 1's widgets are Qt 1's,
which are Windows 95's), `theme_engine_breeze6` brings Breeze,
`theme_engine_adwaita48` brings Adwaita, `theme_engine_macos_tahoe`
brings macOS, `theme_engine_material_expressive` brings Material.

Build tags cannot contain hyphens — `//go:build theme-engine-oxygen` is a
syntax error, `parsing //go:build line: invalid syntax at -` — so the
names use underscores throughout.

## The kits: what belongs to no engine

Engines could not be separated at first, and the reason was always the
same: a helper written inside whichever engine happened to need it first,
which every later engine then called. `snap`, a function that rounds to
the pixel grid, lived in `engine_win95.go` and had **1001 call sites
across 35 engines and the core**. While it lived there, no build could
leave Windows 95 out.

So there are kit files, with no build tag, on one rule: **anything more
than one engine needs belongs to all of them.**

| file | what |
| --- | --- |
| `enginekit.go` | `snap`, and the odds and ends |
| `enginekit_win.go` | the Windows chrome: bevels, glyphs, wells, dotted grips |
| `enginekit_color.go` | OKLab/OKLCh, the accent shifts, percentage lighten and darken |
| `enginekit_pixel.go` | the raster port System 7, the Amiga and BeOS draw through |
| `enginekit_kde3.go` | the KDE 3 vocabulary Keramik, Plastik and KDE 2 share |
| `enginekit_web.go` | the parts the web-derived design systems have in common |
| `enginekit_draw.go` | small drawing helpers |

Finding them is not a reading exercise. Static analysis said 382
declarations were shared and every engine's dependency closure was 36 of
42 — both badly wrong, inflated by method names and local variables that
happen to be called `ok`, `set`, `line`, `rect`. **The compiler is the
only reliable instrument**: tag an engine, build, and read what it says is
missing. `tools/engines-build.sh` does that for every engine, and is how
a new entanglement gets caught.

## Testing a toolkit whose parts are optional

`tools/test.sh ./...` runs the suite with `theme_engine_all`, and that is
the suite that covers what this repository ships. A plain
`go test ./...` tests the *default product build* — one engine — where a
test written about what Aqua paints has nothing to assert; dozens of
tests are like that, and they skip.

The narrowed builds are covered on purpose rather than by accident:

- `tools/testenv.sh go test ./style/` — the default, one engine;
- the same with `-tags theme_engine_oxygen` — somebody else's pick;
- `tools/engines-build.sh` — every engine, built on its own.

## What an engine leaving takes with it

An engine registers itself and its packs from `init()`
([style.RegisterEngine], [style.RegisterPack]). Leave it out and both go:
its packs are not in `ListThemes`, and Settings does not offer them.

A pack names its engine **by string** in `theme.json`, so a name can
outlive the code that paints it. That is deliberate — a user's `look.json`
naming an engine this build does not have must not be fatal — and
`engineFor` falls back to the base engine. `TestUnbuiltEngineFallsBackToBase`
pins it.

## What is not separable yet

Three engines have to be always-built, and the reasons are worth knowing
because they are the work remaining:

| engine | why |
| --- | --- |
| `metal` | it registers `metal-ocean`, which is `style.DefaultThemeName`. The default theme cannot be optional |
| `aqua`, `motif` | `style/packs.go` — core, untagged — carries a **legacy era pack** of the same name, which an engine overrides when present ([style.RegisterPack]). Drop the engine and the name still resolves, to the legacy pack painted by the base engine, and `TestDecorationPaintsInsideItsBoxes` catches it painting outside its box. The legacy pack has to move behind the same tag as the engine that supersedes it |

A further 29 engine groups are entangled with each other: `win95` alone is
reached into by 38 others (and by core, for `snap` and `w95`), `aero` by
10 (and core, for `winSnap`), `material_tone` by 13, `web` by 9. Helpers
live in whichever engine file first needed them, so one cannot leave
without the others. Extracting those into untagged files is the bulk of
the remaining 6 MB.

The same rule applies to the tests, and the same fix: a helper more than
one engine's tests need belongs to all of them
(`style/enginetest_shared_test.go`), and a test that walks a table of
packs skips the rows whose engine is not in this build (`packBuilt`,
`needEngine`).
