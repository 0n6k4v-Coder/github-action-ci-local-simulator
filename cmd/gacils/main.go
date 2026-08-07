package main

import (
	"os"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
