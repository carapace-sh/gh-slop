package main

import (
	"os"

	"github.com/rsteube/gh-slop/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
