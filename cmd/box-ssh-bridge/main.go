package main

import (
	"fmt"
	"os"

	"github.com/zengjunzhe/box-ssh-bridge/internal/bridge"
)

var version = "dev"

func main() {
	if err := bridge.Run(version, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
