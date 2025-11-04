package main

import (
	"os"

	"github.com/rom8726/floxy/internal/floxyctl"
)

func main() {
	rootCmd := floxyctl.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
