package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAuthorizedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authorized_keys")
	line := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGv7mC8fQidR7oR7jdAq4Vp5dS3P4Z+1lQ7vD9AV6mP3 test\n"
	if err := os.WriteFile(path, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	keys, err := LoadAuthorizedKeys(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("keys = %d", len(keys))
	}
}
