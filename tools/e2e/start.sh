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
  # KWin's log (and its scripts' print(), which kwin.py reads back) goes to
  # N/kwin.log: without a terminal Qt would write into the user's journal.
  export QT_FORCE_STDERR_LOGGING=1 QT_LOGGING_RULES="${QT_LOGGING_RULES:-kwin_scripting.debug=true;js.debug=true;qml.debug=true}"
  # KWin's own settings stay in the instance: with the user's config dir a
  # nested session saved its virtual output (and any kscreen-doctor scale)
  # into their real ~/.config/kwinoutputconfig.json.
  export XDG_CONFIG_HOME="$D/kwin-config" XDG_CACHE_HOME="$D/kwin-cache"
  mkdir -p "$XDG_CONFIG_HOME" "$XDG_CACHE_HOME"
  # The desktop's frame is Breeze, as on a stock Plasma: with no kwinrc of
  # its own a nested KWin falls back on another decoration.
  grep -qs '^\[org.kde.kdecoration2\]' "$XDG_CONFIG_HOME/kwinrc" ||
    printf '\n[org.kde.kdecoration2]\nlibrary=org.kde.breeze\ntheme=Breeze\n' >> "$XDG_CONFIG_HOME/kwinrc"
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
# Whatever the instance's bus starts by D-Bus activation (a portal backend,
# a notification server) starts in the nested session with the instance's
# own dirs: with the environment dbus-run-session gave it, a Qt or GTK
# service would connect to wayland-0 — the user's desktop.
DBUS_SESSION_BUS_ADDRESS="$(cat "$D/bus.addr")" dbus-update-activation-environment \
  WAYLAND_DISPLAY="uitk-e2e-$N" DISPLAY="$(cat "$D/display")" \
  XDG_CONFIG_HOME="$D/cfg" XDG_DATA_HOME="$D/data" XDG_CACHE_HOME="$D/cache" 2>/dev/null
sleep 1
echo "instance $N up: WAYLAND_DISPLAY=uitk-e2e-$N DISPLAY=$(cat "$D/display")"
