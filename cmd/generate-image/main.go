package main

import (
	"os"

	"github.com/dyike/skills/internal/genimage"
)

func main() {
	if err := genimage.NewCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
