# Mail — mailclientd + mailclientui

Thunderbird-chrome mail client on uitoolkit **v0.9.0**. Two processes:

| Process | Role |
| --- | --- |
| **mailclientd** | Owns accounts, IMAP/SMTP, local cache, folders, messages, tags, filters, identities, search, mutations. |
| **mailclientui** | Renders chrome and sends JSON-RPC commands. **No IMAP or SMTP** in this process. |

Shared types and the RPC client/server live in [`internal/mail`](../internal/mail).

## How to start

Default socket:

- `$UITK_MAIL_SOCK` if set
- else `$XDG_RUNTIME_DIR/mailclientd.sock`
- else `/tmp/mailclientd-<uid>.sock`

```bash
# Terminal 1 — daemon (offline demo MemoryStore)
UITK_MAIL=memory go run ./cmd/mailclientd

# Terminal 2 — Thunderbird UI
UITK_SCENE=auto go run ./cmd/mailclientui
go run ./cmd/mailclientui -classic -light
go run ./cmd/mailclientui -headless    # writes mail.png
```

Convenience (same architecture, one process: daemon goroutine + UI client on a temp socket):

```bash
UITK_SCENE=auto go run ./examples/mail
go run ./examples/mail -screenshot docs/screenshots
```

## Real IMAP + SMTP (primary path)

`mailclientd` is meant to be pointed at a real account. **Never put a password in the config file.** Use `passEnv` (an environment variable name).

Config file (first existing wins):

- `$UITK_MAIL_CONFIG`
- `$XDG_CONFIG_HOME/uitoolkit/mail.json`
- `~/.config/uitoolkit/mail.json`

Cache / offline store:

- `$UITK_MAIL_DATA`
- `$XDG_DATA_HOME/uitoolkit/mail`
- `~/.local/share/uitoolkit/mail`

Example `mail.json` (mode `0600` recommended):

```json
{
  "accounts": [
    {
      "id": "home",
      "name": "Ada Lovelace",
      "address": "ada@example.com",
      "imap": {
        "host": "imap.example.com:993",
        "user": "ada@example.com",
        "passEnv": "UITK_MAIL_PASS",
        "tls": true
      },
      "smtp": {
        "host": "smtp.example.com:587",
        "user": "ada@example.com",
        "passEnv": "UITK_MAIL_PASS",
        "starttls": true
      },
      "identities": [
        {
          "id": "home-default",
          "accountId": "home",
          "name": "Ada Lovelace",
          "address": "ada@example.com",
          "signature": "Ada",
          "default": true
        },
        {
          "id": "home-alias",
          "accountId": "home",
          "name": "Ada L.",
          "address": "ada.lovelace@example.com"
        }
      ]
    }
  ]
}
```

```bash
export UITK_MAIL_PASS='your-app-password'
UITK_MAIL=imap go run ./cmd/mailclientd
go run ./cmd/mailclientui
```

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

Default when **no** config and `UITK_MAIL` is unset: MemoryStore demo (offline dogfood).

`UITK_MAIL=memory` forces the demo even if a config file exists.

### IMAP / SMTP status (honest)

Implemented in mailclientd:

- IMAP: CONNECT, implicit TLS (993) and STARTTLS (143), LOGIN, AUTH PLAIN, AUTH XOAUTH2 stub (`UITK_MAIL_XOAUTH2` bearer), CAPABILITY, LIST/LSUB, SELECT/EXAMINE, UID FETCH (ENVELOPE, FLAGS, BODYSTRUCTURE, BODY.PEEK[] / sections), UID STORE, UID SEARCH, UID COPY, UID MOVE (or COPY+\\Deleted+EXPUNGE), APPEND, EXPUNGE, IDLE (wake on EXISTS/FETCH/EXPUNGE), CONDSTORE CHANGEDSINCE when advertised.
- Incremental cache: UIDVALIDITY wipe, UIDNEXT, highestmodseq when present. Raw `.eml` on disk after body fetch.
- MIME: multipart (nested), text/plain + text/html, attachments, RFC 2047, charset via `golang.org/x/text`.
- SMTP: implicit TLS (465), STARTTLS (587), AUTH PLAIN / LOGIN fallback. Send then IMAP APPEND to Sent.
- Multiple accounts in one config; folder tree mirrors LIST + local specials.
- Offline read of anything already synced.

