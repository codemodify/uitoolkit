# Mail — mailclientd + mailclientui

Thunderbird-chrome mail client on uitoolkit **v0.10.13**. Two processes:

| Process | Role |
| --- | --- |
| **mailclientd** | Owns accounts, IMAP/POP3/SMTP, OAuth tokens, local cache, folders, messages, tags, filters, identities, smart folders, VIP, categories, outbox, search, mutations. |
| **mailclientui** | Renders chrome, owns the **status item / tray**, and sends JSON-RPC commands. **No IMAP, POP3, SMTP, or OAuth HTTP** in this process. |

Shared types and the RPC client/server live in [`internal/mail`](../internal/mail).

## How to start

Default socket:

- `$UITK_MAIL_SOCK` if set
- else `$XDG_RUNTIME_DIR/mailclientd.sock`
- else `/tmp/mailclientd-<uid>.sock`

```bash
# Terminal 1 — daemon (empty until you add an account)
go run ./cmd/mailclientd

# Terminal 2 — Thunderbird UI (first-run Yes/No if no accounts)
UITK_SCENE=auto go run ./cmd/mailclientui
go run ./cmd/mailclientui -classic -light
go run ./cmd/mailclientui -headless    # writes mail.png

# Seeded MemoryStore dogfood / screenshots only:
UITK_MAIL=memory go run ./cmd/mailclientd
```

Convenience (same architecture, one process: daemon goroutine + UI client on a temp socket):

```bash
UITK_SCENE=auto go run ./examples/mail
go run ./examples/mail -screenshot docs/screenshots
```

## Tray and new-mail toasts (0.16.0, 0.16.1)

`mailclientui` opens a toolkit `StatusItem` while it runs
(v0.16.3: Plasma right-click uses `Menu=/NO_DBUSMENU` and a window-corner
toolkit popup. v0.16.2: envelope tray icon; Wayland Show Mail remaps
after close-to-tray. v0.16.1: Plasma `GetLayout` no longer panics):

- **Click** the tray (or the toast) → show / raise / focus the Mail window
  (create it if it was closed). The window-manager close button **hides**
  to the tray while the item is alive; **Quit** on the tray menu exits.
- **New mail** → daemon broadcasts `mail.notify` `{title, body, count}`.
  The UI shows a desktop notification (sender + subject when there is one
  message). The main window does not need to be visible.
- `notify-send` stays as a daemon fallback only when **no UI client** is
  connected. Prefs → Notify still gates `Enabled` / VIP-only / that fallback.

Linux tray hosts: see [tray.md](tray.md) (KDE SNI, GNOME AppIndicator,
Waybar, …). Headless / `CGO_ENABLED=0` tests use a stub or `UITK_TRAY=fake`.

## Real IMAP or POP3 + SMTP (primary path)

`mailclientd` is meant to be pointed at a real account. **Add Account** takes a typed (masked) password and writes it into `mail.json`. That file is **mode `0600`**. The password field is **temporary plaintext** until a secret store exists. OAuth (encrypted refresh token) and optional `passEnv` / `UITK_MAIL_PASS` still work when `password` is empty.

Config file (first existing wins):

- `$UITK_MAIL_CONFIG`
- `$XDG_CONFIG_HOME/uitoolkit/mail.json`
- `~/.config/uitoolkit/mail.json`

Cache / offline store:

- `$UITK_MAIL_DATA`
- `$XDG_DATA_HOME/uitoolkit/mail`
- `~/.local/share/uitoolkit/mail`

Example `mail.json` (written mode `0600`; `password` is temporary plaintext):

```json
{
  "accounts": [
    {
      "id": "home",
      "name": "Ada Lovelace",
      "address": "ada@example.com",
      "protocol": "imap",
      "imap": {
        "host": "imap.example.com:993",
        "user": "ada@example.com",
        "password": "your-app-password",
        "tls": true
      },
      "smtp": {
        "host": "smtp.example.com:587",
        "user": "ada@example.com",
        "password": "your-app-password",
        "starttls": true
      }
    }
  ]
}
```

