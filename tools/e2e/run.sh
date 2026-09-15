#!/bin/bash
# run.sh N BIN [ARGS...] — kill instance N's previous app, launch BIN (Wayland backend) in it.
# Extra env: pass as `env` before, e.g.  UITK_BACKEND=x11 ./run.sh 3 ./gallery
# Uses private XDG_CONFIG_HOME/XDG_DATA_HOME under $RIG/N so it never touches ~/.config,
# and the instance's own D-Bus session (APP_BUS overrides): on the user's bus an
# app's tray icon, notifications and portal dialogs would reach the real desktop.
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; shift; BIN="$1"; shift; D="$RIG/$N"
if [ -f "$D/app.pid" ]; then kill "$(cat "$D/app.pid")" 2>/dev/null; sleep 0.4; kill -9 "$(cat "$D/app.pid")" 2>/dev/null; fi
WAYLAND_DISPLAY=uitk-e2e-$N DISPLAY="$(cat "$D/display" 2>/dev/null)" XDG_CONFIG_HOME="${XDG_CONFIG_HOME_OVERRIDE:-$D/cfg}" XDG_DATA_HOME="$D/data" \
  DBUS_SESSION_BUS_ADDRESS="${APP_BUS:-$(cat "$D/bus.addr")}" \
  UITK_MAIL_NO_OPEN=1 UITK_MAIL_NO_NOTIFY=1 "$BIN" "$@" > "$D/app.log" 2>&1 &
echo $! > "$D/app.pid"
sleep "${RUN_WAIT:-2.5}"
kill -0 "$(cat "$D/app.pid")" 2>/dev/null && echo "running pid $(cat "$D/app.pid")" || { echo "APP EXITED"; tail -20 "$D/app.log"; }
