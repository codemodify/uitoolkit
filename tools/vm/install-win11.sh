#!/usr/bin/env bash
# install-win11.sh [ISO] — install the Windows 11 test guest, unattended.
#
# Leaves ~/VMs/uitk/win11/disk.qcow2 with a Windows 11 that logs itself
# in as uitk / uitk-test and is ready for tools/test-windows.sh. Nothing
# is typed by hand: unattend/win11.xml answers every page, and this is
# the whole of the setup, so the guest can be thrown away and remade.
#
#   tools/vm/install-win11.sh                     # the ISO in UITK_WIN11_ISO
#   tools/vm/install-win11.sh /path/to/win11.iso
#
# It takes 20 to 40 minutes and nothing watches it: poll with
#   tools/vm/vm.sh shot win11 /tmp/w.png
# and the install is finished when C:\uitk\READY.txt exists in the guest
# (the last thing the unattend does), which is also when the guest will
# be sitting on a logged-in desktop.
#
# Why a fresh install rather than the Windows 11 VM already on this box:
# that one belongs to another project, its password is not the one in its
# own unattend any more, and a test fixture nobody can log into is not a
# fixture. This one is the repository's: known credentials, written down
# here, and remakeable from an ISO in one command.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
ISO=${1:-${UITK_WIN11_ISO:-}}
[ -n "$ISO" ] || { echo "give me a Windows 11 ISO (or set UITK_WIN11_ISO)" >&2; exit 2; }
[ -f "$ISO" ] || { echo "no ISO at $ISO" >&2; exit 2; }

DIR=${UITK_WIN11_DIR:-$HOME/VMs/uitk/win11}
RUN="${XDG_RUNTIME_DIR:-/tmp}/uitk-vm"
OVMF_CODE=${UITK_VM_OVMF_CODE:-/usr/share/edk2/x64/OVMF_CODE.4m.fd}
OVMF_VARS=${UITK_VM_OVMF_VARS:-/usr/share/edk2/x64/OVMF_VARS.4m.fd}
mkdir -p "$DIR" "$RUN"

# qemu splits a -drive argument on commas, so a filename with one in it
# has to double them. The Windows ISOs people actually have are named
# things like "Windows 11 [10.0.26100], Version 24H2 (Updated June
# 2025).iso", and without this qemu reports "Could not open
# '/.../Windows 11 [10.0.26100]'" — the path cut at the comma.
qesc() { printf '%s' "${1//,/,,}"; }

if [ -f "$DIR/disk.qcow2" ]; then
  echo "$DIR/disk.qcow2 is already there." >&2
  echo "Move it aside first if you mean to install over it." >&2
  exit 2
fi

echo "== the answer disc =="
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
cp "$HERE/unattend/win11.xml" "$work/autounattend.xml"
xorriso -as mkisofs -quiet -V UITKANSWER -J -r -o "$DIR/unattend.iso" "$work"

echo "== a 64 GB disk =="
qemu-img create -f qcow2 "$DIR/disk.qcow2" 64G >/dev/null
cp "$OVMF_VARS" "$DIR/OVMF_VARS.fd"

echo "== booting the installer (vnc 127.0.0.1:5911, monitor $RUN/win11.mon) =="
rm -f "$RUN/win11.mon"
nohup qemu-system-x86_64 \
  -name uitk-win11-install \
  -machine q35,accel=kvm,smm=on \
  -cpu host -smp 4 -m 6144 \
  -drive if=pflash,format=raw,unit=0,readonly=on,file="$OVMF_CODE" \
  -drive if=pflash,format=raw,unit=1,file="$(qesc "$DIR/OVMF_VARS.fd")" \
  -device ich9-ahci,id=ahci \
  -drive file="$(qesc "$DIR/disk.qcow2")",if=none,id=hd0,format=qcow2,cache=writeback \
  -device ide-hd,drive=hd0,bus=ahci.0,bootindex=1 \
  -drive file="$(qesc "$ISO")",if=none,id=cd0,media=cdrom,readonly=on \
  -device ide-cd,drive=cd0,bus=ahci.1,bootindex=0 \
  -drive file="$(qesc "$DIR/unattend.iso")",if=none,id=cd1,media=cdrom,readonly=on \
  -device ide-cd,drive=cd1,bus=ahci.2 \
  -netdev user,id=net0 -device e1000e,netdev=net0 \
  -device qemu-xhci,id=usb -device usb-tablet \
  -vga std -display none \
  -vnc 127.0.0.1:11 \
  -monitor unix:"$RUN/win11.mon",server,nowait \
  -pidfile "$RUN/win11.pid" \
  -rtc base=localtime \
  >"$RUN/win11-install.log" 2>&1 &
sleep 3
kill -0 "$(cat "$RUN/win11.pid")" 2>/dev/null ||
  { tail -5 "$RUN/win11-install.log" >&2; echo "the installer did not start" >&2; exit 1; }

# The first boot needs a key pressed: the disk is empty, so the firmware
# falls to the ISO, and the ISO's boot loader asks "Press any key to boot
# from CD or DVD" and gives up after about five seconds — leaving a
# machine with nothing to boot, which reboots, and so on for ever.
#
# So: a key every two seconds, for twenty seconds, and then **stop**.
# Stopping matters more than starting. Setup's own window appears about a
# minute in with a Cancel button on it, and keys still arriving then land
# on Cancel and open "Are you sure you want to quit?" — which is how this
# was first written and what it cost to find out.
echo "== nudging it past "press any key to boot from CD" =="
for _ in $(seq 10); do
  printf 'sendkey spc 120\n' | socat - UNIX-CONNECT:"$RUN/win11.mon" >/dev/null 2>&1 || true
  sleep 2
done

cat <<MSG

Installing. Nothing to do but wait (20-40 minutes).

  watch it      tools/vm/vm.sh shot win11 /tmp/w11.png
  or live       vncviewer 127.0.0.1:5911
  finished      a logged-in desktop, and C:\\uitk\\READY.txt in the guest

The installer reboots a few times; the disk is bootindex 1 and the ISO
bootindex 0, so each reboot tries the ISO first and Windows setup's own
"press any key to boot from CD" prompt times out into the disk, which is
what carries the install forward without anybody pressing anything.

When it is done, take the ISOs away and start it with the rig:

  tools/vm/vm.sh start win11
MSG
