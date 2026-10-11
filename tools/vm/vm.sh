#!/usr/bin/env bash
# vm.sh start|stop|status|shot|keys|type|mon NAME [...] — the Windows test
# guests, from here.
#
#   tools/vm/vm.sh start win10          boot it (no-op if it is up)
#   tools/vm/vm.sh status               what is running
#   tools/vm/vm.sh shot win10 out.png   a picture of its screen
#   tools/vm/vm.sh type win11 'hello'   type into it
#   tools/vm/vm.sh keys win11 ret       press keys
#   tools/vm/vm.sh stop win10           ask it to shut down, then wait
#
# Why this exists: tools/test-windows.sh needs a *logged-in* Windows guest
# and a qemu monitor socket to drive it with, and before this both were
# whatever the person running it had lying around. The guests are listed
# in vms.conf; everything else here is the same for all of them.
#
# Headless on purpose. A guest is driven through its monitor socket —
# `screendump` for a picture, `sendkey` to type — so nothing opens on the
# desktop this box is running and a run cannot be disturbed by, or
# disturb, whatever the person is doing. VNC is there for when you want
# to watch: `vncviewer :5910` for win10.
#
# The monitor socket lives under $XDG_RUNTIME_DIR, not beside the disk: a
# unix socket path is capped at 108 bytes and a scratch directory is
# deeper than that. That cap has cost an hour before.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
RUN="${XDG_RUNTIME_DIR:-/tmp}/uitk-vm"
mkdir -p "$RUN"

OVMF_CODE=${UITK_VM_OVMF_CODE:-/usr/share/edk2/x64/OVMF_CODE.4m.fd}
OVMF_VARS=${UITK_VM_OVMF_VARS:-/usr/share/edk2/x64/OVMF_VARS.4m.fd}

die() { echo "vm.sh: $*" >&2; exit 1; }

# qesc doubles the commas in a path: qemu splits a -drive argument on
# them, so a disk under a directory with a comma in its name cannot be
# opened without this.
qesc() { printf '%s' "${1//,/,,}"; }

