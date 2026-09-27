#!/bin/bash
# resize-tear.sh N BIN [PASSES] — drag the window's right edge in and out,
# shoot all the while, and score every frame for a torn window (tear.py).
# Exits non-zero if any frame is torn.
#
# This is the guard for the class of bug where the compositor is shown a
# paint target that was reallocated by a resize and never painted into:
# the GPU path allocates a new colour texture with no contents, so the
# window comes out as recycled video memory. It needs a real compositor,
# a real EGL device and a real interactive resize, which is why it lives
# here and not in `go test`.
#
#   ./start.sh 7 2560 1600
#   WAYLAND_DISPLAY=uitk-e2e-7 DBUS_SESSION_BUS_ADDRESS=$(cat 7/bus.addr) \
#     kscreen-doctor output.Virtual-0.scale.1.75     # optional: fractional
#   UITK_PAINT=gpu ./resize-tear.sh 7 ./uitoolkit-settings
#   UITK_BACKEND=x11 UITK_PAINT=gpu ./resize-tear.sh 7 ./uitoolkit-settings
#
# Pass env to the app the way run.sh does, in front of the command. The
# drag is 140 steps either way, and 9 screenshots go off during each pass,
# so a torn frame that lasts one compositor frame is still caught over a
# few passes. Shots land in N/shots/tear-p<pass>-<k>.png.
RIG="$(cd "$(dirname "$0")" && pwd)"
N="${1:?instance}"; BIN="${2:?binary}"; PASSES="${3:-3}"; D="$RIG/$N"
"$RIG/run.sh" "$N" "$BIN" >/dev/null || exit 1
geom=$("$RIG/kwin.py" "$N" 'for (const w of workspace.windowList()) if (w.normalWindow) OUT(w.frameGeometry)')
read X Y W H < <(echo "$geom" | sed -E 's/.*RectF\(([0-9.-]+), ([0-9.-]+), ([0-9.-]+), ([0-9.-]+)\).*/\1 \2 \3 \4/')
[ -n "$H" ] || { echo "no window on instance $N: $geom" >&2; exit 1; }
EX=$(python3 -c "print(int($X + $W) + 1)"); EY=$(python3 -c "print(int($Y + $H / 2))")
rm -f "$D/shots/tear"-*.png
for pass in $(seq 1 "$PASSES"); do
  if [ $((pass % 2)) = 1 ]; then A=$EX; B=-260; else A=$((EX - 260)); B=260; fi
  args=(move "$A" "$EY" sleep 100 down)
  for i in $(seq 1 140); do args+=(move $((A + B * i / 140)) "$EY" sleep 20); done
  args+=(sleep 150 up)
  "$RIG/in.sh" "$N" "${args[@]}" >/dev/null 2>&1 &
  inj=$!
  sleep 0.5
  for k in $(seq 1 9); do "$RIG/shot.sh" "$N" "tear-p$pass-$k" >/dev/null; sleep 0.08; done
  wait "$inj"
  sleep 0.4
done
# kwin.py speaks logical pixels and a screenshot is device pixels; the box
# is the top-left of the window's content, which the drag never moves.
shot=$(ls "$D/shots/tear"-*.png | head -1)
logw=$("$RIG/kwin.py" "$N" 'OUT(workspace.workspaceWidth)')
read BX0 BY0 BX1 BY1 < <(python3 "$RIG/tear.py" --box "$shot" "$logw" "$X" "$Y") || exit 1
python3 "$RIG/tear.py" "$BX0" "$BY0" "$BX1" "$BY1" "$D/shots/tear"-*.png
