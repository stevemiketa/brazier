package main

import (
	"os"

	"github.com/brazier/brazier/cmd/cli/cmd"
)

func main() {
	if err := cmd.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
