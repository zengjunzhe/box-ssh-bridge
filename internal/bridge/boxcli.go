package bridge

import (
	"os/exec"
	"path/filepath"
)

func boxRSHCommand(cfg Config, command string, tty bool) *exec.Cmd {
	args := []string{"rsh", "--box", cfg.Box}
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