```bash
# File → Add Account, type the password, Save account
go run ./cmd/mailclientd
go run ./cmd/mailclientui
```

Optional env fallback when `password` is omitted: `"passEnv": "UITK_MAIL_PASS"` and `export UITK_MAIL_PASS='…'`.

Single-account env (no file) still works:

```bash
export UITK_MAIL=imap
export UITK_MAIL_HOST=imap.example.com:993
export UITK_MAIL_USER=you@example.com
export UITK_MAIL_PASS=secret
export UITK_MAIL_SMTP=smtp.example.com:587   # optional; guessed from IMAP host
export UITK_MAIL_NAME='Ada Lovelace'
go run ./cmd/mailclientd
```

POP3 account (inbox retrieve; SMTP still used to send):

```json
{
  "id": "home-pop",
  "name": "Ada Lovelace",
  "address": "ada@example.com",
  "protocol": "pop3",
  "pop": {
    "host": "pop.example.com:995",
    "user": "ada@example.com",
    "password": "your-app-password",
    "tls": true
  },
  "smtp": {
    "host": "smtp.example.com:587",
    "user": "ada@example.com",
    "password": "your-app-password",
    "starttls": true
  }
}
```

Default when **no** config and `UITK_MAIL` is unset: **empty** LocalStore (no demo accounts). The UI asks *There are no accounts, want to add one?* Yes opens File → Add Account. No leaves empty chrome. Password / `mail.json` mode `0600` notes live on the Add Account form, not that Yes/No dialog.

`UITK_MAIL=memory` is the **only** way to load the seeded MemoryStore demo (and `examples/mail` / screenshots still use that on purpose).

### Add Account polish

The wizard has an explicit **IMAP** / **POP3** choice (default IMAP). It guesses hosts from the email domain (`gmail.com` → `imap.gmail.com:993` / `pop.gmail.com:995` / `smtp.gmail.com:465`, Outlook/Hotmail, Yahoo, iCloud, Fastmail, Proton, else `imap.<domain>:993` / `pop.<domain>:995` / `smtp.<domain>:587`). You can still edit the hosts.

**Test connection** dials the typed user/password and reports success or failure (which protocol worked, `host:port`, TLS mode: `ssl` / `starttls` / `plain`). After email + password are entered, a background auto-detect may also probe common `imap.` / `pop.` / `mail.` names on 993/143/995/110 (timeout + status line; it does not freeze the form).

Type the incoming/SMTP password (masked field); **Save account** writes `protocol` plus `password` into `mail.json` (file mode `0600`). `passEnv` is optional fallback only. Account Central and Preferences → Accounts show **IMAP** or **POP3** after save.

### Text-only message view

The Message tab is **plain text**. mailclientd prefers the `text/plain` part; if the message is HTML-only, tags are stripped (`HTMLToText`). There is no HTML engine and no HTML tab. **View → Message Source** (also Message → Message Source, context menu, **Ctrl+U**) opens a read-only JetBrains Mono window of the stored RFC822 (`messages.getSource`). The Source tab in the preview pane uses the same daemon bytes — not a reconstructed header dump.

## OAuth (Google + Microsoft)

Real **authorization-code + PKCE loopback** (`http://127.0.0.1:<port>/oauth/callback`) or **device code** flow. IMAP/SMTP then use **AUTH XOAUTH2**. The `UITK_MAIL_XOAUTH2` bearer passthrough, inline `password`, and `passEnv` / `UITK_MAIL_PASS` still work.

**Honest gap:** uitoolkit does **not** ship Google or Microsoft client IDs. You must register an app and supply credentials:

