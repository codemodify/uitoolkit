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
cover scroll, splitter, table header, and menu geometry.

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

## Metrics we keep in code

| Token | Value | Peer analogue |
| --- | --- | --- |
| `style.ToolItemGap` | 8 | Qt toolbar spacing / Avalonia CommandBar gap |
| `style.ToolButtonChrome` | pad 10, icon→label 8 | Fluent / Adwaita toolbutton padding |
| Focus ring | `Palette.Focus` stroke | GTK focus ring / Avalonia focus adorner |
| Inactive selection | `Selection` @ 0.14 | Qt inactive highlight |
| Menu hot-track | `MenuHover` + `MenuHoverBorder` + `MenuGutter` | Office XP / Win32 menu highlight |

## Adding a row

1. Write a `Norm` in `internal/uitest/norms.go` (`Check` must fail on the old bug).
2. Mention it in the table above and in `widgets/contract_test.go`’s regression map.
3. Keep Mail as dogfood (`go test ./internal/mail`); chrome fixes stay in the toolkit.
