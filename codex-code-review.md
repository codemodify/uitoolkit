**Repository code review: uitoolkit**

Reviewed on September 27, 2026. **29 findings: 4 high-priority (P1), 25 medium-priority (P2).** P1 covers potential data loss or application hangs; P2 covers reproducible functional defects and integration failures.

The review covered the application loop and window lifecycle, Linux/native platform integration, widget rendering/input, layout, rich text, accessibility paths, docking/rack behavior, themes/skins/preferences, sample applications, commands, and shell/Python tools. Source inspection was combined with the existing suites and focused temporary regression probes. Detailed visual fidelity of every theme and binary asset was not exhaustively inspected.

The review started at commit `05aeee5c53deedeff08879b7f930e9242528781e` and finished against `92dd7e173d5951d49314b074ef3a31c594f50982`. The intervening repository changes were limited to README.md and version.go. References below identify the reviewed implementation. This review changed no production code; all temporary probe files were removed.

Validation completed:

- `CGO_ENABLED=0 tools/test.sh -count=1 ./...` — passed with all theme engines.
- `CGO_ENABLED=1 tools/testenv.sh go vet -tags theme_engine_all ./...` — passed.
- `CGO_ENABLED=1 tools/testenv.sh go test -race -tags theme_engine_all ./app ./platform ./widget ./dock ./rack -count=1` — passed after temporary probes were removed.
- Platform package cross-compilation for Windows amd64 and Darwin arm64, CGO disabled — passed.
- Shell syntax checks on 21 scripts; Python AST parsing on eight files; all four crop tests — passed.
- Focused probes reproduced the findings below except the two explicitly marked static native analyses. Wayland and portal probes used fakes/private buses. No live-desktop end-to-end, native Windows/macOS runtime, or real GPU visual verification was performed.

Passing baseline tests do not cover the failing scenarios below. Each finding includes its trigger/evidence and a concrete correction direction.

**1. [P1] The Linux wake pipe can block Post and Quit**

Location: [platform/wake_linux.go:49](/home/user/go/src/github.com/codemodify/uitoolkit/platform/wake_linux.go:49).

`fdWaker.Signal` uses `os.File.Write`, which waits when the pipe fills despite its nonblocking descriptor. Every `WakeSurface` signals the global pipe before waking the surface. X11 and offscreen waits do not drain that global pipe, so even cumulative, promptly handled posts eventually fill it. A later post blocks before sending the surface wake; posting from the UI thread can deadlock the loop.

Evidence: A temporary test filled a fresh waker through its raw descriptor. `Signal` stayed blocked until the test explicitly drained the pipe. The undrained X11 path is [platform/x11_linux.go:1833](/home/user/go/src/github.com/codemodify/uitoolkit/platform/x11_linux.go:1833); unconditional signaling is [platform/wait.go:37](/home/user/go/src/github.com/codemodify/uitoolkit/platform/wait.go:37).

Fix: Use a raw nonblocking write and treat EAGAIN as an already-pending wake, or implement a coalescing wake mechanism.

**2. [P1] Wayland reports incomplete clipboard and drop transfers as successful**

Location: [platform/wayland_linux.go:4485](/home/user/go/src/github.com/codemodify/uitoolkit/platform/wayland_linux.go:4485).

Timeout, read error, exceeding the 64 MiB limit, and successful EOF all reach `return string(out), true`. A delayed sender can therefore produce a truncated paste or accepted drop. For a negotiated Move, the application can acknowledge the partial payload and permit the source to discard the original.

Evidence: A fake-compositor test wrote `partial` and kept the pipe open. `wlReadFD` returned `("partial", true)` after the 500 ms timeout. [app/drop.go:168](/home/user/go/src/github.com/codemodify/uitoolkit/app/drop.go:168) passes that success into drop completion.

Fix: Distinguish EOF from timeout/error/size-limit termination and return failure unless the complete transfer was received.

**3. [P1] Save dialogs return the old selection after the user edits the filename**

