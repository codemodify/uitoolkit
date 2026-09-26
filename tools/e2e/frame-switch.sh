#!/bin/bash
# frame-switch.sh N [BACKEND [PACK]] — Settings' "OS window borders" box
# hands the frame between the desktop and the theme while the window is up,
# which is the one thing headless tests cannot see: a window with no frame
# has no frame to look at.
#
#   ./start.sh 7 2100 1620 && ./frame-switch.sh 7            # wayland, luna
#   ./frame-switch.sh 7 x11 win95                            # the X11 path
#
# It runs Settings in instance N with PACK staged and the desktop's frame,
# unticks OS window borders, presses Apply, and asks KWin whether the frame
# went.
# KWin's own answer is the oracle: while the desktop draws the frame the
# client sits inside it (clientGeometry is smaller than frameGeometry, by
# the title bar); once the toolkit draws it the two are the same rectangle.
# Then the same the other way round, from a window that started with the
# theme's frame. Shots land in $RIG/N/frames/ — look at them: only a
# picture says the caption is the theme's rather than merely gone.
#
# Both directions must pass on both backends. The Wayland half is an
# xdg-decoration mode change on a mapped surface; the X11 half is
# _MOTIF_WM_HINTS written on a mapped window, and a theme whose frame has a
# shadow re-creates the X window on a 32-bit visual on the way (so run it
# once with a flat pack like win95 and once with a soft one like luna).
set -u
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; BACKEND="${2:-wayland}"; PACK="${3:-luna}"
ROOT="$(cd "$RIG/../.." && pwd)"; D="$RIG/$N"; OUT="$D/frames"; BIN="$D/bin"
[ -f "$D/bus.addr" ] || { echo "instance $N is not running: ./start.sh $N"; exit 1; }
mkdir -p "$OUT" "$BIN" "$D/cfg/uitoolkit"
(cd "$ROOT" && CGO_ENABLED=1 go build -o "$BIN/settings" ./cmd/uitoolkit-settings) || exit 1

# Where the box and the button sit inside the window, in logical pixels from
# the client's top left, in the default 1024x860 window at scale 1. They
# differ by the toolkit caption's height: under the desktop's frame the
# client starts at the options row, under the toolkit's it starts at the
# caption above it.
#
# BOX_X is the middle of the "OS window borders" check box, which is the
# fourth and last of the options on the first line of the folding row. It
# moves whenever that row changes: the box was renamed and the row reordered
# for the renderer chooser, and 624 — where it was — now lands on "OS
# open/save dialogs". Re-measure it rather than guess; the numbers come out
# of widget.DeviceOrigin on the box in a 1024x860 headless Settings, which is
# what cmd/uitoolkit-settings/settingsapp/shot_test.go does for the preview.
BOX_X=835; APPLY_X=980
SERVER_BOX_Y=21; SERVER_APPLY_Y=841
CLIENT_BOX_Y=47; CLIENT_APPLY_Y=834

geom() { # prints "FX FY FW FH CX CY CW CH" for the one normal window
  "$RIG/kwin.py" "$N" 'for (const w of workspace.windowList()) if (w.normalWindow) {
     const f = w.frameGeometry, c = w.clientGeometry;
     OUT(f.x + " " + f.y + " " + f.width + " " + f.height + " " + c.x + " " + c.y + " " + c.width + " " + c.height) }'
}

decorated() { # 0 when the desktop draws the frame, 1 when the toolkit does
  read -r fx fy fw fh cx cy cw ch <<<"$(geom)"
  [ -n "${ch:-}" ] || { echo "no window on instance $N" >&2; exit 1; }
  [ "${fh%.*}" = "${ch%.*}" ] && [ "${fy%.*}" = "${cy%.*}" ] && echo 1 || echo 0
}

start() { # start Settings with PACK and a decorations preference
  cat > "$D/cfg/uitoolkit/look.json" <<EOF
{"theme":"$PACK","corners":"theme","iconSize":"small","reduceMotion":true,"decorations":"$1"}
EOF
  UITK_BACKEND="$BACKEND" RUN_WAIT=5 "$RIG/run.sh" "$N" "$BIN/settings" >/dev/null || exit 1
}

toggle() { # click the OS window borders box and Apply, with the offsets of mode $1
  read -r fx fy fw fh cx cy cw ch <<<"$(geom)"
  local by=$SERVER_BOX_Y ay=$SERVER_APPLY_Y
  [ "$1" = client ] && { by=$CLIENT_BOX_Y; ay=$CLIENT_APPLY_Y; }
  "$RIG/in.sh" "$N" move $(( ${cx%.*} + BOX_X )) $(( ${cy%.*} + by )) sleep 250 click sleep 500
  "$RIG/in.sh" "$N" move $(( ${cx%.*} + APPLY_X )) $(( ${cy%.*} + ay )) sleep 250 click sleep 2500
}

fail=0
say() { printf '%-46s %s\n' "$1" "$2"; }
wrote() { grep -o '"decorations": "[a-z]*"' "$D/cfg/uitoolkit/look.json" || echo '(nothing: did the click miss Apply?)'; }

# 1. The desktop's frame, unticked: the theme takes it over, live.
start system
"$RIG/shot.sh" "$N" "fs-$BACKEND-$PACK-1-system" >/dev/null
mv "$D/shots/fs-$BACKEND-$PACK-1-system.png" "$OUT/"
[ "$(decorated)" = 0 ] || { say "$BACKEND/$PACK: starts with the desktop's frame" FAIL; fail=1; }
toggle server
grep -q '"decorations": "toolkit"' "$D/cfg/uitoolkit/look.json" ||
  { say "$BACKEND/$PACK: Apply wrote the toolkit preference" "FAIL: $(wrote)"; fail=1; }
"$RIG/shot.sh" "$N" "fs-$BACKEND-$PACK-2-toolkit" >/dev/null
mv "$D/shots/fs-$BACKEND-$PACK-2-toolkit.png" "$OUT/"
if [ "$(decorated)" = 1 ]; then say "$BACKEND/$PACK: untick hands the frame to the theme" ok
else say "$BACKEND/$PACK: untick hands the frame to the theme" "FAIL (the desktop kept it)"; fail=1; fi

# 2. And back: a window that started with the theme's frame gives it up.
start toolkit
[ "$(decorated)" = 1 ] || { say "$BACKEND/$PACK: starts with the theme's frame" FAIL; fail=1; }
toggle client
grep -q '"decorations": "system"' "$D/cfg/uitoolkit/look.json" ||
  { say "$BACKEND/$PACK: Apply wrote the system preference" "FAIL: $(wrote)"; fail=1; }
"$RIG/shot.sh" "$N" "fs-$BACKEND-$PACK-3-back" >/dev/null
mv "$D/shots/fs-$BACKEND-$PACK-3-back.png" "$OUT/"
if [ "$(decorated)" = 0 ]; then say "$BACKEND/$PACK: tick hands the frame back to the desktop" ok
else say "$BACKEND/$PACK: tick hands the frame back to the desktop" "FAIL (the toolkit kept it)"; fail=1; fi

kill "$(cat "$D/app.pid" 2>/dev/null)" 2>/dev/null
echo "shots in $OUT (1-system, 2-toolkit, 3-back)"
exit $fail
