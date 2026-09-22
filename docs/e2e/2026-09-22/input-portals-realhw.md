# Gestures, X11 smooth scrolling and the portals, on real hardware (instance 30)

Nested KWin 6 on the real GPU, `tools/e2e` instance 30, KWin ScreenShot2
captures, Wayland and X11 (the instance's own Xwayland, XInput 2.4). Nothing
ran against the user's session or the user's D-Bus: every app ran on the
instance's own bus through `run.sh`, with the instance's own XDG dirs, and
the instance bus's activation environment pointed at the nested session
(`start.sh`). The instance bus had no portal and no notification server
(`ListNames`), and none was started: `tools/e2e/fakeportal` owned
`org.freedesktop.portal.Desktop` and `org.freedesktop.Notifications` there
and logged every call (`30/fakeportal.log`), and its `importer` helper made
each file dialog a real window, the child of the window the call named.
Afterwards the fake portal (by its PID) and the instance were stopped and
`/run/user/1000/uitk-e2e-30*` removed.

Build: `examples/tour`, `examples/files`, `examples/mail` from
`feat/input-portals`. Shots are in `tools/e2e/30/shots` (gitignored).

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | A notification with buttons, Wayland | tour `-page desktop`, "Send a notification"; the fake clicks `button-0` after 3 s | ok — `Notifications.Notify app="uitoolkit tour" icon="dialog-information" actions=["default" "Open" "button-0" "Raise the tour" "button-1" "Thanks"] urgency=1`; the page said "Sent through fdo", then "Clicked: raise" (`w-notify-crop.png`) |
| 2 | A link and a folder, Wayland | click "The project's page", "Your home folder" | ok — `OpenURI.OpenURI parent="wayland:7c2f6435-…" uri="https://github.com/codemodify/uitoolkit"`, `OpenURI.OpenDirectory parent="wayland:7c2f6435-…" dir="/home/user"` (the folder went by descriptor); visited links turn muted |
| 3 | The desktop's file dialog as the window's child, Wayland | "The desktop's dialog" | ok — `FileChooser.OpenFile parent="wayland:7c2f6435-a89f-446f-9f3c-1ae24f559e89" modal=true`: the handle KWin's `zxdg_exporter_v2` gave the tour; the importer imported it and KWin reports the dialog `transientFor` the tour window, centred on it (tour 110,60 1060×740; dialog 430,300 420×260 — the same centre) (`w-dialog.png`) |
| 4 | The same on X11 | `UITK_BACKEND=x11` | ok — `parent="x11:400002"`; the importer's dialog is `transientFor` the tour, `modal`, centred (430,286 420×288 over 110,60 1060×740) (`x-dialog.png`); the link went with the same parent |
| 5 | Mail's new-mail notification, through the server | Mail, Fetch; minimise Mail with `kwin.py`; the fake clicks `button-0` | ok — `Notify app="Mail" icon="mail-unread" summary="Fetch Robot <fetch@demo.invalid>" actions=[… "button-0" "Open Mail"] desktop-entry="mailclientui"`; the second fetch replaced it (`replaces=2`); the click un-minimised and activated Mail |
| 6 | The same through the portal | `UITK_NOTIFY=portal` | ok — `Notification.AddNotification id="new-mail" buttons=[{action: "button-0", label: "Open Mail"}] default-action="default" icon=["themed", <["mail-unread"]>] priority="normal"`; `ActionInvoked` un-minimised and activated Mail |
| 7 | One wheel notch = three lines, X11 and Wayland | tour, `wheel 0.6666667` (exactly one notch on both) | ok — both scrolled 66 px, three 22-px lines (`x-scroll.png`, `w-scroll.png`); on X11 through XInput 2 (`UITK_X11_DEBUG=1`: `scroll from device 8: 0, 1 precise false`), the server's emulated button 5 dropped by its flag |
| 8 | No glide on a wheel, X11 | a shot a second after the notch | ok — unchanged (`x-scroll.png`, third row) |
| 9 | Files: a folder opened, back and forward, Wayland | Projects tab, double-click uitoolkit, `click back`, `click forward`, Alt+Left, Alt+Right | ok — titles Projects → uitoolkit → Projects → uitoolkit → Projects → uitoolkit; Back live and Forward greyed after opening (`f-opened.png`) |
| 10 | The same on X11 | `UITK_BACKEND=x11` | ok — the same titles; the thumb buttons are X11 buttons 8 and 9 through XInput 2 |
| 11 | XInput 2 did not break the rest of X11 input | Files on X11: clicks, a double-click, a menu, hover, a system move by the caption | ok — the File menu opened, hovered and opened About (`fx-menu-crop2.png`); dragging the caption moved the window by (−100, +100) through `_NET_WM_MOVERESIZE` after the XInput 2 ungrab; no X protocol errors |

## Found and fixed here

- **Xwayland's pointer is a floating device** until the pointer first enters
  one of its windows; the scan took only attached slaves and found no scroll
  valuators. Every pointer device with scroll valuators is taken now.
- **Xwayland floats and re-attaches its pointer on every crossing**
  (`XI_HierarchyChanged` each time), and every one of them rescanned the
  devices with several round trips. Only a device added or removed rescans
  now; a device switch seeds the valuators from the event itself.
- **Files' Back and Forward looked live in a fresh window**: the first tab
  was selected before anything listened.

## Not provable here

- **Gestures and finger scrolls.** KWin's fake input has no gesture requests
  and no axis source, so pinch, swipe, hold and the X11 glide are proven
  headlessly: the Wayland objects and events on the wire against the fake
  compositor, the X11 curve and classification in unit tests, delivery in the
  window with offscreen windows. The rig's Xwayland reported XInput 2.4 with
  gestures selected without a protocol error.
- **The real portal.** Not started on a nested session (its backends are Qt
  and GTK apps, and it starts the document portal, which mounts itself under
  `/run/user/1000/doc`);
  the D-Bus shapes are the fake's, the xdg-foreign handle and the parenting
  are KWin's own.

## Rig findings

- On Xwayland a fake-input `wheel` arrives only after a `move` in the same
  `in.sh` call; `wheel 1` (15 units) is a notch and a half there. The README
  says how to send exactly one notch.