Location: [widgets/filedialog.go:273](/home/user/go/src/github.com/codemodify/uitoolkit/widgets/filedialog.go:273).

`chosen()` always prefers `table.Selected` over the editable path field. Editing the path does not clear the selected row. A user can select an existing file, type a new filename, and have Save return the existing file instead, exposing callers to unintended overwrite.

Evidence: A temporary FileSave test selected `old.txt`, changed the path to `new.txt`, and accepted the dialog. The callback received `old.txt`.

Fix: Make the entered path authoritative, or synchronize/clear the row selection when the path is edited.

**4. [P1] The Files sample acknowledges external Move drops without preserving the files**

Location: [examples/uitoolkit-sample-files/filesapp/drag.go:146](/home/user/go/src/github.com/codemodify/uitoolkit/examples/uitoolkit-sample-files/filesapp/drag.go:146).

The sample's targets advertise Move, but external drops only add in-memory rows. They neither move nor durably copy files. Directory contents are not imported, and files at least 64 KiB do not even retain a preview body. An external source that deletes originals after a successful Move can consequently lose the files.

Evidence: Dropping a 65,536-byte file with `DragMove` returned true, added one row, and left its Body empty. Targets advertise Move at [examples/uitoolkit-sample-files/filesapp/files.go:338](/home/user/go/src/github.com/codemodify/uitoolkit/examples/uitoolkit-sample-files/filesapp/files.go:338) and [examples/uitoolkit-sample-files/filesapp/files.go:529](/home/user/go/src/github.com/codemodify/uitoolkit/examples/uitoolkit-sample-files/filesapp/files.go:529). The source-removal contract is explicit in [widget/drag.go:48](/home/user/go/src/github.com/codemodify/uitoolkit/widget/drag.go:48).

Fix: Negotiate Copy for external file imports and refuse external Move unless durable filesystem transfer is implemented. Preserve internal row-moving behavior separately.

**5. [P2] Hiding a widget leaves its old pixels in retained scenes**

Location: [widget/scene.go:174](/home/user/go/src/github.com/codemodify/uitoolkit/widget/scene.go:174).

`subtreeDirty` ignores invisible nodes before checking their dirty flags. `SetVisible(false)` invalidates the child but does not request layout. Its parent can therefore reuse a cached group that still contains the previously visible child.

Evidence: An app-level test painted a button in a column, hid it, and called `PumpOnce`. The result differed by 6,800 pixels from the correct frame obtained after explicitly resetting the scene cache.

Fix: Invalidate the containing cached group on visibility changes, or account for the transition before deciding an ancestor is reusable.

**6. [P2] Closing a drag source leaves the application stuck in an active drag**

Location: [app/window.go:2153](/home/user/go/src/github.com/codemodify/uitoolkit/app/window.go:2153).

`Window.Close` destroys and removes the source window without finishing its application-level `dragRun`. A backend completion event queued for that removed window will no longer be pumped. The stale `app.drag` prevents subsequent `StartDrag` calls and retains the drag payload and callbacks.

Evidence: A temporary test created two windows, started a drag from the first, closed it, and pumped the application. The second window still reported Dragging, the Done callback had not run, and starting another drag returned false.

Fix: Finalize or cancel a departing source's drag exactly once, while preserving the existing successful tear-off merge completion path.

**7. [P2] A Wayland transition back to 1× never resets the buffer scale**

Location: [platform/wayland_linux.go:2465](/home/user/go/src/github.com/codemodify/uitoolkit/platform/wayland_linux.go:2465).

The integer-scale presentation branch runs only when the scale exceeds one. After a 2× surface changes to 1×, smaller buffers are attached while the compositor still has buffer scale two. The visible size is wrong; incompatible buffer dimensions can also violate scale constraints.

Evidence: A fake-compositor trace showed `set_buffer_scale 2`, a 200×160 buffer, then a 100×80 buffer with no `set_buffer_scale 1` request.

