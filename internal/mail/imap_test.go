package mail

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestIMAPSkeletonAgainstFakeServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveFakeIMAP(t, ln)

	s := NewIMAPStore(ln.Addr().String(), "ada@example.com", "secret")
	if err := s.Connect(); err != nil {
		t.Fatal(err)
	}
	if err := s.Health(); err != nil {
		t.Fatal(err)
	}
	folders := s.ListFolders("imap")
	if len(folders) < 1 {
		t.Fatal("LIST")
	}
	inbox := folders[0].ID
	msgs := s.ListMessages(inbox)
	if len(msgs) != 1 {
		t.Fatalf("FETCH list %d", len(msgs))
	}
	m, ok := s.GetMessage(msgs[0].ID)
	if !ok || m.Subject == "" {
		t.Fatalf("GET %+v", m)
	}
	if err := s.SetFlags(m.ID, FlagPatch{Read: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
}

func serveFakeIMAP(t *testing.T, ln net.Listener) {
	t.Helper()
	c, err := ln.Accept()
	if err != nil {
		return
	}
	defer c.Close()
	w := bufio.NewWriter(c)
	r := bufio.NewReader(c)
	write := func(s string) {
		_, _ = w.WriteString(s + "\r\n")
		_ = w.Flush()
	}
	write("* OK fake imap")
	for {
		ln, err := r.ReadString('\n')
		if err != nil {
			return
		}
		ln = strings.TrimRight(ln, "\r\n")
		parts := strings.Fields(ln)
		if len(parts) < 2 {
			continue
		}
		tag, cmd := parts[0], strings.ToUpper(parts[1])
		switch cmd {
		case "LOGIN":
			write(tag + " OK logged in")
		case "LIST":
			write(`* LIST (\HasNoChildren \Inbox) "/" "INBOX"`)
			write(tag + " OK list")
		case "SELECT":
			write("* 1 EXISTS")
			write(tag + " OK [READ-WRITE] selected")
		case "FETCH":
			write(`* 1 FETCH (FLAGS (\Seen) RFC822.SIZE 120 BODY[HEADER.FIELDS (FROM TO CC SUBJECT DATE)] {80}`)
			write("From: Bot <bot@example.com>")
			write("To: ada@example.com")
			write("Subject: fake fetch")
			write("Date: " + time.Now().Format(time.RFC1123Z))
			write(")")
			if strings.Contains(strings.ToUpper(ln), "BODY.PEEK[TEXT]") || strings.Contains(strings.ToUpper(ln), "BODY[TEXT]") {
				write(`* 1 FETCH (BODY[TEXT] {12}`)
				write("hello body")
				write(")")
			}
			write(tag + " OK fetch")
		case "STORE":
			write(tag + " OK store")
		case "NOOP":
			write(tag + " OK noop")
		case "LOGOUT":
			write("* BYE")
			write(tag + " OK logout")
			return
		default:
			write(fmt.Sprintf("%s BAD %s", tag, cmd))
		}
	}
}
