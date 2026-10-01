# When the toolkit does not do what you asked

Several layers of this toolkit are entitled to override what an
application states, and each of them is right to:

- the **look's era** decides how your title bar meets the window frame —
  a Windows 95 pack puts a tool bar *under* its title strip, because that
  is where Windows 95 put one;
- **build tags** decide which theme engines exist, so a pack you pinned
  may not be in the binary at all ([engines.md](engines.md));
- the **person's icon directory** decides which artwork is found, and it
  is theirs to manage ([../icons/README.md](../icons/README.md));
- the **compositor** decides whether the toolkit may draw a frame.

Each rule is sensible. Together they used to mean that an application
could state something clearly, be overruled by a layer it had never heard
of, and get no signal at all — a wrong pixel and nothing to pull on. The
sample that looks like Chromium asked for a browser's shape correctly, in
four separate calls, and still came out with a caption row above its tabs.

So every override says so.

```
uitk theme: asked theme pack "adwaita", got the default look; no engine in
  this build draws that pack — build with the engine that draws it;
  docs/engines.md lists the tags

uitk caption: asked a title bar of the application's own (SetTitleBar), got
  it in a row under the look's title strip; pack "win95" draws a stacked
  frame — win.SetCaptionStyle(style.CaptionMerged) makes your bar the
  caption under every pack, the way Chromium and VS Code do
```

Every finding has the same three parts, and the third is the point:
**what you asked**, **what happened instead**, and **the call or build
flag that gets what you asked**. Never "see the docs".

## Reading them

They are logged as they happen, which is what a developer wants while
building. They are also *collected*, which is what a test wants:

```go
a := uitoolkit.New(uitoolkit.Options{Theme: uitoolkit.ThemeOverride{Pack: "adwaita"}})
win, _ := a.NewWindow(...)
// ... build the window ...

for _, f := range a.Diagnostics() {
    log.Print(f)          // f.Level, f.Area, f.Asked, f.Got, f.Fix
}
```

In a test this is how an application stops drifting back into the
confusion without noticing:

```go
func TestTheToolkitIsDoingWhatWeTold(t *testing.T) {
    a, win := openMainWindow(t)
    layOut(t, a, win)
    for _, f := range a.Diagnostics() {
        if f.Level == uitoolkit.Warn {
            t.Errorf("%v", f)
        }
    }
}
```

`uitoolkit.SetDiagnosticLogging(false)` silences the log line without
losing the findings, for a program that shows them its own way.

## When they are reported

Most arrive while a window is being built, but the caption note waits for
the **first layout**, on purpose: an application configures a window over
several calls, and one that sets a title bar and settles its caption
shape on the next line has not been overruled at all. Findings that
depend on painting — a missing icon — arrive when that paint happens. So
read `Diagnostics()` after the window has laid out, not straight after
`New`, and leave the log on to catch the later ones.

## Levels

- **warn** — the application asked for something and did not get it: a
  pack whose engine is not in the build, a client frame the compositor
  refused, a tray menu the host would not let the toolkit draw.
- **note** — the toolkit did something reasonable that you may be
  surprised by, where nothing was asked and denied: a classic look
  putting your title bar in a row under its own strip is the era being
  honoured. The note exists for the developer wondering why.

A finding is never a reason to stop. Anything that must fail is an error.

## It is not tracing

Debug tracing — `UITK_TRAY_DEBUG`, the X11 XInput logs — is a different
thing and stays where it is. Diagnostics are about *intent*: something an
application stated, and a layer that did otherwise. A number of events
per second is not that.

## Adding one

Any layer that may override an application's intent owes it a finding:

```go
diag.Report(diag.Finding{
    Level: diag.Warn,
    Area:  "theme",
    Asked: fmt.Sprintf("theme pack %q", want),
    Got:   "the default look; no engine in this build draws that pack",
    Fix:   "build with the engine that draws it; docs/engines.md lists the tags",
})
```

Identical findings are reported once, so this is safe on a paint path.
`diag` is a leaf package — it imports nothing from the toolkit — so every
layer including `style` can reach it.