Fix: Apply every positive resolved integer scale, including one, and update/reset viewport state when switching presentation modes.

**8. [P2] An explicit 1× Wayland preference is discarded**

Location: [platform/wayland_linux.go:1825](/home/user/go/src/github.com/codemodify/uitoolkit/platform/wayland_linux.go:1825).

`deviceScale` accepts fractional and surface scales only when they exceed one. A compositor's explicit fractional scale of one can therefore fall through to a stale/global scale of two. The same fallback can defeat a surface's resolved 1× monitor scale.

Evidence: A state test with `frac=1`, `bufScale=2`, and `conn.outScale=2` returned two instead of the explicit preference of one.

Fix: Represent unknown scale separately and honor any positive compositor preference before falling back to connection defaults.

**9. [P2] Wayland silently ignores valid logical resize requests at HiDPI**

Location: [platform/wayland_linux.go:2258](/home/user/go/src/github.com/codemodify/uitoolkit/platform/wayland_linux.go:2258).

`Resize` takes logical pixels, but `fitLogicalSize` guesses that an input matching the current buffer dimensions is a device-pixel request. A legitimate size change can consequently be converted into a no-op.

Evidence: At 2×, a 100×80 logical surface receiving `Resize(200,160)` remained 100×80. The heuristic responsible is [platform/scale.go:149](/home/user/go/src/github.com/codemodify/uitoolkit/platform/scale.go:149).

Fix: Honor the logical-pixel contract in Resize and convert device dimensions explicitly at callers that need that conversion.

**10. [P2] Portal file choosers can wait forever on a different returned handle**

Location: [platform/filechooser_linux.go:67](/home/user/go/src/github.com/codemodify/uitoolkit/platform/filechooser_linux.go:67).

The signal handler supports a portal returning a request handle different from the predicted path, but the D-Bus match subscribes only to the predicted path. Responses on the actual handle never reach the handler, leaving the callback pending.

Evidence: A private-bus fake returned a different handle and emitted a successful Response there. The callback never ran, despite the handler's alternate-path check at [platform/filechooser_linux.go:135](/home/user/go/src/github.com/codemodify/uitoolkit/platform/filechooser_linux.go:135).

Fix: Subscribe to the actual handle when it differs, with a strategy that also captures responses arriving during the method call.

**11. [P2] Closing an X11 surface twice can disconnect surviving windows**

Location: [platform/x11_linux.go:2068](/home/user/go/src/github.com/codemodify/uitoolkit/platform/x11_linux.go:2068).

The close guard returns only when the surface is closed and the shared display is also gone. With two surfaces, closing the first twice can decrement the connection's reference count twice and close the display while the other surface remains live.

Evidence: Static native call-path analysis: both closes reach `s.conn.release()` at [platform/x11_linux.go:2117](/home/user/go/src/github.com/codemodify/uitoolkit/platform/x11_linux.go:2117). No native X11 runtime reproduction was performed; the higher-level Window.Close guard protects calls made exclusively through that wrapper.

Fix: Track completed surface teardown separately from server-side closure and release each surface's shared-connection reference exactly once.

**12. [P2] Windows tray items can invoke the first item's callbacks**

Location: [platform/status_windows.go:108](/home/user/go/src/github.com/codemodify/uitoolkit/platform/status_windows.go:108).

Every tray item registers the same window class with a procedure capturing that individual item. Registration errors are ignored, so later windows use the first registered procedure. Clicks on another item, including one created after closing the first, can invoke the first item's handlers.

Evidence: Static Win32 path analysis of class registration and CreateWindow at lines 123–128. The class is not unregistered. Windows execution was unavailable; the platform package cross-compiled successfully.

Fix: Register a shared procedure once and dispatch by HWND, or give each item a distinct class with a defined unregister lifecycle.

**13. [P2] Direct rich-text document edits do not repaint attached editors**

Location: [widgets/richtext_layout.go:440](/home/user/go/src/github.com/codemodify/uitoolkit/widgets/richtext_layout.go:440).

