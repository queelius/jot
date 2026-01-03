package main

import (
	"os"

	"github.com/queelius/jot/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
