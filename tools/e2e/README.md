# uitoolkit e2e rig (nested KWin on the real GPU)

Drives real uitoolkit windows on a KDE Plasma (KWin 6) Wayland desktop without
touching the user's session: every instance is a nested `kwin_wayland --virtual`
with its own D-Bus session, screenshots come from KWin's ScreenShot2 D-Bus API
(`KWIN_SCREENSHOT_NO_PERMISSION_CHECKS=1`), and input is injected through the
`org_kde_kwin_fake_input` protocol. Build the injector once with `./build.sh`.
Needs: kwin_wayland, dbus-run-session, python3-dbus, wayland-scanner, cc.

Each agent uses its OWN instance number N (never share). Everything is isolated:
own Wayland socket `uitk-e2e-N`, own Xwayland (display number in `N/display`), own D-Bus session, own XDG dirs.

    ./start.sh N [W H [SCALE]]   # start nested KWin (default 1280x860, scale 1)
    ./run.sh N /path/to/binary [args]   # launch app in instance N (kills the previous app of N)
    ./in.sh N move X Y sleep 150 click sleep 300     # inject input (compositor coords)
    ./shot.sh N name             # screenshot -> N/shots/name.png  (then Read the png)
    python3 crop.py out.png x0 y0 x1 y1 a.png b.png ...   # stack same crop from several shots
    ./stop.sh N                  # stop app + compositor when done
    ./theme-tour.sh N [PACK...]  # gallery in every pack on the GPU + the same pack on the CPU, in pairs

- The first window is placed at the same spot every time if it is the only window.
- Coordinates are logical px of the 1280x860 virtual screen (x2 if SCALE=2 in the png).
- X11 backend: `UITK_BACKEND=x11 ./run.sh N ./gallery` (runs on the nested Xwayland).
- Paint escape hatches: UITK_PAINT_FULLFRAME=1 (no partial redraw), UITK_PAINT_MSAA=0.
- App log: N/app.log. Key codes: keys.txt.
- NEVER run apps or `go test ./platform` against the user's real session (wayland-0 / :0):
  always export WAYLAND_DISPLAY=uitk-e2e-N DISPLAY=$(cat N/display) for tests that open windows.