| Provider | Register | Env (or wizard fields) |
| --- | --- | --- |
| Google | [Google Cloud Console](https://console.cloud.google.com/apis/credentials) — OAuth client (Desktop or Web). Redirect: `http://127.0.0.1:<port>/oauth/callback`. Scope: `https://mail.google.com/` | `UITK_MAIL_OAUTH_GOOGLE_CLIENT_ID`, `UITK_MAIL_OAUTH_GOOGLE_CLIENT_SECRET` |
| Microsoft | [Azure AD app registration](https://portal.azure.com/) — public client is fine. Redirect same loopback URL, or device code (no redirect). Scopes: `offline_access`, `https://outlook.office.com/IMAP.AccessAsUser.All`, `https://outlook.office.com/SMTP.Send` | `UITK_MAIL_OAUTH_MS_CLIENT_ID`, `UITK_MAIL_OAUTH_MS_CLIENT_SECRET` (secret optional for public clients) |

```bash
export UITK_MAIL_OAUTH_GOOGLE_CLIENT_ID='….apps.googleusercontent.com'
export UITK_MAIL_OAUTH_GOOGLE_CLIENT_SECRET='…'   # if the client is confidential
go run ./cmd/mailclientd
# UI: File → Add Account → Sign in with Google
```

Device flow: **Device code…** on the same dialog (useful when loopback cannot bind or the app only allows device).

### Token storage

Refresh/access tokens are **never** written to `mail.json`. They live under `$XDG_DATA_HOME/uitoolkit/mail/secrets/` (or `$UITK_MAIL_DATA/secrets/`):

- `*.tok` — AES-256-GCM (random nonce prefix)
- `master.key` — 32 random bytes, mode `0600`, used when libsecret is unavailable
- If `secret-tool` is on `PATH`, the master key is also stored as `service=uitoolkit-mail`

This is an encrypted file (plus optional OS keyring), **not** a TPM-backed vault. Backup the `secrets/` directory with the same care as an SSH key.

Expired access tokens are refreshed with the stored refresh token.

## Sync (IDLE / QRESYNC / CONDSTORE)

After listen, LocalStore starts a **push supervisor**:

- **IDLE** on **Inbox and Sent** (one IMAP connection per watched mailbox; IMAP allows only one selected mailbox per connection).
- IDLE wake (EXISTS / FETCH / EXPUNGE / RECENT) → incremental sync of that folder.
- **QRESYNC** when the server advertises it (`ENABLE QRESYNC`): `SELECT … (QRESYNC (uidvalidity highestmodseq))`, apply `VANISHED`, then flag refresh.
- Else **CONDSTORE** `UID FETCH … (CHANGEDSINCE highestmodseq)` for flags.
- Else a full `UID FETCH 1:* (UID FLAGS)`.
- Other folders: CONDSTORE/FLAGS poll about every **2 minutes**, plus File → Get Messages.
- `UIDVALIDITY` change still wipes that folder’s cache.

Manual Get Messages / `sync.run` always does a full account pass (LIST + incremental UID FETCH).

Work Offline (`status.set`) stops treating the transport as reachable; mutations go to the outbox (below). Going online flushes the queue.

## Offline outbox

While offline (or after a transport error), **send / move / delete / flag** apply to the local cache immediately and enqueue an `OutboxOp`. The folder tree **Outbox** node lists queued sends.

Flush: File → Work Offline (toggle back on), Get Messages, or `outbox.flush`.

Conflict-safe cache:

- Pending ops skip CONDSTORE/QRESYNC flag overwrite for that UID (local mutation wins until flush).
- Flush of a vanished UID is a no-op (dropped).
- `UIDVALIDITY` change drops stale UIDs; the user re-syncs.

## Fast search + Smart folders

Daemon-side inverted index over subject / from / to / body (AND of tokens). Quick Filter and `messages.search` use it when a query is present, then apply pins.

**Smart / Search folders:** still exist on the daemon (`smart.*` RPC) but are **not shown** in the folder tree or File/Tools menus as of v0.10.4. Use Quick Filter for ad-hoc search.

## Threading + mute

Conversations group by `In-Reply-To` / `References` when Message-IDs exist, otherwise a normalized subject key. View → **Threaded** nests replies (`↳`) and shows a count on the root.

**Message → Mute Thread** (and Unmute). Muted threads:

- stay in normal folders with a 🔇 marker
- do not drive notifications
- View → Hide muted threads removes them from the list

## Attachments

Message view stays **text-only**. Each attachment row shows its name plus inline **Open** and **Save As** (toolkit `Button`). A single click on the name selects only; a double click — or that row’s **Open** — calls `messages.openPart`: mailclientd writes a cache file under the data dir (`open/` or a temp file for MemoryStore) and launches `xdg-open` (or `open` on macOS) when a display is available. `UITK_MAIL_NO_OPEN=1` skips the spawn (tests / headless). **Save As** uses the toolkit file dialog and writes `messages.part` bytes to the chosen path (mode `0600`). The attachment toolbar (where the shared Open used to sit) has **Save All**: one folder pick via the same file dialog (the confirmed path is treated as a directory — an existing file uses its parent; a missing path with no extension is created), then every attachment on the current message is written there (`0600`; `name-2.ext` on collisions).

## VIP, notifications, categories

- **VIP** senders (Message → Add sender to VIP). The VIP smart folder is **not** shown in the sidebar; Preferences still lists VIP contacts, and VIP-only notifications still work.
- **Notification rules** (Preferences → Notify): new mail, optional VIP-only, optional `notify-send` on Linux. No display / no `notify-send` → stub (RPC event `mail.notify` still fires). `UITK_MAIL_NO_NOTIFY=1` disables the desktop helper.
- **Categories** (Gmail-lite, local) still classify on the daemon; they are **not** a folder-tree section as of v0.10.4.

Calendar / iTip is **not** in this release (Tier C later).

## IMAP / POP3 / SMTP status (honest)

Implemented in mailclientd:

- IMAP: CONNECT, implicit TLS (993) and STARTTLS (143), LOGIN, AUTH PLAIN, AUTH XOAUTH2 (env bearer **or** stored OAuth token), CAPABILITY, ENABLE QRESYNC/CONDSTORE, LIST/LSUB, SELECT/EXAMINE (+ QRESYNC), UID FETCH (ENVELOPE, FLAGS, BODYSTRUCTURE, BODY.PEEK[] / sections), UID STORE, UID SEARCH, UID COPY, UID MOVE (or COPY+\\Deleted+EXPUNGE), APPEND, EXPUNGE, IDLE (Inbox+Sent supervisor), CONDSTORE CHANGEDSINCE, VANISHED when QRESYNC.
- POP3: CONNECT, implicit TLS (995) and STLS (110), USER/PASS, STAT, UIDL, RETR into the local Inbox (leave-on-server — no DELE). Incremental skip by UIDL (or RFC Message-ID if UIDL is missing). Local Drafts/Sent/Trash exist for compose.
- Incremental cache: UIDVALIDITY wipe, UIDNEXT, highestmodseq. Raw `.eml` on disk after body fetch.
- MIME: multipart (nested), text/plain + text/html, attachments, RFC 2047, charset via `golang.org/x/text`.
- SMTP: implicit TLS (465), STARTTLS (587), AUTH PLAIN / LOGIN / XOAUTH2. Send then IMAP APPEND to Sent (or outbox if offline).
- Multiple accounts in one config; folder tree mirrors LIST + local specials + virtuals.
- Offline read of anything already synced; queued mutations.

Known gaps:

- BODYSTRUCTURE walker covers common multipart/alternative + mixed; exotic message/rfc822 nests may miss a part id.
- HTML is **stripped to text** in the Message tab (no HTML engine, no HTML tab). Scripts/iframes never run.
- OAuth needs **your** Google/Microsoft app registration (no bundled client IDs). OAuth accounts are **IMAP + SMTP** only (not POP3).
- IDLE watches Inbox + Sent, not every mailbox (others poll). POP3 has no IDLE (Get Messages / periodic Sync).
- **POP3** is inbox retrieve only: no server folders, no server-side flags (read/starred are local), no MOVE/APPEND on the server, deletes stay local (server copy remains), no TOP preview. Sent goes out SMTP and is filed locally.
- Sieve is not implemented (local Sorting Office rules only).
- Desktop notifications need `notify-send` + a display; otherwise they are a no-op.
- iTip / Calendar is out of scope (Tier C).

## Protocol

Unix domain socket, **JSON-RPC 2.0**, one JSON object per line (NDJSON).

Notifications (no `id`): `mail.changed`, `mail.fetched`, `mail.synced`, `mail.notify`.

| Method | Params |
| --- | --- |
| `ping` | — |
| `status.get` | — |
| `status.set` | `{online}` — Work Offline; going online flushes the outbox |
| `accounts.list` | — |
| `accounts.put` | AccountConfig (`protocol` `imap` or `pop3`; `password` stored in mail.json mode 0600; `passEnv` optional) |
| `accounts.delete` | `{id}` — remove account from mail.json and the local cache (server mail is kept) |
| `oauth.start` | `{provider, address, name?, clientId?, clientSecret?, flow?}` |
| `oauth.poll` | `{sessionId}` |
| `oauth.cancel` | `{sessionId}` |
| `hosts.guess` | `{address}` → IMAP/POP3/SMTP guess |
| `hosts.probe` / `accounts.test` | `{address, user?, password?, protocol?, host?, imap?, pop?, auto?, timeoutMs?}` → Test connection (`ok`, protocol, host, tlsMode) |
| `folders.list` | `{accountId}` |
| `folders.get` | `{id}` |
| `folders.create` | `{accountId, name, parent?}` |
| `folders.virtual` | — Outbox / (hidden unified / categories / smart / VIP) / tags |
| `messages.list` | `{folderId, filter?}` (virtual ids ok) |
| `messages.get` | `{id}` (disk raw / in-memory body if already fetched; no extra IMAP) |
| `messages.search` | `{accountId?, folderId?, filter}` |
| `messages.setFlags` | `{id, patch}` |
| `messages.move` | `{ids, dest}` |
| `messages.delete` | `{ids}` |
| `messages.append` | `{folderId, message}` |
| `messages.update` | `{id, message}` |
| `messages.getPart` | `{id, partId}` |
| `messages.openPart` | `{id, partId}` → `{path}` on disk + `xdg-open` |
| `messages.fetch` | `{accountId}` |
| `sync.run` | `{accountId?}` |
| `unread.get` | `{folderId?}` |
| `compose.send` | `{accountId, identityId?, message, attachPaths?, id?}` |
| `compose.saveDraft` | `{accountId, message, id?}` |
| `outbox.list` / `outbox.flush` | queued send/move/delete/flag |
| `smart.list` / `smart.put` / `smart.delete` | saved search folders |
| `threads.mute` / `threads.muted` | conversation mute |
| `vip.list` / `vip.put` / `vip.delete` | VIP senders |
| `notify.get` / `notify.put` | `{enabled, vipOnly, desktop}` |
| `senders.setCategory` / `senders.categories` | Primary/Other/… override |
| `identities.*` / `tags.*` / `filters.*` | unchanged from v0.9 |

Quick Filter in the UI calls `messages.list` with the pin/query filter so the list is daemon-filtered.

### Filter rules (Sorting Office)

```json
{
  "id": "rule-01",
  "name": "Tag invoices",
  "enabled": true,
  "stop": false,
  "conditions": [{"field": "subject", "op": "contains", "value": "Invoice"}],
  "actions": [{"type": "tag", "tag": "Work"}]
}
```

Condition fields: `from`, `to`, `subject`, `body`, `attachment`, `unread`, `tag`.
Actions: `move` (`folder`), `tag`, `markRead`, `markUnread`, `delete`, `stop`.
AND across conditions. Persist in MemoryStore or the disk cache. Tools → Message Filters.

## UI features (v0.10.13)

- **Empty by default** — no demo accounts unless `UITK_MAIL=memory`. First-run Yes/No is only “There are no accounts, want to add one?” Password / `0600` notes are on the Add Account form.
- **Add account** — IMAP vs POP3 radios, domain auto-guess (including POP hosts), **Test connection** (and optional auto-detect after email+password), masked password field, or Sign in with Google / Microsoft (or device code; IMAP). Saved accounts show the protocol on Account Central and in Preferences. `passEnv` remains an optional fallback.
- **Remove account** — File menu, Account Central, and Preferences → Accounts. Confirm, then drop the account from `mail.json` and the local cache. The folder tree refreshes; if none remain, the first-run “add one?” prompt returns.
- **Text-only Message tab** — prefer `text/plain`; HTML-only mail is tag-stripped. No HTML engine / no HTML tab. The Message/Source body is a read-only `TextView` (scroll + scrollbar; not editable). Compose/Write stays an editable `TextArea`.
- **3-pane splitters** — dragging folder|list or list|preview keeps exclusive pane bounds; preview chrome cannot paint over the thread list.
- **Overflow scrollbars** — thread list, folder tree, and long message bodies show a vertical track/thumb; wheel/trackpad still scroll; offset clamps at the last row. The thread table clips rows under the sticky header (flush at the top; no paint-through while scrolling).
- **Thread columns** — ★, 📎, Topic, Who, When. No Size. View → Sort by Topic / Who / When.
- **Card / Table** — View → Card view or the Cards toolbar toggle. Remembered in `~/.config/uitoolkit/mailui.json`. Star after Message → Star (or context menu) paints immediately.
- **Density** — View → Compact / Default / Relaxed.
- **Folder tree** — account folders and Filters at the top of the sidebar; Outbox is pinned to the **bottom** of the pane (separated from Filters). Unified Folders, Smart Folders, Categories, and VIP are not shown. Click an account root to open that Inbox. The Filters group holds Unread / Starred / Attachment pins (✓ + bold when on) plus remaining tag folders (Important, To Do). Work / Personal / Later are not listed there.
- **Chrome** — no path/subtitle strip, no bottom status bar, no unread-folder-count footer, no sidebar Account / Folders section headers, no identity or Tags ComboBox, and no active-filter banner (`Filter on · N shown` / `Clear filter`) above the thread list. The folder tree (including Filters) starts at the top of the sidebar. The main toolbar keeps Get / Write / Cards / Classic on the left. Tag / Archive / Junk / Delete live on a small toolbar immediately above the Topic / Who / When header; an icon-only Filter button sits in that same row after Delete, right-aligned. The Quick Filter field stays hidden until that button, View → Quick Filter Bar, or Edit → Find / Ctrl+F; Escape hides it again and keeps the query. Filter pins are only on the Filters tree. Reply and Forward are menu / keyboard only (not on the main toolbar). File → Add Account / Remove Account / Account Central still manage stores. Menu and toolbar hover do not refresh the folder tree or the message list.
- **Threaded** view and **Mute Thread**.
- **Attachments** — per-row Open / Save As; toolbar Save All (one folder pick, then write all files); single click selects, double click or row Open opens.
- **Snappy open** — unread click patches the row; preview uses a cached `messages.get` body.
- **Colored tags**, identities, Sorting Office filters — unchanged.

## Keyboard (Thunderbird-like)

Documented in Help → Keyboard and [keyboard.md](keyboard.md). When the thread list (not a text field) has focus:

| Key | Action |
| --- | --- |
| **n** / **p** | Next / previous message |
| **#** | Delete (also Del) |
| **r** | Reply |
| **f** | Forward |
| **c** | Compose |
| **m** | Mark as read |
| **F5** | Get Messages / sync |
| **Ctrl+,** | Preferences |

## How to try (dogfood)

```bash
UITK_MAIL=memory go run ./cmd/mailclientd
UITK_SCENE=auto go run ./cmd/mailclientui
```

- **VIP** — open a Kai / Thunderbird Team message → Message → Add sender to VIP (Preferences → VIP lists senders; no VIP folder in the tree).
- **Threading** — View → Threaded; look for “Thread: lunch plans (3)”. Message → Mute Thread.
- **OAuth** — File → Add Account, enter a Gmail/Outlook address (hosts fill in), paste your client id, Sign in with Google / Microsoft. Approve in the browser (or use Device code). Then Get Messages.

## Screenshots

```bash
go run ./examples/mail -screenshot docs/screenshots
```

Writes `mail-dark.png`, `mail-light.png`, `mail-classic.png`, `mail-compose.png`, `mail-prefs.png`, `mail-cards.png`, `mail-compact.png`, `mail-filters.png`, `mail-empty.png` (first-run dialog), `mail-account.png` (Add Account), `mail-smart.png` (daemon saved-search view; not a tree section).