Known gaps (not production-complete):

- No QRESYNC / vanished vanishing; flag refresh is FLAGS FETCH (+ CHANGEDSINCE when CONDSTORE).
- BODYSTRUCTURE walker covers common multipart/alternative + mixed; exotic message/rfc822 nests may miss a part id.
- HTML is **sanitized and shown as text** (no HTML engine in uitoolkit). Scripts/iframes/on* stripped.
- XOAUTH2 is a token-passthrough stub (no OAuth browser flow).
- IDLE is one mailbox at a time after sync, not a permanent supervisor yet.
- Sieve is not implemented (local Sorting Office rules only).
- Attachment “Open” writes a cache file; the UI does not spawn `xdg-open` (daemon returns the path).

## Protocol

Unix domain socket, **JSON-RPC 2.0**, one JSON object per line (NDJSON).

Notifications (no `id`): `mail.changed`, `mail.fetched`, `mail.synced`.

| Method | Params |
| --- | --- |
| `ping` | — |
| `status.get` | — |
| `accounts.list` | — |
| `folders.list` | `{accountId}` |
| `folders.get` | `{id}` |
| `folders.create` | `{accountId, name, parent?}` |
| `folders.virtual` | — Unified Inbox / Unread / Starred / tag folders |
| `messages.list` | `{folderId, filter?}` (virtual ids ok) |
| `messages.get` | `{id}` (fetches MIME body if needed) |
| `messages.search` | `{accountId?, folderId?, filter}` |
| `messages.setFlags` | `{id, patch}` |
| `messages.move` | `{ids, dest}` |
| `messages.delete` | `{ids}` |
| `messages.append` | `{folderId, message}` |
| `messages.update` | `{id, message}` |
| `messages.getPart` | `{id, partId}` |
| `messages.openPart` | `{id, partId}` → `{path}` on disk |
| `messages.fetch` | `{accountId}` |
| `sync.run` | `{accountId?}` |
| `unread.get` | `{folderId?}` |
| `compose.send` | `{accountId, identityId?, message, attachPaths?, id?}` |
| `compose.saveDraft` | `{accountId, message, id?}` |
| `identities.list` | `{accountId?}` |
| `identities.put` | Identity |
| `identities.delete` | `{id}` |
| `tags.list` | — |
| `tags.put` | `{name, color}` |
| `filters.list` | — |
| `filters.put` | FilterRule |
| `filters.delete` | `{id}` |
| `filters.apply` | `{folderId?}` |

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

## UI features (v0.9)

- **Card / Table** — View → Card view or the Cards toolbar toggle. Remembered in `~/.config/uitoolkit/mailui.json`.
- **Density** — View → Compact / Default / Relaxed (extends v0.8.1 row metrics). Same prefs file.
- **Unified Inbox** — folder pane “Unified Folders” (Inbox / Unread / Starred across accounts).
- **Colored tags** — Message → Tag; Tags section in the folder tree; Quick Filter tag combo.
- **Identities** — compose From picker is not 1:1 with accounts; Preferences → Identities.
- **Filters** — Tools → Message Filters (works on MemoryStore and the disk store).

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

## Screenshots

```bash
go run ./examples/mail -screenshot docs/screenshots
```

Writes `mail-dark.png`, `mail-light.png`, `mail-classic.png`, `mail-compose.png`, `mail-prefs.png`, `mail-cards.png`, `mail-compact.png`, `mail-filters.png`.
