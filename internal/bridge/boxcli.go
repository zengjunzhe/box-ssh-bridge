package bridge

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func boxRSHCommand(cfg Config, box, command string, tty bool) *exec.Cmd {
	args := []string{"rsh", "--box", box}
	if tty {
		args = append(args, "-t")
	}
	args = append(args, "--", command)

	name := cfg.BoxCLI
	if filepath.Base(name) == "npx" || filepath.Base(name) == "npx.cmd" {
		args = append([]string{"--yes", cfg.BoxPackage}, args...)
		return exec.Command(name, args...)
	}
	if filepath.Ext(name) == ".js" {
		return exec.Command("node", append([]string{name}, args...)...)
	}
	return exec.Command(name, args...)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
