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
