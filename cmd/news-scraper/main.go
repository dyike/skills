package main

import (
	"os"

	"github.com/dyike/skills/internal/scraper"
)

func main() {
	if err := scraper.NewCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
