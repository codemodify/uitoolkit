# Mail — mailclientd + mailclientui

Thunderbird-chrome dogfood on uitoolkit **v0.8.2**. Two processes:

| Process | Role |
| --- | --- |
| **mailclientd** | Owns accounts, folders, messages, search, mutations. MemoryStore (default) or a skeleton IMAP backend. |
| **mailclientui** | Renders chrome and sends JSON-RPC commands. **No IMAP** in this process. |

Shared types and the RPC client/server live in [`internal/mail`](../internal/mail).

## How to start

Default socket:

- `$UITK_MAIL_SOCK` if set
- else `$XDG_RUNTIME_DIR/mailclientd.sock`
- else `/tmp/mailclientd-<uid>.sock`

```bash
# Terminal 1 — daemon (offline demo MemoryStore)
go run ./cmd/mailclientd

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

## Protocol

Unix domain socket, **JSON-RPC 2.0**, one JSON object per line (NDJSON).

Request:

```json
{"jsonrpc":"2.0","id":1,"method":"messages.list","params":{"folderId":"ada/inbox","filter":{"unread":true}}}
```

Response:

```json
{"jsonrpc":"2.0","id":1,"result":[...]}
```

Notifications (no `id`): `mail.changed`, `mail.fetched`.

| Method | Params |
| --- | --- |
| `ping` | — |
| `status.get` | — |
| `accounts.list` | — |
| `folders.list` | `{accountId}` |
| `folders.get` | `{id}` |
| `folders.create` | `{accountId, name, parent?}` |
| `messages.list` | `{folderId, filter?}` |
| `messages.get` | `{id}` |
| `messages.search` | `{accountId?, folderId?, filter}` |
| `messages.setFlags` | `{id, patch}` |
| `messages.move` | `{ids, dest}` |
| `messages.delete` | `{ids}` |
| `messages.append` | `{folderId, message}` |
| `messages.update` | `{id, message}` |
| `messages.fetch` | `{accountId}` |
| `unread.get` | `{folderId?}` |
| `compose.send` | `{accountId, message, id?}` |
| `compose.saveDraft` | `{accountId, message, id?}` |

Quick Filter in the UI calls `messages.list` with the pin/query filter so the TableView is daemon-filtered.

## IMAP status (demo vs live)

**Default is always the offline MemoryStore.** Two demo accounts, ~150 messages, Get Messages injects arrivals. No sockets except the local RPC.

**Skeleton IMAP** (`UITK_MAIL=imap`) lives only in mailclientd:

```bash
export UITK_MAIL=imap
export UITK_MAIL_HOST=imap.example.com:993
export UITK_MAIL_USER=you@example.com
export UITK_MAIL_PASS=secret
# UITK_MAIL_TLS=1 by default on :993; set 0 for a local fake
go run ./cmd/mailclientd
```

Implemented: CONNECT, LOGIN, LIST, SELECT, FETCH (flags/size/headers/text), STORE, APPEND, CREATE, COPY+EXPUNGE as MOVE. Not production: MIME trees, IDLE, UTF-7 names, SMTP submission, robust literal parsing. Missing env vars return a **clear Health() error**; the UI still starts and shows the error on `status.get`.

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
| **F5** | Get Messages |
| **Ctrl+,** | Preferences (accounts list stub) |

## Screenshots

```bash
go run ./examples/mail -screenshot docs/screenshots
```

Writes `mail-dark.png`, `mail-light.png`, `mail-classic.png`, `mail-compose.png`, `mail-prefs.png`.
