package main

import (
	"os"

	"github.com/dyike/skills/internal/contentcreator"
)

func main() {
	if err := contentcreator.NewCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
