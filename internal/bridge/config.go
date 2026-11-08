package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	BoxUsers      map[string]BoxRoute
	UsersFile     string
	Password      string
	AuthorizedKey string
	HostKey       string
	BoxCLI        string
	BoxPackage    string
	SFTPCommand   string
	StateDir      string
}

type BoxRoute struct {
	Box     string
	BoxHome string
}

func (c Config) boxForUser(user string) (BoxRoute, bool) {
	if c.BoxUsers != nil {
		route, ok := c.BoxUsers[user]
		return route, ok
	}
	return BoxRoute{Box: c.Box}, user == c.User
}

func (c Config) String() string {
	if len(c.BoxUsers) == 0 {
		return fmt.Sprintf("%s -> %s (user %s)", c.Listen, c.Box, c.User)
	}
	users := make([]string, 0, len(c.BoxUsers))
	for user := range c.BoxUsers {
		users = append(users, user)
	}
	sort.Strings(users)
	routes := make([]string, 0, len(users))
	for _, user := range users {
		routes = append(routes, user+"="+c.BoxUsers[user].Box)
	}
	return fmt.Sprintf("%s -> %s", c.Listen, strings.Join(routes, ", "))
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
