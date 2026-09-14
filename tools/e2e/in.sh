#!/bin/bash
# in.sh N CMD... — inject input into instance N (see inj.c: move X Y | click [left|right|middle] | down | up | wheel DY | key CODE | key+ CODE | key- CODE | sleep MS)
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; shift
WAYLAND_DISPLAY=uitk-e2e-$N exec "$RIG/inj" "$@"
