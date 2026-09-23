# findings — keyboard-a11y-themes (instance 22)

## BUG 1: Keyboard focus rings are drawn outside the widget bounds and clipped away (Button/TextField/ComboBox/ScrollView/List/Tree/Table/TextArea/Card)
- app: all (gallery, mail) ; severity: major ; confidence: high ; partialOnly: false
- repro: launch gallery (default dark theme), pointer off-window; press Tab 4x (Primary action), 10x (Theme), 13x (ComboBox "Paint engine"), 3x (left ScrollView). Screenshot after each.
- actual: focus is practically invisible. Primary button: only pixel row y=313 changes (404059 -> 686881); the other 3 sides of the ring never appear. Tab 12->13 (Columns field -> ComboBox) produces a pixel-identical screenshot (diffbox same). ScrollView focus = a single 1px column at x=597. With UITK_PAINT_MSAA=0 even that sliver is gone: no pixel changes at all for ScrollView/Primary focus (shots m02..m04 identical).
- expected: a complete visible focus ring/border (Qt/GTK draw it inside the widget or the widget reserves margin).
- rootCause: widget/paint.go:27 (paintNode) and widget/scene.go:258 (recordNode) do ctx.ClipRect(local) before c.Paint; but DrawButton (style/look.go:258), DrawTextField (362), DrawComboBox (993), DrawTextArea (1249), DrawCheckbox (299, box at x=0), Switch (1372 track.Inset(-3)), ScrollView.Paint (widgets/scroll.go:169), ListView (widgets/list.go:144), TreeView (widgets/tree.go:239), TableView (widgets/table.go:497), CardList (widgets/card.go:185) all call DrawFocusRing(ctx, b.Inset(-2)) -> ring lies 1..3.75px OUTSIDE the clip. Classic3D ring = 1px stroke at b.Inset(-1) -> fully outside; default/SoftShadow ring strokes span Inset(-3.75..-0.2) -> fully outside. Only AA coverage of a fractional bound edge survives.
- proposedFix: draw focus indication inside the bounds: DrawFocusRing(ctx, b.Inset(1..2)) (Win95 dotted rect inside the face; field/combo: focus-colored border inside), or give widgets an overflow/"ink" rect and let paintNode/recordNode clip to bounds.Inset(-FocusWidth-2) for focusable widgets (and Invalidate the same inflated rect in Base.Invalidate/FocusGained/FocusLost so damage covers the ring).
- files: style/look.go, style/chrome.go, widget/paint.go, widget/scene.go, widgets/scroll.go, widgets/list.go, widgets/tree.go, widgets/table.go, widgets/card.go
- evidence: shots/s03.png s04.png s05.png s12.png s13.png m02-m04.png (MSAA=0), z_prim.png, z_tf.png

## BUG 2: Tab into MenuBar / ToolBar shows no focus; first arrow key skips the first title/tool
- app: gallery (+ any app with MenuBar/ToolBar, e.g. mail menubar) ; severity: major ; confidence: high ; partialOnly: false
- repro: gallery, pointer off-window. Tab (focus=MenuBar) -> screenshot identical to before. Right -> "Edit" ring (File skipped). Tab (focus=ToolBar) -> no ring. Right -> ring on 2nd tool "Open" (New skipped). Enter right after Tab activates an invisible item.
- rootCause: MenuBar (widgets/menu.go:86-93) and ToolBar (widgets/toolbar.go:42-49) declare their own `keyNav bool` field that shadows widget.Base.keyNav. app/window.go:816 tab() calls widget.MarkKeyboardFocus(list[idx]) which resolves to the promoted Base.MarkKeyboardFocus -> sets Base.keyNav only. MenuBar.Paint (menu.go:180) / ToolBar.Paint (toolbar.go:237) test their own m.keyNav/t.keyNav (false until KeyPress) so no title/tool gets StateFocused. KeyPress then sets keyNav=true AND moves focus in the same press (menu.go:262-275, toolbar.go:300-312), so the first visible ring is on item 1. FocusLost also resets focus=-1, so re-entry starts from nothing.
- proposedFix: implement MarkKeyboardFocus() on MenuBar/ToolBar (set own keyNav=true, focus=first enabled if <0, Invalidate) or drop the shadow field and use Base.KeyNav(); in KeyPress, when keyNav was false, first reveal the current item instead of moving.
- files: widgets/menu.go, widgets/toolbar.go
- evidence: shots/mb0..mb4.png, c_mb.png

## BUG 3: Menu accelerators shown in menus (Ctrl+N, Ctrl+O, F1, Ctrl+Q) do nothing
- app: gallery (toolkit-wide: no accelerator dispatch exists) ; severity: major ; confidence: high ; partialOnly: false
- repro: gallery; press Ctrl+Q (no focus, and again with focus on a button): app keeps running. F1: no About dialog. Ctrl+N: no second window.
- rootCause: MenuItem.Shortcut (widgets/menu.go:14) is display-only (used only at menu.go:563/681/788 for layout/paint). app/window.go dispatch (KeyDown, ~530-560) only handles Tab/Escape/Alt-mnemonics/popup/bubbleKey; bubbleKey returns when no focus (window.go:648) and no widget matches Shortcut strings. Only mail has its own handler (examples/mail/mailapp/keys.go wrapShortcuts).
- proposedFix: parse Shortcut into (mods,key) (e.g. widgets.ParseAccel) and have MenuBar implement a window-level accelerator hook (like HandleAlt): in Window.dispatch KeyDown, after popup handling and before/after bubbleKey (when unhandled), walk MenuBars and fire the first enabled item whose accel matches; also fire with no focus.
- files: widgets/menu.go, app/window.go, widgets/mnemonic.go
- evidence: shots/q0..q4.png (identical), app alive after Ctrl+Q

## BUG 4: Tab moves focus to controls hidden below the fold of a ScrollView without scrolling them into view
- app: gallery (left pane), any ScrollView/Form (settings, mail dialogs) ; severity: major ; confidence: high ; partialOnly: false
- repro: gallery, pointer off-window, press Tab 17x -> focus = Slider (content y=875, viewport ends at 732). Screenshot identical to the 14-Tab shot (nothing focused is visible). Press Right 3x: slider changes 60%->75% invisibly. Wheel-scroll the left pane: slider shows focus ring and 75%. Tabs 14..24 (radios, slider, checkboxes, switches, TextArea, "Open file…") are all invisible.
- rootCause: app/window.go:782-817 tab() only RequestFocus + MarkKeyboardFocus; nothing asks enclosing scrollers to reveal the target. widgets/scroll.go has no ensure-visible logic (grep: no EnsureVisible/ScrollIntoView in repo; ScrollTo only called by examples/tests).
- proposedFix: add ScrollView.EnsureVisible(devRect) (adjust OffsetY so rect+focus margin is inside LocalBounds, ScrollTo) and a widget-level helper widget.RevealFocus(c) that walks c.Parent() chain calling any `interface{ EnsureVisible(Component) }`; call it from Window.tab(), mnemonic focus, FocusFirstIn, and when keyboard moves focus inside composite widgets.
- files: app/window.go, widgets/scroll.go, widget/focus.go
- evidence: shots/sv17.png (focus on slider, nothing visible), sv17r.png (after Right x3, unchanged), sv17s.png (after manual scroll: slider focused at 75%)