# conf NAME reads the guest's line into the globals the rest uses.
conf() {
  local name=$1 line
  line=$(grep -E "^$name[[:space:]]" "$HERE/vms.conf" | head -1) ||
    die "no guest called $name in vms.conf"
  [ -n "$line" ] || die "no guest called $name in vms.conf"
  # shellcheck disable=SC2086
  set -- $line
  VM_NAME=$1
  VM_DISK=${2/#\~/$HOME}
  VM_RAM=$3
  VM_CPUS=$4
  VM_VNC=$5
  VM_FIRMWARE=$6
  VM_MON="$RUN/$VM_NAME.mon"
  VM_PID="$RUN/$VM_NAME.pid"
  VM_LOG="$RUN/$VM_NAME.log"
  VM_VARS="$(dirname "$VM_DISK")/OVMF_VARS.fd"
}

running() { [ -s "$VM_PID" ] && kill -0 "$(cat "$VM_PID")" 2>/dev/null; }

start() {
  conf "$1"
  if running; then echo "$VM_NAME is up (pid $(cat "$VM_PID"), monitor $VM_MON)"; return 0; fi
  [ -f "$VM_DISK" ] || die "$VM_NAME has no disk at $VM_DISK (install it first)"
  # Somebody else's qemu may already have this disk open — the guests here
  # are the machine's own and a person starts them by hand too. Two
  # writers on one qcow2 is a corrupted image, not a clash of locks, so
  # this refuses rather than races.
  if pgrep -af "qemu-system.*$VM_DISK" | grep -qv "^$$ "; then
    die "something already has $VM_DISK open:
$(pgrep -af "qemu-system.*$VM_DISK" | sed 's/^/    /' | cut -c1-120)"
  fi
  rm -f "$VM_MON"
  local args=(
    -name "uitk-$VM_NAME"
    -machine q35,accel=kvm,smm=on
    -cpu host -smp "$VM_CPUS" -m "$VM_RAM"
    -device ich9-ahci,id=ahci
    -drive file="$(qesc "$VM_DISK")",if=none,id=hd0,format=qcow2,cache=writeback
    -device ide-hd,drive=hd0,bus=ahci.0,bootindex=0
    -netdev user,id=net0
    -device e1000e,netdev=net0
    -device qemu-xhci,id=usb -device usb-tablet
    -vga std -display none
    -vnc "127.0.0.1:$VM_VNC"
    -monitor unix:"$VM_MON",server,nowait
    -pidfile "$VM_PID"
    -rtc base=localtime
  )
  if [ "$VM_FIRMWARE" = uefi ]; then
    [ -f "$VM_VARS" ] || cp "$OVMF_VARS" "$VM_VARS"
    args+=(
      -drive if=pflash,format=raw,unit=0,readonly=on,file="$(qesc "$OVMF_CODE")"
      -drive if=pflash,format=raw,unit=1,file="$(qesc "$VM_VARS")"
    )
  fi
  # A TPM only when the host can emulate one. Windows 11 wants one to
  # *install* (install-win11.sh tells setup not to look) and runs without
  # it; adding one when swtpm turns up costs nothing and loses nothing.
  if command -v swtpm >/dev/null; then
    local tpmdir; tpmdir="$(dirname "$VM_DISK")/tpm"
    mkdir -p "$tpmdir"
    if ! pgrep -f "swtpm.*$tpmdir" >/dev/null; then
      swtpm socket --tpmstate dir="$tpmdir" --ctrl type=unixio,path="$RUN/$VM_NAME.tpm" \
        --tpm2 --daemon --log level=0
      sleep 1
    fi
    args+=(-chardev socket,id=chrtpm,path="$RUN/$VM_NAME.tpm"
      -tpmdev emulator,id=tpm0,chardev=chrtpm -device tpm-tis,tpmdev=tpm0)
  fi
  shift
  nohup qemu-system-x86_64 "${args[@]}" "$@" >"$VM_LOG" 2>&1 &
  sleep 2
  running || { tail -5 "$VM_LOG" >&2; die "$VM_NAME did not start"; }
  echo "$VM_NAME up: monitor $VM_MON, vnc 127.0.0.1:59$VM_VNC, log $VM_LOG"
}

# mon sends one monitor command and prints what came back.
mon() {
  conf "$1"; shift
  running || die "$VM_NAME is not running"
  printf '%s\n' "$*" | socat - UNIX-CONNECT:"$VM_MON" 2>/dev/null | tail -n +2
}

stop() {
  conf "$1"
  running || { echo "$VM_NAME is not running"; return 0; }
  printf 'system_powerdown\n' | socat - UNIX-CONNECT:"$VM_MON" >/dev/null 2>&1 || true
  for _ in $(seq 60); do running || { echo "$VM_NAME is down"; return 0; }; sleep 2; done
  # It did not take the ACPI button: a guest at a login screen or in a
  # modal dialog often will not. Pull the plug — these are test guests.
  printf 'quit\n' | socat - UNIX-CONNECT:"$VM_MON" >/dev/null 2>&1 || true
  sleep 2
  echo "$VM_NAME killed (it ignored the power button)"
}

shot() {
  conf "$1"
  local out=${2:?a file to write}
  running || die "$VM_NAME is not running"
  local ppm="$RUN/$VM_NAME.ppm"
  printf 'screendump %s\n' "$ppm" | socat - UNIX-CONNECT:"$VM_MON" >/dev/null 2>&1
  sleep 0.5
  if [ "${out##*.}" = ppm ]; then cp "$ppm" "$out"; else magick "$ppm" "$out"; fi
  echo "$out"
}

# sendkeys pushes keys one at a time, slowly. Windows drops most of a
# burst sent at the monitor's natural speed: a ten-character password
# arrived as three characters, which looks exactly like a wrong password.
sendkeys() {
  local sock=$1; shift
  local k
  for k in "$@"; do
    printf 'sendkey %s 120\n' "$k" | socat - UNIX-CONNECT:"$sock" >/dev/null 2>&1
    sleep 0.25
  done
}

keys() { conf "$1"; shift; running || die "$VM_NAME is not running"; sendkeys "$VM_MON" "$@"; }

type_text() {
  conf "$1"; shift
  running || die "$VM_NAME is not running"
  local text=$* out=()
  local i ch
  for ((i = 0; i < ${#text}; i++)); do
    ch=${text:i:1}
    out+=("$(keyname "$ch")")
  done
  sendkeys "$VM_MON" "${out[@]}"
}

# keyname is the qemu key for one character.
keyname() {
  local ch=$1
  case $ch in
    [a-z]) printf '%s' "$ch" ;;
    [A-Z]) printf 'shift-%s' "$(printf '%s' "$ch" | tr 'A-Z' 'a-z')" ;;
    [0-9]) printf '%s' "$ch" ;;
    ' ') printf 'spc' ;;
    '!') printf 'shift-1' ;;
    '@') printf 'shift-2' ;;
    '#') printf 'shift-3' ;;
    '$') printf 'shift-4' ;;
    '%') printf 'shift-5' ;;
    '^') printf 'shift-6' ;;
    '&') printf 'shift-7' ;;
    '*') printf 'shift-8' ;;
    '(') printf 'shift-9' ;;
    ')') printf 'shift-0' ;;
    '-') printf 'minus' ;;
    '_') printf 'shift-minus' ;;
    '=') printf 'equal' ;;
    '+') printf 'shift-equal' ;;
    '.') printf 'dot' ;;
    ',') printf 'comma' ;;
    '/') printf 'slash' ;;
    '\') printf 'backslash' ;;
    ':') printf 'shift-semicolon' ;;
    ';') printf 'semicolon' ;;
    *) echo "vm.sh: no key for '$ch'" >&2; exit 1 ;;
  esac
}

status() {
  local name disk rest
  while read -r name disk rest; do
    case $name in ''|\#*) continue ;; esac
    conf "$name"
    if running; then
      printf '%-6s up    pid %-7s monitor %s\n' "$VM_NAME" "$(cat "$VM_PID")" "$VM_MON"
    elif pgrep -f "qemu-system.*$VM_DISK" >/dev/null; then
      # Somebody started it by hand. Worth saying rather than calling it
      # down: "down" is what makes a person start a second qemu on the
      # same disk, which is how a qcow2 is corrupted.
      printf '%-6s up    (not this rig'"'"'s: pid %s)\n' "$VM_NAME"         "$(pgrep -f "qemu-system.*$VM_DISK" | head -1)"
    elif [ -f "$VM_DISK" ]; then
      printf '%-6s down  disk %s\n' "$VM_NAME" "$VM_DISK"
    else
      printf '%-6s none  no disk at %s\n' "$VM_NAME" "$VM_DISK"
    fi
  done < "$HERE/vms.conf"
}

cmd=${1:-status}; shift || true
case $cmd in
  start) start "$@" ;;
  stop) stop "$@" ;;
  status) status ;;
  shot) shot "$@" ;;
  keys) keys "$@" ;;
  type) type_text "$@" ;;
  mon) mon "$@" ;;
  *) sed -n '2,20p' "$0"; exit 2 ;;
esac
