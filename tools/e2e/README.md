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
    python3 crop_test.py         # crop.py against PNGs of every filter, colour type and depth
    ./kwin.py N 'JS'             # run a KWin script: OUT(x) prints (window geometry / state oracle, maximize, tile)
    ./stop.sh N                  # stop app + compositor when done
    ./theme-tour.sh N [PACK...]  # gallery in every pack on the GPU + the same pack on the CPU, in pairs

- `crop.py` decodes any PNG (every filter type, palette, 16-bit, Adam7), so it
  reads the toolkit's own `WritePNG` stills as well as KWin's; python3 only.
- The first window is placed at the same spot every time if it is the only window.
- The injector releases every key it holds when it exits: to screenshot with a
  key held (Alt for mnemonic underlines), run `./in.sh N key+ 56 sleep 2500 key- 56 &`
  and take the shot during the sleep.
- `start.sh`'s SCALE only enlarges the virtual screen; its output stays at scale 1.
  For a real fractional scale, set it on the nested output:
  `WAYLAND_DISPLAY=uitk-e2e-N DBUS_SESSION_BUS_ADDRESS=$(cat N/bus.addr) kscreen-doctor output.Virtual-0.scale.1.75`
  (apps then get `preferred_scale` 210 and draw 1.75x buffers).
- Coordinates are logical px of the 1280x860 virtual screen (x2 if SCALE=2 in the png).
- X11 backend: `UITK_BACKEND=x11 ./run.sh N ./gallery` (runs on the nested Xwayland).
- Paint escape hatches: UITK_PAINT_FULLFRAME=1 (no partial redraw), UITK_PAINT_MSAA=0.
- App log: N/app.log; KWin's (and its scripts' print()) N/kwin.log. Key codes: keys.txt.
- Apps run on the instance's own D-Bus session (APP_BUS=... overrides): on the
  user's bus a tray icon or notification from the app would show on the real desktop.
- Drags: one in.sh invocation holds the button (`move X Y down move X2 Y2 up`); a
  window move starts only past the drag threshold (8 px; KDE 10), so move a few
  px first, then on: `move 700 50 down move 714 52 sleep 150 move 820 160 up`.
- Client-side frames: `UITK_DECORATIONS=client ./run.sh N BIN` (see docs/decorations.md).
- NEVER run apps or `go test ./platform` against the user's real session (wayland-0 / :0):
  always export WAYLAND_DISPLAY=uitk-e2e-N DISPLAY=$(cat N/display) for tests that open windows.
