# Mail — moved to its own repository

The mail client that grew up in this repository now lives at
**<https://github.com/codemodify/comms-mail>** and builds against
uitoolkit's published module, with no `replace` and no internal imports.

Everything that used to be documented here — the `comms-maild` daemon and
its Unix-socket JSON-RPC protocol, the store, IMAP / POP3 / SMTP, OAuth,
the socket's permissions and lock file, accounts, filters, identities,
smart folders, the tray item and how to run any of it — is documented
there, next to the code.

This file stays because the changelog in [../README.md](../README.md) links
to it from the releases where that work happened.

## What the toolkit kept

The client was dogfood for four years of widget work, and what it pushed
into the toolkit stayed here:

- `widgets.CardList`, the virtualized multi-line row, and the virtualized
  row reuse in `ListView`, `TableView` and `TreeView`
  ([widgets.md](widgets.md)).
- The tray / status item and desktop notifications, including the
  `desktop-entry` hint and the SNI fallbacks ([tray.md](tray.md)).
- Most of the menu-bar, toolbar and context-menu chrome behaviour measured
  against Qt, GTK and Avalonia in [compare.md](compare.md).
- The keyboard contract: a key event that carries both `Key` and `Rune`,
  and window-level shortcut dispatch ([keyboard.md](keyboard.md)).

Two screenshots of it are kept in [screenshots/](screenshots) as evidence
of what the toolkit draws; they are regenerated in the other repository.
