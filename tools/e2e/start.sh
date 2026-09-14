#!/bin/bash
# start.sh N [WIDTH HEIGHT [SCALE]] — start (or reuse) nested KWin instance N.
# Wayland socket: uitk-e2e-N   Xwayland display: in $RIG/N/display   state: $RIG/N/
RIG="$(cd "$(dirname "$0")" && pwd)"
N="${1:?instance number}"; W="${2:-1280}"; H="${3:-860}"; S="${4:-1}"
D="$RIG/$N"; mkdir -p "$D/cfg" "$D/data" "$D/shots"
if [ -S "/run/user/$(id -u)/uitk-e2e-$N" ] && [ -f "$D/kwin.pid" ] && kill -0 "$(cat "$D/kwin.pid")" 2>/dev/null; then
  echo "instance $N already running"; exit 0
fi
rm -f "/run/user/$(id -u)/uitk-e2e-$N" "/run/user/$(id -u)/uitk-e2e-$N.lock"
before=$(ls /tmp/.X11-unix/)
(
  export KWIN_SCREENSHOT_NO_PERMISSION_CHECKS=1 KWIN_WAYLAND_NO_PERMISSION_CHECKS=1
  unset DISPLAY WAYLAND_DISPLAY
  exec setsid dbus-run-session -- bash -c "echo \$DBUS_SESSION_BUS_ADDRESS > '$D/bus.addr'; exec kwin_wayland --virtual --no-lockscreen --no-global-shortcuts --socket uitk-e2e-$N --xwayland --width $W --height $H --scale $S"
) > "$D/kwin.log" 2>&1 &
echo $! > "$D/kwin.pid"
for i in $(seq 1 50); do [ -S "/run/user/$(id -u)/uitk-e2e-$N" ] && [ -s "$D/bus.addr" ] && break; sleep 0.1; done
for i in $(seq 1 50); do
  new=$(comm -13 <(echo "$before" | sort) <(ls /tmp/.X11-unix/ | sort) | head -1)
  [ -n "$new" ] && break; sleep 0.1
done
echo ":${new#X}" > "$D/display"
sleep 1
echo "instance $N up: WAYLAND_DISPLAY=uitk-e2e-$N DISPLAY=$(cat "$D/display")"
