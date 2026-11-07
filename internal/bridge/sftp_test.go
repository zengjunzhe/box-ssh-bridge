package bridge

import (
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
	cmd := boxRSHCommand(cfg, "bash -li", true)
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
