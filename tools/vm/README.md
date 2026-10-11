# The Windows test guests

`tools/test-windows.sh` needs a **logged-in** Windows desktop and a qemu
monitor socket to drive it with. Before this, both were whatever the
person running it happened to have: a VM started by hand, a password
somebody remembered, a monitor socket in a different place each time.
These are the guests, written down.

```bash
tools/vm/vm.sh status                 # what exists and what is up
tools/vm/vm.sh start win10            # boot one
UITK_WIN_VM=win10 tools/test-windows.sh   # start it if needed, run the suite
tools/vm/vm.sh shot win10 /tmp/w.png  # look at it
tools/vm/vm.sh stop win10
```

`vms.conf` lists them: disk, RAM, CPUs, VNC display, firmware. It is
committed because the shape of it is the documentation — change the
paths to the ones on your machine.

| guest | what it is | login |
| --- | --- | --- |
| `win10` | this machine's own general-purpose Windows 10, adopted rather than duplicated: it already auto-logs in, and a second 50 GB image would buy nothing | its own, automatic |
| `win11` | the repository's own, installed by `install-win11.sh` from a Windows 11 ISO | `uitk` / `uitk-test`, automatic |

## Headless, and driven through the monitor

Nothing opens on the desktop this box is running: the guests are
`-display none` with VNC for when you want to watch (`vncviewer
127.0.0.1:5910` for win10, `:5911` for win11). Everything else goes
through the qemu monitor — `screendump` for a picture, `sendkey` to type
— so a run cannot be disturbed by, or disturb, whatever the person is
doing on their screen.

Two things about that monitor, both learned the hard way:

* **The socket path is capped at 108 bytes.** They live under
  `$XDG_RUNTIME_DIR/uitk-vm/`, not beside the disk, because a scratch
  directory's path is longer than that and qemu refuses to start with a
  message about byte counts.
* **Windows drops a burst of keys.** `sendkey` at the monitor's natural
  speed loses most of them — a ten-character password arrived as three,
  which looks exactly like a wrong password. `vm.sh type` holds each key
  120ms and waits 250ms between them.

## Making the Windows 11 guest

```bash
tools/vm/install-win11.sh /path/to/win11.iso
```

20 to 40 minutes, unattended: `unattend/win11.xml` answers every page.
It turns off the TPM and secure-boot checks (this host has no `swtpm`,
and 24H2's setup stops on the first screen without them), creates
`uitk` / `uitk-test` with autologon, excludes `C:\uitk` from Defender —
which otherwise quarantines freshly built unsigned Go binaries, and a
test binary that vanishes between being written and being run looks
exactly like a broken build — and turns off sleep, hibernation and the
lock screen, because a guest that sleeps halfway through a run fails it.

When it is done, take the ISO lines out of nothing: the installer runs in
its own qemu invocation, so `tools/vm/vm.sh start win11` after it is
finished boots the installed system with no ISOs at all.

## Why not the Windows 11 VM already on this machine

`~/VMs/icwin` is another project's guest. Its own `autounattend.xml` no
longer has the password the account actually uses, so nobody can log into
it, and a test fixture nobody can log into is not a fixture. It also
wants `swtpm`, which this host does not have installed. The one here is
the repository's: known credentials, written down above, and remakeable
from an ISO in one command.
