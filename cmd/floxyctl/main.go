package main

import (
	"os"

	"github.com/rom8726/floxy-pro/internal/floxyctl"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	rootCmd := floxyctl.NewRootCommand(version, commit)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