The document watcher adjusts height/layout arrays but never invalidates the editor. This contradicts the Document API's promise that direct edits show in the view, and affects multiple views attached to one document.

Evidence: After arranging a hosted editor, calling `editor.Document().InsertText("hello")` generated zero invalidations. Changes remain stale until another event causes a repaint.

Fix: Invalidate each attached view when its document changes, with notifications observing the completed edit.

**14. [P2] Rejected text drops are incorrectly acknowledged as accepted**

Location: [widgets/dropzone.go:117](/home/user/go/src/github.com/codemodify/uitoolkit/widgets/dropzone.go:117).

TextField.Drop and TextArea.Drop return true even when `replaceSel` refuses the text through the Accept validator. The source is told the drop succeeded although the destination received nothing.

Evidence: A temporary test used an Accept function returning false and observed a true Drop result with an unchanged target. Stock text targets negotiate Copy; source deletion under Move was not established through the normal application path and is not claimed here.

Fix: Return the insertion outcome and report successful completion only when the target actually accepts the payload. Apply the same fix at [widgets/dropzone.go:134](/home/user/go/src/github.com/codemodify/uitoolkit/widgets/dropzone.go:134).

**15. [P2] Removing a Grid child changes the remaining children's cell assignments**

Location: [widgets/grid.go:82](/home/user/go/src/github.com/codemodify/uitoolkit/widgets/grid.go:82).

The cells slice is positional metadata parallel to Base.Children. Inherited Remove and ClearChildren update only the children, so metadata becomes misaligned; reparenting has the same risk.

Evidence: Placed A at (0,0) and B at (1,1), then removed A. `CellOf(B)` reported (0,0).

Fix: Maintain cell metadata through every child mutation, preferably keyed by component identity instead of slice position.

**16. [P2] Grid flexible tracks overflow even when their minimum widths fit**

Location: [widgets/grid.go:201](/home/user/go/src/github.com/codemodify/uitoolkit/widgets/grid.go:201).

The solver calculates weight shares without first reserving the space required by flex tracks whose content is larger than their share. Taking max(content, share) independently can make the total exceed the available width.

Evidence: Two equal Flex columns with zero gap, natural widths 90 and 10, and 100 pixels available received widths 90 and 50. The second child ended at x=140.

Fix: Freeze tracks that need their minimum width, then redistribute the remaining space among the other flexible tracks.

**17. [P2] Disabled editable ComboBoxes leave their text field enabled**

Location: [widgets/combobox.go:132](/home/user/go/src/github.com/codemodify/uitoolkit/widgets/combobox.go:132).

SetEditable creates an enabled TextField. ComboBox inherits Base.SetEnabled and does not propagate state changes to that field, which remains independently focusable and editable.

Evidence: Created an editable combo containing One, disabled it, and entered X in its field. Its text became OneX. Focus enumeration and key dispatch do not enforce ancestor enabled state.

Fix: Propagate the enabled state when creating the field and whenever the combo's state changes, as other composite input controls do.

**18. [P2] Undoing removal of every rich-text block adds an extra paragraph**

Location: [richtext/doc.go:489](/home/user/go/src/github.com/codemodify/uitoolkit/richtext/doc.go:489).

`replace` inserts an implicit empty paragraph when the document becomes empty, but the undo step still records zero replacement blocks. Undo then inserts the original blocks before that unrecorded paragraph.

Evidence: `d := richtext.NewPlain("hello"); d.ReplaceBlocks(0, d.Len()); d.Undo()` yielded `"hello\n"` instead of `"hello"`.

Fix: Normalize empty-document replacements before recording history and notifying watchers so the recorded replacement matches actual state.

**19. [P2] Rich-text transactions absorb the preceding typing undo step**

Location: [richtext/doc.go:541](/home/user/go/src/github.com/codemodify/uitoolkit/richtext/doc.go:541).

