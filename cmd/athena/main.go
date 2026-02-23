package main

import (
	"os"

	"github.com/amr-athena/athena/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(cli.ExitCodeForError(err))
	}
}
