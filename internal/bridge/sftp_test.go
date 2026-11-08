package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSFTPCommandFlag(t *testing.T) {
	cfg, _, err := parseFlags([]string{"--sftp-command", "/tmp/sftp-wrapper"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SFTPCommand != "/tmp/sftp-wrapper" {
		t.Fatalf("SFTPCommand = %q", cfg.SFTPCommand)
	}
}

func TestNpxCommandIncludesPackage(t *testing.T) {
	cfg := Config{Box: "main", BoxCLI: "/usr/local/bin/npx", BoxPackage: "@simular-ai/box"}
	cmd := boxRSHCommand(cfg, "main", "bash -li", true)
	args := cmd.Args
	want := []string{"--yes", "@simular-ai/box", "rsh", "--box", "main", "-t", "--", "bash -li"}
	args = args[1:]
	if len(args) != len(want) {
		t.Fatalf("args = %#v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestLoadUsersAndRoute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	if err := os.WriteFile(path, []byte(`{"work":"box-work","personal":"box-personal"}`), 0600); err != nil {
		t.Fatal(err)
	}
	users, err := LoadUsers(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{BoxUsers: users}
	if route, ok := cfg.boxForUser("work"); !ok || route.Box != "box-work" {
		t.Fatalf("work route = %#v, %v", route, ok)
	}
	if _, ok := cfg.boxForUser("unknown"); ok {
		t.Fatal("unknown user was accepted")
	}
}

func TestSingleBoxRoute(t *testing.T) {
	cfg := Config{Box: "main", User: "user"}
	if route, ok := cfg.boxForUser("user"); !ok || route.Box != "main" {
		t.Fatalf("route = %#v, %v", route, ok)
	}
	if _, ok := cfg.boxForUser("other"); ok {
		t.Fatal("other user was accepted")
	}
}