Transact records the current history length without sealing the preceding typing step. Its first edits can merge into that older step, leaving no new transaction entry to seal and causing Undo to remove text entered before the transaction.

Evidence: Inserted a, transacted insertion of b and c, then undid once. The result was empty instead of a.

Fix: Seal the preceding edit group before entering a transaction and seal/group the transaction's completed edits.

**20. [P2] Appearance round trips silently disable combo-wheel selection**

Location: [app/desktopprefs.go:185](/home/user/go/src/github.com/codemodify/uitoolkit/app/desktopprefs.go:185).

Application.Appearance reconstructs application preferences but omits ComboWheel. The look itself does not retain this field, so the returned value is false even when the option is enabled. Callers that read, modify, and reapply the appearance reset the user's choice.

Evidence: Enabled ComboWheel with ApplyAppearance, then called `a.ApplyAppearance(a.Appearance())`. `style.ComboWheel()` changed from true to false. The tour uses this read-modify-apply pattern.

Fix: Populate ComboWheel from the active preference alongside ReduceMotion and NativeDialogs.

**21. [P2] Dragging a sash after ResetLayout mutates the saved defaults**

Location: [dock/layout.go:241](/home/user/go/src/github.com/codemodify/uitoolkit/dock/layout.go:241).

ApplyLayout assigns slices backed by the input LayoutFrame arrays directly to the live splits. ResetLayout applies the stored default Layout, so subsequent sash movement mutates that default itself. Later resets no longer restore the original proportions.

Evidence: Recorded defaults, reset, moved the middle sash, and reset again. Column weights were `[0.9109313 1.5890689 0.5]` instead of `[1 4 1]`.

Fix: Copy Rows and Cols into independently owned slices before assigning them to the live splits.

**22. [P2] Closing or collapsing dock panels does not trigger layout autosave**

Location: [dock/panel.go:185](/home/user/go/src/github.com/codemodify/uitoolkit/dock/panel.go:185).

Panel.setClosed and SetCollapsed call relayout without layoutChanged. Stack.Select similarly changes the saved current tab without notifying. Applications relying on the documented OnLayoutChanged hook miss these changes; Inspector uses that hook to save its layout.

Evidence: Installed an OnLayoutChanged counter and closed a panel. The counter stayed at zero. Related omissions are [dock/panel.go:206](/home/user/go/src/github.com/codemodify/uitoolkit/dock/panel.go:206) and [dock/stack.go:76](/home/user/go/src/github.com/codemodify/uitoolkit/dock/stack.go:76); Inspector's persistence hook is [examples/uitoolkit-sample-inspector/inspectorapp/inspector.go:379](/home/user/go/src/github.com/codemodify/uitoolkit/examples/uitoolkit-sample-inspector/inspectorapp/inspector.go:379).

Fix: Emit a layout-change notification after close/show, collapse/expand, and current-tab changes, batching notifications during layout restoration if necessary.

**23. [P2] Docking overrides an explicitly featureless panel**

Location: [dock/host.go:329](/home/user/go/src/github.com/codemodify/uitoolkit/dock/host.go:329).

Host.adopt interprets a zero feature mask as uninitialized and restores DefaultFeatures. NewPanel already establishes the defaults, so this overwrites a caller's deliberate SetFeatures(0), making a locked panel closable, movable, floatable, and collapsible.

Evidence: Created a panel, called SetFeatures(0), and docked it. Features changed from zero to 15.

Fix: Apply defaults only during panel construction and preserve an explicitly supplied zero mask.

**24. [P2] Concurrent preference saves share and corrupt one temporary file**

Location: [style/prefs.go:88](/home/user/go/src/github.com/codemodify/uitoolkit/style/prefs.go:88).

Every writer uses `<path>.tmp`. Concurrent saves can truncate the same inode, rename another writer's file, or remove it during cleanup. This breaks atomic replacement and can publish invalid JSON, affecting multiple Settings instances or concurrent users of the save API.

