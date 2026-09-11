// Command mailclientd is the Mail daemon: accounts, folders, messages,
// IMAP+SMTP+OAuth with an on-disk cache. No config means an empty store
// (first-run add-account). MemoryStore is explicit UITK_MAIL=memory.
// It listens on a Unix socket and speaks JSON-RPC 2.0 (NDJSON).
// See docs/mail.md.
//
//	go run ./cmd/mailclientd
//	UITK_MAIL=memory go run ./cmd/mailclientd   # seeded demo
//	# real account: File → Add Account, or ~/.config/uitoolkit/mail.json
//	UITK_MAIL_PASS=secret go run ./cmd/mailclientd
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/codemodify/uitoolkit/internal/mail"
)

func main() {
	sock := mail.DefaultSocket()
	store, err := mail.OpenStore()
	if err != nil && store == nil {
		log.Fatal(err)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "mailclientd: %v (serving anyway; Health reports the error)\n", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("mailclientd  backend=%s  socket=%s\n", store.Backend(), sock)
	fmt.Println("JSON-RPC 2.0 NDJSON. Docs: docs/mail.md")
	if err := mail.ListenAndServe(ctx, sock, store); err != nil {
		log.Fatal(err)
	}
}
