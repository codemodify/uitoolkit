# Desktop chrome comparison

Name maps live in [widgets.md](widgets.md). This page is the **behavior**
map: what Avalonia, Qt, and GTK do for everyday chrome, what uitoolkit
does, and which automated check fails if we regress.

SourceGit is the Avalonia reference app (Fluent theme, `ToggleButton` for
History / Changes, `Menu` + overlay dismiss, command-bar spacing). We
match those norms, not pixel-identical Fluent.

## How to run

```bash
CGO_ENABLED=0 go test ./internal/uitest -run TestDesktopChromeNorms
go run ./cmd/uitest-driver -compare
CGO_ENABLED=0 go test ./...
```

`internal/uitest.ChromeNorms` is the checklist. The driver prints one
`ok` / `FAIL` line per row. Widget contracts in `widgets/contract_test.go`
cover scroll, splitter, table header/resize, and menu geometry. A 1×/2×
chrome-strip golden (`internal/uitest/testdata/chrome-strip-*x.png`)
hashes button + checkbox + menu row paint so dirty-rect hacks fail CI.

## Norms

| ID | Control | Avalonia / SourceGit | Qt | GTK 4 | uitoolkit |
| --- | --- | --- | --- | --- | --- |
| `button-focus-visible` | Button | `:focus-visible` after Tab | `WA_KeyboardFocusChange` | `:focus-visible` | `SetFocusVisibleOnly` + Tab marks keyboard |
| `checkbox-click-release` | CheckBox | click completes on release | `clicked()` on release | activate on click | press arms, release inside toggles; drag-off cancels |
| `radio-click-release` | Radio | same click model | `QRadioButton` click | `GtkCheckButton` | same as checkbox |
| `toggle-vs-action` | ToolToggle | `ToggleButton` checked chrome (SourceGit view modes) | checkable `QToolButton` | `GtkToggleButton` | outlined well off, accent fill on; actions stay flat |
| `toolbar-gaps` | ToolBar | CommandBar item spacing | `QToolBar` layout spacing | toolbar `GtkBox` gap | `ToolItemGap` + shared `ToolButtonChrome` |
| `toolbar-mouse-focus` | ToolBar | no lingering `:focus` after click | no keyboard frame after click | no `:focus-visible` after click | `keyNav` false after mouse |
| `menubar-dismiss-focus` | MenuBar | Menu overlay close | `QMenuBar` unhighlight | popover dismiss | `Close` / pick / dismiss clear `keyNav` |
| `popup-leave-highlight` | PopupMenu | hover ends on leave | active item vs hover | prelight clears on leave | highlight = hover or keyboard `focus` |
| `menu-hover-bordered` | PopupMenu / MenuBar | Office XP hot-track (bordered fill) | `QMenu` highlight rect | menu prelight | `MenuHover` + 1px `MenuHoverBorder` across gutter + label |
| `menubar-title-hover` | MenuBar | only the hot title highlights | `QMenuBar` item hover | menubar prelight | bar `StateHovered` is not inherited by sibling titles |
| `textfield-inactive-sel` | TextField | inactive selection brush | `QLineEdit` inactive | `GtkEntry` unfocused | selection alpha drops when `!Focused` |
| `label-clip` | Label | `TextBlock` clip / trim | `QLabel` elide | `GtkLabel` ellipsize | `ClipRect` + `Font.Fit` |
| `button-label-clip` | Button | content clip | clip to contents rect | clip | fitted + clipped label |
| `combo-popup-clear` | ComboBox | popup below, no field overlap | `QComboBox` list | `GtkDropDown` | existing popup geometry contract |
| `scroll-overflow-thumb` | ScrollView | thumb when content overflows | `QScrollBar` | `GtkScrolledWindow` | thumb in-track, hidden when content fits, wheel clamps |
| `list-scroll-clip` | ListView | viewport clip | `QListView` | `GtkListView` | rows clip; selected wash does not bleed after scroll |
| `table-header-flush` | TableView | sticky header, no gap/bleed | `QTableView` | `GtkColumnView` | first row flush under header; header pixels stable on scroll |
| `table-column-resize` | TableView | `DataGrid` column drag | `QHeaderView` | column resize | header divider drag; pointer restored |
| `splitter-clip-cursor` | Splitter | exclusive panes + clip | `QSplitter` | `GtkPaned` | children clipped to pane; sash cursor returns to pointer |
| `combo-field-height` | ComboBox | closed height + keyboard | `QComboBox` | `GtkDropDown` | `ComboH`; Down opens/navigates; Escape dismisses |
| `field-toolbar-height` | TextField / NumberField | toolbar-height field | `QLineEdit` / `QSpinBox` | `GtkEntry` / `GtkSpinButton` | `FieldHeight` == `ComboH`; spinner is one tab stop |
| `combo-focus-visible` | ComboBox | `:focus-visible` | `WA_KeyboardFocusChange` | `:focus-visible` | mouse click clears ring; Tab paints it |
| `switch-click-release` | Switch | click completes on release | Quick `Switch` | `GtkSwitch` | press arms, release inside toggles; disabled ignores keys |
| `slider-click-disabled` | Slider | click + disabled chrome | `QSlider` | `GtkScale` | click sets value; clamp; muted disabled fill; focus-visible |
| `progress-metrics` | ProgressBar | determinate clamp | `QProgressBar` | `GtkProgressBar` | value 0..1; `ProgressH`; disabled fill muted |
| `tabs-click-focus` | TabBar | select on release | `QTabBar` | `GtkNotebook` | release selects; disabled ignores keys; focus-visible |
| `dialog-buttons` | MessageBox | default + Escape | `QMessageBox` | `GtkAlertDialog` | primary Yes/OK; Escape cancel; click-release |
| `menu-gutter-icons` | PopupMenu | icons + checks in gutter | `QMenu` | menu | widest label wins width; check/icon gutter ink |
| `menu-onscreen-clamp` | PopupMenu | stay on-screen | `QMenu` | `GtkPopover` | right-edge placement keeps intrinsic width |

## Metrics we keep in code

| Token | Value | Peer analogue |
| --- | --- | --- |
| `style.ToolItemGap` | 8 | Qt toolbar spacing / Avalonia CommandBar gap |
| `style.ToolButtonChrome` | pad 10, icon→label 8 | Fluent / Adwaita toolbutton padding |
| Focus ring | `Palette.Focus` stroke | GTK focus ring / Avalonia focus adorner |
| Inactive selection | `Selection` @ 0.14 | Qt inactive highlight |
| Menu hot-track | `MenuHover` + `MenuHoverBorder` + `MenuGutter` | Office XP / Win32 menu highlight |
| `style.ComboH` / `FieldHeight` | 30 (compact 26 / relaxed 36) | Qt toolbar-height combo / spin |
| `style.Metrics.ProgressH` | look metric | Qt / GTK progress thickness |
| Column resize hit | 5 device px | Qt `QHeaderView` handle |

## Adding a row

1. Write a `Norm` in `internal/uitest/norms.go` (`Check` must fail on the old bug).
2. Mention it in the table above and in `widgets/contract_test.go`’s regression map.
3. Chrome fixes stay in the toolkit, never in an application. The mail client
   ([comms-mail](https://github.com/codemodify/comms-mail)) is the dogfood that
   found most of these rows; it builds on the published module, so a row it
   needs has to be a toolkit row.