Evidence: Eight temporary writers performing 250 writes each with differing JSON lengths produced 1,401 write/rename errors and 807 invalid-JSON reads in one run.

Fix: Use a unique temporary file in the destination directory for each save, close it, and rename only that writer's file.

**25. [P2] Cached skin sprites never check whether their PNG changed**

Location: [style/skin_assets.go:263](/home/user/go/src/github.com/codemodify/uitoolkit/style/skin_assets.go:263).

variant returns a cached cut before reaching sheetImage, which owns TTL/stat-based freshness checks. Once rendered, a sprite can keep stale art indefinitely despite repainting after the documented reload interval. Cached failed cuts have the same issue.

Evidence: Replaced a loaded sprite's PNG, waited past skinAssetTTL, and requested the variant again. The same stale pointer was returned. Calling sheetImage directly detected the edit and invalidated the cut.

Fix: Check source freshness before returning dependent cached sprite cuts, masks, and grids.

**26. [P2] Skin strip references succeed or fail according to map iteration order**

Location: [style/skin.go:963](/home/user/go/src/github.com/codemodify/uitoolkit/style/skin.go:963).

Parts are parsed in Go map order, while generated strip sprite names are registered only during their owner's parsing. Another part referring to a generated name such as button.normal therefore fails whenever it is parsed first.

Evidence: Loaded one identical manifest 1,000 times: 865 successes and 135 errors stating `parts.tool.states.normal: no sprite "button.normal"`. The button part defined a normal strip cell and the tool part referenced it.

Fix: Register all generated strip sprites in a first pass, then resolve every part's state references.

**27. [P2] A skin with no family incorrectly overrides a light base with dark**

Location: [style/skin.go:732](/home/user/go/src/github.com/codemodify/uitoolkit/style/skin.go:732).

ParseTheme immediately maps an omitted family to dark. The later fallback that should inherit the base pack's family consequently never runs, leaving light-base skins with inconsistent dark-family metadata and behavior.

Evidence: With all engines enabled, loaded `{"skin":1,"base":"light"}`. The base palette was light but the resulting skin pack palette was dark. Intended inheritance is at [style/skin_pack.go:319](/home/user/go/src/github.com/codemodify/uitoolkit/style/skin_pack.go:319).

Fix: Keep an omitted family unset until the base pack has been resolved.

**28. [P2] The full-atlas script silently omits most theme packs**

Location: [tools/atlas/render.sh:12](/home/user/go/src/github.com/codemodify/uitoolkit/tools/atlas/render.sh:12).

The script builds its Settings, sheet, and shots binaries without theme_engine_all. With opt-in engines, its supposedly complete atlas contains only the default engine's available packs.

Evidence: The default `uitk-themesheet -list` returned 12 packs; the all-engine build returned 131. The script therefore omits 119 packs under its normal build configuration.

Fix: Build all three atlas binaries with `-tags theme_engine_all`.

**29. [P2] Screenshot errors leave a reader thread hanging indefinitely**

Location: [tools/e2e/shot.py:25](/home/user/go/src/github.com/codemodify/uitoolkit/tools/e2e/shot.py:25).

The script starts a non-daemon pipe reader and calls CaptureWorkspace before closing the writer. If D-Bus raises an exception, the writer remains open and the reader never reaches EOF. Python waits for that thread even after printing the traceback.

Evidence: A subprocess using a fake D-Bus CaptureWorkspace that raised RuntimeError printed its traceback but remained alive until terminated by a two-second timeout.

Fix: Close the writer in a finally block and join/clean up the reader on both successful and failed captures.

An additional test-quality gap helps explain missed rendering regressions: [widget/scene_test.go:54](/home/user/go/src/github.com/codemodify/uitoolkit/widget/scene_test.go:54) uses colors such as `RGB(10,10,10)` and `RGB(200,60,60)` although channels are normalized floats. These saturate to the same white pixels, weakening comparisons that should distinguish foreground from background. Use normalized colors and assert a meaningful pixel difference before checking cache behavior.
