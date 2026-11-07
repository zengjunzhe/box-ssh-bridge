package bridge

import (
	"os"
	"path/filepath"
)

const (
	defaultListen = "127.0.0.1:22220"
	defaultBox    = "main"
	defaultUser   = "user"
)

type Config struct {
	Listen        string
	Box           string
	User          string
	Password      string
	AuthorizedKey string
	HostKey       string
	BoxCLI        string
	BoxPackage    string
	SFTPCommand   string
	StateDir      string
}

func DefaultStateDir() string {
	if dir := os.Getenv("BOX_SSH_BRIDGE_HOME"); dir != "" {
		return dir
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", ".box-ssh-bridge")
	}
	return filepath.Join(dir, "box-ssh-bridge")
}

func defaultHostKey(stateDir string) string {
	return filepath.Join(stateDir, "ssh_host_ed25519")
}

func defaultAuthorizedKeys(stateDir string) string {
	return filepath.Join(stateDir, "authorized_keys")
}
