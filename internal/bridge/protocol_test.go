package bridge

import (
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestParsePTY(t *testing.T) {
	payload := ssh.Marshal(struct {
		Term   string
		Cols   uint32
		Rows   uint32
		Width  uint32
		Height uint32
		Modes  string
	}{"xterm-256color", 120, 40, 960, 800, ""})
	term, rows, cols, err := ParsePTY(payload)
	if err != nil {
		t.Fatal(err)
	}
	if term != "xterm-256color" || rows != 40 || cols != 120 {
		t.Fatalf("got term=%q rows=%d cols=%d", term, rows, cols)
	}
}

func TestParseWindowChange(t *testing.T) {
	payload := ssh.Marshal(struct {
		Cols   uint32
		Rows   uint32
		Width  uint32
		Height uint32
	}{100, 30, 800, 600})
	rows, cols, err := ParseWindowChange(payload)
	if err != nil {
		t.Fatal(err)
	}
	if rows != 30 || cols != 100 {
		t.Fatalf("got rows=%d cols=%d", rows, cols)
	}
}

func TestRandomTokenShape(t *testing.T) {
	token := randomToken(32)
	if len(token) != 32 {
		t.Fatalf("length = %d", len(token))
	}
	if token == randomToken(32) {
		t.Fatal("two generated tokens are equal")
	}
}
