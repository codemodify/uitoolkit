# Theme engines are opt-in

131 theme packs are drawn by 33 engines, and an application that wants
three of them should not carry all thirty-three. Engines are selected at
**build time**, with build tags.

```sh
go build ./...                                    # every engine (the default)
go build -tags theme_engines_pick ./...           # only the always-built ones
go build -tags "theme_engines_pick,\
                theme_engine_beos,theme_engine_platinum" ./...
```

Measured on `cmd/uitoolkit-settings`, stripped (`-ldflags "-s -w"`):

| build | size |
| --- | --- |
| default, all engines | 21.06 MB |
| `theme_engines_pick` | **18.71 MB** |
| `theme_engines_pick` + beos + platinum | 19.15 MB |

The ceiling, measured by making every engine unreachable and letting the
linker drop it, is **12.5 MB** — so the 13 engines that are separable
today are worth 2.35 MB of an available 8.6 MB. The rest is blocked by
what is described under *What is not separable yet*.

## Why the default is everything

An opt-in scheme whose default is *nothing* fails the wrong way. uitoolkit
is a library: the application author runs the build, so nothing here can
make them pass a flag. Forget it under a nothing-by-default scheme and you
get a toolkit with no themes, at run time, with no compile error to say
so — the same silent failure the platform boundary spent a release
removing.

So every engine file carries:

```go
//go:build !theme_engines_pick || theme_engine_beos
```

Read it as: build this engine unless the application is picking its own,
and build it anyway if it picked this one. A forgotten flag gives you
**more** than you wanted, never less.

Build tags cannot contain hyphens — `//go:build theme-engine-beos` is a
syntax error, `parsing //go:build line: invalid syntax at -` — so the
names use underscores throughout.

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
