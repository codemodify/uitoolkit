package platform

import (
	"bytes"
	"testing"
)

func TestINCRChunksRoundtrip(t *testing.T) {
	data := bytes.Repeat([]byte("xyz"), 100)
	parts := INCRChunks(data, 64)
	if len(parts) < 2 {
		t.Fatalf("parts %d", len(parts))
	}
	var got []byte
	for i, p := range parts {
		var done bool
		got, done = AppendINCRPiece(got, p)
		if done {
			t.Fatalf("done early at %d", i)
		}
	}
	got, done := AppendINCRPiece(got, nil)
	if !done || !bytes.Equal(got, data) {
		t.Fatalf("got %d want %d done=%v", len(got), len(data), done)
	}
}

func TestINCRThresholdEnv(t *testing.T) {
	t.Setenv("UITK_X11_INCR_THRESHOLD", "128")
	if INCRThreshold(1<<20) != 128 {
		t.Fatalf("env %d", INCRThreshold(1<<20))
	}
	t.Setenv("UITK_X11_INCR_THRESHOLD", "")
	if INCRThreshold(1<<20) != defaultINCRThreshold {
		t.Fatalf("default %d", INCRThreshold(1<<20))
	}
}

func TestINCREmpty(t *testing.T) {
	parts := INCRChunks(nil, 64)
	if len(parts) != 1 || parts[0] != nil {
		t.Fatalf("%v", parts)
	}
	_, done := AppendINCRPiece(nil, nil)
	if !done {
		t.Fatal("empty transfer")
	}
}

func TestINCRThresholdRespectsMaxRequest(t *testing.T) {
	// A server with a small max request size must lower the threshold so
	// a reply cannot exceed one request.
	if got := INCRThreshold(16384); got != 4096 {
		t.Fatalf("INCRThreshold(16384) = %d, want 4096", got)
	}
	// A large max request keeps the default.
	if got := INCRThreshold(1 << 20); got != defaultINCRThreshold {
		t.Fatalf("INCRThreshold(1MiB) = %d, want %d", got, defaultINCRThreshold)
	}
}

func TestAppendINCRPieceEmptyTransfer(t *testing.T) {
	// An INCR transfer of an empty selection is a single empty property.
	out, done := AppendINCRPiece(nil, nil)
	if !done || out == nil || len(out) != 0 {
		t.Fatalf("empty transfer = %v,%v", out, done)
	}
	// Data then terminator.
	out, done = AppendINCRPiece(nil, []byte("abc"))
	if done || string(out) != "abc" {
		t.Fatalf("first piece = %q,%v", out, done)
	}
	out, done = AppendINCRPiece(out, []byte("def"))
	if done || string(out) != "abcdef" {
		t.Fatalf("second piece = %q,%v", out, done)
	}
	out, done = AppendINCRPiece(out, nil)
	if !done || string(out) != "abcdef" {
		t.Fatalf("terminator = %q,%v", out, done)
	}
}
